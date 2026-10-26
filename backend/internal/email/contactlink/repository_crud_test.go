package contactlink

// Repository behaviour against the app-role database (kmuhub_app, RLS on):
// the write path with generated defaults and the conflict rule, the ordered
// per-message read, the tenant-scoped delete and the error paths.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCreate_GeneratesDefaultsAndIgnoresDuplicatePair_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "contactlink create")
	messageID, contactID := seedMessageAndContact(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	before := time.Now().UTC().Add(-time.Second)
	link := &models.EmailContactLink{
		TenantID: tenantID, MessageID: messageID, ContactID: contactID, LinkType: models.LinkTypeAuto,
	}
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_contact_links", link.ID) })

	if link.ID == uuid.Nil {
		t.Fatal("Create left the zero ID in place")
	}
	if link.CreatedAt.Before(before) {
		t.Fatalf("Create did not stamp CreatedAt: %v", link.CreatedAt)
	}

	got, err := repo.GetByMessageID(ctx, messageID, tenantID)
	if err != nil {
		t.Fatalf("get by message: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d links, want 1", len(got))
	}
	if got[0].ID != link.ID || got[0].TenantID != tenantID || got[0].MessageID != messageID ||
		got[0].ContactID != contactID || got[0].LinkType != models.LinkTypeAuto {
		t.Fatalf("stored link = %+v, want the created one %+v", got[0], link)
	}

	// The same (message, contact) pair again — a manual link this time — is
	// swallowed by ON CONFLICT DO NOTHING: no error, no second row, the first
	// link type stays.
	dup := &models.EmailContactLink{
		TenantID: tenantID, MessageID: messageID, ContactID: contactID, LinkType: models.LinkTypeManual,
	}
	if err := repo.Create(ctx, dup); err != nil {
		t.Fatalf("duplicate create must not fail: %v", err)
	}
	got, err = repo.GetByMessageID(ctx, messageID, tenantID)
	if err != nil {
		t.Fatalf("get after duplicate: %v", err)
	}
	if len(got) != 1 || got[0].LinkType != models.LinkTypeAuto {
		t.Fatalf("after duplicate = %d link(s) (type %q), want the single original auto link", len(got), got[0].LinkType)
	}
}

func TestCreate_RejectsForeignTenantRow_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	owner, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, owner, "contactlink create owner")
	testutil.EnsureTenant(t, pool, other, "contactlink create other")
	messageID, contactID := seedMessageAndContact(t, pool, owner)

	repo := NewPostgresRepository(pool)
	// A link stamped with the owner's tenant written under another tenant's
	// context is refused by the RLS policy, not stored.
	link := &models.EmailContactLink{
		TenantID: owner, MessageID: messageID, ContactID: contactID, LinkType: models.LinkTypeManual,
	}
	if err := repo.Create(testutil.WithTenantCtx(context.Background(), other), link); err == nil {
		testutil.CleanupRow(t, pool, "email_contact_links", link.ID)
		t.Fatal("create with a foreign tenant_id succeeded, want an RLS rejection")
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "email_contact_links", link.ID, 0)
}

