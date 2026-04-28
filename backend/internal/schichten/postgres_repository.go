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
		argNum++
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

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
