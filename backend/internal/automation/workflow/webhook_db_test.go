package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestGetByIDUnscoped_BypassesRLS verifies the actual reason GetByIDUnscoped
// exists: under sysctx it must resolve an automation with NO tenant set on
// the caller's context (the webhook URL carries no JWT), while the ordinary
// tenant-scoped GetByID must still reject a mismatched tenant. Each test run
// seeds its own throwaway tenant (not the shared testutil.TenantA/B) per the
// lesson from run 1's CI failure on a per-tenant-unique-index collision.
func TestGetByIDUnscoped_BypassesRLS(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ownTenant := uuid.New()
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, ownTenant, "GetByIDUnscoped Own Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "GetByIDUnscoped Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     ownTenant,
		"email":         fmt.Sprintf("webhook-test-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":      ownTenant,
		"name":           "Webhook Unscoped Test",
		"description":    "",
		"owner_id":       userID,
		"trigger_type":   webhookReceivedTriggerType,
		"trigger_config": `{"secret":"db-test-secret"}`,
		"is_active":      true,
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	repo := NewPostgresRepository(pool)

	// No tenant on the context at all -- exactly the situation an inbound
	// webhook request is in. Only sysctx admits the row.
	unscopedCtx := testutil.WithSystemCtx(context.Background())
	found, err := repo.GetByIDUnscoped(unscopedCtx, autoID)
	if err != nil {
		t.Fatalf("GetByIDUnscoped under sysctx: %v", err)
	}
	if found.ID != autoID || found.TenantID != ownTenant {
		t.Fatalf("GetByIDUnscoped returned wrong row: %+v", found)
	}

	// The ordinary tenant-scoped GetByID must still refuse a foreign tenant.
	if _, err := repo.GetByID(testutil.WithTenantCtx(context.Background(), otherTenant), autoID, otherTenant); err == nil {
		t.Fatal("GetByID must not resolve an automation belonging to a different tenant")
	}
}

// TestTriggerWebhook_RealRepositories runs the full TriggerWebhook path
// against the real Postgres-backed workflow and idempotency repositories
// (not mocks), verifying signature enforcement and cross-call deduplication
// survive an actual round trip through RLS.
func TestTriggerWebhook_RealRepositories(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "TriggerWebhook Real Repo Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("webhook-real-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":      tenantID,
		"name":           "Webhook Real Repo Test",
		"description":    "",
		"owner_id":       userID,
		"trigger_type":   webhookReceivedTriggerType,
		"trigger_config": `{"secret":"real-repo-secret"}`,
		"actions":        `[]`,
		"is_active":      true,
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	repo := NewPostgresRepository(pool)
	idemRepo := idempotency.NewPostgresRepository(pool)
	executor := newFakeExecutor()
	svc := newTestServiceWithWebhookDeps(nil, nil, nil, idemRepo, executor)
	svc.repo = repo // swap the mock in newTestServiceWithWebhookDeps for the real repo

	body := []byte(`{"real":true}`)
	sig := sign(body, "real-repo-secret")

	result, err := svc.TriggerWebhook(context.Background(), autoID, body, sig, "real-repo-delivery-1")
	if err != nil {
		t.Fatalf("TriggerWebhook: %v", err)
	}
	if result.Duplicate {
		t.Fatal("first delivery must not be reported as a duplicate")
	}
	waitExecuted(t, executor)

	// Same delivery key resent -- must dedupe via the real idempotency table.
	result2, err := svc.TriggerWebhook(context.Background(), autoID, body, sig, "real-repo-delivery-1")
	if err != nil {
		t.Fatalf("TriggerWebhook (resend): %v", err)
	}
	if !result2.Duplicate {
		t.Fatal("resent delivery must be reported as a duplicate")
	}
	if got := executor.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 execution across both calls, got %d", got)
	}

	// Wrong signature must still be rejected even with real repos wired.
	if _, err := svc.TriggerWebhook(context.Background(), autoID, body, "sha256=deadbeef", "real-repo-delivery-2"); err == nil {
		t.Fatal("wrong signature must be rejected")
	}
}
