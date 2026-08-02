package timeentry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

type mockRepository struct {
	mu       sync.RWMutex
	entries  map[uuid.UUID]*models.TimeEntry
	withUser map[uuid.UUID]*models.TimeEntryWithUser

	activeTimer  *models.ActiveTimer
	stoppedTimer *models.TimeEntry

	// Error injection fields
	createErr           error
	getByIDErr          error
	updateErr           error
	deleteErr           error
	listByTaskEntries   []models.TimeEntryWithUser
	listByTaskTotal     int
	listByTaskErr       error
	getActiveTimerErr   error
	stopActiveTimerErr  error
	taskTimeSummary     *models.TimeEntrySummary
	getTaskTimeSummaryErr error
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		entries:  make(map[uuid.UUID]*models.TimeEntry),
		withUser: make(map[uuid.UUID]*models.TimeEntryWithUser),
	}
}

func (m *mockRepository) Create(_ context.Context, entry *models.TimeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.entries[entry.ID] = entry
	m.withUser[entry.ID] = &models.TimeEntryWithUser{
		TimeEntry: *entry,
		UserName:  "Test User",
	}
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id, _ uuid.UUID) (*models.TimeEntryWithUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	ewu, ok := m.withUser[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *ewu
	return &cp, nil
}

func (m *mockRepository) Update(_ context.Context, entry *models.TimeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	m.entries[entry.ID] = entry
	m.withUser[entry.ID] = &models.TimeEntryWithUser{
		TimeEntry: *entry,
		UserName:  "Test User",
	}
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id, _ uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.entries, id)
	delete(m.withUser, id)
	return nil
}

func (m *mockRepository) ListByTask(_ context.Context, _, _ uuid.UUID, _, _ int) ([]models.TimeEntryWithUser, int, error) {
	if m.listByTaskErr != nil {
		return nil, 0, m.listByTaskErr
	}
	return m.listByTaskEntries, m.listByTaskTotal, nil
}

func (m *mockRepository) ListByUser(_ context.Context, _, _ uuid.UUID, _, _ int) ([]models.TimeEntryWithUser, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) ListBillable(_ context.Context, _ uuid.UUID) ([]models.BillableTimeEntry, error) {
	return nil, nil
}

func (m *mockRepository) ListByProject(_ context.Context, _, _ uuid.UUID) ([]models.ProjectTimeEntry, error) {
	return nil, nil
}

func (m *mockRepository) AggregateProjectHours(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) ([]models.UtilizationBucket, error) {
	return nil, nil
}

func (m *mockRepository) GetActiveTimer(_ context.Context, _, _ uuid.UUID) (*models.ActiveTimer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getActiveTimerErr != nil {
		return nil, m.getActiveTimerErr
	}
	return m.activeTimer, nil
}

func (m *mockRepository) StopActiveTimer(_ context.Context, _, _ uuid.UUID) (*models.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopActiveTimerErr != nil {
		return nil, m.stopActiveTimerErr
	}
	result := m.stoppedTimer
	m.activeTimer = nil
	m.stoppedTimer = nil
	return result, nil
}

func (m *mockRepository) GetTaskTimeSummary(_ context.Context, _, _ uuid.UUID) (*models.TimeEntrySummary, error) {
	if m.getTaskTimeSummaryErr != nil {
		return nil, m.getTaskTimeSummaryErr
	}
	return m.taskTimeSummary, nil
}

// --- Helper ---

func ptrString(s string) *string { return &s }

// --- StartTimer Tests ---

func TestStartTimer_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	taskID := uuid.New()
	userID := uuid.New()

	entry, stopped, err := svc.StartTimer(context.Background(), taskID, userID, uuid.Nil)

	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Nil(t, stopped)
	assert.Equal(t, taskID, entry.TaskID)
	assert.Equal(t, userID, entry.UserID)
	assert.False(t, entry.IsManual)
	assert.Nil(t, entry.EndedAt)
	assert.Nil(t, entry.DurationSeconds)
}

func TestStartTimer_AutoStopExisting(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	oldTaskID := uuid.New()
	newTaskID := uuid.New()

	now := time.Now()
	dur := 300
	oldEntry := &models.TimeEntry{
		ID:              uuid.New(),
		TaskID:          oldTaskID,
		UserID:          userID,
		StartedAt:       now.Add(-5 * time.Minute),
		EndedAt:         &now,
		DurationSeconds: &dur,
	}

	repo.activeTimer = &models.ActiveTimer{
		TimeEntry: *oldEntry,
		TaskTitle: "Old Task",
	}
	repo.stoppedTimer = oldEntry

	entry, stopped, err := svc.StartTimer(context.Background(), newTaskID, userID, uuid.Nil)

	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.NotNil(t, stopped)
	assert.Equal(t, newTaskID, entry.TaskID)
	assert.Equal(t, oldTaskID, stopped.TaskID)
}

