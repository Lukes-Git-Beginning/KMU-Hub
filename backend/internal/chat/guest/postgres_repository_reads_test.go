package guest

// Covers the read methods on PostgresRepository that tenant_write_test.go
// does not exercise: GetSessionByTokenHash, GetSessionByID,
// ListSessionsByChannel, CleanupExpiredSessions and GetConfigByChannel.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type guestReadFixture struct {
	tenant  uuid.UUID
	user    uuid.UUID
	channel uuid.UUID
}

func newGuestReadFixture(t *testing.T, pool *pgxpool.Pool, guestEnabled bool) guestReadFixture {
	t.Helper()
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Guest Read Tenant")

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("guest-read-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })

	channel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":        tenant,
		"name":             "guest-read-test",
		"created_by":       user,
		"is_guest_enabled": guestEnabled,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channel) })

	return guestReadFixture{tenant: tenant, user: user, channel: channel}
}

func seedGuestSession(t *testing.T, pool *pgxpool.Pool, tenantID, channelID uuid.UUID, tokenHash string, isActive bool, expiresAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "guest_sessions", map[string]any{
		"tenant_id":        tenantID,
		"token_hash":       tokenHash,
		"channel_id":       channelID,
		"display_name":     "Reader Guest",
		"last_activity_at": time.Now().UTC(),
		"is_active":        isActive,
		"created_at":       time.Now().UTC(),
		"expires_at":       expiresAt,
	})
}

// seedGuestSessionWithIP mirrors seedGuestSession but sets ip_address to a
// real value -- ip_address is an INET column, and the pgx scan bug this file
// covers only reproduces when a row actually carries a non-NULL IP.
func seedGuestSessionWithIP(t *testing.T, pool *pgxpool.Pool, tenantID, channelID uuid.UUID, tokenHash string, ipAddress string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "guest_sessions", map[string]any{
		"tenant_id":        tenantID,
		"token_hash":       tokenHash,
		"channel_id":       channelID,
		"display_name":     "Reader Guest",
		"ip_address":       ipAddress,
		"last_activity_at": time.Now().UTC(),
		"is_active":        true,
		"created_at":       time.Now().UTC(),
		"expires_at":       time.Now().UTC().Add(24 * time.Hour),
	})
}

func TestPostgresRepository_GetSessionByTokenHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	tokenHash := fmt.Sprintf("hash-%s", uuid.New())
	sessID := seedGuestSession(t, pool, fx.tenant, fx.channel, tokenHash, true, time.Now().UTC().Add(24*time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessID) })

	got, err := repo.GetSessionByTokenHash(ctxOwn, tokenHash)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if got.ID != sessID {
		t.Fatalf("GetSessionByTokenHash: expected session %s, got %s", sessID, got.ID)
	}

	if _, err := repo.GetSessionByTokenHash(ctxOwn, "does-not-exist"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSessionByTokenHash (unknown hash): expected ErrSessionNotFound, got %v", err)
	}
}

// TestPostgresRepository_GetSessionByTokenHash_ChannelNotGuestEnabled covers
// the JOIN condition (c.is_guest_enabled = true): a valid, unexpired session
// on a channel where guest chat has since been disabled must not resolve.
func TestPostgresRepository_GetSessionByTokenHash_ChannelNotGuestEnabled(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, false)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	tokenHash := fmt.Sprintf("hash-%s", uuid.New())
	sessID := seedGuestSession(t, pool, fx.tenant, fx.channel, tokenHash, true, time.Now().UTC().Add(24*time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessID) })

	if _, err := repo.GetSessionByTokenHash(ctxOwn, tokenHash); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSessionByTokenHash (guest chat disabled on channel): expected ErrSessionNotFound, got %v", err)
	}
}

func TestPostgresRepository_GetSessionByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	sessID := seedGuestSession(t, pool, fx.tenant, fx.channel, fmt.Sprintf("hash-%s", uuid.New()), true, time.Now().UTC().Add(24*time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessID) })

	got, err := repo.GetSessionByID(ctxOwn, sessID)
	if err != nil {
		t.Fatalf("GetSessionByID: %v", err)
	}
	if got.ChannelID != fx.channel {
		t.Fatalf("GetSessionByID: expected channel %s, got %s", fx.channel, got.ChannelID)
	}

	if _, err := repo.GetSessionByID(ctxOwn, uuid.New()); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSessionByID (nonexistent): expected ErrSessionNotFound, got %v", err)
	}
}

