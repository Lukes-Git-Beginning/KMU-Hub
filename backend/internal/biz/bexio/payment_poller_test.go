package bexio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockInvoiceStatusUpdater is a minimal InvoiceStatusUpdater mock with a
// call counter for MarkPaid so idempotency can be verified.
type mockInvoiceStatusUpdater struct {
	markPaidCalls int
	markPaidFn    func(ctx context.Context, tenantID, id uuid.UUID) error
	getByIDFn     func(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

func (m *mockInvoiceStatusUpdater) MarkPaid(ctx context.Context, tenantID, id uuid.UUID) error {
	m.markPaidCalls++
	if m.markPaidFn != nil {
		return m.markPaidFn(ctx, tenantID, id)
	}
	return nil
}

func (m *mockInvoiceStatusUpdater) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, errors.New("not implemented")
}

// openInvoice returns a minimal Invoice with status="open".
func openInvoice(id, tenantID uuid.UUID, gross decimal.Decimal) *models.Invoice {
	return &models.Invoice{
		ID:         id,
		TenantID:   tenantID,
		Status:     "open",
		GrossTotal: gross,
		UpdatedAt:  time.Now().UTC(),
	}
}

// buildPaymentPoller creates a PaymentPoller wired to the given httptest.Server.
func buildPaymentPoller(
	srv *httptest.Server,
	tenantID uuid.UUID,
	repo Repository,
	invoices InvoiceStatusUpdater,
) *PaymentPoller {
	client := newTestClient(srv)
	seedToken(client, tenantID)
	return &PaymentPoller{
		client:   client,
		repo:     repo,
		invoices: invoices,
	}
}

// TestPollPayments_HappyPath_MarkPaid verifies that when the total Bexio
// payment amount equals the invoice gross total the invoice is marked paid.
func TestPollPayments_HappyPath_MarkPaid(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 50
	gross := decimal.NewFromFloat(100.00)

	payments := []BexioPaymentStatus{
		{ID: 1, Value: "100.00", Date: "2024-06-15"},
	}
	rawPayments, _ := json.Marshal(payments)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/payment") {
			w.Write(rawPayments)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	inv := openInvoice(invoiceID, tenantID, gross)
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, id uuid.UUID) (*models.Invoice, error) {
			assert.Equal(t, invoiceID, id)
			return inv, nil
		},
	}

	mapping := models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
			if entityType == "invoice" {
				return []models.BexioEntityMapping{mapping}, nil
			}
			return nil, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	result, err := poller.PollPayments(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsUpdated)
	assert.Equal(t, 0, result.ItemsFailed)
	assert.Equal(t, 1, invoices.markPaidCalls)
}

// TestPollPayments_Idempotent_AlreadyPaid verifies that polling an invoice that
// is already paid in KMU Hub skips the Bexio API call and MarkPaid is never
// called a second time.
func TestPollPayments_Idempotent_AlreadyPaid(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 51

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/payment") {
			// Should not be reached for an already-paid invoice.
			t.Errorf("unexpected payment API call for already-paid invoice: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	paidInvoice := &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     "paid", // already settled
		GrossTotal: decimal.NewFromFloat(100.00),
	}
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return paidInvoice, nil
		},
	}

	mapping := models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{mapping}, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	// Run twice
	result, err := poller.PollPayments(context.Background(), configID, tenantID)
	require.NoError(t, err)
	result2, err := poller.PollPayments(context.Background(), configID, tenantID)
	require.NoError(t, err)

	assert.Equal(t, 0, result.ItemsProcessed, "paid invoice must be skipped")
	assert.Equal(t, 0, result2.ItemsProcessed)
	assert.Equal(t, 0, invoices.markPaidCalls, "MarkPaid must never be called for an already-paid invoice")
}

