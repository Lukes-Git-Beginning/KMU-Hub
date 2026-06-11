package gateway

import (
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
		r.Use(RequireAuthenticated)
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
		r.Use(RequireAuthenticated)
		r.With(middleware.RequirePermission("mentions", "read")).Get("/mentions", ch.HandleGetUserMentions)
		// Reaction batch summary: static segment must be declared before /{id} to avoid chi ambiguity
		r.With(middleware.RequirePermission("messages", "read")).Post("/reactions/summary", ch.HandleGetReactionSummary)
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/thread", ch.HandleGetThreadReplies)
		r.With(middleware.RequirePermission("messages", "write")).Put("/{id}", ch.HandleUpdateMessage)
		r.With(middleware.RequirePermission("messages", "delete")).Delete("/{id}", ch.HandleDeleteMessage)
		// Reaction routes per message
		r.With(middleware.RequirePermission("messages", "write")).Post("/{id}/reactions", ch.HandleToggleReaction)
		r.With(middleware.RequirePermission("messages", "read")).Get("/{id}/reactions", ch.HandleListReactions)
	})

	// Files (individual file operations)
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)
		r.With(middleware.RequirePermission("files", "read")).Get("/{id}/download", ch.HandleGetFileDownloadURL)
		r.With(middleware.RequirePermission("files", "read")).Get("/{id}/thumbnail", ch.HandleGetFileThumbnailURL)
		r.With(middleware.RequirePermission("files", "delete")).Delete("/{id}", ch.HandleDeleteFile)
	})

	// Chat search
	r.Route("/api/v1/chat", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)
		r.Get("/search", ch.HandleSearchChat)
	})
}

// ============================================================================
// Channel Handlers
// ============================================================================

type createChannelRequest struct {
	Name             string   `json:"name" validate:"required"`
	Description      string   `json:"description"`
	IsPrivate        bool     `json:"is_private"`
	InitialMemberIDs []string `json:"initial_member_ids" validate:"omitempty,dive,uuid"`
}

func (ch *ChatRoutes) HandleCreateChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createChannelRequest](w, r)
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
}

func (ch *ChatRoutes) HandleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateChannelRequest](w, r)
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
	Role string `json:"role" validate:"required,oneof=admin member"`
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	targetUserID, ok := validateUUIDParam(w, r, "userId")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateMemberRoleRequest](w, r)
	if !ok {
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
	Content          string   `json:"content" validate:"required"`
	ParentMessageID  *string  `json:"parent_message_id,omitempty" validate:"omitempty,uuid"`
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty" validate:"omitempty,dive,uuid"`
	MentionEveryone  bool     `json:"mention_everyone,omitempty"`
}

func (ch *ChatRoutes) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[sendMessageRequest](w, r)
	if !ok {
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	grpcReq := &chatv1.SendMessageRequest{
		TenantId:         tenantID.String(),
		ChannelId:        channelID,
		Content:          req.Content,
		CreatedBy:        userID,
		MentionEveryone:  req.MentionEveryone,
		MentionedUserIds: req.MentionedUserIDs,
	}

	if req.ParentMessageID != nil {
		grpcReq.ParentMessageId = req.ParentMessageID
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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
	Content string `json:"content" validate:"required"`
}

func (ch *ChatRoutes) HandleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateMessageRequest](w, r)
	if !ok {
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

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
	OtherUserID string `json:"other_user_id" validate:"required,uuid"`
}

func (ch *ChatRoutes) HandleGetOrCreateDM(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[getOrCreateDMRequest](w, r)
	if !ok {
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

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
	MessageID string `json:"message_id" validate:"required,uuid"`
}

func (ch *ChatRoutes) HandleMarkChannelRead(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[markChannelReadRequest](w, r)
	if !ok {
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

	fileID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

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

	fileID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

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

	channelID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

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

	fileID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

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

// ============================================================================
// Reaction Handlers (Phase 8)
// ============================================================================

type toggleReactionRequest struct {
	Emoji string `json:"emoji" validate:"required"`
}

func (ch *ChatRoutes) HandleToggleReaction(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[toggleReactionRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ToggleReaction(r.Context(), &chatv1.ToggleReactionRequest{
		MessageId: messageID,
		UserId:    userID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ch *ChatRoutes) HandleListReactions(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	messageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListReactions(r.Context(), &chatv1.ListReactionsRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type getReactionSummaryRequest struct {
	MessageIDs []string `json:"message_ids" validate:"required,min=1,dive,uuid"`
}

func (ch *ChatRoutes) HandleGetReactionSummary(w http.ResponseWriter, r *http.Request) {
	client, err := ch.getChatClient()
	if err != nil {
		respondServiceUnavailable(w, ch.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[getReactionSummaryRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.GetReactionSummary(r.Context(), &chatv1.GetReactionSummaryRequest{
		MessageIds: req.MessageIDs,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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
