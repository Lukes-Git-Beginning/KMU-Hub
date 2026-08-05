package helpdesk

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestGetHelpdeskStats_CsatAverage exercises the real SQL aggregation against
// Postgres: it has to average only submitted ratings, stay tenant-scoped, and
// return a defined empty value instead of NULL/division-by-zero when a tenant
// has no ratings yet.
func TestGetHelpdeskStats_CsatAverage(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantRated, tenantEmpty, tenantOther := uuid.New(), uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantRated, "Helpdesk Stats Rated Tenant")
	testutil.EnsureTenant(t, pool, tenantEmpty, "Helpdesk Stats Empty Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Stats Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxRated := testutil.WithTenantCtx(context.Background(), tenantRated)
	ctxEmpty := testutil.WithTenantCtx(context.Background(), tenantEmpty)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	// Cleaned up via a single defer at the end of the test body (registered
	// after pool.Close()), so it runs before the pool closes -- see csat_test.go
	// for the same ordering requirement.
	var ticketIDs []uuid.UUID
	defer func() {
		for _, id := range ticketIDs {
			testutil.CleanupRow(t, pool, "tickets", id)
		}
	}()

	newClosedTicket := func(ctx context.Context, tenantID uuid.UUID) *Ticket {
		ticket := &Ticket{
			ID: uuid.New(), TenantID: tenantID, Subject: "CSAT Stats Ticket",
			Status: TicketStatusClosed, Priority: TicketPriorityNormal,
			RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateTicket(ctx, ticket); err != nil {
			t.Fatalf("CreateTicket: %v", err)
		}
		ticketIDs = append(ticketIDs, ticket.ID)
		return ticket
	}

	// Three submitted ratings, average 4.0 exactly (avoids rounding ambiguity).
	for _, rating := range []int16{5, 3, 4} {
		ticket := newClosedTicket(ctxRated, tenantRated)
		if err := repo.SubmitCsatTx(ctxRated, tenantRated, ticket.ID, rating, nil, now); err != nil {
			t.Fatalf("SubmitCsatTx: %v", err)
		}
	}

	// A pending survey (token issued, no rating yet) must not skew the average
	// or count as an answer -- rating/submitted_at are both NULL here.
	pendingTicket := newClosedTicket(ctxRated, tenantRated)
	pendingToken := "pending-survey-token-" + uuid.NewString()
	if _, err := pool.Exec(ctxRated,
		`INSERT INTO ticket_csat_responses (tenant_id, ticket_id, token, token_expires_at)
		 VALUES ($1, $2, $3, $4)`,
		tenantRated, pendingTicket.ID, pendingToken, now.Add(72*time.Hour),
	); err != nil {
		t.Fatalf("insert pending survey row: %v", err)
	}

	// A rating in a different tenant must not leak into tenantRated's average.
	otherTicket := newClosedTicket(ctxOther, tenantOther)
	if err := repo.SubmitCsatTx(ctxOther, tenantOther, otherTicket.ID, 1, nil, now); err != nil {
		t.Fatalf("SubmitCsatTx (other tenant): %v", err)
	}

	stats, err := repo.GetHelpdeskStats(ctxRated, tenantRated)
	if err != nil {
		t.Fatalf("GetHelpdeskStats: %v", err)
	}
	if stats.CustomerSatisfaction != "4.0/5" {
		t.Fatalf("expected average of submitted ratings 4.0/5, got %q", stats.CustomerSatisfaction)
	}

	emptyStats, err := repo.GetHelpdeskStats(ctxEmpty, tenantEmpty)
	if err != nil {
		t.Fatalf("GetHelpdeskStats (empty tenant): %v", err)
	}
	if emptyStats.CustomerSatisfaction != "–" {
		t.Fatalf("expected defined empty value for a tenant with no ratings, got %q", emptyStats.CustomerSatisfaction)
	}
}
