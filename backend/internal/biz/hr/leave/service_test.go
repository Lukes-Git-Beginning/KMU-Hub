package leave

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockLeaveRequestRepo struct {
	requests map[uuid.UUID]*models.LeaveRequest
	overlaps []*models.LeaveRequest
}

func newMockLeaveRequestRepo() *mockLeaveRequestRepo {
	return &mockLeaveRequestRepo{requests: make(map[uuid.UUID]*models.LeaveRequest)}
}

func (m *mockLeaveRequestRepo) Create(_ context.Context, req *models.LeaveRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockLeaveRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*models.LeaveRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, ErrLeaveRequestNotFound
	}
	return req, nil
}

func (m *mockLeaveRequestRepo) List(_ context.Context, _ LeaveRequestFilter) ([]*models.LeaveRequest, int, error) {
	var results []*models.LeaveRequest
	for _, req := range m.requests {
		results = append(results, req)
	}
	return results, len(results), nil
}

func (m *mockLeaveRequestRepo) Update(_ context.Context, req *models.LeaveRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockLeaveRequestRepo) FindOverlaps(_ context.Context, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) ([]*models.LeaveRequest, error) {
	return m.overlaps, nil
}

type mockLeaveBalanceRepo struct {
	balances map[string]*models.HRLeaveBalance // key: tenantID-employeeID-year
}

func newMockLeaveBalanceRepo() *mockLeaveBalanceRepo {
	return &mockLeaveBalanceRepo{balances: make(map[string]*models.HRLeaveBalance)}
}

func balanceKey(tenantID, employeeID uuid.UUID, year int) string {
	return tenantID.String() + "-" + employeeID.String() + "-" + string(rune(year))
}

func (m *mockLeaveBalanceRepo) GetByEmployeeYear(_ context.Context, tenantID, employeeID uuid.UUID, year int) (*models.HRLeaveBalance, error) {
	key := balanceKey(tenantID, employeeID, year)
	b, ok := m.balances[key]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *mockLeaveBalanceRepo) Upsert(_ context.Context, balance *models.HRLeaveBalance) error {
	key := balanceKey(balance.TenantID, balance.EmployeeID, balance.Year)
	m.balances[key] = balance
	return nil
}

type mockLeaveTypeRepo struct {
	types    map[uuid.UUID]*models.LeaveType
	typesByKey map[string]*models.LeaveType // key: tenantID-key
}

func newMockLeaveTypeRepo() *mockLeaveTypeRepo {
	return &mockLeaveTypeRepo{
		types:    make(map[uuid.UUID]*models.LeaveType),
		typesByKey: make(map[string]*models.LeaveType),
	}
}

func (m *mockLeaveTypeRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*models.LeaveType, error) {
	var results []*models.LeaveType
	for _, lt := range m.types {
		if lt.TenantID == tenantID {
			results = append(results, lt)
		}
	}
	return results, nil
}

func (m *mockLeaveTypeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.LeaveType, error) {
	lt, ok := m.types[id]
	if !ok {
		return nil, ErrLeaveTypeNotFound
	}
	return lt, nil
}

func (m *mockLeaveTypeRepo) GetByKey(_ context.Context, tenantID uuid.UUID, key string) (*models.LeaveType, error) {
	lt, ok := m.typesByKey[tenantID.String()+"-"+key]
	if !ok {
		return nil, ErrLeaveTypeNotFound
	}
	return lt, nil
}

func (m *mockLeaveTypeRepo) addType(lt *models.LeaveType) {
	m.types[lt.ID] = lt
	m.typesByKey[lt.TenantID.String()+"-"+lt.Key] = lt
}

type mockHRSettingsRepo struct {
	settings map[uuid.UUID]*models.HRCompanySettings
}

func newMockHRSettingsRepo() *mockHRSettingsRepo {
	return &mockHRSettingsRepo{settings: make(map[uuid.UUID]*models.HRCompanySettings)}
}

func (m *mockHRSettingsRepo) GetByTenant(_ context.Context, tenantID uuid.UUID) (*models.HRCompanySettings, error) {
	s, ok := m.settings[tenantID]
	if !ok {
		return nil, ErrSettingsNotFound
	}
	return s, nil
}

func (m *mockHRSettingsRepo) Upsert(_ context.Context, settings *models.HRCompanySettings) error {
	m.settings[settings.TenantID] = settings
	return nil
}

type mockEmployeeRepo struct {
	employees map[uuid.UUID]*models.EmployeeProfile // key: userID
}

