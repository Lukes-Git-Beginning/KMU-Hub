package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

// BerichteRoutes handles HTTP routes for the Berichte (reports) module.
type BerichteRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewBerichteRoutes creates a new BerichteRoutes with the given service registry and feature flags.
func NewBerichteRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *BerichteRoutes {
	return &BerichteRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (br *BerichteRoutes) ServiceName() string { return "berichte" }

// getClient lazily obtains a gRPC client for the berichte service.
func (br *BerichteRoutes) getClient() (berichtev1.BerichteServiceClient, error) {
	conn, err := br.registry.GetConnection("berichte")
	if err != nil {
		return nil, err
	}
	return berichtev1.NewBerichteServiceClient(conn), nil
}

// RegisterRoutes mounts all Berichte HTTP routes behind the feature flag modules.berichte.
// Routes are only registered if the flag is enabled.
func (br *BerichteRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !br.flags.IsEnabled("modules.berichte") {
		return
	}

	r.Route("/api/v1/berichte", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Definitions
		r.Route("/definitions", func(r chi.Router) {
			r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/", br.HandleListDefinitions)
			r.With(middleware.RequirePermission("berichte:reports", "write")).Post("/", br.HandleCreateDefinition)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/", br.HandleGetDefinition)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Patch("/", br.HandleUpdateDefinition)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Delete("/", br.HandleDeleteDefinition)

				r.With(middleware.RequirePermission("berichte:reports", "read")).Post("/run", br.HandleRunReport)
				r.With(middleware.RequirePermission("berichte:reports", "read")).Post("/export", br.HandleExportReport)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Delete("/cache", br.HandleInvalidateCache)
			})
		})

		// Schedules
		r.Route("/schedules", func(r chi.Router) {
			r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/", br.HandleListSchedules)
			r.With(middleware.RequirePermission("berichte:reports", "write")).Post("/", br.HandleCreateSchedule)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("berichte:reports", "write")).Patch("/", br.HandleUpdateSchedule)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Delete("/", br.HandleDeleteSchedule)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Post("/toggle", br.HandleToggleSchedule)
			})
		})

		// Documents (multi-page authoring)
		r.Route("/documents", func(r chi.Router) {
			r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/", br.HandleListDocuments)
			r.With(middleware.RequirePermission("berichte:reports", "write")).Post("/", br.HandleCreateDocument)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/", br.HandleGetDocument)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Patch("/", br.HandleUpdateDocument)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Delete("/", br.HandleDeleteDocument)
				r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/export/pdf", br.HandleExportDocumentPDF)

				// External share links. Minting one hands out an
				// unauthenticated read path, so it sits behind the same
				// "write" permission as changing the document itself.
				r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/shares", br.HandleListReportShares)
				r.With(middleware.RequirePermission("berichte:reports", "write")).Post("/shares", br.HandleCreateReportShare)
			})
		})

		r.With(middleware.RequirePermission("berichte:reports", "write")).
			Delete("/shares/{shareId}", br.HandleRevokeReportShare)

		// KPIs
		r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/kpis", br.HandleGetDashboardKPIs)

		// Templates (static starter structures)
		r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/templates", br.HandleListTemplates)
	})
}

