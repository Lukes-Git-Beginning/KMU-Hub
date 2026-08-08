package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/produktion"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
)

// mapProduktionExtError maps the extended produktion errors (BOM, Machine, WorkStep, QualityCheck)
// and falls back to mapProduktionError for the base set.
func mapProduktionExtError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, produktion.ErrBOMNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrBOMSKUTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, produktion.ErrWorkStepNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrMachineNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrQualityCheckNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return mapProduktionError(err)
	}
}

// Ensure slog is used (it's used in the helpers below via slog.Error in mapProduktionError).
var _ = slog.Error

// ============================================================================
// BOM RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateBOM(ctx context.Context, req *produktionv1.CreateBOMRequest) (*produktionv1.BOMResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.CreateBOMInput{
		TenantID:    tenantID,
		ProductName: req.GetProductName(),
		SKU:         req.GetSku(),
		Version:     req.GetVersion(),
		Active:      req.GetActive(),
		Notes:       req.GetNotes(),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}
	for _, item := range req.GetItems() {
		input.Items = append(input.Items, produktion.CreateBomItemInput{
			MaterialName: item.GetMaterialName(),
			Quantity:     item.GetQuantity(),
			Unit:         item.GetUnit(),
			SortOrder:    int(item.GetSortOrder()),
		})
	}

	bom, err := s.svc.CreateBOM(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.BOMResponse{Bom: bomToProto(bom)}, nil
}

func (s *ProduktionGRPCServer) UpdateBOM(ctx context.Context, req *produktionv1.UpdateBOMRequest) (*produktionv1.BOMResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	bomID, err := uuid.Parse(req.GetBomId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bom_id: %v", err)
	}

	input := produktion.UpdateBOMInput{
		TenantID:    tenantID,
		BOMID:       bomID,
		ProductName: req.ProductName,
		SKU:         req.Sku,
		Version:     req.Version,
		Active:      req.Active,
		Notes:       req.Notes,
	}

	bom, err := s.svc.UpdateBOM(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.BOMResponse{Bom: bomToProto(bom)}, nil
}

func (s *ProduktionGRPCServer) DeleteBOM(ctx context.Context, req *produktionv1.DeleteBOMRequest) (*produktionv1.DeleteBOMResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	bomID, err := uuid.Parse(req.GetBomId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bom_id: %v", err)
	}

	if err := s.svc.DeleteBOM(ctx, tenantID, bomID); err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.DeleteBOMResponse{}, nil
}

func (s *ProduktionGRPCServer) GetBOM(ctx context.Context, req *produktionv1.GetBOMRequest) (*produktionv1.BOMResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	bomID, err := uuid.Parse(req.GetBomId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bom_id: %v", err)
	}

	bom, err := s.svc.GetBOM(ctx, tenantID, bomID)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.BOMResponse{Bom: bomToProto(bom)}, nil
}

func (s *ProduktionGRPCServer) ListBOMs(ctx context.Context, req *produktionv1.ListBOMsRequest) (*produktionv1.ListBOMsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.ListBOMsInput{
		TenantID:   tenantID,
		Page:       int(req.GetPage()),
		PageSize:   int(req.GetPageSize()),
		ActiveOnly: req.Active,
	}

	boms, total, err := s.svc.ListBOMs(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}

	protoBOMs := make([]*produktionv1.BOM, len(boms))
	for i, b := range boms {
		protoBOMs[i] = bomToProto(b)
	}
	return &produktionv1.ListBOMsResponse{Boms: protoBOMs, Total: int32(total)}, nil
}

// ============================================================================
// Work Step RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateWorkStep(ctx context.Context, req *produktionv1.CreateWorkStepRequest) (*produktionv1.WorkStepResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	step, err := s.svc.CreateWorkStep(ctx, produktion.CreateWorkStepInput{
		TenantID:        tenantID,
		OrderID:         orderID,
		StepNr:          int(req.GetStepNr()),
		Name:            req.GetName(),
		Description:     req.GetDescription(),
		DurationMinutes: int(req.GetDurationMinutes()),
		Assignee:        req.GetAssignee(),
	})
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.WorkStepResponse{Step: workStepToProto(step)}, nil
}

