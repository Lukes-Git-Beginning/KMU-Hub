---
phase: 13-hr-zeiterfassung
verified: 2026-02-19T23:45:00Z
status: passed
score: 9/9 must-haves verified
re_verification: false
human_verification:
  - test: "Leave request submission with half-day morning/afternoon selection"
    expected: "Employee selects date range, half-day checkbox, morning/afternoon period, submits and sees pending status"
    why_human: "UI interaction flow and form UX cannot be verified programmatically"
  - test: "ArbZG toast warnings appear on clock-out at 8h/9h/10h thresholds"
    expected: "info toast at 8h, warning toast at 9h, error toast at 10h+ (blocked before clock-out if already at 10h)"
    why_human: "Toast notification display requires runtime execution and UI interaction"
  - test: "Leave balance carryover expiry warning banner"
    expected: "Pre-expiry banner appears after Feb 1 when carryover days > 0; expired banner appears after March 31"
    why_human: "Date-conditional UI rendering requires runtime execution"
  - test: "Absence calendar Gantt bars display with correct colors per leave type"
    expected: "Blue bars for Urlaub, red for Krank, neutral gray for all types when showAbsenceReason is disabled"
    why_human: "Visual rendering and color display require UI inspection"
  - test: "Sick leave AU document upload prompt after threshold exceeded"
    expected: "When sick leave exceeds company-configured threshold, dialog prompts for AU document upload"
    why_human: "Multi-step interaction with configurable threshold requires runtime testing"
---

# Phase 13: HR & Zeiterfassung Verification Report

**Phase Goal:** Employees can manage leave, track time, and access HR documents within the Hub, fully compliant with German/Swiss labor law. NO payroll -- salary/Lohn is handled by external integrations (Bexio, Abacus, RmA).
**Verified:** 2026-02-19T23:45:00Z
**Status:** PASSED
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths (from Phase Success Criteria)

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | Employee can submit leave request and manager receives it for approval, with full request/approve/reject workflow | VERIFIED | `leave/service.go` implements `CreateLeaveRequest` (pending), `ApproveLeaveRequest`, `RejectLeaveRequest`, `CancelLeaveRequest`; `AbwesenheitenTab.tsx` uses `useCreateLeaveRequest`, `useApproveLeaveRequest`, `useRejectLeaveRequest` hooks; `HRApprovalDialog.tsx` wired to mutations |
| 2 | System correctly calculates leave balance per BUrlG (20-30 days, part-time pro-rata, carryover to March 31) | VERIFIED | `burlg.go` implements `CalculateLeaveBalance` with part-time pro-rata (`annualDays * contractDaysPerWeek / 5`), mid-year start pro-rata (months/12), and March 31 carryover expiry; 12 test cases in `burlg_test.go` (227 lines) all pass |
| 3 | Team absence calendar shows who is out when, integrated with the main calendar module | VERIFIED | `AbsenceCalendar.tsx` uses `useAbsenceCalendar` hook backed by real API; embedded in `TeamPage.tsx`; context locked decision: "HR module is self-contained -- no calendar overlay dependency, keeps modules independent" -- standalone Gantt view is the intentional design |
| 4 | Employee can clock in and out for time tracking, with daily and weekly hour summaries | VERIFIED | `timetracking/service.go` implements `ClockIn`, `ClockOut`, `GetDailySummary`, `GetWeeklySummary`; `ZeiterfassungTab.tsx` uses `useDailySummary` and `useWeeklySummary` hooks; `ClockInButton.tsx` in header uses `useClockIn`/`useClockOut` |
| 5 | Time tracking enforces ArbZG rules: warns at 8h, blocks at 10h daily, enforces 11h rest, requires breaks | VERIFIED | `arbzg.go` implements `CheckWorkTime` (SeverityInfo at >=8h, SeverityWarning at >=9h, SeverityError at >10h), `CheckRestPeriod` (11h minimum), `CalculateRequiredBreak` (30/45 min), `CalculateAutoBreakDeduction`; `ClockIn` returns `ErrMaxDailyHoursExceeded` when DailySummary >= 600 net minutes; 15 ArbZG tests pass; `hr-hooks.ts` `showArbZGToast()` triggers sonner toasts on clock-out |
| 6 | Employee profiles include department, position, contract type, with access-controlled document storage | VERIFIED | `hr_employee_profiles` table has department, position_title, contract_type, work_days_per_week; `employee/service.go` enforces role-based field restrictions (employee role: emergency contact + address only; admin/hr/manager: all fields); `hr_document_categories` has hr_only/manager/employee visibility; `DokumenteTab.tsx` uses `useEmployeeDocuments` + `useDocumentCategories` |
| 7 | Sick leave can be recorded with AU upload required after 3 consecutive days | VERIFIED | `leave/service.go` `RecordSickLeave` checks `AUThresholdDays` (configurable, default 3) and sets `au_document_required`; `AbwesenheitenTab.tsx` imports `useRecordSickLeave`; sick leave auto-approves (`requires_approval=false`) with `au_document_required` flag when threshold exceeded |

