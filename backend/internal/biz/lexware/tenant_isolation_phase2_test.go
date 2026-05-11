package lexware

// Cross-tenant isolation DB tests for lexware_entity_mappings, lexware_field_mappings,
// lexware_sync_log, lexware_webhook_subscriptions (Migration 000122 + 000115).
//
// All tables have tenant_id NOT NULL (mig 000115) and RLS in mig 000122.
// config_id FK → integration_configs(id); we seed a minimal integration_configs row.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Lexware_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("lw-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// integration_configs.
	cfgID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgID)

	// lexware_entity_mappings.
	mapID := testutil.SeedRow(t, pool, "lexware_entity_mappings", map[string]any{
		"tenant_id":    testutil.TenantA,
		"config_id":    cfgID,
		"entity_type":  "contact",
		"kmuhub_id":    uuid.New(),
		"lexware_id":   uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "lexware_entity_mappings", mapID)

	// lexware_field_mappings (UNIQUE on config_id + entity_type).
	fmID := testutil.SeedRow(t, pool, "lexware_field_mappings", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"entity_type": "invoice",
	})
	defer testutil.CleanupRow(t, pool, "lexware_field_mappings", fmID)

	// lexware_sync_log.
	logID := testutil.SeedRow(t, pool, "lexware_sync_log", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"sync_type":   "contact_full",
		"status":      "completed",
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_log", logID)

	// lexware_webhook_subscriptions (UNIQUE on config_id + event_type).
	webhookID := testutil.SeedRow(t, pool, "lexware_webhook_subscriptions", map[string]any{
		"tenant_id":       testutil.TenantA,
		"config_id":       cfgID,
		"subscription_id": uuid.New().String(),
		"event_type":      "contact.created",
		"callback_url":    "https://app.zentria.tech/webhooks/lexware",
	})
	defer testutil.CleanupRow(t, pool, "lexware_webhook_subscriptions", webhookID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"lexware_entity_mappings", "lexware_entity_mappings", mapID},
		{"lexware_field_mappings", "lexware_field_mappings", fmID},
		{"lexware_sync_log", "lexware_sync_log", logID},
		{"lexware_webhook_subscriptions", "lexware_webhook_subscriptions", webhookID},
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
