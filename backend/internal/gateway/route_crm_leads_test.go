package gateway

// The lead wire shape is hand-mapped rather than protojson-marshalled, so
// nothing but a test stands between it and the desktop inbox (the Lead type in
// api/hooks/useLeads.ts). That type is camelCase throughout, which is where it
// departs from this repo's protojson conventions -- and temperatureOverride
// has to stay absent unless a user actually pinned it, because the inbox reads
// its presence as "manual override in effect".

import (
	"encoding/json"
	"testing"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

func TestToLeadWire_CamelCaseKeys(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(toLeadWire(&crmv1.LeadInfo{
		Id:             "33333333-3333-3333-3333-333333333333",
		FirstName:      "Sabine",
		LastName:       "Vogel",
		Company:        "Vogel Dachtechnik GmbH",
		Email:          "sabine.vogel@vogel-dach.de",
		Phone:          "+49 711 4455660",
		Source:         "dialer",
		Score:          90,
		Temperature:    "hot",
		Status:         "new",
		LifecycleStage: "lead",
		Notes:          "Rückruf gewünscht",
		CreatedAt:      "2026-08-01T09:15:00Z",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"firstName", "lastName", "lifecycleStage", "createdAt"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%s missing; got keys %v", key, keysOf(decoded))
		}
	}
	for _, key := range []string{"first_name", "last_name", "lifecycle_stage", "created_at"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("%s present — the inbox reads camelCase", key)
		}
	}
	if score, ok := decoded["score"].(float64); !ok || score != 90 {
		t.Errorf("score = %v (%T), want 90 as a number", decoded["score"], decoded["score"])
	}
	if _, ok := decoded["temperatureOverride"]; ok {
		t.Error("temperatureOverride present without a manual override — the inbox reads that as pinned")
	}
}

func TestToLeadWire_KeepsTemperatureOverrideWhenPinned(t *testing.T) {
	t.Parallel()

	override := "cold"
	wire := toLeadWire(&crmv1.LeadInfo{
		Id:                  "44444444-4444-4444-4444-444444444444",
		Score:               90,
		Temperature:         "cold", // the override wins over the score
		TemperatureOverride: &override,
	})

	if wire.TemperatureOverride == nil || *wire.TemperatureOverride != "cold" {
		t.Fatalf("temperatureOverride dropped: %v", wire.TemperatureOverride)
	}
	if wire.Temperature != "cold" {
		t.Fatalf("temperature should follow the override, got %q", wire.Temperature)
	}
}
