//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func gatewayURL() string {
	if u := os.Getenv("GATEWAY_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func waitForHealth(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("gateway did not become healthy within 60 seconds")
}

type authResponse struct {
	User         map[string]interface{} `json:"user"`
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func doRequest(t *testing.T, method, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var respBody bytes.Buffer
	respBody.ReadFrom(resp.Body)
	return resp, respBody.Bytes()
}

func postJSON(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, token)
}

func getJSON(t *testing.T, url string, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodGet, url, nil, token)
}

func putJSON(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPut, url, body, token)
}

func deleteJSON(t *testing.T, url string, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodDelete, url, nil, token)
}

// registerAndLogin creates a new user and returns the access token and user ID.
func registerAndLogin(t *testing.T, base string) (accessToken, userID string) {
	t.Helper()
	email := fmt.Sprintf("e2e-%d@test.com", time.Now().UnixNano())
	password := "SecurePass123!"

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "E2E",
		"last_name":  "Test",
	}, "")

	resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(body))
	}

	var authResp authResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	id, ok := authResp.User["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected user id in login response")
	}

	return authResp.AccessToken, id
}

// promoteToAdmin grants the admin role to the given user via direct DB update.
// Fresh registrations only get the read-only member role, so flow tests that
// exercise write endpoints need this. Permissions are baked into the JWT at
// login time — callers MUST re-login afterwards for the role to take effect.
func promoteToAdmin(t *testing.T, userID string) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL not set — required to promote e2e users to admin")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, tenant_id)
		SELECT u.id, r.id, u.tenant_id
		FROM users u, roles r
		WHERE u.id = $1 AND r.name = 'admin' AND r.tenant_id IS NULL
		ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		t.Fatalf("failed to assign admin role: %v", err)
	}
}

// registerAndLoginAdmin creates a new user, promotes it to admin in the DB,
// and logs in again so the returned token carries the admin permissions.
func registerAndLoginAdmin(t *testing.T, base string) (accessToken, userID string) {
	t.Helper()
	email := fmt.Sprintf("e2e-admin-%d@test.com", time.Now().UnixNano())
	password := "SecurePass123!"

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "E2E",
		"last_name":  "Admin",
	}, "")

	resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(body))
	}

	var authResp authResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	id, ok := authResp.User["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected user id in login response")
	}

	promoteToAdmin(t, id)

	// Re-login: the first JWT still carries the member-only permission set.
	resp, body = postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-login after admin promotion failed: %d %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &authResp); err != nil {
		t.Fatalf("failed to decode re-login response: %v", err)
	}

	return authResp.AccessToken, id
}

// requireStatus asserts the HTTP status code and returns the body for further inspection.
func requireStatus(t *testing.T, resp *http.Response, body []byte, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// decodeBody unmarshals JSON response body into the target.
func decodeBody(t *testing.T, body []byte, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("failed to decode response: %v\nbody: %s", err, string(body))
	}
}
