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
	workTimeRepo WorkTimeRepository
	breakRepo    BreakRepository
	employeeRepo employee.EmployeeRepository
	settingsRepo leave.HRSettingsRepository
	pool         *pgxpool.Pool // For event emission
	emitter      EventEmitter
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
		_ = s.breakRepo.Update(ctx, activeBreak)
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
		_ = s.workTimeRepo.Update(ctx, shift)
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
