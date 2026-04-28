package message

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// LangDetector detects the language of a text for search indexing
type LangDetector interface {
	Detect(text string) string
}

// EventEmitter is an optional interface for emitting notification events.
// If set, the message service will emit events for mentions and DM messages.
type EventEmitter interface {
	EmitChatEvent(ctx context.Context, payload models.EventPayload) error
}

// Service handles message business logic
type Service struct {
	repo         Repository
	langDetector LangDetector
	eventEmitter EventEmitter
}

// NewService creates a new message service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.eventEmitter = emitter
}

// SetLangDetector sets the language detector (optional, for search indexing)
func (s *Service) SetLangDetector(detector LangDetector) {
	s.langDetector = detector
}

// CreateInput contains the data needed to create a message
type CreateInput struct {
	TenantID         uuid.UUID
	ChannelID        uuid.UUID
	Content          string
	CreatedBy        uuid.UUID
	GuestSessionID   *uuid.UUID  // Set for guest messages, mutually exclusive with CreatedBy
	ParentMessageID  *uuid.UUID  // For thread replies
	MentionedUserIDs []uuid.UUID // User IDs to mention (Sprint 3)
	MentionEveryone  bool        // @everyone/@channel mention (Sprint 3)
}

// Create creates a new message
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.MessageWithSender, error) {
	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrContentRequired
	}

	// Check channel exists within tenant
	exists, err := s.repo.ChannelExists(ctx, input.ChannelID, input.TenantID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrChannelNotFound
	}

	// Check channel not archived (also tenant-scoped)
	archived, err := s.repo.IsChannelArchived(ctx, input.ChannelID, input.TenantID)
	if err != nil {
		return nil, err
	}
	if archived {
		return nil, ErrChannelArchived
	}

	// Guest messages bypass membership check; regular messages require it
	if input.GuestSessionID == nil {
		isMember, memberErr := s.repo.IsMember(ctx, input.ChannelID, input.TenantID, input.CreatedBy)
		if memberErr != nil {
			return nil, memberErr
		}
		if !isMember {
			return nil, ErrNotChannelMember
		}
	}

	// Validate thread reply if parent is specified
	if input.ParentMessageID != nil {
		parent, parentErr := s.repo.GetByID(ctx, *input.ParentMessageID, input.TenantID)
		if parentErr != nil {
			return nil, parentErr
		}
		if parent.ChannelID != input.ChannelID {
			return nil, ErrParentNotInChannel
		}
		if parent.ParentMessageID != nil {
			return nil, ErrNestedThreads
		}
		if parent.IsDeleted {
			return nil, ErrParentDeleted
		}
	}

	// Detect language for search indexing
	lang := "german" // default
	if s.langDetector != nil {
		lang = s.langDetector.Detect(content)
	}

	now := time.Now()
	message := &models.Message{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		ChannelID:       input.ChannelID,
		Content:         content,
		IsDeleted:       false,
		EditedAt:        nil,
		ParentMessageID: input.ParentMessageID,
		Lang:            lang,
		GuestSessionID:  input.GuestSessionID,
		CreatedAt:       now,
	}
	if input.GuestSessionID == nil {
		message.CreatedBy = &input.CreatedBy
	}

	// Use atomic create+increment for thread replies
	if input.ParentMessageID != nil {
		if createErr := s.repo.CreateWithReplyCount(ctx, message); createErr != nil {
			return nil, createErr
		}
	} else {
		if createErr := s.repo.Create(ctx, message); createErr != nil {
			return nil, createErr
		}
	}

	slog.Info("message created",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"created_by", message.CreatedBy,
		"parent_message_id", input.ParentMessageID,
	)

	// Create mentions (Sprint 3)
	mentions, mentionErr := s.processMentions(ctx, message.ID, input.ChannelID, input.TenantID, input.MentionedUserIDs, input.MentionEveryone, now)
	if mentionErr != nil {
		return nil, mentionErr
	}

	result, senderErr := s.getWithSender(ctx, message)
	if senderErr != nil {
		return nil, senderErr
	}
	result.Mentions = mentions

	// Emit notification events (best-effort, don't fail the message creation)
	if s.eventEmitter != nil {
		s.emitMessageEvents(ctx, message, result, input, mentions)
	}

	return result, nil
}

