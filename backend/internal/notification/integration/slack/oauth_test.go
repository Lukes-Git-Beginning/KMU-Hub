package slack

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// roundTripFunc replaces http.DefaultClient's transport: HandleCallback calls
// slack.com through the default client, so that is the one seam.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubSlackToken(t *testing.T, status int, body string) *url.Values {
	t.Helper()

	seen := &url.Values{}
	orig := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "slack.com", r.URL.Host)
		require.Equal(t, "/api/oauth.v2.access", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*seen, err = url.ParseQuery(string(raw))
		require.NoError(t, err)
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	t.Cleanup(func() { http.DefaultClient.Transport = orig })
	return seen
}

func TestOAuthHandler_HandleInstall_RedirectsToSlackWithScopesAndCallback(t *testing.T) {
	h := NewOAuthHandler("cid&x=1", "secret", "https://app.example/cb?x=1&y=2")

	rec := httptest.NewRecorder()
	h.HandleInstall(rec, httptest.NewRequest(http.MethodGet, "/install", nil))

	require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "slack.com", loc.Host)
	require.Equal(t, "/oauth/v2/authorize", loc.Path)
	q := loc.Query()
	require.Equal(t, "cid&x=1", q.Get("client_id"))
	require.Empty(t, q.Get("x"), "client id and redirect are escaped, they cannot inject parameters")
	require.Equal(t, slackBotScopes, q.Get("scope"))
	require.Equal(t, "https://app.example/cb?x=1&y=2", q.Get("redirect_uri"))
	require.NotContains(t, rec.Header().Get("Location"), "secret", "the client secret never goes into the redirect")
}

func TestOAuthHandler_HandleCallback(t *testing.T) {
	h := NewOAuthHandler("cid", "csecret", "https://app.example/cb")

	t.Run("rejects a callback without code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleCallback(rec, httptest.NewRequest(http.MethodGet, "/cb", nil))

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("exchanges the code and returns to the integration settings", func(t *testing.T) {
		seen := stubSlackToken(t, http.StatusOK,
			`{"ok":true,"access_token":"xoxb-new","bot_user_id":"UBOT","team":{"id":"T1","name":"Zentria"}}`)

		rec := httptest.NewRecorder()
		h.HandleCallback(rec, httptest.NewRequest(http.MethodGet, "/cb?code=abc123", nil))

		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		require.Equal(t, "/settings?tab=integrations&slack=connected", rec.Header().Get("Location"))
		require.Equal(t, "abc123", seen.Get("code"))
		require.Equal(t, "cid", seen.Get("client_id"))
		require.Equal(t, "csecret", seen.Get("client_secret"))
		require.Equal(t, "https://app.example/cb", seen.Get("redirect_uri"))
	})

	t.Run("a rejected code is a 500, not a connected banner", func(t *testing.T) {
		stubSlackToken(t, http.StatusOK, `{"ok":false,"error":"invalid_code"}`)

		rec := httptest.NewRecorder()
		h.HandleCallback(rec, httptest.NewRequest(http.MethodGet, "/cb?code=stale", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Empty(t, rec.Header().Get("Location"))
		require.NotContains(t, rec.Body.String(), "invalid_code", "Slack's error code is logged, not echoed")
	})
}