func newMockEmployeeRepo() *mockEmployeeRepo {
	return &mockEmployeeRepo{employees: make(map[uuid.UUID]*models.EmployeeProfile)}
}

func (m *mockEmployeeRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*models.EmployeeProfile, error) {
	emp, ok := m.employees[userID]
	if !ok {
		return nil, ErrLeaveRequestNotFound
	}
	return emp, nil
}

// ============================================================================
// Test fixtures
// ============================================================================

var (
	testTenantID   = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	testEmployeeID = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	testManagerID  = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	testHRUserID   = uuid.MustParse("40000000-0000-0000-0000-000000000001")
)

func setupTestService() (*Service, *mockLeaveRequestRepo, *mockLeaveBalanceRepo, *mockLeaveTypeRepo, *mockHRSettingsRepo, *mockEmployeeRepo) {
	leaveRepo := newMockLeaveRequestRepo()
	balanceRepo := newMockLeaveBalanceRepo()
	typeRepo := newMockLeaveTypeRepo()
	settingsRepo := newMockHRSettingsRepo()
	employeeRepo := newMockEmployeeRepo()

	// Add standard leave types
	urlaubType := &models.LeaveType{
		ID:                 uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		TenantID:           testTenantID,
		Name:               "Urlaub",
		Key:                "urlaub",
		Color:              "#3d8abf",
		DeductsFromBalance: true,
		RequiresApproval:   true,
		RequiresAUDocument: false,
		IsSystem:           true,
	}
	typeRepo.addType(urlaubType)

	krankType := &models.LeaveType{
		ID:                 uuid.MustParse("50000000-0000-0000-0000-000000000002"),
		TenantID:           testTenantID,
		Name:               "Krankheit",
		Key:                "krank",
		Color:              "#bf3d3d",
		DeductsFromBalance: false,
		RequiresApproval:   false,
		RequiresAUDocument: true,
		IsSystem:           true,
	}
	typeRepo.addType(krankType)

	// Add employee with manager
	employeeRepo.employees[testEmployeeID] = &models.EmployeeProfile{
		ID:              uuid.MustParse("60000000-0000-0000-0000-000000000001"),
		UserID:          testEmployeeID,
		Department:      "Engineering",
		PositionTitle:   "Developer",
		ContractType:    models.HRContractFullTime,
		WorkDaysPerWeek: 5,
		AnnualLeaveDays: 30,
		ManagerUserID:   &testManagerID,
		StartDate:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Add settings
	settingsRepo.settings[testTenantID] = &models.HRCompanySettings{
		ID:                     uuid.New(),
		TenantID:               testTenantID,
		AUThresholdDays:        3,
		ShowAbsenceReason:      true,
		DefaultAnnualLeaveDays: 20,
		Timezone:               "Europe/Berlin",
	}

	svc := NewService(leaveRepo, balanceRepo, typeRepo, settingsRepo, employeeRepo)
	return svc, leaveRepo, balanceRepo, typeRepo, settingsRepo, employeeRepo
}

// ============================================================================
// Tests
// ============================================================================

func TestCreateLeaveRequest_PendingStatus(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"), // urlaub
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Sommerurlaub",
	})
	require.NoError(t, err)
	assert.Equal(t, models.LeaveStatusPending, result.Request.Status)
	assert.Equal(t, decimal.NewFromInt(5), result.Request.TotalDays)
	assert.Nil(t, result.Request.ApprovedBy)

	// Verify stored
	stored, ok := leaveRepo.requests[result.Request.ID]
	require.True(t, ok)
	assert.Equal(t, models.LeaveStatusPending, stored.Status)
}

func TestCreateLeaveRequest_HalfDaySupport(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	// 3 days: half-day start + full middle + half-day end = 2.0
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID:    uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
		IsHalfDayStart: true,
		IsHalfDayEnd:   true,
		Reason:         "Half day test",
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(2).Equal(result.Request.TotalDays),
		"expected 2, got %s", result.Request.TotalDays)

	// Single half-day
	result2, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID:    uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		IsHalfDayStart: true,
		Reason:         "Half day single",
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(0.5).Equal(result2.Request.TotalDays),
		"expected 0.5, got %s", result2.Request.TotalDays)
}

