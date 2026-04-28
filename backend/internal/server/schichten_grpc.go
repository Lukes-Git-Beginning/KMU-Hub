package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/schichten"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

// SchichtenGRPCServer implements the SchichtenService gRPC server.
type SchichtenGRPCServer struct {
	schichtenv1.UnimplementedSchichtenServiceServer
	svc *schichten.Service
}

// NewSchichtenGRPCServer creates a new Schichten gRPC server.
func NewSchichtenGRPCServer(svc *schichten.Service) *SchichtenGRPCServer {
	return &SchichtenGRPCServer{svc: svc}
}

// ============================================================================
// Shift RPCs
// ============================================================================

func (s *SchichtenGRPCServer) CreateShift(ctx context.Context, req *schichtenv1.CreateShiftRequest) (*schichtenv1.ShiftResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := schichten.CreateShiftInput{
		TenantID:    tenantID,
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		StartTime:   req.GetStartTime().AsTime(),
		EndTime:     req.GetEndTime().AsTime(),
	}
	if loc := req.GetLocation(); loc != "" {
		input.Location = &loc
	}
	if cb := req.GetCreatedBy(); cb != "" {
		id, parseErr := uuid.Parse(cb)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	shift, err := s.svc.CreateShift(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.ShiftResponse{Shift: shiftToProto(shift)}, nil
}

func (s *SchichtenGRPCServer) GetShift(ctx context.Context, req *schichtenv1.GetShiftRequest) (*schichtenv1.ShiftResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}

	shift, err := s.svc.GetShift(ctx, tenantID, shiftID)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.ShiftResponse{Shift: shiftToProto(shift)}, nil
}

func (s *SchichtenGRPCServer) UpdateShift(ctx context.Context, req *schichtenv1.UpdateShiftRequest) (*schichtenv1.ShiftResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}

	input := schichten.UpdateShiftInput{
		TenantID: tenantID,
		ShiftID:  shiftID,
	}
	if t := req.GetTitle(); t != "" {
		input.Title = &t
	}
	if d := req.GetDescription(); d != "" {
		input.Description = &d
	}
	if req.StartTime != nil {
		t := req.GetStartTime().AsTime()
		input.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.GetEndTime().AsTime()
		input.EndTime = &t
	}
	if l := req.GetLocation(); l != "" {
		input.Location = &l
	}

	shift, err := s.svc.UpdateShift(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.ShiftResponse{Shift: shiftToProto(shift)}, nil
}

func (s *SchichtenGRPCServer) DeleteShift(ctx context.Context, req *schichtenv1.DeleteShiftRequest) (*schichtenv1.DeleteShiftResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}

	if err := s.svc.DeleteShift(ctx, tenantID, shiftID); err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.DeleteShiftResponse{}, nil
}

func (s *SchichtenGRPCServer) ListShifts(ctx context.Context, req *schichtenv1.ListShiftsRequest) (*schichtenv1.ListShiftsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := schichten.ListShiftsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if st := req.GetStatus(); st != "" {
		ss := schichten.ShiftStatus(st)
		input.Status = &ss
	}
	if req.From != nil {
		t := req.GetFrom().AsTime()
		input.From = &t
	}
	if req.To != nil {
		t := req.GetTo().AsTime()
		input.To = &t
	}

	shifts, total, err := s.svc.ListShifts(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}

	var pbShifts []*schichtenv1.Shift
	for _, sh := range shifts {
		pbShifts = append(pbShifts, shiftToProto(sh))
	}
	return &schichtenv1.ListShiftsResponse{Shifts: pbShifts, Total: int32(total)}, nil
}

func (s *SchichtenGRPCServer) PublishShifts(ctx context.Context, req *schichtenv1.PublishShiftsRequest) (*schichtenv1.PublishShiftsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	count, err := s.svc.PublishShifts(ctx, tenantID, req.GetFrom().AsTime(), req.GetTo().AsTime())
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.PublishShiftsResponse{PublishedCount: int32(count)}, nil
}

// ============================================================================
// Assignment RPCs
// ============================================================================

func (s *SchichtenGRPCServer) AssignEmployee(ctx context.Context, req *schichtenv1.AssignEmployeeRequest) (*schichtenv1.AssignmentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}
	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid employee_id: %v", err)
	}

	input := schichten.AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shiftID,
		EmployeeID: employeeID,
	}
	if ab := req.GetAssignedBy(); ab != "" {
		id, parseErr := uuid.Parse(ab)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assigned_by: %v", parseErr)
		}
		input.AssignedBy = &id
	}

	assignment, err := s.svc.AssignEmployee(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.AssignmentResponse{Assignment: assignmentToProto(assignment)}, nil
}

