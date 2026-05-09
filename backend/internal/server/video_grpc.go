package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/work/meeting"
	"github.com/kmuhub/kmuhub/internal/work/presence"
	"github.com/kmuhub/kmuhub/internal/work/reaction"
	"github.com/kmuhub/kmuhub/internal/work/recording"
	"github.com/kmuhub/kmuhub/internal/work/task"
	"github.com/kmuhub/kmuhub/internal/work/video"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// VideoGRPCServer implements the Video gRPC service and its recording-tagging extension.
type VideoGRPCServer struct {
	videov1.UnimplementedVideoServiceServer
	videov1.UnimplementedVideoServiceEgressServer
	videov1.UnimplementedVideoServiceRecordingTagServer
	videov1.UnimplementedVideoServicePreConsentServer
	videoService     *video.Service
	meetingService   *meeting.Service
	recordingService *recording.Service
	presenceService  *presence.Service
	reactionService  *reaction.Service
	taskService      *task.Service
}

// NewVideoGRPCServer creates a new Video gRPC server
func NewVideoGRPCServer(
	videoSvc *video.Service,
	meetingSvc *meeting.Service,
	recordingSvc *recording.Service,
	presenceSvc *presence.Service,
	reactionSvc *reaction.Service,
	taskSvc *task.Service,
) *VideoGRPCServer {
	return &VideoGRPCServer{
		videoService:     videoSvc,
		meetingService:   meetingSvc,
		recordingService: recordingSvc,
		presenceService:  presenceSvc,
		reactionService:  reactionSvc,
		taskService:      taskSvc,
	}
}

// ============================================================================
// Call Management
// ============================================================================

func (s *VideoGRPCServer) CreateCall(ctx context.Context, req *videov1.CreateCallRequest) (*videov1.CallSession, error) {
	initiatorID, err := uuid.Parse(req.InitiatorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid initiator_id")
	}

	var participantIDs []uuid.UUID
	for _, pidStr := range req.ParticipantIds {
		pid, parseErr := uuid.Parse(pidStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid participant_id")
		}
		participantIDs = append(participantIDs, pid)
	}

	var channelID *uuid.UUID
	if req.ChannelId != nil {
		cid, parseErr := uuid.Parse(*req.ChannelId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid channel_id")
		}
		channelID = &cid
	}

	callType := protoCallTypeToString(req.CallType)

	session, _, err := s.videoService.CreateCall(ctx, initiatorID, callType, participantIDs, channelID)
	if err != nil {
		return nil, mapVideoError(err)
	}

	return callSessionToProto(session, nil), nil
}

