package schichten

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Shifts
// ============================================================================

func (r *PostgresRepository) CreateShift(ctx context.Context, shift *Shift) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO shifts
		    (id, tenant_id, title, description, start_time, end_time, status, location, capacity, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		shift.ID, shift.TenantID, shift.Title, shift.Description,
		shift.StartTime, shift.EndTime, shift.Status, shift.Location, shift.Capacity, shift.CreatedBy,
		shift.CreatedAt, shift.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateShift(ctx context.Context, shift *Shift) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE shifts
		 SET title = $1, description = $2, start_time = $3, end_time = $4,
		     status = $5, location = $6, capacity = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10`,
		shift.Title, shift.Description, shift.StartTime, shift.EndTime,
		shift.Status, shift.Location, shift.Capacity, shift.UpdatedAt, shift.ID, shift.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrShiftNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteShift(ctx context.Context, tenantID, shiftID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM shifts WHERE id = $1 AND tenant_id = $2`,
		shiftID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrShiftNotFound
	}
	return nil
}

func (r *PostgresRepository) GetShift(ctx context.Context, tenantID, shiftID uuid.UUID) (*Shift, error) {
	var s Shift
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, description, start_time, end_time, status, location, capacity, created_by, created_at, updated_at
		 FROM shifts WHERE id = $1 AND tenant_id = $2`,
		shiftID, tenantID,
	).Scan(
		&s.ID, &s.TenantID, &s.Title, &s.Description,
		&s.StartTime, &s.EndTime, &s.Status, &s.Location, &s.Capacity, &s.CreatedBy,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShiftNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shift: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) ListShifts(ctx context.Context, tenantID uuid.UUID, filter ListShiftsFilter, offset, limit int) ([]*Shift, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("start_time >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}
	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("end_time <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM shifts %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count shifts: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, title, description, start_time, end_time, status, location, capacity, created_by, created_at, updated_at
		FROM shifts %s
		ORDER BY start_time ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*Shift
	for rows.Next() {
		s, scanErr := r.scanShiftFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		shifts = append(shifts, s)
	}
	return shifts, total, rows.Err()
}

func (r *PostgresRepository) PublishShifts(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error) {
	ct, err := r.pool.Exec(ctx,
		`UPDATE shifts
		 SET status = 'published', updated_at = NOW()
		 WHERE tenant_id = $1
		   AND start_time >= $2
		   AND end_time <= $3
		   AND status = 'draft'`,
		tenantID, from, to,
	)
	if err != nil {
		return 0, fmt.Errorf("publish shifts: %w", err)
	}
	return ct.RowsAffected(), nil
}

// ============================================================================
// Assignments
// ============================================================================

func (r *PostgresRepository) CreateAssignment(ctx context.Context, a *ShiftAssignment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO shift_assignments
		    (id, tenant_id, shift_id, employee_id, assigned_at, assigned_by)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.TenantID, a.ShiftID, a.EmployeeID, a.AssignedAt, a.AssignedBy,
	)
	return err
}

func (r *PostgresRepository) DeleteAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM shift_assignments
		 WHERE tenant_id = $1 AND shift_id = $2 AND employee_id = $3`,
		tenantID, shiftID, employeeID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrAssignmentNotFound
	}
	return nil
}

