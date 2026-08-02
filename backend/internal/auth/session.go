package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// CreateSession creates a new user session with device metadata parsed from user agent.
func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, ipAddress, userAgent string, refreshTokenID uuid.UUID) (*models.UserSession, error) {
	deviceName, deviceType := parseUserAgent(userAgent)

	session := &models.UserSession{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		RefreshTokenID: &refreshTokenID,
		DeviceName:     deviceName,
		DeviceType:     deviceType,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		LastActiveAt:   time.Now(),
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	slog.Info("session created",
		"session_id", session.ID,
		"user_id", userID,
		"device_type", deviceType,
	)

	return session, nil
}

// ListSessions returns all sessions for a specific user.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	return s.repo.ListUserSessions(ctx, userID)
}

// ListAllSessions returns all sessions across all users with pagination (admin use).
func (s *Service) ListAllSessions(ctx context.Context, offset, limit int) ([]*models.UserSession, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAllSessions(ctx, offset, limit)
}

// TerminateSession deletes a session and revokes its associated refresh token.
//
// ownerID is the user the session must belong to. A session id is a bearer
// value in the URL of an authenticated route, so without this check any signed
// in user could sign out any other by guessing or replaying an id. Mismatch is
// reported as not-found rather than forbidden: whether a given id exists is
// not something a stranger should be able to learn.
func (s *Service) TerminateSession(ctx context.Context, sessionID, ownerID uuid.UUID) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UserID != ownerID {
		slog.Warn("rejected attempt to terminate another user's session",
			"session_id", sessionID,
			"session_user_id", session.UserID,
			"caller_user_id", ownerID,
		)
		return ErrSessionNotFound
	}

	// Revoke the associated refresh token
	if session.RefreshTokenID != nil {
		if revokeErr := s.repo.RevokeRefreshToken(ctx, *session.RefreshTokenID); revokeErr != nil {
			slog.Error("failed to revoke refresh token for terminated session",
				"session_id", sessionID,
				"refresh_token_id", session.RefreshTokenID,
				"error", revokeErr,
			)
		}
	}

	if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
		return err
	}

	slog.Info("session terminated",
		"session_id", sessionID,
		"user_id", session.UserID,
	)

	return nil
}

// TerminateAllSessions deletes all sessions for a user (optionally except the current one)
// and revokes all their refresh tokens.
func (s *Service) TerminateAllSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) error {
	if err := s.repo.DeleteAllUserSessions(ctx, userID, currentSessionID); err != nil {
		return err
	}

	// Revoke all refresh tokens for this user
	if revokeErr := s.repo.RevokeAllUserTokens(ctx, userID); revokeErr != nil {
		slog.Error("failed to revoke all tokens for user",
			"user_id", userID,
			"error", revokeErr,
		)
	}

	slog.Info("all sessions terminated",
		"user_id", userID,
		"except_session", currentSessionID,
	)

	return nil
}

// TouchSession updates the last_active_at timestamp for a session.
func (s *Service) TouchSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.UpdateSessionActivity(ctx, sessionID)
}

// parseUserAgent extracts device name and type from a user agent string.
func parseUserAgent(userAgent string) (deviceName, deviceType string) {
	ua := strings.ToLower(userAgent)

	// Detect device type
	switch {
	case strings.Contains(ua, "electron"):
		deviceType = "desktop"
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone"):
		deviceType = "mobile"
	case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
		deviceType = "tablet"
	default:
		deviceType = "browser"
	}

	// Detect device/browser name
	switch {
	case strings.Contains(ua, "electron"):
		deviceName = "KMU Hub Desktop"
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		deviceName = "Microsoft Edge"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg/"):
		deviceName = "Google Chrome"
	case strings.Contains(ua, "firefox"):
		deviceName = "Mozilla Firefox"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		deviceName = "Apple Safari"
	default:
		deviceName = "Unknown Browser"
	}

	// Append OS info
	switch {
	case strings.Contains(ua, "windows"):
		deviceName += " (Windows)"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		deviceName += " (macOS)"
	case strings.Contains(ua, "linux") && !strings.Contains(ua, "android"):
		deviceName += " (Linux)"
	case strings.Contains(ua, "android"):
		deviceName += " (Android)"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		deviceName += " (iOS)"
	}

	return deviceName, deviceType
}
