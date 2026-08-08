package datev

import "testing"

func TestRevenueAccountForRate(t *testing.T) {
	cases := []struct {
		rate string
		want string
	}{
		{"19", AccountRevenue19},
		{"7", AccountRevenue7},
		{"0", AccountRevenueKleinunternehmer}, // 0% defaults to Kleinunternehmer, see doc comment
		{"unknown", AccountRevenueKleinunternehmer},
	}
	for _, tc := range cases {
		if got := RevenueAccountForRate(tc.rate); got != tc.want {
			t.Errorf("RevenueAccountForRate(%q) = %q, want %q", tc.rate, got, tc.want)
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
		{"7 percent ignores mode", "7", "reverse_charge", AccountRevenue7},
		{"0 percent reverse charge is EU", "0", "reverse_charge", AccountRevenueEU},
		{"0 percent standard is Kleinunternehmer", "0", "standard", AccountRevenueKleinunternehmer},
		{"0 percent empty mode is Kleinunternehmer", "0", "", AccountRevenueKleinunternehmer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RevenueAccountForRateAndMode(tc.rate, tc.taxMode); got != tc.want {
				t.Errorf("RevenueAccountForRateAndMode(%q, %q) = %q, want %q", tc.rate, tc.taxMode, got, tc.want)
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
		{"7", BUSchluessel7},
		{"0", BUSchluesselNone},
		{"unknown", BUSchluesselNone},
	}
	for _, tc := range cases {
		if got := BUSchluesselForRate(tc.rate); got != tc.want {
			t.Errorf("BUSchluesselForRate(%q) = %q, want %q", tc.rate, got, tc.want)
		}
	}
}
