package channel

// Covers the read methods on PostgresRepository that tenant_write_test.go
// does not exercise: GetByID/GetByIDForTenant/List, membership reads, user-
// info helpers, GetLastMessage, FindDMChannel and the unread-count reads.
// tenant_write_test.go already proves the write paths land in and stay
// scoped to the caller's tenant — this file proves the read paths return
// the right rows, the right zero-values and the right errors.

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

type readFixture struct {
	tenant, otherTenant uuid.UUID
	userA, userB        uuid.UUID
	channel             uuid.UUID
}

func newReadFixture(t *testing.T, pool *pgxpool.Pool) readFixture {
	t.Helper()
	tenant, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Channel Read Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Channel Read Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("channel-read-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Chana",
		"last_name":     "Reader",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userA) })

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("channel-read-b-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Beno",
		"last_name":     "Reader",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userB) })

	channel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenant,
		"name":       "channel-read-test",
		"created_by": userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channel) })

	return readFixture{tenant: tenant, otherTenant: otherTenant, userA: userA, userB: userB, channel: channel}
}

// seedMembership inserts a channel_memberships row directly — the table has
// a composite primary key (channel_id, user_id), so testutil.SeedRow (which
// always does RETURNING id) does not apply. Mirrors the helper of the same
// name in internal/chat/message/postgres_repository_reads_test.go.
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

func TestPostgresRepository_GetByID_GetByIDForTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	got, err := repo.GetByID(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "channel-read-test" {
		t.Fatalf("GetByID: expected name channel-read-test, got %q", got.Name)
	}

	if _, err := repo.GetByID(ctxOwn, uuid.New()); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("GetByID nonexistent: expected ErrChannelNotFound, got %v", err)
	}

	if _, err := repo.GetByIDForTenant(ctxOwn, fx.channel, fx.tenant); err != nil {
		t.Fatalf("GetByIDForTenant own tenant: %v", err)
	}
	if _, err := repo.GetByIDForTenant(ctxOwn, fx.channel, fx.otherTenant); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("GetByIDForTenant wrong tenant: expected ErrChannelNotFound, got %v", err)
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

	seedMembership(t, pool, fx.channel, fx.userA, fx.tenant, models.ChannelRoleOwner)

	archived := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": fx.tenant, "name": "channel-read-archived", "created_by": fx.userA, "is_archived": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", archived) })
	seedMembership(t, pool, archived, fx.userA, fx.tenant, models.ChannelRoleOwner)

	dmUser1, dmUser2 := fx.userA, fx.userB
	if dmUser1.String() > dmUser2.String() {
		dmUser1, dmUser2 = dmUser2, dmUser1
	}
	dmChannel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": fx.tenant, "name": "channel-read-dm", "created_by": fx.userA, "is_dm": true,
		"dm_user1": dmUser1, "dm_user2": dmUser2,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", dmChannel) })
	seedMembership(t, pool, dmChannel, fx.userA, fx.tenant, models.ChannelRoleMember)

	// Default: archived channel excluded, regular channel and DM included.
	list, total, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, UserID: fx.userA}, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("List: expected 2 non-archived channels, got total=%d len=%d", total, len(list))
	}

	// IncludeArchived surfaces all three.
	withArchived, totalWithArchived, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, UserID: fx.userA, IncludeArchived: true}, 0, 10)
	if err != nil {
		t.Fatalf("List IncludeArchived: %v", err)
	}
	if totalWithArchived != 3 || len(withArchived) != 3 {
		t.Fatalf("List IncludeArchived: expected 3 channels, got total=%d len=%d", totalWithArchived, len(withArchived))
	}

	// IsDM narrows to just the DM channel.
	dmOnly, totalDM, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, UserID: fx.userA, IsDM: ptr(true)}, 0, 10)
	if err != nil {
		t.Fatalf("List IsDM: %v", err)
	}
	if totalDM != 1 || len(dmOnly) != 1 || dmOnly[0].ID != dmChannel {
		t.Fatalf("List IsDM: expected only the DM channel, got total=%d len=%d", totalDM, len(dmOnly))
	}

	// Search filters by name.
	searched, totalSearch, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, UserID: fx.userA, Search: "read-test"}, 0, 10)
	if err != nil {
		t.Fatalf("List Search: %v", err)
	}
	if totalSearch != 1 || len(searched) != 1 || searched[0].ID != fx.channel {
		t.Fatalf("List Search: expected only fx.channel, got total=%d len=%d", totalSearch, len(searched))
	}

	// Error/edge path: a user with no memberships sees nothing.
	empty, totalEmpty, err := repo.List(ctxOwn, ListFilter{TenantID: fx.tenant, UserID: fx.userB}, 0, 10)
	if err != nil {
		t.Fatalf("List no membership: %v", err)
	}
	if totalEmpty != 0 || len(empty) != 0 {
		t.Fatalf("List no membership: expected 0 channels, got total=%d len=%d", totalEmpty, len(empty))
	}
}

