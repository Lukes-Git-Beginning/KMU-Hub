package gateway

// Boundary and validation coverage for route_biz_bank_transactions.go and
// route_biz_transactions.go: before this unit all six handlers across both
// files had zero HTTP-level tests (only the wire-shape mapping in
// route_biz_bank_transactions_test.go / route_biz_transactions_test.go was
// covered). Follows the established ServiceUnavailable/NoTenant table
// pattern from route_biz_expenses_gate_test.go.
//
// Abgrenzung der beiden Dateien (per Scope gefordert): route_biz_bank_
// transactions.go ist die Reconciliation-Queue eines bereits importierten
// Kontoauszugs (list/match/reject-match/ignore, gescoped auf einen
// bank_transactions-Datensatz). route_biz_transactions.go ist die
// mandantenweite, konsolidierte Zahlungsübersicht über ALLE Belegarten
// (Rechnungen, Ausgaben, ...) hinweg -- list/delete auf einen
// finance_transactions-Eintrag. Beide Dateien haben keinerlei Code-Bezug
// zueinander (kein gemeinsamer Typ, kein gemeinsamer Aufruf); die Verwechslung
// im Backlog kam allein von den ähnlichen Dateinamen.
//
// PRÄMISSENKORREKTUR (Regel 11): der Backlog-Scope beschreibt einen Upload als
// "Vertrauensgrenze" mit 400/413-Prüfung und einer Dublettenfrage. Es gibt in
// KEINER der beiden hier gescopten Dateien einen Upload-Handler -- der
// CAMT.053/MT940-Import lebt in route_biz_banking.go
// (HandleImportBankStatement), einer dritten, bereits vollständig getesteten
// Datei (route_biz_banking_test.go, die genau die 400/413/leer-Fälle und die
// AlreadyImported-Dublettenerkennung abdeckt). Die vier hier gescopten
// Handler bekommen bereits fertige BankTransaction-Zeilen aus einem
// vorherigen Import und haben keinen Dateikörper. Ebenso ist eine
// Zuordnung auf eine Rechnung eines fremden Mandanten kein Gateway-Fall:
// ReconcileBankTransaction (biz_grpc_banking.go:164) reicht Tenant-ID und
// Invoice-ID unverändert an internal/biz/banking weiter, das die Rechnung
// tenant-gescoped lädt -- eine fremde Rechnung ist dort schlicht "nicht
// gefunden", nicht über HTTP simulierbar ohne echte RPC. Beide Fragen bleiben
// damit dort, wo sie schon beantwortet sind, statt hier ein zweites Mal
// (und ohne echten Unterbau) geprüft zu werden.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bankTransactionsTestHandlers(routes *BizRoutes) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"HandleListBankTransactions":       routes.HandleListBankTransactions,
		"HandleMatchBankTransaction":       routes.HandleMatchBankTransaction,
		"HandleRejectBankTransactionMatch": routes.HandleRejectBankTransactionMatch,
		"HandleIgnoreBankTransaction":      routes.HandleIgnoreBankTransaction,
		"HandleListTransactions":           routes.HandleListTransactions,
		"HandleDeleteTransaction":          routes.HandleDeleteTransaction,
	}
}

func TestBankTransactionsRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	for name, h := range bankTransactionsTestHandlers(routes) {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

func TestBankTransactionsRoutes_NoTenant(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	for name, h := range bankTransactionsTestHandlers(routes) {
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

// ============================================================================
// HandleListBankTransactions — the interesting case is the status query
// param: bankMatchStatuses is a closed whitelist and an unrecognized value
// must be rejected locally rather than passed through to the RPC.
// ============================================================================

func TestHandleListBankTransactions_UnknownStatusRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-transactions?status=pending", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankTransactions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "unknown status")
}

func TestHandleListBankTransactions_DefaultExcludesIgnored_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-transactions", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankTransactions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBankTransactions_IgnoredStatus_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-transactions?status=ignored", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankTransactions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBankTransactions_StatementIDFilter_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/bank-transactions?statement_id=11111111-1111-1111-1111-111111111111", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListBankTransactions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMatchBankTransaction — invoice_id must be a well-formed UUID and
// invoice_number is capped at 64 chars; both are omitempty because confirming
// the import's own suggestion sends neither.
// ============================================================================

func TestHandleMatchBankTransaction_InvalidInvoiceIDRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"invoice_id":"not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/match", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleMatchBankTransaction(rec, req)
	assertValidationError(t, rec, "invoice_id")
}

func TestHandleMatchBankTransaction_InvoiceNumberTooLongRejected(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"invoice_number":"` + strings.Repeat("R", 65) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/match", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleMatchBankTransaction(rec, req)
	assertValidationError(t, rec, "invoice_number")
}

func TestHandleMatchBankTransaction_InvalidJSON(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/match", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleMatchBankTransaction(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleMatchBankTransaction_ConfirmImportSuggestion_ReachesRPC pins the
// common case: an empty body confirms the suggestion the import already
// attached, so both fields being absent must clear validation.
func TestHandleMatchBankTransaction_ConfirmImportSuggestion_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/match", strings.NewReader("{}"))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleMatchBankTransaction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMatchBankTransaction_ExplicitInvoice_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	body := `{"invoice_id":"22222222-2222-4222-a222-222222222222","invoice_number":"RE-2026-042"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/match", strings.NewReader(body))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleMatchBankTransaction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRejectBankTransactionMatch / HandleIgnoreBankTransaction — no body,
// no validation; the service-level distinction (stays unmatched vs. leaves
// the queue) is internal/biz/banking's concern. This only pins that both
// reach the RPC.
// ============================================================================

func TestHandleRejectBankTransactionMatch_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/reject-match", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleRejectBankTransactionMatch(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleIgnoreBankTransaction_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/bank-transactions/id/ignore", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleIgnoreBankTransaction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// route_biz_transactions.go — HandleListTransactions has no params to reject;
// HandleDeleteTransaction's "decided record stays" guard (an approved expense
// refuses deletion with 409) lives in the underlying services and is
// unreachable here without a real RPC, same as HandleDeleteExpense.
// ============================================================================

func TestHandleListTransactions_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/transactions", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTransactions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteTransaction_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/finance/transactions/id", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "33333333-3333-3333-3333-333333333333")
	routes.HandleDeleteTransaction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
