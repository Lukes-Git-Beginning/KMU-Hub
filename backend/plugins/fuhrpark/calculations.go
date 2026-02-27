package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// FuelEntry represents a single fuel fill-up record
type FuelEntry struct {
	Date          string  `json:"date"`
	Liters        float64 `json:"liters"`
	PricePerLiter float64 `json:"price_per_liter"`
	Total         float64 `json:"total"`
	Mileage       float64 `json:"mileage"`
	FuelType      string  `json:"fuel_type"` // diesel, benzin, electric
}

// FuelSummary provides aggregated fuel cost data
type FuelSummary struct {
	TotalLiters     float64 `json:"total_liters"`
	TotalCost       float64 `json:"total_cost"`
	AvgPricePerLiter float64 `json:"avg_price_per_liter"`
	AvgConsumption  float64 `json:"avg_consumption_per_100km"`
	TotalKM         float64 `json:"total_km"`
	CostPerKM       float64 `json:"cost_per_km"`
	EntryCount      int     `json:"entry_count"`
}

// CalculateFuelSummary calculates fuel statistics from a list of entries
func CalculateFuelSummary(entries []FuelEntry) *FuelSummary {
	if len(entries) == 0 {
		return &FuelSummary{}
	}

	summary := &FuelSummary{
		EntryCount: len(entries),
	}

	var minMileage, maxMileage float64
	minMileage = entries[0].Mileage
	maxMileage = entries[0].Mileage

	for _, entry := range entries {
		summary.TotalLiters += entry.Liters
		summary.TotalCost += entry.Total
		if entry.Mileage < minMileage {
			minMileage = entry.Mileage
		}
		if entry.Mileage > maxMileage {
			maxMileage = entry.Mileage
		}
	}

	summary.TotalKM = maxMileage - minMileage

	if summary.TotalLiters > 0 {
		summary.AvgPricePerLiter = summary.TotalCost / summary.TotalLiters
	}
	if summary.TotalKM > 0 {
		summary.AvgConsumption = (summary.TotalLiters / summary.TotalKM) * 100
		summary.CostPerKM = summary.TotalCost / summary.TotalKM
	}

	return summary
}

// MaintenanceRecord represents a maintenance event
type MaintenanceRecord struct {
	Date        string  `json:"date"`
	Type        string  `json:"type"` // oil_change, tire_change, inspection, repair, other
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	Mileage     float64 `json:"mileage"`
	Workshop    string  `json:"workshop"`
}

// MaintenancePlan represents a scheduled maintenance item
type MaintenancePlan struct {
	Type        string  `json:"type"`
	IntervalKM  float64 `json:"interval_km"`
	IntervalDays int    `json:"interval_days"`
	LastDoneKM  float64 `json:"last_done_km"`
	LastDoneDate string `json:"last_done_date"`
	NextDueKM   float64 `json:"next_due_km"`
	NextDueDate string  `json:"next_due_date"`
}

// CalculateNextMaintenance calculates when the next maintenance is due
func CalculateNextMaintenance(lastMileage, currentMileage, intervalKM float64, lastDate string, intervalDays int) *MaintenancePlan {
	plan := &MaintenancePlan{
		IntervalKM:   intervalKM,
		IntervalDays: intervalDays,
		LastDoneKM:   lastMileage,
		LastDoneDate: lastDate,
		NextDueKM:    lastMileage + intervalKM,
	}

	if lastDate != "" && intervalDays > 0 {
		t, err := time.Parse("2006-01-02", lastDate)
		if err == nil {
			nextDate := t.AddDate(0, 0, intervalDays)
			plan.NextDueDate = nextDate.Format("2006-01-02")
		}
	}

	return plan
}

// InspectionCheck represents a TÜV/HU inspection status
type InspectionCheck struct {
	LicensePlate   string `json:"license_plate"`
	NextInspection string `json:"next_inspection"`
	DaysRemaining  int    `json:"days_remaining"`
	Status         string `json:"status"` // ok, warning, overdue
}

// CheckInspectionStatus calculates the inspection status for a vehicle
func CheckInspectionStatus(licensePlate, nextInspection string, warningDays int) *InspectionCheck {
	check := &InspectionCheck{
		LicensePlate:   licensePlate,
		NextInspection: nextInspection,
	}

	if nextInspection == "" {
		check.Status = "ok"
		return check
	}

	inspDate, err := time.Parse("2006-01-02", nextInspection)
	if err != nil {
		check.Status = "ok"
		return check
	}

	daysRemaining := int(time.Until(inspDate).Hours() / 24)
	check.DaysRemaining = daysRemaining

	switch {
	case daysRemaining < 0:
		check.Status = "overdue"
	case daysRemaining <= warningDays:
		check.Status = "warning"
	default:
		check.Status = "ok"
	}

	return check
}

// SerializeJSON marshals a value to JSON bytes (for KV store storage)
func SerializeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// FormatCurrency formats a number as a currency string
func FormatCurrency(amount float64, currency string) string {
	switch currency {
	case "CHF":
		return fmt.Sprintf("CHF %.2f", amount)
	default:
		return fmt.Sprintf("%.2f €", amount)
	}
}
