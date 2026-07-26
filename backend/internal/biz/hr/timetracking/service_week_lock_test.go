package timetracking

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// lockedWeekFixture submits (and optionally approves) the week containing `day`
// for one employee, and returns the service plus the identifiers used.
func lockedWeekFixture(t *testing.T, day time.Time, approve bool) (*Service, uuid.UUID, uuid.UUID) {
	t.Helper()
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, day)
	require.NoError(t, err)
	if approve {
		_, err = svc.ApproveWeek(ctx, tenantID, approverID, employeeID, day)
		require.NoError(t, err)
	}
	return svc, tenantID, employeeID
}

func manualEntryAt(tenantID, employeeID uuid.UUID, clockIn time.Time) ManualEntryInput {
	return ManualEntryInput{
		TenantID:     tenantID,
		EmployeeID:   employeeID,
		ClockIn:      clockIn,
		ClockOut:     clockIn.Add(8 * time.Hour),
		BreakMinutes: 30,
	}
}

// ============================================================================
// weekStartOf
// ============================================================================

func TestWeekStartOf_EveryDayMapsToItsMonday(t *testing.T) {
	monday := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

	for offset := 0; offset < 7; offset++ {
		day := monday.AddDate(0, 0, offset).Add(17*time.Hour + 42*time.Minute)
		assert.Equal(t, monday, weekStartOf(day),
			"day %s must belong to week starting %s", day.Format(time.RFC3339), monday.Format("2006-01-02"))
	}

	// Sunday belongs to the week that started six days earlier, not the next one.
	sunday := monday.AddDate(0, 0, 6)
	assert.Equal(t, monday, weekStartOf(sunday))
	// And the Monday after is a new week.
	assert.Equal(t, monday.AddDate(0, 0, 7), weekStartOf(monday.AddDate(0, 0, 7)))
}

// ============================================================================
// The lock: writes into a submitted or approved week
// ============================================================================

func TestClockIn_SubmittedWeek_IsRefused(t *testing.T) {
	svc, tenantID, employeeID := lockedWeekFixture(t, time.Now(), false)

	_, _, err := svc.ClockIn(context.Background(), tenantID, employeeID)
	assert.ErrorIs(t, err, ErrWeekLocked)
}

func TestCreateManualEntry_ApprovedWeek_IsRefused(t *testing.T) {
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, true)

	_, err := svc.CreateManualEntry(context.Background(), manualEntryAt(tenantID, employeeID, lastWeek))
	assert.ErrorIs(t, err, ErrWeekLocked)
}

func TestCreateManualEntry_RejectedWeek_IsAllowed(t *testing.T) {
	ctx := context.Background()
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, false)

	_, err := svc.RejectWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek, "missing project codes")
	require.NoError(t, err)

	// Rejection exists to get the week fixed, so it must not stay locked.
	entry, err := svc.CreateManualEntry(ctx, manualEntryAt(tenantID, employeeID, lastWeek))
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCompleted, entry.Status)
}

func TestCreateManualEntry_UnlockedWeek_IsUnaffectedByAnotherLockedWeek(t *testing.T) {
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, true)

	// Same employee, the week before the approved one: still open.
	twoWeeksAgo := lastWeek.AddDate(0, 0, -7)
	_, err := svc.CreateManualEntry(context.Background(), manualEntryAt(tenantID, employeeID, twoWeeksAgo))
	require.NoError(t, err)
}

func TestSubmitTimeCorrection_OriginalInApprovedWeek_IsRefused(t *testing.T) {
	ctx := context.Background()
	lastWeek := weekStartOf(time.Now().AddDate(0, 0, -7)).Add(9 * time.Hour)
	svc, workRepo := newExtendedTestService()
	tenantID := uuid.New()
	employeeID := uuid.New()

	original := &models.HRWorkTimeEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		ClockIn:    lastWeek,
		Status:     models.WorkTimeStatusCompleted,
	}
	require.NoError(t, workRepo.Create(ctx, original))

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, lastWeek)
	require.NoError(t, err)
	_, err = svc.ApproveWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek)
	require.NoError(t, err)

	_, err = svc.SubmitTimeCorrection(ctx, tenantID, employeeID, CorrectionInput{
		OriginalEntryID:       original.ID,
		CorrectedClockIn:      lastWeek,
		CorrectedClockOut:     lastWeek.Add(7 * time.Hour),
		CorrectedBreakMinutes: 30,
		Reason:                "forgot to clock out",
	})
	assert.ErrorIs(t, err, ErrWeekLocked)
}

