package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestGrouperNewNotificationWithoutGroupKey(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	notif := &models.Notification{
		UserID:       uuid.New(),
		EventTypeKey: "chat.mention",
		ModuleID:     "chat",
		Priority:     models.PriorityNormal,
		Title:        "Test notification",
		CreatedAt:    time.Now(),
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.Equal(t, 1, result.Notification.GroupCount)
	assert.NotEqual(t, uuid.Nil, result.Notification.ID)
	assert.Len(t, repo.notifications, 1)
}

func TestGrouperNewNotificationWithGroupKey(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	groupKey := "chat.channel.abc123"
	notif := &models.Notification{
		UserID:       uuid.New(),
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    time.Now(),
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.Equal(t, 1, result.Notification.GroupCount)
}

func TestGrouperCollapseWithExistingNotification(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	userID := uuid.New()
	groupKey := "chat.channel.abc123"
	now := time.Now()

	// Pre-existing notification
	existing := &models.Notification{
		ID:           uuid.New(),
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		GroupCount:   1,
		IsRead:       false,
		CreatedAt:    now.Add(-10 * time.Second),
	}
	repo.notifications = append(repo.notifications, existing)

	// New notification with same group key
	notif := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now,
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, 2, result.Notification.GroupCount)
	assert.Contains(t, result.Notification.Title, "(+1)")
}

func TestGrouperDoesNotCollapseOutsideWindow(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	userID := uuid.New()
	groupKey := "chat.channel.abc123"
	now := time.Now()

	// Old notification outside the window
	old := &models.Notification{
		ID:           uuid.New(),
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		GroupCount:   1,
		IsRead:       false,
		CreatedAt:    now.Add(-60 * time.Second), // 60 seconds ago, outside 30s window
	}
	repo.notifications = append(repo.notifications, old)

	// New notification
	notif := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now,
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.Equal(t, 1, result.Notification.GroupCount)
}

func TestGrouperDoesNotCollapseReadNotifications(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	userID := uuid.New()
	groupKey := "chat.channel.abc123"
	now := time.Now()

	// Already read notification
	read := &models.Notification{
		ID:           uuid.New(),
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		GroupCount:   1,
		IsRead:       true, // Already read
		CreatedAt:    now.Add(-10 * time.Second),
	}
	repo.notifications = append(repo.notifications, read)

	notif := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now,
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.True(t, result.IsNew) // Should create new since existing is read
}

func TestGrouperMultipleCollapses(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	userID := uuid.New()
	groupKey := "chat.channel.abc123"
	now := time.Now()

	// First notification
	first := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now.Add(-20 * time.Second),
	}
	result, err := grouper.ProcessNotification(context.Background(), first)
	require.NoError(t, err)
	assert.True(t, result.IsNew)

	// Second notification - should collapse
	second := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now.Add(-10 * time.Second),
	}
	result, err = grouper.ProcessNotification(context.Background(), second)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, 2, result.Notification.GroupCount)

	// Third notification - should collapse again
	third := &models.Notification{
		UserID:       userID,
		EventTypeKey: "chat.channel.message",
		ModuleID:     "chat",
		Priority:     models.PriorityLow,
		Title:        "New message in #general",
		GroupKey:     &groupKey,
		CreatedAt:    now,
	}
	result, err = grouper.ProcessNotification(context.Background(), third)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, 3, result.Notification.GroupCount)
}

func TestGrouperEmptyGroupKey(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 30*time.Second)

	empty := ""
	notif := &models.Notification{
		UserID:       uuid.New(),
		EventTypeKey: "chat.mention",
		ModuleID:     "chat",
		Priority:     models.PriorityNormal,
		Title:        "Test",
		GroupKey:     &empty,
		CreatedAt:    time.Now(),
	}

	result, err := grouper.ProcessNotification(context.Background(), notif)
	require.NoError(t, err)
	assert.True(t, result.IsNew) // Empty group key = no grouping
}

func TestGrouperDefaultWindowDuration(t *testing.T) {
	repo := newMockNotifRepo()
	grouper := NewGrouper(repo, 0) // 0 should default to 30s
	assert.Equal(t, 30*time.Second, grouper.WindowDuration())
}

func TestGrouperCreateError(t *testing.T) {
	repo := newMockNotifRepo()
	repo.createErr = assert.AnError
	grouper := NewGrouper(repo, 30*time.Second)

	notif := &models.Notification{
		UserID:       uuid.New(),
		EventTypeKey: "chat.mention",
		ModuleID:     "chat",
		Title:        "Test",
		CreatedAt:    time.Now(),
	}

	_, err := grouper.ProcessNotification(context.Background(), notif)
	assert.Error(t, err)
}

func TestGrouperFindRecentError(t *testing.T) {
	repo := newMockNotifRepo()
	repo.findRecentErr = assert.AnError
	grouper := NewGrouper(repo, 30*time.Second)

	groupKey := "test"
	notif := &models.Notification{
		UserID:       uuid.New(),
		EventTypeKey: "chat.mention",
		ModuleID:     "chat",
		Title:        "Test",
		GroupKey:     &groupKey,
		CreatedAt:    time.Now(),
	}

	_, err := grouper.ProcessNotification(context.Background(), notif)
	assert.Error(t, err)
}

func TestGrouperIncrementError(t *testing.T) {
	repo := newMockNotifRepo()
	repo.incrementErr = assert.AnError
	grouper := NewGrouper(repo, 30*time.Second)

	userID := uuid.New()
	groupKey := "test"
	now := time.Now()

	// Pre-existing notification
	existing := &models.Notification{
		ID:         uuid.New(),
		UserID:     userID,
		GroupKey:   &groupKey,
		GroupCount: 1,
		IsRead:     false,
		CreatedAt:  now.Add(-10 * time.Second),
	}
	repo.notifications = append(repo.notifications, existing)

	notif := &models.Notification{
		UserID:    userID,
		Title:     "Test",
		GroupKey:  &groupKey,
		CreatedAt: now,
	}

	_, err := grouper.ProcessNotification(context.Background(), notif)
	assert.Error(t, err)
}

func TestBuildGroupTitle(t *testing.T) {
	grouper := NewGrouper(nil, 30*time.Second)

	assert.Equal(t, "New message in #general (+1)", grouper.buildGroupTitle("New message in #general", 2))
	assert.Equal(t, "New message in #general (+4)", grouper.buildGroupTitle("New message in #general", 5))
}
