package invoice

import (
	"errors"
	"testing"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestBucketCondition covers the four known aging bucket keys: the boundary
// args passed to SQL must match models.AgingBucketUpperDays exactly, since
// that is the single source of truth the Go classification and the SQL
// aggregation both derive from.
func TestBucketCondition(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		wantLower *int
		wantUpper *int
	}{
		{name: "current", key: models.AgingBucketCurrent, wantUpper: intPtr(0)},
		{name: "d30", key: models.AgingBucketD30, wantLower: intPtr(0), wantUpper: intPtr(30)},
		{name: "d60", key: models.AgingBucketD60, wantLower: intPtr(30), wantUpper: intPtr(60)},
		{name: "d60plus", key: models.AgingBucketD60Plus, wantLower: intPtr(60)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cond, args, err := bucketCondition(tc.key, 1)
			if err != nil {
				t.Fatalf("bucketCondition(%q): %v", tc.key, err)
			}
			if cond == "" {
				t.Error("condition string is empty")
			}
			wantArgs := 0
			if tc.wantLower != nil {
				wantArgs++
			}
			if tc.wantUpper != nil {
				wantArgs++
			}
			if len(args) != wantArgs {
				t.Fatalf("args count: got %d, want %d (cond=%q)", len(args), wantArgs, cond)
			}
			i := 0
			if tc.wantLower != nil {
				if args[i] != *tc.wantLower {
					t.Errorf("lower bound arg: got %v, want %d", args[i], *tc.wantLower)
				}
				i++
			}
			if tc.wantUpper != nil {
				if args[i] != *tc.wantUpper {
					t.Errorf("upper bound arg: got %v, want %d", args[i], *tc.wantUpper)
				}
			}
		})
	}
}

// TestBucketCondition_UnknownKeyReturnsError verifies that a typo in the
// bucket key is a hard error, not a silently-empty result.
func TestBucketCondition_UnknownKeyReturnsError(t *testing.T) {
	_, _, err := bucketCondition("not-a-bucket", 1)
	if err == nil {
		t.Fatal("expected error for unknown bucket key, got nil")
	}
	if !errors.Is(err, models.ErrUnknownAgingBucket) {
		t.Errorf("error: got %v, want wrapping ErrUnknownAgingBucket", err)
	}
}

// TestBucketIndexCase verifies the CASE expression carries the same bounds as
// AgingBucketUpperDays, offset by the already-bound arg count.
func TestBucketIndexCase(t *testing.T) {
	caseExpr, args := bucketIndexCase("days_overdue", 2)
	bounds := models.AgingBucketUpperDays()
	if len(args) != len(bounds) {
		t.Fatalf("args count: got %d, want %d", len(args), len(bounds))
	}
	for i, bound := range bounds {
		if args[i] != bound {
			t.Errorf("arg[%d]: got %v, want %d", i, args[i], bound)
		}
	}
	if caseExpr == "" {
		t.Error("case expression is empty")
	}
}

func TestInvoiceNodeStatus(t *testing.T) {
	cases := map[string]string{
		models.InvoiceStatusPaid:      models.ChainNodeCompleted,
		models.InvoiceStatusCancelled: models.ChainNodeCancelled,
		models.InvoiceStatusOverdue:   models.ChainNodeOverdue,
		models.InvoiceStatusDraft:     models.ChainNodeActive,
		models.InvoiceStatusSent:      models.ChainNodeActive,
	}
	for status, want := range cases {
		if got := invoiceNodeStatus(status); got != want {
			t.Errorf("invoiceNodeStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestQuoteNodeStatus(t *testing.T) {
	cases := map[string]string{
		models.QuoteStatusRejected: models.ChainNodeCancelled,
		models.QuoteStatusExpired:  models.ChainNodeCancelled,
		models.QuoteStatusDraft:    models.ChainNodeActive,
		models.QuoteStatusSent:     models.ChainNodeActive,
		models.QuoteStatusAccepted: models.ChainNodeActive,
	}
	for status, want := range cases {
		if got := quoteNodeStatus(status); got != want {
			t.Errorf("quoteNodeStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestDunningNodeStatus(t *testing.T) {
	cases := map[string]string{
		models.DunningStatusPaid:  models.ChainNodeCompleted,
		models.DunningStatusSent:  models.ChainNodeActive,
		models.DunningStatusDraft: models.ChainNodePending,
	}
	for status, want := range cases {
		if got := dunningNodeStatus(status); got != want {
			t.Errorf("dunningNodeStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestPaymentNodeNumber(t *testing.T) {
	if got := paymentNodeNumber("ref-123", "RE-2026-001"); got != "ref-123" {
		t.Errorf("with reference: got %q, want ref-123", got)
	}
	if got := paymentNodeNumber("", "RE-2026-001"); got != "RE-2026-001" {
		t.Errorf("empty reference fallback: got %q, want invoice number", got)
	}
}

func intPtr(i int) *int { return &i }
