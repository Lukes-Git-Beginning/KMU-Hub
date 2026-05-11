package vault

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

var ErrSecretNotFound = errors.New("vault: secret not found")

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed vault repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetByKeyName(ctx context.Context, keyName string) (*models.VaultSecret, error) {
	var s models.VaultSecret
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, key_name, encrypted_value, key_version, description, created_by, created_at, updated_at
		 FROM vault_secrets WHERE key_name = $1`, keyName,
	).Scan(
		&s.ID, &s.TenantID, &s.KeyName, &s.EncryptedValue, &s.KeyVersion,
		&s.Description, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]*models.VaultSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, key_name, key_version, description, created_by, created_at, updated_at
		 FROM vault_secrets ORDER BY key_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*models.VaultSecret
	for rows.Next() {
		var s models.VaultSecret
		if scanErr := rows.Scan(
			&s.ID, &s.TenantID, &s.KeyName, &s.KeyVersion,
			&s.Description, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		// EncryptedValue intentionally left empty -- List returns metadata only
		secrets = append(secrets, &s)
	}
	return secrets, rows.Err()
}

func (r *PostgresRepository) Create(ctx context.Context, secret *models.VaultSecret) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO vault_secrets (id, tenant_id, key_name, encrypted_value, key_version, description, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		secret.ID, secret.TenantID, secret.KeyName, secret.EncryptedValue, secret.KeyVersion,
		secret.Description, secret.CreatedBy, secret.CreatedAt, secret.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, secret *models.VaultSecret) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vault_secrets SET encrypted_value = $1, key_version = $2, description = $3, updated_at = $4
		 WHERE id = $5`,
		secret.EncryptedValue, secret.KeyVersion, secret.Description, secret.UpdatedAt, secret.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM vault_secrets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrSecretNotFound
	}
	return nil
}
