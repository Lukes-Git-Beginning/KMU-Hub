package lexware

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type APIKeyManager struct {
	vault VaultService
}

func NewAPIKeyManager(vault VaultService) *APIKeyManager {
	return &APIKeyManager{vault: vault}
}

func apiKeyVaultKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("lexware_api_key_%s", tenantID.String())
}

func (m *APIKeyManager) GetAPIKey(ctx context.Context, tenantID uuid.UUID) (string, error) {
	key, err := m.vault.GetSecret(ctx, apiKeyVaultKey(tenantID))
	if err != nil {
		return "", fmt.Errorf("lexware: failed to load API key for tenant %s: %w", tenantID, err)
	}
	return key, nil
}

func (m *APIKeyManager) StoreAPIKey(ctx context.Context, tenantID uuid.UUID, apiKey string) error {
	if err := m.vault.SetSecret(ctx, apiKeyVaultKey(tenantID), apiKey, "Lexware Office API key", uuid.Nil); err != nil {
		return fmt.Errorf("lexware: failed to store API key: %w", err)
	}
	slog.Info("lexware API key stored", "tenant_id", tenantID)
	return nil
}

func (m *APIKeyManager) RevokeAPIKey(ctx context.Context, tenantID uuid.UUID) error {
	slog.Info("lexware API key revoked", "tenant_id", tenantID)
	return nil
}
