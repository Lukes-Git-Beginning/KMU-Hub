package tax

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func TestCalculate_Standard19_SingleItem(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("100.00")), "subtotal should be 100.00, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.Equal(d("19.00")), "total tax should be 19.00, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("119.00")), "gross total should be 119.00, got %s", result.GrossTotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("19.00")), "tax at 19%% should be 19.00, got %s", result.TaxByRate["19"])
}

func TestCalculate_Standard7_ReducedRate(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("7.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("100.00")), "subtotal should be 100.00, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.Equal(d("7.00")), "total tax should be 7.00, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("107.00")), "gross total should be 107.00, got %s", result.GrossTotal)
	assert.True(t, result.TaxByRate["7"].Equal(d("7.00")), "tax at 7%% should be 7.00, got %s", result.TaxByRate["7"])
}

func TestCalculate_MixedRates(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("200.00"), TaxRate: d("19.00")},
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("7.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("300.00")), "subtotal should be 300.00, got %s", result.Subtotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("38.00")), "tax at 19%% should be 38.00, got %s", result.TaxByRate["19"])
	assert.True(t, result.TaxByRate["7"].Equal(d("7.00")), "tax at 7%% should be 7.00, got %s", result.TaxByRate["7"])
	assert.True(t, result.TotalTax.Equal(d("45.00")), "total tax should be 45.00, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("345.00")), "gross total should be 345.00, got %s", result.GrossTotal)
}

func TestCalculate_ReverseCharge(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("19.00")},
		{Quantity: d("2"), UnitPrice: d("50.00"), TaxRate: d("7.00")},
	}

	result := Calculate(items, ModeReverseCharge)

	assert.True(t, result.Subtotal.Equal(d("200.00")), "subtotal should be 200.00, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.Equal(d("0")), "total tax should be 0 for Reverse Charge, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("200.00")), "gross total should equal subtotal for Reverse Charge, got %s", result.GrossTotal)
	assert.Empty(t, result.TaxByRate, "TaxByRate should be empty for Reverse Charge")
}

func TestCalculate_Kleinunternehmer(t *testing.T) {
	items := []LineItem{
		{Quantity: d("3"), UnitPrice: d("100.00"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeKleinunternehmer)

	assert.True(t, result.Subtotal.Equal(d("300.00")), "subtotal should be 300.00, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.Equal(d("0")), "total tax should be 0 for Kleinunternehmer, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("300.00")), "gross total should equal subtotal for Kleinunternehmer, got %s", result.GrossTotal)
	assert.Empty(t, result.TaxByRate, "TaxByRate should be empty for Kleinunternehmer")
}

func TestCalculate_PerLineRounding(t *testing.T) {
	// 19% of 123.45 = 23.4555, should round to 23.46 per line
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("123.45"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("123.45")), "subtotal should be 123.45, got %s", result.Subtotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("23.46")), "tax at 19%% should be 23.46 (rounded), got %s", result.TaxByRate["19"])
	assert.True(t, result.TotalTax.Equal(d("23.46")), "total tax should be 23.46, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("146.91")), "gross total should be 146.91, got %s", result.GrossTotal)
}

