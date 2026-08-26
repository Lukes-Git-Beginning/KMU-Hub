package datev

// Covers OAuthManager without ever calling the real DATEV token endpoint:
// expiry-based cache reuse, refresh triggering, form contents sent to the
// token endpoint (via httptest.NewServer), error mapping for vault and HTTP
// failures, and the authorization URL builder. No real or placeholder DATEV
// credentials are used anywhere.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// vaultStub is a minimal in-memory VaultService for oauth.go and uploader.go
// tests — no real vault, no real secrets.
type vaultStub struct {
	secret   string
	getErr   error
	setErr   error
	setCalls []struct{ key, plaintext string }
}

func (v *vaultStub) GetSecret(_ context.Context, _ string) (string, error) {
	return v.secret, v.getErr
}

func (v *vaultStub) SetSecret(_ context.Context, keyName, plaintext, _ string, _ uuid.UUID) error {
	v.setCalls = append(v.setCalls, struct{ key, plaintext string }{keyName, plaintext})
	return v.setErr
}

// tokenServer stands in for the DATEV token endpoint. checkForm, if non-nil,
// receives the parsed request body so a test can assert on the exact grant
// sent.
func tokenServer(t *testing.T, status int, body string, checkForm func(url.Values)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if checkForm != nil {
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			checkForm(r.PostForm)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestGetAccessToken_ReturnsCachedTokenWithoutRefresh(t *testing.T) {
	tenantID := uuid.New()
	vault := &vaultStub{getErr: errors.New("vault must not be called for a fresh cache entry")}
	om := NewOAuthManager(vault, "cid", "csecret", "http://unused.invalid")
	om.cache[tenantID] = &tokenCacheEntry{accessToken: "cached-token", expiresAt: time.Now().Add(time.Hour)}

	token, err := om.GetAccessToken(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetAccessToken: %v", err)
	}
	if token != "cached-token" {
		t.Errorf("token = %q, want cached-token", token)
	}
}

func TestGetAccessToken_RefreshesWhenCacheNearExpiry(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `{"access_token":"fresh-token","token_type":"Bearer","expires_in":3600}`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "refresh-abc"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)
	// Within the 30s early-refresh window used by GetAccessToken.
	om.cache[tenantID] = &tokenCacheEntry{accessToken: "stale-token", expiresAt: time.Now().Add(5 * time.Second)}

	token, err := om.GetAccessToken(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetAccessToken: %v", err)
	}
	if token != "fresh-token" {
		t.Errorf("token = %q, want fresh-token", token)
	}
}

// TestGetAccessToken_ConcurrentColdCacheSameTenantSharesOneRefresh proves the
// thundering-herd fix: two concurrent GetAccessToken calls for the SAME
// tenant with a cold cache collapse into exactly one HTTP roundtrip via
// refreshGroup, instead of each independently reaching RefreshAccessToken
// with the SAME (not yet rotated) refresh token. At a token endpoint that
// rotates refresh tokens on use, two independent requests would have one
// invalidate the token the other is still presenting.
func TestGetAccessToken_ConcurrentColdCacheSameTenantSharesOneRefresh(t *testing.T) {
	tenantID := uuid.New()

	var mu sync.Mutex
	calls := 0
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()

		if n == 1 {
			// Hold the first response until the second request has had a
			// chance to arrive, so the test proves a would-be-second call
			// never reaches the server rather than merely losing a race.
			select {
			case <-release:
			case <-time.After(200 * time.Millisecond):
			}
		} else {
			close(release)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
	}))
	defer server.Close()

	vault := &vaultStub{secret: "refresh-shared"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			if _, err := om.GetAccessToken(context.Background(), tenantID); err != nil {
				t.Errorf("GetAccessToken: %v", err)
			}
		}()
	}
	wg.Wait()

	if calls != 1 {
		t.Errorf("token endpoint called %d times, want exactly 1 (refreshGroup must collapse concurrent same-tenant calls)", calls)
	}
}

