package leave

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Leave Request Repository
// ============================================================================

// PostgresLeaveRequestRepo implements LeaveRequestRepository using PostgreSQL.
type PostgresLeaveRequestRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresLeaveRequestRepo creates a new PostgreSQL leave request repository.
func NewPostgresLeaveRequestRepo(pool *pgxpool.Pool) *PostgresLeaveRequestRepo {
	return &PostgresLeaveRequestRepo{pool: pool}
}

func (r *PostgresLeaveRequestRepo) Create(ctx context.Context, req *models.LeaveRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_leave_requests (
			id, tenant_id, employee_id, leave_type_id,
			start_date, end_date, is_half_day_start, half_day_period_start,
			is_half_day_end, half_day_period_end, total_days, reason,
			status, approved_by, approval_comment, approved_at,
			au_document_required, au_document_file_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18,
			$19, $20
		)`,
		req.ID, req.TenantID, req.EmployeeID, req.LeaveTypeID,
		req.StartDate, req.EndDate, req.IsHalfDayStart, req.HalfDayPeriodStart,
		req.IsHalfDayEnd, req.HalfDayPeriodEnd, req.TotalDays, req.Reason,
		req.Status, req.ApprovedBy, req.ApprovalComment, req.ApprovedAt,
		req.AUDocumentRequired, req.AUDocumentFileID,
		req.CreatedAt, req.UpdatedAt,
	)
	return err
}

func (r *PostgresLeaveRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveRequest, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT lr.id, lr.tenant_id, lr.employee_id, lr.leave_type_id,
			lr.start_date, lr.end_date, lr.is_half_day_start, lr.half_day_period_start,
			lr.is_half_day_end, lr.half_day_period_end, lr.total_days, lr.reason,
			lr.status, lr.approved_by, lr.approval_comment, lr.approved_at,
			lr.au_document_required, lr.au_document_file_id,
			lr.created_at, lr.updated_at,
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS employee_name,
			COALESCE(lt.name, '') AS leave_type_name,
			COALESCE(lt.color, '') AS leave_type_color
		FROM hr_leave_requests lr
		LEFT JOIN users u ON lr.employee_id = u.id
		LEFT JOIN hr_leave_types lt ON lr.leave_type_id = lt.id
		WHERE lr.id = $1`,
		id,
	)
	return scanLeaveRequest(row)
}

func (r *PostgresLeaveRequestRepo) List(ctx context.Context, filter LeaveRequestFilter) ([]*models.LeaveRequest, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("lr.tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

	if filter.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("lr.employee_id = $%d", argNum))
		args = append(args, *filter.EmployeeID)
		argNum++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("lr.status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.StartDateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("lr.start_date >= $%d", argNum))
		args = append(args, *filter.StartDateFrom)
		argNum++
	}

	if filter.StartDateTo != nil {
		conditions = append(conditions, fmt.Sprintf("lr.start_date <= $%d", argNum))
		args = append(args, *filter.StartDateTo)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM hr_leave_requests lr %s`, whereClause,
	)
	var total int
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
		SELECT lr.id, lr.tenant_id, lr.employee_id, lr.leave_type_id,
			lr.start_date, lr.end_date, lr.is_half_day_start, lr.half_day_period_start,
			lr.is_half_day_end, lr.half_day_period_end, lr.total_days, lr.reason,
			lr.status, lr.approved_by, lr.approval_comment, lr.approved_at,
			lr.au_document_required, lr.au_document_file_id,
			lr.created_at, lr.updated_at,
			COALESCE(NULLIF(CONCAT_WS(' ', u.first_name, u.last_name), ''), u.email, '') AS employee_name,
			COALESCE(lt.name, '') AS leave_type_name,
			COALESCE(lt.color, '') AS leave_type_color
		FROM hr_leave_requests lr
		LEFT JOIN users u ON lr.employee_id = u.id
		LEFT JOIN hr_leave_types lt ON lr.leave_type_id = lt.id
		%s
		ORDER BY lr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*models.LeaveRequest
	for rows.Next() {
		req, scanErr := scanLeaveRequestFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		results = append(results, req)
	}

	return results, total, rows.Err()
}

func (r *PostgresLeaveRequestRepo) Update(ctx context.Context, req *models.LeaveRequest) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_leave_requests SET
			status = $1, approved_by = $2, approval_comment = $3, approved_at = $4,
			au_document_required = $5, au_document_file_id = $6, updated_at = $7
		WHERE id = $8`,
		req.Status, req.ApprovedBy, req.ApprovalComment, req.ApprovedAt,
		req.AUDocumentRequired, req.AUDocumentFileID, req.UpdatedAt,
		req.ID,
	)
	return err
}

