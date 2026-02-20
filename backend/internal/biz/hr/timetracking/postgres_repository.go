package timetracking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Work Time Entry Repository
// ============================================================================

// PostgresWorkTimeRepo implements WorkTimeRepository using PostgreSQL.
type PostgresWorkTimeRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresWorkTimeRepo creates a new PostgreSQL work time repository.
func NewPostgresWorkTimeRepo(pool *pgxpool.Pool) *PostgresWorkTimeRepo {
	return &PostgresWorkTimeRepo{pool: pool}
}

func (r *PostgresWorkTimeRepo) Create(ctx context.Context, entry *models.HRWorkTimeEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_work_time_entries (
			id, tenant_id, employee_id, clock_in, clock_out,
			break_minutes, auto_break_deducted, net_work_minutes,
			status, is_correction, original_entry_id,
			correction_reason, correction_approved_by, correction_approved_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16
		)`,
		entry.ID, entry.TenantID, entry.EmployeeID, entry.ClockIn, entry.ClockOut,
		entry.BreakMinutes, entry.AutoBreakDeducted, entry.NetWorkMinutes,
		entry.Status, entry.IsCorrection, entry.OriginalEntryID,
		entry.CorrectionReason, entry.CorrectionApprovedBy, entry.CorrectionApprovedAt,
		entry.CreatedAt, entry.UpdatedAt,
	)
	return err
}

func (r *PostgresWorkTimeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.HRWorkTimeEntry, error) {
	entry := &models.HRWorkTimeEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT w.id, w.tenant_id, w.employee_id, w.clock_in, w.clock_out,
			w.break_minutes, w.auto_break_deducted, w.net_work_minutes,
			w.status, w.is_correction, w.original_entry_id,
			w.correction_reason, w.correction_approved_by, w.correction_approved_at,
			w.created_at, w.updated_at,
			COALESCE(u.first_name || ' ' || u.last_name, '') AS employee_name
		FROM hr_work_time_entries w
		LEFT JOIN hr_employee_profiles ep ON w.employee_id = ep.user_id
		LEFT JOIN users u ON ep.user_id = u.id
		WHERE w.id = $1`,
		id,
	).Scan(
		&entry.ID, &entry.TenantID, &entry.EmployeeID, &entry.ClockIn, &entry.ClockOut,
		&entry.BreakMinutes, &entry.AutoBreakDeducted, &entry.NetWorkMinutes,
		&entry.Status, &entry.IsCorrection, &entry.OriginalEntryID,
		&entry.CorrectionReason, &entry.CorrectionApprovedBy, &entry.CorrectionApprovedAt,
		&entry.CreatedAt, &entry.UpdatedAt,
		&entry.EmployeeName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkTimeEntryNotFound
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *PostgresWorkTimeRepo) GetActiveShift(ctx context.Context, employeeID uuid.UUID) (*models.HRWorkTimeEntry, error) {
	entry := &models.HRWorkTimeEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, employee_id, clock_in, clock_out,
			break_minutes, auto_break_deducted, net_work_minutes,
			status, is_correction, original_entry_id,
			correction_reason, correction_approved_by, correction_approved_at,
			created_at, updated_at
		FROM hr_work_time_entries
		WHERE employee_id = $1 AND status = 'active'
		LIMIT 1`,
		employeeID,
	).Scan(
		&entry.ID, &entry.TenantID, &entry.EmployeeID, &entry.ClockIn, &entry.ClockOut,
		&entry.BreakMinutes, &entry.AutoBreakDeducted, &entry.NetWorkMinutes,
		&entry.Status, &entry.IsCorrection, &entry.OriginalEntryID,
		&entry.CorrectionReason, &entry.CorrectionApprovedBy, &entry.CorrectionApprovedAt,
		&entry.CreatedAt, &entry.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *PostgresWorkTimeRepo) Update(ctx context.Context, entry *models.HRWorkTimeEntry) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_work_time_entries SET
			clock_out = $2, break_minutes = $3, auto_break_deducted = $4,
			net_work_minutes = $5, status = $6,
			correction_approved_by = $7, correction_approved_at = $8,
			updated_at = $9
		WHERE id = $1`,
		entry.ID, entry.ClockOut, entry.BreakMinutes, entry.AutoBreakDeducted,
		entry.NetWorkMinutes, entry.Status,
		entry.CorrectionApprovedBy, entry.CorrectionApprovedAt,
		entry.UpdatedAt,
	)
	return err
}

func (r *PostgresWorkTimeRepo) List(ctx context.Context, filter WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("w.tenant_id = $%d", argIdx))
	args = append(args, filter.TenantID)
	argIdx++

	if filter.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("w.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("w.clock_in >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("w.clock_in < $%d", argIdx))
		args = append(args, filter.DateTo.Add(24*time.Hour))
		argIdx++
	}

	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("w.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_work_time_entries w WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	limit := filter.PerPage
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	dataQuery := fmt.Sprintf(
		`SELECT w.id, w.tenant_id, w.employee_id, w.clock_in, w.clock_out,
			w.break_minutes, w.auto_break_deducted, w.net_work_minutes,
			w.status, w.is_correction, w.original_entry_id,
			w.correction_reason, w.correction_approved_by, w.correction_approved_at,
			w.created_at, w.updated_at,
			COALESCE(u.first_name || ' ' || u.last_name, '') AS employee_name
		FROM hr_work_time_entries w
		LEFT JOIN hr_employee_profiles ep ON w.employee_id = ep.user_id
		LEFT JOIN users u ON ep.user_id = u.id
		WHERE %s
		ORDER BY w.clock_in DESC
		LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*models.HRWorkTimeEntry
	for rows.Next() {
		entry := &models.HRWorkTimeEntry{}
		if scanErr := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.EmployeeID, &entry.ClockIn, &entry.ClockOut,
			&entry.BreakMinutes, &entry.AutoBreakDeducted, &entry.NetWorkMinutes,
			&entry.Status, &entry.IsCorrection, &entry.OriginalEntryID,
			&entry.CorrectionReason, &entry.CorrectionApprovedBy, &entry.CorrectionApprovedAt,
			&entry.CreatedAt, &entry.UpdatedAt,
			&entry.EmployeeName,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		entries = append(entries, entry)
	}
	if entries == nil {
		entries = []*models.HRWorkTimeEntry{}
	}
	return entries, total, nil
}

func (r *PostgresWorkTimeRepo) GetPreviousShiftEnd(ctx context.Context, employeeID uuid.UUID, before time.Time) (*time.Time, error) {
	var clockOut *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT clock_out
		FROM hr_work_time_entries
		WHERE employee_id = $1 AND clock_out IS NOT NULL AND clock_in < $2
		ORDER BY clock_in DESC
		LIMIT 1`,
		employeeID, before,
	).Scan(&clockOut)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return clockOut, nil
}

