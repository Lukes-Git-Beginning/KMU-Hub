package helpdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository backed by a pgxpool.Pool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a helpdesk Repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ---------------------------------------------------------------------------
// Tickets
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateTicket(ctx context.Context, t *Ticket) error {
	// Allocate the next per-tenant ticket number atomically. On first ticket for
	// a tenant the row is inserted with next_number=2 and 1 is returned; on every
	// subsequent ticket next_number is incremented and the prior value returned.
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO helpdesk_ticket_counters (tenant_id, next_number)
		 VALUES ($1, 2)
		 ON CONFLICT (tenant_id)
		   DO UPDATE SET next_number = helpdesk_ticket_counters.next_number + 1
		 RETURNING next_number - 1`,
		t.TenantID,
	).Scan(&t.TicketNumber); err != nil {
		return fmt.Errorf("allocate ticket number: %w", err)
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO tickets
		    (id, tenant_id, subject, status, priority, assignee_id, requester_id,
		     queue_id, due_at, merged_into_id, first_response_at, resolved_at,
		     description, category, ticket_number, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		t.ID, t.TenantID, t.Subject, t.Status, t.Priority,
		t.AssigneeID, t.RequesterID, t.QueueID, t.DueAt,
		t.MergedIntoID, t.FirstResponseAt, t.ResolvedAt,
		t.Description, t.Category, t.TicketNumber,
		t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ticketSelectColumns is the shared SELECT body for ticket reads. It denormalizes
// assignee/requester display names via LEFT JOINs on users (falling back to email,
// then empty) and COALESCEs nullable text columns. Every query feeding scanTicket /
// scanTicketFromRows MUST use this column list so the scan order stays aligned.
const ticketSelectColumns = `
	t.id, t.tenant_id, t.subject, t.status, t.priority, t.assignee_id, t.requester_id,
	t.queue_id, t.due_at, t.merged_into_id, t.first_response_at, t.resolved_at,
	t.created_at, t.updated_at,
	COALESCE(t.description, '') AS description,
	COALESCE(t.category, '')    AS category,
	t.ticket_number,
	COALESCE(NULLIF(CONCAT_WS(' ', a.first_name, a.last_name), ''), a.email)         AS assignee_name,
	COALESCE(NULLIF(CONCAT_WS(' ', req.first_name, req.last_name), ''), req.email, '') AS requester_name
FROM tickets t
LEFT JOIN users a   ON t.assignee_id  = a.id
LEFT JOIN users req ON t.requester_id = req.id`

func (r *PostgresRepository) GetTicketByID(ctx context.Context, id, tenantID uuid.UUID) (*Ticket, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+ticketSelectColumns+` WHERE t.id = $1 AND t.tenant_id = $2`, id, tenantID,
	)
	return scanTicket(row)
}

func (r *PostgresRepository) ListTickets(ctx context.Context, tenantID uuid.UUID, statusFilter *string, participantID *uuid.UUID, page, pageSize int) ([]*Ticket, int, error) {
	offset := (page - 1) * pageSize

	var (
		whereExtra string
		args       []any
	)
	args = append(args, tenantID)
	if statusFilter != nil {
		whereExtra += fmt.Sprintf(" AND t.status = $%d", len(args)+1)
		args = append(args, *statusFilter)
	}
	// The "own" data scope. Assignment counts as ownership too: an agent who
	// cannot see the ticket routed to them cannot work it.
	if participantID != nil {
		whereExtra += fmt.Sprintf(" AND (t.requester_id = $%d OR t.assignee_id = $%d)", len(args)+1, len(args)+1)
		args = append(args, *participantID)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tickets t WHERE t.tenant_id = $1%s", whereExtra)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := len(args) + 1
	offsetArg := limitArg + 1
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(
		`SELECT `+ticketSelectColumns+`
		 WHERE t.tenant_id = $1%s
		 ORDER BY t.created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereExtra, limitArg, offsetArg,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []*Ticket
	for rows.Next() {
		t, err := scanTicketFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, t)
	}
	return tickets, total, rows.Err()
}

