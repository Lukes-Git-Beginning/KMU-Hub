package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL notification repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// sentinelTenantID is used as a placeholder when no tenant_id is available (legacy/system-generated notifications).
var sentinelTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func (r *PostgresRepository) Create(ctx context.Context, notif *models.Notification) error {
	tenantID := notif.TenantID
	if tenantID == uuid.Nil {
		tenantID = sentinelTenantID
	}
	query := `
		INSERT INTO notifications (id, tenant_id, user_id, event_type_key, module_id, priority, actor_id, resource_id,
			title, body, deep_link, group_key, group_count, is_read, delivered_desktop, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := r.pool.Exec(ctx, query,
		notif.ID, tenantID, notif.UserID, notif.EventTypeKey, notif.ModuleID, notif.Priority,
		notif.ActorID, notif.ResourceID, notif.Title, notif.Body, notif.DeepLink,
		notif.GroupKey, notif.GroupCount, notif.IsRead, notif.DeliveredDesktop, notif.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, delivered_desktop, created_at
		FROM notifications WHERE id = $1`

	notif := &models.Notification{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&notif.ID, &notif.UserID, &notif.EventTypeKey, &notif.ModuleID, &notif.Priority,
		&notif.ActorID, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
		&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.DeliveredDesktop, &notif.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotificationNotFound
	}
	return notif, err
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Notification, int, error) {
	where := "WHERE user_id = $1"
	args := []interface{}{filter.UserID}
	argIdx := 2

	if filter.ModuleID != nil {
		where += fmt.Sprintf(" AND module_id = $%d", argIdx)
		args = append(args, *filter.ModuleID)
		argIdx++
	}
	if filter.IsRead != nil {
		where += fmt.Sprintf(" AND is_read = $%d", argIdx)
		args = append(args, *filter.IsRead)
		argIdx++
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM notifications " + where
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, delivered_desktop, created_at
		FROM notifications %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		notif := &models.Notification{}
		err := rows.Scan(
			&notif.ID, &notif.UserID, &notif.EventTypeKey, &notif.ModuleID, &notif.Priority,
			&notif.ActorID, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
			&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.DeliveredDesktop, &notif.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, total, rows.Err()
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) MarkRead(ctx context.Context, id uuid.UUID, readAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notifications SET is_read = true, read_at = $2 WHERE id = $1 AND is_read = false",
		id, readAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Check if it exists
		var exists bool
		_ = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1)", id).Scan(&exists)
		if !exists {
			return ErrNotificationNotFound
		}
		// Already read, no error
	}
	return nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID uuid.UUID, moduleID *string, readAt time.Time) (int, error) {
	var tag interface{ RowsAffected() int64 }
	var err error

	if moduleID != nil {
		tag, err = r.pool.Exec(ctx,
			"UPDATE notifications SET is_read = true, read_at = $3 WHERE user_id = $1 AND module_id = $2 AND is_read = false",
			userID, *moduleID, readAt,
		)
	} else {
		tag, err = r.pool.Exec(ctx,
			"UPDATE notifications SET is_read = true, read_at = $2 WHERE user_id = $1 AND is_read = false",
			userID, readAt,
		)
	}
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) MarkDeliveredDesktop(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE notifications SET delivered_desktop = true WHERE id = $1",
		id,
	)
	return err
}

func (r *PostgresRepository) FindRecentByGroupKey(ctx context.Context, userID uuid.UUID, groupKey string, since time.Time) (*models.Notification, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, delivered_desktop, created_at
		FROM notifications
		WHERE user_id = $1 AND group_key = $2 AND created_at >= $3 AND is_read = false
		ORDER BY created_at DESC
		LIMIT 1`

	notif := &models.Notification{}
	err := r.pool.QueryRow(ctx, query, userID, groupKey, since).Scan(
		&notif.ID, &notif.UserID, &notif.EventTypeKey, &notif.ModuleID, &notif.Priority,
		&notif.ActorID, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
		&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.DeliveredDesktop, &notif.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return notif, err
}

func (r *PostgresRepository) IncrementGroupCount(ctx context.Context, id uuid.UUID, newTitle string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE notifications SET group_count = group_count + 1, title = $2 WHERE id = $1",
		id, newTitle,
	)
	return err
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, event_type_key, module_id, priority, actor_id, resource_id, payload, processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		event.ID, event.EventTypeKey, event.ModuleID, event.Priority,
		event.ActorID, event.ResourceID, event.Payload, event.Processed, event.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListUnprocessedEvents(ctx context.Context, limit int) ([]models.Event, error) {
	query := `
		SELECT id, event_type_key, module_id, priority, actor_id, resource_id, payload, processed, created_at
		FROM events
		WHERE processed = false
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var evt models.Event
		err := rows.Scan(
			&evt.ID, &evt.EventTypeKey, &evt.ModuleID, &evt.Priority,
			&evt.ActorID, &evt.ResourceID, &evt.Payload, &evt.Processed, &evt.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}

func (r *PostgresRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE events SET processed = true WHERE id = $1",
		eventID,
	)
	return err
}

func (r *PostgresRepository) NotifyDelivery(ctx context.Context, payload string) error {
	_, err := r.pool.Exec(ctx, "SELECT pg_notify('notification_delivery', $1)", payload)
	return err
}
