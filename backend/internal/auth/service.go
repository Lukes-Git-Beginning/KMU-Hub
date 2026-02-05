package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
)

const bcryptCost = 12

type Service struct {
	repo       Repository
	tokenMaker *TokenMaker
}

func NewService(repo Repository, tokenMaker *TokenMaker) *Service {
	return &Service{
		repo:       repo,
		tokenMaker: tokenMaker,
	}
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName string) (*models.User, *models.TokenPair, error) {
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

func (s *Service) Login(ctx context.Context, email, password string) (*models.User, *models.TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email)
	return user, tokens, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
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

	accessToken, err := s.tokenMaker.CreateAccessToken(user.ID, roles, permissions)
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	// Revoke all refresh tokens (force re-login on all devices)
	_ = s.repo.RevokeAllUserTokens(ctx, userID)

	slog.Info("password changed", "user_id", userID)
	return nil
}

const invitationExpiry = 7 * 24 * time.Hour // 7 days

// CreateInvitation creates a new user invitation
func (s *Service) CreateInvitation(ctx context.Context, email, role string, createdBy uuid.UUID) (*models.Invitation, string, error) {
	// Check if email already exists as a user
	if existing, _ := s.repo.GetUserByEmail(ctx, email); existing != nil {
		return nil, "", ErrUserExists
	}

	// Generate secure token
	token := generateSecureToken()
	hash := HashToken(token)

	inv := &models.Invitation{
		ID:        uuid.New(),
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

	// Create user with pre-assigned role
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        inv.Email,
		PasswordHash: string(pwHash),
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, nil, err
	}

	// Assign the role from invitation
	if err := s.repo.AssignRole(ctx, user.ID, inv.Role); err != nil {
		slog.Error("failed to assign role from invitation", "user_id", user.ID, "role", inv.Role, "error", err)
	}

	// Mark invitation as accepted
	if err := s.repo.MarkInvitationAccepted(ctx, inv.ID); err != nil {
		slog.Error("failed to mark invitation accepted", "invitation_id", inv.ID, "error", err)
	}

	// Generate tokens
	tokens, err := s.createTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("invitation accepted", "user_id", user.ID, "invitation_id", inv.ID)
	return user, tokens, nil
}

// ListInvitations returns all pending invitations
func (s *Service) ListInvitations(ctx context.Context) ([]*models.Invitation, error) {
	return s.repo.ListPendingInvitations(ctx)
}

// CancelInvitation deletes a pending invitation
func (s *Service) CancelInvitation(ctx context.Context, invitationID uuid.UUID) error {
	inv, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.AcceptedAt != nil {
		return ErrInvitationAlreadyUsed
	}

	return s.repo.DeleteInvitation(ctx, invitationID)
}

func generateSecureToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
