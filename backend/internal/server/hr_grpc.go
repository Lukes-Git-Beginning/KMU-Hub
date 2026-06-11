package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/hr/absence"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// HRGRPCServer implements the HRService gRPC service.
type HRGRPCServer struct {
	hrv1.UnimplementedHRServiceServer
	leaveService        *leave.Service
	timetrackingService *timetracking.Service
	employeeService     *employee.Service
	absenceService      *absence.Service
	settingsRepo        leave.HRSettingsRepository
}

// NewHRGRPCServer creates a new HRGRPCServer with all HR services.
func NewHRGRPCServer(
	leaveService *leave.Service,
	timetrackingService *timetracking.Service,
	employeeService *employee.Service,
	absenceService *absence.Service,
	settingsRepo leave.HRSettingsRepository,
) *HRGRPCServer {
	return &HRGRPCServer{
		leaveService:        leaveService,
		timetrackingService: timetrackingService,
		employeeService:     employeeService,
		absenceService:      absenceService,
		settingsRepo:        settingsRepo,
	}
}

// ============================================================================
// Leave Management RPCs
// ============================================================================

func (s *HRGRPCServer) CreateLeaveRequest(ctx context.Context, req *hrv1.CreateLeaveRequestReq) (*hrv1.CreateLeaveRequestResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	leaveTypeID, err := uuid.Parse(req.GetLeaveTypeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid leave_type_id")
	}

	startDate, err := time.Parse("2006-01-02", req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date")
	}

	endDate, err := time.Parse("2006-01-02", req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date")
	}

	input := leave.CreateLeaveInput{
		LeaveTypeID:    leaveTypeID,
		StartDate:      startDate,
		EndDate:        endDate,
		IsHalfDayStart: req.GetIsHalfDayStart(),
		IsHalfDayEnd:   req.GetIsHalfDayEnd(),
		Reason:         req.GetReason(),
	}

	if req.GetHalfDayPeriodStart() != hrv1.HalfDayPeriod_HALF_DAY_PERIOD_UNSPECIFIED {
		p := halfDayPeriodFromProto(req.GetHalfDayPeriodStart())
		input.HalfDayPeriodStart = &p
	}
	if req.GetHalfDayPeriodEnd() != hrv1.HalfDayPeriod_HALF_DAY_PERIOD_UNSPECIFIED {
		p := halfDayPeriodFromProto(req.GetHalfDayPeriodEnd())
		input.HalfDayPeriodEnd = &p
	}

	result, err := s.leaveService.CreateLeaveRequest(ctx, tenantID, userID, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.CreateLeaveRequestResp{
		LeaveRequest: toProtoLeaveRequest(result.Request),
	}, nil
}

func (s *HRGRPCServer) GetLeaveRequest(ctx context.Context, req *hrv1.GetLeaveRequestReq) (*hrv1.GetLeaveRequestResp, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	lr, err := s.leaveService.GetLeaveRequest(ctx, id)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetLeaveRequestResp{
		LeaveRequest: toProtoLeaveRequest(lr),
	}, nil
}

func (s *HRGRPCServer) ListLeaveRequests(ctx context.Context, req *hrv1.ListLeaveRequestsReq) (*hrv1.ListLeaveRequestsResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := leave.LeaveRequestFilter{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PerPage:  int(req.GetPerPage()),
	}

	if req.GetEmployeeId() != "" {
		empID, empErr := uuid.Parse(req.GetEmployeeId())
		if empErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
		}
		filter.EmployeeID = &empID
	}

	if req.GetStatus() != "" {
		s := req.GetStatus()
		filter.Status = &s
	}

	if req.GetStartDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetStartDate())
		if parseErr == nil {
			filter.StartDateFrom = &t
		}
	}
	if req.GetEndDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetEndDate())
		if parseErr == nil {
			filter.StartDateTo = &t
		}
	}

	requests, total, err := s.leaveService.ListLeaveRequests(ctx, filter)
	if err != nil {
		return nil, mapHRError(err)
	}

	protoRequests := make([]*hrv1.LeaveRequest, 0, len(requests))
	for _, r := range requests {
		protoRequests = append(protoRequests, toProtoLeaveRequest(r))
	}

	return &hrv1.ListLeaveRequestsResp{
		LeaveRequests: protoRequests,
		Total:         int32(total),
		Page:          req.GetPage(),
		PerPage:       req.GetPerPage(),
	}, nil
}

func (s *HRGRPCServer) ApproveLeaveRequest(ctx context.Context, req *hrv1.ApproveLeaveRequestReq) (*hrv1.ApproveLeaveRequestResp, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	approverID, err := uuid.Parse(req.GetApproverId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid approver_id")
	}

	result, err := s.leaveService.ApproveLeaveRequest(ctx, id, approverID, req.GetComment())
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.ApproveLeaveRequestResp{
		LeaveRequest: toProtoLeaveRequest(result.Request),
	}, nil
}

