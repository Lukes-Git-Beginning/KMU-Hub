package schichten

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

const arbzgMinRestDuration = 11 * time.Hour

// JArbSchG limits for employees flagged as minors. They are stricter than the
// ArbZG limits that apply to everyone and are checked on top of them, never
// instead of them.
const (
	// §14: no work before 06:00 or after 20:00.
	jarbschgEarliestStartHour = 6
	jarbschgLatestEndHour     = 20
	// §8: at most eight hours a day.
	jarbschgMaxShiftDuration = 8 * time.Hour
)

// jarbschgLocation is the wall-clock zone the hour limits are read in. A shift
// stored as 19:00 UTC is 21:00 in Berlin and therefore night work; comparing
// the UTC hour would miss exactly the violations this check exists for.
//
// lean: fixed to the tenant timezone default rather than read from
// hr_company_settings.timezone (which schichten has no access to). Thread the
// tenant's zone through once a tenant outside Europe/Berlin exists.
func jarbschgLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.UTC
	}
	return loc
}

// Service handles schichten business logic.
type Service struct {
	repo Repository
}

// NewService creates a new schichten service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types
// ============================================================================

// CreateShiftInput contains data to create a shift.
type CreateShiftInput struct {
	TenantID    uuid.UUID
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    *string
	CreatedBy   *uuid.UUID
}

// UpdateShiftInput contains fields that can be updated on a shift.
type UpdateShiftInput struct {
	TenantID    uuid.UUID
	ShiftID     uuid.UUID
	Title       *string
	Description *string
	StartTime   *time.Time
	EndTime     *time.Time
	Location    *string
}

