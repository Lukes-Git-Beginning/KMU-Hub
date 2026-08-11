package produktion

import (
	"context"
	"errors"
	"sync"
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
	mu            sync.Mutex // guards all map access and CreateBookingWithLock atomicity
	orders        map[uuid.UUID]*ProductionOrder
	bookings      map[uuid.UUID]*MachineBooking
	plans         map[uuid.UUID]*ProductionPlan
	boms          map[uuid.UUID]*BOM
	workSteps     map[uuid.UUID]*WorkStep
	machines      map[uuid.UUID]*Machine
	qualityChecks map[uuid.UUID]*QualityCheck

	// injected errors for specific methods
	createOrderErr   error
	createBookingErr error

	// for FindConflictingBooking: map machineID -> conflicting bookingID
	conflicts map[string]*uuid.UUID

	// captured args from the last List* call, for pagination-clamping assertions
	lastListBOMsOffset, lastListBOMsLimit                   int
	lastListMachinesOffset, lastListMachinesLimit           int
	lastListQualityChecksOffset, lastListQualityChecksLimit int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		orders:        make(map[uuid.UUID]*ProductionOrder),
		bookings:      make(map[uuid.UUID]*MachineBooking),
		plans:         make(map[uuid.UUID]*ProductionPlan),
		boms:          make(map[uuid.UUID]*BOM),
		workSteps:     make(map[uuid.UUID]*WorkStep),
		machines:      make(map[uuid.UUID]*Machine),
		qualityChecks: make(map[uuid.UUID]*QualityCheck),
		conflicts:     make(map[string]*uuid.UUID),
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createBookingErr != nil {
		return m.createBookingErr
	}
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockRepository) UpdateBooking(ctx context.Context, booking *MachineBooking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bookings[booking.ID] = booking
	return nil
}

func (m *mockRepository) GetBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*MachineBooking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[bookingID]
	if !ok || b.TenantID != tenantID {
		return nil, ErrBookingNotFound
	}
	return b, nil
}

func (m *mockRepository) ListBookings(ctx context.Context, tenantID uuid.UUID, filter ListBookingsFilter, offset, limit int) ([]*MachineBooking, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bookings, bookingID)
	return nil
}