func (r *PostgresRepository) GetAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) (*ShiftAssignment, error) {
	var a ShiftAssignment
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, shift_id, employee_id, assigned_at, assigned_by
		 FROM shift_assignments
		 WHERE tenant_id = $1 AND shift_id = $2 AND employee_id = $3`,
		tenantID, shiftID, employeeID,
	).Scan(&a.ID, &a.TenantID, &a.ShiftID, &a.EmployeeID, &a.AssignedAt, &a.AssignedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAssignmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) ListAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) ([]*ShiftAssignment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, shift_id, employee_id, assigned_at, assigned_by
		 FROM shift_assignments
		 WHERE tenant_id = $1 AND shift_id = $2
		 ORDER BY assigned_at ASC`,
		tenantID, shiftID,
	)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*ShiftAssignment
	for rows.Next() {
		a, scanErr := r.scanAssignmentFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// LatestShiftEndBeforeForEmployee finds the most recent shift end_time for an employee
// that ends strictly before the given timestamp. Used for ArbZG rest-period validation.
// Uses strict less-than (<) so a shift ending exactly at newStart is not counted
// as a prior shift — that edge case means zero rest, which is a violation handled by the caller.
func (r *PostgresRepository) LatestShiftEndBeforeForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error) {
	var endTime time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT s.end_time
		 FROM shifts s
		 JOIN shift_assignments sa ON sa.shift_id = s.id
		 WHERE sa.tenant_id = $1
		   AND sa.employee_id = $2
		   AND s.end_time < $3
		 ORDER BY s.end_time DESC
		 LIMIT 1`,
		tenantID, employeeID, before,
	).Scan(&endTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest shift end for employee: %w", err)
	}
	return &endTime, nil
}

// EarliestShiftStartAfterForEmployee finds the earliest shift start_time for an employee
// that begins strictly after the given timestamp. Used for bidirectional ArbZG rest-period validation.
func (r *PostgresRepository) EarliestShiftStartAfterForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, after time.Time) (*time.Time, error) {
	var startTime time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT s.start_time
		 FROM shifts s
		 JOIN shift_assignments sa ON sa.shift_id = s.id
		 WHERE sa.tenant_id = $1
		   AND sa.employee_id = $2
		   AND s.start_time > $3
		 ORDER BY s.start_time ASC
		 LIMIT 1`,
		tenantID, employeeID, after,
	).Scan(&startTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("earliest shift start for employee: %w", err)
	}
	return &startTime, nil
}

