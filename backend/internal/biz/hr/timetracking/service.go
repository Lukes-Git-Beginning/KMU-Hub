// Package timetracking provides the business logic for HR work time tracking.
// It implements clock in/out lifecycle with ArbZG compliance enforcement
// (severity levels, break auto-deduction, rest period validation, 10h block),
// manual break tracking, daily/weekly summaries, and time correction workflow.
//
// This is HR work time tracking (ArbZG compliance), completely SEPARATE from
// Phase 6 task timer (internal/work/timeentry/). Different table, different service,
// different rules.
package timetracking

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/biz/hr/compliance"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// Service handles work time tracking business logic with ArbZG enforcement.
type Service struct {
	workTimeRepo     WorkTimeRepository
	breakRepo        BreakRepository
	employeeRepo     employee.EmployeeRepository
	settingsRepo     leave.HRSettingsRepository
	pool             *pgxpool.Pool // For event emission
	emitter          EventEmitter
	categoryRepo     TimeCategoryRepository
	templateRepo     TimeTemplateRepository
	projectRepo      TimeProjectRepository
	weekApprovalRepo WeekApprovalRepository
}

// NewService creates a new time tracking service.
func NewService(
	workTimeRepo WorkTimeRepository,
	breakRepo BreakRepository,
	employeeRepo employee.EmployeeRepository,
	settingsRepo leave.HRSettingsRepository,
	pool *pgxpool.Pool,
) *Service {
	return &Service{
		workTimeRepo: workTimeRepo,
		breakRepo:    breakRepo,
		employeeRepo: employeeRepo,
		settingsRepo: settingsRepo,
		pool:         pool,
	}
}

// SetExtendedRepos wires the Fall-B repositories after construction.
func (s *Service) SetExtendedRepos(
	categoryRepo TimeCategoryRepository,
	templateRepo TimeTemplateRepository,
	projectRepo TimeProjectRepository,
	weekApprovalRepo WeekApprovalRepository,
) {
	s.categoryRepo = categoryRepo
	s.templateRepo = templateRepo
	s.projectRepo = projectRepo
	s.weekApprovalRepo = weekApprovalRepo
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// emitEvent emits a notification event if the emitter is configured.
// Nil-safe: does nothing if no emitter is set.
func (s *Service) emitEvent(ctx context.Context, eventType, actorID, resourceID string, targetUserIDs []string, title, body, deepLink string) {
	if s.emitter == nil {
		return
	}
	payload := models.EventPayload{
		Type:          eventType,
		Priority:      "low",
		ActorID:       actorID,
		ModuleID:      event.ModuleHR,
		ResourceID:    resourceID,
		TargetUserIDs: targetUserIDs,
		Title:         title,
		Body:          body,
		DeepLink:      deepLink,
		Timestamp:     time.Now(),
	}
	if err := s.emitter.EmitHREvent(ctx, payload); err != nil {
		slog.Error("failed to emit hr event",
			"type", eventType,
			"resource_id", resourceID,
			"error", err,
		)
	}
}

// ClockIn creates a new active work time entry.
// Checks: no existing active shift, 11-hour rest period, 10h daily ArbZG block.
// Returns the entry and an ArbZG rest period check result (for frontend toast).
func (s *Service) ClockIn(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRWorkTimeEntry, *compliance.WorkTimeCheckResult, error) {
	// Check no active shift exists
	existing, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrAlreadyClockedIn
	}

	now := time.Now()

	// ArbZG 10h block: check today's daily summary
	todaySummary, summaryErr := s.workTimeRepo.GetDailySummary(ctx, employeeID, now)
	if summaryErr != nil {
		return nil, nil, summaryErr
	}
	if todaySummary.NetWorkMinutes >= 600 {
		return nil, nil, ErrMaxDailyHoursExceeded
	}

	// Check 11-hour rest period
	previousEnd, prevErr := s.workTimeRepo.GetPreviousShiftEnd(ctx, employeeID, now)
	if prevErr != nil {
		slog.Warn("failed to check previous shift end for rest period validation",
			"employee_id", employeeID,
			"error", prevErr,
		)
	}

	var checkResult *compliance.WorkTimeCheckResult
	restViolation, restHours := compliance.CheckRestPeriod(now, previousEnd)
	if restViolation {
		checkResult = &compliance.WorkTimeCheckResult{
			RestViolation:   true,
			RestHoursActual: restHours,
			Severity:        compliance.SeverityWarning,
			Message:         "ArbZG: Ruhezeit von 11 Stunden nicht eingehalten.",
		}
	}

	entry := &models.HRWorkTimeEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		ClockIn:    now,
		Status:     models.WorkTimeStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if createErr := s.workTimeRepo.Create(ctx, entry); createErr != nil {
		return nil, nil, createErr
	}

	slog.Info("hr.time.clock_in",
		"work_time_entry_id", entry.ID,
		"employee_id", employeeID,
		"rest_violation", restViolation,
	)

	s.emitEvent(ctx, event.EventShiftStarted, employeeID.String(), entry.ID.String(), nil,
		"Schicht gestartet", "", "/hr/zeiterfassung")

	return entry, checkResult, nil
}

