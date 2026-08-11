package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// Test doubles — invoice/credit-note/payment repositories
// ============================================================================

type stubInvoiceRepo struct {
	invoices                 map[uuid.UUID]*models.Invoice
	createErr                error
	getErr                   error
	listErr                  error
	updateErr                error
	updateStatusErr          error
	setLockErr               error
	invoiceNumberExists      bool
	invoiceNumberExistsErr   error
	countByFiscalYear        int
	countByFiscalYearErr     error
	aggregatePaymentStats    invoice.PaymentStats
	aggregatePaymentStatsErr error
}

func newStubInvoiceRepo() *stubInvoiceRepo {
	return &stubInvoiceRepo{invoices: make(map[uuid.UUID]*models.Invoice)}
}

func (r *stubInvoiceRepo) Create(_ context.Context, inv *models.Invoice) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.invoices[inv.ID] = inv
	return nil
}

func (r *stubInvoiceRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	inv, ok := r.invoices[id]
	if !ok || inv.TenantID != tenantID {
		return nil, invoice.ErrInvoiceNotFound
	}
	return inv, nil
}

func (r *stubInvoiceRepo) List(_ context.Context, tenantID uuid.UUID, _ invoice.ListFilter) ([]*models.Invoice, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	var result []*models.Invoice
	for _, inv := range r.invoices {
		if inv.TenantID == tenantID {
			result = append(result, inv)
		}
	}
	return result, len(result), nil
}

func (r *stubInvoiceRepo) Update(_ context.Context, inv *models.Invoice) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.invoices[inv.ID] = inv
	return nil
}

func (r *stubInvoiceRepo) UpdateInTx(ctx context.Context, _ pgx.Tx, inv *models.Invoice) error {
	return r.Update(ctx, inv)
}

func (r *stubInvoiceRepo) UpdateStatus(_ context.Context, _, id uuid.UUID, status string) error {
	if r.updateStatusErr != nil {
		return r.updateStatusErr
	}
	if inv, ok := r.invoices[id]; ok {
		inv.Status = status
	}
	return nil
}

func (r *stubInvoiceRepo) UpdateStatusInTx(ctx context.Context, _ pgx.Tx, tenantID, id uuid.UUID, status string) error {
	return r.UpdateStatus(ctx, tenantID, id, status)
}

func (r *stubInvoiceRepo) GetOverdue(context.Context, uuid.UUID) ([]*models.Invoice, error) {
	return nil, nil
}

func (r *stubInvoiceRepo) GetByQuoteID(context.Context, uuid.UUID, uuid.UUID) (*models.Invoice, error) {
	return nil, invoice.ErrInvoiceNotFound
}

func (r *stubInvoiceRepo) LinkTimeTracking(context.Context, uuid.UUID, uuid.UUID, json.RawMessage) error {
	return nil
}

func (r *stubInvoiceRepo) SetLock(_ context.Context, _, id uuid.UUID, lockedAt time.Time, lockedBy uuid.UUID) error {
	if r.setLockErr != nil {
		return r.setLockErr
	}
	if inv, ok := r.invoices[id]; ok {
		inv.LockedAt = &lockedAt
		inv.LockedBy = &lockedBy
	}
	return nil
}

func (r *stubInvoiceRepo) UpsertImported(_ context.Context, inv *models.Invoice) error {
	r.invoices[inv.ID] = inv
	return nil
}

func (r *stubInvoiceRepo) InvoiceNumberExists(context.Context, uuid.UUID, string) (bool, error) {
	return r.invoiceNumberExists, r.invoiceNumberExistsErr
}

func (r *stubInvoiceRepo) CountByFiscalYear(context.Context, uuid.UUID, int) (int, error) {
	return r.countByFiscalYear, r.countByFiscalYearErr
}

func (r *stubInvoiceRepo) AggregatePaymentStats(context.Context, uuid.UUID, time.Time, time.Time) (invoice.PaymentStats, error) {
	return r.aggregatePaymentStats, r.aggregatePaymentStatsErr
}

