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
			category_id, project_id, location_lat, location_lng, location_address,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18, $19,
			$20, $21
		)`,
		entry.ID, entry.TenantID, entry.EmployeeID, entry.ClockIn, entry.ClockOut,
		entry.BreakMinutes, entry.AutoBreakDeducted, entry.NetWorkMinutes,
		entry.Status, entry.IsCorrection, entry.OriginalEntryID,
		entry.CorrectionReason, entry.CorrectionApprovedBy, entry.CorrectionApprovedAt,
		entry.CategoryID, entry.ProjectID, entry.LocationLat, entry.LocationLng, entry.LocationAddress,
		entry.CreatedAt, entry.UpdatedAt,
	)
	return err
}

func (r *PostgresWorkTimeRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRWorkTimeEntry, error) {
	entry := &models.HRWorkTimeEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT w.id, w.tenant_id, w.employee_id, w.clock_in, w.clock_out,
			w.break_minutes, w.auto_break_deducted, w.net_work_minutes,
			w.status, w.is_correction, w.original_entry_id,
			COALESCE(w.correction_reason, '') AS correction_reason, w.correction_approved_by, w.correction_approved_at,
			w.created_at, w.updated_at, w.project_id,
			w.category_id, w.location_lat, w.location_lng, w.location_address,
			COALESCE(u.first_name || ' ' || u.last_name, '') AS employee_name
		FROM hr_work_time_entries w
		LEFT JOIN hr_employee_profiles ep ON w.employee_id = ep.user_id
		LEFT JOIN users u ON ep.user_id = u.id
		WHERE w.id = $1 AND w.tenant_id = $2`,
		id, tenantID,
	).Scan(
		&entry.ID, &entry.TenantID, &entry.EmployeeID, &entry.ClockIn, &entry.ClockOut,
		&entry.BreakMinutes, &entry.AutoBreakDeducted, &entry.NetWorkMinutes,
		&entry.Status, &entry.IsCorrection, &entry.OriginalEntryID,
		&entry.CorrectionReason, &entry.CorrectionApprovedBy, &entry.CorrectionApprovedAt,
		&entry.CreatedAt, &entry.UpdatedAt, &entry.ProjectID,
		&entry.CategoryID, &entry.LocationLat, &entry.LocationLng, &entry.LocationAddress,
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

func (r *PostgresWorkTimeRepo) GetActiveShift(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRWorkTimeEntry, error) {
	entry := &models.HRWorkTimeEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, employee_id, clock_in, clock_out,
			break_minutes, auto_break_deducted, net_work_minutes,
			status, is_correction, original_entry_id,
			COALESCE(correction_reason, '') AS correction_reason, correction_approved_by, correction_approved_at,
			created_at, updated_at,
			category_id, project_id, location_lat, location_lng, location_address
		FROM hr_work_time_entries
		WHERE tenant_id = $2 AND employee_id = $1 AND status = 'active'
		LIMIT 1`,
		employeeID, tenantID,
	).Scan(
		&entry.ID, &entry.TenantID, &entry.EmployeeID, &entry.ClockIn, &entry.ClockOut,
		&entry.BreakMinutes, &entry.AutoBreakDeducted, &entry.NetWorkMinutes,
		&entry.Status, &entry.IsCorrection, &entry.OriginalEntryID,
		&entry.CorrectionReason, &entry.CorrectionApprovedBy, &entry.CorrectionApprovedAt,
		&entry.CreatedAt, &entry.UpdatedAt,
		&entry.CategoryID, &entry.ProjectID, &entry.LocationLat, &entry.LocationLng, &entry.LocationAddress,
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
		WHERE id = $1 AND tenant_id = $10`,
		entry.ID, entry.ClockOut, entry.BreakMinutes, entry.AutoBreakDeducted,
		entry.NetWorkMinutes, entry.Status,
		entry.CorrectionApprovedBy, entry.CorrectionApprovedAt,
		entry.UpdatedAt, entry.TenantID,
	)
	return err
}

