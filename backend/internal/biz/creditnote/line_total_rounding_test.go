package creditnote

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestService_Create_StoresRoundedLineTotals: four lines of 1.5 x 33.33 EUR sum to
// 199.98 unrounded and 200.00 rounded per line. The credit note is what the tax
// office sees reversed against the original invoice, so its stored lines have to
// add up to its stored subtotal exactly. See internal/biz/invoice for the
// counterpart on the invoice side.
func TestService_Create_StoresRoundedLineTotals(t *testing.T) {
	svc, repo, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	invReader.AddInvoice(inv)

	items := make([]models.LineItem, 4)
	for i := range items {
		items[i] = models.LineItem{
			ID:          uuid.NewString(),
			Position:    i + 1,
			Description: "Gutschrift Beratung",
			Quantity:    decimal.RequireFromString("1.5"),
			UnitPrice:   decimal.RequireFromString("33.33"),
			TaxRate:     decimal.NewFromInt(19),
		}
	}

	cn, err := svc.Create(context.Background(), CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         items,
		TaxMode:           models.TaxModeStandard,
		Reason:            "Teilstorno",
		UserID:            uuid.New(),
	})
	require.NoError(t, err)
	require.Equal(t, "200.00", cn.Subtotal.StringFixed(2))

	var stored []models.LineItem
	require.NoError(t, json.Unmarshal(repo.creditNotes[cn.ID].LineItems, &stored))
	require.Len(t, stored, 4)

	sum := decimal.Zero
	for _, it := range stored {
		require.GreaterOrEqualf(t, it.LineTotal.Exponent(), int32(-2),
			"stored line total %s has more than 2 decimals", it.LineTotal)
		sum = sum.Add(it.LineTotal)
	}
	require.Truef(t, sum.Equal(cn.Subtotal),
		"sum of stored line totals %s != stored subtotal %s", sum, cn.Subtotal)
}
