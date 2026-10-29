package services

import (
	"clawreef/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

// A retained old pool and a newly committed pool can coexist. An excluded
// instance must not silently restart on the new core without data migration.
func (s *RuntimeUpgradeService) filterUpgradePlacement(ctx context.Context, id int, pods []models.RuntimePod) ([]models.RuntimePod, error) {
	return s.filterUpgradePlacementWithInventory(ctx, id, pods, nil)
}

// liveInventory may be supplied only by the current planning call. It is never
// cached across mutations or requests.
func (s *RuntimeUpgradeService) filterUpgradePlacementWithInventory(ctx context.Context, id int, pods, liveInventory []models.RuntimePod) ([]models.RuntimePod, error) {
	var item models.RuntimeUpgradeItem
	var latestStatus string
	row, err := s.sess.SQL().QueryRowContext(ctx, `SELECT ui.id,r.status FROM runtime_upgrade_items ui JOIN runtime_rollouts r ON r.id=ui.rollout_id WHERE ui.instance_id=? AND r.preflight_id IS NOT NULL AND r.status IN('pending','running','finished','error') ORDER BY r.id DESC LIMIT 1`, id)
	if err != nil {
		return nil, err
	}
	var itemID int64
	if err = row.Scan(&itemID, &latestStatus); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		if err = s.sess.Collection("runtime_upgrade_items").Find("id", itemID).One(&item); err != nil {
			return nil, err
		}
		if item.IsolatedAt != nil {
			result := []models.RuntimePod{}
			for _, p := range pods {
				if item.RuntimePodID != nil && p.ID == *item.RuntimePodID && persistedSourcePodMatches(p, item) && runtimePodMatchesImage(p, "retained@"+stringValue(item.SourceImageDigest)) {
					result = append(result, p)
				}
			}
			return result, nil
		}
	}
	var sourceJSON string
	var target string
	var targetDigest string
	var excluded bool
	var rolloutID int64
	row, err = s.sess.SQL().QueryRowContext(ctx, `SELECT r.id,COALESCE(r.source_images_json,'{}'),r.target_image_ref,COALESCE(r.target_image_digest,''),ui.id IS NULL FROM runtime_rollouts r LEFT JOIN runtime_upgrade_items ui ON ui.rollout_id=r.id AND ui.instance_id=? WHERE r.runtime_type='openclaw' AND r.status='finished' AND r.preflight_id IS NOT NULL AND (ui.id IS NOT NULL OR JSON_CONTAINS(JSON_EXTRACT(r.preflight_json,'$.exclude_instance_ids'),JSON_ARRAY(?))) ORDER BY r.id DESC LIMIT 1`, id, id)
	if err != nil {
		return nil, err
	}
	err = row.Scan(&rolloutID, &sourceJSON, &target, &targetDigest, &excluded)
	if err == sql.ErrNoRows {
		// New instances have no format/history affinity. Prefer the newest committed
		// target rather than assigning them to a retained legacy pool.
		row, err = s.sess.SQL().QueryRowContext(ctx, `SELECT id,COALESCE(source_images_json,'{}'),target_image_ref,COALESCE(target_image_digest,'') FROM runtime_rollouts WHERE runtime_type='openclaw' AND status='finished' AND preflight_id IS NOT NULL ORDER BY id DESC LIMIT 1`)
		if err != nil {
			return nil, err
		}
		err = row.Scan(&rolloutID, &sourceJSON, &target, &targetDigest)
		if err == sql.ErrNoRows {
			return pods, nil
		}
		excluded = false
	}
	if err != nil {
		return nil, err
	}
	if targetDigest == "" {
		targetDigest = imageDigestFromReference(target)
	}
	targetApproved := func(pod models.RuntimePod) bool {
		return targetDigest != "" && runtimePodMatchesImage(pod, target) && stringValue(pod.ImageDigest) == targetDigest
	}
	var sources map[string]string
	if err = json.Unmarshal([]byte(sourceJSON), &sources); err != nil {
		return nil, err
	}
	result := make([]models.RuntimePod, 0, len(pods))
	approvedSources := sources
	if excluded {
		approvedSources = map[string]string{}
		if s.bindings != nil && s.pods != nil {
			binding, e := s.bindings.GetByInstanceID(ctx, id)
			if e != nil {
				return nil, e
			}
			if binding != nil {
				pod, e := s.pods.GetByID(ctx, binding.RuntimePodID)
				if e != nil {
					return nil, e
				}
				if pod != nil {
					key := pod.Namespace + "/" + pod.DeploymentName
					if image, ok := sources[key]; ok {
						approvedSources[key] = image
					}
				}
			}
		}
		if len(approvedSources) == 0 {
			unique := ""
			unambiguous := true
			for _, image := range sources {
				if unique == "" {
					unique = image
				}
				if unique != image {
					unambiguous = false
				}
			}
			if unambiguous && imageDigestFromReference(unique) != "" {
				approvedSources = sources
			}
		}
	}
	isTargetPool := func(pod models.RuntimePod) bool {
		isTarget := false
		for source := range sources {
			parts := strings.SplitN(source, "/", 2)
			if len(parts) == 2 && pod.Namespace+"/"+pod.DeploymentName == parts[0]+"/"+runtimeUpgradeTargetDeploymentName(parts[1], rolloutID) {
				isTarget = true
			}
		}
		return isTarget
	}
	// An exclusion approves only its captured source binary. Merely being outside
	// the new Deployment must never admit a changed core.
	sourceApproved := func(pod models.RuntimePod) bool {
		for source, image := range approvedSources {
			parts := strings.SplitN(source, "/", 2)
			if len(parts) == 2 && parts[0] == pod.Namespace && runtimePodMatchesImage(pod, image) {
				return true
			}
		}
		return false
	}
	oldAllowed := func(pod models.RuntimePod) bool {
		if excluded {
			_, canonical := approvedSources[pod.Namespace+"/"+pod.DeploymentName]
			return canonical && !isTargetPool(pod) && sourceApproved(pod)
		}
		return isTargetPool(pod) && runtimePodMatchesImage(pod, target)
	}
	// Unfinished or unresolved recovery never gains a broader placement policy.
	mayUseCopies := itemID == 0 || (latestStatus == "finished" && (item.State == "verified" || item.State == "gateway_verified"))
	if !excluded && targetDigest == "" {
		mayUseCopies = false
	}
	if mayUseCopies && liveInventory == nil && s.deployments != nil {
		liveInventory, err = s.deployments.RuntimeDeploymentPods(ctx, RuntimeTypeOpenClaw)
		if err != nil {
			return nil, err
		}
	}
	witnesses := []models.RuntimePod{}
	confirmedIdentity := func(pod models.RuntimePod) bool {
		uid := stringValue(pod.PodUID)
		return pod.ID > 0 && uid != "" && stringValue(pod.RegisteredPodUID) == uid && stringValue(pod.KubernetesPodUID) == uid
	}
	for _, pod := range liveInventory {
		canonical := isTargetPool(pod)
		if excluded {
			_, canonical = approvedSources[pod.Namespace+"/"+pod.DeploymentName]
		}
		if canonical && confirmedIdentity(pod) && oldAllowed(pod) && (excluded || targetApproved(pod)) && !runtimePodIsUpgradeLab(pod) {
			witnesses = append(witnesses, pod)
		}
	}
	// A later completed verified recovery can have removed the original target
	// pool. Its currently bound approved binary is also an authenticated witness.
	if mayUseCopies && !excluded && itemID > 0 && item.RolloutID == rolloutID && s.bindings != nil {
		binding, e := s.bindings.GetByInstanceID(ctx, id)
		if e != nil {
			return nil, e
		}
		if binding != nil && binding.State == "running" {
			for _, pod := range liveInventory {
				if pod.ID == binding.RuntimePodID && confirmedIdentity(pod) && targetApproved(pod) && !runtimePodIsUpgradeLab(pod) {
					witnesses = append(witnesses, pod)
				}
			}
		}
	}
	for _, pod := range pods {
		allowed := oldAllowed(pod)
		if !allowed && mayUseCopies && !runtimePodIsUpgradeLab(pod) {
			for _, live := range liveInventory {
				if live.ID != pod.ID || !confirmedIdentity(live) || stringValue(live.PodUID) != stringValue(pod.PodUID) {
					continue
				}
				if excluded && !sourceApproved(live) {
					continue
				}
				if excluded {
					immutable := false
					for _, image := range approvedSources {
						if imageDigestFromReference(image) != "" && runtimePodMatchesImage(live, image) {
							immutable = true
						}
					}
					if !immutable {
						continue
					}
				}
				if !excluded && !targetApproved(live) {
					continue
				}
				for _, witness := range witnesses {
					if runtimeManagementCompatibleImage(witness, live) == nil {
						allowed = true
						break
					}
				}
				break
			}
		}
		if allowed {
			result = append(result, pod)
		}
	}
	return result, nil
}
