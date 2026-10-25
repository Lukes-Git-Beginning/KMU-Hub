package reaction

// A reaction is a write on a message, and the summary lists who reacted, so both
// follow the visibility of the message's channel: public channels are open to the
// tenant, private channels (and DMs) only to their members. These tests run the
// real service over the real repository as kmuhub_app (RLS enforced).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type accessFixture struct {
	pool                    *pgxpool.Pool
	svc                     *Service
	ctx                     context.Context
	member, outsider        uuid.UUID
	privateMsg, publicMsg   uuid.UUID
	privateChan, publicChan uuid.UUID
}

func newAccessFixture(t *testing.T) *accessFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Reaction Access")
	f := &accessFixture{pool: pool, svc: NewService(NewPostgresRepository(pool)), ctx: testutil.WithTenantCtx(context.Background(), tenantID)}

	mkUser := func(first string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id": tenantID, "email": first + "-" + uuid.NewString()[:8] + "@reaction-access.local",
			"password_hash": "$argon2id$v=19$test", "first_name": first, "last_name": "Access",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	f.member, f.outsider = mkUser("Mia"), mkUser("Otto")

	mkChannel := func(name string, private bool) (channelID, messageID uuid.UUID) {
		channelID = testutil.SeedRow(t, pool, "channels", map[string]any{
			"tenant_id": tenantID, "name": name, "is_private": private, "created_by": f.member,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channelID) })
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`INSERT INTO channel_memberships (channel_id, user_id, tenant_id) VALUES ($1, $2, $3)`, channelID, f.member, tenantID)
		require.NoError(t, err)
		messageID = testutil.SeedRow(t, pool, "messages", map[string]any{
			"tenant_id": tenantID, "channel_id": channelID, "content": "hi", "lang": "simple",
			"created_by": f.member, "created_at": time.Now().UTC(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", messageID) })
		return channelID, messageID
	}
	f.privateChan, f.privateMsg = mkChannel("reaction-private-"+uuid.NewString()[:8], true)
	f.publicChan, f.publicMsg = mkChannel("reaction-public-"+uuid.NewString()[:8], false)
	return f
}

func TestReactionAccess_PrivateChannelNeedsMembership_RealSQL(t *testing.T) {
	f := newAccessFixture(t)

	// Non-member: not found for every entry point, and nothing is written.
	_, err := f.svc.ToggleReaction(f.ctx, f.privateMsg, f.outsider, "👍")
	require.ErrorIs(t, err, ErrMessageNotFound)
	_, err = f.svc.ListReactions(f.ctx, f.privateMsg, f.outsider)
	require.ErrorIs(t, err, ErrMessageNotFound)
	_, err = f.svc.GetReactionSummary(f.ctx, f.privateMsg, f.outsider)
	require.ErrorIs(t, err, ErrMessageNotFound)

	// Member: allowed, and the response names the channel for the fan-out.
	res, err := f.svc.ToggleReaction(f.ctx, f.privateMsg, f.member, "👍")
	require.NoError(t, err)
	assert.True(t, res.Added)
	assert.Equal(t, f.privateChan, res.ChannelID)

	// The outsider's attempt left no row: only the member's reaction exists.
	list, err := f.svc.ListReactions(f.ctx, f.privateMsg, f.member)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, f.member, list[0].UserID)

	// Batch: the private message disappears for the outsider, the public one stays.
	_, err = f.svc.ToggleReaction(f.ctx, f.publicMsg, f.member, "🎉")
	require.NoError(t, err)
	batch, err := f.svc.GetBatchReactionSummary(f.ctx, []uuid.UUID{f.privateMsg, f.publicMsg}, f.outsider)
	require.NoError(t, err)
	assert.NotContains(t, batch, f.privateMsg, "an outsider must not see who reacted in a private channel")
	assert.Contains(t, batch, f.publicMsg)

	batch, err = f.svc.GetBatchReactionSummary(f.ctx, []uuid.UUID{f.privateMsg, f.publicMsg}, f.member)
	require.NoError(t, err)
	assert.Contains(t, batch, f.privateMsg)
}

func TestReactionAccess_PublicChannelIsOpenToTheTenant_RealSQL(t *testing.T) {
	f := newAccessFixture(t)

	res, err := f.svc.ToggleReaction(f.ctx, f.publicMsg, f.outsider, "👍")
	require.NoError(t, err)
	assert.True(t, res.Added)
	assert.Equal(t, f.publicChan, res.ChannelID)
}

func TestReactionAccess_ForeignTenantMessageIsNotFound_RealSQL(t *testing.T) {
	f := newAccessFixture(t)

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, f.pool, otherTenant, "Reaction Access Other")
	otherCtx := testutil.WithTenantCtx(context.Background(), otherTenant)

	// Even a public channel of another tenant stays invisible (RLS).
	_, err := f.svc.ToggleReaction(otherCtx, f.publicMsg, uuid.New(), "👍")
	require.ErrorIs(t, err, ErrMessageNotFound)
	_, err = f.svc.ToggleReaction(f.ctx, uuid.New(), f.member, "👍")
	require.ErrorIs(t, err, ErrMessageNotFound, "unknown message")
}
