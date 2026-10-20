package server

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/dialer"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

// newDialerTestServerWithAgentStore is a variant of newDialerTestServer that
// wires a real *dialer.AgentStatusStore backed by miniredis instead of nil.
// SetAgentStatus/GetAgentStatus/GetCampaignAgents dereference the service's
// agentStore field directly (it's a concrete *AgentStatusStore, not an
// interface) — calling them against the nil store from newDialerTestServer
// would panic, so any test exercising those three RPCs needs this helper.
func newDialerTestServerWithAgentStore(t *testing.T) (*DialerGRPCServer, *stubCampaignRepo, *stubCallRepo, *stubOutcomeRepo) {
	t.Helper()
	return newDialerTestServerWithAgentRepo(t, &stubAgentStatusRepo{})
}

// newDialerTestServerWithAgentRepo is newDialerTestServerWithAgentStore with
// a caller-supplied agent repo, for tests that pin users to tenants.
func newDialerTestServerWithAgentRepo(t *testing.T, agentRepo *stubAgentStatusRepo) (*DialerGRPCServer, *stubCampaignRepo, *stubCallRepo, *stubOutcomeRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })

	cr := newStubCampaignRepo()
	cl := newStubCallRepo()
	or := newStubOutcomeRepo()
	br := newStubCRMBridge()
	store := dialer.NewAgentStatusStore(rc)
	asserter := consent.NewAsserter(grantingAssertRepo{}, nil)
	svc := dialer.NewServiceWithConsent(cr, cl, or, agentRepo, store, br, nil, asserter)
	srv := NewDialerGRPCServer(svc)
	return srv, cr, cl, or
}

// ============================================================================
// SaveCallNotes
// ============================================================================