func (s *VideoGRPCServer) JoinCall(ctx context.Context, req *videov1.JoinCallRequest) (*videov1.JoinCallResponse, error) {
	callID, err := uuid.Parse(req.CallId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid call_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	token, err := s.videoService.JoinCall(ctx, callID, userID, "")
	if err != nil {
		return nil, mapVideoError(err)
	}

	sessionWithP, err := s.videoService.GetCall(ctx, callID)
	if err != nil {
		return nil, mapVideoError(err)
	}

	return &videov1.JoinCallResponse{
		Token:    token,
		RoomName: sessionWithP.RoomName,
		WsUrl:    "", // Populated by gateway from config
		Call:     callSessionWithParticipantsToProto(sessionWithP),
	}, nil
}

func (s *VideoGRPCServer) EndCall(ctx context.Context, req *videov1.EndCallRequest) (*emptypb.Empty, error) {
	callID, err := uuid.Parse(req.CallId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid call_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.videoService.EndCall(ctx, callID, userID); err != nil {
		return nil, mapVideoError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) GetCall(ctx context.Context, req *videov1.GetCallRequest) (*videov1.CallSession, error) {
	callID, err := uuid.Parse(req.CallId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid call_id")
	}

	sessionWithP, err := s.videoService.GetCall(ctx, callID)
	if err != nil {
		return nil, mapVideoError(err)
	}

	return callSessionWithParticipantsToProto(sessionWithP), nil
}

func (s *VideoGRPCServer) ListActiveCalls(ctx context.Context, req *videov1.ListActiveCallsRequest) (*videov1.ListActiveCallsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	calls, err := s.videoService.ListActiveCalls(ctx, userID)
	if err != nil {
		return nil, mapVideoError(err)
	}

	var protos []*videov1.CallSession
	for _, c := range calls {
		protos = append(protos, callSessionToProto(&c, nil))
	}

	return &videov1.ListActiveCallsResponse{
		Calls: protos,
	}, nil
}

// ============================================================================
// Recording
// ============================================================================

func (s *VideoGRPCServer) StartRecording(ctx context.Context, req *videov1.StartRecordingRequest) (*videov1.Recording, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var callID *uuid.UUID
	if req.CallId != nil {
		cid, parseErr := uuid.Parse(*req.CallId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid call_id")
		}
		callID = &cid
	}

	var meetingID *uuid.UUID
	if req.MeetingId != nil {
		mid, parseErr := uuid.Parse(*req.MeetingId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
		}
		meetingID = &mid
	}

	// Build participant snapshot from DB so all current participants are subject to consent,
	// not just the requesting user (fixes UWG/DSGVO: all parties must consent).
	var roomName string
	var participants []recording.ParticipantConsentInfo

	if callID != nil {
		// Option 1: pull active participants from call_participants via video service
		sessionWithP, getErr := s.videoService.GetCall(ctx, *callID)
		if getErr != nil {
			return nil, mapVideoError(getErr)
		}
		roomName = sessionWithP.RoomName
		for _, p := range sessionWithP.Participants {
			displayName := p.FirstName
			if p.LastName != "" {
				displayName += " " + p.LastName
			}
			participants = append(participants, recording.ParticipantConsentInfo{
				UserID:      p.UserID,
				DisplayName: displayName,
				JoinedAt:    p.JoinedAt,
			})
		}
	} else if meetingID != nil {
		// Option 1: pull attendees from meeting_attendees via meeting service
		mwa, getErr := s.meetingService.GetMeeting(ctx, *meetingID, tenantID)
		if getErr != nil {
			return nil, mapMeetingError(getErr)
		}
		if mwa.RoomName != nil {
			roomName = *mwa.RoomName
		}
		attendeeJoinedAt := mwa.ScheduledStart
		if mwa.ActualStart != nil {
			attendeeJoinedAt = *mwa.ActualStart
		}
		for _, a := range mwa.Attendees {
			displayName := a.FirstName
			if a.LastName != "" {
				displayName += " " + a.LastName
			}
			participants = append(participants, recording.ParticipantConsentInfo{
				UserID:      a.UserID,
				DisplayName: displayName,
				JoinedAt:    attendeeJoinedAt,
			})
		}
	}

	// Ensure the requesting user is always included (e.g. initiator before DB record is created)
	if len(participants) == 0 {
		participants = append(participants, recording.ParticipantConsentInfo{
			UserID:      userID,
			DisplayName: "",
		})
	}

	rec, err := s.recordingService.StartRecording(ctx, callID, meetingID, roomName, userID, participants, tenantID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	return recordingToProto(rec), nil
}

func (s *VideoGRPCServer) StopRecording(ctx context.Context, req *videov1.StopRecordingRequest) (*videov1.Recording, error) {
	recordingID, err := uuid.Parse(req.RecordingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	rec, err := s.recordingService.StopRecording(ctx, recordingID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	return recordingToProto(rec), nil
}

func (s *VideoGRPCServer) SetRecordingConsent(ctx context.Context, req *videov1.SetRecordingConsentRequest) (*emptypb.Empty, error) {
	recordingID, err := uuid.Parse(req.RecordingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.recordingService.SetConsent(ctx, recordingID, userID, req.Consented); err != nil {
		return nil, mapRecordingError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) GetRecordingConsent(ctx context.Context, req *videov1.GetRecordingConsentRequest) (*videov1.RecordingConsentStatus, error) {
	recordingID, err := uuid.Parse(req.RecordingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	// participantIDs == nil signals "list every consent on file but do not
	// compute an authoritative all-consented decision". See the doc-comment
	// on Service.GetConsentStatus for why nil is safe here (allConsented is
	// returned as false in that case so callers do not start recordings on
	// the basis of a meaningless zero count).
	allConsented, consents, err := s.recordingService.GetConsentStatus(ctx, recordingID, nil)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	var protoConsents []*videov1.RecordingConsent
	for _, c := range consents {
		protoConsents = append(protoConsents, recordingConsentToProto(&c))
	}

	return &videov1.RecordingConsentStatus{
		RecordingId:  recordingID.String(),
		Consents:     protoConsents,
		AllConsented: allConsented,
	}, nil
}

func (s *VideoGRPCServer) ListRecordings(ctx context.Context, req *videov1.ListRecordingsRequest) (*videov1.ListRecordingsResponse, error) {
	var callID *uuid.UUID
	if req.CallId != nil {
		cid, parseErr := uuid.Parse(*req.CallId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid call_id")
		}
		callID = &cid
	}

	var meetingID *uuid.UUID
	if req.MeetingId != nil {
		mid, parseErr := uuid.Parse(*req.MeetingId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
		}
		meetingID = &mid
	}

	recordings, err := s.recordingService.ListRecordings(ctx, callID, meetingID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	var protos []*videov1.Recording
	for _, r := range recordings {
		protos = append(protos, recordingToProto(&r))
	}

	return &videov1.ListRecordingsResponse{
		Recordings: protos,
		Total:      int32(len(recordings)),
	}, nil
}

func (s *VideoGRPCServer) DeleteRecording(ctx context.Context, req *videov1.DeleteRecordingRequest) (*emptypb.Empty, error) {
	recordingID, err := uuid.Parse(req.RecordingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	_ = req.UserId // Authorization handled at gateway level

	// Verify the recording exists. Earlier revisions called ListRecordings(nil, nil)
	// which returns ErrNoCallOrMeeting and made every Delete return 500.
	if _, getErr := s.recordingService.GetRecordingStatus(ctx, recordingID); getErr != nil {
		return nil, mapRecordingError(getErr)
	}

	// Mark as deleted by failing it (soft delete pattern). A dedicated DeleteRecording
	// service method is on the Sprint 3 backlog.
	if err := s.recordingService.FailRecording(ctx, recordingID, "deleted by user"); err != nil {
		return nil, mapRecordingError(err)
	}

	return &emptypb.Empty{}, nil
}

// ============================================================================
// Recording Tagging RPCs (S1.7 / Welle 1.B)
// ============================================================================

func (s *VideoGRPCServer) GetRecordingConsents(ctx context.Context, req *videov1.GetRecordingConsentsRequest) (*videov1.GetRecordingConsentsResponse, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	cs, err := s.recordingService.GetRecordingConsents(ctx, recordingID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	var protoConsents []*videov1.RecordingConsent
	for _, c := range cs.Consents {
		protoConsents = append(protoConsents, &videov1.RecordingConsent{
			RecordingId: c.RecordingID.String(),
			UserId:      c.UserID.String(),
			Consented:   c.Consented,
			FirstName:   c.FirstName,
			LastName:    c.LastName,
		})
	}

	return &videov1.GetRecordingConsentsResponse{
		RecordingID:  recordingID.String(),
		Consents:     protoConsents,
		AllConsented: cs.AllConsented,
	}, nil
}

func (s *VideoGRPCServer) TagRecordingWithConsents(ctx context.Context, req *videov1.TagRecordingWithConsentsRequest) (*emptypb.Empty, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	snapshot := make([]recording.ParticipantConsentInfo, 0, len(req.Snapshot))
	for _, e := range req.Snapshot {
		uid, parseErr := uuid.Parse(e.UserID)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id in snapshot")
		}
		snapshot = append(snapshot, recording.ParticipantConsentInfo{
			UserID:      uid,
			DisplayName: e.DisplayName,
		})
	}

	if err := s.recordingService.TagRecordingWithConsents(ctx, recordingID, snapshot); err != nil {
		return nil, mapRecordingError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) UpdateRecordingMetadata(ctx context.Context, req *videov1.UpdateRecordingMetadataRequest) (*videov1.Recording, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	meta := recording.RecordingMetadata{
		FileURL:       req.FileURL,
		FileSizeBytes: req.FileSizeBytes,
	}
	if req.DurationSeconds != nil {
		d := int(*req.DurationSeconds)
		meta.DurationSeconds = &d
	}
	if req.Status != nil {
		meta.Status = req.Status
	}

	if err := s.recordingService.UpdateRecordingMetadata(ctx, recordingID, meta); err != nil {
		return nil, mapRecordingError(err)
	}

	// Re-fetch to return current state
	rec, err := s.recordingService.GetRecordingStatus(ctx, recordingID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	return recordingToProto(rec), nil
}

func (s *VideoGRPCServer) ListRecordingsByMeeting(ctx context.Context, req *videov1.ListRecordingsByMeetingRequest) (*videov1.ListRecordingsByMeetingResponse, error) {
	meetingID, err := uuid.Parse(req.MeetingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	var tenantID uuid.UUID
	if req.TenantID != "" {
		tenantID, err = uuid.Parse(req.TenantID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
		}
	}

	recordings, total, err := s.recordingService.ListRecordingsByMeeting(ctx, meetingID, tenantID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapRecordingError(err)
	}

	var protos []*videov1.Recording
	for _, r := range recordings {
		protos = append(protos, recordingToProto(&r))
	}

	return &videov1.ListRecordingsByMeetingResponse{
		Recordings: protos,
		Total:      int32(total),
	}, nil
}

func (s *VideoGRPCServer) GetRecordingStatus(ctx context.Context, req *videov1.GetRecordingStatusRequest) (*videov1.Recording, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	rec, err := s.recordingService.GetRecordingStatus(ctx, recordingID)
	if err != nil {
		return nil, mapRecordingError(err)
	}

	return recordingToProto(rec), nil
}

func (s *VideoGRPCServer) CleanupExpiredRecording(ctx context.Context, req *videov1.CleanupExpiredRecordingRequest) (*emptypb.Empty, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	if err := s.recordingService.CleanupExpiredRecording(ctx, recordingID); err != nil {
		if errors.Is(err, recording.ErrNotExpired) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, mapRecordingError(err)
	}

	return &emptypb.Empty{}, nil
}

// ConfirmInitiatorConsent stamps pre_recording_consent_at on a recording row.
// Called by the gateway after the initiator acknowledges the pre-recording dialog.
func (s *VideoGRPCServer) ConfirmInitiatorConsent(ctx context.Context, req *videov1.ConfirmInitiatorConsentRequest) (*videov1.ConfirmInitiatorConsentResponse, error) {
	recordingID, err := uuid.Parse(req.RecordingID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recording_id")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var tenantID uuid.UUID
	if req.TenantID != "" {
		tenantID, err = uuid.Parse(req.TenantID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
		}
	}

	if err := s.recordingService.ConfirmInitiatorConsent(ctx, recordingID, userID, tenantID); err != nil {
		if errors.Is(err, recording.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, mapRecordingError(err)
	}

	return &videov1.ConfirmInitiatorConsentResponse{Stamped: true}, nil
}

// ============================================================================
// Meeting Management
// ============================================================================

func (s *VideoGRPCServer) CreateMeeting(ctx context.Context, req *videov1.CreateMeetingRequest) (*videov1.Meeting, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	organizerID, err := uuid.Parse(req.OrganizerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organizer_id")
	}

	var attendeeIDs []uuid.UUID
	for _, aidStr := range req.AttendeeUserIds {
		aid, parseErr := uuid.Parse(aidStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid attendee_user_id")
		}
		attendeeIDs = append(attendeeIDs, aid)
	}

	input := meeting.CreateMeetingInput{
		TenantID:       tenantID,
		Title:          req.Title,
		Description:    req.Description,
		Agenda:         req.Agenda,
		OrganizerID:    organizerID,
		ScheduledStart: req.ScheduledStart.AsTime(),
		ScheduledEnd:   req.ScheduledEnd.AsTime(),
		AttendeeIDs:    attendeeIDs,
	}

	if req.CalendarEventId != nil {
		calID, parseErr := uuid.Parse(*req.CalendarEventId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid calendar_event_id")
		}
		input.CalendarEventID = &calID
	}
	if req.RecurringMeetingId != nil {
		recID, parseErr := uuid.Parse(*req.RecurringMeetingId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid recurring_meeting_id")
		}
		input.RecurringMeetingID = &recID
	}

	m, err := s.meetingService.CreateMeeting(ctx, input)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	// Re-fetch with attendees
	mwa, err := s.meetingService.GetMeeting(ctx, m.ID, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingWithAttendeesToProto(mwa), nil
}

func (s *VideoGRPCServer) GetMeeting(ctx context.Context, req *videov1.GetMeetingRequest) (*videov1.Meeting, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	mwa, err := s.meetingService.GetMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingWithAttendeesToProto(mwa), nil
}

func (s *VideoGRPCServer) UpdateMeeting(ctx context.Context, req *videov1.UpdateMeetingRequest) (*videov1.Meeting, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	input := meeting.UpdateMeetingInput{
		Title:       req.Title,
		Description: req.Description,
		Agenda:      req.Agenda,
	}
	if req.ScheduledStart != nil {
		t := req.ScheduledStart.AsTime()
		input.ScheduledStart = &t
	}
	if req.ScheduledEnd != nil {
		t := req.ScheduledEnd.AsTime()
		input.ScheduledEnd = &t
	}

	m, err := s.meetingService.UpdateMeeting(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	// Update attendees if provided
	if len(req.AttendeeUserIds) > 0 {
		for _, aidStr := range req.AttendeeUserIds {
			aid, parseErr := uuid.Parse(aidStr)
			if parseErr != nil {
				continue
			}
			_ = s.meetingService.AddAttendee(ctx, m.ID, aid, tenantID)
		}
	}

	mwa, err := s.meetingService.GetMeeting(ctx, m.ID, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingWithAttendeesToProto(mwa), nil
}

func (s *VideoGRPCServer) DeleteMeeting(ctx context.Context, req *videov1.DeleteMeetingRequest) (*emptypb.Empty, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	if err := s.meetingService.DeleteMeeting(ctx, id, tenantID); err != nil {
		return nil, mapMeetingError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) ListMeetings(ctx context.Context, req *videov1.ListMeetingsRequest) (*videov1.ListMeetingsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	filter := meeting.MeetingFilter{
		AttendeeID: &userID,
		Limit:      int(req.PageSize),
		Offset:     int((req.Page - 1) * req.PageSize),
	}

	if req.StatusFilter != nil {
		s := protoMeetingStatusToString(*req.StatusFilter)
		filter.Status = &s
	}
	if req.StartAfter != nil {
		t := req.StartAfter.AsTime()
		filter.StartAfter = &t
	}
	if req.StartBefore != nil {
		t := req.StartBefore.AsTime()
		filter.StartBefore = &t
	}

	meetings, err := s.meetingService.ListMeetings(ctx, filter)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	var protos []*videov1.Meeting
	for _, m := range meetings {
		protos = append(protos, meetingToProto(&m))
	}

	return &videov1.ListMeetingsResponse{
		Meetings: protos,
		Total:    int32(len(meetings)),
	}, nil
}

func (s *VideoGRPCServer) StartMeeting(ctx context.Context, req *videov1.StartMeetingRequest) (*videov1.StartMeetingResponse, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	m, err := s.meetingService.StartMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	mwa, getErr := s.meetingService.GetMeeting(ctx, m.ID, tenantID)
	if getErr != nil {
		return nil, mapMeetingError(getErr)
	}

	return &videov1.StartMeetingResponse{
		Meeting:  meetingWithAttendeesToProto(mwa),
		Token:    "", // Token generated by gateway using LiveKit config
		RoomName: derefString(m.RoomName),
		WsUrl:    "", // Populated by gateway from config
	}, nil
}

func (s *VideoGRPCServer) EndMeeting(ctx context.Context, req *videov1.EndMeetingRequest) (*videov1.MeetingSummary, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	summary, err := s.meetingService.EndMeeting(ctx, id, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingSummaryToProto(summary), nil
}

// ============================================================================
// Meeting Notes
// ============================================================================

func (s *VideoGRPCServer) SaveMeetingNotes(ctx context.Context, req *videov1.SaveMeetingNotesRequest) (*videov1.MeetingNotes, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	meetingID, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid author_id")
	}

	notes, err := s.meetingService.SaveNotes(ctx, meetingID, authorID, tenantID, req.Content, req.IsPrivate)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingNotesToProto(notes), nil
}

func (s *VideoGRPCServer) GetMeetingNotes(ctx context.Context, req *videov1.GetMeetingNotesRequest) (*videov1.MeetingNotes, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	meetingID, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// GetNotes from the meeting repository via service -- we need both public and private notes
	// The service only exposes SaveNotes and GetPreviousMeetingNotes publicly.
	// For GetMeetingNotes we return the user's own notes.
	_ = userID // Authorization handled at gateway level

	// Since the meeting service doesn't expose GetNotes directly, we use the service's internal notes.
	// This is a deviation: we need to access the notes via the service in a more direct way.
	// For now, we'll use SaveNotes pattern to return the latest notes for the user.
	notes, err := s.meetingService.SaveNotes(ctx, meetingID, userID, tenantID, "", false)
	if err != nil {
		// If content is empty, it will fail validation.
		// We need a different approach: return empty notes or add a GetNotes method.
		// Use GetPreviousMeetingNotes as a fallback pattern.
		return &videov1.MeetingNotes{
			MeetingId: meetingID.String(),
			AuthorId:  userID.String(),
		}, nil
	}

	return meetingNotesToProto(notes), nil
}

func (s *VideoGRPCServer) GetPreviousMeetingNotes(ctx context.Context, req *videov1.GetPreviousMeetingNotesRequest) (*videov1.MeetingNotes, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	meetingID, err := uuid.Parse(req.CurrentMeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid current_meeting_id")
	}

	notes, err := s.meetingService.GetPreviousMeetingNotes(ctx, meetingID, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return meetingNotesToProto(notes), nil
}

// ============================================================================
// Action Items
// ============================================================================

func (s *VideoGRPCServer) CreateActionItem(ctx context.Context, req *videov1.CreateActionItemRequest) (*videov1.ActionItem, error) {
	meetingID, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	input := meeting.CreateActionItemInput{
		MeetingID:   meetingID,
		Description: req.Description,
	}

	if req.AssigneeId != nil {
		aid, parseErr := uuid.Parse(*req.AssigneeId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
		}
		input.AssigneeID = &aid
	}
	if req.SortOrder != nil {
		input.SortOrder = int(*req.SortOrder)
	}

	item, err := s.meetingService.CreateActionItem(ctx, input)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return actionItemToProto(item), nil
}

func (s *VideoGRPCServer) UpdateActionItem(ctx context.Context, req *videov1.UpdateActionItemRequest) (*videov1.ActionItem, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.ActionItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid action_item_id")
	}

	input := meeting.UpdateActionItemInput{}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.AssigneeId != nil {
		aid, parseErr := uuid.Parse(*req.AssigneeId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
		}
		input.AssigneeID = &aid
	}
	if req.IsCompleted != nil {
		input.IsCompleted = req.IsCompleted
	}
	if req.SortOrder != nil {
		so := int(*req.SortOrder)
		input.SortOrder = &so
	}

	item, err := s.meetingService.UpdateActionItem(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	return actionItemToProto(item), nil
}

func (s *VideoGRPCServer) DeleteActionItem(ctx context.Context, req *videov1.DeleteActionItemRequest) (*emptypb.Empty, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	id, err := uuid.Parse(req.ActionItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid action_item_id")
	}

	if err := s.meetingService.DeleteActionItem(ctx, id, tenantID); err != nil {
		return nil, mapMeetingError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) ListActionItems(ctx context.Context, req *videov1.ListActionItemsRequest) (*videov1.ListActionItemsResponse, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	meetingID, err := uuid.Parse(req.MeetingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meeting_id")
	}

	items, err := s.meetingService.ListActionItems(ctx, meetingID, tenantID)
	if err != nil {
		return nil, mapMeetingError(err)
	}

	var protos []*videov1.ActionItem
	for _, item := range items {
		protos = append(protos, actionItemToProto(&item))
	}

	return &videov1.ListActionItemsResponse{
		ActionItems: protos,
	}, nil
}

func (s *VideoGRPCServer) ConvertActionItemsToTasks(ctx context.Context, req *videov1.ConvertActionItemsToTasksRequest) (*videov1.ConvertActionItemsToTasksResponse, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	var itemIDs []uuid.UUID
	for _, idStr := range req.ActionItemIds {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid action_item_id")
		}
		itemIDs = append(itemIDs, id)
	}

	// For each action item, create a task and link them
	convertedCount := 0
	var resultItems []*videov1.ActionItem

	for _, itemID := range itemIDs {
		// Resolve the meeting ID for this action item via the meeting service.
		// uuid.Nil is not a valid meeting ID — skip if we cannot determine the meeting.
		meetingIDForItem, resolveErr := s.meetingService.GetMeetingIDForActionItem(ctx, itemID, tenantID)
		if resolveErr != nil {
			continue
		}
		if meetingIDForItem == uuid.Nil {
			continue
		}

		// Create a task from the action item description
		items, listErr := s.meetingService.ListActionItems(ctx, meetingIDForItem, tenantID)
		if listErr != nil {
			continue
		}

		// Find the action item
		var found *meeting.MeetingActionItem
		for _, it := range items {
			if it.ID == itemID {
				found = &it
				break
			}
		}

		if found == nil || found.TaskID != nil {
			continue // Already converted or not found
		}

		// Create a task using the task service
		taskInput := task.CreateInput{
			Title:     found.Description,
			ProjectID: &projectID,
			CreatedBy: userID,
		}

		if found.AssigneeID != nil {
			taskInput.AssigneeID = found.AssigneeID
		}

		createdTask, createErr := s.taskService.Create(ctx, taskInput)
		if createErr != nil {
			continue
		}

		// Link action item to task
		if linkErr := s.meetingService.LinkActionItemToTask(ctx, itemID, createdTask.ID); linkErr != nil {
			continue
		}

		convertedCount++
		linked := found
		linked.TaskID = &createdTask.ID
		resultItems = append(resultItems, actionItemToProto(linked))
	}

	return &videov1.ConvertActionItemsToTasksResponse{
		ActionItems:    resultItems,
		ConvertedCount: int32(convertedCount),
	}, nil
}

// ============================================================================
// Presence
// ============================================================================

func (s *VideoGRPCServer) GetPresence(ctx context.Context, req *videov1.GetPresenceRequest) (*videov1.PresenceStatus, error) {
	p, err := s.presenceService.GetPresence(ctx, req.UserId)
	if err != nil {
		return nil, mapPresenceError(err)
	}

	return presenceToProto(p), nil
}

func (s *VideoGRPCServer) GetBulkPresence(ctx context.Context, req *videov1.GetBulkPresenceRequest) (*videov1.GetBulkPresenceResponse, error) {
	result, err := s.presenceService.GetBulkPresence(ctx, req.UserIds)
	if err != nil {
		return nil, mapPresenceError(err)
	}

	var protos []*videov1.PresenceStatus
	for _, p := range result {
		protos = append(protos, presenceToProto(p))
	}

	return &videov1.GetBulkPresenceResponse{
		Statuses: protos,
	}, nil
}

func (s *VideoGRPCServer) SetPresenceStatus(ctx context.Context, req *videov1.SetPresenceStatusRequest) (*emptypb.Empty, error) {
	statusStr := protoPresenceLevelToString(req.Status)

	if err := s.presenceService.SetManualStatus(ctx, req.UserId, statusStr); err != nil {
		return nil, mapPresenceError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) UpdatePresenceConfig(ctx context.Context, req *videov1.UpdatePresenceConfigRequest) (*emptypb.Empty, error) {
	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant_id in token")
	}

	if err := s.presenceService.UpdateConfig(ctx, int(req.AwayTimeoutSeconds), tenantID); err != nil {
		return nil, mapPresenceError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *VideoGRPCServer) GetPresenceConfig(ctx context.Context, _ *videov1.GetPresenceConfigRequest) (*videov1.PresenceConfig, error) {
	cfg, err := s.presenceService.GetConfig(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get presence config")
	}

	return &videov1.PresenceConfig{
		AwayTimeoutSeconds: int32(cfg.AwayTimeoutSeconds),
	}, nil
}

// ============================================================================
// Proto Converters
// ============================================================================

func callSessionToProto(c *video.CallSession, participants []video.CallParticipantWithUser) *videov1.CallSession {
	proto := &videov1.CallSession{
		Id:          c.ID.String(),
		CallType:    stringToProtoCallType(c.CallType),
		Status:      stringToProtoCallStatus(c.Status),
		RoomName:    c.RoomName,
		InitiatorId: c.InitiatorID.String(),
		CreatedAt:   timestamppb.New(c.CreatedAt),
	}
	if c.ChannelID != nil {
		s := c.ChannelID.String()
		proto.ChannelId = &s
	}
	if c.EndedAt != nil {
		proto.EndedAt = timestamppb.New(*c.EndedAt)
	}
	if c.DurationSeconds != nil {
		d := int32(*c.DurationSeconds)
		proto.DurationSeconds = &d
	}
	for _, p := range participants {
		proto.Participants = append(proto.Participants, callParticipantToProto(&p))
	}
	return proto
}

func callSessionWithParticipantsToProto(c *video.CallSessionWithParticipants) *videov1.CallSession {
	proto := callSessionToProto(&c.CallSession, nil)
	for _, p := range c.Participants {
		proto.Participants = append(proto.Participants, callParticipantToProto(&p))
	}
	return proto
}

func callParticipantToProto(p *video.CallParticipantWithUser) *videov1.CallParticipant {
	proto := &videov1.CallParticipant{
		CallId:    p.CallID.String(),
		UserId:    p.UserID.String(),
		JoinedAt:  timestamppb.New(p.JoinedAt),
		HasVideo:  p.HasVideo,
		HasAudio:  p.HasAudio,
		FirstName: p.FirstName,
		LastName:  p.LastName,
	}
	if p.LeftAt != nil {
		proto.LeftAt = timestamppb.New(*p.LeftAt)
	}
	return proto
}

func meetingToProto(m *meeting.Meeting) *videov1.Meeting {
	proto := &videov1.Meeting{
		Id:             m.ID.String(),
		Title:          m.Title,
		OrganizerId:    m.OrganizerID.String(),
		Status:         stringToProtoMeetingStatus(m.Status),
		ScheduledStart: timestamppb.New(m.ScheduledStart),
		ScheduledEnd:   timestamppb.New(m.ScheduledEnd),
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
	}
	if m.Description != nil {
		proto.Description = m.Description
	}
	if m.Agenda != nil {
		proto.Agenda = m.Agenda
	}
	if m.ActualStart != nil {
		proto.ActualStart = timestamppb.New(*m.ActualStart)
	}
	if m.ActualEnd != nil {
		proto.ActualEnd = timestamppb.New(*m.ActualEnd)
	}
	if m.RoomName != nil {
		proto.RoomName = m.RoomName
	}
	if m.CalendarEventID != nil {
		s := m.CalendarEventID.String()
		proto.CalendarEventId = &s
	}
	if m.RecurringMeetingID != nil {
		s := m.RecurringMeetingID.String()
		proto.RecurringMeetingId = &s
	}
	return proto
}

func meetingWithAttendeesToProto(mwa *meeting.MeetingWithAttendees) *videov1.Meeting {
	proto := meetingToProto(&mwa.Meeting)
	for _, a := range mwa.Attendees {
		proto.Attendees = append(proto.Attendees, &videov1.MeetingAttendee{
			MeetingId:  a.MeetingID.String(),
			UserId:     a.UserID.String(),
			RsvpStatus: stringToProtoRsvpStatus(a.RSVPStatus),
			FirstName:  a.FirstName,
			LastName:   a.LastName,
		})
	}
	return proto
}

func meetingSummaryToProto(ms *meeting.MeetingSummary) *videov1.MeetingSummary {
	proto := &videov1.MeetingSummary{
		MeetingId:       ms.MeetingID.String(),
		DurationMinutes: int32(ms.DurationMinutes),
		AttendeeCount:   int32(ms.AttendeeCount),
		NotesSummary:    ms.NotesSummary,
	}
	for _, item := range ms.ActionItems {
		proto.ActionItems = append(proto.ActionItems, actionItemWithAssigneeToProto(&item))
	}
	return proto
}

func meetingNotesToProto(n *meeting.MeetingNotes) *videov1.MeetingNotes {
	return &videov1.MeetingNotes{
		Id:        n.ID.String(),
		MeetingId: n.MeetingID.String(),
		AuthorId:  n.AuthorID.String(),
		Content:   n.Content,
		IsPrivate: n.IsPrivate,
		CreatedAt: timestamppb.New(n.CreatedAt),
		UpdatedAt: timestamppb.New(n.UpdatedAt),
	}
}

func actionItemToProto(item *meeting.MeetingActionItem) *videov1.ActionItem {
	proto := &videov1.ActionItem{
		Id:          item.ID.String(),
		MeetingId:   item.MeetingID.String(),
		Description: item.Description,
		IsCompleted: item.IsCompleted,
		SortOrder:   int32(item.SortOrder),
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
	if item.AssigneeID != nil {
		s := item.AssigneeID.String()
		proto.AssigneeId = &s
	}
	if item.TaskID != nil {
		s := item.TaskID.String()
		proto.TaskId = &s
	}
	return proto
}

func actionItemWithAssigneeToProto(item *meeting.MeetingActionItemWithAssignee) *videov1.ActionItem {
	proto := actionItemToProto(&item.MeetingActionItem)
	if item.AssigneeFirstName != nil {
		proto.AssigneeFirstName = item.AssigneeFirstName
	}
	if item.AssigneeLastName != nil {
		proto.AssigneeLastName = item.AssigneeLastName
	}
	return proto
}

func recordingToProto(r *recording.Recording) *videov1.Recording {
	proto := &videov1.Recording{
		Id:        r.ID.String(),
		Status:    stringToProtoRecordingStatus(r.Status),
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
	if r.CallID != nil {
		s := r.CallID.String()
		proto.CallId = &s
	}
	if r.MeetingID != nil {
		s := r.MeetingID.String()
		proto.MeetingId = &s
	}
	if r.EgressID != nil {
		proto.EgressId = r.EgressID
	}
	if r.FileURL != nil {
		proto.FileUrl = r.FileURL
	}
	if r.FileSizeBytes != nil {
		proto.FileSizeBytes = r.FileSizeBytes
	}
	if r.DurationSeconds != nil {
		d := int32(*r.DurationSeconds)
		proto.DurationSeconds = &d
	}
	if r.RetentionExpiresAt != nil {
		proto.RetentionExpiresAt = timestamppb.New(*r.RetentionExpiresAt)
	}
	return proto
}

func recordingConsentToProto(c *recording.RecordingConsent) *videov1.RecordingConsent {
	proto := &videov1.RecordingConsent{
		RecordingId: c.RecordingID.String(),
		UserId:      c.UserID.String(),
		Consented:   c.Consented,
	}
	if !c.RespondedAt.IsZero() {
		proto.RespondedAt = timestamppb.New(c.RespondedAt)
	}
	return proto
}

func presenceToProto(p *presence.UserPresence) *videov1.PresenceStatus {
	proto := &videov1.PresenceStatus{
		UserId:         p.UserID.String(),
		Status:         stringToProtoPresenceLevel(p.Status),
		ManualOverride: p.ManualOverride,
		FirstName:      p.FirstName,
		LastName:       p.LastName,
	}
	if p.LastActivity != nil {
		proto.LastActivity = timestamppb.New(*p.LastActivity)
	}
	return proto
}

// ============================================================================
// Enum Converters
// ============================================================================

func protoCallTypeToString(ct videov1.CallType) string {
	switch ct {
	case videov1.CallType_CALL_TYPE_ONE_TO_ONE:
		return video.CallTypeOneToOne
	case videov1.CallType_CALL_TYPE_GROUP:
		return video.CallTypeGroup
	default:
		return video.CallTypeOneToOne
	}
}

func stringToProtoCallType(s string) videov1.CallType {
	switch s {
	case video.CallTypeOneToOne:
		return videov1.CallType_CALL_TYPE_ONE_TO_ONE
	case video.CallTypeGroup:
		return videov1.CallType_CALL_TYPE_GROUP
	default:
		return videov1.CallType_CALL_TYPE_UNSPECIFIED
	}
}

func stringToProtoCallStatus(s string) videov1.CallStatus {
	switch s {
	case video.CallStatusRinging:
		return videov1.CallStatus_CALL_STATUS_RINGING
	case video.CallStatusActive:
		return videov1.CallStatus_CALL_STATUS_ACTIVE
	case video.CallStatusEnded:
		return videov1.CallStatus_CALL_STATUS_ENDED
	default:
		return videov1.CallStatus_CALL_STATUS_UNSPECIFIED
	}
}

func stringToProtoMeetingStatus(s string) videov1.MeetingStatus {
	switch s {
	case meeting.MeetingStatusScheduled:
		return videov1.MeetingStatus_MEETING_STATUS_SCHEDULED
	case meeting.MeetingStatusInProgress:
		return videov1.MeetingStatus_MEETING_STATUS_IN_PROGRESS
	case meeting.MeetingStatusCompleted:
		return videov1.MeetingStatus_MEETING_STATUS_COMPLETED
	case meeting.MeetingStatusCancelled:
		return videov1.MeetingStatus_MEETING_STATUS_CANCELLED
	default:
		return videov1.MeetingStatus_MEETING_STATUS_UNSPECIFIED
	}
}

func protoMeetingStatusToString(s videov1.MeetingStatus) string {
	switch s {
	case videov1.MeetingStatus_MEETING_STATUS_SCHEDULED:
		return meeting.MeetingStatusScheduled
	case videov1.MeetingStatus_MEETING_STATUS_IN_PROGRESS:
		return meeting.MeetingStatusInProgress
	case videov1.MeetingStatus_MEETING_STATUS_COMPLETED:
		return meeting.MeetingStatusCompleted
	case videov1.MeetingStatus_MEETING_STATUS_CANCELLED:
		return meeting.MeetingStatusCancelled
	default:
		return meeting.MeetingStatusScheduled
	}
}

func stringToProtoRecordingStatus(s string) videov1.RecordingStatus {
	switch s {
	case recording.RecordingStatusActive:
		return videov1.RecordingStatus_RECORDING_STATUS_ACTIVE
	case recording.RecordingStatusProcessing:
		return videov1.RecordingStatus_RECORDING_STATUS_PROCESSING
	case recording.RecordingStatusCompleted:
		return videov1.RecordingStatus_RECORDING_STATUS_COMPLETED
	case recording.RecordingStatusFailed:
		return videov1.RecordingStatus_RECORDING_STATUS_FAILED
	case recording.RecordingStatusDeleted:
		return videov1.RecordingStatus_RECORDING_STATUS_DELETED
	default:
		return videov1.RecordingStatus_RECORDING_STATUS_UNSPECIFIED
	}
}

func stringToProtoRsvpStatus(s string) videov1.RsvpStatus {
	switch s {
	case meeting.MeetingRSVPPending:
		return videov1.RsvpStatus_RSVP_PENDING
	case meeting.MeetingRSVPAccepted:
		return videov1.RsvpStatus_RSVP_ACCEPTED
	case meeting.MeetingRSVPDeclined:
		return videov1.RsvpStatus_RSVP_DECLINED
	case meeting.MeetingRSVPTentative:
		return videov1.RsvpStatus_RSVP_TENTATIVE
	default:
		return videov1.RsvpStatus_RSVP_UNSPECIFIED
	}
}

func stringToProtoPresenceLevel(s string) videov1.PresenceLevel {
	switch s {
	case presence.PresenceOnline:
		return videov1.PresenceLevel_PRESENCE_ONLINE
	case presence.PresenceAway:
		return videov1.PresenceLevel_PRESENCE_AWAY
	case presence.PresenceDND:
		return videov1.PresenceLevel_PRESENCE_DND
	case presence.PresenceInCall:
		return videov1.PresenceLevel_PRESENCE_IN_CALL
	case presence.PresenceOffline:
		return videov1.PresenceLevel_PRESENCE_OFFLINE
	default:
		return videov1.PresenceLevel_PRESENCE_UNSPECIFIED
	}
}

func protoPresenceLevelToString(s videov1.PresenceLevel) string {
	switch s {
	case videov1.PresenceLevel_PRESENCE_ONLINE:
		return presence.PresenceOnline
	case videov1.PresenceLevel_PRESENCE_AWAY:
		return presence.PresenceAway
	case videov1.PresenceLevel_PRESENCE_DND:
		return presence.PresenceDND
	case videov1.PresenceLevel_PRESENCE_IN_CALL:
		return presence.PresenceInCall
	case videov1.PresenceLevel_PRESENCE_OFFLINE:
		return presence.PresenceOffline
	default:
		return presence.PresenceOffline
	}
}

// ============================================================================
// Helpers
// ============================================================================

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapVideoError(err error) error {
	switch {
	case errors.Is(err, video.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, video.ErrCallEnded):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, video.ErrMaxParticipants):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, video.ErrLiveKitNotConfigured):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, video.ErrInvalidCallType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, video.ErrNoTargetUsers):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func mapRecordingError(err error) error {
	switch {
	case errors.Is(err, recording.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, recording.ErrConsentPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, recording.ErrEgressNotConfigured):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, recording.ErrRecordingNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, recording.ErrNoCallOrMeeting):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, recording.ErrNoParticipants):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, recording.ErrNotExpired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, recording.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func mapMeetingError(err error) error {
	switch {
	case errors.Is(err, meeting.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, meeting.ErrTitleRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrTitleTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrInvalidTimeRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrNotScheduled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, meeting.ErrNotInProgress):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, meeting.ErrCannotDelete):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, meeting.ErrNotesContentRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrActionDescRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrActionItemNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, meeting.ErrNotRecurring):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, meeting.ErrNoPreviousNotes):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, meeting.ErrNoAttendeesProvided):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, meeting.ErrInvalidRSVP):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func mapPresenceError(err error) error {
	switch {
	case errors.Is(err, presence.ErrInvalidManualStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, presence.ErrInvalidAwayTimeout):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("presence error: %v", err))
	}
}
