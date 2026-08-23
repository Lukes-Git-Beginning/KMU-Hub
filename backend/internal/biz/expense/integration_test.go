package expense

// These tests run against a real database because the two things they prove
// cannot be proven against a fake: that every query carries the tenant, and
// that the RLS policy on finance_expenses actually holds under kmuhub_app
// (NOSUPERUSER NOBYPASSRLS). A repository that forgets its tenant predicate
// looks correct in a unit test and produces phantom 404s in production.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedUser inserts a users row. submitted_by/decided_by carry an FK to it, and
// the four-eyes rule is only meaningful with two real identities.
func seedUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, 'hash', $4, 'Test', TRUE, NOW(), NOW())`,
		userID, tenantID, fmt.Sprintf("%s-%s@expense.test", label, userID.String()[:8]), label,
	)
	if err != nil {
		t.Fatalf("seed user %s: %v", label, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func newExpenseRow(t *testing.T, ctx context.Context, repo *PostgresRepository, tenantID, submitter uuid.UUID, description string) *models.Expense {
	t.Helper()
	e := &models.Expense{
		TenantID:    tenantID,
		Description: description,
		Amount:      decimal.RequireFromString("234.50"),
		ExpenseDate: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
		Category:    "Büromaterial",
		Supplier:    "Staples",
		Status:      models.ExpenseStatusPending,
		SubmittedBy: &submitter,
	}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create(%s): %v", description, err)
	}
	return e
}

func TestPostgresRepository_TenantScopedReadsAndWrites(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Private tenants rather than the shared TenantA/TenantB constants: these
	// tests count every row of their tenant, so a neighbour's fixture under
	// -parallel would make them flap.
	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Expense Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Expense Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	submitter := seedUser(t, pool, mine, "submitter")
	foreignSubmitter := seedUser(t, pool, theirs, "foreign")

	// Cleanup by defer, not t.Cleanup: the latter runs after the deferred
	// pool.Close() above and would leave the rows behind on a dead pool.
	ours := newExpenseRow(t, myCtx, repo, mine, submitter, "Ours")
	defer testutil.CleanupRow(t, pool, "finance_expenses", ours.ID)
	foreign := newExpenseRow(t, theirCtx, repo, theirs, foreignSubmitter, "Theirs")
	defer testutil.CleanupRow(t, pool, "finance_expenses", foreign.ID)

	t.Run("list stays inside the tenant", func(t *testing.T) {
		rows, total, err := repo.List(myCtx, mine, ListFilter{Limit: 50})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(rows) != 1 || total != 1 {
			t.Fatalf("list = %d rows / total %d, want 1 / 1", len(rows), total)
		}
		if rows[0].ID != ours.ID {
			t.Errorf("listed %s, want %s", rows[0].ID, ours.ID)
		}
	})

	t.Run("a foreign id reads as not found", func(t *testing.T) {
		if _, err := repo.GetByID(myCtx, mine, foreign.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetByID across tenants: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a foreign id cannot be deleted", func(t *testing.T) {
		if err := repo.Delete(myCtx, mine, foreign.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete across tenants: err = %v, want ErrNotFound", err)
		}
		// And the row is still there for its owner.
		if _, err := repo.GetByID(theirCtx, theirs, foreign.ID); err != nil {
			t.Errorf("the foreign row went missing after a cross-tenant delete: %v", err)
		}
	})

	t.Run("amount survives the round trip exactly", func(t *testing.T) {
		read, err := repo.GetByID(myCtx, mine, ours.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if read.Amount.String() != "234.5" {
			t.Errorf("amount = %s, want 234.5", read.Amount)
		}
		if read.ExpenseDate.Format("2006-01-02") != "2026-05-29" {
			t.Errorf("expense_date = %s, want 2026-05-29", read.ExpenseDate.Format("2006-01-02"))
		}
	})
}

// The approval path end to end, against real rows: the submitter is refused,
// a colleague is not, and the decision is persisted rather than only returned.
func TestService_ApprovalPersists(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Expense Approval Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	submitter := seedUser(t, pool, tenant, "claimer")
	approver := seedUser(t, pool, tenant, "approver")

	created, err := svc.Create(ctx, tenant, submitter, CreateInput{
		Description: "Server-Hosting Hetzner",
		Amount:      "89.00",
		Date:        "2026-05-25",
		Category:    "IT-Infrastruktur",
		Supplier:    "Hetzner Online GmbH",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_expenses", created.ID)

	if _, err := svc.Approve(ctx, tenant, created.ID, submitter); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("self approval err = %v, want ErrSelfApproval", err)
	}
	if _, err := svc.Approve(ctx, tenant, created.ID, approver); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	// Re-read rather than trusting the returned struct: a decision that only
	// exists in memory is the failure this test is here to catch.
	stored, err := repo.GetByID(ctx, tenant, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.Status != models.ExpenseStatusApproved {
		t.Errorf("stored status = %q, want approved", stored.Status)
	}
	if stored.DecidedBy == nil || *stored.DecidedBy != approver {
		t.Errorf("stored decided_by = %v, want %v", stored.DecidedBy, approver)
	}
	if stored.DecidedAt == nil {
		t.Error("stored decided_at is NULL after an approval")
	}
}

// A pending expense is the only thing repo.Delete actually removes; the
// cross-tenant case above proves the guard, this proves the happy path.
func TestPostgresRepository_DeleteRemovesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Expense Delete Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	submitter := seedUser(t, pool, tenant, "deleter")

	e := newExpenseRow(t, ctx, repo, tenant, submitter, "To delete")

	if err := repo.Delete(ctx, tenant, e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, tenant, e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID after delete: err = %v, want ErrNotFound", err)
	}
}

// Update's WHERE clause carries the tenant the same way GetByID's does; this
// proves the RETURNING-less-than-one-row case maps to ErrNotFound rather than
// silently updating nothing and returning success.
func TestPostgresRepository_UpdateMissingOrForeignRowReturnsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Expense Update Mine")
	testutil.EnsureTenant(t, pool, theirs, "Expense Update Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	foreignSubmitter := seedUser(t, pool, theirs, "foreign-update")
	foreign := newExpenseRow(t, theirCtx, repo, theirs, foreignSubmitter, "Theirs")
	defer testutil.CleanupRow(t, pool, "finance_expenses", foreign.ID)

	t.Run("a foreign id updates nothing", func(t *testing.T) {
		e := *foreign
		e.TenantID = mine
		e.Description = "Hijacked"
		if err := repo.Update(myCtx, &e); !errors.Is(err, ErrNotFound) {
			t.Errorf("Update across tenants: err = %v, want ErrNotFound", err)
		}
		stillTheirs, err := repo.GetByID(theirCtx, theirs, foreign.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if stillTheirs.Description != "Theirs" {
			t.Errorf("description = %q, want unchanged %q", stillTheirs.Description, "Theirs")
		}
	})

	t.Run("a missing id updates nothing", func(t *testing.T) {
		e := *foreign
		e.ID = uuid.New()
		e.TenantID = mine
		if err := repo.Update(myCtx, &e); !errors.Is(err, ErrNotFound) {
			t.Errorf("Update of a missing row: err = %v, want ErrNotFound", err)
		}
	})
}

// The repository has no amount guard of its own -- Create relies on the
// finance_expenses_amount_positive CHECK constraint as the last line of
// defense below the service's own parseAmount validation. This proves that
// line actually holds and that the failure surfaces as a real error, not as
// ErrNotFound (scanExpense must not confuse "constraint violation" with
// "no rows").
func TestPostgresRepository_CreateRejectsNonPositiveAmount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Expense Constraint Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	e := &models.Expense{
		TenantID:    tenant,
		Description: "Negative claim",
		Amount:      decimal.RequireFromString("-5.00"),
		ExpenseDate: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
		Status:      models.ExpenseStatusPending,
	}
	err := repo.Create(ctx, e)
	if err == nil {
		t.Fatal("Create with a non-positive amount succeeded, want the CHECK constraint to reject it")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want a database constraint error, not ErrNotFound", err)
	}
}

// The "own" scope has to narrow in SQL, so rows and total agree. A post-filter
// would leave the pager offering a page of rows the caller may not see.
func TestPostgresRepository_ListNarrowsRowsAndTotalTogether(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Expense Own Scope Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	me := seedUser(t, pool, tenant, "me")
	colleague := seedUser(t, pool, tenant, "colleague")

	first := newExpenseRow(t, ctx, repo, tenant, me, "Mine one")
	defer testutil.CleanupRow(t, pool, "finance_expenses", first.ID)
	second := newExpenseRow(t, ctx, repo, tenant, me, "Mine two")
	defer testutil.CleanupRow(t, pool, "finance_expenses", second.ID)
	third := newExpenseRow(t, ctx, repo, tenant, colleague, "Theirs")
	defer testutil.CleanupRow(t, pool, "finance_expenses", third.ID)

	rows, total, err := repo.List(ctx, tenant, ListFilter{SubmittedBy: &me, Limit: 50})
	if err != nil {
		t.Fatalf("List own: %v", err)
	}
	if len(rows) != 2 || total != 2 {
		t.Fatalf("own list = %d rows / total %d, want 2 / 2", len(rows), total)
	}

	// Combined with the status filter, because the placeholder numbering shifts
	// with how many filters came before it.
	rows, total, err = repo.List(ctx, tenant, ListFilter{
		Status: models.ExpenseStatusPending, SubmittedBy: &me, Limit: 50,
	})
	if err != nil {
		t.Fatalf("List own+status: %v", err)
	}
	if len(rows) != 2 || total != 2 {
		t.Fatalf("own+status list = %d rows / total %d, want 2 / 2", len(rows), total)
	}

	rows, total, err = repo.List(ctx, tenant, ListFilter{
		Status: models.ExpenseStatusApproved, SubmittedBy: &me, Limit: 50,
	})
	if err != nil {
		t.Fatalf("List own+approved: %v", err)
	}
	if len(rows) != 0 || total != 0 {
		t.Fatalf("own+approved list = %d rows / total %d, want 0 / 0", len(rows), total)
	}
}

// Offset was untested against real SQL -- every prior List test used the
// default of zero. A caller that pages past row 1 must see the next rows in
// the same order, with the same total, not an empty page or a wrapped-around
// one.
func TestPostgresRepository_ListRespectsOffset(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Expense Offset Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	submitter := seedUser(t, pool, tenant, "pager")

	mk := func(day int, description string) *models.Expense {
		e := &models.Expense{
			TenantID:    tenant,
			Description: description,
			Amount:      decimal.RequireFromString("10.00"),
			ExpenseDate: time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC),
			Status:      models.ExpenseStatusPending,
			SubmittedBy: &submitter,
		}
		if err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create(%s): %v", description, err)
		}
		return e
	}

	// ORDER BY expense_date DESC: the latest day sorts first.
	newest := mk(3, "Newest")
	defer testutil.CleanupRow(t, pool, "finance_expenses", newest.ID)
	middle := mk(2, "Middle")
	defer testutil.CleanupRow(t, pool, "finance_expenses", middle.ID)
	oldest := mk(1, "Oldest")
	defer testutil.CleanupRow(t, pool, "finance_expenses", oldest.ID)

	page1, total, err := repo.List(ctx, tenant, ListFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 3 {
		t.Fatalf("total on page 1 = %d, want 3", total)
	}
	if len(page1) != 2 || page1[0].ID != newest.ID || page1[1].ID != middle.ID {
		t.Fatalf("page 1 = %d rows, want [Newest, Middle]", len(page1))
	}

	page2, total, err := repo.List(ctx, tenant, ListFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if total != 3 {
		t.Fatalf("total on page 2 = %d, want 3 (same filter, same total)", total)
	}
	if len(page2) != 1 || page2[0].ID != oldest.ID {
		t.Fatalf("page 2 = %d rows, want [Oldest]", len(page2))
	}
}
