package signature

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

// NewPostgresRepository creates a new PostgreSQL signature repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, sig *models.EmailSignature) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_signatures (id, tenant_id, user_id, name, html_content, is_default, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		sig.ID, sig.TenantID, sig.UserID, sig.Name, sig.HTMLContent, sig.IsDefault,
		sig.CreatedAt, sig.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	var sig models.EmailSignature
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, name, html_content, is_default, created_at, updated_at
		 FROM email_signatures WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&sig.ID, &sig.TenantID, &sig.UserID, &sig.Name, &sig.HTMLContent,
		&sig.IsDefault, &sig.CreatedAt, &sig.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSignatureNotFound
		}
		return nil, err
	}
	return &sig, nil
}

func (r *PostgresRepository) GetDefault(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	var sig models.EmailSignature
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, name, html_content, is_default, created_at, updated_at
		 FROM email_signatures WHERE user_id = $1 AND tenant_id = $2 AND is_default = true
		 LIMIT 1`, userID, tenantID,
	).Scan(&sig.ID, &sig.TenantID, &sig.UserID, &sig.Name, &sig.HTMLContent,
		&sig.IsDefault, &sig.CreatedAt, &sig.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSignatureNotFound
		}
		return nil, err
	}
	return &sig, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailSignature, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, name, html_content, is_default, created_at, updated_at
		 FROM email_signatures WHERE user_id = $1 AND tenant_id = $2
		 ORDER BY is_default DESC, name ASC`, userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sigs []*models.EmailSignature
	for rows.Next() {
		var sig models.EmailSignature
		if err := rows.Scan(&sig.ID, &sig.TenantID, &sig.UserID, &sig.Name, &sig.HTMLContent,
			&sig.IsDefault, &sig.CreatedAt, &sig.UpdatedAt); err != nil {
			return nil, err
		}
		sigs = append(sigs, &sig)
	}
	return sigs, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, sig *models.EmailSignature) error {
	sig.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE email_signatures SET name = $2, html_content = $3, is_default = $4, updated_at = $5
		 WHERE id = $1 AND tenant_id = $6`,
		sig.ID, sig.Name, sig.HTMLContent, sig.IsDefault, sig.UpdatedAt, sig.TenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_signatures WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *PostgresRepository) ClearDefaultForUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_signatures SET is_default = false, updated_at = $3 WHERE user_id = $1 AND tenant_id = $2`,
		userID, tenantID, time.Now().UTC(),
	)
	return err
}

func (r *PostgresRepository) CountByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_signatures WHERE user_id = $1 AND tenant_id = $2`, userID, tenantID,
	).Scan(&count)
	return count, err
}
