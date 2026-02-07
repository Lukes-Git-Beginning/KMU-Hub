package preference

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL preference repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetEventTypePreference(ctx context.Context, userID uuid.UUID, eventTypeKey string) (*models.NotificationPreference, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, in_app, desktop_push, sound, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND event_type_key = $2`

	pref := &models.NotificationPreference{}
	err := r.pool.QueryRow(ctx, query, userID, eventTypeKey).Scan(
		&pref.ID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
		&pref.InApp, &pref.DesktopPush, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrPreferenceNotFound
	}
	return pref, err
}

func (r *PostgresRepository) GetModuleDefault(ctx context.Context, userID uuid.UUID, moduleID string) (*models.NotificationPreference, error) {
	query := `
		SELECT id, user_id, event_type_key, module_id, in_app, desktop_push, sound, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND module_id = $2 AND event_type_key IS NULL`

	pref := &models.NotificationPreference{}
	err := r.pool.QueryRow(ctx, query, userID, moduleID).Scan(
		&pref.ID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
		&pref.InApp, &pref.DesktopPush, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrPreferenceNotFound
	}
	return pref, err
}

func (r *PostgresRepository) ListPreferences(ctx context.Context, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error) {
	where := "WHERE user_id = $1"
	args := []interface{}{userID}

	if moduleID != nil {
		where += " AND module_id = $2"
		args = append(args, *moduleID)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, event_type_key, module_id, in_app, desktop_push, sound, created_at, updated_at
		FROM notification_preferences %s
		ORDER BY module_id, event_type_key NULLS FIRST`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []*models.NotificationPreference
	for rows.Next() {
		pref := &models.NotificationPreference{}
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
			&pref.InApp, &pref.DesktopPush, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		prefs = append(prefs, pref)
	}

	return prefs, rows.Err()
}

func (r *PostgresRepository) UpsertPreference(ctx context.Context, pref *models.NotificationPreference) error {
	query := `
		INSERT INTO notification_preferences (id, user_id, event_type_key, module_id, in_app, desktop_push, sound, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, event_type_key) WHERE event_type_key IS NOT NULL
		DO UPDATE SET in_app = $5, desktop_push = $6, sound = $7, updated_at = $9`

	_, err := r.pool.Exec(ctx, query,
		pref.ID, pref.UserID, pref.EventTypeKey, pref.ModuleID,
		pref.InApp, pref.DesktopPush, pref.Sound, pref.CreatedAt, pref.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) IsResourceMuted(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM notification_mutes WHERE user_id = $1 AND module_id = $2 AND resource_id = $3)",
		userID, moduleID, resourceID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CreateMute(ctx context.Context, mute *models.NotificationMute) error {
	query := `
		INSERT INTO notification_mutes (id, user_id, module_id, resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query, mute.ID, mute.UserID, mute.ModuleID, mute.ResourceID, mute.CreatedAt)
	if err != nil {
		// Check for unique constraint violation
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteMute(ctx context.Context, userID uuid.UUID, moduleID, resourceID string) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM notification_mutes WHERE user_id = $1 AND module_id = $2 AND resource_id = $3",
		userID, moduleID, resourceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMuteNotFound
	}
	return nil
}

func (r *PostgresRepository) ListMutes(ctx context.Context, userID uuid.UUID, moduleID *string, offset, limit int) ([]*models.NotificationMute, int, error) {
	where := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIdx := 2

	if moduleID != nil {
		where += fmt.Sprintf(" AND module_id = $%d", argIdx)
		args = append(args, *moduleID)
		argIdx++
	}

	// Count
	var total int
	countQuery := "SELECT COUNT(*) FROM notification_mutes " + where
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, module_id, resource_id, created_at
		FROM notification_mutes %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var mutes []*models.NotificationMute
	for rows.Next() {
		mute := &models.NotificationMute{}
		err := rows.Scan(&mute.ID, &mute.UserID, &mute.ModuleID, &mute.ResourceID, &mute.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		mutes = append(mutes, mute)
	}

	return mutes, total, rows.Err()
}

func (r *PostgresRepository) GetQuietHours(ctx context.Context, userID uuid.UUID) (*models.QuietHours, error) {
	query := `
		SELECT id, user_id, start_time, end_time, timezone, days_of_week, enabled,
			manual_dnd, manual_dnd_until, created_at, updated_at
		FROM notification_quiet_hours
		WHERE user_id = $1`

	qh := &models.QuietHours{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&qh.ID, &qh.UserID, &qh.StartTime, &qh.EndTime, &qh.Timezone, &qh.DaysOfWeek,
		&qh.Enabled, &qh.ManualDND, &qh.ManualDNDUntil, &qh.CreatedAt, &qh.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrQuietHoursNotFound
	}
	return qh, err
}

func (r *PostgresRepository) UpsertQuietHours(ctx context.Context, qh *models.QuietHours) error {
	query := `
		INSERT INTO notification_quiet_hours (id, user_id, start_time, end_time, timezone, days_of_week,
			enabled, manual_dnd, manual_dnd_until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id)
		DO UPDATE SET start_time = $3, end_time = $4, timezone = $5, days_of_week = $6,
			enabled = $7, manual_dnd = $8, manual_dnd_until = $9, updated_at = $11`

	_, err := r.pool.Exec(ctx, query,
		qh.ID, qh.UserID, qh.StartTime, qh.EndTime, qh.Timezone, qh.DaysOfWeek,
		qh.Enabled, qh.ManualDND, qh.ManualDNDUntil, qh.CreatedAt, qh.UpdatedAt,
	)
	return err
}
