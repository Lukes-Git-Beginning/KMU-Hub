package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	lkauth "github.com/livekit/protocol/auth"
	lkwebhook "github.com/livekit/protocol/webhook"

	"github.com/go-chi/chi/v5"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// liveKitEgressStatus mirrors the LiveKit EgressStatus enum values that matter
// for recording completion. A value of 3 means EGRESS_COMPLETE in LiveKit's proto.
const liveKitEgressStatusComplete int32 = 3

// recordingBroadcaster is the minimal interface VideoRoutes needs to push
// real-time recording events to connected clients via the WebSocket hub.
// Using an interface avoids a direct import of the server package (circular
// import risk) and makes the handler testable without a real WS hub.
type recordingBroadcaster interface {
	BroadcastRecordingStarted(ctx context.Context, meetingID, recordingID, initiatedBy string)
}

// VideoRoutes handles HTTP routes for the Video service.
// Video runs in the same binary as Work, so we reuse the "work" gRPC connection.
type VideoRoutes struct {
	registry  *ServiceRegistry
	lkKeyProv lkauth.KeyProvider   // nil when LIVEKIT_API_KEY/SECRET are not configured
	wsHub     recordingBroadcaster // optional; set via SetWSHub after hub construction
}

// NewVideoRoutes creates a new VideoRoutes with the given service registry.
//
// livekitAPIKey and livekitAPISecret are used to validate incoming LiveKit webhook
// signatures via the Authorization JWT header (LIVEKIT_API_KEY / LIVEKIT_API_SECRET).
// LiveKit signs webhooks with the API-Key/Secret pair — NOT a separate webhook secret —
// so LIVEKIT_WEBHOOK_SECRET from config is intentionally unused here.
// When both values are empty, signature validation is disabled with a startup warning
// (graceful degradation for dev/staging environments).
func NewVideoRoutes(registry *ServiceRegistry, livekitAPIKey, livekitAPISecret string) *VideoRoutes {
	vr := &VideoRoutes{registry: registry}

	if livekitAPIKey == "" || livekitAPISecret == "" {
		slog.Warn("livekit webhook signature validation disabled: LIVEKIT_API_KEY or LIVEKIT_API_SECRET not configured")
	} else {
		vr.lkKeyProv = lkauth.NewSimpleKeyProvider(livekitAPIKey, livekitAPISecret)
	}

	return vr
}

// SetWSHub injects the WebSocket hub so that recording events can be pushed
// to connected clients in real time. Must be called after the hub is constructed
// (i.e. after setupWebSocketHub in main.go). A nil hub is safe; recording events
// are silently skipped — clients fall back to polling.
func (vr *VideoRoutes) SetWSHub(hub recordingBroadcaster) {
	vr.wsHub = hub
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

// getVideoEgressClient obtains a gRPC client for the Video service.
// The Egress-completion and CompleteMeetingByRoom RPCs are now part of the
// canonical VideoServiceClient (generated from video.proto), so no wrapper needed.
func (vr *VideoRoutes) getVideoEgressClient() (videov1.VideoServiceClient, error) {
	conn, err := vr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return videov1.NewVideoServiceClient(conn), nil
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
		r.Get("/recordings/{id}/download", vr.HandleGetRecordingDownloadURL)
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
	})

	// Meeting routes
	r.Route("/api/v1/meetings", func(r chi.Router) {
		r.Use(authMiddleware)

		// Creating a meeting requires meetings:write permission (seeded in 000131).
		r.With(middleware.RequirePermission("meetings", "write")).Post("/", vr.HandleCreateMeeting)
		r.Get("/", vr.HandleListMeetings)
		r.Get("/{id}", vr.HandleGetMeeting)
		r.With(middleware.RequirePermission("meetings", "write")).Put("/{id}", vr.HandleUpdateMeeting)
		r.With(middleware.RequirePermission("meetings", "write")).Delete("/{id}", vr.HandleDeleteMeeting)
		// Only the organizer may start a meeting; the service enforces the organizer
		// check — the permission guard here ensures the caller is at least an
		// authenticated user with meetings:write capability.
		r.With(middleware.RequirePermission("meetings", "write")).Post("/{id}/start", vr.HandleStartMeeting)
		// Join is the idempotent participant entry (organizer auto-starts, attendees
		// get a token). The service enforces attendee/organizer membership; the
		// guard mirrors start/end and uses meetings:write (seeded for all roles in
		// 000131) so it never 403s the whole tenant.
		r.With(middleware.RequirePermission("meetings", "write")).Post("/{id}/join", vr.HandleJoinMeeting)
		r.With(middleware.RequirePermission("meetings", "write")).Post("/{id}/end", vr.HandleEndMeeting)

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

		// Chat (persisted in-call messages)
		r.Post("/{meetingId}/chat", vr.HandleSaveMeetingChatMessage)
		r.Get("/{meetingId}/chat", vr.HandleListMeetingChatMessages)

		// Host Controls (Wave 3 — meeting-scoped authz in the service, no RBAC guard)
		r.Post("/{meetingId}/cohosts", vr.HandlePromoteCoHost)
		r.Delete("/{meetingId}/cohosts/{userId}", vr.HandleDemoteCoHost)
		r.Get("/{meetingId}/cohosts", vr.HandleListCoHosts)
		r.Post("/{meetingId}/moderation/mute", vr.HandleMuteMeetingParticipant)
		r.Post("/{meetingId}/moderation/mute-all", vr.HandleMuteAllMeetingParticipants)
		r.Post("/{meetingId}/moderation/kick", vr.HandleRemoveMeetingParticipant)
		r.Post("/{meetingId}/lock", vr.HandleSetMeetingLock)
	})

	// LiveKit webhook (no auth middleware -- validated via JWT signature)
	r.Post("/api/v1/webhooks/livekit", vr.HandleLiveKitWebhook)
}

