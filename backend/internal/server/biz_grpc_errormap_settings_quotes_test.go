package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/biz/recurring"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// mapBizError — every sentinel individually against its expected gRPC code
// ============================================================================

func TestMapBizError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		// Quote errors
		{"quote not found", quote.ErrQuoteNotFound, codes.NotFound},
		{"quote not draft", quote.ErrQuoteNotDraft, codes.FailedPrecondition},
		{"quote not sent", quote.ErrQuoteNotSent, codes.FailedPrecondition},
		{"quote no line items", quote.ErrNoLineItems, codes.InvalidArgument},
		// Recurring invoice errors
		{"recurring not found", recurring.ErrNotFound, codes.NotFound},
		{"recurring no line items", recurring.ErrNoLineItems, codes.InvalidArgument},
		{"recurring invalid interval", recurring.ErrInvalidInterval, codes.InvalidArgument},
		{"recurring invalid status", recurring.ErrInvalidStatus, codes.InvalidArgument},
		{"recurring invalid date range", recurring.ErrInvalidDateRange, codes.InvalidArgument},
		{"recurring not active", recurring.ErrNotActive, codes.FailedPrecondition},
		{"recurring schedule exhausted", recurring.ErrScheduleExhausted, codes.FailedPrecondition},
		// Open items (Offene Posten)
		{"unknown aging bucket", models.ErrUnknownAgingBucket, codes.InvalidArgument},
		// Invoice errors
		{"invoice not found", invoice.ErrInvoiceNotFound, codes.NotFound},
		{"invoice immutable", invoice.ErrInvoiceImmutable, codes.FailedPrecondition},
		{"invoice not draft", invoice.ErrInvoiceNotDraft, codes.FailedPrecondition},
		{"invoice already paid", invoice.ErrInvoiceAlreadyPaid, codes.FailedPrecondition},
		{"invoice already cancelled", invoice.ErrInvoiceAlreadyCancelled, codes.FailedPrecondition},
		{"invoice no line items", invoice.ErrNoLineItems, codes.InvalidArgument},
		{"quote not accepted", invoice.ErrQuoteNotAccepted, codes.FailedPrecondition},
		{"invoice locked", invoice.ErrInvoiceLocked, codes.FailedPrecondition},
		{"storno unavailable", invoice.ErrStornoUnavailable, codes.Internal},
		// Credit note errors
		{"credit note not found", creditnote.ErrCreditNoteNotFound, codes.NotFound},
		{"credit note not draft", creditnote.ErrCreditNoteNotDraft, codes.FailedPrecondition},
		{"invalid invoice for credit", creditnote.ErrInvalidInvoiceForCredit, codes.FailedPrecondition},
		{"invoice already credited", creditnote.ErrInvoiceAlreadyCredited, codes.FailedPrecondition},
		{"credit note no line items", creditnote.ErrNoLineItems, codes.InvalidArgument},
		// Outbound e-invoice errors
		{"einvoice validation failed", einvoice.ErrValidationFailed, codes.FailedPrecondition},
		{"einvoice generate failed", einvoice.ErrGenerateFailed, codes.FailedPrecondition},
		{"einvoice totals mismatch", einvoice.ErrTotalsMismatch, codes.FailedPrecondition},
		// Payment errors
		{"payment not found", payment.ErrNotFound, codes.NotFound},
		{"payment invoice locked", payment.ErrInvoiceLocked, codes.FailedPrecondition},
		// Dunning errors
		{"dunning not found", dunning.ErrDunningNotFound, codes.NotFound},
		{"dunning not draft", dunning.ErrDunningNotDraft, codes.FailedPrecondition},
		{"dunning max level", dunning.ErrDunningMaxLevel, codes.FailedPrecondition},
		{"dunning config not found", dunning.ErrConfigNotFound, codes.NotFound},
		// Fallback
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapBizError(tt.err), tt.wantCode)
		})
	}
}

func TestMapBizError_Nil(t *testing.T) {
	assert.NoError(t, mapBizError(nil))
}

// ============================================================================
// Test doubles — company settings, quote repository, number sequence, tx
// ============================================================================

