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
	mentions    map[uuid.UUID][]models.Mention // key: messageID
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
		mentions:    make(map[uuid.UUID][]models.Mention),
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

func (m *MockRepository) CreateMentions(ctx context.Context, messageID uuid.UUID, mentions []models.Mention) error {
	m.mentions[messageID] = mentions
	return nil
}

func (m *MockRepository) GetMentionsByMessages(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.MentionWithUser, error) {
	result := make(map[uuid.UUID][]models.MentionWithUser)
	for _, msgID := range messageIDs {
		if mentions, ok := m.mentions[msgID]; ok {
			for _, mention := range mentions {
				user := m.users[mention.UserID]
				result[msgID] = append(result[msgID], models.MentionWithUser{
					Mention:   mention,
					FirstName: user.firstName,
					LastName:  user.lastName,
				})
			}
		}
	}
	return result, nil
}

func (m *MockRepository) GetMentionsForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MentionDetailRow, int, error) {
	var result []MentionDetailRow
	for msgID, mentions := range m.mentions {
		for _, mention := range mentions {
			if mention.UserID == userID {
				msg := m.messages[msgID]
				if msg != nil && !msg.IsDeleted {
					sender := m.users[msg.CreatedBy]
					result = append(result, MentionDetailRow{
						MessageID:       msgID,
						ChannelID:       msg.ChannelID,
						Content:         msg.Content,
						MentionType:     mention.MentionType,
						SenderFirstName: sender.firstName,
						SenderLastName:  sender.lastName,
						CreatedAt:       msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
					})
				}
			}
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []MentionDetailRow{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *MockRepository) GetChannelMemberIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for key := range m.memberships {
		// key format: channelID-userID
		if len(key) > 37 && key[:36] == channelID.String() {
			uid, err := uuid.Parse(key[37:])
			if err == nil {
				ids = append(ids, uid)
			}
		}
	}
	return ids, nil
}

func (m *MockRepository) GetFilesByMessageIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	return make(map[uuid.UUID][]models.ChatFileWithUploader), nil
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

// ============================================================================
// Mention Tests (Sprint 3)
// ============================================================================

func TestService_Create_WithMentions(t *testing.T) {
	t.Run("success - mentions stored and returned", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		mentionedUserID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddMember(channelID, mentionedUserID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")
		repo.AddUser(mentionedUserID, "Jane", "Smith")

		msg, err := service.Create(context.Background(), CreateInput{
			ChannelID:        channelID,
			Content:          "Hey @Jane!",
			CreatedBy:        userID,
			MentionedUserIDs: []uuid.UUID{mentionedUserID},
		})

		require.NoError(t, err)
		assert.Len(t, msg.Mentions, 1)
		assert.Equal(t, mentionedUserID, msg.Mentions[0].UserID)
		assert.Equal(t, models.MentionTypeUser, msg.Mentions[0].MentionType)
		assert.Equal(t, "Jane", msg.Mentions[0].FirstName)
	})
}

func TestService_Create_WithMentionEveryone(t *testing.T) {
	t.Run("success - expands to all members", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		user1 := uuid.New()
		user2 := uuid.New()
		user3 := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, user1, models.ChannelRoleMember)
		repo.AddMember(channelID, user2, models.ChannelRoleMember)
		repo.AddMember(channelID, user3, models.ChannelRoleMember)
		repo.AddUser(user1, "John", "Doe")
		repo.AddUser(user2, "Jane", "Smith")
		repo.AddUser(user3, "Bob", "Brown")

		msg, err := service.Create(context.Background(), CreateInput{
			ChannelID:       channelID,
			Content:         "@everyone check this out",
			CreatedBy:       user1,
			MentionEveryone: true,
		})

		require.NoError(t, err)
		assert.Len(t, msg.Mentions, 3) // All 3 members
		for _, m := range msg.Mentions {
			assert.Equal(t, models.MentionTypeEveryone, m.MentionType)
		}
	})
}

func TestService_Create_InvalidMention(t *testing.T) {
	t.Run("error - mentioned user not in channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		outsiderID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")
		// outsiderID is NOT added as member

		_, err := service.Create(context.Background(), CreateInput{
			ChannelID:        channelID,
			Content:          "Hey @outsider",
			CreatedBy:        userID,
			MentionedUserIDs: []uuid.UUID{outsiderID},
		})

		assert.Equal(t, ErrInvalidMention, err)
	})
}

func TestService_GetMentionsForUser(t *testing.T) {
	t.Run("returns paginated mentions", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		senderID := uuid.New()
		mentionedID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, senderID, models.ChannelRoleMember)
		repo.AddMember(channelID, mentionedID, models.ChannelRoleMember)
		repo.AddUser(senderID, "John", "Doe")
		repo.AddUser(mentionedID, "Jane", "Smith")

		// Create a message with mention
		_, err := service.Create(context.Background(), CreateInput{
			ChannelID:        channelID,
			Content:          "Hey @Jane",
			CreatedBy:        senderID,
			MentionedUserIDs: []uuid.UUID{mentionedID},
		})
		require.NoError(t, err)

		// Get mentions for the mentioned user
		mentions, total, err := service.GetMentionsForUser(context.Background(), mentionedID, 1, 20)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, mentions, 1)
		assert.Equal(t, "Hey @Jane", mentions[0].Content)
	})
}

func TestService_List_IncludesMentions(t *testing.T) {
	t.Run("mentions batch-loaded for listed messages", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		channelID := uuid.New()
		userID := uuid.New()
		mentionedID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, userID, models.ChannelRoleMember)
		repo.AddMember(channelID, mentionedID, models.ChannelRoleMember)
		repo.AddUser(userID, "John", "Doe")
		repo.AddUser(mentionedID, "Jane", "Smith")

		// Create message with mention
		_, err := service.Create(context.Background(), CreateInput{
			ChannelID:        channelID,
			Content:          "Hey @Jane!",
			CreatedBy:        userID,
			MentionedUserIDs: []uuid.UUID{mentionedID},
		})
		require.NoError(t, err)

		// List messages
		messages, _, err := service.List(context.Background(), ListInput{
			ChannelID: channelID,
			UserID:    userID,
			Limit:     50,
		})

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Len(t, messages[0].Mentions, 1)
		assert.Equal(t, mentionedID, messages[0].Mentions[0].UserID)
	})
}

// Ensure mock implements interface
var _ Repository = (*MockRepository)(nil)
