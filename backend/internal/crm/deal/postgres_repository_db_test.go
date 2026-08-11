package deal

// DB integration tests for the parts of postgres_repository.go that
// rls_test.go and tenant_write_test.go don't reach: List filtering/sorting/
// pagination, the relation name lookups (single and batch), tags, custom
// fields, the exists/GetStage checks, and SetClosedAt.
//
// Several relation lookups (GetStageName, GetContactName, GetCompanyName,
// GetOwnerName, GetStage, the *Exists checks) carry no explicit tenant_id
// predicate in their SQL -- unlike the equivalent contact-package lookups,
// which always pass tenantID as a WHERE argument. That means RLS is the
// *only* thing stopping a cross-tenant lookup here. The cross-tenant cases
// below exist specifically to prove RLS is doing that job, not to describe
// intended behavior of the Go code itself.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRepository_List_FiltersSortsAndPaginatesTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-list-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOther, "email": fmt.Sprintf("crm-deal-list-%s@tenantother.local", uuid.New().String()[:8]), "password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	stageA := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": fmt.Sprintf("Stage-A-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageA)
	stageB := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": fmt.Sprintf("Stage-B-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageB)
	stageForeign := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther, "name": fmt.Sprintf("Stage-Foreign-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageForeign)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Deal", "last_name": "Contact", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id": tenantOwn, "name": "Deal Co GmbH", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	mkDeal := func(name string, value int64, stageID uuid.UUID, contact, company, owner *uuid.UUID) *models.Deal {
		return &models.Deal{
			ID: uuid.New(), TenantID: tenantOwn, Name: name, Value: decimal.NewFromInt(value), Currency: "EUR",
			StageID: stageID, ContactID: contact, CompanyID: company, OwnerID: owner,
			CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		}
	}

	alpha := mkDeal("Alpha Deal", 100, stageA, &contactID, &companyID, &userOwn)
	beta := mkDeal("Beta Deal", 500, stageB, nil, nil, nil)
	zulu := mkDeal("Zulu Search-Match", 50, stageA, nil, nil, nil)
	for _, d := range []*models.Deal{alpha, beta, zulu} {
		if err := repo.Create(ctxOwn, d); err != nil {
			t.Fatalf("Create %s: %v", d.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "deals", d.ID)
	}
	foreign := &models.Deal{
		ID: uuid.New(), TenantID: tenantOther, Name: "Search-Match Foreign", Value: decimal.NewFromInt(1), Currency: "EUR",
		StageID: stageForeign, CreatedBy: userOther, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOther, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "deals", foreign.ID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Hot-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)
	if err := repo.AddTags(ctxOwn, alpha.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags: %v", err)
	}

	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{StageID: &stageA}, 0, 20); err != nil {
		t.Fatalf("List (stage): %v", err)
	} else if total != 2 {
		t.Fatalf("List (stage): expected 2 (alpha+zulu), got total=%d results=%v", total, res)
	}

	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{ContactID: &contactID}, 0, 20); err != nil {
		t.Fatalf("List (contact): %v", err)
	} else if total != 1 || len(res) != 1 || res[0].ID != alpha.ID {
		t.Fatalf("List (contact): expected exactly alpha, got total=%d results=%v", total, res)
	}

	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{CompanyID: &companyID}, 0, 20); err != nil {
		t.Fatalf("List (company): %v", err)
	} else if total != 1 || len(res) != 1 || res[0].ID != alpha.ID {
		t.Fatalf("List (company): expected exactly alpha, got total=%d results=%v", total, res)
	}

	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{OwnerID: &userOwn}, 0, 20); err != nil {
		t.Fatalf("List (owner): %v", err)
	} else if total != 1 || len(res) != 1 || res[0].ID != alpha.ID {
		t.Fatalf("List (owner): expected exactly alpha, got total=%d results=%v", total, res)
	}

	// Search matches "search-match" case-insensitively and stays tenant
	// scoped: the foreign deal shares the substring but must not appear.
	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{Search: "search-match"}, 0, 20); err != nil {
		t.Fatalf("List (search): %v", err)
	} else if total != 1 || len(res) != 1 || res[0].ID != zulu.ID {
		t.Fatalf("List (search): expected exactly zulu, got total=%d results=%v", total, res)
	}

	if res, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{TagIDs: []uuid.UUID{tagID}}, 0, 20); err != nil {
		t.Fatalf("List (tag): %v", err)
	} else if total != 1 || len(res) != 1 || res[0].ID != alpha.ID {
		t.Fatalf("List (tag): expected exactly alpha, got total=%d results=%v", total, res)
	}

	// Sort by name ascending across the full tenant set (3 rows).
	asc, ascTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{SortBy: "name"}, 0, 20)
	if err != nil {
		t.Fatalf("List (sort name asc): %v", err)
	}
	if ascTotal != 3 || len(asc) != 3 || asc[0].ID != alpha.ID || asc[2].ID != zulu.ID {
		t.Fatalf("List (sort name asc): expected alpha..zulu ascending, got %v", asc)
	}
	desc, _, err := repo.List(ctxOwn, tenantOwn, ListFilter{SortBy: "name", SortDesc: true}, 0, 20)
	if err != nil {
		t.Fatalf("List (sort name desc): %v", err)
	}
	if len(desc) != 3 || desc[0].ID != zulu.ID || desc[2].ID != alpha.ID {
		t.Fatalf("List (sort name desc): expected zulu..alpha descending, got %v", desc)
	}

	// Sort by value descending: beta(500) > alpha(100) > zulu(50).
	byValue, _, err := repo.List(ctxOwn, tenantOwn, ListFilter{SortBy: "value", SortDesc: true}, 0, 20)
	if err != nil {
		t.Fatalf("List (sort value desc): %v", err)
	}
	if len(byValue) != 3 || byValue[0].ID != beta.ID || byValue[2].ID != zulu.ID {
		t.Fatalf("List (sort value desc): expected beta..zulu, got %v", byValue)
	}

	// Pagination: page size 1 lands on the alphabetically first row, but the
	// total still reflects the full tenant-scoped count (3), not the page size.
	page, total, err := repo.List(ctxOwn, tenantOwn, ListFilter{SortBy: "name"}, 0, 1)
	if err != nil {
		t.Fatalf("List (page 1): %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ID != alpha.ID {
		t.Fatalf("List (page 1): expected total=3 first=alpha, got total=%d page=%v", total, page)
	}

	// Offset beyond the total returns an empty page, not an error.
	empty, emptyTotal, err := repo.List(ctxOwn, tenantOwn, ListFilter{}, 50, 20)
	if err != nil {
		t.Fatalf("List (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("List (offset beyond total): expected 0 rows with total=3, got %d rows total=%d", len(empty), emptyTotal)
	}

	// A foreign-tenant caller never sees these rows regardless of filter.
	foreignView, foreignTotal, err := repo.List(ctxOther, tenantOther, ListFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("List (foreign tenant): %v", err)
	}
	if foreignTotal != 1 || len(foreignView) != 1 || foreignView[0].ID != foreign.ID {
		t.Fatalf("List (foreign tenant): expected exactly the foreign deal, got total=%d results=%v", foreignTotal, foreignView)
	}
}

func TestRepository_RelationNameLookups_MissingAndCrossTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal RelName Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal RelName Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-relname-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x", "first_name": "Own", "last_name": "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userForeign := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOther, "email": fmt.Sprintf("crm-deal-relname-%s@tenantother.local", uuid.New().String()[:8]), "password_hash": "x", "first_name": "Foreign", "last_name": "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", userForeign)

	stageOwn := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "Own Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOwn)
	stageForeign := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther, "name": "Foreign Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageForeign)

	contactOwn := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Con", "last_name": "Tact", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactOwn)
	contactForeign := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOther, "first_name": "For", "last_name": "Eign", "created_by": userForeign,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactForeign)

	companyOwn := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id": tenantOwn, "name": "Own Co GmbH", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyOwn)
	companyForeign := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id": tenantOther, "name": "Foreign Co GmbH", "created_by": userForeign,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyForeign)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if name, err := repo.GetStageName(ctxOwn, stageOwn); err != nil || name == "" {
		t.Fatalf("GetStageName (own): got name=%q err=%v", name, err)
	}
	if name, err := repo.GetStageName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetStageName (missing): expected (\"\", nil), got (%q, %v)", name, err)
	}
	if name, err := repo.GetStageName(ctxOwn, stageForeign); err != nil || name != "" {
		t.Fatalf("GetStageName (foreign tenant): expected RLS to hide the row, got (%q, %v)", name, err)
	}

	if name, err := repo.GetContactName(ctxOwn, contactOwn); err != nil || name != "Con Tact" {
		t.Fatalf("GetContactName (own): got name=%q err=%v", name, err)
	}
	if name, err := repo.GetContactName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetContactName (missing): expected (\"\", nil), got (%q, %v)", name, err)
	}
	if name, err := repo.GetContactName(ctxOwn, contactForeign); err != nil || name != "" {
		t.Fatalf("GetContactName (foreign tenant): expected RLS to hide the row, got (%q, %v)", name, err)
	}

	if name, err := repo.GetCompanyName(ctxOwn, companyOwn); err != nil || name != "Own Co GmbH" {
		t.Fatalf("GetCompanyName (own): got name=%q err=%v", name, err)
	}
	if name, err := repo.GetCompanyName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetCompanyName (missing): expected (\"\", nil), got (%q, %v)", name, err)
	}
	if name, err := repo.GetCompanyName(ctxOwn, companyForeign); err != nil || name != "" {
		t.Fatalf("GetCompanyName (foreign tenant): expected RLS to hide the row, got (%q, %v)", name, err)
	}

	if name, err := repo.GetOwnerName(ctxOwn, userOwn); err != nil || name != "Own Owner" {
		t.Fatalf("GetOwnerName (own): got name=%q err=%v", name, err)
	}
	if name, err := repo.GetOwnerName(ctxOwn, uuid.New()); err != nil || name != "" {
		t.Fatalf("GetOwnerName (missing): expected (\"\", nil), got (%q, %v)", name, err)
	}
	if name, err := repo.GetOwnerName(ctxOwn, userForeign); err != nil || name != "" {
		t.Fatalf("GetOwnerName (foreign tenant): expected RLS to hide the row, got (%q, %v)", name, err)
	}
}

