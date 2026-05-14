package datev

// Cross-tenant isolation DB tests for datev_upload_configs and datev_upload_log
// (Migration 000122 + 000115).
//
// Both tables have tenant_id NOT NULL (mig 000115) and RLS in mig 000122.
// config_id FK → integration_configs(id); we seed a minimal integration_configs row.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Datev_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("datev-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// integration_configs — UNIQUE on (platform, tenant_id) since mig 000125.
	cfgID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantA,
		"platform":              "datev_api",
		"credentials_vault_key": "datev/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgID)

	// datev_upload_configs (UNIQUE on config_id).
	uploadCfgID := testutil.SeedRow(t, pool, "datev_upload_configs", map[string]any{
		"tenant_id": testutil.TenantA,
		"config_id": cfgID,
	})
	defer testutil.CleanupRow(t, pool, "datev_upload_configs", uploadCfgID)

	// datev_upload_log.
	logID := testutil.SeedRow(t, pool, "datev_upload_log", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"upload_type": "buchungsstapel",
		"status":      "completed",
	})
	defer testutil.CleanupRow(t, pool, "datev_upload_log", logID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"datev_upload_configs", "datev_upload_configs", uploadCfgID},
		{"datev_upload_log", "datev_upload_log", logID},
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
