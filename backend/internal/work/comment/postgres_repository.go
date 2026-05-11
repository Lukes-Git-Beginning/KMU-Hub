package comment

import (
	"context"
	"errors"
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

func (r *PostgresRepository) Create(ctx context.Context, comment *models.TaskComment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_comments (id, task_id, author_id, content, quoted_comment_id, created_at, updated_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7,
		         (SELECT p.tenant_id FROM tasks t JOIN projects p ON t.project_id = p.id WHERE t.id = $2))`,
		comment.ID, comment.TaskID, comment.AuthorID, comment.Content,
		comment.QuotedCommentID, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TaskComment, error) {
	var c models.TaskComment
	err := r.pool.QueryRow(ctx,
		`SELECT tc.id, tc.task_id, tc.author_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS author_name,
		        tc.content, tc.quoted_comment_id,
		        (SELECT SUBSTRING(qc.content FROM 1 FOR 200) FROM task_comments qc WHERE qc.id = tc.quoted_comment_id) AS quoted_preview,
		        tc.created_at, tc.updated_at
		 FROM task_comments tc
		 LEFT JOIN users u ON tc.author_id = u.id
		 WHERE tc.id = $1`, id,
	).Scan(
		&c.ID, &c.TaskID, &c.AuthorID, &c.AuthorName,
		&c.Content, &c.QuotedCommentID, &c.QuotedPreview,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) List(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]TaskCommentWithAuthor, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM task_comments WHERE task_id = $1`, taskID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT tc.id, tc.task_id, tc.author_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS author_name,
		        tc.content, tc.quoted_comment_id,
		        (SELECT SUBSTRING(qc.content FROM 1 FOR 200) FROM task_comments qc WHERE qc.id = tc.quoted_comment_id) AS quoted_preview,
		        tc.created_at, tc.updated_at,
		        (SELECT COALESCE(qu.first_name || ' ' || qu.last_name, '')
		         FROM task_comments qc2
		         LEFT JOIN users qu ON qc2.author_id = qu.id
		         WHERE qc2.id = tc.quoted_comment_id) AS quoted_author_name
		 FROM task_comments tc
		 LEFT JOIN users u ON tc.author_id = u.id
		 WHERE tc.task_id = $1
		 ORDER BY tc.created_at ASC
		 LIMIT $2 OFFSET $3`,
		taskID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []TaskCommentWithAuthor
	for rows.Next() {
		var c TaskCommentWithAuthor
		if scanErr := rows.Scan(
			&c.ID, &c.TaskID, &c.AuthorID, &c.AuthorName,
			&c.Content, &c.QuotedCommentID, &c.QuotedPreview,
			&c.CreatedAt, &c.UpdatedAt, &c.QuotedAuthorName,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, content string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE task_comments SET content = $1, updated_at = $2 WHERE id = $3`,
		content, time.Now(), id,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM task_comments WHERE id = $1`, id)
	return err
}
