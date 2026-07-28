package invoice

// Open items (Offene Posten) — the receivables view of finance_invoices.
//
// The desktop client used to compute this list itself, from one page of
// GET /finance/invoices. That was wrong twice over: it only ever saw the first
// page, so every total was a page total dressed up as a tenant total, and it
// used the invoice gross amount, so an invoice with a partial payment was
// reported as fully outstanding. Both facts live in the database — the payment
// rows and the full result set — so the aggregation belongs here.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// openItemsBase is the common projection: sent or overdue invoices whose gross
// total is not yet covered by recorded payments. Payments are folded per invoice
// before the join so a second payment on the same invoice cannot duplicate the
// invoice row. Both the invoice and the payment side are tenant-scoped: RLS
// already enforces it, but a read path that relies on RLS alone silently returns
// nothing the day it runs under a system context.
//
// $1 = tenant_id, $2 = as-of date.
const openItemsBase = `
	SELECT i.id,
	       i.invoice_number,
	       i.status,
	       i.customer_name,
	       i.customer_email,
	       i.currency,
	       i.gross_total::text                                    AS gross_total,
	       COALESCE(p.paid, 0)::text                              AS paid_amount,
	       (i.gross_total - COALESCE(p.paid, 0))::text            AS open_amount,
	       i.invoice_date,
	       i.due_date,
	       ($2::date - i.due_date)::int                           AS days_overdue,
	       i.contact_id
	FROM finance_invoices i
	LEFT JOIN (
	    SELECT invoice_id, SUM(amount) AS paid
	    FROM finance_payments
	    WHERE tenant_id = $1
	    GROUP BY invoice_id
	) p ON p.invoice_id = i.id
	WHERE i.tenant_id = $1
	  AND i.status IN ('sent', 'overdue')
	  AND i.gross_total - COALESCE(p.paid, 0) > 0`

// ListOpenItems returns one page of unpaid receivables, most overdue first, plus
// the number of open items matching the filter.
func (r *PostgresRepository) ListOpenItems(
	ctx context.Context,
	tenantID uuid.UUID,
	filter models.OpenItemFilter,
) ([]*models.OpenItem, int, error) {
	asOf := filter.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	args := []any{tenantID, asOf}

	var conditions []string
	if filter.OverdueOnly {
		conditions = append(conditions, "days_overdue > 0")
	}
	if filter.Bucket != "" {
		cond, bucketArgs, err := bucketCondition(filter.Bucket, len(args))
		if err != nil {
			return nil, 0, err
		}
		conditions = append(conditions, cond)
		args = append(args, bucketArgs...)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) o%s", openItemsBase, where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count open items: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := max(filter.Offset, 0)

	// The highest dunning level reached on the invoice, so the list shows how far
	// the escalation has already run. LATERAL keeps it one row per invoice even
	// with several notices on record.
	query := fmt.Sprintf(`
		SELECT o.*,
		       COALESCE(d.level, 0)  AS dunning_level,
		       COALESCE(d.status, '') AS dunning_status,
		       d.sent_at
		FROM (%s) o
		LEFT JOIN LATERAL (
		    SELECT level, status, sent_at
		    FROM finance_dunning_records
		    WHERE tenant_id = $1 AND invoice_id = o.id
		    ORDER BY level DESC, created_at DESC
		    LIMIT 1
		) d ON TRUE%s
		ORDER BY o.days_overdue DESC, o.id ASC
		LIMIT $%d OFFSET $%d`,
		openItemsBase, where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list open items: %w", err)
	}
	defer rows.Close()

	items := make([]*models.OpenItem, 0)
	for rows.Next() {
		var (
			item                       models.OpenItem
			grossStr, paidStr, openStr string
			dunningStatus              string
			dunningLevel               int
			sentAt                     *time.Time
		)
		if err := rows.Scan(
			&item.InvoiceID, &item.InvoiceNumber, &item.Status,
			&item.CustomerName, &item.CustomerEmail, &item.Currency,
			&grossStr, &paidStr, &openStr,
			&item.InvoiceDate, &item.DueDate, &item.DaysOverdue, &item.ContactID,
			&dunningLevel, &dunningStatus, &sentAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan open item: %w", err)
		}
		item.GrossTotal, _ = decimal.NewFromString(grossStr)
		item.PaidAmount, _ = decimal.NewFromString(paidStr)
		item.OpenAmount, _ = decimal.NewFromString(openStr)
		item.DunningLevel = dunningLevel
		item.DunningStatus = dunningStatus
		item.LastDunnedAt = sentAt
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate open items: %w", err)
	}

	return items, total, nil
}

