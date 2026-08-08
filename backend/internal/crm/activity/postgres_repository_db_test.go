package activity

// DB integration tests for the parts of postgres_repository.go that carried
// 0% coverage: List filtering/pagination/tags, the relation-name lookups, the
// polymorphic existence checks (ContactExists/CompanyExists/DealExists/
// UserExists — the gate a Service.Create call relies on to reject activities
// linked to a foreign tenant's entity), custom fields and GetContactTimeline.
// rls_test.go and tenant_write_test.go already cover the plain CRUD RLS
// surface — this file covers the SQL those two don't reach.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRepository_List_FiltersScopesAndPaginates(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOwn2 := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-list2-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn2)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-activity-list-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Max",
		"last_name":  "Mustermann",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Acme GmbH",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn,
		"name":      fmt.Sprintf("Stage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)
	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Big Deal",
		"stage_id":   stageID,
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	callActivity := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeCall,
		Subject: "Follow up call", ContactID: &contactID, CreatedBy: userOwn,
		CreatedAt: now, UpdatedAt: now,
	}
	meetingActivity := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeMeeting,
		Subject: "Kickoff meeting", CompanyID: &companyID, IsCompleted: true,
		CreatedBy: userOwn, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}
	noteActivity := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "Internal note", DealID: &dealID, AssignedTo: &userOwn2,
		CreatedBy: userOwn2, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute),
	}
	for _, a := range []*models.Activity{callActivity, meetingActivity, noteActivity} {
		if err := repo.Create(ctxOwn, a); err != nil {
			t.Fatalf("Create %s: %v", a.Subject, err)
		}
		defer testutil.CleanupRow(t, pool, "activities", a.ID)
	}

	foreignActivity := &models.Activity{
		ID: uuid.New(), TenantID: tenantOther, ActivityType: models.ActivityTypeTask,
		Subject: "Foreign task", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOther, foreignActivity); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", foreignActivity.ID)

	// Unrestricted list scopes to the tenant: the foreign activity never leaks.
	all, allTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("List (all): %v", err)
	}
	if allTotal != 3 {
		t.Fatalf("List (all): expected total=3, got %d", allTotal)
	}
	for _, a := range all {
		if a.ID == foreignActivity.ID {
			t.Fatal("List (all) leaked a foreign-tenant activity")
		}
	}

	// ActivityType filter.
	callType := models.ActivityTypeCall
	byType, byTypeTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{ActivityType: &callType}, 0, 20)
	if err != nil {
		t.Fatalf("List (type): %v", err)
	}
	if byTypeTotal != 1 || len(byType) != 1 || byType[0].ID != callActivity.ID {
		t.Fatalf("List (type): expected exactly callActivity, got total=%d", byTypeTotal)
	}

	// ContactID filter.
	byContact, byContactTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{ContactID: &contactID}, 0, 20)
	if err != nil {
		t.Fatalf("List (contact): %v", err)
	}
	if byContactTotal != 1 || byContact[0].ID != callActivity.ID {
		t.Fatalf("List (contact): expected exactly callActivity, got total=%d", byContactTotal)
	}

	// CompanyID filter.
	byCompany, byCompanyTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{CompanyID: &companyID}, 0, 20)
	if err != nil {
		t.Fatalf("List (company): %v", err)
	}
	if byCompanyTotal != 1 || byCompany[0].ID != meetingActivity.ID {
		t.Fatalf("List (company): expected exactly meetingActivity, got total=%d", byCompanyTotal)
	}

	// DealID filter.
	byDeal, byDealTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{DealID: &dealID}, 0, 20)
	if err != nil {
		t.Fatalf("List (deal): %v", err)
	}
	if byDealTotal != 1 || byDeal[0].ID != noteActivity.ID {
		t.Fatalf("List (deal): expected exactly noteActivity, got total=%d", byDealTotal)
	}

	// AssignedTo filter.
	byAssigned, byAssignedTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{AssignedTo: &userOwn2}, 0, 20)
	if err != nil {
		t.Fatalf("List (assigned): %v", err)
	}
	if byAssignedTotal != 1 || byAssigned[0].ID != noteActivity.ID {
		t.Fatalf("List (assigned): expected exactly noteActivity, got total=%d", byAssignedTotal)
	}

	// CreatedBy filter.
	byCreator, byCreatorTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{CreatedBy: &userOwn2}, 0, 20)
	if err != nil {
		t.Fatalf("List (createdBy): %v", err)
	}
	if byCreatorTotal != 1 || byCreator[0].ID != noteActivity.ID {
		t.Fatalf("List (createdBy): expected exactly noteActivity, got total=%d", byCreatorTotal)
	}

	// IsCompleted filter.
	completed := true
	byCompleted, byCompletedTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{IsCompleted: &completed}, 0, 20)
	if err != nil {
		t.Fatalf("List (completed): %v", err)
	}
	if byCompletedTotal != 1 || byCompleted[0].ID != meetingActivity.ID {
		t.Fatalf("List (completed): expected exactly meetingActivity, got total=%d", byCompletedTotal)
	}

	// Search filter (case-insensitive subject match).
	bySearch, bySearchTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{Search: "kickoff"}, 0, 20)
	if err != nil {
		t.Fatalf("List (search): %v", err)
	}
	if bySearchTotal != 1 || bySearch[0].ID != meetingActivity.ID {
		t.Fatalf("List (search): expected exactly meetingActivity, got total=%d", bySearchTotal)
	}

	// Pagination: subject-ascending, page size 1 — "Follow up call" sorts
	// before "Internal note" and "Kickoff meeting".
	page, pageTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{SortBy: "subject", SortDesc: false}, 0, 1)
	if err != nil {
		t.Fatalf("List (page 1): %v", err)
	}
	if pageTotal != 3 || len(page) != 1 || page[0].ID != callActivity.ID {
		t.Fatalf("List (page 1): expected total=3 first=call, got total=%d first=%v", pageTotal, page)
	}

	// Offset beyond the total must return an empty page, not an error.
	empty, emptyTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{}, 50, 20)
	if err != nil {
		t.Fatalf("List (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("List (offset beyond total): expected 0 rows with total=3, got %d rows total=%d", len(empty), emptyTotal)
	}
}

