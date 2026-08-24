package gateway

// Boundary and validation coverage for route_biz_bank_accounts.go: before this
// unit all five handlers had zero HTTP-level tests (only the wire-shape
// mapping in route_biz_bank_accounts_test.go was covered). Follows the
// established ServiceUnavailable/NoTenant table pattern from
// route_biz_expenses_gate_test.go / route_biz_bank_transactions_gate_test.go.
//
// PRÄMISSENKORREKTUR (Regel 11), zwei der drei Backlog-Fragen:
//
//  1. "Löschen eines Kontos, an dem Buchungen hängen: 500 oder 409?" ist
//     gegenstandslos. Migration 000258 (siehe Kopfkommentar dort) legt bewusst
//     KEINE Fremdschlüsselbeziehung zwischen finance_bank_accounts und
//     finance_bank_statements an -- die Zuordnung läuft über den freien Text
//     account_iban, nicht über eine FK-Spalte. DeleteAccount
//     (postgres_repository_accounts.go:114) ist ein reines
//     "DELETE ... WHERE tenant_id = $1 AND id = $2" ohne jede Referenz auf
//     Statements; ein Löschen kann also nie an einer Fremdschlüssel-Constraint
//     scheitern, unabhängig davon, wie viele Buchungen unter derselben IBAN
//     liegen. Es gibt daher keinen 500-vs-409-Fall zu bauen.
//  2. "Fremdtenant: ein Konto eines anderen Mandanten darf weder lesbar noch
//     löschbar sein" ist bereits ein service-Level-Fall und dort belegt:
//     TestDeleteAccount_StaysInsideTheTenant (service_accounts_test.go) und
//     GetAccount/DeleteAccount scopen beide immer per SQL auf
//     "tenant_id = $1 AND id = $2" (postgres_repository_accounts.go:64,116).
//     Der Gateway-Handler reicht die Tenant-ID aus dem Request-Context
//     unverändert durch; ohne echte RPC ist ein fremder Tenant hier nicht
//     simulierbar, genau wie bei den Nachbarrouten.
//
// Die dritte Frage -- IBAN-Prüfziffer -- ist dagegen ein echter Gateway-Fall:
// createBankAccountRequest/updateBankAccountRequest validieren lokal über den
// registrierten "iban"-Tag (dachfmt.ValidateIBAN, ISO 7064 mod-97, siehe
// internal/validation/validation.go:57), bevor irgendeine RPC erreicht wird.
// Eine IBAN mit falscher Prüfziffer oder aus einem unbekannten Land muss also
// mit 400 statt mit einer stillen Weiterleitung an die RPC enden -- und keine
// IBAN erscheint dabei je in einer Fehlermeldung oder einem Log
// (service_accounts.go:6-8 dokumentiert das ausdrücklich, kein Logaufruf im
// Paket trägt IBAN/BIC/BankName als Feld).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bankAccountsTestHandlers(routes *BizRoutes) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"HandleListBankAccounts":   routes.HandleListBankAccounts,
		"HandleCreateBankAccount":  routes.HandleCreateBankAccount,
		"HandleUpdateBankAccount":  routes.HandleUpdateBankAccount,
		"HandleConnectBankAccount": routes.HandleConnectBankAccount,
		"HandleDeleteBankAccount":  routes.HandleDeleteBankAccount,
	}
}

func TestBankAccountsRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	for name, h := range bankAccountsTestHandlers(routes) {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

func TestBankAccountsRoutes_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	for name, h := range bankAccountsTestHandlers(routes) {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			// Deliberately no withTenantID — simulates a token without a tid claim.
			h(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
			assertErrorContains(t, rec, "missing or invalid tenant")
		})
	}
}

func TestHandleListBankAccounts_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-accounts", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankAccounts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateBankAccount — the IBAN check digit is the load-bearing rule:
// a syntactically plausible but wrong IBAN must never reach the RPC.
// ============================================================================

func TestHandleCreateBankAccount_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateBankAccount_MissingBankNameRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"iban":"DE89370400440532013000"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertValidationError(t, rec, "bankName")
}

func TestHandleCreateBankAccount_WrongCheckDigitRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	// Same IBAN as the valid case below, last digit of the BBAN flipped —
	// structurally plausible, fails the ISO 7064 mod-97 check.
	body := `{"bankName":"Commerzbank","iban":"DE89370400440532013001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertValidationError(t, rec, "iban")
	if strings.Contains(rec.Body.String(), "DE89370400440532013001") {
		t.Errorf("response leaks the rejected IBAN: %s", rec.Body.String())
	}
}

func TestHandleCreateBankAccount_UnknownCountryRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"bankName":"Some Bank","iban":"XX89370400440532013000"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertValidationError(t, rec, "iban")
}

func TestHandleCreateBankAccount_InvalidBICRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"bankName":"Commerzbank","iban":"DE89370400440532013000","bic":"NOTABIC"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertValidationError(t, rec, "bic")
}

func TestHandleCreateBankAccount_InvalidCurrencyRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"bankName":"Commerzbank","iban":"DE89370400440532013000","currency":"EURO"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertValidationError(t, rec, "currency")
}

// TestHandleCreateBankAccount_ValidIBANWithSeparators_ReachesRPC pins that a
// human-typed, spaced IBAN with a correct check digit clears validation — the
// grouping is cosmetic and the validator ignores it (dachfmt.ValidateIBAN
// strips whitespace before the mod-97 check).
func TestHandleCreateBankAccount_ValidIBANWithSeparators_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"bankName":"Commerzbank","iban":"DE89 3704 0044 0532 0130 00","bic":"COBADEFFXXX","currency":"eur"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateBankAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateBankAccount — every field is optional (a partial edit), so an
// empty body must clear validation and reach the RPC; a present IBAN is
// checked exactly like on create.
// ============================================================================

func TestHandleUpdateBankAccount_EmptyBody_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/finance/bank-accounts/id", strings.NewReader("{}"))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateBankAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateBankAccount_WrongCheckDigitRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"iban":"DE89370400440532013001"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/finance/bank-accounts/id", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateBankAccount(rec, req)
	assertValidationError(t, rec, "iban")
}

func TestHandleUpdateBankAccount_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/finance/bank-accounts/id", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateBankAccount(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// ============================================================================
// HandleConnectBankAccount / HandleDeleteBankAccount — no body, no
// validation; this only pins that both reach the RPC with the URL id.
// ============================================================================

func TestHandleConnectBankAccount_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-accounts/id/connect", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleConnectBankAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteBankAccount_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/finance/bank-accounts/id", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleDeleteBankAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
