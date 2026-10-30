package task

// Covers the repository paths around a task that no other DB test reaches:
// dependencies, the activity log, entity links, the template listings, the
// actor-name lookup and the NOTIFY emitter. The fixtures live in two tenants and
// every read runs once from the owner's RLS session and once from the foreign
// one, so a dropped tenant predicate or policy shows up as a leaked row.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// relationsFixture is one tenant with a project, a named user and three tasks
// in it, plus a second tenant that must never see any of it.
type relationsFixture struct {
	pool                *pgxpool.Pool
	repo                *PostgresRepository
	tenantOwn           uuid.UUID
	tenantOther         uuid.UUID
	user                uuid.UUID
	project             uuid.UUID
	taskA, taskB, taskC uuid.UUID
	ctxOwn, ctxOther    context.Context
}

func newRelationsFixture(t *testing.T) *relationsFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	f := &relationsFixture{pool: pool, tenantOwn: uuid.New(), tenantOther: uuid.New()}
	testutil.EnsureTenant(t, pool, f.tenantOwn, "Task Relations Tenant")
	testutil.EnsureTenant(t, pool, f.tenantOther, "Task Relations Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantOwn) })
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantOther) })

	f.user = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     f.tenantOwn,
		"email":         "relations-" + uuid.NewString() + "@test.invalid",
		"password_hash": "x",
		"first_name":    "Ada",
		"last_name":     "Lovelace",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.user) })
	f.project = seedWorkProject(t, pool, f.tenantOwn, f.user)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "projects", f.project) })
	f.taskA = seedWorkTask(t, pool, f.tenantOwn, f.project, f.user, "Relations A")
	f.taskB = seedWorkTask(t, pool, f.tenantOwn, f.project, f.user, "Relations B")
	f.taskC = seedWorkTask(t, pool, f.tenantOwn, f.project, f.user, "Relations C")
	t.Cleanup(func() {
		testutil.CleanupRow(t, pool, "tasks", f.taskC)
		testutil.CleanupRow(t, pool, "tasks", f.taskB)
		testutil.CleanupRow(t, pool, "tasks", f.taskA)
	})

	f.repo = NewPostgresRepository(pool)
	f.ctxOwn = testutil.WithTenantCtx(context.Background(), f.tenantOwn)
	f.ctxOther = testutil.WithTenantCtx(context.Background(), f.tenantOther)
	return f
}

func (f *relationsFixture) dependency(source, target uuid.UUID, kind string, at time.Time) *models.TaskDependency {
	return &models.TaskDependency{
		ID: uuid.New(), SourceTaskID: source, TargetTaskID: target,
		DependencyType: kind, CreatedBy: f.user, CreatedAt: at,
	}
}

