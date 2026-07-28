package password

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service provides password policy validation and history checking.
type Service struct {
	repo Repository
}

// NewService creates a new password policy service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ValidatePassword validates a password against the current policy.
// Returns (valid, failure reasons, error). The failure reasons list is empty
// when the password passes all checks.
func (s *Service) ValidatePassword(ctx context.Context, password string) (bool, []string, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("password: tenant id missing from context: %w", err)
	}

	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		return false, nil, fmt.Errorf("password: failed to load policy: %w", err)
	}

	var failures []string

	// Check minimum length
	if len(password) < policy.MinLength {
		failures = append(failures, fmt.Sprintf("password must be at least %d characters", policy.MinLength))
	}

	// Check entropy (NIST SP 800-63B primary validation)
	entropy := passwordvalidator.GetEntropy(password)
	if entropy < policy.MinEntropy {
		failures = append(failures, fmt.Sprintf("password entropy %.1f bits is below minimum %.1f bits", entropy, policy.MinEntropy))
	}

	// Optional complexity requirements (not recommended by NIST, but configurable)
	if policy.RequireUppercase && !containsClass(password, unicode.IsUpper) {
		failures = append(failures, "password must contain at least one uppercase letter")
	}
	if policy.RequireLowercase && !containsClass(password, unicode.IsLower) {
		failures = append(failures, "password must contain at least one lowercase letter")
	}
	if policy.RequireDigit && !containsClass(password, unicode.IsDigit) {
		failures = append(failures, "password must contain at least one digit")
	}
	if policy.RequireSpecial && !containsSpecial(password) {
		failures = append(failures, "password must contain at least one special character")
	}

	valid := len(failures) == 0
	if !valid {
		slog.Info("password validation failed",
			"failure_count", len(failures),
			"reasons", strings.Join(failures, "; "),
		)
	}

	return valid, failures, nil
}

// CheckPasswordHistory checks whether the new password was previously used.
// Returns true if the password is NOT in history (i.e., safe to use).
func (s *Service) CheckPasswordHistory(ctx context.Context, userID uuid.UUID, newPassword string) (bool, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return false, fmt.Errorf("password: tenant id missing from context: %w", err)
	}

	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		return false, fmt.Errorf("password: failed to load policy: %w", err)
	}

	if policy.PreventReuseCount <= 0 {
		return true, nil
	}

	hashes, err := s.repo.GetPasswordHistory(ctx, userID, policy.PreventReuseCount)
	if err != nil {
		return false, fmt.Errorf("password: failed to load history: %w", err)
	}

	for _, hash := range hashes {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(newPassword)); err == nil {
			slog.Info("password reuse detected", "user_id", userID)
			return false, nil
		}
	}

	return true, nil
}

// RecordPassword stores a password hash in history for future reuse prevention.
func (s *Service) RecordPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if err := s.repo.AddPasswordHistory(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("password: failed to record history: %w", err)
	}
	slog.Info("password history recorded", "user_id", userID)
	return nil
}

// GetPolicy returns the current password policy for the caller's tenant.
func (s *Service) GetPolicy(ctx context.Context) (*models.PasswordPolicy, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("password: tenant id missing from context: %w", err)
	}
	return s.repo.GetPolicy(ctx, tenantID)
}

// UpdatePolicy updates the password policy for the caller's tenant. The row ID is
// always resolved server-side via a tenant-scoped GetPolicy -- any ID on the incoming
// policy is ignored, so a caller can never target another tenant's policy row.
func (s *Service) UpdatePolicy(ctx context.Context, policy *models.PasswordPolicy, updatedBy uuid.UUID) error {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return fmt.Errorf("password: tenant id missing from context: %w", err)
	}

	current, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("password: failed to load current policy: %w", err)
	}

	policy.ID = current.ID
	policy.TenantID = tenantID
	policy.UpdatedAt = time.Now().UTC()
	policy.UpdatedBy = &updatedBy

	if err := s.repo.UpdatePolicy(ctx, tenantID, policy); err != nil {
		return fmt.Errorf("password: failed to update policy: %w", err)
	}

	slog.Info("password policy updated",
		"updated_by", updatedBy,
		"min_length", policy.MinLength,
		"min_entropy", policy.MinEntropy,
		"prevent_reuse_count", policy.PreventReuseCount,
	)
	return nil
}

// containsClass checks if password contains at least one rune matching the classifier.
func containsClass(password string, classifier func(rune) bool) bool {
	for _, r := range password {
		if classifier(r) {
			return true
		}
	}
	return false
}

// containsSpecial checks if password contains at least one special character
// (not letter, not digit, not space).
func containsSpecial(password string) bool {
	for _, r := range password {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
