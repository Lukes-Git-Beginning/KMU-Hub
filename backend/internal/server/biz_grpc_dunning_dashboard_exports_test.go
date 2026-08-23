package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/dashboard"
	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Test doubles — dunning repository/config/invoice-reader, dashboard
// repository, DATEV pagers
// ============================================================================

// stubDunningRecordRepo is a configurable dunning.Repository double covering
// the record-level RPCs (List/Create/Escalate/UpdateStatus/SendNotice), unlike
// the minimal read-only stubDunningRepo in biz_grpc_dunning_test.go which only
// backs SendDunning.
type stubDunningRecordRepo struct {
	records    map[uuid.UUID]*models.DunningRecord
	byInvoice  map[uuid.UUID][]*models.DunningRecord
	listResult []*models.DunningRecord
	listTotal  int
	listErr    error
	createErr  error
	getErr     error
	updateErr  error
}

func newStubDunningRecordRepo() *stubDunningRecordRepo {
	return &stubDunningRecordRepo{
		records:   make(map[uuid.UUID]*models.DunningRecord),
		byInvoice: make(map[uuid.UUID][]*models.DunningRecord),
	}
}

func (r *stubDunningRecordRepo) Create(_ context.Context, record *models.DunningRecord) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.records[record.ID] = record
	r.byInvoice[record.InvoiceID] = append(r.byInvoice[record.InvoiceID], record)
	return nil
}

func (r *stubDunningRecordRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.DunningRecord, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	rec, ok := r.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, dunning.ErrDunningNotFound
	}
	return rec, nil
}

func (r *stubDunningRecordRepo) List(_ context.Context, _ uuid.UUID, _ dunning.ListFilter) ([]*models.DunningRecord, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	return r.listResult, r.listTotal, nil
}

func (r *stubDunningRecordRepo) UpdateStatus(_ context.Context, _, id uuid.UUID, status string, sentAt *time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if rec, ok := r.records[id]; ok {
		rec.Status = status
		rec.SentAt = sentAt
	}
	return nil
}

func (r *stubDunningRecordRepo) GetByInvoiceID(_ context.Context, _, invoiceID uuid.UUID) ([]*models.DunningRecord, error) {
	return r.byInvoice[invoiceID], nil
}

func (r *stubDunningRecordRepo) GetByInvoiceIDs(_ context.Context, _ uuid.UUID, invoiceIDs []uuid.UUID) (map[uuid.UUID][]*models.DunningRecord, error) {
	result := make(map[uuid.UUID][]*models.DunningRecord)
	for _, id := range invoiceIDs {
		if recs, ok := r.byInvoice[id]; ok {
			result[id] = recs
		}
	}
	return result, nil
}

func (r *stubDunningRecordRepo) GetHighestLevelByInvoiceID(context.Context, uuid.UUID, uuid.UUID) (*models.DunningRecord, error) {
	return nil, nil
}

// stubDunningConfigRepo is a configurable dunning.ConfigRepository double.
type stubDunningConfigRepo struct {
	config    *models.DunningConfig
	getErr    error
	upsertErr error
}

func (r *stubDunningConfigRepo) Get(_ context.Context, _ uuid.UUID) (*models.DunningConfig, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.config, nil
}

func (r *stubDunningConfigRepo) Upsert(_ context.Context, config *models.DunningConfig) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.config = config
	return nil
}

// stubDunningInvoiceReader is a configurable dunning.InvoiceReader double.
// Only GetOverdue/GetByID are exercised by the RPCs under test; ListOpenItems
// and SummarizeOpenItems back the (separately tested) Offene-Posten RPCs.
type stubDunningInvoiceReader struct {
	overdue []*models.Invoice
	byID    map[uuid.UUID]*models.Invoice
}

func (r *stubDunningInvoiceReader) GetByID(_ context.Context, _, id uuid.UUID) (*models.Invoice, error) {
	if inv, ok := r.byID[id]; ok {
		return inv, nil
	}
	return nil, errors.New("invoice not found")
}

func (r *stubDunningInvoiceReader) GetOverdue(context.Context, uuid.UUID) ([]*models.Invoice, error) {
	return r.overdue, nil
}

func (r *stubDunningInvoiceReader) ListOpenItems(context.Context, uuid.UUID, models.OpenItemFilter) ([]*models.OpenItem, int, error) {
	return nil, 0, nil
}