func (s *ProduktionGRPCServer) UpdateWorkStep(ctx context.Context, req *produktionv1.UpdateWorkStepRequest) (*produktionv1.WorkStepResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	stepID, err := uuid.Parse(req.GetStepId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid step_id: %v", err)
	}

	input := produktion.UpdateWorkStepInput{
		TenantID:    tenantID,
		StepID:      stepID,
		Name:        req.Name,
		Description: req.Description,
		Assignee:    req.Assignee,
	}
	if req.DurationMinutes != nil {
		d := int(*req.DurationMinutes)
		input.DurationMinutes = &d
	}
	if req.Status != nil {
		st := produktion.WorkStepStatus(*req.Status)
		input.Status = &st
	}

	step, err := s.svc.UpdateWorkStep(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.WorkStepResponse{Step: workStepToProto(step)}, nil
}

func (s *ProduktionGRPCServer) DeleteWorkStep(ctx context.Context, req *produktionv1.DeleteWorkStepRequest) (*produktionv1.DeleteWorkStepResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	stepID, err := uuid.Parse(req.GetStepId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid step_id: %v", err)
	}

	if err := s.svc.DeleteWorkStep(ctx, tenantID, stepID); err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.DeleteWorkStepResponse{}, nil
}

func (s *ProduktionGRPCServer) ListWorkSteps(ctx context.Context, req *produktionv1.ListWorkStepsRequest) (*produktionv1.ListWorkStepsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	steps, err := s.svc.ListWorkSteps(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}

	protoSteps := make([]*produktionv1.WorkStep, len(steps))
	for i, st := range steps {
		protoSteps[i] = workStepToProto(st)
	}
	return &produktionv1.ListWorkStepsResponse{Steps: protoSteps, Total: int32(len(steps))}, nil
}

// ============================================================================
// Machine RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateMachine(ctx context.Context, req *produktionv1.CreateMachineRequest) (*produktionv1.MachineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.CreateMachineInput{
		TenantID: tenantID,
		Name:     req.GetName(),
		Type:     req.GetType(),
		Notes:    req.GetNotes(),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	machine, err := s.svc.CreateMachine(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.MachineResponse{Machine: machineToProto(machine)}, nil
}

func (s *ProduktionGRPCServer) UpdateMachine(ctx context.Context, req *produktionv1.UpdateMachineRequest) (*produktionv1.MachineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	machineID, err := uuid.Parse(req.GetMachineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid machine_id: %v", err)
	}

	input := produktion.UpdateMachineInput{
		TenantID:  tenantID,
		MachineID: machineID,
		Name:      req.Name,
		Type:      req.Type,
		Notes:     req.Notes,
	}
	if req.Status != nil {
		st := produktion.MachineStatus(*req.Status)
		input.Status = &st
	}

	machine, err := s.svc.UpdateMachine(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.MachineResponse{Machine: machineToProto(machine)}, nil
}

func (s *ProduktionGRPCServer) DeleteMachine(ctx context.Context, req *produktionv1.DeleteMachineRequest) (*produktionv1.DeleteMachineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	machineID, err := uuid.Parse(req.GetMachineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid machine_id: %v", err)
	}

	if err := s.svc.DeleteMachine(ctx, tenantID, machineID); err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.DeleteMachineResponse{}, nil
}

func (s *ProduktionGRPCServer) GetMachine(ctx context.Context, req *produktionv1.GetMachineRequest) (*produktionv1.MachineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	machineID, err := uuid.Parse(req.GetMachineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid machine_id: %v", err)
	}

	machine, err := s.svc.GetMachine(ctx, tenantID, machineID)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.MachineResponse{Machine: machineToProto(machine)}, nil
}

func (s *ProduktionGRPCServer) ListMachines(ctx context.Context, req *produktionv1.ListMachinesRequest) (*produktionv1.ListMachinesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.ListMachinesInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		st := produktion.MachineStatus(*req.Status)
		input.Status = &st
	}

	machines, total, err := s.svc.ListMachines(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}

	protoMachines := make([]*produktionv1.Machine, len(machines))
	for i, m := range machines {
		protoMachines[i] = machineToProto(m)
	}
	return &produktionv1.ListMachinesResponse{Machines: protoMachines, Total: int32(total)}, nil
}

// ============================================================================
// Quality Check RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateQualityCheck(ctx context.Context, req *produktionv1.CreateQualityCheckRequest) (*produktionv1.QualityCheckResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	input := produktion.CreateQualityCheckInput{
		TenantID:     tenantID,
		OrderID:      orderID,
		Inspector:    req.GetInspector(),
		CheckedAt:    req.GetCheckedAt().AsTime(),
		Passed:       req.GetPassed(),
		DefectsFound: int(req.GetDefectsFound()),
		Notes:        req.GetNotes(),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	check, err := s.svc.CreateQualityCheck(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.QualityCheckResponse{Check: qualityCheckToProto(check)}, nil
}

func (s *ProduktionGRPCServer) GetQualityCheck(ctx context.Context, req *produktionv1.GetQualityCheckRequest) (*produktionv1.QualityCheckResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	checkID, err := uuid.Parse(req.GetCheckId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid check_id: %v", err)
	}

	check, err := s.svc.GetQualityCheck(ctx, tenantID, checkID)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return &produktionv1.QualityCheckResponse{Check: qualityCheckToProto(check)}, nil
}

func (s *ProduktionGRPCServer) ListQualityChecks(ctx context.Context, req *produktionv1.ListQualityChecksRequest) (*produktionv1.ListQualityChecksResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.ListQualityChecksInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.OrderId != nil {
		id, parseErr := uuid.Parse(*req.OrderId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", parseErr)
		}
		input.OrderID = &id
	}

	checks, total, err := s.svc.ListQualityChecks(ctx, input)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}

	protoChecks := make([]*produktionv1.QualityCheck, len(checks))
	for i, c := range checks {
		protoChecks[i] = qualityCheckToProto(c)
	}
	return &produktionv1.ListQualityChecksResponse{Checks: protoChecks, Total: int32(total)}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func bomToProto(b *produktion.BOM) *produktionv1.BOM {
	if b == nil {
		return nil
	}
	p := &produktionv1.BOM{
		Id:          b.ID.String(),
		TenantId:    b.TenantID.String(),
		ProductName: b.ProductName,
		Sku:         b.SKU,
		Version:     b.Version,
		Active:      b.Active,
		Notes:       b.Notes,
		CreatedAt:   timestamppb.New(b.CreatedAt),
		UpdatedAt:   timestamppb.New(b.UpdatedAt),
	}
	if b.CreatedBy != nil {
		s := b.CreatedBy.String()
		p.CreatedBy = &s
	}
	for _, item := range b.Items {
		p.Items = append(p.Items, &produktionv1.BomItem{
			Id:           item.ID.String(),
			TenantId:     item.TenantID.String(),
			BomId:        item.BomID.String(),
			MaterialName: item.MaterialName,
			Quantity:     item.Quantity,
			Unit:         item.Unit,
			SortOrder:    int32(item.SortOrder),
		})
	}
	return p
}

func workStepToProto(st *produktion.WorkStep) *produktionv1.WorkStep {
	if st == nil {
		return nil
	}
	return &produktionv1.WorkStep{
		Id:              st.ID.String(),
		TenantId:        st.TenantID.String(),
		OrderId:         st.OrderID.String(),
		StepNr:          int32(st.StepNr),
		Name:            st.Name,
		Description:     st.Description,
		DurationMinutes: int32(st.DurationMinutes),
		Status:          string(st.Status),
		Assignee:        st.Assignee,
		CreatedAt:       timestamppb.New(st.CreatedAt),
		UpdatedAt:       timestamppb.New(st.UpdatedAt),
	}
}

func machineToProto(m *produktion.Machine) *produktionv1.Machine {
	if m == nil {
		return nil
	}
	p := &produktionv1.Machine{
		Id:        m.ID.String(),
		TenantId:  m.TenantID.String(),
		Name:      m.Name,
		Type:      m.Type,
		Status:    string(m.Status),
		Notes:     m.Notes,
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
	}
	if m.CreatedBy != nil {
		s := m.CreatedBy.String()
		p.CreatedBy = &s
	}
	return p
}

func qualityCheckToProto(c *produktion.QualityCheck) *produktionv1.QualityCheck {
	if c == nil {
		return nil
	}
	p := &produktionv1.QualityCheck{
		Id:           c.ID.String(),
		TenantId:     c.TenantID.String(),
		OrderId:      c.OrderID.String(),
		Inspector:    c.Inspector,
		CheckedAt:    timestamppb.New(c.CheckedAt),
		Passed:       c.Passed,
		DefectsFound: int32(c.DefectsFound),
		Notes:        c.Notes,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
	}
	if c.CreatedBy != nil {
		s := c.CreatedBy.String()
		p.CreatedBy = &s
	}
	return p
}

// ============================================================================
// Material Availability RPC
// ============================================================================

func (s *ProduktionGRPCServer) GetMaterialAvailability(ctx context.Context, req *produktionv1.GetMaterialAvailabilityRequest) (*produktionv1.MaterialAvailabilityResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	availability, err := s.svc.GetMaterialAvailability(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionExtError(err)
	}
	return materialAvailabilityToProto(availability), nil
}

func materialAvailabilityToProto(a *produktion.MaterialAvailability) *produktionv1.MaterialAvailabilityResponse {
	if a == nil {
		return nil
	}
	resp := &produktionv1.MaterialAvailabilityResponse{
		OrderId: a.OrderID.String(),
		BomId:   a.BomID.String(),
		Lines:   make([]*produktionv1.MaterialAvailabilityLine, 0, len(a.Lines)),
	}
	for _, line := range a.Lines {
		p := &produktionv1.MaterialAvailabilityLine{
			MaterialName:     line.MaterialName,
			Unit:             line.Unit,
			RequiredQuantity: line.RequiredQuantity,
		}
		if line.AvailableQuantity != nil {
			p.AvailableQuantity = line.AvailableQuantity
		}
		if line.ShortfallQuantity != nil {
			p.ShortfallQuantity = line.ShortfallQuantity
		}
		resp.Lines = append(resp.Lines, p)
	}
	return resp
}
