package company

// DB integration tests for the parts of postgres_repository.go that carried
// 0% coverage: List filtering, HasContacts/GetContactCount (the ErrCompanyInUse
// gate), tags, custom fields and duplicate-detection/merge. rls_test.go and
// tenant_write_test.go already cover the plain CRUD RLS surface — this file
// covers the SQL these two files don't reach.

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
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-company-list-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	aerospace := "aerospace"
	food := "food"
	rockets := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Acme Rockets GmbH", Industry: &aerospace, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	bakery := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Acme Bakery GmbH", Industry: &food, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	zeta := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Zeta Systems AG", Industry: &aerospace, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	foreign := &models.Company{ID: uuid.New(), TenantID: tenantOther, Name: "Acme Rockets Foreign GmbH", Industry: &aerospace, CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Company{rockets, bakery, zeta} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "companies", c.ID)
	}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", foreign.ID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("Key-Account-%s", uuid.New().String()[:8]),
		"entity_type": "company",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)
	if err := repo.AddTags(ctxOwn, rockets.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}

	// Search scopes to the tenant: "acme" matches two of ours and one foreign
	// company that must not appear.
	searchResults, searchTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, Search: "acme"}, 0, 20)
	if err != nil {
		t.Fatalf("List (search): %v", err)
	}
	if searchTotal != 2 {
		t.Fatalf("List (search): expected total=2, got %d", searchTotal)
	}
	for _, c := range searchResults {
		if c.ID == foreign.ID {
			t.Fatal("List (search) leaked a foreign-tenant company")
		}
	}

	// Industry filter.
	industryResults, industryTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, Industry: &aerospace}, 0, 20)
	if err != nil {
		t.Fatalf("List (industry): %v", err)
	}
	if industryTotal != 2 {
		t.Fatalf("List (industry): expected total=2 (rockets+zeta), got %d", industryTotal)
	}
	for _, c := range industryResults {
		if c.ID == bakery.ID {
			t.Fatal("List (industry): bakery (food) leaked into aerospace filter")
		}
	}

	// Tag filter: only rockets carries the tag.
	tagResults, tagTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, TagIDs: []uuid.UUID{tagID}}, 0, 20)
	if err != nil {
		t.Fatalf("List (tag): %v", err)
	}
	if tagTotal != 1 || len(tagResults) != 1 || tagResults[0].ID != rockets.ID {
		t.Fatalf("List (tag): expected exactly rockets, got total=%d results=%d", tagTotal, len(tagResults))
	}

	// Pagination: name-ascending, page size 1 — first page is the
	// alphabetically first of our three companies (Acme Bakery), total
	// still reflects the full tenant-scoped set.
	page, total, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, SortBy: "name", SortDesc: false}, 0, 1)
	if err != nil {
		t.Fatalf("List (page 1): %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ID != bakery.ID {
		t.Fatalf("List (page 1): expected total=3 first=bakery, got total=%d first=%v", total, page)
	}

	// Offset beyond the total must return an empty page, not an error.
	empty, emptyTotal, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn}, 50, 20)
	if err != nil {
		t.Fatalf("List (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("List (offset beyond total): expected 0 rows with total=3, got %d rows total=%d", len(empty), emptyTotal)
	}
}

