package vermietung

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgErrCodeExclusionViolation is the SQLSTATE Postgres raises when the
// uq_rentals_no_overlap GIST exclusion constraint (migration 000101) rejects
// an overlapping booking. The service pre-checks HasOverlap before INSERT,
// but that check and the INSERT are not in the same transaction, so two
// concurrent CreateRental/UpdateRental calls can both pass the pre-check and
// race to the constraint. Without this mapping the loser surfaces a raw
// Postgres error instead of ErrRentalConflict.
const pgErrCodeExclusionViolation = "23P01"

func asRentalConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeExclusionViolation {
		return ErrRentalConflict
	}
	return err
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Objects
// ============================================================================

func (r *PostgresRepository) CreateObject(ctx context.Context, obj *RentalObject) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rental_objects
		    (id, tenant_id, name, description, category, daily_rate, deposit, location, notes, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		obj.ID, obj.TenantID, obj.Name, obj.Description, obj.Category,
		obj.DailyRate, obj.Deposit, obj.Location, obj.Notes, obj.Active,
		obj.CreatedAt, obj.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateObject(ctx context.Context, obj *RentalObject) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE rental_objects
		 SET name = $1, description = $2, category = $3, daily_rate = $4, deposit = $5,
		     location = $6, notes = $7, active = $8, updated_at = $9
		 WHERE id = $10 AND tenant_id = $11 AND deleted_at IS NULL`,
		obj.Name, obj.Description, obj.Category, obj.DailyRate, obj.Deposit,
		obj.Location, obj.Notes, obj.Active, obj.UpdatedAt, obj.ID, obj.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrObjectNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteObject(ctx context.Context, tenantID, objectID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE rental_objects SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		objectID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrObjectNotFound
	}
	return nil
}

func (r *PostgresRepository) GetObject(ctx context.Context, tenantID, objectID uuid.UUID) (*RentalObject, error) {
	var obj RentalObject
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, category, daily_rate, deposit, location, notes, active,
		        created_at, updated_at, deleted_at
		 FROM rental_objects WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		objectID, tenantID,
	).Scan(
		&obj.ID, &obj.TenantID, &obj.Name, &obj.Description, &obj.Category,
		&obj.DailyRate, &obj.Deposit, &obj.Location, &obj.Notes, &obj.Active,
		&obj.CreatedAt, &obj.UpdatedAt, &obj.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rental object: %w", err)
	}
	return &obj, nil
}

func (r *PostgresRepository) ListObjects(ctx context.Context, tenantID uuid.UUID, filter ListObjectsFilter, offset, limit int) ([]*RentalObject, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argNum))
		args = append(args, *filter.Category)
		argNum++
	}

	if filter.ActiveOnly {
		conditions = append(conditions, "active = TRUE")
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM rental_objects %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rental objects: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, category, daily_rate, deposit, location, notes, active,
		       created_at, updated_at, deleted_at
		FROM rental_objects %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rental objects: %w", err)
	}
	defer rows.Close()

	var objects []*RentalObject
	for rows.Next() {
		obj, scanErr := r.scanObjectFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		objects = append(objects, obj)
	}

	return objects, total, rows.Err()
}

// ============================================================================
// Rentals
// ============================================================================

func (r *PostgresRepository) CreateRental(ctx context.Context, rental *Rental) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rentals
		    (id, tenant_id, object_id, contact_id, renter_name, start_date, end_date,
		     status, total_price, deposit_paid, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		rental.ID, rental.TenantID, rental.ObjectID, rental.ContactID, rental.RenterName,
		rental.StartDate, rental.EndDate, rental.Status, rental.TotalPrice, rental.DepositPaid,
		rental.Notes, rental.CreatedAt, rental.UpdatedAt,
	)
	if err != nil {
		return asRentalConflict(err)
	}
	return nil
}

func (r *PostgresRepository) UpdateRental(ctx context.Context, rental *Rental) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE rentals
		 SET renter_name = $1, start_date = $2, end_date = $3, status = $4,
		     total_price = $5, deposit_paid = $6, notes = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10`,
		rental.RenterName, rental.StartDate, rental.EndDate, rental.Status,
		rental.TotalPrice, rental.DepositPaid, rental.Notes, rental.UpdatedAt,
		rental.ID, rental.TenantID,
	)
	if err != nil {
		return asRentalConflict(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrRentalNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteRental(ctx context.Context, tenantID, rentalID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM rentals WHERE id = $1 AND tenant_id = $2`,
		rentalID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrRentalNotFound
	}
	return nil
}

func (r *PostgresRepository) GetRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error) {
	var rental Rental
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, object_id, contact_id, renter_name, start_date, end_date,
		        status, total_price, deposit_paid, notes,
		        signature_data, signed_at, signed_by,
		        created_at, updated_at
		 FROM rentals WHERE id = $1 AND tenant_id = $2`,
		rentalID, tenantID,
	).Scan(
		&rental.ID, &rental.TenantID, &rental.ObjectID, &rental.ContactID, &rental.RenterName,
		&rental.StartDate, &rental.EndDate, &rental.Status, &rental.TotalPrice, &rental.DepositPaid,
		&rental.Notes, &rental.SignatureData, &rental.SignedAt, &rental.SignedBy,
		&rental.CreatedAt, &rental.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRentalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rental: %w", err)
	}
	return &rental, nil
}

func (r *PostgresRepository) ListRentals(ctx context.Context, tenantID uuid.UUID, filter ListRentalsFilter, offset, limit int) ([]*Rental, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.ObjectID != nil {
		conditions = append(conditions, fmt.Sprintf("object_id = $%d", argNum))
		args = append(args, *filter.ObjectID)
		argNum++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("end_date >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("start_date <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM rentals %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rentals: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, object_id, contact_id, renter_name, start_date, end_date,
		       status, total_price, deposit_paid, notes,
		       signature_data, signed_at, signed_by,
		       created_at, updated_at
		FROM rentals %s
		ORDER BY start_date ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rentals: %w", err)
	}
	defer rows.Close()

	var rentals []*Rental
	for rows.Next() {
		rental, scanErr := r.scanRentalFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		rentals = append(rentals, rental)
	}

	return rentals, total, rows.Err()
}

// HasOverlap checks for date-range overlaps using the GIST index.
// end_date is treated as exclusive (convention: booking ends at end_date 00:00 = day before is last night).
// The tstzrange overlap operator && returns true when ranges share any point.
// We use [start_date, end_date) (lower-inclusive, upper-exclusive) consistent with the migration.
func (r *PostgresRepository) HasOverlap(ctx context.Context, tenantID, objectID uuid.UUID, start, end time.Time, excludeRentalID *uuid.UUID) (bool, error) {
	var exists bool

	if excludeRentalID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(
			    SELECT 1 FROM rentals
			    WHERE tenant_id = $1
			      AND object_id = $2
			      AND status NOT IN ('cancelled')
			      AND id <> $3
			      AND tstzrange(start_date, end_date) && tstzrange($4, $5)
			)`,
			tenantID, objectID, *excludeRentalID, start, end,
		).Scan(&exists)
		return exists, err
	}

	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
		    SELECT 1 FROM rentals
		    WHERE tenant_id = $1
		      AND object_id = $2
		      AND status NOT IN ('cancelled')
		      AND tstzrange(start_date, end_date) && tstzrange($3, $4)
		)`,
		tenantID, objectID, start, end,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// Inspections
// ============================================================================

func (r *PostgresRepository) CreateInspection(ctx context.Context, ins *RentalInspection) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rental_inspections
		    (id, tenant_id, rental_id, kind, notes, photo_urls, performed_by, signature_data, checklist, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		ins.ID, ins.TenantID, ins.RentalID, ins.Kind, ins.Notes,
		ins.PhotoURLs, ins.PerformedBy, ins.SignatureData, ins.Checklist, ins.CreatedAt, ins.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateInspection(ctx context.Context, ins *RentalInspection) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE rental_inspections
		 SET notes = $1, photo_urls = $2, signature_data = $3, checklist = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		ins.Notes, ins.PhotoURLs, ins.SignatureData, ins.Checklist, ins.UpdatedAt, ins.ID, ins.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

func (r *PostgresRepository) GetInspection(ctx context.Context, tenantID, inspectionID uuid.UUID) (*RentalInspection, error) {
	var ins RentalInspection
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, rental_id, kind, notes, photo_urls, performed_by, signature_data, checklist, created_at, updated_at
		 FROM rental_inspections WHERE id = $1 AND tenant_id = $2`,
		inspectionID, tenantID,
	).Scan(
		&ins.ID, &ins.TenantID, &ins.RentalID, &ins.Kind, &ins.Notes,
		&ins.PhotoURLs, &ins.PerformedBy, &ins.SignatureData, &ins.Checklist, &ins.CreatedAt, &ins.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInspectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rental inspection: %w", err)
	}
	return &ins, nil
}

func (r *PostgresRepository) ListInspections(ctx context.Context, tenantID, rentalID uuid.UUID, offset, limit int) ([]*RentalInspection, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rental_inspections WHERE tenant_id = $1 AND rental_id = $2`,
		tenantID, rentalID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inspections: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, rental_id, kind, notes, photo_urls, performed_by, signature_data, checklist, created_at, updated_at
		 FROM rental_inspections
		 WHERE tenant_id = $1 AND rental_id = $2
		 ORDER BY created_at ASC
		 LIMIT $3 OFFSET $4`,
		tenantID, rentalID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list inspections: %w", err)
	}
	defer rows.Close()

	var inspections []*RentalInspection
	for rows.Next() {
		ins, scanErr := r.scanInspectionFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		inspections = append(inspections, ins)
	}

	return inspections, total, rows.Err()
}

// GetInspectionByKind retrieves an inspection by rental + kind (unique per rental).
// Returns ErrInspectionNotFound when no such inspection exists.
func (r *PostgresRepository) GetInspectionByKind(ctx context.Context, tenantID, rentalID uuid.UUID, kind InspectionKind) (*RentalInspection, error) {
	var ins RentalInspection
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, rental_id, kind, notes, photo_urls, performed_by, signature_data, checklist, created_at, updated_at
		 FROM rental_inspections WHERE tenant_id = $1 AND rental_id = $2 AND kind = $3`,
		tenantID, rentalID, kind,
	).Scan(
		&ins.ID, &ins.TenantID, &ins.RentalID, &ins.Kind, &ins.Notes,
		&ins.PhotoURLs, &ins.PerformedBy, &ins.SignatureData, &ins.Checklist, &ins.CreatedAt, &ins.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInspectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inspection by kind: %w", err)
	}
	return &ins, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanObjectFromRows(rows pgx.Rows) (*RentalObject, error) {
	var obj RentalObject
	err := rows.Scan(
		&obj.ID, &obj.TenantID, &obj.Name, &obj.Description, &obj.Category,
		&obj.DailyRate, &obj.Deposit, &obj.Location, &obj.Notes, &obj.Active,
		&obj.CreatedAt, &obj.UpdatedAt, &obj.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan rental object row: %w", err)
	}
	return &obj, nil
}

func (r *PostgresRepository) scanRentalFromRows(rows pgx.Rows) (*Rental, error) {
	var rental Rental
	err := rows.Scan(
		&rental.ID, &rental.TenantID, &rental.ObjectID, &rental.ContactID, &rental.RenterName,
		&rental.StartDate, &rental.EndDate, &rental.Status, &rental.TotalPrice, &rental.DepositPaid,
		&rental.Notes, &rental.SignatureData, &rental.SignedAt, &rental.SignedBy,
		&rental.CreatedAt, &rental.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan rental row: %w", err)
	}
	return &rental, nil
}

func (r *PostgresRepository) scanInspectionFromRows(rows pgx.Rows) (*RentalInspection, error) {
	var ins RentalInspection
	err := rows.Scan(
		&ins.ID, &ins.TenantID, &ins.RentalID, &ins.Kind, &ins.Notes,
		&ins.PhotoURLs, &ins.PerformedBy, &ins.SignatureData, &ins.Checklist, &ins.CreatedAt, &ins.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan rental inspection row: %w", err)
	}
	return &ins, nil
}

// SaveSignature updates a rental's signature fields and returns the updated record.
// Returns ErrRentalNotFound when no rental with the given id+tenant_id exists.
func (r *PostgresRepository) SaveSignature(ctx context.Context, tenantID, rentalID, signatureData, signedBy string) (*Rental, error) {
	var rental Rental
	err := r.pool.QueryRow(ctx,
		`UPDATE rentals
		 SET signature_data = $3, signed_at = NOW(), signed_by = $4, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, object_id, contact_id, renter_name, start_date, end_date,
		           status, total_price, deposit_paid, notes,
		           signature_data, signed_at, signed_by,
		           created_at, updated_at`,
		rentalID, tenantID, signatureData, signedBy,
	).Scan(
		&rental.ID, &rental.TenantID, &rental.ObjectID, &rental.ContactID, &rental.RenterName,
		&rental.StartDate, &rental.EndDate, &rental.Status, &rental.TotalPrice, &rental.DepositPaid,
		&rental.Notes, &rental.SignatureData, &rental.SignedAt, &rental.SignedBy,
		&rental.CreatedAt, &rental.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRentalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("save rental signature: %w", err)
	}
	return &rental, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