func (r *PostgresLeaveRequestRepo) FindOverlaps(ctx context.Context, employeeID uuid.UUID, startDate, endDate time.Time, excludeID *uuid.UUID) ([]*models.LeaveRequest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT lr.id, lr.tenant_id, lr.employee_id, lr.leave_type_id,
			lr.start_date, lr.end_date, lr.is_half_day_start, lr.half_day_period_start,
			lr.is_half_day_end, lr.half_day_period_end, lr.total_days, lr.reason,
			lr.status, lr.approved_by, lr.approval_comment, lr.approved_at,
			lr.au_document_required, lr.au_document_file_id,
			lr.created_at, lr.updated_at,
			'' AS employee_name, '' AS leave_type_name, '' AS leave_type_color
		FROM hr_leave_requests lr
		WHERE lr.employee_id = $1
			AND lr.status IN ('pending', 'approved')
			AND daterange(lr.start_date, lr.end_date, '[]') && daterange($2::date, $3::date, '[]')
			AND ($4::uuid IS NULL OR lr.id != $4)
		ORDER BY lr.start_date`,
		employeeID, startDate, endDate, excludeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.LeaveRequest
	for rows.Next() {
		req, scanErr := scanLeaveRequestFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, req)
	}
	return results, rows.Err()
}

// ============================================================================
// Leave Balance Repository
// ============================================================================

// PostgresLeaveBalanceRepo implements LeaveBalanceRepository using PostgreSQL.
type PostgresLeaveBalanceRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresLeaveBalanceRepo creates a new PostgreSQL leave balance repository.
func NewPostgresLeaveBalanceRepo(pool *pgxpool.Pool) *PostgresLeaveBalanceRepo {
	return &PostgresLeaveBalanceRepo{pool: pool}
}

func (r *PostgresLeaveBalanceRepo) GetByEmployeeYear(ctx context.Context, tenantID, employeeID uuid.UUID, year int) (*models.HRLeaveBalance, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, employee_id, year,
			entitlement, carried_over, used, remaining,
			carryover_expires_at, carryover_notified,
			created_at, updated_at
		FROM hr_leave_balances
		WHERE tenant_id = $1 AND employee_id = $2 AND year = $3`,
		tenantID, employeeID, year,
	)
	return scanLeaveBalance(row)
}

