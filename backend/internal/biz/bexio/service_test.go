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

// --- Mock implementations ---

type mockRepository struct {
	getSyncConfigFn       func(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error)
	listEntityMappingsFn  func(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error)
	getLatestSyncLogFn    func(ctx context.Context, configID uuid.UUID, syncType string) (*models.BexioSyncLog, error)
	listSyncLogsFn        func(ctx context.Context, configID uuid.UUID, limit int) ([]models.BexioSyncLog, error)
	getFieldMappingsFn    func(ctx context.Context, configID uuid.UUID, entityType string) (*models.BexioFieldMapping, error)
	upsertFieldMappingsFn func(ctx context.Context, mapping *models.BexioFieldMapping) error
}

func (m *mockRepository) GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error) {
	if m.getSyncConfigFn != nil {
		return m.getSyncConfigFn(ctx, configID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) UpsertSyncConfig(context.Context, *models.BexioSyncConfig) error {
	return nil
}

func (m *mockRepository) UpdateLastSyncTime(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}

func (m *mockRepository) GetEntityMapping(context.Context, uuid.UUID, string, uuid.UUID) (*models.BexioEntityMapping, error) {
	return nil, nil
}

func (m *mockRepository) GetEntityMappingByBexioID(context.Context, uuid.UUID, string, int) (*models.BexioEntityMapping, error) {
	return nil, nil
}

func (m *mockRepository) UpsertEntityMapping(context.Context, *models.BexioEntityMapping) error {
	return nil
}

func (m *mockRepository) ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
	if m.listEntityMappingsFn != nil {
		return m.listEntityMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}

func (m *mockRepository) DeleteEntityMapping(context.Context, uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.BexioFieldMapping, error) {
	if m.getFieldMappingsFn != nil {
		return m.getFieldMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}

func (m *mockRepository) UpsertFieldMappings(ctx context.Context, mapping *models.BexioFieldMapping) error {
	if m.upsertFieldMappingsFn != nil {
		return m.upsertFieldMappingsFn(ctx, mapping)
	}
	return nil
}

func (m *mockRepository) CreateSyncLog(context.Context, *models.BexioSyncLog) error {
	return nil
}

func (m *mockRepository) UpdateSyncLog(context.Context, *models.BexioSyncLog) error {
	return nil
}

func (m *mockRepository) ListSyncLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.BexioSyncLog, error) {
	if m.listSyncLogsFn != nil {
		return m.listSyncLogsFn(ctx, configID, limit)
	}
	return nil, nil
}

func (m *mockRepository) GetLatestSyncLog(ctx context.Context, configID uuid.UUID, syncType string) (*models.BexioSyncLog, error) {
	if m.getLatestSyncLogFn != nil {
		return m.getLatestSyncLogFn(ctx, configID, syncType)
	}
	return nil, nil
}

type mockConfigRepo struct {
	getByPlatformFn func(ctx context.Context, platform string) (*IntegrationConfig, error)
}

func (m *mockConfigRepo) GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error) {
	if m.getByPlatformFn != nil {
		return m.getByPlatformFn(ctx, platform)
	}
	return nil, errors.New("not found")
}

func (m *mockConfigRepo) ListActiveByPlatform(_ context.Context, _ string) ([]*IntegrationConfig, error) {
	return nil, nil
}

func (m *mockConfigRepo) Upsert(context.Context, *IntegrationConfig) error { return nil }

func (m *mockConfigRepo) Deactivate(context.Context, uuid.UUID) error { return nil }

type mockEmitter struct {
	emittedEvents []EventPayload
}

func (m *mockEmitter) Emit(_ context.Context, payload EventPayload) error {
	m.emittedEvents = append(m.emittedEvents, payload)
	return nil
}

// --- Helper ---

func newTestService(repo Repository, configRepo IntegrationConfigRepo, emitter EventEmitter) *Service {
	if emitter == nil {
		emitter = noopEmitter{}
	}
	return &Service{
		repo:       repo,
		configRepo: configRepo,
		emitter:    emitter,
	}
}

// --- Tests ---

func TestGetConfigID_Success(t *testing.T) {
	configID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(_ context.Context, platform string) (*IntegrationConfig, error) {
			assert.Equal(t, "bexio", platform)
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	svc := newTestService(nil, cr, nil)

	id, err := svc.getConfigID(context.Background())

	require.NoError(t, err)
	assert.Equal(t, configID, id)
}

func TestGetConfigID_NotConfigured(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(nil, cr, nil)

	_, err := svc.getConfigID(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestGetConfigID_Inactive(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: uuid.New(), IsActive: false}, nil
		},
	}
	svc := newTestService(nil, cr, nil)

	_, err := svc.getConfigID(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestGetSyncStatus_Success(t *testing.T) {
	configID := uuid.New()
	now := time.Now().UTC()
	errMsg := "sync failed"

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getSyncConfigFn: func(_ context.Context, id uuid.UUID) (*models.BexioSyncConfig, error) {
			assert.Equal(t, configID, id)
			return &models.BexioSyncConfig{
				ContactSyncEnabled: true,
				InvoicePushEnabled: true,
				QuotePushEnabled:   false,
				PaymentPollEnabled: true,
				LastContactSyncAt:  &now,
			}, nil
		},
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
			switch entityType {
			case "contact":
				return make([]models.BexioEntityMapping, 5), nil
			case "invoice":
				return make([]models.BexioEntityMapping, 3), nil
			case "quote":
				return make([]models.BexioEntityMapping, 1), nil
			}
			return nil, nil
		},
		getLatestSyncLogFn: func(_ context.Context, _ uuid.UUID, _ string) (*models.BexioSyncLog, error) {
			return &models.BexioSyncLog{
				Status:       "failed",
				ErrorMessage: &errMsg,
				StartedAt:    now,
			}, nil
		},
	}
	svc := newTestService(repo, cr, nil)

	status, err := svc.GetSyncStatus(context.Background())

	require.NoError(t, err)
	assert.True(t, status.ContactSyncEnabled)
	assert.True(t, status.InvoicePushEnabled)
	assert.False(t, status.QuotePushEnabled)
	assert.True(t, status.PaymentPollEnabled)
	assert.Equal(t, 5, status.TotalContactsMapped)
	assert.Equal(t, 3, status.TotalInvoicesMapped)
	assert.Equal(t, 1, status.TotalQuotesMapped)
	require.NotNil(t, status.LastSyncError)
	assert.Equal(t, &errMsg, status.LastSyncError)
	require.NotNil(t, status.LastSyncErrorAt)
}

func TestGetSyncStatus_WithError(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: uuid.New(), IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, cr, nil)

	_, err := svc.GetSyncStatus(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "get sync status")
}

