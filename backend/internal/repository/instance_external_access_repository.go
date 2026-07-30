package repository

import (
	"context"
	"fmt"
	"time"

	"clawreef/internal/models"
	"github.com/upper/db/v4"
)

type InstanceExternalAccessRepository interface {
	GetByInstanceID(ctx context.Context, instanceID int) (*models.InstanceExternalAccess, error)
	GetByShortCodeHash(ctx context.Context, codeHash string) (*models.InstanceExternalAccess, error)
	Upsert(ctx context.Context, access *models.InstanceExternalAccess) error
	Disable(ctx context.Context, instanceID int) error
	MarkUsed(ctx context.Context, id int64) error
}

type instanceExternalAccessRepository struct {
	sess db.Session
}

func NewInstanceExternalAccessRepository(sess db.Session) InstanceExternalAccessRepository {
	return &instanceExternalAccessRepository{sess: sess}
}

func (r *instanceExternalAccessRepository) GetByInstanceID(ctx context.Context, instanceID int) (*models.InstanceExternalAccess, error) {
	_ = ctx
	var access models.InstanceExternalAccess
	if err := r.sess.Collection(access.TableName()).Find(db.Cond{"instance_id": instanceID}).One(&access); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance external access: %w", err)
	}
	return &access, nil
}

func (r *instanceExternalAccessRepository) GetByShortCodeHash(ctx context.Context, codeHash string) (*models.InstanceExternalAccess, error) {
	_ = ctx
	var access models.InstanceExternalAccess
	if err := r.sess.Collection(access.TableName()).Find(db.Cond{"short_code_hash": codeHash}).One(&access); err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get instance external access by short code hash: %w", err)
	}
	return &access, nil
}

func (r *instanceExternalAccessRepository) Upsert(ctx context.Context, access *models.InstanceExternalAccess) error {
	now := time.Now().UTC()
	if access.CreatedAt.IsZero() {
		access.CreatedAt = now
	}
	access.UpdatedAt = now
	_, err := r.sess.SQL().ExecContext(ctx, `
		INSERT INTO instance_external_access (
			instance_id, enabled, auth_mode, workspace_access, public_slug, short_code_hash, public_token_hash,
			api_key_hash, password_value, api_key_prefix, expires_at, created_by,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			enabled = VALUES(enabled),
			auth_mode = VALUES(auth_mode),
			workspace_access = VALUES(workspace_access),
			public_slug = VALUES(public_slug),
			short_code_hash = VALUES(short_code_hash),
			public_token_hash = VALUES(public_token_hash),
			api_key_hash = VALUES(api_key_hash),
			password_value = VALUES(password_value),
			api_key_prefix = VALUES(api_key_prefix),
			expires_at = VALUES(expires_at),
			created_by = VALUES(created_by),
			updated_at = VALUES(updated_at)
	`, access.InstanceID, access.Enabled, access.AuthMode, access.WorkspaceAccess, access.PublicSlug, access.ShortCodeHash, access.PublicTokenHash,
		access.PasswordHash, access.PasswordValue, access.PasswordHint, access.ExpiresAt, access.CreatedBy,
		access.CreatedAt, access.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert instance external access: %w", err)
	}
	return nil
}

func (r *instanceExternalAccessRepository) Disable(ctx context.Context, instanceID int) error {
	_, err := r.sess.SQL().ExecContext(ctx, `
		UPDATE instance_external_access
		SET enabled = 0, updated_at = ?
		WHERE instance_id = ?
	`, time.Now().UTC(), instanceID)
	if err != nil {
		return fmt.Errorf("failed to disable instance external access: %w", err)
	}
	return nil
}

func (r *instanceExternalAccessRepository) MarkUsed(ctx context.Context, id int64) error {
	_, err := r.sess.SQL().ExecContext(ctx, `
		UPDATE instance_external_access
		SET last_used_at = ?, updated_at = ?
		WHERE id = ?
	`, time.Now().UTC(), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to mark instance external access used: %w", err)
	}
	return nil
}
