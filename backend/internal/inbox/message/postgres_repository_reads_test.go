package message

// Covers the PostgresRepository read paths that tenant_write_test.go doesn't
// touch: List's filter/pagination logic, GetUnreadCounts' per-channel
// grouping, GetBySourceID's dedup lookup, and UnsnoozeExpired's global sweep.
// All four run against a real Postgres, mirroring the DB-integration pattern
// from produktion/postgres_repository_core_test.go.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedReadMessage(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID, mutate func(*models.InboxMessage)) *models.InboxMessage {
	t.Helper()
	msg := &models.InboxMessage{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Channel:    "email",
		SourceID:   "src-" + uuid.New().String()[:8],
		SenderName: "Read Test Sender",
		Subject:    "Subject",
		Preview:    "Preview",
		Status:     "open",
		Tags:       []string{},
		ReceivedAt: time.Now().UTC(),
	}
	if mutate != nil {
		mutate(msg)
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", msg.ID) })
	return msg
}

func TestList_FiltersByChannelAndReadStatus_WithCursorPagination(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Message List Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-msg-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	base := time.Now().UTC().Add(-1 * time.Hour)

	unreadEmail := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.Channel = "email"
		m.IsRead = false
		m.ReceivedAt = base.Add(3 * time.Minute)
	})
	readEmail := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.Channel = "email"
		m.IsRead = true
		m.ReceivedAt = base.Add(2 * time.Minute)
	})
	unreadChat := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.Channel = "chat"
		m.IsRead = false
		m.ReceivedAt = base.Add(1 * time.Minute)
	})

	t.Run("filters by channel and is_read together", func(t *testing.T) {
		channel := "email"
		isRead := false
		msgs, total, err := repo.List(ctx, ListFilter{
			TenantID: tenantID, UserID: userID, Channel: &channel, IsRead: &isRead, PageSize: 50,
		})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 || len(msgs) != 1 || msgs[0].ID != unreadEmail.ID {
			t.Fatalf("expected only %s, got total=%d msgs=%v", unreadEmail.ID, total, msgIDs(msgs))
		}
		_ = readEmail
		_ = unreadChat
	})

	t.Run("a filter matching nothing returns an empty page, not an error", func(t *testing.T) {
		channel := "sms"
		msgs, total, err := repo.List(ctx, ListFilter{
			TenantID: tenantID, UserID: userID, Channel: &channel, PageSize: 50,
		})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 || len(msgs) != 0 {
			t.Fatalf("expected no matches, got total=%d msgs=%v", total, msgIDs(msgs))
		}
	})

	t.Run("cursor pagination walks the full set in received_at DESC order without gaps or repeats", func(t *testing.T) {
		firstPage, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, UserID: userID, PageSize: 2})
		if err != nil {
			t.Fatalf("List (page 1): %v", err)
		}
		if total != 3 || len(firstPage) != 2 {
			t.Fatalf("expected total=3 page=2, got total=%d page=%d", total, len(firstPage))
		}
		if firstPage[0].ID != unreadEmail.ID || firstPage[1].ID != readEmail.ID {
			t.Fatalf("expected DESC order [%s,%s], got %v", unreadEmail.ID, readEmail.ID, msgIDs(firstPage))
		}

		last := firstPage[len(firstPage)-1]
		secondPage, total2, err := repo.List(ctx, ListFilter{
			TenantID: tenantID, UserID: userID, PageSize: 2,
			CursorReceivedAt: &last.ReceivedAt, CursorID: &last.ID,
		})
		if err != nil {
			t.Fatalf("List (page 2): %v", err)
		}
		if total2 != 3 || len(secondPage) != 1 || secondPage[0].ID != unreadChat.ID {
			t.Fatalf("expected remaining [%s], got total=%d msgs=%v", unreadChat.ID, total2, msgIDs(secondPage))
		}
	})
}

func msgIDs(msgs []*models.InboxMessage) []uuid.UUID {
	ids := make([]uuid.UUID, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	return ids
}

func TestList_ExcludesArchivedAndSnoozedByDefault(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Message List Default Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-msg-listdef-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	visible := seedReadMessage(t, repo, ctx, pool, tenantID, userID, nil)
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.IsArchived = true
	})
	futureSnooze := time.Now().UTC().Add(2 * time.Hour)
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.SnoozedUntil = &futureSnooze
	})

	msgs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, UserID: userID, PageSize: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(msgs) != 1 || msgs[0].ID != visible.ID {
		t.Fatalf("expected only the plain message %s, got total=%d msgs=%v", visible.ID, total, msgIDs(msgs))
	}

	t.Run("IsArchived=true flips the default and returns only archived items", func(t *testing.T) {
		archived := true
		msgs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, UserID: userID, IsArchived: &archived, PageSize: 50})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 || len(msgs) != 1 || msgs[0].ID == visible.ID {
			t.Fatalf("expected exactly one archived message excluding %s, got total=%d msgs=%v", visible.ID, total, msgIDs(msgs))
		}
	})
}