func (s *HRGRPCServer) RejectLeaveRequest(ctx context.Context, req *hrv1.RejectLeaveRequestReq) (*hrv1.RejectLeaveRequestResp, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	approverID, err := uuid.Parse(req.GetApproverId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid approver_id")
	}

	if err := s.leaveService.RejectLeaveRequest(ctx, id, approverID, req.GetComment()); err != nil {
		return nil, mapHRError(err)
	}

	// Re-fetch to return updated state
	lr, _ := s.leaveService.GetLeaveRequest(ctx, id)
	return &hrv1.RejectLeaveRequestResp{
		LeaveRequest: toProtoLeaveRequest(lr),
	}, nil
}

func (s *HRGRPCServer) CancelLeaveRequest(ctx context.Context, req *hrv1.CancelLeaveRequestReq) (*hrv1.CancelLeaveRequestResp, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.leaveService.CancelLeaveRequest(ctx, id, userID); err != nil {
		return nil, mapHRError(err)
	}

	lr, _ := s.leaveService.GetLeaveRequest(ctx, id)
	return &hrv1.CancelLeaveRequestResp{
		LeaveRequest: toProtoLeaveRequest(lr),
	}, nil
}

func (s *HRGRPCServer) GetLeaveBalance(ctx context.Context, req *hrv1.GetLeaveBalanceReq) (*hrv1.GetLeaveBalanceResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	balance, err := s.leaveService.GetLeaveBalance(ctx, tenantID, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetLeaveBalanceResp{
		Balance: toProtoLeaveBalance(balance),
	}, nil
}

func (s *HRGRPCServer) GetEmployeeLeaveBalance(ctx context.Context, req *hrv1.GetEmployeeLeaveBalanceReq) (*hrv1.GetEmployeeLeaveBalanceResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
	}

	balance, err := s.leaveService.GetLeaveBalance(ctx, tenantID, employeeID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetEmployeeLeaveBalanceResp{
		Balance: toProtoLeaveBalance(balance),
	}, nil
}

func (s *HRGRPCServer) ListLeaveTypes(ctx context.Context, req *hrv1.ListLeaveTypesReq) (*hrv1.ListLeaveTypesResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	types, err := s.leaveService.ListLeaveTypes(ctx, tenantID)
	if err != nil {
		return nil, mapHRError(err)
	}

	protoTypes := make([]*hrv1.LeaveType, 0, len(types))
	for _, t := range types {
		protoTypes = append(protoTypes, toProtoLeaveType(t))
	}

	return &hrv1.ListLeaveTypesResp{
		LeaveTypes: protoTypes,
	}, nil
}

func (s *HRGRPCServer) RecordSickLeave(ctx context.Context, req *hrv1.RecordSickLeaveReq) (*hrv1.RecordSickLeaveResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	startDate, err := time.Parse("2006-01-02", req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date")
	}

	endDate, err := time.Parse("2006-01-02", req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date")
	}

	input := leave.SickLeaveInput{
		StartDate: startDate,
		EndDate:   endDate,
		Reason:    req.GetNotes(),
	}

	lr, err := s.leaveService.RecordSickLeave(ctx, tenantID, userID, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.RecordSickLeaveResp{
		LeaveRequest:       toProtoLeaveRequest(lr),
		AuDocumentRequired: lr.AUDocumentRequired,
	}, nil
}

// ============================================================================
// Time Tracking RPCs
// ============================================================================

func (s *HRGRPCServer) ClockIn(ctx context.Context, req *hrv1.ClockInReq) (*hrv1.ClockInResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	entry, _, err := s.timetrackingService.ClockIn(ctx, tenantID, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.ClockInResp{
		Entry: toProtoWorkTimeEntry(entry),
	}, nil
}

func (s *HRGRPCServer) ClockOut(ctx context.Context, req *hrv1.ClockOutReq) (*hrv1.ClockOutResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	entry, checkResult, err := s.timetrackingService.ClockOut(ctx, tenantID, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	resp := &hrv1.ClockOutResp{
		Entry: toProtoWorkTimeEntry(entry),
	}

	if checkResult != nil {
		resp.Severity = string(checkResult.Severity)
		resp.ArbzgMessage = checkResult.Message
	}

	return resp, nil
}

func (s *HRGRPCServer) StartBreak(ctx context.Context, req *hrv1.StartBreakReq) (*hrv1.StartBreakResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	breakEntry, err := s.timetrackingService.StartBreak(ctx, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.StartBreakResp{
		BreakEntry: toProtoBreakEntry(breakEntry),
	}, nil
}

func (s *HRGRPCServer) EndBreak(ctx context.Context, req *hrv1.EndBreakReq) (*hrv1.EndBreakResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	breakEntry, err := s.timetrackingService.EndBreak(ctx, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.EndBreakResp{
		BreakEntry: toProtoBreakEntry(breakEntry),
	}, nil
}

func (s *HRGRPCServer) GetActiveShift(ctx context.Context, req *hrv1.GetActiveShiftReq) (*hrv1.GetActiveShiftResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	shift, breaks, err := s.timetrackingService.GetActiveShift(ctx, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	resp := &hrv1.GetActiveShiftResp{}
	if shift != nil {
		shift.Breaks = make([]models.HRBreakEntry, 0, len(breaks))
		for _, b := range breaks {
			shift.Breaks = append(shift.Breaks, *b)
		}
		resp.Entry = toProtoWorkTimeEntry(shift)

		// Check if currently on break
		for _, b := range breaks {
			if b.EndTime == nil {
				resp.IsOnBreak = true
				resp.ActiveBreak = toProtoBreakEntry(b)
				break
			}
		}
	}

	return resp, nil
}

func (s *HRGRPCServer) ListWorkTimeEntries(ctx context.Context, req *hrv1.ListWorkTimeEntriesReq) (*hrv1.ListWorkTimeEntriesResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := timetracking.WorkTimeFilter{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PerPage:  int(req.GetPerPage()),
	}

	if req.GetEmployeeId() != "" {
		empID, empErr := uuid.Parse(req.GetEmployeeId())
		if empErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
		}
		filter.EmployeeID = &empID
	}

	if req.GetStartDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetStartDate())
		if parseErr == nil {
			filter.DateFrom = &t
		}
	}
	if req.GetEndDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetEndDate())
		if parseErr == nil {
			filter.DateTo = &t
		}
	}

	entries, total, err := s.timetrackingService.ListWorkTimeEntries(ctx, filter)
	if err != nil {
		return nil, mapHRError(err)
	}

	protoEntries := make([]*hrv1.WorkTimeEntry, 0, len(entries))
	for _, e := range entries {
		protoEntries = append(protoEntries, toProtoWorkTimeEntry(e))
	}

	return &hrv1.ListWorkTimeEntriesResp{
		Entries: protoEntries,
		Total:   int32(total),
		Page:    req.GetPage(),
		PerPage: req.GetPerPage(),
	}, nil
}

func (s *HRGRPCServer) GetDailySummary(ctx context.Context, req *hrv1.GetDailySummaryReq) (*hrv1.GetDailySummaryResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	date, err := time.Parse("2006-01-02", req.GetDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date")
	}

	summary, err := s.timetrackingService.GetDailySummary(ctx, userID, date)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetDailySummaryResp{
		Summary: toProtoDailySummary(summary),
	}, nil
}

func (s *HRGRPCServer) GetWeeklySummary(ctx context.Context, req *hrv1.GetWeeklySummaryReq) (*hrv1.GetWeeklySummaryResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	weekStart, err := time.Parse("2006-01-02", req.GetWeekStart())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid week_start")
	}

	summary, err := s.timetrackingService.GetWeeklySummary(ctx, userID, weekStart)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetWeeklySummaryResp{
		Summary: toProtoWeeklySummary(summary),
	}, nil
}

func (s *HRGRPCServer) SubmitTimeCorrection(ctx context.Context, req *hrv1.SubmitTimeCorrectionReq) (*hrv1.SubmitTimeCorrectionResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	originalID, err := uuid.Parse(req.GetOriginalEntryId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid original_entry_id")
	}

	input := timetracking.CorrectionInput{
		OriginalEntryID:      originalID,
		CorrectedClockIn:     req.GetCorrectedClockIn().AsTime(),
		CorrectedClockOut:    req.GetCorrectedClockOut().AsTime(),
		CorrectedBreakMinutes: int(req.GetCorrectedBreakMinutes()),
		Reason:               req.GetReason(),
	}

	correction, err := s.timetrackingService.SubmitTimeCorrection(ctx, tenantID, userID, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.SubmitTimeCorrectionResp{
		Correction: toProtoWorkTimeEntry(correction),
	}, nil
}

func (s *HRGRPCServer) ApproveTimeCorrection(ctx context.Context, req *hrv1.ApproveTimeCorrectionReq) (*hrv1.ApproveTimeCorrectionResp, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	approverID, err := uuid.Parse(req.GetApproverId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid approver_id")
	}

	correction, err := s.timetrackingService.ApproveTimeCorrection(ctx, id, approverID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.ApproveTimeCorrectionResp{
		Correction: toProtoWorkTimeEntry(correction),
	}, nil
}

// ============================================================================
// Absence Calendar RPC
// ============================================================================

func (s *HRGRPCServer) GetAbsenceCalendar(ctx context.Context, req *hrv1.GetAbsenceCalendarReq) (*hrv1.GetAbsenceCalendarResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	startDate, err := time.Parse("2006-01-02", req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date")
	}

	endDate, err := time.Parse("2006-01-02", req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date")
	}

	filter := absence.AbsenceFilter{
		TenantID:   tenantID,
		StartDate:  startDate,
		EndDate:    endDate,
		Department: req.GetDepartment(),
	}

	entries, err := s.absenceService.GetAbsenceCalendar(ctx, tenantID, filter)
	if err != nil {
		return nil, mapHRError(err)
	}

	protoEntries := make([]*hrv1.AbsenceEntry, 0, len(entries))
	for _, e := range entries {
		protoEntries = append(protoEntries, &hrv1.AbsenceEntry{
			EmployeeId:    e.EmployeeID.String(),
			EmployeeName:  e.EmployeeName,
			Department:    e.Department,
			StartDate:     e.StartDate.Format("2006-01-02"),
			EndDate:       e.EndDate.Format("2006-01-02"),
			LeaveTypeName: e.LeaveTypeName,
			LeaveTypeColor: e.Color,
		})
	}

	return &hrv1.GetAbsenceCalendarResp{
		Absences: protoEntries,
	}, nil
}

// ============================================================================
// Employee Profile RPCs
// ============================================================================

func (s *HRGRPCServer) ListEmployees(ctx context.Context, req *hrv1.ListEmployeesReq) (*hrv1.ListEmployeesResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := employee.EmployeeFilter{
		TenantID:   tenantID,
		Department: req.GetDepartment(),
		Page:       int(req.GetPage()),
		PerPage:    int(req.GetPerPage()),
	}

	employees, total, err := s.employeeService.ListEmployees(ctx, filter)
	if err != nil {
		return nil, mapHRError(err)
	}

	protoEmployees := make([]*hrv1.EmployeeProfile, 0, len(employees))
	for _, e := range employees {
		protoEmployees = append(protoEmployees, toProtoEmployeeProfile(e))
	}

	return &hrv1.ListEmployeesResp{
		Employees: protoEmployees,
		Total:     int32(total),
		Page:      req.GetPage(),
		PerPage:   req.GetPerPage(),
	}, nil
}

func (s *HRGRPCServer) GetEmployee(ctx context.Context, req *hrv1.GetEmployeeReq) (*hrv1.GetEmployeeResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	emp, err := s.employeeService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetEmployeeResp{
		Employee: toProtoEmployeeProfile(emp),
	}, nil
}

func (s *HRGRPCServer) UpdateEmployee(ctx context.Context, req *hrv1.UpdateEmployeeReq) (*hrv1.UpdateEmployeeResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Look up employee by user ID
	emp, err := s.employeeService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, mapHRError(err)
	}

	input := employee.UpdateEmployeeInput{}

	if req.GetDepartment() != "" {
		d := req.GetDepartment()
		input.Department = &d
	}
	if req.GetPositionTitle() != "" {
		p := req.GetPositionTitle()
		input.PositionTitle = &p
	}
	if req.GetContractType() != hrv1.ContractType_CONTRACT_TYPE_UNSPECIFIED {
		ct := contractTypeFromProto(req.GetContractType())
		input.ContractType = &ct
	}
	if req.GetWorkDaysPerWeek() > 0 {
		w := int(req.GetWorkDaysPerWeek())
		input.WorkDaysPerWeek = &w
	}
	if req.GetAnnualLeaveDays() > 0 {
		a := int(req.GetAnnualLeaveDays())
		input.AnnualLeaveDays = &a
	}
	if req.GetManagerUserId() != "" {
		mgrID, mgrErr := uuid.Parse(req.GetManagerUserId())
		if mgrErr == nil {
			input.ManagerUserID = &mgrID
		}
	}
	if req.GetStartDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetStartDate())
		if parseErr == nil {
			input.StartDate = &t
		}
	}

	// Admin/manager caller role (gRPC layer assumes authorized caller)
	updated, err := s.employeeService.UpdateEmployee(ctx, emp.ID, input, "admin")
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.UpdateEmployeeResp{
		Employee: toProtoEmployeeProfile(updated),
	}, nil
}

