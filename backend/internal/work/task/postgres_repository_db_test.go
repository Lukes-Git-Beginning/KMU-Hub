package task

// Covers the repository surface tenant_write_test.go/rls_test.go leave open:
// List/Search filter combinations plus their tenant-scoped totals, the
// recursive tree navigation (GetSubtasks/GetParentChain/GetDepth), and
// HasCycle's dependency-graph traversal — all against the real schema, not
// mocks. See BACKLOG.yml unit c-cov-work-task-repo (Lauf 7).

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestList_FiltersAndTenantScopedTotal covers List's filter combinations
// (priority, assignee, status, standalone, title search) and proves the
// COUNT(*) total carries the exact same WHERE as the page — a foreign-tenant
// task with a matching title must never inflate the total, the class of bug
// the roter Faden of this Lauf calls out for gateway/server coverage.
func TestList_FiltersAndTenantScopedTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	statusID := testutil.SeedRow(t, pool, "project_statuses", map[string]any{
		"tenant_id":  tenantOwn,
		"project_id": projectOwn,
		"name":       "Open",
	})
	defer testutil.CleanupRow(t, pool, "project_statuses", statusID)

	taskUrgent := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "List-Match Urgent " + uuid.New().String()[:8],
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"priority":    "urgent",
		"assignee_id": userOwn,
		"status_id":   statusID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskUrgent)

	taskLow := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Unrelated low task " + uuid.New().String()[:8],
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"priority":    "low",
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskLow)

	taskStandalone := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"title":       "Standalone task " + uuid.New().String()[:8],
		"task_number": taskNumber(),
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskStandalone)

	// Foreign-tenant task sharing the "List-Match" title substring — must
	// never appear in tenantOwn's results or its total.
	userOther := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", userOther)
	projectOther := seedWorkProject(t, pool, tenantOther, userOther)
	defer testutil.CleanupRow(t, pool, "projects", projectOther)
	taskForeign := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOther,
		"project_id":  projectOther,
		"title":       "List-Match Urgent foreign",
		"task_number": taskNumber(),
		"created_by":  userOther,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskForeign)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	urgent := "urgent"
	results, total, err := repo.List(ctxOwn, tenantOwn, TaskFilters{Priority: &urgent, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (priority): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != taskUrgent {
		t.Fatalf("List (priority=urgent): expected exactly taskUrgent, got total=%d results=%v", total, results)
	}

	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{AssigneeID: &userOwn, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (assignee): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != taskUrgent {
		t.Fatalf("List (assignee=userOwn): expected exactly taskUrgent, got total=%d results=%v", total, results)
	}

	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{StatusID: &statusID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (status): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != taskUrgent {
		t.Fatalf("List (status): expected exactly taskUrgent, got total=%d results=%v", total, results)
	}

	standalone := true
	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{IsStandalone: &standalone, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (standalone): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != taskStandalone {
		t.Fatalf("List (standalone): expected exactly taskStandalone, got total=%d results=%v", total, results)
	}

	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{Search: "List-Match", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (search): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != taskUrgent {
		t.Fatalf("List (search=List-Match): expected exactly taskUrgent (not the foreign-tenant match), got total=%d results=%v", total, results)
	}

	// Unfiltered: total must be the tenant's own 3 rows — the foreign task
	// (same title substring, different tenant) must not leak into the count.
	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List (unfiltered): %v", err)
	}
	if total != 3 || len(results) != 3 {
		t.Fatalf("List (unfiltered): expected total=3 own-tenant rows, got total=%d results=%d — possible cross-tenant leak into COUNT(*)", total, len(results))
	}

	// Pagination: page size 1 still reports the full tenant-scoped total.
	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{Page: 1, PageSize: 1, SortBy: "title", SortDesc: false})
	if err != nil {
		t.Fatalf("List (paginated): %v", err)
	}
	if total != 3 || len(results) != 1 {
		t.Fatalf("List (paginated): expected total=3 with a single-row page, got total=%d rows=%d", total, len(results))
	}
}

// TestSearch_FullTextAndTenantScopedTotal covers Search's full-text matching
// plus the same tenant-scoped-total guarantee as List.
func TestSearch_FullTextAndTenantScopedTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Search Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task Search Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	// The full-text token below is not a German dictionary word, so the
	// 'german' tsvector config keeps it as a single stable lexeme — safe
	// against stemming surprises across parallel test runs.
	token := "Suchtokenxyz"

	matchA := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       token + " erste Aufgabe",
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"priority":    "high",
	})
	defer testutil.CleanupRow(t, pool, "tasks", matchA)

	matchB := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"description": token + " in der Beschreibung",
		"title":       "Zweite Aufgabe " + uuid.New().String()[:8],
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"priority":    "low",
	})
	defer testutil.CleanupRow(t, pool, "tasks", matchB)

	noMatch := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Ganz andere Aufgabe " + uuid.New().String()[:8],
		"task_number": taskNumber(),
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "tasks", noMatch)

	// Same token, foreign tenant — must never appear in tenantOwn's results
	// or total.
	userOther := seedWorkUser(t, pool, tenantOther)
	defer testutil.CleanupRow(t, pool, "users", userOther)
	projectOther := seedWorkProject(t, pool, tenantOther, userOther)
	defer testutil.CleanupRow(t, pool, "projects", projectOther)
	foreignMatch := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOther,
		"project_id":  projectOther,
		"title":       token + " foreign tenant",
		"task_number": taskNumber(),
		"created_by":  userOther,
	})
	defer testutil.CleanupRow(t, pool, "tasks", foreignMatch)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	results, total, err := repo.Search(ctxOwn, tenantOwn, token, TaskSearchFilters{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 2 {
		t.Fatalf("Search: expected total=2 own-tenant matches, got total=%d results=%v — possible cross-tenant leak into COUNT(*)", total, results)
	}
	gotIDs := map[uuid.UUID]bool{}
	for _, r := range results {
		gotIDs[r.ID] = true
	}
	if !gotIDs[matchA] || !gotIDs[matchB] || gotIDs[noMatch] || gotIDs[foreignMatch] {
		t.Fatalf("Search: expected exactly {matchA,matchB}, got %v", gotIDs)
	}

	// Priority filter narrows the full-text match set the same way List's
	// filters do.
	high := "high"
	results, total, err = repo.Search(ctxOwn, tenantOwn, token, TaskSearchFilters{Priority: &high, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search (priority): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != matchA {
		t.Fatalf("Search (priority=high): expected exactly matchA, got total=%d results=%v", total, results)
	}

	// A query term absent from every own-tenant task must never accidentally
	// fall back to matching everything.
	results, total, err = repo.Search(ctxOwn, tenantOwn, "NoSuchTokenAtAll999", TaskSearchFilters{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search (no match): %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("Search (no match): expected 0 results, got total=%d results=%v", total, results)
	}
}

// TestGetSubtasksAndGetParentChain_MultiLevelOrder covers the recursive tree
// navigation across three levels: parent -> two children -> one grandchild.
func TestGetSubtasksAndGetParentChain_MultiLevelOrder(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Tree Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	parent := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Tree parent",
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"depth":       0,
		"sort_order":  0,
	})
	defer testutil.CleanupRow(t, pool, "tasks", parent)

	child1 := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":      tenantOwn,
		"project_id":     projectOwn,
		"title":          "Tree child 1",
		"task_number":    taskNumber(),
		"created_by":     userOwn,
		"parent_task_id": parent,
		"depth":          1,
		"sort_order":     1,
	})
	defer testutil.CleanupRow(t, pool, "tasks", child1)

	child2 := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":      tenantOwn,
		"project_id":     projectOwn,
		"title":          "Tree child 2",
		"task_number":    taskNumber(),
		"created_by":     userOwn,
		"parent_task_id": parent,
		"depth":          1,
		"sort_order":     2,
	})
	defer testutil.CleanupRow(t, pool, "tasks", child2)

	grandchild := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":      tenantOwn,
		"project_id":     projectOwn,
		"title":          "Tree grandchild",
		"task_number":    taskNumber(),
		"created_by":     userOwn,
		"parent_task_id": child1,
		"depth":          2,
		"sort_order":     1,
	})
	defer testutil.CleanupRow(t, pool, "tasks", grandchild)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// maxDepth=5: full subtree in depth-then-sort_order order.
	subtasks, err := repo.GetSubtasks(ctx, parent, 5)
	if err != nil {
		t.Fatalf("GetSubtasks (maxDepth=5): %v", err)
	}
	if len(subtasks) != 3 {
		t.Fatalf("GetSubtasks (maxDepth=5): expected 3 rows, got %d: %v", len(subtasks), subtasks)
	}
	if subtasks[0].ID != child1 || subtasks[1].ID != child2 || subtasks[2].ID != grandchild {
		t.Fatalf("GetSubtasks (maxDepth=5): wrong order, got [%s %s %s], want [child1 child2 grandchild]",
			subtasks[0].ID, subtasks[1].ID, subtasks[2].ID)
	}

	// maxDepth=1: recursion into depth-2 rows must be cut off — only the
	// direct children come back, not the grandchild.
	subtasks, err = repo.GetSubtasks(ctx, parent, 1)
	if err != nil {
		t.Fatalf("GetSubtasks (maxDepth=1): %v", err)
	}
	if len(subtasks) != 2 {
		t.Fatalf("GetSubtasks (maxDepth=1): expected 2 direct children only, got %d: %v", len(subtasks), subtasks)
	}
	for _, st := range subtasks {
		if st.ID == grandchild {
			t.Fatalf("GetSubtasks (maxDepth=1): grandchild leaked past the depth cutoff")
		}
	}

	// GetParentChain from the grandchild must walk root-first: parent, then
	// child1, then the grandchild itself.
	chain, err := repo.GetParentChain(ctx, grandchild)
	if err != nil {
		t.Fatalf("GetParentChain: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("GetParentChain: expected 3 rows, got %d: %v", len(chain), chain)
	}
	if chain[0].ID != parent || chain[1].ID != child1 || chain[2].ID != grandchild {
		t.Fatalf("GetParentChain: wrong order, got [%s %s %s], want [parent child1 grandchild]",
			chain[0].ID, chain[1].ID, chain[2].ID)
	}
}

