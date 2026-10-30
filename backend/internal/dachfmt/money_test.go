package dachfmt

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		in       string
		currency string
		want     string
	}{
		{"0", "EUR", "0,00 EUR"},
		{"238", "EUR", "238,00 EUR"},
		{"999.99", "EUR", "999,99 EUR"},
		{"1000", "EUR", "1.000,00 EUR"},
		{"1234.5", "EUR", "1.234,50 EUR"},
		{"1234567.891", "EUR", "1.234.567,89 EUR"},
		{"12345.675", "EUR", "12.345,68 EUR"},
		{"-1234.5", "EUR", "-1.234,50 EUR"},
		{"-0.5", "EUR", "-0,50 EUR"},
		{"1500", "CHF", "1.500,00 CHF"},
		{"1500", "", "1.500,00 EUR"},
		{"1500", " chf ", "1.500,00 CHF"},
	}
	for _, c := range cases {
		got := FormatMoney(decimal.RequireFromString(c.in), c.currency)
		if got != c.want {
			t.Errorf("FormatMoney(%s, %q) = %q, want %q", c.in, c.currency, got, c.want)
		}
	}
}
