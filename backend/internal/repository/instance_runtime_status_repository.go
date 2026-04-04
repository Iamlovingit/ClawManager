package repository

import (
	"fmt"
	"time"

	"clawreef/internal/models"

	"github.com/upper/db/v4"
)

type InstanceRuntimeStatusRepository interface {
	GetByInstanceID(instanceID int) (*models.InstanceRuntimeStatus, error)
	Create(status *models.InstanceRuntimeStatus) error
	Update(status *models.InstanceRuntimeStatus) error
}

type instanceRuntimeStatusRepository struct {
	sess db.Session
}

func NewInstanceRuntimeStatusRepository(sess db.Session) InstanceRuntimeStatusRepository {
	return &instanceRuntimeStatusRepository{sess: sess}
}

func (r *instanceRuntimeStatusRepository) GetByInstanceID(instanceID int) (*models.InstanceRuntimeStatus, error) {
	var item models.InstanceRuntimeStatus
	if err := r.sess.Collection("instance_runtime_status").Find(db.Cond{"instance_id": instanceID}).One(&item); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance runtime status: %w", err)
	}
	return &item, nil
}

func (r *instanceRuntimeStatusRepository) Create(status *models.InstanceRuntimeStatus) error {
	ensureTimestamps(&status.CreatedAt, &status.UpdatedAt)
	res, err := r.sess.Collection("instance_runtime_status").Insert(status)
	if err != nil {
		return fmt.Errorf("failed to create instance runtime status: %w", err)
	}
	if id, ok := res.ID().(int64); ok {
		status.ID = int(id)
	}
	return nil
}

func (r *instanceRuntimeStatusRepository) Update(status *models.InstanceRuntimeStatus) error {
	if status.UpdatedAt.IsZero() {
		status.UpdatedAt = time.Now().UTC()
	}
	if err := r.sess.Collection("instance_runtime_status").Find(db.Cond{"id": status.ID}).Update(status); err != nil {
		return fmt.Errorf("failed to update instance runtime status: %w", err)
	}
	return nil
}
