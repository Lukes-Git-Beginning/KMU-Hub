package contact

// DB integration tests for the parts of postgres_repository.go that rls_test.go
// and tenant_write_test.go don't reach: List/ListWithVisibility filtering,
// sorting and pagination (with the count-carries-the-same-tenant-condition
// property), ListByIDs/ListAll, company name lookups, tags, custom fields,
// duplicate detection and the merge transaction. postgres_lead.go has its own
// file, postgres_lead_db_test.go.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRepository_List_FiltersSortsAndPaginatesTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-contact-list-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	alice := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Alice", LastName: "Ackerman", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	bob := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Bob", LastName: "Baumann", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	carol := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Carol", LastName: "Search-Match", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	foreign := &models.Contact{ID: uuid.New(), TenantID: tenantOther, FirstName: "Search-Match", LastName: "Foreign", Visibility: "shared", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Contact{alice, bob, carol} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.FirstName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", foreign.ID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("VIP-%s", uuid.New().String()[:8]),
		"entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)
	if err := repo.AddTags(ctxOwn, alice.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}

	// Search matches "search-match" in either first_name or last_name and
	// scopes to the tenant: our carol matches, the foreign contact (same
	// substring, different tenant) must not appear.
	searchResults, searchTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, Search: "search-match"}, 0, 20)
	if err != nil {
		t.Fatalf("List (search): %v", err)
	}
	if searchTotal != 1 || len(searchResults) != 1 || searchResults[0].ID != carol.ID {
		t.Fatalf("List (search): expected exactly carol, got total=%d results=%v", searchTotal, searchResults)
	}

	// Tag filter.
	tagResults, tagTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, TagIDs: []uuid.UUID{tagID}}, 0, 20)
	if err != nil {
		t.Fatalf("List (tag): %v", err)
	}
	if tagTotal != 1 || len(tagResults) != 1 || tagResults[0].ID != alice.ID {
		t.Fatalf("List (tag): expected exactly alice, got total=%d results=%d", tagTotal, len(tagResults))
	}

	// Sorting: first_name ascending across the full tenant set (3 rows).
	asc, ascTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, SortBy: "first_name", SortDesc: false}, 0, 20)
	if err != nil {
		t.Fatalf("List (sort asc): %v", err)
	}
	if ascTotal != 3 || len(asc) != 3 || asc[0].ID != alice.ID || asc[2].ID != carol.ID {
		t.Fatalf("List (sort asc): expected alice..carol ascending, got %v", asc)
	}
	desc, _, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, SortBy: "first_name", SortDesc: true}, 0, 20)
	if err != nil {
		t.Fatalf("List (sort desc): %v", err)
	}
	if len(desc) != 3 || desc[0].ID != carol.ID || desc[2].ID != alice.ID {
		t.Fatalf("List (sort desc): expected carol..alice descending, got %v", desc)
	}

	// Pagination: page size 1 lands on the alphabetically first row, but the
	// total must still reflect the full tenant-scoped count (3), not the page
	// size — this is the "count carries the same tenant condition as the
	// page" property the unit calls out explicitly.
	page, total, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, SortBy: "first_name"}, 0, 1)
	if err != nil {
		t.Fatalf("List (page 1): %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ID != alice.ID {
		t.Fatalf("List (page 1): expected total=3 first=alice, got total=%d page=%v", total, page)
	}

	// Offset beyond the total returns an empty page, not an error, and the
	// total is unchanged.
	empty, emptyTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn}, 50, 20)
	if err != nil {
		t.Fatalf("List (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("List (offset beyond total): expected 0 rows with total=3, got %d rows total=%d", len(empty), emptyTotal)
	}

	// A foreign-tenant caller never sees these rows regardless of filter.
	foreignView, foreignTotal, err := repo.List(ctxOther, ListFilter{TenantID: tenantOther}, 0, 20)
	if err != nil {
		t.Fatalf("List (foreign tenant): %v", err)
	}
	if foreignTotal != 1 || len(foreignView) != 1 || foreignView[0].ID != foreign.ID {
		t.Fatalf("List (foreign tenant): expected exactly the foreign contact, got total=%d results=%v", foreignTotal, foreignView)
	}
}

