package invoice

// Finance transactions (Transaktionen): a consolidated read of money that
// already moved -- every recorded customer payment and every approved
// expense, merged into one tenant-wide ledger. No new table: both sides
// already exist (finance_payments, finance_expenses); this is a read-time
// union like ListDocumentChains next door.
//
// IDs are prefixed ("pay-"/"exp-") so a caller can tell which underlying
// record an entry names -- deleting one means something different for each:
// undoing a payment (and possibly reverting the invoice's paid status) versus
// withdrawing an approved expense (refused; an approved expense is a decision
// already made, see expense.Service.Delete).

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// financeTransactionIncomeCategory is the fixed category for payment-derived
// entries. finance_payments carries no category of its own -- every income
// movement here is customer revenue.
const financeTransactionIncomeCategory = "Umsatzerlöse"

// ListTransactions returns the tenant's consolidated payment ledger.
func (r *PostgresRepository) ListTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.FinanceTransaction, error) {
	income, err := r.incomeTransactions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load income transactions: %w", err)
	}
	expenses, err := r.expenseTransactions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load expense transactions: %w", err)
	}

	out := make([]*models.FinanceTransaction, 0, len(income)+len(expenses))
	out = append(out, income...)
	out = append(out, expenses...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.After(out[j].Date) })
	return out, nil
}

// incomeTransactions is every recorded payment, described by the invoice it
// paid down since finance_payments carries no description of its own.
func (r *PostgresRepository) incomeTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.FinanceTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.amount::text, p.payment_date, i.id, i.invoice_number, i.customer_name
		FROM finance_payments p
		JOIN finance_invoices i ON i.id = p.invoice_id AND i.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1
		ORDER BY p.payment_date DESC, p.id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.FinanceTransaction
	for rows.Next() {
		var (
			id                          uuid.UUID
			amountStr                   string
			date                        time.Time
			invoiceID                   uuid.UUID
			invoiceNumber, customerName string
		)
		if scanErr := rows.Scan(&id, &amountStr, &date, &invoiceID, &invoiceNumber, &customerName); scanErr != nil {
			return nil, scanErr
		}
		amount, parseErr := decimal.NewFromString(amountStr)
		if parseErr != nil {
			return nil, fmt.Errorf("parse payment amount: %w", parseErr)
		}
		out = append(out, &models.FinanceTransaction{
			ID:          "pay-" + id.String(),
			Type:        "income",
			Description: fmt.Sprintf("Rechnung %s %s", invoiceNumber, customerName),
			Amount:      amount,
			Date:        date,
			Category:    financeTransactionIncomeCategory,
			Status:      "completed",
			Reference:   invoiceNumber,
			InvoiceID:   &invoiceID,
		})
	}
	return out, rows.Err()
}

// expenseTransactions is every approved expense -- pending and rejected ones
// never moved money and do not belong in a payment ledger.
func (r *PostgresRepository) expenseTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.FinanceTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, description, amount::text, expense_date, category
		FROM finance_expenses
		WHERE tenant_id = $1 AND status = 'approved'
		ORDER BY expense_date DESC, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.FinanceTransaction
	for rows.Next() {
		var (
			id          uuid.UUID
			description string
			amountStr   string
			date        time.Time
			category    string
		)
		if scanErr := rows.Scan(&id, &description, &amountStr, &date, &category); scanErr != nil {
			return nil, scanErr
		}
		amount, parseErr := decimal.NewFromString(amountStr)
		if parseErr != nil {
			return nil, fmt.Errorf("parse expense amount: %w", parseErr)
		}
		out = append(out, &models.FinanceTransaction{
			ID:          "exp-" + id.String(),
			Type:        "expense",
			Description: description,
			Amount:      amount,
			Date:        date,
			Category:    category,
			Status:      "completed",
		})
	}
	return out, rows.Err()
}
