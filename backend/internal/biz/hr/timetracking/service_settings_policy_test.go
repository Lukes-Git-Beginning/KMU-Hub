package timetracking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/models"
)

// erroringSettingsRepo fails every read, to prove the policy lookup degrades
// to the defaults instead of failing time tracking.
type erroringSettingsRepo struct{ err error }

func (r *erroringSettingsRepo) GetByTenant(context.Context, uuid.UUID) (*models.HRCompanySettings, error) {
	return nil, r.err
}

func (r *erroringSettingsRepo) Upsert(context.Context, *models.HRCompanySettings) error { return nil }

func newPolicyTestService(settingsRepo leave.HRSettingsRepository) (*Service, *mockWorkTimeRepo) {
	workRepo := newMockWorkTimeRepo()
	empRepo := &mockEmployeeRepo{profile: &models.EmployeeProfile{WorkDaysPerWeek: 5}}
	return NewService(workRepo, newMockBreakRepo(), empRepo, settingsRepo, nil), workRepo
}

func withSettings(s models.HRCompanySettings) *mockSettingsRepo {
	return &mockSettingsRepo{settings: &s}
}

// The same worked week must produce a different balance once the tenant
// changes work_hours_per_day — before the fix the setting had no effect.
func TestGetTimeBalance_UsesTenantWorkHoursPerDay(t *testing.T) {
	ctx := context.Background()

	defaultSvc, defaultRepo := newPolicyTestService(&mockSettingsRepo{})
	defaultRepo.weeklySummary = &WeeklySummary{NetWorkMinutes: 1900}
	balance, target, _, err := defaultSvc.GetTimeBalance(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 2400, target, "no configured hours -> 5 days x 8h")
	assert.Equal(t, -500, balance)

	sixHourSvc, sixHourRepo := newPolicyTestService(withSettings(models.HRCompanySettings{WorkHoursPerDay: 6}))
	sixHourRepo.weeklySummary = &WeeklySummary{NetWorkMinutes: 1900}
	balance, target, _, err = sixHourSvc.GetTimeBalance(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1800, target, "5 days x 6h")
	assert.Equal(t, 100, balance)
}

func TestGetTimeAnalytics_UsesTenantWorkHoursPerDay(t *testing.T) {
	ctx := context.Background()

	defaultSvc, defaultRepo := newPolicyTestService(&mockSettingsRepo{})
	defaultRepo.dailySummary = &DailySummary{NetWorkMinutes: 400}
	base, err := defaultSvc.GetTimeAnalytics(ctx, uuid.New(), uuid.New(), "week")
	require.NoError(t, err)

	svc, repo := newPolicyTestService(withSettings(models.HRCompanySettings{WorkHoursPerDay: 6}))
	repo.dailySummary = &DailySummary{NetWorkMinutes: 400}
	got, err := svc.GetTimeAnalytics(ctx, uuid.New(), uuid.New(), "week")
	require.NoError(t, err)

	// 7 days x 400 min = 2800 worked in both cases.
	assert.Equal(t, 2800, base.TotalMinutes)
	assert.Equal(t, 2800, got.TotalMinutes)
	// Target is pro-rated over 7 days: 2400*7/5 = 3360 vs 1800*7/5 = 2520.
	assert.Equal(t, 3360, base.TargetMinutes)
	assert.Equal(t, 0, base.OvertimeMinutes)
	assert.Equal(t, 2520, got.TargetMinutes)
	assert.Equal(t, 280, got.OvertimeMinutes)
	require.NotEmpty(t, got.DayTrend)
	assert.Equal(t, 480, base.DayTrend[0].TargetMinutes)
	assert.Equal(t, 360, got.DayTrend[0].TargetMinutes)
}

func TestGetTeamTime_UsesTenantWorkHoursPerDay(t *testing.T) {
	workRepo := newMockWorkTimeRepo()
	workRepo.weeklySummary = &WeeklySummary{NetWorkMinutes: 2000}
	empRepo := &mockEmployeeRepo{profiles: []*models.EmployeeProfile{
		{UserID: uuid.New(), UserName: "A", WorkDaysPerWeek: 5},
	}}
	svc := NewService(workRepo, newMockBreakRepo(), empRepo,
		withSettings(models.HRCompanySettings{WorkHoursPerDay: 6}), nil)

	entries, err := svc.GetTeamTime(context.Background(), uuid.New(), time.Now())
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 1800, entries[0].TargetMinutes)
	assert.Equal(t, 200, entries[0].OvertimeMinutes)
}

func TestClockIn_UsesTenantMaxDailyHours(t *testing.T) {
	ctx := context.Background()

	// 8h worked: fine under the default 10h ceiling ...
	defaultSvc, defaultRepo := newPolicyTestService(&mockSettingsRepo{})
	defaultRepo.dailySummary = &DailySummary{NetWorkMinutes: 480}
	_, _, err := defaultSvc.ClockIn(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)

	// ... but blocked once the tenant tightens the maximum to 8h.
	svc, repo := newPolicyTestService(withSettings(models.HRCompanySettings{MaxDailyHours: 8}))
	repo.dailySummary = &DailySummary{NetWorkMinutes: 480}
	_, _, err = svc.ClockIn(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrMaxDailyHoursExceeded)
	assert.Empty(t, repo.entries, "a refused clock-in must not create an entry")
}

// The setting may only tighten the ArbZG §3 ceiling, never lift it.
func TestClockIn_MaxDailyHoursAboveArbZGIsCappedAtTen(t *testing.T) {
	svc, repo := newPolicyTestService(withSettings(models.HRCompanySettings{MaxDailyHours: 12}))
	repo.dailySummary = &DailySummary{NetWorkMinutes: 600}

	_, _, err := svc.ClockIn(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrMaxDailyHoursExceeded)
}

// UpdateHRSettings stores omitted fields as 0, a new tenant has no row, and a
// read can fail: all three fall back to 8h/10h instead of a zero target (every
// minute would count as overtime) or a zero maximum (no clock-in at all).
func TestWorkHoursPolicy_FallsBackToDefaults(t *testing.T) {
	cases := map[string]leave.HRSettingsRepository{
		"nil repo":       nil,
		"zero values":    withSettings(models.HRCompanySettings{}),
		"no settings":    &erroringSettingsRepo{err: leave.ErrSettingsNotFound},
		"read failure":   &erroringSettingsRepo{err: errors.New("connection reset")},
		"negative values": withSettings(models.HRCompanySettings{WorkHoursPerDay: -1, MaxDailyHours: -1}),
	}
	for name, repo := range cases {
		t.Run(name, func(t *testing.T) {
			svc, _ := newPolicyTestService(repo)
			workHours, maxHours := svc.workHoursPolicy(context.Background(), uuid.New())
			assert.Equal(t, 8, workHours)
			assert.Equal(t, 10, maxHours)
		})
	}
}
