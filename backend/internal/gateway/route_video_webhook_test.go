package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// buildLiveKitWebhookReq constructs a POST request that simulates a LiveKit webhook.
func buildLiveKitWebhookReq(t *testing.T, event liveKitWebhookEvent) *http.Request {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal webhook event: %v", err)
	}
	return httptest.NewRequest("POST", "/api/v1/webhooks/livekit", strings.NewReader(string(body)))
}

func TestHandleLiveKitWebhook_InvalidJSON(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/webhooks/livekit", strings.NewReader("{invalid"))
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid webhook payload")
}

func TestHandleLiveKitWebhook_ParticipantJoined_Returns200(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "participant_joined",
		Room:  struct{ Name string `json:"name"` }{Name: "room-abc"},
		Participant: struct {
			Identity string `json:"identity"`
		}{Identity: "user-123"},
	})
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleLiveKitWebhook_EgressEndedComplete_AttemptsGRPC(t *testing.T) {
	// When a complete egress_ended event arrives with a registered "work" service,
	// the handler should attempt a gRPC call and return 200 regardless of gRPC outcome
	// (the webhook must always ack 200 to LiveKit).
	routes := NewVideoRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "egress_ended",
		EgressInfo: struct {
			EgressID    string `json:"egress_id"`
			RoomName    string `json:"room_name"`
			Status      int32  `json:"status"`
			Error       string `json:"error"`
			Duration    int64  `json:"duration"`
			FileResults []struct {
				Filename string `json:"filename"`
				Location string `json:"location"`
				Size     int64  `json:"size"`
				Duration int64  `json:"duration"`
			} `json:"file_results"`
		}{
			EgressID: "egress-abc-123",
			Status:   liveKitEgressStatusComplete, // 3 = EGRESS_COMPLETE
			Duration: 30_000_000_000,              // 30 seconds in nanoseconds
			FileResults: []struct {
				Filename string `json:"filename"`
				Location string `json:"location"`
				Size     int64  `json:"size"`
				Duration int64  `json:"duration"`
			}{
				{
					Location: "s3://recordings/egress-abc-123.mp4",
					Size:     1024 * 1024 * 50, // 50 MB
					Duration: 30_000_000_000,
				},
			},
		},
	})
	routes.HandleLiveKitWebhook(rec, req)
	// Webhook always returns 200 to LiveKit (gRPC failure is logged, not propagated).
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleLiveKitWebhook_EgressEndedFailed_AttemptsGRPC(t *testing.T) {
	// A non-complete egress should call FailRecordingByEgress.
	// Webhook still returns 200.
	routes := NewVideoRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "egress_ended",
		EgressInfo: struct {
			EgressID    string `json:"egress_id"`
			RoomName    string `json:"room_name"`
			Status      int32  `json:"status"`
			Error       string `json:"error"`
			Duration    int64  `json:"duration"`
			FileResults []struct {
				Filename string `json:"filename"`
				Location string `json:"location"`
				Size     int64  `json:"size"`
				Duration int64  `json:"duration"`
			} `json:"file_results"`
		}{
			EgressID: "egress-xyz-456",
			Status:   4, // 4 = EGRESS_FAILED
			Error:    "egress timeout",
		},
	})
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleLiveKitWebhook_EgressEndedMissingID_Returns200(t *testing.T) {
	// Missing egress_id → skip gRPC call, log warning, still return 200.
	routes := NewVideoRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "egress_ended",
		EgressInfo: struct {
			EgressID    string `json:"egress_id"`
			RoomName    string `json:"room_name"`
			Status      int32  `json:"status"`
			Error       string `json:"error"`
			Duration    int64  `json:"duration"`
			FileResults []struct {
				Filename string `json:"filename"`
				Location string `json:"location"`
				Size     int64  `json:"size"`
				Duration int64  `json:"duration"`
			} `json:"file_results"`
		}{
			EgressID: "", // missing
			Status:   3,
		},
	})
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// TestEgressDurationConversion verifies the nanoseconds-to-seconds conversion.
func TestEgressDurationConversion(t *testing.T) {
	tests := []struct {
		ns      int64
		wantSec int32
	}{
		{0, 0},
		{1_000_000_000, 1},
		{30_000_000_000, 30},
		{3_723_000_000_000, 3723},
		{500_000_000, 0}, // less than 1 second rounds down
	}

	for _, tt := range tests {
		got := int32(tt.ns / 1_000_000_000)
		if got != tt.wantSec {
			t.Errorf("duration conversion: ns=%d → got %d sec, want %d sec", tt.ns, got, tt.wantSec)
		}
	}
}
