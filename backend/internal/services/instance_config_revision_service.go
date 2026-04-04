package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"clawreef/internal/models"
	"clawreef/internal/repository"
)

type InstanceConfigRevisionPayload struct {
	ID               int             `json:"id"`
	InstanceID       int             `json:"instance_id"`
	SourceSnapshotID *int            `json:"source_snapshot_id,omitempty"`
	SourceBundleID   *int            `json:"source_bundle_id,omitempty"`
	RevisionNo       int             `json:"revision_no"`
	Content          json.RawMessage `json:"content"`
	Checksum         string          `json:"checksum"`
	Status           string          `json:"status"`
	PublishedBy      *int            `json:"published_by,omitempty"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	ActivatedAt      *time.Time      `json:"activated_at,omitempty"`
}

type InstanceConfigRevisionService interface {
	GetByID(id int) (*InstanceConfigRevisionPayload, error)
	ListByInstanceID(instanceID int, limit int) ([]InstanceConfigRevisionPayload, error)
	CreateFromSnapshot(instanceID int, snapshot *models.OpenClawInjectionSnapshot, publishedBy *int) (*InstanceConfigRevisionPayload, error)
}

type instanceConfigRevisionService struct {
	repo repository.InstanceConfigRevisionRepository
}

func NewInstanceConfigRevisionService(repo repository.InstanceConfigRevisionRepository) InstanceConfigRevisionService {
	return &instanceConfigRevisionService{repo: repo}
}

func (s *instanceConfigRevisionService) GetByID(id int) (*InstanceConfigRevisionPayload, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("instance config revision not found")
	}
	return configRevisionPayloadFromModel(*item)
}

func (s *instanceConfigRevisionService) ListByInstanceID(instanceID int, limit int) ([]InstanceConfigRevisionPayload, error) {
	items, err := s.repo.ListByInstanceID(instanceID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]InstanceConfigRevisionPayload, 0, len(items))
	for _, item := range items {
		payload, err := configRevisionPayloadFromModel(item)
		if err != nil {
			return nil, err
		}
		result = append(result, *payload)
	}
	return result, nil
}

func (s *instanceConfigRevisionService) CreateFromSnapshot(instanceID int, snapshot *models.OpenClawInjectionSnapshot, publishedBy *int) (*InstanceConfigRevisionPayload, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("openclaw injection snapshot is required")
	}

	latest, err := s.repo.GetLatestByInstanceID(instanceID)
	if err != nil {
		return nil, err
	}
	revisionNo := 1
	if latest != nil {
		revisionNo = latest.RevisionNo + 1
	}
	contentJSON := strings.TrimSpace(snapshot.RenderedManifestJSON)
	if contentJSON == "" {
		contentJSON = "{}"
	}
	sum := sha256.Sum256([]byte(contentJSON))
	checksum := "sha256:" + hex.EncodeToString(sum[:])
	now := time.Now().UTC()
	revision := &models.InstanceConfigRevision{
		InstanceID:       instanceID,
		SourceSnapshotID: &snapshot.ID,
		RevisionNo:       revisionNo,
		ContentJSON:      contentJSON,
		Checksum:         checksum,
		Status:           "published",
		PublishedBy:      publishedBy,
		PublishedAt:      &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Create(revision); err != nil {
		return nil, err
	}
	return configRevisionPayloadFromModel(*revision)
}

func configRevisionPayloadFromModel(item models.InstanceConfigRevision) (*InstanceConfigRevisionPayload, error) {
	payload := &InstanceConfigRevisionPayload{
		ID:               item.ID,
		InstanceID:       item.InstanceID,
		SourceSnapshotID: item.SourceSnapshotID,
		SourceBundleID:   item.SourceBundleID,
		RevisionNo:       item.RevisionNo,
		Checksum:         item.Checksum,
		Status:           item.Status,
		PublishedBy:      item.PublishedBy,
		PublishedAt:      item.PublishedAt,
		ActivatedAt:      item.ActivatedAt,
		Content:          json.RawMessage(item.ContentJSON),
	}
	return payload, nil
}
