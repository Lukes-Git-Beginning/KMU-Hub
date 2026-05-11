package password

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed password repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// defaultPolicy returns NIST SP 800-63B aligned defaults:
// no complexity requirements, entropy-based validation, 12 char minimum.
func defaultPolicy() *models.PasswordPolicy {
	return &models.PasswordPolicy{
		ID:                uuid.New(),
		MinLength:         12,
		RequireUppercase:  false,
		RequireLowercase:  false,
		RequireDigit:      false,
		RequireSpecial:    false,
		MinEntropy:        50.0,
		MaxAgeDays:        nil,
		PreventReuseCount: 5,
		UpdatedAt:         time.Now().UTC(),
	}
}

func (r *PostgresRepository) GetPolicy(ctx context.Context) (*models.PasswordPolicy, error) {
	var p models.PasswordPolicy
	err := r.pool.QueryRow(ctx,
		`SELECT id, min_length, require_uppercase, require_lowercase, require_digit, require_special,
		        min_entropy, max_age_days, prevent_reuse_count, updated_at, updated_by
		 FROM password_policies
		 ORDER BY updated_at DESC
		 LIMIT 1`,
	).Scan(
		&p.ID, &p.MinLength, &p.RequireUppercase, &p.RequireLowercase,
		&p.RequireDigit, &p.RequireSpecial, &p.MinEntropy, &p.MaxAgeDays,
		&p.PreventReuseCount, &p.UpdatedAt, &p.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultPolicy(), nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) UpdatePolicy(ctx context.Context, policy *models.PasswordPolicy) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_policies
		 SET min_length = $1, require_uppercase = $2, require_lowercase = $3,
		     require_digit = $4, require_special = $5, min_entropy = $6,
		     max_age_days = $7, prevent_reuse_count = $8, updated_at = $9, updated_by = $10
		 WHERE id = $11`,
		policy.MinLength, policy.RequireUppercase, policy.RequireLowercase,
		policy.RequireDigit, policy.RequireSpecial, policy.MinEntropy,
		policy.MaxAgeDays, policy.PreventReuseCount, policy.UpdatedAt, policy.UpdatedBy,
		policy.ID,
	)
	return err
}

func (r *PostgresRepository) AddPasswordHistory(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_history (id, user_id, tenant_id, password_hash, created_at)
		 VALUES ($1, $2, (SELECT tenant_id FROM users WHERE id = $2), $3, $4)`,
		uuid.New(), userID, passwordHash, time.Now().UTC(),
	)
	return err
}

func (r *PostgresRepository) GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT password_hash FROM password_history
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var hash string
		if scanErr := rows.Scan(&hash); scanErr != nil {
			return nil, scanErr
		}
		hashes = append(hashes, hash)
	}
	return hashes, rows.Err()
}
