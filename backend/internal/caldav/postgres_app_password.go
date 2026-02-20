package caldav

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresAppPasswordRepository implements AppPasswordRepository using PostgreSQL.
type PostgresAppPasswordRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAppPasswordRepository creates a new PostgreSQL app password repository.
func NewPostgresAppPasswordRepository(pool *pgxpool.Pool) *PostgresAppPasswordRepository {
	return &PostgresAppPasswordRepository{pool: pool}
}

func (r *PostgresAppPasswordRepository) Create(ctx context.Context, pw *models.AppSpecificPassword) error {
	query := `
		INSERT INTO app_specific_passwords (
			id, user_id, label, password_hash, password_prefix, scope, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	_, err := r.pool.Exec(ctx, query,
		pw.ID, pw.UserID, pw.Label, pw.PasswordHash, pw.PasswordPrefix, pw.Scope,
	)
	return err
}

func (r *PostgresAppPasswordRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.AppSpecificPassword, error) {
	query := `
		SELECT id, user_id, label, password_hash, password_prefix, scope,
			last_used_at, created_at, revoked_at
		FROM app_specific_passwords
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passwords []*models.AppSpecificPassword
	for rows.Next() {
		pw := &models.AppSpecificPassword{}
		if err := rows.Scan(
			&pw.ID, &pw.UserID, &pw.Label, &pw.PasswordHash, &pw.PasswordPrefix, &pw.Scope,
			&pw.LastUsedAt, &pw.CreatedAt, &pw.RevokedAt,
		); err != nil {
			return nil, err
		}
		passwords = append(passwords, pw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if passwords == nil {
		passwords = []*models.AppSpecificPassword{}
	}
	return passwords, nil
}

func (r *PostgresAppPasswordRepository) Revoke(ctx context.Context, id, userID uuid.UUID) error {
	query := `
		UPDATE app_specific_passwords
		SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPasswordNotFound
	}
	return nil
}

func (r *PostgresAppPasswordRepository) FindActiveByUser(ctx context.Context, userID uuid.UUID) ([]*models.AppSpecificPassword, error) {
	query := `
		SELECT id, user_id, label, password_hash, password_prefix, scope,
			last_used_at, created_at, revoked_at
		FROM app_specific_passwords
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passwords []*models.AppSpecificPassword
	for rows.Next() {
		pw := &models.AppSpecificPassword{}
		if err := rows.Scan(
			&pw.ID, &pw.UserID, &pw.Label, &pw.PasswordHash, &pw.PasswordPrefix, &pw.Scope,
			&pw.LastUsedAt, &pw.CreatedAt, &pw.RevokedAt,
		); err != nil {
			return nil, err
		}
		passwords = append(passwords, pw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if passwords == nil {
		passwords = []*models.AppSpecificPassword{}
	}
	return passwords, nil
}

func (r *PostgresAppPasswordRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE app_specific_passwords SET last_used_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// Compile-time interface check.
var _ AppPasswordRepository = (*PostgresAppPasswordRepository)(nil)
