package quote

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// fractionalLineItems: four lines of 1.5 x 33.33 EUR — unrounded 199.98, rounded
// per line 200.00. See the counterpart test in internal/biz/invoice.
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

func assertLineTotalsAddUp(t *testing.T, itemsJSON []byte, subtotal decimal.Decimal) {
	t.Helper()
	var items []models.LineItem
	require.NoError(t, json.Unmarshal(itemsJSON, &items))
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
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()
	input := sampleCreateInput(tenantID)
	input.LineItems = fractionalLineItems()

	q, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, "200.00", q.Subtotal.StringFixed(2))
	assertLineTotalsAddUp(t, repo.quotes[q.ID].LineItems, q.Subtotal)
}

func TestService_Update_StoresRoundedLineTotals(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()
	q := seedDraftQuote(repo, tenantID)

	updated, err := svc.Update(context.Background(), tenantID, q.ID, UpdateInput{LineItems: fractionalLineItems()})
	require.NoError(t, err)

	require.Equal(t, "200.00", updated.Subtotal.StringFixed(2))
	assertLineTotalsAddUp(t, repo.quotes[q.ID].LineItems, updated.Subtotal)
}
