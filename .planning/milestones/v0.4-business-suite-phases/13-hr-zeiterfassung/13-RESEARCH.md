# Phase 13: HR & Zeiterfassung - Research

**Researched:** 2026-02-19
**Domain:** HR Management, Time Tracking, German Labor Law Compliance (ArbZG, BUrlG)
**Confidence:** HIGH

## Summary

Phase 13 adds employee self-service HR features to KMU Hub: leave management with approval workflows, ArbZG-compliant time tracking with clock-in/out, a Gantt-style team absence calendar, and employee profiles with access-controlled document storage. The backend architecture follows established patterns from the existing codebase -- a new `hr` sub-domain within the `biz` microservice (which already hosts finance), with a new `hr.v1` proto service, gateway route registrar, and PostgreSQL migrations. The frontend has substantial existing mock UI from the design integration (team store, timetracking store, AbsenceCalendar, HRApprovalDialog, ZeiterfassungTab with 7 sub-views) that needs TanStack Query migration and backend wiring.

The primary technical challenge is not the CRUD operations but the ArbZG/BUrlG compliance logic: correct leave balance calculation with pro-rata for part-time and carryover rules, break enforcement with auto-deduction fallback, 11-hour rest validation between shifts, and the configurable AU threshold for sick leave. These rules are well-defined in German law and can be implemented as pure Go functions with comprehensive test coverage.

**Primary recommendation:** Build HR as a sub-domain of the existing `biz` service (not a new microservice), reuse the document service's MinIO integration for HR document storage via entity linking, and implement ArbZG/BUrlG rules as a dedicated `compliance` package with pure functions that can be unit-tested exhaustively.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Leave workflow
- Manager approves leave requests, with HR fallback if manager is unavailable or on leave themselves
- Half-day increments (morning/afternoon) for leave requests -- not hourly, not full-day only
- Overlapping leave: warn manager with overlap info when approving, but allow anyway -- manager decides
- Sick leave AU (Arbeitsunfaehigkeitsbescheinigung) threshold is configurable per company by admin (default 3 days per German standard, but companies can set stricter e.g. 1 day)
- System flags sick leave entries exceeding threshold and prompts for AU document upload

#### Time tracking UX
- Dual access: quick toggle button in header/sidebar for daily clock in/out, plus dedicated time tracking page for history/corrections/summaries
- ArbZG warnings via toast notifications: info at 8h, warning at 9h, error/block at 10h
- Time entry corrections require manager approval -- employee submits correction request, manager approves (audit trail)
- Break tracking is hybrid: manual clock out/in for breaks preferred, but auto-deduct mandatory break time as fallback if no break was clocked (ensures ArbZG compliance either way)

#### Absence calendar
- Absence reason visibility is configurable per company by admin (show leave type like "Urlaub"/"Krank" vs. generic "Abwesend")
- Dedicated absence calendar view lives in the HR module (not as overlay in Phase 7 Kalender)
- Company-wide view with department/team filters -- not scoped per department
- Gantt-style horizontal bars per person across date range -- resource planner visual for seeing coverage gaps

#### Employee profiles & docs
- Essential HR fields only: department, position/title, contract type (full/part-time), start date, manager assignment
- No salary, tax ID, social security, or bank details in v1
- Document categories: predefined (Arbeitsvertrag, Zeugnisse, Abmahnungen, Sonstiges) plus admin-configurable custom categories
- Document access configurable per category -- some visible to manager (e.g., Zeugnisse), others HR-only (e.g., Abmahnungen)
- Limited self-service: employee can update non-critical fields (emergency contact, address), HR/admin manages role/contract/department

