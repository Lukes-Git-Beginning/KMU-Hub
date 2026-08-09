package lexware

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyManager_StoreAPIKey_Success(t *testing.T) {
	tenantID := uuid.New()
	var gotKeyName, gotPlaintext, gotDescription string
	var gotCreatedBy uuid.UUID
	vault := &mockVaultService{
		setSecretFn: func(_ context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error {
			gotKeyName = keyName
			gotPlaintext = plaintext
			gotDescription = description
			gotCreatedBy = createdBy
			return nil
		},
	}
	m := NewAPIKeyManager(vault)

	err := m.StoreAPIKey(context.Background(), tenantID, "secret-key")

	require.NoError(t, err)
	assert.Equal(t, apiKeyVaultKey(tenantID), gotKeyName)
	assert.Equal(t, "secret-key", gotPlaintext)
	assert.Equal(t, "Lexware Office API key", gotDescription)
	// StoreAPIKey is not attributable to a user (system/service flow) — the
	// vault call must carry uuid.Nil, not the tenant id, as its creator.
	assert.Equal(t, uuid.Nil, gotCreatedBy)
}

func TestAPIKeyManager_StoreAPIKey_VaultError(t *testing.T) {
	vault := &mockVaultService{
		setSecretFn: func(context.Context, string, string, string, uuid.UUID) error {
			return errors.New("vault unreachable")
		},
	}
	m := NewAPIKeyManager(vault)

	err := m.StoreAPIKey(context.Background(), uuid.New(), "secret-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store API key")
}

func TestAPIKeyManager_GetAPIKey_Success(t *testing.T) {
	tenantID := uuid.New()
	vault := &mockVaultService{
		getSecretFn: func(_ context.Context, keyName string) (string, error) {
			assert.Equal(t, apiKeyVaultKey(tenantID), keyName)
			return "the-key", nil
		},
	}
	m := NewAPIKeyManager(vault)

	key, err := m.GetAPIKey(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, "the-key", key)
}

func TestAPIKeyManager_GetAPIKey_VaultError(t *testing.T) {
	vault := &mockVaultService{
		getSecretFn: func(context.Context, string) (string, error) {
			return "", errors.New("secret not found")
		},
	}
	m := NewAPIKeyManager(vault)

	_, err := m.GetAPIKey(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load API key")
}

// TestAPIKeyManager_RevokeAPIKey documents the current behavior: revocation
// only logs today (the vault has no delete-secret call wired), so this test
// pins "never errors" as the contract until a real vault deletion is added.
func TestAPIKeyManager_RevokeAPIKey_NeverErrors(t *testing.T) {
	m := NewAPIKeyManager(&mockVaultService{})

	err := m.RevokeAPIKey(context.Background(), uuid.New())

	require.NoError(t, err)
}