func TestRepository_ListWithVisibility_OwnerAdminAndVisibilityFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Visibility Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-vis-a-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-vis-b-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	shared := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Shared", LastName: "Contact", Visibility: "shared", OwnerID: &userA, CreatedBy: userA, CreatedAt: now, UpdatedAt: now}
	personalA := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "PersonalA", LastName: "Contact", Visibility: "personal", OwnerID: &userA, CreatedBy: userA, CreatedAt: now, UpdatedAt: now}
	personalB := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "PersonalB", LastName: "Contact", Visibility: "personal", OwnerID: &userB, CreatedBy: userB, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Contact{shared, personalA, personalB} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.FirstName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}

	// userA (non-admin): shared + own personal, not userB's personal.
	viewA, totalA, err := repo.ListWithVisibility(ctxOwn, userA, false, ListFilter{TenantID: tenantOwn}, 0, 20)
	if err != nil {
		t.Fatalf("ListWithVisibility (userA): %v", err)
	}
	idsA := make(map[uuid.UUID]bool, len(viewA))
	for _, c := range viewA {
		idsA[c.ID] = true
	}
	if totalA != 2 || !idsA[shared.ID] || !idsA[personalA.ID] || idsA[personalB.ID] {
		t.Fatalf("ListWithVisibility (userA): expected shared+personalA only, got total=%d ids=%v", totalA, idsA)
	}

	// userB (non-admin): shared + own personal, not userA's personal.
	viewB, totalB, err := repo.ListWithVisibility(ctxOwn, userB, false, ListFilter{TenantID: tenantOwn}, 0, 20)
	if err != nil {
		t.Fatalf("ListWithVisibility (userB): %v", err)
	}
	idsB := make(map[uuid.UUID]bool, len(viewB))
	for _, c := range viewB {
		idsB[c.ID] = true
	}
	if totalB != 2 || !idsB[shared.ID] || !idsB[personalB.ID] || idsB[personalA.ID] {
		t.Fatalf("ListWithVisibility (userB): expected shared+personalB only, got total=%d ids=%v", totalB, idsB)
	}

	// Admin sees everything regardless of ownership.
	viewAdmin, totalAdmin, err := repo.ListWithVisibility(ctxOwn, userB, true, ListFilter{TenantID: tenantOwn}, 0, 20)
	if err != nil {
		t.Fatalf("ListWithVisibility (admin): %v", err)
	}
	if totalAdmin != 3 || len(viewAdmin) != 3 {
		t.Fatalf("ListWithVisibility (admin): expected all 3 rows, got total=%d results=%d", totalAdmin, len(viewAdmin))
	}

	// VisibilityFilter narrows further: "personal" for userA yields only
	// personalA, even though userA is otherwise entitled to see "shared" too.
	filtered, filteredTotal, err := repo.ListWithVisibility(ctxOwn, userA, false, ListFilter{TenantID: tenantOwn, VisibilityFilter: "personal"}, 0, 20)
	if err != nil {
		t.Fatalf("ListWithVisibility (visibility filter): %v", err)
	}
	if filteredTotal != 1 || len(filtered) != 1 || filtered[0].ID != personalA.ID {
		t.Fatalf("ListWithVisibility (visibility filter): expected exactly personalA, got total=%d results=%v", filteredTotal, filtered)
	}
}

func TestRepository_ListByIDsAndListAll_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact ByIDs Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact ByIDs Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-byids-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-contact-byids-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	own := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Own", LastName: "Contact", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, own); err != nil {
		t.Fatalf("Create own: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", own.ID)
	foreign := &models.Contact{ID: uuid.New(), TenantID: tenantOther, FirstName: "Foreign", LastName: "Contact", Visibility: "shared", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", foreign.ID)

	// ListByIDs with an empty slice must short-circuit without a query.
	if got, err := repo.ListByIDs(ctxOwn, nil, tenantOwn); err != nil || got != nil {
		t.Fatalf("ListByIDs (empty): expected (nil, nil), got (%v, %v)", got, err)
	}

	// A mixed id list (own + foreign + nonexistent) scoped to tenantOwn
	// returns only the own-tenant row.
	mixed, err := repo.ListByIDs(ctxOwn, []uuid.UUID{own.ID, foreign.ID, uuid.New()}, tenantOwn)
	if err != nil {
		t.Fatalf("ListByIDs (mixed): %v", err)
	}
	if len(mixed) != 1 || mixed[0].ID != own.ID {
		t.Fatalf("ListByIDs (mixed): expected exactly own, got %v", mixed)
	}

	// ListAll respects the tenant boundary the same way List does.
	all, err := repo.ListAll(ctxOwn, userOwn, false, tenantOwn)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 || all[0].ID != own.ID {
		t.Fatalf("ListAll: expected exactly own, got %v", all)
	}
	allForeign, err := repo.ListAll(ctxOther, userOther, false, tenantOther)
	if err != nil {
		t.Fatalf("ListAll (foreign): %v", err)
	}
	if len(allForeign) != 1 || allForeign[0].ID != foreign.ID {
		t.Fatalf("ListAll (foreign): expected exactly foreign, got %v", allForeign)
	}
}

