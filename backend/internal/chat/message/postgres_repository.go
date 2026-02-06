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
		`INSERT INTO messages (id, channel_id, content, is_deleted, edited_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		message.ID, message.ChannelID, message.Content, message.IsDeleted,
		message.EditedAt, message.CreatedBy, message.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var m models.Message
	err := r.pool.QueryRow(ctx,
		`SELECT id, channel_id, content, is_deleted, edited_at, created_by, created_at
		 FROM messages WHERE id = $1`, id,
	).Scan(&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt, &m.CreatedBy, &m.CreatedAt)
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

	// Build cursor condition
	cursorCondition := ""
	if filter.Before != nil {
		// Get the created_at of the cursor message
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1`, *filter.Before).Scan(&cursorTime); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Cursor message not found, just ignore
			} else {
				return nil, err
			}
		} else {
			cursorCondition = fmt.Sprintf(" AND m.created_at < $%d", argNum)
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
			cursorCondition = fmt.Sprintf(" AND m.created_at > $%d", argNum)
			args = append(args, cursorTime)
			argNum++
		}
	}

	// Build query - always exclude deleted messages
	query := fmt.Sprintf(`
		SELECT m.id, m.channel_id, m.content, m.is_deleted, m.edited_at, m.created_by, m.created_at,
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.created_by = u.id
		WHERE m.channel_id = $1 AND m.is_deleted = FALSE%s
		ORDER BY m.created_at DESC
		LIMIT $%d`, cursorCondition, argNum)

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
			&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt, &m.CreatedBy, &m.CreatedAt,
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
