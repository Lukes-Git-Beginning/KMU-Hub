// Package changerequest implements HR self-service change requests: an employee
// who may not edit their own master data proposes a change, somebody holding
// team:data_personal:edit decides, and only an approval writes to the personnel
// file.
//
// The point of the flow is the decision, so the two rules that carry it are the
// ones to read first: an approver may only decide about people their permission
// scope reaches (approveScopeAllows), and an approval may only touch a field
// from the allowlist below (proposableFields).
package changerequest

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
)

// proposableField ties a field name the frontend sends to the profile column an
// approval writes and to the drawer it belongs to.
type proposableField struct {
	column string
	drawer string
}

// proposableFields is the allowlist. It is deliberately small: every entry has
// to be a column an approval can really write, otherwise the flow accepts a
// proposal that approval can never carry out.
//
// The frontend also offers "phone" and "mobile" (SelfServiceView.tsx). Neither
// exists on hr_employee_profiles or on users — the values it renders come from
// fields the real backend never fills. Accepting a proposal for them would mean
// storing a request that approval must fail on, so they are refused at submit
// time with ErrFieldNotProposable. Adding them means adding the columns first.
var proposableFields = map[string]proposableField{
	"addressStreet":         {column: "address_street", drawer: "personal"},
	"addressCity":           {column: "address_city", drawer: "personal"},
	"addressPostalCode":     {column: "address_postal_code", drawer: "personal"},
	"addressCountry":        {column: "address_country", drawer: "personal"},
	"emergencyContactName":  {column: "emergency_contact_name", drawer: "personal"},
	"emergencyContactPhone": {column: "emergency_contact_phone", drawer: "personal"},
}

const (
	maxFieldLabelLen = 120
	maxValueLen      = 500
)

// Service holds the change request business logic.
type Service struct {
	repo Repository
}

// NewService creates a change request service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListInput selects which requests to return. OwnOnly is what the self-service
// view sets; the inbox leaves it false and gets what ActorScope permits.
type ListInput struct {
	TenantID   uuid.UUID
	ActorID    uuid.UUID
	ActorScope string
	OwnOnly    bool
	Status     string
}

// CreateInput is one proposal. The proposer is ActorID from the auth context —
// never a user id from the request body, or anybody could propose for anybody.
type CreateInput struct {
	TenantID   uuid.UUID
	ActorID    uuid.UUID
	Drawer     string
	Field      string
	FieldLabel string
	OldValue   string
	NewValue   string
}

// DecideInput carries an approval or a rejection. ActorScope is the approver's
// team:data_personal:edit scope as the gateway resolved it.
type DecideInput struct {
	TenantID    uuid.UUID
	ActorID     uuid.UUID
	ActorScope  string
	RequestID   uuid.UUID
	Reason      string
}

// List returns the change requests the actor may see, newest first.
//
// The scope narrows the answer to the same set Approve would accept: "own"
// collapses to the actor's own proposals even without OwnOnly, "team" to the
// actor plus their direct reports. An inbox that listed more would offer rows
// that answer 403 when clicked.
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.HRProfileChangeRequest, error) {
	filter := Filter{TenantID: input.TenantID, Status: input.Status}
	switch {
	case input.OwnOnly, input.ActorScope == auth.ScopeOwn:
		filter.OwnerID = &input.ActorID
	case input.ActorScope == auth.ScopeTeam:
		filter.ManagerID = &input.ActorID
	}
	return s.repo.List(ctx, filter)
}

// Create opens a proposal for one field of the caller's own profile.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.HRProfileChangeRequest, error) {
	spec, ok := proposableFields[input.Field]
	if !ok {
		return nil, ErrFieldNotProposable
	}
	// A drawer that contradicts the field is a client bug, not a second way of
	// classifying it: the drawer decides which section of the UI shows the
	// request, so a mismatch would hide it from the section it belongs to.
	if input.Drawer != "" && input.Drawer != spec.drawer {
		return nil, ErrFieldNotProposable
	}

	newValue := strings.TrimSpace(input.NewValue)
	if newValue == "" || len(newValue) > maxValueLen {
		return nil, ErrFieldNotProposable
	}

	label := strings.TrimSpace(input.FieldLabel)
	if label == "" {
		label = input.Field
	}
	if len(label) > maxFieldLabelLen {
		label = label[:maxFieldLabelLen]
	}

	oldValue := input.OldValue
	if len(oldValue) > maxValueLen {
		oldValue = oldValue[:maxValueLen]
	}

	req := &models.HRProfileChangeRequest{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		UserID:     input.ActorID,
		Drawer:     spec.drawer,
		Field:      input.Field,
		FieldLabel: label,
		OldValue:   oldValue,
		NewValue:   newValue,
		Status:     models.HRChangeRequestPending,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	slog.Info("hr change request submitted",
		"request_id", req.ID,
		"user_id", req.UserID,
		"tenant_id", req.TenantID,
		"field", req.Field,
	)
	return s.repo.GetByID(ctx, input.TenantID, req.ID)
}