// ListShiftsInput contains pagination and filtering for shift listing.
type ListShiftsInput struct {
	TenantID uuid.UUID
	Status   *ShiftStatus
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

// AssignEmployeeInput contains data to assign an employee to a shift.
type AssignEmployeeInput struct {
	TenantID   uuid.UUID
	ShiftID    uuid.UUID
	EmployeeID uuid.UUID
	AssignedBy *uuid.UUID
}

// CreateTemplateInput contains data to create a shift template.
type CreateTemplateInput struct {
	TenantID        uuid.UUID
	Name            string
	Description     string
	DayOfWeek       int
	StartHour       int
	StartMinute     int
	DurationMinutes int
	Location        *string
}

// UpdateTemplateInput contains fields that can be updated on a template.
type UpdateTemplateInput struct {
	TenantID        uuid.UUID
	TemplateID      uuid.UUID
	Name            *string
	Description     *string
	DayOfWeek       *int
	StartHour       *int
	StartMinute     *int
	DurationMinutes *int
	Location        *string
}

// ApplyTemplateInput contains data to generate shifts from a template.
type ApplyTemplateInput struct {
	TenantID   uuid.UUID
	TemplateID uuid.UUID
	RangeStart time.Time
	RangeEnd   time.Time
	CreatedBy  *uuid.UUID
}

// ListTemplatesInput contains pagination for template listing.
type ListTemplatesInput struct {
	TenantID uuid.UUID
	Page     int
	PageSize int
}

// GetShiftStatsInput contains optional time range for stats.
type GetShiftStatsInput struct {
	TenantID uuid.UUID
	From     *time.Time
	To       *time.Time
}

// ============================================================================
// Shift methods
// ============================================================================

// CreateShift creates a new shift in draft status.
func (s *Service) CreateShift(ctx context.Context, input CreateShiftInput) (*Shift, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	if !input.EndTime.After(input.StartTime) {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	shift := &Shift{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Title:       title,
		Description: input.Description,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Status:      ShiftStatusDraft,
		Location:    input.Location,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateShift(ctx, shift); err != nil {
		return nil, fmt.Errorf("create shift: %w", err)
	}

	slog.Info("shift created", "shift_id", shift.ID, "tenant_id", shift.TenantID)
	return shift, nil
}

// UpdateShift updates mutable fields on a shift.
func (s *Service) UpdateShift(ctx context.Context, input UpdateShiftInput) (*Shift, error) {
	shift, err := s.repo.GetShift(ctx, input.TenantID, input.ShiftID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return nil, ErrInvalidInput
		}
		shift.Title = t
	}
	if input.Description != nil {
		shift.Description = *input.Description
	}
	if input.StartTime != nil {
		shift.StartTime = *input.StartTime
	}
	if input.EndTime != nil {
		shift.EndTime = *input.EndTime
	}
	if !shift.EndTime.After(shift.StartTime) {
		return nil, ErrInvalidInput
	}
	if input.Location != nil {
		shift.Location = input.Location
	}
	shift.UpdatedAt = time.Now()

	if err := s.repo.UpdateShift(ctx, shift); err != nil {
		return nil, fmt.Errorf("update shift: %w", err)
	}

	slog.Info("shift updated", "shift_id", shift.ID, "tenant_id", shift.TenantID)
	return shift, nil
}

// DeleteShift hard-deletes a shift and its assignments (CASCADE).
func (s *Service) DeleteShift(ctx context.Context, tenantID, shiftID uuid.UUID) error {
	if _, err := s.repo.GetShift(ctx, tenantID, shiftID); err != nil {
		return err
	}
	if err := s.repo.DeleteShift(ctx, tenantID, shiftID); err != nil {
		return fmt.Errorf("delete shift: %w", err)
	}
	slog.Info("shift deleted", "shift_id", shiftID, "tenant_id", tenantID)
	return nil
}

// GetShift retrieves a single shift by ID.
func (s *Service) GetShift(ctx context.Context, tenantID, shiftID uuid.UUID) (*Shift, error) {
	return s.repo.GetShift(ctx, tenantID, shiftID)
}

// ListShifts retrieves shifts with optional filtering and pagination.
func (s *Service) ListShifts(ctx context.Context, input ListShiftsInput) ([]*Shift, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListShiftsFilter{
		Status: input.Status,
		From:   input.From,
		To:     input.To,
	}
	return s.repo.ListShifts(ctx, input.TenantID, filter, offset, input.PageSize)
}

// PublishShifts sets all draft shifts within a time window to published.
// Idempotent: already-published shifts are not double-counted.
func (s *Service) PublishShifts(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error) {
	if !to.After(from) {
		return 0, ErrInvalidInput
	}
	count, err := s.repo.PublishShifts(ctx, tenantID, from, to)
	if err != nil {
		return 0, fmt.Errorf("publish shifts: %w", err)
	}
	slog.Info("shifts published", "tenant_id", tenantID, "count", count, "from", from, "to", to)
	return count, nil
}

// ============================================================================
// Assignment methods
// ============================================================================

// AssignEmployee assigns an employee to a shift after ArbZG validation and capacity check.
func (s *Service) AssignEmployee(ctx context.Context, input AssignEmployeeInput) (*ShiftAssignment, error) {
	// Pre-check: verify shift exists under tenant
	shift, err := s.repo.GetShift(ctx, input.TenantID, input.ShiftID)
	if err != nil {
		return nil, err
	}

	// Guard 1: capacity check — reject if shift is already full
	if shift.Capacity != nil {
		count, countErr := s.repo.CountAssignments(ctx, input.TenantID, input.ShiftID)
		if countErr != nil {
			return nil, fmt.Errorf("capacity check: %w", countErr)
		}
		if count >= *shift.Capacity {
			slog.Warn("shift capacity reached",
				"shift_id", input.ShiftID,
				"tenant_id", input.TenantID,
				"capacity", *shift.Capacity,
				"current_count", count,
			)
			return nil, ErrShiftFull
		}
	}

	// Guard 2: ArbZG §5 rest-period check before DB write
	if err := s.validateRestPeriod(ctx, input.TenantID, input.EmployeeID, shift.StartTime, shift.EndTime); err != nil {
		return nil, err
	}

	// Guard 3: JArbSchG limits, for employees flagged as minors only. A shift
	// a minor may not legally work must not become an assignment that the
	// compliance endpoint later reports as a violation after the fact.
	if err := s.validateMinorProtection(ctx, input.TenantID, input.EmployeeID, shift.StartTime, shift.EndTime); err != nil {
		return nil, err
	}

	assignment := &ShiftAssignment{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		ShiftID:    input.ShiftID,
		EmployeeID: input.EmployeeID,
		AssignedAt: time.Now(),
		AssignedBy: input.AssignedBy,
	}

	if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("assign employee: %w", err)
	}

	slog.Info("employee assigned to shift",
		"shift_id", input.ShiftID,
		"employee_id", input.EmployeeID,
		"tenant_id", input.TenantID,
	)
	return assignment, nil
}

// UnassignEmployee removes an employee from a shift.
func (s *Service) UnassignEmployee(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) error {
	// Pre-check: ensure shift exists under tenant
	if _, err := s.repo.GetShift(ctx, tenantID, shiftID); err != nil {
		return err
	}

	if err := s.repo.DeleteAssignment(ctx, tenantID, shiftID, employeeID); err != nil {
		return fmt.Errorf("unassign employee: %w", err)
	}

	slog.Info("employee unassigned from shift",
		"shift_id", shiftID,
		"employee_id", employeeID,
		"tenant_id", tenantID,
	)
	return nil
}

// ListAssignments returns all assignments for a shift.
func (s *Service) ListAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) ([]*ShiftAssignment, error) {
	if _, err := s.repo.GetShift(ctx, tenantID, shiftID); err != nil {
		return nil, err
	}
	return s.repo.ListAssignments(ctx, tenantID, shiftID)
}

