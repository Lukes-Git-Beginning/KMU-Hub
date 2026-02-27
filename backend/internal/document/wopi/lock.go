package wopi

import (
	"context"
	"errors"

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
func (s *LockService) Lock(ctx context.Context, fileID, lockID, userID string) error {
	query := `
		INSERT INTO document_wopi_locks (file_id, lock_id, locked_by, created_at, expires_at)
		VALUES ($1, $2, $3, NOW(), NOW() + INTERVAL '30 minutes')
		ON CONFLICT (file_id) DO UPDATE
		SET lock_id = $2, locked_by = $3, expires_at = NOW() + INTERVAL '30 minutes'
		WHERE document_wopi_locks.lock_id = $2
		   OR document_wopi_locks.expires_at < NOW()
	`

	result, err := s.pool.Exec(ctx, query, fileID, lockID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Lock exists with different lockID and not expired -- conflict
		currentLock, getErr := s.GetLock(ctx, fileID)
		if getErr != nil {
			return getErr
		}
		return &LockConflictError{CurrentLockID: currentLock}
	}

	return nil
}

// Unlock releases a lock on a file. Only succeeds if the lockID matches.
func (s *LockService) Unlock(ctx context.Context, fileID, lockID string) error {
	query := `DELETE FROM document_wopi_locks WHERE file_id = $1 AND lock_id = $2`
	result, err := s.pool.Exec(ctx, query, fileID, lockID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrLockNotFound
	}

	return nil
}

// RefreshLock extends the lock expiry by 30 minutes. Only succeeds if the lockID matches.
func (s *LockService) RefreshLock(ctx context.Context, fileID, lockID string) error {
	query := `
		UPDATE document_wopi_locks
		SET expires_at = NOW() + INTERVAL '30 minutes'
		WHERE file_id = $1 AND lock_id = $2
	`
	result, err := s.pool.Exec(ctx, query, fileID, lockID)
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
func (s *LockService) GetLock(ctx context.Context, fileID string) (string, error) {
	query := `
		SELECT lock_id FROM document_wopi_locks
		WHERE file_id = $1 AND expires_at > NOW()
	`
	var lockID string
	if err := s.pool.QueryRow(ctx, query, fileID).Scan(&lockID); err != nil {
		return "", ErrLockNotFound
	}
	return lockID, nil
}

// CleanExpired removes all expired locks and returns the count of deleted rows.
func (s *LockService) CleanExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM document_wopi_locks WHERE expires_at < NOW()`
	result, err := s.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
