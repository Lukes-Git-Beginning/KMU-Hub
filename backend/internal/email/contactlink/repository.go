package contactlink

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements the EmailContactLinkRepository for PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new contact link repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts a new email-contact link.
func (r *PostgresRepository) Create(ctx context.Context, link *models.EmailContactLink) error {
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_contact_links (id, tenant_id, message_id, contact_id, link_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (message_id, contact_id) DO NOTHING`,
		link.ID, link.TenantID, link.MessageID, link.ContactID, link.LinkType, link.CreatedAt,
	)
	return err
}

// GetByMessageID returns all CRM contact links for an email message within a tenant.
func (r *PostgresRepository) GetByMessageID(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailContactLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, message_id, contact_id, link_type, created_at
		 FROM email_contact_links WHERE message_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC`, messageID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.EmailContactLink
	for rows.Next() {
		var link models.EmailContactLink
		if err := rows.Scan(&link.ID, &link.TenantID, &link.MessageID, &link.ContactID, &link.LinkType, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, &link)
	}
	return links, rows.Err()
}

// GetByContactID returns all email messages linked to a CRM contact within a tenant, with pagination.
func (r *PostgresRepository) GetByContactID(ctx context.Context, contactID uuid.UUID, tenantID uuid.UUID, page, perPage int) ([]*models.EmailMessage, int, error) {
	offset := (page - 1) * perPage

	// Count total
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_contact_links WHERE contact_id = $1 AND tenant_id = $2`, contactID, tenantID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch messages via join
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.account_id, m.folder_id, m.uid, m.message_id,
			m.in_reply_to, m.references, m.thread_id,
			m.from_name, m.from_email,
			m.to_addresses, m.cc_addresses, m.bcc_addresses,
			m.subject, m.preview, m.body_text, m.body_html,
			m.is_read, m.is_starred, m.is_draft, m.has_attachments,
			m.date, m.size_bytes, m.created_at, m.updated_at
		 FROM email_messages m
		 INNER JOIN email_contact_links l ON l.message_id = m.id
		 WHERE l.contact_id = $1 AND l.tenant_id = $2 AND m.tenant_id = $2
		 ORDER BY m.date DESC
		 LIMIT $3 OFFSET $4`,
		contactID, tenantID, perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*models.EmailMessage
	for rows.Next() {
		var m models.EmailMessage
		var toJSON, ccJSON, bccJSON []byte
		if err := rows.Scan(
			&m.ID, &m.AccountID, &m.FolderID, &m.UID, &m.MessageID,
			&m.InReplyTo, &m.References, &m.ThreadID,
			&m.FromName, &m.FromEmail,
			&toJSON, &ccJSON, &bccJSON,
			&m.Subject, &m.Preview, &m.BodyText, &m.BodyHTML,
			&m.IsRead, &m.IsStarred, &m.IsDraft, &m.HasAttachments,
			&m.Date, &m.SizeBytes, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		// JSON addresses are stored as JSONB in PostgreSQL
		_ = toJSON
		_ = ccJSON
		_ = bccJSON
		messages = append(messages, &m)
	}

	return messages, total, rows.Err()
}

// Delete removes a link between a message and a contact within a tenant.
func (r *PostgresRepository) Delete(ctx context.Context, messageID, contactID uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM email_contact_links WHERE message_id = $1 AND contact_id = $2 AND tenant_id = $3`,
		messageID, contactID, tenantID,
	)
	return err
}