func TestRepository_CompanyNameLookups_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact CompanyName Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact CompanyName Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-coname-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-contact-coname-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	ownCompany := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": "Own Co GmbH", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", ownCompany)
	foreignCompany := testutil.SeedRow(t, pool, "companies", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOther, "name": "Foreign Co GmbH", "created_by": userOther,
	})
	defer testutil.CleanupRow(t, pool, "companies", foreignCompany)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	name, err := repo.GetCompanyName(ctxOwn, ownCompany, tenantOwn)
	if err != nil {
		t.Fatalf("GetCompanyName (own): %v", err)
	}
	if name != "Own Co GmbH" {
		t.Fatalf("GetCompanyName (own): got %q", name)
	}

	// A company id that exists but belongs to a different tenant is
	// indistinguishable from "does not exist" -- empty string, no error.
	foreignName, err := repo.GetCompanyName(ctxOwn, foreignCompany, tenantOwn)
	if err != nil {
		t.Fatalf("GetCompanyName (foreign): unexpected error %v", err)
	}
	if foreignName != "" {
		t.Fatalf("GetCompanyName (foreign): expected empty string, got %q", foreignName)
	}

	names, err := repo.GetCompanyNames(ctxOwn, []uuid.UUID{ownCompany, foreignCompany}, tenantOwn)
	if err != nil {
		t.Fatalf("GetCompanyNames: %v", err)
	}
	if len(names) != 1 || names[ownCompany] != "Own Co GmbH" {
		t.Fatalf("GetCompanyNames: expected only own company, got %v", names)
	}
	if _, ok := names[foreignCompany]; ok {
		t.Fatal("GetCompanyNames: leaked a foreign-tenant company")
	}

	if empty, err := repo.GetCompanyNames(ctxOwn, nil, tenantOwn); err != nil || empty != nil {
		t.Fatalf("GetCompanyNames (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}
}

func TestRepository_Tags_AddGetRemoveAndBatchRoundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Tags Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-tags-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c1 := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Tag", LastName: "One", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	c2 := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Tag", LastName: "Two", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Contact{c1, c2} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.LastName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("Round-%s", uuid.New().String()[:8]),
		"entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	if err := repo.AddTags(ctxOwn, c1.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags c1: %v", err)
	}
	if err := repo.AddTags(ctxOwn, c2.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags c2: %v", err)
	}
	// AddTags with an empty slice must be a no-op, not an error.
	if err := repo.AddTags(ctxOwn, c1.ID, nil); err != nil {
		t.Fatalf("AddTags (empty): %v", err)
	}

	tags, err := repo.GetTags(ctxOwn, c1.ID)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tagID {
		t.Fatalf("GetTags: expected exactly [%s], got %v", tagID, tags)
	}

	batch, err := repo.GetTagsBatch(ctxOwn, []uuid.UUID{c1.ID, c2.ID})
	if err != nil {
		t.Fatalf("GetTagsBatch: %v", err)
	}
	if len(batch[c1.ID]) != 1 || len(batch[c2.ID]) != 1 {
		t.Fatalf("GetTagsBatch: expected one tag per contact, got %v", batch)
	}
	if empty, err := repo.GetTagsBatch(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetTagsBatch (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}

	if err := repo.RemoveTags(ctxOwn, c1.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("RemoveTags: %v", err)
	}
	afterRemove, err := repo.GetTags(ctxOwn, c1.ID)
	if err != nil {
		t.Fatalf("GetTags (after remove): %v", err)
	}
	if len(afterRemove) != 0 {
		t.Fatalf("GetTags (after remove): expected none, got %v", afterRemove)
	}
	// c2 must be unaffected by c1's removal.
	c2Tags, err := repo.GetTags(ctxOwn, c2.ID)
	if err != nil {
		t.Fatalf("GetTags (c2): %v", err)
	}
	if len(c2Tags) != 1 {
		t.Fatalf("GetTags (c2): expected the tag to remain, got %v", c2Tags)
	}
}

