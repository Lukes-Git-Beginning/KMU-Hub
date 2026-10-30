package dachfmt

import (
	"math"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// fallbackCurrency mirrors models.DefaultCurrency; dachfmt stays a leaf package.
const fallbackCurrency = "EUR"

// FormatMoney renders d the German way with two decimals: dot as thousands
// separator, comma as decimal separator, then the ISO 4217 code ("1.234,50 EUR",
// "-5,00 CHF"). An empty currency falls back to EUR. Rounding is StringFixed(2),
// i.e. half away from zero.
func FormatMoney(d decimal.Decimal, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = fallbackCurrency
	}
	s := d.StringFixed(2)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return sign + b.String() + "," + frac + " " + currency
}

// FormatMoneyFloat is FormatMoney for float64 columns. NaN and Inf (reachable
// from a Postgres NUMERIC 'NaN') yield "" instead of panicking in decimal.
func FormatMoneyFloat(f float64, currency string) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	return FormatMoney(decimal.NewFromFloat(f), currency)
}

// FormatFloatDE renders f with the given number of decimals and a decimal
// comma, without grouping ("12,5"), for percentages and similar ratios.
func FormatFloatDE(f float64, places int) string {
	return strings.Replace(strconv.FormatFloat(f, 'f', places, 64), ".", ",", 1)
}
