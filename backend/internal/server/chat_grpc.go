package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/chat/bookmark"
	"github.com/kmuhub/kmuhub/internal/chat/channel"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/message"
	"github.com/kmuhub/kmuhub/internal/chat/search"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/reaction"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// ChatGRPCServer implements the Chat gRPC service
type ChatGRPCServer struct {
	chatv1.UnimplementedChatServiceServer
	channelService  *channel.Service
	messageService  *message.Service
	fileService     *file.Service
	searchService   *search.Service
	reactionService *reaction.Service
	bookmarkService *bookmark.Service
}

// NewChatGRPCServer creates a new Chat gRPC server.
// Signature change (fe-messages-bookmark): added bookmarkService *bookmark.Service parameter.
func NewChatGRPCServer(
	channelService *channel.Service,
	messageService *message.Service,
	fileService *file.Service,
	searchService *search.Service,
	reactionService *reaction.Service,
	bookmarkService *bookmark.Service,
) *ChatGRPCServer {
	return &ChatGRPCServer{
		channelService:  channelService,
		messageService:  messageService,
		fileService:     fileService,
		searchService:   searchService,
		reactionService: reactionService,
		bookmarkService: bookmarkService,
	}
}

// ============================================================================
// Channels
// ============================================================================

func (s *ChatGRPCServer) CreateChannel(ctx context.Context, req *chatv1.CreateChannelRequest) (*chatv1.CreateChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	var initialMemberIDs []uuid.UUID
	for _, idStr := range req.InitialMemberIds {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid initial member id")
		}
		initialMemberIDs = append(initialMemberIDs, id)
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	input := channel.CreateInput{
		TenantID:         tenantID,
		Name:             req.Name,
		Description:      description,
		IsPrivate:        req.IsPrivate,
		CreatedBy:        createdBy,
		InitialMemberIDs: initialMemberIDs,
	}

	ch, err := s.channelService.Create(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.CreateChannelResponse{
		Channel: toChannelInfo(ch),
	}, nil
}

func (s *ChatGRPCServer) GetChannel(ctx context.Context, req *chatv1.GetChannelRequest) (*chatv1.GetChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	ch, err := s.channelService.GetByID(ctx, id, userID, tenantID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.GetChannelResponse{
		Channel: toChannelInfo(ch),
	}, nil
}

func (s *ChatGRPCServer) ListChannels(ctx context.Context, req *chatv1.ListChannelsRequest) (*chatv1.ListChannelsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	input := channel.ListInput{
		TenantID:        tenantID,
		UserID:          userID,
		IncludeArchived: req.IncludeArchived,
		Page:            int(req.Page),
		PageSize:        int(req.PageSize),
	}

	channels, total, err := s.channelService.List(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		infos = append(infos, toChannelInfo(ch))
	}

	return &chatv1.ListChannelsResponse{
		Channels: infos,
		Total:    int32(total),
	}, nil
}

func (s *ChatGRPCServer) UpdateChannel(ctx context.Context, req *chatv1.UpdateChannelRequest) (*chatv1.UpdateChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	input := channel.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
	}

	ch, err := s.channelService.Update(ctx, id, userID, tenantID, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.UpdateChannelResponse{
		Channel: toChannelInfo(ch),
	}, nil
}

func (s *ChatGRPCServer) DeleteChannel(ctx context.Context, req *chatv1.DeleteChannelRequest) (*chatv1.DeleteChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.channelService.Delete(ctx, id, userID, tenantID); err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.DeleteChannelResponse{}, nil
}

func (s *ChatGRPCServer) ArchiveChannel(ctx context.Context, req *chatv1.ArchiveChannelRequest) (*chatv1.ArchiveChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	ch, err := s.channelService.Archive(ctx, id, userID, tenantID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.ArchiveChannelResponse{
		Channel: toChannelInfo(ch),
	}, nil
}

// ============================================================================
// Channel Memberships
// ============================================================================

func (s *ChatGRPCServer) JoinChannel(ctx context.Context, req *chatv1.JoinChannelRequest) (*chatv1.JoinChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	member, err := s.channelService.Join(ctx, channelID, userID, tenantID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.JoinChannelResponse{
		Member: toMemberInfo(member),
	}, nil
}

func (s *ChatGRPCServer) LeaveChannel(ctx context.Context, req *chatv1.LeaveChannelRequest) (*chatv1.LeaveChannelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.channelService.Leave(ctx, channelID, userID, tenantID); err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.LeaveChannelResponse{}, nil
}

func (s *ChatGRPCServer) GetChannelMembers(ctx context.Context, req *chatv1.GetChannelMembersRequest) (*chatv1.GetChannelMembersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	members, total, err := s.channelService.GetMembers(ctx, channelID, tenantID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.MemberInfo, 0, len(members))
	for _, m := range members {
		infos = append(infos, toMemberInfo(m))
	}

	return &chatv1.GetChannelMembersResponse{
		Members: infos,
		Total:   int32(total),
	}, nil
}

func (s *ChatGRPCServer) UpdateMemberRole(ctx context.Context, req *chatv1.UpdateMemberRoleRequest) (*chatv1.UpdateMemberRoleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	targetUserID, err := uuid.Parse(req.TargetUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target user id")
	}

	requesterUserID, err := uuid.Parse(req.RequesterUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester user id")
	}

	newRole := models.ChannelRole(req.NewRole)
	if !newRole.IsValid() {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	member, err := s.channelService.UpdateMemberRole(ctx, channelID, targetUserID, requesterUserID, tenantID, newRole)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.UpdateMemberRoleResponse{
		Member: toMemberInfo(member),
	}, nil
}

// ============================================================================
// Messages
// ============================================================================

func (s *ChatGRPCServer) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	chatTenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	input := message.CreateInput{
		TenantID:        chatTenantID,
		ChannelID:       channelID,
		Content:         req.Content,
		MentionEveryone: req.MentionEveryone,
	}

	// Guest messages: guest_session_id is set, created_by may be empty
	if req.GuestSessionId != nil && *req.GuestSessionId != "" {
		guestID, parseErr := uuid.Parse(*req.GuestSessionId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid guest_session_id")
		}
		input.GuestSessionID = &guestID
	} else {
		createdBy, parseErr := uuid.Parse(req.CreatedBy)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
		}
		input.CreatedBy = createdBy
	}

	if req.ParentMessageId != nil && *req.ParentMessageId != "" {
		parentID, parseErr := uuid.Parse(*req.ParentMessageId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_message_id")
		}
		input.ParentMessageID = &parentID
	}

	for _, idStr := range req.MentionedUserIds {
		uid, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid mentioned_user_id")
		}
		input.MentionedUserIDs = append(input.MentionedUserIDs, uid)
	}

	msg, err := s.messageService.Create(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.SendMessageResponse{
		Message: toMessageInfo(msg),
	}, nil
}

func (s *ChatGRPCServer) GetMessages(ctx context.Context, req *chatv1.GetMessagesRequest) (*chatv1.GetMessagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	input := message.ListInput{
		TenantID:  tenantID,
		ChannelID: channelID,
		UserID:    userID,
		Limit:     int(req.Limit),
	}

	// Guest access: uuid.Nil means guest — skip membership, verify channel is guest-enabled
	if userID == uuid.Nil {
		guestEnabled, chkErr := s.channelService.IsGuestEnabled(ctx, channelID, tenantID)
		if chkErr != nil || !guestEnabled {
			return nil, status.Error(codes.PermissionDenied, "channel does not allow guest access")
		}
		input.SkipMembershipCheck = true
	}

	if req.Before != nil && *req.Before != "" {
		beforeID, parseErr := uuid.Parse(*req.Before)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid before message id")
		}
		input.Before = &beforeID
	}

	if req.After != nil && *req.After != "" {
		afterID, parseErr := uuid.Parse(*req.After)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid after message id")
		}
		input.After = &afterID
	}

	messages, hasMore, err := s.messageService.List(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.MessageInfo, 0, len(messages))
	for _, m := range messages {
		infos = append(infos, toMessageInfo(m))
	}

	return &chatv1.GetMessagesResponse{
		Messages: infos,
		HasMore:  hasMore,
	}, nil
}

func (s *ChatGRPCServer) UpdateMessage(ctx context.Context, req *chatv1.UpdateMessageRequest) (*chatv1.UpdateMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	input := message.UpdateInput{
		Content: req.Content,
	}

	msg, err := s.messageService.Update(ctx, id, tenantID, userID, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.UpdateMessageResponse{
		Message: toMessageInfo(msg),
	}, nil
}

func (s *ChatGRPCServer) DeleteMessage(ctx context.Context, req *chatv1.DeleteMessageRequest) (*chatv1.DeleteMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.messageService.Delete(ctx, id, tenantID, userID); err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.DeleteMessageResponse{}, nil
}

// ============================================================================
// Direct Messages (Sprint 2)
// ============================================================================

func (s *ChatGRPCServer) GetOrCreateDM(ctx context.Context, req *chatv1.GetOrCreateDMRequest) (*chatv1.GetOrCreateDMResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	otherUserID, err := uuid.Parse(req.OtherUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid other_user_id")
	}

	result, err := s.channelService.GetOrCreateDM(ctx, channel.GetOrCreateDMInput{
		TenantID:    tenantID,
		UserID:      userID,
		OtherUserID: otherUserID,
	})
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.GetOrCreateDMResponse{
		Channel: toChannelInfo(result.Channel),
		Created: result.Created,
	}, nil
}

func (s *ChatGRPCServer) ListDMs(ctx context.Context, req *chatv1.ListDMsRequest) (*chatv1.ListDMsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	channels, total, err := s.channelService.ListDMs(ctx, userID, tenantID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		infos = append(infos, toChannelInfo(ch))
	}

	return &chatv1.ListDMsResponse{
		Channels: infos,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Threads (Sprint 2)
// ============================================================================

func (s *ChatGRPCServer) GetThreadReplies(ctx context.Context, req *chatv1.GetThreadRepliesRequest) (*chatv1.GetThreadRepliesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	parentMessageID, err := uuid.Parse(req.ParentMessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent_message_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := message.GetThreadRepliesInput{
		TenantID:        tenantID,
		ParentMessageID: parentMessageID,
		UserID:          userID,
		Limit:           int(req.Limit),
	}

	if req.Before != nil && *req.Before != "" {
		beforeID, parseErr := uuid.Parse(*req.Before)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid before message id")
		}
		input.Before = &beforeID
	}

	if req.After != nil && *req.After != "" {
		afterID, parseErr := uuid.Parse(*req.After)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid after message id")
		}
		input.After = &afterID
	}

	result, err := s.messageService.GetThreadReplies(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	var replyInfos []*chatv1.MessageInfo
	for _, reply := range result.Replies {
		replyInfos = append(replyInfos, toMessageInfo(reply))
	}

	return &chatv1.GetThreadRepliesResponse{
		Parent:  toMessageInfo(result.Parent),
		Replies: replyInfos,
		HasMore: result.HasMore,
	}, nil
}

// ============================================================================
// Mentions & Read Receipts (Sprint 3)
// ============================================================================

func (s *ChatGRPCServer) MarkChannelRead(ctx context.Context, req *chatv1.MarkChannelReadRequest) (*chatv1.MarkChannelReadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	// Get the message to find its created_at (tenant-scoped)
	msg, err := s.messageService.GetByID(ctx, messageID, tenantID, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	if err := s.channelService.MarkRead(ctx, channelID, userID, tenantID, msg.CreatedAt); err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.MarkChannelReadResponse{}, nil
}

func (s *ChatGRPCServer) GetUnreadCounts(ctx context.Context, req *chatv1.GetUnreadCountsRequest) (*chatv1.GetUnreadCountsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	counts, err := s.channelService.GetUnreadCounts(ctx, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	result := make(map[string]int32, len(counts))
	for channelID, count := range counts {
		result[channelID.String()] = int32(count)
	}

	return &chatv1.GetUnreadCountsResponse{
		UnreadCounts: result,
	}, nil
}

func (s *ChatGRPCServer) GetUserMentions(ctx context.Context, req *chatv1.GetUserMentionsRequest) (*chatv1.GetUserMentionsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mentions, total, err := s.messageService.GetMentionsForUser(ctx, userID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapChatError(err)
	}

	var details []*chatv1.MentionDetail
	for _, m := range mentions {
		details = append(details, &chatv1.MentionDetail{
			MessageId:       m.MessageID.String(),
			ChannelId:       m.ChannelID.String(),
			Content:         m.Content,
			MentionType:     toProtoMentionType(m.MentionType),
			SenderFirstName: m.SenderFirstName,
			SenderLastName:  m.SenderLastName,
			CreatedAt:       m.CreatedAt,
		})
	}

	return &chatv1.GetUserMentionsResponse{
		Mentions: details,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Files (Sprint 4)
// ============================================================================

func (s *ChatGRPCServer) GetFileDownloadURL(ctx context.Context, req *chatv1.GetFileDownloadURLRequest) (*chatv1.GetFileDownloadURLResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	url, err := s.fileService.GetDownloadURL(ctx, fileID, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.GetFileDownloadURLResponse{Url: url}, nil
}

func (s *ChatGRPCServer) GetFileThumbnailURL(ctx context.Context, req *chatv1.GetFileThumbnailURLRequest) (*chatv1.GetFileThumbnailURLResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	url, err := s.fileService.GetThumbnailURL(ctx, fileID, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.GetFileThumbnailURLResponse{Url: url}, nil
}

func (s *ChatGRPCServer) ListChannelFiles(ctx context.Context, req *chatv1.ListChannelFilesRequest) (*chatv1.ListChannelFilesResponse, error) {
	channelID, err := uuid.Parse(req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid channel_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	files, total, err := s.fileService.ListChannelFiles(ctx, channelID, userID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.FileInfo, 0, len(files))
	for _, f := range files {
		infos = append(infos, toFileInfo(f))
	}

	return &chatv1.ListChannelFilesResponse{
		Files: infos,
		Total: int32(total),
	}, nil
}

func (s *ChatGRPCServer) DeleteFile(ctx context.Context, req *chatv1.DeleteFileRequest) (*chatv1.DeleteFileResponse, error) {
	fileID, err := uuid.Parse(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.fileService.Delete(ctx, fileID, userID); err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.DeleteFileResponse{}, nil
}

// ============================================================================
// Search (Sprint 4)
// ============================================================================

func (s *ChatGRPCServer) SearchChat(ctx context.Context, req *chatv1.SearchChatRequest) (*chatv1.SearchChatResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := search.SearchInput{
		Query:    req.Query,
		UserID:   userID,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.ChannelId != nil && *req.ChannelId != "" {
		chID, parseErr := uuid.Parse(*req.ChannelId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid channel_id")
		}
		input.ChannelID = &chID
	}

	results, total, err := s.searchService.Search(ctx, input)
	if err != nil {
		return nil, mapChatError(err)
	}

	protos := make([]*chatv1.ChatSearchResultProto, 0, len(results))
	for _, r := range results {
		protos = append(protos, toChatSearchResultProto(&r))
	}

	return &chatv1.SearchChatResponse{
		Results: protos,
		Total:   int32(total),
	}, nil
}

// ============================================================================
// Reactions (Phase 8)
// ============================================================================

func (s *ChatGRPCServer) ToggleReaction(ctx context.Context, req *chatv1.ToggleReactionRequest) (*chatv1.ToggleReactionResponse, error) {
	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	result, err := s.reactionService.ToggleReaction(ctx, messageID, userID, req.Emoji)
	if err != nil {
		return nil, mapChatError(err)
	}

	reactions := make([]*chatv1.Reaction, 0, len(result.Reactions))
	for _, rx := range result.Reactions {
		for _, uid := range rx.UserIDs {
			reactions = append(reactions, &chatv1.Reaction{
				MessageId: rx.MessageID.String(),
				UserId:    uid.String(),
				Emoji:     rx.Emoji,
			})
		}
	}

	return &chatv1.ToggleReactionResponse{
		Reactions: reactions,
		Added:     result.Added,
	}, nil
}

func (s *ChatGRPCServer) ListReactions(ctx context.Context, req *chatv1.ListReactionsRequest) (*chatv1.ListReactionsResponse, error) {
	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	rxList, err := s.reactionService.ListReactions(ctx, messageID)
	if err != nil {
		return nil, mapChatError(err)
	}

	reactions := make([]*chatv1.Reaction, 0, len(rxList))
	for _, rx := range rxList {
		reactions = append(reactions, &chatv1.Reaction{
			MessageId: rx.MessageID.String(),
			UserId:    rx.UserID.String(),
			Emoji:     rx.Emoji,
			CreatedAt: rx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &chatv1.ListReactionsResponse{Reactions: reactions}, nil
}

func (s *ChatGRPCServer) GetReactionSummary(ctx context.Context, req *chatv1.GetReactionSummaryRequest) (*chatv1.GetReactionSummaryResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	var messageIDs []uuid.UUID
	for _, idStr := range req.MessageIds {
		mid, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid message_id: "+idStr)
		}
		messageIDs = append(messageIDs, mid)
	}

	batchResult, err := s.reactionService.GetBatchReactionSummary(ctx, messageIDs, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	var summaries []*chatv1.ReactionSummary
	for _, msgSummaries := range batchResult {
		for _, rs := range msgSummaries {
			var userIDStrings []string
			for _, uid := range rs.UserIDs {
				userIDStrings = append(userIDStrings, uid.String())
			}
			summaries = append(summaries, &chatv1.ReactionSummary{
				MessageId:          rs.MessageID.String(),
				Emoji:              rs.Emoji,
				Count:              int32(rs.Count),
				UserIds:            userIDStrings,
				CurrentUserReacted: rs.CurrentUserReacted,
			})
		}
	}

	return &chatv1.GetReactionSummaryResponse{Summaries: summaries}, nil
}

// ============================================================================
// Bookmarks
// ============================================================================

func (s *ChatGRPCServer) ToggleBookmark(ctx context.Context, req *chatv1.ToggleBookmarkRequest) (*chatv1.ToggleBookmarkResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	bookmarked, err := s.bookmarkService.Toggle(ctx, tenantID, userID, messageID)
	if err != nil {
		return nil, mapChatError(err)
	}

	return &chatv1.ToggleBookmarkResponse{Bookmarked: bookmarked}, nil
}

func (s *ChatGRPCServer) ListBookmarks(ctx context.Context, req *chatv1.ListBookmarksRequest) (*chatv1.ListBookmarksResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	messages, err := s.bookmarkService.List(ctx, tenantID, userID)
	if err != nil {
		return nil, mapChatError(err)
	}

	infos := make([]*chatv1.MessageInfo, 0, len(messages))
	for _, m := range messages {
		infos = append(infos, toMessageInfo(m))
	}

	return &chatv1.ListBookmarksResponse{Messages: infos}, nil
}

// ============================================================================
// Converters
// ============================================================================

func toChannelInfo(ch *models.ChannelWithRelations) *chatv1.ChannelInfo {
	info := &chatv1.ChannelInfo{
		Id:             ch.ID.String(),
		Name:           ch.Name,
		IsPrivate:      ch.IsPrivate,
		IsDm:           ch.IsDM,
		IsArchived:     ch.IsArchived,
		IsGuestEnabled: ch.IsGuestEnabled,
		CreatedBy:      ch.CreatedBy.String(),
		CreatedAt:      ch.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      ch.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		MemberCount:    int32(ch.MemberCount),
		UnreadCount:    int32(ch.UnreadCount),
	}

	if ch.Description != nil {
		info.Description = ch.Description
	}

	if ch.MyRole != nil {
		role := ch.MyRole.String()
		info.MyRole = &role
	}

	if ch.LastMessage != nil {
		info.LastMessage = toMessageInfoFromBase(ch.LastMessage)
	}

	return info
}

func toMemberInfo(m *models.ChannelMember) *chatv1.MemberInfo {
	info := &chatv1.MemberInfo{
		UserId:    m.UserID.String(),
		Role:      m.Role.String(),
		JoinedAt:  m.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
	}

	if m.LastReadAt != nil {
		lastReadAt := m.LastReadAt.Format("2006-01-02T15:04:05Z07:00")
		info.LastReadAt = &lastReadAt
	}

	return info
}

func toMessageInfo(m *models.MessageWithSender) *chatv1.MessageInfo {
	info := &chatv1.MessageInfo{
		Id:              m.ID.String(),
		ChannelId:       m.ChannelID.String(),
		Content:         m.Content,
		IsDeleted:       m.IsDeleted,
		CreatedAt:       m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		SenderFirstName: m.SenderFirstName,
		SenderLastName:  m.SenderLastName,
		ReplyCount:      int32(m.ReplyCount),
	}

	if m.CreatedBy != nil {
		info.CreatedBy = m.CreatedBy.String()
	}

	if m.EditedAt != nil {
		editedAt := m.EditedAt.Format("2006-01-02T15:04:05Z07:00")
		info.EditedAt = &editedAt
	}

	if m.ParentMessageID != nil {
		parentID := m.ParentMessageID.String()
		info.ParentMessageId = &parentID
	}

	if m.GuestSessionID != nil {
		guestID := m.GuestSessionID.String()
		info.GuestSessionId = &guestID
	}

	if m.GuestDisplayName != "" {
		info.GuestDisplayName = &m.GuestDisplayName
	}

	for _, mention := range m.Mentions {
		info.Mentions = append(info.Mentions, toMentionInfo(&mention))
	}

	for _, f := range m.Files {
		info.Files = append(info.Files, toFileInfo(&f))
	}

	return info
}

func toMentionInfo(m *models.MentionWithUser) *chatv1.MentionInfo {
	return &chatv1.MentionInfo{
		UserId:      m.UserID.String(),
		MentionType: toProtoMentionType(m.MentionType),
		FirstName:   m.FirstName,
		LastName:    m.LastName,
	}
}

func toProtoMentionType(mt models.MentionType) chatv1.MentionType {
	switch mt {
	case models.MentionTypeChannel:
		return chatv1.MentionType_MENTION_TYPE_CHANNEL
	case models.MentionTypeEveryone:
		return chatv1.MentionType_MENTION_TYPE_EVERYONE
	default:
		return chatv1.MentionType_MENTION_TYPE_USER
	}
}

func toMessageInfoFromBase(m *models.Message) *chatv1.MessageInfo {
	info := &chatv1.MessageInfo{
		Id:         m.ID.String(),
		ChannelId:  m.ChannelID.String(),
		Content:    m.Content,
		IsDeleted:  m.IsDeleted,
		CreatedAt:  m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		ReplyCount: int32(m.ReplyCount),
	}

	if m.CreatedBy != nil {
		info.CreatedBy = m.CreatedBy.String()
	}

	if m.EditedAt != nil {
		editedAt := m.EditedAt.Format("2006-01-02T15:04:05Z07:00")
		info.EditedAt = &editedAt
	}

	if m.ParentMessageID != nil {
		parentID := m.ParentMessageID.String()
		info.ParentMessageId = &parentID
	}

	return info
}

func toFileInfo(f *models.ChatFileWithUploader) *chatv1.FileInfo {
	info := &chatv1.FileInfo{
		Id:                f.ID.String(),
		ChannelId:         f.ChannelID.String(),
		Filename:          f.Filename,
		MimeType:          f.MimeType,
		FileSize:          f.FileSize,
		UploadedBy:        f.UploadedBy.String(),
		CreatedAt:         f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UploaderFirstName: f.UploaderFirstName,
		UploaderLastName:  f.UploaderLastName,
	}

	if f.MessageID != nil {
		msgID := f.MessageID.String()
		info.MessageId = &msgID
	}

	return info
}

func toChatSearchResultProto(r *models.ChatSearchResult) *chatv1.ChatSearchResultProto {
	proto := &chatv1.ChatSearchResultProto{
		Type:        string(r.Type),
		Id:          r.ID.String(),
		ChannelId:   r.ChannelID.String(),
		ChannelName: r.ChannelName,
		Score:       r.Score,
		Snippet:     r.Snippet,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		FirstName:   r.FirstName,
		LastName:    r.LastName,
	}

	if r.MessageID != nil {
		msgID := r.MessageID.String()
		proto.MessageId = &msgID
	}
	if r.ParentMessageID != nil {
		parentID := r.ParentMessageID.String()
		proto.ParentMessageId = &parentID
	}
	if r.FileID != nil {
		fileID := r.FileID.String()
		proto.FileId = &fileID
	}
	if r.Filename != nil {
		proto.Filename = r.Filename
	}
	if r.MimeType != nil {
		proto.MimeType = r.MimeType
	}
	if r.FileSize != nil {
		proto.FileSize = r.FileSize
	}

	return proto
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapChatError(err error) error {
	switch {
	// Channel errors
	case errors.Is(err, channel.ErrChannelNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, channel.ErrChannelNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, channel.ErrChannelArchived):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, channel.ErrNotChannelMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, channel.ErrAlreadyMember):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, channel.ErrCannotJoinPrivate):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, channel.ErrCannotLeaveOwner):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, channel.ErrCannotChangeOwner):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrInvalidRole):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, channel.ErrCannotDMSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, channel.ErrCannotJoinDM):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrCannotLeaveDM):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrCannotDeleteDM):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, channel.ErrChannelHasCallSessions):
		return status.Error(codes.FailedPrecondition, err.Error())

	// Message errors
	case errors.Is(err, message.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrContentRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, message.ErrChannelNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrChannelArchived):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, message.ErrNotChannelMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, message.ErrNotMessageAuthor):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, message.ErrNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, message.ErrMessageDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, message.ErrParentNotInChannel):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, message.ErrNestedThreads):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, message.ErrParentDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, message.ErrInvalidMention):
		return status.Error(codes.InvalidArgument, err.Error())

	// File errors
	case errors.Is(err, file.ErrNotChannelMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, file.ErrChannelArchived):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, file.ErrFileTooLarge):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, file.ErrInvalidMimeType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, file.ErrQuotaExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, file.ErrFileNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, file.ErrFileDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, file.ErrNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, file.ErrScanFailed):
		return status.Error(codes.FailedPrecondition, err.Error())

	// Search errors
	case errors.Is(err, search.ErrQueryTooShort):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, search.ErrQueryTooLong):
		return status.Error(codes.InvalidArgument, err.Error())

	// Reaction errors
	case errors.Is(err, reaction.ErrEmojiRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, reaction.ErrEmojiTooLong):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		// Log the original error so unmapped failures (DB constraint violations,
		// connection errors, …) don't vanish into a generic codes.Internal.
		// Without this, a missing tenant_id INSERT silently surfaces as 500
		// "internal error" with empty service logs — see Welle-0.6 debug session.
		slog.Error("unmapped chat service error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