func (r *stubInvoiceRepo) ListForGoBDExport(context.Context, uuid.UUID, time.Time, time.Time) ([]*models.Invoice, error) {
	return nil, nil
}

func (r *stubInvoiceRepo) ListForDATEVExport(context.Context, uuid.UUID, time.Time, time.Time, *time.Time, *uuid.UUID, int) ([]*models.Invoice, error) {
	return nil, nil
}

func (r *stubInvoiceRepo) ListDocumentChains(context.Context, uuid.UUID) ([]*models.DocumentChain, error) {
	return nil, nil
}

func (r *stubInvoiceRepo) ListTransactions(context.Context, uuid.UUID) ([]*models.FinanceTransaction, error) {
	return nil, nil
}

type stubCreditNoteRepo struct {
	notes     map[uuid.UUID]*models.CreditNote
	createErr error
	getErr    error
	updateErr error
}

func newStubCreditNoteRepo() *stubCreditNoteRepo {
	return &stubCreditNoteRepo{notes: make(map[uuid.UUID]*models.CreditNote)}
}

func (r *stubCreditNoteRepo) Create(_ context.Context, cn *models.CreditNote) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.notes[cn.ID] = cn
	return nil
}

func (r *stubCreditNoteRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	cn, ok := r.notes[id]
	if !ok || cn.TenantID != tenantID {
		return nil, creditnote.ErrCreditNoteNotFound
	}
	return cn, nil
}

func (r *stubCreditNoteRepo) Update(_ context.Context, cn *models.CreditNote) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.notes[cn.ID] = cn
	return nil
}

func (r *stubCreditNoteRepo) UpdateInTx(ctx context.Context, _ pgx.Tx, cn *models.CreditNote) error {
	return r.Update(ctx, cn)
}

func (r *stubCreditNoteRepo) List(_ context.Context, tenantID uuid.UUID, _ creditnote.ListFilter) ([]*models.CreditNote, int, error) {
	var result []*models.CreditNote
	for _, cn := range r.notes {
		if cn.TenantID == tenantID {
			result = append(result, cn)
		}
	}
	return result, len(result), nil
}

func (r *stubCreditNoteRepo) GetByInvoiceID(context.Context, uuid.UUID, uuid.UUID) ([]*models.CreditNote, error) {
	return nil, nil
}

func (r *stubCreditNoteRepo) ListForDATEVExport(context.Context, uuid.UUID, time.Time, time.Time, *time.Time, *uuid.UUID, int) ([]*models.CreditNote, error) {
	return nil, nil
}

type stubPaymentRepo struct {
	payments  map[uuid.UUID]*models.Payment
	createErr error
	getErr    error
	listErr   error
	deleteErr error
	sumErr    error
}

func newStubPaymentRepo() *stubPaymentRepo {
	return &stubPaymentRepo{payments: make(map[uuid.UUID]*models.Payment)}
}

func (r *stubPaymentRepo) Create(_ context.Context, p *models.Payment) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.payments[p.ID] = p
	return nil
}

func (r *stubPaymentRepo) List(_ context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*models.Payment
	for _, p := range r.payments {
		if p.TenantID == tenantID && p.InvoiceID == invoiceID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *stubPaymentRepo) Delete(_ context.Context, _, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.payments, id)
	return nil
}

func (r *stubPaymentRepo) sumByInvoiceID(tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	if r.sumErr != nil {
		return decimal.Decimal{}, r.sumErr
	}
	total := decimal.Zero
	for _, p := range r.payments {
		if p.TenantID == tenantID && p.InvoiceID == invoiceID {
			total = total.Add(p.Amount)
		}
	}
	return total, nil
}

func (r *stubPaymentRepo) SumByInvoiceID(_ context.Context, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	return r.sumByInvoiceID(tenantID, invoiceID)
}

func (r *stubPaymentRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Payment, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	p, ok := r.payments[id]
	if !ok || p.TenantID != tenantID {
		return nil, payment.ErrNotFound
	}
	return p, nil
}

func (r *stubPaymentRepo) GetByIdempotencyKey(_ context.Context, tenantID uuid.UUID, key string) (*models.Payment, error) {
	for _, p := range r.payments {
		if p.TenantID == tenantID && p.IdempotencyKey == key {
			return p, nil
		}
	}
	return nil, nil
}

