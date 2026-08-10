package lexware

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
	getSyncConfigFn       func(ctx context.Context, configID uuid.UUID) (*models.LexwareSyncConfig, error)
	upsertSyncConfigFn    func(ctx context.Context, config *models.LexwareSyncConfig) error
	listEntityMappingsFn  func(ctx context.Context, configID uuid.UUID, entityType string) ([]models.LexwareEntityMapping, error)
	getLatestSyncLogFn    func(ctx context.Context, configID uuid.UUID, syncType string) (*models.LexwareSyncLog, error)
	listSyncLogsFn        func(ctx context.Context, configID uuid.UUID, limit int) ([]models.LexwareSyncLog, error)
	getFieldMappingsFn    func(ctx context.Context, configID uuid.UUID, entityType string) (*models.LexwareFieldMapping, error)
	upsertFieldMappingsFn func(ctx context.Context, mapping *models.LexwareFieldMapping) error
	createSyncLogFn       func(ctx context.Context, log *models.LexwareSyncLog) error
	updateSyncLogFn       func(ctx context.Context, log *models.LexwareSyncLog) error
	updateLastSyncTimeFn  func(ctx context.Context, configID uuid.UUID, syncType string, syncedAt time.Time) error

	getEntityMappingFn            func(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.LexwareEntityMapping, error)
	getEntityMappingByLexwareIDFn func(ctx context.Context, configID uuid.UUID, entityType, lexwareID string) (*models.LexwareEntityMapping, error)
	upsertEntityMappingFn         func(ctx context.Context, mapping *models.LexwareEntityMapping) error
}

func (m *mockRepository) GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.LexwareSyncConfig, error) {
	if m.getSyncConfigFn != nil {
		return m.getSyncConfigFn(ctx, configID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) UpsertSyncConfig(ctx context.Context, config *models.LexwareSyncConfig) error {
	if m.upsertSyncConfigFn != nil {
		return m.upsertSyncConfigFn(ctx, config)
	}
	return nil
}

func (m *mockRepository) UpdateLastSyncTime(ctx context.Context, configID uuid.UUID, syncType string, syncedAt time.Time) error {
	if m.updateLastSyncTimeFn != nil {
		return m.updateLastSyncTimeFn(ctx, configID, syncType, syncedAt)
	}
	return nil
}

func (m *mockRepository) GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.LexwareEntityMapping, error) {
	if m.getEntityMappingFn != nil {
		return m.getEntityMappingFn(ctx, configID, entityType, kmuhubID)
	}
	return nil, nil
}

func (m *mockRepository) GetEntityMappingByLexwareID(ctx context.Context, configID uuid.UUID, entityType, lexwareID string) (*models.LexwareEntityMapping, error) {
	if m.getEntityMappingByLexwareIDFn != nil {
		return m.getEntityMappingByLexwareIDFn(ctx, configID, entityType, lexwareID)
	}
	return nil, nil
}

func (m *mockRepository) UpsertEntityMapping(ctx context.Context, mapping *models.LexwareEntityMapping) error {
	if m.upsertEntityMappingFn != nil {
		return m.upsertEntityMappingFn(ctx, mapping)
	}
	return nil
}

func (m *mockRepository) ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.LexwareEntityMapping, error) {
	if m.listEntityMappingsFn != nil {
		return m.listEntityMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}

func (m *mockRepository) DeleteEntityMapping(context.Context, uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.LexwareFieldMapping, error) {
	if m.getFieldMappingsFn != nil {
		return m.getFieldMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}

func (m *mockRepository) UpsertFieldMappings(ctx context.Context, mapping *models.LexwareFieldMapping) error {
	if m.upsertFieldMappingsFn != nil {
		return m.upsertFieldMappingsFn(ctx, mapping)
	}
	return nil
}

func (m *mockRepository) CreateSyncLog(ctx context.Context, log *models.LexwareSyncLog) error {
	if m.createSyncLogFn != nil {
		return m.createSyncLogFn(ctx, log)
	}
	return nil
}

func (m *mockRepository) UpdateSyncLog(ctx context.Context, log *models.LexwareSyncLog) error {
	if m.updateSyncLogFn != nil {
		return m.updateSyncLogFn(ctx, log)
	}
	return nil
}

func (m *mockRepository) ListSyncLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.LexwareSyncLog, error) {
	if m.listSyncLogsFn != nil {
		return m.listSyncLogsFn(ctx, configID, limit)
	}
	return nil, nil
}

func (m *mockRepository) GetLatestSyncLog(ctx context.Context, configID uuid.UUID, syncType string) (*models.LexwareSyncLog, error) {
	if m.getLatestSyncLogFn != nil {
		return m.getLatestSyncLogFn(ctx, configID, syncType)
	}
	return nil, nil
}

func (m *mockRepository) UpsertWebhookSubscription(context.Context, uuid.UUID, uuid.UUID, string, string, string) error {
	return nil
}

func (m *mockRepository) ListWebhookSubscriptions(context.Context, uuid.UUID) ([]WebhookSubscriptionRecord, error) {
	return nil, nil
}

func (m *mockRepository) DeleteWebhookSubscription(context.Context, uuid.UUID) error {
	return nil
}

type mockConfigRepo struct {
	getByPlatformFn func(ctx context.Context, platform string) (*IntegrationConfig, error)
	upsertFn        func(ctx context.Context, config *IntegrationConfig) error
	deactivateFn    func(ctx context.Context, id uuid.UUID) error
}

func (m *mockConfigRepo) GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error) {
	if m.getByPlatformFn != nil {
		return m.getByPlatformFn(ctx, platform)
	}
	return nil, errors.New("not found")
}

