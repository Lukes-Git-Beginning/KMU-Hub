package employee

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Employee Repository
// ============================================================================

// PostgresEmployeeRepo implements EmployeeRepository using PostgreSQL.
type PostgresEmployeeRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresEmployeeRepo creates a new PostgreSQL employee repository.
func NewPostgresEmployeeRepo(pool *pgxpool.Pool) *PostgresEmployeeRepo {
	return &PostgresEmployeeRepo{pool: pool}
}

func (r *PostgresEmployeeRepo) Create(ctx context.Context, profile *models.EmployeeProfile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_employee_profiles (
			id, user_id, department, position_title, contract_type,
			work_days_per_week, annual_leave_days, manager_user_id, start_date,
			emergency_contact_name, emergency_contact_phone,
			address_street, address_city, address_postal_code, address_country,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		profile.ID, profile.UserID, profile.Department, profile.PositionTitle, profile.ContractType,
		profile.WorkDaysPerWeek, profile.AnnualLeaveDays, profile.ManagerUserID, profile.StartDate,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressPostalCode, profile.AddressCountry,
		profile.CreatedAt, profile.UpdatedAt,
	)
	return err
}

func (r *PostgresEmployeeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmployeeProfile, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT ep.id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at,
			COALESCE(u.display_name, u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(mu.display_name, mu.email, '') AS manager_name
		FROM hr_employee_profiles ep
		LEFT JOIN users u ON ep.user_id = u.id
		LEFT JOIN users mu ON ep.manager_user_id = mu.id
		WHERE ep.id = $1`,
		id,
	)
	return scanEmployeeProfile(row)
}

func (r *PostgresEmployeeRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmployeeProfile, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT ep.id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at,
			COALESCE(u.display_name, u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(mu.display_name, mu.email, '') AS manager_name
		FROM hr_employee_profiles ep
		LEFT JOIN users u ON ep.user_id = u.id
		LEFT JOIN users mu ON ep.manager_user_id = mu.id
		WHERE ep.user_id = $1`,
		userID,
	)
	return scanEmployeeProfile(row)
}

