package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/work/recording"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// ---------------------------------------------------------------------------
// Egress webhook callbacks — CompleteRecordingByEgress / FailRecordingByEgress
// (b-cov-server-video-egress-callbacks). These run in system context without a
// caller tenant, so the tenant-scoping tests that dominate the rest of this
// package don't apply here — the ID lookup itself is the trust boundary.
// ---------------------------------------------------------------------------

func seedEgressRecording(ts *videoCallTestServer, egressID string) uuid.UUID {
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{
		ID:       recID,
		Status:   recording.RecordingStatusProcessing,
		EgressID: &egressID,
	}
	ts.recRepo.byEgress[egressID] = recID
	return recID
}

func TestCompleteRecordingByEgress_KnownEgressID_UpdatesRecording(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := seedEgressRecording(ts, "egress-complete-1")

	_, err := ts.srv.CompleteRecordingByEgress(context.Background(), &videov1.CompleteRecordingByEgressRequest{
		EgressId:        "egress-complete-1",
		FileUrl:         "https://example.com/rec.mp4",
		FileSizeBytes:   2048,
		DurationSeconds: 90,
	})
	require.NoError(t, err)

	rec := ts.recRepo.recordings[recID]
	assert.Equal(t, recording.RecordingStatusCompleted, rec.Status)
	require.NotNil(t, rec.FileURL)
	assert.Equal(t, "https://example.com/rec.mp4", *rec.FileURL)
	require.NotNil(t, rec.FileSizeBytes)
	assert.Equal(t, int64(2048), *rec.FileSizeBytes)
	require.NotNil(t, rec.DurationSeconds)
	assert.Equal(t, 90, *rec.DurationSeconds)
}

