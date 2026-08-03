//go:build smoke

package smoke

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

func smokeURL() string {
	if u := os.Getenv("SMOKE_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func expectVersion() string {
	return os.Getenv("SMOKE_EXPECT_VERSION")
}

func smokeCredentials() (email, password string) {
	email = fmt.Sprintf("smoke-%d@test.kmuhub.local", time.Now().UnixNano())
	password = "SmokeTest123!"
	return
}

type healthResponse struct {
	Status             string   `json:"status"`
	RegisteredServices []string `json:"registered_services"`
	Version            string   `json:"version"`
	Commit             string   `json:"commit"`
	BuildTime          string   `json:"build_time"`
}

type authResponse struct {
	User         map[string]interface{} `json:"user"`
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
}

func doRequest(t *testing.T, method, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

func doGet(t *testing.T, url, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodGet, url, nil, token)
}

func doPost(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, token)
}

func doDelete(t *testing.T, url, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodDelete, url, nil, token)
}

func requireStatus(t *testing.T, resp *http.Response, body []byte, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

func decodeBody(t *testing.T, body []byte, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(body))
	}
}

func registerAndLogin(t *testing.T, base string) (accessToken, userID string) {
	t.Helper()
	email, password := smokeCredentials()

	doPost(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Smoke",
		"last_name":  "Test",
	}, "")

	resp, body := doPost(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(body))
	}

	var auth authResponse
	decodeBody(t, body, &auth)

	id, ok := auth.User["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected user id in login response")
	}

	return auth.AccessToken, id
}

// promoteToAdmin grants the admin role to the given user via direct DB
// update. Fresh registrations only get the read-only member role, so smoke
// tests that exercise write endpoints need this. Skips the test when no
// DATABASE_URL is available (e.g. when pointing SMOKE_URL at production).
// Permissions are baked into the JWT at login time — re-login afterwards.
func promoteToAdmin(t *testing.T, userID string) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — cannot promote smoke user to admin")
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
	email, password := smokeCredentials()

	doPost(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Smoke",
		"last_name":  "Admin",
	}, "")

	resp, body := doPost(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(body))
	}

	var auth authResponse
	decodeBody(t, body, &auth)
	id, ok := auth.User["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected user id in login response")
	}

	promoteToAdmin(t, id)

	// Re-login: the first JWT still carries the member-only permission set.
	resp, body = doPost(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-login after admin promotion failed: %d %s", resp.StatusCode, string(body))
	}
	decodeBody(t, body, &auth)

	return auth.AccessToken, id
}

func cleanupUser(t *testing.T, base, userID, token string) {
	t.Helper()
	// Best-effort cleanup — don't fail test if this doesn't work
	resp, _ := doDelete(t, base+"/api/v1/auth/users/"+userID, token)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Logf("warning: cleanup user %s returned %d", userID, resp.StatusCode)
	}
}
