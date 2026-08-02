package banking

// Reconciliation: which open receivable does a bank credit belong to?
//
// The matcher never books anything. It attaches a suggestion and the reason it
// arrived at it, and a human confirms. Automatic booking would be a real
// business event decided by a heuristic — an invoice marked paid because a
// customer happened to transfer the same amount for something else is a worse
// failure than an unreconciled entry sitting in a queue.
//
// It also refuses to guess between equals: a candidate is only suggested when it
// is the single one of its kind. Two invoices over 119,00 EUR produce no
// suggestion at all rather than a coin flip.

import (
	"strings"
	"unicode"

	"github.com/kmuhub/kmuhub/internal/models"
)

// minReferenceLength is the shortest invoice number still searched for inside
// remittance text. Below it a number like "7" would match any transfer carrying
// a seven, so short numbers are only reachable through the amount rule.
const minReferenceLength = 4

// Match reasons, stored verbatim on the transaction so a later reader can tell
// why an entry was suggested without re-running the matcher.
const (
	// MatchReasonNumberAndAmount: the invoice number appears in the remittance
	// text and the amount settles the open balance exactly.
	MatchReasonNumberAndAmount = "invoice_number+amount"
	// MatchReasonNumber: the invoice number appears, the amount differs — a
	// partial payment, a deduction, or a customer paying two invoices at once.
	MatchReasonNumber = "invoice_number"
	// MatchReasonAmount: no reference found, but exactly one open item matches
	// the amount and currency.
	MatchReasonAmount = "amount"
)

// MatchResult is the matcher's verdict for one transaction.
type MatchResult struct {
	// Status is models.BankMatchSuggested or models.BankMatchUnmatched.
	Status string
	// InvoiceID is the suggested receivable, empty when unmatched.
	InvoiceID string
	// InvoiceNumber is that receivable's number. Carried alongside the id so a
	// freshly imported statement can name what it suggests without reading the
	// invoices back.
	InvoiceNumber string
	// Reason is one of the MatchReason* constants, empty when unmatched.
	Reason string
}

// MatchEntry picks a candidate for one parsed entry out of the tenant's open
// items. Debits are never matched: money leaving the account cannot settle a
// receivable, and treating it as one would clear an invoice on an outgoing
// payment.
func MatchEntry(entry *ParsedEntry, openItems []*models.OpenItem) MatchResult {
	if !entry.Amount.IsPositive() {
		return MatchResult{Status: models.BankMatchUnmatched}
	}

	haystack := normaliseReference(strings.Join(
		[]string{entry.RemittanceInfo, entry.EndToEndID, entry.EntryRef}, " ",
	))

	// Pass one: the invoice number in the remittance text. The strongest signal
	// there is, because the customer put it there on purpose.
	var byNumber []*models.OpenItem
	for _, item := range openItems {
		ref := normaliseReference(item.InvoiceNumber)
		if len(ref) < minReferenceLength {
			continue
		}
		if strings.Contains(haystack, ref) {
			byNumber = append(byNumber, item)
		}
	}
	if len(byNumber) == 1 {
		item := byNumber[0]
		reason := MatchReasonNumber
		if amountSettles(entry, item) {
			reason = MatchReasonNumberAndAmount
		}
		return MatchResult{
			Status:        models.BankMatchSuggested,
			InvoiceID:     item.InvoiceID.String(),
			InvoiceNumber: item.InvoiceNumber,
			Reason:        reason,
		}
	}
	if len(byNumber) > 1 {
		// A transfer naming several invoices is a collective payment. Splitting
		// it across them needs a decision this matcher must not take on its own.
		return MatchResult{Status: models.BankMatchUnmatched}
	}

	// Pass two: the amount alone, and only when it is unambiguous.
	var byAmount []*models.OpenItem
	for _, item := range openItems {
		if amountSettles(entry, item) {
			byAmount = append(byAmount, item)
		}
	}
	if len(byAmount) == 1 {
		return MatchResult{
			Status:        models.BankMatchSuggested,
			InvoiceID:     byAmount[0].InvoiceID.String(),
			InvoiceNumber: byAmount[0].InvoiceNumber,
			Reason:        MatchReasonAmount,
		}
	}

	return MatchResult{Status: models.BankMatchUnmatched}
}

// amountSettles reports whether the credit covers the item's open balance
// exactly, in the item's currency. Currency is compared rather than assumed:
// without a stored exchange rate a CHF credit says nothing about a EUR
// receivable, and folding them would suggest a match across currencies.
func amountSettles(entry *ParsedEntry, item *models.OpenItem) bool {
	if !strings.EqualFold(entryCurrency(entry), itemCurrency(item)) {
		return false
	}
	return entry.Amount.Equal(item.OpenAmount)
}

func entryCurrency(entry *ParsedEntry) string {
	if entry.Currency == "" {
		return "EUR"
	}
	return entry.Currency
}

func itemCurrency(item *models.OpenItem) string {
	if item.Currency == "" {
		return "EUR"
	}
	return item.Currency
}

// normaliseReference reduces a string to upper-case letters and digits.
// Remittance text passes through several systems on its way here, each free to
// drop a hyphen, insert a line break or lower-case the whole line; comparing the
// stripped forms is what makes "RE-2026-0001" match "re 2026 0001".
func normaliseReference(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}
