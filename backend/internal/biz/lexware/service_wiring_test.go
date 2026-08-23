package lexware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// These tests cover the wiring between Service and the ContactSyncer /
// InvoicePusher / QuotePusher / WebhookHandler implementations.  Before this,
// all five public operations were placeholders that logged and returned nil, so
// every one of them reported success without ever reaching the Lexware API.
// Each test therefore asserts against a stub API server: the proof that the
// operation happened is a real HTTP request, not a return value of nil.

// --- Test doubles ---

type mockContactService struct {
	mu sync.Mutex

	getByEmailFn        func(ctx context.Context, email string) (*ContactResult, error)
	createdFromSync     []*ContactSyncData
	updatedFromSync     map[uuid.UUID]*ContactSyncData
	listModifiedResults []ContactResult
	nextCreatedID       uuid.UUID
}

func newMockContactService() *mockContactService {
	return &mockContactService{
		updatedFromSync: map[uuid.UUID]*ContactSyncData{},
		nextCreatedID:   uuid.New(),
	}
}

func (m *mockContactService) GetByID(context.Context, uuid.UUID) (*ContactResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockContactService) GetByEmail(ctx context.Context, email string) (*ContactResult, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockContactService) CreateForSync(_ context.Context, data *ContactSyncData, _ uuid.UUID) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createdFromSync = append(m.createdFromSync, data)
	return m.nextCreatedID, nil
}

func (m *mockContactService) UpdateForSync(_ context.Context, id uuid.UUID, data *ContactSyncData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedFromSync[id] = data
	return nil
}

func (m *mockContactService) ListModifiedSince(context.Context, time.Time) ([]ContactResult, error) {
	return m.listModifiedResults, nil
}

type mockInvoiceReader struct {
	invoice *models.Invoice
	err     error
}

func (m *mockInvoiceReader) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.Invoice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.invoice, nil
}

type mockQuoteReader struct {
	quote *models.Quote
	err   error
}

func (m *mockQuoteReader) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.Quote, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.quote, nil
}

// recordedRequest is one call the stub Lexware API received.
type recordedRequest struct {
	Method string
	Path   string
	Auth   string
	Body   []byte
}

// stubAPI is a fake Lexware Office API.  Handlers are keyed by
// "METHOD /path"; an unmapped route fails the test rather than silently
// returning 404, so a wiring change that hits the wrong endpoint is visible.
type stubAPI struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	routes   map[string]http.HandlerFunc
}

func newStubAPI(t *testing.T, routes map[string]http.HandlerFunc) *stubAPI {
	t.Helper()
	s := &stubAPI{t: t, routes: routes}
	s.server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.server.Close)
	return s
}

func (s *stubAPI) serve(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, 0)
	if r.Body != nil {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body = buf[:n]
	}

	s.mu.Lock()
	s.requests = append(s.requests, recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Auth:   r.Header.Get("Authorization"),
		Body:   body,
	})
	s.mu.Unlock()

	if h, ok := s.routes[r.Method+" "+r.URL.Path]; ok {
		h(w, r)
		return
	}

	s.t.Errorf("stub lexware API: unexpected request %s %s", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *stubAPI) recorded() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

func jsonRoute(status int, payload any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if payload != nil {
			_ = json.NewEncoder(w).Encode(payload)
		}
	}
}

// newWiredService builds a Service whose client points at the stub API.
func newWiredService(
	stub *stubAPI,
	repo *mockRepository,
	configRepo *mockConfigRepo,
	vault *mockVaultService,
	contacts ContactService,
	invoices InvoiceReader,
	quotes QuoteReader,
) *Service {
	client := NewClient(ClientConfig{BaseURL: stub.server.URL}, vault)
	return NewService(client, repo, configRepo, vault, contacts, invoices, quotes)
}

