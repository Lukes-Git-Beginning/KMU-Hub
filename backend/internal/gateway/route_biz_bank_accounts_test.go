package gateway

// The bank account wire shape is hand-mapped rather than protojson-marshalled,
// so nothing but a test stands between it and the desktop client. These pin the
// places where the frontend contract (the BankAccount type in
// types/finance-types.ts, read through financeBankingApi) deliberately departs
// from this repo's protojson conventions.

import (
	"encoding/json"
	"testing"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func TestToBankAccountWire_MatchesTheFrontendContract(t *testing.T) {
	t.Parallel()

	lastSync := "2026-05-31"
	got, err := json.Marshal(toBankAccountWire(&bizv1.BankAccount{
		Id:        "11111111-1111-1111-1111-111111111111",
		BankName:  "Commerzbank",
		Iban:      "DE89370400440532013000",
		Bic:       "COBADEFFXXX",
		Balance:   "142850.75",
		Currency:  "EUR",
		Connected: true,
		LastSync:  &lastSync,
		// Internal bookkeeping the frontend never reads — it must not leak into
		// the response just because the proto carries it.
		TenantId: "22222222-2222-2222-2222-222222222222",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// bankName/lastSync are camelCase here where UseProtoNames would emit
	// bank_name and last_sync.
	for _, key := range []string{"bankName", "lastSync"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%s missing; got keys %v", key, keysOf(decoded))
		}
	}
	for _, key := range []string{"bank_name", "last_sync", "tenant_id", "tenantId"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("%s present — it is not part of the frontend contract", key)
		}
	}
	// balance is a JSON number where the proto carries a decimal string.
	if balance, ok := decoded["balance"].(float64); !ok || balance != 142850.75 {
		t.Errorf("balance = %v (%T), want 142850.75 as a number",
			decoded["balance"], decoded["balance"])
	}
	// The IBAN is stored canonically and grouped on the way out — the client
	// prints this straight onto a card.
	if decoded["iban"] != "DE89 3704 0044 0532 0130 00" {
		t.Errorf("iban = %v, want the grouped form", decoded["iban"])
	}
	if decoded["lastSync"] != lastSync {
		t.Errorf("lastSync = %v, want %q", decoded["lastSync"], lastSync)
	}
	if decoded["connected"] != true {
		t.Errorf("connected = %v, want true", decoded["connected"])
	}
}

// An account that was never synced must carry lastSync as an explicit null: the
// frontend types it string | null and renders "never synced" from that branch,
// so an omitted key would leave it reading undefined.
func TestToBankAccountWire_NeverSyncedKeepsANullLastSync(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(toBankAccountWire(&bizv1.BankAccount{
		Id:       "11111111-1111-1111-1111-111111111111",
		BankName: "Sparkasse München",
		Iban:     "DE72701500000012345678",
		Balance:  "0.00",
		Currency: "EUR",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	value, present := decoded["lastSync"]
	if !present {
		t.Fatalf("lastSync key missing; got keys %v", keysOf(decoded))
	}
	if value != nil {
		t.Errorf("lastSync = %v, want null", value)
	}
	if balance, ok := decoded["balance"].(float64); !ok || balance != 0 {
		t.Errorf("balance = %v, want 0", decoded["balance"])
	}
	if decoded["connected"] != false {
		t.Errorf("connected = %v, want false", decoded["connected"])
	}
}
