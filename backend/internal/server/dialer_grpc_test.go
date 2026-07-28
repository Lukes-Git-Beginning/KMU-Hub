package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/dialer"
	"github.com/kmuhub/kmuhub/internal/middleware"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

// ============================================================================
// mapDialerError table-driven test
// ============================================================================

func TestMapDialerError_AllSentinels(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantNil  bool
	}{
		{"nil", nil, 0, true},
		{"ErrCampaignNotFound", dialer.ErrCampaignNotFound, codes.NotFound, false},
		{"ErrCallSessionNotFound", dialer.ErrCallSessionNotFound, codes.NotFound, false},
		{"ErrOutcomeNotFound", dialer.ErrOutcomeNotFound, codes.NotFound, false},
		{"ErrCampaignNotDraft", dialer.ErrCampaignNotDraft, codes.FailedPrecondition, false},
		{"ErrCampaignNotActive", dialer.ErrCampaignNotActive, codes.FailedPrecondition, false},
		{"ErrInvalidStatusTransition", dialer.ErrInvalidStatusTransition, codes.FailedPrecondition, false},
		{"ErrNoContactsAvailable", dialer.ErrNoContactsAvailable, codes.NotFound, false},
		{"ErrContactAlreadyInCampaign", dialer.ErrContactAlreadyInCampaign, codes.AlreadyExists, false},
		{"ErrAgentNotAvailable", dialer.ErrAgentNotAvailable, codes.FailedPrecondition, false},
		{"ErrCampaignHasNoContacts", dialer.ErrCampaignHasNoContacts, codes.FailedPrecondition, false},
		// Bonus: consent.ErrNoConsent must map to PermissionDenied (not Internal)
		{"consent.ErrNoConsent", consent.ErrNoConsent, codes.PermissionDenied, false},
		// Unknown error must fall back to Internal.
		{"unknown error", errors.New("some unexpected error"), codes.Internal, false},
		// Wrapped sentinels must still be recognised.
		{"wrapped ErrCampaignNotFound", errors.Join(errors.New("outer"), dialer.ErrCampaignNotFound), codes.NotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapDialerError(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("mapDialerError(nil) = %v, want nil", got)
				}
				return
			}
			requireGRPCCode(t, got, tt.wantCode)
		})
	}
}

// ============================================================================
// Minimal dialer repository stubs for server-level handler tests
// These are lightweight stand-ins that fulfil the dialer interfaces without
// requiring a real database or Redis connection.
// ============================================================================

type stubCampaignRepo struct {
	campaigns map[uuid.UUID]*dialer.Campaign
	contacts  map[uuid.UUID]*dialer.CampaignContact
}

func newStubCampaignRepo() *stubCampaignRepo {
	return &stubCampaignRepo{
		campaigns: make(map[uuid.UUID]*dialer.Campaign),
		contacts:  make(map[uuid.UUID]*dialer.CampaignContact),
	}
}

