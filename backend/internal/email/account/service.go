package account

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// VaultEncryptor encrypts and decrypts email credentials at rest via the vault service.
type VaultEncryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) (string, error)
	Decrypt(ctx context.Context, encrypted string) ([]byte, error)
}

// Credentials holds decrypted IMAP/SMTP connection credentials.
type Credentials struct {
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	UseSSL   bool
}

// CreateInput contains the data needed to create an email account.
type CreateInput struct {
	UserID       uuid.UUID
	EmailAddress string
	DisplayName  string
	IMAPHost     string
	IMAPPort     int
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	UseSSL       bool
}

// UpdateInput contains the data that can be updated on an email account.
type UpdateInput struct {
	EmailAddress *string
	DisplayName  *string
	IMAPHost     *string
	IMAPPort     *int
	SMTPHost     *string
	SMTPPort     *int
	Username     *string
	Password     *string
	UseSSL       *bool
	SyncEnabled  *bool
}

// Service handles email account business logic.
type Service struct {
	repo      Repository
	encryptor VaultEncryptor
}

// NewService creates a new email account service.
func NewService(repo Repository, encryptor VaultEncryptor) *Service {
	return &Service{
		repo:      repo,
		encryptor: encryptor,
	}
}

// Create creates a new email account with encrypted credentials.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, input CreateInput) (*models.EmailAccount, error) {
	// Validate email address
	trimmed := strings.TrimSpace(input.EmailAddress)
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return nil, fmt.Errorf("invalid email address: %w", err)
	}

	// The first account a user creates within a tenant becomes their default;
	// every account after that is added alongside the existing ones.
	existing, err := s.repo.ListByUserAndTenant(ctx, input.UserID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing accounts: %w", err)
	}
	isDefault := len(existing) == 0

	// Validate required fields
	if strings.TrimSpace(input.IMAPHost) == "" {
		return nil, fmt.Errorf("IMAP host is required")
	}
	if strings.TrimSpace(input.SMTPHost) == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if strings.TrimSpace(input.Username) == "" {
		return nil, fmt.Errorf("username is required")
	}
	if input.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Encrypt password
	encrypted, err := s.encryptor.Encrypt(ctx, []byte(input.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	now := time.Now().UTC()
	account := &models.EmailAccount{
		ID:                uuid.New(),
		TenantID:          tenantID,
		UserID:            input.UserID,
		EmailAddress:      trimmed,
		DisplayName:       strings.TrimSpace(input.DisplayName),
		IMAPHost:          strings.TrimSpace(input.IMAPHost),
		IMAPPort:          input.IMAPPort,
		SMTPHost:          strings.TrimSpace(input.SMTPHost),
		SMTPPort:          input.SMTPPort,
		Username:          strings.TrimSpace(input.Username),
		PasswordEncrypted: encrypted,
		UseSSL:            input.UseSSL,
		SyncEnabled:       true,
		IsDefault:         isDefault,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if account.IMAPPort == 0 {
		account.IMAPPort = 993
	}
	if account.SMTPPort == 0 {
		account.SMTPPort = 587
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	slog.Info("email account created",
		"account_id", account.ID,
		"user_id", account.UserID,
		"email", account.EmailAddress,
		"is_default", account.IsDefault,
	)

	// Return without password
	account.PasswordEncrypted = ""
	return account, nil
}

// GetByID retrieves an email account by ID (without decrypted password).
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	account, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	account.PasswordEncrypted = ""
	return account, nil
}

// GetByUserIDAndTenant retrieves the user's default account (without decrypted password).
func (s *Service) GetByUserIDAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	account, err := s.repo.GetByUserIDAndTenant(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	account.PasswordEncrypted = ""
	return account, nil
}

// ListByUserAndTenant returns every account a user holds within a tenant,
// oldest first, without decrypted passwords.
func (s *Service) ListByUserAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	accounts, err := s.repo.ListByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	for _, a := range accounts {
		a.PasswordEncrypted = ""
	}
	return accounts, nil
}

// SetDefault marks an account as the default for its user, clearing any prior default.
func (s *Service) SetDefault(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	acct, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if err := s.repo.SetDefault(ctx, id, acct.UserID, tenantID); err != nil {
		return fmt.Errorf("failed to set default account: %w", err)
	}

	slog.Info("email account set as default",
		"account_id", id,
		"user_id", acct.UserID,
		"tenant_id", tenantID,
	)

	return nil
}

// GetDecryptedCredentials returns the decrypted IMAP/SMTP credentials for connection.
func (s *Service) GetDecryptedCredentials(ctx context.Context, accountID uuid.UUID, tenantID uuid.UUID) (*Credentials, error) {
	account, err := s.repo.GetByID(ctx, accountID, tenantID)
	if err != nil {
		return nil, err
	}

	password, err := s.encryptor.Decrypt(ctx, account.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return &Credentials{
		IMAPHost: account.IMAPHost,
		IMAPPort: account.IMAPPort,
		SMTPHost: account.SMTPHost,
		SMTPPort: account.SMTPPort,
		Username: account.Username,
		Password: string(password),
		UseSSL:   account.UseSSL,
	}, nil
}

// Update updates an existing email account.
func (s *Service) Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, input UpdateInput) (*models.EmailAccount, error) {
	account, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if input.EmailAddress != nil {
		trimmed := strings.TrimSpace(*input.EmailAddress)
		if _, parseErr := mail.ParseAddress(trimmed); parseErr != nil {
			return nil, fmt.Errorf("invalid email address: %w", parseErr)
		}
		account.EmailAddress = trimmed
	}
	if input.DisplayName != nil {
		account.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.IMAPHost != nil {
		account.IMAPHost = strings.TrimSpace(*input.IMAPHost)
	}
	if input.IMAPPort != nil {
		account.IMAPPort = *input.IMAPPort
	}
	if input.SMTPHost != nil {
		account.SMTPHost = strings.TrimSpace(*input.SMTPHost)
	}
	if input.SMTPPort != nil {
		account.SMTPPort = *input.SMTPPort
	}
	if input.Username != nil {
		account.Username = strings.TrimSpace(*input.Username)
	}
	if input.Password != nil && *input.Password != "" {
		encrypted, encErr := s.encryptor.Encrypt(ctx, []byte(*input.Password))
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", encErr)
		}
		account.PasswordEncrypted = encrypted
	}
	if input.UseSSL != nil {
		account.UseSSL = *input.UseSSL
	}
	if input.SyncEnabled != nil {
		account.SyncEnabled = *input.SyncEnabled
	}

	if err := s.repo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	slog.Info("email account updated",
		"account_id", account.ID,
		"email", account.EmailAddress,
	)

	account.PasswordEncrypted = ""
	return account, nil
}

// Delete removes an email account. If it was the user's default and other
// accounts remain, the oldest remaining account is promoted to default --
// callers that resolve "the" account for a user (GetByUserIDAndTenant, and
// through it the notification adapter) depend on a default always existing
// as long as the user holds at least one account.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	if account.IsDefault {
		remaining, err := s.repo.ListByUserAndTenant(ctx, account.UserID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to list remaining accounts: %w", err)
		}
		if len(remaining) > 0 {
			if err := s.repo.SetDefault(ctx, remaining[0].ID, account.UserID, tenantID); err != nil {
				return fmt.Errorf("failed to promote new default account: %w", err)
			}
			slog.Info("email account promoted to default after deletion",
				"account_id", remaining[0].ID,
				"user_id", account.UserID,
				"tenant_id", tenantID,
			)
		}
	}

	slog.Info("email account deleted",
		"account_id", account.ID,
		"user_id", account.UserID,
		"tenant_id", tenantID,
	)

	return nil
}

