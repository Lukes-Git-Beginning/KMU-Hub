package workflow

// Real-repo tenant isolation tests for automations and automation_executions.
//
// tenant_isolation_phase2_test.go already proves the DB-level RLS boundary, but
// it seeds rows via testutil.SeedRow (hand-written SQL with tenant_id set
// directly) rather than through PostgresRepository — the same pattern that hid
// dead writes in caldav/formulare (see wp-branchen-module-3, iteration 58).
// These tests go through the real repo methods instead, and additionally cover
// the two reads that were found to skip the tenant_id predicate entirely
// (GetExecution, ListExecutions) and the one write that did too
// (UpdateExecution) — all three relied solely on RLS, with no explicit
// predicate in the Go query.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAutomationWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Dedicated tenants rather than the shared testutil.TenantA/TenantB: this
	// package's other DB test (TestTenantIsolation_Automations) also seeds
	// against TenantA/TenantB and both tests run with t.Parallel().
	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Automation Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Automation Write Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantWrite,
		"email":         fmt.Sprintf("auto-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)

	auto := &models.Automation{
		TenantID:      tenantWrite,
		Name:          "Tenant Write Test Automation",
		Scope:         models.AutomationScopePersonal,
		OwnerID:       userID,
		TriggerType:   "contact.created",
		TriggerConfig: []byte(`{}`),
		Conditions:    []byte(`{}`),
		Actions:       []byte(`[]`),
		IsActive:      true,
	}
	require.NoError(t, repo.Create(ctxA, auto))
	defer testutil.CleanupRow(t, pool, "automations", auto.ID)

	t.Run("automations", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "automations", auto.ID, 1)
		testutil.AssertRowCount(t, pool, ctxB, "automations", auto.ID, 0)
	})

	// Update must also require the caller's tenant to match: an update
	// carrying the wrong tenant_id must not touch the row.
	auto.Name = "Renamed by Tenant A"
	require.NoError(t, repo.Update(ctxA, auto))

	wrongTenantAuto := &models.Automation{
		ID:          auto.ID,
		TenantID:    tenantOther,
		Name:        "Should Not Apply",
		OwnerID:     userID,
		TriggerType: "contact.created",
	}
	err := repo.Update(ctxB, wrongTenantAuto)
	assert.ErrorIs(t, err, ErrAutomationNotFound, "update with a foreign tenant_id must not find the row")

	got, err := repo.GetByID(ctxA, auto.ID, tenantWrite)
	require.NoError(t, err)
	assert.Equal(t, "Renamed by Tenant A", got.Name, "the foreign-tenant update must not have applied")

	exec := &models.AutomationExecution{
		TenantID:        tenantWrite,
		AutomationID:    auto.ID,
		ChainID:         uuid.New(),
		TriggerEvent:    []byte(`{"type":"contact.created"}`),
		ConditionResult: true,
		Status:          models.ExecutionStatusRunning,
		Steps:           []byte(`[]`),
	}
	require.NoError(t, repo.CreateExecution(ctxA, exec))
	defer testutil.CleanupRow(t, pool, "automation_executions", exec.ID)

	t.Run("automation_executions", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "automation_executions", exec.ID, 1)
		testutil.AssertRowCount(t, pool, ctxB, "automation_executions", exec.ID, 0)
	})

	// GetExecution must fail closed on a foreign tenant_id, not just rely on
	// the RLS session GUC — the Go-level query used to have no predicate at
	// all (WHERE id = $1), reachable via the user-facing GetExecution RPC.
	t.Run("GetExecution_foreign_tenant_not_found", func(t *testing.T) {
		_, err := repo.GetExecution(ctxB, exec.ID, tenantOther)
		assert.ErrorIs(t, err, ErrExecutionNotFound)

		got, err := repo.GetExecution(ctxA, exec.ID, tenantWrite)
		require.NoError(t, err)
		assert.Equal(t, exec.ID, got.ID)
	})

	// UpdateExecution used to run as `WHERE id = $1` with no tenant predicate
	// at all — a foreign tenant_id on the struct must not be able to mutate
	// another tenant's execution row.
	t.Run("UpdateExecution_foreign_tenant_rejected", func(t *testing.T) {
		foreign := *exec
		foreign.TenantID = tenantOther
		foreign.Status = models.ExecutionStatusFailed
		err := repo.UpdateExecution(ctxB, &foreign)
		assert.ErrorIs(t, err, ErrExecutionNotFound)

		still, err := repo.GetExecution(ctxA, exec.ID, tenantWrite)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStatusRunning, still.Status, "foreign-tenant update must not have applied")

		exec.Status = models.ExecutionStatusCompleted
		require.NoError(t, repo.UpdateExecution(ctxA, exec))
		updated, err := repo.GetExecution(ctxA, exec.ID, tenantWrite)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStatusCompleted, updated.Status)
	})

	// ListExecutions used to build its query with no tenant_id condition at
	// all — a caller passing another tenant's ID must see nothing, even
	// though automation_id correctly identifies the row.
	t.Run("ListExecutions_scoped_to_tenant", func(t *testing.T) {
		execs, total, err := repo.ListExecutions(ctxA, ExecutionFilter{TenantID: tenantWrite, AutomationID: &auto.ID})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, execs, 1)
		assert.Equal(t, exec.ID, execs[0].ID)

		execs, total, err = repo.ListExecutions(ctxB, ExecutionFilter{TenantID: tenantOther, AutomationID: &auto.ID})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, execs)
	})
}
