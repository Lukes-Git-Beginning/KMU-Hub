package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/gobdarchive"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// TestBizGRPCServer_ArchiveInvoiceDocument_UsesTenantCompanySettings_DB wires the
// real invoice and company-settings repositories (as cmd/biz does) on a kmuhub_app
// pool. The archive PDF is the legal copy of the invoice, so it must be rendered
// from the requesting tenant's own master data: the tenant with a complete row gets
// a PDF, a tenant without one is refused instead of getting a letterhead-less PDF
// (or a 500, as when the generator was built once at startup for uuid.Nil).
func TestBizGRPCServer_ArchiveInvoiceDocument_UsesTenantCompanySettings_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	configured, unconfigured := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{configured, unconfigured} {
		testutil.EnsureTenant(t, pool, id, "Gobd Archive Tenant")
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", id) })
	}

	settingsRepo := quote.NewPostgresCompanySettingsRepo(pool)
	settings := pdfReadyCompanySettings()
	settings.TenantID = configured
	require.NoError(t, settingsRepo.Upsert(ctxWithTenant(configured), &settings))

	invRepo := invoice.NewPostgresRepository(pool)
	invSvc := invoice.NewService(invRepo, quote.NewPostgresNumberSequenceRepo(pool), settingsRepo, quote.NewPostgresRepository(pool), pool)
	archiveRepo, archiveStore := newStubGobdRepo(), newStubGobdStore()
	srv := &BizGRPCServer{
		invoiceService:  invSvc,
		companySettings: settingsRepo,
		gobdArchiveSvc:  gobdarchive.NewService(archiveRepo, archiveStore, invSvc),
	}
	f := &bizActorFixture{invRepo: invRepo}

	seedLocked := func(t *testing.T, tenantID uuid.UUID) *models.Invoice {
		t.Helper()
		f.tenantID = tenantID
		inv := f.seedInvoice(t, models.InvoiceStatusSent)
		require.NoError(t, invRepo.SetLock(ctxWithTenant(tenantID), tenantID, inv.ID, time.Now().UTC(), uuid.New()))
		return inv
	}
	archive := func(tenantID uuid.UUID, inv *models.Invoice) error {
		_, err := srv.ArchiveInvoiceDocument(ctxWithTenant(tenantID), &bizv1.ArchiveInvoiceDocumentRequest{
			TenantId: tenantID.String(), InvoiceId: inv.ID.String(), ArchivedBy: uuid.NewString(),
		})
		return err
	}

	t.Run("tenant with company settings archives a real PDF", func(t *testing.T) {
		inv := seedLocked(t, configured)
		require.NoError(t, archive(configured, inv))

		require.Len(t, archiveStore.uploads, 1)
		for _, content := range archiveStore.uploads {
			assert.True(t, len(content) > 4 && string(content[:4]) == "%PDF", "archived bytes must be a PDF")
		}
	})

	t.Run("tenant without company settings is refused and archives nothing", func(t *testing.T) {
		before := len(archiveStore.uploads)
		inv := seedLocked(t, unconfigured)

		requireGRPCCode(t, archive(unconfigured, inv), codes.FailedPrecondition)
		assert.Len(t, archiveStore.uploads, before)
	})
}