// ClockOut completes an active shift, calculating net work minutes with ArbZG compliance.
// Returns the completed entry and an ArbZG compliance check result (for frontend toast).
func (s *Service) ClockOut(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRWorkTimeEntry, *compliance.WorkTimeCheckResult, error) {
	entry, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		return nil, nil, ErrNotClockedIn
	}

	now := time.Now()

	// End any active break first
	activeBreak, breakErr := s.breakRepo.GetActiveBreak(ctx, entry.ID)
	if breakErr == nil && activeBreak != nil {
		duration := int(now.Sub(activeBreak.StartTime).Minutes())
		activeBreak.EndTime = &now
		activeBreak.DurationMinutes = &duration
		if err := s.breakRepo.Update(ctx, activeBreak); err != nil {
			return nil, nil, err
		}
	}

	// Calculate total worked minutes from clock_in to now
	totalWorkedMinutes := int(now.Sub(entry.ClockIn).Minutes())

	// Get manual breaks for this shift, sum break_minutes
	breaks, breakListErr := s.breakRepo.ListByWorkTimeEntry(ctx, entry.ID)
	manualBreakMinutes := 0
	if breakListErr == nil {
		for _, b := range breaks {
			if b.DurationMinutes != nil {
				manualBreakMinutes += *b.DurationMinutes
			}
		}
	}

	// Auto break deduction
	autoDeduction := compliance.CalculateAutoBreakDeduction(totalWorkedMinutes, manualBreakMinutes)

	// Net work minutes
	netWorkMinutes := compliance.CalculateNetWorkMinutes(entry.ClockIn, now, manualBreakMinutes, autoDeduction)

	// ArbZG severity check on total worked minutes
	checkResult := compliance.CheckWorkTime(totalWorkedMinutes)
	checkResult.BreakMinutesTaken = manualBreakMinutes
	checkResult.BreakMinutesRequired = compliance.CalculateRequiredBreak(totalWorkedMinutes)
	checkResult.BreakDeficit = autoDeduction

	// Update entry
	entry.ClockOut = &now
	entry.BreakMinutes = manualBreakMinutes
	entry.AutoBreakDeducted = autoDeduction
	entry.NetWorkMinutes = &netWorkMinutes
	entry.Status = models.WorkTimeStatusCompleted
	entry.UpdatedAt = now

	if updateErr := s.workTimeRepo.Update(ctx, entry); updateErr != nil {
		return nil, nil, updateErr
	}

	slog.Info("hr.time.clock_out",
		"work_time_entry_id", entry.ID,
		"employee_id", employeeID,
		"total_worked_minutes", totalWorkedMinutes,
		"manual_break_minutes", manualBreakMinutes,
		"auto_break_deducted", autoDeduction,
		"net_work_minutes", netWorkMinutes,
		"severity", checkResult.Severity,
	)

	s.emitEvent(ctx, event.EventShiftEnded, employeeID.String(), entry.ID.String(), nil,
		"Schicht beendet", "", "/hr/zeiterfassung")

	return entry, &checkResult, nil
}

