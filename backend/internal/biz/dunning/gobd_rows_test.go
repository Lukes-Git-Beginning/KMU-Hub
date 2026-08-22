package dunning

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/models"
)

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

func lineItemsJSON(t *testing.T, items []models.LineItem) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}
	return raw
}

var goBDDate = time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

// A 7.5% line must not be booked on the 7% revenue account with BU key 2 —
// DATEV would auto-compute 7% VAT on a line that carries 7.5%, and the posting
// stops reconciling at the tax advisor.
func TestBuildGoBDRows_SeparatesFractionalRateFromWholeRate(t *testing.T) {
	items := []models.LineItem{
		{Description: "reduced", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: mustDec(t, "7")},
		{Description: "fractional", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(200), TaxRate: mustDec(t, "7.5")},
	}
	rows := BuildGoBDRows("RE-1", goBDDate, "Kunde", "sent", models.TaxModeStandard, lineItemsJSON(t, items), decimal.NewFromInt(300), 1)

	if len(rows) != 2 {
		t.Fatalf("want 2 posting rows (one per rate), got %d: %+v", len(rows), rows)
	}
	if rows[0].TaxRate != "7" || rows[0].Account != datev.AccountRevenue7 || rows[0].TaxKey != datev.BUSchluessel7 {
		t.Errorf("7%% row = rate %q account %q key %q, want 7/%s/%s",
			rows[0].TaxRate, rows[0].Account, rows[0].TaxKey, datev.AccountRevenue7, datev.BUSchluessel7)
	}
	if rows[1].TaxRate != "7.5" || rows[1].Account != datev.AccountRevenueOther || rows[1].TaxKey != datev.BUSchluesselNone {
		t.Errorf("7.5%% row = rate %q account %q key %q, want 7.5/%s/empty",
			rows[1].TaxRate, rows[1].Account, rows[1].TaxKey, datev.AccountRevenueOther)
	}
	if rows[0].NetAmount != "100.00" || rows[1].NetAmount != "200.00" {
		t.Errorf("nets = %q / %q, want 100.00 / 200.00", rows[0].NetAmount, rows[1].NetAmount)
	}
	if rows[1].TaxAmount != "15.00" {
		t.Errorf("7.5%% of 200 = %q, want 15.00", rows[1].TaxAmount)
	}
}

// Two lines at 4.995 must be booked as 5.00 each. Summing unrounded and cutting
// at the end would report 9.99 and leave a cent unaccounted for.
func TestBuildGoBDRows_RoundsLineNetBeforeSumming(t *testing.T) {
	line := models.LineItem{
		Quantity:  mustDec(t, "1.5"),
		UnitPrice: mustDec(t, "3.33"),
		TaxRate:   decimal.NewFromInt(19),
	}
	rows := BuildGoBDRows("RE-2", goBDDate, "Kunde", "sent", models.TaxModeStandard,
		lineItemsJSON(t, []models.LineItem{line, line}), decimal.NewFromInt(10), 1)

	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].NetAmount != "10.00" {
		t.Errorf("net = %q, want 10.00 (2 x round(4.995))", rows[0].NetAmount)
	}
	if rows[0].TaxAmount != "1.90" {
		t.Errorf("tax = %q, want 1.90", rows[0].TaxAmount)
	}
	if rows[0].GrossTotal != "11.90" {
		t.Errorf("gross = %q, want 11.90", rows[0].GrossTotal)
	}
}

// A rate written as 19.00 is the same posting group as 19.
func TestBuildGoBDRows_NormalisesTrailingZeroRates(t *testing.T) {
	items := []models.LineItem{
		{Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), TaxRate: mustDec(t, "19")},
		{Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), TaxRate: mustDec(t, "19.00")},
	}
	rows := BuildGoBDRows("RE-3", goBDDate, "Kunde", "sent", models.TaxModeStandard, lineItemsJSON(t, items), decimal.NewFromInt(100), 1)

	if len(rows) != 1 {
		t.Fatalf("19 and 19.00 must share one group, got %d rows", len(rows))
	}
	if rows[0].TaxRate != "19" || rows[0].NetAmount != "100.00" {
		t.Errorf("row = rate %q net %q, want 19 / 100.00", rows[0].TaxRate, rows[0].NetAmount)
	}
}

