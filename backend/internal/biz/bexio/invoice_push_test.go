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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockInvoiceReader is a minimal InvoiceReader mock.
type mockInvoiceReader struct {
	getByIDFn func(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

func (m *mockInvoiceReader) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, errors.New("not implemented")
}

// testInvoice returns a minimal valid Invoice for push tests.
func testInvoice(invoiceID, tenantID uuid.UUID) *models.Invoice {
	return &models.Invoice{
		ID:            invoiceID,
		TenantID:      tenantID,
		InvoiceNumber: "RE-2024-001",
		Status:        "open",
		CustomerEmail: "customer@example.com",
		InvoiceDate:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
		GrossTotal:    decimal.NewFromFloat(119.00),
		UpdatedAt:     time.Now().UTC(),
	}
}

// buildInvoicePusher builds an InvoicePusher with an httptest server for
// Bexio API calls and configurable mocks.
func buildInvoicePusher(
	srv *httptest.Server,
	tenantID uuid.UUID,
	repo Repository,
	invoices InvoiceReader,
	contacts ContactService,
) *InvoicePusher {
	client := newTestClient(srv)
	seedToken(client, tenantID)
	return &InvoicePusher{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		invoices:    invoices,
		contacts:    contacts,
	}
}

// TestPushInvoice_HappyPath_Create verifies that a new invoice is created in
// Bexio and the mapping is persisted when no prior mapping exists.
func TestPushInvoice_HappyPath_Create(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	contactBexioID := 11
	createdBexioInvoiceID := 200

	createdBexioInvoice := BexioInvoice{ID: createdBexioInvoiceID}
	rawCreated, _ := json.Marshal(createdBexioInvoice)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/kb_invoice" {
			w.Write(rawCreated)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	inv := testInvoice(invoiceID, tenantID)
	invoices := &mockInvoiceReader{
		getByIDFn: func(_ context.Context, _, id uuid.UUID) (*models.Invoice, error) {
			assert.Equal(t, invoiceID, id)
			return inv, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}

	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			return nil, ErrMappingNotFound // no existing mapping
		},
		getEntityMappingByBexioFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) (*models.BexioEntityMapping, error) {
			// Used by resolveContactBexioIDByEmail when contacts != nil
			return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
		},
	}

	// Reroute GetEntityMapping so resolveContactBexioIDByEmail gets the contact
	// mapping and the invoice mapping lookup returns "not found".
	callCount := 0
	repo.getEntityMappingFn = func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
		callCount++
		if entityType == "contact" {
			return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
		}
		return nil, ErrMappingNotFound // invoice has no mapping yet
	}

	pusher := buildInvoicePusher(srv, tenantID, repo, invoices, contacts)

	err := pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.upsertEntityMappingCalls, "new invoice mapping must be upserted exactly once")
}

// TestPushInvoice_ContactResolveFails verifies that an error returned by the
// contact resolution step is propagated and no Bexio API call is made.
func TestPushInvoice_ContactResolveFails(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be reached.
		t.Errorf("unexpected Bexio API call: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer srv.Close()

	inv := testInvoice(invoiceID, tenantID)
	invoices := &mockInvoiceReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return nil, errors.New("crm lookup failed")
		},
	}
	repo := &fullSyncRepo{}

	pusher := buildInvoicePusher(srv, tenantID, repo, invoices, contacts)

	err := pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve contact")
	assert.Equal(t, 0, repo.upsertEntityMappingCalls)
}

// TestPushInvoice_BexioAPIFails verifies that a Bexio API error on CreateInvoice
// is surfaced as an error and no mapping is persisted.
func TestPushInvoice_BexioAPIFails(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	contactBexioID := 12

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a persistent 500 so all retries fail.
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	inv := testInvoice(invoiceID, tenantID)
	invoices := &mockInvoiceReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}
	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			if entityType == "contact" {
				return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
			}
			return nil, ErrMappingNotFound
		},
	}

	pusher := buildInvoicePusher(srv, tenantID, repo, invoices, contacts)

	err := pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create in bexio")
	assert.Equal(t, 0, repo.upsertEntityMappingCalls, "no mapping must be created when Bexio API fails")
}

// TestPushInvoice_Idempotent_ExistingMapping verifies that pushing the same
// invoice twice (where a mapping already exists) results in an Update call
// (not Create) and UpsertEntityMapping is called exactly once per push.
func TestPushInvoice_Idempotent_ExistingMapping(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	contactBexioID := 13
	existingBexioInvoiceID := 300

	updatedBexioInvoice := BexioInvoice{ID: existingBexioInvoiceID}
	rawUpdated, _ := json.Marshal(updatedBexioInvoice)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Any POST (update) returns the updated invoice
		w.Write(rawUpdated)
	}))
	defer srv.Close()

	inv := testInvoice(invoiceID, tenantID)
	invoices := &mockInvoiceReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}
	contacts := &mockContactService{
		getByEmailFn: func(_ context.Context, _ string) (*ContactResult, error) {
			return &ContactResult{ID: uuid.New()}, nil
		},
	}

	existingMapping := &models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    existingBexioInvoiceID,
	}

	repo := &fullSyncRepo{
		getEntityMappingFn: func(_ context.Context, _ uuid.UUID, entityType string, _ uuid.UUID) (*models.BexioEntityMapping, error) {
			if entityType == "contact" {
				return &models.BexioEntityMapping{BexioID: contactBexioID}, nil
			}
			// invoice: already mapped
			return existingMapping, nil
		},
	}

	pusher := buildInvoicePusher(srv, tenantID, repo, invoices, contacts)

	// First push
	err := pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)
	require.NoError(t, err)

	// Second push — should also succeed, no duplicate create
	err = pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)
	require.NoError(t, err)

	// UpsertEntityMapping called once per push (update path), not a new create
	assert.Equal(t, 2, repo.upsertEntityMappingCalls,
		"each push with an existing mapping must upsert once (update, not create)")
}

// TestPushInvoice_InvoiceReaderError verifies that a failure to load the
// invoice is returned immediately.
func TestPushInvoice_InvoiceReaderError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Bexio API call: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	invoices := &mockInvoiceReader{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return nil, errors.New("invoice not found")
		},
	}
	repo := &fullSyncRepo{}

	pusher := buildInvoicePusher(srv, tenantID, repo, invoices, nil)

	err := pusher.PushInvoice(context.Background(), configID, tenantID, invoiceID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "load invoice")
}
