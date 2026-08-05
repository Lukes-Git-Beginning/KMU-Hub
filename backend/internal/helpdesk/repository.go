package helpdesk

import (
	"context"
	"time"

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
	GetTicketByID(ctx context.Context, id, tenantID uuid.UUID) (*Ticket, error)
	ListTickets(ctx context.Context, tenantID uuid.UUID, statusFilter *string, participantID, contactID, orgID *uuid.UUID, page, pageSize int) ([]*Ticket, int, error)
	UpdateTicket(ctx context.Context, t *Ticket) error
	DeleteTicket(ctx context.Context, id, tenantID uuid.UUID) error

	// ContactExists reports whether the contact belongs to the given tenant.
	ContactExists(ctx context.Context, contactID, tenantID uuid.UUID) (bool, error)
	// CompanyExists reports whether the company (CRM "organization") belongs
	// to the given tenant.
	CompanyExists(ctx context.Context, companyID, tenantID uuid.UUID) (bool, error)

	// FindOpenTicketsByRequester returns open/pending tickets for the given
	// requester that have a subject starting with the given prefix.
	// Used by merge duplicate detection.
	FindOpenTicketsByRequester(ctx context.Context, tenantID, requesterID uuid.UUID, subjectPrefix string) ([]*Ticket, error)

	// GetTicketBySourceMessage returns the ticket already linked to messageID
	// for tenantID, or nil (no error) if none exists yet. Backs the
	// CreateTicketFromMessage idempotency pre-check.
	GetTicketBySourceMessage(ctx context.Context, tenantID, messageID uuid.UUID) (*Ticket, error)

	// MergeTicketTx atomically reassigns all messages from source to target and
	// marks source as merged (status='merged', merged_into_id=targetID) in a
	// single database transaction. The source ticket must have its Status,
	// MergedIntoID, and UpdatedAt fields set before calling this method.
	// Note: Partition/retention DDL (DROP TABLE) is unaffected and operates
	// outside of row-level transactions.
	MergeTicketTx(ctx context.Context, source *Ticket, targetID uuid.UUID) error

	// SubmitCsatTx upserts the CSAT response row for a ticket and mirrors
	// rating/comment onto the ticket itself in a single transaction, so the
	// denormalised ticket columns can never drift from the response row.
	// A pending survey token on an existing row is left untouched.
	// Returns ErrTicketNotFound if the ticket does not belong to tenantID.
	SubmitCsatTx(ctx context.Context, tenantID, ticketID uuid.UUID, rating int16, comment *string, submittedAt time.Time) error

	// IssueCsatSurveyTokenTx stores a pending survey token on the ticket's CSAT
	// response row, creating that row if the ticket has none yet. It reports
	// false (without error) when no token was stored, which happens for a
	// ticket of another tenant and for a ticket that already carries a rating --
	// both are ordinary outcomes, not failures.
	IssueCsatSurveyTokenTx(ctx context.Context, tenantID, ticketID uuid.UUID, token string, sendAfter, expiresAt, now time.Time) (bool, error)

	// The survey dispatch surface (list due / claim / release / cancel) is
	// deliberately NOT part of this interface: only the background dispatcher
	// uses it, and it declares its own narrow CsatDispatchRepository
	// (csat_dispatch.go) -- same split as berichte's ScheduleRepository.

	// -----------------------------------------------------------------------
	// Ticket messages
	// -----------------------------------------------------------------------

	CreateMessage(ctx context.Context, m *TicketMessage) error
	ListMessagesByTicket(ctx context.Context, ticketID, tenantID uuid.UUID) ([]*TicketMessage, error)
	// ReassignMessages moves all messages from sourceTicketID to targetTicketID.
	ReassignMessages(ctx context.Context, sourceTicketID, targetTicketID, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Queues
	// -----------------------------------------------------------------------

	CreateQueue(ctx context.Context, q *TicketQueue) error
	GetQueueByID(ctx context.Context, id, tenantID uuid.UUID) (*TicketQueue, error)
	ListQueues(ctx context.Context, tenantID uuid.UUID) ([]*TicketQueue, error)
	UpdateQueue(ctx context.Context, q *TicketQueue) error
	DeleteQueue(ctx context.Context, id, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Canned responses
	// -----------------------------------------------------------------------

	CreateCannedResponse(ctx context.Context, cr *CannedResponse) error
	GetCannedResponseByID(ctx context.Context, id, tenantID uuid.UUID) (*CannedResponse, error)
	ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error)
	UpdateCannedResponse(ctx context.Context, cr *CannedResponse) error
	DeleteCannedResponse(ctx context.Context, id, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// SLA policies
	// -----------------------------------------------------------------------

	CreateSLAPolicy(ctx context.Context, p *SLAPolicy) error
	GetSLAPolicyByID(ctx context.Context, id, tenantID uuid.UUID) (*SLAPolicy, error)
	ListSLAPolicies(ctx context.Context, tenantID uuid.UUID) ([]*SLAPolicy, error)
	UpdateSLAPolicy(ctx context.Context, p *SLAPolicy) error
	DeleteSLAPolicy(ctx context.Context, id, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Knowledge-base articles
	// -----------------------------------------------------------------------

	CreateKBArticle(ctx context.Context, a *KBArticle) error
	GetKBArticleByID(ctx context.Context, id, tenantID uuid.UUID) (*KBArticle, error)
	ListKBArticles(ctx context.Context, tenantID uuid.UUID) ([]*KBArticle, error)
	UpdateKBArticle(ctx context.Context, a *KBArticle) error
	DeleteKBArticle(ctx context.Context, id, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Routing rules
	// -----------------------------------------------------------------------

	CreateRoutingRule(ctx context.Context, rr *RoutingRule) error
	GetRoutingRuleByID(ctx context.Context, id, tenantID uuid.UUID) (*RoutingRule, error)
	ListRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rr *RoutingRule) error
	DeleteRoutingRule(ctx context.Context, id, tenantID uuid.UUID) error

	// -----------------------------------------------------------------------
	// Business hours (one row per tenant, upsert)
	// -----------------------------------------------------------------------

	// GetBusinessHours returns the persisted config for the tenant, or a
	// default BusinessHours struct (empty Schedule/Holidays, Europe/Berlin)
	// when no row exists yet.
	GetBusinessHours(ctx context.Context, tenantID uuid.UUID) (*BusinessHours, error)
	// UpsertBusinessHours inserts or updates the business-hours config for
	// the tenant (INSERT … ON CONFLICT (tenant_id) DO UPDATE).
	UpsertBusinessHours(ctx context.Context, bh *BusinessHours) error

	// -----------------------------------------------------------------------
	// Stats (aggregate query, no own table)
	// -----------------------------------------------------------------------

	GetHelpdeskStats(ctx context.Context, tenantID uuid.UUID) (*HelpdeskStats, error)
}