func (r *PostgresRepository) UpdateTicket(ctx context.Context, t *Ticket) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tickets
		 SET subject = $1, status = $2, priority = $3, assignee_id = $4,
		     queue_id = $5, due_at = $6, merged_into_id = $7,
		     first_response_at = $8, resolved_at = $9, updated_at = $10
		 WHERE id = $11 AND tenant_id = $12`,
		t.Subject, t.Status, t.Priority, t.AssigneeID,
		t.QueueID, t.DueAt, t.MergedIntoID,
		t.FirstResponseAt, t.ResolvedAt, t.UpdatedAt,
		t.ID, t.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTicketNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteTicket(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tickets WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTicketNotFound
	}
	return nil
}

// FindOpenTicketsByRequester returns non-merged tickets for requester whose
// subject begins with subjectPrefix (case-insensitive ILIKE).
//
// Note: pg_trgm similarity search would give better fuzzy results but requires
// the pg_trgm extension. We use ILIKE-prefix as a safe default. If pg_trgm is
// available in production, replace the WHERE clause with:
//
//	similarity(subject, $3) > 0.4
func (r *PostgresRepository) FindOpenTicketsByRequester(ctx context.Context, tenantID, requesterID uuid.UUID, subjectPrefix string) ([]*Ticket, error) {
	prefix := strings.TrimSpace(subjectPrefix)
	if len(prefix) > 60 {
		prefix = prefix[:60]
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+ticketSelectColumns+`
		 WHERE t.tenant_id = $1
		   AND t.requester_id = $2
		   AND t.status NOT IN ('merged','closed','solved')
		   AND t.subject ILIKE $3
		 ORDER BY t.created_at DESC
		 LIMIT 20`,
		tenantID, requesterID, prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*Ticket
	for rows.Next() {
		t, err := scanTicketFromRows(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

// ---------------------------------------------------------------------------
// Ticket messages
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateMessage(ctx context.Context, m *TicketMessage) error {
	attachJSON, err := json.Marshal(m.Attachments)
	if err != nil {
		return fmt.Errorf("marshal attachments: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO ticket_messages
		    (id, tenant_id, ticket_id, author_id, body, internal, attachments, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m.ID, m.TenantID, m.TicketID, m.AuthorID, m.Body, m.Internal, attachJSON, m.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListMessagesByTicket(ctx context.Context, ticketID, tenantID uuid.UUID) ([]*TicketMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, ticket_id, author_id, body, internal, attachments, created_at
		 FROM ticket_messages
		 WHERE ticket_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC`,
		ticketID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*TicketMessage
	for rows.Next() {
		m, err := scanMessageFromRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *PostgresRepository) ReassignMessages(ctx context.Context, sourceTicketID, targetTicketID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ticket_messages SET ticket_id = $1 WHERE ticket_id = $2 AND tenant_id = $3`,
		targetTicketID, sourceTicketID, tenantID,
	)
	return err
}

// MergeTicketTx reassigns messages and closes source atomically in one
// transaction. source must already have Status, MergedIntoID, and UpdatedAt
// populated by the caller (merge.go sets these before invoking this method).
func (r *PostgresRepository) MergeTicketTx(ctx context.Context, source *Ticket, targetID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("merge tx begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// 1. Reassign messages from source → target.
	if _, err := tx.Exec(ctx,
		`UPDATE ticket_messages SET ticket_id = $1 WHERE ticket_id = $2 AND tenant_id = $3`,
		targetID, source.ID, source.TenantID,
	); err != nil {
		return fmt.Errorf("merge tx reassign messages: %w", err)
	}

	// 2. Close source ticket (mark as merged).
	tag, err := tx.Exec(ctx,
		`UPDATE tickets
		 SET subject = $1, status = $2, priority = $3, assignee_id = $4,
		     queue_id = $5, due_at = $6, merged_into_id = $7,
		     first_response_at = $8, resolved_at = $9, updated_at = $10
		 WHERE id = $11 AND tenant_id = $12`,
		source.Subject, source.Status, source.Priority, source.AssigneeID,
		source.QueueID, source.DueAt, source.MergedIntoID,
		source.FirstResponseAt, source.ResolvedAt, source.UpdatedAt,
		source.ID, source.TenantID,
	)
	if err != nil {
		return fmt.Errorf("merge tx update source ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTicketNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("merge tx commit: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Queues
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateQueue(ctx context.Context, q *TicketQueue) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ticket_queues
		    (id, tenant_id, name, default_assignee_id, sla_policy_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		q.ID, q.TenantID, q.Name, q.DefaultAssigneeID, q.SLAPolicyID, q.CreatedAt, q.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetQueueByID(ctx context.Context, id, tenantID uuid.UUID) (*TicketQueue, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, default_assignee_id, sla_policy_id, created_at, updated_at
		 FROM ticket_queues WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanQueue(row)
}

func (r *PostgresRepository) ListQueues(ctx context.Context, tenantID uuid.UUID) ([]*TicketQueue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, default_assignee_id, sla_policy_id, created_at, updated_at
		 FROM ticket_queues
		 WHERE tenant_id = $1
		 ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queues []*TicketQueue
	for rows.Next() {
		q, err := scanQueueFromRows(rows)
		if err != nil {
			return nil, err
		}
		queues = append(queues, q)
	}
	return queues, rows.Err()
}

func (r *PostgresRepository) UpdateQueue(ctx context.Context, q *TicketQueue) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE ticket_queues
		 SET name = $1, default_assignee_id = $2, sla_policy_id = $3, updated_at = $4
		 WHERE id = $5 AND tenant_id = $6`,
		q.Name, q.DefaultAssigneeID, q.SLAPolicyID, q.UpdatedAt, q.ID, q.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrQueueNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteQueue(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM ticket_queues WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrQueueNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Canned responses
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateCannedResponse(ctx context.Context, cr *CannedResponse) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO canned_responses
		    (id, tenant_id, name, body, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		cr.ID, cr.TenantID, cr.Name, cr.Body, cr.CreatedAt, cr.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetCannedResponseByID(ctx context.Context, id, tenantID uuid.UUID) (*CannedResponse, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, body, created_at, updated_at
		 FROM canned_responses WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanCannedResponse(row)
}

func (r *PostgresRepository) ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, body, created_at, updated_at
		 FROM canned_responses
		 WHERE tenant_id = $1
		 ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*CannedResponse
	for rows.Next() {
		cr, err := scanCannedResponseFromRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, cr)
	}
	return results, rows.Err()
}

func (r *PostgresRepository) UpdateCannedResponse(ctx context.Context, cr *CannedResponse) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE canned_responses
		 SET name = $1, body = $2, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5`,
		cr.Name, cr.Body, cr.UpdatedAt, cr.ID, cr.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCannedResponseNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteCannedResponse(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM canned_responses WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCannedResponseNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// SLA policies
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateSLAPolicy(ctx context.Context, p *SLAPolicy) error {
	bhJSON, err := json.Marshal(p.BusinessHours)
	if err != nil {
		return fmt.Errorf("marshal business_hours: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO sla_policies
		    (id, tenant_id, name, first_response_mins, resolution_mins,
		     business_hours, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.TenantID, p.Name, p.FirstResponseMins, p.ResolutionMins,
		bhJSON, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetSLAPolicyByID(ctx context.Context, id, tenantID uuid.UUID) (*SLAPolicy, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, first_response_mins, resolution_mins,
		        business_hours, created_at, updated_at
		 FROM sla_policies WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanSLAPolicy(row)
}

func (r *PostgresRepository) ListSLAPolicies(ctx context.Context, tenantID uuid.UUID) ([]*SLAPolicy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, first_response_mins, resolution_mins,
		        business_hours, created_at, updated_at
		 FROM sla_policies
		 WHERE tenant_id = $1
		 ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*SLAPolicy
	for rows.Next() {
		p, err := scanSLAPolicyFromRows(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *PostgresRepository) UpdateSLAPolicy(ctx context.Context, p *SLAPolicy) error {
	bhJSON, err := json.Marshal(p.BusinessHours)
	if err != nil {
		return fmt.Errorf("marshal business_hours: %w", err)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE sla_policies
		 SET name = $1, first_response_mins = $2, resolution_mins = $3,
		     business_hours = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		p.Name, p.FirstResponseMins, p.ResolutionMins, bhJSON, p.UpdatedAt, p.ID, p.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSLAPolicyNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteSLAPolicy(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sla_policies WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSLAPolicyNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Knowledge-base articles
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateKBArticle(ctx context.Context, a *KBArticle) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO helpdesk_kb_articles
		    (id, tenant_id, title, content, category, status, author_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.TenantID, a.Title, a.Content, a.Category, a.Status, a.AuthorID, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetKBArticleByID(ctx context.Context, id, tenantID uuid.UUID) (*KBArticle, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, content, category, status, author_id, created_at, updated_at
		 FROM helpdesk_kb_articles WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanKBArticle(row)
}

func (r *PostgresRepository) ListKBArticles(ctx context.Context, tenantID uuid.UUID) ([]*KBArticle, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, title, content, category, status, author_id, created_at, updated_at
		 FROM helpdesk_kb_articles
		 WHERE tenant_id = $1
		 ORDER BY updated_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []*KBArticle
	for rows.Next() {
		a, err := scanKBArticleFromRows(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

func (r *PostgresRepository) UpdateKBArticle(ctx context.Context, a *KBArticle) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE helpdesk_kb_articles
		 SET title = $1, content = $2, category = $3, status = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		a.Title, a.Content, a.Category, a.Status, a.UpdatedAt, a.ID, a.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrKBArticleNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteKBArticle(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM helpdesk_kb_articles WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrKBArticleNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Routing rules
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateRoutingRule(ctx context.Context, rr *RoutingRule) error {
	condJSON, err := json.Marshal(rr.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO helpdesk_routing_rules
		    (id, tenant_id, name, conditions, target_queue_id, priority, enabled, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		rr.ID, rr.TenantID, rr.Name, condJSON, rr.TargetQueueID, rr.Priority, rr.Enabled, rr.CreatedAt, rr.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetRoutingRuleByID(ctx context.Context, id, tenantID uuid.UUID) (*RoutingRule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, conditions, target_queue_id, priority, enabled, created_at, updated_at
		 FROM helpdesk_routing_rules WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanRoutingRule(row)
}

func (r *PostgresRepository) ListRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*RoutingRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, conditions, target_queue_id, priority, enabled, created_at, updated_at
		 FROM helpdesk_routing_rules
		 WHERE tenant_id = $1
		 ORDER BY priority ASC, created_at ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*RoutingRule
	for rows.Next() {
		rr, err := scanRoutingRuleFromRows(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rr)
	}
	return rules, rows.Err()
}

func (r *PostgresRepository) UpdateRoutingRule(ctx context.Context, rr *RoutingRule) error {
	condJSON, err := json.Marshal(rr.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE helpdesk_routing_rules
		 SET name = $1, conditions = $2, target_queue_id = $3, priority = $4, enabled = $5, updated_at = $6
		 WHERE id = $7 AND tenant_id = $8`,
		rr.Name, condJSON, rr.TargetQueueID, rr.Priority, rr.Enabled, rr.UpdatedAt, rr.ID, rr.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRoutingRuleNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteRoutingRule(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM helpdesk_routing_rules WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRoutingRuleNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func (r *PostgresRepository) GetHelpdeskStats(ctx context.Context, tenantID uuid.UUID) (*HelpdeskStats, error) {
	var stats HelpdeskStats

	// Open ticket count
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets WHERE tenant_id = $1 AND status NOT IN ('closed','solved','merged')`,
		tenantID,
	).Scan(&stats.OpenTickets); err != nil {
		return nil, fmt.Errorf("stats open tickets: %w", err)
	}

	// Resolved this week
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets
		 WHERE tenant_id = $1
		   AND status IN ('solved','closed')
		   AND resolved_at >= date_trunc('week', NOW())`,
		tenantID,
	).Scan(&stats.ResolvedThisWeek); err != nil {
		return nil, fmt.Errorf("stats resolved this week: %w", err)
	}

	// Average first response time (human-readable)
	var avgMinutes *float64
	if err := r.pool.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM AVG(first_response_at - created_at)) / 60
		 FROM tickets
		 WHERE tenant_id = $1 AND first_response_at IS NOT NULL`,
		tenantID,
	).Scan(&avgMinutes); err != nil {
		return nil, fmt.Errorf("stats avg response: %w", err)
	}
	if avgMinutes == nil || *avgMinutes == 0 {
		stats.AvgResponseTime = "–"
	} else if *avgMinutes < 60 {
		stats.AvgResponseTime = fmt.Sprintf("%.0f min", *avgMinutes)
	} else {
		stats.AvgResponseTime = fmt.Sprintf("%.1f h", *avgMinutes/60)
	}

	// Customer satisfaction placeholder (requires CSAT table, not yet in schema)
	stats.CustomerSatisfaction = "–"

	// Weekly breakdown: tickets created per day of current week
	rows, err := r.pool.Query(ctx,
		`SELECT to_char(created_at, 'Dy') AS day_label, COUNT(*) AS cnt
		 FROM tickets
		 WHERE tenant_id = $1
		   AND created_at >= date_trunc('week', NOW())
		 GROUP BY date_trunc('day', created_at), to_char(created_at, 'Dy')
		 ORDER BY date_trunc('day', created_at)`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("stats weekly breakdown: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d WeeklyDayCount
		if err := rows.Scan(&d.Label, &d.Count); err != nil {
			return nil, err
		}
		stats.WeeklyBreakdown = append(stats.WeeklyBreakdown, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ---------------------------------------------------------------------------
// Business Hours
// ---------------------------------------------------------------------------

func (r *PostgresRepository) GetBusinessHours(ctx context.Context, tenantID uuid.UUID) (*BusinessHours, error) {
	var (
		bh        BusinessHours
		schedJSON []byte
		holsJSON  []byte
	)
	err := r.pool.QueryRow(ctx,
		`SELECT tenant_id, schedule, holidays, timezone, updated_at
		 FROM helpdesk_business_hours
		 WHERE tenant_id = $1`,
		tenantID,
	).Scan(&bh.TenantID, &schedJSON, &holsJSON, &bh.Timezone, &bh.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Return sensible defaults when no config exists yet.
		return &BusinessHours{
			TenantID: tenantID,
			Schedule: map[string]DaySchedule{},
			Holidays: []HolidayEntry{},
			Timezone: "Europe/Berlin",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get business hours: %w", err)
	}
	if err := json.Unmarshal(schedJSON, &bh.Schedule); err != nil {
		return nil, fmt.Errorf("unmarshal schedule: %w", err)
	}
	if err := json.Unmarshal(holsJSON, &bh.Holidays); err != nil {
		return nil, fmt.Errorf("unmarshal holidays: %w", err)
	}
	return &bh, nil
}

func (r *PostgresRepository) UpsertBusinessHours(ctx context.Context, bh *BusinessHours) error {
	schedJSON, err := json.Marshal(bh.Schedule)
	if err != nil {
		return fmt.Errorf("marshal schedule: %w", err)
	}
	holsJSON, err := json.Marshal(bh.Holidays)
	if err != nil {
		return fmt.Errorf("marshal holidays: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO helpdesk_business_hours
		    (tenant_id, schedule, holidays, timezone, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE
		 SET schedule   = EXCLUDED.schedule,
		     holidays   = EXCLUDED.holidays,
		     timezone   = EXCLUDED.timezone,
		     updated_at = NOW()`,
		bh.TenantID, schedJSON, holsJSON, bh.Timezone,
	)
	return err
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type scannable interface {
	Scan(dest ...any) error
}

