package server

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/models"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// ============================================================================
// Stub repositories for leave.Service (server-package copies, mirroring the
// stubFormulareRepo pattern in formulare_grpc_test.go). ctxWithTenant is
// reused from dialer_grpc_test.go.
// ============================================================================

type stubLeaveRequestRepo struct {
	mu       sync.Mutex
	requests map[uuid.UUID]*models.LeaveRequest
}

func newStubLeaveRequestRepo() *stubLeaveRequestRepo {
	return &stubLeaveRequestRepo{requests: make(map[uuid.UUID]*models.LeaveRequest)}
}

func (r *stubLeaveRequestRepo) put(req *models.LeaveRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.requests[req.ID] = &cp
}

func (r *stubLeaveRequestRepo) Create(_ context.Context, req *models.LeaveRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.requests[req.ID] = &cp
	return nil
}

func (r *stubLeaveRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*models.LeaveRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.requests[id]
	if !ok {
		return nil, leave.ErrLeaveRequestNotFound
	}
	cp := *req
	return &cp, nil
}

func (r *stubLeaveRequestRepo) List(_ context.Context, filter leave.LeaveRequestFilter) ([]*models.LeaveRequest, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []*models.LeaveRequest
	for _, req := range r.requests {
		if req.TenantID != filter.TenantID {
			continue
		}
		if filter.EmployeeID != nil && req.EmployeeID != *filter.EmployeeID {
			continue
		}
		if filter.Status != nil && string(req.Status) != *filter.Status {
			continue
		}
		if filter.StartDateFrom != nil && req.StartDate.Before(*filter.StartDateFrom) {
			continue
		}
		if filter.StartDateTo != nil && req.StartDate.After(*filter.StartDateTo) {
			continue
		}
		cp := *req
		all = append(all, &cp)
	}
	return all, len(all), nil
}

func (r *stubLeaveRequestRepo) Update(_ context.Context, req *models.LeaveRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.requests[req.ID]; !ok {
		return leave.ErrLeaveRequestNotFound
	}
	cp := *req
	r.requests[req.ID] = &cp
	return nil
}

func (r *stubLeaveRequestRepo) FindOverlaps(_ context.Context, employeeID uuid.UUID, startDate, endDate time.Time, excludeID *uuid.UUID) ([]*models.LeaveRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.LeaveRequest
	for _, req := range r.requests {
		if req.EmployeeID != employeeID {
			continue
		}
		if excludeID != nil && req.ID == *excludeID {
			continue
		}
		if req.StartDate.After(endDate) || req.EndDate.Before(startDate) {
			continue
		}
		cp := *req
		out = append(out, &cp)
	}
	return out, nil
}

type stubLeaveBalanceRepo struct {
	mu       sync.Mutex
	balances map[string]*models.HRLeaveBalance
}

func newStubLeaveBalanceRepo() *stubLeaveBalanceRepo {
	return &stubLeaveBalanceRepo{balances: make(map[string]*models.HRLeaveBalance)}
}

func leaveBalanceKey(tenantID, employeeID uuid.UUID, year int) string {
	return tenantID.String() + "|" + employeeID.String() + "|" + strconv.Itoa(year)
}

func (r *stubLeaveBalanceRepo) GetByEmployeeYear(_ context.Context, tenantID, employeeID uuid.UUID, year int) (*models.HRLeaveBalance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.balances[leaveBalanceKey(tenantID, employeeID, year)]
	if !ok {
		// Mirrors PostgresLeaveBalanceRepo.GetByEmployeeYear: no row = not found, not an error.
		return nil, nil
	}
	cp := *b
	return &cp, nil
}

func (r *stubLeaveBalanceRepo) Upsert(_ context.Context, balance *models.HRLeaveBalance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *balance
	r.balances[leaveBalanceKey(balance.TenantID, balance.EmployeeID, balance.Year)] = &cp
	return nil
}

type stubLeaveTypeRepo struct {
	mu    sync.Mutex
	types map[uuid.UUID]*models.LeaveType
}

func newStubLeaveTypeRepo() *stubLeaveTypeRepo {
	return &stubLeaveTypeRepo{types: make(map[uuid.UUID]*models.LeaveType)}
}

