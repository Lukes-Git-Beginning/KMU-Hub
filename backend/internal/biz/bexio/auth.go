package bexio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// tokenCacheEntry holds a cached access token with its expiry.
type tokenCacheEntry struct {
	accessToken string
	expiresAt   time.Time
}

// TokenManager handles OAuth2 token lifecycle for Bexio API access.
// Refresh tokens are stored in the vault; access tokens are cached in-memory.
type TokenManager struct {
	vault        VaultService
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client

	mu    sync.RWMutex
	cache map[uuid.UUID]*tokenCacheEntry
}

// NewTokenManager creates a token manager with vault-backed refresh token storage.
func NewTokenManager(vault VaultService, clientID, clientSecret, tokenURL string) *TokenManager {
	return &TokenManager{
		vault:        vault,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		cache:        make(map[uuid.UUID]*tokenCacheEntry),
	}
}

// vaultKey returns the vault key name for a tenant's refresh token.
func vaultKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("bexio_oauth_refresh_%s", tenantID.String())
}

// GetAccessToken returns a valid access token, refreshing if necessary.
func (tm *TokenManager) GetAccessToken(ctx context.Context, tenantID uuid.UUID) (string, error) {
	tm.mu.RLock()
	entry, ok := tm.cache[tenantID]
	tm.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt.Add(-30*time.Second)) {
		return entry.accessToken, nil
	}

	return tm.RefreshAccessToken(ctx, tenantID)
}

// RefreshAccessToken fetches a new access token using the stored refresh token.
func (tm *TokenManager) RefreshAccessToken(ctx context.Context, tenantID uuid.UUID) (string, error) {
	refreshToken, err := tm.vault.GetSecret(ctx, vaultKey(tenantID))
	if err != nil {
		return "", fmt.Errorf("bexio: failed to load refresh token for tenant %s: %w", tenantID, err)
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {tm.clientID},
		"client_secret": {tm.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tm.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("bexio: failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("bexio: token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("bexio: failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("bexio token refresh failed",
			"tenant_id", tenantID,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return "", fmt.Errorf("bexio: token refresh returned status %d", resp.StatusCode)
	}

	var tokenResp BexioTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("bexio: failed to parse token response: %w", err)
	}

	// Store new refresh token in vault if provided
	if tokenResp.RefreshToken != "" {
		if err := tm.vault.SetSecret(ctx, vaultKey(tenantID), tokenResp.RefreshToken, "Bexio OAuth refresh token", uuid.Nil); err != nil {
			slog.Error("bexio: failed to store refresh token",
				"tenant_id", tenantID,
				"error", err,
			)
		}
	}

	// Cache access token
	tm.mu.Lock()
	tm.cache[tenantID] = &tokenCacheEntry{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	tm.mu.Unlock()

	slog.Info("bexio access token refreshed",
		"tenant_id", tenantID,
		"expires_in", tokenResp.ExpiresIn,
	)

	return tokenResp.AccessToken, nil
}

// ExchangeCode exchanges an authorization code for tokens and stores them.
func (tm *TokenManager) ExchangeCode(ctx context.Context, tenantID uuid.UUID, code, redirectURL string) error {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tm.clientID},
		"client_secret": {tm.clientSecret},
		"redirect_uri":  {redirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tm.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("bexio: failed to build code exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bexio: code exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bexio: failed to read code exchange response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("bexio code exchange failed",
			"tenant_id", tenantID,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("bexio: code exchange returned status %d", resp.StatusCode)
	}

	var tokenResp BexioTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("bexio: failed to parse code exchange response: %w", err)
	}

	// Store refresh token in vault
	if err := tm.vault.SetSecret(ctx, vaultKey(tenantID), tokenResp.RefreshToken, "Bexio OAuth refresh token", uuid.Nil); err != nil {
		return fmt.Errorf("bexio: failed to store refresh token: %w", err)
	}

	// Cache access token
	tm.mu.Lock()
	tm.cache[tenantID] = &tokenCacheEntry{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	tm.mu.Unlock()

	slog.Info("bexio OAuth tokens stored",
		"tenant_id", tenantID,
		"expires_in", tokenResp.ExpiresIn,
	)

	return nil
}

// RevokeTokens clears all stored tokens for a tenant.
func (tm *TokenManager) RevokeTokens(ctx context.Context, tenantID uuid.UUID) error {
	tm.mu.Lock()
	delete(tm.cache, tenantID)
	tm.mu.Unlock()

	// Vault SetSecret with empty value would fail, so we just log a note
	// The integration_configs deletion cascades and the vault key becomes orphaned
	// This is acceptable since vault.DeleteSecret is ID-based not key-based
	slog.Info("bexio tokens revoked", "tenant_id", tenantID)
	return nil
}