func TestGetUnreadCounts_GroupsByChannelAndExcludesArchivedRead(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Message Unread Counts Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-msg-unread-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false })
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false })
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "chat"; m.IsRead = false })
	// Must not count: already read.
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = true })
	// Must not count: archived.
	seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false; m.IsArchived = true })

	counts, err := repo.GetUnreadCounts(ctx, userID)
	if err != nil {
		t.Fatalf("GetUnreadCounts: %v", err)
	}

	byChannel := map[string]int{}
	for _, c := range counts {
		byChannel[c.Channel] = c.Count
	}
	if byChannel["email"] != 2 {
		t.Fatalf("expected 2 unread email, got %d (all=%v)", byChannel["email"], counts)
	}
	if byChannel["chat"] != 1 {
		t.Fatalf("expected 1 unread chat, got %d (all=%v)", byChannel["chat"], counts)
	}

	t.Run("a user with no messages gets an empty result, not an error", func(t *testing.T) {
		otherUser := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("inbox-msg-unread-empty-%s@tenantown.local", uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUser) })

		counts, err := repo.GetUnreadCounts(ctx, otherUser)
		if err != nil {
			t.Fatalf("GetUnreadCounts: %v", err)
		}
		if len(counts) != 0 {
			t.Fatalf("expected no rows for a user with no messages, got %v", counts)
		}
	})

	t.Run("a foreign-tenant ctx sees nothing -- GetUnreadCounts has no tenant_id predicate, RLS is the only backstop", func(t *testing.T) {
		tenantOther := uuid.New()
		testutil.EnsureTenant(t, pool, tenantOther, "Inbox Message Unread Counts Other Tenant")
		ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

		counts, err := repo.GetUnreadCounts(ctxOther, userID)
		if err != nil {
			t.Fatalf("GetUnreadCounts (foreign ctx): %v", err)
		}
		if len(counts) != 0 {
			t.Fatalf("a foreign-tenant ctx must not see another tenant's unread counts, got %v", counts)
		}
	})
}

func TestGetBySourceID_FindsTripleAndReturnsNilOnMiss(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Message GetBySourceID Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-msg-bysource-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	sourceID := "dedup-src-" + uuid.New().String()[:8]
	seeded := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.Channel = "chat"
		m.SourceID = sourceID
	})

	got, err := repo.GetBySourceID(ctx, userID, "chat", sourceID)
	if err != nil {
		t.Fatalf("GetBySourceID: %v", err)
	}
	if got == nil || got.ID != seeded.ID {
		t.Fatalf("expected to find %s, got %v", seeded.ID, got)
	}

	t.Run("an unknown triple returns (nil, nil), the documented dedup-check contract", func(t *testing.T) {
		// GetBySourceID intentionally returns no error on a miss (see the
		// "Not found is not an error for dedup checks" comment in
		// postgres_repository.go) -- Service.Create treats a nil message as
		// "no duplicate" and proceeds to insert.
		got, err := repo.GetBySourceID(ctx, userID, "chat", "does-not-exist-"+uuid.New().String())
		if err != nil {
			t.Fatalf("GetBySourceID (miss): expected nil error, got %v", err)
		}
		if got != nil {
			t.Fatalf("GetBySourceID (miss): expected nil message, got %v", got)
		}
	})

	t.Run("channel is part of the lookup key", func(t *testing.T) {
		got, err := repo.GetBySourceID(ctx, userID, "email", sourceID)
		if err != nil {
			t.Fatalf("GetBySourceID (wrong channel): %v", err)
		}
		if got != nil {
			t.Fatalf("GetBySourceID (wrong channel): expected nil, got %v", got)
		}
	})

	t.Run("a foreign-tenant ctx sees nothing -- GetBySourceID has no tenant_id predicate, RLS is the only backstop", func(t *testing.T) {
		tenantOther := uuid.New()
		testutil.EnsureTenant(t, pool, tenantOther, "Inbox Message GetBySourceID Other Tenant")
		ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

		got, err := repo.GetBySourceID(ctxOther, userID, "chat", sourceID)
		if err != nil {
			t.Fatalf("GetBySourceID (foreign ctx): expected nil error, got %v", err)
		}
		if got != nil {
			t.Fatalf("a foreign-tenant ctx must not find another tenant's message, got %v", got)
		}
	})
}

func TestUnsnoozeExpired_ResetsOnlyPastEntries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Message Unsnooze Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-msg-unsnooze-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())

	t.Run("no expired entries yet: reports zero, no error", func(t *testing.T) {
		count, err := repo.UnsnoozeExpired(sysCtx)
		if err != nil {
			t.Fatalf("UnsnoozeExpired: %v", err)
		}
		if count != 0 {
			// Another parallel test could theoretically have an expired row,
			// but no test in this package snoozes into the past except the
			// ones created below, after this subtest.
			t.Fatalf("expected 0 expired entries before any are seeded, got %d", count)
		}
	})

	past := time.Now().UTC().Add(-1 * time.Hour)
	expired := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.IsRead = true
		m.SnoozedUntil = &past
	})
	future := time.Now().UTC().Add(2 * time.Hour)
	stillSnoozed := seedReadMessage(t, repo, ctx, pool, tenantID, userID, func(m *models.InboxMessage) {
		m.IsRead = true
		m.SnoozedUntil = &future
	})

	count, err := repo.UnsnoozeExpired(sysCtx)
	if err != nil {
		t.Fatalf("UnsnoozeExpired: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least the seeded expired entry to be reset, got count=%d", count)
	}

	gotExpired, err := repo.GetByID(ctx, expired.ID, tenantID)
	if err != nil {
		t.Fatalf("GetByID (expired): %v", err)
	}
	if gotExpired.SnoozedUntil != nil {
		t.Fatalf("expected expired entry's SnoozedUntil to be cleared, got %v", gotExpired.SnoozedUntil)
	}
	if gotExpired.IsRead {
		t.Fatalf("expected expired entry to be marked unread again")
	}

	gotFuture, err := repo.GetByID(ctx, stillSnoozed.ID, tenantID)
	if err != nil {
		t.Fatalf("GetByID (future): %v", err)
	}
	if gotFuture.SnoozedUntil == nil {
		t.Fatalf("expected future-snoozed entry to remain snoozed")
	}
	if !gotFuture.IsRead {
		t.Fatalf("expected future-snoozed entry's read flag to remain untouched")
	}
}
