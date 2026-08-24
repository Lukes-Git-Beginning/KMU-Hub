package vendoraccess

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service errors.
var (
	ErrInvalidStatus        = errors.New("vendoraccess: request is not in a valid state for this operation")
	ErrSensitiveAckRequired = errors.New("vendoraccess: sensitive_ack required for a sensitive scope")
)

// sensitiveAreas mirrors desktop/src/renderer/src/api/vendor-access-types.ts
// VENDOR_ACCESS_AREAS. There is no shared source of truth for this static
// catalogue across the FE/BE boundary -- keep both lists in sync by hand.
var sensitiveAreas = map[string]bool{
	"hr_data": true,
	"salary":  true,
}

func hasSensitiveScope(scope []string) bool {
	for _, s := range scope {
		if sensitiveAreas[s] {
			return true
		}
	}
	return false
}

// Service orchestrates the vendor access request lifecycle
// (RBAC R-5 B, GDAP-light v3).
type Service struct {
	repo Repository
}

// NewService creates a new vendor access Service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create inserts a new pending vendor access request for the caller's tenant.
// Not wired to an HTTP route -- the frontend contract has no create call.
func (s *Service) Create(ctx context.Context, req *models.VendorAccessRequest) (*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}

	req.ID = uuid.New()
	req.TenantID = tenantID
	req.Status = models.VendorAccessStatusPending
	req.CreatedAt = time.Now().UTC()

	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("vendoraccess: failed to create request: %w", err)
	}
	return req, nil
}

// List returns all vendor access requests for the caller's tenant.
func (s *Service) List(ctx context.Context) ([]*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}
	return s.repo.ListRequests(ctx, tenantID)
}

// Approve moves a pending or counter-proposed request to active. A sensitive
// scope requires an explicit sensitiveAck from the caller (422 otherwise).
func (s *Service) Approve(ctx context.Context, id, actorID uuid.UUID, sensitiveAck bool) (*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetRequest(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Status != models.VendorAccessStatusPending && req.Status != models.VendorAccessStatusCounterProposed {
		return nil, ErrInvalidStatus
	}

	sensitive := hasSensitiveScope(req.Scope)
	if sensitive && !sensitiveAck {
		return nil, ErrSensitiveAckRequired
	}

	now := time.Now().UTC()
	req.Status = models.VendorAccessStatusActive
	req.ApprovedAt = &now
	req.ApprovedBy = &actorID
	req.CounterProposedStart = nil
	if sensitive {
		ack := true
		req.SensitiveAck = &ack
	}

	if err := s.repo.UpdateStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("vendoraccess: failed to approve request: %w", err)
	}

	slog.Info("vendoraccess: request approved", "request_id", id, "actor_id", actorID)

	// Re-fetch to resolve the reviewer's display name via the join.
	return s.repo.GetRequest(ctx, tenantID, id)
}

// Decline moves a pending or counter-proposed request to declined.
func (s *Service) Decline(ctx context.Context, id uuid.UUID) (*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetRequest(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Status != models.VendorAccessStatusPending && req.Status != models.VendorAccessStatusCounterProposed {
		return nil, ErrInvalidStatus
	}

	req.Status = models.VendorAccessStatusDeclined
	req.CounterProposedStart = nil

	if err := s.repo.UpdateStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("vendoraccess: failed to decline request: %w", err)
	}

	slog.Info("vendoraccess: request declined", "request_id", id)
	return req, nil
}

// CounterPropose moves a pending request to counter_proposed with an
// alternative start date. Only valid from pending (not from an already
// counter-proposed request) -- matches the frontend's single round-trip UX.
func (s *Service) CounterPropose(ctx context.Context, id uuid.UUID, proposedStart time.Time) (*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetRequest(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Status != models.VendorAccessStatusPending {
		return nil, ErrInvalidStatus
	}

	req.Status = models.VendorAccessStatusCounterProposed
	req.CounterProposedStart = &proposedStart

	if err := s.repo.UpdateStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("vendoraccess: failed to counter-propose request: %w", err)
	}

	slog.Info("vendoraccess: request counter-proposed", "request_id", id, "proposed_start", proposedStart)
	return req, nil
}

// Revoke moves an active request to revoked. This changes only the request's
// own status column: there is no other consumer of VendorAccessStatusActive
// anywhere in internal/ or cmd/ (verified 2026-08-23 per
// cov-gateway-security-vendor-access-routes) -- no middleware, session check,
// or connection gate reads vendor_access_requests. The record is a
// consent/audit trail for the "ein Server pro Kunde" delivery model, not an
// access-control mechanism; whatever channel the vendor actually used to
// reach the customer's data (SSH, DB console, etc.) keeps working until it is
// closed by other means. A revoke here means "we withdrew consent", not "we
// cut off access".
func (s *Service) Revoke(ctx context.Context, id, actorID uuid.UUID) (*models.VendorAccessRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("vendoraccess: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetRequest(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Status != models.VendorAccessStatusActive {
		return nil, ErrInvalidStatus
	}

	now := time.Now().UTC()
	req.Status = models.VendorAccessStatusRevoked
	req.RevokedAt = &now
	req.RevokedBy = &actorID

	if err := s.repo.UpdateStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("vendoraccess: failed to revoke request: %w", err)
	}

	slog.Info("vendoraccess: request revoked", "request_id", id, "actor_id", actorID)

	// Re-fetch to resolve the reviewer's display name via the join.
	return s.repo.GetRequest(ctx, tenantID, id)
}
