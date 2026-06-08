package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SchichtenRoutes handles HTTP routes for the Schichten (shift planning) module.
type SchichtenRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewSchichtenRoutes creates a new SchichtenRoutes.
func NewSchichtenRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *SchichtenRoutes {
	return &SchichtenRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (sr *SchichtenRoutes) ServiceName() string { return "schichten" }

// getClient lazily obtains a gRPC client for the schichten service.
func (sr *SchichtenRoutes) getClient() (schichtenv1.SchichtenServiceClient, error) {
	conn, err := sr.registry.GetConnection("schichten")
	if err != nil {
		return nil, err
	}
	return schichtenv1.NewSchichtenServiceClient(conn), nil
}

// RegisterRoutes mounts all Schichten HTTP routes behind the feature flag modules.schichten.
func (sr *SchichtenRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !sr.flags.IsEnabled("modules.schichten") {
		return
	}

	r.Route("/api/v1/schichten", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Shifts
		r.Route("/shifts", func(r chi.Router) {
			r.With(middleware.RequirePermission("schichten:shift", "read")).Get("/", sr.HandleListShifts)
			r.With(middleware.RequirePermission("schichten:shift", "write")).Post("/", sr.HandleCreateShift)
			r.With(middleware.RequirePermission("schichten:shift", "write")).Post("/publish", sr.HandlePublishShifts)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("schichten:shift", "read")).Get("/", sr.HandleGetShift)
				r.With(middleware.RequirePermission("schichten:shift", "write")).Patch("/", sr.HandleUpdateShift)
				r.With(middleware.RequirePermission("schichten:shift", "write")).Delete("/", sr.HandleDeleteShift)

				// Assignments
				r.With(middleware.RequirePermission("schichten:assignment", "read")).Get("/assignments", sr.HandleListAssignments)
				r.With(middleware.RequirePermission("schichten:assignment", "write")).Post("/assignments", sr.HandleAssignEmployee)
				r.With(middleware.RequirePermission("schichten:assignment", "write")).Delete("/assignments/{employee_id}", sr.HandleUnassignEmployee)
			})
		})

		// Templates
		r.Route("/templates", func(r chi.Router) {
			r.With(middleware.RequirePermission("schichten:template", "read")).Get("/", sr.HandleListTemplates)
			r.With(middleware.RequirePermission("schichten:template", "write")).Post("/", sr.HandleCreateTemplate)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("schichten:template", "write")).Patch("/", sr.HandleUpdateTemplate)
				r.With(middleware.RequirePermission("schichten:template", "write")).Delete("/", sr.HandleDeleteTemplate)
				r.With(middleware.RequirePermission("schichten:shift", "write")).Post("/apply", sr.HandleApplyTemplate)
			})
		})

		// Compliance & Stats
		r.With(middleware.RequirePermission("schichten:shift", "read")).Get("/compliance", sr.HandleCheckArbzgCompliance)
		r.With(middleware.RequirePermission("schichten:shift", "read")).Get("/stats", sr.HandleGetShiftStats)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createShiftRequest struct {
	Title       string  `json:"title"       validate:"required"`
	Description string  `json:"description"`
	StartTime   string  `json:"start_time"  validate:"required"` // RFC3339
	EndTime     string  `json:"end_time"    validate:"required"` // RFC3339
	Location    *string `json:"location,omitempty"`
	CreatedBy   *string `json:"created_by,omitempty" validate:"omitempty,uuid"`
}

type updateShiftRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	StartTime   *string `json:"start_time,omitempty"`
	EndTime     *string `json:"end_time,omitempty"`
	Location    *string `json:"location,omitempty"`
}

type publishShiftsRequest struct {
	From string `json:"from" validate:"required"` // RFC3339
	To   string `json:"to"   validate:"required"` // RFC3339
}