func (s *HRGRPCServer) UpdateSelfProfile(ctx context.Context, req *hrv1.UpdateSelfProfileReq) (*hrv1.UpdateSelfProfileResp, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := employee.SelfProfileInput{}

	if req.GetEmergencyContactName() != "" {
		n := req.GetEmergencyContactName()
		input.EmergencyContactName = &n
	}
	if req.GetEmergencyContactPhone() != "" {
		p := req.GetEmergencyContactPhone()
		input.EmergencyContactPhone = &p
	}
	if req.GetAddressStreet() != "" {
		s := req.GetAddressStreet()
		input.AddressStreet = &s
	}
	if req.GetAddressCity() != "" {
		c := req.GetAddressCity()
		input.AddressCity = &c
	}
	if req.GetAddressPostalCode() != "" {
		p := req.GetAddressPostalCode()
		input.AddressPostalCode = &p
	}
	if req.GetAddressCountry() != "" {
		c := req.GetAddressCountry()
		input.AddressCountry = &c
	}

	updated, err := s.employeeService.UpdateSelfProfile(ctx, userID, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.UpdateSelfProfileResp{
		Employee: toProtoEmployeeProfile(updated),
	}, nil
}

func (s *HRGRPCServer) ListEmployeeDocuments(ctx context.Context, req *hrv1.ListEmployeeDocumentsReq) (*hrv1.ListEmployeeDocumentsResp, error) {
	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
	}

	// Use admin role for gRPC-level access (gateway will enforce actual user role)
	docs, err := s.employeeService.ListEmployeeDocuments(ctx, employeeID, uuid.Nil, "admin")
	if err != nil {
		return nil, mapHRError(err)
	}

	protoDocs := make([]*hrv1.EmployeeDocument, 0, len(docs))
	for _, d := range docs {
		protoDocs = append(protoDocs, toProtoEmployeeDocument(d))
	}

	return &hrv1.ListEmployeeDocumentsResp{
		Documents: protoDocs,
	}, nil
}

