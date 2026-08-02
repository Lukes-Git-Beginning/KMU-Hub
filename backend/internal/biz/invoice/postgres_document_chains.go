package invoice

// Document chains (Belegkette): the lifecycle of a customer document from
// quote through invoice to payment, dunning, or credit note.
//
// No new table. The links already exist as foreign keys between the
// document tables (finance_invoices.source_quote_id, finance_payments/
// finance_dunning_records.invoice_id, finance_credit_notes.original_invoice_id)
// — a chain is a read-time assembly across five tenant-scoped queries, not a
// tracked entity of its own.
//
// A chain is anchored on an invoice, prefixed with its source quote if one
// converted into it. A quote that has not (yet) become an invoice is its own
// one-node chain — the "Innovate Labs" case in the desktop mock, a rejected
// quote nothing else will ever happen to.

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

type chainPaymentRow struct {
	Amount    decimal.Decimal
	Date      time.Time
	Reference string
}

type chainDunningRow struct {
	Level     int
	Status    string
	SentAt    *time.Time
	CreatedAt time.Time
}

type chainCreditNoteRow struct {
	Number    string
	Status    string
	Amount    decimal.Decimal
	CreatedAt time.Time
}

// ListDocumentChains returns every document chain of the tenant.
func (r *PostgresRepository) ListDocumentChains(ctx context.Context, tenantID uuid.UUID) ([]*models.DocumentChain, error) {
	payments, err := r.chainPayments(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load chain payments: %w", err)
	}
	dunnings, err := r.chainDunnings(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load chain dunnings: %w", err)
	}
	creditNotes, err := r.chainCreditNotes(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load chain credit notes: %w", err)
	}

	chains := make([]*models.DocumentChain, 0)

	invoiceRows, err := r.pool.Query(ctx, `
		SELECT i.id, i.invoice_number, i.status, i.customer_name, i.gross_total::text, i.currency, i.invoice_date,
		       q.id, q.quote_number, q.gross_total::text, q.created_at
		FROM finance_invoices i
		LEFT JOIN finance_quotes q ON q.id = i.source_quote_id AND q.tenant_id = i.tenant_id
		WHERE i.tenant_id = $1
		ORDER BY i.invoice_date, i.id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list chain invoices: %w", err)
	}
	for invoiceRows.Next() {
		var (
			invID                          uuid.UUID
			invNumber, invStatus, customer string
			invGrossStr, currency          string
			invDate                        time.Time
			quoteID                        *uuid.UUID
			quoteNumber, quoteGrossStr     *string
			quoteCreatedAt                 *time.Time
		)
		if err := invoiceRows.Scan(&invID, &invNumber, &invStatus, &customer, &invGrossStr, &currency, &invDate,
			&quoteID, &quoteNumber, &quoteGrossStr, &quoteCreatedAt); err != nil {
			invoiceRows.Close()
			return nil, fmt.Errorf("scan chain invoice: %w", err)
		}
		invGross, err := decimal.NewFromString(invGrossStr)
		if err != nil {
			invoiceRows.Close()
			return nil, fmt.Errorf("parse invoice gross total: %w", err)
		}

		var nodes []models.ChainNode
		if quoteID != nil {
			quoteGross, err := decimal.NewFromString(*quoteGrossStr)
			if err != nil {
				invoiceRows.Close()
				return nil, fmt.Errorf("parse quote gross total: %w", err)
			}
			nodes = append(nodes, models.ChainNode{
				Type: models.ChainNodeQuote, Number: *quoteNumber, Date: quoteCreatedAt,
				Amount: quoteGross,
				// A quote that produced this invoice already did its job, whatever
				// its own status column says (converting sets it accepted, but the
				// chain step is complete either way).
				Status: models.ChainNodeCompleted,
			})
		}
		invDateCopy := invDate
		nodes = append(nodes, models.ChainNode{
			Type: models.ChainNodeInvoice, Number: invNumber, Date: &invDateCopy,
			Amount: invGross, Status: invoiceNodeStatus(invStatus),
		})

		paid := decimal.Zero
		for _, p := range payments[invID] {
			paid = paid.Add(p.Amount)
			date := p.Date
			nodes = append(nodes, models.ChainNode{
				Type: models.ChainNodePayment, Number: paymentNodeNumber(p.Reference, invNumber), Date: &date,
				Amount: p.Amount, Status: models.ChainNodeCompleted,
			})
		}
		credited := decimal.Zero
		for _, c := range creditNotes[invID] {
			status := models.ChainNodePending
			if c.Status == models.CreditNoteStatusSent {
				status = models.ChainNodeCompleted
				credited = credited.Add(c.Amount)
			}
			date := c.CreatedAt
			nodes = append(nodes, models.ChainNode{
				Type: models.ChainNodeCreditNote, Number: c.Number, Date: &date,
				Amount: c.Amount, Status: status,
			})
		}
		for _, d := range dunnings[invID] {
			date := d.CreatedAt
			if d.SentAt != nil {
				date = *d.SentAt
			}
			// Dunning records carry no number sequence of their own (only
			// invoices/quotes/credit notes do) — the invoice number plus level
			// is a real, derived reference, not a fabricated one.
			nodes = append(nodes, models.ChainNode{
				Type: models.ChainNodeDunning, Number: fmt.Sprintf("%s/M%d", invNumber, d.Level), Date: &date,
				// The dunning chases the invoice's full amount, not the (much
				// smaller) dunning fee — there is no stored "outstanding as of
				// this notice" snapshot to show instead.
				Amount: invGross, Status: dunningNodeStatus(d.Status),
			})
		}

		sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Date.Before(*nodes[j].Date) })

		remaining := invGross.Sub(paid).Sub(credited)
		cancelled := invStatus == models.InvoiceStatusCancelled
		if !cancelled && remaining.IsPositive() {
			nodes = append(nodes, models.ChainNode{
				Type: models.ChainNodePayment, Amount: remaining, Status: models.ChainNodePending,
			})
		}

		chains = append(chains, &models.DocumentChain{
			ID: invID, Customer: customer, Currency: currency, TotalValue: invGross,
			IsComplete: cancelled || !remaining.IsPositive(),
			Nodes:      nodes,
		})
	}
	if err := invoiceRows.Err(); err != nil {
		invoiceRows.Close()
		return nil, fmt.Errorf("iterate chain invoices: %w", err)
	}
	invoiceRows.Close()

	quoteRows, err := r.pool.Query(ctx, `
		SELECT q.id, q.quote_number, q.status, q.customer_name, q.gross_total::text, q.currency, q.created_at
		FROM finance_quotes q
		WHERE q.tenant_id = $1
		  AND NOT EXISTS (
		      SELECT 1 FROM finance_invoices i
		      WHERE i.tenant_id = q.tenant_id AND i.source_quote_id = q.id
		  )
		ORDER BY q.created_at, q.id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list standalone chain quotes: %w", err)
	}
	defer quoteRows.Close()
	for quoteRows.Next() {
		var (
			quoteID                  uuid.UUID
			number, status, customer string
			grossStr, currency       string
			createdAt                time.Time
		)
		if err := quoteRows.Scan(&quoteID, &number, &status, &customer, &grossStr, &currency, &createdAt); err != nil {
			return nil, fmt.Errorf("scan standalone chain quote: %w", err)
		}
		gross, err := decimal.NewFromString(grossStr)
		if err != nil {
			return nil, fmt.Errorf("parse quote gross total: %w", err)
		}
		nodeStatus := quoteNodeStatus(status)
		chains = append(chains, &models.DocumentChain{
			ID: quoteID, Customer: customer, Currency: currency, TotalValue: gross,
			IsComplete: nodeStatus == models.ChainNodeCancelled,
			Nodes: []models.ChainNode{
				{Type: models.ChainNodeQuote, Number: number, Date: &createdAt, Amount: gross, Status: nodeStatus},
			},
		})
	}
	if err := quoteRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate standalone chain quotes: %w", err)
	}

	return chains, nil
}

