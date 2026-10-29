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

	"github.com/kmuhub/kmuhub/internal/linksecret"
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

	// A foreign tenant cannot schedule a survey even while passing the ticket's
	// real tenant id -- so only RLS, not the WHERE clause, can be what stops it.
	issued, err := repo.IssueCsatSurveyTokenTx(ctxOther, tenantOwn, ticket.ID, sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (foreign ctx): %v", err)
	}
	if issued {
		t.Fatal("a foreign-tenant caller scheduled a survey")
	}

	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (own ctx): %v", err)
	}
	if !issued {
		t.Fatal("expected the owning tenant to schedule a survey")
	}
	// Decision 2026-10-26: closing stores no link secret; the dispatch claim
	// adds the hash of the link it mails.
	if got := csatTokenHashOf(t, pool, sysCtx, ticket.ID); got != "" {
		t.Fatalf("scheduling stored a link hash %q", got)
	}
	claimCsatSurveyLink(t, pool, repo, ticket.ID, "token-one", now)

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

	// Closing again replaces the pending survey rather than piling up rows,
	// and the link mailed after the first close dies with it.
	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (second close): %v", err)
	}
	if !issued {
		t.Fatal("expected a pending survey to be replaced")
	}
	if got := csatTokenHashOf(t, pool, sysCtx, ticket.ID); got != "" {
		t.Fatalf("the link of the first close survived the second close: %q", got)
	}
	claimCsatSurveyLink(t, pool, repo, ticket.ID, "token-two", now)

	// Once a rating is in, no further survey goes out for this ticket.
	if err := repo.SubmitCsatTx(ctxOwn, tenantOwn, ticket.ID, 5, nil, now); err != nil {
		t.Fatalf("SubmitCsatTx: %v", err)
	}
	issued, err = repo.IssueCsatSurveyTokenTx(ctxOwn, tenantOwn, ticket.ID, sendAfter, expiresAt, now)
	if err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (after rating): %v", err)
	}
	if issued {
		t.Fatal("an already rated ticket was handed a new survey")
	}
	if got := csatTokenHashOf(t, pool, sysCtx, ticket.ID); got != linksecret.Hash("token-two") {
		t.Fatalf("the rating overwrote the link hash: %q", got)
	}
}

// csatTokenHashOf reads the token_hash column of a ticket's CSAT row under ctx.
func csatTokenHashOf(t *testing.T, pool *pgxpool.Pool, ctx context.Context, ticketID uuid.UUID) string {
	t.Helper()
	var hash *string
	if err := pool.QueryRow(ctx,
		`SELECT token_hash FROM ticket_csat_responses WHERE ticket_id = $1`, ticketID,
	).Scan(&hash); err != nil {
		t.Fatalf("read csat token hash: %v", err)
	}
	if hash == nil {
		return ""
	}
	return *hash
}

// claimCsatSurveyLink does what the dispatcher does at send time: it claims
// the ticket's pending survey and stores the hash of token, which from then on
// is the redeemable link. Tests that redeem pick their own token this way.
func claimCsatSurveyLink(t *testing.T, pool *pgxpool.Pool, repo *PostgresRepository, ticketID uuid.UUID, token string, now time.Time) {
	t.Helper()
	sysCtx := testutil.WithSystemCtx(context.Background())
	var responseID uuid.UUID
	if err := pool.QueryRow(sysCtx,
		`SELECT id FROM ticket_csat_responses WHERE ticket_id = $1`, ticketID,
	).Scan(&responseID); err != nil {
		t.Fatalf("read csat response id: %v", err)
	}
	claimed, err := repo.ClaimCsatSurveyDispatch(sysCtx, responseID, now, linksecret.Hash(token))
	if err != nil || !claimed {
		t.Fatalf("ClaimCsatSurveyDispatch: claimed=%v err=%v", claimed, err)
	}
}
