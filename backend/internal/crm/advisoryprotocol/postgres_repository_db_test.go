package advisoryprotocol

// DB integration tests for postgres_repository.go, which carried 0% coverage
// before this file (service_test.go only exercises the mock repository).
// The property that matters most for a MiFID II / §64 WpHG record with a
// ten-year retention duty is immutability after hand-over — and it must hold
// on every repository write path, not just the service-level precondition
// check that GetByID + status=="finalized" performs before calling Update or
// Delete. This file drives Update/Delete directly against an already
// finalized row (bypassing the service) to prove the repository itself
// refuses the write.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func fullProtocol(id, tenantID, contactID, createdBy uuid.UUID) *Protocol {
	date := "2026-03-15"
	birth := "1980-01-01"
	selfAssessment := 4
	income := 4200.50
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &Protocol{
		ID:        id,
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: createdBy,
		Status:    "draft",

		Date:             &date,
		TimeFrom:         "09:00",
		TimeTo:           "10:00",
		Location:         "office",
		Advisor:          "Erika Musterberaterin",
		Occasion:         "initial",
		OccasionNote:     "Erstberatung",
		CustomerCategory: "private",

		BirthDate:     &birth,
		MaritalStatus: "married",
		TaxStatus:     "standard",

		KnownAssetClasses:     []string{"stocks", "etf"},
		PastTransactions:      "Aktienfonds seit 2015",
		FinancialEducation:    "Bankkaufmann",
		ProfessionalExperience: "10 Jahre Finanzbranche",
		SelfAssessment:        &selfAssessment,

		MonthlyNetIncome: &income,
		RealEstate:       "Eigentumswohnung",
		ExistingInsurance: "Berufsunfähigkeit",

		InvestmentPurpose: []string{"retirement", "growth"},
		Horizon:           "gt10",
		RiskTolerance:     "medium",
		RiskCapacity:      "medium",
		RiskClass:         4,
		EsgPreference:     true,
		EsgDetails:        "Nachhaltige Fonds bevorzugt",

		Products: []Product{
			{
				ID:           "p1",
				Name:         "Global Equity ETF",
				RiskClass:    4,
				Risks:        "Marktrisiko",
				Recommended:  true,
			},
		},

		RecommendationSummary: "Breit gestreutes ETF-Portfolio",
		SuitabilityReasoning:  "Passt zu Zielen und Risikoprofil",
		GoalReference:         "Altersvorsorge",

		MainConcerns:         "Keine",
		WarningsGiven:        []string{"risk", "costs"},
		DocumentDelivered:    true,
		DeliveryForm:         "email",
		AdvisorSignature:     "E. Musterberaterin",
		CustomerConfirmation: true,

		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestRepository_CreateGetByID_RoundTripsAndScopesToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol RoundTrip Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "AdvisoryProtocol RoundTrip Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-roundtrip-%s@tenantown.local", uuid.New().String()[:8]),
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

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	got, err := repo.GetByID(ctxOwn, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own tenant): %v", err)
	}
	if got.Advisor != p.Advisor || got.RiskClass != p.RiskClass || *got.Date != *p.Date {
		t.Fatalf("GetByID: scalar fields did not round-trip, got %+v", got)
	}
	if len(got.KnownAssetClasses) != 2 || got.KnownAssetClasses[0] != "stocks" {
		t.Fatalf("GetByID: known_asset_classes did not round-trip, got %v", got.KnownAssetClasses)
	}
	if len(got.InvestmentPurpose) != 2 || len(got.WarningsGiven) != 2 {
		t.Fatalf("GetByID: text arrays did not round-trip, got purpose=%v warnings=%v", got.InvestmentPurpose, got.WarningsGiven)
	}
	if len(got.Products) != 1 || got.Products[0].Name != "Global Equity ETF" || !got.Products[0].Recommended {
		t.Fatalf("GetByID: products JSONB did not round-trip, got %+v", got.Products)
	}
	if got.SelfAssessment == nil || *got.SelfAssessment != 4 {
		t.Fatalf("GetByID: self_assessment pointer did not round-trip, got %v", got.SelfAssessment)
	}
	if got.MonthlyNetIncome == nil || *got.MonthlyNetIncome != 4200.50 {
		t.Fatalf("GetByID: monthly_net_income did not round-trip, got %v", got.MonthlyNetIncome)
	}

	// RLS: a foreign tenant session must not see the row even with the correct id.
	if _, err := repo.GetByID(ctxOther, p.ID, tenantOther); err != ErrProtocolNotFound {
		t.Fatalf("GetByID (foreign tenant session): expected ErrProtocolNotFound, got %v", err)
	}
	// A stolen id with the wrong tenant parameter under the owning session must also fail.
	if _, err := repo.GetByID(ctxOwn, p.ID, tenantOther); err != ErrProtocolNotFound {
		t.Fatalf("GetByID (wrong tenant param): expected ErrProtocolNotFound, got %v", err)
	}
	if _, err := repo.GetByID(ctxOwn, uuid.New(), tenantOwn); err != ErrProtocolNotFound {
		t.Fatalf("GetByID (unknown id): expected ErrProtocolNotFound, got %v", err)
	}
}

