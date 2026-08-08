package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/work/meeting"
	"github.com/kmuhub/kmuhub/internal/work/recording"
	"github.com/kmuhub/kmuhub/internal/work/video"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// newFakeS3LocationServer starts an httptest server that answers any request the
// way a real S3-compatible store answers GetBucketLocation, so minio-go's region
// auto-discovery (triggered on the first presign call when no Region is
// configured) succeeds without live infrastructure. Returns the endpoint in
// "host:port" form, as minio.New expects (no scheme).
func newFakeS3LocationServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

// ---------------------------------------------------------------------------
// video.Repository stateful mock — covers the calls/recordings/pre-consent
// half of video_grpc.go (b-cov-server-video-calls).
// ---------------------------------------------------------------------------

type callMockRepo struct {
	sessions     map[uuid.UUID]*video.CallSession
	byRoom       map[string]*video.CallSession
	participants map[uuid.UUID][]video.CallParticipant
}

func newCallMockRepo() *callMockRepo {
	return &callMockRepo{
		sessions:     make(map[uuid.UUID]*video.CallSession),
		byRoom:       make(map[string]*video.CallSession),
		participants: make(map[uuid.UUID][]video.CallParticipant),
	}
}

func (m *callMockRepo) CreateCallSession(_ context.Context, session *video.CallSession) error {
	cp := *session
	m.sessions[session.ID] = &cp
	m.byRoom[session.RoomName] = &cp
	return nil
}

