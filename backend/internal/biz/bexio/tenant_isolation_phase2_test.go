package bexio

// Cross-tenant isolation DB tests for bexio_entity_mappings, bexio_field_mappings,
// bexio_sync_log (Migration 000122 + 000115).
//
// These tables have tenant_id NOT NULL (added in mig 000115) and RLS in mig 000122.
// They also have config_id FK → integration_configs(id); we seed a minimal
// integration_configs row under system-context.
//
// The existing tenant_isolation_test.go uses mock repos — this file exercises real DB.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Bexio_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (integration_configs.created_by FK).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("bexio-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed integration_configs — UNIQUE on (platform, tenant_id) since mig 000125.
	cfgID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             testutil.TenantA,
		"platform":              "bexio",
		"credentials_vault_key": "bexio/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgID)

	// bexio_entity_mappings.
	mapID := testutil.SeedRow(t, pool, "bexio_entity_mappings", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"entity_type": "contact",
		"kmuhub_id":   uuid.New(),
		"bexio_id":    99999,
	})
	defer testutil.CleanupRow(t, pool, "bexio_entity_mappings", mapID)

	// bexio_field_mappings (UNIQUE on config_id + entity_type).
	fmID := testutil.SeedRow(t, pool, "bexio_field_mappings", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"entity_type": "invoice",
	})
	defer testutil.CleanupRow(t, pool, "bexio_field_mappings", fmID)

	// bexio_sync_log.
	logID := testutil.SeedRow(t, pool, "bexio_sync_log", map[string]any{
		"tenant_id":   testutil.TenantA,
		"config_id":   cfgID,
		"sync_type":   "contact_full",
		"status":      "completed",
	})
	defer testutil.CleanupRow(t, pool, "bexio_sync_log", logID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"bexio_entity_mappings", "bexio_entity_mappings", mapID},
		{"bexio_field_mappings", "bexio_field_mappings", fmID},
		{"bexio_sync_log", "bexio_sync_log", logID},
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
