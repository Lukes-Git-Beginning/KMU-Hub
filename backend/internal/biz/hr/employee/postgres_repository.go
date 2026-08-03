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
			id, tenant_id, user_id, department, position_title, contract_type,
			work_days_per_week, annual_leave_days, manager_user_id, start_date,
			emergency_contact_name, emergency_contact_phone,
			address_street, address_city, address_postal_code, address_country,
			created_at, updated_at, hourly_rate
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		profile.ID, profile.TenantID, profile.UserID, profile.Department, profile.PositionTitle, profile.ContractType,
		profile.WorkDaysPerWeek, profile.AnnualLeaveDays, profile.ManagerUserID, profile.StartDate,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressPostalCode, profile.AddressCountry,
		profile.CreatedAt, profile.UpdatedAt, profile.HourlyRate,
	)
	return err
}

func (r *PostgresEmployeeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmployeeProfile, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT ep.id, ep.tenant_id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at, ep.hourly_rate,
			ep.status, ep.last_work_day, ep.exit_date, COALESCE(ep.exit_type, ''), COALESCE(ep.exit_reason, ''),
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(NULLIF(CONCAT_WS(' ', mu.first_name, mu.last_name), ''), mu.email, '') AS manager_name
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
		`SELECT ep.id, ep.tenant_id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at, ep.hourly_rate,
			ep.status, ep.last_work_day, ep.exit_date, COALESCE(ep.exit_type, ''), COALESCE(ep.exit_reason, ''),
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(NULLIF(CONCAT_WS(' ', mu.first_name, mu.last_name), ''), mu.email, '') AS manager_name
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

	if filter.TenantID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("ep.tenant_id = $%d", argNum))
		args = append(args, filter.TenantID)
		argNum++
	}

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
		SELECT ep.id, ep.tenant_id, ep.user_id, ep.department, ep.position_title, ep.contract_type,
			ep.work_days_per_week, ep.annual_leave_days, ep.manager_user_id, ep.start_date,
			ep.emergency_contact_name, ep.emergency_contact_phone,
			ep.address_street, ep.address_city, ep.address_postal_code, ep.address_country,
			ep.created_at, ep.updated_at, ep.hourly_rate,
			ep.status, ep.last_work_day, ep.exit_date, COALESCE(ep.exit_type, ''), COALESCE(ep.exit_reason, ''),
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS user_name,
			COALESCE(u.email, '') AS user_email,
			COALESCE(NULLIF(CONCAT_WS(' ', mu.first_name, mu.last_name), ''), mu.email, '') AS manager_name
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
			hourly_rate = $14,
			updated_at = $15
		WHERE id = $16 AND tenant_id = $17`,
		profile.Department, profile.PositionTitle, profile.ContractType,
		profile.WorkDaysPerWeek, profile.AnnualLeaveDays, profile.ManagerUserID,
		profile.StartDate,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressPostalCode, profile.AddressCountry,
		profile.HourlyRate,
		profile.UpdatedAt, profile.ID, profile.TenantID,
	)
	return err
}

// CountOtherActiveRoleAdmins counts the tenant's active accounts that still
// carry role administration, ignoring one user. Mirrors
// auth.PostgresRepository.CountActiveRoleAdminsExcludingUser — the same query
// behind the same guard, kept here because HR runs the offboard transaction and
// there is no service-to-service gRPC in this repository.
//
// user_roles carries its own tenant_id/RLS since migration 000286, but this
// join on users predates it and stays as the same defense-in-depth every
// other user_roles query in auth/postgres_repository.go keeps, so it must
// not be dropped.
func (r *PostgresEmployeeRepo) CountOtherActiveRoleAdmins(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT ur.user_id)
		   FROM user_roles ur
		   JOIN users u ON u.id = ur.user_id AND u.is_active
		   JOIN role_permissions rp ON rp.role_id = ur.role_id
		   JOIN permissions p ON p.id = rp.permission_id
		  WHERE p.name = ANY($1) AND ur.user_id <> $2`,
		roleAdminKeys, userID,
	).Scan(&count)
	return count, err
}

// CountDirectReports returns how many active employees report to userID.
func (r *PostgresEmployeeRepo) CountDirectReports(ctx context.Context, tenantID, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM hr_employee_profiles
		  WHERE tenant_id = $1 AND manager_user_id = $2 AND status = 'active'`,
		tenantID, userID,
	).Scan(&count)
	return count, err
}

