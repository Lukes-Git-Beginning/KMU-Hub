package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Invitations used to produce a token and nothing else: no mail left the
// building, while the desktop told the admin it had been sent. These pin the
// mail, the link it carries, and that a failed send neither loses the
// invitation nor goes unnoticed.

func newInviteMailService(t *testing.T) (*Service, *mockRepository, *mockMailer, *mockPasswordValidator) {
	t.Helper()
	svc, repo, m := newResetTestService()
	v := &mockPasswordValidator{valid: true}
	svc.SetPasswordValidator(v)
	return svc, repo, m, v
}

// captureLogs routes slog into a buffer for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestCreateInvitation_MailsOneInviteLink(t *testing.T) {
	svc, _, m, _ := newInviteMailService(t)

	_, token, err := svc.CreateInvitation(context.Background(), uuid.New(), "Neu@Example.com", "member", uuid.New())
	require.NoError(t, err)

	require.Len(t, m.invites, 1, "exactly one mail per invitation")
	sent := m.invites[0]
	assert.Equal(t, "neu@example.com", sent.to)
	assert.Equal(t, "neu@example.com", sent.name, "no name on the invitation: the address greets")
	assert.Equal(t, "https://app.test.local/invite?token="+url.QueryEscape(token), sent.link)
	assert.False(t, m.called, "an invitation is not a password reset")
}

func TestCreateInvitation_MailFailureKeepsInvitationAndIsLogged(t *testing.T) {
	svc, repo, m, _ := newInviteMailService(t)
	m.inviteErr = errors.New("smtp: connection refused")
	logs := captureLogs(t)

	inv, token, err := svc.CreateInvitation(context.Background(), uuid.New(), "fail@example.com", "member", uuid.New())
	require.NoError(t, err, "a failed send must not fail the invitation")
	require.Contains(t, repo.invitations, inv.ID, "the invitation stays, a resend mails it again")

	out := logs.String()
	assert.Contains(t, out, "failed to send invitation email")
	assert.Contains(t, out, inv.ID.String())
	assert.NotContains(t, out, token, "the token is the credential and never logged")
}

func TestCreateInvitation_WithoutMailerStillInvites(t *testing.T) {
	svc, repo, _, _ := newInviteMailService(t)
	svc.SetMailer(nil)
	logs := captureLogs(t)

	inv, _, err := svc.CreateInvitation(context.Background(), uuid.New(), "nomail@example.com", "member", uuid.New())
	require.NoError(t, err)
	require.Contains(t, repo.invitations, inv.ID)
	assert.Contains(t, logs.String(), "invitation email not sent")
}

func TestInviteLink_DerivesFromResetBase(t *testing.T) {
	cases := []struct{ base, want string }{
		{"https://app.zentria.tech/reset-password", "https://app.zentria.tech/invite?token=abc"},
		{"http://localhost:8080/reset-password?x=1#frag", "http://localhost:8080/invite?token=abc"},
		{"", "https://app.zentria.tech/invite?token=abc"},
		{"not a url", "https://app.zentria.tech/invite?token=abc"},
	}
	for _, tc := range cases {
		svc := NewService(newMockRepository(), nil)
		svc.SetResetBaseURL(tc.base)
		assert.Equal(t, tc.want, svc.inviteLink("abc"), "base %q", tc.base)
	}
}

func seedPreviewInvitation(repo *mockRepository, token string, expires time.Time, accepted bool) *models.Invitation {
	inv := &models.Invitation{
		ID: uuid.New(), TenantID: uuid.New(), Email: "p@example.com",
		FirstName: "Pia", LastName: "Vorschau", Role: "member",
		TokenHash: HashToken(token), ExpiresAt: expires, CreatedAt: time.Now(),
	}
	if accepted {
		now := time.Now()
		inv.AcceptedAt = &now
	}
	repo.invitations[inv.ID] = inv
	repo.invByToken[inv.TokenHash] = inv
	return inv
}

func TestPreviewInvitation_States(t *testing.T) {
	svc, repo, _, _ := newInviteMailService(t)
	seedPreviewInvitation(repo, "live", time.Now().Add(time.Hour), false)
	seedPreviewInvitation(repo, "old", time.Now().Add(-time.Hour), false)
	seedPreviewInvitation(repo, "spent", time.Now().Add(time.Hour), true)

	inv, err := svc.PreviewInvitation(context.Background(), "live")
	require.NoError(t, err)
	assert.Equal(t, "Pia", inv.FirstName)
	assert.Equal(t, "Vorschau", inv.LastName)

	_, err = svc.PreviewInvitation(context.Background(), "old")
	assert.ErrorIs(t, err, ErrInvitationExpired)
	_, err = svc.PreviewInvitation(context.Background(), "spent")
	assert.ErrorIs(t, err, ErrInvitationAlreadyUsed)
	_, err = svc.PreviewInvitation(context.Background(), "unknown")
	assert.ErrorIs(t, err, ErrInvitationNotFound)
	assert.Len(t, repo.users, 0, "a preview never creates anything")
}

// The tenant's password policy used to be unreachable from the accept path,
// and on the reset path it always failed for want of a tenant in ctx. The
// validator now gets the invitation's tenant explicitly.
func TestAcceptInvitation_EnforcesTenantPasswordPolicy(t *testing.T) {
	svc, repo, _, v := newInviteMailService(t)
	inv := seedPreviewInvitation(repo, "weak", time.Now().Add(time.Hour), false)
	v.valid = false
	v.reasons = []string{"password must be at least 12 characters"}

	_, _, err := svc.AcceptInvitation(context.Background(), "weak", "kurz1234", "A", "B")
	require.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Equal(t, inv.TenantID, v.gotTenant, "the policy is the invitation's tenant's")
	assert.Nil(t, repo.invitations[inv.ID].AcceptedAt, "a rejected password does not spend the link")
	assert.Len(t, repo.users, 0)

	v.valid = true
	_, _, err = svc.AcceptInvitation(context.Background(), "weak", "a-much-longer-passphrase", "A", "B")
	require.NoError(t, err)
}

func TestResetPassword_ValidatesAgainstTheTokensTenant(t *testing.T) {
	svc, repo, m := newResetTestService()
	v := &mockPasswordValidator{valid: true}
	svc.SetPasswordValidator(v)
	user := seedResetUser(repo)

	require.NoError(t, svc.RequestPasswordReset(context.Background(), user.Email))
	plain := m.lastLink[len(m.lastLink)-64:]
	require.NoError(t, svc.ResetPassword(context.Background(), plain, "a-much-longer-passphrase"))
	assert.Equal(t, user.TenantID, v.gotTenant)
}