func TestDependencies_CreateListAndTenantScope(t *testing.T) {
	f := newRelationsFixture(t)
	sysCtx := testutil.WithSystemCtx(context.Background())
	base := time.Now().Add(-time.Hour)

	ab := f.dependency(f.taskA, f.taskB, "blocks", base)
	bc := f.dependency(f.taskB, f.taskC, "relates_to", base.Add(time.Minute))
	for _, dep := range []*models.TaskDependency{bc, ab} { // inserted out of order on purpose
		if err := f.repo.CreateDependency(f.ctxOwn, dep); err != nil {
			t.Fatalf("CreateDependency: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "task_dependencies", dep.ID) })
	}

	// The row's tenant comes from the source task, not from the caller.
	var tenantOfRow uuid.UUID
	if err := f.pool.QueryRow(sysCtx, `SELECT tenant_id FROM task_dependencies WHERE id = $1`, ab.ID).Scan(&tenantOfRow); err != nil {
		t.Fatalf("read back tenant_id: %v", err)
	}
	if tenantOfRow != f.tenantOwn {
		t.Fatalf("dependency tenant_id = %s, want the source task's tenant %s", tenantOfRow, f.tenantOwn)
	}

	// B is the target of A->B and the source of B->C: both show up, oldest first.
	deps, err := f.repo.ListDependencies(f.ctxOwn, f.taskB)
	if err != nil {
		t.Fatalf("ListDependencies: %v", err)
	}
	if len(deps) != 2 || deps[0].ID != ab.ID || deps[1].ID != bc.ID {
		t.Fatalf("ListDependencies(B) = %+v, want [A->B, B->C] in created_at order", deps)
	}
	if deps[0].DependencyType != "blocks" || deps[0].SourceTaskID != f.taskA || deps[0].TargetTaskID != f.taskB {
		t.Fatalf("A->B read back wrong: %+v", deps[0])
	}

	// A task with no dependencies yields an empty result, not an error.
	none, err := f.repo.ListDependencies(f.ctxOwn, uuid.New())
	if err != nil || len(none) != 0 {
		t.Fatalf("ListDependencies(unknown) = %v, %v; want empty, nil", none, err)
	}

	// The project listing follows the source task.
	byProject, err := f.repo.ListDependenciesByProject(f.ctxOwn, f.project)
	if err != nil {
		t.Fatalf("ListDependenciesByProject: %v", err)
	}
	if len(byProject) != 2 {
		t.Fatalf("ListDependenciesByProject = %d rows, want 2", len(byProject))
	}
	if other, err := f.repo.ListDependenciesByProject(f.ctxOwn, uuid.New()); err != nil || len(other) != 0 {
		t.Fatalf("ListDependenciesByProject(unknown project) = %v, %v; want empty, nil", other, err)
	}

	// The other tenant's session sees none of it.
	if foreign, err := f.repo.ListDependencies(f.ctxOther, f.taskB); err != nil || len(foreign) != 0 {
		t.Fatalf("ListDependencies from foreign tenant = %v, %v; want empty", foreign, err)
	}
	if foreign, err := f.repo.ListDependenciesByProject(f.ctxOther, f.project); err != nil || len(foreign) != 0 {
		t.Fatalf("ListDependenciesByProject from foreign tenant = %v, %v; want empty", foreign, err)
	}

	// A foreign session cannot attach a dependency to a task it cannot see: the
	// tenant subselect finds nothing, tenant_id would be NULL, the insert fails.
	forged := f.dependency(f.taskA, f.taskC, "blocks", base)
	if err := f.repo.CreateDependency(f.ctxOther, forged); err == nil {
		t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "task_dependencies", forged.ID) })
		t.Fatal("CreateDependency from a foreign tenant succeeded against a task it cannot see")
	}
	testutil.AssertRowCount(t, f.pool, sysCtx, "task_dependencies", forged.ID, 0)

	// The service passes the delete through with the tenant it was given.
	svc := NewService(f.repo, nil)
	if err := svc.DeleteDependency(f.ctxOwn, f.tenantOther, ab.ID, nil); err != nil {
		t.Fatalf("Service.DeleteDependency (foreign tenantID): %v", err)
	}
	testutil.AssertRowCount(t, f.pool, sysCtx, "task_dependencies", ab.ID, 1)
	if err := svc.DeleteDependency(f.ctxOwn, f.tenantOwn, ab.ID, nil); err != nil {
		t.Fatalf("Service.DeleteDependency: %v", err)
	}
	testutil.AssertRowCount(t, f.pool, sysCtx, "task_dependencies", ab.ID, 0)
}