// emitMessageEvents emits notification events for mentions and DM messages.
func (s *Service) emitMessageEvents(ctx context.Context, message *models.Message, result *models.MessageWithSender, input CreateInput, mentions []models.MentionWithUser) {
	if message.CreatedBy == nil {
		return // Guest messages have separate event handling
	}

	senderName := result.SenderFirstName + " " + result.SenderLastName

	// Truncate content for preview
	contentPreview := message.Content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100]
	}

	// Emit mention events
	if len(mentions) > 0 {
		var targetUserIDs []string
		for _, m := range mentions {
			// Don't notify the sender about their own mention
			if m.UserID != *message.CreatedBy {
				targetUserIDs = append(targetUserIDs, m.UserID.String())
			}
		}

		if len(targetUserIDs) > 0 {
			channelName, _ := s.repo.GetChannelName(ctx, message.ChannelID)

			_ = s.eventEmitter.EmitChatEvent(ctx, models.EventPayload{
				Type:          "chat.mention",
				Priority:      models.PriorityNormal,
				ActorID:       message.CreatedBy.String(),
				ResourceID:    message.ChannelID.String(),
				ModuleID:      "chat",
				Title:         senderName + " mentioned you in #" + channelName,
				Body:          contentPreview,
				DeepLink:      "/chat/channels/" + message.ChannelID.String() + "/messages/" + message.ID.String(),
				GroupKey:      "chat-mention:" + message.ChannelID.String(),
				TargetUserIDs: targetUserIDs,
				Timestamp:     time.Now(),
			})
		}
	}

	// Emit DM event
	recipientID, err := s.repo.GetDMRecipient(ctx, message.ChannelID, *message.CreatedBy)
	if err == nil && recipientID != nil {
		_ = s.eventEmitter.EmitChatEvent(ctx, models.EventPayload{
			Type:          "chat.dm.new",
			Priority:      models.PriorityNormal,
			ActorID:       message.CreatedBy.String(),
			ResourceID:    message.ChannelID.String(),
			ModuleID:      "chat",
			Title:         senderName + " sent you a message",
			Body:          contentPreview,
			DeepLink:      "/chat/dm/" + message.ChannelID.String(),
			GroupKey:      "chat-dm:" + message.ChannelID.String(),
			TargetUserIDs: []string{recipientID.String()},
			Timestamp:     time.Now(),
		})
	}
}

// GetByID retrieves a message by ID
func (s *Service) GetByID(ctx context.Context, id, tenantID, userID uuid.UUID) (*models.MessageWithSender, error) {
	message, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Check user is a member of the channel (tenant-scoped)
	isMember, memberErr := s.repo.IsMember(ctx, message.ChannelID, tenantID, userID)
	if memberErr != nil {
		return nil, memberErr
	}
	if !isMember {
		return nil, ErrNotChannelMember
	}

	return s.getWithSender(ctx, message)
}

// ListInput contains options for listing messages
type ListInput struct {
	TenantID            uuid.UUID
	ChannelID           uuid.UUID
	UserID              uuid.UUID // For membership check
	Limit               int
	Before              *uuid.UUID
	After               *uuid.UUID
	SkipMembershipCheck bool // Set for guest-enabled channels where guests bypass membership
}

// List retrieves messages from a channel
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.MessageWithSender, bool, error) {
	// Check channel exists within tenant
	exists, err := s.repo.ChannelExists(ctx, input.ChannelID, input.TenantID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, ErrChannelNotFound
	}

	// Check user is a member (skip for guest-enabled channels)
	if !input.SkipMembershipCheck {
		isMember, memberErr := s.repo.IsMember(ctx, input.ChannelID, input.TenantID, input.UserID)
		if memberErr != nil {
			return nil, false, memberErr
		}
		if !isMember {
			return nil, false, ErrNotChannelMember
		}
	}

	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 50
	}

	filter := ListFilter{
		TenantID:       input.TenantID,
		ChannelID:      input.ChannelID,
		Limit:          input.Limit + 1, // Fetch one extra to determine if there are more
		Before:         input.Before,
		After:          input.After,
		ExcludeReplies: true, // Thread replies only via GetThreadReplies
	}

	messages, listErr := s.repo.List(ctx, filter)
	if listErr != nil {
		return nil, false, listErr
	}

	// Check if there are more messages
	hasMore := len(messages) > input.Limit
	if hasMore {
		messages = messages[:input.Limit] // Remove the extra
	}

	// Batch-load mentions
	if loadErr := s.loadMentionsForMessages(ctx, messages); loadErr != nil {
		return nil, false, loadErr
	}

	// Batch-load files
	if loadErr := s.loadFilesForMessages(ctx, messages); loadErr != nil {
		return nil, false, loadErr
	}

	return messages, hasMore, nil
}