// RegisterPublicRoutes mounts the unauthenticated read of a shared report.
// Called from cmd/gateway/main.go OUTSIDE the registrars loop, directly on r,
// following the public booking-routes pattern.
//
// publicRateLimit must be the stricter public limiter, not the global one: this
// is the only path in the module that answers without a JWT, so per-IP
// throttling is the only thing standing between a scraper and an unbounded
// series of attempts.
func (br *BerichteRoutes) RegisterPublicRoutes(r chi.Router, publicRateLimit func(http.Handler) http.Handler) {
	if !br.flags.IsEnabled("modules.berichte") {
		return
	}

	r.Group(func(r chi.Router) {
		r.Use(publicRateLimit)
		r.Post("/api/v1/public/berichte/reports/{token}", br.HandleGetSharedReport)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createDefinitionRequest struct {
	Name          string `json:"name"           validate:"required"`
	Description   string `json:"description,omitempty"`
	Module        string `json:"module"         validate:"required"`
	Kind          string `json:"kind,omitempty"`
	QueryConfig   []byte `json:"query_config,omitempty"`
	DefaultFormat string `json:"default_format,omitempty"`
	IsPublished   bool   `json:"is_published"`
}

type updateDefinitionRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Module        *string `json:"module,omitempty"`
	QueryConfig   []byte  `json:"query_config,omitempty"`
	DefaultFormat *string `json:"default_format,omitempty"`
	IsPublished   *bool   `json:"is_published,omitempty"`
}

type runReportRequest struct {
	Params       []byte `json:"params,omitempty"`
	ForceRefresh bool   `json:"force_refresh,omitempty"`
}

type exportReportRequest struct {
	Params []byte `json:"params,omitempty"`
}

type createScheduleRequest struct {
	DefinitionID   string   `json:"definition_id"             validate:"omitempty,uuid"`
	Name           string   `json:"name"                      validate:"required"`
	CronExpression string   `json:"cron_expression"           validate:"required"`
	Recipients     []string `json:"recipients,omitempty"      validate:"omitempty,dive,email"`
	Format         string   `json:"format,omitempty"          validate:"omitempty,oneof=pdf csv xlsx"`
	Params         []byte   `json:"params,omitempty"`
	Active         bool     `json:"active"`
}

type updateScheduleRequest struct {
	Name           *string  `json:"name,omitempty"`
	CronExpression *string  `json:"cron_expression,omitempty"`
	Recipients     []string `json:"recipients,omitempty"`
	RecipientsSet  bool     `json:"recipients_set,omitempty"`
	Format         *string  `json:"format,omitempty"`
	Params         []byte   `json:"params,omitempty"`
	Active         *bool    `json:"active,omitempty"`
}

type toggleScheduleRequest struct {
	Active bool `json:"active"`
}

// ============================================================================
// Definition Handlers
// ============================================================================

func (br *BerichteRoutes) HandleListDefinitions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &berichtev1.ListDefinitionsRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   q.Get("sort_by"),
	}

	if sd := q.Get("sort_desc"); sd == "true" || sd == "1" {
		grpcReq.SortDesc = true
	}
	if module := q.Get("module"); module != "" {
		grpcReq.Module = &module
	}
	if kind := q.Get("kind"); kind != "" {
		grpcReq.Kind = &kind
	}
	if pub := q.Get("is_published"); pub != "" {
		v := pub == "true" || pub == "1"
		grpcReq.IsPublished = &v
	}

	resp, err := client.ListDefinitions(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleCreateDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createDefinitionRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.CreateDefinitionRequest{
		TenantId:      tenantID.String(),
		Name:          req.Name,
		Description:   req.Description,
		Module:        req.Module,
		Kind:          req.Kind,
		QueryConfig:   req.QueryConfig,
		DefaultFormat: req.DefaultFormat,
		IsPublished:   req.IsPublished,
		CreatedBy:     &userID,
	}

	resp, err := client.CreateDefinition(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (br *BerichteRoutes) HandleGetDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetDefinition(r.Context(), &berichtev1.GetDefinitionRequest{
		TenantId:     tenantID.String(),
		DefinitionId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleUpdateDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateDefinitionRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.UpdateDefinitionRequest{
		TenantId:      tenantID.String(),
		DefinitionId:  id,
		Name:          req.Name,
		Description:   req.Description,
		Module:        req.Module,
		QueryConfig:   req.QueryConfig,
		DefaultFormat: req.DefaultFormat,
		IsPublished:   req.IsPublished,
		EditorId:      userID,
	}

	resp, err := client.UpdateDefinition(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleDeleteDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteDefinition(r.Context(), &berichtev1.DeleteDefinitionRequest{
		TenantId:     tenantID.String(),
		DefinitionId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Document Handlers (multi-page authoring)
// ============================================================================

// reportDocumentWire is the JSON shape the frontend types as ReportDocument
// (berichte-types.ts). Documents are not served through response.Proto because
// `rows`/`settings` are raw JSONB: protojson would emit the proto `bytes`
// fields as base64 strings instead of the nested block tree the editor reads.
type reportDocumentWire struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Module      string          `json:"module"`
	Status      string          `json:"status"`
	Rows        json.RawMessage `json:"rows"`
	Settings    json.RawMessage `json:"settings"`
	TemplateID  *string         `json:"template_id"`
	CreatedBy   *string         `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ReleasedAt  *time.Time      `json:"released_at"`
}

func toReportDocumentWire(d *berichtev1.Document) reportDocumentWire {
	wire := reportDocumentWire{
		ID:          d.GetId(),
		TenantID:    d.GetTenantId(),
		Title:       d.GetTitle(),
		Description: d.GetDescription(),
		Module:      d.GetModule(),
		Status:      d.GetStatus(),
		Rows:        json.RawMessage(d.GetRows()),
		Settings:    json.RawMessage(d.GetSettings()),
		TemplateID:  d.TemplateId,
		CreatedBy:   d.CreatedBy,
		CreatedAt:   d.GetCreatedAt().AsTime(),
		UpdatedAt:   d.GetUpdatedAt().AsTime(),
	}
	// Never emit a JSON `null` for the block tree: the editor iterates it.
	if len(wire.Rows) == 0 {
		wire.Rows = json.RawMessage("[]")
	}
	if len(wire.Settings) == 0 {
		wire.Settings = json.RawMessage("{}")
	}
	if d.ReleasedAt != nil {
		released := d.GetReleasedAt().AsTime()
		wire.ReleasedAt = &released
	}
	return wire
}

type createReportDocumentRequest struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Module      string          `json:"module,omitempty"      validate:"omitempty,oneof=finanzen crm helpdesk inventar produktion cross"`
	TemplateID  *string         `json:"template_id,omitempty"`
	Rows        json.RawMessage `json:"rows,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

type updateReportDocumentRequest struct {
	Title       *string         `json:"title,omitempty"`
	Description *string         `json:"description,omitempty"`
	Module      *string         `json:"module,omitempty"   validate:"omitempty,oneof=finanzen crm helpdesk inventar produktion cross"`
	Status      *string         `json:"status,omitempty"   validate:"omitempty,oneof=draft final released archived"`
	Rows        json.RawMessage `json:"rows,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

func (br *BerichteRoutes) HandleListDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	// The library view has no pager; default to a page large enough for it.
	page, pageSize := parsePagination(r, 1, 100)
	q := r.URL.Query()

	grpcReq := &berichtev1.ListDocumentsRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if module := q.Get("module"); module != "" {
		grpcReq.Module = &module
	}
	if docStatus := q.Get("status"); docStatus != "" {
		grpcReq.Status = &docStatus
	}

	resp, err := client.ListDocuments(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	documents := make([]reportDocumentWire, 0, len(resp.GetDocuments()))
	for _, d := range resp.GetDocuments() {
		documents = append(documents, toReportDocumentWire(d))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"documents": documents,
		"total":     resp.GetTotal(),
	})
}

func (br *BerichteRoutes) HandleGetDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetDocument(r.Context(), &berichtev1.GetDocumentRequest{
		TenantId:   tenantID.String(),
		DocumentId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"document": toReportDocumentWire(resp.GetDocument())})
}

// HandleExportDocumentPDF GET /api/v1/berichte/documents/{id}/export/pdf
// Returns a server-rendered PDF of the document's block tree (maroto, see
// berichte/export/document_pdf.go). The FE's window.print() path stays the
// primary "print" affordance; this is the real downloadable file behind it.
func (br *BerichteRoutes) HandleExportDocumentPDF(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ExportDocumentPDF(r.Context(), &berichtev1.ExportDocumentPDFRequest{
		TenantId:   tenantID.String(),
		DocumentId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	respondPDF(w, resp.GetPdfData(), resp.GetFilename())
}

func (br *BerichteRoutes) HandleCreateDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createReportDocumentRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateDocument(r.Context(), &berichtev1.CreateDocumentRequest{
		TenantId:    tenantID.String(),
		Title:       req.Title,
		Description: req.Description,
		Module:      req.Module,
		TemplateId:  req.TemplateID,
		Rows:        req.Rows,
		Settings:    req.Settings,
		CreatedBy:   &userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"document": toReportDocumentWire(resp.GetDocument())})
}

func (br *BerichteRoutes) HandleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateReportDocumentRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateDocument(r.Context(), &berichtev1.UpdateDocumentRequest{
		TenantId:    tenantID.String(),
		DocumentId:  id,
		Title:       req.Title,
		Description: req.Description,
		Module:      req.Module,
		Status:      req.Status,
		Rows:        req.Rows,
		Settings:    req.Settings,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"document": toReportDocumentWire(resp.GetDocument())})
}

func (br *BerichteRoutes) HandleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if _, err := client.DeleteDocument(r.Context(), &berichtev1.DeleteDocumentRequest{
		TenantId:   tenantID.String(),
		DocumentId: id,
	}); err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Template Handlers (static starter structures)
// ============================================================================

// reportTemplateWire is the JSON shape the frontend types as ReportTemplate
// (berichte-types.ts). Same rationale as reportDocumentWire: rows/settings
// are raw JSONB, not proto bytes, so this is served via response.JSON rather
// than response.Proto.
type reportTemplateWire struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Module      string          `json:"module"`
	Icon        *string         `json:"icon,omitempty"`
	Rows        json.RawMessage `json:"rows"`
	Settings    json.RawMessage `json:"settings"`
}

func toReportTemplateWire(t *berichtev1.Template) reportTemplateWire {
	wire := reportTemplateWire{
		ID:          t.GetId(),
		Title:       t.GetTitle(),
		Description: t.GetDescription(),
		Module:      t.GetModule(),
		Icon:        t.Icon,
		Rows:        json.RawMessage(t.GetRows()),
		Settings:    json.RawMessage(t.GetSettings()),
	}
	if len(wire.Rows) == 0 {
		wire.Rows = json.RawMessage("[]")
	}
	if len(wire.Settings) == 0 {
		wire.Settings = json.RawMessage("{}")
	}
	return wire
}

func (br *BerichteRoutes) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	resp, err := client.ListTemplates(r.Context(), &berichtev1.ListTemplatesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	templates := make([]reportTemplateWire, 0, len(resp.GetTemplates()))
	for _, t := range resp.GetTemplates() {
		templates = append(templates, toReportTemplateWire(t))
	}
	response.JSON(w, http.StatusOK, map[string]any{"templates": templates})
}

// ============================================================================
// Run & Export Handlers
// ============================================================================

func (br *BerichteRoutes) HandleRunReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[runReportRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.RunReportRequest{
		TenantId:     tenantID.String(),
		DefinitionId: id,
		Params:       req.Params,
		ForceRefresh: req.ForceRefresh,
		Trigger:      "manual",
	}

	resp, err := client.RunReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleExportReport runs and streams a report as a binary download (PDF/CSV/XLSX).
// The format is taken from the query parameter ?format=pdf|csv|xlsx; defaults to "pdf".
// Content-Disposition uses %q formatting to prevent filename injection.
func (br *BerichteRoutes) HandleExportReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}

	req, ok := decodeAndValidate[exportReportRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.ExportReportRequest{
		TenantId:     tenantID.String(),
		DefinitionId: id,
		Params:       req.Params,
		Format:       format,
	}

	resp, err := client.ExportReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	contentType := resp.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := resp.GetFilename()
	if filename == "" {
		filename = "report." + format
	}
	// Use %q to quote the filename safely — prevents Content-Disposition injection.
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}

// formatFilename returns a safely-quoted filename value for Content-Disposition headers.
// It strips any path separators and newlines before quoting to prevent header injection.
func formatFilename(name string) string {
	// Strip path separators and control characters
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\r' || r == '\n' || r == '"' {
			return -1
		}
		return r
	}, name)
	if name == "" {
		name = "report"
	}
	return `"` + name + `"`
}

func (br *BerichteRoutes) HandleInvalidateCache(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.InvalidateCache(r.Context(), &berichtev1.InvalidateCacheRequest{
		TenantId:     tenantID.String(),
		DefinitionId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Schedule Handlers
// ============================================================================

func (br *BerichteRoutes) HandleListSchedules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &berichtev1.ListSchedulesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if defID := q.Get("definition_id"); defID != "" {
		grpcReq.DefinitionId = &defID
	}
	if active := q.Get("active"); active != "" {
		v := active == "true" || active == "1"
		grpcReq.Active = &v
	}

	resp, err := client.ListSchedules(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createScheduleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.CreateScheduleRequest{
		TenantId:       tenantID.String(),
		DefinitionId:   req.DefinitionID,
		Name:           req.Name,
		CronExpression: req.CronExpression,
		Recipients:     req.Recipients,
		Format:         req.Format,
		Params:         req.Params,
		Active:         req.Active,
		CreatedBy:      &userID,
	}

	resp, err := client.CreateSchedule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (br *BerichteRoutes) HandleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateScheduleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.UpdateScheduleRequest{
		TenantId:       tenantID.String(),
		ScheduleId:     id,
		Name:           req.Name,
		CronExpression: req.CronExpression,
		Recipients:     req.Recipients,
		RecipientsSet:  req.RecipientsSet,
		Format:         req.Format,
		Params:         req.Params,
		Active:         req.Active,
	}

	resp, err := client.UpdateSchedule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteSchedule(r.Context(), &berichtev1.DeleteScheduleRequest{
		TenantId:   tenantID.String(),
		ScheduleId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (br *BerichteRoutes) HandleToggleSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[toggleScheduleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ToggleSchedule(r.Context(), &berichtev1.ToggleScheduleRequest{
		TenantId:   tenantID.String(),
		ScheduleId: id,
		Active:     req.Active,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// KPI Handler
// ============================================================================

func (br *BerichteRoutes) HandleGetDashboardKPIs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	// ?modules=finanzen,crm,helpdesk — empty means all modules
	var modules []string
	if m := r.URL.Query().Get("modules"); m != "" {
		for _, mod := range strings.Split(m, ",") {
			mod = strings.TrimSpace(mod)
			if mod != "" {
				modules = append(modules, mod)
			}
		}
	}

	resp, err := client.GetDashboardKPIs(r.Context(), &berichtev1.DashboardKPIsRequest{
		TenantId: tenantID.String(),
		Modules:  modules,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Share links
// ============================================================================

// maxSharePasswordBody caps the public read's request body. The only field it
// may carry is a password, so anything past a kilobyte is not a visitor.
const maxSharePasswordBody = 1 << 10

type createReportShareHTTPRequest struct {
	ExpiresInDays *int32 `json:"expires_in_days"`
	Password      string `json:"password"`
}

type publicReadShareHTTPRequest struct {
	Password string `json:"password"`
}

// reportShareWire mirrors ReportShareToken in berichte-types.ts.
type reportShareWire struct {
	ID          string     `json:"id"`
	DocumentID  string     `json:"document_id"`
	Token       string     `json:"token"`
	ExpiresAt   *time.Time `json:"expires_at"`
	HasPassword bool       `json:"has_password"`
	ViewCount   int32      `json:"view_count"`
	CreatedAt   time.Time  `json:"created_at"`
}

// publicReportDocumentWire is the reduced document an anonymous visitor sees.
//
// It is a separate type from reportDocumentWire on purpose: tenant_id,
// created_by and the editing status are internal facts that have no business
// on an unauthenticated response, and a shared field set would leak them the
// first time someone adds a field to the authenticated one.
type publicReportDocumentWire struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Module      string          `json:"module"`
	Rows        json.RawMessage `json:"rows"`
	Settings    json.RawMessage `json:"settings"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ReleasedAt  *time.Time      `json:"released_at"`
}

func toReportShareWire(s *berichtev1.ShareToken) reportShareWire {
	wire := reportShareWire{
		ID:          s.GetId(),
		DocumentID:  s.GetDocumentId(),
		Token:       s.GetToken(),
		HasPassword: s.GetHasPassword(),
		ViewCount:   s.GetViewCount(),
		CreatedAt:   s.GetCreatedAt().AsTime(),
	}
	if s.ExpiresAt != nil {
		expires := s.GetExpiresAt().AsTime()
		wire.ExpiresAt = &expires
	}
	return wire
}

func toPublicReportDocumentWire(d *berichtev1.Document) publicReportDocumentWire {
	wire := publicReportDocumentWire{
		ID:          d.GetId(),
		Title:       d.GetTitle(),
		Description: d.GetDescription(),
		Module:      d.GetModule(),
		Rows:        json.RawMessage(d.GetRows()),
		Settings:    json.RawMessage(d.GetSettings()),
		UpdatedAt:   d.GetUpdatedAt().AsTime(),
	}
	if len(wire.Rows) == 0 {
		wire.Rows = json.RawMessage("[]")
	}
	if len(wire.Settings) == 0 {
		wire.Settings = json.RawMessage("{}")
	}
	if d.ReleasedAt != nil {
		released := d.GetReleasedAt().AsTime()
		wire.ReleasedAt = &released
	}
	return wire
}

func (br *BerichteRoutes) HandleListReportShares(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListShareTokens(r.Context(), &berichtev1.ListShareTokensRequest{
		TenantId:   tenantID.String(),
		DocumentId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	shares := make([]reportShareWire, 0, len(resp.GetTokens()))
	for _, t := range resp.GetTokens() {
		shares = append(shares, toReportShareWire(t))
	}
	response.JSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (br *BerichteRoutes) HandleCreateReportShare(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createReportShareHTTPRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &berichtev1.CreateShareTokenRequest{
		TenantId:      tenantID.String(),
		DocumentId:    id,
		ExpiresInDays: req.ExpiresInDays,
	}
	if req.Password != "" {
		grpcReq.Password = &req.Password
	}
	if userID := middleware.GetUserID(r.Context()); userID != "" {
		grpcReq.CreatedBy = &userID
	}

	resp, err := client.CreateShareToken(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"share": toReportShareWire(resp.GetShare())})
}

func (br *BerichteRoutes) HandleRevokeReportShare(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	shareID, ok := validateUUIDParam(w, r, "shareId")
	if !ok {
		return
	}

	if _, err := client.RevokeShareToken(r.Context(), &berichtev1.RevokeShareTokenRequest{
		TenantId: tenantID.String(),
		ShareId:  shareID,
	}); err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetSharedReport serves the unauthenticated public read of a shared
// report.
//
// It is a POST, not a GET, for two reasons that both matter more than the verb
// reading as a read: the password must not land in a URL, where it would end up
// in access logs, browser history and Referer headers, and the call increments
// the link's view counter, which no prefetching GET should be free to do.
func (br *BerichteRoutes) HandleGetSharedReport(w http.ResponseWriter, r *http.Request) {
	client, err := br.getClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" || len(token) > 128 {
		// Same answer a well-formed unknown token gets: this route never
		// distinguishes "malformed" from "no such link".
		response.Error(w, http.StatusNotFound, "share link not found")
		return
	}

	// The body is optional — a link without a password is read with no body at
	// all — so an empty or absent one is not an error, only a malformed one is.
	var body publicReadShareHTTPRequest
	raw, readErr := io.ReadAll(io.LimitReader(r.Body, maxSharePasswordBody))
	if readErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	grpcReq := &berichtev1.GetSharedDocumentRequest{Token: token}
	if body.Password != "" {
		grpcReq.Password = &body.Password
	}

	resp, err := client.GetSharedDocument(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"document": toPublicReportDocumentWire(resp.GetDocument())})
}
