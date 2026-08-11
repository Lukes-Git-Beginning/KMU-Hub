package gateway

import (
	_ "embed"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// The password-reset mail links to PASSWORD_RESET_BASE_URL, which points at
// this gateway (Caddy proxies app.zentria.tech wholesale to :8080). Without
// these two routes that link 404s and a user who forgot their password has no
// way back into the product — the desktop app only handles the in-app hash
// route `#/reset-password`, which a mail link never reaches.
//
// This is deliberately a server-rendered form rather than the Vite SPA under
// `guest-chat/`: a single embedded page needs no build step, ships with the
// gateway binary, and lets the CSP below forbid scripts outright.
//
//go:embed reset_password_page.html
var resetPasswordPageHTML string

var resetPasswordPageTmpl = template.Must(
	template.New("reset-password").Parse(resetPasswordPageHTML),
)

// resetPageState selects which branch of the template renders.
const (
	resetStateForm    = "form"    // show the form (fresh, or after a recoverable error)
	resetStateDone    = "done"    // password changed
	resetStateInvalid = "invalid" // token missing/unknown/expired/used — no retry possible
)

// resetPageData is the template payload. Token is echoed into a hidden field so
// the POST does not have to carry it in the URL; html/template escapes it.
type resetPageData struct {
	State   string
	Token   string
	Message string
}

// RegisterPublicRoutes registers the unauthenticated password-reset page. The
// caller passes the strict per-IP public limiter: the token in the link is the
// whole credential, so an unthrottled POST loop would be a guessing run.
func (a *AuthRoutes) RegisterPublicRoutes(r chi.Router, publicRateLimit func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(publicRateLimit)
		r.Get("/reset-password", a.HandleResetPasswordPage)
		r.Post("/reset-password", a.HandleResetPasswordPageSubmit)
	})
}

// renderResetPage writes the page with headers that keep the token out of
// referers, proxies and search indexes, and forbid scripts entirely.
func renderResetPage(w http.ResponseWriter, code int, data resetPageData) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	// No scripts at all: the page is a plain form, so the strictest policy
	// that still renders is also the correct one.
	h.Set("Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Robots-Tag", "noindex, nofollow")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	// The URL carries the reset token — never let a shared cache keep it.
	h.Set("Cache-Control", "no-store, max-age=0")

	w.WriteHeader(code)
	if err := resetPasswordPageTmpl.Execute(w, data); err != nil {
		// Header and status are already on the wire; log rather than
		// double-write a broken response.
		slog.Error("reset password page render failed", "error", err)
	}
}

// HandleResetPasswordPage renders the form for a link opened from the mail.
// The token is not verified here: doing so would need a second round trip and
// would let an attacker probe token validity without ever submitting one.
func (a *AuthRoutes) HandleResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		renderResetPage(w, http.StatusBadRequest, resetPageData{
			State:   resetStateInvalid,
			Message: "Dieser Link enthält kein Token.",
		})
		return
	}
	renderResetPage(w, http.StatusOK, resetPageData{State: resetStateForm, Token: token})
}

// HandleResetPasswordPageSubmit accepts the form post and performs the reset
// through the same gRPC call the JSON API uses.
func (a *AuthRoutes) HandleResetPasswordPageSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderResetPage(w, http.StatusBadRequest, resetPageData{
			State:   resetStateInvalid,
			Message: "Die Anfrage konnte nicht gelesen werden.",
		})
		return
	}

	token := r.PostFormValue("token")
	newPassword := r.PostFormValue("new_password")
	confirm := r.PostFormValue("confirm_password")

	if token == "" {
		renderResetPage(w, http.StatusBadRequest, resetPageData{
			State:   resetStateInvalid,
			Message: "Dieser Link enthält kein Token.",
		})
		return
	}

	// Recoverable input problems keep the user on the form with the token
	// intact, so a typo does not cost them the link.
	switch {
	case newPassword != confirm:
		renderResetPage(w, http.StatusBadRequest, resetPageData{
			State:   resetStateForm,
			Token:   token,
			Message: "Die beiden Passwörter stimmen nicht überein.",
		})
		return
	case len(newPassword) < 8:
		renderResetPage(w, http.StatusBadRequest, resetPageData{
			State:   resetStateForm,
			Token:   token,
			Message: "Das Passwort muss mindestens 8 Zeichen lang sein.",
		})
		return
	}

	client, err := a.getAuthClient()
	if err != nil {
		renderResetPage(w, http.StatusServiceUnavailable, resetPageData{
			State:   resetStateForm,
			Token:   token,
			Message: "Der Dienst ist gerade nicht erreichbar. Bitte versuche es in einigen Minuten erneut.",
		})
		return
	}

	if _, err := client.ResetPassword(r.Context(), &authv1.ResetPasswordRequest{
		Token:       token,
		NewPassword: newPassword,
	}); err != nil {
		code, msg := resetPageErrorFor(err)
		state := resetStateInvalid
		httpStatus := http.StatusBadRequest
		// A weak password is the one failure the user can fix right away.
		if code == codes.InvalidArgument {
			state = resetStateForm
		}
		if code == codes.Unavailable || code == codes.Internal {
			state = resetStateForm
			httpStatus = http.StatusServiceUnavailable
		}
		data := resetPageData{State: state, Message: msg}
		if state == resetStateForm {
			data.Token = token
		}
		renderResetPage(w, httpStatus, data)
		return
	}

	renderResetPage(w, http.StatusOK, resetPageData{State: resetStateDone})
}

// resetPageErrorFor turns a gRPC status into a German, user-facing sentence.
// The service maps invalid/expired/used tokens to NotFound and
// FailedPrecondition and a rejected password to InvalidArgument (see
// mapError in internal/server/grpc.go); the strength reasons themselves are
// only logged, never returned, so the wording stays general.
func resetPageErrorFor(err error) (codes.Code, string) {
	st, ok := status.FromError(err)
	if !ok {
		return codes.Internal, "Beim Zurücksetzen ist ein Fehler aufgetreten."
	}
	switch st.Code() {
	case codes.NotFound:
		return st.Code(), "Dieser Link ist ungültig."
	case codes.FailedPrecondition:
		return st.Code(), "Dieser Link ist abgelaufen oder wurde bereits verwendet."
	case codes.InvalidArgument:
		return st.Code(), "Das Passwort erfüllt die Sicherheitsanforderungen nicht. Wähle ein längeres oder komplexeres Passwort."
	case codes.Unavailable:
		return st.Code(), "Der Dienst ist gerade nicht erreichbar. Bitte versuche es in einigen Minuten erneut."
	default:
		return codes.Internal, "Beim Zurücksetzen ist ein Fehler aufgetreten."
	}
}