// TestRepository_CreateGetByID_BlankDraftRoundTrips reproduces the actual
// Service.Create() shape: TimeFrom/TimeTo/Location/Occasion/CustomerCategory
// and all four DATE fields start unset. Those five columns are NULLable
// VARCHAR/TIME columns bound to non-pointer Go string fields, and the four
// DATE columns decode in Postgres binary wire format, which pgx cannot
// scan into *string without a cast — before the fix in this commit,
// GetByID crashed immediately after every single protocol creation.
func TestRepository_CreateGetByID_BlankDraftRoundTrips(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Blank Draft Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-blank-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Blank",
		"last_name":  "Draft",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	now := time.Now().UTC().Truncate(time.Microsecond)

	p := &Protocol{
		ID:                uuid.New(),
		TenantID:          tenantOwn,
		ContactID:         contactID,
		CreatedBy:         userOwn,
		Status:            "draft",
		KnownAssetClasses: []string{},
		InvestmentPurpose: []string{},
		WarningsGiven:     []string{},
		Products:          []Product{},
		RiskClass:         3,
		DeliveryForm:      "email",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create (blank draft): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	got, err := repo.GetByID(ctxOwn, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (blank draft, the actual post-Create flow): %v", err)
	}
	if got.TimeFrom != "" || got.TimeTo != "" || got.Location != "" || got.Occasion != "" || got.CustomerCategory != "" {
		t.Fatalf("GetByID (blank draft): expected empty strings for unset nullable fields, got %+v", got)
	}
	if got.Date != nil || got.BirthDate != nil || got.DocumentDeliveredDate != nil || got.FollowupDate != nil {
		t.Fatalf("GetByID (blank draft): expected nil pointers for unset date fields, got %+v", got)
	}

	list, err := repo.ListByContact(ctxOwn, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ListByContact (blank draft): %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByContact (blank draft): expected 1 protocol, got %d", len(list))
	}
}

func TestRepository_ListByContact_OrdersByDateDescAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol List Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "AdvisoryProtocol List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-list-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "List",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)
	foreignContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	older := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	older.Date = strPtr("2026-01-10")
	newer := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	newer.Date = strPtr("2026-02-20")
	noDate := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	noDate.Date = nil

	for _, p := range []*Protocol{older, newer, noDate} {
		if err := repo.Create(ctxOwn, p); err != nil {
			t.Fatalf("Create %s: %v", p.ID, err)
		}
		defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)
	}

	foreignProtocol := fullProtocol(uuid.New(), tenantOther, foreignContactID, userOwn)
	if err := repo.Create(ctxOther, foreignProtocol); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", foreignProtocol.ID)

	list, err := repo.ListByContact(ctxOwn, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ListByContact: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListByContact: expected 3 protocols, got %d", len(list))
	}
	// newer (2026-02-20) before older (2026-01-10); noDate falls back to
	// created_at::date and was created last, so it sorts first among ties.
	if list[0].ID != noDate.ID || list[1].ID != newer.ID || list[2].ID != older.ID {
		t.Fatalf("ListByContact: unexpected order %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
	}

	// A stolen contact id under a foreign session must not leak via RLS, even
	// though the WHERE clause alone would match tenant_id=tenantOwn.
	leaked, err := repo.ListByContact(ctxOther, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ListByContact (foreign session, stolen tenant param): %v", err)
	}
	if len(leaked) != 0 {
		t.Fatalf("ListByContact (foreign session): expected 0 rows, RLS leaked %d", len(leaked))
	}
}

