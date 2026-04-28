package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// liveKitEgressStatus mirrors the LiveKit EgressStatus enum values that matter
// for recording completion. A value of 3 means EGRESS_COMPLETE in LiveKit's proto.
const liveKitEgressStatusComplete int32 = 3

// VideoRoutes handles HTTP routes for the Video service.
// Video runs in the same binary as Work, so we reuse the "work" gRPC connection.
type VideoRoutes struct {
	registry *ServiceRegistry
}

// NewVideoRoutes creates a new VideoRoutes with the given service registry.
func NewVideoRoutes(registry *ServiceRegistry) *VideoRoutes {
	return &VideoRoutes{registry: registry}
}

// ServiceName returns the backend service name (shares "work" gRPC port).
func (vr *VideoRoutes) ServiceName() string { return "work" }

// getVideoClient lazily obtains a gRPC client for the Video service.
func (vr *VideoRoutes) getVideoClient() (videov1.VideoServiceClient, error) {
	conn, err := vr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServiceClient(conn), nil
}

// getVideoEgressClient obtains a gRPC client that includes the Egress-completion RPCs.
func (vr *VideoRoutes) getVideoEgressClient() (videov1.VideoServiceEgressClient, error) {
	conn, err := vr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServiceEgressClient(conn), nil
}

// RegisterRoutes registers all Video, Meeting, Recording, Presence, and Reaction HTTP routes.
func (vr *VideoRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Call routes
	r.Route("/api/v1/video", func(r chi.Router) {
		r.Use(authMiddleware)

		// Calls
		r.Post("/calls", vr.HandleCreateCall)
		r.Get("/calls", vr.HandleListActiveCalls)
		r.Get("/calls/{id}", vr.HandleGetCall)
		r.Post("/calls/{id}/join", vr.HandleJoinCall)
		r.Post("/calls/{id}/end", vr.HandleEndCall)

		// Recording
		r.Post("/recordings/start", vr.HandleStartRecording)
		r.Post("/recordings/{id}/stop", vr.HandleStopRecording)
		r.Post("/recordings/{id}/consent", vr.HandleSetRecordingConsent)
		r.Get("/recordings/{id}/consent", vr.HandleGetRecordingConsent)
		r.Get("/recordings/{id}/consents", vr.HandleGetRecordingConsents)
		// Mutating recording-tag/metadata/cleanup endpoints touch forensic GDPR
		// state — they require the recordings:admin permission so a regular user
		// cannot overwrite the consent snapshot of someone else's recording.
		r.With(middleware.RequirePermission("recordings", "admin")).Post("/recordings/{id}/tag-consents", vr.HandleTagRecordingWithConsents)
		r.With(middleware.RequirePermission("recordings", "admin")).Patch("/recordings/{id}/metadata", vr.HandleUpdateRecordingMetadata)
		r.Get("/recordings/{id}/status", vr.HandleGetRecordingStatus)
		r.With(middleware.RequirePermission("recordings", "admin")).Post("/recordings/{id}/cleanup", vr.HandleCleanupExpiredRecording)
		r.Get("/recordings", vr.HandleListRecordings)
		r.Get("/meetings/{meetingId}/recordings", vr.HandleListRecordingsByMeeting)
		r.Delete("/recordings/{id}", vr.HandleDeleteRecording)
		// Initiator pre-recording consent (R2-P0.4 / Migration 000107)
		// Requires recording:write to ensure only authenticated users with recording
		// capability can stamp pre-consent (no anonymous or read-only users).
		r.With(middleware.RequirePermission("recording", "write")).Post("/recordings/{id}/initiator-consent", vr.HandleConfirmInitiatorConsent)

		// Presence
		r.Get("/presence/{userId}", vr.HandleGetPresence)
		r.Post("/presence/bulk", vr.HandleGetBulkPresence)
		r.Post("/presence/status", vr.HandleSetPresenceStatus)
		r.With(middleware.RequirePermission("settings", "write")).Put("/presence/config", vr.HandleUpdatePresenceConfig)
		r.Get("/presence/config", vr.HandleGetPresenceConfig)

		// Reactions (extends chat messages)
		r.Post("/reactions/toggle", vr.HandleToggleReaction)
		r.Get("/reactions/{messageId}", vr.HandleListReactions)
		r.Post("/reactions/summary", vr.HandleGetReactionSummary)
	})

	// Meeting routes
	r.Route("/api/v1/meetings", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", vr.HandleCreateMeeting)
		r.Get("/", vr.HandleListMeetings)
		r.Get("/{id}", vr.HandleGetMeeting)
		r.Put("/{id}", vr.HandleUpdateMeeting)
		r.Delete("/{id}", vr.HandleDeleteMeeting)
		r.Post("/{id}/start", vr.HandleStartMeeting)
		r.Post("/{id}/end", vr.HandleEndMeeting)

		// Notes
		r.Put("/{id}/notes", vr.HandleSaveMeetingNotes)
		r.Get("/{id}/notes", vr.HandleGetMeetingNotes)
		r.Get("/{id}/previous-notes", vr.HandleGetPreviousMeetingNotes)

		// Action Items
		r.Post("/{id}/action-items", vr.HandleCreateActionItem)
		r.Get("/{id}/action-items", vr.HandleListActionItems)
		r.Put("/action-items/{itemId}", vr.HandleUpdateActionItem)
		r.Delete("/action-items/{itemId}", vr.HandleDeleteActionItem)
		r.Post("/{id}/action-items/convert", vr.HandleConvertActionItemsToTasks)
	})

	// LiveKit webhook (no auth middleware -- validated via JWT signature)
	r.Post("/api/v1/webhooks/livekit", vr.HandleLiveKitWebhook)
}

