package produktion

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repository
// ============================================================================

type mockRepository struct {
	orders   map[uuid.UUID]*ProductionOrder
	bookings map[uuid.UUID]*MachineBooking
	plans    map[uuid.UUID]*ProductionPlan

	// injected errors for specific methods
	createOrderErr   error
	createBookingErr error

	// for FindConflictingBooking: map machineID -> conflicting bookingID
	conflicts map[string]*uuid.UUID
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		orders:    make(map[uuid.UUID]*ProductionOrder),
		bookings:  make(map[uuid.UUID]*MachineBooking),
		plans:     make(map[uuid.UUID]*ProductionPlan),
		conflicts: make(map[string]*uuid.UUID),
	}
}

func (m *mockRepository) CreateOrder(ctx context.Context, order *ProductionOrder) error {
	if m.createOrderErr != nil {
		return m.createOrderErr
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockRepository) UpdateOrder(ctx context.Context, order *ProductionOrder) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockRepository) GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error) {
	o, ok := m.orders[orderID]
	if !ok || o.TenantID != tenantID {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (m *mockRepository) ListOrders(ctx context.Context, tenantID uuid.UUID, filter ListOrdersFilter, offset, limit int) ([]*ProductionOrder, int, error) {
	var result []*ProductionOrder
	for _, o := range m.orders {
		if o.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && o.Status != *filter.Status {
			continue
		}
		result = append(result, o)
	}
	total := len(result)
	if offset >= total {
		return []*ProductionOrder{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) DeleteOrder(ctx context.Context, tenantID, orderID uuid.UUID) error {
	delete(m.orders, orderID)
	return nil
}

func (m *mockRepository) OrderNumberExists(ctx context.Context, tenantID uuid.UUID, orderNumber string, excludeID *uuid.UUID) (bool, error) {
	for _, o := range m.orders {
		if o.TenantID == tenantID && o.OrderNumber == orderNumber {
			if excludeID != nil && o.ID == *excludeID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) CreateBooking(ctx context.Context, booking *MachineBooking) error {
	if m.createBookingErr != nil {
		return m.createBookingErr
	}
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockRepository) UpdateBooking(ctx context.Context, booking *MachineBooking) error {
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockRepository) GetBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*MachineBooking, error) {
	b, ok := m.bookings[bookingID]
	if !ok || b.TenantID != tenantID {
		return nil, ErrBookingNotFound
	}
	return b, nil
}

func (m *mockRepository) ListBookings(ctx context.Context, tenantID uuid.UUID, filter ListBookingsFilter, offset, limit int) ([]*MachineBooking, int, error) {
	var result []*MachineBooking
	for _, b := range m.bookings {
		if b.TenantID != tenantID {
			continue
		}
		if filter.MachineID != nil && b.MachineID != *filter.MachineID {
			continue
		}
		result = append(result, b)
	}
	total := len(result)
	if offset >= total {
		return []*MachineBooking{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) DeleteBooking(ctx context.Context, tenantID, bookingID uuid.UUID) error {
	delete(m.bookings, bookingID)
	return nil
}

func (m *mockRepository) FindConflictingBooking(ctx context.Context, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error) {
	// Simple overlap check across all stored bookings (mirrors real SQL logic).
	for _, b := range m.bookings {
		if b.TenantID != tenantID || b.MachineID != machineID {
			continue
		}
		if b.Status != BookingStatusBooked && b.Status != BookingStatusInUse {
			continue
		}
		if excludeID != nil && b.ID == *excludeID {
			continue
		}
		// Overlap: b.starts_at < endsAt AND b.ends_at > startsAt
		if b.StartsAt.Before(endsAt) && b.EndsAt.After(startsAt) {
			id := b.ID
			return &id, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) CreatePlan(ctx context.Context, plan *ProductionPlan) error {
	m.plans[plan.ID] = plan
	return nil
}

func (m *mockRepository) UpdatePlan(ctx context.Context, plan *ProductionPlan) error {
	m.plans[plan.ID] = plan
	return nil
}

func (m *mockRepository) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*ProductionPlan, error) {
	p, ok := m.plans[planID]
	if !ok || p.TenantID != tenantID {
		return nil, ErrPlanNotFound
	}
	return p, nil
}

func (m *mockRepository) GetCapacityOverview(ctx context.Context, tenantID uuid.UUID, machineID string, planID uuid.UUID) (*CapacityOverview, error) {
	plan, err := m.GetPlan(ctx, tenantID, planID)
	if err != nil {
		return nil, err
	}

	weekStart, weekEnd := isoWeekBounds(plan.Year, plan.WeekNumber)

	var booked float64
	for _, b := range m.bookings {
		if b.TenantID != tenantID || b.MachineID != machineID {
			continue
		}
		if b.Status == BookingStatusCancelled {
			continue
		}
		if b.StartsAt.Before(weekEnd) && b.EndsAt.After(weekStart) {
			booked += b.EndsAt.Sub(b.StartsAt).Hours()
		}
	}

	avail := plan.TotalCapacityHours - booked
	if avail < 0 {
		avail = 0
	}
	return &CapacityOverview{
		MachineID:          machineID,
		TotalCapacityHours: plan.TotalCapacityHours,
		BookedHours:        booked,
		AvailableHours:     avail,
	}, nil
}

// compile-time interface check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

var (
	t0 = time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	t1 = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	t2 = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	t3 = time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
)

func addOrder(repo *mockRepository, tenantID uuid.UUID, orderNumber string, status OrderStatus) *ProductionOrder {
	o := &ProductionOrder{
		ID:           uuid.New(),
		TenantID:     tenantID,
		OrderNumber:  orderNumber,
		ProductName:  "Widget",
		Quantity:     10,
		Status:       status,
		PlannedStart: t0,
		PlannedEnd:   t1,
		Priority:     3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.orders[o.ID] = o
	return o
}

func addBooking(repo *mockRepository, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, status BookingStatus) *MachineBooking {
	b := &MachineBooking{
		ID:                uuid.New(),
		TenantID:          tenantID,
		MachineID:         machineID,
		ProductionOrderID: uuid.New(),
		StartsAt:          startsAt,
		EndsAt:            endsAt,
		Status:            status,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.bookings[b.ID] = b
	return b
}

// ============================================================================
// CreateOrder Tests
// ============================================================================

func TestService_CreateOrder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     tenantID,
		OrderNumber:  "PO-001",
		ProductName:  "Schraube M8",
		Quantity:     100,
		PlannedStart: t0,
		PlannedEnd:   t1,
		Priority:     2,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "PO-001", order.OrderNumber)
	assert.Equal(t, OrderStatusPlanned, order.Status)
	assert.Equal(t, 2, order.Priority)
}

func TestService_CreateOrder_DefaultPriority(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     uuid.New(),
		OrderNumber:  "PO-002",
		ProductName:  "Bolzen",
		Quantity:     50,
		PlannedStart: t0,
		PlannedEnd:   t1,
		Priority:     0, // invalid -> default 3
	})

	require.NoError(t, err)
	assert.Equal(t, 3, order.Priority)
}

func TestService_CreateOrder_EmptyProductName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     uuid.New(),
		OrderNumber:  "PO-003",
		ProductName:  "  ",
		Quantity:     10,
		PlannedStart: t0,
		PlannedEnd:   t1,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateOrder_InvalidDates(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     uuid.New(),
		OrderNumber:  "PO-004",
		ProductName:  "Widget",
		Quantity:     10,
		PlannedStart: t1,
		PlannedEnd:   t0, // end before start
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateOrder_DuplicateOrderNumber(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addOrder(repo, tenantID, "PO-DUP", OrderStatusPlanned)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     tenantID,
		OrderNumber:  "PO-DUP",
		ProductName:  "Widget",
		Quantity:     5,
		PlannedStart: t0,
		PlannedEnd:   t1,
	})

	assert.ErrorIs(t, err, ErrOrderNumberTaken)
}

func TestService_CreateOrder_ZeroQuantity(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     uuid.New(),
		OrderNumber:  "PO-005",
		ProductName:  "Widget",
		Quantity:     0,
		PlannedStart: t0,
		PlannedEnd:   t1,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// DeleteOrder Tests
// ============================================================================

func TestService_DeleteOrder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-DEL", OrderStatusPlanned)

	err := svc.DeleteOrder(context.Background(), tenantID, o.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.orders, o.ID)
}

func TestService_DeleteOrder_NonPlanned(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-IP", OrderStatusInProgress)

	err := svc.DeleteOrder(context.Background(), tenantID, o.ID)

	assert.ErrorIs(t, err, ErrOrderNotDeletable)
}

func TestService_DeleteOrder_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteOrder(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrOrderNotFound)
}

// ============================================================================
// Status Transition Tests
// ============================================================================

func TestService_StartOrder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-START", OrderStatusPlanned)

	result, err := svc.StartOrder(context.Background(), tenantID, o.ID)

	require.NoError(t, err)
	assert.Equal(t, OrderStatusInProgress, result.Status)
	assert.NotNil(t, result.ActualStart)
}

func TestService_StartOrder_InvalidTransition(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-COMP", OrderStatusCompleted)

	_, err := svc.StartOrder(context.Background(), tenantID, o.ID)

	assert.ErrorIs(t, err, ErrOrderInvalidTransition)
}

func TestService_CompleteOrder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-COMPL", OrderStatusInProgress)

	result, err := svc.CompleteOrder(context.Background(), tenantID, o.ID)

	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, result.Status)
	assert.NotNil(t, result.ActualEnd)
}

func TestService_CancelOrder_FromPlanned(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-CANC", OrderStatusPlanned)

	result, err := svc.CancelOrder(context.Background(), tenantID, o.ID)

	require.NoError(t, err)
	assert.Equal(t, OrderStatusCancelled, result.Status)
}

func TestService_CancelOrder_AlreadyCompleted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-COMPCANC", OrderStatusCompleted)

	_, err := svc.CancelOrder(context.Background(), tenantID, o.ID)

	assert.ErrorIs(t, err, ErrOrderInvalidTransition)
}