func TestRepository_Update_DraftAppliesAllFieldsIncludingProductsJSON(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Update Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-update-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Update",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	p.Advisor = "Neuer Berater"
	p.RiskClass = 6
	p.Products = []Product{{ID: "p2", Name: "Bond Fund", RiskClass: 2, Risks: "Zinsrisiko"}}
	p.KnownAssetClasses = []string{"bonds"}
	p.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)

	if err := repo.Update(ctxOwn, p); err != nil {
		t.Fatalf("Update (draft): %v", err)
	}

	got, err := repo.GetByID(ctxOwn, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got.Advisor != "Neuer Berater" || got.RiskClass != 6 {
		t.Fatalf("Update: scalar fields not applied, got %+v", got)
	}
	if len(got.Products) != 1 || got.Products[0].Name != "Bond Fund" {
		t.Fatalf("Update: products not applied, got %+v", got.Products)
	}
	if len(got.KnownAssetClasses) != 1 || got.KnownAssetClasses[0] != "bonds" {
		t.Fatalf("Update: known_asset_classes not applied, got %v", got.KnownAssetClasses)
	}
}

func TestRepository_Update_OnFinalizedReturnsErrProtocolFinalizedAndDoesNotMutate(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Update Finalized Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-update-final-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Final",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	// Finalize through the real write path (HandOver), not a manual UPDATE —
	// this is what a concurrent request would do between the service's
	// GetByID precondition check and the attacker/racer's Update call.
	if err := repo.HandOver(ctxOwn, p.ID, tenantOwn, time.Now().UTC()); err != nil {
		t.Fatalf("HandOver: %v", err)
	}

	attempt := fullProtocol(p.ID, tenantOwn, contactID, userOwn)
	attempt.Advisor = "should not be persisted"
	attempt.RiskClass = 1
	if err := repo.Update(ctxOwn, attempt); err != ErrProtocolFinalized {
		t.Fatalf("Update (finalized): expected ErrProtocolFinalized, got %v", err)
	}

	got, err := repo.GetByID(ctxOwn, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID after rejected update: %v", err)
	}
	if got.Advisor != p.Advisor || got.RiskClass != p.RiskClass {
		t.Fatalf("Update (finalized): row was mutated despite ErrProtocolFinalized, got %+v", got)
	}
	if got.Status != "finalized" {
		t.Fatalf("expected status to remain finalized, got %s", got.Status)
	}
}

func TestRepository_Delete_DraftRemovesRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-delete-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Delete",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctxOwn, p.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (draft): %v", err)
	}
	if _, err := repo.GetByID(ctxOwn, p.ID, tenantOwn); err != ErrProtocolNotFound {
		t.Fatalf("GetByID after delete: expected ErrProtocolNotFound, got %v", err)
	}
}

func TestRepository_Delete_OnFinalizedReturnsErrProtocolFinalizedAndKeepsRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Delete Finalized Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-delete-final-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "DeleteFinal",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	if err := repo.HandOver(ctxOwn, p.ID, tenantOwn, time.Now().UTC()); err != nil {
		t.Fatalf("HandOver: %v", err)
	}

	if err := repo.Delete(ctxOwn, p.ID, tenantOwn); err != ErrProtocolFinalized {
		t.Fatalf("Delete (finalized): expected ErrProtocolFinalized, got %v", err)
	}

	if _, err := repo.GetByID(ctxOwn, p.ID, tenantOwn); err != nil {
		t.Fatalf("GetByID after rejected delete: expected the row to still exist, got %v", err)
	}
}