func (s *HRGRPCServer) UploadEmployeeDocument(ctx context.Context, req *hrv1.UploadEmployeeDocumentReq) (*hrv1.UploadEmployeeDocumentResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
	}

	categoryID, err := uuid.Parse(req.GetCategoryId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category_id")
	}

	fileID, err := uuid.Parse(req.GetFileId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	uploadedBy, err := uuid.Parse(req.GetUploadedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid uploaded_by")
	}

	input := employee.UploadDocumentInput{
		EmployeeID: employeeID,
		CategoryID: categoryID,
		FileID:     fileID,
		UploadedBy: uploadedBy,
		Notes:      req.GetNotes(),
	}

	doc, err := s.employeeService.UploadEmployeeDocument(ctx, tenantID, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.UploadEmployeeDocumentResp{
		Document: toProtoEmployeeDocument(doc),
	}, nil
}

func (s *HRGRPCServer) CreateEmployee(ctx context.Context, req *hrv1.CreateEmployeeReq) (*hrv1.CreateEmployeeResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := employee.CreateEmployeeInput{
		TenantID:              tenantID,
		UserID:                userID,
		Department:            req.GetDepartment(),
		PositionTitle:         req.GetPositionTitle(),
		ContractType:          contractTypeFromProto(req.GetContractType()),
		WorkDaysPerWeek:       int(req.GetWorkDaysPerWeek()),
		AnnualLeaveDays:       int(req.GetAnnualLeaveDays()),
		StartDate:             time.Now(), // default; overridden below if provided
		EmergencyContactName:  req.GetEmergencyContactName(),
		EmergencyContactPhone: req.GetEmergencyContactPhone(),
		AddressStreet:         req.GetAddressStreet(),
		AddressCity:           req.GetAddressCity(),
		AddressPostalCode:     req.GetAddressPostalCode(),
		AddressCountry:        req.GetAddressCountry(),
	}

	if req.GetManagerUserId() != "" {
		mgrID, mgrErr := uuid.Parse(req.GetManagerUserId())
		if mgrErr == nil {
			input.ManagerUserID = &mgrID
		}
	}

	if req.GetStartDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetStartDate())
		if parseErr == nil {
			input.StartDate = t
		}
	}

	profile, err := s.employeeService.CreateEmployee(ctx, input)
	if err != nil {
		return nil, mapHRError(err)
	}

	// Fetch with denormalized fields (user_name, user_email, manager_name)
	full, getErr := s.employeeService.GetEmployee(ctx, profile.ID)
	if getErr != nil {
		full = profile
	}

	return &hrv1.CreateEmployeeResp{
		Employee: toProtoEmployeeProfile(full),
	}, nil
}