// UpdateInput contains the data for updating a message
type UpdateInput struct {
	Content string
}

// Update updates an existing message
func (s *Service) Update(ctx context.Context, id, tenantID, userID uuid.UUID, input UpdateInput) (*models.MessageWithSender, error) {
	message, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if message.IsDeleted {
		return nil, ErrMessageDeleted
	}

	// Only author can edit
	if message.CreatedBy == nil || *message.CreatedBy != userID {
		return nil, ErrNotMessageAuthor
	}

	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrContentRequired
	}

	now := time.Now()
	message.Content = content
	message.EditedAt = &now

	if updateErr := s.repo.Update(ctx, message); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("message updated", "message_id", message.ID, "tenant_id", tenantID, "updated_by", userID)

	return s.getWithSender(ctx, message)
}

// Delete soft-deletes a message
func (s *Service) Delete(ctx context.Context, id, tenantID, userID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if message.IsDeleted {
		return nil // Already deleted, idempotent
	}

	// Check authorization: author or channel admin/owner
	if message.CreatedBy == nil || *message.CreatedBy != userID {
		role, roleErr := s.repo.GetMemberRole(ctx, message.ChannelID, userID)
		if roleErr != nil {
			return ErrNotChannelMember
		}
		if !role.CanModerate() {
			return ErrNotAuthorized
		}
	}

	if deleteErr := s.repo.Delete(ctx, id, tenantID); deleteErr != nil {
		return deleteErr
	}

	// Decrement parent reply_count if this was a thread reply
	if message.ParentMessageID != nil {
		if decErr := s.repo.DecrementReplyCount(ctx, *message.ParentMessageID); decErr != nil {
			slog.Error("failed to decrement reply count",
				"parent_message_id", message.ParentMessageID,
				"error", decErr,
			)
		}
	}

	slog.Info("message deleted",
		"message_id", message.ID,
		"tenant_id", tenantID,
		"channel_id", message.ChannelID,
		"deleted_by", userID,
	)

	return nil
}

// GetThreadRepliesInput contains options for listing thread replies
type GetThreadRepliesInput struct {
	TenantID        uuid.UUID
	ParentMessageID uuid.UUID
	UserID          uuid.UUID
	Limit           int
	Before          *uuid.UUID
	After           *uuid.UUID
}

// GetThreadRepliesResult contains the parent and its replies
type GetThreadRepliesResult struct {
	Parent  *models.MessageWithSender
	Replies []*models.MessageWithSender
	HasMore bool
}

// GetThreadReplies returns a parent message and its thread replies
func (s *Service) GetThreadReplies(ctx context.Context, input GetThreadRepliesInput) (*GetThreadRepliesResult, error) {
	// Get parent message (tenant-scoped)
	parent, err := s.repo.GetByID(ctx, input.ParentMessageID, input.TenantID)
	if err != nil {
		return nil, err
	}

	// Validate parent is not a reply itself (no nested threads)
	if parent.ParentMessageID != nil {
		return nil, ErrNestedThreads
	}

	// Check user is member of the channel (tenant-scoped)
	isMember, memberErr := s.repo.IsMember(ctx, parent.ChannelID, input.TenantID, input.UserID)
	if memberErr != nil {
		return nil, memberErr
	}
	if !isMember {
		return nil, ErrNotChannelMember
	}

	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 50
	}

	filter := ThreadListFilter{
		ParentMessageID: input.ParentMessageID,
		Limit:           input.Limit + 1,
		Before:          input.Before,
		After:           input.After,
	}

	replies, listErr := s.repo.ListReplies(ctx, filter)
	if listErr != nil {
		return nil, listErr
	}

	hasMore := len(replies) > input.Limit
	if hasMore {
		replies = replies[:input.Limit]
	}

	parentWithSender, senderErr := s.getWithSender(ctx, parent)
	if senderErr != nil {
		return nil, senderErr
	}

	// Batch-load mentions for parent and replies
	allMessages := append([]*models.MessageWithSender{parentWithSender}, replies...)
	if loadErr := s.loadMentionsForMessages(ctx, allMessages); loadErr != nil {
		return nil, loadErr
	}

	// Batch-load files for parent and replies
	if loadErr := s.loadFilesForMessages(ctx, allMessages); loadErr != nil {
		return nil, loadErr
	}

	return &GetThreadRepliesResult{
		Parent:  parentWithSender,
		Replies: replies,
		HasMore: hasMore,
	}, nil
}

