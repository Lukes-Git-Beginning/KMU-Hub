package banking

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/models"
)

// openItemScanLimit bounds how many receivables one import matches against.
//
// lean: the whole open-items page is loaded once per import and matched in
// memory — one query instead of one per transaction, and small enough for a
// KMU's receivables. Push the matching into SQL when a tenant runs past this
// many simultaneously open invoices; until then the extra join would be
// complexity without a payer.
const openItemScanLimit = 2000

// Service imports bank statements and reconciles their credits against open
// receivables.
type Service struct {
	repo      Repository
	openItems OpenItemReader
	payments  PaymentRecorder
	logger    *slog.Logger
}

// NewService builds the banking service. openItems and payments may be nil in a
// deployment that only wants the import; the reconcile path then reports the
// feature as unavailable rather than booking a half-wired payment.
func NewService(repo Repository, openItems OpenItemReader, payments PaymentRecorder, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, openItems: openItems, payments: payments, logger: logger}
}

// ImportInput is one uploaded statement file.
type ImportInput struct {
	TenantID uuid.UUID
	Filename string
	Content  []byte
	UserID   uuid.UUID
}

// ImportResult carries the statement and its transactions, plus whether this
// call actually created them.
type ImportResult struct {
	Statement    *models.BankStatement
	Transactions []*models.BankTransaction
	// AlreadyImported is true when the identical file had been imported before.
	// The caller then holds the original statement, not a second copy.
	AlreadyImported bool
}

// Import parses an uploaded statement, stores it with its transactions and
// attaches a match suggestion to every credit.
//
// Importing the same file twice is a no-op that returns the first import. The
// file hash is what makes that true, and it is enforced by a unique constraint
// rather than by the lookup below: two uploads arriving at once would both pass
// a check-then-write.
func (s *Service) Import(ctx context.Context, input ImportInput) (*ImportResult, error) {
	if input.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant is required", ErrMalformed)
	}

	sum := sha256.Sum256(input.Content)
	hash := hex.EncodeToString(sum[:])

	if existing, err := s.repo.GetStatementByHash(ctx, input.TenantID, hash); err == nil {
		txs, listErr := s.repo.ListTransactionsByStatement(ctx, input.TenantID, existing.ID)
		if listErr != nil {
			return nil, fmt.Errorf("list transactions of existing statement: %w", listErr)
		}
		s.logger.InfoContext(ctx, "bank statement already imported",
			"tenant_id", input.TenantID, "statement_id", existing.ID, "filename", input.Filename)
		return &ImportResult{Statement: existing, Transactions: txs, AlreadyImported: true}, nil
	} else if !errors.Is(err, ErrStatementNotFound) {
		return nil, fmt.Errorf("look up statement by hash: %w", err)
	}

	parsed, err := Parse(input.Content)
	if err != nil {
		return nil, err
	}

	stmt := &models.BankStatement{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		Format:           parsed.Format,
		Filename:         input.Filename,
		ContentHash:      hash,
		AccountIBAN:      parsed.AccountIBAN,
		StatementRef:     parsed.StatementRef,
		Currency:         defaultCurrency(parsed.Currency),
		OpeningBalance:   parsed.OpeningBalance,
		ClosingBalance:   parsed.ClosingBalance,
		StatementDate:    parsed.StatementDate,
		TransactionCount: len(parsed.Entries),
		CreatedAt:        time.Now().UTC(),
	}
	if input.UserID != uuid.Nil {
		userID := input.UserID
		stmt.ImportedBy = &userID
	}

	openItems := s.loadOpenItems(ctx, input.TenantID)

	txs := make([]*models.BankTransaction, 0, len(parsed.Entries))
	for i := range parsed.Entries {
		entry := &parsed.Entries[i]
		tx := &models.BankTransaction{
			ID:               uuid.New(),
			TenantID:         input.TenantID,
			StatementID:      stmt.ID,
			EntryRef:         entry.EntryRef,
			EndToEndID:       entry.EndToEndID,
			ValueDate:        entry.ValueDate,
			BookingDate:      entry.BookingDate,
			Amount:           entry.Amount,
			Currency:         defaultCurrency(firstNonEmpty(entry.Currency, stmt.Currency)),
			CounterpartyName: entry.CounterpartyName,
			CounterpartyIBAN: entry.CounterpartyIBAN,
			RemittanceInfo:   entry.RemittanceInfo,
			MatchStatus:      models.BankMatchUnmatched,
			CreatedAt:        stmt.CreatedAt,
		}

		match := MatchEntry(entry, openItems)
		if match.Status == models.BankMatchSuggested {
			invoiceID, parseErr := uuid.Parse(match.InvoiceID)
			if parseErr == nil {
				tx.MatchStatus = models.BankMatchSuggested
				tx.MatchReason = match.Reason
				tx.MatchedInvoiceID = &invoiceID
			}
		}
		txs = append(txs, tx)
	}

	if err := s.repo.CreateStatement(ctx, stmt, txs); err != nil {
		return nil, fmt.Errorf("create statement: %w", err)
	}

	s.logger.InfoContext(ctx, "bank statement imported",
		"tenant_id", input.TenantID, "statement_id", stmt.ID, "format", stmt.Format,
		"entries", len(txs), "suggested", countSuggested(txs))

	return &ImportResult{Statement: stmt, Transactions: txs}, nil
}

