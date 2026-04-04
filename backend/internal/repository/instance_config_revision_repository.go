package repository

import (
	"fmt"
	"time"

	"clawreef/internal/models"

	"github.com/upper/db/v4"
)

type InstanceConfigRevisionRepository interface {
	Create(revision *models.InstanceConfigRevision) error
	Update(revision *models.InstanceConfigRevision) error
	GetByID(id int) (*models.InstanceConfigRevision, error)
	GetLatestByInstanceID(instanceID int) (*models.InstanceConfigRevision, error)
	ListByInstanceID(instanceID int, limit int) ([]models.InstanceConfigRevision, error)
}

type instanceConfigRevisionRepository struct {
	sess db.Session
}

func NewInstanceConfigRevisionRepository(sess db.Session) InstanceConfigRevisionRepository {
	return &instanceConfigRevisionRepository{sess: sess}
}

func (r *instanceConfigRevisionRepository) Create(revision *models.InstanceConfigRevision) error {
	ensureTimestamps(&revision.CreatedAt, &revision.UpdatedAt)
	res, err := r.sess.Collection("instance_config_revisions").Insert(revision)
	if err != nil {
		return fmt.Errorf("failed to create instance config revision: %w", err)
	}
	if id, ok := res.ID().(int64); ok {
		revision.ID = int(id)
	}
	return nil
}

func (r *instanceConfigRevisionRepository) Update(revision *models.InstanceConfigRevision) error {
	if revision.UpdatedAt.IsZero() {
		revision.UpdatedAt = time.Now().UTC()
	}
	if err := r.sess.Collection("instance_config_revisions").Find(db.Cond{"id": revision.ID}).Update(revision); err != nil {
		return fmt.Errorf("failed to update instance config revision: %w", err)
	}
	return nil
}

func (r *instanceConfigRevisionRepository) GetByID(id int) (*models.InstanceConfigRevision, error) {
	var item models.InstanceConfigRevision
	if err := r.sess.Collection("instance_config_revisions").Find(db.Cond{"id": id}).One(&item); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance config revision: %w", err)
	}
	return &item, nil
}

func (r *instanceConfigRevisionRepository) GetLatestByInstanceID(instanceID int) (*models.InstanceConfigRevision, error) {
	var item models.InstanceConfigRevision
	if err := r.sess.Collection("instance_config_revisions").Find(db.Cond{"instance_id": instanceID}).OrderBy("-revision_no", "-id").One(&item); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest instance config revision: %w", err)
	}
	return &item, nil
}

func (r *instanceConfigRevisionRepository) ListByInstanceID(instanceID int, limit int) ([]models.InstanceConfigRevision, error) {
	if limit <= 0 {
		limit = 20
	}

	var items []models.InstanceConfigRevision
	if err := r.sess.Collection("instance_config_revisions").Find(db.Cond{"instance_id": instanceID}).OrderBy("-revision_no", "-id").Limit(limit).All(&items); err != nil {
		return nil, fmt.Errorf("failed to list instance config revisions: %w", err)
	}
	return items, nil
}
