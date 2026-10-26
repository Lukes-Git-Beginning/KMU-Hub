package payment

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestMigration000363_BackfillsPaymentCurrencyFromInvoice runs the real up file
// of migration 000363 against session-local shadow tables: temporary tables are
// resolved before public ones, so the unqualified names in the migration hit the
// fixtures and the shared schema stays untouched (the tx is rolled back). It
// proves the backfill copies the invoice currency per payment, matches on the
// tenant as well as the invoice id, and that NOT NULL then refuses a row the
// backfill could not resolve instead of letting it through as NULL.
func TestMigration000363_BackfillsPaymentCurrencyFromInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	upSQL, err := os.ReadFile("../../../migrations/000363_finance_payments_currency.up.sql")
	require.NoError(t, err)

	tenantA, tenantB := uuid.New(), uuid.New()
	invEUR, invCHF := uuid.New(), uuid.New()

	withShadowTables := func(t *testing.T, fn func(ctx context.Context, tx pgx.Tx)) {
		t.Helper()
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx) //nolint:errcheck
		_, err = tx.Exec(ctx, `
			CREATE TEMP TABLE finance_invoices (id UUID, tenant_id UUID, currency CHAR(3) NOT NULL) ON COMMIT DROP;
			CREATE TEMP TABLE finance_payments (id UUID, tenant_id UUID, invoice_id UUID) ON COMMIT DROP;`)
		require.NoError(t, err)
		_, err = tx.Exec(ctx,
			`INSERT INTO finance_invoices (id, tenant_id, currency) VALUES ($1, $2, 'EUR'), ($3, $2, 'CHF')`,
			invEUR, tenantA, invCHF)
		require.NoError(t, err)
		fn(ctx, tx)
	}

	t.Run("copies the invoice currency per payment", func(t *testing.T) {
		withShadowTables(t, func(ctx context.Context, tx pgx.Tx) {
			payEUR, payCHF1, payCHF2 := uuid.New(), uuid.New(), uuid.New()
			_, err := tx.Exec(ctx,
				`INSERT INTO finance_payments (id, tenant_id, invoice_id) VALUES ($1, $4, $5), ($2, $4, $6), ($3, $4, $6)`,
				payEUR, payCHF1, payCHF2, tenantA, invEUR, invCHF)
			require.NoError(t, err)

			_, err = tx.Exec(ctx, string(upSQL))
			require.NoError(t, err)

			rows, err := tx.Query(ctx, `SELECT id, currency FROM finance_payments`)
			require.NoError(t, err)
			got := map[uuid.UUID]string{}
			for rows.Next() {
				var id uuid.UUID
				var cur string
				require.NoError(t, rows.Scan(&id, &cur))
				got[id] = cur
			}
			require.NoError(t, rows.Err())
			assert.Equal(t, map[uuid.UUID]string{payEUR: "EUR", payCHF1: "CHF", payCHF2: "CHF"}, got)

			var nullable string
			require.NoError(t, tx.QueryRow(ctx, `
				SELECT is_nullable FROM information_schema.columns
				WHERE table_schema = pg_my_temp_schema()::regnamespace::text
				  AND table_name = 'finance_payments' AND column_name = 'currency'`).Scan(&nullable))
			assert.Equal(t, "NO", nullable)
		})
	})

	t.Run("a payment whose invoice sits in another tenant blocks the migration", func(t *testing.T) {
		withShadowTables(t, func(ctx context.Context, tx pgx.Tx) {
			_, err := tx.Exec(ctx,
				`INSERT INTO finance_payments (id, tenant_id, invoice_id) VALUES ($1, $2, $3)`,
				uuid.New(), tenantB, invEUR)
			require.NoError(t, err)

			_, err = tx.Exec(ctx, string(upSQL))
			require.Error(t, err, "an unresolved currency must fail SET NOT NULL, not be guessed")
			assert.Contains(t, err.Error(), "currency")
		})
	})
}