func TestActivities_CreateAndListPaged(t *testing.T) {
	f := newRelationsFixture(t)
	sysCtx := testutil.WithSystemCtx(context.Background())
	base := time.Now().Add(-time.Hour)

	field := "title"
	oldVal, newVal := json.RawMessage(`"Old"`), json.RawMessage(`"New"`)
	ids := make([]uuid.UUID, 0, 25)
	for i := 0; i < 25; i++ {
		a := &models.TaskActivity{
			ID: uuid.New(), TaskID: f.taskA, ActorID: f.user, Action: "updated",
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if i == 24 { // the newest entry carries a field change
			a.FieldName, a.OldValue, a.NewValue = &field, &oldVal, &newVal
		}
		if err := f.repo.CreateActivity(f.ctxOwn, a); err != nil {
			t.Fatalf("CreateActivity %d: %v", i, err)
		}
		ids = append(ids, a.ID)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			testutil.CleanupRow(t, f.pool, "task_activities", id)
		}
	})

	var tenantOfRow uuid.UUID
	if err := f.pool.QueryRow(sysCtx, `SELECT tenant_id FROM task_activities WHERE id = $1`, ids[0]).Scan(&tenantOfRow); err != nil {
		t.Fatalf("read back tenant_id: %v", err)
	}
	if tenantOfRow != f.tenantOwn {
		t.Fatalf("activity tenant_id = %s, want the task's tenant %s", tenantOfRow, f.tenantOwn)
	}

	// Defaults: page 0 -> 1, page size 0 -> 20. Newest first, total counts all 25.
	page, total, err := f.repo.ListActivities(f.ctxOwn, f.taskA, 0, 0)
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if total != 25 || len(page) != 20 {
		t.Fatalf("first page: total=%d len=%d, want 25 and 20", total, len(page))
	}
	if page[0].ID != ids[24] || page[19].ID != ids[5] {
		t.Fatalf("first page not newest-first: first=%s last=%s", page[0].ID, page[19].ID)
	}
	if page[0].ActorName != "Ada Lovelace" {
		t.Fatalf("ActorName = %q, want the joined user name", page[0].ActorName)
	}
	if page[0].FieldName == nil || *page[0].FieldName != "title" ||
		page[0].OldValue == nil || string(*page[0].OldValue) != `"Old"` ||
		page[0].NewValue == nil || string(*page[0].NewValue) != `"New"` {
		t.Fatalf("field change did not round-trip: %+v", page[0])
	}
	if page[1].FieldName != nil || page[1].OldValue != nil {
		t.Fatalf("plain entry grew a field change: %+v", page[1])
	}

	// Second page holds the five oldest; an oversize page size falls back to 20.
	page2, total2, err := f.repo.ListActivities(f.ctxOwn, f.taskA, 2, 500)
	if err != nil {
		t.Fatalf("ListActivities page 2: %v", err)
	}
	if total2 != 25 || len(page2) != 5 || page2[4].ID != ids[0] {
		t.Fatalf("second page: total=%d len=%d, want 25 and the 5 oldest", total2, len(page2))
	}

	// Past the end: empty page, total unchanged.
	past, totalPast, err := f.repo.ListActivities(f.ctxOwn, f.taskA, 9, 20)
	if err != nil || len(past) != 0 || totalPast != 25 {
		t.Fatalf("page past the end = %d rows, total %d, err %v; want 0 rows, total 25", len(past), totalPast, err)
	}

	// The other tenant sees neither rows nor a count.
	foreign, foreignTotal, err := f.repo.ListActivities(f.ctxOther, f.taskA, 1, 20)
	if err != nil || len(foreign) != 0 || foreignTotal != 0 {
		t.Fatalf("foreign tenant sees %d rows, total %d, err %v; want nothing", len(foreign), foreignTotal, err)
	}
}

