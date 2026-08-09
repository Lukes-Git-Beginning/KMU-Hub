package changerequest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// selectColumns keeps every read on the same shape. The two display names are
// resolved by join rather than stored: a renamed user has to show up under the
// new name everywhere, including in requests decided years ago.
const selectColumns = `
	cr.id, cr.tenant_id, cr.user_id, cr.drawer, cr.field, cr.field_label,
	cr.old_value, cr.new_value, cr.status, cr.reason,
	cr.created_at, cr.decided_at, cr.decided_by,
	COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.first_name, u.last_name)), ''), u.email, '') AS user_name,
	COALESCE(NULLIF(TRIM(CONCAT_WS(' ', d.first_name, d.last_name)), ''), d.email, '') AS decided_by_name`

const fromClause = `
	FROM hr_profile_change_requests cr
	LEFT JOIN users u ON u.id = cr.user_id
	LEFT JOIN users d ON d.id = cr.decided_by`

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL change request repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequest(row rowScanner) (*models.HRProfileChangeRequest, error) {
	var r models.HRProfileChangeRequest
	err := row.Scan(
		&r.ID, &r.TenantID, &r.UserID, &r.Drawer, &r.Field, &r.FieldLabel,
		&r.OldValue, &r.NewValue, &r.Status, &r.Reason,
		&r.CreatedAt, &r.DecidedAt, &r.DecidedBy,
		&r.UserName, &r.DecidedByName,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *PostgresRepository) Create(ctx context.Context, req *models.HRProfileChangeRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hr_profile_change_requests (
			id, tenant_id, user_id, drawer, field, field_label,
			old_value, new_value, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		req.ID, req.TenantID, req.UserID, req.Drawer, req.Field, req.FieldLabel,
		req.OldValue, req.NewValue, req.Status, req.CreatedAt,
	)
	// The partial unique index is the real guard against a second open proposal
	// for the same field — two concurrent submits both pass a preceding SELECT.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrPendingRequestExists
	}
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRProfileChangeRequest, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT`+selectColumns+fromClause+`
		 WHERE cr.id = $1 AND cr.tenant_id = $2`,
		id, tenantID,
	)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return req, err
}

func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]*models.HRProfileChangeRequest, error) {
	// Every parameter is always bound so the statement text stays constant; a
	// NULL parameter disables its predicate.
	var ownerID any
	if filter.OwnerID != nil {
		ownerID = *filter.OwnerID
	}
	var statusFilter any
	if filter.Status != "" {
		statusFilter = filter.Status
	}
	var managerID any
	if filter.ManagerID != nil {
		managerID = *filter.ManagerID
	}

	rows, err := r.pool.Query(ctx,
		`SELECT`+selectColumns+fromClause+`
		 WHERE cr.tenant_id = $1
		   AND ($2::uuid IS NULL OR cr.user_id = $2)
		   AND ($3::text IS NULL OR cr.status = $3)
		   AND ($4::uuid IS NULL OR cr.user_id = $4 OR cr.user_id IN (
		         SELECT ep.user_id FROM hr_employee_profiles ep
		          WHERE ep.tenant_id = $1 AND ep.manager_user_id = $4))
		 ORDER BY cr.created_at DESC`,
		filter.TenantID, ownerID, statusFilter, managerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*models.HRProfileChangeRequest, 0)
	for rows.Next() {
		req, scanErr := scanRequest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *PostgresRepository) ApproveAndApply(
	ctx context.Context,
	tenantID, id, decidedBy uuid.UUID,
	column, value string,
	decidedAt time.Time,
) (*models.HRProfileChangeRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Claim the request first. The WHERE on status is the concurrency guard:
	// a second approver arriving at the same moment updates zero rows and gets
	// ErrNotPending instead of applying the value twice.
	var userID uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE hr_profile_change_requests
		    SET status = 'approved', decided_at = $3, decided_by = $4
		  WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
		  RETURNING user_id`,
		id, tenantID, decidedAt, decidedBy,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotPending
	}
	if err != nil {
		return nil, err
	}

	// column is an allowlisted identifier from the service, never client input.
	tag, err := tx.Exec(ctx,
		fmt.Sprintf(
			`UPDATE hr_employee_profiles
			    SET %s = $3, updated_at = NOW()
			  WHERE tenant_id = $1 AND user_id = $2`, column),
		tenantID, userID, value,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		// No profile to write to. Rolling back keeps the request pending rather
		// than reporting an approval that changed nothing.
		return nil, ErrProfileNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *PostgresRepository) Decide(
	ctx context.Context,
	tenantID, id uuid.UUID,
	status models.HRChangeRequestStatus,
	reason string,
	decidedBy *uuid.UUID,
	decidedAt *time.Time,
) (*models.HRProfileChangeRequest, error) {
	var updatedID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`UPDATE hr_profile_change_requests
		    SET status = $3, reason = $4, decided_at = $5, decided_by = $6
		  WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
		  RETURNING id`,
		id, tenantID, status, reason, decidedAt, decidedBy,
	).Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotPending
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *PostgresRepository) ManagerOf(ctx context.Context, tenantID, userID uuid.UUID) (*uuid.UUID, error) {
	var managerID *uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT manager_user_id FROM hr_employee_profiles
		  WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&managerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return managerID, err
}
