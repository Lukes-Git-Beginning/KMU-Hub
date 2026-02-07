package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, message *models.Message) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO messages (id, channel_id, content, is_deleted, edited_at, parent_message_id, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		message.ID, message.ChannelID, message.Content, message.IsDeleted,
		message.EditedAt, message.ParentMessageID, message.CreatedBy, message.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var m models.Message
	err := r.pool.QueryRow(ctx,
		`SELECT id, channel_id, content, is_deleted, edited_at, parent_message_id, reply_count, created_by, created_at
		 FROM messages WHERE id = $1`, id,
	).Scan(&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt,
		&m.ParentMessageID, &m.ReplyCount, &m.CreatedBy, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.MessageWithSender, error) {
	var args []any
	args = append(args, filter.ChannelID)
	argNum := 2

	// Build extra conditions
	extraConditions := ""

	// Exclude thread replies from main channel view
	if filter.ExcludeReplies {
		extraConditions += " AND m.parent_message_id IS NULL"
	}

	// Build cursor condition
	if filter.Before != nil {
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1`, *filter.Before).Scan(&cursorTime); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		} else {
			extraConditions += fmt.Sprintf(" AND m.created_at < $%d", argNum)
			args = append(args, cursorTime)
			argNum++
		}
	} else if filter.After != nil {
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1`, *filter.After).Scan(&cursorTime); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		} else {
			extraConditions += fmt.Sprintf(" AND m.created_at > $%d", argNum)
			args = append(args, cursorTime)
			argNum++
		}
	}

	query := fmt.Sprintf(`
		SELECT m.id, m.channel_id, m.content, m.is_deleted, m.edited_at,
		       m.parent_message_id, m.reply_count, m.created_by, m.created_at,
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.created_by = u.id
		WHERE m.channel_id = $1 AND m.is_deleted = FALSE%s
		ORDER BY m.created_at DESC
		LIMIT $%d`, extraConditions, argNum)

	args = append(args, filter.Limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.MessageWithSender
	for rows.Next() {
		var m models.MessageWithSender
		if scanErr := rows.Scan(
			&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt,
			&m.ParentMessageID, &m.ReplyCount, &m.CreatedBy, &m.CreatedAt,
			&m.SenderFirstName, &m.SenderLastName,
		); scanErr != nil {
			return nil, scanErr
		}
		messages = append(messages, &m)
	}

	return messages, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, message *models.Message) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET content = $1, edited_at = $2 WHERE id = $3`,
		message.Content, message.EditedAt, message.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete
	_, err := r.pool.Exec(ctx, `UPDATE messages SET is_deleted = TRUE WHERE id = $1`, id)
	return err
}

// Channel checks

func (r *PostgresRepository) ChannelExists(ctx context.Context, channelID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1)`, channelID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) IsChannelArchived(ctx context.Context, channelID uuid.UUID) (bool, error) {
	var archived bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_archived FROM channels WHERE id = $1`, channelID,
	).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrChannelNotFound
	}
	return archived, err
}

func (r *PostgresRepository) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM channel_memberships WHERE channel_id = $1 AND user_id = $2)`,
		channelID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetMemberRole(ctx context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	var role models.ChannelRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM channel_memberships WHERE channel_id = $1 AND user_id = $2`,
		channelID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotChannelMember
	}
	return role, err
}

// User info

func (r *PostgresRepository) GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT first_name, last_name FROM users WHERE id = $1`, userID,
	).Scan(&firstName, &lastName)
	return
}

// Thread operations

