package team

// team_inboxes and routing_rules already have a raw-row cross-tenant proof
// (tenant_isolation_phase2_test.go), but PostgresRepository itself -- the
// thing the service layer actually calls -- had no DB test at all
// (e-cov-inbox-repo-infra). This file exercises every method against a real
// Postgres: writes from a foreign-tenant ctx must not land (either an
// explicit tenant_id predicate rejects them, or -- for team_inbox_members,
// which carries no repository-level tenant_id parameter at all -- RLS alone
// has to stop them), and the owning ctx must see them succeed.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTeamInboxWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Team Inbox Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Team Inbox Write Other Tenant")

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("team-inbox-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("team-inbox-write-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// CreateTeamInbox
	inbox := &models.TeamInbox{
		ID:             uuid.New(),
		TenantID:       tenantOwn,
		Name:           "Support " + uuid.New().String()[:6],
		AssignmentMode: "manual",
		Visibility:     "open",
		CreatedBy:      userOwn,
	}
	if err := repo.CreateTeamInbox(ctxOwn, inbox); err != nil {
		t.Fatalf("CreateTeamInbox: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "team_inboxes", inbox.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "team_inboxes", inbox.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "team_inboxes", inbox.ID, 0)

	// GetTeamInbox
	if _, err := repo.GetTeamInbox(ctxOther, tenantOwn, inbox.ID); err != ErrTeamInboxNotFound {
		t.Fatalf("GetTeamInbox (foreign ctx): expected ErrTeamInboxNotFound, got %v", err)
	}
	got, err := repo.GetTeamInbox(ctxOwn, tenantOwn, inbox.ID)
	if err != nil {
		t.Fatalf("GetTeamInbox (own ctx): %v", err)
	}
	if got.Name != inbox.Name {
		t.Fatalf("GetTeamInbox (own ctx): name mismatch, got %q", got.Name)
	}
	if _, err := repo.GetTeamInbox(ctxOwn, tenantOwn, uuid.New()); err != ErrTeamInboxNotFound {
		t.Fatalf("GetTeamInbox (unknown id): expected ErrTeamInboxNotFound, got %v", err)
	}

	// UpdateTeamInbox — foreign ctx must not land even though the caller
	// supplies the real owning tenantID on the struct; only RLS (session
	// tenant, not the WHERE clause literal) can be stopping it.
	updateAttempt := &models.TeamInbox{
		ID:             inbox.ID,
		TenantID:       tenantOwn,
		Name:           "Hacked Name",
		AssignmentMode: "manual",
		Visibility:     "open",
	}
	if err := repo.UpdateTeamInbox(ctxOther, updateAttempt); err != ErrTeamInboxNotFound {
		t.Fatalf("UpdateTeamInbox (foreign ctx): expected ErrTeamInboxNotFound, got %v", err)
	}
	got, _ = repo.GetTeamInbox(ctxOwn, tenantOwn, inbox.ID)
	if got.Name == "Hacked Name" {
		t.Fatalf("a foreign-tenant UpdateTeamInbox reached the row")
	}
	updateAttempt.Name = "Renamed Support"
	if err := repo.UpdateTeamInbox(ctxOwn, updateAttempt); err != nil {
		t.Fatalf("UpdateTeamInbox (own ctx): %v", err)
	}
	got, _ = repo.GetTeamInbox(ctxOwn, tenantOwn, inbox.ID)
	if got.Name != "Renamed Support" {
		t.Fatalf("own-tenant UpdateTeamInbox did not land: name=%q", got.Name)
	}
	unknownUpdate := &models.TeamInbox{ID: uuid.New(), TenantID: tenantOwn, Name: "Ghost", AssignmentMode: "manual", Visibility: "open"}
	if err := repo.UpdateTeamInbox(ctxOwn, unknownUpdate); err != ErrTeamInboxNotFound {
		t.Fatalf("UpdateTeamInbox (unknown id): expected ErrTeamInboxNotFound, got %v", err)
	}

	// ListByUser — add the owning user as a member first via AddMember below
	// happens after the member-block; re-checked there.

	// AddMember from a foreign ctx: the INSERT resolves tenant_id via a
	// sub-SELECT against team_inboxes, which RLS blocks for a foreign
	// session — the sub-SELECT returns no row, so tenant_id is NULL and the
	// NOT NULL constraint rejects the insert. Either way it must not land.
	if err := repo.AddMember(ctxOther, &models.TeamInboxMember{TeamInboxID: inbox.ID, UserID: userOther, Role: "member"}); err == nil {
		t.Fatalf("AddMember (foreign ctx): expected an error, got nil")
	}
	if isMember, _ := repo.IsMember(ctxOwn, inbox.ID, userOther); isMember {
		t.Fatalf("a foreign-tenant AddMember reached the row")
	}

	// AddMember from the owning ctx.
	if err := repo.AddMember(ctxOwn, &models.TeamInboxMember{TeamInboxID: inbox.ID, UserID: userOwn, Role: "admin"}); err != nil {
		t.Fatalf("AddMember (own ctx): %v", err)
	}
	if err := repo.AddMember(ctxOwn, &models.TeamInboxMember{TeamInboxID: inbox.ID, UserID: userOwn, Role: "admin"}); err != ErrAlreadyMember {
		t.Fatalf("AddMember (duplicate): expected ErrAlreadyMember, got %v", err)
	}

	// IsMember / GetMemberRole / ListMembers / GetMemberCount / CountAdmins —
	// none of these repository methods take a tenantID parameter at all, so
	// RLS on team_inbox_members is the *only* thing that can stop a foreign
	// ctx from seeing the row.
	if isMember, err := repo.IsMember(ctxOther, inbox.ID, userOwn); err != nil || isMember {
		t.Fatalf("IsMember (foreign ctx): expected false/nil, got %v/%v", isMember, err)
	}
	if isMember, err := repo.IsMember(ctxOwn, inbox.ID, userOwn); err != nil || !isMember {
		t.Fatalf("IsMember (own ctx): expected true/nil, got %v/%v", isMember, err)
	}
	if role, err := repo.GetMemberRole(ctxOther, inbox.ID, userOwn); err != nil || role != "" {
		t.Fatalf("GetMemberRole (foreign ctx): expected empty/nil, got %q/%v", role, err)
	}
	if role, err := repo.GetMemberRole(ctxOwn, inbox.ID, userOwn); err != nil || role != "admin" {
		t.Fatalf("GetMemberRole (own ctx): expected admin/nil, got %q/%v", role, err)
	}
	if members, err := repo.ListMembers(ctxOther, inbox.ID); err != nil || len(members) != 0 {
		t.Fatalf("ListMembers (foreign ctx): expected empty/nil, got %d/%v", len(members), err)
	}
	if members, err := repo.ListMembers(ctxOwn, inbox.ID); err != nil || len(members) != 1 {
		t.Fatalf("ListMembers (own ctx): expected 1/nil, got %d/%v", len(members), err)
	}
	if count, err := repo.GetMemberCount(ctxOther, inbox.ID); err != nil || count != 0 {
		t.Fatalf("GetMemberCount (foreign ctx): expected 0/nil, got %d/%v", count, err)
	}
	if count, err := repo.GetMemberCount(ctxOwn, inbox.ID); err != nil || count != 1 {
		t.Fatalf("GetMemberCount (own ctx): expected 1/nil, got %d/%v", count, err)
	}
	if count, err := repo.CountAdmins(ctxOther, inbox.ID); err != nil || count != 0 {
		t.Fatalf("CountAdmins (foreign ctx): expected 0/nil, got %d/%v", count, err)
	}
	if count, err := repo.CountAdmins(ctxOwn, inbox.ID); err != nil || count != 1 {
		t.Fatalf("CountAdmins (own ctx): expected 1/nil, got %d/%v", count, err)
	}

	// ListByUser
	if list, err := repo.ListByUser(ctxOther, tenantOwn, userOwn); err != nil || len(list) != 0 {
		t.Fatalf("ListByUser (foreign ctx): expected empty/nil, got %d/%v", len(list), err)
	}
	if list, err := repo.ListByUser(ctxOwn, tenantOwn, userOwn); err != nil || len(list) != 1 {
		t.Fatalf("ListByUser (own ctx): expected 1/nil, got %d/%v", len(list), err)
	}

	// IncrementAssigneeIndex — round-robin basis: repeated calls ascend.
	if _, err := repo.IncrementAssigneeIndex(ctxOther, tenantOwn, inbox.ID); err != ErrTeamInboxNotFound {
		t.Fatalf("IncrementAssigneeIndex (foreign ctx): expected ErrTeamInboxNotFound, got %v", err)
	}
	first, err := repo.IncrementAssigneeIndex(ctxOwn, tenantOwn, inbox.ID)
	if err != nil {
		t.Fatalf("IncrementAssigneeIndex #1: %v", err)
	}
	second, err := repo.IncrementAssigneeIndex(ctxOwn, tenantOwn, inbox.ID)
	if err != nil {
		t.Fatalf("IncrementAssigneeIndex #2: %v", err)
	}
	if second != first+1 {
		t.Fatalf("IncrementAssigneeIndex: expected ascending index, got %d then %d", first, second)
	}

	// RemoveMember — foreign ctx must not land (RLS-only, no tenantID param).
	if err := repo.RemoveMember(ctxOther, inbox.ID, userOwn); err != ErrNotTeamMember {
		t.Fatalf("RemoveMember (foreign ctx): expected ErrNotTeamMember, got %v", err)
	}
	if isMember, _ := repo.IsMember(ctxOwn, inbox.ID, userOwn); !isMember {
		t.Fatalf("a foreign-tenant RemoveMember reached the row")
	}
	if err := repo.RemoveMember(ctxOwn, inbox.ID, userOwn); err != nil {
		t.Fatalf("RemoveMember (own ctx): %v", err)
	}
	if err := repo.RemoveMember(ctxOwn, inbox.ID, userOwn); err != ErrNotTeamMember {
		t.Fatalf("RemoveMember (already removed): expected ErrNotTeamMember, got %v", err)
	}

	// DeleteTeamInbox — foreign ctx must not land, own ctx must.
	if err := repo.DeleteTeamInbox(ctxOther, tenantOwn, inbox.ID); err != ErrTeamInboxNotFound {
		t.Fatalf("DeleteTeamInbox (foreign ctx): expected ErrTeamInboxNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "team_inboxes", inbox.ID, 1)
	if err := repo.DeleteTeamInbox(ctxOwn, tenantOwn, inbox.ID); err != nil {
		t.Fatalf("DeleteTeamInbox (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "team_inboxes", inbox.ID, 0)
	if err := repo.DeleteTeamInbox(ctxOwn, tenantOwn, inbox.ID); err != ErrTeamInboxNotFound {
		t.Fatalf("DeleteTeamInbox (already deleted): expected ErrTeamInboxNotFound, got %v", err)
	}
}
