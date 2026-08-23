package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/security/vendoraccess"
	"github.com/kmuhub/kmuhub/internal/testutil"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// Vendor access is the one legitimate remote-access channel in the "ein
// Server pro Kunde" delivery model (feat-vendor-access-audit-trail). Every
// status change on a request has to append a real audit_log row -- the
// server-side slog.Info the service already emits proves nothing to a
// customer asking "who approved this and when", since it carries none of the
// hash chain, append-only guarantee or export path audit_log has.
//
// Same fixture shape as rbac_audit_events_db_test.go: a fixed, idempotently
// seeded tenant/actor pair reused across runs (audit_log rows are
// append-only and never cleaned up), so assertions are row-count deltas, not
// absolute counts.
var (
	vendorAccessAuditTenant = uuid.MustParse("a0d17000-0000-4000-8000-000000000002")
	vendorAccessAuditActor  = uuid.MustParse("a0d17000-0000-4000-8000-0000000000a2")
)

func vendorAccessAuditSetup(t *testing.T) (*pgxpool.Pool, *SecurityGRPCServer, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, vendorAccessAuditTenant, "VendorAccessAuditTenant")

	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'vendor-access-audit-actor@audit-events.test.local', 'x', 'Vendor', 'Actor')
		 ON CONFLICT (id) DO NOTHING`,
		vendorAccessAuditActor, vendorAccessAuditTenant)
	require.NoError(t, err)

	auditSvc := audit.NewService(audit.NewPostgresRepository(pool))
	vendorAccessSvc := vendoraccess.NewService(vendoraccess.NewPostgresRepository(pool))
	srv := NewSecurityGRPCServer(auditSvc, nil, nil, nil, vendorAccessSvc, pool)

	return pool, srv, vendorAccessAuditActor
}

// vendorAccessActorCtx carries both tenant and user id, matching what
// TenantInboundUnaryInterceptor injects from gRPC metadata in production
// (gateway -> SecurityGRPCServer is a real network hop, not an in-process
// call -- see middleware/grpc_tenant.go).
func vendorAccessActorCtx(actor uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), vendorAccessAuditTenant)
	return context.WithValue(ctx, middleware.UserIDKey, actor.String())
}

func vendorAccessAuditEventCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM audit_log WHERE tenant_id = $1`, vendorAccessAuditTenant,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