func TestSaveCallNotes_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.SaveCallNotes(context.Background(), &dialerv1.SaveCallNotesRequest{
		DialerCallSessionId: uuid.New().String(),
		Notes:               "test",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSaveCallNotes_InvalidSessionID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SaveCallNotes(ctx, &dialerv1.SaveCallNotesRequest{DialerCallSessionId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSaveCallNotes_SessionNotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SaveCallNotes(ctx, &dialerv1.SaveCallNotesRequest{DialerCallSessionId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSaveCallNotes_Happy(t *testing.T) {
	srv, _, cl, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	sessionID := uuid.New()
	cl.sessions[sessionID] = &dialer.CallSession{ID: sessionID, TenantID: tenantID}

	_, err := srv.SaveCallNotes(ctx, &dialerv1.SaveCallNotesRequest{
		DialerCallSessionId: sessionID.String(),
		Notes:               "Ansprechpartner nicht erreicht",
	})
	requireGRPCOK(t, err)
	if cl.sessions[sessionID].Notes == nil || *cl.sessions[sessionID].Notes != "Ansprechpartner nicht erreicht" {
		t.Error("expected notes to be persisted on the session")
	}
}

// ============================================================================
// CompleteWrapUp
// ============================================================================

func TestCompleteWrapUp_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.CompleteWrapUp(context.Background(), &dialerv1.CompleteWrapUpRequest{
		DialerCallSessionId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCompleteWrapUp_InvalidSessionID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.CompleteWrapUp(ctx, &dialerv1.CompleteWrapUpRequest{DialerCallSessionId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCompleteWrapUp_SessionNotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.CompleteWrapUp(ctx, &dialerv1.CompleteWrapUpRequest{DialerCallSessionId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCompleteWrapUp_Happy(t *testing.T) {
	srv, _, cl, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	sessionID := uuid.New()
	cl.sessions[sessionID] = &dialer.CallSession{ID: sessionID, TenantID: tenantID, CampaignContactID: uuid.New()}

	_, err := srv.CompleteWrapUp(ctx, &dialerv1.CompleteWrapUpRequest{DialerCallSessionId: sessionID.String()})
	requireGRPCOK(t, err)
	if cl.sessions[sessionID].WrapUpCompletedAt == nil {
		t.Error("expected WrapUpCompletedAt to be set")
	}
}

// ============================================================================
// SetAgentStatus
// ============================================================================

func TestSetAgentStatus_MissingTenant(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	_, err := srv.SetAgentStatus(context.Background(), &dialerv1.SetAgentStatusRequest{
		UserId: uuid.New().String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSetAgentStatus_InvalidUserID(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId: "not-a-uuid",
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetAgentStatus_InvalidCampaignID(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	ctx := ctxWithTenant(uuid.New())
	badCampaign := "not-a-uuid"

	_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId:     uuid.New().String(),
		Status:     dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
		CampaignId: &badCampaign,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestSetAgentStatus_InvalidTransition: the dialer state machine rejects an offline
// agent jumping straight to wrap_up; that must surface as FailedPrecondition, not Internal.
func TestSetAgentStatus_InvalidTransition(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId: uuid.New().String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_WRAP_UP,
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestSetAgentStatus_Happy(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	ctx := ctxWithTenant(uuid.New())
	userID := uuid.New()

	resp, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId: userID.String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCOK(t, err)
	if resp.GetUserId() != userID.String() {
		t.Errorf("expected user_id %s, got %s", userID, resp.GetUserId())
	}
	if resp.GetStatus() != dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE {
		t.Errorf("expected AVAILABLE, got %v", resp.GetStatus())
	}
}

// TestSetAgentStatus_ForeignUserMatchesUnknown verifies that a caller cannot
// write the live status of an agent in another tenant: the RPC answers
// NotFound exactly as for an unknown user ID, and the owner still reads the
// status it set.
func TestSetAgentStatus_ForeignUserMatchesUnknown(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	agentB := uuid.New()
	srv, _, _, _ := newDialerTestServerWithAgentRepo(t, &stubAgentStatusRepo{
		userTenants: map[uuid.UUID]uuid.UUID{agentB: tenantB},
	})

	_, err := srv.SetAgentStatus(ctxWithTenant(tenantB), &dialerv1.SetAgentStatusRequest{
		UserId: agentB.String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCOK(t, err)

	_, foreignErr := srv.SetAgentStatus(ctxWithTenant(tenantA), &dialerv1.SetAgentStatusRequest{
		UserId: agentB.String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_BREAK,
	})
	_, unknownErr := srv.SetAgentStatus(ctxWithTenant(tenantA), &dialerv1.SetAgentStatusRequest{
		UserId: uuid.New().String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_BREAK,
	})
	requireGRPCCode(t, foreignErr, codes.NotFound)
	requireGRPCCode(t, unknownErr, codes.NotFound)
	if status.Convert(foreignErr).Message() != status.Convert(unknownErr).Message() {
		t.Errorf("expected identical answers, got %q and %q", foreignErr, unknownErr)
	}

	resp, err := srv.GetAgentStatus(ctxWithTenant(tenantB), &dialerv1.GetAgentStatusRequest{UserId: agentB.String()})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE {
		t.Errorf("expected tenant B's agent still AVAILABLE, got %v", resp.GetStatus())
	}
}

// TestSetAgentStatus_ForeignCampaignNotFound verifies that a campaign of
// another tenant is refused with NotFound.
func TestSetAgentStatus_ForeignCampaignNotFound(t *testing.T) {
	srv, cr, _, _ := newDialerTestServerWithAgentStore(t)
	foreignCampaign := uuid.New()
	cr.campaigns[foreignCampaign] = &dialer.Campaign{ID: foreignCampaign, TenantID: uuid.New()}
	campaignIDStr := foreignCampaign.String()

	_, err := srv.SetAgentStatus(ctxWithTenant(uuid.New()), &dialerv1.SetAgentStatusRequest{
		UserId:     uuid.New().String(),
		Status:     dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
		CampaignId: &campaignIDStr,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// GetAgentStatus
// ============================================================================

func TestGetAgentStatus_InvalidUserID(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	_, err := srv.GetAgentStatus(ctxWithTenant(uuid.New()), &dialerv1.GetAgentStatusRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAgentStatus_Happy(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	ctx := ctxWithTenant(uuid.New())
	userID := uuid.New()

	_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId: userID.String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCOK(t, err)

	resp, err := srv.GetAgentStatus(ctx, &dialerv1.GetAgentStatusRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE {
		t.Errorf("expected AVAILABLE, got %v", resp.GetStatus())
	}
}

func TestGetAgentStatus_MissingTenant(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	_, err := srv.GetAgentStatus(context.Background(), &dialerv1.GetAgentStatusRequest{UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// TestGetAgentStatus_ForeignTenantMatchesUnknown verifies that a supervisor
// asking for an agent of another tenant gets the very answer an unknown user
// ID gets — offline, no since — so the response confirms nothing.
func TestGetAgentStatus_ForeignTenantMatchesUnknown(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	foreignAgent := uuid.New()

	_, err := srv.SetAgentStatus(ctxWithTenant(uuid.New()), &dialerv1.SetAgentStatusRequest{
		UserId: foreignAgent.String(),
		Status: dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
	})
	requireGRPCOK(t, err)

	ctx := ctxWithTenant(uuid.New())
	unknownAgent := uuid.New()
	foreign, err := srv.GetAgentStatus(ctx, &dialerv1.GetAgentStatusRequest{UserId: foreignAgent.String()})
	requireGRPCOK(t, err)
	unknown, err := srv.GetAgentStatus(ctx, &dialerv1.GetAgentStatusRequest{UserId: unknownAgent.String()})
	requireGRPCOK(t, err)

	for name, resp := range map[string]*dialerv1.AgentStatus{"foreign": foreign, "unknown": unknown} {
		if resp.GetStatus() != dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_OFFLINE {
			t.Errorf("%s: expected OFFLINE, got %v", name, resp.GetStatus())
		}
		if resp.GetSince() != nil || resp.GetCampaignId() != "" {
			t.Errorf("%s: expected no since/campaign, got %+v", name, resp)
		}
	}
	if foreign.GetUserId() != foreignAgent.String() || unknown.GetUserId() != unknownAgent.String() {
		t.Errorf("expected the requested user IDs echoed, got %q and %q", foreign.GetUserId(), unknown.GetUserId())
	}
}

// ============================================================================
// GetCampaignAgents
// ============================================================================

func TestGetCampaignAgents_InvalidCampaignID(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	_, err := srv.GetCampaignAgents(ctxWithTenant(uuid.New()), &dialerv1.GetCampaignAgentsRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCampaignAgents_Happy(t *testing.T) {
	srv, cr, _, _ := newDialerTestServerWithAgentStore(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	userID := uuid.New()
	campaignID := uuid.New()
	campaignIDStr := campaignID.String()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID}

	_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
		UserId:     userID.String(),
		Status:     dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
		CampaignId: &campaignIDStr,
	})
	requireGRPCOK(t, err)

	resp, err := srv.GetCampaignAgents(ctx, &dialerv1.GetCampaignAgentsRequest{CampaignId: campaignIDStr})
	requireGRPCOK(t, err)
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.GetAgents()))
	}
	if resp.GetAgents()[0].GetUserId() != userID.String() {
		t.Errorf("expected agent %s, got %s", userID, resp.GetAgents()[0].GetUserId())
	}
}

func TestGetCampaignAgents_MissingTenant(t *testing.T) {
	srv, _, _, _ := newDialerTestServerWithAgentStore(t)
	_, err := srv.GetCampaignAgents(context.Background(), &dialerv1.GetCampaignAgentsRequest{CampaignId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// TestGetCampaignAgents_DropsForeignTenantAgents shares one campaign set
// between two tenants and verifies the caller only sees its own agent.
func TestGetCampaignAgents_DropsForeignTenantAgents(t *testing.T) {
	srv, cr, _, _ := newDialerTestServerWithAgentStore(t)
	campaignID := uuid.New()
	campaignIDStr := campaignID.String()
	// Seeded without a tenant, so the stub lets both callers join: stands in
	// for a mixed set written before SetAgentStatus checked the campaign tenant.
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID}
	ownCtx, foreignCtx := ctxWithTenant(uuid.New()), ctxWithTenant(uuid.New())
	ownAgent, foreignAgent := uuid.New(), uuid.New()

	for agent, ctx := range map[uuid.UUID]context.Context{ownAgent: ownCtx, foreignAgent: foreignCtx} {
		_, err := srv.SetAgentStatus(ctx, &dialerv1.SetAgentStatusRequest{
			UserId:     agent.String(),
			Status:     dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE,
			CampaignId: &campaignIDStr,
		})
		requireGRPCOK(t, err)
	}

	resp, err := srv.GetCampaignAgents(ownCtx, &dialerv1.GetCampaignAgentsRequest{CampaignId: campaignIDStr})
	requireGRPCOK(t, err)
	if len(resp.GetAgents()) != 1 || resp.GetAgents()[0].GetUserId() != ownAgent.String() {
		t.Errorf("expected only own agent %s, got %v", ownAgent, resp.GetAgents())
	}
}

// ============================================================================
// ListCallOutcomes
// ============================================================================

func TestListCallOutcomes_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.ListCallOutcomes(context.Background(), &dialerv1.ListCallOutcomesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListCallOutcomes_EmptyWireShape(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	resp, err := srv.ListCallOutcomes(ctx, &dialerv1.ListCallOutcomesRequest{})
	requireGRPCOK(t, err)
	if resp.GetOutcomes() == nil {
		t.Error("expected empty slice, got nil (wire-shape must be [] not null)")
	}
}

// ============================================================================
// CreateCallOutcome
// ============================================================================

func TestCreateCallOutcome_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.CreateCallOutcome(context.Background(), &dialerv1.CreateCallOutcomeRequest{Label: "Erreicht"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateCallOutcome_Happy(t *testing.T) {
	srv, _, _, or, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	resp, err := srv.CreateCallOutcome(ctx, &dialerv1.CreateCallOutcomeRequest{
		Label:      "Erreicht",
		Color:      "#00FF00",
		IsPositive: true,
	})
	requireGRPCOK(t, err)
	if resp.GetLabel() != "Erreicht" {
		t.Errorf("expected label Erreicht, got %s", resp.GetLabel())
	}
	if len(or.outcomes) != 1 {
		t.Errorf("expected 1 stored outcome, got %d", len(or.outcomes))
	}
}

// ============================================================================
// UpdateCallOutcome
// ============================================================================

func TestUpdateCallOutcome_InvalidOutcomeID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.UpdateCallOutcome(ctx, &dialerv1.UpdateCallOutcomeRequest{OutcomeId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCallOutcome_NotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.UpdateCallOutcome(ctx, &dialerv1.UpdateCallOutcomeRequest{OutcomeId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateCallOutcome_Happy(t *testing.T) {
	srv, _, _, or, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	id := uuid.New()
	or.outcomes[id] = &dialer.CallOutcome{ID: id, TenantID: tenantID, Label: "Alt", IsActive: true}

	newLabel := "Neu"
	resp, err := srv.UpdateCallOutcome(ctx, &dialerv1.UpdateCallOutcomeRequest{
		OutcomeId: id.String(),
		Label:     &newLabel,
	})
	requireGRPCOK(t, err)
	if resp.GetLabel() != "Neu" {
		t.Errorf("expected label Neu, got %s", resp.GetLabel())
	}
}

// ============================================================================
// DeleteCallOutcome
// ============================================================================

func TestDeleteCallOutcome_InvalidOutcomeID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.DeleteCallOutcome(ctx, &dialerv1.DeleteCallOutcomeRequest{OutcomeId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestDeleteCallOutcome_NotFound is the real error path for this RPC. The
// done_when note about a "still referenced" rejection does not apply:
// migration 000130 deliberately set fk_dcc_outcome/fk_dcs_outcome to
// ON DELETE SET NULL (outcome labels are tenant-configurable and must stay
// deletable without cascading into business/audit data), so deleting a
// referenced outcome succeeds by design. The only failure mode left is a
// missing/already-deleted outcome.
func TestDeleteCallOutcome_NotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.DeleteCallOutcome(ctx, &dialerv1.DeleteCallOutcomeRequest{OutcomeId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteCallOutcome_Happy(t *testing.T) {
	srv, _, _, or, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	id := uuid.New()
	or.outcomes[id] = &dialer.CallOutcome{ID: id, TenantID: tenantID, Label: "Erreicht"}

	_, err := srv.DeleteCallOutcome(ctx, &dialerv1.DeleteCallOutcomeRequest{OutcomeId: id.String()})
	requireGRPCOK(t, err)
	if _, ok := or.outcomes[id]; ok {
		t.Error("expected outcome removed from store")
	}
}

// ============================================================================
// GetCampaignDashboard
// ============================================================================

func TestGetCampaignDashboard_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetCampaignDashboard(ctx, &dialerv1.GetCampaignDashboardRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCampaignDashboard_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.GetCampaignDashboard(context.Background(), &dialerv1.GetCampaignDashboardRequest{CampaignId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetCampaignDashboard_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())
	campaignID := uuid.New()

	resp, err := srv.GetCampaignDashboard(ctx, &dialerv1.GetCampaignDashboardRequest{CampaignId: campaignID.String()})
	requireGRPCOK(t, err)
	if resp.GetCampaignId() != campaignID.String() {
		t.Errorf("expected campaign_id %s, got %s", campaignID, resp.GetCampaignId())
	}
	if resp.GetPendingContacts() != 1 {
		t.Errorf("expected pending_contacts 1 (stub default), got %d", resp.GetPendingContacts())
	}
}

// ============================================================================
// GetAgentDashboard
// ============================================================================

func TestGetAgentDashboard_InvalidAgentID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetAgentDashboard(ctx, &dialerv1.GetAgentDashboardRequest{AgentId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAgentDashboard_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())
	agentID := uuid.New()

	resp, err := srv.GetAgentDashboard(ctx, &dialerv1.GetAgentDashboardRequest{AgentId: agentID.String()})
	requireGRPCOK(t, err)
	if resp.GetAgentId() != agentID.String() {
		t.Errorf("expected agent_id %s, got %s", agentID, resp.GetAgentId())
	}
}

// ============================================================================
// GetSupervisorOverview
// ============================================================================

func TestGetSupervisorOverview_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.GetSupervisorOverview(context.Background(), &dialerv1.GetSupervisorOverviewRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// TestGetSupervisorOverview_Happy uses newDialerTestServer (nil agentStore)
// deliberately: GetActiveAgentIDsForTenant on stubAgentStatusRepo returns an
// empty slice, so the per-agent loop that would otherwise dereference the nil
// *AgentStatusStore never executes.
func TestGetSupervisorOverview_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	resp, err := srv.GetSupervisorOverview(ctx, &dialerv1.GetSupervisorOverviewRequest{})
	requireGRPCOK(t, err)
	if resp.GetAgents() == nil || resp.GetRecentCalls() == nil {
		t.Error("expected empty slices, got nil (wire-shape must be [] not null)")
	}
	if resp.GetTotals() == nil {
		t.Error("expected non-nil totals")
	}
}

// ============================================================================
// GetContactCalls
// ============================================================================

func TestGetContactCalls_InvalidCampaignContactID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetContactCalls(ctx, &dialerv1.GetContactCallsRequest{CampaignContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContactCalls_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	_, err := srv.GetContactCalls(context.Background(), &dialerv1.GetContactCallsRequest{CampaignContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetContactCalls_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())
	ccID := uuid.New()

	resp, err := srv.GetContactCalls(ctx, &dialerv1.GetContactCallsRequest{CampaignContactId: ccID.String()})
	requireGRPCOK(t, err)
	if resp.GetCalls() == nil {
		t.Error("expected empty slice, got nil (wire-shape must be [] not null)")
	}
}
