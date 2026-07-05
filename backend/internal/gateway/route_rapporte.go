package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// RapporteRoutes handles HTTP routes for the Rapporte (field reports) module.
type RapporteRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewRapporteRoutes creates a new RapporteRoutes with the given service registry and feature flags.
func NewRapporteRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *RapporteRoutes {
	return &RapporteRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (rr *RapporteRoutes) ServiceName() string { return "rapporte" }

// getClient lazily obtains a gRPC client for the rapporte service.
func (rr *RapporteRoutes) getClient() (rapportev1.RapporteServiceClient, error) {
	conn, err := rr.registry.GetConnection("rapporte")
	if err != nil {
		return nil, err
	}
	return rapportev1.NewRapporteServiceClient(conn), nil
}

// RegisterRoutes mounts all Rapporte HTTP routes behind the feature flag modules.rapporte.
func (rr *RapporteRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !rr.flags.IsEnabled("modules.rapporte") {
		return
	}

	r.Route("/api/v1/rapporte", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Reports
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/reports", rr.HandleListReports)
		r.With(middleware.RequirePermission("rapporte:report", "write")).Post("/reports", rr.HandleCreateReport)

		r.Route("/reports/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/", rr.HandleGetReport)
			r.With(middleware.RequirePermission("rapporte:report", "write")).Patch("/", rr.HandleUpdateReport)
			r.With(middleware.RequirePermission("rapporte:report", "write")).Delete("/", rr.HandleDeleteReport)

			// Signature
			r.With(middleware.RequirePermission("rapporte:report", "write")).Put("/signature", rr.HandleSaveReportSignature)

			// State transitions
			r.With(middleware.RequirePermission("rapporte:report", "write")).Post("/submit", rr.HandleSubmitReport)
			// Bug #3: approve/reject require dedicated "approve" action, not generic "write"
			r.With(middleware.RequirePermission("rapporte:report", "approve")).Post("/approve", rr.HandleApproveReport)
			r.With(middleware.RequirePermission("rapporte:report", "approve")).Post("/reject", rr.HandleRejectReport)

			// Lines
			r.With(middleware.RequirePermission("rapporte:line", "read")).Get("/lines", rr.HandleListLines)
			r.With(middleware.RequirePermission("rapporte:line", "write")).Post("/lines", rr.HandleAddLine)

			// Attachments
			r.With(middleware.RequirePermission("rapporte:attachment", "read")).Get("/attachments", rr.HandleListAttachments)
			r.With(middleware.RequirePermission("rapporte:attachment", "write")).Post("/attachments", rr.HandleUploadAttachment)

			// Export
			r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/export/pdf", rr.HandleExportPDF)
		})

		// Lines (by line ID)
		r.Route("/lines/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:line", "write")).Patch("/", rr.HandleUpdateLine)
			r.With(middleware.RequirePermission("rapporte:line", "write")).Delete("/", rr.HandleDeleteLine)
		})

		// Attachments (by attachment ID)
		r.Route("/attachments/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:attachment", "write")).Delete("/", rr.HandleDeleteAttachment)
		})

		// Stats & approvals
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/stats", rr.HandleGetReportStats)
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/pending", rr.HandleListPendingApprovals)

		// Measurements
		r.With(middleware.RequirePermission("rapporte:measurement", "read")).Get("/measurements", rr.HandleListMeasurements)
		r.With(middleware.RequirePermission("rapporte:measurement", "write")).Post("/measurements", rr.HandleCreateMeasurement)
		r.Route("/measurements/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:measurement", "read")).Get("/", rr.HandleGetMeasurement)
			r.With(middleware.RequirePermission("rapporte:measurement", "write")).Put("/", rr.HandleUpdateMeasurement)
			r.With(middleware.RequirePermission("rapporte:measurement", "write")).Delete("/", rr.HandleDeleteMeasurement)
			r.With(middleware.RequirePermission("rapporte:measurement", "write")).Post("/positions", rr.HandleAddMeasurementPosition)
		})
		r.Route("/measurements/positions/{pos_id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:measurement", "write")).Delete("/", rr.HandleDeleteMeasurementPosition)
		})

		// Templates
		r.With(middleware.RequirePermission("rapporte:template", "read")).Get("/templates", rr.HandleListTemplates)
		r.With(middleware.RequirePermission("rapporte:template", "write")).Post("/templates", rr.HandleCreateTemplate)
		r.Route("/templates/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:template", "read")).Get("/", rr.HandleGetTemplate)
			r.With(middleware.RequirePermission("rapporte:template", "write")).Put("/", rr.HandleUpdateTemplate)
			r.With(middleware.RequirePermission("rapporte:template", "write")).Delete("/", rr.HandleDeleteTemplate)
		})
	})
}