// TestService_Delete_HangingContactsBlockDeletionButForeignTenantContactDoesNot
// exercises the ErrCompanyInUse gate against a real repository: a contact in
// the SAME tenant blocks deletion, but a contact row that (incorrectly, by
// data-integrity accident) references the company_id of a company belonging
// to a DIFFERENT tenant must not — HasContacts/GetContactCount filter on
// tenant_id, not just company_id, so that contact is invisible to the query
// that decides whether the company is "in use".
func TestService_Delete_HangingContactsBlockDeletionButForeignTenantContactDoesNot(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company Delete Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company Delete Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-del-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-company-del-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// Deleting a company that does not exist at all.
	if err := svc.Delete(ctxOwn, uuid.New(), tenantOwn); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("Delete (missing company): expected ErrCompanyNotFound, got %v", err)
	}

	target := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Hanging Contacts GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, target); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", target.ID)

	// A contact belonging to a DIFFERENT tenant that happens to carry this
	// company's id (seeded directly — this is the data-integrity accident the
	// scoped query has to survive, not a state the application would create).
	foreignContact := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Contact",
		"created_by": userOther,
		"company_id": target.ID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContact)

	if hasContacts, err := repo.HasContacts(ctxOwn, target.ID, tenantOwn); err != nil {
		t.Fatalf("HasContacts (foreign-tenant dangling contact): %v", err)
	} else if hasContacts {
		t.Fatal("HasContacts: a foreign-tenant contact blocked deletion")
	}
	if count, err := repo.GetContactCount(ctxOwn, target.ID, tenantOwn); err != nil {
		t.Fatalf("GetContactCount (foreign-tenant dangling contact): %v", err)
	} else if count != 0 {
		t.Fatalf("GetContactCount: expected 0, got %d", count)
	}
	if err := svc.Delete(ctxOwn, target.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (only a foreign-tenant contact hangs off it): unexpected error %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "companies", target.ID, 0)

	// Now the real case: a same-tenant contact must block deletion.
	blocked := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Blocked GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, blocked); err != nil {
		t.Fatalf("Create blocked: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", blocked.ID)
	ownContact := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOwn,
		"first_name": "Own",
		"last_name":  "Contact",
		"created_by": userOwn,
		"company_id": blocked.ID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", ownContact)

	if hasContacts, err := repo.HasContacts(ctxOwn, blocked.ID, tenantOwn); err != nil {
		t.Fatalf("HasContacts (own contact): %v", err)
	} else if !hasContacts {
		t.Fatal("HasContacts: expected true for a same-tenant contact")
	}
	if err := svc.Delete(ctxOwn, blocked.ID, tenantOwn); !errors.Is(err, ErrCompanyInUse) {
		t.Fatalf("Delete (hanging same-tenant contact): expected ErrCompanyInUse, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "companies", blocked.ID, 1)
}

func TestRepository_Tags_AddGetRemoveRoundtripAndTagExistsIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company Tags Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company Tags Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-tags-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	company := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Tag Test GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, company); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", company.ID)

	ownTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"name":        fmt.Sprintf("Own-Tag-%s", uuid.New().String()[:8]),
		"entity_type": "company",
	})
	defer testutil.CleanupRow(t, pool, "tags", ownTag)
	foreignTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOther,
		"name":        fmt.Sprintf("Foreign-Tag-%s", uuid.New().String()[:8]),
		"entity_type": "company",
	})
	defer testutil.CleanupRow(t, pool, "tags", foreignTag)

	// TagExists carries no explicit tenant_id predicate in its SQL — RLS is the
	// only thing standing between "own tenant" and "any tenant" here.
	if exists, err := repo.TagExists(ctxOwn, ownTag, models.EntityTypeCompany); err != nil {
		t.Fatalf("TagExists (own): %v", err)
	} else if !exists {
		t.Fatal("TagExists: expected true for own-tenant tag")
	}
	if exists, err := repo.TagExists(ctxOwn, foreignTag, models.EntityTypeCompany); err != nil {
		t.Fatalf("TagExists (foreign): %v", err)
	} else if exists {
		t.Fatal("TagExists: a foreign-tenant tag was visible")
	}

	// Same isolation, one layer up: Service.AddTags rejects a foreign tag as
	// "not found" (RLS hides the row, the service can't tell the difference
	// between "does not exist" and "belongs to someone else" — by design).
	if _, err := svc.AddTags(ctxOwn, company.ID, []uuid.UUID{foreignTag}, tenantOwn); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("AddTags (foreign tag): expected ErrTagNotFound, got %v", err)
	}

	if err := repo.AddTags(ctxOwn, company.ID, []uuid.UUID{ownTag}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}
	tags, err := repo.GetTags(ctxOwn, company.ID)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != ownTag {
		t.Fatalf("GetTags: expected exactly [%s], got %v", ownTag, tags)
	}

	// Foreign-tenant read of the same company's tags — RLS on companies
	// already hides the company itself, but confirm company_tags does not
	// leak the join independently.
	foreignTags, err := repo.GetTags(ctxOther, company.ID)
	if err != nil {
		t.Fatalf("GetTags (foreign ctx): %v", err)
	}
	if len(foreignTags) != 0 {
		t.Fatalf("GetTags (foreign ctx): expected no tags visible, got %v", foreignTags)
	}

	if err := repo.RemoveTags(ctxOwn, company.ID, []uuid.UUID{ownTag}); err != nil {
		t.Fatalf("RemoveTags: %v", err)
	}
	tagsAfterRemove, err := repo.GetTags(ctxOwn, company.ID)
	if err != nil {
		t.Fatalf("GetTags (after remove): %v", err)
	}
	if len(tagsAfterRemove) != 0 {
		t.Fatalf("GetTags (after remove): expected none, got %v", tagsAfterRemove)
	}
}