func (m *mockConfigRepo) Upsert(ctx context.Context, config *IntegrationConfig) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, config)
	}
	return nil
}

func (m *mockConfigRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	if m.deactivateFn != nil {
		return m.deactivateFn(ctx, id)
	}
	return nil
}

type mockVaultService struct {
	getSecretFn func(ctx context.Context, keyName string) (string, error)
	setSecretFn func(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error
}

func (m *mockVaultService) GetSecret(ctx context.Context, keyName string) (string, error) {
	if m.getSecretFn != nil {
		return m.getSecretFn(ctx, keyName)
	}
	return "", nil
}

func (m *mockVaultService) SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error {
	if m.setSecretFn != nil {
		return m.setSecretFn(ctx, keyName, plaintext, description, createdBy)
	}
	return nil
}

type mockEmitter struct {
	emittedEvents []EventPayload
}

func (m *mockEmitter) Emit(_ context.Context, payload EventPayload) error {
	m.emittedEvents = append(m.emittedEvents, payload)
	return nil
}

// --- Helper ---

// newTestService builds a service without an API client — for the paths that
// fail before any Lexware call is made.  It goes through the real constructor
// so the wired syncer/pusher/webhook-handler are the ones under test.
func newTestService(repo *mockRepository, configRepo *mockConfigRepo, vault *mockVaultService) *Service {
	return NewService(nil, repo, configRepo, vault, nil, nil, nil)
}

// --- Tests ---

func TestConnect_Success(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()

	var storedKey string
	vault := &mockVaultService{
		setSecretFn: func(_ context.Context, keyName, plaintext, _ string, createdBy uuid.UUID) error {
			storedKey = plaintext
			assert.Contains(t, keyName, tenantID.String())
			assert.Equal(t, tenantID, createdBy)
			return nil
		},
	}

	var savedConfig *IntegrationConfig
	cr := &mockConfigRepo{
		upsertFn: func(_ context.Context, config *IntegrationConfig) error {
			// Simulate DB assigning an ID
			config.ID = configID
			savedConfig = config
			return nil
		},
	}

	var savedSyncConfig *models.LexwareSyncConfig
	repo := &mockRepository{
		upsertSyncConfigFn: func(_ context.Context, config *models.LexwareSyncConfig) error {
			savedSyncConfig = config
			return nil
		},
		// Scheduler.AddTenant calls GetSyncConfig, provide a response
		getSyncConfigFn: func(_ context.Context, id uuid.UUID) (*models.LexwareSyncConfig, error) {
			return &models.LexwareSyncConfig{
				ConfigID:               id,
				ContactSyncEnabled:     true,
				ContactSyncIntervalMin: 15,
			}, nil
		},
	}

	emitter := &mockEmitter{}
	svc := newTestService(repo, cr, vault)
	svc.emitter = emitter

	err := svc.Connect(context.Background(), tenantID, "test-api-key-123")

	require.NoError(t, err)
	assert.Equal(t, "test-api-key-123", storedKey)
	require.NotNil(t, savedConfig)
	assert.Equal(t, "lexware", savedConfig.Platform)
	assert.True(t, savedConfig.IsActive)
	require.NotNil(t, savedSyncConfig)
	assert.Equal(t, configID, savedSyncConfig.ConfigID)
	assert.True(t, savedSyncConfig.ContactSyncEnabled)
	assert.Equal(t, 15, savedSyncConfig.ContactSyncIntervalMin)

	// Verify event emitted
	require.Len(t, emitter.emittedEvents, 1)
	assert.Equal(t, EventConnected, emitter.emittedEvents[0].Type)
}

func TestConnect_VaultError(t *testing.T) {
	vault := &mockVaultService{
		setSecretFn: func(context.Context, string, string, string, uuid.UUID) error {
			return errors.New("vault unavailable")
		},
	}
	svc := newTestService(&mockRepository{}, &mockConfigRepo{}, vault)

	err := svc.Connect(context.Background(), uuid.New(), "key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "store api key")
}

func TestDisconnect_Success(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	var deactivatedID uuid.UUID
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
		deactivateFn: func(_ context.Context, id uuid.UUID) error {
			deactivatedID = id
			return nil
		},
	}
	emitter := &mockEmitter{}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})
	svc.emitter = emitter

	err := svc.Disconnect(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, configID, deactivatedID)
	require.Len(t, emitter.emittedEvents, 1)
	assert.Equal(t, EventDisconnected, emitter.emittedEvents[0].Type)
}

