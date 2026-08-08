package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// hrMarshalSlice marshals a slice of proto messages via protojson for embedding in an
// ad-hoc map envelope (e.g. {"leave_requests": [...], "total": N}), mirroring
// response.ProtoList's per-element encoding without writing directly to the response
// writer. Reuses cannedResponseMarshaler (defined in route_inbox.go, same gateway
// package) since its protojson options already match response.Proto/response.ProtoList.
func hrMarshalSlice[T proto.Message](msgs []T) ([]json.RawMessage, error) {
	parts := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		b, err := cannedResponseMarshaler.Marshal(m)
		if err != nil {
			return nil, err
		}
		parts = append(parts, b)
	}
	return parts, nil
}

// hrMarshalProto marshals a single proto message the same way, for envelopes
// that carry one entity next to other fields (e.g. {"statement": {...},
// "transactions": [...]}).
func hrMarshalProto[T proto.Message](msg T) (json.RawMessage, error) {
	return cannedResponseMarshaler.Marshal(msg)
}

// HRRoutes handles HTTP routes for the HR backend service.
// HR services run on the same gRPC server as Finance (the "biz" binary),
// so ServiceName returns "biz" to reuse the existing connection.
type HRRoutes struct {
	registry *ServiceRegistry
	bizExt   *BizExtRoutes // optional: direct-DB extension handlers
}

// NewHRRoutes creates a new HRRoutes with the given service registry.
// bizExt may be nil if extension features are not configured.
func NewHRRoutes(registry *ServiceRegistry, bizExt *BizExtRoutes) *HRRoutes {
	return &HRRoutes{registry: registry, bizExt: bizExt}
}

// ServiceName returns "biz" because HR services share the biz gRPC server.
func (h *HRRoutes) ServiceName() string { return "biz" }

// getHRClient lazily obtains a gRPC client for the HR service.
func (h *HRRoutes) getHRClient() (hrv1.HRServiceClient, error) {
	conn, err := h.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return hrv1.NewHRServiceClient(conn), nil
}

