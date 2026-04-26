package livekit

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 required by coturn REST API spec
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	lkauth "github.com/livekit/protocol/auth"
)

// ErrLiveKitNotConfigured is returned when LiveKit is not configured
var ErrLiveKitNotConfigured = fmt.Errorf("livekit is not configured: set LIVEKIT_API_KEY, LIVEKIT_API_SECRET, and LIVEKIT_WS_URL")

// Service provides LiveKit room and token management.
// Feature-flag-enabled: if API key/secret/URL are not configured, it gracefully
// returns errors from token generation but still allows room name/link formatting.
type Service struct {
	apiKey     string
	apiSecret  string
	wsURL      string
	enabled    bool
	turnSecret string // shared secret for coturn REST API auth (empty = TURN disabled)
	coturnHost string // FQDN of the coturn server, e.g. "turn.zentria.tech"
}

// NewService creates a new LiveKit service without TURN support.
// If any of apiKey, apiSecret, or wsURL are empty, the service is disabled.
func NewService(apiKey, apiSecret, wsURL string) *Service {
	return NewServiceWithTURN(apiKey, apiSecret, wsURL, "", "")
}

// NewServiceWithTURN creates a new LiveKit service with optional TURN support.
// turnSecret and coturnHost are optional: if either is empty, TURNIceServers
// returns nil and TURN relaying is disabled for token responses.
// When both are provided, per-session TURN credentials are embedded in the
// access token Metadata field as JSON so the LiveKit client SDK can apply them.
func NewServiceWithTURN(apiKey, apiSecret, wsURL, turnSecret, coturnHost string) *Service {
	svc := &Service{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		wsURL:      wsURL,
		enabled:    apiKey != "" && apiSecret != "" && wsURL != "",
		turnSecret: turnSecret,
		coturnHost: coturnHost,
	}
	if svc.IsTURNEnabled() {
		slog.Info("TURN configured", "host", coturnHost)
	} else {
		slog.Warn("TURN not configured — clients will rely on STUN only",
			"hint", "set TURN_SECRET and COTURN_HOST in env")
	}
	return svc
}

// IsTURNEnabled returns whether TURN relaying is configured.
func (s *Service) IsTURNEnabled() bool {
	return s.turnSecret != "" && s.coturnHost != ""
}

// IsEnabled returns whether LiveKit is configured and ready
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// GenerateRoomName creates a deterministic room name from an event ID.
// Format: "cal-{first 8 chars of eventID}"
// Works regardless of enabled state.
func (s *Service) GenerateRoomName(eventID uuid.UUID) string {
	idStr := eventID.String()
	return "cal-" + idStr[:8]
}

// GenerateMeetingLink creates a meeting URL for a given room name.
// Format: "{wsURL}/room/{roomName}"
// Works regardless of enabled state.
func (s *Service) GenerateMeetingLink(roomName string) string {
	return fmt.Sprintf("%s/room/%s", s.wsURL, roomName)
}

// turnMetadata is the JSON shape embedded in the LiveKit access token Metadata
// field so that the JavaScript client SDK can extract and apply TURN server
// configuration before opening the RTCPeerConnection.
//
// The LiveKit JS SDK (≥ 1.x) reads participant.metadata on connect and passes
// iceServers to the underlying RTCConfiguration automatically when the client
// calls Room.connect(). Clients on older SDK versions can read the metadata
// field manually and set RTCConfiguration.iceServers before connecting.
type turnMetadata struct {
	// IceServers follows the RTCIceServer dictionary shape so it can be spread
	// directly into RTCConfiguration.iceServers on the client side.
	IceServers []IceServer `json:"iceServers"`
}

// GenerateJoinToken creates a JWT token for a user to join a LiveKit room.
// Returns ErrLiveKitNotConfigured if the service is not enabled.
// Token is valid for 24 hours with VideoGrant (RoomJoin + Room access).
//
// When TURN is configured, per-session coturn credentials are embedded in the
// token Metadata field as JSON (key "iceServers"). The LiveKit client reads
// this and applies the TURN servers to the WebRTC peer connection, enabling
// relay for participants behind symmetric NAT.
func (s *Service) GenerateJoinToken(roomName, userID, displayName string) (string, error) {
	if !s.enabled {
		return "", ErrLiveKitNotConfigured
	}

	at := lkauth.NewAccessToken(s.apiKey, s.apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.SetVideoGrant(grant).
		SetIdentity(userID).
		SetName(displayName).
		SetValidFor(24 * time.Hour)

	// Embed per-session TURN credentials in token Metadata when coturn is configured.
	// The LiveKit JS SDK reads participant.metadata on connect; the client extracts
	// iceServers and applies them to RTCConfiguration before opening the peer connection.
	if servers := s.TURNIceServers(userID); len(servers) > 0 {
		meta := turnMetadata{IceServers: servers}
		metaJSON, err := json.Marshal(meta)
		if err == nil {
			at.SetMetadata(string(metaJSON))
		}
	}

	return at.ToJWT()
}

// IceServer represents a WebRTC ICE server configuration.
// Returned by TURNIceServers for clients that need explicit TURN credentials.
type IceServer struct {
	// URLs contains one or more TURN/STUN URIs.
	// Example: ["turns:turn.zentria.tech:5349", "turn:turn.zentria.tech:3478?transport=udp"]
	URLs []string `json:"urls"`
	// Username is the time-limited coturn username (format: "<ttl-unix-timestamp>:<userID>").
	Username string `json:"username"`
	// Credential is the HMAC-SHA1 of Username using the shared TURN secret.
	Credential string `json:"credential"`
}

// turnCredentialTTL is how long generated TURN credentials remain valid.
// 24 hours matches the LiveKit join token TTL so they expire together.
const turnCredentialTTL = 24 * time.Hour

// TURNIceServers generates time-limited TURN credentials using the coturn
// REST API shared-secret mechanism (RFC 5766 + coturn extension).
//
// Credential generation algorithm (coturn wiki):
//   username = "<unix-expiry-timestamp>:<arbitrary-userID>"
//   credential = base64(HMAC-SHA1(turnSecret, username))
//
// The returned IceServer entries include both TLS (turns:) and plain (turn:)
// URLs for maximum client compatibility. WebRTC clients prefer turns: when
// available.
//
// Returns nil if TURN is not configured (turnSecret or coturnHost empty).
// This is the zero-value fallback — callers must treat nil as "no TURN".
//
// Reference: https://github.com/coturn/coturn/wiki/turnserver#secret-based-authorization
func (s *Service) TURNIceServers(userID string) []IceServer {
	if !s.IsTURNEnabled() {
		return nil
	}

	ttl := time.Now().Add(turnCredentialTTL).Unix()
	username := strconv.FormatInt(ttl, 10) + ":" + userID

	//nolint:gosec // HMAC-SHA1 is mandated by the coturn REST API spec
	mac := hmac.New(sha1.New, []byte(s.turnSecret))
	mac.Write([]byte(username)) //nolint:errcheck // hmac.Write never returns an error
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return []IceServer{
		{
			// TURN over TLS — preferred by WebRTC in HTTPS contexts
			URLs:       []string{"turns:" + s.coturnHost + ":5349"},
			Username:   username,
			Credential: credential,
		},
		{
			// Plain TURN/UDP — fallback when TLS is blocked by firewall
			URLs:       []string{"turn:" + s.coturnHost + ":3478?transport=udp"},
			Username:   username,
			Credential: credential,
		},
	}
}