func (m *callMockRepo) GetCallSession(_ context.Context, id uuid.UUID) (*video.CallSession, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, video.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *callMockRepo) GetCallSessionByRoomName(_ context.Context, roomName string) (*video.CallSession, error) {
	s, ok := m.byRoom[roomName]
	if !ok {
		return nil, video.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *callMockRepo) UpdateCallSession(_ context.Context, session *video.CallSession) error {
	if _, ok := m.sessions[session.ID]; !ok {
		return video.ErrNotFound
	}
	cp := *session
	m.sessions[session.ID] = &cp
	m.byRoom[session.RoomName] = &cp
	return nil
}

func (m *callMockRepo) ListActiveCallsForUser(_ context.Context, userID uuid.UUID) ([]video.CallSession, error) {
	var result []video.CallSession
	for _, s := range m.sessions {
		if s.Status == video.CallStatusEnded {
			continue
		}
		for _, p := range m.participants[s.ID] {
			if p.UserID == userID && p.LeftAt == nil {
				result = append(result, *s)
				break
			}
		}
	}
	return result, nil
}

func (m *callMockRepo) AddParticipant(_ context.Context, p *video.CallParticipant) error {
	parts := m.participants[p.CallID]
	for i, existing := range parts {
		if existing.UserID == p.UserID {
			parts[i].LeftAt = nil
			parts[i].JoinedAt = p.JoinedAt
			m.participants[p.CallID] = parts
			return nil
		}
	}
	m.participants[p.CallID] = append(m.participants[p.CallID], *p)
	return nil
}

func (m *callMockRepo) RemoveParticipant(_ context.Context, callID, userID uuid.UUID) error {
	parts := m.participants[callID]
	now := time.Now()
	for i, p := range parts {
		if p.UserID == userID && p.LeftAt == nil {
			parts[i].LeftAt = &now
		}
	}
	m.participants[callID] = parts
	return nil
}

func (m *callMockRepo) GetParticipants(_ context.Context, callID uuid.UUID) ([]video.CallParticipant, error) {
	return m.participants[callID], nil
}

func (m *callMockRepo) CountActiveParticipants(_ context.Context, callID uuid.UUID) (int, error) {
	count := 0
	for _, p := range m.participants[callID] {
		if p.LeftAt == nil {
			count++
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// video.RoomManager mock
// ---------------------------------------------------------------------------

type callMockRoomManager struct {
	rooms      map[string]uint32
	createErr  error
	deleteErr  error
	tokenErr   error
	iceServers []video.IceServerConfig
}

func newCallMockRoomManager() *callMockRoomManager {
	return &callMockRoomManager{rooms: make(map[string]uint32)}
}

func (m *callMockRoomManager) CreateRoom(_ context.Context, name string, maxParticipants uint32) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.rooms[name] = maxParticipants
	return nil
}

func (m *callMockRoomManager) DeleteRoom(_ context.Context, name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.rooms, name)
	return nil
}

func (m *callMockRoomManager) GenerateToken(_, userID, _ string) (string, error) {
	if m.tokenErr != nil {
		return "", m.tokenErr
	}
	return "mock-token-" + userID, nil
}

func (m *callMockRoomManager) TURNIceServers(_ string) []video.IceServerConfig {
	return m.iceServers
}

func (m *callMockRoomManager) ListParticipants(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (m *callMockRoomManager) MuteParticipant(_ context.Context, _, _ string) error {
	return nil
}

func (m *callMockRoomManager) RemoveParticipant(_ context.Context, _, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// recording.Repository stateful mock
// ---------------------------------------------------------------------------

type recordingMockRepo struct {
	recordings       map[uuid.UUID]*recording.Recording
	byEgress         map[string]uuid.UUID
	consents         map[uuid.UUID][]recording.RecordingConsent
	participantUsers map[uuid.UUID][]uuid.UUID // recordingID -> ACL'd user IDs
	pendingOverride  *int
}

func newRecordingMockRepo() *recordingMockRepo {
	return &recordingMockRepo{
		recordings:       make(map[uuid.UUID]*recording.Recording),
		byEgress:         make(map[string]uuid.UUID),
		consents:         make(map[uuid.UUID][]recording.RecordingConsent),
		participantUsers: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *recordingMockRepo) CreateRecording(_ context.Context, rec *recording.Recording) error {
	cp := *rec
	m.recordings[rec.ID] = &cp
	if rec.EgressID != nil {
		m.byEgress[*rec.EgressID] = rec.ID
	}
	return nil
}

func (m *recordingMockRepo) GetRecording(_ context.Context, id uuid.UUID) (*recording.Recording, error) {
	rec, ok := m.recordings[id]
	if !ok {
		return nil, recording.ErrNotFound
	}
	cp := *rec
	return &cp, nil
}

func (m *recordingMockRepo) GetRecordingByEgressID(_ context.Context, egressID string) (*recording.Recording, error) {
	id, ok := m.byEgress[egressID]
	if !ok {
		return nil, recording.ErrNotFound
	}
	rec := m.recordings[id]
	cp := *rec
	return &cp, nil
}

func (m *recordingMockRepo) UpdateRecording(_ context.Context, rec *recording.Recording) error {
	cp := *rec
	m.recordings[rec.ID] = &cp
	if rec.EgressID != nil {
		m.byEgress[*rec.EgressID] = rec.ID
	}
	return nil
}

func (m *recordingMockRepo) TagRecordingWithConsents(_ context.Context, recordingID uuid.UUID, snapshot []recording.ParticipantConsentInfo) error {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return recording.ErrNotFound
	}
	rec.ConsentSnapshot = snapshot
	return nil
}

func (m *recordingMockRepo) ListRecordingsByCall(_ context.Context, callID uuid.UUID) ([]recording.Recording, error) {
	var result []recording.Recording
	for _, rec := range m.recordings {
		if rec.CallID != nil && *rec.CallID == callID {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *recordingMockRepo) ListRecordingsByMeeting(_ context.Context, meetingID uuid.UUID) ([]recording.Recording, error) {
	var result []recording.Recording
	for _, rec := range m.recordings {
		if rec.MeetingID != nil && *rec.MeetingID == meetingID {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *recordingMockRepo) DeleteRecording(_ context.Context, id uuid.UUID) error {
	delete(m.recordings, id)
	return nil
}

func (m *recordingMockRepo) SetConsent(_ context.Context, consent *recording.RecordingConsent) error {
	existing := m.consents[consent.RecordingID]
	found := false
	for i, c := range existing {
		if c.UserID == consent.UserID {
			existing[i] = *consent
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, *consent)
	}
	m.consents[consent.RecordingID] = existing
	return nil
}

func (m *recordingMockRepo) GetConsents(_ context.Context, recordingID uuid.UUID) ([]recording.RecordingConsent, error) {
	return m.consents[recordingID], nil
}

func (m *recordingMockRepo) GetConsentsWithUser(_ context.Context, recordingID uuid.UUID) ([]recording.RecordingConsentWithUser, error) {
	raw := m.consents[recordingID]
	result := make([]recording.RecordingConsentWithUser, len(raw))
	for i, c := range raw {
		result[i] = recording.RecordingConsentWithUser{RecordingConsent: c, FirstName: "Mock", LastName: "User"}
	}
	return result, nil
}

func (m *recordingMockRepo) CountPendingConsents(_ context.Context, recordingID uuid.UUID, participantIDs []uuid.UUID) (int, error) {
	if m.pendingOverride != nil {
		return *m.pendingOverride, nil
	}
	responded := make(map[uuid.UUID]bool)
	for _, c := range m.consents[recordingID] {
		responded[c.UserID] = true
	}
	pending := 0
	for _, pid := range participantIDs {
		if !responded[pid] {
			pending++
		}
	}
	return pending, nil
}

func (m *recordingMockRepo) ListExpiredRecordings(_ context.Context, before time.Time) ([]recording.Recording, error) {
	var result []recording.Recording
	for _, rec := range m.recordings {
		if rec.RetentionExpiresAt != nil && rec.RetentionExpiresAt.Before(before) {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *recordingMockRepo) ListRecordingsWithAccess(_ context.Context, userID uuid.UUID, callID, meetingID *uuid.UUID) ([]recording.Recording, error) {
	var result []recording.Recording
	for _, rec := range m.recordings {
		if !slices.Contains(m.participantUsers[rec.ID], userID) {
			continue
		}
		if callID != nil && (rec.CallID == nil || *rec.CallID != *callID) {
			continue
		}
		if meetingID != nil && (rec.MeetingID == nil || *rec.MeetingID != *meetingID) {
			continue
		}
		result = append(result, *rec)
	}
	return result, nil
}

func (m *recordingMockRepo) GetRecordingParticipants(_ context.Context, recordingID uuid.UUID) ([]uuid.UUID, error) {
	return m.participantUsers[recordingID], nil
}

func (m *recordingMockRepo) MarkInitiatorConsent(_ context.Context, recordingID, _, _ uuid.UUID) error {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return recording.ErrNotFound
	}
	now := time.Now()
	rec.PreRecordingConsentAt = &now
	return nil
}

func (m *recordingMockRepo) GetPreConsentStatus(_ context.Context, recordingID, _ uuid.UUID) (bool, error) {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return false, recording.ErrNotFound
	}
	return rec.PreRecordingConsentAt != nil, nil
}

// ---------------------------------------------------------------------------
// recording.EgressManager mock
// ---------------------------------------------------------------------------

type callMockEgressManager struct {
	startErr     error
	stopErr      error
	stopCalled   bool
	lastEgressID string
}

func newCallMockEgressManager() *callMockEgressManager {
	return &callMockEgressManager{lastEgressID: "egress-mock-1"}
}

func (m *callMockEgressManager) StartRoomCompositeEgress(_ context.Context, _, _ string, _ recording.S3Config) (string, error) {
	if m.startErr != nil {
		return "", m.startErr
	}
	return m.lastEgressID, nil
}

func (m *callMockEgressManager) StopEgress(_ context.Context, _ string) error {
	m.stopCalled = true
	return m.stopErr
}

// ---------------------------------------------------------------------------
// Test server construction
// ---------------------------------------------------------------------------

type videoCallTestServer struct {
	srv         *VideoGRPCServer
	callRepo    *callMockRepo
	recRepo     *recordingMockRepo
	roomMgr     *callMockRoomManager
	egress      *callMockEgressManager
	meetingRepo *meetingMockRepo
}

// newTestVideoCallServer wires the same roomMgr instance into both video.NewService
// (call room lifecycle) and NewVideoGRPCServer's tokenGen param (TURN ICE servers on
// join), mirroring cmd/work/main.go's production wiring at line 209.
func newTestVideoCallServer() *videoCallTestServer {
	callRepo := newCallMockRepo()
	roomMgr := newCallMockRoomManager()
	videoSvc := video.NewService(callRepo, roomMgr)

	recRepo := newRecordingMockRepo()
	egress := newCallMockEgressManager()
	recSvc := recording.NewService(recRepo, egress, "https://template.example.com", recording.S3Config{
		Endpoint:  "minio.example.com",
		AccessKey: "mock-access-key",
		Secret:    "mock-secret-key",
		Bucket:    "recordings",
	})

	meetingRepo := newMeetingMockRepo()
	meetingSvc := meeting.NewService(meetingRepo)

	srv := NewVideoGRPCServer(videoSvc, meetingSvc, recSvc, nil, nil, nil, roomMgr, "wss://video.example.com")
	return &videoCallTestServer{srv: srv, callRepo: callRepo, recRepo: recRepo, roomMgr: roomMgr, egress: egress, meetingRepo: meetingRepo}
}

func callCtxTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func callCtxAuth(tenantID, userID uuid.UUID) context.Context {
	ctx := callCtxTenant(tenantID)
	return context.WithValue(ctx, middleware.UserIDKey, userID.String())
}

// ---------------------------------------------------------------------------
// Error mapping — mapVideoError / mapRecordingError
// ---------------------------------------------------------------------------

func TestMapVideoError_AllSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{video.ErrNotFound, codes.NotFound},
		{video.ErrCallEnded, codes.FailedPrecondition},
		{video.ErrMaxParticipants, codes.ResourceExhausted},
		{video.ErrLiveKitNotConfigured, codes.Unavailable},
		{video.ErrInvalidCallType, codes.InvalidArgument},
		{video.ErrNoTargetUsers, codes.InvalidArgument},
		{errors.New("boom"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.err.Error(), func(t *testing.T) {
			mapped := mapVideoError(c.err)
			requireGRPCCode(t, mapped, c.code)
		})
	}
}

func TestMapRecordingError_AllSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{recording.ErrNotFound, codes.NotFound},
		{recording.ErrConsentPending, codes.FailedPrecondition},
		{recording.ErrEgressNotConfigured, codes.Unavailable},
		{recording.ErrRecordingNotActive, codes.FailedPrecondition},
		{recording.ErrNoCallOrMeeting, codes.InvalidArgument},
		{recording.ErrNoParticipants, codes.InvalidArgument},
		{recording.ErrNotExpired, codes.FailedPrecondition},
		{recording.ErrInvalidStatus, codes.InvalidArgument},
		// ErrPreConsentMissing is declared but never actually returned anywhere in
		// this codebase (see journal finding on the dead pre-consent gate) — it
		// falls through to Internal exactly like any other unmapped error.
		{recording.ErrPreConsentMissing, codes.Internal},
		{errors.New("boom"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.err.Error(), func(t *testing.T) {
			mapped := mapRecordingError(c.err)
			requireGRPCCode(t, mapped, c.code)
		})
	}
}

// ---------------------------------------------------------------------------
// toProto* conversions
// ---------------------------------------------------------------------------

func TestCallSessionToProto_NilOptionalsAndParticipants(t *testing.T) {
	c := &video.CallSession{
		ID:          uuid.New(),
		CallType:    video.CallTypeOneToOne,
		Status:      video.CallStatusRinging,
		RoomName:    "call-abc",
		InitiatorID: uuid.New(),
		CreatedAt:   time.Now(),
	}
	proto := callSessionToProto(c, nil)
	assert.Nil(t, proto.ChannelId)
	assert.Nil(t, proto.EndedAt)
	assert.Nil(t, proto.DurationSeconds)
	assert.Empty(t, proto.Participants)
}

func TestCallSessionToProto_AllFieldsAndParticipants(t *testing.T) {
	channelID := uuid.New()
	endedAt := time.Now()
	duration := 42
	c := &video.CallSession{
		ID:              uuid.New(),
		CallType:        video.CallTypeGroup,
		Status:          video.CallStatusEnded,
		RoomName:        "call-xyz",
		InitiatorID:     uuid.New(),
		ChannelID:       &channelID,
		CreatedAt:       time.Now(),
		EndedAt:         &endedAt,
		DurationSeconds: &duration,
	}
	participants := []video.CallParticipantWithUser{
		{CallParticipant: video.CallParticipant{CallID: c.ID, UserID: uuid.New(), JoinedAt: time.Now()}},
	}
	proto := callSessionToProto(c, participants)
	require.NotNil(t, proto.ChannelId)
	assert.Equal(t, channelID.String(), *proto.ChannelId)
	require.NotNil(t, proto.EndedAt)
	require.NotNil(t, proto.DurationSeconds)
	assert.Equal(t, int32(42), *proto.DurationSeconds)
	assert.Len(t, proto.Participants, 1)
}

func TestCallSessionWithParticipantsToProto(t *testing.T) {
	c := &video.CallSessionWithParticipants{
		CallSession: video.CallSession{
			ID:          uuid.New(),
			CallType:    video.CallTypeOneToOne,
			Status:      video.CallStatusActive,
			RoomName:    "call-p",
			InitiatorID: uuid.New(),
			CreatedAt:   time.Now(),
		},
		Participants: []video.CallParticipantWithUser{
			{CallParticipant: video.CallParticipant{UserID: uuid.New(), JoinedAt: time.Now()}},
			{CallParticipant: video.CallParticipant{UserID: uuid.New(), JoinedAt: time.Now()}},
		},
	}
	proto := callSessionWithParticipantsToProto(c)
	assert.Len(t, proto.Participants, 2)
}

func TestCallParticipantToProto_LeftAtNilAndSet(t *testing.T) {
	p := &video.CallParticipantWithUser{
		CallParticipant: video.CallParticipant{
			CallID:   uuid.New(),
			UserID:   uuid.New(),
			JoinedAt: time.Now(),
			HasVideo: true,
			HasAudio: false,
		},
		FirstName: "Ada",
		LastName:  "Lovelace",
	}
	proto := callParticipantToProto(p)
	assert.Nil(t, proto.LeftAt)
	assert.True(t, proto.HasVideo)
	assert.False(t, proto.HasAudio)
	assert.Equal(t, "Ada", proto.FirstName)

	leftAt := time.Now()
	p.LeftAt = &leftAt
	proto2 := callParticipantToProto(p)
	require.NotNil(t, proto2.LeftAt)
}

func TestRecordingToProto_NilOptionals(t *testing.T) {
	r := &recording.Recording{
		ID:        uuid.New(),
		Status:    recording.RecordingStatusActive,
		CreatedAt: time.Now(),
	}
	proto := recordingToProto(r)
	assert.Nil(t, proto.CallId)
	assert.Nil(t, proto.MeetingId)
	assert.Nil(t, proto.EgressId)
	assert.Nil(t, proto.FileUrl)
	assert.Nil(t, proto.FileSizeBytes)
	assert.Nil(t, proto.DurationSeconds)
	assert.Nil(t, proto.RetentionExpiresAt)
	assert.Nil(t, proto.StartedBy)
}

func TestRecordingToProto_AllFieldsSet(t *testing.T) {
	callID := uuid.New()
	meetingID := uuid.New()
	egressID := "egress-1"
	fileURL := "https://example.com/rec.mp4"
	size := int64(1024)
	duration := 300
	retention := time.Now().Add(30 * 24 * time.Hour)
	startedBy := uuid.New()
	r := &recording.Recording{
		ID:                  uuid.New(),
		CallID:              &callID,
		MeetingID:           &meetingID,
		Status:              recording.RecordingStatusCompleted,
		EgressID:            &egressID,
		FileURL:             &fileURL,
		FileSizeBytes:       &size,
		DurationSeconds:     &duration,
		RetentionExpiresAt:  &retention,
		StartedBy:           &startedBy,
		CreatedAt:           time.Now(),
	}
	proto := recordingToProto(r)
	require.NotNil(t, proto.CallId)
	assert.Equal(t, callID.String(), *proto.CallId)
	require.NotNil(t, proto.MeetingId)
	assert.Equal(t, meetingID.String(), *proto.MeetingId)
	require.NotNil(t, proto.EgressId)
	assert.Equal(t, egressID, *proto.EgressId)
	require.NotNil(t, proto.FileUrl)
	assert.Equal(t, fileURL, *proto.FileUrl)
	require.NotNil(t, proto.FileSizeBytes)
	assert.Equal(t, size, *proto.FileSizeBytes)
	require.NotNil(t, proto.DurationSeconds)
	assert.Equal(t, int32(300), *proto.DurationSeconds)
	require.NotNil(t, proto.RetentionExpiresAt)
	require.NotNil(t, proto.StartedBy)
	assert.Equal(t, startedBy.String(), *proto.StartedBy)
}

func TestRecordingConsentToProto_RespondedAtZeroAndSet(t *testing.T) {
	c := &recording.RecordingConsent{
		RecordingID: uuid.New(),
		UserID:      uuid.New(),
		Consented:   true,
	}
	proto := recordingConsentToProto(c)
	assert.Nil(t, proto.RespondedAt)

	c.RespondedAt = time.Now()
	proto2 := recordingConsentToProto(c)
	require.NotNil(t, proto2.RespondedAt)
}

// ---------------------------------------------------------------------------
// Enum converters
// ---------------------------------------------------------------------------

func TestProtoCallTypeRoundTrip(t *testing.T) {
	assert.Equal(t, video.CallTypeOneToOne, protoCallTypeToString(videov1.CallType_CALL_TYPE_ONE_TO_ONE))
	assert.Equal(t, video.CallTypeGroup, protoCallTypeToString(videov1.CallType_CALL_TYPE_GROUP))
	// Unspecified/unknown defaults to one-to-one (documented default, not a bug —
	// CreateCallRequest always carries an explicit type from the FE).
	assert.Equal(t, video.CallTypeOneToOne, protoCallTypeToString(videov1.CallType_CALL_TYPE_UNSPECIFIED))

	assert.Equal(t, videov1.CallType_CALL_TYPE_ONE_TO_ONE, stringToProtoCallType(video.CallTypeOneToOne))
	assert.Equal(t, videov1.CallType_CALL_TYPE_GROUP, stringToProtoCallType(video.CallTypeGroup))
	assert.Equal(t, videov1.CallType_CALL_TYPE_UNSPECIFIED, stringToProtoCallType("bogus"))
}

func TestStringToProtoCallStatus_AllValuesAndDefault(t *testing.T) {
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_RINGING, stringToProtoCallStatus(video.CallStatusRinging))
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_ACTIVE, stringToProtoCallStatus(video.CallStatusActive))
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_ENDED, stringToProtoCallStatus(video.CallStatusEnded))
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_UNSPECIFIED, stringToProtoCallStatus("bogus"))
}

func TestStringToProtoRecordingStatus_AllValuesAndDefault(t *testing.T) {
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_ACTIVE, stringToProtoRecordingStatus(recording.RecordingStatusActive))
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_PROCESSING, stringToProtoRecordingStatus(recording.RecordingStatusProcessing))
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_COMPLETED, stringToProtoRecordingStatus(recording.RecordingStatusCompleted))
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_FAILED, stringToProtoRecordingStatus(recording.RecordingStatusFailed))
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_DELETED, stringToProtoRecordingStatus(recording.RecordingStatusDeleted))
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_UNSPECIFIED, stringToProtoRecordingStatus("bogus"))
}

