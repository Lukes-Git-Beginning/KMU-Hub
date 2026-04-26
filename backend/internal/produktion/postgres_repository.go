package produktion

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

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Production Orders
// ============================================================================

func (r *PostgresRepository) CreateOrder(ctx context.Context, order *ProductionOrder) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO production_orders
		    (id, tenant_id, order_number, product_name, quantity, status,
		     planned_start, planned_end, actual_start, actual_end,
		     priority, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		order.ID, order.TenantID, order.OrderNumber, order.ProductName,
		order.Quantity, order.Status, order.PlannedStart, order.PlannedEnd,
		order.ActualStart, order.ActualEnd, order.Priority, order.Notes,
		order.CreatedBy, order.CreatedAt, order.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateOrder(ctx context.Context, order *ProductionOrder) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE production_orders
		 SET product_name=$1, quantity=$2, status=$3,
		     planned_start=$4, planned_end=$5, actual_start=$6, actual_end=$7,
		     priority=$8, notes=$9, updated_at=$10
		 WHERE id=$11 AND tenant_id=$12`,
		order.ProductName, order.Quantity, order.Status,
		order.PlannedStart, order.PlannedEnd, order.ActualStart, order.ActualEnd,
		order.Priority, order.Notes, order.UpdatedAt,
		order.ID, order.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	return r.scanOrder(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, order_number, product_name, quantity, status,
		        planned_start, planned_end, actual_start, actual_end,
		        priority, notes, created_by, created_at, updated_at
		 FROM production_orders WHERE id=$1 AND tenant_id=$2`,
		orderID, tenantID,
	))
}

func (r *PostgresRepository) ListOrders(ctx context.Context, tenantID uuid.UUID, filter ListOrdersFilter, offset, limit int) ([]*ProductionOrder, int, error) {
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

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argNum))
		args = append(args, *filter.Priority)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("planned_start >= $%d", argNum))
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("planned_end <= $%d", argNum))
		args = append(args, *filter.DateTo)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM production_orders %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, order_number, product_name, quantity, status,
		       planned_start, planned_end, actual_start, actual_end,
		       priority, notes, created_by, created_at, updated_at
		FROM production_orders %s
		ORDER BY priority ASC, planned_start ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []*ProductionOrder
	for rows.Next() {
		o, scanErr := r.scanOrderFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		orders = append(orders, o)
	}

	return orders, total, rows.Err()
}

func (r *PostgresRepository) DeleteOrder(ctx context.Context, tenantID, orderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM production_orders WHERE id=$1 AND tenant_id=$2`,
		orderID, tenantID,
	)
	return err
}

func (r *PostgresRepository) OrderNumberExists(ctx context.Context, tenantID uuid.UUID, orderNumber string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM production_orders WHERE tenant_id=$1 AND order_number=$2 AND id <> $3)`,
			tenantID, orderNumber, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM production_orders WHERE tenant_id=$1 AND order_number=$2)`,
		tenantID, orderNumber,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// Machine Bookings
// ============================================================================

func (r *PostgresRepository) CreateBooking(ctx context.Context, booking *MachineBooking) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO machine_bookings
		    (id, tenant_id, machine_id, production_order_id, starts_at, ends_at,
		     status, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		booking.ID, booking.TenantID, booking.MachineID, booking.ProductionOrderID,
		booking.StartsAt, booking.EndsAt, booking.Status, booking.Notes,
		booking.CreatedBy, booking.CreatedAt, booking.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateBooking(ctx context.Context, booking *MachineBooking) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE machine_bookings
		 SET machine_id=$1, production_order_id=$2, starts_at=$3, ends_at=$4,
		     status=$5, notes=$6, updated_at=$7
		 WHERE id=$8 AND tenant_id=$9`,
		booking.MachineID, booking.ProductionOrderID, booking.StartsAt, booking.EndsAt,
		booking.Status, booking.Notes, booking.UpdatedAt,
		booking.ID, booking.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*MachineBooking, error) {
	return r.scanBooking(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, machine_id, production_order_id, starts_at, ends_at,
		        status, notes, created_by, created_at, updated_at
		 FROM machine_bookings WHERE id=$1 AND tenant_id=$2`,
		bookingID, tenantID,
	))
}

func (r *PostgresRepository) ListBookings(ctx context.Context, tenantID uuid.UUID, filter ListBookingsFilter, offset, limit int) ([]*MachineBooking, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.MachineID != nil {
		conditions = append(conditions, fmt.Sprintf("machine_id = $%d", argNum))
		args = append(args, *filter.MachineID)
		argNum++
	}

	if filter.ProductionOrderID != nil {
		conditions = append(conditions, fmt.Sprintf("production_order_id = $%d", argNum))
		args = append(args, *filter.ProductionOrderID)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("ends_at >= $%d", argNum))
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("starts_at <= $%d", argNum))
		args = append(args, *filter.DateTo)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM machine_bookings %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, machine_id, production_order_id, starts_at, ends_at,
		       status, notes, created_by, created_at, updated_at
		FROM machine_bookings %s
		ORDER BY starts_at ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*MachineBooking
	for rows.Next() {
		b, scanErr := r.scanBookingFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		bookings = append(bookings, b)
	}

	return bookings, total, rows.Err()
}

func (r *PostgresRepository) DeleteBooking(ctx context.Context, tenantID, bookingID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM machine_bookings WHERE id=$1 AND tenant_id=$2`,
		bookingID, tenantID,
	)
	return err
}

