package vertraege

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVertraegeWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Vertraege Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	contract := &Contract{
		ID:             uuid.New(),
		TenantID:       tenantWrite,
		ContractNumber: "V-" + uuid.New().String()[:8],
		Title:          "Mietvertrag Buero",
		ContractType:   ContractTypeRental,
		Status:         ContractStatusDraft,
		StartsOn:       now,
		Notes:          "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateContract(ctxA, contract); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", contract.ID)

	externalName := "Externe Partei GmbH"
	party := &ContractParty{
		ID:             uuid.New(),
		TenantID:       tenantWrite,
		ContractID:     contract.ID,
		PartyType:      PartyTypeExternal,
		ExternalName:   &externalName,
		RoleInContract: "Vermieter",
		CreatedAt:      now,
	}
	if err := repo.AddParty(ctxA, party); err != nil {
		t.Fatalf("AddParty: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_parties", party.ID)

	reminder := &ContractReminder{
		ID:           uuid.New(),
		TenantID:     tenantWrite,
		ContractID:   contract.ID,
		RemindAt:     now.Add(30 * 24 * time.Hour),
		ReminderType: ReminderTypeRenewal,
		Subject:      "Vertragsverlaengerung pruefen",
		Status:       ReminderStatusPending,
		CreatedAt:    now,
	}
	if err := repo.CreateReminder(ctxA, reminder); err != nil {
		t.Fatalf("CreateReminder: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_reminders", reminder.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"contracts", contract.ID},
		{"contract_parties", party.ID},
		{"contract_reminders", reminder.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
