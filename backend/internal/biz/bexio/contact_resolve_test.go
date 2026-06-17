package bexio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- mock ContactService for resolve tests ---

type mockContactService struct {
	getByEmailFn func(ctx context.Context, email string) (*ContactResult, error)
}

func (m *mockContactService) GetByID(_ context.Context, _ uuid.UUID) (*ContactResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockContactService) GetByEmail(ctx context.Context, email string) (*ContactResult, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockContactService) CreateForSync(_ context.Context, _ *ContactSyncData, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not implemented")
}

func (m *mockContactService) UpdateForSync(_ context.Context, _ uuid.UUID, _ *ContactSyncData) error {
	return errors.New("not implemented")
}

func (m *mockContactService) ListModifiedSince(_ context.Context, _ time.Time) ([]ContactResult, error) {
	return nil, errors.New("not implemented")
}

// --- mock Repository for resolve tests (minimal) ---

type resolveTestRepo struct {
	getEntityMappingFn    func(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.BexioEntityMapping, error)
	listEntityMappingsFn  func(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error)
}

func (r *resolveTestRepo) GetSyncConfig(_ context.Context, _ uuid.UUID) (*models.BexioSyncConfig, error) {
	return nil, errors.New("not implemented")
}
func (r *resolveTestRepo) UpsertSyncConfig(_ context.Context, _ *models.BexioSyncConfig) error {
	return nil
}
func (r *resolveTestRepo) UpdateLastSyncTime(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (r *resolveTestRepo) GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.BexioEntityMapping, error) {
	if r.getEntityMappingFn != nil {
		return r.getEntityMappingFn(ctx, configID, entityType, kmuhubID)
	}
	return nil, ErrMappingNotFound
}
func (r *resolveTestRepo) GetEntityMappingByBexioID(_ context.Context, _ uuid.UUID, _ string, _ int) (*models.BexioEntityMapping, error) {
	return nil, ErrMappingNotFound
}
func (r *resolveTestRepo) UpsertEntityMapping(_ context.Context, _ *models.BexioEntityMapping) error {
	return nil
}
func (r *resolveTestRepo) ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
	if r.listEntityMappingsFn != nil {
		return r.listEntityMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}
func (r *resolveTestRepo) DeleteEntityMapping(_ context.Context, _ uuid.UUID) error { return nil }
func (r *resolveTestRepo) GetFieldMappings(_ context.Context, _ uuid.UUID, _ string) (*models.BexioFieldMapping, error) {
	return nil, nil
}
func (r *resolveTestRepo) UpsertFieldMappings(_ context.Context, _ *models.BexioFieldMapping) error {
	return nil
}
func (r *resolveTestRepo) CreateSyncLog(_ context.Context, _ *models.BexioSyncLog) error { return nil }
func (r *resolveTestRepo) UpdateSyncLog(_ context.Context, _ *models.BexioSyncLog) error { return nil }
func (r *resolveTestRepo) ListSyncLogs(_ context.Context, _ uuid.UUID, _ int) ([]models.BexioSyncLog, error) {
	return nil, nil
}
func (r *resolveTestRepo) GetLatestSyncLog(_ context.Context, _ uuid.UUID, _ string) (*models.BexioSyncLog, error) {
	return nil, nil
}

// --- Tests ---

func TestResolveContactBexioIDByEmail_EmptyEmail(t *testing.T) {
	_, err := resolveContactBexioIDByEmail(context.Background(), &resolveTestRepo{}, nil, uuid.New(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email is empty")
}

func TestResolveContactBexioIDByEmail_WithContactService_HappyPath(t *testing.T) {
	configID := uuid.New()
	localID := uuid.New()
	bexioID := 42

	contactSvc := &mockContactService{
		getByEmailFn: func(_ context.Context, email string) (*ContactResult, error) {
			assert.Equal(t, "customer@example.com", email)
			return &ContactResult{ID: localID}, nil
		},
	}
	repo := &resolveTestRepo{
		getEntityMappingFn: func(_ context.Context, cID uuid.UUID, entityType string, kID uuid.UUID) (*models.BexioEntityMapping, error) {
			assert.Equal(t, configID, cID)
			assert.Equal(t, "contact", entityType)
			assert.Equal(t, localID, kID)
			return &models.BexioEntityMapping{BexioID: bexioID}, nil
		},
	}

	got, err := resolveContactBexioIDByEmail(context.Background(), repo, contactSvc, configID, "customer@example.com")
	require.NoError(t, err)
	assert.Equal(t, bexioID, got)
}

func TestResolveContactBexioIDByEmail_WithContactService_EmailNotFound(t *testing.T) {
	contactSvc := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return nil, errors.New("not found")
		},
	}

	_, err := resolveContactBexioIDByEmail(context.Background(), &resolveTestRepo{}, contactSvc, uuid.New(), "nobody@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup local contact by email")
}

func TestResolveContactBexioIDByEmail_WithContactService_NoMapping(t *testing.T) {
	localID := uuid.New()
	contactSvc := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: localID}, nil
		},
	}
	repo := &resolveTestRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			return nil, ErrMappingNotFound
		},
	}

	_, err := resolveContactBexioIDByEmail(context.Background(), repo, contactSvc, uuid.New(), "synced@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no bexio mapping for contact")
}

func TestResolveContactBexioIDByEmail_NoContactService_SingleMapping(t *testing.T) {
	bexioID := 7
	repo := &resolveTestRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{{BexioID: bexioID}}, nil
		},
	}

	got, err := resolveContactBexioIDByEmail(context.Background(), repo, nil, uuid.New(), "solo@example.com")
	require.NoError(t, err)
	assert.Equal(t, bexioID, got)
}

func TestResolveContactBexioIDByEmail_NoContactService_MultipleMapping_Errors(t *testing.T) {
	repo := &resolveTestRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{
				{BexioID: 1},
				{BexioID: 2},
			}, nil
		},
	}

	_, err := resolveContactBexioIDByEmail(context.Background(), repo, nil, uuid.New(), "ambiguous@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 contact mappings exist")
	assert.Contains(t, err.Error(), "contact service is not wired")
}

func TestResolveContactBexioIDByEmail_NoContactService_NoMapping_Errors(t *testing.T) {
	repo := &resolveTestRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return nil, nil
		},
	}

	_, err := resolveContactBexioIDByEmail(context.Background(), repo, nil, uuid.New(), "none@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no bexio contact mapping found")
}
