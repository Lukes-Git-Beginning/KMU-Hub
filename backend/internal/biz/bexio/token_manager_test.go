package bexio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTokenManager creates a TokenManager pointing its token-endpoint at the
// given httptest.Server URL.
func buildTokenManager(srv *httptest.Server, vault VaultService) *TokenManager {
	return NewTokenManager(vault, "client-id", "client-secret", srv.URL+"/oauth/token")
}

// TestGetAccessToken_CacheHit verifies that a still-valid cached token is
// returned without hitting the token endpoint.
func TestGetAccessToken_CacheHit(t *testing.T) {
	hitCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		// Should not be called.
		http.Error(w, "unexpected token refresh call", http.StatusInternalServerError)
	}))
	defer srv.Close()

	vault := &mockVaultService{token: "refresh-token"}
	tm := buildTokenManager(srv, vault)
	tenantID := uuid.New()

	// Pre-seed the cache with a valid token expiring in 10 minutes
	tm.mu.Lock()
	tm.cache[tenantID] = &tokenCacheEntry{
		accessToken: "cached-access-token",
		expiresAt:   time.Now().Add(10 * time.Minute),
	}
	tm.mu.Unlock()

	token, err := tm.GetAccessToken(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, "cached-access-token", token)
	assert.Equal(t, 0, hitCount, "token endpoint must not be called when cache is valid")
}

// TestGetAccessToken_RefreshOnExpiry verifies that when the cached token is
// expiring soon (within 30 s grace period) RefreshAccessToken is called and
// the new token is returned + cached.
func TestGetAccessToken_RefreshOnExpiry(t *testing.T) {
	newToken := "new-access-token"
	newRefreshToken := "new-refresh-token"

	var vaultSetCalls int
	vault := &mockVaultService{
		token: "old-refresh-token",
		// Count SetSecret calls so we know the new refresh token was persisted.
	}
	vaultWithCounting := &countingVaultService{
		inner:    vault,
		setCalls: &vaultSetCalls,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			resp := BexioTokenResponse{
				AccessToken:  newToken,
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: newRefreshToken,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	tm := buildTokenManager(srv, vaultWithCounting)
	tenantID := uuid.New()

	// Seed with an almost-expired token (expires in 10 seconds = within 30s grace)
	tm.mu.Lock()
	tm.cache[tenantID] = &tokenCacheEntry{
		accessToken: "expiring-token",
		expiresAt:   time.Now().Add(10 * time.Second),
	}
	tm.mu.Unlock()

	token, err := tm.GetAccessToken(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, newToken, token)

	// Verify new token is now cached
	tm.mu.RLock()
	cached := tm.cache[tenantID]
	tm.mu.RUnlock()
	require.NotNil(t, cached)
	assert.Equal(t, newToken, cached.accessToken)
	assert.True(t, cached.expiresAt.After(time.Now().Add(30*time.Minute)),
		"new token expiry must be in the future")

	assert.Equal(t, 1, vaultSetCalls, "new refresh token must be persisted to vault")
}

// TestGetAccessToken_NoCache_RefreshCalled verifies that when there is no
// cached token at all, RefreshAccessToken is called to obtain one.
func TestGetAccessToken_NoCache_RefreshCalled(t *testing.T) {
	accessToken := "fresh-access-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			resp := BexioTokenResponse{
				AccessToken: accessToken,
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	vault := &mockVaultService{token: "stored-refresh-token"}
	tm := buildTokenManager(srv, vault)
	tenantID := uuid.New()
	// No pre-seeded cache entry.

	token, err := tm.GetAccessToken(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, accessToken, token)
}

// TestRefreshAccessToken_VaultError verifies that when the vault cannot return
// the refresh token, RefreshAccessToken returns an error.
func TestRefreshAccessToken_VaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("token endpoint called unexpectedly")
	}))
	defer srv.Close()

	vault := &mockVaultService{getErr: errVaultUnavailable}
	tm := buildTokenManager(srv, vault)
	tenantID := uuid.New()

	_, err := tm.RefreshAccessToken(context.Background(), tenantID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load refresh token")
}

// TestRefreshAccessToken_ServerError verifies that a non-200 token endpoint
// response returns an error.
func TestRefreshAccessToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "token server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	vault := &mockVaultService{token: "refresh-token"}
	tm := buildTokenManager(srv, vault)
	tenantID := uuid.New()

	_, err := tm.RefreshAccessToken(context.Background(), tenantID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token refresh returned status")
}

// TestRefreshAccessToken_NoNewRefreshToken verifies that when the token endpoint
// does not return a new refresh token, the vault SetSecret is NOT called
// (we keep the existing refresh token).
func TestRefreshAccessToken_NoNewRefreshToken(t *testing.T) {
	var vaultSetCalls int
	vaultWithCounting := &countingVaultService{
		inner:    &mockVaultService{token: "existing-refresh"},
		setCalls: &vaultSetCalls,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			// No RefreshToken in response (server did not rotate it)
			resp := BexioTokenResponse{
				AccessToken: "new-access",
				ExpiresIn:   3600,
				// RefreshToken intentionally omitted
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer srv.Close()

	tm := buildTokenManager(srv, vaultWithCounting)
	tenantID := uuid.New()

	_, err := tm.RefreshAccessToken(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, vaultSetCalls, "vault.SetSecret must not be called when token endpoint omits refresh token")
}

// --- helpers ---

var errVaultUnavailable = errors.New("vault service unavailable")

// countingVaultService wraps a VaultService and counts SetSecret calls.
type countingVaultService struct {
	inner    VaultService
	setCalls *int
}

func (c *countingVaultService) GetSecret(ctx context.Context, keyName string) (string, error) {
	return c.inner.GetSecret(ctx, keyName)
}

func (c *countingVaultService) SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error {
	*c.setCalls++
	return c.inner.SetSecret(ctx, keyName, plaintext, description, createdBy)
}
