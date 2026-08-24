package rule_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/email/rule"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// mailbox is the fixture chain a rule needs to exist against: tenant -> user ->
// account -> folders -> messages. email_accounts is UNIQUE(user_id), so every
// tenant in these tests gets its own freshly minted user.
type mailbox struct {
	tenant  uuid.UUID
	inbox   uuid.UUID
	archive uuid.UUID
	trash   uuid.UUID
}

// seedMailbox builds one tenant's mailbox. Tenants are minted per test rather
// than reusing testutil.TenantA/B: these fixtures hit UNIQUE(user_id) on
// email_accounts and UNIQUE(account_id, imap_name) on email_folders, and shared
// constants collide under t.Parallel() (the failure mode from night run 1).
func seedMailbox(t *testing.T, pool *pgxpool.Pool, label string) mailbox {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "email rule test "+label)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         "rule-" + label + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x",
		"first_name":    "Rule",
		"last_name":     "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"id":                 uuid.New(),
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      "rule-" + label + "@test.local",
		"imap_host":          "imap.test.local",
		"smtp_host":          "smtp.test.local",
		"username":           "rule-" + label,
		"password_encrypted": "x",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })

	mb := mailbox{tenant: tenantID}
	for _, f := range []struct {
		target     *uuid.UUID
		name, kind string
	}{
		{&mb.inbox, "Posteingang", "inbox"},
		{&mb.archive, "Archiv", "archive"},
		{&mb.trash, "Papierkorb", "trash"},
	} {
		*f.target = testutil.SeedRow(t, pool, "email_folders", map[string]any{
			"id":          uuid.New(),
			"tenant_id":   tenantID,
			"account_id":  accountID,
			"name":        f.name,
			"imap_name":   f.kind,
			"folder_type": f.kind,
		})
	}
	// Folders cascade with the account; messages do too.

	seedMessage := func(folderID uuid.UUID, uid int64, subject, fromName, fromEmail string) uuid.UUID {
		return testutil.SeedRow(t, pool, "email_messages", map[string]any{
			"id":         uuid.New(),
			"tenant_id":  tenantID,
			"account_id": accountID,
			"folder_id":  folderID,
			"uid":        uid,
			"from_name":  fromName,
			"from_email": fromEmail,
			"subject":    subject,
			"date":       time.Now().UTC(),
		})
	}
	seedMessage(mb.inbox, 1, "Ihre RECHNUNG 2026-04", "Buchhaltung", "buchhaltung@example.de")
	seedMessage(mb.inbox, 2, "Mittagessen?", "Ben", "ben@example.de")
	seedMessage(mb.trash, 3, "Rechnung alt", "Buchhaltung", "buchhaltung@example.de")

	return mb
}

// TestRepository_CreateAndListAreTenantScoped covers the read and the write
// side together: a rule written under one tenant must be invisible to another.
func TestRepository_CreateAndListAreTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := rule.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "own")
	other := seedMailbox(t, pool, "other")

	ruleID := uuid.New()
	now := time.Now().UTC()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	err := repo.Create(ctxMine, &models.EmailRule{
		ID: ruleID, TenantID: mine.tenant, Name: "Rechnungen", Field: models.EmailRuleFieldSubject,
		Op: models.EmailRuleOpContains, Value: "Rechnung",
		ActionType: models.EmailRuleActionMove, ActionTarget: mine.archive,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_rules", ruleID) })

	mineRules, err := repo.List(ctxMine, mine.tenant)
	if err != nil {
		t.Fatalf("List own: %v", err)
	}
	if len(mineRules) != 1 || mineRules[0].ID != ruleID {
		t.Fatalf("own list = %d rules, want the one just written", len(mineRules))
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	otherRules, err := repo.List(ctxOther, other.tenant)
	if err != nil {
		t.Fatalf("List other: %v", err)
	}
	if len(otherRules) != 0 {
		t.Fatalf("foreign tenant sees %d rules, want 0", len(otherRules))
	}
	if _, err := repo.GetByID(ctxOther, ruleID, other.tenant); !errors.Is(err, rule.ErrRuleNotFound) {
		t.Fatalf("GetByID across tenants: %v, want ErrRuleNotFound", err)
	}
}

// TestRepository_FolderBelongsToTenant guards the move-target check. The
// folder itself carries no tenant_id (migration 000041), so the answer is only
// correct if the join through email_accounts holds.
func TestRepository_FolderBelongsToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := rule.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "folder-own")
	other := seedMailbox(t, pool, "folder-other")

	sysCtx := testutil.WithSystemCtx(context.Background())

	ok, err := repo.FolderBelongsToTenant(sysCtx, mine.archive, mine.tenant)
	if err != nil || !ok {
		t.Fatalf("own folder: ok=%v err=%v, want true", ok, err)
	}
	ok, err = repo.FolderBelongsToTenant(sysCtx, other.archive, mine.tenant)
	if err != nil || ok {
		t.Fatalf("foreign folder: ok=%v err=%v, want false", ok, err)
	}
	ok, err = repo.FolderBelongsToTenant(sysCtx, uuid.New(), mine.tenant)
	if err != nil || ok {
		t.Fatalf("unknown folder: ok=%v err=%v, want false", ok, err)
	}
}

