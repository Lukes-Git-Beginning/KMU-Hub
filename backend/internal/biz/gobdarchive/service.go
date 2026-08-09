// Package gobdarchive implements a GoBD-compliant, immutable document archive
// for Belege (receipts, invoices, contracts, correspondence) as mandated by §147 AO.
//
// Key invariants:
//   - Documents are NEVER modified or deleted after archival.
//   - SHA-256 is computed via io.TeeReader during upload — content and hash are always consistent.
//   - Retention = 31.12 of (archivedAt.Year + 10) UTC (§147 AO: 10-year retention).
//   - ArchiveInvoice only accepts invoices with LockedAt set (GoBD: finalize before archive).
package gobdarchive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	// maxArchiveBytes is the maximum file size accepted for archival (50 MiB).
	maxArchiveBytes = 50 << 20

	// presignedURLExpiry is how long download presigned URLs remain valid.
	presignedURLExpiry = time.Hour
)

// InvoiceReader is the minimal invoice accessor needed for ArchiveInvoice.
// Implemented by *invoice.Service to avoid a circular dependency.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// Service handles GoBD archive business logic.
type Service struct {
	repo    Repository
	store   chatfile.FileStore
	invoice InvoiceReader // optional; nil disables ArchiveInvoiceDocument
}

// NewService creates a GoBD archive service.
// store: MinIO/S3 adapter for the actual file bytes.
// invoiceReader: optional; pass nil if invoice-based archival is not needed at startup.
func NewService(repo Repository, store chatfile.FileStore, invoiceReader InvoiceReader) *Service {
	return &Service{
		repo:    repo,
		store:   store,
		invoice: invoiceReader,
	}
}

// ============================================================================
// ArchiveDocument
// ============================================================================

// ArchiveInput contains the data required to archive a document.
type ArchiveInput struct {
	TenantID        uuid.UUID
	DocType         string
	SourceInvoiceID *uuid.UUID // nil unless the document relates to an invoice
	OriginalFilename string
	MimeType        string
	Content         io.Reader
	Size            int64 // must be known upfront for MinIO
	ArchivedBy      uuid.UUID
}

// ArchiveDocument validates, uploads, hashes and records a new archived document.
func (s *Service) ArchiveDocument(ctx context.Context, input ArchiveInput) (*models.GobdDocument, error) {
	if input.Size <= 0 {
		return nil, ErrEmptyFile
	}
	if input.Size > maxArchiveBytes {
		return nil, ErrFileTooLarge
	}
	if !isValidDocType(input.DocType) {
		return nil, ErrInvalidDocumentType
	}

	docID := uuid.New()
	storageKey := fmt.Sprintf("gobd/%s/%s/%s",
		input.TenantID.String(),
		docID.String(),
		input.OriginalFilename,
	)

	// Compute SHA-256 while streaming to MinIO via TeeReader.
	h := sha256.New()
	tee := io.TeeReader(input.Content, h)

	if err := s.store.Upload(ctx, storageKey, tee, input.Size, input.MimeType); err != nil {
		return nil, fmt.Errorf("upload gobd document: %w", err)
	}

	checksum := hex.EncodeToString(h.Sum(nil))
	now := time.Now().UTC()

	doc := &models.GobdDocument{
		ID:               docID,
		TenantID:         input.TenantID,
		DocType:          input.DocType,
		SourceInvoiceID:  input.SourceInvoiceID,
		StorageKey:       storageKey,
		SHA256:           checksum,
		OriginalFilename: input.OriginalFilename,
		MimeType:         input.MimeType,
		FileSizeBytes:    input.Size,
		ArchivedAt:       now,
		ArchivedBy:       input.ArchivedBy,
		RetentionUntil:   computeRetentionUntil(now),
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("persist gobd document: %w", err)
	}

	// Append archived event.
	if err := s.appendEvent(ctx, doc.TenantID, doc.ID, models.GobdEventTypeArchived, input.ArchivedBy, "", nil); err != nil {
		// Non-fatal: the document is already persisted.
		slog.Warn("failed to append archived event",
			"document_id", doc.ID,
			"error", err,
		)
	}

	slog.Info("gobd document archived",
		"document_id", doc.ID,
		"tenant_id", doc.TenantID,
		"doc_type", doc.DocType,
		"sha256", doc.SHA256,
		"retention_until", doc.RetentionUntil.Format("2006-01-02"),
	)

	return doc, nil
}

// ============================================================================
// ArchiveInvoiceDocument
// ============================================================================

