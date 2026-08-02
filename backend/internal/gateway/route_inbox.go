package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// cannedResponseMarshaler mirrors response.Proto's protojson options for ad-hoc map
// envelopes that wrap one or more proto.Message values (with real Timestamp fields)
// inside a hand-built map[string]interface{}, which response.Proto/response.ProtoList
// can't serialize directly. response.JSON can't fix nested proto Timestamps on its
// own, so these spots marshal via protojson first and embed the result as
// json.RawMessage. Originally scoped to the two canned-response handlers below;
// route_hr.go's map-envelope handlers (leave requests, work-time entries, employees,
// etc.) reuse it too since it's a package-level var with identical protojson options.
// lean: duplicates response.go's unexported protoMarshaler config; the "third envelope
// case" trigger has now fired (hr added ~12 more call sites) — fold into an exported
// response.ProtoEnvelope helper next time this pattern spreads to another module.
var cannedResponseMarshaler = protojson.MarshalOptions{
	UseProtoNames:  true,
	UseEnumNumbers: true,
}

// InboxRoutes handles HTTP routes for the inbox service.
// InboxService is co-hosted in the notification binary, so ServiceName
// returns "notification" to reuse the existing gRPC connection.
type InboxRoutes struct {
	registry *ServiceRegistry
}

// NewInboxRoutes creates a new InboxRoutes with the given service registry.
func NewInboxRoutes(registry *ServiceRegistry) *InboxRoutes {
	return &InboxRoutes{registry: registry}
}

// ServiceName returns "notification" because InboxService is co-hosted in the notification binary.
func (ir *InboxRoutes) ServiceName() string { return "notification" }

// getInboxClient lazily obtains a gRPC client for the inbox service.
func (ir *InboxRoutes) getInboxClient() (inboxv1.InboxServiceClient, error) {
	conn, err := ir.registry.GetConnection("notification")
	if err != nil {
		return nil, err
	}
	return inboxv1.NewInboxServiceClient(conn), nil
}

// RegisterRoutes registers all inbox HTTP routes.
func (ir *InboxRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Additive guards: keep the legacy coarse "inbox" key AND accept the
	// matching capability-catalogue key. Extend, never swap — see
	// middleware.RequirePermissionAny doc comment.
	//
	// kommunikation:team_inbox:manage only extends the two team-inbox routes
	// already gated by RequirePermission("inbox", ...); create/delete team
	// inbox use RequireRole("admin","manager") instead, a different guard
	// mechanism with no permission-based alt-key to extend, so they stay as
	// they are (not tightened, not loosened). Same reasoning empties out
	// kommunikation:routing:manage — every routing-rule mutation route is
	// RequireRole-gated, so there is no additive target for it at all.
	// kommunikation:webhook:manage has no backend endpoint (chat/inbox never
	// grew a webhook route) — key stays unused, no route invented.
	var (
		inboxCannedManage    = middleware.RequirePermissionAny([2]string{"inbox", "write"}, [2]string{"kommunikation:canned", "manage"})
		inboxTeamInboxManage = middleware.RequirePermissionAny([2]string{"inbox", "write"}, [2]string{"kommunikation:team_inbox", "manage"})
	)

	r.Route("/api/v1/inbox", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Message endpoints
		r.With(middleware.RequirePermission("inbox", "read")).Get("/messages", ir.HandleListMessages)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/messages/unread-count", ir.HandleGetUnreadCount)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/messages/{id}", ir.HandleGetMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/read", ir.HandleMarkRead)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/unread", ir.HandleMarkUnread)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/star", ir.HandleToggleStar)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/status", ir.HandleSetMessageStatus)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/tags", ir.HandleAddMessageTag)
		r.With(middleware.RequirePermission("inbox", "write")).Delete("/messages/{id}/tags", ir.HandleRemoveMessageTag)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/forward", ir.HandleForwardMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/archive", ir.HandleArchiveMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/unarchive", ir.HandleUnarchiveMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/snooze", ir.HandleSnoozeMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/unsnooze", ir.HandleUnsnoozeMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/reply", ir.HandleReplyToMessage)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/assign", ir.HandleAssignMessage)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/unread-count", ir.HandleGetUnreadCount)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/bulk-read", ir.HandleBulkMarkRead)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/bulk-archive", ir.HandleBulkArchive)

		// Thread + canned-response endpoints (reuse inbox read/write permissions)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/messages/{id}/thread", ir.HandleListThreadMessages)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/canned-responses", ir.HandleListCannedResponses)
		r.With(inboxCannedManage).Post("/canned-responses", ir.HandleCreateCannedResponse)
		r.With(inboxCannedManage).Put("/canned-responses/{id}", ir.HandleUpdateCannedResponse)
		r.With(inboxCannedManage).Delete("/canned-responses/{id}", ir.HandleDeleteCannedResponse)

		// Team inbox endpoints
		r.With(middleware.RequireRole("admin", "manager")).Post("/teams", ir.HandleCreateTeamInbox)
		r.With(inboxTeamInboxManage).Put("/teams/{id}", ir.HandleUpdateTeamInbox)
		r.With(middleware.RequireRole("admin", "manager")).Delete("/teams/{id}", ir.HandleDeleteTeamInbox)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/teams", ir.HandleListTeamInboxes)
		r.With(inboxTeamInboxManage).Post("/teams/{id}/members", ir.HandleAddTeamMember)
		r.With(inboxTeamInboxManage).Delete("/teams/{id}/members/{user_id}", ir.HandleRemoveTeamMember)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/teams/{id}/members", ir.HandleListTeamMembers)
		r.With(middleware.RequirePermission("inbox", "write")).Post("/messages/{id}/claim", ir.HandleClaimMessage)

		// Routing rule endpoints (admin/manager only)
		r.With(middleware.RequireRole("admin", "manager")).Post("/rules", ir.HandleCreateRoutingRule)
		r.With(middleware.RequireRole("admin", "manager")).Put("/rules/{id}", ir.HandleUpdateRoutingRule)
		r.With(middleware.RequireRole("admin", "manager")).Delete("/rules/{id}", ir.HandleDeleteRoutingRule)
		r.With(middleware.RequirePermission("inbox", "read")).Get("/rules", ir.HandleListRoutingRules)
		r.With(middleware.RequireRole("admin", "manager")).Post("/rules/test", ir.HandleTestRoutingRule)
	})
}

