package caldav

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	// bcryptCost is the cost parameter for bcrypt hashing of app-specific passwords.
	bcryptCost = 12

	// passwordBytes is the number of random bytes for generated passwords (hex = 2x chars).
	passwordBytes = 32

	// passwordPrefixLen is the number of characters to store as prefix for UI identification.
	passwordPrefixLen = 4

	// settingCalDAVEnabled is the caldav_settings key for the org-wide toggle.
	settingCalDAVEnabled = "caldav_carddav_enabled"
)

// AppPasswordService manages app-specific passwords for CalDAV/CardDAV access.
type AppPasswordService struct {
	repo AppPasswordRepository
	pool *pgxpool.Pool
}

// NewAppPasswordService creates a new app-specific password service.
func NewAppPasswordService(repo AppPasswordRepository, pool *pgxpool.Pool) *AppPasswordService {
	return &AppPasswordService{
		repo: repo,
		pool: pool,
	}
}

// Create generates a new app-specific password for the given user.
// Returns the plaintext password (shown once) and the persisted record.
func (s *AppPasswordService) Create(ctx context.Context, userID uuid.UUID, label string) (string, *models.AppSpecificPassword, error) {
	// Generate 32-byte random password (64 hex chars)
	raw := make([]byte, passwordBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	plaintext := hex.EncodeToString(raw)

	// Hash with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return "", nil, err
	}

	pw := &models.AppSpecificPassword{
		ID:             uuid.New(),
		UserID:         userID,
		Label:          label,
		PasswordHash:   string(hash),
		PasswordPrefix: plaintext[:passwordPrefixLen],
		Scope:          models.AppPasswordScopeCalDAV,
	}

	if err := s.repo.Create(ctx, pw); err != nil {
		return "", nil, err
	}

	slog.Info("app-specific password created",
		"user_id", userID,
		"password_id", pw.ID,
		"label", label,
	)

	return plaintext, pw, nil
}

// Validate checks a username (user UUID string) and plaintext password against
// stored app-specific passwords. Returns the user ID on success.
// Checks the org-level toggle first.
func (s *AppPasswordService) Validate(ctx context.Context, username, password string) (uuid.UUID, error) {
	// Check org toggle
	enabled, err := s.IsOrgEnabled(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if !enabled {
		return uuid.Nil, ErrCalDAVDisabled
	}

	// Parse username as user UUID
	userID, err := uuid.Parse(username)
	if err != nil {
		slog.Warn("caldav auth: invalid username format",
			"username", username,
		)
		return uuid.Nil, ErrInvalidCredentials
	}

	// Retrieve active passwords for this user
	passwords, err := s.repo.FindActiveByUser(ctx, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if len(passwords) == 0 {
		return uuid.Nil, ErrInvalidCredentials
	}

	// Compare against each active password
	for _, pw := range passwords {
		if bcrypt.CompareHashAndPassword([]byte(pw.PasswordHash), []byte(password)) == nil {
			// Match found -- update last_used_at in background
			if updateErr := s.repo.UpdateLastUsed(ctx, pw.ID); updateErr != nil {
				slog.Warn("failed to update last_used_at for app password",
					"password_id", pw.ID,
					"error", updateErr,
				)
			}

			slog.Info("caldav auth: password validated",
				"user_id", userID,
				"password_id", pw.ID,
			)
			return userID, nil
		}
	}

	return uuid.Nil, ErrInvalidCredentials
}

// List returns all app-specific passwords (active + revoked) for display.
func (s *AppPasswordService) List(ctx context.Context, userID uuid.UUID) ([]*models.AppSpecificPassword, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Revoke revokes a specific app-specific password.
func (s *AppPasswordService) Revoke(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.repo.Revoke(ctx, id, userID); err != nil {
		return err
	}

	slog.Info("app-specific password revoked",
		"password_id", id,
		"user_id", userID,
	)
	return nil
}

// IsOrgEnabled queries the caldav_settings table for the org-wide toggle.
func (s *AppPasswordService) IsOrgEnabled(ctx context.Context) (bool, error) {
	var value string
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM caldav_settings WHERE key = $1`,
		settingCalDAVEnabled,
	).Scan(&value)
	if err != nil {
		// If row not found, default to disabled
		return false, nil
	}
	return value == "true", nil
}

// SetOrgEnabled upserts the org-wide CalDAV/CardDAV toggle.
func (s *AppPasswordService) SetOrgEnabled(ctx context.Context, enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO caldav_settings (key, value, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		settingCalDAVEnabled, val,
	)
	if err != nil {
		return err
	}

	slog.Info("caldav org toggle updated",
		"enabled", enabled,
	)
	return nil
}
