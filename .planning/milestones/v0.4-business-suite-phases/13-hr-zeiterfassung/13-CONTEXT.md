# Phase 13: HR & Zeiterfassung - Context

**Gathered:** 2026-02-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Employee self-service for leave management, time tracking, and HR document storage — compliant with German labor law (ArbZG, BUrlG). Includes team absence calendar, employee profiles with department/position data, and sick leave with configurable AU requirement. NO payroll/Lohnabrechnung (integration only via Bexio/Abacus/RmA in later phases). Depends on Phase 7 (Calendar) for absence integration and Phase 12 (Biz service) for HR sub-domain.

</domain>

<decisions>
## Implementation Decisions

### Leave workflow
- Manager approves leave requests, with HR fallback if manager is unavailable or on leave themselves
- Half-day increments (morning/afternoon) for leave requests — not hourly, not full-day only
- Overlapping leave: warn manager with overlap info when approving, but allow anyway — manager decides
- Sick leave AU (Arbeitsunfaehigkeitsbescheinigung) threshold is configurable per company by admin (default 3 days per German standard, but companies can set stricter e.g. 1 day)
- System flags sick leave entries exceeding threshold and prompts for AU document upload

### Time tracking UX
- Dual access: quick toggle button in header/sidebar for daily clock in/out, plus dedicated time tracking page for history/corrections/summaries
- ArbZG warnings via toast notifications: info at 8h, warning at 9h, error/block at 10h
- Time entry corrections require manager approval — employee submits correction request, manager approves (audit trail)
- Break tracking is hybrid: manual clock out/in for breaks preferred, but auto-deduct mandatory break time as fallback if no break was clocked (ensures ArbZG compliance either way)

### Absence calendar
- Absence reason visibility is configurable per company by admin (show leave type like "Urlaub"/"Krank" vs. generic "Abwesend")
- Dedicated absence calendar view lives in the HR module (not as overlay in Phase 7 Kalender)
- Company-wide view with department/team filters — not scoped per department
- Gantt-style horizontal bars per person across date range — resource planner visual for seeing coverage gaps

### Employee profiles & docs
- Essential HR fields only: department, position/title, contract type (full/part-time), start date, manager assignment
- No salary, tax ID, social security, or bank details in v1
- Document categories: predefined (Arbeitsvertrag, Zeugnisse, Abmahnungen, Sonstiges) plus admin-configurable custom categories
- Document access configurable per category — some visible to manager (e.g., Zeugnisse), others HR-only (e.g., Abmahnungen)
- Limited self-service: employee can update non-critical fields (emergency contact, address), HR/admin manages role/contract/department

### Claude's Discretion
- Leave carryover rules implementation (BUrlG specifies March 31 deadline)
- Leave types beyond Urlaub/Krank (Sonderurlaub, Elternzeit, etc.) — include what's standard
- Overtime calculation and display approach
- Time tracking page layout and weekly/monthly summary design
- Absence calendar color coding scheme
- Integration approach with existing Phase 6 task timer (separate or unified)
- Employee profile page layout and navigation

</decisions>

<specifics>
## Specific Ideas

- Clock in/out button pattern should feel similar to the existing task timer from Phase 6 — familiar interaction
- Gantt-style absence calendar is the key visual — think resource planner, not traditional calendar
- AU threshold configurability is important because some companies (especially Swiss) have different rules than the German 3-day standard
- HR module is self-contained — no calendar overlay dependency, keeps modules independent

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 13-hr-zeiterfassung*
*Context gathered: 2026-02-19*
