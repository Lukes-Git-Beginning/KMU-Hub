//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Enum ordinals as they leave the gateway: quotes and invoices keep the proto number
// (the desktop maps it back in api/finance-status.ts), credit notes leave as a word.
const (
	quoteDraft      = 1.0
	quoteSent       = 2.0
	quoteAccepted   = 3.0
	invoiceDraft    = 1.0
	invoiceSent     = 2.0
	invoiceCanceled = 5.0
)

// financeRequest is doRequest plus the Idempotency-Key header. Production runs the
// gateway with IDEMPOTENCY_MODE=hard, which answers every finance mutation without the
// header with 400, so a flow that omits it passes here and fails for every real user.
func financeRequest(t *testing.T, method, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if method != http.MethodGet {
		req.Header.Set("Idempotency-Key", fmt.Sprintf("e2e-finance-%d", time.Now().UnixNano()))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

// decimalField reads a decimal string ("238.00", "238") from m[key].
func decimalField(t *testing.T, m map[string]interface{}, key string) float64 {
	t.Helper()
	s, ok := m[key].(string)
	if !ok {
		t.Fatalf("expected %s as a decimal string, got %v", key, m[key])
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("%s is not a decimal: %q", key, s)
	}
	return v
}

// TestFinanceQuoteToCreditNoteFlow walks the money path of a customer order through the
// gateway as an admin user: quote -> send -> accept -> convert -> invoice send -> cancel.
// Cancelling an issued invoice books a Storno credit note, so the last steps prove that
// the credit note exists, is finalised, mirrors the invoice and carries the caller as
// creator. send/convert/cancel read the actor from the token (callerID), so a nil actor
// shows up as a 403 on exactly those steps.
func TestFinanceQuoteToCreditNoteFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, userID := registerAndLoginAdmin(t, base)

	customer := map[string]string{
		"name":    fmt.Sprintf("E2E Kunde %d", time.Now().UnixNano()),
		"address": "Musterstrasse 1, 10115 Berlin",
		"email":   "kunde@example.com",
	}

	var quoteID, quoteNumber, invoiceID, invoiceNumber, creditNoteID string

	t.Run("create_quote_validation", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes", map[string]interface{}{
			"customer":   customer,
			"line_items": []map[string]string{},
			"tax_mode":   "standard",
		}, token)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes", map[string]interface{}{
			"customer":   customer,
			"line_items": []map[string]string{{"description": "x", "quantity": "1", "unit_price": "1", "tax_rate": "19"}},
			"tax_mode":   "no_such_mode",
		}, token)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("create_quote", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes", map[string]interface{}{
			"customer": customer,
			"line_items": []map[string]string{
				{"description": "Beratung", "quantity": "2", "unit_price": "100.00", "tax_rate": "19"},
			},
			"tax_mode": "standard",
			"notes":    "E2E",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var q map[string]interface{}
		decodeBody(t, body, &q)
		quoteID, _ = q["id"].(string)
		if quoteID == "" {
			t.Fatalf("expected quote id, got %s", string(body))
		}
		if q["status"] != quoteDraft {
			t.Fatalf("expected a draft quote (1), got %v", q["status"])
		}
		if q["created_by"] != userID {
			t.Fatalf("expected created_by %s, got %v", userID, q["created_by"])
		}
		if q["tax_mode"] != "standard" {
			t.Fatalf("expected tax_mode standard, got %v", q["tax_mode"])
		}
		if n, _ := q["quote_number"].(string); n != "" {
			t.Fatalf("a draft quote has no number yet, got %q", n)
		}
		items, _ := q["line_items"].([]interface{})
		if len(items) != 1 {
			t.Fatalf("expected one line item, got %s", string(body))
		}
		tb, _ := q["tax_breakdown"].(map[string]interface{})
		if tb == nil {
			t.Fatalf("expected a tax_breakdown, got %s", string(body))
		}
		if gross := decimalField(t, tb, "gross_total"); gross != 238 {
			t.Fatalf("expected gross_total 238 (2 x 100 + 19%%), got %v", gross)
		}
	})

	t.Run("convert_draft_quote_is_rejected", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/convert", map[string]string{}, token)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("send_quote", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/send", nil, token)
		requireStatus(t, resp, body, http.StatusOK)

		var q map[string]interface{}
		decodeBody(t, body, &q)
		if q["status"] != quoteSent {
			t.Fatalf("expected a sent quote (2), got %v", q["status"])
		}
		quoteNumber, _ = q["quote_number"].(string)
		if !strings.HasPrefix(quoteNumber, "AN-") {
			t.Fatalf("expected a gap-free AN- number after sending, got %q", quoteNumber)
		}
	})

	t.Run("send_quote_twice_conflicts", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/send", nil, token)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("accept_quote", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/accept", nil, token)
		requireStatus(t, resp, body, http.StatusOK)

		var q map[string]interface{}
		decodeBody(t, body, &q)
		if q["status"] != quoteAccepted {
			t.Fatalf("expected an accepted quote (3), got %v", q["status"])
		}
		if q["quote_number"] != quoteNumber {
			t.Fatalf("the number must not change on accept: %v vs %s", q["quote_number"], quoteNumber)
		}
	})

	t.Run("convert_quote_to_invoice", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/convert", map[string]string{}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var inv map[string]interface{}
		decodeBody(t, body, &inv)
		invoiceID, _ = inv["id"].(string)
		if invoiceID == "" {
			t.Fatalf("expected invoice id, got %s", string(body))
		}
		if inv["status"] != invoiceDraft {
			t.Fatalf("expected a draft invoice (1), got %v", inv["status"])
		}
		if inv["source_quote_id"] != quoteID {
			t.Fatalf("expected source_quote_id %s, got %v", quoteID, inv["source_quote_id"])
		}
		if inv["created_by"] != userID {
			t.Fatalf("expected created_by %s, got %v", userID, inv["created_by"])
		}
		cust, _ := inv["customer"].(map[string]interface{})
		if cust["name"] != customer["name"] {
			t.Fatalf("expected the customer copied from the quote, got %s", string(body))
		}
		tb, _ := inv["tax_breakdown"].(map[string]interface{})
		if tb == nil {
			t.Fatalf("expected a tax_breakdown, got %s", string(body))
		}
		if gross := decimalField(t, tb, "gross_total"); gross != 238 {
			t.Fatalf("expected the quote total 238 on the invoice, got %v", gross)
		}
	})

	t.Run("convert_twice_conflicts", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/quotes/"+quoteID+"/convert", map[string]string{}, token)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("send_invoice", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/invoices/"+invoiceID+"/send", nil, token)
		requireStatus(t, resp, body, http.StatusOK)

		var inv map[string]interface{}
		decodeBody(t, body, &inv)
		if inv["status"] != invoiceSent {
			t.Fatalf("expected a sent invoice (2), got %v", inv["status"])
		}
		invoiceNumber, _ = inv["invoice_number"].(string)
		if !strings.HasPrefix(invoiceNumber, "RE-") {
			t.Fatalf("expected a gap-free RE- number after sending, got %q", invoiceNumber)
		}
	})

	t.Run("send_invoice_twice_conflicts", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/invoices/"+invoiceID+"/send", nil, token)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("quote_shows_the_invoice_number", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/finance/quotes/"+quoteID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var q map[string]interface{}
		decodeBody(t, body, &q)
		if q["converted_invoice_number"] != invoiceNumber {
			t.Fatalf("expected converted_invoice_number %s, got %v", invoiceNumber, q["converted_invoice_number"])
		}
	})

	t.Run("cancel_invoice_books_storno", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/invoices/"+invoiceID+"/cancel", nil, token)
		requireStatus(t, resp, body, http.StatusOK)

		var inv map[string]interface{}
		decodeBody(t, body, &inv)
		if inv["status"] != invoiceCanceled {
			t.Fatalf("expected a cancelled invoice (5), got %v", inv["status"])
		}
		if inv["invoice_number"] != invoiceNumber {
			t.Fatalf("the invoice keeps its number when cancelled: %v vs %s", inv["invoice_number"], invoiceNumber)
		}
	})

	t.Run("cancel_twice_conflicts", func(t *testing.T) {
		resp, body := financeRequest(t, http.MethodPost, base+"/api/v1/finance/invoices/"+invoiceID+"/cancel", nil, token)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("storno_credit_note_is_listed", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/finance/credit-notes?invoice_id="+invoiceID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			CreditNotes []map[string]interface{} `json:"credit_notes"`
			Total       float64                  `json:"total"`
		}
		decodeBody(t, body, &result)
		if len(result.CreditNotes) != 1 || result.Total != 1 {
			t.Fatalf("expected exactly one Storno credit note, got %s", string(body))
		}
		cn := result.CreditNotes[0]
		creditNoteID, _ = cn["id"].(string)
		if creditNoteID == "" {
			t.Fatalf("expected credit note id, got %s", string(body))
		}
		if cn["status"] != "sent" {
			t.Fatalf("expected the Storno credit note finalised (sent), got %v", cn["status"])
		}
		if n, _ := cn["credit_note_number"].(string); !strings.HasPrefix(n, "GS-") {
			t.Fatalf("expected a gap-free GS- number, got %v", cn["credit_note_number"])
		}
		if cn["original_invoice_id"] != invoiceID {
			t.Fatalf("expected original_invoice_id %s, got %v", invoiceID, cn["original_invoice_id"])
		}
		if cn["created_by"] != userID {
			t.Fatalf("expected the Storno created_by %s (the caller), got %v", userID, cn["created_by"])
		}
		if reason, _ := cn["reason"].(string); !strings.Contains(reason, invoiceNumber) {
			t.Fatalf("expected the reason to name invoice %s, got %q", invoiceNumber, reason)
		}
		tb, _ := cn["tax_breakdown"].(map[string]interface{})
		if tb == nil {
			t.Fatalf("expected a tax_breakdown, got %s", string(body))
		}
		if gross := decimalField(t, tb, "gross_total"); gross != 238 {
			t.Fatalf("expected the Storno to mirror the invoice total 238, got %v", gross)
		}
	})

	t.Run("get_credit_note", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/finance/credit-notes/"+creditNoteID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			CreditNote map[string]interface{} `json:"credit_note"`
		}
		decodeBody(t, body, &result)
		if result.CreditNote["id"] != creditNoteID {
			t.Fatalf("expected credit_note.id %s, got %s", creditNoteID, string(body))
		}
	})

	t.Run("cancelled_invoice_frees_the_quote", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/finance/quotes/"+quoteID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var q map[string]interface{}
		decodeBody(t, body, &q)
		if n, _ := q["converted_invoice_number"].(string); n != "" {
			t.Fatalf("a cancelled conversion must leave converted_invoice_number empty, got %q", n)
		}
	})

	t.Run("missing_idempotency_key_is_not_a_server_error", func(t *testing.T) {
		// Dev stacks run WarnMode (passes through), production HardMode (400); either way
		// the request must be answered, never fail with a 5xx.
		resp, body := postJSON(t, base+"/api/v1/finance/quotes/"+quoteID+"/send", nil, token)
		if resp.StatusCode >= 500 {
			t.Fatalf("expected no 5xx without an Idempotency-Key, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("unknown_ids_return_404", func(t *testing.T) {
		const missing = "00000000-0000-4000-8000-000000000001"
		for _, path := range []string{"/quotes/" + missing, "/invoices/" + missing, "/credit-notes/" + missing} {
			resp, body := getJSON(t, base+"/api/v1/finance"+path, token)
			requireStatus(t, resp, body, http.StatusNotFound)
		}
	})

	t.Run("no_auth", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/finance/quotes/"+quoteID, "")
		requireStatus(t, resp, body, http.StatusUnauthorized)
	})
}
