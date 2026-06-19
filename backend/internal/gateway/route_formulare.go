package gateway

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

// FormulareRoutes handles HTTP routes for the Formulare (forms) module.
type FormulareRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewFormulareRoutes creates a new FormulareRoutes with the given service registry and feature flags.
func NewFormulareRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *FormulareRoutes {
	return &FormulareRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (fr *FormulareRoutes) ServiceName() string { return "formulare" }

// getClient lazily obtains a gRPC client for the formulare service.
func (fr *FormulareRoutes) getClient() (formularev1.FormulareServiceClient, error) {
	conn, err := fr.registry.GetConnection("formulare")
	if err != nil {
		return nil, err
	}
	return formularev1.NewFormulareServiceClient(conn), nil
}

// RegisterRoutes mounts all Formulare HTTP routes behind the feature flag modules.formulare.
// Routes are only registered if the flag is enabled.
func (fr *FormulareRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !fr.flags.IsEnabled("modules.formulare") {
		return
	}

	r.Route("/api/v1/formulare", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Schemas
		r.Route("/schemas", func(r chi.Router) {
			r.With(middleware.RequirePermission("formulare:schemas", "read")).Get("/", fr.HandleListFormSchemas)
			r.With(middleware.RequirePermission("formulare:schemas", "write")).Post("/", fr.HandleCreateFormSchema)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("formulare:schemas", "read")).Get("/", fr.HandleGetFormSchema)
				r.With(middleware.RequirePermission("formulare:schemas", "write")).Patch("/", fr.HandleUpdateFormSchema)
				r.With(middleware.RequirePermission("formulare:schemas", "write")).Delete("/", fr.HandleDeleteFormSchema)
				r.With(middleware.RequirePermission("formulare:schemas", "write")).Post("/duplicate", fr.HandleDuplicateFormSchema)
				r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/stats", fr.HandleGetFormStats)

				// Submissions nested under schema
				r.Route("/submissions", func(r chi.Router) {
					r.With(middleware.RequirePermission("formulare:submissions", "write")).Post("/", fr.HandleCreateSubmission)
					r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/", fr.HandleListSubmissions)
					r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/export", fr.HandleExportSubmissions)
				})

				// Webhooks nested under schema
				r.Route("/webhooks", func(r chi.Router) {
					r.With(middleware.RequirePermission("formulare:webhooks", "read")).Get("/", fr.HandleListWebhooks)
					r.With(middleware.RequirePermission("formulare:webhooks", "write")).Post("/", fr.HandleCreateWebhook)
				})
			})
		})

		// Submissions flat routes
		r.Route("/submissions/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/", fr.HandleGetSubmission)
			r.With(middleware.RequirePermission("formulare:submissions", "write")).Patch("/", fr.HandleUpdateSubmissionStatus)
		})

		// Webhooks flat routes
		r.Route("/webhooks/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("formulare:webhooks", "read")).Get("/", fr.HandleGetWebhook)
			r.With(middleware.RequirePermission("formulare:webhooks", "write")).Patch("/", fr.HandleUpdateWebhook)
			r.With(middleware.RequirePermission("formulare:webhooks", "write")).Delete("/", fr.HandleDeleteWebhook)

			r.With(middleware.RequirePermission("formulare:webhooks", "read")).Get("/deliveries", fr.HandleListWebhookDeliveriesForWebhook)
		})

		// Global delivery observability
		r.With(middleware.RequirePermission("formulare:webhooks", "read")).Get("/deliveries", fr.HandleListWebhookDeliveries)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createFormSchemaRequest struct {
	Title       string `json:"title"                validate:"required"`
	Description string `json:"description,omitempty"`
	Fields      []byte `json:"fields,omitempty"`
	IsTemplate  bool   `json:"is_template,omitempty"`
	IsPublic    bool   `json:"is_public,omitempty"`
	PageCount   int32  `json:"page_count,omitempty"`
}

type updateFormSchemaRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Fields      []byte  `json:"fields,omitempty"`
	IsTemplate  *bool   `json:"is_template,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
	PageCount   *int32  `json:"page_count,omitempty"`
}

type duplicateFormSchemaRequest struct {
	NewTitle *string `json:"new_title,omitempty"`
}

type createSubmissionRequest struct {
	Answers     []byte  `json:"answers,omitempty"`
	SubmittedBy *string `json:"submitted_by,omitempty"`
	IPAddress   *string `json:"ip_address,omitempty"`
}

type updateSubmissionStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=new read archived"`
}

type createWebhookRequest struct {
	URL    string   `json:"url"            validate:"required,url"`
	Secret *string  `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
	Active bool     `json:"active,omitempty"`
}

type updateWebhookRequest struct {
	URL       *string  `json:"url,omitempty"`
	Secret    *string  `json:"secret,omitempty"`
	Events    []string `json:"events,omitempty"`
	EventsSet bool     `json:"events_set,omitempty"`
	Active    *bool    `json:"active,omitempty"`
}

// ============================================================================
// Schema Handlers
// ============================================================================

func (fr *FormulareRoutes) HandleListFormSchemas(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	q := r.URL.Query()
	page, pageSize := parsePagination(r, 1, 20)
	offset := int32((page - 1) * pageSize)

	grpcReq := &formularev1.ListFormSchemasRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Limit:    int32(pageSize),
		Offset:   offset,
	}

	resp, err := client.ListFormSchemas(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleCreateFormSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createFormSchemaRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &formularev1.CreateFormSchemaRequest{
		TenantId:    tenantID.String(),
		Title:       req.Title,
		Description: req.Description,
		Fields:      req.Fields,
		IsTemplate:  req.IsTemplate,
		IsPublic:    req.IsPublic,
		PageCount:   req.PageCount,
		CreatedBy:   userID,
	}

	resp, err := client.CreateFormSchema(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FormulareRoutes) HandleGetFormSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetFormSchema(r.Context(), &formularev1.GetFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleUpdateFormSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateFormSchemaRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &formularev1.UpdateFormSchemaRequest{
		TenantId:    tenantID.String(),
		SchemaId:    id,
		Title:       req.Title,
		Description: req.Description,
		Fields:      req.Fields,
		IsTemplate:  req.IsTemplate,
		IsPublic:    req.IsPublic,
		PageCount:   req.PageCount,
		EditorId:    userID,
	}

	resp, err := client.UpdateFormSchema(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleDeleteFormSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteFormSchema(r.Context(), &formularev1.DeleteFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (fr *FormulareRoutes) HandleDuplicateFormSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[duplicateFormSchemaRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.DuplicateFormSchema(r.Context(), &formularev1.DuplicateFormSchemaRequest{
		TenantId: tenantID.String(),
		SchemaId: id,
		NewTitle: req.NewTitle,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

// ============================================================================
// Submission Handlers
// ============================================================================

func (fr *FormulareRoutes) HandleCreateSubmission(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	schemaID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createSubmissionRequest](w, r)
	if !ok {
		return
	}

	// Use authenticated user as submitter if not explicitly provided
	submittedBy := req.SubmittedBy
	if submittedBy == nil {
		submittedBy = &userID
	}

	grpcReq := &formularev1.CreateSubmissionRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
		Answers:      req.Answers,
		SubmittedBy:  submittedBy,
		IpAddress:    req.IPAddress,
	}

	resp, err := client.CreateSubmission(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FormulareRoutes) HandleListSubmissions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	schemaID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	offset := int32((page - 1) * pageSize)

	grpcReq := &formularev1.ListSubmissionsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: &schemaID,
		Limit:        int32(pageSize),
		Offset:       offset,
	}

	resp, err := client.ListSubmissions(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleGetSubmission(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetSubmission(r.Context(), &formularev1.GetSubmissionRequest{
		TenantId:     tenantID.String(),
		SubmissionId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleUpdateSubmissionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateSubmissionStatusRequest](w, r)
	if !ok {
		return
	}

	var status formularev1.FormSubmissionStatus
	switch req.Status {
	case "new":
		status = formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_NEW
	case "read":
		status = formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ
	case "archived":
		status = formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_ARCHIVED
	}

	resp, err := client.UpdateSubmissionStatus(r.Context(), &formularev1.UpdateSubmissionStatusRequest{
		TenantId:     tenantID.String(),
		SubmissionId: id,
		Status:       status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// HandleExportSubmissions exports submissions for a schema as CSV or XLSX.
// The format is taken from the query parameter ?format=csv|xlsx; defaults to "csv".
// Content-Disposition uses safe filename quoting to prevent header injection.
func (fr *FormulareRoutes) HandleExportSubmissions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	schemaID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	format := r.URL.Query().Get("format")
	var exportFormat formularev1.ExportFormat
	switch format {
	case "xlsx":
		exportFormat = formularev1.ExportFormat_EXPORT_FORMAT_XLSX
	default:
		exportFormat = formularev1.ExportFormat_EXPORT_FORMAT_CSV
		format = "csv"
	}

	grpcReq := &formularev1.ExportSubmissionsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
		Format:       exportFormat,
	}

	resp, err := client.ExportSubmissions(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	mimeType := resp.GetMimeType()
	if mimeType == "" {
		if format == "xlsx" {
			mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		} else {
			mimeType = "text/csv; charset=utf-8"
		}
	}
	filename := resp.GetFilename()
	if filename == "" {
		filename = "submissions." + format
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetContent())
}

// ============================================================================
// Webhook Handlers
// ============================================================================

func (fr *FormulareRoutes) HandleListWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	schemaID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListWebhooks(r.Context(), &formularev1.ListWebhooksRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	schemaID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createWebhookRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &formularev1.CreateWebhookRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
		Url:          req.URL,
		Secret:       req.Secret,
		Events:       req.Events,
		Active:       req.Active,
	}

	resp, err := client.CreateWebhook(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FormulareRoutes) HandleGetWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetWebhook(r.Context(), &formularev1.GetWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateWebhookRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &formularev1.UpdateWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: id,
		Url:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		EventsSet: req.EventsSet,
		Active:    req.Active,
	}

	resp, err := client.UpdateWebhook(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteWebhook(r.Context(), &formularev1.DeleteWebhookRequest{
		TenantId:  tenantID.String(),
		WebhookId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Delivery Handlers
// ============================================================================

// HandleListWebhookDeliveriesForWebhook lists deliveries filtered to a specific webhook ID.
// Route: GET /webhooks/{id}/deliveries
func (fr *FormulareRoutes) HandleListWebhookDeliveriesForWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	webhookID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	offset := int32((page - 1) * pageSize)

	grpcReq := &formularev1.ListWebhookDeliveriesRequest{
		TenantId:  tenantID.String(),
		WebhookId: &webhookID,
		Limit:     int32(pageSize),
		Offset:    offset,
	}

	resp, err := client.ListWebhookDeliveries(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// HandleGetFormStats returns aggregated submission statistics for a form schema.
// Route: GET /schemas/{id}/stats
func (fr *FormulareRoutes) HandleGetFormStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetFormStats(r.Context(), &formularev1.GetFormStatsRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// HandleListWebhookDeliveries lists deliveries with optional filters via query params.
// Route: GET /deliveries?webhook_id=...&submission_id=...&status=...
func (fr *FormulareRoutes) HandleListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	q := r.URL.Query()
	page, pageSize := parsePagination(r, 1, 20)
	offset := int32((page - 1) * pageSize)

	grpcReq := &formularev1.ListWebhookDeliveriesRequest{
		TenantId: tenantID.String(),
		Limit:    int32(pageSize),
		Offset:   offset,
	}

	if wid := q.Get("webhook_id"); wid != "" {
		grpcReq.WebhookId = &wid
	}
	if sid := q.Get("submission_id"); sid != "" {
		grpcReq.SubmissionId = &sid
	}
	if statusStr := q.Get("status"); statusStr != "" {
		statusInt, err := strconv.Atoi(statusStr)
		if err == nil {
			s := formularev1.WebhookDeliveryStatus(statusInt)
			grpcReq.Status = &s
		}
	}

	resp, err := client.ListWebhookDeliveries(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}