// TestGetDepth_DeepChainAndNotFound covers GetDepth at a non-trivial nesting
// level and its ErrNotFound path.
func TestGetDepth_DeepChainAndNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Depth Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	// Build a chain reaching MaxNestingDepth-1 (depth 4), the deepest level
	// the service layer is meant to allow.
	var parentID *uuid.UUID
	var leaf uuid.UUID
	for depth := range MaxNestingDepth - 1 {
		cols := map[string]any{
			"tenant_id":   tenantOwn,
			"project_id":  projectOwn,
			"title":       fmt.Sprintf("Depth chain level %d", depth),
			"task_number": taskNumber(),
			"created_by":  userOwn,
			"depth":       depth,
		}
		if parentID != nil {
			cols["parent_task_id"] = *parentID
		}
		id := testutil.SeedRow(t, pool, "tasks", cols)
		defer testutil.CleanupRow(t, pool, "tasks", id)
		parentID = &id
		leaf = id
	}

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	got, err := repo.GetDepth(ctx, leaf)
	if err != nil {
		t.Fatalf("GetDepth (deep chain): %v", err)
	}
	if got != MaxNestingDepth-2 {
		t.Fatalf("GetDepth (deep chain): expected depth=%d, got %d", MaxNestingDepth-2, got)
	}

	if _, err := repo.GetDepth(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetDepth (missing task): expected ErrNotFound, got %v", err)
	}
}

