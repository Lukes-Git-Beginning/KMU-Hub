package fuhrpark

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

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)

// ============================================================================
// Vehicles
// ============================================================================

func (r *PostgresRepository) CreateVehicle(ctx context.Context, v *Vehicle) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicles
		    (id, tenant_id, license_plate, make, model, year, vin, color, fuel_type,
		     status, mileage_km, tuev_due_date, assigned_driver_id, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		v.ID, v.TenantID, v.LicensePlate, v.Make, v.Model, v.Year,
		v.VIN, v.Color, v.FuelType, v.Status, v.MileageKm,
		v.TuevDueDate, v.AssignedDriverID, v.Notes, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE vehicles
		SET license_plate=$1, make=$2, model=$3, year=$4, vin=$5, color=$6,
		    fuel_type=$7, status=$8, mileage_km=$9, tuev_due_date=$10,
		    assigned_driver_id=$11, notes=$12, updated_at=$13
		WHERE id=$14 AND tenant_id=$15 AND deleted_at IS NULL`,
		v.LicensePlate, v.Make, v.Model, v.Year, v.VIN, v.Color,
		v.FuelType, v.Status, v.MileageKm, v.TuevDueDate,
		v.AssignedDriverID, v.Notes, v.UpdatedAt, v.ID, v.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrVehicleNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE vehicles SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		vehicleID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrVehicleNotFound
	}
	return nil
}

func (r *PostgresRepository) GetVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) (*Vehicle, error) {
	var v Vehicle
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, license_plate, make, model, year, vin, color, fuel_type,
		       status, mileage_km, tuev_due_date, tuev_reminder_sent_at,
		       assigned_driver_id, notes, created_at, updated_at, deleted_at
		FROM vehicles WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		vehicleID, tenantID,
	).Scan(
		&v.ID, &v.TenantID, &v.LicensePlate, &v.Make, &v.Model, &v.Year,
		&v.VIN, &v.Color, &v.FuelType, &v.Status, &v.MileageKm,
		&v.TuevDueDate, &v.TuevReminderSentAt, &v.AssignedDriverID, &v.Notes,
		&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVehicleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vehicle: %w", err)
	}
	return &v, nil
}

func (r *PostgresRepository) ListVehicles(ctx context.Context, tenantID uuid.UUID, filter ListVehiclesFilter, offset, limit int) ([]*Vehicle, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(license_plate) LIKE $%d OR LOWER(make) LIKE $%d OR LOWER(model) LIKE $%d)",
			argNum, argNum, argNum,
		))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.FuelType != nil {
		conditions = append(conditions, fmt.Sprintf("fuel_type = $%d", argNum))
		args = append(args, *filter.FuelType)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM vehicles %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vehicles: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, license_plate, make, model, year, vin, color, fuel_type,
		       status, mileage_km, tuev_due_date, tuev_reminder_sent_at,
		       assigned_driver_id, notes, created_at, updated_at, deleted_at
		FROM vehicles %s
		ORDER BY license_plate ASC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*Vehicle
	for rows.Next() {
		v, scanErr := r.scanVehicle(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, total, rows.Err()
}

func (r *PostgresRepository) PlateExists(ctx context.Context, tenantID uuid.UUID, plate string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM vehicles WHERE tenant_id = $1 AND license_plate = $2 AND id <> $3 AND deleted_at IS NULL)`,
			tenantID, plate, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM vehicles WHERE tenant_id = $1 AND license_plate = $2 AND deleted_at IS NULL)`,
		tenantID, plate,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// Services
// ============================================================================

func (r *PostgresRepository) CreateService(ctx context.Context, s *VehicleService) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicle_services
		    (id, tenant_id, vehicle_id, service_type, description, scheduled_at,
		     completed_at, cost_cents, workshop, mileage_km, status, notes, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		s.ID, s.TenantID, s.VehicleID, s.ServiceType, s.Description, s.ScheduledAt,
		s.CompletedAt, s.CostCents, s.Workshop, s.MileageKm, s.Status, s.Notes,
		s.CreatedBy, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateService(ctx context.Context, s *VehicleService) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE vehicle_services
		SET service_type=$1, description=$2, scheduled_at=$3, completed_at=$4,
		    cost_cents=$5, workshop=$6, mileage_km=$7, status=$8, notes=$9, updated_at=$10
		WHERE id=$11 AND tenant_id=$12`,
		s.ServiceType, s.Description, s.ScheduledAt, s.CompletedAt,
		s.CostCents, s.Workshop, s.MileageKm, s.Status, s.Notes, s.UpdatedAt,
		s.ID, s.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrServiceNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteService(ctx context.Context, tenantID, serviceID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM vehicle_services WHERE id = $1 AND tenant_id = $2`,
		serviceID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrServiceNotFound
	}
	return nil
}

func (r *PostgresRepository) GetService(ctx context.Context, tenantID, serviceID uuid.UUID) (*VehicleService, error) {
	var s VehicleService
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, vehicle_id, service_type, description, scheduled_at,
		       completed_at, cost_cents, workshop, mileage_km, status, notes, created_by, created_at, updated_at
		FROM vehicle_services WHERE id = $1 AND tenant_id = $2`,
		serviceID, tenantID,
	).Scan(
		&s.ID, &s.TenantID, &s.VehicleID, &s.ServiceType, &s.Description, &s.ScheduledAt,
		&s.CompletedAt, &s.CostCents, &s.Workshop, &s.MileageKm, &s.Status, &s.Notes,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrServiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vehicle service: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) ListServices(ctx context.Context, tenantID uuid.UUID, filter ListServicesFilter, offset, limit int) ([]*VehicleService, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.VehicleID != nil {
		conditions = append(conditions, fmt.Sprintf("vehicle_id = $%d", argNum))
		args = append(args, *filter.VehicleID)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.ScheduledBefore != nil {
		conditions = append(conditions, fmt.Sprintf("scheduled_at <= $%d", argNum))
		args = append(args, *filter.ScheduledBefore)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM vehicle_services %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count services: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, vehicle_id, service_type, description, scheduled_at,
		       completed_at, cost_cents, workshop, mileage_km, status, notes, created_by, created_at, updated_at
		FROM vehicle_services %s
		ORDER BY scheduled_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var services []*VehicleService
	for rows.Next() {
		s, scanErr := r.scanService(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		services = append(services, s)
	}
	return services, total, rows.Err()
}

// ============================================================================
// Damages
// ============================================================================

func (r *PostgresRepository) CreateDamage(ctx context.Context, d *VehicleDamage) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicle_damages
		    (id, tenant_id, vehicle_id, description, severity, status,
		     reported_by, resolved_by, resolved_at, photo_keys, cost_cents, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		d.ID, d.TenantID, d.VehicleID, d.Description, d.Severity, d.Status,
		d.ReportedBy, d.ResolvedBy, d.ResolvedAt, d.PhotoKeys, d.CostCents, d.Notes,
		d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateDamage(ctx context.Context, d *VehicleDamage) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE vehicle_damages
		SET description=$1, severity=$2, status=$3, resolved_by=$4, resolved_at=$5,
		    photo_keys=$6, cost_cents=$7, notes=$8, updated_at=$9
		WHERE id=$10 AND tenant_id=$11`,
		d.Description, d.Severity, d.Status, d.ResolvedBy, d.ResolvedAt,
		d.PhotoKeys, d.CostCents, d.Notes, d.UpdatedAt, d.ID, d.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrDamageNotFound
	}
	return nil
}

func (r *PostgresRepository) GetDamage(ctx context.Context, tenantID, damageID uuid.UUID) (*VehicleDamage, error) {
	var d VehicleDamage
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, vehicle_id, description, severity, status,
		       reported_by, resolved_by, resolved_at, photo_keys, cost_cents, notes, created_at, updated_at
		FROM vehicle_damages WHERE id = $1 AND tenant_id = $2`,
		damageID, tenantID,
	).Scan(
		&d.ID, &d.TenantID, &d.VehicleID, &d.Description, &d.Severity, &d.Status,
		&d.ReportedBy, &d.ResolvedBy, &d.ResolvedAt, &d.PhotoKeys, &d.CostCents, &d.Notes,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDamageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vehicle damage: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) ListDamages(ctx context.Context, tenantID uuid.UUID, filter ListDamagesFilter, offset, limit int) ([]*VehicleDamage, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.VehicleID != nil {
		conditions = append(conditions, fmt.Sprintf("vehicle_id = $%d", argNum))
		args = append(args, *filter.VehicleID)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM vehicle_damages %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count damages: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, vehicle_id, description, severity, status,
		       reported_by, resolved_by, resolved_at, photo_keys, cost_cents, notes, created_at, updated_at
		FROM vehicle_damages %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list damages: %w", err)
	}
	defer rows.Close()

	var damages []*VehicleDamage
	for rows.Next() {
		d, scanErr := r.scanDamage(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		damages = append(damages, d)
	}
	return damages, total, rows.Err()
}

// ============================================================================
// History
// ============================================================================

func (r *PostgresRepository) GetVehicleHistory(ctx context.Context, tenantID, vehicleID uuid.UUID, offset, limit int) ([]*HistoryEntry, int, error) {
	const countQ = `
		SELECT (
		    SELECT COUNT(*) FROM vehicle_services WHERE tenant_id = $1 AND vehicle_id = $2
		) + (
		    SELECT COUNT(*) FROM vehicle_damages WHERE tenant_id = $1 AND vehicle_id = $2
		)`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, tenantID, vehicleID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vehicle history: %w", err)
	}

	const q = `
		SELECT 'service' AS kind, id::TEXT, service_type AS summary, scheduled_at AS occurred_at
		FROM vehicle_services WHERE tenant_id = $1 AND vehicle_id = $2
		UNION ALL
		SELECT 'damage' AS kind, id::TEXT, description AS summary, created_at AS occurred_at
		FROM vehicle_damages WHERE tenant_id = $1 AND vehicle_id = $2
		ORDER BY occurred_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, q, tenantID, vehicleID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list vehicle history: %w", err)
	}
	defer rows.Close()

	var entries []*HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var idStr string
		if scanErr := rows.Scan(&e.Kind, &idStr, &e.Summary, &e.OccurredAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan history entry: %w", scanErr)
		}
		parsed, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, 0, fmt.Errorf("parse history id: %w", parseErr)
		}
		e.ID = parsed
		entries = append(entries, &e)
	}
	return entries, total, rows.Err()
}

// ============================================================================
// TUEV Cron
// ============================================================================

func (r *PostgresRepository) FindVehiclesDueTuev(ctx context.Context, from, to time.Time) ([]*Vehicle, error) {
	// Idempotency guard: only notify if we haven't sent a reminder within the last 23 hours.
	// This prevents double-notification if the cron runs twice within the same hour window.
	// Query is cross-tenant by design: the cron scans all tenants' vehicles.
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, license_plate, make, model, year, vin, color, fuel_type,
		       status, mileage_km, tuev_due_date, tuev_reminder_sent_at,
		       assigned_driver_id, notes, created_at, updated_at, deleted_at
		FROM vehicles
		WHERE deleted_at IS NULL
		  AND tuev_due_date BETWEEN $1::date AND $2::date
		  AND (tuev_reminder_sent_at IS NULL OR tuev_reminder_sent_at < NOW() - INTERVAL '23 hours')`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("find vehicles due tuev: %w", err)
	}
	defer rows.Close()

	var vehicles []*Vehicle
	for rows.Next() {
		v, scanErr := r.scanVehicle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// MarkTuevReminderSent stamps tuev_reminder_sent_at = now for a specific vehicle.
// The tenant_id guard prevents cross-tenant manipulation from a compromised cron path.
func (r *PostgresRepository) MarkTuevReminderSent(ctx context.Context, vehicleID, tenantID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE vehicles SET tuev_reminder_sent_at = NOW() WHERE id = $1 AND tenant_id = $2`,
		vehicleID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrVehicleNotFound
	}
	return nil
}

// ============================================================================
// Fuel Logs
// ============================================================================

func (r *PostgresRepository) ListFuelLogs(ctx context.Context, params ListFuelLogsParams) ([]FuelLog, int, error) {
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var logs []FuelLog
	var total int
	var err error

	zeroID := uuid.UUID{}
	if params.VehicleID != zeroID {
		rows, queryErr := r.pool.Query(ctx, `
			SELECT id, tenant_id, vehicle_id, date, liters, cost_cents, mileage_km, fuel_type, notes, created_at, updated_at
			FROM fuel_logs
			WHERE tenant_id=$1 AND vehicle_id=$2
			ORDER BY date DESC, created_at DESC
			LIMIT $3 OFFSET $4`,
			params.TenantID, params.VehicleID, pageSize, offset)
		if queryErr != nil {
			return nil, 0, fmt.Errorf("list fuel logs: %w", queryErr)
		}
		logs, err = scanFuelLogs(rows)
		if err != nil {
			return nil, 0, err
		}
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM fuel_logs WHERE tenant_id=$1 AND vehicle_id=$2`,
			params.TenantID, params.VehicleID).Scan(&total)
	} else {
		rows, queryErr := r.pool.Query(ctx, `
			SELECT id, tenant_id, vehicle_id, date, liters, cost_cents, mileage_km, fuel_type, notes, created_at, updated_at
			FROM fuel_logs
			WHERE tenant_id=$1
			ORDER BY date DESC, created_at DESC
			LIMIT $2 OFFSET $3`,
			params.TenantID, pageSize, offset)
		if queryErr != nil {
			return nil, 0, fmt.Errorf("list fuel logs: %w", queryErr)
		}
		logs, err = scanFuelLogs(rows)
		if err != nil {
			return nil, 0, err
		}
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM fuel_logs WHERE tenant_id=$1`,
			params.TenantID).Scan(&total)
	}
	return logs, total, err
}

func (r *PostgresRepository) CreateFuelLog(ctx context.Context, log FuelLog) (FuelLog, error) {
	log.ID = uuid.New()
	now := time.Now()
	log.CreatedAt = now
	log.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fuel_logs (id, tenant_id, vehicle_id, date, liters, cost_cents, mileage_km, fuel_type, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		log.ID, log.TenantID, log.VehicleID, log.Date, log.Liters,
		log.CostCents, log.MileageKm, log.FuelType, log.Notes,
		log.CreatedAt, log.UpdatedAt)
	return log, err
}

func (r *PostgresRepository) UpdateFuelLog(ctx context.Context, log FuelLog) (FuelLog, error) {
	log.UpdatedAt = time.Now()
	ct, err := r.pool.Exec(ctx, `
		UPDATE fuel_logs SET
			date=$3, liters=$4, cost_cents=$5, mileage_km=$6, fuel_type=$7, notes=$8, updated_at=$9
		WHERE id=$1 AND tenant_id=$2`,
		log.ID, log.TenantID, log.Date, log.Liters, log.CostCents,
		log.MileageKm, log.FuelType, log.Notes, log.UpdatedAt)
	if err != nil {
		return FuelLog{}, err
	}
	if ct.RowsAffected() == 0 {
		return FuelLog{}, ErrFuelLogNotFound
	}
	var updated FuelLog
	err = r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, vehicle_id, date, liters, cost_cents, mileage_km, fuel_type, notes, created_at, updated_at
		 FROM fuel_logs WHERE id=$1 AND tenant_id=$2`,
		log.ID, log.TenantID).Scan(
		&updated.ID, &updated.TenantID, &updated.VehicleID, &updated.Date,
		&updated.Liters, &updated.CostCents, &updated.MileageKm, &updated.FuelType,
		&updated.Notes, &updated.CreatedAt, &updated.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return FuelLog{}, ErrFuelLogNotFound
	}
	return updated, err
}

func (r *PostgresRepository) DeleteFuelLog(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM fuel_logs WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrFuelLogNotFound
	}
	return nil
}

// ============================================================================
// Trip Logs
// ============================================================================

func (r *PostgresRepository) ListTripLogs(ctx context.Context, params ListTripLogsParams) ([]TripLog, int, error) {
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var logs []TripLog
	var total int
	var err error

	zeroID := uuid.UUID{}
	if params.VehicleID != zeroID {
		rows, queryErr := r.pool.Query(ctx, `
			SELECT id, tenant_id, vehicle_id, date, start_location, end_location, purpose,
			       start_km, end_km, km, is_private, driver_name, notes, created_at, updated_at
			FROM trip_logs
			WHERE tenant_id=$1 AND vehicle_id=$2
			ORDER BY date DESC, created_at DESC
			LIMIT $3 OFFSET $4`,
			params.TenantID, params.VehicleID, pageSize, offset)
		if queryErr != nil {
			return nil, 0, fmt.Errorf("list trip logs: %w", queryErr)
		}
		logs, err = scanTripLogs(rows)
		if err != nil {
			return nil, 0, err
		}
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM trip_logs WHERE tenant_id=$1 AND vehicle_id=$2`,
			params.TenantID, params.VehicleID).Scan(&total)
	} else {
		rows, queryErr := r.pool.Query(ctx, `
			SELECT id, tenant_id, vehicle_id, date, start_location, end_location, purpose,
			       start_km, end_km, km, is_private, driver_name, notes, created_at, updated_at
			FROM trip_logs
			WHERE tenant_id=$1
			ORDER BY date DESC, created_at DESC
			LIMIT $2 OFFSET $3`,
			params.TenantID, pageSize, offset)
		if queryErr != nil {
			return nil, 0, fmt.Errorf("list trip logs: %w", queryErr)
		}
		logs, err = scanTripLogs(rows)
		if err != nil {
			return nil, 0, err
		}
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM trip_logs WHERE tenant_id=$1`,
			params.TenantID).Scan(&total)
	}
	return logs, total, err
}

