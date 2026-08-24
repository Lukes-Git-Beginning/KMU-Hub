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
	// ErrAccountNotFound is returned when no bank account carries the given id
	// within the tenant.
	ErrAccountNotFound = errors.New("banking: bank account not found")
	// ErrAccountExists is returned when the tenant already holds an account
	// with that IBAN.
	ErrAccountExists = errors.New("banking: an account with this IBAN already exists")
	// ErrStatementHashConflict is returned when CreateStatement collides with the
	// unique constraint on (tenant_id, content_hash) — two near-simultaneous
	// imports of the same file both passed the check-then-write in Service.Import
	// and only one could win the insert. The caller re-reads the winner's row
	// instead of surfacing a raw duplicate-key error.
	ErrStatementHashConflict = errors.New("banking: a statement with this content hash already exists")
	// ErrInvalidIBAN is returned when the given IBAN fails the ISO 7064 check
	// or is not a shape this system knows.
	ErrInvalidIBAN = errors.New("banking: invalid IBAN")
	// ErrInvalidAccount is returned when a required field of an account is
	// missing or malformed.
	ErrInvalidAccount = errors.New("banking: invalid bank account")
	// ErrInvoiceNotFound is returned when an invoice picked by its number does
	// not exist within the tenant.
	ErrInvoiceNotFound = errors.New("banking: invoice not found")
	// ErrNothingToReject is returned when a transaction carries no suggestion to
	// discard — a booked or set-aside entry has to be reversed first.
	ErrNothingToReject = errors.New("banking: transaction has no suggested match to reject")
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
	// FindInvoiceIDByNumber resolves an invoice an operator picked by number.
	// Returns ErrInvoiceNotFound when the tenant holds no such invoice.
	FindInvoiceIDByNumber(ctx context.Context, tenantID uuid.UUID, number string) (uuid.UUID, error)

	// Bank accounts (Migration 000258). The master data the statements above
	// attach to. Balance and LastSync are read from the newest statement of the
	// account's IBAN, so the two never disagree about what was imported.

	// CreateAccount writes a new account. Returns ErrAccountExists when the
	// tenant already holds that IBAN -- decided by the unique constraint, not by
	// a preceding lookup, because two concurrent creates would race past one.
	CreateAccount(ctx context.Context, acc *models.BankAccount) error
	// GetAccount resolves one account. Returns ErrAccountNotFound when the id
	// does not belong to the tenant.
	GetAccount(ctx context.Context, tenantID, id uuid.UUID) (*models.BankAccount, error)
	// ListAccounts returns the tenant's accounts ordered by bank name. There is
	// no pagination: a KMU holds a handful of accounts and the frontend renders
	// them all as cards.
	ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*models.BankAccount, error)
	// UpdateAccount writes the mutable master-data fields and the connected
	// flag. Returns ErrAccountNotFound when the row is not the tenant's, and
	// ErrAccountExists when the new IBAN collides.
	UpdateAccount(ctx context.Context, acc *models.BankAccount) error
	// DeleteAccount removes an account. Imported statements are unaffected:
	// they record what a bank reported and outlive the master-data row.
	DeleteAccount(ctx context.Context, tenantID, id uuid.UUID) error
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
