package services

import (
	"context"
	"fmt"
	"time"

	"clawreef/internal/models"
	"clawreef/internal/repository"
	"clawreef/internal/services/k8s"

	corev1 "k8s.io/api/core/v1"
)

// SyncService handles synchronization between database and K8s state
type SyncService struct {
	instanceRepo         repository.InstanceRepository
	runtimeStatusService InstanceRuntimeStatusService
	podService           *k8s.PodService
	interval             time.Duration
	stopChan             chan struct{}
}

// NewSyncService creates a new sync service
func NewSyncService(instanceRepo repository.InstanceRepository, runtimeStatusService InstanceRuntimeStatusService) *SyncService {
	return &SyncService{
		instanceRepo:         instanceRepo,
		runtimeStatusService: runtimeStatusService,
		podService:           k8s.NewPodService(),
		interval:             5 * time.Second, // Sync every 5 seconds for more responsive status updates
		stopChan:             make(chan struct{}),
	}
}

// Start starts the sync service
func (s *SyncService) Start() {
	fmt.Println("Starting K8s state sync service...")
	go s.syncLoop()
}

// Stop stops the sync service
func (s *SyncService) Stop() {
	close(s.stopChan)
}

// syncLoop runs the synchronization loop
func (s *SyncService) syncLoop() {
	fmt.Printf("[SyncService] Starting sync loop with interval %v\n", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	fmt.Println("[SyncService] Running initial sync...")
	s.syncAllInstances()
	fmt.Println("[SyncService] Initial sync complete")

	for {
		select {
		case <-ticker.C:
			fmt.Println("[SyncService] Tick - running scheduled sync...")
			s.syncAllInstances()
		case <-s.stopChan:
			fmt.Println("[SyncService] Stopping K8s state sync service...")
			return
		}
	}
}

// syncAllInstances synchronizes the state of all instances
func (s *SyncService) syncAllInstances() {
	ctx := context.Background()

	fmt.Println("[SyncService] Fetching all instances from database...")
	// Get all running instances from database
	instances, err := s.instanceRepo.GetAllRunning()
	if err != nil {
		fmt.Printf("[SyncService] Error getting running instances: %v\n", err)
		return
	}

	fmt.Printf("[SyncService] Found %d instances to sync\n", len(instances))

	if len(instances) == 0 {
		fmt.Println("[SyncService] No instances found, skipping sync")
		return
	}

	for i, instance := range instances {
		fmt.Printf("[SyncService] Syncing instance %d/%d: ID=%d, Status=%s\n",
			i+1, len(instances), instance.ID, instance.Status)
		s.syncInstance(ctx, &instance)
	}

	fmt.Println("[SyncService] Sync complete")
}

// syncInstance synchronizes a single instance's state
func (s *SyncService) syncInstance(ctx context.Context, instance *models.Instance) {
	// Check if pod exists in K8s
	pod, err := s.podService.GetPod(ctx, instance.UserID, instance.ID)
	if err != nil {
		// Pod doesn't exist in K8s
		if instance.Status == "running" || instance.Status == "creating" {
			nextStatus := "stopped"
			if instance.Status == "creating" {
				nextStatus = "error"
			}

			fmt.Printf("Instance %d marked as %s but pod not found in K8s, updating status to %s\n",
				instance.ID, instance.Status, nextStatus)
			instance.Status = nextStatus
			instance.PodName = nil
			instance.PodNamespace = nil
			instance.PodIP = nil
			instance.UpdatedAt = time.Now()

			if err := s.instanceRepo.Update(instance); err != nil {
				fmt.Printf("Error updating instance %d status: %v\n", instance.ID, err)
			} else {
				s.updateInfraStatus(instance.ID, nextStatus)
				// Broadcast status update
				GetHub().BroadcastInstanceStatus(instance.UserID, instance)
			}
		}
		return
	}

	// Pod exists, update instance info
	needsUpdate := false

	// Check pod status and update instance accordingly
	desiredStatus := mapPodToInstanceStatus(pod)
	if instance.Status != desiredStatus {
		fmt.Printf("Instance %d: Pod status %s/ready=%v but instance status is %s, updating to %s\n",
			instance.ID, pod.Status.Phase, isPodReady(pod), instance.Status, desiredStatus)
		instance.Status = desiredStatus
		needsUpdate = true
	}
	s.updateInfraStatus(instance.ID, desiredStatus)

	// Update Pod IP if changed
	if pod.Status.PodIP != "" {
		if instance.PodIP == nil || *instance.PodIP != pod.Status.PodIP {
			instance.PodIP = &pod.Status.PodIP
			needsUpdate = true
		}
	}

	// Update Pod name if changed
	if instance.PodName == nil || *instance.PodName != pod.Name {
		instance.PodName = &pod.Name
		needsUpdate = true
	}

	// Update namespace if changed
	if instance.PodNamespace == nil || *instance.PodNamespace != pod.Namespace {
		instance.PodNamespace = &pod.Namespace
		needsUpdate = true
	}

	if needsUpdate {
		instance.UpdatedAt = time.Now()
		if err := s.instanceRepo.Update(instance); err != nil {
			fmt.Printf("Error updating instance %d: %v\n", instance.ID, err)
		} else {
			fmt.Printf("Instance %d status synced: %s (Pod: %s, IP: %s)\n",
				instance.ID, instance.Status, pod.Name, pod.Status.PodIP)
			// Broadcast status update
			GetHub().BroadcastInstanceStatus(instance.UserID, instance)
		}
	}
}

func (s *SyncService) updateInfraStatus(instanceID int, instanceStatus string) {
	if s.runtimeStatusService == nil {
		return
	}
	infraStatus := mapInstanceStatusToInfraStatus(instanceStatus)
	if err := s.runtimeStatusService.UpsertInfraStatus(instanceID, infraStatus); err != nil {
		fmt.Printf("Error updating runtime infra status for instance %d: %v\n", instanceID, err)
	}
}

func mapInstanceStatusToInfraStatus(instanceStatus string) string {
	switch instanceStatus {
	case "running":
		return "ready"
	case "stopped":
		return "stopped"
	case "error":
		return "error"
	case "creating":
		return "creating"
	default:
		return "creating"
	}
}

func mapPodToInstanceStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "error"
	}

	switch pod.Status.Phase {
	case corev1.PodRunning:
		if isPodReady(pod) {
			return "running"
		}
		return "creating"
	case corev1.PodPending:
		return "creating"
	case corev1.PodSucceeded:
		return "stopped"
	case corev1.PodFailed, corev1.PodUnknown:
		return "error"
	default:
		return "creating"
	}
}

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// ForceSync forces an immediate sync of all instances
func (s *SyncService) ForceSync() {
	s.syncAllInstances()
}
