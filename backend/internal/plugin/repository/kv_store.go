package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KVStoreRepository handles plugin key-value storage persistence
type KVStoreRepository struct {
	pool *pgxpool.Pool
}

// NewKVStoreRepository creates a new KV store repository
func NewKVStoreRepository(pool *pgxpool.Pool) *KVStoreRepository {
	return &KVStoreRepository{pool: pool}
}

// Get retrieves a value by installation ID and key
func (r *KVStoreRepository) Get(ctx context.Context, installationID uuid.UUID, key string) (json.RawMessage, bool, error) {
	var value json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT value FROM plugin_kv_store
		WHERE installation_id = $1 AND key = $2`, installationID, key,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// Set inserts or updates a key-value pair. tenant_id is derived from the
// owning plugin installation via subquery so RLS (mig 000126) and the caller
// signature stay decoupled — the installation already encodes the tenancy.
// If installationID does not resolve under the subquery (unknown, deleted, or
// foreign-tenant installation), the INSERT affects zero rows and Set returns
// ErrInstallationNotFound instead of silently no-op'ing.
func (r *KVStoreRepository) Set(ctx context.Context, installationID uuid.UUID, key string, value json.RawMessage) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO plugin_kv_store (id, installation_id, tenant_id, key, value, created_at, updated_at)
		SELECT $1, $2, pi.tenant_id, $3, $4, NOW(), NOW()
		FROM plugin_installations pi
		WHERE pi.id = $2
		ON CONFLICT (installation_id, key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		uuid.New(), installationID, key, value,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInstallationNotFound
	}
	return nil
}

// Delete removes a key-value pair. Deleting a key that does not exist (unknown
// installation or unknown key) is intentionally left as a no-op — unlike Set,
// the caller has no expectation that a delete of an absent entry must succeed
// differently from a delete of an entry that was never there.
func (r *KVStoreRepository) Delete(ctx context.Context, installationID uuid.UUID, key string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM plugin_kv_store WHERE installation_id = $1 AND key = $2`, installationID, key,
	)
	return err
}

// List retrieves all key-value pairs for an installation with optional prefix filter
func (r *KVStoreRepository) List(ctx context.Context, installationID uuid.UUID, keyPrefix string) (map[string]json.RawMessage, error) {
	query := `SELECT key, value FROM plugin_kv_store WHERE installation_id = $1`
	args := []any{installationID}

	if keyPrefix != "" {
		query += ` AND key LIKE $2`
		args = append(args, keyPrefix+"%")
	}
	query += ` ORDER BY key`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value json.RawMessage
		if scanErr := rows.Scan(&key, &value); scanErr != nil {
			return nil, scanErr
		}
		entries[key] = value
	}
	return entries, nil
}
