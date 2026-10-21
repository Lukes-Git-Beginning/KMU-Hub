package gdpr

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSearchByQuery_ContactFinanceBeyondInvoices_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Finance Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Finance Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Agnes", "Agentin", true)
	agentOther := seedDSARUser(t, pool, tenantOther, "Otto", "Fremd", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)
	defer testutil.CleanupRow(t, pool, "users", agentOther)

	var cleanups []func()
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}()
	seed := func(table string, cols map[string]any) uuid.UUID {
		id := testutil.SeedRow(t, pool, table, cols)
		cleanups = append(cleanups, func() { testutil.CleanupRow(t, pool, table, id) })
		return id
	}

	email := fmt.Sprintf("fiona.%s@finanz.invalid", uuid.New())
	company := seed("companies", map[string]any{"tenant_id": tenantOwn, "name": "Finanz AG", "created_by": agentID})
	contactID := seed("contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Fiona", "last_name": "Finanzbuch", "email": email,
		"company_id": company, "created_by": agentID,
	})
	namesakeID := seed("contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Bert", "last_name": "Buchhalter",
		"email": fmt.Sprintf("bert.%s@finanz.invalid", uuid.New()), "created_by": agentID,
	})
	foreignEmail := fmt.Sprintf("fiona.%s@finanz.invalid", uuid.New())
	companyOther := seed("companies", map[string]any{"tenant_id": tenantOther, "name": "Finanz AG", "created_by": agentOther})
	seed("contacts", map[string]any{
		"tenant_id": tenantOther, "first_name": "Fiona", "last_name": "Finanzbuch", "email": foreignEmail,
		"company_id": companyOther, "created_by": agentOther,
	})

	// --- invoices (the contact's own and a colleague contact's) ---------------
	today := time.Now().UTC().Truncate(24 * time.Hour)
	invoice := func(tenantID, contactID, by uuid.UUID, number, mail string) uuid.UUID {
		return seed("finance_invoices", map[string]any{
			"tenant_id": tenantID, "contact_id": contactID, "invoice_number": number, "customer_email": mail,
			"invoice_date": today, "due_date": today, "created_by": by,
		})
	}
	invOwn := invoice(tenantOwn, contactID, agentID, "RE-DSAR-"+uuid.NewString()[:8], "other@x.invalid")
	invNamesake := invoice(tenantOwn, namesakeID, agentID, "RE-DSAR-"+uuid.NewString()[:8], "bert@x.invalid")

	// --- dunning: reached through the contact's invoice -------------------------
	seed("finance_dunning_records", map[string]any{
		"tenant_id": tenantOwn, "invoice_id": invOwn, "level": 2, "fee": 5, "created_by": agentID,
	})
	seed("finance_dunning_records", map[string]any{
		"tenant_id": tenantOwn, "invoice_id": invNamesake, "level": 1, "created_by": agentID,
	})

	// --- credit notes: by e-mail OR via the original invoice -----------------------
	creditNote := func(inv uuid.UUID, number, mail string) {
		seed("finance_credit_notes", map[string]any{
			"tenant_id": tenantOwn, "original_invoice_id": inv, "credit_note_number": number,
			"customer_name": "Fiona F.", "customer_email": mail, "reason": "Retoure", "gross_total": 119, "created_by": agentID,
		})
	}
	creditNote(invNamesake, "GS-MAIL", strings.ToUpper(email)) // e-mail only, case-insensitive
	creditNote(invOwn, "GS-INVOICE", "")                       // invoice link only
	creditNote(invNamesake, "GS-FOREIGN", "bert@x.invalid")    // neither

	// --- quotes and subscriptions: e-mail only -----------------------------------
	quote := func(tenantID, by uuid.UUID, number, mail string) {
		seed("finance_quotes", map[string]any{
			"tenant_id": tenantID, "quote_number": number, "customer_name": "Fiona Finanzbuch",
			"customer_address": "Hafenstr. 1, Hamburg", "customer_email": mail, "gross_total": 238, "created_by": by,
		})
	}
	quote(tenantOwn, agentID, "AN-OWN", strings.ToUpper(email))
	quote(tenantOwn, agentID, "AN-COLLEAGUE", "bert@x.invalid")
	quote(tenantOther, agentOther, "AN-FOREIGN", foreignEmail)
	recurring := func(tenantID uuid.UUID, title, mail string) {
		seed("finance_recurring_invoices", map[string]any{
			"tenant_id": tenantID, "title": title, "customer_name": "Fiona Finanzbuch", "customer_email": mail,
			"recurrence_interval": "monthly", "start_date": today, "next_run": today,
		})
	}
	recurring(tenantOwn, "Wartungsabo", email)
	recurring(tenantOwn, "Fremdabo", "bert@x.invalid")

	// --- incoming invoices and bank bookings: name match ---------------------------
	incoming := func(tenantID uuid.UUID, by uuid.UUID, supplier, number string) {
		seed("finance_incoming_invoices", map[string]any{
			"tenant_id": tenantID, "supplier_name": supplier, "supplier_address": "Kaistr. 2", "invoice_number": number,
			"invoice_date": today, "source_format": "xml_only", "original_xml": "<x/>", "gross_total": 50, "created_by": by,
		})
	}
	incoming(tenantOwn, agentID, "FINANZ AG", "ER-1")
	incoming(tenantOwn, agentID, "Fiona Finanzbuch", "ER-2")
	incoming(tenantOwn, agentID, "Anderer Lieferant GmbH", "ER-3")
	incoming(tenantOther, agentOther, "Finanz AG", "ER-4")

	statement := seed("finance_bank_statements", map[string]any{
		"tenant_id": tenantOwn, "format": "camt053", "content_hash": uuid.NewString(),
	})
	booking := func(counterparty, iban, purpose string) {
		seed("finance_bank_transactions", map[string]any{
			"tenant_id": tenantOwn, "statement_id": statement, "value_date": today, "amount": 119.5,
			"counterparty_name": counterparty, "counterparty_iban": iban, "remittance_info": purpose,
		})
	}
	booking("Fiona Finanzbuch", "DE89 3704 0044 0532 0130 00", "Rechnung RE-17")
	booking("Jemand Anderes", "DE02120300000000202051", "Gehalt")

	// --- act ------------------------------------------------------------------
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Finanzbuch")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	column := func(module, key string) []string {
		var out []string
		for _, r := range recordMaps(dsarModule(t, person, module)) {
			out = append(out, r[key])
		}
		return out
	}

	assert.ElementsMatch(t, []string{"AN-OWN"}, column("Angebote", "Angebotsnummer"), "quote matched by e-mail, case-insensitive")
	assert.Equal(t, []string{"238.00 EUR"}, column("Angebote", "Betrag"))
	assert.ElementsMatch(t, []string{"Wartungsabo"}, column("Abos (wiederkehrende Rechnungen)", "Abo"))
	assert.ElementsMatch(t, []string{"GS-MAIL", "GS-INVOICE"}, column("Gutschriften", "Gutschriftsnummer"),
		"credit note by e-mail or via the contact's invoice, never the colleague's")

	dunning := recordMaps(dsarModule(t, person, "Mahnungen"))
	require.Len(t, dunning, 1, "only the dunning of the contact's own invoice")
	assert.Equal(t, "2", dunning[0]["Mahnstufe"])

	assert.ElementsMatch(t, []string{"ER-1", "ER-2"}, column("Eingangsrechnungen (Namensabgleich)", "Rechnungsnummer"),
		"supplier name equal to company or full name, other tenant's namesake supplier excluded")

	bank := recordMaps(dsarModule(t, person, "Bankumsätze (Namensabgleich)"))
	require.Len(t, bank, 1)
	assert.Equal(t, "****3000", bank[0]["IBAN"], "IBAN must be masked to the last four digits")
	assert.Equal(t, "Rechnung RE-17", bank[0]["Verwendungszweck"])
	assert.Equal(t, "119.50 EUR", bank[0]["Betrag"])
	for _, m := range person.Modules {
		for _, rec := range m.Records {
			for _, f := range rec.Fields {
				assert.NotContains(t, f.Value, "DE89", "full IBAN must never be disclosed")
			}
		}
	}

	// A contact without e-mail and without a matching name sees none of this.
	namesakePersons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Buchhalter")
	require.NoError(t, err)
	require.Len(t, namesakePersons, 1)
	for _, name := range []string{"Angebote", "Abos (wiederkehrende Rechnungen)", "Eingangsrechnungen (Namensabgleich)", "Bankumsätze (Namensabgleich)"} {
		assert.NotContains(t, moduleNames(namesakePersons[0]), name)
	}

	// --- tenant isolation -----------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Finanzbuch")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	assert.Equal(t, []string{"AN-FOREIGN"}, column2(t, foreign[0], "Angebote", "Angebotsnummer"))
	assert.Equal(t, []string{"ER-4"}, column2(t, foreign[0], "Eingangsrechnungen (Namensabgleich)", "Rechnungsnummer"))
	assert.NotContains(t, moduleNames(foreign[0]), "Bankumsätze (Namensabgleich)")
	assert.NotContains(t, moduleNames(foreign[0]), "Gutschriften")
}

