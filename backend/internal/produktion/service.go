package produktion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles produktion business logic.
type Service struct {
	repo Repository
}

// NewService creates a new produktion service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types — Orders
// ============================================================================

// CreateOrderInput contains the data needed to create a production order.
type CreateOrderInput struct {
	TenantID     uuid.UUID
	OrderNumber  string
	ProductName  string
	Quantity     int
	PlannedStart time.Time
	PlannedEnd   time.Time
	Priority     int
	Notes        string
	CreatedBy    *uuid.UUID
}

// UpdateOrderInput contains the data that can be updated on a production order.
type UpdateOrderInput struct {
	TenantID     uuid.UUID
	OrderID      uuid.UUID
	ProductName  *string
	Quantity     *int
	PlannedStart *time.Time
	PlannedEnd   *time.Time
	Priority     *int
	Notes        *string
}

// ListOrdersInput contains filtering and pagination for listing orders.
type ListOrdersInput struct {
	TenantID uuid.UUID
	Status   *OrderStatus
	Priority *int
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PageSize int
}

// ============================================================================
// Input types — Bookings
// ============================================================================

// CreateBookingInput contains the data needed to create a machine booking.
type CreateBookingInput struct {
	TenantID          uuid.UUID
	MachineID         string
	ProductionOrderID uuid.UUID
	StartsAt          time.Time
	EndsAt            time.Time
	Notes             string
	CreatedBy         *uuid.UUID
}

// UpdateBookingInput contains the data that can be updated on a machine booking.
type UpdateBookingInput struct {
	TenantID          uuid.UUID
	BookingID         uuid.UUID
	MachineID         *string
	ProductionOrderID *uuid.UUID
	StartsAt          *time.Time
	EndsAt            *time.Time
	Notes             *string
}

// ListBookingsInput contains filtering and pagination for listing bookings.
type ListBookingsInput struct {
	TenantID          uuid.UUID
	MachineID         *string
	DateFrom          *time.Time
	DateTo            *time.Time
	ProductionOrderID *uuid.UUID
	Page              int
	PageSize          int
}

// ============================================================================
// Input types — Plans
// ============================================================================

// CreatePlanInput contains the data needed to create a production plan.
type CreatePlanInput struct {
	TenantID             uuid.UUID
	Name                 string
	WeekNumber           int
	Year                 int
	TotalCapacityHours   float64
	PlannedCapacityHours float64
	Notes                string
	CreatedBy            *uuid.UUID
}

// UpdatePlanInput contains the data that can be updated on a production plan.
type UpdatePlanInput struct {
	TenantID             uuid.UUID
	PlanID               uuid.UUID
	Name                 *string
	WeekNumber           *int
	Year                 *int
	TotalCapacityHours   *float64
	PlannedCapacityHours *float64
	Status               *PlanStatus
	Notes                *string
}

// ============================================================================
// Order methods
// ============================================================================

