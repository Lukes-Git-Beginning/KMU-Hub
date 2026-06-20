package livekit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	lkauth "github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/kmuhub/kmuhub/internal/work/video"
)

// livekitAPITimeout bounds each LiveKit server API call so a hung control-plane
// connection cannot block the caller indefinitely when the caller passes a
// context without its own deadline.
const livekitAPITimeout = 10 * time.Second

// RoomManager implements the video.RoomManager interface using the LiveKit server SDK.
// When LiveKit is not configured, callers should pass nil (the video service handles nil gracefully).
type RoomManager struct {
	client     *lksdk.RoomServiceClient
	apiKey     string
	apiSecret  string
	turnSecret string // coturn shared secret; empty = TURN disabled
	coturnHost string // FQDN of coturn server, e.g. "turn.zentria.tech"
}

// NewRoomManager creates a new LiveKit room manager without TURN support.
// Returns nil if any required parameter is empty.
func NewRoomManager(apiKey, apiSecret, wsURL string) *RoomManager {
	return NewRoomManagerWithTURN(apiKey, apiSecret, wsURL, "", "")
}

// NewRoomManagerWithTURN creates a new LiveKit room manager with optional TURN support.
// turnSecret and coturnHost are both required for TURN to be active; either empty disables it.
// Returns nil if any of apiKey, apiSecret, or wsURL is empty.
func NewRoomManagerWithTURN(apiKey, apiSecret, wsURL, turnSecret, coturnHost string) *RoomManager {
	if apiKey == "" || apiSecret == "" || wsURL == "" {
		return nil
	}
	client := lksdk.NewRoomServiceClient(wsURL, apiKey, apiSecret)
	return &RoomManager{
		client:     client,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		turnSecret: turnSecret,
		coturnHost: coturnHost,
	}
}

// CreateRoom creates a new LiveKit room with the given name and max participants.
func (rm *RoomManager) CreateRoom(ctx context.Context, name string, maxParticipants uint32) error {
	ctx, cancel := context.WithTimeout(ctx, livekitAPITimeout)
	defer cancel()
	_, err := rm.client.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:            name,
		MaxParticipants: maxParticipants,
	})
	if err != nil {
		return fmt.Errorf("livekit create room: %w", err)
	}
	return nil
}

// DeleteRoom deletes a LiveKit room by name.
func (rm *RoomManager) DeleteRoom(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, livekitAPITimeout)
	defer cancel()
	_, err := rm.client.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{
		Room: name,
	})
	if err != nil {
		return fmt.Errorf("livekit delete room: %w", err)
	}
	return nil
}

// GenerateToken creates a JWT token for a user to join a specific LiveKit room.
// When TURN is configured on this RoomManager, per-session coturn credentials
// are embedded in the token Metadata field as JSON so that the LiveKit client
// SDK can apply them to the WebRTC peer connection automatically.
func (rm *RoomManager) GenerateToken(roomName, userID, displayName string) (string, error) {
	at := lkauth.NewAccessToken(rm.apiKey, rm.apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.SetVideoGrant(grant).
		SetIdentity(userID).
		SetName(displayName)

	// Embed per-session TURN credentials when coturn is configured.
	if rm.turnSecret != "" && rm.coturnHost != "" {
		svc := &Service{turnSecret: rm.turnSecret, coturnHost: rm.coturnHost}
		if servers := svc.TURNIceServers(userID); len(servers) > 0 {
			meta := turnMetadata{IceServers: servers}
			metaJSON, err := json.Marshal(meta)
			if err == nil {
				at.SetMetadata(string(metaJSON))
			}
		}
	}

	return at.ToJWT()
}

// TURNIceServers returns per-session coturn credentials for the user as
// video.IceServerConfig, or nil when TURN is not configured on this manager.
// This lets the gateway expose iceServers in the join response so clients can
// set RTCConfiguration before connecting (the token-embedded metadata alone is
// not applied by the declarative LiveKit client before connect).
func (rm *RoomManager) TURNIceServers(userID string) []video.IceServerConfig {
	if rm.turnSecret == "" || rm.coturnHost == "" {
		return nil
	}
	svc := &Service{turnSecret: rm.turnSecret, coturnHost: rm.coturnHost}
	servers := svc.TURNIceServers(userID)
	if len(servers) == 0 {
		return nil
	}
	out := make([]video.IceServerConfig, 0, len(servers))
	for _, s := range servers {
		out = append(out, video.IceServerConfig{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: s.Credential,
		})
	}
	return out
}
