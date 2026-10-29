package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// inviteAuthStub plays the auth service's invitation state machine over a real
// gRPC hop: "good" is pending until accepted once, "expired" is past its date,
// "weak-policy" rejects every password like a strict tenant policy would.
type inviteAuthStub struct {
	authv1.UnimplementedAuthServiceServer
	mu       sync.Mutex
	accepted map[string]bool
	got      *authv1.AcceptInvitationRequest
}

func (s *inviteAuthStub) state(token string) error {
	switch {
	case token == "expired":
		return status.Error(codes.FailedPrecondition, "invitation expired")
	case token == "seatless":
		return nil
	case token != "good" && token != "weak-policy":
		return status.Error(codes.NotFound, "invitation not found")
	case s.accepted[token]:
		return status.Error(codes.FailedPrecondition, "invitation already used")
	}
	return nil
}

func (s *inviteAuthStub) GetInvitationPreview(_ context.Context, r *authv1.GetInvitationPreviewRequest) (*authv1.GetInvitationPreviewResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state(r.Token); err != nil {
		return nil, err
	}
	return &authv1.GetInvitationPreviewResponse{Email: "neu@example.com", FirstName: "Nora", LastName: "Neu"}, nil
}

func (s *inviteAuthStub) AcceptInvitation(_ context.Context, r *authv1.AcceptInvitationRequest) (*authv1.AcceptInvitationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = r
	if err := s.state(r.Token); err != nil {
		return nil, err
	}
	switch r.Token {
	case "weak-policy":
		return nil, status.Error(codes.InvalidArgument, "password too weak")
	case "seatless":
		return nil, status.Error(codes.FailedPrecondition, "no seat available: the booked seats are all taken")
	}
	s.accepted[r.Token] = true
	return &authv1.AcceptInvitationResponse{
		User:        &authv1.UserInfo{Email: "neu@example.com"},
		AccessToken: "access-must-not-leak", RefreshToken: "refresh-must-not-leak",
	}, nil
}

