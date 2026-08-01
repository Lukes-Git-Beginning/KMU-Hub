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

	// ErrGenerateFailed is returned when an outbound e-invoice cannot be rendered
	// because the source invoice lacks a structurally required value.
	ErrGenerateFailed = errors.New("e-invoice generation failed")

	// ErrTotalsMismatch is returned when the totals recomputed from the line items
	// disagree with the totals stored on the invoice. Emitting the document anyway
	// would send the customer a different amount than the PDF shows.
	ErrTotalsMismatch = errors.New("recomputed invoice totals disagree with the stored totals")

	// ErrValidationFailed is returned when an invoice is renderable but misses
	// values the receiver requires, so the document would be rejected on arrival.
	// The concrete violations travel on *ValidationError; use errors.As to read
	// them and report all of them to the user at once.
	ErrValidationFailed = errors.New("invoice does not meet the e-invoice requirements")
)