// ============================================================================
// Request Types
// ============================================================================

type createCallRequest struct {
	CallType       string   `json:"call_type"`
	ParticipantIDs []string `json:"participant_ids"`
	ChannelID      *string  `json:"channel_id,omitempty"`
}

type startRecordingRequest struct {
	CallID    *string `json:"call_id,omitempty"`
	MeetingID *string `json:"meeting_id,omitempty"`
}

type setRecordingConsentRequest struct {
	Consented bool `json:"consented"`
}

type createMeetingHTTPRequest struct {
	Title              string   `json:"title"`
	Description        *string  `json:"description,omitempty"`
	Agenda             *string  `json:"agenda,omitempty"`
	ScheduledStart     string   `json:"scheduled_start"`
	ScheduledEnd       string   `json:"scheduled_end"`
	CalendarEventID    *string  `json:"calendar_event_id,omitempty"`
	RecurringMeetingID *string  `json:"recurring_meeting_id,omitempty"`
	AttendeeUserIDs    []string `json:"attendee_user_ids,omitempty"`
}

type updateMeetingHTTPRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Agenda         *string `json:"agenda,omitempty"`
	ScheduledStart *string `json:"scheduled_start,omitempty"`
	ScheduledEnd   *string `json:"scheduled_end,omitempty"`
	AttendeeUserIDs []string `json:"attendee_user_ids,omitempty"`
}

type saveMeetingNotesRequest struct {
	Content   string `json:"content"`
	IsPrivate bool   `json:"is_private"`
}

