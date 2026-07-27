package teams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func probeClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &Client{
		appID:       "app-id",
		appPassword: "app-secret",
		tokenURL:    srv.URL,
		httpClient:  srv.Client(),
	}
}

func TestProbeConnectionRequestsAChannelToken(t *testing.T) {
	var gotForm url.Values

	c := probeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"eyJ0eXAi","expires_in":3600}`))
	})

	result, err := c.ProbeConnection(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(result.Detail, "app-id") {
		t.Fatalf("expected the app id in the detail, got %q", result.Detail)
	}
	if got := gotForm.Get("grant_type"); got != "client_credentials" {
		t.Fatalf("expected a client_credentials grant, got %q", got)
	}
	if got := gotForm.Get("client_secret"); got != "app-secret" {
		t.Fatalf("expected the app password to be sent, got %q", got)
	}
}

// An HTTP 401 from AAD is the shape a wrong app password takes; it must not
// become a passing connection test.
func TestProbeConnectionFailsOnRejectedCredentials(t *testing.T) {
	c := probeClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
	})

	_, err := c.ProbeConnection(context.Background())
	if err == nil {
		t.Fatal("expected an error for rejected credentials")
	}
	if !strings.Contains(err.Error(), "unauthorized_client") {
		t.Fatalf("expected the platform reason in the error, got %v", err)
	}
}

// A 200 without a token is not a working integration either.
func TestProbeConnectionFailsWithoutAccessToken(t *testing.T) {
	c := probeClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer"}`))
	})

	if _, err := c.ProbeConnection(context.Background()); err == nil {
		t.Fatal("expected an error when no access token is returned")
	}
}
