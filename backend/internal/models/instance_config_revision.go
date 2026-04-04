package models

import "time"

type InstanceConfigRevision struct {
	ID               int        `db:"id,primarykey,autoincrement" json:"id"`
	InstanceID       int        `db:"instance_id" json:"instance_id"`
	SourceSnapshotID *int       `db:"source_snapshot_id" json:"source_snapshot_id,omitempty"`
	SourceBundleID   *int       `db:"source_bundle_id" json:"source_bundle_id,omitempty"`
	RevisionNo       int        `db:"revision_no" json:"revision_no"`
	ContentJSON      string     `db:"content_json" json:"-"`
	Checksum         string     `db:"checksum" json:"checksum"`
	Status           string     `db:"status" json:"status"`
	PublishedBy      *int       `db:"published_by" json:"published_by,omitempty"`
	PublishedAt      *time.Time `db:"published_at" json:"published_at,omitempty"`
	ActivatedAt      *time.Time `db:"activated_at" json:"activated_at,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

func (r InstanceConfigRevision) TableName() string {
	return "instance_config_revisions"
}