func (r *stubLeaveTypeRepo) put(t *models.LeaveType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	r.types[t.ID] = &cp
}

func (r *stubLeaveTypeRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*models.LeaveType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.LeaveType
	for _, t := range r.types {
		if t.TenantID == tenantID {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubLeaveTypeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.LeaveType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.types[id]
	if !ok {
		return nil, leave.ErrLeaveTypeNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubLeaveTypeRepo) GetByKey(_ context.Context, tenantID uuid.UUID, key string) (*models.LeaveType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.types {
		if t.TenantID == tenantID && t.Key == key {
			cp := *t
			return &cp, nil
		}
	}
	return nil, leave.ErrLeaveTypeNotFound
}

type stubLeaveEmployeeRepo struct {
	mu        sync.Mutex
	employees map[uuid.UUID]*models.EmployeeProfile
}

func newStubLeaveEmployeeRepo() *stubLeaveEmployeeRepo {
	return &stubLeaveEmployeeRepo{employees: make(map[uuid.UUID]*models.EmployeeProfile)}
}

func (r *stubLeaveEmployeeRepo) put(e *models.EmployeeProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *e
	r.employees[e.UserID] = &cp
}

func (r *stubLeaveEmployeeRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*models.EmployeeProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.employees[userID]
	if !ok {
		return nil, employee.ErrEmployeeNotFound
	}
	cp := *e
	return &cp, nil
}

// leaveTestFixtures bundles the stubs wired into a real leave.Service, and the
// server under test, so each RPC test only has to seed the data it needs.
type leaveTestFixtures struct {
	srv         *HRGRPCServer
	requestRepo *stubLeaveRequestRepo
	balanceRepo *stubLeaveBalanceRepo
	typeRepo    *stubLeaveTypeRepo
	employeeRepo *stubLeaveEmployeeRepo
}

func newLeaveTestFixtures() *leaveTestFixtures {
	requestRepo := newStubLeaveRequestRepo()
	balanceRepo := newStubLeaveBalanceRepo()
	typeRepo := newStubLeaveTypeRepo()
	employeeRepo := newStubLeaveEmployeeRepo()

	// settingsRepo is nil: none of the leave types seeded by these tests set
	// RequiresAUDocument, so the AU-threshold branch that reads it never runs.
	svc := leave.NewService(requestRepo, balanceRepo, typeRepo, nil, employeeRepo)
	srv := NewHRGRPCServer(svc, nil, nil, nil, nil, nil)

	return &leaveTestFixtures{
		srv:          srv,
		requestRepo:  requestRepo,
		balanceRepo:  balanceRepo,
		typeRepo:     typeRepo,
		employeeRepo: employeeRepo,
	}
}

// seedEmployee registers an employee profile with a full-year start date, so
// getOrCreateBalance always yields the full annual entitlement (no pro-rata
// surprises in assertions).
func (f *leaveTestFixtures) seedEmployee(userID uuid.UUID, managerID *uuid.UUID) *models.EmployeeProfile {
	e := &models.EmployeeProfile{
		ID:              uuid.New(),
		TenantID:        uuid.Nil,
		UserID:          userID,
		WorkDaysPerWeek: 5,
		AnnualLeaveDays: 30,
		ManagerUserID:   managerID,
		StartDate:       time.Now().AddDate(-2, 0, 0),
	}
	f.employeeRepo.put(e)
	return e
}

func (f *leaveTestFixtures) seedLeaveType(tenantID uuid.UUID, key string, requiresApproval, deducts bool) *models.LeaveType {
	lt := &models.LeaveType{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Name:               key,
		Key:                key,
		DeductsFromBalance: deducts,
		RequiresApproval:   requiresApproval,
	}
	f.typeRepo.put(lt)
	return lt
}

func (f *leaveTestFixtures) seedRequest(tenantID, employeeID, leaveTypeID uuid.UUID, status models.LeaveRequestStatus) *models.LeaveRequest {
	now := time.Now()
	req := &models.LeaveRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
		StartDate:   now.AddDate(0, 0, 5),
		EndDate:     now.AddDate(0, 0, 6),
		TotalDays:   decimal.NewFromInt(2),
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	f.requestRepo.put(req)
	return req
}

// ============================================================================
// CreateLeaveRequest
// ============================================================================

func TestCreateLeaveRequest_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.CreateLeaveRequest(ctx, &hrv1.CreateLeaveRequestReq{
		UserId:      employeeID.String(),
		LeaveTypeId: lt.ID.String(),
		StartDate:   "2027-03-01",
		EndDate:     "2027-03-02",
		Reason:      "Familienurlaub",
	})
	require.NoError(t, err)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_PENDING, resp.GetLeaveRequest().GetStatus())
	require.Equal(t, "2", resp.GetLeaveRequest().GetTotalDays())
}

func TestCreateLeaveRequest_InvalidDateRange(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)

	ctx := ctxWithTenant(tenantID)
	_, err := f.srv.CreateLeaveRequest(ctx, &hrv1.CreateLeaveRequestReq{
		UserId:      employeeID.String(),
		LeaveTypeId: lt.ID.String(),
		StartDate:   "2027-03-10",
		EndDate:     "2027-03-01",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// GetLeaveRequest
// ============================================================================

func TestGetLeaveRequest_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusPending)

	resp, err := f.srv.GetLeaveRequest(context.Background(), &hrv1.GetLeaveRequestReq{Id: req.ID.String()})
	require.NoError(t, err)
	require.Equal(t, req.ID.String(), resp.GetLeaveRequest().GetId())
}

func TestGetLeaveRequest_NotFound(t *testing.T) {
	f := newLeaveTestFixtures()
	_, err := f.srv.GetLeaveRequest(context.Background(), &hrv1.GetLeaveRequestReq{Id: uuid.New().String()})
	assertGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// ListLeaveRequests
// ============================================================================

func TestListLeaveRequests_FiltersByEmployee(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeA := uuid.New()
	employeeB := uuid.New()
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	f.seedRequest(tenantID, employeeA, lt.ID, models.LeaveStatusPending)
	f.seedRequest(tenantID, employeeB, lt.ID, models.LeaveStatusPending)
	f.seedRequest(uuid.New(), employeeA, lt.ID, models.LeaveStatusPending) // other tenant

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.ListLeaveRequests(ctx, &hrv1.ListLeaveRequestsReq{EmployeeId: employeeA.String()})
	require.NoError(t, err)
	require.Len(t, resp.GetLeaveRequests(), 1)
	require.Equal(t, employeeA.String(), resp.GetLeaveRequests()[0].GetEmployeeId())
}

func TestListLeaveRequests_MissingTenant(t *testing.T) {
	f := newLeaveTestFixtures()
	_, err := f.srv.ListLeaveRequests(context.Background(), &hrv1.ListLeaveRequestsReq{})
	assertGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// ApproveLeaveRequest
// ============================================================================

func TestApproveLeaveRequest_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	managerID := uuid.New()
	f.seedEmployee(employeeID, &managerID)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusPending)

	resp, err := f.srv.ApproveLeaveRequest(context.Background(), &hrv1.ApproveLeaveRequestReq{
		Id:         req.ID.String(),
		ApproverId: managerID.String(),
		Comment:    "passt",
	})
	require.NoError(t, err)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_APPROVED, resp.GetLeaveRequest().GetStatus())
	require.Equal(t, managerID.String(), resp.GetLeaveRequest().GetApprovedBy())
}

