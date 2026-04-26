package livekit

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 required by coturn REST API spec
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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

// ---------------------------------------------------------------------------
// TURN / coturn credential tests
// ---------------------------------------------------------------------------

func TestIsTURNEnabled_BothSet(t *testing.T) {
	svc := NewServiceWithTURN("key", "secret", "wss://livekit.example.com", "turnsecret", "turn.zentria.tech")
	if !svc.IsTURNEnabled() {
		t.Error("expected IsTURNEnabled = true when both turnSecret and coturnHost are set")
	}
}

func TestIsTURNEnabled_NoSecret(t *testing.T) {
	svc := NewServiceWithTURN("key", "secret", "wss://livekit.example.com", "", "turn.zentria.tech")
	if svc.IsTURNEnabled() {
		t.Error("expected IsTURNEnabled = false when turnSecret is empty")
	}
}

func TestIsTURNEnabled_NoHost(t *testing.T) {
	svc := NewServiceWithTURN("key", "secret", "wss://livekit.example.com", "turnsecret", "")
	if svc.IsTURNEnabled() {
		t.Error("expected IsTURNEnabled = false when coturnHost is empty")
	}
}

func TestTURNIceServers_Disabled(t *testing.T) {
	// NewService (no TURN params) must return nil — zero-value fallback
	svc := NewService("key", "secret", "wss://livekit.example.com")
	servers := svc.TURNIceServers("user-123")
	if servers != nil {
		t.Errorf("expected nil IceServers when TURN disabled, got %v", servers)
	}
}

func TestTURNIceServers_ReturnsCredentials(t *testing.T) {
	const turnSecret = "test-turn-shared-secret"
	const coturnHost = "turn.zentria.tech"
	const userID = "user-123"

	svc := NewServiceWithTURN("key", "secret", "wss://livekit.example.com", turnSecret, coturnHost)
	servers := svc.TURNIceServers(userID)

	if len(servers) != 2 {
		t.Fatalf("expected 2 ICE servers (TLS + plain), got %d", len(servers))
	}

	// Both entries must have the same username and credential
	tls := servers[0]
	plain := servers[1]

	if tls.Username == "" {
		t.Error("TLS ICE server username is empty")
	}
	if tls.Credential == "" {
		t.Error("TLS ICE server credential is empty")
	}
	if tls.Username != plain.Username {
		t.Errorf("username mismatch: TLS=%q plain=%q", tls.Username, plain.Username)
	}
	if tls.Credential != plain.Credential {
		t.Errorf("credential mismatch: TLS=%q plain=%q", tls.Credential, plain.Credential)
	}

	// Username must contain the userID suffix
	if !strings.HasSuffix(tls.Username, ":"+userID) {
		t.Errorf("username %q does not end with :%s", tls.Username, userID)
	}

	// URL format checks
	if len(tls.URLs) != 1 || !strings.HasPrefix(tls.URLs[0], "turns:") {
		t.Errorf("TLS ICE server URL should start with 'turns:', got %v", tls.URLs)
	}
	if len(plain.URLs) != 1 || !strings.HasPrefix(plain.URLs[0], "turn:") {
		t.Errorf("plain ICE server URL should start with 'turn:', got %v", plain.URLs)
	}

	// Validate HMAC-SHA1 credential against the known secret
	//nolint:gosec
	mac := hmac.New(sha1.New, []byte(turnSecret))
	mac.Write([]byte(tls.Username)) //nolint:errcheck
	expectedCred := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if tls.Credential != expectedCred {
		t.Errorf("credential HMAC mismatch: got %q, want %q", tls.Credential, expectedCred)
	}
}

func TestTURNIceServers_CredentialsExpireInFuture(t *testing.T) {
	svc := NewServiceWithTURN("key", "secret", "wss://livekit.example.com", "secret", "turn.zentria.tech")
	servers := svc.TURNIceServers("user-123")

	if len(servers) == 0 {
		t.Fatal("expected ICE servers")
	}

	// Parse the unix timestamp from username (format: "<timestamp>:<userID>")
	parts := strings.SplitN(servers[0].Username, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected username format: %q", servers[0].Username)
	}

	var expiry int64
	if _, err := fmt.Sscanf(parts[0], "%d", &expiry); err != nil {
		t.Fatalf("cannot parse expiry timestamp from %q: %v", parts[0], err)
	}

	now := time.Now().Unix()
	if expiry <= now {
		t.Errorf("TURN credential expiry %d is in the past (now=%d)", expiry, now)
	}
	// Should be ~24h in the future (allow 60s slack)
	minExpiry := now + int64(23*time.Hour/time.Second)
	if expiry < minExpiry {
		t.Errorf("TURN credential expires too soon: expiry=%d, want >=%d (now+23h)", expiry, minExpiry)
	}
}

