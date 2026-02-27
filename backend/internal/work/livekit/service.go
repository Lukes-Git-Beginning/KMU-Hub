package livekit

import (
	"fmt"
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
	apiKey    string
	apiSecret string
	wsURL     string
	enabled   bool
}

// NewService creates a new LiveKit service.
// If any of apiKey, apiSecret, or wsURL are empty, the service is disabled.
func NewService(apiKey, apiSecret, wsURL string) *Service {
	return &Service{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		wsURL:     wsURL,
		enabled:   apiKey != "" && apiSecret != "" && wsURL != "",
	}
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

// GenerateJoinToken creates a JWT token for a user to join a LiveKit room.
// Returns ErrLiveKitNotConfigured if the service is not enabled.
// Token is valid for 24 hours with VideoGrant (RoomJoin + Room access).
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

	return at.ToJWT()
}
