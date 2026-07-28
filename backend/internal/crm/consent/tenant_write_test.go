package consent

// wp-crm-meta: closes the write-surface gap for the consent package. The
// existing rls_test.go seeds exclusively through testutil.SeedRow and never
// calls the real repository methods.
//
// Writing this test surfaced a dead write: CreateDeletionRequest never set
// gdpr_deletion_requests.tenant_id, a column that has been NOT NULL since
// migration 000114 with no default — every RequestDeletion call failed with
// a NOT NULL violation against real Postgres, "GDPR-Loeschantrag stellen"
// was completely broken. GetDeletionRequest/UpdateDeletionRequest/
// AnonymizeContact/ContactExists were also missing an explicit tenant_id
// predicate: ContactExists in particular is the existence guard GrantConsent/
// RevokeConsent/RequestDeletion all call before writing, so a caller from
// tenant B passing tenant A's real contact_id would have passed the check
// and created a consent_records/gdpr_deletion_requests row whose tenant_id
// (B, correct) pointed at a contact_id that belongs to a different tenant
// (A) — a cross-tenant data-integrity break, not just an RLS gap. All fixed
// in the same commit: TenantID added to GDPRDeletionRequest, every affected
// repository method takes/uses tenantID explicitly, and RequestDeletion/
// ProcessDeletion thread it through from the gRPC handler
// (middleware.GetTenantID).

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestConsentRecordWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Consent Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Consent Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("consent-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Consent",
		"last_name":  "WriteTest",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	record := &ConsentRecord{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		ContactID:   contactID,
		ConsentType: "marketing_email",
		Granted:     true,
		LegalBasis:  "consent",
		Source:      "web_form",
		GrantedAt:   &now,
		CreatedBy:   &userID,
		CreatedAt:   now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.CreateConsentRecord(ctxOther, record); err == nil {
		t.Fatalf("CreateConsentRecord (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "consent_records", record.ID, 0)

	if err := repo.CreateConsentRecord(ctxOwn, record); err != nil {
		t.Fatalf("CreateConsentRecord (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "consent_records", record.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "consent_records", record.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "consent_records", record.ID, 0)
}

func TestGDPRDeletionRequestWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "GDPR Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "GDPR Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("gdpr-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "GDPR",
		"last_name":  "WriteTest",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	req := &GDPRDeletionRequest{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		ContactID: contactID,
		Reason:    "write-surface test",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.CreateDeletionRequest(ctxOther, req); err == nil {
		t.Fatalf("CreateDeletionRequest (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "gdpr_deletion_requests", req.ID, 0)

	// This is the dead write: before the fix this failed with a NOT NULL
	// violation on tenant_id regardless of ctx.
	if err := repo.CreateDeletionRequest(ctxOwn, req); err != nil {
		t.Fatalf("CreateDeletionRequest (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_deletion_requests", req.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "gdpr_deletion_requests", req.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "gdpr_deletion_requests", req.ID, 0)

	// GetDeletionRequest carries an explicit tenantID predicate — a foreign
	// session passing the row's real tenantID must still see nothing (RLS).
	if _, err := repo.GetDeletionRequest(ctxOther, req.ID, tenantOwn); !errors.Is(err, ErrDeletionRequestNotFound) {
		t.Fatalf("GetDeletionRequest (foreign ctx): expected ErrDeletionRequestNotFound, got %v", err)
	}
	got, err := repo.GetDeletionRequest(ctxOwn, req.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetDeletionRequest (own ctx): %v", err)
	}
	if got.Status != "pending" {
		t.Fatalf("unexpected status: %q", got.Status)
	}

	// UpdateDeletionRequest carries the same explicit predicate (req.TenantID).
	foreign := *req
	foreign.Status = "completed"
	if err := repo.UpdateDeletionRequest(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateDeletionRequest (foreign ctx): unexpected error %v", err)
	}
	got, err = repo.GetDeletionRequest(ctxOwn, req.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetDeletionRequest after foreign update: %v", err)
	}
	if got.Status != "pending" {
		t.Fatalf("a foreign-tenant write reached the deletion request: status=%q", got.Status)
	}

	if err := repo.UpdateDeletionRequest(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateDeletionRequest (own ctx): %v", err)
	}
	got, err = repo.GetDeletionRequest(ctxOwn, req.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetDeletionRequest after own update: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("own-tenant write did not land: status=%q", got.Status)
	}
}

func TestContactExistsAndAnonymize_ScopedToCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Consent Anon Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Consent Anon Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("consent-anon-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Anon",
		"last_name":  "WriteTest",
		"email":      fmt.Sprintf("anon-%s@tenantown.local", uuid.New().String()[:8]),
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// A caller from a foreign tenant passing the victim's real contact_id and
	// tenant_id must not see the contact as existing — otherwise a foreign
	// tenant could pass the RequestDeletion/GrantConsent existence guard
	// using a contact_id that belongs to someone else's tenant.
	existsForeign, err := repo.ContactExists(ctxOther, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (foreign ctx): %v", err)
	}
	if existsForeign {
		t.Fatalf("ContactExists (foreign ctx): contact from another tenant reported as existing")
	}

	existsOwn, err := repo.ContactExists(ctxOwn, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (own ctx): %v", err)
	}
	if !existsOwn {
		t.Fatalf("ContactExists (own ctx): expected the contact to exist")
	}

	// AnonymizeContact carries the same explicit tenantID predicate.
	if err := repo.AnonymizeContact(ctxOther, contactID, tenantOwn); err != nil {
		t.Fatalf("AnonymizeContact (foreign ctx): unexpected error %v", err)
	}
	var firstName string
	if err := pool.QueryRow(ctxOwn, "SELECT first_name FROM contacts WHERE id = $1", contactID).Scan(&firstName); err != nil {
		t.Fatalf("read back contact: %v", err)
	}
	if firstName != "Anon" {
		t.Fatalf("a foreign-tenant anonymize reached the contact: first_name=%q", firstName)
	}

	if err := repo.AnonymizeContact(ctxOwn, contactID, tenantOwn); err != nil {
		t.Fatalf("AnonymizeContact (own ctx): %v", err)
	}
	if err := pool.QueryRow(ctxOwn, "SELECT first_name FROM contacts WHERE id = $1", contactID).Scan(&firstName); err != nil {
		t.Fatalf("read back contact after anonymize: %v", err)
	}
	if firstName != "Gelöschte" {
		t.Fatalf("own-tenant anonymize did not land: first_name=%q", firstName)
	}
}
