package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL full-text search.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Search performs full-text search on document_files using PostgreSQL tsvector
// with ts_rank_cd scoring and ts_headline snippet generation.
func (r *PostgresRepository) Search(ctx context.Context, query string, filter SearchFilter) ([]SearchResult, int, error) {
	// Build the dynamic query
	var (
		conditions []string
		args       []interface{}
		argIdx     = 1
	)

	// Base condition: full-text match using German config
	conditions = append(conditions, fmt.Sprintf(
		"f.search_vector @@ plainto_tsquery('german', $%d)", argIdx,
	))
	args = append(args, query)
	queryArgIdx := argIdx
	argIdx++

	// Always exclude soft-deleted files
	conditions = append(conditions, "f.is_deleted = FALSE")

	// Optional: scope to folder
	if filter.FolderID != nil {
		conditions = append(conditions, fmt.Sprintf("f.folder_id = $%d", argIdx))
		args = append(args, *filter.FolderID)
		argIdx++
	}

	// Optional: scope to owner
	if filter.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("f.owner_id = $%d", argIdx))
		args = append(args, *filter.OwnerID)
		argIdx++
	}

	// Optional: filter by tags (any match)
	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM document_file_tags dft WHERE dft.file_id = f.id AND dft.tag_id = ANY($%d))", argIdx,
		))
		args = append(args, filter.TagIDs)
		argIdx++
	}

	// Apply defaults for limit/offset
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM document_files f WHERE %s`, whereClause)

	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		slog.Error("document search count failed",
			"query", query,
			"error", err,
		)
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	if total == 0 {
		return []SearchResult{}, 0, nil
	}

	// Main search query with ranking and snippets
	// ts_rank_cd with normalization 32 (divide by rank + 1) per research recommendation
	searchSQL := fmt.Sprintf(`
		SELECT f.id, f.folder_id, f.filename, f.mime_type, f.file_size, f.storage_key,
		       f.thumbnail_key, f.current_version, f.owner_id, f.is_favorite, f.is_deleted,
		       f.content_text, f.created_at, f.updated_at, f.deleted_at,
		       ts_rank_cd(f.search_vector, plainto_tsquery('german', $%d), 32) AS rank,
		       ts_headline('german',
		           coalesce(f.filename || ' ' || coalesce(f.content_text, ''), ''),
		           plainto_tsquery('german', $%d),
		           'StartSel=<<, StopSel=>>, MaxWords=35, MinWords=15'
		       ) AS snippet
		FROM document_files f
		WHERE %s
		ORDER BY rank DESC
		LIMIT $%d OFFSET $%d`,
		queryArgIdx, queryArgIdx, whereClause, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, searchSQL, args...)
	if err != nil {
		slog.Error("document search query failed",
			"query", query,
			"error", err,
		)
		return nil, 0, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var (
			sr   SearchResult
			file models.DocumentFile
		)
		if err := rows.Scan(
			&file.ID, &file.FolderID, &file.Filename, &file.MimeType,
			&file.FileSize, &file.StorageKey, &file.ThumbnailKey,
			&file.CurrentVersion, &file.OwnerID, &file.IsFavorite,
			&file.IsDeleted, &file.ContentText, &file.CreatedAt,
			&file.UpdatedAt, &file.DeletedAt,
			&sr.Rank, &sr.Snippet,
		); err != nil {
			slog.Error("document search row scan failed",
				"error", err,
			)
			return nil, 0, fmt.Errorf("search scan: %w", err)
		}
		sr.File = file
		results = append(results, sr)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("search rows: %w", err)
	}

	slog.Info("document search completed",
		"query", query,
		"total", total,
		"returned", len(results),
	)

	return results, total, nil
}
