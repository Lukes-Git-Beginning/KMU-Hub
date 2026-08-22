package gdpr

// Coverage for ContactRetentionHandler (Lauf 10, first handler on the
// retention engine from A10). The two cases that matter: a policy configured
// to delete must skip a contact blocked by a RESTRICT foreign key instead of
// failing, and the same contact must be reachable when the policy is
// configured to anonymize instead.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestContactRetentionHandler_Plan_DeleteSkipsBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "contact-retention-plan-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	deletable := seedRetentionContact(t, pool, tenantID, agentID, "Alt Loeschbar", old)
	defer testutil.CleanupRow(t, pool, "contacts", deletable)

	blocked := seedRetentionContact(t, pool, tenantID, agentID, "Alt Blockiert", old)
	defer testutil.CleanupRow(t, pool, "contacts", blocked)

	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  tenantID,
		"name":       "Retention Test Campaign",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)
	campaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   tenantID,
		"campaign_id": campaignID,
		"contact_id":  blocked,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	// Inside the retention period -- must never show up as due, in either mode.
	recent := seedRetentionContact(t, pool, tenantID, agentID, "Neu", fresh)
	defer testutil.CleanupRow(t, pool, "contacts", recent)

	contactRepo := contact.NewPostgresRepository(pool)
	consentRepo := consent.NewPostgresRepository(pool)
	handler := NewContactRetentionHandler(contactRepo, consentRepo)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	deletePlan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{deletable}, deletePlan.Due,
		"the blocked contact must not be Due when the policy deletes")
	require.Len(t, deletePlan.Skipped, 1)
	assert.Equal(t, blocked, deletePlan.Skipped[0].RecordID)
	assert.Contains(t, deletePlan.Skipped[0].Reason, "call campaign history")

	anonymizePlan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{deletable, blocked}, anonymizePlan.Due,
		"the blocked contact must be reachable when the policy anonymizes")
	assert.Empty(t, anonymizePlan.Skipped)
}

func TestContactRetentionHandler_Apply(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Retention Apply Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	agentID := seedExportUser(t, pool, tenantID, "contact-retention-apply-agent")
	defer testutil.CleanupRow(t, pool, "users", agentID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	toDelete := seedRetentionContact(t, pool, tenantID, agentID, "Wird Geloescht", old)
	toAnonymize := seedRetentionContact(t, pool, tenantID, agentID, "Wird Anonymisiert", old)
	defer testutil.CleanupRow(t, pool, "contacts", toAnonymize)

	contactRepo := contact.NewPostgresRepository(pool)
	consentRepo := consent.NewPostgresRepository(pool)
	handler := NewContactRetentionHandler(contactRepo, consentRepo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{toDelete}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM contacts WHERE id = $1`, toDelete).Scan(&count))
	assert.Zero(t, count, "deleted contact must be gone")

	affected, err = handler.Apply(ctx, tenantID, []uuid.UUID{toAnonymize}, models.RetentionActionAnonymize, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var firstName, lastName string
	var email *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT first_name, last_name, email FROM contacts WHERE id = $1`, toAnonymize,
	).Scan(&firstName, &lastName, &email))
	assert.Equal(t, consent.AnonymizedFirstName, firstName)
	assert.Equal(t, consent.AnonymizedLastName, lastName)
	assert.Nil(t, email)

	// Idempotency: AnonymizeContact bumped updated_at to NOW(), so a Plan
	// against the same cutoff must not offer the contact again immediately.
	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, toAnonymize)
}

func TestContactRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewContactRetentionHandler(nil, nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, "retain", "")
	require.Error(t, err)
}

func TestContactRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewContactRetentionHandler(nil, nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "contacts", handler.ResourceType())
	assert.Equal(t, "contacts", handler.Table())
}

func seedRetentionContact(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, lastName string, updatedAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Retention",
		"last_name":  lastName,
		"email":      fmt.Sprintf("retention-%s@contacts.invalid", uuid.New()),
		"created_by": createdBy,
		"created_at": updatedAt,
		"updated_at": updatedAt,
	})
}
