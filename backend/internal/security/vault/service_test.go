package vault

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// testTenantID is the default-tenant sentinel used in unit tests.
var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testCtx returns a context pre-loaded with the test tenant so that
// middleware.GetTenantID succeeds inside service methods.
func testCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, testTenantID.String())
}

const testMasterSecret = "this-is-a-test-master-secret-key!"

// MockRepository implements Repository for testing with error injection.
type MockRepository struct {
	secrets   map[string]*models.VaultSecret // keyName -> secret
	secretsID map[uuid.UUID]*models.VaultSecret

	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		secrets:   make(map[string]*models.VaultSecret),
		secretsID: make(map[uuid.UUID]*models.VaultSecret),
	}
}

func (m *MockRepository) GetByKeyName(ctx context.Context, tenantID uuid.UUID, keyName string) (*models.VaultSecret, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.secrets[keyName]
	if !ok {
		return nil, ErrSecretNotFound
	}
	return s, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.VaultSecret, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*models.VaultSecret
	for _, s := range m.secrets {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockRepository) Create(ctx context.Context, secret *models.VaultSecret) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.secrets[secret.KeyName] = secret
	m.secretsID[secret.ID] = secret
	return nil
}

func (m *MockRepository) Update(ctx context.Context, secret *models.VaultSecret) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.secrets[secret.KeyName] = secret
	m.secretsID[secret.ID] = secret
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	s, ok := m.secretsID[id]
	if !ok {
		return ErrSecretNotFound
	}
	delete(m.secrets, s.KeyName)
	delete(m.secretsID, id)
	return nil
}

// newTestService creates a Service with the test master secret, failing the test on error.
func newTestService(t *testing.T, repo *MockRepository) *Service {
	t.Helper()
	svc, err := NewService(repo, testMasterSecret)
	require.NoError(t, err)
	return svc
}

// ============================================================================
// NewService Tests
// ============================================================================

func TestNewService_Success(t *testing.T) {
	repo := NewMockRepository()
	svc, err := NewService(repo, testMasterSecret)

	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Len(t, svc.vaultKey, 32)
	assert.Len(t, svc.totpKey, 32)
	// Vault and TOTP keys must be different (derived with different contexts)
	assert.NotEqual(t, svc.vaultKey, svc.totpKey)
}

func TestNewService_ShortSecret(t *testing.T) {
	repo := NewMockRepository()
	svc, err := NewService(repo, "too-short")

	assert.Nil(t, svc)
	assert.ErrorIs(t, err, ErrMasterSecretTooShort)
}

// ============================================================================
// GetSecret Tests
// ============================================================================

func TestGetSecret_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)
	ctx := testCtx()

	// Set a secret first, then retrieve it (round-trip)
	createdBy := uuid.New()
	err := svc.SetSecret(ctx, "db-password", "super-secret-value", "Database password", createdBy)
	require.NoError(t, err)

	plaintext, err := svc.GetSecret(ctx, "db-password")

	require.NoError(t, err)
	assert.Equal(t, "super-secret-value", plaintext)
}

func TestGetSecret_EmptyKeyName(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	_, err := svc.GetSecret(context.Background(), "")

	assert.ErrorIs(t, err, ErrEmptyKeyName)
}

func TestGetSecret_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	_, err := svc.GetSecret(testCtx(), "nonexistent")

	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestGetSecret_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	repoErr := errors.New("database connection lost")
	repo.getErr = repoErr

	_, err := svc.GetSecret(testCtx(), "any-key")

	assert.ErrorIs(t, err, repoErr)
}

// ============================================================================
// SetSecret Tests
// ============================================================================

func TestSetSecret_CreateNew(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)
	ctx := testCtx()
	createdBy := uuid.New()

	err := svc.SetSecret(ctx, "api-key", "my-api-key-value", "External API key", createdBy)

	require.NoError(t, err)

	// Verify stored in repo
	stored, ok := repo.secrets["api-key"]
	require.True(t, ok)
	assert.Equal(t, "api-key", stored.KeyName)
	assert.Equal(t, "External API key", stored.Description)
	assert.Equal(t, &createdBy, stored.CreatedBy)
	assert.Equal(t, 1, stored.KeyVersion)
	assert.NotEmpty(t, stored.EncryptedValue)
	// Encrypted value must differ from plaintext
	assert.NotEqual(t, "my-api-key-value", stored.EncryptedValue)
}

