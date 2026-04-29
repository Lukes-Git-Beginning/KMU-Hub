package channel

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func (r *PostgresRepository) Create(ctx context.Context, channel *models.Channel) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channels (id, tenant_id, name, description, is_private, is_dm, is_archived, created_by, dm_user1, dm_user2, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		channel.ID, channel.TenantID, channel.Name, channel.Description, channel.IsPrivate,
		channel.IsDM, channel.IsArchived, channel.CreatedBy, channel.DMUser1, channel.DMUser2,
		channel.CreatedAt, channel.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return r.scanChannel(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_private, is_dm, is_archived, created_by, dm_user1, dm_user2, created_at, updated_at
		 FROM channels WHERE id = $1`, id,
	))
}

func (r *PostgresRepository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*models.Channel, error) {
	return r.scanChannel(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_private, is_dm, is_archived, created_by, dm_user1, dm_user2, created_at, updated_at
		 FROM channels WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	))
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Channel, int, error) {
	// Build WHERE clause - always filter by tenant and membership
	var conditions []string
	var args []any
	argNum := 1

	// Tenant isolation
	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

	// User must be a member
	conditions = append(conditions, fmt.Sprintf(`c.id IN (SELECT channel_id FROM channel_memberships WHERE user_id = $%d)`, argNum))
	args = append(args, filter.UserID)
	argNum++

	if !filter.IncludeArchived {
		conditions = append(conditions, "c.is_archived = FALSE")
	}

	if filter.IsDM != nil {
		conditions = append(conditions, fmt.Sprintf("c.is_dm = $%d", argNum))
		args = append(args, *filter.IsDM)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(c.name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM channels c %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Query with pagination, ordered by last message or created_at
	query := fmt.Sprintf(`
		SELECT c.id, c.tenant_id, c.name, c.description, c.is_private, c.is_dm, c.is_archived,
		       c.created_by, c.dm_user1, c.dm_user2, c.created_at, c.updated_at
		FROM channels c
		%s
		ORDER BY c.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var channels []*models.Channel
	for rows.Next() {
		ch, scanErr := r.scanChannelFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		channels = append(channels, ch)
	}

	return channels, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, channel *models.Channel) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE channels SET name = $1, description = $2, is_archived = $3, is_guest_enabled = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		channel.Name, channel.Description, channel.IsArchived, channel.IsGuestEnabled, channel.UpdatedAt,
		channel.ID, channel.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrChannelNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM channels WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrChannelNotFound
	}
	return nil
}

// Membership operations

func (r *PostgresRepository) AddMember(ctx context.Context, membership *models.ChannelMembership) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, role, joined_at, last_read_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		membership.ChannelID, membership.UserID, membership.Role, membership.JoinedAt, membership.LastReadAt,
	)
	return err
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM channel_memberships WHERE channel_id = $1 AND user_id = $2`,
		channelID, userID,
	)
	return err
}

func (r *PostgresRepository) GetMembership(ctx context.Context, channelID, userID uuid.UUID) (*models.ChannelMembership, error) {
	var m models.ChannelMembership
	err := r.pool.QueryRow(ctx,
		`SELECT channel_id, user_id, role, joined_at, last_read_at
		 FROM channel_memberships WHERE channel_id = $1 AND user_id = $2`,
		channelID, userID,
	).Scan(&m.ChannelID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotChannelMember
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PostgresRepository) UpdateMembership(ctx context.Context, membership *models.ChannelMembership) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channel_memberships SET role = $1, last_read_at = $2
		 WHERE channel_id = $3 AND user_id = $4`,
		membership.Role, membership.LastReadAt, membership.ChannelID, membership.UserID,
	)
	return err
}

func (r *PostgresRepository) ListMembers(ctx context.Context, channelID uuid.UUID, offset, limit int) ([]*models.ChannelMember, int, error) {
	// Count total
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE channel_id = $1`, channelID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Query with user info
	rows, err := r.pool.Query(ctx, `
		SELECT cm.channel_id, cm.user_id, cm.role, cm.joined_at, cm.last_read_at,
		       u.first_name, u.last_name, u.email
		FROM channel_memberships cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.channel_id = $1
		ORDER BY cm.role, cm.joined_at
		LIMIT $2 OFFSET $3
	`, channelID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var members []*models.ChannelMember
	for rows.Next() {
		var m models.ChannelMember
		if scanErr := rows.Scan(
			&m.ChannelID, &m.UserID, &m.Role, &m.JoinedAt, &m.LastReadAt,
			&m.FirstName, &m.LastName, &m.Email,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		members = append(members, &m)
	}

	return members, total, rows.Err()
}

func (r *PostgresRepository) GetMemberCount(ctx context.Context, channelID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channel_memberships WHERE channel_id = $1`, channelID,
	).Scan(&count)
	return count, err
}

