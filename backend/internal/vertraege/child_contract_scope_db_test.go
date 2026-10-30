package vertraege

// DB integration test for the nested party and reminder routes: a party or a
// reminder is only reachable through the path of the contract it belongs to.
// Before, RemoveParty, UpdateReminder and DeleteReminder filtered on id +
// tenant only, so any contract path of the tenant could change them.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestChildRoutes_RequireTheirOwnContract(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Child Scope Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	newContract := func(prefix string) *Contract {
		c := &Contract{
			ID: uuid.New(), TenantID: tenantID, ContractNumber: prefix + "-" + uuid.New().String()[:8],
			Title: prefix, ContractType: ContractTypeService, Status: ContractStatusActive,
			StartsOn: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateContract(ctx, c); err != nil {
			t.Fatalf("CreateContract %s: %v", prefix, err)
		}
		return c
	}
	contractA := newContract("CSA")
	contractB := newContract("CSB")
	defer testutil.CleanupRow(t, pool, "contracts", contractA.ID)
	defer testutil.CleanupRow(t, pool, "contracts", contractB.ID)

	t.Run("party", func(t *testing.T) {
		extName := "Partei von A"
		party := &ContractParty{ID: uuid.New(), TenantID: tenantID, ContractID: contractA.ID, PartyType: PartyTypeExternal, ExternalName: &extName, RoleInContract: "x", CreatedAt: now}
		if err := repo.AddParty(ctx, party); err != nil {
			t.Fatalf("AddParty: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "contract_parties", party.ID)

		if err := svc.RemoveParty(ctx, tenantID, contractB.ID, party.ID); !errors.Is(err, ErrPartyNotFound) {
			t.Fatalf("RemoveParty via contract B: expected ErrPartyNotFound, got %v", err)
		}
		left, err := repo.ListParties(ctx, tenantID, contractA.ID)
		if err != nil {
			t.Fatalf("ListParties: %v", err)
		}
		if len(left) != 1 || left[0].ID != party.ID {
			t.Fatalf("party must survive the removal via contract B, got %v", left)
		}

		if err := svc.RemoveParty(ctx, tenantID, contractA.ID, party.ID); err != nil {
			t.Fatalf("RemoveParty via contract A: %v", err)
		}
		left, err = repo.ListParties(ctx, tenantID, contractA.ID)
		if err != nil {
			t.Fatalf("ListParties after removal: %v", err)
		}
		if len(left) != 0 {
			t.Fatalf("party must be gone after the removal via contract A, got %v", left)
		}
	})

	t.Run("signed party via foreign contract", func(t *testing.T) {
		extName := "Unterzeichner von A"
		party := &ContractParty{ID: uuid.New(), TenantID: tenantID, ContractID: contractA.ID, PartyType: PartyTypeExternal, ExternalName: &extName, RoleInContract: "x", SignedOn: &now, CreatedAt: now}
		if err := repo.AddParty(ctx, party); err != nil {
			t.Fatalf("AddParty: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "contract_parties", party.ID)

		if err := svc.RemoveParty(ctx, tenantID, contractB.ID, party.ID); !errors.Is(err, ErrPartyNotFound) {
			t.Fatalf("RemoveParty (signed) via contract B: expected ErrPartyNotFound, got %v", err)
		}
	})

	t.Run("reminder", func(t *testing.T) {
		rem := &ContractReminder{
			ID: uuid.New(), TenantID: tenantID, ContractID: contractA.ID, RemindAt: now.Add(24 * time.Hour),
			ReminderType: ReminderTypeRenewal, Subject: "Original", Status: ReminderStatusPending, CreatedAt: now,
		}
		if err := repo.CreateReminder(ctx, rem); err != nil {
			t.Fatalf("CreateReminder: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "contract_reminders", rem.ID)

		subject := "Über B geändert"
		if _, err := svc.UpdateReminder(ctx, UpdateReminderInput{TenantID: tenantID, ContractID: contractB.ID, ReminderID: rem.ID, Subject: &subject}); !errors.Is(err, ErrReminderNotFound) {
			t.Fatalf("UpdateReminder via contract B: expected ErrReminderNotFound, got %v", err)
		}
		if err := svc.DeleteReminder(ctx, tenantID, contractB.ID, rem.ID); !errors.Is(err, ErrReminderNotFound) {
			t.Fatalf("DeleteReminder via contract B: expected ErrReminderNotFound, got %v", err)
		}
		got, err := repo.GetReminder(ctx, tenantID, rem.ID)
		if err != nil {
			t.Fatalf("reminder must survive the calls via contract B: %v", err)
		}
		if got.Subject != "Original" {
			t.Fatalf("reminder must be unchanged after the update via contract B, got subject %q", got.Subject)
		}

		subject = "Über A geändert"
		updated, err := svc.UpdateReminder(ctx, UpdateReminderInput{TenantID: tenantID, ContractID: contractA.ID, ReminderID: rem.ID, Subject: &subject})
		if err != nil {
			t.Fatalf("UpdateReminder via contract A: %v", err)
		}
		if updated.Subject != subject {
			t.Fatalf("UpdateReminder via contract A: subject not applied, got %q", updated.Subject)
		}
		if err := svc.DeleteReminder(ctx, tenantID, contractA.ID, rem.ID); err != nil {
			t.Fatalf("DeleteReminder via contract A: %v", err)
		}
		if _, err := repo.GetReminder(ctx, tenantID, rem.ID); !errors.Is(err, ErrReminderNotFound) {
			t.Fatalf("reminder must be gone after the delete via contract A, got %v", err)
		}
	})
}
