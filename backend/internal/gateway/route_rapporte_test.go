package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newRapporteRoutes(registry *ServiceRegistry) *RapporteRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_RAPPORTE_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewRapporteRoutes(registry, flags)
}

// RapporteRoutes.ServiceName() uses "rapporte".

func TestRapporteRoutes_ServiceName(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "rapporte" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "rapporte")
	}
}

// --- HandleCreateReport ---

func TestHandleCreateReport_MissingTitle(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"author_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateReport_MissingAuthorID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"title": "Test Report",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "author_id")
}

func TestHandleCreateReport_InvalidAuthorIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"title":     "Test Report",
		"author_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "author_id")
}

func TestHandleCreateReport_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleApproveReport ---

func TestHandleApproveReport_MissingReviewerID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{
			"review_note": "looks good",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveReport(rec, req)
	assertValidationError(t, rec, "reviewer_id")
}

func TestHandleApproveReport_InvalidReviewerIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{
			"reviewer_id": "not-a-uuid",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveReport(rec, req)
	assertValidationError(t, rec, "reviewer_id")
}

// --- HandleAddLine ---

func TestHandleAddLine_MissingDescription(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"quantity": 5.0,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddLine(rec, req)
	assertValidationError(t, rec, "description")
}

func TestHandleAddLine_ZeroQuantity(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"description": "Some work",
			"quantity":    0.0,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddLine(rec, req)
	assertValidationError(t, rec, "quantity")
}

// --- HandleUploadAttachment ---

func TestHandleUploadAttachment_MissingFilename(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments",
		jsonBody(t, map[string]interface{}{
			"object_key":  "files/test.pdf",
			"uploaded_by": "550e8400-e29b-41d4-a716-446655440001",
			"size_bytes":  1024,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertValidationError(t, rec, "filename")
}

func TestHandleUploadAttachment_InvalidLineIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	lineID := "not-a-uuid"
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments",
		jsonBody(t, map[string]interface{}{
			"filename":    "test.pdf",
			"object_key":  "files/test.pdf",
			"uploaded_by": "550e8400-e29b-41d4-a716-446655440001",
			"size_bytes":  1024,
			"line_id":     lineID,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertValidationError(t, rec, "line_id")
}