// ============================================================================
// Template methods
// ============================================================================

// CreateTemplate creates a new shift template.
func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput) (*ShiftTemplate, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	if input.DayOfWeek < 0 || input.DayOfWeek > 6 {
		return nil, ErrInvalidInput
	}
	if input.StartHour < 0 || input.StartHour > 23 {
		return nil, ErrInvalidInput
	}
	if input.StartMinute < 0 || input.StartMinute > 59 {
		return nil, ErrInvalidInput
	}
	if input.DurationMinutes <= 0 {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	tmpl := &ShiftTemplate{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		Name:            name,
		Description:     input.Description,
		DayOfWeek:       input.DayOfWeek,
		StartHour:       input.StartHour,
		StartMinute:     input.StartMinute,
		DurationMinutes: input.DurationMinutes,
		Location:        input.Location,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateTemplate(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("create shift template: %w", err)
	}

	slog.Info("shift template created", "template_id", tmpl.ID, "tenant_id", tmpl.TenantID)
	return tmpl, nil
}

// UpdateTemplate updates mutable fields on a template.
func (s *Service) UpdateTemplate(ctx context.Context, input UpdateTemplateInput) (*ShiftTemplate, error) {
	tmpl, err := s.repo.GetTemplate(ctx, input.TenantID, input.TemplateID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		tmpl.Name = n
	}
	if input.Description != nil {
		tmpl.Description = *input.Description
	}
	if input.DayOfWeek != nil {
		if *input.DayOfWeek < 0 || *input.DayOfWeek > 6 {
			return nil, ErrInvalidInput
		}
		tmpl.DayOfWeek = *input.DayOfWeek
	}
	if input.StartHour != nil {
		if *input.StartHour < 0 || *input.StartHour > 23 {
			return nil, ErrInvalidInput
		}
		tmpl.StartHour = *input.StartHour
	}
	if input.StartMinute != nil {
		if *input.StartMinute < 0 || *input.StartMinute > 59 {
			return nil, ErrInvalidInput
		}
		tmpl.StartMinute = *input.StartMinute
	}
	if input.DurationMinutes != nil {
		if *input.DurationMinutes <= 0 {
			return nil, ErrInvalidInput
		}
		tmpl.DurationMinutes = *input.DurationMinutes
	}
	if input.Location != nil {
		tmpl.Location = input.Location
	}
	tmpl.UpdatedAt = time.Now()

	if err := s.repo.UpdateTemplate(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("update shift template: %w", err)
	}

	slog.Info("shift template updated", "template_id", tmpl.ID, "tenant_id", tmpl.TenantID)
	return tmpl, nil
}

// DeleteTemplate removes a shift template.
func (s *Service) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	if _, err := s.repo.GetTemplate(ctx, tenantID, templateID); err != nil {
		return err
	}
	if err := s.repo.DeleteTemplate(ctx, tenantID, templateID); err != nil {
		return fmt.Errorf("delete shift template: %w", err)
	}
	slog.Info("shift template deleted", "template_id", templateID, "tenant_id", tenantID)
	return nil
}

// GetTemplate retrieves a single template by ID.
func (s *Service) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error) {
	return s.repo.GetTemplate(ctx, tenantID, templateID)
}

