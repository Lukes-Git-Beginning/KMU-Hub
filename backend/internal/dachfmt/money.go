package dachfmt

import (
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
