package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	totpIssuer         = "KMU Hub"
	totpDigits         = otp.DigitsSix
	totpAlgorithm      = otp.AlgorithmSHA1
	totpPeriod         = 30
	recoveryCodeCount  = 8
	recoveryCodeLength = 10 // hex chars (5 bytes)
)

// VaultEncryptor encrypts and decrypts TOTP secrets at rest via the vault service.
type VaultEncryptor interface {
	EncryptTOTPSecret(ctx context.Context, plainSecret string) (string, error)
	DecryptTOTPSecret(ctx context.Context, encryptedSecret string) (string, error)
}

// SetVaultService sets the vault encryption service for TOTP secret storage.
// When nil, TOTP secrets are stored as-is (for development only).
func (s *Service) SetVaultService(vaultSvc VaultEncryptor) {
	s.vaultSvc = vaultSvc
}

// Setup2FAResult contains the QR code and secret for 2FA enrollment.
type Setup2FAResult struct {
	Secret    string `json:"secret"`
	QRCodePNG []byte `json:"-"`
	OTPURL    string `json:"otp_url"`
}

// Setup2FA initiates TOTP enrollment for a user. Returns a QR code image and the secret.
// The user must verify with a valid code before 2FA is activated.
func (s *Service) Setup2FA(ctx context.Context, userID uuid.UUID) (*Setup2FAResult, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.TwoFactorEnabled {
		return nil, ErrTwoFactorAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: user.Email,
		Digits:      totpDigits,
		Algorithm:   totpAlgorithm,
		Period:      totpPeriod,
	})
	if err != nil {
		return nil, fmt.Errorf("generate TOTP key: %w", err)
	}

	// Generate QR code PNG
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("generate QR image: %w", err)
	}

	var qrBuf bytes.Buffer
	if err := png.Encode(&qrBuf, img); err != nil {
		return nil, fmt.Errorf("encode QR PNG: %w", err)
	}

	// Encrypt and store pending secret
	encrypted, err := s.encryptSecret(ctx, key.Secret())
	if err != nil {
		return nil, fmt.Errorf("encrypt TOTP secret: %w", err)
	}

	if err := s.repo.StorePending2FASecret(ctx, userID, encrypted); err != nil {
		return nil, fmt.Errorf("store pending 2FA secret: %w", err)
	}

	slog.Info("2FA setup initiated", "user_id", userID)
	return &Setup2FAResult{
		Secret:    key.Secret(),
		QRCodePNG: qrBuf.Bytes(),
		OTPURL:    key.URL(),
	}, nil
}

// Verify2FA completes the 2FA setup by validating the user's first TOTP code.
// On success, generates and returns recovery codes, and enables 2FA on the account.
func (s *Service) Verify2FA(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.TwoFactorEnabled {
		return nil, ErrTwoFactorAlreadyEnabled
	}

	// Get and decrypt the pending secret
	encryptedSecret, err := s.repo.GetPending2FASecret(ctx, userID)
	if err != nil {
		return nil, err
	}

	secret, err := s.decryptSecret(ctx, encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt pending secret: %w", err)
	}

	// Validate the TOTP code
	valid := totp.Validate(code, secret)
	if !valid {
		return nil, ErrInvalidTOTPCode
	}

	// Generate recovery codes
	plainCodes, hashedCodes, err := generateRecoveryCodes(userID)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	// Enable 2FA transactionally
	if err := s.repo.Enable2FA(ctx, userID, encryptedSecret, hashedCodes); err != nil {
		return nil, fmt.Errorf("enable 2FA: %w", err)
	}

	slog.Info("2FA enabled", "user_id", userID)
	return plainCodes, nil
}

