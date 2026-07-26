package dunning

// Open items (Offene Posten). The parts worth testing without a database are the
// ones the desktop client used to get wrong: which items count as overdue, how
// the per-currency totals fold, and that the aging boundaries land where the
// repository's SQL expects them to.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- MockInvoiceReader: open-item half -------------------------------------

func (m *MockInvoiceReader) ListOpenItems(
	_ context.Context,
	_ uuid.UUID,
	filter models.OpenItemFilter,
) ([]*models.OpenItem, int, error) {
	m.lastOpenAsOf = filter.AsOf
	if m.openErr != nil {
		return nil, 0, m.openErr
	}
	return m.openItems, m.openTotal, nil
}

func (m *MockInvoiceReader) SummarizeOpenItems(
	_ context.Context,
	_ uuid.UUID,
	_ time.Time,
) ([]*models.OpenItemBucketTotal, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.openBuckets, nil
}

// --- Aging classification --------------------------------------------------

func TestAgingBucketFor_Boundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		days int
		want string
	}{
		{-30, models.AgingBucketCurrent},
		{-1, models.AgingBucketCurrent},
		{0, models.AgingBucketCurrent}, // due today is not overdue yet
		{1, models.AgingBucketD30},
		{30, models.AgingBucketD30},
		{31, models.AgingBucketD60},
		{60, models.AgingBucketD60},
		{61, models.AgingBucketD60Plus},
		{4000, models.AgingBucketD60Plus},
	}

	for _, tc := range tests {
		if got := models.AgingBucketFor(tc.days); got != tc.want {
			t.Errorf("AgingBucketFor(%d) = %q, want %q", tc.days, got, tc.want)
		}
	}
}

// The repository builds its CASE expression from AgingBucketUpperDays and maps
// the resulting index back through AgingBucketKeyAt. If the two ever disagree,
// the page rows and the summary buckets would classify the same invoice
// differently — this pins them together.
func TestAgingBucketIndex_MatchesKeyOrder(t *testing.T) {
	t.Parallel()

	bounds := models.AgingBucketUpperDays()
	if len(bounds) != 3 {
		t.Fatalf("expected 3 upper bounds, got %d", len(bounds))
	}

	for i, bound := range bounds {
		if idx := models.AgingBucketIndexFor(bound); idx != i {
			t.Errorf("day %d classified as index %d, want %d", bound, idx, i)
		}
		if idx := models.AgingBucketIndexFor(bound + 1); idx != i+1 {
			t.Errorf("day %d classified as index %d, want %d", bound+1, idx, i+1)
		}
		if key := models.AgingBucketKeyAt(i); key != models.AgingBucketFor(bound) {
			t.Errorf("index %d maps to %q, but day %d buckets as %q", i, key, bound, models.AgingBucketFor(bound))
		}
	}

	// Out-of-range indices must not produce an empty label.
	if key := models.AgingBucketKeyAt(99); key != models.AgingBucketD60Plus {
		t.Errorf("AgingBucketKeyAt(99) = %q, want %q", key, models.AgingBucketD60Plus)
	}
	if models.AgingBucketIndexOf("nonsense") != -1 {
		t.Error("AgingBucketIndexOf accepted an unknown bucket key")
	}
}

// --- Summary folding -------------------------------------------------------

