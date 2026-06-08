package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Test helpers
// ============================================================================

// mockMailer records SendPasswordResetEmail calls for assertion.
type mockMailer struct {
	called    bool
	lastEmail string
	lastLink  string
	err       error
}

func (m *mockMailer) SendPasswordResetEmail(_ context.Context, toEmail, _ string, resetLink string) error {
	m.called = true
	m.lastEmail = toEmail
	m.lastLink = resetLink
	return m.err
}

// mockPasswordValidator always passes unless configured to fail.
type mockPasswordValidator struct {
	valid   bool
	reasons []string
	err     error
}

func (v *mockPasswordValidator) ValidatePassword(_ context.Context, _ string) (bool, []string, error) {
	return v.valid, v.reasons, v.err
}

func newResetTestService() (*Service, *mockRepository, *mockMailer) {
	repo := newMockRepository()
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	svc := NewService(repo, tm)
	m := &mockMailer{}
	svc.SetMailer(m)
	svc.SetResetBaseURL("https://app.test.local/reset-password")
	svc.SetPasswordValidator(&mockPasswordValidator{valid: true})
	return svc, repo, m
}

func seedResetUser(repo *mockRepository) *models.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("OldPass1!"), bcryptCost)
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     models.DefaultTenantID,
		Email:        "reset-user@example.com",
		PasswordHash: string(hash),
		FirstName:    "Reset",
		LastName:     "User",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[user.ID] = user
	repo.usersByEmail[user.Email] = user
	return user
}

// issueTokenForUser creates a PasswordResetToken in the mock repo and returns
// the plaintext token.
func issueTokenForUser(t *testing.T, svc *Service, user *models.User) string {
	t.Helper()
	// Trigger via RequestPasswordReset; the mailer captures the link.
	m := svc.mailer.(*mockMailer)
	require.NoError(t, svc.RequestPasswordReset(context.Background(), user.Email))
	require.True(t, m.called, "expected mailer to be called")

	// Extract plain token from the link (format: baseURL?token=<plain>)
	link := m.lastLink
	const pfx = "?token="
	idx := len(link) - len(link[len(pfx):])
	_ = idx
	// Parse token from link
	plainToken := link[len("https://app.test.local/reset-password?token="):]
	return plainToken
}

// ============================================================================
// RequestPasswordReset
// ============================================================================

func TestRequestPasswordReset_KnownEmail_SendsMail(t *testing.T) {
	svc, repo, m := newResetTestService()
	user := seedResetUser(repo)

	err := svc.RequestPasswordReset(context.Background(), user.Email)
	require.NoError(t, err, "should never return error")
	assert.True(t, m.called, "mailer should have been called")
	assert.Equal(t, user.Email, m.lastEmail)
	assert.NotEmpty(t, m.lastLink)
	assert.Contains(t, m.lastLink, "?token=")
}

func TestRequestPasswordReset_UnknownEmail_NoEnumerationLeak(t *testing.T) {
	svc, _, m := newResetTestService()

	// Unknown email — must succeed silently and NOT call the mailer.
	err := svc.RequestPasswordReset(context.Background(), "nobody@example.com")
	require.NoError(t, err, "must return nil regardless of whether user exists")
	assert.False(t, m.called, "mailer must NOT be called for unknown email")
}

func TestRequestPasswordReset_TokenStoredInRepo(t *testing.T) {
	svc, repo, m := newResetTestService()
	user := seedResetUser(repo)

	require.NoError(t, svc.RequestPasswordReset(context.Background(), user.Email))
	require.True(t, m.called)

	// There should be exactly one token in the repo.
	assert.Equal(t, 1, len(repo.passwordResetTokens))
	for _, tok := range repo.passwordResetTokens {
		assert.Equal(t, user.ID, tok.UserID)
		assert.Equal(t, user.TenantID, tok.TenantID)
		assert.True(t, tok.ExpiresAt.After(time.Now()), "token must not be expired on issue")
		assert.Nil(t, tok.UsedAt, "token must not be used on issue")
	}
}

