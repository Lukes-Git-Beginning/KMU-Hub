package contact

// Lead lifecycle tests. The DB-backed ones follow the pattern in
// tenant_write_test.go: every read and write is exercised once from a foreign
// tenant's session with the row's *real* tenantID passed explicitly, so only
// RLS can be stopping it -- then repeated from the owning session.
//
// Run with: DATABASE_URL='postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable' go test -count=1 ./internal/crm/contact

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestComputeLeadScore(t *testing.T) {
	tests := []struct {
		name string
		in   LeadScoreInput
		want int16
	}{
		{"dialer, nothing filled", LeadScoreInput{Source: LeadSourceDialer}, 35},
		{"csv, nothing filled", LeadScoreInput{Source: LeadSourceCSV}, 10},
		{"manual with email and phone", LeadScoreInput{Source: LeadSourceManual, Email: "a@b.de", Phone: "+49 30 1"}, 60},
		{"unknown source falls back", LeadScoreInput{Source: "linkedin"}, 15},
		{"whitespace does not count as filled", LeadScoreInput{Source: LeadSourceCSV, Email: "   ", Notes: "\t"}, 10},
		{"everything filled clamps at 100", LeadScoreInput{
			Source: LeadSourceDialer, Email: "a@b.de", Phone: "+49 30 1", Company: "ACME", Notes: "callback",
		}, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeLeadScore(tc.in); got != tc.want {
				t.Fatalf("ComputeLeadScore = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestScoreToTemperature(t *testing.T) {
	tests := []struct {
		score int16
		want  string
	}{
		{100, LeadTemperatureHot},
		{66, LeadTemperatureHot},
		{65, LeadTemperatureWarm},
		{33, LeadTemperatureWarm},
		{32, LeadTemperatureCold},
		{0, LeadTemperatureCold},
	}
	for _, tc := range tests {
		if got := ScoreToTemperature(tc.score); got != tc.want {
			t.Fatalf("ScoreToTemperature(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestEffectiveTemperature_OverrideWins(t *testing.T) {
	score := int16(90) // would be hot
	override := LeadTemperatureCold

	if got := EffectiveTemperature(&score, &override); got != LeadTemperatureCold {
		t.Fatalf("override ignored: got %q", got)
	}
	if got := EffectiveTemperature(&score, nil); got != LeadTemperatureHot {
		t.Fatalf("without an override the score should decide: got %q", got)
	}
	empty := ""
	if got := EffectiveTemperature(&score, &empty); got != LeadTemperatureHot {
		t.Fatalf("an empty override is not an override: got %q", got)
	}
	if got := EffectiveTemperature(nil, nil); got != LeadTemperatureCold {
		t.Fatalf("a lead without a score is cold, got %q", got)
	}
}

// seedLead inserts a lead directly so the read tests do not depend on the
// write path, and returns its id.
func seedLead(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, lastName, status string, score int16) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":              uuid.New(),
		"tenant_id":       tenantID,
		"first_name":      "Lead",
		"last_name":       lastName,
		"created_by":      userID,
		"lifecycle_stage": LifecycleLead,
		"lead_source":     LeadSourceDialer,
		"lead_score":      score,
		"lead_status":     status,
		"lead_company":    lastName + " GmbH",
	})
}

func TestLeads_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Lead Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Lead Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-lead-own-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	userOther := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("crm-lead-other-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOther)

	// Own tenant: two leads and one ordinary contact. The contact must never
	// show up in the inbox.
	leadNew := seedLead(t, pool, tenantOwn, userOwn, "Vogel", LeadStatusNew, 70)
	defer testutil.CleanupRow(t, pool, "contacts", leadNew)
	leadContacted := seedLead(t, pool, tenantOwn, userOwn, "Brandt", LeadStatusContacted, 40)
	defer testutil.CleanupRow(t, pool, "contacts", leadContacted)

	customer := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOwn,
		"first_name": "Regular",
		"last_name":  "Customer",
		"created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "contacts", customer)

	// Foreign tenant gets one lead of its own, so "sees nothing" cannot be
	// confused with "the query is broken".
	foreignLead := seedLead(t, pool, tenantOther, userOther, "Fremd", LeadStatusNew, 55)
	defer testutil.CleanupRow(t, pool, "contacts", foreignLead)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// --- Read side -----------------------------------------------------
	leads, total, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn}, 0, 50)
	if err != nil {
		t.Fatalf("ListLeads (own ctx): %v", err)
	}
	if total != 2 || len(leads) != 2 {
		t.Fatalf("owner should see exactly its 2 leads (the customer row is not one): total=%d len=%d", total, len(leads))
	}
	for _, l := range leads {
		if l.ID == customer {
			t.Fatalf("an ordinary contact leaked into the lead inbox")
		}
		if l.LifecycleStage != LifecycleLead {
			t.Fatalf("unexpected stage %q in the inbox", l.LifecycleStage)
		}
		if l.CompanyName == nil || *l.CompanyName == "" {
			t.Fatalf("lead_company did not surface as company_name for %s", l.ID)
		}
	}

	// Same call from the foreign session, passing the victim's real tenantID:
	// only RLS can be stopping this.
	stolen, stolenTotal, err := repo.ListLeads(ctxOther, LeadFilter{TenantID: tenantOwn}, 0, 50)
	if err != nil {
		t.Fatalf("ListLeads (foreign ctx): %v", err)
	}
	if stolenTotal != 0 || len(stolen) != 0 {
		t.Fatalf("a foreign tenant read %d of the owner's leads", len(stolen))
	}

	own, ownTotal, err := repo.ListLeads(ctxOther, LeadFilter{TenantID: tenantOther}, 0, 50)
	if err != nil {
		t.Fatalf("ListLeads (foreign ctx, own tenant): %v", err)
	}
	if ownTotal != 1 || len(own) != 1 || own[0].ID != foreignLead {
		t.Fatalf("the other tenant should see exactly its own lead, got %d", len(own))
	}

	// Status filter narrows within the tenant.
	filtered, filteredTotal, err := repo.ListLeads(ctxOwn, LeadFilter{TenantID: tenantOwn, Status: LeadStatusContacted}, 0, 50)
	if err != nil {
		t.Fatalf("ListLeads (status filter): %v", err)
	}
	if filteredTotal != 1 || len(filtered) != 1 || filtered[0].ID != leadContacted {
		t.Fatalf("status filter returned %d rows, want just the contacted one", len(filtered))
	}

	// --- Write side ----------------------------------------------------
	qualified := LifecycleQualified
	statusQualified := LeadStatusQualified
	if _, err := repo.UpdateLead(ctxOther, leadNew, tenantOwn, LeadPatch{Stage: &qualified, Status: &statusQualified}); !errors.Is(err, ErrLeadNotFound) {
		t.Fatalf("UpdateLead (foreign ctx) should have found nothing to update, got %v", err)
	}

	updated, err := repo.UpdateLead(ctxOwn, leadNew, tenantOwn, LeadPatch{Stage: &qualified, Status: &statusQualified})
	if err != nil {
		t.Fatalf("UpdateLead (own ctx): %v", err)
	}
	if updated.LifecycleStage != LifecycleQualified {
		t.Fatalf("stage did not move: %q", updated.LifecycleStage)
	}
	if updated.LeadStatus == nil || *updated.LeadStatus != LeadStatusQualified {
		t.Fatalf("status did not move: %v", updated.LeadStatus)
	}

	// A customer row must not be editable through the lead endpoint.
	if _, err := repo.UpdateLead(ctxOwn, customer, tenantOwn, LeadPatch{Status: &statusQualified}); !errors.Is(err, ErrLeadNotFound) {
		t.Fatalf("the lead endpoint edited an ordinary contact, got %v", err)
	}

	// Temperature override round-trips, and clearing it wins over setting it.
	cold := LeadTemperatureCold
	pinned, err := repo.UpdateLead(ctxOwn, leadContacted, tenantOwn, LeadPatch{Temperature: &cold})
	if err != nil {
		t.Fatalf("UpdateLead (temperature): %v", err)
	}
	if pinned.LeadTemperature == nil || *pinned.LeadTemperature != LeadTemperatureCold {
		t.Fatalf("temperature override did not persist: %v", pinned.LeadTemperature)
	}
	cleared, err := repo.UpdateLead(ctxOwn, leadContacted, tenantOwn, LeadPatch{Temperature: &cold, ClearTemperature: true})
	if err != nil {
		t.Fatalf("UpdateLead (clear temperature): %v", err)
	}
	if cleared.LeadTemperature != nil {
		t.Fatalf("clearing the override left %q behind", *cleared.LeadTemperature)
	}
}

func TestCreateLead_PersistsComputedScore(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Lead Create Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("crm-lead-create-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := svc.CreateLead(ctx, CreateLeadInput{
		FirstName: "Sabine",
		LastName:  "Vogel",
		Company:   "Vogel Dachtechnik GmbH",
		Email:     fmt.Sprintf("sabine-%s@vogel-dach.invalid", uuid.New().String()[:8]),
		Phone:     "+49 711 4455660",
		Notes:     "Rueckruf gewuenscht",
		Source:    LeadSourceDialer,
		CreatedBy: userID,
		TenantID:  tenantID,
	})
	if err != nil {
		t.Fatalf("CreateLead: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", created.ID)

	if created.LifecycleStage != LifecycleLead {
		t.Fatalf("a new lead should start at stage lead, got %q", created.LifecycleStage)
	}
	if created.LeadScore == nil || *created.LeadScore != 100 {
		t.Fatalf("score should be computed server-side (dialer + all fields = 100), got %v", created.LeadScore)
	}
	if created.LeadStatus == nil || *created.LeadStatus != LeadStatusNew {
		t.Fatalf("a new lead should be status new, got %v", created.LeadStatus)
	}
	if created.LeadCompany == nil || *created.LeadCompany != "Vogel Dachtechnik GmbH" {
		t.Fatalf("the free-text employer was dropped: %v", created.LeadCompany)
	}
	if created.LeadTemperature != nil {
		t.Fatalf("a fresh lead has no manual override, got %q", *created.LeadTemperature)
	}

	// It is a lead, so it reaches the inbox -- and it is a contact, so the
	// contact list still finds it (that is the point of one row per person).
	leads, total, err := repo.ListLeads(ctx, LeadFilter{TenantID: tenantID}, 0, 10)
	if err != nil {
		t.Fatalf("ListLeads: %v", err)
	}
	if total != 1 || len(leads) != 1 || leads[0].ID != created.ID {
		t.Fatalf("the created lead is not in the inbox: total=%d", total)
	}

	asContact, err := repo.GetByID(ctx, created.ID, tenantID)
	if err != nil {
		t.Fatalf("GetByID: the lead is not readable as a contact: %v", err)
	}
	if asContact.LastName != "Vogel" {
		t.Fatalf("unexpected contact row: %+v", asContact)
	}

	// Convert is a stage change on that same row, not a copy.
	converted, err := svc.ConvertLead(ctx, created.ID, tenantID)
	if err != nil {
		t.Fatalf("ConvertLead: %v", err)
	}
	if converted.ID != created.ID {
		t.Fatalf("convert produced a second row: %s != %s", converted.ID, created.ID)
	}
	if converted.LifecycleStage != LifecycleQualified {
		t.Fatalf("convert did not qualify the lead: %q", converted.LifecycleStage)
	}

	var rowCount int
	sysCtx := testutil.WithSystemCtx(context.Background())
	if err := pool.QueryRow(sysCtx, "SELECT COUNT(*) FROM contacts WHERE tenant_id = $1", tenantID).Scan(&rowCount); err != nil {
		t.Fatalf("count contacts: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("one person should be one row, found %d", rowCount)
	}
}

func TestCreateLead_RejectsUnknownSource(t *testing.T) {
	svc := NewService(NewMockRepository())

	_, err := svc.CreateLead(context.Background(), CreateLeadInput{
		FirstName: "Any",
		LastName:  "One",
		Source:    "linkedin",
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidLeadSource) {
		t.Fatalf("expected ErrInvalidLeadSource, got %v", err)
	}
}

func TestUpdateLead_Validation(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	leadID := uuid.New()
	now := time.Now().UTC()
	status := LeadStatusNew

	repo.contacts[leadID] = &models.Contact{
		ID:             leadID,
		TenantID:       tenantID,
		FirstName:      "Petra",
		LastName:       "Lindner",
		CreatedAt:      now,
		UpdatedAt:      now,
		LifecycleStage: LifecycleLead,
		LeadStatus:     &status,
	}

	if _, err := svc.UpdateLead(context.Background(), leadID, UpdateLeadInput{}, tenantID); !errors.Is(err, ErrNoLeadChanges) {
		t.Fatalf("an empty patch should be rejected, got %v", err)
	}

	bogus := "lukewarm"
	if _, err := svc.UpdateLead(context.Background(), leadID, UpdateLeadInput{Temperature: &bogus}, tenantID); !errors.Is(err, ErrInvalidLeadTemperature) {
		t.Fatalf("expected ErrInvalidLeadTemperature, got %v", err)
	}

	// Qualifying from the inbox moves the lifecycle stage too -- it is the
	// same act as converting.
	qualified := LeadStatusQualified
	updated, err := svc.UpdateLead(context.Background(), leadID, UpdateLeadInput{Status: &qualified}, tenantID)
	if err != nil {
		t.Fatalf("UpdateLead: %v", err)
	}
	if updated.LifecycleStage != LifecycleQualified {
		t.Fatalf("qualifying did not move the stage: %q", updated.LifecycleStage)
	}

	// Disqualifying does not: the person stays on file, just unworked.
	disqualified := LeadStatusDisqualified
	repo.contacts[leadID].LifecycleStage = LifecycleLead
	updated, err = svc.UpdateLead(context.Background(), leadID, UpdateLeadInput{Status: &disqualified}, tenantID)
	if err != nil {
		t.Fatalf("UpdateLead (disqualify): %v", err)
	}
	if updated.LifecycleStage != LifecycleLead {
		t.Fatalf("disqualifying should leave the stage at lead, got %q", updated.LifecycleStage)
	}
}