func TestDisconnect_NotConfigured(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.Disconnect(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "get config for disconnect")
}

func TestTestConnection_NoClient(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{
				ID:                  uuid.New(),
				IsActive:            true,
				CredentialsVaultKey: "lexware_api_key_test",
			}, nil
		},
	}
	vault := &mockVaultService{
		getSecretFn: func(_ context.Context, keyName string) (string, error) {
			assert.Equal(t, "lexware_api_key_test", keyName)
			return "valid-api-key", nil
		},
	}
	svc := newTestService(&mockRepository{}, cr, vault)

	err := svc.TestConnection(context.Background())

	// A stored key alone is not a connection: without an API client there is
	// nothing to verify it against, and reporting success would be a lie.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api client is not configured")
}

func TestTestConnection_NotConnected(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.TestConnection(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestTestConnection_Inactive(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: uuid.New(), IsActive: false}, nil
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.TestConnection(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestTestConnection_EmptyKey(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{
				ID:                  uuid.New(),
				IsActive:            true,
				CredentialsVaultKey: "lexware_api_key_test",
			}, nil
		},
	}
	vault := &mockVaultService{
		getSecretFn: func(context.Context, string) (string, error) {
			return "", nil
		},
	}
	svc := newTestService(&mockRepository{}, cr, vault)

	err := svc.TestConnection(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key is empty")
}

func TestGetConnectionStatus_Connected(t *testing.T) {
	configID := uuid.New()
	connectedAt := time.Now().UTC()

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{
				ID:        configID,
				IsActive:  true,
				CreatedAt: connectedAt,
			}, nil
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	status, err := svc.GetConnectionStatus(context.Background())

	require.NoError(t, err)
	assert.True(t, status.Connected)
	require.NotNil(t, status.ConfigID)
	assert.Equal(t, configID, *status.ConfigID)
	require.NotNil(t, status.ConnectedAt)
}

func TestGetConnectionStatus_NotConnected(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	status, err := svc.GetConnectionStatus(context.Background())

	require.NoError(t, err)
	assert.False(t, status.Connected)
}

func TestGetSyncStatus_Success(t *testing.T) {
	configID := uuid.New()
	now := time.Now().UTC()

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		getSyncConfigFn: func(_ context.Context, id uuid.UUID) (*models.LexwareSyncConfig, error) {
			return &models.LexwareSyncConfig{
				ContactSyncEnabled: true,
				InvoicePushEnabled: true,
				QuotePushEnabled:   false,
				WebhookEnabled:     true,
				LastContactSyncAt:  &now,
			}, nil
		},
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, entityType string) ([]models.LexwareEntityMapping, error) {
			switch entityType {
			case "contact":
				return make([]models.LexwareEntityMapping, 4), nil
			case "invoice":
				return make([]models.LexwareEntityMapping, 2), nil
			case "quote":
				return nil, nil
			}
			return nil, nil
		},
		getLatestSyncLogFn: func(context.Context, uuid.UUID, string) (*models.LexwareSyncLog, error) {
			return nil, nil // No errors in sync log
		},
	}
	svc := newTestService(repo, cr, &mockVaultService{})

	status, err := svc.GetSyncStatus(context.Background())

	require.NoError(t, err)
	assert.True(t, status.ContactSyncEnabled)
	assert.True(t, status.InvoicePushEnabled)
	assert.False(t, status.QuotePushEnabled)
	assert.True(t, status.WebhookEnabled)
	assert.Equal(t, 4, status.TotalContactsMapped)
	assert.Equal(t, 2, status.TotalInvoicesMapped)
	assert.Equal(t, 0, status.TotalQuotesMapped)
	assert.Nil(t, status.LastSyncError)
}

