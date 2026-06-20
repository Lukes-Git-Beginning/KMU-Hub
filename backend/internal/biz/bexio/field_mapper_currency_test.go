package bexio

import "testing"

// TestResolveBexioCurrencyID covers the B6 currency resolution: a populated lookup
// cache maps the document currency to the tenant's Bexio currency ID, and every
// degenerate case falls back to 1 (CHF) — the historical hardcoded default.
func TestResolveBexioCurrencyID(t *testing.T) {
	cache := NewLookupCache()
	cache.currencies["EUR"] = 2
	cache.currencies["CHF"] = 1

	tests := []struct {
		name  string
		cache *LookupCache
		code  string
		want  int
	}{
		{"known EUR via cache", cache, "EUR", 2},
		{"empty code resolves to EUR default", cache, "", 2},
		{"unknown code falls back to CHF", cache, "USD", 1},
		{"nil cache falls back to CHF", nil, "EUR", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveBexioCurrencyID(tc.cache, tc.code); got != tc.want {
				t.Errorf("resolveBexioCurrencyID(%q) = %d, want %d", tc.code, got, tc.want)
			}
		})
	}
}
