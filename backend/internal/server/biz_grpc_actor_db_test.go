package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// bizActorFixture wires the real finance services (as cmd/biz does) onto a
// kmuhub_app pool, so the actor that reaches the persisted rows is the one the
// RPC resolved from the caller context.
type bizActorFixture struct {
	srv      *BizGRPCServer
	invRepo  *invoice.PostgresRepository
	cnRepo   *creditnote.PostgresRepository
	quoteRep *quote.PostgresRepository
	tenantID uuid.UUID
}

func newBizActorFixture(t *testing.T, pool *pgxpool.Pool) *bizActorFixture {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Biz Actor Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	quoteRepo := quote.NewPostgresRepository(pool)
	invRepo := invoice.NewPostgresRepository(pool)
	cnRepo := creditnote.NewPostgresRepository(pool)
	seqRepo := quote.NewPostgresNumberSequenceRepo(pool)
	settings := quote.NewPostgresCompanySettingsRepo(pool)

	invSvc := invoice.NewService(invRepo, seqRepo, settings, quoteRepo, pool)
	cnSvc := creditnote.NewService(cnRepo, invRepo, seqRepo, pool)
	cnSvc.SetInvoiceStatusUpdater(invRepo)
	invSvc.SetStornoCreator(cnSvc)

	return &bizActorFixture{
		srv:      &BizGRPCServer{invoiceService: invSvc, creditNoteService: cnSvc},
		invRepo:  invRepo,
		cnRepo:   cnRepo,
		quoteRep: quoteRepo,
		tenantID: tenantID,
	}
}

func (f *bizActorFixture) seedInvoice(t *testing.T, status string) *models.Invoice {
	t.Helper()
	raw, err := json.Marshal([]models.LineItem{{Description: "Beratung", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: decimal.NewFromInt(19)}})
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	inv := &models.Invoice{
		ID: uuid.New(), TenantID: f.tenantID, Status: status, CustomerName: "Actor GmbH",
		TaxMode: models.TaxModeStandard, LineItems: raw,
		Subtotal: decimal.NewFromInt(100), TotalTax: decimal.NewFromInt(19), GrossTotal: decimal.NewFromInt(119),
		InvoiceDate: now, DueDate: now.Add(14 * 24 * time.Hour), PaymentTerms: "14 days",
		CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if status != models.InvoiceStatusDraft {
		inv.InvoiceNumber = "RE-ACT-" + uuid.NewString()[:8]
	}
	require.NoError(t, f.invRepo.Create(ctxWithTenant(f.tenantID), inv))
	return inv
}

func TestBizGRPCServer_FinanceMutationsAttributeTheCaller_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	f := newBizActorFixture(t, pool)
	actor := uuid.New()
	callerCtx := ctxWithTenantAndUser(f.tenantID, actor)
	tenantCtx := ctxWithTenant(f.tenantID)

	t.Run("cancelling a sent invoice stamps the storno credit note with the caller", func(t *testing.T) {
		inv := f.seedInvoice(t, models.InvoiceStatusSent)

		_, err := f.srv.CancelInvoice(tenantCtx, &bizv1.CancelInvoiceRequest{TenantId: f.tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)

		_, err = f.srv.CancelInvoice(callerCtx, &bizv1.CancelInvoiceRequest{TenantId: f.tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)

		notes, err := f.cnRepo.GetByInvoiceID(tenantCtx, f.tenantID, inv.ID)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		assert.Equal(t, actor, notes[0].CreatedBy)
	})

	t.Run("sending an invoice records the caller in the GoBD snapshot", func(t *testing.T) {
		inv := f.seedInvoice(t, models.InvoiceStatusDraft)

		_, err := f.srv.SendInvoice(tenantCtx, &bizv1.SendInvoiceRequest{TenantId: f.tenantID.String(), Id: inv.ID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)

		_, err = f.srv.SendInvoice(callerCtx, &bizv1.SendInvoiceRequest{TenantId: f.tenantID.String(), Id: inv.ID.String()})
		require.NoError(t, err)

		sent, err := f.invRepo.GetByID(tenantCtx, f.tenantID, inv.ID)
		require.NoError(t, err)
		var snap map[string]any
		require.NoError(t, json.Unmarshal(sent.SnapshotData, &snap))
		assert.Equal(t, actor.String(), snap["sent_by"])
	})

	t.Run("converting a quote stamps the invoice with the caller", func(t *testing.T) {
		raw, err := json.Marshal([]models.LineItem{{Description: "Beratung", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: decimal.NewFromInt(19)}})
		require.NoError(t, err)
		now := time.Now().UTC().Truncate(time.Second)
		q := &models.Quote{
			ID: uuid.New(), TenantID: f.tenantID, QuoteNumber: "AN-ACT-" + uuid.NewString()[:8],
			Status: models.QuoteStatusAccepted, CustomerName: "Actor GmbH", TaxMode: models.TaxModeStandard,
			LineItems: raw, Subtotal: decimal.NewFromInt(100), TotalTax: decimal.NewFromInt(19), GrossTotal: decimal.NewFromInt(119),
			CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, f.quoteRep.Create(tenantCtx, q))

		_, err = f.srv.ConvertQuoteToInvoice(tenantCtx, &bizv1.ConvertQuoteToInvoiceRequest{TenantId: f.tenantID.String(), Id: q.ID.String()})
		requireGRPCCode(t, err, codes.Unauthenticated)

		resp, err := f.srv.ConvertQuoteToInvoice(callerCtx, &bizv1.ConvertQuoteToInvoiceRequest{TenantId: f.tenantID.String(), Id: q.ID.String()})
		require.NoError(t, err)

		created, err := f.invRepo.GetByID(tenantCtx, f.tenantID, uuid.MustParse(resp.GetInvoice().GetId()))
		require.NoError(t, err)
		assert.Equal(t, actor, created.CreatedBy)
	})
}

