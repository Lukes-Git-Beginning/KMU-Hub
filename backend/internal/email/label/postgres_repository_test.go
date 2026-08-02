package label_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/email/label"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// mailbox mirrors the fixture chain in internal/email/rule's tests: tenant ->
// user -> account -> folder -> message. Tenants are minted per test rather
// than reusing testutil.TenantA/B -- these fixtures hit UNIQUE(user_id) on
// email_accounts, and shared constants collide under t.Parallel().
type mailbox struct {
	tenant  uuid.UUID
	folder  uuid.UUID
	message uuid.UUID
}

func seedMailbox(t *testing.T, pool *pgxpool.Pool, name string) mailbox {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "email label test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "label-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Label", "last_name": "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"email_address": "label-" + name + "@test.local",
		"imap_host":     "imap.test.local", "smtp_host": "smtp.test.local",
		"username": "label-" + name, "password_encrypted": "x",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })

	folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID,
		"name": "Posteingang", "imap_name": "inbox", "folder_type": "inbox",
	})

	messageID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID, "folder_id": folderID,
		"uid": 1, "from_name": "Sender", "from_email": "sender@example.de",
		"subject": "Hallo", "date": time.Now().UTC(),
	})

	return mailbox{tenant: tenantID, folder: folderID, message: messageID}
}

