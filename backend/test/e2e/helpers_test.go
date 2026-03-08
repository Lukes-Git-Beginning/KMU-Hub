//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
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