func (r *PostgresRepository) chainPayments(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID][]chainPaymentRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT invoice_id, amount::text, payment_date, reference
		FROM finance_payments
		WHERE tenant_id = $1
		ORDER BY payment_date, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]chainPaymentRow)
	for rows.Next() {
		var (
			invoiceID            uuid.UUID
			amountStr, reference string
			date                 time.Time
		)
		if err := rows.Scan(&invoiceID, &amountStr, &date, &reference); err != nil {
			return nil, err
		}
		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("parse payment amount: %w", err)
		}
		out[invoiceID] = append(out[invoiceID], chainPaymentRow{Amount: amount, Date: date, Reference: reference})
	}
	return out, rows.Err()
}

func (r *PostgresRepository) chainDunnings(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID][]chainDunningRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT invoice_id, level, status, sent_at, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1
		ORDER BY COALESCE(sent_at, created_at), id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]chainDunningRow)
	for rows.Next() {
		var (
			invoiceID uuid.UUID
			level     int
			status    string
			sentAt    *time.Time
			createdAt time.Time
		)
		if err := rows.Scan(&invoiceID, &level, &status, &sentAt, &createdAt); err != nil {
			return nil, err
		}
		out[invoiceID] = append(out[invoiceID], chainDunningRow{Level: level, Status: status, SentAt: sentAt, CreatedAt: createdAt})
	}
	return out, rows.Err()
}

func (r *PostgresRepository) chainCreditNotes(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID][]chainCreditNoteRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT original_invoice_id, credit_note_number, status, gross_total::text, created_at
		FROM finance_credit_notes
		WHERE tenant_id = $1
		ORDER BY created_at, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]chainCreditNoteRow)
	for rows.Next() {
		var (
			invoiceID      uuid.UUID
			number, status string
			amountStr      string
			createdAt      time.Time
		)
		if err := rows.Scan(&invoiceID, &number, &status, &amountStr, &createdAt); err != nil {
			return nil, err
		}
		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("parse credit note amount: %w", err)
		}
		out[invoiceID] = append(out[invoiceID], chainCreditNoteRow{Number: number, Status: status, Amount: amount, CreatedAt: createdAt})
	}
	return out, rows.Err()
}

func invoiceNodeStatus(status string) string {
	switch status {
	case models.InvoiceStatusPaid:
		return models.ChainNodeCompleted
	case models.InvoiceStatusCancelled:
		return models.ChainNodeCancelled
	case models.InvoiceStatusOverdue:
		return models.ChainNodeOverdue
	default: // draft, sent
		return models.ChainNodeActive
	}
}

func quoteNodeStatus(status string) string {
	switch status {
	case models.QuoteStatusRejected, models.QuoteStatusExpired:
		return models.ChainNodeCancelled
	default: // draft, sent, accepted (accepted-without-invoice-yet is still in flight)
		return models.ChainNodeActive
	}
}

func dunningNodeStatus(status string) string {
	switch status {
	case models.DunningStatusPaid:
		return models.ChainNodeCompleted
	case models.DunningStatusSent:
		return models.ChainNodeActive
	default: // draft
		return models.ChainNodePending
	}
}

// paymentNodeNumber falls back to the invoice number: finance_payments.reference
// is free text and often empty, and a chain node needs something to show.
func paymentNodeNumber(reference, invoiceNumber string) string {
	if reference != "" {
		return reference
	}
	return invoiceNumber
}
