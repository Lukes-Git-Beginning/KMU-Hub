package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the case-insensitive email normalization (lower+trim) added
// to every auth entrypoint, so "User@x" and "user@x" are the same account and a
// case-mismatched password reset still resolves the user. The service lowercases
// input before the repo lookup; the real DB query uses lower(email) (migration
// 000148). The mock store is keyed on the canonical lowercase form here,
// mirroring prod data after the backfill.

func TestService_Login_CaseInsensitiveEmail(t *testing.T) {
	svc, repo := newTestService()
	createTestUser(repo, "user@example.com", "correct-password", true)

	result, err := svc.Login(context.Background(), "  USER@Example.COM  ", "correct-password")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
}

func TestService_Register_StoresLowercaseEmail(t *testing.T) {
	svc, _ := newTestService()

	user, _, err := svc.Register(context.Background(), " New@Example.COM ", "StrongPass123!", "First", "Last")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "new@example.com", user.Email,
		"Register must persist the email lowercased+trimmed")
}

func TestService_Register_CaseInsensitiveDuplicate(t *testing.T) {
	svc, repo := newTestService()
	createTestUser(repo, "existing@example.com", "password", true)

	user, tokens, err := svc.Register(context.Background(), "Existing@Example.com", "StrongPass123!", "First", "Last")
	assert.ErrorIs(t, err, ErrUserExists)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestService_CreateInvitation_CaseInsensitiveDuplicate(t *testing.T) {
	svc, repo := newTestService()
	createTestUser(repo, "existing@example.com", "pass", true)

	inv, token, err := svc.CreateInvitation(context.Background(), testInviteTenant, "EXISTING@Example.com", "member", uuid.New())
	assert.ErrorIs(t, err, ErrUserExists)
	assert.Nil(t, inv)
	assert.Empty(t, token)
}

func TestService_CreateInvitation_StoresLowercaseEmail(t *testing.T) {
	svc, _ := newTestService()

	inv, token, err := svc.CreateInvitation(context.Background(), testInviteTenant, " New@Example.COM ", "member", uuid.New())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.NotEmpty(t, token)
	assert.Equal(t, "new@example.com", inv.Email,
		"CreateInvitation must persist the email lowercased+trimmed")
}

func TestService_RequestPasswordReset_CaseInsensitiveEmail(t *testing.T) {
	svc, repo := newTestService()
	createTestUser(repo, "user@example.com", "pass", true)

	// Mixed-case + whitespace must still resolve the user and mint a reset token.
	err := svc.RequestPasswordReset(context.Background(), "  USER@Example.COM  ")
	require.NoError(t, err)
	assert.Len(t, repo.passwordResetTokens, 1,
		"reset token must be created for a case-mismatched but existing email")
}
