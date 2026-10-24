package gateway

// A member holds rapporte:report:edit at scope "own" and nothing else of the
// rapporte keys. The line, attachment, submit and signature routes used to demand
// rapporte:line:* / rapporte:attachment:* / rapporte:report:write, so the member
// could create a report but never fill or submit it. These tests drive the real
// router against a real gRPC server and pin two things: the member gets through,
// and the request reaches the service flagged "author only" (without which the
// service would let them write into a colleague's report).

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// rapporteScopeStub records, per RPC name, whether the gateway flagged the request
// as author-only, and the uploader it forwarded.
type rapporteScopeStub struct {
	rapportev1.UnimplementedRapporteServiceServer
	mu         sync.Mutex
	restricted map[string]bool
	uploadedBy string
	authorID   string
}

func (s *rapporteScopeStub) record(rpc string, restricted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restricted == nil {
		s.restricted = map[string]bool{}
	}
	s.restricted[rpc] = restricted
}

func (s *rapporteScopeStub) CreateReport(_ context.Context, r *rapportev1.CreateReportRequest) (*rapportev1.ReportResponse, error) {
	s.mu.Lock()
	s.authorID = r.GetAuthorId()
	s.mu.Unlock()
	return &rapportev1.ReportResponse{}, nil
}
func (s *rapporteScopeStub) GetReport(_ context.Context, r *rapportev1.GetReportRequest) (*rapportev1.ReportResponse, error) {
	s.record("GetReport", r.GetRestrictToAuthor())
	return &rapportev1.ReportResponse{}, nil
}
func (s *rapporteScopeStub) UpdateReport(_ context.Context, r *rapportev1.UpdateReportRequest) (*rapportev1.ReportResponse, error) {
	s.record("UpdateReport", r.GetRestrictToAuthor())
	return &rapportev1.ReportResponse{}, nil
}
func (s *rapporteScopeStub) SubmitReport(_ context.Context, r *rapportev1.SubmitReportRequest) (*rapportev1.ReportResponse, error) {
	s.record("SubmitReport", r.GetRestrictToAuthor())
	return &rapportev1.ReportResponse{}, nil
}
func (s *rapporteScopeStub) SaveSignature(_ context.Context, r *rapportev1.SaveReportSignatureRequest) (*rapportev1.ReportResponse, error) {
	s.record("SaveSignature", r.GetRestrictToAuthor())
	return &rapportev1.ReportResponse{}, nil
}
func (s *rapporteScopeStub) AddLine(_ context.Context, r *rapportev1.AddLineRequest) (*rapportev1.LineResponse, error) {
	s.record("AddLine", r.GetRestrictToAuthor())
	return &rapportev1.LineResponse{}, nil
}
func (s *rapporteScopeStub) UpdateLine(_ context.Context, r *rapportev1.UpdateLineRequest) (*rapportev1.LineResponse, error) {
	s.record("UpdateLine", r.GetRestrictToAuthor())
	return &rapportev1.LineResponse{}, nil
}
func (s *rapporteScopeStub) DeleteLine(_ context.Context, r *rapportev1.DeleteLineRequest) (*rapportev1.DeleteLineResponse, error) {
	s.record("DeleteLine", r.GetRestrictToAuthor())
	return &rapportev1.DeleteLineResponse{}, nil
}
func (s *rapporteScopeStub) ListLines(_ context.Context, r *rapportev1.ListLinesRequest) (*rapportev1.ListLinesResponse, error) {
	s.record("ListLines", r.GetRestrictToAuthor())
	return &rapportev1.ListLinesResponse{}, nil
}
func (s *rapporteScopeStub) UploadAttachment(_ context.Context, r *rapportev1.UploadAttachmentRequest) (*rapportev1.AttachmentResponse, error) {
	s.record("UploadAttachment", r.GetRestrictToAuthor())
	s.mu.Lock()
	s.uploadedBy = r.GetUploadedBy()
	s.mu.Unlock()
	return &rapportev1.AttachmentResponse{}, nil
}
func (s *rapporteScopeStub) ListAttachments(_ context.Context, r *rapportev1.ListAttachmentsRequest) (*rapportev1.ListAttachmentsResponse, error) {
	s.record("ListAttachments", r.GetRestrictToAuthor())
	return &rapportev1.ListAttachmentsResponse{}, nil
}
func (s *rapporteScopeStub) DeleteAttachment(_ context.Context, r *rapportev1.DeleteAttachmentRequest) (*rapportev1.DeleteAttachmentResponse, error) {
	s.record("DeleteAttachment", r.GetRestrictToAuthor())
	return &rapportev1.DeleteAttachmentResponse{}, nil
}
func (s *rapporteScopeStub) CreateMeasurement(_ context.Context, r *rapportev1.CreateMeasurementRequest) (*rapportev1.CreateMeasurementResponse, error) {
	s.record("CreateMeasurement", r.GetRestrictToAuthor())
	return &rapportev1.CreateMeasurementResponse{}, nil
}
func (s *rapporteScopeStub) AddMeasurementPosition(_ context.Context, r *rapportev1.AddMeasurementPositionRequest) (*rapportev1.AddMeasurementPositionResponse, error) {
	s.record("AddMeasurementPosition", r.GetRestrictToAuthor())
	return &rapportev1.AddMeasurementPositionResponse{}, nil
}