type createActionItemHTTPRequest struct {
	Description string  `json:"description"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	SortOrder   *int32  `json:"sort_order,omitempty"`
}

type updateActionItemHTTPRequest struct {
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	IsCompleted *bool   `json:"is_completed,omitempty"`
	SortOrder   *int32  `json:"sort_order,omitempty"`
}

type convertActionItemsRequest struct {
	ActionItemIDs []string `json:"action_item_ids"`
	ProjectID     string   `json:"project_id"`
}

type setPresenceStatusRequest struct {
	Status string `json:"status"`
}

type bulkPresenceRequest struct {
	UserIDs []string `json:"user_ids"`
}

type updatePresenceConfigHTTPRequest struct {
	AwayTimeoutSeconds int32 `json:"away_timeout_seconds"`
}

// ============================================================================
// Call Handlers
// ============================================================================

func (vr *VideoRoutes) HandleCreateCall(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	callType := videov1.CallType_CALL_TYPE_ONE_TO_ONE
	if req.CallType == "group" {
		callType = videov1.CallType_CALL_TYPE_GROUP
	}

	grpcReq := &videov1.CreateCallRequest{
		CallType:       callType,
		InitiatorId:    userID,
		ParticipantIds: req.ParticipantIDs,
		ChannelId:      req.ChannelID,
	}

	resp, err := client.CreateCall(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VideoRoutes) HandleJoinCall(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	callID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.JoinCall(r.Context(), &videov1.JoinCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleEndCall(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	callID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.EndCall(r.Context(), &videov1.EndCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "call ended"})
}

func (vr *VideoRoutes) HandleGetCall(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	callID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCall(r.Context(), &videov1.GetCallRequest{CallId: callID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleListActiveCalls(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ListActiveCalls(r.Context(), &videov1.ListActiveCallsRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Recording Handlers
// ============================================================================

func (vr *VideoRoutes) HandleStartRecording(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req startRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.StartRecording(r.Context(), &videov1.StartRecordingRequest{
		CallId:    req.CallID,
		MeetingId: req.MeetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VideoRoutes) HandleStopRecording(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.StopRecording(r.Context(), &videov1.StopRecordingRequest{
		RecordingId: recordingID,
		UserId:      userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleSetRecordingConsent(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req setRecordingConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.SetRecordingConsent(r.Context(), &videov1.SetRecordingConsentRequest{
		RecordingId: recordingID,
		UserId:      userID,
		Consented:   req.Consented,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "consent recorded"})
}

func (vr *VideoRoutes) HandleGetRecordingConsent(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRecordingConsent(r.Context(), &videov1.GetRecordingConsentRequest{
		RecordingId: recordingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleListRecordings(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	q := r.URL.Query()
	grpcReq := &videov1.ListRecordingsRequest{}
	if callID := q.Get("call_id"); callID != "" {
		grpcReq.CallId = &callID
	}
	if meetingID := q.Get("meeting_id"); meetingID != "" {
		grpcReq.MeetingId = &meetingID
	}

	page, pageSize := parsePagination(r, 1, 20)
	grpcReq.Page = int32(page)
	grpcReq.PageSize = int32(pageSize)

	resp, err := client.ListRecordings(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleDeleteRecording(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteRecording(r.Context(), &videov1.DeleteRecordingRequest{
		RecordingId: recordingID,
		UserId:      userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "recording deleted"})
}

// ============================================================================
// Meeting Handlers
// ============================================================================

func (vr *VideoRoutes) HandleCreateMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createMeetingHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		response.Error(w, http.StatusBadRequest, "title is required")
		return
	}

	startTime, err := parseTimestamp(req.ScheduledStart)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid scheduled_start format")
		return
	}
	endTime, err := parseTimestamp(req.ScheduledEnd)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid scheduled_end format")
		return
	}

	grpcReq := &videov1.CreateMeetingRequest{
		Title:           req.Title,
		Description:     req.Description,
		Agenda:          req.Agenda,
		OrganizerId:     userID,
		ScheduledStart:  startTime,
		ScheduledEnd:    endTime,
		CalendarEventId: req.CalendarEventID,
		RecurringMeetingId: req.RecurringMeetingID,
		AttendeeUserIds: req.AttendeeUserIDs,
	}

	resp, err := client.CreateMeeting(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VideoRoutes) HandleGetMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetMeeting(r.Context(), &videov1.GetMeetingRequest{MeetingId: meetingID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleUpdateMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateMeetingHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &videov1.UpdateMeetingRequest{
		MeetingId:       meetingID,
		Title:           req.Title,
		Description:     req.Description,
		Agenda:          req.Agenda,
		AttendeeUserIds: req.AttendeeUserIDs,
	}
	if req.ScheduledStart != nil {
		t, parseErr := parseTimestamp(*req.ScheduledStart)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid scheduled_start format")
			return
		}
		grpcReq.ScheduledStart = t
	}
	if req.ScheduledEnd != nil {
		t, parseErr := parseTimestamp(*req.ScheduledEnd)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid scheduled_end format")
			return
		}
		grpcReq.ScheduledEnd = t
	}

	resp, err := client.UpdateMeeting(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleDeleteMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteMeeting(r.Context(), &videov1.DeleteMeetingRequest{
		MeetingId: meetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "meeting deleted"})
}

func (vr *VideoRoutes) HandleListMeetings(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	q := r.URL.Query()

	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &videov1.ListMeetingsRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	if statusStr := q.Get("status"); statusStr != "" {
		statusMap := map[string]videov1.MeetingStatus{
			"scheduled":   videov1.MeetingStatus_MEETING_STATUS_SCHEDULED,
			"in_progress": videov1.MeetingStatus_MEETING_STATUS_IN_PROGRESS,
			"completed":   videov1.MeetingStatus_MEETING_STATUS_COMPLETED,
			"cancelled":   videov1.MeetingStatus_MEETING_STATUS_CANCELLED,
		}
		if s, ok := statusMap[statusStr]; ok {
			grpcReq.StatusFilter = &s
		}
	}
	if startAfter := q.Get("start_after"); startAfter != "" {
		t, parseErr := parseTimestamp(startAfter)
		if parseErr == nil {
			grpcReq.StartAfter = t
		}
	}
	if startBefore := q.Get("start_before"); startBefore != "" {
		t, parseErr := parseTimestamp(startBefore)
		if parseErr == nil {
			grpcReq.StartBefore = t
		}
	}

	resp, err := client.ListMeetings(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleStartMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.StartMeeting(r.Context(), &videov1.StartMeetingRequest{
		MeetingId: meetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleEndMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.EndMeeting(r.Context(), &videov1.EndMeetingRequest{
		MeetingId: meetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Meeting Notes Handlers
// ============================================================================

func (vr *VideoRoutes) HandleSaveMeetingNotes(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req saveMeetingNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.SaveMeetingNotes(r.Context(), &videov1.SaveMeetingNotesRequest{
		MeetingId: meetingID,
		AuthorId:  userID,
		Content:   req.Content,
		IsPrivate: req.IsPrivate,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleGetMeetingNotes(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetMeetingNotes(r.Context(), &videov1.GetMeetingNotesRequest{
		MeetingId: meetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleGetPreviousMeetingNotes(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetPreviousMeetingNotes(r.Context(), &videov1.GetPreviousMeetingNotesRequest{
		CurrentMeetingId: meetingID,
		UserId:           userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Action Item Handlers
// ============================================================================

func (vr *VideoRoutes) HandleCreateActionItem(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req createActionItemHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &videov1.CreateActionItemRequest{
		MeetingId:   meetingID,
		Description: req.Description,
		AssigneeId:  req.AssigneeID,
		SortOrder:   req.SortOrder,
	}

	resp, err := client.CreateActionItem(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VideoRoutes) HandleUpdateActionItem(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	itemID, ok := validateUUIDParam(w, r, "itemId")
	if !ok {
		return
	}

	var req updateActionItemHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &videov1.UpdateActionItemRequest{
		ActionItemId: itemID,
		Description:  req.Description,
		AssigneeId:   req.AssigneeID,
		IsCompleted:  req.IsCompleted,
		SortOrder:    req.SortOrder,
	}

	resp, err := client.UpdateActionItem(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleDeleteActionItem(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	itemID, ok := validateUUIDParam(w, r, "itemId")
	if !ok {
		return
	}

	_, err = client.DeleteActionItem(r.Context(), &videov1.DeleteActionItemRequest{
		ActionItemId: itemID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "action item deleted"})
}

func (vr *VideoRoutes) HandleListActionItems(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListActionItems(r.Context(), &videov1.ListActionItemsRequest{
		MeetingId: meetingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleConvertActionItemsToTasks(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req convertActionItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ConvertActionItemsToTasks(r.Context(), &videov1.ConvertActionItemsToTasksRequest{
		ActionItemIds: req.ActionItemIDs,
		ProjectId:     req.ProjectID,
		UserId:        userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Presence Handlers
// ============================================================================

func (vr *VideoRoutes) HandleGetPresence(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userIDParam := chi.URLParam(r, "userId")

	resp, err := client.GetPresence(r.Context(), &videov1.GetPresenceRequest{
		UserId: userIDParam,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleGetBulkPresence(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	var req bulkPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.GetBulkPresence(r.Context(), &videov1.GetBulkPresenceRequest{
		UserIds: req.UserIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleSetPresenceStatus(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req setPresenceStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	statusMap := map[string]videov1.PresenceLevel{
		"online":  videov1.PresenceLevel_PRESENCE_ONLINE,
		"away":    videov1.PresenceLevel_PRESENCE_AWAY,
		"dnd":     videov1.PresenceLevel_PRESENCE_DND,
		"in_call": videov1.PresenceLevel_PRESENCE_IN_CALL,
		"offline": videov1.PresenceLevel_PRESENCE_OFFLINE,
	}

	presenceLevel, ok := statusMap[req.Status]
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid status value")
		return
	}

	_, err = client.SetPresenceStatus(r.Context(), &videov1.SetPresenceStatusRequest{
		UserId: userID,
		Status: presenceLevel,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "presence updated"})
}

func (vr *VideoRoutes) HandleUpdatePresenceConfig(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	var req updatePresenceConfigHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.UpdatePresenceConfig(r.Context(), &videov1.UpdatePresenceConfigRequest{
		AwayTimeoutSeconds: req.AwayTimeoutSeconds,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "presence config updated"})
}

func (vr *VideoRoutes) HandleGetPresenceConfig(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	resp, err := client.GetPresenceConfig(r.Context(), &videov1.GetPresenceConfigRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Reaction Handlers (via VideoService gRPC -- reactions live in Work binary)
// ============================================================================

func (vr *VideoRoutes) HandleToggleReaction(w http.ResponseWriter, r *http.Request) {
	// Reactions are not in the video proto -- they were added to the reaction service
	// which is wired directly in the Work binary. For now, return 501 until we
	// add reaction RPCs to the video proto or create a separate route.
	// The reaction service is accessible via the work service gRPC connection.
	response.Error(w, http.StatusNotImplemented, "reactions available via WebSocket events")
}

func (vr *VideoRoutes) HandleListReactions(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "reactions available via WebSocket events")
}

func (vr *VideoRoutes) HandleGetReactionSummary(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "reactions available via WebSocket events")
}

// ============================================================================
// LiveKit Webhook Handler
// ============================================================================

// liveKitWebhookEvent mirrors the relevant fields of the LiveKit webhook payload.
// LiveKit EgressInfo status values: 0=starting, 1=active, 2=ending, 3=complete, 4=failed, 5=aborted.
type liveKitWebhookEvent struct {
	Event string `json:"event"`
	Room  struct {
		Name string `json:"name"`
	} `json:"room"`
	Participant struct {
		Identity string `json:"identity"`
	} `json:"participant"`
	EgressInfo struct {
		EgressID string `json:"egress_id"`
		RoomName string `json:"room_name"`
		Status   int32  `json:"status"`
		// Error is set when Status indicates failure.
		Error string `json:"error"`
		// Duration is in nanoseconds (LiveKit convention).
		Duration int64 `json:"duration"`
		// FileResults holds details about uploaded recording files.
		FileResults []struct {
			Filename  string `json:"filename"`
			Location  string `json:"location"`
			Size      int64  `json:"size"`
			Duration  int64  `json:"duration"`
		} `json:"file_results"`
	} `json:"egress_info"`
}

func (vr *VideoRoutes) HandleLiveKitWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body for webhook validation.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var evt liveKitWebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		slog.Warn("invalid livekit webhook payload", "error", err)
		response.Error(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	slog.Info("livekit webhook received",
		"event", evt.Event,
		"room", evt.Room.Name,
	)

	switch evt.Event {
	case "participant_joined":
		slog.Info("participant joined via webhook",
			"room", evt.Room.Name,
			"identity", evt.Participant.Identity,
		)

	case "participant_left":
		slog.Info("participant left via webhook",
			"room", evt.Room.Name,
			"identity", evt.Participant.Identity,
		)

	case "room_finished":
		slog.Info("room finished via webhook",
			"room", evt.Room.Name,
		)

	case "egress_ended":
		vr.handleEgressEnded(r, evt)

	default:
		slog.Debug("unhandled livekit webhook event", "event", evt.Event)
	}

	w.WriteHeader(http.StatusOK)
}

// handleEgressEnded processes an egress_ended webhook event.
// It calls CompleteRecordingByEgress or FailRecordingByEgress on the video
// gRPC service so that the recording row is updated from "processing" to the
// final status and the file URL is persisted.
func (vr *VideoRoutes) handleEgressEnded(r *http.Request, evt liveKitWebhookEvent) {
	egress := evt.EgressInfo
	slog.Info("egress ended via webhook",
		"egress_id", egress.EgressID,
		"status", egress.Status,
	)

	if egress.EgressID == "" {
		slog.Warn("egress_ended webhook missing egress_id — cannot update recording")
		return
	}

	client, err := vr.getVideoEgressClient()
	if err != nil {
		slog.Error("egress_ended: failed to obtain video gRPC client",
			"egress_id", egress.EgressID,
			"error", err,
		)
		return
	}

	ctx := r.Context()

	// LiveKit EgressStatus 3 = EGRESS_COMPLETE.
	if egress.Status == liveKitEgressStatusComplete {
		// Extract file metadata from the first FileResult, if present.
		var fileURL string
		var fileSizeBytes int64
		if len(egress.FileResults) > 0 {
			fr := egress.FileResults[0]
			if fr.Location != "" {
				fileURL = fr.Location
			} else {
				fileURL = fr.Filename
			}
			fileSizeBytes = fr.Size
		}

		// LiveKit Duration is in nanoseconds — convert to whole seconds.
		durationNs := egress.Duration
		if len(egress.FileResults) > 0 && egress.FileResults[0].Duration > 0 {
			durationNs = egress.FileResults[0].Duration
		}
		durationSeconds := int32(durationNs / 1_000_000_000) //nolint:mnd

		_, grpcErr := client.CompleteRecordingByEgress(ctx, &videov1.CompleteRecordingByEgressRequest{
			EgressID:        egress.EgressID,
			FileURL:         fileURL,
			FileSizeBytes:   fileSizeBytes,
			DurationSeconds: durationSeconds,
		})
		if grpcErr != nil {
			slog.Error("egress_ended: CompleteRecordingByEgress failed",
				"egress_id", egress.EgressID,
				"error", grpcErr,
			)
			return
		}

		slog.Info("recording completed via egress webhook",
			"egress_id", egress.EgressID,
			"file_url", fileURL,
			"duration_seconds", durationSeconds,
		)
		return
	}

	// Any non-complete status is treated as failure.
	reason := egress.Error
	if reason == "" {
		reason = "egress ended with non-complete status"
	}

	slog.Warn("egress ended with failure status",
		"egress_id", egress.EgressID,
		"status", egress.Status,
		"reason", reason,
	)

	_, grpcErr := client.FailRecordingByEgress(ctx, &videov1.FailRecordingByEgressRequest{
		EgressID: egress.EgressID,
		Reason:   reason,
	})
	if grpcErr != nil {
		slog.Error("egress_ended: FailRecordingByEgress failed",
			"egress_id", egress.EgressID,
			"error", grpcErr,
		)
	}
}

// ============================================================================
// Recording Tagging Handlers (S1.7 / Welle 1.B)
// ============================================================================

// getVideoRecordingTagClient obtains the recording-tag extended gRPC client.
func (vr *VideoRoutes) getVideoRecordingTagClient() (videov1.VideoServiceRecordingTagClient, error) {
	conn, err := vr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServiceRecordingTagClient(conn), nil
}

// HandleGetRecordingConsents returns the consent snapshot + live consent rows for a recording.
// GET /api/v1/video/recordings/{id}/consents
func (vr *VideoRoutes) HandleGetRecordingConsents(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRecordingConsents(r.Context(), &videov1.GetRecordingConsentsRequest{
		RecordingID: recordingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type tagRecordingConsentsRequest struct {
	Snapshot []struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
		JoinedAt    string `json:"joined_at,omitempty"`
	} `json:"snapshot"`
}

// HandleTagRecordingWithConsents overwrites the consent snapshot on a recording.
// POST /api/v1/video/recordings/{id}/tag-consents
func (vr *VideoRoutes) HandleTagRecordingWithConsents(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req tagRecordingConsentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entries := make([]*videov1.ConsentSnapshotEntry, 0, len(req.Snapshot))
	for _, e := range req.Snapshot {
		entries = append(entries, &videov1.ConsentSnapshotEntry{
			UserID:      e.UserID,
			DisplayName: e.DisplayName,
			JoinedAt:    e.JoinedAt,
		})
	}

	_, err = client.TagRecordingWithConsents(r.Context(), &videov1.TagRecordingWithConsentsRequest{
		RecordingID: recordingID,
		Snapshot:    entries,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "consent snapshot updated"})
}

type updateRecordingMetadataRequest struct {
	FileURL         *string `json:"file_url,omitempty"`
	FileSizeBytes   *int64  `json:"file_size_bytes,omitempty"`
	DurationSeconds *int32  `json:"duration_seconds,omitempty"`
	Status          *string `json:"status,omitempty"`
}

// HandleUpdateRecordingMetadata updates mutable fields on a recording.
// PATCH /api/v1/video/recordings/{id}/metadata
func (vr *VideoRoutes) HandleUpdateRecordingMetadata(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateRecordingMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateRecordingMetadata(r.Context(), &videov1.UpdateRecordingMetadataRequest{
		RecordingID:     recordingID,
		FileURL:         req.FileURL,
		FileSizeBytes:   req.FileSizeBytes,
		DurationSeconds: req.DurationSeconds,
		Status:          req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleGetRecordingStatus returns the current status of a single recording.
// GET /api/v1/video/recordings/{id}/status
func (vr *VideoRoutes) HandleGetRecordingStatus(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRecordingStatus(r.Context(), &videov1.GetRecordingStatusRequest{
		RecordingID: recordingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleListRecordingsByMeeting lists recordings for a specific meeting with pagination.
// GET /api/v1/video/meetings/{meetingId}/recordings
func (vr *VideoRoutes) HandleListRecordingsByMeeting(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListRecordingsByMeeting(r.Context(), &videov1.ListRecordingsByMeetingRequest{
		MeetingID: meetingID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleCleanupExpiredRecording deletes a single recording past its retention period.
// POST /api/v1/video/recordings/{id}/cleanup  (admin/cron endpoint)
func (vr *VideoRoutes) HandleCleanupExpiredRecording(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoRecordingTagClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.CleanupExpiredRecording(r.Context(), &videov1.CleanupExpiredRecordingRequest{
		RecordingID: recordingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "recording deleted"})
}

// ============================================================================
// Initiator Pre-Recording Consent Handler (R2-P0.4 / Migration 000107)
// ============================================================================

// getVideoPreConsentClient obtains the pre-consent extended gRPC client.
func (vr *VideoRoutes) getVideoPreConsentClient() (videov1.VideoServicePreConsentClient, error) {
	conn, err := vr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServicePreConsentClient(conn), nil
}

// HandleConfirmInitiatorConsent stamps pre_recording_consent_at on a recording row.
// Must be called by the initiator before StartRecording when the pre-dialog flow is used.
// Returns 412 Precondition Failed when the recording does not belong to the calling user.
// POST /api/v1/video/recordings/{id}/initiator-consent
func (vr *VideoRoutes) HandleConfirmInitiatorConsent(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoPreConsentClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	tenantID, tenantErr := middleware.GetTenantID(r.Context())
	if tenantErr != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}
	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ConfirmInitiatorConsent(r.Context(), &videov1.ConfirmInitiatorConsentRequest{
		RecordingID: recordingID,
		UserID:      userID,
		TenantID:    tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	if !resp.Stamped {
		response.Error(w, http.StatusPreconditionFailed, "pre-recording consent could not be confirmed: recording not found or not owned by user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"stamped": true})
}

// parseTimestamp is defined in route_work.go and shared across gateway routes.