func TestEntityLinks_LinkListAndTasksForEntity(t *testing.T) {
	f := newRelationsFixture(t)
	sysCtx := testutil.WithSystemCtx(context.Background())
	contactID := uuid.New()
	base := time.Now().Add(-time.Hour)

	// A is linked to the contact and to a deal; B is linked to the same contact.
	linkAContact := &models.TaskEntityLink{ID: uuid.New(), TenantID: f.tenantOwn, TaskID: f.taskA, EntityType: "contact", EntityID: contactID, CreatedBy: f.user, CreatedAt: base}
	linkADeal := &models.TaskEntityLink{ID: uuid.New(), TenantID: f.tenantOwn, TaskID: f.taskA, EntityType: "deal", EntityID: uuid.New(), CreatedBy: f.user, CreatedAt: base.Add(time.Minute)}
	linkBContact := &models.TaskEntityLink{ID: uuid.New(), TenantID: f.tenantOwn, TaskID: f.taskB, EntityType: "contact", EntityID: contactID, CreatedBy: f.user, CreatedAt: base.Add(2 * time.Minute)}
	for _, l := range []*models.TaskEntityLink{linkADeal, linkAContact, linkBContact} {
		if err := f.repo.LinkEntity(f.ctxOwn, l); err != nil {
			t.Fatalf("LinkEntity %s: %v", l.EntityType, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "task_entity_links", l.ID) })
	}

	// The same task cannot be linked to the same entity twice.
	dup := *linkAContact
	dup.ID = uuid.New()
	if err := f.repo.LinkEntity(f.ctxOwn, &dup); err == nil {
		t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "task_entity_links", dup.ID) })
		t.Fatal("LinkEntity accepted the same task/entity pair twice")
	}

	links, err := f.repo.ListEntityLinks(f.ctxOwn, f.taskA)
	if err != nil {
		t.Fatalf("ListEntityLinks: %v", err)
	}
	if len(links) != 2 || links[0].ID != linkAContact.ID || links[1].ID != linkADeal.ID {
		t.Fatalf("ListEntityLinks(A) = %+v, want [contact, deal] in created_at order", links)
	}
	if links[0].EntityType != "contact" || links[0].EntityID != contactID || links[0].CreatedBy != f.user {
		t.Fatalf("link read back wrong: %+v", links[0])
	}
	if links[0].EntityDisplayName != "" {
		t.Fatalf("EntityDisplayName = %q, want the column default", links[0].EntityDisplayName)
	}
	if foreign, err := f.repo.ListEntityLinks(f.ctxOther, f.taskA); err != nil || len(foreign) != 0 {
		t.Fatalf("ListEntityLinks from foreign tenant = %v, %v; want empty", foreign, err)
	}

	// Both tasks come back for the contact, newest first, with their relations.
	status := testutil.SeedRow(t, f.pool, "project_statuses", map[string]any{
		"tenant_id": f.tenantOwn, "project_id": f.project, "name": "In Review", "is_closed": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "project_statuses", status) })
	if _, err := f.pool.Exec(sysCtx, `UPDATE tasks SET status_id = $1, assignee_id = $2, created_at = $3 WHERE id = $4`, status, f.user, base, f.taskA); err != nil {
		t.Fatalf("decorate task A: %v", err)
	}
	if _, err := f.pool.Exec(sysCtx, `UPDATE tasks SET created_at = $1 WHERE id = $2`, base.Add(time.Minute), f.taskB); err != nil {
		t.Fatalf("date task B: %v", err)
	}
	child := seedWorkTask(t, f.pool, f.tenantOwn, f.project, f.user, "Relations A child")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tasks", child) })
	if _, err := f.pool.Exec(sysCtx, `UPDATE tasks SET parent_task_id = $1, depth = 1 WHERE id = $2`, f.taskA, child); err != nil {
		t.Fatalf("parent child task: %v", err)
	}
	blocks := testutil.SeedRow(t, f.pool, "task_dependencies", map[string]any{
		"tenant_id": f.tenantOwn, "source_task_id": f.taskC, "target_task_id": f.taskA,
		"dependency_type": "blocks", "created_by": f.user,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "task_dependencies", blocks) })

	tasks, err := f.repo.ListTasksForEntity(f.ctxOwn, "contact", contactID)
	if err != nil {
		t.Fatalf("ListTasksForEntity: %v", err)
	}
	if len(tasks) != 2 || tasks[0].ID != f.taskB || tasks[1].ID != f.taskA {
		t.Fatalf("ListTasksForEntity(contact) = %d tasks, want [B, A] newest first", len(tasks))
	}
	a := tasks[1]
	if a.StatusName == nil || *a.StatusName != "In Review" || !a.StatusIsClosed {
		t.Fatalf("status relation missing on A: name=%v closed=%v", a.StatusName, a.StatusIsClosed)
	}
	if a.AssigneeName == nil || *a.AssigneeName != "Ada Lovelace" {
		t.Fatalf("assignee relation missing on A: %v", a.AssigneeName)
	}
	if a.ProjectKey == nil || *a.ProjectKey == "" {
		t.Fatalf("project key missing on A: %v", a.ProjectKey)
	}
	if a.SubtaskCount != 1 || !a.HasBlockedDeps {
		t.Fatalf("A: subtask_count=%d has_blocked_deps=%v, want 1 and true", a.SubtaskCount, a.HasBlockedDeps)
	}
	if tasks[0].SubtaskCount != 0 || tasks[0].HasBlockedDeps {
		t.Fatalf("B picked up A's relations: %+v", tasks[0])
	}

	// Entity type and id are both part of the match.
	if wrongType, err := f.repo.ListTasksForEntity(f.ctxOwn, "company", contactID); err != nil || len(wrongType) != 0 {
		t.Fatalf("ListTasksForEntity(company, contactID) = %v, %v; want empty", wrongType, err)
	}
	if foreign, err := f.repo.ListTasksForEntity(f.ctxOther, "contact", contactID); err != nil || len(foreign) != 0 {
		t.Fatalf("ListTasksForEntity from foreign tenant = %v, %v; want empty", foreign, err)
	}

	// Unlinking drops the task from the entity's list.
	if err := f.repo.UnlinkEntity(f.ctxOwn, f.tenantOwn, linkBContact.ID); err != nil {
		t.Fatalf("UnlinkEntity: %v", err)
	}
	remaining, err := f.repo.ListTasksForEntity(f.ctxOwn, "contact", contactID)
	if err != nil || len(remaining) != 1 || remaining[0].ID != f.taskA {
		t.Fatalf("after unlink = %v, %v; want only task A", remaining, err)
	}
}

