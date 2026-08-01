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
	tenantID := uuid.New()
	roles := []string{"admin", "member"}
	perms := []string{"contacts:read", "contacts:write"}

	token, err := tm.CreateAccessToken(userID, tenantID.String(), roles, perms, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := tm.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, tenantID.String(), claims.TenantID)
	assert.Equal(t, roles, claims.Roles)
	assert.Equal(t, perms, claims.Permissions)
	assert.Equal(t, "kmuhub", claims.Issuer)
}

// TestTokenMaker_TenantID_Roundtrip verifies that tid claim survives sign→parse intact.
func TestTokenMaker_TenantID_Roundtrip(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	tenantID := uuid.New()

	token, err := tm.CreateAccessToken(userID, tenantID.String(), nil, nil, nil)
	require.NoError(t, err)

	claims, err := tm.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, tenantID.String(), claims.TenantID)
}

// TestTokenMaker_TenantID_EmptyLegacy verifies that a token issued with an empty tid (legacy)
// parses successfully but exposes an empty TenantID — middleware must reject it.
func TestTokenMaker_TenantID_EmptyLegacy(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := tm.CreateAccessToken(userID, "", nil, nil, nil)
	require.NoError(t, err)

	claims, err := tm.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "", claims.TenantID, "empty tid must be preserved as-is — middleware is responsible for rejection")
}

func TestTokenMaker_ExpiredToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", -1*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := tm.CreateAccessToken(userID, uuid.New().String(), nil, nil, nil)
	require.NoError(t, err)

	_, err = tm.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestTokenMaker_InvalidSecret(t *testing.T) {
	tm1 := NewTokenMaker("secret-one-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	tm2 := NewTokenMaker("secret-two-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	token, err := tm1.CreateAccessToken(uuid.New(), uuid.New().String(), nil, nil, nil)
	require.NoError(t, err)

	_, err = tm2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestTokenMaker_TamperedToken(t *testing.T) {
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)

	token, err := tm.CreateAccessToken(uuid.New(), uuid.New().String(), nil, nil, nil)
	require.NoError(t, err)

	// Tamper with the signature by flipping multiple characters
	bytes := []byte(token)
	for i := len(bytes) - 5; i < len(bytes); i++ {
		if bytes[i] == 'A' {
			bytes[i] = 'B'
		} else {
			bytes[i] = 'A'
		}
	}
	_, err = tm.ValidateAccessToken(string(bytes))
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
