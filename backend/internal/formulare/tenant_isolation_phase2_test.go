package formulare

// Cross-tenant isolation tests for formulare tables (Migration 000122).
// form_schemas, form_submissions, form_webhooks, form_webhook_deliveries all have
// tenant_id NOT NULL and RLS was enabled in mig 000122.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Formulare(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a form_schema for tenant A — basis for form_webhooks and form_submissions.
	schemaID := testutil.SeedRow(t, pool, "form_schemas", map[string]any{
		"tenant_id": testutil.TenantA,
		"title":     "Contact Form " + uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaID)

	// Seed a form_submission for tenant A.
	submID := testutil.SeedRow(t, pool, "form_submissions", map[string]any{
		"tenant_id":      testutil.TenantA,
		"form_schema_id": schemaID,
		"answers":        `{"name":"Alice"}`,
	})
	defer testutil.CleanupRow(t, pool, "form_submissions", submID)

	// Seed a form_webhook for tenant A.
	webhookID := testutil.SeedRow(t, pool, "form_webhooks", map[string]any{
		"tenant_id":      testutil.TenantA,
		"form_schema_id": schemaID,
		"url":            "https://example.com/hook",
	})
	defer testutil.CleanupRow(t, pool, "form_webhooks", webhookID)

	// Seed a form_webhook_delivery for tenant A.
	deliveryID := testutil.SeedRow(t, pool, "form_webhook_deliveries", map[string]any{
		"tenant_id":     testutil.TenantA,
		"webhook_id":    webhookID,
		"submission_id": submID,
		"payload":       `{"event":"submission.created"}`,
	})
	defer testutil.CleanupRow(t, pool, "form_webhook_deliveries", deliveryID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"form_schemas", "form_schemas", schemaID},
		{"form_submissions", "form_submissions", submID},
		{"form_webhooks", "form_webhooks", webhookID},
		{"form_webhook_deliveries", "form_webhook_deliveries", deliveryID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