// ============================================================================
// Machine Booking Conflict Tests (core feature)
// ============================================================================

func TestService_CreateMachineBooking_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	booking, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         "M-001",
		ProductionOrderID: uuid.New(),
		StartsAt:          t0,
		EndsAt:            t1,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, booking.ID)
	assert.Equal(t, BookingStatusBooked, booking.Status)
}

func TestService_CreateMachineBooking_ConflictDetected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Existing booking: t0..t2
	addBooking(repo, tenantID, "M-001", t0, t2, BookingStatusBooked)

	// New booking overlaps: t1..t3
	_, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         "M-001",
		ProductionOrderID: uuid.New(),
		StartsAt:          t1,
		EndsAt:            t3,
	})

	assert.ErrorIs(t, err, ErrBookingConflict)
}

func TestService_CreateMachineBooking_NoConflictDifferentMachine(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Existing booking on M-001
	addBooking(repo, tenantID, "M-001", t0, t2, BookingStatusBooked)

	// New booking on M-002 — same time window, different machine
	booking, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         "M-002",
		ProductionOrderID: uuid.New(),
		StartsAt:          t0,
		EndsAt:            t2,
	})

	require.NoError(t, err)
	assert.Equal(t, "M-002", booking.MachineID)
}

func TestService_CreateMachineBooking_CancelledDoesNotConflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Cancelled booking — must not block new booking
	addBooking(repo, tenantID, "M-001", t0, t2, BookingStatusCancelled)

	booking, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         "M-001",
		ProductionOrderID: uuid.New(),
		StartsAt:          t0,
		EndsAt:            t2,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, booking.ID)
}

