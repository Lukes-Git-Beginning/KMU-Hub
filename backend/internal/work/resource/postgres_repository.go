package resource

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

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, resource *models.Resource) error {
	query := `
		INSERT INTO resources (id, tenant_id, name, resource_type, capacity, floor, location, description, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		resource.ID, resource.TenantID, resource.Name, resource.ResourceType,
		resource.Capacity, resource.Floor, resource.Location, resource.Description,
		resource.IsActive, resource.CreatedBy,
		resource.CreatedAt, resource.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Resource, error) {
	query := `
		SELECT r.id, r.tenant_id, r.name, r.resource_type, r.capacity, r.floor, r.location, r.description,
		       r.is_active, r.created_by, r.created_at, r.updated_at,
		       COALESCE(array_agg(rt.tag) FILTER (WHERE rt.tag IS NOT NULL), '{}') AS tags
		FROM resources r
		LEFT JOIN resource_tags rt ON rt.resource_id = r.id
		WHERE r.id = $1 AND r.tenant_id = $2
		GROUP BY r.id`

	var res models.Resource
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&res.ID, &res.TenantID, &res.Name, &res.ResourceType,
		&res.Capacity, &res.Floor, &res.Location, &res.Description,
		&res.IsActive, &res.CreatedBy,
		&res.CreatedAt, &res.UpdatedAt,
		&res.Tags,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}
	return &res, nil
}

func (r *PostgresRepository) List(ctx context.Context, filters ResourceFilters) ([]models.Resource, error) {
	var conditions []string
	var args []any
	argN := 1

	// Tenant isolation mandatory
	conditions = append(conditions, fmt.Sprintf("r.tenant_id = $%d", argN))
	args = append(args, filters.TenantID)
	argN++

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("r.resource_type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}
	if filters.MinCapacity != nil {
		conditions = append(conditions, fmt.Sprintf("r.capacity >= $%d", argN))
		args = append(args, *filters.MinCapacity)
		argN++
	}
	if filters.Floor != nil {
		conditions = append(conditions, fmt.Sprintf("r.floor = $%d", argN))
		args = append(args, *filters.Floor)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("r.is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}
	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT resource_id FROM resource_tags WHERE tag = ANY($%d))", argN))
		args = append(args, filters.Tags)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.tenant_id, r.name, r.resource_type, r.capacity, r.floor, r.location, r.description,
		       r.is_active, r.created_by, r.created_at, r.updated_at,
		       COALESCE(array_agg(rt.tag) FILTER (WHERE rt.tag IS NOT NULL), '{}') AS tags
		FROM resources r
		LEFT JOIN resource_tags rt ON rt.resource_id = r.id
		%s
		GROUP BY r.id
		ORDER BY r.name`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	var resources []models.Resource
	for rows.Next() {
		var res models.Resource
		if scanErr := rows.Scan(
			&res.ID, &res.TenantID, &res.Name, &res.ResourceType,
			&res.Capacity, &res.Floor, &res.Location, &res.Description,
			&res.IsActive, &res.CreatedBy,
			&res.CreatedAt, &res.UpdatedAt,
			&res.Tags,
		); scanErr != nil {
			return nil, fmt.Errorf("scan resource: %w", scanErr)
		}
		resources = append(resources, res)
	}
	return resources, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, resource *models.Resource) error {
	query := `
		UPDATE resources
		SET name = $3, resource_type = $4, capacity = $5, floor = $6,
		    location = $7, description = $8, is_active = $9, updated_at = $10
		WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		resource.ID, resource.TenantID, resource.Name, resource.ResourceType,
		resource.Capacity, resource.Floor, resource.Location, resource.Description,
		resource.IsActive, resource.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `UPDATE resources SET is_active = false, updated_at = $3 WHERE id = $1 AND tenant_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete resource: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (r *PostgresRepository) SetTags(ctx context.Context, resourceID uuid.UUID, tags []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `DELETE FROM resource_tags WHERE resource_id = $1`, resourceID)
	if err != nil {
		return fmt.Errorf("delete tags: %w", err)
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			_, err = tx.Exec(ctx,
				`INSERT INTO resource_tags (resource_id, tag, tenant_id)
				 VALUES ($1, $2, (SELECT tenant_id FROM resources WHERE id = $1))`,
				resourceID, tag)
			if err != nil {
				return fmt.Errorf("insert tag: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) CreateBooking(ctx context.Context, booking *models.ResourceBooking) error {
	query := `
		INSERT INTO resource_bookings (id, tenant_id, resource_id, event_id, booked_by, start_time, end_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		booking.ID, booking.TenantID, booking.ResourceID, booking.EventID,
		booking.BookedBy, booking.StartTime, booking.EndTime,
		booking.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23P01 = exclusion_violation (from EXCLUDE USING GIST constraint)
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
			return ErrBookingConflict
		}
		return fmt.Errorf("create booking: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CancelBooking(ctx context.Context, bookingID, tenantID uuid.UUID) error {
	query := `UPDATE resource_bookings SET cancelled_at = $3 WHERE id = $1 AND tenant_id = $2 AND cancelled_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, bookingID, tenantID, time.Now())
	if err != nil {
		return fmt.Errorf("cancel booking: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBookingNotFound
	}
	return nil
}

func (r *PostgresRepository) ListBookings(ctx context.Context, resourceID uuid.UUID, start, end time.Time) ([]models.ResourceBooking, error) {
	query := `
		SELECT rb.id, rb.tenant_id, rb.resource_id, rb.event_id, rb.booked_by, rb.start_time, rb.end_time,
		       rb.cancelled_at, rb.created_at,
		       COALESCE(res.name, '') AS resource_name,
		       '' AS event_title
		FROM resource_bookings rb
		LEFT JOIN resources res ON res.id = rb.resource_id
		WHERE rb.resource_id = $1
		  AND rb.start_time < $3 AND rb.end_time > $2
		  AND rb.cancelled_at IS NULL
		ORDER BY rb.start_time`

	rows, err := r.pool.Query(ctx, query, resourceID, start, end)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	return scanBookings(rows)
}

func (r *PostgresRepository) ListBookingsByEvent(ctx context.Context, eventID uuid.UUID) ([]models.ResourceBooking, error) {
	query := `
		SELECT rb.id, rb.tenant_id, rb.resource_id, rb.event_id, rb.booked_by, rb.start_time, rb.end_time,
		       rb.cancelled_at, rb.created_at,
		       COALESCE(res.name, '') AS resource_name,
		       '' AS event_title
		FROM resource_bookings rb
		LEFT JOIN resources res ON res.id = rb.resource_id
		WHERE rb.event_id = $1
		  AND rb.cancelled_at IS NULL
		ORDER BY rb.start_time`

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list bookings by event: %w", err)
	}
	defer rows.Close()

	return scanBookings(rows)
}

func (r *PostgresRepository) GetBooking(ctx context.Context, bookingID, tenantID uuid.UUID) (*models.ResourceBooking, error) {
	query := `
		SELECT rb.id, rb.tenant_id, rb.resource_id, rb.event_id, rb.booked_by, rb.start_time, rb.end_time,
		       rb.cancelled_at, rb.created_at,
		       COALESCE(res.name, '') AS resource_name,
		       '' AS event_title
		FROM resource_bookings rb
		LEFT JOIN resources res ON res.id = rb.resource_id
		WHERE rb.id = $1 AND rb.tenant_id = $2`

	var b models.ResourceBooking
	err := r.pool.QueryRow(ctx, query, bookingID, tenantID).Scan(
		&b.ID, &b.TenantID, &b.ResourceID, &b.EventID, &b.BookedBy,
		&b.StartTime, &b.EndTime, &b.CancelledAt, &b.CreatedAt,
		&b.ResourceName, &b.EventTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) FindAvailableResources(ctx context.Context, start, end time.Time, filters ResourceFilters) ([]models.Resource, error) {
	var conditions []string
	var args []any
	argN := 3 // $1 = start, $2 = end

	args = append(args, start, end)

	conditions = append(conditions, `r.id NOT IN (
		SELECT resource_id FROM resource_bookings
		WHERE tstzrange(start_time, end_time) && tstzrange($1, $2)
		  AND cancelled_at IS NULL
	)`)
	conditions = append(conditions, "r.is_active = true")

	// Tenant isolation mandatory
	conditions = append(conditions, fmt.Sprintf("r.tenant_id = $%d", argN))
	args = append(args, filters.TenantID)
	argN++

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("r.resource_type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}
	if filters.MinCapacity != nil {
		conditions = append(conditions, fmt.Sprintf("r.capacity >= $%d", argN))
		args = append(args, *filters.MinCapacity)
		argN++
	}
	if filters.Floor != nil {
		conditions = append(conditions, fmt.Sprintf("r.floor = $%d", argN))
		args = append(args, *filters.Floor)
		argN++
	}
	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT resource_id FROM resource_tags WHERE tag = ANY($%d))", argN))
		args = append(args, filters.Tags)
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT r.id, r.tenant_id, r.name, r.resource_type, r.capacity, r.floor, r.location, r.description,
		       r.is_active, r.created_by, r.created_at, r.updated_at,
		       COALESCE(array_agg(rt.tag) FILTER (WHERE rt.tag IS NOT NULL), '{}') AS tags
		FROM resources r
		LEFT JOIN resource_tags rt ON rt.resource_id = r.id
		%s
		GROUP BY r.id
		ORDER BY r.name`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find available resources: %w", err)
	}
	defer rows.Close()

	var resources []models.Resource
	for rows.Next() {
		var res models.Resource
		if scanErr := rows.Scan(
			&res.ID, &res.TenantID, &res.Name, &res.ResourceType,
			&res.Capacity, &res.Floor, &res.Location, &res.Description,
			&res.IsActive, &res.CreatedBy,
			&res.CreatedAt, &res.UpdatedAt,
			&res.Tags,
		); scanErr != nil {
			return nil, fmt.Errorf("scan resource: %w", scanErr)
		}
		resources = append(resources, res)
	}
	return resources, rows.Err()
}

func (r *PostgresRepository) FindAlternatives(ctx context.Context, excludeID uuid.UUID, start, end time.Time, resourceType string, tenantID uuid.UUID) ([]AlternativeResource, error) {
	query := `
		SELECT r.id, r.name, r.capacity, r.floor
		FROM resources r
		WHERE r.id != $1
		  AND r.resource_type = $4
		  AND r.tenant_id = $5
		  AND r.is_active = true
		  AND r.id NOT IN (
		    SELECT resource_id FROM resource_bookings
		    WHERE tstzrange(start_time, end_time) && tstzrange($2, $3)
		      AND cancelled_at IS NULL
		  )
		ORDER BY r.name
		LIMIT 5`

	rows, err := r.pool.Query(ctx, query, excludeID, start, end, resourceType, tenantID)
	if err != nil {
		return nil, fmt.Errorf("find alternatives: %w", err)
	}
	defer rows.Close()

	var alternatives []AlternativeResource
	for rows.Next() {
		var alt AlternativeResource
		if scanErr := rows.Scan(&alt.ResourceID, &alt.ResourceName, &alt.Capacity, &alt.Floor); scanErr != nil {
			return nil, fmt.Errorf("scan alternative: %w", scanErr)
		}
		alternatives = append(alternatives, alt)
	}
	return alternatives, rows.Err()
}

func scanBookings(rows pgx.Rows) ([]models.ResourceBooking, error) {
	var bookings []models.ResourceBooking
	for rows.Next() {
		var b models.ResourceBooking
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.ResourceID, &b.EventID, &b.BookedBy,
			&b.StartTime, &b.EndTime, &b.CancelledAt, &b.CreatedAt,
			&b.ResourceName, &b.EventTitle,
		); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}