func TestGetByMessageID_OrdersByCreatedAtAndScopesTenant_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "contactlink order")
	messageID, firstContact := seedMessageAndContact(t, pool, tenantID)
	// A second contact on the same message: seedMessageAndContact builds a
	// whole chain, so reuse its tenant user via a contact seeded directly.
	creator := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "order-" + uuid.NewString()[:8] + "@example.local", "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", creator) })
	secondContact := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Second", "last_name": "Contact", "created_by": creator,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", secondContact) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	// Insert the newer link first so insertion order cannot pass for the sort.
	newer := &models.EmailContactLink{
		TenantID: tenantID, MessageID: messageID, ContactID: secondContact,
		LinkType: models.LinkTypeManual, CreatedAt: now,
	}
	older := &models.EmailContactLink{
		TenantID: tenantID, MessageID: messageID, ContactID: firstContact,
		LinkType: models.LinkTypeAuto, CreatedAt: now.Add(-time.Hour),
	}
	for _, l := range []*models.EmailContactLink{newer, older} {
		if err := repo.Create(ctx, l); err != nil {
			t.Fatalf("create: %v", err)
		}
		l := l
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_contact_links", l.ID) })
	}

	got, err := repo.GetByMessageID(ctx, messageID, tenantID)
	if err != nil {
		t.Fatalf("get by message: %v", err)
	}
	if len(got) != 2 || got[0].ID != older.ID || got[1].ID != newer.ID {
		t.Fatalf("links = %v, want oldest first [%s %s]", got, older.ID, newer.ID)
	}

	// An unknown message yields an empty result, not an error.
	none, err := repo.GetByMessageID(ctx, uuid.New(), tenantID)
	if err != nil || len(none) != 0 {
		t.Fatalf("unknown message = %d link(s), err %v; want none", len(none), err)
	}

	// The explicit tenant predicate holds on its own: in system context RLS is
	// open, so only the WHERE clause can keep a foreign tenant out.
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "contactlink order other")
	sys := testutil.WithSystemCtx(context.Background())
	foreign, err := repo.GetByMessageID(sys, messageID, other)
	if err != nil || len(foreign) != 0 {
		t.Fatalf("foreign tenant id in system context = %d link(s), err %v; want none", len(foreign), err)
	}
}

func TestGetByContactID_ScopesTenantPredicate_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "contactlink contact scope")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "scope-" + uuid.NewString()[:8] + "@example.local", "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Scope", "last_name": "Target", "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })
	seedLinkedMail(t, pool, tenantID, userID, contactID, time.Now().UTC())

	repo := NewPostgresRepository(pool)

	// RLS is open in system context, so a foreign tenant ID only stays out
	// because of the explicit predicates on the link and the message.
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "contactlink contact scope other")
	sys := testutil.WithSystemCtx(context.Background())
	foreign, total, err := repo.GetByContactID(sys, contactID, other, nil, 1, 50)
	if err != nil || len(foreign) != 0 || total != 0 {
		t.Fatalf("foreign tenant id in system context = %d message(s), total %d, err %v; want none", len(foreign), total, err)
	}
	own, total, err := repo.GetByContactID(sys, contactID, tenantID, nil, 1, 50)
	if err != nil || len(own) != 1 || total != 1 {
		t.Fatalf("own tenant in system context = %d message(s), total %d, err %v; want 1 and 1", len(own), total, err)
	}
	// The scanned columns survive the join.
	if own[0].Subject != "Angebot" || own[0].FromEmail != "kontakt@example.com" {
		t.Fatalf("scanned message = %+v, want subject Angebot from kontakt@example.com", own[0])
	}

	// A page past the end is empty but still reports the total.
	past, total, err := repo.GetByContactID(sys, contactID, tenantID, nil, 5, 50)
	if err != nil || len(past) != 0 || total != 1 {
		t.Fatalf("page past the end = %d message(s), total %d, err %v; want none and total 1", len(past), total, err)
	}
}

