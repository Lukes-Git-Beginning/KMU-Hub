package message

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

// NewPostgresRepository creates a new PostgreSQL inbox message repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, msg *models.InboxMessage) error {
	if msg.Status == "" {
		msg.Status = "open"
	}

	query := `
		INSERT INTO inbox_messages (
			id, tenant_id, user_id, channel, source_id, sender_name, sender_id, sender_email,
			subject, preview, is_read, is_starred, is_archived, status, snoozed_until,
			assigned_to, team_inbox_id, tags, deep_link, crm_contact_id,
			metadata, received_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22, NOW(), NOW()
		)`

	_, err := r.pool.Exec(ctx, query,
		msg.ID, msg.TenantID, msg.UserID, msg.Channel, msg.SourceID, msg.SenderName, msg.SenderID, msg.SenderEmail,
		msg.Subject, msg.Preview, msg.IsRead, msg.IsStarred, msg.IsArchived, msg.Status, msg.SnoozedUntil,
		msg.AssignedTo, msg.TeamInboxID, msg.Tags, msg.DeepLink, msg.CRMContactID,
		msg.Metadata, msg.ReceivedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.InboxMessage, error) {
	query := `
		SELECT id, user_id, channel, source_id, sender_name, sender_id, sender_email,
			subject, preview, is_read, is_starred, is_archived, status, snoozed_until,
			assigned_to, team_inbox_id, tags, deep_link, crm_contact_id,
			metadata, received_at, created_at, updated_at
		FROM inbox_messages WHERE id = $1 AND tenant_id = $2`

	msg := &models.InboxMessage{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&msg.ID, &msg.UserID, &msg.Channel, &msg.SourceID, &msg.SenderName, &msg.SenderID, &msg.SenderEmail,
		&msg.Subject, &msg.Preview, &msg.IsRead, &msg.IsStarred, &msg.IsArchived, &msg.Status, &msg.SnoozedUntil,
		&msg.AssignedTo, &msg.TeamInboxID, &msg.Tags, &msg.DeepLink, &msg.CRMContactID,
		&msg.Metadata, &msg.ReceivedAt, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrMessageNotFound
	}
	return msg, err
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.InboxMessage, int, error) {
	where := "WHERE tenant_id = $1 AND user_id = $2"
	args := []interface{}{filter.TenantID, filter.UserID}
	argIdx := 3

	if filter.Channel != nil {
		where += fmt.Sprintf(" AND channel = $%d", argIdx)
		args = append(args, *filter.Channel)
		argIdx++
	}
	if filter.IsRead != nil {
		where += fmt.Sprintf(" AND is_read = $%d", argIdx)
		args = append(args, *filter.IsRead)
		argIdx++
	}
	if filter.IsStarred != nil {
		where += fmt.Sprintf(" AND is_starred = $%d", argIdx)
		args = append(args, *filter.IsStarred)
		argIdx++
	}
	if filter.IsArchived != nil {
		where += fmt.Sprintf(" AND is_archived = $%d", argIdx)
		args = append(args, *filter.IsArchived)
		argIdx++
	} else {
		// Default: exclude archived items
		where += " AND is_archived = false"
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.TeamInboxID != nil {
		where += fmt.Sprintf(" AND team_inbox_id = $%d", argIdx)
		args = append(args, *filter.TeamInboxID)
		argIdx++
	}
	if filter.Search != nil && *filter.Search != "" {
		where += fmt.Sprintf(" AND (subject ILIKE '%%' || $%d || '%%' OR preview ILIKE '%%' || $%d || '%%' OR sender_name ILIKE '%%' || $%d || '%%')", argIdx, argIdx, argIdx)
		args = append(args, *filter.Search)
		argIdx++
	}

	// Exclude snoozed items from default view
	where += " AND (snoozed_until IS NULL OR snoozed_until <= NOW())"

	// Cursor-based pagination
	if filter.CursorReceivedAt != nil && filter.CursorID != nil {
		where += fmt.Sprintf(" AND (received_at, id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, *filter.CursorReceivedAt, *filter.CursorID)
		argIdx += 2
	}

	// Count query (without cursor and limit for total count)
	countWhere := "WHERE tenant_id = $1 AND user_id = $2"
	countArgs := []interface{}{filter.TenantID, filter.UserID}
	countIdx := 3

	if filter.Channel != nil {
		countWhere += fmt.Sprintf(" AND channel = $%d", countIdx)
		countArgs = append(countArgs, *filter.Channel)
		countIdx++
	}
	if filter.IsRead != nil {
		countWhere += fmt.Sprintf(" AND is_read = $%d", countIdx)
		countArgs = append(countArgs, *filter.IsRead)
		countIdx++
	}
	if filter.IsStarred != nil {
		countWhere += fmt.Sprintf(" AND is_starred = $%d", countIdx)
		countArgs = append(countArgs, *filter.IsStarred)
		countIdx++
	}
	if filter.IsArchived != nil {
		countWhere += fmt.Sprintf(" AND is_archived = $%d", countIdx)
		countArgs = append(countArgs, *filter.IsArchived)
		countIdx++
	} else {
		countWhere += " AND is_archived = false"
	}
	if filter.Status != nil {
		countWhere += fmt.Sprintf(" AND status = $%d", countIdx)
		countArgs = append(countArgs, *filter.Status)
		countIdx++
	}
	if filter.TeamInboxID != nil {
		countWhere += fmt.Sprintf(" AND team_inbox_id = $%d", countIdx)
		countArgs = append(countArgs, *filter.TeamInboxID)
		countIdx++
	}
	if filter.Search != nil && *filter.Search != "" {
		countWhere += fmt.Sprintf(" AND (subject ILIKE '%%' || $%d || '%%' OR preview ILIKE '%%' || $%d || '%%' OR sender_name ILIKE '%%' || $%d || '%%')", countIdx, countIdx, countIdx)
		countArgs = append(countArgs, *filter.Search)
	}
	countWhere += " AND (snoozed_until IS NULL OR snoozed_until <= NOW())"

	countQuery := "SELECT COUNT(*) FROM inbox_messages " + countWhere
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, channel, source_id, sender_name, sender_id, sender_email,
			subject, preview, is_read, is_starred, is_archived, status, snoozed_until,
			assigned_to, team_inbox_id, tags, deep_link, crm_contact_id,
			metadata, received_at, created_at, updated_at
		FROM inbox_messages %s
		ORDER BY received_at DESC, id DESC
		LIMIT $%d`, where, argIdx)
	args = append(args, pageSize)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*models.InboxMessage
	for rows.Next() {
		msg := &models.InboxMessage{}
		if err := rows.Scan(
			&msg.ID, &msg.UserID, &msg.Channel, &msg.SourceID, &msg.SenderName, &msg.SenderID, &msg.SenderEmail,
			&msg.Subject, &msg.Preview, &msg.IsRead, &msg.IsStarred, &msg.IsArchived, &msg.Status, &msg.SnoozedUntil,
			&msg.AssignedTo, &msg.TeamInboxID, &msg.Tags, &msg.DeepLink, &msg.CRMContactID,
			&msg.Metadata, &msg.ReceivedAt, &msg.CreatedAt, &msg.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		messages = append(messages, msg)
	}

	return messages, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, msg *models.InboxMessage) error {
	query := `
		UPDATE inbox_messages SET
			sender_name = $2, sender_id = $3, sender_email = $4,
			subject = $5, preview = $6, is_read = $7, is_starred = $8, is_archived = $9,
			snoozed_until = $10, assigned_to = $11, team_inbox_id = $12,
			tags = $13, deep_link = $14, crm_contact_id = $15,
			metadata = $16, updated_at = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		msg.ID, msg.SenderName, msg.SenderID, msg.SenderEmail,
		msg.Subject, msg.Preview, msg.IsRead, msg.IsStarred, msg.IsArchived,
		msg.SnoozedUntil, msg.AssignedTo, msg.TeamInboxID,
		msg.Tags, msg.DeepLink, msg.CRMContactID,
		msg.Metadata,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) MarkRead(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_read = true, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) MarkUnread(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_read = false, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) ToggleStar(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_starred = NOT is_starred, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET status = $2, updated_at = NOW() WHERE id = $1",
		id, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inbox_messages
		 SET tags = CASE WHEN $2 = ANY(tags) THEN tags ELSE array_append(tags, $2) END,
		     updated_at = NOW()
		 WHERE id = $1`,
		id, tag,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	ct, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET tags = array_remove(tags, $2), updated_at = NOW() WHERE id = $1",
		id, tag,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) Archive(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_archived = true, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_archived = false, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) Snooze(ctx context.Context, id uuid.UUID, until time.Time) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET snoozed_until = $2, is_read = true, updated_at = NOW() WHERE id = $1",
		id, until,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) Unsnooze(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET snoozed_until = NULL, is_read = false, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) UnsnoozeExpired(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET snoozed_until = NULL, is_read = false, updated_at = NOW() WHERE snoozed_until IS NOT NULL AND snoozed_until <= NOW() AND is_archived = false",
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) AssignMessage(ctx context.Context, id uuid.UUID, assigneeID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET assigned_to = $2, updated_at = NOW() WHERE id = $1 AND assigned_to IS NULL",
		id, assigneeID,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		// Check if message exists
		var exists bool
		_ = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inbox_messages WHERE id = $1)", id).Scan(&exists)
		if !exists {
			return false, ErrMessageNotFound
		}
		// Message exists but already assigned
		return false, nil
	}
	return true, nil
}

func (r *PostgresRepository) GetUnreadCounts(ctx context.Context, userID uuid.UUID) ([]models.UnreadCount, error) {
	query := `
		SELECT channel, COUNT(*) as count
		FROM inbox_messages
		WHERE user_id = $1 AND is_read = false AND is_archived = false
			AND (snoozed_until IS NULL OR snoozed_until <= NOW())
		GROUP BY channel`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []models.UnreadCount
	for rows.Next() {
		var c models.UnreadCount
		if err := rows.Scan(&c.Channel, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}

	return counts, rows.Err()
}

func (r *PostgresRepository) BulkMarkRead(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_read = true, updated_at = NOW() WHERE id = ANY($1)",
		ids,
	)
	return err
}

func (r *PostgresRepository) BulkArchive(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		"UPDATE inbox_messages SET is_archived = true, updated_at = NOW() WHERE id = ANY($1)",
		ids,
	)
	return err
}

func (r *PostgresRepository) GetBySourceID(ctx context.Context, userID uuid.UUID, channel, sourceID string) (*models.InboxMessage, error) {
	query := `
		SELECT id, user_id, channel, source_id, sender_name, sender_id, sender_email,
			subject, preview, is_read, is_starred, is_archived, status, snoozed_until,
			assigned_to, team_inbox_id, tags, deep_link, crm_contact_id,
			metadata, received_at, created_at, updated_at
		FROM inbox_messages
		WHERE user_id = $1 AND channel = $2 AND source_id = $3`

	msg := &models.InboxMessage{}
	err := r.pool.QueryRow(ctx, query, userID, channel, sourceID).Scan(
		&msg.ID, &msg.UserID, &msg.Channel, &msg.SourceID, &msg.SenderName, &msg.SenderID, &msg.SenderEmail,
		&msg.Subject, &msg.Preview, &msg.IsRead, &msg.IsStarred, &msg.IsArchived, &msg.Status, &msg.SnoozedUntil,
		&msg.AssignedTo, &msg.TeamInboxID, &msg.Tags, &msg.DeepLink, &msg.CRMContactID,
		&msg.Metadata, &msg.ReceivedAt, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil // Not found is not an error for dedup checks
	}
	return msg, err
}

func (r *PostgresRepository) NotifyDelivery(ctx context.Context, payload string) error {
	_, err := r.pool.Exec(ctx, "SELECT pg_notify('notification_delivery', $1)", payload)
	return err
}
