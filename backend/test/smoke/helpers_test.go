//go:build smoke

package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
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

func cleanupUser(t *testing.T, base, userID, token string) {
	t.Helper()
	// Best-effort cleanup — don't fail test if this doesn't work
	resp, _ := doDelete(t, base+"/api/v1/auth/users/"+userID, token)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Logf("warning: cleanup user %s returned %d", userID, resp.StatusCode)
	}
}
