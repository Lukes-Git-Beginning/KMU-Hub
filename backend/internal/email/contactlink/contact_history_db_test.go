package contactlink

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedLinkedMail gives userID a further (non-default) mailbox with one message linked to contactID
// and returns the message ID. The nullable text columns are set as the sync
// engine always writes them: the history scan reads them as plain strings.
func seedLinkedMail(t *testing.T, pool *pgxpool.Pool, tenantID, userID, contactID uuid.UUID, date time.Time) uuid.UUID {
	t.Helper()
	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      fmt.Sprintf("history-%s@example.com", uuid.New().String()[:8]),
		"display_name":       "History",
		"imap_host":          "imap.example.com",
		"smtp_host":          "smtp.example.com",
		"username":           "history",
		"password_encrypted": "enc:dummy",
		"is_default":         false,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })
	folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"tenant_id": tenantID, "account_id": accountID, "name": "INBOX", "imap_name": "INBOX",
	})
	messageID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
		"tenant_id": tenantID, "account_id": accountID, "folder_id": folderID,
		"uid": 1, "message_id": "<" + uuid.NewString() + "@example.com>", "in_reply_to": "",
		"from_name": "Kontakt", "from_email": "kontakt@example.com", "subject": "Angebot",
		"preview": "", "body_text": "", "body_html": "", "raw_headers": "", "date": date,
	})
	testutil.SeedRow(t, pool, "email_contact_links", map[string]any{
		"tenant_id": tenantID, "message_id": messageID, "contact_id": contactID, "link_type": "auto",
	})
	return messageID
}

// GetByContactID with an owner lists, and counts, only the linked mail of
// that user's own mailboxes; without one it lists the linked mail of the
// whole tenant.
func TestGetByContactID_OwnerConfinesHistoryToOwnMailboxes_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "contactlink history")
	user := func() uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("history-%s@example.local", uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	alice, bob, carol := user(), user(), user()

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "History", "last_name": "Target", "created_by": alice,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	now := time.Now().UTC()
	aliceOld := seedLinkedMail(t, pool, tenantID, alice, contactID, now.Add(-time.Hour))
	aliceNew := seedLinkedMail(t, pool, tenantID, alice, contactID, now)
	bobMsg := seedLinkedMail(t, pool, tenantID, bob, contactID, now.Add(-30*time.Minute))

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	all, total, err := repo.GetByContactID(ctx, contactID, tenantID, nil, 1, 50)
	if err != nil {
		t.Fatalf("history without owner: %v", err)
	}
	if len(all) != 3 || total != 3 {
		t.Fatalf("history without owner = %d messages, total %d; want 3 and 3", len(all), total)
	}

	own, total, err := repo.GetByContactID(ctx, contactID, tenantID, &alice, 1, 1)
	if err != nil {
		t.Fatalf("alice history: %v", err)
	}
	if len(own) != 1 || own[0].ID != aliceNew || total != 2 {
		t.Fatalf("alice page 1 = %v, total %d; want newest own message %s and total 2", own, total, aliceNew)
	}
	own, _, err = repo.GetByContactID(ctx, contactID, tenantID, &alice, 2, 1)
	if err != nil {
		t.Fatalf("alice history page 2: %v", err)
	}
	if len(own) != 1 || own[0].ID != aliceOld {
		t.Fatalf("alice page 2 = %v, want %s (bob's %s must not take its place)", own, aliceOld, bobMsg)
	}

	none, total, err := repo.GetByContactID(ctx, contactID, tenantID, &carol, 1, 50)
	if err != nil {
		t.Fatalf("carol history: %v", err)
	}
	if len(none) != 0 || total != 0 {
		t.Fatalf("carol without a mailbox sees %d messages, total %d; want none", len(none), total)
	}

	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "contactlink history other")
	foreign, total, err := repo.GetByContactID(testutil.WithTenantCtx(context.Background(), other), contactID, other, nil, 1, 50)
	if err != nil {
		t.Fatalf("foreign tenant history: %v", err)
	}
	if len(foreign) != 0 || total != 0 {
		t.Fatalf("foreign tenant sees %d messages, total %d; want none", len(foreign), total)
	}
}