func (r *stubCampaignRepo) Create(_ context.Context, c *dialer.Campaign) error { r.campaigns[c.ID] = c; return nil }
func (r *stubCampaignRepo) GetByID(_ context.Context, id uuid.UUID) (*dialer.Campaign, error) {
	if c, ok := r.campaigns[id]; ok { return c, nil }
	return nil, dialer.ErrCampaignNotFound
}
func (r *stubCampaignRepo) GetByIDForTenant(_ context.Context, id, _ uuid.UUID) (*dialer.Campaign, error) {
	return r.GetByID(context.Background(), id)
}
func (r *stubCampaignRepo) List(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*dialer.Campaign, int, error) {
	return nil, 0, nil
}
func (r *stubCampaignRepo) Update(_ context.Context, _ *dialer.Campaign, _ uuid.UUID) error { return nil }
func (r *stubCampaignRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, _ string) error  { return nil }
func (r *stubCampaignRepo) Delete(_ context.Context, _, _ uuid.UUID) error                  { return nil }
func (r *stubCampaignRepo) AddContacts(_ context.Context, _ uuid.UUID, contacts []dialer.CampaignContact) (int, int, error) {
	return len(contacts), 0, nil
}
func (r *stubCampaignRepo) GetNextPendingContact(_ context.Context, _ uuid.UUID) (*dialer.CampaignContact, error) {
	return nil, dialer.ErrNoContactsAvailable
}
func (r *stubCampaignRepo) ListContacts(_ context.Context, _ uuid.UUID, _ *string, _, _ int) ([]*dialer.CampaignContact, int, error) {
	return nil, 0, nil
}
func (r *stubCampaignRepo) UpdateContactStatus(_ context.Context, _, _ uuid.UUID, _ string, _ *uuid.UUID) error { return nil }
func (r *stubCampaignRepo) SetContactCallback(_ context.Context, _, _ uuid.UUID, _ time.Time) error            { return nil }
func (r *stubCampaignRepo) SkipContact(_ context.Context, _, _ uuid.UUID) error                                { return nil }
func (r *stubCampaignRepo) RequeueContact(_ context.Context, _, _ uuid.UUID) error                             { return nil }
func (r *stubCampaignRepo) IncrementContactCallCount(_ context.Context, _, _ uuid.UUID) error                  { return nil }
func (r *stubCampaignRepo) GetCampaignContactByID(_ context.Context, id, _ uuid.UUID) (*dialer.CampaignContact, error) {
	if cc, ok := r.contacts[id]; ok { return cc, nil }
	return nil, dialer.ErrCampaignContactNotFound
}
func (r *stubCampaignRepo) GetCampaignStats(_ context.Context, _ uuid.UUID) (*dialer.CampaignStats, error) {
	return &dialer.CampaignStats{PendingContacts: 1}, nil
}
func (r *stubCampaignRepo) GetAgentStats(_ context.Context, _ uuid.UUID) (*dialer.AgentStats, error) {
	return &dialer.AgentStats{}, nil
}
func (r *stubCampaignRepo) UpdateCampaignCounts(_ context.Context, _ uuid.UUID) error { return nil }

type stubCallRepo struct {
	sessions map[uuid.UUID]*dialer.CallSession
	events   []*dialer.CallEvent
}

func newStubCallRepo() *stubCallRepo {
	return &stubCallRepo{sessions: make(map[uuid.UUID]*dialer.CallSession)}
}

func (r *stubCallRepo) CreateSession(_ context.Context, s *dialer.CallSession) error {
	r.sessions[s.ID] = s; return nil
}
func (r *stubCallRepo) GetSessionByID(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*dialer.CallSession, error) {
	s, ok := r.sessions[id]
	if !ok { return nil, dialer.ErrCallSessionNotFound }
	if tenantID != uuid.Nil && s.TenantID != uuid.Nil && s.TenantID != tenantID {
		return nil, dialer.ErrCallSessionNotFound
	}
	copy := *s
	return &copy, nil
}
func (r *stubCallRepo) UpdateSession(_ context.Context, s *dialer.CallSession) error {
	r.sessions[s.ID] = s; return nil
}
func (r *stubCallRepo) AppendEvent(_ context.Context, e *dialer.CallEvent) error {
	r.events = append(r.events, e); return nil
}
func (r *stubCallRepo) UpdateSessionWithEventAndContact(_ context.Context, s *dialer.CallSession, e *dialer.CallEvent, _ uuid.UUID, _ string, _ *uuid.UUID, _ *time.Time) error {
	if _, ok := r.sessions[s.ID]; !ok { return dialer.ErrCallSessionNotFound }
	r.sessions[s.ID] = s
	r.events = append(r.events, e)
	return nil
}
func (r *stubCallRepo) ListEventsBySession(_ context.Context, sessionID uuid.UUID) ([]*dialer.CallEvent, error) {
	var result []*dialer.CallEvent
	for _, e := range r.events {
		if e.DialerCallSessionID == sessionID { result = append(result, e) }
	}
	return result, nil
}
func (r *stubCallRepo) GetRecentCallsForTenant(_ context.Context, _ uuid.UUID, _ int) ([]dialer.RecentCallRow, error) {
	return nil, nil
}
func (r *stubCallRepo) ListCallsByContact(_ context.Context, _ uuid.UUID) ([]dialer.ContactCallRow, error) {
	return nil, nil
}
func (r *stubCallRepo) GetTenantCallsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *stubCallRepo) GetTenantAppointmentsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *stubCallRepo) GetAgentCallsTodayCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *stubCallRepo) GetAgentAvgDurationToday(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}

