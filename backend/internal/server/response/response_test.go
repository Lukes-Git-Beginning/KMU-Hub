package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// badInvoice cannot be marshaled: protojson rejects a string field that is not valid UTF-8.
func badInvoice() *bizv1.Invoice { return &bizv1.Invoice{Id: "i", Notes: "\xff"} }

func assertInternalError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	if got["error"] != "internal server error" {
		t.Errorf("error = %q, want the generic message (no marshal detail leaked)", got["error"])
	}
}

func TestJSON_WritesStatusHeaderAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	response.JSON(rec, http.StatusCreated, map[string]int{"n": 1})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	var got map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got["n"] != 1 {
		t.Errorf("body = %q (%v)", rec.Body.String(), err)
	}
}

func TestJSON_NilDataWritesNoBody(t *testing.T) {
	rec := httptest.NewRecorder()
	response.JSON(rec, http.StatusNoContent, nil)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestError_WrapsMessageUnderErrorKey(t *testing.T) {
	rec := httptest.NewRecorder()
	response.Error(rec, http.StatusConflict, "already exists")

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	if got["error"] != "already exists" || len(got) != 1 {
		t.Errorf("body = %v, want only {error: already exists}", got)
	}
}

func TestProtoWrapped_PutsEntityUnderKey(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoWrapped(rec, http.StatusCreated, "credit_note",
		&bizv1.CreditNote{Id: "c1", Status: bizv1.CreditNoteStatus_CREDIT_SENT})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	inner, ok := got["credit_note"]
	if !ok || len(got) != 1 {
		t.Fatalf("body = %s, want a single credit_note member", rec.Body.String())
	}
	if inner["id"] != "c1" {
		t.Errorf("id = %v", inner["id"])
	}
	// The wrapped entity goes through MarshalProto, so the status rewrite still applies.
	if inner["status"] != "sent" {
		t.Errorf("status = %v, want the string \"sent\"", inner["status"])
	}
}

func TestProtoWrapped_MarshalErrorIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoWrapped(rec, http.StatusOK, "invoice", badInvoice())
	assertInternalError(t, rec)
}

func TestProto_MarshalErrorIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	response.Proto(rec, http.StatusOK, badInvoice())
	assertInternalError(t, rec)

	rec = httptest.NewRecorder()
	response.ProtoFull(rec, http.StatusOK, badInvoice())
	assertInternalError(t, rec)
}

func TestProtoList_MarshalErrorIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoList(rec, http.StatusOK, []*bizv1.Invoice{{Id: "ok"}, badInvoice()})
	assertInternalError(t, rec)

	rec = httptest.NewRecorder()
	response.ProtoListWrapped(rec, http.StatusOK, "items", []*bizv1.Invoice{badInvoice()}, nil)
	assertInternalError(t, rec)
}

func TestProtoList_NilSliceIsEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoList(rec, http.StatusOK, []*bizv1.Invoice(nil))

	if got := rec.Body.String(); got != "[]\n" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestMarshalProto_ErrorsAreReturned(t *testing.T) {
	if _, err := response.MarshalProto(badInvoice()); err == nil {
		t.Error("MarshalProto swallowed the marshal error")
	}
	if _, err := response.MarshalProtoFull(badInvoice()); err == nil {
		t.Error("MarshalProtoFull swallowed the marshal error")
	}
}

func TestProtoFlatten_LiftsNestedMessageToTopLevel(t *testing.T) {
	msg := &calendarv1.ExpandedEventProto{
		Event:        &calendarv1.CalendarEventProto{Id: "e1", Title: "Standup"},
		CalendarName: "Team",
	}

	b, err := response.ProtoFlatten(msg, "event")
	if err != nil {
		t.Fatalf("ProtoFlatten: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	if _, nested := got["event"]; nested {
		t.Errorf("event key still present: %s", b)
	}
	if got["id"] != "e1" || got["title"] != "Standup" {
		t.Errorf("event fields not lifted: %s", b)
	}
	if got["calendar_name"] != "Team" {
		t.Errorf("outer field lost: %s", b)
	}
}

func TestProtoFlatten_MissingKeyKeepsOuterFields(t *testing.T) {
	msg := &bizv1.DunningRecord{Id: "d1", Status: bizv1.DunningStatus_DUNNING_SENT}

	b, err := response.ProtoFlatten(msg, "no_such_key")
	if err != nil {
		t.Fatalf("ProtoFlatten: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	if got["id"] != "d1" || got["status"] != "sent" {
		t.Errorf("body = %s", b)
	}
}

func TestProtoFlatten_MarshalErrorIsReturned(t *testing.T) {
	if _, err := response.ProtoFlatten(badInvoice(), "x"); err == nil {
		t.Error("ProtoFlatten swallowed the marshal error")
	}
}

// fillOptionalNulls must descend into nested messages: an unset `optional` scalar inside a
// nested message gets `null` just like one at the top level.
func TestProtoFull_NestedMessageGetsOptionalNulls(t *testing.T) {
	msg := &calendarv1.ExpandedEventProto{
		Event: &calendarv1.CalendarEventProto{Id: "e1"},
	}

	b, err := response.MarshalProtoFull(msg)
	if err != nil {
		t.Fatalf("MarshalProtoFull: %v", err)
	}
	var got struct {
		OriginalEventID json.RawMessage `json:"original_event_id"`
		Event           map[string]any  `json:"event"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	if string(got.OriginalEventID) != "null" {
		t.Errorf("top-level optional = %s, want null", got.OriginalEventID)
	}
	if v, ok := got.Event["description"]; !ok || v != nil {
		t.Errorf("nested optional description = %v (present %v), want null", v, ok)
	}
}

func TestMarshalProtoFull_SetOptionalIsNotOverwrittenByNull(t *testing.T) {
	msg := &calendarv1.ExpandedEventProto{
		Event:           &calendarv1.CalendarEventProto{Id: "e1", Description: proto.String("hi")},
		OriginalEventId: proto.String("orig"),
	}

	b, err := response.MarshalProtoFull(msg)
	if err != nil {
		t.Fatalf("MarshalProtoFull: %v", err)
	}
	var got struct {
		OriginalEventID string         `json:"original_event_id"`
		Event           map[string]any `json:"event"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	if got.OriginalEventID != "orig" || got.Event["description"] != "hi" {
		t.Errorf("set optionals lost: %s", b)
	}
}
