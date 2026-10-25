package gateway

// Next contact, call start and dashboard act for the CALLER; only agent:manage may name another
// agent through agent_id. A real gRPC server on loopback plays the dialer service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

type agentPinStub struct {
	dialerv1.UnimplementedDialerServiceServer
	next      *dialerv1.GetNextContactRequest
	initiated *dialerv1.InitiateDialerCallRequest
	dashboard *dialerv1.GetAgentDashboardRequest
}

func (s *agentPinStub) GetNextContact(_ context.Context, r *dialerv1.GetNextContactRequest) (*dialerv1.CampaignContact, error) {
	s.next = r
	return &dialerv1.CampaignContact{}, nil
}

func (s *agentPinStub) InitiateDialerCall(_ context.Context, r *dialerv1.InitiateDialerCallRequest) (*dialerv1.DialerCallSession, error) {
	s.initiated = r
	return &dialerv1.DialerCallSession{}, nil
}

func (s *agentPinStub) GetAgentDashboard(_ context.Context, r *dialerv1.GetAgentDashboardRequest) (*dialerv1.AgentDashboard, error) {
	s.dashboard = r
	return &dialerv1.AgentDashboard{}, nil
}

func TestDialerAgentID_AgentIsPinnedToTheCallerAndManagerMayNameAnother(t *testing.T) {
	stub := &agentPinStub{}
	dr := NewDialerRoutes(registryFor(t, "dialer", func(s *grpc.Server) { dialerv1.RegisterDialerServiceServer(s, stub) }))
	router := chi.NewRouter()
	dr.RegisterRoutes(router, guardTestAuth)

	agent := tokenShape{perms: []string{"dialer:calls:write"}}
	manager := tokenShape{perms: []string{"dialer:calls:write", "dialer:agent:manage"}}
	const campaign = "66666666-6666-6666-6666-666666666666"
	const contact = "77777777-7777-7777-7777-777777777777"

	next := func(tk tokenShape, query string) {
		stub.next = nil
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tk.request(http.MethodGet, "/api/v1/dialer/campaigns/"+campaign+"/next"+query))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, stub.next)
	}
	initiate := func(tk tokenShape, agentID string) {
		stub.initiated = nil
		body := `{"campaign_contact_id":"` + contact + `"`
		if agentID != "" {
			body += `,"agent_id":"` + agentID + `"`
		}
		req := tk.request(http.MethodPost, "/api/v1/dialer/calls")
		req.Body = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body+"}")).Body
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NotNil(t, stub.initiated)
	}
	// The dashboard route is guarded by agent:manage today, so the pin is exercised on the handler
	// itself: it must hold the day the guard opens for self-service.
	dashboard := func(tk tokenShape, query string) {
		stub.dashboard = nil
		rec := httptest.NewRecorder()
		dr.HandleGetAgentDashboard(rec, tk.request(http.MethodGet, "/api/v1/dialer/dashboard/agent"+query))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, stub.dashboard)
	}

	next(agent, "")
	assert.Equal(t, leaveCallerID, stub.next.GetAgentId())
	next(agent, "?agent_id="+leaveOtherID)
	assert.Equal(t, leaveCallerID, stub.next.GetAgentId(), "an agent cannot draw a colleague's next contact")
	next(manager, "?agent_id="+leaveOtherID)
	assert.Equal(t, leaveOtherID, stub.next.GetAgentId())

	initiate(agent, "")
	assert.Equal(t, leaveCallerID, stub.initiated.GetAgentId())
	initiate(agent, leaveOtherID)
	assert.Equal(t, leaveCallerID, stub.initiated.GetAgentId(), "an agent cannot start a call in a colleague's name")
	initiate(manager, leaveOtherID)
	assert.Equal(t, leaveOtherID, stub.initiated.GetAgentId())

	dashboard(agent, "")
	assert.Equal(t, leaveCallerID, stub.dashboard.GetAgentId())
	dashboard(agent, "?agent_id="+leaveOtherID)
	assert.Equal(t, leaveCallerID, stub.dashboard.GetAgentId(), "an agent cannot read a colleague's figures")
	dashboard(manager, "?agent_id="+leaveOtherID)
	assert.Equal(t, leaveOtherID, stub.dashboard.GetAgentId())
}