func (r *PostgresLeaveBalanceRepo) Upsert(ctx context.Context, balance *models.HRLeaveBalance) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_leave_balances (
			id, tenant_id, employee_id, year,
			entitlement, carried_over, used, remaining,
			carryover_expires_at, carryover_notified,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id, employee_id, year) DO UPDATE SET
			entitlement = EXCLUDED.entitlement,
			carried_over = EXCLUDED.carried_over,
			used = EXCLUDED.used,
			remaining = EXCLUDED.remaining,
			carryover_expires_at = EXCLUDED.carryover_expires_at,
			carryover_notified = EXCLUDED.carryover_notified,
			updated_at = EXCLUDED.updated_at`,
		balance.ID, balance.TenantID, balance.EmployeeID, balance.Year,
		balance.Entitlement, balance.CarriedOver, balance.Used, balance.Remaining,
		balance.CarryoverExpiresAt, balance.CarryoverNotified,
		balance.CreatedAt, balance.UpdatedAt,
	)
	return err
}

// ============================================================================
// Leave Type Repository
// ============================================================================

// PostgresLeaveTypeRepo implements LeaveTypeRepository using PostgreSQL.
type PostgresLeaveTypeRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresLeaveTypeRepo creates a new PostgreSQL leave type repository.
func NewPostgresLeaveTypeRepo(pool *pgxpool.Pool) *PostgresLeaveTypeRepo {
	return &PostgresLeaveTypeRepo{pool: pool}
}

func (r *PostgresLeaveTypeRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.LeaveType, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key, color,
			deducts_from_balance, requires_approval, requires_au_document,
			is_system, sort_order, created_at
		FROM hr_leave_types
		WHERE tenant_id = $1
		ORDER BY sort_order, name`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.LeaveType
	for rows.Next() {
		lt, scanErr := scanLeaveTypeFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, lt)
	}
	return results, rows.Err()
}

func (r *PostgresLeaveTypeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key, color,
			deducts_from_balance, requires_approval, requires_au_document,
			is_system, sort_order, created_at
		FROM hr_leave_types
		WHERE id = $1`,
		id,
	)
	var lt models.LeaveType
	err := row.Scan(
		&lt.ID, &lt.TenantID, &lt.Name, &lt.Key, &lt.Color,
		&lt.DeductsFromBalance, &lt.RequiresApproval, &lt.RequiresAUDocument,
		&lt.IsSystem, &lt.SortOrder, &lt.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLeaveTypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &lt, nil
}

func (r *PostgresLeaveTypeRepo) GetByKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.LeaveType, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key, color,
			deducts_from_balance, requires_approval, requires_au_document,
			is_system, sort_order, created_at
		FROM hr_leave_types
		WHERE tenant_id = $1 AND key = $2`,
		tenantID, key,
	)
	var lt models.LeaveType
	err := row.Scan(
		&lt.ID, &lt.TenantID, &lt.Name, &lt.Key, &lt.Color,
		&lt.DeductsFromBalance, &lt.RequiresApproval, &lt.RequiresAUDocument,
		&lt.IsSystem, &lt.SortOrder, &lt.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLeaveTypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &lt, nil
}

// ============================================================================
// HR Settings Repository
// ============================================================================

// PostgresHRSettingsRepo implements HRSettingsRepository using PostgreSQL.
type PostgresHRSettingsRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresHRSettingsRepo creates a new PostgreSQL HR settings repository.
func NewPostgresHRSettingsRepo(pool *pgxpool.Pool) *PostgresHRSettingsRepo {
	return &PostgresHRSettingsRepo{pool: pool}
}

func (r *PostgresHRSettingsRepo) GetByTenant(ctx context.Context, tenantID uuid.UUID) (*models.HRCompanySettings, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, au_threshold_days, show_absence_reason,
			default_annual_leave_days, timezone,
			work_hours_per_day, max_daily_hours, break_after_hours,
			created_at, updated_at
		FROM hr_company_settings
		WHERE tenant_id = $1`,
		tenantID,
	)
	var s models.HRCompanySettings
	err := row.Scan(
		&s.ID, &s.TenantID, &s.AUThresholdDays, &s.ShowAbsenceReason,
		&s.DefaultAnnualLeaveDays, &s.Timezone,
		&s.WorkHoursPerDay, &s.MaxDailyHours, &s.BreakAfterHours,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSettingsNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresHRSettingsRepo) Upsert(ctx context.Context, settings *models.HRCompanySettings) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_company_settings (
			id, tenant_id, au_threshold_days, show_absence_reason,
			default_annual_leave_days, timezone,
			work_hours_per_day, max_daily_hours, break_after_hours,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id) DO UPDATE SET
			au_threshold_days = EXCLUDED.au_threshold_days,
			show_absence_reason = EXCLUDED.show_absence_reason,
			default_annual_leave_days = EXCLUDED.default_annual_leave_days,
			timezone = EXCLUDED.timezone,
			work_hours_per_day = EXCLUDED.work_hours_per_day,
			max_daily_hours = EXCLUDED.max_daily_hours,
			break_after_hours = EXCLUDED.break_after_hours,
			updated_at = EXCLUDED.updated_at`,
		settings.ID, settings.TenantID, settings.AUThresholdDays, settings.ShowAbsenceReason,
		settings.DefaultAnnualLeaveDays, settings.Timezone,
		settings.WorkHoursPerDay, settings.MaxDailyHours, settings.BreakAfterHours,
		settings.CreatedAt, settings.UpdatedAt,
	)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

