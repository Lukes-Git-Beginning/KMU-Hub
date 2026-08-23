package invoice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// fractionalLineItems: four lines of 1.5 x 33.33 EUR. Unrounded these sum to
// 199.98, rounded per line they sum to 200.00 — the exact divergence the e-invoice
// generator's BR-CO-10 check (which always rounds per line) used to absorb through
// the scaled totals tolerance.
func fractionalLineItems() []models.LineItem {
	items := make([]models.LineItem, 4)
	for i := range items {
		items[i] = models.LineItem{
			ID:          uuid.NewString(),
			Position:    i + 1,
			Description: "Beratung",
			Quantity:    decimal.RequireFromString("1.5"),
			UnitPrice:   decimal.RequireFromString("33.33"),
			TaxRate:     decimal.NewFromInt(19),
		}
	}
	return items
}

// assertLineTotalsAddUp is the invariant the write path owes the PDF and the
// XRechnung: every stored line net has at most two decimals, and the stored lines
// add up to the stored subtotal exactly.
func assertLineTotalsAddUp(t *testing.T, itemsJSON []byte, subtotal decimal.Decimal) {
	t.Helper()
	items, err := unmarshalLineItems(itemsJSON)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	sum := decimal.Zero
	for _, it := range items {
		require.GreaterOrEqualf(t, it.LineTotal.Exponent(), int32(-2),
			"stored line total %s has more than 2 decimals", it.LineTotal)
		sum = sum.Add(it.LineTotal)
	}
	require.Truef(t, sum.Equal(subtotal),
		"sum of stored line totals %s != stored subtotal %s", sum, subtotal)
}

func TestService_Create_StoresRoundedLineTotals(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	input := testCreateInput(tenantID, uuid.New())
	input.LineItems = fractionalLineItems()

	inv, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, "200.00", inv.Subtotal.StringFixed(2))
	assertLineTotalsAddUp(t, repo.invoices[inv.ID].LineItems, inv.Subtotal)
}

func TestService_Update_StoresRoundedLineTotals(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	items := fractionalLineItems()
	updated, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{LineItems: items})
	require.NoError(t, err)

	require.Equal(t, "200.00", updated.Subtotal.StringFixed(2))
	assertLineTotalsAddUp(t, repo.invoices[inv.ID].LineItems, updated.Subtotal)
}
