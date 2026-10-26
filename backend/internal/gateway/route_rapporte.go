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

		// Additive RBAC guards (RequirePermissionAny keeps the legacy "write"
		// token valid while granting the capability-catalog.ts fine keys that
		// RapportePage.tsx/ReportDetailModal.tsx actually gate on). Reads
		// already match the fine key 1:1 and stay plain RequirePermission.
		// approve/reject already use the fine "approve" key directly (no
		// legacy "write" alternative existed for that action), so they are
		// left untouched. rapporte:measurement:* has no FE caller at all —
		// the "Aufmass" tab runs entirely on a local Zustand mock store, no
		// backend call — so the measurement routes stay legacy-only too.
		reportSubmit := requireAnyOf(rapporteEditKeys)
		lineRead := requireAnyOf(rapporteLineReadKeys)
		lineWrite := requireAnyOf(rapporteLineWriteKeys)
		attachmentRead := requireAnyOf(rapporteAttachmentReadKeys)
		attachmentWrite := requireAnyOf(rapporteAttachmentWriteKeys)
		measurementWrite := requireAnyOf(rapporteMeasurementWriteKeys)
		reportCreate := middleware.RequirePermissionAny([2]string{"rapporte:report", "write"}, [2]string{"rapporte:report", "create"})
		reportEdit := middleware.RequirePermissionAny([2]string{"rapporte:report", "write"}, [2]string{"rapporte:report", "edit"})
		reportDelete := middleware.RequirePermissionAny([2]string{"rapporte:report", "write"}, [2]string{"rapporte:report", "delete"})

		// Reports
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/reports", rr.HandleListReports)
		r.With(reportCreate).Post("/reports", rr.HandleCreateReport)

		r.Route("/reports/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/", rr.HandleGetReport)
			r.With(reportEdit).Patch("/", rr.HandleUpdateReport)
			r.With(reportDelete).Delete("/", rr.HandleDeleteReport)

			// Signature
			r.With(reportSubmit).Put("/signature", rr.HandleSaveReportSignature)

			// State transitions
			r.With(reportSubmit).Post("/submit", rr.HandleSubmitReport)
			// Bug #3: approve/reject require dedicated "approve" action, not generic "write"
			r.With(middleware.RequirePermission("rapporte:report", "approve")).Post("/approve", rr.HandleApproveReport)
			r.With(middleware.RequirePermission("rapporte:report", "approve")).Post("/reject", rr.HandleRejectReport)

			// Lines
			r.With(lineRead).Get("/lines", rr.HandleListLines)
			r.With(lineWrite).Post("/lines", rr.HandleAddLine)

			// Attachments
			r.With(attachmentRead).Get("/attachments", rr.HandleListAttachments)
			r.With(attachmentWrite).Post("/attachments", rr.HandleUploadAttachment)

			// Export
			r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/export/pdf", rr.HandleExportPDF)
		})

		// Lines (by line ID)
		r.Route("/lines/{id}", func(r chi.Router) {
			r.With(lineWrite).Patch("/", rr.HandleUpdateLine)
			r.With(lineWrite).Delete("/", rr.HandleDeleteLine)
		})

		// Attachments (by attachment ID)
		r.Route("/attachments/{id}", func(r chi.Router) {
			r.With(attachmentWrite).Delete("/", rr.HandleDeleteAttachment)
		})

		// Stats & approvals
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/stats", rr.HandleGetReportStats)
		r.With(middleware.RequirePermission("rapporte:report", "read")).Get("/pending", rr.HandleListPendingApprovals)

		// Measurements
		r.With(middleware.RequirePermission("rapporte:measurement", "read")).Get("/measurements", rr.HandleListMeasurements)
		r.With(measurementWrite).Post("/measurements", rr.HandleCreateMeasurement)
		r.Route("/measurements/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("rapporte:measurement", "read")).Get("/", rr.HandleGetMeasurement)
			r.With(measurementWrite).Put("/", rr.HandleUpdateMeasurement)
			r.With(measurementWrite).Delete("/", rr.HandleDeleteMeasurement)
			r.With(measurementWrite).Post("/positions", rr.HandleAddMeasurementPosition)
		})
		r.Route("/measurements/positions/{pos_id}", func(r chi.Router) {
			r.With(measurementWrite).Delete("/", rr.HandleDeleteMeasurementPosition)
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

// Key sets per capability. The fine report keys the member/manager presets hold
// stand in for the legacy line/attachment keys, so a guard and the "own" narrowing
// of its handler read the same pairs.
var (
	rapporteEditKeys             = [][2]string{{"rapporte:report", "write"}, {"rapporte:report", "edit"}}
	rapporteDeleteKeys           = [][2]string{{"rapporte:report", "write"}, {"rapporte:report", "delete"}}
	rapporteReadKeys             = [][2]string{{"rapporte:report", "read"}}
	rapporteLineReadKeys         = [][2]string{{"rapporte:line", "read"}, {"rapporte:report", "read"}}
	rapporteLineWriteKeys        = [][2]string{{"rapporte:line", "write"}, {"rapporte:report", "edit"}}
	rapporteAttachmentReadKeys   = [][2]string{{"rapporte:attachment", "read"}, {"rapporte:report", "read"}}
	rapporteAttachmentWriteKeys  = [][2]string{{"rapporte:attachment", "write"}, {"rapporte:report", "edit"}}
	rapporteMeasurementWriteKeys = [][2]string{{"rapporte:measurement", "write"}, {"rapporte:measurement", "manage"}}
	// Measurements are gated by their own keys, but they hang off a report, so the
	// report-edit scope narrows them as well.
	rapporteMeasurementScopeKeys = [][2]string{{"rapporte:measurement", "write"}, {"rapporte:measurement", "manage"}, {"rapporte:report", "edit"}}
	// Reading follows the report-read scope the same way (member: measurement:read
	// all, report:read own).
	rapporteMeasurementReadScopeKeys = [][2]string{{"rapporte:measurement", "read"}, {"rapporte:report", "read"}}
)

// requireAnyOf is RequirePermissionAny over a key set.
func requireAnyOf(keys [][2]string) func(http.Handler) http.Handler {
	return middleware.RequirePermissionAny(keys[0], keys[1:]...)
}

// restrictToAuthor reports whether the caller's grant under any of the given keys is
// scoped to "own", i.e. the service must only let them reach reports they wrote.
func restrictToAuthor(w http.ResponseWriter, r *http.Request, pairs ...[2]string) (restrict, ok bool) {
	owner, ok := ownerFilterForScopeAny(w, r, pairs...)
	return owner != nil, ok
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
	// Site-diary fields the report form collects.
	Weather      string   `json:"weather,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	WorkStart    string   `json:"work_start,omitempty"` // HH:MM
	WorkEnd      string   `json:"work_end,omitempty"`   // HH:MM
	BreakMinutes *int32   `json:"break_minutes,omitempty"`
	ProjectName  string   `json:"project_name,omitempty"`
}

type updateReportRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Lat         *float64 `json:"lat,omitempty"`
	Lon         *float64 `json:"lon,omitempty"`
	ReportDate  *string  `json:"report_date,omitempty"`
	// Site-diary fields; an omitted key keeps the stored value.
	Weather      *string  `json:"weather,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	WorkStart    *string  `json:"work_start,omitempty"`
	WorkEnd      *string  `json:"work_end,omitempty"`
	BreakMinutes *int32   `json:"break_minutes,omitempty"`
	ProjectName  *string  `json:"project_name,omitempty"`
}

// approveRejectRequest still names reviewer_id because the desktop sends it, but the
// handlers ignore it: the reviewer is the authenticated caller, never whoever the body names.
type approveRejectRequest struct {
	ReviewerID string `json:"reviewer_id"`
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

	page, pageSize := parsePaginationMax(r, 1, 50, listPageSizeMax)
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
	// A grant scoped to "own" overrides the client's author_id filter — after
	// this point the request can only ever ask for the caller's own reports.
	if ownerID, ok := ownerFilterForScope(w, r, "rapporte:report", "read"); !ok {
		return
	} else if ownerID != nil {
		grpcReq.AuthorId = ownerID
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
	// A caller who may only handle their own reports cannot file one under somebody
	// else's name — they could not touch it afterwards.
	restrict, ok := restrictToAuthor(w, r, rapporteEditKeys...)
	if !ok {
		return
	}
	if restrict {
		req.AuthorID = middleware.GetUserID(r.Context())
	}

	grpcReq := &rapportev1.CreateReportRequest{
		TenantId:    tenantID.String(),
		Title:       req.Title,
		Description: req.Description,
		AuthorId:    req.AuthorID,
		Lat:         req.Lat,
		Lon:         req.Lon,
		ReportDate:  req.ReportDate,

		Weather:      req.Weather,
		Temperature:  req.Temperature,
		WorkStart:    req.WorkStart,
		WorkEnd:      req.WorkEnd,
		BreakMinutes: req.BreakMinutes,
		ProjectName:  req.ProjectName,
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

	restrict, ok := restrictToAuthor(w, r, rapporteReadKeys...)
	if !ok {
		return
	}

	resp, err := client.GetReport(r.Context(), &rapportev1.GetReportRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         id,
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

		Weather:      req.Weather,
		Temperature:  req.Temperature,
		WorkStart:    req.WorkStart,
		WorkEnd:      req.WorkEnd,
		BreakMinutes: req.BreakMinutes,
		ProjectName:  req.ProjectName,
	}

	restrict, ok := restrictToAuthor(w, r, rapporteEditKeys...)
	if !ok {
		return
	}
	grpcReq.RestrictToAuthor = restrict

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

	restrict, ok := restrictToAuthor(w, r, rapporteDeleteKeys...)
	if !ok {
		return
	}

	_, err = client.DeleteReport(r.Context(), &rapportev1.DeleteReportRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         id,
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

	restrict, ok := restrictToAuthor(w, r, rapporteEditKeys...)
	if !ok {
		return
	}

	resp, err := client.SubmitReport(r.Context(), &rapportev1.SubmitReportRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         id,
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
	reviewerID := middleware.GetUserID(r.Context())
	if reviewerID == "" {
		response.Error(w, http.StatusUnauthorized, "missing user in token")
		return
	}

	resp, err := client.ApproveReport(r.Context(), &rapportev1.ApproveReportRequest{
		TenantId:   tenantID.String(),
		ReportId:   id,
		ReviewerId: reviewerID,
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
	reviewerID := middleware.GetUserID(r.Context())
	if reviewerID == "" {
		response.Error(w, http.StatusUnauthorized, "missing user in token")
		return
	}

	resp, err := client.RejectReport(r.Context(), &rapportev1.RejectReportRequest{
		TenantId:   tenantID.String(),
		ReportId:   id,
		ReviewerId: reviewerID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteLineReadKeys...)
	if !ok {
		return
	}

	resp, err := client.ListLines(r.Context(), &rapportev1.ListLinesRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         reportID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteLineWriteKeys...)
	if !ok {
		return
	}

	resp, err := client.AddLine(r.Context(), &rapportev1.AddLineRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         reportID,
		Description:      req.Description,
		Quantity:         req.Quantity,
		Unit:             req.Unit,
		Notes:            req.Notes,
		Position:         req.Position,
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

	restrict, ok := restrictToAuthor(w, r, rapporteLineWriteKeys...)
	if !ok {
		return
	}
	grpcReq.RestrictToAuthor = restrict

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

	restrict, ok := restrictToAuthor(w, r, rapporteLineWriteKeys...)
	if !ok {
		return
	}

	_, err = client.DeleteLine(r.Context(), &rapportev1.DeleteLineRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		LineId:           lineID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteAttachmentReadKeys...)
	if !ok {
		return
	}
	grpcReq.RestrictToAuthor = restrict

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

	// The uploader is the authenticated caller, not whoever the body names.
	uploadedBy := middleware.GetUserID(r.Context())
	if uploadedBy == "" {
		response.Error(w, http.StatusUnauthorized, "missing user in token")
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
		UploadedBy:  uploadedBy,
	}

	restrict, ok := restrictToAuthor(w, r, rapporteAttachmentWriteKeys...)
	if !ok {
		return
	}
	grpcReq.RestrictToAuthor = restrict

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

	restrict, ok := restrictToAuthor(w, r, rapporteAttachmentWriteKeys...)
	if !ok {
		return
	}

	_, err = client.DeleteAttachment(r.Context(), &rapportev1.DeleteAttachmentRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		AttachmentId:     attID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteEditKeys...)
	if !ok {
		return
	}

	resp, err := client.SaveSignature(r.Context(), &rapportev1.SaveReportSignatureRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         reportID,
		SignatureData:    req.SignatureData,
		SignedBy:         req.SignedBy,
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
	response.ProtoFull(w, http.StatusOK, resp)
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

	page, pageSize := parsePaginationMax(r, 1, 50, listPageSizeMax)

	// Same key as the report list, so the same narrowing applies: at scope
	// "own" the approval queue holds only the caller's own submissions.
	ownerID, ok := ownerFilterForScope(w, r, "rapporte:report", "read")
	if !ok {
		return
	}

	resp, err := client.ListPendingApprovals(r.Context(), &rapportev1.ListPendingApprovalsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
		AuthorId: ownerID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteReadKeys...)
	if !ok {
		return
	}

	resp, err := client.ExportPDF(r.Context(), &rapportev1.ExportPDFRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         reportID,
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementReadScopeKeys...)
	if !ok {
		return
	}

	page, pageSize := parsePaginationMax(r, 1, 50, listPageSizeMax)
	grpcReq := &rapportev1.ListMeasurementsRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		Page:             int32(page),
		PageSize:         int32(pageSize),
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementScopeKeys...)
	if !ok {
		return
	}

	resp, err := client.CreateMeasurement(r.Context(), &rapportev1.CreateMeasurementRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		ReportId:         req.ReportID,
		Title:            req.Title,
		Location:         req.Location,
		MeasuredBy:       req.MeasuredBy,
		MeasuredAt:       req.MeasuredAt,
		Notes:            req.Notes,
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementReadScopeKeys...)
	if !ok {
		return
	}

	resp, err := client.GetMeasurement(r.Context(), &rapportev1.GetMeasurementRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		MeasurementId:    id,
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementScopeKeys...)
	if !ok {
		return
	}

	resp, err := client.UpdateMeasurement(r.Context(), &rapportev1.UpdateMeasurementRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		MeasurementId:    id,
		Title:            req.Title,
		Location:         req.Location,
		MeasuredBy:       req.MeasuredBy,
		MeasuredAt:       req.MeasuredAt,
		Notes:            req.Notes,
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementScopeKeys...)
	if !ok {
		return
	}

	_, err = client.DeleteMeasurement(r.Context(), &rapportev1.DeleteMeasurementRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		MeasurementId:    id,
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

	restrict, ok := restrictToAuthor(w, r, rapporteMeasurementScopeKeys...)
	if !ok {
		return
	}

	resp, err := client.AddMeasurementPosition(r.Context(), &rapportev1.AddMeasurementPositionRequest{
		TenantId:         tenantID.String(),
		RestrictToAuthor: restrict,
		MeasurementId:    measurementID,
		PositionNumber:   int32(req.PositionNumber),
		Description:      req.Description,
		Unit:             req.Unit,
		Quantity:         req.Quantity,
		UnitPrice:        req.UnitPrice,
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

	page, pageSize := parsePaginationMax(r, 1, 50, listPageSizeMax)
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