func (r *PostgresWorkTimeRepo) GetDailySummary(ctx context.Context, employeeID uuid.UUID, date time.Time) (*DailySummary, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	summary := &DailySummary{Date: dayStart}

	err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(clock_out, NOW()) - clock_in)) / 60)::int, 0),
			COALESCE(SUM(break_minutes + auto_break_deducted), 0),
			COALESCE(SUM(COALESCE(net_work_minutes, 0)), 0),
			COUNT(*)
		FROM hr_work_time_entries
		WHERE employee_id = $1
			AND clock_in >= $2 AND clock_in < $3
			AND status IN ('active', 'completed', 'correction_approved')`,
		employeeID, dayStart, dayEnd,
	).Scan(
		&summary.TotalWorkedMinutes,
		&summary.TotalBreakMinutes,
		&summary.NetWorkMinutes,
		&summary.EntryCount,
	)
	if err != nil {
		return nil, err
	}

	// For active shifts, recalculate net from total - breaks
	// since net_work_minutes is NULL for active shifts
	if summary.NetWorkMinutes == 0 && summary.TotalWorkedMinutes > 0 {
		summary.NetWorkMinutes = summary.TotalWorkedMinutes - summary.TotalBreakMinutes
		if summary.NetWorkMinutes < 0 {
			summary.NetWorkMinutes = 0
		}
	}

	// Overtime = net work - 8h (480 min standard)
	summary.OvertimeMinutes = summary.NetWorkMinutes - 480
	if summary.OvertimeMinutes < 0 {
		summary.OvertimeMinutes = 0
	}

	return summary, nil
}

func (r *PostgresWorkTimeRepo) GetWeeklySummary(ctx context.Context, employeeID uuid.UUID, weekStart time.Time) (*WeeklySummary, error) {
	// Ensure weekStart is a Monday
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	summary := &WeeklySummary{
		WeekStart: weekStart,
		Days:      make([]DailySummary, 7),
	}

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dailySummary, err := r.GetDailySummary(ctx, employeeID, day)
		if err != nil {
			return nil, err
		}
		summary.Days[i] = *dailySummary
		summary.TotalWorkedMinutes += dailySummary.TotalWorkedMinutes
		summary.TotalBreakMinutes += dailySummary.TotalBreakMinutes
		summary.NetWorkMinutes += dailySummary.NetWorkMinutes
		summary.TotalOvertimeMinutes += dailySummary.OvertimeMinutes
	}

	return summary, nil
}

// ============================================================================
// Break Repository
// ============================================================================

// PostgresBreakRepo implements BreakRepository using PostgreSQL.
type PostgresBreakRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresBreakRepo creates a new PostgreSQL break repository.
func NewPostgresBreakRepo(pool *pgxpool.Pool) *PostgresBreakRepo {
	return &PostgresBreakRepo{pool: pool}
}

func (r *PostgresBreakRepo) Create(ctx context.Context, entry *models.HRBreakEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_break_entries (id, work_time_entry_id, start_time, end_time, duration_minutes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.WorkTimeEntryID, entry.StartTime, entry.EndTime, entry.DurationMinutes, entry.CreatedAt,
	)
	return err
}

func (r *PostgresBreakRepo) GetActiveBreak(ctx context.Context, workTimeEntryID uuid.UUID) (*models.HRBreakEntry, error) {
	entry := &models.HRBreakEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, work_time_entry_id, start_time, end_time, duration_minutes, created_at
		FROM hr_break_entries
		WHERE work_time_entry_id = $1 AND end_time IS NULL
		LIMIT 1`,
		workTimeEntryID,
	).Scan(&entry.ID, &entry.WorkTimeEntryID, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *PostgresBreakRepo) Update(ctx context.Context, entry *models.HRBreakEntry) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_break_entries SET end_time = $2, duration_minutes = $3 WHERE id = $1`,
		entry.ID, entry.EndTime, entry.DurationMinutes,
	)
	return err
}

func (r *PostgresBreakRepo) ListByWorkTimeEntry(ctx context.Context, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, work_time_entry_id, start_time, end_time, duration_minutes, created_at
		FROM hr_break_entries
		WHERE work_time_entry_id = $1
		ORDER BY start_time ASC`,
		workTimeEntryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.HRBreakEntry
	for rows.Next() {
		entry := &models.HRBreakEntry{}
		if scanErr := rows.Scan(
			&entry.ID, &entry.WorkTimeEntryID, &entry.StartTime, &entry.EndTime,
			&entry.DurationMinutes, &entry.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if entries == nil {
		entries = []*models.HRBreakEntry{}
	}
	return entries, nil
}
