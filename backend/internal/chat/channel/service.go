package channel

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles channel business logic
type Service struct {
	repo Repository
}

// NewService creates a new channel service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a channel
type CreateInput struct {
	Name             string
	Description      *string
	IsPrivate        bool
	CreatedBy        uuid.UUID
	InitialMemberIDs []uuid.UUID
}

// Create creates a new channel
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.ChannelWithRelations, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrChannelNameRequired
	}

	// Validate initial members exist
	for _, memberID := range input.InitialMemberIDs {
		exists, err := s.repo.UserExists(ctx, memberID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrUserNotFound
		}
	}

	now := time.Now()
	channel := &models.Channel{
		ID:          uuid.New(),
		Name:        name,
		Description: input.Description,
		IsPrivate:   input.IsPrivate,
		IsDM:        false,
		IsArchived:  false,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, channel); err != nil {
		return nil, err
	}

	// Add creator as owner
	ownerMembership := &models.ChannelMembership{
		ChannelID: channel.ID,
		UserID:    input.CreatedBy,
		Role:      models.ChannelRoleOwner,
		JoinedAt:  now,
	}
	if err := s.repo.AddMember(ctx, ownerMembership); err != nil {
		return nil, err
	}

	// Add initial members
	for _, memberID := range input.InitialMemberIDs {
		if memberID == input.CreatedBy {
			continue // Skip creator, already added as owner
		}
		membership := &models.ChannelMembership{
			ChannelID: channel.ID,
			UserID:    memberID,
			Role:      models.ChannelRoleMember,
			JoinedAt:  now,
		}
		if err := s.repo.AddMember(ctx, membership); err != nil {
			return nil, err
		}
	}

	slog.Info("channel created",
		"channel_id", channel.ID,
		"name", channel.Name,
		"is_private", channel.IsPrivate,
		"created_by", channel.CreatedBy,
	)

	return s.getWithRelations(ctx, channel, input.CreatedBy)
}

// GetByID retrieves a channel by ID
func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.ChannelWithRelations, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user is a member (for private channels)
	if channel.IsPrivate {
		_, membershipErr := s.repo.GetMembership(ctx, id, userID)
		if membershipErr != nil {
			return nil, ErrNotChannelMember
		}
	}

	return s.getWithRelations(ctx, channel, userID)
}

// ListInput contains filtering options for listing channels
type ListInput struct {
	UserID          uuid.UUID
	IncludeArchived bool
	IsDM            *bool
	Search          string
	Page            int
	PageSize        int
}

// List retrieves channels the user is a member of
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.ChannelWithRelations, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		UserID:          input.UserID,
		IncludeArchived: input.IncludeArchived,
		IsDM:            input.IsDM,
		Search:          input.Search,
	}

	channels, total, err := s.repo.List(ctx, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	var results []*models.ChannelWithRelations
	for _, c := range channels {
		withRel, relErr := s.getWithRelations(ctx, c, input.UserID)
		if relErr != nil {
			return nil, 0, relErr
		}
		results = append(results, withRel)
	}

	return results, total, nil
}

// UpdateInput contains the data that can be updated on a channel
type UpdateInput struct {
	Name        *string
	Description *string
}

// Update updates an existing channel
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, input UpdateInput) (*models.ChannelWithRelations, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if channel.IsArchived {
		return nil, ErrChannelArchived
	}

	// Check authorization (must be owner or admin)
	membership, membershipErr := s.repo.GetMembership(ctx, id, userID)
	if membershipErr != nil {
		return nil, ErrNotChannelMember
	}
	if !membership.Role.CanModerate() {
		return nil, ErrNotAuthorized
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrChannelNameRequired
		}
		channel.Name = name
	}

	if input.Description != nil {
		channel.Description = input.Description
	}

	channel.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, channel); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("channel updated", "channel_id", channel.ID, "updated_by", userID)

	return s.getWithRelations(ctx, channel, userID)
}

// Delete removes a channel
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// DM channels cannot be deleted
	if channel.IsDM {
		return ErrCannotDeleteDM
	}

	// Check authorization (must be owner)
	membership, membershipErr := s.repo.GetMembership(ctx, id, userID)
	if membershipErr != nil {
		return ErrNotChannelMember
	}
	if membership.Role != models.ChannelRoleOwner {
		return ErrNotAuthorized
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("channel deleted",
		"channel_id", channel.ID,
		"name", channel.Name,
		"deleted_by", userID,
	)

	return nil
}

