package gobdarchive

import "errors"

var (
	// ErrDocumentNotFound is returned when a GoBD document does not exist for the tenant.
	ErrDocumentNotFound = errors.New("gobd document not found")

	// ErrDocumentImmutable is returned when trying to modify an archived document.
	// GoBD compliance: archived documents are permanently immutable.
	ErrDocumentImmutable = errors.New("gobd document is immutable after archival (§147 AO)")

	// ErrInvalidDocumentType is returned when an unknown document type string is supplied.
	ErrInvalidDocumentType = errors.New("invalid gobd document type")

	// ErrFileTooLarge is returned when the uploaded file exceeds the 50 MiB limit.
	ErrFileTooLarge = errors.New("file exceeds maximum archive size of 50 MiB")

	// ErrEmptyFile is returned when a zero-byte file is submitted for archival.
	ErrEmptyFile = errors.New("file must not be empty")

	// ErrInvoiceNotLocked is returned when ArchiveInvoice is called but the invoice
	// has not yet been locked via LockInvoice. GoBD: archive only completed documents.
	ErrInvoiceNotLocked = errors.New("invoice must be locked before archival (GoBD: only finalized documents)")
)