// CreateOrder creates a new production order.
func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (*ProductionOrder, error) {
	orderNumber := strings.TrimSpace(input.OrderNumber)
	if orderNumber == "" {
		return nil, ErrInvalidInput
	}
	productName := strings.TrimSpace(input.ProductName)
	if productName == "" {
		return nil, ErrInvalidInput
	}
	if input.Quantity <= 0 {
		return nil, ErrInvalidInput
	}
	if !input.PlannedEnd.After(input.PlannedStart) {
		return nil, ErrInvalidInput
	}

	priority := input.Priority
	if priority < 1 || priority > 5 {
		priority = 3
	}

	exists, err := s.repo.OrderNumberExists(ctx, input.TenantID, orderNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("check order number: %w", err)
	}
	if exists {
		return nil, ErrOrderNumberTaken
	}

	now := time.Now()
	order := &ProductionOrder{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		OrderNumber:  orderNumber,
		ProductName:  productName,
		Quantity:     input.Quantity,
		Status:       OrderStatusPlanned,
		PlannedStart: input.PlannedStart,
		PlannedEnd:   input.PlannedEnd,
		Priority:     priority,
		Notes:        input.Notes,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if createErr := s.repo.CreateOrder(ctx, order); createErr != nil {
		return nil, fmt.Errorf("create order: %w", createErr)
	}

	slog.Info("production order created",
		"order_id", order.ID,
		"tenant_id", order.TenantID,
		"order_number", order.OrderNumber,
	)

	return order, nil
}

// UpdateOrder updates mutable fields of an existing production order.
func (s *Service) UpdateOrder(ctx context.Context, input UpdateOrderInput) (*ProductionOrder, error) {
	order, err := s.repo.GetOrder(ctx, input.TenantID, input.OrderID)
	if err != nil {
		return nil, err
	}

	if input.ProductName != nil {
		name := strings.TrimSpace(*input.ProductName)
		if name == "" {
			return nil, ErrInvalidInput
		}
		order.ProductName = name
	}

	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
		order.Quantity = *input.Quantity
	}

	if input.PlannedStart != nil {
		order.PlannedStart = *input.PlannedStart
	}

	if input.PlannedEnd != nil {
		order.PlannedEnd = *input.PlannedEnd
	}

	if !order.PlannedEnd.After(order.PlannedStart) {
		return nil, ErrInvalidInput
	}

	if input.Priority != nil {
		p := *input.Priority
		if p < 1 || p > 5 {
			return nil, ErrInvalidInput
		}
		order.Priority = p
	}

	if input.Notes != nil {
		order.Notes = *input.Notes
	}

	order.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateOrder(ctx, order); updateErr != nil {
		return nil, fmt.Errorf("update order: %w", updateErr)
	}

	slog.Info("production order updated",
		"order_id", order.ID,
		"tenant_id", order.TenantID,
	)

	return order, nil
}

// DeleteOrder removes a production order. Only orders in planned status can be deleted.
func (s *Service) DeleteOrder(ctx context.Context, tenantID, orderID uuid.UUID) error {
	order, err := s.repo.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return err
	}

	if order.Status != OrderStatusPlanned {
		return ErrOrderNotDeletable
	}

	if delErr := s.repo.DeleteOrder(ctx, tenantID, orderID); delErr != nil {
		return fmt.Errorf("delete order: %w", delErr)
	}

	slog.Info("production order deleted",
		"order_id", orderID,
		"tenant_id", tenantID,
	)

	return nil
}

// GetOrder retrieves a production order by ID.
func (s *Service) GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	return s.repo.GetOrder(ctx, tenantID, orderID)
}

// ListOrders retrieves orders with optional filtering and pagination.
func (s *Service) ListOrders(ctx context.Context, input ListOrdersInput) ([]*ProductionOrder, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListOrdersFilter{
		Status:   input.Status,
		Priority: input.Priority,
		DateFrom: input.DateFrom,
		DateTo:   input.DateTo,
	}
	return s.repo.ListOrders(ctx, input.TenantID, filter, offset, input.PageSize)
}

// StartOrder transitions an order from planned to in_progress and records actual_start.
func (s *Service) StartOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	order, err := s.repo.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != OrderStatusPlanned {
		return nil, ErrOrderInvalidTransition
	}

	now := time.Now()
	order.Status = OrderStatusInProgress
	order.ActualStart = &now
	order.UpdatedAt = now

	if updateErr := s.repo.UpdateOrder(ctx, order); updateErr != nil {
		return nil, fmt.Errorf("start order: %w", updateErr)
	}

	slog.Info("production order started",
		"order_id", order.ID,
		"tenant_id", tenantID,
	)

	return order, nil
}

// CompleteOrder transitions an order from in_progress to completed and records actual_end.
func (s *Service) CompleteOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	order, err := s.repo.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != OrderStatusInProgress {
		return nil, ErrOrderInvalidTransition
	}

	now := time.Now()
	order.Status = OrderStatusCompleted
	order.ActualEnd = &now
	order.UpdatedAt = now

	if updateErr := s.repo.UpdateOrder(ctx, order); updateErr != nil {
		return nil, fmt.Errorf("complete order: %w", updateErr)
	}

	slog.Info("production order completed",
		"order_id", order.ID,
		"tenant_id", tenantID,
	)

	return order, nil
}