func (r *stubPaymentRepo) CreateInTx(ctx context.Context, _ pgx.Tx, p *models.Payment) error {
	return r.Create(ctx, p)
}

func (r *stubPaymentRepo) DeleteInTx(ctx context.Context, _ pgx.Tx, tenantID, id uuid.UUID) error {
	return r.Delete(ctx, tenantID, id)
}

func (r *stubPaymentRepo) SumByInvoiceIDInTx(_ context.Context, _ pgx.Tx, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	return r.sumByInvoiceID(tenantID, invoiceID)
}

// stubInvoiceNumberSeqRepo satisfies invoice.NumberSequenceRepo (NextNumber,
// NextNumberInTx, GetSequenceInfo) — a superset of quote/creditnote's
// two-method NumberSequenceRepo, so it is dedicated to the invoice service
// rather than reusing stubNumberSeqRepo from biz_grpc_errormap_settings_quotes_test.go.
type stubInvoiceNumberSeqRepo struct {
	number       string
	err          error
	sequenceInfo *invoice.SequenceInfo
	sequenceErr  error
}

func (s *stubInvoiceNumberSeqRepo) NextNumber(context.Context, uuid.UUID, string, int, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.number, nil
}

func (s *stubInvoiceNumberSeqRepo) NextNumberInTx(context.Context, pgx.Tx, uuid.UUID, string, int, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.number, nil
}

func (s *stubInvoiceNumberSeqRepo) GetSequenceInfo(context.Context, uuid.UUID, string, int) (*invoice.SequenceInfo, error) {
	if s.sequenceErr != nil {
		return nil, s.sequenceErr
	}
	return s.sequenceInfo, nil
}

// newFinanceTestServer wires the invoice/credit-note/payment services against
// the stub repositories above, mirroring newQuoteTestServer's pattern. Any of
// the number-sequence stubs may be nil (typed-nil-safe) as long as the test
// does not exercise the codepath that calls it (Send/RecordPayment/Delete).
func newFinanceTestServer(invRepo *stubInvoiceRepo, cnRepo *stubCreditNoteRepo, payRepo *stubPaymentRepo, settings *stubCompanySettingsRepo, invNumberSeq *stubInvoiceNumberSeqRepo, cnNumberSeq *stubNumberSeqRepo) *BizGRPCServer {
	invSvc := invoice.NewService(invRepo, invNumberSeq, settings, nil, fakeTxBeginner{})
	cnSvc := creditnote.NewService(cnRepo, invRepo, cnNumberSeq, fakeTxBeginner{})
	paySvc := payment.NewService(payRepo, invRepo, invRepo, fakeTxBeginner{})
	return &BizGRPCServer{
		invoiceService:    invSvc,
		creditNoteService: cnSvc,
		paymentService:    paySvc,
		companySettings:   settings,
	}
}

func draftInvoice(tenantID uuid.UUID) *models.Invoice {
	return &models.Invoice{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Status:       models.InvoiceStatusDraft,
		CustomerName: "Acme GmbH",
		LineItems:    []byte(`[]`),
		GrossTotal:   decimal.NewFromInt(100),
	}
}

func creditableInvoice(tenantID uuid.UUID) *models.Invoice {
	inv := draftInvoice(tenantID)
	inv.Status = models.InvoiceStatusSent
	inv.InvoiceNumber = "RE-2026-0001"
	return inv
}

// ============================================================================
// Invoices
// ============================================================================

func TestCreateInvoice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateInvoice(context.Background(), &bizv1.CreateInvoiceRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateInvoice(context.Background(), &bizv1.CreateInvoiceRequest{TenantId: tenantID.String(), CreatedBy: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no line items maps to InvalidArgument", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateInvoice(context.Background(), &bizv1.CreateInvoiceRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		resp, err := srv.CreateInvoice(context.Background(), &bizv1.CreateInvoiceRequest{
			TenantId:  tenantID.String(),
			CreatedBy: uuid.New().String(),
			Customer:  &bizv1.CustomerSnapshot{Name: "Acme GmbH"},
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		require.NoError(t, err)
		assert.Equal(t, "Acme GmbH", resp.GetInvoice().GetCustomer().GetName())
		assert.Equal(t, bizv1.InvoiceStatus_INVOICE_DRAFT, resp.GetInvoice().GetStatus())
	})
}