func TestRepository_HandOver_SetsFinalizedStatusAndHandedOverAt(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol HandOver Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-handover-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "HandOver",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	p := fullProtocol(uuid.New(), tenantOwn, contactID, userOwn)
	if err := repo.Create(ctxOwn, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "advisory_protocols", p.ID)

	at := time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.HandOver(ctxOwn, p.ID, tenantOwn, at); err != nil {
		t.Fatalf("HandOver: %v", err)
	}

	got, err := repo.GetByID(ctxOwn, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID after HandOver: %v", err)
	}
	if got.Status != "finalized" {
		t.Fatalf("expected status=finalized, got %s", got.Status)
	}
	if got.HandedOverAt == nil || !got.HandedOverAt.Equal(at) {
		t.Fatalf("expected handed_over_at=%v, got %v", at, got.HandedOverAt)
	}
}

func TestRepository_ContactExists_IsTenantScopedBothWaysByParamAndByRLS(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol ContactExists Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "AdvisoryProtocol ContactExists Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-contactexists-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Exists",
		"last_name":  "Kontakt",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	exists, err := repo.ContactExists(ctxOwn, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (own): %v", err)
	}
	if !exists {
		t.Fatal("ContactExists (own): expected true")
	}

	// Wrong tenant parameter under the owning session: WHERE clause rejects it.
	exists, err = repo.ContactExists(ctxOwn, contactID, tenantOther)
	if err != nil {
		t.Fatalf("ContactExists (wrong tenant param): %v", err)
	}
	if exists {
		t.Fatal("ContactExists (wrong tenant param): expected false")
	}

	// Correct tenant parameter but a foreign session: RLS rejects it.
	exists, err = repo.ContactExists(ctxOther, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (foreign session): %v", err)
	}
	if exists {
		t.Fatal("ContactExists (foreign session): expected false, RLS leaked the row")
	}
}

func TestRepository_GetReferralReport_AggregatesPerReferrerAndScopesToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "AdvisoryProtocol Referral Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "AdvisoryProtocol Referral Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("advisory-referral-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	referrer := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Referrer",
		"last_name":  "Person",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", referrer)
	referred1 := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":              tenantOwn,
		"first_name":             "Referred",
		"last_name":              "One",
		"created_by":             userOwn,
		"referred_by_contact_id": referrer,
	})
	defer testutil.CleanupRow(t, pool, "contacts", referred1)
	referred2 := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":              tenantOwn,
		"first_name":             "Referred",
		"last_name":              "Two",
		"created_by":             userOwn,
		"referred_by_contact_id": referrer,
	})
	defer testutil.CleanupRow(t, pool, "contacts", referred2)

	// A referral in another tenant must never contaminate tenantOwn's report.
	foreignReferrer := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Referrer",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignReferrer)
	foreignReferred := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":              tenantOther,
		"first_name":             "Foreign",
		"last_name":              "Referred",
		"created_by":             userOwn,
		"referred_by_contact_id": foreignReferrer,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignReferred)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	report, err := repo.GetReferralReport(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("GetReferralReport: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("GetReferralReport: expected exactly 1 referrer, got %d (%+v)", len(report), report)
	}
	if report[0].ReferrerID != referrer || report[0].ReferredCount != 2 {
		t.Fatalf("GetReferralReport: expected referrer=%s count=2, got %+v", referrer, report[0])
	}
	if report[0].ReferrerFirstName != "Referrer" || report[0].ReferrerLastName != "Person" {
		t.Fatalf("GetReferralReport: unexpected referrer name %+v", report[0])
	}
}

func strPtr(s string) *string { return &s }
