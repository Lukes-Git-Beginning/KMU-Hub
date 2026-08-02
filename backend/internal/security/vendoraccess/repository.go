package vendoraccess

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for vendor access requests.
type Repository interface {
	// CreateRequest inserts a new vendor access request. Not exposed over HTTP
	// today (the frontend has no create call) -- used by tests and by a future
	// Zentria-operator caller.
	CreateRequest(ctx context.Context, req *models.VendorAccessRequest) error

	// GetRequest retrieves a request by ID, scoped to tenantID. Reviewer names
	// (ApprovedByName/RevokedByName) are resolved via a join.
	GetRequest(ctx context.Context, tenantID, id uuid.UUID) (*models.VendorAccessRequest, error)

	// ListRequests returns all requests for tenantID, newest first.
	ListRequests(ctx context.Context, tenantID uuid.UUID) ([]*models.VendorAccessRequest, error)

	// UpdateStatus persists the status and lifecycle fields of req, scoped to tenantID.
	UpdateStatus(ctx context.Context, tenantID uuid.UUID, req *models.VendorAccessRequest) error
}