func TestApproveLeaveRequest_AlreadyDecided(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	managerID := uuid.New()
	f.seedEmployee(employeeID, &managerID)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusApproved)

	_, err := f.srv.ApproveLeaveRequest(context.Background(), &hrv1.ApproveLeaveRequestReq{
		Id:         req.ID.String(),
		ApproverId: managerID.String(),
	})
	assertGRPCCode(t, err, codes.FailedPrecondition)
}

// ============================================================================
// RejectLeaveRequest
// ============================================================================

func TestRejectLeaveRequest_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	managerID := uuid.New()
	f.seedEmployee(employeeID, &managerID)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusPending)

	resp, err := f.srv.RejectLeaveRequest(context.Background(), &hrv1.RejectLeaveRequestReq{
		Id:         req.ID.String(),
		ApproverId: managerID.String(),
		Comment:    "kein Personal frei",
	})
	require.NoError(t, err)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_REJECTED, resp.GetLeaveRequest().GetStatus())
}

func TestRejectLeaveRequest_AlreadyDecided(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	managerID := uuid.New()
	f.seedEmployee(employeeID, &managerID)
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusRejected)

	_, err := f.srv.RejectLeaveRequest(context.Background(), &hrv1.RejectLeaveRequestReq{
		Id:         req.ID.String(),
		ApproverId: managerID.String(),
	})
	assertGRPCCode(t, err, codes.FailedPrecondition)
}

