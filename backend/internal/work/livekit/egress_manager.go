package livekit

import (
	"context"
	"fmt"

	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/kmuhub/kmuhub/internal/work/recording"
)

// EgressManager implements the recording.EgressManager interface using the LiveKit Egress SDK.
// When LiveKit Egress is not configured, callers should pass nil (the recording service handles nil gracefully).
type EgressManager struct {
	client *lksdk.EgressClient
}

// NewEgressManager creates a new LiveKit egress manager.
// Returns nil if any required parameter is empty.
func NewEgressManager(apiKey, apiSecret, wsURL string) *EgressManager {
	if apiKey == "" || apiSecret == "" || wsURL == "" {
		return nil
	}
	client := lksdk.NewEgressClient(wsURL, apiKey, apiSecret)
	return &EgressManager{
		client: client,
	}
}

// StartRoomCompositeEgress starts a room composite recording to S3.
func (em *EgressManager) StartRoomCompositeEgress(ctx context.Context, roomName, templateURL string, s3Config recording.S3Config) (string, error) {
	req := &lkproto.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   templateURL,
		Output: &lkproto.RoomCompositeEgressRequest_File{
			File: &lkproto.EncodedFileOutput{
				FileType: lkproto.EncodedFileType_MP4,
				Output: &lkproto.EncodedFileOutput_S3{
					S3: &lkproto.S3Upload{
						AccessKey:      s3Config.AccessKey,
						Secret:         s3Config.Secret,
						Endpoint:       s3Config.Endpoint,
						Bucket:         s3Config.Bucket,
						ForcePathStyle: true,
					},
				},
			},
		},
	}

	info, err := em.client.StartRoomCompositeEgress(ctx, req)
	if err != nil {
		return "", fmt.Errorf("livekit start egress: %w", err)
	}

	return info.EgressId, nil
}

// StopEgress stops an active egress by its ID.
func (em *EgressManager) StopEgress(ctx context.Context, egressID string) error {
	_, err := em.client.StopEgress(ctx, &lkproto.StopEgressRequest{
		EgressId: egressID,
	})
	if err != nil {
		return fmt.Errorf("livekit stop egress: %w", err)
	}
	return nil
}