func TestNewService_BackwardCompat(t *testing.T) {
	// NewService must still work exactly as before (no TURN params)
	svc := NewService("key", "secret", "wss://livekit.example.com")
	if !svc.IsEnabled() {
		t.Error("expected IsEnabled = true")
	}
	if svc.IsTURNEnabled() {
		t.Error("expected IsTURNEnabled = false when created via NewService")
	}
	if svc.TURNIceServers("u") != nil {
		t.Error("expected nil TURNIceServers when created via NewService")
	}
}

// ---------------------------------------------------------------------------
// GenerateJoinToken + TURN Metadata embedding tests
// ---------------------------------------------------------------------------

// decodeTokenPayload decodes the middle (payload) section of a JWT and returns
// the raw JSON as a map. The JWT must be compact-serialized (3 dot-separated parts).
func decodeTokenPayload(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT has %d parts, want 3", len(parts))
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("base64-decode payload: %v", err)
	}
	var payload map[string]any
	if jsonErr := json.Unmarshal(payloadJSON, &payload); jsonErr != nil {
		t.Fatalf("json-decode payload: %v", jsonErr)
	}
	return payload
}

func TestGenerateJoinToken_NoTURN_NoMetadata(t *testing.T) {
	// Without TURN config the token must not carry a metadata field (backward compat).
	svc := NewService("test-api-key", "test-api-secret-that-is-long-enough", "wss://livekit.example.com")

	token, err := svc.GenerateJoinToken("cal-550e8400", "user-123", "Max Mustermann")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := decodeTokenPayload(t, token)
	if md, ok := payload["metadata"]; ok && md != "" {
		t.Errorf("expected no metadata when TURN is not configured, got %v", md)
	}
}

func TestGenerateJoinToken_WithTURN_EmbedsTURNMetadata(t *testing.T) {
	const turnSecret = "test-turn-shared-secret"
	const coturnHost = "turn.zentria.tech"
	const userID = "user-456"

	svc := NewServiceWithTURN(
		"test-api-key", "test-api-secret-that-is-long-enough",
		"wss://livekit.example.com",
		turnSecret, coturnHost,
	)

	token, err := svc.GenerateJoinToken("cal-room1", userID, "Test User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := decodeTokenPayload(t, token)

	// metadata must be a non-empty string
	mdRaw, ok := payload["metadata"].(string)
	if !ok || mdRaw == "" {
		t.Fatalf("expected metadata string in token, got %v", payload["metadata"])
	}

	// metadata must be valid JSON with an iceServers array
	var meta map[string]any
	if err := json.Unmarshal([]byte(mdRaw), &meta); err != nil {
		t.Fatalf("metadata is not valid JSON: %v — raw: %s", err, mdRaw)
	}

	iceRaw, ok := meta["iceServers"]
	if !ok {
		t.Fatal("metadata JSON missing 'iceServers' key")
	}

	iceServers, ok := iceRaw.([]any)
	if !ok || len(iceServers) == 0 {
		t.Fatalf("iceServers must be a non-empty array, got %T %v", iceRaw, iceRaw)
	}

	// Each ICE server entry must have urls, username, credential
	for i, entry := range iceServers {
		srv, ok := entry.(map[string]any)
		if !ok {
			t.Errorf("iceServers[%d] is not an object: %T", i, entry)
			continue
		}
		if _, ok := srv["urls"]; !ok {
			t.Errorf("iceServers[%d] missing 'urls'", i)
		}
		username, _ := srv["username"].(string)
		if username == "" {
			t.Errorf("iceServers[%d] missing 'username'", i)
		}
		credential, _ := srv["credential"].(string)
		if credential == "" {
			t.Errorf("iceServers[%d] missing 'credential'", i)
		}

		// Validate HMAC-SHA1 credential matches the known secret
		//nolint:gosec
		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(username)) //nolint:errcheck
		expectedCred := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if credential != expectedCred {
			t.Errorf("iceServers[%d] credential HMAC mismatch: got %q, want %q", i, credential, expectedCred)
		}

		// Username must end with the participant userID
		if !strings.HasSuffix(username, ":"+userID) {
			t.Errorf("iceServers[%d] username %q does not end with :%s", i, username, userID)
		}
	}
}

func TestGenerateJoinToken_WithTURN_TokenIsValidJWT(t *testing.T) {
	// Full round-trip: token with TURN metadata must still be a valid 3-part JWT.
	svc := NewServiceWithTURN(
		"test-api-key", "test-api-secret-that-is-long-enough",
		"wss://livekit.example.com",
		"turnsecret", "turn.zentria.tech",
	)

	token, err := svc.GenerateJoinToken("cal-room2", "user-789", "Another User")
	if err != nil {
		t.Fatalf("unexpected error generating JWT with TURN metadata: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT has %d parts, want 3", len(parts))
	}

	payload := decodeTokenPayload(t, token)

	// Verify core JWT claims survive alongside the TURN metadata
	if sub, ok := payload["sub"].(string); !ok || sub != "user-789" {
		t.Errorf("sub = %v, want %q", payload["sub"], "user-789")
	}
	if iss, ok := payload["iss"].(string); !ok || iss != "test-api-key" {
		t.Errorf("iss = %v, want %q", payload["iss"], "test-api-key")
	}
}
