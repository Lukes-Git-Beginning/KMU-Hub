package timetracking

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Time Category Repository
// ============================================================================

// PostgresTimeCategoryRepo implements TimeCategoryRepository.
type PostgresTimeCategoryRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresTimeCategoryRepo creates a new category repository.
func NewPostgresTimeCategoryRepo(pool *pgxpool.Pool) *PostgresTimeCategoryRepo {
	return &PostgresTimeCategoryRepo{pool: pool}
}

func (r *PostgresTimeCategoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, color, icon, is_default, created_at, updated_at
		FROM hr_time_categories
		WHERE tenant_id = $1
		ORDER BY is_default DESC, name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*models.HRTimeCategory
	for rows.Next() {
		c := &models.HRTimeCategory{}
		if scanErr := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Color, &c.Icon, &c.IsDefault,
			&c.CreatedAt, &c.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []*models.HRTimeCategory{}
	}
	return cats, rows.Err()
}

func (r *PostgresTimeCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.HRTimeCategory, error) {
	c := &models.HRTimeCategory{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, color, icon, is_default, created_at, updated_at
		FROM hr_time_categories WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.TenantID, &c.Name, &c.Color, &c.Icon, &c.IsDefault, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTimeCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *PostgresTimeCategoryRepo) Create(ctx context.Context, cat *models.HRTimeCategory) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_time_categories (id, tenant_id, name, color, icon, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		cat.ID, cat.TenantID, cat.Name, cat.Color, cat.Icon, cat.IsDefault, cat.CreatedAt, cat.UpdatedAt,
	)
	return err
}

func (r *PostgresTimeCategoryRepo) Update(ctx context.Context, cat *models.HRTimeCategory) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_time_categories
		SET name=$2, color=$3, icon=$4, is_default=$5, updated_at=$6
		WHERE id=$1`,
		cat.ID, cat.Name, cat.Color, cat.Icon, cat.IsDefault, cat.UpdatedAt,
	)
	return err
}

func (r *PostgresTimeCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM hr_time_categories WHERE id=$1`, id)
	return err
}

// ============================================================================
// Time Template Repository
// ============================================================================

// PostgresTimeTemplateRepo implements TimeTemplateRepository.
type PostgresTimeTemplateRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresTimeTemplateRepo creates a new template repository.
func NewPostgresTimeTemplateRepo(pool *pgxpool.Pool) *PostgresTimeTemplateRepo {
	return &PostgresTimeTemplateRepo{pool: pool}
}

func (r *PostgresTimeTemplateRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, category_id, description, estimated_minutes, created_at, updated_at
		FROM hr_time_templates
		WHERE tenant_id = $1
		ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.HRTimeTemplate
	for rows.Next() {
		t := &models.HRTimeTemplate{}
		if scanErr := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.CategoryID, &t.Description,
			&t.EstimatedMinutes, &t.CreatedAt, &t.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []*models.HRTimeTemplate{}
	}
	return templates, rows.Err()
}

func (r *PostgresTimeTemplateRepo) Create(ctx context.Context, t *models.HRTimeTemplate) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_time_templates
		(id, tenant_id, name, category_id, description, estimated_minutes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.TenantID, t.Name, t.CategoryID, t.Description, t.EstimatedMinutes, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *PostgresTimeTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM hr_time_templates WHERE id=$1`, id)
	return err
}

// ============================================================================
// Time Project Repository
// ============================================================================

// PostgresTimeProjectRepo implements TimeProjectRepository.
type PostgresTimeProjectRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresTimeProjectRepo creates a new project repository.
func NewPostgresTimeProjectRepo(pool *pgxpool.Pool) *PostgresTimeProjectRepo {
	return &PostgresTimeProjectRepo{pool: pool}
}

func (r *PostgresTimeProjectRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeProject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, customer_name, color, billable_default, created_at, updated_at
		FROM hr_time_projects
		WHERE tenant_id = $1
		ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.HRTimeProject
	for rows.Next() {
		p := &models.HRTimeProject{}
		if scanErr := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.CustomerName, &p.Color,
			&p.BillableDefault, &p.CreatedAt, &p.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		projects = append(projects, p)
	}
	if projects == nil {
		projects = []*models.HRTimeProject{}
	}
	return projects, rows.Err()
}

func (r *PostgresTimeProjectRepo) Create(ctx context.Context, p *models.HRTimeProject) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_time_projects
		(id, tenant_id, name, customer_name, color, billable_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.TenantID, p.Name, p.CustomerName, p.Color, p.BillableDefault, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// ============================================================================
// Week Approval Repository
// ============================================================================

// PostgresWeekApprovalRepo implements WeekApprovalRepository.
type PostgresWeekApprovalRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresWeekApprovalRepo creates a new week approval repository.
func NewPostgresWeekApprovalRepo(pool *pgxpool.Pool) *PostgresWeekApprovalRepo {
	return &PostgresWeekApprovalRepo{pool: pool}
}

func (r *PostgresWeekApprovalRepo) GetByEmployeeWeek(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, error) {
	w := &models.HRWeekApproval{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, employee_id, week_start, status,
			submitted_at, approved_by, approved_at, rejection_reason,
			created_at, updated_at
		FROM hr_week_approvals
		WHERE tenant_id=$1 AND employee_id=$2 AND week_start=$3`,
		tenantID, employeeID, weekStart,
	).Scan(
		&w.ID, &w.TenantID, &w.EmployeeID, &w.WeekStart, &w.Status,
		&w.SubmittedAt, &w.ApprovedBy, &w.ApprovedAt, &w.RejectionReason,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWeekApprovalNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *PostgresWeekApprovalRepo) Upsert(ctx context.Context, w *models.HRWeekApproval) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_week_approvals
		(id, tenant_id, employee_id, week_start, status,
		 submitted_at, approved_by, approved_at, rejection_reason,
		 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, employee_id, week_start) DO UPDATE SET
			status = EXCLUDED.status,
			submitted_at = EXCLUDED.submitted_at,
			approved_by = EXCLUDED.approved_by,
			approved_at = EXCLUDED.approved_at,
			rejection_reason = EXCLUDED.rejection_reason,
			updated_at = EXCLUDED.updated_at`,
		w.ID, w.TenantID, w.EmployeeID, w.WeekStart, w.Status,
		w.SubmittedAt, w.ApprovedBy, w.ApprovedAt, w.RejectionReason,
		w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (r *PostgresWeekApprovalRepo) ListByWeek(ctx context.Context, tenantID uuid.UUID, weekStart time.Time) ([]*models.HRWeekApproval, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, employee_id, week_start, status,
			submitted_at, approved_by, approved_at, rejection_reason,
			created_at, updated_at
		FROM hr_week_approvals
		WHERE tenant_id=$1 AND week_start=$2
		ORDER BY employee_id ASC`,
		tenantID, weekStart,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.HRWeekApproval
	for rows.Next() {
		w := &models.HRWeekApproval{}
		if scanErr := rows.Scan(
			&w.ID, &w.TenantID, &w.EmployeeID, &w.WeekStart, &w.Status,
			&w.SubmittedAt, &w.ApprovedBy, &w.ApprovedAt, &w.RejectionReason,
			&w.CreatedAt, &w.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, w)
	}
	if results == nil {
		results = []*models.HRWeekApproval{}
	}
	return results, rows.Err()
}