// SummarizeOpenItems aggregates every open item of the tenant by currency and
// aging bucket. It deliberately ignores the page filter: the totals a receivables
// view shows are tenant totals, and computing them from a page is how the client
// side got them wrong.
func (r *PostgresRepository) SummarizeOpenItems(
	ctx context.Context,
	tenantID uuid.UUID,
	asOf time.Time,
) ([]*models.OpenItemBucketTotal, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	args := []any{tenantID, asOf}
	caseExpr, boundArgs := bucketIndexCase("days_overdue", len(args))
	args = append(args, boundArgs...)

	query := fmt.Sprintf(`
		SELECT currency,
		       %s AS bucket_index,
		       COUNT(*)                          AS item_count,
		       SUM(open_amount::numeric)::text   AS bucket_amount,
		       SUM(GREATEST(days_overdue, 0))    AS days_overdue_sum
		FROM (%s) o
		GROUP BY currency, bucket_index
		ORDER BY currency, bucket_index`, caseExpr, openItemsBase)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("summarize open items: %w", err)
	}
	defer rows.Close()

	totals := make([]*models.OpenItemBucketTotal, 0)
	for rows.Next() {
		var (
			bucket    models.OpenItemBucketTotal
			amountStr string
		)
		if err := rows.Scan(
			&bucket.Currency, &bucket.BucketIndex,
			&bucket.Count, &amountStr, &bucket.DaysOverdueSum,
		); err != nil {
			return nil, fmt.Errorf("scan open item summary: %w", err)
		}
		bucket.Amount, _ = decimal.NewFromString(amountStr)
		totals = append(totals, &bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate open item summary: %w", err)
	}

	return totals, nil
}

// bucketIndexCase builds the CASE expression that maps a day count to a bucket
// index. The day bounds come from models.AgingBucketUpperDays and travel as
// query parameters, so the SQL classification and the Go one cannot drift apart:
// there is exactly one place where a bucket boundary is written down.
// startArg is the number of parameters already bound.
func bucketIndexCase(daysExpr string, startArg int) (string, []any) {
	bounds := models.AgingBucketUpperDays()
	var sb strings.Builder
	args := make([]any, 0, len(bounds))
	sb.WriteString("CASE")
	for i, bound := range bounds {
		fmt.Fprintf(&sb, " WHEN %s <= $%d THEN %d", daysExpr, startArg+i+1, i)
		args = append(args, bound)
	}
	fmt.Fprintf(&sb, " ELSE %d END", len(bounds))
	return sb.String(), args
}

// bucketCondition restricts a query to a single aging bucket. An unknown key is
// an error rather than an empty result, so a typo in a query parameter does not
// read as "no open items".
func bucketCondition(bucketKey string, startArg int) (string, []any, error) {
	bounds := models.AgingBucketUpperDays()
	index := models.AgingBucketIndexOf(bucketKey)
	if index < 0 {
		return "", nil, fmt.Errorf("%w: %q", models.ErrUnknownAgingBucket, bucketKey)
	}

	var (
		parts []string
		args  []any
	)
	if index > 0 {
		parts = append(parts, fmt.Sprintf("days_overdue > $%d", startArg+len(args)+1))
		args = append(args, bounds[index-1])
	}
	if index < len(bounds) {
		parts = append(parts, fmt.Sprintf("days_overdue <= $%d", startArg+len(args)+1))
		args = append(args, bounds[index])
	}
	return "(" + strings.Join(parts, " AND ") + ")", args, nil
}