func TestGetInvoice(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	t.Run("invalid id", func(t *testing.T) {
		_, err := srv.GetInvoice(context.Background(), &bizv1.GetInvoiceRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found maps to NotFound", func(t *testing.T) {
		_, err := srv.GetInvoice(context.Background(), &bizv1.GetInvoiceRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.GetInvoice(context.Background(), &bizv1.GetInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, inv.ID.String(), resp.GetInvoice().GetId())
	})
}

func TestListInvoices(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := draftInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListInvoices(context.Background(), &bizv1.ListInvoicesRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid contact_id", func(t *testing.T) {
		_, err := srv.ListInvoices(context.Background(), &bizv1.ListInvoicesRequest{TenantId: tenantID.String(), ContactId: strPtr("not-a-uuid")})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.ListInvoices(context.Background(), &bizv1.ListInvoicesRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.GetTotal())
	})
}

func TestUpdateInvoice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusSent
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.UpdateInvoice(context.Background(), &bizv1.UpdateInvoiceRequest{
			TenantId: tenantID.String(), Id: inv.ID.String(), Notes: "aktualisiert",
		})
		require.NoError(t, err)
		assert.Equal(t, "aktualisiert", resp.GetInvoice().GetNotes())
	})
}

func TestSendInvoice(t *testing.T) {
	tenantID := uuid.New()
	numberSeq := &stubInvoiceNumberSeqRepo{number: "RE-2026-0001"}

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)
		_, err := srv.SendInvoice(context.Background(), &bizv1.SendInvoiceRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusSent
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)

		_, err := srv.SendInvoice(context.Background(), &bizv1.SendInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)

		resp, err := srv.SendInvoice(context.Background(), &bizv1.SendInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, "RE-2026-0001", resp.GetInvoice().GetInvoiceNumber())
		assert.Equal(t, bizv1.InvoiceStatus_INVOICE_SENT, resp.GetInvoice().GetStatus())
	})
}

func TestMarkInvoicePaid(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.MarkInvoicePaid(context.Background(), &bizv1.MarkInvoicePaidRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("already paid maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusPaid
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.MarkInvoicePaid(context.Background(), &bizv1.MarkInvoicePaidRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusSent
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.MarkInvoicePaid(context.Background(), &bizv1.MarkInvoicePaidRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, bizv1.InvoiceStatus_INVOICE_PAID, resp.GetInvoice().GetStatus())
	})
}

func TestCancelInvoice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CancelInvoice(context.Background(), &bizv1.CancelInvoiceRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("already paid maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusPaid
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.CancelInvoice(context.Background(), &bizv1.CancelInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("sent invoice without storno creator maps to Internal", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		inv.Status = models.InvoiceStatusSent
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.CancelInvoice(context.Background(), &bizv1.CancelInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("happy path cancels a draft invoice directly", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.CancelInvoice(context.Background(), &bizv1.CancelInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, bizv1.InvoiceStatus_INVOICE_CANCELLED, resp.GetInvoice().GetStatus())
	})
}

func TestValidateInvoiceNumber(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.ValidateInvoiceNumber(context.Background(), &bizv1.ValidateInvoiceNumberRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid format returns ValidFormat false without error", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		resp, err := srv.ValidateInvoiceNumber(context.Background(), &bizv1.ValidateInvoiceNumberRequest{
			TenantId: tenantID.String(), InvoiceNumber: "not-a-number",
		})
		require.NoError(t, err)
		assert.False(t, resp.GetValidFormat())
	})

	t.Run("already used", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		repo.invoiceNumberExists = true
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		resp, err := srv.ValidateInvoiceNumber(context.Background(), &bizv1.ValidateInvoiceNumberRequest{
			TenantId: tenantID.String(), InvoiceNumber: "RE-2026-0001",
		})
		require.NoError(t, err)
		assert.True(t, resp.GetValidFormat())
		assert.True(t, resp.GetAlreadyUsed())
	})

	t.Run("happy path canonicalizes the number", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		resp, err := srv.ValidateInvoiceNumber(context.Background(), &bizv1.ValidateInvoiceNumberRequest{
			TenantId: tenantID.String(), InvoiceNumber: "RE-2026-00001",
		})
		require.NoError(t, err)
		assert.True(t, resp.GetValidFormat())
		assert.False(t, resp.GetAlreadyUsed())
		assert.Equal(t, "RE-2026-0001", resp.GetCanonical())
	})
}