func TestRepository_TagExists_TenantScopedByEntityType(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact TagExists Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact TagExists Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	ownTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Exists-Own-%s", uuid.New().String()[:8]), "entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", ownTag)
	foreignTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOther, "name": fmt.Sprintf("Exists-Foreign-%s", uuid.New().String()[:8]), "entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", foreignTag)
	dealTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Exists-Deal-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", dealTag)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if exists, err := repo.TagExists(ctxOwn, ownTag, models.EntityTypeContact); err != nil || !exists {
		t.Fatalf("TagExists (own): expected true, got exists=%v err=%v", exists, err)
	}
	// RLS is the only thing standing between "own tenant" and "any tenant" --
	// the query itself carries no tenant_id predicate.
	if exists, err := repo.TagExists(ctxOwn, foreignTag, models.EntityTypeContact); err != nil {
		t.Fatalf("TagExists (foreign): %v", err)
	} else if exists {
		t.Fatal("TagExists (foreign): a foreign-tenant tag was visible")
	}
	// Right tenant, wrong entity_type.
	if exists, err := repo.TagExists(ctxOwn, dealTag, models.EntityTypeContact); err != nil {
		t.Fatalf("TagExists (wrong entity type): %v", err)
	} else if exists {
		t.Fatal("TagExists (wrong entity type): a deal tag matched the contact entity type")
	}
}

