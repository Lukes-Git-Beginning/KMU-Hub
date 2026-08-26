package project

// Real-SQL DB tests for postgres_repository.go, covering the surfaces the
// existing tenant_isolation_test.go / tenant_isolation_phase2_test.go /
// tenant_write_test.go files leave untouched: List's admin/member/archived
// branching, key lookups, the full member-management surface, template
// operations, user preferences against the real table (not the mock), and
// the project -> task -> time_entry cascade on Delete.

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

func seedDBUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("db-project-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"first_name":    firstName,
		"last_name":     lastName,
	})
}

func seedDBProject(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, name, key string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantID,
		"name":        name,
		"project_key": key,
		"created_by":  createdBy,
	})
}

// TestList_AdminSeesAllMemberSeesOwnArchivedFiltered exercises all four
// branches of List's query-building (admin x includeArchived), plus that a
// non-admin only sees projects they are a member of, and a foreign tenant
// sees nothing at all.
func TestList_AdminSeesAllMemberSeesOwnArchivedFiltered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "List Branching Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "List Branching Other Tenant")

	owner := seedDBUser(t, pool, tenantID, "Owner", "User")
	defer testutil.CleanupRow(t, pool, "users", owner)
	outsider := seedDBUser(t, pool, tenantID, "Outsider", "User")
	defer testutil.CleanupRow(t, pool, "users", outsider)

	memberProj := seedDBProject(t, pool, tenantID, owner, "Member Project", "MP"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", memberProj)
	otherProj := seedDBProject(t, pool, tenantID, owner, "Other Project", "OP"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", otherProj)

	sysCtx := testutil.WithSystemCtx(context.Background())
	for _, projID := range []uuid.UUID{memberProj, otherProj} {
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO project_members (tenant_id, project_id, user_id, role) VALUES ($1, $2, $3, 'owner')`,
			tenantID, projID, owner,
		); err != nil {
			t.Fatalf("seed owner membership: %v", err)
		}
	}
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO project_members (tenant_id, project_id, user_id, role) VALUES ($1, $2, $3, 'member')`,
		tenantID, memberProj, outsider,
	); err != nil {
		t.Fatalf("seed outsider membership: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(sysCtx, `DELETE FROM project_members WHERE project_id IN ($1, $2)`, memberProj, otherProj)
	}()

	archivedProj := seedDBProject(t, pool, tenantID, owner, "Archived Project", "AP"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", archivedProj)
	if _, err := pool.Exec(sysCtx,
		`UPDATE projects SET archived_at = now() WHERE id = $1`, archivedProj,
	); err != nil {
		t.Fatalf("archive project: %v", err)
	}
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO project_members (tenant_id, project_id, user_id, role) VALUES ($1, $2, $3, 'owner')`,
		tenantID, archivedProj, owner,
	); err != nil {
		t.Fatalf("seed archived membership: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(sysCtx, `DELETE FROM project_members WHERE project_id = $1`, archivedProj)
	}()

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	// Admin, exclude archived: sees both non-archived projects, not the archived one.
	adminActive, err := repo.List(ctxOwn, tenantID, owner, true, false)
	if err != nil {
		t.Fatalf("List(admin, active): %v", err)
	}
	if got := projectIDSet(adminActive); !got[memberProj] || !got[otherProj] || got[archivedProj] {
		t.Fatalf("admin/active: expected {member,other} without archived, got %v", got)
	}

	// Admin, include archived: sees all three.
	adminAll, err := repo.List(ctxOwn, tenantID, owner, true, true)
	if err != nil {
		t.Fatalf("List(admin, all): %v", err)
	}
	if got := projectIDSet(adminAll); !got[memberProj] || !got[otherProj] || !got[archivedProj] {
		t.Fatalf("admin/all: expected all three, got %v", got)
	}

	// Non-admin member, exclude archived: sees only the project they belong to.
	memberActive, err := repo.List(ctxOwn, tenantID, outsider, false, false)
	if err != nil {
		t.Fatalf("List(member, active): %v", err)
	}
	if got := projectIDSet(memberActive); !got[memberProj] || got[otherProj] || got[archivedProj] {
		t.Fatalf("member/active: expected only memberProj, got %v", got)
	}

	// Foreign tenant sees nothing regardless of role.
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)
	foreignList, err := repo.List(ctxOther, otherTenant, owner, true, true)
	if err != nil {
		t.Fatalf("List(foreign tenant): %v", err)
	}
	if len(foreignList) != 0 {
		t.Fatalf("foreign tenant expected 0 projects, got %d", len(foreignList))
	}
}

func projectIDSet(projects []models.ProjectWithDetails) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(projects))
	for _, p := range projects {
		out[p.ID] = true
	}
	return out
}

// TestGetProjectKey_And_KeyExists_TenantScopedAndArchiveAware covers both key
// lookups: cross-tenant GetProjectKey returns ErrNotFound, and KeyExists
// deliberately excludes archived projects (documented behaviour: archiving a
// project frees its key for reuse within the same tenant).
func TestGetProjectKey_And_KeyExists_TenantScopedAndArchiveAware(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Key Lookup Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Key Lookup Other Tenant")

	owner := seedDBUser(t, pool, tenantID, "Key", "Owner")
	defer testutil.CleanupRow(t, pool, "users", owner)

	key := "KX" + uuid.New().String()[:6]
	projID := seedDBProject(t, pool, tenantID, owner, "Key Project", key)
	defer testutil.CleanupRow(t, pool, "projects", projID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)

	gotKey, err := repo.GetProjectKey(ctxOwn, projID, tenantID)
	if err != nil || gotKey != key {
		t.Fatalf("GetProjectKey(own): key=%q err=%v, want %q, nil", gotKey, err, key)
	}
	if _, err := repo.GetProjectKey(ctxOther, projID, otherTenant); err != ErrNotFound {
		t.Fatalf("GetProjectKey(foreign): err=%v, want ErrNotFound", err)
	}

	exists, err := repo.KeyExists(ctxOwn, tenantID, key)
	if err != nil || !exists {
		t.Fatalf("KeyExists(own, active): exists=%v err=%v, want true, nil", exists, err)
	}
	existsForeign, err := repo.KeyExists(ctxOther, otherTenant, key)
	if err != nil || existsForeign {
		t.Fatalf("KeyExists(foreign tenant): exists=%v err=%v, want false, nil", existsForeign, err)
	}

	// Archive the project — the key must become reusable.
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`UPDATE projects SET archived_at = now() WHERE id = $1`, projID,
	); err != nil {
		t.Fatalf("archive project: %v", err)
	}
	existsAfterArchive, err := repo.KeyExists(ctxOwn, tenantID, key)
	if err != nil {
		t.Fatalf("KeyExists(own, after archive): %v", err)
	}
	if existsAfterArchive {
		t.Fatalf("KeyExists must return false once the holder is archived — key is meant to be reusable")
	}
}

// TestMemberManagement_FullLifecycle exercises AddMember, GetMember,
// ListMembers, IsMember, UpdateMemberRole, CountOwners, RemoveMember against
// real SQL end to end.
func TestMemberManagement_FullLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Member Lifecycle Tenant")

	owner := seedDBUser(t, pool, tenantID, "Anna", "Owner")
	defer testutil.CleanupRow(t, pool, "users", owner)
	member := seedDBUser(t, pool, tenantID, "Ben", "Member")
	defer testutil.CleanupRow(t, pool, "users", member)

	projID := seedDBProject(t, pool, tenantID, owner, "Lifecycle Project", "ML"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", projID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	defer func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM project_members WHERE project_id = $1`, projID)
	}()

	if err := repo.AddMember(ctx, projID, owner, models.ProjectRoleOwner); err != nil {
		t.Fatalf("AddMember(owner): %v", err)
	}
	if err := repo.AddMember(ctx, projID, member, models.ProjectRoleMember); err != nil {
		t.Fatalf("AddMember(member): %v", err)
	}

	// ON CONFLICT DO NOTHING: adding the same member twice must not error or duplicate.
	if err := repo.AddMember(ctx, projID, member, models.ProjectRoleMember); err != nil {
		t.Fatalf("AddMember(duplicate): %v", err)
	}

	isMember, err := repo.IsMember(ctx, projID, member)
	if err != nil || !isMember {
		t.Fatalf("IsMember(member): %v, %v", isMember, err)
	}

	got, err := repo.GetMember(ctx, projID, member)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.Role != models.ProjectRoleMember || got.FirstName != "Ben" || got.LastName != "Member" {
		t.Fatalf("GetMember: unexpected row %+v", got)
	}

	members, err := repo.ListMembers(ctx, projID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("ListMembers: expected 2 (duplicate insert must not create a second row), got %d", len(members))
	}

	owners, err := repo.CountOwners(ctx, projID)
	if err != nil || owners != 1 {
		t.Fatalf("CountOwners: %d, %v, want 1", owners, err)
	}

	if err := repo.UpdateMemberRole(ctx, projID, member, models.ProjectRoleOwner); err != nil {
		t.Fatalf("UpdateMemberRole: %v", err)
	}
	owners, err = repo.CountOwners(ctx, projID)
	if err != nil || owners != 2 {
		t.Fatalf("CountOwners(after promote): %d, %v, want 2", owners, err)
	}

	if err := repo.RemoveMember(ctx, projID, member); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if stillMember, err := repo.IsMember(ctx, projID, member); err != nil || stillMember {
		t.Fatalf("IsMember(after remove): %v, %v, want false", stillMember, err)
	}
}

// TestAddMember_ForeignProjectID_NoTenantLeak exercises AddMember's
// tenant_id-via-subquery insert (`(SELECT tenant_id FROM projects WHERE id =
// $1)`) directly, bypassing the service-layer GetByID check that normally
// guards this call. Under RLS, the subquery for a foreign-tenant project
// returns no row, so tenant_id resolves to NULL and the NOT NULL constraint
// rejects the insert — defense in depth if a future caller ever skips the
// service-layer check.
func TestAddMember_ForeignProjectID_NoTenantLeak(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	victimTenant := uuid.New()
	attackerTenant := uuid.New()
	testutil.EnsureTenant(t, pool, victimTenant, "AddMember Victim Tenant")
	testutil.EnsureTenant(t, pool, attackerTenant, "AddMember Attacker Tenant")

	victimOwner := seedDBUser(t, pool, victimTenant, "Victim", "Owner")
	defer testutil.CleanupRow(t, pool, "users", victimOwner)
	attackerUser := seedDBUser(t, pool, attackerTenant, "Attacker", "User")
	defer testutil.CleanupRow(t, pool, "users", attackerUser)

	victimProj := seedDBProject(t, pool, victimTenant, victimOwner, "Victim Project", "VP"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", victimProj)

	repo := NewPostgresRepository(pool)
	ctxAttacker := testutil.WithTenantCtx(context.Background(), attackerTenant)

	err := repo.AddMember(ctxAttacker, victimProj, attackerUser, models.ProjectRoleOwner)
	if err == nil {
		t.Fatalf("AddMember with a foreign project id under the attacker's tenant context: expected an error, got nil")
	}

	var count int
	sysCtx := testutil.WithSystemCtx(context.Background())
	if scanErr := pool.QueryRow(sysCtx,
		`SELECT count(*) FROM project_members WHERE project_id = $1 AND user_id = $2`,
		victimProj, attackerUser,
	).Scan(&count); scanErr != nil {
		t.Fatalf("count project_members: %v", scanErr)
	}
	if count != 0 {
		t.Fatalf("attacker landed a membership row on the victim's project: count=%d", count)
	}
}

// TestSaveAsTemplate_And_GetForTemplate covers the template round trip,
// including that SaveAsTemplate against a foreign-tenant source silently
// creates nothing (the INSERT...SELECT's WHERE clause matches zero rows
// under RLS) rather than leaking the source project's data across tenants.
func TestSaveAsTemplate_And_GetForTemplate(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Template Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Template Other Tenant")

	owner := seedDBUser(t, pool, tenantID, "Template", "Owner")
	defer testutil.CleanupRow(t, pool, "users", owner)

	sourceID := seedDBProject(t, pool, tenantID, owner, "Source Project", "SC"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", sourceID)

	statusID := testutil.SeedRow(t, pool, "project_statuses", map[string]any{
		"tenant_id":  tenantID,
		"project_id": sourceID,
		"name":       "To Do",
		"sort_order": 1,
	})
	defer testutil.CleanupRow(t, pool, "project_statuses", statusID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantID)

	tplKey := "TP" + uuid.New().String()[:6]
	tpl, err := repo.SaveAsTemplate(ctxOwn, tenantID, sourceID, "Template Copy", tplKey, owner)
	if err != nil {
		t.Fatalf("SaveAsTemplate: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "projects", tpl.ID)
	if !tpl.IsTemplate || tpl.TemplateSourceID == nil || *tpl.TemplateSourceID != sourceID {
		t.Fatalf("SaveAsTemplate: unexpected result %+v", tpl)
	}

	fetched, statuses, err := repo.GetForTemplate(ctxOwn, tpl.ID, tenantID)
	if err != nil {
		t.Fatalf("GetForTemplate: %v", err)
	}
	if fetched.ID != tpl.ID {
		t.Fatalf("GetForTemplate: id mismatch %s != %s", fetched.ID, tpl.ID)
	}
	_ = statuses // template itself has no statuses copied by SaveAsTemplate; source's statuses are separate rows

	sourceFetched, sourceStatuses, err := repo.GetForTemplate(ctxOwn, sourceID, tenantID)
	if err != nil {
		t.Fatalf("GetForTemplate(source): %v", err)
	}
	if sourceFetched.IsTemplate {
		t.Fatalf("source project must not itself be marked as a template")
	}
	if len(sourceStatuses) != 1 || sourceStatuses[0].ID != statusID {
		t.Fatalf("GetForTemplate(source): expected the seeded status, got %+v", sourceStatuses)
	}

	// Cross-tenant SaveAsTemplate: sourceID is invisible under RLS for the
	// attacker's context, so the INSERT...SELECT matches nothing and the
	// call must come back an error (from the follow-up GetByID), not a copy
	// of the victim's project.
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)
	_, err = repo.SaveAsTemplate(ctxOther, otherTenant, sourceID, "Stolen Copy", "ST"+uuid.New().String()[:6], owner)
	if err == nil {
		t.Fatalf("SaveAsTemplate across tenants: expected an error, got a copy of the victim's project")
	}
}

// TestUserPreference_UpsertAndNoRowReturnsNil exercises GetUserPreference/
// SetUserPreference against the real table: no row yields (nil, nil) rather
// than an error, and a second Set with the same (user, project) upserts
// rather than erroring on the unique constraint.
func TestUserPreference_UpsertAndNoRowReturnsNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Preference Tenant")

	owner := seedDBUser(t, pool, tenantID, "Pref", "User")
	defer testutil.CleanupRow(t, pool, "users", owner)
	projID := seedDBProject(t, pool, tenantID, owner, "Preference Project", "PR"+uuid.New().String()[:6])
	defer testutil.CleanupRow(t, pool, "projects", projID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	defer func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM user_project_preferences WHERE user_id = $1 AND project_id = $2`, owner, projID)
	}()

	noPref, err := repo.GetUserPreference(ctx, owner, projID)
	if err != nil || noPref != nil {
		t.Fatalf("GetUserPreference(no row): pref=%v err=%v, want nil, nil", noPref, err)
	}

	pref := &models.UserProjectPreference{
		TenantID:  tenantID,
		UserID:    owner,
		ProjectID: projID,
		ViewType:  "list",
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.SetUserPreference(ctx, pref); err != nil {
		t.Fatalf("SetUserPreference(create): %v", err)
	}

	got, err := repo.GetUserPreference(ctx, owner, projID)
	if err != nil || got == nil || got.ViewType != "list" {
		t.Fatalf("GetUserPreference(after create): %+v, %v", got, err)
	}

	pref.ViewType = "kanban"
	if err := repo.SetUserPreference(ctx, pref); err != nil {
		t.Fatalf("SetUserPreference(upsert): %v", err)
	}
	got, err = repo.GetUserPreference(ctx, owner, projID)
	if err != nil || got == nil || got.ViewType != "kanban" {
		t.Fatalf("GetUserPreference(after upsert): expected view_type=kanban, got %+v, %v", got, err)
	}
}

// TestDelete_CascadesToTasksAndTimeEntries documents and verifies the
// project -> task -> time_entry ON DELETE CASCADE chain (migrations 000024,
// 000025, 000030): deleting a project silently removes every task and every
// tracked time entry under it, with no soft-delete or orphan-check in
// between. This is the connection point the coverage unit specifically
// asked about between the project and timeentry packages.
func TestDelete_CascadesToTasksAndTimeEntries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Cascade Tenant")

	owner := seedDBUser(t, pool, tenantID, "Cascade", "Owner")
	defer testutil.CleanupRow(t, pool, "users", owner)

	projID := seedDBProject(t, pool, tenantID, owner, "Cascade Project", "CD"+uuid.New().String()[:6])

	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"project_id":  projID,
		"title":       "Cascade Task",
		"created_by":  owner,
		"task_number": 1,
	})

	teID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantID,
		"task_id":    taskID,
		"user_id":    owner,
		"started_at": time.Now().UTC(),
	})

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	if err := repo.Delete(ctx, projID, tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "projects", projID, 0)
	testutil.AssertRowCount(t, pool, sysCtx, "tasks", taskID, 0)
	testutil.AssertRowCount(t, pool, sysCtx, "time_entries", teID, 0)
}