func scanTicket(row scannable) (*Ticket, error) {
	var t Ticket
	err := row.Scan(
		&t.ID, &t.TenantID, &t.Subject, &t.Status, &t.Priority,
		&t.AssigneeID, &t.RequesterID, &t.QueueID, &t.DueAt,
		&t.MergedIntoID, &t.FirstResponseAt, &t.ResolvedAt,
		&t.CreatedAt, &t.UpdatedAt,
		&t.Description, &t.Category, &t.TicketNumber,
		&t.AssigneeName, &t.RequesterName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTicketNotFound
	}
	return &t, err
}

func scanTicketFromRows(rows pgx.Rows) (*Ticket, error) {
	var t Ticket
	err := rows.Scan(
		&t.ID, &t.TenantID, &t.Subject, &t.Status, &t.Priority,
		&t.AssigneeID, &t.RequesterID, &t.QueueID, &t.DueAt,
		&t.MergedIntoID, &t.FirstResponseAt, &t.ResolvedAt,
		&t.CreatedAt, &t.UpdatedAt,
		&t.Description, &t.Category, &t.TicketNumber,
		&t.AssigneeName, &t.RequesterName,
	)
	return &t, err
}

func scanMessageFromRows(rows pgx.Rows) (*TicketMessage, error) {
	var (
		m          TicketMessage
		attachJSON []byte
	)
	err := rows.Scan(
		&m.ID, &m.TenantID, &m.TicketID, &m.AuthorID, &m.Body, &m.Internal, &attachJSON, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(attachJSON, &m.Attachments); err != nil {
		return nil, fmt.Errorf("unmarshal attachments: %w", err)
	}
	return &m, nil
}

func scanQueue(row scannable) (*TicketQueue, error) {
	var q TicketQueue
	err := row.Scan(
		&q.ID, &q.TenantID, &q.Name, &q.DefaultAssigneeID, &q.SLAPolicyID,
		&q.CreatedAt, &q.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQueueNotFound
	}
	return &q, err
}

func scanQueueFromRows(rows pgx.Rows) (*TicketQueue, error) {
	var q TicketQueue
	err := rows.Scan(
		&q.ID, &q.TenantID, &q.Name, &q.DefaultAssigneeID, &q.SLAPolicyID,
		&q.CreatedAt, &q.UpdatedAt,
	)
	return &q, err
}

func scanCannedResponse(row scannable) (*CannedResponse, error) {
	var cr CannedResponse
	err := row.Scan(&cr.ID, &cr.TenantID, &cr.Name, &cr.Body, &cr.CreatedAt, &cr.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCannedResponseNotFound
	}
	return &cr, err
}

func scanCannedResponseFromRows(rows pgx.Rows) (*CannedResponse, error) {
	var cr CannedResponse
	err := rows.Scan(&cr.ID, &cr.TenantID, &cr.Name, &cr.Body, &cr.CreatedAt, &cr.UpdatedAt)
	return &cr, err
}

func scanSLAPolicy(row scannable) (*SLAPolicy, error) {
	var (
		p      SLAPolicy
		bhJSON []byte
	)
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.FirstResponseMins, &p.ResolutionMins,
		&bhJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSLAPolicyNotFound
	}
	if err != nil {
		return nil, err
	}
	if bhJSON != nil {
		if err := json.Unmarshal(bhJSON, &p.BusinessHours); err != nil {
			return nil, fmt.Errorf("unmarshal business_hours: %w", err)
		}
	}
	return &p, nil
}

