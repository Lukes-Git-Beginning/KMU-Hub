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

// connectDB opens the e2e database connection used for the role and
// reporting-line fixtures that have no public endpoint.
func connectDB(t *testing.T, ctx context.Context) *pgx.Conn {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL not set — required for e2e role and reporting-line fixtures")
	}
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	return conn
}

// promoteToRole grants the given preset role (member|manager|admin) to the
// user via direct DB writes. Fresh registrations only get the member role, so
// flow tests that exercise write endpoints or the manager surface need this.
// Permissions are baked into the JWT at login time — callers MUST re-login
// afterwards for the role to take effect.
//
// Production replaces the whole role set when an admin edits a user
// (UpdateAdminUser), so a manager is promoted by swapping member for manager:
// keeping the member row would only widen the manager's token with member keys
// and could hide a gap in the manager preset. admin stays additive on top of
// member, as promoteToAdmin always did.
func promoteToRole(t *testing.T, userID, role string) {
	t.Helper()
	switch role {
	case "member", "manager", "admin":
	default:
		t.Fatalf("promoteToRole: unsupported role %q", role)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := connectDB(t, ctx)
	defer conn.Close(ctx)

	tag, err := conn.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, tenant_id)
		SELECT u.id, r.id, u.tenant_id
		FROM users u, roles r
		WHERE u.id = $1 AND r.name = $2 AND r.tenant_id IS NULL
		ON CONFLICT DO NOTHING`, userID, role)
	if err != nil {
		t.Fatalf("failed to assign %s role: %v", role, err)
	}
	if tag.RowsAffected() == 0 {
		// ON CONFLICT swallows "already assigned"; only a missing user or
		// preset is a fixture error.
		var assigned bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
			               WHERE ur.user_id = $1 AND r.name = $2 AND r.tenant_id IS NULL)`,
			userID, role).Scan(&assigned); err != nil || !assigned {
			t.Fatalf("role %s was not assigned to user %s (user or preset missing): %v", role, userID, err)
		}
	}

	if role == "manager" {
		if _, err := conn.Exec(ctx, `
			DELETE FROM user_roles
			WHERE user_id = $1
			  AND role_id = (SELECT id FROM roles WHERE name = 'member' AND tenant_id IS NULL)`, userID); err != nil {
			t.Fatalf("failed to drop member role: %v", err)
		}
	}
}

// promoteToAdmin grants the admin role; see promoteToRole.
func promoteToAdmin(t *testing.T, userID string) {
	t.Helper()
	promoteToRole(t, userID, "admin")
}

// registerAndLoginWithRole creates a new user, promotes it to the given preset
// role (member|manager|admin) in the DB, and logs in again so the returned
// token carries that role's permissions. member needs no promotion.
func registerAndLoginWithRole(t *testing.T, base, role string) (accessToken, userID string) {
	t.Helper()
	lastName := map[string]string{"member": "Test", "manager": "Manager", "admin": "Admin"}[role]
	if lastName == "" {
		t.Fatalf("registerAndLoginWithRole: unsupported role %q", role)
	}
	email := fmt.Sprintf("e2e-%s-%d@test.com", role, time.Now().UnixNano())
	password := "SecurePass123!"

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "E2E",
		"last_name":  lastName,
	}, "")

	login := func() (string, string) {
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

	token, id := login()
	if role == "member" {
		return token, id
	}

	promoteToRole(t, id, role)

	// Re-login: the first JWT still carries the member-only permission set.
	return login()
}

// registerAndLoginAdmin creates a new user, promotes it to admin in the DB,
// and logs in again so the returned token carries the admin permissions.
func registerAndLoginAdmin(t *testing.T, base string) (accessToken, userID string) {
	t.Helper()
	return registerAndLoginWithRole(t, base, "admin")
}

// setReportingLine points an employee's HR profile at their manager
// (hr_employee_profiles.manager_user_id). The profile must already exist —
// create it through the API first — and the manager scope of the team-time and
// leave-approval guards reads exactly this column.
func setReportingLine(t *testing.T, employeeUserID, managerUserID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := connectDB(t, ctx)
	defer conn.Close(ctx)

	tag, err := conn.Exec(ctx,
		`UPDATE hr_employee_profiles SET manager_user_id = $2 WHERE user_id = $1`,
		employeeUserID, managerUserID)
	if err != nil {
		t.Fatalf("failed to set reporting line: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("no HR profile for employee %s — create it before setting the reporting line", employeeUserID)
	}
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
