package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

const bcryptCost = 12

// passwordResetExpiry is how long a reset link remains valid.
const passwordResetExpiry = 1 * time.Hour

// PasswordValidator is the narrow interface the service uses to check
// password strength. The concrete implementation wraps
// security/password.Service.
type PasswordValidator interface {
	ValidatePassword(ctx context.Context, password string) (valid bool, reasons []string, err error)
}

type Service struct {
	repo              Repository
	tokenMaker        *TokenMaker
	vaultSvc          VaultEncryptor
	mailer            Mailer
	passwordValidator PasswordValidator
	// resetBaseURL is the frontend URL prefix for reset links,
	// e.g. "https://app.zentria.tech/reset-password". The plain token
	// is appended as a query parameter: ?token=<plaintext>.
	resetBaseURL string
}

func NewService(repo Repository, tokenMaker *TokenMaker) *Service {
	return &Service{
		repo:       repo,
		tokenMaker: tokenMaker,
	}
}

// SetMailer injects the transactional mailer (called from cmd/auth/main.go).
func (s *Service) SetMailer(m Mailer) {
	s.mailer = m
}

// SetPasswordValidator injects the password-strength validator.
func (s *Service) SetPasswordValidator(v PasswordValidator) {
	s.passwordValidator = v
}

// SetResetBaseURL sets the frontend base URL for password-reset links.
func (s *Service) SetResetBaseURL(url string) {
	s.resetBaseURL = url
}

// normalizeEmail lowercases and trims an email so lookups and uniqueness are
// case-insensitive. Without this, "User@x" and "user@x" become two distinct
// accounts (a real prod bug) and a case-mismatched password reset silently
// finds no user. Applied at every service entrypoint that reads or writes an
// email; the DB unique index (migration 000148) is on lower(email) to match.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName string) (*models.User, *models.TokenPair, error) {
	email = normalizeEmail(email)
	// Pre-JWT path: no tenant context in caller ctx. RLS policies on users
	// would reject the lookup without system bypass.
	ctx = sysctx.With(ctx)

	existing, _ := s.repo.GetUserByEmail(ctx, email)
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		TenantID:     models.DefaultTenantID,
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if createErr := s.repo.CreateUser(ctx, user); createErr != nil {
		return nil, nil, createErr
	}

	if roleErr := s.repo.AssignRole(ctx, user.ID, "member"); roleErr != nil {
		slog.Error("failed to assign default role", "user_id", user.ID, "error", roleErr)
	}

	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email)
	return user, tokens, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*models.LoginResult, error) {
	email = normalizeEmail(email)
	// Pre-JWT path: caller has no tenant in ctx. RLS would otherwise
	// reject the user lookup.
	ctx = sysctx.With(ctx)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, ErrInvalidCredentials
	}

	// If 2FA is enabled, issue a pending token instead of full tokens
	if user.TwoFactorEnabled {
		pendingToken, err := s.tokenMaker.CreatePendingToken(user.ID)
		if err != nil {
			return nil, fmt.Errorf("create pending token: %w", err)
		}
		slog.Info("login requires 2FA", "user_id", user.ID, "email", user.Email)
		return &models.LoginResult{
			RequiresTwoFactor: true,
			PendingToken:      pendingToken,
		}, nil
	}

	// Check 2FA enforcement: if required and grace expired, block login
	enforcement, err := s.Check2FAEnforcement(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("check 2FA enforcement: %w", err)
	}
	if enforcement != nil && enforcement.Required && enforcement.GraceExpired {
		return nil, Err2FAEnforcementRequired
	}

	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email)
	return &models.LoginResult{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

// CompleteTwoFactorLogin validates the 2FA code after credential verification and issues full tokens.
func (s *Service) CompleteTwoFactorLogin(ctx context.Context, pendingToken, code string, isRecoveryCode bool) (*models.User, *models.TokenPair, error) {
	// Pre-JWT path: pending token is not a JWT. The DB reads below
	// (Validate2FALogin → recovery_codes; GetUserByID) need RLS bypass.
	ctx = sysctx.With(ctx)

	// Validate the pending token
	userID, err := s.tokenMaker.ValidatePendingToken(pendingToken)
	if err != nil {
		return nil, nil, err
	}

	// Validate the 2FA code
	recoveryUsed, err := s.Validate2FALogin(ctx, userID, code, isRecoveryCode)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	if recoveryUsed {
		slog.Warn("2FA login completed with recovery code", "user_id", userID)
	} else {
		slog.Info("2FA login completed", "user_id", userID)
	}

	return user, tokens, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	// Pre-JWT path: the access token has expired by definition, no tenant
	// in ctx. RLS bypass needed for the user_sessions + users lookups.
	ctx = sysctx.With(ctx)

	hash := HashToken(refreshToken)

	stored, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	if stored.Revoked {
		// Possible token theft: revoke all tokens for this user
		_ = s.repo.RevokeAllUserTokens(ctx, stored.UserID)
		slog.Warn("revoked refresh token reuse detected, revoking all tokens", "user_id", stored.UserID)
		return nil, ErrTokenRevoked
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Rotate: revoke old token
	if revokeErr := s.repo.RevokeRefreshToken(ctx, stored.ID); revokeErr != nil {
		return nil, revokeErr
	}

	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	slog.Info("token refreshed", "user_id", user.ID)
	return tokens, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	// Logout takes the same opaque refresh token as RefreshToken, so the
	// user_sessions lookup is unauthenticated by the same logic.
	ctx = sysctx.With(ctx)

	hash := HashToken(refreshToken)

	stored, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		// Token not found: logout is idempotent
		return nil
	}

	if err := s.repo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		return err
	}

	slog.Info("user logged out", "user_id", stored.UserID)
	return nil
}