// ArchiveInvoiceDocument archives a PDF/document for a locked invoice.
// The invoice must have LockedAt set (GoBD: only finalized documents are archived).
// pdfContent should be the rendered invoice PDF bytes.
func (s *Service) ArchiveInvoiceDocument(ctx context.Context, tenantID, invoiceID, archivedBy uuid.UUID, pdfContent []byte) (*models.GobdDocument, error) {
	if s.invoice == nil {
		return nil, fmt.Errorf("invoice reader not configured")
	}

	inv, err := s.invoice.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}

	if inv.LockedAt == nil {
		return nil, ErrInvoiceNotLocked
	}

	filename := fmt.Sprintf("invoice-%s.pdf", inv.InvoiceNumber)
	if inv.InvoiceNumber == "" {
		filename = fmt.Sprintf("invoice-%s.pdf", invoiceID.String())
	}

	return s.ArchiveDocument(ctx, ArchiveInput{
		TenantID:         tenantID,
		DocType:          models.GobdDocTypeInvoice,
		SourceInvoiceID:  &invoiceID,
		OriginalFilename: filename,
		MimeType:         "application/pdf",
		Content:          bytes.NewReader(pdfContent),
		Size:             int64(len(pdfContent)),
		ArchivedBy:       archivedBy,
	})
}

// ============================================================================
// GetByID
// ============================================================================

// GetByID retrieves a single document, ensuring tenant isolation.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.GobdDocument, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// ============================================================================
// List
// ============================================================================

// List returns paginated documents matching the filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.GobdDocument, int, error) {
	return s.repo.List(ctx, filter)
}

// ============================================================================
// GetDownloadURL
// ============================================================================

// GetDownloadURL generates a short-lived presigned URL for the document and
// records an access event.
func (s *Service) GetDownloadURL(ctx context.Context, tenantID, id, requestedBy uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", err
	}

	url, err := s.store.GetPresignedURL(ctx, doc.StorageKey, presignedURLExpiry)
	if err != nil {
		return "", fmt.Errorf("generate presigned url: %w", err)
	}

	// Append access event (non-fatal).
	if evErr := s.appendEvent(ctx, tenantID, id, models.GobdEventTypeAccess, requestedBy, "", nil); evErr != nil {
		slog.Warn("failed to append access event",
			"document_id", id,
			"error", evErr,
		)
	}

	return url, nil
}

// ============================================================================
// AddAnnotation
// ============================================================================

// AddAnnotation appends a text annotation to an existing document's audit trail.
func (s *Service) AddAnnotation(ctx context.Context, tenantID, id, addedBy uuid.UUID, note string) error {
	// Load document to verify existence and tenant isolation.
	if _, err := s.repo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}

	if err := s.appendEvent(ctx, tenantID, id, models.GobdEventTypeAnnotation, addedBy, note, nil); err != nil {
		return fmt.Errorf("append annotation: %w", err)
	}

	slog.Info("gobd document annotation added",
		"document_id", id,
		"tenant_id", tenantID,
	)

	return nil
}

// ============================================================================
// ListEvents
// ============================================================================

// ListEvents returns the full audit trail for a document.
func (s *Service) ListEvents(ctx context.Context, tenantID, id uuid.UUID) ([]*models.GobdDocumentEvent, error) {
	// Verify document exists and belongs to tenant.
	if _, err := s.repo.GetByID(ctx, tenantID, id); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, tenantID, id)
}

// ============================================================================
// Private helpers
// ============================================================================

// computeRetentionUntil returns 31.12 of (archivedAt.Year + 10) in UTC.
// Example: archivedAt=2026-03-15 → 2036-12-31; archivedAt=2026-12-31 → 2036-12-31.
// §147 Abs. 3 AO mandates a 10-year retention for invoices and books, counted from
// the end of the calendar year of the last entry — hence year+10 at 31.12.
func computeRetentionUntil(archivedAt time.Time) time.Time {
	return time.Date(archivedAt.Year()+10, time.December, 31, 0, 0, 0, 0, time.UTC)
}

// isValidDocType returns true if the string is a known gobd_document_type ENUM value.
func isValidDocType(docType string) bool {
	switch docType {
	case models.GobdDocTypeInvoice,
		models.GobdDocTypeCreditNote,
		models.GobdDocTypeReceipt,
		models.GobdDocTypeContract,
		models.GobdDocTypeCorrespondence,
		models.GobdDocTypeOther:
		return true
	default:
		return false
	}
}

// appendEvent is an internal helper that creates and persists an audit event.
func (s *Service) appendEvent(ctx context.Context, tenantID, documentID uuid.UUID, eventType string, createdBy uuid.UUID, note string, metadata []byte) error {
	if metadata == nil {
		metadata = []byte("{}")
	}
	ev := &models.GobdDocumentEvent{
		ID:         uuid.New(),
		TenantID:   tenantID,
		DocumentID: documentID,
		EventType:  eventType,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now().UTC(),
		Note:       note,
		Metadata:   metadata,
	}
	return s.repo.AppendEvent(ctx, ev)
}
