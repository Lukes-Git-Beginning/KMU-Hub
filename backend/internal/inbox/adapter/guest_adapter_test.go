package adapter

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGuestAdapter_Channel(t *testing.T) {
	assert.Equal(t, "guest", NewGuestAdapter(nil).Channel())
}

func TestGuestAdapter_HandleReplyForwardAndMarkRead(t *testing.T) {
	a := NewGuestAdapter(nil)

	assert.NoError(t, a.HandleReply(context.Background(), "msg", uuid.New(), "x"), "agents reply via normal chat")
	assert.ErrorIs(t, a.HandleForward(context.Background(), "msg", uuid.New(), "a@b.de", ""), ErrForwardNotSupported)
	assert.NoError(t, a.MarkReadOnSource(context.Background(), "msg", uuid.New()))
}

// guestFixture is one tenant with a guest-enabled channel, a member agent, a
// user who is not in the channel, and one guest session.
type guestFixture struct {
	tenant    uuid.UUID
	member    uuid.UUID
	outsider  uuid.UUID
	channel   uuid.UUID
	sessionID uuid.UUID
}

func seedGuestFixture(t *testing.T, pool *pgxpool.Pool) guestFixture {
	t.Helper()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Inbox Guest Adapter Tenant")

	newUser := func(label string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         fmt.Sprintf("inbox-guest-%s-%s@test.local", label, uuid.NewString()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	member, outsider := newUser("member"), newUser("outsider")

	channel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": tenant, "name": "Kundenchat", "created_by": member, "is_guest_enabled": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channel) })

	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id, role, joined_at) VALUES ($1, $2, $3, 'member', $4)`,
		channel, member, tenant, time.Now().UTC())
	require.NoError(t, err)

	session := testutil.SeedRow(t, pool, "guest_sessions", map[string]any{
		"tenant_id": tenant, "token_hash": "hash-" + uuid.NewString(), "channel_id": channel,
		"display_name": "Gast Müller", "last_activity_at": time.Now().UTC(), "is_active": true,
		"created_at": time.Now().UTC(), "expires_at": time.Now().UTC().Add(24 * time.Hour),
	})

	return guestFixture{tenant: tenant, member: member, outsider: outsider, channel: channel, sessionID: session}
}

// guestMessage seeds a message written by the guest session.
func (fx guestFixture) guestMessage(t *testing.T, pool *pgxpool.Pool, content string, createdAt time.Time, deleted bool) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": content, "lang": "german",
		"guest_session_id": fx.sessionID, "created_at": createdAt, "is_deleted": deleted,
	})
}

func TestGuestAdapter_FetchNewMessages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedGuestFixture(t, pool)
	adapter := NewGuestAdapter(pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	older := fx.guestMessage(t, pool, "Hallo, ich habe eine Frage", base, false)
	newer := fx.guestMessage(t, pool, strings.Repeat("z", 250), base.Add(time.Minute), false)
	fx.guestMessage(t, pool, "wieder gelöscht", base.Add(2*time.Minute), true)

	// An agent reply in the same channel carries created_by, not a guest session.
	testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channel, "content": "Antwort vom Agenten", "lang": "german",
		"created_by": fx.member, "created_at": base.Add(3 * time.Minute),
	})

	t.Run("maps guest messages newest first", func(t *testing.T) {
		got, err := adapter.FetchNewMessages(ctx, fx.member, base.Add(-time.Minute))
		require.NoError(t, err)
		require.Len(t, got, 2, "deleted messages and agent replies are excluded")

		assert.Equal(t, newer.String(), got[0].SourceID)
		assert.Equal(t, older.String(), got[1].SourceID)

		m := got[1]
		assert.NotEqual(t, uuid.Nil, m.ID)
		assert.Equal(t, fx.member, m.UserID)
		assert.Equal(t, "guest", m.Channel)
		assert.Equal(t, "Gast Müller", m.SenderName)
		assert.Equal(t, "Kundenchat", m.Subject, "subject is the channel name")
		assert.Equal(t, "Hallo, ich habe eine Frage", m.Preview)
		assert.Equal(t, "/chat/"+fx.channel.String(), m.DeepLink)
		assert.True(t, base.Equal(m.ReceivedAt), "received_at = %s, want %s", m.ReceivedAt, base)

		assert.Len(t, got[0].Preview, 200, "preview is capped at 200 bytes")
	})

	t.Run("since is exclusive", func(t *testing.T) {
		got, err := adapter.FetchNewMessages(ctx, fx.member, base)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, newer.String(), got[0].SourceID)

		got, err = adapter.FetchNewMessages(ctx, fx.member, base.Add(time.Minute))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("only channel members see the conversation", func(t *testing.T) {
		got, err := adapter.FetchNewMessages(ctx, fx.outsider, base.Add(-time.Minute))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("another tenant sees nothing even for a member id", func(t *testing.T) {
		other := uuid.New()
		testutil.EnsureTenant(t, pool, other, "Inbox Guest Adapter Other Tenant")
		otherCtx := testutil.WithTenantCtx(context.Background(), other)

		got, err := adapter.FetchNewMessages(otherCtx, fx.member, base.Add(-time.Minute))
		require.NoError(t, err)
		assert.Empty(t, got, "RLS hides the messages of a foreign tenant")
	})
}

func TestGuestAdapter_FetchNewMessages_QueryErrorIsWrapped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	ctx, cancel := context.WithCancel(testutil.WithTenantCtx(context.Background(), uuid.New()))
	cancel()

	got, err := NewGuestAdapter(pool).FetchNewMessages(ctx, uuid.New(), time.Time{})
	assert.Nil(t, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "guest adapter fetch")
}
