package helpdesk

// Proves the survey dispatch queries against a real database as kmuhub_app (no
// BYPASSRLS). The point of the first test is not that the due query runs
// without error but that it actually RETURNS a row for seeded data -- iteration
// 45's poller filtered on a trigger type that did not exist and looped into the
// void for months while every query "succeeded".

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedDueSurvey creates a tenant, a requester with an address, a closed ticket
// and a pending survey token whose send time lies sendOffset from now.
func seedDueSurvey(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string, now time.Time, sendOffset time.Duration) (*PostgresRepository, uuid.UUID) {
	t.Helper()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("hd-csat-%s@%s.local", uuid.New().String()[:8], label),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Cara",
		"last_name":     "Customer",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "Drucker <klemmt>",
		Status: TicketStatusClosed, Priority: TicketPriorityNormal,
		RequesterID: userID, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctx, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tickets", ticket.ID) })

	sendAfter := now.Add(sendOffset)
	issued, err := repo.IssueCsatSurveyTokenTx(
		ctx, tenantID, ticket.ID, "tok-"+uuid.New().String(),
		sendAfter, sendAfter.Add(CsatSurveyTokenTTL), now,
	)
	if err != nil || !issued {
		t.Fatalf("IssueCsatSurveyTokenTx: issued=%v err=%v", issued, err)
	}
	return repo, ticket.ID
}

func TestListDueCsatSurveys_FindsDueRowsAndSkipsTheRest(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: seedDueSurvey registers its row cleanups later, and
	// cleanups run last-in-first-out -- a deferred Close would shut the pool
	// before those rows could be deleted.
	t.Cleanup(pool.Close)

	tenantDue, tenantPending := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantDue, "Helpdesk CSAT Dispatch Due Tenant")
	testutil.EnsureTenant(t, pool, tenantPending, "Helpdesk CSAT Dispatch Pending Tenant")

	now := time.Now().UTC()
	sysCtx := testutil.WithSystemCtx(context.Background())

	// Due an hour ago, and one whose delay has not elapsed yet.
	repo, dueTicket := seedDueSurvey(t, pool, tenantDue, "due", now, -time.Hour)
	_, pendingTicket := seedDueSurvey(t, pool, tenantPending, "pending", now, +24*time.Hour)

	due, err := repo.ListDueCsatSurveys(sysCtx, now, csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys: %v", err)
	}

	found := mustSurveyFor(t, due, dueTicket)
	if found.TenantID != tenantDue {
		t.Fatalf("tenant on the due row is %s, expected %s", found.TenantID, tenantDue)
	}
	if found.RecipientEmail == "" {
		t.Fatal("due row carries no recipient address")
	}
	if found.RecipientName != "Cara Customer" {
		t.Fatalf("recipient name is %q, expected %q", found.RecipientName, "Cara Customer")
	}
	if found.TicketNumber == 0 {
		t.Fatal("due row carries no ticket number")
	}
	if surveyFor(due, pendingTicket) != nil {
		t.Fatal("a survey whose delay has not elapsed was returned as due")
	}

	// A tenant-scoped reader must not see the other tenant's CSAT row at all.
	var visible int
	if err := pool.QueryRow(testutil.WithTenantCtx(context.Background(), tenantPending),
		`SELECT count(*) FROM ticket_csat_responses WHERE ticket_id = $1`, dueTicket,
	).Scan(&visible); err != nil {
		t.Fatalf("cross-tenant count: %v", err)
	}
	if visible != 0 {
		t.Fatalf("foreign tenant sees %d csat row(s), expected 0", visible)
	}
}