func (s *Service) ValidateToken(ctx context.Context, accessToken string) (*Claims, error) {
	claims, err := s.tokenMaker.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, []string, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	roles, _ := s.repo.GetUserRoles(ctx, userID)
	return user, roles, nil
}

func (s *Service) ListUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.ListUsers(ctx, offset, pageSize)
}

func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, firstName, lastName *string, isActive *bool) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if firstName != nil {
		user.FirstName = *firstName
	}
	if lastName != nil {
		user.LastName = *lastName
	}
	if isActive != nil {
		user.IsActive = *isActive
	}
	user.UpdatedAt = time.Now()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	slog.Info("user updated", "user_id", userID)
	return user, nil
}

// UpdateProfile applies the self-service profile fields for the authenticated
// user. avatarURL holds the presigned-upload object key. Cannot change is_active.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, avatarURL *string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if firstName != nil {
		user.FirstName = *firstName
	}
	if lastName != nil {
		user.LastName = *lastName
	}
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	user.UpdatedAt = time.Now()

	if err := s.repo.UpdateProfile(ctx, user); err != nil {
		return nil, err
	}

	slog.Info("profile updated", "user_id", userID)
	return user, nil
}

func (s *Service) AssignRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return ErrUserNotFound
	}
	return s.repo.AssignRole(ctx, userID, roleName)
}

func (s *Service) RemoveRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return ErrUserNotFound
	}
	return s.repo.RemoveRole(ctx, userID, roleName)
}

func (s *Service) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	return s.repo.UserHasPermission(ctx, userID, resource, action)
}

func (s *Service) createTokenPair(ctx context.Context, user *models.User) (*models.TokenPair, error) {
	roles, _ := s.repo.GetUserRoles(ctx, user.ID)
	permissions, _ := s.repo.GetUserPermissions(ctx, user.ID)

	accessToken, err := s.tokenMaker.CreateAccessToken(user.ID, user.TenantID.String(), roles, permissions)
	if err != nil {
		return nil, err
	}

	plain, hash, expiresAt := s.tokenMaker.CreateRefreshToken()

	refreshToken := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.repo.StoreRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: plain,
	}, nil
}

// GetProfile returns the current user's profile (same as GetUser, convenience method)
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, []string, error) {
	return s.GetUser(ctx, userID)
}

