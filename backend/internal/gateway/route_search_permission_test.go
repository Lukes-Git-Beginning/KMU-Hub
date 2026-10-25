package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/middleware"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// Both search routes hand out rows of several CRM lists, and the global search
// adds documents and email. A search must not find a row the caller could not
// open in its own module, so each source — and each CRM entity type — is asked
// only when the caller holds that list's read key. The CRM stub below records
// which entity types reach it; the document service is not registered, so a
// "documents" module in the response proves the gateway tried to ask it.

// crmSearchStub records the entity types of every Search call it receives.
type crmSearchStub struct {
	crmv1.UnimplementedCRMServiceServer
	mu    sync.Mutex
	calls [][]string
}

func (s *crmSearchStub) Search(_ context.Context, req *crmv1.SearchRequest) (*crmv1.SearchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, req.EntityTypes)
	return &crmv1.SearchResponse{}, nil
}

func (s *crmSearchStub) recorded() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func newCRMSearchStubRegistry(t *testing.T) (*ServiceRegistry, *crmSearchStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &crmSearchStub{}
	srv := grpc.NewServer()
	crmv1.RegisterCRMServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("crm", lis.Addr().String())
	t.Cleanup(reg.Close)
	return reg, stub
}

// Permission sets as the presets hand them out (migrations 129, 256, 337, 338).
var (
	searchMemberPerms = []string{
		"search:read", "contacts:read", "companies:read", "deals:read", "activities:read",
		"crm:contact:read", "crm:deal:read", "documents:file:read", "email:mailbox:read",
	}
	// readonly/it_admin/hr_admin carry crm:contact:read but no deal key.
	searchContactOnlyPerms = []string{"search:read", "crm:contact:read", "documents:file:read"}
	// extern reads documents and nothing in CRM.
	searchExternPerms = []string{"search:read", "documents:file:read"}
)

func searchRequest(path string, perms []string, denied ...string) *http.Request {
	req := withPermissions(withAuth(httptest.NewRequest(http.MethodGet, path, nil), testHRUserID, testTenantID), perms...)
	if len(denied) > 0 {
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserDeniedKey, denied))
	}
	return req
}

func moduleNames(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	resp := decodeGlobalSearchResponse(t, rec)
	names := make([]string, 0, len(resp.Modules))
	for _, m := range resp.Modules {
		names = append(names, m.Module)
	}
	sort.Strings(names)
	return names
}

func TestGlobalSearch_SourcesFollowTheCallersReadKeys(t *testing.T) {
	cases := []struct {
		name        string
		perms       []string
		denied      []string
		wantModules []string
		wantCRM     []string // nil = CRM must not be asked at all
	}{
		{
			name:        "member finds contacts, firms, deals, activities, documents and own mail",
			perms:       searchMemberPerms,
			wantModules: []string{"crm", "documents", "email"},
			wantCRM:     []string{"contact", "company", "deal", "activity"},
		},
		{
			name:        "contact reader without a deal key gets no deals",
			perms:       searchContactOnlyPerms,
			wantModules: []string{"crm", "documents"},
			wantCRM:     []string{"contact", "company", "activity"},
		},
		{
			name:        "extern without any CRM key never reaches CRM",
			perms:       searchExternPerms,
			wantModules: []string{"documents"},
		},
		{
			name: "a denied fine key also silences its coarse stand-in",
			perms: []string{
				"search:read", "contacts:read", "companies:read", "deals:read", "activities:read",
			},
			denied:      []string{"crm:deal:read"},
			wantModules: []string{"crm"},
			wantCRM:     []string{"contact", "company", "activity"},
		},
		{
			name: "admin reaches every source",
			perms: append(append([]string{}, searchMemberPerms...),
				"documents:read", "email:read"),
			wantModules: []string{"crm", "documents", "email"},
			wantCRM:     []string{"contact", "company", "deal", "activity"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg, stub := newCRMSearchStubRegistry(t)
			router := chi.NewRouter()
			NewGlobalSearchRoutes(reg).RegisterRoutes(router, guardTestAuth)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, searchRequest("/api/v1/search/global?q=acme", tc.perms, tc.denied...))
			assertStatus(t, rec, http.StatusOK)

			assert.Equal(t, tc.wantModules, moduleNames(t, rec))
			if tc.wantCRM == nil {
				assert.Empty(t, stub.recorded(), "CRM must not be asked")
				return
			}
			require.Len(t, stub.recorded(), 1)
			assert.Equal(t, tc.wantCRM, stub.recorded()[0])
		})
	}
}

func TestGlobalSearch_RouteNeedsSearchRead(t *testing.T) {
	router := chi.NewRouter()
	NewGlobalSearchRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	perms := []string{"contacts:read", "crm:contact:read", "documents:file:read"}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, searchRequest("/api/v1/search/global?q=acme", perms))
	assertStatus(t, rec, http.StatusForbidden)
}

func TestCRMSearch_EntityTypesFollowTheCallersReadKeys(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		perms      []string
		wantStatus int
		wantCRM    []string // nil = CRM must not be asked at all
	}{
		{
			name:       "extern without any CRM key is refused",
			query:      "?q=acme",
			perms:      searchExternPerms,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "contact reader asking for all types gets no deals",
			query:      "?q=acme",
			perms:      searchContactOnlyPerms,
			wantStatus: http.StatusOK,
			wantCRM:    []string{"contact", "company", "activity"},
		},
		{
			name:       "contact reader asking only for deals gets an empty answer without a CRM call",
			query:      "?q=acme&types=deal",
			perms:      searchContactOnlyPerms,
			wantStatus: http.StatusOK,
		},
		{
			name:       "member keeps the requested types",
			query:      "?q=acme&types=deal&types=contact",
			perms:      searchMemberPerms,
			wantStatus: http.StatusOK,
			wantCRM:    []string{"deal", "contact"},
		},
		{
			name:       "comma-separated types as the desktop sends them are split and narrowed",
			query:      "?q=acme&types=contact,company,deal",
			perms:      searchContactOnlyPerms,
			wantStatus: http.StatusOK,
			wantCRM:    []string{"contact", "company"},
		},
		{
			name:       "an unknown type passes through for CRM to reject",
			query:      "?q=acme&types=bogus",
			perms:      searchMemberPerms,
			wantStatus: http.StatusOK,
			wantCRM:    []string{"bogus"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg, stub := newCRMSearchStubRegistry(t)
			router := chi.NewRouter()
			NewCRMRoutes(reg, nil).RegisterRoutes(router, guardTestAuth)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, searchRequest("/api/v1/search"+tc.query, tc.perms))
			assertStatus(t, rec, tc.wantStatus)

			if tc.wantCRM == nil {
				assert.Empty(t, stub.recorded(), "CRM must not be asked")
				return
			}
			require.Len(t, stub.recorded(), 1)
			assert.Equal(t, tc.wantCRM, stub.recorded()[0])
		})
	}
}
