package banking

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

func openItem(number, amount string) *models.OpenItem {
	return &models.OpenItem{
		InvoiceID:     uuid.New(),
		InvoiceNumber: number,
		Currency:      "EUR",
		OpenAmount:    decimal.RequireFromString(amount),
	}
}

func creditEntry(amount, remittance string) *ParsedEntry {
	return &ParsedEntry{
		Amount:         decimal.RequireFromString(amount),
		Currency:       "EUR",
		RemittanceInfo: remittance,
	}
}

func TestMatchEntryByNumberAndAmount(t *testing.T) {
	target := openItem("RE-2026-0001", "119.00")
	items := []*models.OpenItem{openItem("RE-2026-0002", "250.00"), target}

	got := MatchEntry(creditEntry("119.00", "Zahlung Rechnung RE-2026-0001"), items)

	if got.Status != models.BankMatchSuggested {
		t.Fatalf("status = %q, want suggested", got.Status)
	}
	if got.InvoiceID != target.InvoiceID.String() {
		t.Errorf("invoice = %s, want %s", got.InvoiceID, target.InvoiceID)
	}
	if got.Reason != MatchReasonNumberAndAmount {
		t.Errorf("reason = %q, want %q", got.Reason, MatchReasonNumberAndAmount)
	}
}

func TestMatchEntryNumberSurvivesReformatting(t *testing.T) {
	// The remittance text passes through several systems, each free to drop the
	// hyphens, insert line breaks or lower-case the line.
	target := openItem("RE-2026-0001", "119.00")
	got := MatchEntry(creditEntry("119.00", "re 2026\n0001 danke"), []*models.OpenItem{target})

	if got.InvoiceID != target.InvoiceID.String() {
		t.Fatalf("invoice = %q, want the reformatted number to still match", got.InvoiceID)
	}
}

func TestMatchEntryPartialPaymentKeepsNumberReason(t *testing.T) {
	// The number is there, the amount is not the full balance: still the right
	// invoice, but the reason must say the amount did not settle it.
	target := openItem("RE-2026-0001", "119.00")
	got := MatchEntry(creditEntry("60.00", "RE-2026-0001 Teilzahlung"), []*models.OpenItem{target})

	if got.Status != models.BankMatchSuggested || got.Reason != MatchReasonNumber {
		t.Fatalf("got %+v, want a suggestion with reason %q", got, MatchReasonNumber)
	}
}

func TestMatchEntryByAmountOnlyWhenUnique(t *testing.T) {
	unique := []*models.OpenItem{openItem("RE-2026-0001", "119.00"), openItem("RE-2026-0002", "250.00")}
	if got := MatchEntry(creditEntry("250.00", "Ueberweisung"), unique); got.Reason != MatchReasonAmount {
		t.Fatalf("got %+v, want a suggestion by amount", got)
	}

	// Two invoices over the same amount: guessing between them would clear the
	// wrong one half the time.
	ambiguous := []*models.OpenItem{openItem("RE-2026-0001", "119.00"), openItem("RE-2026-0002", "119.00")}
	if got := MatchEntry(creditEntry("119.00", "Ueberweisung"), ambiguous); got.Status != models.BankMatchUnmatched {
		t.Fatalf("got %+v, want unmatched on an ambiguous amount", got)
	}
}

func TestMatchEntryCollectivePaymentIsNotSplit(t *testing.T) {
	// One transfer naming two invoices. Splitting it is a decision the matcher
	// must not take on its own.
	items := []*models.OpenItem{openItem("RE-2026-0001", "119.00"), openItem("RE-2026-0002", "250.00")}
	got := MatchEntry(creditEntry("369.00", "RE-2026-0001 und RE-2026-0002"), items)

	if got.Status != models.BankMatchUnmatched {
		t.Fatalf("got %+v, want unmatched", got)
	}
}

func TestMatchEntryIgnoresDebits(t *testing.T) {
	// Money leaving the account cannot settle a receivable — not even when the
	// remittance text names one.
	items := []*models.OpenItem{openItem("RE-2026-0001", "119.00")}
	got := MatchEntry(&ParsedEntry{
		Amount:         decimal.RequireFromString("-119.00"),
		Currency:       "EUR",
		RemittanceInfo: "RE-2026-0001",
	}, items)

	if got.Status != models.BankMatchUnmatched {
		t.Fatalf("got %+v, want unmatched for a debit", got)
	}
}

func TestMatchEntryDoesNotCrossCurrencies(t *testing.T) {
	// Without a stored exchange rate a CHF credit says nothing about a EUR
	// receivable of the same figure.
	item := openItem("RE-2026-0001", "119.00")
	entry := creditEntry("119.00", "Ueberweisung")
	entry.Currency = "CHF"

	if got := MatchEntry(entry, []*models.OpenItem{item}); got.Status != models.BankMatchUnmatched {
		t.Fatalf("got %+v, want unmatched across currencies", got)
	}
}

func TestMatchEntryEmptyCurrencyDefaultsToEUR(t *testing.T) {
	// The parsed entry names no currency (some MT940 segments omit it); it must
	// default to EUR and still match an item that spells EUR out explicitly.
	// The remittance text carries no invoice number, so this only succeeds
	// through the amount-only pass — which is the one that needs the default.
	item := &models.OpenItem{
		InvoiceID:     uuid.New(),
		InvoiceNumber: "RE-2026-0001",
		Currency:      "EUR",
		OpenAmount:    decimal.RequireFromString("119.00"),
	}
	entry := &ParsedEntry{
		Amount:         decimal.RequireFromString("119.00"),
		RemittanceInfo: "Ueberweisung",
	}

	got := MatchEntry(entry, []*models.OpenItem{item})
	if got.Status != models.BankMatchSuggested || got.Reason != MatchReasonAmount {
		t.Fatalf("got %+v, want a suggestion by amount when the entry's blank currency defaults to EUR", got)
	}
}

func TestMatchEntryIgnoresShortInvoiceNumbers(t *testing.T) {
	// "7" would appear in almost any remittance text; such a number is only
	// reachable through the amount rule.
	item := openItem("7", "42.00")
	got := MatchEntry(creditEntry("99.00", "Verwendungszweck 7 Stueck"), []*models.OpenItem{item})

	if got.Status != models.BankMatchUnmatched {
		t.Fatalf("got %+v, want unmatched on a too-short reference", got)
	}
}
