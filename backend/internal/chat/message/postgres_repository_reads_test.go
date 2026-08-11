package message

// Covers the read and helper methods on PostgresRepository that
// tenant_write_test.go does not exercise: GetByID/List/ListReplies,
// the channel/membership checks, user/mention/file lookups, DM/channel-info
// helpers and the guest-session helpers. tenant_write_test.go already proves
// the write paths land in and stay scoped to the caller's tenant — this file
// proves the read paths return the right rows and the right errors.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// readFixture bundles the tenant/user/channel scaffolding shared by the
// tests below. Each test gets its own fixture (fresh tenant) so they stay
// safe to run under t.Parallel().
type readFixture struct {
	tenant, otherTenant uuid.UUID
	userA, userB        uuid.UUID
	channel             uuid.UUID
}

func newReadFixture(t *testing.T, pool *pgxpool.Pool) readFixture {
	t.Helper()
	tenant, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Read Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Message Read Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("message-read-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Anna",
		"last_name":     "Reader",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userA) })

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("message-read-b-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Ben",
		"last_name":     "Reader",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })

	channel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenant,
		"name":       "message-read-test",
		"created_by": userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channel) })

	return readFixture{tenant: tenant, otherTenant: otherTenant, userA: userA, userB: userB, channel: channel}
}

// seedMembership inserts a channel_memberships row directly — the table has
// a composite primary key (channel_id, user_id), so testutil.SeedRow (which
// always does RETURNING id) does not apply.
func seedMembership(t *testing.T, pool *pgxpool.Pool, channelID, userID, tenantID uuid.UUID, role models.ChannelRole) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id, role, joined_at) VALUES ($1, $2, $3, $4, $5)`,
		channelID, userID, tenantID, role, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}
}

func seedMessage(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, createdBy uuid.UUID, content string, at time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantID,
		"channel_id": channelID,
		"content":    content,
		"lang":       "german",
		"created_by": createdBy,
		"created_at": at,
	})
}

func TestPostgresRepository_GetByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	msgID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "hello read", time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgID) })

	got, err := repo.GetByID(ctxOwn, msgID, fx.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Content != "hello read" {
		t.Fatalf("GetByID: expected content %q, got %q", "hello read", got.Content)
	}
	if got.ChannelID != fx.channel {
		t.Fatalf("GetByID: expected channel %s, got %s", fx.channel, got.ChannelID)
	}

	if _, err := repo.GetByID(ctxOwn, uuid.New(), fx.tenant); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("GetByID nonexistent: expected ErrMessageNotFound, got %v", err)
	}
}

func TestPostgresRepository_List(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	base := time.Now().UTC()
	parentID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "parent", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", parentID) })

	reply := &models.Message{
		ID: uuid.New(), TenantID: fx.tenant, ChannelID: fx.channel, Content: "a reply",
		Lang: "german", CreatedBy: &fx.userA, ParentMessageID: &parentID, CreatedAt: base.Add(time.Second),
	}
	if err := repo.CreateWithReplyCount(ctxOwn, reply); err != nil {
		t.Fatalf("seed reply: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", reply.ID) })

	all, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, ChannelID: fx.channel, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List: expected 2 messages (parent + reply), got %d", len(all))
	}
	if all[0].ID != reply.ID {
		t.Fatalf("List: expected newest message (the reply) first, got %s", all[0].ID)
	}
	if all[0].SenderFirstName != "Anna" || all[0].SenderLastName != "Reader" {
		t.Fatalf("List: expected sender Anna Reader, got %s %s", all[0].SenderFirstName, all[0].SenderLastName)
	}

	excl, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, ChannelID: fx.channel, Limit: 10, ExcludeReplies: true})
	if err != nil {
		t.Fatalf("List ExcludeReplies: %v", err)
	}
	if len(excl) != 1 || excl[0].ID != parentID {
		t.Fatalf("List ExcludeReplies: expected only the parent message, got %d messages", len(excl))
	}
}

func TestPostgresRepository_ListReplies(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	base := time.Now().UTC()
	parentID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "parent", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", parentID) })

	reply1 := &models.Message{
		ID: uuid.New(), TenantID: fx.tenant, ChannelID: fx.channel, Content: "first reply",
		Lang: "german", CreatedBy: &fx.userA, ParentMessageID: &parentID, CreatedAt: base.Add(time.Second),
	}
	if err := repo.CreateWithReplyCount(ctxOwn, reply1); err != nil {
		t.Fatalf("seed reply1: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", reply1.ID) })

	reply2 := &models.Message{
		ID: uuid.New(), TenantID: fx.tenant, ChannelID: fx.channel, Content: "second reply",
		Lang: "german", CreatedBy: &fx.userB, ParentMessageID: &parentID, CreatedAt: base.Add(2 * time.Second),
	}
	if err := repo.CreateWithReplyCount(ctxOwn, reply2); err != nil {
		t.Fatalf("seed reply2: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", reply2.ID) })

	replies, err := repo.ListReplies(ctxOwn, ThreadListFilter{TenantID: fx.tenant, ParentMessageID: parentID, Limit: 10})
	if err != nil {
		t.Fatalf("ListReplies: %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("ListReplies: expected 2 replies, got %d", len(replies))
	}
	if replies[0].ID != reply1.ID || replies[1].ID != reply2.ID {
		t.Fatalf("ListReplies: expected oldest-first order [%s, %s], got [%s, %s]",
			reply1.ID, reply2.ID, replies[0].ID, replies[1].ID)
	}

	limited, err := repo.ListReplies(ctxOwn, ThreadListFilter{TenantID: fx.tenant, ParentMessageID: parentID, Limit: 1})
	if err != nil {
		t.Fatalf("ListReplies limited: %v", err)
	}
	if len(limited) != 1 || limited[0].ID != reply1.ID {
		t.Fatalf("ListReplies limited: expected only the oldest reply, got %d rows", len(limited))
	}
}

func TestPostgresRepository_ChannelAndMembershipChecks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	seedMembership(t, pool, fx.channel, fx.userA, fx.tenant, models.ChannelRoleOwner)

	if exists, err := repo.ChannelExists(ctxOwn, fx.channel, fx.tenant); err != nil || !exists {
		t.Fatalf("ChannelExists: expected true, got %v (err %v)", exists, err)
	}
	if exists, err := repo.ChannelExists(ctxOwn, uuid.New(), fx.tenant); err != nil || exists {
		t.Fatalf("ChannelExists: expected false for unknown channel, got %v (err %v)", exists, err)
	}

	if archived, err := repo.IsChannelArchived(ctxOwn, fx.channel, fx.tenant); err != nil || archived {
		t.Fatalf("IsChannelArchived: expected false, got %v (err %v)", archived, err)
	}
	if _, err := repo.IsChannelArchived(ctxOwn, uuid.New(), fx.tenant); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("IsChannelArchived unknown channel: expected ErrChannelNotFound, got %v", err)
	}

	if member, err := repo.IsMember(ctxOwn, fx.channel, fx.tenant, fx.userA); err != nil || !member {
		t.Fatalf("IsMember: expected true for userA, got %v (err %v)", member, err)
	}
	if member, err := repo.IsMember(ctxOwn, fx.channel, fx.tenant, fx.userB); err != nil || member {
		t.Fatalf("IsMember: expected false for userB (not added), got %v (err %v)", member, err)
	}

	if role, err := repo.GetMemberRole(ctxOwn, fx.channel, fx.userA); err != nil || role != models.ChannelRoleOwner {
		t.Fatalf("GetMemberRole: expected owner, got %q (err %v)", role, err)
	}
	if _, err := repo.GetMemberRole(ctxOwn, fx.channel, fx.userB); !errors.Is(err, ErrNotChannelMember) {
		t.Fatalf("GetMemberRole userB: expected ErrNotChannelMember, got %v", err)
	}
}

func TestPostgresRepository_GetUserInfo(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	firstName, lastName, err := repo.GetUserInfo(ctxOwn, fx.userA)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if firstName != "Anna" || lastName != "Reader" {
		t.Fatalf("GetUserInfo: expected Anna Reader, got %s %s", firstName, lastName)
	}
}

func TestPostgresRepository_Mentions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	msgID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "hey @Ben", time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgID) })

	mention := models.Mention{
		MessageID: msgID, TenantID: fx.tenant, UserID: fx.userB,
		MentionType: models.MentionTypeUser, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateMentions(ctxOwn, msgID, []models.Mention{mention}); err != nil {
		t.Fatalf("seed mention: %v", err)
	}

	byMsg, err := repo.GetMentionsByMessages(ctxOwn, []uuid.UUID{msgID})
	if err != nil {
		t.Fatalf("GetMentionsByMessages: %v", err)
	}
	got := byMsg[msgID]
	if len(got) != 1 || got[0].UserID != fx.userB || got[0].FirstName != "Ben" {
		t.Fatalf("GetMentionsByMessages: expected one mention of Ben, got %+v", got)
	}

	rows, total, err := repo.GetMentionsForUser(ctxOwn, fx.userB, 10, 0)
	if err != nil {
		t.Fatalf("GetMentionsForUser: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("GetMentionsForUser: expected 1 total/1 row, got total=%d rows=%d", total, len(rows))
	}
	if rows[0].MessageID != msgID || rows[0].SenderFirstName != "Anna" {
		t.Fatalf("GetMentionsForUser: expected message %s from Anna, got %+v", msgID, rows[0])
	}
}

func TestPostgresRepository_GetFilesByMessageIDs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	msgID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "see attachment", time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgID) })

	fileID := testutil.SeedRow(t, pool, "chat_files", map[string]any{
		"tenant_id":   fx.tenant,
		"message_id":  msgID,
		"channel_id":  fx.channel,
		"filename":    "report.pdf",
		"mime_type":   "application/pdf",
		"file_size":   1024,
		"storage_key": "chat/report.pdf",
		"uploaded_by": fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", fileID) })

	files, err := repo.GetFilesByMessageIDs(ctxOwn, []uuid.UUID{msgID})
	if err != nil {
		t.Fatalf("GetFilesByMessageIDs: %v", err)
	}
	got := files[msgID]
	if len(got) != 1 || got[0].Filename != "report.pdf" || got[0].UploaderFirstName != "Anna" {
		t.Fatalf("GetFilesByMessageIDs: expected report.pdf uploaded by Anna, got %+v", got)
	}
}

func TestPostgresRepository_GetChannelMemberIDs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	seedMembership(t, pool, fx.channel, fx.userA, fx.tenant, models.ChannelRoleOwner)
	seedMembership(t, pool, fx.channel, fx.userB, fx.tenant, models.ChannelRoleMember)

	ids, err := repo.GetChannelMemberIDs(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("GetChannelMemberIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("GetChannelMemberIDs: expected 2 members, got %d", len(ids))
	}
	seen := map[uuid.UUID]bool{ids[0]: true, ids[1]: true}
	if !seen[fx.userA] || !seen[fx.userB] {
		t.Fatalf("GetChannelMemberIDs: expected userA and userB, got %v", ids)
	}
}

func TestPostgresRepository_DMAndChannelInfo(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	name, err := repo.GetChannelName(ctxOwn, fx.channel)
	if err != nil || name != "message-read-test" {
		t.Fatalf("GetChannelName: expected message-read-test, got %q (err %v)", name, err)
	}
	if _, err := repo.GetChannelName(ctxOwn, uuid.New()); err == nil {
		t.Fatalf("GetChannelName unknown channel: expected an error, got nil")
	}

	// fx.channel is a regular (non-DM) channel — no recipient to resolve.
	recipient, err := repo.GetDMRecipient(ctxOwn, fx.channel, fx.userA)
	if err != nil || recipient != nil {
		t.Fatalf("GetDMRecipient non-DM channel: expected nil recipient, got %v (err %v)", recipient, err)
	}

	dmUser1, dmUser2 := fx.userA, fx.userB
	if dmUser1.String() > dmUser2.String() {
		dmUser1, dmUser2 = dmUser2, dmUser1
	}
	dmChannel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  fx.tenant,
		"name":       "dm",
		"is_dm":      true,
		"dm_user1":   dmUser1,
		"dm_user2":   dmUser2,
		"created_by": fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", dmChannel) })

	other, err := repo.GetDMRecipient(ctxOwn, dmChannel, dmUser1)
	if err != nil || other == nil || *other != dmUser2 {
		t.Fatalf("GetDMRecipient from dmUser1: expected dmUser2, got %v (err %v)", other, err)
	}
	other2, err := repo.GetDMRecipient(ctxOwn, dmChannel, dmUser2)
	if err != nil || other2 == nil || *other2 != dmUser1 {
		t.Fatalf("GetDMRecipient from dmUser2: expected dmUser1, got %v (err %v)", other2, err)
	}
}

func TestPostgresRepository_GuestInfo(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	if enabled, err := repo.IsChannelGuestEnabled(ctxOwn, fx.channel); err != nil || enabled {
		t.Fatalf("IsChannelGuestEnabled: expected false by default, got %v (err %v)", enabled, err)
	}

	guestChannel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":        fx.tenant,
		"name":             "guest-enabled",
		"is_guest_enabled": true,
		"created_by":       fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", guestChannel) })

	if enabled, err := repo.IsChannelGuestEnabled(ctxOwn, guestChannel); err != nil || !enabled {
		t.Fatalf("IsChannelGuestEnabled: expected true, got %v (err %v)", enabled, err)
	}

	guestSession := testutil.SeedRow(t, pool, "guest_sessions", map[string]any{
		"tenant_id":    fx.tenant,
		"token_hash":   fmt.Sprintf("hash-%s", uuid.New().String()),
		"channel_id":   guestChannel,
		"display_name": "Guest Gustav",
		"expires_at":   time.Now().UTC().Add(24 * time.Hour),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", guestSession) })

	if name, err := repo.GetGuestDisplayName(ctxOwn, guestSession); err != nil || name != "Guest Gustav" {
		t.Fatalf("GetGuestDisplayName: expected Guest Gustav, got %q (err %v)", name, err)
	}
	if _, err := repo.GetGuestDisplayName(ctxOwn, uuid.New()); err == nil {
		t.Fatalf("GetGuestDisplayName unknown session: expected an error, got nil")
	}
}

func TestPostgresRepository_DecrementReplyCount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo := NewPostgresRepository(pool)

	base := time.Now().UTC()
	parentID := seedMessage(t, pool, fx.tenant, fx.channel, fx.userA, "parent", base)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", parentID) })

	reply := &models.Message{
		ID: uuid.New(), TenantID: fx.tenant, ChannelID: fx.channel, Content: "a reply",
		Lang: "german", CreatedBy: &fx.userA, ParentMessageID: &parentID, CreatedAt: base.Add(time.Second),
	}
	if err := repo.CreateWithReplyCount(ctxOwn, reply); err != nil {
		t.Fatalf("seed reply: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", reply.ID) })

	readReplyCount := func() int {
		t.Helper()
		var count int
		if err := pool.QueryRow(sysCtx, "SELECT reply_count FROM messages WHERE id = $1", parentID).Scan(&count); err != nil {
			t.Fatalf("read reply_count: %v", err)
		}
		return count
	}

	if got := readReplyCount(); got != 1 {
		t.Fatalf("reply_count after CreateWithReplyCount: expected 1, got %d", got)
	}

	if err := repo.DecrementReplyCount(ctxOwn, parentID); err != nil {
		t.Fatalf("DecrementReplyCount: %v", err)
	}
	if got := readReplyCount(); got != 0 {
		t.Fatalf("reply_count after DecrementReplyCount: expected 0, got %d", got)
	}

	// GREATEST(reply_count - 1, 0) floors at zero instead of going negative.
	if err := repo.DecrementReplyCount(ctxOwn, parentID); err != nil {
		t.Fatalf("DecrementReplyCount below zero: %v", err)
	}
	if got := readReplyCount(); got != 0 {
		t.Fatalf("reply_count floored at zero: expected 0, got %d", got)
	}
}