func TestRepository_CreateAndListAreTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "own")
	other := seedMailbox(t, pool, "other")

	labelID := uuid.New()
	now := time.Now().UTC()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	err := repo.Create(ctxMine, &models.EmailLabel{
		ID: labelID, TenantID: mine.tenant, Name: "Wichtig", Color: "#EF4444",
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", labelID) })

	mineLabels, err := repo.List(ctxMine, mine.tenant)
	if err != nil {
		t.Fatalf("List own: %v", err)
	}
	if len(mineLabels) != 1 || mineLabels[0].ID != labelID {
		t.Fatalf("own list = %d labels, want the one just written", len(mineLabels))
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	otherLabels, err := repo.List(ctxOther, other.tenant)
	if err != nil {
		t.Fatalf("List other: %v", err)
	}
	if len(otherLabels) != 0 {
		t.Fatalf("foreign tenant sees %d labels, want 0", len(otherLabels))
	}
	if _, err := repo.GetByID(ctxOther, labelID, other.tenant); !errors.Is(err, label.ErrNotFound) {
		t.Fatalf("GetByID across tenants: %v, want ErrNotFound", err)
	}
}

func TestRepository_DuplicateNameIsPerTenantNotGlobal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "dup-own")
	other := seedMailbox(t, pool, "dup-other")

	now := time.Now().UTC()
	first := uuid.New()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Create(ctxMine, &models.EmailLabel{
		ID: first, TenantID: mine.tenant, Name: "Wichtig", Color: "#EF4444", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", first) })

	dupe := uuid.New()
	err := repo.Create(ctxMine, &models.EmailLabel{
		ID: dupe, TenantID: mine.tenant, Name: "Wichtig", Color: "#3B82F6", CreatedAt: now, UpdatedAt: now,
	})
	if !errors.Is(err, label.ErrDuplicateName) {
		t.Fatalf("same-tenant duplicate: err = %v, want ErrDuplicateName", err)
	}

	// The same name under a different tenant must not collide -- the unique
	// index is per tenant, not global (migration 000261's whole point).
	otherLabel := uuid.New()
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	if err := repo.Create(ctxOther, &models.EmailLabel{
		ID: otherLabel, TenantID: other.tenant, Name: "Wichtig", Color: "#3B82F6", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("other tenant same name: %v, want success", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", otherLabel) })
}

func TestRepository_UpdateIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "update-own")
	other := seedMailbox(t, pool, "update-other")

	labelID := uuid.New()
	now := time.Now().UTC()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Create(ctxMine, &models.EmailLabel{
		ID: labelID, TenantID: mine.tenant, Name: "Wichtig", Color: "#EF4444", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", labelID) })

	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	err := repo.Update(ctxOther, &models.EmailLabel{
		ID: labelID, TenantID: other.tenant, Name: "Gekapert", Color: "#000000",
	})
	if !errors.Is(err, label.ErrNotFound) {
		t.Fatalf("cross-tenant update: err = %v, want ErrNotFound", err)
	}

	unchanged, err := repo.GetByID(ctxMine, labelID, mine.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if unchanged.Name != "Wichtig" {
		t.Fatalf("label was modified by a foreign tenant's update: %+v", unchanged)
	}
}

func TestRepository_LabelIDsBelongToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "belong-own")
	other := seedMailbox(t, pool, "belong-other")

	now := time.Now().UTC()
	mineLabel := uuid.New()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Create(ctxMine, &models.EmailLabel{
		ID: mineLabel, TenantID: mine.tenant, Name: "A", Color: "#EF4444", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create mine: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", mineLabel) })

	otherLabel := uuid.New()
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	if err := repo.Create(ctxOther, &models.EmailLabel{
		ID: otherLabel, TenantID: other.tenant, Name: "B", Color: "#3B82F6", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create other: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", otherLabel) })

	sysCtx := testutil.WithSystemCtx(context.Background())

	ok, err := repo.LabelIDsBelongToTenant(sysCtx, mine.tenant, []uuid.UUID{mineLabel})
	if err != nil || !ok {
		t.Fatalf("own label: ok=%v err=%v, want true", ok, err)
	}
	ok, err = repo.LabelIDsBelongToTenant(sysCtx, mine.tenant, []uuid.UUID{mineLabel, otherLabel})
	if err != nil || ok {
		t.Fatalf("mixed set with one foreign id: ok=%v err=%v, want false", ok, err)
	}
	ok, err = repo.LabelIDsBelongToTenant(sysCtx, mine.tenant, nil)
	if err != nil || !ok {
		t.Fatalf("empty set: ok=%v err=%v, want true (vacuously)", ok, err)
	}
}

func TestRepository_AssignToMessageIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "assign-own")
	other := seedMailbox(t, pool, "assign-other")

	now := time.Now().UTC()
	labelID := uuid.New()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Create(ctxMine, &models.EmailLabel{
		ID: labelID, TenantID: mine.tenant, Name: "A", Color: "#EF4444", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_labels", labelID) })

	// Cross-tenant assignment must not reach the other tenant's message.
	err := repo.AssignToMessage(ctxMine, mine.tenant, other.message, []uuid.UUID{labelID})
	if !errors.Is(err, label.ErrMessageNotFound) {
		t.Fatalf("cross-tenant assign: err = %v, want ErrMessageNotFound", err)
	}

	if err := repo.AssignToMessage(ctxMine, mine.tenant, mine.message, []uuid.UUID{labelID}); err != nil {
		t.Fatalf("own assign: %v", err)
	}

	var labelIDs []uuid.UUID
	if err := pool.QueryRow(ctxMine, `SELECT label_ids FROM email_messages WHERE id = $1`, mine.message).Scan(&labelIDs); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(labelIDs) != 1 || labelIDs[0] != labelID {
		t.Fatalf("label_ids = %v, want [%s]", labelIDs, labelID)
	}
}

// TestRepository_DeleteStripsLabelFromMessages proves the cascade the DB
// cannot express: label_ids has no foreign key, so a dangling id after
// deletion is a bug this repository has to prevent in code.
func TestRepository_DeleteStripsLabelFromMessages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := label.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "delete-strip")

	now := time.Now().UTC()
	labelID := uuid.New()
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Create(ctxMine, &models.EmailLabel{
		ID: labelID, TenantID: mine.tenant, Name: "A", Color: "#EF4444", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.AssignToMessage(ctxMine, mine.tenant, mine.message, []uuid.UUID{labelID}); err != nil {
		t.Fatalf("AssignToMessage: %v", err)
	}

	if err := repo.Delete(ctxMine, labelID, mine.tenant); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var labelIDs []uuid.UUID
	if err := pool.QueryRow(ctxMine, `SELECT label_ids FROM email_messages WHERE id = $1`, mine.message).Scan(&labelIDs); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(labelIDs) != 0 {
		t.Fatalf("label_ids after delete = %v, want empty (dangling reference)", labelIDs)
	}
}
