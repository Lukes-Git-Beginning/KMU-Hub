package password

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for password policies and history.
type Repository interface {
	// GetPolicy returns the current password policy for tenantID.
	// If no policy row exists, returns a sensible default (NIST-aligned).
	GetPolicy(ctx context.Context, tenantID uuid.UUID) (*models.PasswordPolicy, error)

	// UpdatePolicy updates the password policy row, scoped to tenantID.
	UpdatePolicy(ctx context.Context, tenantID uuid.UUID, policy *models.PasswordPolicy) error

	// AddPasswordHistory records a password hash for reuse prevention.
	AddPasswordHistory(ctx context.Context, userID uuid.UUID, passwordHash string) error

	// GetPasswordHistory returns the most recent N password hashes for a user,
	// ordered newest first.
	GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
}