type stubOutcomeRepo struct {
	outcomes map[uuid.UUID]*dialer.CallOutcome
}

func newStubOutcomeRepo() *stubOutcomeRepo {
	return &stubOutcomeRepo{outcomes: make(map[uuid.UUID]*dialer.CallOutcome)}
}

func (r *stubOutcomeRepo) Create(_ context.Context, o *dialer.CallOutcome) error {
	r.outcomes[o.ID] = o; return nil
}
func (r *stubOutcomeRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*dialer.CallOutcome, error) {
	if o, ok := r.outcomes[id]; ok { return o, nil }
	return nil, dialer.ErrOutcomeNotFound
}
func (r *stubOutcomeRepo) List(_ context.Context, _ uuid.UUID, _ bool) ([]*dialer.CallOutcome, error) { return nil, nil }
func (r *stubOutcomeRepo) Update(_ context.Context, _ *dialer.CallOutcome) error { return nil }
func (r *stubOutcomeRepo) Delete(_ context.Context, _, _ uuid.UUID) error        { return nil }
func (r *stubOutcomeRepo) EnsureDefaults(_ context.Context, _ uuid.UUID) error   { return nil }

type stubAgentStatusRepo struct{}

func (r *stubAgentStatusRepo) LogStatusChange(_ context.Context, _ *dialer.AgentStatusLogEntry) error { return nil }
func (r *stubAgentStatusRepo) GetActiveAgentIDsForTenant(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (r *stubAgentStatusRepo) GetUserDisplayNames(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][2]string, error) {
	return nil, nil
}

type stubCRMBridge struct {
	contactDetails map[uuid.UUID]*dialer.ContactDetails
}

func newStubCRMBridge() *stubCRMBridge {
	return &stubCRMBridge{contactDetails: make(map[uuid.UUID]*dialer.ContactDetails)}
}

func (b *stubCRMBridge) CreateCallActivity(_ context.Context, _ dialer.CallActivityInput) error { return nil }
func (b *stubCRMBridge) GetContactDetails(_ context.Context, id uuid.UUID) (*dialer.ContactDetails, error) {
	if d, ok := b.contactDetails[id]; ok { return d, nil }
	return nil, errors.New("contact not found")
}
func (b *stubCRMBridge) ResolveFilterContacts(_ context.Context, _ uuid.UUID) ([]dialer.ContactImport, error) {
	return nil, nil
}

// newDialerTestServer builds a DialerGRPCServer backed by in-memory stubs.
// Returns the server plus the underlying repos for state assertions.
// grantingAssertRepo is a consent.AssertRepo stub that always reports active
// phone consent, so InitiateDialerCall's fail-closed consent gate passes in the
// happy-path test. Validation-path tests fail before the consent check is reached.
type grantingAssertRepo struct{}

func (grantingAssertRepo) ActiveConsent(_ context.Context, _ uuid.UUID, _ consent.Channel) (bool, error) {
	return true, nil
}