func TestCompleteRecordingByEgress_UnknownEgressID_NotFound(t *testing.T) {
	ts := newTestVideoCallServer()
	_, err := ts.srv.CompleteRecordingByEgress(context.Background(), &videov1.CompleteRecordingByEgressRequest{
		EgressId: "does-not-exist",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCompleteRecordingByEgress_EmptyEgressID_InvalidArgument(t *testing.T) {
	ts := newTestVideoCallServer()
	_, err := ts.srv.CompleteRecordingByEgress(context.Background(), &videov1.CompleteRecordingByEgressRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestFailRecordingByEgress_KnownEgressID_MarksFailed(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := seedEgressRecording(ts, "egress-fail-1")

	_, err := ts.srv.FailRecordingByEgress(context.Background(), &videov1.FailRecordingByEgressRequest{
		EgressId: "egress-fail-1",
		Reason:   "livekit egress error",
	})
	require.NoError(t, err)

	rec := ts.recRepo.recordings[recID]
	assert.Equal(t, recording.RecordingStatusFailed, rec.Status)
}

func TestFailRecordingByEgress_UnknownEgressID_NotFound(t *testing.T) {
	ts := newTestVideoCallServer()
	_, err := ts.srv.FailRecordingByEgress(context.Background(), &videov1.FailRecordingByEgressRequest{
		EgressId: "does-not-exist",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestFailRecordingByEgress_EmptyEgressID_InvalidArgument(t *testing.T) {
	ts := newTestVideoCallServer()
	_, err := ts.srv.FailRecordingByEgress(context.Background(), &videov1.FailRecordingByEgressRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// GetMeetingNotes
//
// Reads the caller's own notes via meeting.Service.GetNotes. Before that
// existed, the handler faked the read with a SaveNotes call on empty content
// (always rejected) and answered every request with an empty stub, so saved
// notes could never be read back and an unknown meeting_id returned 200.
// ---------------------------------------------------------------------------

func TestGetMeetingNotes_ReturnsWhatSaveMeetingNotesStored(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	organizerID := uuid.New()
	meetingResp := seedMeeting(t, srv, repo, tenantID, organizerID)
	ctx := meetingCtxTenant(tenantID)

	_, err := srv.StartMeeting(ctx, &videov1.StartMeetingRequest{MeetingId: meetingResp.Id, UserId: organizerID.String()})
	require.NoError(t, err)
	_, err = srv.SaveMeetingNotes(ctx, &videov1.SaveMeetingNotesRequest{
		MeetingId: meetingResp.Id,
		AuthorId:  organizerID.String(),
		Content:   "Decisions: ship on Friday",
	})
	require.NoError(t, err)

	resp, err := srv.GetMeetingNotes(ctx, &videov1.GetMeetingNotesRequest{
		MeetingId: meetingResp.Id,
		UserId:    organizerID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Decisions: ship on Friday", resp.Content)
	assert.Equal(t, organizerID.String(), resp.AuthorId)
	assert.NotEmpty(t, resp.Id)
}

func TestGetMeetingNotes_NoNotesYet_ReturnsEmptyNotesNotError(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	organizerID := uuid.New()
	meetingResp := seedMeeting(t, srv, repo, tenantID, organizerID)

	resp, err := srv.GetMeetingNotes(meetingCtxTenant(tenantID), &videov1.GetMeetingNotesRequest{
		MeetingId: meetingResp.Id,
		UserId:    organizerID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, meetingResp.Id, resp.MeetingId)
	assert.Equal(t, organizerID.String(), resp.AuthorId)
	assert.Empty(t, resp.Content)
	assert.Empty(t, resp.Id)
}

func TestGetMeetingNotes_OtherAuthorsNotesAreNotReturned(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	tenantID := uuid.New()
	organizerID := uuid.New()
	meetingResp := seedMeeting(t, srv, repo, tenantID, organizerID)
	ctx := meetingCtxTenant(tenantID)

	_, err := srv.StartMeeting(ctx, &videov1.StartMeetingRequest{MeetingId: meetingResp.Id, UserId: organizerID.String()})
	require.NoError(t, err)
	_, err = srv.SaveMeetingNotes(ctx, &videov1.SaveMeetingNotesRequest{
		MeetingId: meetingResp.Id, AuthorId: organizerID.String(), Content: "organizer only", IsPrivate: true,
	})
	require.NoError(t, err)

	resp, err := srv.GetMeetingNotes(ctx, &videov1.GetMeetingNotesRequest{
		MeetingId: meetingResp.Id,
		UserId:    uuid.New().String(),
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Content)
}

func TestGetMeetingNotes_UnknownMeetingID_NotFound(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	_, err := srv.GetMeetingNotes(meetingCtxTenant(uuid.New()), &videov1.GetMeetingNotesRequest{
		MeetingId: uuid.New().String(),
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetMeetingNotes_ForeignTenantMeeting_NotFound(t *testing.T) {
	srv, repo, _, _ := newTestVideoMeetingServer()
	organizerID := uuid.New()
	meetingResp := seedMeeting(t, srv, repo, uuid.New(), organizerID)

	_, err := srv.GetMeetingNotes(meetingCtxTenant(uuid.New()), &videov1.GetMeetingNotesRequest{
		MeetingId: meetingResp.Id,
		UserId:    organizerID.String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetMeetingNotes_InvalidMeetingID(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	_, err := srv.GetMeetingNotes(meetingCtxTenant(uuid.New()), &videov1.GetMeetingNotesRequest{
		MeetingId: "bogus",
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetMeetingNotes_InvalidUserID(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	_, err := srv.GetMeetingNotes(meetingCtxTenant(uuid.New()), &videov1.GetMeetingNotesRequest{
		MeetingId: uuid.New().String(),
		UserId:    "bogus",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetMeetingNotes_MissingTenant(t *testing.T) {
	srv, _, _, _ := newTestVideoMeetingServer()
	_, err := srv.GetMeetingNotes(context.Background(), &videov1.GetMeetingNotesRequest{
		MeetingId: uuid.New().String(),
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}