// ============================================================================
// HR Settings RPCs
// ============================================================================

func (s *HRGRPCServer) GetHRSettings(ctx context.Context, req *hrv1.GetHRSettingsReq) (*hrv1.GetHRSettingsResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	settings, err := s.settingsRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.GetHRSettingsResp{
		Settings: toProtoHRSettings(settings),
	}, nil
}

func (s *HRGRPCServer) UpdateHRSettings(ctx context.Context, req *hrv1.UpdateHRSettingsReq) (*hrv1.UpdateHRSettingsResp, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	settings := &models.HRCompanySettings{
		TenantID:               tenantID,
		AUThresholdDays:        int(req.GetAuThresholdDays()),
		ShowAbsenceReason:      req.GetShowAbsenceReason(),
		DefaultAnnualLeaveDays: int(req.GetDefaultAnnualLeaveDays()),
		Timezone:               req.GetTimezone(),
		UpdatedAt:              time.Now(),
	}

	if err := s.settingsRepo.Upsert(ctx, settings); err != nil {
		return nil, mapHRError(err)
	}

	updated, err := s.settingsRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, mapHRError(err)
	}

	return &hrv1.UpdateHRSettingsResp{
		Settings: toProtoHRSettings(updated),
	}, nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toProtoLeaveRequest(lr *models.LeaveRequest) *hrv1.LeaveRequest {
	if lr == nil {
		return nil
	}
	plr := &hrv1.LeaveRequest{
		Id:                 lr.ID.String(),
		TenantId:           lr.TenantID.String(),
		EmployeeId:         lr.EmployeeID.String(),
		LeaveTypeId:        lr.LeaveTypeID.String(),
		StartDate:          lr.StartDate.Format("2006-01-02"),
		EndDate:            lr.EndDate.Format("2006-01-02"),
		IsHalfDayStart:     lr.IsHalfDayStart,
		IsHalfDayEnd:       lr.IsHalfDayEnd,
		TotalDays:          lr.TotalDays.String(),
		Reason:             lr.Reason,
		Status:             leaveStatusToProto(lr.Status),
		ApprovalComment:    lr.ApprovalComment,
		AuDocumentRequired: lr.AUDocumentRequired,
		CreatedAt:          timestamppb.New(lr.CreatedAt),
		UpdatedAt:          timestamppb.New(lr.UpdatedAt),
		EmployeeName:       lr.EmployeeName,
		LeaveTypeName:      lr.LeaveTypeName,
		LeaveTypeColor:     lr.LeaveTypeColor,
	}
	if lr.HalfDayPeriodStart != nil {
		plr.HalfDayPeriodStart = halfDayPeriodToProto(*lr.HalfDayPeriodStart)
	}
	if lr.HalfDayPeriodEnd != nil {
		plr.HalfDayPeriodEnd = halfDayPeriodToProto(*lr.HalfDayPeriodEnd)
	}
	if lr.ApprovedBy != nil {
		plr.ApprovedBy = lr.ApprovedBy.String()
	}
	if lr.ApprovedAt != nil {
		plr.ApprovedAt = timestamppb.New(*lr.ApprovedAt)
	}
	if lr.AUDocumentFileID != nil {
		plr.AuDocumentFileId = lr.AUDocumentFileID.String()
	}
	return plr
}

