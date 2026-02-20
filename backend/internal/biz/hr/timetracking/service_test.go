package timetracking

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock Repositories
// ============================================================================

type mockWorkTimeRepo struct {
	entries          map[uuid.UUID]*models.HRWorkTimeEntry
	activeShift      *models.HRWorkTimeEntry
	previousShiftEnd *time.Time
	dailySummary     *DailySummary
	weeklySummary    *WeeklySummary
}

func newMockWorkTimeRepo() *mockWorkTimeRepo {
	return &mockWorkTimeRepo{
		entries: make(map[uuid.UUID]*models.HRWorkTimeEntry),
	}
}

func (m *mockWorkTimeRepo) Create(_ context.Context, entry *models.HRWorkTimeEntry) error {
	m.entries[entry.ID] = entry
	if entry.Status == models.WorkTimeStatusActive {
		m.activeShift = entry
	}
	return nil
}

func (m *mockWorkTimeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.HRWorkTimeEntry, error) {
	entry, ok := m.entries[id]
	if !ok {
		return nil, ErrWorkTimeEntryNotFound
	}
	return entry, nil
}

func (m *mockWorkTimeRepo) GetActiveShift(_ context.Context, _ uuid.UUID) (*models.HRWorkTimeEntry, error) {
	return m.activeShift, nil
}

func (m *mockWorkTimeRepo) Update(_ context.Context, entry *models.HRWorkTimeEntry) error {
	m.entries[entry.ID] = entry
	if entry.Status != models.WorkTimeStatusActive {
		if m.activeShift != nil && m.activeShift.ID == entry.ID {
			m.activeShift = nil
		}
	}
	return nil
}

func (m *mockWorkTimeRepo) List(_ context.Context, _ WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error) {
	var result []*models.HRWorkTimeEntry
	for _, e := range m.entries {
		result = append(result, e)
	}
	return result, len(result), nil
}

func (m *mockWorkTimeRepo) GetPreviousShiftEnd(_ context.Context, _ uuid.UUID, _ time.Time) (*time.Time, error) {
	return m.previousShiftEnd, nil
}

func (m *mockWorkTimeRepo) GetDailySummary(_ context.Context, _ uuid.UUID, _ time.Time) (*DailySummary, error) {
	if m.dailySummary != nil {
		return m.dailySummary, nil
	}
	return &DailySummary{}, nil
}

func (m *mockWorkTimeRepo) GetWeeklySummary(_ context.Context, _ uuid.UUID, _ time.Time) (*WeeklySummary, error) {
	if m.weeklySummary != nil {
		return m.weeklySummary, nil
	}
	return &WeeklySummary{}, nil
}

type mockBreakRepo struct {
	breaks      map[uuid.UUID]*models.HRBreakEntry
	activeBreak *models.HRBreakEntry
}

func newMockBreakRepo() *mockBreakRepo {
	return &mockBreakRepo{
		breaks: make(map[uuid.UUID]*models.HRBreakEntry),
	}
}

func (m *mockBreakRepo) Create(_ context.Context, entry *models.HRBreakEntry) error {
	m.breaks[entry.ID] = entry
	if entry.EndTime == nil {
		m.activeBreak = entry
	}
	return nil
}

func (m *mockBreakRepo) GetActiveBreak(_ context.Context, _ uuid.UUID) (*models.HRBreakEntry, error) {
	return m.activeBreak, nil
}

func (m *mockBreakRepo) Update(_ context.Context, entry *models.HRBreakEntry) error {
	m.breaks[entry.ID] = entry
	if entry.EndTime != nil && m.activeBreak != nil && m.activeBreak.ID == entry.ID {
		m.activeBreak = nil
	}
	return nil
}

func (m *mockBreakRepo) ListByWorkTimeEntry(_ context.Context, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error) {
	var result []*models.HRBreakEntry
	for _, b := range m.breaks {
		if b.WorkTimeEntryID == workTimeEntryID {
			result = append(result, b)
		}
	}
	if result == nil {
		result = []*models.HRBreakEntry{}
	}
	return result, nil
}

type mockEmployeeRepo struct {
	profile *models.EmployeeProfile
}

func (m *mockEmployeeRepo) Create(_ context.Context, _ *models.EmployeeProfile) error { return nil }
func (m *mockEmployeeRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.EmployeeProfile, error) {
	return m.profile, nil
}
func (m *mockEmployeeRepo) GetByUserID(_ context.Context, _ uuid.UUID) (*models.EmployeeProfile, error) {
	return m.profile, nil
}
func (m *mockEmployeeRepo) List(_ context.Context, _ employee.EmployeeFilter) ([]*models.EmployeeProfile, int, error) {
	return nil, 0, nil
}
func (m *mockEmployeeRepo) Update(_ context.Context, _ *models.EmployeeProfile) error { return nil }