// ============================================================================
// Request types
// ============================================================================

type createReportRequest struct {
	Title       string   `json:"title"       validate:"required"`
	Description string   `json:"description"`
	AuthorID    string   `json:"author_id"   validate:"required,uuid"`
	Lat         *float64 `json:"lat,omitempty"`
	Lon         *float64 `json:"lon,omitempty"`
	ReportDate  string   `json:"report_date,omitempty"`
}

type updateReportRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Lat         *float64 `json:"lat,omitempty"`
	Lon         *float64 `json:"lon,omitempty"`
	ReportDate  *string  `json:"report_date,omitempty"`
}

type approveRejectRequest struct {
	ReviewerID string `json:"reviewer_id" validate:"required,uuid"`
	ReviewNote string `json:"review_note"`
}

type addLineRequest struct {
	Description string  `json:"description" validate:"required"`
	Quantity    float64 `json:"quantity"    validate:"gt=0"`
	Unit        string  `json:"unit"`
	Notes       string  `json:"notes"`
	Position    int32   `json:"position"`
}

type updateLineRequest struct {
	Description *string  `json:"description,omitempty"`
	Quantity    *float64 `json:"quantity,omitempty"`
	Unit        *string  `json:"unit,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
	Position    *int32   `json:"position,omitempty"`
}

type saveReportSignatureRequest struct {
	SignatureData string `json:"signature_data" validate:"required"`
	SignedBy      string `json:"signed_by"      validate:"required"`
}

type rapporteUploadAttachmentRequest struct {
	LineID      *string `json:"line_id,omitempty"  validate:"omitempty,uuid"`
	Filename    string  `json:"filename"            validate:"required"`
	ContentType string  `json:"content_type"`
	SizeBytes   int64   `json:"size_bytes"          validate:"gt=0"`
	ObjectKey   string  `json:"object_key"          validate:"required"`
	UploadedBy  string  `json:"uploaded_by"         validate:"required,uuid"`
}

// ============================================================================
// Report Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListReports(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &rapportev1.ListReportsRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if st := q.Get("status"); st != "" {
		grpcReq.Status = &st
	}
	if aid := q.Get("author_id"); aid != "" {
		grpcReq.AuthorId = &aid
	}

	resp, err := client.ListReports(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleCreateReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createReportRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &rapportev1.CreateReportRequest{
		TenantId:    tenantID.String(),
		Title:       req.Title,
		Description: req.Description,
		AuthorId:    req.AuthorID,
		Lat:         req.Lat,
		Lon:         req.Lon,
		ReportDate:  req.ReportDate,
	}

	resp, err := client.CreateReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleGetReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetReport(r.Context(), &rapportev1.GetReportRequest{
		TenantId: tenantID.String(),
		ReportId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUpdateReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateReportRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &rapportev1.UpdateReportRequest{
		TenantId:    tenantID.String(),
		ReportId:    id,
		Title:       req.Title,
		Description: req.Description,
		Lat:         req.Lat,
		Lon:         req.Lon,
		ReportDate:  req.ReportDate,
	}

	resp, err := client.UpdateReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteReport(r.Context(), &rapportev1.DeleteReportRequest{
		TenantId: tenantID.String(),
		ReportId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// State Transition Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleSubmitReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.SubmitReport(r.Context(), &rapportev1.SubmitReportRequest{
		TenantId: tenantID.String(),
		ReportId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleApproveReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[approveRejectRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ApproveReport(r.Context(), &rapportev1.ApproveReportRequest{
		TenantId:   tenantID.String(),
		ReportId:   id,
		ReviewerId: req.ReviewerID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleRejectReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[approveRejectRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RejectReport(r.Context(), &rapportev1.RejectReportRequest{
		TenantId:   tenantID.String(),
		ReportId:   id,
		ReviewerId: req.ReviewerID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Line Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListLines(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListLines(r.Context(), &rapportev1.ListLinesRequest{
		TenantId: tenantID.String(),
		ReportId: reportID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleAddLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[addLineRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddLine(r.Context(), &rapportev1.AddLineRequest{
		TenantId:    tenantID.String(),
		ReportId:    reportID,
		Description: req.Description,
		Quantity:    req.Quantity,
		Unit:        req.Unit,
		Notes:       req.Notes,
		Position:    req.Position,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleUpdateLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	lineID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateLineRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &rapportev1.UpdateLineRequest{
		TenantId:    tenantID.String(),
		LineId:      lineID,
		Description: req.Description,
		Unit:        req.Unit,
		Notes:       req.Notes,
	}
	if req.Quantity != nil {
		grpcReq.Quantity = req.Quantity
	}
	if req.Position != nil {
		grpcReq.Position = req.Position
	}

	resp, err := client.UpdateLine(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	lineID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteLine(r.Context(), &rapportev1.DeleteLineRequest{
		TenantId: tenantID.String(),
		LineId:   lineID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Attachment Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListAttachments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	grpcReq := &rapportev1.ListAttachmentsRequest{
		TenantId: tenantID.String(),
		ReportId: reportID,
	}
	if lid := r.URL.Query().Get("line_id"); lid != "" {
		grpcReq.LineId = &lid
	}

	resp, err := client.ListAttachments(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[rapporteUploadAttachmentRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &rapportev1.UploadAttachmentRequest{
		TenantId:    tenantID.String(),
		ReportId:    reportID,
		LineId:      req.LineID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		ObjectKey:   req.ObjectKey,
		UploadedBy:  req.UploadedBy,
	}

	resp, err := client.UploadAttachment(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	attID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteAttachment(r.Context(), &rapportev1.DeleteAttachmentRequest{
		TenantId:     tenantID.String(),
		AttachmentId: attID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Signature Handler
// ============================================================================

func (rr *RapporteRoutes) HandleSaveReportSignature(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[saveReportSignatureRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.SaveSignature(r.Context(), &rapportev1.SaveReportSignatureRequest{
		TenantId:      tenantID.String(),
		ReportId:      reportID,
		SignatureData: req.SignatureData,
		SignedBy:      req.SignedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Stats & Export Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleGetReportStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	resp, err := client.GetReportStats(r.Context(), &rapportev1.GetReportStatsRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListPendingApprovals(r.Context(), &rapportev1.ListPendingApprovalsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleExportPDF(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ExportPDF(r.Context(), &rapportev1.ExportPDFRequest{
		TenantId: tenantID.String(),
		ReportId: reportID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", resp.GetContentType())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+resp.GetFilename()+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}

// ============================================================================
// Measurement request types
// ============================================================================

type createMeasurementRequest struct {
	ReportID   string `json:"report_id,omitempty"  validate:"omitempty,uuid"`
	Title      string `json:"title"                validate:"required"`
	Location   string `json:"location,omitempty"`
	MeasuredBy string `json:"measured_by,omitempty"`
	MeasuredAt string `json:"measured_at,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type updateMeasurementRequest struct {
	Title      string `json:"title"                validate:"required"`
	Location   string `json:"location,omitempty"`
	MeasuredBy string `json:"measured_by,omitempty"`
	MeasuredAt string `json:"measured_at,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type addMeasurementPositionRequest struct {
	PositionNumber int     `json:"position_number"`
	Description    string  `json:"description"   validate:"required"`
	Unit           string  `json:"unit,omitempty"`
	Quantity       float64 `json:"quantity,omitempty"`
	UnitPrice      float64 `json:"unit_price,omitempty"`
}

// ============================================================================
// Measurement Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListMeasurements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	grpcReq := &rapportev1.ListMeasurementsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if rid := r.URL.Query().Get("report_id"); rid != "" {
		grpcReq.ReportId = rid
	}

	resp, err := client.ListMeasurements(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleCreateMeasurement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createMeasurementRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateMeasurement(r.Context(), &rapportev1.CreateMeasurementRequest{
		TenantId:   tenantID.String(),
		ReportId:   req.ReportID,
		Title:      req.Title,
		Location:   req.Location,
		MeasuredBy: req.MeasuredBy,
		MeasuredAt: req.MeasuredAt,
		Notes:      req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleGetMeasurement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetMeasurement(r.Context(), &rapportev1.GetMeasurementRequest{
		TenantId:      tenantID.String(),
		MeasurementId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUpdateMeasurement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateMeasurementRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateMeasurement(r.Context(), &rapportev1.UpdateMeasurementRequest{
		TenantId:      tenantID.String(),
		MeasurementId: id,
		Title:         req.Title,
		Location:      req.Location,
		MeasuredBy:    req.MeasuredBy,
		MeasuredAt:    req.MeasuredAt,
		Notes:         req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteMeasurement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteMeasurement(r.Context(), &rapportev1.DeleteMeasurementRequest{
		TenantId:      tenantID.String(),
		MeasurementId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rr *RapporteRoutes) HandleAddMeasurementPosition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	measurementID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[addMeasurementPositionRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddMeasurementPosition(r.Context(), &rapportev1.AddMeasurementPositionRequest{
		TenantId:       tenantID.String(),
		MeasurementId:  measurementID,
		PositionNumber: int32(req.PositionNumber),
		Description:    req.Description,
		Unit:           req.Unit,
		Quantity:       req.Quantity,
		UnitPrice:      req.UnitPrice,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleDeleteMeasurementPosition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	posID, ok := validateUUIDParam(w, r, "pos_id")
	if !ok {
		return
	}

	_, err = client.DeleteMeasurementPosition(r.Context(), &rapportev1.DeleteMeasurementPositionRequest{
		TenantId:   tenantID.String(),
		PositionId: posID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Template request types
// ============================================================================

type createRapporteTemplateRequest struct {
	Name             string `json:"name"              validate:"required"`
	Description      string `json:"description,omitempty"`
	Category         string `json:"category,omitempty"`
	DefaultLinesJSON string `json:"default_lines_json,omitempty"`
}

type updateRapporteTemplateRequest struct {
	Name             string `json:"name"              validate:"required"`
	Description      string `json:"description,omitempty"`
	Category         string `json:"category,omitempty"`
	DefaultLinesJSON string `json:"default_lines_json,omitempty"`
	IsActive         bool   `json:"is_active"`
}

// ============================================================================
// Template Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	activeOnly := r.URL.Query().Get("active_only") == "true"

	resp, err := client.ListTemplates(r.Context(), &rapportev1.ListTemplatesRequest{
		TenantId:   tenantID.String(),
		ActiveOnly: activeOnly,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createRapporteTemplateRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTemplate(r.Context(), &rapportev1.CreateTemplateRequest{
		TenantId:         tenantID.String(),
		Name:             req.Name,
		Description:      req.Description,
		Category:         req.Category,
		DefaultLinesJson: req.DefaultLinesJSON,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleGetTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetTemplate(r.Context(), &rapportev1.GetTemplateRequest{
		TenantId:   tenantID.String(),
		TemplateId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateRapporteTemplateRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateTemplate(r.Context(), &rapportev1.UpdateTemplateRequest{
		TenantId:         tenantID.String(),
		TemplateId:       id,
		Name:             req.Name,
		Description:      req.Description,
		Category:         req.Category,
		DefaultLinesJson: req.DefaultLinesJSON,
		IsActive:         req.IsActive,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteTemplate(r.Context(), &rapportev1.DeleteTemplateRequest{
		TenantId:   tenantID.String(),
		TemplateId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