func activeConfigRepo(configID, tenantID uuid.UUID) *mockConfigRepo {
	return &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			return &IntegrationConfig{
				ID:                  configID,
				TenantID:            tenantID,
				Platform:            "lexware",
				IsActive:            true,
				CredentialsVaultKey: apiKeyVaultKey(tenantID),
			}, nil
		},
	}
}

func keyVault(apiKey string) *mockVaultService {
	return &mockVaultService{
		getSecretFn: func(context.Context, string) (string, error) { return apiKey, nil },
	}
}

// --- TestConnection ---

func TestTestConnection_MakesRealAPICall(t *testing.T) {
	tenantID := uuid.New()
	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"GET /v1/profile": jsonRoute(http.StatusOK, LexwareProfile{
			OrganizationID: "org-1", CompanyName: "Muster GmbH",
		}),
	})

	svc := newWiredService(stub, &mockRepository{}, activeConfigRepo(uuid.New(), tenantID),
		keyVault("valid-api-key"), nil, nil, nil)

	require.NoError(t, svc.TestConnection(context.Background()))

	reqs := stub.recorded()
	require.Len(t, reqs, 1, "TestConnection must actually call the Lexware API")
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "/v1/profile", reqs[0].Path)
	assert.Equal(t, "Bearer valid-api-key", reqs[0].Auth)
}

func TestTestConnection_RejectedKeyFails(t *testing.T) {
	tenantID := uuid.New()
	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"GET /v1/profile": jsonRoute(http.StatusUnauthorized, nil),
	})

	svc := newWiredService(stub, &mockRepository{}, activeConfigRepo(uuid.New(), tenantID),
		keyVault("revoked-key"), nil, nil, nil)

	err := svc.TestConnection(context.Background())

	// The old implementation returned nil here: a stored but revoked key
	// counted as "connected" and only broke on the next sync.
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLexwareUnauthorized)
	assert.Contains(t, err.Error(), "connection test failed")
}

// --- SyncContacts ---

func TestSyncContacts_DelegatesToSyncer(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	email := "kunde@example.com"

	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"GET /v1/contacts": jsonRoute(http.StatusOK, LexwareListResponse[LexwareContact]{
			Content: []LexwareContact{{
				ID:             "lx-contact-1",
				Version:        3,
				Person:         &LexwareContactPerson{FirstName: "Erika", LastName: "Musterfrau"},
				EmailAddresses: &LexwareEmails{Business: []string{email}},
				UpdatedDate:    "2026-07-01T10:00:00.000Z",
			}},
			Last: true,
		}),
	})

	contacts := newMockContactService()
	var storedMapping *models.LexwareEntityMapping
	repo := &mockRepository{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.LexwareSyncConfig, error) {
			return &models.LexwareSyncConfig{ConfigID: configID, ContactSyncEnabled: true}, nil
		},
		upsertEntityMappingFn: func(_ context.Context, m *models.LexwareEntityMapping) error {
			storedMapping = m
			return nil
		},
	}

	svc := newWiredService(stub, repo, activeConfigRepo(configID, tenantID),
		keyVault("k"), contacts, nil, nil)

	result, err := svc.SyncContacts(context.Background(), tenantID)
	require.NoError(t, err)

	// The placeholder returned an all-zero SyncResult without any API traffic.
	require.Equal(t, 1, result.ItemsProcessed)
	require.Equal(t, 1, result.ItemsCreated)
	require.Len(t, contacts.createdFromSync, 1)
	assert.Equal(t, "Erika", contacts.createdFromSync[0].FirstName)

	require.NotNil(t, storedMapping, "a new contact must be recorded in the entity mapping")
	assert.Equal(t, "lx-contact-1", storedMapping.LexwareID)
	assert.Equal(t, 3, storedMapping.LexwareVersion)
}

