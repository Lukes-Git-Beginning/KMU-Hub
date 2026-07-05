package bexio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockInvoiceImporter is a minimal InvoiceImporter with call capture so tests can
// assert loop-prevention (UpsertImported never called) and contact resolution.
type mockInvoiceImporter struct {
	upsertCalls int
	lastInput   models.ImportedInvoiceInput
	upsertFn    func(ctx context.Context, tenantID uuid.UUID, in models.ImportedInvoiceInput) (*models.Invoice, error)
	getByIDFn   func(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

func (m *mockInvoiceImporter) UpsertImported(ctx context.Context, tenantID uuid.UUID, in models.ImportedInvoiceInput) (*models.Invoice, error) {
	m.upsertCalls++
	m.lastInput = in
	if m.upsertFn != nil {
		return m.upsertFn(ctx, tenantID, in)
	}
	return &models.Invoice{ID: uuid.New(), TenantID: tenantID, Source: models.InvoiceSourceBexio}, nil
}

func (m *mockInvoiceImporter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &models.Invoice{ID: id, TenantID: tenantID, Source: models.InvoiceSourceBexio}, nil
}

// pullRepo wraps fullSyncRepo to capture UpdateLastSyncTime (cursor) calls.
type pullRepo struct {
	*fullSyncRepo
	lastSyncType  string
	lastSyncCalls int
}

func (r *pullRepo) UpdateLastSyncTime(_ context.Context, _ uuid.UUID, syncType string, _ time.Time) error {
	r.lastSyncCalls++
	r.lastSyncType = syncType
	return nil
}

// invoiceListServer serves the given Bexio invoices as a kb_invoice JSON array.
func invoiceListServer(t *testing.T, invoices []BexioInvoice) *httptest.Server {
	t.Helper()
	raw, err := json.Marshal(invoices)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
}

func buildInvoicePuller(srv *httptest.Server, tenantID uuid.UUID, repo Repository, importer InvoiceImporter) *InvoicePuller {
	client := newTestClient(srv)
	seedToken(client, tenantID)
	return &InvoicePuller{
		client:      client,
		repo:        repo,
		fieldMapper: NewFieldMapper(),
		importer:    importer,
		lookupCache: NewLookupCache(),
	}
}

// pastCursor is a non-nil cursor so the delta-forward seed guard is bypassed.
func pastCursor() *time.Time {
	c := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return &c
}

func sampleBexioInvoice(id int, updatedAt string) BexioInvoice {
	nr := "B-2024-1"
	return BexioInvoice{
		ID:             id,
		DocumentNr:     &nr,
		ContactID:      0,
		KBItemStatusID: BexioStatusSent,
		IsValidFrom:    "2024-06-01",
		Total:          strPtr("100.00"),
		UpdatedAt:      updatedAt,
	}
}

func strPtr(s string) *string { return &s }

// TestPullInvoices_DeltaForwardSeed verifies the first pull (nil cursor) imports no
// history and only seeds the cursor.
func TestPullInvoices_DeltaForwardSeed(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()

	repo := &pullRepo{fullSyncRepo: &fullSyncRepo{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return &models.BexioSyncConfig{InvoicePullEnabled: true, LastInvoicePullAt: nil}, nil
		},
	}}
	importer := &mockInvoiceImporter{}

	// Server must not be hit; a nil-cursor pull returns before ListInvoices.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("ListInvoices must not be called on delta-forward seed")
	}))
	defer srv.Close()

	puller := buildInvoicePuller(srv, tenantID, repo, importer)
	result, err := puller.PullInvoices(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ItemsProcessed)
	assert.Equal(t, 0, importer.upsertCalls)
	assert.Equal(t, 1, repo.lastSyncCalls)
	assert.Equal(t, "invoice_pull", repo.lastSyncType)
}

// TestPullInvoices_NewImport verifies an unmapped Bexio invoice is imported and a new
// inbound mapping is created.
func TestPullInvoices_NewImport(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	importedID := uuid.New()

	var captured *models.BexioEntityMapping
	repo := &pullRepo{fullSyncRepo: &fullSyncRepo{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return &models.BexioSyncConfig{InvoicePullEnabled: true, LastInvoicePullAt: pastCursor()}, nil
		},
		upsertEntityMappingFn: func(_ context.Context, m *models.BexioEntityMapping) error {
			captured = m
			return nil
		},
	}}
	importer := &mockInvoiceImporter{
		upsertFn: func(_ context.Context, tenantID uuid.UUID, _ models.ImportedInvoiceInput) (*models.Invoice, error) {
			return &models.Invoice{ID: importedID, TenantID: tenantID, Source: models.InvoiceSourceBexio}, nil
		},
	}

	srv := invoiceListServer(t, []BexioInvoice{sampleBexioInvoice(100, "2024-06-10 12:00:00")})
	defer srv.Close()

	puller := buildInvoicePuller(srv, tenantID, repo, importer)
	result, err := puller.PullInvoices(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsCreated)
	assert.Equal(t, 0, result.ItemsFailed)
	assert.Equal(t, 1, importer.upsertCalls)
	require.NotNil(t, captured)
	assert.Equal(t, "invoice", captured.EntityType)
	assert.Equal(t, "inbound", captured.SyncDirection)
	assert.Equal(t, importedID, captured.KmuhubID)
	assert.Equal(t, 100, captured.BexioID)
	assert.Equal(t, "invoice_pull", repo.lastSyncType)
}