func TestRepository_CustomFieldValues_SetGetRoundtripAndBatch(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact CustomFields Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-cf-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	c1 := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "CF", LastName: "One", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	c2 := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "CF", LastName: "Two", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Contact{c1, c2} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.LastName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}

	// A contact without any values returns an empty slice, not nil-vs-error
	// ambiguity.
	empty, err := repo.GetCustomFieldValues(ctxOwn, c1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetCustomFieldValues (empty): expected none, got %v", empty)
	}
	// SetCustomFieldValues with an empty map is a no-op, not an error.
	if err := repo.SetCustomFieldValues(ctxOwn, c1.ID, nil); err != nil {
		t.Fatalf("SetCustomFieldValues (empty): %v", err)
	}

	segment := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "entity_type": "contact",
		"field_name": fmt.Sprintf("segment_%s", uuid.New().String()[:8]), "field_label": "Segment", "field_type": "text", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", segment)

	if err := repo.SetCustomFieldValues(ctxOwn, c1.ID, map[uuid.UUID]any{segment: "enterprise"}); err != nil {
		t.Fatalf("SetCustomFieldValues c1: %v", err)
	}
	if err := repo.SetCustomFieldValues(ctxOwn, c2.ID, map[uuid.UUID]any{segment: "smb"}); err != nil {
		t.Fatalf("SetCustomFieldValues c2: %v", err)
	}

	values, err := repo.GetCustomFieldValues(ctxOwn, c1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if len(values) != 1 || values[0].Value != "enterprise" {
		t.Fatalf("GetCustomFieldValues: expected enterprise, got %v", values)
	}

	// Upsert semantics: setting again overwrites, does not duplicate.
	if err := repo.SetCustomFieldValues(ctxOwn, c1.ID, map[uuid.UUID]any{segment: "startup"}); err != nil {
		t.Fatalf("SetCustomFieldValues (overwrite): %v", err)
	}
	afterOverwrite, err := repo.GetCustomFieldValues(ctxOwn, c1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (after overwrite): %v", err)
	}
	if len(afterOverwrite) != 1 || afterOverwrite[0].Value != "startup" {
		t.Fatalf("GetCustomFieldValues (after overwrite): expected startup, got %v", afterOverwrite)
	}

	batch, err := repo.GetCustomFieldValuesBatch(ctxOwn, []uuid.UUID{c1.ID, c2.ID})
	if err != nil {
		t.Fatalf("GetCustomFieldValuesBatch: %v", err)
	}
	if len(batch[c1.ID]) != 1 || batch[c1.ID][0].Value != "startup" {
		t.Fatalf("GetCustomFieldValuesBatch: c1 mismatch, got %v", batch[c1.ID])
	}
	if len(batch[c2.ID]) != 1 || batch[c2.ID][0].Value != "smb" {
		t.Fatalf("GetCustomFieldValuesBatch: c2 mismatch, got %v", batch[c2.ID])
	}
	if empty, err := repo.GetCustomFieldValuesBatch(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetCustomFieldValuesBatch (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}
}

// TestRepository_FindDuplicateCandidates_PhoneFuzzyExcludeMergedAndForeignTenant
// covers the phone_exact and name_fuzzy branches plus the merged/foreign-tenant
// exclusions. It deliberately does NOT exercise email_exact: idx_contacts_email
// (migration 000007) is a global unique index on LOWER(email) with no tenant_id
// in its definition, so two contacts anywhere in the system -- not just this
// tenant -- can never share an email, and repo.Create enforces the same via
// GetByEmail. That makes the email_exact branch in FindDuplicateCandidates
// unreachable through any current write path; worth a look for Luke (schema
// question, not a coverage gap), out of scope for this unit to change.
func TestRepository_FindDuplicateCandidates_PhoneFuzzyExcludeMergedAndForeignTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Dup Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact Dup Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-dup-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-contact-dup-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	phone := fmt.Sprintf("+49-%s", uuid.New().String()[:8])

	src := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Heinrich", LastName: "Baumann", Phone: &phone, Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, src); err != nil {
		t.Fatalf("Create src: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", src.ID)

	phoneMatch := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Also", LastName: "Different", Phone: &phone, Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, phoneMatch); err != nil {
		t.Fatalf("Create phoneMatch: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", phoneMatch.ID)

	nameFuzzy := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Heinrich", LastName: "Baumannn", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, nameFuzzy); err != nil {
		t.Fatalf("Create nameFuzzy: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", nameFuzzy.ID)

	// Already merged: shares the exact phone but must not surface.
	merged := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Merged", LastName: "Away", Phone: &phone, Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, merged); err != nil {
		t.Fatalf("Create merged: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", merged.ID)
	if _, err := pool.Exec(ctxOwn, `UPDATE contacts SET merged_into_id = $1 WHERE id = $2`, src.ID, merged.ID); err != nil {
		t.Fatalf("mark merged: %v", err)
	}

	// Foreign tenant, same phone — must never appear.
	foreign := &models.Contact{ID: uuid.New(), TenantID: tenantOther, FirstName: "Foreign", LastName: "Duplicate", Phone: &phone, Visibility: "shared", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", foreign.ID)

	candidates, err := repo.FindDuplicateCandidates(ctxOwn, src.ID, tenantOwn)
	if err != nil {
		t.Fatalf("FindDuplicateCandidates: %v", err)
	}

	byID := make(map[uuid.UUID]*DuplicateCandidate, len(candidates))
	for _, c := range candidates {
		byID[c.Contact.ID] = c
	}
	if c, ok := byID[phoneMatch.ID]; !ok || c.MatchType != "phone_exact" {
		t.Fatalf("expected phone_exact candidate for %s, got %+v", phoneMatch.ID, byID[phoneMatch.ID])
	}
	if c, ok := byID[nameFuzzy.ID]; !ok || c.MatchType != "name_fuzzy" {
		t.Fatalf("expected name_fuzzy candidate for %s, got %+v", nameFuzzy.ID, byID[nameFuzzy.ID])
	}
	if _, ok := byID[merged.ID]; ok {
		t.Fatal("FindDuplicateCandidates surfaced an already-merged contact")
	}
	if _, ok := byID[foreign.ID]; ok {
		t.Fatal("FindDuplicateCandidates leaked a foreign-tenant contact")
	}
}

// TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes
// mirrors the equivalent company test: activities/deals reassigned to the
// primary, tags and custom fields merged onto it, duplicate soft-deleted via
// merged_into_id, and the cross-tenant guard at the top of MergeInto.
func TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Merge Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact Merge Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-merge-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-contact-merge-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	primary := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Primary", LastName: "Contact", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	duplicate := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Duplicate", LastName: "Contact", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Contact{primary, duplicate} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.FirstName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}

	// Cross-tenant guard: a duplicateID belonging to a foreign tenant must
	// error out, not merge/leak.
	foreignContact := &models.Contact{ID: uuid.New(), TenantID: tenantOther, FirstName: "Foreign", LastName: "Contact", Visibility: "shared", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(testutil.WithTenantCtx(context.Background(), tenantOther), foreignContact); err != nil {
		t.Fatalf("Create foreignContact: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", foreignContact.ID)
	if err := repo.MergeInto(ctxOwn, primary.ID, foreignContact.ID, tenantOwn); err == nil {
		t.Fatal("MergeInto: expected an error merging a foreign-tenant duplicate, got nil")
	}

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Merge-Stage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)
	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "activity_type": "note", "subject": "Merge test activity",
		"contact_id": duplicate.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)
	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": "Moved Deal", "stage_id": stageID,
		"contact_id": duplicate.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	dupTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Merge-Tag-%s", uuid.New().String()[:8]), "entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", dupTag)
	if err := repo.AddTags(ctxOwn, duplicate.ID, []uuid.UUID{dupTag}); err != nil {
		t.Fatalf("AddTags (duplicate): %v", err)
	}

	cfDef := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "entity_type": "contact", "field_name": fmt.Sprintf("merge_field_%s", uuid.New().String()[:8]),
		"field_label": "Merge Field", "field_type": "text", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", cfDef)
	if err := repo.SetCustomFieldValues(ctxOwn, duplicate.ID, map[uuid.UUID]any{cfDef: "carried-over"}); err != nil {
		t.Fatalf("SetCustomFieldValues (duplicate): %v", err)
	}

	if err := repo.MergeInto(ctxOwn, primary.ID, duplicate.ID, tenantOwn); err != nil {
		t.Fatalf("MergeInto: %v", err)
	}

	var activityContact, dealContact uuid.UUID
	if err := pool.QueryRow(sysCtx, `SELECT contact_id FROM activities WHERE id = $1`, activityID).Scan(&activityContact); err != nil {
		t.Fatalf("read activity after merge: %v", err)
	}
	if activityContact != primary.ID {
		t.Fatalf("activity not reassigned: contact_id=%s, want %s", activityContact, primary.ID)
	}
	if err := pool.QueryRow(sysCtx, `SELECT contact_id FROM deals WHERE id = $1`, dealID).Scan(&dealContact); err != nil {
		t.Fatalf("read deal after merge: %v", err)
	}
	if dealContact != primary.ID {
		t.Fatalf("deal not reassigned: contact_id=%s, want %s", dealContact, primary.ID)
	}

	primaryTags, err := repo.GetTags(ctxOwn, primary.ID)
	if err != nil {
		t.Fatalf("GetTags (primary after merge): %v", err)
	}
	found := false
	for _, tag := range primaryTags {
		if tag.ID == dupTag {
			found = true
		}
	}
	if !found {
		t.Fatalf("MergeInto did not carry the duplicate's tag onto the primary, got %v", primaryTags)
	}

	primaryValues, err := repo.GetCustomFieldValues(ctxOwn, primary.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (primary after merge): %v", err)
	}
	valueFound := false
	for _, v := range primaryValues {
		if v.FieldID == cfDef && v.Value == "carried-over" {
			valueFound = true
		}
	}
	if !valueFound {
		t.Fatalf("MergeInto did not carry the duplicate's custom field onto the primary, got %v", primaryValues)
	}

	var mergedInto *uuid.UUID
	if err := pool.QueryRow(sysCtx, `SELECT merged_into_id FROM contacts WHERE id = $1`, duplicate.ID).Scan(&mergedInto); err != nil {
		t.Fatalf("read duplicate after merge: %v", err)
	}
	if mergedInto == nil || *mergedInto != primary.ID {
		t.Fatalf("duplicate not soft-deleted onto primary: merged_into_id=%v", mergedInto)
	}

	// Merging an already-merged contact a second time must fail cleanly at
	// the service layer (ErrAlreadyMerged) rather than silently re-running.
	svc := NewService(repo)
	if _, err := svc.MergeContacts(ctxOwn, primary.ID, duplicate.ID, tenantOwn); !errors.Is(err, ErrAlreadyMerged) {
		t.Fatalf("MergeContacts (already merged): expected ErrAlreadyMerged, got %v", err)
	}
}

