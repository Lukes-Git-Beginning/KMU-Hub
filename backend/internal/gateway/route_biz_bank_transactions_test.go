package gateway

// The bank transaction wire shape is hand-mapped rather than
// protojson-marshalled, so nothing but a test stands between it and the desktop
// client. These pin the places where the frontend contract (the BankTransaction
// type in types/finance-types.ts, read through financeBankingApi) deliberately
// departs from this repo's protojson conventions.

import (
	"encoding/json"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestToBankTransactionWire_MatchesTheFrontendContract(t *testing.T) {
	t.Parallel()

	invoiceID := "33333333-3333-3333-3333-333333333333"
	invoiceNumber := "RE-2026-003"
	got, err := json.Marshal(toBankTransactionWire(&bizv1.BankTransaction{
		Id:                   "11111111-1111-1111-1111-111111111111",
		ValueDate:            "2026-07-30",
		RemittanceInfo:       "Eingang Gruber Maschinenbau — Abschlag 3",
		Amount:               "16000.00",
		Currency:             "EUR",
		CounterpartyName:     "Gruber Maschinenbau GmbH",
		MatchStatus:          "matched",
		MatchedInvoiceNumber: &invoiceNumber,
		// Internal bookkeeping the frontend never reads — it must not leak into
		// the response just because the proto carries it.
		StatementId:      "22222222-2222-2222-2222-222222222222",
		MatchedInvoiceId: &invoiceID,
		CounterpartyIban: "DE72701500000012345678",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// matchStatus/matchedInvoice are camelCase here where UseProtoNames would
	// emit match_status and matched_invoice_number.
	for _, key := range []string{"date", "description", "type", "counterpart", "matchStatus", "matchedInvoice"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%s missing; got keys %v", key, keysOf(decoded))
		}
	}
	for _, key := range []string{
		"match_status", "matched_invoice_number", "matched_invoice_id", "matchedInvoiceId",
		"statement_id", "counterparty_iban", "remittance_info", "value_date",
	} {
		if _, ok := decoded[key]; ok {
			t.Errorf("%s present — it is not part of the frontend contract", key)
		}
	}

	// amount is a JSON number where the proto carries a decimal string.
	if amount, ok := decoded["amount"].(float64); !ok || amount != 16000.00 {
		t.Errorf("amount = %v (%T), want 16000 as a number", decoded["amount"], decoded["amount"])
	}
	// matchedInvoice carries the number, not the id — the client prints it.
	if decoded["matchedInvoice"] != invoiceNumber {
		t.Errorf("matchedInvoice = %v, want %q", decoded["matchedInvoice"], invoiceNumber)
	}
	if decoded["type"] != "credit" {
		t.Errorf("type = %v, want credit for a positive amount", decoded["type"])
	}
}

func TestToBankTransactionWire_DebitAndEmptyFields(t *testing.T) {
	t.Parallel()

	// A bank fee: negative, no remittance information, nothing matched.
	w := toBankTransactionWire(&bizv1.BankTransaction{
		Id:          "44444444-4444-4444-4444-444444444444",
		ValueDate:   "2026-07-28",
		EntryRef:    "NONREF-8891",
		Amount:      "-1890.00",
		MatchStatus: "unmatched",
	})
	if w.Type != "debit" {
		t.Errorf("type = %q, want debit for a negative amount", w.Type)
	}
	if w.Amount != -1890.00 {
		t.Errorf("amount = %v, want the sign preserved", w.Amount)
	}
	// The entry reference stands in where the payer left the purpose empty —
	// an unidentifiable row in the queue cannot be decided on.
	if w.Description != "NONREF-8891" {
		t.Errorf("description = %q, want the entry reference as fallback", w.Description)
	}

	// matchedInvoice is optional in the frontend type, so an unmatched entry
	// must omit the key rather than send an empty string.
	got, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["matchedInvoice"]; ok {
		t.Error("matchedInvoice present on an unmatched entry — the frontend types it optional")
	}
}
