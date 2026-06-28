package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/inbox/thread"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// InboxGRPCServer implements the Inbox gRPC service.
type InboxGRPCServer struct {
	inboxv1.UnimplementedInboxServiceServer
	messageService *message.Service
	teamService    *team.Service
	routingService *routing.Service
	threadService  *thread.Service
}

// NewInboxGRPCServer creates a new Inbox gRPC server.
func NewInboxGRPCServer(
	messageSvc *message.Service,
	teamSvc *team.Service,
	routingSvc *routing.Service,
	threadSvc *thread.Service,
) *InboxGRPCServer {
	return &InboxGRPCServer{
		messageService: messageSvc,
		teamService:    teamSvc,
		routingService: routingSvc,
		threadService:  threadSvc,
	}
}

// ============================================================================
// Messages (14 RPCs)
// ============================================================================

func (s *InboxGRPCServer) ListMessages(ctx context.Context, req *inboxv1.ListMessagesRequest) (*inboxv1.ListMessagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	filter := message.ListFilter{
		TenantID: tenantID,
		UserID:   userID,
		PageSize: int(req.PageSize),
	}

	if req.Channel != nil {
		ch := channelToString(*req.Channel)
		filter.Channel = &ch
	}
	if req.IsRead != nil {
		filter.IsRead = req.IsRead
	}
	if req.IsStarred != nil {
		filter.IsStarred = req.IsStarred
	}
	if req.IsArchived != nil {
		filter.IsArchived = req.IsArchived
	}
	if req.TeamInboxId != nil {
		teamID, parseErr := uuid.Parse(*req.TeamInboxId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
		}
		filter.TeamInboxID = &teamID
	}
	if req.SearchQuery != nil {
		filter.Search = req.SearchQuery
	}

	// Decode page_token as cursor (received_at:id)
	if req.PageToken != "" {
		cursorAt, cursorID, parseErr := parsePageToken(req.PageToken)
		if parseErr == nil {
			filter.CursorReceivedAt = &cursorAt
			filter.CursorID = &cursorID
		}
	}

	msgs, total, err := s.messageService.List(ctx, filter)
	if err != nil {
		return nil, mapInboxError(err)
	}

	var infos []*inboxv1.InboxMessageInfo
	for _, m := range msgs {
		infos = append(infos, toInboxMessageInfo(m))
	}

	// Build next_page_token from the last message
	var nextPageToken string
	if len(msgs) > 0 && len(msgs) == filter.PageSize {
		last := msgs[len(msgs)-1]
		nextPageToken = buildPageToken(last.ReceivedAt, last.ID)
	}

	return &inboxv1.ListMessagesResponse{
		Messages:      infos,
		NextPageToken: nextPageToken,
		TotalCount:    int32(total),
	}, nil
}

