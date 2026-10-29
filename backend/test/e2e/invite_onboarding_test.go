//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The pilot onboarding path end to end: an admin invites a colleague, the
// colleague opens /invite?token=, sets name and password, signs in and works as
// member. Neither CI nor the local stack has SMTP, so the token comes from the
// invite response (inviteToken); that the mail carries the same link is the
// unit test's job (internal/auth, fake mailer).
//
// The invited person is the actor from the accept step on: after the page they
// hold nothing but their own login.

// invitePageGet fetches the public accept page without following redirects.
func invitePageGet(t *testing.T, base, token string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(base + "/invite?token=" + url.QueryEscape(token))
	if err != nil {
		t.Fatalf("GET /invite failed: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// invitePagePost submits the accept form the way a browser does.
func invitePagePost(t *testing.T, base string, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := http.PostForm(base+"/invite", form)
	if err != nil {
		t.Fatalf("POST /invite failed: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// memberRoleID looks up the member preset's UUID. The server sends a system
// role's UUID as id and its preset key as name.
func memberRoleID(t *testing.T, base, adminToken string) string {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/admin/roles", adminToken)
	requireStatus(t, resp, body, http.StatusOK)
	var roles struct {
		Roles []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			IsSystem bool   `json:"isSystem"`
		} `json:"roles"`
	}
	decodeBody(t, body, &roles)
	for _, r := range roles.Roles {
		if r.IsSystem && r.Name == "member" {
			return r.ID
		}
	}
	t.Fatalf("no system role named member in %s", string(body))
	return ""
}

type inviteResult struct {
	InviteToken string `json:"inviteToken"`
	User        struct {
		ID     string   `json:"id"`
		Email  string   `json:"email"`
		Roles  []string `json:"roles"`
		Status string   `json:"status"`
	} `json:"user"`
}

func inviteMember(t *testing.T, base, adminToken, roleID, email string) inviteResult {
	t.Helper()
	resp, body := postJSON(t, base+"/api/v1/admin/users/invite", map[string]interface{}{
		"email":     email,
		"firstName": "Neu",
		"lastName":  "Kollege",
		"roles":     []string{roleID},
	}, adminToken)
	requireStatus(t, resp, body, http.StatusCreated)
	var res inviteResult
	decodeBody(t, body, &res)
	if res.InviteToken == "" {
		t.Fatalf("invite response carries no inviteToken: %s", string(body))
	}
	return res
}

func TestInviteOnboarding(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginAdmin(t, base)
	roleID := memberRoleID(t, base, adminToken)

	email := fmt.Sprintf("invite-onboarding-%d@test.com", time.Now().UnixNano())
	const password = "Onboarding-Kollege-2026!"
	inv := inviteMember(t, base, adminToken, roleID, email)

	t.Run("invited_row_is_pending", func(t *testing.T) {
		if inv.User.Email != email {
			t.Fatalf("expected invited email %s, got %s", email, inv.User.Email)
		}
		if inv.User.Status != "invited" {
			t.Fatalf("expected status invited, got %q", inv.User.Status)
		}
	})

	t.Run("get_page_shows_prefilled_form", func(t *testing.T) {
		resp, html := invitePageGet(t, base, inv.InviteToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, html)
		}
		for _, want := range []string{`action="/invite"`, `value="Neu"`, `value="Kollege"`, email} {
			if !strings.Contains(html, want) {
				t.Fatalf("form lacks %q:\n%s", want, html)
			}
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Fatalf("expected Cache-Control no-store, got %q", cc)
		}
	})

	t.Run("get_page_with_unknown_token_is_dead", func(t *testing.T) {
		resp, html := invitePageGet(t, base, "not-a-real-token")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
		if !strings.Contains(html, "Einladung nicht mehr gültig") {
			t.Fatalf("expected the dead-link state:\n%s", html)
		}
	})

	t.Run("post_creates_the_account", func(t *testing.T) {
		resp, html := invitePagePost(t, base, url.Values{
			"token":            {inv.InviteToken},
			"first_name":       {"Neu"},
			"last_name":        {"Kollege"},
			"password":         {password},
			"confirm_password": {password},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, html)
		}
		if !strings.Contains(html, "Konto angelegt") {
			t.Fatalf("expected the success state:\n%s", html)
		}
		// The page sets no session and prints no token pair.
		if resp.Header.Get("Set-Cookie") != "" {
			t.Fatalf("accept page must not set a cookie, got %q", resp.Header.Get("Set-Cookie"))
		}
		if strings.Contains(html, "access_token") || strings.Contains(html, "eyJ") {
			t.Fatalf("accept page leaks a token:\n%s", html)
		}
	})

	var memberToken string
	t.Run("invited_user_logs_in_as_member", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, "")
		requireStatus(t, resp, body, http.StatusOK)
		var auth authResponse
		decodeBody(t, body, &auth)
		memberToken = auth.AccessToken
		if memberToken == "" {
			t.Fatal("login returned no access token")
		}
	})

	t.Run("me_shows_name_and_member_role", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/auth/me", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var me struct {
			User struct {
				Email     string   `json:"email"`
				FirstName string   `json:"first_name"`
				LastName  string   `json:"last_name"`
				Roles     []string `json:"roles"`
			} `json:"user"`
		}
		decodeBody(t, body, &me)
		if me.User.Email != email || me.User.FirstName != "Neu" || me.User.LastName != "Kollege" {
			t.Fatalf("unexpected profile: %s", string(body))
		}
		if len(me.User.Roles) != 1 || me.User.Roles[0] != "member" {
			t.Fatalf("expected exactly the member role, got %v", me.User.Roles)
		}
	})

	t.Run("member_cannot_use_admin_routes", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/admin/users/", memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("admin_roster_lists_the_account_as_active", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/admin/users/", adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var roster struct {
			Users []struct {
				Email  string `json:"email"`
				Status string `json:"status"`
			} `json:"users"`
		}
		decodeBody(t, body, &roster)
		for _, u := range roster.Users {
			if u.Email == email {
				if u.Status != "active" {
					t.Fatalf("expected status active after accept, got %q", u.Status)
				}
				return
			}
		}
		t.Fatalf("accepted account %s missing from the admin roster", email)
	})

	t.Run("second_post_with_same_token_is_rejected", func(t *testing.T) {
		resp, html := invitePagePost(t, base, url.Values{
			"token":            {inv.InviteToken},
			"first_name":       {"Zweiter"},
			"last_name":        {"Versuch"},
			"password":         {"Another-Passphrase-2026!"},
			"confirm_password": {"Another-Passphrase-2026!"},
		})
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("a spent token was accepted again: %s", html)
		}
		if !strings.Contains(html, "Einladung nicht mehr gültig") {
			t.Fatalf("expected the dead-link state, got %d:\n%s", resp.StatusCode, html)
		}
		// The first password still works, the second attempt created nothing.
		loginResp, loginBody := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email": email, "password": password,
		}, "")
		requireStatus(t, loginResp, loginBody, http.StatusOK)
	})

	t.Run("get_page_after_accept_is_dead", func(t *testing.T) {
		resp, html := invitePageGet(t, base, inv.InviteToken)
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("a spent token still renders the form:\n%s", html)
		}
	})
}

