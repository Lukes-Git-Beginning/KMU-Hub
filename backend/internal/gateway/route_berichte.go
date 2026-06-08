package gateway

import (
	"net/http"
	"strings"

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

		// KPIs
		r.With(middleware.RequirePermission("berichte:reports", "read")).Get("/kpis", br.HandleGetDashboardKPIs)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleCreateDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusCreated, resp)
}

func (br *BerichteRoutes) HandleGetDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleUpdateDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleDeleteDefinition(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
// Run & Export Handlers
// ============================================================================

func (br *BerichteRoutes) HandleRunReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

// HandleExportReport runs and streams a report as a binary download (PDF/CSV/XLSX).
// The format is taken from the query parameter ?format=pdf|csv|xlsx; defaults to "pdf".
// Content-Disposition uses %q formatting to prevent filename injection.
func (br *BerichteRoutes) HandleExportReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Schedule Handlers
// ============================================================================

func (br *BerichteRoutes) HandleListSchedules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusCreated, resp)
}

func (br *BerichteRoutes) HandleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

func (br *BerichteRoutes) HandleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// KPI Handler
// ============================================================================

func (br *BerichteRoutes) HandleGetDashboardKPIs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	response.JSON(w, http.StatusOK, resp)
}
