package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"clawreef/internal/models"
)

func TestInstanceExternalAccessShareLinkValidation(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)

	result, err := service.EnableShareLink(context.Background(), 42, 7, ExternalAccessExpirationRequest{Mode: ExternalAccessExpirationPermanent})
	if err != nil {
		t.Fatalf("EnableShareLink failed: %v", err)
	}
	if !strings.HasPrefix(result.ShareURL, "/s/") || strings.Contains(result.ShareURL, "token=") || strings.Contains(result.ShareURL, "/api/v1/") || strings.Contains(result.ShareURL, "/proxy") {
		t.Fatalf("unexpected short share URL: %q", result.ShareURL)
	}
	code := strings.Trim(strings.TrimPrefix(result.ShareURL, "/s/"), "/")
	if code == "" {
		t.Fatalf("short code missing from URL %q", result.ShareURL)
	}
	if result.Access.ShortCodeHash == nil || *result.Access.ShortCodeHash == code {
		t.Fatalf("expected stored short code to be hashed")
	}
	if result.Access.AuthMode != ExternalAccessModeShareLink {
		t.Fatalf("auth mode = %q, want %q", result.Access.AuthMode, ExternalAccessModeShareLink)
	}
	if result.Access.WorkspaceAccess != ExternalWorkspaceAccessNone {
		t.Fatalf("default workspace access = %q, want none", result.Access.WorkspaceAccess)
	}

	access, err := service.ValidateShortLink(context.Background(), code, "")
	if err != nil {
		t.Fatalf("ValidateShortLink failed: %v", err)
	}
	if access.InstanceID != 42 {
		t.Fatalf("validated instance ID = %d, want 42", access.InstanceID)
	}
	if repo.markUsedCount != 1 {
		t.Fatalf("mark used count = %d, want 1", repo.markUsedCount)
	}

	if _, err := service.ValidateShortLink(context.Background(), "wrong-code", ""); err == nil {
		t.Fatalf("expected wrong short code to fail")
	}
}

func TestInstanceExternalAccessWorkspaceScope(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)

	result, err := service.EnableShareLink(context.Background(), 42, 7, ExternalAccessExpirationRequest{
		Mode:            ExternalAccessExpirationPermanent,
		WorkspaceAccess: ExternalWorkspaceAccessWrite,
	})
	if err != nil {
		t.Fatalf("EnableShareLink failed: %v", err)
	}
	if result.Access.WorkspaceAccess != ExternalWorkspaceAccessWrite {
		t.Fatalf("workspace access = %q, want write", result.Access.WorkspaceAccess)
	}

	if _, err := service.CreatePassword(context.Background(), 42, 7, ExternalAccessExpirationRequest{
		Mode:            ExternalAccessExpirationPermanent,
		WorkspaceAccess: "administrator",
	}); err == nil {
		t.Fatal("unsupported workspace access must fail")
	}
}

func TestInstanceExternalAccessShareLinkPresetExpiration(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)
	before := time.Now().UTC().Add(24 * time.Hour)

	result, err := service.EnableShareLink(context.Background(), 42, 7, ExternalAccessExpirationRequest{
		Mode:   ExternalAccessExpirationPreset,
		Preset: ExternalAccessPreset24Hours,
	})
	if err != nil {
		t.Fatalf("EnableShareLink failed: %v", err)
	}
	if result.Access.ExpiresAt == nil {
		t.Fatal("preset expiration must set expires_at")
	}
	after := time.Now().UTC().Add(24 * time.Hour)
	if result.Access.ExpiresAt.Before(before) || result.Access.ExpiresAt.After(after.Add(time.Second)) {
		t.Fatalf("preset expires_at = %v, want around 24h from now", result.Access.ExpiresAt)
	}
}