// Validate2FALogin validates either a TOTP code or a recovery code during login.
// Returns true if a recovery code was used (for informational purposes).
func (s *Service) Validate2FALogin(ctx context.Context, userID uuid.UUID, code string, isRecoveryCode bool) (recoveryCodeUsed bool, err error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return false, ErrUserNotFound
	}

	if !user.TwoFactorEnabled {
		return false, ErrTwoFactorNotEnabled
	}

	if isRecoveryCode {
		return true, s.validateRecoveryCode(ctx, userID, code)
	}

	return false, s.validateTOTPCode(ctx, user, code)
}

// Disable2FA disables two-factor authentication for a user.
// The user must provide a valid TOTP code to confirm.
func (s *Service) Disable2FA(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.TwoFactorEnabled {
		return ErrTwoFactorNotEnabled
	}

	// Validate the code before disabling
	if err := s.validateTOTPCode(ctx, user, code); err != nil {
		return err
	}

	if err := s.repo.Disable2FA(ctx, userID); err != nil {
		return fmt.Errorf("disable 2FA: %w", err)
	}

	slog.Info("2FA disabled by user", "user_id", userID)
	return nil
}

// RegenerateRecoveryCodes generates a new set of recovery codes, replacing old ones.
// The user must provide a valid TOTP code to confirm.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.TwoFactorEnabled {
		return nil, ErrTwoFactorNotEnabled
	}

	// Validate code before regeneration
	if err := s.validateTOTPCode(ctx, user, code); err != nil {
		return nil, err
	}

	plainCodes, hashedCodes, err := generateRecoveryCodes(userID)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	if err := s.repo.ReplaceRecoveryCodes(ctx, userID, hashedCodes); err != nil {
		return nil, fmt.Errorf("replace recovery codes: %w", err)
	}

	slog.Info("recovery codes regenerated", "user_id", userID)
	return plainCodes, nil
}

// AdminReset2FA disables 2FA on a user's account by an admin. Requires a non-empty reason.
func (s *Service) AdminReset2FA(ctx context.Context, targetUserID uuid.UUID, adminUserID uuid.UUID, reason string) error {
	if reason == "" {
		return fmt.Errorf("reason is required for admin 2FA reset")
	}

	user, err := s.repo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.TwoFactorEnabled {
		return ErrTwoFactorNotEnabled
	}

	if err := s.repo.Disable2FA(ctx, targetUserID); err != nil {
		return fmt.Errorf("admin disable 2FA: %w", err)
	}

	slog.Warn("2FA reset by admin",
		"target_user_id", targetUserID,
		"admin_user_id", adminUserID,
		"reason", reason,
	)
	return nil
}

// TwoFactorEnforcementResult describes the 2FA enforcement status for a user.
type TwoFactorEnforcementResult struct {
	Required        bool       `json:"required"`
	GraceDeadline   *time.Time `json:"grace_deadline,omitempty"`
	GraceExpired    bool       `json:"grace_expired"`
	EnforcedByRoles []string   `json:"enforced_by_roles,omitempty"`
}

// Check2FAEnforcement checks if 2FA is required for a user based on role policies.
func (s *Service) Check2FAEnforcement(ctx context.Context, userID uuid.UUID) (*TwoFactorEnforcementResult, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// If 2FA is already enabled, enforcement is satisfied
	if user.TwoFactorEnabled {
		return &TwoFactorEnforcementResult{Required: false}, nil
	}

	// Get user roles
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	// Check policies for each role
	result := &TwoFactorEnforcementResult{}
	for _, roleName := range roles {
		policy, err := s.repo.GetTwoFactorPolicy(ctx, roleName)
		if err != nil {
			return nil, fmt.Errorf("get 2FA policy for role %s: %w", roleName, err)
		}
		if policy == nil || !policy.Enforced {
			continue
		}

		result.Required = true
		result.EnforcedByRoles = append(result.EnforcedByRoles, roleName)

		// Calculate grace deadline from user creation time
		deadline := user.CreatedAt.Add(time.Duration(policy.GracePeriodDays) * 24 * time.Hour)
		if result.GraceDeadline == nil || deadline.Before(*result.GraceDeadline) {
			result.GraceDeadline = &deadline
		}
	}

	if result.Required && result.GraceDeadline != nil {
		result.GraceExpired = time.Now().After(*result.GraceDeadline)
	}

	return result, nil
}

