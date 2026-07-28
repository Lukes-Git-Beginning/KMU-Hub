package dunning

// Open items (Offene Posten) — the receivables list that feeds the dunning run.
// It sits in this package because it is the same domain seen from the other end:
// dunning escalates what this list reports as overdue, and every row carries the
// level it has already reached.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// OpenItemsPage is one page of receivables plus the tenant-wide summary. The
// summary always covers every open item, never just the page.
type OpenItemsPage struct {
	Items   []*models.OpenItem
	Total   int
	Summary *models.OpenItemsSummary
}

// ListOpenItemsInput selects and pages the receivables list.
type ListOpenItemsInput struct {
	// Bucket restricts the page to one aging bucket key (empty = all).
	Bucket string
	// OverdueOnly drops items that are not past due yet.
	OverdueOnly bool
	Limit       int
	Offset      int
	// AsOf overrides the reference date for aging. Zero means today.
	AsOf time.Time
}

// ListOpenItems returns the tenant's unpaid receivables, most overdue first,
// with the aging summary per currency.
func (s *Service) ListOpenItems(
	ctx context.Context,
	tenantID uuid.UUID,
	input ListOpenItemsInput,
) (*OpenItemsPage, error) {
	if input.Bucket != "" && models.AgingBucketIndexOf(input.Bucket) < 0 {
		return nil, fmt.Errorf("%w: %q", models.ErrUnknownAgingBucket, input.Bucket)
	}

	asOf := input.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	// Aging counts whole days, so the reference is the date, not the instant —
	// otherwise the same invoice ages by one day mid-morning depending on when
	// the request happens to arrive.
	asOf = asOf.Truncate(24 * time.Hour)

	items, total, err := s.invReader.ListOpenItems(ctx, tenantID, models.OpenItemFilter{
		AsOf:        asOf,
		Bucket:      input.Bucket,
		OverdueOnly: input.OverdueOnly,
		Limit:       input.Limit,
		Offset:      input.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list open items: %w", err)
	}
	for _, item := range items {
		item.AgingBucket = models.AgingBucketFor(item.DaysOverdue)
	}

	buckets, err := s.invReader.SummarizeOpenItems(ctx, tenantID, asOf)
	if err != nil {
		return nil, fmt.Errorf("summarize open items: %w", err)
	}

	summary := foldOpenItemSummary(buckets)

	slog.DebugContext(ctx, "listed open items",
		"tenant_id", tenantID,
		"page_items", len(items),
		"total", total,
		"currencies", len(summary.Totals),
	)

	return &OpenItemsPage{Items: items, Total: total, Summary: summary}, nil
}

// foldOpenItemSummary turns the repository's (currency, bucket) groups into the
// per-currency totals the receivables view shows. Everything but the "current"
// bucket counts as overdue — that is the only place the distinction is made, so
// the average days figure cannot silently include items that are not due yet.
func foldOpenItemSummary(buckets []*models.OpenItemBucketTotal) *models.OpenItemsSummary {
	summary := &models.OpenItemsSummary{
		Totals:  make([]*models.OpenItemCurrencyTotal, 0, len(buckets)),
		Buckets: make([]*models.OpenItemBucketTotal, 0, len(buckets)),
	}

	byCurrency := make(map[string]*models.OpenItemCurrencyTotal)
	overdueDays := make(map[string]int)

	for _, bucket := range buckets {
		if bucket == nil {
			continue
		}
		bucket.Bucket = models.AgingBucketKeyAt(bucket.BucketIndex)
		summary.Buckets = append(summary.Buckets, bucket)

		total, ok := byCurrency[bucket.Currency]
		if !ok {
			total = &models.OpenItemCurrencyTotal{
				Currency:      bucket.Currency,
				OpenAmount:    decimal.Zero,
				OverdueAmount: decimal.Zero,
			}
			byCurrency[bucket.Currency] = total
			summary.Totals = append(summary.Totals, total)
		}

		total.OpenAmount = total.OpenAmount.Add(bucket.Amount)
		total.OpenCount += bucket.Count
		if bucket.Bucket != models.AgingBucketCurrent {
			total.OverdueAmount = total.OverdueAmount.Add(bucket.Amount)
			total.OverdueCount += bucket.Count
			overdueDays[bucket.Currency] += bucket.DaysOverdueSum
		}
	}

	for currency, total := range byCurrency {
		if total.OverdueCount > 0 {
			total.AvgDaysOverdue = overdueDays[currency] / total.OverdueCount
		}
	}

	return summary
}
