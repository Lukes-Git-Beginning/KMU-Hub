package timetracking

// Day and week boundaries are calendar concepts of the tenant, not of the
// process. The containers run in UTC, so a boundary built from time.Now() or from
// a parsed "2006-01-02" date lands one or two hours off: a night shift starting at
// 00:30 local time was counted for the previous day, and a Monday 00:30 entry for
// the previous week — in the daily total, the break duty, the weekly balance and
// the week lock alike.

import (
	"context"
	"errors"
	"log/slog"
	"time"
	_ "time/tzdata" // the alpine runtime image ships no zoneinfo; LoadLocation would fail there

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
)

// defaultTenantTimezone matches the column default of hr_company_settings.timezone.
const defaultTenantTimezone = "Europe/Berlin"

// tenantLocation returns the zone the tenant's days and weeks are counted in
// (hr_company_settings.timezone). A missing repo, row or unknown zone name falls
// back to Europe/Berlin — never silently to UTC, which would reintroduce the bug.
//
// lean: one settings read per call, like workHoursPolicy; cache per tenant when
// the extra query shows up in a profile.
func (s *Service) tenantLocation(ctx context.Context, tenantID uuid.UUID) *time.Location {
	name := defaultTenantTimezone
	if s.settingsRepo != nil {
		settings, err := s.settingsRepo.GetByTenant(ctx, tenantID)
		switch {
		case err != nil && !errors.Is(err, leave.ErrSettingsNotFound):
			slog.Warn("failed to load HR settings timezone, using default",
				"tenant_id", tenantID, "error", err)
		case err == nil && settings != nil && settings.Timezone != "":
			name = settings.Timezone
		}
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		slog.Warn("unknown HR timezone, using default",
			"tenant_id", tenantID, "timezone", name, "error", err)
		if loc, err = time.LoadLocation(defaultTenantTimezone); err != nil {
			return time.UTC
		}
	}
	return loc
}

// localInstant expresses an instant (a clock-in, time.Now()) in the tenant zone,
// so Year/Month/Day/Weekday read the tenant's calendar.
func (s *Service) localInstant(ctx context.Context, tenantID uuid.UUID, t time.Time) time.Time {
	return t.In(s.tenantLocation(ctx, tenantID))
}

// localDate re-anchors a calendar date (a parsed "2006-01-02", midnight UTC) to
// midnight of the same date in the tenant zone. The date keeps its Y/M/D; only
// the instant it stands for moves.
func (s *Service) localDate(ctx context.Context, tenantID uuid.UUID, d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, s.tenantLocation(ctx, tenantID))
}

// weekStartForInstant is the Monday 00:00 (tenant zone) of the week an instant falls in.
func (s *Service) weekStartForInstant(ctx context.Context, tenantID uuid.UUID, t time.Time) time.Time {
	return weekStartOf(s.localInstant(ctx, tenantID, t))
}

// weekStartForDate is the Monday 00:00 (tenant zone) of the week a calendar date falls in.
func (s *Service) weekStartForDate(ctx context.Context, tenantID uuid.UUID, d time.Time) time.Time {
	return weekStartOf(s.localDate(ctx, tenantID, d))
}
