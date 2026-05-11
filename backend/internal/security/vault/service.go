package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	minMasterSecretLength = 32
	currentKeyVersion     = 1
)

var (
	ErrMasterSecretTooShort = errors.New("vault: master secret must be at least 32 characters")
	ErrEmptyKeyName         = errors.New("vault: key name must not be empty")
)

// Service provides encrypted secret storage using AES-256-GCM with HKDF-derived keys.
type Service struct {
	repo     Repository
	vaultKey []byte
	totpKey  []byte
}

// NewService creates a vault service by deriving separate encryption keys from the master secret.
// The vaultKey is used for general secret storage, the totpKey for 2FA TOTP secret encryption.
func NewService(repo Repository, masterSecret string) (*Service, error) {
	if len(masterSecret) < minMasterSecretLength {
		return nil, ErrMasterSecretTooShort
	}

	masterBytes := []byte(masterSecret)

	vaultKey, err := DeriveKey(masterBytes, KeyContextVault)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to derive vault key: %w", err)
	}

	totpKey, err := DeriveKey(masterBytes, KeyContextTOTP)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to derive TOTP key: %w", err)
	}

	slog.Info("vault service initialized", "key_contexts", []string{KeyContextVault, KeyContextTOTP})

	return &Service{
		repo:     repo,
		vaultKey: vaultKey,
		totpKey:  totpKey,
	}, nil
}

// GetSecret retrieves and decrypts a vault secret by key name.
func (s *Service) GetSecret(ctx context.Context, keyName string) (string, error) {
	if keyName == "" {
		return "", ErrEmptyKeyName
	}

	secret, err := s.repo.GetByKeyName(ctx, keyName)
	if err != nil {
		return "", err
	}

	plaintext, err := Decrypt(secret.EncryptedValue, s.vaultKey)
	if err != nil {
		slog.Error("vault decryption failed", "key_name", keyName, "error", err)
		return "", fmt.Errorf("vault: failed to decrypt secret %q: %w", keyName, err)
	}

	slog.Info("vault secret retrieved", "key_name", keyName)
	return string(plaintext), nil
}

// SetSecret encrypts and stores a secret. Creates a new entry or updates an existing one.
func (s *Service) SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error {
	if keyName == "" {
		return ErrEmptyKeyName
	}
	if plaintext == "" {
		return ErrEmptyPayload
	}

	encrypted, err := Encrypt([]byte(plaintext), s.vaultKey)
	if err != nil {
		return fmt.Errorf("vault: failed to encrypt secret %q: %w", keyName, err)
	}

	now := time.Now().UTC()

	// Check if secret already exists
	existing, err := s.repo.GetByKeyName(ctx, keyName)
	if err != nil && !errors.Is(err, ErrSecretNotFound) {
		return err
	}

	if existing != nil {
		existing.EncryptedValue = encrypted
		existing.Description = description
		existing.UpdatedAt = now
		if err := s.repo.Update(ctx, existing); err != nil {
			return err
		}
		slog.Info("vault secret updated", "key_name", keyName)
		return nil
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return fmt.Errorf("vault: failed to determine tenant: %w", err)
	}

	secret := &models.VaultSecret{
		ID:             uuid.New(),
		TenantID:       tenantID,
		KeyName:        keyName,
		EncryptedValue: encrypted,
		KeyVersion:     currentKeyVersion,
		Description:    description,
		CreatedBy:      &createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, secret); err != nil {
		return err
	}

	slog.Info("vault secret created", "key_name", keyName)
	return nil
}

// ListSecrets returns metadata for all vault secrets without decrypted values.
func (s *Service) ListSecrets(ctx context.Context) ([]*models.VaultSecret, error) {
	secrets, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	slog.Info("vault secrets listed", "count", len(secrets))
	return secrets, nil
}

// DeleteSecret removes a vault secret by ID.
func (s *Service) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	slog.Info("vault secret deleted", "secret_id", id)
	return nil
}

// EncryptTOTPSecret encrypts a TOTP secret using the dedicated TOTP key.
// This keeps TOTP encryption separate from general vault encryption.
func (s *Service) EncryptTOTPSecret(secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptyPayload
	}
	return Encrypt([]byte(secret), s.totpKey)
}

// DecryptTOTPSecret decrypts a TOTP secret using the dedicated TOTP key.
func (s *Service) DecryptTOTPSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", ErrEmptyPayload
	}
	plaintext, err := Decrypt(encrypted, s.totpKey)
	if err != nil {
		return "", fmt.Errorf("vault: failed to decrypt TOTP secret: %w", err)
	}
	return string(plaintext), nil
}
