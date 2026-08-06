package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/clientctx"
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

		// Additive RBAC guards (RequirePermissionAny keeps the legacy
		// "read"/"write" tokens valid while granting the capability-catalog.ts
		// fine keys FormularePage.tsx actually gates on). Reads/submission
		// writes already match their fine key 1:1 ("formulare:submissions:read"
		// / "formulare:submissions:write") and stay plain RequirePermission.
		// formulare:share:manage got its routes in B8 (see shareManage below):
		// {BASE}/schemas/{id}/share-links and {BASE}/share-links/{id}, the two
		// paths useFormulare.ts was already calling. Webhooks stay legacy-only:
		// the catalog note says the webhook UI is an unmounted stub with no key
		// issued yet.
		schemasCreate := middleware.RequirePermissionAny([2]string{"formulare:schemas", "write"}, [2]string{"formulare:schemas", "create"})
		// PATCH carries both content edits and the draft/active/closed/archived
		// lifecycle toggle (FormularePage.tsx canSchemasPublish gates the same
		// updateSchema call as canSchemasEdit), so it needs both fine keys.
		schemasUpdate := middleware.RequirePermissionAny([2]string{"formulare:schemas", "write"}, [2]string{"formulare:schemas", "edit"}, [2]string{"formulare:schemas", "publish"})
		schemasDelete := middleware.RequirePermissionAny([2]string{"formulare:schemas", "write"}, [2]string{"formulare:schemas", "delete"})
		exportRun := middleware.RequirePermissionAny([2]string{"formulare:submissions", "read"}, [2]string{"formulare:export", "run"})
		// formulare:share:manage now HAS backend routes (B8): minting, listing
		// and cutting the public fill-out links the note above said were
		// missing. Plain RequirePermission -- the fine key is the only key
		// there ever was for this, no legacy token to keep valid.
		shareManage := middleware.RequirePermission("formulare:share", "manage")

		// Schemas
		r.Route("/schemas", func(r chi.Router) {
			r.With(middleware.RequirePermission("formulare:schemas", "read")).Get("/", fr.HandleListFormSchemas)
			r.With(schemasCreate).Post("/", fr.HandleCreateFormSchema)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("formulare:schemas", "read")).Get("/", fr.HandleGetFormSchema)
				r.With(schemasUpdate).Patch("/", fr.HandleUpdateFormSchema)
				r.With(schemasDelete).Delete("/", fr.HandleDeleteFormSchema)
				// Duplicate is FE-gated by canSchemasCreate (formulare.actions.duplizieren
				// sits in the canSchemasCreate action group), not schemas:edit.
				r.With(schemasCreate).Post("/duplicate", fr.HandleDuplicateFormSchema)
				r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/stats", fr.HandleGetFormStats)

				// Submissions nested under schema
				r.Route("/submissions", func(r chi.Router) {
					r.With(middleware.RequirePermission("formulare:submissions", "write")).Post("/", fr.HandleCreateSubmission)
					r.With(middleware.RequirePermission("formulare:submissions", "read")).Get("/", fr.HandleListSubmissions)
					r.With(exportRun).Get("/export", fr.HandleExportSubmissions)
				})

				// Share links nested under schema (B8). formulare:share:manage
				// is already seeded (000256) for admin and manager; it had no
				// backend route until now.
				r.Route("/share-links", func(r chi.Router) {
					r.With(shareManage).Get("/", fr.HandleListShareLinks)
					r.With(shareManage).Post("/", fr.HandleCreateShareLink)
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

		// Share links flat routes (B8). DELETE is a soft revoke, see
		// formulare.Service.RevokeShareLink.
		r.With(shareManage).Delete("/share-links/{id}", fr.HandleRevokeShareLink)

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

// RegisterPublicRoutes mounts the unauthenticated fill-out of a shared form.
// Called from cmd/gateway/main.go on the root router, OUTSIDE the registrars
// loop and therefore outside any authMiddleware group -- the token is the
// whole credential, and a skip-list inside the auth middleware would be a far
// easier thing to widen by accident.
//
// publicRateLimit must be the stricter public limiter, not the global one:
// this is the only formulare path that answers without a JWT, and it WRITES.
// Per-IP throttling is what stands between a scraper and both an unbounded run
// of token guesses and an unbounded pile of junk submissions.
func (fr *FormulareRoutes) RegisterPublicRoutes(r chi.Router, publicRateLimit func(http.Handler) http.Handler) {
	if !fr.flags.IsEnabled("modules.formulare") {
		return
	}

	r.Group(func(r chi.Router) {
		r.Use(publicRateLimit)
		r.Post("/api/v1/public/formulare/submit/{token}", fr.HandleSubmitByShareToken)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createFormSchemaRequest struct {
	Title          string  `json:"title"                validate:"required"`
	Description    string  `json:"description,omitempty"`
	Fields         []byte  `json:"fields,omitempty"`
	IsTemplate     bool    `json:"is_template,omitempty"`
	IsPublic       bool    `json:"is_public,omitempty"`
	PageCount      int32   `json:"page_count,omitempty"`
	IntakeTargetID *string `json:"intake_target_id,omitempty"`
}

type updateFormSchemaRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Fields         []byte  `json:"fields,omitempty"`
	IsTemplate     *bool   `json:"is_template,omitempty"`
	IsPublic       *bool   `json:"is_public,omitempty"`
	PageCount      *int32  `json:"page_count,omitempty"`
	IntakeTargetID *string `json:"intake_target_id,omitempty"`
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
	response.Proto(w, http.StatusOK, resp)
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
		TenantId:       tenantID.String(),
		Title:          req.Title,
		Description:    req.Description,
		Fields:         req.Fields,
		IsTemplate:     req.IsTemplate,
		IsPublic:       req.IsPublic,
		PageCount:      req.PageCount,
		CreatedBy:      userID,
		IntakeTargetId: req.IntakeTargetID,
	}

	resp, err := client.CreateFormSchema(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
		TenantId:       tenantID.String(),
		SchemaId:       id,
		Title:          req.Title,
		Description:    req.Description,
		Fields:         req.Fields,
		IsTemplate:     req.IsTemplate,
		IsPublic:       req.IsPublic,
		PageCount:      req.PageCount,
		EditorId:       userID,
		IntakeTargetId: req.IntakeTargetID,
	}

	resp, err := client.UpdateFormSchema(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusCreated, resp)
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

	// The submission is stored; only now do we try to create the record the
	// form is bound to. Order matters: a failing target must never cost the
	// submitter their answers, so the dispatch error is logged and the
	// response stays 201 (see runIntakeDispatch).
	fr.dispatchIntake(r.Context(), client, tenantID, schemaID, resp.GetSubmission().GetId(), req.Answers, intakeOrigin{
		Channel:     intakeChannelSelfService,
		RequesterID: userID,
	})

	response.Proto(w, http.StatusCreated, resp)
}

// dispatchIntake creates the module record bound to the submitted schema, if
// the schema declares one. Errors never reach the caller: the submission is
// already safe, and an unreachable Helpdesk is not a reason to tell the
// submitter their form failed. It is logged loudly instead -- a form whose
// intake silently stops producing tickets is exactly the failure this run is
// closing elsewhere, so it must be visible in the log.
func (fr *FormulareRoutes) dispatchIntake(
	ctx context.Context,
	client formularev1.FormulareServiceClient,
	tenantID uuid.UUID,
	schemaID, submissionID string,
	answers []byte,
	origin intakeOrigin,
) {
	lookup := func(ctx context.Context) (string, []byte, error) {
		schemaResp, err := client.GetFormSchema(ctx, &formularev1.GetFormSchemaRequest{
			TenantId: tenantID.String(),
			SchemaId: schemaID,
		})
		if err != nil {
			return "", nil, err
		}
		schema := schemaResp.GetSchema()
		return schema.GetIntakeTargetId(), schema.GetFields(), nil
	}

	recordID, err := runIntakeDispatch(ctx, fr.registry, lookup, tenantID, answers, origin)
	if err != nil {
		slog.Error("formulare intake dispatch failed",
			"submission_id", submissionID,
			"form_schema_id", schemaID,
			"tenant_id", tenantID,
			"error", err,
		)
		return
	}
	if recordID != "" {
		slog.Info("formulare intake dispatched",
			"submission_id", submissionID,
			"form_schema_id", schemaID,
			"tenant_id", tenantID,
			"record_id", recordID,
		)
	}
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusCreated, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
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
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Share Links (B8)
// ============================================================================

// maxPublicSubmitBody caps the public fill-out request body. A form's answers
// are JSON text, and 256 KiB is far past any hand-typed form; the service
// enforces the same ceiling on the decoded answers, so the limit holds even
// for a caller that reaches the RPC another way.
const maxPublicSubmitBody = 256 << 10

type createFormShareLinkRequest struct {
	// ExpiresAt is RFC 3339; nil = never expires.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// MaxSubmissions nil or 0 = unlimited.
	MaxSubmissions *int32 `json:"max_submissions,omitempty" validate:"omitempty,min=1"`
}

type publicSubmitRequest struct {
	Answers []byte `json:"answers" validate:"required"`
}

func (fr *FormulareRoutes) HandleListShareLinks(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.ListFormShareLinks(r.Context(), &formularev1.ListFormShareLinksRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (fr *FormulareRoutes) HandleCreateShareLink(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createFormShareLinkRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &formularev1.CreateFormShareLinkRequest{
		TenantId:     tenantID.String(),
		FormSchemaId: schemaID,
	}
	if req.ExpiresAt != nil {
		grpcReq.ExpiresAt = timestamppb.New(*req.ExpiresAt)
	}
	if req.MaxSubmissions != nil {
		grpcReq.MaxSubmissions = *req.MaxSubmissions
	}
	if userID := middleware.GetUserID(r.Context()); userID != "" {
		grpcReq.CreatedBy = &userID
	}

	resp, err := client.CreateFormShareLink(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (fr *FormulareRoutes) HandleRevokeShareLink(w http.ResponseWriter, r *http.Request) {
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

	linkID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.RevokeFormShareLink(r.Context(), &formularev1.RevokeFormShareLinkRequest{
		TenantId:    tenantID.String(),
		ShareLinkId: linkID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "share link revoked"})
}

// HandleSubmitByShareToken serves the unauthenticated fill-out of a form
// through a share link.
//
// POST, not GET, for the same reasons the shared-report read and the CSAT
// redemption are: the token must not land in access logs, browser history or
// Referer headers, and this writes -- no prefetch may be free to trigger it.
//
// Every rejection that concerns the token itself -- missing, over-long,
// unknown, expired, revoked, used up, or belonging to a form that is no longer
// public -- comes back as the same 404, so the route cannot be used to find
// out which tokens exist. A malformed or oversized body is the caller's own
// error and stays 400: it says nothing about the token.
func (fr *FormulareRoutes) HandleSubmitByShareToken(w http.ResponseWriter, r *http.Request) {
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" || len(token) > 128 {
		response.Error(w, http.StatusNotFound, "share link not found")
		return
	}

	// Body cap before the decode, not after: an unauthenticated writer must
	// not be able to make the gateway buffer a megabyte to find out it is
	// invalid.
	r.Body = http.MaxBytesReader(w, r.Body, maxPublicSubmitBody)

	req, ok := decodeAndValidate[publicSubmitRequest](w, r)
	if !ok {
		return
	}

	// The submitter's IP is the only forensic handle on an unauthenticated
	// write. It is taken from the context middleware.ClientInfo filled under
	// the configured proxy-trust setting, never from the raw header, so a
	// spoofed X-Forwarded-For cannot launder the record.
	submitIP := clientctx.From(r.Context()).IP

	grpcReq := &formularev1.SubmitFormByShareTokenRequest{
		Token:   token,
		Answers: req.Answers,
	}
	if submitIP != "" {
		grpcReq.IpAddress = &submitIP
	}

	resp, err := client.SubmitFormByShareToken(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// The submission is stored; only now the intake dispatch, exactly as on
	// the authenticated route. The tenant comes from the response, i.e. from
	// the token the service resolved -- the gateway never guesses one, and the
	// dispatch stays tenant-scoped from here on.
	tenantID, parseErr := uuid.Parse(resp.GetTenantId())
	if parseErr != nil {
		slog.Error("formulare public submission returned an unusable tenant",
			"submission_id", resp.GetSubmissionId(),
			"error", parseErr,
		)
	} else {
		external := true
		fr.dispatchIntake(
			withGatewayTenant(r.Context(), tenantID),
			client, tenantID, resp.GetFormSchemaId(), resp.GetSubmissionId(), req.Answers,
			intakeOrigin{
				Channel: intakeChannelExternal,
				// No RequesterID: nobody is logged in. The requester's identity
				// can only come from the form's own requester_email /
				// requester_name field roles -- if the author bound an intake
				// target without them, the target refuses the record and the
				// dispatch failure is logged. The submission is safe either way.
				RequesterIsExternal: &external,
			},
		)
	}

	// Only the submission id goes back over the wire. The proto also carries
	// tenant_id and form_schema_id because the dispatch above needs them, but
	// handing an unauthenticated visitor the tenant UUID would tell them which
	// installation they just wrote into -- the same reason the shared wiki
	// article and the shared report answer through their own narrow wire types
	// instead of the raw proto.
	response.JSON(w, http.StatusCreated, map[string]string{
		"submission_id": resp.GetSubmissionId(),
	})
}

// withGatewayTenant attaches the tenant a share token resolved to, so the
// outbound gRPC interceptor carries it on the dispatch calls. This is the one
// place in the gateway a tenant comes from anywhere but the authenticated
// request context, and it is only ever the tenant the service itself reported.
func withGatewayTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}
