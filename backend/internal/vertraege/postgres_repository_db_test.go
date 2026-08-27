package vertraege

// DB integration tests for the parts of postgres_repository.go that
// tenant_write_test.go (Create*) and tenant_isolation_phase2_test.go (RLS
// SELECT on hand-seeded rows) don't reach: SaveSignature, UpdateContract,
// GetContract (incl. its Parties/Reminders assembly), ListContracts
// filtering/sorting/pagination, DeleteContract, ContractNumberExists,
// ExpireContracts, RemoveParty/ListParties, the reminder CRUD + worker
// queries (ClaimDueReminders/MarkReminderSent), and the contract events
// read/write pair. signature_test.go covers the Service.SaveSignature
// validation logic against a mock repo, not this SQL statement.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRepository_SaveSignature_UpdatesFieldsAndTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege Signature Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Signature Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "SIG-" + uuid.New().String()[:8],
		Title: "Zu unterschreiben", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	signed, err := repo.SaveSignature(ctxOwn, tenantOwn.String(), c.ID.String(), "data:image/png;base64,AAAA", "Anna Muster")
	if err != nil {
		t.Fatalf("SaveSignature: %v", err)
	}
	if signed.SignatureData == nil || *signed.SignatureData != "data:image/png;base64,AAAA" {
		t.Fatalf("SaveSignature: SignatureData not applied, got %v", signed.SignatureData)
	}
	if signed.SignedBy == nil || *signed.SignedBy != "Anna Muster" {
		t.Fatalf("SaveSignature: SignedBy not applied, got %v", signed.SignedBy)
	}
	if signed.SignedAt == nil {
		t.Fatalf("SaveSignature: SignedAt not set")
	}

	// A foreign-tenant call against the same contract id must report not found,
	// not silently touch someone else's contract.
	if _, err := repo.SaveSignature(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther.String(), c.ID.String(), "data:image/png;base64,BBBB", "Fremd"); err != ErrContractNotFound {
		t.Fatalf("SaveSignature (foreign tenant): expected ErrContractNotFound, got %v", err)
	}

	if _, err := repo.SaveSignature(ctxOwn, tenantOwn.String(), uuid.New().String(), "data:image/png;base64,CCCC", "Egal"); err != ErrContractNotFound {
		t.Fatalf("SaveSignature (missing contract): expected ErrContractNotFound, got %v", err)
	}
}