func (m *mockRepository) FindConflictingBooking(ctx context.Context, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

// CreateBookingWithLock simulates the advisory-lock Tx from PostgresRepository.
// The mutex on mockRepository ensures only one goroutine runs the check+insert at a time,
// mirroring the serialisation that pg_advisory_xact_lock provides in production.
func (m *mockRepository) CreateBookingWithLock(ctx context.Context, booking *MachineBooking) (*uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Conflict check (identical logic to FindConflictingBooking but under the same lock).
	for _, b := range m.bookings {
		if b.TenantID != booking.TenantID || b.MachineID != booking.MachineID {
			continue
		}
		if b.Status != BookingStatusBooked && b.Status != BookingStatusInUse {
			continue
		}
		// Overlap: b.starts_at < booking.EndsAt AND b.ends_at > booking.StartsAt
		if b.StartsAt.Before(booking.EndsAt) && b.EndsAt.After(booking.StartsAt) {
			id := b.ID
			return &id, nil
		}
	}

	// No conflict — store the booking.
	if m.createBookingErr != nil {
		return nil, m.createBookingErr
	}
	m.bookings[booking.ID] = booking
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

// ============================================================================
// Extended Repository methods (BOMs, WorkSteps, Machines, QualityChecks)
// BOM Create/Get are backed by the boms map (needed for material
// availability tests); the rest are minimal not-found stubs.
// ============================================================================

func (m *mockRepository) CreateBOM(_ context.Context, bom *BOM) error {
	m.boms[bom.ID] = bom
	return nil
}
func (m *mockRepository) UpdateBOM(_ context.Context, bom *BOM) error {
	existing, ok := m.boms[bom.ID]
	if !ok || existing.TenantID != bom.TenantID {
		return ErrBOMNotFound
	}
	m.boms[bom.ID] = bom
	return nil
}
func (m *mockRepository) GetBOM(_ context.Context, tenantID, bomID uuid.UUID) (*BOM, error) {
	bom, ok := m.boms[bomID]
	if !ok || bom.TenantID != tenantID {
		return nil, ErrBOMNotFound
	}
	return bom, nil
}
func (m *mockRepository) ListBOMs(_ context.Context, tenantID uuid.UUID, activeOnly *bool, offset, limit int) ([]*BOM, int, error) {
	m.lastListBOMsOffset, m.lastListBOMsLimit = offset, limit
	var result []*BOM
	for _, bom := range m.boms {
		if bom.TenantID != tenantID {
			continue
		}
		if activeOnly != nil && *activeOnly && !bom.Active {
			continue
		}
		result = append(result, bom)
	}
	return result, len(result), nil
}
func (m *mockRepository) DeleteBOM(_ context.Context, tenantID, bomID uuid.UUID) error {
	bom, ok := m.boms[bomID]
	if !ok || bom.TenantID != tenantID {
		return ErrBOMNotFound
	}
	delete(m.boms, bomID)
	return nil
}

func (m *mockRepository) CreateWorkStep(_ context.Context, step *WorkStep) error {
	m.workSteps[step.ID] = step
	return nil
}
func (m *mockRepository) UpdateWorkStep(_ context.Context, step *WorkStep) error {
	existing, ok := m.workSteps[step.ID]
	if !ok || existing.TenantID != step.TenantID {
		return ErrWorkStepNotFound
	}
	m.workSteps[step.ID] = step
	return nil
}
func (m *mockRepository) GetWorkStep(_ context.Context, tenantID, stepID uuid.UUID) (*WorkStep, error) {
	step, ok := m.workSteps[stepID]
	if !ok || step.TenantID != tenantID {
		return nil, ErrWorkStepNotFound
	}
	return step, nil
}
func (m *mockRepository) ListWorkSteps(_ context.Context, tenantID, orderID uuid.UUID) ([]*WorkStep, error) {
	var result []*WorkStep
	for _, step := range m.workSteps {
		if step.TenantID == tenantID && step.OrderID == orderID {
			result = append(result, step)
		}
	}
	return result, nil
}
func (m *mockRepository) DeleteWorkStep(_ context.Context, tenantID, stepID uuid.UUID) error {
	step, ok := m.workSteps[stepID]
	if !ok || step.TenantID != tenantID {
		return ErrWorkStepNotFound
	}
	delete(m.workSteps, stepID)
	return nil
}

func (m *mockRepository) CreateMachine(_ context.Context, machine *Machine) error {
	m.machines[machine.ID] = machine
	return nil
}
func (m *mockRepository) UpdateMachine(_ context.Context, machine *Machine) error {
	existing, ok := m.machines[machine.ID]
	if !ok || existing.TenantID != machine.TenantID {
		return ErrMachineNotFound
	}
	m.machines[machine.ID] = machine
	return nil
}
func (m *mockRepository) GetMachine(_ context.Context, tenantID, machineID uuid.UUID) (*Machine, error) {
	machine, ok := m.machines[machineID]
	if !ok || machine.TenantID != tenantID {
		return nil, ErrMachineNotFound
	}
	return machine, nil
}
func (m *mockRepository) ListMachines(_ context.Context, tenantID uuid.UUID, status *MachineStatus, offset, limit int) ([]*Machine, int, error) {
	m.lastListMachinesOffset, m.lastListMachinesLimit = offset, limit
	var result []*Machine
	for _, machine := range m.machines {
		if machine.TenantID != tenantID {
			continue
		}
		if status != nil && machine.Status != *status {
			continue
		}
		result = append(result, machine)
	}
	return result, len(result), nil
}
func (m *mockRepository) DeleteMachine(_ context.Context, tenantID, machineID uuid.UUID) error {
	machine, ok := m.machines[machineID]
	if !ok || machine.TenantID != tenantID {
		return ErrMachineNotFound
	}
	delete(m.machines, machineID)
	return nil
}

func (m *mockRepository) CreateQualityCheck(_ context.Context, check *QualityCheck) error {
	m.qualityChecks[check.ID] = check
	return nil
}
func (m *mockRepository) GetQualityCheck(_ context.Context, tenantID, checkID uuid.UUID) (*QualityCheck, error) {
	check, ok := m.qualityChecks[checkID]
	if !ok || check.TenantID != tenantID {
		return nil, ErrQualityCheckNotFound
	}
	return check, nil
}
func (m *mockRepository) ListQualityChecks(_ context.Context, tenantID uuid.UUID, orderID *uuid.UUID, offset, limit int) ([]*QualityCheck, int, error) {
	m.lastListQualityChecksOffset, m.lastListQualityChecksLimit = offset, limit
	var result []*QualityCheck
	for _, check := range m.qualityChecks {
		if check.TenantID != tenantID {
			continue
		}
		if orderID != nil && check.OrderID != *orderID {
			continue
		}
		result = append(result, check)
	}
	return result, len(result), nil
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
// UpdateOrder Validation Tests
// ============================================================================

func TestService_UpdateOrder_ValidationErrors(t *testing.T) {
	tenantID := uuid.New()

	cases := []struct {
		name  string
		input func(orderID uuid.UUID) UpdateOrderInput
	}{
		{
			name: "empty product name",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				empty := "   "
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, ProductName: &empty}
			},
		},
		{
			name: "zero quantity",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				zero := 0
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, Quantity: &zero}
			},
		},
		{
			name: "negative quantity",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				neg := -5
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, Quantity: &neg}
			},
		},
		{
			name: "planned end before planned start",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				newStart := t2
				newEnd := t0
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, PlannedStart: &newStart, PlannedEnd: &newEnd}
			},
		},
		{
			name: "priority below range",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				p := 0
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, Priority: &p}
			},
		},
		{
			name: "priority above range",
			input: func(orderID uuid.UUID) UpdateOrderInput {
				p := 6
				return UpdateOrderInput{TenantID: tenantID, OrderID: orderID, Priority: &p}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)
			o := addOrder(repo, tenantID, "PO-UPD-"+tc.name, OrderStatusPlanned)

			_, err := svc.UpdateOrder(context.Background(), tc.input(o.ID))

			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_UpdateOrder_UnknownBomID(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	o := addOrder(repo, tenantID, "PO-UPD-BOM", OrderStatusPlanned)
	unknownBomID := uuid.New()

	_, err := svc.UpdateOrder(context.Background(), UpdateOrderInput{
		TenantID: tenantID,
		OrderID:  o.ID,
		BomID:    &unknownBomID,
	})

	assert.ErrorIs(t, err, ErrBOMNotFound)
}

func TestService_UpdateOrder_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateOrder(context.Background(), UpdateOrderInput{
		TenantID: uuid.New(),
		OrderID:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrOrderNotFound)
}

// ============================================================================
// ListOrders / ListMachineBookings Tests
// ============================================================================

func TestService_ListOrders_FilterAndPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addOrder(repo, tenantID, "PO-L1", OrderStatusPlanned)
	addOrder(repo, tenantID, "PO-L2", OrderStatusInProgress)
	addOrder(repo, uuid.New(), "PO-OTHER-TENANT", OrderStatusPlanned)

	status := OrderStatusInProgress
	results, total, err := svc.ListOrders(context.Background(), ListOrdersInput{
		TenantID: tenantID,
		Status:   &status,
		Page:     0,   // invalid -> clamps to 1
		PageSize: 500, // out of range -> clamps to 20
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, OrderStatusInProgress, results[0].Status)
}

func TestService_ListMachineBookings_FilterAndPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addBooking(repo, tenantID, "M-WANTED", t0, t1, BookingStatusBooked)
	addBooking(repo, tenantID, "M-OTHER", t0, t1, BookingStatusBooked)
	addBooking(repo, uuid.New(), "M-WANTED", t0, t1, BookingStatusBooked)

	machineID := "M-WANTED"
	results, total, err := svc.ListMachineBookings(context.Background(), ListBookingsInput{
		TenantID:  tenantID,
		MachineID: &machineID,
		Page:      -1,  // invalid -> clamps to 1
		PageSize:  200, // out of range -> clamps to 20
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "M-WANTED", results[0].MachineID)
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

func TestService_GetPlan_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	created, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:           tenantID,
		Name:               "KW 30 2026",
		WeekNumber:         30,
		Year:               2026,
		TotalCapacityHours: 35,
	})
	require.NoError(t, err)

	plan, err := svc.GetPlan(context.Background(), tenantID, created.ID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, plan.ID)
	assert.Equal(t, "KW 30 2026", plan.Name)
}

