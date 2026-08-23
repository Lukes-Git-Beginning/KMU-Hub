package quote

// postgres_repository_converted_invoice_db_test.go exercises the
// converted_invoice_number join added to GetByID and List
// (feat-quote-converted-invoice-number-on-read): the newest non-cancelled
// invoice a quote was converted into, joined via LATERAL against
// finance_invoices.source_quote_id.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_GetByID_PopulatesConvertedInvoiceNumber(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Converted Invoice Number Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Converted Invoice Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	now := time.Now().UTC()
	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "sent", "invoice_number": "RE-2026-CONV-001",
		"customer_name": "Converted Invoice Test GmbH",
		"invoice_date":  now, "due_date": now.AddDate(0, 0, 14),
		"source_quote_id": quoteID, "created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	got, err := repo.GetByID(ctx, tenantID, quoteID)
	require.NoError(t, err)
	require.NotNil(t, got.ConvertedInvoiceNumber, "GetByID must populate ConvertedInvoiceNumber for a live conversion")
	assert.Equal(t, "RE-2026-CONV-001", *got.ConvertedInvoiceNumber)

	list, total, err := repo.List(ctx, tenantID, ListFilter{Limit: 20})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].ConvertedInvoiceNumber, "List must populate ConvertedInvoiceNumber like GetByID")
	assert.Equal(t, "RE-2026-CONV-001", *list[0].ConvertedInvoiceNumber)
}

// TestPostgresRepository_GetByID_CancelledConversionLeavesFieldEmpty proves a
// stornoed conversion does not count: the quote may be invoiced again, so the
// FE's "already converted" banner and the disabled convert-button must not
// key off a cancelled invoice's number.
func TestPostgresRepository_GetByID_CancelledConversionLeavesFieldEmpty(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Cancelled Conversion Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	quoteID := testutil.SeedRow(t, pool, "finance_quotes", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "accepted", "customer_name": "Cancelled Conversion Test GmbH",
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_quotes", quoteID) })

	now := time.Now().UTC()
	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"status": "cancelled", "invoice_number": "RE-2026-CANC-001",
		"customer_name": "Cancelled Conversion Test GmbH",
		"invoice_date":  now, "due_date": now.AddDate(0, 0, 14),
		"source_quote_id": quoteID, "created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	got, err := repo.GetByID(ctx, tenantID, quoteID)
	require.NoError(t, err)
	assert.Nil(t, got.ConvertedInvoiceNumber, "a cancelled conversion must leave ConvertedInvoiceNumber empty so the quote can be invoiced again")
}
