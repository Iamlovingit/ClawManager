package models

import "time"

type InstanceDesiredState struct {
	ID                      int       `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID              int       `db:"instance_id" json:"instance_id"`
	DesiredPowerState       string    `db:"desired_power_state" json:"desired_power_state"`
	DesiredConfigRevisionID *int      `db:"desired_config_revision_id" json:"desired_config_revision_id,omitempty"`
	DesiredRuntimeAction    *string   `db:"desired_runtime_action" json:"desired_runtime_action,omitempty"`
	UpdatedBy               *int      `db:"updated_by" json:"updated_by,omitempty"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
}

func (s InstanceDesiredState) TableName() string {
	return "instance_desired_state"
}