func TestCreateLeaveRequest_InsufficientBalance(t *testing.T) {
	svc, _, balanceRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	// Pre-create a balance with very little remaining
	key := balanceKey(testTenantID, testEmployeeID, time.Now().Year())
	balanceRepo.balances[key] = &models.HRLeaveBalance{
		ID:          uuid.New(),
		TenantID:    testTenantID,
		EmployeeID:  testEmployeeID,
		Year:        time.Now().Year(),
		Entitlement: decimal.NewFromInt(30),
		Used:        decimal.NewFromInt(29),
		Remaining:   decimal.NewFromInt(1),
	}

	_, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"), // urlaub
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Too much leave",
	})
	assert.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestCreateLeaveRequest_InvalidDateRange(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	_, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Reason:      "Invalid dates",
	})
	assert.ErrorIs(t, err, ErrInvalidDateRange)
}

func TestApproveLeaveRequest_Success(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	// Create pending request
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Urlaub",
	})
	require.NoError(t, err)

	approveResult, err := svc.ApproveLeaveRequest(ctx, result.Request.ID, testManagerID, "Genehmigt")
	require.NoError(t, err)

	stored := leaveRepo.requests[result.Request.ID]
	assert.Equal(t, models.LeaveStatusApproved, stored.Status)
	assert.Equal(t, &testManagerID, stored.ApprovedBy)
	assert.Equal(t, "Genehmigt", stored.ApprovalComment)
	assert.NotNil(t, stored.ApprovedAt)
	assert.Empty(t, approveResult.Overlaps) // No overlaps in this scenario
}

func TestApproveLeaveRequest_WithOverlaps(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	// Create pending request
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Urlaub 1",
	})
	require.NoError(t, err)

	// Set up overlapping request in mock
	overlap := &models.LeaveRequest{
		ID:          uuid.New(),
		EmployeeID:  testEmployeeID,
		StartDate:   time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC),
		Status:      models.LeaveStatusApproved,
	}
	leaveRepo.overlaps = []*models.LeaveRequest{overlap}

	// Approve should succeed despite overlaps (warn but allow)
	approveResult, err := svc.ApproveLeaveRequest(ctx, result.Request.ID, testManagerID, "OK trotz Overlap")
	require.NoError(t, err)
	assert.Equal(t, models.LeaveStatusApproved, approveResult.Request.Status)
	assert.Len(t, approveResult.Overlaps, 1)
}

func TestApproveLeaveRequest_SelfApprovalDenied(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Self approve attempt",
	})
	require.NoError(t, err)

	_, err = svc.ApproveLeaveRequest(ctx, result.Request.ID, testEmployeeID, "Self approve")
	assert.ErrorIs(t, err, ErrCannotApproveOwnRequest)
}

func TestApproveLeaveRequest_NonManagerDenied(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Non-manager approve",
	})
	require.NoError(t, err)

	randomUser := uuid.MustParse("90000000-0000-0000-0000-000000000099")
	_, err = svc.ApproveLeaveRequest(ctx, result.Request.ID, randomUser, "Random user")
	assert.ErrorIs(t, err, ErrNotAuthorizedToApprove)
}

func TestRejectLeaveRequest_Success(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Urlaub",
	})
	require.NoError(t, err)

	err = svc.RejectLeaveRequest(ctx, result.Request.ID, testManagerID, "Team unterbesetzt")
	require.NoError(t, err)

	stored := leaveRepo.requests[result.Request.ID]
	assert.Equal(t, models.LeaveStatusRejected, stored.Status)
	assert.Equal(t, "Team unterbesetzt", stored.ApprovalComment)
}

func TestCancelLeaveRequest_PendingSuccess(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Cancel test",
	})
	require.NoError(t, err)

	err = svc.CancelLeaveRequest(ctx, result.Request.ID, testEmployeeID)
	require.NoError(t, err)

	stored := leaveRepo.requests[result.Request.ID]
	assert.Equal(t, models.LeaveStatusCancelled, stored.Status)
}

func TestCancelLeaveRequest_ApprovedBeforeStartDate(t *testing.T) {
	svc, leaveRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()

	// Create and approve
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC), // Far future
		EndDate:     time.Date(2027, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Cancel approved test",
	})
	require.NoError(t, err)

	_, err = svc.ApproveLeaveRequest(ctx, result.Request.ID, testManagerID, "OK")
	require.NoError(t, err)

	// Cancel approved (before start date)
	err = svc.CancelLeaveRequest(ctx, result.Request.ID, testEmployeeID)
	require.NoError(t, err)

	stored := leaveRepo.requests[result.Request.ID]
	assert.Equal(t, models.LeaveStatusCancelled, stored.Status)
}

