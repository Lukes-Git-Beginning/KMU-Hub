package thread

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository persists inbox thread messages and canned responses.
type Repository interface {
	// AppendThreadMessage inserts a single thread row (outbound reply / internal note).
	AppendThreadMessage(ctx context.Context, m *ThreadMessage) error

	// SeedInboundIfEmpty inserts the inbound original row only when the thread has
	// no rows yet (idempotent — safe under concurrent first reads).
	SeedInboundIfEmpty(ctx context.Context, tenantID, messageID uuid.UUID, senderName, body string, createdAt time.Time) error

	// ListThreadMessages returns the thread for a message ordered by created_at,
	// resolving author display names via users.
	ListThreadMessages(ctx context.Context, tenantID, messageID uuid.UUID) ([]*ThreadMessage, error)

	CreateCannedResponse(ctx context.Context, c *CannedResponse) error
	ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error)
	GetCannedResponse(ctx context.Context, tenantID, id uuid.UUID) (*CannedResponse, error)
	UpdateCannedResponse(ctx context.Context, c *CannedResponse) error
	DeleteCannedResponse(ctx context.Context, tenantID, id uuid.UUID) error
}
