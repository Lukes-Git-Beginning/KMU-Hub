package gateway

import (
	_ "embed"
	"html/template"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// The invitation mail links to /invite?token=… on the host of
// PASSWORD_RESET_BASE_URL (auth.Service.inviteLink). A link from a mail cannot
// open the Electron app, so the account is created here, server-rendered and
// script-free like the reset page next to it.
//
// Unlike the reset page, GET does look the token up: the form prefills the name
// the admin typed, and a dead invitation should say so before anyone picks a
// password. Probing is no concern — the token is 256 random bits behind the
// public limiter — and the answer carries nothing the mail did not.
//
//go:embed invite_page.html
var invitePageHTML string

var invitePageTmpl = template.Must(template.New("invite").Parse(invitePageHTML))

// invitePageData is the template payload. State reuses the reset page's three
// states; Token rides in a hidden field so the POST URL carries no credential.
type invitePageData struct {
	State     string
	Token     string
	Email     string
	FirstName string
	LastName  string
	Message   string
}

const invitePageUnavailable = "Der Dienst ist gerade nicht erreichbar. Bitte versuche es in einigen Minuten erneut."

// HandleInvitePage renders the accept form for a link opened from the mail.
func (a *AuthRoutes) HandleInvitePage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		renderTokenPage(w, http.StatusBadRequest, invitePageTmpl, invitePageData{
			State:   resetStateInvalid,
			Message: "Dieser Link enthält kein Token.",
		})
		return
	}

	client, err := a.getAuthClient()
	if err != nil {
		renderTokenPage(w, http.StatusServiceUnavailable, invitePageTmpl, invitePageData{
			State: resetStateForm, Token: token, Message: invitePageUnavailable,
		})
		return
	}

	preview, err := client.GetInvitationPreview(r.Context(), &authv1.GetInvitationPreviewRequest{Token: token})
	if err != nil {
		httpStatus, msg := invitePageErrorFor(err)
		data := invitePageData{State: resetStateForm, Token: token, Message: msg}
		// Only a dead token is terminal; after an outage the link is still good.
		if httpStatus == http.StatusNotFound || httpStatus == http.StatusGone {
			data = invitePageData{State: resetStateInvalid, Message: msg}
		}
		renderTokenPage(w, httpStatus, invitePageTmpl, data)
		return
	}

	renderTokenPage(w, http.StatusOK, invitePageTmpl, invitePageData{
		State:     resetStateForm,
		Token:     token,
		Email:     preview.GetEmail(),
		FirstName: preview.GetFirstName(),
		LastName:  preview.GetLastName(),
	})
}

// HandleInvitePageSubmit creates the account through the same gRPC call the
// JSON route uses. The returned token pair is dropped on purpose: the page
// sets no session, the person signs in in the app.
func (a *AuthRoutes) HandleInvitePageSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderTokenPage(w, http.StatusBadRequest, invitePageTmpl, invitePageData{
			State:   resetStateInvalid,
			Message: "Die Anfrage konnte nicht gelesen werden.",
		})
		return
	}

	data := invitePageData{
		State:     resetStateForm,
		Token:     r.PostFormValue("token"),
		FirstName: strings.TrimSpace(r.PostFormValue("first_name")),
		LastName:  strings.TrimSpace(r.PostFormValue("last_name")),
	}
	password := r.PostFormValue("password")

	if data.Token == "" {
		renderTokenPage(w, http.StatusBadRequest, invitePageTmpl, invitePageData{
			State:   resetStateInvalid,
			Message: "Dieser Link enthält kein Token.",
		})
		return
	}

	// Recoverable input problems keep the form, token and names intact.
	switch {
	case data.FirstName == "" || data.LastName == "":
		data.Message = "Bitte gib Vor- und Nachnamen an."
	case len(data.FirstName) > 100 || len(data.LastName) > 100:
		data.Message = "Vor- und Nachname dürfen höchstens 100 Zeichen lang sein."
	case password != r.PostFormValue("confirm_password"):
		data.Message = "Die beiden Passwörter stimmen nicht überein."
	case len(password) < 8:
		data.Message = "Das Passwort muss mindestens 8 Zeichen lang sein."
	}
	if data.Message != "" {
		renderTokenPage(w, http.StatusBadRequest, invitePageTmpl, data)
		return
	}

	client, err := a.getAuthClient()
	if err != nil {
		data.Message = invitePageUnavailable
		renderTokenPage(w, http.StatusServiceUnavailable, invitePageTmpl, data)
		return
	}

	resp, err := client.AcceptInvitation(r.Context(), &authv1.AcceptInvitationRequest{
		Token:     data.Token,
		Password:  password,
		FirstName: data.FirstName,
		LastName:  data.LastName,
	})
	if err != nil {
		httpStatus, msg := invitePageErrorFor(err)
		data.Message = msg
		if httpStatus == http.StatusNotFound || httpStatus == http.StatusGone {
			// Dead or spent token: nothing on this form can fix it. Every
			// other failure (weak password, full seat plan, outage) keeps the
			// form, because the same link works once that is resolved.
			data = invitePageData{State: resetStateInvalid, Message: msg}
		}
		renderTokenPage(w, httpStatus, invitePageTmpl, data)
		return
	}

	renderTokenPage(w, http.StatusOK, invitePageTmpl, invitePageData{
		State: resetStateDone,
		Email: resp.GetUser().GetEmail(),
	})
}

// invitePageErrorFor turns a gRPC status into an HTTP status and a German
// sentence. mapError (internal/server/grpc.go) answers an unknown token with
// NotFound, an expired or accepted one and a full seat plan with
// FailedPrecondition, and a password the tenant policy rejects with
// InvalidArgument.
func invitePageErrorFor(err error) (int, string) {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.NotFound:
		return http.StatusNotFound, "Diese Einladung ist ungültig."
	case codes.FailedPrecondition:
		if strings.Contains(st.Message(), "seat") {
			return http.StatusConflict, "Für dein Unternehmen ist gerade kein Platz frei. Bitte wende dich an die Person, die dich eingeladen hat."
		}
		return http.StatusGone, "Diese Einladung ist abgelaufen oder wurde bereits angenommen."
	case codes.InvalidArgument:
		return http.StatusBadRequest, "Das Passwort erfüllt die Passwort-Richtlinie deines Unternehmens nicht. Wähle ein längeres Passwort — am besten mehrere Wörter."
	case codes.Unavailable:
		return http.StatusServiceUnavailable, invitePageUnavailable
	default:
		return http.StatusServiceUnavailable, "Beim Anlegen des Kontos ist ein Fehler aufgetreten. Bitte versuche es erneut."
	}
}