type mockSettingsRepo struct {
	settings *models.HRCompanySettings
}

func (m *mockSettingsRepo) GetByTenant(_ context.Context, _ uuid.UUID) (*models.HRCompanySettings, error) {
	if m.settings != nil {
		return m.settings, nil
	}
	return &models.HRCompanySettings{Timezone: "Europe/Berlin"}, nil
}

func (m *mockSettingsRepo) Upsert(_ context.Context, _ *models.HRCompanySettings) error {
	return nil
}

// ============================================================================
// Helper
// ============================================================================

func newTestService() (*Service, *mockWorkTimeRepo, *mockBreakRepo) {
	workRepo := newMockWorkTimeRepo()
	breakRepo := newMockBreakRepo()
	empRepo := &mockEmployeeRepo{
		profile: &models.EmployeeProfile{
			WorkDaysPerWeek: 5,
			AnnualLeaveDays: 30,
		},
	}
	settingsRepo := &mockSettingsRepo{}

	svc := NewService(workRepo, breakRepo, empRepo, settingsRepo, nil)
	return svc, workRepo, breakRepo
}

// ============================================================================
// Tests
// ============================================================================

func TestClockIn_CreatesActiveEntry(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, checkResult, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, models.WorkTimeStatusActive, entry.Status)
	assert.Equal(t, tenantID, entry.TenantID)
	assert.Equal(t, employeeID, entry.EmployeeID)
	assert.Nil(t, checkResult) // No rest violation

	// Verify stored
	stored, ok := workRepo.entries[entry.ID]
	assert.True(t, ok)
	assert.Equal(t, models.WorkTimeStatusActive, stored.Status)
}

func TestClockIn_TwiceReturnsError(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	_, _, err = svc.ClockIn(ctx, tenantID, employeeID)
	assert.ErrorIs(t, err, ErrAlreadyClockedIn)
}

func TestClockOut_CalculatesNetWorkMinutes(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// Simulate clock-in 2 hours ago
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	entry.ClockIn = twoHoursAgo
	workRepo.entries[entry.ID] = entry
	workRepo.activeShift = entry

	completed, checkResult, err := svc.ClockOut(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, completed)
	assert.Equal(t, models.WorkTimeStatusCompleted, completed.Status)
	assert.NotNil(t, completed.ClockOut)
	assert.NotNil(t, completed.NetWorkMinutes)
	// ~120 minutes of work, no break required (<6h)
	assert.True(t, *completed.NetWorkMinutes >= 119 && *completed.NetWorkMinutes <= 121)
	assert.NotNil(t, checkResult)
	assert.Equal(t, "none", string(checkResult.Severity))
}

