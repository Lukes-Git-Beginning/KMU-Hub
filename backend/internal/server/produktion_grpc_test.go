package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/produktion"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
)

// ---------------------------------------------------------------------------
// stub produktion.Repository
// ---------------------------------------------------------------------------

// errStubProduktionFailure is a generic non-sentinel error used to reach the
// fallback "slog.Error + codes.Internal" branch of mapProduktionError.
var errStubProduktionFailure = errors.New("stub produktion repository failure")

type stubProduktionRepo struct {
	forceErr error

	orders   map[uuid.UUID]*produktion.ProductionOrder
	bookings map[uuid.UUID]*produktion.MachineBooking
	plans    map[uuid.UUID]*produktion.ProductionPlan
	boms     map[uuid.UUID]*produktion.BOM
	steps    map[uuid.UUID]*produktion.WorkStep
	machines map[uuid.UUID]*produktion.Machine
	checks   map[uuid.UUID]*produktion.QualityCheck

	capacityOverview *produktion.CapacityOverview
}

func newStubProduktionRepo() *stubProduktionRepo {
	return &stubProduktionRepo{
		orders:   make(map[uuid.UUID]*produktion.ProductionOrder),
		bookings: make(map[uuid.UUID]*produktion.MachineBooking),
		plans:    make(map[uuid.UUID]*produktion.ProductionPlan),
		boms:     make(map[uuid.UUID]*produktion.BOM),
		steps:    make(map[uuid.UUID]*produktion.WorkStep),
		machines: make(map[uuid.UUID]*produktion.Machine),
		checks:   make(map[uuid.UUID]*produktion.QualityCheck),
	}
}

// --- Orders ---

func (r *stubProduktionRepo) CreateOrder(_ context.Context, order *produktion.ProductionOrder) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.orders[order.ID] = order
	return nil
}

func (r *stubProduktionRepo) UpdateOrder(_ context.Context, order *produktion.ProductionOrder) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.orders[order.ID] = order
	return nil
}

func (r *stubProduktionRepo) GetOrder(_ context.Context, tenantID, orderID uuid.UUID) (*produktion.ProductionOrder, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	o, ok := r.orders[orderID]
	if !ok || o.TenantID != tenantID {
		return nil, produktion.ErrOrderNotFound
	}
	// Return a copy — mirrors PostgresRepository.GetOrder building a fresh
	// struct from a SELECT. Returning the live map pointer would let a
	// caller's in-place mutation (e.g. UpdateOrder validating fields before
	// persisting) leak into the stored order even when the update errors out.
	cp := *o
	return &cp, nil
}

func (r *stubProduktionRepo) ListOrders(_ context.Context, tenantID uuid.UUID, filter produktion.ListOrdersFilter, offset, limit int) ([]*produktion.ProductionOrder, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*produktion.ProductionOrder
	for _, o := range r.orders {
		if o.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && o.Status != *filter.Status {
			continue
		}
		if filter.Priority != nil && o.Priority != *filter.Priority {
			continue
		}
		matched = append(matched, o)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubProduktionRepo) DeleteOrder(_ context.Context, tenantID, orderID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	o, ok := r.orders[orderID]
	if !ok || o.TenantID != tenantID {
		return produktion.ErrOrderNotFound
	}
	delete(r.orders, orderID)
	return nil
}

func (r *stubProduktionRepo) OrderNumberExists(_ context.Context, tenantID uuid.UUID, orderNumber string, excludeID *uuid.UUID) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	for _, o := range r.orders {
		if o.TenantID != tenantID || o.OrderNumber != orderNumber {
			continue
		}
		if excludeID != nil && o.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

// --- Machine Bookings ---

func (r *stubProduktionRepo) CreateBooking(_ context.Context, booking *produktion.MachineBooking) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.bookings[booking.ID] = booking
	return nil
}

func (r *stubProduktionRepo) UpdateBooking(_ context.Context, booking *produktion.MachineBooking) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.bookings[booking.ID] = booking
	return nil
}

func (r *stubProduktionRepo) GetBooking(_ context.Context, tenantID, bookingID uuid.UUID) (*produktion.MachineBooking, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	b, ok := r.bookings[bookingID]
	if !ok || b.TenantID != tenantID {
		return nil, produktion.ErrBookingNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *stubProduktionRepo) ListBookings(_ context.Context, tenantID uuid.UUID, filter produktion.ListBookingsFilter, offset, limit int) ([]*produktion.MachineBooking, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*produktion.MachineBooking
	for _, b := range r.bookings {
		if b.TenantID != tenantID {
			continue
		}
		if filter.MachineID != nil && b.MachineID != *filter.MachineID {
			continue
		}
		if filter.ProductionOrderID != nil && b.ProductionOrderID != *filter.ProductionOrderID {
			continue
		}
		matched = append(matched, b)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubProduktionRepo) DeleteBooking(_ context.Context, tenantID, bookingID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	b, ok := r.bookings[bookingID]
	if !ok || b.TenantID != tenantID {
		return produktion.ErrBookingNotFound
	}
	delete(r.bookings, bookingID)
	return nil
}

// findConflict mirrors PostgresRepository.FindConflictingBooking's half-open
// overlap arithmetic: existing.starts_at < new.ends_at AND existing.ends_at >
// new.starts_at, scoped to booked/in_use bookings on the same tenant+machine.
// Adjacent bookings (existing.ends_at == new.starts_at, or the reverse) are
// NOT a conflict — the interval is half-open.
func (r *stubProduktionRepo) findConflict(tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) *uuid.UUID {
	for _, b := range r.bookings {
		if b.TenantID != tenantID || b.MachineID != machineID {
			continue
		}
		if b.Status != produktion.BookingStatusBooked && b.Status != produktion.BookingStatusInUse {
			continue
		}
		if excludeID != nil && b.ID == *excludeID {
			continue
		}
		if b.StartsAt.Before(endsAt) && b.EndsAt.After(startsAt) {
			id := b.ID
			return &id
		}
	}
	return nil
}

func (r *stubProduktionRepo) FindConflictingBooking(_ context.Context, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.findConflict(tenantID, machineID, startsAt, endsAt, excludeID), nil
}

func (r *stubProduktionRepo) CreateBookingWithLock(_ context.Context, booking *produktion.MachineBooking) (*uuid.UUID, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	if conflictID := r.findConflict(booking.TenantID, booking.MachineID, booking.StartsAt, booking.EndsAt, nil); conflictID != nil {
		return conflictID, nil
	}
	r.bookings[booking.ID] = booking
	return nil, nil
}

// --- Plans ---

func (r *stubProduktionRepo) CreatePlan(_ context.Context, plan *produktion.ProductionPlan) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.plans[plan.ID] = plan
	return nil
}

func (r *stubProduktionRepo) UpdatePlan(_ context.Context, plan *produktion.ProductionPlan) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.plans[plan.ID] = plan
	return nil
}

