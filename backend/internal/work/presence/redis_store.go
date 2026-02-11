package presence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// presenceKeyPrefix is the Redis key prefix for user presence
	presenceKeyPrefix = "presence:"

	// presenceTTL is the default TTL for presence keys
	presenceTTL = 90 * time.Second
)

// Store defines the interface for presence state storage
type Store interface {
	SetPresence(ctx context.Context, userID string, status PresenceStatus, manual bool) error
	GetPresence(ctx context.Context, userID string) (*UserPresence, error)
	GetBulkPresence(ctx context.Context, userIDs []string) (map[string]*UserPresence, error)
	RemovePresence(ctx context.Context, userID string) error
	UpdateLastActivity(ctx context.Context, userID string) error
}

// PresenceStatus is a string type for presence status values
type PresenceStatus = string

// presenceData is the JSON structure stored in Redis
type presenceData struct {
	Status       string    `json:"status"`
	Manual       bool      `json:"manual"`
	LastActivity time.Time `json:"last_activity"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RedisStore implements Store using Redis
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed presence store
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func presenceKey(userID string) string {
	return presenceKeyPrefix + userID
}

// SetPresence sets presence status for a user with TTL
func (s *RedisStore) SetPresence(ctx context.Context, userID string, status PresenceStatus, manual bool) error {
	now := time.Now().UTC()
	data := presenceData{
		Status:       status,
		Manual:       manual,
		LastActivity: now,
		UpdatedAt:    now,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal presence data: %w", err)
	}

	if err := s.client.Set(ctx, presenceKey(userID), jsonData, presenceTTL).Err(); err != nil {
		return fmt.Errorf("set presence: %w", err)
	}

	return nil
}

// GetPresence retrieves presence for a single user
func (s *RedisStore) GetPresence(ctx context.Context, userID string) (*UserPresence, error) {
	val, err := s.client.Get(ctx, presenceKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil // Not found = offline
	}
	if err != nil {
		return nil, fmt.Errorf("get presence: %w", err)
	}

	return parsePresenceData(userID, val)
}

// GetBulkPresence retrieves presence for multiple users using pipeline
func (s *RedisStore) GetBulkPresence(ctx context.Context, userIDs []string) (map[string]*UserPresence, error) {
	if len(userIDs) == 0 {
		return make(map[string]*UserPresence), nil
	}

	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(userIDs))

	for _, userID := range userIDs {
		cmds[userID] = pipe.Get(ctx, presenceKey(userID))
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Pipeline may return nil errors for individual keys that don't exist;
		// only treat non-nil/non-redis.Nil as actual errors
		// Check if all commands failed
		allFailed := true
		for _, cmd := range cmds {
			if cmd.Err() == nil || cmd.Err() == redis.Nil {
				allFailed = false
				break
			}
		}
		if allFailed {
			return nil, fmt.Errorf("bulk get presence: %w", err)
		}
	}

	result := make(map[string]*UserPresence, len(userIDs))
	for userID, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			continue // User is offline, skip
		}
		if err != nil {
			continue // Skip individual failures
		}

		presence, err := parsePresenceData(userID, val)
		if err != nil {
			continue // Skip corrupted data
		}
		result[userID] = presence
	}

	return result, nil
}

// RemovePresence removes a user's presence key
func (s *RedisStore) RemovePresence(ctx context.Context, userID string) error {
	if err := s.client.Del(ctx, presenceKey(userID)).Err(); err != nil {
		return fmt.Errorf("remove presence: %w", err)
	}
	return nil
}

// UpdateLastActivity updates only the last_activity timestamp and refreshes TTL
func (s *RedisStore) UpdateLastActivity(ctx context.Context, userID string) error {
	val, err := s.client.Get(ctx, presenceKey(userID)).Result()
	if err == redis.Nil {
		// No existing presence, set to online
		return s.SetPresence(ctx, userID, PresenceOnline, false)
	}
	if err != nil {
		return fmt.Errorf("get for activity update: %w", err)
	}

	var data presenceData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return fmt.Errorf("unmarshal for activity update: %w", err)
	}

	data.LastActivity = time.Now().UTC()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal updated presence: %w", err)
	}

	if err := s.client.Set(ctx, presenceKey(userID), jsonData, presenceTTL).Err(); err != nil {
		return fmt.Errorf("update activity: %w", err)
	}

	return nil
}

// parsePresenceData parses raw Redis value into UserPresence
func parsePresenceData(userID string, raw string) (*UserPresence, error) {
	var data presenceData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("unmarshal presence: %w", err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	return &UserPresence{
		UserID:         uid,
		Status:         data.Status,
		ManualOverride: data.Manual,
		LastActivity:   &data.LastActivity,
	}, nil
}
