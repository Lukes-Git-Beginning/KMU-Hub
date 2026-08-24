// Package datev provides DATEV Buchungsstapel CSV export in EXTF format.
// Implements SKR03 account mapping for German standard chart of accounts.
package datev

import "github.com/shopspring/decimal"

// SKR03 revenue accounts
const (
	// AccountRevenue19 is the revenue account for 19% USt (Erloese 19% USt).
	AccountRevenue19 = "8400"
	// AccountRevenue7 is the revenue account for 7% USt (Erloese 7% USt).
	AccountRevenue7 = "8300"
	// AccountRevenueEU is the revenue account for 0% USt EU (steuerfreie innergemeinschaftliche Lieferungen).
	AccountRevenueEU = "8125"
	// AccountRevenueKleinunternehmer is the revenue account for 0% USt Kleinunternehmer (steuerfreie Umsaetze Inland).
	AccountRevenueKleinunternehmer = "8195"
	// AccountRevenueOther is the generic revenue account (Erloese) used for any
	// non-zero rate that is not a German standard rate — e.g. 7.5% or a historic
	// 16%. Booking such a line on 8300/8400 would make DATEV auto-compute the
	// wrong VAT; 8200 without a BU-Schluessel forces the accountant to assign it.
	AccountRevenueOther = "8200"
)

// BU-Schluessel (booking keys) for DATEV automatic tax calculation
const (
	// BUSchluessel19 is the BU-Schluessel for 19% USt.
	BUSchluessel19 = "3"
	// BUSchluessel7 is the BU-Schluessel for 7% USt.
	BUSchluessel7 = "2"
	// BUSchluesselNone is empty for 0% USt and for unmapped rates (no automatic tax).
	BUSchluesselNone = ""
)

// DebitorAccountBase is the base for auto-generated debitor accounts.
// Customer debitor account = DebitorAccountBase + customer index.
const DebitorAccountBase = 10000

var (
	rate19 = decimal.NewFromInt(19)
	rate7  = decimal.NewFromInt(7)
)

// RateKey renders a tax rate as its canonical export key: "19", "7", "0", "7.5".
// Trailing zeros are trimmed so that 19 and 19.00 share one key, but fractional
// rates keep their fraction — truncating to the integer part would book 7.5% as
// 7% (wrong revenue account, wrong BU-Schluessel, VAT that no longer reconciles).
func RateKey(rate decimal.Decimal) string {
	return rate.String()
}

// RevenueAccountForRateAndMode returns the SKR03 revenue account for an exact tax
// rate, considering tax mode. Rates other than 19%, 7% and 0% have no SKR03
// standard account and map to AccountRevenueOther.
func RevenueAccountForRateAndMode(rate decimal.Decimal, taxMode string) string {
	switch {
	case rate.Equal(rate19):
		return AccountRevenue19
	case rate.Equal(rate7):
		return AccountRevenue7
	case rate.IsZero():
		// 0% rate: distinguish between EU reverse charge and Kleinunternehmer
		if taxMode == "reverse_charge" {
			return AccountRevenueEU
		}
		return AccountRevenueKleinunternehmer
	default:
		return AccountRevenueOther
	}
}

// BUSchluesselForRate returns the DATEV BU-Schluessel for an exact tax rate.
// Only the German standard rates carry an automatic-tax key; anything else
// stays empty so DATEV does not compute VAT the document does not carry.
func BUSchluesselForRate(rate decimal.Decimal) string {
	switch {
	case rate.Equal(rate19):
		return BUSchluessel19
	case rate.Equal(rate7):
		return BUSchluessel7
	default:
		return BUSchluesselNone
	}
}
