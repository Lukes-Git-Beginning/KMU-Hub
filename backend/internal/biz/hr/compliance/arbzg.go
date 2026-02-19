package compliance

import (
	"time"
)

// ArbZG thresholds (in minutes)
const (
	// Break thresholds: >6h needs 30min, >9h needs 45min
	breakThreshold6h  = 360 // 6 * 60
	breakThreshold9h  = 540 // 9 * 60
	breakRequired6h   = 30
	breakRequired9h   = 45

	// Work time severity thresholds
	workTimeStandard = 480 // 8 * 60 (info)
	workTimeWarning  = 540 // 9 * 60 (warning)
	workTimeMax      = 600 // 10 * 60 (error/block)

	// Rest period
	restPeriodHours = 11
)

// CalculateRequiredBreak returns the required break minutes per ArbZG Section 4.
//
// ArbZG Section 4:
//   - Work >6h: minimum 30 minutes break
//   - Work >9h: minimum 45 minutes break
//   - Work <=6h: no break required
func CalculateRequiredBreak(workedMinutes int) int {
	switch {
	case workedMinutes > breakThreshold9h:
		return breakRequired9h
	case workedMinutes > breakThreshold6h:
		return breakRequired6h
	default:
		return 0
	}
}

// CalculateAutoBreakDeduction returns the minutes to auto-deduct.
// This is the deficit between required break and manually clocked break.
// Formula: max(0, required - manual)
func CalculateAutoBreakDeduction(workedMinutes, manualBreakMinutes int) int {
	required := CalculateRequiredBreak(workedMinutes)
	deficit := required - manualBreakMinutes
	if deficit > 0 {
		return deficit
	}
	return 0
}

// CheckWorkTime validates total worked minutes against ArbZG thresholds.
//
// Severity levels:
//   - None: <8h (within normal limits)
//   - Info: >=8h (standard daily limit reached)
//   - Warning: >=9h (approaching absolute limit)
//   - Error: >10h (absolute maximum exceeded, must stop)
func CheckWorkTime(totalWorkedMinutes int) WorkTimeCheckResult {
	result := WorkTimeCheckResult{
		TotalWorkedMinutes: totalWorkedMinutes,
	}

	switch {
	case totalWorkedMinutes > workTimeMax:
		result.Severity = SeverityError
		result.Message = "ArbZG: Hoechstarbeitszeit von 10 Stunden ueberschritten. Arbeit muss beendet werden."
	case totalWorkedMinutes >= workTimeWarning:
		result.Severity = SeverityWarning
		result.Message = "ArbZG: 9 Stunden erreicht. Noch maximal 1 Stunde erlaubt."
	case totalWorkedMinutes >= workTimeStandard:
		result.Severity = SeverityInfo
		result.Message = "ArbZG: Regelarbeitszeit von 8 Stunden erreicht."
	default:
		result.Severity = SeverityNone
		result.Message = ""
	}

	return result
}

// CheckRestPeriod validates the 11-hour rest period between shifts per ArbZG Section 5.
// Returns (violation bool, actualHours float64).
// If previousEnd is nil (first day), no violation is possible.
func CheckRestPeriod(currentStart time.Time, previousEnd *time.Time) (bool, float64) {
	if previousEnd == nil {
		return false, 0
	}

	duration := currentStart.Sub(*previousEnd)
	hours := duration.Hours()

	return hours < float64(restPeriodHours), hours
}

// CalculateNetWorkMinutes computes net work minutes.
// Formula: total duration - manual breaks - auto breaks
func CalculateNetWorkMinutes(clockIn, clockOut time.Time, breakMinutes, autoBreakMinutes int) int {
	totalMinutes := int(clockOut.Sub(clockIn).Minutes())
	net := totalMinutes - breakMinutes - autoBreakMinutes
	if net < 0 {
		return 0
	}
	return net
}
