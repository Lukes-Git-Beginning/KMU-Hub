package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
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
		INSERT INTO notifications (id, tenant_id, user_id, event_type_key, module_id, priority, actor_id, actor_name,
			resource_id, title, body, deep_link, group_key, group_count, is_read, is_pinned, is_dismissed,
			delivered_desktop, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err := r.pool.Exec(ctx, query,
		notif.ID, tenantID, notif.UserID, notif.EventTypeKey, notif.ModuleID, notif.Priority,
		notif.ActorID, notif.ActorName, notif.ResourceID, notif.Title, notif.Body, notif.DeepLink,
		notif.GroupKey, notif.GroupCount, notif.IsRead, notif.IsPinned, notif.IsDismissed,
		notif.DeliveredDesktop, notif.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Notification, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, actor_name, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, is_pinned, is_dismissed,
			delivered_desktop, snoozed_until, created_at
		FROM notifications WHERE id = $1 AND tenant_id = $2`

	notif := &models.Notification{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&notif.ID, &notif.UserID, &notif.EventTypeKey, &notif.ModuleID, &notif.Priority,
		&notif.ActorID, &notif.ActorName, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
		&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.IsPinned, &notif.IsDismissed,
		&notif.DeliveredDesktop, &notif.SnoozedUntil, &notif.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotificationNotFound
	}
	return notif, err
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Notification, int, error) {
	// Snoozed notifications stay hidden from every list until their snooze window elapses.
	where := "WHERE tenant_id = $1 AND user_id = $2 AND (snoozed_until IS NULL OR snoozed_until <= NOW())"
	args := []interface{}{filter.TenantID, filter.UserID}
	argIdx := 3

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
	if filter.IsPinned != nil {
		where += fmt.Sprintf(" AND is_pinned = $%d", argIdx)
		args = append(args, *filter.IsPinned)
		argIdx++
	}
	if filter.IsDismissed != nil {
		where += fmt.Sprintf(" AND is_dismissed = $%d", argIdx)
		args = append(args, *filter.IsDismissed)
		argIdx++
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM notifications " + where
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query — pinned notifications first, then by created_at DESC
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, actor_name, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, is_pinned, is_dismissed,
			delivered_desktop, snoozed_until, created_at
		FROM notifications %s
		ORDER BY is_pinned DESC, created_at DESC
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
			&notif.ActorID, &notif.ActorName, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
			&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.IsPinned, &notif.IsDismissed,
			&notif.DeliveredDesktop, &notif.SnoozedUntil, &notif.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, total, rows.Err()
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND user_id = $2 AND is_read = false AND (snoozed_until IS NULL OR snoozed_until <= NOW())",
		tenantID, userID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) MarkRead(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, readAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notifications SET is_read = true, read_at = $3 WHERE id = $1 AND tenant_id = $2 AND is_read = false",
		id, tenantID, readAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Check if it exists within the tenant
		var exists bool
		_ = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND tenant_id = $2)", id, tenantID).Scan(&exists)
		if !exists {
			return ErrNotificationNotFound
		}
		// Already read, no error
	}
	return nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string, readAt time.Time) (int, error) {
	var tag interface{ RowsAffected() int64 }
	var err error

	if moduleID != nil {
		tag, err = r.pool.Exec(ctx,
			"UPDATE notifications SET is_read = true, read_at = $4 WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3 AND is_read = false",
			tenantID, userID, *moduleID, readAt,
		)
	} else {
		tag, err = r.pool.Exec(ctx,
			"UPDATE notifications SET is_read = true, read_at = $3 WHERE tenant_id = $1 AND user_id = $2 AND is_read = false",
			tenantID, userID, readAt,
		)
	}
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) MarkDeliveredDesktop(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE notifications SET delivered_desktop = true WHERE id = $1 AND tenant_id = $2",
		id, tenantID,
	)
	return err
}

func (r *PostgresRepository) FindRecentByGroupKey(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, groupKey string, since time.Time) (*models.Notification, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, priority, actor_id, actor_name, resource_id,
			title, body, deep_link, group_key, group_count, is_read, read_at, is_pinned, is_dismissed,
			delivered_desktop, snoozed_until, created_at
		FROM notifications
		WHERE tenant_id = $1 AND user_id = $2 AND group_key = $3 AND created_at >= $4 AND is_read = false
		ORDER BY created_at DESC
		LIMIT 1`

	notif := &models.Notification{}
	err := r.pool.QueryRow(ctx, query, tenantID, userID, groupKey, since).Scan(
		&notif.ID, &notif.UserID, &notif.EventTypeKey, &notif.ModuleID, &notif.Priority,
		&notif.ActorID, &notif.ActorName, &notif.ResourceID, &notif.Title, &notif.Body, &notif.DeepLink,
		&notif.GroupKey, &notif.GroupCount, &notif.IsRead, &notif.ReadAt, &notif.IsPinned, &notif.IsDismissed,
		&notif.DeliveredDesktop, &notif.SnoozedUntil, &notif.CreatedAt,
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

// CreateEvent persists an event to the durability table. It deliberately runs
// under the caller's tenant context rather than a system context: the RLS policy
// added in 000271 then checks the insert for real, so a tenant stamped onto the
// row that disagrees with the one on the connection is rejected instead of
// silently stored. EventBus.dispatch stamps that context from the payload.
func (r *PostgresRepository) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, tenant_id, event_type_key, module_id, priority, actor_id, resource_id, payload, processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		event.ID, event.TenantID, event.EventTypeKey, event.ModuleID, event.Priority,
		event.ActorID, event.ResourceID, event.Payload, event.Processed, event.CreatedAt,
	)
	return err
}

// ListUnprocessedEvents is the catch-up read after downtime and spans tenants by
// definition — there is no single tenant to run it as. It therefore needs the
// system context; without it the 000271 policy admits nothing and the backlog
// would silently look empty. Each replayed event carries its own tenant_id, and
// EventBus.ProcessBacklog puts that back on the context before dispatching.
func (r *PostgresRepository) ListUnprocessedEvents(ctx context.Context, limit int) ([]models.Event, error) {
	query := `
		SELECT id, tenant_id, event_type_key, module_id, priority, actor_id, resource_id, payload, processed, created_at
		FROM events
		WHERE processed = false
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.pool.Query(sysctx.With(ctx), query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var evt models.Event
		err := rows.Scan(
			&evt.ID, &evt.TenantID, &evt.EventTypeKey, &evt.ModuleID, &evt.Priority,
			&evt.ActorID, &evt.ResourceID, &evt.Payload, &evt.Processed, &evt.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}

// MarkEventProcessed closes out an event. Like the catch-up read it runs as
// system: ProcessBacklog calls it from the worker context, which carries no
// tenant, and an event that cannot be marked processed is replayed forever.
func (r *PostgresRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	_, err := r.pool.Exec(sysctx.With(ctx),
		"UPDATE events SET processed = true WHERE id = $1",
		eventID,
	)
	return err
}

func (r *PostgresRepository) SetPinned(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, pinned bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notifications SET is_pinned = $3 WHERE id = $1 AND tenant_id = $2",
		id, tenantID, pinned,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *PostgresRepository) SetDismissed(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, dismissed bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notifications SET is_dismissed = $3 WHERE id = $1 AND tenant_id = $2",
		id, tenantID, dismissed,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *PostgresRepository) Snooze(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, until time.Time) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notifications SET snoozed_until = $3, is_read = true, read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND tenant_id = $2",
		id, tenantID, until,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *PostgresRepository) NotifyDelivery(ctx context.Context, payload string) error {
	_, err := r.pool.Exec(ctx, "SELECT pg_notify('notification_delivery', $1)", payload)
	return err
}
