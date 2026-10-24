package gdpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Finance records beyond invoices — Art. 15 GDPR.
//
// Read-only. These tables are bound by retention duties (§ 147 AO, § 257 HGB):
// disclosure is owed, erasure is not, so there is no retention counterpart.
//
// Contact branch: only finance_invoices carries a contact_id. The other
// documents are matched heuristically — customer_email equality (same rule as
// formSubmissionsModule) for quotes, credit notes and subscriptions; exact
// case-insensitive name equality with the contact's full name for incoming
// invoices and bank bookings, which hold no e-mail at all. Never by the
// contact's company: the company's supplier invoices and bookings are not the
// contact's data and would disclose the employer's business to them. Name hits
// are flagged "(Namensabgleich)" in the module title: a namesake in the same
// tenant would show up, the reviewer has to confirm. Credit notes and dunning
// records are additionally reachable through the contact's invoices.
//
// IBANs are masked in SQL so the full number never leaves the database.
// ---------------------------------------------------------------------------

// contactFinanceModules collects the finance modules for one contact; empty
// ones are skipped.
func contactFinanceModules(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, c contactRow) ([]DSARModule, error) {
	email := strings.ToLower(strings.TrimSpace(c.email))
	fullName := strings.ToLower(strings.TrimSpace(joinName(c.first, c.last)))

	var out []DSARModule
	add := func(mod *DSARModule, err error) error {
		if err != nil {
			return fmt.Errorf("dsar: finance: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
		return nil
	}

	if email != "" {
		cols := []string{"Angebotsnummer", "Status", "Kunde", "Adresse", "Gültig bis", "Betrag", "Erstellt"}
		if err := add(userRowsModule(ctx, pool, "quotes", "Angebote", cols,
			`SELECT quote_number, status, customer_name, customer_address, valid_until,
			        to_char(gross_total, 'FM999999999990.00') || ' ' || currency, created_at
			 FROM finance_quotes
			 WHERE tenant_id = $1 AND LOWER(customer_email) = $2
			 ORDER BY created_at DESC
			 LIMIT $3`, []any{tenantID, email, dsarMaxRows}, valuesRecord(cols))); err != nil {
			return nil, err
		}

		cols = []string{"Abo", "Status", "Kunde", "Adresse", "Intervall", "Start", "Ende", "Erstellt"}
		if err := add(userRowsModule(ctx, pool, "recurring invoices", "Abos (wiederkehrende Rechnungen)", cols,
			`SELECT title, status, customer_name, customer_address, recurrence_interval, start_date, end_date, created_at
			 FROM finance_recurring_invoices
			 WHERE tenant_id = $1 AND LOWER(customer_email) = $2
			 ORDER BY created_at DESC
			 LIMIT $3`, []any{tenantID, email, dsarMaxRows}, valuesRecord(cols))); err != nil {
			return nil, err
		}
	}

	cols := []string{"Gutschriftsnummer", "Status", "Kunde", "Adresse", "Grund", "Betrag", "Erstellt"}
	if err := add(userRowsModule(ctx, pool, "credit notes", "Gutschriften", cols,
		`SELECT cn.credit_note_number, cn.status, cn.customer_name, cn.customer_address, cn.reason,
		        to_char(cn.gross_total, 'FM999999999990.00') || ' ' || cn.currency, cn.created_at
		 FROM finance_credit_notes cn
		 WHERE cn.tenant_id = $1
		   AND (($2 <> '' AND LOWER(cn.customer_email) = $2)
		        OR EXISTS (SELECT 1 FROM finance_invoices i
		                   WHERE i.id = cn.original_invoice_id AND i.tenant_id = cn.tenant_id AND i.contact_id = $3))
		 ORDER BY cn.created_at DESC
		 LIMIT $4`, []any{tenantID, email, c.id, dsarMaxRows}, valuesRecord(cols))); err != nil {
		return nil, err
	}

	cols = []string{"Rechnungsnummer", "Mahnstufe", "Status", "Gebühr", "Zinsen", "Versendet", "Erstellt"}
	if err := add(userRowsModule(ctx, pool, "dunning records", "Mahnungen", cols,
		`SELECT i.invoice_number, d.level, d.status, d.fee, d.interest, d.sent_at, d.created_at
		 FROM finance_dunning_records d
		 JOIN finance_invoices i ON i.id = d.invoice_id AND i.tenant_id = d.tenant_id
		 WHERE d.tenant_id = $1 AND i.contact_id = $2
		 ORDER BY d.created_at DESC
		 LIMIT $3`, []any{tenantID, c.id, dsarMaxRows}, valuesRecord(cols))); err != nil {
		return nil, err
	}

	if fullName != "" {
		cols = []string{"Rechnungsnummer", "Lieferant", "Adresse", "Datum", "Status", "Betrag"}
		if err := add(userRowsModule(ctx, pool, "incoming invoices", "Eingangsrechnungen (Namensabgleich)", cols,
			`SELECT invoice_number, supplier_name, supplier_address, invoice_date, status,
			        to_char(gross_total, 'FM999999999990.00') || ' ' || currency
			 FROM finance_incoming_invoices
			 WHERE tenant_id = $1 AND LOWER(BTRIM(supplier_name)) = $2
			 ORDER BY invoice_date DESC
			 LIMIT $3`, []any{tenantID, fullName, dsarMaxRows}, valuesRecord(cols))); err != nil {
			return nil, err
		}

		cols = []string{"Wertstellung", "Gegenpartei", "IBAN", "Verwendungszweck", "Betrag"}
		if err := add(userRowsModule(ctx, pool, "bank transactions", "Bankumsätze (Namensabgleich)", cols,
			`SELECT value_date, counterparty_name,
			        CASE WHEN length(replace(counterparty_iban, ' ', '')) > 4
			             THEN '****' || right(replace(counterparty_iban, ' ', ''), 4) ELSE '' END,
			        remittance_info, to_char(amount, 'FM999999999990.00') || ' ' || currency
			 FROM finance_bank_transactions
			 WHERE tenant_id = $1 AND LOWER(BTRIM(counterparty_name)) = $2
			 ORDER BY value_date DESC
			 LIMIT $3`, []any{tenantID, fullName, dsarMaxRows}, valuesRecord(cols))); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// expensesModule discloses expense claims the user submitted or decided on.
// lean: EUR hardcoded, finance_expenses has no currency column (DACH/EUR-only
// product); revisit if multi-currency expenses land.
func expensesModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Rolle", "Beschreibung", "Lieferant", "Kategorie", "Datum", "Betrag", "Status", "Beleg", "Entschieden am"}
	return userRowsModule(ctx, pool, "expenses", "Spesen", columns,
		`SELECT CASE WHEN submitted_by = $2 AND decided_by = $2 THEN 'Einreicher und Entscheider'
		             WHEN submitted_by = $2 THEN 'Einreicher' ELSE 'Entscheider' END,
		        description, supplier, category, expense_date,
		        to_char(amount, 'FM999999999990.00') || ' EUR', status, COALESCE(receipt_name, ''), decided_at
		 FROM finance_expenses
		 WHERE tenant_id = $1 AND (submitted_by = $2 OR decided_by = $2)
		 ORDER BY expense_date DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows}, valuesRecord(columns))
}
