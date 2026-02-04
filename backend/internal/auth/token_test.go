package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenMaker_CreateAndValidateAccessToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	roles := []string{"admin", "member"}
	perms := []string{"contacts:read", "contacts:write"}

	token, err := tm.CreateAccessToken(userID, roles, perms)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := tm.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, roles, claims.Roles)
	assert.Equal(t, perms, claims.Permissions)
	assert.Equal(t, "kmuhub", claims.Issuer)
}

func TestTokenMaker_ExpiredToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", -1*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := tm.CreateAccessToken(userID, nil, nil)
	require.NoError(t, err)

	_, err = tm.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestTokenMaker_InvalidSecret(t *testing.T) {
	tm1 := NewTokenMaker("secret-one-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	tm2 := NewTokenMaker("secret-two-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	token, err := tm1.CreateAccessToken(uuid.New(), nil, nil)
	require.NoError(t, err)

	_, err = tm2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestTokenMaker_TamperedToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	token, err := tm.CreateAccessToken(uuid.New(), nil, nil)
	require.NoError(t, err)

	// Tamper with the token
	tampered := token[:len(token)-1] + "X"
	_, err = tm.ValidateAccessToken(tampered)
	assert.Error(t, err)
}

func TestTokenMaker_EmptyToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	_, err := tm.ValidateAccessToken("")
	assert.Error(t, err)
}

func TestTokenMaker_CreateRefreshToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	plain, hash, expiresAt := tm.CreateRefreshToken()

	assert.NotEmpty(t, plain)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, plain, hash)
	assert.True(t, expiresAt.After(time.Now()))

	// Hash should be deterministic
	assert.Equal(t, hash, HashToken(plain))
}

func TestTokenMaker_RefreshTokenUniqueness(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	plain1, hash1, _ := tm.CreateRefreshToken()
	plain2, hash2, _ := tm.CreateRefreshToken()

	assert.NotEqual(t, plain1, plain2)
	assert.NotEqual(t, hash1, hash2)
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"normal token", "abc123def456"},
		{"empty string", ""},
		{"long token", "a-very-long-token-string-that-simulates-a-real-refresh-token-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := HashToken(tt.token)
			h2 := HashToken(tt.token)
			assert.Equal(t, h1, h2, "hash should be deterministic")
			assert.Len(t, h1, 64, "SHA-256 hex should be 64 chars")
		})
	}
}