// StartBreak starts a break for an active shift.
func (s *Service) StartBreak(ctx context.Context, employeeID uuid.UUID) (*models.HRBreakEntry, error) {
	shift, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, ErrNotClockedIn
	}

	// Check no active break
	existingBreak, breakErr := s.breakRepo.GetActiveBreak(ctx, shift.ID)
	if breakErr != nil {
		return nil, breakErr
	}
	if existingBreak != nil {
		return nil, ErrAlreadyOnBreak
	}

	now := time.Now()
	breakEntry := &models.HRBreakEntry{
		ID:              uuid.New(),
		WorkTimeEntryID: shift.ID,
		StartTime:       now,
		CreatedAt:       now,
	}

	if createErr := s.breakRepo.Create(ctx, breakEntry); createErr != nil {
		return nil, createErr
	}

	slog.Info("hr.time.break_start",
		"break_id", breakEntry.ID,
		"work_time_entry_id", shift.ID,
		"employee_id", employeeID,
	)

	return breakEntry, nil
}

// EndBreak ends an active break, calculates duration, and updates shift break total.
func (s *Service) EndBreak(ctx context.Context, employeeID uuid.UUID) (*models.HRBreakEntry, error) {
	shift, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, ErrNotClockedIn
	}

	activeBreak, breakErr := s.breakRepo.GetActiveBreak(ctx, shift.ID)
	if breakErr != nil {
		return nil, breakErr
	}
	if activeBreak == nil {
		return nil, ErrNotOnBreak
	}

	now := time.Now()
	duration := int(now.Sub(activeBreak.StartTime).Minutes())
	activeBreak.EndTime = &now
	activeBreak.DurationMinutes = &duration

	if updateErr := s.breakRepo.Update(ctx, activeBreak); updateErr != nil {
		return nil, updateErr
	}

	// Update shift's total break_minutes (sum of all completed breaks)
	breaks, listErr := s.breakRepo.ListByWorkTimeEntry(ctx, shift.ID)
	if listErr == nil {
		totalBreak := 0
		for _, b := range breaks {
			if b.DurationMinutes != nil {
				totalBreak += *b.DurationMinutes
			}
		}
		shift.BreakMinutes = totalBreak
		shift.UpdatedAt = now
		if err := s.workTimeRepo.Update(ctx, shift); err != nil {
			return nil, err
		}
	}

	slog.Info("hr.time.break_end",
		"break_id", activeBreak.ID,
		"work_time_entry_id", shift.ID,
		"employee_id", employeeID,
		"duration_minutes", duration,
	)

	return activeBreak, nil
}

// GetActiveShift returns the active shift and its break entries for display.
func (s *Service) GetActiveShift(ctx context.Context, employeeID uuid.UUID) (*models.HRWorkTimeEntry, []*models.HRBreakEntry, error) {
	shift, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, nil, err
	}
	if shift == nil {
		return nil, nil, nil
	}

	breaks, breakErr := s.breakRepo.ListByWorkTimeEntry(ctx, shift.ID)
	if breakErr != nil {
		slog.Warn("failed to list breaks for active shift",
			"work_time_entry_id", shift.ID,
			"error", breakErr,
		)
		breaks = []*models.HRBreakEntry{}
	}

	return shift, breaks, nil
}

// ListWorkTimeEntries retrieves work time entries with optional filtering.
func (s *Service) ListWorkTimeEntries(ctx context.Context, filter WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error) {
	return s.workTimeRepo.List(ctx, filter)
}