func (s *SchichtenGRPCServer) UnassignEmployee(ctx context.Context, req *schichtenv1.UnassignEmployeeRequest) (*schichtenv1.UnassignEmployeeResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}
	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid employee_id: %v", err)
	}

	if err := s.svc.UnassignEmployee(ctx, tenantID, shiftID, employeeID); err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.UnassignEmployeeResponse{}, nil
}

func (s *SchichtenGRPCServer) ListAssignments(ctx context.Context, req *schichtenv1.ListAssignmentsRequest) (*schichtenv1.ListAssignmentsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shiftID, err := uuid.Parse(req.GetShiftId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid shift_id: %v", err)
	}

	assignments, err := s.svc.ListAssignments(ctx, tenantID, shiftID)
	if err != nil {
		return nil, mapSchichtenError(err)
	}

	var pbAssignments []*schichtenv1.ShiftAssignment
	for _, a := range assignments {
		pbAssignments = append(pbAssignments, assignmentToProto(a))
	}
	return &schichtenv1.ListAssignmentsResponse{Assignments: pbAssignments}, nil
}

// ============================================================================
// Template RPCs
// ============================================================================

func (s *SchichtenGRPCServer) CreateTemplate(ctx context.Context, req *schichtenv1.CreateTemplateRequest) (*schichtenv1.TemplateResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	tmpl, err := s.svc.CreateTemplate(ctx, schichten.CreateTemplateInput{
		TenantID:        tenantID,
		Name:            req.GetName(),
		Description:     req.GetDescription(),
		DayOfWeek:       int(req.GetDayOfWeek()),
		StartHour:       int(req.GetStartHour()),
		StartMinute:     int(req.GetStartMinute()),
		DurationMinutes: int(req.GetDurationMinutes()),
	})
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.TemplateResponse{Template: shiftTemplateToProto(tmpl)}, nil
}

