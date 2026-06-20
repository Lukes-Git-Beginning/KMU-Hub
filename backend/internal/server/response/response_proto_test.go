package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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