func TestClockOut_SevenHoursNoBreak_AutoDeducts30Min(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// Simulate clock-in 7 hours ago
	sevenHoursAgo := time.Now().Add(-7 * time.Hour)
	entry.ClockIn = sevenHoursAgo
	workRepo.entries[entry.ID] = entry
	workRepo.activeShift = entry

	completed, checkResult, err := svc.ClockOut(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.Equal(t, 30, completed.AutoBreakDeducted)
	assert.Equal(t, 0, completed.BreakMinutes) // No manual breaks
	// Net work = ~420 - 0 - 30 = ~390
	assert.True(t, *completed.NetWorkMinutes >= 389 && *completed.NetWorkMinutes <= 391)
	assert.NotNil(t, checkResult)
}

func TestClockOut_TenHoursWithManualBreak_AutoDeducts15Min(t *testing.T) {
	svc, workRepo, breakRepo := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	entry, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// Simulate clock-in 10 hours ago
	tenHoursAgo := time.Now().Add(-10 * time.Hour)
	entry.ClockIn = tenHoursAgo
	workRepo.entries[entry.ID] = entry
	workRepo.activeShift = entry

	// Add a 30-minute manual break
	breakDuration := 30
	breakEnd := tenHoursAgo.Add(4*time.Hour + 30*time.Minute)
	breakEntry := &models.HRBreakEntry{
		ID:              uuid.New(),
		WorkTimeEntryID: entry.ID,
		StartTime:       tenHoursAgo.Add(4 * time.Hour),
		EndTime:         &breakEnd,
		DurationMinutes: &breakDuration,
		CreatedAt:       tenHoursAgo,
	}
	breakRepo.breaks[breakEntry.ID] = breakEntry

	completed, checkResult, err := svc.ClockOut(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// >9h work needs 45min break, manual 30min, so auto-deduct 15min
	assert.Equal(t, 15, completed.AutoBreakDeducted)
	assert.Equal(t, 30, completed.BreakMinutes)
	assert.NotNil(t, checkResult)
	// At exactly 600min (10h), CheckWorkTime returns warning (>=9h);
	// Error is only for >10h (>600min). This is consistent with ArbZG impl.
	assert.Equal(t, "warning", string(checkResult.Severity))
}

func TestStartBreak_EndBreak_Lifecycle(t *testing.T) {
	svc, _, breakRepo := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	// Must be clocked in first
	_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// Start break
	breakEntry, err := svc.StartBreak(ctx, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, breakEntry)
	assert.Nil(t, breakEntry.EndTime)

	// Can't start another break
	_, err = svc.StartBreak(ctx, employeeID)
	assert.ErrorIs(t, err, ErrAlreadyOnBreak)

	// End break
	endedBreak, err := svc.EndBreak(ctx, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, endedBreak)
	assert.NotNil(t, endedBreak.EndTime)
	assert.NotNil(t, endedBreak.DurationMinutes)

	// Verify no active break
	assert.Nil(t, breakRepo.activeBreak)
}

func TestStartBreak_NotClockedIn(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.StartBreak(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrNotClockedIn)
}

func TestEndBreak_NotOnBreak(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	employeeID := uuid.New()

	// Clock in first
	_, _, err := svc.ClockIn(ctx, uuid.New(), employeeID)
	require.NoError(t, err)

	// Try to end break without starting one
	_, err = svc.EndBreak(ctx, employeeID)
	assert.ErrorIs(t, err, ErrNotOnBreak)
}

func TestClockIn_RestPeriodViolation(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	// Set previous shift ended 5 hours ago (<11h rest period)
	fiveHoursAgo := time.Now().Add(-5 * time.Hour)
	workRepo.previousShiftEnd = &fiveHoursAgo

	entry, checkResult, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, entry) // Still allowed (informational warning)
	assert.NotNil(t, checkResult)
	assert.True(t, checkResult.RestViolation)
	assert.True(t, checkResult.RestHoursActual < 11)
}

func TestClockIn_MaxDailyHoursExceeded(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	// Set daily summary showing 600+ net minutes (10h)
	workRepo.dailySummary = &DailySummary{
		NetWorkMinutes: 600,
	}

	_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	assert.ErrorIs(t, err, ErrMaxDailyHoursExceeded)
}

func TestSubmitTimeCorrection_CreatesPendingEntry(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	// Create original entry
	now := time.Now()
	original := &models.HRWorkTimeEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		ClockIn:    now.Add(-8 * time.Hour),
		Status:     models.WorkTimeStatusCompleted,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	workRepo.entries[original.ID] = original

	clockOut := now.Add(-1 * time.Hour)
	input := CorrectionInput{
		OriginalEntryID:      original.ID,
		CorrectedClockIn:     now.Add(-9 * time.Hour),
		CorrectedClockOut:    clockOut,
		CorrectedBreakMinutes: 30,
		Reason:               "Forgot to clock in on time",
	}

	correction, err := svc.SubmitTimeCorrection(ctx, tenantID, employeeID, input)
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCorrectionPending, correction.Status)
	assert.True(t, correction.IsCorrection)
	assert.Equal(t, &original.ID, correction.OriginalEntryID)
	assert.Equal(t, "Forgot to clock in on time", correction.CorrectionReason)
}

func TestApproveTimeCorrection_TransitionsStatus(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	approverID := uuid.New()

	now := time.Now()
	clockOut := now.Add(-1 * time.Hour)
	netWork := 450
	correction := &models.HRWorkTimeEntry{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		EmployeeID:       uuid.New(),
		ClockIn:          now.Add(-8 * time.Hour),
		ClockOut:         &clockOut,
		NetWorkMinutes:   &netWork,
		Status:           models.WorkTimeStatusCorrectionPending,
		IsCorrection:     true,
		CorrectionReason: "Test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	workRepo.entries[correction.ID] = correction

	approved, err := svc.ApproveTimeCorrection(ctx, correction.ID, approverID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCorrectionApproved, approved.Status)
	assert.Equal(t, &approverID, approved.CorrectionApprovedBy)
	assert.NotNil(t, approved.CorrectionApprovedAt)
}

func TestClockOut_NotClockedIn(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, _, err := svc.ClockOut(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotClockedIn)
}

func TestGetActiveShift_NoShift(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	shift, breaks, err := svc.GetActiveShift(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, shift)
	assert.Nil(t, breaks)
}

func TestGetWorkTimeStatus_NotClockedIn(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	status, err := svc.GetWorkTimeStatus(ctx, uuid.New())
	require.NoError(t, err)
	assert.False(t, status.IsClockedIn)
	assert.False(t, status.IsOnBreak)
	assert.Equal(t, "none", status.ArbZGSeverity)
}

func TestGetWorkTimeStatus_ClockedIn(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	status, err := svc.GetWorkTimeStatus(ctx, employeeID)
	require.NoError(t, err)
	assert.True(t, status.IsClockedIn)
	assert.False(t, status.IsOnBreak)
	assert.NotNil(t, status.CurrentShiftStart)
}
