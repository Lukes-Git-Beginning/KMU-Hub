package contact

// fix-contact-erasure-leaves-inbox-thread-messages: the hard-delete path runs
// the same consent.ScrubDependentPII as anonymization, so the inbound texts of
// the contact's inbox conversation must be gone after Delete too.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDelete_ScrubsInboundInboxThreadTexts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Contact Delete Inbox Thread Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("contact-delete-thread-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Delete", "last_name": "ThreadScrub", "created_by": userID,
	})

	msgID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id": tenantID, "user_id": userID, "channel": "email",
		"source_id": "src-" + uuid.NewString(), "crm_contact_id": contactID,
		"sender_name": "Delete ThreadScrub", "received_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", msgID)

	inboundID := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": tenantID, "message_id": msgID, "author_name": "Delete ThreadScrub",
		"direction": "inbound", "body": "Hallo, ich bin Delete ThreadScrub",
	})
	defer testutil.CleanupRow(t, pool, "inbox_thread_messages", inboundID)

	outboundID := testutil.SeedRow(t, pool, "inbox_thread_messages", map[string]any{
		"tenant_id": tenantID, "message_id": msgID, "author_id": userID,
		"direction": "outbound", "body": "Antwort des Mitarbeiters",
	})
	defer testutil.CleanupRow(t, pool, "inbox_thread_messages", outboundID)

	repo := NewPostgresRepository(pool)
	if err := repo.Delete(testutil.WithTenantCtx(context.Background(), tenantID), contactID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	var body string
	var author *string
	if err := pool.QueryRow(sysCtx, "SELECT body, author_name FROM inbox_thread_messages WHERE id = $1", inboundID).Scan(&body, &author); err != nil {
		t.Fatalf("read back inbound thread row: %v", err)
	}
	if body != "" || author != nil {
		t.Fatalf("expected inbound thread row scrubbed after Delete, got body=%q author_name=%v", body, author)
	}
	if err := pool.QueryRow(sysCtx, "SELECT body FROM inbox_thread_messages WHERE id = $1", outboundID).Scan(&body); err != nil || body != "Antwort des Mitarbeiters" {
		t.Fatalf("outbound reply must survive Delete, got body=%q err=%v", body, err)
	}
}
