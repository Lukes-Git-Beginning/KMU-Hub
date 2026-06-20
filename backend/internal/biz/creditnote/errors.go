package creditnote

import "errors"

var (
	// ErrCreditNoteNotFound is returned when a credit note does not exist.
	ErrCreditNoteNotFound = errors.New("credit note not found")
	// ErrCreditNoteNotDraft is returned when an operation requires draft status.
	ErrCreditNoteNotDraft = errors.New("credit note is not in draft status")
	// ErrInvalidInvoiceForCredit is returned when the referenced invoice is not in a valid state for crediting.
	ErrInvalidInvoiceForCredit = errors.New("invoice must be sent, paid, or overdue to create a credit note")
	// ErrNoLineItems is returned when a credit note has no line items.
	ErrNoLineItems = errors.New("credit note must have at least one line item")
	// ErrInvoiceAlreadyCredited is returned when an automatic Storno is requested
	// for an invoice that already carries a sent credit note (would over-credit).
	ErrInvoiceAlreadyCredited = errors.New("invoice already has a sent credit note; resolve the storno manually")
)
