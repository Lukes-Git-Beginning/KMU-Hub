package helpdesk

import "time"

// ApplyPolicy calculates and sets the due_at field on a ticket based on the
// SLA policy. Business-hours logic is not yet implemented — we assume 24/7
// availability (simple time addition). When business_hours JSONB is populated,
// a future implementation can subtract non-business minutes from the window.
//
// Rules:
//   - If ticket.FirstResponseAt is nil, due_at = now + first_response_mins.
//   - Otherwise                        due_at = now + resolution_mins.
func ApplyPolicy(ticket *Ticket, policy *SLAPolicy, now time.Time) *Ticket {
	if policy == nil {
		return ticket
	}

	var windowMins int
	if ticket.FirstResponseAt == nil {
		windowMins = policy.FirstResponseMins
	} else {
		windowMins = policy.ResolutionMins
	}

	due := now.Add(time.Duration(windowMins) * time.Minute)
	ticket.DueAt = &due
	return ticket
}

// ComputeStatus returns the SLA health of a ticket relative to now.
//
// Thresholds:
//   - breached  → now >= due_at
//   - at_risk   → now >= due_at - 20% of the window (warn 20 min before on a
//                 100-min window, etc.)
//   - on_track  → everything else
//
// When no due_at is set the ticket is considered on_track (policy not applied).
func ComputeStatus(ticket *Ticket, policy *SLAPolicy, now time.Time) SLAStatus {
	if ticket.DueAt == nil || policy == nil {
		return SLAStatusOnTrack
	}

	due := *ticket.DueAt

	if !now.Before(due) {
		return SLAStatusBreached
	}

	// Determine the relevant window length for the at-risk threshold.
	windowMins := policy.FirstResponseMins
	if ticket.FirstResponseAt != nil {
		windowMins = policy.ResolutionMins
	}
	atRiskThreshold := due.Add(-time.Duration(windowMins/5) * time.Minute)

	if !now.Before(atRiskThreshold) {
		return SLAStatusAtRisk
	}

	return SLAStatusOnTrack
}
