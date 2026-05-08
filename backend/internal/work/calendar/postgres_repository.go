package calendar

import (
	"context"
	"errors"
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

// NewPostgresRepository creates a new PostgreSQL calendar repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, calendar *models.Calendar) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO calendars (id, tenant_id, name, description, calendar_type, color, owner_id, is_default, timezone, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		calendar.ID, calendar.TenantID, calendar.Name, calendar.Description, calendar.CalendarType,
		calendar.Color, calendar.OwnerID, calendar.IsDefault, calendar.Timezone,
		calendar.CreatedAt, calendar.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Calendar, error) {
	var c models.Calendar
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, calendar_type, color, owner_id, is_default, timezone, created_at, updated_at
		 FROM calendars WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CalendarType, &c.Color,
		&c.OwnerID, &c.IsDefault, &c.Timezone, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCalendarNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID, tenantID uuid.UUID) ([]models.CalendarWithMemberInfo, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.tenant_id, c.name, c.description, c.calendar_type, c.color, c.owner_id,
		        c.is_default, c.timezone, c.created_at, c.updated_at,
		        COALESCE(cm.permission, 'admin') AS permission,
		        cm.color_override,
		        COALESCE(cm.is_visible, true) AS is_visible
		 FROM calendars c
		 LEFT JOIN calendar_members cm ON c.id = cm.calendar_id AND cm.user_id = $1
		 WHERE c.tenant_id = $2
		   AND (c.owner_id = $1 OR cm.user_id = $1)
		 ORDER BY c.is_default DESC, c.name ASC`,
		userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []models.CalendarWithMemberInfo
	for rows.Next() {
		var cwm models.CalendarWithMemberInfo
		if scanErr := rows.Scan(
			&cwm.ID, &cwm.TenantID, &cwm.Name, &cwm.Description, &cwm.CalendarType, &cwm.Color,
			&cwm.OwnerID, &cwm.IsDefault, &cwm.Timezone, &cwm.CreatedAt, &cwm.UpdatedAt,
			&cwm.Permission, &cwm.ColorOverride, &cwm.IsVisible,
		); scanErr != nil {
			return nil, scanErr
		}
		calendars = append(calendars, cwm)
	}
	return calendars, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, calendar *models.Calendar) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendars SET name = $1, description = $2, color = $3, timezone = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		calendar.Name, calendar.Description, calendar.Color, calendar.Timezone,
		calendar.UpdatedAt, calendar.ID, calendar.TenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calendars WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// Members

func (r *PostgresRepository) AddMember(ctx context.Context, member *models.CalendarMember) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO calendar_members (calendar_id, tenant_id, user_id, permission, color_override, is_visible, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		member.CalendarID, member.TenantID, member.UserID, member.Permission,
		member.ColorOverride, member.IsVisible, member.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrMemberAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, calendarID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM calendar_members WHERE calendar_id = $1 AND user_id = $2`,
		calendarID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, calendarID uuid.UUID) ([]models.CalendarMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cm.calendar_id, cm.tenant_id, cm.user_id, cm.permission, cm.color_override, cm.is_visible, cm.created_at,
		        COALESCE(u.first_name, '') AS first_name, COALESCE(u.last_name, '') AS last_name
		 FROM calendar_members cm
		 JOIN users u ON cm.user_id = u.id
		 WHERE cm.calendar_id = $1
		 ORDER BY cm.created_at ASC`,
		calendarID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.CalendarMember
	for rows.Next() {
		var m models.CalendarMember
		if scanErr := rows.Scan(&m.CalendarID, &m.TenantID, &m.UserID, &m.Permission, &m.ColorOverride,
			&m.IsVisible, &m.CreatedAt, &m.FirstName, &m.LastName); scanErr != nil {
			return nil, scanErr
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *PostgresRepository) GetMember(ctx context.Context, calendarID, userID uuid.UUID) (*models.CalendarMember, error) {
	var m models.CalendarMember
	err := r.pool.QueryRow(ctx,
		`SELECT cm.calendar_id, cm.tenant_id, cm.user_id, cm.permission, cm.color_override, cm.is_visible, cm.created_at,
		        COALESCE(u.first_name, '') AS first_name, COALESCE(u.last_name, '') AS last_name
		 FROM calendar_members cm
		 JOIN users u ON cm.user_id = u.id
		 WHERE cm.calendar_id = $1 AND cm.user_id = $2`,
		calendarID, userID,
	).Scan(&m.CalendarID, &m.TenantID, &m.UserID, &m.Permission, &m.ColorOverride,
		&m.IsVisible, &m.CreatedAt, &m.FirstName, &m.LastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PostgresRepository) UpdateMemberPermission(ctx context.Context, calendarID, userID uuid.UUID, permission string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendar_members SET permission = $1 WHERE calendar_id = $2 AND user_id = $3`,
		permission, calendarID, userID,
	)
	return err
}

func (r *PostgresRepository) UpdateMemberVisibility(ctx context.Context, calendarID, userID uuid.UUID, visible bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendar_members SET is_visible = $1 WHERE calendar_id = $2 AND user_id = $3`,
		visible, calendarID, userID,
	)
	return err
}

func (r *PostgresRepository) UpdateMemberColorOverride(ctx context.Context, calendarID, userID uuid.UUID, color *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendar_members SET color_override = $1 WHERE calendar_id = $2 AND user_id = $3`,
		color, calendarID, userID,
	)
	return err
}

// Discovery & Subscription