// CancelOrder transitions any non-completed order to cancelled.
func (s *Service) CancelOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	order, err := s.repo.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == OrderStatusCompleted || order.Status == OrderStatusCancelled {
		return nil, ErrOrderInvalidTransition
	}

	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateOrder(ctx, order); updateErr != nil {
		return nil, fmt.Errorf("cancel order: %w", updateErr)
	}

	slog.Info("production order cancelled",
		"order_id", order.ID,
		"tenant_id", tenantID,
	)

	return order, nil
}

// ============================================================================
// Booking methods
// ============================================================================

// CreateMachineBooking creates a new machine booking with atomic conflict detection.
//
// The conflict-check and INSERT are executed inside a single Postgres transaction protected by
// pg_advisory_xact_lock (via CreateBookingWithLock), eliminating the TOCTOU race that existed
// when FindConflictingBooking and CreateBooking were two separate non-transactional calls.
func (s *Service) CreateMachineBooking(ctx context.Context, input CreateBookingInput) (*MachineBooking, error) {
	machineID := strings.TrimSpace(input.MachineID)
	if machineID == "" {
		return nil, ErrInvalidInput
	}
	if !input.EndsAt.After(input.StartsAt) {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	booking := &MachineBooking{
		ID:                uuid.New(),
		TenantID:          input.TenantID,
		MachineID:         machineID,
		ProductionOrderID: input.ProductionOrderID,
		StartsAt:          input.StartsAt,
		EndsAt:            input.EndsAt,
		Status:            BookingStatusBooked,
		Notes:             input.Notes,
		CreatedBy:         input.CreatedBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	conflictID, err := s.repo.CreateBookingWithLock(ctx, booking)
	if err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}
	if conflictID != nil {
		return nil, fmt.Errorf("%w: conflicts with booking %s", ErrBookingConflict, conflictID)
	}

	slog.Info("machine booking created",
		"booking_id", booking.ID,
		"machine_id", machineID,
		"tenant_id", input.TenantID,
	)

	return booking, nil
}

// UpdateMachineBooking updates a machine booking with conflict detection.
func (s *Service) UpdateMachineBooking(ctx context.Context, input UpdateBookingInput) (*MachineBooking, error) {
	booking, err := s.repo.GetBooking(ctx, input.TenantID, input.BookingID)
	if err != nil {
		return nil, err
	}

	if input.MachineID != nil {
		mid := strings.TrimSpace(*input.MachineID)
		if mid == "" {
			return nil, ErrInvalidInput
		}
		booking.MachineID = mid
	}

	if input.ProductionOrderID != nil {
		booking.ProductionOrderID = *input.ProductionOrderID
	}

	if input.StartsAt != nil {
		booking.StartsAt = *input.StartsAt
	}

	if input.EndsAt != nil {
		booking.EndsAt = *input.EndsAt
	}

	if !booking.EndsAt.After(booking.StartsAt) {
		return nil, ErrInvalidInput
	}

	if input.Notes != nil {
		booking.Notes = *input.Notes
	}

	// Conflict check excluding self.
	conflictID, err := s.repo.FindConflictingBooking(ctx, input.TenantID, booking.MachineID, booking.StartsAt, booking.EndsAt, &booking.ID)
	if err != nil {
		return nil, fmt.Errorf("check booking conflict: %w", err)
	}
	if conflictID != nil {
		return nil, fmt.Errorf("%w: conflicts with booking %s", ErrBookingConflict, conflictID)
	}

	booking.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateBooking(ctx, booking); updateErr != nil {
		return nil, fmt.Errorf("update booking: %w", updateErr)
	}

	slog.Info("machine booking updated",
		"booking_id", booking.ID,
		"tenant_id", input.TenantID,
	)

	return booking, nil
}

// DeleteMachineBooking removes a machine booking.
func (s *Service) DeleteMachineBooking(ctx context.Context, tenantID, bookingID uuid.UUID) error {
	if _, err := s.repo.GetBooking(ctx, tenantID, bookingID); err != nil {
		return err
	}

	if delErr := s.repo.DeleteBooking(ctx, tenantID, bookingID); delErr != nil {
		return fmt.Errorf("delete booking: %w", delErr)
	}

	slog.Info("machine booking deleted",
		"booking_id", bookingID,
		"tenant_id", tenantID,
	)

	return nil
}

// ListMachineBookings retrieves bookings with optional filtering and pagination.
func (s *Service) ListMachineBookings(ctx context.Context, input ListBookingsInput) ([]*MachineBooking, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListBookingsFilter{
		MachineID:         input.MachineID,
		DateFrom:          input.DateFrom,
		DateTo:            input.DateTo,
		ProductionOrderID: input.ProductionOrderID,
	}
	return s.repo.ListBookings(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ============================================================================
// Plan methods
// ============================================================================

// CreatePlan creates a new production plan.
func (s *Service) CreatePlan(ctx context.Context, input CreatePlanInput) (*ProductionPlan, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	if input.WeekNumber < 1 || input.WeekNumber > 53 {
		return nil, ErrInvalidInput
	}
	if input.Year < 2000 {
		return nil, ErrInvalidInput
	}
	if input.TotalCapacityHours < 0 {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	plan := &ProductionPlan{
		ID:                   uuid.New(),
		TenantID:             input.TenantID,
		Name:                 name,
		WeekNumber:           input.WeekNumber,
		Year:                 input.Year,
		TotalCapacityHours:   input.TotalCapacityHours,
		PlannedCapacityHours: input.PlannedCapacityHours,
		Status:               PlanStatusDraft,
		Notes:                input.Notes,
		CreatedBy:            input.CreatedBy,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if createErr := s.repo.CreatePlan(ctx, plan); createErr != nil {
		return nil, fmt.Errorf("create plan: %w", createErr)
	}

	slog.Info("production plan created",
		"plan_id", plan.ID,
		"tenant_id", plan.TenantID,
		"week", plan.WeekNumber,
		"year", plan.Year,
	)

	return plan, nil
}

// UpdatePlan updates an existing production plan.
func (s *Service) UpdatePlan(ctx context.Context, input UpdatePlanInput) (*ProductionPlan, error) {
	plan, err := s.repo.GetPlan(ctx, input.TenantID, input.PlanID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		plan.Name = name
	}

	if input.WeekNumber != nil {
		if *input.WeekNumber < 1 || *input.WeekNumber > 53 {
			return nil, ErrInvalidInput
		}
		plan.WeekNumber = *input.WeekNumber
	}

	if input.Year != nil {
		if *input.Year < 2000 {
			return nil, ErrInvalidInput
		}
		plan.Year = *input.Year
	}

	if input.TotalCapacityHours != nil {
		if *input.TotalCapacityHours < 0 {
			return nil, ErrInvalidInput
		}
		plan.TotalCapacityHours = *input.TotalCapacityHours
	}

	if input.PlannedCapacityHours != nil {
		plan.PlannedCapacityHours = *input.PlannedCapacityHours
	}

	if input.Status != nil {
		plan.Status = *input.Status
	}

	if input.Notes != nil {
		plan.Notes = *input.Notes
	}

	plan.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdatePlan(ctx, plan); updateErr != nil {
		return nil, fmt.Errorf("update plan: %w", updateErr)
	}

	slog.Info("production plan updated",
		"plan_id", plan.ID,
		"tenant_id", plan.TenantID,
	)

	return plan, nil
}

// GetPlan retrieves a production plan by ID.
func (s *Service) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*ProductionPlan, error) {
	return s.repo.GetPlan(ctx, tenantID, planID)
}

// GetCapacityOverview returns machine capacity utilization for a given plan.
func (s *Service) GetCapacityOverview(ctx context.Context, tenantID uuid.UUID, machineID string, planID uuid.UUID) (*CapacityOverview, error) {
	if strings.TrimSpace(machineID) == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.GetCapacityOverview(ctx, tenantID, machineID, planID)
}