// TestHasCycle_DirectAndTransitiveDetection covers HasCycle's dependency
// graph walk: it must catch both a direct (A -> B, adding B -> A) and a
// transitive (A -> B -> C, adding C -> A) cycle before either is inserted,
// while an unrelated task pair must not be flagged.
func TestHasCycle_DirectAndTransitiveDetection(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Cycle Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	taskA := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Cycle A")
	defer testutil.CleanupRow(t, pool, "tasks", taskA)
	taskB := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Cycle B")
	defer testutil.CleanupRow(t, pool, "tasks", taskB)
	taskC := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Cycle C")
	defer testutil.CleanupRow(t, pool, "tasks", taskC)
	taskD := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Cycle D (unrelated)")
	defer testutil.CleanupRow(t, pool, "tasks", taskD)

	depBA := testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
		"tenant_id":       tenantOwn,
		"source_task_id":  taskB,
		"target_task_id":  taskA,
		"dependency_type": "blocks",
		"created_by":      userOwn,
	})
	defer testutil.CleanupRow(t, pool, "task_dependencies", depBA)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// Direct cycle: B -> A already exists, so proposing A -> B must be
	// rejected as a cycle.
	hasCycle, err := repo.HasCycle(ctx, taskA, taskB)
	if err != nil {
		t.Fatalf("HasCycle (direct): %v", err)
	}
	if !hasCycle {
		t.Fatalf("HasCycle (direct): expected true for A->B given existing B->A dependency")
	}

	// Unrelated pair: D has no dependency edges at all, proposing D -> A
	// must not be flagged.
	hasCycle, err = repo.HasCycle(ctx, taskD, taskA)
	if err != nil {
		t.Fatalf("HasCycle (unrelated): %v", err)
	}
	if hasCycle {
		t.Fatalf("HasCycle (unrelated): expected false for D->A, D has no existing dependencies")
	}

	// Extend to a transitive chain: A -> B already implied by B -> A above
	// (blocks is directional, so add B -> C to build A(target)<-B(source),
	// B(target)<-C(source) i.e. the walk C -> B -> A).
	depCB := testutil.SeedRow(t, pool, "task_dependencies", map[string]any{
		"tenant_id":       tenantOwn,
		"source_task_id":  taskC,
		"target_task_id":  taskB,
		"dependency_type": "blocks",
		"created_by":      userOwn,
	})
	defer testutil.CleanupRow(t, pool, "task_dependencies", depCB)

	// Transitive cycle: C -> B -> A already exists, so proposing A -> C must
	// be rejected as a cycle.
	hasCycle, err = repo.HasCycle(ctx, taskA, taskC)
	if err != nil {
		t.Fatalf("HasCycle (transitive): %v", err)
	}
	if !hasCycle {
		t.Fatalf("HasCycle (transitive): expected true for A->C given existing C->B->A chain")
	}

	// D remains unrelated even after the chain grew.
	hasCycle, err = repo.HasCycle(ctx, taskD, taskC)
	if err != nil {
		t.Fatalf("HasCycle (unrelated after chain growth): %v", err)
	}
	if hasCycle {
		t.Fatalf("HasCycle (unrelated after chain growth): expected false for D->C")
	}
}