func TestBuildGoBDRows_CreditNoteNegatesAmounts(t *testing.T) {
	items := []models.LineItem{
		{Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: decimal.NewFromInt(19)},
	}
	rows := BuildGoBDRows("GS-1", goBDDate, "Kunde", "credit_note", models.TaxModeStandard, lineItemsJSON(t, items), decimal.NewFromInt(100), -1)

	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].NetAmount != "-100.00" || rows[0].TaxAmount != "-19.00" || rows[0].GrossTotal != "-119.00" {
		t.Errorf("credit note row = %q / %q / %q, want -100.00 / -19.00 / -119.00",
			rows[0].NetAmount, rows[0].TaxAmount, rows[0].GrossTotal)
	}
}

func TestBuildGoBDRows_ExemptModesEmitOneZeroRatedRow(t *testing.T) {
	items := []models.LineItem{
		{Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: decimal.NewFromInt(19)},
	}
	cases := []struct {
		mode        string
		wantAccount string
	}{
		{models.TaxModeReverseCharge, datev.AccountRevenueEU},
		{models.TaxModeKleinunternehmer, datev.AccountRevenueKleinunternehmer},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			rows := BuildGoBDRows("RE-4", goBDDate, "Kunde", "sent", tc.mode, lineItemsJSON(t, items), mustDec(t, "250.50"), 1)
			if len(rows) != 1 {
				t.Fatalf("want 1 exempt row, got %d", len(rows))
			}
			r := rows[0]
			if r.Account != tc.wantAccount || r.TaxRate != "0" || r.TaxKey != datev.BUSchluesselNone {
				t.Errorf("row = account %q rate %q key %q, want %s / 0 / empty", r.Account, r.TaxRate, r.TaxKey, tc.wantAccount)
			}
			if r.NetAmount != "250.50" || r.TaxAmount != "0.00" || r.GrossTotal != "250.50" {
				t.Errorf("amounts = %q / %q / %q, want 250.50 / 0.00 / 250.50", r.NetAmount, r.TaxAmount, r.GrossTotal)
			}
		})
	}
}

func TestBuildGoBDRows_UnparseableLineItemsYieldNoRows(t *testing.T) {
	rows := BuildGoBDRows("RE-5", goBDDate, "Kunde", "sent", models.TaxModeStandard, json.RawMessage(`{"not":"an array"}`), decimal.NewFromInt(100), 1)
	if len(rows) != 0 {
		t.Fatalf("want no rows for unparseable line items, got %d", len(rows))
	}
}

func TestBuildGoBDRows_CarriesDocumentMetadata(t *testing.T) {
	items := []models.LineItem{
		{Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(10), TaxRate: decimal.NewFromInt(19)},
	}
	rows := BuildGoBDRows("RE-2026-0007", goBDDate, "Müller GmbH", "paid", models.TaxModeStandard, lineItemsJSON(t, items), decimal.NewFromInt(10), 1)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.InvoiceNumber != "RE-2026-0007" || r.InvoiceDate != "2026-03-14" || r.CustomerName != "Müller GmbH" {
		t.Errorf("header = %q / %q / %q", r.InvoiceNumber, r.InvoiceDate, r.CustomerName)
	}
	if r.Status != "paid" || r.TaxMode != models.TaxModeStandard {
		t.Errorf("status/mode = %q / %q", r.Status, r.TaxMode)
	}
	if r.BookingText != "Rechnung RE-2026-0007 Müller GmbH" {
		t.Errorf("booking text = %q", r.BookingText)
	}
}
