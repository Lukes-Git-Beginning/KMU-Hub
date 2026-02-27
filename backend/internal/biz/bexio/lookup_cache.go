package bexio

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// LookupCache caches Bexio reference data (countries, currencies, salutations).
// Refreshed at the start of each sync run.
type LookupCache struct {
	mu           sync.RWMutex
	countries    map[string]int // name -> ID
	countryCodes map[string]int // ISO code -> ID
	currencies   map[string]int // code -> ID
	salutations  map[string]int // name -> ID
}

// NewLookupCache creates an empty lookup cache.
func NewLookupCache() *LookupCache {
	return &LookupCache{
		countries:    make(map[string]int),
		countryCodes: make(map[string]int),
		currencies:   make(map[string]int),
		salutations:  make(map[string]int),
	}
}

// Refresh fetches lookup tables from the Bexio API and populates the cache.
func (lc *LookupCache) Refresh(ctx context.Context, client *Client, tenantID uuid.UUID) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Fetch countries
	var countries []BexioLookupEntry
	if err := client.doList(ctx, tenantID, "country", nil, &countries); err != nil {
		return fmt.Errorf("bexio: refresh countries: %w", err)
	}
	lc.countries = make(map[string]int, len(countries))
	for _, c := range countries {
		lc.countries[strings.ToLower(c.Name)] = c.ID
	}

	// Fetch currencies
	var currencies []BexioLookupEntry
	if err := client.doList(ctx, tenantID, "currency", nil, &currencies); err != nil {
		return fmt.Errorf("bexio: refresh currencies: %w", err)
	}
	lc.currencies = make(map[string]int, len(currencies))
	for _, c := range currencies {
		lc.currencies[strings.ToUpper(c.Name)] = c.ID
	}

	// Fetch salutations
	var salutations []BexioLookupEntry
	if err := client.doList(ctx, tenantID, "salutation", nil, &salutations); err != nil {
		return fmt.Errorf("bexio: refresh salutations: %w", err)
	}
	lc.salutations = make(map[string]int, len(salutations))
	for _, s := range salutations {
		lc.salutations[strings.ToLower(s.Name)] = s.ID
	}

	slog.Info("bexio lookup cache refreshed",
		"countries", len(lc.countries),
		"currencies", len(lc.currencies),
		"salutations", len(lc.salutations),
	)

	return nil
}

// GetCountryID resolves a country name to its Bexio ID.
func (lc *LookupCache) GetCountryID(name string) (int, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	id, ok := lc.countries[strings.ToLower(name)]
	return id, ok
}

// GetCurrencyID resolves a currency code to its Bexio ID.
func (lc *LookupCache) GetCurrencyID(code string) (int, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	id, ok := lc.currencies[strings.ToUpper(code)]
	return id, ok
}

// GetSalutationID resolves a salutation name to its Bexio ID.
func (lc *LookupCache) GetSalutationID(name string) (int, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	id, ok := lc.salutations[strings.ToLower(name)]
	return id, ok
}