func TestListOpenItems_FoldsSummaryPerCurrency(t *testing.T) {
	t.Parallel()

	invReader := NewMockInvoiceReader()
	invReader.openTotal = 4
	invReader.openItems = []*models.OpenItem{
		{InvoiceID: uuid.New(), Currency: "EUR", DaysOverdue: 45, OpenAmount: decimal.NewFromInt(100)},
		{InvoiceID: uuid.New(), Currency: "EUR", DaysOverdue: 0, OpenAmount: decimal.NewFromInt(50)},
	}
	invReader.openBuckets = []*models.OpenItemBucketTotal{
		{Currency: "EUR", BucketIndex: 0, Count: 2, Amount: decimal.NewFromInt(500)},
		{Currency: "EUR", BucketIndex: 1, Count: 3, Amount: decimal.NewFromInt(300), DaysOverdueSum: 45},
		{Currency: "EUR", BucketIndex: 2, Count: 1, Amount: decimal.NewFromInt(200), DaysOverdueSum: 55},
		{Currency: "CHF", BucketIndex: 3, Count: 2, Amount: decimal.NewFromInt(900), DaysOverdueSum: 400},
	}

	svc := NewService(NewMockRepository(), &MockConfigRepository{}, invReader)

	page, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{})
	if err != nil {
		t.Fatalf("ListOpenItems: %v", err)
	}

	if page.Total != 4 {
		t.Errorf("Total = %d, want 4", page.Total)
	}

	// Items get their bucket label from their own day count.
	if page.Items[0].AgingBucket != models.AgingBucketD60 {
		t.Errorf("45 days overdue bucketed as %q, want d60", page.Items[0].AgingBucket)
	}
	if page.Items[1].AgingBucket != models.AgingBucketCurrent {
		t.Errorf("0 days overdue bucketed as %q, want current", page.Items[1].AgingBucket)
	}

	totals := make(map[string]*models.OpenItemCurrencyTotal, len(page.Summary.Totals))
	for _, total := range page.Summary.Totals {
		totals[total.Currency] = total
	}

	eur, ok := totals["EUR"]
	if !ok {
		t.Fatal("no EUR total in summary")
	}
	if got := eur.OpenAmount.String(); got != "1000" {
		t.Errorf("EUR open_amount = %s, want 1000", got)
	}
	if eur.OpenCount != 6 {
		t.Errorf("EUR open_count = %d, want 6", eur.OpenCount)
	}
	// The current bucket must stay out of the overdue figures.
	if got := eur.OverdueAmount.String(); got != "500" {
		t.Errorf("EUR overdue_amount = %s, want 500", got)
	}
	if eur.OverdueCount != 4 {
		t.Errorf("EUR overdue_count = %d, want 4", eur.OverdueCount)
	}
	// (45 + 55) / 4 overdue items.
	if eur.AvgDaysOverdue != 25 {
		t.Errorf("EUR avg_days_overdue = %d, want 25", eur.AvgDaysOverdue)
	}

	chf, ok := totals["CHF"]
	if !ok {
		t.Fatal("no CHF total in summary — currencies must not be folded together")
	}
	if got := chf.OpenAmount.String(); got != "900" {
		t.Errorf("CHF open_amount = %s, want 900", got)
	}
	if chf.AvgDaysOverdue != 200 {
		t.Errorf("CHF avg_days_overdue = %d, want 200", chf.AvgDaysOverdue)
	}

	// Bucket keys are resolved from the repository's indices.
	wantBuckets := []string{
		models.AgingBucketCurrent,
		models.AgingBucketD30,
		models.AgingBucketD60,
		models.AgingBucketD60Plus,
	}
	for i, want := range wantBuckets {
		if page.Summary.Buckets[i].Bucket != want {
			t.Errorf("bucket %d labelled %q, want %q", i, page.Summary.Buckets[i].Bucket, want)
		}
	}
}

func TestListOpenItems_NoOpenItems(t *testing.T) {
	t.Parallel()

	invReader := NewMockInvoiceReader()
	svc := NewService(NewMockRepository(), &MockConfigRepository{}, invReader)

	page, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{})
	if err != nil {
		t.Fatalf("ListOpenItems: %v", err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Errorf("expected an empty page, got total=%d items=%d", page.Total, len(page.Items))
	}
	// Empty means empty slices, not nil — a nil slice serializes to null and the
	// client then has to guard every read.
	if page.Summary.Totals == nil || page.Summary.Buckets == nil {
		t.Error("summary slices must be empty, not nil")
	}
}

func TestListOpenItems_RejectsUnknownBucket(t *testing.T) {
	t.Parallel()

	svc := NewService(NewMockRepository(), &MockConfigRepository{}, NewMockInvoiceReader())

	_, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{Bucket: "d90"})
	if !errors.Is(err, models.ErrUnknownAgingBucket) {
		t.Errorf("expected ErrUnknownAgingBucket, got %v", err)
	}
}

func TestListOpenItems_PropagatesRepoErrors(t *testing.T) {
	t.Parallel()

	listErr := errors.New("list boom")
	invReader := NewMockInvoiceReader()
	invReader.openErr = listErr
	svc := NewService(NewMockRepository(), &MockConfigRepository{}, invReader)
	if _, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{}); !errors.Is(err, listErr) {
		t.Errorf("expected the list error to surface, got %v", err)
	}

	summaryErr := errors.New("summary boom")
	invReader = NewMockInvoiceReader()
	invReader.summaryErr = summaryErr
	svc = NewService(NewMockRepository(), &MockConfigRepository{}, invReader)
	if _, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{}); !errors.Is(err, summaryErr) {
		t.Errorf("expected the summary error to surface, got %v", err)
	}
}

// The aging reference is a date. Passing the request instant through would age
// the same invoice differently depending on the time of day it is fetched.
func TestListOpenItems_AsOfIsTruncatedToADate(t *testing.T) {
	t.Parallel()

	invReader := NewMockInvoiceReader()
	svc := NewService(NewMockRepository(), &MockConfigRepository{}, invReader)

	asOf := time.Date(2026, 7, 26, 17, 43, 12, 0, time.UTC)
	if _, err := svc.ListOpenItems(context.Background(), uuid.New(), ListOpenItemsInput{AsOf: asOf}); err != nil {
		t.Fatalf("ListOpenItems: %v", err)
	}

	if got := invReader.lastOpenAsOf; got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("as-of passed to the repository was %s, want a whole date", got)
	}
}
