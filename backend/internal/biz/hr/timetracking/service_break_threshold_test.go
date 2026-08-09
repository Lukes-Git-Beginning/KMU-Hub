package timetracking

// The 6h auto-break threshold as ClockOut actually wires it. compliance's own
// tests already pin CalculateRequiredBreak at 6h01min; what they cannot catch
// is a ClockOut regression that feeds the wrong value into that formula (off
// by a status, a rounding direction, or a manual-break double count). These
// two tests exercise the service's real minute math right at the boundary,
// not the formula in isolation.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClockOut_ExactlySixHours_NoAutoDeduction(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	entry.ClockIn = time.Now().Add(-6 * time.Hour)
	workRepo.entries[entry.ID] = entry
	workRepo.activeShift = entry

	completed, _, err := svc.ClockOut(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.Equal(t, 0, completed.AutoBreakDeducted,
		"exactly 6h worked must not cross the >6h break threshold")
}

func TestClockOut_SixHoursOneMinute_AutoDeducts30Min(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	entry.ClockIn = time.Now().Add(-6*time.Hour - time.Minute)
	workRepo.entries[entry.ID] = entry
	workRepo.activeShift = entry

	completed, _, err := svc.ClockOut(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.Equal(t, 30, completed.AutoBreakDeducted,
		"one minute past the 6h threshold must trigger the 30min auto-deduction")
}