// Archive archives a channel
func (s *Service) Archive(ctx context.Context, id, userID uuid.UUID) (*models.ChannelWithRelations, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check authorization (must be owner or admin)
	membership, membershipErr := s.repo.GetMembership(ctx, id, userID)
	if membershipErr != nil {
		return nil, ErrNotChannelMember
	}
	if !membership.Role.CanModerate() {
		return nil, ErrNotAuthorized
	}

	// Idempotent
	if channel.IsArchived {
		return s.getWithRelations(ctx, channel, userID)
	}

	channel.IsArchived = true
	channel.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, channel); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("channel archived", "channel_id", channel.ID, "archived_by", userID)

	return s.getWithRelations(ctx, channel, userID)
}

// Join adds a user to a public channel
func (s *Service) Join(ctx context.Context, channelID, userID uuid.UUID) (*models.ChannelMember, error) {
	channel, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if channel.IsDM {
		return nil, ErrCannotJoinDM
	}

	if channel.IsArchived {
		return nil, ErrChannelArchived
	}

	if channel.IsPrivate {
		return nil, ErrCannotJoinPrivate
	}

	// Check if already a member
	existing, _ := s.repo.GetMembership(ctx, channelID, userID)
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	membership := &models.ChannelMembership{
		ChannelID: channelID,
		UserID:    userID,
		Role:      models.ChannelRoleMember,
		JoinedAt:  time.Now(),
	}

	if addErr := s.repo.AddMember(ctx, membership); addErr != nil {
		return nil, addErr
	}

	slog.Info("user joined channel", "channel_id", channelID, "user_id", userID)

	return s.getMemberWithInfo(ctx, membership)
}

// Leave removes a user from a channel
func (s *Service) Leave(ctx context.Context, channelID, userID uuid.UUID) error {
	channel, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}

	if channel.IsDM {
		return ErrCannotLeaveDM
	}

	membership, membershipErr := s.repo.GetMembership(ctx, channelID, userID)
	if membershipErr != nil {
		return ErrNotChannelMember
	}

	// Owner cannot leave
	if membership.Role == models.ChannelRoleOwner {
		return ErrCannotLeaveOwner
	}

	if removeErr := s.repo.RemoveMember(ctx, channelID, userID); removeErr != nil {
		return removeErr
	}

	slog.Info("user left channel", "channel_id", channelID, "user_id", userID)

	return nil
}

// GetMembers lists members of a channel
func (s *Service) GetMembers(ctx context.Context, channelID uuid.UUID, page, pageSize int) ([]*models.ChannelMember, int, error) {
	_, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.repo.ListMembers(ctx, channelID, offset, pageSize)
}

// UpdateMemberRole updates a member's role in the channel
func (s *Service) UpdateMemberRole(ctx context.Context, channelID, targetUserID, requesterUserID uuid.UUID, newRole models.ChannelRole) (*models.ChannelMember, error) {
	_, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// Validate role
	if !newRole.IsValid() {
		return nil, ErrInvalidRole
	}

	// Cannot change to owner
	if newRole == models.ChannelRoleOwner {
		return nil, ErrCannotChangeOwner
	}

	// Check requester authorization
	requesterMembership, reqErr := s.repo.GetMembership(ctx, channelID, requesterUserID)
	if reqErr != nil {
		return nil, ErrNotChannelMember
	}
	if requesterMembership.Role != models.ChannelRoleOwner {
		return nil, ErrNotAuthorized
	}

	// Get target membership
	targetMembership, targetErr := s.repo.GetMembership(ctx, channelID, targetUserID)
	if targetErr != nil {
		return nil, ErrNotChannelMember
	}

	// Cannot change owner role
	if targetMembership.Role == models.ChannelRoleOwner {
		return nil, ErrCannotChangeOwner
	}

	targetMembership.Role = newRole

	if updateErr := s.repo.UpdateMembership(ctx, targetMembership); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("member role updated",
		"channel_id", channelID,
		"user_id", targetUserID,
		"new_role", newRole,
		"updated_by", requesterUserID,
	)

	return s.getMemberWithInfo(ctx, targetMembership)
}

