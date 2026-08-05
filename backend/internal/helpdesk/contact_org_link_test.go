package helpdesk

// g-helpdesk-contact-link: tickets can carry an optional contact_id/org_id
// pointing at the CRM's contacts/companies tables. Both are tenant-scoped
// checks at the service layer (ContactExists/CompanyExists), not just FK
// constraints -- a bare FK would happily accept another tenant's contact_id,
// since contacts.id is globally unique across tenants. These tests exercise
// the real repository against Postgres, not a mock, so a missing tenant_id
// predicate in ContactExists/CompanyExists would show up here.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestContactExistsAndCompanyExists_ScopedToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Contact Link Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Contact Link Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-contact-link-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Helpdesk",
		"last_name":  "ContactLink",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Helpdesk Contact Link Org",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	repo := NewPostgresRepository(pool)
	// RLS restricts visible rows to the caller's own tenant context regardless
	// of the explicit tenantID predicate — exercise both, mirroring
	// crm/consent's TestContactExistsAndAnonymize_ScopedToCallerTenant.
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	exists, err := repo.ContactExists(ctxOwn, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (own ctx, own tenant): %v", err)
	}
	if !exists {
		t.Fatal("ContactExists (own ctx, own tenant): expected true")
	}
	exists, err = repo.ContactExists(ctxOther, contactID, tenantOwn)
	if err != nil {
		t.Fatalf("ContactExists (foreign ctx): %v", err)
	}
	if exists {
		t.Fatal("ContactExists (foreign ctx): a foreign-tenant caller must not see another tenant's contact")
	}

	exists, err = repo.CompanyExists(ctxOwn, companyID, tenantOwn)
	if err != nil {
		t.Fatalf("CompanyExists (own ctx, own tenant): %v", err)
	}
	if !exists {
		t.Fatal("CompanyExists (own ctx, own tenant): expected true")
	}
	exists, err = repo.CompanyExists(ctxOther, companyID, tenantOwn)
	if err != nil {
		t.Fatalf("CompanyExists (foreign ctx): %v", err)
	}
	if exists {
		t.Fatal("CompanyExists (foreign ctx): a foreign-tenant caller must not see another tenant's company")
	}
}

func TestCreateTicket_RejectsForeignTenantContactAndOrg(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk CreateTicket Contact Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk CreateTicket Contact Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("hd-create-ticket-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Contact/company belong to tenantOther; CreateTicket is called for tenantOwn.
	foreignContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "Contact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContactID)

	foreignOrgID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOther,
		"name":       "Foreign Org",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", foreignOrgID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	_, err := svc.CreateTicket(ctxOwn, tenantOwn, uuid.New(), "Ticket with foreign contact", TicketPriorityNormal, nil, nil, "", "", &foreignContactID, nil, TicketIntake{})
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("CreateTicket with foreign contact_id: expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.CreateTicket(ctxOwn, tenantOwn, uuid.New(), "Ticket with foreign org", TicketPriorityNormal, nil, nil, "", "", nil, &foreignOrgID, TicketIntake{})
	if !errors.Is(err, ErrOrgNotFound) {
		t.Fatalf("CreateTicket with foreign org_id: expected ErrOrgNotFound, got %v", err)
	}
}

func TestCreateAndUpdateTicket_LinksOwnTenantContactAndOrgAndFilters(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Contact Link Own-Tenant Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-own-link-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Own",
		"last_name":  "Contact",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	orgID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Own Org",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "companies", orgID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	linked, err := svc.CreateTicket(ctxOwn, tenantOwn, uuid.New(), "Linked ticket", TicketPriorityNormal, nil, nil, "", "", &contactID, &orgID, TicketIntake{})
	if err != nil {
		t.Fatalf("CreateTicket with own-tenant contact/org: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", linked.ID)
	if linked.ContactID == nil || *linked.ContactID != contactID {
		t.Fatalf("expected ticket.ContactID = %s, got %v", contactID, linked.ContactID)
	}
	if linked.OrgID == nil || *linked.OrgID != orgID {
		t.Fatalf("expected ticket.OrgID = %s, got %v", orgID, linked.OrgID)
	}

	unlinked, err := svc.CreateTicket(ctxOwn, tenantOwn, uuid.New(), "Unlinked ticket", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("CreateTicket without contact/org: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", unlinked.ID)

	// ListTickets filters by contact_id/org_id against the real DB.
	byContact, total, err := svc.ListTickets(ctxOwn, tenantOwn, nil, nil, &contactID, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (contact_id filter): %v", err)
	}
	if total != 1 || len(byContact) != 1 || byContact[0].ID != linked.ID {
		t.Fatalf("ListTickets (contact_id filter): got %d tickets (total %d), want exactly the linked ticket", len(byContact), total)
	}

	byOrg, total, err := svc.ListTickets(ctxOwn, tenantOwn, nil, nil, nil, &orgID, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (org_id filter): %v", err)
	}
	if total != 1 || len(byOrg) != 1 || byOrg[0].ID != linked.ID {
		t.Fatalf("ListTickets (org_id filter): got %d tickets (total %d), want exactly the linked ticket", len(byOrg), total)
	}

	// UpdateTicket with a fresh own-tenant contact links a previously unlinked ticket.
	updated, err := svc.UpdateTicket(ctxOwn, unlinked.ID, tenantOwn, nil, nil, nil, nil, &contactID, nil)
	if err != nil {
		t.Fatalf("UpdateTicket with own-tenant contact: %v", err)
	}
	if updated.ContactID == nil || *updated.ContactID != contactID {
		t.Fatalf("expected updated ticket.ContactID = %s, got %v", contactID, updated.ContactID)
	}

	byContact, total, err = svc.ListTickets(ctxOwn, tenantOwn, nil, nil, &contactID, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListTickets (contact_id filter after update): %v", err)
	}
	if total != 2 || len(byContact) != 2 {
		t.Fatalf("ListTickets (contact_id filter after update): got %d tickets (total %d), want 2", len(byContact), total)
	}
}

func TestUpdateTicket_RejectsForeignTenantContact(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk UpdateTicket Contact Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk UpdateTicket Contact Other Tenant")

	ownUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("hd-update-ticket-own-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", ownUserID)

	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOther,
		"email":         fmt.Sprintf("hd-update-ticket-other-%s@tenantother.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	foreignContactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOther,
		"first_name": "Foreign",
		"last_name":  "UpdateContact",
		"created_by": otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", foreignContactID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	ticket, err := svc.CreateTicket(ctxOwn, tenantOwn, uuid.New(), "Update target", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	_, err = svc.UpdateTicket(ctxOwn, ticket.ID, tenantOwn, nil, nil, nil, nil, &foreignContactID, nil)
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("UpdateTicket with foreign contact_id: expected ErrContactNotFound, got %v", err)
	}

	got, err := repo.GetTicketByID(ctxOwn, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.ContactID != nil {
		t.Fatalf("a rejected UpdateTicket must not leave contact_id set, got %v", got.ContactID)
	}
}