func TestPostgresRepository_ListSessionsByChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	sessA := seedGuestSession(t, pool, fx.tenant, fx.channel, fmt.Sprintf("hash-%s", uuid.New()), true, time.Now().UTC().Add(24*time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessA) })
	sessB := seedGuestSession(t, pool, fx.tenant, fx.channel, fmt.Sprintf("hash-%s", uuid.New()), true, time.Now().UTC().Add(24*time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessB) })

	sessions, err := repo.ListSessionsByChannel(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("ListSessionsByChannel: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListSessionsByChannel: expected 2 sessions, got %d", len(sessions))
	}

	empty, err := repo.ListSessionsByChannel(ctxOwn, uuid.New())
	if err != nil {
		t.Fatalf("ListSessionsByChannel (empty channel): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("ListSessionsByChannel (empty channel): expected no sessions, got %d", len(empty))
	}
}

// TestPostgresRepository_ReadPaths_WithRealIPAddress covers GetSessionByTokenHash,
// GetSessionByID and ListSessionsByChannel against a session that carries a
// real (non-NULL) ip_address. The column is INET; pgx cannot scan it into a
// bare *string without a cast, so this bug is invisible unless at least one
// seeded row actually has an IP set (the tests above never set one).
func TestPostgresRepository_ReadPaths_WithRealIPAddress(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	const ip = "203.0.113.5"
	tokenHash := fmt.Sprintf("hash-%s", uuid.New())
	sessID := seedGuestSessionWithIP(t, pool, fx.tenant, fx.channel, tokenHash, ip)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", sessID) })

	byToken, err := repo.GetSessionByTokenHash(ctxOwn, tokenHash)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if byToken.IPAddress == nil || *byToken.IPAddress != ip {
		t.Fatalf("GetSessionByTokenHash: expected ip_address %q, got %v", ip, byToken.IPAddress)
	}

	byID, err := repo.GetSessionByID(ctxOwn, sessID)
	if err != nil {
		t.Fatalf("GetSessionByID: %v", err)
	}
	if byID.IPAddress == nil || *byID.IPAddress != ip {
		t.Fatalf("GetSessionByID: expected ip_address %q, got %v", ip, byID.IPAddress)
	}

	list, err := repo.ListSessionsByChannel(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("ListSessionsByChannel: %v", err)
	}
	if len(list) != 1 || list[0].IPAddress == nil || *list[0].IPAddress != ip {
		t.Fatalf("ListSessionsByChannel: expected 1 session with ip_address %q, got %+v", ip, list)
	}
}

func TestPostgresRepository_CleanupExpiredSessions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo := NewPostgresRepository(pool)

	now := time.Now().UTC()
	expired := seedGuestSession(t, pool, fx.tenant, fx.channel, fmt.Sprintf("hash-%s", uuid.New()), true, now.Add(-time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", expired) })
	stillValid := seedGuestSession(t, pool, fx.tenant, fx.channel, fmt.Sprintf("hash-%s", uuid.New()), true, now.Add(time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_sessions", stillValid) })

	n, err := repo.CleanupExpiredSessions(ctxOwn)
	if err != nil {
		t.Fatalf("CleanupExpiredSessions: %v", err)
	}
	if n < 1 {
		t.Fatalf("CleanupExpiredSessions: expected at least 1 row affected, got %d", n)
	}

	var expiredActive, stillValidActive bool
	if err := pool.QueryRow(sysCtx, "SELECT is_active FROM guest_sessions WHERE id = $1", expired).Scan(&expiredActive); err != nil {
		t.Fatalf("read back expired session: %v", err)
	}
	if expiredActive {
		t.Fatalf("CleanupExpiredSessions: expected the expired session to be deactivated")
	}
	if err := pool.QueryRow(sysCtx, "SELECT is_active FROM guest_sessions WHERE id = $1", stillValid).Scan(&stillValidActive); err != nil {
		t.Fatalf("read back still-valid session: %v", err)
	}
	if !stillValidActive {
		t.Fatalf("CleanupExpiredSessions: expected the non-expired session to remain active")
	}
}

func TestPostgresRepository_GetConfigByChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newGuestReadFixture(t, pool, true)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	now := time.Now().UTC()
	cfgID := testutil.SeedRow(t, pool, "guest_channel_config", map[string]any{
		"tenant_id":          fx.tenant,
		"channel_id":         fx.channel,
		"welcome_message":    "Willkommen im Support-Chat!",
		"primary_color":      "#3b82f6",
		"token_expiry_hours": 168,
		"max_file_size_mb":   10,
		"allowed_file_types": "image/png",
		"is_active":          true,
		"created_by":         fx.user,
		"created_at":         now,
		"updated_at":         now,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "guest_channel_config", cfgID) })

	got, err := repo.GetConfigByChannel(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("GetConfigByChannel: %v", err)
	}
	if got.WelcomeMessage != "Willkommen im Support-Chat!" {
		t.Fatalf("GetConfigByChannel: expected welcome message to match, got %q", got.WelcomeMessage)
	}

	if _, err := repo.GetConfigByChannel(ctxOwn, uuid.New()); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("GetConfigByChannel (no config): expected ErrConfigNotFound, got %v", err)
	}
}