// ============================================================================
// Message Handlers
// ============================================================================

func (ir *InboxRoutes) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	grpcReq := &inboxv1.ListMessagesRequest{
		UserId:    userID,
		PageToken: r.URL.Query().Get("page_token"),
	}

	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, parseErr := strconv.Atoi(ps); parseErr == nil && parsed > 0 && parsed <= 100 {
			grpcReq.PageSize = int32(parsed)
		}
	}
	if grpcReq.PageSize == 0 {
		grpcReq.PageSize = 50
	}

	if ch := r.URL.Query().Get("channel"); ch != "" {
		channel := parseChannelQuery(ch)
		grpcReq.Channel = &channel
	}
	if isRead := r.URL.Query().Get("is_read"); isRead != "" {
		val := isRead == "true"
		grpcReq.IsRead = &val
	}
	if isStarred := r.URL.Query().Get("is_starred"); isStarred != "" {
		val := isStarred == "true"
		grpcReq.IsStarred = &val
	}
	if teamInboxID := r.URL.Query().Get("team_inbox_id"); teamInboxID != "" {
		grpcReq.TeamInboxId = &teamInboxID
	}
	if search := r.URL.Query().Get("search"); search != "" {
		grpcReq.SearchQuery = &search
	}
	if inboxStatus := r.URL.Query().Get("status"); inboxStatus != "" {
		grpcReq.Status = &inboxStatus
	}

	resp, err := client.ListMessages(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetMessage(r.Context(), &inboxv1.GetMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.MarkRead(r.Context(), &inboxv1.MarkReadRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleMarkUnread(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.MarkUnread(r.Context(), &inboxv1.MarkUnreadRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleToggleStar(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ToggleStar(r.Context(), &inboxv1.ToggleStarRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type setMessageStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=open pending resolved closed"`
}

func (ir *InboxRoutes) HandleSetMessageStatus(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[setMessageStatusRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.SetMessageStatus(r.Context(), &inboxv1.SetMessageStatusRequest{
		MessageId: messageID,
		UserId:    userID,
		Status:    body.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type inboxMessageTagRequest struct {
	Tag string `json:"tag" validate:"required"`
}

func (ir *InboxRoutes) HandleAddMessageTag(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[inboxMessageTagRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddMessageTag(r.Context(), &inboxv1.AddMessageTagRequest{
		MessageId: messageID,
		UserId:    userID,
		Tag:       body.Tag,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleRemoveMessageTag(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[inboxMessageTagRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RemoveMessageTag(r.Context(), &inboxv1.RemoveMessageTagRequest{
		MessageId: messageID,
		UserId:    userID,
		Tag:       body.Tag,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type forwardMessageRequest struct {
	To   string  `json:"to" validate:"required"`
	Note *string `json:"note"`
}

func (ir *InboxRoutes) HandleForwardMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[forwardMessageRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ForwardMessage(r.Context(), &inboxv1.ForwardMessageRequest{
		MessageId: messageID,
		UserId:    userID,
		To:        body.To,
		Note:      body.Note,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleArchiveMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ArchiveMessage(r.Context(), &inboxv1.ArchiveMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleUnarchiveMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.UnarchiveMessage(r.Context(), &inboxv1.UnarchiveMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type snoozeMessageRequest struct {
	SnoozeUntil string `json:"snooze_until" validate:"required"`
}

func (ir *InboxRoutes) HandleSnoozeMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[snoozeMessageRequest](w, r)
	if !ok {
		return
	}

	snoozeUntil, parseErr := time.Parse(time.RFC3339, body.SnoozeUntil)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid snooze_until timestamp, expected RFC3339 format")
		return
	}

	resp, err := client.SnoozeMessage(r.Context(), &inboxv1.SnoozeMessageRequest{
		MessageId:   messageID,
		UserId:      userID,
		SnoozeUntil: timestamppb.New(snoozeUntil),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleUnsnoozeMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.UnsnoozeMessage(r.Context(), &inboxv1.UnsnoozeMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type replyToMessageRequest struct {
	Body string `json:"body" validate:"required"`
}

func (ir *InboxRoutes) HandleReplyToMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[replyToMessageRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ReplyToMessage(r.Context(), &inboxv1.ReplyToMessageRequest{
		MessageId: messageID,
		UserId:    userID,
		Body:      body.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type assignMessageRequest struct {
	AssigneeID string `json:"assignee_id" validate:"required,uuid"`
}

func (ir *InboxRoutes) HandleAssignMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[assignMessageRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AssignMessage(r.Context(), &inboxv1.AssignMessageRequest{
		MessageId:  messageID,
		UserId:     userID,
		AssigneeId: body.AssigneeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetUnreadCount(r.Context(), &inboxv1.GetUnreadCountRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type bulkIDsRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,dive,uuid"`
}

func (ir *InboxRoutes) HandleBulkMarkRead(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	body, ok := decodeAndValidate[bulkIDsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.BulkMarkRead(r.Context(), &inboxv1.BulkMarkReadRequest{
		UserId:     userID,
		MessageIds: body.IDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleBulkArchive(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	body, ok := decodeAndValidate[bulkIDsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.BulkArchive(r.Context(), &inboxv1.BulkArchiveRequest{
		UserId:     userID,
		MessageIds: body.IDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Thread + Canned-Response Handlers
// ============================================================================

func (ir *InboxRoutes) HandleListThreadMessages(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListThreadMessages(r.Context(), &inboxv1.ListThreadMessagesRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type inboxCannedResponseRequest struct {
	Name string `json:"name" validate:"required,max=200"`
	Body string `json:"body" validate:"required"`
}

type updateInboxCannedResponseRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Body *string `json:"body,omitempty" validate:"omitempty,min=1"`
}

func (ir *InboxRoutes) HandleListCannedResponses(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	resp, err := client.ListCannedResponses(r.Context(), &inboxv1.ListCannedResponsesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleCreateCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	body, ok := decodeAndValidate[inboxCannedResponseRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateCannedResponse(r.Context(), &inboxv1.CreateCannedResponseRequest{
		Name: body.Name,
		Body: body.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	b, err := cannedResponseMarshaler.Marshal(resp)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]json.RawMessage{"canned_response": b})
}

func (ir *InboxRoutes) HandleUpdateCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[updateInboxCannedResponseRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &inboxv1.UpdateCannedResponseRequest{Id: id}
	if body.Name != nil {
		grpcReq.Name = body.Name
	}
	if body.Body != nil {
		grpcReq.Body = body.Body
	}

	resp, err := client.UpdateCannedResponse(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	b, err := cannedResponseMarshaler.Marshal(resp)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]json.RawMessage{"canned_response": b})
}

func (ir *InboxRoutes) HandleDeleteCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCannedResponse(r.Context(), &inboxv1.DeleteCannedResponseRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "canned response deleted"})
}

// ============================================================================
// Team Inbox Handlers
// ============================================================================

type createTeamInboxRequest struct {
	Name           string `json:"name" validate:"required"`
	Description    string `json:"description"`
	AssignmentMode string `json:"assignment_mode" validate:"omitempty,oneof=manual round_robin"`
	Visibility     string `json:"visibility" validate:"omitempty,oneof=open private"`
}

func (ir *InboxRoutes) HandleCreateTeamInbox(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	body, ok := decodeAndValidate[createTeamInboxRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &inboxv1.CreateTeamInboxRequest{
		UserId:         userID,
		Name:           body.Name,
		Description:    body.Description,
		AssignmentMode: parseAssignmentMode(body.AssignmentMode),
		Visibility:     parseVisibility(body.Visibility),
	}

	resp, err := client.CreateTeamInbox(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

type updateTeamInboxRequest struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Description    *string `json:"description,omitempty"`
	AssignmentMode *string `json:"assignment_mode,omitempty" validate:"omitempty,oneof=manual round_robin"`
	Visibility     *string `json:"visibility,omitempty" validate:"omitempty,oneof=open private"`
}

func (ir *InboxRoutes) HandleUpdateTeamInbox(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	teamInboxID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[updateTeamInboxRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &inboxv1.UpdateTeamInboxRequest{
		TeamInboxId: teamInboxID,
		UserId:      userID,
	}

	if body.Name != nil {
		grpcReq.Name = body.Name
	}
	if body.Description != nil {
		grpcReq.Description = body.Description
	}
	if body.AssignmentMode != nil {
		mode := parseAssignmentMode(*body.AssignmentMode)
		grpcReq.AssignmentMode = &mode
	}
	if body.Visibility != nil {
		vis := parseVisibility(*body.Visibility)
		grpcReq.Visibility = &vis
	}

	resp, err := client.UpdateTeamInbox(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleDeleteTeamInbox(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	teamInboxID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteTeamInbox(r.Context(), &inboxv1.DeleteTeamInboxRequest{
		TeamInboxId: teamInboxID,
		UserId:      userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "team inbox deleted"})
}

func (ir *InboxRoutes) HandleListTeamInboxes(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.ListTeamInboxes(r.Context(), &inboxv1.ListTeamInboxesRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type addTeamMemberRequest struct {
	MemberUserID string `json:"member_user_id" validate:"required,uuid"`
	Role         string `json:"role" validate:"required"`
}

func (ir *InboxRoutes) HandleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	teamInboxID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[addTeamMemberRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &inboxv1.AddTeamMemberRequest{
		TeamInboxId:  teamInboxID,
		UserId:       userID,
		MemberUserId: body.MemberUserID,
		Role:         parseTeamMemberRole(body.Role),
	}

	resp, err := client.AddTeamMember(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (ir *InboxRoutes) HandleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	teamInboxID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	memberUserID, ok2 := validateUUIDParam(w, r, "user_id")
	if !ok2 {
		return
	}

	_, err = client.RemoveTeamMember(r.Context(), &inboxv1.RemoveTeamMemberRequest{
		TeamInboxId:  teamInboxID,
		UserId:       userID,
		MemberUserId: memberUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "member removed"})
}

func (ir *InboxRoutes) HandleListTeamMembers(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	teamInboxID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListTeamMembers(r.Context(), &inboxv1.ListTeamMembersRequest{
		TeamInboxId: teamInboxID,
		UserId:      userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleClaimMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ClaimMessage(r.Context(), &inboxv1.ClaimMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Routing Rule Handlers
// ============================================================================

type createRoutingRuleRequest struct {
	Name       string          `json:"name" validate:"required"`
	Channel    *string         `json:"channel,omitempty"`
	Conditions json.RawMessage `json:"conditions"`
	Actions    json.RawMessage `json:"actions"`
	Priority   int32           `json:"priority"`
}

func (ir *InboxRoutes) HandleCreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	body, ok := decodeAndValidate[createRoutingRuleRequest](w, r)
	if !ok {
		return
	}

	conditionsStruct, err := rawJSONToStruct(body.Conditions)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid conditions JSON")
		return
	}

	actionsStruct, err := rawJSONToStruct(body.Actions)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid actions JSON")
		return
	}

	grpcReq := &inboxv1.CreateRoutingRuleRequest{
		UserId:     userID,
		Name:       body.Name,
		Conditions: conditionsStruct,
		Actions:    actionsStruct,
		Priority:   body.Priority,
	}

	if body.Channel != nil {
		ch := parseChannelQuery(*body.Channel)
		grpcReq.Channel = &ch
	}

	resp, err := client.CreateRoutingRule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

type updateRoutingRuleRequest struct {
	Name       *string          `json:"name,omitempty" validate:"omitempty,min=1"`
	Channel    *string          `json:"channel,omitempty"`
	Conditions *json.RawMessage `json:"conditions,omitempty"`
	Actions    *json.RawMessage `json:"actions,omitempty"`
	Priority   *int32           `json:"priority,omitempty"`
	IsActive   *bool            `json:"is_active,omitempty"`
}

func (ir *InboxRoutes) HandleUpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	ruleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[updateRoutingRuleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &inboxv1.UpdateRoutingRuleRequest{
		RuleId: ruleID,
		UserId: userID,
	}

	if body.Name != nil {
		grpcReq.Name = body.Name
	}
	if body.Channel != nil {
		ch := parseChannelQuery(*body.Channel)
		grpcReq.Channel = &ch
	}
	if body.Conditions != nil {
		condStruct, structErr := rawJSONToStruct(*body.Conditions)
		if structErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid conditions JSON")
			return
		}
		grpcReq.Conditions = condStruct
	}
	if body.Actions != nil {
		actStruct, structErr := rawJSONToStruct(*body.Actions)
		if structErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid actions JSON")
			return
		}
		grpcReq.Actions = actStruct
	}
	if body.Priority != nil {
		grpcReq.Priority = body.Priority
	}
	if body.IsActive != nil {
		grpcReq.IsActive = body.IsActive
	}

	resp, err := client.UpdateRoutingRule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleDeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	ruleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteRoutingRule(r.Context(), &inboxv1.DeleteRoutingRuleRequest{
		RuleId: ruleID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "routing rule deleted"})
}

func (ir *InboxRoutes) HandleListRoutingRules(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	grpcReq := &inboxv1.ListRoutingRulesRequest{
		UserId: userID,
	}

	resp, err := client.ListRoutingRules(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *InboxRoutes) HandleTestRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getInboxClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var body struct {
		Conditions  json.RawMessage           `json:"conditions"`
		TestMessage *inboxv1.InboxMessageInfo `json:"test_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	conditionsStruct, err := rawJSONToStruct(body.Conditions)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid conditions JSON")
		return
	}

	grpcReq := &inboxv1.TestRoutingRuleRequest{
		UserId:      userID,
		Conditions:  conditionsStruct,
		TestMessage: body.TestMessage,
	}

	resp, err := client.TestRoutingRule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Enum Parsers
// ============================================================================

func parseChannelQuery(ch string) inboxv1.Channel {
	switch ch {
	case "email":
		return inboxv1.Channel_CHANNEL_EMAIL
	case "chat":
		return inboxv1.Channel_CHANNEL_CHAT
	case "notification":
		return inboxv1.Channel_CHANNEL_NOTIFICATION
	default:
		return inboxv1.Channel_CHANNEL_UNSPECIFIED
	}
}

func parseAssignmentMode(mode string) inboxv1.AssignmentMode {
	switch mode {
	case "manual":
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL
	case "round_robin":
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN
	default:
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL
	}
}

func parseVisibility(vis string) inboxv1.TeamInboxVisibility {
	switch vis {
	case "open":
		return inboxv1.TeamInboxVisibility_VISIBILITY_OPEN
	case "private":
		return inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE
	default:
		return inboxv1.TeamInboxVisibility_VISIBILITY_OPEN
	}
}

func parseTeamMemberRole(role string) inboxv1.TeamMemberRole {
	switch role {
	case "admin":
		return inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN
	case "member":
		return inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER
	default:
		return inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER
	}
}

// rawJSONToStruct converts raw JSON bytes into a protobuf Struct.
func rawJSONToStruct(data json.RawMessage) (*structpb.Struct, error) {
	if len(data) == 0 {
		return nil, nil
	}
	s := &structpb.Struct{}
	if err := s.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return s, nil
}