func TestInviteResendKillsTheOldLink(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginAdmin(t, base)
	roleID := memberRoleID(t, base, adminToken)

	email := fmt.Sprintf("invite-resend-%d@test.com", time.Now().UnixNano())
	const password = "Resend-Passphrase-2026!"
	inv := inviteMember(t, base, adminToken, roleID, email)

	// The path id of a resend is the INVITATION id, which the invited roster
	// row carries as its id.
	resp, body := postJSON(t, base+"/api/v1/admin/users/"+inv.User.ID+"/resend-invite", nil, adminToken)
	requireStatus(t, resp, body, http.StatusOK)
	var resent inviteResult
	decodeBody(t, body, &resent)
	if resent.InviteToken == "" || resent.InviteToken == inv.InviteToken {
		t.Fatalf("resend must issue a fresh token, got %q (old %q)", resent.InviteToken, inv.InviteToken)
	}

	t.Run("old_link_is_dead_on_get_and_post", func(t *testing.T) {
		getResp, html := invitePageGet(t, base, inv.InviteToken)
		if getResp.StatusCode == http.StatusOK {
			t.Fatalf("old link still renders the form:\n%s", html)
		}
		postResp, html := invitePagePost(t, base, url.Values{
			"token":            {inv.InviteToken},
			"first_name":       {"Alt"},
			"last_name":        {"Link"},
			"password":         {password},
			"confirm_password": {password},
		})
		if postResp.StatusCode == http.StatusOK {
			t.Fatalf("old link created an account:\n%s", html)
		}
		loginResp, _ := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email": email, "password": password,
		}, "")
		if loginResp.StatusCode == http.StatusOK {
			t.Fatal("the dead link produced a working login")
		}
	})

	t.Run("new_link_accepts", func(t *testing.T) {
		getResp, html := invitePageGet(t, base, resent.InviteToken)
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for the new link, got %d:\n%s", getResp.StatusCode, html)
		}
		postResp, html := invitePagePost(t, base, url.Values{
			"token":            {resent.InviteToken},
			"first_name":       {"Neu"},
			"last_name":        {"Kollege"},
			"password":         {password},
			"confirm_password": {password},
		})
		if postResp.StatusCode != http.StatusOK || !strings.Contains(html, "Konto angelegt") {
			t.Fatalf("expected the success state, got %d:\n%s", postResp.StatusCode, html)
		}
		loginResp, loginBody := postJSON(t, base+"/api/v1/auth/login", map[string]string{
			"email": email, "password": password,
		}, "")
		requireStatus(t, loginResp, loginBody, http.StatusOK)
	})
}
