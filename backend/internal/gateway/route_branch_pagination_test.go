package gateway

// Branch-module lists (vermietung, inventar, fuhrpark, rapporte, schichten up to 200 rows per page;
// produktion, vertraege, einkauf up to 100) must clamp an oversized page_size to the ceiling their
// service accepts, not fall back to the route default — the desktop asks for 200 and 500 rows.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
)

type branchListStub struct {
	vermietungv1.UnimplementedVermietungServiceServer
	objects *vermietungv1.ListObjectsRequest
}

func (s *branchListStub) ListObjects(_ context.Context, r *vermietungv1.ListObjectsRequest) (*vermietungv1.ListObjectsResponse, error) {
	s.objects = r
	return &vermietungv1.ListObjectsResponse{}, nil
}

type produktionListStub struct {
	produktionv1.UnimplementedProduktionServiceServer
	orders *produktionv1.ListOrdersRequest
}

func (s *produktionListStub) ListOrders(_ context.Context, r *produktionv1.ListOrdersRequest) (*produktionv1.ListOrdersResponse, error) {
	s.orders = r
	return &produktionv1.ListOrdersResponse{}, nil
}

func serveBranchStub(t *testing.T, name string, register func(*grpc.Server)) *ServiceRegistry {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register(name, lis.Addr().String())
	t.Cleanup(reg.Close)
	return reg
}

func TestBranchPagination_VermietungObjectsClampToTwoHundred(t *testing.T) {
	for query, want := range map[string]int32{"": 50, "?page_size=200": 200, "?page_size=500": 200, "?page_size=25": 25} {
		stub := &branchListStub{}
		reg := serveBranchStub(t, "vermietung", func(s *grpc.Server) { vermietungv1.RegisterVermietungServiceServer(s, stub) })

		req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/vermietung/objects"+query, nil), "user-123", testTenantID)
		rec := httptest.NewRecorder()
		NewVermietungRoutes(reg, nil).HandleListObjects(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, stub.objects)
		assert.Equal(t, want, stub.objects.GetPageSize(), "query %q", query)
	}
}

func TestBranchPagination_ProduktionOrdersClampToOneHundred(t *testing.T) {
	for query, want := range map[string]int32{"": 20, "?page_size=100": 100, "?page_size=500": 100} {
		stub := &produktionListStub{}
		reg := serveBranchStub(t, "produktion", func(s *grpc.Server) { produktionv1.RegisterProduktionServiceServer(s, stub) })

		req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/produktion/orders"+query, nil), "user-123", testTenantID)
		rec := httptest.NewRecorder()
		NewProduktionRoutes(reg, nil).HandleListOrders(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, stub.orders)
		assert.Equal(t, want, stub.orders.GetPageSize(), "query %q", query)
	}
}

// Every list handler of the branch modules parses page_size through parsePaginationMax; a plain
// parsePagination call would silently turn page_size=200 back into the route default.
func TestBranchPagination_NoPlainParsePaginationInBranchRoutes(t *testing.T) {
	for _, mod := range []string{"vermietung", "inventar", "fuhrpark", "rapporte", "schichten", "produktion", "vertraege", "einkauf"} {
		files, err := filepath.Glob("route_" + mod + "*.go")
		require.NoError(t, err)
		require.NotEmpty(t, files, mod)
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			require.NoError(t, err)
			assert.NotContains(t, string(src), "parsePagination(r,", f)
		}
	}
}