func TestCancelLeaveRequest_ByNonOwnerDenied(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Reason:      "Cancel by other",
	})
	require.NoError(t, err)

	err = svc.CancelLeaveRequest(ctx, result.Request.ID, testManagerID)
	assert.ErrorIs(t, err, ErrNotAuthorizedToApprove)
}

func TestSickLeave_AutoApproves(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	req, err := svc.RecordSickLeave(ctx, testTenantID, testEmployeeID, SickLeaveInput{
		StartDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
		Reason:    "Erkaeltet",
	})
	require.NoError(t, err)
	assert.Equal(t, models.LeaveStatusApproved, req.Status)
	assert.NotNil(t, req.ApprovedBy)
	assert.False(t, req.AUDocumentRequired) // 2 days < 3 day threshold
}

func TestSickLeave_AUThresholdFlagged(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	req, err := svc.RecordSickLeave(ctx, testTenantID, testEmployeeID, SickLeaveInput{
		StartDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC), // 5 days >= 3 threshold
		Reason:    "Grippe",
	})
	require.NoError(t, err)
	assert.Equal(t, models.LeaveStatusApproved, req.Status)
	assert.True(t, req.AUDocumentRequired)
}

func TestCalculateTotalDays(t *testing.T) {
	tests := []struct {
		name           string
		startDate      time.Time
		endDate        time.Time
		isHalfDayStart bool
		isHalfDayEnd   bool
		expected       decimal.Decimal
	}{
		{
			name:      "Full single day",
			startDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			expected:  decimal.NewFromInt(1),
		},
		{
			name:      "Full 5 days",
			startDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
			expected:  decimal.NewFromInt(5),
		},
		{
			name:           "Half day start only",
			startDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:        time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			isHalfDayStart: true,
			expected:       decimal.NewFromFloat(0.5),
		},
		{
			name:           "Half day start + full end (2 days)",
			startDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:        time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
			isHalfDayStart: true,
			expected:       decimal.NewFromFloat(1.5),
		},
		{
			name:           "Full start + half day end (3 days span)",
			startDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:        time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC),
			isHalfDayEnd:   true,
			expected:       decimal.NewFromFloat(2.5),
		},
		{
			name:           "Both half days (3 days span)",
			startDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			endDate:        time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC),
			isHalfDayStart: true,
			isHalfDayEnd:   true,
			expected:       decimal.NewFromFloat(2.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTotalDays(tt.startDate, tt.endDate, tt.isHalfDayStart, tt.isHalfDayEnd)
			assert.True(t, tt.expected.Equal(result), "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestApproveLeaveRequest_UpdatesBalance(t *testing.T) {
	svc, _, balanceRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	// Create request
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"), // urlaub, deducts
		StartDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), // 3 days
		Reason:      "Balance test",
	})
	require.NoError(t, err)

	// Approve
	_, err = svc.ApproveLeaveRequest(ctx, result.Request.ID, testManagerID, "OK")
	require.NoError(t, err)

	// Check balance was updated
	key := balanceKey(testTenantID, testEmployeeID, time.Now().Year())
	balance := balanceRepo.balances[key]
	require.NotNil(t, balance)
	assert.True(t, balance.Used.GreaterThan(decimal.NewFromInt(0)))
}

func TestCancelLeaveRequest_RestoresBalance(t *testing.T) {
	svc, _, balanceRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	// Create and approve
	result, err := svc.CreateLeaveRequest(ctx, testTenantID, testEmployeeID, CreateLeaveInput{
		LeaveTypeID: uuid.MustParse("50000000-0000-0000-0000-000000000001"),
		StartDate:   time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2027, 6, 3, 0, 0, 0, 0, time.UTC),
		Reason:      "Cancel restore test",
	})
	require.NoError(t, err)

	_, err = svc.ApproveLeaveRequest(ctx, result.Request.ID, testManagerID, "OK")
	require.NoError(t, err)

	// Get balance after approval
	key := balanceKey(testTenantID, testEmployeeID, time.Now().Year())
	balanceAfterApprove := balanceRepo.balances[key]
	usedAfterApprove := balanceAfterApprove.Used

	// Cancel
	err = svc.CancelLeaveRequest(ctx, result.Request.ID, testEmployeeID)
	require.NoError(t, err)

	// Balance should be restored
	balanceAfterCancel := balanceRepo.balances[key]
	assert.True(t, balanceAfterCancel.Used.LessThan(usedAfterApprove))
}
