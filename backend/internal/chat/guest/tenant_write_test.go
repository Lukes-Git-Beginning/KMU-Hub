package guest

// Writes into guest_sessions and guest_channel_config must land in — and
// stay visible only to — the caller's tenant. Both tables carry tenant_id
// NOT NULL since migration 000115 and were already RLS-enabled in the
// mig 000122 sweep (unlike channels/messages, which were dropped from that
// sweep — see internal/chat/channel/tenant_write_test.go). The mock-backed
// tests in tenant_isolation_test.go never touch a real database; this test
// runs against PostgresRepository directly and additionally verifies the
// RLS backstop on the UPDATE/DELETE paths: a write issued from a foreign
// tenant context must be a silent no-op, not just invisible to a later read.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGuestSessionWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Guest Session Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Guest Session Write Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("guest-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "guest-write-test",
		"created_by": userA,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// CreateSession
	sess := &GuestSession{
		ID: uuid.New(), TenantID: tenantOwn, TokenHash: fmt.Sprintf("hash-%s", uuid.New()),
		ChannelID: channelID, DisplayName: "Guest", LastActivityAt: now,
		IsActive: true, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := repo.CreateSession(ctxOwn, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "guest_sessions", sess.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "guest_sessions", sess.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "guest_sessions", sess.ID, 0)

	// Baseline read of the stored value — timestamptz round-trips at
	// microsecond precision, so comparing against the in-memory `now`
	// (nanosecond precision) would spuriously differ even with no write at all.
	var baseline time.Time
	if err := pool.QueryRow(sysCtx, "SELECT last_activity_at FROM guest_sessions WHERE id = $1", sess.ID).Scan(&baseline); err != nil {
		t.Fatalf("read back baseline last_activity_at: %v", err)
	}

	// UpdateLastActivity from a foreign tenant must be a no-op.
	if err := repo.UpdateLastActivity(ctxOther, sess.ID); err != nil {
		t.Fatalf("UpdateLastActivity (foreign tenant): %v", err)
	}
	var lastActivity time.Time
	if err := pool.QueryRow(sysCtx, "SELECT last_activity_at FROM guest_sessions WHERE id = $1", sess.ID).Scan(&lastActivity); err != nil {
		t.Fatalf("read back last_activity_at: %v", err)
	}
	if !lastActivity.Equal(baseline) {
		t.Fatalf("UpdateLastActivity: foreign tenant context updated a session it cannot see")
	}

	// UpdateLastActivity from the owning tenant does update.
	if err := repo.UpdateLastActivity(ctxOwn, sess.ID); err != nil {
		t.Fatalf("UpdateLastActivity: %v", err)
	}
	if err := pool.QueryRow(sysCtx, "SELECT last_activity_at FROM guest_sessions WHERE id = $1", sess.ID).Scan(&lastActivity); err != nil {
		t.Fatalf("read back last_activity_at: %v", err)
	}
	if !lastActivity.After(baseline) {
		t.Fatalf("UpdateLastActivity: expected last_activity_at to advance")
	}

	// DeactivateSession from a foreign tenant must be a no-op.
	if err := repo.DeactivateSession(ctxOther, sess.ID); err != nil {
		t.Fatalf("DeactivateSession (foreign tenant): %v", err)
	}
	var isActive bool
	if err := pool.QueryRow(sysCtx, "SELECT is_active FROM guest_sessions WHERE id = $1", sess.ID).Scan(&isActive); err != nil {
		t.Fatalf("read back is_active: %v", err)
	}
	if !isActive {
		t.Fatalf("DeactivateSession: foreign tenant context deactivated a session it cannot see")
	}

	// DeactivateSession from the owning tenant does deactivate.
	if err := repo.DeactivateSession(ctxOwn, sess.ID); err != nil {
		t.Fatalf("DeactivateSession: %v", err)
	}
	if err := pool.QueryRow(sysCtx, "SELECT is_active FROM guest_sessions WHERE id = $1", sess.ID).Scan(&isActive); err != nil {
		t.Fatalf("read back is_active: %v", err)
	}
	if isActive {
		t.Fatalf("DeactivateSession: expected is_active = false, got true")
	}
}

func TestGuestChannelConfigWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Guest Config Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Guest Config Write Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("guest-config-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "guest-config-write-test",
		"created_by": userA,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// CreateConfig
	cfg := &GuestChannelConfig{
		ID: uuid.New(), TenantID: tenantOwn, ChannelID: channelID,
		WelcomeMessage: "Willkommen!", PrimaryColor: "#3b82f6",
		TokenExpiryHours: 168, MaxFileSizeMB: 10,
		AllowedFileTypes: "image/png", IsActive: true,
		CreatedBy: userA, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateConfig(ctxOwn, cfg); err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "guest_channel_config", cfg.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "guest_channel_config", cfg.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "guest_channel_config", cfg.ID, 0)

	// UpdateConfig from a foreign tenant must be a no-op.
	foreignUpdate := &GuestChannelConfig{
		ChannelID: channelID, WelcomeMessage: "Hijacked", PrimaryColor: "#000000",
		TokenExpiryHours: 1, MaxFileSizeMB: 1, AllowedFileTypes: "text/plain",
		IsActive: false, UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpdateConfig(ctxOther, foreignUpdate); err != nil {
		t.Fatalf("UpdateConfig (foreign tenant): %v", err)
	}
	var welcome string
	if err := pool.QueryRow(sysCtx, "SELECT welcome_message FROM guest_channel_config WHERE id = $1", cfg.ID).Scan(&welcome); err != nil {
		t.Fatalf("read back welcome_message: %v", err)
	}
	if welcome != "Willkommen!" {
		t.Fatalf("UpdateConfig: foreign tenant context updated a config it cannot see")
	}

	// UpdateConfig from the owning tenant does update.
	ownUpdate := &GuestChannelConfig{
		ChannelID: channelID, WelcomeMessage: "Aktualisiert", PrimaryColor: "#111111",
		TokenExpiryHours: 24, MaxFileSizeMB: 5, AllowedFileTypes: "image/png,image/jpeg",
		IsActive: true, UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpdateConfig(ctxOwn, ownUpdate); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}
	if err := pool.QueryRow(sysCtx, "SELECT welcome_message FROM guest_channel_config WHERE id = $1", cfg.ID).Scan(&welcome); err != nil {
		t.Fatalf("read back welcome_message: %v", err)
	}
	if welcome != "Aktualisiert" {
		t.Fatalf("UpdateConfig: expected welcome_message to update, got %q", welcome)
	}

	// DeleteConfig from a foreign tenant must be a no-op.
	if err := repo.DeleteConfig(ctxOther, channelID); err != nil {
		t.Fatalf("DeleteConfig (foreign tenant): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "guest_channel_config", cfg.ID, 1)

	// DeleteConfig from the owning tenant does delete.
	if err := repo.DeleteConfig(ctxOwn, channelID); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "guest_channel_config", cfg.ID, 0)
}
