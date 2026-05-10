package guest

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool

	// In-memory abuse tracking (ephemeral, resets on restart)
	failedMu    sync.Mutex
	failedCount map[string]int // ip_address -> failed validation count
}

// NewPostgresRepository creates a new PostgreSQL repository for guest data
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:        pool,
		failedCount: make(map[string]int),
	}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *GuestSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO guest_sessions (id, tenant_id, token_hash, channel_id, display_name, email, ip_address, user_agent, last_activity_at, is_active, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		session.ID, session.TenantID, session.TokenHash, session.ChannelID, session.DisplayName,
		session.Email, session.IPAddress, session.UserAgent,
		session.LastActivityAt, session.IsActive, session.CreatedAt, session.ExpiresAt,
	)
	return err
}

func (r *PostgresRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*GuestSession, error) {
	var s GuestSession
	err := r.pool.QueryRow(ctx,
		`SELECT gs.id, gs.token_hash, gs.channel_id, gs.display_name, gs.email,
		        gs.ip_address, gs.user_agent, gs.last_activity_at, gs.is_active,
		        gs.created_at, gs.expires_at
		 FROM guest_sessions gs
		 JOIN channels c ON gs.channel_id = c.id
		 WHERE gs.token_hash = $1 AND c.is_guest_enabled = true`, tokenHash,
	).Scan(
		&s.ID, &s.TokenHash, &s.ChannelID, &s.DisplayName, &s.Email,
		&s.IPAddress, &s.UserAgent, &s.LastActivityAt, &s.IsActive,
		&s.CreatedAt, &s.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*GuestSession, error) {
	var s GuestSession
	err := r.pool.QueryRow(ctx,
		`SELECT id, token_hash, channel_id, display_name, email,
		        ip_address, user_agent, last_activity_at, is_active,
		        created_at, expires_at
		 FROM guest_sessions WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TokenHash, &s.ChannelID, &s.DisplayName, &s.Email,
		&s.IPAddress, &s.UserAgent, &s.LastActivityAt, &s.IsActive,
		&s.CreatedAt, &s.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) ListSessionsByChannel(ctx context.Context, channelID uuid.UUID) ([]*GuestSession, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, token_hash, channel_id, display_name, email,
		        ip_address, user_agent, last_activity_at, is_active,
		        created_at, expires_at
		 FROM guest_sessions
		 WHERE channel_id = $1
		 ORDER BY created_at DESC`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*GuestSession
	for rows.Next() {
		var s GuestSession
		if scanErr := rows.Scan(
			&s.ID, &s.TokenHash, &s.ChannelID, &s.DisplayName, &s.Email,
			&s.IPAddress, &s.UserAgent, &s.LastActivityAt, &s.IsActive,
			&s.CreatedAt, &s.ExpiresAt,
		); scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

func (r *PostgresRepository) UpdateLastActivity(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE guest_sessions SET last_activity_at = NOW() WHERE id = $1`, id,
	)
	return err
}

func (r *PostgresRepository) DeactivateSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE guest_sessions SET is_active = false WHERE id = $1`, id,
	)
	return err
}

func (r *PostgresRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE guest_sessions SET is_active = false WHERE expires_at < NOW() AND is_active = true`,
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) CreateConfig(ctx context.Context, config *GuestChannelConfig) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO guest_channel_config (id, tenant_id, channel_id, welcome_message, logo_url, primary_color, token_expiry_hours, max_file_size_mb, allowed_file_types, is_active, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		config.ID, config.TenantID, config.ChannelID, config.WelcomeMessage, config.LogoURL,
		config.PrimaryColor, config.TokenExpiryHours, config.MaxFileSizeMB,
		config.AllowedFileTypes, config.IsActive, config.CreatedBy,
		config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetConfigByChannel(ctx context.Context, channelID uuid.UUID) (*GuestChannelConfig, error) {
	var c GuestChannelConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, channel_id, welcome_message, logo_url, primary_color,
		        token_expiry_hours, max_file_size_mb, allowed_file_types,
		        is_active, created_by, created_at, updated_at
		 FROM guest_channel_config WHERE channel_id = $1`, channelID,
	).Scan(
		&c.ID, &c.ChannelID, &c.WelcomeMessage, &c.LogoURL, &c.PrimaryColor,
		&c.TokenExpiryHours, &c.MaxFileSizeMB, &c.AllowedFileTypes,
		&c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) UpdateConfig(ctx context.Context, config *GuestChannelConfig) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE guest_channel_config
		 SET welcome_message = $1, logo_url = $2, primary_color = $3,
		     token_expiry_hours = $4, max_file_size_mb = $5, allowed_file_types = $6,
		     is_active = $7, updated_at = $8
		 WHERE channel_id = $9`,
		config.WelcomeMessage, config.LogoURL, config.PrimaryColor,
		config.TokenExpiryHours, config.MaxFileSizeMB, config.AllowedFileTypes,
		config.IsActive, config.UpdatedAt, config.ChannelID,
	)
	return err
}

func (r *PostgresRepository) DeleteConfig(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM guest_channel_config WHERE channel_id = $1`, channelID,
	)
	return err
}

// IncrementFailedValidations tracks failed token validation attempts per IP (in-memory)
func (r *PostgresRepository) IncrementFailedValidations(ipAddress string) int {
	r.failedMu.Lock()
	defer r.failedMu.Unlock()
	r.failedCount[ipAddress]++
	return r.failedCount[ipAddress]
}

// GetFailedValidations returns the number of failed validation attempts for an IP
func (r *PostgresRepository) GetFailedValidations(ipAddress string) int {
	r.failedMu.Lock()
	defer r.failedMu.Unlock()
	return r.failedCount[ipAddress]
}

// ResetFailedValidations clears the failed validation counter for an IP
func (r *PostgresRepository) ResetFailedValidations(ipAddress string) {
	r.failedMu.Lock()
	defer r.failedMu.Unlock()
	delete(r.failedCount, ipAddress)
}
