package invoice

import "errors"

var (
	// ErrInvoiceNotFound is returned when an invoice does not exist.
	ErrInvoiceNotFound = errors.New("invoice not found")
	// ErrInvoiceImmutable is returned when trying to modify a non-draft invoice (GoBD enforcement).
	ErrInvoiceImmutable = errors.New("invoice is immutable after being sent (GoBD compliance)")
	// ErrInvoiceNotDraft is returned when an operation requires draft status.
	ErrInvoiceNotDraft = errors.New("invoice is not in draft status")
	// ErrInvoiceAlreadyPaid is returned when trying to pay or cancel an already-paid invoice.
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
	// ErrInvoiceAlreadyCancelled is returned when trying to modify a cancelled invoice.
	ErrInvoiceAlreadyCancelled = errors.New("invoice is already cancelled")
	// ErrNoLineItems is returned when an invoice has no line items.
	ErrNoLineItems = errors.New("invoice must have at least one line item")
	// ErrQuoteNotAccepted is returned when trying to create an invoice from a non-accepted quote.
	ErrQuoteNotAccepted = errors.New("quote must be accepted before conversion to invoice")
	// ErrStornoUnavailable is returned when a sent/overdue invoice cannot be
	// cancelled because no StornoCreator is wired — the invoice is never silently
	// flipped to cancelled without issuing the reversing credit note (GoBD §146).
	ErrStornoUnavailable = errors.New("cannot cancel an issued invoice: storno credit note path not configured")
)
