package auth_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type recordedInvite struct{ to, name, link string }

// recordingMailer is the fake behind the invite routes' mail: it keeps what
// would have gone out, so the link in it can be followed like a recipient would.
type recordingMailer struct{ invites []recordedInvite }

func (m *recordingMailer) SendPasswordResetEmail(context.Context, string, string, string) error {
	return nil
}

func (m *recordingMailer) SendInvitationEmail(_ context.Context, to, name, link string) error {
	m.invites = append(m.invites, recordedInvite{to, name, link})
	return nil
}

// tokenFromLink reads the token back out of a mailed /invite link.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	u, err := url.Parse(link)
	require.NoError(t, err)
	require.Equal(t, "/invite", u.Path)
	return u.Query().Get("token")
}

// The admin invite and its resend each mail exactly one link; the mailed link
// is the only way in, the resend kills the first one, and the link works once.
func TestInviteAdminUser_DB_MailsWorkingSingleUseLink(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	mailer := &recordingMailer{}
	svc.SetMailer(mailer)
	svc.SetResetBaseURL("https://app.test.local/reset-password")
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "mail-actor@test.local", true)
	memberPreset := adminRosterPresetID(t, pool, "member")

	row, token, err := svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"mail-target@test.local", "Mira", "Mail", []uuid.UUID{memberPreset})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", row.ID) })

	require.Len(t, mailer.invites, 1, "one invite, one mail")
	assert.Equal(t, "mail-target@test.local", mailer.invites[0].to)
	assert.Equal(t, "Mira Mail", mailer.invites[0].name)
	assert.Equal(t, token, tokenFromLink(t, mailer.invites[0].link))

	_, _, err = svc.ResendAdminUserInvite(ctx, adminWriteTenant, row.ID)
	require.NoError(t, err)
	require.Len(t, mailer.invites, 2, "a resend mails again")
	resentToken := tokenFromLink(t, mailer.invites[1].link)
	require.NotEqual(t, token, resentToken)

	_, err = svc.PreviewInvitation(context.Background(), token)
	require.ErrorIs(t, err, auth.ErrInvitationNotFound, "the first mail's link is dead after a resend")

	preview, err := svc.PreviewInvitation(context.Background(), resentToken)
	require.NoError(t, err)
	assert.Equal(t, "Mira", preview.FirstName)

	user, _, err := svc.AcceptInvitation(context.Background(), resentToken, "sufficiently-long-pw", "Mira", "Mail")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user.ID) })
	assert.Equal(t, adminWriteTenant, user.TenantID)

	_, _, err = svc.AcceptInvitation(context.Background(), resentToken, "sufficiently-long-pw", "Mira", "Mail")
	require.ErrorIs(t, err, auth.ErrInvitationAlreadyUsed, "an accepted link is spent")
	_, err = svc.PreviewInvitation(context.Background(), resentToken)
	require.ErrorIs(t, err, auth.ErrInvitationAlreadyUsed)
}

// The legacy /invitations route goes through the same createInvitation body,
// so it mails too.
func TestCreateInvitation_DB_Mails(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	mailer := &recordingMailer{}
	svc.SetMailer(mailer)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "legacy-mail-actor@test.local", true)
	inv, token, err := svc.CreateInvitation(ctx, adminWriteTenant, "legacy-mail@test.local", "member", actorID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", inv.ID) })

	require.Len(t, mailer.invites, 1)
	assert.Equal(t, token, tokenFromLink(t, mailer.invites[0].link))
}