func TestGetSyncLogs_Success(t *testing.T) {
	configID := uuid.New()
	expectedLogs := []models.BexioSyncLog{
		{ID: uuid.New(), ConfigID: configID, SyncType: "contact_delta", Status: "completed"},
		{ID: uuid.New(), ConfigID: configID, SyncType: "payment_poll", Status: "completed"},
	}

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		listSyncLogsFn: func(_ context.Context, id uuid.UUID, limit int) ([]models.BexioSyncLog, error) {
			assert.Equal(t, configID, id)
			assert.Equal(t, 10, limit)
			return expectedLogs, nil
		},
	}
	svc := newTestService(repo, cr, nil)

	logs, err := svc.GetSyncLogs(context.Background(), 10)

	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestGetFieldMappings_Success(t *testing.T) {
	configID := uuid.New()
	expected := &models.BexioFieldMapping{
		ConfigID:   configID,
		EntityType: "contact",
		Mappings:   DefaultContactMappings(),
	}

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getFieldMappingsFn: func(_ context.Context, id uuid.UUID, entityType string) (*models.BexioFieldMapping, error) {
			assert.Equal(t, configID, id)
			assert.Equal(t, "contact", entityType)
			return expected, nil
		},
	}
	svc := newTestService(repo, cr, nil)

	fm, err := svc.GetFieldMappings(context.Background(), "contact")

	require.NoError(t, err)
	assert.Equal(t, "contact", fm.EntityType)
	assert.Len(t, fm.Mappings, len(DefaultContactMappings()))
}

func TestUpdateFieldMappings_Success(t *testing.T) {
	configID := uuid.New()
	mappings := []models.BexioFieldMappingEntry{
		{KmuhubField: "first_name", BexioField: "name_1", Direction: "both", Required: true},
		{KmuhubField: "email", BexioField: "mail", Direction: "both", Required: false},
	}

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	var savedMapping *models.BexioFieldMapping
	repo := &mockRepository{
		upsertFieldMappingsFn: func(_ context.Context, fm *models.BexioFieldMapping) error {
			savedMapping = fm
			return nil
		},
	}
	svc := newTestService(repo, cr, nil)

	err := svc.UpdateFieldMappings(context.Background(), "contact", mappings)

	require.NoError(t, err)
	require.NotNil(t, savedMapping)
	assert.Equal(t, configID, savedMapping.ConfigID)
	assert.Equal(t, "contact", savedMapping.EntityType)
	assert.Len(t, savedMapping.Mappings, 2)
}

func TestUpdateFieldMappings_InvalidEntityType(t *testing.T) {
	// contact mapping missing required name_1 field
	mappings := []models.BexioFieldMappingEntry{
		{KmuhubField: "email", BexioField: "mail", Direction: "both", Required: false},
	}

	svc := newTestService(nil, nil, nil)

	err := svc.UpdateFieldMappings(context.Background(), "contact", mappings)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFieldMapping)
	assert.Contains(t, err.Error(), "name_1")
}

func TestSetEventEmitter(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	emitter := &mockEmitter{}

	svc.SetEventEmitter(emitter)

	assert.Equal(t, emitter, svc.emitter)
}