func (r *PostgresRepository) CreateWithReplyCount(ctx context.Context, message *models.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert the reply message
	_, err = tx.Exec(ctx,
		`INSERT INTO messages (id, channel_id, content, is_deleted, edited_at, parent_message_id, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		message.ID, message.ChannelID, message.Content, message.IsDeleted,
		message.EditedAt, message.ParentMessageID, message.CreatedBy, message.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Increment parent's reply_count
	_, err = tx.Exec(ctx,
		`UPDATE messages SET reply_count = reply_count + 1 WHERE id = $1`,
		message.ParentMessageID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) ListReplies(ctx context.Context, filter ThreadListFilter) ([]*models.MessageWithSender, error) {
	var args []any
	args = append(args, filter.ParentMessageID)
	argNum := 2

	extraConditions := ""

	if filter.Before != nil {
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1`, *filter.Before).Scan(&cursorTime); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		} else {
			extraConditions += fmt.Sprintf(" AND m.created_at < $%d", argNum)
			args = append(args, cursorTime)
			argNum++
		}
	} else if filter.After != nil {
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1`, *filter.After).Scan(&cursorTime); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		} else {
			extraConditions += fmt.Sprintf(" AND m.created_at > $%d", argNum)
			args = append(args, cursorTime)
			argNum++
		}
	}

	query := fmt.Sprintf(`
		SELECT m.id, m.channel_id, m.content, m.is_deleted, m.edited_at,
		       m.parent_message_id, m.reply_count, m.created_by, m.created_at,
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.created_by = u.id
		WHERE m.parent_message_id = $1 AND m.is_deleted = FALSE%s
		ORDER BY m.created_at ASC
		LIMIT $%d`, extraConditions, argNum)

	args = append(args, filter.Limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.MessageWithSender
	for rows.Next() {
		var m models.MessageWithSender
		if scanErr := rows.Scan(
			&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt,
			&m.ParentMessageID, &m.ReplyCount, &m.CreatedBy, &m.CreatedAt,
			&m.SenderFirstName, &m.SenderLastName,
		); scanErr != nil {
			return nil, scanErr
		}
		messages = append(messages, &m)
	}

	return messages, rows.Err()
}

func (r *PostgresRepository) DecrementReplyCount(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET reply_count = GREATEST(reply_count - 1, 0) WHERE id = $1`,
		messageID,
	)
	return err
}

// Mention operations (Sprint 3)

func (r *PostgresRepository) CreateMentions(ctx context.Context, messageID uuid.UUID, mentions []models.Mention) error {
	if len(mentions) == 0 {
		return nil
	}

	// Batch insert mentions
	query := `INSERT INTO message_mentions (message_id, user_id, mention_type, created_at) VALUES `
	var args []any
	argNum := 1
	for i, m := range mentions {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, $%d, $%d)", argNum, argNum+1, argNum+2, argNum+3)
		args = append(args, messageID, m.UserID, string(m.MentionType), m.CreatedAt)
		argNum += 4
	}
	query += " ON CONFLICT DO NOTHING"

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PostgresRepository) GetMentionsByMessages(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.MentionWithUser, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	// Build IN clause
	query := `
		SELECT mm.message_id, mm.user_id, mm.mention_type, mm.created_at,
		       u.first_name, u.last_name
		FROM message_mentions mm
		JOIN users u ON mm.user_id = u.id
		WHERE mm.message_id = ANY($1)
		ORDER BY mm.created_at`

	rows, err := r.pool.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.MentionWithUser)
	for rows.Next() {
		var mw models.MentionWithUser
		if scanErr := rows.Scan(
			&mw.MessageID, &mw.UserID, &mw.MentionType, &mw.CreatedAt,
			&mw.FirstName, &mw.LastName,
		); scanErr != nil {
			return nil, scanErr
		}
		result[mw.MessageID] = append(result[mw.MessageID], mw)
	}

	return result, rows.Err()
}

func (r *PostgresRepository) GetMentionsForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MentionDetailRow, int, error) {
	// Count total
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_mentions mm
		 JOIN messages m ON mm.message_id = m.id
		 WHERE mm.user_id = $1 AND m.is_deleted = FALSE`, userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.channel_id, m.content, mm.mention_type,
		       u.first_name, u.last_name, m.created_at
		FROM message_mentions mm
		JOIN messages m ON mm.message_id = m.id
		JOIN users u ON m.created_by = u.id
		WHERE mm.user_id = $1 AND m.is_deleted = FALSE
		ORDER BY mm.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var mentions []MentionDetailRow
	for rows.Next() {
		var md MentionDetailRow
		var createdAt time.Time
		if scanErr := rows.Scan(
			&md.MessageID, &md.ChannelID, &md.Content, &md.MentionType,
			&md.SenderFirstName, &md.SenderLastName, &createdAt,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		md.CreatedAt = createdAt.Format("2006-01-02T15:04:05Z07:00")
		mentions = append(mentions, md)
	}

	return mentions, total, rows.Err()
}

func (r *PostgresRepository) GetChannelMemberIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM channel_memberships WHERE channel_id = $1`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
