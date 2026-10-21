package timetracking_test

// Day and week buckets follow the tenant's hr_company_settings.timezone, not the
// zone of the process (UTC in the containers). These tests run the real service
// over the real repository: the bug lived in the seam between the two — the
// service handed over UTC-anchored dates and the repository bucketed on them.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type fixedZoneSettings struct{ timezone string }

func (f fixedZoneSettings) GetByTenant(context.Context, uuid.UUID) (*models.HRCompanySettings, error) {
	return &models.HRCompanySettings{Timezone: f.timezone}, nil
}

func (fixedZoneSettings) Upsert(context.Context, *models.HRCompanySettings) error { return nil }

type tzFixture struct {
	svc        *timetracking.Service
	ctx        context.Context
	employeeID uuid.UUID
}

func newTZFixture(t *testing.T, timezone string, clockIns map[string]time.Duration) *tzFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	employeeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "timetz-" + uuid.NewString() + "@test.local",
		"password_hash": "x",
		"first_name":    "Zone",
		"last_name":     "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", employeeID) })

	for rfc3339, worked := range clockIns {
		in, err := time.Parse(time.RFC3339, rfc3339)
		require.NoError(t, err)
		id := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
			"id":               uuid.New(),
			"tenant_id":        testutil.TenantA,
			"employee_id":      employeeID,
			"clock_in":         in,
			"clock_out":        in.Add(worked),
			"break_minutes":    0,
			"net_work_minutes": int(worked.Minutes()),
			"status":           "completed",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_work_time_entries", id) })
	}

	svc := timetracking.NewService(timetracking.NewPostgresWorkTimeRepo(pool), nil, nil, fixedZoneSettings{timezone}, nil)
	return &tzFixture{svc: svc, ctx: testutil.WithTenantCtx(context.Background(), testutil.TenantA), employeeID: employeeID}
}

func parseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s) // exactly what the gRPC layer does: midnight UTC
	require.NoError(t, err)
	return d
}

// A night shift starting 00:30 Berlin time on Monday 2026-03-30 is 22:30 UTC on
// Sunday 03-29 — the day the clocks went forward. It belongs to Monday and to
// the week starting 03-30, not to Sunday and the week before.
func TestTimezone_NightShiftAfterMidnightBelongsToLocalDayAndWeek(t *testing.T) {
	f := newTZFixture(t, "Europe/Berlin", map[string]time.Duration{
		"2026-03-30T00:30:00+02:00": 6 * time.Hour,    // Monday, 00:30 local
		"2026-03-29T23:30:00+02:00": 30 * time.Minute, // Sunday, last half hour of the 23h DST day
	})

	monday, err := f.svc.GetDailySummary(f.ctx, testutil.TenantA, f.employeeID, parseDate(t, "2026-03-30"))
	require.NoError(t, err)
	assert.Equal(t, 1, monday.EntryCount, "00:30 local is Monday")
	assert.Equal(t, 360, monday.NetWorkMinutes)

	sunday, err := f.svc.GetDailySummary(f.ctx, testutil.TenantA, f.employeeID, parseDate(t, "2026-03-29"))
	require.NoError(t, err)
	assert.Equal(t, 1, sunday.EntryCount, "23:30 local is still Sunday, although the day has only 23 hours")
	assert.Equal(t, 30, sunday.NetWorkMinutes)

	thisWeek, err := f.svc.GetWeeklySummary(f.ctx, testutil.TenantA, f.employeeID, parseDate(t, "2026-03-30"))
	require.NoError(t, err)
	assert.Equal(t, 360, thisWeek.NetWorkMinutes, "Monday 00:30 local counts for the week starting 03-30")
	assert.Equal(t, 1, thisWeek.Days[0].EntryCount)

	lastWeek, err := f.svc.GetWeeklySummary(f.ctx, testutil.TenantA, f.employeeID, parseDate(t, "2026-03-23"))
	require.NoError(t, err)
	assert.Equal(t, 30, lastWeek.NetWorkMinutes, "only the Sunday 23:30 entry belongs to the week before")
	assert.Equal(t, 1, lastWeek.Days[6].EntryCount, "Sunday of the DST week is day index 6")
}

// The zone is read from the tenant settings: the same instant is on different
// calendar days in New York and in Berlin.
func TestTimezone_FollowsTenantSetting(t *testing.T) {
	const instant = "2026-06-10T23:30:00Z" // 19:30 on 06-10 in New York, 01:30 on 06-11 in Berlin
	entries := map[string]time.Duration{instant: time.Hour}

	ny := newTZFixture(t, "America/New_York", entries)
	d, err := ny.svc.GetDailySummary(ny.ctx, testutil.TenantA, ny.employeeID, parseDate(t, "2026-06-10"))
	require.NoError(t, err)
	assert.Equal(t, 1, d.EntryCount)

	berlin := newTZFixture(t, "Europe/Berlin", entries)
	d, err = berlin.svc.GetDailySummary(berlin.ctx, testutil.TenantA, berlin.employeeID, parseDate(t, "2026-06-10"))
	require.NoError(t, err)
	assert.Equal(t, 0, d.EntryCount)
	d, err = berlin.svc.GetDailySummary(berlin.ctx, testutil.TenantA, berlin.employeeID, parseDate(t, "2026-06-11"))
	require.NoError(t, err)
	assert.Equal(t, 1, d.EntryCount)
}

// An unknown zone name must not degrade to UTC: it falls back to Europe/Berlin.
func TestTimezone_UnknownZoneFallsBackToBerlin(t *testing.T) {
	f := newTZFixture(t, "Mars/Olympus_Mons", map[string]time.Duration{
		"2026-03-30T00:30:00+02:00": time.Hour,
	})
	d, err := f.svc.GetDailySummary(f.ctx, testutil.TenantA, f.employeeID, parseDate(t, "2026-03-30"))
	require.NoError(t, err)
	assert.Equal(t, 1, d.EntryCount)
}