// TestPullInvoices_LoopPreventionSkipsCosmiOrigin verifies a Bexio invoice whose Cosmi
// mapping is source='cosmi' (pushed from Cosmi) is skipped, never re-imported.
func TestPullInvoices_LoopPreventionSkipsCosmiOrigin(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	cosmiInvID := uuid.New()

	repo := &pullRepo{fullSyncRepo: &fullSyncRepo{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return &models.BexioSyncConfig{InvoicePullEnabled: true, LastInvoicePullAt: pastCursor()}, nil
		},
		getEntityMappingByBexioFn: func(_ context.Context, cfg uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error) {
			if entityType == "invoice" && bexioID == 100 {
				return &models.BexioEntityMapping{ConfigID: cfg, EntityType: "invoice", KmuhubID: cosmiInvID, BexioID: 100, SyncDirection: "outbound"}, nil
			}
			return nil, ErrMappingNotFound
		},
	}}
	importer := &mockInvoiceImporter{
		getByIDFn: func(_ context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
			assert.Equal(t, cosmiInvID, id)
			return &models.Invoice{ID: id, TenantID: tenantID, Source: models.InvoiceSourceCosmi}, nil
		},
	}

	srv := invoiceListServer(t, []BexioInvoice{sampleBexioInvoice(100, "2024-06-10 12:00:00")})
	defer srv.Close()

	puller := buildInvoicePuller(srv, tenantID, repo, importer)
	result, err := puller.PullInvoices(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 0, result.ItemsCreated)
	assert.Equal(t, 0, result.ItemsUpdated)
	assert.Equal(t, 0, importer.upsertCalls, "must not re-import a source='cosmi' invoice")
}

// TestPullInvoices_LWWSkipsStaleBexio verifies an already-mirrored invoice whose Bexio
// version is not newer than the recorded one is skipped.
func TestPullInvoices_LWWSkipsStaleBexio(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	mirrorID := uuid.New()
	recorded := time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)

	repo := &pullRepo{fullSyncRepo: &fullSyncRepo{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return &models.BexioSyncConfig{InvoicePullEnabled: true, LastInvoicePullAt: pastCursor()}, nil
		},
		getEntityMappingByBexioFn: func(_ context.Context, cfg uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error) {
			if entityType == "invoice" {
				return &models.BexioEntityMapping{ConfigID: cfg, EntityType: "invoice", KmuhubID: mirrorID, BexioID: 100, SyncDirection: "inbound", BexioUpdatedAt: &recorded}, nil
			}
			return nil, ErrMappingNotFound
		},
	}}
	importer := &mockInvoiceImporter{
		getByIDFn: func(_ context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
			return &models.Invoice{ID: id, TenantID: tenantID, Source: models.InvoiceSourceBexio}, nil
		},
	}

	// Bexio updated_at (2024-06-01) is older than the recorded mirror (2024-06-05).
	srv := invoiceListServer(t, []BexioInvoice{sampleBexioInvoice(100, "2024-06-01 00:00:00")})
	defer srv.Close()

	puller := buildInvoicePuller(srv, tenantID, repo, importer)
	result, err := puller.PullInvoices(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Equal(t, 0, importer.upsertCalls, "stale Bexio version must be skipped by LWW")
}

// TestPullInvoices_ResolvesContact verifies the Bexio contact_id is resolved to the
// Cosmi contact UUID via the existing contact entity-mapping.
func TestPullInvoices_ResolvesContact(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	contactUUID := uuid.New()

	repo := &pullRepo{fullSyncRepo: &fullSyncRepo{
		getSyncConfigFn: func(context.Context, uuid.UUID) (*models.BexioSyncConfig, error) {
			return &models.BexioSyncConfig{InvoicePullEnabled: true, LastInvoicePullAt: pastCursor()}, nil
		},
		getEntityMappingByBexioFn: func(_ context.Context, cfg uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error) {
			if entityType == "contact" && bexioID == 42 {
				return &models.BexioEntityMapping{ConfigID: cfg, EntityType: "contact", KmuhubID: contactUUID, BexioID: 42}, nil
			}
			return nil, ErrMappingNotFound
		},
	}}
	importer := &mockInvoiceImporter{}

	inv := sampleBexioInvoice(100, "2024-06-10 12:00:00")
	inv.ContactID = 42
	srv := invoiceListServer(t, []BexioInvoice{inv})
	defer srv.Close()

	puller := buildInvoicePuller(srv, tenantID, repo, importer)
	_, err := puller.PullInvoices(context.Background(), configID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, importer.upsertCalls)
	assert.Equal(t, contactUUID, importer.lastInput.ContactID)
}