// ListTemplates returns paginated templates.
func (s *Service) ListTemplates(ctx context.Context, input ListTemplatesInput) ([]*ShiftTemplate, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListTemplates(ctx, input.TenantID, offset, input.PageSize)
}

// ApplyTemplate generates draft shifts for every matching weekday within the date range.
func (s *Service) ApplyTemplate(ctx context.Context, input ApplyTemplateInput) ([]*Shift, error) {
	if !input.RangeEnd.After(input.RangeStart) {
		return nil, ErrInvalidInput
	}

	tmpl, err := s.repo.GetTemplate(ctx, input.TenantID, input.TemplateID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.UTC
	}

	var created []*Shift

	// Walk through each day in [RangeStart, RangeEnd)
	current := input.RangeStart.In(loc)
	end := input.RangeEnd.In(loc)

	for !current.After(end) {
		// time.Weekday: Sunday=0 … Saturday=6 — same encoding as template
		if int(current.Weekday()) == tmpl.DayOfWeek {
			shiftStart := time.Date(
				current.Year(), current.Month(), current.Day(),
				tmpl.StartHour, tmpl.StartMinute, 0, 0, loc,
			)
			shiftEnd := shiftStart.Add(time.Duration(tmpl.DurationMinutes) * time.Minute)

			now := time.Now()
			shift := &Shift{
				ID:          uuid.New(),
				TenantID:    input.TenantID,
				Title:       tmpl.Name,
				Description: tmpl.Description,
				StartTime:   shiftStart,
				EndTime:     shiftEnd,
				Status:      ShiftStatusDraft,
				Location:    tmpl.Location,
				CreatedBy:   input.CreatedBy,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// Idempotency guard: skip CreateShift if an identical shift already exists for this day.
			exists, checkErr := s.repo.ShiftExistsForTemplate(ctx, input.TenantID, shiftStart, shiftEnd, tmpl.Name)
			if checkErr != nil {
				return nil, fmt.Errorf("apply template: check duplicate for %s: %w", shiftStart.Format("2006-01-02"), checkErr)
			}
			if !exists {
				if createErr := s.repo.CreateShift(ctx, shift); createErr != nil {
					return nil, fmt.Errorf("apply template: create shift for %s: %w", shiftStart.Format("2006-01-02"), createErr)
				}
				created = append(created, shift)
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	slog.Info("template applied",
		"template_id", input.TemplateID,
		"tenant_id", input.TenantID,
		"shifts_created", len(created),
	)
	return created, nil
}

// ============================================================================
// Compliance & Stats
// ============================================================================

// CheckArbzgCompliance validates the ArbZG §5 rest requirement and, for
// employees flagged as minors, the JArbSchG limits on top of it. It writes
// nothing to the DB.
func (s *Service) CheckArbzgCompliance(ctx context.Context, tenantID, employeeID uuid.UUID, newStart, newEnd time.Time) (bool, string, error) {
	if err := s.validateRestPeriod(ctx, tenantID, employeeID, newStart, newEnd); err != nil {
		return false, err.Error(), nil
	}
	if err := s.validateMinorProtection(ctx, tenantID, employeeID, newStart, newEnd); err != nil {
		// A rule violation is an answer, a failed lookup is not: reporting
		// "compliant: false, no reason" for a dead database would read as a
		// clean verdict.
		if isJArbSchGViolation(err) {
			return false, err.Error(), nil
		}
		return false, "", err
	}
	return true, "", nil
}

// GetShiftStats returns aggregated shift statistics.
func (s *Service) GetShiftStats(ctx context.Context, input GetShiftStatsInput) (*ShiftStats, error) {
	return s.repo.GetStats(ctx, input.TenantID, input.From, input.To)
}

// ============================================================================
// SwapRequest input types
// ============================================================================

// CreateSwapRequestInput contains data to create a shift-swap request.
type CreateSwapRequestInput struct {
	TenantID              uuid.UUID
	AssignmentID          uuid.UUID
	RequestedByEmployeeID uuid.UUID
	SwapWithEmployeeID    uuid.UUID
	ShiftID               uuid.UUID
	Reason                string
	IdempotencyKey        string
}

// ListSwapRequestsInput contains pagination and filtering for swap request listing.
type ListSwapRequestsInput struct {
	TenantID uuid.UUID
	ShiftID  *uuid.UUID
	Status   *SwapRequestStatus
	Page     int
	PageSize int
}

// ============================================================================
// SwapRequest methods
// ============================================================================

// CreateSwapRequest creates a new shift-swap request with idempotency.
func (s *Service) CreateSwapRequest(ctx context.Context, input CreateSwapRequestInput) (*SwapRequest, error) {
	if input.IdempotencyKey == "" {
		return nil, ErrInvalidInput
	}
	if input.RequestedByEmployeeID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if input.SwapWithEmployeeID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if input.RequestedByEmployeeID == input.SwapWithEmployeeID {
		return nil, ErrInvalidInput
	}
	if input.ShiftID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	req := &SwapRequest{
		ID:                    uuid.New(),
		TenantID:              input.TenantID,
		AssignmentID:          input.AssignmentID,
		RequestedByEmployeeID: input.RequestedByEmployeeID,
		SwapWithEmployeeID:    input.SwapWithEmployeeID,
		ShiftID:               input.ShiftID,
		Status:                SwapRequestStatusPending,
		Reason:                input.Reason,
		IdempotencyKey:        input.IdempotencyKey,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.CreateSwapRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("create swap request: %w", err)
	}

	slog.Info("swap request created",
		"swap_request_id", req.ID,
		"shift_id", req.ShiftID,
		"requested_by", req.RequestedByEmployeeID,
		"swap_with", req.SwapWithEmployeeID,
		"tenant_id", req.TenantID,
	)
	return req, nil
}

// ListSwapRequests returns paginated swap requests with optional filtering.
func (s *Service) ListSwapRequests(ctx context.Context, input ListSwapRequestsInput) ([]*SwapRequest, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	filter := SwapRequestFilter{
		ShiftID: input.ShiftID,
		Status:  input.Status,
	}
	return s.repo.ListSwapRequests(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ApproveSwapRequest approves a swap request and atomically swaps the two assignments.
func (s *Service) ApproveSwapRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*SwapRequest, error) {
	req, err := s.repo.GetSwapRequest(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}

	if req.Status != SwapRequestStatusPending {
		return nil, ErrSwapAlreadyProcessed
	}

	// Atomically: update status + swap assignments in one transaction.
	if err := s.repo.SwapAssignmentsForRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("approve swap request (swap assignments): %w", err)
	}
	if err := s.repo.UpdateSwapRequestStatus(ctx, tenantID, requestID, SwapRequestStatusApproved); err != nil {
		return nil, fmt.Errorf("approve swap request (update status): %w", err)
	}

	req.Status = SwapRequestStatusApproved
	req.UpdatedAt = time.Now()

	slog.Info("swap request approved",
		"swap_request_id", requestID,
		"tenant_id", tenantID,
	)
	return req, nil
}

// RejectSwapRequest rejects a pending swap request.
func (s *Service) RejectSwapRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*SwapRequest, error) {
	req, err := s.repo.GetSwapRequest(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}

	if req.Status != SwapRequestStatusPending {
		return nil, ErrSwapAlreadyProcessed
	}

	if err := s.repo.UpdateSwapRequestStatus(ctx, tenantID, requestID, SwapRequestStatusRejected); err != nil {
		return nil, fmt.Errorf("reject swap request: %w", err)
	}

	req.Status = SwapRequestStatusRejected
	req.UpdatedAt = time.Now()

	slog.Info("swap request rejected",
		"swap_request_id", requestID,
		"tenant_id", tenantID,
	)
	return req, nil
}

// ============================================================================
// Internal helpers
// ============================================================================

// validateRestPeriod enforces ArbZG §5: minimum 11 hours rest between consecutive
// shifts for the same employee. Called before AssignEmployee writes to DB.
//
// Both directions are checked:
//   - The shift that ends BEFORE newStart (previous shift)
//   - The shift that starts AFTER newEnd (following shift)
//
// Cases:
//   - No adjacent shift found → OK.
//   - restDuration >= 11h → OK.
//   - restDuration < 11h → ErrArbzgViolation.
func (s *Service) validateRestPeriod(ctx context.Context, tenantID, employeeID uuid.UUID, newStart, newEnd time.Time) error {
	// Direction 1: find the most recent shift end BEFORE newStart.
	latestEnd, err := s.repo.LatestShiftEndBeforeForEmployee(ctx, tenantID, employeeID, newStart)
	if err != nil {
		return fmt.Errorf("arbzg check (before): %w", err)
	}
	if latestEnd != nil {
		restBefore := newStart.Sub(*latestEnd)
		if restBefore < arbzgMinRestDuration {
			slog.Warn("ArbZG §5 violation detected (shift before)",
				"employee_id", employeeID,
				"tenant_id", tenantID,
				"rest_duration", restBefore,
				"required", arbzgMinRestDuration,
			)
			return ErrArbzgViolation
		}
	}

	// Direction 2: find the earliest shift start AFTER newEnd.
	earliestStart, err := s.repo.EarliestShiftStartAfterForEmployee(ctx, tenantID, employeeID, newEnd)
	if err != nil {
		return fmt.Errorf("arbzg check (after): %w", err)
	}
	if earliestStart != nil {
		restAfter := earliestStart.Sub(newEnd)
		if restAfter < arbzgMinRestDuration {
			slog.Warn("ArbZG §5 violation detected (shift after)",
				"employee_id", employeeID,
				"tenant_id", tenantID,
				"rest_duration", restAfter,
				"required", arbzgMinRestDuration,
			)
			return ErrArbzgViolation
		}
	}

	return nil
}

// validateMinorProtection applies the JArbSchG limits to employees whose HR
// profile carries the minor flag. Employees without the flag are unaffected,
// which is why it can sit in front of every assignment without changing the
// behaviour of an existing plan.
//
// Only the first violation is reported. The response carries one reason
// string, the same shape the rest-period check has always returned, so the
// frontend learns nothing new.
func (s *Service) validateMinorProtection(ctx context.Context, tenantID, employeeID uuid.UUID, newStart, newEnd time.Time) error {
	isMinor, err := s.repo.IsMinorEmployee(ctx, tenantID, employeeID)
	if err != nil {
		return fmt.Errorf("jarbschg check: %w", err)
	}
	if !isMinor {
		return nil
	}

	if violation := checkMinorShiftLimits(newStart, newEnd); violation != nil {
		slog.Warn("JArbSchG violation detected",
			"employee_id", employeeID,
			"tenant_id", tenantID,
			"shift_start", newStart,
			"shift_end", newEnd,
			"violation", violation.Error(),
		)
		return violation
	}
	return nil
}

// checkMinorShiftLimits holds the JArbSchG rules themselves. It touches no
// storage, so the rules are testable on their own.
func checkMinorShiftLimits(newStart, newEnd time.Time) error {
	loc := jarbschgLocation()
	start := newStart.In(loc)
	end := newEnd.In(loc)

	// §16/§17: no Saturdays or Sundays. A shift running past midnight is
	// checked on both ends, so a Friday night into Saturday is caught too.
	if isWeekend(start) || isWeekend(end) {
		return ErrJArbSchGWeekend
	}

	// §14: the working day is bounded by 06:00 and 20:00 of the start date.
	// Anchoring both bounds to the start date means a shift ending after
	// midnight lands after the upper bound rather than looking like an early
	// morning.
	earliest := time.Date(start.Year(), start.Month(), start.Day(), jarbschgEarliestStartHour, 0, 0, 0, loc)
	latest := time.Date(start.Year(), start.Month(), start.Day(), jarbschgLatestEndHour, 0, 0, 0, loc)
	if start.Before(earliest) || end.After(latest) {
		return ErrJArbSchGNightWork
	}

	// §8: at most eight hours.
	if end.Sub(start) > jarbschgMaxShiftDuration {
		return ErrJArbSchGDailyHours
	}

	return nil
}

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func isJArbSchGViolation(err error) bool {
	return errors.Is(err, ErrJArbSchGNightWork) ||
		errors.Is(err, ErrJArbSchGDailyHours) ||
		errors.Is(err, ErrJArbSchGWeekend)
}
