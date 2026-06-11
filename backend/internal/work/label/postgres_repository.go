package label

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL label repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, tenantID, name, color string) (*Label, error) {
	l := &Label{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO work_labels (tenant_id, name, color, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $4)
		 RETURNING id, tenant_id, name, color, created_at, updated_at`,
		tenantID, name, color, time.Now(),
	).Scan(&l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	return l, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id string) (*Label, error) {
	l := &Label{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, color, created_at, updated_at
		 FROM work_labels
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return l, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string) ([]*Label, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, color, created_at, updated_at
		 FROM work_labels
		 WHERE tenant_id = $1
		 ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []*Label
	for rows.Next() {
		l := &Label{}
		if scanErr := rows.Scan(&l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, tenantID, id, name, color string) (*Label, error) {
	l := &Label{}
	err := r.pool.QueryRow(ctx,
		`UPDATE work_labels
		 SET name = $3, color = $4, updated_at = $5
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, name, color, created_at, updated_at`,
		id, tenantID, name, color, time.Now(),
	).Scan(&l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isDuplicateError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	return l, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM work_labels WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetLabelsByTaskIDs(ctx context.Context, tenantID string, taskIDs []string) (map[string][]*Label, error) {
	if len(taskIDs) == 0 {
		return make(map[string][]*Label), nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT tl.task_id, wl.id, wl.tenant_id, wl.name, wl.color, wl.created_at, wl.updated_at
		 FROM task_labels tl
		 JOIN work_labels wl ON wl.id = tl.label_id
		 WHERE tl.task_id = ANY($1)
		   AND wl.tenant_id = $2
		 ORDER BY tl.task_id, wl.name`,
		taskIDs, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*Label)
	for rows.Next() {
		var taskID string
		l := &Label{}
		if scanErr := rows.Scan(&taskID, &l.ID, &l.TenantID, &l.Name, &l.Color, &l.CreatedAt, &l.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		result[taskID] = append(result[taskID], l)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) SetTaskLabels(ctx context.Context, tenantID, taskID string, labelIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove all current assignments for this task
	if _, err := tx.Exec(ctx,
		`DELETE FROM task_labels
		 WHERE task_id = $1
		   AND label_id IN (
		       SELECT id FROM work_labels WHERE tenant_id = $2
		   )`,
		taskID, tenantID,
	); err != nil {
		return err
	}

	// Insert new assignments
	for _, labelID := range labelIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO task_labels (task_id, label_id)
			 SELECT $1, $2
			 WHERE EXISTS (
			     SELECT 1 FROM work_labels
			     WHERE id = $2 AND tenant_id = $3
			 )
			 ON CONFLICT DO NOTHING`,
			taskID, labelID, tenantID,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetTaskLabelIDs(ctx context.Context, tenantID, taskID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tl.label_id
		 FROM task_labels tl
		 JOIN work_labels wl ON wl.id = tl.label_id
		 WHERE tl.task_id = $1
		   AND wl.tenant_id = $2
		 ORDER BY wl.name`,
		taskID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// isDuplicateError checks whether a pgx error is a unique-constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps pg error codes; the unique_violation code is "23505"
	return containsCode(err, "23505")
}

func containsCode(err error, code string) bool {
	type pgErr interface {
		SQLState() string
	}
	var pe pgErr
	if errors.As(err, &pe) {
		return pe.SQLState() == code
	}
	return false
}
