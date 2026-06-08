package gateway

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

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

	// Time tracking
	r.Route("/api/v1/hr/time", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "write")).Post("/clock-in", h.HandleClockIn)
		r.With(middleware.RequirePermission("hr", "write")).Post("/clock-out", h.HandleClockOut)
		r.With(middleware.RequirePermission("hr", "write")).Post("/break/start", h.HandleBreakStart)
		r.With(middleware.RequirePermission("hr", "write")).Post("/break/end", h.HandleBreakEnd)
		r.With(middleware.RequirePermission("hr", "read")).Get("/active", h.HandleGetActiveShift)
		r.With(middleware.RequirePermission("hr", "read")).Get("/status", h.HandleGetWorkTimeStatus)
		r.With(middleware.RequirePermission("hr", "read")).Get("/entries", h.HandleListWorkTimeEntries)
		r.With(middleware.RequirePermission("hr", "read")).Get("/summary/daily", h.HandleDailySummary)
		r.With(middleware.RequirePermission("hr", "read")).Get("/summary/weekly", h.HandleWeeklySummary)
		r.With(middleware.RequirePermission("hr", "write")).Post("/corrections", h.HandleSubmitCorrection)
		r.With(middleware.RequirePermission("hr", "write")).Post("/corrections/{id}/approve", h.HandleApproveCorrection)

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

	// Employee profiles
	r.Route("/api/v1/hr/employees", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("hr", "read")).Get("/", h.HandleListEmployees)
		r.With(middleware.RequirePermission("hr", "read")).Get("/me", h.HandleGetSelfProfile)
		r.With(middleware.RequirePermission("hr", "write")).Put("/me", h.HandleUpdateSelfProfile)
		r.With(middleware.RequirePermission("hr", "read")).Get("/{id}", h.HandleGetEmployee)
		r.With(middleware.RequirePermission("hr", "admin")).Put("/{id}", h.HandleUpdateEmployee)
		r.With(middleware.RequirePermission("hr", "read")).Get("/{id}/documents", h.HandleListEmployeeDocuments)
		r.With(middleware.RequirePermission("hr", "write")).Post("/{id}/documents", h.HandleUploadEmployeeDocument)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusCreated, resp.LeaveRequest)
}

func (h *HRRoutes) HandleListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"leave_requests": resp.LeaveRequests,
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

	response.JSON(w, http.StatusOK, resp.LeaveRequest)
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

	response.JSON(w, http.StatusOK, resp.LeaveRequest)
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

	response.JSON(w, http.StatusOK, resp.LeaveRequest)
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

	response.JSON(w, http.StatusOK, resp.LeaveRequest)
}

func (h *HRRoutes) HandleGetLeaveBalance(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, resp.Balance)
}

func (h *HRRoutes) HandleGetEmployeeLeaveBalance(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, resp.Balance)
}

func (h *HRRoutes) HandleListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.ListLeaveTypes(r.Context(), &hrv1.ListLeaveTypesReq{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.LeaveTypes)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"leave_request":        resp.LeaveRequest,
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusCreated, resp.Entry)
}

func (h *HRRoutes) HandleClockOut(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	result := map[string]interface{}{
		"entry": resp.Entry,
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

	response.JSON(w, http.StatusCreated, resp.BreakEntry)
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

	response.JSON(w, http.StatusOK, resp.BreakEntry)
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

	result := map[string]interface{}{
		"entry":        resp.Entry,
		"is_on_break":  resp.IsOnBreak,
		"active_break": resp.ActiveBreak,
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"entries": resp.Entries,
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

	response.JSON(w, http.StatusOK, resp.Summary)
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

	response.JSON(w, http.StatusOK, resp.Summary)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusCreated, resp.Correction)
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

	response.JSON(w, http.StatusOK, resp.Correction)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, resp.Absences)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"employees": resp.Employees,
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

	response.JSON(w, http.StatusOK, resp.Employee)
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

	response.JSON(w, http.StatusOK, resp.Employee)
}

type updateEmployeeHTTPReq struct {
	Department      string `json:"department"`
	PositionTitle   string `json:"position_title"`
	ContractType    string `json:"contract_type"      validate:"omitempty,oneof=full_time part_time mini_job intern temporary"`
	WorkDaysPerWeek int32  `json:"work_days_per_week" validate:"omitempty,gte=1,lte=7"`
	AnnualLeaveDays int32  `json:"annual_leave_days"  validate:"omitempty,gte=0"`
	ManagerUserID   string `json:"manager_user_id"`
	StartDate       string `json:"start_date"`
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
	}

	resp, err := client.UpdateEmployee(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Employee)
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

	response.JSON(w, http.StatusOK, resp.Employee)
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

	response.JSON(w, http.StatusOK, resp.Documents)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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

	response.JSON(w, http.StatusCreated, resp.Document)
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
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.GetHRSettings(r.Context(), &hrv1.GetHRSettingsReq{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Settings)
}

type updateHRSettingsHTTPReq struct {
	AUThresholdDays        int32  `json:"au_threshold_days"         validate:"omitempty,gte=0"`
	ShowAbsenceReason      bool   `json:"show_absence_reason"`
	DefaultAnnualLeaveDays int32  `json:"default_annual_leave_days" validate:"omitempty,gte=0"`
	Timezone               string `json:"timezone"`
}

func (h *HRRoutes) HandleUpdateHRSettings(w http.ResponseWriter, r *http.Request) {
	client, err := h.getHRClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
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
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Settings)
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