// ============================================================================
// UpdatePlan Validation Tests (analogous to CreatePlan)
// ============================================================================

func TestService_UpdatePlan_ValidationErrors(t *testing.T) {
	tenantID := uuid.New()

	cases := []struct {
		name  string
		input func(planID uuid.UUID) UpdatePlanInput
	}{
		{
			name: "empty name",
			input: func(planID uuid.UUID) UpdatePlanInput {
				empty := "   "
				return UpdatePlanInput{TenantID: tenantID, PlanID: planID, Name: &empty}
			},
		},
		{
			name: "week below range",
			input: func(planID uuid.UUID) UpdatePlanInput {
				w := 0
				return UpdatePlanInput{TenantID: tenantID, PlanID: planID, WeekNumber: &w}
			},
		},
		{
			name: "week above range",
			input: func(planID uuid.UUID) UpdatePlanInput {
				w := 54
				return UpdatePlanInput{TenantID: tenantID, PlanID: planID, WeekNumber: &w}
			},
		},
		{
			name: "year below 2000",
			input: func(planID uuid.UUID) UpdatePlanInput {
				y := 1999
				return UpdatePlanInput{TenantID: tenantID, PlanID: planID, Year: &y}
			},
		},
		{
			name: "negative total capacity hours",
			input: func(planID uuid.UUID) UpdatePlanInput {
				h := -1.0
				return UpdatePlanInput{TenantID: tenantID, PlanID: planID, TotalCapacityHours: &h}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)
			plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
				TenantID:           tenantID,
				Name:               "Baseline " + tc.name,
				WeekNumber:         10,
				Year:               2026,
				TotalCapacityHours: 40,
			})
			require.NoError(t, err)

			_, err = svc.UpdatePlan(context.Background(), tc.input(plan.ID))

			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestService_UpdatePlan_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:           tenantID,
		Name:               "Original",
		WeekNumber:         10,
		Year:               2026,
		TotalCapacityHours: 40,
	})
	require.NoError(t, err)

	newName := "Updated"
	active := PlanStatusActive
	updated, err := svc.UpdatePlan(context.Background(), UpdatePlanInput{
		TenantID: tenantID,
		PlanID:   plan.ID,
		Name:     &newName,
		Status:   &active,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, PlanStatusActive, updated.Status)
}