func (s *InboxGRPCServer) GetMessage(ctx context.Context, req *inboxv1.GetMessageRequest) (*inboxv1.GetMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.GetMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) MarkRead(ctx context.Context, req *inboxv1.MarkReadRequest) (*inboxv1.MarkReadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if err := s.messageService.MarkRead(ctx, msgID, tenantID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.MarkReadResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) MarkUnread(ctx context.Context, req *inboxv1.MarkUnreadRequest) (*inboxv1.MarkUnreadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if err := s.messageService.MarkUnread(ctx, msgID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.MarkUnreadResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) ToggleStar(ctx context.Context, req *inboxv1.ToggleStarRequest) (*inboxv1.ToggleStarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if err := s.messageService.ToggleStar(ctx, msgID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.ToggleStarResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) ArchiveMessage(ctx context.Context, req *inboxv1.ArchiveMessageRequest) (*inboxv1.ArchiveMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if err := s.messageService.Archive(ctx, msgID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.ArchiveMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) UnarchiveMessage(ctx context.Context, req *inboxv1.UnarchiveMessageRequest) (*inboxv1.UnarchiveMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if err := s.messageService.Unarchive(ctx, msgID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.UnarchiveMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) SnoozeMessage(ctx context.Context, req *inboxv1.SnoozeMessageRequest) (*inboxv1.SnoozeMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if req.SnoozeUntil == nil {
		return nil, status.Error(codes.InvalidArgument, "snooze_until is required")
	}

	snoozeUntil := req.SnoozeUntil.AsTime()
	if err := s.messageService.Snooze(ctx, msgID, snoozeUntil); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.SnoozeMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) UnsnoozeMessage(ctx context.Context, req *inboxv1.UnsnoozeMessageRequest) (*inboxv1.UnsnoozeMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	// Unsnooze is equivalent to snooze with a past time, handled via repo directly
	// The message service doesn't have a dedicated Unsnooze, use the repo pattern
	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}
	msg.SnoozedUntil = nil
	if err := s.messageService.Update(ctx, msg); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.UnsnoozeMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) ReplyToMessage(ctx context.Context, req *inboxv1.ReplyToMessageRequest) (*inboxv1.ReplyToMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.messageService.Reply(ctx, msgID, tenantID, userID, req.Body); err != nil {
		return nil, mapInboxError(err)
	}

	// Persist the reply into the durable thread (best-effort: the reply itself
	// already succeeded via the channel adapter).
	if err := s.threadService.AppendReply(ctx, tenantID, msgID, userID, req.Body); err != nil {
		slog.Warn("inbox: failed to persist reply to thread", "message_id", msgID, "error", err)
	}

	return &inboxv1.ReplyToMessageResponse{
		Success: true,
	}, nil
}

// ============================================================================
// Thread + Canned Responses
// ============================================================================

func (s *InboxGRPCServer) ListThreadMessages(ctx context.Context, req *inboxv1.ListThreadMessagesRequest) (*inboxv1.ListThreadMessagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	// Load the originating message (enforces tenant isolation) to seed the
	// inbound original on first read.
	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	seed := thread.InboundSeed{
		SenderName: msg.SenderName,
		Body:       msg.Preview,
		CreatedAt:  msg.ReceivedAt,
	}
	msgs, err := s.threadService.ListThread(ctx, tenantID, msgID, seed)
	if err != nil {
		return nil, mapInboxError(err)
	}

	out := make([]*inboxv1.ThreadMessageInfo, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toThreadMessageInfo(m))
	}
	return &inboxv1.ListThreadMessagesResponse{Messages: out}, nil
}

func (s *InboxGRPCServer) CreateCannedResponse(ctx context.Context, req *inboxv1.CreateCannedResponseRequest) (*inboxv1.CannedResponseInfo, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	if req.Name == "" || req.Body == "" {
		return nil, status.Error(codes.InvalidArgument, "name and body are required")
	}

	c, err := s.threadService.CreateCanned(ctx, tenantID, req.Name, req.Body)
	if err != nil {
		return nil, mapInboxError(err)
	}
	return toCannedResponseInfo(c), nil
}

func (s *InboxGRPCServer) ListCannedResponses(ctx context.Context, _ *inboxv1.ListCannedResponsesRequest) (*inboxv1.ListCannedResponsesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	items, err := s.threadService.ListCanned(ctx, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	out := make([]*inboxv1.CannedResponseInfo, 0, len(items))
	for _, c := range items {
		out = append(out, toCannedResponseInfo(c))
	}
	return &inboxv1.ListCannedResponsesResponse{CannedResponses: out}, nil
}

func (s *InboxGRPCServer) UpdateCannedResponse(ctx context.Context, req *inboxv1.UpdateCannedResponseRequest) (*inboxv1.CannedResponseInfo, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	c, err := s.threadService.UpdateCanned(ctx, tenantID, id, req.Name, req.Body)
	if err != nil {
		return nil, mapInboxError(err)
	}
	return toCannedResponseInfo(c), nil
}

func (s *InboxGRPCServer) DeleteCannedResponse(ctx context.Context, req *inboxv1.DeleteCannedResponseRequest) (*inboxv1.DeleteCannedResponseResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.threadService.DeleteCanned(ctx, tenantID, id); err != nil {
		return nil, mapInboxError(err)
	}
	return &inboxv1.DeleteCannedResponseResponse{}, nil
}

func toThreadMessageInfo(m *thread.ThreadMessage) *inboxv1.ThreadMessageInfo {
	info := &inboxv1.ThreadMessageInfo{
		Id:         m.ID.String(),
		MessageId:  m.MessageID.String(),
		AuthorName: m.AuthorName,
		Direction:  m.Direction,
		Body:       m.Body,
		CreatedAt:  timestamppb.New(m.CreatedAt),
	}
	if m.AuthorID != nil {
		aid := m.AuthorID.String()
		info.AuthorId = &aid
	}
	return info
}

func toCannedResponseInfo(c *thread.CannedResponse) *inboxv1.CannedResponseInfo {
	return &inboxv1.CannedResponseInfo{
		Id:        c.ID.String(),
		Name:      c.Name,
		Body:      c.Body,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func (s *InboxGRPCServer) AssignMessage(ctx context.Context, req *inboxv1.AssignMessageRequest) (*inboxv1.AssignMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	assigneeID, err := uuid.Parse(req.AssigneeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
	}

	if err := s.messageService.AssignMessage(ctx, msgID, assigneeID); err != nil {
		return nil, mapInboxError(err)
	}

	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.AssignMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

func (s *InboxGRPCServer) GetUnreadCount(ctx context.Context, req *inboxv1.GetUnreadCountRequest) (*inboxv1.GetUnreadCountResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	counts, err := s.messageService.GetUnreadCounts(ctx, userID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	var total int32
	var byChannel []*inboxv1.UnreadCountByChannel
	for _, c := range counts {
		total += int32(c.Count)
		byChannel = append(byChannel, &inboxv1.UnreadCountByChannel{
			Channel: stringToChannel(c.Channel),
			Count:   int32(c.Count),
		})
	}

	return &inboxv1.GetUnreadCountResponse{
		Total:     total,
		ByChannel: byChannel,
	}, nil
}

func (s *InboxGRPCServer) BulkMarkRead(ctx context.Context, req *inboxv1.BulkMarkReadRequest) (*inboxv1.BulkMarkReadResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.MessageIds))
	for _, raw := range req.MessageIds {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid message_id: "+raw)
		}
		ids = append(ids, id)
	}

	if err := s.messageService.BulkMarkRead(ctx, ids); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.BulkMarkReadResponse{
		UpdatedCount: int32(len(ids)),
	}, nil
}

func (s *InboxGRPCServer) BulkArchive(ctx context.Context, req *inboxv1.BulkArchiveRequest) (*inboxv1.BulkArchiveResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.MessageIds))
	for _, raw := range req.MessageIds {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid message_id: "+raw)
		}
		ids = append(ids, id)
	}

	if err := s.messageService.BulkArchive(ctx, ids); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.BulkArchiveResponse{
		UpdatedCount: int32(len(ids)),
	}, nil
}

// ============================================================================
// Team Inboxes (8 RPCs)
// ============================================================================

func (s *InboxGRPCServer) CreateTeamInbox(ctx context.Context, req *inboxv1.CreateTeamInboxRequest) (*inboxv1.CreateTeamInboxResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	inbox := &models.TeamInbox{
		TenantID:       tenantID,
		Name:           req.Name,
		AssignmentMode: assignmentModeToString(req.AssignmentMode),
		Visibility:     visibilityToString(req.Visibility),
		CreatedBy:      userID,
	}
	if req.Description != "" {
		desc := req.Description
		inbox.Description = &desc
	}

	if err := s.teamService.CreateTeamInbox(ctx, inbox); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.CreateTeamInboxResponse{
		TeamInbox: toTeamInboxInfo(inbox),
	}, nil
}

func (s *InboxGRPCServer) UpdateTeamInbox(ctx context.Context, req *inboxv1.UpdateTeamInboxRequest) (*inboxv1.UpdateTeamInboxResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	teamInboxID, err := uuid.Parse(req.TeamInboxId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Get existing inbox to apply partial updates
	existing, err := s.teamService.GetTeamInbox(ctx, tenantID, teamInboxID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		desc := *req.Description
		existing.Description = &desc
	}
	if req.AssignmentMode != nil {
		existing.AssignmentMode = assignmentModeToString(*req.AssignmentMode)
	}
	if req.Visibility != nil {
		existing.Visibility = visibilityToString(*req.Visibility)
	}

	if err := s.teamService.UpdateTeamInbox(ctx, existing, userID); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.UpdateTeamInboxResponse{
		TeamInbox: toTeamInboxInfo(existing),
	}, nil
}

func (s *InboxGRPCServer) DeleteTeamInbox(ctx context.Context, req *inboxv1.DeleteTeamInboxRequest) (*inboxv1.DeleteTeamInboxResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	teamInboxID, err := uuid.Parse(req.TeamInboxId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.teamService.DeleteTeamInbox(ctx, tenantID, teamInboxID, userID); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.DeleteTeamInboxResponse{}, nil
}

func (s *InboxGRPCServer) ListTeamInboxes(ctx context.Context, req *inboxv1.ListTeamInboxesRequest) (*inboxv1.ListTeamInboxesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	inboxes, err := s.teamService.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	var infos []*inboxv1.TeamInboxInfo
	for _, ib := range inboxes {
		infos = append(infos, toTeamInboxInfo(ib))
	}

	return &inboxv1.ListTeamInboxesResponse{
		TeamInboxes: infos,
	}, nil
}

func (s *InboxGRPCServer) AddTeamMember(ctx context.Context, req *inboxv1.AddTeamMemberRequest) (*inboxv1.AddTeamMemberResponse, error) {
	teamInboxID, err := uuid.Parse(req.TeamInboxId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
	}

	memberUserID, err := uuid.Parse(req.MemberUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid member_user_id")
	}

	member := &models.TeamInboxMember{
		TeamInboxID: teamInboxID,
		UserID:      memberUserID,
		Role:        teamMemberRoleToString(req.Role),
	}

	if err := s.teamService.AddMember(ctx, member); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.AddTeamMemberResponse{
		Member: toTeamMemberInfo(member),
	}, nil
}

func (s *InboxGRPCServer) RemoveTeamMember(ctx context.Context, req *inboxv1.RemoveTeamMemberRequest) (*inboxv1.RemoveTeamMemberResponse, error) {
	teamInboxID, err := uuid.Parse(req.TeamInboxId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
	}

	memberUserID, err := uuid.Parse(req.MemberUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid member_user_id")
	}

	if err := s.teamService.RemoveMember(ctx, teamInboxID, memberUserID); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.RemoveTeamMemberResponse{}, nil
}

func (s *InboxGRPCServer) ListTeamMembers(ctx context.Context, req *inboxv1.ListTeamMembersRequest) (*inboxv1.ListTeamMembersResponse, error) {
	teamInboxID, err := uuid.Parse(req.TeamInboxId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_inbox_id")
	}

	members, err := s.teamService.ListMembers(ctx, teamInboxID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	var infos []*inboxv1.TeamMemberInfo
	for _, m := range members {
		infos = append(infos, toTeamMemberInfo(m))
	}

	return &inboxv1.ListTeamMembersResponse{
		Members: infos,
	}, nil
}

func (s *InboxGRPCServer) ClaimMessage(ctx context.Context, req *inboxv1.ClaimMessageRequest) (*inboxv1.ClaimMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Get the message to find its team_inbox_id
	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	if msg.TeamInboxID == nil {
		return nil, status.Error(codes.FailedPrecondition, "message is not in a team inbox")
	}

	if err := s.teamService.ClaimMessage(ctx, tenantID, *msg.TeamInboxID, msgID, userID); err != nil {
		return nil, mapInboxError(err)
	}

	// Refresh the message after claiming
	msg, err = s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.ClaimMessageResponse{
		Message: toInboxMessageInfo(msg),
	}, nil
}

// ============================================================================
// Routing Rules (5 RPCs)
// ============================================================================

func (s *InboxGRPCServer) CreateRoutingRule(ctx context.Context, req *inboxv1.CreateRoutingRuleRequest) (*inboxv1.CreateRoutingRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	conditionsJSON, err := structToJSON(req.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid conditions")
	}

	actionsJSON, err := structToJSON(req.Actions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid actions")
	}

	rule := &models.RoutingRule{
		TenantID:   tenantID,
		Name:       req.Name,
		Conditions: conditionsJSON,
		Actions:    actionsJSON,
		Priority:   int(req.Priority),
		IsActive:   true,
		CreatedBy:  userID,
	}

	if req.Channel != nil {
		ch := channelToString(*req.Channel)
		rule.Channel = &ch
	}

	if err := s.routingService.Create(ctx, rule); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.CreateRoutingRuleResponse{
		Rule: toRoutingRuleInfo(rule),
	}, nil
}

func (s *InboxGRPCServer) UpdateRoutingRule(ctx context.Context, req *inboxv1.UpdateRoutingRuleRequest) (*inboxv1.UpdateRoutingRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	ruleID, err := uuid.Parse(req.RuleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	existing, err := s.routingService.GetByID(ctx, tenantID, ruleID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Channel != nil {
		ch := channelToString(*req.Channel)
		existing.Channel = &ch
	}
	if req.Conditions != nil {
		conditionsJSON, jsonErr := structToJSON(req.Conditions)
		if jsonErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid conditions")
		}
		existing.Conditions = conditionsJSON
	}
	if req.Actions != nil {
		actionsJSON, jsonErr := structToJSON(req.Actions)
		if jsonErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid actions")
		}
		existing.Actions = actionsJSON
	}
	if req.Priority != nil {
		existing.Priority = int(*req.Priority)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := s.routingService.Update(ctx, existing); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.UpdateRoutingRuleResponse{
		Rule: toRoutingRuleInfo(existing),
	}, nil
}

func (s *InboxGRPCServer) DeleteRoutingRule(ctx context.Context, req *inboxv1.DeleteRoutingRuleRequest) (*inboxv1.DeleteRoutingRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	ruleID, err := uuid.Parse(req.RuleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	if err := s.routingService.Delete(ctx, tenantID, ruleID); err != nil {
		return nil, mapInboxError(err)
	}

	return &inboxv1.DeleteRoutingRuleResponse{}, nil
}

func (s *InboxGRPCServer) ListRoutingRules(ctx context.Context, req *inboxv1.ListRoutingRulesRequest) (*inboxv1.ListRoutingRulesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	rules, err := s.routingService.ListAll(ctx, tenantID)
	if err != nil {
		return nil, mapInboxError(err)
	}

	var infos []*inboxv1.RoutingRuleInfo
	for _, r := range rules {
		infos = append(infos, toRoutingRuleInfo(r))
	}

	return &inboxv1.ListRoutingRulesResponse{
		Rules: infos,
	}, nil
}

func (s *InboxGRPCServer) TestRoutingRule(ctx context.Context, req *inboxv1.TestRoutingRuleRequest) (*inboxv1.TestRoutingRuleResponse, error) {
	conditionsJSON, err := structToJSON(req.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid conditions")
	}

	var conditions models.Condition
	if err := json.Unmarshal(conditionsJSON, &conditions); err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse conditions")
	}

	testMsg := protoMessageToInboxMessage(req.TestMessage)

	matches := s.routingService.TestRule(ctx, conditions, testMsg)

	return &inboxv1.TestRoutingRuleResponse{
		Matches: matches,
	}, nil
}

// ============================================================================
// Converters
// ============================================================================

func toInboxMessageInfo(m *models.InboxMessage) *inboxv1.InboxMessageInfo {
	info := &inboxv1.InboxMessageInfo{
		Id:         m.ID.String(),
		UserId:     m.UserID.String(),
		Channel:    stringToChannel(m.Channel),
		SourceId:   m.SourceID,
		SenderName: m.SenderName,
		Subject:    m.Subject,
		Preview:    m.Preview,
		IsRead:     m.IsRead,
		IsStarred:  m.IsStarred,
		IsArchived: m.IsArchived,
		Tags:       m.Tags,
		DeepLink:   m.DeepLink,
		ReceivedAt: timestamppb.New(m.ReceivedAt),
		CreatedAt:  timestamppb.New(m.CreatedAt),
		UpdatedAt:  timestamppb.New(m.UpdatedAt),
	}

	if m.SenderID != nil {
		senderID := m.SenderID.String()
		info.SenderId = &senderID
	}
	if m.SenderEmail != nil {
		info.SenderEmail = m.SenderEmail
	}
	if m.SnoozedUntil != nil {
		info.SnoozedUntil = timestamppb.New(*m.SnoozedUntil)
	}
	if m.AssignedTo != nil {
		assignedTo := m.AssignedTo.String()
		info.AssignedTo = &assignedTo
	}
	if m.TeamInboxID != nil {
		teamInboxID := m.TeamInboxID.String()
		info.TeamInboxId = &teamInboxID
	}
	if m.CRMContactID != nil {
		crmContactID := m.CRMContactID.String()
		info.CrmContactId = &crmContactID
	}
	if m.Metadata != nil {
		if md, err := structpb.NewStruct(nil); err == nil {
			_ = json.Unmarshal(m.Metadata, &md)
			info.Metadata = md
		}
	}

	return info
}

func toTeamInboxInfo(t *models.TeamInbox) *inboxv1.TeamInboxInfo {
	info := &inboxv1.TeamInboxInfo{
		Id:             t.ID.String(),
		Name:           t.Name,
		AssignmentMode: stringToAssignmentMode(t.AssignmentMode),
		Visibility:     stringToVisibility(t.Visibility),
		CreatedBy:      t.CreatedBy.String(),
		CreatedAt:      timestamppb.New(t.CreatedAt),
		UpdatedAt:      timestamppb.New(t.UpdatedAt),
	}
	if t.Description != nil {
		info.Description = *t.Description
	}
	return info
}

func toTeamMemberInfo(m *models.TeamInboxMember) *inboxv1.TeamMemberInfo {
	return &inboxv1.TeamMemberInfo{
		TeamInboxId: m.TeamInboxID.String(),
		UserId:      m.UserID.String(),
		Role:        stringToTeamMemberRole(m.Role),
		CreatedAt:   timestamppb.New(m.CreatedAt),
	}
}

func toRoutingRuleInfo(r *models.RoutingRule) *inboxv1.RoutingRuleInfo {
	info := &inboxv1.RoutingRuleInfo{
		Id:        r.ID.String(),
		Name:      r.Name,
		Priority:  int32(r.Priority),
		IsActive:  r.IsActive,
		CreatedBy: r.CreatedBy.String(),
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}

	if r.Channel != nil {
		ch := stringToChannel(*r.Channel)
		info.Channel = &ch
	}

	// Convert conditions JSON to Struct
	if r.Conditions != nil {
		if st, err := jsonToStruct(r.Conditions); err == nil {
			info.Conditions = st
		}
	}

	// Convert actions JSON to Struct (wrap in a Struct with "actions" key)
	if r.Actions != nil {
		if st, err := jsonToStruct(r.Actions); err == nil {
			info.Actions = st
		}
	}

	return info
}

// protoMessageToInboxMessage converts a proto InboxMessageInfo to a domain InboxMessage for testing.
func protoMessageToInboxMessage(p *inboxv1.InboxMessageInfo) *models.InboxMessage {
	if p == nil {
		return &models.InboxMessage{}
	}

	msg := &models.InboxMessage{
		Channel:    channelToString(p.Channel),
		SourceID:   p.SourceId,
		SenderName: p.SenderName,
		Subject:    p.Subject,
		Preview:    p.Preview,
		IsRead:     p.IsRead,
		IsStarred:  p.IsStarred,
		IsArchived: p.IsArchived,
		Tags:       p.Tags,
		DeepLink:   p.DeepLink,
	}

	if id, err := uuid.Parse(p.Id); err == nil {
		msg.ID = id
	}
	if userID, err := uuid.Parse(p.UserId); err == nil {
		msg.UserID = userID
	}
	if p.SenderId != nil {
		if senderID, err := uuid.Parse(*p.SenderId); err == nil {
			msg.SenderID = &senderID
		}
	}
	if p.SenderEmail != nil {
		msg.SenderEmail = p.SenderEmail
	}
	if p.ReceivedAt != nil {
		msg.ReceivedAt = p.ReceivedAt.AsTime()
	}

	return msg
}

// ============================================================================
// Enum converters
// ============================================================================

func channelToString(c inboxv1.Channel) string {
	switch c {
	case inboxv1.Channel_CHANNEL_EMAIL:
		return "email"
	case inboxv1.Channel_CHANNEL_CHAT:
		return "chat"
	case inboxv1.Channel_CHANNEL_NOTIFICATION:
		return "notification"
	default:
		return ""
	}
}

func stringToChannel(s string) inboxv1.Channel {
	switch s {
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

func assignmentModeToString(m inboxv1.AssignmentMode) string {
	switch m {
	case inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL:
		return "manual"
	case inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN:
		return "round_robin"
	default:
		return "manual"
	}
}

func stringToAssignmentMode(s string) inboxv1.AssignmentMode {
	switch s {
	case "manual":
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL
	case "round_robin":
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN
	default:
		return inboxv1.AssignmentMode_ASSIGNMENT_MODE_UNSPECIFIED
	}
}

func visibilityToString(v inboxv1.TeamInboxVisibility) string {
	switch v {
	case inboxv1.TeamInboxVisibility_VISIBILITY_OPEN:
		return "open"
	case inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE:
		return "private"
	default:
		return "open"
	}
}

func stringToVisibility(s string) inboxv1.TeamInboxVisibility {
	switch s {
	case "open":
		return inboxv1.TeamInboxVisibility_VISIBILITY_OPEN
	case "private":
		return inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE
	default:
		return inboxv1.TeamInboxVisibility_VISIBILITY_UNSPECIFIED
	}
}

func teamMemberRoleToString(r inboxv1.TeamMemberRole) string {
	switch r {
	case inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN:
		return "admin"
	case inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER:
		return "member"
	default:
		return "member"
	}
}

func stringToTeamMemberRole(s string) inboxv1.TeamMemberRole {
	switch s {
	case "admin":
		return inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN
	case "member":
		return inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER
	default:
		return inboxv1.TeamMemberRole_TEAM_ROLE_UNSPECIFIED
	}
}

// ============================================================================
// Helpers
// ============================================================================

func structToJSON(s *structpb.Struct) (json.RawMessage, error) {
	if s == nil {
		return json.RawMessage("{}"), nil
	}
	data, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func jsonToStruct(data json.RawMessage) (*structpb.Struct, error) {
	if data == nil {
		return nil, nil
	}
	s := &structpb.Struct{}
	if err := s.UnmarshalJSON(data); err != nil {
		// If it fails as a Struct (e.g., it's an array), wrap it
		wrapper := map[string]any{}
		if wrapErr := json.Unmarshal(data, &wrapper); wrapErr == nil {
			return structpb.NewStruct(wrapper)
		}
		return nil, err
	}
	return s, nil
}

func parsePageToken(token string) (time.Time, uuid.UUID, error) {
	// Format: RFC3339Nano|UUID
	var t time.Time
	var id uuid.UUID

	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '|' {
			var err error
			t, err = time.Parse(time.RFC3339Nano, token[:i])
			if err != nil {
				return t, id, err
			}
			id, err = uuid.Parse(token[i+1:])
			return t, id, err
		}
	}

	return t, id, errors.New("invalid page token format")
}

func buildPageToken(t time.Time, id uuid.UUID) string {
	return t.Format(time.RFC3339Nano) + "|" + id.String()
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapInboxError(err error) error {
	switch {
	case errors.Is(err, message.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrAlreadyAssigned):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, message.ErrInvalidSnoozeTime):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, message.ErrAdapterNotFound):
		return status.Error(codes.Unimplemented, err.Error())
	case errors.Is(err, message.ErrDuplicateMessage):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, thread.ErrCannedResponseNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, team.ErrTeamInboxNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, team.ErrNotTeamAdmin):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, team.ErrNotTeamMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, team.ErrAlreadyMember):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, team.ErrCannotRemoveLastAdmin):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, team.ErrManualClaimInRoundRobin):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, routing.ErrRuleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, routing.ErrInvalidConditions):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, routing.ErrInvalidAction):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled inbox service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