func TestRepository_List_TagFilterRequiresAllTags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Tag Filter Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-tagfilter-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	doubleTagged := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "Double tagged", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	singleTagged := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "Single tagged", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	for _, a := range []*models.Activity{doubleTagged, singleTagged} {
		if err := repo.Create(ctxOwn, a); err != nil {
			t.Fatalf("Create %s: %v", a.Subject, err)
		}
		defer testutil.CleanupRow(t, pool, "activities", a.ID)
	}

	tagA := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("Urgent-%s", uuid.New().String()[:8]),
		"entity_type": "activity",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagA)
	tagB := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("VIP-%s", uuid.New().String()[:8]),
		"entity_type": "activity",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagB)

	if err := repo.AddTags(ctxOwn, doubleTagged.ID, []uuid.UUID{tagA, tagB}); err != nil {
		t.Fatalf("AddTags (double): %v", err)
	}
	if err := repo.AddTags(ctxOwn, singleTagged.ID, []uuid.UUID{tagA}); err != nil {
		t.Fatalf("AddTags (single): %v", err)
	}

	// Both tags required (AND semantics) — only the double-tagged activity matches.
	results, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{TagIDs: []uuid.UUID{tagA, tagB}}, 0, 20)
	if err != nil {
		t.Fatalf("List (tags): %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != doubleTagged.ID {
		t.Fatalf("List (tags): expected exactly doubleTagged, got total=%d", total)
	}
}

func TestRepository_GetRelationNames_FoundAndNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Names Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"first_name":    "Erika",
		"last_name":     "Musterfrau",
		"email":         fmt.Sprintf("crm-activity-names-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Max",
		"last_name":  "Mustermann",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Acme GmbH",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn,
		"name":      fmt.Sprintf("Stage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)
	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Big Deal",
		"stage_id":   stageID,
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if name, err := repo.GetContactName(ctxOwn, contactID); err != nil || name != "Max Mustermann" {
		t.Fatalf("GetContactName (found): name=%q err=%v", name, err)
	}
	if name, err := repo.GetContactName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetContactName (not found): expected empty name and nil error, got name=%q err=%v", name, err)
	}

	if name, err := repo.GetCompanyName(ctxOwn, companyID); err != nil || name != "Acme GmbH" {
		t.Fatalf("GetCompanyName (found): name=%q err=%v", name, err)
	}
	if name, err := repo.GetCompanyName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetCompanyName (not found): expected empty name and nil error, got name=%q err=%v", name, err)
	}

	if name, err := repo.GetDealName(ctxOwn, dealID); err != nil || name != "Big Deal" {
		t.Fatalf("GetDealName (found): name=%q err=%v", name, err)
	}
	if name, err := repo.GetDealName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetDealName (not found): expected empty name and nil error, got name=%q err=%v", name, err)
	}

	if name, err := repo.GetUserName(ctxOwn, userOwn); err != nil || name != "Erika Musterfrau" {
		t.Fatalf("GetUserName (found): name=%q err=%v", name, err)
	}
	if name, err := repo.GetUserName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetUserName (not found): expected empty name and nil error, got name=%q err=%v", name, err)
	}
}