// FindConflictingBooking checks whether there is an active booking for the same machine
// that overlaps the given time interval [startsAt, endsAt).
//
// Overlap detection uses simple arithmetic: booking A overlaps B iff A.starts_at < B.ends_at AND A.ends_at > B.starts_at.
// We prefer this over tstzrange/btree_gist because the btree_gist extension is not guaranteed to be
// available in all deployment environments (e.g., managed Postgres without superuser).
// If btree_gist is confirmed available the query can be replaced with:
//
//	tstzrange(starts_at, ends_at, '[)') && tstzrange($4, $5, '[)')
//
// Note: when excludeID is nil, the WHERE clause omits the id-exclusion entirely.
// Earlier revisions substituted uuid.UUID{} as a sentinel, which silently filtered
// out any booking that happened to use the zero UUID — the dedicated branches below
// remove that fragile coupling.
//
// Race condition note: this method alone does NOT protect against concurrent double-booking.
// For atomic conflict-check + insert use CreateBookingWithLock, which wraps both operations
// in a single transaction guarded by pg_advisory_xact_lock(hashtext(tenant_id||':'||machine_id)).
func (r *PostgresRepository) FindConflictingBooking(ctx context.Context, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error) {
	var (
		conflictID uuid.UUID
		err        error
	)

	if excludeID == nil {
		err = r.pool.QueryRow(ctx,
			`SELECT id FROM machine_bookings
			 WHERE tenant_id = $1
			   AND machine_id = $2
			   AND status IN ('booked', 'in_use')
			   AND starts_at < $4
			   AND ends_at > $3
			 LIMIT 1`,
			tenantID, machineID, startsAt, endsAt,
		).Scan(&conflictID)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT id FROM machine_bookings
			 WHERE tenant_id = $1
			   AND machine_id = $2
			   AND status IN ('booked', 'in_use')
			   AND id != $3
			   AND starts_at < $5
			   AND ends_at > $4
			 LIMIT 1`,
			tenantID, machineID, *excludeID, startsAt, endsAt,
		).Scan(&conflictID)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find conflicting booking: %w", err)
	}
	return &conflictID, nil
}

// CreateBookingWithLock atomically creates a machine booking under a Postgres advisory lock.
//
// The lock key is hashtext(tenant_id::text || ':' || machine_id), scoped to the transaction.
// This serialises all concurrent booking attempts for the same (tenant, machine) pair without
// touching any unrelated rows or tables. The approach avoids btree_gist (extension dependency)
// and the impossibility of SELECT FOR UPDATE on a not-yet-existing row.
//
// Flow:
//  1. BeginTx (ReadCommitted)
//  2. pg_advisory_xact_lock — blocks until no other session holds the lock for this key
//  3. Conflict check inside the locked Tx
//  4a. Conflict found → Rollback; return (conflictID, nil)
//  4b. No conflict    → INSERT + Commit; return (nil, nil)
func (r *PostgresRepository) CreateBookingWithLock(ctx context.Context, booking *MachineBooking) (*uuid.UUID, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if already committed

	// Acquire advisory lock scoped to this transaction.
	lockKey := booking.TenantID.String() + ":" + booking.MachineID
	if _, lockErr := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); lockErr != nil {
		return nil, fmt.Errorf("acquire advisory lock: %w", lockErr)
	}

	// Conflict check (same logic as FindConflictingBooking but using the open Tx).
	var conflictID uuid.UUID
	scanErr := tx.QueryRow(ctx,
		`SELECT id FROM machine_bookings
		 WHERE tenant_id = $1
		   AND machine_id = $2
		   AND status IN ('booked', 'in_use')
		   AND starts_at < $4
		   AND ends_at > $3
		 LIMIT 1`,
		booking.TenantID, booking.MachineID, booking.StartsAt, booking.EndsAt,
	).Scan(&conflictID)

	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("conflict check in tx: %w", scanErr)
	}
	if scanErr == nil {
		// Conflict found — rollback and signal caller.
		_ = tx.Rollback(ctx)
		id := conflictID
		return &id, nil
	}

	// No conflict — insert.
	_, insertErr := tx.Exec(ctx,
		`INSERT INTO machine_bookings
		    (id, tenant_id, machine_id, production_order_id, starts_at, ends_at,
		     status, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		booking.ID, booking.TenantID, booking.MachineID, booking.ProductionOrderID,
		booking.StartsAt, booking.EndsAt, booking.Status, booking.Notes,
		booking.CreatedBy, booking.CreatedAt, booking.UpdatedAt,
	)
	if insertErr != nil {
		return nil, fmt.Errorf("insert booking in tx: %w", insertErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, fmt.Errorf("commit booking tx: %w", commitErr)
	}

	return nil, nil
}

// ============================================================================
// Production Plans
// ============================================================================

func (r *PostgresRepository) CreatePlan(ctx context.Context, plan *ProductionPlan) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO production_plans
		    (id, tenant_id, name, week_number, year, total_capacity_hours,
		     planned_capacity_hours, status, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		plan.ID, plan.TenantID, plan.Name, plan.WeekNumber, plan.Year,
		plan.TotalCapacityHours, plan.PlannedCapacityHours, plan.Status,
		plan.Notes, plan.CreatedBy, plan.CreatedAt, plan.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdatePlan(ctx context.Context, plan *ProductionPlan) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE production_plans
		 SET name=$1, week_number=$2, year=$3, total_capacity_hours=$4,
		     planned_capacity_hours=$5, status=$6, notes=$7, updated_at=$8
		 WHERE id=$9 AND tenant_id=$10`,
		plan.Name, plan.WeekNumber, plan.Year, plan.TotalCapacityHours,
		plan.PlannedCapacityHours, plan.Status, plan.Notes, plan.UpdatedAt,
		plan.ID, plan.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*ProductionPlan, error) {
	return r.scanPlan(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, week_number, year, total_capacity_hours,
		        planned_capacity_hours, status, notes, created_by, created_at, updated_at
		 FROM production_plans WHERE id=$1 AND tenant_id=$2`,
		planID, tenantID,
	))
}