// getWithSender loads sender info for a message
func (s *Service) getWithSender(ctx context.Context, message *models.Message) (*models.MessageWithSender, error) {
	result := &models.MessageWithSender{
		Message: *message,
	}

	if message.CreatedBy != nil {
		firstName, lastName, err := s.repo.GetUserInfo(ctx, *message.CreatedBy)
		if err != nil {
			return nil, err
		}
		result.SenderFirstName = firstName
		result.SenderLastName = lastName
	} else if message.GuestSessionID != nil {
		displayName, err := s.repo.GetGuestDisplayName(ctx, *message.GuestSessionID)
		if err != nil {
			slog.Warn("failed to get guest display name", "session_id", message.GuestSessionID, "error", err)
			displayName = "Gast"
		}
		result.GuestDisplayName = displayName
	}

	return result, nil
}

// processMentions validates and creates mentions for a message
func (s *Service) processMentions(ctx context.Context, messageID, channelID, tenantID uuid.UUID, mentionedUserIDs []uuid.UUID, mentionEveryone bool, now time.Time) ([]models.MentionWithUser, error) {
	if len(mentionedUserIDs) == 0 && !mentionEveryone {
		return nil, nil
	}

	var mentions []models.Mention

	if mentionEveryone {
		// Expand @everyone to all channel members
		memberIDs, err := s.repo.GetChannelMemberIDs(ctx, channelID)
		if err != nil {
			return nil, err
		}
		mentionType := models.MentionTypeEveryone
		for _, memberID := range memberIDs {
			mentions = append(mentions, models.Mention{
				MessageID:   messageID,
				UserID:      memberID,
				MentionType: mentionType,
				CreatedAt:   now,
			})
		}
	} else {
		// Validate each mentioned user is a channel member (tenant-scoped)
		for _, userID := range mentionedUserIDs {
			isMember, err := s.repo.IsMember(ctx, channelID, tenantID, userID)
			if err != nil {
				return nil, err
			}
			if !isMember {
				return nil, ErrInvalidMention
			}
			mentions = append(mentions, models.Mention{
				MessageID:   messageID,
				UserID:      userID,
				MentionType: models.MentionTypeUser,
				CreatedAt:   now,
			})
		}
	}

	if len(mentions) == 0 {
		return nil, nil
	}

	if err := s.repo.CreateMentions(ctx, messageID, mentions); err != nil {
		return nil, err
	}

	// Return mentions with user info via batch load
	mentionMap, err := s.repo.GetMentionsByMessages(ctx, []uuid.UUID{messageID})
	if err != nil {
		return nil, err
	}
	return mentionMap[messageID], nil
}

// loadMentionsForMessages batch-loads mentions for a set of messages
func (s *Service) loadMentionsForMessages(ctx context.Context, messages []*models.MessageWithSender) error {
	if len(messages) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(messages))
	for i, m := range messages {
		ids[i] = m.ID
	}

	mentionMap, err := s.repo.GetMentionsByMessages(ctx, ids)
	if err != nil {
		return err
	}

	for _, m := range messages {
		if mentions, ok := mentionMap[m.ID]; ok {
			m.Mentions = mentions
		}
	}
	return nil
}

// loadFilesForMessages batch-loads files for a set of messages
func (s *Service) loadFilesForMessages(ctx context.Context, messages []*models.MessageWithSender) error {
	if len(messages) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(messages))
	for i, m := range messages {
		ids[i] = m.ID
	}

	fileMap, err := s.repo.GetFilesByMessageIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, m := range messages {
		if files, ok := fileMap[m.ID]; ok {
			m.Files = files
		}
	}
	return nil
}

// GetMentionsForUser returns paginated mentions for a user
func (s *Service) GetMentionsForUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]MentionDetailRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.GetMentionsForUser(ctx, userID, pageSize, offset)
}