// TestGetAccessToken_ConcurrentColdCacheDifferentTenantsBothRefresh proves the
// per-tenant lock does not serialize unrelated tenants against each other —
// a global lock would fix the herd but wrongly block tenants that share
// nothing.
func TestGetAccessToken_ConcurrentColdCacheDifferentTenantsBothRefresh(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()

	var mu sync.Mutex
	calls := 0
	bothArrived := make(chan struct{})
	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()

		if n == 1 {
			// Block the first arrival until the second tenant's request also
			// reaches the server, proving neither tenant waited on the other.
			select {
			case <-bothArrived:
			case <-time.After(2 * time.Second):
			}
		} else {
			once.Do(func() { close(bothArrived) })
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
	}))
	defer server.Close()

	vault := &vaultStub{secret: "refresh-shared"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	var wg sync.WaitGroup
	wg.Add(2)
	for _, tenantID := range []uuid.UUID{tenantA, tenantB} {
		go func() {
			defer wg.Done()
			if _, err := om.GetAccessToken(context.Background(), tenantID); err != nil {
				t.Errorf("GetAccessToken: %v", err)
			}
		}()
	}
	wg.Wait()

	if calls != 2 {
		t.Errorf("token endpoint called %d times, want 2 (different tenants must not block each other)", calls)
	}
}

// TestRefreshLocked_ReturnsCacheHitWithoutHittingVaultOrServer proves the
// double-checked cache read inside refreshLocked: a caller that enters the
// per-tenant singleflight section after the cache was already refreshed by
// someone else must not trigger a second, avoidable HTTP roundtrip. The vault
// stub errors if touched at all, so any refresh attempt fails the test.
func TestRefreshLocked_ReturnsCacheHitWithoutHittingVaultOrServer(t *testing.T) {
	tenantID := uuid.New()
	vault := &vaultStub{getErr: errors.New("must not refresh: the cache is already warm")}
	om := NewOAuthManager(vault, "cid", "csecret", "http://unused.invalid")
	om.cache[tenantID] = &tokenCacheEntry{accessToken: "warm-token", expiresAt: time.Now().Add(time.Hour)}

	token, err := om.refreshLocked(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("refreshLocked: %v", err)
	}
	if token != "warm-token" {
		t.Errorf("token = %q, want warm-token (double-checked cache hit, not a fresh refresh)", token)
	}
}

func TestRefreshAccessToken_VaultErrorPropagates(t *testing.T) {
	tenantID := uuid.New()
	vault := &vaultStub{getErr: errors.New("vault down")}
	om := NewOAuthManager(vault, "cid", "csecret", "http://unused.invalid")

	_, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err == nil || !strings.Contains(err.Error(), "failed to load refresh token") {
		t.Fatalf("err = %v, want a wrapped vault error", err)
	}
}

func TestRefreshAccessToken_SendsExpectedFormAndCachesToken(t *testing.T) {
	tenantID := uuid.New()
	var gotForm url.Values
	server := tokenServer(t, http.StatusOK, `{"access_token":"tok-1","token_type":"Bearer","expires_in":120}`, func(f url.Values) {
		gotForm = f
	})
	defer server.Close()

	vault := &vaultStub{secret: "refresh-xyz"}
	om := NewOAuthManager(vault, "the-client", "the-secret", server.URL)

	token, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if token != "tok-1" {
		t.Errorf("token = %q, want tok-1", token)
	}
	if gotForm.Get("grant_type") != "refresh_token" || gotForm.Get("refresh_token") != "refresh-xyz" ||
		gotForm.Get("client_id") != "the-client" || gotForm.Get("client_secret") != "the-secret" {
		t.Errorf("unexpected form sent to token endpoint: %v", gotForm)
	}

	om.mu.RLock()
	entry := om.cache[tenantID]
	om.mu.RUnlock()
	if entry == nil || entry.accessToken != "tok-1" {
		t.Fatal("refreshed token was not cached")
	}
	if entry.expiresAt.Before(time.Now().Add(100 * time.Second)) {
		t.Error("expiresAt was not computed from expires_in")
	}
}