// TestCustomFieldValues_RoundTripAgainstWorkDefinitions pins the write and read
// path of task custom fields to work_custom_field_definitions — the table the
// /api/v1/work/custom-fields API actually hands ids out of. Until Migration
// 000320 the FK on task_custom_field_values.field_id still pointed at the CRM
// table custom_field_definitions (Migration 000026), so every SetCustomFieldValues
// failed with a foreign_key_violation and GetCustomFieldValues joined the wrong
// table. See BACKLOG.yml unit fix-work-task-custom-field-values-wrong-fk.
func TestCustomFieldValues_RoundTripAgainstWorkDefinitions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task Custom Field Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	taskID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "Custom Field Task")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	fieldName := "severity-" + uuid.New().String()[:8]
	fieldID := testutil.SeedRow(t, pool, "work_custom_field_definitions", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       fieldName,
		"field_type": "text",
	})
	defer testutil.CleanupRow(t, pool, "work_custom_field_definitions", fieldID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if err := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{fieldID: "high"}); err != nil {
		t.Fatalf("SetCustomFieldValues with a work_custom_field_definitions id must succeed: %v", err)
	}

	values, err := repo.GetCustomFieldValues(ctx, taskID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if got := values[fieldName]; got != "high" {
		t.Fatalf("GetCustomFieldValues[%q] = %v, want \"high\" (join must resolve work_custom_field_definitions.name)", fieldName, got)
	}

	// Upsert on the composite PK keeps a single row and returns the new value.
	if err := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{fieldID: "low"}); err != nil {
		t.Fatalf("SetCustomFieldValues (upsert): %v", err)
	}
	values, err = repo.GetCustomFieldValues(ctx, taskID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (after upsert): %v", err)
	}
	if len(values) != 1 || values[fieldName] != "low" {
		t.Fatalf("after upsert got %v, want exactly {%q: \"low\"}", values, fieldName)
	}

	// A field id that exists in the CRM definition table must not be accepted —
	// that is the wiring the old FK allowed and the work API can never produce.
	crmFieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   tenantOwn,
		"entity_type": "contact",
		"field_name":  "crm-" + uuid.New().String()[:8],
		"field_label": "CRM Field",
		"field_type":  "text",
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", crmFieldID)

	if err := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{crmFieldID: "nope"}); err == nil {
		t.Fatalf("SetCustomFieldValues with a CRM definition id must be rejected by the FK, got nil error")
	}
}

