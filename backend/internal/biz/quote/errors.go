package quote

import "errors"

var (
	// ErrQuoteNotFound is returned when a quote does not exist.
	ErrQuoteNotFound = errors.New("quote not found")
	// ErrQuoteNotDraft is returned when an operation requires draft status.
	ErrQuoteNotDraft = errors.New("quote is not in draft status")
	// ErrQuoteNotSent is returned when an operation requires sent status.
	ErrQuoteNotSent = errors.New("quote is not in sent status")
	// ErrNoLineItems is returned when a quote has no line items.
	ErrNoLineItems = errors.New("quote must have at least one line item")
)
