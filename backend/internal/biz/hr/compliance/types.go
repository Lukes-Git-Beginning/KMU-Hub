// Package compliance implements pure functions for German labor law compliance.
// BUrlG (Bundesurlaubsgesetz) - leave entitlement calculation
// ArbZG (Arbeitszeitgesetz) - work time validation
//
// All functions are pure: no I/O, no database, no context.
// This enables exhaustive unit testing without mocking.
package compliance

import (
	"time"

	"github.com/shopspring/decimal"
)

// LeaveEntitlementInput contains all data needed to calculate leave balance.
type LeaveEntitlementInput struct {
	ContractDaysPerWeek   int             // Work days per week (1-7, standard: 5)
	AnnualEntitlementDays int             // Contractual annual entitlement (min 20 for 5-day week)
	StartDate             time.Time       // Employment start date
	CalculationDate       time.Time       // Date to calculate balance for
	UsedDays              decimal.Decimal // Days already taken (supports 0.5 increments)
	CarriedOverDays       decimal.Decimal // Days carried over from previous year
}

// LeaveBalance is the result of leave entitlement calculation.
type LeaveBalance struct {
	TotalEntitlement   decimal.Decimal // Full annual entitlement (adjusted for part-time)
	ProRataEntitlement decimal.Decimal // Adjusted for mid-year start
	CarriedOver        decimal.Decimal // From previous year (0 if expired)
	Used               decimal.Decimal // Already taken
	Remaining          decimal.Decimal // Available to take
	CarryoverDeadline  time.Time       // March 31 of calculation year
	CarryoverExpired   bool            // Whether carryover has expired
}

// WorkTimeEntry represents a single work segment for ArbZG checking.
type WorkTimeEntry struct {
	StartTime time.Time
	EndTime   time.Time
}

// WorkTimeCheckResult contains ArbZG compliance check results.
type WorkTimeCheckResult struct {
	TotalWorkedMinutes   int
	BreakMinutesTaken    int
	BreakMinutesRequired int
	BreakDeficit         int             // >0 means auto-deduction needed
	Severity             WarningSeverity // Info, Warning, Error
	Message              string
	RestViolation        bool    // True if <11h since last shift end
	RestHoursActual      float64 // Actual rest hours between shifts
}

// WarningSeverity indicates the severity of an ArbZG compliance check.
type WarningSeverity string

const (
	SeverityNone    WarningSeverity = "none"
	SeverityInfo    WarningSeverity = "info"    // At 8h (standard limit)
	SeverityWarning WarningSeverity = "warning" // At 9h (approaching limit)
	SeverityError   WarningSeverity = "error"   // At 10h (absolute maximum)
)