type assignEmployeeRequest struct {
	EmployeeID string  `json:"employee_id"           validate:"required,uuid"`
	AssignedBy *string `json:"assigned_by,omitempty" validate:"omitempty,uuid"`
}

type createTemplateRequest struct {
	Name            string  `json:"name"             validate:"required"`
	Description     string  `json:"description"`
	DayOfWeek       int32   `json:"day_of_week"      validate:"gte=0,lte=6"`
	StartHour       int32   `json:"start_hour"       validate:"gte=0,lte=23"`
	StartMinute     int32   `json:"start_minute"     validate:"gte=0,lte=59"`
	DurationMinutes int32   `json:"duration_minutes" validate:"gt=0"`
	Location        *string `json:"location,omitempty"`
}

type updateTemplateRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	DayOfWeek       *int32  `json:"day_of_week,omitempty"`
	StartHour       *int32  `json:"start_hour,omitempty"`
	StartMinute     *int32  `json:"start_minute,omitempty"`
	DurationMinutes *int32  `json:"duration_minutes,omitempty"`
	Location        *string `json:"location,omitempty"`
}

type applyTemplateRequest struct {
	RangeStart string  `json:"range_start"           validate:"required"` // RFC3339
	RangeEnd   string  `json:"range_end"             validate:"required"` // RFC3339
	CreatedBy  *string `json:"created_by,omitempty"  validate:"omitempty,uuid"`
}

// ============================================================================
// Shift Handlers
// ============================================================================

