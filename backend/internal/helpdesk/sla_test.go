package helpdesk

import (
	"testing"
	"time"
)

// TestComputeStatus_AcrossWeekend documents current behaviour: SLA due dates
// are flat calendar-time additions (see ApplyPolicy's doc comment --
// business-hours subtraction is not implemented, 24/7 availability is
// assumed). A due_at set on a Friday evening is breached over the weekend
// exactly like any other 48h window, with no business-hours skip.
func TestComputeStatus_AcrossWeekend(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	// Friday 2026-08-28 18:00 CEST, window 2880 min (48h) -> due Sunday
	// 2026-08-30 18:00 CEST. FirstResponseMins is the active window here
	// (ticket has no FirstResponseAt yet), so it carries the 48h figure.
	friday := time.Date(2026, 8, 28, 18, 0, 0, 0, loc)
	policy := &SLAPolicy{FirstResponseMins: 2880}
	ticket := &Ticket{}
	ticket = ApplyPolicy(ticket, policy, friday)

	wantDue := friday.Add(2880 * time.Minute)
	if !ticket.DueAt.Equal(wantDue) {
		t.Fatalf("DueAt = %v, want %v", ticket.DueAt, wantDue)
	}

	// Saturday, well inside the window and before the 20% at-risk threshold
	// (last 576 min / 9.6h before due) -> still on_track.
	saturdayNoon := time.Date(2026, 8, 29, 12, 0, 0, 0, loc)
	if got := ComputeStatus(ticket, policy, saturdayNoon); got != SLAStatusOnTrack {
		t.Errorf("Saturday noon status = %v, want %v", got, SLAStatusOnTrack)
	}

	// Inside the last 576 minutes before the Sunday 18:00 due time ->
	// at_risk, weekend or not.
	sundayMorning := wantDue.Add(-400 * time.Minute)
	if got := ComputeStatus(ticket, policy, sundayMorning); got != SLAStatusAtRisk {
		t.Errorf("Sunday morning status = %v, want %v", got, SLAStatusAtRisk)
	}

	// Past due -> breached, even though it falls on a weekend with no staff
	// on duty. Confirms the "24/7 assumption" documented on ApplyPolicy:
	// nothing in ComputeStatus special-cases non-business hours.
	sundayEvening := wantDue.Add(time.Hour)
	if got := ComputeStatus(ticket, policy, sundayEvening); got != SLAStatusBreached {
		t.Errorf("Sunday evening status = %v, want %v", got, SLAStatusBreached)
	}
}

// TestApplyPolicy_AcrossDSTSpringForward proves the SLA window survives a DST
// transition. now.Add(duration) operates on the absolute instant, not on
// local wall-clock fields, so a window spanning the spring-forward gap
// (Europe/Berlin: 2026-03-29 02:00 CET jumps to 03:00 CEST) still lands
// exactly windowMins of *elapsed* time later -- the wall-clock gap grows by
// the skipped hour, but the instant arithmetic (and therefore the at-risk
// and breached thresholds derived from it) stays correct.
func TestApplyPolicy_AcrossDSTSpringForward(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	// 01:30 CET, 30 minutes before the 02:00 -> 03:00 jump.
	now := time.Date(2026, 3, 29, 1, 30, 0, 0, loc)
	policy := &SLAPolicy{FirstResponseMins: 90}
	ticket := &Ticket{}
	ticket = ApplyPolicy(ticket, policy, now)

	wantDue := now.Add(90 * time.Minute)
	if !ticket.DueAt.Equal(wantDue) {
		t.Fatalf("DueAt = %v, want %v", ticket.DueAt, wantDue)
	}
	// Wall-clock reads 04:00 CEST: 90 elapsed minutes, but the local clock
	// covers a 150-minute span (01:30 -> 04:00) because the 02:00-03:00 hour
	// never happens that night. Asserting the offset switched to +0200 pins
	// down that the due instant really landed on the CEST side of the jump.
	if got, want := ticket.DueAt.Format("15:04 -0700"), "04:00 +0200"; got != want {
		t.Errorf("DueAt wall clock = %q, want %q", got, want)
	}

	// 18 elapsed minutes before due (90/5 at-risk threshold) -> at_risk.
	beforeDue := wantDue.Add(-10 * time.Minute)
	if got := ComputeStatus(ticket, policy, beforeDue); got != SLAStatusAtRisk {
		t.Errorf("10 min before due status = %v, want %v", got, SLAStatusAtRisk)
	}

	// One elapsed minute past due -> breached.
	afterDue := wantDue.Add(time.Minute)
	if got := ComputeStatus(ticket, policy, afterDue); got != SLAStatusBreached {
		t.Errorf("1 min after due status = %v, want %v", got, SLAStatusBreached)
	}
}

func TestComputeStatus_NilPolicy(t *testing.T) {
	due := time.Now().Add(time.Hour)
	ticket := &Ticket{DueAt: &due}
	if got := ComputeStatus(ticket, nil, time.Now()); got != SLAStatusOnTrack {
		t.Errorf("status = %v, want %v", got, SLAStatusOnTrack)
	}
}

func TestComputeStatus_UsesResolutionWindowAfterFirstResponse(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	firstResponseAt := now
	policy := &SLAPolicy{FirstResponseMins: 30, ResolutionMins: 100}
	due := now.Add(50 * time.Minute)
	ticket := &Ticket{DueAt: &due, FirstResponseAt: &firstResponseAt}

	// At-risk threshold uses ResolutionMins (100/5=20 min), not
	// FirstResponseMins (30/5=6 min), once first response has happened.
	atRiskPoint := due.Add(-15 * time.Minute)
	if got := ComputeStatus(ticket, policy, atRiskPoint); got != SLAStatusAtRisk {
		t.Errorf("status = %v, want %v (resolution window at-risk threshold)", got, SLAStatusAtRisk)
	}

	onTrackPoint := due.Add(-25 * time.Minute)
	if got := ComputeStatus(ticket, policy, onTrackPoint); got != SLAStatusOnTrack {
		t.Errorf("status = %v, want %v", got, SLAStatusOnTrack)
	}
}

func TestApplyPolicy_UsesFirstResponseWindowBeforeFirstResponse(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	policy := &SLAPolicy{FirstResponseMins: 15, ResolutionMins: 240}
	ticket := &Ticket{}
	got := ApplyPolicy(ticket, policy, now)
	want := now.Add(15 * time.Minute)
	if !got.DueAt.Equal(want) {
		t.Errorf("DueAt = %v, want %v", got.DueAt, want)
	}
}