func TestRepository_BatchNameLookups_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal BatchName Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal BatchName Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-batchname-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x", "first_name": "Batch", "last_name": "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userForeign := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOther, "email": fmt.Sprintf("crm-deal-batchname-%s@tenantother.local", uuid.New().String()[:8]), "password_hash": "x", "first_name": "Foreign", "last_name": "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", userForeign)

	stageOwn := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "Batch Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOwn)
	stageForeign := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther, "name": "Batch Foreign Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageForeign)

	contactOwn := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Batch", "last_name": "Contact", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactOwn)

	companyOwn := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id": tenantOwn, "name": "Batch Co GmbH", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	stageNames, err := repo.GetStageNames(ctxOwn, []uuid.UUID{stageOwn, stageForeign})
	if err != nil {
		t.Fatalf("GetStageNames: %v", err)
	}
	if len(stageNames) != 1 {
		t.Fatalf("GetStageNames: expected only the own-tenant stage, got %v", stageNames)
	}
	if _, ok := stageNames[stageForeign]; ok {
		t.Fatal("GetStageNames: leaked a foreign-tenant stage")
	}
	if empty, err := repo.GetStageNames(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetStageNames (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}

	contactNames, err := repo.GetContactNames(ctxOwn, []uuid.UUID{contactOwn})
	if err != nil {
		t.Fatalf("GetContactNames: %v", err)
	}
	if contactNames[contactOwn] != "Batch Contact" {
		t.Fatalf("GetContactNames: got %v", contactNames)
	}
	if empty, err := repo.GetContactNames(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetContactNames (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}

	companyNames, err := repo.GetCompanyNames(ctxOwn, []uuid.UUID{companyOwn})
	if err != nil {
		t.Fatalf("GetCompanyNames: %v", err)
	}
	if companyNames[companyOwn] != "Batch Co GmbH" {
		t.Fatalf("GetCompanyNames: got %v", companyNames)
	}
	if empty, err := repo.GetCompanyNames(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetCompanyNames (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}

	ownerNames, err := repo.GetOwnerNames(ctxOwn, []uuid.UUID{userOwn, userForeign})
	if err != nil {
		t.Fatalf("GetOwnerNames: %v", err)
	}
	if len(ownerNames) != 1 || ownerNames[userOwn] != "Batch Owner" {
		t.Fatalf("GetOwnerNames: expected only the own-tenant owner, got %v", ownerNames)
	}
	if empty, err := repo.GetOwnerNames(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetOwnerNames (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}
}

func TestRepository_Tags_AddGetRemoveAndBatchRoundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal Tags Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-tags-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "Tags Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	d1 := &models.Deal{ID: uuid.New(), TenantID: tenantOwn, Name: "Tag Deal One", Value: decimal.NewFromInt(1), Currency: "EUR", StageID: stageID, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	d2 := &models.Deal{ID: uuid.New(), TenantID: tenantOwn, Name: "Tag Deal Two", Value: decimal.NewFromInt(1), Currency: "EUR", StageID: stageID, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, d := range []*models.Deal{d1, d2} {
		if err := repo.Create(ctxOwn, d); err != nil {
			t.Fatalf("Create %s: %v", d.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "deals", d.ID)
	}

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Round-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	if err := repo.AddTags(ctxOwn, d1.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags d1: %v", err)
	}
	if err := repo.AddTags(ctxOwn, d2.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("AddTags d2: %v", err)
	}
	// AddTags with an empty slice must be a no-op, not an error.
	if err := repo.AddTags(ctxOwn, d1.ID, nil); err != nil {
		t.Fatalf("AddTags (empty): %v", err)
	}

	tags, err := repo.GetTags(ctxOwn, d1.ID)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tagID {
		t.Fatalf("GetTags: expected exactly [%s], got %v", tagID, tags)
	}

	batch, err := repo.GetTagsBatch(ctxOwn, []uuid.UUID{d1.ID, d2.ID})
	if err != nil {
		t.Fatalf("GetTagsBatch: %v", err)
	}
	if len(batch[d1.ID]) != 1 || len(batch[d2.ID]) != 1 {
		t.Fatalf("GetTagsBatch: expected one tag per deal, got %v", batch)
	}
	if empty, err := repo.GetTagsBatch(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetTagsBatch (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}

	if err := repo.RemoveTags(ctxOwn, d1.ID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("RemoveTags: %v", err)
	}
	afterRemove, err := repo.GetTags(ctxOwn, d1.ID)
	if err != nil {
		t.Fatalf("GetTags (after remove): %v", err)
	}
	if len(afterRemove) != 0 {
		t.Fatalf("GetTags (after remove): expected none, got %v", afterRemove)
	}
	// d2 must be unaffected by d1's removal.
	d2Tags, err := repo.GetTags(ctxOwn, d2.ID)
	if err != nil {
		t.Fatalf("GetTags (d2): %v", err)
	}
	if len(d2Tags) != 1 {
		t.Fatalf("GetTags (d2): expected the tag to remain, got %v", d2Tags)
	}
}

func TestRepository_CustomFieldValues_SetGetRoundtripAndBatch(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal CustomFields Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-cf-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "CF Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	d1 := &models.Deal{ID: uuid.New(), TenantID: tenantOwn, Name: "CF Deal One", Value: decimal.NewFromInt(1), Currency: "EUR", StageID: stageID, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	d2 := &models.Deal{ID: uuid.New(), TenantID: tenantOwn, Name: "CF Deal Two", Value: decimal.NewFromInt(1), Currency: "EUR", StageID: stageID, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	for _, d := range []*models.Deal{d1, d2} {
		if err := repo.Create(ctxOwn, d); err != nil {
			t.Fatalf("Create %s: %v", d.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "deals", d.ID)
	}

	// A deal without any values returns an empty slice, not nil-vs-error
	// ambiguity.
	empty, err := repo.GetCustomFieldValues(ctxOwn, d1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetCustomFieldValues (empty): expected none, got %v", empty)
	}
	// SetCustomFieldValues with an empty map is a no-op, not an error.
	if err := repo.SetCustomFieldValues(ctxOwn, d1.ID, nil); err != nil {
		t.Fatalf("SetCustomFieldValues (empty): %v", err)
	}

	segment := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "entity_type": "deal",
		"field_name": fmt.Sprintf("segment_%s", uuid.New().String()[:8]), "field_label": "Segment", "field_type": "text", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", segment)

	if err := repo.SetCustomFieldValues(ctxOwn, d1.ID, map[uuid.UUID]any{segment: "enterprise"}); err != nil {
		t.Fatalf("SetCustomFieldValues d1: %v", err)
	}
	if err := repo.SetCustomFieldValues(ctxOwn, d2.ID, map[uuid.UUID]any{segment: "smb"}); err != nil {
		t.Fatalf("SetCustomFieldValues d2: %v", err)
	}

	values, err := repo.GetCustomFieldValues(ctxOwn, d1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues: %v", err)
	}
	if len(values) != 1 || values[0].Value != "enterprise" {
		t.Fatalf("GetCustomFieldValues: expected enterprise, got %v", values)
	}

	// Upsert semantics: setting again overwrites, does not duplicate.
	if err := repo.SetCustomFieldValues(ctxOwn, d1.ID, map[uuid.UUID]any{segment: "startup"}); err != nil {
		t.Fatalf("SetCustomFieldValues (overwrite): %v", err)
	}
	afterOverwrite, err := repo.GetCustomFieldValues(ctxOwn, d1.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues (after overwrite): %v", err)
	}
	if len(afterOverwrite) != 1 || afterOverwrite[0].Value != "startup" {
		t.Fatalf("GetCustomFieldValues (after overwrite): expected startup, got %v", afterOverwrite)
	}

	batch, err := repo.GetCustomFieldValuesBatch(ctxOwn, []uuid.UUID{d1.ID, d2.ID})
	if err != nil {
		t.Fatalf("GetCustomFieldValuesBatch: %v", err)
	}
	if len(batch[d1.ID]) != 1 || batch[d1.ID][0].Value != "startup" {
		t.Fatalf("GetCustomFieldValuesBatch: d1 mismatch, got %v", batch[d1.ID])
	}
	if len(batch[d2.ID]) != 1 || batch[d2.ID][0].Value != "smb" {
		t.Fatalf("GetCustomFieldValuesBatch: d2 mismatch, got %v", batch[d2.ID])
	}
	if empty, err := repo.GetCustomFieldValuesBatch(ctxOwn, nil); err != nil || empty != nil {
		t.Fatalf("GetCustomFieldValuesBatch (empty): expected (nil, nil), got (%v, %v)", empty, err)
	}
}

func TestRepository_ExistsChecksAndGetStage(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal Exists Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal Exists Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-exists-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	stageOwn := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "Exists Stage " + uuid.New().String()[:8], "is_won": true, "probability": 100,
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageOwn)
	stageForeign := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOther, "name": "Exists Foreign Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageForeign)

	contactOwn := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Exists", "last_name": "Contact", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactOwn)
	companyOwn := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id": tenantOwn, "name": "Exists Co GmbH", "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyOwn)
	dealTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Exists-Deal-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", dealTag)
	contactTag := testutil.SeedRow(t, pool, "tags", map[string]any{
		"id": uuid.New(), "tenant_id": tenantOwn, "name": fmt.Sprintf("Exists-Contact-%s", uuid.New().String()[:8]), "entity_type": "contact",
	})
	defer testutil.CleanupRow(t, pool, "tags", contactTag)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	if exists, err := repo.StageExists(ctxOwn, stageOwn, tenantOwn); err != nil || !exists {
		t.Fatalf("StageExists (own): expected true, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.StageExists(ctxOwn, stageForeign, tenantOwn); err != nil {
		t.Fatalf("StageExists (foreign): %v", err)
	} else if exists {
		t.Fatal("StageExists (foreign): the explicit tenant_id predicate did not stop a foreign-tenant match")
	}

	stage, err := repo.GetStage(ctxOwn, stageOwn)
	if err != nil {
		t.Fatalf("GetStage (own): %v", err)
	}
	if !stage.IsWon || !stage.Probability.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("GetStage (own): expected is_won=true probability=100, got %+v", stage)
	}
	if _, err := repo.GetStage(ctxOwn, uuid.New()); err != ErrStageNotFound {
		t.Fatalf("GetStage (missing): expected ErrStageNotFound, got %v", err)
	}
	// No explicit tenant predicate here either -- RLS is the only guard.
	if _, err := repo.GetStage(ctxOwn, stageForeign); err != ErrStageNotFound {
		t.Fatalf("GetStage (foreign tenant): expected RLS to hide the row as ErrStageNotFound, got %v", err)
	}

	if exists, err := repo.ContactExists(ctxOwn, contactOwn); err != nil || !exists {
		t.Fatalf("ContactExists (own): expected true, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.ContactExists(ctxOwn, uuid.New()); err != nil || exists {
		t.Fatalf("ContactExists (missing): expected false, got exists=%v err=%v", exists, err)
	}

	if exists, err := repo.CompanyExists(ctxOwn, companyOwn); err != nil || !exists {
		t.Fatalf("CompanyExists (own): expected true, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.CompanyExists(ctxOwn, uuid.New()); err != nil || exists {
		t.Fatalf("CompanyExists (missing): expected false, got exists=%v err=%v", exists, err)
	}

	if exists, err := repo.OwnerExists(ctxOwn, userOwn); err != nil || !exists {
		t.Fatalf("OwnerExists (own): expected true, got exists=%v err=%v", exists, err)
	}
	if exists, err := repo.OwnerExists(ctxOwn, uuid.New()); err != nil || exists {
		t.Fatalf("OwnerExists (missing): expected false, got exists=%v err=%v", exists, err)
	}

	if exists, err := repo.TagExists(ctxOwn, dealTag, models.EntityTypeDeal); err != nil || !exists {
		t.Fatalf("TagExists (own, deal): expected true, got exists=%v err=%v", exists, err)
	}
	// Right tag, wrong entity_type.
	if exists, err := repo.TagExists(ctxOwn, contactTag, models.EntityTypeDeal); err != nil {
		t.Fatalf("TagExists (wrong entity type): %v", err)
	} else if exists {
		t.Fatal("TagExists (wrong entity type): a contact tag matched the deal entity type")
	}
}

func TestRepository_SetClosedAt_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal ClosedAt Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal ClosedAt Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantOwn, "email": fmt.Sprintf("crm-deal-closedat-%s@tenantown.local", uuid.New().String()[:8]), "password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn, "name": "ClosedAt Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	d := &models.Deal{ID: uuid.New(), TenantID: tenantOwn, Name: "Closing Deal", Value: decimal.NewFromInt(1), Currency: "EUR", StageID: stageID, CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctxOwn, d); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "deals", d.ID)

	// The query carries an explicit tenant_id predicate (unlike the relation
	// lookups above), so a foreign-ctx call with the real tenantID as the
	// explicit argument must fail on RowsAffected()==0, not just rely on RLS.
	if err := repo.SetClosedAt(ctxOther, d.ID, tenantOwn, &now); err != ErrDealNotFound {
		t.Fatalf("SetClosedAt (foreign ctx): expected ErrDealNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, d.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ClosedAt != nil {
		t.Fatalf("SetClosedAt (foreign ctx): closed_at was set despite the error: %v", got.ClosedAt)
	}

	// Postgres timestamptz only stores microsecond precision, so round-tripping
	// a Go nanosecond-precision time.Time through the DB truncates it.
	closedAt := now.Add(time.Hour).Truncate(time.Microsecond)
	if err := repo.SetClosedAt(ctxOwn, d.ID, tenantOwn, &closedAt); err != nil {
		t.Fatalf("SetClosedAt (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, d.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (after close): %v", err)
	}
	if got.ClosedAt == nil || !got.ClosedAt.Equal(closedAt) {
		t.Fatalf("SetClosedAt (own ctx): expected closed_at=%v, got %v", closedAt, got.ClosedAt)
	}

	// Reopening (nil) must clear closed_at, not merely no-op.
	if err := repo.SetClosedAt(ctxOwn, d.ID, tenantOwn, nil); err != nil {
		t.Fatalf("SetClosedAt (reopen): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, d.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (after reopen): %v", err)
	}
	if got.ClosedAt != nil {
		t.Fatalf("SetClosedAt (reopen): expected closed_at=nil, got %v", got.ClosedAt)
	}
}