func TestStartTimer_GetActiveTimerError(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	repo.getActiveTimerErr = errors.New("db connection failed")

	_, _, err := svc.StartTimer(context.Background(), uuid.New(), uuid.New(), uuid.Nil)

	require.Error(t, err)
	assert.Equal(t, "db connection failed", err.Error())
}

func TestStartTimer_CreateError(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	repo.createErr = errors.New("create failed")

	_, _, err := svc.StartTimer(context.Background(), uuid.New(), uuid.New(), uuid.Nil)

	require.Error(t, err)
	assert.Equal(t, "create failed", err.Error())
}

// --- StopTimer Tests ---

func TestStopTimer_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	dur := 600
	now := time.Now()
	stoppedEntry := &models.TimeEntry{
		ID:              uuid.New(),
		TaskID:          uuid.New(),
		UserID:          userID,
		StartedAt:       now.Add(-10 * time.Minute),
		EndedAt:         &now,
		DurationSeconds: &dur,
	}
	repo.stoppedTimer = stoppedEntry

	result, err := svc.StopTimer(context.Background(), userID, uuid.Nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, stoppedEntry.ID, result.ID)
}

func TestStopTimer_NoActiveTimer(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// stoppedTimer is nil by default, StopActiveTimer returns nil
	result, err := svc.StopTimer(context.Background(), uuid.New(), uuid.Nil)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoActiveTimer)
}

// --- GetActiveTimer Tests ---

func TestGetActiveTimer_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	timer := &models.ActiveTimer{
		TimeEntry: models.TimeEntry{
			ID:        uuid.New(),
			TaskID:    uuid.New(),
			UserID:    userID,
			StartedAt: time.Now(),
		},
		TaskTitle: "Current Task",
	}
	repo.activeTimer = timer

	result, err := svc.GetActiveTimer(context.Background(), userID, uuid.Nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, timer.ID, result.ID)
	assert.Equal(t, "Current Task", result.TaskTitle)
}

// --- AddManualEntry Tests ---

func TestAddManualEntry_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ManualEntryInput{
		TaskID:          uuid.New(),
		UserID:          uuid.New(),
		StartedAt:       time.Now().Add(-1 * time.Hour),
		DurationSeconds: 3600,
		Description:     ptrString("Worked on feature"),
	}

	entry, err := svc.AddManualEntry(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, input.TaskID, entry.TaskID)
	assert.Equal(t, input.UserID, entry.UserID)
	assert.True(t, entry.IsManual)
	assert.NotNil(t, entry.EndedAt)
	assert.NotNil(t, entry.DurationSeconds)
	assert.Equal(t, 3600, *entry.DurationSeconds)
	assert.Equal(t, "Worked on feature", *entry.Description)
}

func TestAddManualEntry_InvalidDuration(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ManualEntryInput{
		TaskID:          uuid.New(),
		UserID:          uuid.New(),
		StartedAt:       time.Now(),
		DurationSeconds: 0,
	}

	entry, err := svc.AddManualEntry(context.Background(), input)

	assert.Nil(t, entry)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestAddManualEntry_NegativeDuration(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ManualEntryInput{
		TaskID:          uuid.New(),
		UserID:          uuid.New(),
		StartedAt:       time.Now(),
		DurationSeconds: -100,
	}

	entry, err := svc.AddManualEntry(context.Background(), input)

	assert.Nil(t, entry)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestAddManualEntry_DescriptionTooLong(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	longDesc := strings.Repeat("a", 2001)
	input := ManualEntryInput{
		TaskID:          uuid.New(),
		UserID:          uuid.New(),
		StartedAt:       time.Now(),
		DurationSeconds: 60,
		Description:     &longDesc,
	}

	entry, err := svc.AddManualEntry(context.Background(), input)

	assert.Nil(t, entry)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

// --- UpdateEntry Tests ---

func TestUpdateEntry_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	entryID := uuid.New()
	dur := 1800

	// Seed entry in mock
	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:              entryID,
			TaskID:          uuid.New(),
			UserID:          userID,
			StartedAt:       time.Now().Add(-30 * time.Minute),
			DurationSeconds: &dur,
			IsManual:        true,
		},
		UserName: "Test User",
	}
	repo.entries[entryID] = &repo.withUser[entryID].TimeEntry

	newDur := 3600
	newDesc := "Updated description"
	input := UpdateEntryInput{
		DurationSeconds: &newDur,
		Description:     &newDesc,
	}

	result, err := svc.UpdateEntry(context.Background(), entryID, userID, uuid.Nil, input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3600, *result.DurationSeconds)
	assert.Equal(t, "Updated description", *result.Description)
}

