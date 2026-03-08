//go:build smoke

package smoke

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	base := smokeURL()

	start := time.Now()
	resp, body := doGet(t, base+"/health", "")
	elapsed := time.Since(start)

	requireStatus(t, resp, body, http.StatusOK)

	var health healthResponse
	decodeBody(t, body, &health)

	if health.Status != "healthy" {
		t.Fatalf("expected healthy status, got %q", health.Status)
	}

	if len(health.RegisteredServices) < 10 {
		t.Fatalf("expected >= 10 registered services, got %d: %v", len(health.RegisteredServices), health.RegisteredServices)
	}

	if elapsed > 2*time.Second {
		t.Fatalf("health response took %s, expected < 2s", elapsed)
	}
}

func TestHealthVersionInfo(t *testing.T) {
	base := smokeURL()

	_, body := doGet(t, base+"/health", "")

	var raw map[string]interface{}
	decodeBody(t, body, &raw)

	for _, field := range []string{"version", "commit", "build_time"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("health response missing %q field", field)
		}
	}

	if ev := expectVersion(); ev != "" {
		commit, _ := raw["commit"].(string)
		version, _ := raw["version"].(string)
		if commit != ev && version != ev {
			t.Errorf("expected version/commit %q, got version=%q commit=%q", ev, version, commit)
		}
	}
}

func TestAuthRegisterAndLogin(t *testing.T) {
	base := smokeURL()
	token, userID := registerAndLogin(t, base)

	if token == "" {
		t.Fatal("expected non-empty access token")
	}
	if userID == "" {
		t.Fatal("expected non-empty user ID")
	}

	// Verify /auth/me
	resp, body := doGet(t, base+"/api/v1/auth/me", token)
	requireStatus(t, resp, body, http.StatusOK)

	// Cleanup
	cleanupUser(t, base, userID, token)
}

func TestCRMContactCRUD(t *testing.T) {
	base := smokeURL()
	token, userID := registerAndLogin(t, base)
	defer cleanupUser(t, base, userID, token)

	// Create contact
	createResp, createBody := doPost(t, base+"/api/v1/contacts", map[string]string{
		"first_name": "Smoke",
		"last_name":  "Contact",
		"email":      "smoke-contact@test.kmuhub.local",
	}, token)

	if createResp.StatusCode != http.StatusCreated && createResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /contacts: expected 201 or 200, got %d: %s", createResp.StatusCode, string(createBody))
	}

	var created map[string]interface{}
	decodeBody(t, createBody, &created)

	contactID := extractID(t, created)

	// Get contact
	getResp, getBody := doGet(t, base+"/api/v1/contacts/"+contactID, token)
	requireStatus(t, getResp, getBody, http.StatusOK)

	// Delete contact
	delResp, delBody := doDelete(t, base+"/api/v1/contacts/"+contactID, token)
	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /contacts/%s: expected 200 or 204, got %d: %s", contactID, delResp.StatusCode, string(delBody))
	}
}

func TestSecurityUnauthenticated(t *testing.T) {
	base := smokeURL()

	resp, _ := doGet(t, base+"/api/v1/contacts", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /contacts, got %d", resp.StatusCode)
	}
}

func TestSecurityCORSHeaders(t *testing.T) {
	base := smokeURL()

	req, err := http.NewRequest(http.MethodOptions, base+"/api/v1/contacts", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://app.zentria.tech")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") == "" &&
		resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("no CORS headers in preflight response")
	}
}

func TestPerformanceHealth(t *testing.T) {
	base := smokeURL()

	start := time.Now()
	resp, body := doGet(t, base+"/health", "")
	elapsed := time.Since(start)

	requireStatus(t, resp, body, http.StatusOK)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("health took %s, expected < 500ms", elapsed)
	}
}

func TestPerformanceLogin(t *testing.T) {
	base := smokeURL()
	email, password := smokeCredentials()

	// Register first
	doPost(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Perf",
		"last_name":  "Test",
	}, "")

	start := time.Now()
	resp, body := doPost(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	elapsed := time.Since(start)

	requireStatus(t, resp, body, http.StatusOK)

	if elapsed > 2*time.Second {
		t.Fatalf("login took %s, expected < 2s", elapsed)
	}

	// Cleanup
	var auth authResponse
	decodeBody(t, body, &auth)
	if id, ok := auth.User["id"].(string); ok {
		cleanupUser(t, base, id, auth.AccessToken)
	}
}

func TestPerformanceContactsList(t *testing.T) {
	base := smokeURL()
	token, userID := registerAndLogin(t, base)
	defer cleanupUser(t, base, userID, token)

	start := time.Now()
	resp, body := doGet(t, base+"/api/v1/contacts", token)
	elapsed := time.Since(start)

	requireStatus(t, resp, body, http.StatusOK)

	if elapsed > 1*time.Second {
		t.Fatalf("contacts list took %s, expected < 1s", elapsed)
	}
}

func TestCrossServiceChat(t *testing.T) {
	base := smokeURL()
	token, userID := registerAndLogin(t, base)
	defer cleanupUser(t, base, userID, token)

	resp, body := doPost(t, base+"/api/v1/chat/channels", map[string]string{
		"name": "smoke-test-channel",
		"type": "public",
	}, token)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /chat/channels: expected 201 or 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// extractID tries common ID field patterns from a JSON response.
func extractID(t *testing.T, m map[string]interface{}) string {
	t.Helper()

	// Direct "id" field
	if id, ok := m["id"].(string); ok && id != "" {
		return id
	}

	// Nested under common wrappers
	for _, key := range []string{"contact", "data"} {
		if nested, ok := m[key].(map[string]interface{}); ok {
			if id, ok := nested["id"].(string); ok && id != "" {
				return id
			}
		}
	}

	// Try as json.Number
	if id, ok := m["id"].(json.Number); ok {
		return id.String()
	}

	t.Fatal("could not extract ID from response")
	return ""
}
