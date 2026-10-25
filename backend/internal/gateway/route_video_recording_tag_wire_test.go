package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

type recordingTagStub struct {
	videov1.UnimplementedVideoServiceServer
	tagged     *videov1.TagRecordingWithConsentsRequest
	meta       *videov1.UpdateRecordingMetadataRequest
	byMeeting  *videov1.ListRecordingsByMeetingRequest
	cleanedUp  string
	recordings []*videov1.Recording
}

func (s *recordingTagStub) GetRecordingConsents(_ context.Context, r *videov1.GetRecordingConsentsRequest) (*videov1.GetRecordingConsentsResponse, error) {
	return &videov1.GetRecordingConsentsResponse{
		RecordingId: r.GetRecordingId(),
		Consents:    []*videov1.RecordingConsent{{RecordingId: r.GetRecordingId(), UserId: "u1", Consented: false}},
	}, nil
}

func (s *recordingTagStub) TagRecordingWithConsents(_ context.Context, r *videov1.TagRecordingWithConsentsRequest) (*emptypb.Empty, error) {
	s.tagged = r
	return &emptypb.Empty{}, nil
}

func (s *recordingTagStub) UpdateRecordingMetadata(_ context.Context, r *videov1.UpdateRecordingMetadataRequest) (*videov1.Recording, error) {
	s.meta = r
	return &videov1.Recording{Id: r.GetRecordingId(), FileUrl: r.FileUrl}, nil
}

func (s *recordingTagStub) ListRecordingsByMeeting(_ context.Context, r *videov1.ListRecordingsByMeetingRequest) (*videov1.ListRecordingsByMeetingResponse, error) {
	s.byMeeting = r
	return &videov1.ListRecordingsByMeetingResponse{Recordings: s.recordings, Total: int32(len(s.recordings))}, nil
}

func (s *recordingTagStub) GetRecordingStatus(_ context.Context, r *videov1.GetRecordingStatusRequest) (*videov1.Recording, error) {
	return &videov1.Recording{Id: r.GetRecordingId(), Status: videov1.RecordingStatus_RECORDING_STATUS_COMPLETED}, nil
}

func (s *recordingTagStub) CleanupExpiredRecording(_ context.Context, r *videov1.CleanupExpiredRecordingRequest) (*emptypb.Empty, error) {
	s.cleanedUp = r.GetRecordingId()
	return &emptypb.Empty{}, nil
}

// TestRecordingTagRoutes_Wire runs the six recording tagging handlers against a
// real gRPC server and pins each response to its type in video-types.ts. Until
// the RPCs lived in video.proto every one of these routes answered 500.
func TestRecordingTagRoutes_Wire(t *testing.T) {
	recID := uuid.New().String()
	meetingID := uuid.New().String()

	call := func(t *testing.T, stub *recordingTagStub, h func(*VideoRoutes) http.HandlerFunc, method, body, param, value string) map[string]any {
		t.Helper()
		routes := NewVideoRoutes(registryFor(t, "work", func(s *grpc.Server) { videov1.RegisterVideoServiceServer(s, stub) }), "", "")
		req := httptest.NewRequest(method, "/x", strings.NewReader(body))
		req = withChiURLParam(withAuth(req, uuid.New().String(), testTenantID), param, value)
		rec := httptest.NewRecorder()
		h(routes)(rec, req)
		assertStatus(t, rec, http.StatusOK)
		var out map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
		return out
	}

	t.Run("GET consents keeps every RecordingConsentsResponse member", func(t *testing.T) {
		out := call(t, &recordingTagStub{}, func(v *VideoRoutes) http.HandlerFunc { return v.HandleGetRecordingConsents }, http.MethodGet, "", "id", recID)
		assert.Equal(t, recID, out["recording_id"])
		assert.Equal(t, false, out["all_consented"])
		consents, ok := out["consents"].([]any)
		require.True(t, ok, "consents must be an array: %v", out)
		require.Len(t, consents, 1)
		c := consents[0].(map[string]any)
		assert.Equal(t, "u1", c["user_id"])
		assert.Equal(t, false, c["consented"])
	})

	t.Run("POST tag-consents forwards the snapshot", func(t *testing.T) {
		stub := &recordingTagStub{}
		out := call(t, stub, func(v *VideoRoutes) http.HandlerFunc { return v.HandleTagRecordingWithConsents }, http.MethodPost,
			`{"snapshot":[{"user_id":"u1","display_name":"Ada","joined_at":"2026-10-25T02:00:00Z"}]}`, "id", recID)
		assert.Equal(t, "consent snapshot updated", out["status"])
		require.NotNil(t, stub.tagged)
		assert.Equal(t, recID, stub.tagged.GetRecordingId())
		require.Len(t, stub.tagged.GetSnapshot(), 1)
		e := stub.tagged.GetSnapshot()[0]
		assert.Equal(t, "u1", e.GetUserId())
		assert.Equal(t, "Ada", e.GetDisplayName())
		assert.Equal(t, "2026-10-25T02:00:00Z", e.GetJoinedAt())
	})

	t.Run("PATCH metadata forwards only the sent fields and returns the Recording", func(t *testing.T) {
		stub := &recordingTagStub{}
		out := call(t, stub, func(v *VideoRoutes) http.HandlerFunc { return v.HandleUpdateRecordingMetadata }, http.MethodPatch,
			`{"file_url":"https://example.com/r.mp4"}`, "id", recID)
		assert.Equal(t, recID, out["id"])
		assert.Equal(t, "https://example.com/r.mp4", out["file_url"])
		require.NotNil(t, stub.meta)
		assert.Equal(t, "https://example.com/r.mp4", stub.meta.GetFileUrl())
		assert.Nil(t, stub.meta.Status, "an unsent status must stay unset")
		assert.Nil(t, stub.meta.FileSizeBytes)
	})

	t.Run("GET status returns the Recording", func(t *testing.T) {
		out := call(t, &recordingTagStub{}, func(v *VideoRoutes) http.HandlerFunc { return v.HandleGetRecordingStatus }, http.MethodGet, "", "id", recID)
		assert.Equal(t, recID, out["id"])
		assert.NotEmpty(t, out["status"])
	})

	t.Run("GET meeting recordings keeps the envelope, empty list is []", func(t *testing.T) {
		stub := &recordingTagStub{}
		out := call(t, stub, func(v *VideoRoutes) http.HandlerFunc { return v.HandleListRecordingsByMeeting }, http.MethodGet, "", "meetingId", meetingID)
		assert.Equal(t, []any{}, out["recordings"])
		assert.EqualValues(t, 0, out["total"])
		require.NotNil(t, stub.byMeeting)
		assert.Equal(t, meetingID, stub.byMeeting.GetMeetingId())

		stub = &recordingTagStub{recordings: []*videov1.Recording{{Id: recID}}}
		out = call(t, stub, func(v *VideoRoutes) http.HandlerFunc { return v.HandleListRecordingsByMeeting }, http.MethodGet, "", "meetingId", meetingID)
		require.Len(t, out["recordings"], 1)
		assert.EqualValues(t, 1, out["total"])
	})

	t.Run("POST cleanup reaches the service", func(t *testing.T) {
		stub := &recordingTagStub{}
		out := call(t, stub, func(v *VideoRoutes) http.HandlerFunc { return v.HandleCleanupExpiredRecording }, http.MethodPost, "", "id", recID)
		assert.Equal(t, "recording deleted", out["status"])
		assert.Equal(t, recID, stub.cleanedUp)
	})
}
