package channel

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	channels    map[uuid.UUID]*models.Channel
	memberships map[string]*models.ChannelMembership // key: channelID-userID
	users       map[uuid.UUID]struct {
		firstName, lastName, email string
	}
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
	memberAddErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		channels:    make(map[uuid.UUID]*models.Channel),
		memberships: make(map[string]*models.ChannelMembership),
		users: make(map[uuid.UUID]struct {
			firstName, lastName, email string
		}),
	}
}

func membershipKey(channelID, userID uuid.UUID) string {
	return channelID.String() + "-" + userID.String()
}

func (m *MockRepository) Create(ctx context.Context, channel *models.Channel) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.channels[channel.ID] = channel
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	channel, ok := m.channels[id]
	if !ok {
		return nil, ErrChannelNotFound
	}
	return channel, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Channel, int, error) {
	var result []*models.Channel
	for _, ch := range m.channels {
		// Check if user is member
		key := membershipKey(ch.ID, filter.UserID)
		if _, isMember := m.memberships[key]; !isMember {
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

func (m *MockRepository) Update(ctx context.Context, channel *models.Channel) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.channels[channel.ID] = channel
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.channels, id)
	return nil
}

func (m *MockRepository) AddMember(ctx context.Context, membership *models.ChannelMembership) error {
	if m.memberAddErr != nil {
		return m.memberAddErr
	}
	key := membershipKey(membership.ChannelID, membership.UserID)
	m.memberships[key] = membership
	return nil
}

func (m *MockRepository) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	key := membershipKey(channelID, userID)
	delete(m.memberships, key)
	return nil
}

func (m *MockRepository) GetMembership(ctx context.Context, channelID, userID uuid.UUID) (*models.ChannelMembership, error) {
	key := membershipKey(channelID, userID)
	membership, ok := m.memberships[key]
	if !ok {
		return nil, ErrNotChannelMember
	}
	return membership, nil
}

func (m *MockRepository) UpdateMembership(ctx context.Context, membership *models.ChannelMembership) error {
	key := membershipKey(membership.ChannelID, membership.UserID)
	m.memberships[key] = membership
	return nil
}

func (m *MockRepository) ListMembers(ctx context.Context, channelID uuid.UUID, offset, limit int) ([]*models.ChannelMember, int, error) {
	var result []*models.ChannelMember
	for key, mem := range m.memberships {
		if mem.ChannelID == channelID {
			user := m.users[mem.UserID]
			result = append(result, &models.ChannelMember{
				ChannelMembership: *mem,
				FirstName:         user.firstName,
				LastName:          user.lastName,
				Email:             user.email,
			})
			_ = key
		}
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

func (m *MockRepository) GetMemberCount(ctx context.Context, channelID uuid.UUID) (int, error) {
	count := 0
	for _, mem := range m.memberships {
		if mem.ChannelID == channelID {
			count++
		}
	}
	return count, nil
}

func (m *MockRepository) GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName, email string, err error) {
	user, ok := m.users[userID]
	if !ok {
		return "", "", "", ErrUserNotFound
	}
	return user.firstName, user.lastName, user.email, nil
}

func (m *MockRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, exists := m.users[userID]
	return exists, nil
}

func (m *MockRepository) GetLastMessage(ctx context.Context, channelID uuid.UUID) (*models.MessageWithSender, error) {
	return nil, nil
}

func (m *MockRepository) FindDMChannel(ctx context.Context, user1, user2 uuid.UUID) (*models.Channel, error) {
	for _, ch := range m.channels {
		if ch.IsDM && ch.DMUser1 != nil && ch.DMUser2 != nil &&
			*ch.DMUser1 == user1 && *ch.DMUser2 == user2 {
			return ch, nil
		}
	}
	return nil, ErrChannelNotFound
}

func (m *MockRepository) CreateDMChannel(ctx context.Context, channel *models.Channel, mem1, mem2 *models.ChannelMembership) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.channels[channel.ID] = channel
	m.memberships[membershipKey(mem1.ChannelID, mem1.UserID)] = mem1
	m.memberships[membershipKey(mem2.ChannelID, mem2.UserID)] = mem2
	return nil
}

