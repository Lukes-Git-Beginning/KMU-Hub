package helpdesk

// Proves IssueCsatSurveyTokenTx against a real database as kmuhub_app (no
// BYPASSRLS): a foreign-tenant caller writes nothing even when it passes the
// row's real tenantID, an already-rated ticket keeps its rating and gets no
// new link, and the CSAT row stays invisible across tenants.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestIssueCsatSurveyTokenTx_TenantScopedAndRatingAware(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk CSAT Survey Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk CSAT Survey Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()
	sendAfter := now.Add(24 * time.Hour)
	expiresAt := sendAfter.Add(CsatSurveyTokenTTL)

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "CSAT Survey Ticket",
		Status: TicketStatusClosed, Priority: TicketPriorityNormal,
		RequesterID: uuidPtr(uuid.New()), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	// The CSAT row hangs off the ticket with ON DELETE CASCADE.
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	// A foreign tenant cannot mint a token even while passing the ticket's real
	// tenant id -- so only RLS, not the WHERE clause, can be what stops it.
	issued, err := repo.IssueCsatSurveyTokenTx(ctxOther, tenantOwn, ticket.ID, "foreign-token", sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (foreign ctx): %v", err)
	}
	if issued {
		t.Fatal("a foreign-tenant caller minted a survey token")
	}

	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, "token-one", sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (own ctx): %v", err)
	}
	if !issued {
		t.Fatal("expected the owning tenant to mint a survey token")
	}
	if got := csatTokenOf(t, pool, sysCtx, ticket.ID); got != "token-one" {
		t.Fatalf("stored token is %q, expected token-one", got)
	}

	// Cross-tenant read of the CSAT row itself: RLS must hide it entirely.
	var visible int
	if err := pool.QueryRow(ctxOther,
		`SELECT count(*) FROM ticket_csat_responses WHERE ticket_id = $1`, ticket.ID,
	).Scan(&visible); err != nil {
		t.Fatalf("count csat rows (foreign ctx): %v", err)
	}
	if visible != 0 {
		t.Fatalf("foreign tenant sees %d csat row(s), expected 0", visible)
	}

	// Closing again replaces a pending link rather than piling up rows.
	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, "token-two", sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (second close): %v", err)
	}
	if !issued {
		t.Fatal("expected a pending token to be replaced")
	}
	if got := csatTokenOf(t, pool, sysCtx, ticket.ID); got != "token-two" {
		t.Fatalf("stored token is %q, expected token-two", got)
	}

	// Once a rating is in, no further survey goes out for this ticket.
	if err := repo.SubmitCsatTx(ctxOwn, tenantOwn, ticket.ID, 5, nil, now); err != nil {
		t.Fatalf("SubmitCsatTx: %v", err)
	}
	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, "token-three", sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (after rating): %v", err)
	}
	if issued {
		t.Fatal("an already rated ticket was handed a new survey token")
	}
	if got := csatTokenOf(t, pool, sysCtx, ticket.ID); got != "token-two" {
		t.Fatalf("the rating overwrote the token: %q", got)
	}
}

// csatTokenOf reads the token column of a ticket's CSAT row under ctx.
func csatTokenOf(t *testing.T, pool *pgxpool.Pool, ctx context.Context, ticketID uuid.UUID) string {
	t.Helper()
	var token *string
	if err := pool.QueryRow(ctx,
		`SELECT token FROM ticket_csat_responses WHERE ticket_id = $1`, ticketID,
	).Scan(&token); err != nil {
		t.Fatalf("read csat token: %v", err)
	}
	if token == nil {
		return ""
	}
	return *token
}