// TestSetCustomFieldValues_RejectsForeignAndUnknownDefinitions pins the tenant
// check on the write side. The FK on task_custom_field_values.field_id only
// proves that the definition row exists — it is evaluated in system context and
// never looks at tenant_id, so before this guard a caller could persist a
// definition id owned by another tenant. The row was invisible afterwards
// (GetCustomFieldValues joins the RLS-filtered definitions table) but stayed in
// the table, and success/failure of the write leaked whether a given id existed
// somewhere in the system. Foreign and non-existent ids must fail identically.
// See BACKLOG.yml unit fix-work-task-custom-field-foreign-tenant-writable.
func TestSetCustomFieldValues_RejectsForeignAndUnknownDefinitions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Task CF Guard Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Task CF Guard Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)
	taskID := seedWorkTask(t, pool, tenantOwn, projectOwn, userOwn, "CF Guard Task")
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	ownFieldName := "own-" + uuid.New().String()[:8]
	ownFieldID := testutil.SeedRow(t, pool, "work_custom_field_definitions", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       ownFieldName,
		"field_type": "text",
	})
	defer testutil.CleanupRow(t, pool, "work_custom_field_definitions", ownFieldID)

	foreignFieldID := testutil.SeedRow(t, pool, "work_custom_field_definitions", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "foreign-" + uuid.New().String()[:8],
		"field_type": "text",
	})
	defer testutil.CleanupRow(t, pool, "work_custom_field_definitions", foreignFieldID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// Positive control: the tenant's own definition still writes through.
	if err := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{ownFieldID: "ok"}); err != nil {
		t.Fatalf("SetCustomFieldValues with an own definition must succeed: %v", err)
	}

	// A definition owned by another tenant must be rejected, not silently dropped.
	foreignErr := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{foreignFieldID: "leak"})
	if !errors.Is(foreignErr, ErrCustomFieldNotFound) {
		t.Fatalf("SetCustomFieldValues with a foreign-tenant definition = %v, want ErrCustomFieldNotFound", foreignErr)
	}

	// A definition id that exists nowhere must fail the exact same way — any
	// difference between the two would restore the existence oracle.
	unknownErr := repo.SetCustomFieldValues(ctx, taskID, tenantOwn, map[uuid.UUID]any{uuid.New(): "nope"})
	if !errors.Is(unknownErr, ErrCustomFieldNotFound) {
		t.Fatalf("SetCustomFieldValues with an unknown definition = %v, want ErrCustomFieldNotFound", unknownErr)
	}
	if foreignErr.Error() != unknownErr.Error() {
		t.Fatalf("foreign (%q) and unknown (%q) must be indistinguishable to the caller",
			foreignErr.Error(), unknownErr.Error())
	}

	// Same call in system context: RLS admits every definition row there, so the
	// explicit tenant_id predicate in the EXISTS clause is the only thing left to
	// block the foreign definition. Drop it and this assertion is what goes red.
	sysWriteErr := repo.SetCustomFieldValues(testutil.WithSystemCtx(context.Background()),
		taskID, tenantOwn, map[uuid.UUID]any{foreignFieldID: "leak"})
	if !errors.Is(sysWriteErr, ErrCustomFieldNotFound) {
		t.Fatalf("SetCustomFieldValues in system context with a foreign-tenant definition = %v, want ErrCustomFieldNotFound", sysWriteErr)
	}

	// Nothing may have been persisted for the foreign definition — checked in
	// system context so RLS cannot hide a row that is actually there.
	sysCtx := testutil.WithSystemCtx(context.Background())
	var orphans int
	if err := pool.QueryRow(sysCtx,
		`SELECT COUNT(*) FROM task_custom_field_values WHERE task_id = $1 AND field_id = $2`,
		taskID, foreignFieldID).Scan(&orphans); err != nil {
		t.Fatalf("counting orphan rows: %v", err)
	}
	if orphans != 0 {
		t.Fatalf("task_custom_field_values holds %d row(s) for a foreign-tenant definition, want 0", orphans)
	}

	// The own value survived all of it.
	values, err := repo.GetCustomFieldValues(ctx, taskID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if len(values) != 1 || values[ownFieldName] != "ok" {
		t.Fatalf("got %v, want exactly {%q: \"ok\"}", values, ownFieldName)
	}
}