// TestSyncContactsWithConfig_SchedulerPathDoesNotResolveViaGetByPlatform proves
// the G8 fix: with more than one active Lexware tenant, GetByPlatform has no
// tenant filter and could return either tenant's config. The scheduler ticker
// now calls SyncContactsWithConfig with its own known configID and must never
// touch GetByPlatform at all — a call would panic the test.
func TestSyncContactsWithConfig_SchedulerPathDoesNotResolveViaGetByPlatform(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	email := "kunde@example.com"

	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"GET /v1/contacts": jsonRoute(http.StatusOK, LexwareListResponse[LexwareContact]{
			Content: []LexwareContact{{
				ID:             "lx-contact-a",
				Version:        1,
				Person:         &LexwareContactPerson{FirstName: "Erika", LastName: "Musterfrau"},
				EmailAddresses: &LexwareEmails{Business: []string{email}},
				UpdatedDate:    "2026-07-01T10:00:00.000Z",
			}},
			Last: true,
		}),
	})

	contacts := newMockContactService()
	var storedMapping *models.LexwareEntityMapping
	repo := &mockRepository{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.LexwareSyncConfig, error) {
			return &models.LexwareSyncConfig{ConfigID: configID, ContactSyncEnabled: true}, nil
		},
		upsertEntityMappingFn: func(_ context.Context, m *models.LexwareEntityMapping) error {
			storedMapping = m
			return nil
		},
	}
	cr := &mockConfigRepo{
		getByPlatformFn: func(context.Context, string) (*IntegrationConfig, error) {
			t.Fatal("SyncContactsWithConfig must not re-resolve the config via GetByPlatform under system context")
			return nil, nil
		},
	}

	svc := newWiredService(stub, repo, cr, keyVault("k"), contacts, nil, nil)

	result, err := svc.SyncContactsWithConfig(context.Background(), configID, tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemsCreated)
	require.NotNil(t, storedMapping)
	assert.Equal(t, "lx-contact-a", storedMapping.LexwareID)
}

// --- PushInvoice / PushQuote ---

func testInvoice(email string) *models.Invoice {
	return &models.Invoice{
		ID:            uuid.New(),
		InvoiceNumber: "RE-2026-0001",
		CustomerName:  "Muster GmbH",
		CustomerEmail: email,
		Currency:      "EUR",
		InvoiceDate:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		LineItems:     json.RawMessage(`[{"description":"Beratung","quantity":"2","unit_price":"100.00","tax_rate":"19"}]`),
		UpdatedAt:     time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestPushInvoice_DelegatesToPusher(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	localContactID := uuid.New()
	email := "kunde@example.com"

	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"POST /v1/invoices": jsonRoute(http.StatusCreated, LexwareInvoice{ID: "lx-invoice-9", Version: 1}),
	})

	contacts := newMockContactService()
	contacts.getByEmailFn = func(_ context.Context, got string) (*ContactResult, error) {
		assert.Equal(t, email, got)
		return &ContactResult{ID: localContactID}, nil
	}

	var invoiceMapping *models.LexwareEntityMapping
	repo := &mockRepository{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, id uuid.UUID) (*models.LexwareEntityMapping, error) {
			if entityType == "contact" {
				assert.Equal(t, localContactID, id)
				return &models.LexwareEntityMapping{EntityType: "contact", LexwareID: "lx-contact-42"}, nil
			}
			return nil, ErrMappingNotFound
		},
		upsertEntityMappingFn: func(_ context.Context, m *models.LexwareEntityMapping) error {
			if m.EntityType == "invoice" {
				invoiceMapping = m
			}
			return nil
		},
	}

	svc := newWiredService(stub, repo, activeConfigRepo(configID, tenantID),
		keyVault("k"), contacts, &mockInvoiceReader{invoice: testInvoice(email)}, nil)

	require.NoError(t, svc.PushInvoice(context.Background(), tenantID, uuid.New()))

	reqs := stub.recorded()
	require.Len(t, reqs, 1, "PushInvoice must POST the invoice to Lexware")
	assert.Equal(t, "POST", reqs[0].Method)
	assert.Equal(t, "/v1/invoices", reqs[0].Path)

	var sent LexwareInvoice
	require.NoError(t, json.Unmarshal(reqs[0].Body, &sent))
	assert.Equal(t, "lx-contact-42", sent.Address.ContactID,
		"the invoice must be addressed to the customer's own Lexware contact")
	require.Len(t, sent.LineItems, 1)

	require.NotNil(t, invoiceMapping)
	assert.Equal(t, "lx-invoice-9", invoiceMapping.LexwareID)
}

