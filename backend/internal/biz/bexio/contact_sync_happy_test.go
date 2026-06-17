package bexio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- helpers ---

// newTestClient creates a *Client pointing at the given httptest.Server.
// The embedded TokenManager is pre-seeded with a valid in-memory token so
// tests never hit the vault / token endpoint unless they explicitly test that
// path.
func newTestClient(srv *httptest.Server) *Client {
	vault := &mockVaultService{token: "test-access-token"}
	cfg := ClientConfig{
		BaseURL:      srv.URL + "/",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     srv.URL + "/oauth/token",
	}
	c := NewClient(cfg, vault)
	// Pre-seed the token cache so the TokenManager never calls the vault for
	// the access token during test runs.
	tenantID := uuid.New() // will be overwritten per-test via seedToken
	_ = tenantID
	return c
}

// seedToken injects a pre-cached access token for the given tenant so test
// requests never trigger a token-refresh roundtrip.
func seedToken(c *Client, tenantID uuid.UUID) {
	c.tokenManager.mu.Lock()
	c.tokenManager.cache[tenantID] = &tokenCacheEntry{
		accessToken: "test-access-token",
		expiresAt:   time.Now().Add(1 * time.Hour),
	}
	c.tokenManager.mu.Unlock()
}

// mockVaultService is a minimal VaultService for tests.
type mockVaultService struct {
	token     string
	getErr    error
	setErr    error
}

func (m *mockVaultService) GetSecret(_ context.Context, _ string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.token, nil
}

func (m *mockVaultService) SetSecret(_ context.Context, _, _, _ string, _ uuid.UUID) error {
	return m.setErr
}

// fullSyncRepo is a Repository mock that supports all methods needed by
// ContactSyncer.SyncContacts.  Call-counts on UpsertEntityMapping let tests
// assert idempotency.
type fullSyncRepo struct {
	getSyncConfigFn           func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error)
	getFieldMappingsFn        func(context.Context, uuid.UUID, string) (*models.BexioFieldMapping, error)
	getEntityMappingFn        func(context.Context, uuid.UUID, string, uuid.UUID) (*models.BexioEntityMapping, error)
	getEntityMappingByBexioFn func(context.Context, uuid.UUID, string, int) (*models.BexioEntityMapping, error)
	upsertEntityMappingCalls  int
	upsertEntityMappingFn     func(context.Context, *models.BexioEntityMapping) error
	listEntityMappingsFn      func(context.Context, uuid.UUID, string) ([]models.BexioEntityMapping, error)
}

func (r *fullSyncRepo) GetSyncConfig(ctx context.Context, id uuid.UUID) (*models.BexioSyncConfig, error) {
	if r.getSyncConfigFn != nil {
		return r.getSyncConfigFn(ctx, id)
	}
	return &models.BexioSyncConfig{}, nil
}

func (r *fullSyncRepo) UpsertSyncConfig(_ context.Context, _ *models.BexioSyncConfig) error { return nil }
func (r *fullSyncRepo) UpdateLastSyncTime(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (r *fullSyncRepo) GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.BexioEntityMapping, error) {
	if r.getEntityMappingFn != nil {
		return r.getEntityMappingFn(ctx, configID, entityType, kmuhubID)
	}
	return nil, ErrMappingNotFound
}

func (r *fullSyncRepo) GetEntityMappingByBexioID(ctx context.Context, configID uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error) {
	if r.getEntityMappingByBexioFn != nil {
		return r.getEntityMappingByBexioFn(ctx, configID, entityType, bexioID)
	}
	return nil, ErrMappingNotFound
}

func (r *fullSyncRepo) UpsertEntityMapping(ctx context.Context, m *models.BexioEntityMapping) error {
	r.upsertEntityMappingCalls++
	if r.upsertEntityMappingFn != nil {
		return r.upsertEntityMappingFn(ctx, m)
	}
	return nil
}

func (r *fullSyncRepo) ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
	if r.listEntityMappingsFn != nil {
		return r.listEntityMappingsFn(ctx, configID, entityType)
	}
	return nil, nil
}

func (r *fullSyncRepo) DeleteEntityMapping(_ context.Context, _ uuid.UUID) error { return nil }

func (r *fullSyncRepo) GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.BexioFieldMapping, error) {
	if r.getFieldMappingsFn != nil {
		return r.getFieldMappingsFn(ctx, configID, entityType)
	}
	return nil, nil // nil = use defaults
}

func (r *fullSyncRepo) UpsertFieldMappings(_ context.Context, _ *models.BexioFieldMapping) error {
	return nil
}

func (r *fullSyncRepo) CreateSyncLog(_ context.Context, _ *models.BexioSyncLog) error { return nil }
func (r *fullSyncRepo) UpdateSyncLog(_ context.Context, _ *models.BexioSyncLog) error { return nil }

func (r *fullSyncRepo) ListSyncLogs(_ context.Context, _ uuid.UUID, _ int) ([]models.BexioSyncLog, error) {
	return nil, nil
}

