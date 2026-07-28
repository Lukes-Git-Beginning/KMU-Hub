package vault

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for encrypted vault secrets.
type Repository interface {
	// GetByKeyName retrieves a vault secret by its unique key name, scoped to tenantID.
	// Returns the full record including encrypted_value for service-layer decryption.
	GetByKeyName(ctx context.Context, tenantID uuid.UUID, keyName string) (*models.VaultSecret, error)

	// List returns all vault secrets for tenantID with encrypted_value redacted (empty string).
	// Actual decryption happens in the service layer via GetByKeyName.
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.VaultSecret, error)

	// Create inserts a new vault secret.
	Create(ctx context.Context, secret *models.VaultSecret) error

	// Update modifies an existing vault secret's encrypted value and metadata, scoped to secret.TenantID.
	Update(ctx context.Context, secret *models.VaultSecret) error

	// Delete removes a vault secret by ID, scoped to tenantID.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
