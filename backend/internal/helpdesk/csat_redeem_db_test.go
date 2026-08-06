package helpdesk

// Proves the public survey redemption against a real database as kmuhub_app
// (no BYPASSRLS). The two properties that matter here are the ones a mock
// cannot show: the token lookup really does resolve a row without any tenant
// on the context (RLS would otherwise hide the one row naming the tenant), and
// the write that follows really is tenant-scoped again.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSubmitCsatByToken_RedeemsOnceAndScopesToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk CSAT Redeem Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk CSAT Redeem Other Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "CSAT Redeem Ticket",
		Status: TicketStatusClosed, Priority: TicketPriorityNormal,
		RequesterID: uuidPtr(uuid.New()), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	token := "redeem-" + uuid.NewString()
	issued, err := repo.IssueCsatSurveyTokenTx(
		ctxOwn, tenantOwn, ticket.ID, token, now, now.Add(CsatSurveyTokenTTL), now,
	)
	if err != nil || !issued {
		t.Fatalf("IssueCsatSurveyTokenTx: issued=%v err=%v", issued, err)
	}

	// The redemption itself carries NO tenant: this is the unauthenticated
	// customer's context, exactly as the gateway hands it over. If the lookup
	// were tenant-scoped, this call would 404 for a perfectly good link.
	comment := "Alles bestens"
	res, err := svc.SubmitCsatByToken(context.Background(), token, 4, &comment)
	if err != nil {
		t.Fatalf("SubmitCsatByToken: %v", err)
	}
	if res.TicketNumber != ticket.TicketNumber || res.Rating != 4 {
		t.Fatalf("redemption returned ticket #%d rating %d, want #%d rating 4",
			res.TicketNumber, res.Rating, ticket.TicketNumber)
	}

	// The rating landed on both the response row and the ticket mirror, and the
	// token is gone -- the link is spent.
	if got := csatTokenOf(t, pool, sysCtx, ticket.ID); got != "" {
		t.Fatalf("token survived redemption: %q", got)
	}
	var (
		rowRating   *int16
		rowComment  *string
		submittedAt *time.Time
	)
	if err := pool.QueryRow(ctxOwn,
		`SELECT rating, comment, submitted_at FROM ticket_csat_responses WHERE ticket_id = $1`,
		ticket.ID,
	).Scan(&rowRating, &rowComment, &submittedAt); err != nil {
		t.Fatalf("read csat response: %v", err)
	}
	if rowRating == nil || *rowRating != 4 || submittedAt == nil {
		t.Fatalf("response row not rated: rating=%v submitted_at=%v", rowRating, submittedAt)
	}
	if rowComment == nil || *rowComment != comment {
		t.Fatalf("comment not stored: %v", rowComment)
	}
	stored, err := repo.GetTicketByID(ctxOwn, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if stored.CsatRating == nil || *stored.CsatRating != 4 {
		t.Fatalf("ticket mirror not written: %v", stored.CsatRating)
	}

	// Single use: the second attempt is indistinguishable from an invented
	// token, and it must not overwrite the rating that is already in.
	if _, err := svc.SubmitCsatByToken(context.Background(), token, 1, nil); !errors.Is(err, ErrCsatSurveyNotFound) {
		t.Fatalf("second redemption: got %v, want ErrCsatSurveyNotFound", err)
	}

	// A foreign tenant still cannot see the rated row at all.
	var visible int
	if err := pool.QueryRow(ctxOther,
		`SELECT count(*) FROM ticket_csat_responses WHERE ticket_id = $1`, ticket.ID,
	).Scan(&visible); err != nil {
		t.Fatalf("count csat rows (foreign ctx): %v", err)
	}
	if visible != 0 {
		t.Fatalf("foreign tenant sees %d csat row(s), expected 0", visible)
	}
}

