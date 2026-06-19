package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestTaxRateForMode guards F11: tax-exempt modes must yield a 0% line rate,
// the standard rate otherwise.
func TestTaxRateForMode(t *testing.T) {
	assert.True(t, taxRateForMode(models.TaxModeKleinunternehmer).IsZero(),
		"kleinunternehmer must be tax-exempt")
	assert.True(t, taxRateForMode(models.TaxModeReverseCharge).IsZero(),
		"reverse-charge must be tax-exempt")
	assert.Equal(t, "19", taxRateForMode(models.TaxModeStandard).String(),
		"standard mode must be 19%")
	assert.Equal(t, "19", taxRateForMode("").String(),
		"empty mode must fall back to the standard rate")
}

// TestResolveTaxMode guards F11: an explicit request mode wins; an empty mode
// derives from company settings (Kleinunternehmer → kleinunternehmer).
func TestResolveTaxMode(t *testing.T) {
	klein := &models.CompanySettings{IsKleinunternehmer: true}
	regular := &models.CompanySettings{IsKleinunternehmer: false}

	// Explicit request mode always wins.
	assert.Equal(t, models.TaxModeReverseCharge,
		resolveTaxMode(models.TaxModeReverseCharge, klein))
	assert.Equal(t, models.TaxModeStandard,
		resolveTaxMode(models.TaxModeStandard, klein))

	// Empty request mode derives from settings.
	assert.Equal(t, models.TaxModeKleinunternehmer, resolveTaxMode("", klein))
	assert.Equal(t, models.TaxModeStandard, resolveTaxMode("", regular))

	// Nil settings (load error / not configured) → standard.
	assert.Equal(t, models.TaxModeStandard, resolveTaxMode("", nil))
}
