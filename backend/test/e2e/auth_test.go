//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

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

func TestGetProfile(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	email := fmt.Sprintf("profile-%d@test.com", time.Now().UnixNano())
	password := "SecurePass123!"

	// Register and login
	_, _ = postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Profile",
		"last_name":  "Test",
	}, "")

	resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")

	var loginResp authResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	accessToken := loginResp.AccessToken

	t.Run("get_profile_success", func(t *testing.T) {
		resp, body = getJSON(t, base+"/api/v1/auth/me", accessToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var profileResp struct {
			User map[string]interface{} `json:"user"`
		}
		if err := json.Unmarshal(body, &profileResp); err != nil {
			t.Fatalf("failed to decode profile response: %v", err)
		}
		if profileResp.User["email"] != email {
			t.Fatalf("expected email %s, got %v", email, profileResp.User["email"])
		}
	})

	t.Run("get_profile_no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/auth/me", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestChangePassword(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	email := fmt.Sprintf("changepw-%d@test.com", time.Now().UnixNano())
	oldPassword := "OldPass123!"
	newPassword := "NewPass456!"

	// Register
	_, _ = postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      email,
		"password":   oldPassword,
		"first_name": "Change",
		"last_name":  "Password",
	}, "")

	// Login with old password
	resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": oldPassword,
	}, "")

	var loginResp authResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	accessToken := loginResp.AccessToken

	t.Run("change_password_success", func(t *testing.T) {
		resp, body = postJSON(t, base+"/api/v1/auth/change-password", map[string]string{
			"old_password": oldPassword,
			"new_password": newPassword,
		}, accessToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("login_with_old_password_fails", func(t *testing.T) {
		resp, _ := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": oldPassword,
		}, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 with old password, got %d", resp.StatusCode)
		}
	})

	t.Run("login_with_new_password_succeeds", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": newPassword,
		}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 with new password, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("change_password_wrong_old", func(t *testing.T) {
		// Login first to get new token
		resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": newPassword,
		}, "")
		var auth authResponse
		json.Unmarshal(body, &auth)

		resp, _ = postJSON(t, base+"/api/v1/auth/change-password", map[string]string{
			"old_password": "WrongPass!",
			"new_password": "AnotherPass!",
		}, auth.AccessToken)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for wrong old password, got %d", resp.StatusCode)
		}
	})
}

func TestInvitationFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminEmail := fmt.Sprintf("admin-%d@test.com", time.Now().UnixNano())
	adminPassword := "AdminPass123!"

	_, _ = postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":      adminEmail,
		"password":   adminPassword,
		"first_name": "Admin",
		"last_name":  "User",
	}, "")

	resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    adminEmail,
		"password": adminPassword,
	}, "")

	var adminAuth authResponse
	if err := json.Unmarshal(body, &adminAuth); err != nil {
		t.Fatalf("failed to decode admin login response: %v", err)
	}
	adminUserID := adminAuth.User["id"].(string)

	// Bootstrap: a fresh registration only has the member role, so the role
	// assignment API would 403. Promote directly in the DB, then re-login —
	// permissions are baked into the JWT at login time.
	promoteToAdmin(t, adminUserID)

	resp, body = postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email":    adminEmail,
		"password": adminPassword,
	}, "")
	requireStatus(t, resp, body, http.StatusOK)
	if err := json.Unmarshal(body, &adminAuth); err != nil {
		t.Fatalf("failed to decode admin re-login response: %v", err)
	}
	adminToken := adminAuth.AccessToken

	invitedEmail := fmt.Sprintf("invited-%d@test.com", time.Now().UnixNano())
	var invitationToken string
	var invitationID string

	t.Run("create_invitation", func(t *testing.T) {
		resp, body = postJSON(t, base+"/api/v1/invitations", map[string]string{
			"email": invitedEmail,
			"role":  "member",
		}, adminToken)

		requireStatus(t, resp, body, http.StatusCreated)

		var invResp struct {
			Invitation map[string]interface{} `json:"invitation"`
			Token      string                 `json:"token"`
		}
		decodeBody(t, body, &invResp)

		if invResp.Token == "" {
			t.Fatal("expected invitation token")
		}
		invitationToken = invResp.Token
		invitationID = invResp.Invitation["id"].(string)
	})

	t.Run("list_invitations", func(t *testing.T) {
		resp, body = getJSON(t, base+"/api/v1/invitations", adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		var listResp struct {
			Invitations []map[string]interface{} `json:"invitations"`
		}
		decodeBody(t, body, &listResp)

		found := false
		for _, inv := range listResp.Invitations {
			if inv["email"] == invitedEmail {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("created invitation not found in list")
		}
	})

	t.Run("accept_invitation", func(t *testing.T) {
		resp, body = postJSON(t, base+"/api/v1/invitations/"+invitationToken+"/accept", map[string]string{
			"password":   "InvitedPass123!",
			"first_name": "Invited",
			"last_name":  "User",
		}, "")

		requireStatus(t, resp, body, http.StatusCreated)

		var acceptResp authResponse
		decodeBody(t, body, &acceptResp)

		if acceptResp.AccessToken == "" {
			t.Fatal("expected access_token after accepting invitation")
		}
	})

	t.Run("login_as_invited_user", func(t *testing.T) {
		resp, body = postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    invitedEmail,
			"password": "InvitedPass123!",
		}, "")
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("accept_same_invitation_again_fails", func(t *testing.T) {
		resp, _ := postJSON(t, base+"/api/v1/invitations/"+invitationToken+"/accept", map[string]string{
			"password":   "AnotherPass123!",
			"first_name": "Another",
			"last_name":  "User",
		}, "")

		if resp.StatusCode == http.StatusCreated {
			t.Fatal("expected error when accepting same invitation twice")
		}
	})

	t.Run("cancel_invitation", func(t *testing.T) {
		cancelEmail := fmt.Sprintf("cancel-%d@test.com", time.Now().UnixNano())
		resp, body = postJSON(t, base+"/api/v1/invitations", map[string]string{
			"email": cancelEmail,
			"role":  "member",
		}, adminToken)

		var invResp struct {
			Invitation map[string]interface{} `json:"invitation"`
		}
		json.Unmarshal(body, &invResp)
		cancelID := invResp.Invitation["id"].(string)

		resp, body = deleteJSON(t, base+"/api/v1/invitations/"+cancelID, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/invitations", adminToken)
		var listResp struct {
			Invitations []map[string]interface{} `json:"invitations"`
		}
		json.Unmarshal(body, &listResp)

		for _, inv := range listResp.Invitations {
			if inv["email"] == cancelEmail {
				t.Fatal("cancelled invitation should not appear in list")
			}
		}
	})

	_ = invitationID
}
