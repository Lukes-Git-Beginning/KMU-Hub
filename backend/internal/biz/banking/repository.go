package banking

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository errors.
var (
	// ErrStatementNotFound is returned when no statement carries the given id.
	ErrStatementNotFound = errors.New("banking: bank statement not found")
	// ErrTransactionNotFound is returned when no transaction carries the given id.
	ErrTransactionNotFound = errors.New("banking: bank transaction not found")
	// ErrAlreadyReconciled is returned when a transaction has already been
	// booked or set aside — the caller must reverse that before deciding anew.
	ErrAlreadyReconciled = errors.New("banking: transaction is already reconciled")
	// ErrNotACredit is returned when a debit is to be booked against a
	// receivable. Money leaving the account cannot settle an invoice.
	ErrNotACredit = errors.New("banking: only a credit can settle a receivable")
)

// Repository persists imported statements and their transactions. Every method
// takes the tenant explicitly: RLS is the second lock, not the only one, so a
// read running under a system context still returns one tenant's rows.
type Repository interface {
	// GetStatementByHash resolves an already-imported file. Returns
	// ErrStatementNotFound when the file is new.
	GetStatementByHash(ctx context.Context, tenantID uuid.UUID, hash string) (*models.BankStatement, error)
	// CreateStatement writes the statement and its transactions in one
	// transaction. A partial import would leave a statement claiming entries it
	// does not have.
	CreateStatement(ctx context.Context, stmt *models.BankStatement, txs []*models.BankTransaction) error
	GetStatement(ctx context.Context, tenantID, id uuid.UUID) (*models.BankStatement, error)
	ListStatements(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BankStatement, int, error)

	GetTransaction(ctx context.Context, tenantID, id uuid.UUID) (*models.BankTransaction, error)
	ListTransactions(ctx context.Context, tenantID uuid.UUID, filter models.BankTransactionFilter) ([]*models.BankTransaction, int, error)
	// ListTransactionsByStatement returns every transaction of one statement,
	// oldest first. Used to answer an import with what it produced.
	ListTransactionsByStatement(ctx context.Context, tenantID, statementID uuid.UUID) ([]*models.BankTransaction, error)
	// UpdateTransactionMatch writes the reconciliation state. It only touches
	// the match columns; what the bank reported stays as imported.
	UpdateTransactionMatch(ctx context.Context, tx *models.BankTransaction) error
}

// OpenItemReader supplies the receivables a credit can be matched against. It is
// satisfied by the invoice repository, so the open-items projection (gross total
// minus recorded payments) lives in exactly one place.
type OpenItemReader interface {
	ListOpenItems(ctx context.Context, tenantID uuid.UUID, filter models.OpenItemFilter) ([]*models.OpenItem, int, error)
}

// PaymentRecorder books a confirmed match as a payment against the invoice.
// Satisfied by *payment.Service, so the invoice status transition and the GoBD
// trail keep running through the one path that owns them — this package never
// writes finance_payments itself.
type PaymentRecorder interface {
	Record(ctx context.Context, input payment.RecordInput) (*models.Payment, error)
}
