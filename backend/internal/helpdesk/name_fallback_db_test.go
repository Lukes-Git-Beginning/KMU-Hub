package helpdesk

// name_fallback_db_test.go is the regression test for the CONCAT_WS blank
// name bug: users.first_name/last_name are NOT NULL DEFAULT '', never NULL,
// so CONCAT_WS(' ', '', '') returns a single space rather than NULL, and
// NULLIF(' ', '') does not recognize that as blank. Two independent copies of
// the expression carried the bug: ticketSelectColumns' assignee_name and
// requester_name joins (used by GetTicketByID/ListTickets/...), and
// ListDueCsatSurveys' recipient name join. Each fails against the pre-fix
// query (COALESCE(NULLIF(CONCAT_WS(...), ''), ...)) and passes only with a
// TRIM() around the CONCAT_WS before the NULLIF.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedBlankNamedUser inserts a users row with first_name/last_name explicitly
// blank -- what every real user with no name set looks like (NOT NULL
// DEFAULT ''), not a synthetic edge case.
func seedBlankNamedUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) (uuid.UUID, string) {
	t.Helper()
	email := fmt.Sprintf("hd-blank-%s-%s@example.test", label, uuid.New().String()[:8])
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": email,
		"password_hash": "$argon2id$v=19$test", "first_name": "", "last_name": "",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id, email
}

func TestGetTicketByID_BlankNamedAssigneeAndRequesterFallBackToEmail(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Name Fallback Tenant")

	assigneeID, assigneeEmail := seedBlankNamedUser(t, pool, tenantID, "assignee")
	requesterID, requesterEmail := seedBlankNamedUser(t, pool, tenantID, "requester")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Blank Name Ticket",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		AssigneeID: &assigneeID, RequesterID: &requesterID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", ticket.ID) })

	got, err := repo.GetTicketByID(ctx, ticket.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.AssigneeName == nil || *got.AssigneeName != assigneeEmail {
		t.Errorf("AssigneeName: got %v, want the email fallback %q", got.AssigneeName, assigneeEmail)
	}
	if got.RequesterName != requesterEmail {
		t.Errorf("RequesterName: got %q, want the email fallback %q", got.RequesterName, requesterEmail)
	}
}

func TestListDueCsatSurveys_BlankNamedRequesterFallsBackToEmail(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Blank Name Tenant")

	requesterID, requesterEmail := seedBlankNamedUser(t, pool, tenantID, "csat-requester")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Blank Name CSAT Ticket",
		Status: TicketStatusClosed, Priority: TicketPriorityNormal,
		RequesterID: &requesterID, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", ticket.ID) })

	sendAfter := now.Add(-time.Hour)
	issued, err := repo.IssueCsatSurveyTokenTx(
		ctx, tenantID, ticket.ID, "tok-"+uuid.New().String(),
		sendAfter, sendAfter.Add(CsatSurveyTokenTTL), now,
	)
	if err != nil || !issued {
		t.Fatalf("IssueCsatSurveyTokenTx: issued=%v err=%v", issued, err)
	}

	due, err := repo.ListDueCsatSurveys(testutil.WithSystemCtx(context.Background()), now, csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys: %v", err)
	}
	found := mustSurveyFor(t, due, ticket.ID)
	if found.RecipientName != requesterEmail {
		t.Errorf("RecipientName: got %q, want the email fallback %q", found.RecipientName, requesterEmail)
	}
}