func (r *PostgresRepository) CreateTripLog(ctx context.Context, log TripLog) (TripLog, error) {
	if log.EndKm < log.StartKm {
		return TripLog{}, ErrInvalidInput
	}
	log.ID = uuid.New()
	now := time.Now()
	log.CreatedAt = now
	log.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO trip_logs (id, tenant_id, vehicle_id, date, start_location, end_location, purpose,
		                       start_km, end_km, is_private, driver_name, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		log.ID, log.TenantID, log.VehicleID, log.Date, log.StartLocation, log.EndLocation,
		log.Purpose, log.StartKm, log.EndKm, log.IsPrivate, log.DriverName, log.Notes,
		log.CreatedAt, log.UpdatedAt)
	if err != nil {
		return TripLog{}, err
	}
	// Re-fetch to get computed km column
	var created TripLog
	err = r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, vehicle_id, date, start_location, end_location, purpose,
		        start_km, end_km, km, is_private, driver_name, notes, created_at, updated_at
		 FROM trip_logs WHERE id=$1`,
		log.ID).Scan(
		&created.ID, &created.TenantID, &created.VehicleID, &created.Date,
		&created.StartLocation, &created.EndLocation, &created.Purpose,
		&created.StartKm, &created.EndKm, &created.Km,
		&created.IsPrivate, &created.DriverName, &created.Notes,
		&created.CreatedAt, &created.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TripLog{}, ErrTripLogNotFound
	}
	return created, err
}

func (r *PostgresRepository) UpdateTripLog(ctx context.Context, log TripLog) (TripLog, error) {
	if log.EndKm < log.StartKm {
		return TripLog{}, ErrInvalidInput
	}
	log.UpdatedAt = time.Now()
	ct, err := r.pool.Exec(ctx, `
		UPDATE trip_logs SET
			date=$3, start_location=$4, end_location=$5, purpose=$6,
			start_km=$7, end_km=$8, is_private=$9, driver_name=$10, notes=$11, updated_at=$12
		WHERE id=$1 AND tenant_id=$2`,
		log.ID, log.TenantID, log.Date, log.StartLocation, log.EndLocation,
		log.Purpose, log.StartKm, log.EndKm, log.IsPrivate, log.DriverName,
		log.Notes, log.UpdatedAt)
	if err != nil {
		return TripLog{}, err
	}
	if ct.RowsAffected() == 0 {
		return TripLog{}, ErrTripLogNotFound
	}
	var updated TripLog
	err = r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, vehicle_id, date, start_location, end_location, purpose,
		        start_km, end_km, km, is_private, driver_name, notes, created_at, updated_at
		 FROM trip_logs WHERE id=$1 AND tenant_id=$2`,
		log.ID, log.TenantID).Scan(
		&updated.ID, &updated.TenantID, &updated.VehicleID, &updated.Date,
		&updated.StartLocation, &updated.EndLocation, &updated.Purpose,
		&updated.StartKm, &updated.EndKm, &updated.Km,
		&updated.IsPrivate, &updated.DriverName, &updated.Notes,
		&updated.CreatedAt, &updated.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TripLog{}, ErrTripLogNotFound
	}
	return updated, err
}