func (r *stubDunningInvoiceReader) SummarizeOpenItems(context.Context, uuid.UUID, time.Time) ([]*models.OpenItemBucketTotal, error) {
	return nil, nil
}

func defaultDunningConfig(tenantID uuid.UUID) *models.DunningConfig {
	return &models.DunningConfig{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		Level1DaysAfterDue:    14,
		Level2DaysAfterLevel1: 14,
		Level3DaysAfterLevel2: 14,
		Level1Fee:             decimal.Zero,
		Level2Fee:             decimal.NewFromInt(5),
		Level3Fee:             decimal.NewFromInt(10),
		UpdatedAt:             time.Now(),
	}
}

func newDunningTestServer(repo dunning.Repository, configRepo dunning.ConfigRepository, invReader dunning.InvoiceReader) *BizGRPCServer {
	return &BizGRPCServer{dunningService: dunning.NewService(repo, configRepo, invReader)}
}

// stubDashboardRepo is a configurable dashboard.Repository double.
type stubDashboardRepo struct {
	metrics *dashboard.Metrics
	err     error
}

func (r *stubDashboardRepo) GetDashboardMetrics(context.Context, uuid.UUID, time.Time, time.Time) (*dashboard.Metrics, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.metrics, nil
}

func newDashboardTestServer(repo dashboard.Repository) *BizGRPCServer {
	return &BizGRPCServer{dashboardService: dashboard.NewService(repo)}
}

// stubInvoicePager/stubCreditNotePager back the DATEV Buchungsstapel builder
// used by ExportDATEV. Each holds a fixed list of keyset pages; a real repo
// would page by (date, id) but a single page is enough to exercise the
// builder without pulling in the full invoice/creditnote services.
type stubInvoicePager struct {
	pages [][]*models.Invoice
	err   error
	calls int
}

func (p *stubInvoicePager) ListForDATEVExport(context.Context, uuid.UUID, time.Time, time.Time, *time.Time, *uuid.UUID, int) ([]*models.Invoice, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.calls >= len(p.pages) {
		return nil, nil
	}
	page := p.pages[p.calls]
	p.calls++
	return page, nil
}

type stubCreditNotePager struct {
	pages [][]*models.CreditNote
	err   error
	calls int
}

func (p *stubCreditNotePager) ListForDATEVExport(context.Context, uuid.UUID, time.Time, time.Time, *time.Time, *uuid.UUID, int) ([]*models.CreditNote, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.calls >= len(p.pages) {
		return nil, nil
	}
	page := p.pages[p.calls]
	p.calls++
	return page, nil
}

func newDatevExportTestServer(invPager datev.InvoicePager, cnPager datev.CreditNotePager, settings datev.CompanySettingsReader) *BizGRPCServer {
	builder := datev.NewBuchungsstapelBuilder(datev.NewExporter(), invPager, cnPager, settings)
	return &BizGRPCServer{datevBuchungsstapel: builder}
}

// ============================================================================
// ListDunnings
// ============================================================================

