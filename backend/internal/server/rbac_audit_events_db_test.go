package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/testutil"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// Every RBAC write path (p1b-guardrails) has to append exactly one row to the
// real audit_log — the hash chain, the append-only trigger (migration 000222)
// and the RLS write policy are database properties a mock repository cannot
// stand in for. Own tenant, never testutil.TenantA/B: audit_log rows cannot be
// deleted (the append-only trigger blocks it), so every assertion here is a
// row-count delta rather than an absolute count — that also makes the tenant
// safe to reuse across repeated local runs.
//
// auditEventsActor is a fixed id, seeded idempotently and never cleaned up —
// unlike the target user below, it ends up as audit_log.user_id, and
// audit_log_user_id_fkey then blocks deleting it for good. Reusing one row
// across runs avoids piling up orphaned users the append-only trigger leaves
// no way to remove.
var (
	auditEventsTenant = uuid.MustParse("a0d17000-0000-4000-8000-000000000001")
	auditEventsActor  = uuid.MustParse("a0d17000-0000-4000-8000-0000000000a1")
)

func auditEventsSetup(t *testing.T) (*pgxpool.Pool, *AuthGRPCServer, uuid.UUID, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, auditEventsTenant, "AuditEventsTenant")

	authSvc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	auditSvc := audit.NewService(audit.NewPostgresRepository(pool))
	srv := NewAuthGRPCServer(authSvc, auditSvc)

	sysCtx := testutil.WithSystemCtx(context.Background())

	actor := auditEventsActor
	_, err := pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'audit-events-actor@audit-events.test.local', 'x', 'Audit', 'Actor')
		 ON CONFLICT (id) DO NOTHING`,
		actor, auditEventsTenant)
	require.NoError(t, err)

	_, err = pool.Exec(sysCtx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, id FROM roles WHERE name = 'admin' AND tenant_id IS NULL
		 ON CONFLICT DO NOTHING`, actor)
	require.NoError(t, err, "admin preset missing — migration 000256 not applied?")

	target := uuid.New()
	_, err = pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Audit', 'Target')`,
		target, auditEventsTenant, target.String()+"@audit-events.test.local")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", target) })

	return pool, srv, actor, target
}

func auditActorCtx(actor uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), auditEventsTenant)
	return context.WithValue(ctx, middleware.UserIDKey, actor.String())
}

// auditEventCount returns how many audit_log rows exist for the tenant right
// now — a delta baseline, not an absolute expectation, since the append-only
// trigger means old runs' rows never go away.
func auditEventCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM audit_log WHERE tenant_id = $1`, auditEventsTenant,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

// latestAuditEvent reads back the newest row for the tenant, to be called
// right after a single write is known to have happened (guarded by a count
// delta of exactly 1 at the call site).
func latestAuditEvent(t *testing.T, pool *pgxpool.Pool) (action, target, targetType string, userID uuid.UUID) {
	t.Helper()
	var uid *uuid.UUID
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT action, target, target_type, user_id FROM audit_log
		 WHERE tenant_id = $1 ORDER BY sequence_num DESC LIMIT 1`, auditEventsTenant,
	).Scan(&action, &target, &targetType, &uid)
	require.NoError(t, err)
	require.NotNil(t, uid, "permission-change events always carry an actor")
	return action, target, targetType, *uid
}

