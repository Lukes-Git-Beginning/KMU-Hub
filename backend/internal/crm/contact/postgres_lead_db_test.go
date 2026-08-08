package contact

// DB integration tests for postgres_lead.go, which carried no DB coverage at
// all before this file: ListLeads' default-stage/status/search filters, its
// tenant-scoped count, and UpdateLead's partial patch plus the
// lifecycle_stage <> customer guard that keeps this endpoint from becoming a
// side door onto ordinary contacts.

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

func TestRepository_ListLeads_DefaultStageStatusSearchAndTenantScopedCount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Lead List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Lead List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-lead-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-lead-list-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	newStatus, contactedStatus := LeadStatusNew, LeadStatusContacted
	dialerSource := LeadSourceDialer
	leadCompany := fmt.Sprintf("Findable-Employer-%s", uuid.New().String()[:8])

	newLead := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "New", LastName: "Lead", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleLead, LeadSource: &dialerSource, LeadStatus: &newStatus, LeadCompany: &leadCompany,
	}
	contactedLead := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "Contacted", LastName: "Lead", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleLead, LeadSource: &dialerSource, LeadStatus: &contactedStatus,
	}
	qualified := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "Qualified", LastName: "Prospect", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleQualified, LeadSource: &dialerSource, LeadStatus: &newStatus,
	}
	customer := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "Ordinary", LastName: "Customer", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleCustomer,
	}
	for _, c := range []*models.Contact{newLead, contactedLead, qualified, customer} {
		if err := repo.Create(ctxOwn, c); err != nil {
			t.Fatalf("Create %s: %v", c.FirstName, err)
		}
		defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	}
	foreignLead := &models.Contact{
		ID: uuid.New(), TenantID: tenantOther, FirstName: "Foreign", LastName: "Lead", Visibility: "shared",
		CreatedBy: userOther, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleLead, LeadSource: &dialerSource, LeadStatus: &newStatus,
	}
	if err := repo.Create(ctxOther, foreignLead); err != nil {
		t.Fatalf("Create foreignLead: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", foreignLead.ID)

	// Default stage ("") is the open inbox: lead + qualified, never customer.
	inbox, inboxTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn}, 0, 20)
	if err != nil {
		t.Fatalf("ListLeads (default stage): %v", err)
	}
	inboxIDs := make(map[uuid.UUID]bool, len(inbox))
	for _, l := range inbox {
		inboxIDs[l.ID] = true
	}
	if inboxTotal != 3 || inboxIDs[customer.ID] || !inboxIDs[newLead.ID] || !inboxIDs[qualified.ID] {
		t.Fatalf("ListLeads (default stage): expected lead+lead+qualified (3), no customer, got total=%d ids=%v", inboxTotal, inboxIDs)
	}
	if inboxIDs[foreignLead.ID] {
		t.Fatal("ListLeads (default stage): leaked a foreign-tenant lead")
	}

	// Explicit stage filter narrows to exactly "lead".
	leadsOnly, leadsTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn, Stage: LifecycleLead}, 0, 20)
	if err != nil {
		t.Fatalf("ListLeads (stage=lead): %v", err)
	}
	if leadsTotal != 2 {
		t.Fatalf("ListLeads (stage=lead): expected 2, got %d", leadsTotal)
	}
	for _, l := range leadsOnly {
		if l.ID == qualified.ID {
			t.Fatal("ListLeads (stage=lead): a qualified contact leaked into the lead-only filter")
		}
	}

	// Status filter.
	contactedOnly, contactedTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn, Status: LeadStatusContacted}, 0, 20)
	if err != nil {
		t.Fatalf("ListLeads (status=contacted): %v", err)
	}
	if contactedTotal != 1 || len(contactedOnly) != 1 || contactedOnly[0].ID != contactedLead.ID {
		t.Fatalf("ListLeads (status=contacted): expected exactly contactedLead, got total=%d results=%v", contactedTotal, contactedOnly)
	}

	// Search matches lead_company (free-text employer), case-insensitively.
	searchResults, searchTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn, Search: "findable-employer"}, 0, 20)
	if err != nil {
		t.Fatalf("ListLeads (search): %v", err)
	}
	if searchTotal != 1 || len(searchResults) != 1 || searchResults[0].ID != newLead.ID {
		t.Fatalf("ListLeads (search): expected exactly newLead, got total=%d results=%v", searchTotal, searchResults)
	}
	if searchResults[0].CompanyName == nil || *searchResults[0].CompanyName != leadCompany {
		t.Fatalf("ListLeads (search): company_name should fall back to lead_company, got %v", searchResults[0].CompanyName)
	}

	// Pagination: page size 1 still reports the full tenant-scoped total (3
	// for the default open-inbox stage), same "count carries the same
	// condition as the page" property checked on the plain contact List.
	page, pageTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn}, 0, 1)
	if err != nil {
		t.Fatalf("ListLeads (page 1): %v", err)
	}
	if pageTotal != 3 || len(page) != 1 {
		t.Fatalf("ListLeads (page 1): expected total=3 page-len=1, got total=%d page-len=%d", pageTotal, len(page))
	}

	// Offset beyond the total: empty page, unchanged total, no error.
	empty, emptyTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn}, 50, 20)
	if err != nil {
		t.Fatalf("ListLeads (offset beyond total): unexpected error %v", err)
	}
	if emptyTotal != 3 || len(empty) != 0 {
		t.Fatalf("ListLeads (offset beyond total): expected 0 rows total=3, got %d rows total=%d", len(empty), emptyTotal)
	}
}

