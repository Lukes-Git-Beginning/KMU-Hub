package dialer

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
)

// dialerStubAssertRepo implements consent.AssertRepo for dialer tests.
type dialerStubAssertRepo struct {
	active bool
}

func (s *dialerStubAssertRepo) ActiveConsent(_ context.Context, _ uuid.UUID, _ consent.Channel) (bool, error) {
	return s.active, nil
}

func (s *dialerStubAssertRepo) ContactHasEmail(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil // phone channel never calls this
}

// newTestHarnessWithConsent creates a test harness with a consent asserter.
func newTestHarnessWithConsent(activeConsent bool) *testHarness {
	cr := newMockCampaignRepo()
	cl := newMockCallRepo()
	or := newMockOutcomeRepo()
	br := newMockCRMBridge()
	repo := &dialerStubAssertRepo{active: activeConsent}
	asserter := consent.NewAsserter(repo, nil)
	svc := NewServiceWithConsent(cr, cl, or, &mockAgentStatusRepo{}, nil, br, nil, asserter)
	return &testHarness{svc: svc, campaigns: cr, calls: cl, outcomes: or, bridge: br}
}

// TestInitiateDialerCall_BlockedByConsent verifies that InitiateDialerCall
// returns ErrNoConsent and never creates a call session when consent is absent.
func TestInitiateDialerCall_BlockedByConsent(t *testing.T) {
	h := newTestHarnessWithConsent(false) // no active consent

	campaignID := uuid.New()
	ccID := uuid.New()
	realContactID := uuid.New()

	// Register the campaign contact so the resolver succeeds.
	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: campaignID,
		ContactID:  realContactID,
	}

	_, err := h.svc.InitiateDialerCall(context.Background(), ccID, uuid.New(), nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, consent.ErrNoConsent)
	// No call session must have been created.
	assert.Empty(t, h.calls.sessions, "call session must not be created without consent")
}

// TestInitiateDialerCall_AllowedByConsent verifies that InitiateDialerCall
// creates a session when active consent is present.
func TestInitiateDialerCall_AllowedByConsent(t *testing.T) {
	h := newTestHarnessWithConsent(true) // active consent present

	campaignID := uuid.New()
	ccID := uuid.New()
	realContactID := uuid.New()

	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: campaignID,
		ContactID:  realContactID,
	}

	session, err := h.svc.InitiateDialerCall(context.Background(), ccID, uuid.New(), nil)

	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, ccID, session.CampaignContactID)
	assert.Len(t, h.calls.sessions, 1, "one session must be created when consent is active")
}

// TestInitiateDialerCall_NoAsserter_Allowed verifies the nil-safe path:
// when no asserter is wired in (legacy / test path), the call is allowed.
func TestInitiateDialerCall_NoAsserter_Allowed(t *testing.T) {
	h := newTestHarness() // no asserter injected

	campaignID := uuid.New()
	ccID := uuid.New()

	h.campaigns.contacts[ccID] = &CampaignContact{
		ID:         ccID,
		CampaignID: campaignID,
		ContactID:  uuid.New(),
	}

	session, err := h.svc.InitiateDialerCall(context.Background(), ccID, uuid.New(), nil)

	require.NoError(t, err)
	require.NotNil(t, session)
}
