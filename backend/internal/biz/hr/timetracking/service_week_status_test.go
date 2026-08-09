package timetracking

// GetMyWeekStatus and the illegal edges of the week-approval state machine.
// service_extended_test.go covers the happy paths (submit, approve, reject)
// and service_week_lock_test.go covers the lock itself; this file covers the
// two things neither did: GetMyWeekStatus was at 0% coverage, and the only
// illegal transition under test was ApproveWeek on a never-submitted week —
// double-approve, double-reject, and approve/reject on the wrong terminal
// state were all unexercised.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var mondayForStatusTests = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

// ============================================================================
// GetMyWeekStatus
// ============================================================================

func TestGetMyWeekStatus_NoRecord_SynthesizesOpen(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	wa, totalMinutes, targetMinutes, err := svc.GetMyWeekStatus(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	require.NotNil(t, wa)
	assert.Equal(t, models.WeekApprovalOpen, wa.Status)
	assert.Equal(t, tenantID, wa.TenantID)
	assert.Equal(t, employeeID, wa.EmployeeID)
	assert.Equal(t, 0, totalMinutes, "no weekly summary configured on the mock, so total must be 0")
	assert.Equal(t, 5*8*60, targetMinutes, "default fixture profile works 5 days/week")
}

func TestGetMyWeekStatus_ExistingRecord_ReturnsIt(t *testing.T) {
	svc, workRepo := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	workRepo.weeklySummary = &WeeklySummary{NetWorkMinutes: 1234}

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)

	wa, totalMinutes, _, err := svc.GetMyWeekStatus(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, wa.Status)
	assert.Equal(t, 1234, totalMinutes)
}

// TestGetMyWeekStatus_ReadFailure_IsNotTreatedAsOpen pins the branch that
// distinguishes "no record" (errors.Is ErrWeekApprovalNotFound, synthesise
// open) from a genuine repository failure. Silently falling back to open on
// any error would tell an employee an approved week is still editable.
func TestGetMyWeekStatus_ReadFailure_IsNotTreatedAsOpen(t *testing.T) {
	svc, _ := newExtendedTestService()
	waRepo := svc.weekApprovalRepo.(*mockWeekApprovalRepo)
	waRepo.getErr = errors.New("connection reset")

	_, _, _, err := svc.GetMyWeekStatus(context.Background(), uuid.New(), uuid.New(), mondayForStatusTests)
	assert.ErrorContains(t, err, "connection reset")
}

// ============================================================================
// Illegal transitions not covered elsewhere
// ============================================================================

func TestSubmitWeek_AfterReject_IsAllowed(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	_, err = svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "wrong hours")
	require.NoError(t, err)

	// Rejection exists to get the week fixed and resubmitted — a second
	// SubmitWeek must not be treated as "already submitted".
	wa, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, wa.Status)
}

func TestApproveWeek_ApprovedWeek_IsRefused(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	_, err = svc.ApproveWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests)
	require.NoError(t, err)

	// Double-approve: the week is already terminal, not submitted.
	_, err = svc.ApproveWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests)
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}

func TestApproveWeek_RejectedWeek_IsRefused(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	_, err = svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "wrong hours")
	require.NoError(t, err)

	// A rejected week must be resubmitted before it can be approved.
	_, err = svc.ApproveWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests)
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}

func TestRejectWeek_NeverSubmittedWeek_IsRefused(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	// An explicit open record (as ReopenWeek would leave behind), never submitted.
	now := time.Now()
	require.NoError(t, svc.weekApprovalRepo.Upsert(ctx, &models.HRWeekApproval{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		WeekStart:  mondayForStatusTests,
		Status:     models.WeekApprovalOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	}))

	_, err := svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "n/a")
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}

func TestRejectWeek_ApprovedWeek_IsRefused(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	_, err = svc.ApproveWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests)
	require.NoError(t, err)

	// An approved week is signed off; it must be reopened, not rejected.
	_, err = svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "too late")
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}

func TestRejectWeek_RejectedWeek_IsRefused(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, mondayForStatusTests)
	require.NoError(t, err)
	_, err = svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "first reason")
	require.NoError(t, err)

	// Double-reject: the week is no longer in the submitted state.
	_, err = svc.RejectWeek(ctx, tenantID, approverID, employeeID, mondayForStatusTests, "second reason")
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}
