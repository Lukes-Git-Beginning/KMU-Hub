package gateway

// An empty list must reach the client as `[]` next to its `total`. response.Proto drops
// an empty repeated field, so a caller without reports used to get `{}`.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

type rapporteEmptyListStub struct {
	rapportev1.UnimplementedRapporteServiceServer
}

func (rapporteEmptyListStub) ListReports(context.Context, *rapportev1.ListReportsRequest) (*rapportev1.ListReportsResponse, error) {
	return &rapportev1.ListReportsResponse{}, nil
}
func (rapporteEmptyListStub) ListPendingApprovals(context.Context, *rapportev1.ListPendingApprovalsRequest) (*rapportev1.ListReportsResponse, error) {
	return &rapportev1.ListReportsResponse{}, nil
}
func (rapporteEmptyListStub) ListLines(context.Context, *rapportev1.ListLinesRequest) (*rapportev1.ListLinesResponse, error) {
	return &rapportev1.ListLinesResponse{}, nil
}
func (rapporteEmptyListStub) ListAttachments(context.Context, *rapportev1.ListAttachmentsRequest) (*rapportev1.ListAttachmentsResponse, error) {
	return &rapportev1.ListAttachmentsResponse{}, nil
}
func (rapporteEmptyListStub) ListMeasurements(context.Context, *rapportev1.ListMeasurementsRequest) (*rapportev1.ListMeasurementsResponse, error) {
	return &rapportev1.ListMeasurementsResponse{}, nil
}
func (rapporteEmptyListStub) ListTemplates(context.Context, *rapportev1.ListTemplatesRequest) (*rapportev1.ListTemplatesResponse, error) {
	return &rapportev1.ListTemplatesResponse{}, nil
}

func TestRapporteRoutes_EmptyListsAreWrappedArrays(t *testing.T) {
	const reportID = "11111111-1111-1111-1111-111111111111"
	routes := newRapporteRoutes(registryFor(t, "rapporte", func(s *grpc.Server) {
		rapportev1.RegisterRapporteServiceServer(s, rapporteEmptyListStub{})
	}))
	router := chi.NewRouter()
	routes.RegisterRoutes(router, guardTestAuth)

	perms := []string{"rapporte:report:read", "rapporte:measurement:read", "rapporte:template:read"}
	scopes := map[string]string{"rapporte:report:read": auth.ScopeAll}

	cases := []struct{ path, key string }{
		{"/api/v1/rapporte/reports", "reports"},
		{"/api/v1/rapporte/pending", "reports"},
		{"/api/v1/rapporte/reports/" + reportID + "/lines", "lines"},
		{"/api/v1/rapporte/reports/" + reportID + "/attachments", "attachments"},
		{"/api/v1/rapporte/measurements", "measurements"},
		{"/api/v1/rapporte/templates", "templates"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", c.path, strings.NewReader(""))
			req = withScopes(withPermissions(withAuth(req, uuid.NewString(), testTenantID), perms...), scopes)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var body map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
			assert.JSONEq(t, `[]`, string(body[c.key]), "%s must be an empty array, got %s", c.key, rec.Body.String())
			assert.JSONEq(t, `0`, string(body["total"]), "total must be present, got %s", rec.Body.String())
		})
	}
}
