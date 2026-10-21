package auth

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The issuer is the label the authenticator app shows for a freshly enrolled account.
func TestSetup2FA_OTPURLNamesCosmiAsIssuer(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "issuer@example.com", "pw", true)

	setup, err := svc.Setup2FA(context.Background(), user.ID)
	require.NoError(t, err)

	u, err := url.Parse(setup.OTPURL)
	require.NoError(t, err)
	assert.Equal(t, "Cosmi", u.Query().Get("issuer"))
	assert.Contains(t, u.Path, "Cosmi:", "label prefix carries the issuer too")
}

func TestParseUserAgent_ElectronIsCosmiDesktop(t *testing.T) {
	name, kind := parseUserAgent("Mozilla/5.0 Electron/30.0.0 Safari/537.36")
	assert.Equal(t, "Cosmi Desktop", name)
	assert.Equal(t, "desktop", kind)
}
