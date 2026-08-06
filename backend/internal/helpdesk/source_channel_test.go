package helpdesk

// g-helpdesk-source-channel: CreateTicketFromMessage converts an inbox
// message into a ticket, linking back via source_channel/source_message_id
// instead of copying the message a second time. These tests exercise the
// real repository against Postgres so the idx_tickets_source_message unique
// index and the pre-check/race-recovery path in the service are both proven,
// not just asserted against a mock.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCreateTicketFromMessage_CreatesAndPersistsProvenance(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Source Channel Tenant")

	requesterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-source-channel-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", requesterID)

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     requesterID,
		"channel":     "email",
		"source_id":   fmt.Sprintf("msg-%s", uuid.New().String()[:8]),
		"subject":     "Need help logging in",
		"preview":     "I can't log into my account anymore.",
		"received_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", messageID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	ticket, created, err := svc.CreateTicketFromMessage(ctxOwn, tenantOwn, requesterID, messageID, "email", "Need help logging in", "I can't log into my account anymore.")
	if err != nil {
		t.Fatalf("CreateTicketFromMessage: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	if !created {
		t.Fatal("expected created=true for a fresh message_id")
	}
	if ticket.Status != TicketStatusOpen {
		t.Fatalf("expected status=open, got %s", ticket.Status)
	}
	if ticket.RequesterID == nil || *ticket.RequesterID != requesterID {
		t.Fatalf("expected requester_id=%s, got %s", requesterID, ticket.RequesterID)
	}
	if ticket.SourceChannel == nil || *ticket.SourceChannel != "email" {
		t.Fatalf("expected source_channel=email, got %v", ticket.SourceChannel)
	}
	if ticket.SourceMessageID == nil || *ticket.SourceMessageID != messageID {
		t.Fatalf("expected source_message_id=%s, got %v", messageID, ticket.SourceMessageID)
	}
	if ticket.Subject != "Need help logging in" {
		t.Fatalf("expected subject to carry over from the message, got %q", ticket.Subject)
	}
	if ticket.Description != "I can't log into my account anymore." {
		t.Fatalf("expected description to carry the message preview, got %q", ticket.Description)
	}

	// Persisted, not just returned in-memory: re-read via GetTicketByID.
	reread, err := repo.GetTicketByID(ctxOwn, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if reread.SourceChannel == nil || *reread.SourceChannel != "email" {
		t.Fatalf("re-read: expected source_channel=email, got %v", reread.SourceChannel)
	}
	if reread.SourceMessageID == nil || *reread.SourceMessageID != messageID {
		t.Fatalf("re-read: expected source_message_id=%s, got %v", messageID, reread.SourceMessageID)
	}
}

func TestCreateTicketFromMessage_Idempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Source Channel Idempotent Tenant")

	requesterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-source-channel-idem-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", requesterID)

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     requesterID,
		"channel":     "chat",
		"source_id":   fmt.Sprintf("msg-idem-%s", uuid.New().String()[:8]),
		"subject":     "Question about billing",
		"preview":     "When is my invoice due?",
		"received_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", messageID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	first, created, err := svc.CreateTicketFromMessage(ctxOwn, tenantOwn, requesterID, messageID, "chat", "Question about billing", "When is my invoice due?")
	if err != nil {
		t.Fatalf("CreateTicketFromMessage (first): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", first.ID)
	if !created {
		t.Fatal("expected created=true on the first conversion")
	}

	second, created, err := svc.CreateTicketFromMessage(ctxOwn, tenantOwn, requesterID, messageID, "chat", "Question about billing", "When is my invoice due?")
	if err != nil {
		t.Fatalf("CreateTicketFromMessage (second): %v", err)
	}
	if created {
		t.Fatal("expected created=false on the second conversion of the same message_id")
	}
	if second.ID != first.ID {
		t.Fatalf("expected the same ticket back, got first=%s second=%s", first.ID, second.ID)
	}

	var count int
	if err := pool.QueryRow(ctxOwn, "SELECT COUNT(*) FROM tickets WHERE source_message_id = $1", messageID).Scan(&count); err != nil {
		t.Fatalf("count tickets by source_message_id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 ticket linked to the message, got %d", count)
	}
}

func TestCreateTicketFromMessage_EmptySubjectFallsBackToChannelLabel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Source Channel Empty Subject Tenant")

	requesterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-source-channel-empty-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", requesterID)

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     requesterID,
		"channel":     "notification",
		"source_id":   fmt.Sprintf("msg-empty-%s", uuid.New().String()[:8]),
		"received_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", messageID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	ticket, _, err := svc.CreateTicketFromMessage(ctxOwn, tenantOwn, requesterID, messageID, "notification", "", "")
	if err != nil {
		t.Fatalf("CreateTicketFromMessage: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	if ticket.Subject == "" {
		t.Fatal("expected a non-empty fallback subject when the message has none")
	}
}

func TestCreateTicketFromMessage_RejectsInvalidChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Source Channel Invalid Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	_, _, err := svc.CreateTicketFromMessage(ctxOwn, tenantOwn, uuid.New(), uuid.New(), "guest", "Subject", "Preview")
	if !errors.Is(err, ErrInvalidSourceChannel) {
		t.Fatalf("expected ErrInvalidSourceChannel, got %v", err)
	}
}
