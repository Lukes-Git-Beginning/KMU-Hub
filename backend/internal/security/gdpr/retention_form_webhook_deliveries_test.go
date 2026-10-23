package gdpr

// Coverage for FormWebhookDeliveryRetentionHandler
// (fix-form-webhook-deliveries-unbounded-retention). What matters: only final
// deliveries (delivered/dead) past the cutoff are Due, pending/failed rows
// stay untouched however old, Apply never deletes a non-final row or another
// tenant's row, and a second run is a no-op.

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

func TestFormWebhookDeliveryRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewFormWebhookDeliveryRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "form_webhook_deliveries", handler.ResourceType())
	assert.Equal(t, "form_webhook_deliveries", handler.Table())
}

func TestFormWebhookDeliveryRetentionHandler_PlanAndApplyOnlyTouchFinalDeliveriesOfTheTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Form Webhook Delivery Retention Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	webhookID, submissionID, cleanupParents := seedFormWebhookParents(t, pool, tenantID)
	defer cleanupParents()

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Form Webhook Delivery Retention Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherWebhookID, otherSubmissionID, cleanupOtherParents := seedFormWebhookParents(t, pool, otherTenantID)
	defer cleanupOtherParents()

	old := time.Now().UTC().AddDate(0, 0, -200)
	fresh := time.Now().UTC().AddDate(0, 0, -3)

	oldDelivered := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "delivered", old)
	oldDead := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "dead", old)
	oldPending := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "pending", old)
	oldFailed := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "failed", old)
	freshDelivered := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "delivered", fresh)
	otherOld := seedFormWebhookDelivery(t, pool, otherTenantID, otherWebhookID, otherSubmissionID, "delivered", old)

	for _, id := range []uuid.UUID{oldDelivered, oldDead, oldPending, oldFailed, freshDelivered, otherOld} {
		defer testutil.CleanupRow(t, pool, "form_webhook_deliveries", id)
	}

	handler := NewFormWebhookDeliveryRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{oldDelivered, oldDead}, plan.Due,
		"only final deliveries past the cutoff are due; pending/failed are still owed, fresh ones are within retention")

	// Apply is given every id on purpose: the status filter in Apply must
	// hold even if Plan's answer went stale, and the tenant filter must hold
	// even if a foreign id slips in.
	all := []uuid.UUID{oldDelivered, oldDead, oldPending, oldFailed, freshDelivered, otherOld}
	affected, err := handler.Apply(ctx, tenantID, all, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 3, affected, "old delivered + old dead + the fresh delivered passed in explicitly; pending/failed/foreign must survive")

	sys := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sys, "form_webhook_deliveries", oldDelivered, 0)
	testutil.AssertRowCount(t, pool, sys, "form_webhook_deliveries", oldDead, 0)
	testutil.AssertRowCount(t, pool, sys, "form_webhook_deliveries", oldPending, 1) // pending is still owed
	testutil.AssertRowCount(t, pool, sys, "form_webhook_deliveries", oldFailed, 1) // failed is still in the retry loop
	testutil.AssertRowCount(t, pool, sys, "form_webhook_deliveries", otherOld, 1) // foreign tenant

	again, err := handler.Apply(ctx, tenantID, []uuid.UUID{oldDelivered, oldDead}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 0, again, "a second run over the same ids is a no-op")

	_, err = handler.Apply(ctx, tenantID, []uuid.UUID{oldDelivered}, models.RetentionActionAnonymize, "x")
	require.Error(t, err, "anonymize is unsupported")
}

func seedFormWebhookParents(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) (webhookID, submissionID uuid.UUID, cleanup func()) {
	t.Helper()
	schemaID := testutil.SeedRow(t, pool, "form_schemas", map[string]any{
		"tenant_id": tenantID,
		"title":     "Retention Delivery Schema",
	})
	webhookID = testutil.SeedRow(t, pool, "form_webhooks", map[string]any{
		"tenant_id":      tenantID,
		"form_schema_id": schemaID,
		"url":            "https://example.invalid/hook",
	})
	submissionID = seedFormSubmission(t, pool, tenantID, time.Now().UTC())
	return webhookID, submissionID, func() {
		testutil.CleanupRow(t, pool, "form_submissions", submissionID)
		testutil.CleanupRow(t, pool, "form_webhooks", webhookID)
		testutil.CleanupRow(t, pool, "form_schemas", schemaID)
	}
}

func seedFormWebhookDelivery(t *testing.T, pool *pgxpool.Pool, tenantID, webhookID, submissionID uuid.UUID, status string, createdAt time.Time) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "form_webhook_deliveries", map[string]any{
		"tenant_id":     tenantID,
		"webhook_id":    webhookID,
		"submission_id": submissionID,
		"payload":       `{"answers":{"email":"person@example.com"}}`,
		"status":        status,
		"created_at":    createdAt,
	})
	return id
}

// fix-form-submission-anonymize-leaves-webhook-delivery-payload: anonymizing
// a submission must not leave its answers behind in the delivery payload.
// Final rows lose the payload only; owed rows (pending/failed) are closed as
// dead instead of being delivered empty; a foreign tenant's delivery of the
// same-shaped submission is untouched.
func TestFormSubmissionRetentionHandler_ApplyAnonymizeScrubsWebhookDeliveryPayloads(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Form Anonymize Delivery Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	webhookID, submissionID, cleanupParents := seedFormWebhookParents(t, pool, tenantID)
	defer cleanupParents()

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Form Anonymize Delivery Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherWebhookID, otherSubmissionID, cleanupOtherParents := seedFormWebhookParents(t, pool, otherTenantID)
	defer cleanupOtherParents()

	now := time.Now().UTC()
	delivered := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "delivered", now)
	dead := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "dead", now)
	pending := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "pending", now)
	failed := seedFormWebhookDelivery(t, pool, tenantID, webhookID, submissionID, "failed", now)
	foreign := seedFormWebhookDelivery(t, pool, otherTenantID, otherWebhookID, otherSubmissionID, "pending", now)
	for _, id := range []uuid.UUID{delivered, dead, pending, failed, foreign} {
		defer testutil.CleanupRow(t, pool, "form_webhook_deliveries", id)
	}

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewFormSubmissionRetentionHandler(pool)
	// The foreign submission id is passed in on purpose: the tenant filter must hold.
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{submissionID, otherSubmissionID}, models.RetentionActionAnonymize, "Geloeschte Einreichung 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	sys := testutil.WithSystemCtx(context.Background())
	read := func(id uuid.UUID) (payload, status string, lastError *string) {
		require.NoError(t, pool.QueryRow(sys,
			`SELECT payload::text, status::text, last_error FROM form_webhook_deliveries WHERE id = $1`, id,
		).Scan(&payload, &status, &lastError))
		return payload, status, lastError
	}

	for id, wantStatus := range map[uuid.UUID]string{delivered: "delivered", dead: "dead", pending: "dead", failed: "dead"} {
		payload, status, lastError := read(id)
		assert.JSONEq(t, `{}`, payload, "no delivery of the anonymized submission may keep the answers")
		assert.Equal(t, wantStatus, status)
		if wantStatus == "dead" && id != dead {
			require.NotNil(t, lastError)
			assert.Contains(t, *lastError, "anonymized")
		}
	}

	payload, status, _ := read(foreign)
	assert.Contains(t, payload, "person@example.com", "another tenant's delivery must stay untouched")
	assert.Equal(t, "pending", status)
}