func (s *SchichtenGRPCServer) UpdateTemplate(ctx context.Context, req *schichtenv1.UpdateTemplateRequest) (*schichtenv1.TemplateResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	input := schichten.UpdateTemplateInput{
		TenantID:   tenantID,
		TemplateID: templateID,
	}
	if n := req.GetName(); n != "" {
		input.Name = &n
	}
	if d := req.GetDescription(); d != "" {
		input.Description = &d
	}
	if req.DayOfWeek != nil {
		v := int(req.GetDayOfWeek())
		input.DayOfWeek = &v
	}
	if req.StartHour != nil {
		v := int(req.GetStartHour())
		input.StartHour = &v
	}
	if req.StartMinute != nil {
		v := int(req.GetStartMinute())
		input.StartMinute = &v
	}
	if req.DurationMinutes != nil {
		v := int(req.GetDurationMinutes())
		input.DurationMinutes = &v
	}
	if l := req.GetLocation(); l != "" {
		input.Location = &l
	}

	tmpl, err := s.svc.UpdateTemplate(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.TemplateResponse{Template: shiftTemplateToProto(tmpl)}, nil
}

func (s *SchichtenGRPCServer) DeleteTemplate(ctx context.Context, req *schichtenv1.DeleteTemplateRequest) (*schichtenv1.DeleteTemplateResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	if err := s.svc.DeleteTemplate(ctx, tenantID, templateID); err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.DeleteTemplateResponse{}, nil
}

func (s *SchichtenGRPCServer) ListTemplates(ctx context.Context, req *schichtenv1.ListTemplatesRequest) (*schichtenv1.ListTemplatesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	templates, total, err := s.svc.ListTemplates(ctx, schichten.ListTemplatesInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapSchichtenError(err)
	}

	var pbTemplates []*schichtenv1.ShiftTemplate
	for _, t := range templates {
		pbTemplates = append(pbTemplates, shiftTemplateToProto(t))
	}
	return &schichtenv1.ListTemplatesResponse{Templates: pbTemplates, Total: int32(total)}, nil
}

func (s *SchichtenGRPCServer) ApplyTemplate(ctx context.Context, req *schichtenv1.ApplyTemplateRequest) (*schichtenv1.ApplyTemplateResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	input := schichten.ApplyTemplateInput{
		TenantID:   tenantID,
		TemplateID: templateID,
		RangeStart: req.GetRangeStart().AsTime(),
		RangeEnd:   req.GetRangeEnd().AsTime(),
	}
	if cb := req.GetCreatedBy(); cb != "" {
		id, parseErr := uuid.Parse(cb)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	shifts, err := s.svc.ApplyTemplate(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}

	var pbShifts []*schichtenv1.Shift
	for _, sh := range shifts {
		pbShifts = append(pbShifts, shiftToProto(sh))
	}
	return &schichtenv1.ApplyTemplateResponse{Shifts: pbShifts, CreatedCount: int32(len(shifts))}, nil
}

// ============================================================================
// Compliance & Stats RPCs
// ============================================================================

func (s *SchichtenGRPCServer) CheckArbzgCompliance(ctx context.Context, req *schichtenv1.CheckArbzgComplianceRequest) (*schichtenv1.CheckArbzgComplianceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid employee_id: %v", err)
	}

	compliant, reason, err := s.svc.CheckArbzgCompliance(ctx, tenantID, employeeID,
		req.GetNewShiftStart().AsTime(), req.GetNewShiftEnd().AsTime())
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.CheckArbzgComplianceResponse{Compliant: compliant, ViolationReason: reason}, nil
}

func (s *SchichtenGRPCServer) GetShiftStats(ctx context.Context, req *schichtenv1.GetShiftStatsRequest) (*schichtenv1.ShiftStatsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := schichten.GetShiftStatsInput{TenantID: tenantID}
	if req.From != nil {
		t := req.GetFrom().AsTime()
		input.From = &t
	}
	if req.To != nil {
		t := req.GetTo().AsTime()
		input.To = &t
	}

	stats, err := s.svc.GetShiftStats(ctx, input)
	if err != nil {
		return nil, mapSchichtenError(err)
	}
	return &schichtenv1.ShiftStatsResponse{
		TotalShifts:      int32(stats.TotalShifts),
		PublishedShifts:  int32(stats.PublishedShifts),
		DraftShifts:      int32(stats.DraftShifts),
		TotalAssignments: int32(stats.TotalAssignments),
		UniqueEmployees:  int32(stats.UniqueEmployees),
	}, nil
}

// ============================================================================
// Mapping helpers
// ============================================================================

func shiftToProto(s *schichten.Shift) *schichtenv1.Shift {
	pb := &schichtenv1.Shift{
		TenantId:    s.TenantID.String(),
		Id:          s.ID.String(),
		Title:       s.Title,
		Description: s.Description,
		StartTime:   timestamppb.New(s.StartTime),
		EndTime:     timestamppb.New(s.EndTime),
		Status:      string(s.Status),
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
	if s.Location != nil {
		pb.Location = s.Location
	}
	if s.CreatedBy != nil {
		v := s.CreatedBy.String()
		pb.CreatedBy = &v
	}
	return pb
}

func assignmentToProto(a *schichten.ShiftAssignment) *schichtenv1.ShiftAssignment {
	pb := &schichtenv1.ShiftAssignment{
		TenantId:   a.TenantID.String(),
		Id:         a.ID.String(),
		ShiftId:    a.ShiftID.String(),
		EmployeeId: a.EmployeeID.String(),
		AssignedAt: timestamppb.New(a.AssignedAt),
	}
	if a.AssignedBy != nil {
		v := a.AssignedBy.String()
		pb.AssignedBy = &v
	}
	return pb
}

func shiftTemplateToProto(t *schichten.ShiftTemplate) *schichtenv1.ShiftTemplate {
	pb := &schichtenv1.ShiftTemplate{
		TenantId:        t.TenantID.String(),
		Id:              t.ID.String(),
		Name:            t.Name,
		Description:     t.Description,
		DayOfWeek:       int32(t.DayOfWeek),
		StartHour:       int32(t.StartHour),
		StartMinute:     int32(t.StartMinute),
		DurationMinutes: int32(t.DurationMinutes),
		CreatedAt:       timestamppb.New(t.CreatedAt),
		UpdatedAt:       timestamppb.New(t.UpdatedAt),
	}
	if t.Location != nil {
		pb.Location = t.Location
	}
	return pb
}

func mapSchichtenError(err error) error {
	switch {
	case errors.Is(err, schichten.ErrShiftNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, schichten.ErrTemplateNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, schichten.ErrAssignmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, schichten.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, schichten.ErrAlreadyAssigned):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, schichten.ErrArbzgViolation):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
