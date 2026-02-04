package auth

import (
	"context"
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

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, nil, err
	}

	if err := s.repo.AssignRole(ctx, user.ID, "member"); err != nil {
		slog.Error("failed to assign default role", "user_id", user.ID, "error", err)
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
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
	if err := s.repo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		return nil, err
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