// ---------------------------------------------------------------------------
// Empty list -> [] not null (root-cause fixes applied alongside this unit,
// same pattern as the meeting half in b-cov-server-video-meetings/Iteration 40)
// ---------------------------------------------------------------------------

func TestListActiveCalls_EmptyReturnsEmptySlice(t *testing.T) {
	ts := newTestVideoCallServer()
	resp, err := ts.srv.ListActiveCalls(context.Background(), &videov1.ListActiveCallsRequest{UserId: uuid.New().String()})
	require.NoError(t, err)
	assert.NotNil(t, resp.Calls)
	assert.Empty(t, resp.Calls)
}

func TestListRecordings_EmptyReturnsEmptySlice(t *testing.T) {
	ts := newTestVideoCallServer()
	userID := uuid.New().String()
	resp, err := ts.srv.ListRecordings(context.Background(), &videov1.ListRecordingsRequest{UserId: &userID})
	require.NoError(t, err)
	assert.NotNil(t, resp.Recordings)
	assert.Empty(t, resp.Recordings)
}

func TestGetRecordingConsent_EmptyReturnsEmptySlice(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusActive, CreatedAt: time.Now()}
	resp, err := ts.srv.GetRecordingConsent(context.Background(), &videov1.GetRecordingConsentRequest{RecordingId: recID.String()})
	require.NoError(t, err)
	assert.NotNil(t, resp.Consents)
	assert.Empty(t, resp.Consents)
}

