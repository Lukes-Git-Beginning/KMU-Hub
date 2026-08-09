package thread

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository backed by a pgxpool.Pool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a thread/canned Repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ---------------------------------------------------------------------------
// Thread messages
// ---------------------------------------------------------------------------

func (r *PostgresRepository) AppendThreadMessage(ctx context.Context, m *ThreadMessage) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inbox_thread_messages
		    (id, tenant_id, message_id, author_id, author_name, direction, body, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m.ID, m.TenantID, m.MessageID, m.AuthorID, nullString(m.AuthorName), m.Direction, m.Body, m.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) SeedInboundIfEmpty(ctx context.Context, tenantID, messageID uuid.UUID, senderName, body string, createdAt time.Time) error {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inbox_thread_messages (id, tenant_id, message_id, author_name, direction, body, created_at)
		 SELECT gen_random_uuid(), $1, $2, $3, 'inbound', $4, $5
		 WHERE NOT EXISTS (
		     SELECT 1 FROM inbox_thread_messages WHERE message_id = $2 AND tenant_id = $1
		 )`,
		tenantID, messageID, nullString(senderName), body, createdAt,
	)
	return err
}

func (r *PostgresRepository) ListThreadMessages(ctx context.Context, tenantID, messageID uuid.UUID) ([]*ThreadMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tm.id, tm.tenant_id, tm.message_id, tm.author_id, tm.direction, tm.body, tm.created_at,
		        COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.first_name, u.last_name)), ''), u.email, tm.author_name, '') AS author_name
		 FROM inbox_thread_messages tm
		 LEFT JOIN users u ON tm.author_id = u.id
		 WHERE tm.message_id = $1 AND tm.tenant_id = $2
		 ORDER BY tm.created_at ASC`,
		messageID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ThreadMessage
	for rows.Next() {
		var m ThreadMessage
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.MessageID, &m.AuthorID, &m.Direction, &m.Body, &m.CreatedAt, &m.AuthorName,
		); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Canned responses
// ---------------------------------------------------------------------------

func (r *PostgresRepository) CreateCannedResponse(ctx context.Context, c *CannedResponse) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inbox_canned_responses (id, tenant_id, name, body, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		c.ID, c.TenantID, c.Name, c.Body, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) ListCannedResponses(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, body, created_at, updated_at
		 FROM inbox_canned_responses WHERE tenant_id = $1 ORDER BY name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*CannedResponse
	for rows.Next() {
		var c CannedResponse
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetCannedResponse(ctx context.Context, tenantID, id uuid.UUID) (*CannedResponse, error) {
	var c CannedResponse
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, body, created_at, updated_at
		 FROM inbox_canned_responses WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.Name, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCannedResponseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) UpdateCannedResponse(ctx context.Context, c *CannedResponse) error {
	c.UpdatedAt = time.Now().UTC()
	ct, err := r.pool.Exec(ctx,
		`UPDATE inbox_canned_responses SET name = $1, body = $2, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5`,
		c.Name, c.Body, c.UpdatedAt, c.ID, c.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrCannedResponseNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteCannedResponse(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM inbox_canned_responses WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrCannedResponseNotFound
	}
	return nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