// Approve applies the proposed value to the employee profile.
func (s *Service) Approve(ctx context.Context, input DecideInput) (*models.HRProfileChangeRequest, error) {
	req, err := s.decidable(ctx, input)
	if err != nil {
		return nil, err
	}

	spec, ok := proposableFields[req.Field]
	if !ok {
		// The allowlist shrank after the request was filed. Refusing beats
		// writing to a column that is no longer meant to be proposable.
		return nil, ErrFieldNotProposable
	}

	updated, err := s.repo.ApproveAndApply(ctx, input.TenantID, req.ID, input.ActorID, spec.column, req.NewValue, time.Now())
	if err != nil {
		return nil, err
	}

	slog.Info("hr change request approved",
		"request_id", updated.ID,
		"user_id", updated.UserID,
		"decided_by", input.ActorID,
		"field", updated.Field,
	)
	return updated, nil
}

// Reject closes a proposal without touching the profile. The reason is
// mandatory: a rejection the employee cannot understand produces a second,
// identical proposal.
func (s *Service) Reject(ctx context.Context, input DecideInput) (*models.HRProfileChangeRequest, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrReasonRequired
	}
	if len(reason) > maxValueLen {
		reason = reason[:maxValueLen]
	}

	req, err := s.decidable(ctx, input)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updated, err := s.repo.Decide(ctx, input.TenantID, req.ID, models.HRChangeRequestRejected, reason, &input.ActorID, &now)
	if err != nil {
		return nil, err
	}

	slog.Info("hr change request rejected",
		"request_id", updated.ID,
		"user_id", updated.UserID,
		"decided_by", input.ActorID,
	)
	return updated, nil
}

// Cancel withdraws a proposal. Only the proposer may, and only while pending —
// a decided request stays in the record as it was decided.
func (s *Service) Cancel(ctx context.Context, tenantID, actorID, requestID uuid.UUID) (*models.HRProfileChangeRequest, error) {
	req, err := s.repo.GetByID(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}
	if req.UserID != actorID {
		return nil, ErrNotProposer
	}
	if req.Status != models.HRChangeRequestPending {
		return nil, ErrNotPending
	}

	// No decider and no decision time: a withdrawal is not somebody else's call.
	return s.repo.Decide(ctx, tenantID, requestID, models.HRChangeRequestCancelled, req.Reason, nil, nil)
}

// decidable loads the request and answers whether input's actor may decide it.
func (s *Service) decidable(ctx context.Context, input DecideInput) (*models.HRProfileChangeRequest, error) {
	req, err := s.repo.GetByID(ctx, input.TenantID, input.RequestID)
	if err != nil {
		return nil, err
	}
	if req.Status != models.HRChangeRequestPending {
		return nil, ErrNotPending
	}
	allowed, err := s.approveScopeAllows(ctx, input.TenantID, input.ActorID, req.UserID, input.ActorScope)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrOutOfScope
	}
	return req, nil
}

// approveScopeAllows decides whether an approver holding team:data_personal:edit
// at scope may decide about proposer.
//
// This is the first place in the codebase that resolves auth.ScopeTeam against a
// real reporting line (hr_employee_profiles.manager_user_id) instead of treating
// it like auth.ScopeAll — see the note on middleware.PermissionScope. Without it
// a manager could approve changes to the master data of anybody in the tenant,
// which is exactly the check the flow exists for.
//
// An unknown or empty scope resolves to auth.ScopeAll, mirroring
// middleware.PermissionScope: a caller who got past the RequirePermission guard
// but whose token predates the scopes claim keeps behaving as it did.
func (s *Service) approveScopeAllows(ctx context.Context, tenantID, actorID, proposerID uuid.UUID, scope string) (bool, error) {
	switch scope {
	case auth.ScopeOwn:
		return actorID == proposerID, nil
	case auth.ScopeTeam:
		if actorID == proposerID {
			return true, nil
		}
		managerID, err := s.repo.ManagerOf(ctx, tenantID, proposerID)
		if err != nil {
			// No profile means no reporting line to prove, so the team scope
			// cannot reach this person.
			if errors.Is(err, ErrProfileNotFound) {
				return false, nil
			}
			return false, err
		}
		return managerID != nil && *managerID == actorID, nil
	default:
		return true, nil
	}
}
