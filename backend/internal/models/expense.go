package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Expense statuses. They mirror the frontend's Expense.status union
// (desktop/src/renderer/src/stores/finance.ts) and the CHECK constraint in
// migration 000257 one-for-one -- adding a value means touching all three.
const (
	ExpenseStatusPending  = "pending"
	ExpenseStatusApproved = "approved"
	ExpenseStatusRejected = "rejected"
)

// Expense is a claim for money already spent (Ausgabe), entered by one employee
// and decided by another (Migration 000257).
//
// SubmittedBy and DecidedBy are separate fields rather than one "last actor"
// because the approval rule is about them being different people; collapsing
// them would make the rule unenforceable after the fact.
type Expense struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	// ExpenseDate is the day the money was spent, not a point in time. It is
	// carried as a date-only value across the wire (YYYY-MM-DD).
	ExpenseDate time.Time `json:"expense_date"`
	Category    string    `json:"category"`
	Supplier    string    `json:"supplier"`
	Project     *string   `json:"project,omitempty"`
	// Account is the SKR03/04 Sachkonto assigned during Kontierung; nil until
	// somebody books it.
	Account *string `json:"account,omitempty"`
	// ReceiptName is the filename of the attached document, nil when none is
	// attached. There is no blob: see internal/biz/expense/service.go.
	ReceiptName *string    `json:"receipt_name,omitempty"`
	Status      string     `json:"status"`
	SubmittedBy *uuid.UUID `json:"submitted_by,omitempty"`
	DecidedBy   *uuid.UUID `json:"decided_by,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// HasReceipt reports whether a document filename is attached. The frontend
// renders a separate boolean next to the name, and deriving it here keeps the
// two from disagreeing.
func (e *Expense) HasReceipt() bool {
	return e.ReceiptName != nil && *e.ReceiptName != ""
}