func (m *MockRepository) UpdateLastRead(_ context.Context, channelID, userID uuid.UUID, readAt time.Time) error {
	key := membershipKey(channelID, userID)
	if mem, ok := m.memberships[key]; ok {
		mem.LastReadAt = &readAt
		return nil
	}
	return ErrNotChannelMember
}

func (m *MockRepository) GetUnreadCount(_ context.Context, channelID, userID uuid.UUID) (int, error) {
	// For test simplicity, return 0 (no messages to count)
	return 0, nil
}

func (m *MockRepository) GetUnreadCountsForUser(_ context.Context, userID uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int)
	for _, ch := range m.channels {
		key := membershipKey(ch.ID, userID)
		if _, ok := m.memberships[key]; ok {
			result[ch.ID] = 0
		}
	}
	return result, nil
}

// Helper to add a user to mock
func (m *MockRepository) AddUser(id uuid.UUID, firstName, lastName, email string) {
	m.users[id] = struct {
		firstName, lastName, email string
	}{firstName, lastName, email}
}

// ============================================================================
// Tests
// ============================================================================

func TestService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		input := CreateInput{
			Name:        "General",
			Description: ptr("General discussion"),
			IsPrivate:   false,
			CreatedBy:   userID,
		}

		ch, err := service.Create(context.Background(), input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ch.ID)
		assert.Equal(t, "General", ch.Name)
		assert.Equal(t, "General discussion", *ch.Description)
		assert.False(t, ch.IsPrivate)
		assert.Equal(t, userID, ch.CreatedBy)
		assert.Equal(t, 1, ch.MemberCount)
		assert.NotNil(t, ch.MyRole)
		assert.Equal(t, models.ChannelRoleOwner, *ch.MyRole)
	})

	t.Run("empty name error", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		input := CreateInput{
			Name:      "   ",
			CreatedBy: userID,
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrChannelNameRequired, err)
	})

	t.Run("with initial members", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		input := CreateInput{
			Name:             "Team Chat",
			CreatedBy:        userID,
			InitialMemberIDs: []uuid.UUID{memberID},
		}

		ch, err := service.Create(context.Background(), input)

		require.NoError(t, err)
		assert.Equal(t, 2, ch.MemberCount)
	})

	t.Run("invalid initial member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		input := CreateInput{
			Name:             "Team Chat",
			CreatedBy:        userID,
			InitialMemberIDs: []uuid.UUID{uuid.New()}, // Non-existent user
		}

		_, err := service.Create(context.Background(), input)

		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestService_GetByID(t *testing.T) {
	t.Run("success public channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		// Create channel first
		input := CreateInput{Name: "General", CreatedBy: userID}
		created, err := service.Create(context.Background(), input)
		require.NoError(t, err)

		// Get by ID
		ch, err := service.GetByID(context.Background(), created.ID, userID)

		require.NoError(t, err)
		assert.Equal(t, created.ID, ch.ID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		_, err := service.GetByID(context.Background(), uuid.New(), uuid.New())

		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("private channel non-member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		// Create private channel
		input := CreateInput{Name: "Secret", IsPrivate: true, CreatedBy: ownerID}
		created, err := service.Create(context.Background(), input)
		require.NoError(t, err)

		// Try to get as non-member
		_, err = service.GetByID(context.Background(), created.ID, otherID)

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

func TestService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		// Create two channels
		_, err := service.Create(context.Background(), CreateInput{Name: "Channel 1", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Create(context.Background(), CreateInput{Name: "Channel 2", CreatedBy: userID})
		require.NoError(t, err)

		// List
		channels, total, err := service.List(context.Background(), ListInput{
			UserID:   userID,
			Page:     1,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, channels, 2)
	})

	t.Run("excludes archived by default", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		// Create and archive channel
		created, err := service.Create(context.Background(), CreateInput{Name: "Old Channel", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Archive(context.Background(), created.ID, userID)
		require.NoError(t, err)

		// Create active channel
		_, err = service.Create(context.Background(), CreateInput{Name: "Active", CreatedBy: userID})
		require.NoError(t, err)

		// List without archived
		channels, _, err := service.List(context.Background(), ListInput{
			UserID:          userID,
			IncludeArchived: false,
		})

		require.NoError(t, err)
		assert.Len(t, channels, 1)
		assert.Equal(t, "Active", channels[0].Name)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("success as owner", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		// Create channel
		created, err := service.Create(context.Background(), CreateInput{Name: "Original", CreatedBy: userID})
		require.NoError(t, err)

		// Update
		newName := "Updated"
		updated, err := service.Update(context.Background(), created.ID, userID, UpdateInput{
			Name: &newName,
		})

		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Name)
	})

	t.Run("not authorized as member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		// Create channel with member
		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		// Try to update as member
		newName := "Hacked"
		_, err = service.Update(context.Background(), created.ID, memberID, UpdateInput{
			Name: &newName,
		})

		assert.Equal(t, ErrNotAuthorized, err)
	})

	t.Run("archived channel error", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		// Create and archive
		created, err := service.Create(context.Background(), CreateInput{Name: "Old", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Archive(context.Background(), created.ID, userID)
		require.NoError(t, err)

		// Try to update
		newName := "New"
		_, err = service.Update(context.Background(), created.ID, userID, UpdateInput{Name: &newName})

		assert.Equal(t, ErrChannelArchived, err)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("success as owner", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "ToDelete", CreatedBy: userID})
		require.NoError(t, err)

		err = service.Delete(context.Background(), created.ID, userID)

		require.NoError(t, err)

		// Verify deleted
		_, err = service.GetByID(context.Background(), created.ID, userID)
		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("not authorized as non-owner", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		adminID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(adminID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{adminID},
		})
		require.NoError(t, err)

		// Make admin
		_, err = service.UpdateMemberRole(context.Background(), created.ID, adminID, ownerID, models.ChannelRoleAdmin)
		require.NoError(t, err)

		// Admin tries to delete - should fail (only owner can delete)
		err = service.Delete(context.Background(), created.ID, adminID)

		assert.Equal(t, ErrNotAuthorized, err)
	})
}

func TestService_Archive(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "ToArchive", CreatedBy: userID})
		require.NoError(t, err)

		archived, err := service.Archive(context.Background(), created.ID, userID)

		require.NoError(t, err)
		assert.True(t, archived.IsArchived)
	})

	t.Run("idempotent", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "ToArchive", CreatedBy: userID})
		require.NoError(t, err)

		// Archive twice
		_, err = service.Archive(context.Background(), created.ID, userID)
		require.NoError(t, err)
		archived, err := service.Archive(context.Background(), created.ID, userID)

		require.NoError(t, err)
		assert.True(t, archived.IsArchived)
	})
}

