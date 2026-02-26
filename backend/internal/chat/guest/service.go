package guest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles guest session lifecycle management
type Service struct {
	repo        Repository
	rateLimiter *GuestRateLimiter
}

// NewService creates a new guest session service
func NewService(repo Repository) *Service {
	return &Service{
		repo:        repo,
		rateLimiter: NewGuestRateLimiter(30, time.Minute),
	}
}

// CreateSession generates a new guest session with an opaque token
func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (*TokenResult, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if len(displayName) > 200 {
		return nil, ErrDisplayNameTooLong
	}

	// Determine token expiry from channel config (or default 168h = 7 days)
	expiryHours := 168
	config, err := s.repo.GetConfigByChannel(ctx, input.ChannelID)
	if err == nil && config != nil {
		expiryHours = config.TokenExpiryHours
	}

	// Generate opaque token and hash for storage
	token := uuid.New().String()
	tokenHash := hashToken(token)

	now := time.Now()
	session := &GuestSession{
		ID:             uuid.New(),
		TokenHash:      tokenHash,
		ChannelID:      input.ChannelID,
		DisplayName:    displayName,
		Email:          input.Email,
		IPAddress:      input.IPAddress,
		UserAgent:      input.UserAgent,
		LastActivityAt: now,
		IsActive:       true,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Duration(expiryHours) * time.Hour),
	}

	if createErr := s.repo.CreateSession(ctx, session); createErr != nil {
		return nil, createErr
	}

	slog.Info("guest session created",
		"session_id", session.ID,
		"channel_id", session.ChannelID,
		"display_name", session.DisplayName,
	)

	return &TokenResult{
		Token:   token,
		Session: session,
	}, nil
}

// ValidateToken validates an opaque guest token and returns the session
func (s *Service) ValidateToken(ctx context.Context, token string) (*GuestSession, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(token)
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !session.IsActive {
		return nil, ErrSessionInactive
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	// Update last activity (best-effort)
	if updateErr := s.repo.UpdateLastActivity(ctx, session.ID); updateErr != nil {
		slog.Warn("failed to update guest last activity",
			"session_id", session.ID,
			"error", updateErr,
		)
	}

	return session, nil
}

// DeactivateSession marks a guest session as inactive
func (s *Service) DeactivateSession(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.repo.DeactivateSession(ctx, sessionID); err != nil {
		return err
	}

	slog.Info("guest session deactivated", "session_id", sessionID)
	return nil
}

// CleanupExpired deactivates all expired guest sessions
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	count, err := s.repo.CleanupExpiredSessions(ctx)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		slog.Info("expired guest sessions cleaned up", "count", count)
	}
	return count, nil
}

// CreateConfig creates guest channel configuration with defaults for nil fields
func (s *Service) CreateConfig(ctx context.Context, input CreateConfigInput) (*GuestChannelConfig, error) {
	now := time.Now()
	config := &GuestChannelConfig{
		ID:               uuid.New(),
		ChannelID:        input.ChannelID,
		WelcomeMessage:   "Willkommen! Wie können wir Ihnen helfen?",
		PrimaryColor:     "#3b82f6",
		TokenExpiryHours: 168,
		MaxFileSizeMB:    10,
		AllowedFileTypes: "image/png,image/jpeg,image/gif,image/webp,application/pdf",
		IsActive:         true,
		CreatedBy:        input.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if input.WelcomeMessage != nil {
		config.WelcomeMessage = *input.WelcomeMessage
	}
	if input.LogoURL != nil {
		config.LogoURL = input.LogoURL
	}
	if input.PrimaryColor != nil {
		config.PrimaryColor = *input.PrimaryColor
	}
	if input.TokenExpiryHours != nil {
		config.TokenExpiryHours = *input.TokenExpiryHours
	}
	if input.MaxFileSizeMB != nil {
		config.MaxFileSizeMB = *input.MaxFileSizeMB
	}
	if input.AllowedFileTypes != nil {
		config.AllowedFileTypes = *input.AllowedFileTypes
	}

	if err := s.repo.CreateConfig(ctx, config); err != nil {
		return nil, err
	}

	slog.Info("guest channel config created",
		"channel_id", config.ChannelID,
		"created_by", config.CreatedBy,
	)

	return config, nil
}

// GetConfig returns the guest channel configuration
func (s *Service) GetConfig(ctx context.Context, channelID uuid.UUID) (*GuestChannelConfig, error) {
	return s.repo.GetConfigByChannel(ctx, channelID)
}

// UpdateConfig updates an existing guest channel configuration
func (s *Service) UpdateConfig(ctx context.Context, config *GuestChannelConfig) error {
	config.UpdatedAt = time.Now()
	return s.repo.UpdateConfig(ctx, config)
}

// CheckRateLimit checks if the guest session is within its message rate limit
func (s *Service) CheckRateLimit(sessionID uuid.UUID) error {
	if !s.rateLimiter.Allow(sessionID.String()) {
		return ErrRateLimited
	}
	return nil
}

// hashToken returns the SHA-256 hex digest of a token string
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
