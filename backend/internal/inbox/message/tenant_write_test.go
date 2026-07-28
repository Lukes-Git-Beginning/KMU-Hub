package message

// inbox_messages carries tenant_id NOT NULL and has had an RLS policy since
// migration 000122 (enable_tenant_rls). But most write methods on
// PostgresRepository -- MarkRead, MarkUnread, ToggleStar, SetStatus, AddTag,
// RemoveTag, Archive, Unarchive, Snooze, AssignMessage, BulkMarkRead,
// BulkArchive, and Update -- ran with no tenant_id predicate at all (`WHERE
// id = $1`, sometimes `WHERE id = ANY($1)`), relying purely on RLS as the
// only backstop. GetByID/GetBySourceID didn't even select tenant_id, so a
// message fetched through the service layer carried a zero-value TenantID.
// This test exercises every one of those methods from a foreign-tenant ctx,
// passing the row's *real* tenantID as the explicit parameter -- so only RLS,
// not a WHERE clause, could be stopping the write before this fix -- before
// repeating the same call from the owning ctx and confirming it lands.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestMessageWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Inbox Message Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Inbox Message Write Other Tenant")

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("inbox-msg-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	assigneeOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("inbox-msg-assignee-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", assigneeOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	msg := &models.InboxMessage{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		UserID:     userOwn,
		Channel:    "email",
		SourceID:   "src-" + uuid.New().String()[:8],
		SenderName: "Write Test Sender",
		Subject:    "Original Subject",
		Preview:    "Original preview",
		Tags:       []string{},
		ReceivedAt: now,
	}
	if err := repo.Create(ctxOwn, msg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "inbox_messages", msg.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "inbox_messages", msg.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "inbox_messages", msg.ID, 0)

	// MarkRead
	if err := repo.MarkRead(ctxOther, tenantOwn, msg.ID); err == nil {
		t.Fatalf("MarkRead (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.IsRead {
		t.Fatalf("a foreign-tenant MarkRead reached the message")
	}
	if err := repo.MarkRead(ctxOwn, tenantOwn, msg.ID); err != nil {
		t.Fatalf("MarkRead (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if !got.IsRead {
		t.Fatalf("own-tenant MarkRead did not land")
	}

	// MarkUnread
	if err := repo.MarkUnread(ctxOther, tenantOwn, msg.ID); err == nil {
		t.Fatalf("MarkUnread (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if !got.IsRead {
		t.Fatalf("a foreign-tenant MarkUnread reached the message")
	}
	if err := repo.MarkUnread(ctxOwn, tenantOwn, msg.ID); err != nil {
		t.Fatalf("MarkUnread (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.IsRead {
		t.Fatalf("own-tenant MarkUnread did not land")
	}

	// ToggleStar
	if err := repo.ToggleStar(ctxOther, tenantOwn, msg.ID); err == nil {
		t.Fatalf("ToggleStar (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.IsStarred {
		t.Fatalf("a foreign-tenant ToggleStar reached the message")
	}
	if err := repo.ToggleStar(ctxOwn, tenantOwn, msg.ID); err != nil {
		t.Fatalf("ToggleStar (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if !got.IsStarred {
		t.Fatalf("own-tenant ToggleStar did not land")
	}

	// SetStatus
	if err := repo.SetStatus(ctxOther, tenantOwn, msg.ID, "resolved"); err == nil {
		t.Fatalf("SetStatus (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.Status == "resolved" {
		t.Fatalf("a foreign-tenant SetStatus reached the message")
	}
	if err := repo.SetStatus(ctxOwn, tenantOwn, msg.ID, "resolved"); err != nil {
		t.Fatalf("SetStatus (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.Status != "resolved" {
		t.Fatalf("own-tenant SetStatus did not land: status=%q", got.Status)
	}

	// AddTag / RemoveTag
	if err := repo.AddTag(ctxOther, tenantOwn, msg.ID, "urgent"); err == nil {
		t.Fatalf("AddTag (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if len(got.Tags) != 0 {
		t.Fatalf("a foreign-tenant AddTag reached the message: tags=%v", got.Tags)
	}
	if err := repo.AddTag(ctxOwn, tenantOwn, msg.ID, "urgent"); err != nil {
		t.Fatalf("AddTag (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Fatalf("own-tenant AddTag did not land: tags=%v", got.Tags)
	}
	if err := repo.RemoveTag(ctxOther, tenantOwn, msg.ID, "urgent"); err == nil {
		t.Fatalf("RemoveTag (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if len(got.Tags) != 1 {
		t.Fatalf("a foreign-tenant RemoveTag reached the message: tags=%v", got.Tags)
	}
	if err := repo.RemoveTag(ctxOwn, tenantOwn, msg.ID, "urgent"); err != nil {
		t.Fatalf("RemoveTag (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if len(got.Tags) != 0 {
		t.Fatalf("own-tenant RemoveTag did not land: tags=%v", got.Tags)
	}

	// Archive / Unarchive
	if err := repo.Archive(ctxOther, tenantOwn, msg.ID); err == nil {
		t.Fatalf("Archive (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.IsArchived {
		t.Fatalf("a foreign-tenant Archive reached the message")
	}
	if err := repo.Archive(ctxOwn, tenantOwn, msg.ID); err != nil {
		t.Fatalf("Archive (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if !got.IsArchived {
		t.Fatalf("own-tenant Archive did not land")
	}
	if err := repo.Unarchive(ctxOther, tenantOwn, msg.ID); err == nil {
		t.Fatalf("Unarchive (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if !got.IsArchived {
		t.Fatalf("a foreign-tenant Unarchive reached the message")
	}
	if err := repo.Unarchive(ctxOwn, tenantOwn, msg.ID); err != nil {
		t.Fatalf("Unarchive (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.IsArchived {
		t.Fatalf("own-tenant Unarchive did not land")
	}

	// Snooze
	until := time.Now().Add(1 * time.Hour).UTC()
	if err := repo.Snooze(ctxOther, tenantOwn, msg.ID, until); err == nil {
		t.Fatalf("Snooze (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.SnoozedUntil != nil {
		t.Fatalf("a foreign-tenant Snooze reached the message")
	}
	if err := repo.Snooze(ctxOwn, tenantOwn, msg.ID, until); err != nil {
		t.Fatalf("Snooze (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.SnoozedUntil == nil {
		t.Fatalf("own-tenant Snooze did not land")
	}

	// AssignMessage
	if assigned, err := repo.AssignMessage(ctxOther, tenantOwn, msg.ID, assigneeOwn); err == nil && assigned {
		t.Fatalf("AssignMessage (foreign ctx): expected no assignment, got assigned=true err=nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.AssignedTo != nil {
		t.Fatalf("a foreign-tenant AssignMessage reached the message")
	}
	assigned, err := repo.AssignMessage(ctxOwn, tenantOwn, msg.ID, assigneeOwn)
	if err != nil {
		t.Fatalf("AssignMessage (own ctx): %v", err)
	}
	if !assigned {
		t.Fatalf("own-tenant AssignMessage did not report success")
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.AssignedTo == nil || *got.AssignedTo != assigneeOwn {
		t.Fatalf("own-tenant AssignMessage did not land: assigned_to=%v", got.AssignedTo)
	}

	// Update — msg carries its real TenantID from GetByID, mirroring how the
	// routing package (routing.Service.actionRouteToTeam etc.) mutates a
	// message and writes it back through the same Update method.
	toUpdate, _ := repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	toUpdate.Subject = "Hacked Subject"
	if err := repo.Update(ctxOther, toUpdate); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByID(sysCtx, msg.ID, tenantOwn)
	if got.Subject == "Hacked Subject" {
		t.Fatalf("a foreign-tenant Update reached the message")
	}
	toUpdate.Subject = "Updated Subject"
	if err := repo.Update(ctxOwn, toUpdate); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, msg.ID, tenantOwn)
	if got.Subject != "Updated Subject" {
		t.Fatalf("own-tenant Update did not land: subject=%q", got.Subject)
	}
}

// TestBulkMessageWrites_LandInCallerTenant covers BulkMarkRead and
// BulkArchive separately: unlike the single-message writes above, these
// return no NotFound error on a no-op match, so the isolation check has to
// read the rows back instead of asserting on the returned error.
func TestBulkMessageWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Inbox Bulk Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Inbox Bulk Write Other Tenant")

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("inbox-bulk-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	var ids []uuid.UUID
	for range 2 {
		msg := &models.InboxMessage{
			ID:         uuid.New(),
			TenantID:   tenantOwn,
			UserID:     userOwn,
			Channel:    "email",
			SourceID:   fmt.Sprintf("bulk-src-%s", uuid.New().String()[:8]),
			Tags:       []string{},
			ReceivedAt: now,
		}
		if err := repo.Create(ctxOwn, msg); err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer testutil.CleanupRow(t, pool, "inbox_messages", msg.ID)
		ids = append(ids, msg.ID)
	}

	if err := repo.BulkMarkRead(ctxOther, tenantOwn, ids); err != nil {
		t.Fatalf("BulkMarkRead (foreign ctx): %v", err)
	}
	for _, id := range ids {
		got, err := repo.GetByID(sysCtx, id, tenantOwn)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.IsRead {
			t.Fatalf("a foreign-tenant BulkMarkRead reached message %s", id)
		}
	}
	if err := repo.BulkMarkRead(ctxOwn, tenantOwn, ids); err != nil {
		t.Fatalf("BulkMarkRead (own ctx): %v", err)
	}
	for _, id := range ids {
		got, _ := repo.GetByID(ctxOwn, id, tenantOwn)
		if !got.IsRead {
			t.Fatalf("own-tenant BulkMarkRead did not land for message %s", id)
		}
	}

	if err := repo.BulkArchive(ctxOther, tenantOwn, ids); err != nil {
		t.Fatalf("BulkArchive (foreign ctx): %v", err)
	}
	for _, id := range ids {
		got, _ := repo.GetByID(sysCtx, id, tenantOwn)
		if got.IsArchived {
			t.Fatalf("a foreign-tenant BulkArchive reached message %s", id)
		}
	}
	if err := repo.BulkArchive(ctxOwn, tenantOwn, ids); err != nil {
		t.Fatalf("BulkArchive (own ctx): %v", err)
	}
	for _, id := range ids {
		got, _ := repo.GetByID(ctxOwn, id, tenantOwn)
		if !got.IsArchived {
			t.Fatalf("own-tenant BulkArchive did not land for message %s", id)
		}
	}
}
