package lexware

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Service.TriggerSync ---

func TestTriggerSync_CallsSyncContactsAndNeverErrors(t *testing.T) {
	configID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	var syncCalled bool
	repo := &mockRepository{
		getSyncConfigFn: func(_ context.Context, id uuid.UUID) (*models.LexwareSyncConfig, error) {
			syncCalled = true
			return &models.LexwareSyncConfig{ConfigID: id}, nil
		},
	}
	svc := newTestService(repo, cr, &mockVaultService{})

	// TriggerSync swallows SyncContacts' error (logs it) — assert it never
	// bubbles up, since a background trigger must not fail the caller for a
	// downstream sync hiccup.
	err := svc.TriggerSync(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, syncCalled)
}

func TestTriggerSync_SwallowsSyncContactsError(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not configured")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.TriggerSync(context.Background(), uuid.New())
	require.NoError(t, err)
}

// --- Service.GetFieldMappings / UpdateFieldMappings ---

func TestServiceGetFieldMappings_Success(t *testing.T) {
	configID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getFieldMappingsFn: func(_ context.Context, id uuid.UUID, entityType string) (*models.LexwareFieldMapping, error) {
			assert.Equal(t, configID, id)
			assert.Equal(t, "contact", entityType)
			return &models.LexwareFieldMapping{ConfigID: id, EntityType: entityType}, nil
		},
	}
	svc := newTestService(repo, cr, &mockVaultService{})

	fm, err := svc.GetFieldMappings(context.Background(), "contact")
	require.NoError(t, err)
	assert.Equal(t, "contact", fm.EntityType)
}

func TestServiceGetFieldMappings_NotConnected(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	_, err := svc.GetFieldMappings(context.Background(), "contact")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestServiceUpdateFieldMappings_InvalidMappingRejectedBeforeConfigLookup(t *testing.T) {
	// getByPlatformFn is intentionally left nil (errors "not found" by
	// default) — an invalid mapping must fail validation before the service
	// ever looks up the config, proving ValidateFieldMappings runs first.
	cr := &mockConfigRepo{}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.UpdateFieldMappings(context.Background(), "contact", []models.LexwareFieldMappingEntry{
		{KmuhubField: "", LexwareField: "person.firstName", Direction: "both"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFieldMapping)
}

func TestServiceUpdateFieldMappings_Success(t *testing.T) {
	configID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	var saved *models.LexwareFieldMapping
	repo := &mockRepository{
		upsertFieldMappingsFn: func(_ context.Context, fm *models.LexwareFieldMapping) error {
			saved = fm
			return nil
		},
	}
	svc := newTestService(repo, cr, &mockVaultService{})

	mappings := []models.LexwareFieldMappingEntry{
		{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "both"},
	}
	err := svc.UpdateFieldMappings(context.Background(), "contact", mappings)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, configID, saved.ConfigID)
	assert.Equal(t, "contact", saved.EntityType)
	assert.Equal(t, mappings, saved.Mappings)
}

// --- Service.StartScheduler / StopScheduler ---

func TestServiceStartScheduler_DelegatesAndReturnsNilWhenNoActiveIntegration(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	// No active integration: StartScheduler must not error just because the
	// tenant hasn't connected Lexware yet.
	err := svc.StartScheduler(context.Background())
	require.NoError(t, err)
}

func TestServiceStopScheduler_NoopWhenNothingRunning(t *testing.T) {
	svc := newTestService(&mockRepository{}, &mockConfigRepo{}, &mockVaultService{})
	// Must not panic on an empty scheduler.
	svc.StopScheduler()
}

// --- Scheduler.StartAll / StopAll ---

func TestSchedulerStartAll_NoActiveConfigIsNotAnError(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	sched := NewScheduler(nil, &mockRepository{}, cr)

	err := sched.StartAll(context.Background())
	require.NoError(t, err)
}

func TestSchedulerStartAll_InactiveIntegrationIsNotAnError(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: uuid.New(), IsActive: false}, nil
		},
	}
	sched := NewScheduler(nil, &mockRepository{}, cr)

	err := sched.StartAll(context.Background())
	require.NoError(t, err)
}

func TestSchedulerStartAll_ActiveIntegrationStartsTenantAndStopAllCleansUp(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, TenantID: tenantID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getSyncConfigFn: func(_ context.Context, id uuid.UUID) (*models.LexwareSyncConfig, error) {
			return &models.LexwareSyncConfig{ConfigID: id, ContactSyncIntervalMin: 15}, nil
		},
	}
	sched := NewScheduler(nil, repo, cr)

	require.NoError(t, sched.StartAll(context.Background()))
	require.Len(t, sched.tenants, 1)

	// StopAll must cancel every running tenant scheduler and leave the map
	// empty — a leaked goroutine here would keep polling a disconnected
	// tenant forever.
	sched.StopAll()
	assert.Empty(t, sched.tenants)
}

func TestSchedulerAddTenant_RepoErrorLeavesNoTenantRegistered(t *testing.T) {
	repo := &mockRepository{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.LexwareSyncConfig, error) {
			return nil, errors.New("db down")
		},
	}
	sched := NewScheduler(nil, repo, &mockConfigRepo{})

	err := sched.AddTenant(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Empty(t, sched.tenants, "a failed AddTenant must not leave a half-registered scheduler")
}

// --- Client / types trivia ---

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()
	assert.Equal(t, "https://api.lexware.io", cfg.BaseURL)
}

func TestClient_APIKeyManager_ReturnsWiredManager(t *testing.T) {
	vault := &mockVaultService{}
	c := NewClient(DefaultClientConfig(), vault)
	assert.NotNil(t, c.APIKeyManager())
}
