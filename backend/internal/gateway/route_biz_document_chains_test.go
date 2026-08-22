package gateway

// route_biz_document_chains.go had no dedicated test file.
//
// Note on scope: the backlog draft for this unit assumed a chain-traversal
// path here that could loop forever on a gap or cycle. That does not apply —
// PostgresRepository.ListDocumentChains (internal/biz/invoice/postgres_document_chains.go)
// assembles each chain as a flat, invoice-anchored fan-out (map lookups by
// invoice ID across five independent tenant-scoped queries), not a graph
// traversal; there is no adjacency list to have a gap or cycle in, on either
// the service or the gateway side. HandleListDocumentChains itself only
// ranges once over whatever finite slice the RPC returns. Verified by reading
// the repository; no test for that scenario is meaningful because there is no
// code path it could exercise.
//
// What IS worth isolating here, same idea as toBankAccountWire next door, is
// the hand-mapped wire shape: toDocumentChainWire/formatChainAmount/
// groupThousands are the only formatting logic in this file and are pure
// functions, testable without a live RPC.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// --- HandleListDocumentChains ---

func TestHandleListDocumentChains_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListDocumentChains)
}

func TestHandleListDocumentChains_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/document-chains", nil)
	routes.HandleListDocumentChains(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListDocumentChains_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/document-chains", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListDocumentChains(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- formatChainAmount ---

func TestFormatChainAmount_PositiveGrouped(t *testing.T) {
	got := formatChainAmount("12450.00", "EUR")
	if want := "EUR 12.450,00"; got != want {
		t.Errorf("formatChainAmount = %q, want %q", got, want)
	}
}

func TestFormatChainAmount_Negative(t *testing.T) {
	got := formatChainAmount("-1234.50", "CHF")
	if want := "CHF -1.234,50"; got != want {
		t.Errorf("formatChainAmount = %q, want %q", got, want)
	}
}

func TestFormatChainAmount_Unparsable_FallsBackToZero(t *testing.T) {
	got := formatChainAmount("not-a-number", "EUR")
	if want := "EUR 0,00"; got != want {
		t.Errorf("formatChainAmount = %q, want %q (unparsable input must not propagate)", got, want)
	}
}

func TestFormatChainAmount_EmptyString_FallsBackToZero(t *testing.T) {
	got := formatChainAmount("", "EUR")
	if want := "EUR 0,00"; got != want {
		t.Errorf("formatChainAmount = %q, want %q", got, want)
	}
}

// TestFormatChainAmount_GroupingBoundary locks down the exact digit count
// where a grouping dot appears — the classic off-by-one in grouping code.
func TestFormatChainAmount_GroupingBoundary(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"999.00", "EUR 999,00"},        // exactly three digits: no dot yet
		{"1000.00", "EUR 1.000,00"},     // fourth digit: dot appears
		{"1000000.00", "EUR 1.000.000,00"}, // two dots
	}
	for _, c := range cases {
		if got := formatChainAmount(c.raw, "EUR"); got != c.want {
			t.Errorf("formatChainAmount(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestFormatChainAmount_RoundsToTwoDecimals(t *testing.T) {
	got := formatChainAmount("12.005", "EUR")
	if want := "EUR 12,01"; got != want {
		t.Errorf("formatChainAmount = %q, want %q", got, want)
	}
}

// --- toDocumentChainWire ---

func TestToDocumentChainWire_MatchesTheFrontendContract(t *testing.T) {
	t.Parallel()

	number, date := "RE-2026-042", "2026-06-01"
	got, err := json.Marshal(toDocumentChainWire(&bizv1.DocumentChain{
		Id:         "11111111-1111-1111-1111-111111111111",
		Customer:   "Musterfirma GmbH",
		TotalValue: "1000.00",
		Currency:   "EUR",
		IsComplete: true,
		Nodes: []*bizv1.ChainNode{
			{Type: "invoice", Number: &number, Date: &date, Amount: "1000.00", Status: "completed"},
		},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"totalValue", "isComplete"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%s missing; got keys %v", key, keysOf(decoded))
		}
	}
	for _, key := range []string{"total_value", "is_complete"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("%s present — it is not part of the frontend contract", key)
		}
	}
	if decoded["totalValue"] != "EUR 1.000,00" {
		t.Errorf("totalValue = %v, want the formatted amount", decoded["totalValue"])
	}
}

// TestToDocumentChainWire_EmptyNumberAndDate_UseEmDash locks down the
// placeholder for the synthetic "outstanding balance" node the service emits
// with no number and no date (postgres_document_chains.go's trailing pending
// payment node).
func TestToDocumentChainWire_EmptyNumberAndDate_UseEmDash(t *testing.T) {
	wire := toDocumentChainWire(&bizv1.DocumentChain{
		Id:       "11111111-1111-1111-1111-111111111111",
		Currency: "EUR",
		Nodes: []*bizv1.ChainNode{
			{Type: "payment", Amount: "250.00", Status: "pending"},
		},
	})
	if len(wire.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(wire.Nodes))
	}
	if wire.Nodes[0].Number != "—" {
		t.Errorf("Number = %q, want the em-dash placeholder", wire.Nodes[0].Number)
	}
	if wire.Nodes[0].Date != "—" {
		t.Errorf("Date = %q, want the em-dash placeholder", wire.Nodes[0].Date)
	}
}

func TestToDocumentChainWire_NoNodes_ProducesEmptySliceNotNil(t *testing.T) {
	got, err := json.Marshal(toDocumentChainWire(&bizv1.DocumentChain{
		Id:       "11111111-1111-1111-1111-111111111111",
		Currency: "EUR",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	nodes, ok := decoded["nodes"].([]any)
	if !ok {
		t.Fatalf("nodes = %v (%T), want an array", decoded["nodes"], decoded["nodes"])
	}
	if len(nodes) != 0 {
		t.Errorf("len(nodes) = %d, want 0", len(nodes))
	}
}
