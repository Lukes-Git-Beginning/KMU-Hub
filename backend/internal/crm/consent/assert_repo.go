package consent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAssertRepo implements AssertRepo using PostgreSQL.
// It is intentionally separate from PostgresRepository to keep the
// consent-gate dependency surface minimal.
type PostgresAssertRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresAssertRepo creates a PostgresAssertRepo backed by the given pool.
func NewPostgresAssertRepo(pool *pgxpool.Pool) *PostgresAssertRepo {
	return &PostgresAssertRepo{pool: pool}
}

// ActiveConsent returns true when the contact's most-recent consent record for
// the given channel has granted=true and revoked_at IS NULL.
func (r *PostgresAssertRepo) ActiveConsent(ctx context.Context, contactID uuid.UUID, channel Channel) (bool, error) {
	const query = `
		SELECT granted, revoked_at
		FROM consent_records
		WHERE contact_id = $1 AND consent_type = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var granted bool
	var revokedAt *time.Time

	err := r.pool.QueryRow(ctx, query, contactID, string(channel)).Scan(&granted, &revokedAt)
	if err != nil {
		// No row found → no consent on record.
		return false, nil //nolint:nilerr
	}

	// Active: granted=true and not yet revoked.
	return granted && revokedAt == nil, nil
}

// ContactHasEmail returns true when the contacts table has a non-empty email
// address for the given contact ID.
func (r *PostgresAssertRepo) ContactHasEmail(ctx context.Context, contactID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM contacts
			WHERE id = $1 AND email IS NOT NULL AND email != ''
		)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, contactID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