func (r *fullSyncRepo) GetLatestSyncLog(_ context.Context, _ uuid.UUID, _ string) (*models.BexioSyncLog, error) {
	return nil, nil
}

// fullContactService is a ContactService mock with call counters.
type fullContactService struct {
	getByIDFn            func(context.Context, uuid.UUID) (*ContactResult, error)
	getByEmailFn         func(context.Context, string) (*ContactResult, error)
	createForSyncCalls   int
	createForSyncFn      func(context.Context, *ContactSyncData, uuid.UUID) (uuid.UUID, error)
	updateForSyncCalls   int
	updateForSyncFn      func(context.Context, uuid.UUID, *ContactSyncData) error
	listModifiedSinceFn  func(context.Context, time.Time) ([]ContactResult, error)
}

func (m *fullContactService) GetByID(ctx context.Context, id uuid.UUID) (*ContactResult, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *fullContactService) GetByEmail(ctx context.Context, email string) (*ContactResult, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *fullContactService) CreateForSync(ctx context.Context, data *ContactSyncData, createdBy uuid.UUID) (uuid.UUID, error) {
	m.createForSyncCalls++
	if m.createForSyncFn != nil {
		return m.createForSyncFn(ctx, data, createdBy)
	}
	return uuid.New(), nil
}

func (m *fullContactService) UpdateForSync(ctx context.Context, id uuid.UUID, data *ContactSyncData) error {
	m.updateForSyncCalls++
	if m.updateForSyncFn != nil {
		return m.updateForSyncFn(ctx, id, data)
	}
	return nil
}

func (m *fullContactService) ListModifiedSince(ctx context.Context, since time.Time) ([]ContactResult, error) {
	if m.listModifiedSinceFn != nil {
		return m.listModifiedSinceFn(ctx, since)
	}
	return nil, nil
}

// --- Tests ---

// TestContactSyncer_SyncContacts_GetSyncConfig_Error verifies that SyncContacts
// returns an error when the repository cannot load the sync config.
func TestContactSyncer_SyncContacts_GetSyncConfig_Error(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	repo := &fullSyncRepo{
		getSyncConfigFn: func(_ context.Context, _ uuid.UUID) (*models.BexioSyncConfig, error) {
			return nil, errors.New("db connection lost")
		},
	}
	syncer := &ContactSyncer{
		client:      nil, // not reached
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		contacts:    nil,
		lookupCache: NewLookupCache(),
	}

	_, err := syncer.SyncContacts(context.Background(), configID, tenantID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "get config")
}

// TestSyncInbound_HappyPath_CreateNewContact tests that a Bexio contact that
// has no local mapping is created in KMU Hub and a new mapping is upserted.
func TestSyncInbound_HappyPath_CreateNewContact(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	email := "alice@example.com"
	bexioContact := BexioContact{
		ID:            77,
		ContactTypeID: BexioContactTypePerson,
		NameFirst:     "Alice",
		Mail:          &email,
	}
	rawContacts, _ := json.Marshal([]BexioContact{bexioContact})

	// httptest server: serves contacts list + lookup endpoints
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/contact":
			w.Header().Set("Content-Type", "application/json")
			w.Write(rawContacts)
		default:
			// Return empty array for lookup endpoints (country, currency, salutation)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer srv.Close()

	client := newTestClient(srv)
	seedToken(client, tenantID)

	newKmuhubID := uuid.New()
	contacts := &fullContactService{
		createForSyncFn: func(_ context.Context, _ *ContactSyncData, _ uuid.UUID) (uuid.UUID, error) {
			return newKmuhubID, nil
		},
	}
	repo := &fullSyncRepo{}

	syncer := &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncInbound(context.Background(), configID, tenantID, nil, DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsCreated)
	assert.Equal(t, 0, result.ItemsUpdated)
	assert.Equal(t, 0, result.ItemsFailed)
	assert.Equal(t, 1, contacts.createForSyncCalls)
	assert.Equal(t, 1, repo.upsertEntityMappingCalls, "new mapping must be upserted")
}

