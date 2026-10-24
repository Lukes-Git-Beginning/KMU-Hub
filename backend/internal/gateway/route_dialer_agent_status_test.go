package gateway

// An agent (calls:write) reads and sets their OWN dialer status; only agent:manage may name another
// user. A real gRPC server on loopback plays the dialer service.

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

type agentStatusStub struct {
	dialerv1.UnimplementedDialerServiceServer
	got *dialerv1.GetAgentStatusRequest
	set *dialerv1.SetAgentStatusRequest
}

func (s *agentStatusStub) GetAgentStatus(_ context.Context, r *dialerv1.GetAgentStatusRequest) (*dialerv1.AgentStatus, error) {
	s.got = r
	return &dialerv1.AgentStatus{UserId: r.GetUserId()}, nil
}

func (s *agentStatusStub) SetAgentStatus(_ context.Context, r *dialerv1.SetAgentStatusRequest) (*dialerv1.AgentStatus, error) {
	s.set = r
	return &dialerv1.AgentStatus{UserId: r.GetUserId()}, nil
}

func TestAgentStatus_AgentIsPinnedToThemselvesAndManagerMayNameAnother(t *testing.T) {
	stub := &agentStatusStub{}
	router := chi.NewRouter()
	NewDialerRoutes(registryFor(t, "dialer", func(s *grpc.Server) { dialerv1.RegisterDialerServiceServer(s, stub) })).
		RegisterRoutes(router, guardTestAuth)

	agent := tokenShape{perms: []string{"dialer:calls:write"}}
	manager := tokenShape{perms: []string{"dialer:agent:manage"}}
	const target = "/api/v1/dialer/agent-status"

	get := func(tk tokenShape, query string) int {
		stub.got = nil
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tk.request(http.MethodGet, target+query))
		return rec.Code
	}
	put := func(tk tokenShape, body string) int {
		stub.set = nil
		req := tk.request(http.MethodPut, target)
		req.Body = httptest.NewRequest(http.MethodPut, target, strings.NewReader(body)).Body
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	require.Equal(t, http.StatusOK, get(agent, ""))
	assert.Equal(t, leaveCallerID, stub.got.GetUserId())

	require.Equal(t, http.StatusOK, get(agent, "?user_id="+leaveOtherID))
	assert.Equal(t, leaveCallerID, stub.got.GetUserId(), "an agent cannot read a colleague's status")

	require.Equal(t, http.StatusOK, put(agent, `{"status":1}`))
	assert.Equal(t, leaveCallerID, stub.set.GetUserId())

	require.Equal(t, http.StatusOK, put(agent, `{"status":4,"user_id":"`+leaveOtherID+`"}`))
	assert.Equal(t, leaveCallerID, stub.set.GetUserId(), "an agent cannot set a colleague's status")

	require.Equal(t, http.StatusOK, get(manager, "?user_id="+leaveOtherID))
	assert.Equal(t, leaveOtherID, stub.got.GetUserId())

	require.Equal(t, http.StatusOK, put(manager, `{"status":4,"user_id":"`+leaveOtherID+`"}`))
	assert.Equal(t, leaveOtherID, stub.set.GetUserId())

	assert.Equal(t, http.StatusForbidden, get(tokenShape{perms: []string{"dialer:campaigns:read"}}, ""))
	assert.Equal(t, http.StatusForbidden, put(tokenShape{perms: []string{"dialer:campaigns:read"}}, `{"status":1}`))
}