func TestInstanceExternalAccessShareLinkCustomAndPermanentExpiration(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)
	customAt := time.Now().UTC().Add(3 * time.Hour).Truncate(time.Second)

	custom, err := service.EnableShareLink(context.Background(), 42, 7, ExternalAccessExpirationRequest{
		Mode:      ExternalAccessExpirationCustom,
		ExpiresAt: &customAt,
	})
	if err != nil {
		t.Fatalf("custom EnableShareLink failed: %v", err)
	}
	if custom.Access.ExpiresAt == nil || !custom.Access.ExpiresAt.Equal(customAt) {
		t.Fatalf("custom expires_at = %v, want %v", custom.Access.ExpiresAt, customAt)
	}

	permanent, err := service.EnableShareLink(context.Background(), 42, 7, ExternalAccessExpirationRequest{Mode: ExternalAccessExpirationPermanent})
	if err != nil {
		t.Fatalf("permanent EnableShareLink failed: %v", err)
	}
	if permanent.Access.ExpiresAt != nil {
		t.Fatalf("permanent expires_at = %v, want nil", permanent.Access.ExpiresAt)
	}
}

func TestInstanceExternalAccessPasswordDisable(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)

	result, err := service.CreatePassword(context.Background(), 100, 9, ExternalAccessExpirationRequest{Mode: ExternalAccessExpirationPermanent})
	if err != nil {
		t.Fatalf("CreatePassword failed: %v", err)
	}
	if !strings.HasPrefix(result.Password, "pwd_") {
		t.Fatalf("unexpected password prefix: %q", result.Password)
	}
	if result.Access.PasswordHash == nil || *result.Access.PasswordHash == result.Password {
		t.Fatalf("expected stored password to be hashed")
	}
	if result.Access.PasswordHint == nil || !strings.HasPrefix(result.Password, *result.Access.PasswordHint) {
		t.Fatalf("stored password hint does not match generated password")
	}
	if result.Access.AuthMode != ExternalAccessModePassword {
		t.Fatalf("auth mode = %q, want %q", result.Access.AuthMode, ExternalAccessModePassword)
	}
	if !strings.HasPrefix(result.ShareURL, "/s/") || strings.Contains(result.ShareURL, "token=") || strings.Contains(result.ShareURL, "/api/v1/") {
		t.Fatalf("unexpected password short URL: %q", result.ShareURL)
	}
	code := strings.Trim(strings.TrimPrefix(result.ShareURL, "/s/"), "/")

	if _, err := service.ValidateShortLink(context.Background(), code, result.Password); err != nil {
		t.Fatalf("ValidateShortLink with password failed: %v", err)
	}

	if err := service.Disable(context.Background(), 100); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	if _, err := service.ValidateShortLink(context.Background(), code, result.Password); err == nil {
		t.Fatalf("expected disabled password access to fail")
	}
}

func TestInstanceExternalAccessRestoresShareURLAfterRefresh(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)

	result, err := service.CreatePassword(context.Background(), 100, 9, ExternalAccessExpirationRequest{Mode: ExternalAccessExpirationPermanent})
	if err != nil {
		t.Fatalf("CreatePassword failed: %v", err)
	}
	reloaded, err := service.Get(context.Background(), 100)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got := ExternalAccessShareURL(reloaded); got != result.ShareURL {
		t.Fatalf("ExternalAccessShareURL after reload = %q, want %q", got, result.ShareURL)
	}
}

func TestInstanceExternalAccessRestoresPasswordAfterRefresh(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)

	result, err := service.CreatePassword(context.Background(), 100, 9, ExternalAccessExpirationRequest{Mode: ExternalAccessExpirationPermanent})
	if err != nil {
		t.Fatalf("CreatePassword failed: %v", err)
	}
	reloaded, err := service.Get(context.Background(), 100)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got := ExternalAccessPassword(reloaded); got != result.Password {
		t.Fatalf("ExternalAccessPassword after reload = %q, want generated password", got)
	}
}

func TestInstanceExternalAccessExpiration(t *testing.T) {
	repo := newFakeExternalAccessRepo()
	service := NewInstanceExternalAccessService(repo)
	expiredAt := time.Now().UTC().Add(-time.Minute)

	result, err := service.EnableShareLink(context.Background(), 77, 3, ExternalAccessExpirationRequest{
		Mode:      ExternalAccessExpirationCustom,
		ExpiresAt: &expiredAt,
	})
	if err != nil {
		t.Fatalf("EnableShareLink failed: %v", err)
	}
	code := strings.Trim(strings.TrimPrefix(result.ShareURL, "/s/"), "/")

	if _, err := service.ValidateShortLink(context.Background(), code, ""); err == nil {
		t.Fatalf("expected expired public access to fail")
	}
}