// AddMember adds a user to a channel (for private channels, invitations)
func (s *Service) AddMember(ctx context.Context, channelID, userID, inviterID uuid.UUID) (*models.ChannelMember, error) {
	channel, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if channel.IsArchived {
		return nil, ErrChannelArchived
	}

	// Check inviter authorization
	inviterMembership, inviterErr := s.repo.GetMembership(ctx, channelID, inviterID)
	if inviterErr != nil {
		return nil, ErrNotChannelMember
	}
	if !inviterMembership.Role.CanModerate() {
		return nil, ErrNotAuthorized
	}

	// Check if user exists
	exists, userErr := s.repo.UserExists(ctx, userID)
	if userErr != nil {
		return nil, userErr
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Check if already a member
	existing, _ := s.repo.GetMembership(ctx, channelID, userID)
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	membership := &models.ChannelMembership{
		ChannelID: channelID,
		UserID:    userID,
		Role:      models.ChannelRoleMember,
		JoinedAt:  time.Now(),
	}

	if addErr := s.repo.AddMember(ctx, membership); addErr != nil {
		return nil, addErr
	}

	slog.Info("member added to channel",
		"channel_id", channelID,
		"user_id", userID,
		"invited_by", inviterID,
	)

	return s.getMemberWithInfo(ctx, membership)
}

// getWithRelations loads relations for a channel
func (s *Service) getWithRelations(ctx context.Context, channel *models.Channel, userID uuid.UUID) (*models.ChannelWithRelations, error) {
	result := &models.ChannelWithRelations{
		Channel: *channel,
	}

	// Get member count
	count, err := s.repo.GetMemberCount(ctx, channel.ID)
	if err != nil {
		return nil, err
	}
	result.MemberCount = count

	// Get user's role
	membership, _ := s.repo.GetMembership(ctx, channel.ID, userID)
	if membership != nil {
		result.MyRole = &membership.Role
	}

	// Get last message
	lastMsg, _ := s.repo.GetLastMessage(ctx, channel.ID)
	if lastMsg != nil {
		result.LastMessage = &lastMsg.Message
	}

	return result, nil
}

// getMemberWithInfo loads user info for a membership
func (s *Service) getMemberWithInfo(ctx context.Context, membership *models.ChannelMembership) (*models.ChannelMember, error) {
	firstName, lastName, email, err := s.repo.GetUserInfo(ctx, membership.UserID)
	if err != nil {
		return nil, err
	}

	return &models.ChannelMember{
		ChannelMembership: *membership,
		FirstName:         firstName,
		LastName:          lastName,
		Email:             email,
	}, nil
}

// GetOrCreateDMInput contains the data needed to get or create a DM
type GetOrCreateDMInput struct {
	UserID      uuid.UUID
	OtherUserID uuid.UUID
}

// GetOrCreateDMResult contains the result of GetOrCreateDM
type GetOrCreateDMResult struct {
	Channel *models.ChannelWithRelations
	Created bool
}

// GetOrCreateDM finds an existing DM between two users or creates a new one
func (s *Service) GetOrCreateDM(ctx context.Context, input GetOrCreateDMInput) (*GetOrCreateDMResult, error) {
	// Cannot DM yourself
	if input.UserID == input.OtherUserID {
		return nil, ErrCannotDMSelf
	}

	// Validate other user exists
	exists, err := s.repo.UserExists(ctx, input.OtherUserID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Sort user IDs canonically (user1 < user2 for DB constraint)
	user1, user2 := input.UserID, input.OtherUserID
	if user1.String() > user2.String() {
		user1, user2 = user2, user1
	}

	// Try to find existing DM
	existing, findErr := s.repo.FindDMChannel(ctx, user1, user2)
	if findErr == nil && existing != nil {
		withRel, relErr := s.getWithRelations(ctx, existing, input.UserID)
		if relErr != nil {
			return nil, relErr
		}
		return &GetOrCreateDMResult{Channel: withRel, Created: false}, nil
	}
	if findErr != nil && findErr != ErrChannelNotFound {
		return nil, findErr
	}

	// Create new DM channel
	now := time.Now()
	channel := &models.Channel{
		ID:         uuid.New(),
		Name:       "DM",
		IsPrivate:  true,
		IsDM:       true,
		IsArchived: false,
		CreatedBy:  input.UserID,
		DMUser1:    &user1,
		DMUser2:    &user2,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	mem1 := &models.ChannelMembership{
		ChannelID: channel.ID,
		UserID:    user1,
		Role:      models.ChannelRoleMember,
		JoinedAt:  now,
	}
	mem2 := &models.ChannelMembership{
		ChannelID: channel.ID,
		UserID:    user2,
		Role:      models.ChannelRoleMember,
		JoinedAt:  now,
	}

	if createErr := s.repo.CreateDMChannel(ctx, channel, mem1, mem2); createErr != nil {
		return nil, createErr
	}

	slog.Info("dm channel created",
		"channel_id", channel.ID,
		"user1", user1,
		"user2", user2,
	)

	withRel, relErr := s.getWithRelations(ctx, channel, input.UserID)
	if relErr != nil {
		return nil, relErr
	}
	return &GetOrCreateDMResult{Channel: withRel, Created: true}, nil
}

// ListDMs lists DM channels for a user
func (s *Service) ListDMs(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*models.ChannelWithRelations, int, error) {
	isDM := true
	return s.List(ctx, ListInput{
		UserID:   userID,
		IsDM:     &isDM,
		Page:     page,
		PageSize: pageSize,
	})
}
