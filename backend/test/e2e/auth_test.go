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

func postJSON(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
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

func getJSON(t *testing.T, url string, token string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
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

func TestAuthFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	email := fmt.Sprintf("e2e-%d@test.com", time.Now().UnixNano())
	password := "SecurePass123!"

	// Step 1: Register
	t.Run("register", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/register", map[string]string{
			"email":      email,
			"password":   password,
			"first_name": "E2E",
			"last_name":  "Test",
		}, "")

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
		}

		var authResp authResponse
		if err := json.Unmarshal(body, &authResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if authResp.AccessToken == "" {
			t.Fatal("expected access_token in register response")
		}
		if authResp.RefreshToken == "" {
			t.Fatal("expected refresh_token in register response")
		}
	})

	// Step 2: Register duplicate should fail
	t.Run("register_duplicate", func(t *testing.T) {
		resp, _ := postJSON(t, base+"/api/v1/auth/register", map[string]string{
			"email":    email,
			"password": password,
		}, "")
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409 for duplicate, got %d", resp.StatusCode)
		}
	})

	// Step 3: Login
	var accessToken, refreshToken, userID string
	t.Run("login", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, "")

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var authResp authResponse
		if err := json.Unmarshal(body, &authResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		accessToken = authResp.AccessToken
		refreshToken = authResp.RefreshToken

		id, ok := authResp.User["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected user id in login response")
		}
		userID = id
	})

	// Step 4: Access protected route with token
	t.Run("get_user", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/users/"+userID, accessToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	// Step 5: Access protected route without token should fail
	t.Run("get_user_no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/users/"+userID, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
		}
	})

	// Step 6: Refresh token
	t.Run("refresh", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/refresh", map[string]string{
			"refresh_token": refreshToken,
		}, "")

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var refreshResp refreshResponse
		if err := json.Unmarshal(body, &refreshResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if refreshResp.AccessToken == "" {
			t.Fatal("expected new access_token")
		}
		accessToken = refreshResp.AccessToken
		refreshToken = refreshResp.RefreshToken
	})

	// Step 7: Logout
	t.Run("logout", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/logout", map[string]string{
			"refresh_token": refreshToken,
		}, "")

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	// Step 8: Refresh with old token should fail after logout
	t.Run("refresh_after_logout", func(t *testing.T) {
		resp, _ := postJSON(t, base+"/api/v1/auth/refresh", map[string]string{
			"refresh_token": refreshToken,
		}, "")

		if resp.StatusCode == http.StatusOK {
			t.Fatal("expected refresh to fail after logout")
		}
	})
}