func TestService_Join(t *testing.T) {
	t.Run("success public channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		userID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(userID, "Jane", "Doe", "jane@example.com")

		// Create public channel
		created, err := service.Create(context.Background(), CreateInput{Name: "Public", CreatedBy: ownerID})
		require.NoError(t, err)

		// Join
		member, err := service.Join(context.Background(), created.ID, userID)

		require.NoError(t, err)
		assert.Equal(t, userID, member.UserID)
		assert.Equal(t, models.ChannelRoleMember, member.Role)
	})

	t.Run("cannot join private", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		userID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(userID, "Jane", "Doe", "jane@example.com")

		// Create private channel
		created, err := service.Create(context.Background(), CreateInput{Name: "Private", IsPrivate: true, CreatedBy: ownerID})
		require.NoError(t, err)

		// Try to join
		_, err = service.Join(context.Background(), created.ID, userID)

		assert.Equal(t, ErrCannotJoinPrivate, err)
	})

	t.Run("already member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "Public", CreatedBy: userID})
		require.NoError(t, err)

		// Try to join own channel
		_, err = service.Join(context.Background(), created.ID, userID)

		assert.Equal(t, ErrAlreadyMember, err)
	})
}

func TestService_Leave(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		err = service.Leave(context.Background(), created.ID, memberID)

		require.NoError(t, err)
	})

	t.Run("owner cannot leave", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "MyChannel", CreatedBy: userID})
		require.NoError(t, err)

		err = service.Leave(context.Background(), created.ID, userID)

		assert.Equal(t, ErrCannotLeaveOwner, err)
	})
}