// loadOpenItems fetches the receivables to match against. A failure here costs
// suggestions, not the import: the transactions are still worth storing, and the
// user can reconcile them by hand.
func (s *Service) loadOpenItems(ctx context.Context, tenantID uuid.UUID) []*models.OpenItem {
	if s.openItems == nil {
		return nil
	}
	items, _, err := s.openItems.ListOpenItems(ctx, tenantID, models.OpenItemFilter{
		AsOf:  time.Now().UTC(),
		Limit: openItemScanLimit,
	})
	if err != nil {
		s.logger.WarnContext(ctx, "bank import: open items unavailable, importing without match suggestions",
			"tenant_id", tenantID, "error", err)
		return nil
	}
	return items
}

// ListStatements returns one page of the tenant's imports, newest first.
func (s *Service) ListStatements(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BankStatement, int, error) {
	return s.repo.ListStatements(ctx, tenantID, normaliseLimit(limit), max(offset, 0))
}

// GetStatement returns one import with its transactions.
func (s *Service) GetStatement(ctx context.Context, tenantID, id uuid.UUID) (*models.BankStatement, []*models.BankTransaction, error) {
	stmt, err := s.repo.GetStatement(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	txs, err := s.repo.ListTransactionsByStatement(ctx, tenantID, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list transactions: %w", err)
	}
	return stmt, txs, nil
}

// ListTransactions returns one page of imported transactions.
func (s *Service) ListTransactions(ctx context.Context, tenantID uuid.UUID, filter models.BankTransactionFilter) ([]*models.BankTransaction, int, error) {
	filter.Limit = normaliseLimit(filter.Limit)
	filter.Offset = max(filter.Offset, 0)
	return s.repo.ListTransactions(ctx, tenantID, filter)
}

// ReconcileInput confirms one transaction against one invoice.
type ReconcileInput struct {
	TenantID      uuid.UUID
	TransactionID uuid.UUID
	// InvoiceID overrides the suggestion. Zero means "take the suggested one",
	// which is the common case: the user is confirming what was proposed.
	InvoiceID uuid.UUID
	UserID    uuid.UUID
}

// Reconcile books a bank credit as a payment against an invoice.
//
// The payment is recorded through the payment service, so the invoice status
// transition, its GoBD trail and the paid-amount arithmetic keep happening in
// the one place that owns them. Only after that does the transaction get its
// matched state — if the payment fails, nothing here claims it succeeded.
func (s *Service) Reconcile(ctx context.Context, input ReconcileInput) (*models.BankTransaction, error) {
	if s.payments == nil {
		return nil, fmt.Errorf("banking: payment recording is not configured")
	}

	tx, err := s.repo.GetTransaction(ctx, input.TenantID, input.TransactionID)
	if err != nil {
		return nil, err
	}
	if tx.MatchStatus == models.BankMatchMatched || tx.MatchStatus == models.BankMatchIgnored {
		return nil, ErrAlreadyReconciled
	}
	if !tx.IsCredit() {
		return nil, ErrNotACredit
	}

	invoiceID := input.InvoiceID
	if invoiceID == uuid.Nil {
		if tx.MatchedInvoiceID == nil {
			return nil, fmt.Errorf("%w: no invoice given and none suggested", ErrTransactionNotFound)
		}
		invoiceID = *tx.MatchedInvoiceID
	}

	pmt, err := s.payments.Record(ctx, payment.RecordInput{
		TenantID:  input.TenantID,
		InvoiceID: invoiceID,
		Amount:    tx.Amount,
		Date:      tx.ValueDate,
		Method:    "bank_transfer",
		Reference: reconcileReference(tx),
		Notes:     reconcileNotes(tx),
		UserID:    input.UserID,
		// Keyed on the bank transaction: a retried confirmation, or one whose
		// status write was lost, records the same payment once.
		IdempotencyKey: "bank-tx:" + tx.ID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("record payment: %w", err)
	}

	now := time.Now().UTC()
	tx.MatchStatus = models.BankMatchMatched
	tx.MatchedInvoiceID = &invoiceID
	tx.ReconciledAt = &now
	if input.UserID != uuid.Nil {
		userID := input.UserID
		tx.ReconciledBy = &userID
	}
	if pmt != nil {
		paymentID := pmt.ID
		tx.PaymentID = &paymentID
	}
	if tx.MatchReason == "" {
		tx.MatchReason = "manual"
	}

	if err := s.repo.UpdateTransactionMatch(ctx, tx); err != nil {
		// The payment is booked; only the bookkeeping of this queue failed. Say
		// so loudly — a silent failure here would offer the same entry again and
		// the idempotency key is the only thing then stopping a second payment.
		s.logger.ErrorContext(ctx, "bank transaction booked but match state not persisted",
			"tenant_id", input.TenantID, "transaction_id", tx.ID, "invoice_id", invoiceID, "error", err)
		return nil, fmt.Errorf("persist match: %w", err)
	}

	s.logger.InfoContext(ctx, "bank transaction reconciled",
		"tenant_id", input.TenantID, "transaction_id", tx.ID, "invoice_id", invoiceID, "reason", tx.MatchReason)
	return tx, nil
}

// Ignore takes a transaction out of the reconciliation queue without booking it
// — a bank fee, an own transfer, anything that is not a customer payment.
func (s *Service) Ignore(ctx context.Context, tenantID, transactionID, userID uuid.UUID) (*models.BankTransaction, error) {
	tx, err := s.repo.GetTransaction(ctx, tenantID, transactionID)
	if err != nil {
		return nil, err
	}
	if tx.MatchStatus == models.BankMatchMatched {
		// A booked entry carries a payment. Undoing it means deleting that
		// payment, which is the payment service's decision, not this one's.
		return nil, ErrAlreadyReconciled
	}

	now := time.Now().UTC()
	tx.MatchStatus = models.BankMatchIgnored
	tx.MatchedInvoiceID = nil
	tx.MatchReason = "ignored"
	tx.ReconciledAt = &now
	if userID != uuid.Nil {
		id := userID
		tx.ReconciledBy = &id
	}
	if err := s.repo.UpdateTransactionMatch(ctx, tx); err != nil {
		return nil, fmt.Errorf("persist match: %w", err)
	}
	return tx, nil
}

// reconcileReference is what shows up on the payment: the bank's own reference
// where there is one, otherwise the end-to-end id the payer set.
func reconcileReference(tx *models.BankTransaction) string {
	ref := firstNonEmpty(tx.EntryRef, tx.EndToEndID)
	if ref == "" {
		return "bank statement import"
	}
	return truncate(ref, 100) // finance_payments.reference is VARCHAR(100)
}

func reconcileNotes(tx *models.BankTransaction) string {
	parts := make([]string, 0, 2)
	if tx.CounterpartyName != "" {
		parts = append(parts, tx.CounterpartyName)
	}
	if tx.RemittanceInfo != "" {
		parts = append(parts, tx.RemittanceInfo)
	}
	if len(parts) == 0 {
		return ""
	}
	return truncate(joinNonEmpty(parts, " — "), 500)
}

func joinNonEmpty(parts []string, sep string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}

// truncate cuts on a rune boundary so a multi-byte character is never split into
// invalid UTF-8 on its way into the database.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	runes := []rune(s)
	for len(string(runes)) > limit {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func countSuggested(txs []*models.BankTransaction) int {
	n := 0
	for _, tx := range txs {
		if tx.MatchStatus == models.BankMatchSuggested {
			n++
		}
	}
	return n
}

func defaultCurrency(c string) string {
	if c == "" {
		return "EUR"
	}
	return c
}

func normaliseLimit(limit int) int {
	switch {
	case limit <= 0:
		return 50
	case limit > 200:
		return 200
	default:
		return limit
	}
}
