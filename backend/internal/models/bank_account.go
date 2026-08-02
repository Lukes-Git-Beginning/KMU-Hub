package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BankAccount is one account a tenant holds (Migration 000258). It is master
// data: the statements imported by internal/biz/banking attach to it through
// the IBAN.
//
// Balance and LastSync are derived on read, not stored -- see the migration
// header. They describe the newest statement imported for this IBAN, so an
// account without any import reports a zero balance and no sync, which is what
// this system honestly knows about it.
type BankAccount struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	BankName string    `json:"bank_name"`
	// IBAN is canonical: upper-case, no separators. Format it for display with
	// dachfmt.FormatIBAN rather than storing a grouped copy.
	IBAN     string `json:"iban"`
	BIC      string `json:"bic"`
	Currency string `json:"currency"`
	// Connected marks the account as linked in the (simulated) PSD2 flow. The
	// real bank connection is P5; until then this is a flag a user sets.
	Connected   bool       `json:"connected"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Balance is the closing balance of the newest imported statement, zero
	// when none was imported. Read-only.
	Balance decimal.Decimal `json:"balance"`
	// LastSync is the date of that statement, nil when none was imported.
	// Read-only.
	LastSync *time.Time `json:"last_sync,omitempty"`
}
