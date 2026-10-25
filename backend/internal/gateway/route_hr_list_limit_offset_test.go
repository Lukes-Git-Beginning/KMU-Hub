package gateway

// The desktop's HR hooks (ListLeaveRequestsParams, ListWorkTimeEntriesParams, ListEmployeesParams)
// speak limit/offset, the handlers speak page/page_size. A real gRPC server on loopback records the
// paging each list RPC actually receives.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type hrPagingStub struct {
	hrv1.UnimplementedHRServiceServer
	page, perPage int32
}

func (s *hrPagingStub) ListLeaveRequests(_ context.Context, r *hrv1.ListLeaveRequestsReq) (*hrv1.ListLeaveRequestsResp, error) {
	s.page, s.perPage = r.GetPage(), r.GetPerPage()
	return &hrv1.ListLeaveRequestsResp{}, nil
}

func (s *hrPagingStub) ListWorkTimeEntries(_ context.Context, r *hrv1.ListWorkTimeEntriesReq) (*hrv1.ListWorkTimeEntriesResp, error) {
	s.page, s.perPage = r.GetPage(), r.GetPerPage()
	return &hrv1.ListWorkTimeEntriesResp{}, nil
}

func (s *hrPagingStub) ListEmployees(_ context.Context, r *hrv1.ListEmployeesReq) (*hrv1.ListEmployeesResp, error) {
	s.page, s.perPage = r.GetPage(), r.GetPerPage()
	return &hrv1.ListEmployeesResp{}, nil
}

func TestHRLists_LimitOffsetBecomePageAndPageSize(t *testing.T) {
	stub := &hrPagingStub{}
	router := chi.NewRouter()
	NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil).
		RegisterRoutes(router, guardTestAuth)

	routes := []string{"/api/v1/hr/leave/requests", "/api/v1/hr/time/entries", "/api/v1/hr/employees"}
	cases := []struct {
		name, query   string
		page, perPage int32
	}{
		{"limit and offset", "limit=10&offset=20", 3, 10},
		{"limit alone", "limit=25", 1, 25},
		{"offset inside a page rounds down", "limit=10&offset=25", 3, 10},
		{"page and page_size win over limit and offset", "page=2&page_size=5&limit=10&offset=20", 2, 5},
		{"a lone page keeps the default size, limit ignored", "page=4&limit=10", 4, 50},
		{"limit over the ceiling falls back to the default", "limit=500&offset=40", 1, 50},
		{"garbage limit falls back to the default", "limit=abc&offset=20", 1, 50},
		{"no paging at all", "", 1, 50},
	}
	for _, route := range routes {
		for _, tc := range cases {
			t.Run(route+" "+tc.name, func(t *testing.T) {
				*stub = hrPagingStub{}
				rec := httptest.NewRecorder()
				target := route
				if tc.query != "" {
					target += "?" + tc.query
				}
				router.ServeHTTP(rec, adminToken.request(http.MethodGet, target))
				require.Equal(t, http.StatusOK, rec.Code, fmt.Sprintf("%s: %s", target, rec.Body.String()))
				assert.Equal(t, tc.page, stub.page, "page")
				assert.Equal(t, tc.perPage, stub.perPage, "page size")
			})
		}
	}
}