func toProtoLeaveBalance(b *models.HRLeaveBalance) *hrv1.LeaveBalance {
	if b == nil {
		return nil
	}
	pb := &hrv1.LeaveBalance{
		Id:          b.ID.String(),
		TenantId:    b.TenantID.String(),
		EmployeeId:  b.EmployeeID.String(),
		Year:        int32(b.Year),
		Entitlement: b.Entitlement.String(),
		CarriedOver: b.CarriedOver.String(),
		Used:        b.Used.String(),
		Remaining:   b.Remaining.String(),
		CreatedAt:   timestamppb.New(b.CreatedAt),
		UpdatedAt:   timestamppb.New(b.UpdatedAt),
	}
	if b.CarryoverExpiresAt != nil {
		pb.CarryoverExpiresAt = b.CarryoverExpiresAt.Format("2006-01-02")
	}
	pb.CarryoverNotified = b.CarryoverNotified
	return pb
}

func toProtoLeaveType(lt *models.LeaveType) *hrv1.LeaveType {
	if lt == nil {
		return nil
	}
	return &hrv1.LeaveType{
		Id:                 lt.ID.String(),
		TenantId:           lt.TenantID.String(),
		Name:               lt.Name,
		Key:                lt.Key,
		Color:              lt.Color,
		DeductsFromBalance: lt.DeductsFromBalance,
		RequiresApproval:   lt.RequiresApproval,
		RequiresAuDocument: lt.RequiresAUDocument,
		IsSystem:           lt.IsSystem,
		SortOrder:          int32(lt.SortOrder),
		CreatedAt:          timestamppb.New(lt.CreatedAt),
	}
}

func toProtoWorkTimeEntry(e *models.HRWorkTimeEntry) *hrv1.WorkTimeEntry {
	if e == nil {
		return nil
	}
	pe := &hrv1.WorkTimeEntry{
		Id:               e.ID.String(),
		TenantId:         e.TenantID.String(),
		EmployeeId:       e.EmployeeID.String(),
		ClockIn:          timestamppb.New(e.ClockIn),
		BreakMinutes:     int32(e.BreakMinutes),
		AutoBreakDeducted: int32(e.AutoBreakDeducted),
		Status:           workTimeStatusToProto(e.Status),
		IsCorrection:     e.IsCorrection,
		CorrectionReason: e.CorrectionReason,
		CreatedAt:        timestamppb.New(e.CreatedAt),
		UpdatedAt:        timestamppb.New(e.UpdatedAt),
		EmployeeName:     e.EmployeeName,
	}
	if e.ClockOut != nil {
		pe.ClockOut = timestamppb.New(*e.ClockOut)
	}
	if e.NetWorkMinutes != nil {
		pe.NetWorkMinutes = int32(*e.NetWorkMinutes)
	}
	if e.OriginalEntryID != nil {
		pe.OriginalEntryId = e.OriginalEntryID.String()
	}
	if e.CorrectionApprovedBy != nil {
		pe.CorrectionApprovedBy = e.CorrectionApprovedBy.String()
	}
	if e.CorrectionApprovedAt != nil {
		pe.CorrectionApprovedAt = timestamppb.New(*e.CorrectionApprovedAt)
	}
	// Nested breaks
	for _, b := range e.Breaks {
		pe.Breaks = append(pe.Breaks, toProtoBreakEntry(&b))
	}
	return pe
}

