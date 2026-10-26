package video_test

// postgres_repository_db_test.go runs the call-session repository against the
// real schema as kmuhub_app (NOSUPERUSER NOBYPASSRLS), with RLS enforced.
// service_test.go only sees the in-memory mockRepo, so every SQL statement in
// postgres_repository.go was unexercised.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/video"
)

type videoFixture struct {
	pool   *pgxpool.Pool
	repo   *video.PostgresRepository
	tenant uuid.UUID
	ctx    context.Context
}

// newVideoFixture creates an isolated tenant. Cleanup removes the call rows
// first (participants cascade) because their tenant FK has no ON DELETE.
func newVideoFixture(t *testing.T) *videoFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Video Repo Tenant")
	t.Cleanup(func() {
		sys := testutil.WithSystemCtx(context.Background())
		for _, table := range []string{"call_sessions", "users"} {
			if _, err := pool.Exec(sys, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table), tenant); err != nil {
				t.Logf("cleanup %s: %v", table, err)
			}
		}
		testutil.CleanupRow(t, pool, "tenants", tenant)
	})

	return &videoFixture{
		pool:   pool,
		repo:   video.NewPostgresRepository(pool),
		tenant: tenant,
		ctx:    testutil.WithTenantCtx(context.Background(), tenant),
	}
}

func (f *videoFixture) user(t *testing.T) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     f.tenant,
		"email":         fmt.Sprintf("wp-video-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func (f *videoFixture) session(t *testing.T, initiator uuid.UUID, createdAt time.Time) *video.CallSession {
	t.Helper()
	s := &video.CallSession{
		ID:          uuid.New(),
		TenantID:    f.tenant,
		CallType:    video.CallTypeGroup,
		Status:      video.CallStatusRinging,
		RoomName:    "room-" + uuid.NewString(),
		InitiatorID: initiator,
		CreatedAt:   createdAt,
	}
	require.NoError(t, f.repo.CreateCallSession(f.ctx, s))
	return s
}

func (f *videoFixture) join(t *testing.T, callID, userID uuid.UUID) {
	t.Helper()
	require.NoError(t, f.repo.AddParticipant(f.ctx, &video.CallParticipant{
		CallID: callID, TenantID: f.tenant, UserID: userID,
		JoinedAt: time.Now().UTC(), HasVideo: true, HasAudio: true,
	}))
}

func now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

func TestVideoRepo_CreateAndGetRoundtrip(t *testing.T) {
	f := newVideoFixture(t)
	initiator := f.user(t)
	created := now()
	ended := created.Add(90 * time.Second)
	dur := 90

	s := &video.CallSession{
		ID: uuid.New(), TenantID: f.tenant, CallType: video.CallTypeOneToOne,
		Status: video.CallStatusEnded, RoomName: "room-" + uuid.NewString(),
		InitiatorID: initiator, CreatedAt: created, EndedAt: &ended, DurationSeconds: &dur,
	}
	require.NoError(t, f.repo.CreateCallSession(f.ctx, s))

	got, err := f.repo.GetCallSession(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, f.tenant, got.TenantID, "tenant_id must be read back: JoinCall inherits it into call_participants")
	assert.Equal(t, video.CallTypeOneToOne, got.CallType)
	assert.Equal(t, video.CallStatusEnded, got.Status)
	assert.Equal(t, s.RoomName, got.RoomName)
	assert.Equal(t, initiator, got.InitiatorID)
	assert.Nil(t, got.ChannelID)
	assert.True(t, created.Equal(got.CreatedAt))
	require.NotNil(t, got.EndedAt)
	assert.True(t, ended.Equal(*got.EndedAt))
	require.NotNil(t, got.DurationSeconds)
	assert.Equal(t, 90, *got.DurationSeconds)
}

func TestVideoRepo_GetCallSession_NotFound(t *testing.T) {
	f := newVideoFixture(t)

	_, err := f.repo.GetCallSession(f.ctx, uuid.New())
	assert.ErrorIs(t, err, video.ErrNotFound)
}

