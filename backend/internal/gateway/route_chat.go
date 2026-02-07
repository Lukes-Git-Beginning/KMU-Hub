package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// ChatRoutes handles HTTP routes for the chat backend service.
type ChatRoutes struct {
	registry *ServiceRegistry
}

// NewChatRoutes creates a new ChatRoutes with the given service registry.
func NewChatRoutes(registry *ServiceRegistry) *ChatRoutes {
	return &ChatRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (ch *ChatRoutes) ServiceName() string { return "chat" }

// getChatClient lazily obtains a gRPC client for the chat service.
func (ch *ChatRoutes) getChatClient() (chatv1.ChatServiceClient, error) {
	conn, err := ch.registry.GetConnection("chat")
	if err != nil {
		return nil, err
	}
	return chatv1.NewChatServiceClient(conn), nil
}

// RegisterRoutes registers all chat HTTP routes.
func (ch *ChatRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Channels
	r.Route("/api/v1/channels", func(r chi.Router) {
		r.Use(authMiddleware)
		// DM routes (before /{id} to avoid conflict)
		r.With(middleware.RequirePermission("channels", "write")).Post("/dm", ch.HandleGetOrCreateDM)
		r.With(middleware.RequirePermission("channels", "read")).Get("/dm", ch.HandleListDMs)
		// Unread counts (before /{id} to avoid conflict)
		r.With(middleware.RequirePermission("channels", "read")).Get("/unread", ch.HandleGetUnreadCounts)
		// Channel CRUD
		r.With(middleware.RequirePermission("channels", "read")).Get("/", ch.HandleListChannels)
		r.With(middleware.RequirePermission("channels", "write")).Post("/", ch.HandleCreateChannel)
		r.With(middleware.RequirePermission("channels", "read")).Get("/{id}", ch.HandleGetChannel)
		r.With(middleware.RequirePermission("channels", "write")).Put("/{id}", ch.HandleUpdateChannel)
		r.With(middleware.RequirePermission("channels", "delete")).Delete("/{id}", ch.HandleDeleteChannel)
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/archive", ch.HandleArchiveChannel)
		// Read receipts
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/read", ch.HandleMarkChannelRead)
		// Membership routes
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/join", ch.HandleJoinChannel)
		r.With(middleware.RequirePermission("channels", "write")).Post("/{id}/leave", ch.HandleLeaveChannel)
		r.With(middleware.RequirePermission("channels", "read")).Get("/{id}/members", ch.HandleGetChannelMembers)
		r.With(middleware.RequirePermission("channels", "write")).Put("/{id}/members/{userId}/role", ch.HandleUpdateMemberRole)
		// Message routes
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/messages", ch.HandleGetMessages)
		r.With(middleware.RequirePermission("messages", "write")).Post("/{id}/messages", ch.HandleSendMessage)
		// File routes
		r.With(middleware.RequirePermission("files", "read")).Get("/{id}/files", ch.HandleListChannelFiles)
	})

	// Messages (individual message operations)
	r.Route("/api/v1/messages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("mentions", "read")).Get("/mentions", ch.HandleGetUserMentions)
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/thread", ch.HandleGetThreadReplies)
		r.With(middleware.RequirePermission("messages", "write")).Put("/{id}", ch.HandleUpdateMessage)
		r.With(middleware.RequirePermission("messages", "delete")).Delete("/{id}", ch.HandleDeleteMessage)
	})

	// Files (individual file operations)
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("files", "read")).Get("/{id}/download", ch.HandleGetFileDownloadURL)
		r.With(middleware.RequirePermission("files", "read")).Get("/{id}/thumbnail", ch.HandleGetFileThumbnailURL)
		r.With(middleware.RequirePermission("files", "delete")).Delete("/{id}", ch.HandleDeleteFile)
	})

	// Chat search
	r.Route("/api/v1/chat", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/search", ch.HandleSearchChat)
	})
}

// ============================================================================
// Channel Handlers
// ============================================================================

type createChannelRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	IsPrivate        bool     `json:"is_private"`
	InitialMemberIDs []string `json:"initial_member_ids"`
}

func (ch *ChatRoutes) HandleCreateChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	grpcReq := &chatv1.CreateChannelRequest{
		Name:             req.Name,
		IsPrivate:        req.IsPrivate,
		CreatedBy:        userID,
		InitialMemberIds: req.InitialMemberIDs,
	}
	if req.Description != "" {
		grpcReq.Description = &req.Description
	}

	resp, err := client.CreateChannel(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (ch *ChatRoutes) HandleGetChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := client.GetChannel(r.Context(), &chatv1.GetChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleListChannels(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListChannels(r.Context(), &chatv1.ListChannelsRequest{
		UserId:          userID,
		IncludeArchived: includeArchived,
		Page:            int32(page),
		PageSize:        int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateChannelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (ch *ChatRoutes) HandleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &chatv1.UpdateChannelRequest{
		Id:     channelID,
		UserId: userID,
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := client.UpdateChannel(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	_, err = client.DeleteChannel(r.Context(), &chatv1.DeleteChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "channel deleted"})
}

func (ch *ChatRoutes) HandleArchiveChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := client.ArchiveChannel(r.Context(), &chatv1.ArchiveChannelRequest{
		Id:     channelID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Channel Membership Handlers
// ============================================================================

func (ch *ChatRoutes) HandleJoinChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	resp, err := client.JoinChannel(r.Context(), &chatv1.JoinChannelRequest{
		ChannelId: channelID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleLeaveChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	_, err = client.LeaveChannel(r.Context(), &chatv1.LeaveChannelRequest{
		ChannelId: channelID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "left channel"})
}

func (ch *ChatRoutes) HandleGetChannelMembers(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.GetChannelMembers(r.Context(), &chatv1.GetChannelMembersRequest{
		ChannelId: channelID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (ch *ChatRoutes) HandleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	requesterID := middleware.GetUserID(r.Context())
	if requesterID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	if _, err := uuid.Parse(targetUserID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role == "" {
		response.Error(w, http.StatusBadRequest, "role is required")
		return
	}

	resp, err := client.UpdateMemberRole(r.Context(), &chatv1.UpdateMemberRoleRequest{
		ChannelId:       channelID,
		TargetUserId:    targetUserID,
		RequesterUserId: requesterID,
		NewRole:         req.Role,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Message Handlers
// ============================================================================

type sendMessageRequest struct {
	Content          string   `json:"content"`
	ParentMessageID  *string  `json:"parent_message_id,omitempty"`
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty"`
	MentionEveryone  bool     `json:"mention_everyone,omitempty"`
}

func (ch *ChatRoutes) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	grpcReq := &chatv1.SendMessageRequest{
		ChannelId:       channelID,
		Content:         req.Content,
		CreatedBy:       userID,
		MentionEveryone: req.MentionEveryone,
	}

	if req.ParentMessageID != nil && *req.ParentMessageID != "" {
		if _, parseErr := uuid.Parse(*req.ParentMessageID); parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid parent_message_id")
			return
		}
		grpcReq.ParentMessageId = req.ParentMessageID
	}

	for _, idStr := range req.MentionedUserIDs {
		if _, parseErr := uuid.Parse(idStr); parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid mentioned_user_id")
			return
		}
		grpcReq.MentionedUserIds = append(grpcReq.MentionedUserIds, idStr)
	}

	resp, err := client.SendMessage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (ch *ChatRoutes) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	limit := parseLimit(r, 50, 100)

	grpcReq := &chatv1.GetMessagesRequest{
		ChannelId: channelID,
		UserId:    userID,
		Limit:     int32(limit),
	}

	if before := r.URL.Query().Get("before"); before != "" {
		grpcReq.Before = &before
	}
	if after := r.URL.Query().Get("after"); after != "" {
		grpcReq.After = &after
	}

	resp, err := client.GetMessages(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateMessageRequest struct {
	Content string `json:"content"`
}

func (ch *ChatRoutes) HandleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	var req updateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, "content is required")
		return
	}

	resp, err := client.UpdateMessage(r.Context(), &chatv1.UpdateMessageRequest{
		Id:      messageID,
		UserId:  userID,
		Content: req.Content,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	_, err = client.DeleteMessage(r.Context(), &chatv1.DeleteMessageRequest{
		Id:     messageID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "message deleted"})
}

// ============================================================================
// DM Handlers
// ============================================================================

type getOrCreateDMRequest struct {
	OtherUserID string `json:"other_user_id"`
}

func (ch *ChatRoutes) HandleGetOrCreateDM(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req getOrCreateDMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OtherUserID == "" {
		response.Error(w, http.StatusBadRequest, "other_user_id is required")
		return
	}

	if _, err := uuid.Parse(req.OtherUserID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid other_user_id")
		return
	}

	resp, err := client.GetOrCreateDM(r.Context(), &chatv1.GetOrCreateDMRequest{
		UserId:      userID,
		OtherUserId: req.OtherUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statusCode := http.StatusOK
	if resp.Created {
		statusCode = http.StatusCreated
	}

	response.JSON(w, statusCode, resp)
}

func (ch *ChatRoutes) HandleListDMs(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListDMs(r.Context(), &chatv1.ListDMsRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Thread Handlers
// ============================================================================

func (ch *ChatRoutes) HandleGetThreadReplies(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	messageID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(messageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}

	limit := parseLimit(r, 50, 100)

	grpcReq := &chatv1.GetThreadRepliesRequest{
		ParentMessageId: messageID,
		UserId:          userID,
		Limit:           int32(limit),
	}

	if before := r.URL.Query().Get("before"); before != "" {
		grpcReq.Before = &before
	}
	if after := r.URL.Query().Get("after"); after != "" {
		grpcReq.After = &after
	}

	resp, err := client.GetThreadReplies(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Mentions & Read Receipts Handlers
// ============================================================================

type markChannelReadRequest struct {
	MessageID string `json:"message_id"`
}

func (ch *ChatRoutes) HandleMarkChannelRead(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req markChannelReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MessageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id is required")
		return
	}

	if _, err := uuid.Parse(req.MessageID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid message_id")
		return
	}

	_, err = client.MarkChannelRead(r.Context(), &chatv1.MarkChannelReadRequest{
		ChannelId: channelID,
		UserId:    userID,
		MessageId: req.MessageID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "channel marked as read"})
}

func (ch *ChatRoutes) HandleGetUnreadCounts(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetUnreadCounts(r.Context(), &chatv1.GetUnreadCountsRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleGetUserMentions(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.GetUserMentions(r.Context(), &chatv1.GetUserMentionsRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// File Handlers
// ============================================================================

func (ch *ChatRoutes) HandleGetFileDownloadURL(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid file id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetFileDownloadURL(r.Context(), &chatv1.GetFileDownloadURLRequest{
		FileId: fileID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleGetFileThumbnailURL(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid file id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetFileThumbnailURL(r.Context(), &chatv1.GetFileThumbnailURLRequest{
		FileId: fileID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleListChannelFiles(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	channelID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(channelID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListChannelFiles(r.Context(), &chatv1.ListChannelFilesRequest{
		ChannelId: channelID,
		UserId:    userID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid file id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	_, err = client.DeleteFile(r.Context(), &chatv1.DeleteFileRequest{
		FileId: fileID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "file deleted"})
}

// ============================================================================
// Chat Search Handler
// ============================================================================

func (ch *ChatRoutes) HandleSearchChat(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	req := &chatv1.SearchChatRequest{
		Query:  query,
		UserId: userID,
	}

	if channelID := r.URL.Query().Get("channel_id"); channelID != "" {
		if _, err := uuid.Parse(channelID); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid channel_id")
			return
		}
		req.ChannelId = &channelID
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	req.Page = int32(page)

	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	req.PageSize = int32(pageSize)

	resp, err := client.SearchChat(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