func (r *PostgresRepository) DeleteTripLog(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM trip_logs WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrTripLogNotFound
	}
	return nil
}

// ============================================================================
// Vehicle Documents
// ============================================================================

func (r *PostgresRepository) ListVehicleDocuments(ctx context.Context, params ListVehicleDocumentsParams) ([]VehicleDocument, int, error) {
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, vehicle_id, doc_type, name, object_key, upload_date, expiry_date, created_at, updated_at
		FROM vehicle_documents
		WHERE tenant_id=$1 AND vehicle_id=$2
		ORDER BY upload_date DESC, created_at DESC
		LIMIT $3 OFFSET $4`,
		params.TenantID, params.VehicleID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list vehicle documents: %w", err)
	}
	docs, err := scanVehicleDocuments(rows)
	if err != nil {
		return nil, 0, err
	}
	var total int
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM vehicle_documents WHERE tenant_id=$1 AND vehicle_id=$2`,
		params.TenantID, params.VehicleID).Scan(&total)
	return docs, total, err
}

func (r *PostgresRepository) CreateVehicleDocument(ctx context.Context, doc VehicleDocument) (VehicleDocument, error) {
	doc.ID = uuid.New()
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicle_documents (id, tenant_id, vehicle_id, doc_type, name, object_key, upload_date, expiry_date, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		doc.ID, doc.TenantID, doc.VehicleID, doc.DocType, doc.Name, doc.ObjectKey,
		doc.UploadDate, doc.ExpiryDate, doc.CreatedAt, doc.UpdatedAt)
	return doc, err
}