// ============================================================================
// Request Types
// ============================================================================

type createCallRequest struct {
	CallType       string   `json:"call_type"`
	ParticipantIDs []string `json:"participant_ids" validate:"required,min=1"`
	ChannelID      *string  `json:"channel_id,omitempty" validate:"omitempty,uuid"`
}

type startRecordingRequest struct {
	CallID    *string `json:"call_id,omitempty" validate:"omitempty,uuid"`
	MeetingID *string `json:"meeting_id,omitempty" validate:"omitempty,uuid"`
}

type setRecordingConsentRequest struct {
	Consented bool `json:"consented"`
}

type createMeetingHTTPRequest struct {
	Title              string   `json:"title" validate:"required"`
	Description        *string  `json:"description,omitempty"`
	Agenda             *string  `json:"agenda,omitempty"`
	ScheduledStart     string   `json:"scheduled_start" validate:"required"`
	ScheduledEnd       string   `json:"scheduled_end" validate:"required"`
	CalendarEventID    *string  `json:"calendar_event_id,omitempty" validate:"omitempty,uuid"`
	RecurringMeetingID *string  `json:"recurring_meeting_id,omitempty" validate:"omitempty,uuid"`
	AttendeeUserIDs    []string `json:"attendee_user_ids,omitempty"`
}

type updateMeetingHTTPRequest struct {
	Title           *string  `json:"title,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Agenda          *string  `json:"agenda,omitempty"`
	ScheduledStart  *string  `json:"scheduled_start,omitempty"`
	ScheduledEnd    *string  `json:"scheduled_end,omitempty"`
	AttendeeUserIDs []string `json:"attendee_user_ids,omitempty"`
}

type saveMeetingNotesRequest struct {
	Content   string `json:"content" validate:"required"`
	IsPrivate bool   `json:"is_private"`
}

type saveMeetingChatMessageRequest struct {
	// SenderName is the display name of the authenticated user (from auth session).
	// The server resolves sender_id from the JWT — clients cannot spoof it.
	SenderName string `json:"sender_name"`
	Message    string `json:"message" validate:"required"`
}

type createActionItemHTTPRequest struct {
	Description string  `json:"description" validate:"required"`
	AssigneeID  *string `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	SortOrder   *int32  `json:"sort_order,omitempty"`
}

type updateActionItemHTTPRequest struct {
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	IsCompleted *bool   `json:"is_completed,omitempty"`
	SortOrder   *int32  `json:"sort_order,omitempty"`
}

type convertActionItemsRequest struct {
	ActionItemIDs []string `json:"action_item_ids" validate:"required,min=1"`
	ProjectID     string   `json:"project_id" validate:"required,uuid"`
}

type setPresenceStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=online away dnd in_call offline"`
}