func TestLockInvoice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.LockInvoice(context.Background(), &bizv1.LockInvoiceRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid locked_by", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.LockInvoice(context.Background(), &bizv1.LockInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String(), LockedBy: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("draft invoice cannot be locked", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.LockInvoice(context.Background(), &bizv1.LockInvoiceRequest{
			TenantId: tenantID.String(), Id: inv.ID.String(), LockedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("already locked maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		now := time.Now()
		lockedBy := uuid.New()
		inv.LockedAt = &now
		inv.LockedBy = &lockedBy
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.LockInvoice(context.Background(), &bizv1.LockInvoiceRequest{
			TenantId: tenantID.String(), Id: inv.ID.String(), LockedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path locks a sent invoice", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.LockInvoice(context.Background(), &bizv1.LockInvoiceRequest{
			TenantId: tenantID.String(), Id: inv.ID.String(), LockedBy: uuid.New().String(),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetLockedAt())
		assert.NotEmpty(t, resp.GetInvoiceJson())
	})
}

func TestGetPaymentStats(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.GetPaymentStats(context.Background(), &bizv1.GetPaymentStatsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid from_date", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.GetPaymentStats(context.Background(), &bizv1.GetPaymentStatsRequest{TenantId: tenantID.String(), FromDate: "not-a-date", ToDate: "2026-01-31"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("to_date before from_date", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.GetPaymentStats(context.Background(), &bizv1.GetPaymentStatsRequest{
			TenantId: tenantID.String(), FromDate: "2026-02-01", ToDate: "2026-01-01",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		repo.aggregatePaymentStats = invoice.PaymentStats{
			TotalInvoices:          3,
			TotalPaid:              2,
			TotalOutstanding:       1,
			TotalPaidAmount:        decimal.NewFromInt(500),
			TotalOutstandingAmount: decimal.NewFromInt(100),
			AverageDaysToPay:       decimal.NewFromFloat(4.5),
		}
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.GetPaymentStats(context.Background(), &bizv1.GetPaymentStatsRequest{
			TenantId: tenantID.String(), FromDate: "2026-01-01", ToDate: "2026-01-31",
		})
		require.NoError(t, err)
		assert.Equal(t, int32(3), resp.GetTotalInvoices())
		assert.Equal(t, "500.00", resp.GetTotalPaidAmount())
		assert.Equal(t, "4.5", resp.GetAverageDaysToPay())
	})
}