func (grantingAssertRepo) ContactHasEmail(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func newDialerTestServer() (*DialerGRPCServer, *stubCampaignRepo, *stubCallRepo, *stubOutcomeRepo, *stubCRMBridge) {
	cr := newStubCampaignRepo()
	cl := newStubCallRepo()
	or := newStubOutcomeRepo()
	br := newStubCRMBridge()
	// InitiateDialerCall fails closed without a consent asserter (8c66164c), so
	// wire one that grants consent for the happy path.
	asserter := consent.NewAsserter(grantingAssertRepo{}, nil)
	svc := dialer.NewServiceWithConsent(cr, cl, or, &stubAgentStatusRepo{}, nil, br, nil, asserter)
	srv := NewDialerGRPCServer(svc)
	return srv, cr, cl, or, br
}

// ctxWithTenant creates a context that satisfies middleware.GetTenantID.
func ctxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// InitiateDialerCall handler tests
// ============================================================================

func TestInitiateDialerCall_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	// Plain context — no tenant set.
	_, err := srv.InitiateDialerCall(context.Background(), &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: uuid.New().String(),
		AgentId:           uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInitiateDialerCall_InvalidCampaignContactID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	_, err := srv.InitiateDialerCall(ctx, &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: "not-a-uuid",
		AgentId:           uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInitiateDialerCall_InvalidAgentID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	_, err := srv.InitiateDialerCall(ctx, &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: uuid.New().String(),
		AgentId:           "bad-agent-id",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInitiateDialerCall_InvalidCallSessionID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	// Provide an invalid optional call_session_id — should return InvalidArgument.
	badCallSessionID := "not-a-uuid"
	_, err := srv.InitiateDialerCall(ctx, &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: uuid.New().String(),
		AgentId:           uuid.New().String(),
		CallSessionId:     &badCallSessionID,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInitiateDialerCall_Happy(t *testing.T) {
	srv, cr, cl, _, br := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	ccID := uuid.New()
	contactID := uuid.New()
	agentID := uuid.New()

	cr.contacts[ccID] = &dialer.CampaignContact{
		ID:        ccID,
		ContactID: contactID,
	}
	br.contactDetails[contactID] = &dialer.ContactDetails{ID: contactID, Name: "Test"}

	resp, err := srv.InitiateDialerCall(ctx, &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: ccID.String(),
		AgentId:           agentID.String(),
	})
	requireGRPCOK(t, err)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	// Verify a session was actually created in the store.
	if len(cl.sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(cl.sessions))
	}
}

// ============================================================================
// LogCallOutcome handler tests
// ============================================================================

func TestLogCallOutcome_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.LogCallOutcome(context.Background(), &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: uuid.New().String(),
		OutcomeId:           uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLogCallOutcome_InvalidSessionID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.LogCallOutcome(ctx, &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: "not-a-uuid",
		OutcomeId:           uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLogCallOutcome_InvalidOutcomeID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.LogCallOutcome(ctx, &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: uuid.New().String(),
		OutcomeId:           "bad-outcome",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLogCallOutcome_SessionNotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	// Valid UUIDs but no session seeded → NotFound.
	_, err := srv.LogCallOutcome(ctx, &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: uuid.New().String(),
		OutcomeId:           uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestLogCallOutcome_Happy(t *testing.T) {
	srv, cr, cl, or, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	sessionID := uuid.New()
	outcomeID := uuid.New()
	ccID := uuid.New()

	cl.sessions[sessionID] = &dialer.CallSession{
		ID:                sessionID,
		TenantID:          tenantID,
		CampaignContactID: ccID,
		AgentID:           uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	cl.events = append(cl.events, &dialer.CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           dialer.CallEventInitiated,
		OccurredAt:          time.Now().Add(-20 * time.Second),
	})
	or.outcomes[outcomeID] = &dialer.CallOutcome{
		ID:         outcomeID,
		Label:      "Erreicht",
		IsPositive: true,
	}
	cr.contacts[ccID] = &dialer.CampaignContact{
		ID:         ccID,
		CampaignID: uuid.New(),
		ContactID:  uuid.New(),
	}

	resp, err := srv.LogCallOutcome(ctx, &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: sessionID.String(),
		OutcomeId:           outcomeID.String(),
	})
	requireGRPCOK(t, err)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
