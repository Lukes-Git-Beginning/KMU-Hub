// Package idempotency implements per-request deduplication backed by PostgreSQL.
// A record is reserved (inserted) before the handler runs and completed
// (response stored) after. Replayed requests receive the cached response without
// hitting the downstream service.
package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrConflict is returned when a key exists with a different request_hash
// (same key, different payload — client bug).
var ErrConflict = errors.New("idempotency key reused with different request")

// ErrInFlight is returned when a key is reserved but not yet completed
// (concurrent in-flight request with same key).
var ErrInFlight = errors.New("idempotency key request in flight")

// Record holds the stored state for a single idempotency key.
type Record struct {
	Key            string
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Method         string
	Path           string
	RequestHash    string
	ResponseStatus *int
	ResponseBody   []byte
	CreatedAt      time.Time
	CompletedAt    *time.Time
	ExpiresAt      time.Time
}

// Repository defines idempotency key persistence operations.
type Repository interface {
	// Reserve inserts a new key record. If the key already exists:
	//   - same hash, no response yet → ErrInFlight
	//   - same hash, response stored → returns existing Record, nil error (replay)
	//   - different hash             → ErrConflict
	Reserve(ctx context.Context, key string, tenantID, userID uuid.UUID, method, path, hash string) (*Record, error)

	// Get returns the record for the given key, or pgx.ErrNoRows if not found.
	Get(ctx context.Context, key string) (*Record, error)

	// Complete stores the response for a previously reserved key.
	Complete(ctx context.Context, key string, status int, body []byte) error

	// Cleanup deletes expired records and returns the number of rows deleted.
	Cleanup(ctx context.Context) (int, error)
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a Repository backed by a pgxpool.
func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Reserve(
	ctx context.Context,
	key string,
	tenantID, userID uuid.UUID,
	method, path, hash string,
) (*Record, error) {
	// INSERT … ON CONFLICT DO NOTHING handles the race: only one writer wins.
	const insertSQL = `
		INSERT INTO idempotency_keys
			(key, tenant_id, user_id, method, path, request_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO NOTHING`

	tag, err := r.pool.Exec(ctx, insertSQL, key, tenantID, userID, method, path, hash)
	if err != nil {
		return nil, err
	}

	if tag.RowsAffected() == 1 {
		// Happy path: we inserted, key is fresh.
		return nil, nil
	}

	// Key already exists — fetch the existing record to determine the state.
	rec, err := r.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if rec.RequestHash != hash {
		return nil, ErrConflict
	}

	if rec.CompletedAt == nil {
		return nil, ErrInFlight
	}

	// Completed replay — return existing record so middleware can serve cached response.
	return rec, nil
}

func (r *postgresRepository) Get(ctx context.Context, key string) (*Record, error) {
	const q = `
		SELECT key, tenant_id, user_id, method, path, request_hash,
		       response_status, response_body, created_at, completed_at, expires_at
		FROM idempotency_keys
		WHERE key = $1`

	rec := &Record{}
	err := r.pool.QueryRow(ctx, q, key).Scan(
		&rec.Key,
		&rec.TenantID,
		&rec.UserID,
		&rec.Method,
		&rec.Path,
		&rec.RequestHash,
		&rec.ResponseStatus,
		&rec.ResponseBody,
		&rec.CreatedAt,
		&rec.CompletedAt,
		&rec.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *postgresRepository) Complete(ctx context.Context, key string, status int, body []byte) error {
	const q = `
		UPDATE idempotency_keys
		SET response_status = $2,
		    response_body   = $3,
		    completed_at    = NOW()
		WHERE key = $1`

	_, err := r.pool.Exec(ctx, q, key, status, body)
	return err
}

func (r *postgresRepository) Cleanup(ctx context.Context) (int, error) {
	const q = `DELETE FROM idempotency_keys WHERE expires_at < NOW()`
	tag, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
