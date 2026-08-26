package timetracking

import (
	"context"
	"slices"
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

	// Call counters for the batched N+1-free read paths.
	weeklyBatchCalls      int
	activeShiftBatchCalls int
	dailyRangeCalls       int

	// Every tenantID the service handed down, in call order. The queries
	// themselves are pinned against a real database in
	// postgres_tenant_scope_test.go; this records that the service passes the
	// caller's tenant through instead of uuid.Nil.
	seenTenantIDs []uuid.UUID
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

func (m *mockWorkTimeRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.HRWorkTimeEntry, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	entry, ok := m.entries[id]
	if !ok {
		return nil, ErrWorkTimeEntryNotFound
	}
	// The real query filters on tenant_id, so a foreign entry is not found
	// rather than returned to the wrong caller.
	if tenantID != uuid.Nil && entry.TenantID != tenantID {
		return nil, ErrWorkTimeEntryNotFound
	}
	return entry, nil
}

func (m *mockWorkTimeRepo) GetActiveShift(_ context.Context, tenantID, _ uuid.UUID) (*models.HRWorkTimeEntry, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
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

func (m *mockWorkTimeRepo) ApproveCorrection(_ context.Context, correction *models.HRWorkTimeEntry, originalID *uuid.UUID) error {
	m.entries[correction.ID] = correction
	if originalID == nil {
		return nil
	}
	original, ok := m.entries[*originalID]
	if !ok || original.TenantID != correction.TenantID {
		return nil
	}
	if original.Status == models.WorkTimeStatusActive || original.Status == models.WorkTimeStatusCompleted {
		original.Status = models.WorkTimeStatusSuperseded
		original.UpdatedAt = correction.UpdatedAt
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

func (m *mockWorkTimeRepo) GetPreviousShiftEnd(_ context.Context, tenantID, _ uuid.UUID, _ time.Time) (*time.Time, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	return m.previousShiftEnd, nil
}

func (m *mockWorkTimeRepo) GetDailySummary(_ context.Context, tenantID, _ uuid.UUID, _ time.Time) (*DailySummary, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	if m.dailySummary != nil {
		return m.dailySummary, nil
	}
	return &DailySummary{}, nil
}

func (m *mockWorkTimeRepo) GetWeeklySummary(_ context.Context, tenantID, _ uuid.UUID, _ time.Time) (*WeeklySummary, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	if m.weeklySummary != nil {
		return m.weeklySummary, nil
	}
	return &WeeklySummary{}, nil
}

func (m *mockWorkTimeRepo) GetDailySummaryRange(_ context.Context, tenantID, _ uuid.UUID, start time.Time, numDays int) ([]DailySummary, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	m.dailyRangeCalls++
	days := make([]DailySummary, numDays)
	for i := range days {
		days[i] = DailySummary{Date: start.AddDate(0, 0, i)}
		if m.dailySummary != nil {
			days[i].NetWorkMinutes = m.dailySummary.NetWorkMinutes
		}
	}
	return days, nil
}

func (m *mockWorkTimeRepo) GetWeeklySummaryBatch(_ context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID, _ time.Time) (map[uuid.UUID]*WeeklySummary, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	m.weeklyBatchCalls++
	result := make(map[uuid.UUID]*WeeklySummary, len(employeeIDs))
	for _, id := range employeeIDs {
		if m.weeklySummary != nil {
			result[id] = m.weeklySummary
		} else {
			result[id] = &WeeklySummary{}
		}
	}
	return result, nil
}

func (m *mockWorkTimeRepo) GetActiveShiftEmployeeIDs(_ context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	m.seenTenantIDs = append(m.seenTenantIDs, tenantID)
	m.activeShiftBatchCalls++
	result := make(map[uuid.UUID]bool, len(employeeIDs))
	if m.activeShift != nil {
		result[m.activeShift.EmployeeID] = true
	}
	return result, nil
}

func (m *mockWorkTimeRepo) ReserveWorkTimeForInvoice(_ context.Context, _, _ uuid.UUID, _, _ time.Time) (int, []string, error) {
	return 0, nil, nil
}

func (m *mockWorkTimeRepo) ConfirmInvoiceReservation(_ context.Context, _, _ uuid.UUID, _ []string) error {
	return nil
}

func (m *mockWorkTimeRepo) ReleaseInvoiceReservation(_ context.Context, _ uuid.UUID, _ []string) error {
	return nil
}

func (m *mockWorkTimeRepo) GetProjectBreakdown(_ context.Context, _, _ uuid.UUID, _, _ time.Time) ([]ProjectBreakdown, error) {
	return []ProjectBreakdown{}, nil
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

// GetActiveBreak filters on the tenant the caller passed, not just on the
// entry id — otherwise a service that forgets to pass one would still look
// correct here while writing uuid.Nil into the query in production.
func (m *mockBreakRepo) GetActiveBreak(_ context.Context, tenantID, _ uuid.UUID) (*models.HRBreakEntry, error) {
	if m.activeBreak != nil && m.activeBreak.TenantID != tenantID {
		return nil, nil
	}
	return m.activeBreak, nil
}

func (m *mockBreakRepo) Update(_ context.Context, entry *models.HRBreakEntry) error {
	m.breaks[entry.ID] = entry
	if entry.EndTime != nil && m.activeBreak != nil && m.activeBreak.ID == entry.ID {
		m.activeBreak = nil
	}
	return nil
}

func (m *mockBreakRepo) ListByWorkTimeEntry(_ context.Context, tenantID, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error) {
	var result []*models.HRBreakEntry
	for _, b := range m.breaks {
		if b.TenantID == tenantID && b.WorkTimeEntryID == workTimeEntryID {
			result = append(result, b)
		}
	}
	if result == nil {
		result = []*models.HRBreakEntry{}
	}
	return result, nil
}

type mockEmployeeRepo struct {
	profile  *models.EmployeeProfile
	profiles []*models.EmployeeProfile
}

func (m *mockEmployeeRepo) Create(_ context.Context, _ *models.EmployeeProfile) error { return nil }
func (m *mockEmployeeRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.EmployeeProfile, error) {
	return m.profile, nil
}
func (m *mockEmployeeRepo) GetByUserID(_ context.Context, _ uuid.UUID) (*models.EmployeeProfile, error) {
	return m.profile, nil
}
func (m *mockEmployeeRepo) List(_ context.Context, _ employee.EmployeeFilter) ([]*models.EmployeeProfile, int, error) {
	return m.profiles, len(m.profiles), nil
}
func (m *mockEmployeeRepo) Update(_ context.Context, _ *models.EmployeeProfile) error { return nil }
func (m *mockEmployeeRepo) CountOtherActiveRoleAdmins(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockEmployeeRepo) CountDirectReports(_ context.Context, _, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockEmployeeRepo) Offboard(_ context.Context, _ employee.OffboardWrite) (*models.EmployeeProfile, error) {
	return nil, nil
}

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
		TenantID:        tenantID,
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
	breakEntry, err := svc.StartBreak(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.NotNil(t, breakEntry)
	assert.Nil(t, breakEntry.EndTime)
	// hr_break_entries.tenant_id is NOT NULL since migration 000230; a break
	// created without one does not reach the database at all.
	assert.Equal(t, tenantID, breakEntry.TenantID)

	// Can't start another break
	_, err = svc.StartBreak(ctx, tenantID, employeeID)
	assert.ErrorIs(t, err, ErrAlreadyOnBreak)

	// End break
	endedBreak, err := svc.EndBreak(ctx, tenantID, employeeID)
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

	_, err := svc.StartBreak(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotClockedIn)
}

func TestEndBreak_NotOnBreak(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	// Clock in first
	_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
	require.NoError(t, err)

	// Try to end break without starting one
	_, err = svc.EndBreak(ctx, tenantID, employeeID)
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

	approved, err := svc.ApproveTimeCorrection(ctx, correction.TenantID, correction.ID, approverID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCorrectionApproved, approved.Status)
	assert.Equal(t, &approverID, approved.CorrectionApprovedBy)
	assert.NotNil(t, approved.CorrectionApprovedAt)
}

// sumBalanceMinutes mirrors what the daily and weekly aggregates do in SQL: sum
// net_work_minutes over the entries whose status counts towards a balance. It reads
// the same balanceStatuses the queries are parameterised with, so the expectation
// here cannot drift away from the one the database applies.
func sumBalanceMinutes(entries map[uuid.UUID]*models.HRWorkTimeEntry) int {
	var total int
	for _, e := range entries {
		if !slices.Contains(balanceStatuses, string(e.Status)) {
			continue
		}
		if e.NetWorkMinutes != nil {
			total += *e.NetWorkMinutes
		}
	}
	return total
}

func TestApproveTimeCorrection_SupersedesOriginal(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()

	now := time.Now()
	originalNet := 480
	originalOut := now.Add(-1 * time.Hour)
	original := &models.HRWorkTimeEntry{
		ID:             uuid.New(),
		TenantID:       tenantID,
		EmployeeID:     employeeID,
		ClockIn:        now.Add(-9 * time.Hour),
		ClockOut:       &originalOut,
		NetWorkMinutes: &originalNet,
		Status:         models.WorkTimeStatusCompleted,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	workRepo.entries[original.ID] = original

	correctedNet := 360
	correctedOut := now.Add(-3 * time.Hour)
	correction := &models.HRWorkTimeEntry{
		ID:               uuid.New(),
		TenantID:         tenantID,
		EmployeeID:       employeeID,
		ClockIn:          now.Add(-9 * time.Hour),
		ClockOut:         &correctedOut,
		NetWorkMinutes:   &correctedNet,
		Status:           models.WorkTimeStatusCorrectionPending,
		IsCorrection:     true,
		OriginalEntryID:  &original.ID,
		CorrectionReason: "Left three hours earlier than recorded",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	workRepo.entries[correction.ID] = correction

	// A pending correction is a proposal: until it is approved the day still counts
	// the original alone.
	assert.Equal(t, originalNet, sumBalanceMinutes(workRepo.entries))

	approved, err := svc.ApproveTimeCorrection(ctx, correction.TenantID, correction.ID, approverID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCorrectionApproved, approved.Status)

	assert.Equal(t, models.WorkTimeStatusSuperseded, workRepo.entries[original.ID].Status,
		"the replaced original must leave the balance")
	assert.Equal(t, correctedNet, sumBalanceMinutes(workRepo.entries),
		"an approved correction replaces the original, it does not add to it")
}

func TestApproveTimeCorrection_LeavesForeignOriginalAlone(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()

	now := time.Now()
	foreignNet := 480
	foreign := &models.HRWorkTimeEntry{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		EmployeeID:     uuid.New(),
		ClockIn:        now.Add(-9 * time.Hour),
		NetWorkMinutes: &foreignNet,
		Status:         models.WorkTimeStatusCompleted,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	workRepo.entries[foreign.ID] = foreign

	correctedNet := 360
	correction := &models.HRWorkTimeEntry{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		EmployeeID:      uuid.New(),
		ClockIn:         now.Add(-9 * time.Hour),
		NetWorkMinutes:  &correctedNet,
		Status:          models.WorkTimeStatusCorrectionPending,
		IsCorrection:    true,
		OriginalEntryID: &foreign.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	workRepo.entries[correction.ID] = correction

	_, err := svc.ApproveTimeCorrection(ctx, correction.TenantID, correction.ID, uuid.New())
	require.NoError(t, err)

	assert.Equal(t, models.WorkTimeStatusCompleted, workRepo.entries[foreign.ID].Status,
		"a correction must never retire an entry of another tenant")
}

// TestApproveTimeCorrection_EmployeeCanApproveOwnCorrection documents a real
// finding from cov-gateway-hr-time-tracking-paths (BACKLOG.yml): the service
// never compares approverID against the correction's own EmployeeID. Anyone
// holding "hr:write" or the narrower "zeiterfassung:corrections:approve" key
// can submit a correction for themselves and then approve it — the guard
// controls WHO may approve corrections, not WHOSE. Filed as
// fix-hr-approve-correction-self-approval at the backlog end; this test pins
// today's (buggy) behaviour so the fix has something to flip red.
func TestApproveTimeCorrection_EmployeeCanApproveOwnCorrection(t *testing.T) {
	svc, workRepo, _ := newTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	now := time.Now()
	clockOut := now.Add(-1 * time.Hour)
	netWork := 450
	correction := &models.HRWorkTimeEntry{
		ID:               uuid.New(),
		TenantID:         tenantID,
		EmployeeID:       employeeID,
		ClockIn:          now.Add(-8 * time.Hour),
		ClockOut:         &clockOut,
		NetWorkMinutes:   &netWork,
		Status:           models.WorkTimeStatusCorrectionPending,
		IsCorrection:     true,
		CorrectionReason: "Forgot to clock out for lunch",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	workRepo.entries[correction.ID] = correction

	// approverID == employeeID: the correction's own author approves it.
	approved, err := svc.ApproveTimeCorrection(ctx, tenantID, correction.ID, employeeID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkTimeStatusCorrectionApproved, approved.Status)
	assert.Equal(t, &employeeID, approved.CorrectionApprovedBy)
}

// TestGetTimeBalance_FractionalWorkWeek proves the balance subtraction is
// exact for a non-round weekly total and a non-standard work week, not just
// for tidy 5×8h numbers: 6 work days × 8h = 2880 target minutes, 2647 worked
// minutes worked (a Friday cut short by 53 minutes) must yield exactly -233,
// not a rounded neighbour.
func TestGetTimeBalance_FractionalWorkWeek(t *testing.T) {
	workRepo := newMockWorkTimeRepo()
	workRepo.weeklySummary = &WeeklySummary{NetWorkMinutes: 2647}
	breakRepo := newMockBreakRepo()
	empRepo := &mockEmployeeRepo{profile: &models.EmployeeProfile{WorkDaysPerWeek: 6}}
	svc := NewService(workRepo, breakRepo, empRepo, &mockSettingsRepo{}, nil)

	balance, target, _, err := svc.GetTimeBalance(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 2880, target)
	assert.Equal(t, -233, balance)
}

// TestWeekStartOf_AcrossDSTSpringForward pins weekStartOf against Germany's
// 2026 spring DST transition (Sunday 2026-03-29, clocks jump 02:00->03:00).
// A naive Monday-minus-days walk that adds/subtracts calendar days is exactly
// this function's contract (AddDate, not a duration subtraction), so this
// guards against a future rewrite reintroducing a fixed-duration walk that
// would land on the wrong wall-clock hour once a DST jump falls inside the week.
func TestWeekStartOf_AcrossDSTSpringForward(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	// Wednesday 2026-04-01, 14:17 — the week containing the Sunday DST jump.
	t1 := time.Date(2026, 4, 1, 14, 17, 0, 0, berlin)
	got := weekStartOf(t1)

	want := time.Date(2026, 3, 30, 0, 0, 0, 0, berlin)
	assert.True(t, got.Equal(want), "got %v, want %v", got, want)
	assert.Equal(t, time.Monday, got.Weekday())
	assert.Equal(t, 0, got.Hour(), "must land on local midnight, not 23:00 or 01:00 across the DST jump")
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

	shift, breaks, err := svc.GetActiveShift(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, shift)
	assert.Nil(t, breaks)
}

func TestGetWorkTimeStatus_NotClockedIn(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	status, err := svc.GetWorkTimeStatus(ctx, uuid.New(), uuid.New())
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

	status, err := svc.GetWorkTimeStatus(ctx, tenantID, employeeID)
	require.NoError(t, err)
	assert.True(t, status.IsClockedIn)
	assert.False(t, status.IsOnBreak)
	assert.NotNil(t, status.CurrentShiftStart)
}

// TestGetTeamTime_BatchesRepoQueries proves GetTeamTime issues a constant number of
// repository round-trips regardless of employee count (was 1 + 9×N → N+1 / P0).
func TestGetTeamTime_BatchesRepoQueries(t *testing.T) {
	workRepo := newMockWorkTimeRepo()
	breakRepo := newMockBreakRepo()

	var profiles []*models.EmployeeProfile
	for i := 0; i < 5; i++ {
		profiles = append(profiles, &models.EmployeeProfile{
			UserID:          uuid.New(),
			UserName:        "Employee",
			WorkDaysPerWeek: 5,
		})
	}
	empRepo := &mockEmployeeRepo{profiles: profiles}
	svc := NewService(workRepo, breakRepo, empRepo, &mockSettingsRepo{}, nil)

	entries, err := svc.GetTeamTime(context.Background(), uuid.New(), time.Now())
	require.NoError(t, err)
	assert.Len(t, entries, 5)

	// Exactly one batched weekly-summary query and one batched active-shift query,
	// independent of the 5 employees (the naive impl did one of each per employee).
	assert.Equal(t, 1, workRepo.weeklyBatchCalls)
	assert.Equal(t, 1, workRepo.activeShiftBatchCalls)
}

// TestGetTimeAnalytics_BatchesDailyQueries proves the day trend is loaded with a single
// range query instead of one GetDailySummary call per day.
func TestGetTimeAnalytics_BatchesDailyQueries(t *testing.T) {
	svc, workRepo, _ := newTestService()

	_, err := svc.GetTimeAnalytics(context.Background(), uuid.New(), uuid.New(), "week")
	require.NoError(t, err)

	assert.Equal(t, 1, workRepo.dailyRangeCalls)
}

// TestServicePassesCallerTenantToRepo pins the plumbing half of the tenant scope.
// The queries filter on tenant_id (see postgres_tenant_scope_test.go), but that
// only helps if the service hands its caller's tenant down instead of uuid.Nil —
// a zero tenant would match no row and turn every read into a phantom 404.
func TestServicePassesCallerTenantToRepo(t *testing.T) {
	tenantID := uuid.New()
	employeeID := uuid.New()

	t.Run("clock in reads the balance and rest period for the caller's tenant", func(t *testing.T) {
		svc, workRepo, _ := newTestService()
		_, _, err := svc.ClockIn(context.Background(), tenantID, employeeID)
		require.NoError(t, err)
		assertAllTenants(t, workRepo, tenantID)
	})

	t.Run("break and status paths carry the tenant too", func(t *testing.T) {
		svc, workRepo, _ := newTestService()
		ctx := context.Background()
		_, _, err := svc.ClockIn(ctx, tenantID, employeeID)
		require.NoError(t, err)
		_, err = svc.StartBreak(ctx, tenantID, employeeID)
		require.NoError(t, err)
		_, err = svc.EndBreak(ctx, tenantID, employeeID)
		require.NoError(t, err)
		_, err = svc.GetWorkTimeStatus(ctx, tenantID, employeeID)
		require.NoError(t, err)
		assertAllTenants(t, workRepo, tenantID)
	})

	t.Run("approving a correction scopes the lookup", func(t *testing.T) {
		svc, workRepo, _ := newTestService()
		now := time.Now()
		correction := &models.HRWorkTimeEntry{
			ID:              uuid.New(),
			TenantID:        tenantID,
			EmployeeID:      employeeID,
			ClockIn:         now.Add(-3 * time.Hour),
			ClockOut:        &now,
			Status:          models.WorkTimeStatusCorrectionPending,
			IsCorrection:    true,
			NetWorkMinutes:  ptrTo(180),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		workRepo.entries[correction.ID] = correction

		_, err := svc.ApproveTimeCorrection(context.Background(), tenantID, correction.ID, uuid.New())
		require.NoError(t, err)
		assertAllTenants(t, workRepo, tenantID)
	})

	t.Run("a foreign tenant cannot approve the correction", func(t *testing.T) {
		svc, workRepo, _ := newTestService()
		now := time.Now()
		correction := &models.HRWorkTimeEntry{
			ID:           uuid.New(),
			TenantID:     tenantID,
			EmployeeID:   employeeID,
			ClockIn:      now.Add(-3 * time.Hour),
			ClockOut:     &now,
			Status:       models.WorkTimeStatusCorrectionPending,
			IsCorrection: true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		workRepo.entries[correction.ID] = correction

		_, err := svc.ApproveTimeCorrection(context.Background(), uuid.New(), correction.ID, uuid.New())
		assert.ErrorIs(t, err, ErrWorkTimeEntryNotFound)
		assert.Equal(t, models.WorkTimeStatusCorrectionPending, correction.Status,
			"a foreign approval must leave the correction untouched")
	})
}

func assertAllTenants(t *testing.T, repo *mockWorkTimeRepo, want uuid.UUID) {
	t.Helper()
	require.NotEmpty(t, repo.seenTenantIDs, "no tenant-scoped repo call was made")
	for i, got := range repo.seenTenantIDs {
		assert.Equal(t, want, got, "repo call %d received the wrong tenant", i)
	}
}

func ptrTo[T any](v T) *T { return &v }
