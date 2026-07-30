package models

import "time"

type InstanceExternalAccess struct {
	ID              int64      `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID      int        `db:"instance_id" json:"instance_id"`
	Enabled         bool       `db:"enabled" json:"enabled"`
	AuthMode        string     `db:"auth_mode" json:"auth_mode"`
	WorkspaceAccess string     `db:"workspace_access" json:"workspace_access"`
	PublicSlug      *string    `db:"public_slug" json:"-"`
	PublicTokenHash *string    `db:"public_token_hash" json:"-"`
	ShortCodeHash   *string    `db:"short_code_hash" json:"-"`
	PasswordHash    *string    `db:"api_key_hash" json:"-"`
	PasswordValue   *string    `db:"password_value" json:"-"`
	PasswordHint    *string    `db:"api_key_prefix" json:"password_hint,omitempty"`
	ExpiresAt       *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	CreatedBy       *int       `db:"created_by" json:"created_by,omitempty"`
	LastUsedAt      *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

func (InstanceExternalAccess) TableName() string {
	return "instance_external_access"
}
