package caldav

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CalDAVUserInfo holds user metadata for admin CalDAV management.
type CalDAVUserInfo struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	CalDAVEnabled bool       `json:"caldav_enabled"`
	PasswordCount int        `json:"password_count"`
	LastUsed      *time.Time `json:"last_used"`
}

// UserPreferenceRepository abstracts user-level CalDAV preference persistence.
type UserPreferenceRepository interface {
	// GetCalDAVEnabled returns whether CalDAV is enabled for the given user.
	GetCalDAVEnabled(ctx context.Context, userID uuid.UUID) (bool, error)

	// SetCalDAVEnabled enables or disables CalDAV for the given user.
	SetCalDAVEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error

	// ListCalDAVUsers returns all users that have CalDAV enabled, with password counts.
	ListCalDAVUsers(ctx context.Context) ([]CalDAVUserInfo, error)

	// RevokeAllUserPasswords revokes all active app-specific passwords for a user.
	// Returns the number of rows affected.
	RevokeAllUserPasswords(ctx context.Context, userID uuid.UUID) (int64, error)
}

// PostgresUserPreferenceRepository implements UserPreferenceRepository using PostgreSQL.
type PostgresUserPreferenceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserPreferenceRepository creates a new PostgreSQL user preference repository.
func NewPostgresUserPreferenceRepository(pool *pgxpool.Pool) *PostgresUserPreferenceRepository {
	return &PostgresUserPreferenceRepository{pool: pool}
}

// GetCalDAVEnabled returns whether CalDAV is enabled for the given user.
func (r *PostgresUserPreferenceRepository) GetCalDAVEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	var enabled bool
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(caldav_enabled, false) FROM users WHERE id = $1`,
		userID,
	).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// SetCalDAVEnabled enables or disables CalDAV for the given user.
func (r *PostgresUserPreferenceRepository) SetCalDAVEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET caldav_enabled = $1 WHERE id = $2`,
		enabled, userID,
	)
	return err
}

// ListCalDAVUsers returns all users that have CalDAV enabled, with password counts.
func (r *PostgresUserPreferenceRepository) ListCalDAVUsers(ctx context.Context) ([]CalDAVUserInfo, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, u.first_name, u.last_name, u.caldav_enabled,
		        (SELECT COUNT(*) FROM app_specific_passwords asp
		         WHERE asp.user_id = u.id AND asp.revoked_at IS NULL) AS password_count,
		        (SELECT MAX(asp.last_used_at) FROM app_specific_passwords asp
		         WHERE asp.user_id = u.id) AS last_used
		 FROM users u
		 WHERE u.caldav_enabled = true
		 ORDER BY u.email`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []CalDAVUserInfo
	for rows.Next() {
		var u CalDAVUserInfo
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName,
			&u.CalDAVEnabled, &u.PasswordCount, &u.LastUsed,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if users == nil {
		users = []CalDAVUserInfo{}
	}
	return users, nil
}

// RevokeAllUserPasswords revokes all active app-specific passwords for a user.
// Returns the number of rows affected.
func (r *PostgresUserPreferenceRepository) RevokeAllUserPasswords(ctx context.Context, userID uuid.UUID) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE app_specific_passwords
		 SET revoked_at = NOW()
		 WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Compile-time interface check.
var _ UserPreferenceRepository = (*PostgresUserPreferenceRepository)(nil)