// ============================================================================
// Capacity
// ============================================================================

func (r *PostgresRepository) GetCapacityOverview(ctx context.Context, tenantID uuid.UUID, machineID string, planID uuid.UUID) (*CapacityOverview, error) {
	// Fetch plan to get total_capacity_hours and the week's time bounds.
	plan, err := r.GetPlan(ctx, tenantID, planID)
	if err != nil {
		return nil, err
	}

	// Compute week start/end from year + week_number (ISO week).
	weekStart, weekEnd := isoWeekBounds(plan.Year, plan.WeekNumber)

	var bookedHours float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (ends_at - starts_at)) / 3600.0), 0)
		 FROM machine_bookings
		 WHERE tenant_id = $1
		   AND machine_id = $2
		   AND status IN ('booked', 'in_use', 'completed')
		   AND starts_at >= $3
		   AND ends_at <= $4`,
		tenantID, machineID, weekStart, weekEnd,
	).Scan(&bookedHours)
	if err != nil {
		return nil, fmt.Errorf("get capacity overview: %w", err)
	}

	total := plan.TotalCapacityHours
	available := total - bookedHours
	if available < 0 {
		available = 0
	}

	return &CapacityOverview{
		MachineID:          machineID,
		TotalCapacityHours: total,
		BookedHours:        bookedHours,
		AvailableHours:     available,
	}, nil
}

// isoWeekBounds returns Monday 00:00 UTC and the following Monday 00:00 UTC
// for the given ISO year and week number.
func isoWeekBounds(year, week int) (time.Time, time.Time) {
	// Jan 4 is always in week 1 per ISO 8601.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	// Offset to Monday of week 1.
	_, w1 := jan4.ISOWeek()
	_ = w1
	daysToMonday := int(jan4.Weekday()) - 1
	if daysToMonday < 0 {
		daysToMonday += 7
	}
	monday1 := jan4.AddDate(0, 0, -daysToMonday)
	weekStart := monday1.AddDate(0, 0, (week-1)*7)
	weekEnd := weekStart.AddDate(0, 0, 7)
	return weekStart, weekEnd
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanOrder(row pgx.Row) (*ProductionOrder, error) {
	var o ProductionOrder
	err := row.Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.ProductName, &o.Quantity, &o.Status,
		&o.PlannedStart, &o.PlannedEnd, &o.ActualStart, &o.ActualEnd,
		&o.Priority, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan order: %w", err)
	}
	return &o, nil
}

func (r *PostgresRepository) scanOrderFromRows(rows pgx.Rows) (*ProductionOrder, error) {
	var o ProductionOrder
	err := rows.Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.ProductName, &o.Quantity, &o.Status,
		&o.PlannedStart, &o.PlannedEnd, &o.ActualStart, &o.ActualEnd,
		&o.Priority, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan order row: %w", err)
	}
	return &o, nil
}

func (r *PostgresRepository) scanBooking(row pgx.Row) (*MachineBooking, error) {
	var b MachineBooking
	err := row.Scan(
		&b.ID, &b.TenantID, &b.MachineID, &b.ProductionOrderID,
		&b.StartsAt, &b.EndsAt, &b.Status, &b.Notes,
		&b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan booking: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) scanBookingFromRows(rows pgx.Rows) (*MachineBooking, error) {
	var b MachineBooking
	err := rows.Scan(
		&b.ID, &b.TenantID, &b.MachineID, &b.ProductionOrderID,
		&b.StartsAt, &b.EndsAt, &b.Status, &b.Notes,
		&b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan booking row: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) scanPlan(row pgx.Row) (*ProductionPlan, error) {
	var p ProductionPlan
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.WeekNumber, &p.Year,
		&p.TotalCapacityHours, &p.PlannedCapacityHours, &p.Status,
		&p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan plan: %w", err)
	}
	return &p, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
