package team

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// WithManageAll waives the per-inbox admin check, not the tenant boundary: a
// manage-all caller of tenant B must not reach a team inbox of tenant A, and a
// manage-all caller of tenant A reaches it without being a member.
func TestManageAll_StopsAtTheTenantBoundary(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Manage-All Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Manage-All Other Tenant")

	seedUser := func(tenant uuid.UUID, tag string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         fmt.Sprintf("manage-all-%s-%s@test.local", tag, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	owner, tenantAdmin, foreignAdmin, newbie := seedUser(tenantOwn, "owner"), seedUser(tenantOwn, "ta"), seedUser(tenantOther, "fa"), seedUser(tenantOwn, "new")

	svc := NewService(NewPostgresRepository(pool), nil)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	inbox := &models.TeamInbox{ID: uuid.New(), TenantID: tenantOwn, Name: "ManageAll " + uuid.New().String()[:6], CreatedBy: owner}
	if err := svc.CreateTeamInbox(ctxOwn, inbox); err != nil {
		t.Fatalf("CreateTeamInbox: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "team_inboxes", inbox.ID) })

	member := func(u uuid.UUID) *models.TeamInboxMember {
		return &models.TeamInboxMember{TeamInboxID: inbox.ID, UserID: u, Role: "member"}
	}

	// Without the permission, a non-member of the own tenant is refused.
	if err := svc.AddMember(ctxOwn, tenantOwn, tenantAdmin, member(newbie)); err != ErrNotTeamAdmin {
		t.Fatalf("AddMember without manage-all: expected ErrNotTeamAdmin, got %v", err)
	}
	// With it, the same call lands.
	if err := svc.AddMember(WithManageAll(ctxOwn), tenantOwn, tenantAdmin, member(newbie)); err != nil {
		t.Fatalf("AddMember with manage-all: %v", err)
	}
	if ok, _ := NewPostgresRepository(pool).IsMember(ctxOwn, inbox.ID, newbie); !ok {
		t.Fatalf("member added by a manage-all caller is missing")
	}
	// A manage-all caller of another tenant gets NotFound and changes nothing.
	if err := svc.RemoveMember(WithManageAll(ctxOther), tenantOther, inbox.ID, foreignAdmin, newbie); err != ErrTeamInboxNotFound {
		t.Fatalf("RemoveMember across tenants: expected ErrTeamInboxNotFound, got %v", err)
	}
	if err := svc.AddMember(WithManageAll(ctxOther), tenantOther, foreignAdmin, member(foreignAdmin)); err != ErrTeamInboxNotFound {
		t.Fatalf("AddMember across tenants: expected ErrTeamInboxNotFound, got %v", err)
	}
	if ok, _ := NewPostgresRepository(pool).IsMember(ctxOwn, inbox.ID, newbie); !ok {
		t.Fatalf("a cross-tenant manage-all call removed a member")
	}
}