// TestSaveSignature_OverwritesExistingSignatureWithoutGuard documents a
// VERIFIED finding against the real schema, found while covering this
// handler group (cov-gateway-vertraege-lifecycle-and-signature): the UPDATE
// in PostgresRepository.SaveSignature has no "AND signature_data IS NULL"
// clause and Service.SaveSignature has no check against the contract's
// existing SignedAt/SignatureData — a contract that already carries a
// signature silently accepts and persists a second one, with no trace of
// the first left anywhere in the contracts row. Same bug class as
// fix-rapporte-signature-overwritable-after-signing (rapporte) and
// fix-vermietung-rental-signature-overwritable-after-signing (vermietung).
// Filed as its own fix-unit — a coverage unit changes no behaviour.
func TestSaveSignature_OverwritesExistingSignatureWithoutGuard(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Signature Overwrite Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "SIG-OVR-" + uuid.New().String()[:8],
		Title: "Zweimal unterschrieben", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	first, err := repo.SaveSignature(ctx, tenantID.String(), c.ID.String(), "data:image/png;base64,firstSignature", "Max Mustermann")
	if err != nil {
		t.Fatalf("SaveSignature (first): %v", err)
	}
	if first.SignedBy == nil || *first.SignedBy != "Max Mustermann" {
		t.Fatalf("expected first signature persisted, got %+v", first)
	}
	firstSignedAt := *first.SignedAt

	second, err := repo.SaveSignature(ctx, tenantID.String(), c.ID.String(), "data:image/png;base64,secondSignatureReplacingTheFirst", "Erika Musterfrau")
	if err != nil {
		t.Fatalf("expected the re-sign to currently SUCCEED (documenting the gap), got error: %v", err)
	}
	if second.SignedBy == nil || *second.SignedBy != "Erika Musterfrau" {
		t.Fatalf("expected the second signer to have overwritten the first, got %+v", second)
	}
	if !second.SignedAt.After(firstSignedAt) {
		t.Fatalf("expected signed_at to have advanced past the first signature, first=%v second=%v", firstSignedAt, *second.SignedAt)
	}

	reloaded, err := repo.GetContract(ctx, tenantID, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if reloaded.SignedBy == nil || *reloaded.SignedBy != "Erika Musterfrau" {
		t.Fatalf("expected the persisted row to show the second signer with no trace of the first, got %+v", reloaded.SignedBy)
	}
}

func TestRepository_UpdateContract_ChangesFieldsAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege Update Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Update Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID:             uuid.New(),
		TenantID:       tenantOwn,
		ContractNumber: "UPD-" + uuid.New().String()[:8],
		Title:          "Alter Titel",
		ContractType:   ContractTypeService,
		Status:         ContractStatusDraft,
		StartsOn:       now,
		Notes:          "alt",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	// ends_on is a DATE column; the server truncates any time-of-day component,
	// so the value used for comparison after the round-trip must already be
	// midnight UTC.
	ends := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(365 * 24 * time.Hour)
	docURL := "s3://contracts/updated.pdf"
	provider := "skribble"
	updated := &Contract{
		ID:                c.ID,
		TenantID:          tenantOwn,
		Title:             "Neuer Titel",
		ContractType:      ContractTypeRental,
		Status:            ContractStatusActive,
		StartsOn:          now,
		EndsOn:            &ends,
		DocumentURL:       &docURL,
		Notes:             "neu",
		SignatureProvider: &provider,
		UpdatedAt:         now.Add(time.Hour),
	}
	if err := repo.UpdateContract(ctxOwn, updated); err != nil {
		t.Fatalf("UpdateContract: %v", err)
	}

	got, err := repo.GetContract(ctxOwn, tenantOwn, c.ID)
	if err != nil {
		t.Fatalf("GetContract after update: %v", err)
	}
	if got.Title != "Neuer Titel" || got.ContractType != ContractTypeRental || got.Status != ContractStatusActive {
		t.Fatalf("UpdateContract: fields not applied, got %+v", got)
	}
	if got.EndsOn == nil || !got.EndsOn.Equal(ends) {
		t.Fatalf("UpdateContract: EndsOn not applied, got %v", got.EndsOn)
	}
	if got.DocumentURL == nil || *got.DocumentURL != docURL {
		t.Fatalf("UpdateContract: DocumentURL not applied, got %v", got.DocumentURL)
	}
	if got.Notes != "neu" {
		t.Fatalf("UpdateContract: Notes not applied, got %q", got.Notes)
	}
	if got.SignatureProvider == nil || *got.SignatureProvider != provider {
		t.Fatalf("UpdateContract: SignatureProvider not applied, got %v", got.SignatureProvider)
	}

	// A foreign-tenant update targeting the same contract ID must be a silent
	// no-op — the WHERE clause carries tenant_id, not just id.
	foreignAttempt := &Contract{
		ID:           c.ID,
		TenantID:     tenantOther,
		Title:        "Uebernommen",
		ContractType: ContractTypeOther,
		Status:       ContractStatusTerminated,
		StartsOn:     now,
		UpdatedAt:    now,
	}
	if err := repo.UpdateContract(testutil.WithTenantCtx(context.Background(), tenantOther), foreignAttempt); err != nil {
		t.Fatalf("UpdateContract (foreign tenant): unexpected error %v", err)
	}
	stillOwn, err := repo.GetContract(ctxOwn, tenantOwn, c.ID)
	if err != nil {
		t.Fatalf("GetContract after foreign update attempt: %v", err)
	}
	if stillOwn.Title != "Neuer Titel" {
		t.Fatalf("UpdateContract (foreign tenant) leaked through: title now %q", stillOwn.Title)
	}
}

