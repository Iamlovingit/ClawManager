package models

import "time"

type InstanceAgent struct {
	ID               int        `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID       int        `db:"instance_id" json:"instance_id"`
	AgentID          string     `db:"agent_id" json:"agent_id"`
	AgentVersion     string     `db:"agent_version" json:"agent_version"`
	ProtocolVersion  string     `db:"protocol_version" json:"protocol_version"`
	Status           string     `db:"status" json:"status"`
	CapabilitiesJSON string     `db:"capabilities_json" json:"-"`
	HostInfoJSON     *string    `db:"host_info_json" json:"-"`
	SessionToken     *string    `db:"session_token" json:"-"`
	SessionExpiresAt *time.Time `db:"session_expires_at" json:"session_expires_at,omitempty"`
	LastHeartbeatAt  *time.Time `db:"last_heartbeat_at" json:"last_heartbeat_at,omitempty"`
	LastReportedAt   *time.Time `db:"last_reported_at" json:"last_reported_at,omitempty"`
	LastSeenIP       *string    `db:"last_seen_ip" json:"last_seen_ip,omitempty"`
	RegisteredAt     *time.Time `db:"registered_at" json:"registered_at,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

func (a InstanceAgent) TableName() string {
	return "instance_agents"
}
