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

// ErrKeyMissing is returned by Complete when the key no longer exists
// (e.g. the cleanup worker ran between Reserve and Complete).
var ErrKeyMissing = errors.New("idempotency key no longer exists")

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

	// Get returns the record for the given (tenantID, key) pair, or pgx.ErrNoRows if not found.
	Get(ctx context.Context, tenantID uuid.UUID, key string) (*Record, error)

	// Complete stores the response for a previously reserved key.
	// tenantID is required to scope the UPDATE to the correct tenant and prevent
	// cross-tenant cache-replay via the composite PK (tenant_id, key).
	// Returns ErrKeyMissing when the key no longer exists (e.g. cleaned up).
	Complete(ctx context.Context, tenantID uuid.UUID, key string, status int, body []byte) error

	// Cleanup deletes expired records and returns the number of rows deleted.
	Cleanup(ctx context.Context) (int, error)

	// CleanupWithLock acquires pg_try_advisory_lock(lockKey) first, so only one
	// replica runs the delete per tick. Returns 0 without error when the lock
	// is held by another process.
	CleanupWithLock(ctx context.Context, lockKey int64) (int, error)
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
	// INSERT … ON CONFLICT (tenant_id, key) DO UPDATE.
	// Pattern: we use a sentinel column update (expires_at = expires_at, a no-op) so
	// RETURNING fires for both insert and conflict paths, giving us the stored record
	// atomically without a separate SELECT — eliminating the TOCTOU window.
	// `(xmax = 0) AS inserted` tells the two paths apart: a row version produced by our
	// own INSERT carries xmax 0, while the ON CONFLICT branch updates an existing row and
	// leaves the locking transaction id in xmax. Without it a caller that lost the race
	// looks exactly like the winner (both see completed_at IS NULL) and both run the
	// handler — the double-booking the key exists to prevent.
	const insertSQL = `
		INSERT INTO idempotency_keys
			(key, tenant_id, user_id, method, path, request_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, key) DO UPDATE SET expires_at = idempotency_keys.expires_at
		RETURNING key, tenant_id, user_id, method, path, request_hash,
		          response_status, response_body, created_at, completed_at, expires_at,
		          (xmax = 0) AS inserted`

	rec := &Record{}
	var inserted bool
	err := r.pool.QueryRow(ctx, insertSQL, key, tenantID, userID, method, path, hash).Scan(
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
		&inserted,
	)
	if err != nil {
		return nil, err
	}

	// A different hash means the key was used with a different payload (client bug).
	if rec.RequestHash != hash {
		return nil, ErrConflict
	}

	if rec.CompletedAt == nil {
		if inserted {
			// We created the reservation — the caller owns it and may run the handler.
			return nil, nil
		}
		// The row already existed and nobody has stored a response yet: another
		// request holds this key right now.
		return nil, ErrInFlight
	}

	// Completed replay — return existing record so middleware can serve cached response.
	return rec, nil
}

func (r *postgresRepository) Get(ctx context.Context, tenantID uuid.UUID, key string) (*Record, error) {
	const q = `
		SELECT key, tenant_id, user_id, method, path, request_hash,
		       response_status, response_body, created_at, completed_at, expires_at
		FROM idempotency_keys
		WHERE tenant_id = $1 AND key = $2`

	rec := &Record{}
	err := r.pool.QueryRow(ctx, q, tenantID, key).Scan(
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

func (r *postgresRepository) Complete(ctx context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	const q = `
		UPDATE idempotency_keys
		SET response_status = $3,
		    response_body   = $4,
		    completed_at    = NOW()
		WHERE tenant_id = $1 AND key = $2`

	tag, err := r.pool.Exec(ctx, q, tenantID, key, status, body)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Either the cleanup worker ran between Reserve and Complete (key is gone),
		// or the tenantID does not match (cross-tenant attempt — treated as missing).
		return ErrKeyMissing
	}
	return nil
}

func (r *postgresRepository) Cleanup(ctx context.Context) (int, error) {
	const q = `DELETE FROM idempotency_keys WHERE expires_at < NOW()`
	tag, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *postgresRepository) CleanupWithLock(ctx context.Context, lockKey int64) (int, error) {
	// A session-level advisory lock belongs to the connection that took it.
	// Acquiring through r.pool and releasing through r.pool hands out two
	// different pooled connections whenever more than one is warm, so the
	// unlock targets a connection that never held the lock: it silently
	// returns false and the lock survives until its connection is recycled.
	// Every later tick then finds the lock held and skips — the cleanup stops
	// running altogether. One connection therefore carries lock and unlock.
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	// pg_try_advisory_lock returns true if we acquired the session-level lock,
	// false if another session already holds it. We release immediately after the
	// delete so the lock is not held for the full hour.
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		return 0, err
	}
	if !locked {
		// Another replica is running cleanup — skip this tick.
		return 0, nil
	}
	defer func() {
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey)
	}()

	return r.Cleanup(ctx)
}
