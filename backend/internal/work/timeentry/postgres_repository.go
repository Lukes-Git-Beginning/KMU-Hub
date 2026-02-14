package timeentry

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

// NewPostgresRepository creates a new PostgreSQL time entry repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, entry *models.TimeEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO time_entries (id, task_id, user_id, started_at, ended_at, duration_seconds,
		  description, is_manual, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.ID, entry.TaskID, entry.UserID, entry.StartedAt, entry.EndedAt,
		entry.DurationSeconds, entry.Description, entry.IsManual,
		entry.CreatedAt, entry.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TimeEntryWithUser, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT te.id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.id = $1`, id,
	)

	var e models.TimeEntryWithUser
	err := row.Scan(
		&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
		&e.DurationSeconds, &e.Description, &e.IsManual,
		&e.CreatedAt, &e.UpdatedAt, &e.UserName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get time entry: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) Update(ctx context.Context, entry *models.TimeEntry) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE time_entries SET
		  started_at = $2, ended_at = $3, duration_seconds = $4,
		  description = $5, updated_at = $6
		 WHERE id = $1`,
		entry.ID, entry.StartedAt, entry.EndedAt,
		entry.DurationSeconds, entry.Description, entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update time entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM time_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByTask(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Count
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM time_entries WHERE task_id = $1`, taskID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count time entries: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.task_id = $1
		 ORDER BY te.started_at DESC
		 LIMIT $2 OFFSET $3`,
		taskID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.TimeEntryWithUser
	for rows.Next() {
		var e models.TimeEntryWithUser
		if scanErr := rows.Scan(
			&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM time_entries WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user time entries: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.user_id = $1
		 ORDER BY te.started_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.TimeEntryWithUser
	for rows.Next() {
		var e models.TimeEntryWithUser
		if scanErr := rows.Scan(
			&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan user time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

func (r *PostgresRepository) GetActiveTimer(ctx context.Context, userID uuid.UUID) (*models.ActiveTimer, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT te.id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        t.title AS task_title
		 FROM time_entries te
		 JOIN tasks t ON t.id = te.task_id
		 WHERE te.user_id = $1 AND te.ended_at IS NULL
		 LIMIT 1`, userID,
	)

	var a models.ActiveTimer
	err := row.Scan(
		&a.ID, &a.TaskID, &a.UserID, &a.StartedAt, &a.EndedAt,
		&a.DurationSeconds, &a.Description, &a.IsManual,
		&a.CreatedAt, &a.UpdatedAt, &a.TaskTitle,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active timer is not an error
		}
		return nil, fmt.Errorf("get active timer: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) StopActiveTimer(ctx context.Context, userID uuid.UUID) (*models.TimeEntry, error) {
	now := time.Now()
	row := r.pool.QueryRow(ctx,
		`UPDATE time_entries
		 SET ended_at = $2,
		     duration_seconds = EXTRACT(EPOCH FROM ($2 - started_at))::INTEGER,
		     updated_at = $2
		 WHERE user_id = $1 AND ended_at IS NULL
		 RETURNING id, task_id, user_id, started_at, ended_at, duration_seconds,
		           description, is_manual, created_at, updated_at`,
		userID, now,
	)

	var e models.TimeEntry
	err := row.Scan(
		&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
		&e.DurationSeconds, &e.Description, &e.IsManual,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active timer to stop
		}
		return nil, fmt.Errorf("stop active timer: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) GetTaskTimeSummary(ctx context.Context, taskID uuid.UUID) (*models.TimeEntrySummary, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
		   COALESCE(SUM(
		     CASE
		       WHEN ended_at IS NOT NULL THEN duration_seconds
		       ELSE EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
		     END
		   ), 0) AS total_duration,
		   COUNT(*) AS entry_count
		 FROM time_entries
		 WHERE task_id = $1`, taskID,
	)

	s := &models.TimeEntrySummary{TaskID: taskID}
	if err := row.Scan(&s.TotalDurationSeconds, &s.EntryCount); err != nil {
		return nil, fmt.Errorf("get task time summary: %w", err)
	}
	return s, nil
}