func TestListByProject_OrderAndTenantScope(t *testing.T) {
	f := newRelationsFixture(t)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// Order is depth first, then sort_order: C(depth 0, 1) < A(depth 0, 2) < B(depth 1, 0).
	for _, row := range []struct {
		id    uuid.UUID
		depth int
		sort  float64
	}{{f.taskA, 0, 2}, {f.taskB, 1, 0}, {f.taskC, 0, 1}} {
		if _, err := f.pool.Exec(sysCtx, `UPDATE tasks SET depth = $1, sort_order = $2 WHERE id = $3`, row.depth, row.sort, row.id); err != nil {
			t.Fatalf("arrange tasks: %v", err)
		}
	}

	tasks, err := f.repo.ListByProject(f.ctxOwn, f.project)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(tasks) != 3 || tasks[0].ID != f.taskC || tasks[1].ID != f.taskA || tasks[2].ID != f.taskB {
		t.Fatalf("ListByProject order wrong: %+v", tasks)
	}
	if tasks[2].Depth != 1 || tasks[0].Title != "Relations C" {
		t.Fatalf("task columns did not round-trip: %+v", tasks[2])
	}

	if empty, err := f.repo.ListByProject(f.ctxOwn, uuid.New()); err != nil || len(empty) != 0 {
		t.Fatalf("ListByProject(unknown) = %v, %v; want empty", empty, err)
	}
	if foreign, err := f.repo.ListByProject(f.ctxOther, f.project); err != nil || len(foreign) != 0 {
		t.Fatalf("ListByProject from foreign tenant = %v, %v; want empty", foreign, err)
	}
}

func TestGetUserDisplayName_FoundBlankAndUnknown(t *testing.T) {
	f := newRelationsFixture(t)

	name, err := f.repo.GetUserDisplayName(f.ctxOwn, f.user)
	if err != nil || name != "Ada Lovelace" {
		t.Fatalf("GetUserDisplayName = %q, %v; want %q", name, err, "Ada Lovelace")
	}

	unnamed := testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     f.tenantOwn,
		"email":         "unnamed-" + uuid.NewString() + "@test.invalid",
		"password_hash": "x",
		"first_name":    "",
		"last_name":     "",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "users", unnamed) })
	if blank, err := f.repo.GetUserDisplayName(f.ctxOwn, unnamed); err != nil || blank != "" {
		t.Fatalf("GetUserDisplayName(blank names) = %q, %v; want empty string", blank, err)
	}

	// A user that does not exist is not an error: emitting must never fail on it.
	if missing, err := f.repo.GetUserDisplayName(f.ctxOwn, uuid.New()); err != nil || missing != "" {
		t.Fatalf("GetUserDisplayName(unknown) = %q, %v; want empty string, nil", missing, err)
	}

	// The service swallows the lookup entirely and falls back to "".
	svc := NewService(f.repo, nil)
	if got := svc.resolveActorName(f.ctxOwn, f.user); got != "Ada Lovelace" {
		t.Fatalf("resolveActorName = %q, want %q", got, "Ada Lovelace")
	}
}

func TestPGEventEmitter_RequiresTenantAndEmits(t *testing.T) {
	f := newRelationsFixture(t)
	emitter := NewPGEventEmitter(f.pool)

	payload := models.EventPayload{Type: "task.assigned", ModuleID: "work", Priority: "normal", Timestamp: time.Now()}

	// No tenant on the payload and none on the context: refused, not sent.
	if err := emitter.EmitTaskEvent(context.Background(), payload); !errors.Is(err, event.ErrMissingTenant) {
		t.Fatalf("EmitTaskEvent without tenant = %v, want ErrMissingTenant", err)
	}

	// The tenant is taken from the caller's context when the payload has none.
	if err := emitter.EmitTaskEvent(f.ctxOwn, payload); err != nil {
		t.Fatalf("EmitTaskEvent with tenant context: %v", err)
	}
}