// TestPermissionAuditEvents_DB walks one custom role through create, update,
// assign, revoke and delete — the five change kinds PHASE-1-RBAC-PLAN §4
// requires an audit trail for — and checks that each call appends exactly one
// audit_log row naming the tenant, the actor and the changed resource.
func TestPermissionAuditEvents_DB(t *testing.T) {
	pool, srv, actor, target := auditEventsSetup(t)
	ctx := auditActorCtx(actor)

	adminPresetID := ""
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id::text FROM roles WHERE name = 'admin' AND tenant_id IS NULL`,
	).Scan(&adminPresetID))

	t.Run("role_created", func(t *testing.T) {
		before := auditEventCount(t, pool)

		resp, err := srv.CreateRole(ctx, &authv1.CreateRoleRequest{
			Name: "Audit Events Role", BasedOn: adminPresetID,
		})
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "roles", uuid.MustParse(resp.Role.Id)) })

		require.Equal(t, before+1, auditEventCount(t, pool), "exactly one event for the create")
		action, evTarget, targetType, userID := latestAuditEvent(t, pool)
		require.Equal(t, "permission.role_created", action)
		require.Equal(t, resp.Role.Id, evTarget)
		require.Equal(t, "role", targetType)
		require.Equal(t, actor, userID)

		roleID := resp.Role.Id

		t.Run("role_updated", func(t *testing.T) {
			before := auditEventCount(t, pool)
			newName := "Audit Events Role (renamed)"

			_, err := srv.UpdateRole(ctx, &authv1.UpdateRoleRequest{RoleId: roleID, Name: &newName})
			require.NoError(t, err)

			require.Equal(t, before+1, auditEventCount(t, pool), "exactly one event for the update")
			action, evTarget, targetType, userID := latestAuditEvent(t, pool)
			require.Equal(t, "permission.role_updated", action)
			require.Equal(t, roleID, evTarget)
			require.Equal(t, "role", targetType)
			require.Equal(t, actor, userID)
		})

		t.Run("assigned", func(t *testing.T) {
			before := auditEventCount(t, pool)

			_, err := srv.AssignUserRole(ctx, &authv1.AssignUserRoleRequest{
				UserId: target.String(), RoleId: roleID,
			})
			require.NoError(t, err)

			require.Equal(t, before+1, auditEventCount(t, pool), "exactly one event for the assignment")
			action, evTarget, targetType, userID := latestAuditEvent(t, pool)
			require.Equal(t, "permission.assigned", action)
			require.Equal(t, target.String(), evTarget)
			require.Equal(t, "user", targetType)
			require.Equal(t, actor, userID, "the actor is the admin who assigned it, not the recipient")
		})

		t.Run("revoked", func(t *testing.T) {
			before := auditEventCount(t, pool)

			_, err := srv.RevokeUserRole(ctx, &authv1.RevokeUserRoleRequest{
				UserId: target.String(), RoleId: roleID,
			})
			require.NoError(t, err)

			require.Equal(t, before+1, auditEventCount(t, pool), "exactly one event for the revoke")
			action, evTarget, targetType, userID := latestAuditEvent(t, pool)
			require.Equal(t, "permission.revoked", action)
			require.Equal(t, target.String(), evTarget)
			require.Equal(t, "user", targetType)
			require.Equal(t, actor, userID)
		})

		t.Run("role_deleted", func(t *testing.T) {
			before := auditEventCount(t, pool)

			_, err := srv.DeleteRole(ctx, &authv1.DeleteRoleRequest{RoleId: roleID})
			require.NoError(t, err)

			require.Equal(t, before+1, auditEventCount(t, pool), "exactly one event for the delete")
			action, evTarget, targetType, userID := latestAuditEvent(t, pool)
			require.Equal(t, "permission.role_deleted", action)
			require.Equal(t, roleID, evTarget)
			require.Equal(t, "role", targetType)
			require.Equal(t, actor, userID)
		})
	})
}

// TestPermissionAuditEvents_DB_RejectedWriteLeavesNoEvent pins the ordering
// the guardrails depend on: they run inside the service, before this RPC ever
// reaches logPermissionEvent, so a call the guardrails refuse must not grow
// the audit trail at all. DeleteRole on a role that still has a member is the
// cheapest rejection to construct — role_has_members fires before any DELETE.
func TestPermissionAuditEvents_DB_RejectedWriteLeavesNoEvent(t *testing.T) {
	pool, srv, actor, target := auditEventsSetup(t)
	ctx := auditActorCtx(actor)

	var adminPresetID string
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id::text FROM roles WHERE name = 'admin' AND tenant_id IS NULL`,
	).Scan(&adminPresetID))

	resp, err := srv.CreateRole(ctx, &authv1.CreateRoleRequest{
		Name: "Audit Events Guarded Role", BasedOn: adminPresetID,
	})
	require.NoError(t, err)
	roleID := uuid.MustParse(resp.Role.Id)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "roles", roleID) })

	_, err = srv.AssignUserRole(ctx, &authv1.AssignUserRoleRequest{
		UserId: target.String(), RoleId: resp.Role.Id,
	})
	require.NoError(t, err)

	before := auditEventCount(t, pool)

	_, err = srv.DeleteRole(ctx, &authv1.DeleteRoleRequest{RoleId: resp.Role.Id})
	require.Error(t, err, "a role with a member must not delete")

	require.Equal(t, before, auditEventCount(t, pool), "a rejected write must not append an audit event")
}

