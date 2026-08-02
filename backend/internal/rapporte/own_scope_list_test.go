package rapporte

// The "own" data scope reaches the database as an author_id predicate, so the
// paged rows and the reported total describe the same set. Driven through the
// service (not the repository) because that is where page/pageSize become
// offset/limit — a filter applied after that split would still hand back the
// tenant-wide count.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestListReports_OwnScopeNarrowsRowsAndTotal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Private tenant: this test counts every report in it.
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Rapporte Own Scope Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	now := time.Now().UTC()

	me, colleague := uuid.New(), uuid.New()

	newReport := func(title string, author uuid.UUID, status ReportStatus) *WorkReport {
		report := &WorkReport{
			ID: uuid.New(), TenantID: tenant, Title: title, Status: status,
			AuthorID: author, ReportDate: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateReport(ctx, report); err != nil {
			t.Fatalf("CreateReport(%s): %v", title, err)
		}
		return report
	}

	mine := newReport("Mine, submitted", me, StatusSubmitted)
	defer testutil.CleanupRow(t, pool, "work_reports", mine.ID)
	mineDraft := newReport("Mine, draft", me, StatusDraft)
	defer testutil.CleanupRow(t, pool, "work_reports", mineDraft.ID)
	foreign := newReport("Theirs, submitted", colleague, StatusSubmitted)
	defer testutil.CleanupRow(t, pool, "work_reports", foreign.ID)

	reports, total, err := svc.ListReports(ctx, ListReportsInput{
		TenantID: tenant, AuthorID: &me, Page: 1, PageSize: 50,
	})
	if err != nil {
		t.Fatalf("ListReports (own scope): %v", err)
	}
	if total != 2 || len(reports) != 2 {
		t.Fatalf("own scope: got %d rows / total %d, want 2 / 2", len(reports), total)
	}
	for _, report := range reports {
		if report.AuthorID != me {
			t.Fatalf("a foreign author's report leaked into the own-scoped list: %q", report.Title)
		}
	}

	// The approval queue runs through the same filter, so a member at scope
	// "own" reviews their own submissions instead of the whole tenant's.
	pending, pendingTotal, err := svc.ListPendingApprovals(ctx, tenant, &me, 1, 50)
	if err != nil {
		t.Fatalf("ListPendingApprovals (own scope): %v", err)
	}
	if pendingTotal != 1 || len(pending) != 1 {
		t.Fatalf("own-scoped approval queue: got %d rows / total %d, want 1 / 1", len(pending), pendingTotal)
	}
	if pending[0].ID != mine.ID {
		t.Fatalf("own-scoped approval queue returned the wrong report: %q", pending[0].Title)
	}

	// Without the filter the same queue still shows both submissions — the
	// narrowing must come from the scope, not from a change in default
	// behaviour for everyone else.
	all, allTotal, err := svc.ListPendingApprovals(ctx, tenant, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListPendingApprovals (unscoped): %v", err)
	}
	if allTotal != 2 || len(all) != 2 {
		t.Fatalf("unscoped approval queue: got %d rows / total %d, want 2 / 2", len(all), allTotal)
	}
}
