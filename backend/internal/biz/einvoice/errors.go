package einvoice

import "errors"

var (
	// ErrNoEmbeddedXML is returned when a PDF has no embedded XML attachment.
	ErrNoEmbeddedXML = errors.New("PDF does not contain an embedded e-invoice XML attachment")

	// ErrUnsupportedFormat is returned when the XML root element is neither CII nor UBL.
	ErrUnsupportedFormat = errors.New("unsupported e-invoice format: expected ZUGFeRD/Factur-X CII or XRechnung UBL")

	// ErrParseFailed is returned when XML parsing fails structurally.
	ErrParseFailed = errors.New("e-invoice XML parsing failed")

	// ErrDuplicateInvoice is returned when the same supplier + invoice number already exists for this tenant.
	ErrDuplicateInvoice = errors.New("incoming invoice already imported (duplicate supplier + invoice number)")

	// ErrNotFound is returned when an incoming invoice does not exist or belongs to another tenant.
	ErrNotFound = errors.New("incoming invoice not found")

	// ErrInvalidStatusTransition is returned when the requested status change is not permitted.
	ErrInvalidStatusTransition = errors.New("invalid incoming invoice status transition")
)
