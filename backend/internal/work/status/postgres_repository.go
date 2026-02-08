package status

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

func (r *PostgresRepository) Create(ctx context.Context, status *models.ProjectStatus) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_statuses (id, project_id, name, color, sort_order, is_default, is_closed, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		status.ID, status.ProjectID, status.Name, status.Color,
		status.SortOrder, status.IsDefault, status.IsClosed, status.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ProjectStatus, error) {
	var s models.ProjectStatus
	err := r.pool.QueryRow(ctx,
		`SELECT id, project_id, name, color, sort_order, is_default, is_closed, created_at
		 FROM project_statuses WHERE id = $1`, id,
	).Scan(&s.ID, &s.ProjectID, &s.Name, &s.Color, &s.SortOrder, &s.IsDefault, &s.IsClosed, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectStatus, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, name, color, sort_order, is_default, is_closed, created_at
		 FROM project_statuses WHERE project_id = $1 ORDER BY sort_order ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.ProjectStatus
	for rows.Next() {
		var s models.ProjectStatus
		if scanErr := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Color, &s.SortOrder, &s.IsDefault, &s.IsClosed, &s.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, status *models.ProjectStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE project_statuses SET name = $1, color = $2, is_default = $3, is_closed = $4
		 WHERE id = $5`,
		status.Name, status.Color, status.IsDefault, status.IsClosed, status.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM project_statuses WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) Reorder(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for i, id := range statusIDs {
		_, execErr := tx.Exec(ctx,
			`UPDATE project_statuses SET sort_order = $1 WHERE id = $2 AND project_id = $3`,
			i, id, projectID,
		)
		if execErr != nil {
			return execErr
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) CountTasksWithStatus(ctx context.Context, statusID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks WHERE status_id = $1`, statusID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) HasDefault(ctx context.Context, projectID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_statuses WHERE project_id = $1 AND is_default = true)`,
		projectID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetNextSortOrder(ctx context.Context, projectID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM project_statuses WHERE project_id = $1`,
		projectID,
	).Scan(&maxOrder)
	if err != nil {
		return 0, err
	}
	return maxOrder + 1, nil
}

func (r *PostgresRepository) NameExistsInProject(ctx context.Context, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var err error
	if excludeID != nil {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM project_statuses WHERE project_id = $1 AND LOWER(name) = LOWER($2) AND id != $3)`,
			projectID, name, *excludeID,
		).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM project_statuses WHERE project_id = $1 AND LOWER(name) = LOWER($2))`,
			projectID, name,
		).Scan(&exists)
	}
	return exists, err
}

func (r *PostgresRepository) CopyStatusesForProject(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_statuses (id, project_id, name, color, sort_order, is_default, is_closed, created_at)
		 SELECT uuid_generate_v4(), $2, name, color, sort_order, is_default, is_closed, $3
		 FROM project_statuses WHERE project_id = $1 ORDER BY sort_order ASC`,
		sourceProjectID, targetProjectID, time.Now(),
	)
	return err
}
