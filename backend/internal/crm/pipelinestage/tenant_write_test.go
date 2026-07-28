package pipelinestage

// wp-crm-meta: closes the write-surface gap the same way wp-crm-core did for
// the core CRM entities. rls_test.go in this package already covers RLS via
// testutil.SeedRow; this file exercises the real Create/Update/Delete
// methods instead.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPipelineStageWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "PipelineStage Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "PipelineStage Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	stage := &models.PipelineStage{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		Name:        "Write-Test-" + uuid.New().String()[:8],
		Color:       "#3d8abf",
		SortOrder:   1,
		Probability: decimal.NewFromInt(10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, stage); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "pipeline_stages", stage.ID, 0)

	if err := repo.Create(ctxOwn, stage); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stage.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "pipeline_stages", stage.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "pipeline_stages", stage.ID, 0)

	// Update carries an explicit tenant_id predicate (stage.TenantID) and
	// treats zero affected rows as not-found.
	foreign := *stage
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrStageNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, stage.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != stage.Name {
		t.Fatalf("a foreign-tenant write reached the pipeline stage: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, stage.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Delete carries the same explicit predicate + not-found-on-zero-rows.
	if err := repo.Delete(ctxOther, stage.ID, tenantOwn); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("Delete (foreign ctx): expected ErrStageNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "pipeline_stages", stage.ID, 1)

	if err := repo.Delete(ctxOwn, stage.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "pipeline_stages", stage.ID, 0)
}

// TestWonLostStages_UniquePerTenantNotGlobally is a regression test for a
// pre-existing bug found while writing the write-surface test above:
// idx_pipeline_stages_won/idx_pipeline_stages_lost (migration 000008) were
// never re-scoped by tenant_id when tenant_id was retrofitted onto
// pipeline_stages (migration 000106) — only the very first tenant to ever
// mark a stage "Won"/"Lost" could do so; every other tenant's Create/Update
// hit a raw unique-violation 500, even though Service.Create/Update already
// re-implements the same uniqueness rule scoped correctly per tenant
// (HasWonStage/HasLostStage). Fixed in migration 000255
// (tenant_id) WHERE is_won/is_lost = TRUE.
func TestWonLostStages_UniquePerTenantNotGlobally(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "PipelineStage Unique Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "PipelineStage Unique Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	for i, tenantCtx := range []struct {
		tenantID uuid.UUID
		ctx      context.Context
	}{{tenantA, ctxA}, {tenantB, ctxB}} {
		stage := &models.PipelineStage{
			ID:          uuid.New(),
			TenantID:    tenantCtx.tenantID,
			Name:        fmt.Sprintf("Won-%d-%s", i, uuid.New().String()[:8]),
			Color:       "#22c55e",
			SortOrder:   1,
			IsWon:       true,
			Probability: decimal.NewFromInt(100),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.Create(tenantCtx.ctx, stage); err != nil {
			t.Fatalf("Create won stage for tenant %d: %v (idx_pipeline_stages_won is not tenant-scoped)", i, err)
		}
		defer testutil.CleanupRow(t, pool, "pipeline_stages", stage.ID)
	}
}