// TestExistenceChecks_AreTenantScopedByRLS proves the property the "activity
// linked to a foreign entity" gap depends on: ContactExists/CompanyExists/
// DealExists/UserExists carry no explicit tenant_id predicate in their SQL —
// RLS on the target tables is the only thing making a foreign-tenant row
// invisible. If RLS on any of these four tables were ever relaxed, this test
// (not just the SELECT policy) would be the thing that goes red.
func TestExistenceChecks_AreTenantScopedByRLS(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Exists Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity Exists Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-activity-exists-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)
	contactOther := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Contact",
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactOther)
	companyOther := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign GmbH",
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyOther)
	stageOther := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther,
		"name":      fmt.Sprintf("Stage-Foreign-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOther)
	dealOther := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign Deal",
		"stage_id":   stageOther,
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// From the owning tenant's session, every row it just seeded is visible.
	if exists, err := repo.ContactExists(ctxOther, contactOther); err != nil || !exists {
		t.Fatalf("ContactExists (own tenant): exists=%v err=%v", exists, err)
	}
	if exists, err := repo.CompanyExists(ctxOther, companyOther); err != nil || !exists {
		t.Fatalf("CompanyExists (own tenant): exists=%v err=%v", exists, err)
	}
	if exists, err := repo.DealExists(ctxOther, dealOther); err != nil || !exists {
		t.Fatalf("DealExists (own tenant): exists=%v err=%v", exists, err)
	}
	if exists, err := repo.UserExists(ctxOther, userOther); err != nil || !exists {
		t.Fatalf("UserExists (own tenant): exists=%v err=%v", exists, err)
	}

	// From a *different* tenant's session, the same rows must appear
	// nonexistent — RLS hides them, the query has no other guard.
	if exists, err := repo.ContactExists(ctxOwn, contactOther); err != nil || exists {
		t.Fatalf("ContactExists (foreign tenant): expected false, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.CompanyExists(ctxOwn, companyOther); err != nil || exists {
		t.Fatalf("CompanyExists (foreign tenant): expected false, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.DealExists(ctxOwn, dealOther); err != nil || exists {
		t.Fatalf("DealExists (foreign tenant): expected false, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.UserExists(ctxOwn, userOther); err != nil || exists {
		t.Fatalf("UserExists (foreign tenant): expected false, got exists=%v err=%v", exists, err)
	}

	// A random UUID is never found, own tenant or not.
	if exists, err := repo.ContactExists(ctxOwn, uuid.New()); err != nil || exists {
		t.Fatalf("ContactExists (random uuid): expected false, got exists=%v err=%v", exists, err)
	}
}

// TestService_Create_RejectsActivityLinkedToForeignTenantEntity is the
// integration proof for the done_when criterion "Aktivitaet an fremder
// Entitaet wird abgelehnt": Service.Create's pre-checks call ContactExists/
// CompanyExists/DealExists with no tenant_id argument at all, so this only
// works because RLS makes the foreign row invisible to the caller's session.
func TestService_Create_RejectsActivityLinkedToForeignTenantEntity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Create Guard Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity Create Guard Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-guard-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-activity-guard-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)
	contactOther := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Contact",
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactOther)
	companyOther := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign GmbH",
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyOther)
	stageOther := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther,
		"name":      fmt.Sprintf("Stage-Guard-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOther)
	dealOther := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign Deal",
		"stage_id":   stageOther,
		"created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealOther)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if _, err := svc.Create(ctxOwn, CreateInput{
		TenantID: tenantOwn, ActivityType: models.ActivityTypeCall, Subject: "Sneaky",
		ContactID: &contactOther, CreatedBy: userOwn,
	}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("Create (foreign contact): expected ErrContactNotFound, got %v", err)
	}

	if _, err := svc.Create(ctxOwn, CreateInput{
		TenantID: tenantOwn, ActivityType: models.ActivityTypeCall, Subject: "Sneaky",
		CompanyID: &companyOther, CreatedBy: userOwn,
	}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("Create (foreign company): expected ErrCompanyNotFound, got %v", err)
	}

	if _, err := svc.Create(ctxOwn, CreateInput{
		TenantID: tenantOwn, ActivityType: models.ActivityTypeCall, Subject: "Sneaky",
		DealID: &dealOther, CreatedBy: userOwn,
	}); !errors.Is(err, ErrDealNotFound) {
		t.Fatalf("Create (foreign deal): expected ErrDealNotFound, got %v", err)
	}
}

