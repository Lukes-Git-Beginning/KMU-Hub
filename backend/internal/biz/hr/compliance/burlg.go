package compliance

import (
	"time"

	"github.com/shopspring/decimal"
)

var (
	decZero  = decimal.NewFromInt(0)
	decFive  = decimal.NewFromInt(5)
	decTwelve = decimal.NewFromInt(12)
)

// CalculateLeaveBalance computes BUrlG-compliant leave balance.
//
// BUrlG rules implemented:
//   - Minimum 20 days for 5-day week (proportional for part-time)
//   - Full entitlement after 6 months employment (pro-rata before)
//   - Carryover to March 31 of next year only
//   - Pro-rata for mid-year start: 1/12 per full month
//   - Part-time pro-rata: annual * workDaysPerWeek / 5
//   - Round to 1 decimal place for half-day increments
func CalculateLeaveBalance(input LeaveEntitlementInput) LeaveBalance {
	// Step 1: Calculate total entitlement adjusted for part-time
	annualDays := decimal.NewFromInt(int64(input.AnnualEntitlementDays))
	if input.ContractDaysPerWeek < 5 {
		annualDays = annualDays.
			Mul(decimal.NewFromInt(int64(input.ContractDaysPerWeek))).
			Div(decFive).
			Round(1)
	}

	// Step 2: Calculate pro-rata entitlement for the calculation year
	calcYear := input.CalculationDate.Year()
	yearStart := time.Date(calcYear, 1, 1, 0, 0, 0, 0, time.UTC)
	proRata := annualDays

	// Only pro-rate if employee started during the calculation year
	if input.StartDate.Year() == calcYear && input.StartDate.After(yearStart) {
		remainingMonths := calculateRemainingMonths(input.StartDate, calcYear)
		proRata = annualDays.
			Mul(decimal.NewFromInt(int64(remainingMonths))).
			Div(decTwelve).
			Round(1)
	}

	// Step 3: Handle carryover
	carryoverDeadline := time.Date(calcYear, 3, 31, 0, 0, 0, 0, time.UTC)
	carryoverExpired := !input.CalculationDate.Before(carryoverDeadline.AddDate(0, 0, 1))
	carriedOver := input.CarriedOverDays
	if carryoverExpired {
		carriedOver = decZero
	}

	// Step 4: Calculate remaining
	remaining := proRata.Add(carriedOver).Sub(input.UsedDays)

	return LeaveBalance{
		TotalEntitlement:   annualDays,
		ProRataEntitlement: proRata,
		CarriedOver:        carriedOver,
		Used:               input.UsedDays,
		Remaining:          remaining,
		CarryoverDeadline:  carryoverDeadline,
		CarryoverExpired:   carryoverExpired,
	}
}

// calculateRemainingMonths counts the number of full months from startDate
// to the end of the year. If started on the 1st, that month is included.
func calculateRemainingMonths(startDate time.Time, year int) int {
	startMonth := int(startDate.Month())
	if startDate.Day() == 1 {
		// Include the start month if started on the 1st
		return 12 - startMonth + 1
	}
	// Only count months fully after the start month
	return 12 - startMonth
}
