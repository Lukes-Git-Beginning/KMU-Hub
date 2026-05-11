package vertraege

// Cross-tenant isolation tests for vertraege tables (Migration 000122).
// contracts, contract_parties, contract_reminders all have tenant_id NOT NULL
// and RLS was enabled in mig 000122.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Vertraege(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	now := time.Now().UTC()

	contractID := testutil.SeedRow(t, pool, "contracts", map[string]any{
		"tenant_id":       testutil.TenantA,
		"contract_number": "VTR-" + uuid.New().String()[:8],
		"title":           "Service Agreement",
		"contract_type":   "service",
		"starts_on":       now.Format("2006-01-02"),
	})
	defer testutil.CleanupRow(t, pool, "contracts", contractID)

	partyID := testutil.SeedRow(t, pool, "contract_parties", map[string]any{
		"tenant_id":       testutil.TenantA,
		"contract_id":     contractID,
		"party_type":      "external",
		"role_in_contract": "service_provider",
	})
	defer testutil.CleanupRow(t, pool, "contract_parties", partyID)

	reminderID := testutil.SeedRow(t, pool, "contract_reminders", map[string]any{
		"tenant_id":     testutil.TenantA,
		"contract_id":   contractID,
		"remind_at":     now.Add(30 * 24 * time.Hour),
		"reminder_type": "renewal",
		"subject":       "Contract renewal reminder",
	})
	defer testutil.CleanupRow(t, pool, "contract_reminders", reminderID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"contracts", "contracts", contractID},
		{"contract_parties", "contract_parties", partyID},
		{"contract_reminders", "contract_reminders", reminderID},
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
