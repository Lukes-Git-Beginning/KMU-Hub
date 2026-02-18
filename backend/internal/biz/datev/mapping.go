// Package datev provides DATEV Buchungsstapel CSV export in EXTF format.
// Implements SKR03 account mapping for German standard chart of accounts.
package datev

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
)

// BU-Schluessel (booking keys) for DATEV automatic tax calculation
const (
	// BUSchluessel19 is the BU-Schluessel for 19% USt.
	BUSchluessel19 = "3"
	// BUSchluessel7 is the BU-Schluessel for 7% USt.
	BUSchluessel7 = "2"
	// BUSchluesselNone is empty for 0% USt (no automatic tax).
	BUSchluesselNone = ""
)

// DebitorAccountBase is the base for auto-generated debitor accounts.
// Customer debitor account = DebitorAccountBase + customer index.
const DebitorAccountBase = 10000

// RevenueAccountForRate returns the SKR03 revenue account for the given tax rate.
// Rate is expected as a whole number string: "19", "7", or "0".
func RevenueAccountForRate(rate string) string {
	switch rate {
	case "19":
		return AccountRevenue19
	case "7":
		return AccountRevenue7
	default:
		// 0% rate: could be EU or Kleinunternehmer.
		// For DATEV export, we default to Kleinunternehmer.
		// The caller should use RevenueAccountForRateAndMode for precision.
		return AccountRevenueKleinunternehmer
	}
}

// RevenueAccountForRateAndMode returns the SKR03 revenue account considering tax mode.
func RevenueAccountForRateAndMode(rate string, taxMode string) string {
	switch rate {
	case "19":
		return AccountRevenue19
	case "7":
		return AccountRevenue7
	default:
		// 0% rate: distinguish between EU reverse charge and Kleinunternehmer
		if taxMode == "reverse_charge" {
			return AccountRevenueEU
		}
		return AccountRevenueKleinunternehmer
	}
}

// BUSchluesselForRate returns the DATEV BU-Schluessel for the given tax rate.
func BUSchluesselForRate(rate string) string {
	switch rate {
	case "19":
		return BUSchluessel19
	case "7":
		return BUSchluessel7
	default:
		return BUSchluesselNone
	}
}
