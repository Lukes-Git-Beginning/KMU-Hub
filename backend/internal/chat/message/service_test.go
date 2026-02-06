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
		user := m.users[msg.CreatedBy]
		result = append(result, &models.MessageWithSender{
			Message:         *msg,
			SenderFirstName: user.firstName,
			SenderLastName:  user.lastName,
		})
	}
	// Sort by created_at desc and apply limit
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

// Ensure mock implements interface
var _ Repository = (*MockRepository)(nil)