func TestPushInvoice_RefusesAmbiguousContact(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()

	// No route registered: reaching the API at all would fail the test.
	stub := newStubAPI(t, map[string]http.HandlerFunc{})

	repo := &mockRepository{
		listEntityMappingsFn: func(context.Context, uuid.UUID, string) ([]models.LexwareEntityMapping, error) {
			return []models.LexwareEntityMapping{
				{EntityType: "contact", LexwareID: "lx-contact-1"},
				{EntityType: "contact", LexwareID: "lx-contact-2"},
			}, nil
		},
	}

	// contacts nil: no exact lookup available.
	svc := newWiredService(stub, repo, activeConfigRepo(configID, tenantID),
		keyVault("k"), nil, &mockInvoiceReader{invoice: testInvoice("kunde@example.com")}, nil)

	err := svc.PushInvoice(context.Background(), tenantID, uuid.New())

	// The pre-wiring resolver took mappings[0] here — every invoice would have
	// been booked onto whichever contact the database happened to return first.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot determine which belongs to email")
	assert.Empty(t, stub.recorded(), "nothing may be pushed when the addressee is ambiguous")
}

func TestPushQuote_DelegatesToPusher(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	localContactID := uuid.New()
	email := "kunde@example.com"
	validUntil := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"POST /v1/quotations": jsonRoute(http.StatusCreated, LexwareQuotation{ID: "lx-quote-5", Version: 1}),
	})

	contacts := newMockContactService()
	contacts.getByEmailFn = func(context.Context, string) (*ContactResult, error) {
		return &ContactResult{ID: localContactID}, nil
	}

	repo := &mockRepository{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.LexwareEntityMapping, error) {
			if entityType == "contact" {
				return &models.LexwareEntityMapping{EntityType: "contact", LexwareID: "lx-contact-42"}, nil
			}
			return nil, ErrMappingNotFound
		},
	}

	quote := &models.Quote{
		ID:            uuid.New(),
		QuoteNumber:   "AN-2026-0001",
		CustomerName:  "Muster GmbH",
		CustomerEmail: email,
		Currency:      "EUR",
		CreatedAt:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil:    &validUntil,
		LineItems:     json.RawMessage(`[{"description":"Workshop","quantity":"1","unit_price":"500.00","tax_rate":"19"}]`),
		UpdatedAt:     time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	}

	svc := newWiredService(stub, repo, activeConfigRepo(configID, tenantID),
		keyVault("k"), contacts, nil, &mockQuoteReader{quote: quote})

	require.NoError(t, svc.PushQuote(context.Background(), tenantID, uuid.New()))

	reqs := stub.recorded()
	require.Len(t, reqs, 1, "PushQuote must POST the quotation to Lexware")
	assert.Equal(t, "/v1/quotations", reqs[0].Path)

	var sent LexwareQuotation
	require.NoError(t, json.Unmarshal(reqs[0].Body, &sent))
	assert.Equal(t, "lx-contact-42", sent.Address.ContactID)
}

// --- Webhook ---