func (r *PostgresRepository) DeleteVehicleDocument(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM vehicle_documents WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// ============================================================================
// Driver Licenses
// ============================================================================

func (r *PostgresRepository) ListDriverLicenses(ctx context.Context, params ListDriverLicensesParams) ([]DriverLicense, int, error) {
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	hasDriverFilter := params.DriverID != uuid.Nil

	var rows pgx.Rows
	var err error
	if hasDriverFilter {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, driver_id, license_classes, expiry_date, checked_at, next_check_due_date, notes, created_at, updated_at
			FROM driver_licenses
			WHERE tenant_id=$1 AND driver_id=$2
			ORDER BY checked_at DESC, created_at DESC
			LIMIT $3 OFFSET $4`,
			params.TenantID, params.DriverID, pageSize, offset)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, driver_id, license_classes, expiry_date, checked_at, next_check_due_date, notes, created_at, updated_at
			FROM driver_licenses
			WHERE tenant_id=$1
			ORDER BY checked_at DESC, created_at DESC
			LIMIT $2 OFFSET $3`,
			params.TenantID, pageSize, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list driver licenses: %w", err)
	}
	licenses, err := scanDriverLicenses(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if hasDriverFilter {
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM driver_licenses WHERE tenant_id=$1 AND driver_id=$2`,
			params.TenantID, params.DriverID).Scan(&total)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM driver_licenses WHERE tenant_id=$1`,
			params.TenantID).Scan(&total)
	}
	return licenses, total, err
}

// CreateDriverLicense inserts through a join on users so the driver_id can
// never resolve to another tenant's account, the same defence-in-depth used
// by AssignUserRole for user_roles (which has no RLS of its own). Here
// driver_licenses already has RLS on tenant_id, but the join still prevents
// attaching a check to a driver who isn't visible in the caller's tenant.
func (r *PostgresRepository) CreateDriverLicense(ctx context.Context, lic DriverLicense) (DriverLicense, error) {
	lic.ID = uuid.New()
	now := time.Now()
	lic.CreatedAt = now
	lic.UpdatedAt = now
	ct, err := r.pool.Exec(ctx, `
		INSERT INTO driver_licenses (id, tenant_id, driver_id, license_classes, expiry_date, checked_at, next_check_due_date, notes, created_at, updated_at)
		SELECT $1, $3, u.id, $4, $5, $6, $7, $8, $9, $9
		FROM users u
		WHERE u.id = $2 AND u.tenant_id = $3`,
		lic.ID, lic.DriverID, lic.TenantID, lic.LicenseClasses, lic.ExpiryDate,
		lic.CheckedAt, lic.NextCheckDueDate, lic.Notes, lic.CreatedAt)
	if err != nil {
		return DriverLicense{}, err
	}
	if ct.RowsAffected() == 0 {
		return DriverLicense{}, ErrDriverNotFound
	}
	return lic, nil
}

func (r *PostgresRepository) UpdateDriverLicense(ctx context.Context, lic DriverLicense) (DriverLicense, error) {
	lic.UpdatedAt = time.Now()
	ct, err := r.pool.Exec(ctx, `
		UPDATE driver_licenses SET
			license_classes=$3, expiry_date=$4, checked_at=$5, next_check_due_date=$6, notes=$7, updated_at=$8
		WHERE id=$1 AND tenant_id=$2`,
		lic.ID, lic.TenantID, lic.LicenseClasses, lic.ExpiryDate, lic.CheckedAt,
		lic.NextCheckDueDate, lic.Notes, lic.UpdatedAt)
	if err != nil {
		return DriverLicense{}, err
	}
	if ct.RowsAffected() == 0 {
		return DriverLicense{}, ErrDriverLicenseNotFound
	}
	var updated DriverLicense
	err = r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, driver_id, license_classes, expiry_date, checked_at, next_check_due_date, notes, created_at, updated_at
		FROM driver_licenses WHERE id=$1 AND tenant_id=$2`,
		lic.ID, lic.TenantID).Scan(
		&updated.ID, &updated.TenantID, &updated.DriverID, &updated.LicenseClasses, &updated.ExpiryDate,
		&updated.CheckedAt, &updated.NextCheckDueDate, &updated.Notes, &updated.CreatedAt, &updated.UpdatedAt)
	return updated, err
}

func (r *PostgresRepository) DeleteDriverLicense(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM driver_licenses WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrDriverLicenseNotFound
	}
	return nil
}

func scanDriverLicenses(rows pgx.Rows) ([]DriverLicense, error) {
	defer rows.Close()
	var licenses []DriverLicense
	for rows.Next() {
		var l DriverLicense
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.DriverID, &l.LicenseClasses, &l.ExpiryDate,
			&l.CheckedAt, &l.NextCheckDueDate, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan driver license row: %w", err)
		}
		licenses = append(licenses, l)
	}
	return licenses, rows.Err()
}

// ============================================================================
// GPS Positions
// ============================================================================

func (r *PostgresRepository) IngestGpsPositions(ctx context.Context, tenantID, vehicleID uuid.UUID, positions []GpsPosition) (int, error) {
	if len(positions) == 0 {
		return 0, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	count := 0
	now := time.Now()
	for _, pos := range positions {
		pos.ID = uuid.New()
		pos.TenantID = tenantID
		pos.VehicleID = vehicleID
		pos.CreatedAt = now
		_, execErr := tx.Exec(ctx, `
			INSERT INTO gps_positions (id, tenant_id, vehicle_id, lat, lng, speed_kmh, recorded_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			pos.ID, pos.TenantID, pos.VehicleID, pos.Lat, pos.Lng, pos.SpeedKmh, pos.RecordedAt, pos.CreatedAt)
		if execErr != nil {
			return count, execErr
		}
		count++
	}
	return count, tx.Commit(ctx)
}