// stubCompanySettingsRepo satisfies both server.CompanySettingsRepository
// (GetByTenantID+Upsert) and quote.CompanySettingsRepo (GetByTenantID only).
type stubCompanySettingsRepo struct {
	settings  *models.CompanySettings
	getErr    error
	upsertErr error
}

func (s *stubCompanySettingsRepo) GetByTenantID(_ context.Context, _ uuid.UUID) (*models.CompanySettings, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.settings, nil
}

func (s *stubCompanySettingsRepo) Upsert(_ context.Context, settings *models.CompanySettings) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.settings = settings
	return nil
}

type stubQuoteRepo struct {
	quotes          map[uuid.UUID]*models.Quote
	createErr       error
	getErr          error
	listErr         error
	updateErr       error
	deleteErr       error
	updateStatusErr error
}

func newStubQuoteRepo() *stubQuoteRepo {
	return &stubQuoteRepo{quotes: make(map[uuid.UUID]*models.Quote)}
}

func (r *stubQuoteRepo) Create(_ context.Context, q *models.Quote) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.quotes[q.ID] = q
	return nil
}

func (r *stubQuoteRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	q, ok := r.quotes[id]
	if !ok || q.TenantID != tenantID {
		return nil, quote.ErrQuoteNotFound
	}
	return q, nil
}

func (r *stubQuoteRepo) List(_ context.Context, tenantID uuid.UUID, filter quote.ListFilter) ([]*models.Quote, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	var result []*models.Quote
	for _, q := range r.quotes {
		if q.TenantID != tenantID {
			continue
		}
		if filter.Status != "" && q.Status != filter.Status {
			continue
		}
		if filter.DealID != nil && (q.DealID == nil || *q.DealID != *filter.DealID) {
			continue
		}
		result = append(result, q)
	}
	return result, len(result), nil
}

func (r *stubQuoteRepo) Update(_ context.Context, q *models.Quote) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.quotes[q.ID] = q
	return nil
}

func (r *stubQuoteRepo) UpdateInTx(ctx context.Context, _ pgx.Tx, q *models.Quote) error {
	return r.Update(ctx, q)
}

func (r *stubQuoteRepo) Delete(_ context.Context, _, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.quotes, id)
	return nil
}

func (r *stubQuoteRepo) UpdateStatus(_ context.Context, _, id uuid.UUID, status string) error {
	if r.updateStatusErr != nil {
		return r.updateStatusErr
	}
	if q, ok := r.quotes[id]; ok {
		q.Status = status
	}
	return nil
}

func (r *stubQuoteRepo) GetByDealID(_ context.Context, tenantID, dealID uuid.UUID) ([]*models.Quote, error) {
	var result []*models.Quote
	for _, q := range r.quotes {
		if q.DealID != nil && *q.DealID == dealID && q.TenantID == tenantID {
			result = append(result, q)
		}
	}
	return result, nil
}

type stubNumberSeqRepo struct {
	number string
	err    error
}

func (s *stubNumberSeqRepo) NextNumber(context.Context, uuid.UUID, string, int, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.number, nil
}

func (s *stubNumberSeqRepo) NextNumberInTx(context.Context, pgx.Tx, uuid.UUID, string, int, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.number, nil
}

// fakeTxBeginner/fakeTx let Send's atomic numbering run against the stub repos
// without a database — same pattern as quote.noopTxBeginner in the biz package's
// own unit tests, duplicated here because that type is unexported.
type fakeTxBeginner struct{}

func (fakeTxBeginner) Begin(context.Context) (pgx.Tx, error) { return fakeTx{}, nil }

type fakeTx struct{ pgx.Tx }

func (fakeTx) Commit(context.Context) error   { return nil }
func (fakeTx) Rollback(context.Context) error { return nil }

func newQuoteTestServer(repo *stubQuoteRepo, settings *stubCompanySettingsRepo, numberSeq *stubNumberSeqRepo) *BizGRPCServer {
	svc := quote.NewService(repo, numberSeq, settings, nil, fakeTxBeginner{})
	return &BizGRPCServer{quoteService: svc, companySettings: settings}
}

func draftQuote(tenantID uuid.UUID) *models.Quote {
	return &models.Quote{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Status:       models.QuoteStatusDraft,
		CustomerName: "Acme GmbH",
		LineItems:    []byte(`[]`),
	}
}

// ============================================================================
// Company Settings
// ============================================================================

