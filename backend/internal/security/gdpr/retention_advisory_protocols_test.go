package gdpr

// Coverage for AdvisoryProtocolRetentionHandler
// (harden-advisory-protocols-retention-guard). The cases that matter: the
// handler never supports an action (delete or anonymize), the engine's run
// report names the legal basis instead of the generic "unsupported" text,
// Plan counts only finalized protocols past cutoff (a draft never starts
// the clock) and is tenant-scoped, and Apply never touches a row.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAdvisoryProtocolRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewAdvisoryProtocolRetentionHandler(nil)
	assert.False(t, handler.SupportsAction(models.RetentionActionDelete), "advisory_protocols carries a 10-year statutory retention duty, deletion is never automatic")
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize), "an anonymized advisory protocol would no longer satisfy the §18a FinVermV evidentiary purpose")
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "advisory_protocols", handler.ResourceType())
	assert.Equal(t, "advisory_protocols", handler.Table())
	assert.Contains(t, handler.DateColumn(), "handed_over_at")
}

func TestAdvisoryProtocolRetentionHandler_UnsupportedReasonNamesLegalBasisAndTerm(t *testing.T) {
	t.Parallel()

	handler := NewAdvisoryProtocolRetentionHandler(nil)
	reason := handler.UnsupportedReason(models.RetentionActionDelete)
	assert.Contains(t, reason, "FinVermV", "reason must name the statute the retention duty rests on")
	assert.Contains(t, reason, "10-Jahres", "reason must name the retention term, not just cite the law")
}

func TestAdvisoryProtocolRetentionHandler_Run_ReportsUnsupportedWithLegalReasonNotUnmapped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Advisory Protocol Retention Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "advisory-retention-run-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := seedAdvisoryContact(t, pool, tenantID, userID, "run-contact")
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	old := time.Now().UTC().AddDate(-11, 0, 0)
	protocolID := seedAdvisoryProtocol(t, pool, tenantID, contactID, &old)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", protocolID)

	policyID := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantID,
		"resource_type":  "advisory_protocols",
		"retention_days": 3650,
		"action":         models.RetentionActionDelete,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", policyID)

	engine := NewRetentionEngine(pool, NewPostgresRepository(pool), NewRetentionRegistry(
		NewAdvisoryProtocolRetentionHandler(pool),
	))

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	result, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_runs", result.RunID)

	item := itemFor(t, result, "advisory_protocols")
	assert.Equal(t, RetentionItemUnsupported, item.Status, "must be unsupported, not unmapped — a handler IS registered")
	assert.Contains(t, item.Message, "FinVermV")
	assert.Contains(t, item.Message, "10-Jahres")
	assert.Zero(t, item.Affected, "enforce mode must not touch advisory_protocols")

	var stillThere int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM advisory_protocols WHERE id = $1`, protocolID).Scan(&stillThere))
	assert.Equal(t, 1, stillThere, "the protocol must survive an enforce run untouched")
}

func TestAdvisoryProtocolRetentionHandler_PlanCountsOnlyFinalizedPastCutoffAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Advisory Protocol Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "advisory-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := seedAdvisoryContact(t, pool, tenantID, userID, "plan-contact")
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Advisory Protocol Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "advisory-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)
	otherContactID := seedAdvisoryContact(t, pool, otherTenantID, otherUserID, "plan-other-contact")
	defer testutil.CleanupRow(t, pool, "contacts", otherContactID)

	old := time.Now().UTC().AddDate(-11, 0, 0)
	fresh := time.Now().UTC().AddDate(-1, 0, 0)

	finalizedOld := seedAdvisoryProtocol(t, pool, tenantID, contactID, &old)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", finalizedOld)

	finalizedFresh := seedAdvisoryProtocol(t, pool, tenantID, contactID, &fresh)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", finalizedFresh)

	draftOld := seedAdvisoryProtocol(t, pool, tenantID, contactID, nil)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", draftOld)

	otherFinalizedOld := seedAdvisoryProtocol(t, pool, otherTenantID, otherContactID, &old)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", otherFinalizedOld)

	handler := NewAdvisoryProtocolRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(-10, 0, 0)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{finalizedOld}, plan.Due)
	assert.NotContains(t, plan.Due, finalizedFresh, "a finalized protocol still within the 10-year term must not be due")
	assert.NotContains(t, plan.Due, draftOld, "a draft with no handed_over_at has not started the retention clock")
	assert.NotContains(t, plan.Due, otherFinalizedOld, "another tenant's protocol must never leak into this tenant's plan")
}

func TestAdvisoryProtocolRetentionHandler_ApplyAlwaysErrorsAndTouchesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Advisory Protocol Retention Apply Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "advisory-retention-apply-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := seedAdvisoryContact(t, pool, tenantID, userID, "apply-contact")
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	old := time.Now().UTC().AddDate(-11, 0, 0)
	protocolID := seedAdvisoryProtocol(t, pool, tenantID, contactID, &old)
	defer testutil.CleanupRow(t, pool, "advisory_protocols", protocolID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewAdvisoryProtocolRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{protocolID}, models.RetentionActionDelete, "")
	require.Error(t, err)
	assert.Zero(t, affected)

	var stillThere int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM advisory_protocols WHERE id = $1`, protocolID).Scan(&stillThere))
	assert.Equal(t, 1, stillThere, "Apply must never delete an advisory protocol")
}

// seedAdvisoryContact seeds the minimal contact an advisory_protocols row
// requires via its RESTRICT foreign key. created_by references users(id),
// unlike advisory_protocols.created_by which has no such constraint.
func seedAdvisoryContact(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, lastName string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Advisory",
		"last_name":  lastName,
		"email":      fmt.Sprintf("advisory-retention-%s@contacts.invalid", uuid.New()),
		"created_by": createdBy,
	})
}

// seedAdvisoryProtocol seeds a minimal advisory_protocols row. handedOverAt
// nil means a still-open draft; a non-nil pointer finalizes it.
func seedAdvisoryProtocol(t *testing.T, pool *pgxpool.Pool, tenantID, contactID uuid.UUID, handedOverAt *time.Time) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":  tenantID,
		"contact_id": contactID,
		"created_by": uuid.New(),
	}
	if handedOverAt != nil {
		cols["status"] = "finalized"
		cols["handed_over_at"] = *handedOverAt
	}
	return testutil.SeedRow(t, pool, "advisory_protocols", cols)
}
