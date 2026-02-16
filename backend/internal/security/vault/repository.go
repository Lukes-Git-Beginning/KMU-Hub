package vault

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for encrypted vault secrets.
type Repository interface {
	// GetByKeyName retrieves a vault secret by its unique key name.
	// Returns the full record including encrypted_value for service-layer decryption.
	GetByKeyName(ctx context.Context, keyName string) (*models.VaultSecret, error)

	// List returns all vault secrets with encrypted_value redacted (empty string).
	// Actual decryption happens in the service layer via GetByKeyName.
	List(ctx context.Context) ([]*models.VaultSecret, error)

	// Create inserts a new vault secret.
	Create(ctx context.Context, secret *models.VaultSecret) error

	// Update modifies an existing vault secret's encrypted value and metadata.
	Update(ctx context.Context, secret *models.VaultSecret) error

	// Delete removes a vault secret by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
