package livekit

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewService_Enabled(t *testing.T) {
	svc := NewService("key", "secret", "wss://livekit.example.com")
	if !svc.IsEnabled() {
		t.Error("expected IsEnabled = true")
	}
}

func TestNewService_Disabled_NoKey(t *testing.T) {
	svc := NewService("", "secret", "wss://livekit.example.com")
	if svc.IsEnabled() {
		t.Error("expected IsEnabled = false when apiKey is empty")
	}
}

func TestNewService_Disabled_NoSecret(t *testing.T) {
	svc := NewService("key", "", "wss://livekit.example.com")
	if svc.IsEnabled() {
		t.Error("expected IsEnabled = false when apiSecret is empty")
	}
}

func TestNewService_Disabled_NoURL(t *testing.T) {
	svc := NewService("key", "secret", "")
	if svc.IsEnabled() {
		t.Error("expected IsEnabled = false when wsURL is empty")
	}
}

func TestNewService_Disabled_AllEmpty(t *testing.T) {
	svc := NewService("", "", "")
	if svc.IsEnabled() {
		t.Error("expected IsEnabled = false when all empty")
	}
}

func TestGenerateRoomName(t *testing.T) {
	svc := NewService("", "", "")
	eventID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	name := svc.GenerateRoomName(eventID)
	if name != "cal-550e8400" {
		t.Errorf("room name = %q, want %q", name, "cal-550e8400")
	}
}

func TestGenerateRoomName_DifferentIDs(t *testing.T) {
	svc := NewService("", "", "")

	id1 := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	id2 := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	name1 := svc.GenerateRoomName(id1)
	name2 := svc.GenerateRoomName(id2)

	if name1 == name2 {
		t.Error("different event IDs should produce different room names")
	}
	if name1 != "cal-aaaaaaaa" {
		t.Errorf("name1 = %q, want %q", name1, "cal-aaaaaaaa")
	}
	if name2 != "cal-11111111" {
		t.Errorf("name2 = %q, want %q", name2, "cal-11111111")
	}
}

func TestGenerateMeetingLink(t *testing.T) {
	svc := NewService("", "", "wss://livekit.example.com")

	link := svc.GenerateMeetingLink("cal-550e8400")
	expected := "wss://livekit.example.com/room/cal-550e8400"
	if link != expected {
		t.Errorf("link = %q, want %q", link, expected)
	}
}

func TestGenerateMeetingLink_EmptyURL(t *testing.T) {
	svc := NewService("", "", "")

	link := svc.GenerateMeetingLink("cal-550e8400")
	expected := "/room/cal-550e8400"
	if link != expected {
		t.Errorf("link = %q, want %q", link, expected)
	}
}

func TestGenerateJoinToken_Enabled(t *testing.T) {
	svc := NewService("test-api-key", "test-api-secret-that-is-long-enough", "wss://livekit.example.com")

	token, err := svc.GenerateJoinToken("cal-550e8400", "user-123", "Max Mustermann")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	// Verify it's a valid JWT (3 parts separated by dots)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT parts = %d, want 3", len(parts))
	}

	// Decode the header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var header map[string]any
	if jsonErr := json.Unmarshal(headerJSON, &header); jsonErr != nil {
		t.Fatalf("parse header: %v", jsonErr)
	}
	if header["typ"] != "JWT" {
		t.Errorf("header typ = %v, want JWT", header["typ"])
	}

	// Decode the payload and verify key fields
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var payload map[string]any
	if jsonErr := json.Unmarshal(payloadJSON, &payload); jsonErr != nil {
		t.Fatalf("parse payload: %v", jsonErr)
	}

	// Verify identity (sub field in LiveKit tokens)
	if sub, ok := payload["sub"].(string); !ok || sub != "user-123" {
		t.Errorf("sub = %v, want %q", payload["sub"], "user-123")
	}

	// Verify issuer matches API key
	if iss, ok := payload["iss"].(string); !ok || iss != "test-api-key" {
		t.Errorf("iss = %v, want %q", payload["iss"], "test-api-key")
	}

	// Verify video grant exists
	video, ok := payload["video"].(map[string]any)
	if !ok {
		t.Fatal("missing video grant in payload")
	}
	if room, ok := video["room"].(string); !ok || room != "cal-550e8400" {
		t.Errorf("video.room = %v, want %q", video["room"], "cal-550e8400")
	}
	if roomJoin, ok := video["roomJoin"].(bool); !ok || !roomJoin {
		t.Errorf("video.roomJoin = %v, want true", video["roomJoin"])
	}
}

func TestGenerateJoinToken_Disabled(t *testing.T) {
	svc := NewService("", "", "")

	_, err := svc.GenerateJoinToken("cal-550e8400", "user-123", "Max Mustermann")
	if err != ErrLiveKitNotConfigured {
		t.Errorf("err = %v, want ErrLiveKitNotConfigured", err)
	}
}

func TestGenerateJoinToken_PartialConfig(t *testing.T) {
	svc := NewService("key", "", "wss://livekit.example.com")

	_, err := svc.GenerateJoinToken("cal-550e8400", "user-123", "Max Mustermann")
	if err != ErrLiveKitNotConfigured {
		t.Errorf("err = %v, want ErrLiveKitNotConfigured", err)
	}
}