func TestCalculate_MultipleQuantities(t *testing.T) {
	// 3 x 49.99 = 149.97 net
	// 19% of 149.97 = 28.4943, rounds to 28.49
	items := []LineItem{
		{Quantity: d("3"), UnitPrice: d("49.99"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("149.97")), "subtotal should be 149.97, got %s", result.Subtotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("28.49")), "tax at 19%% should be 28.49, got %s", result.TaxByRate["19"])
	assert.True(t, result.TotalTax.Equal(d("28.49")), "total tax should be 28.49, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("178.46")), "gross total should be 178.46, got %s", result.GrossTotal)
}

func TestCalculate_EmptyLineItems(t *testing.T) {
	result := Calculate(nil, ModeStandard)

	assert.True(t, result.Subtotal.IsZero(), "subtotal should be zero for empty items, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.IsZero(), "total tax should be zero for empty items, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.IsZero(), "gross total should be zero for empty items, got %s", result.GrossTotal)
	assert.Empty(t, result.TaxByRate, "TaxByRate should be empty for empty items")
}

func TestCalculate_ZeroQuantityOrPrice(t *testing.T) {
	items := []LineItem{
		{Quantity: d("0"), UnitPrice: d("100.00"), TaxRate: d("19.00")},
		{Quantity: d("5"), UnitPrice: d("0"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.IsZero(), "subtotal should be zero, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.IsZero(), "total tax should be zero, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.IsZero(), "gross total should be zero, got %s", result.GrossTotal)
}

func TestCalculate_LargeValues(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("999999.99"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	// 19% of 999999.99 = 189999.9981, rounds to 190000.00
	assert.True(t, result.Subtotal.Equal(d("999999.99")), "subtotal should be 999999.99, got %s", result.Subtotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("190000.00")), "tax at 19%% should be 190000.00, got %s", result.TaxByRate["19"])
	assert.True(t, result.TotalTax.Equal(d("190000.00")), "total tax should be 190000.00, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("1189999.99")), "gross total should be 1189999.99, got %s", result.GrossTotal)
}

func TestCalculate_MultipleItemsSameRate_Aggregation(t *testing.T) {
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("50.00"), TaxRate: d("19.00")},
		{Quantity: d("1"), UnitPrice: d("30.00"), TaxRate: d("19.00")},
		{Quantity: d("1"), UnitPrice: d("20.00"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("100.00")), "subtotal should be 100.00, got %s", result.Subtotal)
	// Per-line: 9.50 + 5.70 + 3.80 = 19.00
	assert.True(t, result.TaxByRate["19"].Equal(d("19.00")), "tax at 19%% should be 19.00, got %s", result.TaxByRate["19"])
	assert.True(t, result.TotalTax.Equal(d("19.00")), "total tax should be 19.00, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("119.00")), "gross total should be 119.00, got %s", result.GrossTotal)
}

func TestCalculate_FractionalRate_DoesNotCollideWithWholeRate(t *testing.T) {
	// 7.5% and 7% must aggregate into two separate groups, not one "7" bucket.
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("7.50")},
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("7.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.Len(t, result.TaxByRate, 2, "7.5%% and 7%% must be distinct groups, got %v", result.TaxByRate)
	assert.True(t, result.TaxByRate["7.5"].Equal(d("7.50")), "tax at 7.5%% should be 7.50, got %s", result.TaxByRate["7.5"])
	assert.True(t, result.TaxByRate["7"].Equal(d("7.00")), "tax at 7%% should be 7.00, got %s", result.TaxByRate["7"])
	assert.True(t, result.TotalTax.Equal(d("14.50")), "total tax should be 14.50, got %s", result.TotalTax)
}

func TestCalculate_FractionalQuantity_RoundsLineNetBeforeSumming(t *testing.T) {
	// 1.5 hours at 33.333 EUR/hour = 49.9995 unrounded. The line net must be
	// rounded to 50.00 before it enters the subtotal -- an unrounded sum would
	// leave the stored net at 49.9995, three decimal places deep, which then
	// disagrees with the e-invoice's BT-106 (also rounded per line, see
	// einvoice.buildLinesAndTaxGroups) by fractions of a cent.
	items := []LineItem{
		{Quantity: d("1.5"), UnitPrice: d("33.333"), TaxRate: d("19.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.True(t, result.Subtotal.Equal(d("50.00")), "subtotal should be rounded to 50.00, got %s", result.Subtotal)
	assert.True(t, result.TaxByRate["19"].Equal(d("9.50")), "tax at 19%% should be 9.50, got %s", result.TaxByRate["19"])
	assert.True(t, result.TotalTax.Equal(d("9.50")), "total tax should be 9.50, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(d("59.50")), "gross total should be 59.50, got %s", result.GrossTotal)
}

func TestCalculate_ZeroRateLineInStandardMode_CreatesNoRateGroup(t *testing.T) {
	// A 0% line alongside taxable lines must not create a "0" TaxByRate group --
	// that would render and export as VAT category S at 0%, which EN 16931 rejects.
	items := []LineItem{
		{Quantity: d("1"), UnitPrice: d("100.00"), TaxRate: d("19.00")},
		{Quantity: d("1"), UnitPrice: d("50.00"), TaxRate: d("0.00")},
	}

	result := Calculate(items, ModeStandard)

	assert.Len(t, result.TaxByRate, 1, "the 0%% line must not create its own group, got %v", result.TaxByRate)
	_, hasZeroGroup := result.TaxByRate["0"]
	assert.False(t, hasZeroGroup, "TaxByRate must not carry a \"0\" key")
	assert.True(t, result.Subtotal.Equal(d("150.00")), "subtotal should still include the 0%% line, got %s", result.Subtotal)
	assert.True(t, result.TotalTax.Equal(d("19.00")), "total tax should be unaffected, got %s", result.TotalTax)
}

func TestRequiresReverseChargeNote(t *testing.T) {
	assert.True(t, RequiresReverseChargeNote(ModeReverseCharge))
	assert.False(t, RequiresReverseChargeNote(ModeStandard))
	assert.False(t, RequiresReverseChargeNote(ModeKleinunternehmer))
}

func TestRequiresKleinunternehmerNote(t *testing.T) {
	assert.True(t, RequiresKleinunternehmerNote(ModeKleinunternehmer))
	assert.False(t, RequiresKleinunternehmerNote(ModeStandard))
	assert.False(t, RequiresKleinunternehmerNote(ModeReverseCharge))
}

func TestStandardRate(t *testing.T) {
	assert.True(t, StandardRate().Equal(d("19.00")), "standard rate should be 19.00")
}

func TestReducedRate(t *testing.T) {
	assert.True(t, ReducedRate().Equal(d("7.00")), "reduced rate should be 7.00")
}