func TestRepository_GetContract_AssemblesPartiesAndReminders(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Get Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	c := &Contract{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ContractNumber: "GET-" + uuid.New().String()[:8],
		Title:          "Mit Anhang",
		ContractType:   ContractTypeService,
		Status:         ContractStatusActive,
		StartsOn:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	extName := "Externe Partei GmbH"
	party := &ContractParty{
		ID: uuid.New(), TenantID: tenantID, ContractID: c.ID,
		PartyType: PartyTypeExternal, ExternalName: &extName,
		RoleInContract: "Vermieter", CreatedAt: now,
	}
	if err := repo.AddParty(ctx, party); err != nil {
		t.Fatalf("AddParty: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_parties", party.ID)

	reminder := &ContractReminder{
		ID: uuid.New(), TenantID: tenantID, ContractID: c.ID,
		RemindAt: now.Add(24 * time.Hour), ReminderType: ReminderTypeExpiry,
		Subject: "Bald faellig", Status: ReminderStatusPending, CreatedAt: now,
	}
	if err := repo.CreateReminder(ctx, reminder); err != nil {
		t.Fatalf("CreateReminder: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_reminders", reminder.ID)

	got, err := repo.GetContract(ctx, tenantID, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if len(got.Parties) != 1 || got.Parties[0].ID != party.ID {
		t.Fatalf("GetContract: expected 1 party %s, got %v", party.ID, got.Parties)
	}
	if len(got.Reminders) != 1 || got.Reminders[0].ID != reminder.ID {
		t.Fatalf("GetContract: expected 1 reminder %s, got %v", reminder.ID, got.Reminders)
	}

	if _, err := repo.GetContract(ctx, tenantID, uuid.New()); err != ErrContractNotFound {
		t.Fatalf("GetContract (missing): expected ErrContractNotFound, got %v", err)
	}
	if _, err := repo.GetContract(ctx, uuid.New(), c.ID); err != ErrContractNotFound {
		t.Fatalf("GetContract (foreign tenant): expected ErrContractNotFound, got %v", err)
	}
}

func TestRepository_ListContracts_FiltersSortsAndPaginatesTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege List Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()
	day := 24 * time.Hour

	active := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "LST-" + uuid.New().String()[:8],
		Title: "Aktiv", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now.Add(-30 * day), CreatedAt: now, UpdatedAt: now,
	}
	ends := now.Add(10 * day)
	rental := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "LST-" + uuid.New().String()[:8],
		Title: "Miete", ContractType: ContractTypeRental, Status: ContractStatusActive,
		StartsOn: now.Add(-5 * day), EndsOn: &ends, CreatedAt: now.Add(time.Second), UpdatedAt: now,
	}
	draft := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "LST-" + uuid.New().String()[:8],
		Title: "Entwurf", ContractType: ContractTypeNDA, Status: ContractStatusDraft,
		StartsOn: now.Add(30 * day), CreatedAt: now.Add(2 * time.Second), UpdatedAt: now,
	}
	foreign := &Contract{
		ID: uuid.New(), TenantID: tenantOther, ContractNumber: "LST-" + uuid.New().String()[:8],
		Title: "Fremd", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	for _, c := range []*Contract{active, rental, draft} {
		if err := repo.CreateContract(ctxOwn, c); err != nil {
			t.Fatalf("CreateContract %s: %v", c.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "contracts", c.ID)
	}
	if err := repo.CreateContract(ctxOther, foreign); err != nil {
		t.Fatalf("CreateContract foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", foreign.ID)

	// Status filter.
	draftStatus := ContractStatusDraft
	draftResults, draftTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{Status: &draftStatus}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (status): %v", err)
	}
	if draftTotal != 1 || len(draftResults) != 1 || draftResults[0].ID != draft.ID {
		t.Fatalf("ListContracts (status): expected exactly draft, got total=%d results=%v", draftTotal, draftResults)
	}

	// Type filter.
	rentalType := ContractTypeRental
	typeResults, typeTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{Type: &rentalType}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (type): %v", err)
	}
	if typeTotal != 1 || len(typeResults) != 1 || typeResults[0].ID != rental.ID {
		t.Fatalf("ListContracts (type): expected exactly rental, got total=%d results=%v", typeTotal, typeResults)
	}

	// StartsAfter / StartsBefore.
	startsAfter := now.Add(-10 * day)
	afterResults, afterTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{StartsAfter: &startsAfter}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (starts after): %v", err)
	}
	if afterTotal != 2 {
		t.Fatalf("ListContracts (starts after): expected rental+draft (2), got total=%d results=%v", afterTotal, afterResults)
	}
	startsBefore := now
	beforeResults, beforeTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{StartsBefore: &startsBefore}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (starts before): %v", err)
	}
	if beforeTotal != 2 {
		t.Fatalf("ListContracts (starts before): expected active+rental (2), got total=%d results=%v", beforeTotal, beforeResults)
	}

	// EndsAfter / EndsBefore — only rental has ends_on set.
	endsAfter := now
	eaResults, eaTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{EndsAfter: &endsAfter}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (ends after): %v", err)
	}
	if eaTotal != 1 || eaResults[0].ID != rental.ID {
		t.Fatalf("ListContracts (ends after): expected exactly rental, got total=%d results=%v", eaTotal, eaResults)
	}
	endsBefore := now.Add(20 * day)
	ebResults, ebTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{EndsBefore: &endsBefore}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (ends before): %v", err)
	}
	if ebTotal != 1 || ebResults[0].ID != rental.ID {
		t.Fatalf("ListContracts (ends before): expected exactly rental, got total=%d results=%v", ebTotal, ebResults)
	}

	// Pagination: created_at DESC, so draft (latest) leads.
	page, total, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{}, 0, 1)
	if err != nil {
		t.Fatalf("ListContracts (page 1): %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ID != draft.ID {
		t.Fatalf("ListContracts (page 1): expected total=3 first=draft, got total=%d page=%v", total, page)
	}
	empty, emptyTotal, err := repo.ListContracts(ctxOwn, tenantOwn, ListContractsFilter{}, 50, 20)
	if err != nil {
		t.Fatalf("ListContracts (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("ListContracts (offset beyond total): expected 0 rows total=3, got %d rows total=%d", len(empty), emptyTotal)
	}

	// Tenant scoping: the foreign tenant only ever sees its own row.
	foreignView, foreignTotal, err := repo.ListContracts(ctxOther, tenantOther, ListContractsFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (foreign tenant): %v", err)
	}
	if foreignTotal != 1 || len(foreignView) != 1 || foreignView[0].ID != foreign.ID {
		t.Fatalf("ListContracts (foreign tenant): expected exactly the foreign contract, got total=%d results=%v", foreignTotal, foreignView)
	}
}

func TestRepository_ListContracts_ContactIDFilterUsesPartyExists(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege ContactID Filter Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("vertraege-contactid-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Kontakt",
		"last_name":  "Partei",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	withParty := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "CID-" + uuid.New().String()[:8],
		Title: "Mit Kontaktpartei", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	withoutParty := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "CID-" + uuid.New().String()[:8],
		Title: "Ohne Kontaktpartei", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	for _, c := range []*Contract{withParty, withoutParty} {
		if err := repo.CreateContract(ctx, c); err != nil {
			t.Fatalf("CreateContract %s: %v", c.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "contracts", c.ID)
	}

	party := &ContractParty{
		ID: uuid.New(), TenantID: tenantID, ContractID: withParty.ID,
		PartyType: PartyTypeContact, ContactID: &contactID,
		RoleInContract: "Vertragspartner", CreatedAt: now,
	}
	if err := repo.AddParty(ctx, party); err != nil {
		t.Fatalf("AddParty: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_parties", party.ID)

	results, total, err := repo.ListContracts(ctx, tenantID, ListContractsFilter{ContactID: &contactID}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (contact id): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != withParty.ID {
		t.Fatalf("ListContracts (contact id): expected exactly withParty, got total=%d results=%v", total, results)
	}
}

func TestRepository_DeleteContract_RemovesRowAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege Delete Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Delete Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "DEL-" + uuid.New().String()[:8],
		Title: "Zu loeschen", ContractType: ContractTypeOther, Status: ContractStatusDraft,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	// A foreign-tenant delete on the same id must be a no-op.
	if err := repo.DeleteContract(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther, c.ID); err != nil {
		t.Fatalf("DeleteContract (foreign tenant): unexpected error %v", err)
	}
	if _, err := repo.GetContract(ctxOwn, tenantOwn, c.ID); err != nil {
		t.Fatalf("GetContract after foreign delete attempt: expected contract to still exist, got %v", err)
	}

	if err := repo.DeleteContract(ctxOwn, tenantOwn, c.ID); err != nil {
		t.Fatalf("DeleteContract: %v", err)
	}
	if _, err := repo.GetContract(ctxOwn, tenantOwn, c.ID); err != ErrContractNotFound {
		t.Fatalf("GetContract after delete: expected ErrContractNotFound, got %v", err)
	}
}

func TestRepository_ContractNumberExists_WithAndWithoutExclude(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Number Exists Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	number := "NUM-" + uuid.New().String()[:8]

	c := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: number,
		Title: "Nummer", ContractType: ContractTypeOther, Status: ContractStatusDraft,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	exists, err := repo.ContractNumberExists(ctx, tenantID, number, nil)
	if err != nil {
		t.Fatalf("ContractNumberExists: %v", err)
	}
	if !exists {
		t.Fatalf("ContractNumberExists: expected true for existing number")
	}

	notExists, err := repo.ContractNumberExists(ctx, tenantID, "does-not-exist-"+number, nil)
	if err != nil {
		t.Fatalf("ContractNumberExists (missing): %v", err)
	}
	if notExists {
		t.Fatalf("ContractNumberExists (missing): expected false")
	}

	// Excluding the contract's own id must report the number as free — this is
	// the "am I renaming myself back to my own number" case.
	excludedSelf, err := repo.ContractNumberExists(ctx, tenantID, number, &c.ID)
	if err != nil {
		t.Fatalf("ContractNumberExists (exclude self): %v", err)
	}
	if excludedSelf {
		t.Fatalf("ContractNumberExists (exclude self): expected false when excluding the only holder")
	}

	otherID := uuid.New()
	excludedOther, err := repo.ContractNumberExists(ctx, tenantID, number, &otherID)
	if err != nil {
		t.Fatalf("ContractNumberExists (exclude other): %v", err)
	}
	if !excludedOther {
		t.Fatalf("ContractNumberExists (exclude other): expected true when excluding an unrelated id")
	}
}

// A contract that ended on the last day of last year, and one that ends on
// the last day of last month, are both firmly in the past — but only if the
// starts_on/ends_on DATE comparisons roll over year and month boundaries
// correctly instead of e.g. comparing day-of-month in isolation.
func TestRepository_ListContracts_DateFiltersCrossMonthAndYearBoundaries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Date Boundary Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Last day of the previous month (crosses a month boundary, and in
	// January also a year boundary).
	firstOfThisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDayPrevMonth := firstOfThisMonth.Add(-24 * time.Hour)
	// Dec 31 of last year — an explicit year-boundary case independent of
	// which month "now" happens to fall in.
	dec31LastYear := time.Date(today.Year()-1, time.December, 31, 0, 0, 0, 0, time.UTC)

	monthBoundary := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "MBD-" + uuid.New().String()[:8],
		Title: "Endete letzten Monat", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: lastDayPrevMonth.Add(-10 * 24 * time.Hour), EndsOn: &lastDayPrevMonth, CreatedAt: now, UpdatedAt: now,
	}
	yearBoundary := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "YBD-" + uuid.New().String()[:8],
		Title: "Endete letztes Jahr", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: dec31LastYear.Add(-30 * 24 * time.Hour), EndsOn: &dec31LastYear, CreatedAt: now, UpdatedAt: now,
	}
	for _, c := range []*Contract{monthBoundary, yearBoundary} {
		if err := repo.CreateContract(ctx, c); err != nil {
			t.Fatalf("CreateContract %s: %v", c.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "contracts", c.ID)
	}

	// EndsBefore today must include both — both ended strictly before today,
	// across a month boundary and a year boundary respectively.
	results, total, err := repo.ListContracts(ctx, tenantID, ListContractsFilter{EndsBefore: &today}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (ends before today): %v", err)
	}
	if total != 2 {
		t.Fatalf("ListContracts (ends before today): expected both boundary contracts, got total=%d results=%v", total, results)
	}

	// EndsAfter the day right after the more recent of the two boundary dates
	// excludes both, regardless of which calendar month "now" falls in
	// (lastDayPrevMonth is never before dec31LastYear: they coincide only
	// when today is in January, where the "previous month" is December).
	dayAfterMostRecentBoundary := lastDayPrevMonth.Add(24 * time.Hour)
	afterResults, afterTotal, err := repo.ListContracts(ctx, tenantID, ListContractsFilter{EndsAfter: &dayAfterMostRecentBoundary}, 0, 20)
	if err != nil {
		t.Fatalf("ListContracts (ends after day-after boundary): %v", err)
	}
	if afterTotal != 0 || len(afterResults) != 0 {
		t.Fatalf("ListContracts (ends after day-after boundary): expected neither boundary contract, got total=%d results=%v", afterTotal, afterResults)
	}

	// The auto-expiry job must flip both to expired, exercising the same DATE
	// comparison (ends_on < CURRENT_DATE) from the write side.
	if _, err := repo.ExpireContracts(testutil.WithSystemCtx(context.Background())); err != nil {
		t.Fatalf("ExpireContracts: %v", err)
	}
	for _, c := range []*Contract{monthBoundary, yearBoundary} {
		got, err := repo.GetContract(ctx, tenantID, c.ID)
		if err != nil {
			t.Fatalf("GetContract %s: %v", c.Title, err)
		}
		if got.Status != ContractStatusExpired {
			t.Fatalf("ExpireContracts: expected %s to be expired, got %s", c.Title, got.Status)
		}
	}
}

func TestRepository_ExpireContracts_OnlyActiveWithPastEndsOn(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Expire Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	nextMonth := now.Add(30 * 24 * time.Hour)

	pastEnded := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "EXP-" + uuid.New().String()[:8],
		Title: "Abgelaufen", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: yesterday.Add(-100 * 24 * time.Hour), EndsOn: &yesterday, CreatedAt: now, UpdatedAt: now,
	}
	stillActive := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "EXP-" + uuid.New().String()[:8],
		Title: "Laeuft weiter", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, EndsOn: &nextMonth, CreatedAt: now, UpdatedAt: now,
	}
	openEnded := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "EXP-" + uuid.New().String()[:8],
		Title: "Unbefristet", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	alreadyDraft := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "EXP-" + uuid.New().String()[:8],
		Title: "Noch Entwurf", ContractType: ContractTypeService, Status: ContractStatusDraft,
		StartsOn: now, EndsOn: &yesterday, CreatedAt: now, UpdatedAt: now,
	}
	for _, c := range []*Contract{pastEnded, stillActive, openEnded, alreadyDraft} {
		if err := repo.CreateContract(ctx, c); err != nil {
			t.Fatalf("CreateContract %s: %v", c.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "contracts", c.ID)
	}

	// ExpireContracts scans across all tenants (it's the auto-expiry worker
	// job, not a tenant-scoped call) — that only clears RLS with a system
	// context, exactly like ReminderWorker.Run wraps its ctx via
	// database.WithSystemContext before calling it.
	if _, err := repo.ExpireContracts(testutil.WithSystemCtx(context.Background())); err != nil {
		t.Fatalf("ExpireContracts: %v", err)
	}

	got, err := repo.GetContract(ctx, tenantID, pastEnded.ID)
	if err != nil {
		t.Fatalf("GetContract pastEnded: %v", err)
	}
	if got.Status != ContractStatusExpired {
		t.Fatalf("ExpireContracts: expected pastEnded to be expired, got %s", got.Status)
	}

	for _, unaffected := range []*Contract{stillActive, openEnded, alreadyDraft} {
		got, err := repo.GetContract(ctx, tenantID, unaffected.ID)
		if err != nil {
			t.Fatalf("GetContract %s: %v", unaffected.Title, err)
		}
		if got.Status == ContractStatusExpired {
			t.Fatalf("ExpireContracts: %s must not be expired, got %s", unaffected.Title, got.Status)
		}
	}
}

func TestRepository_ListParties_OrderedByCreatedAt(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege ListParties Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "PTY-" + uuid.New().String()[:8],
		Title: "Mit Parteien", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	firstName, secondName := "Erste Partei", "Zweite Partei"
	first := &ContractParty{ID: uuid.New(), TenantID: tenantID, ContractID: c.ID, PartyType: PartyTypeExternal, ExternalName: &firstName, RoleInContract: "a", CreatedAt: now}
	second := &ContractParty{ID: uuid.New(), TenantID: tenantID, ContractID: c.ID, PartyType: PartyTypeExternal, ExternalName: &secondName, RoleInContract: "b", CreatedAt: now.Add(time.Second)}
	if err := repo.AddParty(ctx, first); err != nil {
		t.Fatalf("AddParty first: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_parties", first.ID)
	if err := repo.AddParty(ctx, second); err != nil {
		t.Fatalf("AddParty second: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_parties", second.ID)

	parties, err := repo.ListParties(ctx, tenantID, c.ID)
	if err != nil {
		t.Fatalf("ListParties: %v", err)
	}
	if len(parties) != 2 || parties[0].ID != first.ID || parties[1].ID != second.ID {
		t.Fatalf("ListParties: expected [first, second] in creation order, got %v", parties)
	}
}

func TestRepository_RemoveParty_ReturnsContractIDOrNilOnMissing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege RemoveParty Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege RemoveParty Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "RMP-" + uuid.New().String()[:8],
		Title: "Partei entfernen", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	extName := "Zu entfernende Partei"
	party := &ContractParty{ID: uuid.New(), TenantID: tenantOwn, ContractID: c.ID, PartyType: PartyTypeExternal, ExternalName: &extName, RoleInContract: "x", CreatedAt: now}
	if err := repo.AddParty(ctxOwn, party); err != nil {
		t.Fatalf("AddParty: %v", err)
	}

	// Missing party: no-op, returns uuid.Nil and no error.
	gotID, err := repo.RemoveParty(ctxOwn, tenantOwn, uuid.New())
	if err != nil {
		t.Fatalf("RemoveParty (missing): unexpected error %v", err)
	}
	if gotID != uuid.Nil {
		t.Fatalf("RemoveParty (missing): expected uuid.Nil, got %v", gotID)
	}

	// Foreign tenant cannot remove another tenant's party.
	gotID, err = repo.RemoveParty(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther, party.ID)
	if err != nil {
		t.Fatalf("RemoveParty (foreign tenant): unexpected error %v", err)
	}
	if gotID != uuid.Nil {
		t.Fatalf("RemoveParty (foreign tenant): expected uuid.Nil, got %v", gotID)
	}
	stillThere, err := repo.ListParties(ctxOwn, tenantOwn, c.ID)
	if err != nil {
		t.Fatalf("ListParties after foreign remove attempt: %v", err)
	}
	if len(stillThere) != 1 {
		t.Fatalf("RemoveParty (foreign tenant) leaked through: %v", stillThere)
	}

	// Owning tenant removes successfully and gets the contract id back.
	gotID, err = repo.RemoveParty(ctxOwn, tenantOwn, party.ID)
	if err != nil {
		t.Fatalf("RemoveParty: %v", err)
	}
	if gotID != c.ID {
		t.Fatalf("RemoveParty: expected contract id %s, got %v", c.ID, gotID)
	}
	afterRemoval, err := repo.ListParties(ctxOwn, tenantOwn, c.ID)
	if err != nil {
		t.Fatalf("ListParties after remove: %v", err)
	}
	if len(afterRemoval) != 0 {
		t.Fatalf("RemoveParty: expected 0 parties left, got %v", afterRemoval)
	}
}

func TestRepository_ReminderCRUDAndFiltering(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege Reminder Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Reminder Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "RMD-" + uuid.New().String()[:8],
		Title: "Mit Erinnerungen", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	pending := &ContractReminder{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: c.ID, RemindAt: now.Add(24 * time.Hour),
		ReminderType: ReminderTypeRenewal, Subject: "Bald", Status: ReminderStatusPending, CreatedAt: now,
	}
	sent := &ContractReminder{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: c.ID, RemindAt: now.Add(-24 * time.Hour),
		ReminderType: ReminderTypeExpiry, Subject: "Schon verschickt", Status: ReminderStatusSent, CreatedAt: now,
	}
	for _, r := range []*ContractReminder{pending, sent} {
		if err := repo.CreateReminder(ctxOwn, r); err != nil {
			t.Fatalf("CreateReminder %s: %v", r.Subject, err)
		}
		defer testutil.CleanupRow(t, pool, "contract_reminders", r.ID)
	}

	// GetReminder: found + cross-tenant not found.
	got, err := repo.GetReminder(ctxOwn, tenantOwn, pending.ID)
	if err != nil {
		t.Fatalf("GetReminder: %v", err)
	}
	if got.Subject != "Bald" {
		t.Fatalf("GetReminder: unexpected subject %q", got.Subject)
	}
	if _, err := repo.GetReminder(ctxOwn, tenantOther, pending.ID); err != ErrReminderNotFound {
		t.Fatalf("GetReminder (foreign tenant): expected ErrReminderNotFound, got %v", err)
	}
	if _, err := repo.GetReminder(ctxOwn, tenantOwn, uuid.New()); err != ErrReminderNotFound {
		t.Fatalf("GetReminder (missing): expected ErrReminderNotFound, got %v", err)
	}

	// UpdateReminder.
	updated := &ContractReminder{
		ID: pending.ID, TenantID: tenantOwn, RemindAt: now.Add(48 * time.Hour),
		ReminderType: ReminderTypePayment, Subject: "Verschoben", Message: "neu terminiert", Status: ReminderStatusPending,
	}
	if err := repo.UpdateReminder(ctxOwn, updated); err != nil {
		t.Fatalf("UpdateReminder: %v", err)
	}
	afterUpdate, err := repo.GetReminder(ctxOwn, tenantOwn, pending.ID)
	if err != nil {
		t.Fatalf("GetReminder after update: %v", err)
	}
	if afterUpdate.Subject != "Verschoben" || afterUpdate.ReminderType != ReminderTypePayment {
		t.Fatalf("UpdateReminder: fields not applied, got %+v", afterUpdate)
	}

	// ListReminders: all vs. onlyPending, ordered by remind_at ASC.
	all, err := repo.ListReminders(ctxOwn, tenantOwn, c.ID, false)
	if err != nil {
		t.Fatalf("ListReminders (all): %v", err)
	}
	if len(all) != 2 || all[0].ID != sent.ID || all[1].ID != pending.ID {
		t.Fatalf("ListReminders (all): expected [sent, pending] by remind_at asc, got %v", all)
	}
	onlyPending, err := repo.ListReminders(ctxOwn, tenantOwn, c.ID, true)
	if err != nil {
		t.Fatalf("ListReminders (only pending): %v", err)
	}
	if len(onlyPending) != 1 || onlyPending[0].ID != pending.ID {
		t.Fatalf("ListReminders (only pending): expected exactly pending, got %v", onlyPending)
	}

	// DeleteReminder is tenant-scoped.
	if err := repo.DeleteReminder(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther, sent.ID); err != nil {
		t.Fatalf("DeleteReminder (foreign tenant): unexpected error %v", err)
	}
	if _, err := repo.GetReminder(ctxOwn, tenantOwn, sent.ID); err != nil {
		t.Fatalf("GetReminder after foreign delete attempt: expected reminder to still exist, got %v", err)
	}
	if err := repo.DeleteReminder(ctxOwn, tenantOwn, sent.ID); err != nil {
		t.Fatalf("DeleteReminder: %v", err)
	}
	if _, err := repo.GetReminder(ctxOwn, tenantOwn, sent.ID); err != ErrReminderNotFound {
		t.Fatalf("GetReminder after delete: expected ErrReminderNotFound, got %v", err)
	}
}

func TestRepository_ClaimDueRemindersAndMarkSent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vertraege Worker Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantID, ContractNumber: "WRK-" + uuid.New().String()[:8],
		Title: "Worker", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	due := &ContractReminder{
		ID: uuid.New(), TenantID: tenantID, ContractID: c.ID, RemindAt: now.Add(-time.Hour),
		ReminderType: ReminderTypeCustom, Subject: "Faellig", Status: ReminderStatusPending, CreatedAt: now,
	}
	future := &ContractReminder{
		ID: uuid.New(), TenantID: tenantID, ContractID: c.ID, RemindAt: now.Add(48 * time.Hour),
		ReminderType: ReminderTypeCustom, Subject: "Noch nicht faellig", Status: ReminderStatusPending, CreatedAt: now,
	}
	for _, r := range []*ContractReminder{due, future} {
		if err := repo.CreateReminder(ctx, r); err != nil {
			t.Fatalf("CreateReminder %s: %v", r.Subject, err)
		}
		defer testutil.CleanupRow(t, pool, "contract_reminders", r.ID)
	}

	// Same as ExpireContracts: a cross-tenant worker query, only visible under
	// a system context.
	claimed, err := repo.ClaimDueReminders(testutil.WithSystemCtx(context.Background()))
	if err != nil {
		t.Fatalf("ClaimDueReminders: %v", err)
	}
	var claimedDue bool
	for _, r := range claimed {
		if r.ID == due.ID {
			claimedDue = true
			if r.Status != ReminderStatusSent || r.SentAt == nil {
				t.Fatalf("ClaimDueReminders: expected due reminder marked sent with sent_at, got %+v", r)
			}
		}
		if r.ID == future.ID {
			t.Fatalf("ClaimDueReminders: future reminder must not be claimed")
		}
	}
	if !claimedDue {
		t.Fatalf("ClaimDueReminders: expected due reminder to be claimed, got %v", claimed)
	}

	// The worker ticks every five minutes for as long as the process runs —
	// a second real-SQL claim in the same window must not re-select the
	// reminder it already claimed (the WHERE status='pending' clause on the
	// UPDATE is what makes this safe, not process-level state). This is the
	// DB-level counterpart to the mock-repo proof in
	// TestReminderWorker_EmitsEventForDueReminder.
	reclaimed, err := repo.ClaimDueReminders(testutil.WithSystemCtx(context.Background()))
	if err != nil {
		t.Fatalf("ClaimDueReminders (second run): %v", err)
	}
	for _, r := range reclaimed {
		if r.ID == due.ID {
			t.Fatalf("ClaimDueReminders (second run): already-sent reminder was claimed again: %+v", r)
		}
	}

	stillPending, err := repo.GetReminder(ctx, tenantID, future.ID)
	if err != nil {
		t.Fatalf("GetReminder future: %v", err)
	}
	if stillPending.Status != ReminderStatusPending {
		t.Fatalf("ClaimDueReminders: future reminder status changed unexpectedly to %s", stillPending.Status)
	}

	// MarkReminderSent on an already-claimed reminder updates sent_at again —
	// used by the worker after the send actually succeeds downstream.
	// timestamptz only carries microsecond precision; truncate before the
	// round trip so the post-read comparison isn't chasing lost nanoseconds.
	laterSentAt := now.Add(2 * time.Hour).Truncate(time.Microsecond)
	if err := repo.MarkReminderSent(testutil.WithSystemCtx(context.Background()), due.ID, laterSentAt); err != nil {
		t.Fatalf("MarkReminderSent: %v", err)
	}
	final, err := repo.GetReminder(ctx, tenantID, due.ID)
	if err != nil {
		t.Fatalf("GetReminder after MarkReminderSent: %v", err)
	}
	if final.SentAt == nil || !final.SentAt.Equal(laterSentAt) {
		t.Fatalf("MarkReminderSent: expected sent_at %v, got %v", laterSentAt, final.SentAt)
	}
}

func TestRepository_ContractEventsCreateAndList(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Vertraege Events Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vertraege Events Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c := &Contract{
		ID: uuid.New(), TenantID: tenantOwn, ContractNumber: "EVT-" + uuid.New().String()[:8],
		Title: "Mit Verlauf", ContractType: ContractTypeService, Status: ContractStatusActive,
		StartsOn: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateContract(ctxOwn, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contracts", c.ID)

	// A nil payload must be stored as '{}', not SQL NULL — the column is NOT
	// NULL DEFAULT '{}'.
	noPayload := &ContractEvent{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: c.ID,
		Action: ContractEventCreated, Payload: nil, CreatedAt: now,
	}
	if err := repo.CreateContractEvent(ctxOwn, noPayload); err != nil {
		t.Fatalf("CreateContractEvent (nil payload): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_events", noPayload.ID)

	withPayload := &ContractEvent{
		ID: uuid.New(), TenantID: tenantOwn, ContractID: c.ID,
		Action: ContractEventUpdated, Payload: map[string]any{"fields": []string{"title"}}, CreatedAt: now.Add(time.Second),
	}
	if err := repo.CreateContractEvent(ctxOwn, withPayload); err != nil {
		t.Fatalf("CreateContractEvent (with payload): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contract_events", withPayload.ID)

	events, total, err := repo.ListContractEvents(ctxOwn, tenantOwn, c.ID, 0, 20)
	if err != nil {
		t.Fatalf("ListContractEvents: %v", err)
	}
	if total != 2 {
		t.Fatalf("ListContractEvents: expected total=2, got %d", total)
	}
	// Newest first (created_at DESC).
	if len(events) != 2 || events[0].ID != withPayload.ID || events[1].ID != noPayload.ID {
		t.Fatalf("ListContractEvents: expected [withPayload, noPayload] newest first, got %v", events)
	}
	if events[1].Payload == nil {
		t.Fatalf("ListContractEvents: nil payload was not normalized to an empty object")
	}
	if len(events[1].Payload) != 0 {
		t.Fatalf("ListContractEvents: expected empty payload map, got %v", events[1].Payload)
	}
	fields, ok := events[0].Payload["fields"].([]any)
	if !ok || len(fields) != 1 || fields[0] != "title" {
		t.Fatalf("ListContractEvents: expected payload fields=[title], got %v", events[0].Payload["fields"])
	}

	// Pagination.
	page, pageTotal, err := repo.ListContractEvents(ctxOwn, tenantOwn, c.ID, 0, 1)
	if err != nil {
		t.Fatalf("ListContractEvents (page 1): %v", err)
	}
	if pageTotal != 2 || len(page) != 1 || page[0].ID != withPayload.ID {
		t.Fatalf("ListContractEvents (page 1): expected total=2 first=withPayload, got total=%d page=%v", pageTotal, page)
	}

	// Tenant scoping.
	foreignEvents, foreignTotal, err := repo.ListContractEvents(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther, c.ID, 0, 20)
	if err != nil {
		t.Fatalf("ListContractEvents (foreign tenant): %v", err)
	}
	if foreignTotal != 0 || len(foreignEvents) != 0 {
		t.Fatalf("ListContractEvents (foreign tenant): expected no events visible, got total=%d events=%v", foreignTotal, foreignEvents)
	}
}
