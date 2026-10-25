package gateway

// The edit-member dialog sends the address and emergency-contact fields in the same body as
// the HR fields. They used to be dropped at this border (200, nothing stored), so the tests
// read the request the HR service actually receives.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHRWire_UpdateEmployeeForwardsAddressAndEmergencyContact(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateEmployee(rec, hrRequest(t, http.MethodPut, map[string]any{
		"emergency_contact_name":  "Erika Muster",
		"emergency_contact_phone": "+49 30 1234567",
		"address_street":          "Hauptstr. 1",
		"address_city":            "Berlin",
		"address_postal_code":     "10115",
		"address_country":         "DE",
	}, testHRUserID))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	got := stub.employeeUpdated
	require.NotNil(t, got)
	assert.Equal(t, "Erika Muster", got.GetEmergencyContactName())
	assert.Equal(t, "+49 30 1234567", got.GetEmergencyContactPhone())
	assert.Equal(t, "Hauptstr. 1", got.GetAddressStreet())
	assert.Equal(t, "Berlin", got.GetAddressCity())
	assert.Equal(t, "10115", got.GetAddressPostalCode())
	assert.Equal(t, "DE", got.GetAddressCountry())
}

// Omitted must stay nil (keep the stored value); "" must arrive as a present empty string
// (clear it). A plain string field would make both look the same.
func TestHRWire_UpdateEmployeeKeepsOmittedAndClearsEmptyFields(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateEmployee(rec, hrRequest(t, http.MethodPut, map[string]any{
		"department":     "Vertrieb",
		"address_street": "",
	}, testHRUserID))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	got := stub.employeeUpdated
	require.NotNil(t, got)
	require.NotNil(t, got.AddressStreet, "empty string must be sent as present")
	assert.Empty(t, got.GetAddressStreet())
	assert.Nil(t, got.AddressCity)
	assert.Nil(t, got.AddressCountry)
	assert.Nil(t, got.EmergencyContactName)
}

func TestHRWire_UpdateEmployeeRejectsAMalformedCountryCode(t *testing.T) {
	for _, country := range []string{"DEU", "D"} {
		routes, stub := newHRWireRoutes(t)
		rec := httptest.NewRecorder()
		routes.HandleUpdateEmployee(rec, hrRequest(t, http.MethodPut,
			map[string]any{"address_country": country}, testHRUserID))
		assert.Equal(t, http.StatusBadRequest, rec.Code, country)
		assert.Nil(t, stub.employeeUpdated, "must not reach the service: %s", country)
	}
}

// Deactivating is an offboarding (exit date and type are mandatory); a 200 that changes
// nothing would let the Team page show a deactivation that never happened.
func TestHRWire_UpdateEmployeeRefusesStatusWithAnOffboardingHint(t *testing.T) {
	for _, status := range []string{"inactive", "active"} {
		routes, stub := newHRWireRoutes(t)
		rec := httptest.NewRecorder()
		routes.HandleUpdateEmployee(rec, hrRequest(t, http.MethodPut,
			map[string]any{"status": status, "department": "Vertrieb"}, testHRUserID))
		assert.Equal(t, http.StatusBadRequest, rec.Code, status)
		assert.Contains(t, rec.Body.String(), "Offboarding")
		assert.Nil(t, stub.employeeUpdated, "nothing of a refused body may be applied")
	}
}