func TestVideoRepo_GetCallSession_CrossTenantIsNotFound(t *testing.T) {
	f := newVideoFixture(t)
	s := f.session(t, f.user(t), now())

	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, "Video Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", other) })
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	_, err := f.repo.GetCallSession(otherCtx, s.ID)
	assert.ErrorIs(t, err, video.ErrNotFound, "RLS must hide another tenant's session")
	_, err = f.repo.GetCallSessionByRoomName(otherCtx, s.RoomName)
	assert.ErrorIs(t, err, video.ErrNotFound, "RLS must hide another tenant's session by room name")

	_, err = f.repo.GetCallSession(f.ctx, s.ID)
	assert.NoError(t, err, "owner tenant still sees it")
}

func TestVideoRepo_CreateCallSession_DuplicateRoomName(t *testing.T) {
	f := newVideoFixture(t)
	initiator := f.user(t)
	first := f.session(t, initiator, now())

	dup := &video.CallSession{
		ID: uuid.New(), TenantID: f.tenant, CallType: video.CallTypeGroup,
		Status: video.CallStatusRinging, RoomName: first.RoomName,
		InitiatorID: initiator, CreatedAt: now(),
	}
	assert.Error(t, f.repo.CreateCallSession(f.ctx, dup), "room_name is UNIQUE")
}

func TestVideoRepo_CreateCallSession_RejectsInvalidEnums(t *testing.T) {
	f := newVideoFixture(t)
	initiator := f.user(t)

	bad := &video.CallSession{
		ID: uuid.New(), TenantID: f.tenant, CallType: "conference",
		Status: video.CallStatusRinging, RoomName: "room-" + uuid.NewString(),
		InitiatorID: initiator, CreatedAt: now(),
	}
	assert.Error(t, f.repo.CreateCallSession(f.ctx, bad), "call_type CHECK constraint")

	bad.CallType = video.CallTypeGroup
	bad.Status = "paused"
	assert.Error(t, f.repo.CreateCallSession(f.ctx, bad), "status CHECK constraint")
}

func TestVideoRepo_CreateCallSession_ForeignTenantRejectedByRLS(t *testing.T) {
	f := newVideoFixture(t)
	initiator := f.user(t)

	s := &video.CallSession{
		ID: uuid.New(), TenantID: uuid.New(), CallType: video.CallTypeGroup,
		Status: video.CallStatusRinging, RoomName: "room-" + uuid.NewString(),
		InitiatorID: initiator, CreatedAt: now(),
	}
	assert.Error(t, f.repo.CreateCallSession(f.ctx, s), "WITH CHECK must refuse a tenant_id other than the session tenant")
}

func TestVideoRepo_GetCallSessionByRoomName(t *testing.T) {
	f := newVideoFixture(t)
	s := f.session(t, f.user(t), now())
	f.session(t, f.user(t), now()) // a second room must not be returned

	got, err := f.repo.GetCallSessionByRoomName(f.ctx, s.RoomName)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, f.tenant, got.TenantID)
	assert.Equal(t, s.InitiatorID, got.InitiatorID)

	_, err = f.repo.GetCallSessionByRoomName(f.ctx, "room-does-not-exist")
	assert.ErrorIs(t, err, video.ErrNotFound)
}

func TestVideoRepo_UpdateCallSession_PersistsStatusAndDuration(t *testing.T) {
	f := newVideoFixture(t)
	initiator := f.user(t)
	s := f.session(t, initiator, now())
	other := f.session(t, initiator, now())

	ended := now().Add(time.Minute)
	dur := 60
	s.Status = video.CallStatusEnded
	s.EndedAt = &ended
	s.DurationSeconds = &dur
	require.NoError(t, f.repo.UpdateCallSession(f.ctx, s))

	got, err := f.repo.GetCallSession(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, video.CallStatusEnded, got.Status)
	require.NotNil(t, got.EndedAt)
	assert.True(t, ended.Equal(*got.EndedAt))
	require.NotNil(t, got.DurationSeconds)
	assert.Equal(t, 60, *got.DurationSeconds)
	assert.Equal(t, s.RoomName, got.RoomName, "update touches only status/ended_at/duration")

	untouched, err := f.repo.GetCallSession(f.ctx, other.ID)
	require.NoError(t, err)
	assert.Equal(t, video.CallStatusRinging, untouched.Status, "WHERE id = $4 must not hit other sessions")
	assert.Nil(t, untouched.EndedAt)
}