// Offboard performs the whole exit cascade in one transaction: the personnel
// record goes inactive, the account loses its login and its roles, and the
// direct reports move to the successor.
//
// One transaction rather than a sequence of service calls: users,
// user_roles and hr_employee_profiles live in the same database, so the
// distributed-transaction problem the split into services would normally create
// does not arise here. A partial offboard — a personnel file marked inactive
// behind an account that can still log in — is the failure nobody notices.
//
// The seat comes back on its own: auth counts seats as active users
// (CountSeatsInUse), so is_active = false releases it without a separate step.
func (r *PostgresEmployeeRepo) Offboard(ctx context.Context, in OffboardWrite) (*models.EmployeeProfile, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The condition on status is the concurrency guard: a second offboard
	// arriving at the same time finds no row and gets ErrAlreadyOffboarded
	// instead of overwriting the first one's exit data.
	tag, err := tx.Exec(ctx,
		`UPDATE hr_employee_profiles
		    SET status = 'inactive', last_work_day = $1, exit_date = $2,
		        exit_type = $3, exit_reason = $4, updated_at = NOW()
		  WHERE id = $5 AND tenant_id = $6 AND status = 'active'`,
		in.LastWorkDay, in.ExitDate, in.ExitType, in.ExitReason, in.EmployeeID, in.TenantID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadyOffboarded
	}

	// Locking the login first: everything below can still fail and roll the
	// whole thing back, but no ordering inside the transaction can leave the
	// account reachable while the file says the person is gone.
	if _, err := tx.Exec(ctx,
		`UPDATE users SET is_active = false, updated_at = NOW()
		  WHERE id = $1 AND tenant_id = $2`,
		in.UserID, in.TenantID,
	); err != nil {
		return nil, err
	}

	// The subquery on users, not the session, is what pins the tenant here —
	// this transaction runs under the caller's own context, but naming the
	// tenant explicitly keeps this delete correct independent of how ctx got
	// built, the same defense-in-depth the auth package keeps for user_roles.
	if _, err := tx.Exec(ctx,
		`DELETE FROM user_roles
		  WHERE user_id = (SELECT id FROM users WHERE id = $1 AND tenant_id = $2)`,
		in.UserID, in.TenantID,
	); err != nil {
		return nil, err
	}

	if in.SuccessorUserID != nil {
		if err := reassignReports(ctx, tx, in); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, in.EmployeeID)
}

// reassignReports moves the leaver's direct reports to the successor without
// creating a management cycle.
//
// The successor may sit anywhere below the leaver. Handing every report to them
// unconditionally would make the successor their own manager (successor is a
// direct report) or close a longer loop (successor is a grandchild: their
// manager gets moved under them). Everyone on the chain from the successor
// upwards therefore inherits the leaver's own manager instead — which is what a
// promotion actually means — and everybody else gets the successor.
func reassignReports(ctx context.Context, tx pgx.Tx, in OffboardWrite) error {
	_, err := tx.Exec(ctx,
		`WITH RECURSIVE chain AS (
		     SELECT user_id, manager_user_id, 1 AS depth
		       FROM hr_employee_profiles
		      WHERE tenant_id = $1 AND user_id = $2
		     UNION ALL
		     SELECT p.user_id, p.manager_user_id, c.depth + 1
		       FROM hr_employee_profiles p
		       JOIN chain c ON p.user_id = c.manager_user_id
		      WHERE p.tenant_id = $1 AND c.depth < 64
		 )
		 UPDATE hr_employee_profiles
		    SET manager_user_id = CASE
		            WHEN user_id IN (SELECT user_id FROM chain) THEN $3
		            ELSE $2
		        END,
		        updated_at = NOW()
		  WHERE tenant_id = $1 AND manager_user_id = $4`,
		in.TenantID, *in.SuccessorUserID, in.LeaverManagerUserID, in.UserID,
	)
	return err
}

// roleAdminKeys are the capability keys that make an account able to hand out
// roles. Copy of auth.roleAdminKeys (unexported there); if that list grows,
// this one has to grow with it.
var roleAdminKeys = []string{"roles:manage", "admin:role:assign"}

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

// systemSeedTenantID is the placeholder tenant the system document categories
// carry (migration 000046). The RLS policy from migration 000123 explicitly
// permits reading those rows from any tenant, and they are the only categories
// that exist until a tenant defines its own — a query restricted to the real
// tenant returns nothing and leaves the upload dialog without any category.
var systemSeedTenantID = uuid.UUID{}

func (r *PostgresDocCategoryRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.HRDocumentCategory, error) {
	// lean: system seeds and tenant rows are returned side by side; a tenant row
	// does not supersede the system row carrying the same key. Add that
	// precedence once categories become creatable per tenant (no write path today).
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key, visibility, is_system, sort_order, created_at
		FROM hr_document_categories
		WHERE tenant_id IN ($1, $2)
		ORDER BY sort_order, name`,
		tenantID, systemSeedTenantID,
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

func (r *PostgresDocCategoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRDocumentCategory, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key, visibility, is_system, sort_order, created_at
		FROM hr_document_categories
		WHERE id = $1 AND tenant_id IN ($2, $3)`,
		id, tenantID, systemSeedTenantID,
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
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS uploaded_by_name
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

func (r *PostgresEmployeeDocRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM hr_employee_documents WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

func scanEmployeeProfile(row pgx.Row) (*models.EmployeeProfile, error) {
	var p models.EmployeeProfile
	err := row.Scan(
		&p.ID, &p.TenantID, &p.UserID, &p.Department, &p.PositionTitle, &p.ContractType,
		&p.WorkDaysPerWeek, &p.AnnualLeaveDays, &p.ManagerUserID, &p.StartDate,
		&p.EmergencyContactName, &p.EmergencyContactPhone,
		&p.AddressStreet, &p.AddressCity, &p.AddressPostalCode, &p.AddressCountry,
		&p.CreatedAt, &p.UpdatedAt, &p.HourlyRate,
		&p.Status, &p.LastWorkDay, &p.ExitDate, &p.ExitType, &p.ExitReason,
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
		&p.ID, &p.TenantID, &p.UserID, &p.Department, &p.PositionTitle, &p.ContractType,
		&p.WorkDaysPerWeek, &p.AnnualLeaveDays, &p.ManagerUserID, &p.StartDate,
		&p.EmergencyContactName, &p.EmergencyContactPhone,
		&p.AddressStreet, &p.AddressCity, &p.AddressPostalCode, &p.AddressCountry,
		&p.CreatedAt, &p.UpdatedAt, &p.HourlyRate,
		&p.Status, &p.LastWorkDay, &p.ExitDate, &p.ExitType, &p.ExitReason,
		&p.UserName, &p.UserEmail, &p.ManagerName,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
