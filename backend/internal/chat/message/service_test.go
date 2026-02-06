package message

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	messages    map[uuid.UUID]*models.Message
	channels    map[uuid.UUID]struct{ archived bool }
	memberships map[string]models.ChannelRole // key: channelID-userID
	users       map[uuid.UUID]struct{ firstName, lastName string }
	createErr   error
	getErr      error
	updateErr   error
	deleteErr   error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		messages:    make(map[uuid.UUID]*models.Message),
		channels:    make(map[uuid.UUID]struct{ archived bool }),
		memberships: make(map[string]models.ChannelRole),
		users:       make(map[uuid.UUID]struct{ firstName, lastName string }),
	}
}

func membershipKey(channelID, userID uuid.UUID) string {
	return channelID.String() + "-" + userID.String()
}

func (m *MockRepository) Create(ctx context.Context, message *models.Message) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.messages[message.ID] = message
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	msg, ok := m.messages[id]
	if !ok {
		return nil, ErrMessageNotFound
	}
	return msg, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter) ([]*models.MessageWithSender, error) {
	var result []*models.MessageWithSender
	for _, msg := range m.messages {
		if msg.ChannelID != filter.ChannelID {
			continue
		}
		if msg.IsDeleted {
			continue
		}
		if filter.ExcludeReplies && msg.ParentMessageID != nil {
			continue
		}
		user := m.users[msg.CreatedBy]
		result = append(result, &models.MessageWithSender{
			Message:         *msg,
			SenderFirstName: user.firstName,
			SenderLastName:  user.lastName,
		})
	}
	if len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, message *models.Message) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.messages[message.ID] = message
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if msg, ok := m.messages[id]; ok {
		msg.IsDeleted = true
	}
	return nil
}

func (m *MockRepository) ChannelExists(ctx context.Context, channelID uuid.UUID) (bool, error) {
	_, exists := m.channels[channelID]
	return exists, nil
}

func (m *MockRepository) IsChannelArchived(ctx context.Context, channelID uuid.UUID) (bool, error) {
	ch, ok := m.channels[channelID]
	if !ok {
		return false, ErrChannelNotFound
	}
	return ch.archived, nil
}

func (m *MockRepository) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	key := membershipKey(channelID, userID)
	_, exists := m.memberships[key]
	return exists, nil
}

func (m *MockRepository) GetMemberRole(ctx context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	key := membershipKey(channelID, userID)
	role, ok := m.memberships[key]
	if !ok {
		return "", ErrNotChannelMember
	}
	return role, nil
}

func (m *MockRepository) GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName string, err error) {
	user, ok := m.users[userID]
	if !ok {
		return "", "", nil
	}
	return user.firstName, user.lastName, nil
}

func (m *MockRepository) CreateWithReplyCount(ctx context.Context, message *models.Message) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.messages[message.ID] = message
	// Increment parent reply count
	if message.ParentMessageID != nil {
		if parent, ok := m.messages[*message.ParentMessageID]; ok {
			parent.ReplyCount++
		}
	}
	return nil
}

