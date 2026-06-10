package gobdarchive

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for the GoBD archive.
// IMPORTANT: No Update or Delete methods — GoBD documents are immutable.
type Repository interface {
	// Create persists a new immutable GoBD document record.
	Create(ctx context.Context, doc *models.GobdDocument) error

	// GetByID retrieves a document by tenant + document ID.
	// Returns ErrDocumentNotFound if not accessible.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.GobdDocument, error)

	// List returns paginated documents matching the filter.
	List(ctx context.Context, filter ListFilter) ([]*models.GobdDocument, int, error)

	// AppendEvent appends an audit trail event to a document.
	AppendEvent(ctx context.Context, event *models.GobdDocumentEvent) error

	// ListEvents returns all events for a document ordered by created_at DESC.
	ListEvents(ctx context.Context, tenantID, documentID uuid.UUID) ([]*models.GobdDocumentEvent, error)
}

// ListFilter contains options for listing GoBD documents.
type ListFilter struct {
	TenantID        uuid.UUID
	DocType         string     // Empty = all types
	SourceInvoiceID *uuid.UUID // Nil = all
	DateFrom        *time.Time // Nil = no lower bound on archived_at
	DateTo          *time.Time // Nil = no upper bound on archived_at
	Page            int        // 1-based
	PerPage         int        // Default 50
}