func TestGetJournalSummary(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		numberSeq := &stubInvoiceNumberSeqRepo{}
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)
		_, err := srv.GetJournalSummary(context.Background(), &bizv1.GetJournalSummaryRequest{TenantId: "not-a-uuid", Year: 2026})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid year", func(t *testing.T) {
		numberSeq := &stubInvoiceNumberSeqRepo{}
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)
		_, err := srv.GetJournalSummary(context.Background(), &bizv1.GetJournalSummaryRequest{TenantId: tenantID.String(), Year: 1999})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("sequence lookup error maps to Internal", func(t *testing.T) {
		numberSeq := &stubInvoiceNumberSeqRepo{sequenceErr: errSequenceLookupFailed}
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)
		_, err := srv.GetJournalSummary(context.Background(), &bizv1.GetJournalSummaryRequest{TenantId: tenantID.String(), Year: 2026})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("no invoices issued in year", func(t *testing.T) {
		numberSeq := &stubInvoiceNumberSeqRepo{}
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)
		resp, err := srv.GetJournalSummary(context.Background(), &bizv1.GetJournalSummaryRequest{TenantId: tenantID.String(), Year: 2026})
		require.NoError(t, err)
		assert.Equal(t, int32(2026), resp.GetYear())
		assert.Equal(t, int32(0), resp.GetTotalInvoicesIssued())
	})

	t.Run("gap detected when issued count trails the sequence", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		repo.countByFiscalYear = 3
		numberSeq := &stubInvoiceNumberSeqRepo{sequenceInfo: &invoice.SequenceInfo{CurrentNumber: 5, FiscalYear: 2026}}
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, numberSeq, nil)

		resp, err := srv.GetJournalSummary(context.Background(), &bizv1.GetJournalSummaryRequest{TenantId: tenantID.String(), Year: 2026})
		require.NoError(t, err)
		assert.Equal(t, int32(5), resp.GetHighestSequence())
		assert.Equal(t, int32(3), resp.GetTotalInvoicesIssued())
		assert.Equal(t, int32(2), resp.GetGapsDetected())
		assert.Equal(t, "RE-2026-0005", resp.GetLastNumber())
	})
}

var errSequenceLookupFailed = errors.New("sequence lookup failed")

// ============================================================================
// GenerateInvoicePDF / GenerateZUGFeRDInvoicePDF / GenerateEInvoice
// ============================================================================

