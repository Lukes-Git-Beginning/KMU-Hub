// Package leave provides the business logic for leave request management.
// It implements the leave approval state machine (pending -> approved/rejected/cancelled),
// BUrlG-compliant balance enforcement, half-day support, AU threshold flagging,
// and manager/HR authorization for approvals.
package leave

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/hr/compliance"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// EmployeeRepository provides access to employee profiles for manager lookup.
type EmployeeRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmployeeProfile, error)
}

// Service handles leave request business logic with BUrlG balance enforcement.
type Service struct {
	leaveRepo    LeaveRequestRepository
	balanceRepo  LeaveBalanceRepository
	typeRepo     LeaveTypeRepository
	settingsRepo HRSettingsRepository
	employeeRepo EmployeeRepository
	emitter      EventEmitter
}

// NewService creates a new leave service.
func NewService(
	leaveRepo LeaveRequestRepository,
	balanceRepo LeaveBalanceRepository,
	typeRepo LeaveTypeRepository,
	settingsRepo HRSettingsRepository,
	employeeRepo EmployeeRepository,
) *Service {
	return &Service{
		leaveRepo:    leaveRepo,
		balanceRepo:  balanceRepo,
		typeRepo:     typeRepo,
		settingsRepo: settingsRepo,
		employeeRepo: employeeRepo,
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
		Priority:      "normal",
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

// CreateLeaveInput contains the data needed to create a leave request.
type CreateLeaveInput struct {
	LeaveTypeID    uuid.UUID
	StartDate      time.Time
	EndDate        time.Time
	IsHalfDayStart bool
	HalfDayPeriodStart *models.HalfDayPeriod
	IsHalfDayEnd   bool
	HalfDayPeriodEnd *models.HalfDayPeriod
	Reason         string
}

// SickLeaveInput contains the data needed to record sick leave.
type SickLeaveInput struct {
	StartDate      time.Time
	EndDate        time.Time
	IsHalfDayStart bool
	HalfDayPeriodStart *models.HalfDayPeriod
	IsHalfDayEnd   bool
	HalfDayPeriodEnd *models.HalfDayPeriod
	Reason         string
}

// CreateLeaveResult contains the leave request and any overlap warnings.
type CreateLeaveResult struct {
	Request  *models.LeaveRequest
	Overlaps []*models.LeaveRequest
}

// ApproveResult contains the result of approving a leave request.
type ApproveResult struct {
	Request  *models.LeaveRequest
	Overlaps []*models.LeaveRequest
}

// CreateLeaveRequest creates a new leave request with BUrlG balance validation.
func (s *Service) CreateLeaveRequest(ctx context.Context, tenantID, employeeID uuid.UUID, input CreateLeaveInput) (*CreateLeaveResult, error) {
	// Validate dates
	if input.EndDate.Before(input.StartDate) {
		return nil, ErrInvalidDateRange
	}

	// Look up leave type
	leaveType, err := s.typeRepo.GetByID(ctx, input.LeaveTypeID)
	if err != nil {
		return nil, err
	}

	// Calculate total days considering half-day flags
	totalDays := calculateTotalDays(input.StartDate, input.EndDate, input.IsHalfDayStart, input.IsHalfDayEnd)

	// Check balance if leave type deducts
	if leaveType.DeductsFromBalance {
		balance, balErr := s.getOrCreateBalance(ctx, tenantID, employeeID)
		if balErr != nil {
			return nil, balErr
		}
		if balance.Remaining.LessThan(totalDays) {
			return nil, ErrInsufficientBalance
		}
	}

	// Determine AU document requirement for sick leave
	auDocRequired := false
	if leaveType.RequiresAUDocument {
		settings, settingsErr := s.settingsRepo.GetByTenant(ctx, tenantID)
		if settingsErr == nil && settings != nil {
			daysFloat, _ := totalDays.Float64()
			if int(math.Ceil(daysFloat)) >= settings.AUThresholdDays {
				auDocRequired = true
			}
		}
	}

	// Determine initial status
	status := models.LeaveStatusPending
	if !leaveType.RequiresApproval {
		status = models.LeaveStatusApproved
	}

	now := time.Now()
	req := &models.LeaveRequest{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		EmployeeID:         employeeID,
		LeaveTypeID:        input.LeaveTypeID,
		StartDate:          input.StartDate,
		EndDate:            input.EndDate,
		IsHalfDayStart:     input.IsHalfDayStart,
		HalfDayPeriodStart: input.HalfDayPeriodStart,
		IsHalfDayEnd:       input.IsHalfDayEnd,
		HalfDayPeriodEnd:   input.HalfDayPeriodEnd,
		TotalDays:          totalDays,
		Reason:             input.Reason,
		Status:             status,
		AUDocumentRequired: auDocRequired,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Auto-approve: set approved_by as system (employee themselves for sick leave)
	if status == models.LeaveStatusApproved {
		req.ApprovedBy = &employeeID
		approvedAt := now
		req.ApprovedAt = &approvedAt

		// Update balance for auto-approved sick leave if it deducts
		if leaveType.DeductsFromBalance {
			if balErr := s.deductBalance(ctx, tenantID, employeeID, totalDays); balErr != nil {
				return nil, balErr
			}
		}
	}

	if createErr := s.leaveRepo.Create(ctx, req); createErr != nil {
		return nil, createErr
	}

	emitType := event.EventLeaveRequested
	if status == models.LeaveStatusApproved {
		emitType = event.EventLeaveApproved
	}
	slog.Info(emitType,
		"leave_request_id", req.ID,
		"employee_id", employeeID,
		"leave_type", leaveType.Key,
		"total_days", totalDays,
		"status", status,
		"au_document_required", auDocRequired,
	)

	s.emitEvent(ctx, emitType, employeeID.String(), req.ID.String(), nil,
		"Urlaubsantrag eingereicht", input.Reason, "/hr/urlaub/"+req.ID.String())

	return &CreateLeaveResult{Request: req}, nil
}

// ApproveLeaveRequest approves a pending leave request.
func (s *Service) ApproveLeaveRequest(ctx context.Context, requestID, approverID uuid.UUID, comment string) (*ApproveResult, error) {
	req, err := s.leaveRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	// Validate current status
	if req.Status != models.LeaveStatusPending {
		return nil, ErrInvalidTransition
	}

	// Prevent self-approval
	if approverID == req.EmployeeID {
		return nil, ErrCannotApproveOwnRequest
	}

	// Verify approver is manager or HR
	if authErr := s.verifyApprover(ctx, approverID, req.EmployeeID); authErr != nil {
		return nil, authErr
	}

	// Check overlaps (warn but allow)
	overlaps, overlapErr := s.leaveRepo.FindOverlaps(ctx, req.EmployeeID, req.StartDate, req.EndDate, &req.ID)
	if overlapErr != nil {
		slog.Warn("failed to check overlaps during approval",
			"leave_request_id", requestID,
			"error", overlapErr,
		)
	}

	// Transition to approved
	now := time.Now()
	req.Status = models.LeaveStatusApproved
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now
	req.ApprovalComment = comment
	req.UpdatedAt = now

	if updateErr := s.leaveRepo.Update(ctx, req); updateErr != nil {
		return nil, updateErr
	}

	// Update leave balance if leave type deducts
	leaveType, typeErr := s.typeRepo.GetByID(ctx, req.LeaveTypeID)
	if typeErr == nil && leaveType.DeductsFromBalance {
		if balErr := s.deductBalance(ctx, req.TenantID, req.EmployeeID, req.TotalDays); balErr != nil {
			slog.Error("failed to deduct leave balance after approval",
				"leave_request_id", requestID,
				"employee_id", req.EmployeeID,
				"error", balErr,
			)
		}
	}

	slog.Info("hr.leave.approved",
		"leave_request_id", requestID,
		"approver_id", approverID,
		"employee_id", req.EmployeeID,
		"overlaps_count", len(overlaps),
	)

	s.emitEvent(ctx, event.EventLeaveApproved, approverID.String(), requestID.String(),
		[]string{req.EmployeeID.String()},
		"Urlaubsantrag genehmigt", "", "/hr/urlaub/"+requestID.String())

	return &ApproveResult{Request: req, Overlaps: overlaps}, nil
}

// RejectLeaveRequest rejects a pending leave request.
func (s *Service) RejectLeaveRequest(ctx context.Context, requestID, approverID uuid.UUID, comment string) error {
	req, err := s.leaveRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}

	if req.Status != models.LeaveStatusPending {
		return ErrInvalidTransition
	}

	// Prevent self-rejection (same auth as approve)
	if approverID == req.EmployeeID {
		return ErrCannotApproveOwnRequest
	}

	if authErr := s.verifyApprover(ctx, approverID, req.EmployeeID); authErr != nil {
		return authErr
	}

	now := time.Now()
	req.Status = models.LeaveStatusRejected
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now
	req.ApprovalComment = comment
	req.UpdatedAt = now

	if updateErr := s.leaveRepo.Update(ctx, req); updateErr != nil {
		return updateErr
	}

	slog.Info("hr.leave.rejected",
		"leave_request_id", requestID,
		"approver_id", approverID,
		"employee_id", req.EmployeeID,
	)

	s.emitEvent(ctx, event.EventLeaveRejected, approverID.String(), requestID.String(),
		[]string{req.EmployeeID.String()},
		"Urlaubsantrag abgelehnt", comment, "/hr/urlaub/"+requestID.String())

	return nil
}

// CancelLeaveRequest cancels a leave request. Only the employee can cancel their own request.
// Pending requests can always be cancelled. Approved requests can only be cancelled before start date.
func (s *Service) CancelLeaveRequest(ctx context.Context, requestID, employeeID uuid.UUID) error {
	req, err := s.leaveRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}

	// Only employee can cancel their own request
	if req.EmployeeID != employeeID {
		return ErrNotAuthorizedToApprove
	}

	switch req.Status {
	case models.LeaveStatusPending:
		// Always allowed
	case models.LeaveStatusApproved:
		// Only before start date
		if !time.Now().Before(req.StartDate) {
			return ErrCannotCancelPastLeave
		}
	default:
		return ErrInvalidTransition
	}

	// If was approved and deducted balance, restore it
	wasApproved := req.Status == models.LeaveStatusApproved
	if wasApproved {
		leaveType, typeErr := s.typeRepo.GetByID(ctx, req.LeaveTypeID)
		if typeErr == nil && leaveType.DeductsFromBalance {
			if balErr := s.restoreBalance(ctx, req.TenantID, req.EmployeeID, req.TotalDays); balErr != nil {
				slog.Error("failed to restore leave balance after cancel",
					"leave_request_id", requestID,
					"employee_id", req.EmployeeID,
					"error", balErr,
				)
			}
		}
	}

	now := time.Now()
	req.Status = models.LeaveStatusCancelled
	req.UpdatedAt = now

	if updateErr := s.leaveRepo.Update(ctx, req); updateErr != nil {
		return updateErr
	}

	slog.Info("hr.leave.cancelled",
		"leave_request_id", requestID,
		"employee_id", employeeID,
		"was_approved", wasApproved,
	)

	return nil
}

// GetLeaveBalance returns the current leave balance for an employee,
// calculated using BUrlG compliance functions.
func (s *Service) GetLeaveBalance(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRLeaveBalance, error) {
	balance, err := s.getOrCreateBalance(ctx, tenantID, employeeID)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// ListLeaveRequests retrieves leave requests with optional filtering.
func (s *Service) ListLeaveRequests(ctx context.Context, filter LeaveRequestFilter) ([]*models.LeaveRequest, int, error) {
	return s.leaveRepo.List(ctx, filter)
}

// GetLeaveRequest retrieves a leave request by ID.
func (s *Service) GetLeaveRequest(ctx context.Context, id uuid.UUID) (*models.LeaveRequest, error) {
	return s.leaveRepo.GetByID(ctx, id)
}

// ListLeaveTypes retrieves all leave types for a tenant.
func (s *Service) ListLeaveTypes(ctx context.Context, tenantID uuid.UUID) ([]*models.LeaveType, error) {
	return s.typeRepo.ListByTenant(ctx, tenantID)
}

// RecordSickLeave is a convenience method to record sick leave (auto-approved).
func (s *Service) RecordSickLeave(ctx context.Context, tenantID, employeeID uuid.UUID, input SickLeaveInput) (*models.LeaveRequest, error) {
	// Find "krank" leave type for tenant
	krankType, err := s.typeRepo.GetByKey(ctx, tenantID, "krank")
	if err != nil {
		return nil, err
	}

	createInput := CreateLeaveInput{
		LeaveTypeID:        krankType.ID,
		StartDate:          input.StartDate,
		EndDate:            input.EndDate,
		IsHalfDayStart:     input.IsHalfDayStart,
		HalfDayPeriodStart: input.HalfDayPeriodStart,
		IsHalfDayEnd:       input.IsHalfDayEnd,
		HalfDayPeriodEnd:   input.HalfDayPeriodEnd,
		Reason:             input.Reason,
	}

	result, createErr := s.CreateLeaveRequest(ctx, tenantID, employeeID, createInput)
	if createErr != nil {
		return nil, createErr
	}

	// The request should be auto-approved since krank type has requires_approval=false
	if result.Request.AUDocumentRequired {
		slog.Info("hr.sick.au_required",
			"leave_request_id", result.Request.ID,
			"employee_id", employeeID,
			"total_days", result.Request.TotalDays,
		)
	}

	return result.Request, nil
}

// ============================================================================
// Internal helpers
// ============================================================================

// calculateTotalDays calculates the total working days for a leave request,
// considering half-day flags. Each half-day subtracts 0.5 from the total.
func calculateTotalDays(startDate, endDate time.Time, isHalfDayStart, isHalfDayEnd bool) decimal.Decimal {
	// Calculate calendar days (inclusive)
	days := int(endDate.Sub(startDate).Hours()/24) + 1
	total := decimal.NewFromInt(int64(days))

	if isHalfDayStart {
		total = total.Sub(decimal.NewFromFloat(0.5))
	}
	if isHalfDayEnd && !startDate.Equal(endDate) {
		// Only subtract for end if it's a different day than start
		total = total.Sub(decimal.NewFromFloat(0.5))
	}
	// Same day with both half-day flags: no additional subtraction needed
	// (already handled by the isHalfDayStart block above, which yields 0.5 days)

	// Ensure minimum 0.5
	halfDay := decimal.NewFromFloat(0.5)
	if total.LessThan(halfDay) {
		total = halfDay
	}

	return total
}

// getOrCreateBalance gets or creates a leave balance for the current year using BUrlG.
func (s *Service) getOrCreateBalance(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRLeaveBalance, error) {
	now := time.Now()
	year := now.Year()

	// Try to get existing balance
	existing, err := s.balanceRepo.GetByEmployeeYear(ctx, tenantID, employeeID, year)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new balance using BUrlG compliance engine
	employee, empErr := s.employeeRepo.GetByUserID(ctx, employeeID)
	if empErr != nil {
		return nil, empErr
	}

	// Get previous year carryover
	var carriedOver decimal.Decimal
	prevBalance, prevErr := s.balanceRepo.GetByEmployeeYear(ctx, tenantID, employeeID, year-1)
	if prevErr == nil && prevBalance != nil {
		carriedOver = prevBalance.Remaining
	}

	// Calculate entitlement using BUrlG
	input := compliance.LeaveEntitlementInput{
		ContractDaysPerWeek:   employee.WorkDaysPerWeek,
		AnnualEntitlementDays: employee.AnnualLeaveDays,
		StartDate:             employee.StartDate,
		CalculationDate:       now,
		UsedDays:              decimal.NewFromInt(0),
		CarriedOverDays:       carriedOver,
	}
	burlgBalance := compliance.CalculateLeaveBalance(input)

	carryoverDeadline := burlgBalance.CarryoverDeadline
	balance := &models.HRLeaveBalance{
		ID:                uuid.New(),
		TenantID:          tenantID,
		EmployeeID:        employeeID,
		Year:              year,
		Entitlement:       burlgBalance.ProRataEntitlement,
		CarriedOver:       burlgBalance.CarriedOver,
		Used:              decimal.NewFromInt(0),
		Remaining:         burlgBalance.Remaining,
		CarryoverExpiresAt: &carryoverDeadline,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if upsertErr := s.balanceRepo.Upsert(ctx, balance); upsertErr != nil {
		return nil, upsertErr
	}

	return balance, nil
}

// deductBalance decreases the leave balance by the given number of days.
func (s *Service) deductBalance(ctx context.Context, tenantID, employeeID uuid.UUID, days decimal.Decimal) error {
	balance, err := s.getOrCreateBalance(ctx, tenantID, employeeID)
	if err != nil {
		return err
	}

	balance.Used = balance.Used.Add(days)
	balance.Remaining = balance.Remaining.Sub(days)
	balance.UpdatedAt = time.Now()

	return s.balanceRepo.Upsert(ctx, balance)
}

// restoreBalance increases the leave balance by the given number of days (for cancel).
func (s *Service) restoreBalance(ctx context.Context, tenantID, employeeID uuid.UUID, days decimal.Decimal) error {
	balance, err := s.getOrCreateBalance(ctx, tenantID, employeeID)
	if err != nil {
		return err
	}

	balance.Used = balance.Used.Sub(days)
	balance.Remaining = balance.Remaining.Add(days)
	balance.UpdatedAt = time.Now()

	return s.balanceRepo.Upsert(ctx, balance)
}

// verifyApprover checks that the approverID is either the employee's manager or has HR role.
// For now, we check the manager relationship. HR role check will be done at the gRPC layer
// where the caller's role is available.
func (s *Service) verifyApprover(ctx context.Context, approverID, employeeID uuid.UUID) error {
	employee, err := s.employeeRepo.GetByUserID(ctx, employeeID)
	if err != nil {
		return err
	}

	// Check if approver is the employee's manager
	if employee.ManagerUserID != nil && *employee.ManagerUserID == approverID {
		return nil
	}

	// If no manager or not the manager, check if approver has an employee profile with HR indicators.
	// The gRPC layer will additionally check for HR role; here we allow if manager matches.
	// HR fallback: if the employee has no manager assigned, any authenticated user with HR role can approve.
	// Since we don't have role info here, we allow if no manager is set (gRPC layer enforces HR role).
	if employee.ManagerUserID == nil {
		return nil // No manager assigned -- HR fallback allowed (gRPC enforces role)
	}

	return ErrNotAuthorizedToApprove
}