func TestRepository_DeletionImpact_TenantScopedLiveFromCatalog(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact DeletionImpact Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact DeletionImpact Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-impact-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	target := &models.Contact{ID: uuid.New(), TenantID: tenantOwn, FirstName: "Target", LastName: "Contact", Visibility: "shared", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, target); err != nil {
		t.Fatalf("Create target: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", target.ID)

	// CASCADE reference.
	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Impact-Tag-%s", uuid.New().String()[:8]), "entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)
	if err := repo.AddTags(ctxOwn, target.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}

	// SET NULL reference.
	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "activity_type": "note", "subject": "Impact test activity",
		"contact_id": target.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	// RESTRICT reference.
	protocolID := testutil.SeedRow(t, pool, "advisory_protocols", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "contact_id": target.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "advisory_protocols", protocolID)

	items, err := repo.DeletionImpact(ctxOwn, target.ID, tenantOwn)
	if err != nil {
		t.Fatalf("DeletionImpact: %v", err)
	}

	byTable := make(map[string]DeletionImpactItem, len(items))
	for _, item := range items {
		byTable[item.Table] = item
	}

	if got, ok := byTable["contact_tags"]; !ok || got.Action != "cascade" || got.Count != 1 {
		t.Fatalf("contact_tags impact = %+v, ok=%v, want action=cascade count=1", got, ok)
	}
	if got, ok := byTable["activities"]; !ok || got.Action != "set_null" || got.Count != 1 {
		t.Fatalf("activities impact = %+v, ok=%v, want action=set_null count=1", got, ok)
	}
	if got, ok := byTable["advisory_protocols"]; !ok || got.Action != "restrict" || got.Count != 1 {
		t.Fatalf("advisory_protocols impact = %+v, ok=%v, want action=restrict count=1", got, ok)
	}
	// A table with zero matching rows for this contact (e.g. deals, never
	// seeded here) must not show up at all -- the preview lists impact, not
	// every table pg_constraint knows about.
	if _, ok := byTable["deals"]; ok {
		t.Fatalf("deals should not appear with zero rows referencing the contact, got %+v", byTable["deals"])
	}

	// Cross-tenant: the same contact ID under the wrong tenant context must
	// come back empty, not a leaked count. RLS scopes the session to
	// tenantOther, and DeletionImpact's own tenant_id filter (where the
	// referencing table has one) reinforces that as defense-in-depth.
	foreignItems, err := repo.DeletionImpact(ctxOther, target.ID, tenantOther)
	if err != nil {
		t.Fatalf("DeletionImpact (foreign tenant): %v", err)
	}
	if len(foreignItems) != 0 {
		t.Fatalf("DeletionImpact (foreign tenant) = %+v, want empty", foreignItems)
	}
}

