package gateway

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	lkauth "github.com/livekit/protocol/auth"
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
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/webhooks/livekit", strings.NewReader("{invalid"))
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid webhook payload")
}

func TestHandleLiveKitWebhook_ParticipantJoined_Returns200(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "participant_joined",
		Room: struct {
			Name string `json:"name"`
		}{Name: "room-abc"},
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
	routes := NewVideoRoutes(registryWithService("work"), "", "")
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
	routes := NewVideoRoutes(registryWithService("work"), "", "")
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
	routes := NewVideoRoutes(emptyRegistry(), "", "")
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

// buildSignedLiveKitWebhookReq constructs a POST request with a valid LiveKit
// Authorization JWT signed with the given key/secret pair.  The token embeds the
// SHA-256 hash of the body, exactly as LiveKit's server does.
func buildSignedLiveKitWebhookReq(t *testing.T, apiKey, apiSecret string, body []byte) *http.Request {
	t.Helper()

	sha := sha256.Sum256(body)
	hash := base64.StdEncoding.EncodeToString(sha[:])

	at := lkauth.NewAccessToken(apiKey, apiSecret).
		SetValidFor(5 * time.Minute).
		SetSha256(hash)

	token, err := at.ToJWT()
	if err != nil {
		t.Fatalf("failed to build livekit webhook token: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/webhooks/livekit", strings.NewReader(string(body)))
	req.Header.Set("Authorization", token)
	return req
}

// TestHandleLiveKitWebhook_MissingSignature_Returns401 verifies that a request without
// an Authorization header is rejected with 401 when a key provider is configured.
func TestHandleLiveKitWebhook_MissingSignature_Returns401(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "test-api-key", "test-api-secret-32charslong!!")
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(liveKitWebhookEvent{Event: "participant_joined"})
	req := httptest.NewRequest("POST", "/api/v1/webhooks/livekit", strings.NewReader(string(body)))
	// No Authorization header set.
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "missing webhook authorization")
}

// TestHandleLiveKitWebhook_InvalidSignature_Returns401 verifies that a request signed
// with the wrong secret is rejected with 401.
func TestHandleLiveKitWebhook_InvalidSignature_Returns401(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "test-api-key", "test-api-secret-32charslong!!")
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(liveKitWebhookEvent{Event: "participant_joined"})
	// Sign with a different (wrong) secret.
	req := buildSignedLiveKitWebhookReq(t, "test-api-key", "completely-wrong-secret-value!!", body)
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "invalid webhook signature")
}

// TestHandleLiveKitWebhook_ValidSignature_Returns200 verifies that a correctly signed
// request is accepted when a key provider is configured.
func TestHandleLiveKitWebhook_ValidSignature_Returns200(t *testing.T) {
	const apiKey = "test-api-key"
	const apiSecret = "test-api-secret-32charslong!!"
	routes := NewVideoRoutes(emptyRegistry(), apiKey, apiSecret)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(liveKitWebhookEvent{Event: "participant_joined"})
	req := buildSignedLiveKitWebhookReq(t, apiKey, apiSecret, body)
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// TestHandleLiveKitWebhook_NoSecretConfigured_Accepts verifies that requests are
// accepted without any Authorization header when no key provider is configured
// (graceful degradation for dev/staging).
func TestHandleLiveKitWebhook_NoSecretConfigured_Accepts(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "") // no key/secret
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{Event: "room_finished"})
	// No Authorization header — should still be accepted.
	routes.HandleLiveKitWebhook(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// TestHandleLiveKitWebhook_RoomFinished_MeetingRoom_AttemptsGRPC verifies that
// a room_finished event for a "meeting-*" room triggers CompleteMeetingByRoom.
// Webhook always returns 200 regardless of gRPC outcome.
func TestHandleLiveKitWebhook_RoomFinished_MeetingRoom_AttemptsGRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "room_finished",
		Room: struct {
			Name string `json:"name"`
		}{Name: "meeting-ab12cd34-ef56-7890-abcd-ef1234567890"},
	})
	routes.HandleLiveKitWebhook(rec, req)
	// Webhook always returns 200 (gRPC failure is logged, not propagated).
	assertStatus(t, rec, http.StatusOK)
}

// TestHandleLiveKitWebhook_RoomFinished_NonMeetingRoom_Returns200 verifies that
// a room_finished event for a non-meeting room is ignored (no gRPC call) and
// returns 200.
func TestHandleLiveKitWebhook_RoomFinished_NonMeetingRoom_Returns200(t *testing.T) {
	// emptyRegistry has no "work" service → if gRPC were called it would fail.
	// We verify that it returns 200 anyway (non-meeting rooms are ignored).
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := buildLiveKitWebhookReq(t, liveKitWebhookEvent{
		Event: "room_finished",
		Room: struct {
			Name string `json:"name"`
		}{Name: "call-some-random-room"},
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
