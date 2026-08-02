package helpdesk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service is the thick service layer for the Cosmi Helpdesk module. All
// business logic lives here; handlers only parse requests and format responses.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService creates a Service wired to the given Repository and logger.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ---------------------------------------------------------------------------
// Ticket lifecycle
// ---------------------------------------------------------------------------

// CreateTicket creates a new open ticket.
func (s *Service) CreateTicket(
	ctx context.Context,
	tenantID uuid.UUID,
	requesterID uuid.UUID,
	subject string,
	priority string,
	assigneeID *uuid.UUID,
	queueID *uuid.UUID,
	description string,
	category string,
	contactID *uuid.UUID,
	orgID *uuid.UUID,
) (*Ticket, error) {
	if subject == "" {
		return nil, fmt.Errorf("subject must not be empty")
	}
	if priority == "" {
		priority = TicketPriorityNormal
	}
	if !ValidTicketPriorities[priority] {
		return nil, ErrInvalidPriority
	}
	if err := s.checkContactOrgTenant(ctx, tenantID, contactID, orgID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	t := &Ticket{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Subject:     subject,
		Status:      TicketStatusOpen,
		Priority:    priority,
		AssigneeID:  assigneeID,
		RequesterID: requesterID,
		QueueID:     queueID,
		Description: description,
		Category:    category,
		ContactID:   contactID,
		OrgID:       orgID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTicket(ctx, t); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket created",
		"ticket_id", t.ID,
		"tenant_id", tenantID,
		"requester_id", requesterID,
	)

	return t, nil
}

// sourceChannelLabels renders a ValidSourceChannels key for the fallback
// ticket subject CreateTicketFromMessage uses when the source message itself
// has no subject.
var sourceChannelLabels = map[string]string{
	"email":        "E-Mail",
	"chat":         "Chat",
	"notification": "Benachrichtigung",
}

// CreateTicketFromMessage converts an inbox message into a ticket: the ticket
// links back to the message via sourceMessageID/sourceChannel instead of
// copying its content a second time, so ticket and inbox never drift apart.
// Converting the same messageID twice is a no-op: the second call returns the
// already-existing ticket (created=false) instead of creating a duplicate.
// channel/subject/preview are supplied by the caller (the gRPC handler fetches
// them from the inbox service) -- this method holds no cross-service client.
func (s *Service) CreateTicketFromMessage(
	ctx context.Context,
	tenantID uuid.UUID,
	requesterID uuid.UUID,
	messageID uuid.UUID,
	channel string,
	subject string,
	preview string,
) (*Ticket, bool, error) {
	if !ValidSourceChannels[channel] {
		return nil, false, ErrInvalidSourceChannel
	}

	if existing, err := s.repo.GetTicketBySourceMessage(ctx, tenantID, messageID); err != nil {
		return nil, false, fmt.Errorf("check existing ticket for message: %w", err)
	} else if existing != nil {
		return existing, false, nil
	}

	if subject == "" {
		subject = fmt.Sprintf("Ticket aus %s", sourceChannelLabels[channel])
	}

	now := time.Now().UTC()
	t := &Ticket{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Subject:         subject,
		Status:          TicketStatusOpen,
		Priority:        TicketPriorityNormal,
		RequesterID:     requesterID,
		Description:     preview,
		SourceChannel:   &channel,
		SourceMessageID: &messageID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateTicket(ctx, t); err != nil {
		if errors.Is(err, ErrMessageAlreadyLinked) {
			// Lost the race against a concurrent conversion of the same
			// message: the unique index caught it, so load and return the
			// winner instead of surfacing an error for a non-error situation.
			existing, existErr := s.repo.GetTicketBySourceMessage(ctx, tenantID, messageID)
			if existErr != nil {
				return nil, false, fmt.Errorf("load ticket after concurrent conversion: %w", existErr)
			}
			if existing != nil {
				return existing, false, nil
			}
		}
		return nil, false, fmt.Errorf("create ticket from message: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket created from inbox message",
		"ticket_id", t.ID,
		"tenant_id", tenantID,
		"message_id", messageID,
		"channel", channel,
	)

	return t, true, nil
}

// GetTicket retrieves a ticket by ID.
func (s *Service) GetTicket(ctx context.Context, id, tenantID uuid.UUID) (*Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	return t, nil
}

// ListTickets returns a paginated list of tickets for a tenant.
// ListTickets returns one page of the tenant's tickets. A non-nil participantID
// narrows the page to tickets that user raised or is assigned — what a caller
// whose helpdesk:ticket:read grant is scoped to "own" may see.
func (s *Service) ListTickets(
	ctx context.Context,
	tenantID uuid.UUID,
	statusFilter *string,
	participantID, contactID, orgID *uuid.UUID,
	page, pageSize int,
) ([]*Ticket, int, error) {
	if statusFilter != nil && !ValidTicketStatuses[*statusFilter] {
		return nil, 0, ErrInvalidStatus
	}
	tickets, total, err := s.repo.ListTickets(ctx, tenantID, statusFilter, participantID, contactID, orgID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	return tickets, total, nil
}

// checkContactOrgTenant verifies that a non-nil contactID/orgID belongs to
// tenantID before it is allowed onto a ticket -- otherwise a ticket could be
// linked to another tenant's CRM data.
func (s *Service) checkContactOrgTenant(ctx context.Context, tenantID uuid.UUID, contactID, orgID *uuid.UUID) error {
	if contactID != nil {
		exists, err := s.repo.ContactExists(ctx, *contactID, tenantID)
		if err != nil {
			return fmt.Errorf("check contact tenant: %w", err)
		}
		if !exists {
			return ErrContactNotFound
		}
	}
	if orgID != nil {
		exists, err := s.repo.CompanyExists(ctx, *orgID, tenantID)
		if err != nil {
			return fmt.Errorf("check org tenant: %w", err)
		}
		if !exists {
			return ErrOrgNotFound
		}
	}
	return nil
}

// UpdateTicket applies field-level patches to a ticket.
func (s *Service) UpdateTicket(
	ctx context.Context,
	id, tenantID uuid.UUID,
	subject *string,
	priority *string,
	assigneeID *uuid.UUID,
	queueID *uuid.UUID,
	contactID *uuid.UUID,
	orgID *uuid.UUID,
) (*Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update ticket – load: %w", err)
	}
	if err := s.checkContactOrgTenant(ctx, tenantID, contactID, orgID); err != nil {
		return nil, err
	}

	if subject != nil {
		if *subject == "" {
			return nil, fmt.Errorf("subject must not be empty")
		}
		t.Subject = *subject
	}
	if priority != nil {
		if !ValidTicketPriorities[*priority] {
			return nil, ErrInvalidPriority
		}
		t.Priority = *priority
	}
	if assigneeID != nil {
		t.AssigneeID = assigneeID
	}
	if queueID != nil {
		t.QueueID = queueID
	}
	if contactID != nil {
		t.ContactID = contactID
	}
	if orgID != nil {
		t.OrgID = orgID
	}
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateTicket(ctx, t); err != nil {
		return nil, fmt.Errorf("update ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket updated", "ticket_id", id)
	return t, nil
}

// CloseTicket transitions a ticket to closed status.
func (s *Service) CloseTicket(ctx context.Context, id, tenantID uuid.UUID) (*Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("close ticket – load: %w", err)
	}

	now := time.Now().UTC()
	t.Status = TicketStatusClosed
	t.ResolvedAt = &now
	t.UpdatedAt = now

	if err := s.repo.UpdateTicket(ctx, t); err != nil {
		return nil, fmt.Errorf("close ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket closed", "ticket_id", id)
	return t, nil
}

// ReopenTicket transitions a closed or solved ticket back to open.
func (s *Service) ReopenTicket(ctx context.Context, id, tenantID uuid.UUID) (*Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reopen ticket – load: %w", err)
	}

	if t.Status != TicketStatusClosed && t.Status != TicketStatusSolved {
		return nil, fmt.Errorf("reopen ticket: only closed or solved tickets can be reopened")
	}

	t.Status = TicketStatusOpen
	t.ResolvedAt = nil
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateTicket(ctx, t); err != nil {
		return nil, fmt.Errorf("reopen ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket reopened", "ticket_id", id)
	return t, nil
}

// AssignTicket sets the assignee on a ticket.
func (s *Service) AssignTicket(ctx context.Context, ticketID, tenantID, assigneeID uuid.UUID) (*Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("assign ticket – load: %w", err)
	}

	t.AssigneeID = &assigneeID
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateTicket(ctx, t); err != nil {
		return nil, fmt.Errorf("assign ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: ticket assigned",
		"ticket_id", ticketID,
		"assignee_id", assigneeID,
	)
	return t, nil
}

// MergeTickets merges sourceID into targetID via the merge package logic.
func (s *Service) MergeTickets(ctx context.Context, tenantID, sourceID, targetID uuid.UUID) error {
	if err := MergeTickets(ctx, s.repo, tenantID, sourceID, targetID); err != nil {
		return fmt.Errorf("merge tickets: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: tickets merged",
		"source_id", sourceID,
		"target_id", targetID,
	)
	return nil
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// AddMessage appends a reply or internal note to a ticket. Sets
// first_response_at on the ticket if this is the first non-internal message.
func (s *Service) AddMessage(
	ctx context.Context,
	ticketID, tenantID uuid.UUID,
	authorID uuid.UUID,
	body string,
	internal bool,
	attachments []string,
) (*TicketMessage, error) {
	if body == "" {
		return nil, fmt.Errorf("message body must not be empty")
	}

	ticket, err := s.repo.GetTicketByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("add message – load ticket: %w", err)
	}

	now := time.Now().UTC()
	if attachments == nil {
		attachments = []string{}
	}

	m := &TicketMessage{
		ID:          uuid.New(),
		TenantID:    ticket.TenantID,
		TicketID:    ticketID,
		AuthorID:    authorID,
		Body:        body,
		Internal:    internal,
		Attachments: attachments,
		CreatedAt:   now,
	}

	if err := s.repo.CreateMessage(ctx, m); err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}

	// Mark first response timestamp on the ticket if this is the first public reply.
	if !internal && ticket.FirstResponseAt == nil {
		ticket.FirstResponseAt = &now
		ticket.UpdatedAt = now
		if err := s.repo.UpdateTicket(ctx, ticket); err != nil {
			s.log.WarnContext(ctx, "helpdesk: set first_response_at failed",
				"ticket_id", ticketID,
				"error", err,
			)
		}
	}

	s.log.InfoContext(ctx, "helpdesk: message added",
		"ticket_id", ticketID,
		"message_id", m.ID,
		"internal", internal,
	)

	return m, nil
}

// ListMessages returns all messages for a ticket ordered by created_at ASC.
func (s *Service) ListMessages(ctx context.Context, ticketID, tenantID uuid.UUID) ([]*TicketMessage, error) {
	messages, err := s.repo.ListMessagesByTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return messages, nil
}

// ---------------------------------------------------------------------------
// Queue management
// ---------------------------------------------------------------------------

// CreateQueue creates a new ticket queue for a tenant.
func (s *Service) CreateQueue(
	ctx context.Context,
	tenantID uuid.UUID,
	name string,
	defaultAssigneeID *uuid.UUID,
	slaPolicyID *uuid.UUID,
) (*TicketQueue, error) {
	if name == "" {
		return nil, fmt.Errorf("queue name must not be empty")
	}

	now := time.Now().UTC()
	q := &TicketQueue{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Name:              name,
		DefaultAssigneeID: defaultAssigneeID,
		SLAPolicyID:       slaPolicyID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateQueue(ctx, q); err != nil {
		return nil, fmt.Errorf("create queue: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: queue created",
		"queue_id", q.ID,
		"tenant_id", tenantID,
		"name", name,
	)

	return q, nil
}

// UpdateQueue applies field patches to a queue.
func (s *Service) UpdateQueue(
	ctx context.Context,
	id, tenantID uuid.UUID,
	name *string,
	defaultAssigneeID *uuid.UUID,
	slaPolicyID *uuid.UUID,
) (*TicketQueue, error) {
	q, err := s.repo.GetQueueByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update queue – load: %w", err)
	}

	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("queue name must not be empty")
		}
		q.Name = *name
	}
	if defaultAssigneeID != nil {
		q.DefaultAssigneeID = defaultAssigneeID
	}
	if slaPolicyID != nil {
		q.SLAPolicyID = slaPolicyID
	}
	q.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateQueue(ctx, q); err != nil {
		return nil, fmt.Errorf("update queue: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: queue updated", "queue_id", id)
	return q, nil
}

// ListQueues returns all queues for a tenant.
func (s *Service) ListQueues(ctx context.Context, tenantID uuid.UUID) ([]*TicketQueue, error) {
	queues, err := s.repo.ListQueues(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}
	return queues, nil
}

// DeleteQueue removes a queue.
func (s *Service) DeleteQueue(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteQueue(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete queue: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: queue deleted", "queue_id", id)
	return nil
}

// ---------------------------------------------------------------------------
// Canned responses
// ---------------------------------------------------------------------------

// CreateCannedResponse creates a new reply template.
func (s *Service) CreateCannedResponse(
	ctx context.Context,
	tenantID uuid.UUID,
	name, body string,
) (*CannedResponse, error) {
	if name == "" {
		return nil, fmt.Errorf("canned response name must not be empty")
	}
	if body == "" {
		return nil, fmt.Errorf("canned response body must not be empty")
	}

	now := time.Now().UTC()
	cr := &CannedResponse{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateCannedResponse(ctx, cr); err != nil {
		return nil, fmt.Errorf("create canned response: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: canned response created",
		"canned_response_id", cr.ID,
		"tenant_id", tenantID,
	)

	return cr, nil
}

// UpdateCannedResponse patches name/body of a canned response.
func (s *Service) UpdateCannedResponse(
	ctx context.Context,
	id, tenantID uuid.UUID,
	name *string,
	body *string,
) (*CannedResponse, error) {
	cr, err := s.repo.GetCannedResponseByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update canned response – load: %w", err)
	}

	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("canned response name must not be empty")
		}
		cr.Name = *name
	}
	if body != nil {
		if *body == "" {
			return nil, fmt.Errorf("canned response body must not be empty")
		}
		cr.Body = *body
	}
	cr.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateCannedResponse(ctx, cr); err != nil {
		return nil, fmt.Errorf("update canned response: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: canned response updated", "canned_response_id", id)
	return cr, nil
}

// DeleteCannedResponse removes a canned response template.
func (s *Service) DeleteCannedResponse(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteCannedResponse(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete canned response: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: canned response deleted", "canned_response_id", id)
	return nil
}

// ListCannedResponses returns all canned responses for a tenant.
func (s *Service) ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error) {
	list, err := s.repo.ListCannedResponses(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list canned responses: %w", err)
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// SLA policies
// ---------------------------------------------------------------------------

// CreateSLAPolicy creates a new SLA policy for a tenant.
func (s *Service) CreateSLAPolicy(
	ctx context.Context,
	tenantID uuid.UUID,
	name string,
	firstResponseMins, resolutionMins int,
	businessHours map[string]any,
) (*SLAPolicy, error) {
	if name == "" {
		return nil, fmt.Errorf("sla policy name must not be empty")
	}
	if firstResponseMins <= 0 {
		return nil, fmt.Errorf("first_response_mins must be positive")
	}
	if resolutionMins <= 0 {
		return nil, fmt.Errorf("resolution_mins must be positive")
	}

	now := time.Now().UTC()
	p := &SLAPolicy{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Name:              name,
		FirstResponseMins: firstResponseMins,
		ResolutionMins:    resolutionMins,
		BusinessHours:     businessHours,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateSLAPolicy(ctx, p); err != nil {
		return nil, fmt.Errorf("create sla policy: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: sla policy created",
		"policy_id", p.ID,
		"tenant_id", tenantID,
		"name", name,
	)

	return p, nil
}

// UpdateSLAPolicy applies patches to an existing SLA policy.
func (s *Service) UpdateSLAPolicy(
	ctx context.Context,
	id, tenantID uuid.UUID,
	name *string,
	firstResponseMins *int,
	resolutionMins *int,
	businessHours map[string]any,
) (*SLAPolicy, error) {
	p, err := s.repo.GetSLAPolicyByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update sla policy – load: %w", err)
	}

	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("sla policy name must not be empty")
		}
		p.Name = *name
	}
	if firstResponseMins != nil {
		if *firstResponseMins <= 0 {
			return nil, fmt.Errorf("first_response_mins must be positive")
		}
		p.FirstResponseMins = *firstResponseMins
	}
	if resolutionMins != nil {
		if *resolutionMins <= 0 {
			return nil, fmt.Errorf("resolution_mins must be positive")
		}
		p.ResolutionMins = *resolutionMins
	}
	if businessHours != nil {
		p.BusinessHours = businessHours
	}
	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateSLAPolicy(ctx, p); err != nil {
		return nil, fmt.Errorf("update sla policy: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: sla policy updated", "policy_id", id)
	return p, nil
}

// ListSLAPolicies returns all SLA policies for a tenant.
func (s *Service) ListSLAPolicies(ctx context.Context, tenantID uuid.UUID) ([]*SLAPolicy, error) {
	policies, err := s.repo.ListSLAPolicies(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list sla policies: %w", err)
	}
	return policies, nil
}

// ApplySLAPolicy assigns a policy to a ticket and recalculates due_at.
func (s *Service) ApplySLAPolicy(ctx context.Context, ticketID, tenantID, policyID uuid.UUID) (*Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("apply sla policy – load ticket: %w", err)
	}

	policy, err := s.repo.GetSLAPolicyByID(ctx, policyID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("apply sla policy – load policy: %w", err)
	}

	now := time.Now().UTC()
	ticket = ApplyPolicy(ticket, policy, now)
	ticket.UpdatedAt = now

	if err := s.repo.UpdateTicket(ctx, ticket); err != nil {
		return nil, fmt.Errorf("apply sla policy – update ticket: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: sla policy applied",
		"ticket_id", ticketID,
		"policy_id", policyID,
		"due_at", ticket.DueAt,
	)

	return ticket, nil
}

// DeleteSLAPolicy removes an SLA policy.
func (s *Service) DeleteSLAPolicy(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteSLAPolicy(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete sla policy: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: sla policy deleted", "policy_id", id)
	return nil
}

// GetSLAStatus computes the current SLA health of a ticket.
func (s *Service) GetSLAStatus(ctx context.Context, ticketID, tenantID uuid.UUID, policyID *uuid.UUID) (SLAStatus, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID, tenantID)
	if err != nil {
		return "", fmt.Errorf("get sla status – load ticket: %w", err)
	}

	var policy *SLAPolicy
	if policyID != nil {
		policy, err = s.repo.GetSLAPolicyByID(ctx, *policyID, tenantID)
		if err != nil {
			return "", fmt.Errorf("get sla status – load policy: %w", err)
		}
	}

	return ComputeStatus(ticket, policy, time.Now().UTC()), nil
}

// ---------------------------------------------------------------------------
// Knowledge-base articles
// ---------------------------------------------------------------------------

// CreateKBArticle creates a new knowledge-base article.
func (s *Service) CreateKBArticle(
	ctx context.Context,
	tenantID uuid.UUID,
	authorID uuid.UUID,
	title, content, category, articleStatus string,
) (*KBArticle, error) {
	if title == "" {
		return nil, fmt.Errorf("kb article title must not be empty")
	}
	if articleStatus == "" {
		articleStatus = string(KBArticleStatusDraft)
	}

	now := time.Now().UTC()
	a := &KBArticle{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Title:     title,
		Content:   content,
		Category:  category,
		Status:    articleStatus,
		AuthorID:  authorID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateKBArticle(ctx, a); err != nil {
		return nil, fmt.Errorf("create kb article: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: kb article created",
		"article_id", a.ID,
		"tenant_id", tenantID,
	)
	return a, nil
}

// GetKBArticle retrieves a KB article by ID.
func (s *Service) GetKBArticle(ctx context.Context, id, tenantID uuid.UUID) (*KBArticle, error) {
	a, err := s.repo.GetKBArticleByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get kb article: %w", err)
	}
	return a, nil
}

// ListKBArticles returns all KB articles for a tenant.
func (s *Service) ListKBArticles(ctx context.Context, tenantID uuid.UUID) ([]*KBArticle, error) {
	articles, err := s.repo.ListKBArticles(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list kb articles: %w", err)
	}
	return articles, nil
}

// UpdateKBArticle patches a KB article.
func (s *Service) UpdateKBArticle(
	ctx context.Context,
	id, tenantID uuid.UUID,
	title, content, category, articleStatus *string,
) (*KBArticle, error) {
	a, err := s.repo.GetKBArticleByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update kb article – load: %w", err)
	}

	if title != nil {
		if *title == "" {
			return nil, fmt.Errorf("kb article title must not be empty")
		}
		a.Title = *title
	}
	if content != nil {
		a.Content = *content
	}
	if category != nil {
		a.Category = *category
	}
	if articleStatus != nil {
		a.Status = *articleStatus
	}
	a.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateKBArticle(ctx, a); err != nil {
		return nil, fmt.Errorf("update kb article: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: kb article updated", "article_id", id)
	return a, nil
}

// DeleteKBArticle removes a KB article.
func (s *Service) DeleteKBArticle(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteKBArticle(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete kb article: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: kb article deleted", "article_id", id)
	return nil
}

// ---------------------------------------------------------------------------
// Routing rules
// ---------------------------------------------------------------------------

// CreateRoutingRule creates a new ticket routing rule.
func (s *Service) CreateRoutingRule(
	ctx context.Context,
	tenantID uuid.UUID,
	name string,
	conditions map[string]any,
	targetQueueID *uuid.UUID,
	priority int,
	enabled bool,
) (*RoutingRule, error) {
	if name == "" {
		return nil, fmt.Errorf("routing rule name must not be empty")
	}
	if conditions == nil {
		conditions = map[string]any{}
	}

	now := time.Now().UTC()
	rr := &RoutingRule{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          name,
		Conditions:    conditions,
		TargetQueueID: targetQueueID,
		Priority:      priority,
		Enabled:       enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateRoutingRule(ctx, rr); err != nil {
		return nil, fmt.Errorf("create routing rule: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: routing rule created",
		"rule_id", rr.ID,
		"tenant_id", tenantID,
	)
	return rr, nil
}

// ListRoutingRules returns all routing rules for a tenant.
func (s *Service) ListRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*RoutingRule, error) {
	rules, err := s.repo.ListRoutingRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list routing rules: %w", err)
	}
	return rules, nil
}

// UpdateRoutingRule patches a routing rule.
func (s *Service) UpdateRoutingRule(
	ctx context.Context,
	id, tenantID uuid.UUID,
	name *string,
	conditions map[string]any,
	targetQueueID *uuid.UUID,
	priority *int,
	enabled *bool,
) (*RoutingRule, error) {
	rr, err := s.repo.GetRoutingRuleByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update routing rule – load: %w", err)
	}

	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("routing rule name must not be empty")
		}
		rr.Name = *name
	}
	if conditions != nil {
		rr.Conditions = conditions
	}
	if targetQueueID != nil {
		rr.TargetQueueID = targetQueueID
	}
	if priority != nil {
		rr.Priority = *priority
	}
	if enabled != nil {
		rr.Enabled = *enabled
	}
	rr.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateRoutingRule(ctx, rr); err != nil {
		return nil, fmt.Errorf("update routing rule: %w", err)
	}

	s.log.InfoContext(ctx, "helpdesk: routing rule updated", "rule_id", id)
	return rr, nil
}

// DeleteRoutingRule removes a routing rule.
func (s *Service) DeleteRoutingRule(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.DeleteRoutingRule(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete routing rule: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: routing rule deleted", "rule_id", id)
	return nil
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

// GetHelpdeskStats returns aggregated helpdesk metrics for a tenant.
func (s *Service) GetHelpdeskStats(ctx context.Context, tenantID uuid.UUID) (*HelpdeskStats, error) {
	stats, err := s.repo.GetHelpdeskStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get helpdesk stats: %w", err)
	}
	return stats, nil
}

// ---------------------------------------------------------------------------
// Business Hours
// ---------------------------------------------------------------------------

// GetBusinessHours returns the business-hours config for a tenant.
// When no config exists yet, defaults are returned (no error).
func (s *Service) GetBusinessHours(ctx context.Context, tenantID uuid.UUID) (*BusinessHours, error) {
	bh, err := s.repo.GetBusinessHours(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get business hours: %w", err)
	}
	return bh, nil
}

// UpsertBusinessHours persists the business-hours config for a tenant.
func (s *Service) UpsertBusinessHours(ctx context.Context, bh *BusinessHours) (*BusinessHours, error) {
	if bh.Timezone == "" {
		bh.Timezone = "Europe/Berlin"
	}
	if bh.Schedule == nil {
		bh.Schedule = map[string]DaySchedule{}
	}
	if bh.Holidays == nil {
		bh.Holidays = []HolidayEntry{}
	}
	if err := s.repo.UpsertBusinessHours(ctx, bh); err != nil {
		return nil, fmt.Errorf("upsert business hours: %w", err)
	}
	// Re-read to get the server-set updated_at timestamp.
	updated, err := s.repo.GetBusinessHours(ctx, bh.TenantID)
	if err != nil {
		return nil, fmt.Errorf("re-read business hours after upsert: %w", err)
	}
	s.log.InfoContext(ctx, "helpdesk: business hours updated", "tenant_id", bh.TenantID)
	return updated, nil
}
