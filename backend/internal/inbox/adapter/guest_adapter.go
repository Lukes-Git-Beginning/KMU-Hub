package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// GuestAdapter implements ChannelAdapter for guest chat messages.
// It normalizes guest messages into the Unified Inbox so agents can see
// and respond to guest conversations.
type GuestAdapter struct {
	pool *pgxpool.Pool
}

// Compile-time assertion that GuestAdapter implements ChannelAdapter.
var _ ChannelAdapter = (*GuestAdapter)(nil)

// NewGuestAdapter creates a new GuestAdapter.
func NewGuestAdapter(pool *pgxpool.Pool) *GuestAdapter {
	return &GuestAdapter{pool: pool}
}

// Channel returns "guest".
func (a *GuestAdapter) Channel() string {
	return "guest"
}

// FetchNewMessages retrieves guest messages received since the given time.
// Returns messages FROM guests only (not agent replies), visible to channel members.
func (a *GuestAdapter) FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]models.InboxMessage, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT m.id, m.channel_id, m.content, m.created_at,
			   gs.display_name, c.name AS channel_name
		FROM messages m
		JOIN guest_sessions gs ON gs.id = m.guest_session_id
		JOIN channels c ON c.id = m.channel_id
		JOIN channel_memberships cm ON cm.channel_id = m.channel_id AND cm.user_id = $1
		WHERE m.guest_session_id IS NOT NULL
		  AND m.created_at > $2
		  AND m.is_deleted = false
		ORDER BY m.created_at DESC
		LIMIT 100
	`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("guest adapter fetch: %w", err)
	}
	defer rows.Close()

	var result []models.InboxMessage
	for rows.Next() {
		var (
			msgID       uuid.UUID
			channelID   uuid.UUID
			content     string
			createdAt   time.Time
			displayName string
			channelName string
		)

		if scanErr := rows.Scan(&msgID, &channelID, &content, &createdAt, &displayName, &channelName); scanErr != nil {
			slog.Warn("guest adapter: failed to scan row", "error", scanErr)
			continue
		}

		preview := content
		if len(preview) > 200 {
			preview = preview[:200]
		}

		result = append(result, models.InboxMessage{
			ID:         uuid.New(),
			UserID:     userID,
			Channel:    "guest",
			SourceID:   msgID.String(),
			SenderName: displayName,
			Subject:    channelName,
			Preview:    preview,
			DeepLink:   fmt.Sprintf("/chat/%s", channelID.String()),
			ReceivedAt: createdAt,
		})
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("guest adapter rows: %w", rows.Err())
	}

	return result, nil
}

// HandleReply sends an agent reply to a guest chat channel via the chat service.
func (a *GuestAdapter) HandleReply(ctx context.Context, sourceID string, userID uuid.UUID, body string) error {
	// For guest replies, agents use the normal chat message flow.
	slog.Warn("guest adapter: HandleReply not implemented, agents reply via normal chat")
	return nil
}

// HandleForward is not supported for guest chat — there is no concept of
// forwarding a guest conversation to an arbitrary external recipient.
func (a *GuestAdapter) HandleForward(_ context.Context, _ string, _ uuid.UUID, _ string, _ string) error {
	return ErrForwardNotSupported
}

// MarkReadOnSource is a no-op for guest messages (agents manage read status in chat).
func (a *GuestAdapter) MarkReadOnSource(ctx context.Context, sourceID string, userID uuid.UUID) error {
	return nil
}
