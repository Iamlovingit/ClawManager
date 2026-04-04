package models

import "time"

type InstanceRuntimeStatus struct {
	ID                      int        `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID              int        `db:"instance_id" json:"instance_id"`
	InfraStatus             string     `db:"infra_status" json:"infra_status"`
	AgentStatus             string     `db:"agent_status" json:"agent_status"`
	OpenClawStatus          string     `db:"openclaw_status" json:"openclaw_status"`
	OpenClawPID             *int       `db:"openclaw_pid" json:"openclaw_pid,omitempty"`
	OpenClawVersion         *string    `db:"openclaw_version" json:"openclaw_version,omitempty"`
	CurrentConfigRevisionID *int       `db:"current_config_revision_id" json:"current_config_revision_id,omitempty"`
	DesiredConfigRevisionID *int       `db:"desired_config_revision_id" json:"desired_config_revision_id,omitempty"`
	SummaryJSON             *string    `db:"summary_json" json:"-"`
	SystemInfoJSON          *string    `db:"system_info_json" json:"-"`
	HealthJSON              *string    `db:"health_json" json:"-"`
	LastReportedAt          *time.Time `db:"last_reported_at" json:"last_reported_at,omitempty"`
	CreatedAt               time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time  `db:"updated_at" json:"updated_at"`
}

func (s InstanceRuntimeStatus) TableName() string {
	return "instance_runtime_status"
}
