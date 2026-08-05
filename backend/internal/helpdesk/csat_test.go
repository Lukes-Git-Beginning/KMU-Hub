package helpdesk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedTicket puts a minimal open ticket into the mock repo and returns it.
func seedTicket(h *testHarness, tenantID uuid.UUID) *Ticket {
	t := &Ticket{
		ID:       uuid.New(),
		TenantID: tenantID,
		Subject:  "Printer on fire",
		Status:   TicketStatusClosed,
		Priority: TicketPriorityNormal,
	}
	h.repo.tickets[t.ID] = t
	return t
}

func TestSubmitCsat_Success(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedTicket(h, tenantID)
	comment := "  quick and friendly  "

	got, err := h.svc.SubmitCsat(context.Background(), ticket.ID, tenantID, 5, &comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.CsatRating == nil || *got.CsatRating != 5 {
		t.Fatalf("expected returned ticket to carry rating 5, got %v", got.CsatRating)
	}
	if got.CsatComment == nil || *got.CsatComment != comment {
		t.Fatalf("expected comment to be passed through, got %v", got.CsatComment)
	}
	// The rating has to reach persistence, not just the struct handed back --
	// a service that only decorates its return value makes the read path lie.
	if h.repo.csatCalls != 1 {
		t.Fatalf("expected exactly one persistence call, got %d", h.repo.csatCalls)
	}
	if stored := h.repo.tickets[ticket.ID]; stored.CsatRating == nil || *stored.CsatRating != 5 {
		t.Fatalf("expected stored ticket mirror to be 5, got %v", stored.CsatRating)
	}
}

func TestSubmitCsat_RejectsOutOfRange(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedTicket(h, tenantID)

	for _, rating := range []int16{0, -1, 6, 127} {
		if _, err := h.svc.SubmitCsat(context.Background(), ticket.ID, tenantID, rating, nil); !errors.Is(err, ErrInvalidCsatRating) {
			t.Errorf("rating %d: expected ErrInvalidCsatRating, got %v", rating, err)
		}
	}
	if h.repo.csatCalls != 0 {
		t.Errorf("expected no persistence for invalid ratings, got %d calls", h.repo.csatCalls)
	}
	if stored := h.repo.tickets[ticket.ID]; stored.CsatRating != nil {
		t.Errorf("expected ticket mirror to stay empty, got %v", *stored.CsatRating)
	}
}

func TestSubmitCsat_SecondRatingUpdatesInsteadOfFailing(t *testing.T) {
	h := newTestHarness()
	tenantID := uuid.New()
	ticket := seedTicket(h, tenantID)

	if _, err := h.svc.SubmitCsat(context.Background(), ticket.ID, tenantID, 2, nil); err != nil {
		t.Fatalf("first rating: %v", err)
	}
	got, err := h.svc.SubmitCsat(context.Background(), ticket.ID, tenantID, 4, nil)
	if err != nil {
		t.Fatalf("second rating: %v", err)
	}
	if got.CsatRating == nil || *got.CsatRating != 4 {
		t.Fatalf("expected the changed rating 4 to win, got %v", got.CsatRating)
	}
}

func TestSubmitCsat_ForeignTenantIsNotFound(t *testing.T) {
	h := newTestHarness()
	ticket := seedTicket(h, uuid.New())

	_, err := h.svc.SubmitCsat(context.Background(), ticket.ID, uuid.New(), 3, nil)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for a foreign tenant, got %v", err)
	}
	if h.repo.csatCalls != 0 {
		t.Errorf("expected no persistence for a foreign ticket, got %d calls", h.repo.csatCalls)
	}
}

// ---------------------------------------------------------------------------
// Repository level, against a real database
// ---------------------------------------------------------------------------

// TestSubmitCsatTx_UpsertsAndStaysInTenant exercises the transaction against
// Postgres: the mock cannot prove that the ON CONFLICT target matches the real
// unique index, that rating and submitted_at satisfy
// chk_ticket_csat_rating_submitted, or that RLS stops a foreign tenant.
func TestSubmitCsatTx_UpsertsAndStaysInTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk CSAT Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk CSAT Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "CSAT Test Ticket",
		Status: TicketStatusClosed, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	// The response row cascades away with the ticket.
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	// A foreign tenant passing the row's real tenantID must still be stopped --
	// by RLS on the ticket UPDATE, before anything is written.
	if err := repo.SubmitCsatTx(ctxOther, tenantOwn, ticket.ID, 5, nil, now); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("SubmitCsatTx (foreign ctx): expected ErrTicketNotFound, got %v", err)
	}

	comment := "everything worked"
	if err := repo.SubmitCsatTx(ctxOwn, tenantOwn, ticket.ID, 4, &comment, now); err != nil {
		t.Fatalf("SubmitCsatTx (own ctx): %v", err)
	}

	got, err := repo.GetTicketByID(ctxOwn, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.CsatRating == nil || *got.CsatRating != 4 {
		t.Fatalf("ticket read did not return the rating: %v", got.CsatRating)
	}
	if got.CsatComment == nil || *got.CsatComment != comment {
		t.Fatalf("ticket read did not return the comment: %v", got.CsatComment)
	}

	// Rating again must update the single row, not add a second one and not
	// trip the per-ticket unique index.
	if err := repo.SubmitCsatTx(ctxOwn, tenantOwn, ticket.ID, 2, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SubmitCsatTx (second rating): %v", err)
	}

	var (
		rows        int
		rating      int16
		submittedAt *time.Time
		storedNote  *string
	)
	if err := pool.QueryRow(sysCtx,
		`SELECT count(*) FROM ticket_csat_responses WHERE ticket_id = $1`, ticket.ID,
	).Scan(&rows); err != nil {
		t.Fatalf("count responses: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly one CSAT response row, got %d", rows)
	}
	if err := pool.QueryRow(sysCtx,
		`SELECT rating, comment, submitted_at FROM ticket_csat_responses WHERE ticket_id = $1`, ticket.ID,
	).Scan(&rating, &storedNote, &submittedAt); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if rating != 2 {
		t.Fatalf("expected the second rating to win, got %d", rating)
	}
	if storedNote != nil {
		t.Fatalf("expected the comment to be cleared with the new rating, got %q", *storedNote)
	}
	if submittedAt == nil {
		t.Fatal("submitted_at must move with the rating")
	}

	// The foreign tenant sees neither the response nor the mirror.
	var foreignRows int
	if err := pool.QueryRow(ctxOther,
		`SELECT count(*) FROM ticket_csat_responses WHERE ticket_id = $1`, ticket.ID,
	).Scan(&foreignRows); err != nil {
		t.Fatalf("count responses (foreign ctx): %v", err)
	}
	if foreignRows != 0 {
		t.Fatalf("foreign tenant sees %d CSAT rows", foreignRows)
	}
}