func toProtoBreakEntry(b *models.HRBreakEntry) *hrv1.BreakEntry {
	if b == nil {
		return nil
	}
	pb := &hrv1.BreakEntry{
		Id:              b.ID.String(),
		WorkTimeEntryId: b.WorkTimeEntryID.String(),
		StartTime:       timestamppb.New(b.StartTime),
	}
	if b.EndTime != nil {
		pb.EndTime = timestamppb.New(*b.EndTime)
	}
	if b.DurationMinutes != nil {
		pb.DurationMinutes = int32(*b.DurationMinutes)
	}
	return pb
}

func toProtoEmployeeProfile(e *models.EmployeeProfile) *hrv1.EmployeeProfile {
	if e == nil {
		return nil
	}
	pe := &hrv1.EmployeeProfile{
		Id:                    e.ID.String(),
		UserId:                e.UserID.String(),
		Department:            e.Department,
		PositionTitle:         e.PositionTitle,
		ContractType:          contractTypeToProto(e.ContractType),
		WorkDaysPerWeek:       int32(e.WorkDaysPerWeek),
		AnnualLeaveDays:       int32(e.AnnualLeaveDays),
		StartDate:             e.StartDate.Format("2006-01-02"),
		EmergencyContactName:  e.EmergencyContactName,
		EmergencyContactPhone: e.EmergencyContactPhone,
		AddressStreet:         e.AddressStreet,
		AddressCity:           e.AddressCity,
		AddressPostalCode:     e.AddressPostalCode,
		AddressCountry:        e.AddressCountry,
		CreatedAt:             timestamppb.New(e.CreatedAt),
		UpdatedAt:             timestamppb.New(e.UpdatedAt),
		UserName:              e.UserName,
		UserEmail:             e.UserEmail,
		ManagerName:           e.ManagerName,
	}
	if e.ManagerUserID != nil {
		pe.ManagerUserId = e.ManagerUserID.String()
	}
	return pe
}

func toProtoEmployeeDocument(d *models.EmployeeDocument) *hrv1.EmployeeDocument {
	if d == nil {
		return nil
	}
	return &hrv1.EmployeeDocument{
		Id:             d.ID.String(),
		TenantId:       d.TenantID.String(),
		EmployeeId:     d.EmployeeID.String(),
		CategoryId:     d.CategoryID.String(),
		FileId:         d.FileID.String(),
		UploadedBy:     d.UploadedBy.String(),
		Notes:          d.Notes,
		CreatedAt:      timestamppb.New(d.CreatedAt),
		CategoryName:   d.CategoryName,
		FileName:       d.FileName,
		FileSize:       d.FileSize,
		UploadedByName: d.UploadedByName,
	}
}

func toProtoHRSettings(s *models.HRCompanySettings) *hrv1.HRSettings {
	if s == nil {
		return nil
	}
	return &hrv1.HRSettings{
		Id:                     s.ID.String(),
		TenantId:               s.TenantID.String(),
		AuThresholdDays:        int32(s.AUThresholdDays),
		ShowAbsenceReason:      s.ShowAbsenceReason,
		DefaultAnnualLeaveDays: int32(s.DefaultAnnualLeaveDays),
		Timezone:               s.Timezone,
		CreatedAt:              timestamppb.New(s.CreatedAt),
		UpdatedAt:              timestamppb.New(s.UpdatedAt),
	}
}

func toProtoDailySummary(s *timetracking.DailySummary) *hrv1.DailySummary {
	if s == nil {
		return nil
	}
	return &hrv1.DailySummary{
		Date:             s.Date.Format("2006-01-02"),
		TotalWorkMinutes: int32(s.TotalWorkedMinutes),
		TotalBreakMinutes: int32(s.TotalBreakMinutes),
		NetWorkMinutes:   int32(s.NetWorkMinutes),
		OvertimeMinutes:  int32(s.OvertimeMinutes),
	}
}

func toProtoWeeklySummary(s *timetracking.WeeklySummary) *hrv1.WeeklySummary {
	if s == nil {
		return nil
	}
	ws := &hrv1.WeeklySummary{
		WeekStart:         s.WeekStart.Format("2006-01-02"),
		WeekEnd:           s.WeekStart.AddDate(0, 0, 6).Format("2006-01-02"),
		TotalWorkMinutes:  int32(s.TotalWorkedMinutes),
		TotalBreakMinutes: int32(s.TotalBreakMinutes),
		NetWorkMinutes:    int32(s.NetWorkMinutes),
		OvertimeMinutes:   int32(s.TotalOvertimeMinutes),
	}
	workDays := 0
	for _, d := range s.Days {
		ws.DailySummaries = append(ws.DailySummaries, toProtoDailySummary(&d))
		if d.TotalWorkedMinutes > 0 {
			workDays++
		}
	}
	ws.WorkDays = int32(workDays)
	return ws
}