// ============================================================================
// CancelLeaveRequest
// ============================================================================

func TestCancelLeaveRequest_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusPending)

	resp, err := f.srv.CancelLeaveRequest(context.Background(), &hrv1.CancelLeaveRequestReq{
		Id:     req.ID.String(),
		UserId: employeeID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_CANCELLED, resp.GetLeaveRequest().GetStatus())
}

func TestCancelLeaveRequest_NotOwner(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	lt := f.seedLeaveType(tenantID, "urlaub", true, true)
	req := f.seedRequest(tenantID, employeeID, lt.ID, models.LeaveStatusPending)

	_, err := f.srv.CancelLeaveRequest(context.Background(), &hrv1.CancelLeaveRequestReq{
		Id:     req.ID.String(),
		UserId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.PermissionDenied)
}

// ============================================================================
// GetLeaveBalance / GetEmployeeLeaveBalance
// ============================================================================

func TestGetLeaveBalance_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.GetLeaveBalance(ctx, &hrv1.GetLeaveBalanceReq{UserId: employeeID.String()})
	require.NoError(t, err)
	require.Equal(t, "30", resp.GetBalance().GetEntitlement())
}

func TestGetLeaveBalance_InvalidUserID(t *testing.T) {
	f := newLeaveTestFixtures()
	ctx := ctxWithTenant(uuid.New())
	_, err := f.srv.GetLeaveBalance(ctx, &hrv1.GetLeaveBalanceReq{UserId: "not-a-uuid"})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetEmployeeLeaveBalance_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.GetEmployeeLeaveBalance(ctx, &hrv1.GetEmployeeLeaveBalanceReq{EmployeeId: employeeID.String()})
	require.NoError(t, err)
	require.Equal(t, "30", resp.GetBalance().GetEntitlement())
}

// ============================================================================
// ListLeaveTypes
// ============================================================================

func TestListLeaveTypes_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	f.seedLeaveType(tenantID, "urlaub", true, true)
	f.seedLeaveType(tenantID, "krank", false, false)
	f.seedLeaveType(uuid.New(), "other-tenant", true, true)

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.ListLeaveTypes(ctx, &hrv1.ListLeaveTypesReq{})
	require.NoError(t, err)
	require.Len(t, resp.GetLeaveTypes(), 2)
}

// ============================================================================
// RecordSickLeave
// ============================================================================

func TestRecordSickLeave_HappyPath(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)
	f.seedLeaveType(tenantID, "krank", false, false)

	ctx := ctxWithTenant(tenantID)
	resp, err := f.srv.RecordSickLeave(ctx, &hrv1.RecordSickLeaveReq{
		UserId:    employeeID.String(),
		StartDate: "2027-03-01",
		EndDate:   "2027-03-01",
		Notes:     "Grippe",
	})
	require.NoError(t, err)
	require.Equal(t, hrv1.LeaveRequestStatus_LEAVE_APPROVED, resp.GetLeaveRequest().GetStatus())
	require.False(t, resp.GetAuDocumentRequired())
}

func TestRecordSickLeave_LeaveTypeNotFound(t *testing.T) {
	f := newLeaveTestFixtures()
	tenantID := uuid.New()
	employeeID := uuid.New()
	f.seedEmployee(employeeID, nil)
	// No "krank" leave type seeded for this tenant.

	ctx := ctxWithTenant(tenantID)
	_, err := f.srv.RecordSickLeave(ctx, &hrv1.RecordSickLeaveReq{
		UserId:    employeeID.String(),
		StartDate: "2027-03-01",
		EndDate:   "2027-03-01",
	})
	// mapHRError has no case for leave.ErrLeaveTypeNotFound (falls through to the
	// default codes.Internal branch) -- documented gap for a future unit, not a
	// regression introduced here.
	assertGRPCCode(t, err, codes.Internal)
}
