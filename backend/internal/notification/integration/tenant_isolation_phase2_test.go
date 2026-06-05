package integration

// Cross-tenant isolation DB tests for notification integration tables (Migration 000122 + 000115).
// integration_channel_mappings, integration_account_links, integration_delivery_log,
// integration_link_tokens — all have tenant_id NOT NULL and RLS in mig 000122.
//
// integration_channel_mappings has config_id FK → integration_configs(id).
// integration_account_links has kmuhub_user_id FK → users(id) + UNIQUE constraints.
// integration_delivery_log has mapping_id FK → integration_channel_mappings(id).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Integration_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("integ-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed integration_configs — UNIQUE on (platform, tenant_id) since mig 000125.
	cfgID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantA,
		"platform":              "slack",
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgID)

	// integration_channel_mappings.
	chanMapID := testutil.SeedRow(t, pool, "integration_channel_mappings", map[string]any{
		"tenant_id":    testutil.TenantA,
		"config_id":    cfgID,
		"channel_id":   "C" + uuid.New().String()[:8],
		"channel_name": "general",
	})
	defer testutil.CleanupRow(t, pool, "integration_channel_mappings", chanMapID)

	// integration_account_links (UNIQUE on platform+external_user_id and platform+kmuhub_user_id).
	extUserID := uuid.New().String()[:20]
	acctLinkID := testutil.SeedRow(t, pool, "integration_account_links", map[string]any{
		"tenant_id":        testutil.TenantA,
		"platform":         "slack",
		"external_user_id": extUserID,
		"kmuhub_user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_account_links", acctLinkID)

	// integration_delivery_log.
	delivID := testutil.SeedRow(t, pool, "integration_delivery_log", map[string]any{
		"tenant_id":       testutil.TenantA,
		"notification_id": uuid.New(),
		"mapping_id":      chanMapID,
		"platform":        "slack",
		"status":          "sent",
	})
	defer testutil.CleanupRow(t, pool, "integration_delivery_log", delivID)

	// integration_link_tokens (no FK deps — ephemeral).
	tokenID := testutil.SeedRow(t, pool, "integration_link_tokens", map[string]any{
		"tenant_id":        testutil.TenantA,
		"platform":         "slack",
		"external_user_id": uuid.New().String()[:20],
		"token_hash":       uuid.New().String(),
		"expires_at":       time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "integration_link_tokens", tokenID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"integration_channel_mappings", "integration_channel_mappings", chanMapID},
		{"integration_account_links", "integration_account_links", acctLinkID},
		{"integration_delivery_log", "integration_delivery_log", delivID},
		{"integration_link_tokens", "integration_link_tokens", tokenID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
