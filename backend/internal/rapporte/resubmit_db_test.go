package rapporte

// A rejected report is corrected by its author and handed in again. Runs against the real
// schema so the atomic UPDATE ... WHERE status IN (...) is what is under test, not a mock.

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestSubmitReport_RejectedReportCanBeResubmitted(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	svc := NewService(repo)

	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle Resubmit", StatusDraft)
	reviewer := uuid.New()
	in := SubmitReportInput{TenantID: tenantID, ReportID: report.ID}

	if _, err := svc.SubmitReport(ctx, in); err != nil {
		t.Fatalf("draft -> submitted: %v", err)
	}
	if _, err := svc.RejectReport(ctx, RejectReportInput{
		TenantID: tenantID, ReportID: report.ID, ReviewerID: reviewer, ReviewNote: "Stunden fehlen",
	}); err != nil {
		t.Fatalf("submitted -> rejected: %v", err)
	}

	resubmitted, err := svc.SubmitReport(ctx, in)
	if err != nil {
		t.Fatalf("rejected -> submitted: %v", err)
	}
	if resubmitted.Status != StatusSubmitted {
		t.Fatalf("expected submitted, got %q", resubmitted.Status)
	}
	if resubmitted.ReviewNote != "Stunden fehlen" {
		t.Fatalf("expected the review note to stay as history, got %q", resubmitted.ReviewNote)
	}

	// submitted -> submitted is not a transition.
	if _, err := svc.SubmitReport(ctx, in); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition for a second submit, got %v", err)
	}

	if _, err := svc.ApproveReport(ctx, ApproveReportInput{
		TenantID: tenantID, ReportID: report.ID, ReviewerID: reviewer,
	}); err != nil {
		t.Fatalf("submitted -> approved: %v", err)
	}
	if _, err := svc.SubmitReport(ctx, in); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition for an approved report, got %v", err)
	}
}

func TestSubmitReport_ParallelSubmitOfRejectedReportCountsOnce(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	svc := NewService(repo)

	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle Parallel", StatusRejected)
	in := SubmitReportInput{TenantID: tenantID, ReportID: report.ID}

	const callers = 8
	errs := make([]error, callers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = svc.SubmitReport(context.WithoutCancel(ctx), in)
		}()
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, ErrInvalidStateTransition):
			t.Fatalf("unexpected error from a losing submit: %v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly one submit to win, got %d", succeeded)
	}
}