// ShiftExistsForTemplate checks whether a shift with the given tenant, start, end, and title already exists.
// Used by ApplyTemplate for idempotency (prevents duplicate application).
func (r *PostgresRepository) ShiftExistsForTemplate(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time, title string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM shifts
			WHERE tenant_id=$1 AND start_time=$2 AND end_time=$3 AND title=$4
		)`,
		tenantID, startTime, endTime, title,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check shift exists for template: %w", err)
	}
	return exists, nil
}

// ============================================================================
// Templates
// ============================================================================

func (r *PostgresRepository) CreateTemplate(ctx context.Context, t *ShiftTemplate) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO shift_templates
		    (id, tenant_id, name, description, day_of_week, start_hour, start_minute, duration_minutes, location, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID, t.TenantID, t.Name, t.Description,
		t.DayOfWeek, t.StartHour, t.StartMinute, t.DurationMinutes, t.Location,
		t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateTemplate(ctx context.Context, t *ShiftTemplate) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE shift_templates
		 SET name = $1, description = $2, day_of_week = $3, start_hour = $4, start_minute = $5,
		     duration_minutes = $6, location = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10`,
		t.Name, t.Description, t.DayOfWeek, t.StartHour, t.StartMinute,
		t.DurationMinutes, t.Location, t.UpdatedAt, t.ID, t.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM shift_templates WHERE id = $1 AND tenant_id = $2`,
		templateID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (r *PostgresRepository) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error) {
	var t ShiftTemplate
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, day_of_week, start_hour, start_minute, duration_minutes, location, created_at, updated_at
		 FROM shift_templates WHERE id = $1 AND tenant_id = $2`,
		templateID, tenantID,
	).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Description,
		&t.DayOfWeek, &t.StartHour, &t.StartMinute, &t.DurationMinutes, &t.Location,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shift template: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) ListTemplates(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*ShiftTemplate, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM shift_templates WHERE tenant_id = $1`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count shift templates: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, day_of_week, start_hour, start_minute, duration_minutes, location, created_at, updated_at
		 FROM shift_templates
		 WHERE tenant_id = $1
		 ORDER BY day_of_week ASC, start_hour ASC, start_minute ASC
		 LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list shift templates: %w", err)
	}
	defer rows.Close()

	var templates []*ShiftTemplate
	for rows.Next() {
		t, scanErr := r.scanTemplateFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		templates = append(templates, t)
	}
	return templates, total, rows.Err()
}

// ============================================================================
// Stats
// ============================================================================

func (r *PostgresRepository) GetStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*ShiftStats, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if from != nil {
		conditions = append(conditions, fmt.Sprintf("start_time >= $%d", argNum))
		args = append(args, *from)
		argNum++
	}
	if to != nil {
		conditions = append(conditions, fmt.Sprintf("end_time <= $%d", argNum))
		args = append(args, *to)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var stats ShiftStats
	err := r.pool.QueryRow(ctx,
		fmt.Sprintf(`
			SELECT
			    COUNT(*) AS total,
			    COUNT(*) FILTER (WHERE status = 'published') AS published,
			    COUNT(*) FILTER (WHERE status = 'draft') AS draft
			FROM shifts %s
		`, whereClause),
		args...,
	).Scan(&stats.TotalShifts, &stats.PublishedShifts, &stats.DraftShifts)
	if err != nil {
		return nil, fmt.Errorf("get shift stats: %w", err)
	}

	// Assignment stats (join to shifts to apply same time filters)
	assignArgs := []any{tenantID}
	assignArgNum := 2
	var assignConds []string
	assignConds = append(assignConds, "sa.tenant_id = $1")
	if from != nil {
		assignConds = append(assignConds, fmt.Sprintf("s.start_time >= $%d", assignArgNum))
		assignArgs = append(assignArgs, *from)
		assignArgNum++
	}
	if to != nil {
		assignConds = append(assignConds, fmt.Sprintf("s.end_time <= $%d", assignArgNum))
		assignArgs = append(assignArgs, *to)
	}
	assignWhere := "WHERE " + strings.Join(assignConds, " AND ")

	err = r.pool.QueryRow(ctx,
		fmt.Sprintf(`
			SELECT COUNT(*), COUNT(DISTINCT sa.employee_id)
			FROM shift_assignments sa
			JOIN shifts s ON s.id = sa.shift_id
			%s
		`, assignWhere),
		assignArgs...,
	).Scan(&stats.TotalAssignments, &stats.UniqueEmployees)
	if err != nil {
		return nil, fmt.Errorf("get assignment stats: %w", err)
	}

	return &stats, nil
}

// ============================================================================
// SwapRequests
// ============================================================================

// CreateSwapRequest inserts a new swap request with idempotency via ON CONFLICT DO NOTHING.
// If the idempotency_key already exists, it fetches and returns the existing record.
func (r *PostgresRepository) CreateSwapRequest(ctx context.Context, req *SwapRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO shift_swap_requests
		    (id, tenant_id, assignment_id, requested_by_employee_id, swap_with_employee_id, shift_id,
		     status, reason, idempotency_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (idempotency_key) DO NOTHING`,
		req.ID, req.TenantID, req.AssignmentID,
		req.RequestedByEmployeeID, req.SwapWithEmployeeID, req.ShiftID,
		req.Status, req.Reason, req.IdempotencyKey,
		req.CreatedAt, req.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create swap request: %w", err)
	}
	// Re-fetch by idempotency_key to populate req.ID in case of conflict.
	existing := &SwapRequest{}
	fetchErr := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, assignment_id, requested_by_employee_id, swap_with_employee_id, shift_id,
		        status, reason, idempotency_key, created_at, updated_at
		 FROM shift_swap_requests WHERE idempotency_key = $1 AND tenant_id = $2`,
		req.IdempotencyKey, req.TenantID,
	).Scan(
		&existing.ID, &existing.TenantID, &existing.AssignmentID,
		&existing.RequestedByEmployeeID, &existing.SwapWithEmployeeID, &existing.ShiftID,
		&existing.Status, &existing.Reason, &existing.IdempotencyKey,
		&existing.CreatedAt, &existing.UpdatedAt,
	)
	if fetchErr != nil {
		return fmt.Errorf("re-fetch swap request by idempotency_key: %w", fetchErr)
	}
	*req = *existing
	return nil
}

// GetSwapRequest retrieves a single swap request by ID and tenant.
func (r *PostgresRepository) GetSwapRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*SwapRequest, error) {
	var req SwapRequest
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, assignment_id, requested_by_employee_id, swap_with_employee_id, shift_id,
		        status, reason, idempotency_key, created_at, updated_at
		 FROM shift_swap_requests WHERE id = $1 AND tenant_id = $2`,
		requestID, tenantID,
	).Scan(
		&req.ID, &req.TenantID, &req.AssignmentID,
		&req.RequestedByEmployeeID, &req.SwapWithEmployeeID, &req.ShiftID,
		&req.Status, &req.Reason, &req.IdempotencyKey,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSwapRequestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get swap request: %w", err)
	}
	return &req, nil
}

