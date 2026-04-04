package repository

import (
	"fmt"
	"time"

	"clawreef/internal/models"

	"github.com/upper/db/v4"
)

type InstanceAgentRepository interface {
	GetByInstanceID(instanceID int) (*models.InstanceAgent, error)
	GetBySessionToken(sessionToken string) (*models.InstanceAgent, error)
	Create(agent *models.InstanceAgent) error
	Update(agent *models.InstanceAgent) error
}

type instanceAgentRepository struct {
	sess db.Session
}

func NewInstanceAgentRepository(sess db.Session) InstanceAgentRepository {
	return &instanceAgentRepository{sess: sess}
}

func (r *instanceAgentRepository) GetByInstanceID(instanceID int) (*models.InstanceAgent, error) {
	var item models.InstanceAgent
	if err := r.sess.Collection("instance_agents").Find(db.Cond{"instance_id": instanceID}).One(&item); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance agent: %w", err)
	}
	return &item, nil
}

func (r *instanceAgentRepository) GetBySessionToken(sessionToken string) (*models.InstanceAgent, error) {
	var item models.InstanceAgent
	if err := r.sess.Collection("instance_agents").Find(db.Cond{"session_token": sessionToken}).One(&item); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance agent by session token: %w", err)
	}
	return &item, nil
}

func (r *instanceAgentRepository) Create(agent *models.InstanceAgent) error {
	ensureTimestamps(&agent.CreatedAt, &agent.UpdatedAt)
	res, err := r.sess.Collection("instance_agents").Insert(agent)
	if err != nil {
		return fmt.Errorf("failed to create instance agent: %w", err)
	}
	if id, ok := res.ID().(int64); ok {
		agent.ID = int(id)
	}
	return nil
}

func (r *instanceAgentRepository) Update(agent *models.InstanceAgent) error {
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = time.Now().UTC()
	}
	if err := r.sess.Collection("instance_agents").Find(db.Cond{"id": agent.ID}).Update(agent); err != nil {
		return fmt.Errorf("failed to update instance agent: %w", err)
	}
	return nil
}
