package models

import "time"

type InstanceCommand struct {
	ID             int        `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID     int        `db:"instance_id" json:"instance_id"`
	AgentID        *string    `db:"agent_id" json:"agent_id,omitempty"`
	CommandType    string     `db:"command_type" json:"command_type"`
	PayloadJSON    *string    `db:"payload_json" json:"-"`
	Status         string     `db:"status" json:"status"`
	IdempotencyKey string     `db:"idempotency_key" json:"idempotency_key"`
	IssuedBy       *int       `db:"issued_by" json:"issued_by,omitempty"`
	IssuedAt       time.Time  `db:"issued_at" json:"issued_at"`
	DispatchedAt   *time.Time `db:"dispatched_at" json:"dispatched_at,omitempty"`
	StartedAt      *time.Time `db:"started_at" json:"started_at,omitempty"`
	FinishedAt     *time.Time `db:"finished_at" json:"finished_at,omitempty"`
	TimeoutSeconds int        `db:"timeout_seconds" json:"timeout_seconds"`
	ResultJSON     *string    `db:"result_json" json:"-"`
	ErrorMessage   *string    `db:"error_message" json:"error_message,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

func (c InstanceCommand) TableName() string {
	return "instance_commands"
}