func TestGetSyncLogs_Success(t *testing.T) {
	configID := uuid.New()
	expectedLogs := []models.LexwareSyncLog{
		{ID: uuid.New(), ConfigID: configID, SyncType: "contact_delta", Status: "completed"},
	}

	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configID, IsActive: true}, nil
		},
	}
	repo := &mockRepository{
		listSyncLogsFn: func(_ context.Context, id uuid.UUID, limit int) ([]models.LexwareSyncLog, error) {
			assert.Equal(t, configID, id)
			assert.Equal(t, 5, limit)
			return expectedLogs, nil
		},
	}
	svc := newTestService(repo, cr, &mockVaultService{})

	logs, err := svc.GetSyncLogs(context.Background(), 5)

	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestHandleWebhookEvent_Success(t *testing.T) {
	tenantID := uuid.New()
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: uuid.New(), TenantID: tenantID, IsActive: true}, nil
		},
	}
	emitter := &mockEmitter{}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})
	svc.SetEventEmitter(emitter)

	// "contact.updated" is not a Lexware event name (the API sends
	// "contact.changed") — it must be acknowledged, not dispatched.
	payload := map[string]any{"resource_id": "abc-123"}
	err := svc.HandleWebhookEvent(context.Background(), "contact.updated", payload)

	require.NoError(t, err)
	require.Len(t, emitter.emittedEvents, 1)
	assert.Equal(t, EventWebhookReceived, emitter.emittedEvents[0].Type)
	assert.Equal(t, "contact.updated", emitter.emittedEvents[0].Data["event_type"])
	assert.Equal(t, "abc-123", emitter.emittedEvents[0].Data["resource_id"])
}

func TestHandleWebhookEvent_NotConfigured(t *testing.T) {
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(&mockRepository{}, cr, &mockVaultService{})

	err := svc.HandleWebhookEvent(context.Background(), "contact.changed", map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unconfigured integration")
}