func scanLeaveRequest(row pgx.Row) (*models.LeaveRequest, error) {
	var req models.LeaveRequest
	var totalDaysFloat float64
	err := row.Scan(
		&req.ID, &req.TenantID, &req.EmployeeID, &req.LeaveTypeID,
		&req.StartDate, &req.EndDate, &req.IsHalfDayStart, &req.HalfDayPeriodStart,
		&req.IsHalfDayEnd, &req.HalfDayPeriodEnd, &totalDaysFloat, &req.Reason,
		&req.Status, &req.ApprovedBy, &req.ApprovalComment, &req.ApprovedAt,
		&req.AUDocumentRequired, &req.AUDocumentFileID,
		&req.CreatedAt, &req.UpdatedAt,
		&req.EmployeeName, &req.LeaveTypeName, &req.LeaveTypeColor,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLeaveRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	req.TotalDays = decimal.NewFromFloat(totalDaysFloat)
	return &req, nil
}

func scanLeaveRequestFromRows(rows pgx.Rows) (*models.LeaveRequest, error) {
	var req models.LeaveRequest
	var totalDaysFloat float64
	err := rows.Scan(
		&req.ID, &req.TenantID, &req.EmployeeID, &req.LeaveTypeID,
		&req.StartDate, &req.EndDate, &req.IsHalfDayStart, &req.HalfDayPeriodStart,
		&req.IsHalfDayEnd, &req.HalfDayPeriodEnd, &totalDaysFloat, &req.Reason,
		&req.Status, &req.ApprovedBy, &req.ApprovalComment, &req.ApprovedAt,
		&req.AUDocumentRequired, &req.AUDocumentFileID,
		&req.CreatedAt, &req.UpdatedAt,
		&req.EmployeeName, &req.LeaveTypeName, &req.LeaveTypeColor,
	)
	if err != nil {
		return nil, err
	}
	req.TotalDays = decimal.NewFromFloat(totalDaysFloat)
	return &req, nil
}

func scanLeaveBalance(row pgx.Row) (*models.HRLeaveBalance, error) {
	var b models.HRLeaveBalance
	var entFloat, carriedFloat, usedFloat, remainFloat float64
	err := row.Scan(
		&b.ID, &b.TenantID, &b.EmployeeID, &b.Year,
		&entFloat, &carriedFloat, &usedFloat, &remainFloat,
		&b.CarryoverExpiresAt, &b.CarryoverNotified,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No balance record = not found, not an error
	}
	if err != nil {
		return nil, err
	}
	b.Entitlement = decimal.NewFromFloat(entFloat)
	b.CarriedOver = decimal.NewFromFloat(carriedFloat)
	b.Used = decimal.NewFromFloat(usedFloat)
	b.Remaining = decimal.NewFromFloat(remainFloat)
	return &b, nil
}

func scanLeaveTypeFromRows(rows pgx.Rows) (*models.LeaveType, error) {
	var lt models.LeaveType
	err := rows.Scan(
		&lt.ID, &lt.TenantID, &lt.Name, &lt.Key, &lt.Color,
		&lt.DeductsFromBalance, &lt.RequiresApproval, &lt.RequiresAUDocument,
		&lt.IsSystem, &lt.SortOrder, &lt.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &lt, nil
}
