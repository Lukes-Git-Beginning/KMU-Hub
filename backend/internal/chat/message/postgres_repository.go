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
		`INSERT INTO messages (id, tenant_id, channel_id, content, is_deleted, edited_at, parent_message_id, lang, created_by, guest_session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		message.ID, message.TenantID, message.ChannelID, message.Content, message.IsDeleted,
		message.EditedAt, message.ParentMessageID, message.Lang, message.CreatedBy,
		message.GuestSessionID, message.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Message, error) {
	var m models.Message
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, channel_id, content, is_deleted, edited_at, parent_message_id, reply_count, lang, created_by, guest_session_id, created_at
		 FROM messages WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&m.ID, &m.TenantID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt,
		&m.ParentMessageID, &m.ReplyCount, &m.Lang, &m.CreatedBy, &m.GuestSessionID, &m.CreatedAt)
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
	args = append(args, filter.TenantID, filter.ChannelID)
	argNum := 3

	// Build extra conditions
	extraConditions := ""

	// Exclude thread replies from main channel view
	if filter.ExcludeReplies {
		extraConditions += " AND m.parent_message_id IS NULL"
	}

	// Build cursor condition — scope cursor lookup to tenant to prevent cross-tenant time leaks
	if filter.Before != nil {
		var cursorTime time.Time
		if err := r.pool.QueryRow(ctx,
			`SELECT created_at FROM messages WHERE id = $1 AND tenant_id = $2`,
			*filter.Before, filter.TenantID,
		).Scan(&cursorTime); err != nil {
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
		if err := r.pool.QueryRow(ctx,
			`SELECT created_at FROM messages WHERE id = $1 AND tenant_id = $2`,
			*filter.After, filter.TenantID,
		).Scan(&cursorTime); err != nil {
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
		       m.parent_message_id, m.reply_count, m.lang, m.created_by, m.guest_session_id, m.created_at,
		       COALESCE(u.first_name, ''), COALESCE(u.last_name, ''),
		       COALESCE(gs.display_name, '')
		FROM messages m
		LEFT JOIN users u ON m.created_by = u.id
		LEFT JOIN guest_sessions gs ON m.guest_session_id = gs.id
		WHERE m.tenant_id = $1 AND m.channel_id = $2 AND m.is_deleted = FALSE%s
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
			&m.ParentMessageID, &m.ReplyCount, &m.Lang, &m.CreatedBy, &m.GuestSessionID, &m.CreatedAt,
			&m.SenderFirstName, &m.SenderLastName,
			&m.GuestDisplayName,
		); scanErr != nil {
			return nil, scanErr
		}
		messages = append(messages, &m)
	}

	return messages, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, message *models.Message) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE messages SET content = $1, edited_at = $2 WHERE id = $3 AND tenant_id = $4`,
		message.Content, message.EditedAt, message.ID, message.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	// Soft delete
	tag, err := r.pool.Exec(ctx,
		`UPDATE messages SET is_deleted = TRUE WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// Channel checks

func (r *PostgresRepository) ChannelExists(ctx context.Context, channelID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1 AND tenant_id = $2)`, channelID, tenantID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) IsChannelArchived(ctx context.Context, channelID, tenantID uuid.UUID) (bool, error) {
	var archived bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_archived FROM channels WHERE id = $1 AND tenant_id = $2`, channelID, tenantID,
	).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrChannelNotFound
	}
	return archived, err
}

func (r *PostgresRepository) IsMember(ctx context.Context, channelID, tenantID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM channel_memberships cm
			JOIN channels c ON cm.channel_id = c.id
			WHERE cm.channel_id = $1 AND c.tenant_id = $2 AND cm.user_id = $3
		)`,
		channelID, tenantID, userID,
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
		`INSERT INTO messages (id, tenant_id, channel_id, content, is_deleted, edited_at, parent_message_id, lang, created_by, guest_session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		message.ID, message.TenantID, message.ChannelID, message.Content, message.IsDeleted,
		message.EditedAt, message.ParentMessageID, message.Lang, message.CreatedBy,
		message.GuestSessionID, message.CreatedAt,
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
		       m.parent_message_id, m.reply_count, m.lang, m.created_by, m.guest_session_id, m.created_at,
		       COALESCE(u.first_name, ''), COALESCE(u.last_name, ''),
		       COALESCE(gs.display_name, '')
		FROM messages m
		LEFT JOIN users u ON m.created_by = u.id
		LEFT JOIN guest_sessions gs ON m.guest_session_id = gs.id
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
			&m.ParentMessageID, &m.ReplyCount, &m.Lang, &m.CreatedBy, &m.GuestSessionID, &m.CreatedAt,
			&m.SenderFirstName, &m.SenderLastName,
			&m.GuestDisplayName,
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

func (r *PostgresRepository) GetFilesByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT cf.id, cf.message_id, cf.channel_id, cf.filename, cf.mime_type, cf.file_size,
		       cf.storage_key, cf.thumbnail_key, cf.uploaded_by, cf.is_deleted, cf.created_at, cf.deleted_at,
		       u.first_name, u.last_name
		FROM chat_files cf
		JOIN users u ON cf.uploaded_by = u.id
		WHERE cf.message_id = ANY($1) AND cf.is_deleted = FALSE
		ORDER BY cf.created_at
	`, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.ChatFileWithUploader)
	for rows.Next() {
		var f models.ChatFileWithUploader
		if scanErr := rows.Scan(
			&f.ID, &f.MessageID, &f.ChannelID, &f.Filename, &f.MimeType, &f.FileSize,
			&f.StorageKey, &f.ThumbnailKey, &f.UploadedBy, &f.IsDeleted, &f.CreatedAt, &f.DeletedAt,
			&f.UploaderFirstName, &f.UploaderLastName,
		); scanErr != nil {
			return nil, scanErr
		}
		if f.MessageID != nil {
			result[*f.MessageID] = append(result[*f.MessageID], f)
		}
	}

	return result, rows.Err()
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

func (r *PostgresRepository) GetDMRecipient(ctx context.Context, channelID, senderID uuid.UUID) (*uuid.UUID, error) {
	var dmUser1, dmUser2 *uuid.UUID
	var isDM bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_dm, dm_user1, dm_user2 FROM channels WHERE id = $1`, channelID,
	).Scan(&isDM, &dmUser1, &dmUser2)
	if err != nil {
		return nil, err
	}
	if !isDM {
		return nil, nil
	}

	// Return the other participant
	if dmUser1 != nil && *dmUser1 == senderID {
		return dmUser2, nil
	}
	return dmUser1, nil
}

func (r *PostgresRepository) GetChannelName(ctx context.Context, channelID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM channels WHERE id = $1`, channelID,
	).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *PostgresRepository) GetGuestDisplayName(ctx context.Context, sessionID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT display_name FROM guest_sessions WHERE id = $1`, sessionID,
	).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *PostgresRepository) IsChannelGuestEnabled(ctx context.Context, channelID uuid.UUID) (bool, error) {
	var enabled bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_guest_enabled FROM channels WHERE id = $1`, channelID,
	).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}
