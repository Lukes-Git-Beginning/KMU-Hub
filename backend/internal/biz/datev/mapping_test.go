package datev

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

func TestRateKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"19", "19"},
		{"19.00", "19"},
		{"7", "7"},
		{"7.5", "7.5"},
		{"0", "0"},
		{"0.00", "0"},
	}
	for _, tc := range cases {
		if got := RateKey(dec(t, tc.in)); got != tc.want {
			t.Errorf("RateKey(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRevenueAccountForRateAndMode(t *testing.T) {
	cases := []struct {
		name    string
		rate    string
		taxMode string
		want    string
	}{
		{"19 percent ignores mode", "19", "standard", AccountRevenue19},
		{"19.00 is still 19 percent", "19.00", "standard", AccountRevenue19},
		{"7 percent ignores mode", "7", "reverse_charge", AccountRevenue7},
		{"0 percent reverse charge is EU", "0", "reverse_charge", AccountRevenueEU},
		{"0 percent standard is Kleinunternehmer", "0", "standard", AccountRevenueKleinunternehmer},
		{"0 percent empty mode is Kleinunternehmer", "0", "", AccountRevenueKleinunternehmer},
		{"7.5 percent is not 7 percent", "7.5", "standard", AccountRevenueOther},
		{"19.5 percent is not 19 percent", "19.5", "standard", AccountRevenueOther},
		{"historic 16 percent is unmapped", "16", "standard", AccountRevenueOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RevenueAccountForRateAndMode(dec(t, tc.rate), tc.taxMode); got != tc.want {
				t.Errorf("RevenueAccountForRateAndMode(%s, %q) = %q, want %q", tc.rate, tc.taxMode, got, tc.want)
			}
		})
	}
}

func TestBUSchluesselForRate(t *testing.T) {
	cases := []struct {
		rate string
		want string
	}{
		{"19", BUSchluessel19},
		{"19.00", BUSchluessel19},
		{"7", BUSchluessel7},
		{"7.5", BUSchluesselNone},
		{"0", BUSchluesselNone},
		{"16", BUSchluesselNone},
	}
	for _, tc := range cases {
		if got := BUSchluesselForRate(dec(t, tc.rate)); got != tc.want {
			t.Errorf("BUSchluesselForRate(%s) = %q, want %q", tc.rate, got, tc.want)
		}
	}
}
