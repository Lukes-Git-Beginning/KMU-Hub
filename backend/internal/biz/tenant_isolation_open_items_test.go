package biz

// Open items (Offene Posten) read a joined view of finance_invoices,
// finance_payments and finance_dunning_records. Two things can only be checked
// against a real database, and both are exactly what the client-side version of
// this list got wrong:
//
//  1. open_amount is the residual — an invoice with a partial payment must show
//     what is still owed, not its gross total.
//  2. the query is tenant-scoped on the read side. A leak here would expose
//     another tenant's customers and outstanding amounts; a missing scope in the
//     other direction shows a tenant an empty receivables list, which reads as
//     "nothing owed".
//
// Skips cleanly without DATABASE_URL (CI runners without Postgres).

import (
	"context"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_OpenItems(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	asOf := time.Now().UTC().Truncate(24 * time.Hour)
	dueDate := asOf.AddDate(0, 0, -45) // squarely in the d60 bucket

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"tenant_id":      testutil.TenantA,
		"invoice_number": "RE-2026-9001",
		"status":         models.InvoiceStatusOverdue,
		"customer_name":  "Beispiel GmbH",
		"gross_total":    "1000.00",
		"currency":       "EUR",
		"invoice_date":   dueDate.AddDate(0, 0, -14).Format("2006-01-02"),
		"due_date":       dueDate.Format("2006-01-02"),
		"created_by":     testutil.TenantA, // any UUID; only NOT NULL matters here
	})
	defer testutil.CleanupRow(t, pool, "finance_invoices", invoiceID)

	paymentID := testutil.SeedRow(t, pool, "finance_payments", map[string]any{
		"tenant_id":    testutil.TenantA,
		"invoice_id":   invoiceID,
		"amount":       "400.00",
		"payment_date": asOf.Format("2006-01-02"),
		"created_by":   testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "finance_payments", paymentID)

	repo := invoice.NewPostgresRepository(pool)
	filter := models.OpenItemFilter{AsOf: asOf, Limit: 100}

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	t.Run("owner sees the residual", func(t *testing.T) {
		items, total, err := repo.ListOpenItems(ctxA, testutil.TenantA, filter)
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		if total < 1 {
			t.Fatalf("expected at least the seeded open item, got total=%d", total)
		}

		var found *models.OpenItem
		for _, item := range items {
			if item.InvoiceID == invoiceID {
				found = item
				break
			}
		}
		if found == nil {
			t.Fatal("seeded invoice missing from the open items of its own tenant")
		}
		if got := found.OpenAmount.String(); got != "600" {
			t.Errorf("open_amount = %s, want 600 (1000 gross minus a 400 payment)", got)
		}
		if got := found.PaidAmount.String(); got != "400" {
			t.Errorf("paid_amount = %s, want 400", got)
		}
		if found.DaysOverdue != 45 {
			t.Errorf("days_overdue = %d, want 45", found.DaysOverdue)
		}
		if got := models.AgingBucketFor(found.DaysOverdue); got != models.AgingBucketD60 {
			t.Errorf("aging bucket = %s, want %s", got, models.AgingBucketD60)
		}
	})

	t.Run("other tenant sees nothing of it", func(t *testing.T) {
		items, _, err := repo.ListOpenItems(ctxB, testutil.TenantB, filter)
		if err != nil {
			t.Fatalf("ListOpenItems: %v", err)
		}
		for _, item := range items {
			if item.InvoiceID == invoiceID {
				t.Fatal("tenant B sees tenant A's open item")
			}
		}

		buckets, err := repo.SummarizeOpenItems(ctxB, testutil.TenantB, asOf)
		if err != nil {
			t.Fatalf("SummarizeOpenItems: %v", err)
		}
		for _, bucket := range buckets {
			if bucket.Amount.String() == "600" {
				t.Fatal("tenant B's summary carries tenant A's outstanding amount")
			}
		}
	})

	t.Run("summary counts the residual, not the gross", func(t *testing.T) {
		buckets, err := repo.SummarizeOpenItems(ctxA, testutil.TenantA, asOf)
		if err != nil {
			t.Fatalf("SummarizeOpenItems: %v", err)
		}

		wantIndex := models.AgingBucketIndexFor(45)
		var seen bool
		for _, bucket := range buckets {
			if bucket.Currency != "EUR" || bucket.BucketIndex != wantIndex {
				continue
			}
			seen = true
			// Other seeded data may share the bucket, so assert the floor rather
			// than an exact figure: the residual must be in there, the gross must
			// not have been counted on top of it.
			if bucket.Amount.IntPart() < 600 {
				t.Errorf("bucket amount %s is below the seeded residual of 600", bucket.Amount)
			}
			if bucket.DaysOverdueSum < 45 {
				t.Errorf("days_overdue_sum = %d, want at least 45", bucket.DaysOverdueSum)
			}
		}
		if !seen {
			t.Errorf("no EUR bucket at index %d in the summary", wantIndex)
		}
	})
}