// ApproveCorrection stores the approved correction and retires the entry it
// replaces in one transaction, so the two writes that together keep the balance
// correct can never land separately.
func (r *PostgresWorkTimeRepo) ApproveCorrection(ctx context.Context, correction *models.HRWorkTimeEntry, originalID *uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx,
		`UPDATE hr_work_time_entries SET
			clock_out = $2, break_minutes = $3, auto_break_deducted = $4,
			net_work_minutes = $5, status = $6,
			correction_approved_by = $7, correction_approved_at = $8,
			updated_at = $9
		WHERE id = $1 AND tenant_id = $10`,
		correction.ID, correction.ClockOut, correction.BreakMinutes, correction.AutoBreakDeducted,
		correction.NetWorkMinutes, correction.Status,
		correction.CorrectionApprovedBy, correction.CorrectionApprovedAt,
		correction.UpdatedAt, correction.TenantID,
	); err != nil {
		return err
	}

	if originalID != nil {
		// Scoped to the correction's tenant and to the states an original can be in:
		// a second approval must not pull an already superseded entry around again.
		if _, err = tx.Exec(ctx,
			`UPDATE hr_work_time_entries
			SET status = $3, updated_at = $4
			WHERE id = $1 AND tenant_id = $2 AND status = ANY($5)`,
			*originalID, correction.TenantID, models.WorkTimeStatusSuperseded, correction.UpdatedAt,
			[]string{string(models.WorkTimeStatusActive), string(models.WorkTimeStatusCompleted)},
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
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
			COALESCE(w.correction_reason, '') AS correction_reason, w.correction_approved_by, w.correction_approved_at,
			w.created_at, w.updated_at, w.project_id,
			w.category_id, w.location_lat, w.location_lng, w.location_address,
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
			&entry.CreatedAt, &entry.UpdatedAt, &entry.ProjectID,
			&entry.CategoryID, &entry.LocationLat, &entry.LocationLng, &entry.LocationAddress,
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

func (r *PostgresWorkTimeRepo) GetPreviousShiftEnd(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error) {
	var clockOut *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT clock_out
		FROM hr_work_time_entries
		WHERE tenant_id = $3 AND employee_id = $1 AND clock_out IS NOT NULL AND clock_in < $2
		ORDER BY clock_in DESC
		LIMIT 1`,
		employeeID, before, tenantID,
	).Scan(&clockOut)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return clockOut, nil
}

func (r *PostgresWorkTimeRepo) GetDailySummary(ctx context.Context, tenantID, employeeID uuid.UUID, date time.Time) (*DailySummary, error) {
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
		WHERE tenant_id = $5 AND employee_id = $1
			AND clock_in >= $2 AND clock_in < $3
			AND status = ANY($4)`,
		employeeID, dayStart, dayEnd, balanceStatuses, tenantID,
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

func (r *PostgresWorkTimeRepo) GetWeeklySummary(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*WeeklySummary, error) {
	batch, err := r.GetWeeklySummaryBatch(ctx, tenantID, []uuid.UUID{employeeID}, weekStart)
	if err != nil {
		return nil, err
	}
	return batch[employeeID], nil
}

// GetWeeklySummaryBatch aggregates the weekly summary for every requested employee
// in a single GROUP BY query, replacing the previous N×7 per-day round-trips.
func (r *PostgresWorkTimeRepo) GetWeeklySummaryBatch(ctx context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID, weekStart time.Time) (map[uuid.UUID]*WeeklySummary, error) {
	// Ensure weekStart is a Monday at local midnight.
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	buckets, err := r.aggregateDailyBuckets(ctx, tenantID, employeeIDs, weekStart, 7)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*WeeklySummary, len(employeeIDs))
	for _, id := range employeeIDs {
		days := buckets[id]
		summary := &WeeklySummary{WeekStart: weekStart, Days: days}
		for i := range days {
			summary.TotalWorkedMinutes += days[i].TotalWorkedMinutes
			summary.TotalBreakMinutes += days[i].TotalBreakMinutes
			summary.NetWorkMinutes += days[i].NetWorkMinutes
			summary.TotalOvertimeMinutes += days[i].OvertimeMinutes
		}
		result[id] = summary
	}
	return result, nil
}

// GetDailySummaryRange returns one DailySummary per day for [start, start+numDays)
// in a single aggregated query, replacing numDays per-day GetDailySummary calls.
func (r *PostgresWorkTimeRepo) GetDailySummaryRange(ctx context.Context, tenantID, employeeID uuid.UUID, start time.Time, numDays int) ([]DailySummary, error) {
	buckets, err := r.aggregateDailyBuckets(ctx, tenantID, []uuid.UUID{employeeID}, start, numDays)
	if err != nil {
		return nil, err
	}
	return buckets[employeeID], nil
}

// aggregateDailyBuckets runs a single GROUP BY query that buckets work time entries
// into consecutive 24h windows starting at `start`, for all given employees over
// numDays days. It returns, per requested employee, a slice of numDays DailySummary
// values (index == day offset), applying the same per-day net/overtime recomputation
// as GetDailySummary. Employees without entries get a zeroed slice.
func (r *PostgresWorkTimeRepo) aggregateDailyBuckets(ctx context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID, start time.Time, numDays int) (map[uuid.UUID][]DailySummary, error) {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	// Use exact 24h windows so day_idx is always in [0, numDays): no gaps/overlaps,
	// even across DST (matches GetDailySummary's dayEnd = dayStart + 24h semantics).
	end := start.Add(time.Duration(numDays) * 24 * time.Hour)

	result := make(map[uuid.UUID][]DailySummary, len(employeeIDs))
	for _, id := range employeeIDs {
		days := make([]DailySummary, numDays)
		for i := range days {
			days[i] = DailySummary{Date: start.AddDate(0, 0, i)}
		}
		result[id] = days
	}
	if len(employeeIDs) == 0 || numDays <= 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT
			employee_id,
			FLOOR(EXTRACT(EPOCH FROM (clock_in - $2::timestamptz)) / 86400)::int AS day_idx,
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(clock_out, NOW()) - clock_in)) / 60)::int, 0),
			COALESCE(SUM(break_minutes + auto_break_deducted), 0),
			COALESCE(SUM(COALESCE(net_work_minutes, 0)), 0),
			COUNT(*)
		FROM hr_work_time_entries
		WHERE tenant_id = $5 AND employee_id = ANY($1)
			AND clock_in >= $2 AND clock_in < $3
			AND status = ANY($4)
		GROUP BY employee_id, day_idx`,
		employeeIDs, start, end, balanceStatuses, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var empID uuid.UUID
		var dayIdx int
		var d DailySummary
		if scanErr := rows.Scan(
			&empID, &dayIdx,
			&d.TotalWorkedMinutes, &d.TotalBreakMinutes, &d.NetWorkMinutes, &d.EntryCount,
		); scanErr != nil {
			return nil, scanErr
		}
		days, ok := result[empID]
		if !ok || dayIdx < 0 || dayIdx >= numDays {
			continue
		}
		d.Date = days[dayIdx].Date // preserve pre-seeded label
		// For active shifts net_work_minutes is NULL, so recompute from total - breaks.
		if d.NetWorkMinutes == 0 && d.TotalWorkedMinutes > 0 {
			d.NetWorkMinutes = d.TotalWorkedMinutes - d.TotalBreakMinutes
			if d.NetWorkMinutes < 0 {
				d.NetWorkMinutes = 0
			}
		}
		d.OvertimeMinutes = d.NetWorkMinutes - 480
		if d.OvertimeMinutes < 0 {
			d.OvertimeMinutes = 0
		}
		days[dayIdx] = d
	}
	return result, rows.Err()
}

// GetActiveShiftEmployeeIDs returns the set of employee IDs with a currently active
// shift in a single query, replacing N per-employee GetActiveShift calls.
func (r *PostgresWorkTimeRepo) GetActiveShiftEmployeeIDs(ctx context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(employeeIDs))
	if len(employeeIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT employee_id
		FROM hr_work_time_entries
		WHERE tenant_id = $2 AND employee_id = ANY($1) AND status = 'active'`,
		employeeIDs, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		result[id] = true
	}
	return result, rows.Err()
}

