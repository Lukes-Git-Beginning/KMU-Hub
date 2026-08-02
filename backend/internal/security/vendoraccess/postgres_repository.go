package vendoraccess

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrRequestNotFound is returned when a request id does not resolve within
// the caller's tenant (either it never existed or it belongs to another
// tenant -- RLS makes the two cases indistinguishable, by design).
var ErrRequestNotFound = errors.New("vendoraccess: request not found")

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed vendor access repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// selectColumns resolves approved_by/revoked_by to display names via a join
// on users, the same shape p1b-roles-list uses for memberCount/capabilityCount.
const selectColumns = `
	r.id, r.reason, r.description, COALESCE(r.ticket_ref, ''), r.agents, r.scope,
	r.requested_start, r.duration_days, r.expires_at, r.status,
	r.counter_proposed_start, r.approved_at, r.approved_by,
	COALESCE(NULLIF(TRIM(au.first_name || ' ' || au.last_name), ''), ''),
	r.sensitive_ack, r.revoked_at, r.revoked_by,
	COALESCE(NULLIF(TRIM(ru.first_name || ' ' || ru.last_name), ''), ''),
	r.completed_at, r.created_at`

const selectJoins = `
	FROM vendor_access_requests r
	LEFT JOIN users au ON au.id = r.approved_by
	LEFT JOIN users ru ON ru.id = r.revoked_by`

func scanRequest(row pgx.Row) (*models.VendorAccessRequest, error) {
	var req models.VendorAccessRequest
	var agentsRaw []byte
	var counterProposedStart, approvedAt, revokedAt, completedAt sql.NullTime
	var approvedBy, revokedBy uuid.NullUUID
	var sensitiveAck sql.NullBool

	err := row.Scan(
		&req.ID, &req.Reason, &req.Description, &req.TicketRef, &agentsRaw, &req.Scope,
		&req.RequestedStart, &req.DurationDays, &req.ExpiresAt, &req.Status,
		&counterProposedStart, &approvedAt, &approvedBy, &req.ApprovedByName,
		&sensitiveAck, &revokedAt, &revokedBy, &req.RevokedByName,
		&completedAt, &req.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if agentsRaw != nil {
		if jsonErr := json.Unmarshal(agentsRaw, &req.Agents); jsonErr != nil {
			return nil, jsonErr
		}
	}
	if counterProposedStart.Valid {
		req.CounterProposedStart = &counterProposedStart.Time
	}
	if approvedAt.Valid {
		req.ApprovedAt = &approvedAt.Time
	}
	if approvedBy.Valid {
		id := approvedBy.UUID
		req.ApprovedBy = &id
	}
	if sensitiveAck.Valid {
		v := sensitiveAck.Bool
		req.SensitiveAck = &v
	}
	if revokedAt.Valid {
		req.RevokedAt = &revokedAt.Time
	}
	if revokedBy.Valid {
		id := revokedBy.UUID
		req.RevokedBy = &id
	}
	if completedAt.Valid {
		req.CompletedAt = &completedAt.Time
	}

	return &req, nil
}

func (r *PostgresRepository) CreateRequest(ctx context.Context, req *models.VendorAccessRequest) error {
	agentsJSON, err := json.Marshal(req.Agents)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO vendor_access_requests
			(id, tenant_id, reason, description, ticket_ref, agents, scope,
			 requested_start, duration_days, expires_at, status, created_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.TenantID, req.Reason, req.Description, req.TicketRef, agentsJSON, req.Scope,
		req.RequestedStart, req.DurationDays, req.ExpiresAt, req.Status, req.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetRequest(ctx context.Context, tenantID, id uuid.UUID) (*models.VendorAccessRequest, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+selectColumns+selectJoins+` WHERE r.tenant_id = $1 AND r.id = $2`,
		tenantID, id,
	)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	req.TenantID = tenantID
	return req, nil
}

func (r *PostgresRepository) ListRequests(ctx context.Context, tenantID uuid.UUID) ([]*models.VendorAccessRequest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+selectColumns+selectJoins+` WHERE r.tenant_id = $1 ORDER BY r.created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*models.VendorAccessRequest, 0)
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		req.TenantID = tenantID
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, req *models.VendorAccessRequest) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vendor_access_requests SET
			status = $1,
			counter_proposed_start = $2,
			approved_at = $3,
			approved_by = $4,
			sensitive_ack = $5,
			revoked_at = $6,
			revoked_by = $7,
			completed_at = $8
		 WHERE tenant_id = $9 AND id = $10`,
		req.Status, nullTime(req.CounterProposedStart), nullTime(req.ApprovedAt), nullUUID(req.ApprovedBy),
		nullBool(req.SensitiveAck), nullTime(req.RevokedAt), nullUUID(req.RevokedBy), nullTime(req.CompletedAt),
		tenantID, req.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRequestNotFound
	}
	return nil
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func nullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}