// TestPollPayments_PartialPayment verifies that an invoice with partial payment
// (less than gross) is NOT marked as paid.
func TestPollPayments_PartialPayment(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 52

	payments := []BexioPaymentStatus{
		{ID: 1, Value: "50.00", Date: "2024-06-15"}, // only half paid
	}
	rawPayments, _ := json.Marshal(payments)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/payment") {
			w.Write(rawPayments)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	inv := openInvoice(invoiceID, tenantID, decimal.NewFromFloat(100.00))
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}

	mapping := models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{mapping}, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	result, err := poller.PollPayments(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 0, result.ItemsUpdated, "partial payment must not mark invoice as paid")
	assert.Equal(t, 0, invoices.markPaidCalls)
}

// TestPollPayments_ClientError verifies that a Bexio API error on
// ListInvoicePayments is counted as a failure per mapping.
func TestPollPayments_ClientError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 53

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	inv := openInvoice(invoiceID, tenantID, decimal.NewFromFloat(100.00))
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}

	mapping := models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{mapping}, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	result, err := poller.PollPayments(context.Background(), configID, tenantID)

	require.NoError(t, err, "PollPayments itself must not return error on individual item failure")
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsFailed)
	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, 0, invoices.markPaidCalls)
}

// TestPollPayments_ListMappingsError verifies that a repository error when
// listing entity mappings is returned as a hard error.
func TestPollPayments_ListMappingsError(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return nil, errors.New("db connection lost")
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, &mockInvoiceStatusUpdater{})

	_, err := poller.PollPayments(context.Background(), configID, tenantID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list mappings")
}

// TestPollPayments_Idempotent_NoDoubleMark verifies that even if polling is
// called twice for the same set of mappings, MarkPaid is called exactly once
// (because after the first call the invoice becomes "paid" and is skipped
// on the second run).
func TestPollPayments_Idempotent_NoDoubleMark(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 54

	payments := []BexioPaymentStatus{
		{ID: 1, Value: "200.00", Date: "2024-06-15"},
	}
	rawPayments, _ := json.Marshal(payments)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/payment") {
			w.Write(rawPayments)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	// The status transitions from "open" → "paid" after the first MarkPaid call.
	status := "open"
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return &models.Invoice{
				ID:         invoiceID,
				TenantID:   tenantID,
				Status:     status,
				GrossTotal: decimal.NewFromFloat(200.00),
			}, nil
		},
		markPaidFn: func(_ context.Context, _, _ uuid.UUID) error {
			status = "paid"
			return nil
		},
	}

	mapping := models.BexioEntityMapping{
		ID:         uuid.New(),
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{mapping}, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	_, err := poller.PollPayments(context.Background(), configID, tenantID)
	require.NoError(t, err)

	_, err = poller.PollPayments(context.Background(), configID, tenantID)
	require.NoError(t, err)

	assert.Equal(t, 1, invoices.markPaidCalls,
		"MarkPaid must be called exactly once even when polling runs twice")
}

// TestPollPayments_MultiplePayments_Summed verifies that multiple partial
// payments are summed correctly and MarkPaid is called when the total reaches
// the gross amount.
func TestPollPayments_MultiplePayments_Summed(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()
	bexioInvoiceID := 55

	payments := []BexioPaymentStatus{
		{ID: 1, Value: "40.00"},
		{ID: 2, Value: "35.00"},
		{ID: 3, Value: "25.00"}, // total = 100.00
	}
	rawPayments, _ := json.Marshal(payments)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, fmt.Sprintf("kb_invoice/%d/payment", bexioInvoiceID)) {
			w.Write(rawPayments)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	inv := openInvoice(invoiceID, tenantID, decimal.NewFromFloat(100.00))
	invoices := &mockInvoiceStatusUpdater{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Invoice, error) {
			return inv, nil
		},
	}

	mapping := models.BexioEntityMapping{
		ConfigID:   configID,
		EntityType: "invoice",
		KmuhubID:   invoiceID,
		BexioID:    bexioInvoiceID,
	}

	repo := &fullSyncRepo{
		listEntityMappingsFn: func(_ context.Context, _ uuid.UUID, _ string) ([]models.BexioEntityMapping, error) {
			return []models.BexioEntityMapping{mapping}, nil
		},
	}

	poller := buildPaymentPoller(srv, tenantID, repo, invoices)

	result, err := poller.PollPayments(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsUpdated)
	assert.Equal(t, 1, invoices.markPaidCalls)
}
