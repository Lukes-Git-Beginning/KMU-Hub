package gateway

// GET /api/v1/users is read by the team module's assignment tab without any paging
// parameter. The tests read the request the auth service receives over a real gRPC hop.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

type listUsersStub struct {
	authv1.UnimplementedAuthServiceServer
	got *authv1.ListUsersRequest
}

func (s *listUsersStub) ListUsers(_ context.Context, r *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	s.got = r
	return &authv1.ListUsersResponse{}, nil
}

func listUsersViaHandler(t *testing.T, query string) *authv1.ListUsersRequest {
	t.Helper()
	stub := &listUsersStub{}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("auth", lis.Addr().String())
	t.Cleanup(reg.Close)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users"+query, nil)
	rec := httptest.NewRecorder()
	NewAuthRoutes(reg).HandleListUsers(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.got)
	return stub.got
}

func TestAuthWire_ListUsersWithoutParametersAsksForTheLargestPage(t *testing.T) {
	got := listUsersViaHandler(t, "")

	assert.EqualValues(t, 1, got.GetPage())
	assert.EqualValues(t, userListPageSizeMax, got.GetPageSize())
}

func TestAuthWire_ListUsersForwardsPageAndPageSize(t *testing.T) {
	got := listUsersViaHandler(t, "?page=2&page_size=50")

	assert.EqualValues(t, 2, got.GetPage())
	assert.EqualValues(t, 50, got.GetPageSize())
}

func TestAuthWire_ListUsersClampsAnOversizedPageToTheServiceCeiling(t *testing.T) {
	// the auth service answers page_size > 100 with its default of 20, so the gateway must
	// never forward more than the ceiling
	got := listUsersViaHandler(t, "?page_size=500")

	assert.EqualValues(t, userListPageSizeMax, got.GetPageSize())
}
