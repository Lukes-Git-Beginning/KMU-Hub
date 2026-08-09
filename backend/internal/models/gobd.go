package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// GoBD Archive Models (§147 AO — Revisionssichere Beleg-Ablage)
// ============================================================================

// GobdDocument represents an immutable archived document in the GoBD-compliant
// archive. Once created, the record must NEVER be modified or deleted.
// Retention is calculated as 31.12 of (archivedAt.Year + 10) UTC (§147 AO).
type GobdDocument struct {
	ID               uuid.UUID  `json:"id"                 db:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"          db:"tenant_id"`
	DocType          string     `json:"doc_type"           db:"doc_type"`
	SourceInvoiceID  *uuid.UUID `json:"source_invoice_id"  db:"source_invoice_id"`
	StorageKey       string     `json:"storage_key"        db:"storage_key"`
	SHA256           string     `json:"sha256"             db:"sha256"`
	OriginalFilename string     `json:"original_filename"  db:"original_filename"`
	MimeType         string     `json:"mime_type"          db:"mime_type"`
	FileSizeBytes    int64      `json:"file_size_bytes"    db:"file_size_bytes"`
	ArchivedAt       time.Time  `json:"archived_at"        db:"archived_at"`
	ArchivedBy       uuid.UUID  `json:"archived_by"        db:"archived_by"`
	RetentionUntil   time.Time  `json:"retention_until"    db:"retention_until"`
}

// GobdDocumentEvent is an append-only audit trail entry for a GoBD document.
// Events are never deleted (they form the immutable audit trail).
type GobdDocumentEvent struct {
	ID         uuid.UUID       `json:"id"          db:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"   db:"tenant_id"`
	DocumentID uuid.UUID       `json:"document_id" db:"document_id"`
	EventType  string          `json:"event_type"  db:"event_type"`
	CreatedBy  uuid.UUID       `json:"created_by"  db:"created_by"`
	CreatedAt  time.Time       `json:"created_at"  db:"created_at"`
	Note       string          `json:"note"        db:"note"`
	Metadata   json.RawMessage `json:"metadata"    db:"metadata"`
}

// GoBD document type constants (mirrors gobd_document_type ENUM in DB)
const (
	GobdDocTypeInvoice        = "invoice"
	GobdDocTypeCreditNote     = "credit_note"
	GobdDocTypeReceipt        = "receipt"
	GobdDocTypeContract       = "contract"
	GobdDocTypeCorrespondence = "correspondence"
	GobdDocTypeOther          = "other"
)

// GoBD event type constants (mirrors gobd_event_type ENUM in DB)
const (
	GobdEventTypeArchived        = "archived"
	GobdEventTypeAnnotation      = "annotation"
	GobdEventTypeAccess          = "access"
	GobdEventTypeIntegrityCheck  = "integrity_check"
)