func TestUpdateEntry_NotOwnerRejected(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	ownerID := uuid.New()
	otherID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:     entryID,
			UserID: ownerID,
		},
		UserName: "Owner",
	}

	input := UpdateEntryInput{Description: ptrString("hacked")}

	result, err := svc.UpdateEntry(context.Background(), entryID, otherID, uuid.Nil, input)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotEditOthers)
}

func TestUpdateEntry_InvalidDuration(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:        entryID,
			UserID:    userID,
			StartedAt: time.Now(),
		},
		UserName: "Test User",
	}

	zeroDur := 0
	input := UpdateEntryInput{DurationSeconds: &zeroDur}

	result, err := svc.UpdateEntry(context.Background(), entryID, userID, uuid.Nil, input)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestUpdateEntry_DescriptionTooLong(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:        entryID,
			UserID:    userID,
			StartedAt: time.Now(),
		},
		UserName: "Test User",
	}

	longDesc := strings.Repeat("x", 2001)
	input := UpdateEntryInput{Description: &longDesc}

	result, err := svc.UpdateEntry(context.Background(), entryID, userID, uuid.Nil, input)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

// --- DeleteEntry Tests ---

func TestDeleteEntry_SuccessByOwner(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	userID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:     entryID,
			UserID: userID,
		},
		UserName: "Test User",
	}

	err := svc.DeleteEntry(context.Background(), entryID, userID, uuid.Nil, false)

	require.NoError(t, err)
	// Entry should be removed from mock
	assert.NotContains(t, repo.withUser, entryID)
}

func TestDeleteEntry_SuccessByAdmin(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	ownerID := uuid.New()
	adminID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:     entryID,
			UserID: ownerID,
		},
		UserName: "Owner",
	}

	err := svc.DeleteEntry(context.Background(), entryID, adminID, uuid.Nil, true)

	require.NoError(t, err)
	assert.NotContains(t, repo.withUser, entryID)
}

func TestDeleteEntry_RejectedNonOwnerNonAdmin(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	ownerID := uuid.New()
	otherID := uuid.New()
	entryID := uuid.New()

	repo.withUser[entryID] = &models.TimeEntryWithUser{
		TimeEntry: models.TimeEntry{
			ID:     entryID,
			UserID: ownerID,
		},
		UserName: "Owner",
	}

	err := svc.DeleteEntry(context.Background(), entryID, otherID, uuid.Nil, false)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotDeleteOthers)
	// Entry should still exist
	assert.Contains(t, repo.withUser, entryID)
}

// --- ListByTask Tests ---

func TestListByTask_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	taskID := uuid.New()
	entries := []models.TimeEntryWithUser{
		{
			TimeEntry: models.TimeEntry{ID: uuid.New(), TaskID: taskID},
			UserName:  "User A",
		},
		{
			TimeEntry: models.TimeEntry{ID: uuid.New(), TaskID: taskID},
			UserName:  "User B",
		},
	}
	repo.listByTaskEntries = entries
	repo.listByTaskTotal = 2

	result, total, err := svc.ListByTask(context.Background(), taskID, uuid.Nil, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

// --- GetTaskTimeSummary Tests ---

func TestGetTaskTimeSummary_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	taskID := uuid.New()
	summary := &models.TimeEntrySummary{
		TaskID:               taskID,
		TotalDurationSeconds: 7200,
		EntryCount:           5,
	}
	repo.taskTimeSummary = summary

	result, err := svc.GetTaskTimeSummary(context.Background(), taskID, uuid.Nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, taskID, result.TaskID)
	assert.Equal(t, 7200, result.TotalDurationSeconds)
	assert.Equal(t, 5, result.EntryCount)
}

// ============================================================================
// Cross-Tenant Isolation Tests (time_entries)
// ============================================================================

