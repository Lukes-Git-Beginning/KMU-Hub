package consent

// fix-contact-erasure-leaves-inbox-thread-messages: ScrubDependentPII cleared
// inbox_messages.sender_*/preview but not the conversation thread behind it --
// inbox_thread_messages.body/.author_name still carried the contact's name and
// words after anonymization.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAnonymizeContact_ScrubsInboundInboxThreadTexts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Consent Inbox Thread Scrub Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Consent Inbox Thread Scrub Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-thread-scrub-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     otherTenantID,
		"email":         fmt.Sprintf("inbox-thread-scrub-other-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	newContact := func(tenant, user uuid.UUID, first string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "contacts", map[string]any{
			"tenant_id": tenant, "first_name": first, "last_name": "ThreadScrub", "created_by": user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
		return id
	}
	newMessage := func(tenant, user, contact uuid.UUID, sender string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
			"tenant_id": tenant, "user_id": user, "channel": "email",
			"source_id": "src-" + uuid.NewString(), "crm_contact_id": contact,
			"sender_name": sender, "received_at": time.Now().UTC(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", id) })
		return id
	}
	newThreadRow := func(tenant, message uuid.UUID, author *uuid.UUID, name *string, direction, body string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
			"tenant_id": tenant, "message_id": message, "author_id": author,
			"author_name": name, "direction": direction, "body": body,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_thread_messages", id) })
		return id
	}

	contactID := newContact(tenantID, userID, "Thread")
	siblingContactID := newContact(tenantID, userID, "Sibling")
	otherContactID := newContact(otherTenantID, otherUserID, "Foreign")

	msgID := newMessage(tenantID, userID, contactID, "Thread ThreadScrub")
	siblingMsgID := newMessage(tenantID, userID, siblingContactID, "Sibling ThreadScrub")
	otherMsgID := newMessage(otherTenantID, otherUserID, otherContactID, "Foreign ThreadScrub")

	name := "Thread ThreadScrub"
	sibName := "Sibling ThreadScrub"
	otherName := "Foreign ThreadScrub"
	inboundID := newThreadRow(tenantID, msgID, nil, &name, "inbound", "Hallo, ich bin Thread ThreadScrub")
	outboundID := newThreadRow(tenantID, msgID, &userID, nil, "outbound", "Guten Tag Herr ThreadScrub")
	internalID := newThreadRow(tenantID, msgID, &userID, nil, "internal", "Notiz: Kunde ist schwierig")
	siblingID := newThreadRow(tenantID, siblingMsgID, nil, &sibName, "inbound", "Text eines anderen Kontakts")
	foreignID := newThreadRow(otherTenantID, otherMsgID, nil, &otherName, "inbound", "Text eines fremden Tenants")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)
	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantID); err != nil {
		t.Fatalf("AnonymizeContact: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	read := func(id uuid.UUID) (body string, author *string) {
		t.Helper()
		if err := pool.QueryRow(sysCtx, "SELECT body, author_name FROM inbox_thread_messages WHERE id = $1", id).Scan(&body, &author); err != nil {
			t.Fatalf("read back thread row %s: %v", id, err)
		}
		return body, author
	}

	if body, author := read(inboundID); body != "" || author != nil {
		t.Fatalf("expected inbound thread row scrubbed, got body=%q author_name=%v", body, author)
	}
	if body, _ := read(outboundID); body != "Guten Tag Herr ThreadScrub" {
		t.Fatalf("outbound staff reply must keep its row and text, got %q", body)
	}
	if body, _ := read(internalID); body != "Notiz: Kunde ist schwierig" {
		t.Fatalf("internal note must keep its row and text, got %q", body)
	}
	if body, author := read(siblingID); body != "Text eines anderen Kontakts" || author == nil || *author != sibName {
		t.Fatalf("another contact's thread row must survive, got body=%q author_name=%v", body, author)
	}
	if body, author := read(foreignID); body != "Text eines fremden Tenants" || author == nil || *author != otherName {
		t.Fatalf("foreign tenant's thread row must survive, got body=%q author_name=%v", body, author)
	}
}
