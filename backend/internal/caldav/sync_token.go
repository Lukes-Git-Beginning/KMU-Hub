package caldav

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	// syncTokenPrefix is the prefix for sync token strings.
	syncTokenPrefix = "sync-token-"
)

// SyncTokenService manages sync version tracking and change logs
// for CalDAV/CardDAV incremental sync (RFC 6578).
type SyncTokenService struct {
	pool     *pgxpool.Pool
	notifier *PushNotifier // may be nil (push disabled)
}

// NewSyncTokenService creates a new sync token service.
// The optional PushNotifier parameter enables WebDAV-Push notifications
// when sync tokens are incremented. Pass nil to disable push (graceful degradation).
func NewSyncTokenService(pool *pgxpool.Pool, notifier ...*PushNotifier) *SyncTokenService {
	var n *PushNotifier
	if len(notifier) > 0 {
		n = notifier[0]
	}
	return &SyncTokenService{pool: pool, notifier: n}
}

// GetSyncToken returns the current sync token for a collection.
// Creates the sync version row (version 0) if it does not exist.
func (s *SyncTokenService) GetSyncToken(ctx context.Context, tenantID uuid.UUID, collectionType string, collectionID uuid.UUID) (string, error) {
	// Ensure the row exists with version 0 if missing
	_, err := s.pool.Exec(ctx,
		`INSERT INTO caldav_sync_versions (collection_type, collection_id, tenant_id, sync_version)
		 VALUES ($1, $2, $3, 0)
		 ON CONFLICT DO NOTHING`,
		collectionType, collectionID, tenantID,
	)
	if err != nil {
		return "", fmt.Errorf("ensure sync version row: %w", err)
	}

	var version int64
	err = s.pool.QueryRow(ctx,
		`SELECT sync_version FROM caldav_sync_versions
		 WHERE collection_type = $1 AND collection_id = $2 AND tenant_id = $3`,
		collectionType, collectionID, tenantID,
	).Scan(&version)
	if err != nil {
		return "", fmt.Errorf("get sync version: %w", err)
	}

	return formatSyncToken(version), nil
}

// IncrementAndLog atomically increments the sync version and records a change log entry.
// This is called whenever an object in the collection is created, modified, or deleted.
func (s *SyncTokenService) IncrementAndLog(ctx context.Context, tenantID uuid.UUID, collectionType string, collectionID uuid.UUID, objectPath, changeType string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert and increment sync version
	var newVersion int64
	err = tx.QueryRow(ctx,
		`INSERT INTO caldav_sync_versions (collection_type, collection_id, tenant_id, sync_version)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (collection_type, collection_id)
		 DO UPDATE SET sync_version = caldav_sync_versions.sync_version + 1
		 RETURNING sync_version`,
		collectionType, collectionID, tenantID,
	).Scan(&newVersion)
	if err != nil {
		return fmt.Errorf("increment sync version: %w", err)
	}

	// Insert change log entry
	_, err = tx.Exec(ctx,
		`INSERT INTO caldav_change_log (collection_type, collection_id, tenant_id, object_path, change_type, sync_version, changed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		collectionType, collectionID, tenantID, objectPath, changeType, newVersion,
	)
	if err != nil {
		return fmt.Errorf("insert change log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("caldav sync version incremented",
		"collection_type", collectionType,
		"collection_id", collectionID,
		"object_path", objectPath,
		"change_type", changeType,
		"new_version", newVersion,
	)

	// Fire push notifications to subscribed clients (non-blocking)
	if s.notifier != nil {
		if pushErr := s.notifier.NotifyCollectionChanged(ctx, collectionType, collectionID); pushErr != nil {
			slog.Warn("failed to send push notifications after sync token increment",
				"collection_type", collectionType,
				"collection_id", collectionID,
				"error", pushErr,
			)
		}
	}

	return nil
}

// GetChangesSince returns all change log entries since the given sync version,
// plus the current sync token. Used for incremental sync (RFC 6578 sync-collection).
func (s *SyncTokenService) GetChangesSince(ctx context.Context, tenantID uuid.UUID, collectionType string, collectionID uuid.UUID, sinceVersion int64) ([]models.CalDAVChangeLogEntry, string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, collection_type, collection_id, object_path, change_type, sync_version, changed_at
		 FROM caldav_change_log
		 WHERE collection_type = $1 AND collection_id = $2 AND tenant_id = $3 AND sync_version > $4
		 ORDER BY sync_version ASC`,
		collectionType, collectionID, tenantID, sinceVersion,
	)
	if err != nil {
		return nil, "", fmt.Errorf("query change log: %w", err)
	}
	defer rows.Close()

	var entries []models.CalDAVChangeLogEntry
	for rows.Next() {
		var e models.CalDAVChangeLogEntry
		if err := rows.Scan(
			&e.ID, &e.CollectionType, &e.CollectionID,
			&e.ObjectPath, &e.ChangeType, &e.SyncVersion, &e.ChangedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan change log entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate change log: %w", err)
	}

	if entries == nil {
		entries = []models.CalDAVChangeLogEntry{}
	}

	// Get the current sync token
	token, err := s.GetSyncToken(ctx, tenantID, collectionType, collectionID)
	if err != nil {
		return nil, "", err
	}

	return entries, token, nil
}

// ParseSyncToken extracts the version number from a "sync-token-{N}" string.
func (s *SyncTokenService) ParseSyncToken(token string) (int64, error) {
	if !strings.HasPrefix(token, syncTokenPrefix) {
		return 0, ErrInvalidSyncToken
	}

	versionStr := strings.TrimPrefix(token, syncTokenPrefix)
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return 0, ErrInvalidSyncToken
	}

	if version < 0 {
		return 0, ErrInvalidSyncToken
	}

	return version, nil
}

// formatSyncToken formats a version number as a sync token string.
func formatSyncToken(version int64) string {
	return fmt.Sprintf("%s%d", syncTokenPrefix, version)
}