// inviteRouter serves the public routes against the stub, the way main.go
// mounts them (limiter is a pass-through here).
func inviteRouter(t *testing.T) (http.Handler, *inviteAuthStub) {
	t.Helper()
	stub := &inviteAuthStub{accepted: map[string]bool{}}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("auth", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewAuthRoutes(reg).RegisterPublicRoutes(r, func(next http.Handler) http.Handler { return next })
	return r, stub
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func postInvite(h http.Handler, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/invite", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return serve(h, req)
}

func validInviteForm(token string) url.Values {
	return url.Values{
		"token": {token}, "first_name": {"Nora"}, "last_name": {"Neu"},
		"password": {"ein-langes-passwort"}, "confirm_password": {"ein-langes-passwort"},
	}
}

func TestInvitePage_GetValidTokenShowsPrefilledForm(t *testing.T) {
	h, _ := inviteRouter(t)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/invite?token=good", nil))

	assertStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	assert.Contains(t, body, `action="/invite"`)
	assert.Contains(t, body, `name="token" value="good"`, "token rides in the hidden field")
	assert.Contains(t, body, `value="Nora"`)
	assert.Contains(t, body, `value="Neu"`)
	assert.Contains(t, body, "neu@example.com")
}

func TestInvitePage_GetDeadOrMissingTokenShowsErrorState(t *testing.T) {
	h, _ := inviteRouter(t)
	cases := []struct {
		path     string
		wantCode int
		wantText string
	}{
		{"/invite", http.StatusBadRequest, "kein Token"},
		{"/invite?token=nope", http.StatusNotFound, "ungültig"},
		{"/invite?token=expired", http.StatusGone, "abgelaufen"},
	}
	for _, tc := range cases {
		rec := serve(h, httptest.NewRequest(http.MethodGet, tc.path, nil))
		assertStatus(t, rec, tc.wantCode)
		body := rec.Body.String()
		assert.Contains(t, body, tc.wantText, tc.path)
		assert.Contains(t, body, "Einladung nicht mehr gültig", tc.path)
		assert.NotContains(t, body, "<form", "%s: a dead link offers no form", tc.path)
	}
}

func TestInvitePage_SecurityHeadersAndHead(t *testing.T) {
	h, _ := inviteRouter(t)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		rec := serve(h, httptest.NewRequest(method, "/invite?token=good", nil))
		require.NotEqual(t, http.StatusMethodNotAllowed, rec.Code, "%s /invite must be registered", method)
		assertStatus(t, rec, http.StatusOK)
		hd := rec.Header()
		assert.Contains(t, hd.Get("Content-Security-Policy"), "default-src 'none'", method)
		assert.NotContains(t, hd.Get("Content-Security-Policy"), "script-src", method)
		assert.Equal(t, "no-store, max-age=0", hd.Get("Cache-Control"), method)
		assert.Equal(t, "noindex, nofollow", hd.Get("X-Robots-Tag"), method)
		assert.Equal(t, "no-referrer", hd.Get("Referrer-Policy"), method)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/invite?token=good", nil))
	assert.NotContains(t, rec.Body.String(), "<script", "the page ships no JavaScript")
}

func TestInvitePage_PostCreatesAccountOnceWithoutSession(t *testing.T) {
	h, stub := inviteRouter(t)

	rec := postInvite(h, validInviteForm("good"))
	assertStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	assert.Contains(t, body, "Konto angelegt")
	assert.Contains(t, body, "Cosmi-App")
	assert.NotContains(t, body, "must-not-leak", "the token pair never reaches the page")
	assert.Empty(t, rec.Header().Values("Set-Cookie"), "the page sets no session")

	require.NotNil(t, stub.got)
	assert.Equal(t, "Nora", stub.got.FirstName)
	assert.Equal(t, "Neu", stub.got.LastName)
	assert.Equal(t, "ein-langes-passwort", stub.got.Password)

	// The same link a second time: spent, terminal state, no form.
	again := postInvite(h, validInviteForm("good"))
	assertStatus(t, again, http.StatusGone)
	assert.Contains(t, again.Body.String(), "Einladung nicht mehr gültig")
	assert.NotContains(t, again.Body.String(), "<form")
}

func TestInvitePage_PostRecoverableErrorsKeepForm(t *testing.T) {
	h, stub := inviteRouter(t)
	mismatch := validInviteForm("good")
	mismatch.Set("confirm_password", "etwas-anderes")
	short := validInviteForm("good")
	short.Set("password", "kurz")
	short.Set("confirm_password", "kurz")
	noName := validInviteForm("good")
	noName.Set("last_name", "  ")

	cases := []struct {
		name     string
		form     url.Values
		wantCode int
		wantText string
	}{
		{"mismatch", mismatch, http.StatusBadRequest, "stimmen nicht überein"},
		{"short", short, http.StatusBadRequest, "mindestens 8 Zeichen"},
		{"no_name", noName, http.StatusBadRequest, "Vor- und Nachnamen"},
		{"policy", validInviteForm("weak-policy"), http.StatusBadRequest, "Passwort-Richtlinie"},
		{"seat", validInviteForm("seatless"), http.StatusConflict, "kein Platz frei"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postInvite(h, tc.form)
			assertStatus(t, rec, tc.wantCode)
			body := rec.Body.String()
			assert.Contains(t, body, tc.wantText)
			assert.Contains(t, body, "<form", "a fixable problem keeps the form")
			assert.Contains(t, body, `name="token" value="`+tc.form.Get("token")+`"`)
			assert.NotContains(t, body, "ein-langes-passwort", "a password is never echoed back")
		})
	}
	assert.False(t, stub.accepted["good"], "client-side checks stop before the service")
}

func TestInvitePage_PostWithoutTokenOrServiceUnavailable(t *testing.T) {
	h, _ := inviteRouter(t)
	form := validInviteForm("")
	rec := postInvite(h, form)
	assertStatus(t, rec, http.StatusBadRequest)
	assert.Contains(t, rec.Body.String(), "kein Token")

	// No auth service registered: the form stays, with the token, and says so.
	routes := NewAuthRoutes(emptyRegistry())
	r := chi.NewRouter()
	routes.RegisterPublicRoutes(r, func(next http.Handler) http.Handler { return next })
	rec = postInvite(r, validInviteForm("good"))
	assertStatus(t, rec, http.StatusServiceUnavailable)
	assert.Contains(t, rec.Body.String(), "nicht erreichbar")
	assert.Contains(t, rec.Body.String(), `name="token" value="good"`)

	rec = serve(r, httptest.NewRequest(http.MethodGet, "/invite?token=good", nil))
	assertStatus(t, rec, http.StatusServiceUnavailable)
	assert.Contains(t, rec.Body.String(), "nicht erreichbar")
}

func TestInvitePageErrorFor_UnknownCodeIsGeneric(t *testing.T) {
	code, msg := invitePageErrorFor(status.Error(codes.DataLoss, "x"))
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Contains(t, msg, "Fehler aufgetreten")
}