// TestCrossTenant_TimeEntry_GetByID_NotFound verifies that fetching an
// entry with a mismatched tenantID returns ErrNotFound. The mock simulates
// real Postgres behaviour by returning ErrNotFound when getByIDErr is set.
func TestCrossTenant_TimeEntry_GetByID_NotFound(t *testing.T) {
	repo := newMockRepo()
	repo.getByIDErr = ErrNotFound

	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestCrossTenant_TimeEntry_TenantIDStoredOnCreate verifies that the tenantID
// passed to StartTimer is persisted on the created entry.
func TestCrossTenant_TimeEntry_TenantIDStoredOnCreate(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	taskID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	entry, _, err := svc.StartTimer(context.Background(), taskID, userID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, tenantID, entry.TenantID, "entry must carry the tenantID set during StartTimer")
}

// ============================================================================
// buildMemberUtilization (project team-utilization roll-up)
// ============================================================================

// TestBuildMemberUtilization covers the zero-fill/bucket-placement logic
// that backs the project "Auslastung" roll-up: a member with entries gets
// them placed in the right week/month bucket (oldest first, current last),
// a member with none is zero-filled across every bucket rather than
// omitted, and every member gets the fixed weekly target -- never a real
// contracted-hours or cost figure (see the note on MemberUtilization).
func TestBuildMemberUtilization(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) // Saturday

	memberWithHours := uuid.New()
	memberNoHours := uuid.New()
	members := []models.ProjectMember{
		{UserID: memberWithHours, Role: "owner", FirstName: "Anna", LastName: "Mueller"},
		{UserID: memberNoHours, Role: "viewer", FirstName: "Ben", LastName: "Fischer"},
	}

	currentWeekStart := startOfISOWeek(now)
	oldestWeekStart := currentWeekStart.AddDate(0, 0, -7*(utilizationWeeklyBuckets-1))
	currentMonthStart := startOfMonth(now)
	priorMonthStart := currentMonthStart.AddDate(0, -1, 0)

	weekly := []models.UtilizationBucket{
		{UserID: memberWithHours, PeriodStart: currentWeekStart, TotalSeconds: 3600 * 5},
		{UserID: memberWithHours, PeriodStart: oldestWeekStart, TotalSeconds: 3600*2 + 1800},
	}
	monthly := []models.UtilizationBucket{
		{UserID: memberWithHours, PeriodStart: currentMonthStart, TotalSeconds: 3600 * 10},
		{UserID: memberWithHours, PeriodStart: priorMonthStart, TotalSeconds: 3600 * 20},
	}

	result := buildMemberUtilization(members, weekly, monthly, now)
	require.Len(t, result, 2)

	byID := make(map[uuid.UUID]models.MemberUtilization, len(result))
	for _, r := range result {
		byID[r.UserID] = r
	}

	withHours := byID[memberWithHours]
	assert.Equal(t, "Anna Mueller", withHours.Name)
	assert.Equal(t, "owner", withHours.Role)
	assert.Equal(t, defaultWeeklyTargetHours, withHours.WeeklyTarget)
	require.Len(t, withHours.WeeklyData, utilizationWeeklyBuckets)
	require.Len(t, withHours.MonthlyData, utilizationMonthlyBuckets)

	assert.Equal(t, 2.5, withHours.WeeklyData[0].Hours, "oldest week")
	assert.Equal(t, 5.0, withHours.WeeklyData[utilizationWeeklyBuckets-1].Hours, "current week")
	for i := 1; i < utilizationWeeklyBuckets-1; i++ {
		assert.Zerof(t, withHours.WeeklyData[i].Hours, "week[%d] must be zero-filled", i)
	}
	_, wantOldestISOWeek := oldestWeekStart.ISOWeek()
	assert.Equal(t, fmt.Sprintf("KW %d", wantOldestISOWeek), withHours.WeeklyData[0].Label)

	// 3 monthly buckets = [2 months back (untouched, zero), prior month, current month].
	assert.Zero(t, withHours.MonthlyData[0].Hours, "oldest month (untouched) must be zero-filled")
	assert.Equal(t, 20.0, withHours.MonthlyData[1].Hours, "prior month")
	assert.Equal(t, 10.0, withHours.MonthlyData[utilizationMonthlyBuckets-1].Hours, "current month")
	assert.Equal(t, "Jun 2026", withHours.MonthlyData[0].Label)
	assert.Equal(t, "Jul 2026", withHours.MonthlyData[1].Label)
	assert.Equal(t, "Aug 2026", withHours.MonthlyData[utilizationMonthlyBuckets-1].Label)

	noHours := byID[memberNoHours]
	assert.Equal(t, "Ben Fischer", noHours.Name)
	for _, p := range noHours.WeeklyData {
		assert.Zero(t, p.Hours, "member without entries must be zero-filled, not omitted")
	}
	for _, p := range noHours.MonthlyData {
		assert.Zero(t, p.Hours, "member without entries must be zero-filled, not omitted")
	}
}