func TestSearchByQuery_UserExpenses_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Expenses Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Expenses Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Sven", "Spesenritter", true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kai", "Kollege", true)
	foreignID := seedDSARUser(t, pool, tenantOther, "Sven", "Spesenritter", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	var ids []uuid.UUID
	expense := func(tenantID uuid.UUID, desc string, submitted, decided *uuid.UUID) {
		cols := map[string]any{
			"tenant_id": tenantID, "description": desc, "amount": 42.5, "expense_date": today,
			"supplier": "Tankstelle", "category": "Reise", "receipt_name": "beleg.pdf", "status": "approved",
		}
		if submitted != nil {
			cols["submitted_by"] = *submitted
		}
		if decided != nil {
			cols["decided_by"] = *decided
			cols["decided_at"] = time.Now().UTC()
		}
		ids = append(ids, testutil.SeedRow(t, pool, "finance_expenses", cols))
	}
	defer func() {
		for _, id := range ids {
			testutil.CleanupRow(t, pool, "finance_expenses", id)
		}
	}()
	expense(tenantOwn, "Taxi Kundentermin", &subjectID, &colleagueID)
	expense(tenantOwn, "Hotel Messe", &colleagueID, &subjectID)
	expense(tenantOwn, "Selbstfreigabe", &subjectID, &subjectID)
	expense(tenantOwn, "Kollegenspesen", &colleagueID, &colleagueID)
	expense(tenantOwn, "Ohne Bezug", nil, nil)
	expense(tenantOther, "Fremde Spesen", &foreignID, nil)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Spesenritter")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	byDesc := map[string]string{}
	for _, r := range recordMaps(dsarModule(t, persons[0], "Spesen")) {
		byDesc[r["Beschreibung"]] = r["Rolle"]
		assert.Equal(t, "42.50 EUR", r["Betrag"])
	}
	assert.Equal(t, map[string]string{
		"Taxi Kundentermin": "Einreicher", "Hotel Messe": "Entscheider", "Selbstfreigabe": "Einreicher und Entscheider",
	}, byDesc)

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Spesenritter")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	recs := recordMaps(dsarModule(t, foreign[0], "Spesen"))
	require.Len(t, recs, 1)
	assert.Equal(t, "Fremde Spesen", recs[0]["Beschreibung"])
}

// column2 returns one column of a module as a slice.
func column2(t *testing.T, p DSARPerson, module, key string) []string {
	t.Helper()
	var out []string
	for _, r := range recordMaps(dsarModule(t, p, module)) {
		out = append(out, r[key])
	}
	return out
}