func (sr *SchichtenRoutes) HandleListShifts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &schichtenv1.ListShiftsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}

	resp, err := client.ListShifts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleCreateShift(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createShiftRequest](w, r)
	if !ok {
		return
	}

	start, err := parseRFC3339ToTimestamp(req.StartTime)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid start_time: use RFC3339")
		return
	}
	end, err := parseRFC3339ToTimestamp(req.EndTime)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid end_time: use RFC3339")
		return
	}

	grpcReq := &schichtenv1.CreateShiftRequest{
		TenantId:    tenantID.String(),
		Title:       req.Title,
		Description: req.Description,
		StartTime:   start,
		EndTime:     end,
		Location:    req.Location,
		CreatedBy:   req.CreatedBy,
	}

	resp, err := client.CreateShift(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (sr *SchichtenRoutes) HandleGetShift(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetShift(r.Context(), &schichtenv1.GetShiftRequest{
		TenantId: tenantID.String(),
		ShiftId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleUpdateShift(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateShiftRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &schichtenv1.UpdateShiftRequest{
		TenantId:    tenantID.String(),
		ShiftId:     id,
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
	}
	if req.StartTime != nil {
		ts, err := parseRFC3339ToTimestamp(*req.StartTime)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid start_time")
			return
		}
		grpcReq.StartTime = ts
	}
	if req.EndTime != nil {
		ts, err := parseRFC3339ToTimestamp(*req.EndTime)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid end_time")
			return
		}
		grpcReq.EndTime = ts
	}

	resp, err := client.UpdateShift(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleDeleteShift(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteShift(r.Context(), &schichtenv1.DeleteShiftRequest{
		TenantId: tenantID.String(),
		ShiftId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sr *SchichtenRoutes) HandlePublishShifts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[publishShiftsRequest](w, r)
	if !ok {
		return
	}

	from, err := parseRFC3339ToTimestamp(req.From)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid from: use RFC3339")
		return
	}
	to, err := parseRFC3339ToTimestamp(req.To)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid to: use RFC3339")
		return
	}

	resp, err := client.PublishShifts(r.Context(), &schichtenv1.PublishShiftsRequest{
		TenantId: tenantID.String(),
		From:     from,
		To:       to,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Assignment Handlers
// ============================================================================

func (sr *SchichtenRoutes) HandleListAssignments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListAssignments(r.Context(), &schichtenv1.ListAssignmentsRequest{
		TenantId: tenantID.String(),
		ShiftId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleAssignEmployee(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	shiftID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[assignEmployeeRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &schichtenv1.AssignEmployeeRequest{
		TenantId:   tenantID.String(),
		ShiftId:    shiftID,
		EmployeeId: req.EmployeeID,
		AssignedBy: req.AssignedBy,
	}

	resp, err := client.AssignEmployee(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (sr *SchichtenRoutes) HandleUnassignEmployee(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	shiftID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	employeeID, ok := validateUUIDParam(w, r, "employee_id")
	if !ok {
		return
	}

	_, err = client.UnassignEmployee(r.Context(), &schichtenv1.UnassignEmployeeRequest{
		TenantId:   tenantID.String(),
		ShiftId:    shiftID,
		EmployeeId: employeeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Template Handlers
// ============================================================================

func (sr *SchichtenRoutes) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListTemplates(r.Context(), &schichtenv1.ListTemplatesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createTemplateRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTemplate(r.Context(), &schichtenv1.CreateTemplateRequest{
		TenantId:        tenantID.String(),
		Name:            req.Name,
		Description:     req.Description,
		DayOfWeek:       req.DayOfWeek,
		StartHour:       req.StartHour,
		StartMinute:     req.StartMinute,
		DurationMinutes: req.DurationMinutes,
		Location:        req.Location,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (sr *SchichtenRoutes) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateTemplateRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &schichtenv1.UpdateTemplateRequest{
		TenantId:        tenantID.String(),
		TemplateId:      id,
		Name:            req.Name,
		Description:     req.Description,
		DayOfWeek:       req.DayOfWeek,
		StartHour:       req.StartHour,
		StartMinute:     req.StartMinute,
		DurationMinutes: req.DurationMinutes,
		Location:        req.Location,
	}

	resp, err := client.UpdateTemplate(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteTemplate(r.Context(), &schichtenv1.DeleteTemplateRequest{
		TenantId:   tenantID.String(),
		TemplateId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sr *SchichtenRoutes) HandleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[applyTemplateRequest](w, r)
	if !ok {
		return
	}

	rangeStart, err := parseRFC3339ToTimestamp(req.RangeStart)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid range_start: use RFC3339")
		return
	}
	rangeEnd, err := parseRFC3339ToTimestamp(req.RangeEnd)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid range_end: use RFC3339")
		return
	}

	resp, err := client.ApplyTemplate(r.Context(), &schichtenv1.ApplyTemplateRequest{
		TenantId:   tenantID.String(),
		TemplateId: id,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		CreatedBy:  req.CreatedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

// ============================================================================
// Compliance & Stats Handlers
// ============================================================================

func (sr *SchichtenRoutes) HandleCheckArbzgCompliance(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	q := r.URL.Query()
	employeeID := q.Get("employee_id")
	if employeeID == "" {
		response.Error(w, http.StatusBadRequest, "employee_id is required")
		return
	}

	newShiftStart, err := parseRFC3339ToTimestamp(q.Get("new_shift_start"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid new_shift_start: use RFC3339")
		return
	}
	newShiftEnd, err := parseRFC3339ToTimestamp(q.Get("new_shift_end"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid new_shift_end: use RFC3339")
		return
	}

	resp, err := client.CheckArbzgCompliance(r.Context(), &schichtenv1.CheckArbzgComplianceRequest{
		TenantId:      tenantID.String(),
		EmployeeId:    employeeID,
		NewShiftStart: newShiftStart,
		NewShiftEnd:   newShiftEnd,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (sr *SchichtenRoutes) HandleGetShiftStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := sr.getClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	resp, err := client.GetShiftStats(r.Context(), &schichtenv1.GetShiftStatsRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Internal helpers
// ============================================================================

func parseRFC3339ToTimestamp(s string) (*timestamppb.Timestamp, error) {
	if s == "" {
		return nil, fmt.Errorf("empty timestamp string")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(t), nil
}