// TestRepository_LabelBelongsToTenant guards the label action's target check,
// mirroring TestRepository_FolderBelongsToTenant for the move side.
func TestRepository_LabelBelongsToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := rule.NewPostgresRepository(pool)
	mineTenant := uuid.New()
	testutil.EnsureTenant(t, pool, mineTenant, "email rule label test own")
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "email rule label test other")

	mineLabel := testutil.SeedRow(t, pool, "email_labels", map[string]any{
		"id": uuid.New(), "tenant_id": mineTenant, "name": "Wichtig", "color": "#EF4444",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", mineLabel) })
	otherLabel := testutil.SeedRow(t, pool, "email_labels", map[string]any{
		"id": uuid.New(), "tenant_id": otherTenant, "name": "Wichtig", "color": "#EF4444",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", otherLabel) })

	sysCtx := testutil.WithSystemCtx(context.Background())

	ok, err := repo.LabelBelongsToTenant(sysCtx, mineLabel, mineTenant)
	if err != nil || !ok {
		t.Fatalf("own label: ok=%v err=%v, want true", ok, err)
	}
	ok, err = repo.LabelBelongsToTenant(sysCtx, otherLabel, mineTenant)
	if err != nil || ok {
		t.Fatalf("foreign label: ok=%v err=%v, want false", ok, err)
	}
	ok, err = repo.LabelBelongsToTenant(sysCtx, uuid.New(), mineTenant)
	if err != nil || ok {
		t.Fatalf("unknown label: ok=%v err=%v, want false", ok, err)
	}
}

// TestRepository_ApplyCandidatesAndWrite exercises the two queries the unit
// tests cannot reach: the trash-excluding candidate scan and the UUID[]
// round-trip of label_ids.
func TestRepository_ApplyCandidatesAndWrite(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := rule.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "apply")
	ctx := testutil.WithTenantCtx(context.Background(), mine.tenant)

	candidates, err := repo.ListApplyCandidates(ctx, mine.tenant, 100)
	if err != nil {
		t.Fatalf("ListApplyCandidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d, want 2 (the trashed message must be excluded)", len(candidates))
	}
	for _, c := range candidates {
		if c.FolderID == mine.trash {
			t.Fatalf("trashed message %s was returned", c.ID)
		}
		if len(c.LabelIDs) != 0 {
			t.Fatalf("fresh message already has labels: %v", c.LabelIDs)
		}
	}

	target := candidates[0]
	label := uuid.New()
	if err := repo.ApplyToMessage(ctx, mine.tenant, target.ID, mine.archive, []uuid.UUID{label}); err != nil {
		t.Fatalf("ApplyToMessage: %v", err)
	}

	after, err := repo.ListApplyCandidates(ctx, mine.tenant, 100)
	if err != nil {
		t.Fatalf("ListApplyCandidates after write: %v", err)
	}
	idx := slices.IndexFunc(after, func(c *models.EmailRuleCandidate) bool { return c.ID == target.ID })
	if idx < 0 {
		t.Fatalf("message %s vanished from the candidate set", target.ID)
	}
	if after[idx].FolderID != mine.archive {
		t.Errorf("folder = %s, want %s", after[idx].FolderID, mine.archive)
	}
	if !slices.Contains(after[idx].LabelIDs, label) {
		t.Errorf("label_ids = %v, want to contain %s", after[idx].LabelIDs, label)
	}
}

// TestRepository_ApplyToMessageIsTenantScoped proves the write cannot reach a
// foreign tenant's message even when the id is known.
func TestRepository_ApplyToMessageIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := rule.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "write-own")
	other := seedMailbox(t, pool, "write-other")

	otherCtx := testutil.WithTenantCtx(context.Background(), other.tenant)
	victims, err := repo.ListApplyCandidates(otherCtx, other.tenant, 1)
	if err != nil || len(victims) == 0 {
		t.Fatalf("seed candidates for the other tenant: %d, err=%v", len(victims), err)
	}
	victim := victims[0]

	mineCtx := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.ApplyToMessage(mineCtx, mine.tenant, victim.ID, mine.archive, []uuid.UUID{uuid.New()}); err != nil {
		t.Fatalf("cross-tenant ApplyToMessage errored instead of matching zero rows: %v", err)
	}

	after, err := repo.ListApplyCandidates(otherCtx, other.tenant, 100)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	idx := slices.IndexFunc(after, func(c *models.EmailRuleCandidate) bool { return c.ID == victim.ID })
	if idx < 0 {
		t.Fatalf("victim message vanished")
	}
	if after[idx].FolderID != victim.FolderID || len(after[idx].LabelIDs) != 0 {
		t.Fatalf("foreign tenant's message was modified: %+v", after[idx])
	}
}