// RegisterRoutes registers all HR HTTP routes.
func (h *HRRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Leave management
	r.Route("/api/v1/hr/leave", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "write")).Post("/requests", h.HandleCreateLeaveRequest)
		r.With(middleware.RequirePermission("hr", "read")).Get("/requests", h.HandleListLeaveRequests)
		r.With(middleware.RequirePermission("hr", "read")).Get("/requests/{id}", h.HandleGetLeaveRequest)
		r.With(middleware.RequirePermission("hr", "write")).Post("/requests/{id}/approve", h.HandleApproveLeaveRequest)
		r.With(middleware.RequirePermission("hr", "write")).Post("/requests/{id}/reject", h.HandleRejectLeaveRequest)
		r.With(middleware.RequirePermission("hr", "write")).Post("/requests/{id}/cancel", h.HandleCancelLeaveRequest)
		r.With(middleware.RequirePermission("hr", "read")).Get("/balance", h.HandleGetLeaveBalance)
		r.With(middleware.RequirePermission("hr", "read")).Get("/balance/{userId}", h.HandleGetEmployeeLeaveBalance)
		r.With(middleware.RequirePermission("hr", "read")).Get("/types", h.HandleListLeaveTypes)
		r.With(middleware.RequirePermission("hr", "write")).Post("/sick", h.HandleRecordSickLeave)
	})

	// Time tracking (Zeiterfassung). Additive guards on the supervisory routes:
	// the legacy `hr:read`/`hr:write` key stays valid AND the matching
	// capability-catalogue key grants access. Permissions are baked into the
	// access token at login and never re-read per request, so swapping a key
	// outright would 403 every user holding a still-valid token.
	//
	// Personal daily use (clock in/out, own entries, own week) stays on the
	// coarse key by design — the catalogue deliberately carries no capability
	// for it. `zeiterfassung:export:run` has no endpoint yet.
	var (
		zeitTeamView    = middleware.RequirePermissionAny([2]string{"hr", "read"}, [2]string{"zeiterfassung:team", "view"})
		zeitWeekApprove = middleware.RequirePermissionAny([2]string{"hr", "write"}, [2]string{"zeiterfassung:week", "approve"})
		zeitCorrApprove = middleware.RequirePermissionAny([2]string{"hr", "write"}, [2]string{"zeiterfassung:corrections", "approve"})
	)

	r.Route("/api/v1/hr/time", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "write")).Post("/clock-in", h.HandleClockIn)
		r.With(middleware.RequirePermission("hr", "write")).Post("/clock-out", h.HandleClockOut)
		r.With(middleware.RequirePermission("hr", "write")).Post("/break/start", h.HandleBreakStart)
		r.With(middleware.RequirePermission("hr", "write")).Post("/break/end", h.HandleBreakEnd)
		r.With(middleware.RequirePermission("hr", "read")).Get("/active", h.HandleGetActiveShift)
		r.With(middleware.RequirePermission("hr", "read")).Get("/status", h.HandleGetWorkTimeStatus)
		r.With(middleware.RequirePermission("hr", "read")).Get("/entries", h.HandleListWorkTimeEntries)
		r.With(middleware.RequirePermission("hr", "write")).Post("/entries", h.HandleCreateManualEntry)
		r.With(middleware.RequirePermission("hr", "read")).Get("/summary/daily", h.HandleDailySummary)
		r.With(middleware.RequirePermission("hr", "read")).Get("/summary/weekly", h.HandleWeeklySummary)
		r.With(middleware.RequirePermission("hr", "write")).Post("/corrections", h.HandleSubmitCorrection)
		r.With(zeitCorrApprove).Post("/corrections/{id}/approve", h.HandleApproveCorrection)
		r.With(middleware.RequirePermission("hr", "read")).Get("/balance", h.HandleGetTimeBalance)
		r.With(middleware.RequirePermission("hr", "read")).Get("/analytics", h.HandleGetTimeAnalytics)
		r.With(zeitTeamView).Get("/team", h.HandleGetTeamTime)
		// Week approvals
		r.With(middleware.RequirePermission("hr", "read")).Get("/weeks/status", h.HandleGetMyWeekStatus)
		r.With(middleware.RequirePermission("hr", "write")).Post("/weeks/submit", h.HandleSubmitWeek)
		r.With(zeitWeekApprove).Post("/weeks/approve", h.HandleApproveWeek)
		r.With(zeitWeekApprove).Post("/weeks/reject", h.HandleRejectWeek)
		r.With(zeitWeekApprove).Post("/weeks/reopen", h.HandleReopenWeek)
		// Time categories
		r.With(middleware.RequirePermission("hr:time_category", "read")).Get("/categories", h.HandleListTimeCategories)
		r.With(middleware.RequirePermission("hr:time_category", "write")).Post("/categories", h.HandleCreateTimeCategory)
		r.With(middleware.RequirePermission("hr:time_category", "write")).Put("/categories/{id}", h.HandleUpdateTimeCategory)
		r.With(middleware.RequirePermission("hr:time_category", "write")).Delete("/categories/{id}", h.HandleDeleteTimeCategory)
		// Time templates
		r.With(middleware.RequirePermission("hr:time_template", "read")).Get("/templates", h.HandleListTimeTemplates)
		r.With(middleware.RequirePermission("hr:time_template", "write")).Post("/templates", h.HandleCreateTimeTemplate)
		r.With(middleware.RequirePermission("hr:time_template", "write")).Delete("/templates/{id}", h.HandleDeleteTimeTemplate)
		// Time projects
		r.With(middleware.RequirePermission("hr:time_project", "read")).Get("/projects", h.HandleListTimeProjects)
		r.With(middleware.RequirePermission("hr:time_project", "write")).Post("/projects", h.HandleCreateTimeProject)

		// Extension: time-tracking → invoice conversion
		if h.bizExt != nil {
			h.bizExt.registerTimeExtRoutes(r)
		}
	})

	// Absence calendar
	r.Route("/api/v1/hr/absences", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "read")).Get("/calendar", h.HandleGetAbsenceCalendar)
	})

	// Same additive-guard rationale as zeitTeamView etc. above: "hr:read" is
	// admin-only today (migration 000129), so without the second key manager
	// and member (both hold team:documents:view, see migration 000256) could
	// never reach this route at all.
	hrDocumentCategoriesGuard := middleware.RequirePermissionAny(
		[2]string{"hr", "read"}, [2]string{"team:documents", "view"},
	)

	// Employee profiles
	r.Route("/api/v1/hr/employees", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "read")).Get("/", h.HandleListEmployees)
		r.With(middleware.RequirePermission("hr", "admin")).Post("/", h.HandleCreateEmployee)
		r.With(middleware.RequirePermission("hr", "read")).Get("/me", h.HandleGetSelfProfile)
		r.With(middleware.RequirePermission("hr", "write")).Put("/me", h.HandleUpdateSelfProfile)
		r.With(middleware.RequirePermission("hr", "read")).Get("/{id}", h.HandleGetEmployee)
		r.With(middleware.RequirePermission("hr", "admin")).Put("/{id}", h.HandleUpdateEmployee)
		// team:employee:offboard is in the catalogue since 000256 and assigned
		// to admin and hr_admin, so no seed migration is needed here.
		r.With(middleware.RequirePermission("team:employee", "offboard")).Post("/{id}/offboard", h.HandleOffboardEmployee)
		r.With(middleware.RequirePermission("hr", "read")).Get("/{id}/documents", h.HandleListEmployeeDocuments)
		r.With(middleware.RequirePermission("hr", "write")).Post("/{id}/documents", h.HandleUploadEmployeeDocument)
		// Additive guard: the legacy "hr:read" key stays valid AND the
		// capability-catalogue key (manager scope='team', member scope='own')
		// grants access — see the zeiterfassung guards above for the same
		// pattern and its rationale (never replace a baked-in JWT key outright).
		r.With(hrDocumentCategoriesGuard).Get("/{id}/documents/categories", h.HandleListEmployeeDocumentCategories)
	})

	// Personalakte — the tenant-wide document view the Personalakte tab reads.
	// Guards are the team:documents keys from migration 000256 (no new key, so
	// no seed migration); which rows a holder actually sees is decided by RLS
	// policy hr_document_access, not by the guard.
	r.Route("/api/v1/hr/personnel-documents", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermissionAny(
			[2]string{"hr", "read"}, [2]string{"team:documents", "view"},
		)).Get("/", h.HandleListPersonnelDocuments)
		r.With(middleware.RequirePermissionAny(
			[2]string{"hr", "write"}, [2]string{"team:documents", "edit"},
		)).Post("/", h.HandleCreatePersonnelDocument)
	})

	// Profile change requests. The list is readable by anyone who may propose
	// OR decide; what each of them actually sees is narrowed in the handler,
	// not by the guard (a proposer sees only their own rows).
	changeRequestReadGuard := middleware.RequirePermissionAny(
		[2]string{"team:self", "propose"}, [2]string{"team:data_personal", "edit"},
	)
	r.Route("/api/v1/hr/change-requests", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(changeRequestReadGuard).Get("/", h.HandleListChangeRequests)
		r.With(middleware.RequirePermission("team:self", "propose")).Post("/", h.HandleCreateChangeRequest)
		r.With(middleware.RequirePermission("team:data_personal", "edit")).Post("/{id}/approve", h.HandleApproveChangeRequest)
		r.With(middleware.RequirePermission("team:data_personal", "edit")).Post("/{id}/reject", h.HandleRejectChangeRequest)
		r.With(middleware.RequirePermission("team:self", "propose")).Post("/{id}/cancel", h.HandleCancelChangeRequest)
	})

	// HR settings (admin only)
	r.Route("/api/v1/hr/settings", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin")).Get("/", h.HandleGetHRSettings)
		r.With(middleware.RequireRole("admin")).Put("/", h.HandleUpdateHRSettings)
	})
}