func TestRepository_CustomFields_SetGetRoundtripAndEmptyReturnsNoRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company CustomFields Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-cf-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	company := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Custom Field GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, company); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", company.ID)

	// A company without any values must return an empty slice and no error,
	// not a nil-vs-error ambiguity.
	empty, err := repo.GetCustomFieldValues(ctxOwn, company.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetCustomFieldValues (empty): expected none, got %v", empty)
	}

	textField := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"entity_type": "company",
		"field_name":  fmt.Sprintf("segment_%s", uuid.New().String()[:8]),
		"field_label": "Segment",
		"field_type":  "text",
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", textField)
	numberField := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"entity_type": "company",
		"field_name":  fmt.Sprintf("headcount_%s", uuid.New().String()[:8]),
		"field_label": "Headcount",
		"field_type":  "number",
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", numberField)

	if err := repo.SetCustomFieldValues(ctxOwn, company.ID, map[uuid.UUID]any{
		textField:   "enterprise",
		numberField: float64(42),
	}); err != nil {
		t.Fatalf("SetCustomFieldValues: %v", err)
	}

	values, err := repo.GetCustomFieldValues(ctxOwn, company.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("GetCustomFieldValues: expected 2 values, got %d", len(values))
	}
	seen := make(map[uuid.UUID]any, len(values))
	for _, v := range values {
		seen[v.FieldID] = v.Value
	}
	if seen[textField] != "enterprise" {
		t.Fatalf("GetCustomFieldValues: text field = %v, want %q", seen[textField], "enterprise")
	}
	if seen[numberField] != float64(42) {
		t.Fatalf("GetCustomFieldValues: number field = %v, want 42", seen[numberField])
	}

	// Upsert semantics: setting again overwrites, does not duplicate.
	if err := repo.SetCustomFieldValues(ctxOwn, company.ID, map[uuid.UUID]any{textField: "smb"}); err != nil {
		t.Fatalf("SetCustomFieldValues (overwrite): %v", err)
	}
	afterOverwrite, err := repo.GetCustomFieldValues(ctxOwn, company.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (after overwrite): %v", err)
	}
	if len(afterOverwrite) != 2 {
		t.Fatalf("GetCustomFieldValues (after overwrite): expected still 2 rows, got %d", len(afterOverwrite))
	}
	for _, v := range afterOverwrite {
		if v.FieldID == textField && v.Value != "smb" {
			t.Fatalf("GetCustomFieldValues (after overwrite): text field = %v, want %q", v.Value, "smb")
		}
	}
}

