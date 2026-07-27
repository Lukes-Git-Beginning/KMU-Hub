package channel

// Writes into channels and channel_memberships must land in — and stay
// visible only to — the caller's tenant. Both tables carry tenant_id NOT
// NULL since migration 000106/000112, but channels itself had no RLS policy
// until migration 000253: every sibling chat table (channel_memberships,
// message_mentions, chat_files, guest_sessions, ...) was RLS-enabled in the
// mig 000122 sweep, channels was silently dropped from that list. The
// service layer scopes reads correctly via GetByIDForTenant, so the gap was
// invisible from the application — a raw query or a future missed WHERE
// clause had no database-level backstop. This test would have gone red
// against the pre-000253 schema (a foreign tenant's context could read a
// freshly created channel straight back).
//
// channel_memberships has a composite primary key (channel_id, user_id), no
// `id` column, so testutil.AssertRowCount (keys on `id`) does not apply —
// assertRowCountWhere below generalizes the pattern already used in
// internal/settings/tenant_write_test.go and internal/auth/rls_provisioning_test.go
// (assertModuleCount) to an arbitrary predicate.

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

func TestChannelWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Channel Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Channel Write Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("channel-write-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("channel-write-b-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// Create
	ch := &models.Channel{
		ID: uuid.New(), TenantID: tenantOwn, Name: "general", CreatedBy: userA,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, ch); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "channels", ch.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "channels", ch.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "channels", ch.ID, 0)

	// Update
	ch.Name = "general-renamed"
	ch.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOwn, ch); err != nil {
		t.Fatalf("Update: %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOther, "channels", ch.ID, 0)

	// AddMember
	mem := &models.ChannelMembership{ChannelID: ch.ID, UserID: userA, TenantID: tenantOwn, Role: models.ChannelRoleOwner, JoinedAt: now}
	if err := repo.AddMember(ctxOwn, mem); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	assertRowCountWhere(t, pool, ctxOwn, "channel_memberships", "channel_id = $1 AND user_id = $2", 1, ch.ID, userA)
	assertRowCountWhere(t, pool, ctxOther, "channel_memberships", "channel_id = $1 AND user_id = $2", 0, ch.ID, userA)

	// UpdateMembership
	mem.Role = models.ChannelRoleAdmin
	if err := repo.UpdateMembership(ctxOwn, mem); err != nil {
		t.Fatalf("UpdateMembership: %v", err)
	}

	// UpdateLastRead
	if err := repo.UpdateLastRead(ctxOwn, ch.ID, userA, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateLastRead: %v", err)
	}

	// RemoveMember
	if err := repo.RemoveMember(ctxOwn, ch.ID, userA); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	assertRowCountWhere(t, pool, sysCtx, "channel_memberships", "channel_id = $1 AND user_id = $2", 0, ch.ID, userA)

	// CreateDMChannel — transactional insert of one channel + two memberships.
	dmCh := &models.Channel{
		ID: uuid.New(), TenantID: tenantOwn, Name: "dm", IsDM: true, CreatedBy: userA,
		DMUser1: &userA, DMUser2: &userB, CreatedAt: now, UpdatedAt: now,
	}
	dmMem1 := &models.ChannelMembership{ChannelID: dmCh.ID, UserID: userA, TenantID: tenantOwn, Role: models.ChannelRoleMember, JoinedAt: now}
	dmMem2 := &models.ChannelMembership{ChannelID: dmCh.ID, UserID: userB, TenantID: tenantOwn, Role: models.ChannelRoleMember, JoinedAt: now}
	if err := repo.CreateDMChannel(ctxOwn, dmCh, dmMem1, dmMem2); err != nil {
		t.Fatalf("CreateDMChannel: %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "channels", dmCh.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "channels", dmCh.ID, 0)
	assertRowCountWhere(t, pool, ctxOwn, "channel_memberships", "channel_id = $1", 2, dmCh.ID)
	assertRowCountWhere(t, pool, ctxOther, "channel_memberships", "channel_id = $1", 0, dmCh.ID)

	// Delete — channel_memberships cascade via ON DELETE CASCADE.
	if err := repo.Delete(ctxOwn, dmCh.ID, tenantOwn); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "channels", dmCh.ID, 0)
	assertRowCountWhere(t, pool, sysCtx, "channel_memberships", "channel_id = $1", 0, dmCh.ID)
}

// assertRowCountWhere runs `SELECT count(*) FROM <table> WHERE <where>` under
// ctx (which determines RLS visibility) and fails if the count differs.
// Generalizes testutil.AssertRowCount (keys on `id`) for tables with a
// composite primary key, such as channel_memberships.
func assertRowCountWhere(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, where string, expected int, args ...any) {
	t.Helper()
	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	if count != expected {
		t.Fatalf("table %s where %s: expected %d row(s), got %d", table, where, expected, count)
	}
}
