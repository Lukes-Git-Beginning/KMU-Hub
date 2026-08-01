package gateway

// The expense wire shape is hand-mapped rather than protojson-marshalled, so
// nothing but a test stands between it and the desktop client. These pin the
// two places where the frontend contract (the Expense type in
// stores/finance.ts, read through financeExpenseApi in finance-client.ts)
// deliberately departs from this repo's protojson conventions.

import (
	"encoding/json"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestToExpenseWire_MatchesTheFrontendContract(t *testing.T) {
	t.Parallel()

	project, account, receiptName := "Projekt Müller-CRM", "4930", "staples-rechnung-q2.pdf"
	got, err := json.Marshal(toExpenseWire(&bizv1.Expense{
		Id:          "11111111-1111-1111-1111-111111111111",
		Description: "Bürobedarf Q2",
		Amount:      "234.50",
		ExpenseDate: "2026-05-29",
		Category:    "Büromaterial",
		Supplier:    "Staples",
		Project:     &project,
		Account:     &account,
		ReceiptName: &receiptName,
		Status:      "pending",
		// Internal bookkeeping the frontend never reads — it must not leak into
		// the response just because the proto carries it.
		TenantId:    "22222222-2222-2222-2222-222222222222",
		SubmittedBy: &account,
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// receiptName is camelCase here where UseProtoNames would emit receipt_name.
	if _, ok := decoded["receiptName"]; !ok {
		t.Errorf("receiptName missing; got keys %v", keysOf(decoded))
	}
	if _, ok := decoded["receipt_name"]; ok {
		t.Error("receipt_name present — the frontend reads receiptName")
	}
	// amount is a JSON number where the proto carries a decimal string.
	if amount, ok := decoded["amount"].(float64); !ok || amount != 234.50 {
		t.Errorf("amount = %v (%T), want 234.5 as a number", decoded["amount"], decoded["amount"])
	}
	// receipt is derived, not stored.
	if receipt, ok := decoded["receipt"].(bool); !ok || !receipt {
		t.Errorf("receipt = %v, want true when a filename is set", decoded["receipt"])
	}
	if decoded["date"] != "2026-05-29" {
		t.Errorf("date = %v, want the plain calendar day", decoded["date"])
	}
	for _, leaked := range []string{"tenant_id", "tenantId", "submitted_by", "submittedBy", "decided_by"} {
		if _, ok := decoded[leaked]; ok {
			t.Errorf("%s leaked into the response", leaked)
		}
	}
}

func TestToExpenseWire_OmitsAbsentOptionals(t *testing.T) {
	t.Parallel()

	// exp-3 in the mock seed: no account, no receipt, no project. The list view
	// renders the "Kontieren" prompt off exactly these absences.
	got, err := json.Marshal(toExpenseWire(&bizv1.Expense{
		Id:          "33333333-3333-3333-3333-333333333333",
		Description: "Geschäftsessen Kunde Müller",
		Amount:      "142.80",
		ExpenseDate: "2026-05-20",
		Status:      "pending",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, absent := range []string{"project", "account", "receiptName"} {
		if _, ok := decoded[absent]; ok {
			t.Errorf("%s present, want omitted when unset", absent)
		}
	}
	// receipt is not optional: the frontend reads it as a boolean, so false has
	// to be spelled out rather than left to undefined.
	if receipt, ok := decoded["receipt"].(bool); !ok || receipt {
		t.Errorf("receipt = %v, want an explicit false", decoded["receipt"])
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
