package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
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

// TestProto_DunningAndCreditNoteStatusIsString: the frontend types both statuses as strings
// and indexes a config table with them, so a number there crashes the panel. Other messages
// keep their numeric status.
func TestProto_DunningAndCreditNoteStatusIsString(t *testing.T) {
	status := func(msg proto.Message) any {
		t.Helper()
		b, err := response.MarshalProto(msg)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("not json: %v: %s", err, b)
		}
		return got["status"]
	}

	for number, want := range map[bizv1.DunningStatus]string{
		bizv1.DunningStatus_DUNNING_DRAFT: "draft",
		bizv1.DunningStatus_DUNNING_SENT:  "sent",
		bizv1.DunningStatus_DUNNING_PAID:  "paid",
	} {
		if got := status(&bizv1.DunningRecord{Id: "d", Status: number}); got != want {
			t.Errorf("dunning %v: status = %v, want %s", number, got, want)
		}
	}
	for number, want := range map[bizv1.CreditNoteStatus]string{
		bizv1.CreditNoteStatus_CREDIT_DRAFT: "draft",
		bizv1.CreditNoteStatus_CREDIT_SENT:  "sent",
	} {
		if got := status(&bizv1.CreditNote{Id: "c", Status: number}); got != want {
			t.Errorf("credit note %v: status = %v, want %s", number, got, want)
		}
	}
	// The desktop's RecordingStatus words; useActiveMeetingRecording compares 'active'.
	for number, want := range map[videov1.RecordingStatus]string{
		videov1.RecordingStatus_RECORDING_STATUS_ACTIVE:     "active",
		videov1.RecordingStatus_RECORDING_STATUS_PROCESSING: "processing",
		videov1.RecordingStatus_RECORDING_STATUS_COMPLETED:  "completed",
		videov1.RecordingStatus_RECORDING_STATUS_FAILED:     "failed",
		videov1.RecordingStatus_RECORDING_STATUS_DELETED:    "deleted",
		videov1.RecordingStatus_RECORDING_STATUS_PENDING:    "pending",
	} {
		if got := status(&videov1.Recording{Id: "r", Status: number}); got != want {
			t.Errorf("recording %v: status = %v, want %s", number, got, want)
		}
	}
	// An unknown number is left alone rather than named wrongly.
	if got := status(&bizv1.DunningRecord{Id: "d", Status: 9}); got != float64(9) {
		t.Errorf("unknown dunning status = %v, want 9", got)
	}
	// A message that is not listed keeps the enum number.
	if got := status(&bizv1.Invoice{Id: "i", Status: bizv1.InvoiceStatus_INVOICE_SENT}); got != float64(2) {
		t.Errorf("invoice status = %v, want 2", got)
	}
}

// ProtoFull keeps zero values, empty lists and unset optionals in the JSON, also inside
// nested messages; Proto drops them.
func TestProtoFull_KeepsZeroValuesAndUnsetOptionals(t *testing.T) {
	session := &inventarv1.InventurSession{Id: "s", Counts: []*inventarv1.InventurCount{
		{Id: "uncounted"},
		{Id: "zero", Counted: proto.Int64(0)},
	}}

	rec := httptest.NewRecorder()
	response.ProtoFull(rec, 200, session)
	var got struct {
		Counts []map[string]any `json:"counts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	if len(got.Counts) != 2 {
		t.Fatalf("counts = %v", got.Counts)
	}
	if v, ok := got.Counts[0]["counted"]; !ok || v != nil {
		t.Errorf("uncounted position: counted = %v (present %v), want null", v, ok)
	}
	if got.Counts[0]["expected"] != "0" {
		t.Errorf("expected = %v, want \"0\"", got.Counts[0]["expected"])
	}
	if got.Counts[1]["counted"] != "0" {
		t.Errorf("counted as zero = %v, want \"0\"", got.Counts[1]["counted"])
	}

	plain := httptest.NewRecorder()
	response.Proto(plain, 200, session)
	if strings.Contains(plain.Body.String(), `"expected"`) {
		t.Errorf("Proto must stay lean, got %s", plain.Body.String())
	}
}

// An empty list answers `[]`, not a missing member: protojson drops empty repeated fields and the
// frontend then met `{}` where it reads `.items.map`. Populated lists and nested messages stay as
// they were.
func TestProto_EmptyTopLevelListIsEmptyArray(t *testing.T) {
	empty := httptest.NewRecorder()
	response.Proto(empty, 200, &inventarv1.ListLocationsResponse{})
	var got map[string]json.RawMessage
	if err := json.Unmarshal(empty.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", empty.Body.String(), err)
	}
	if string(got["locations"]) != "[]" {
		t.Errorf("locations = %s in %s, want []", got["locations"], empty.Body.String())
	}

	filled := httptest.NewRecorder()
	response.Proto(filled, 200, &inventarv1.ListLocationsResponse{Locations: []*inventarv1.Location{{Id: "l1"}}})
	if !strings.Contains(filled.Body.String(), `"id":"l1"`) || strings.Contains(filled.Body.String(), `"locations":[]`) {
		t.Errorf("populated list changed: %s", filled.Body.String())
	}

	wrapped := httptest.NewRecorder()
	response.ProtoWrapped(wrapped, 200, "result", &inventarv1.ListLocationsResponse{})
	if !strings.Contains(wrapped.Body.String(), `"locations":[]`) {
		t.Errorf("wrapped empty list missing: %s", wrapped.Body.String())
	}
}