// ============================================================================
// Enum Conversion Helpers
// ============================================================================

func leaveStatusToProto(s models.LeaveRequestStatus) hrv1.LeaveRequestStatus {
	switch s {
	case models.LeaveStatusPending:
		return hrv1.LeaveRequestStatus_LEAVE_PENDING
	case models.LeaveStatusApproved:
		return hrv1.LeaveRequestStatus_LEAVE_APPROVED
	case models.LeaveStatusRejected:
		return hrv1.LeaveRequestStatus_LEAVE_REJECTED
	case models.LeaveStatusCancelled:
		return hrv1.LeaveRequestStatus_LEAVE_CANCELLED
	default:
		return hrv1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_UNSPECIFIED
	}
}

func workTimeStatusToProto(s models.WorkTimeEntryStatus) hrv1.WorkTimeEntryStatus {
	switch s {
	case models.WorkTimeStatusActive:
		return hrv1.WorkTimeEntryStatus_WORK_TIME_ACTIVE
	case models.WorkTimeStatusCompleted:
		return hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED
	case models.WorkTimeStatusCorrectionPending:
		return hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_PENDING
	case models.WorkTimeStatusCorrectionApproved:
		return hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_APPROVED
	default:
		return hrv1.WorkTimeEntryStatus_WORK_TIME_ENTRY_STATUS_UNSPECIFIED
	}
}

func contractTypeToProto(ct models.HRContractType) hrv1.ContractType {
	switch ct {
	case models.HRContractFullTime:
		return hrv1.ContractType_CONTRACT_FULL_TIME
	case models.HRContractPartTime:
		return hrv1.ContractType_CONTRACT_PART_TIME
	case models.HRContractMiniJob:
		return hrv1.ContractType_CONTRACT_MINI_JOB
	case models.HRContractIntern:
		return hrv1.ContractType_CONTRACT_INTERN
	case models.HRContractTemporary:
		return hrv1.ContractType_CONTRACT_TEMPORARY
	default:
		return hrv1.ContractType_CONTRACT_TYPE_UNSPECIFIED
	}
}

func contractTypeFromProto(ct hrv1.ContractType) models.HRContractType {
	switch ct {
	case hrv1.ContractType_CONTRACT_FULL_TIME:
		return models.HRContractFullTime
	case hrv1.ContractType_CONTRACT_PART_TIME:
		return models.HRContractPartTime
	case hrv1.ContractType_CONTRACT_MINI_JOB:
		return models.HRContractMiniJob
	case hrv1.ContractType_CONTRACT_INTERN:
		return models.HRContractIntern
	case hrv1.ContractType_CONTRACT_TEMPORARY:
		return models.HRContractTemporary
	default:
		return models.HRContractFullTime
	}
}

func halfDayPeriodToProto(p models.HalfDayPeriod) hrv1.HalfDayPeriod {
	switch p {
	case models.HalfDayMorning:
		return hrv1.HalfDayPeriod_HALF_DAY_MORNING
	case models.HalfDayAfternoon:
		return hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON
	default:
		return hrv1.HalfDayPeriod_HALF_DAY_PERIOD_UNSPECIFIED
	}
}

func halfDayPeriodFromProto(p hrv1.HalfDayPeriod) models.HalfDayPeriod {
	switch p {
	case hrv1.HalfDayPeriod_HALF_DAY_MORNING:
		return models.HalfDayMorning
	case hrv1.HalfDayPeriod_HALF_DAY_AFTERNOON:
		return models.HalfDayAfternoon
	default:
		return models.HalfDayMorning
	}
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapHRError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// Leave errors
	case errors.Is(err, leave.ErrLeaveRequestNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, leave.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, leave.ErrInsufficientBalance):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, leave.ErrCannotApproveOwnRequest):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, leave.ErrNotAuthorizedToApprove):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, leave.ErrInvalidDateRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, leave.ErrCannotCancelPastLeave):
		return status.Error(codes.FailedPrecondition, err.Error())

	// Time tracking errors
	case errors.Is(err, timetracking.ErrAlreadyClockedIn):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timetracking.ErrNotClockedIn):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timetracking.ErrAlreadyOnBreak):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timetracking.ErrNotOnBreak):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timetracking.ErrMaxDailyHoursExceeded):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timetracking.ErrCorrectionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, timetracking.ErrWorkTimeEntryNotFound):
		return status.Error(codes.NotFound, err.Error())

	// Employee errors
	case errors.Is(err, employee.ErrEmployeeNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, employee.ErrProfileAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, employee.ErrUnauthorizedFieldUpdate):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, employee.ErrDocumentCategoryNotFound):
		return status.Error(codes.NotFound, err.Error())

	// Absence errors
	case errors.Is(err, absence.ErrInvalidDateRange):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		slog.Error("unhandled HR service error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}

// Suppress unused import warnings for decimal
var _ = decimal.Zero