func TestVideoRepo_UpdateCallSession_CrossTenantIsNoOp(t *testing.T) {
	f := newVideoFixture(t)
	s := f.session(t, f.user(t), now())

	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, "Video Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", other) })

	hijack := *s
	hijack.Status = video.CallStatusEnded
	require.NoError(t, f.repo.UpdateCallSession(testutil.WithTenantCtx(context.Background(), other), &hijack))

	got, err := f.repo.GetCallSession(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, video.CallStatusRinging, got.Status, "RLS must make a foreign-tenant UPDATE match zero rows")
}

func TestVideoRepo_Participants_AddListRemoveCount(t *testing.T) {
	f := newVideoFixture(t)
	a, b := f.user(t), f.user(t)
	s := f.session(t, a, now())

	base := now().Add(-time.Hour)
	require.NoError(t, f.repo.AddParticipant(f.ctx, &video.CallParticipant{
		CallID: s.ID, TenantID: f.tenant, UserID: b, JoinedAt: base.Add(time.Minute), HasVideo: false, HasAudio: true,
	}))
	require.NoError(t, f.repo.AddParticipant(f.ctx, &video.CallParticipant{
		CallID: s.ID, TenantID: f.tenant, UserID: a, JoinedAt: base, HasVideo: true, HasAudio: false,
	}))

	parts, err := f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	require.Len(t, parts, 2)
	assert.Equal(t, a, parts[0].UserID, "ordered by joined_at ASC")
	assert.Equal(t, b, parts[1].UserID)
	assert.True(t, parts[0].HasVideo)
	assert.False(t, parts[0].HasAudio)
	assert.False(t, parts[1].HasVideo)
	assert.True(t, parts[1].HasAudio)
	assert.Equal(t, s.ID, parts[0].CallID)
	assert.Nil(t, parts[0].LeftAt)
	assert.True(t, base.Equal(parts[0].JoinedAt))

	// A second call in the same tenant must not leak into the first one's
	// participant list or active count.
	bystander := f.session(t, a, now())
	f.join(t, bystander.ID, a)
	f.join(t, bystander.ID, f.user(t))
	f.join(t, bystander.ID, f.user(t))

	n, err := f.repo.CountActiveParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	require.NoError(t, f.repo.RemoveParticipant(f.ctx, s.ID, a))
	n, err = f.repo.CountActiveParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "a participant who left no longer counts")

	parts, err = f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	require.Len(t, parts, 2, "GetParticipants keeps participants who left")
	require.NotNil(t, parts[0].LeftAt)
	assert.Nil(t, parts[1].LeftAt, "removing a leaves b untouched")
}

func TestVideoRepo_RemoveParticipant_KeepsFirstLeftAt(t *testing.T) {
	f := newVideoFixture(t)
	a := f.user(t)
	s := f.session(t, a, now())
	f.join(t, s.ID, a)

	require.NoError(t, f.repo.RemoveParticipant(f.ctx, s.ID, a))
	first, err := f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	require.NotNil(t, first[0].LeftAt)
	firstLeft := *first[0].LeftAt

	time.Sleep(5 * time.Millisecond)
	require.NoError(t, f.repo.RemoveParticipant(f.ctx, s.ID, a), "removing twice is not an error")
	second, err := f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.True(t, firstLeft.Equal(*second[0].LeftAt), "left_at IS NULL guard keeps the original leave time")
}