// TestSubmitCsatByToken_TwoTenantsRedemptionStaysIsolated seeds a ticket and a
// pending survey token in EACH of two tenants at the same time, then redeems
// tenant A's token. The other coverage in this file proves a foreign SELECT
// can't see the row after the fact; this one proves the redemption itself --
// the token lookup plus the write that follows -- never touches tenant B's
// ticket, even though both rows exist side by side while it runs.
func TestSubmitCsatByToken_TwoTenantsRedemptionStaysIsolated(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Helpdesk CSAT Cross Redeem Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Helpdesk CSAT Cross Redeem Tenant B")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	mkTicket := func(ctx context.Context, tenantID uuid.UUID, subject string) *Ticket {
		t.Helper()
		tk := &Ticket{
			ID: uuid.New(), TenantID: tenantID, Subject: subject,
			Status: TicketStatusClosed, Priority: TicketPriorityNormal,
			RequesterID: uuidPtr(uuid.New()), CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateTicket(ctx, tk); err != nil {
			t.Fatalf("CreateTicket(%s): %v", subject, err)
		}
		return tk
	}

	ticketA := mkTicket(ctxA, tenantA, "CSAT Cross Redeem Ticket A")
	defer testutil.CleanupRow(t, pool, "tickets", ticketA.ID)
	ticketB := mkTicket(ctxB, tenantB, "CSAT Cross Redeem Ticket B")
	defer testutil.CleanupRow(t, pool, "tickets", ticketB.ID)

	run := uuid.NewString()
	tokenA, tokenB := "cross-a-"+run, "cross-b-"+run
	if issued, err := repo.IssueCsatSurveyTokenTx(
		ctxA, tenantA, ticketA.ID, tokenA, now, now.Add(CsatSurveyTokenTTL), now,
	); err != nil || !issued {
		t.Fatalf("IssueCsatSurveyTokenTx (A): issued=%v err=%v", issued, err)
	}
	if issued, err := repo.IssueCsatSurveyTokenTx(
		ctxB, tenantB, ticketB.ID, tokenB, now, now.Add(CsatSurveyTokenTTL), now,
	); err != nil || !issued {
		t.Fatalf("IssueCsatSurveyTokenTx (B): issued=%v err=%v", issued, err)
	}

	// Redeem A's token. If the token lookup or the write that follows ever
	// slipped tenant scope, this is where B's ticket would get hit instead of
	// (or in addition to) A's.
	if _, err := svc.SubmitCsatByToken(context.Background(), tokenA, 5, nil); err != nil {
		t.Fatalf("SubmitCsatByToken (A): %v", err)
	}

	storedA, err := repo.GetTicketByID(ctxA, ticketA.ID, tenantA)
	if err != nil {
		t.Fatalf("GetTicketByID (A): %v", err)
	}
	if storedA.CsatRating == nil || *storedA.CsatRating != 5 {
		t.Fatalf("ticket A did not receive the rating: %v", storedA.CsatRating)
	}

	// Ticket B must be untouched: no rating, and its own token is still live
	// and independently redeemable.
	storedB, err := repo.GetTicketByID(ctxB, ticketB.ID, tenantB)
	if err != nil {
		t.Fatalf("GetTicketByID (B): %v", err)
	}
	if storedB.CsatRating != nil {
		t.Fatalf("ticket B was rated by tenant A's redemption: %v", *storedB.CsatRating)
	}
	if got := csatTokenOf(t, pool, ctxB, ticketB.ID); got != tokenB {
		t.Fatalf("ticket B's token is %q, expected untouched %q", got, tokenB)
	}

	if _, err := svc.SubmitCsatByToken(context.Background(), tokenB, 2, nil); err != nil {
		t.Fatalf("SubmitCsatByToken (B): %v", err)
	}
	storedB, err = repo.GetTicketByID(ctxB, ticketB.ID, tenantB)
	if err != nil {
		t.Fatalf("GetTicketByID (B, after own redemption): %v", err)
	}
	if storedB.CsatRating == nil || *storedB.CsatRating != 2 {
		t.Fatalf("ticket B's own redemption did not land: %v", storedB.CsatRating)
	}
}