func TestService_UpdateMemberRole(t *testing.T) {
	t.Run("success owner promotes to admin", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		member, err := service.UpdateMemberRole(context.Background(), created.ID, memberID, ownerID, models.ChannelRoleAdmin)

		require.NoError(t, err)
		assert.Equal(t, models.ChannelRoleAdmin, member.Role)
	})

	t.Run("cannot change to owner", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		_, err = service.UpdateMemberRole(context.Background(), created.ID, memberID, ownerID, models.ChannelRoleOwner)

		assert.Equal(t, ErrCannotChangeOwner, err)
	})

	t.Run("non-owner cannot change roles", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		adminID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(adminID, "Admin", "User", "admin@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{adminID, memberID},
		})
		require.NoError(t, err)

		// Promote admin
		_, err = service.UpdateMemberRole(context.Background(), created.ID, adminID, ownerID, models.ChannelRoleAdmin)
		require.NoError(t, err)

		// Admin tries to promote member - should fail (only owner)
		_, err = service.UpdateMemberRole(context.Background(), created.ID, memberID, adminID, models.ChannelRoleAdmin)

		assert.Equal(t, ErrNotAuthorized, err)
	})
}

func TestService_GetMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		members, total, err := service.GetMembers(context.Background(), created.ID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, members, 2)
	})
}

func TestService_AddMember(t *testing.T) {
	t.Run("success private channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		inviteeID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(inviteeID, "Jane", "Doe", "jane@example.com")

		// Create private channel
		created, err := service.Create(context.Background(), CreateInput{
			Name:      "Private",
			IsPrivate: true,
			CreatedBy: ownerID,
		})
		require.NoError(t, err)

		// Owner adds member
		member, err := service.AddMember(context.Background(), created.ID, inviteeID, ownerID)

		require.NoError(t, err)
		assert.Equal(t, inviteeID, member.UserID)
		assert.Equal(t, models.ChannelRoleMember, member.Role)
	})

	t.Run("non-moderator cannot add", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		ownerID := uuid.New()
		memberID := uuid.New()
		inviteeID := uuid.New()
		repo.AddUser(ownerID, "John", "Doe", "john@example.com")
		repo.AddUser(memberID, "Jane", "Doe", "jane@example.com")
		repo.AddUser(inviteeID, "Bob", "Smith", "bob@example.com")

		created, err := service.Create(context.Background(), CreateInput{
			Name:             "Team",
			IsPrivate:        true,
			CreatedBy:        ownerID,
			InitialMemberIDs: []uuid.UUID{memberID},
		})
		require.NoError(t, err)

		// Member tries to add someone
		_, err = service.AddMember(context.Background(), created.ID, inviteeID, memberID)

		assert.Equal(t, ErrNotAuthorized, err)
	})
}

// ============================================================================
// DM Tests
// ============================================================================

func TestService_GetOrCreateDM(t *testing.T) {
	t.Run("creates new DM", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		result, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})

		require.NoError(t, err)
		assert.True(t, result.Created)
		assert.True(t, result.Channel.IsDM)
		assert.True(t, result.Channel.IsPrivate)
		assert.Equal(t, 2, result.Channel.MemberCount)
	})

	t.Run("returns existing DM", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		// Create first DM
		result1, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)
		assert.True(t, result1.Created)

		// Get the same DM again
		result2, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)
		assert.False(t, result2.Created)
		assert.Equal(t, result1.Channel.ID, result2.Channel.ID)
	})

	t.Run("idempotent with swapped user order", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		// Create DM as user->other
		result1, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)
		assert.True(t, result1.Created)

		// Try as other->user (should find existing)
		result2, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      otherID,
			OtherUserID: userID,
		})
		require.NoError(t, err)
		assert.False(t, result2.Created)
		assert.Equal(t, result1.Channel.ID, result2.Channel.ID)
	})

	t.Run("cannot DM self", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		_, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: userID,
		})

		assert.Equal(t, ErrCannotDMSelf, err)
	})

	t.Run("other user not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		_, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: uuid.New(),
		})

		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestService_ListDMs(t *testing.T) {
	t.Run("returns only DM channels", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		// Create a regular channel
		_, err := service.Create(context.Background(), CreateInput{
			Name:      "General",
			CreatedBy: userID,
		})
		require.NoError(t, err)

		// Create a DM
		_, err = service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)

		// List DMs should only return the DM
		dms, total, err := service.ListDMs(context.Background(), userID, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, dms, 1)
		assert.True(t, dms[0].IsDM)
	})
}