func TestDelete_RemovesOnlyTheTenantsPair_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "contactlink delete")
	messageID, contactID := seedMessageAndContact(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	link := &models.EmailContactLink{
		TenantID: tenantID, MessageID: messageID, ContactID: contactID, LinkType: models.LinkTypeManual,
	}
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_contact_links", link.ID) })

	sys := testutil.WithSystemCtx(context.Background())
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "contactlink delete other")

	// A foreign tenant ID deletes nothing, even where RLS is open.
	if err := repo.Delete(sys, messageID, contactID, other); err != nil {
		t.Fatalf("delete as foreign tenant: %v", err)
	}
	testutil.AssertRowCount(t, pool, sys, "email_contact_links", link.ID, 1)

	// Another contact on the same message is not touched by a delete of a
	// non-existent pair.
	if err := repo.Delete(ctx, messageID, uuid.New(), tenantID); err != nil {
		t.Fatalf("delete of an unknown pair must not fail: %v", err)
	}
	testutil.AssertRowCount(t, pool, sys, "email_contact_links", link.ID, 1)

	// The same contact linked to a second message is not touched either.
	secondMsg, _ := seedMessageAndContact(t, pool, tenantID)
	sibling := &models.EmailContactLink{
		TenantID: tenantID, MessageID: secondMsg, ContactID: contactID, LinkType: models.LinkTypeManual,
	}
	if err := repo.Create(ctx, sibling); err != nil {
		t.Fatalf("create sibling: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_contact_links", sibling.ID) })

	if err := repo.Delete(ctx, messageID, contactID, tenantID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	testutil.AssertRowCount(t, pool, sys, "email_contact_links", link.ID, 0)
	testutil.AssertRowCount(t, pool, sys, "email_contact_links", sibling.ID, 1)
}

// Each of the two tenant predicates in the contact history query must hold on
// its own: a link row stamped with one tenant on a message of another is
// listed for neither. Such rows cannot arise through the repository (RLS), so
// they are forged in system context.
func TestGetByContactID_LinkAndMessageTenantMustBothMatch_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "contactlink mismatch A")
	testutil.EnsureTenant(t, pool, tenantB, "contactlink mismatch B")
	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantA, "email": "mismatch-" + uuid.NewString()[:8] + "@example.local", "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userA) })
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantB, "email": "mismatch-" + uuid.NewString()[:8] + "@example.local", "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantA, "first_name": "Mismatch", "last_name": "Target", "created_by": userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	// Message of tenant A, link stamped tenant B; and message of tenant B, link
	// stamped tenant A.
	linkedMsgA := seedLinkedMail(t, pool, tenantA, userA, contactID, time.Now().UTC())
	linkedMsgB := seedLinkedMail(t, pool, tenantB, userB, contactID, time.Now().UTC())
	sys := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sys, `UPDATE email_contact_links SET tenant_id = $1 WHERE message_id = $2`, tenantB, linkedMsgA); err != nil {
		t.Fatalf("forge link tenant: %v", err)
	}
	if _, err := pool.Exec(sys, `UPDATE email_contact_links SET tenant_id = $1 WHERE message_id = $2`, tenantA, linkedMsgB); err != nil {
		t.Fatalf("forge link tenant: %v", err)
	}

	repo := NewPostgresRepository(pool)
	for _, tenant := range []uuid.UUID{tenantA, tenantB} {
		got, total, err := repo.GetByContactID(sys, contactID, tenant, nil, 1, 50)
		if err != nil || len(got) != 0 || total != 0 {
			t.Fatalf("tenant %s lists %d message(s), total %d, err %v; want none for mismatched link/message tenants", tenant, len(got), total, err)
		}
	}
}

func TestRepository_ErrorsOnClosedPool_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	repo := NewPostgresRepository(pool)
	pool.Close()

	ctx := testutil.WithTenantCtx(context.Background(), uuid.New())
	tenantID, id := uuid.New(), uuid.New()

	if err := repo.Create(ctx, &models.EmailContactLink{TenantID: tenantID, MessageID: id, ContactID: id, LinkType: models.LinkTypeAuto}); err == nil {
		t.Error("Create on a closed pool returned nil")
	}
	if _, err := repo.GetByMessageID(ctx, id, tenantID); err == nil {
		t.Error("GetByMessageID on a closed pool returned nil")
	}
	if _, _, err := repo.GetByContactID(ctx, id, tenantID, nil, 1, 10); err == nil {
		t.Error("GetByContactID on a closed pool returned nil")
	}
	if err := repo.Delete(ctx, id, id, tenantID); err == nil {
		t.Error("Delete on a closed pool returned nil")
	}
}