// TestRefreshAccessToken_NonOKStatusReturnsError covers the invalid_grant case
// DATEV returns for an expired or revoked refresh token: a 4xx from the token
// endpoint. This must come back as ErrReauthRequired, not a generic error —
// mapDatevUploadError uses errors.Is on it to give the admin an actionable
// "please reconnect" instead of an opaque "DATEV upload failed".
func TestRefreshAccessToken_NonOKStatusReturnsError(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusUnauthorized, `{"error":"invalid_grant"}`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "refresh-xyz"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	_, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want a status-401 error", err)
	}
	if !errors.Is(err, ErrReauthRequired) {
		t.Errorf("err = %v, want errors.Is(err, ErrReauthRequired)", err)
	}
}

// TestRefreshAccessToken_ServerErrorIsNotReauthRequired distinguishes a DATEV
// outage (5xx) from a rejected refresh token (4xx): only the latter means
// retrying is futile. A 5xx must NOT be wrapped as ErrReauthRequired, or an
// admin would be told to reconnect DATEV when the actual connection is fine
// and DATEV is just down.
func TestRefreshAccessToken_ServerErrorIsNotReauthRequired(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusServiceUnavailable, `{"error":"unavailable"}`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "refresh-xyz"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	_, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v, want a status-503 error", err)
	}
	if errors.Is(err, ErrReauthRequired) {
		t.Errorf("err = %v, must not be ErrReauthRequired for a 5xx", err)
	}
}

func TestRefreshAccessToken_MalformedJSONReturnsError(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `not json`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "refresh-xyz"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	_, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err == nil || !strings.Contains(err.Error(), "failed to parse token response") {
		t.Fatalf("err = %v, want a parse error", err)
	}
}

func TestRefreshAccessToken_StoresNewRefreshTokenInVault(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `{"access_token":"tok-1","expires_in":60,"refresh_token":"new-refresh"}`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "old-refresh"}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	if _, err := om.RefreshAccessToken(context.Background(), tenantID); err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if len(vault.setCalls) != 1 || vault.setCalls[0].plaintext != "new-refresh" {
		t.Fatalf("vault.SetSecret calls = %+v, want one call storing new-refresh", vault.setCalls)
	}
	if vault.setCalls[0].key != datevVaultKey(tenantID) {
		t.Errorf("vault key = %q, want %q", vault.setCalls[0].key, datevVaultKey(tenantID))
	}
}

func TestRefreshAccessToken_SucceedsEvenWhenVaultStoreFails(t *testing.T) {
	// A refresh must not fail the caller just because persisting the new
	// refresh token failed — the fresh access token is still valid.
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `{"access_token":"tok-1","expires_in":60,"refresh_token":"new-refresh"}`, nil)
	defer server.Close()

	vault := &vaultStub{secret: "old-refresh", setErr: errors.New("vault write failed")}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	token, err := om.RefreshAccessToken(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v, want success despite the vault store failure", err)
	}
	if token != "tok-1" {
		t.Errorf("token = %q, want tok-1", token)
	}
}