func TestGetCompanySettings(t *testing.T) {
	tenantID := uuid.New()
	settings := &stubCompanySettingsRepo{settings: &models.CompanySettings{TenantID: tenantID, Name: "Acme GmbH"}}
	srv := newQuoteTestServer(newStubQuoteRepo(), settings, nil)

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.GetCompanySettings(context.Background(), &bizv1.GetCompanySettingsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.GetCompanySettings(context.Background(), &bizv1.GetCompanySettingsRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, "Acme GmbH", resp.GetSettings().GetName())
	})
}

func TestUpdateCompanySettings(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.UpdateCompanySettings(context.Background(), &bizv1.UpdateCompanySettingsRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("settings required", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.UpdateCompanySettings(context.Background(), &bizv1.UpdateCompanySettingsRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid basiszinssatz", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.UpdateCompanySettings(context.Background(), &bizv1.UpdateCompanySettingsRequest{
			TenantId: tenantID.String(),
			Settings: &bizv1.CompanySettings{Name: "Acme GmbH", Basiszinssatz: "not-a-number"},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		settings := &stubCompanySettingsRepo{settings: &models.CompanySettings{TenantID: tenantID}}
		srv := newQuoteTestServer(newStubQuoteRepo(), settings, nil)
		resp, err := srv.UpdateCompanySettings(context.Background(), &bizv1.UpdateCompanySettingsRequest{
			TenantId: tenantID.String(),
			Settings: &bizv1.CompanySettings{Name: "Neuer Name", Basiszinssatz: "3.62"},
		})
		require.NoError(t, err)
		assert.Equal(t, "Neuer Name", resp.GetSettings().GetName())
	})
}

// ============================================================================
// Quotes
// ============================================================================

func lineItem() *bizv1.LineItem {
	return &bizv1.LineItem{Description: "Beratung", Quantity: "1", UnitPrice: "100.00", TaxRate: "19.00"}
}

func TestCreateQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{TenantId: tenantID.String(), CreatedBy: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid valid_until", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), ValidUntil: "not-a-date",
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid deal_id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(), DealId: "not-a-uuid",
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no line items maps to InvalidArgument", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{
			TenantId: tenantID.String(), CreatedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		resp, err := srv.CreateQuote(context.Background(), &bizv1.CreateQuoteRequest{
			TenantId:  tenantID.String(),
			CreatedBy: uuid.New().String(),
			Customer:  &bizv1.CustomerSnapshot{Name: "Acme GmbH"},
			LineItems: []*bizv1.LineItem{lineItem()},
		})
		require.NoError(t, err)
		assert.Equal(t, "Acme GmbH", resp.GetQuote().GetCustomer().GetName())
		assert.Equal(t, bizv1.QuoteStatus_QUOTE_DRAFT, resp.GetQuote().GetStatus())
	})
}

