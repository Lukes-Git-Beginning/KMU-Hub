package search

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL full-text search
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL search repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SearchMessages(ctx context.Context, query string, lang string, channelIDs []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error) {
	if len(channelIDs) == 0 {
		return nil, 0, nil
	}

	// Count total matches
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM messages m
		WHERE m.channel_id = ANY($1)
		  AND m.is_deleted = FALSE
		  AND m.search_vector @@ plainto_tsquery($2::REGCONFIG, $3)`

	if err := r.pool.QueryRow(ctx, countQuery, channelIDs, lang, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count messages search: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	searchQuery := `
		SELECT m.id, m.channel_id, c.name, m.created_by, m.created_at,
		       m.parent_message_id, m.lang,
		       ts_rank(m.search_vector, plainto_tsquery($2::REGCONFIG, $3)) AS score,
		       ts_headline($2::REGCONFIG, m.content, plainto_tsquery($2::REGCONFIG, $3),
		                   'MaxWords=30, MinWords=10, StartSel=<mark>, StopSel=</mark>') AS snippet,
		       u.first_name, u.last_name
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.created_by = u.id
		WHERE m.channel_id = ANY($1)
		  AND m.is_deleted = FALSE
		  AND m.search_vector @@ plainto_tsquery($2::REGCONFIG, $3)
		ORDER BY score DESC, m.created_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.pool.Query(ctx, searchQuery, channelIDs, lang, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	var results []models.ChatSearchResult
	for rows.Next() {
		var res models.ChatSearchResult
		res.Type = models.ChatSearchResultMessage
		if scanErr := rows.Scan(
			&res.ID, &res.ChannelID, &res.ChannelName, &res.CreatedBy, &res.CreatedAt,
			&res.ParentMessageID, &res.Lang,
			&res.Score, &res.Snippet,
			&res.FirstName, &res.LastName,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		msgID := res.ID
		res.MessageID = &msgID
		results = append(results, res)
	}

	return results, total, rows.Err()
}

func (r *PostgresRepository) SearchFiles(ctx context.Context, query string, channelIDs []uuid.UUID, limit, offset int) ([]models.ChatSearchResult, int, error) {
	if len(channelIDs) == 0 {
		return nil, 0, nil
	}

	// Count total
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM chat_files cf
		WHERE cf.channel_id = ANY($1)
		  AND cf.is_deleted = FALSE
		  AND cf.search_vector @@ plainto_tsquery('simple', $2)`

	if err := r.pool.QueryRow(ctx, countQuery, channelIDs, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count files search: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	searchQuery := `
		SELECT cf.id, cf.channel_id, c.name, cf.uploaded_by, cf.created_at,
		       cf.filename, cf.mime_type, cf.file_size,
		       ts_rank(cf.search_vector, plainto_tsquery('simple', $2)) AS score,
		       ts_headline('simple', cf.filename, plainto_tsquery('simple', $2),
		                   'StartSel=<mark>, StopSel=</mark>') AS snippet,
		       u.first_name, u.last_name
		FROM chat_files cf
		JOIN channels c ON cf.channel_id = c.id
		JOIN users u ON cf.uploaded_by = u.id
		WHERE cf.channel_id = ANY($1)
		  AND cf.is_deleted = FALSE
		  AND cf.search_vector @@ plainto_tsquery('simple', $2)
		ORDER BY score DESC, cf.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, searchQuery, channelIDs, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search files: %w", err)
	}
	defer rows.Close()

	var results []models.ChatSearchResult
	for rows.Next() {
		var res models.ChatSearchResult
		res.Type = models.ChatSearchResultFile
		if scanErr := rows.Scan(
			&res.ID, &res.ChannelID, &res.ChannelName, &res.CreatedBy, &res.CreatedAt,
			&res.Filename, &res.MimeType, &res.FileSize,
			&res.Score, &res.Snippet,
			&res.FirstName, &res.LastName,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		fileID := res.ID
		res.FileID = &fileID
		results = append(results, res)
	}

	return results, total, rows.Err()
}

func (r *PostgresRepository) GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT channel_id FROM channel_memberships WHERE user_id = $1`, userID,
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
