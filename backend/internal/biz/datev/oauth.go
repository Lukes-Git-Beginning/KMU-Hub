package datev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// ErrReauthRequired means DATEV rejected the stored refresh token (a 4xx from
// the token endpoint — typically invalid_grant because the token expired or
// was revoked). Retrying the same request cannot succeed; the admin has to
// go through the OAuth flow again. Distinct from a 5xx/network failure at the
// token endpoint, which is transient and worth retrying.
var ErrReauthRequired = errors.New("datev: reauthorization required, please reconnect")

type VaultService interface {
	GetSecret(ctx context.Context, keyName string) (string, error)
	SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error
}

type DatevTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type tokenCacheEntry struct {
	accessToken string
	expiresAt   time.Time
}

type OAuthManager struct {
	vault        VaultService
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client

	mu    sync.RWMutex
	cache map[uuid.UUID]*tokenCacheEntry

	// refreshGroup collapses concurrent GetAccessToken calls for the same
	// tenant into one HTTP roundtrip instead of each caller independently
	// refreshing (and, when DATEV rotates refresh tokens on use, invalidating
	// each other's requests). Keyed by tenantID.String().
	refreshGroup singleflight.Group
}

func NewOAuthManager(vault VaultService, clientID, clientSecret, tokenURL string) *OAuthManager {
	return &OAuthManager{
		vault:        vault,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		cache:        make(map[uuid.UUID]*tokenCacheEntry),
	}
}

func datevVaultKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("datev_oauth_refresh_%s", tenantID.String())
}

func (om *OAuthManager) GetAccessToken(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if token, ok := om.cachedToken(tenantID); ok {
		return token, nil
	}

	// Per-tenant singleflight: concurrent callers for the SAME tenant share one
	// refresh instead of each independently hitting the token endpoint (and,
	// on a token endpoint that rotates refresh tokens, invalidating each
	// other's requests). Different tenants use different keys and never
	// block each other.
	v, err, _ := om.refreshGroup.Do(tenantID.String(), func() (any, error) {
		return om.refreshLocked(ctx, tenantID)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// refreshLocked runs inside refreshGroup's per-tenant section. It re-checks
// the cache before refreshing (double-checked): a caller that only entered
// the singleflight group after a previous refresh for this tenant already
// completed would otherwise trigger a redundant, avoidable HTTP roundtrip.
func (om *OAuthManager) refreshLocked(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if token, ok := om.cachedToken(tenantID); ok {
		return token, nil
	}
	return om.RefreshAccessToken(ctx, tenantID)
}

func (om *OAuthManager) cachedToken(tenantID uuid.UUID) (string, bool) {
	om.mu.RLock()
	entry, ok := om.cache[tenantID]
	om.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt.Add(-30*time.Second)) {
		return entry.accessToken, true
	}
	return "", false
}

func (om *OAuthManager) RefreshAccessToken(ctx context.Context, tenantID uuid.UUID) (string, error) {
	refreshToken, err := om.vault.GetSecret(ctx, datevVaultKey(tenantID))
	if err != nil {
		return "", fmt.Errorf("datev: failed to load refresh token for tenant %s: %w", tenantID, err)
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {om.clientID},
		"client_secret": {om.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, om.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("datev: failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := om.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("datev: token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("datev: failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("datev token refresh failed",
			"tenant_id", tenantID,
			"status", resp.StatusCode,
			"body", string(body),
		)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// The token endpoint rejected the refresh token itself (invalid_grant),
			// not a transient server problem — reissuing the same request will
			// fail again until the admin reconnects.
			return "", fmt.Errorf("%w (status %d)", ErrReauthRequired, resp.StatusCode)
		}
		return "", fmt.Errorf("datev: token refresh returned status %d", resp.StatusCode)
	}

	var tokenResp DatevTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("datev: failed to parse token response: %w", err)
	}

	if tokenResp.RefreshToken != "" {
		if err := om.vault.SetSecret(ctx, datevVaultKey(tenantID), tokenResp.RefreshToken, "DATEV OAuth refresh token", uuid.Nil); err != nil {
			slog.Error("datev: failed to store refresh token", "tenant_id", tenantID, "error", err)
		}
	}

	om.mu.Lock()
	om.cache[tenantID] = &tokenCacheEntry{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	om.mu.Unlock()

	slog.Info("datev access token refreshed", "tenant_id", tenantID, "expires_in", tokenResp.ExpiresIn)
	return tokenResp.AccessToken, nil
}

func (om *OAuthManager) ExchangeCode(ctx context.Context, tenantID uuid.UUID, code, redirectURL string) error {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {om.clientID},
		"client_secret": {om.clientSecret},
		"redirect_uri":  {redirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, om.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("datev: failed to build code exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := om.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("datev: code exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("datev: failed to read code exchange response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("datev code exchange failed",
			"tenant_id", tenantID,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("datev: code exchange returned status %d", resp.StatusCode)
	}

	var tokenResp DatevTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("datev: failed to parse code exchange response: %w", err)
	}

	if err := om.vault.SetSecret(ctx, datevVaultKey(tenantID), tokenResp.RefreshToken, "DATEV OAuth refresh token", uuid.Nil); err != nil {
		return fmt.Errorf("datev: failed to store refresh token: %w", err)
	}

	om.mu.Lock()
	om.cache[tenantID] = &tokenCacheEntry{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	om.mu.Unlock()

	slog.Info("datev OAuth tokens stored", "tenant_id", tenantID, "expires_in", tokenResp.ExpiresIn)
	return nil
}

func (om *OAuthManager) RevokeTokens(ctx context.Context, tenantID uuid.UUID) error {
	om.mu.Lock()
	delete(om.cache, tenantID)
	om.mu.Unlock()

	slog.Info("datev tokens revoked", "tenant_id", tenantID)
	return nil
}

func (om *OAuthManager) GetAuthorizationURL(tenantID uuid.UUID, redirectURL, authBaseURL string) string {
	params := url.Values{
		"client_id":     {om.clientID},
		"redirect_uri":  {redirectURL},
		"response_type": {"code"},
		"scope":         {"datev:accounting:documents"},
		"state":         {tenantID.String()},
	}
	return authBaseURL + "?" + params.Encode()
}
