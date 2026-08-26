package presence

// Covers RedisStore against a real Redis protocol implementation (miniredis)
// instead of the mockStore used in service_test.go — see BACKLOG.yml unit
// cov-work-customfield-and-presence-zero-coverage. The store had zero
// coverage; the questions the unit asks are answered inline below:
//
//   - TTL: every write sets presenceTTL (90s) — TestRedisStore_SetPresence_SetsTTL,
//     TestRedisStore_UpdateLastActivity_RefreshesTTL.
//   - Tenant in the key: presenceKey is "presence:" + userID with NO tenant
//     component (redis_store.go:51-53). userID is a globally-unique UUID, so two
//     tenants cannot collide on the same key by chance — but nothing in the store
//     or in the key itself proves a caller is looking at its own tenant's user.
//     TestRedisStore_KeyHasNoTenantComponent pins that shape. The real exposure
//     is one layer up: WebSocketHub.handlePresenceSubscribe
//     (internal/server/websocket.go:1116) accepts an arbitrary user_id list from
//     any authenticated client with no tenant check at all, so a caller who
//     knows/guesses a foreign user's UUID is subscribed to their presence
//     broadcasts. That is a real cross-tenant info leak, but fixing the
//     websocket handler is a behaviour change outside this coverage unit's
//     scope — filed as fix-websocket-presence-subscribe-missing-tenant-check.

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisPresenceStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })
	return NewRedisStore(rc), mr
}

func TestRedisStore_SetPresence_SetsTTL(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, store.SetPresence(ctx, userID, PresenceOnline, false))

	ttl := mr.TTL(presenceKey(userID))
	assert.Equal(t, presenceTTL, ttl, "SetPresence must set the fixed presence TTL")
}

func TestRedisStore_KeyHasNoTenantComponent(t *testing.T) {
	// Pins the current key shape documented in the package comment above:
	// presence keys carry no tenant, only the user id.
	userID := uuid.New().String()
	assert.Equal(t, "presence:"+userID, presenceKey(userID))
}

func TestRedisStore_GetPresence_NotFound_ReturnsNilNoError(t *testing.T) {
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()

	p, err := store.GetPresence(ctx, uuid.New().String())
	require.NoError(t, err)
	assert.Nil(t, p, "missing key must mean offline (nil, no error), not an error")
}

func TestRedisStore_SetThenGetPresence_RoundTrips(t *testing.T) {
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, store.SetPresence(ctx, userID, PresenceDND, true))

	p, err := store.GetPresence(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, userID, p.UserID.String())
	assert.Equal(t, PresenceDND, p.Status)
	assert.True(t, p.ManualOverride)
	require.NotNil(t, p.LastActivity)
}

func TestRedisStore_GetPresence_CorruptedJSON_ReturnsError(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, mr.Set(presenceKey(userID), "{not-json"))

	_, err := store.GetPresence(ctx, userID)
	assert.Error(t, err)
}

func TestRedisStore_RemovePresence(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, store.SetPresence(ctx, userID, PresenceOnline, false))
	require.True(t, mr.Exists(presenceKey(userID)))

	require.NoError(t, store.RemovePresence(ctx, userID))
	assert.False(t, mr.Exists(presenceKey(userID)))
}

func TestRedisStore_RemovePresence_MissingKey_NoError(t *testing.T) {
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()

	// DEL on a missing key is a no-op in Redis, not an error.
	assert.NoError(t, store.RemovePresence(ctx, uuid.New().String()))
}

func TestRedisStore_UpdateLastActivity_NoExisting_SetsOnline(t *testing.T) {
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, store.UpdateLastActivity(ctx, userID))

	p, err := store.GetPresence(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, PresenceOnline, p.Status)
	assert.False(t, p.ManualOverride)
}

func TestRedisStore_UpdateLastActivity_PreservesStatusAndRefreshesTTL(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, store.SetPresence(ctx, userID, PresenceDND, true))
	firstActivity, err := store.GetPresence(ctx, userID)
	require.NoError(t, err)

	// Let the TTL tick down, then confirm UpdateLastActivity resets it and
	// leaves the manually-set status untouched.
	mr.FastForward(30 * time.Second)
	require.NoError(t, store.UpdateLastActivity(ctx, userID))

	assert.Equal(t, presenceTTL, mr.TTL(presenceKey(userID)), "activity update must refresh the TTL")

	updated, err := store.GetPresence(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, PresenceDND, updated.Status, "status must survive an activity-only update")
	assert.True(t, updated.ManualOverride)
	assert.True(t, updated.LastActivity.After(*firstActivity.LastActivity) || updated.LastActivity.Equal(*firstActivity.LastActivity),
		"last_activity must not move backwards")
}

func TestRedisStore_UpdateLastActivity_CorruptedExisting_ReturnsError(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()
	userID := uuid.New().String()

	require.NoError(t, mr.Set(presenceKey(userID), "{not-json"))

	assert.Error(t, store.UpdateLastActivity(ctx, userID))
}

func TestRedisStore_GetBulkPresence_Empty_ReturnsEmptyMap(t *testing.T) {
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()

	result, err := store.GetBulkPresence(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestRedisStore_GetBulkPresence_MixedPresentAbsentCorrupted(t *testing.T) {
	store, mr := setupRedisPresenceStore(t)
	ctx := context.Background()

	online := uuid.New().String()
	offline := uuid.New().String() // never written
	corrupted := uuid.New().String()

	require.NoError(t, store.SetPresence(ctx, online, PresenceOnline, false))
	require.NoError(t, mr.Set(presenceKey(corrupted), "{not-json"))

	result, err := store.GetBulkPresence(ctx, []string{online, offline, corrupted})
	require.NoError(t, err)

	require.Contains(t, result, online)
	assert.Equal(t, PresenceOnline, result[online].Status)

	// Missing and corrupted entries are both silently skipped by
	// GetBulkPresence itself — GetBulkPresence only fills in "offline" for
	// keys the caller asked for that never round-tripped, and the presence
	// Service layer (not the store) backfills unmapped ids as offline.
	assert.NotContains(t, result, offline)
	assert.NotContains(t, result, corrupted)
}

func TestRedisStore_GetBulkPresence_TenantMixedUserIDs_NoServerSideFilter(t *testing.T) {
	// Documents the finding in the package comment: GetBulkPresence answers
	// for whatever user ids it is given, with no tenant check of its own.
	// Isolation depends entirely on callers only ever passing ids from their
	// own tenant.
	store, _ := setupRedisPresenceStore(t)
	ctx := context.Background()

	tenantAUser := uuid.New().String()
	tenantBUser := uuid.New().String()
	require.NoError(t, store.SetPresence(ctx, tenantAUser, PresenceOnline, false))
	require.NoError(t, store.SetPresence(ctx, tenantBUser, PresenceDND, true))

	result, err := store.GetBulkPresence(ctx, []string{tenantAUser, tenantBUser})
	require.NoError(t, err)
	assert.Len(t, result, 2, "the store itself does not filter by tenant")
}