func TestListDunnings(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.ListDunnings(context.Background(), &bizv1.ListDunningsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id filter", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.ListDunnings(context.Background(), &bizv1.ListDunningsRequest{
			TenantId: tenantID.String(), InvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns records and total", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		repo.listResult = []*models.DunningRecord{{
			ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(),
			Level: 1, Status: models.DunningStatusDraft,
			Fee: decimal.Zero, Interest: decimal.Zero, CreatedBy: uuid.New(), CreatedAt: time.Now(),
		}}
		repo.listTotal = 1
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		resp, err := srv.ListDunnings(context.Background(), &bizv1.ListDunningsRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.GetDunnings(), 1)
		assert.Equal(t, int32(1), resp.GetTotal())
	})

	t.Run("repo error maps to internal", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		repo.listErr = errors.New("boom")
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.ListDunnings(context.Background(), &bizv1.ListDunningsRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// CreateDunning
// ============================================================================

func TestCreateDunning(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.CreateDunning(context.Background(), &bizv1.CreateDunningRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.CreateDunning(context.Background(), &bizv1.CreateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no overdue invoices found", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.CreateDunning(context.Background(), &bizv1.CreateDunningRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("no dunning created for specified invoice", func(t *testing.T) {
		otherInvoiceID := uuid.New()
		requestedInvoiceID := uuid.New()
		invReader := &stubDunningInvoiceReader{overdue: []*models.Invoice{{
			ID: otherInvoiceID, TenantID: tenantID, DueDate: time.Now().Add(-30 * 24 * time.Hour),
		}}}
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, invReader)
		_, err := srv.CreateDunning(context.Background(), &bizv1.CreateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: requestedInvoiceID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path creates level 1 draft for overdue invoice", func(t *testing.T) {
		invoiceID := uuid.New()
		invReader := &stubDunningInvoiceReader{overdue: []*models.Invoice{{
			ID: invoiceID, TenantID: tenantID, DueDate: time.Now().Add(-30 * 24 * time.Hour),
		}}}
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, invReader)
		resp, err := srv.CreateDunning(context.Background(), &bizv1.CreateDunningRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		require.NotNil(t, resp.GetDunning())
		assert.Equal(t, invoiceID.String(), resp.GetDunning().GetInvoiceId())
		assert.Equal(t, int32(1), resp.GetDunning().GetLevel())
	})
}

// ============================================================================
// EscalateDunning
// ============================================================================

func TestEscalateDunning(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.EscalateDunning(context.Background(), &bizv1.EscalateDunningRequest{
			TenantId: "not-a-uuid", InvoiceId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.EscalateDunning(context.Background(), &bizv1.EscalateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// Guards the escalation ceiling: once the highest sent level already equals
	// MaxDunningLevel (3), DetectAndCreateDunnings must not create a level-4
	// draft — the loop below then finds nothing for this invoice.
	t.Run("max level reached: no escalation available", func(t *testing.T) {
		invoiceID := uuid.New()
		repo := newStubDunningRecordRepo()
		sentAt := time.Now().Add(-40 * 24 * time.Hour)
		repo.byInvoice[invoiceID] = []*models.DunningRecord{{
			ID: uuid.New(), TenantID: tenantID, InvoiceID: invoiceID,
			Level: dunning.MaxDunningLevel, Status: models.DunningStatusSent, SentAt: &sentAt,
			Fee: decimal.Zero, Interest: decimal.Zero, CreatedBy: uuid.New(), CreatedAt: sentAt,
		}}
		invReader := &stubDunningInvoiceReader{overdue: []*models.Invoice{{
			ID: invoiceID, TenantID: tenantID, DueDate: time.Now().Add(-60 * 24 * time.Hour),
		}}}
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, invReader)
		_, err := srv.EscalateDunning(context.Background(), &bizv1.EscalateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: invoiceID.String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	// A paid invoice is never returned by GetOverdue (PostgresRepository.GetOverdue
	// filters status = 'sent', see postgres_repository.go) — this proves the
	// resulting "not found in the overdue set" path maps to the same client-safe
	// FailedPrecondition as the max-level case, not a 500 or a silent no-op.
	t.Run("paid invoice: no escalation available", func(t *testing.T) {
		invoiceID := uuid.New()
		invReader := &stubDunningInvoiceReader{overdue: []*models.Invoice{}}
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, invReader)
		_, err := srv.EscalateDunning(context.Background(), &bizv1.EscalateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: invoiceID.String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path escalates to level 1", func(t *testing.T) {
		invoiceID := uuid.New()
		invReader := &stubDunningInvoiceReader{overdue: []*models.Invoice{{
			ID: invoiceID, TenantID: tenantID, DueDate: time.Now().Add(-30 * 24 * time.Hour),
		}}}
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, invReader)
		resp, err := srv.EscalateDunning(context.Background(), &bizv1.EscalateDunningRequest{
			TenantId: tenantID.String(), InvoiceId: invoiceID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.GetDunning().GetLevel())
	})
}

// ============================================================================
// GetDunningConfig / UpdateDunningConfig
// ============================================================================

func TestGetDunningConfig(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.GetDunningConfig(context.Background(), &bizv1.GetDunningConfigRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns existing config", func(t *testing.T) {
		cfg := defaultDunningConfig(tenantID)
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: cfg}, &stubDunningInvoiceReader{})
		resp, err := srv.GetDunningConfig(context.Background(), &bizv1.GetDunningConfigRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(14), resp.GetConfig().GetLevel1DaysAfterDue())
	})

	t.Run("repo error maps to internal", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{getErr: errors.New("db down")}, &stubDunningInvoiceReader{})
		_, err := srv.GetDunningConfig(context.Background(), &bizv1.GetDunningConfigRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestUpdateDunningConfig(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("config required", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid level1_fee", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{
			TenantId: tenantID.String(),
			Config:   &bizv1.DunningConfig{Level1Fee: "not-a-decimal"},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path updates level days and fees", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{config: defaultDunningConfig(tenantID)}, &stubDunningInvoiceReader{})
		resp, err := srv.UpdateDunningConfig(context.Background(), &bizv1.UpdateDunningConfigRequest{
			TenantId: tenantID.String(),
			Config:   &bizv1.DunningConfig{Level1DaysAfterDue: 21, Level1Fee: "2.50"},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(21), resp.GetConfig().GetLevel1DaysAfterDue())
		gotFee, parseErr := decimal.NewFromString(resp.GetConfig().GetLevel1Fee())
		require.NoError(t, parseErr)
		assert.True(t, gotFee.Equal(decimal.NewFromFloat(2.5)))
	})
}

// ============================================================================
// UpdateDunningStatus / SendDunningNotice
// ============================================================================

func TestUpdateDunningStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningStatus(context.Background(), &bizv1.UpdateDunningStatusRequest{
			TenantId: tenantID.String(), Id: "not-a-uuid", Status: models.DunningStatusSent,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid status value maps to internal", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		dunningID := uuid.New()
		repo.records[dunningID] = &models.DunningRecord{ID: dunningID, TenantID: tenantID, Status: models.DunningStatusDraft}
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.UpdateDunningStatus(context.Background(), &bizv1.UpdateDunningStatusRequest{
			TenantId: tenantID.String(), Id: dunningID.String(), Status: "not-a-status",
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("happy path sets sent_at when flipping to sent", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		dunningID := uuid.New()
		repo.records[dunningID] = &models.DunningRecord{ID: dunningID, TenantID: tenantID, Status: models.DunningStatusDraft}
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		resp, err := srv.UpdateDunningStatus(context.Background(), &bizv1.UpdateDunningStatusRequest{
			TenantId: tenantID.String(), Id: dunningID.String(), Status: models.DunningStatusSent,
		})
		require.NoError(t, err)
		assert.Equal(t, bizv1.DunningStatus_DUNNING_SENT, resp.GetDunning().GetStatus())
		assert.NotNil(t, resp.GetDunning().GetSentAt())
	})
}

func TestSendDunningNotice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.SendDunningNotice(context.Background(), &bizv1.SendDunningNoticeRequest{
			TenantId: tenantID.String(), Id: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		dunningID := uuid.New()
		repo.records[dunningID] = &models.DunningRecord{ID: dunningID, TenantID: tenantID, Status: models.DunningStatusSent}
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		_, err := srv.SendDunningNotice(context.Background(), &bizv1.SendDunningNoticeRequest{
			TenantId: tenantID.String(), Id: dunningID.String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path flips draft to sent, no mailer configured", func(t *testing.T) {
		repo := newStubDunningRecordRepo()
		dunningID := uuid.New()
		repo.records[dunningID] = &models.DunningRecord{ID: dunningID, TenantID: tenantID, Status: models.DunningStatusDraft}
		srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
		resp, err := srv.SendDunningNotice(context.Background(), &bizv1.SendDunningNoticeRequest{
			TenantId: tenantID.String(), Id: dunningID.String(),
		})
		require.NoError(t, err)
		assert.False(t, resp.GetEmailQueued())
		assert.Equal(t, bizv1.DunningStatus_DUNNING_SENT, resp.GetDunning().GetStatus())
	})
}

// ============================================================================
// GenerateDunningPDF — NotFound/validation only, no maroto/v2 rendering
// (same pattern as TestGenerateQuotePDF_QuoteNotFound).
// ============================================================================

func TestGenerateDunningPDF_InvalidID(t *testing.T) {
	tenantID := uuid.New()
	srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
	_, err := srv.GenerateDunningPDF(context.Background(), &bizv1.GenerateDunningPDFRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGenerateDunningPDF_NotFound(t *testing.T) {
	tenantID := uuid.New()
	srv := newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})
	_, err := srv.GenerateDunningPDF(context.Background(), &bizv1.GenerateDunningPDFRequest{
		TenantId: tenantID.String(), Id: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// GetFinanceDashboard
// ============================================================================

func TestGetFinanceDashboard(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDashboardTestServer(&stubDashboardRepo{})
		_, err := srv.GetFinanceDashboard(context.Background(), &bizv1.GetFinanceDashboardRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path aggregates metrics", func(t *testing.T) {
		repo := &stubDashboardRepo{metrics: &dashboard.Metrics{
			Revenue: models.RevenueMetrics{
				TotalInvoiced:    decimal.NewFromInt(1000),
				TotalPaid:        decimal.NewFromInt(600),
				TotalOutstanding: decimal.NewFromInt(400),
				OverdueAmount:    decimal.NewFromInt(100),
			},
			Pipeline: models.PipelineMetrics{
				QuotesPending: 3, ConversionRate: decimal.NewFromInt(50), AverageDealSize: decimal.NewFromInt(200),
			},
			StatusBreakdown: models.InvoiceStatusBreakdown{Draft: 1, Sent: 2, Overdue: 1, Paid: 3, Cancelled: 0},
			MonthlyRevenue:  decimal.NewFromInt(600),
			MonthsInRange:   6,
		}}
		srv := newDashboardTestServer(repo)
		resp, err := srv.GetFinanceDashboard(context.Background(), &bizv1.GetFinanceDashboardRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, "1000", resp.GetDashboard().GetTotalInvoiced())
		assert.Equal(t, int32(3), resp.GetDashboard().GetQuotesPending())
	})

	t.Run("repo error maps to internal", func(t *testing.T) {
		srv := newDashboardTestServer(&stubDashboardRepo{err: errors.New("db down")})
		_, err := srv.GetFinanceDashboard(context.Background(), &bizv1.GetFinanceDashboardRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// ExportDATEV
// ============================================================================

func TestExportDATEV(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: "not-a-uuid", StartDate: "2026-01-01", EndDate: "2026-01-31",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid start_date", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "not-a-date", EndDate: "2026-01-31",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid end_date", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "2026-01-01", EndDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid fiscal_year_start", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "2026-01-01", EndDate: "2026-01-31", FiscalYearStart: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("end before start maps to invalid argument", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "2026-02-01", EndDate: "2026-01-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path renders an empty batch for the period", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		resp, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "2026-01-01", EndDate: "2026-01-31",
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.GetRecordCount())
		assert.NotEmpty(t, resp.GetCsvData())
		assert.Contains(t, resp.GetFilename(), "DATEV_Buchungsstapel")
	})

	t.Run("pager error maps to internal", func(t *testing.T) {
		srv := newDatevExportTestServer(&stubInvoicePager{err: errors.New("db down")}, &stubCreditNotePager{}, &stubCompanySettingsRepo{})
		_, err := srv.ExportDATEV(context.Background(), &bizv1.ExportDATEVRequest{
			TenantId: tenantID.String(), StartDate: "2026-01-01", EndDate: "2026-01-31",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// GenerateGoBDExport — validation only. The happy path reads through the
// concrete *invoice.Service and *creditnote.Service (BizGRPCServer.invoiceService/
// creditNoteService are struct fields, not interfaces), which would need the
// same heavyweight construction the invoice-coverage iteration already
// flagged as out of scope for CreateInvoiceFromTimeEntries/CreateQuoteFromDeal.
// See JOURNAL.md iteration for this unit.
// ============================================================================

func TestGenerateGoBDExport_Validation(t *testing.T) {
	tenantID := uuid.New()
	srv := &BizGRPCServer{}

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.GenerateGoBDExport(context.Background(), &bizv1.GenerateGoBDExportRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid from_date", func(t *testing.T) {
		_, err := srv.GenerateGoBDExport(context.Background(), &bizv1.GenerateGoBDExportRequest{
			TenantId: tenantID.String(), FromDate: "not-a-date", ToDate: "2026-01-31",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid to_date", func(t *testing.T) {
		_, err := srv.GenerateGoBDExport(context.Background(), &bizv1.GenerateGoBDExportRequest{
			TenantId: tenantID.String(), FromDate: "2026-01-01", ToDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// Before this check existed, a swapped range reached ListForDATEVExport
	// (fromDate > toDate can never match an invoice_date), which silently
	// returned zero rows — a 200 with an empty CSV, indistinguishable from
	// "no revenue in this period" instead of a rejected request.
	t.Run("end before start maps to invalid argument", func(t *testing.T) {
		_, err := srv.GenerateGoBDExport(context.Background(), &bizv1.GenerateGoBDExportRequest{
			TenantId: tenantID.String(), FromDate: "2026-02-01", ToDate: "2026-01-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
