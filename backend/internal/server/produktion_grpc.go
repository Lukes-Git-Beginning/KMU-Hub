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

// ProduktionGRPCServer implements the ProduktionService gRPC server.
type ProduktionGRPCServer struct {
	produktionv1.UnimplementedProduktionServiceServer
	svc *produktion.Service
}

// NewProduktionGRPCServer creates a new Produktion gRPC server.
func NewProduktionGRPCServer(svc *produktion.Service) *ProduktionGRPCServer {
	return &ProduktionGRPCServer{svc: svc}
}

// ============================================================================
// Order RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateOrder(ctx context.Context, req *produktionv1.CreateOrderRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.CreateOrderInput{
		TenantID:     tenantID,
		OrderNumber:  req.GetOrderNumber(),
		ProductName:  req.GetProductName(),
		Quantity:     int(req.GetQuantity()),
		PlannedStart: req.GetPlannedStart().AsTime(),
		PlannedEnd:   req.GetPlannedEnd().AsTime(),
		Priority:     int(req.GetPriority()),
		Notes:        req.GetNotes(),
	}

	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	order, err := s.svc.CreateOrder(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

func (s *ProduktionGRPCServer) UpdateOrder(ctx context.Context, req *produktionv1.UpdateOrderRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	input := produktion.UpdateOrderInput{
		TenantID: tenantID,
		OrderID:  orderID,
	}

	if req.ProductName != nil {
		input.ProductName = req.ProductName
	}
	if req.Quantity != nil {
		q := int(*req.Quantity)
		input.Quantity = &q
	}
	if req.PlannedStart != nil {
		t := req.PlannedStart.AsTime()
		input.PlannedStart = &t
	}
	if req.PlannedEnd != nil {
		t := req.PlannedEnd.AsTime()
		input.PlannedEnd = &t
	}
	if req.Priority != nil {
		p := int(*req.Priority)
		input.Priority = &p
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	order, err := s.svc.UpdateOrder(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

func (s *ProduktionGRPCServer) DeleteOrder(ctx context.Context, req *produktionv1.DeleteOrderRequest) (*produktionv1.DeleteOrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	if err := s.svc.DeleteOrder(ctx, tenantID, orderID); err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.DeleteOrderResponse{}, nil
}

func (s *ProduktionGRPCServer) GetOrder(ctx context.Context, req *produktionv1.GetOrderRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	order, err := s.svc.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

func (s *ProduktionGRPCServer) ListOrders(ctx context.Context, req *produktionv1.ListOrdersRequest) (*produktionv1.ListOrdersResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.ListOrdersInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	if req.Status != nil {
		st := produktion.OrderStatus(*req.Status)
		input.Status = &st
	}
	if req.Priority != nil {
		p := int(*req.Priority)
		input.Priority = &p
	}
	if req.DateFrom != nil {
		t := req.DateFrom.AsTime()
		input.DateFrom = &t
	}
	if req.DateTo != nil {
		t := req.DateTo.AsTime()
		input.DateTo = &t
	}

	orders, total, err := s.svc.ListOrders(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}

	protoOrders := make([]*produktionv1.ProductionOrder, len(orders))
	for i, o := range orders {
		protoOrders[i] = orderToProto(o)
	}
	return &produktionv1.ListOrdersResponse{
		Orders: protoOrders,
		Total:  int32(total),
	}, nil
}

func (s *ProduktionGRPCServer) StartOrder(ctx context.Context, req *produktionv1.OrderActionRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	order, err := s.svc.StartOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

func (s *ProduktionGRPCServer) CompleteOrder(ctx context.Context, req *produktionv1.OrderActionRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	order, err := s.svc.CompleteOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

func (s *ProduktionGRPCServer) CancelOrder(ctx context.Context, req *produktionv1.OrderActionRequest) (*produktionv1.OrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	order, err := s.svc.CancelOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.OrderResponse{Order: orderToProto(order)}, nil
}

// ============================================================================
// Machine Booking RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreateMachineBooking(ctx context.Context, req *produktionv1.CreateMachineBookingRequest) (*produktionv1.MachineBookingResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	orderID, err := uuid.Parse(req.GetProductionOrderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid production_order_id: %v", err)
	}

	input := produktion.CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         req.GetMachineId(),
		ProductionOrderID: orderID,
		StartsAt:          req.GetStartsAt().AsTime(),
		EndsAt:            req.GetEndsAt().AsTime(),
		Notes:             req.GetNotes(),
	}

	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	booking, err := s.svc.CreateMachineBooking(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.MachineBookingResponse{Booking: machineBookingToProto(booking)}, nil
}

func (s *ProduktionGRPCServer) UpdateMachineBooking(ctx context.Context, req *produktionv1.UpdateMachineBookingRequest) (*produktionv1.MachineBookingResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	bookingID, err := uuid.Parse(req.GetBookingId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid booking_id: %v", err)
	}

	input := produktion.UpdateBookingInput{
		TenantID:  tenantID,
		BookingID: bookingID,
	}

	if req.MachineId != nil {
		input.MachineID = req.MachineId
	}
	if req.ProductionOrderId != nil {
		id, parseErr := uuid.Parse(*req.ProductionOrderId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid production_order_id: %v", parseErr)
		}
		input.ProductionOrderID = &id
	}
	if req.StartsAt != nil {
		t := req.StartsAt.AsTime()
		input.StartsAt = &t
	}
	if req.EndsAt != nil {
		t := req.EndsAt.AsTime()
		input.EndsAt = &t
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	booking, err := s.svc.UpdateMachineBooking(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.MachineBookingResponse{Booking: machineBookingToProto(booking)}, nil
}

func (s *ProduktionGRPCServer) DeleteMachineBooking(ctx context.Context, req *produktionv1.DeleteMachineBookingRequest) (*produktionv1.DeleteMachineBookingResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	bookingID, err := uuid.Parse(req.GetBookingId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid booking_id: %v", err)
	}

	if err := s.svc.DeleteMachineBooking(ctx, tenantID, bookingID); err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.DeleteMachineBookingResponse{}, nil
}

func (s *ProduktionGRPCServer) ListMachineBookings(ctx context.Context, req *produktionv1.ListMachineBookingsRequest) (*produktionv1.ListMachineBookingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.ListBookingsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	if req.MachineId != nil {
		input.MachineID = req.MachineId
	}
	if req.DateFrom != nil {
		t := req.DateFrom.AsTime()
		input.DateFrom = &t
	}
	if req.DateTo != nil {
		t := req.DateTo.AsTime()
		input.DateTo = &t
	}
	if req.ProductionOrderId != nil {
		id, parseErr := uuid.Parse(*req.ProductionOrderId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid production_order_id: %v", parseErr)
		}
		input.ProductionOrderID = &id
	}

	bookings, total, err := s.svc.ListMachineBookings(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}

	protoBookings := make([]*produktionv1.MachineBooking, len(bookings))
	for i, b := range bookings {
		protoBookings[i] = machineBookingToProto(b)
	}
	return &produktionv1.ListMachineBookingsResponse{
		Bookings: protoBookings,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Plan RPCs
// ============================================================================

func (s *ProduktionGRPCServer) CreatePlan(ctx context.Context, req *produktionv1.CreatePlanRequest) (*produktionv1.PlanResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := produktion.CreatePlanInput{
		TenantID:             tenantID,
		Name:                 req.GetName(),
		WeekNumber:           int(req.GetWeekNumber()),
		Year:                 int(req.GetYear()),
		TotalCapacityHours:   req.GetTotalCapacityHours(),
		PlannedCapacityHours: req.GetPlannedCapacityHours(),
		Notes:                req.GetNotes(),
	}

	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	plan, err := s.svc.CreatePlan(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *ProduktionGRPCServer) UpdatePlan(ctx context.Context, req *produktionv1.UpdatePlanRequest) (*produktionv1.PlanResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	planID, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}

	input := produktion.UpdatePlanInput{
		TenantID: tenantID,
		PlanID:   planID,
	}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.WeekNumber != nil {
		wn := int(*req.WeekNumber)
		input.WeekNumber = &wn
	}
	if req.Year != nil {
		y := int(*req.Year)
		input.Year = &y
	}
	if req.TotalCapacityHours != nil {
		input.TotalCapacityHours = req.TotalCapacityHours
	}
	if req.PlannedCapacityHours != nil {
		input.PlannedCapacityHours = req.PlannedCapacityHours
	}
	if req.Status != nil {
		st := produktion.PlanStatus(*req.Status)
		input.Status = &st
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	plan, err := s.svc.UpdatePlan(ctx, input)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *ProduktionGRPCServer) GetPlan(ctx context.Context, req *produktionv1.GetPlanRequest) (*produktionv1.PlanResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	planID, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}

	plan, err := s.svc.GetPlan(ctx, tenantID, planID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *ProduktionGRPCServer) GetCapacityOverview(ctx context.Context, req *produktionv1.GetCapacityOverviewRequest) (*produktionv1.CapacityOverviewResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	planID, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}

	overview, err := s.svc.GetCapacityOverview(ctx, tenantID, req.GetMachineId(), planID)
	if err != nil {
		return nil, mapProduktionError(err)
	}
	return &produktionv1.CapacityOverviewResponse{
		Overview: &produktionv1.CapacityOverview{
			MachineId:          overview.MachineID,
			TotalCapacityHours: overview.TotalCapacityHours,
			BookedHours:        overview.BookedHours,
			AvailableHours:     overview.AvailableHours,
		},
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func orderToProto(o *produktion.ProductionOrder) *produktionv1.ProductionOrder {
	if o == nil {
		return nil
	}
	p := &produktionv1.ProductionOrder{
		Id:           o.ID.String(),
		TenantId:     o.TenantID.String(),
		OrderNumber:  o.OrderNumber,
		ProductName:  o.ProductName,
		Quantity:     int32(o.Quantity),
		Status:       string(o.Status),
		PlannedStart: timestamppb.New(o.PlannedStart),
		PlannedEnd:   timestamppb.New(o.PlannedEnd),
		Priority:     int32(o.Priority),
		Notes:        o.Notes,
		CreatedAt:    timestamppb.New(o.CreatedAt),
		UpdatedAt:    timestamppb.New(o.UpdatedAt),
	}
	if o.ActualStart != nil {
		p.ActualStart = timestamppb.New(*o.ActualStart)
	}
	if o.ActualEnd != nil {
		p.ActualEnd = timestamppb.New(*o.ActualEnd)
	}
	if o.CreatedBy != nil {
		s := o.CreatedBy.String()
		p.CreatedBy = &s
	}
	return p
}

func machineBookingToProto(b *produktion.MachineBooking) *produktionv1.MachineBooking {
	if b == nil {
		return nil
	}
	p := &produktionv1.MachineBooking{
		Id:                b.ID.String(),
		TenantId:          b.TenantID.String(),
		MachineId:         b.MachineID,
		ProductionOrderId: b.ProductionOrderID.String(),
		StartsAt:          timestamppb.New(b.StartsAt),
		EndsAt:            timestamppb.New(b.EndsAt),
		Status:            string(b.Status),
		Notes:             b.Notes,
		CreatedAt:         timestamppb.New(b.CreatedAt),
		UpdatedAt:         timestamppb.New(b.UpdatedAt),
	}
	if b.CreatedBy != nil {
		s := b.CreatedBy.String()
		p.CreatedBy = &s
	}
	return p
}

func planToProto(pl *produktion.ProductionPlan) *produktionv1.ProductionPlan {
	if pl == nil {
		return nil
	}
	p := &produktionv1.ProductionPlan{
		Id:                   pl.ID.String(),
		TenantId:             pl.TenantID.String(),
		Name:                 pl.Name,
		WeekNumber:           int32(pl.WeekNumber),
		Year:                 int32(pl.Year),
		TotalCapacityHours:   pl.TotalCapacityHours,
		PlannedCapacityHours: pl.PlannedCapacityHours,
		Status:               string(pl.Status),
		Notes:                pl.Notes,
		CreatedAt:            timestamppb.New(pl.CreatedAt),
		UpdatedAt:            timestamppb.New(pl.UpdatedAt),
	}
	if pl.CreatedBy != nil {
		s := pl.CreatedBy.String()
		p.CreatedBy = &s
	}
	return p
}

// ============================================================================
// Error mapping
// ============================================================================

func mapProduktionError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, produktion.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrBookingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrPlanNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, produktion.ErrOrderNumberTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, produktion.ErrOrderNotDeletable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, produktion.ErrOrderInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, produktion.ErrBookingConflict):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, produktion.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled produktion service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