// ListBrowsable returns shared calendars within the tenant that the given user
// has not yet subscribed to (i.e. calendars available for discovery/joining).
//
// Semantics (verified against Welle 4B retrofit):
//   - Only calendars with calendar_type = 'shared' are included. Personal and
//     holiday calendars are never shown in the browse/discover view.
//   - The tenant_id filter is mandatory: a user must never see shared calendars
//     from a different tenant. This filter was added in Migration 000109 (Welle 4B).
//   - Calendars owned by the requesting user are excluded (already in their list).
//   - Calendars the user is already a member of are excluded via NOT EXISTS.
//
// Note: the calendar_type = 'shared' condition was present before Welle 4B and
// was intentionally retained after the tenant_id retrofit (F9 Sprint 3 review).
func (r *PostgresRepository) ListBrowsable(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Calendar, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.tenant_id, c.name, c.description, c.calendar_type, c.color, c.owner_id,
		        c.is_default, c.timezone, c.created_at, c.updated_at
		 FROM calendars c
		 WHERE c.tenant_id = $3
		   AND c.calendar_type = 'shared'
		   AND c.owner_id != $1
		   AND NOT EXISTS (
		       SELECT 1 FROM calendar_members cm
		       WHERE cm.calendar_id = c.id AND cm.user_id = $1
		   )
		 ORDER BY c.name ASC`,
		userID, userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []models.Calendar
	for rows.Next() {
		var c models.Calendar
		if scanErr := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CalendarType, &c.Color,
			&c.OwnerID, &c.IsDefault, &c.Timezone, &c.CreatedAt, &c.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		calendars = append(calendars, c)
	}
	return calendars, rows.Err()
}

func (r *PostgresRepository) Subscribe(ctx context.Context, calendarID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO calendar_members (calendar_id, user_id, permission, is_visible, created_at)
		 VALUES ($1, $2, 'view', true, $3)`,
		calendarID, userID, time.Now(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrSubscriptionExists
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) Unsubscribe(ctx context.Context, calendarID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM calendar_members WHERE calendar_id = $1 AND user_id = $2`,
		calendarID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

// Event Categories

func (r *PostgresRepository) CreateCategory(ctx context.Context, category *models.EventCategory) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_categories (id, tenant_id, user_id, name, color, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		category.ID, category.TenantID, category.UserID, category.Name, category.Color,
		category.SortOrder, category.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateCategoryName
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) ListCategories(ctx context.Context, userID, tenantID uuid.UUID) ([]models.EventCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, name, color, sort_order, created_at
		 FROM event_categories WHERE user_id = $1 AND tenant_id = $2
		 ORDER BY sort_order ASC, name ASC`,
		userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.EventCategory
	for rows.Next() {
		var c models.EventCategory
		if scanErr := rows.Scan(&c.ID, &c.TenantID, &c.UserID, &c.Name, &c.Color, &c.SortOrder, &c.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *PostgresRepository) DeleteCategory(ctx context.Context, id, userID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM event_categories WHERE id = $1 AND user_id = $2 AND tenant_id = $3`,
		id, userID, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// Preferences

func (r *PostgresRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.UserCalendarPreferences, error) {
	var p models.UserCalendarPreferences
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, default_view, week_days, default_reminder_minutes,
		        default_allday_reminder_minutes, subdivision_code, show_task_deadlines, updated_at
		 FROM user_calendar_preferences WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.DefaultView, &p.WeekDays, &p.DefaultReminderMinutes,
		&p.DefaultAllDayReminderMinutes, &p.SubdivisionCode, &p.ShowTaskDeadlines, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) UpsertPreferences(ctx context.Context, prefs *models.UserCalendarPreferences) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_calendar_preferences
		 (user_id, default_view, week_days, default_reminder_minutes, default_allday_reminder_minutes,
		  subdivision_code, show_task_deadlines, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id) DO UPDATE SET
		 default_view = $2, week_days = $3, default_reminder_minutes = $4,
		 default_allday_reminder_minutes = $5, subdivision_code = $6,
		 show_task_deadlines = $7, updated_at = $8`,
		prefs.UserID, prefs.DefaultView, prefs.WeekDays, prefs.DefaultReminderMinutes,
		prefs.DefaultAllDayReminderMinutes, prefs.SubdivisionCode, prefs.ShowTaskDeadlines,
		prefs.UpdatedAt,
	)
	return err
}

// EnsurePersonalCalendar creates a default personal calendar if one does not exist
func (r *PostgresRepository) EnsurePersonalCalendar(ctx context.Context, userID, tenantID uuid.UUID) (*models.Calendar, error) {
	var c models.Calendar

	// Try to find existing default personal calendar
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, calendar_type, color, owner_id, is_default, timezone, created_at, updated_at
		 FROM calendars WHERE owner_id = $1 AND tenant_id = $2 AND is_default = true AND calendar_type = 'personal'`,
		userID, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CalendarType, &c.Color,
		&c.OwnerID, &c.IsDefault, &c.Timezone, &c.CreatedAt, &c.UpdatedAt)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create default personal calendar
	now := time.Now()
	c = models.Calendar{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         "Mein Kalender",
		CalendarType: models.CalendarTypePersonal,
		Color:        "#4285F4",
		OwnerID:      userID,
		IsDefault:    true,
		Timezone:     "Europe/Berlin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, insertErr := r.pool.Exec(ctx,
		`INSERT INTO calendars (id, tenant_id, name, description, calendar_type, color, owner_id, is_default, timezone, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT DO NOTHING`,
		c.ID, c.TenantID, c.Name, c.Description, c.CalendarType, c.Color,
		c.OwnerID, c.IsDefault, c.Timezone, c.CreatedAt, c.UpdatedAt,
	)
	if insertErr != nil {
		return nil, insertErr
	}

	return &c, nil
}