func TestClaimCsatSurveyDispatch_OnlyOneClaimWinsAndReleaseRestores(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: seedDueSurvey registers its row cleanups later, and
	// cleanups run last-in-first-out -- a deferred Close would shut the pool
	// before those rows could be deleted.
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Claim Tenant")

	now := time.Now().UTC()
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo, ticketID := seedDueSurvey(t, pool, tenantID, "claim", now, -time.Hour)

	due, err := repo.ListDueCsatSurveys(sysCtx, now, csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys: %v", err)
	}
	row := mustSurveyFor(t, due, ticketID)

	claimed, err := repo.ClaimCsatSurveyDispatch(sysCtx, row.ResponseID, now)
	if err != nil || !claimed {
		t.Fatalf("first claim: claimed=%v err=%v", claimed, err)
	}
	// Second claim on the same row: this is the double-send guard.
	claimed, err = repo.ClaimCsatSurveyDispatch(sysCtx, row.ResponseID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if claimed {
		t.Fatal("a second claim succeeded -- the same survey would be mailed twice")
	}

	// A claimed row is no longer due, so a later tick does not even see it.
	after, err := repo.ListDueCsatSurveys(sysCtx, now.Add(time.Minute), csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys (after claim): %v", err)
	}
	if surveyFor(after, ticketID) != nil {
		t.Fatal("a claimed survey is still reported as due")
	}

	// Releasing the claim puts it back in the queue, with the attempt spent.
	if err := repo.ReleaseCsatSurveyDispatch(sysCtx, row.ResponseID, now); err != nil {
		t.Fatalf("ReleaseCsatSurveyDispatch: %v", err)
	}
	back, err := repo.ListDueCsatSurveys(sysCtx, now.Add(time.Minute), csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys (after release): %v", err)
	}
	if surveyFor(back, ticketID) == nil {
		t.Fatal("a released survey is not due again")
	}

	var attempts int16
	if err := pool.QueryRow(sysCtx,
		`SELECT survey_dispatch_attempts FROM ticket_csat_responses WHERE id = $1`, row.ResponseID,
	).Scan(&attempts); err != nil {
		t.Fatalf("read attempts: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d after one claim+release, expected 1", attempts)
	}

	// Exhausted attempts take the row out of the queue for good.
	if _, err := pool.Exec(sysCtx,
		`UPDATE ticket_csat_responses SET survey_dispatch_attempts = $2 WHERE id = $1`,
		row.ResponseID, csatSurveyMaxDispatchAttempts,
	); err != nil {
		t.Fatalf("exhaust attempts: %v", err)
	}
	exhausted, err := repo.ListDueCsatSurveys(sysCtx, now.Add(time.Minute), csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys (exhausted): %v", err)
	}
	if surveyFor(exhausted, ticketID) != nil {
		t.Fatal("a survey past its attempt budget is still retried")
	}
}

func TestCancelCsatSurvey_RevokesLinkAndKeepsRating(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: seedDueSurvey registers its row cleanups later, and
	// cleanups run last-in-first-out -- a deferred Close would shut the pool
	// before those rows could be deleted.
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Cancel Tenant")

	now := time.Now().UTC()
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo, ticketID := seedDueSurvey(t, pool, tenantID, "cancel", now, -time.Hour)

	due, err := repo.ListDueCsatSurveys(sysCtx, now, csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys: %v", err)
	}
	row := mustSurveyFor(t, due, ticketID)

	if err := repo.CancelCsatSurvey(sysCtx, row.ResponseID, now); err != nil {
		t.Fatalf("CancelCsatSurvey: %v", err)
	}
	if got := csatTokenOf(t, pool, sysCtx, ticketID); got != "" {
		t.Fatalf("token survived the cancellation: %q", got)
	}
	after, err := repo.ListDueCsatSurveys(sysCtx, now.Add(time.Minute), csatSurveyMaxDispatchAttempts, 100)
	if err != nil {
		t.Fatalf("ListDueCsatSurveys (after cancel): %v", err)
	}
	if surveyFor(after, ticketID) != nil {
		t.Fatal("a cancelled survey is still due")
	}

	// A rating that arrived through another path stays untouched.
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	if err := repo.SubmitCsatTx(ctx, tenantID, ticketID, 4, nil, now); err != nil {
		t.Fatalf("SubmitCsatTx: %v", err)
	}
	if err := repo.CancelCsatSurvey(sysCtx, row.ResponseID, now); err != nil {
		t.Fatalf("CancelCsatSurvey (after rating): %v", err)
	}
	var rating *int16
	if err := pool.QueryRow(sysCtx,
		`SELECT rating FROM ticket_csat_responses WHERE id = $1`, row.ResponseID,
	).Scan(&rating); err != nil {
		t.Fatalf("read rating: %v", err)
	}
	if rating == nil || *rating != 4 {
		t.Fatalf("cancelling wiped the rating: %v", rating)
	}
}

// mustSurveyFor is surveyFor with the "not due" case turned into a failure, so
// the seeded row not showing up is reported as what it is: a due query that
// matches nothing on real data.
func mustSurveyFor(t *testing.T, due []*DueCsatSurvey, ticketID uuid.UUID) *DueCsatSurvey {
	t.Helper()
	s := surveyFor(due, ticketID)
	if s == nil {
		t.Fatalf("survey for ticket %s is not in the due set", ticketID)
	}
	return s
}

func surveyFor(due []*DueCsatSurvey, ticketID uuid.UUID) *DueCsatSurvey {
	for _, s := range due {
		if s.TicketID == ticketID {
			return s
		}
	}
	return nil
}
