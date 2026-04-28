package gateway

import (
	"encoding/json"
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

const rapportePlaceholderTenantID = "00000000-0000-0000-0000-000000000001"

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

			// State transitions
			r.With(middleware.RequirePermission("rapporte:report", "write")).Post("/submit", rr.HandleSubmitReport)
			r.With(middleware.RequirePermission("rapporte:report", "write")).Post("/approve", rr.HandleApproveReport)
			r.With(middleware.RequirePermission("rapporte:report", "write")).Post("/reject", rr.HandleRejectReport)

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
	})
}

// ============================================================================
// Request types
// ============================================================================

type createReportRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	AuthorID    string   `json:"author_id"`
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
	ReviewerID string `json:"reviewer_id"`
	ReviewNote string `json:"review_note"`
}

type addLineRequest struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
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

type rapporteUploadAttachmentRequest struct {
	LineID      *string `json:"line_id,omitempty"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   int64   `json:"size_bytes"`
	ObjectKey   string  `json:"object_key"`
	UploadedBy  string  `json:"uploaded_by"`
}

// ============================================================================
// Report Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListReports(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &rapportev1.ListReportsRequest{
		TenantId: rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleCreateReport(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	var req createReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		response.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.AuthorID == "" {
		response.Error(w, http.StatusBadRequest, "author_id is required")
		return
	}

	grpcReq := &rapportev1.CreateReportRequest{
		TenantId:    rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleGetReport(w http.ResponseWriter, r *http.Request) {
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
		TenantId: rapportePlaceholderTenantID,
		ReportId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUpdateReport(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &rapportev1.UpdateReportRequest{
		TenantId:    rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteReport(w http.ResponseWriter, r *http.Request) {
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
		TenantId: rapportePlaceholderTenantID,
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
		TenantId: rapportePlaceholderTenantID,
		ReportId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleApproveReport(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req approveRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ReviewerID == "" {
		response.Error(w, http.StatusBadRequest, "reviewer_id is required")
		return
	}

	resp, err := client.ApproveReport(r.Context(), &rapportev1.ApproveReportRequest{
		TenantId:   rapportePlaceholderTenantID,
		ReportId:   id,
		ReviewerId: req.ReviewerID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleRejectReport(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req approveRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ReviewerID == "" {
		response.Error(w, http.StatusBadRequest, "reviewer_id is required")
		return
	}

	resp, err := client.RejectReport(r.Context(), &rapportev1.RejectReportRequest{
		TenantId:   rapportePlaceholderTenantID,
		ReportId:   id,
		ReviewerId: req.ReviewerID,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Line Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleListLines(w http.ResponseWriter, r *http.Request) {
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
		TenantId: rapportePlaceholderTenantID,
		ReportId: reportID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleAddLine(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req addLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Description == "" {
		response.Error(w, http.StatusBadRequest, "description is required")
		return
	}

	resp, err := client.AddLine(r.Context(), &rapportev1.AddLineRequest{
		TenantId:    rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleUpdateLine(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	lineID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &rapportev1.UpdateLineRequest{
		TenantId:    rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleDeleteLine(w http.ResponseWriter, r *http.Request) {
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
		TenantId: rapportePlaceholderTenantID,
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
		TenantId: rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	reportID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req rapporteUploadAttachmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Filename == "" || req.ObjectKey == "" || req.UploadedBy == "" {
		response.Error(w, http.StatusBadRequest, "filename, object_key and uploaded_by are required")
		return
	}

	grpcReq := &rapportev1.UploadAttachmentRequest{
		TenantId:    rapportePlaceholderTenantID,
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
	response.JSON(w, http.StatusCreated, resp)
}

func (rr *RapporteRoutes) HandleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
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
		TenantId:     rapportePlaceholderTenantID,
		AttachmentId: attID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Stats & Export Handlers
// ============================================================================

func (rr *RapporteRoutes) HandleGetReportStats(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	resp, err := client.GetReportStats(r.Context(), &rapportev1.GetReportStatsRequest{
		TenantId: rapportePlaceholderTenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	client, err := rr.getClient()
	if err != nil {
		respondServiceUnavailable(w, rr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListPendingApprovals(r.Context(), &rapportev1.ListPendingApprovalsRequest{
		TenantId: rapportePlaceholderTenantID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (rr *RapporteRoutes) HandleExportPDF(w http.ResponseWriter, r *http.Request) {
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
		TenantId: rapportePlaceholderTenantID,
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