func TestRepository_FindDuplicateCandidates_DomainExactAndFuzzyNameExcludeMergedAndForeignTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company Dup Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company Dup Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-dup-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-company-dup-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	domain := fmt.Sprintf("dup-%s.example", uuid.New().String()[:8])
	src := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Baumann Elektrotechnik GmbH", Domain: &domain, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, src); err != nil {
		t.Fatalf("Create src: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", src.ID)

	domainExact := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Completely Different Name Ltd", Domain: &domain, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, domainExact); err != nil {
		t.Fatalf("Create domainExact: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", domainExact.ID)

	nameFuzzy := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Baumann Elektrotechnick GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, nameFuzzy); err != nil {
		t.Fatalf("Create nameFuzzy: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", nameFuzzy.ID)

	// Already-merged: shares the exact domain but must not surface as a candidate.
	merged := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Merged Away GmbH", Domain: &domain, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, merged); err != nil {
		t.Fatalf("Create merged: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", merged.ID)
	if _, err := pool.Exec(ctxOwn, `UPDATE companies SET merged_into_id = $1 WHERE id = $2`, src.ID, merged.ID); err != nil {
		t.Fatalf("mark merged: %v", err)
	}

	// Foreign tenant, same domain — must never appear regardless of match strength.
	foreign := &models.Company{ID: uuid.New(), TenantID: tenantOther, Name: "Foreign Duplicate GmbH", Domain: &domain, CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", foreign.ID)

	candidates, err := repo.FindDuplicateCandidates(ctxOwn, src.ID, tenantOwn)
	if err != nil {
		t.Fatalf("FindDuplicateCandidates: %v", err)
	}

	byID := make(map[uuid.UUID]*DuplicateCandidate, len(candidates))
	for _, c := range candidates {
		byID[c.Company.ID] = c
	}
	if c, ok := byID[domainExact.ID]; !ok || c.MatchType != "domain_exact" {
		t.Fatalf("expected domain_exact candidate for %s, got %+v", domainExact.ID, byID[domainExact.ID])
	}
	if c, ok := byID[nameFuzzy.ID]; !ok || c.MatchType != "name_fuzzy" {
		t.Fatalf("expected name_fuzzy candidate for %s, got %+v", nameFuzzy.ID, byID[nameFuzzy.ID])
	}
	if _, ok := byID[merged.ID]; ok {
		t.Fatal("FindDuplicateCandidates surfaced an already-merged company")
	}
	if _, ok := byID[foreign.ID]; ok {
		t.Fatal("FindDuplicateCandidates leaked a foreign-tenant company")
	}
}

// TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes
// exercises the full merge transaction: contacts/activities/deals reassigned
// to the primary, tags and custom fields merged onto it, and the duplicate
// soft-deleted via merged_into_id. It also covers the cross-tenant guard at
// the top of MergeInto.
func TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company Merge Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company Merge Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-merge-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-company-merge-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	primary := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Primary GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	duplicate := &models.Company{ID: uuid.New(), TenantID: tenantOwn, Name: "Duplicate GmbH", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, c := range []*models.Company{primary, duplicate} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "companies", c.ID)
	}

	// Cross-tenant guard: a duplicateID from a foreign tenant must error out,
	// not merge/leak. Uses a company id that plainly belongs to tenantOther —
	// RLS hides the row under ctxOwn, so this also proves the guard doesn't
	// depend on it being able to read the foreign row.
	foreignCo := &models.Company{ID: uuid.New(), TenantID: tenantOther, Name: "Foreign GmbH", CreatedBy: userOther, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(testutil.WithTenantCtx(context.Background(), tenantOther), foreignCo); err != nil {
		t.Fatalf("Create foreignCo: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", foreignCo.ID)
	if err := repo.MergeInto(ctxOwn, primary.ID, foreignCo.ID, tenantOwn); err == nil {
		t.Fatal("MergeInto: expected an error merging a foreign-tenant duplicate, got nil")
	}

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "first_name": "Moved", "last_name": "Contact",
		"created_by": userOwn, "company_id": duplicate.ID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "activity_type": "note", "subject": "Merge test activity",
		"company_id": duplicate.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Merge-Stage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)
	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": "Moved Deal", "stage_id": stageID,
		"company_id": duplicate.ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	dupTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Merge-Tag-%s", uuid.New().String()[:8]), "entity_type": "company",
	})
	defer testutil.CleanupRow(t, pool, "tags", dupTag)
	if err := repo.AddTags(ctxOwn, duplicate.ID, []uuid.UUID{dupTag}); err != nil {
		t.Fatalf("AddTags (duplicate): %v", err)
	}

	cfDef := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "entity_type": "company", "field_name": fmt.Sprintf("merge_field_%s", uuid.New().String()[:8]),
		"field_label": "Merge Field", "field_type": "text", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", cfDef)
	if err := repo.SetCustomFieldValues(ctxOwn, duplicate.ID, map[uuid.UUID]any{cfDef: "carried-over"}); err != nil {
		t.Fatalf("SetCustomFieldValues (duplicate): %v", err)
	}

	if err := repo.MergeInto(ctxOwn, primary.ID, duplicate.ID, tenantOwn); err != nil {
		t.Fatalf("MergeInto: %v", err)
	}

	var contactCompany, activityCompany, dealCompany uuid.UUID
	if err := pool.QueryRow(sysCtx, `SELECT company_id FROM contacts WHERE id = $1`, contactID).Scan(&contactCompany); err != nil {
		t.Fatalf("read contact after merge: %v", err)
	}
	if contactCompany != primary.ID {
		t.Fatalf("contact not reassigned: company_id=%s, want %s", contactCompany, primary.ID)
	}
	if err := pool.QueryRow(sysCtx, `SELECT company_id FROM activities WHERE id = $1`, activityID).Scan(&activityCompany); err != nil {
		t.Fatalf("read activity after merge: %v", err)
	}
	if activityCompany != primary.ID {
		t.Fatalf("activity not reassigned: company_id=%s, want %s", activityCompany, primary.ID)
	}
	if err := pool.QueryRow(sysCtx, `SELECT company_id FROM deals WHERE id = $1`, dealID).Scan(&dealCompany); err != nil {
		t.Fatalf("read deal after merge: %v", err)
	}
	if dealCompany != primary.ID {
		t.Fatalf("deal not reassigned: company_id=%s, want %s", dealCompany, primary.ID)
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
	if err := pool.QueryRow(sysCtx, `SELECT merged_into_id FROM companies WHERE id = $1`, duplicate.ID).Scan(&mergedInto); err != nil {
		t.Fatalf("read duplicate after merge: %v", err)
	}
	if mergedInto == nil || *mergedInto != primary.ID {
		t.Fatalf("duplicate not soft-deleted onto primary: merged_into_id=%v", mergedInto)
	}

	// Merging an already-merged company a second time must fail cleanly at
	// the service layer (ErrAlreadyMerged) rather than silently re-running.
	svc := NewService(repo)
	if _, err := svc.MergeCompanies(ctxOwn, primary.ID, duplicate.ID, tenantOwn); !errors.Is(err, ErrAlreadyMerged) {
		t.Fatalf("MergeCompanies (already merged): expected ErrAlreadyMerged, got %v", err)
	}
}