// TestConnection attempts an IMAP login with the provided credentials to verify they work.
func (s *Service) TestConnection(ctx context.Context, input CreateInput) error {
	addr := fmt.Sprintf("%s:%d", input.IMAPHost, input.IMAPPort)

	var client *imapclient.Client
	var err error

	if input.UseSSL || input.IMAPPort == 993 {
		client, err = imapclient.DialTLS(addr, nil)
	} else {
		client, err = imapclient.DialStartTLS(addr, nil)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer client.Close()

	if err := client.Login(input.Username, input.Password).Wait(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}

	if err := client.Logout().Wait(); err != nil {
		slog.Warn("IMAP logout failed during test connection", "error", err)
	}

	slog.Info("email connection test successful",
		"imap_host", input.IMAPHost,
		"username", input.Username,
	)

	return nil
}

// ListActive returns all sync-enabled accounts for a tenant (for sync engine startup).
func (s *Service) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	return s.repo.ListActive(ctx, tenantID)
}

// ListAllActive returns all sync-enabled accounts across all tenants (for sync engine bootstrap).
func (s *Service) ListAllActive(ctx context.Context) ([]*models.EmailAccount, error) {
	return s.repo.ListAllActive(ctx)
}

// UpdateLastSync updates the last_sync_at timestamp for an account.
func (s *Service) UpdateLastSync(ctx context.Context, accountID uuid.UUID, tenantID uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, accountID, tenantID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	account.LastSyncAt = &now

	return s.repo.Update(ctx, account)
}