// ============================================================================
// Leave Management Handlers
// ============================================================================

type createLeaveRequestHTTPReq struct {
	LeaveTypeID        string `json:"leave_type_id"          validate:"omitempty,uuid"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	IsHalfDayStart     bool   `json:"is_half_day_start"`
	IsHalfDayEnd       bool   `json:"is_half_day_end"`
	HalfDayPeriodStart string `json:"half_day_period_start"  validate:"omitempty,oneof=morning afternoon"`
	HalfDayPeriodEnd   string `json:"half_day_period_end"    validate:"omitempty,oneof=morning afternoon"`
	Reason             string `json:"reason"`
}

func (h *HRRoutes) HandleCreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createLeaveRequestHTTPReq](w, r)
	if !ok {
		return
	}

	grpcReq := &hrv1.CreateLeaveRequestReq{
		TenantId:       tenantID,
		UserId:         userID,
		LeaveTypeId:    req.LeaveTypeID,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		IsHalfDayStart: req.IsHalfDayStart,
		IsHalfDayEnd:   req.IsHalfDayEnd,
		Reason:         req.Reason,
	}

	// oneof tag guarantees only valid values reach here; default stays UNSPECIFIED
	if req.HalfDayPeriodStart == "morning" {
		grpcReq.HalfDayPeriodStart = hrv1.HalfDayPeriod_HALF_DAY_MORNING
	} else if req.HalfDayPeriodStart == "afternoon" {
		grpcReq.HalfDayPeriodStart = hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON
	}
	if req.HalfDayPeriodEnd == "morning" {
		grpcReq.HalfDayPeriodEnd = hrv1.HalfDayPeriod_HALF_DAY_MORNING
	} else if req.HalfDayPeriodEnd == "afternoon" {
		grpcReq.HalfDayPeriodEnd = hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON
	}

	resp, err := client.CreateLeaveRequest(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.LeaveRequest)
}

func (h *HRRoutes) HandleListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	page, pageSize := parsePagination(r, 1, 50)

	grpcReq := &hrv1.ListLeaveRequestsReq{
		TenantId:   tenantID,
		EmployeeId: r.URL.Query().Get("employee_id"),
		Status:     r.URL.Query().Get("status"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		Page:       int32(page),
		PerPage:    int32(pageSize),
	}

	resp, err := client.ListLeaveRequests(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	leaveRequestsJSON, err := hrMarshalSlice(resp.LeaveRequests)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"leave_requests": leaveRequestsJSON,
		"total":          resp.Total,
	})
}

func (h *HRRoutes) HandleGetLeaveRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	resp, err := client.GetLeaveRequest(r.Context(), &hrv1.GetLeaveRequestReq{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.LeaveRequest)
}

type approveLeaveRequestHTTPReq struct {
	Comment string `json:"comment"`
}

func (h *HRRoutes) HandleApproveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	approverID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[approveLeaveRequestHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.ApproveLeaveRequest(r.Context(), &hrv1.ApproveLeaveRequestReq{
		Id:         id,
		ApproverId: approverID,
		Comment:    req.Comment,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.LeaveRequest)
}

type rejectLeaveRequestHTTPReq struct {
	Comment string `json:"comment"`
}

func (h *HRRoutes) HandleRejectLeaveRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	approverID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[rejectLeaveRequestHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.RejectLeaveRequest(r.Context(), &hrv1.RejectLeaveRequestReq{
		Id:         id,
		ApproverId: approverID,
		Comment:    req.Comment,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.LeaveRequest)
}

func (h *HRRoutes) HandleCancelLeaveRequest(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	resp, err := client.CancelLeaveRequest(r.Context(), &hrv1.CancelLeaveRequestReq{
		Id:     id,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.LeaveRequest)
}

func (h *HRRoutes) HandleGetLeaveBalance(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetLeaveBalance(r.Context(), &hrv1.GetLeaveBalanceReq{
		TenantId: tenantID,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Balance)
}

func (h *HRRoutes) HandleGetEmployeeLeaveBalance(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	employeeID := chi.URLParam(r, "userId")

	resp, err := client.GetEmployeeLeaveBalance(r.Context(), &hrv1.GetEmployeeLeaveBalanceReq{
		TenantId:   tenantID,
		EmployeeId: employeeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Balance)
}

func (h *HRRoutes) HandleListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListLeaveTypes(r.Context(), &hrv1.ListLeaveTypesReq{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoList(w, http.StatusOK, resp.LeaveTypes)
}

type recordSickLeaveHTTPReq struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Notes     string `json:"notes"`
}

func (h *HRRoutes) HandleRecordSickLeave(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[recordSickLeaveHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.RecordSickLeave(r.Context(), &hrv1.RecordSickLeaveReq{
		TenantId:  tenantID,
		UserId:    userID,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Notes:     req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	leaveRequestJSON, err := cannedResponseMarshaler.Marshal(resp.LeaveRequest)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"leave_request":        json.RawMessage(leaveRequestJSON),
		"au_document_required": resp.AuDocumentRequired,
	})
}

// ============================================================================
// Time Tracking Handlers
// ============================================================================

func (h *HRRoutes) HandleClockIn(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.ClockIn(r.Context(), &hrv1.ClockInReq{
		TenantId: tenantID,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Entry)
}

func (h *HRRoutes) HandleClockOut(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.ClockOut(r.Context(), &hrv1.ClockOutReq{
		TenantId: tenantID,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	entryJSON, err := cannedResponseMarshaler.Marshal(resp.Entry)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	result := map[string]interface{}{
		"entry": json.RawMessage(entryJSON),
	}
	if resp.Severity != "" {
		result["severity"] = resp.Severity
		result["arbzg_message"] = resp.ArbzgMessage
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *HRRoutes) HandleBreakStart(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.StartBreak(r.Context(), &hrv1.StartBreakReq{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.BreakEntry)
}

func (h *HRRoutes) HandleBreakEnd(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.EndBreak(r.Context(), &hrv1.EndBreakReq{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.BreakEntry)
}

func (h *HRRoutes) HandleGetActiveShift(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetActiveShift(r.Context(), &hrv1.GetActiveShiftReq{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Entry and ActiveBreak are optional (nil if no active shift / not on break).
	// protojson.Marshal(nil-typed-pointer) yields "{}", not "null" — marshal only
	// when present so the envelope keeps encoding/json's null semantics.
	entryJSON := json.RawMessage("null")
	if resp.Entry != nil {
		b, err := cannedResponseMarshaler.Marshal(resp.Entry)
		if err != nil {
			slog.Error("proto json marshal failed", "error", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		entryJSON = b
	}

	activeBreakJSON := json.RawMessage("null")
	if resp.ActiveBreak != nil {
		b, err := cannedResponseMarshaler.Marshal(resp.ActiveBreak)
		if err != nil {
			slog.Error("proto json marshal failed", "error", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		activeBreakJSON = b
	}

	result := map[string]interface{}{
		"entry":        entryJSON,
		"is_on_break":  resp.IsOnBreak,
		"active_break": activeBreakJSON,
	}

	response.JSON(w, http.StatusOK, result)
}

// HandleGetWorkTimeStatus returns a convenience status for the header quick-toggle button.
// Composes data from GetActiveShift and GetDailySummary RPCs since no dedicated status RPC exists.
func (h *HRRoutes) HandleGetWorkTimeStatus(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	status := map[string]interface{}{
		"is_clocked_in":       false,
		"is_on_break":         false,
		"current_shift_start": nil,
		"current_break_start": nil,
		"today_total_minutes": 0,
		"arbzg_severity":      "none",
	}

	// Check active shift
	shiftResp, shiftErr := client.GetActiveShift(r.Context(), &hrv1.GetActiveShiftReq{
		UserId: userID,
	})
	if shiftErr == nil && shiftResp.Entry != nil {
		status["is_clocked_in"] = true
		status["current_shift_start"] = shiftResp.Entry.ClockIn.AsTime().Format(time.RFC3339)
		if shiftResp.IsOnBreak && shiftResp.ActiveBreak != nil {
			status["is_on_break"] = true
			status["current_break_start"] = shiftResp.ActiveBreak.StartTime.AsTime().Format(time.RFC3339)
		}
	}

	// Get today's summary for total minutes
	today := time.Now().Format("2006-01-02")
	summaryResp, summaryErr := client.GetDailySummary(r.Context(), &hrv1.GetDailySummaryReq{
		UserId: userID,
		Date:   today,
	})
	if summaryErr == nil && summaryResp.Summary != nil {
		status["today_total_minutes"] = summaryResp.Summary.NetWorkMinutes

		// Determine ArbZG severity based on net work minutes
		netMinutes := int(summaryResp.Summary.NetWorkMinutes)
		switch {
		case netMinutes > 600:
			status["arbzg_severity"] = "error"
		case netMinutes >= 540:
			status["arbzg_severity"] = "warning"
		case netMinutes >= 480:
			status["arbzg_severity"] = "info"
		default:
			status["arbzg_severity"] = "none"
		}
	}

	response.JSON(w, http.StatusOK, status)
}

func (h *HRRoutes) HandleListWorkTimeEntries(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	page, pageSize := parsePagination(r, 1, 50)

	grpcReq := &hrv1.ListWorkTimeEntriesReq{
		TenantId:   tenantID,
		EmployeeId: r.URL.Query().Get("employee_id"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		Page:       int32(page),
		PerPage:    int32(pageSize),
	}

	resp, err := client.ListWorkTimeEntries(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	entriesJSON, err := hrMarshalSlice(resp.Entries)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"entries": entriesJSON,
		"total":   resp.Total,
	})
}

func (h *HRRoutes) HandleDailySummary(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	resp, err := client.GetDailySummary(r.Context(), &hrv1.GetDailySummaryReq{
		UserId: userID,
		Date:   date,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Summary)
}

func (h *HRRoutes) HandleWeeklySummary(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	weekStart := r.URL.Query().Get("week_start")
	if weekStart == "" {
		// Default to current week's Monday
		now := time.Now()
		for now.Weekday() != time.Monday {
			now = now.AddDate(0, 0, -1)
		}
		weekStart = now.Format("2006-01-02")
	}

	resp, err := client.GetWeeklySummary(r.Context(), &hrv1.GetWeeklySummaryReq{
		UserId:    userID,
		WeekStart: weekStart,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Summary)
}

type submitCorrectionHTTPReq struct {
	OriginalEntryID       string `json:"original_entry_id"        validate:"omitempty,uuid"`
	CorrectedClockIn      string `json:"corrected_clock_in"        validate:"required"`
	CorrectedClockOut     string `json:"corrected_clock_out"       validate:"required"`
	CorrectedBreakMinutes int32  `json:"corrected_break_minutes"   validate:"gte=0"`
	Reason                string `json:"reason"`
}

func (h *HRRoutes) HandleSubmitCorrection(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[submitCorrectionHTTPReq](w, r)
	if !ok {
		return
	}

	// RFC3339 parsing remains as code — tags only assert non-empty
	clockIn, err := time.Parse(time.RFC3339, req.CorrectedClockIn)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid corrected_clock_in format, use RFC3339")
		return
	}

	clockOut, err := time.Parse(time.RFC3339, req.CorrectedClockOut)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid corrected_clock_out format, use RFC3339")
		return
	}

	resp, err := client.SubmitTimeCorrection(r.Context(), &hrv1.SubmitTimeCorrectionReq{
		TenantId:              tenantID,
		UserId:                userID,
		OriginalEntryId:       req.OriginalEntryID,
		CorrectedClockIn:      timestamppb.New(clockIn),
		CorrectedClockOut:     timestamppb.New(clockOut),
		CorrectedBreakMinutes: req.CorrectedBreakMinutes,
		Reason:                req.Reason,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Correction)
}

func (h *HRRoutes) HandleApproveCorrection(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")
	approverID := middleware.GetUserID(r.Context())

	resp, err := client.ApproveTimeCorrection(r.Context(), &hrv1.ApproveTimeCorrectionReq{
		Id:         id,
		ApproverId: approverID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Correction)
}

// ============================================================================
// Absence Calendar Handler
// ============================================================================

func (h *HRRoutes) HandleGetAbsenceCalendar(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	grpcReq := &hrv1.GetAbsenceCalendarReq{
		TenantId:   tenantID,
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		Department: r.URL.Query().Get("department"),
	}

	resp, err := client.GetAbsenceCalendar(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoList(w, http.StatusOK, resp.Absences)
}

// ============================================================================
// Employee Profile Handlers
// ============================================================================

func (h *HRRoutes) HandleListEmployees(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListEmployees(r.Context(), &hrv1.ListEmployeesReq{
		TenantId:   tenantID,
		Department: r.URL.Query().Get("department"),
		Page:       int32(page),
		PerPage:    int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	employeesJSON, err := hrMarshalSlice(resp.Employees)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"employees": employeesJSON,
		"total":     resp.Total,
	})
}

func (h *HRRoutes) HandleGetSelfProfile(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetEmployee(r.Context(), &hrv1.GetEmployeeReq{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Employee)
}

func (h *HRRoutes) HandleGetEmployee(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	resp, err := client.GetEmployee(r.Context(), &hrv1.GetEmployeeReq{
		UserId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Employee)
}

type updateEmployeeHTTPReq struct {
	Department      string `json:"department"`
	PositionTitle   string `json:"position_title"`
	ContractType    string `json:"contract_type"      validate:"omitempty,oneof=full_time part_time mini_job intern temporary"`
	WorkDaysPerWeek int32  `json:"work_days_per_week" validate:"omitempty,gte=1,lte=7"`
	AnnualLeaveDays int32  `json:"annual_leave_days"  validate:"omitempty,gte=0"`
	ManagerUserID   string `json:"manager_user_id"`
	StartDate       string `json:"start_date"`
	// Pointer so an omitted field leaves the flag untouched instead of
	// clearing it — the other fields here use their zero value for that.
	IsMinor *bool `json:"is_minor"`
}

func (h *HRRoutes) HandleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[updateEmployeeHTTPReq](w, r)
	if !ok {
		return
	}

	grpcReq := &hrv1.UpdateEmployeeReq{
		UserId:          id,
		Department:      req.Department,
		PositionTitle:   req.PositionTitle,
		ContractType:    contractTypeHTTPToProto(req.ContractType),
		WorkDaysPerWeek: req.WorkDaysPerWeek,
		AnnualLeaveDays: req.AnnualLeaveDays,
		ManagerUserId:   req.ManagerUserID,
		StartDate:       req.StartDate,
		IsMinor:         req.IsMinor,
	}

	resp, err := client.UpdateEmployee(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Employee)
}

type offboardEmployeeHTTPReq struct {
	LastWorkDay string `json:"lastWorkDay"                validate:"omitempty,datetime=2006-01-02"`
	ExitDate    string `json:"exitDate"                   validate:"required,datetime=2006-01-02"`
	// The five values the offboard dialog offers; the service checks the set
	// again, this only keeps a typo out of the service.
	ExitType        string `json:"exitType"           validate:"required,oneof=resignation termination fixed_term_expired mutual_termination retirement"`
	Reason          string `json:"reason"`
	Backfill        bool   `json:"backfill"`
	SuccessorUserID string `json:"successorUserId"    validate:"omitempty,uuid"`
}

// HandleOffboardEmployee ends an employment. The body is camelCase, unlike the
// rest of the HR routes: the desktop client posts the offboard dialog's form
// state unchanged (hr-client.ts:943), so snake_case keys would arrive empty.
func (h *HRRoutes) HandleOffboardEmployee(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	req, ok := decodeAndValidate[offboardEmployeeHTTPReq](w, r)
	if !ok {
		return
	}

	// The tenant and the acting user are not passed on: the service reads both
	// from the context, which is what makes the self-offboard guard hold.
	resp, err := client.OffboardEmployee(r.Context(), &hrv1.OffboardEmployeeReq{
		EmployeeId:      chi.URLParam(r, "id"),
		LastWorkDay:     req.LastWorkDay,
		ExitDate:        req.ExitDate,
		ExitType:        req.ExitType,
		Reason:          req.Reason,
		Backfill:        req.Backfill,
		SuccessorUserId: req.SuccessorUserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// The whole response, not resp.Employee: the client reads `{employee}`.
	response.Proto(w, http.StatusOK, resp)
}

type updateSelfProfileHTTPReq struct {
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone" validate:"omitempty,phone_dach"`
	AddressStreet         string `json:"address_street"`
	AddressCity           string `json:"address_city"`
	AddressPostalCode     string `json:"address_postal_code"     validate:"omitempty,plz_dach"`
	AddressCountry        string `json:"address_country"`
}