func TestPostgresRepository_MembershipReads(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	seedMembership(t, pool, fx.channel, fx.userA, fx.tenant, models.ChannelRoleOwner)
	seedMembership(t, pool, fx.channel, fx.userB, fx.tenant, models.ChannelRoleMember)

	mem, err := repo.GetMembership(ctxOwn, fx.channel, fx.userA)
	if err != nil {
		t.Fatalf("GetMembership: %v", err)
	}
	if mem.Role != models.ChannelRoleOwner {
		t.Fatalf("GetMembership: expected owner, got %q", mem.Role)
	}

	if _, err := repo.GetMembership(ctxOwn, fx.channel, uuid.New()); !errors.Is(err, ErrNotChannelMember) {
		t.Fatalf("GetMembership unknown user: expected ErrNotChannelMember, got %v", err)
	}

	members, total, err := repo.ListMembers(ctxOwn, fx.channel, 0, 10)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if total != 2 || len(members) != 2 {
		t.Fatalf("ListMembers: expected 2 members, got total=%d len=%d", total, len(members))
	}
	// ORDER BY cm.role sorts by the channel_role ENUM's declaration order
	// ('owner', 'admin', 'member' — migration 000014), not alphabetically:
	// owner sorts before member even though 'm' < 'o'.
	if members[0].Role != models.ChannelRoleOwner || members[0].FirstName != "Chana" {
		t.Fatalf("ListMembers: expected owner Chana first (enum declaration order), got %q %q", members[0].Role, members[0].FirstName)
	}
	if members[1].Role != models.ChannelRoleMember || members[1].FirstName != "Beno" {
		t.Fatalf("ListMembers: expected member Beno second, got %q %q", members[1].Role, members[1].FirstName)
	}

	count, err := repo.GetMemberCount(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("GetMemberCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("GetMemberCount: expected 2, got %d", count)
	}

	if emptyCount, err := repo.GetMemberCount(ctxOwn, uuid.New()); err != nil || emptyCount != 0 {
		t.Fatalf("GetMemberCount unknown channel: expected 0, got %d (err %v)", emptyCount, err)
	}
}

func TestPostgresRepository_UserInfo(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	firstName, lastName, email, err := repo.GetUserInfo(ctxOwn, fx.userA)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if firstName != "Chana" || lastName != "Reader" || email == "" {
		t.Fatalf("GetUserInfo: expected Chana Reader with an email, got %s %s %q", firstName, lastName, email)
	}

	if _, _, _, err := repo.GetUserInfo(ctxOwn, uuid.New()); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUserInfo unknown user: expected ErrUserNotFound, got %v", err)
	}

	if exists, err := repo.UserExists(ctxOwn, fx.userA); err != nil || !exists {
		t.Fatalf("UserExists: expected true, got %v (err %v)", exists, err)
	}
	if exists, err := repo.UserExists(ctxOwn, uuid.New()); err != nil || exists {
		t.Fatalf("UserExists unknown user: expected false, got %v (err %v)", exists, err)
	}
}

func TestPostgresRepository_GetLastMessage(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	if last, err := repo.GetLastMessage(ctxOwn, fx.channel); err != nil || last != nil {
		t.Fatalf("GetLastMessage empty channel: expected nil, nil, got %v (err %v)", last, err)
	}

	base := time.Now().UTC()
	older := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "older", "lang": "german",
		"created_by": fx.userA, "created_at": base,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", older) })

	newer := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "newer", "lang": "german",
		"created_by": fx.userA, "created_at": base.Add(time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", newer) })

	reply := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "a reply", "lang": "german",
		"created_by": fx.userA, "created_at": base.Add(2 * time.Minute), "parent_message_id": newer,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", reply) })

	last, err := repo.GetLastMessage(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("GetLastMessage: %v", err)
	}
	if last == nil || last.ID != newer {
		t.Fatalf("GetLastMessage: expected the newer top-level message (replies excluded), got %v", last)
	}
	if last.SenderFirstName != "Chana" {
		t.Fatalf("GetLastMessage: expected sender Chana, got %s", last.SenderFirstName)
	}
}

func TestPostgresRepository_FindDMChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	// scanChannel maps pgx.ErrNoRows to ErrChannelNotFound — unlike GetLastMessage,
	// FindDMChannel does NOT special-case "no rows" into (nil, nil). The service
	// layer (service.go:582) treats ErrChannelNotFound as the "create a new DM" signal.
	if got, err := repo.FindDMChannel(ctxOwn, fx.userA, fx.userB, fx.tenant); got != nil || !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("FindDMChannel none exists: expected nil, ErrChannelNotFound, got %v (err %v)", got, err)
	}

	dmUser1, dmUser2 := fx.userA, fx.userB
	if dmUser1.String() > dmUser2.String() {
		dmUser1, dmUser2 = dmUser2, dmUser1
	}
	dmChannel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": fx.tenant, "name": "dm", "is_dm": true,
		"dm_user1": dmUser1, "dm_user2": dmUser2, "created_by": fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", dmChannel) })

	got, err := repo.FindDMChannel(ctxOwn, dmUser1, dmUser2, fx.tenant)
	if err != nil {
		t.Fatalf("FindDMChannel: %v", err)
	}
	if got == nil || got.ID != dmChannel {
		t.Fatalf("FindDMChannel: expected %s, got %v", dmChannel, got)
	}

	// The query matches dm_user1/dm_user2 positionally, not symmetrically —
	// swapping the argument order must not find the same row.
	if got, err := repo.FindDMChannel(ctxOwn, dmUser2, dmUser1, fx.tenant); got != nil || !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("FindDMChannel swapped args: expected ErrChannelNotFound (positional match, not symmetric), got %v (err %v)", got, err)
	}
}