func (r *PostgresRepository) GetVehicleRoutes(ctx context.Context, params GetVehicleRoutesParams) ([]VehicleRouteAggregation, error) {
	type aggRow struct {
		VehicleID    uuid.UUID `json:"vehicle_id"`
		LicensePlate string    `json:"license_plate"`
		Date         time.Time `json:"date"`
		LastRecorded time.Time `json:"last_recorded_at"`
	}

	query := `
		SELECT
			g.vehicle_id,
			v.license_plate,
			DATE(g.recorded_at) AS date,
			MAX(g.recorded_at) AS last_recorded_at
		FROM gps_positions g
		JOIN vehicles v ON v.id = g.vehicle_id
		WHERE g.tenant_id=$1
		  AND DATE(g.recorded_at) BETWEEN $2 AND $3`

	args := []any{params.TenantID, params.DateFrom.Format("2006-01-02"), params.DateTo.Format("2006-01-02")}
	argIdx := 4
	zeroID := uuid.UUID{}
	if params.VehicleID != zeroID {
		query += fmt.Sprintf(" AND g.vehicle_id=$%d", argIdx)
		args = append(args, params.VehicleID)
	}
	query += " GROUP BY g.vehicle_id, v.license_plate, DATE(g.recorded_at) ORDER BY date DESC, g.vehicle_id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get vehicle routes: %w", err)
	}
	defer rows.Close()

	var aggRows []aggRow
	for rows.Next() {
		var row aggRow
		if scanErr := rows.Scan(&row.VehicleID, &row.LicensePlate, &row.Date, &row.LastRecorded); scanErr != nil {
			return nil, fmt.Errorf("scan vehicle route row: %w", scanErr)
		}
		aggRows = append(aggRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var routes []VehicleRouteAggregation
	for _, row := range aggRows {
		status := "parked"
		if time.Since(row.LastRecorded) < 5*time.Minute {
			status = "driving"
		}
		posRows, posErr := r.pool.Query(ctx,
			`SELECT id, tenant_id, vehicle_id, lat, lng, speed_kmh, recorded_at, created_at
			 FROM gps_positions
			 WHERE tenant_id=$1 AND vehicle_id=$2 AND DATE(recorded_at)=$3
			 ORDER BY recorded_at ASC`,
			params.TenantID, row.VehicleID, row.Date.Format("2006-01-02"))
		var positions []GpsPosition
		if posErr == nil {
			positions, _ = scanGpsPositions(posRows)
		}
		routes = append(routes, VehicleRouteAggregation{
			VehicleID:   row.VehicleID,
			VehicleName: row.LicensePlate,
			Date:        row.Date,
			DailyKm:     0, // calculated from positions if needed
			Status:      status,
			Positions:   positions,
		})
	}
	return routes, nil
}

func (r *PostgresRepository) GetGpsPositions(ctx context.Context, params GetGpsPositionsParams) ([]GpsPosition, error) {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, vehicle_id, lat, lng, speed_kmh, recorded_at, created_at
		FROM gps_positions
		WHERE tenant_id=$1 AND vehicle_id=$2
		  AND recorded_at BETWEEN $3 AND $4
		ORDER BY recorded_at ASC
		LIMIT $5`,
		params.TenantID, params.VehicleID, params.From, params.To, limit)
	if err != nil {
		return nil, fmt.Errorf("get gps positions: %w", err)
	}
	return scanGpsPositions(rows)
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanVehicle(rows pgx.Rows) (*Vehicle, error) {
	var v Vehicle
	err := rows.Scan(
		&v.ID, &v.TenantID, &v.LicensePlate, &v.Make, &v.Model, &v.Year,
		&v.VIN, &v.Color, &v.FuelType, &v.Status, &v.MileageKm,
		&v.TuevDueDate, &v.TuevReminderSentAt, &v.AssignedDriverID, &v.Notes,
		&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan vehicle row: %w", err)
	}
	return &v, nil
}

func (r *PostgresRepository) scanService(rows pgx.Rows) (*VehicleService, error) {
	var s VehicleService
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.VehicleID, &s.ServiceType, &s.Description, &s.ScheduledAt,
		&s.CompletedAt, &s.CostCents, &s.Workshop, &s.MileageKm, &s.Status, &s.Notes,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan service row: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) scanDamage(rows pgx.Rows) (*VehicleDamage, error) {
	var d VehicleDamage
	err := rows.Scan(
		&d.ID, &d.TenantID, &d.VehicleID, &d.Description, &d.Severity, &d.Status,
		&d.ReportedBy, &d.ResolvedBy, &d.ResolvedAt, &d.PhotoKeys, &d.CostCents, &d.Notes,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan damage row: %w", err)
	}
	return &d, nil
}

func scanFuelLogs(rows pgx.Rows) ([]FuelLog, error) {
	defer rows.Close()
	var logs []FuelLog
	for rows.Next() {
		var l FuelLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.VehicleID, &l.Date,
			&l.Liters, &l.CostCents, &l.MileageKm, &l.FuelType,
			&l.Notes, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan fuel log row: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func scanTripLogs(rows pgx.Rows) ([]TripLog, error) {
	defer rows.Close()
	var logs []TripLog
	for rows.Next() {
		var l TripLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.VehicleID, &l.Date,
			&l.StartLocation, &l.EndLocation, &l.Purpose,
			&l.StartKm, &l.EndKm, &l.Km,
			&l.IsPrivate, &l.DriverName, &l.Notes,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan trip log row: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func scanVehicleDocuments(rows pgx.Rows) ([]VehicleDocument, error) {
	defer rows.Close()
	var docs []VehicleDocument
	for rows.Next() {
		var d VehicleDocument
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.VehicleID, &d.DocType, &d.Name, &d.ObjectKey,
			&d.UploadDate, &d.ExpiryDate, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan vehicle document row: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func scanGpsPositions(rows pgx.Rows) ([]GpsPosition, error) {
	defer rows.Close()
	var positions []GpsPosition
	for rows.Next() {
		var p GpsPosition
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.VehicleID, &p.Lat, &p.Lng,
			&p.SpeedKmh, &p.RecordedAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan gps position row: %w", err)
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// ============================================================================
// Bookings
// ============================================================================

// bookingColumns is the single source of truth for the SELECT list, so a new
// column cannot land in one read and be forgotten in the other.
const bookingColumns = `id, tenant_id, vehicle_id, user_id, starts_at, ends_at,
	       purpose, status, created_by, created_at, updated_at`

// activeBookingStatusSQL is the set of states that occupy the vehicle.
// Completed and cancelled bookings release it again.
const activeBookingStatusSQL = `('booked', 'in_use')`

// bookingOverlapPredicate is the half-open [starts_at, ends_at) test: A overlaps
// B iff A.starts_at < B.ends_at AND A.ends_at > B.starts_at. Two bookings that
// only touch at the boundary (ends_at == starts_at) do NOT overlap -- handing
// the keys over at 12:00 has to stay possible. Same reasoning as
// produktion.FindConflictingBooking, which also explains why this is arithmetic
// rather than tstzrange && : btree_gist is not guaranteed to be installed in
// every deployment target.
//
// Placeholders are fixed at $3 = startsAt, $4 = endsAt in both callers.
const bookingOverlapPredicate = `starts_at < $4 AND ends_at > $3`

func (r *PostgresRepository) CreateBookingWithLock(ctx context.Context, b *VehicleBooking) (*uuid.UUID, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin booking tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	// Serialises concurrent bookings of the same vehicle. Without it two
	// requests can both pass the conflict check and both insert.
	lockKey := b.TenantID.String() + ":" + b.VehicleID.String()
	if _, lockErr := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); lockErr != nil {
		return nil, fmt.Errorf("acquire booking lock: %w", lockErr)
	}

	var conflictID uuid.UUID
	scanErr := tx.QueryRow(ctx,
		`SELECT id FROM vehicle_bookings
		 WHERE tenant_id = $1
		   AND vehicle_id = $2
		   AND status IN `+activeBookingStatusSQL+`
		   AND `+bookingOverlapPredicate+`
		 LIMIT 1`,
		b.TenantID, b.VehicleID, b.StartsAt, b.EndsAt,
	).Scan(&conflictID)
	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("booking conflict check: %w", scanErr)
	}
	if scanErr == nil {
		id := conflictID
		return &id, nil
	}

	// INSERT ... SELECT FROM users is the guard CreateDriverLicense already
	// uses: it refuses to book a vehicle for a user who is not visible in the
	// calling tenant, which the plain FK on users would happily allow.
	ct, insertErr := tx.Exec(ctx,
		`INSERT INTO vehicle_bookings
		    (id, tenant_id, vehicle_id, user_id, starts_at, ends_at, purpose, status,
		     created_by, created_at, updated_at)
		 SELECT $1, $2, $3, u.id, $5, $6, $7, $8, $9, $10, $11
		 FROM users u
		 WHERE u.id = $4 AND u.tenant_id = $2`,
		b.ID, b.TenantID, b.VehicleID, b.UserID, b.StartsAt, b.EndsAt,
		b.Purpose, string(b.Status), b.CreatedBy, b.CreatedAt, b.UpdatedAt,
	)
	if insertErr != nil {
		return nil, fmt.Errorf("insert booking: %w", insertErr)
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrDriverNotFound
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, fmt.Errorf("commit booking tx: %w", commitErr)
	}
	return nil, nil
}

func (r *PostgresRepository) FindConflictingBooking(ctx context.Context, tenantID, vehicleID uuid.UUID, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error) {
	query := `SELECT id FROM vehicle_bookings
		 WHERE tenant_id = $1
		   AND vehicle_id = $2
		   AND status IN ` + activeBookingStatusSQL + `
		   AND ` + bookingOverlapPredicate
	args := []any{tenantID, vehicleID, startsAt, endsAt}
	// Separate branches instead of a zero-UUID sentinel: substituting
	// uuid.UUID{} would silently exclude a row that happens to carry it.
	if excludeID != nil {
		query += ` AND id <> $5`
		args = append(args, *excludeID)
	}
	query += ` LIMIT 1`

	var conflictID uuid.UUID
	err := r.pool.QueryRow(ctx, query, args...).Scan(&conflictID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find conflicting booking: %w", err)
	}
	return &conflictID, nil
}

func (r *PostgresRepository) UpdateBooking(ctx context.Context, b *VehicleBooking) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE vehicle_bookings
		SET starts_at = $3, ends_at = $4, purpose = $5, status = $6, updated_at = $7
		WHERE id = $1 AND tenant_id = $2`,
		b.ID, b.TenantID, b.StartsAt, b.EndsAt, b.Purpose, string(b.Status), b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update booking: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrBookingNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteBooking(ctx context.Context, tenantID, bookingID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM vehicle_bookings WHERE id = $1 AND tenant_id = $2`,
		bookingID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete booking: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrBookingNotFound
	}
	return nil
}

func (r *PostgresRepository) GetBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*VehicleBooking, error) {
	var b VehicleBooking
	err := r.pool.QueryRow(ctx,
		`SELECT `+bookingColumns+`
		 FROM vehicle_bookings WHERE id = $1 AND tenant_id = $2`,
		bookingID, tenantID,
	).Scan(
		&b.ID, &b.TenantID, &b.VehicleID, &b.UserID, &b.StartsAt, &b.EndsAt,
		&b.Purpose, &b.Status, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) ListBookings(ctx context.Context, params ListVehicleBookingsParams) ([]*VehicleBooking, int, error) {
	where := []string{"tenant_id = $1"}
	args := []any{params.TenantID}
	argNum := 2

	if params.VehicleID != uuid.Nil {
		where = append(where, fmt.Sprintf("vehicle_id = $%d", argNum))
		args = append(args, params.VehicleID)
		argNum++
	}
	if params.UserID != uuid.Nil {
		where = append(where, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, params.UserID)
		argNum++
	}
	if params.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*params.Status))
		argNum++
	}
	// Window filter on the interval, not on starts_at alone: a booking that
	// began before `from` and is still running belongs in the window.
	if params.From != nil {
		where = append(where, fmt.Sprintf("ends_at > $%d", argNum))
		args = append(args, *params.From)
		argNum++
	}
	if params.To != nil {
		where = append(where, fmt.Sprintf("starts_at < $%d", argNum))
		args = append(args, *params.To)
		argNum++
	}
	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM vehicle_bookings "+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query := fmt.Sprintf(`SELECT %s FROM vehicle_bookings %s
		ORDER BY starts_at ASC
		LIMIT $%d OFFSET $%d`, bookingColumns, whereClause, argNum, argNum+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	bookings := make([]*VehicleBooking, 0)
	for rows.Next() {
		var b VehicleBooking
		if scanErr := rows.Scan(
			&b.ID, &b.TenantID, &b.VehicleID, &b.UserID, &b.StartsAt, &b.EndsAt,
			&b.Purpose, &b.Status, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan booking row: %w", scanErr)
		}
		bookings = append(bookings, &b)
	}
	return bookings, total, rows.Err()
}
