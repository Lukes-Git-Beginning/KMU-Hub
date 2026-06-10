package gobdarchive

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed GoBD archive repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create persists a new GoBD document. The caller must have already stamped tenant
// context on the context (via middleware.TenantIDKey) so RLS allows the insert.
func (r *PostgresRepository) Create(ctx context.Context, doc *models.GobdDocument) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gobd_documents (
			id, tenant_id, doc_type, source_invoice_id,
			storage_key, sha256, original_filename, mime_type,
			file_size_bytes, archived_at, archived_by, retention_until
		) VALUES (
			$1, $2, $3::gobd_document_type, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12
		)`,
		doc.ID, doc.TenantID, doc.DocType, doc.SourceInvoiceID,
		doc.StorageKey, doc.SHA256, doc.OriginalFilename, doc.MimeType,
		doc.FileSizeBytes, doc.ArchivedAt, doc.ArchivedBy, doc.RetentionUntil,
	)
	if err != nil {
		return fmt.Errorf("insert gobd_documents: %w", err)
	}
	return nil
}

// GetByID retrieves a document by tenant + ID.
func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.GobdDocument, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
			id, tenant_id, doc_type::text,
			COALESCE(source_invoice_id, '00000000-0000-0000-0000-000000000000'::uuid) AS source_invoice_id,
			storage_key, sha256, original_filename, mime_type,
			file_size_bytes, archived_at, archived_by, retention_until
		FROM gobd_documents
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return r.scanDocument(row)
}

// List returns paginated documents matching the filter.
func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.GobdDocument, int, error) {
	perPage := filter.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	where := []string{"tenant_id = $1"}
	args := []any{filter.TenantID}
	idx := 2

	if filter.DocType != "" {
		where = append(where, fmt.Sprintf("doc_type = $%d::gobd_document_type", idx))
		args = append(args, filter.DocType)
		idx++
	}
	if filter.SourceInvoiceID != nil {
		where = append(where, fmt.Sprintf("source_invoice_id = $%d", idx))
		args = append(args, *filter.SourceInvoiceID)
		idx++
	}
	if filter.DateFrom != nil {
		where = append(where, fmt.Sprintf("archived_at >= $%d", idx))
		args = append(args, *filter.DateFrom)
		idx++
	}
	if filter.DateTo != nil {
		where = append(where, fmt.Sprintf("archived_at <= $%d", idx))
		args = append(args, *filter.DateTo)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gobd_documents WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count gobd_documents: %w", err)
	}

	// Paginated list
	listQuery := fmt.Sprintf(`
		SELECT
			id, tenant_id, doc_type::text,
			COALESCE(source_invoice_id, '00000000-0000-0000-0000-000000000000'::uuid) AS source_invoice_id,
			storage_key, sha256, original_filename, mime_type,
			file_size_bytes, archived_at, archived_by, retention_until
		FROM gobd_documents
		WHERE %s
		ORDER BY archived_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, idx, idx+1,
	)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list gobd_documents: %w", err)
	}
	defer rows.Close()

	var docs []*models.GobdDocument
	for rows.Next() {
		doc, scanErr := r.scanDocumentFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate gobd_documents: %w", err)
	}

	return docs, total, nil
}

// AppendEvent appends an audit event record.
func (r *PostgresRepository) AppendEvent(ctx context.Context, event *models.GobdDocumentEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gobd_document_events (
			id, tenant_id, document_id, event_type,
			created_by, created_at, note, metadata
		) VALUES (
			$1, $2, $3, $4::gobd_event_type,
			$5, $6, $7, $8
		)`,
		event.ID, event.TenantID, event.DocumentID, event.EventType,
		event.CreatedBy, event.CreatedAt, event.Note, event.Metadata,
	)
	if err != nil {
		return fmt.Errorf("insert gobd_document_events: %w", err)
	}
	return nil
}

// ListEvents returns all events for a document ordered by created_at DESC.
func (r *PostgresRepository) ListEvents(ctx context.Context, tenantID, documentID uuid.UUID) ([]*models.GobdDocumentEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, document_id, event_type::text,
			created_by, created_at, note, metadata
		FROM gobd_document_events
		WHERE tenant_id = $1 AND document_id = $2
		ORDER BY created_at DESC`,
		tenantID, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list gobd_document_events: %w", err)
	}
	defer rows.Close()

	var events []*models.GobdDocumentEvent
	for rows.Next() {
		ev := &models.GobdDocumentEvent{}
		if scanErr := rows.Scan(
			&ev.ID, &ev.TenantID, &ev.DocumentID, &ev.EventType,
			&ev.CreatedBy, &ev.CreatedAt, &ev.Note, &ev.Metadata,
		); scanErr != nil {
			return nil, fmt.Errorf("scan gobd_document_event: %w", scanErr)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gobd_document_events: %w", err)
	}
	return events, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

// scanDocument scans a single row into a GobdDocument.
// source_invoice_id is COALESCE'd to uuid.Nil in the query; we convert it back
// to nil pointer here so callers see the logical nullable.
func (r *PostgresRepository) scanDocument(row pgx.Row) (*models.GobdDocument, error) {
	doc := &models.GobdDocument{}
	var sourceInvoiceID uuid.UUID
	err := row.Scan(
		&doc.ID, &doc.TenantID, &doc.DocType,
		&sourceInvoiceID,
		&doc.StorageKey, &doc.SHA256, &doc.OriginalFilename, &doc.MimeType,
		&doc.FileSizeBytes, &doc.ArchivedAt, &doc.ArchivedBy, &doc.RetentionUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("scan gobd_document: %w", err)
	}
	if sourceInvoiceID != uuid.Nil {
		doc.SourceInvoiceID = &sourceInvoiceID
	}
	return doc, nil
}

// scanDocumentFromRows scans from pgx.Rows (used in List).
func (r *PostgresRepository) scanDocumentFromRows(rows pgx.Rows) (*models.GobdDocument, error) {
	doc := &models.GobdDocument{}
	var sourceInvoiceID uuid.UUID
	err := rows.Scan(
		&doc.ID, &doc.TenantID, &doc.DocType,
		&sourceInvoiceID,
		&doc.StorageKey, &doc.SHA256, &doc.OriginalFilename, &doc.MimeType,
		&doc.FileSizeBytes, &doc.ArchivedAt, &doc.ArchivedBy, &doc.RetentionUntil,
	)
	if err != nil {
		return nil, fmt.Errorf("scan gobd_document row: %w", err)
	}
	if sourceInvoiceID != uuid.Nil {
		doc.SourceInvoiceID = &sourceInvoiceID
	}
	return doc, nil
}