func TestService_CreateMachineBooking_AdjacentNoConflict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Booking t0..t1, new booking starts exactly at t1 — no overlap.
	addBooking(repo, tenantID, "M-001", t0, t1, BookingStatusBooked)

	booking, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          tenantID,
		MachineID:         "M-001",
		ProductionOrderID: uuid.New(),
		StartsAt:          t1,
		EndsAt:            t2,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, booking.ID)
}

func TestService_UpdateMachineBooking_ConflictExcludesSelf(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// A single booking t0..t1; we update it to t0..t2 — no external conflict.
	b := addBooking(repo, tenantID, "M-001", t0, t1, BookingStatusBooked)

	newEnd := t2
	result, err := svc.UpdateMachineBooking(context.Background(), UpdateBookingInput{
		TenantID:  tenantID,
		BookingID: b.ID,
		EndsAt:    &newEnd,
	})

	require.NoError(t, err)
	assert.Equal(t, t2, result.EndsAt)
}

func TestService_UpdateMachineBooking_ConflictWithOther(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Two bookings on same machine, adjacent.
	addBooking(repo, tenantID, "M-001", t0, t1, BookingStatusBooked)
	b2 := addBooking(repo, tenantID, "M-001", t2, t3, BookingStatusBooked)

	// Extend b2 backward to overlap first booking.
	newStart := t0
	_, err := svc.UpdateMachineBooking(context.Background(), UpdateBookingInput{
		TenantID:  tenantID,
		BookingID: b2.ID,
		StartsAt:  &newStart,
	})

	assert.ErrorIs(t, err, ErrBookingConflict)
}