// TestUserOverrideAuditEvents_DB is the R-6 half of the trail. The override
// editor writes a whole map at once, so the interesting property is not "one
// call, one event" but "one event per key that actually changed": re-saving an
// unchanged map is a legitimate no-op, and filling the log with events that
// describe nothing would make the trail useless exactly when it matters.
func TestUserOverrideAuditEvents_DB(t *testing.T) {
	pool, srv, actor, target := auditEventsSetup(t)
	ctx := auditActorCtx(actor)

	t.Run("two_new_keys_write_two_events", func(t *testing.T) {
		before := auditEventCount(t, pool)

		_, err := srv.SetUserOverrides(ctx, &authv1.SetUserOverridesRequest{
			UserId: target.String(),
			Overrides: []*authv1.CapabilityOverride{
				{Key: "work:project:edit", Mode: "allow", Scope: "all"},
				{Key: "rapporte:report:create", Mode: "deny", Scope: "all"},
			},
		})
		require.NoError(t, err)

		require.Equal(t, before+2, auditEventCount(t, pool), "one event per changed key")
		action, evTarget, targetType, userID := latestAuditEvent(t, pool)
		require.Equal(t, "permission.override_set", action)
		require.Equal(t, target.String(), evTarget)
		require.Equal(t, "user", targetType)
		require.Equal(t, actor, userID)
	})

	t.Run("resaving_the_same_map_writes_nothing", func(t *testing.T) {
		before := auditEventCount(t, pool)

		_, err := srv.SetUserOverrides(ctx, &authv1.SetUserOverridesRequest{
			UserId: target.String(),
			Overrides: []*authv1.CapabilityOverride{
				{Key: "work:project:edit", Mode: "allow", Scope: "all"},
				{Key: "rapporte:report:create", Mode: "deny", Scope: "all"},
			},
		})
		require.NoError(t, err)

		require.Equal(t, before, auditEventCount(t, pool), "nothing changed, so nothing is audited")
	})

	t.Run("clearing_audits_every_dropped_key", func(t *testing.T) {
		before := auditEventCount(t, pool)

		_, err := srv.ClearUserOverrides(ctx, &authv1.ClearUserOverridesRequest{
			UserId: target.String(),
		})
		require.NoError(t, err)

		require.Equal(t, before+2, auditEventCount(t, pool), "both overrides disappeared, both are audited")
		action, _, _, _ := latestAuditEvent(t, pool)
		require.Equal(t, "permission.override_removed", action)
	})

	t.Run("rejected_write_leaves_no_event", func(t *testing.T) {
		before := auditEventCount(t, pool)

		// The self-edit block fires inside the service, before the RPC ever
		// reaches logOverrideChanges.
		_, err := srv.SetUserOverrides(ctx, &authv1.SetUserOverridesRequest{
			UserId: actor.String(),
			Overrides: []*authv1.CapabilityOverride{
				{Key: "work:project:edit", Mode: "allow", Scope: "all"},
			},
		})
		require.Error(t, err)

		require.Equal(t, before, auditEventCount(t, pool), "a refused write must not append an audit event")
	})
}
