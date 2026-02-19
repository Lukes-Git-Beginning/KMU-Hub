package compliance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// CalculateRequiredBreak tests
// ============================================================================

func TestCalculateRequiredBreak_5Hours(t *testing.T) {
	// 5h work = 0 min break
	result := CalculateRequiredBreak(300) // 5 * 60
	assert.Equal(t, 0, result)
}

func TestCalculateRequiredBreak_6Hours01Min(t *testing.T) {
	// 6h01m work = 30 min break
	result := CalculateRequiredBreak(361) // 6*60 + 1
	assert.Equal(t, 30, result)
}

func TestCalculateRequiredBreak_9Hours01Min(t *testing.T) {
	// 9h01m work = 45 min break
	result := CalculateRequiredBreak(541) // 9*60 + 1
	assert.Equal(t, 45, result)
}

// ============================================================================
// CalculateAutoBreakDeduction tests
// ============================================================================

func TestCalculateAutoBreakDeduction_SufficientManualBreak(t *testing.T) {
	// 7h work, 30min manual break = 0 auto-deduct
	result := CalculateAutoBreakDeduction(420, 30)
	assert.Equal(t, 0, result)
}

func TestCalculateAutoBreakDeduction_PartialManualBreak(t *testing.T) {
	// 7h work, 20min manual break = 10 auto-deduct
	result := CalculateAutoBreakDeduction(420, 20)
	assert.Equal(t, 10, result)
}

func TestCalculateAutoBreakDeduction_NoManualBreak(t *testing.T) {
	// 7h work, 0 manual break = 30 auto-deduct
	result := CalculateAutoBreakDeduction(420, 0)
	assert.Equal(t, 30, result)
}

func TestCalculateAutoBreakDeduction_LongWorkPartialBreak(t *testing.T) {
	// 10h work, 30min manual break = 15 auto-deduct (needs 45)
	result := CalculateAutoBreakDeduction(600, 30)
	assert.Equal(t, 15, result)
}

// ============================================================================
// CheckWorkTime tests
// ============================================================================

func TestCheckWorkTime_7Hours_None(t *testing.T) {
	// 7h total = SeverityNone
	result := CheckWorkTime(420) // 7 * 60
	assert.Equal(t, SeverityNone, result.Severity)
	assert.Equal(t, 420, result.TotalWorkedMinutes)
}

func TestCheckWorkTime_8Hours_Info(t *testing.T) {
	// 8h total = SeverityInfo
	result := CheckWorkTime(480) // 8 * 60
	assert.Equal(t, SeverityInfo, result.Severity)
}

func TestCheckWorkTime_9Hours_Warning(t *testing.T) {
	// 9h total = SeverityWarning
	result := CheckWorkTime(540) // 9 * 60
	assert.Equal(t, SeverityWarning, result.Severity)
}

func TestCheckWorkTime_10Hours01Min_Error(t *testing.T) {
	// 10h01m total = SeverityError
	result := CheckWorkTime(601) // 10*60 + 1
	assert.Equal(t, SeverityError, result.Severity)
}

// ============================================================================
// CheckRestPeriod tests
// ============================================================================

func TestCheckRestPeriod_12HourGap_NoViolation(t *testing.T) {
	// 12h gap = no violation
	currentStart := time.Date(2025, 1, 2, 8, 0, 0, 0, time.UTC)
	previousEnd := time.Date(2025, 1, 1, 20, 0, 0, 0, time.UTC)

	violation, hours := CheckRestPeriod(currentStart, &previousEnd)
	assert.False(t, violation)
	assert.InDelta(t, 12.0, hours, 0.01)
}

func TestCheckRestPeriod_10HourGap_Violation(t *testing.T) {
	// 10h gap = violation
	currentStart := time.Date(2025, 1, 2, 6, 0, 0, 0, time.UTC)
	previousEnd := time.Date(2025, 1, 1, 20, 0, 0, 0, time.UTC)

	violation, hours := CheckRestPeriod(currentStart, &previousEnd)
	assert.True(t, violation)
	assert.InDelta(t, 10.0, hours, 0.01)
}

func TestCheckRestPeriod_NilPreviousEnd_NoViolation(t *testing.T) {
	// nil previous end = no violation (first day)
	currentStart := time.Date(2025, 1, 2, 8, 0, 0, 0, time.UTC)

	violation, hours := CheckRestPeriod(currentStart, nil)
	assert.False(t, violation)
	assert.Equal(t, 0.0, hours)
}

// ============================================================================
// CalculateNetWorkMinutes tests
// ============================================================================

func TestCalculateNetWorkMinutes(t *testing.T) {
	// 8h clock in/out, 30min break = 450 net minutes
	clockIn := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	clockOut := time.Date(2025, 1, 1, 16, 0, 0, 0, time.UTC)

	result := CalculateNetWorkMinutes(clockIn, clockOut, 20, 10) // 20 manual + 10 auto = 30 total
	assert.Equal(t, 450, result)
}
