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

func (r *PostgresRepository) GetEventTypePreference(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, eventTypeKey string) (*models.NotificationPreference, error) {
	query := `
		SELECT id, tenant_id, user_id, event_type_key, module_id, in_app, desktop_push, email, sms, sound, created_at, updated_at
		FROM notification_preferences
		WHERE tenant_id = $1 AND user_id = $2 AND event_type_key = $3`

	pref := &models.NotificationPreference{}
	err := r.pool.QueryRow(ctx, query, tenantID, userID, eventTypeKey).Scan(
		&pref.ID, &pref.TenantID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
		&pref.InApp, &pref.DesktopPush, &pref.Email, &pref.SMS, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrPreferenceNotFound
	}
	return pref, err
}

func (r *PostgresRepository) GetModuleDefault(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID string) (*models.NotificationPreference, error) {
	query := `
		SELECT id, tenant_id, user_id, event_type_key, module_id, in_app, desktop_push, email, sms, sound, created_at, updated_at
		FROM notification_preferences
		WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3 AND event_type_key IS NULL`

	pref := &models.NotificationPreference{}
	err := r.pool.QueryRow(ctx, query, tenantID, userID, moduleID).Scan(
		&pref.ID, &pref.TenantID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
		&pref.InApp, &pref.DesktopPush, &pref.Email, &pref.SMS, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrPreferenceNotFound
	}
	return pref, err
}

func (r *PostgresRepository) ListPreferences(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error) {
	where := "WHERE tenant_id = $1 AND user_id = $2"
	args := []interface{}{tenantID, userID}
	argIdx := 3

	if moduleID != nil {
		where += fmt.Sprintf(" AND module_id = $%d", argIdx)
		args = append(args, *moduleID)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, user_id, event_type_key, module_id, in_app, desktop_push, email, sms, sound, created_at, updated_at
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
			&pref.ID, &pref.TenantID, &pref.UserID, &pref.EventTypeKey, &pref.ModuleID,
			&pref.InApp, &pref.DesktopPush, &pref.Email, &pref.SMS, &pref.Sound, &pref.CreatedAt, &pref.UpdatedAt,
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
		INSERT INTO notification_preferences (id, tenant_id, user_id, event_type_key, module_id, in_app, desktop_push, email, sms, sound, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id, user_id, event_type_key) WHERE event_type_key IS NOT NULL
		DO UPDATE SET in_app = $6, desktop_push = $7, email = $8, sms = $9, sound = $10, updated_at = $12`

	_, err := r.pool.Exec(ctx, query,
		pref.ID, pref.TenantID, pref.UserID, pref.EventTypeKey, pref.ModuleID,
		pref.InApp, pref.DesktopPush, pref.Email, pref.SMS, pref.Sound, pref.CreatedAt, pref.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) IsResourceMuted(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID, resourceID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM notification_mutes WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3 AND resource_id = $4)",
		tenantID, userID, moduleID, resourceID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CreateMute(ctx context.Context, mute *models.NotificationMute) error {
	query := `
		INSERT INTO notification_mutes (id, tenant_id, user_id, module_id, resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query, mute.ID, mute.TenantID, mute.UserID, mute.ModuleID, mute.ResourceID, mute.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteMute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, muteID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM notification_mutes WHERE id = $1 AND tenant_id = $2 AND user_id = $3",
		muteID, tenantID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMuteNotFound
	}
	return nil
}

func (r *PostgresRepository) ListMutes(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, moduleID *string, offset, limit int) ([]*models.NotificationMute, int, error) {
	where := "WHERE tenant_id = $1 AND user_id = $2"
	args := []interface{}{tenantID, userID}
	argIdx := 3

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
		SELECT id, tenant_id, user_id, module_id, resource_id, created_at
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
		err := rows.Scan(&mute.ID, &mute.TenantID, &mute.UserID, &mute.ModuleID, &mute.ResourceID, &mute.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		mutes = append(mutes, mute)
	}

	return mutes, total, rows.Err()
}

func (r *PostgresRepository) GetQuietHours(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (*models.QuietHours, error) {
	query := `
		SELECT id, tenant_id, user_id, start_time, end_time, timezone, days_of_week, enabled,
			manual_dnd, manual_dnd_until, created_at, updated_at
		FROM notification_quiet_hours
		WHERE tenant_id = $1 AND user_id = $2`

	qh := &models.QuietHours{}
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(
		&qh.ID, &qh.TenantID, &qh.UserID, &qh.StartTime, &qh.EndTime, &qh.Timezone, &qh.DaysOfWeek,
		&qh.Enabled, &qh.ManualDND, &qh.ManualDNDUntil, &qh.CreatedAt, &qh.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrQuietHoursNotFound
	}
	return qh, err
}

func (r *PostgresRepository) UpsertQuietHours(ctx context.Context, qh *models.QuietHours) error {
	query := `
		INSERT INTO notification_quiet_hours (id, tenant_id, user_id, start_time, end_time, timezone, days_of_week,
			enabled, manual_dnd, manual_dnd_until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET start_time = $4, end_time = $5, timezone = $6, days_of_week = $7,
			enabled = $8, manual_dnd = $9, manual_dnd_until = $10, updated_at = $12`

	_, err := r.pool.Exec(ctx, query,
		qh.ID, qh.TenantID, qh.UserID, qh.StartTime, qh.EndTime, qh.Timezone, qh.DaysOfWeek,
		qh.Enabled, qh.ManualDND, qh.ManualDNDUntil, qh.CreatedAt, qh.UpdatedAt,
	)
	return err
}