// TestRepository_ListRetentionCandidates covers the query the retention
// engine's ContactRetentionHandler (internal/security/gdpr/retention_contacts.go)
// relies on: the later of updated_at and the most recent activity decides
// whether a contact is due, not updated_at alone.
func TestRepository_ListRetentionCandidates(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Retention Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact Retention Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-retention-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	// Due: no activity, updated_at alone is past cutoff.
	due := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Due", "last_name": "Contact",
		"created_by": userOwn, "created_at": old, "updated_at": old,
	})
	defer testutil.CleanupRow(t, pool, "contacts", due)

	// Not due: updated_at is recent.
	recent := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Recent", "last_name": "Contact",
		"created_by": userOwn, "created_at": fresh, "updated_at": fresh,
	})
	defer testutil.CleanupRow(t, pool, "contacts", recent)

	// Not due despite an old updated_at: a recent activity keeps the retention
	// clock from elapsing -- the whole reason ListRetentionCandidates joins
	// activities instead of reading updated_at alone.
	stillWorked := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "StillWorked", "last_name": "Contact",
		"created_by": userOwn, "created_at": old, "updated_at": old,
	})
	defer testutil.CleanupRow(t, pool, "contacts", stillWorked)
	recentActivity := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id": tenantOwn, "activity_type": "note", "subject": "Still in progress",
		"contact_id": stillWorked, "created_by": userOwn, "created_at": fresh,
	})
	defer testutil.CleanupRow(t, pool, "activities", recentActivity)

	// Foreign tenant, old updated_at -- must never appear for tenantOwn.
	foreign := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOther, "first_name": "Foreign", "last_name": "Contact",
		"created_by": userOwn, "created_at": old, "updated_at": old,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreign)

	got, err := repo.ListRetentionCandidates(ctxOwn, tenantOwn, cutoff)
	if err != nil {
		t.Fatalf("ListRetentionCandidates: %v", err)
	}

	byID := make(map[uuid.UUID]bool, len(got))
	for _, id := range got {
		byID[id] = true
	}
	if !byID[due] {
		t.Fatalf("ListRetentionCandidates = %v, want it to include the due contact %s", got, due)
	}
	if byID[recent] {
		t.Fatalf("ListRetentionCandidates = %v, must not include the recently updated contact %s", got, recent)
	}
	if byID[stillWorked] {
		t.Fatalf("ListRetentionCandidates = %v, must not include the contact with a recent activity %s", got, stillWorked)
	}
	if byID[foreign] {
		t.Fatalf("ListRetentionCandidates = %v, must not include the foreign-tenant contact %s", got, foreign)
	}
}

