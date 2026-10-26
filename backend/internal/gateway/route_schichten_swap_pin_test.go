package gateway

// A swap request names its requester in the body. At scope "own" the requester must be the caller:
// otherwise a member files a swap for a colleague's assignment. The gateway forwards the caller's
// user id as owner_user_id and the service decides (the requester may be the user id or the
// caller's HR profile id, see route_schichten_swap_own_identity_db_test.go). A real gRPC server on
// loopback plays the schichten service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

type swapPinStub struct {
	schichtenv1.UnimplementedSchichtenServiceServer
	created *schichtenv1.CreateSwapRequestRequest
}

func (s *swapPinStub) CreateSwapRequest(_ context.Context, r *schichtenv1.CreateSwapRequestRequest) (*schichtenv1.SwapRequestResponse, error) {
	s.created = r
	return &schichtenv1.SwapRequestResponse{SwapRequest: &schichtenv1.SwapRequest{Id: "s1"}}, nil
}

func TestHandleCreateSwapRequest_OwnScopeForwardsCallerToService(t *testing.T) {
	stub := &swapPinStub{}
	sr := newSchichtenRoutes(registryFor(t, "schichten", func(s *grpc.Server) { schichtenv1.RegisterSchichtenServiceServer(s, stub) }))

	call := func(scope, requester string) *httptest.ResponseRecorder {
		body := `{"assignment_id":"` + testSwapAssignmentID + `","requested_by_employee_id":"` + requester +
			`","swap_with_employee_id":"` + testSwapPartnerID + `","shift_id":"` + testSwapShiftID + `"}`
		req := withScopes(withAuth(httptest.NewRequest(http.MethodPost, "/swap-requests", strings.NewReader(body)), leaveCallerID, testTenantID),
			map[string]string{"schichten:swap:create": scope})
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "k1")
		rec := httptest.NewRecorder()
		sr.HandleCreateSwapRequest(rec, req)
		return rec
	}

	rec := call(auth.ScopeOwn, leaveOtherID)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, leaveCallerID, stub.created.GetOwnerUserId(), "at scope own the service gets the caller to check against")
	assert.Equal(t, leaveOtherID, stub.created.GetRequestedByEmployeeId(), "the requester is passed through, not rewritten")

	stub.created = nil
	rec = call(auth.ScopeAll, leaveOtherID)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Nil(t, stub.created.OwnerUserId, "a manager files on behalf of an employee without an owner check")
	assert.Equal(t, leaveOtherID, stub.created.GetRequestedByEmployeeId())
}