// ============================================================================
// ResetPassword — happy path
// ============================================================================

func TestResetPassword_ValidToken_UpdatesPassword(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)
	plainToken := issueTokenForUser(t, svc, user)

	newPw := "NewStrongPass99!"
	err := svc.ResetPassword(context.Background(), plainToken, newPw)
	require.NoError(t, err)

	// Password in repo must be updated.
	updated := repo.users[user.ID]
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte(newPw))
	assert.NoError(t, err, "new password should match bcrypt hash")
}

func TestResetPassword_TokenMarkedUsed(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)
	plainToken := issueTokenForUser(t, svc, user)

	require.NoError(t, svc.ResetPassword(context.Background(), plainToken, "NewStrongPass99!"))

	hash := HashToken(plainToken)
	stored := repo.passwordResetTokens[hash]
	require.NotNil(t, stored)
	assert.NotNil(t, stored.UsedAt, "used_at must be set after reset")
}

func TestResetPassword_RevokeAllTokensAfterReset(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)

	// Store a refresh token for the user.
	plain, hash, exp := svc.tokenMaker.CreateRefreshToken()
	_ = plain
	repo.refreshTokens[hash] = &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: exp,
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	plainToken := issueTokenForUser(t, svc, user)
	require.NoError(t, svc.ResetPassword(context.Background(), plainToken, "NewStrongPass99!"))

	// All refresh tokens must be revoked.
	for _, rt := range repo.refreshTokens {
		if rt.UserID == user.ID {
			assert.True(t, rt.Revoked, "refresh token must be revoked after password reset")
		}
	}
}

// ============================================================================
// ResetPassword — error paths
// ============================================================================

func TestResetPassword_InvalidToken(t *testing.T) {
	svc, _, _ := newResetTestService()

	err := svc.ResetPassword(context.Background(), "completely-invalid-token", "NewPass99!")
	assert.ErrorIs(t, err, ErrResetTokenInvalid)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)

	// Manually insert an already-expired token.
	plain := generateSecureToken()
	hash := HashToken(plain)
	repo.passwordResetTokens[hash] = &models.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(-1 * time.Minute), // expired
		CreatedAt: time.Now().Add(-2 * time.Minute),
	}

	err := svc.ResetPassword(context.Background(), plain, "NewPass99!")
	assert.ErrorIs(t, err, ErrResetTokenExpired)
}

func TestResetPassword_AlreadyUsedToken(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)
	plainToken := issueTokenForUser(t, svc, user)

	// First use — must succeed.
	require.NoError(t, svc.ResetPassword(context.Background(), plainToken, "NewStrongPass99!"))

	// Second use of the same token — must be rejected.
	err := svc.ResetPassword(context.Background(), plainToken, "AnotherPass99!")
	assert.ErrorIs(t, err, ErrResetTokenUsed)
}

func TestResetPassword_WeakPassword_Rejected(t *testing.T) {
	svc, repo, _ := newResetTestService()
	// Override validator to reject.
	svc.SetPasswordValidator(&mockPasswordValidator{
		valid:   false,
		reasons: []string{"password entropy too low"},
	})
	user := seedResetUser(repo)
	plainToken := issueTokenForUser(t, svc, user)

	err := svc.ResetPassword(context.Background(), plainToken, "weak")
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
}

// ============================================================================
// Tenant isolation: token can only be used for the user's tenant
// ============================================================================

func TestResetPassword_TenantIsolation(t *testing.T) {
	svc, repo, _ := newResetTestService()
	user := seedResetUser(repo)
	plainToken := issueTokenForUser(t, svc, user)
	hash := HashToken(plainToken)

	stored := repo.passwordResetTokens[hash]
	require.NotNil(t, stored)
	assert.Equal(t, user.TenantID, stored.TenantID,
		"token's tenant_id must match the user's tenant_id")
}