// TestRepository_IsInUse_AdvisoryProtocolBlocksDeletion_DB proves the RESTRICT
// path end to end against real Postgres, not the in-memory MockRepository that
// contact/service_test.go uses for TestService_Delete_InUse: a contact with a
// hanging advisory_protocols row (migration 000137, ON DELETE RESTRICT) must
// come back from IsInUse with the "advisory protocols" reason, and
// Service.Delete must surface that reason wrapped in ErrContactInUse instead
// of letting the RESTRICT constraint bubble up as a raw SQL error. Gateway/
// gRPC map ErrContactInUse to codes.FailedPrecondition -> HTTP 409
// (internal/server/crm_grpc.go), so this is also the DB-level half of
// cov-gateway-crm-advisory-protocols' "409 with reason, not 500" done_when.
func TestRepository_IsInUse_AdvisoryProtocolBlocksDeletion_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Contact IsInUse Advisory Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("crm-contact-isinuse-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	target := &models.Contact{ID: uuid.New(), TenantID: tenantID, FirstName: "Advisory", LastName: "Bound", Visibility: "shared", CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatalf("Create target: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", target.ID)

	inUse, reason, err := repo.IsInUse(ctx, target.ID, tenantID)
	if err != nil {
		t.Fatalf("IsInUse (before protocol): %v", err)
	}
	if inUse {
		t.Fatalf("IsInUse (before protocol) = true, reason %q, want false", reason)
	}

	protocolID := testutil.SeedRow(t, pool, "advisory_protocols", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "contact_id": target.ID, "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "advisory_protocols", protocolID)

	inUse, reason, err = repo.IsInUse(ctx, target.ID, tenantID)
	if err != nil {
		t.Fatalf("IsInUse (with protocol): %v", err)
	}
	if !inUse {
		t.Fatalf("IsInUse (with protocol) = false, want true")
	}
	if !strings.Contains(reason, "advisory protocols") {
		t.Fatalf("IsInUse reason = %q, want it to mention advisory protocols", reason)
	}

	err = svc.Delete(ctx, target.ID, tenantID)
	if !errors.Is(err, ErrContactInUse) {
		t.Fatalf("Delete error = %v, want ErrContactInUse", err)
	}
	if !strings.Contains(err.Error(), "advisory protocols") {
		t.Fatalf("Delete error = %q, want it to mention advisory protocols", err.Error())
	}

	// Not deleted: the RESTRICT constraint was never reached because the
	// service-level check short-circuits first.
	if _, getErr := repo.GetByID(ctx, target.ID, tenantID); getErr != nil {
		t.Fatalf("GetByID after blocked delete: %v, want the contact to still exist", getErr)
	}
}

// TestRepository_Delete_MergedPrimaryContact_DB proves the fix for
// contacts_merged_into_id_fkey: before migration 000318, deleting a contact
// that served as the primary of a completed merge failed with a raw
// unhandled FK violation (confdeltype NO ACTION), because IsInUse only ever
// checked the two RESTRICT FKs (advisory_protocols, dialer_campaign_contacts)
// and never knew about merged_into_id. Deleting the primary must now succeed
// and clear the duplicate's merged_into_id rather than blocking or erroring.
func TestRepository_Delete_MergedPrimaryContact_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Contact Delete-Merged-Primary Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("crm-contact-delete-merged-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	primary := &models.Contact{ID: uuid.New(), TenantID: tenantID, FirstName: "Primary", LastName: "ToDelete", Visibility: "shared", CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	duplicate := &models.Contact{ID: uuid.New(), TenantID: tenantID, FirstName: "Duplicate", LastName: "Survives", Visibility: "shared", CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctx, primary); err != nil {
		t.Fatalf("Create primary: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", primary.ID)
	if err := repo.Create(ctx, duplicate); err != nil {
		t.Fatalf("Create duplicate: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", duplicate.ID)

	if err := repo.MergeInto(ctx, primary.ID, duplicate.ID, tenantID); err != nil {
		t.Fatalf("MergeInto: %v", err)
	}

	inUse, reason, err := repo.IsInUse(ctx, primary.ID, tenantID)
	if err != nil {
		t.Fatalf("IsInUse (merge primary): %v", err)
	}
	if inUse {
		t.Fatalf("IsInUse (merge primary) = true (%q), want false — merged_into_id is not one of the RESTRICT FKs IsInUse checks", reason)
	}

	if err := repo.Delete(ctx, primary.ID, tenantID); err != nil {
		t.Fatalf("Delete primary of a completed merge: %v — merged_into_id must be ON DELETE SET NULL, not NO ACTION (migration 000318)", err)
	}

	if _, getErr := repo.GetByID(ctx, primary.ID, tenantID); getErr == nil {
		t.Fatal("GetByID: primary still exists after Delete")
	}

	var mergedInto *uuid.UUID
	if err := pool.QueryRow(sysCtx, `SELECT merged_into_id FROM contacts WHERE id = $1`, duplicate.ID).Scan(&mergedInto); err != nil {
		t.Fatalf("read duplicate after primary deletion: %v", err)
	}
	if mergedInto != nil {
		t.Fatalf("duplicate.merged_into_id = %v after its primary was deleted, want NULL (ON DELETE SET NULL)", *mergedInto)
	}
}
