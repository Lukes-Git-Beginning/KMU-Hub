package changerequest

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Filter narrows a list read. OwnerID set means "only this user's proposals"
// (the self-service view); an empty Status means every status.
//
// ManagerID set means "only proposals this manager may also decide": their own
// plus their direct reports'. It mirrors the team-scope check in
// Service.approveScopeAllows, so an inbox never lists a row that would answer
// 403 when clicked. The filter belongs in the query, not in a post-filter on
// the result — see the note on middleware.PermissionScope.
type Filter struct {
	TenantID  uuid.UUID
	OwnerID   *uuid.UUID
	ManagerID *uuid.UUID
	Status    string
}

// Repository persists profile change requests.
type Repository interface {
	Create(ctx context.Context, req *models.HRProfileChangeRequest) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRProfileChangeRequest, error)
	List(ctx context.Context, filter Filter) ([]*models.HRProfileChangeRequest, error)

	// ApproveAndApply writes value into the employee profile's column and flips
	// the request to approved in ONE transaction. Splitting the two would allow
	// a request marked approved whose value never reached the personnel file —
	// the failure nobody notices until somebody compares the two.
	//
	// column comes from the service's field allowlist, never from the request
	// payload: it is interpolated into the statement, so a client-supplied value
	// would be an injection.
	ApproveAndApply(ctx context.Context, tenantID, id, decidedBy uuid.UUID, column, value string, decidedAt time.Time) (*models.HRProfileChangeRequest, error)

	// Decide closes a request without touching the profile (reject, cancel).
	// decidedBy is nil for a cancellation: the proposer withdrawing is not a
	// decision by somebody else, and the frontend shows no decider for it.
	Decide(ctx context.Context, tenantID, id uuid.UUID, status models.HRChangeRequestStatus, reason string, decidedBy *uuid.UUID, decidedAt *time.Time) (*models.HRProfileChangeRequest, error)

	// ManagerOf resolves the proposer's reporting line for the team-scope check.
	// Returns ErrProfileNotFound when the user has no employee profile.
	ManagerOf(ctx context.Context, tenantID, userID uuid.UUID) (*uuid.UUID, error)
}