// GetProjectBreakdown returns per-project aggregated net_work_minutes for the given employee and date range.
func (r *PostgresWorkTimeRepo) GetProjectBreakdown(ctx context.Context, tenantID, employeeID uuid.UUID, dateFrom, dateTo time.Time) ([]ProjectBreakdown, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT w.project_id, COALESCE(p.name, 'Kein Projekt') AS project_name,
			COALESCE(SUM(w.net_work_minutes), 0) AS minutes
		FROM hr_work_time_entries w
		LEFT JOIN hr_time_projects p ON w.project_id = p.id
		WHERE w.tenant_id = $1 AND w.employee_id = $2
			AND w.clock_in >= $3 AND w.clock_in < $4
			AND w.status = ANY($5)
		GROUP BY w.project_id, p.name
		ORDER BY minutes DESC`,
		tenantID, employeeID, dateFrom, dateTo.Add(24*time.Hour), billableStatuses,
	)
	if err != nil {
		return nil, fmt.Errorf("get project breakdown: %w", err)
	}
	defer rows.Close()

	var result []ProjectBreakdown
	for rows.Next() {
		var pb ProjectBreakdown
		var projectID *uuid.UUID
		if err := rows.Scan(&projectID, &pb.ProjectName, &pb.Minutes); err != nil {
			return nil, err
		}
		if projectID != nil {
			pb.ProjectID = *projectID
		}
		result = append(result, pb)
	}
	if result == nil {
		result = []ProjectBreakdown{}
	}
	return result, rows.Err()
}

// ReserveWorkTimeForInvoice locks the billable work time entries of the given
// employee within [from, to] — finished entries and approved corrections,
// never the originals they replaced, and never an entry already billed — and
// marks them billed in the same transaction before returning their total.
//
// The SELECT ... FOR UPDATE and the UPDATE that stamps billed_at happen
// inside one transaction, so a second call for the same employee/period
// (double-click, retry after timeout, two staff members working the same
// invoice) blocks on the row lock until the first call commits, then sees
// billed_at already set and returns zero — never a second overlapping set of
// entries. Returns (0, nil, nil) if nothing was available to bill.
//
// If the caller fails to turn the reservation into an invoice, it must call
// ReleaseInvoiceReservation with the returned entry IDs, or the hours are
// stuck billed with no invoice. If it succeeds, it should call
// ConfirmInvoiceReservation to record which invoice claimed them.
func (r *PostgresWorkTimeRepo) ReserveWorkTimeForInvoice(ctx context.Context, tenantID, employeeID uuid.UUID, from, to time.Time) (int, []string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx,
		`SELECT id, net_work_minutes
		 FROM hr_work_time_entries
		 WHERE tenant_id = $1
		   AND employee_id = $2
		   AND status = ANY($5)
		   AND clock_in >= $3
		   AND clock_in <= $4
		   AND net_work_minutes IS NOT NULL
		   AND billed_at IS NULL
		 FOR UPDATE`,
		tenantID, employeeID, from, to, billableStatuses,
	)
	if err != nil {
		return 0, nil, err
	}

	var total int
	var entryIDs []string
	for rows.Next() {
		var entryID uuid.UUID
		var netMinutes int
		if scanErr := rows.Scan(&entryID, &netMinutes); scanErr != nil {
			rows.Close()
			return 0, nil, scanErr
		}
		total += netMinutes
		entryIDs = append(entryIDs, entryID.String())
	}
	if scanErr := rows.Err(); scanErr != nil {
		rows.Close()
		return 0, nil, scanErr
	}
	rows.Close()

	if len(entryIDs) == 0 {
		return 0, nil, tx.Commit(ctx)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE hr_work_time_entries SET billed_at = NOW()
		 WHERE tenant_id = $1 AND id = ANY($2::uuid[])`,
		tenantID, entryIDs,
	); err != nil {
		return 0, nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, nil, err
	}
	return total, entryIDs, nil
}

