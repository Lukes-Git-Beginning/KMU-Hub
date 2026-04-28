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
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, license_plate, make, model, year, vin, color, fuel_type,
		       status, mileage_km, tuev_due_date, tuev_reminder_sent_at,
		       assigned_driver_id, notes, created_at, updated_at, deleted_at
		FROM vehicles
		WHERE deleted_at IS NULL
		  AND tuev_due_date BETWEEN $1 AND $2
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

func (r *PostgresRepository) MarkTuevReminderSent(ctx context.Context, vehicleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vehicles SET tuev_reminder_sent_at = NOW() WHERE id = $1`,
		vehicleID,
	)
	return err
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