func scanSLAPolicyFromRows(rows pgx.Rows) (*SLAPolicy, error) {
	var (
		p      SLAPolicy
		bhJSON []byte
	)
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.FirstResponseMins, &p.ResolutionMins,
		&bhJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if bhJSON != nil {
		if err := json.Unmarshal(bhJSON, &p.BusinessHours); err != nil {
			return nil, fmt.Errorf("unmarshal business_hours: %w", err)
		}
	}
	return &p, nil
}

func scanKBArticle(row scannable) (*KBArticle, error) {
	var a KBArticle
	err := row.Scan(
		&a.ID, &a.TenantID, &a.Title, &a.Content, &a.Category, &a.Status, &a.AuthorID, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrKBArticleNotFound
	}
	return &a, err
}

func scanKBArticleFromRows(rows pgx.Rows) (*KBArticle, error) {
	var a KBArticle
	err := rows.Scan(
		&a.ID, &a.TenantID, &a.Title, &a.Content, &a.Category, &a.Status, &a.AuthorID, &a.CreatedAt, &a.UpdatedAt,
	)
	return &a, err
}

func scanRoutingRule(row scannable) (*RoutingRule, error) {
	var (
		rr       RoutingRule
		condJSON []byte
	)
	err := row.Scan(
		&rr.ID, &rr.TenantID, &rr.Name, &condJSON, &rr.TargetQueueID, &rr.Priority, &rr.Enabled, &rr.CreatedAt, &rr.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoutingRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	if condJSON != nil {
		if err := json.Unmarshal(condJSON, &rr.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}
	}
	return &rr, nil
}

func scanRoutingRuleFromRows(rows pgx.Rows) (*RoutingRule, error) {
	var (
		rr       RoutingRule
		condJSON []byte
	)
	err := rows.Scan(
		&rr.ID, &rr.TenantID, &rr.Name, &condJSON, &rr.TargetQueueID, &rr.Priority, &rr.Enabled, &rr.CreatedAt, &rr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if condJSON != nil {
		if err := json.Unmarshal(condJSON, &rr.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}
	}
	return &rr, nil
}
