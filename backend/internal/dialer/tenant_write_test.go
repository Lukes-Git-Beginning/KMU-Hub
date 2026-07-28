package dialer

// Iteration 52 gave CampaignRepository.Update/UpdateStatus/Delete, the
// campaign-contact queue writes (UpdateContactStatus, SetContactCallback,
// SkipContact, RequeueContact, IncrementContactCallCount,
// GetCampaignContactByID) and OutcomeRepository.GetByID/Update/Delete an
// explicit tenant_id predicate -- before that they trusted RLS alone
// (mig 000119/000238). This file proves the fix against real repository
// calls, not just a raw-SQL row count (rls_test.go already covers that):
// each write is exercised once from a foreign-tenant ctx, passing the row's
// *real* tenantID as the explicit parameter -- so only RLS, not the WHERE
// clause, could be stopping it -- before repeating the same call from the
// owning ctx and confirming it lands.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCampaignWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Campaign Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer Campaign Write Other Tenant")

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantOwn,
		"email":         "dialer-campwrite@test.local",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	camp := &Campaign{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Write Test Campaign",
		Status: CampaignStatusDraft, Mode: CampaignModePreview,
		Settings: CampaignSettings{}, CreatedBy: userID,
		AssignedAgentIDs: []uuid.UUID{},
		CreatedAt:        now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, camp); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", camp.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "dialer_campaigns", camp.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "dialer_campaigns", camp.ID, 0)

	// Update
	foreign := *camp
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign, tenantOwn); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByIDForTenant(sysCtx, camp.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByIDForTenant: %v", err)
	}
	if got.Name == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the campaign")
	}
	foreign.Name = "Renamed"
	if err := repo.Update(ctxOwn, &foreign, tenantOwn); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, _ = repo.GetByIDForTenant(ctxOwn, camp.ID, tenantOwn)
	if got.Name != "Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// UpdateStatus
	if err := repo.UpdateStatus(ctxOther, camp.ID, tenantOwn, CampaignStatusActive); err == nil {
		t.Fatalf("UpdateStatus (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetByIDForTenant(sysCtx, camp.ID, tenantOwn)
	if got.Status == CampaignStatusActive {
		t.Fatalf("a foreign-tenant UpdateStatus reached the campaign")
	}
	if err := repo.UpdateStatus(ctxOwn, camp.ID, tenantOwn, CampaignStatusActive); err != nil {
		t.Fatalf("UpdateStatus (own ctx): %v", err)
	}
	got, _ = repo.GetByIDForTenant(ctxOwn, camp.ID, tenantOwn)
	if got.Status != CampaignStatusActive {
		t.Fatalf("own-tenant UpdateStatus did not land: status=%q", got.Status)
	}

	// Delete
	if err := repo.Delete(ctxOther, camp.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_campaigns", camp.ID, 1)
	if err := repo.Delete(ctxOwn, camp.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_campaigns", camp.ID, 0)
}

func TestCampaignContactWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Contact Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer Contact Write Other Tenant")

	repo := NewPostgresCampaignRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantOwn,
		"email":         "dialer-ccwrite@test.local",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	campID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOwn,
		"name":       "Contact Write Campaign",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOwn,
		"first_name": "Write",
		"last_name":  "Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ccID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   tenantOwn,
		"campaign_id": campID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", ccID)

	// UpdateContactStatus
	if err := repo.UpdateContactStatus(ctxOther, ccID, tenantOwn, ContactStatusCompleted, nil); err == nil {
		t.Fatalf("UpdateContactStatus (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetCampaignContactByID(sysCtx, ccID, tenantOwn)
	if err != nil {
		t.Fatalf("GetCampaignContactByID: %v", err)
	}
	if got.Status == ContactStatusCompleted {
		t.Fatalf("a foreign-tenant UpdateContactStatus reached the contact")
	}
	if err := repo.UpdateContactStatus(ctxOwn, ccID, tenantOwn, ContactStatusCompleted, nil); err != nil {
		t.Fatalf("UpdateContactStatus (own ctx): %v", err)
	}
	got, _ = repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if got.Status != ContactStatusCompleted {
		t.Fatalf("own-tenant UpdateContactStatus did not land: status=%q", got.Status)
	}

	// SetContactCallback
	callbackAt := now.Add(24 * time.Hour)
	if err := repo.SetContactCallback(ctxOther, ccID, tenantOwn, callbackAt); err == nil {
		t.Fatalf("SetContactCallback (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetCampaignContactByID(sysCtx, ccID, tenantOwn)
	if got.Status == ContactStatusCallback {
		t.Fatalf("a foreign-tenant SetContactCallback reached the contact")
	}
	if err := repo.SetContactCallback(ctxOwn, ccID, tenantOwn, callbackAt); err != nil {
		t.Fatalf("SetContactCallback (own ctx): %v", err)
	}
	got, _ = repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if got.Status != ContactStatusCallback {
		t.Fatalf("own-tenant SetContactCallback did not land: status=%q", got.Status)
	}

	// SkipContact
	if err := repo.SkipContact(ctxOther, ccID, tenantOwn); err == nil {
		t.Fatalf("SkipContact (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetCampaignContactByID(sysCtx, ccID, tenantOwn)
	if got.Status == ContactStatusSkipped {
		t.Fatalf("a foreign-tenant SkipContact reached the contact")
	}
	if err := repo.SkipContact(ctxOwn, ccID, tenantOwn); err != nil {
		t.Fatalf("SkipContact (own ctx): %v", err)
	}
	got, _ = repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if got.Status != ContactStatusSkipped {
		t.Fatalf("own-tenant SkipContact did not land: status=%q", got.Status)
	}

	// RequeueContact
	if err := repo.RequeueContact(ctxOther, ccID, tenantOwn); err == nil {
		t.Fatalf("RequeueContact (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetCampaignContactByID(sysCtx, ccID, tenantOwn)
	if got.Status == ContactStatusPending {
		t.Fatalf("a foreign-tenant RequeueContact reached the contact")
	}
	if err := repo.RequeueContact(ctxOwn, ccID, tenantOwn); err != nil {
		t.Fatalf("RequeueContact (own ctx): %v", err)
	}
	got, _ = repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if got.Status != ContactStatusPending {
		t.Fatalf("own-tenant RequeueContact did not land: status=%q", got.Status)
	}

	// IncrementContactCallCount
	if err := repo.IncrementContactCallCount(ctxOther, ccID, tenantOwn); err == nil {
		t.Fatalf("IncrementContactCallCount (foreign ctx): expected an error, got nil")
	}
	got, _ = repo.GetCampaignContactByID(sysCtx, ccID, tenantOwn)
	if got.CallCount != 0 {
		t.Fatalf("a foreign-tenant IncrementContactCallCount reached the contact: call_count=%d", got.CallCount)
	}
	if err := repo.IncrementContactCallCount(ctxOwn, ccID, tenantOwn); err != nil {
		t.Fatalf("IncrementContactCallCount (own ctx): %v", err)
	}
	got, _ = repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn)
	if got.CallCount != 1 {
		t.Fatalf("own-tenant IncrementContactCallCount did not land: call_count=%d", got.CallCount)
	}

	// GetCampaignContactByID itself: the real tenantOwn passed explicitly from
	// a foreign session ctx must still be blocked by RLS.
	if _, err := repo.GetCampaignContactByID(ctxOther, ccID, tenantOwn); err == nil {
		t.Fatalf("GetCampaignContactByID (foreign ctx): expected an error, got nil")
	}
	if _, err := repo.GetCampaignContactByID(ctxOwn, ccID, tenantOwn); err != nil {
		t.Fatalf("GetCampaignContactByID (own ctx): %v", err)
	}
}

func TestOutcomeWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Dialer Outcome Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Dialer Outcome Write Other Tenant")

	repo := NewPostgresOutcomeRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	o := &CallOutcome{
		ID: uuid.New(), TenantID: tenantOwn, Label: "Write Test Outcome",
		Color: "#00ff00", IsActive: true, CreatedAt: now,
	}
	if err := repo.Create(ctxOwn, o); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "dialer_call_outcomes", o.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "dialer_call_outcomes", o.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "dialer_call_outcomes", o.ID, 0)

	// GetByID: real tenantOwn passed explicitly from a foreign session ctx
	// must still be blocked by RLS.
	if _, err := repo.GetByID(ctxOther, o.ID, tenantOwn); err == nil {
		t.Fatalf("GetByID (foreign ctx): expected an error, got nil")
	}
	loaded, err := repo.GetByID(ctxOwn, o.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}

	// Update trusts loaded.TenantID (== tenantOwn) as its predicate.
	foreign := *loaded
	foreign.Label = "Hacked"
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(sysCtx, o.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (sys ctx): %v", err)
	}
	if got.Label == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the outcome")
	}
	foreign.Label = "Renamed"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, o.ID, tenantOwn)
	if got.Label != "Renamed" {
		t.Fatalf("own-tenant write did not land: label=%q", got.Label)
	}

	// Delete
	if err := repo.Delete(ctxOther, o.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_call_outcomes", o.ID, 1)
	if err := repo.Delete(ctxOwn, o.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "dialer_call_outcomes", o.ID, 0)
}