func latestVendorAccessAuditEvent(t *testing.T, pool *pgxpool.Pool) (action, target, targetType string, userID uuid.UUID) {
	t.Helper()
	var uid *uuid.UUID
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT action, target, target_type, user_id FROM audit_log
		 WHERE tenant_id = $1 ORDER BY sequence_num DESC LIMIT 1`, vendorAccessAuditTenant,
	).Scan(&action, &target, &targetType, &uid)
	require.NoError(t, err)
	require.NotNil(t, uid, "vendor-access events always carry an actor")
	return action, target, targetType, *uid
}

// seedVendorAccessRequest inserts a request directly via the repository (not
// through Service.Create, which is unreachable from any HTTP route) so each
// subtest can start from the exact status its transition needs.
func seedVendorAccessRequest(t *testing.T, pool *pgxpool.Pool, status string, scope []string) uuid.UUID {
	t.Helper()
	repo := vendoraccess.NewPostgresRepository(pool)
	now := time.Now().UTC()
	req := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       vendorAccessAuditTenant,
		Reason:         "audit trail test",
		Description:    "seeded by vendor_access_audit_events_db_test.go",
		Agents:         []models.VendorAccessAgent{{Name: "Test Agent"}},
		Scope:          scope,
		RequestedStart: now,
		DurationDays:   7,
		ExpiresAt:      now.Add(7 * 24 * time.Hour),
		Status:         status,
		CreatedAt:      now,
	}
	err := repo.CreateRequest(testutil.WithTenantCtx(context.Background(), vendorAccessAuditTenant), req)
	require.NoError(t, err)
	return req.ID
}

func TestVendorAccessAuditEvents_DB(t *testing.T) {
	pool, srv, actor := vendorAccessAuditSetup(t)
	ctx := vendorAccessActorCtx(actor)

	t.Run("approve_writes_one_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusPending, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
			RequestId: id.String(), ActorId: actor.String(),
		})
		require.NoError(t, err)

		require.Equal(t, before+1, vendorAccessAuditEventCount(t, pool), "exactly one event for the approval")
		action, target, targetType, userID := latestVendorAccessAuditEvent(t, pool)
		require.Equal(t, "vendor_access.approve", action)
		require.Equal(t, id.String(), target)
		require.Equal(t, "vendor_access_request", targetType)
		require.Equal(t, actor, userID)
	})

	t.Run("approve_rejected_sensitive_without_ack_writes_no_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusPending, []string{"hr_data"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
			RequestId: id.String(), ActorId: actor.String(), SensitiveAck: false,
		})
		require.Error(t, err, "a sensitive scope without ack must be rejected (422)")

		require.Equal(t, before, vendorAccessAuditEventCount(t, pool), "a rejected approval must not append an audit event")
	})

	t.Run("decline_writes_one_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusPending, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.DeclineVendorAccessRequest(ctx, &securityv1.DeclineVendorAccessRequestRequest{
			RequestId: id.String(),
		})
		require.NoError(t, err)

		require.Equal(t, before+1, vendorAccessAuditEventCount(t, pool), "exactly one event for the decline")
		action, target, targetType, userID := latestVendorAccessAuditEvent(t, pool)
		require.Equal(t, "vendor_access.decline", action)
		require.Equal(t, id.String(), target)
		require.Equal(t, "vendor_access_request", targetType)
		require.Equal(t, actor, userID, "the actor comes from the gRPC-propagated caller context, not a request field")
	})

	t.Run("decline_rejected_active_writes_no_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusActive, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.DeclineVendorAccessRequest(ctx, &securityv1.DeclineVendorAccessRequestRequest{
			RequestId: id.String(),
		})
		require.Error(t, err, "an already-active request cannot be declined")

		require.Equal(t, before, vendorAccessAuditEventCount(t, pool), "a rejected decline must not append an audit event")
	})

	t.Run("counter_propose_writes_one_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusPending, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.CounterProposeVendorAccessRequest(ctx, &securityv1.CounterProposeVendorAccessRequestRequest{
			RequestId: id.String(), ProposedStart: time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02"),
		})
		require.NoError(t, err)

		require.Equal(t, before+1, vendorAccessAuditEventCount(t, pool), "exactly one event for the counter-proposal")
		action, target, targetType, userID := latestVendorAccessAuditEvent(t, pool)
		require.Equal(t, "vendor_access.counter_propose", action)
		require.Equal(t, id.String(), target)
		require.Equal(t, "vendor_access_request", targetType)
		require.Equal(t, actor, userID)
	})

	t.Run("counter_propose_rejected_already_counter_proposed_writes_no_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusCounterProposed, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.CounterProposeVendorAccessRequest(ctx, &securityv1.CounterProposeVendorAccessRequestRequest{
			RequestId: id.String(), ProposedStart: time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02"),
		})
		require.Error(t, err, "only a single counter-proposal round trip is allowed")

		require.Equal(t, before, vendorAccessAuditEventCount(t, pool), "a rejected counter-proposal must not append an audit event")
	})

	t.Run("revoke_writes_one_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusActive, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.RevokeVendorAccessRequest(ctx, &securityv1.RevokeVendorAccessRequestRequest{
			RequestId: id.String(), ActorId: actor.String(),
		})
		require.NoError(t, err)

		require.Equal(t, before+1, vendorAccessAuditEventCount(t, pool), "exactly one event for the revoke")
		action, target, targetType, userID := latestVendorAccessAuditEvent(t, pool)
		require.Equal(t, "vendor_access.revoke", action)
		require.Equal(t, id.String(), target)
		require.Equal(t, "vendor_access_request", targetType)
		require.Equal(t, actor, userID)
	})

	t.Run("revoke_rejected_pending_writes_no_event", func(t *testing.T) {
		id := seedVendorAccessRequest(t, pool, models.VendorAccessStatusPending, []string{"crm"})
		before := vendorAccessAuditEventCount(t, pool)

		_, err := srv.RevokeVendorAccessRequest(ctx, &securityv1.RevokeVendorAccessRequestRequest{
			RequestId: id.String(), ActorId: actor.String(),
		})
		require.Error(t, err, "a pending request cannot be revoked, only an active one")

		require.Equal(t, before, vendorAccessAuditEventCount(t, pool), "a rejected revoke must not append an audit event")
	})
}
