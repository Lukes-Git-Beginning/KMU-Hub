package server

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestSecondsToHours(t *testing.T) {
	ninety := 5400
	if got := secondsToHours(nil); got != 0 {
		t.Errorf("nil duration = %v, want 0", got)
	}
	if got := secondsToHours(&ninety); got != 1.5 {
		t.Errorf("5400s = %v, want 1.5", got)
	}
}

// 100 one-minute entries: the rounded per-row sum is 2.00 h, the exact total
// (what CreateInvoiceFromTimeEntries bills) is 1.67 h. The billable rows must
// carry the exact value so a client summing them lands on the invoice figure.
func TestBillableTimeEntryToProto_RowsSumToInvoiceFigure(t *testing.T) {
	sec := 60
	var sum float64
	for i := 0; i < 100; i++ {
		p := billableTimeEntryToProto(&models.BillableTimeEntry{TimeEntry: models.TimeEntry{
			ID:              uuid.New(),
			StartedAt:       time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
			DurationSeconds: &sec,
		}})
		sum += p.GetHours()
	}
	if got := math.Round(sum*100) / 100; got != 1.67 {
		t.Errorf("rounded sum of billable rows = %v, want 1.67", got)
	}
}

func TestBillableTimeEntryToProto_RunningEntryHasZeroHours(t *testing.T) {
	p := billableTimeEntryToProto(&models.BillableTimeEntry{TimeEntry: models.TimeEntry{
		ID:        uuid.New(),
		StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}})
	if p.GetHours() != 0 {
		t.Errorf("hours = %v, want 0", p.GetHours())
	}
}

// The project rows stay rounded to 2 decimals: the project dialog invoices one
// line per row with exactly that quantity.
func TestProjectTimeEntryToProto_RoundsToTwoDecimals(t *testing.T) {
	sec := 100 // 0.02777... h
	p := projectTimeEntryToProto(&models.ProjectTimeEntry{TimeEntry: models.TimeEntry{
		ID:              uuid.New(),
		StartedAt:       time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		DurationSeconds: &sec,
	}})
	if p.GetHours() != 0.03 {
		t.Errorf("hours = %v, want 0.03", p.GetHours())
	}
}