func TestService_Join_DM_Guard(t *testing.T) {
	t.Run("cannot join DM channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		thirdID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")
		repo.AddUser(thirdID, "Bob", "Smith", "bob@example.com")

		dm, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)

		_, err = service.Join(context.Background(), dm.Channel.ID, thirdID)
		assert.Equal(t, ErrCannotJoinDM, err)
	})
}

func TestService_Leave_DM_Guard(t *testing.T) {
	t.Run("cannot leave DM channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		dm, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)

		err = service.Leave(context.Background(), dm.Channel.ID, userID)
		assert.Equal(t, ErrCannotLeaveDM, err)
	})
}

func TestService_Delete_DM_Guard(t *testing.T) {
	t.Run("cannot delete DM channel", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		dm, err := service.GetOrCreateDM(context.Background(), GetOrCreateDMInput{
			UserID:      userID,
			OtherUserID: otherID,
		})
		require.NoError(t, err)

		err = service.Delete(context.Background(), dm.Channel.ID, userID)
		assert.Equal(t, ErrCannotDeleteDM, err)
	})
}

// ============================================================================
// Read Receipt Tests (Sprint 3)
// ============================================================================

func TestService_MarkRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "General", CreatedBy: userID})
		require.NoError(t, err)

		now := time.Now()
		err = service.MarkRead(context.Background(), created.ID, userID, now)

		require.NoError(t, err)
		// Verify the membership was updated
		mem, _ := repo.GetMembership(context.Background(), created.ID, userID)
		assert.NotNil(t, mem.LastReadAt)
	})

	t.Run("not a member", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		otherID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		repo.AddUser(otherID, "Jane", "Doe", "jane@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "General", CreatedBy: userID})
		require.NoError(t, err)

		err = service.MarkRead(context.Background(), created.ID, otherID, time.Now())

		assert.Equal(t, ErrNotChannelMember, err)
	})

	t.Run("archived channel error", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		created, err := service.Create(context.Background(), CreateInput{Name: "Old", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Archive(context.Background(), created.ID, userID)
		require.NoError(t, err)

		err = service.MarkRead(context.Background(), created.ID, userID, time.Now())

		assert.Equal(t, ErrChannelArchived, err)
	})
}

func TestService_GetUnreadCounts(t *testing.T) {
	t.Run("returns counts map", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		_, err := service.Create(context.Background(), CreateInput{Name: "Channel 1", CreatedBy: userID})
		require.NoError(t, err)
		_, err = service.Create(context.Background(), CreateInput{Name: "Channel 2", CreatedBy: userID})
		require.NoError(t, err)

		counts, err := service.GetUnreadCounts(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, counts, 2)
	})
}

func TestService_List_IncludesUnreadCount(t *testing.T) {
	t.Run("unread populated in channel list", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")

		_, err := service.Create(context.Background(), CreateInput{Name: "General", CreatedBy: userID})
		require.NoError(t, err)

		channels, _, err := service.List(context.Background(), ListInput{
			UserID: userID,
		})

		require.NoError(t, err)
		require.Len(t, channels, 1)
		// UnreadCount should be populated (0 since mock returns 0)
		assert.Equal(t, 0, channels[0].UnreadCount)
	})
}

// Helper function
func ptr[T any](v T) *T {
	return &v
}

// Ensure mock implements interface
var _ Repository = (*MockRepository)(nil)
