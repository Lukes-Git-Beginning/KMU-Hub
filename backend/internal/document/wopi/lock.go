package wopi

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Errors for WOPI lock operations.
var (
	ErrLockNotFound = errors.New("lock not found")
)

// LockConflictError is returned when a lock operation conflicts with an existing lock.
type LockConflictError struct {
	CurrentLockID string
}

func (e *LockConflictError) Error() string {
	return "lock conflict: file is locked by another session"
}

// LockService manages WOPI file locks in PostgreSQL with 30-minute auto-expiry.
type LockService struct {
	pool *pgxpool.Pool
}

// NewLockService creates a new WOPI lock service.
func NewLockService(pool *pgxpool.Pool) *LockService {
	return &LockService{pool: pool}
}

// Lock acquires a lock on a file. If the file is already locked by the same lockID,
// the lock is refreshed. If locked by a different lockID that hasn't expired,
// a LockConflictError is returned with the current lock ID.
// ctx must carry the caller's tenant (see middleware.TenantIDKey) so RLS admits
// the row — tenantID is also passed explicitly as a defense-in-depth filter.
func (s *LockService) Lock(ctx context.Context, fileID, lockID, userID string, tenantID uuid.UUID) error {
	query := `
		INSERT INTO wopi_locks (file_id, lock_id, locked_by, tenant_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '30 minutes')
		ON CONFLICT (file_id) DO UPDATE
		SET lock_id = $2, locked_by = $3, expires_at = NOW() + INTERVAL '30 minutes'
		WHERE wopi_locks.tenant_id = $4
		  AND (wopi_locks.lock_id = $2 OR wopi_locks.expires_at < NOW())
	`

	result, err := s.pool.Exec(ctx, query, fileID, lockID, userID, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Lock exists with different lockID and not expired -- conflict
		currentLock, getErr := s.GetLock(ctx, fileID, tenantID)
		if getErr != nil {
			return getErr
		}
		return &LockConflictError{CurrentLockID: currentLock}
	}

	return nil
}

// Unlock releases a lock on a file. Only succeeds if the lockID matches.
func (s *LockService) Unlock(ctx context.Context, fileID, lockID string, tenantID uuid.UUID) error {
	query := `DELETE FROM wopi_locks WHERE file_id = $1 AND lock_id = $2 AND tenant_id = $3`
	result, err := s.pool.Exec(ctx, query, fileID, lockID, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrLockNotFound
	}

	return nil
}

// RefreshLock extends the lock expiry by 30 minutes. Only succeeds if the lockID matches.
func (s *LockService) RefreshLock(ctx context.Context, fileID, lockID string, tenantID uuid.UUID) error {
	query := `
		UPDATE wopi_locks
		SET expires_at = NOW() + INTERVAL '30 minutes'
		WHERE file_id = $1 AND lock_id = $2 AND tenant_id = $3
	`
	result, err := s.pool.Exec(ctx, query, fileID, lockID, tenantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrLockNotFound
	}

	return nil
}

// GetLock returns the current lock ID for a file, or ErrLockNotFound if not locked.
// Only returns locks that haven't expired.
func (s *LockService) GetLock(ctx context.Context, fileID string, tenantID uuid.UUID) (string, error) {
	query := `
		SELECT lock_id FROM wopi_locks
		WHERE file_id = $1 AND tenant_id = $2 AND expires_at > NOW()
	`
	var lockID string
	if err := s.pool.QueryRow(ctx, query, fileID, tenantID).Scan(&lockID); err != nil {
		return "", ErrLockNotFound
	}
	return lockID, nil
}

// CleanExpired removes all expired locks across every tenant and returns the
// count of deleted rows. Callers MUST wrap ctx with database.WithSystemContext
// (see cmd/document/main.go's cleanup goroutine) — RLS admits no rows for an
// unscoped, non-system context, so CleanExpired would silently delete nothing.
func (s *LockService) CleanExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM wopi_locks WHERE expires_at < NOW()`
	result, err := s.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
