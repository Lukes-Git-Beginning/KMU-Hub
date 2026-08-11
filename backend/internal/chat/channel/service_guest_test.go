package channel

// Covers EnableGuest/DisableGuest/IsGuestEnabled, the only three Service
// methods in this package that no existing service_test.go touches.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_EnableDisableIsGuestEnabled(t *testing.T) {
	t.Run("enable then disable roundtrip", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		userID := uuid.New()
		repo.AddUser(userID, "John", "Doe", "john@example.com")
		created, err := service.Create(context.Background(), CreateInput{Name: "Support", CreatedBy: userID})
		require.NoError(t, err)

		enabled, err := service.IsGuestEnabled(context.Background(), created.ID, uuid.Nil)
		require.NoError(t, err)
		assert.False(t, enabled, "guest chat must be disabled by default")

		err = service.EnableGuest(context.Background(), created.ID, uuid.Nil)
		require.NoError(t, err)

		enabled, err = service.IsGuestEnabled(context.Background(), created.ID, uuid.Nil)
		require.NoError(t, err)
		assert.True(t, enabled)

		err = service.DisableGuest(context.Background(), created.ID, uuid.Nil)
		require.NoError(t, err)

		enabled, err = service.IsGuestEnabled(context.Background(), created.ID, uuid.Nil)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("EnableGuest channel not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		err := service.EnableGuest(context.Background(), uuid.New(), uuid.Nil)
		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("DisableGuest channel not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		err := service.DisableGuest(context.Background(), uuid.New(), uuid.Nil)
		assert.Equal(t, ErrChannelNotFound, err)
	})

	t.Run("IsGuestEnabled channel not found", func(t *testing.T) {
		repo := NewMockRepository()
		service := NewService(repo)

		_, err := service.IsGuestEnabled(context.Background(), uuid.New(), uuid.Nil)
		assert.Equal(t, ErrChannelNotFound, err)
	})
}