func (r *stubProduktionRepo) GetPlan(_ context.Context, tenantID, planID uuid.UUID) (*produktion.ProductionPlan, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	p, ok := r.plans[planID]
	if !ok || p.TenantID != tenantID {
		return nil, produktion.ErrPlanNotFound
	}
	cp := *p
	return &cp, nil
}

// --- Capacity ---

func (r *stubProduktionRepo) GetCapacityOverview(_ context.Context, _ uuid.UUID, machineID string, _ uuid.UUID) (*produktion.CapacityOverview, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	if r.capacityOverview != nil {
		return r.capacityOverview, nil
	}
	return &produktion.CapacityOverview{MachineID: machineID}, nil
}

// --- BOMs ---

func (r *stubProduktionRepo) CreateBOM(_ context.Context, bom *produktion.BOM) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	for _, existing := range r.boms {
		if existing.TenantID == bom.TenantID && existing.SKU == bom.SKU {
			return produktion.ErrBOMSKUTaken
		}
	}
	r.boms[bom.ID] = bom
	return nil
}

func (r *stubProduktionRepo) UpdateBOM(_ context.Context, bom *produktion.BOM) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	for _, existing := range r.boms {
		if existing.ID != bom.ID && existing.TenantID == bom.TenantID && existing.SKU == bom.SKU {
			return produktion.ErrBOMSKUTaken
		}
	}
	r.boms[bom.ID] = bom
	return nil
}

func (r *stubProduktionRepo) GetBOM(_ context.Context, tenantID, bomID uuid.UUID) (*produktion.BOM, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	b, ok := r.boms[bomID]
	if !ok || b.TenantID != tenantID {
		return nil, produktion.ErrBOMNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *stubProduktionRepo) ListBOMs(_ context.Context, tenantID uuid.UUID, activeOnly *bool, offset, limit int) ([]*produktion.BOM, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*produktion.BOM
	for _, b := range r.boms {
		if b.TenantID != tenantID {
			continue
		}
		if activeOnly != nil && *activeOnly && !b.Active {
			continue
		}
		matched = append(matched, b)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubProduktionRepo) DeleteBOM(_ context.Context, tenantID, bomID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	b, ok := r.boms[bomID]
	if !ok || b.TenantID != tenantID {
		return produktion.ErrBOMNotFound
	}
	delete(r.boms, bomID)
	return nil
}

// --- Work Steps ---

func (r *stubProduktionRepo) CreateWorkStep(_ context.Context, step *produktion.WorkStep) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.steps[step.ID] = step
	return nil
}

func (r *stubProduktionRepo) UpdateWorkStep(_ context.Context, step *produktion.WorkStep) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.steps[step.ID] = step
	return nil
}

func (r *stubProduktionRepo) GetWorkStep(_ context.Context, tenantID, stepID uuid.UUID) (*produktion.WorkStep, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	st, ok := r.steps[stepID]
	if !ok || st.TenantID != tenantID {
		return nil, produktion.ErrWorkStepNotFound
	}
	cp := *st
	return &cp, nil
}

func (r *stubProduktionRepo) ListWorkSteps(_ context.Context, tenantID, orderID uuid.UUID) ([]*produktion.WorkStep, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var matched []*produktion.WorkStep
	for _, st := range r.steps {
		if st.TenantID == tenantID && st.OrderID == orderID {
			matched = append(matched, st)
		}
	}
	return matched, nil
}

func (r *stubProduktionRepo) DeleteWorkStep(_ context.Context, tenantID, stepID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	st, ok := r.steps[stepID]
	if !ok || st.TenantID != tenantID {
		return produktion.ErrWorkStepNotFound
	}
	delete(r.steps, stepID)
	return nil
}

// --- Machines ---

func (r *stubProduktionRepo) CreateMachine(_ context.Context, machine *produktion.Machine) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.machines[machine.ID] = machine
	return nil
}

func (r *stubProduktionRepo) UpdateMachine(_ context.Context, machine *produktion.Machine) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.machines[machine.ID] = machine
	return nil
}

