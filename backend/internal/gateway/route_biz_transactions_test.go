package gateway

// The transaction wire shape is hand-mapped rather than protojson-marshalled,
// so nothing but a test stands between it and the desktop client. These pin
// the one place the frontend contract (the Transaction type in
// stores/finance.ts, read through financeTransactionApi in finance-client.ts)
// departs from this repo's protojson conventions: invoiceId is camelCase
// where UseProtoNames would emit invoice_id, and amount is a JSON number
// where the proto carries a decimal string.

import (
	"encoding/json"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestToFinanceTransactionWire_Income(t *testing.T) {
	t.Parallel()

	reference := "RE-2026-042"
	invoiceID := "11111111-1111-1111-1111-111111111111"
	got, err := json.Marshal(toFinanceTransactionWire(&bizv1.FinanceTransaction{
		Id:          "pay-22222222-2222-2222-2222-222222222222",
		Type:        "income",
		Description: "Rechnung RE-2026-042 Müller GmbH",
		Amount:      "4280.00",
		Date:        "2026-05-29",
		Category:    "Umsatzerlöse",
		Status:      "completed",
		Reference:   &reference,
		InvoiceId:   &invoiceID,
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := decoded["invoiceId"]; !ok {
		t.Errorf("invoiceId missing; got keys %v", keysOf(decoded))
	}
	if _, ok := decoded["invoice_id"]; ok {
		t.Error("invoice_id present — the frontend reads invoiceId")
	}
	if amount, ok := decoded["amount"].(float64); !ok || amount != 4280.0 {
		t.Errorf("amount = %v (%T), want 4280 as a number", decoded["amount"], decoded["amount"])
	}
	if decoded["reference"] != "RE-2026-042" {
		t.Errorf("reference = %v, want RE-2026-042", decoded["reference"])
	}
	if decoded["id"] != "pay-22222222-2222-2222-2222-222222222222" {
		t.Errorf("id = %v, want the pay- prefixed id verbatim", decoded["id"])
	}
}

func TestToFinanceTransactionWire_ExpenseOmitsAbsentOptionals(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(toFinanceTransactionWire(&bizv1.FinanceTransaction{
		Id:          "exp-33333333-3333-3333-3333-333333333333",
		Type:        "expense",
		Description: "Bürobedarf Staples",
		Amount:      "234.50",
		Date:        "2026-05-29",
		Category:    "Büromaterial",
		Status:      "completed",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, absent := range []string{"reference", "invoiceId"} {
		if _, ok := decoded[absent]; ok {
			t.Errorf("%s present, want omitted for an expense entry", absent)
		}
	}
	if amount, ok := decoded["amount"].(float64); !ok || amount != 234.5 {
		t.Errorf("amount = %v (%T), want 234.5 as a number", decoded["amount"], decoded["amount"])
	}
}
