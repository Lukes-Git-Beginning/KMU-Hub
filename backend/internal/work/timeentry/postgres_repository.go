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
		`INSERT INTO time_entries (id, tenant_id, task_id, user_id, started_at, ended_at, duration_seconds,
		  description, is_manual, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		entry.ID, entry.TenantID, entry.TaskID, entry.UserID, entry.StartedAt, entry.EndedAt,
		entry.DurationSeconds, entry.Description, entry.IsManual,
		entry.CreatedAt, entry.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.TimeEntryWithUser, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.id = $1 AND te.tenant_id = $2`, id, tenantID,
	)

	var e models.TimeEntryWithUser
	err := row.Scan(
		&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
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
		 WHERE id = $1 AND tenant_id = $7`,
		entry.ID, entry.StartedAt, entry.EndedAt,
		entry.DurationSeconds, entry.Description, entry.UpdatedAt,
		entry.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update time entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM time_entries WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByTask(ctx context.Context, taskID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
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
		`SELECT COUNT(*) FROM time_entries WHERE task_id = $1 AND tenant_id = $2`, taskID, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count time entries: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.task_id = $1 AND te.tenant_id = $2
		 ORDER BY te.started_at DESC
		 LIMIT $3 OFFSET $4`,
		taskID, tenantID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.TimeEntryWithUser
	for rows.Next() {
		var e models.TimeEntryWithUser
		if scanErr := rows.Scan(
			&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM time_entries WHERE user_id = $1 AND tenant_id = $2`, userID, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user time entries: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 WHERE te.user_id = $1 AND te.tenant_id = $2
		 ORDER BY te.started_at DESC
		 LIMIT $3 OFFSET $4`,
		userID, tenantID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.TimeEntryWithUser
	for rows.Next() {
		var e models.TimeEntryWithUser
		if scanErr := rows.Scan(
			&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan user time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

// ListBillable returns completed time entries (a duration was recorded, the
// timer isn't still running) across every task/project of the tenant, joined
// with the task title, project name, and employee display name. Used by the
// finance "Stunden -> Rechnung" view, which invoices across projects rather
// than one task at a time like ListByTask.
func (r *PostgresRepository) ListBillable(ctx context.Context, tenantID uuid.UUID) ([]models.BillableTimeEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name,
		        t.title AS task_title,
		        COALESCE(p.name, '') AS project_name
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 JOIN tasks t ON t.id = te.task_id AND t.tenant_id = te.tenant_id
		 LEFT JOIN projects p ON p.id = t.project_id AND p.tenant_id = te.tenant_id
		 WHERE te.tenant_id = $1
		   AND te.ended_at IS NOT NULL
		   AND te.duration_seconds IS NOT NULL
		 ORDER BY te.started_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list billable time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.BillableTimeEntry
	for rows.Next() {
		var e models.BillableTimeEntry
		if scanErr := rows.Scan(
			&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName, &e.TaskTitle, &e.ProjectName,
		); scanErr != nil {
			return nil, fmt.Errorf("scan billable time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListByProject returns completed time entries whose task belongs to the
// given project, joined with the task title and contributor display name,
// for that project's "Stunden abrechnen" roll-up. Scoped through the
// tasks/projects join on tenant_id, not just te.tenant_id, so a project_id
// from another tenant returns zero rows instead of leaking that tenant's
// entries -- defense in depth alongside RLS, not a replacement for it.
func (r *PostgresRepository) ListByProject(ctx context.Context, projectID, tenantID uuid.UUID) ([]models.ProjectTimeEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        u.first_name || ' ' || u.last_name AS user_name,
		        t.title AS task_title
		 FROM time_entries te
		 JOIN users u ON u.id = te.user_id
		 JOIN tasks t ON t.id = te.task_id AND t.tenant_id = te.tenant_id
		 JOIN projects p ON p.id = t.project_id AND p.tenant_id = te.tenant_id
		 WHERE p.id = $1 AND p.tenant_id = $2 AND te.tenant_id = $2
		   AND te.ended_at IS NOT NULL
		   AND te.duration_seconds IS NOT NULL
		 ORDER BY te.started_at DESC`,
		projectID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list project time entries: %w", err)
	}
	defer rows.Close()

	var entries []models.ProjectTimeEntry
	for rows.Next() {
		var e models.ProjectTimeEntry
		if scanErr := rows.Scan(
			&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
			&e.DurationSeconds, &e.Description, &e.IsManual,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName, &e.TaskTitle,
		); scanErr != nil {
			return nil, fmt.Errorf("scan project time entry: %w", scanErr)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// AggregateProjectHours sums completed time-entry hours per member per
// period bucket (trunc is "week" or "month"), for the project's
// team-utilization roll-up. since bounds the window so a long-lived project
// doesn't force scanning every entry ever logged. Bucketing happens via
// date_trunc in SQL -- callers must not re-sum the raw rows in Go. Truncates
// against UTC explicitly (not the session timezone) so bucket boundaries
// match the Go-side period keys the caller zero-fills against.
func (r *PostgresRepository) AggregateProjectHours(ctx context.Context, projectID, tenantID uuid.UUID, trunc string, since time.Time) ([]models.UtilizationBucket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT te.user_id,
		        date_trunc($1, te.started_at AT TIME ZONE 'UTC') AS period,
		        SUM(te.duration_seconds) AS total_seconds
		 FROM time_entries te
		 JOIN tasks t ON t.id = te.task_id AND t.tenant_id = te.tenant_id
		 JOIN projects p ON p.id = t.project_id AND p.tenant_id = te.tenant_id
		 WHERE p.id = $2 AND p.tenant_id = $3 AND te.tenant_id = $3
		   AND te.ended_at IS NOT NULL
		   AND te.duration_seconds IS NOT NULL
		   AND te.started_at >= $4
		 GROUP BY te.user_id, period`,
		trunc, projectID, tenantID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate project hours: %w", err)
	}
	defer rows.Close()

	var buckets []models.UtilizationBucket
	for rows.Next() {
		var b models.UtilizationBucket
		if scanErr := rows.Scan(&b.UserID, &b.PeriodStart, &b.TotalSeconds); scanErr != nil {
			return nil, fmt.Errorf("scan utilization bucket: %w", scanErr)
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (r *PostgresRepository) GetActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.ActiveTimer, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT te.id, te.tenant_id, te.task_id, te.user_id, te.started_at, te.ended_at,
		        te.duration_seconds, te.description, te.is_manual,
		        te.created_at, te.updated_at,
		        t.title AS task_title
		 FROM time_entries te
		 JOIN tasks t ON t.id = te.task_id
		 WHERE te.user_id = $1 AND te.tenant_id = $2 AND te.ended_at IS NULL
		 LIMIT 1`, userID, tenantID,
	)

	var a models.ActiveTimer
	err := row.Scan(
		&a.ID, &a.TenantID, &a.TaskID, &a.UserID, &a.StartedAt, &a.EndedAt,
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

func (r *PostgresRepository) StopActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.TimeEntry, error) {
	now := time.Now()
	row := r.pool.QueryRow(ctx,
		`UPDATE time_entries
		 SET ended_at = $3,
		     duration_seconds = EXTRACT(EPOCH FROM ($3 - started_at))::INTEGER,
		     updated_at = $3
		 WHERE user_id = $1 AND tenant_id = $2 AND ended_at IS NULL
		 RETURNING id, tenant_id, task_id, user_id, started_at, ended_at, duration_seconds,
		           description, is_manual, created_at, updated_at`,
		userID, tenantID, now,
	)

	var e models.TimeEntry
	err := row.Scan(
		&e.ID, &e.TenantID, &e.TaskID, &e.UserID, &e.StartedAt, &e.EndedAt,
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

func (r *PostgresRepository) GetTaskTimeSummary(ctx context.Context, taskID, tenantID uuid.UUID) (*models.TimeEntrySummary, error) {
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
		 WHERE task_id = $1 AND tenant_id = $2`, taskID, tenantID,
	)

	s := &models.TimeEntrySummary{TaskID: taskID}
	if err := row.Scan(&s.TotalDurationSeconds, &s.EntryCount); err != nil {
		return nil, fmt.Errorf("get task time summary: %w", err)
	}
	return s, nil
}
