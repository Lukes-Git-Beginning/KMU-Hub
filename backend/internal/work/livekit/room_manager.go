package livekit

import (
	"context"
	"fmt"

	lkauth "github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// RoomManager implements the video.RoomManager interface using the LiveKit server SDK.
// When LiveKit is not configured, callers should pass nil (the video service handles nil gracefully).
type RoomManager struct {
	client    *lksdk.RoomServiceClient
	apiKey    string
	apiSecret string
}

// NewRoomManager creates a new LiveKit room manager.
// Returns nil if any required parameter is empty.
func NewRoomManager(apiKey, apiSecret, wsURL string) *RoomManager {
	if apiKey == "" || apiSecret == "" || wsURL == "" {
		return nil
	}
	client := lksdk.NewRoomServiceClient(wsURL, apiKey, apiSecret)
	return &RoomManager{
		client:    client,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// CreateRoom creates a new LiveKit room with the given name and max participants.
func (rm *RoomManager) CreateRoom(ctx context.Context, name string, maxParticipants uint32) error {
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
	_, err := rm.client.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{
		Room: name,
	})
	if err != nil {
		return fmt.Errorf("livekit delete room: %w", err)
	}
	return nil
}

// GenerateToken creates a JWT token for a user to join a specific LiveKit room.
func (rm *RoomManager) GenerateToken(roomName, userID, displayName string) (string, error) {
	at := lkauth.NewAccessToken(rm.apiKey, rm.apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.AddGrant(grant).
		SetIdentity(userID).
		SetName(displayName)

	return at.ToJWT()
}