func TestSubmitCsatByToken_RefusesDeadLinks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Dead Link Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()

	// Cleanup via defer, not t.Cleanup: t.Cleanup runs after the deferred
	// pool.Close() above and would find a closed pool.
	mkTicket := func(subject string) *Ticket {
		t.Helper()
		tk := &Ticket{
			ID: uuid.New(), TenantID: tenantID, Subject: subject,
			Status: TicketStatusClosed, Priority: TicketPriorityNormal,
			RequesterID: uuidPtr(uuid.New()), CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateTicket(ctx, tk); err != nil {
			t.Fatalf("CreateTicket(%s): %v", subject, err)
		}
		return tk
	}

	// Tokens are globally unique, so they carry a run id -- a fixed literal
	// would collide with the leftovers of any earlier failed run.
	run := uuid.NewString()

	// Expired link.
	expiredTicket := mkTicket("CSAT Expired Link")
	defer testutil.CleanupRow(t, pool, "tickets", expiredTicket.ID)
	expiredToken := "expired-" + run
	if _, err := repo.IssueCsatSurveyTokenTx(
		ctx, tenantID, expiredTicket.ID, expiredToken,
		now.Add(-48*time.Hour), now.Add(-time.Hour), now,
	); err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (expired): %v", err)
	}

	// Link on a ticket that was rated through the authenticated route in the
	// meantime: the token is still out there, the row is no longer pending.
	ratedTicket := mkTicket("CSAT Already Rated")
	defer testutil.CleanupRow(t, pool, "tickets", ratedTicket.ID)
	ratedToken := "rated-" + run
	if _, err := repo.IssueCsatSurveyTokenTx(
		ctx, tenantID, ratedTicket.ID, ratedToken, now, now.Add(CsatSurveyTokenTTL), now,
	); err != nil {
		t.Fatalf("IssueCsatSurveyTokenTx (rated): %v", err)
	}
	if err := repo.SubmitCsatTx(ctx, tenantID, ratedTicket.ID, 5, nil, now); err != nil {
		t.Fatalf("SubmitCsatTx: %v", err)
	}

	// Every dead link gives the same verdict -- that sameness is the property
	// under test, not each individual case.
	for name, token := range map[string]string{
		"expired":       expiredToken,
		"already rated": ratedToken,
		"unknown":       "no-such-token-" + run,
		"empty":         "",
		"over-long":     string(make([]byte, 200)),
	} {
		if _, err := svc.SubmitCsatByToken(context.Background(), token, 3, nil); !errors.Is(err, ErrCsatSurveyNotFound) {
			t.Fatalf("%s link: got %v, want ErrCsatSurveyNotFound", name, err)
		}
	}

	// The rated ticket kept the rating it had; a refused redemption writes
	// nothing.
	stored, err := repo.GetTicketByID(ctx, ratedTicket.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if stored.CsatRating == nil || *stored.CsatRating != 5 {
		t.Fatalf("refused redemption changed the rating: %v", stored.CsatRating)
	}
}

// TestSubmitCsatByToken_RejectsRatingBeforeTouchingToken pins the order: an
// invalid rating is the caller's own error and must be answered without the
// token ever being looked up, so a probe cannot use a bad rating to find out
// whether a token exists.
func TestSubmitCsatByToken_RejectsRatingBeforeTouchingToken(t *testing.T) {
	t.Parallel()

	svc := NewService(&csatLookupSpy{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	spy := svc.repo.(*csatLookupSpy)

	for _, rating := range []int16{0, 6, -1} {
		if _, err := svc.SubmitCsatByToken(context.Background(), "some-token", rating, nil); !errors.Is(err, ErrInvalidCsatRating) {
			t.Fatalf("rating %d: got %v, want ErrInvalidCsatRating", rating, err)
		}
	}
	long := string(make([]byte, csatCommentMaxLength+1))
	if _, err := svc.SubmitCsatByToken(context.Background(), "some-token", 3, &long); !errors.Is(err, ErrCsatCommentTooLong) {
		t.Fatalf("over-long comment: got %v, want ErrCsatCommentTooLong", err)
	}
	if spy.lookups != 0 {
		t.Fatalf("rejected requests hit the token lookup %d time(s)", spy.lookups)
	}
}

// csatLookupSpy counts token lookups and nothing else; every other Repository
// method is inherited from the shared mock.
type csatLookupSpy struct {
	mockRepo
	lookups int
}

func (s *csatLookupSpy) GetCsatSurveyByToken(context.Context, string) (*CsatSurveyToken, error) {
	s.lookups++
	return nil, ErrCsatSurveyNotFound
}
