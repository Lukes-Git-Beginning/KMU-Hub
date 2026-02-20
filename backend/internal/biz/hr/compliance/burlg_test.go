package compliance

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func d(val string) decimal.Decimal {
	return decimal.RequireFromString(val)
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestCalculateLeaveBalance_FullTime_FullYear_20Days(t *testing.T) {
	// Full-time 5-day week, full year, minimum 20 days
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 6, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("20").Equal(result.TotalEntitlement), "expected 20, got %s", result.TotalEntitlement)
	assert.True(t, d("20").Equal(result.ProRataEntitlement), "expected 20, got %s", result.ProRataEntitlement)
	assert.True(t, d("20").Equal(result.Remaining), "expected 20, got %s", result.Remaining)
}

func TestCalculateLeaveBalance_FullTime_FullYear_25Days(t *testing.T) {
	// Full-time 5-day week, contractual 25 days
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 25,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 6, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("25").Equal(result.TotalEntitlement), "expected 25, got %s", result.TotalEntitlement)
	assert.True(t, d("25").Equal(result.ProRataEntitlement), "expected 25, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_PartTime_3Day_20Contract(t *testing.T) {
	// Part-time 3-day week, 20 days contract = 12 days (20 * 3/5)
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   3,
		AnnualEntitlementDays: 20,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 6, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("12").Equal(result.TotalEntitlement), "expected 12, got %s", result.TotalEntitlement)
	assert.True(t, d("12").Equal(result.ProRataEntitlement), "expected 12, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_PartTime_4Day_25Contract(t *testing.T) {
	// Part-time 4-day week, 25 days contract = 20 days (25 * 4/5)
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   4,
		AnnualEntitlementDays: 25,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 6, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("20").Equal(result.TotalEntitlement), "expected 20, got %s", result.TotalEntitlement)
	assert.True(t, d("20").Equal(result.ProRataEntitlement), "expected 20, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_MidYearStart_July(t *testing.T) {
	// Mid-year start (July 1), 5-day week, 20 days = 10 days pro-rata (6/12 * 20)
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2025, 7, 1),
		CalculationDate:       date(2025, 10, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("20").Equal(result.TotalEntitlement), "expected total 20, got %s", result.TotalEntitlement)
	assert.True(t, d("10").Equal(result.ProRataEntitlement), "expected pro-rata 10, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_MidYearStart_March15(t *testing.T) {
	// Mid-year start (March 15), full-time, 20 days
	// Remaining full months: April through December = 9 months
	// Pro-rata: 9/12 * 20 = 15.0
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2025, 3, 15),
		CalculationDate:       date(2025, 10, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("15").Equal(result.ProRataEntitlement), "expected pro-rata 15, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_StartOnFirstOfMonth(t *testing.T) {
	// Started on 1st of month gets that full month counted
	// Start September 1 -> Sep, Oct, Nov, Dec = 4 months
	// Pro-rata: 4/12 * 20 = 6.7 (rounded to 1 decimal)
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2025, 9, 1),
		CalculationDate:       date(2025, 11, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("6.7").Equal(result.ProRataEntitlement), "expected pro-rata 6.7, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_HalfDayUsage(t *testing.T) {
	// 25 entitlement, 10.5 used = 14.5 remaining
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 25,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 6, 15),
		UsedDays:              d("10.5"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("25").Equal(result.TotalEntitlement), "expected 25, got %s", result.TotalEntitlement)
	assert.True(t, d("14.5").Equal(result.Remaining), "expected 14.5, got %s", result.Remaining)
}

func TestCalculateLeaveBalance_CarryoverBeforeMarch31(t *testing.T) {
	// 5 days carried over, before March 31 = included in remaining
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 2, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("5"),
	}

	result := CalculateLeaveBalance(input)

	assert.False(t, result.CarryoverExpired, "carryover should not be expired before March 31")
	assert.True(t, d("5").Equal(result.CarriedOver), "expected carried over 5, got %s", result.CarriedOver)
	assert.True(t, d("25").Equal(result.Remaining), "expected 25 (20+5), got %s", result.Remaining)
}

func TestCalculateLeaveBalance_CarryoverAfterMarch31(t *testing.T) {
	// 5 days carried over, after March 31 = NOT included
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 4, 1),
		UsedDays:              d("0"),
		CarriedOverDays:       d("5"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, result.CarryoverExpired, "carryover should be expired after March 31")
	assert.True(t, d("0").Equal(result.CarriedOver), "expected carried over 0, got %s", result.CarriedOver)
	assert.True(t, d("20").Equal(result.Remaining), "expected 20 (carryover expired), got %s", result.Remaining)
}

func TestCalculateLeaveBalance_FirstYearProRata(t *testing.T) {
	// First year employment (within 6 months): pro-rata 1/12 per full month worked
	// Started Oct 1, calculation Dec 15 = 3 full months (Oct, Nov, Dec)
	// 3/12 * 20 = 5.0
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2025, 10, 1),
		CalculationDate:       date(2025, 12, 15),
		UsedDays:              d("0"),
		CarriedOverDays:       d("0"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("5").Equal(result.ProRataEntitlement), "expected pro-rata 5, got %s", result.ProRataEntitlement)
}

func TestCalculateLeaveBalance_ZeroUsedDays(t *testing.T) {
	// Zero used days: remaining equals full entitlement + carryover
	input := LeaveEntitlementInput{
		ContractDaysPerWeek:   5,
		AnnualEntitlementDays: 20,
		StartDate:             date(2024, 1, 1),
		CalculationDate:       date(2025, 2, 1),
		UsedDays:              d("0"),
		CarriedOverDays:       d("3"),
	}

	result := CalculateLeaveBalance(input)

	assert.True(t, d("23").Equal(result.Remaining), "expected 23 (20+3), got %s", result.Remaining)
	assert.True(t, d("0").Equal(result.Used), "expected 0 used, got %s", result.Used)
}