### Claude's Discretion
- Leave carryover rules implementation (BUrlG specifies March 31 deadline)
- Leave types beyond Urlaub/Krank (Sonderurlaub, Elternzeit, etc.) -- include what's standard
- Overtime calculation and display approach
- Time tracking page layout and weekly/monthly summary design
- Absence calendar color coding scheme
- Integration approach with existing Phase 6 task timer (separate or unified)
- Employee profile page layout and navigation

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HR-01 | Employee can submit leave/vacation requests for manager approval with workflow | Leave service with request/approve/reject state machine, HR fallback logic, half-day increments, notification events for manager |
| HR-02 | System tracks leave balance with Urlaubsanspruch calculation (BUrlG-konform, pro-rata, carry-over) | BUrlG compliance package: 20 days min (5-day week), pro-rata by work days/week, carryover to March 31, first-6-months pro-rata |
| HR-03 | Team absence calendar shows who is out when (integrated with main calendar) | Gantt-style AbsenceCalendar component exists in design, needs backend endpoint for company-wide absence data with dept filters |
| HR-04 | Employee can clock in/out for time tracking with daily and weekly summaries | New HR time tracking separate from Phase 6 task timer; work_time_entries table with clock-in/out semantics, daily/weekly aggregation queries |
| HR-05 | Time tracking enforces Arbeitszeitgesetz rules (max 8h/10h, 11h rest, break requirements) | ArbZG compliance package: 8h info/9h warn/10h block, 30min break >6h/45min break >9h, 11h rest between shifts, 6-month averaging for 10h exception |
| HR-06 | Employee profiles include department, position, contract type, and access-controlled document storage | Employee profile extension of users table (new hr_employee_profiles table), document storage via existing document service entity linking with category-based ACL |
| HR-07 | Sick leave recording with AU (doctor's note) upload after 3 days | Configurable AU threshold in hr_company_settings, sick leave type with AU document flag, file upload via existing MinIO/document service |
</phase_requirements>

## Standard Stack

### Core (already in project -- no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `time` | Go 1.25 | Date arithmetic for leave/time calculations | No external date library needed; Go's time package handles all BUrlG/ArbZG date math |
| `jackc/pgx/v5` | v5.x | PostgreSQL driver (already used everywhere) | Existing pattern, connection pooling, LISTEN/NOTIFY |
| `google/uuid` | v1.x | UUID generation for all entities | Already used in all models |
| `shopspring/decimal` | v1.x | Precise leave balance calculations (half-days) | Already used in biz/finance, needed for 0.5 day increments |
| `go-chi/chi/v5` | v5.x | HTTP routing in gateway | Existing pattern for all route registrars |
| `google.golang.org/grpc` | v1.x | Service communication | Existing biz service pattern |
| `google.golang.org/protobuf` | v1.x | Proto definitions for HR service | Existing pattern |

### Frontend (already in project -- no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React 19 | 19.x | UI framework | Already in use |
| Zustand | 5.x | Mock stores (existing, to be migrated to TanStack Query) | Design integration stores exist |
| openapi-fetch | latest | Type-safe API client | Existing `api/client.ts` pattern |
| Sonner | latest | Toast notifications (ArbZG warnings) | Already used for toasts throughout app |
| lucide-react | latest | Icons | Already used throughout |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Decimal for half-days | Float64 | Float64 would cause rounding errors with pro-rata; decimal is already a dependency |
| New HR microservice | Sub-domain in biz service | New service adds operational complexity (port, Dockerfile, config); biz is the right home per Phase 12 dependency |
| External HR library | Custom ArbZG/BUrlG logic | No Go library for German labor law compliance; rules are well-defined and testable as pure functions |

**No new dependencies needed.** All required functionality can be built with existing stack.

## Architecture Patterns

### Recommended Backend Structure

```
backend/
  internal/
    biz/
      hr/
        leave/            # Leave request CRUD + approval workflow
          errors.go
          repository.go
          postgres_repository.go
          service.go
          service_test.go
        timetracking/     # Work time clock in/out (separate from Phase 6 task timer)
          errors.go
          repository.go
          postgres_repository.go
          service.go
          service_test.go
        employee/         # Employee profile CRUD + document management
          errors.go
          repository.go
          postgres_repository.go
          service.go
          service_test.go
        compliance/       # ArbZG + BUrlG rule engine (pure functions, no I/O)
          arbzg.go        # Working time rules
          arbzg_test.go
          burlg.go        # Leave entitlement rules
          burlg_test.go
          types.go        # Shared types
        absence/          # Absence calendar query service
          errors.go
          repository.go
          postgres_repository.go
          service.go
  proto/
    hr/
      v1/
        hr.proto          # HR gRPC service definition
        hr.pb.go
        hr_grpc.pb.go
  cmd/
    biz/
      main.go             # Extended to register HR services (same binary)
  migrations/
    000046_create_hr_tables.up.sql
    000046_create_hr_tables.down.sql
```

### Recommended Frontend Structure

```
desktop/src/renderer/src/
  api/
    hr-client.ts           # API client for HR endpoints
    hr-types.ts            # TypeScript types
  modules/
    team/                  # Existing: enhance with backend integration
      TeamPage.tsx         # Existing: wire to API
      AbsenceCalendar.tsx  # Existing: wire to API, enhance to Gantt
      HRApprovalDialog.tsx # Existing: wire to API
    zeiterfassung/         # Existing: enhance with ArbZG warnings
      ZeiterfassungPage.tsx
    profil/
      tabs/
        AbwesenheitenTab.tsx   # Existing: wire to API
        ZeiterfassungTab.tsx   # Existing: wire to API
        DokumenteTab.tsx       # Existing: add HR document categories
  stores/
    team.ts                # Existing mock -> migrate to TanStack Query hooks
    timetracking.ts        # Existing mock -> migrate to TanStack Query hooks
```

### Pattern 1: Sub-domain in Existing Microservice

**What:** HR lives as a sub-domain within the `biz` microservice, not as a separate service.
**When to use:** When the domain is related to business operations and shares infrastructure (DB, config).
**Why:** The biz service already exists with PostgreSQL pool, health checks, metrics, gRPC server. Adding HR sub-domain services follows the same pattern as invoice/quote/dunning.

```go
// In cmd/biz/main.go -- extend existing service
// After finance services initialization:

// HR Services
leaveRepo := leave.NewPostgresRepository(pool)
employeeRepo := employee.NewPostgresRepository(pool)
hrTimeRepo := timetracking.NewPostgresRepository(pool)
absenceRepo := absence.NewPostgresRepository(pool)

leaveSvc := leave.NewService(leaveRepo, employeeRepo, complianceSvc)
hrTimeSvc := timetracking.NewService(hrTimeRepo, complianceSvc)
employeeSvc := employee.NewService(employeeRepo)
absenceSvc := absence.NewService(absenceRepo, employeeRepo)

hrGRPC := server.NewHRGRPCServer(leaveSvc, hrTimeSvc, employeeSvc, absenceSvc)
hrv1.RegisterHRServiceServer(grpcServer, hrGRPC)
```

### Pattern 2: Pure Compliance Functions (No I/O)

**What:** ArbZG and BUrlG rules implemented as pure functions that take inputs and return results, with no database access.
**When to use:** When business rules are well-defined, testable, and should be separated from persistence concerns.
**Why:** Labor law compliance logic is the most critical part of this phase. Separating it into pure functions enables exhaustive unit testing without mocking.

```go
// internal/biz/hr/compliance/burlg.go
package compliance

import "github.com/shopspring/decimal"

// LeaveEntitlementInput contains all data needed to calculate leave balance
type LeaveEntitlementInput struct {
    ContractDaysPerWeek int             // 5 for full-time, 3 for part-time etc.
    AnnualEntitlementDays int           // Contractual entitlement (min 20 for 5-day week)
    StartDate           time.Time       // Employment start date
    CalculationDate     time.Time       // Date to calculate balance for
    UsedDays            decimal.Decimal // Days already taken (supports 0.5 increments)
    CarriedOverDays     decimal.Decimal // Days carried over from previous year
}

// LeaveBalance is the result of leave entitlement calculation
type LeaveBalance struct {
    TotalEntitlement    decimal.Decimal // Full year entitlement
    ProRataEntitlement  decimal.Decimal // Adjusted for start date
    CarriedOver         decimal.Decimal // From previous year
    Used                decimal.Decimal // Already taken
    Remaining           decimal.Decimal // Available to take
    CarryoverDeadline   time.Time       // March 31 of current year
    CarryoverExpired    bool            // Whether carryover has expired
}

// CalculateLeaveBalance computes BUrlG-compliant leave balance.
// BUrlG rules:
// - Minimum 20 days for 5-day week (proportional for part-time)
// - Full entitlement after 6 months employment (pro-rata before)
// - Carryover to March 31 of next year only
// - Pro-rata for mid-year start: 1/12 per full month
func CalculateLeaveBalance(input LeaveEntitlementInput) LeaveBalance {
    // ... pure calculation, fully testable
}
```

```go
// internal/biz/hr/compliance/arbzg.go
package compliance

// WorkTimeCheckResult contains ArbZG compliance check results
type WorkTimeCheckResult struct {
    TotalWorkedMinutes  int
    BreakMinutesTaken   int
    BreakMinutesRequired int
    BreakDeficit        int            // >0 means auto-deduction needed
    Severity            WarningSeverity // Info, Warning, Error
    Message             string
    RestViolation       bool           // True if <11h since last shift end
    RestHoursActual     float64
}

type WarningSeverity string
const (
    SeverityNone    WarningSeverity = "none"
    SeverityInfo    WarningSeverity = "info"     // At 8h
    SeverityWarning WarningSeverity = "warning"  // At 9h
    SeverityError   WarningSeverity = "error"    // At 10h (block)
)

// CheckWorkTime validates a day's work against ArbZG rules.
// ArbZG rules:
// - Max 8h/day standard, 10h/day absolute maximum
// - 30min break required after 6h, 45min after 9h
// - 11h uninterrupted rest between work days
// - Average must not exceed 8h/day over 6 months/24 weeks
func CheckWorkTime(entries []WorkTimeEntry, previousDayEnd *time.Time) WorkTimeCheckResult {
    // ... pure validation, fully testable
}
```

### Pattern 3: Approval Workflow State Machine

**What:** Leave requests follow a state machine: `pending` -> `approved`/`rejected`, with notifications at each transition.
**When to use:** Any request/approval workflow.
**Why:** Clear states prevent invalid transitions, audit trail tracks all changes.

```go
// Valid state transitions for leave requests
var validTransitions = map[string][]string{
    "pending":   {"approved", "rejected", "cancelled"},
    "approved":  {"cancelled"},  // Can cancel approved leave before it starts
    "rejected":  {},             // Terminal state
    "cancelled": {},             // Terminal state
}

// Leave request status change emits notification events
func (s *Service) ApproveLeave(ctx context.Context, requestID, approverID uuid.UUID, comment string) error {
    req, err := s.repo.GetByID(ctx, requestID)
    if err != nil {
        return err
    }
    if req.Status != "pending" {
        return ErrInvalidTransition
    }
    // Check overlaps and warn (but allow per locked decision)
    overlaps, _ := s.repo.FindOverlaps(ctx, req.EmployeeID, req.StartDate, req.EndDate)

    req.Status = "approved"
    req.ApprovedBy = &approverID
    req.ApprovalComment = comment
    req.ApprovedAt = timePtr(time.Now())

    if err := s.repo.Update(ctx, req); err != nil {
        return err
    }

    // Emit notification to employee
    event.EmitEvent(ctx, s.pool, models.EventPayload{
        Type:          "hr.leave.approved",
        ModuleID:      "hr",
        ActorID:       approverID.String(),
        TargetUserIDs: []string{req.EmployeeID.String()},
        Title:         "Urlaubsantrag genehmigt",
        Body:          fmt.Sprintf("Dein Antrag vom %s bis %s wurde genehmigt.", ...),
    })
    return nil
}
```

### Pattern 4: HR Time Tracking Separate from Task Timer

**What:** Phase 13 work time tracking (clock in/out for the workday) is a separate system from Phase 6 task time entries (tracking time per task).
**When to use:** When the same concept (time tracking) serves fundamentally different purposes.
**Why:** Task timer tracks time against tasks/projects for billing/productivity. HR time tracking tracks total work hours for ArbZG compliance. They have different data models, different rules, and different UI. Merging them would create unnecessary coupling.

**Integration point:** The frontend clock-in/out button in the header should feel similar to the Phase 6 task timer button (familiar interaction pattern), but they are independent systems. A future enhancement could correlate task time with work time, but that is out of scope.

```
Phase 6 Task Timer:          Phase 13 HR Work Time:
- Per task/project            - Per employee/day
- Duration tracking           - Clock in/out with breaks
- Billing/productivity        - ArbZG compliance
- work.time_entries table     - hr_work_time_entries table
- Work service gRPC           - HR/Biz service gRPC
```

### Anti-Patterns to Avoid

- **Mixing task time and work time:** Do NOT use the existing `work.time_entries` table for HR clock-in/out. Different domain, different rules, different table.
- **Implementing payroll fields:** The user explicitly excluded salary, tax ID, social security, bank details. Do NOT add these fields.
- **Calendar overlay in Phase 7:** The absence calendar is self-contained in the HR module. Do NOT add absence overlay to the main calendar (keeps modules independent).
- **Complex approval chains:** Keep it simple: direct manager approves, HR fallback. No multi-level approval chains.
- **Hardcoding German rules:** AU threshold and break rules should be configurable via company settings, not hardcoded (supports future CH/AT variations).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File storage for AU documents | Custom file upload system | Existing MinIO via document service entity linking | Phase 11 already built the entire file storage stack with versioning, tags, and entity links |
| Notification delivery | Custom WebSocket push | Existing `event.EmitEvent` + pg_notify + WebSocket hub | Phase 4 notification system with real-time push already works |
| UUID generation | Custom ID schemes | `google/uuid` | Already standard across all models |
| Decimal arithmetic | Float64 for leave balances | `shopspring/decimal` | Already a dependency, prevents rounding errors with half-day increments |
| Date parsing/formatting | Custom date utilities | Go `time` package | Built-in, timezone-aware, handles all needed operations |
| Public holiday awareness | Holiday calendar from scratch | Existing `work/holiday` service with Nager.Date API | Phase 7 already built DACH holiday management with subdivision support |

**Key insight:** Phase 13 can reuse significant infrastructure from earlier phases. The document service (Phase 11), notification system (Phase 4), holiday service (Phase 7), and the biz service pattern (Phase 12) provide all the building blocks. The new code is primarily domain-specific business logic (ArbZG/BUrlG rules) and CRUD operations.

## Common Pitfalls

### Pitfall 1: Leave Balance Off-by-One with Half-Days
**What goes wrong:** Calculating leave balance with half-day increments using integer arithmetic or float64 causes rounding errors. An employee with 20.5 days used and 25 days entitlement ends up with 4 or 5 remaining instead of 4.5.
**Why it happens:** Float64 cannot represent 0.1 exactly; cascading arithmetic amplifies errors.
**How to avoid:** Use `shopspring/decimal` for all leave balance calculations. Store leave as `NUMERIC(5,1)` in PostgreSQL. Test with half-day edge cases.
**Warning signs:** Leave balance not matching manual calculation, especially with many half-day requests.

### Pitfall 2: Timezone Issues in Work Time Calculations
**What goes wrong:** An employee clocks in at 23:30 and clocks out at 08:00 the next day. If dates are compared in UTC, the day assignment is wrong. Breaks that cross midnight also cause issues.
**Why it happens:** PostgreSQL stores TIMESTAMPTZ in UTC; frontend sends local time; ArbZG rules apply to the *working day*, not the calendar day.
**How to avoid:** Store all times as TIMESTAMPTZ. Always convert to the company's local timezone for ArbZG calculations. Define "working day" as the date when the shift *started*. Include timezone in the company settings.
**Warning signs:** Night shift workers getting incorrect daily totals; break calculations wrong around midnight.

### Pitfall 3: Carryover Deadline Calculation
**What goes wrong:** The March 31 carryover deadline is implemented as a simple date comparison, but doesn't account for: (a) employees who were sick during the carryover period (entitled to keep leave), (b) the employer's obligation to notify employees about expiring leave (EU Court of Justice ruling), (c) pro-rata carryover amounts.
**Why it happens:** BUrlG Section 7(3) seems simple ("March 31") but case law has added complexity.
**How to avoid:** Implement carryover as a scheduled job that runs on April 1, but only expires leave if the employee was notified AND was not on extended sick leave. Store a `carryover_notified_at` timestamp. For v1, the conservative approach: auto-expire on April 1 but allow HR to manually reinstate.
**Warning signs:** Employees losing leave they should have kept due to illness.

### Pitfall 4: Break Auto-Deduction Double-Counting
**What goes wrong:** An employee clocks a 20-minute break, but ArbZG requires 30 minutes for >6h work. The system auto-deducts 30 minutes on top of the manual 20 minutes, resulting in 50 minutes of break.
**Why it happens:** The auto-deduction fallback doesn't consider partial manual breaks.
**How to avoid:** Auto-deduction = max(0, required_break - manually_clocked_break). If an employee clocks 20 minutes but needs 30, auto-deduct only 10 minutes. If they clock 30+, no auto-deduction.
**Warning signs:** Employees' net work hours being lower than expected on days with partial break tracking.

### Pitfall 5: Overlapping Leave Requests Data Integrity
**What goes wrong:** Two managers approve overlapping leave for the same employee from different devices simultaneously. Or an employee submits two overlapping requests before either is processed.
**Why it happens:** No database-level constraint on leave date ranges; relies only on application-level checks.
**How to avoid:** Use a PostgreSQL exclusion constraint with `tsrange` (date range overlap) on the leave_requests table for approved leaves. Allow pending overlaps (manager decides), but prevent double-approved overlaps at the DB level.
**Warning signs:** Same employee showing as on leave twice in the absence calendar for the same dates.

### Pitfall 6: Existing Frontend Mock Data Conflicts
**What goes wrong:** The design integration added Zustand stores with hardcoded mock data (team.ts, timetracking.ts). When the backend is wired, the mock data conflicts with real data or the store shape doesn't match API responses.
**Why it happens:** Mock stores were built for demo purposes without knowing the final API shape.
**How to avoid:** Create new TanStack Query hooks (`useLeaveRequests`, `useWorkTime`, etc.) alongside existing stores. Migrate pages incrementally. Do NOT modify existing Zustand stores -- replace them.
**Warning signs:** Stale mock data appearing alongside real data; type mismatches between store types and API types.

## Code Examples

### Database Schema for HR Tables (Migration 000046)

```sql
-- HR company settings (per-tenant configuration)
CREATE TABLE hr_company_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    au_threshold_days INT NOT NULL DEFAULT 3,
    show_absence_reason BOOLEAN NOT NULL DEFAULT TRUE,
    default_annual_leave_days INT NOT NULL DEFAULT 20,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_company_settings_tenant UNIQUE (tenant_id)
);

-- Employee profiles (extends users table with HR data)
CREATE TABLE hr_employee_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    department VARCHAR(100),
    position_title VARCHAR(200),
    contract_type VARCHAR(20) NOT NULL DEFAULT 'full_time',
    work_days_per_week INT NOT NULL DEFAULT 5,
    annual_leave_days INT NOT NULL DEFAULT 20,
    manager_user_id UUID REFERENCES users(id),
    start_date DATE NOT NULL,
    emergency_contact_name VARCHAR(200),
    emergency_contact_phone VARCHAR(50),
    address_street VARCHAR(255),
    address_city VARCHAR(100),
    address_postal_code VARCHAR(10),
    address_country VARCHAR(2) NOT NULL DEFAULT 'DE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_employee_user UNIQUE (user_id)
);

CREATE INDEX idx_hr_employee_profiles_manager ON hr_employee_profiles(manager_user_id);
CREATE INDEX idx_hr_employee_profiles_department ON hr_employee_profiles(department);

-- Leave types (predefined + admin-configurable)
CREATE TABLE hr_leave_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    key VARCHAR(50) NOT NULL,
    color VARCHAR(20) NOT NULL DEFAULT '#3d8abf',
    deducts_from_balance BOOLEAN NOT NULL DEFAULT TRUE,
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    requires_au_document BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_leave_type_key UNIQUE (tenant_id, key)
);

-- Leave requests
CREATE TABLE hr_leave_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    leave_type_id UUID NOT NULL REFERENCES hr_leave_types(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_half_day_start BOOLEAN NOT NULL DEFAULT FALSE,
    half_day_period_start VARCHAR(10),  -- 'morning' or 'afternoon'
    is_half_day_end BOOLEAN NOT NULL DEFAULT FALSE,
    half_day_period_end VARCHAR(10),
    total_days NUMERIC(5,1) NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_by UUID REFERENCES users(id),
    approval_comment TEXT,
    approved_at TIMESTAMPTZ,
    au_document_required BOOLEAN NOT NULL DEFAULT FALSE,
    au_document_file_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_leave_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_leave_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'))
);

CREATE INDEX idx_hr_leave_requests_employee ON hr_leave_requests(employee_id);
CREATE INDEX idx_hr_leave_requests_dates ON hr_leave_requests(start_date, end_date);
CREATE INDEX idx_hr_leave_requests_status ON hr_leave_requests(status);

-- Leave balance tracking (per employee per year)
CREATE TABLE hr_leave_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    year INT NOT NULL,
    entitlement NUMERIC(5,1) NOT NULL,
    carried_over NUMERIC(5,1) NOT NULL DEFAULT 0,
    used NUMERIC(5,1) NOT NULL DEFAULT 0,
    remaining NUMERIC(5,1) NOT NULL,
    carryover_expires_at DATE,
    carryover_notified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_leave_balance UNIQUE (tenant_id, employee_id, year)
);

-- Work time entries (clock in/out for ArbZG compliance)
CREATE TABLE hr_work_time_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    clock_in TIMESTAMPTZ NOT NULL,
    clock_out TIMESTAMPTZ,
    break_minutes INT NOT NULL DEFAULT 0,
    auto_break_deducted INT NOT NULL DEFAULT 0,
    net_work_minutes INT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_correction BOOLEAN NOT NULL DEFAULT FALSE,
    original_entry_id UUID REFERENCES hr_work_time_entries(id),
    correction_reason TEXT,
    correction_approved_by UUID REFERENCES users(id),
    correction_approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_work_time_status CHECK (status IN ('active', 'completed', 'correction_pending', 'correction_approved'))
);

CREATE INDEX idx_hr_work_time_employee ON hr_work_time_entries(employee_id);
CREATE INDEX idx_hr_work_time_clock_in ON hr_work_time_entries(clock_in);

-- HR document categories (predefined + admin-configurable)
CREATE TABLE hr_document_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    key VARCHAR(50) NOT NULL,
    visibility VARCHAR(20) NOT NULL DEFAULT 'hr_only',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_doc_category_key UNIQUE (tenant_id, key),
    CONSTRAINT chk_doc_visibility CHECK (visibility IN ('hr_only', 'manager', 'employee'))
);

-- HR document links (links document service files to HR context)
CREATE TABLE hr_employee_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES hr_document_categories(id),
    file_id UUID NOT NULL,  -- References document_files.id from Phase 11
    uploaded_by UUID NOT NULL REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_doc_file UNIQUE (employee_id, file_id)
);

CREATE INDEX idx_hr_employee_documents_employee ON hr_employee_documents(employee_id);
CREATE INDEX idx_hr_employee_documents_category ON hr_employee_documents(category_id);
```

### ArbZG Break Calculation Example

```go
// Source: ArbZG Section 4 - verified via official law text
// Break requirements:
// - >6h work: minimum 30 minutes break
// - >9h work: minimum 45 minutes break
// - Breaks can be split into minimum 15-minute segments

func CalculateRequiredBreak(workedMinutes int) int {
    switch {
    case workedMinutes > 540: // >9h
        return 45
    case workedMinutes > 360: // >6h
        return 30
    default:
        return 0
    }
}

func CalculateAutoBreakDeduction(workedMinutes, manualBreakMinutes int) int {
    required := CalculateRequiredBreak(workedMinutes)
    deficit := required - manualBreakMinutes
    if deficit > 0 {
        return deficit
    }
    return 0
}
```

### BUrlG Pro-Rata Leave Calculation

```go
// Source: BUrlG Section 3, 4, 5
// - Minimum 20 days for 5-day week
// - Part-time: proportional to work days per week
// - First 6 months: 1/12 per full calendar month
// - Mid-year start: pro-rata for remaining months

func CalculateProRataEntitlement(
    annualDays int,
    workDaysPerWeek int,
    startDate time.Time,
    year int,
) decimal.Decimal {
    // Adjust for part-time
    statutory := decimal.NewFromInt(int64(annualDays))
    if workDaysPerWeek < 5 {
        statutory = statutory.Mul(decimal.NewFromInt(int64(workDaysPerWeek))).
            Div(decimal.NewFromInt(5))
    }

    // Full year if employed before January 1
    yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
    if startDate.Before(yearStart) {
        return statutory
    }

    // Pro-rata: months remaining / 12
    startMonth := int(startDate.Month())
    fullMonths := 12 - startMonth // Months after start month
    if startDate.Day() == 1 {
        fullMonths++ // Include start month if started on 1st
    }

    return statutory.Mul(decimal.NewFromInt(int64(fullMonths))).
        Div(decimal.NewFromInt(12)).
        Round(1) // Round to 1 decimal for half-days
}
```

### Gateway Route Registration Pattern

```go
// internal/gateway/route_hr.go -- follows exact same pattern as route_biz.go
package gateway

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/kmuhub/kmuhub/internal/middleware"
    hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type HRRoutes struct {
    registry *ServiceRegistry
}

func NewHRRoutes(registry *ServiceRegistry) *HRRoutes {
    return &HRRoutes{registry: registry}
}

func (h *HRRoutes) ServiceName() string { return "biz" } // Same service, different routes

func (h *HRRoutes) getHRClient() (hrv1.HRServiceClient, error) {
    conn, err := h.registry.GetConnection("biz") // Reuses biz connection
    if err != nil {
        return nil, err
    }
    return hrv1.NewHRServiceClient(conn), nil
}

func (h *HRRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
    // Leave management
    r.Route("/api/v1/hr/leave", func(r chi.Router) {
        r.Use(authMiddleware)
        r.Get("/requests", h.HandleListLeaveRequests)
        r.Post("/requests", h.HandleCreateLeaveRequest)
        r.Get("/requests/{id}", h.HandleGetLeaveRequest)
        r.Post("/requests/{id}/approve", h.HandleApproveLeaveRequest)
        r.Post("/requests/{id}/reject", h.HandleRejectLeaveRequest)
        r.Post("/requests/{id}/cancel", h.HandleCancelLeaveRequest)
        r.Get("/balance", h.HandleGetLeaveBalance)
        r.Get("/balance/{userId}", h.HandleGetEmployeeLeaveBalance)
        r.Get("/types", h.HandleListLeaveTypes)
    })

    // Time tracking
    r.Route("/api/v1/hr/time", func(r chi.Router) {
        r.Use(authMiddleware)
        r.Post("/clock-in", h.HandleClockIn)
        r.Post("/clock-out", h.HandleClockOut)
        r.Post("/break/start", h.HandleBreakStart)
        r.Post("/break/end", h.HandleBreakEnd)
        r.Get("/active", h.HandleGetActiveShift)
        r.Get("/entries", h.HandleListWorkTimeEntries)
        r.Get("/summary/daily", h.HandleDailySummary)
        r.Get("/summary/weekly", h.HandleWeeklySummary)
        r.Post("/corrections", h.HandleSubmitCorrection)
        r.Post("/corrections/{id}/approve", h.HandleApproveCorrection)
    })

    // Absence calendar
    r.Route("/api/v1/hr/absences", func(r chi.Router) {
        r.Use(authMiddleware)
        r.Get("/calendar", h.HandleGetAbsenceCalendar)
    })

    // Employee profiles
    r.Route("/api/v1/hr/employees", func(r chi.Router) {
        r.Use(authMiddleware)
        r.Get("/", h.HandleListEmployees)
        r.Get("/{id}", h.HandleGetEmployee)
        r.Put("/{id}", h.HandleUpdateEmployee)
        r.Get("/{id}/documents", h.HandleListEmployeeDocuments)
        r.Post("/{id}/documents", h.HandleUploadEmployeeDocument)
    })

    // HR settings (admin only)
    r.Route("/api/v1/hr/settings", func(r chi.Router) {
        r.Use(authMiddleware)
        r.With(middleware.RequireRole("admin")).Get("/", h.HandleGetHRSettings)
        r.With(middleware.RequireRole("admin")).Put("/", h.HandleUpdateHRSettings)
    })
}
```

### Notification Events for HR Module

```go
// Add to internal/notification/event/types.go
const (
    // HR events
    EventHRLeaveRequested        = "hr.leave.requested"
    EventHRLeaveApproved         = "hr.leave.approved"
    EventHRLeaveRejected         = "hr.leave.rejected"
    EventHRLeaveCancelled        = "hr.leave.cancelled"
    EventHRCorrectionRequested   = "hr.leave.correction_requested"
    EventHRCorrectionApproved    = "hr.leave.correction_approved"
    EventHRAUDocumentRequired    = "hr.sick.au_required"

    ModuleHR = "hr"
)
```

## Discretion Recommendations

### Leave Types (Claude's Discretion)

**Recommendation:** Include the following standard German leave types as system defaults:

| Key | Name (de) | Deducts Balance | Requires Approval | Notes |
|-----|-----------|-----------------|-------------------|-------|
| `urlaub` | Urlaub | Yes | Yes | Standard vacation |
| `krank` | Krankheit | No | No (notification only) | Sick leave, AU threshold applies |
| `sonderurlaub_hochzeit` | Sonderurlaub (Hochzeit) | No | Yes | 1 day per BGB 616 |
| `sonderurlaub_geburt` | Sonderurlaub (Geburt) | No | Yes | 1 day per BGB 616 |
| `sonderurlaub_todesfall` | Sonderurlaub (Todesfall) | No | Yes | 1-3 days per BGB 616 |
| `sonderurlaub_umzug` | Sonderurlaub (Umzug) | No | Yes | 1 day (common, not statutory) |
| `elternzeit` | Elternzeit | No | Yes | Parental leave, long-term |
| `unbezahlter_urlaub` | Unbezahlter Urlaub | No | Yes | Unpaid leave |
| `homeoffice` | Homeoffice | No | Yes | Not really leave, but tracked for planning |
| `weiterbildung` | Weiterbildung | No | Yes | Education/training days |

**Rationale:** These cover the BGB 616 statutory types plus the most common company-offered types. Admin can add custom types on top.

### Leave Carryover Implementation (Claude's Discretion)

**Recommendation:** Implement carryover with a conservative approach:
1. At year-end, automatically carry over unused leave balance (max = unused days)
2. Set `carryover_expires_at` to March 31 of the new year
3. On April 1, a scheduled check sets expired carryover to 0
4. Exception: if employee was on sick leave for the entire carryover period (Jan-Mar), do NOT expire
5. HR/admin can manually reinstate expired carryover (audit logged)
6. Display a warning banner in the leave balance UI starting February showing "X days expire on 31.03."

### Overtime Tracking (Claude's Discretion)

**Recommendation:** Track overtime implicitly, not as a separate request type:
- Overtime = total worked hours - contracted daily hours (from employee profile)
- Display daily/weekly/monthly overtime in the time tracking summaries
- Do NOT create a formal "overtime balance" or "overtime compensation" system (that enters payroll territory)
- The UI shows: "Heute: 9h 15m gearbeitet (1h 15m Ueberstunden)" as informational
- ArbZG 6-month averaging: track but only warn, not enforce (employer responsibility)

### Absence Calendar Color Coding (Claude's Discretion)

**Recommendation:** Use the existing color scheme from the design integration's `AbsenceCalendar.tsx`, which already has:
- Urlaub: Blue (#3d8abf)
- Krank: Red (#bf3d3d)
- Ueberstunden: Orange (#bf8a3d)
- Arzt: Teal (#1e7e74)
- Homeoffice: Green (#4a7c6a)
- Weiterbildung: Purple (#7c5a8a)

When `show_absence_reason` is disabled (company setting), show all absences in a neutral gray.

### Phase 6 Task Timer Integration (Claude's Discretion)

**Recommendation:** Keep them completely separate:
- Phase 6 `useTimeTrackingStore` stays for task-level time tracking (per project/task)
- Phase 13 gets a new `useWorkTimeStore` / TanStack Query hooks for HR clock-in/out
- The header/sidebar clock-in/out button is independent of the task timer button
- Both can run simultaneously (employee can be clocked in for work AND tracking a task)
- No data sharing between the two systems in v1

**Rationale:** Unifying them would be complex with minimal benefit. Task time is for productivity/billing; work time is for labor law compliance. Different audiences (manager vs HR), different rules.

### Employee Profile Page Layout (Claude's Discretion)

**Recommendation:** Use the existing `ProfilPage.tsx` tab pattern:
- Tab 1: Profil (existing) -- personal info, emergency contact (self-editable)
- Tab 2: Abwesenheiten (existing) -- leave requests, balance card
- Tab 3: Zeiterfassung (existing) -- clock-in/out history, daily/weekly summaries
- Tab 4: Dokumente (existing) -- HR documents by category

For the admin/HR view of another employee, use the existing `MemberDetailPanel.tsx` pattern (slide-in panel from the team list).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Paper AU (Arbeitsunfaehigkeitsbescheinigung) | Digital eAU (elektronische AU) since 2023 | 2023 | AU is sent directly from doctor to health insurer to employer digitally; the system only needs to track "AU received: yes/no" and optionally store a document |
| Daily max working hours (ArbZG) | Proposed weekly max (coalition agreement 2025) | Under discussion | The 8h/10h daily limit may shift to a 48h weekly limit; build the system to support both models via configuration |
| Manual time recording | Mandatory electronic time recording (BAG ruling 2022) | Sept 2022 | German employers MUST record working hours electronically; this phase implements exactly that |

**Deprecated/outdated:**
- Paper-based AU process: replaced by eAU since Jan 2023, but companies still need document upload capability for edge cases
- CHF formatting in mock data: design integration used CH locale; this phase targets DE-first per strategy decision

## Open Questions

1. **Multi-tenant scoping**
   - What we know: The biz service uses `tenant_id` on all finance tables. Current single-tenant mode uses user_id as tenant_id (`getTenantID` in route_biz.go).
   - What's unclear: Should HR tables follow the same tenant_id pattern, or is it premature?
   - Recommendation: Follow the same pattern. Add `tenant_id` to all HR tables. Use the same `getTenantID` helper. This is a 2-minute decision now vs. a painful migration later.

2. **"hr" role requirement**
   - What we know: Backend has 3 roles (admin, manager, member). Memory notes say frontend needs 5 roles (admin, manager, member, hr, it_support).
   - What's unclear: Is the "hr" role needed now, or can we use "admin" + specific permissions for HR operations?
   - Recommendation: Add "hr" role in this phase. Leave approval and employee profile management need role-based access that is distinct from "admin" (HR should not have system admin rights). Add it to the auth service's role seeding.

3. **Break tracking granularity**
   - What we know: Decision says "manual clock out/in for breaks preferred, auto-deduct as fallback."
   - What's unclear: Should breaks be tracked as separate entries (clock out for break, clock in after break) or as a single break_minutes field on the work time entry?
   - Recommendation: Separate `hr_break_entries` table linked to work time entry. This allows multiple breaks per day and exact tracking. The `break_minutes` field on the main entry is a denormalized sum for quick queries. Auto-deduction updates the denormalized field without creating break entries.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: All architecture patterns, code examples, and integration points verified by reading actual source files in the repository
- ArbZG (Arbeitszeitgesetz): Official German law text - Sections 3 (max hours), 4 (breaks), 5 (rest period)
- BUrlG (Bundesurlaubsgesetz): Official German law text - Sections 3 (entitlement), 4 (waiting period), 5 (partial year), 7 (carryover)

### Secondary (MEDIUM confidence)
- [Payroll Germany - ArbZG overview](https://payrollgermany.de/blog/understanding-the-working-hours-act-arbeitszeitgesetz-in-germany/) - Verified against official law text
- [ZMI - Working Hours Act](https://zmi.de/en/lexikon/working-hours-act-arbzg/) - Break and rest rules
- [HRlab - Legal Vacation Entitlement](https://www.hrlab.de/en/hr-lexicon/legal-vacation-entitlement) - BUrlG calculation rules
- [Handbook Germany - Sick Leave](https://handbookgermany.de/en/sick-leave) - AU threshold rules
- [Haas Eschborn - Sonderurlaub](https://haas-eschborn.de/en/sonderurlaub-diese-9-faelle-sollten-sie-kennen/) - BGB 616 special leave types
- [Deel - Germany Mandatory Time Tracking](https://www.deel.com/blog/germany-mandatory-time-tracking/) - BAG ruling on electronic time recording

### Tertiary (LOW confidence)
- Coalition agreement proposal for weekly instead of daily max hours: Under political discussion, not yet law. Build configurable, not assumption-based.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use; no new dependencies needed
- Architecture: HIGH - Follows exact patterns established in Phases 4, 6, 7, 11, 12 (service structure, gateway routes, proto definitions, migrations)
- Compliance rules (ArbZG/BUrlG): HIGH - Well-defined German law with clear numerical rules, verified via multiple official sources
- Pitfalls: MEDIUM - Based on domain knowledge of time tracking systems; timezone and half-day edge cases are real but standard patterns exist

**Research date:** 2026-02-19
**Valid until:** 2026-03-19 (stable domain; labor law doesn't change frequently)