func (m *MockRepository) ListReplies(ctx context.Context, filter ThreadListFilter) ([]*models.MessageWithSender, error) {
	var result []*models.MessageWithSender
	for _, msg := range m.messages {
		if msg.ParentMessageID == nil || *msg.ParentMessageID != filter.ParentMessageID {
			continue
		}
		if msg.IsDeleted {
			continue
		}
		user := m.users[msg.CreatedBy]
		result = append(result, &models.MessageWithSender{
			Message:         *msg,
			SenderFirstName: user.firstName,
			SenderLastName:  user.lastName,
		})
	}
	if len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (m *MockRepository) DecrementReplyCount(ctx context.Context, messageID uuid.UUID) error {
	if msg, ok := m.messages[messageID]; ok {
		if msg.ReplyCount > 0 {
			msg.ReplyCount--
		}
	}
	return nil
}

// Helpers for setting up test data
func (m *MockRepository) AddChannel(id uuid.UUID, archived bool) {
	m.channels[id] = struct{ archived bool }{archived}
}

func (m *MockRepository) AddMember(channelID, userID uuid.UUID, role models.ChannelRole) {
	key := membershipKey(channelID, userID)
	m.memberships[key] = role
}

func (m *MockRepository) AddUser(id uuid.UUID, firstName, lastName string) {
	m.users[id] = struct{ firstName, lastName string }{firstName, lastName}
}

// ============================================================================
// Tests
// ============================================================================

func TestService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		input := CreateInput{
			ChannelID: channelID,
			Content:   "Hello, world!",
			CreatedBy: userID,
		}

		msg, err := service.Create(context.Background(), input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, msg.ID)
		assert.Equal(t, "Hello, world!", msg.Content)
		assert.Equal(t, channelID, msg.ChannelID)
		assert.Equal(t, userID, msg.CreatedBy)
		assert.Equal(t, "John", msg.SenderFirstName)
		assert.Equal(t, "Doe", msg.SenderLastName)
	})

	t.Run("empty content error", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)

		input := CreateInput{
			ChannelID: channelID,
			Content:   "   ",
			CreatedBy: userID,
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrContentRequired, err)
	})

	t.Run("channel not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		input := CreateInput{
			ChannelID: uuid.New(),
			Content:   "Hello",
			CreatedBy: uuid.New(),
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("channel archived", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, true) // archived
		repo.AddMember(channelID, userID, models.ChannelRoleMember)

		input := CreateInput{
			ChannelID: channelID,
			Content:   "Hello",
			CreatedBy: userID,
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrChannelArchived, err)
	})

	t.Run("not a member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		repo.AddChannel(channelID, false)
		// Note: not adding membership

		input := CreateInput{
			ChannelID: channelID,
			Content:   "Hello",
			CreatedBy: uuid.New(),
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

func TestService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create message first
		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Test",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Get by ID
		msg, err := service.GetByID(context.Background(), created.ID, userID)

		require.NoError(t, err)
		assert.Equal(t, created.ID, msg.ID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		_, err := service.GetByID(context.Background(), uuid.New(), uuid.New())

		assert.Equal(t, ErrMessageNotFound, err)
	})

	t.Run("not a member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		authorID := uuid.New()
		otherUserID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, authorID, models.ChannelRoleMember)
		repo.AddUser(authorID, "John", "Doe")

		// Create message
		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Test",
			CreatedBy: authorID,
		})
		require.NoError(t, err)

		// Try to get as non-member
		_, err = service.GetByID(context.Background(), created.ID, otherUserID)

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

func TestService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create messages
		_, err := service.Create(context.Background(), CreateInput{ChannelID: channelID, Content: "Message 1", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Create(context.Background(), CreateInput{ChannelID: channelID, Content: "Message 2", CreatedBy: userID})
		require.NoError(t, err)

		// List
		messages, hasMore, err := service.List(context.Background(), ListInput{
			ChannelID: channelID,
			UserID:    userID,
			Limit:     10,
		})

		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.False(t, hasMore)
	})

	t.Run("channel not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		_, _, err := service.List(context.Background(), ListInput{
			ChannelID: uuid.New(),
			UserID:    uuid.New(),
		})

		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("not a member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		repo.AddChannel(channelID, false)

		_, _, err := service.List(context.Background(), ListInput{
			ChannelID: channelID,
			UserID:    uuid.New(),
		})

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("success as author", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Original",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		updated, err := service.Update(context.Background(), created.ID, userID, UpdateInput{
			Content: "Updated content",
		})

		require.NoError(t, err)
		assert.Equal(t, "Updated content", updated.Content)
		assert.NotNil(t, updated.EditedAt)
	})

	t.Run("not author", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		authorID := uuid.New()
		otherID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, authorID, models.ChannelRoleMember)
		repo.AddMember(channelID, otherID, models.ChannelRoleMember)
		repo.AddUser(authorID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Original",
			CreatedBy: authorID,
		})
		require.NoError(t, err)

		_, err = service.Update(context.Background(), created.ID, otherID, UpdateInput{
			Content: "Hacked",
		})

		assert.Equal(t, ErrNotMessageAuthor, err)
	})

	t.Run("empty content", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Original",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		_, err = service.Update(context.Background(), created.ID, userID, UpdateInput{
			Content: "  ",
		})

		assert.Equal(t, ErrContentRequired, err)
	})

	t.Run("deleted message", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Original",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Delete the message
		err = service.Delete(context.Background(), created.ID, userID)
		require.NoError(t, err)

		// Try to update
		_, err = service.Update(context.Background(), created.ID, userID, UpdateInput{
			Content: "Updated",
		})

		assert.Equal(t, ErrMessageDeleted, err)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("success as author", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "To delete",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		err = service.Delete(context.Background(), created.ID, userID)

		require.NoError(t, err)

		// Verify soft deleted
		msg, _ := repo.GetByID(context.Background(), created.ID)
		assert.True(t, msg.IsDeleted)
	})

	t.Run("success as channel admin", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		authorID := uuid.New()
		adminID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, authorID, models.ChannelRoleMember)
		repo.AddMember(channelID, adminID, models.ChannelRoleAdmin)
		repo.AddUser(authorID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "To delete",
			CreatedBy: authorID,
		})
		require.NoError(t, err)

		// Admin deletes someone else's message
		err = service.Delete(context.Background(), created.ID, adminID)

		require.NoError(t, err)
	})

	t.Run("not authorized", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		authorID := uuid.New()
		otherID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, authorID, models.ChannelRoleMember)
		repo.AddMember(channelID, otherID, models.ChannelRoleMember)
		repo.AddUser(authorID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "To delete",
			CreatedBy: authorID,
		})
		require.NoError(t, err)

		// Other member tries to delete
		err = service.Delete(context.Background(), created.ID, otherID)

		assert.Equal(t, ErrNotAuthorized, err)
	})

	t.Run("idempotent", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		created, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "To delete",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Delete twice
		err = service.Delete(context.Background(), created.ID, userID)
		require.NoError(t, err)
		err = service.Delete(context.Background(), created.ID, userID)
		require.NoError(t, err) // Should not error
	})
}

