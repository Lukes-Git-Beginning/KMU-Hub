package gdpr

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestMoneyAndPercentOrEmpty_GermanFormat(t *testing.T) {
	t.Parallel()
	v, pct := 12345.5, 12.5
	assert.Equal(t, "12.345,50 EUR", moneyOrEmpty(&v))
	assert.Equal(t, "12,50 %", percentOrEmpty(&pct))
	assert.Empty(t, moneyOrEmpty(nil))
	assert.Empty(t, percentOrEmpty(nil))
}

func TestFinanceAndDealsModule_GermanMoneyFormat_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "DSAR Money Format Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	agentID := seedDSARUser(t, pool, tenantID, "Mona", "Money", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	seed := func(table string, cols map[string]any) uuid.UUID {
		id := testutil.SeedRow(t, pool, table, cols)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, table, id) })
		return id
	}
	contactID := seed("contacts", map[string]any{
		"tenant_id": tenantID, "first_name": "Mona", "last_name": "Betrag",
		"email": fmt.Sprintf("mona.%s@betrag.invalid", uuid.New()), "created_by": agentID,
	})
	today := "2026-10-30"
	seed("finance_invoices", map[string]any{
		"tenant_id": tenantID, "contact_id": contactID, "invoice_number": "RE-MONEY-" + uuid.NewString()[:8],
		"customer_email": "m@x.invalid", "invoice_date": today, "due_date": today, "gross_total": 1234.5, "created_by": agentID,
	})
	stage := seed("pipeline_stages", map[string]any{"tenant_id": tenantID, "name": "Angebot"})
	seed("deals", map[string]any{
		"tenant_id": tenantID, "name": "Grossauftrag", "stage_id": stage, "contact_id": contactID,
		"value": 98765.4, "currency": "CHF", "created_by": agentID,
	})

	ctx := testutil.WithSystemCtx(context.Background())
	inv, err := financeModule(ctx, pool, tenantID, contactID)
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "1.234,50 EUR", recordMaps(*inv)[0]["Betrag"])

	deals, err := dealsModule(ctx, pool, tenantID, contactID)
	require.NoError(t, err)
	require.NotNil(t, deals)
	assert.Equal(t, "98.765,40 CHF", recordMaps(*deals)[0]["Wert"])
}