// ConfirmInvoiceReservation stamps the invoice that claimed a set of entries
// reserved by ReserveWorkTimeForInvoice. Best-effort traceability: the
// double-billing guard is already enforced by billed_at, set at reservation
// time, so a failure here does not need to roll back the invoice.
func (r *PostgresWorkTimeRepo) ConfirmInvoiceReservation(ctx context.Context, tenantID, invoiceID uuid.UUID, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_work_time_entries SET invoice_id = $1
		 WHERE tenant_id = $2 AND id = ANY($3::uuid[])`,
		invoiceID, tenantID, entryIDs,
	)
	return err
}

// ReleaseInvoiceReservation undoes a reservation whose invoice was never
// created (e.g. invoiceService.Create failed after the reserve). Scoped to
// invoice_id IS NULL so it can never clear an entry a concurrent, already
// confirmed invoice claimed in between.
func (r *PostgresWorkTimeRepo) ReleaseInvoiceReservation(ctx context.Context, tenantID uuid.UUID, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE hr_work_time_entries SET billed_at = NULL
		 WHERE tenant_id = $1 AND id = ANY($2::uuid[]) AND invoice_id IS NULL`,
		tenantID, entryIDs,
	)
	return err
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
		`INSERT INTO hr_break_entries (id, tenant_id, work_time_entry_id, start_time, end_time, duration_minutes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.TenantID, entry.WorkTimeEntryID, entry.StartTime, entry.EndTime, entry.DurationMinutes, entry.CreatedAt,
	)
	return err
}

func (r *PostgresBreakRepo) GetActiveBreak(ctx context.Context, tenantID, workTimeEntryID uuid.UUID) (*models.HRBreakEntry, error) {
	entry := &models.HRBreakEntry{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, work_time_entry_id, start_time, end_time, duration_minutes, created_at
		FROM hr_break_entries
		WHERE tenant_id = $2 AND work_time_entry_id = $1 AND end_time IS NULL
		LIMIT 1`,
		workTimeEntryID, tenantID,
	).Scan(&entry.ID, &entry.TenantID, &entry.WorkTimeEntryID, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.CreatedAt)
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
		`UPDATE hr_break_entries SET end_time = $2, duration_minutes = $3 WHERE id = $1 AND tenant_id = $4`,
		entry.ID, entry.EndTime, entry.DurationMinutes, entry.TenantID,
	)
	return err
}

func (r *PostgresBreakRepo) ListByWorkTimeEntry(ctx context.Context, tenantID, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, work_time_entry_id, start_time, end_time, duration_minutes, created_at
		FROM hr_break_entries
		WHERE tenant_id = $2 AND work_time_entry_id = $1
		ORDER BY start_time ASC`,
		workTimeEntryID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.HRBreakEntry
	for rows.Next() {
		entry := &models.HRBreakEntry{}
		if scanErr := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.WorkTimeEntryID, &entry.StartTime, &entry.EndTime,
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