func TestSetSecret_UpdateExisting(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)
	ctx := testCtx()
	createdBy := uuid.New()

	// Create initial secret
	err := svc.SetSecret(ctx, "api-key", "old-value", "Old description", createdBy)
	require.NoError(t, err)

	originalID := repo.secrets["api-key"].ID

	// Update with new value
	err = svc.SetSecret(ctx, "api-key", "new-value", "New description", createdBy)
	require.NoError(t, err)

	// ID should remain the same (update, not create)
	stored := repo.secrets["api-key"]
	assert.Equal(t, originalID, stored.ID)
	assert.Equal(t, "New description", stored.Description)

	// Round-trip: verify the new value is retrievable
	plaintext, err := svc.GetSecret(ctx, "api-key")
	require.NoError(t, err)
	assert.Equal(t, "new-value", plaintext)
}

func TestSetSecret_EmptyKeyName(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	err := svc.SetSecret(context.Background(), "", "value", "desc", uuid.New())

	assert.ErrorIs(t, err, ErrEmptyKeyName)
}

func TestSetSecret_EmptyPlaintext(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	err := svc.SetSecret(context.Background(), "key", "", "desc", uuid.New())

	assert.ErrorIs(t, err, ErrEmptyPayload)
}

func TestSetSecret_RepoCreateError(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	repoErr := errors.New("disk full")
	repo.createErr = repoErr

	err := svc.SetSecret(testCtx(), "key", "value", "desc", uuid.New())

	assert.ErrorIs(t, err, repoErr)
}

// ============================================================================
// ListSecrets Tests
// ============================================================================

func TestListSecrets_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)
	ctx := testCtx()

	// Populate via SetSecret for realistic data
	require.NoError(t, svc.SetSecret(ctx, "key-1", "val-1", "First", uuid.New()))
	require.NoError(t, svc.SetSecret(ctx, "key-2", "val-2", "Second", uuid.New()))

	secrets, err := svc.ListSecrets(ctx)

	require.NoError(t, err)
	assert.Len(t, secrets, 2)
}

func TestListSecrets_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	secrets, err := svc.ListSecrets(testCtx())

	require.NoError(t, err)
	assert.Empty(t, secrets)
}

// ============================================================================
// DeleteSecret Tests
// ============================================================================

func TestDeleteSecret_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)
	ctx := testCtx()

	require.NoError(t, svc.SetSecret(ctx, "to-delete", "val", "desc", uuid.New()))
	secretID := repo.secrets["to-delete"].ID

	err := svc.DeleteSecret(ctx, secretID)

	require.NoError(t, err)
	assert.Empty(t, repo.secrets)
}

func TestDeleteSecret_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	// Deleting a non-existent ID returns ErrSecretNotFound from mock
	err := svc.DeleteSecret(testCtx(), uuid.New())

	assert.ErrorIs(t, err, ErrSecretNotFound)
}

// ============================================================================
// TOTP Encrypt/Decrypt Tests
// ============================================================================

func TestEncryptDecryptTOTP_RoundTrip(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	original := "JBSWY3DPEHPK3PXP"

	encrypted, err := svc.EncryptTOTPSecret(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, original, encrypted)

	decrypted, err := svc.DecryptTOTPSecret(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestEncryptTOTP_EmptySecret(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	_, err := svc.EncryptTOTPSecret("")

	assert.ErrorIs(t, err, ErrEmptyPayload)
}

func TestDecryptTOTP_EmptyInput(t *testing.T) {
	repo := NewMockRepository()
	svc := newTestService(t, repo)

	_, err := svc.DecryptTOTPSecret("")

	assert.ErrorIs(t, err, ErrEmptyPayload)
}