func TestPostgresRepository_UnreadCounts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	joinedAt := time.Now().UTC()
	seedMembership(t, pool, fx.channel, fx.userA, fx.tenant, models.ChannelRoleOwner)

	before := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "before join", "lang": "german",
		"created_by": fx.userB, "created_at": joinedAt.Add(-time.Hour),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", before) })

	afterJoin := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "after join", "lang": "german",
		"created_by": fx.userB, "created_at": joinedAt.Add(time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", afterJoin) })

	count, err := repo.GetUnreadCount(ctxOwn, fx.channel, fx.userA)
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 1 {
		t.Fatalf("GetUnreadCount before any read: expected 1 (only the post-join message), got %d", count)
	}

	// Error/edge path: a user who never joined the channel has no membership
	// row to join against, so the count is zero rather than an error.
	if count, err := repo.GetUnreadCount(ctxOwn, fx.channel, fx.userB); err != nil || count != 0 {
		t.Fatalf("GetUnreadCount non-member: expected 0, got %d (err %v)", count, err)
	}

	if err := repo.UpdateLastRead(ctxOwn, fx.channel, fx.userA, joinedAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("UpdateLastRead: %v", err)
	}

	afterRead := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "after read", "lang": "german",
		"created_by": fx.userB, "created_at": joinedAt.Add(3 * time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", afterRead) })

	if count, err := repo.GetUnreadCount(ctxOwn, fx.channel, fx.userA); err != nil || count != 1 {
		t.Fatalf("GetUnreadCount after UpdateLastRead: expected 1 (only the message after last_read_at), got %d (err %v)", count, err)
	}

	counts, err := repo.GetUnreadCountsForUser(ctxOwn, fx.userA)
	if err != nil {
		t.Fatalf("GetUnreadCountsForUser: %v", err)
	}
	if counts[fx.channel] != 1 {
		t.Fatalf("GetUnreadCountsForUser: expected 1 for fx.channel, got %d", counts[fx.channel])
	}
}

// TestPostgresRepository_HasCallSessions proves both sides of
// fix-channel-delete-call-sessions-fk-crash: HasCallSessions correctly
// detects an attached call_sessions row, and — without that guard — deleting
// the channel really does fail with a raw FK violation (call_sessions.channel_id
// is ON DELETE NO ACTION), confirming the crash this unit prevents is real.
func TestPostgresRepository_HasCallSessions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	if has, err := repo.HasCallSessions(ctxOwn, fx.channel); err != nil || has {
		t.Fatalf("HasCallSessions before seeding: expected false, got %v (err %v)", has, err)
	}

	callSession := testutil.SeedRow(t, pool, "call_sessions", map[string]any{
		"tenant_id":    fx.tenant,
		"call_type":    "group",
		"status":       "ended",
		"room_name":    fmt.Sprintf("channel-hascalls-%s", uuid.New().String()[:8]),
		"initiator_id": fx.userA,
		"channel_id":   fx.channel,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "call_sessions", callSession) })

	if has, err := repo.HasCallSessions(ctxOwn, fx.channel); err != nil || !has {
		t.Fatalf("HasCallSessions after seeding: expected true, got %v (err %v)", has, err)
	}

	// A raw delete (bypassing the guard this unit adds to Service.Delete) must
	// still fail with the FK violation — otherwise the guard is defending
	// against a bug that no longer exists.
	if _, err := pool.Exec(ctxOwn, `DELETE FROM channels WHERE id = $1`, fx.channel); err == nil {
		t.Fatalf("raw DELETE with attached call_sessions: expected FK violation, got no error")
	}
}
