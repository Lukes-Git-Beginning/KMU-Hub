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
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tickets
		    (id, tenant_id, subject, status, priority, assignee_id, requester_id,
		     queue_id, due_at, merged_into_id, first_response_at, resolved_at,
		     created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.ID, t.TenantID, t.Subject, t.Status, t.Priority,
		t.AssigneeID, t.RequesterID, t.QueueID, t.DueAt,
		t.MergedIntoID, t.FirstResponseAt, t.ResolvedAt,
		t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, subject, status, priority, assignee_id, requester_id,
		        queue_id, due_at, merged_into_id, first_response_at, resolved_at,
		        created_at, updated_at
		 FROM tickets WHERE id = $1`, id,
	)
	return scanTicket(row)
}

func (r *PostgresRepository) ListTickets(ctx context.Context, tenantID uuid.UUID, statusFilter *string, page, pageSize int) ([]*Ticket, int, error) {
	offset := (page - 1) * pageSize

	var (
		whereExtra string
		args       []any
	)
	args = append(args, tenantID)
	if statusFilter != nil {
		whereExtra = " AND status = $2"
		args = append(args, *statusFilter)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tickets WHERE tenant_id = $1%s", whereExtra)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := len(args) + 1
	offsetArg := limitArg + 1
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, subject, status, priority, assignee_id, requester_id,
		        queue_id, due_at, merged_into_id, first_response_at, resolved_at,
		        created_at, updated_at
		 FROM tickets
		 WHERE tenant_id = $1%s
		 ORDER BY created_at DESC
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
	_, err := r.pool.Exec(ctx,
		`UPDATE tickets
		 SET subject = $1, status = $2, priority = $3, assignee_id = $4,
		     queue_id = $5, due_at = $6, merged_into_id = $7,
		     first_response_at = $8, resolved_at = $9, updated_at = $10
		 WHERE id = $11`,
		t.Subject, t.Status, t.Priority, t.AssigneeID,
		t.QueueID, t.DueAt, t.MergedIntoID,
		t.FirstResponseAt, t.ResolvedAt, t.UpdatedAt,
		t.ID,
	)
	return err
}

func (r *PostgresRepository) DeleteTicket(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tickets WHERE id = $1`, id)
	return err
}

// FindOpenTicketsByRequester returns non-merged tickets for requester whose
// subject begins with subjectPrefix (case-insensitive ILIKE).
//
// Note: pg_trgm similarity search would give better fuzzy results but requires
// the pg_trgm extension. We use ILIKE-prefix as a safe default. If pg_trgm is
// available in production, replace the WHERE clause with:
//   similarity(subject, $3) > 0.4
func (r *PostgresRepository) FindOpenTicketsByRequester(ctx context.Context, tenantID, requesterID uuid.UUID, subjectPrefix string) ([]*Ticket, error) {
	prefix := strings.TrimSpace(subjectPrefix)
	if len(prefix) > 60 {
		prefix = prefix[:60]
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, subject, status, priority, assignee_id, requester_id,
		        queue_id, due_at, merged_into_id, first_response_at, resolved_at,
		        created_at, updated_at
		 FROM tickets
		 WHERE tenant_id = $1
		   AND requester_id = $2
		   AND status NOT IN ('merged','closed','solved')
		   AND subject ILIKE $3
		 ORDER BY created_at DESC
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

// ListMessagesByTicket — tenant scoping is enforced via RLS (mig 000126).
func (r *PostgresRepository) ListMessagesByTicket(ctx context.Context, ticketID uuid.UUID) ([]*TicketMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, ticket_id, author_id, body, internal, attachments, created_at
		 FROM ticket_messages
		 WHERE ticket_id = $1
		 ORDER BY created_at ASC`,
		ticketID,
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

func (r *PostgresRepository) ReassignMessages(ctx context.Context, sourceTicketID, targetTicketID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ticket_messages SET ticket_id = $1 WHERE ticket_id = $2`,
		targetTicketID, sourceTicketID,
	)
	return err
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

func (r *PostgresRepository) GetQueueByID(ctx context.Context, id uuid.UUID) (*TicketQueue, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, default_assignee_id, sla_policy_id, created_at, updated_at
		 FROM ticket_queues WHERE id = $1`, id,
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
	_, err := r.pool.Exec(ctx,
		`UPDATE ticket_queues
		 SET name = $1, default_assignee_id = $2, sla_policy_id = $3, updated_at = $4
		 WHERE id = $5`,
		q.Name, q.DefaultAssigneeID, q.SLAPolicyID, q.UpdatedAt, q.ID,
	)
	return err
}

func (r *PostgresRepository) DeleteQueue(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM ticket_queues WHERE id = $1`, id)
	return err
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

func (r *PostgresRepository) GetCannedResponseByID(ctx context.Context, id uuid.UUID) (*CannedResponse, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, body, created_at, updated_at
		 FROM canned_responses WHERE id = $1`, id,
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
	_, err := r.pool.Exec(ctx,
		`UPDATE canned_responses
		 SET name = $1, body = $2, updated_at = $3
		 WHERE id = $4`,
		cr.Name, cr.Body, cr.UpdatedAt, cr.ID,
	)
	return err
}

func (r *PostgresRepository) DeleteCannedResponse(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM canned_responses WHERE id = $1`, id)
	return err
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

func (r *PostgresRepository) GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*SLAPolicy, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, first_response_mins, resolution_mins,
		        business_hours, created_at, updated_at
		 FROM sla_policies WHERE id = $1`, id,
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
	_, err = r.pool.Exec(ctx,
		`UPDATE sla_policies
		 SET name = $1, first_response_mins = $2, resolution_mins = $3,
		     business_hours = $4, updated_at = $5
		 WHERE id = $6`,
		p.Name, p.FirstResponseMins, p.ResolutionMins, bhJSON, p.UpdatedAt, p.ID,
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
	)
	return &t, err
}

func scanMessageFromRows(rows pgx.Rows) (*TicketMessage, error) {
	var (
		m           TicketMessage
		attachJSON  []byte
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