type bulkPresenceRequest struct {
	UserIDs []string `json:"user_ids" validate:"required,min=1"`
}

type promoteCoHostHTTPRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type muteMeetingParticipantHTTPRequest struct {
	TargetUserID string `json:"target_user_id" validate:"required,uuid"`
}

type removeMeetingParticipantHTTPRequest struct {
	TargetUserID string `json:"target_user_id" validate:"required,uuid"`
}

type setMeetingLockHTTPRequest struct {
	Locked bool `json:"locked"`
}

type updatePresenceConfigHTTPRequest struct {
	AwayTimeoutSeconds int32 `json:"away_timeout_seconds" validate:"required,gt=0"`
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

	req, ok := decodeAndValidate[createCallRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	// protojson so ice_servers and the nested call timestamps serialize in the
	// wire shape the frontend expects (RFC3339, not {seconds,nanos}).
	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[startRecordingRequest](w, r)
	if !ok {
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

	// Push recording.started WS event so participants receive a real-time
	// consent prompt without polling. Only fires for meeting-context recordings
	// (call-context uses LiveKit's native useIsRecording hook instead).
	// Best-effort: hub may be nil (dev without WS) or the broadcast may fail;
	// neither case should block the HTTP response.
	if vr.wsHub != nil && req.MeetingID != nil && *req.MeetingID != "" {
		go vr.wsHub.BroadcastRecordingStarted(
			context.Background(),
			*req.MeetingID,
			resp.GetId(),
			userID,
		)
	}

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[setRecordingConsentRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[createMeetingHTTPRequest](w, r)
	if !ok {
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
		Title:              req.Title,
		Description:        req.Description,
		Agenda:             req.Agenda,
		OrganizerId:        userID,
		ScheduledStart:     startTime,
		ScheduledEnd:       endTime,
		CalendarEventId:    req.CalendarEventID,
		RecurringMeetingId: req.RecurringMeetingID,
		AttendeeUserIds:    req.AttendeeUserIDs,
	}

	resp, err := client.CreateMeeting(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[updateMeetingHTTPRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	// The frontend (useMeetings) types this as Meeting[], so return a bare array
	// rather than the wrapped {meetings:[...]} proto envelope. An empty result
	// serializes as [] (not {}), which the desktop client maps with .map().
	response.ProtoList(w, http.StatusOK, resp.Meetings)
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

	response.Proto(w, http.StatusOK, resp)
}

// HandleJoinMeeting is the idempotent participant entry point: POST /meetings/{id}/join.
// Returns a LiveKit token, ws_url and per-session TURN ice_servers for the room.
func (vr *VideoRoutes) HandleJoinMeeting(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.JoinMeeting(r.Context(), &videov1.JoinMeetingRequest{
		MeetingId: meetingID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// protojson so ice_servers and the nested meeting timestamps serialize in the
	// wire shape the frontend expects (RFC3339, not {seconds,nanos}).
	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[saveMeetingNotesRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	req, ok := decodeAndValidate[createActionItemHTTPRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	req, ok := decodeAndValidate[updateActionItemHTTPRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleConvertActionItemsToTasks(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[convertActionItemsRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleGetBulkPresence(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[bulkPresenceRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.GetBulkPresence(r.Context(), &videov1.GetBulkPresenceRequest{
		UserIds: req.UserIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (vr *VideoRoutes) HandleSetPresenceStatus(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[setPresenceStatusRequest](w, r)
	if !ok {
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

	req, ok := decodeAndValidate[updatePresenceConfigHTTPRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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
			Filename string `json:"filename"`
			Location string `json:"location"`
			Size     int64  `json:"size"`
			Duration int64  `json:"duration"`
		} `json:"file_results"`
	} `json:"egress_info"`
}

func (vr *VideoRoutes) HandleLiveKitWebhook(w http.ResponseWriter, r *http.Request) {
	// --- Signature validation ---
	// When a key provider is configured (LIVEKIT_API_KEY + LIVEKIT_API_SECRET set),
	// lkwebhook.Receive reads and verifies the Authorization JWT header, checks the
	// SHA-256 body hash embedded in the token claims, and returns the raw body.
	// If no key provider is configured (dev/staging), we fall back to reading the
	// body manually and skip validation (startup already logged a warning).
	var body []byte

	if vr.lkKeyProv != nil {
		var err error
		body, err = lkwebhook.Receive(r, vr.lkKeyProv)
		if err != nil {
			if errors.Is(err, lkwebhook.ErrNoAuthHeader) {
				slog.Warn("livekit webhook: missing Authorization header — rejecting request")
				response.Error(w, http.StatusUnauthorized, "missing webhook authorization")
				return
			}
			slog.Warn("livekit webhook: invalid signature — rejecting request", "error", err)
			response.Error(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	} else {
		// Validation disabled: accept all requests (no secret configured).
		defer r.Body.Close()
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "failed to read request body")
			return
		}
	}

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
		vr.handleRoomFinished(r, evt)

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
			EgressId:        egress.EgressID,
			FileUrl:         fileURL,
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
		EgressId: egress.EgressID,
		Reason:   reason,
	})
	if grpcErr != nil {
		slog.Error("egress_ended: FailRecordingByEgress failed",
			"egress_id", egress.EgressID,
			"error", grpcErr,
		)
	}
}

// handleRoomFinished processes a room_finished webhook event.
// Only rooms with names prefixed "meeting-" are acted upon (non-meeting rooms
// are ignored with a debug log). For meeting rooms, it calls CompleteMeetingByRoom
// on the video gRPC service (system-context, no tenant required in the gateway —
// the server side resolves the tenant from the meeting row).
func (vr *VideoRoutes) handleRoomFinished(r *http.Request, evt liveKitWebhookEvent) {
	roomName := evt.Room.Name
	if !strings.HasPrefix(roomName, "meeting-") {
		slog.Debug("room_finished: ignoring non-meeting room", "room", roomName)
		return
	}

	client, err := vr.getVideoEgressClient()
	if err != nil {
		slog.Error("room_finished: failed to obtain video gRPC client",
			"room", roomName,
			"error", err,
		)
		return
	}

	ctx := r.Context()
	_, grpcErr := client.CompleteMeetingByRoom(ctx, &videov1.CompleteMeetingByRoomRequest{
		RoomName: roomName,
	})
	if grpcErr != nil {
		slog.Error("room_finished: CompleteMeetingByRoom failed",
			"room", roomName,
			"error", grpcErr,
		)
		return
	}

	slog.Info("meeting completed via room_finished webhook", "room", roomName)
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
		UserID      string `json:"user_id" validate:"required"`
		DisplayName string `json:"display_name" validate:"required"`
		JoinedAt    string `json:"joined_at,omitempty"`
	} `json:"snapshot" validate:"required,min=1,dive"`
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

	req, ok := decodeAndValidate[tagRecordingConsentsRequest](w, r)
	if !ok {
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

	req, ok := decodeAndValidate[updateRecordingMetadataRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	// Return a bare Recording[] array so timestamps serialize via protojson (RFC3339)
	// and the frontend can directly map over the result without unwrapping an envelope.
	response.ProtoList(w, http.StatusOK, resp.Recordings)
}

// HandleGetRecordingDownloadURL generates a presigned MinIO GET URL for a completed recording.
// GET /api/v1/video/recordings/{id}/download
// ACL: caller must be a participant of the recording (enforced in the service layer).
func (vr *VideoRoutes) HandleGetRecordingDownloadURL(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	recordingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRecordingDownloadURL(r.Context(), &videov1.GetRecordingDownloadURLRequest{
		RecordingId: recordingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
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

// ============================================================================
// Meeting Chat Handlers
// ============================================================================

// HandleSaveMeetingChatMessage persists a single in-call chat message.
// POST /api/v1/meetings/{meetingId}/chat
// Response 201: flat MeetingChatMessage JSON (created_at as ISO-8601 string).
func (vr *VideoRoutes) HandleSaveMeetingChatMessage(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[saveMeetingChatMessageRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.SaveMeetingChatMessage(r.Context(), &videov1.SaveMeetingChatMessageRequest{
		MeetingId:  meetingID,
		SenderName: req.SenderName,
		Message:    req.Message,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// response.Proto serialises via protojson — Timestamps become RFC3339 strings, not {seconds,nanos}.
	response.Proto(w, http.StatusCreated, resp)
}

// HandleListMeetingChatMessages returns the persisted chat history for a meeting.
// GET /api/v1/meetings/{meetingId}/chat?limit=100
// Response 200: {"messages":[...]} with created_at as ISO-8601 strings.
func (vr *VideoRoutes) HandleListMeetingChatMessages(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}

	limit := int32(100)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		var parsed int32
		if _, scanErr := fmt.Sscanf(lStr, "%d", &parsed); scanErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	resp, err := client.ListMeetingChatMessages(r.Context(), &videov1.ListMeetingChatMessagesRequest{
		MeetingId: meetingID,
		Limit:     limit,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// response.Proto serialises via protojson — Timestamps become RFC3339 strings, not {seconds,nanos}.
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Host Controls Handlers (Wave 3)
// ============================================================================

// HandlePromoteCoHost grants co-host rights to a user.
// POST /api/v1/meetings/{meetingId}/cohosts
// Body: {"user_id": "<uuid>"}
// Auth: organizer only (service enforces).
func (vr *VideoRoutes) HandlePromoteCoHost(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[promoteCoHostHTTPRequest](w, r)
	if !ok {
		return
	}
	_, err = client.PromoteCoHost(r.Context(), &videov1.PromoteCoHostRequest{
		MeetingId:    meetingID,
		TargetUserId: req.UserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "co-host promoted"})
}

// HandleDemoteCoHost revokes co-host rights from a user.
// DELETE /api/v1/meetings/{meetingId}/cohosts/{userId}
// Auth: organizer only (service enforces).
func (vr *VideoRoutes) HandleDemoteCoHost(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	targetUserID, ok := validateUUIDParam(w, r, "userId")
	if !ok {
		return
	}
	_, err = client.DemoteCoHost(r.Context(), &videov1.DemoteCoHostRequest{
		MeetingId:    meetingID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "co-host demoted"})
}

// HandleListCoHosts returns all co-hosts for a meeting.
// GET /api/v1/meetings/{meetingId}/cohosts
func (vr *VideoRoutes) HandleListCoHosts(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	resp, err := client.ListCoHosts(r.Context(), &videov1.ListCoHostsRequest{
		MeetingId: meetingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleMuteMeetingParticipant server-mutes a single participant.
// POST /api/v1/meetings/{meetingId}/moderation/mute
// Body: {"target_user_id": "<uuid>"}
// Auth: host or co-host (service enforces).
func (vr *VideoRoutes) HandleMuteMeetingParticipant(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[muteMeetingParticipantHTTPRequest](w, r)
	if !ok {
		return
	}
	_, err = client.MuteMeetingParticipant(r.Context(), &videov1.MuteMeetingParticipantRequest{
		MeetingId:    meetingID,
		TargetUserId: req.TargetUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "participant muted"})
}

// HandleMuteAllMeetingParticipants server-mutes all non-host participants.
// POST /api/v1/meetings/{meetingId}/moderation/mute-all
// Auth: host or co-host (service enforces).
func (vr *VideoRoutes) HandleMuteAllMeetingParticipants(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	_, err = client.MuteAllMeetingParticipants(r.Context(), &videov1.MuteAllMeetingParticipantsRequest{
		MeetingId: meetingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "all participants muted"})
}

// HandleRemoveMeetingParticipant kicks a participant from the meeting room.
// POST /api/v1/meetings/{meetingId}/moderation/kick
// Body: {"target_user_id": "<uuid>"}
// Auth: host or co-host (service enforces).
func (vr *VideoRoutes) HandleRemoveMeetingParticipant(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[removeMeetingParticipantHTTPRequest](w, r)
	if !ok {
		return
	}
	_, err = client.RemoveMeetingParticipant(r.Context(), &videov1.RemoveMeetingParticipantRequest{
		MeetingId:    meetingID,
		TargetUserId: req.TargetUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "participant removed"})
}

// HandleSetMeetingLock locks or unlocks a meeting.
// POST /api/v1/meetings/{meetingId}/lock
// Body: {"locked": true}
// Auth: host or co-host (service enforces).
func (vr *VideoRoutes) HandleSetMeetingLock(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getVideoClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}
	meetingID, ok := validateUUIDParam(w, r, "meetingId")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[setMeetingLockHTTPRequest](w, r)
	if !ok {
		return
	}
	_, err = client.SetMeetingLock(r.Context(), &videov1.SetMeetingLockRequest{
		MeetingId: meetingID,
		Locked:    req.Locked,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "meeting lock updated"})
}