// User info

func (r *PostgresRepository) GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName, email string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT first_name, last_name, email FROM users WHERE id = $1`, userID,
	).Scan(&firstName, &lastName, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrUserNotFound
	}
	return
}

func (r *PostgresRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID,
	).Scan(&exists)
	return exists, err
}

// Last message

func (r *PostgresRepository) GetLastMessage(ctx context.Context, channelID uuid.UUID) (*models.MessageWithSender, error) {
	var m models.MessageWithSender
	err := r.pool.QueryRow(ctx, `
		SELECT m.id, m.channel_id, m.content, m.is_deleted, m.edited_at,
		       m.parent_message_id, m.reply_count, m.created_by, m.created_at,
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.created_by = u.id
		WHERE m.channel_id = $1 AND m.is_deleted = FALSE AND m.parent_message_id IS NULL
		ORDER BY m.created_at DESC
		LIMIT 1
	`, channelID).Scan(
		&m.ID, &m.ChannelID, &m.Content, &m.IsDeleted, &m.EditedAt,
		&m.ParentMessageID, &m.ReplyCount, &m.CreatedBy, &m.CreatedAt,
		&m.SenderFirstName, &m.SenderLastName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Helper functions

func (r *PostgresRepository) scanChannel(row pgx.Row) (*models.Channel, error) {
	var c models.Channel
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.IsPrivate, &c.IsDM, &c.IsArchived,
		&c.CreatedBy, &c.DMUser1, &c.DMUser2, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChannelNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) scanChannelFromRows(rows pgx.Rows) (*models.Channel, error) {
	var c models.Channel
	err := rows.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.IsPrivate, &c.IsDM, &c.IsArchived,
		&c.CreatedBy, &c.DMUser1, &c.DMUser2, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DM operations

func (r *PostgresRepository) FindDMChannel(ctx context.Context, user1, user2, tenantID uuid.UUID) (*models.Channel, error) {
	return r.scanChannel(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_private, is_dm, is_archived, created_by, dm_user1, dm_user2, created_at, updated_at
		 FROM channels WHERE is_dm = TRUE AND dm_user1 = $1 AND dm_user2 = $2 AND tenant_id = $3`, user1, user2, tenantID,
	))
}

// Read receipts (Sprint 3)

func (r *PostgresRepository) UpdateLastRead(ctx context.Context, channelID, userID uuid.UUID, readAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channel_memberships SET last_read_at = $1 WHERE channel_id = $2 AND user_id = $3`,
		readAt, channelID, userID,
	)
	return err
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, channelID, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM messages m
		JOIN channel_memberships cm ON cm.channel_id = m.channel_id AND cm.user_id = $2
		WHERE m.channel_id = $1
		  AND m.created_at > COALESCE(cm.last_read_at, cm.joined_at)
		  AND m.parent_message_id IS NULL
		  AND m.is_deleted = FALSE
	`, channelID, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) GetUnreadCountsForUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cm.channel_id, COUNT(m.id)
		FROM channel_memberships cm
		LEFT JOIN messages m ON m.channel_id = cm.channel_id
		  AND m.created_at > COALESCE(cm.last_read_at, cm.joined_at)
		  AND m.parent_message_id IS NULL
		  AND m.is_deleted = FALSE
		WHERE cm.user_id = $1
		GROUP BY cm.channel_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]int)
	for rows.Next() {
		var channelID uuid.UUID
		var count int
		if scanErr := rows.Scan(&channelID, &count); scanErr != nil {
			return nil, scanErr
		}
		result[channelID] = count
	}
	return result, rows.Err()
}

func (r *PostgresRepository) CreateDMChannel(ctx context.Context, channel *models.Channel, mem1, mem2 *models.ChannelMembership) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert channel
	_, err = tx.Exec(ctx,
		`INSERT INTO channels (id, tenant_id, name, description, is_private, is_dm, is_archived, created_by, dm_user1, dm_user2, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		channel.ID, channel.TenantID, channel.Name, channel.Description, channel.IsPrivate,
		channel.IsDM, channel.IsArchived, channel.CreatedBy, channel.DMUser1, channel.DMUser2,
		channel.CreatedAt, channel.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Insert both memberships
	_, err = tx.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`,
		mem1.ChannelID, mem1.UserID, mem1.Role, mem1.JoinedAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`,
		mem2.ChannelID, mem2.UserID, mem2.Role, mem2.JoinedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
