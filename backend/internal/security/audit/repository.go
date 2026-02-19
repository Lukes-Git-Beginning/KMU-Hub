package audit

import (
	"context"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the data access interface for the audit log.
type Repository interface {
	// Create inserts a new audit entry with hash chain integrity (uses advisory lock).
	Create(ctx context.Context, entry *models.AuditEntry) error

	// List returns audit entries matching the filter, plus total count for pagination.
	List(ctx context.Context, filter *models.AuditFilter) ([]*models.AuditEntry, int, error)

	// GetLastHash returns the entry_hash of the most recent audit entry, or empty string if none.
	GetLastHash(ctx context.Context) (string, error)

	// VerifyChain checks hash chain integrity from fromSequence to toSequence.
	// Returns (valid, firstBrokenSequence, error). If valid, firstBrokenSequence is 0.
	VerifyChain(ctx context.Context, fromSequence, toSequence int64) (bool, int64, error)
}
