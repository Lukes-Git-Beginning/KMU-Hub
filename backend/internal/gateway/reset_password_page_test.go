package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The reset page is unauthenticated and echoes a URL-supplied token back into
// the markup, so escaping and the response headers are the parts worth pinning.

func TestResetPasswordPage_NoToken(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reset-password", nil)

	routes.HandleResetPasswordPage(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	if body := rec.Body.String(); !strings.Contains(body, "Link nicht mehr gültig") {
		t.Errorf("expected the invalid-link branch, got:\n%s", body)
	}
	if strings.Contains(rec.Body.String(), "<form") {
		t.Error("a token-less link must not render the form")
	}
}

func TestResetPasswordPage_RendersFormWithToken(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=abc123", nil)

	routes.HandleResetPasswordPage(rec, req)

	assertStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `name="token" value="abc123"`) {
		t.Errorf("token should be carried in a hidden field, got:\n%s", body)
	}
	if !strings.Contains(body, `name="new_password"`) {
		t.Error("form is missing the password field")
	}
}

// A token lands in the HTML unmodified, so a crafted one must not break out of
// the attribute. html/template handles this — the test keeps it that way.
func TestResetPasswordPage_EscapesToken(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	raw := `"><script>alert(1)</script>`
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token="+url.QueryEscape(raw), nil)

	routes.HandleResetPasswordPage(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Errorf("token was injected unescaped:\n%s", body)
	}
}

func TestResetPasswordPage_SecurityHeaders(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=abc123", nil)

	routes.HandleResetPasswordPage(rec, req)

	want := map[string]string{
		"Referrer-Policy": "no-referrer",
		"X-Robots-Tag":    "noindex, nofollow",
		"Cache-Control":   "no-store, max-age=0",
	}
	for header, expected := range want {
		if got := rec.Header().Get(header); got != expected {
			t.Errorf("%s = %q, want %q", header, got, expected)
		}
	}
	// The page carries no scripts; the policy must say so, otherwise an
	// injected tag would execute.
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("CSP = %q, want it to start from default-src 'none'", csp)
	}
}

func TestResetPasswordPageSubmit_RecoverableErrorsKeepToken(t *testing.T) {
	tests := []struct {
		name     string
		form     url.Values
		wantText string
	}{
		{
			name:     "mismatch",
			form:     url.Values{"token": {"tok"}, "new_password": {"longenough1"}, "confirm_password": {"different22"}},
			wantText: "stimmen nicht überein",
		},
		{
			name:     "too_short",
			form:     url.Values{"token": {"tok"}, "new_password": {"short"}, "confirm_password": {"short"}},
			wantText: "mindestens 8 Zeichen",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			routes := NewAuthRoutes(emptyRegistry())
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			routes.HandleResetPasswordPageSubmit(rec, req)

			assertStatus(t, rec, http.StatusBadRequest)
			body := rec.Body.String()
			if !strings.Contains(body, tc.wantText) {
				t.Errorf("expected %q, got:\n%s", tc.wantText, body)
			}
			// A typo must not cost the user their one-time link.
			if !strings.Contains(body, `name="token" value="tok"`) {
				t.Error("token was dropped from the re-rendered form")
			}
		})
	}
}

func TestResetPasswordPageSubmit_NoToken(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	form := url.Values{"new_password": {"longenough1"}, "confirm_password": {"longenough1"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routes.HandleResetPasswordPageSubmit(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	if !strings.Contains(rec.Body.String(), "kein Token") {
		t.Errorf("expected the missing-token message, got:\n%s", rec.Body.String())
	}
}

// With no auth backend reachable the user must land back on a retryable form,
// not on a dead end that burns the link.
func TestResetPasswordPageSubmit_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	form := url.Values{"token": {"tok"}, "new_password": {"longenough1"}, "confirm_password": {"longenough1"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routes.HandleResetPasswordPageSubmit(rec, req)

	assertStatus(t, rec, http.StatusServiceUnavailable)
	if !strings.Contains(rec.Body.String(), `name="token" value="tok"`) {
		t.Error("token should survive a transient backend outage")
	}
}

func TestResetPageErrorFor(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantText string
	}{
		{"invalid", status.Error(codes.NotFound, "x"), codes.NotFound, "ungültig"},
		{"expired_or_used", status.Error(codes.FailedPrecondition, "x"), codes.FailedPrecondition, "abgelaufen"},
		{"weak_password", status.Error(codes.InvalidArgument, "x"), codes.InvalidArgument, "Sicherheitsanforderungen"},
		{"unavailable", status.Error(codes.Unavailable, "x"), codes.Unavailable, "nicht erreichbar"},
		{"unknown", status.Error(codes.DataLoss, "x"), codes.Internal, "Fehler aufgetreten"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, msg := resetPageErrorFor(tc.err)
			if code != tc.wantCode {
				t.Errorf("code = %v, want %v", code, tc.wantCode)
			}
			if !strings.Contains(msg, tc.wantText) {
				t.Errorf("message %q should contain %q", msg, tc.wantText)
			}
		})
	}
}

// A rejected password is the one failure the user can fix, so it must return
// the form (with the token) rather than the terminal "link invalid" branch.
func TestResetPasswordPage_WeakPasswordStaysOnForm(t *testing.T) {
	code, _ := resetPageErrorFor(status.Error(codes.InvalidArgument, "weak"))
	if code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument to be preserved, got %v", code)
	}
}

// A HEAD on the page used to answer 405: chi does not fold HEAD into Get, and
// only GET and POST were registered. Irrelevant to a browser, not to a link
// checker or an uptime probe -- and `curl -I` showed the global headers rather
// than the hardened ones, which made the page look unprotected during diagnosis.
func TestResetPasswordPage_HeadIsRegistered(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	r := chi.NewRouter()
	routes.RegisterPublicRoutes(r, func(next http.Handler) http.Handler { return next })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/reset-password?token=abc123", nil))

	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatal("HEAD /reset-password answered 405 — the route is GET/POST only again")
	}
	assertStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer — a HEAD must carry the same hardening as the GET", got)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}
