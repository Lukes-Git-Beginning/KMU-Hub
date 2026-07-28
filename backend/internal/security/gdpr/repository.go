package gdpr

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for GDPR export and erasure operations.
type Repository interface {
	// CreateExportRequest inserts a new GDPR data export request.
	CreateExportRequest(ctx context.Context, req *models.GDPRExportRequest) error

	// GetExportRequest retrieves an export request by ID, scoped to tenantID.
	GetExportRequest(ctx context.Context, tenantID, id uuid.UUID) (*models.GDPRExportRequest, error)

	// ListExportRequests returns export requests for tenantID, filtered by user and/or status.
	// If userID is uuid.Nil, returns all requests for the tenant (admin view).
	// If status is empty, returns all statuses.
	ListExportRequests(ctx context.Context, tenantID, userID uuid.UUID, status string) ([]*models.GDPRExportRequest, error)

	// UpdateExportStatus updates the status and review fields of an export request, scoped to tenantID.
	UpdateExportStatus(ctx context.Context, tenantID uuid.UUID, req *models.GDPRExportRequest) error

	// StoreExportResult saves the generated export data, download token, and expiration, scoped to tenantID.
	StoreExportResult(ctx context.Context, tenantID, id uuid.UUID, data []byte, token string, expiresAt interface{}) error

	// GetExportByToken retrieves an export request by its download token, scoped to tenantID.
	GetExportByToken(ctx context.Context, tenantID uuid.UUID, token string) (*models.GDPRExportRequest, error)

	// MarkDownloaded sets the downloaded_at timestamp on an export request, scoped to tenantID.
	MarkDownloaded(ctx context.Context, tenantID, id uuid.UUID) error

	// CreateErasureLog inserts a new GDPR erasure log entry.
	CreateErasureLog(ctx context.Context, log *models.GDPRErasureLog) error

	// GetNextAnonymizedLabel returns the next sequential anonymized label.
	// Format: "Geloeschter Benutzer #NNN" where NNN = count of existing erasure entries + 1.
	GetNextAnonymizedLabel(ctx context.Context) (string, error)
}