func TestRepository_Tags_AddGetRemoveRoundtripAndTagExistsIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Tags Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity Tags Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-tags-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	a := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "Tag roundtrip", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", a.ID)

	tagOwn := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("Own-Tag-%s", uuid.New().String()[:8]),
		"entity_type": "activity",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagOwn)
	tagOther := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   tenantOther,
		"name":        fmt.Sprintf("Other-Tag-%s", uuid.New().String()[:8]),
		"entity_type": "activity",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagOther)

	// TagExists has no tenant_id predicate in its SQL — RLS is the only guard.
	if exists, err := repo.TagExists(ctxOwn, tagOwn, models.EntityTypeActivity); err != nil || !exists {
		t.Fatalf("TagExists (own tag): exists=%v err=%v", exists, err)
	}
	if exists, err := repo.TagExists(ctxOwn, tagOther, models.EntityTypeActivity); err != nil || exists {
		t.Fatalf("TagExists (foreign tag): expected false, got exists=%v err=%v", exists, err)
	}

	if err := repo.AddTags(ctxOwn, a.ID, []uuid.UUID{tagOwn}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}
	tags, err := repo.GetTags(ctxOwn, a.ID)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tagOwn {
		t.Fatalf("GetTags: expected exactly [tagOwn], got %v", tags)
	}

	// A foreign-tenant session cannot see the tag link either.
	foreignView, err := repo.GetTags(ctxOther, a.ID)
	if err != nil {
		t.Fatalf("GetTags (foreign ctx): %v", err)
	}
	if len(foreignView) != 0 {
		t.Fatalf("GetTags (foreign ctx): expected no tags visible, got %v", foreignView)
	}

	if err := repo.RemoveTags(ctxOwn, a.ID, []uuid.UUID{tagOwn}); err != nil {
		t.Fatalf("RemoveTags: %v", err)
	}
	tagsAfterRemove, err := repo.GetTags(ctxOwn, a.ID)
	if err != nil {
		t.Fatalf("GetTags (after remove): %v", err)
	}
	if len(tagsAfterRemove) != 0 {
		t.Fatalf("GetTags (after remove): expected no tags, got %v", tagsAfterRemove)
	}
}

