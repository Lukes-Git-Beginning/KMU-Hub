package helpdesk

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the unified persistence interface for the helpdesk domain.
// All DB-access goes through this interface; handlers and services never touch
// the DB layer directly.
type Repository interface {
	// -----------------------------------------------------------------------
	// Tickets
	// -----------------------------------------------------------------------

	CreateTicket(ctx context.Context, t *Ticket) error
	GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error)
	ListTickets(ctx context.Context, tenantID uuid.UUID, statusFilter *string, page, pageSize int) ([]*Ticket, int, error)
	UpdateTicket(ctx context.Context, t *Ticket) error
	DeleteTicket(ctx context.Context, id uuid.UUID) error

	// FindOpenTicketsByRequester returns open/pending tickets for the given
	// requester that have a subject starting with the given prefix.
	// Used by merge duplicate detection.
	FindOpenTicketsByRequester(ctx context.Context, tenantID, requesterID uuid.UUID, subjectPrefix string) ([]*Ticket, error)

	// -----------------------------------------------------------------------
	// Ticket messages
	// -----------------------------------------------------------------------

	CreateMessage(ctx context.Context, m *TicketMessage) error
	ListMessagesByTicket(ctx context.Context, ticketID uuid.UUID) ([]*TicketMessage, error)
	// ReassignMessages moves all messages from sourceTicketID to targetTicketID.
	ReassignMessages(ctx context.Context, sourceTicketID, targetTicketID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Queues
	// -----------------------------------------------------------------------

	CreateQueue(ctx context.Context, q *TicketQueue) error
	GetQueueByID(ctx context.Context, id uuid.UUID) (*TicketQueue, error)
	ListQueues(ctx context.Context, tenantID uuid.UUID) ([]*TicketQueue, error)
	UpdateQueue(ctx context.Context, q *TicketQueue) error
	DeleteQueue(ctx context.Context, id uuid.UUID) error

	// -----------------------------------------------------------------------
	// Canned responses
	// -----------------------------------------------------------------------

	CreateCannedResponse(ctx context.Context, cr *CannedResponse) error
	GetCannedResponseByID(ctx context.Context, id uuid.UUID) (*CannedResponse, error)
	ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error)
	UpdateCannedResponse(ctx context.Context, cr *CannedResponse) error
	DeleteCannedResponse(ctx context.Context, id uuid.UUID) error

	// -----------------------------------------------------------------------
	// SLA policies
	// -----------------------------------------------------------------------

	CreateSLAPolicy(ctx context.Context, p *SLAPolicy) error
	GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*SLAPolicy, error)
	ListSLAPolicies(ctx context.Context, tenantID uuid.UUID) ([]*SLAPolicy, error)
	UpdateSLAPolicy(ctx context.Context, p *SLAPolicy) error
}