func TestRapporteRoutes_MemberFillsAndSubmitsOwnReport(t *testing.T) {
	const (
		reportID = "11111111-1111-1111-1111-111111111111"
		lineID   = "22222222-2222-2222-2222-222222222222"
		attID    = "33333333-3333-3333-3333-333333333333"
		signData = "data:image/png;base64,AAAA"
	)
	callerID := uuid.NewString()

	// What the member preset holds: the fine report keys, edit and read at "own",
	// measurement:manage at "all" — and no line/attachment/write key.
	memberPerms := []string{"rapporte:report:create", "rapporte:report:edit", "rapporte:report:read", "rapporte:measurement:manage"}
	memberScopes := map[string]string{
		"rapporte:report:edit":        auth.ScopeOwn,
		"rapporte:report:read":        auth.ScopeOwn,
		"rapporte:measurement:manage": auth.ScopeAll,
	}
	// What a manager (and the admin's legacy keys) hold: the same keys at "all".
	managerScopes := map[string]string{
		"rapporte:report:edit":        auth.ScopeAll,
		"rapporte:report:read":        auth.ScopeAll,
		"rapporte:measurement:manage": auth.ScopeAll,
	}

	calls := []struct {
		name, rpc, method, path, body string
		wantStatus                    int
	}{
		{"list lines", "ListLines", "GET", "/api/v1/rapporte/reports/" + reportID + "/lines", "", http.StatusOK},
		{"add line", "AddLine", "POST", "/api/v1/rapporte/reports/" + reportID + "/lines", `{"description":"Kabel","quantity":2}`, http.StatusCreated},
		{"update line", "UpdateLine", "PATCH", "/api/v1/rapporte/lines/" + lineID + "/", `{"notes":"x"}`, http.StatusOK},
		{"delete line", "DeleteLine", "DELETE", "/api/v1/rapporte/lines/" + lineID + "/", "", http.StatusNoContent},
		{"list attachments", "ListAttachments", "GET", "/api/v1/rapporte/reports/" + reportID + "/attachments", "", http.StatusOK},
		{"upload attachment", "UploadAttachment", "POST", "/api/v1/rapporte/reports/" + reportID + "/attachments",
			`{"filename":"a.jpg","size_bytes":10,"object_key":"tenants/x/a.jpg","uploaded_by":"` + uuid.NewString() + `"}`, http.StatusCreated},
		{"delete attachment", "DeleteAttachment", "DELETE", "/api/v1/rapporte/attachments/" + attID + "/", "", http.StatusNoContent},
		{"sign", "SaveSignature", "PUT", "/api/v1/rapporte/reports/" + reportID + "/signature", `{"signature_data":"` + signData + `","signed_by":"Kunde"}`, http.StatusOK},
		{"submit", "SubmitReport", "POST", "/api/v1/rapporte/reports/" + reportID + "/submit", "", http.StatusOK},
		{"update report", "UpdateReport", "PATCH", "/api/v1/rapporte/reports/" + reportID + "/", `{"title":"neu"}`, http.StatusOK},
		{"get report", "GetReport", "GET", "/api/v1/rapporte/reports/" + reportID + "/", "", http.StatusOK},
		{"add measurement", "CreateMeasurement", "POST", "/api/v1/rapporte/measurements", `{"title":"Aufmaß","report_id":"` + reportID + `"}`, http.StatusCreated},
		{"add measurement position", "AddMeasurementPosition", "POST", "/api/v1/rapporte/measurements/" + lineID + "/positions", `{"description":"Wand"}`, http.StatusCreated},
	}

	run := func(t *testing.T, perms []string, scopes map[string]string, wantRestricted bool) *rapporteScopeStub {
		t.Helper()
		stub := &rapporteScopeStub{}
		routes := newRapporteRoutes(registryFor(t, "rapporte", func(s *grpc.Server) { rapportev1.RegisterRapporteServiceServer(s, stub) }))
		router := chi.NewRouter()
		routes.RegisterRoutes(router, guardTestAuth)

		for _, c := range calls {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			req = withScopes(withPermissions(withAuth(req, callerID, testTenantID), perms...), scopes)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, c.wantStatus, rec.Code, "%s: %s", c.name, rec.Body.String())
			stub.mu.Lock()
			got, seen := stub.restricted[c.rpc]
			stub.mu.Unlock()
			require.True(t, seen, "%s never reached the service", c.name)
			assert.Equal(t, wantRestricted, got, "%s: author-only flag", c.name)
		}
		return stub
	}

	t.Run("member at own scope gets through and is flagged author-only", func(t *testing.T) {
		stub := run(t, memberPerms, memberScopes, true)
		// The uploader is the caller, not the uuid the body named.
		assert.Equal(t, callerID, stub.uploadedBy)
	})

	t.Run("manager at all scope is not narrowed", func(t *testing.T) {
		run(t, memberPerms, managerScopes, false)
	})

	t.Run("a token without any report key is refused", func(t *testing.T) {
		stub := &rapporteScopeStub{}
		routes := newRapporteRoutes(registryFor(t, "rapporte", func(s *grpc.Server) { rapportev1.RegisterRapporteServiceServer(s, stub) }))
		router := chi.NewRouter()
		routes.RegisterRoutes(router, guardTestAuth)
		for _, c := range calls {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req = withPermissions(withAuth(req, callerID, testTenantID), "rapporte:template:read")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusForbidden, rec.Code, c.name)
		}
		assert.Empty(t, stub.restricted)
	})

	t.Run("a member cannot file a report under somebody else's name", func(t *testing.T) {
		stub := &rapporteScopeStub{}
		routes := newRapporteRoutes(registryFor(t, "rapporte", func(s *grpc.Server) { rapportev1.RegisterRapporteServiceServer(s, stub) }))
		router := chi.NewRouter()
		routes.RegisterRoutes(router, guardTestAuth)

		body := `{"title":"T","author_id":"` + uuid.NewString() + `"}`
		req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", strings.NewReader(body))
		req = withScopes(withPermissions(withAuth(req, callerID, testTenantID), memberPerms...), memberScopes)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		assert.Equal(t, callerID, stub.authorID)
	})
}