func TestService_UpdatePlan_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdatePlan(context.Background(), UpdatePlanInput{
		TenantID: uuid.New(),
		PlanID:   uuid.New(),
	})

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

// ============================================================================
// Race Condition Test
// ============================================================================

// TestService_CreateMachineBooking_RaceConditionMutex verifies that the
// advisory-lock pattern (simulated via sync.Mutex in the mock) prevents
// double-booking under concurrent load.
//
// 50 goroutines all attempt to book the exact same machine for the exact same
// time window simultaneously.  Exactly one must succeed; the other 49 must
// receive ErrBookingConflict.  After the test, the repo must contain exactly
// one booking for that machine.
func TestService_CreateMachineBooking_RaceConditionMutex(t *testing.T) {
	const goroutines = 50

	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	orderID := uuid.New()
	machineID := "RACE-MACHINE-001"
	startsAt := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	endsAt := time.Date(2026, 7, 1, 16, 0, 0, 0, time.UTC)

	type result struct {
		booking *MachineBooking
		err     error
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			b, err := svc.CreateMachineBooking(context.Background(), CreateBookingInput{
				TenantID:          tenantID,
				MachineID:         machineID,
				ProductionOrderID: orderID,
				StartsAt:          startsAt,
				EndsAt:            endsAt,
			})
			results[idx] = result{booking: b, err: err}
		}(i)
	}

	wg.Wait()

	var successes, conflicts, other int
	for _, r := range results {
		switch {
		case r.err == nil:
			successes++
		case isBookingConflict(r.err):
			conflicts++
		default:
			other++
			t.Errorf("unexpected error: %v", r.err)
		}
	}

	assert.Equal(t, 1, successes, "exactly one goroutine must succeed")
	assert.Equal(t, goroutines-1, conflicts, "all other goroutines must get ErrBookingConflict")
	assert.Equal(t, 0, other, "no unexpected errors")

	// Verify exactly one booking in the repo.
	repo.mu.Lock()
	bookingCount := 0
	for _, b := range repo.bookings {
		if b.MachineID == machineID && b.TenantID == tenantID {
			bookingCount++
		}
	}
	repo.mu.Unlock()

	assert.Equal(t, 1, bookingCount, "repo must contain exactly one booking after concurrent attempts")
}

// isBookingConflict unwraps wrapped errors to check for ErrBookingConflict.
func isBookingConflict(err error) bool {
	return errors.Is(err, ErrBookingConflict)
}
