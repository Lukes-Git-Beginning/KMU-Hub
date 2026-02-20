package caldav

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// defaultPushTTL is the default subscription expiry duration.
	defaultPushTTL = 7 * 24 * time.Hour // 7 days
)

// PushSubscription represents a WebDAV-Push notification subscription.
type PushSubscription struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	CollectionType string    `json:"collection_type"`
	CollectionID   uuid.UUID `json:"collection_id"`
	PushURL        string    `json:"push_url"`
	PushTopic      string    `json:"push_topic"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PushSubscriptionService manages WebDAV-Push notification subscriptions.
type PushSubscriptionService struct {
	pool *pgxpool.Pool
}

// NewPushSubscriptionService creates a new push subscription service.
func NewPushSubscriptionService(pool *pgxpool.Pool) *PushSubscriptionService {
	return &PushSubscriptionService{pool: pool}
}

// Subscribe creates or renews a push subscription for a collection.
// Uses upsert semantics: if the subscription already exists for the same
// user+collection+push_url combination, it renews the expiry.
func (s *PushSubscriptionService) Subscribe(
	ctx context.Context,
	userID uuid.UUID,
	collectionType string,
	collectionID uuid.UUID,
	pushURL string,
	pushTopic string,
	ttl time.Duration,
) (*PushSubscription, error) {
	if ttl <= 0 {
		ttl = defaultPushTTL
	}
	expiresAt := time.Now().Add(ttl)

	var sub PushSubscription
	err := s.pool.QueryRow(ctx,
		`INSERT INTO caldav_push_subscriptions
			(user_id, collection_type, collection_id, push_url, push_topic, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, collection_type, collection_id, push_url)
		 DO UPDATE SET
			push_topic = EXCLUDED.push_topic,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		 RETURNING id, user_id, collection_type, collection_id, push_url, push_topic,
		           expires_at, created_at, updated_at`,
		userID, collectionType, collectionID, pushURL, pushTopic, expiresAt,
	).Scan(
		&sub.ID, &sub.UserID, &sub.CollectionType, &sub.CollectionID,
		&sub.PushURL, &sub.PushTopic, &sub.ExpiresAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert push subscription: %w", err)
	}

	slog.Info("push subscription registered",
		"subscription_id", sub.ID,
		"user_id", userID,
		"collection_type", collectionType,
		"collection_id", collectionID,
		"push_url", pushURL,
		"expires_at", expiresAt,
	)

	return &sub, nil
}

// Unsubscribe removes a push subscription matching the given parameters.
func (s *PushSubscriptionService) Unsubscribe(
	ctx context.Context,
	userID uuid.UUID,
	collectionType string,
	collectionID uuid.UUID,
	pushURL string,
) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM caldav_push_subscriptions
		 WHERE user_id = $1 AND collection_type = $2
		   AND collection_id = $3 AND push_url = $4`,
		userID, collectionType, collectionID, pushURL,
	)
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}

	slog.Info("push subscription removed",
		"user_id", userID,
		"collection_type", collectionType,
		"collection_id", collectionID,
		"rows_affected", tag.RowsAffected(),
	)
	return nil
}

// GetSubscriptionsForCollection returns all active (non-expired) subscriptions
// for a given collection.
func (s *PushSubscriptionService) GetSubscriptionsForCollection(
	ctx context.Context,
	collectionType string,
	collectionID uuid.UUID,
) ([]*PushSubscription, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, collection_type, collection_id,
		        push_url, push_topic, expires_at, created_at, updated_at
		 FROM caldav_push_subscriptions
		 WHERE collection_type = $1 AND collection_id = $2
		   AND expires_at > NOW()
		 ORDER BY created_at`,
		collectionType, collectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*PushSubscription
	for rows.Next() {
		var sub PushSubscription
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.CollectionType, &sub.CollectionID,
			&sub.PushURL, &sub.PushTopic, &sub.ExpiresAt, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan push subscription: %w", err)
		}
		subs = append(subs, &sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate push subscriptions: %w", err)
	}

	if subs == nil {
		subs = []*PushSubscription{}
	}
	return subs, nil
}

// CleanupExpired deletes all expired push subscriptions.
// Returns the count of deleted subscriptions.
func (s *PushSubscriptionService) CleanupExpired(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM caldav_push_subscriptions WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired push subscriptions: %w", err)
	}

	count := tag.RowsAffected()
	if count > 0 {
		slog.Info("cleaned up expired push subscriptions", "count", count)
	}
	return count, nil
}

// UnsubscribeByURL removes a subscription by its push URL.
// Used when a push endpoint returns 410 Gone.
func (s *PushSubscriptionService) UnsubscribeByURL(ctx context.Context, pushURL string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM caldav_push_subscriptions WHERE push_url = $1`,
		pushURL,
	)
	if err != nil {
		return fmt.Errorf("delete push subscription by URL: %w", err)
	}
	return nil
}
