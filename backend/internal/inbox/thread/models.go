// Package thread provides persistence for inbox conversation threads (the
// materialized inbound original plus outbound replies and internal notes) and
// tenant-scoped canned responses for the unified inbox.
package thread

import (
	"time"

	"github.com/google/uuid"
)

// Thread message direction values.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
	DirectionInternal = "internal"
)

// ThreadMessage is one entry in an inbox conversation thread.
type ThreadMessage struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	MessageID uuid.UUID
	// AuthorID is nil for inbound/external messages (AuthorName holds the sender).
	AuthorID   *uuid.UUID
	AuthorName string
	Direction  string
	Body       string
	CreatedAt  time.Time
}

// InboundSeed carries the data used to materialize the first (inbound) thread
// row from the originating inbox message when the thread is still empty.
type InboundSeed struct {
	SenderName string
	Body       string
	CreatedAt  time.Time
}

// CannedResponse is a tenant-scoped reply template.
type CannedResponse struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