func (r *PostgresEmployeeRepo) List(ctx context.Context, filter EmployeeFilter) ([]*models.EmployeeProfile, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	// Always filter by tenant: employee profiles don't have tenant_id directly,
	// but we can use it in future or JOIN via users table.
	// For now, list all accessible profiles.

	if filter.Department != "" {
		conditions = append(conditions, fmt.Sprintf("ep.department = $%d", argNum))
		args = append(args, filter.Department)
		argNum++
	}

	if filter.ManagerUserID != nil {
		conditions = append(conditions, fmt.Sprintf("ep.manager_user_id = $%d", argNum))
		args = append(args, *filter.ManagerUserID)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_employee_profiles ep %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Pagination defaults
	perPage := filter.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT ep.id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at,
			COALESCE(u.display_name, u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(mu.display_name, mu.email, '') AS manager_name
		FROM hr_employee_profiles ep
		LEFT JOIN users u ON ep.user_id = u.id
		LEFT JOIN users mu ON ep.manager_user_id = mu.id
		%s
		ORDER BY user_name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*models.EmployeeProfile
	for rows.Next() {
		p, scanErr := scanEmployeeProfileFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		results = append(results, p)
	}

	return results, total, rows.Err()
}

func (r *PostgresEmployeeRepo) Update(ctx context.Context, profile *models.EmployeeProfile) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_employee_profiles SET
			department = $1, position_title = $2, contract_type = $3,
			work_days_per_week = $4, annual_leave_days = $5, manager_user_id = $6,
			start_date = $7,
			emergency_contact_name = $8, emergency_contact_phone = $9,
			address_street = $10, address_city = $11, address_postal_code = $12, address_country = $13,
			updated_at = $14
		WHERE id = $15`,
		profile.Department, profile.PositionTitle, profile.ContractType,
		profile.WorkDaysPerWeek, profile.AnnualLeaveDays, profile.ManagerUserID,
		profile.StartDate,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressPostalCode, profile.AddressCountry,
		profile.UpdatedAt, profile.ID,
	)
	return err
}

// ============================================================================
// Document Category Repository
// ============================================================================

// PostgresDocCategoryRepo implements DocumentCategoryRepository using PostgreSQL.
type PostgresDocCategoryRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresDocCategoryRepo creates a new PostgreSQL document category repository.
func NewPostgresDocCategoryRepo(pool *pgxpool.Pool) *PostgresDocCategoryRepo {
	return &PostgresDocCategoryRepo{pool: pool}
}

func (r *PostgresDocCategoryRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.HRDocumentCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key, visibility, is_system, sort_order, created_at
		FROM hr_document_categories
		WHERE tenant_id = $1
		ORDER BY sort_order, name`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.HRDocumentCategory
	for rows.Next() {
		cat := &models.HRDocumentCategory{}
		if scanErr := rows.Scan(
			&cat.ID, &cat.TenantID, &cat.Name, &cat.Key,
			&cat.Visibility, &cat.IsSystem, &cat.SortOrder, &cat.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, cat)
	}
	return results, rows.Err()
}

func (r *PostgresDocCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.HRDocumentCategory, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key, visibility, is_system, sort_order, created_at
		FROM hr_document_categories
		WHERE id = $1`,
		id,
	)
	var cat models.HRDocumentCategory
	err := row.Scan(
		&cat.ID, &cat.TenantID, &cat.Name, &cat.Key,
		&cat.Visibility, &cat.IsSystem, &cat.SortOrder, &cat.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDocumentCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

// ============================================================================
// Employee Document Repository
// ============================================================================

// PostgresEmployeeDocRepo implements EmployeeDocumentRepository using PostgreSQL.
type PostgresEmployeeDocRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresEmployeeDocRepo creates a new PostgreSQL employee document repository.
func NewPostgresEmployeeDocRepo(pool *pgxpool.Pool) *PostgresEmployeeDocRepo {
	return &PostgresEmployeeDocRepo{pool: pool}
}

func (r *PostgresEmployeeDocRepo) Create(ctx context.Context, doc *models.EmployeeDocument) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_employee_documents (id, tenant_id, employee_id, category_id, file_id, uploaded_by, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		doc.ID, doc.TenantID, doc.EmployeeID, doc.CategoryID, doc.FileID, doc.UploadedBy, doc.Notes, doc.CreatedAt,
	)
	return err
}

func (r *PostgresEmployeeDocRepo) ListByEmployee(ctx context.Context, employeeID uuid.UUID, callerRole string) ([]*models.EmployeeDocument, error) {
	// Build visibility filter based on caller role
	var visibilityFilter string
	switch callerRole {
	case "admin", "hr":
		// Admin and HR see all categories
		visibilityFilter = "" // No filter
	case "manager":
		// Manager sees manager and employee visibility
		visibilityFilter = "AND dc.visibility IN ('manager', 'employee')"
	default:
		// Employee (self) only sees employee visibility
		visibilityFilter = "AND dc.visibility = 'employee'"
	}

	query := fmt.Sprintf(`
		SELECT d.id, d.tenant_id, d.employee_id, d.category_id, d.file_id, d.uploaded_by, d.notes, d.created_at,
			COALESCE(dc.name, '') AS category_name,
			'' AS file_name,
			'' AS file_size,
			COALESCE(u.display_name, u.email, '') AS uploaded_by_name
		FROM hr_employee_documents d
		LEFT JOIN hr_document_categories dc ON d.category_id = dc.id
		LEFT JOIN users u ON d.uploaded_by = u.id
		WHERE d.employee_id = $1 %s
		ORDER BY d.created_at DESC
	`, visibilityFilter)

	rows, err := r.pool.Query(ctx, query, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.EmployeeDocument
	for rows.Next() {
		doc := &models.EmployeeDocument{}
		if scanErr := rows.Scan(
			&doc.ID, &doc.TenantID, &doc.EmployeeID, &doc.CategoryID,
			&doc.FileID, &doc.UploadedBy, &doc.Notes, &doc.CreatedAt,
			&doc.CategoryName, &doc.FileName, &doc.FileSize, &doc.UploadedByName,
		); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, doc)
	}
	return results, rows.Err()
}

func (r *PostgresEmployeeDocRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM hr_employee_documents WHERE id = $1`, id,
	)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

func scanEmployeeProfile(row pgx.Row) (*models.EmployeeProfile, error) {
	var p models.EmployeeProfile
	err := row.Scan(
		&p.ID, &p.UserID, &p.Department, &p.PositionTitle, &p.ContractType,
		&p.WorkDaysPerWeek, &p.AnnualLeaveDays, &p.ManagerUserID, &p.StartDate,
		&p.EmergencyContactName, &p.EmergencyContactPhone,
		&p.AddressStreet, &p.AddressCity, &p.AddressPostalCode, &p.AddressCountry,
		&p.CreatedAt, &p.UpdatedAt,
		&p.UserName, &p.UserEmail, &p.ManagerName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEmployeeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanEmployeeProfileFromRows(rows pgx.Rows) (*models.EmployeeProfile, error) {
	var p models.EmployeeProfile
	err := rows.Scan(
		&p.ID, &p.UserID, &p.Department, &p.PositionTitle, &p.ContractType,
		&p.WorkDaysPerWeek, &p.AnnualLeaveDays, &p.ManagerUserID, &p.StartDate,
		&p.EmergencyContactName, &p.EmergencyContactPhone,
		&p.AddressStreet, &p.AddressCity, &p.AddressPostalCode, &p.AddressCountry,
		&p.CreatedAt, &p.UpdatedAt,
		&p.UserName, &p.UserEmail, &p.ManagerName,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
