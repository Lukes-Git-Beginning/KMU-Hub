package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// TestProto_TimestampEnumAndFieldNames locks in the R3-P0-1 fix: proto messages
// must serialize google.protobuf.Timestamp as RFC3339 strings (not {seconds,nanos}),
// keep enums as integers (frontend types them as number) and use snake_case field
// names (matching the pb.go json tags). A plain encoding/json marshal would emit
// {"seconds":...,"nanos":...} and break every dialer timestamp in the UI.
func TestProto_TimestampEnumAndFieldNames(t *testing.T) {
	ts := time.Date(2026, 6, 20, 10, 30, 0, 0, time.UTC)
	camp := &dialerv1.Campaign{
		Name:      "Test",
		Status:    dialerv1.CampaignStatus(2),
		CreatedAt: timestamppb.New(ts),
	}

	rec := httptest.NewRecorder()
	response.Proto(rec, 200, camp)

	body := rec.Body.String()

	if !strings.Contains(body, `"created_at":"2026-06-20T10:30:00Z"`) {
		t.Errorf("expected RFC3339 created_at, got body: %s", body)
	}
	if strings.Contains(body, `"seconds"`) {
		t.Errorf("timestamp leaked {seconds,nanos} shape: %s", body)
	}
	if !strings.Contains(body, `"status":2`) {
		t.Errorf("expected integer enum status:2 (UseEnumNumbers), got body: %s", body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json content-type, got %q", ct)
	}

	// Ensure it is valid JSON.
	var sink map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Errorf("Proto output is not valid JSON: %v", err)
	}
}

// TestProtoListWrapped_EmptySliceIsEmptyArray locks in the document-routes
// wire-shape fix: a nil proto slice must serialize as `[]` under the given
// key, never `null` — the frontend types these endpoints as T[] and a null
// crashes any .map()/.length call on the wire-shape-tolerant client.
func TestProtoListWrapped_EmptySliceIsEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoListWrapped(rec, 200, "folders", []*dialerv1.Campaign(nil), map[string]any{"total": 0})

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("ProtoListWrapped output is not valid JSON: %v", err)
	}

	folders, ok := body["folders"].([]any)
	if !ok {
		t.Fatalf("expected folders to be a JSON array, got %T: %v", body["folders"], body["folders"])
	}
	if len(folders) != 0 {
		t.Errorf("expected empty array, got %d items", len(folders))
	}
	if total, _ := body["total"].(float64); total != 0 {
		t.Errorf("expected total 0, got %v", body["total"])
	}
}

// TestProtoListWrapped_ExtraFieldsAndItems verifies the key holds the
// protojson-encoded items and extra top-level fields survive alongside it.
func TestProtoListWrapped_ExtraFieldsAndItems(t *testing.T) {
	rec := httptest.NewRecorder()
	items := []*dialerv1.Campaign{{Name: "A"}, {Name: "B"}}
	response.ProtoListWrapped(rec, 200, "campaigns", items, map[string]any{"total": 2})

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("ProtoListWrapped output is not valid JSON: %v", err)
	}

	campaigns, ok := body["campaigns"].([]any)
	if !ok || len(campaigns) != 2 {
		t.Fatalf("expected 2 campaigns, got %v", body["campaigns"])
	}
	if total, _ := body["total"].(float64); total != 2 {
		t.Errorf("expected total 2, got %v", body["total"])
	}
}

// TestProto_TaxModeIsString: biz.v1.TaxMode goes out as the frontend's string union, in
// single entities, bare lists and wrapped lists, while a tax_mode that only appears inside
// a string value is left alone.
func TestProto_TaxModeIsString(t *testing.T) {
	inv := &bizv1.Invoice{Id: "i", TaxMode: bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER, Notes: `"tax_mode":2`}

	rec := httptest.NewRecorder()
	response.Proto(rec, 200, inv)
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("not json: %v: %s", err, rec.Body.String())
	}
	if got["tax_mode"] != "kleinunternehmer" {
		t.Errorf("tax_mode = %v, want kleinunternehmer", got["tax_mode"])
	}
	if got["notes"] != `"tax_mode":2` {
		t.Errorf("notes mangled: %v", got["notes"])
	}

	rec = httptest.NewRecorder()
	response.ProtoList(rec, 200, []*bizv1.Invoice{{Id: "a", TaxMode: bizv1.TaxMode_TAX_MODE_STANDARD}})
	if !strings.Contains(rec.Body.String(), `"tax_mode":"standard"`) {
		t.Errorf("list: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	response.ProtoListWrapped(rec, 200, "items", []*bizv1.Invoice{{Id: "a", TaxMode: bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE}}, nil)
	if !strings.Contains(rec.Body.String(), `"tax_mode":"reverse_charge"`) {
		t.Errorf("wrapped list: %s", rec.Body.String())
	}

	// Unspecified is omitted by protojson, so there is nothing to rewrite.
	b, err := response.MarshalProto(&bizv1.Invoice{Id: "i"})
	if err != nil || strings.Contains(string(b), "tax_mode") {
		t.Errorf("unspecified: %s %v", b, err)
	}
}