func TestRepository_UpdateLead_PartialPatchTemperatureClearAndCustomerGuard(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Lead Update Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Lead Update Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-lead-update-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC()

	manualSource := LeadSourceManual
	newStatus := LeadStatusNew
	lead := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "Update", LastName: "Lead", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleLead, LeadSource: &manualSource, LeadStatus: &newStatus,
	}
	if err := repo.Create(ctxOwn, lead); err != nil {
		t.Fatalf("Create lead: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", lead.ID)

	customer := &models.Contact{
		ID: uuid.New(), TenantID: tenantOwn, FirstName: "Untouchable", LastName: "Customer", Visibility: "shared",
		CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
		LifecycleStage: LifecycleCustomer,
	}
	if err := repo.Create(ctxOwn, customer); err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", customer.ID)

	// UpdateLead's WHERE clause excludes customers on purpose: it must not
	// become a side door for editing ordinary contacts.
	newSourceForCustomer := LeadSourceCSV
	if _, err := repo.UpdateLead(ctxOwn, customer.ID, tenantOwn, LeadPatch{Source: &newSourceForCustomer}); !errors.Is(err, ErrLeadNotFound) {
		t.Fatalf("UpdateLead (customer): expected ErrLeadNotFound, got %v", err)
	}
	var customerSourceAfter *string
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()), `SELECT lead_source FROM contacts WHERE id = $1`, customer.ID).Scan(&customerSourceAfter); err != nil {
		t.Fatalf("read customer after blocked update: %v", err)
	}
	if customerSourceAfter != nil {
		t.Fatalf("UpdateLead wrote through the customer guard: lead_source=%v", *customerSourceAfter)
	}

	// A tenant-scoped id that does not exist for this tenant: ErrLeadNotFound.
	if _, err := repo.UpdateLead(ctxOwn, uuid.New(), tenantOwn, LeadPatch{Source: &newSourceForCustomer}); !errors.Is(err, ErrLeadNotFound) {
		t.Fatalf("UpdateLead (missing id): expected ErrLeadNotFound, got %v", err)
	}
	if _, err := repo.UpdateLead(ctxOwn, lead.ID, tenantOther, LeadPatch{Source: &newSourceForCustomer}); !errors.Is(err, ErrLeadNotFound) {
		t.Fatalf("UpdateLead (foreign tenant id): expected ErrLeadNotFound, got %v", err)
	}

	// Partial patch: only Stage + Status change, Source/Score/Temperature stay put.
	qualifiedStage, qualifiedStatus := LifecycleQualified, LeadStatusQualified
	updated, err := repo.UpdateLead(ctxOwn, lead.ID, tenantOwn, LeadPatch{Stage: &qualifiedStage, Status: &qualifiedStatus})
	if err != nil {
		t.Fatalf("UpdateLead (stage+status): %v", err)
	}
	if updated.LifecycleStage != LifecycleQualified || updated.LeadStatus == nil || *updated.LeadStatus != LeadStatusQualified {
		t.Fatalf("UpdateLead (stage+status): got stage=%q status=%v", updated.LifecycleStage, updated.LeadStatus)
	}
	if updated.LeadSource == nil || *updated.LeadSource != LeadSourceManual {
		t.Fatalf("UpdateLead (stage+status): unrelated field lead_source changed, got %v", updated.LeadSource)
	}

	// Manual temperature override, then Score update, then clearing the
	// override hands control back to the score-derived value.
	hot := LeadTemperatureHot
	var score16 int16 = 80
	withTemp, err := repo.UpdateLead(ctxOwn, lead.ID, tenantOwn, LeadPatch{Temperature: &hot, Score: &score16})
	if err != nil {
		t.Fatalf("UpdateLead (temperature+score): %v", err)
	}
	if withTemp.LeadTemperature == nil || *withTemp.LeadTemperature != LeadTemperatureHot {
		t.Fatalf("UpdateLead (temperature+score): expected temperature=hot, got %v", withTemp.LeadTemperature)
	}
	if withTemp.LeadScore == nil || *withTemp.LeadScore != 80 {
		t.Fatalf("UpdateLead (temperature+score): expected score=80, got %v", withTemp.LeadScore)
	}

	cleared, err := repo.UpdateLead(ctxOwn, lead.ID, tenantOwn, LeadPatch{ClearTemperature: true})
	if err != nil {
		t.Fatalf("UpdateLead (clear temperature): %v", err)
	}
	if cleared.LeadTemperature != nil {
		t.Fatalf("UpdateLead (clear temperature): expected nil, got %v", *cleared.LeadTemperature)
	}
	// Score set by the previous patch must survive an unrelated patch.
	if cleared.LeadScore == nil || *cleared.LeadScore != 80 {
		t.Fatalf("UpdateLead (clear temperature): unrelated field lead_score changed, got %v", cleared.LeadScore)
	}
}