func (r *stubProduktionRepo) GetMachine(_ context.Context, tenantID, machineID uuid.UUID) (*produktion.Machine, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	m, ok := r.machines[machineID]
	if !ok || m.TenantID != tenantID {
		return nil, produktion.ErrMachineNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *stubProduktionRepo) ListMachines(_ context.Context, tenantID uuid.UUID, status *produktion.MachineStatus, offset, limit int) ([]*produktion.Machine, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*produktion.Machine
	for _, m := range r.machines {
		if m.TenantID != tenantID {
			continue
		}
		if status != nil && m.Status != *status {
			continue
		}
		matched = append(matched, m)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubProduktionRepo) DeleteMachine(_ context.Context, tenantID, machineID uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	m, ok := r.machines[machineID]
	if !ok || m.TenantID != tenantID {
		return produktion.ErrMachineNotFound
	}
	delete(r.machines, machineID)
	return nil
}

// --- Quality Checks ---

func (r *stubProduktionRepo) CreateQualityCheck(_ context.Context, check *produktion.QualityCheck) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.checks[check.ID] = check
	return nil
}

func (r *stubProduktionRepo) GetQualityCheck(_ context.Context, tenantID, checkID uuid.UUID) (*produktion.QualityCheck, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	c, ok := r.checks[checkID]
	if !ok || c.TenantID != tenantID {
		return nil, produktion.ErrQualityCheckNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *stubProduktionRepo) ListQualityChecks(_ context.Context, tenantID uuid.UUID, orderID *uuid.UUID, offset, limit int) ([]*produktion.QualityCheck, int, error) {
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var matched []*produktion.QualityCheck
	for _, c := range r.checks {
		if c.TenantID != tenantID {
			continue
		}
		if orderID != nil && c.OrderID != *orderID {
			continue
		}
		matched = append(matched, c)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

// ---------------------------------------------------------------------------
// test server helpers
// ---------------------------------------------------------------------------

func newTestProduktionServer() *ProduktionGRPCServer {
	return NewProduktionGRPCServer(nil)
}

func newProduktionServerWithRepo(repo produktion.Repository) *ProduktionGRPCServer {
	return NewProduktionGRPCServer(produktion.NewService(repo))
}

// ---------------------------------------------------------------------------
// UUID / required-field validation — table test across every RPC
// ---------------------------------------------------------------------------

func TestProduktion_ValidationPaths(t *testing.T) {
	srv := newTestProduktionServer()
	ctx := context.Background()
	badID := "not-a-uuid"
	validTenant := uuid.New().String()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateOrder/tenant_id", func() error {
			_, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{TenantId: badID})
			return err
		}},
		{"CreateOrder/created_by", func() error {
			cb := badID
			_, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{TenantId: validTenant, CreatedBy: &cb})
			return err
		}},
		{"CreateOrder/bom_id", func() error {
			bomID := badID
			_, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{TenantId: validTenant, BomId: &bomID})
			return err
		}},
		{"UpdateOrder/tenant_id", func() error {
			_, err := srv.UpdateOrder(ctx, &produktionv1.UpdateOrderRequest{TenantId: badID})
			return err
		}},
		{"UpdateOrder/order_id", func() error {
			_, err := srv.UpdateOrder(ctx, &produktionv1.UpdateOrderRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"UpdateOrder/bom_id", func() error {
			bomID := badID
			_, err := srv.UpdateOrder(ctx, &produktionv1.UpdateOrderRequest{TenantId: validTenant, OrderId: uuid.New().String(), BomId: &bomID})
			return err
		}},
		{"DeleteOrder/tenant_id", func() error {
			_, err := srv.DeleteOrder(ctx, &produktionv1.DeleteOrderRequest{TenantId: badID})
			return err
		}},
		{"DeleteOrder/order_id", func() error {
			_, err := srv.DeleteOrder(ctx, &produktionv1.DeleteOrderRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"GetOrder/tenant_id", func() error {
			_, err := srv.GetOrder(ctx, &produktionv1.GetOrderRequest{TenantId: badID})
			return err
		}},
		{"GetOrder/order_id", func() error {
			_, err := srv.GetOrder(ctx, &produktionv1.GetOrderRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"ListOrders/tenant_id", func() error {
			_, err := srv.ListOrders(ctx, &produktionv1.ListOrdersRequest{TenantId: badID})
			return err
		}},
		{"StartOrder/tenant_id", func() error {
			_, err := srv.StartOrder(ctx, &produktionv1.OrderActionRequest{TenantId: badID})
			return err
		}},
		{"StartOrder/order_id", func() error {
			_, err := srv.StartOrder(ctx, &produktionv1.OrderActionRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"CompleteOrder/tenant_id", func() error {
			_, err := srv.CompleteOrder(ctx, &produktionv1.OrderActionRequest{TenantId: badID})
			return err
		}},
		{"CompleteOrder/order_id", func() error {
			_, err := srv.CompleteOrder(ctx, &produktionv1.OrderActionRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"CancelOrder/tenant_id", func() error {
			_, err := srv.CancelOrder(ctx, &produktionv1.OrderActionRequest{TenantId: badID})
			return err
		}},
		{"CancelOrder/order_id", func() error {
			_, err := srv.CancelOrder(ctx, &produktionv1.OrderActionRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"CreateMachineBooking/tenant_id", func() error {
			_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{TenantId: badID})
			return err
		}},
		{"CreateMachineBooking/production_order_id", func() error {
			_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{TenantId: validTenant, ProductionOrderId: badID})
			return err
		}},
		{"CreateMachineBooking/created_by", func() error {
			cb := badID
			_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{TenantId: validTenant, ProductionOrderId: uuid.New().String(), CreatedBy: &cb})
			return err
		}},
		{"UpdateMachineBooking/tenant_id", func() error {
			_, err := srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{TenantId: badID})
			return err
		}},
		{"UpdateMachineBooking/booking_id", func() error {
			_, err := srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{TenantId: validTenant, BookingId: badID})
			return err
		}},
		{"UpdateMachineBooking/production_order_id", func() error {
			poID := badID
			_, err := srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{TenantId: validTenant, BookingId: uuid.New().String(), ProductionOrderId: &poID})
			return err
		}},
		{"DeleteMachineBooking/tenant_id", func() error {
			_, err := srv.DeleteMachineBooking(ctx, &produktionv1.DeleteMachineBookingRequest{TenantId: badID})
			return err
		}},
		{"DeleteMachineBooking/booking_id", func() error {
			_, err := srv.DeleteMachineBooking(ctx, &produktionv1.DeleteMachineBookingRequest{TenantId: validTenant, BookingId: badID})
			return err
		}},
		{"ListMachineBookings/tenant_id", func() error {
			_, err := srv.ListMachineBookings(ctx, &produktionv1.ListMachineBookingsRequest{TenantId: badID})
			return err
		}},
		{"ListMachineBookings/production_order_id", func() error {
			poID := badID
			_, err := srv.ListMachineBookings(ctx, &produktionv1.ListMachineBookingsRequest{TenantId: validTenant, ProductionOrderId: &poID})
			return err
		}},
		{"CreatePlan/tenant_id", func() error {
			_, err := srv.CreatePlan(ctx, &produktionv1.CreatePlanRequest{TenantId: badID})
			return err
		}},
		{"CreatePlan/created_by", func() error {
			cb := badID
			_, err := srv.CreatePlan(ctx, &produktionv1.CreatePlanRequest{TenantId: validTenant, CreatedBy: &cb})
			return err
		}},
		{"UpdatePlan/tenant_id", func() error {
			_, err := srv.UpdatePlan(ctx, &produktionv1.UpdatePlanRequest{TenantId: badID})
			return err
		}},
		{"UpdatePlan/plan_id", func() error {
			_, err := srv.UpdatePlan(ctx, &produktionv1.UpdatePlanRequest{TenantId: validTenant, PlanId: badID})
			return err
		}},
		{"GetPlan/tenant_id", func() error {
			_, err := srv.GetPlan(ctx, &produktionv1.GetPlanRequest{TenantId: badID})
			return err
		}},
		{"GetPlan/plan_id", func() error {
			_, err := srv.GetPlan(ctx, &produktionv1.GetPlanRequest{TenantId: validTenant, PlanId: badID})
			return err
		}},
		{"GetCapacityOverview/tenant_id", func() error {
			_, err := srv.GetCapacityOverview(ctx, &produktionv1.GetCapacityOverviewRequest{TenantId: badID})
			return err
		}},
		{"GetCapacityOverview/plan_id", func() error {
			_, err := srv.GetCapacityOverview(ctx, &produktionv1.GetCapacityOverviewRequest{TenantId: validTenant, PlanId: badID})
			return err
		}},
		{"CreateBOM/tenant_id", func() error {
			_, err := srv.CreateBOM(ctx, &produktionv1.CreateBOMRequest{TenantId: badID})
			return err
		}},
		{"CreateBOM/created_by", func() error {
			cb := badID
			_, err := srv.CreateBOM(ctx, &produktionv1.CreateBOMRequest{TenantId: validTenant, CreatedBy: &cb})
			return err
		}},
		{"UpdateBOM/tenant_id", func() error {
			_, err := srv.UpdateBOM(ctx, &produktionv1.UpdateBOMRequest{TenantId: badID})
			return err
		}},
		{"UpdateBOM/bom_id", func() error {
			_, err := srv.UpdateBOM(ctx, &produktionv1.UpdateBOMRequest{TenantId: validTenant, BomId: badID})
			return err
		}},
		{"DeleteBOM/tenant_id", func() error {
			_, err := srv.DeleteBOM(ctx, &produktionv1.DeleteBOMRequest{TenantId: badID})
			return err
		}},
		{"DeleteBOM/bom_id", func() error {
			_, err := srv.DeleteBOM(ctx, &produktionv1.DeleteBOMRequest{TenantId: validTenant, BomId: badID})
			return err
		}},
		{"GetBOM/tenant_id", func() error {
			_, err := srv.GetBOM(ctx, &produktionv1.GetBOMRequest{TenantId: badID})
			return err
		}},
		{"GetBOM/bom_id", func() error {
			_, err := srv.GetBOM(ctx, &produktionv1.GetBOMRequest{TenantId: validTenant, BomId: badID})
			return err
		}},
		{"ListBOMs/tenant_id", func() error {
			_, err := srv.ListBOMs(ctx, &produktionv1.ListBOMsRequest{TenantId: badID})
			return err
		}},
		{"CreateWorkStep/tenant_id", func() error {
			_, err := srv.CreateWorkStep(ctx, &produktionv1.CreateWorkStepRequest{TenantId: badID})
			return err
		}},
		{"CreateWorkStep/order_id", func() error {
			_, err := srv.CreateWorkStep(ctx, &produktionv1.CreateWorkStepRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"UpdateWorkStep/tenant_id", func() error {
			_, err := srv.UpdateWorkStep(ctx, &produktionv1.UpdateWorkStepRequest{TenantId: badID})
			return err
		}},
		{"UpdateWorkStep/step_id", func() error {
			_, err := srv.UpdateWorkStep(ctx, &produktionv1.UpdateWorkStepRequest{TenantId: validTenant, StepId: badID})
			return err
		}},
		{"DeleteWorkStep/tenant_id", func() error {
			_, err := srv.DeleteWorkStep(ctx, &produktionv1.DeleteWorkStepRequest{TenantId: badID})
			return err
		}},
		{"DeleteWorkStep/step_id", func() error {
			_, err := srv.DeleteWorkStep(ctx, &produktionv1.DeleteWorkStepRequest{TenantId: validTenant, StepId: badID})
			return err
		}},
		{"ListWorkSteps/tenant_id", func() error {
			_, err := srv.ListWorkSteps(ctx, &produktionv1.ListWorkStepsRequest{TenantId: badID})
			return err
		}},
		{"ListWorkSteps/order_id", func() error {
			_, err := srv.ListWorkSteps(ctx, &produktionv1.ListWorkStepsRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"CreateMachine/tenant_id", func() error {
			_, err := srv.CreateMachine(ctx, &produktionv1.CreateMachineRequest{TenantId: badID})
			return err
		}},
		{"CreateMachine/created_by", func() error {
			cb := badID
			_, err := srv.CreateMachine(ctx, &produktionv1.CreateMachineRequest{TenantId: validTenant, CreatedBy: &cb})
			return err
		}},
		{"UpdateMachine/tenant_id", func() error {
			_, err := srv.UpdateMachine(ctx, &produktionv1.UpdateMachineRequest{TenantId: badID})
			return err
		}},
		{"UpdateMachine/machine_id", func() error {
			_, err := srv.UpdateMachine(ctx, &produktionv1.UpdateMachineRequest{TenantId: validTenant, MachineId: badID})
			return err
		}},
		{"DeleteMachine/tenant_id", func() error {
			_, err := srv.DeleteMachine(ctx, &produktionv1.DeleteMachineRequest{TenantId: badID})
			return err
		}},
		{"DeleteMachine/machine_id", func() error {
			_, err := srv.DeleteMachine(ctx, &produktionv1.DeleteMachineRequest{TenantId: validTenant, MachineId: badID})
			return err
		}},
		{"GetMachine/tenant_id", func() error {
			_, err := srv.GetMachine(ctx, &produktionv1.GetMachineRequest{TenantId: badID})
			return err
		}},
		{"GetMachine/machine_id", func() error {
			_, err := srv.GetMachine(ctx, &produktionv1.GetMachineRequest{TenantId: validTenant, MachineId: badID})
			return err
		}},
		{"ListMachines/tenant_id", func() error {
			_, err := srv.ListMachines(ctx, &produktionv1.ListMachinesRequest{TenantId: badID})
			return err
		}},
		{"CreateQualityCheck/tenant_id", func() error {
			_, err := srv.CreateQualityCheck(ctx, &produktionv1.CreateQualityCheckRequest{TenantId: badID})
			return err
		}},
		{"CreateQualityCheck/order_id", func() error {
			_, err := srv.CreateQualityCheck(ctx, &produktionv1.CreateQualityCheckRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
		{"CreateQualityCheck/created_by", func() error {
			cb := badID
			_, err := srv.CreateQualityCheck(ctx, &produktionv1.CreateQualityCheckRequest{TenantId: validTenant, OrderId: uuid.New().String(), CreatedBy: &cb})
			return err
		}},
		{"GetQualityCheck/tenant_id", func() error {
			_, err := srv.GetQualityCheck(ctx, &produktionv1.GetQualityCheckRequest{TenantId: badID})
			return err
		}},
		{"GetQualityCheck/check_id", func() error {
			_, err := srv.GetQualityCheck(ctx, &produktionv1.GetQualityCheckRequest{TenantId: validTenant, CheckId: badID})
			return err
		}},
		{"ListQualityChecks/tenant_id", func() error {
			_, err := srv.ListQualityChecks(ctx, &produktionv1.ListQualityChecksRequest{TenantId: badID})
			return err
		}},
		{"ListQualityChecks/order_id", func() error {
			orderID := badID
			_, err := srv.ListQualityChecks(ctx, &produktionv1.ListQualityChecksRequest{TenantId: validTenant, OrderId: &orderID})
			return err
		}},
		{"GetMaterialAvailability/tenant_id", func() error {
			_, err := srv.GetMaterialAvailability(ctx, &produktionv1.GetMaterialAvailabilityRequest{TenantId: badID})
			return err
		}},
		{"GetMaterialAvailability/order_id", func() error {
			_, err := srv.GetMaterialAvailability(ctx, &produktionv1.GetMaterialAvailabilityRequest{TenantId: validTenant, OrderId: badID})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), codes.InvalidArgument)
		})
	}
}

// ---------------------------------------------------------------------------
// Order lifecycle
// ---------------------------------------------------------------------------

func TestProduktion_Order_CreateGetUpdateDelete(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     10,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.Order)
	assert.Equal(t, "planned", createResp.Order.Status)
	orderID := createResp.Order.Id

	// duplicate order number -> AlreadyExists, not Internal.
	_, err = srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Other",
		Quantity:     1,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)

	// referencing a nonexistent BOM propagates ErrBOMNotFound -> NotFound.
	_, err = srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-2",
		ProductName:  "Widget",
		Quantity:     1,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
		BomId:        strPtr(uuid.New().String()),
	})
	requireGRPCCode(t, err, codes.NotFound)

	getResp, err := srv.GetOrder(ctx, &produktionv1.GetOrderRequest{TenantId: tenantID, OrderId: orderID})
	require.NoError(t, err)
	assert.Equal(t, "Widget", getResp.Order.ProductName)

	newName := "Widget Pro"
	updResp, err := srv.UpdateOrder(ctx, &produktionv1.UpdateOrderRequest{
		TenantId:    tenantID,
		OrderId:     orderID,
		ProductName: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Widget Pro", updResp.Order.ProductName)

	// UpdateOrder to a nonexistent order -> NotFound.
	_, err = srv.UpdateOrder(ctx, &produktionv1.UpdateOrderRequest{TenantId: tenantID, OrderId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.DeleteOrder(ctx, &produktionv1.DeleteOrderRequest{TenantId: tenantID, OrderId: orderID})
	require.NoError(t, err)

	_, err = srv.GetOrder(ctx, &produktionv1.GetOrderRequest{TenantId: tenantID, OrderId: orderID})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestProduktion_Order_StatusTransitions(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     10,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	orderID := createResp.Order.Id

	// CompleteOrder on a planned (not in_progress) order is an invalid transition.
	_, err = srv.CompleteOrder(ctx, &produktionv1.OrderActionRequest{TenantId: tenantID, OrderId: orderID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// DeleteOrder is blocked once the order is no longer planned — verified below
	// after StartOrder, so first confirm planned-status delete works, then re-create.
	startResp, err := srv.StartOrder(ctx, &produktionv1.OrderActionRequest{TenantId: tenantID, OrderId: orderID})
	require.NoError(t, err)
	assert.Equal(t, "in_progress", startResp.Order.Status)
	require.NotNil(t, startResp.Order.ActualStart)

	// StartOrder again (already in_progress) -> invalid transition.
	_, err = srv.StartOrder(ctx, &produktionv1.OrderActionRequest{TenantId: tenantID, OrderId: orderID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// order is no longer planned -> not deletable.
	_, err = srv.DeleteOrder(ctx, &produktionv1.DeleteOrderRequest{TenantId: tenantID, OrderId: orderID})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	completeResp, err := srv.CompleteOrder(ctx, &produktionv1.OrderActionRequest{TenantId: tenantID, OrderId: orderID})
	require.NoError(t, err)
	assert.Equal(t, "completed", completeResp.Order.Status)
	require.NotNil(t, completeResp.Order.ActualEnd)

	// CancelOrder on a completed order -> invalid transition.
	_, err = srv.CancelOrder(ctx, &produktionv1.OrderActionRequest{TenantId: tenantID, OrderId: orderID})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestProduktion_ListOrders_PaginationAndTotal(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	for range 3 {
		_, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
			TenantId:     tenantID,
			OrderNumber:  uuid.New().String(),
			ProductName:  "Widget",
			Quantity:     1,
			PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
			PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
		})
		require.NoError(t, err)
	}

	listResp, err := srv.ListOrders(ctx, &produktionv1.ListOrdersRequest{TenantId: tenantID, Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, listResp.Orders, 2)
	assert.EqualValues(t, 3, listResp.Total)
}

// ---------------------------------------------------------------------------
// Machine Bookings — conflict detection is the core done_when of this unit.
// ---------------------------------------------------------------------------

func TestProduktion_CreateMachineBooking_ConflictMapsToFailedPrecondition(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	// overlaps [08:00, 10:00) at 09:00-11:00 -> conflict, must NOT be codes.Internal.
	_, err = srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestProduktion_CreateMachineBooking_AdjacentIsNotAConflict(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	// starts exactly when the first booking ends -> half-open interval, no conflict.
	resp, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Booking)

	// a different machine is never a conflict, even at the exact same slot.
	resp2, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-2",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.NotNil(t, resp2.Booking)
}

func TestProduktion_CreateMachineBooking_InvalidInput(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	// empty machine_id -> InvalidArgument.
	_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "  ",
		ProductionOrderId: uuid.New().String(),
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	// ends_at not after starts_at -> InvalidArgument.
	_, err = srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: uuid.New().String(),
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestProduktion_UpdateMachineBooking_ConflictAndNotFound(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	first, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	second, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	// moving the second booking into the first one's window -> conflict.
	newStart := timestamppb.New(time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	newEnd := timestamppb.New(time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC))
	_, err = srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{
		TenantId:  tenantID,
		BookingId: second.Booking.Id,
		StartsAt:  newStart,
		EndsAt:    newEnd,
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// updating the first booking to its own unchanged slot must not conflict with itself.
	sameStart := timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC))
	sameEnd := timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	_, err = srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{
		TenantId:  tenantID,
		BookingId: first.Booking.Id,
		StartsAt:  sameStart,
		EndsAt:    sameEnd,
	})
	require.NoError(t, err)

	_, err = srv.UpdateMachineBooking(ctx, &produktionv1.UpdateMachineBookingRequest{TenantId: tenantID, BookingId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestProduktion_ListMachineBookings(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	_, err := srv.CreateMachineBooking(ctx, &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID,
		MachineId:         "cnc-1",
		ProductionOrderId: orderID,
		StartsAt:          timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		EndsAt:            timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	listResp, err := srv.ListMachineBookings(ctx, &produktionv1.ListMachineBookingsRequest{TenantId: tenantID, MachineId: strPtr("cnc-1")})
	require.NoError(t, err)
	require.Len(t, listResp.Bookings, 1)
	assert.EqualValues(t, 1, listResp.Total)

	_, err = srv.DeleteMachineBooking(ctx, &produktionv1.DeleteMachineBookingRequest{TenantId: tenantID, BookingId: listResp.Bookings[0].Id})
	require.NoError(t, err)

	_, err = srv.DeleteMachineBooking(ctx, &produktionv1.DeleteMachineBookingRequest{TenantId: tenantID, BookingId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// Plans + Capacity Overview
// ---------------------------------------------------------------------------

func TestProduktion_Plan_CreateUpdateGet(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreatePlan(ctx, &produktionv1.CreatePlanRequest{
		TenantId:           tenantID,
		Name:               "KW 1",
		WeekNumber:         1,
		Year:               2026,
		TotalCapacityHours: 160,
	})
	require.NoError(t, err)
	assert.Equal(t, "draft", createResp.Plan.Status)

	_, err = srv.GetPlan(ctx, &produktionv1.GetPlanRequest{TenantId: tenantID, PlanId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)

	newName := "KW 1 revised"
	updResp, err := srv.UpdatePlan(ctx, &produktionv1.UpdatePlanRequest{TenantId: tenantID, PlanId: createResp.Plan.Id, Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "KW 1 revised", updResp.Plan.Name)
}

func TestProduktion_GetCapacityOverview(t *testing.T) {
	repo := newStubProduktionRepo()
	repo.capacityOverview = &produktion.CapacityOverview{
		MachineID:          "cnc-1",
		TotalCapacityHours: 40,
		BookedHours:        30,
		AvailableHours:     10,
	}
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	resp, err := srv.GetCapacityOverview(ctx, &produktionv1.GetCapacityOverviewRequest{
		TenantId:  tenantID,
		MachineId: "cnc-1",
		PlanId:    uuid.New().String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "cnc-1", resp.Overview.MachineId)
	assert.InDelta(t, 10.0, resp.Overview.AvailableHours, 0.0001)

	// blank machine_id -> ErrInvalidInput -> InvalidArgument, checked at the service layer.
	_, err = srv.GetCapacityOverview(ctx, &produktionv1.GetCapacityOverviewRequest{
		TenantId:  tenantID,
		MachineId: "  ",
		PlanId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// BOMs
// ---------------------------------------------------------------------------

func TestProduktion_BOM_CreateGetUpdateListDelete(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateBOM(ctx, &produktionv1.CreateBOMRequest{
		TenantId:    tenantID,
		ProductName: "Widget",
		Sku:         "SKU-1",
		Active:      true,
		Items: []*produktionv1.CreateBomItemInput{
			{MaterialName: "Steel", Quantity: 2.5, Unit: "kg", SortOrder: 5},
		},
	})
	require.NoError(t, err)
	require.Len(t, createResp.Bom.Items, 1)
	assert.Equal(t, "Steel", createResp.Bom.Items[0].MaterialName)
	// sort order is re-derived from insertion index in the service, not passed through.
	assert.EqualValues(t, 0, createResp.Bom.Items[0].SortOrder)

	// duplicate SKU for the same tenant -> AlreadyExists, not Internal.
	_, err = srv.CreateBOM(ctx, &produktionv1.CreateBOMRequest{TenantId: tenantID, ProductName: "Other", Sku: "SKU-1"})
	requireGRPCCode(t, err, codes.AlreadyExists)

	getResp, err := srv.GetBOM(ctx, &produktionv1.GetBOMRequest{TenantId: tenantID, BomId: createResp.Bom.Id})
	require.NoError(t, err)
	assert.Equal(t, "1.0", getResp.Bom.Version)

	listResp, err := srv.ListBOMs(ctx, &produktionv1.ListBOMsRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, listResp.Total)

	newSKU := "SKU-2"
	updResp, err := srv.UpdateBOM(ctx, &produktionv1.UpdateBOMRequest{TenantId: tenantID, BomId: createResp.Bom.Id, Sku: &newSKU})
	require.NoError(t, err)
	assert.Equal(t, "SKU-2", updResp.Bom.Sku)

	_, err = srv.DeleteBOM(ctx, &produktionv1.DeleteBOMRequest{TenantId: tenantID, BomId: createResp.Bom.Id})
	require.NoError(t, err)

	_, err = srv.GetBOM(ctx, &produktionv1.GetBOMRequest{TenantId: tenantID, BomId: createResp.Bom.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// Work Steps
// ---------------------------------------------------------------------------

func TestProduktion_WorkStep_CreateUpdateListDelete(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	createResp, err := srv.CreateWorkStep(ctx, &produktionv1.CreateWorkStepRequest{
		TenantId: tenantID,
		OrderId:  orderID,
		StepNr:   1,
		Name:     "Assembly",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", createResp.Step.Status)

	// empty name -> InvalidArgument.
	_, err = srv.CreateWorkStep(ctx, &produktionv1.CreateWorkStepRequest{TenantId: tenantID, OrderId: orderID, Name: "  "})
	requireGRPCCode(t, err, codes.InvalidArgument)

	listResp, err := srv.ListWorkSteps(ctx, &produktionv1.ListWorkStepsRequest{TenantId: tenantID, OrderId: orderID})
	require.NoError(t, err)
	require.Len(t, listResp.Steps, 1)
	assert.EqualValues(t, 1, listResp.Total)

	newStatus := "completed"
	updResp, err := srv.UpdateWorkStep(ctx, &produktionv1.UpdateWorkStepRequest{TenantId: tenantID, StepId: createResp.Step.Id, Status: &newStatus})
	require.NoError(t, err)
	assert.Equal(t, "completed", updResp.Step.Status)

	_, err = srv.UpdateWorkStep(ctx, &produktionv1.UpdateWorkStepRequest{TenantId: tenantID, StepId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.DeleteWorkStep(ctx, &produktionv1.DeleteWorkStepRequest{TenantId: tenantID, StepId: createResp.Step.Id})
	require.NoError(t, err)

	_, err = srv.DeleteWorkStep(ctx, &produktionv1.DeleteWorkStepRequest{TenantId: tenantID, StepId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// Machines
// ---------------------------------------------------------------------------

func TestProduktion_Machine_CreateGetUpdateListDelete(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateMachine(ctx, &produktionv1.CreateMachineRequest{TenantId: tenantID, Name: "CNC 1", Type: "cnc"})
	require.NoError(t, err)
	assert.Equal(t, "available", createResp.Machine.Status)

	_, err = srv.CreateMachine(ctx, &produktionv1.CreateMachineRequest{TenantId: tenantID, Name: "  "})
	requireGRPCCode(t, err, codes.InvalidArgument)

	getResp, err := srv.GetMachine(ctx, &produktionv1.GetMachineRequest{TenantId: tenantID, MachineId: createResp.Machine.Id})
	require.NoError(t, err)
	assert.Equal(t, "CNC 1", getResp.Machine.Name)

	newStatus := "maintenance"
	updResp, err := srv.UpdateMachine(ctx, &produktionv1.UpdateMachineRequest{TenantId: tenantID, MachineId: createResp.Machine.Id, Status: &newStatus})
	require.NoError(t, err)
	assert.Equal(t, "maintenance", updResp.Machine.Status)

	listResp, err := srv.ListMachines(ctx, &produktionv1.ListMachinesRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, listResp.Total)

	_, err = srv.DeleteMachine(ctx, &produktionv1.DeleteMachineRequest{TenantId: tenantID, MachineId: createResp.Machine.Id})
	require.NoError(t, err)

	_, err = srv.GetMachine(ctx, &produktionv1.GetMachineRequest{TenantId: tenantID, MachineId: createResp.Machine.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// Quality Checks
// ---------------------------------------------------------------------------

func TestProduktion_QualityCheck_CreateGetList(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	createResp, err := srv.CreateQualityCheck(ctx, &produktionv1.CreateQualityCheckRequest{
		TenantId:     tenantID,
		OrderId:      orderID,
		Inspector:    "Jane Doe",
		CheckedAt:    timestamppb.New(time.Now()),
		Passed:       false,
		DefectsFound: 3,
	})
	require.NoError(t, err)
	assert.False(t, createResp.Check.Passed)
	assert.EqualValues(t, 3, createResp.Check.DefectsFound)

	// empty inspector -> InvalidArgument.
	_, err = srv.CreateQualityCheck(ctx, &produktionv1.CreateQualityCheckRequest{TenantId: tenantID, OrderId: orderID, Inspector: "  "})
	requireGRPCCode(t, err, codes.InvalidArgument)

	getResp, err := srv.GetQualityCheck(ctx, &produktionv1.GetQualityCheckRequest{TenantId: tenantID, CheckId: createResp.Check.Id})
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", getResp.Check.Inspector)

	_, err = srv.GetQualityCheck(ctx, &produktionv1.GetQualityCheckRequest{TenantId: tenantID, CheckId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)

	listResp, err := srv.ListQualityChecks(ctx, &produktionv1.ListQualityChecksRequest{TenantId: tenantID, OrderId: strPtr(orderID)})
	require.NoError(t, err)
	assert.EqualValues(t, 1, listResp.Total)
}

// ---------------------------------------------------------------------------
// Material Availability
// ---------------------------------------------------------------------------

func TestProduktion_GetMaterialAvailability_NoBOM(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	createResp, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     1,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	// order has no linked BOM -> ErrOrderHasNoBOM -> FailedPrecondition, not Internal.
	_, err = srv.GetMaterialAvailability(ctx, &produktionv1.GetMaterialAvailabilityRequest{TenantId: tenantID, OrderId: createResp.Order.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestProduktion_GetMaterialAvailability_WithBOM(t *testing.T) {
	repo := newStubProduktionRepo()
	srv := newProduktionServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	bomResp, err := srv.CreateBOM(ctx, &produktionv1.CreateBOMRequest{
		TenantId:    tenantID,
		ProductName: "Widget",
		Sku:         "SKU-1",
		Items: []*produktionv1.CreateBomItemInput{
			{MaterialName: "Steel", Quantity: 2, Unit: "kg"},
		},
	})
	require.NoError(t, err)

	orderResp, err := srv.CreateOrder(ctx, &produktionv1.CreateOrderRequest{
		TenantId:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     5,
		PlannedStart: timestamppb.New(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)),
		PlannedEnd:   timestamppb.New(time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)),
		BomId:        &bomResp.Bom.Id,
	})
	require.NoError(t, err)

	availResp, err := srv.GetMaterialAvailability(ctx, &produktionv1.GetMaterialAvailabilityRequest{TenantId: tenantID, OrderId: orderResp.Order.Id})
	require.NoError(t, err)
	require.Len(t, availResp.Lines, 1)
	assert.Equal(t, "Steel", availResp.Lines[0].MaterialName)
	// no InventarLookup configured on this service -> availability stays unknown, not zero.
	assert.Nil(t, availResp.Lines[0].AvailableQuantity)
	assert.InDelta(t, 10.0, availResp.Lines[0].RequiredQuantity, 0.0001) // 2 * quantity(5)
}

// ---------------------------------------------------------------------------
// Error mapping — table tests
// ---------------------------------------------------------------------------

func TestMapProduktionError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"order_not_found", produktion.ErrOrderNotFound, codes.NotFound},
		{"booking_not_found", produktion.ErrBookingNotFound, codes.NotFound},
		{"plan_not_found", produktion.ErrPlanNotFound, codes.NotFound},
		{"bom_not_found", produktion.ErrBOMNotFound, codes.NotFound},
		{"order_has_no_bom", produktion.ErrOrderHasNoBOM, codes.FailedPrecondition},
		{"order_number_taken", produktion.ErrOrderNumberTaken, codes.AlreadyExists},
		{"order_not_deletable", produktion.ErrOrderNotDeletable, codes.FailedPrecondition},
		{"order_invalid_transition", produktion.ErrOrderInvalidTransition, codes.FailedPrecondition},
		{"booking_conflict", produktion.ErrBookingConflict, codes.FailedPrecondition},
		{"invalid_input", produktion.ErrInvalidInput, codes.InvalidArgument},
		{"generic_fallback", errStubProduktionFailure, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapProduktionError(tc.err), tc.code)
		})
	}
	assert.NoError(t, mapProduktionError(nil))
}

func TestMapProduktionExtError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"bom_not_found", produktion.ErrBOMNotFound, codes.NotFound},
		{"bom_sku_taken", produktion.ErrBOMSKUTaken, codes.AlreadyExists},
		{"work_step_not_found", produktion.ErrWorkStepNotFound, codes.NotFound},
		{"machine_not_found", produktion.ErrMachineNotFound, codes.NotFound},
		{"quality_check_not_found", produktion.ErrQualityCheckNotFound, codes.NotFound},
		// falls through to mapProduktionError for the base sentinel set.
		{"falls_through_booking_conflict", produktion.ErrBookingConflict, codes.FailedPrecondition},
		{"generic_fallback", errStubProduktionFailure, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapProduktionExtError(tc.err), tc.code)
		})
	}
	assert.NoError(t, mapProduktionExtError(nil))
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func TestProduktion_ToProtoConversions_NilInputs(t *testing.T) {
	assert.Nil(t, orderToProto(nil))
	assert.Nil(t, machineBookingToProto(nil))
	assert.Nil(t, planToProto(nil))
	assert.Nil(t, bomToProto(nil))
	assert.Nil(t, workStepToProto(nil))
	assert.Nil(t, machineToProto(nil))
	assert.Nil(t, qualityCheckToProto(nil))
	assert.Nil(t, materialAvailabilityToProto(nil))
}

func TestProduktion_OrderToProto_OptionalFields(t *testing.T) {
	createdBy := uuid.New()
	bomID := uuid.New()
	now := time.Now()

	o := &produktion.ProductionOrder{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     10,
		Status:       produktion.OrderStatusInProgress,
		PlannedStart: now,
		PlannedEnd:   now.Add(24 * time.Hour),
		ActualStart:  &now,
		ActualEnd:    &now,
		CreatedBy:    &createdBy,
		BomID:        &bomID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	p := orderToProto(o)
	require.NotNil(t, p)
	require.NotNil(t, p.ActualStart)
	require.NotNil(t, p.ActualEnd)
	require.NotNil(t, p.CreatedBy)
	require.NotNil(t, p.BomId)
	assert.Equal(t, createdBy.String(), *p.CreatedBy)
	assert.Equal(t, bomID.String(), *p.BomId)

	// without the optional pointer fields the proto fields must stay unset.
	minimal := &produktion.ProductionOrder{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		OrderNumber:  "PO-2",
		ProductName:  "Widget",
		Quantity:     1,
		Status:       produktion.OrderStatusPlanned,
		PlannedStart: now,
		PlannedEnd:   now.Add(time.Hour),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	pm := orderToProto(minimal)
	assert.Nil(t, pm.ActualStart)
	assert.Nil(t, pm.ActualEnd)
	assert.Nil(t, pm.CreatedBy)
	assert.Nil(t, pm.BomId)
}

func TestProduktion_MaterialAvailabilityToProto_OptionalLines(t *testing.T) {
	available := 5.0
	shortfall := 2.0
	a := &produktion.MaterialAvailability{
		OrderID: uuid.New(),
		BomID:   uuid.New(),
		Lines: []produktion.MaterialAvailabilityLine{
			{MaterialName: "Steel", Unit: "kg", RequiredQuantity: 7, AvailableQuantity: &available, ShortfallQuantity: &shortfall},
			{MaterialName: "Glue", Unit: "l", RequiredQuantity: 1}, // unresolved -> both nil
		},
	}
	resp := materialAvailabilityToProto(a)
	require.NotNil(t, resp)
	require.Len(t, resp.Lines, 2)
	require.NotNil(t, resp.Lines[0].AvailableQuantity)
	require.NotNil(t, resp.Lines[0].ShortfallQuantity)
	assert.InDelta(t, 5.0, *resp.Lines[0].AvailableQuantity, 0.0001)
	assert.Nil(t, resp.Lines[1].AvailableQuantity)
	assert.Nil(t, resp.Lines[1].ShortfallQuantity)
}