// GetDailySummary returns the daily work time summary for an employee.
func (s *Service) GetDailySummary(ctx context.Context, employeeID uuid.UUID, date time.Time) (*DailySummary, error) {
	return s.workTimeRepo.GetDailySummary(ctx, employeeID, date)
}

// GetWeeklySummary returns the weekly work time summary for an employee.
func (s *Service) GetWeeklySummary(ctx context.Context, employeeID uuid.UUID, weekStart time.Time) (*WeeklySummary, error) {
	return s.workTimeRepo.GetWeeklySummary(ctx, employeeID, weekStart)
}

// SubmitTimeCorrection creates a correction entry pending manager approval.
func (s *Service) SubmitTimeCorrection(ctx context.Context, tenantID, employeeID uuid.UUID, input CorrectionInput) (*models.HRWorkTimeEntry, error) {
	// Validate original entry exists
	_, origErr := s.workTimeRepo.GetByID(ctx, input.OriginalEntryID)
	if origErr != nil {
		return nil, origErr
	}

	now := time.Now()
	netWork := compliance.CalculateNetWorkMinutes(input.CorrectedClockIn, input.CorrectedClockOut, input.CorrectedBreakMinutes, 0)
	clockOut := input.CorrectedClockOut

	correction := &models.HRWorkTimeEntry{
		ID:               uuid.New(),
		TenantID:         tenantID,
		EmployeeID:       employeeID,
		ClockIn:          input.CorrectedClockIn,
		ClockOut:         &clockOut,
		BreakMinutes:     input.CorrectedBreakMinutes,
		NetWorkMinutes:   &netWork,
		Status:           models.WorkTimeStatusCorrectionPending,
		IsCorrection:     true,
		OriginalEntryID:  &input.OriginalEntryID,
		CorrectionReason: input.Reason,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if createErr := s.workTimeRepo.Create(ctx, correction); createErr != nil {
		return nil, createErr
	}

	slog.Info("hr.time.correction_requested",
		"correction_id", correction.ID,
		"original_entry_id", input.OriginalEntryID,
		"employee_id", employeeID,
		"reason", input.Reason,
	)

	return correction, nil
}

// ApproveTimeCorrection approves a pending time correction.
func (s *Service) ApproveTimeCorrection(ctx context.Context, correctionID, approverID uuid.UUID) (*models.HRWorkTimeEntry, error) {
	correction, err := s.workTimeRepo.GetByID(ctx, correctionID)
	if err != nil {
		return nil, err
	}

	if correction.Status != models.WorkTimeStatusCorrectionPending {
		return nil, ErrCorrectionNotFound
	}

	now := time.Now()
	correction.Status = models.WorkTimeStatusCorrectionApproved
	correction.CorrectionApprovedBy = &approverID
	correction.CorrectionApprovedAt = &now
	correction.UpdatedAt = now

	if updateErr := s.workTimeRepo.Update(ctx, correction); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("hr.time.correction_approved",
		"correction_id", correctionID,
		"approver_id", approverID,
		"original_entry_id", correction.OriginalEntryID,
	)

	return correction, nil
}

// ============================================================================
// Manual Entry
// ============================================================================

// CreateManualEntry inserts a completed work time entry manually (no active-shift lifecycle).
func (s *Service) CreateManualEntry(ctx context.Context, input ManualEntryInput) (*models.HRWorkTimeEntry, error) {
	if !input.ClockOut.After(input.ClockIn) {
		return nil, ErrInvalidManualEntry
	}

	// Calculate net work minutes
	totalMinutes := int(input.ClockOut.Sub(input.ClockIn).Minutes())
	netMinutes := totalMinutes - input.BreakMinutes
	if netMinutes < 0 {
		netMinutes = 0
	}

	now := time.Now()
	clockOut := input.ClockOut
	entry := &models.HRWorkTimeEntry{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		EmployeeID:      input.EmployeeID,
		ClockIn:         input.ClockIn,
		ClockOut:        &clockOut,
		BreakMinutes:    input.BreakMinutes,
		NetWorkMinutes:  &netMinutes,
		Status:          models.WorkTimeStatusCompleted,
		CategoryID:      input.CategoryID,
		ProjectID:       input.ProjectID,
		LocationLat:     input.LocationLat,
		LocationLng:     input.LocationLng,
		LocationAddress: &input.LocationAddress,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if input.LocationAddress == "" {
		entry.LocationAddress = nil
	}

	if err := s.workTimeRepo.Create(ctx, entry); err != nil {
		return nil, err
	}

	slog.Info("hr.time.manual_entry",
		"work_time_entry_id", entry.ID,
		"employee_id", input.EmployeeID,
		"net_work_minutes", netMinutes,
	)

	return entry, nil
}

// ============================================================================
// Time Balance
// ============================================================================

// GetTimeBalance returns the accumulated overtime/undertime balance for the current period.
func (s *Service) GetTimeBalance(ctx context.Context, tenantID, employeeID uuid.UUID) (balanceMinutes, targetWeeklyMinutes int, periodStart time.Time, err error) {
	// Get employee profile for contracted weekly hours
	ep, epErr := s.employeeRepo.GetByUserID(ctx, employeeID)
	workDays := 5
	if epErr == nil && ep.WorkDaysPerWeek > 0 {
		workDays = ep.WorkDaysPerWeek
	}
	targetWeeklyMinutes = workDays * 8 * 60 // 8h per work day

	// Current week start (Monday)
	now := time.Now()
	weekStart := now
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	weekly, summaryErr := s.workTimeRepo.GetWeeklySummary(ctx, employeeID, weekStart)
	if summaryErr != nil {
		return 0, targetWeeklyMinutes, weekStart, summaryErr
	}

	balanceMinutes = weekly.NetWorkMinutes - targetWeeklyMinutes
	return balanceMinutes, targetWeeklyMinutes, weekStart, nil
}

// ============================================================================
// Time Analytics
// ============================================================================

// GetTimeAnalytics returns KPIs and trends for the employee over a given range.
func (s *Service) GetTimeAnalytics(ctx context.Context, tenantID, employeeID uuid.UUID, rangeStr string) (*TimeAnalytics, error) {
	ep, epErr := s.employeeRepo.GetByUserID(ctx, employeeID)
	workDays := 5
	if epErr == nil && ep.WorkDaysPerWeek > 0 {
		workDays = ep.WorkDaysPerWeek
	}
	targetWeeklyMinutes := workDays * 8 * 60

	now := time.Now()
	var start time.Time
	var numDays int
	if rangeStr == "month" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		numDays = now.Day()
	} else {
		// Default: week
		start = now
		for start.Weekday() != time.Monday {
			start = start.AddDate(0, 0, -1)
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		numDays = 7
	}

	targetMinutes := targetWeeklyMinutes * numDays / 5 // pro-rated

	// Load the whole day range in a single aggregated query (was numDays round-trips).
	dailies, dailyErr := s.workTimeRepo.GetDailySummaryRange(ctx, employeeID, start, numDays)
	if dailyErr != nil {
		return nil, dailyErr
	}
	var trend []DayTrendEntry
	totalMinutes := 0
	for _, daily := range dailies {
		trend = append(trend, DayTrendEntry{
			Date:          daily.Date,
			NetMinutes:    daily.NetWorkMinutes,
			TargetMinutes: workDays * 8 * 60 / 5,
		})
		totalMinutes += daily.NetWorkMinutes
	}

	overtimeMinutes := totalMinutes - targetMinutes
	if overtimeMinutes < 0 {
		overtimeMinutes = 0
	}

	avgDaily := 0
	if numDays > 0 {
		avgDaily = totalMinutes / numDays
	}

	// Compute project breakdown for the period
	byProject := []ProjectBreakdown{}
	if s.projectRepo != nil {
		pb, pbErr := s.workTimeRepo.GetProjectBreakdown(ctx, tenantID, employeeID, start, start.AddDate(0, 0, numDays-1))
		if pbErr == nil {
			byProject = pb
		}
	}

	return &TimeAnalytics{
		TotalMinutes:    totalMinutes,
		TargetMinutes:   targetMinutes,
		OvertimeMinutes: overtimeMinutes,
		AvgDailyMinutes: avgDaily,
		DayTrend:        trend,
		ByProject:       byProject,
	}, nil
}

// ============================================================================
// Team Time
// ============================================================================

// GetTeamTime returns aggregated weekly time for all employees of the tenant.
func (s *Service) GetTeamTime(ctx context.Context, tenantID uuid.UUID, weekStart time.Time) ([]TeamTimeEntry, error) {
	// Ensure Monday
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	employees, _, listErr := s.employeeRepo.List(ctx, employee.EmployeeFilter{TenantID: tenantID, Page: 1, PerPage: 500})
	if listErr != nil {
		return nil, listErr
	}

	// Batch-load weekly summaries, active shifts and week approvals to avoid an N+1.
	// Previously this issued ~9 queries per employee (GetWeeklySummary = 7 daily
	// round-trips, plus GetActiveShift and GetByEmployeeWeek) — up to ~4500 sequential
	// round-trips for 500 employees. Now it is a constant 3 queries regardless of N.
	// Work time entries are keyed by employee_id = user_id.
	empIDs := make([]uuid.UUID, 0, len(employees))
	for _, emp := range employees {
		empIDs = append(empIDs, emp.UserID)
	}

	weeklyByEmp, weeklyErr := s.workTimeRepo.GetWeeklySummaryBatch(ctx, empIDs, weekStart)
	if weeklyErr != nil {
		return nil, weeklyErr
	}
	activeByEmp, activeErr := s.workTimeRepo.GetActiveShiftEmployeeIDs(ctx, empIDs)
	if activeErr != nil {
		return nil, activeErr
	}
	statusByEmp := make(map[uuid.UUID]string)
	if s.weekApprovalRepo != nil {
		approvals, waErr := s.weekApprovalRepo.ListByWeek(ctx, tenantID, weekStart)
		if waErr == nil {
			for _, wa := range approvals {
				statusByEmp[wa.EmployeeID] = string(wa.Status)
			}
		}
	}

	teamEntries := make([]TeamTimeEntry, 0, len(employees))
	for _, emp := range employees {
		weekMinutes := 0
		if weekly, ok := weeklyByEmp[emp.UserID]; ok && weekly != nil {
			weekMinutes = weekly.NetWorkMinutes
		}

		workDays := emp.WorkDaysPerWeek
		if workDays <= 0 {
			workDays = 5
		}
		targetMinutes := workDays * 8 * 60

		overtime := weekMinutes - targetMinutes
		if overtime < 0 {
			overtime = 0
		}

		weekStatus := "open"
		if st, ok := statusByEmp[emp.UserID]; ok {
			weekStatus = st
		}

		teamEntries = append(teamEntries, TeamTimeEntry{
			EmployeeID:      emp.UserID,
			Name:            emp.UserName,
			Department:      emp.Department,
			WeekMinutes:     weekMinutes,
			TargetMinutes:   targetMinutes,
			OvertimeMinutes: overtime,
			ClockedIn:       activeByEmp[emp.UserID],
			WeekStatus:      weekStatus,
		})
	}

	return teamEntries, nil
}

// ============================================================================
// Week Approvals
// ============================================================================

// GetMyWeekStatus returns the approval status for an employee's week.
func (s *Service) GetMyWeekStatus(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, int, int, error) {
	// Ensure Monday
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	ep, _ := s.employeeRepo.GetByUserID(ctx, employeeID)
	workDays := 5
	if ep != nil && ep.WorkDaysPerWeek > 0 {
		workDays = ep.WorkDaysPerWeek
	}
	targetMinutes := workDays * 8 * 60

	weekly, weekErr := s.workTimeRepo.GetWeeklySummary(ctx, employeeID, weekStart)
	totalMinutes := 0
	if weekErr == nil {
		totalMinutes = weekly.NetWorkMinutes
	}

	wa, waErr := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStart)
	if waErr != nil {
		// No record yet → synthesise an open one
		now := time.Now()
		wa = &models.HRWeekApproval{
			ID:         uuid.New(),
			TenantID:   tenantID,
			EmployeeID: employeeID,
			WeekStart:  weekStart,
			Status:     models.WeekApprovalOpen,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	return wa, totalMinutes, targetMinutes, nil
}

// SubmitWeek transitions an employee's week to submitted state.
func (s *Service) SubmitWeek(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, error) {
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	wa, waErr := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStart)
	now := time.Now()
	if waErr != nil {
		// Create fresh record
		wa = &models.HRWeekApproval{
			ID:         uuid.New(),
			TenantID:   tenantID,
			EmployeeID: employeeID,
			WeekStart:  weekStart,
			Status:     models.WeekApprovalOpen,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	if wa.Status == models.WeekApprovalSubmitted || wa.Status == models.WeekApprovalApproved {
		return nil, ErrWeekAlreadySubmitted
	}

	wa.Status = models.WeekApprovalSubmitted
	wa.SubmittedAt = &now
	wa.UpdatedAt = now

	if err := s.weekApprovalRepo.Upsert(ctx, wa); err != nil {
		return nil, err
	}

	slog.Info("hr.time.week_submitted",
		"employee_id", employeeID,
		"week_start", weekStart.Format("2006-01-02"),
	)
	return wa, nil
}

// ApproveWeek approves a submitted week.
func (s *Service) ApproveWeek(ctx context.Context, tenantID, approverID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, error) {
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	wa, waErr := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStart)
	if waErr != nil {
		return nil, ErrWeekApprovalNotFound
	}

	if wa.Status != models.WeekApprovalSubmitted {
		return nil, ErrWeekNotSubmitted
	}

	now := time.Now()
	wa.Status = models.WeekApprovalApproved
	wa.ApprovedBy = &approverID
	wa.ApprovedAt = &now
	wa.UpdatedAt = now

	if err := s.weekApprovalRepo.Upsert(ctx, wa); err != nil {
		return nil, err
	}

	slog.Info("hr.time.week_approved",
		"employee_id", employeeID,
		"approver_id", approverID,
		"week_start", weekStart.Format("2006-01-02"),
	)
	return wa, nil
}

// RejectWeek rejects a submitted week with a reason.
func (s *Service) RejectWeek(ctx context.Context, tenantID, approverID, employeeID uuid.UUID, weekStart time.Time, reason string) (*models.HRWeekApproval, error) {
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	wa, waErr := s.weekApprovalRepo.GetByEmployeeWeek(ctx, tenantID, employeeID, weekStart)
	if waErr != nil {
		return nil, ErrWeekApprovalNotFound
	}

	if wa.Status != models.WeekApprovalSubmitted {
		return nil, ErrWeekNotSubmitted
	}

	now := time.Now()
	wa.Status = models.WeekApprovalRejected
	wa.RejectionReason = &reason
	wa.UpdatedAt = now

	if err := s.weekApprovalRepo.Upsert(ctx, wa); err != nil {
		return nil, err
	}

	slog.Info("hr.time.week_rejected",
		"employee_id", employeeID,
		"approver_id", approverID,
		"week_start", weekStart.Format("2006-01-02"),
	)
	return wa, nil
}

// ============================================================================
// Time Categories
// ============================================================================

func (s *Service) ListTimeCategories(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeCategory, error) {
	return s.categoryRepo.List(ctx, tenantID)
}

func (s *Service) CreateTimeCategory(ctx context.Context, tenantID uuid.UUID, name, color, icon string, isDefault bool) (*models.HRTimeCategory, error) {
	now := time.Now()
	cat := &models.HRTimeCategory{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Color:     color,
		Icon:      icon,
		IsDefault: isDefault,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.categoryRepo.Create(ctx, cat); err != nil {
		return nil, err
	}
	slog.Info("hr.time.category_created", "category_id", cat.ID, "name", name)
	return cat, nil
}

func (s *Service) UpdateTimeCategory(ctx context.Context, id uuid.UUID, name, color, icon string, isDefault bool) (*models.HRTimeCategory, error) {
	cat, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTimeCategoryNotFound
	}
	cat.Name = name
	cat.Color = color
	cat.Icon = icon
	cat.IsDefault = isDefault
	cat.UpdatedAt = time.Now()
	if err := s.categoryRepo.Update(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *Service) DeleteTimeCategory(ctx context.Context, id uuid.UUID) error {
	return s.categoryRepo.Delete(ctx, id)
}

// ============================================================================
// Time Templates
// ============================================================================

func (s *Service) ListTimeTemplates(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeTemplate, error) {
	return s.templateRepo.List(ctx, tenantID)
}

func (s *Service) CreateTimeTemplate(ctx context.Context, tenantID uuid.UUID, name string, categoryID *uuid.UUID, description string, estimatedMinutes int) (*models.HRTimeTemplate, error) {
	now := time.Now()
	t := &models.HRTimeTemplate{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             name,
		CategoryID:       categoryID,
		Description:      description,
		EstimatedMinutes: estimatedMinutes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.templateRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	slog.Info("hr.time.template_created", "template_id", t.ID, "name", name)
	return t, nil
}

func (s *Service) DeleteTimeTemplate(ctx context.Context, id uuid.UUID) error {
	return s.templateRepo.Delete(ctx, id)
}

// ============================================================================
// Time Projects
// ============================================================================

func (s *Service) ListTimeProjects(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeProject, error) {
	return s.projectRepo.List(ctx, tenantID)
}

func (s *Service) CreateTimeProject(ctx context.Context, tenantID uuid.UUID, name, customerName, color string, billableDefault bool) (*models.HRTimeProject, error) {
	now := time.Now()
	p := &models.HRTimeProject{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            name,
		CustomerName:    customerName,
		Color:           color,
		BillableDefault: billableDefault,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.projectRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	slog.Info("hr.time.project_created", "project_id", p.ID, "name", name)
	return p, nil
}

// GetWorkTimeStatus returns a convenience status for the header quick-toggle button.
func (s *Service) GetWorkTimeStatus(ctx context.Context, employeeID uuid.UUID) (*WorkTimeStatus, error) {
	status := &WorkTimeStatus{}

	shift, err := s.workTimeRepo.GetActiveShift(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	if shift != nil {
		status.IsClockedIn = true
		status.CurrentShiftStart = &shift.ClockIn

		// Check for active break
		activeBreak, breakErr := s.breakRepo.GetActiveBreak(ctx, shift.ID)
		if breakErr == nil && activeBreak != nil {
			status.IsOnBreak = true
			status.CurrentBreakStart = &activeBreak.StartTime
		}
	}

	// Today's total minutes
	now := time.Now()
	todaySummary, summaryErr := s.workTimeRepo.GetDailySummary(ctx, employeeID, now)
	if summaryErr == nil {
		status.TodayTotalMinutes = todaySummary.NetWorkMinutes
	}

	// ArbZG severity
	checkResult := compliance.CheckWorkTime(status.TodayTotalMinutes)
	status.ArbZGSeverity = string(checkResult.Severity)

	return status, nil
}