func (h *HRRoutes) HandleUpdateSelfProfile(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[updateSelfProfileHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateSelfProfile(r.Context(), &hrv1.UpdateSelfProfileReq{
		UserId:                userID,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		AddressStreet:         req.AddressStreet,
		AddressCity:           req.AddressCity,
		AddressPostalCode:     req.AddressPostalCode,
		AddressCountry:        req.AddressCountry,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Employee)
}

func (h *HRRoutes) HandleListEmployeeDocuments(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	employeeID := chi.URLParam(r, "id")

	resp, err := client.ListEmployeeDocuments(r.Context(), &hrv1.ListEmployeeDocumentsReq{
		EmployeeId: employeeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoList(w, http.StatusOK, resp.Documents)
}

// HandleListEmployeeDocumentCategories lists document categories, filtered by
// the visibility tiers the caller's team:documents:view scope permits for
// employee {id}'s Akte (hr_only/manager/employee — same tiers RLS migration
// 000127 already enforces on the documents themselves). Category master data
// carries no per-employee RLS of its own, so this is the only place that
// stops a caller opening a stranger's file from learning hr_only categories
// exist at all.
func (h *HRRoutes) HandleListEmployeeDocumentCategories(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	callerScope := middleware.PermissionScope(r.Context(), "team:documents", "view")

	resp, err := client.ListDocumentCategories(r.Context(), &hrv1.ListDocumentCategoriesReq{
		TenantId:    tenantID,
		CallerScope: callerScope,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "categories", resp.Categories, nil)
}

// HandleListPersonnelDocuments serves the Personalakte tab: every document of
// the tenant the caller may see, across employees. The visibility tiers
// (hr_only / manager / employee) are applied by RLS policy hr_document_access
// against the caller's roles, so this handler adds no filter of its own — and
// must not, or the two would drift apart.
func (h *HRRoutes) HandleListPersonnelDocuments(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListPersonnelDocuments(r.Context(), &hrv1.ListPersonnelDocumentsReq{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.ProtoListWrapped(w, http.StatusOK, "documents", resp.Documents, nil)
}

type createPersonnelDocumentHTTPReq struct {
	EmployeeID  string `json:"employee_id"  validate:"required,uuid"`
	Title       string `json:"title"        validate:"required,max=255"`
	CategoryKey string `json:"category"     validate:"required,max=50"`
	FileName    string `json:"file_name"    validate:"required,max=255"`
	FileSize    string `json:"file_size"    validate:"omitempty,max=50"`
	FileID      string `json:"file_id"      validate:"omitempty,uuid"`
	ExpiresAt   string `json:"expires_at"   validate:"omitempty,datetime=2006-01-02"`
	Notes       string `json:"notes"        validate:"omitempty,max=2000"`
}

// HandleCreatePersonnelDocument records a document in an employee's Akte.
// employee_id is required rather than defaulting to anyone: a document filed
// into the wrong personnel record is worse than a rejected request.
func (h *HRRoutes) HandleCreatePersonnelDocument(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createPersonnelDocumentHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.UploadEmployeeDocument(r.Context(), &hrv1.UploadEmployeeDocumentReq{
		TenantId:    tenantID,
		EmployeeId:  req.EmployeeID,
		CategoryKey: req.CategoryKey,
		FileId:      req.FileID,
		UploadedBy:  middleware.GetUserID(r.Context()),
		Title:       req.Title,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ExpiresAt:   req.ExpiresAt,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Document)
}

type uploadDocumentHTTPReq struct {
	CategoryID string `json:"category_id" validate:"omitempty,uuid"`
	FileID     string `json:"file_id"     validate:"omitempty,uuid"`
	Notes      string `json:"notes"`
}

func (h *HRRoutes) HandleUploadEmployeeDocument(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	employeeID := chi.URLParam(r, "id")
	uploadedBy := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[uploadDocumentHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.UploadEmployeeDocument(r.Context(), &hrv1.UploadEmployeeDocumentReq{
		TenantId:   tenantID,
		EmployeeId: employeeID,
		CategoryId: req.CategoryID,
		FileId:     req.FileID,
		UploadedBy: uploadedBy,
		Notes:      req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Document)
}

// ============================================================================
// HR Settings Handlers
// ============================================================================

func (h *HRRoutes) HandleGetHRSettings(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GetHRSettings(r.Context(), &hrv1.GetHRSettingsReq{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Settings)
}

type updateHRSettingsHTTPReq struct {
	AUThresholdDays        int32  `json:"au_threshold_days"         validate:"omitempty,gte=0"`
	ShowAbsenceReason      bool   `json:"show_absence_reason"`
	DefaultAnnualLeaveDays int32  `json:"default_annual_leave_days" validate:"omitempty,gte=0"`
	Timezone               string `json:"timezone"`
	WorkHoursPerDay        int32  `json:"work_hours_per_day"        validate:"omitempty,gte=1,lte=24"`
	MaxDailyHours          int32  `json:"max_daily_hours"           validate:"omitempty,gte=1,lte=24"`
	BreakAfterHours        int32  `json:"break_after_hours"         validate:"omitempty,gte=0,lte=24"`
}

func (h *HRRoutes) HandleUpdateHRSettings(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[updateHRSettingsHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateHRSettings(r.Context(), &hrv1.UpdateHRSettingsReq{
		TenantId:               tenantID,
		AuThresholdDays:        req.AUThresholdDays,
		ShowAbsenceReason:      req.ShowAbsenceReason,
		DefaultAnnualLeaveDays: req.DefaultAnnualLeaveDays,
		Timezone:               req.Timezone,
		// Work-hours policy fields (proto fields 6-8, require proto regen to compile)
		WorkHoursPerDay: req.WorkHoursPerDay,
		MaxDailyHours:   req.MaxDailyHours,
		BreakAfterHours: req.BreakAfterHours,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Settings)
}

// ============================================================================
// Employee Profile Handlers
// ============================================================================

type createEmployeeHTTPReq struct {
	UserID                string `json:"user_id"                 validate:"required,uuid"`
	Department            string `json:"department"`
	PositionTitle         string `json:"position_title"`
	ContractType          string `json:"contract_type"           validate:"omitempty,oneof=full_time part_time mini_job intern temporary"`
	WorkDaysPerWeek       int32  `json:"work_days_per_week"      validate:"omitempty,gte=1,lte=7"`
	AnnualLeaveDays       int32  `json:"annual_leave_days"       validate:"omitempty,gte=0"`
	ManagerUserID         string `json:"manager_user_id"         validate:"omitempty,uuid"`
	StartDate             string `json:"start_date"              validate:"required"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone" validate:"omitempty,phone_dach"`
	AddressStreet         string `json:"address_street"`
	AddressCity           string `json:"address_city"`
	AddressPostalCode     string `json:"address_postal_code"     validate:"omitempty,plz_dach"`
	AddressCountry        string `json:"address_country"`
	IsMinor               bool   `json:"is_minor"`
}

func (h *HRRoutes) HandleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createEmployeeHTTPReq](w, r)
	if !ok {
		return
	}

	grpcReq := &hrv1.CreateEmployeeReq{
		TenantId:              tenantID,
		UserId:                req.UserID,
		Department:            req.Department,
		PositionTitle:         req.PositionTitle,
		ContractType:          contractTypeHTTPToProto(req.ContractType),
		WorkDaysPerWeek:       req.WorkDaysPerWeek,
		AnnualLeaveDays:       req.AnnualLeaveDays,
		ManagerUserId:         req.ManagerUserID,
		StartDate:             req.StartDate,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		AddressStreet:         req.AddressStreet,
		AddressCity:           req.AddressCity,
		AddressPostalCode:     req.AddressPostalCode,
		AddressCountry:        req.AddressCountry,
		IsMinor:               req.IsMinor,
	}

	resp, err := client.CreateEmployee(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Employee)
}

// ============================================================================
// Extended Time Tracking Handlers (Fall-B)
// ============================================================================

type createManualEntryHTTPReq struct {
	ClockIn         string  `json:"clock_in"          validate:"required"`
	ClockOut        string  `json:"clock_out"         validate:"required"`
	BreakMinutes    int32   `json:"break_minutes"     validate:"gte=0"`
	CategoryID      string  `json:"category_id"       validate:"omitempty,uuid"`
	Activity        string  `json:"activity"`
	Note            string  `json:"note"`
	Billable        bool    `json:"billable"`
	LocationLat     float64 `json:"location_lat"`
	LocationLng     float64 `json:"location_lng"`
	LocationAddress string  `json:"location_address"`
}

func (h *HRRoutes) HandleCreateManualEntry(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		response.Error(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	req, ok := decodeAndValidate[createManualEntryHTTPReq](w, r)
	if !ok {
		return
	}

	clockIn, err := time.Parse(time.RFC3339, req.ClockIn)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid clock_in, use RFC3339")
		return
	}
	clockOut, err := time.Parse(time.RFC3339, req.ClockOut)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid clock_out, use RFC3339")
		return
	}

	grpcReq := &hrv1.CreateManualEntryReq{
		TenantId:        tenantID,
		UserId:          userID,
		ClockIn:         timestamppb.New(clockIn),
		ClockOut:        timestamppb.New(clockOut),
		BreakMinutes:    req.BreakMinutes,
		CategoryId:      req.CategoryID,
		Activity:        req.Activity,
		Note:            req.Note,
		Billable:        req.Billable,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		LocationAddress: req.LocationAddress,
		IdempotencyKey:  idempotencyKey,
	}

	resp, err := client.CreateManualEntry(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Entry)
}

func (h *HRRoutes) HandleGetTimeBalance(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetTimeBalance(r.Context(), &hrv1.GetTimeBalanceReq{
		TenantId: tenantID,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"balance_minutes":       resp.BalanceMinutes,
		"target_weekly_minutes": resp.TargetWeeklyMinutes,
		"period_start":          resp.PeriodStart,
		"as_of":                 resp.AsOf,
	})
}

func (h *HRRoutes) HandleGetTimeAnalytics(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "week"
	}

	resp, err := client.GetTimeAnalytics(r.Context(), &hrv1.GetTimeAnalyticsReq{
		TenantId: tenantID,
		UserId:   userID,
		Range:    rangeParam,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	dayTrendJSON, err := hrMarshalSlice(resp.DayTrend)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	byProjectJSON, err := hrMarshalSlice(resp.ByProject)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"total_minutes":     resp.TotalMinutes,
		"target_minutes":    resp.TargetMinutes,
		"overtime_minutes":  resp.OvertimeMinutes,
		"avg_daily_minutes": resp.AvgDailyMinutes,
		"day_trend":         dayTrendJSON,
		"by_project":        byProjectJSON,
	})
}

func (h *HRRoutes) HandleGetTeamTime(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GetTeamTime(r.Context(), &hrv1.GetTeamTimeReq{
		TenantId:  tenantID,
		WeekStart: r.URL.Query().Get("week_start"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	teamJSON, err := hrMarshalSlice(resp.Team)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"team": teamJSON,
	})
}

func (h *HRRoutes) HandleGetMyWeekStatus(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetMyWeekStatus(r.Context(), &hrv1.GetMyWeekStatusReq{
		TenantId:  tenantID,
		UserId:    userID,
		WeekStart: r.URL.Query().Get("week_start"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	weekApprovalJSON, err := cannedResponseMarshaler.Marshal(resp.WeekApproval)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"week_approval":  json.RawMessage(weekApprovalJSON),
		"total_minutes":  resp.TotalMinutes,
		"target_minutes": resp.TargetMinutes,
	})
}

type submitWeekHTTPReq struct {
	WeekStart string `json:"week_start" validate:"required"`
}

func (h *HRRoutes) HandleSubmitWeek(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[submitWeekHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.SubmitWeek(r.Context(), &hrv1.SubmitWeekReq{
		TenantId:  tenantID,
		UserId:    userID,
		WeekStart: req.WeekStart,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.WeekApproval)
}

type approveWeekHTTPReq struct {
	EmployeeID string `json:"employee_id" validate:"required,uuid"`
	WeekStart  string `json:"week_start"  validate:"required"`
}

func (h *HRRoutes) HandleApproveWeek(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	approverID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[approveWeekHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.ApproveWeek(r.Context(), &hrv1.ApproveWeekReq{
		TenantId:   tenantID,
		ApproverId: approverID,
		EmployeeId: req.EmployeeID,
		WeekStart:  req.WeekStart,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.WeekApproval)
}

type rejectWeekHTTPReq struct {
	EmployeeID      string `json:"employee_id"       validate:"required,uuid"`
	WeekStart       string `json:"week_start"        validate:"required"`
	RejectionReason string `json:"rejection_reason"`
}

func (h *HRRoutes) HandleRejectWeek(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	approverID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[rejectWeekHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.RejectWeek(r.Context(), &hrv1.RejectWeekReq{
		TenantId:        tenantID,
		ApproverId:      approverID,
		EmployeeId:      req.EmployeeID,
		WeekStart:       req.WeekStart,
		RejectionReason: req.RejectionReason,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.WeekApproval)
}

type reopenWeekHTTPReq struct {
	EmployeeID string `json:"employee_id" validate:"required,uuid"`
	WeekStart  string `json:"week_start"  validate:"required"`
	Reason     string `json:"reason"`
}

// HandleReopenWeek unlocks a submitted or approved week for corrections.
// POST /api/v1/hr/time/weeks/reopen
func (h *HRRoutes) HandleReopenWeek(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	approverID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[reopenWeekHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.ReopenWeek(r.Context(), &hrv1.ReopenWeekReq{
		TenantId:   tenantID,
		ApproverId: approverID,
		EmployeeId: req.EmployeeID,
		WeekStart:  req.WeekStart,
		Reason:     req.Reason,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.WeekApproval)
}

// ============================================================================
// Time Category Handlers
// ============================================================================

func (h *HRRoutes) HandleListTimeCategories(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListTimeCategories(r.Context(), &hrv1.ListTimeCategoriesReq{TenantId: tenantID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	categoriesJSON, err := hrMarshalSlice(resp.Categories)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"categories": categoriesJSON,
	})
}

type createTimeCategoryHTTPReq struct {
	Name      string `json:"name"       validate:"required"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	IsDefault bool   `json:"is_default"`
}

func (h *HRRoutes) HandleCreateTimeCategory(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createTimeCategoryHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTimeCategory(r.Context(), &hrv1.CreateTimeCategoryReq{
		TenantId:  tenantID,
		Name:      req.Name,
		Color:     req.Color,
		Icon:      req.Icon,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Category)
}

type updateTimeCategoryHTTPReq struct {
	Name      string `json:"name"       validate:"required"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	IsDefault bool   `json:"is_default"`
}

func (h *HRRoutes) HandleUpdateTimeCategory(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	id := chi.URLParam(r, "id")
	req, ok := decodeAndValidate[updateTimeCategoryHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateTimeCategory(r.Context(), &hrv1.UpdateTimeCategoryReq{
		Id:        id,
		TenantId:  tenantID,
		Name:      req.Name,
		Color:     req.Color,
		Icon:      req.Icon,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp.Category)
}

func (h *HRRoutes) HandleDeleteTimeCategory(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	_, err = client.DeleteTimeCategory(r.Context(), &hrv1.DeleteTimeCategoryReq{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Time Template Handlers
// ============================================================================

func (h *HRRoutes) HandleListTimeTemplates(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListTimeTemplates(r.Context(), &hrv1.ListTimeTemplatesReq{TenantId: tenantID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	templatesJSON, err := hrMarshalSlice(resp.Templates)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"templates": templatesJSON,
	})
}

type createTimeTemplateHTTPReq struct {
	Name             string `json:"name"              validate:"required"`
	CategoryID       string `json:"category_id"       validate:"omitempty,uuid"`
	Description      string `json:"description"`
	EstimatedMinutes int32  `json:"estimated_minutes" validate:"gte=0"`
}

func (h *HRRoutes) HandleCreateTimeTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createTimeTemplateHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTimeTemplate(r.Context(), &hrv1.CreateTimeTemplateReq{
		TenantId:         tenantID,
		Name:             req.Name,
		CategoryId:       req.CategoryID,
		Description:      req.Description,
		EstimatedMinutes: req.EstimatedMinutes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Template)
}

func (h *HRRoutes) HandleDeleteTimeTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	id := chi.URLParam(r, "id")

	_, err = client.DeleteTimeTemplate(r.Context(), &hrv1.DeleteTimeTemplateReq{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Time Project Handlers
// ============================================================================

func (h *HRRoutes) HandleListTimeProjects(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListTimeProjects(r.Context(), &hrv1.ListTimeProjectsReq{TenantId: tenantID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	projectsJSON, err := hrMarshalSlice(resp.Projects)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"projects": projectsJSON,
	})
}

type createTimeProjectHTTPReq struct {
	Name            string `json:"name"             validate:"required"`
	CustomerName    string `json:"customer_name"`
	Color           string `json:"color"`
	BillableDefault bool   `json:"billable_default"`
}

func (h *HRRoutes) HandleCreateTimeProject(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createTimeProjectHTTPReq](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateTimeProject(r.Context(), &hrv1.CreateTimeProjectReq{
		TenantId:        tenantID,
		Name:            req.Name,
		CustomerName:    req.CustomerName,
		Color:           req.Color,
		BillableDefault: req.BillableDefault,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp.Project)
}

// ============================================================================
// Proto Enum Helpers
// ============================================================================

func contractTypeHTTPToProto(ct string) hrv1.ContractType {
	switch ct {
	case "full_time":
		return hrv1.ContractType_CONTRACT_FULL_TIME
	case "part_time":
		return hrv1.ContractType_CONTRACT_PART_TIME
	case "mini_job":
		return hrv1.ContractType_CONTRACT_MINI_JOB
	case "intern":
		return hrv1.ContractType_CONTRACT_INTERN
	case "temporary":
		return hrv1.ContractType_CONTRACT_TEMPORARY
	default:
		return hrv1.ContractType_CONTRACT_TYPE_UNSPECIFIED
	}
}