func TestRepository_CustomFields_SetGetRoundtripUpsertAndEmptyReturnsNoRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Custom Fields Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-cf-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	a := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "Custom field roundtrip", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", a.ID)

	empty := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeNote,
		Subject: "No custom fields", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, empty); err != nil {
		t.Fatalf("Create (empty): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", empty.ID)

	// custom_field_definitions.tenant_id defaults to the system tenant when
	// unset — this must be explicit, or the field lands invisibly outside
	// tenantOwn's RLS view (the fixture trap iteration 30 hit on company_tags).
	fieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   tenantOwn,
		"entity_type": "activity",
		"field_name":  fmt.Sprintf("priority_%s", uuid.New().String()[:8]),
		"field_label": "Priority",
		"field_type":  "text",
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", fieldID)

	if err := repo.SetCustomFieldValues(ctxOwn, a.ID, map[uuid.UUID]any{fieldID: "high"}); err != nil {
		t.Fatalf("SetCustomFieldValues: %v", err)
	}
	values, err := repo.GetCustomFieldValues(ctxOwn, a.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if len(values) != 1 || values[0].Value != "high" {
		t.Fatalf("GetCustomFieldValues: expected [priority=high], got %v", values)
	}

	// Upsert overwrite: setting again with a new value replaces, not duplicates.
	if err := repo.SetCustomFieldValues(ctxOwn, a.ID, map[uuid.UUID]any{fieldID: "low"}); err != nil {
		t.Fatalf("SetCustomFieldValues (overwrite): %v", err)
	}
	valuesAfter, err := repo.GetCustomFieldValues(ctxOwn, a.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (after overwrite): %v", err)
	}
	if len(valuesAfter) != 1 || valuesAfter[0].Value != "low" {
		t.Fatalf("GetCustomFieldValues (after overwrite): expected [priority=low], got %v", valuesAfter)
	}

	// An activity that never had a value set returns an empty slice, not an error.
	emptyValues, err := repo.GetCustomFieldValues(ctxOwn, empty.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (empty): %v", err)
	}
	if len(emptyValues) != 0 {
		t.Fatalf("GetCustomFieldValues (empty): expected no rows, got %v", emptyValues)
	}
}

func TestRepository_GetContactTimeline_CombinesActivitiesAndDealsScopedAndPaginated(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Timeline Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity Timeline Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-timeline-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-activity-timeline-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Timeline",
		"last_name":  "Contact",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	stageOwn := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn,
		"name":      fmt.Sprintf("Stage-TL-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOwn)
	stageOther := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther,
		"name":      fmt.Sprintf("Stage-TL-Foreign-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	act1 := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeCall,
		Subject: "First call", ContactID: &contactID, CreatedBy: userOwn,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, act1); err != nil {
		t.Fatalf("Create act1: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", act1.ID)

	act2 := &models.Activity{
		ID: uuid.New(), TenantID: tenantOwn, ActivityType: models.ActivityTypeMeeting,
		Subject: "Second meeting", ContactID: &contactID, CreatedBy: userOwn,
		CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}
	if err := repo.Create(ctxOwn, act2); err != nil {
		t.Fatalf("Create act2: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", act2.ID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Timeline Deal",
		"stage_id":   stageOwn,
		"contact_id": contactID,
		"created_by": userOwn,
		"created_at": now.Add(2 * time.Minute),
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	// A foreign-tenant deal referencing the same contact_id — a data-integrity
	// accident no code path in this repo would produce, but the query's own
	// tenant_id predicate (not just RLS) must still exclude it.
	foreignDealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign Timeline Deal",
		"stage_id":   stageOther,
		"contact_id": contactID,
		"created_by": userOther,
		"created_at": now.Add(3 * time.Minute),
	})
	defer testutil.CleanupRow(t, pool, "deals", foreignDealID)
	testutil.AssertRowCount(t, pool, sysCtx, "deals", foreignDealID, 1)

	events, total, err := repo.GetContactTimeline(ctxOwn, contactID, tenantOwn, 0, 20)
	if err != nil {
		t.Fatalf("GetContactTimeline: %v", err)
	}
	if total != 3 {
		t.Fatalf("GetContactTimeline: expected total=3 (2 activities + 1 own deal), got %d", total)
	}
	for _, e := range events {
		if e.ID == foreignDealID {
			t.Fatal("GetContactTimeline leaked a foreign-tenant deal")
		}
	}
	// Most recent first: the own-tenant deal link (t+2m) precedes act2 (t+1m).
	if len(events) != 3 || events[0].ID != dealID || events[0].EventType != "deal_linked" {
		t.Fatalf("GetContactTimeline: expected newest-first with deal_linked head, got %+v", events)
	}

	// Pagination: page size 1 returns just the newest event, total unchanged.
	page, pageTotal, err := repo.GetContactTimeline(ctxOwn, contactID, tenantOwn, 0, 1)
	if err != nil {
		t.Fatalf("GetContactTimeline (page): %v", err)
	}
	if pageTotal != 3 || len(page) != 1 || page[0].ID != dealID {
		t.Fatalf("GetContactTimeline (page): expected total=3 first=deal, got total=%d first=%v", pageTotal, page)
	}

	// An attacker session passing the *real* owning tenantID as the explicit
	// argument sees nothing — only RLS on the attacker's own session can be
	// stopping this, since the query's tenant_id predicate alone would match.
	stolenEvents, stolenTotal, err := repo.GetContactTimeline(ctxOther, contactID, tenantOwn, 0, 20)
	if err != nil {
		t.Fatalf("GetContactTimeline (stolen tenantID): %v", err)
	}
	if stolenTotal != 0 || len(stolenEvents) != 0 {
		t.Fatalf("GetContactTimeline (stolen tenantID): expected no events, got total=%d events=%v", stolenTotal, stolenEvents)
	}
}
