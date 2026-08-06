package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresBookingRepository implements BookingRepository using PostgreSQL.
type PostgresBookingRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresBookingRepository creates a new booking repository.
func NewPostgresBookingRepository(pool *pgxpool.Pool) *PostgresBookingRepository {
	return &PostgresBookingRepository{pool: pool}
}

// ============================================================================
// Booking Pages
// ============================================================================

func (r *PostgresBookingRepository) CreateBookingPage(ctx context.Context, page *models.BookingPage) error {
	svcsJSON, err := json.Marshal(page.Services)
	if err != nil {
		return err
	}
	rulesJSON, err := json.Marshal(page.AvailabilityRules)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO booking_pages
		 (id, tenant_id, calendar_id, slug, company_name, logo_url, services, availability_rules, active, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		page.ID, page.TenantID, page.CalendarID, page.Slug, page.CompanyName,
		page.LogoURL, svcsJSON, rulesJSON, page.Active, page.CreatedAt, page.UpdatedAt,
	)
	return err
}

func (r *PostgresBookingRepository) GetBookingPageByID(ctx context.Context, id, tenantID uuid.UUID) (*models.BookingPage, error) {
	return r.scanBookingPage(ctx,
		`SELECT id, tenant_id, calendar_id, slug, company_name, logo_url,
		        services, availability_rules, active, created_at, updated_at
		 FROM booking_pages WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
}

// SECURITY: this unauth lookup has no tenant in ctx, so it resolves the slug
// globally. A UNIQUE partial index on (slug) WHERE active (migration 000149)
// guarantees at most one ACTIVE page per slug across all tenants, so the lookup
// is unambiguous and cannot misroute bookings. The full multi-tenant answer —
// a tenant discriminator (e.g. subdomain) in the public URL — is deferred to
// the Option-B retrofit; until then the active-slug index is the guard. The base
// UNIQUE(tenant_id, slug) (booking_pages_tenant_slug_unique) still permits the
// same slug INACTIVE across tenants, hence ORDER BY/LIMIT 1 as defense-in-depth.
func (r *PostgresBookingRepository) GetBookingPageBySlug(ctx context.Context, slug string) (*models.BookingPage, error) {
	return r.scanBookingPage(ctx,
		`SELECT id, tenant_id, calendar_id, slug, company_name, logo_url,
		        services, availability_rules, active, created_at, updated_at
		 FROM booking_pages WHERE slug = $1 AND active = true ORDER BY created_at ASC LIMIT 1`,
		slug,
	)
}

func (r *PostgresBookingRepository) scanBookingPage(ctx context.Context, query string, args ...interface{}) (*models.BookingPage, error) {
	var p models.BookingPage
	var svcsJSON, rulesJSON []byte

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.TenantID, &p.CalendarID, &p.Slug, &p.CompanyName,
		&p.LogoURL, &svcsJSON, &rulesJSON, &p.Active, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBookingPageNotFound
	}
	if err != nil {
		return nil, err
	}

	if unmarshalErr := json.Unmarshal(svcsJSON, &p.Services); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if unmarshalErr := json.Unmarshal(rulesJSON, &p.AvailabilityRules); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &p, nil
}

func (r *PostgresBookingRepository) UpdateBookingPage(ctx context.Context, page *models.BookingPage) error {
	svcsJSON, err := json.Marshal(page.Services)
	if err != nil {
		return err
	}
	rulesJSON, err := json.Marshal(page.AvailabilityRules)
	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE booking_pages
		 SET company_name=$1, logo_url=$2, services=$3, availability_rules=$4, active=$5, updated_at=$6
		 WHERE id=$7 AND tenant_id=$8`,
		page.CompanyName, page.LogoURL, svcsJSON, rulesJSON, page.Active, page.UpdatedAt,
		page.ID, page.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBookingPageNotFound
	}
	return nil
}

func (r *PostgresBookingRepository) DeleteBookingPage(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM booking_pages WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBookingPageNotFound
	}
	return nil
}

