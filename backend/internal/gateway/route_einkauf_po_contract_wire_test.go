package gateway

// framework_contract_id on POST/PATCH /einkauf/pos as the desktop would send it:
// create forwards a UUID and drops an absent key, update distinguishes omitted
// (leave), `null`/"" (clear) and a UUID (bind), and a malformed id is a 400 that
// never reaches the service. The einkauf stub is the real loopback server of
// route_einkauf_dates_wire_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

const testFrameworkContractID = "22222222-2222-2222-2222-222222222222"

func TestEinkaufPOContract_CreateForwardsTheBinding(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreatePO(rec, taxModeRequest(http.MethodPost,
		`{`+supplierJSON+`,"po_number":"PO-1","framework_contract_id":"`+testFrameworkContractID+`"}`))
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.createPO)
	assert.Equal(t, testFrameworkContractID, stub.createPO.GetFrameworkContractId())

	stub.createPO = nil
	rec = httptest.NewRecorder()
	routes.HandleCreatePO(rec, taxModeRequest(http.MethodPost, `{`+supplierJSON+`,"po_number":"PO-2"}`))
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.createPO)
	assert.Nil(t, stub.createPO.FrameworkContractId, "an absent key creates an unbound PO")

	stub.createPO = nil
	rec = httptest.NewRecorder()
	routes.HandleCreatePO(rec, taxModeRequest(http.MethodPost,
		`{`+supplierJSON+`,"po_number":"PO-3","framework_contract_id":"RV-2026"}`))
	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.createPO, "a malformed id must not reach the service")
}

func TestEinkaufPOContract_UpdateLeavesClearsAndBinds(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)
	update := func(body string) *einkaufv1.UpdatePORequest {
		t.Helper()
		stub.updatePO = nil
		rec := httptest.NewRecorder()
		routes.HandleUpdatePO(rec, taxModeRequest(http.MethodPatch, body))
		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.updatePO)
		return stub.updatePO
	}

	assert.Nil(t, update(`{"notes":"x"}`).FrameworkContractId, "an omitted key leaves the binding alone")
	for _, body := range []string{`{"framework_contract_id":null}`, `{"framework_contract_id":""}`} {
		got := update(body).FrameworkContractId
		require.NotNil(t, got, "%s must send the clear marker", body)
		assert.Empty(t, *got)
	}
	assert.Equal(t, testFrameworkContractID,
		update(`{"framework_contract_id":"`+testFrameworkContractID+`"}`).GetFrameworkContractId())

	stub.updatePO = nil
	rec := httptest.NewRecorder()
	routes.HandleUpdatePO(rec, taxModeRequest(http.MethodPatch, `{"framework_contract_id":"RV-2026"}`))
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "framework_contract_id")
	assert.Nil(t, stub.updatePO, "a malformed id must not reach the service")
}
