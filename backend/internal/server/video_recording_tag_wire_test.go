package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kmuhub/kmuhub/internal/work/recording"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// dialVideoServer serves srv the way cmd/work does (RegisterVideoServiceServer)
// and returns a generated client connected to it over a real socket.
func dialVideoServer(t *testing.T, srv videov1.VideoServiceServer) videov1.VideoServiceClient {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	gs := grpc.NewServer()
	videov1.RegisterVideoServiceServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return videov1.NewVideoServiceClient(conn)
}

// TestRecordingTagRPCs_OverTheWire drives the six recording tagging RPCs
// through a real gRPC server. Their hand-written predecessors were neither
// proto.Message nor part of the service descriptor, so every call failed with
// Internal before it left the client.
func TestRecordingTagRPCs_OverTheWire(t *testing.T) {
	ts := newTestVideoCallServer()
	client := dialVideoServer(t, ts.srv)
	ctx := context.Background()

	meetingID := uuid.New()
	recID := uuid.New()
	past := time.Now().Add(-time.Hour)
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, MeetingID: &meetingID, Status: recording.RecordingStatusProcessing, CreatedAt: time.Now()}
	expiredID := uuid.New()
	ts.recRepo.recordings[expiredID] = &recording.Recording{ID: expiredID, Status: recording.RecordingStatusCompleted, RetentionExpiresAt: &past, CreatedAt: time.Now()}
	user := uuid.New()
	_, err := client.SetRecordingConsent(ctx, &videov1.SetRecordingConsentRequest{RecordingId: recID.String(), UserId: user.String(), Consented: true})
	require.NoError(t, err)

	t.Run("GetRecordingConsents", func(t *testing.T) {
		resp, err := client.GetRecordingConsents(ctx, &videov1.GetRecordingConsentsRequest{RecordingId: recID.String()})
		require.NoError(t, err)
		assert.Equal(t, recID.String(), resp.GetRecordingId())
		require.Len(t, resp.GetConsents(), 1)
		assert.Equal(t, user.String(), resp.GetConsents()[0].GetUserId())
		assert.True(t, resp.GetAllConsented())
	})

	t.Run("TagRecordingWithConsents", func(t *testing.T) {
		_, err := client.TagRecordingWithConsents(ctx, &videov1.TagRecordingWithConsentsRequest{
			RecordingId: recID.String(),
			Snapshot:    []*videov1.ConsentSnapshotEntry{{UserId: user.String(), DisplayName: "Ada"}},
		})
		require.NoError(t, err)
		snap := ts.recRepo.recordings[recID].ConsentSnapshot
		require.Len(t, snap, 1)
		assert.Equal(t, user, snap[0].UserID)
		assert.Equal(t, "Ada", snap[0].DisplayName)
	})

	t.Run("UpdateRecordingMetadata", func(t *testing.T) {
		fileURL := "https://example.com/rec.mp4"
		resp, err := client.UpdateRecordingMetadata(ctx, &videov1.UpdateRecordingMetadataRequest{RecordingId: recID.String(), FileUrl: &fileURL})
		require.NoError(t, err)
		assert.Equal(t, fileURL, resp.GetFileUrl())
	})

	t.Run("ListRecordingsByMeeting", func(t *testing.T) {
		resp, err := client.ListRecordingsByMeeting(ctx, &videov1.ListRecordingsByMeetingRequest{MeetingId: meetingID.String(), Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, resp.GetRecordings(), 1)
		assert.Equal(t, recID.String(), resp.GetRecordings()[0].GetId())
		assert.Equal(t, int32(1), resp.GetTotal())
	})

	t.Run("GetRecordingStatus", func(t *testing.T) {
		resp, err := client.GetRecordingStatus(ctx, &videov1.GetRecordingStatusRequest{RecordingId: recID.String()})
		require.NoError(t, err)
		assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_PROCESSING, resp.GetStatus())
	})

	t.Run("CleanupExpiredRecording", func(t *testing.T) {
		_, err := client.CleanupExpiredRecording(ctx, &videov1.CleanupExpiredRecordingRequest{RecordingId: expiredID.String()})
		require.NoError(t, err)
		_, stillThere := ts.recRepo.recordings[expiredID]
		assert.False(t, stillThere)
	})
}
