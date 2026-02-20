package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AccountLinkService handles the token-based account linking flow.
// Users run /kmuhub link in Teams/Slack to get a token, then paste it in KMU Hub.
type AccountLinkService struct {
	repo Repository
}

// NewAccountLinkService creates a new account link service.
func NewAccountLinkService(repo Repository) *AccountLinkService {
	return &AccountLinkService{repo: repo}
}

// GenerateLinkToken creates a short-lived token for account linking.
// The raw token is returned to the user; only the SHA-256 hash is stored.
func (s *AccountLinkService) GenerateLinkToken(ctx context.Context, platform, externalUserID string) (string, error) {
	if !ValidPlatforms[platform] {
		return "", ErrInvalidPlatform
	}

	// Generate 32-byte random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)

	// SHA-256 hash for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Invalidate any existing unused tokens for same platform+external_user_id
	// by cleaning up expired tokens (includes old unused ones for this user)
	if _, err := s.repo.CleanupExpiredTokens(ctx); err != nil {
		// Non-fatal: continue even if cleanup fails
		_ = err
	}

	// Store link token with 5-minute expiry
	token := &LinkToken{
		ID:             uuid.New(),
		Platform:       platform,
		ExternalUserID: externalUserID,
		TokenHash:      tokenHash,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		Used:           false,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateLinkToken(ctx, token); err != nil {
		return "", fmt.Errorf("store link token: %w", err)
	}

	return rawToken, nil
}

// VerifyAndLink verifies a raw token and creates/updates the account link.
func (s *AccountLinkService) VerifyAndLink(ctx context.Context, rawToken string, kmuhubUserID uuid.UUID) (*AccountLink, error) {
	// SHA-256 hash the raw token for lookup
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Look up token by hash
	token, err := s.repo.GetLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrLinkTokenNotFound
	}

	// Check if already used
	if token.Used {
		return nil, ErrLinkTokenUsed
	}

	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrLinkTokenExpired
	}

	// Mark token as used
	if err := s.repo.MarkLinkTokenUsed(ctx, token.ID); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Create/upsert account link
	link := &AccountLink{
		ID:             uuid.New(),
		Platform:       token.Platform,
		ExternalUserID: token.ExternalUserID,
		KMUHubUserID:   kmuhubUserID,
		LinkedAt:       time.Now(),
	}

	if err := s.repo.CreateAccountLink(ctx, link); err != nil {
		return nil, fmt.Errorf("create account link: %w", err)
	}

	return link, nil
}

// GetLinkStatus checks if a user has a linked account for a platform.
func (s *AccountLinkService) GetLinkStatus(ctx context.Context, platform string, kmuhubUserID uuid.UUID) (*AccountLink, error) {
	if !ValidPlatforms[platform] {
		return nil, ErrInvalidPlatform
	}
	return s.repo.GetAccountLinkByKMUHubUser(ctx, platform, kmuhubUserID)
}

// Unlink removes an account link for a platform.
func (s *AccountLinkService) Unlink(ctx context.Context, platform string, kmuhubUserID uuid.UUID) error {
	if !ValidPlatforms[platform] {
		return ErrInvalidPlatform
	}

	link, err := s.repo.GetAccountLinkByKMUHubUser(ctx, platform, kmuhubUserID)
	if err != nil {
		return ErrAccountLinkNotFound
	}

	return s.repo.DeleteAccountLink(ctx, link.ID)
}
