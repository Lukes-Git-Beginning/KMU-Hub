package gdpr

// Coverage for InvitationRetentionHandler (Lauf 12, ninth handler on the
// retention engine from A10). The cases that matter: a pending invitation is
// only Due once expires_at has passed (not created_at), an accepted
// invitation is only Due once accepted_at has passed, a fresh row in either
// state is never Due, the handler only supports delete, delete is
// idempotent, and a second tenant's expired invitation never appears in the
// first tenant's plan.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestInvitationRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewInvitationRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize), "anonymizing an invitation is worthless once its token has no reason to be kept")
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "invitations", handler.ResourceType())
	assert.Equal(t, "invitations", handler.Table())
}

func TestInvitationRetentionHandler_PlanMatchesBothAgeStatesPastCutoffAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invitation Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "invitation-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Invitation Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "invitation-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	pendingExpiredOld := seedInvitation(t, pool, tenantID, userID, "pending-expired-old", old, nil)
	defer testutil.CleanupRow(t, pool, "invitations", pendingExpiredOld)

	pendingExpiredFresh := seedInvitation(t, pool, tenantID, userID, "pending-expired-fresh", fresh, nil)
	defer testutil.CleanupRow(t, pool, "invitations", pendingExpiredFresh)

	acceptedOld := seedInvitation(t, pool, tenantID, userID, "accepted-old", old, &old)
	defer testutil.CleanupRow(t, pool, "invitations", acceptedOld)

	acceptedFresh := seedInvitation(t, pool, tenantID, userID, "accepted-fresh", old, &fresh)
	defer testutil.CleanupRow(t, pool, "invitations", acceptedFresh)

	// Another tenant's old pending invitation must never leak into this
	// tenant's plan.
	otherPendingExpiredOld := seedInvitation(t, pool, otherTenantID, otherUserID, "other-pending-expired-old", old, nil)
	defer testutil.CleanupRow(t, pool, "invitations", otherPendingExpiredOld)

	handler := NewInvitationRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{pendingExpiredOld, acceptedOld}, plan.Due)
	assert.NotContains(t, plan.Due, pendingExpiredFresh, "a pending invitation whose expires_at is still within the cutoff must not be due")
	assert.NotContains(t, plan.Due, acceptedFresh, "an accepted invitation whose accepted_at is still within the cutoff must not be due")
}

func TestInvitationRetentionHandler_ApplyDeleteIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invitation Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "invitation-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	invitationID := seedInvitation(t, pool, tenantID, userID, "apply-delete", old, nil)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewInvitationRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{invitationID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM invitations WHERE id = $1`, invitationID).Scan(&count))
	assert.Zero(t, count, "deleted invitation must be gone")

	// A second run over the same id finds nothing left to delete.
	againAffected, err := handler.Apply(ctx, tenantID, []uuid.UUID{invitationID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, againAffected, "second run must not report affected rows for an already-deleted invitation")

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, invitationID)
}

func TestInvitationRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewInvitationRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "")
	require.Error(t, err)
}

// seedInvitation seeds a minimal invitation. acceptedAt nil means still
// pending; a non-nil pointer sets accepted_at.
func seedInvitation(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, emailPrefix string, expiresAt time.Time, acceptedAt *time.Time) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":  tenantID,
		"email":      emailPrefix + "@retention-test.example",
		"role":       "member",
		"token_hash": uuid.New().String(),
		"created_by": createdBy,
		"expires_at": expiresAt,
	}
	if acceptedAt != nil {
		cols["accepted_at"] = *acceptedAt
	}
	return testutil.SeedRow(t, pool, "invitations", cols)
}
