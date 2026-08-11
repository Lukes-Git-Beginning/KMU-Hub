package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/chat/channel"
	"github.com/kmuhub/kmuhub/internal/chat/message"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// ============================================================================
// stubChannelRepo -- implements channel.Repository (in-memory, no DB)
// ============================================================================

func chMembershipKey(channelID, userID uuid.UUID) string {
	return channelID.String() + "-" + userID.String()
}

type stubChannelRepo struct {
	channels    map[uuid.UUID]*models.Channel
	memberships map[string]*models.ChannelMembership
	users       map[uuid.UUID]struct{ firstName, lastName, email string }
}

func newStubChannelRepo() *stubChannelRepo {
	return &stubChannelRepo{
		channels:    make(map[uuid.UUID]*models.Channel),
		memberships: make(map[string]*models.ChannelMembership),
		users:       make(map[uuid.UUID]struct{ firstName, lastName, email string }),
	}
}

func (r *stubChannelRepo) Create(_ context.Context, ch *models.Channel) error {
	r.channels[ch.ID] = ch
	return nil
}

func (r *stubChannelRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Channel, error) {
	ch, ok := r.channels[id]
	if !ok {
		return nil, channel.ErrChannelNotFound
	}
	return ch, nil
}

func (r *stubChannelRepo) GetByIDForTenant(_ context.Context, id, _ uuid.UUID) (*models.Channel, error) {
	ch, ok := r.channels[id]
	if !ok {
		return nil, channel.ErrChannelNotFound
	}
	return ch, nil
}