func TestGenerateInvoicePDF_InvoiceNotFound(t *testing.T) {
	tenantID := uuid.New()
	srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	_, err := srv.GenerateInvoicePDF(context.Background(), &bizv1.GenerateInvoicePDFRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGenerateInvoicePDF_MissingCompanySettings(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubInvoiceRepo()
	inv := creditableInvoice(tenantID)
	repo.invoices[inv.ID] = inv
	srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	_, err := srv.GenerateInvoicePDF(context.Background(), &bizv1.GenerateInvoicePDFRequest{TenantId: tenantID.String(), Id: inv.ID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGenerateZUGFeRDInvoicePDF_InvoiceNotFound(t *testing.T) {
	tenantID := uuid.New()
	srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	_, err := srv.GenerateZUGFeRDInvoicePDF(context.Background(), &bizv1.GenerateZUGFeRDInvoicePDFRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGenerateEInvoice(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invoice not found", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.GenerateEInvoice(context.Background(), &bizv1.GenerateEInvoiceRequest{TenantId: tenantID.String(), Id: uuid.New().String(), Format: "xrechnung"})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("unknown format maps to InvalidArgument", func(t *testing.T) {
		repo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		repo.invoices[inv.ID] = inv
		settings := &stubCompanySettingsRepo{settings: &models.CompanySettings{TenantID: tenantID, Name: "Acme GmbH"}}
		srv := newFinanceTestServer(repo, newStubCreditNoteRepo(), newStubPaymentRepo(), settings, nil, nil)

		_, err := srv.GenerateEInvoice(context.Background(), &bizv1.GenerateEInvoiceRequest{TenantId: tenantID.String(), Id: inv.ID.String(), Format: "not-a-format"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// CreateInvoiceFromTimeEntries — validation paths only. A happy path needs a
// full hr/timetracking.WorkTimeRepository fake (12 methods); out of scope for
// this iteration, tracked in the journal under "offen".
// ============================================================================

func TestCreateInvoiceFromTimeEntries_Validation(t *testing.T) {
	tenantID := uuid.New()
	employeeID := uuid.New()

	newSrv := func() *BizGRPCServer {
		return newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
	}

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid employee_id", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("customer_name required", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: employeeID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid date_from", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: employeeID.String(), CustomerName: "Acme GmbH", DateFrom: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid date_to", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: employeeID.String(), CustomerName: "Acme GmbH",
			DateFrom: "2026-01-01", DateTo: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid hourly_rate", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: employeeID.String(), CustomerName: "Acme GmbH",
			DateFrom: "2026-01-01", DateTo: "2026-01-31", HourlyRate: "not-a-number",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no timetracking repository configured maps to Unavailable", func(t *testing.T) {
		_, err := newSrv().CreateInvoiceFromTimeEntries(context.Background(), &bizv1.CreateInvoiceFromTimeEntriesRequest{
			TenantId: tenantID.String(), EmployeeId: employeeID.String(), CustomerName: "Acme GmbH",
			DateFrom: "2026-01-01", DateTo: "2026-01-31", HourlyRate: "50.00",
		})
		requireGRPCCode(t, err, codes.Unavailable)
	})
}

// ============================================================================
// Credit Notes
// ============================================================================

func TestCreateCreditNote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{TenantId: tenantID.String(), CreatedBy: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid original_invoice_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), OriginalInvoiceId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no line items maps to InvalidArgument", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), OriginalInvoiceId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("original invoice not found maps to NotFound", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), OriginalInvoiceId: uuid.New().String(),
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("original invoice not in a creditable state", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID) // still draft — not sent/paid/overdue
		invRepo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(invRepo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), OriginalInvoiceId: inv.ID.String(),
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(invRepo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.CreateCreditNote(context.Background(), &bizv1.CreateCreditNoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), OriginalInvoiceId: inv.ID.String(),
			LineItems: []*bizv1.LineItem{lineItem()}, Reason: "Rabatt",
		})
		require.NoError(t, err)
		assert.Equal(t, bizv1.CreditNoteStatus_CREDIT_DRAFT, resp.GetCreditNote().GetStatus())
	})
}

func TestGetCreditNote(t *testing.T) {
	tenantID := uuid.New()
	cnRepo := newStubCreditNoteRepo()
	cn := &models.CreditNote{ID: uuid.New(), TenantID: tenantID, Status: models.CreditNoteStatusDraft, LineItems: []byte(`[]`)}
	cnRepo.notes[cn.ID] = cn
	srv := newFinanceTestServer(newStubInvoiceRepo(), cnRepo, newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	t.Run("invalid id", func(t *testing.T) {
		_, err := srv.GetCreditNote(context.Background(), &bizv1.GetCreditNoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found maps to NotFound", func(t *testing.T) {
		_, err := srv.GetCreditNote(context.Background(), &bizv1.GetCreditNoteRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.GetCreditNote(context.Background(), &bizv1.GetCreditNoteRequest{TenantId: tenantID.String(), Id: cn.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, cn.ID.String(), resp.GetCreditNote().GetId())
	})
}

func TestListCreditNotes(t *testing.T) {
	tenantID := uuid.New()
	cnRepo := newStubCreditNoteRepo()
	cn := &models.CreditNote{ID: uuid.New(), TenantID: tenantID, Status: models.CreditNoteStatusDraft, LineItems: []byte(`[]`)}
	cnRepo.notes[cn.ID] = cn
	srv := newFinanceTestServer(newStubInvoiceRepo(), cnRepo, newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListCreditNotes(context.Background(), &bizv1.ListCreditNotesRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		_, err := srv.ListCreditNotes(context.Background(), &bizv1.ListCreditNotesRequest{TenantId: tenantID.String(), InvoiceId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.ListCreditNotes(context.Background(), &bizv1.ListCreditNotesRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.GetTotal())
	})
}

func TestSendCreditNote(t *testing.T) {
	tenantID := uuid.New()
	cnNumberSeq := &stubNumberSeqRepo{number: "GS-2026-0001"}

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, cnNumberSeq)
		_, err := srv.SendCreditNote(context.Background(), &bizv1.SendCreditNoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		cnRepo := newStubCreditNoteRepo()
		cn := &models.CreditNote{ID: uuid.New(), TenantID: tenantID, Status: models.CreditNoteStatusSent, LineItems: []byte(`[]`)}
		cnRepo.notes[cn.ID] = cn
		srv := newFinanceTestServer(newStubInvoiceRepo(), cnRepo, newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, cnNumberSeq)

		_, err := srv.SendCreditNote(context.Background(), &bizv1.SendCreditNoteRequest{TenantId: tenantID.String(), Id: cn.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		cnRepo := newStubCreditNoteRepo()
		cn := &models.CreditNote{ID: uuid.New(), TenantID: tenantID, Status: models.CreditNoteStatusDraft, LineItems: []byte(`[]`)}
		cnRepo.notes[cn.ID] = cn
		srv := newFinanceTestServer(newStubInvoiceRepo(), cnRepo, newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, cnNumberSeq)

		resp, err := srv.SendCreditNote(context.Background(), &bizv1.SendCreditNoteRequest{TenantId: tenantID.String(), Id: cn.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, "GS-2026-0001", resp.GetCreditNote().GetCreditNoteNumber())
		assert.Equal(t, bizv1.CreditNoteStatus_CREDIT_SENT, resp.GetCreditNote().GetStatus())
	})
}

func TestGenerateCreditNotePDF_NotFound(t *testing.T) {
	tenantID := uuid.New()
	srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

	_, err := srv.GenerateCreditNotePDF(context.Background(), &bizv1.GenerateCreditNotePDFRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Payments
// ============================================================================

func TestRecordPayment(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.RecordPayment(context.Background(), &bizv1.RecordPaymentRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.RecordPayment(context.Background(), &bizv1.RecordPaymentRequest{TenantId: tenantID.String(), InvoiceId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid amount", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.RecordPayment(context.Background(), &bizv1.RecordPaymentRequest{
			TenantId: tenantID.String(), InvoiceId: uuid.New().String(), CreatedBy: uuid.New().String(), Amount: "not-a-number",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invoice not payable in draft status", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := draftInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(invRepo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.RecordPayment(context.Background(), &bizv1.RecordPaymentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), CreatedBy: uuid.New().String(), Amount: "50.00",
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("happy path records a partial payment", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv
		srv := newFinanceTestServer(invRepo, newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)

		resp, err := srv.RecordPayment(context.Background(), &bizv1.RecordPaymentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), CreatedBy: uuid.New().String(), Amount: "50.00",
		})
		require.NoError(t, err)
		assert.Equal(t, "50", resp.GetPayment().GetAmount())
		assert.Equal(t, models.InvoiceStatusSent, inv.Status, "partial payment must not flip the invoice to paid")
	})
}

func TestListPayments(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	payRepo := newStubPaymentRepo()
	p := &models.Payment{ID: uuid.New(), TenantID: tenantID, InvoiceID: invoiceID, Amount: decimal.NewFromInt(50)}
	payRepo.payments[p.ID] = p
	srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), payRepo, &stubCompanySettingsRepo{}, nil, nil)

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListPayments(context.Background(), &bizv1.ListPaymentsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid invoice_id", func(t *testing.T) {
		_, err := srv.ListPayments(context.Background(), &bizv1.ListPaymentsRequest{TenantId: tenantID.String(), InvoiceId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.ListPayments(context.Background(), &bizv1.ListPaymentsRequest{TenantId: tenantID.String(), InvoiceId: invoiceID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.GetTotal())
	})
}

func TestDeletePayment(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.DeletePayment(context.Background(), &bizv1.DeletePaymentRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found maps to NotFound", func(t *testing.T) {
		srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
		_, err := srv.DeletePayment(context.Background(), &bizv1.DeletePaymentRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		invRepo := newStubInvoiceRepo()
		inv := creditableInvoice(tenantID)
		invRepo.invoices[inv.ID] = inv
		payRepo := newStubPaymentRepo()
		p := &models.Payment{ID: uuid.New(), TenantID: tenantID, InvoiceID: inv.ID, Amount: decimal.NewFromInt(50)}
		payRepo.payments[p.ID] = p
		srv := newFinanceTestServer(invRepo, newStubCreditNoteRepo(), payRepo, &stubCompanySettingsRepo{}, nil, nil)

		_, err := srv.DeletePayment(context.Background(), &bizv1.DeletePaymentRequest{TenantId: tenantID.String(), Id: p.ID.String()})
		require.NoError(t, err)
		_, stillThere := payRepo.payments[p.ID]
		assert.False(t, stillThere)
	})
}
