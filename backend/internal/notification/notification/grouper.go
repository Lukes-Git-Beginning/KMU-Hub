package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Grouper handles smart grouping of notifications. When multiple similar
// notifications arrive within a time window, they are collapsed into a
// single notification with an incrementing count.
type Grouper struct {
	repo           Repository
	windowDuration time.Duration
}

// NewGrouper creates a new notification grouper.
func NewGrouper(repo Repository, windowDuration time.Duration) *Grouper {
	if windowDuration == 0 {
		windowDuration = 30 * time.Second
	}
	return &Grouper{
		repo:           repo,
		windowDuration: windowDuration,
	}
}

// GroupResult contains the result of a grouping operation.
type GroupResult struct {
	// Notification is the notification that was created or updated.
	Notification *models.Notification
	// IsNew is true if a new notification was created, false if an existing one was updated.
	IsNew bool
}

// ProcessNotification checks whether the notification should be grouped with
// an existing one or created as new. If groupKey is empty, no grouping is performed.
func (g *Grouper) ProcessNotification(ctx context.Context, notif *models.Notification) (*GroupResult, error) {
	// No grouping if no group key
	if notif.GroupKey == nil || *notif.GroupKey == "" {
		notif.ID = uuid.New()
		notif.GroupCount = 1
		if err := g.repo.Create(ctx, notif); err != nil {
			return nil, err
		}
		return &GroupResult{Notification: notif, IsNew: true}, nil
	}

	// Check for recent notification with same group key (tenant-scoped)
	since := notif.CreatedAt.Add(-g.windowDuration)
	existing, err := g.repo.FindRecentByGroupKey(ctx, notif.TenantID, notif.UserID, *notif.GroupKey, since)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Collapse into existing notification
		newCount := existing.GroupCount + 1
		newTitle := g.buildGroupTitle(notif.Title, newCount)
		if err := g.repo.IncrementGroupCount(ctx, existing.ID, newTitle); err != nil {
			return nil, err
		}
		existing.GroupCount = newCount
		existing.Title = newTitle
		return &GroupResult{Notification: existing, IsNew: false}, nil
	}

	// Create new notification
	notif.ID = uuid.New()
	notif.GroupCount = 1
	if err := g.repo.Create(ctx, notif); err != nil {
		return nil, err
	}
	return &GroupResult{Notification: notif, IsNew: true}, nil
}

// buildGroupTitle creates a grouped title like "5 new messages in #general".
func (g *Grouper) buildGroupTitle(originalTitle string, count int) string {
	return fmt.Sprintf("%s (+%d)", originalTitle, count-1)
}

// WindowDuration returns the current grouping window duration.
func (g *Grouper) WindowDuration() time.Duration {
	return g.windowDuration
}