func (r *stubChannelRepo) List(_ context.Context, filter channel.ListFilter, offset, limit int) ([]*models.Channel, int, error) {
	var result []*models.Channel
	for _, ch := range r.channels {
		if _, isMember := r.memberships[chMembershipKey(ch.ID, filter.UserID)]; !isMember {
			continue
		}
		if !filter.IncludeArchived && ch.IsArchived {
			continue
		}
		if filter.IsDM != nil && ch.IsDM != *filter.IsDM {
			continue
		}
		result = append(result, ch)
	}
	total := len(result)
	if offset >= len(result) {
		return []*models.Channel{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *stubChannelRepo) Update(_ context.Context, ch *models.Channel) error {
	r.channels[ch.ID] = ch
	return nil
}

func (r *stubChannelRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	delete(r.channels, id)
	return nil
}

func (r *stubChannelRepo) AddMember(_ context.Context, m *models.ChannelMembership) error {
	r.memberships[chMembershipKey(m.ChannelID, m.UserID)] = m
	return nil
}

func (r *stubChannelRepo) RemoveMember(_ context.Context, channelID, userID uuid.UUID) error {
	delete(r.memberships, chMembershipKey(channelID, userID))
	return nil
}

func (r *stubChannelRepo) GetMembership(_ context.Context, channelID, userID uuid.UUID) (*models.ChannelMembership, error) {
	m, ok := r.memberships[chMembershipKey(channelID, userID)]
	if !ok {
		return nil, channel.ErrNotChannelMember
	}
	return m, nil
}

func (r *stubChannelRepo) UpdateMembership(_ context.Context, m *models.ChannelMembership) error {
	r.memberships[chMembershipKey(m.ChannelID, m.UserID)] = m
	return nil
}

func (r *stubChannelRepo) ListMembers(_ context.Context, channelID uuid.UUID, offset, limit int) ([]*models.ChannelMember, int, error) {
	var result []*models.ChannelMember
	for _, m := range r.memberships {
		if m.ChannelID != channelID {
			continue
		}
		u := r.users[m.UserID]
		result = append(result, &models.ChannelMember{
			ChannelMembership: *m,
			FirstName:         u.firstName,
			LastName:          u.lastName,
			Email:             u.email,
		})
	}
	total := len(result)
	if offset >= len(result) {
		return []*models.ChannelMember{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *stubChannelRepo) GetMemberCount(_ context.Context, channelID uuid.UUID) (int, error) {
	count := 0
	for _, m := range r.memberships {
		if m.ChannelID == channelID {
			count++
		}
	}
	return count, nil
}

func (r *stubChannelRepo) GetUserInfo(_ context.Context, userID uuid.UUID) (string, string, string, error) {
	u, ok := r.users[userID]
	if !ok {
		return "", "", "", channel.ErrUserNotFound
	}
	return u.firstName, u.lastName, u.email, nil
}

func (r *stubChannelRepo) UserExists(_ context.Context, userID uuid.UUID) (bool, error) {
	_, ok := r.users[userID]
	return ok, nil
}

func (r *stubChannelRepo) GetLastMessage(_ context.Context, _ uuid.UUID) (*models.MessageWithSender, error) {
	return nil, nil
}

func (r *stubChannelRepo) FindDMChannel(_ context.Context, user1, user2, _ uuid.UUID) (*models.Channel, error) {
	for _, ch := range r.channels {
		if ch.IsDM && ch.DMUser1 != nil && ch.DMUser2 != nil && *ch.DMUser1 == user1 && *ch.DMUser2 == user2 {
			return ch, nil
		}
	}
	return nil, channel.ErrChannelNotFound
}

func (r *stubChannelRepo) CreateDMChannel(_ context.Context, ch *models.Channel, mem1, mem2 *models.ChannelMembership) error {
	r.channels[ch.ID] = ch
	r.memberships[chMembershipKey(mem1.ChannelID, mem1.UserID)] = mem1
	r.memberships[chMembershipKey(mem2.ChannelID, mem2.UserID)] = mem2
	return nil
}

func (r *stubChannelRepo) UpdateLastRead(_ context.Context, channelID, userID uuid.UUID, readAt time.Time) error {
	m, ok := r.memberships[chMembershipKey(channelID, userID)]
	if !ok {
		return channel.ErrNotChannelMember
	}
	m.LastReadAt = &readAt
	return nil
}

func (r *stubChannelRepo) GetUnreadCount(_ context.Context, _, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *stubChannelRepo) GetUnreadCountsForUser(_ context.Context, userID uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int)
	for _, ch := range r.channels {
		if _, ok := r.memberships[chMembershipKey(ch.ID, userID)]; ok {
			result[ch.ID] = 0
		}
	}
	return result, nil
}

func (r *stubChannelRepo) addUser(id uuid.UUID, firstName, lastName, email string) {
	r.users[id] = struct{ firstName, lastName, email string }{firstName, lastName, email}
}

// ============================================================================
// stubChatMessageRepo -- implements message.Repository (in-memory, no DB)
// ============================================================================

type stubChatMessageRepo struct {
	messages    map[uuid.UUID]*models.Message
	channels    map[uuid.UUID]struct{ archived bool }
	memberships map[string]models.ChannelRole
	users       map[uuid.UUID]struct{ firstName, lastName string }
	mentions    map[uuid.UUID][]models.Mention
}

func newStubChatMessageRepo() *stubChatMessageRepo {
	return &stubChatMessageRepo{
		messages:    make(map[uuid.UUID]*models.Message),
		channels:    make(map[uuid.UUID]struct{ archived bool }),
		memberships: make(map[string]models.ChannelRole),
		users:       make(map[uuid.UUID]struct{ firstName, lastName string }),
		mentions:    make(map[uuid.UUID][]models.Mention),
	}
}

func (r *stubChatMessageRepo) Create(_ context.Context, msg *models.Message) error {
	r.messages[msg.ID] = msg
	return nil
}

func (r *stubChatMessageRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*models.Message, error) {
	msg, ok := r.messages[id]
	if !ok {
		return nil, message.ErrMessageNotFound
	}
	return msg, nil
}

func (r *stubChatMessageRepo) List(_ context.Context, filter message.ListFilter) ([]*models.MessageWithSender, error) {
	var result []*models.MessageWithSender
	for _, msg := range r.messages {
		if msg.ChannelID != filter.ChannelID || msg.IsDeleted {
			continue
		}
		if filter.ExcludeReplies && msg.ParentMessageID != nil {
			continue
		}
		mws := &models.MessageWithSender{Message: *msg}
		if msg.CreatedBy != nil {
			u := r.users[*msg.CreatedBy]
			mws.SenderFirstName = u.firstName
			mws.SenderLastName = u.lastName
		}
		result = append(result, mws)
	}
	if len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (r *stubChatMessageRepo) Update(_ context.Context, msg *models.Message) error {
	r.messages[msg.ID] = msg
	return nil
}

func (r *stubChatMessageRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if msg, ok := r.messages[id]; ok {
		msg.IsDeleted = true
	}
	return nil
}

func (r *stubChatMessageRepo) ChannelExists(_ context.Context, channelID, _ uuid.UUID) (bool, error) {
	_, ok := r.channels[channelID]
	return ok, nil
}

func (r *stubChatMessageRepo) IsChannelArchived(_ context.Context, channelID, _ uuid.UUID) (bool, error) {
	ch, ok := r.channels[channelID]
	if !ok {
		return false, message.ErrChannelNotFound
	}
	return ch.archived, nil
}

func (r *stubChatMessageRepo) IsMember(_ context.Context, channelID, _ uuid.UUID, userID uuid.UUID) (bool, error) {
	_, ok := r.memberships[chMembershipKey(channelID, userID)]
	return ok, nil
}

func (r *stubChatMessageRepo) GetMemberRole(_ context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	role, ok := r.memberships[chMembershipKey(channelID, userID)]
	if !ok {
		return "", message.ErrNotChannelMember
	}
	return role, nil
}

func (r *stubChatMessageRepo) GetUserInfo(_ context.Context, userID uuid.UUID) (string, string, error) {
	u, ok := r.users[userID]
	if !ok {
		return "", "", nil
	}
	return u.firstName, u.lastName, nil
}

func (r *stubChatMessageRepo) CreateWithReplyCount(_ context.Context, msg *models.Message) error {
	r.messages[msg.ID] = msg
	if msg.ParentMessageID != nil {
		if parent, ok := r.messages[*msg.ParentMessageID]; ok {
			parent.ReplyCount++
		}
	}
	return nil
}

func (r *stubChatMessageRepo) ListReplies(_ context.Context, filter message.ThreadListFilter) ([]*models.MessageWithSender, error) {
	var result []*models.MessageWithSender
	for _, msg := range r.messages {
		if msg.ParentMessageID == nil || *msg.ParentMessageID != filter.ParentMessageID || msg.IsDeleted {
			continue
		}
		mws := &models.MessageWithSender{Message: *msg}
		if msg.CreatedBy != nil {
			u := r.users[*msg.CreatedBy]
			mws.SenderFirstName = u.firstName
			mws.SenderLastName = u.lastName
		}
		result = append(result, mws)
	}
	if len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (r *stubChatMessageRepo) DecrementReplyCount(_ context.Context, messageID uuid.UUID) error {
	if msg, ok := r.messages[messageID]; ok && msg.ReplyCount > 0 {
		msg.ReplyCount--
	}
	return nil
}

func (r *stubChatMessageRepo) CreateMentions(_ context.Context, messageID uuid.UUID, mentions []models.Mention) error {
	r.mentions[messageID] = mentions
	return nil
}

func (r *stubChatMessageRepo) GetMentionsByMessages(_ context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.MentionWithUser, error) {
	result := make(map[uuid.UUID][]models.MentionWithUser)
	for _, id := range messageIDs {
		for _, m := range r.mentions[id] {
			u := r.users[m.UserID]
			result[id] = append(result[id], models.MentionWithUser{Mention: m, FirstName: u.firstName, LastName: u.lastName})
		}
	}
	return result, nil
}

func (r *stubChatMessageRepo) GetMentionsForUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]message.MentionDetailRow, int, error) {
	var result []message.MentionDetailRow
	for msgID, mentions := range r.mentions {
		for _, m := range mentions {
			if m.UserID != userID {
				continue
			}
			msg := r.messages[msgID]
			if msg == nil || msg.IsDeleted {
				continue
			}
			var sender struct{ firstName, lastName string }
			if msg.CreatedBy != nil {
				sender = r.users[*msg.CreatedBy]
			}
			result = append(result, message.MentionDetailRow{
				MessageID:       msgID,
				ChannelID:       msg.ChannelID,
				Content:         msg.Content,
				MentionType:     m.MentionType,
				SenderFirstName: sender.firstName,
				SenderLastName:  sender.lastName,
				CreatedAt:       msg.CreatedAt.Format(time.RFC3339),
			})
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []message.MentionDetailRow{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *stubChatMessageRepo) GetChannelMemberIDs(_ context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for key, role := range r.memberships {
		_ = role
		if len(key) > 37 && key[:36] == channelID.String() {
			if uid, err := uuid.Parse(key[37:]); err == nil {
				ids = append(ids, uid)
			}
		}
	}
	return ids, nil
}

func (r *stubChatMessageRepo) GetFilesByMessageIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	return make(map[uuid.UUID][]models.ChatFileWithUploader), nil
}

func (r *stubChatMessageRepo) GetDMRecipient(_ context.Context, _, _ uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}

func (r *stubChatMessageRepo) GetChannelName(_ context.Context, _ uuid.UUID) (string, error) {
	return "test-channel", nil
}

func (r *stubChatMessageRepo) GetGuestDisplayName(_ context.Context, _ uuid.UUID) (string, error) {
	return "Guest", nil
}

func (r *stubChatMessageRepo) IsChannelGuestEnabled(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (r *stubChatMessageRepo) addChannel(id uuid.UUID, archived bool) {
	r.channels[id] = struct{ archived bool }{archived}
}

func (r *stubChatMessageRepo) addMember(channelID, userID uuid.UUID, role models.ChannelRole) {
	r.memberships[chMembershipKey(channelID, userID)] = role
}

func (r *stubChatMessageRepo) addUser(id uuid.UUID, firstName, lastName string) {
	r.users[id] = struct{ firstName, lastName string }{firstName, lastName}
}

// ============================================================================
// Fixture
// ============================================================================

// chatChannelsMessagesFixture wires real channel.Service/message.Service
// instances against independent in-memory repos, mirroring how NewChatGRPCServer
// is wired in cmd/server. fileService/searchService/reactionService/bookmarkService
// stay nil -- this unit's scope is channels and messages only.
type chatChannelsMessagesFixture struct {
	srv     *ChatGRPCServer
	chRepo  *stubChannelRepo
	msgRepo *stubChatMessageRepo
}

func newChatChannelsMessagesFixture() *chatChannelsMessagesFixture {
	chRepo := newStubChannelRepo()
	msgRepo := newStubChatMessageRepo()
	srv := NewChatGRPCServer(channel.NewService(chRepo), message.NewService(msgRepo), nil, nil, nil, nil)
	return &chatChannelsMessagesFixture{srv: srv, chRepo: chRepo, msgRepo: msgRepo}
}

// seedChannel creates a channel visible to both the channel and message
// service views (they hold independent repos, as in production) and adds
// ownerID as its owner member in both.
func (f *chatChannelsMessagesFixture) seedChannel(tenantID, ownerID uuid.UUID, isPrivate bool) uuid.UUID {
	channelID := uuid.New()
	now := time.Now()
	f.chRepo.channels[channelID] = &models.Channel{
		ID:        channelID,
		TenantID:  tenantID,
		Name:      "general",
		IsPrivate: isPrivate,
		CreatedBy: ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.chRepo.memberships[chMembershipKey(channelID, ownerID)] = &models.ChannelMembership{
		ChannelID: channelID,
		UserID:    ownerID,
		TenantID:  tenantID,
		Role:      models.ChannelRoleOwner,
		JoinedAt:  now,
	}
	f.msgRepo.addChannel(channelID, false)
	f.msgRepo.addMember(channelID, ownerID, models.ChannelRoleOwner)
	return channelID
}

func (f *chatChannelsMessagesFixture) addMemberBoth(channelID, tenantID, userID uuid.UUID, role models.ChannelRole) {
	f.chRepo.memberships[chMembershipKey(channelID, userID)] = &models.ChannelMembership{
		ChannelID: channelID,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		JoinedAt:  time.Now(),
	}
	f.msgRepo.addMember(channelID, userID, role)
}

func (f *chatChannelsMessagesFixture) addUserBoth(userID uuid.UUID, firstName, lastName, email string) {
	f.chRepo.addUser(userID, firstName, lastName, email)
	f.msgRepo.addUser(userID, firstName, lastName)
}

func chatCtx(userID, tenantID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID.String())
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// Channels
// ============================================================================

func TestChatCreateChannel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		f.addUserBoth(ownerID, "Anna", "Admin", "anna@example.com")

		resp, err := f.srv.CreateChannel(chatCtx(ownerID, tenantID), &chatv1.CreateChannelRequest{
			Name:      "general",
			CreatedBy: ownerID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, "general", resp.Channel.Name)
		require.False(t, resp.Channel.IsPrivate)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID := uuid.New()
		_, err := f.srv.CreateChannel(chatCtx(uuid.New(), tenantID), &chatv1.CreateChannelRequest{
			Name:      "general",
			CreatedBy: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestChatGetChannel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		resp, err := f.srv.GetChannel(chatCtx(ownerID, tenantID), &chatv1.GetChannelRequest{
			Id:     channelID.String(),
			UserId: ownerID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, channelID.String(), resp.Channel.Id)
	})

	t.Run("not found", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()

		_, err := f.srv.GetChannel(chatCtx(ownerID, tenantID), &chatv1.GetChannelRequest{
			Id:     uuid.New().String(),
			UserId: ownerID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestChatListChannels(t *testing.T) {
	t.Run("success wraps empty list", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, userID := uuid.New(), uuid.New()

		resp, err := f.srv.ListChannels(chatCtx(userID, tenantID), &chatv1.ListChannelsRequest{
			UserId: userID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Channels, "Channels should be an empty slice, not nil, so it serializes to [] rather than null")
		require.Empty(t, resp.Channels)
		require.EqualValues(t, 0, resp.Total)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		_, err := f.srv.ListChannels(chatCtx(uuid.New(), uuid.New()), &chatv1.ListChannelsRequest{
			UserId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestChatUpdateChannel(t *testing.T) {
	t.Run("owner can rename", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		newName := "renamed"
		resp, err := f.srv.UpdateChannel(chatCtx(ownerID, tenantID), &chatv1.UpdateChannelRequest{
			Id:     channelID.String(),
			UserId: ownerID.String(),
			Name:   &newName,
		})
		require.NoError(t, err)
		require.Equal(t, "renamed", resp.Channel.Name)
	})

	t.Run("non-member is denied", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, stranger := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		newName := "renamed"
		_, err := f.srv.UpdateChannel(chatCtx(stranger, tenantID), &chatv1.UpdateChannelRequest{
			Id:     channelID.String(),
			UserId: stranger.String(),
			Name:   &newName,
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestChatDeleteChannel(t *testing.T) {
	t.Run("owner can delete", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		_, err := f.srv.DeleteChannel(chatCtx(ownerID, tenantID), &chatv1.DeleteChannelRequest{
			Id:     channelID.String(),
			UserId: ownerID.String(),
		})
		require.NoError(t, err)
	})

	t.Run("non-owner is denied", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, member := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addMemberBoth(channelID, tenantID, member, models.ChannelRoleMember)

		_, err := f.srv.DeleteChannel(chatCtx(member, tenantID), &chatv1.DeleteChannelRequest{
			Id:     channelID.String(),
			UserId: member.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestChatArchiveChannel(t *testing.T) {
	t.Run("owner can archive", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		resp, err := f.srv.ArchiveChannel(chatCtx(ownerID, tenantID), &chatv1.ArchiveChannelRequest{
			Id:     channelID.String(),
			UserId: ownerID.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.Channel.IsArchived)
	})

	t.Run("non-member is denied", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, stranger := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		_, err := f.srv.ArchiveChannel(chatCtx(stranger, tenantID), &chatv1.ArchiveChannelRequest{
			Id:     channelID.String(),
			UserId: stranger.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestChatJoinChannel(t *testing.T) {
	t.Run("public channel can be joined", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, joiner := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addUserBoth(joiner, "Jo", "Iner", "jo@example.com")

		resp, err := f.srv.JoinChannel(chatCtx(joiner, tenantID), &chatv1.JoinChannelRequest{
			ChannelId: channelID.String(),
			UserId:    joiner.String(),
		})
		require.NoError(t, err)
		require.Equal(t, joiner.String(), resp.Member.UserId)
	})

	t.Run("private channel is denied", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, joiner := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, true)
		f.addUserBoth(joiner, "Jo", "Iner", "jo@example.com")

		_, err := f.srv.JoinChannel(chatCtx(joiner, tenantID), &chatv1.JoinChannelRequest{
			ChannelId: channelID.String(),
			UserId:    joiner.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestChatLeaveChannel(t *testing.T) {
	t.Run("member can leave", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, member := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addMemberBoth(channelID, tenantID, member, models.ChannelRoleMember)

		_, err := f.srv.LeaveChannel(chatCtx(member, tenantID), &chatv1.LeaveChannelRequest{
			ChannelId: channelID.String(),
			UserId:    member.String(),
		})
		require.NoError(t, err)
	})

	t.Run("owner cannot leave", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		_, err := f.srv.LeaveChannel(chatCtx(ownerID, tenantID), &chatv1.LeaveChannelRequest{
			ChannelId: channelID.String(),
			UserId:    ownerID.String(),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestChatGetChannelMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addUserBoth(ownerID, "Anna", "Admin", "anna@example.com")

		resp, err := f.srv.GetChannelMembers(chatCtx(ownerID, tenantID), &chatv1.GetChannelMembersRequest{
			ChannelId: channelID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Members, 1)
		require.EqualValues(t, 1, resp.Total)
	})

	t.Run("unknown channel", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		_, err := f.srv.GetChannelMembers(chatCtx(uuid.New(), uuid.New()), &chatv1.GetChannelMembersRequest{
			ChannelId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	// TestChatGetChannelMembers/empty_is_not_nil documents the wire-shape fix
	// applied alongside fix-chat-grpc-nil-slice-wire-shape (Block C): the
	// handler (chat_grpc.go) now pre-allocates `infos` with make(..., 0, ...),
	// so a channel with zero members serializes to `[]` rather than `null`.
	t.Run("empty is not nil", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, channelID := uuid.New(), uuid.New()
		f.chRepo.channels[channelID] = &models.Channel{
			ID:       channelID,
			TenantID: tenantID,
			Name:     "empty-channel",
		}

		resp, err := f.srv.GetChannelMembers(chatCtx(uuid.New(), tenantID), &chatv1.GetChannelMembersRequest{
			ChannelId: channelID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Members, "Members should be an empty slice, not nil, so it serializes to [] rather than null")
		require.Empty(t, resp.Members)
	})
}

func TestChatUpdateMemberRole(t *testing.T) {
	t.Run("owner promotes member to admin", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, member := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addMemberBoth(channelID, tenantID, member, models.ChannelRoleMember)
		f.addUserBoth(member, "Mel", "Ber", "mel@example.com")

		resp, err := f.srv.UpdateMemberRole(chatCtx(ownerID, tenantID), &chatv1.UpdateMemberRoleRequest{
			ChannelId:       channelID.String(),
			TargetUserId:    member.String(),
			RequesterUserId: ownerID.String(),
			NewRole:         string(models.ChannelRoleAdmin),
		})
		require.NoError(t, err)
		require.Equal(t, string(models.ChannelRoleAdmin), resp.Member.Role)
	})

	t.Run("demoting the (last) owner is rejected", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, requester := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		// requester must itself be an owner to pass the authorization check,
		// so the FailedPrecondition below is provably reached via the
		// "cannot change owner" branch, not "not authorized".
		f.addMemberBoth(channelID, tenantID, requester, models.ChannelRoleOwner)

		_, err := f.srv.UpdateMemberRole(chatCtx(requester, tenantID), &chatv1.UpdateMemberRoleRequest{
			ChannelId:       channelID.String(),
			TargetUserId:    ownerID.String(),
			RequesterUserId: requester.String(),
			NewRole:         string(models.ChannelRoleMember),
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

// ============================================================================
// Messages
// ============================================================================

func TestChatSendMessage(t *testing.T) {
	t.Run("member can send", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		resp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hello",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, "hello", resp.Message.Content)
	})

	t.Run("empty content is rejected", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		_, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "   ",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestChatGetMessages(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		_, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hi",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)

		resp, err := f.srv.GetMessages(chatCtx(ownerID, tenantID), &chatv1.GetMessagesRequest{
			ChannelId: channelID.String(),
			UserId:    ownerID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Messages, 1)
	})

	t.Run("unknown channel", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		ownerID := uuid.New()
		_, err := f.srv.GetMessages(chatCtx(ownerID, uuid.New()), &chatv1.GetMessagesRequest{
			ChannelId: uuid.New().String(),
			UserId:    ownerID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	// TestChatGetMessages/empty_is_not_nil documents the wire-shape fix applied
	// alongside fix-chat-grpc-nil-slice-wire-shape (Block C): the handler
	// (chat_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so a
	// channel with zero messages serializes to `[]` rather than `null`.
	t.Run("empty is not nil", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		resp, err := f.srv.GetMessages(chatCtx(ownerID, tenantID), &chatv1.GetMessagesRequest{
			ChannelId: channelID.String(),
			UserId:    ownerID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Messages, "Messages should be an empty slice, not nil, so it serializes to [] rather than null")
		require.Empty(t, resp.Messages)
	})
}

func TestChatUpdateMessage(t *testing.T) {
	t.Run("author can edit", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		sendResp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hi",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)

		resp, err := f.srv.UpdateMessage(chatCtx(ownerID, tenantID), &chatv1.UpdateMessageRequest{
			Id:      sendResp.Message.Id,
			UserId:  ownerID.String(),
			Content: "edited",
		})
		require.NoError(t, err)
		require.Equal(t, "edited", resp.Message.Content)
	})

	t.Run("non-author is denied", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, other := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addMemberBoth(channelID, tenantID, other, models.ChannelRoleMember)
		sendResp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hi",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)

		_, err = f.srv.UpdateMessage(chatCtx(other, tenantID), &chatv1.UpdateMessageRequest{
			Id:      sendResp.Message.Id,
			UserId:  other.String(),
			Content: "hijacked",
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestChatDeleteMessage(t *testing.T) {
	t.Run("author can delete", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		sendResp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hi",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)

		_, err = f.srv.DeleteMessage(chatCtx(ownerID, tenantID), &chatv1.DeleteMessageRequest{
			Id:     sendResp.Message.Id,
			UserId: ownerID.String(),
		})
		require.NoError(t, err)
	})

	t.Run("unknown message", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		_, err := f.srv.DeleteMessage(chatCtx(ownerID, tenantID), &chatv1.DeleteMessageRequest{
			Id:     uuid.New().String(),
			UserId: ownerID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestChatGetOrCreateDM(t *testing.T) {
	t.Run("creates a new dm", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, userA, userB := uuid.New(), uuid.New(), uuid.New()
		f.addUserBoth(userA, "A", "A", "a@example.com")
		f.addUserBoth(userB, "B", "B", "b@example.com")

		resp, err := f.srv.GetOrCreateDM(chatCtx(userA, tenantID), &chatv1.GetOrCreateDMRequest{
			UserId:      userA.String(),
			OtherUserId: userB.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.Created)
		require.True(t, resp.Channel.IsDm)
	})

	t.Run("cannot dm self", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, userA := uuid.New(), uuid.New()
		_, err := f.srv.GetOrCreateDM(chatCtx(userA, tenantID), &chatv1.GetOrCreateDMRequest{
			UserId:      userA.String(),
			OtherUserId: userA.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestChatListDMs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, userA, userB := uuid.New(), uuid.New(), uuid.New()
		f.addUserBoth(userA, "A", "A", "a@example.com")
		f.addUserBoth(userB, "B", "B", "b@example.com")
		_, err := f.srv.GetOrCreateDM(chatCtx(userA, tenantID), &chatv1.GetOrCreateDMRequest{
			UserId:      userA.String(),
			OtherUserId: userB.String(),
		})
		require.NoError(t, err)

		resp, err := f.srv.ListDMs(chatCtx(userA, tenantID), &chatv1.ListDMsRequest{
			UserId: userA.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Channels, 1)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		_, err := f.srv.ListDMs(chatCtx(uuid.New(), uuid.New()), &chatv1.ListDMsRequest{
			UserId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// TestChatListDMs/empty_is_not_nil documents the wire-shape fix applied
	// alongside fix-chat-grpc-nil-slice-wire-shape (Block C): the handler
	// (chat_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so a
	// user with no DMs serializes to `[]` rather than `null`.
	t.Run("empty is not nil", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, userA := uuid.New(), uuid.New()

		resp, err := f.srv.ListDMs(chatCtx(userA, tenantID), &chatv1.ListDMsRequest{
			UserId: userA.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Channels, "Channels should be an empty slice, not nil, so it serializes to [] rather than null")
		require.Empty(t, resp.Channels)
	})
}

func TestChatGetThreadReplies(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		parentResp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "parent",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)
		parentID := parentResp.Message.Id
		_, err = f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId:       channelID.String(),
			Content:         "reply",
			CreatedBy:       ownerID.String(),
			TenantId:        tenantID.String(),
			ParentMessageId: &parentID,
		})
		require.NoError(t, err)

		resp, err := f.srv.GetThreadReplies(chatCtx(ownerID, tenantID), &chatv1.GetThreadRepliesRequest{
			ParentMessageId: parentID,
			UserId:          ownerID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, parentID, resp.Parent.Id)
		require.Len(t, resp.Replies, 1)
	})

	t.Run("unknown parent", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		_, err := f.srv.GetThreadReplies(chatCtx(ownerID, tenantID), &chatv1.GetThreadRepliesRequest{
			ParentMessageId: uuid.New().String(),
			UserId:          ownerID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestChatMarkChannelRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		sendResp, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId: channelID.String(),
			Content:   "hi",
			CreatedBy: ownerID.String(),
			TenantId:  tenantID.String(),
		})
		require.NoError(t, err)

		_, err = f.srv.MarkChannelRead(chatCtx(ownerID, tenantID), &chatv1.MarkChannelReadRequest{
			ChannelId: channelID.String(),
			UserId:    ownerID.String(),
			MessageId: sendResp.Message.Id,
		})
		require.NoError(t, err)
	})

	t.Run("unknown message", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)

		_, err := f.srv.MarkChannelRead(chatCtx(ownerID, tenantID), &chatv1.MarkChannelReadRequest{
			ChannelId: channelID.String(),
			UserId:    ownerID.String(),
			MessageId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestChatGetUnreadCounts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID := uuid.New(), uuid.New()
		f.seedChannel(tenantID, ownerID, false)

		resp, err := f.srv.GetUnreadCounts(chatCtx(ownerID, tenantID), &chatv1.GetUnreadCountsRequest{
			UserId: ownerID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.UnreadCounts, 1)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		_, err := f.srv.GetUnreadCounts(context.Background(), &chatv1.GetUnreadCountsRequest{
			UserId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestChatGetUserMentions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		tenantID, ownerID, mentioned := uuid.New(), uuid.New(), uuid.New()
		channelID := f.seedChannel(tenantID, ownerID, false)
		f.addMemberBoth(channelID, tenantID, mentioned, models.ChannelRoleMember)
		f.addUserBoth(mentioned, "Men", "Tioned", "men@example.com")

		_, err := f.srv.SendMessage(chatCtx(ownerID, tenantID), &chatv1.SendMessageRequest{
			ChannelId:        channelID.String(),
			Content:          "hi @mentioned",
			CreatedBy:        ownerID.String(),
			TenantId:         tenantID.String(),
			MentionedUserIds: []string{mentioned.String()},
		})
		require.NoError(t, err)

		resp, err := f.srv.GetUserMentions(chatCtx(mentioned, tenantID), &chatv1.GetUserMentionsRequest{
			UserId: mentioned.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Mentions, 1)
	})

	t.Run("invalid user id", func(t *testing.T) {
		f := newChatChannelsMessagesFixture()
		_, err := f.srv.GetUserMentions(context.Background(), &chatv1.GetUserMentionsRequest{
			UserId: "not-a-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