func TestService_CreateMachineBooking_EmptyMachineID(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
		TenantID:          uuid.New(),
		MachineID:         "  ",
		ProductionOrderID: uuid.New(),
		StartsAt:          t0,
		EndsAt:            t1,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_DeleteMachineBooking_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	b := addBooking(repo, tenantID, "M-001", t0, t1, BookingStatusBooked)

	err := svc.DeleteMachineBooking(context.Background(), tenantID, b.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.bookings, b.ID)
}

func TestService_DeleteMachineBooking_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteMachineBooking(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrBookingNotFound)
}

// ============================================================================
// Plan Tests
// ============================================================================

func TestService_CreatePlan_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:           tenantID,
		Name:               "KW 22 2026",
		WeekNumber:         22,
		Year:               2026,
		TotalCapacityHours: 40,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, plan.ID)
	assert.Equal(t, PlanStatusDraft, plan.Status)
	assert.Equal(t, 40.0, plan.TotalCapacityHours)
}

func TestService_CreatePlan_InvalidWeek(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:   uuid.New(),
		Name:       "Bad",
		WeekNumber: 0, // invalid
		Year:       2026,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_GetPlan_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetPlan(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrPlanNotFound)
}

func TestService_GetCapacityOverview_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Create a plan for ISO week 22 year 2026 (Mon 25 May – Sun 31 May).
	plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:           tenantID,
		Name:               "KW 22",
		WeekNumber:         22,
		Year:               2026,
		TotalCapacityHours: 40,
	})
	require.NoError(t, err)

	// Add 8h booking within the week.
	monday := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)
	addBooking(repo, tenantID, "M-001", monday, monday.Add(8*time.Hour), BookingStatusBooked)

	overview, err := svc.GetCapacityOverview(context.Background(), tenantID, "M-001", plan.ID)

	require.NoError(t, err)
	assert.Equal(t, 40.0, overview.TotalCapacityHours)
	assert.Equal(t, 8.0, overview.BookedHours)
	assert.Equal(t, 32.0, overview.AvailableHours)
}