func TestApproveTimeCorrection_WeekApprovedAfterRequest_IsRefused(t *testing.T) {
	ctx := context.Background()
	lastWeek := weekStartOf(time.Now().AddDate(0, 0, -7)).Add(9 * time.Hour)
	svc, workRepo := newExtendedTestService()
	tenantID := uuid.New()
	employeeID := uuid.New()

	original := &models.HRWorkTimeEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		ClockIn:    lastWeek,
		Status:     models.WorkTimeStatusCompleted,
	}
	require.NoError(t, workRepo.Create(ctx, original))

	// Correction requested while the week is still open.
	correction, err := svc.SubmitTimeCorrection(ctx, tenantID, employeeID, CorrectionInput{
		OriginalEntryID:       original.ID,
		CorrectedClockIn:      lastWeek,
		CorrectedClockOut:     lastWeek.Add(7 * time.Hour),
		CorrectedBreakMinutes: 30,
		Reason:                "forgot to clock out",
	})
	require.NoError(t, err)

	_, err = svc.SubmitWeek(ctx, tenantID, employeeID, lastWeek)
	require.NoError(t, err)
	_, err = svc.ApproveWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek)
	require.NoError(t, err)

	// Approval is what makes the correction count towards the week, so it is a
	// write into a signed-off total even though the request predates the lock.
	_, err = svc.ApproveTimeCorrection(ctx, correction.ID, uuid.New())
	assert.ErrorIs(t, err, ErrWeekLocked)
}

// ============================================================================
// ReopenWeek
// ============================================================================

func TestReopenWeek_ApprovedWeek_BecomesWritableAgain(t *testing.T) {
	ctx := context.Background()
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, true)

	wa, err := svc.ReopenWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek, "wrong project booked")
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalOpen, wa.Status)
	assert.Nil(t, wa.SubmittedAt)
	assert.Nil(t, wa.ApprovedBy)
	assert.Nil(t, wa.ApprovedAt)
	assert.Nil(t, wa.RejectionReason)

	_, err = svc.CreateManualEntry(ctx, manualEntryAt(tenantID, employeeID, lastWeek))
	require.NoError(t, err)
}

func TestReopenWeek_SubmittedWeek_BecomesWritableAgain(t *testing.T) {
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, false)

	wa, err := svc.ReopenWeek(context.Background(), tenantID, uuid.New(), employeeID, lastWeek, "")
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalOpen, wa.Status)
}

func TestReopenWeek_OpenWeek_IsRefused(t *testing.T) {
	ctx := context.Background()
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, false)

	_, err := svc.ReopenWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek, "")
	require.NoError(t, err)

	// Already open — there is nothing to unlock.
	_, err = svc.ReopenWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek, "")
	assert.ErrorIs(t, err, ErrWeekNotLocked)
}

func TestReopenWeek_UnknownWeek_IsNotFound(t *testing.T) {
	svc, _ := newExtendedTestService()

	_, err := svc.ReopenWeek(context.Background(), uuid.New(), uuid.New(), uuid.New(), time.Now(), "")
	assert.ErrorIs(t, err, ErrWeekApprovalNotFound)
}

// A reopened week can be submitted again — otherwise reopening would strand the
// week outside the approval workflow.
func TestSubmitWeek_AfterReopen_IsAllowed(t *testing.T) {
	ctx := context.Background()
	lastWeek := time.Now().AddDate(0, 0, -7)
	svc, tenantID, employeeID := lockedWeekFixture(t, lastWeek, true)

	_, err := svc.ReopenWeek(ctx, tenantID, uuid.New(), employeeID, lastWeek, "")
	require.NoError(t, err)

	wa, err := svc.SubmitWeek(ctx, tenantID, employeeID, lastWeek)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, wa.Status)
}