// ChangePassword verifies old password and updates to new, then revokes all tokens
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); compareErr != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}

	if updateErr := s.repo.UpdatePassword(ctx, userID, string(hash)); updateErr != nil {
		return updateErr
	}

	// Revoke all refresh tokens (force re-login on all devices)
	_ = s.repo.RevokeAllUserTokens(ctx, userID)

	slog.Info("password changed", "user_id", userID)
	return nil
}

const invitationExpiry = 7 * 24 * time.Hour // 7 days

// CreateInvitation creates a new user invitation for the given tenant. The
// tenant is passed in rather than read from ctx: internal/auth cannot import
// internal/middleware (middleware → auth), so the gRPC layer resolves it.
func (s *Service) CreateInvitation(ctx context.Context, tenantID uuid.UUID, email, role string, createdBy uuid.UUID) (*models.Invitation, string, error) {
	email = normalizeEmail(email)
	// Check if email already exists as a user
	if existing, _ := s.repo.GetUserByEmail(ctx, email); existing != nil {
		return nil, "", ErrUserExists
	}

	// A pending invitation holds a seat, so the limit has to be checked here
	// and not only on acceptance — otherwise a tenant can invite past its plan
	// and the overrun only surfaces days later, when people cannot sign up.
	if err := s.assertSeatAvailable(ctx, tenantID, nil); err != nil {
		return nil, "", err
	}

	// Generate secure token
	token := generateSecureToken()
	hash := HashToken(token)

	inv := &models.Invitation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		TokenHash: hash,
		CreatedBy: createdBy,
		ExpiresAt: time.Now().Add(invitationExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateInvitation(ctx, inv); err != nil {
		return nil, "", err
	}

	slog.Info("invitation created", "email", email, "role", role, "created_by", createdBy)
	return inv, token, nil
}

// AcceptInvitation creates a user from an invitation
func (s *Service) AcceptInvitation(ctx context.Context, token, password, firstName, lastName string) (*models.User, *models.TokenPair, error) {
	// Pre-JWT path: invitation token is opaque, no tenant context in ctx.
	// invitations + users + user_roles lookups all need RLS bypass.
	ctx = sysctx.With(ctx)

	hash := HashToken(token)

	inv, err := s.repo.GetInvitationByToken(ctx, hash)
	if err != nil {
		return nil, nil, ErrInvitationNotFound
	}

	if inv.AcceptedAt != nil {
		return nil, nil, ErrInvitationAlreadyUsed
	}

	if time.Now().After(inv.ExpiresAt) {
		return nil, nil, ErrInvitationExpired
	}

	// The seat this invitation reserved may have been withdrawn in the meantime
	// — a plan downgrade shrinks the limit without touching open invitations.
	// The invitation itself is excluded from the count; it is about to stop
	// being pending and become the user it invited.
	if err := s.assertSeatAvailable(ctx, inv.TenantID, &inv.ID); err != nil {
		return nil, nil, err
	}

	// Create user with pre-assigned role
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, err
	}

	user := &models.User{
		ID: uuid.New(),
		// The invitation carries the tenant — the account belongs to whoever
		// invited it, not to the default tenant.
		TenantID:     inv.TenantID,
		Email:        inv.Email,
		PasswordHash: string(pwHash),
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// One transaction: claim the invitation, create the account, assign the
	// role. The claim is what makes the token single-use — the earlier
	// AcceptedAt check above is only there to answer a replay cheaply.
	if err := s.repo.AcceptInvitation(ctx, inv, user); err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("invitation accepted", "user_id", user.ID, "invitation_id", inv.ID, "tenant_id", inv.TenantID)
	return user, tokens, nil
}

// ListInvitations returns the tenant's pending invitations.
func (s *Service) ListInvitations(ctx context.Context, tenantID uuid.UUID) ([]*models.Invitation, error) {
	return s.repo.ListPendingInvitations(ctx, tenantID)
}

// CancelInvitation deletes a pending invitation of the given tenant.
func (s *Service) CancelInvitation(ctx context.Context, tenantID, invitationID uuid.UUID) error {
	inv, err := s.repo.GetInvitationByID(ctx, tenantID, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.AcceptedAt != nil {
		return ErrInvitationAlreadyUsed
	}

	return s.repo.DeleteInvitation(ctx, tenantID, invitationID)
}

// assertSeatAvailable returns ErrSeatLimitReached when the tenant has no free
// seat. A nil limit means the tenant has no cap booked, which is the state
// every tenant is in until the licence service writes one.
func (s *Service) assertSeatAvailable(ctx context.Context, tenantID uuid.UUID, excludeInvitation *uuid.UUID) error {
	used, limit, err := s.repo.CountSeatsInUse(ctx, tenantID, excludeInvitation)
	if err != nil {
		return err
	}
	if limit == nil {
		return nil
	}
	if used >= *limit {
		slog.Warn("seat limit reached", "tenant_id", tenantID, "used", used, "limit", *limit)
		return ErrSeatLimitReached
	}
	return nil
}

func generateSecureToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ============================================================================
// Password Reset
// ============================================================================

// RequestPasswordReset initiates a password-reset flow for the given email.
// To prevent user enumeration the method always returns nil — callers must
// respond with the same HTTP 200 body whether or not the email is known.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	// Pre-JWT path: no tenant in ctx, need RLS bypass to read users by email.
	sysCtx := sysctx.With(ctx)

	user, err := s.repo.GetUserByEmail(sysCtx, email)
	if err != nil {
		// Unknown email — silently succeed (no enumeration leak).
		slog.Info("password reset requested for unknown email", "email", email)
		return nil
	}

	plain := generateSecureToken()
	hash := HashToken(plain)

	prt := &models.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(passwordResetExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreatePasswordResetToken(sysCtx, prt); err != nil {
		slog.Error("failed to create password reset token", "user_id", user.ID, "error", err)
		// Still return nil to avoid timing-based enumeration.
		return nil
	}

	if s.mailer != nil {
		baseURL := s.resetBaseURL
		if baseURL == "" {
			baseURL = "https://app.zentria.tech/reset-password"
		}
		resetLink := fmt.Sprintf("%s?token=%s", baseURL, plain)
		name := user.FirstName
		if name == "" {
			name = user.Email
		}
		if mailErr := s.mailer.SendPasswordResetEmail(sysCtx, user.Email, name, resetLink); mailErr != nil {
			slog.Error("failed to send password reset email", "user_id", user.ID, "error", mailErr)
		}
	} else {
		slog.Warn("mailer not configured, password reset email not sent", "user_id", user.ID)
	}

	slog.Info("password reset token issued", "user_id", user.ID)
	return nil
}

// ResetPassword validates the reset token, checks password strength, updates
// the password, marks the token as used, and revokes all refresh tokens.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Pre-JWT path: no tenant in ctx.
	sysCtx := sysctx.With(ctx)

	hash := HashToken(token)

	prt, err := s.repo.GetPasswordResetToken(sysCtx, hash)
	if err != nil {
		return ErrResetTokenInvalid
	}

	if prt.UsedAt != nil {
		return ErrResetTokenUsed
	}

	if time.Now().After(prt.ExpiresAt) {
		return ErrResetTokenExpired
	}

	// Strength check via injected validator (if wired).
	if s.passwordValidator != nil {
		valid, reasons, valErr := s.passwordValidator.ValidatePassword(sysCtx, newPassword)
		if valErr != nil {
			slog.Warn("password validator error during reset, skipping strength check", "error", valErr)
		} else if !valid {
			slog.Info("password reset rejected: weak password", "reasons", reasons)
			return ErrPasswordTooWeak
		}
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(sysCtx, prt.UserID, string(pwHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Mark token as consumed (idempotent on double-call).
	if err := s.repo.MarkPasswordResetTokenUsed(sysCtx, prt.ID); err != nil {
		slog.Error("failed to mark reset token used", "token_id", prt.ID, "error", err)
	}

	// Revoke all active refresh tokens so old sessions are invalidated.
	if err := s.repo.RevokeAllUserTokens(sysCtx, prt.UserID); err != nil {
		slog.Error("failed to revoke tokens after password reset", "user_id", prt.UserID, "error", err)
	}

	slog.Info("password reset completed", "user_id", prt.UserID)
	return nil
}