func TestVideoRepo_AddParticipant_RejoinResetsLeftAt(t *testing.T) {
	f := newVideoFixture(t)
	a := f.user(t)
	s := f.session(t, a, now())
	f.join(t, s.ID, a)
	require.NoError(t, f.repo.RemoveParticipant(f.ctx, s.ID, a))

	rejoin := now().Add(time.Minute)
	require.NoError(t, f.repo.AddParticipant(f.ctx, &video.CallParticipant{
		CallID: s.ID, TenantID: f.tenant, UserID: a, JoinedAt: rejoin, HasVideo: false, HasAudio: false,
	}))

	parts, err := f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	require.Len(t, parts, 1, "(call_id, user_id) upsert, no duplicate row")
	assert.Nil(t, parts[0].LeftAt, "rejoin clears left_at")
	assert.True(t, rejoin.Equal(parts[0].JoinedAt), "rejoin refreshes joined_at")
	assert.False(t, parts[0].HasVideo, "rejoin takes the new media flags")
	assert.False(t, parts[0].HasAudio)

	n, err := f.repo.CountActiveParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestVideoRepo_AddParticipant_ForeignTenantRejectedByRLS(t *testing.T) {
	f := newVideoFixture(t)
	a := f.user(t)
	s := f.session(t, a, now())

	err := f.repo.AddParticipant(f.ctx, &video.CallParticipant{
		CallID: s.ID, TenantID: uuid.New(), UserID: a, JoinedAt: now(), HasVideo: true, HasAudio: true,
	})
	assert.Error(t, err, "participant tenant_id must match the session tenant")
}

func TestVideoRepo_ParticipantReads_CrossTenantSeeNothing(t *testing.T) {
	f := newVideoFixture(t)
	a := f.user(t)
	s := f.session(t, a, now())
	f.join(t, s.ID, a)

	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, "Video Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", other) })
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	parts, err := f.repo.GetParticipants(otherCtx, s.ID)
	require.NoError(t, err)
	assert.Empty(t, parts)

	n, err := f.repo.CountActiveParticipants(otherCtx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	require.NoError(t, f.repo.RemoveParticipant(otherCtx, s.ID, a))
	n, err = f.repo.CountActiveParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "foreign-tenant RemoveParticipant must not mark the user as left")

	active, err := f.repo.ListActiveCallsForUser(otherCtx, a)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestVideoRepo_GetParticipants_EmptyCall(t *testing.T) {
	f := newVideoFixture(t)
	s := f.session(t, f.user(t), now())

	parts, err := f.repo.GetParticipants(f.ctx, s.ID)
	require.NoError(t, err)
	assert.Empty(t, parts)
}

func TestVideoRepo_ListActiveCallsForUser(t *testing.T) {
	f := newVideoFixture(t)
	me, peer := f.user(t), f.user(t)

	base := now().Add(-time.Hour)
	older := f.session(t, me, base)
	newer := f.session(t, me, base.Add(10*time.Minute))
	ended := f.session(t, me, base.Add(20*time.Minute))
	left := f.session(t, me, base.Add(30*time.Minute))
	foreign := f.session(t, peer, base.Add(40*time.Minute))

	for _, s := range []*video.CallSession{older, newer, ended, left} {
		f.join(t, s.ID, me)
	}
	f.join(t, foreign.ID, peer) // me never joined this one

	ended.Status = video.CallStatusEnded
	endedAt := now()
	ended.EndedAt = &endedAt
	require.NoError(t, f.repo.UpdateCallSession(f.ctx, ended))
	require.NoError(t, f.repo.RemoveParticipant(f.ctx, left.ID, me))

	got, err := f.repo.ListActiveCallsForUser(f.ctx, me)
	require.NoError(t, err)
	require.Len(t, got, 2, "only unended calls the user has not left")
	assert.Equal(t, newer.ID, got[0].ID, "newest first")
	assert.Equal(t, older.ID, got[1].ID)
	assert.Equal(t, f.tenant, got[0].TenantID)
	assert.Equal(t, newer.RoomName, got[0].RoomName)
	assert.Equal(t, video.CallTypeGroup, got[0].CallType)
	assert.Equal(t, video.CallStatusRinging, got[0].Status)
	assert.Equal(t, me, got[0].InitiatorID)

	// A user in no call gets an empty result, not an error.
	none, err := f.repo.ListActiveCallsForUser(f.ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestVideoRepo_CancelledContext_SurfacesErrors(t *testing.T) {
	f := newVideoFixture(t)
	a := f.user(t)
	s := f.session(t, a, now())

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()

	assert.Error(t, f.repo.CreateCallSession(ctx, s))
	_, err := f.repo.GetCallSession(ctx, s.ID)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, video.ErrNotFound, "a DB failure is not a missing row")
	_, err = f.repo.GetCallSessionByRoomName(ctx, s.RoomName)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, video.ErrNotFound)
	assert.Error(t, f.repo.UpdateCallSession(ctx, s))
	_, err = f.repo.ListActiveCallsForUser(ctx, a)
	assert.Error(t, err)
	assert.Error(t, f.repo.AddParticipant(ctx, &video.CallParticipant{CallID: s.ID, TenantID: f.tenant, UserID: a}))
	assert.Error(t, f.repo.RemoveParticipant(ctx, s.ID, a))
	_, err = f.repo.GetParticipants(ctx, s.ID)
	assert.Error(t, err)
	_, err = f.repo.CountActiveParticipants(ctx, s.ID)
	assert.Error(t, err)
}