func TestHandleWebhookEvent_ContactChanged_PullsAndAppliesContact(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	knownContactID := uuid.New()

	stub := newStubAPI(t, map[string]http.HandlerFunc{
		"GET /v1/contacts/lx-contact-7": jsonRoute(http.StatusOK, LexwareContact{
			ID:          "lx-contact-7",
			Version:     4,
			Person:      &LexwareContactPerson{FirstName: "Max", LastName: "Neu"},
			UpdatedDate: "2026-07-15T08:00:00.000Z",
		}),
	})

	contacts := newMockContactService()
	repo := &mockRepository{
		getEntityMappingByLexwareIDFn: func(context.Context, uuid.UUID, string, string) (*models.LexwareEntityMapping, error) {
			return &models.LexwareEntityMapping{
				EntityType: "contact",
				KmuhubID:   knownContactID,
				LexwareID:  "lx-contact-7",
			}, nil
		},
	}

	svc := newWiredService(stub, repo, activeConfigRepo(configID, tenantID),
		keyVault("k"), contacts, nil, nil)

	err := svc.HandleWebhookEvent(context.Background(), "contact.changed", map[string]any{
		"resource_id":     "lx-contact-7",
		"organization_id": "org-1",
	})
	require.NoError(t, err)

	reqs := stub.recorded()
	require.Len(t, reqs, 1, "a contact webhook must fetch the changed record")
	assert.Equal(t, "/v1/contacts/lx-contact-7", reqs[0].Path)

	updated, ok := contacts.updatedFromSync[knownContactID]
	require.True(t, ok, "the mapped CRM contact must be updated from the webhook payload")
	assert.Equal(t, "Max", updated.FirstName)
}

func TestHandleWebhookEvent_DocumentStatus_IsAcknowledgedOnly(t *testing.T) {
	configID, tenantID := uuid.New(), uuid.New()
	stub := newStubAPI(t, map[string]http.HandlerFunc{})

	svc := newWiredService(stub, &mockRepository{}, activeConfigRepo(configID, tenantID),
		keyVault("k"), newMockContactService(), nil, nil)

	err := svc.HandleWebhookEvent(context.Background(), "invoice.status.changed", map[string]any{
		"resource_id": "lx-invoice-1",
	})

	// Documented gap, not an oversight: no InvoiceStatusUpdater is wired into
	// the Lexware service, so there is nowhere to apply the new status.
	require.NoError(t, err)
	assert.Empty(t, stub.recorded())
}

// --- Contact resolution ---

func TestResolveContactLexwareIDByEmail_ExactLookup(t *testing.T) {
	configID := uuid.New()
	localID := uuid.New()

	contacts := newMockContactService()
	contacts.getByEmailFn = func(context.Context, string) (*ContactResult, error) {
		return &ContactResult{ID: localID}, nil
	}

	repo := &mockRepository{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, _ string, id uuid.UUID) (*models.LexwareEntityMapping, error) {
			assert.Equal(t, localID, id)
			return &models.LexwareEntityMapping{LexwareID: "lx-contact-99"}, nil
		},
		listEntityMappingsFn: func(context.Context, uuid.UUID, string) ([]models.LexwareEntityMapping, error) {
			t.Error("exact lookup must not fall back to listing all mappings")
			return nil, nil
		},
	}

	got, err := resolveContactLexwareIDByEmail(context.Background(), repo, contacts, configID, "kunde@example.com")
	require.NoError(t, err)
	assert.Equal(t, "lx-contact-99", got)
}

func TestResolveContactLexwareIDByEmail_EmptyEmail(t *testing.T) {
	_, err := resolveContactLexwareIDByEmail(context.Background(), &mockRepository{}, nil, uuid.New(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer email is empty")
}

func TestResolveContactLexwareIDByEmail_SingleMappingFallback(t *testing.T) {
	repo := &mockRepository{
		listEntityMappingsFn: func(context.Context, uuid.UUID, string) ([]models.LexwareEntityMapping, error) {
			return []models.LexwareEntityMapping{{LexwareID: "lx-contact-only"}}, nil
		},
	}

	got, err := resolveContactLexwareIDByEmail(context.Background(), repo, nil, uuid.New(), "kunde@example.com")
	require.NoError(t, err)
	assert.Equal(t, "lx-contact-only", got)
}