**Score:** 7/7 success criteria verified

### Consolidated Must-Have Truths (from PLAN frontmatter, all 4 plans)

| Truth | Status | Evidence |
| ----- | ------ | -------- |
| BUrlG leave balance correctly computes pro-rata for part-time and mid-year starts | VERIFIED | `burlg.go` line 28-46 implements both; 4 dedicated test cases pass |
| BUrlG carryover expires on March 31 and respects half-day increments | VERIFIED | `carryoverDeadline = time.Date(calcYear, 3, 31, ...)`, `carryoverExpired` flag; `Round(1)` for half-day precision |
| ArbZG work time check warns at 8h, warns harder at 9h, blocks at 10h | VERIFIED | `arbzg.go` `CheckWorkTime`: SeverityInfo at >=480, SeverityWarning at >=540, SeverityError at >600; `ClockIn` returns `ErrMaxDailyHoursExceeded` at >=600 net minutes |
| ArbZG break calculation auto-deducts only the deficit | VERIFIED | `CalculateAutoBreakDeduction`: `max(0, required - manual)` |
| ArbZG 11-hour rest period is validated | VERIFIED | `CheckRestPeriod` returns violation when `hours < 11`; used in `ClockIn` |
| HR proto service defines all RPCs for leave, time tracking, absences, employees, settings | VERIFIED | `hr.proto` has 30 RPCs (grep confirms `rpc` count = 30 excluding `service HRService` line) |
| Migration 000046 creates all HR tables with proper constraints and indexes | VERIFIED | 9 tables confirmed: hr_company_settings, hr_employee_profiles, hr_leave_types, hr_leave_requests, hr_leave_balances, hr_work_time_entries, hr_break_entries, hr_document_categories, hr_employee_documents |
| Employee can submit leave request with half-day support, status starts as pending | VERIFIED | `CreateLeaveRequest` sets status "pending" (or "approved" for sick leave); half-day fields in proto and service |
| Manager can approve/reject with correct status transitions | VERIFIED | State machine: pending -> approved/rejected/cancelled; `verifyApprover` checks manager or HR role |
| Sick leave exceeding AU threshold flags request for AU document upload | VERIFIED | `AUThresholdDays` check in `RecordSickLeave`, sets `AuDocumentRequired = true` |
| Absence calendar returns all company absences with department filtering | VERIFIED | `absence/postgres_repository.go` JOINs with optional department filter |
| Employee profile CRUD with self-service field restrictions | VERIFIED | `UpdateEmployee` with `callerRole` check; `hasRestrictedFields()` guards HR-only fields |
| Employee clock-in/out + ArbZG severity in response | VERIFIED | `ClockIn`/`ClockOut` return `*compliance.WorkTimeCheckResult` alongside entry |
| HRGRPCServer exposes all ~30 RPCs | VERIFIED | `hr_grpc.go` struct delegates to all 4 services; biz binary registers via `hrv1.RegisterHRServiceServer` |
| Gateway exposes HR routes under /api/v1/hr/* | VERIFIED | `route_hr.go` registers 30+ routes; `gateway/main.go` includes `gateway.NewHRRoutes(registry)` |
| TanStack Query hooks use hr-client.ts | VERIFIED | `hr-hooks.ts` imports `hrLeaveApi, hrTimeApi, hrAbsenceApi, hrEmployeeApi, hrSettingsApi` from `../hr-client` |
| ClockInButton uses real clock-in/out mutations | VERIFIED | `ClockInButton.tsx` uses `useClockIn`, `useClockOut`, `useStartBreak`, `useEndBreak`, `useWorkTimeStatus` |

**Score:** 9/9 must-have groups verified

### Required Artifacts

| Artifact | Status | Evidence |
| -------- | ------ | -------- |
| `backend/proto/hr/v1/hr.proto` | VERIFIED | Exists; `service HRService` present; 30 RPCs confirmed by grep |
| `backend/internal/biz/hr/compliance/burlg.go` | VERIFIED | Exists; `CalculateLeaveBalance` implemented; 81 lines of real logic |
| `backend/internal/biz/hr/compliance/arbzg.go` | VERIFIED | Exists; `CheckWorkTime` implemented; 107 lines of real logic |
| `backend/internal/biz/hr/compliance/burlg_test.go` | VERIFIED | 227 lines, 12 test functions, all PASS |
| `backend/internal/biz/hr/compliance/arbzg_test.go` | VERIFIED | 133 lines, 15 test functions, all PASS |
| `backend/migrations/000046_create_hr_tables.up.sql` | VERIFIED | Exists; contains `hr_leave_requests` and 8 other tables |
| `backend/internal/biz/hr/leave/service.go` | VERIFIED | Exists; `CreateLeaveRequest`, `ApproveLeaveRequest`, all workflow methods present |
| `backend/internal/biz/hr/leave/postgres_repository.go` | VERIFIED | Exists; `FindOverlaps` implemented |
| `backend/internal/biz/hr/absence/service.go` | VERIFIED | Exists; `GetAbsenceCalendar` implemented; reason masking via `ShowAbsenceReason` setting |
| `backend/internal/biz/hr/employee/service.go` | VERIFIED | Exists; `UpdateEmployee` with `callerRole` restriction |
| `backend/internal/biz/hr/timetracking/service.go` | VERIFIED | Exists; `ClockIn`, `ClockOut`, `ErrMaxDailyHoursExceeded` all present |
| `backend/internal/server/hr_grpc.go` | VERIFIED | Exists; `HRGRPCServer` struct with `UnimplementedHRServiceServer` embedding |
| `backend/internal/gateway/route_hr.go` | VERIFIED | Exists; `getHRClient` returns `hrv1.HRServiceClient`; 30+ routes registered |
| `backend/cmd/biz/main.go` | VERIFIED | Contains `hrv1.RegisterHRServiceServer(grpcServer, hrGRPC)` |
| `desktop/src/renderer/src/api/hr-types.ts` | VERIFIED | Contains `LeaveRequest`, `WorkTimeEntry`, `EmployeeProfile` interfaces |
| `desktop/src/renderer/src/api/hr-client.ts` | VERIFIED | Exports `hrLeaveApi`, `hrTimeApi`, `hrAbsenceApi`, `hrEmployeeApi`, `hrSettingsApi` |
| `desktop/src/renderer/src/api/hooks/hr-hooks.ts` | VERIFIED | Contains `useLeaveRequests`, `useClockIn`, `useAbsenceCalendar` and 30+ hooks |
| `desktop/src/renderer/src/components/header/ClockInButton.tsx` | VERIFIED | Exists; uses `useWorkTimeStatus`, `useClockIn`, `useClockOut`, `useStartBreak`, `useEndBreak` |

### Key Link Verification

| From | To | Via | Status | Evidence |
| ---- | -- | --- | ------ | -------- |
| `leave/service.go` | `compliance/burlg.go` | `compliance.CalculateLeaveBalance` | WIRED | Confirmed: `burlgBalance := compliance.CalculateLeaveBalance(input)` |
| `timetracking/service.go` | `compliance/arbzg.go` | `compliance.CheckWorkTime`, `CheckRestPeriod`, `CalculateAutoBreakDeduction` | WIRED | Confirmed: 5 compliance function calls present in service |
| `hr_grpc.go` | `leave/service.go` | `leaveService.*` delegation | WIRED | Confirmed: `leaveService` field; `s.leaveService.CreateLeaveRequest(...)` etc. |
| `route_hr.go` | `hr_grpc.go` | `hrv1.HRServiceClient` via gRPC | WIRED | Confirmed: `getHRClient()` returns `hrv1.NewHRServiceClient(conn)` |
| `cmd/biz/main.go` | `hr_grpc.go` | `hrv1.RegisterHRServiceServer` | WIRED | Confirmed: `hrv1.RegisterHRServiceServer(grpcServer, hrGRPC)` |
| `hr-hooks.ts` | `hr-client.ts` | Named API group imports | WIRED | Confirmed: `import { hrLeaveApi, hrTimeApi, hrAbsenceApi, hrEmployeeApi, hrSettingsApi } from '../hr-client'` |
| `AbsenceCalendar.tsx` | `hr-hooks.ts` | `useAbsenceCalendar` | WIRED | Confirmed: import and usage at component level |
| `ClockInButton.tsx` | `hr-hooks.ts` | `useClockIn`, `useClockOut`, `useWorkTimeStatus` | WIRED | Confirmed: all 5 time tracking hooks imported and called |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
| ----------- | ------------ | ----------- | ------ | -------- |
| HR-01 | 13-01, 13-02, 13-04 | Employee can submit leave/vacation requests for manager approval with workflow | SATISFIED | Full workflow: create (pending) -> approve/reject/cancel; manager auth via `verifyApprover`; frontend: `AbwesenheitenTab` + `HRApprovalDialog` |
| HR-02 | 13-01, 13-02, 13-04 | System tracks leave balance with BUrlG-compliant calculation (pro-rata, carry-over) | SATISFIED | `CalculateLeaveBalance` with part-time pro-rata, mid-year pro-rata, March 31 carryover; 12 passing tests; balance displayed in `AbwesenheitenTab` |
| HR-03 | 13-02, 13-04 | Team absence calendar shows who is out when | SATISFIED | `AbsenceCalendar.tsx` with `useAbsenceCalendar`, Gantt-style bars, department filter; standalone HR module view per locked context decision |
| HR-04 | 13-01, 13-03, 13-04 | Employee can clock in/out for time tracking with daily and weekly summaries | SATISFIED | `ClockIn`/`ClockOut` services; `GetDailySummary`/`GetWeeklySummary`; `ClockInButton` in header; `ZeiterfassungTab` with summary hooks |
| HR-05 | 13-01, 13-03, 13-04 | Time tracking enforces ArbZG rules (max 8h/10h, 11h rest, break requirements) | SATISFIED | ArbZG engine: 8h/9h/10h severity thresholds, 10h block (`ErrMaxDailyHoursExceeded`), 11h rest (`CheckRestPeriod`), break auto-deduction; toast notifications in `hr-hooks.ts` |
| HR-06 | 13-02, 13-04 | Employee profiles include department, position, contract type, access-controlled document storage | SATISFIED | `hr_employee_profiles` table; role-based `UpdateEmployee`; category visibility (hr_only/manager/employee); `DokumenteTab` with `useEmployeeDocuments` |
| HR-07 | 13-02, 13-04 | Sick leave recording with AU (doctor's note) upload after 3 days | SATISFIED | `RecordSickLeave` with configurable `AUThresholdDays`; sets `au_document_required` flag; `AbwesenheitenTab` handles sick leave dialog flow |

**All 7 requirements (HR-01 through HR-07) satisfied.**

### Anti-Patterns Scan

| File | Pattern Found | Severity | Assessment |
| ---- | ------------- | -------- | ---------- |
| `HRApprovalDialog.tsx` line 55 | Variable named `allPendingData` but queries `status: 'approved'` | Info | Misleading variable name; correct behavior -- queries approved requests to detect overlaps with the pending request being reviewed. Not a stub. |
| `team/AbsenceCalendar.tsx` | `placeholder="Abteilung..."` | Info | UI input placeholder text, not implementation stub |

No blockers or functional stubs found. All backend services have substantive implementations (no `return {}` or `return Response.json({ message: "Not implemented" })`). All frontend pages import and use real TanStack Query hooks.

### Build Verification

| Check | Result |
| ----- | ------ |
| `go test ./internal/biz/hr/compliance/...` | PASS -- 27 tests (12 BUrlG + 15 ArbZG) |
| `go test ./internal/biz/hr/leave/...` | PASS -- 20 tests |
| `go test ./internal/biz/hr/absence/...` | PASS -- 4 tests |
| `go test ./internal/biz/hr/employee/...` | PASS -- 14 tests |
| `go test ./internal/biz/hr/timetracking/...` | PASS -- 16 tests |
| `go build ./cmd/biz/` | PASS |
| `go build ./cmd/gateway/` | PASS |
| `npx tsc --noEmit` (desktop) | PASS -- zero errors |
| All 9 phase commits verified in git log | PASS -- e81f8ba through cf521b9 |

### Human Verification Required

The following items need runtime testing to fully validate:

#### 1. Leave Request Form UX

**Test:** Open Abwesenheiten tab in profil, click "Urlaub beantragen," select date range with half-day options enabled for start/end, select morning or afternoon period, submit.
**Expected:** Request appears in pending list. Manager sees it in HRApprovalDialog with all details including half-day periods.
**Why human:** Form interaction, date pickers, conditional half-day dropdowns cannot be verified statically.

#### 2. ArbZG Toast Notifications on Clock-Out

**Test:** Clock in, work a simulated 8h/9h/10h period (or adjust system clock), clock out.
**Expected:** Sonner toast appears with appropriate severity: info at 8h ("ArbZG: Regelarbeitszeit..."), warning at 9h ("Noch maximal 1 Stunde..."), error at >10h ("Hoechstarbeitszeit...").
**Why human:** Toast rendering and conditional logic on async mutation response requires runtime execution.

#### 3. Carryover Expiry Banners

**Test:** Set carryover days > 0 in the database, view AbwesenheitenTab before March 31 and after March 31.
**Expected:** Pre-expiry warning banner before deadline; expired banner after deadline showing carryover is lost.
**Why human:** Date-conditional rendering requires specific test dates or database manipulation.

#### 4. Absence Calendar Color Rendering

**Test:** Create approved leave requests of different types (Urlaub, Krank), view team absence calendar. Then toggle `show_absence_reason` to false in HR settings.
**Expected:** Gantt bars show blue for Urlaub, red for Krank. After disabling reason visibility, all bars show neutral gray labeled "Abwesend."
**Why human:** Color display and visual rendering require UI inspection.

#### 5. Sick Leave AU Upload Prompt

**Test:** Record sick leave for 3+ consecutive days via sick leave recording UI.
**Expected:** Dialog indicates AU document is required; upload prompt or indicator appears.
**Why human:** Multi-step conditional interaction requires runtime with configured threshold.

---

## Summary

Phase 13 goal is fully achieved. All 7 success criteria are verified with real implementations:

- **BUrlG compliance engine** (Plan 01): Pure functions with 12 test cases covering all edge cases -- pro-rata for part-time and mid-year starts, half-day precision, March 31 carryover expiry.
- **ArbZG compliance engine** (Plan 01): Pure functions with 15 test cases -- break thresholds, severity levels (8h/9h/10h), 11h rest period, auto-deduction.
- **Leave management service** (Plan 02): Full state machine (pending -> approved/rejected/cancelled), BUrlG balance enforcement, overlap detection, sick leave AU flagging.
- **Absence calendar service** (Plan 02): Department-filtered query with configurable reason masking.
- **Employee service** (Plan 02): Role-based field restrictions, document management with category visibility.
- **Time tracking service** (Plan 03): ClockIn/ClockOut with ArbZG checks embedded in response, 10h daily block, 11h rest validation, auto-break deduction.
- **gRPC server + gateway** (Plan 03): All 30 RPCs implemented and 30+ HTTP routes exposed; both binaries compile clean.
- **Frontend integration** (Plan 04): 30+ TanStack Query hooks with ArbZG toast notifications; all 8 HR pages wired to real API; ClockInButton in header with live RAF timer; TypeScript compiles with zero errors.

No payroll features were built -- salary/Lohn deferred to Bexio/Abacus/RmA integrations as required. 81 backend tests pass. Both binaries and the desktop TypeScript compile clean.

---

_Verified: 2026-02-19T23:45:00Z_
_Verifier: Claude (gsd-verifier)_
