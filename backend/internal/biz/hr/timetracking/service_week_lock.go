package timetracking

// Week locking. Submitting a week hands a number to an approver, approving it
// signs that number off. Until this file existed both were decorative: every
// write path — clock-in, manual entry, correction — happily added time to a week
// that was already submitted or approved, and the signed-off total silently
// changed afterwards. A locked week now refuses time writes, and the only way
// back to editable is ReopenWeek, which is an explicit and logged act.

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// weekStartOf truncates t to Monday 00:00 in t's own location.
//
// Every week-scoped read, summary and approval key runs through this one
// function. Written twice, the lock and the weekly total would eventually
// disagree about which week a timestamp belongs to, and an entry could land in a
// week that the summary counts elsewhere.
func weekStartOf(t time.Time) time.Time {
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// weekLocked reports whether a week in this state refuses further time writes.
// Rejected weeks are open again by design: rejection exists to get them fixed.
func weekLocked(status models.WeekApprovalStatus) bool {
	return status == models.WeekApprovalSubmitted || status == models.WeekApprovalApproved
}

// assertWeekEditable refuses a write whose work time falls into a submitted or
// approved week.
//
// `at` is the timestamp the work time itself carries, never the request time: a
// manual entry backdated into an approved week is precisely the case this
// guards. Absent approval record means the week was never submitted, so it is
// open. Any other repository error refuses the write — failing open here would
// let a database hiccup wave through exactly the mutation being prevented.
func (s *Service) assertWeekEditable(ctx context.Context, tenantID, employeeID uuid.UUID, at time.Time) error {
	if s.weekApprovalRepo == nil {
		// Extended repositories not wired (construction without SetExtendedRepos).
		// There is no approval workflow in that deployment, so nothing to enforce.
		return nil
	}
	wa, err := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStartOf(at))
	if errors.Is(err, ErrWeekApprovalNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if weekLocked(wa.Status) {
		return ErrWeekLocked
	}
	return nil
}

// ReopenWeek returns a submitted or approved week to the open state so it can be
// corrected. This is the only way a locked week becomes writable again — no
// write path unlocks one as a side effect.
//
// Without it the lock would be a trap: an approved week could never be fixed,
// and the pressure would be to weaken the lock instead.
func (s *Service) ReopenWeek(
	ctx context.Context,
	tenantID, approverID, employeeID uuid.UUID,
	weekStart time.Time,
	reason string,
) (*models.HRWeekApproval, error) {
	weekStart = weekStartOf(weekStart)

	wa, err := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStart)
	if err != nil {
		return nil, err
	}
	if !weekLocked(wa.Status) {
		return nil, ErrWeekNotLocked
	}

	now := time.Now()
	previous := wa.Status
	wa.Status = models.WeekApprovalOpen
	wa.SubmittedAt = nil
	wa.ApprovedBy = nil
	wa.ApprovedAt = nil
	// The week is neutral again, so a stale rejection reason from an earlier
	// round must not stay attached to it.
	wa.RejectionReason = nil
	wa.UpdatedAt = now

	if upsertErr := s.weekApprovalRepo.Upsert(ctx, wa); upsertErr != nil {
		return nil, upsertErr
	}

	// lean: the reason is logged, not persisted — hr_week_approvals has no column
	// for it and rejection_reason means something else. Add a reopen_reason column
	// when week approvals need a queryable audit trail (GoBD/works council).
	slog.Info("hr.time.week_reopened",
		"employee_id", employeeID,
		"approver_id", approverID,
		"week_start", weekStart.Format("2006-01-02"),
		"previous_status", string(previous),
		"reason", reason,
	)
	return wa, nil
}