// TestSyncInbound_HappyPath_UpdateExistingContact tests that when a Bexio
// contact has a newer timestamp than the local mapping, UpdateForSync is called
// and the mapping timestamp is updated.
func TestSyncInbound_HappyPath_UpdateExistingContact(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	kmuhubID := uuid.New()

	email := "bob@example.com"
	bexioUpdatedAt := "2024-06-01T12:00:00"
	bexioContact := BexioContact{
		ID:            42,
		ContactTypeID: BexioContactTypePerson,
		NameFirst:     "Bob",
		Mail:          &email,
		UpdatedAt:     bexioUpdatedAt,
	}
	rawContacts, _ := json.Marshal([]BexioContact{bexioContact})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/contact":
			w.Write(rawContacts)
		default:
			w.Write([]byte("[]"))
		}
	}))
	defer srv.Close()

	client := newTestClient(srv)
	seedToken(client, tenantID)

	// Existing mapping with an older local timestamp
	oldTime := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	existingMapping := &models.BexioEntityMapping{
		ID:              uuid.New(),
		ConfigID:        configID,
		EntityType:      "contact",
		KmuhubID:        kmuhubID,
		BexioID:         42,
		KmuhubUpdatedAt: &oldTime,
	}

	contacts := &fullContactService{}
	repo := &fullSyncRepo{
		getEntityMappingByBexioFn: func(_ context.Context, _ uuid.UUID, _ string, id int) (*models.BexioEntityMapping, error) {
			if id == 42 {
				return existingMapping, nil
			}
			return nil, ErrMappingNotFound
		},
	}

	syncer := &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncInbound(context.Background(), configID, tenantID, nil, DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsUpdated)
	assert.Equal(t, 0, result.ItemsCreated)
	assert.Equal(t, 0, result.ItemsFailed)
	assert.Equal(t, 1, contacts.updateForSyncCalls)
	assert.Equal(t, 1, repo.upsertEntityMappingCalls, "updated mapping must be re-upserted")
}

// TestSyncInbound_ClientError tests that a Bexio API error is counted as a
// failure and propagated into SyncResult.Errors, but does not crash.
func TestSyncInbound_ClientError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate Bexio returning 500 for all paths so doWithRetry exhausts retries
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	seedToken(client, tenantID)

	syncer := &ContactSyncer{
		client:      client,
		repo:        &fullSyncRepo{},
		fieldMapper: NewFieldMapper(),
		contacts:    &fullContactService{},
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncInbound(context.Background(), configID, tenantID, nil, DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsFailed, "a client-level error must increment ItemsFailed")
	assert.NotEmpty(t, result.Errors)
}

// TestSyncInbound_ContactServiceCreateError verifies that when CreateForSync
// fails the failure counter is incremented and no mapping is upserted.
func TestSyncInbound_ContactServiceCreateError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	email := "carol@example.com"
	bexioContact := BexioContact{
		ID:            99,
		ContactTypeID: BexioContactTypePerson,
		NameFirst:     "Carol",
		Mail:          &email,
	}
	rawContacts, _ := json.Marshal([]BexioContact{bexioContact})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/contact":
			w.Write(rawContacts)
		default:
			w.Write([]byte("[]"))
		}
	}))
	defer srv.Close()

	client := newTestClient(srv)
	seedToken(client, tenantID)

	contacts := &fullContactService{
		createForSyncFn: func(_ context.Context, _ *ContactSyncData, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, errors.New("crm unavailable")
		},
	}
	repo := &fullSyncRepo{}

	syncer := &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncInbound(context.Background(), configID, tenantID, nil, DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsFailed)
	assert.Equal(t, 0, result.ItemsCreated)
	assert.Equal(t, 0, repo.upsertEntityMappingCalls, "no mapping must be created on CreateForSync error")
}

// TestSyncOutbound_HappyPath_CreateNewBexioContact verifies that a locally
// modified contact without a Bexio mapping is created in Bexio and a new
// mapping is persisted.
func TestSyncOutbound_HappyPath_CreateNewBexioContact(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	kmuhubID := uuid.New()

	createdBexioContact := BexioContact{ID: 500, NameFirst: "Dave"}
	rawCreated, _ := json.Marshal(createdBexioContact)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/contact" {
			w.Write(rawCreated)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	seedToken(client, tenantID)

	since := time.Now().Add(-time.Hour)
	contacts := &fullContactService{
		listModifiedSinceFn: func(_ context.Context, _ time.Time) ([]ContactResult, error) {
			return []ContactResult{
				{
					ID:        kmuhubID,
					FirstName: "Dave",
					LastName:  "Test",
					UpdatedAt: time.Now().UTC(),
				},
			}, nil
		},
	}
	repo := &fullSyncRepo{}

	syncer := &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncOutbound(context.Background(), configID, tenantID, since, DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsCreated)
	assert.Equal(t, 0, result.ItemsFailed)
	assert.Equal(t, 1, repo.upsertEntityMappingCalls)
}

// TestSyncOutbound_RepoError verifies that a ListModifiedSince error is
// counted as a failure.
func TestSyncOutbound_RepoError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	contacts := &fullContactService{
		listModifiedSinceFn: func(_ context.Context, _ time.Time) ([]ContactResult, error) {
			return nil, errors.New("crm db down")
		},
	}

	syncer := &ContactSyncer{
		client:      nil, // not reached
		repo:        &fullSyncRepo{},
		fieldMapper: NewFieldMapper(),
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	syncer.syncOutbound(context.Background(), configID, tenantID, time.Now().Add(-time.Hour), DefaultContactMappings(), result)

	assert.Equal(t, 1, result.ItemsFailed)
	assert.NotEmpty(t, result.Errors)
}