func (r *PostgresBookingRepository) ListBookingPages(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.BookingPage, error) {
	query := `SELECT id, tenant_id, calendar_id, slug, company_name, logo_url,
	                 services, availability_rules, active, created_at, updated_at
	          FROM booking_pages WHERE tenant_id = $1`
	if !includeInactive {
		query += ` AND active = true`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.BookingPage
	for rows.Next() {
		var p models.BookingPage
		var svcsJSON, rulesJSON []byte

		if scanErr := rows.Scan(
			&p.ID, &p.TenantID, &p.CalendarID, &p.Slug, &p.CompanyName,
			&p.LogoURL, &svcsJSON, &rulesJSON, &p.Active, &p.CreatedAt, &p.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}

		if unmarshalErr := json.Unmarshal(svcsJSON, &p.Services); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		if unmarshalErr := json.Unmarshal(rulesJSON, &p.AvailabilityRules); unmarshalErr != nil {
			return nil, unmarshalErr
		}

		pages = append(pages, p)
	}

	return pages, rows.Err()
}

// ============================================================================
// Public Bookings
// ============================================================================

func (r *PostgresBookingRepository) CreatePublicBooking(ctx context.Context, b *models.PublicBooking) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public_bookings
		 (id, tenant_id, booking_page_id, service_id, customer_name, customer_email,
		  customer_phone, notes, date, time_slot, staff_user_id, status, calendar_event_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		b.ID, b.TenantID, b.BookingPageID, b.ServiceID, b.CustomerName, b.CustomerEmail,
		b.CustomerPhone, b.Notes, b.Date, b.TimeSlot, b.StaffUserID, b.Status,
		b.CalendarEventID, b.CreatedAt,
	)
	return err
}

// UpdatePublicBookingCalendarEventID writes back the calendar event created for a
// public booking after the fact: the event only exists once the booking row has
// already been inserted, so this is the write-back half of that best-effort step.
func (r *PostgresBookingRepository) UpdatePublicBookingCalendarEventID(ctx context.Context, bookingID, tenantID, eventID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public_bookings SET calendar_event_id = $1 WHERE id = $2 AND tenant_id = $3`,
		eventID, bookingID, tenantID,
	)
	return err
}

func (r *PostgresBookingRepository) GetBookedSlotsForPage(ctx context.Context, bookingPageID uuid.UUID, from, to time.Time) ([]models.PublicBooking, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, booking_page_id, service_id, customer_name, customer_email,
		        customer_phone, notes, date, time_slot, staff_user_id, status, calendar_event_id, created_at
		 FROM public_bookings
		 WHERE booking_page_id = $1
		   AND date >= $2::date AND date <= $3::date
		   AND status NOT IN ('cancelled')
		 ORDER BY date, time_slot`,
		bookingPageID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.PublicBooking
	for rows.Next() {
		var b models.PublicBooking
		if scanErr := rows.Scan(
			&b.ID, &b.TenantID, &b.BookingPageID, &b.ServiceID, &b.CustomerName, &b.CustomerEmail,
			&b.CustomerPhone, &b.Notes, &b.Date, &b.TimeSlot, &b.StaffUserID, &b.Status,
			&b.CalendarEventID, &b.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		bookings = append(bookings, b)
	}

	return bookings, rows.Err()
}

// GetCalendarEventsInRange returns minimal start/end projections for calendar events
// on the given calendar within [from, to). Used for staff availability calculation.
func (r *PostgresBookingRepository) GetCalendarEventsInRange(ctx context.Context, calendarID uuid.UUID, from, to time.Time) ([]eventSlot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT start_time, end_time
		 FROM calendar_events
		 WHERE calendar_id = $1
		   AND start_time < $3 AND end_time > $2
		 ORDER BY start_time`,
		calendarID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []eventSlot
	for rows.Next() {
		var s eventSlot
		if scanErr := rows.Scan(&s.StartTime, &s.EndTime); scanErr != nil {
			return nil, scanErr
		}
		slots = append(slots, s)
	}

	return slots, rows.Err()
}
