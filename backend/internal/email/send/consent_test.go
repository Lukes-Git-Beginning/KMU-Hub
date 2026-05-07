package send

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
)

// stubAssertRepo implements consent.AssertRepo for send-package tests.
type stubAssertRepo struct {
	active    bool
	activeErr error
	hasEmail  bool
}

func (s *stubAssertRepo) ActiveConsent(_ context.Context, _ uuid.UUID, _ consent.Channel) (bool, error) {
	return s.active, s.activeErr
}

func (s *stubAssertRepo) ContactHasEmail(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.hasEmail, nil
}

// errAccountProvider always returns an error, simulating a missing SMTP config.
// Used to prove that the consent check passed (errors past that gate are credential errors).
type errAccountProvider struct{}

func (e *errAccountProvider) GetDecryptedCredentials(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*Credentials, error) {
	return nil, errors.New("no SMTP credentials configured")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestSend_BlockedByConsent verifies that Send returns ErrNoConsent and never
// reaches the SMTP/message-store layer when the asserter blocks the request.
func TestSend_BlockedByConsent(t *testing.T) {
	repo := &stubAssertRepo{
		active:   false, // no active consent
		hasEmail: true,
	}
	asserter := consent.NewAsserter(repo, nil)

	creator := &mockMessageCreator{}
	svc := NewServiceWithConsent(nil, creator, nil, asserter)

	contactID := uuid.New()
	input := SendInput{
		AccountID: uuid.New(),
		UserID:    uuid.New(),
		ContactID: &contactID,
		From:      EmailAddress{Email: "from@example.com"},
		To:        []EmailAddress{{Email: "to@example.com"}},
		Subject:   "Test",
	}

	_, err := svc.Send(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, consent.ErrNoConsent)
	// SMTP was never reached — no message was stored.
	assert.Empty(t, creator.messages, "message store must not be called when consent is missing")
}

// TestSend_AllowedByConsent verifies that Send proceeds past the consent gate
// (and fails on credential fetch, not on consent) when active consent is present.
func TestSend_AllowedByConsent(t *testing.T) {
	repo := &stubAssertRepo{
		active:   true, // active consent present
		hasEmail: true,
	}
	asserter := consent.NewAsserter(repo, nil)

	creator := &mockMessageCreator{}
	// errAccountProvider returns an error so Send fails after the consent check.
	svc := NewServiceWithConsent(&errAccountProvider{}, creator, nil, asserter)

	contactID := uuid.New()
	input := SendInput{
		AccountID: uuid.New(),
		UserID:    uuid.New(),
		ContactID: &contactID,
		From:      EmailAddress{Email: "from@example.com"},
		To:        []EmailAddress{{Email: "to@example.com"}},
		Subject:   "Test",
	}

	_, err := svc.Send(context.Background(), input)

	// Error should be about credentials, NOT about consent.
	require.Error(t, err)
	assert.NotErrorIs(t, err, consent.ErrNoConsent, "consent check must pass when active consent exists")
}

// TestSend_NoContactID_SkipsConsentCheck verifies that emails without a
// ContactID bypass the consent gate (system / transactional emails).
func TestSend_NoContactID_SkipsConsentCheck(t *testing.T) {
	repo := &stubAssertRepo{
		active:   false, // would block if checked
		hasEmail: true,
	}
	asserter := consent.NewAsserter(repo, nil)

	creator := &mockMessageCreator{}
	svc := NewServiceWithConsent(&errAccountProvider{}, creator, nil, asserter)

	input := SendInput{
		AccountID: uuid.New(),
		// ContactID intentionally omitted — system/transactional email
		From:    EmailAddress{Email: "system@example.com"},
		To:      []EmailAddress{{Email: "user@example.com"}},
		Subject: "Password reset",
	}

	_, err := svc.Send(context.Background(), input)

	// Should fail on nil accountProvider (credentials), not on consent.
	require.Error(t, err)
	assert.NotErrorIs(t, err, consent.ErrNoConsent)
}

// TestSend_RepoError_Propagates verifies that a repo error during the consent
// check surfaces to the caller without reaching the SMTP layer.
func TestSend_RepoError_Propagates(t *testing.T) {
	dbErr := errors.New("database unavailable")
	repo := &stubAssertRepo{
		activeErr: dbErr,
		hasEmail:  true,
	}
	asserter := consent.NewAsserter(repo, nil)

	creator := &mockMessageCreator{}
	svc := NewServiceWithConsent(&errAccountProvider{}, creator, nil, asserter)

	contactID := uuid.New()
	input := SendInput{
		AccountID: uuid.New(),
		ContactID: &contactID,
		From:      EmailAddress{Email: "from@example.com"},
		To:        []EmailAddress{{Email: "to@example.com"}},
		Subject:   "Test",
	}

	_, err := svc.Send(context.Background(), input)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}