func TestGetQuote(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubQuoteRepo()
	q := draftQuote(tenantID)
	repo.quotes[q.ID] = q
	srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

	t.Run("invalid id", func(t *testing.T) {
		_, err := srv.GetQuote(context.Background(), &bizv1.GetQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found maps to NotFound", func(t *testing.T) {
		_, err := srv.GetQuote(context.Background(), &bizv1.GetQuoteRequest{TenantId: tenantID.String(), Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.GetQuote(context.Background(), &bizv1.GetQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, q.ID.String(), resp.GetQuote().GetId())
	})
}

func TestListQuotes(t *testing.T) {
	tenantID := uuid.New()
	repo := newStubQuoteRepo()
	q := draftQuote(tenantID)
	repo.quotes[q.ID] = q
	srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

	t.Run("invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListQuotes(context.Background(), &bizv1.ListQuotesRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid deal_id", func(t *testing.T) {
		_, err := srv.ListQuotes(context.Background(), &bizv1.ListQuotesRequest{TenantId: tenantID.String(), DealId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		resp, err := srv.ListQuotes(context.Background(), &bizv1.ListQuotesRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.GetTotal())
	})
}

func TestUpdateQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		q.Status = models.QuoteStatusSent
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("invalid valid_until", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{
			TenantId: tenantID.String(), Id: q.ID.String(), ValidUntil: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		resp, err := srv.UpdateQuote(context.Background(), &bizv1.UpdateQuoteRequest{
			TenantId: tenantID.String(), Id: q.ID.String(),
			Notes: "aktualisiert",
		})
		require.NoError(t, err)
		assert.Equal(t, "aktualisiert", resp.GetQuote().GetNotes())
	})
}

func TestDeleteQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.DeleteQuote(context.Background(), &bizv1.DeleteQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		q.Status = models.QuoteStatusAccepted
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.DeleteQuote(context.Background(), &bizv1.DeleteQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.DeleteQuote(context.Background(), &bizv1.DeleteQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)
		_, stillThere := repo.quotes[q.ID]
		assert.False(t, stillThere)
	})
}

func TestSendQuote(t *testing.T) {
	tenantID := uuid.New()
	numberSeq := &stubNumberSeqRepo{number: "AN-2026-0001"}

	t.Run("invalid id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, numberSeq)
		_, err := srv.SendQuote(context.Background(), &bizv1.SendQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not draft maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		q.Status = models.QuoteStatusSent
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, numberSeq)

		_, err := srv.SendQuote(context.Background(), &bizv1.SendQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, numberSeq)

		resp, err := srv.SendQuote(context.Background(), &bizv1.SendQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, "AN-2026-0001", resp.GetQuote().GetQuoteNumber())
		assert.Equal(t, bizv1.QuoteStatus_QUOTE_SENT, resp.GetQuote().GetStatus())
	})
}

func TestAcceptQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.AcceptQuote(context.Background(), &bizv1.AcceptQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not sent maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.AcceptQuote(context.Background(), &bizv1.AcceptQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		q.Status = models.QuoteStatusSent
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		resp, err := srv.AcceptQuote(context.Background(), &bizv1.AcceptQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, bizv1.QuoteStatus_QUOTE_ACCEPTED, resp.GetQuote().GetStatus())
	})
}

func TestRejectQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.RejectQuote(context.Background(), &bizv1.RejectQuoteRequest{TenantId: tenantID.String(), Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not sent maps to FailedPrecondition", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		_, err := srv.RejectQuote(context.Background(), &bizv1.RejectQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("happy path", func(t *testing.T) {
		repo := newStubQuoteRepo()
		q := draftQuote(tenantID)
		q.Status = models.QuoteStatusSent
		repo.quotes[q.ID] = q
		srv := newQuoteTestServer(repo, &stubCompanySettingsRepo{}, nil)

		resp, err := srv.RejectQuote(context.Background(), &bizv1.RejectQuoteRequest{TenantId: tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, bizv1.QuoteStatus_QUOTE_REJECTED, resp.GetQuote().GetStatus())
	})
}

func TestExpireQuote(t *testing.T) {
	tenantID := uuid.New()

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.ExpireQuote(context.Background(), &bizv1.ExpireQuoteRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no expired quotes returns empty response", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		resp, err := srv.ExpireQuote(context.Background(), &bizv1.ExpireQuoteRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Nil(t, resp.GetQuote())
	})
}

func TestConvertQuoteToInvoice(t *testing.T) {
	tenantID := uuid.New()
	srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)

	_, err := srv.ConvertQuoteToInvoice(context.Background(), &bizv1.ConvertQuoteToInvoiceRequest{
		TenantId: tenantID.String(), Id: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGenerateQuotePDF_QuoteNotFound(t *testing.T) {
	tenantID := uuid.New()
	srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)

	_, err := srv.GenerateQuotePDF(context.Background(), &bizv1.GenerateQuotePDFRequest{
		TenantId: tenantID.String(), Id: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateQuoteFromDeal(t *testing.T) {
	tenantID := uuid.New()

	t.Run("deal_id required", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuoteFromDeal(context.Background(), &bizv1.CreateQuoteFromDealRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid tenant_id", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuoteFromDeal(context.Background(), &bizv1.CreateQuoteFromDealRequest{
			TenantId: "not-a-uuid", DealId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no CRM client configured maps to Unavailable", func(t *testing.T) {
		srv := newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil)
		_, err := srv.CreateQuoteFromDeal(context.Background(), &bizv1.CreateQuoteFromDealRequest{
			TenantId: tenantID.String(), DealId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.Unavailable)
	})
}