func TestGetRecordingConsents_EmptyReturnsEmptySlice(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusActive, CreatedAt: time.Now()}
	resp, err := ts.srv.GetRecordingConsents(context.Background(), &videov1.GetRecordingConsentsRequest{RecordingID: recID.String()})
	require.NoError(t, err)
	assert.NotNil(t, resp.Consents)
	assert.Empty(t, resp.Consents)
}

func TestListRecordingsByMeeting_EmptyReturnsEmptySlice(t *testing.T) {
	ts := newTestVideoCallServer()
	resp, err := ts.srv.ListRecordingsByMeeting(context.Background(), &videov1.ListRecordingsByMeetingRequest{MeetingID: uuid.New().String()})
	require.NoError(t, err)
	assert.NotNil(t, resp.Recordings)
	assert.Empty(t, resp.Recordings)
}

// ---------------------------------------------------------------------------
// Validation paths — at least one invalid-input case per method in scope
// ---------------------------------------------------------------------------

func TestVideoCallHandlers_Validation(t *testing.T) {
	badChannel := "not-a-uuid"
	someUserID := uuid.New().String()
	emptyUserID := ""

	t.Run("CreateCall_InvalidInitiatorID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.CreateCall(context.Background(), &videov1.CreateCallRequest{InitiatorId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateCall_InvalidParticipantID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.CreateCall(context.Background(), &videov1.CreateCallRequest{InitiatorId: uuid.New().String(), ParticipantIds: []string{"bogus"}})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateCall_InvalidChannelID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.CreateCall(context.Background(), &videov1.CreateCallRequest{InitiatorId: uuid.New().String(), ChannelId: &badChannel})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("JoinCall_InvalidCallID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: "bogus", UserId: someUserID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("JoinCall_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: uuid.New().String(), UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("EndCall_InvalidCallID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: "bogus", UserId: someUserID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("EndCall_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: uuid.New().String(), UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetCall_InvalidCallID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetCall(context.Background(), &videov1.GetCallRequest{CallId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListActiveCalls_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ListActiveCalls(context.Background(), &videov1.ListActiveCallsRequest{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("StartRecording_MissingTenant", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.StartRecording(context.Background(), &videov1.StartRecordingRequest{UserId: someUserID})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("StartRecording_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.StartRecording(callCtxTenant(uuid.New()), &videov1.StartRecordingRequest{UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("StartRecording_InvalidCallID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		badCall := "bogus"
		_, err := ts.srv.StartRecording(callCtxTenant(uuid.New()), &videov1.StartRecordingRequest{UserId: someUserID, CallId: &badCall})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("StartRecording_InvalidMeetingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		badMeeting := "bogus"
		_, err := ts.srv.StartRecording(callCtxTenant(uuid.New()), &videov1.StartRecordingRequest{UserId: someUserID, MeetingId: &badMeeting})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("StopRecording_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.StopRecording(callCtxAuth(uuid.New(), uuid.New()), &videov1.StopRecordingRequest{RecordingId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("StopRecording_MissingTenant", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.StopRecording(context.Background(), &videov1.StopRecordingRequest{RecordingId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("StopRecording_MissingUserInCtx", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.StopRecording(callCtxTenant(uuid.New()), &videov1.StopRecordingRequest{RecordingId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
	t.Run("SetRecordingConsent_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.SetRecordingConsent(context.Background(), &videov1.SetRecordingConsentRequest{RecordingId: "bogus", UserId: someUserID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SetRecordingConsent_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.SetRecordingConsent(context.Background(), &videov1.SetRecordingConsentRequest{RecordingId: uuid.New().String(), UserId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetRecordingConsent_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetRecordingConsent(context.Background(), &videov1.GetRecordingConsentRequest{RecordingId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListRecordings_MissingUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ListRecordings(context.Background(), &videov1.ListRecordingsRequest{UserId: &emptyUserID})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
	t.Run("ListRecordings_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		bogus := "bogus"
		_, err := ts.srv.ListRecordings(context.Background(), &videov1.ListRecordingsRequest{UserId: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListRecordings_InvalidCallID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		bogus := "bogus"
		_, err := ts.srv.ListRecordings(context.Background(), &videov1.ListRecordingsRequest{UserId: &someUserID, CallId: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListRecordings_InvalidMeetingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		bogus := "bogus"
		_, err := ts.srv.ListRecordings(context.Background(), &videov1.ListRecordingsRequest{UserId: &someUserID, MeetingId: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteRecording_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.DeleteRecording(context.Background(), &videov1.DeleteRecordingRequest{RecordingId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetRecordingConsents_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetRecordingConsents(context.Background(), &videov1.GetRecordingConsentsRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("TagRecordingWithConsents_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.TagRecordingWithConsents(context.Background(), &videov1.TagRecordingWithConsentsRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("TagRecordingWithConsents_InvalidSnapshotUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		recID := uuid.New()
		ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, CreatedAt: time.Now()}
		_, err := ts.srv.TagRecordingWithConsents(context.Background(), &videov1.TagRecordingWithConsentsRequest{
			RecordingID: recID.String(),
			Snapshot:    []*videov1.ConsentSnapshotEntry{{UserID: "bogus"}},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateRecordingMetadata_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.UpdateRecordingMetadata(context.Background(), &videov1.UpdateRecordingMetadataRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListRecordingsByMeeting_InvalidMeetingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ListRecordingsByMeeting(context.Background(), &videov1.ListRecordingsByMeetingRequest{MeetingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListRecordingsByMeeting_InvalidTenantID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ListRecordingsByMeeting(context.Background(), &videov1.ListRecordingsByMeetingRequest{MeetingID: uuid.New().String(), TenantID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetRecordingStatus_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetRecordingStatus(context.Background(), &videov1.GetRecordingStatusRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CleanupExpiredRecording_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.CleanupExpiredRecording(context.Background(), &videov1.CleanupExpiredRecordingRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ConfirmInitiatorConsent_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{RecordingID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ConfirmInitiatorConsent_InvalidUserID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{RecordingID: uuid.New().String(), UserID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ConfirmInitiatorConsent_InvalidTenantID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{RecordingID: uuid.New().String(), UserID: someUserID, TenantID: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetRecordingDownloadURL_InvalidRecordingID", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetRecordingDownloadURL(callCtxAuth(uuid.New(), uuid.New()), &videov1.GetRecordingDownloadURLRequest{RecordingId: "bogus"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetRecordingDownloadURL_MissingUserInCtx", func(t *testing.T) {
		ts := newTestVideoCallServer()
		_, err := ts.srv.GetRecordingDownloadURL(context.Background(), &videov1.GetRecordingDownloadURLRequest{RecordingId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
}

// ---------------------------------------------------------------------------
// Call lifecycle
// ---------------------------------------------------------------------------

func TestCallLifecycle_CreateJoinGetEnd(t *testing.T) {
	ts := newTestVideoCallServer()
	initiator := uuid.New()
	target := uuid.New()
	tenantID := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{
		InitiatorId:    initiator.String(),
		ParticipantIds: []string{target.String()},
		CallType:       videov1.CallType_CALL_TYPE_ONE_TO_ONE,
	})
	require.NoError(t, err)
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_RINGING, created.Status)
	callID := created.Id

	joinResp, err := ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: callID, UserId: target.String()})
	require.NoError(t, err)
	assert.NotEmpty(t, joinResp.Token)
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_ACTIVE, joinResp.Call.Status)
	assert.Len(t, joinResp.Call.Participants, 2)

	got, err := ts.srv.GetCall(context.Background(), &videov1.GetCallRequest{CallId: callID})
	require.NoError(t, err)
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_ACTIVE, got.Status)

	_, err = ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: callID, UserId: initiator.String()})
	require.NoError(t, err)

	ended, err := ts.srv.GetCall(context.Background(), &videov1.GetCallRequest{CallId: callID})
	require.NoError(t, err)
	assert.Equal(t, videov1.CallStatus_CALL_STATUS_ENDED, ended.Status)
	require.NotNil(t, ended.DurationSeconds)

	// EndCall is idempotent — ending an already-ended call is a no-op, not an error.
	_, err = ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: callID, UserId: initiator.String()})
	require.NoError(t, err)
}

func TestJoinCall_MaxParticipantsRejected(t *testing.T) {
	ts := newTestVideoCallServer()
	initiator := uuid.New()
	target1 := uuid.New()
	target2 := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(uuid.New()), &videov1.CreateCallRequest{
		InitiatorId:    initiator.String(),
		ParticipantIds: []string{target1.String()},
		CallType:       videov1.CallType_CALL_TYPE_ONE_TO_ONE,
	})
	require.NoError(t, err)

	_, err = ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: created.Id, UserId: target1.String()})
	require.NoError(t, err)

	// One-to-one calls cap at MaxOneToOneParticipants=2; initiator+target1 already fill it.
	_, err = ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: created.Id, UserId: target2.String()})
	requireGRPCCode(t, err, codes.ResourceExhausted)
}

func TestJoinCall_EndedCallRejected(t *testing.T) {
	ts := newTestVideoCallServer()
	initiator := uuid.New()
	target := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(uuid.New()), &videov1.CreateCallRequest{
		InitiatorId:    initiator.String(),
		ParticipantIds: []string{target.String()},
		CallType:       videov1.CallType_CALL_TYPE_ONE_TO_ONE,
	})
	require.NoError(t, err)

	_, err = ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: created.Id, UserId: initiator.String()})
	require.NoError(t, err)

	_, err = ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: created.Id, UserId: target.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGetCall_NotFound(t *testing.T) {
	ts := newTestVideoCallServer()
	_, err := ts.srv.GetCall(context.Background(), &videov1.GetCallRequest{CallId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateCall_LiveKitNotConfigured(t *testing.T) {
	callRepo := newCallMockRepo()
	videoSvc := video.NewService(callRepo, nil) // nil RoomManager -> disabled mode
	srv := NewVideoGRPCServer(videoSvc, nil, nil, nil, nil, nil, nil, "")

	_, err := srv.CreateCall(callCtxTenant(uuid.New()), &videov1.CreateCallRequest{
		InitiatorId:    uuid.New().String(),
		ParticipantIds: []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.Unavailable)
}

func TestListActiveCalls_FiltersOtherUsersAndEnded(t *testing.T) {
	ts := newTestVideoCallServer()
	userA := uuid.New()
	userB := uuid.New()

	callA, err := ts.srv.CreateCall(callCtxTenant(uuid.New()), &videov1.CreateCallRequest{InitiatorId: userA.String(), ParticipantIds: []string{uuid.New().String()}, CallType: videov1.CallType_CALL_TYPE_ONE_TO_ONE})
	require.NoError(t, err)
	_, err = ts.srv.CreateCall(callCtxTenant(uuid.New()), &videov1.CreateCallRequest{InitiatorId: userB.String(), ParticipantIds: []string{uuid.New().String()}, CallType: videov1.CallType_CALL_TYPE_ONE_TO_ONE})
	require.NoError(t, err)

	resp, err := ts.srv.ListActiveCalls(context.Background(), &videov1.ListActiveCallsRequest{UserId: userA.String()})
	require.NoError(t, err)
	require.Len(t, resp.Calls, 1)
	assert.Equal(t, callA.Id, resp.Calls[0].Id)

	_, err = ts.srv.EndCall(context.Background(), &videov1.EndCallRequest{CallId: callA.Id, UserId: userA.String()})
	require.NoError(t, err)

	resp2, err := ts.srv.ListActiveCalls(context.Background(), &videov1.ListActiveCallsRequest{UserId: userA.String()})
	require.NoError(t, err)
	assert.Empty(t, resp2.Calls)
}

// ---------------------------------------------------------------------------
// Recording — start/stop/consent
// ---------------------------------------------------------------------------

func TestStartRecording_CallContext_Success(t *testing.T) {
	ts := newTestVideoCallServer()
	initiator := uuid.New()
	target := uuid.New()
	tenantID := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{
		InitiatorId:    initiator.String(),
		ParticipantIds: []string{target.String()},
		CallType:       videov1.CallType_CALL_TYPE_ONE_TO_ONE,
	})
	require.NoError(t, err)
	_, err = ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: created.Id, UserId: target.String()})
	require.NoError(t, err)

	// Bypass the participant-consent gate (tested separately at the recording-service
	// level) so this test proves the call-context participant snapshot wiring only.
	zero := 0
	ts.recRepo.pendingOverride = &zero

	rec, err := ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{
		UserId: initiator.String(),
		CallId: &created.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_ACTIVE, rec.Status)
	require.NotNil(t, rec.StartedBy)
	assert.Equal(t, initiator.String(), *rec.StartedBy)
}

func TestStartRecording_MeetingContext_Success(t *testing.T) {
	ts := newTestVideoCallServer()
	tenantID := uuid.New()
	organizer := uuid.New()
	starter := uuid.New()

	meetingResp := seedMeeting(t, ts.srv, ts.meetingRepo, tenantID, organizer)

	zero := 0
	ts.recRepo.pendingOverride = &zero

	rec, err := ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{
		UserId:    starter.String(),
		MeetingId: &meetingResp.Id,
	})
	require.NoError(t, err)
	require.NotNil(t, rec.MeetingId)
	assert.Equal(t, meetingResp.Id, *rec.MeetingId)
}

func TestStartRecording_ConsentPendingByDefault(t *testing.T) {
	ts := newTestVideoCallServer()
	tenantID := uuid.New()
	initiator := uuid.New()
	target := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{
		InitiatorId:    initiator.String(),
		ParticipantIds: []string{target.String()},
		CallType:       videov1.CallType_CALL_TYPE_ONE_TO_ONE,
	})
	require.NoError(t, err)
	_, err = ts.srv.JoinCall(context.Background(), &videov1.JoinCallRequest{CallId: created.Id, UserId: target.String()})
	require.NoError(t, err)

	// No consents pre-populated -> every participant is "pending" by default.
	_, err = ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{
		UserId: initiator.String(),
		CallId: &created.Id,
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestStartRecording_EgressNotConfigured(t *testing.T) {
	callRepo := newCallMockRepo()
	roomMgr := newCallMockRoomManager()
	videoSvc := video.NewService(callRepo, roomMgr)
	recRepo := newRecordingMockRepo()
	recSvc := recording.NewService(recRepo, nil, "", recording.S3Config{}) // nil EgressManager -> disabled
	srv := NewVideoGRPCServer(videoSvc, nil, recSvc, nil, nil, nil, roomMgr, "")

	tenantID := uuid.New()
	initiator := uuid.New()
	created, err := srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{InitiatorId: initiator.String(), ParticipantIds: []string{uuid.New().String()}, CallType: videov1.CallType_CALL_TYPE_ONE_TO_ONE})
	require.NoError(t, err)

	_, err = srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{UserId: initiator.String(), CallId: &created.Id})
	requireGRPCCode(t, err, codes.Unavailable)
}

func TestStopRecording_AuthorizedInitiator(t *testing.T) {
	ts := newTestVideoCallServer()
	tenantID := uuid.New()
	initiator := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{InitiatorId: initiator.String(), ParticipantIds: []string{uuid.New().String()}, CallType: videov1.CallType_CALL_TYPE_ONE_TO_ONE})
	require.NoError(t, err)
	zero := 0
	ts.recRepo.pendingOverride = &zero
	rec, err := ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{UserId: initiator.String(), CallId: &created.Id})
	require.NoError(t, err)

	stopped, err := ts.srv.StopRecording(callCtxAuth(tenantID, initiator), &videov1.StopRecordingRequest{RecordingId: rec.Id})
	require.NoError(t, err)
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_PROCESSING, stopped.Status)
	assert.True(t, ts.egress.stopCalled)

	// Second stop: no longer active -> ErrRecordingNotActive.
	_, err = ts.srv.StopRecording(callCtxAuth(tenantID, initiator), &videov1.StopRecordingRequest{RecordingId: rec.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestStopRecording_AuthorizedMeetingHost(t *testing.T) {
	ts := newTestVideoCallServer()
	tenantID := uuid.New()
	organizer := uuid.New()
	starter := uuid.New()

	meetingResp := seedMeeting(t, ts.srv, ts.meetingRepo, tenantID, organizer)
	zero := 0
	ts.recRepo.pendingOverride = &zero
	rec, err := ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{UserId: starter.String(), MeetingId: &meetingResp.Id})
	require.NoError(t, err)

	// Organizer is not the initiator (starter is) but is host -> authorized via IsHostOrCoHost.
	_, err = ts.srv.StopRecording(callCtxAuth(tenantID, organizer), &videov1.StopRecordingRequest{RecordingId: rec.Id})
	require.NoError(t, err)
}

func TestStopRecording_UnauthorizedCallerDenied(t *testing.T) {
	ts := newTestVideoCallServer()
	tenantID := uuid.New()
	initiator := uuid.New()
	stranger := uuid.New()

	created, err := ts.srv.CreateCall(callCtxTenant(tenantID), &videov1.CreateCallRequest{InitiatorId: initiator.String(), ParticipantIds: []string{uuid.New().String()}, CallType: videov1.CallType_CALL_TYPE_ONE_TO_ONE})
	require.NoError(t, err)
	zero := 0
	ts.recRepo.pendingOverride = &zero
	rec, err := ts.srv.StartRecording(callCtxTenant(tenantID), &videov1.StartRecordingRequest{UserId: initiator.String(), CallId: &created.Id})
	require.NoError(t, err)

	_, err = ts.srv.StopRecording(callCtxAuth(tenantID, stranger), &videov1.StopRecordingRequest{RecordingId: rec.Id})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestSetAndGetRecordingConsent(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusActive, CreatedAt: time.Now()}
	userID := uuid.New()

	_, err := ts.srv.SetRecordingConsent(context.Background(), &videov1.SetRecordingConsentRequest{RecordingId: recID.String(), UserId: userID.String(), Consented: true})
	require.NoError(t, err)

	resp, err := ts.srv.GetRecordingConsent(context.Background(), &videov1.GetRecordingConsentRequest{RecordingId: recID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Consents, 1)
	assert.True(t, resp.Consents[0].Consented)
	// GetConsentStatus is called with participantIDs=nil here, which is documented to
	// always report AllConsented=false (see Service.GetConsentStatus doc-comment) —
	// not a bug, but easy to misread as one, so pinned explicitly.
	assert.False(t, resp.AllConsented)
}

func TestGetRecordingConsents_ReflectsAllConsentedFlag(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusActive, CreatedAt: time.Now()}
	user1 := uuid.New()

	_, err := ts.srv.SetRecordingConsent(context.Background(), &videov1.SetRecordingConsentRequest{RecordingId: recID.String(), UserId: user1.String(), Consented: true})
	require.NoError(t, err)

	resp, err := ts.srv.GetRecordingConsents(context.Background(), &videov1.GetRecordingConsentsRequest{RecordingID: recID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Consents, 1)
	assert.True(t, resp.AllConsented)

	_, err = ts.srv.SetRecordingConsent(context.Background(), &videov1.SetRecordingConsentRequest{RecordingId: recID.String(), UserId: uuid.New().String(), Consented: false})
	require.NoError(t, err)
	resp2, err := ts.srv.GetRecordingConsents(context.Background(), &videov1.GetRecordingConsentsRequest{RecordingID: recID.String()})
	require.NoError(t, err)
	assert.False(t, resp2.AllConsented)
}

func TestTagRecordingWithConsents_Success(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, CreatedAt: time.Now()}
	userID := uuid.New()

	_, err := ts.srv.TagRecordingWithConsents(context.Background(), &videov1.TagRecordingWithConsentsRequest{
		RecordingID: recID.String(),
		Snapshot:    []*videov1.ConsentSnapshotEntry{{UserID: userID.String(), DisplayName: "Ada"}},
	})
	require.NoError(t, err)
	assert.Len(t, ts.recRepo.recordings[recID].ConsentSnapshot, 1)
}

func TestUpdateRecordingMetadata_SuccessAndInvalidStatus(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusProcessing, CreatedAt: time.Now()}

	fileURL := "https://example.com/rec.mp4"
	resp, err := ts.srv.UpdateRecordingMetadata(context.Background(), &videov1.UpdateRecordingMetadataRequest{
		RecordingID: recID.String(),
		FileURL:     &fileURL,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.FileUrl)
	assert.Equal(t, fileURL, *resp.FileUrl)

	bogusStatus := "bogus-status"
	_, err = ts.srv.UpdateRecordingMetadata(context.Background(), &videov1.UpdateRecordingMetadataRequest{
		RecordingID: recID.String(),
		Status:      &bogusStatus,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListRecordingsByMeeting_PaginationAndTenant(t *testing.T) {
	ts := newTestVideoCallServer()
	meetingID := uuid.New()
	tenantID := uuid.New()
	for range 3 {
		id := uuid.New()
		ts.recRepo.recordings[id] = &recording.Recording{ID: id, MeetingID: &meetingID, TenantID: tenantID, Status: recording.RecordingStatusCompleted, CreatedAt: time.Now()}
	}

	resp, err := ts.srv.ListRecordingsByMeeting(context.Background(), &videov1.ListRecordingsByMeetingRequest{
		MeetingID: meetingID.String(),
		TenantID:  tenantID.String(),
		Page:      1,
		PageSize:  2,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Recordings, 2)
	assert.Equal(t, int32(3), resp.Total)
}

func TestGetRecordingStatus_SuccessAndNotFound(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusCompleted, CreatedAt: time.Now()}

	resp, err := ts.srv.GetRecordingStatus(context.Background(), &videov1.GetRecordingStatusRequest{RecordingID: recID.String()})
	require.NoError(t, err)
	assert.Equal(t, videov1.RecordingStatus_RECORDING_STATUS_COMPLETED, resp.Status)

	_, err = ts.srv.GetRecordingStatus(context.Background(), &videov1.GetRecordingStatusRequest{RecordingID: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCleanupExpiredRecording_ExpiredAndNotExpired(t *testing.T) {
	ts := newTestVideoCallServer()
	expiredID := uuid.New()
	past := time.Now().Add(-time.Hour)
	ts.recRepo.recordings[expiredID] = &recording.Recording{ID: expiredID, Status: recording.RecordingStatusCompleted, RetentionExpiresAt: &past, CreatedAt: time.Now()}

	_, err := ts.srv.CleanupExpiredRecording(context.Background(), &videov1.CleanupExpiredRecordingRequest{RecordingID: expiredID.String()})
	require.NoError(t, err)
	_, stillThere := ts.recRepo.recordings[expiredID]
	assert.False(t, stillThere)

	notExpiredID := uuid.New()
	future := time.Now().Add(time.Hour)
	ts.recRepo.recordings[notExpiredID] = &recording.Recording{ID: notExpiredID, Status: recording.RecordingStatusCompleted, RetentionExpiresAt: &future, CreatedAt: time.Now()}
	_, err = ts.srv.CleanupExpiredRecording(context.Background(), &videov1.CleanupExpiredRecordingRequest{RecordingID: notExpiredID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestConfirmInitiatorConsent_SuccessAndNotFound(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, CreatedAt: time.Now()}

	resp, err := ts.srv.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{
		RecordingID: recID.String(),
		UserID:      uuid.New().String(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Stamped)
	require.NotNil(t, ts.recRepo.recordings[recID].PreRecordingConsentAt)

	_, err = ts.srv.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{
		RecordingID: uuid.New().String(),
		UserID:      uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// GetRecordingDownloadURL — the participant-ACL check the unit's notes ask us
// to verify. It IS present (Service.GetRecordingDownloadURL checks
// GetRecordingParticipants before presigning) — these tests prove that, and
// prove the failure modes around it. See journal for the contrasting finding
// on DeleteRecording, which has no equivalent check.
// ---------------------------------------------------------------------------

func TestGetRecordingDownloadURL_Success(t *testing.T) {
	// minio-go's PresignedGetObject auto-discovers the bucket region via a real
	// GetBucketLocation HTTP call when no Region is configured (matches production:
	// S3Config carries no Region field either) — a fake S3-shaped HTTP server stands
	// in for MinIO so this exercises real request signing without live infra.
	fakeS3 := newFakeS3LocationServer(t)

	recRepo := newRecordingMockRepo()
	egress := newCallMockEgressManager()
	recSvc := recording.NewService(recRepo, egress, "https://template.example.com", recording.S3Config{
		Endpoint:  fakeS3,
		AccessKey: "mock-access-key",
		Secret:    "mock-secret-key",
		Bucket:    "recordings",
	})
	srv := NewVideoGRPCServer(nil, nil, recSvc, nil, nil, nil, nil, "")

	recID := uuid.New()
	fileURL := "recordings/rec-1.mp4"
	caller := uuid.New()
	recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusCompleted, FileURL: &fileURL, CreatedAt: time.Now()}
	recRepo.participantUsers[recID] = []uuid.UUID{caller}

	resp, err := srv.GetRecordingDownloadURL(callCtxAuth(uuid.New(), caller), &videov1.GetRecordingDownloadURLRequest{RecordingId: recID.String()})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Url)
	require.NotNil(t, resp.ExpiresAt)
}

func TestGetRecordingDownloadURL_NotCompletedRejected(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	caller := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusProcessing, CreatedAt: time.Now()}
	ts.recRepo.participantUsers[recID] = []uuid.UUID{caller}

	_, err := ts.srv.GetRecordingDownloadURL(callCtxAuth(uuid.New(), caller), &videov1.GetRecordingDownloadURLRequest{RecordingId: recID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGetRecordingDownloadURL_MissingFileURLRejected(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	caller := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusCompleted, CreatedAt: time.Now()}
	ts.recRepo.participantUsers[recID] = []uuid.UUID{caller}

	_, err := ts.srv.GetRecordingDownloadURL(callCtxAuth(uuid.New(), caller), &videov1.GetRecordingDownloadURLRequest{RecordingId: recID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGetRecordingDownloadURL_NonParticipantDenied(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	fileURL := "recordings/rec-1.mp4"
	participant := uuid.New()
	stranger := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusCompleted, FileURL: &fileURL, CreatedAt: time.Now()}
	ts.recRepo.participantUsers[recID] = []uuid.UUID{participant}

	_, err := ts.srv.GetRecordingDownloadURL(callCtxAuth(uuid.New(), stranger), &videov1.GetRecordingDownloadURLRequest{RecordingId: recID.String()})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

// TestDeleteRecording_NoParticipantOrInitiatorCheck documents the finding required
// by this unit's notes: unlike StopRecording (initiator-or-host authz) and
// GetRecordingDownloadURL (participant ACL), DeleteRecording performs NO check at
// all — any recording ID a caller can guess/enumerate within the tenant can be
// soft-deleted by ANY user, participant or not. The handler comment
// "Authorization handled at gateway level" is false: HandleDeleteRecording in
// route_video.go has no RequirePermission wrapper and does not check req.UserId
// against the recording's participants either. Documented per this unit's scope
// ("Befund fuers Journal, nicht etwas, das diese Unit nebenbei einbaut") — not
// fixed here.
func TestDeleteRecording_NoParticipantOrInitiatorCheck(t *testing.T) {
	ts := newTestVideoCallServer()
	recID := uuid.New()
	initiator := uuid.New()
	stranger := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, Status: recording.RecordingStatusCompleted, StartedBy: &initiator, CreatedAt: time.Now()}
	// stranger is never added to participantUsers[recID] — not a participant, not the initiator.

	_, err := ts.srv.DeleteRecording(context.Background(), &videov1.DeleteRecordingRequest{
		RecordingId: recID.String(),
		UserId:      stranger.String(),
	})
	require.NoError(t, err, "gap: a non-participant successfully deleted someone else's recording")
	assert.Equal(t, recording.RecordingStatusFailed, ts.recRepo.recordings[recID].Status)
}