// ListSwapRequests returns paginated swap requests with optional filtering.
func (r *PostgresRepository) ListSwapRequests(ctx context.Context, tenantID uuid.UUID, filter SwapRequestFilter, offset, limit int) ([]*SwapRequest, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.ShiftID != nil {
		conditions = append(conditions, fmt.Sprintf("shift_id = $%d", argNum))
		args = append(args, *filter.ShiftID)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.OwnEmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"(requested_by_employee_id = $%d OR swap_with_employee_id = $%d)", argNum, argNum))
		args = append(args, *filter.OwnEmployeeID)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM shift_swap_requests %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count swap requests: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, assignment_id, requested_by_employee_id, swap_with_employee_id, shift_id,
		       status, reason, idempotency_key, created_at, updated_at
		FROM shift_swap_requests %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list swap requests: %w", err)
	}
	defer rows.Close()

	var reqs []*SwapRequest
	for rows.Next() {
		req, scanErr := r.scanSwapRequestFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		reqs = append(reqs, req)
	}
	return reqs, total, rows.Err()
}

// UpdateSwapRequestStatus updates the status of a swap request.
func (r *PostgresRepository) UpdateSwapRequestStatus(ctx context.Context, tenantID, requestID uuid.UUID, status SwapRequestStatus) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE shift_swap_requests
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $3`,
		status, requestID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update swap request status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrSwapRequestNotFound
	}
	return nil
}

// SwapAssignmentsForRequest atomically swaps the employee IDs of two shift assignments:
// the assignment identified by req.AssignmentID (requested_by) gets the swap_with employee,
// and the complementary assignment (same shift, swap_with employee) gets requested_by.
// A swap is only valid between two employees who are both already assigned to the
// shift -- if the swap partner has no assignment there, ErrSwapPartnerNotAssigned is
// returned and nothing is changed.
//
// Both updates run in a single DB transaction, scoped by assignment row id rather
// than by (shift_id, employee_id) -- scoping the second UPDATE by employee_id would
// re-match the row the first UPDATE just moved onto that same employee_id within
// this same transaction. uq_shift_assignments_tenant is DEFERRABLE (migration
// 000311): the swap defers its check to COMMIT, because immediately after the first
// UPDATE the requester's row and the partner's still-unmodified row briefly hold the
// same (tenant_id, shift_id, employee_id).
func (r *PostgresRepository) SwapAssignmentsForRequest(ctx context.Context, req *SwapRequest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin swap transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_shift_assignments_tenant DEFERRED`); err != nil {
		return fmt.Errorf("defer swap unique constraint: %w", err)
	}

	var partnerAssignmentID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM shift_assignments
		 WHERE tenant_id = $1 AND shift_id = $2 AND employee_id = $3
		 FOR UPDATE`,
		req.TenantID, req.ShiftID, req.SwapWithEmployeeID,
	).Scan(&partnerAssignmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSwapPartnerNotAssigned
	}
	if err != nil {
		return fmt.Errorf("lock swap partner assignment: %w", err)
	}

	// Update the requester's assignment to swap_with employee.
	_, err = tx.Exec(ctx,
		`UPDATE shift_assignments
		 SET employee_id = $1
		 WHERE id = $2 AND tenant_id = $3`,
		req.SwapWithEmployeeID, req.AssignmentID, req.TenantID,
	)
	if err != nil {
		return fmt.Errorf("swap assignment (requester): %w", err)
	}

	// Update the swap_with employee's assignment, identified by the row id
	// locked above, to requested_by.
	_, err = tx.Exec(ctx,
		`UPDATE shift_assignments
		 SET employee_id = $1
		 WHERE id = $2 AND tenant_id = $3`,
		req.RequestedByEmployeeID, partnerAssignmentID, req.TenantID,
	)
	if err != nil {
		return fmt.Errorf("swap assignment (swap_with): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit swap transaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) scanSwapRequestFromRows(rows pgx.Rows) (*SwapRequest, error) {
	var req SwapRequest
	err := rows.Scan(
		&req.ID, &req.TenantID, &req.AssignmentID,
		&req.RequestedByEmployeeID, &req.SwapWithEmployeeID, &req.ShiftID,
		&req.Status, &req.Reason, &req.IdempotencyKey,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan swap request row: %w", err)
	}
	return &req, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanShiftFromRows(rows pgx.Rows) (*Shift, error) {
	var s Shift
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.Title, &s.Description,
		&s.StartTime, &s.EndTime, &s.Status, &s.Location, &s.Capacity, &s.CreatedBy,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan shift row: %w", err)
	}
	return &s, nil
}

// CountAssignments returns the count of current assignments for a shift.
func (r *PostgresRepository) CountAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM shift_assignments WHERE tenant_id = $1 AND shift_id = $2`,
		tenantID, shiftID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count assignments: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) scanAssignmentFromRows(rows pgx.Rows) (*ShiftAssignment, error) {
	var a ShiftAssignment
	err := rows.Scan(&a.ID, &a.TenantID, &a.ShiftID, &a.EmployeeID, &a.AssignedAt, &a.AssignedBy)
	if err != nil {
		return nil, fmt.Errorf("scan assignment row: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) scanTemplateFromRows(rows pgx.Rows) (*ShiftTemplate, error) {
	var t ShiftTemplate
	err := rows.Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Description,
		&t.DayOfWeek, &t.StartHour, &t.StartMinute, &t.DurationMinutes, &t.Location,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan shift template row: %w", err)
	}
	return &t, nil
}

// IsMinorEmployee reports whether the employee's HR profile carries the
// JArbSchG minor flag. No row means no profile, which is not a claim about the
// employee's age — the check stays out of the way rather than blocking a shift
// on missing HR data.
//
// The query reads an HR table directly for the same reason
// PostgresEmployeeRepo.CountOtherActiveRoleAdmins reads an auth table: there is
// no service-to-service gRPC in this repository. RLS covers the read, and the
// tenant predicate stays explicit as defense in depth.
//
// lean: matches the profile id and the user id, because shift_assignments
// .employee_id has no foreign key and the desktop grid feeds it profile ids
// while the swap payload calls the same value a user id. Narrow this to one
// column once that column gets a foreign key.
func (r *PostgresRepository) IsMinorEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (bool, error) {
	var isMinor bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_minor FROM hr_employee_profiles
		 WHERE tenant_id = $1 AND (id = $2 OR user_id = $2)
		 LIMIT 1`,
		tenantID, employeeID,
	).Scan(&isMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("look up minor flag: %w", err)
	}
	return isMinor, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