// TestRepository_Delete_MergedPrimaryCompany_DB proves the fix for
// companies_merged_into_id_fkey: before migration 000321, deleting a company
// that served as the primary of a completed merge failed with a raw
// unhandled FK violation (confdeltype NO ACTION), because Service.Delete
// only ever checked HasContacts and never knew about merged_into_id.
// Deleting the primary must now succeed and clear the duplicate's
// merged_into_id rather than blocking or erroring — mirrors the identical
// fix already applied to contacts.merged_into_id.
func TestRepository_Delete_MergedPrimaryCompany_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Company Delete-Merged-Primary Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("crm-company-delete-merged-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	primary := &models.Company{ID: uuid.New(), TenantID: tenantID, Name: "Primary ToDelete", CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	duplicate := &models.Company{ID: uuid.New(), TenantID: tenantID, Name: "Duplicate Survives", CreatedBy: userID, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctx, primary); err != nil {
		t.Fatalf("Create primary: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", primary.ID)
	if err := repo.Create(ctx, duplicate); err != nil {
		t.Fatalf("Create duplicate: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", duplicate.ID)

	if err := repo.MergeInto(ctx, primary.ID, duplicate.ID, tenantID); err != nil {
		t.Fatalf("MergeInto: %v", err)
	}

	hasContacts, err := repo.HasContacts(ctx, primary.ID, tenantID)
	if err != nil {
		t.Fatalf("HasContacts (merge primary): %v", err)
	}
	if hasContacts {
		t.Fatalf("HasContacts (merge primary) = true, want false — merged_into_id is not a contact")
	}

	if err := repo.Delete(ctx, primary.ID, tenantID); err != nil {
		t.Fatalf("Delete primary of a completed merge: %v — merged_into_id must be ON DELETE SET NULL, not NO ACTION (migration 000321)", err)
	}

	if _, getErr := repo.GetByID(ctx, primary.ID, tenantID); getErr == nil {
		t.Fatal("GetByID: primary still exists after Delete")
	}

	var mergedInto *uuid.UUID
	if err := pool.QueryRow(sysCtx, `SELECT merged_into_id FROM companies WHERE id = $1`, duplicate.ID).Scan(&mergedInto); err != nil {
		t.Fatalf("read duplicate after primary deletion: %v", err)
	}
	if mergedInto != nil {
		t.Fatalf("duplicate.merged_into_id = %v after its primary was deleted, want NULL (ON DELETE SET NULL)", *mergedInto)
	}
}