// GetTwoFactorPolicy returns the 2FA enforcement policy for a role.
func (s *Service) GetTwoFactorPolicy(ctx context.Context, roleName string) (*models.TwoFactorPolicy, error) {
	return s.repo.GetTwoFactorPolicy(ctx, roleName)
}

// UpdateTwoFactorPolicy creates or updates the 2FA enforcement policy for a role.
func (s *Service) UpdateTwoFactorPolicy(ctx context.Context, roleName string, enforced bool, gracePeriodDays int, updatedBy uuid.UUID) (*models.TwoFactorPolicy, error) {
	policy := &models.TwoFactorPolicy{
		ID:              uuid.New(),
		RoleName:        roleName,
		Enforced:        enforced,
		GracePeriodDays: gracePeriodDays,
		UpdatedAt:       time.Now(),
		UpdatedBy:       &updatedBy,
	}

	if err := s.repo.UpsertTwoFactorPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("upsert 2FA policy: %w", err)
	}

	slog.Info("2FA policy updated",
		"role_name", roleName,
		"enforced", enforced,
		"grace_period_days", gracePeriodDays,
		"updated_by", updatedBy,
	)
	return policy, nil
}

// --- Internal helpers ---

func (s *Service) encryptSecret(ctx context.Context, plainSecret string) (string, error) {
	if s.vaultSvc != nil {
		return s.vaultSvc.EncryptTOTPSecret(ctx, plainSecret)
	}
	// Development fallback: store as-is (NOT secure for production)
	return plainSecret, nil
}

func (s *Service) decryptSecret(ctx context.Context, encryptedSecret string) (string, error) {
	if s.vaultSvc != nil {
		return s.vaultSvc.DecryptTOTPSecret(ctx, encryptedSecret)
	}
	// Development fallback: return as-is
	return encryptedSecret, nil
}

func (s *Service) validateTOTPCode(ctx context.Context, user *models.User, code string) error {
	secret, err := s.decryptSecret(ctx, user.TwoFactorSecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt TOTP secret: %w", err)
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return ErrInvalidTOTPCode
	}
	return nil
}

func (s *Service) validateRecoveryCode(ctx context.Context, userID uuid.UUID, code string) error {
	codes, err := s.repo.GetRecoveryCodes(ctx, userID)
	if err != nil {
		return fmt.Errorf("get recovery codes: %w", err)
	}

	hash := hashRecoveryCode(code)

	// Check if all codes are used
	allUsed := true
	for _, c := range codes {
		if c.UsedAt == nil {
			allUsed = false
			break
		}
	}
	if len(codes) > 0 && allUsed {
		return ErrAllRecoveryCodesUsed
	}

	// Find matching unused code
	for _, c := range codes {
		if c.UsedAt != nil {
			continue
		}
		if c.CodeHash == hash {
			if err := s.repo.UseRecoveryCode(ctx, c.ID); err != nil {
				return fmt.Errorf("mark recovery code used: %w", err)
			}
			slog.Warn("recovery code used", "user_id", userID, "code_id", c.ID)
			return nil
		}
	}

	return ErrInvalidRecoveryCode
}

// generateRecoveryCodes creates a set of recovery codes and their hashed counterparts.
func generateRecoveryCodes(userID uuid.UUID) (plain []string, hashed []*models.RecoveryCode, err error) {
	now := time.Now()
	for i := 0; i < recoveryCodeCount; i++ {
		b := make([]byte, recoveryCodeLength/2) // 5 bytes = 10 hex chars
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		code := hex.EncodeToString(b)
		plain = append(plain, code)
		hashed = append(hashed, &models.RecoveryCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  hashRecoveryCode(code),
			CreatedAt: now,
		})
	}
	return plain, hashed, nil
}

// hashRecoveryCode hashes a recovery code with SHA-256.
func hashRecoveryCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
