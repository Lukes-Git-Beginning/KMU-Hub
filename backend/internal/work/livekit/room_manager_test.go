package livekit

import (
	"strings"
	"testing"
)

// TestRoomManager_TURNIceServers verifies the adapter maps coturn credentials to
// the video-domain IceServerConfig shape the gateway exposes in join responses.
func TestRoomManager_TURNIceServers(t *testing.T) {
	rm := NewRoomManagerWithTURN("key", "secret", "wss://livekit.example.com", "turn-shared-secret", "turn.zentria.tech")
	if rm == nil {
		t.Fatal("NewRoomManagerWithTURN returned nil")
	}

	servers := rm.TURNIceServers("user-123")
	if len(servers) != 2 {
		t.Fatalf("expected 2 ICE servers (turns + turn/udp), got %d", len(servers))
	}
	for i, s := range servers {
		if len(s.URLs) == 0 {
			t.Errorf("server %d: empty URLs", i)
		}
		if s.Username == "" || s.Credential == "" {
			t.Errorf("server %d: missing username/credential", i)
		}
		if !strings.Contains(s.URLs[0], "turn.zentria.tech") {
			t.Errorf("server %d: URL does not reference the coturn host: %v", i, s.URLs)
		}
	}
}

// TestRoomManager_TURNIceServers_Disabled verifies that a manager without TURN
// configured returns nil (no relay), so callers fall back to STUN only.
func TestRoomManager_TURNIceServers_Disabled(t *testing.T) {
	rm := NewRoomManager("key", "secret", "wss://livekit.example.com")
	if rm == nil {
		t.Fatal("NewRoomManager returned nil")
	}
	if servers := rm.TURNIceServers("user-123"); servers != nil {
		t.Errorf("expected nil ICE servers when TURN is not configured, got %v", servers)
	}
}