func TestExchangeCode_SendsExpectedFormAndStoresTokens(t *testing.T) {
	tenantID := uuid.New()
	var gotForm url.Values
	server := tokenServer(t, http.StatusOK, `{"access_token":"tok-ex","expires_in":90,"refresh_token":"refresh-ex"}`, func(f url.Values) {
		gotForm = f
	})
	defer server.Close()

	vault := &vaultStub{}
	om := NewOAuthManager(vault, "the-client", "the-secret", server.URL)

	err := om.ExchangeCode(context.Background(), tenantID, "auth-code-1", "https://app.example/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if gotForm.Get("grant_type") != "authorization_code" || gotForm.Get("code") != "auth-code-1" ||
		gotForm.Get("redirect_uri") != "https://app.example/callback" {
		t.Errorf("unexpected form sent to token endpoint: %v", gotForm)
	}
	if len(vault.setCalls) != 1 || vault.setCalls[0].plaintext != "refresh-ex" {
		t.Fatalf("vault.SetSecret calls = %+v, want one call storing refresh-ex", vault.setCalls)
	}

	om.mu.RLock()
	entry := om.cache[tenantID]
	om.mu.RUnlock()
	if entry == nil || entry.accessToken != "tok-ex" {
		t.Fatal("token was not cached after code exchange")
	}
}

func TestExchangeCode_VaultStoreFailureFailsTheExchange(t *testing.T) {
	// Unlike a refresh, ExchangeCode has no prior refresh token on file —
	// losing the new one here leaves the tenant stuck, so it must be a hard
	// error rather than a logged-and-ignored one.
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `{"access_token":"tok-ex","expires_in":90,"refresh_token":"refresh-ex"}`, nil)
	defer server.Close()

	vault := &vaultStub{setErr: errors.New("vault write failed")}
	om := NewOAuthManager(vault, "cid", "csecret", server.URL)

	err := om.ExchangeCode(context.Background(), tenantID, "auth-code-1", "https://app.example/callback")
	if err == nil || !strings.Contains(err.Error(), "failed to store refresh token") {
		t.Fatalf("err = %v, want a store-failure error", err)
	}
}

func TestExchangeCode_NonOKStatusReturnsError(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusBadRequest, `{"error":"invalid_code"}`, nil)
	defer server.Close()

	om := NewOAuthManager(&vaultStub{}, "cid", "csecret", server.URL)

	err := om.ExchangeCode(context.Background(), tenantID, "bad-code", "https://app.example/callback")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v, want a status-400 error", err)
	}
}

func TestExchangeCode_MalformedJSONReturnsError(t *testing.T) {
	tenantID := uuid.New()
	server := tokenServer(t, http.StatusOK, `not json`, nil)
	defer server.Close()

	om := NewOAuthManager(&vaultStub{}, "cid", "csecret", server.URL)

	err := om.ExchangeCode(context.Background(), tenantID, "code", "https://app.example/callback")
	if err == nil || !strings.Contains(err.Error(), "failed to parse code exchange response") {
		t.Fatalf("err = %v, want a parse error", err)
	}
}

func TestRevokeTokens_ClearsCachedToken(t *testing.T) {
	tenantID := uuid.New()
	om := NewOAuthManager(&vaultStub{}, "cid", "csecret", "http://unused.invalid")
	om.cache[tenantID] = &tokenCacheEntry{accessToken: "tok", expiresAt: time.Now().Add(time.Hour)}

	if err := om.RevokeTokens(context.Background(), tenantID); err != nil {
		t.Fatalf("RevokeTokens: %v", err)
	}

	om.mu.RLock()
	_, ok := om.cache[tenantID]
	om.mu.RUnlock()
	if ok {
		t.Error("cache entry survived RevokeTokens")
	}
}

func TestGetAuthorizationURL_BuildsExpectedQuery(t *testing.T) {
	tenantID := uuid.New()
	om := NewOAuthManager(&vaultStub{}, "the-client", "csecret", "http://unused.invalid")

	got := om.GetAuthorizationURL(tenantID, "https://app.example/callback", "https://login.datev.de/authorize")

	if !strings.HasPrefix(got, "https://login.datev.de/authorize?") {
		t.Fatalf("URL = %q, want the authBaseURL prefix", got)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "the-client" {
		t.Errorf("client_id = %q, want the-client", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://app.example/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want code", q.Get("response_type"))
	}
	if q.Get("state") != tenantID.String() {
		t.Errorf("state = %q, want tenant id %s", q.Get("state"), tenantID)
	}
}
