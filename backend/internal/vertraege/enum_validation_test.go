package vertraege

// The contracts tables refuse values outside their CHECK constraints and rows that do not
// exist; the service and repository have to say so as input errors, not let the constraint
// surface as an unhandled 500.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestService_EnumFields_OutsideWhitelist_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-ENUM", "Enum", ContractStatusDraft)
	rem := &ContractReminder{ID: uuid.New(), TenantID: tenantID, ContractID: c.ID, ReminderType: ReminderTypeCustom, Status: ReminderStatusPending}
	repo.reminders[rem.ID] = rem

	bogusType, bogusStatus := ContractType("bogus"), ContractStatus("bogus")
	bogusReminderType, bogusReminderStatus := ReminderType("bogus"), ReminderStatus("bogus")
	remindAt := time.Now().Add(time.Hour)

	cases := map[string]func() error{
		"create type": func() error {
			_, err := svc.CreateContract(ctx, CreateContractInput{TenantID: tenantID, ContractNumber: "CT-X", Title: "x", ContractType: bogusType})
			return err
		},
		"patch type": func() error {
			_, err := svc.UpdateContract(ctx, UpdateContractInput{TenantID: tenantID, ContractID: c.ID, ContractType: &bogusType})
			return err
		},
		"patch status": func() error {
			_, err := svc.UpdateContract(ctx, UpdateContractInput{TenantID: tenantID, ContractID: c.ID, Status: &bogusStatus})
			return err
		},
		"party type": func() error {
			_, err := svc.AddParty(ctx, AddPartyInput{TenantID: tenantID, ContractID: c.ID, PartyType: "bogus", RoleInContract: "r"})
			return err
		},
		"party no type": func() error {
			_, err := svc.AddParty(ctx, AddPartyInput{TenantID: tenantID, ContractID: c.ID, RoleInContract: "r"})
			return err
		},
		"reminder type": func() error {
			_, err := svc.CreateReminder(ctx, CreateReminderInput{TenantID: tenantID, ContractID: c.ID, RemindAt: remindAt, ReminderType: bogusReminderType, Subject: "s"})
			return err
		},
		"reminder no type": func() error {
			_, err := svc.CreateReminder(ctx, CreateReminderInput{TenantID: tenantID, ContractID: c.ID, RemindAt: remindAt, Subject: "s"})
			return err
		},
		"reminder patch type": func() error {
			_, err := svc.UpdateReminder(ctx, UpdateReminderInput{TenantID: tenantID, ContractID: rem.ContractID, ReminderID: rem.ID, ReminderType: &bogusReminderType})
			return err
		},
		"reminder patch status": func() error {
			_, err := svc.UpdateReminder(ctx, UpdateReminderInput{TenantID: tenantID, ContractID: rem.ContractID, ReminderID: rem.ID, Status: &bogusReminderStatus})
			return err
		},
	}
	for name, do := range cases {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, do(), ErrInvalidInput)
		})
	}

	// A refused patch changed nothing.
	assert.Equal(t, ContractTypeService, repo.contracts[c.ID].ContractType)
	assert.Equal(t, ContractStatusDraft, repo.contracts[c.ID].Status)
	assert.Equal(t, ReminderTypeCustom, repo.reminders[rem.ID].ReminderType)
}

func TestRepository_PartyAndReminder_UnknownOrForeignReferences(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	own, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, own, "Vertraege Ref Own")
	testutil.EnsureTenant(t, pool, other, "Vertraege Ref Other")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), own)
	now := time.Now().UTC()

	mine := &Contract{
		ID: uuid.New(), TenantID: own, ContractNumber: "REF-" + uuid.NewString()[:8],
		Title: "Mein Vertrag", ContractType: ContractTypeService, Status: ContractStatusDraft,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.CreateContract(ctx, mine))
	defer testutil.CleanupRow(t, pool, "contracts", mine.ID)

	foreignContract := testutil.SeedRow(t, pool, "contracts", map[string]any{
		"id": uuid.New(), "tenant_id": other, "contract_number": "REF-" + uuid.NewString()[:8],
		"title": "Fremd", "contract_type": "service", "starts_on": now,
	})
	defer testutil.CleanupRow(t, pool, "contracts", foreignContract)
	foreignUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": other, "email": "ref-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Ref", "last_name": "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", foreignUser)
	foreignContact := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id": uuid.New(), "tenant_id": other, "first_name": "Fremd", "last_name": "Kontakt", "created_by": foreignUser,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContact)
	foreignCompany := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id": uuid.New(), "tenant_id": other, "name": "Fremd GmbH", "created_by": foreignUser,
	})
	defer testutil.CleanupRow(t, pool, "companies", foreignCompany)

	extName := "Extern"
	party := func(contractID uuid.UUID, contactID, companyID *uuid.UUID) *ContractParty {
		return &ContractParty{
			ID: uuid.New(), TenantID: own, ContractID: contractID, PartyType: PartyTypeExternal,
			ContactID: contactID, CompanyID: companyID, ExternalName: &extName, RoleInContract: "r", CreatedAt: now,
		}
	}
	unknown := uuid.New()

	// A contract that does not exist, or belongs to another tenant, is not found: the foreign
	// key alone would accept the other tenant's contract.
	for name, id := range map[string]uuid.UUID{"unknown": unknown, "foreign": foreignContract} {
		err := repo.AddParty(ctx, party(id, nil, nil))
		assert.ErrorIs(t, err, ErrContractNotFound, "party on %s contract", name)
		err = repo.CreateReminder(ctx, &ContractReminder{
			ID: uuid.New(), TenantID: own, ContractID: id, RemindAt: now, ReminderType: ReminderTypeCustom,
			Subject: "s", Status: ReminderStatusPending, CreatedAt: now,
		})
		assert.ErrorIs(t, err, ErrContractNotFound, "reminder on %s contract", name)
	}

	// An unknown or foreign contact / company is a bad reference, not a missing contract.
	for name, ids := range map[string][2]uuid.UUID{"unknown": {unknown, unknown}, "foreign": {foreignContact, foreignCompany}} {
		err := repo.AddParty(ctx, party(mine.ID, &ids[0], nil))
		assert.True(t, errors.Is(err, ErrPartyRefNotFound), "%s contact: %v", name, err)
		err = repo.AddParty(ctx, party(mine.ID, nil, &ids[1]))
		assert.True(t, errors.Is(err, ErrPartyRefNotFound), "%s company: %v", name, err)
	}

	parties, err := repo.ListParties(ctx, own, mine.ID)
	require.NoError(t, err)
	assert.Empty(t, parties, "refused parties must not be stored")
}