type fakeExternalAccessRepo struct {
	byInstance    map[int]*models.InstanceExternalAccess
	byCodeHash    map[string]*models.InstanceExternalAccess
	nextID        int64
	markUsedCount int
}

func newFakeExternalAccessRepo() *fakeExternalAccessRepo {
	return &fakeExternalAccessRepo{
		byInstance: make(map[int]*models.InstanceExternalAccess),
		byCodeHash: make(map[string]*models.InstanceExternalAccess),
		nextID:     1,
	}
}

func (r *fakeExternalAccessRepo) GetByInstanceID(ctx context.Context, instanceID int) (*models.InstanceExternalAccess, error) {
	_ = ctx
	return cloneExternalAccess(r.byInstance[instanceID]), nil
}

func (r *fakeExternalAccessRepo) GetByShortCodeHash(ctx context.Context, codeHash string) (*models.InstanceExternalAccess, error) {
	_ = ctx
	return cloneExternalAccess(r.byCodeHash[codeHash]), nil
}

func (r *fakeExternalAccessRepo) Upsert(ctx context.Context, access *models.InstanceExternalAccess) error {
	_ = ctx
	saved := cloneExternalAccess(access)
	if saved.ID == 0 {
		if existing := r.byInstance[saved.InstanceID]; existing != nil {
			saved.ID = existing.ID
		} else {
			saved.ID = r.nextID
			r.nextID++
		}
	}
	if existing := r.byInstance[saved.InstanceID]; existing != nil && existing.ShortCodeHash != nil {
		delete(r.byCodeHash, *existing.ShortCodeHash)
	}
	r.byInstance[saved.InstanceID] = cloneExternalAccess(saved)
	if saved.ShortCodeHash != nil {
		r.byCodeHash[*saved.ShortCodeHash] = cloneExternalAccess(saved)
	}
	return nil
}

func (r *fakeExternalAccessRepo) Disable(ctx context.Context, instanceID int) error {
	_ = ctx
	if existing := r.byInstance[instanceID]; existing != nil {
		existing.Enabled = false
		if existing.ShortCodeHash != nil {
			if byCode := r.byCodeHash[*existing.ShortCodeHash]; byCode != nil {
				byCode.Enabled = false
			}
		}
	}
	return nil
}

func (r *fakeExternalAccessRepo) MarkUsed(ctx context.Context, id int64) error {
	_ = ctx
	r.markUsedCount++
	now := time.Now().UTC()
	for _, access := range r.byInstance {
		if access.ID == id {
			access.LastUsedAt = &now
		}
	}
	for _, access := range r.byCodeHash {
		if access.ID == id {
			access.LastUsedAt = &now
		}
	}
	return nil
}

func cloneExternalAccess(access *models.InstanceExternalAccess) *models.InstanceExternalAccess {
	if access == nil {
		return nil
	}
	clone := *access
	if access.PublicSlug != nil {
		value := *access.PublicSlug
		clone.PublicSlug = &value
	}
	if access.PublicTokenHash != nil {
		value := *access.PublicTokenHash
		clone.PublicTokenHash = &value
	}
	if access.ShortCodeHash != nil {
		value := *access.ShortCodeHash
		clone.ShortCodeHash = &value
	}
	if access.PasswordHash != nil {
		value := *access.PasswordHash
		clone.PasswordHash = &value
	}
	if access.PasswordValue != nil {
		value := *access.PasswordValue
		clone.PasswordValue = &value
	}
	if access.PasswordHint != nil {
		value := *access.PasswordHint
		clone.PasswordHint = &value
	}
	if access.ExpiresAt != nil {
		value := *access.ExpiresAt
		clone.ExpiresAt = &value
	}
	if access.CreatedBy != nil {
		value := *access.CreatedBy
		clone.CreatedBy = &value
	}
	if access.LastUsedAt != nil {
		value := *access.LastUsedAt
		clone.LastUsedAt = &value
	}
	return &clone
}