// ============================================================================
// Thread Tests
// ============================================================================

func TestService_Create_ThreadReply(t *testing.T) {
	t.Run("success - creates reply and increments parent", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create parent message
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent message",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Create thread reply
		reply, err := service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply to parent",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})

		require.NoError(t, err)
		assert.Equal(t, &parent.ID, reply.ParentMessageID)
		// Parent reply_count should be incremented
		parentMsg := repo.messages[parent.ID]
		assert.Equal(t, 1, parentMsg.ReplyCount)
	})

	t.Run("error - parent not in same channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID1 := uuid.New()
		channelID2 := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID1, false)
		repo.AddChannel(channelID2, false)
		repo.AddMember(channelID1, userID, models.ChannelRoleMember)
		repo.AddMember(channelID2, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create message in channel 1
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID1,
			Content:   "Parent in channel 1",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Try to reply from channel 2
		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID2,
			Content:         "Reply from wrong channel",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})

		assert.Equal(t, ErrParentNotInChannel, err)
	})

	t.Run("error - nested threads not allowed", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create parent
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Create reply
		reply, err := service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})
		require.NoError(t, err)

		// Try to reply to the reply (nested)
		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Nested reply",
			CreatedBy:       userID,
			ParentMessageID: &reply.ID,
		})

		assert.Equal(t, ErrNestedThreads, err)
	})

	t.Run("error - reply to deleted parent", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create and delete parent
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		err = service.Delete(context.Background(), parent.ID, userID)
		require.NoError(t, err)

		// Try to reply
		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply to deleted",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})

		assert.Equal(t, ErrParentDeleted, err)
	})
}

func TestService_GetThreadReplies(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create parent and replies
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply 1",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})
		require.NoError(t, err)

		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply 2",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})
		require.NoError(t, err)

		result, err := service.GetThreadReplies(context.Background(), GetThreadRepliesInput{
			ParentMessageID: parent.ID,
			UserID:          userID,
			Limit:           50,
		})

		require.NoError(t, err)
		assert.Equal(t, parent.ID, result.Parent.ID)
		assert.Len(t, result.Replies, 2)
		assert.False(t, result.HasMore)
	})

	t.Run("error - not a member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		otherID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		_, err = service.GetThreadReplies(context.Background(), GetThreadRepliesInput{
			ParentMessageID: parent.ID,
			UserID:          otherID,
		})

		assert.Equal(t, ErrNotChannelMember, err)
	})

	t.Run("error - parent not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		_, err := service.GetThreadReplies(context.Background(), GetThreadRepliesInput{
			ParentMessageID: uuid.New(),
			UserID:          uuid.New(),
		})

		assert.Equal(t, ErrMessageNotFound, err)
	})
}

func TestService_Delete_ThreadReply(t *testing.T) {
	t.Run("decrements parent reply_count", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create parent and reply
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		reply, err := service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})
		require.NoError(t, err)

		// Verify parent has reply_count=1
		assert.Equal(t, 1, repo.messages[parent.ID].ReplyCount)

		// Delete reply
		err = service.Delete(context.Background(), reply.ID, userID)
		require.NoError(t, err)

		// Verify parent has reply_count=0
		assert.Equal(t, 0, repo.messages[parent.ID].ReplyCount)
	})
}

func TestService_List_ExcludesReplies(t *testing.T) {
	t.Run("thread replies excluded from channel messages", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")

		// Create parent
		parent, err := service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Parent",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Create a reply
		_, err = service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "Reply",
			CreatedBy:       userID,
			ParentMessageID: &parent.ID,
		})
		require.NoError(t, err)

		// Create another top-level message
		_, err = service.Create(context.Background(), CreateInput{
			ChannelID: channelID,
			Content:   "Another message",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// List should only return top-level messages
		messages, _, err := service.List(context.Background(), ListInput{
			ChannelID: channelID,
			UserID:    userID,
			Limit:     50,
		})

		require.NoError(t, err)
		assert.Len(t, messages, 2) // parent + another, NOT the reply
	})
}

// Ensure mock implements interface
var _ Repository = (*MockRepository)(nil)
