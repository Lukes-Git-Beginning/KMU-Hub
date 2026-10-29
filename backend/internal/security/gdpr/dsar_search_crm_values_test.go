package gdpr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestWebhookHost(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hooks.example.invalid:8443", webhookHost("https://user:pw@hooks.example.invalid:8443/a/b?token=geheim"))
	assert.Equal(t, "(ungültige URL)", webhookHost("::nicht-gueltig"))
	assert.Equal(t, "(ungültige URL)", webhookHost("/nur/ein/pfad"))
}

func TestSearchByQuery_ContactCrmValuesCsatAndWebhooks_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR CRM Values Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR CRM Values Other Tenant")
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
	// The value tables have no id column; they vanish with their parent row.
	sysCtx := testutil.WithSystemCtx(context.Background())
	seedValue := func(table, parentCol string, parent, field uuid.UUID, valueJSON string) {
		_, err := pool.Exec(sysCtx,
			fmt.Sprintf(`INSERT INTO %s (%s, field_id, value) VALUES ($1, $2, $3::jsonb)`, table, parentCol),
			parent, field, valueJSON)
		require.NoError(t, err)
	}

	email := fmt.Sprintf("vera.%s@werte.invalid", uuid.New())
	emailOther := fmt.Sprintf("vera.%s@werte.invalid", uuid.New())

	field := func(tenantID, by uuid.UUID, entity, name, label string) uuid.UUID {
		return seed("custom_field_definitions", map[string]any{
			"tenant_id": tenantID, "entity_type": entity, "field_name": name + uuid.NewString()[:6],
			"field_label": label, "field_type": "text", "created_by": by,
		})
	}
	companyField := field(tenantOwn, agentID, "company", "umsatz", "Umsatzklasse")
	dealField := field(tenantOwn, agentID, "deal", "quelle", "Lead-Quelle")
	activityField := field(tenantOwn, agentID, "activity", "gespraech", "Gesprächsnotiz")
	otherField := field(tenantOther, agentOther, "deal", "quelle", "Fremd-Feld")

	stage := seed("pipeline_stages", map[string]any{"tenant_id": tenantOwn, "name": "Angebot"})
	stageOther := seed("pipeline_stages", map[string]any{"tenant_id": tenantOther, "name": "Angebot"})

	company := seed("companies", map[string]any{"tenant_id": tenantOwn, "name": "Werte GmbH", "created_by": agentID})
	contactID := seed("contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Vera", "last_name": "Wertekontakt", "email": email,
		"company_id": company, "created_by": agentID,
	})
	colleagueCompany := seed("companies", map[string]any{"tenant_id": tenantOwn, "name": "Kollegen KG", "created_by": agentID})
	colleagueID := seed("contacts", map[string]any{
		"tenant_id": tenantOwn, "first_name": "Bernd", "last_name": "Kollege",
		"email": fmt.Sprintf("bernd.%s@werte.invalid", uuid.New()), "company_id": colleagueCompany, "created_by": agentID,
	})

	seedValue("company_custom_field_values", "company_id", company, companyField, `"Klasse B"`)
	seedValue("company_custom_field_values", "company_id", colleagueCompany, companyField, `"Klasse Z"`)

	dealOwn := seed("deals", map[string]any{
		"tenant_id": tenantOwn, "name": "Rahmenvertrag", "stage_id": stage, "contact_id": contactID, "created_by": agentID,
	})
	dealColleague := seed("deals", map[string]any{
		"tenant_id": tenantOwn, "name": "Kollegen-Deal", "stage_id": stage, "contact_id": colleagueID, "created_by": agentID,
	})
	seedValue("deal_custom_field_values", "deal_id", dealOwn, dealField, `"Messe Köln"`)
	seedValue("deal_custom_field_values", "deal_id", dealColleague, dealField, `"Kaltakquise"`)

	activityOwn := seed("activities", map[string]any{
		"tenant_id": tenantOwn, "activity_type": "call", "subject": "Rückruf", "contact_id": contactID, "created_by": agentID,
	})
	activityColleague := seed("activities", map[string]any{
		"tenant_id": tenantOwn, "activity_type": "note", "subject": "Kollegen-Notiz", "contact_id": colleagueID, "created_by": agentID,
	})
	seedValue("activity_custom_field_values", "activity_id", activityOwn, activityField, `"wünscht Rückruf am Montag"`)
	seedValue("activity_custom_field_values", "activity_id", activityColleague, activityField, `"geheim"`)

	// --- CSAT: submitted comment shown, pending survey and colleague not --------------
	submittedAt := time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)
	ticketDone := seed("tickets", map[string]any{
		"tenant_id": tenantOwn, "subject": "Lieferverzug", "requester_id": agentID, "contact_id": contactID,
	})
	seed("ticket_csat_responses", map[string]any{
		"tenant_id": tenantOwn, "ticket_id": ticketDone, "rating": 2,
		"comment": "Zu langsam, Frau Müller war unfreundlich.", "submitted_at": submittedAt,
	})
	ticketPending := seed("tickets", map[string]any{
		"tenant_id": tenantOwn, "subject": "Offene Umfrage", "requester_id": agentID, "contact_id": contactID,
	})
	seed("ticket_csat_responses", map[string]any{
		"tenant_id": tenantOwn, "ticket_id": ticketPending,
		"token_hash": "geheimes-umfrage-token-" + uuid.NewString(), "token_expires_at": time.Now().Add(time.Hour),
	})
	ticketColleague := seed("tickets", map[string]any{
		"tenant_id": tenantOwn, "subject": "Kollegen-Ticket", "requester_id": agentID, "contact_id": colleagueID,
	})
	seed("ticket_csat_responses", map[string]any{
		"tenant_id": tenantOwn, "ticket_id": ticketColleague, "rating": 5, "comment": "Kollege zufrieden", "submitted_at": submittedAt,
	})

	// --- webhook deliveries of the contact's form submission --------------------------
	fields := []byte(`[
		{"id":"f_name","type":"text","label":"Name"},
		{"id":"f_email","type":"email","label":"E-Mail"}
	]`)
	schema := seed("form_schemas", map[string]any{
		"tenant_id": tenantOwn, "title": "Anfrageformular", "fields": fields, "created_by": agentID,
	})
	hook := seed("form_webhooks", map[string]any{
		"tenant_id": tenantOwn, "form_schema_id": schema, "secret": "signing-secret-nie-ausgeben",
		"url": "https://user:pw@hooks.example.invalid/in/abc?token=url-token-nie-ausgeben",
	})
	delivered := time.Now().UTC().Truncate(time.Minute)
	submission := func(answers string, ago time.Duration) uuid.UUID {
		return seed("form_submissions", map[string]any{
			"tenant_id": tenantOwn, "form_schema_id": schema, "answers": []byte(answers),
			"submitted_at": time.Now().UTC().Truncate(time.Minute).Add(-ago),
		})
	}
	subOwn := submission(fmt.Sprintf(`{"f_name":"Vera","f_email":"%s"}`, email), 2*time.Hour)
	subColleague := submission(`{"f_name":"Bernd","f_email":"bernd@werte.invalid"}`, 3*time.Hour)
	seed("form_webhook_deliveries", map[string]any{
		"tenant_id": tenantOwn, "webhook_id": hook, "submission_id": subOwn, "status": "delivered",
		"attempt_count": 2, "payload": []byte(`{"answers":{"f_name":"Vera"}}`), "delivered_at": delivered,
	})
	seed("form_webhook_deliveries", map[string]any{
		"tenant_id": tenantOwn, "webhook_id": hook, "submission_id": subColleague, "status": "delivered",
		"attempt_count": 1, "payload": []byte(`{"answers":{"f_name":"Bernd"}}`),
	})

	// --- other tenant: the same-named contact with its own data ----------------------------
	companyOther := seed("companies", map[string]any{"tenant_id": tenantOther, "name": "Fremd GmbH", "created_by": agentOther})
	contactOther := seed("contacts", map[string]any{
		"tenant_id": tenantOther, "first_name": "Vera", "last_name": "Wertekontakt", "email": emailOther,
		"company_id": companyOther, "created_by": agentOther,
	})
	dealOther := seed("deals", map[string]any{
		"tenant_id": tenantOther, "name": "Fremd-Deal", "stage_id": stageOther, "contact_id": contactOther, "created_by": agentOther,
	})
	seedValue("deal_custom_field_values", "deal_id", dealOther, otherField, `"Fremdwert"`)

	// --- act ------------------------------------------------------------------------------
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	// Timestamps are rendered in the session zone; read them back instead of
	// formatting the UTC values the test seeded.
	var csatAt, deliveredAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn,
		`SELECT submitted_at FROM ticket_csat_responses WHERE ticket_id = $1`, ticketDone).Scan(&csatAt))
	require.NoError(t, pool.QueryRow(ctxOwn,
		`SELECT delivered_at FROM form_webhook_deliveries WHERE submission_id = $1`, subOwn).Scan(&deliveredAt))

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Wertekontakt")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	company1 := dsarModule(t, person, "Benutzerdefinierte Felder (Firma)")
	assert.Equal(t, []string{"Firma", "Feld", "Wert"}, company1.Columns)
	assert.Equal(t, []map[string]string{{"Firma": "Werte GmbH", "Feld": "Umsatzklasse", "Wert": "Klasse B"}}, recordMaps(company1),
		"only the contact's own company, never the colleague's")
	assert.Equal(t, []map[string]string{{"Deal": "Rahmenvertrag", "Feld": "Lead-Quelle", "Wert": "Messe Köln"}},
		recordMaps(dsarModule(t, person, "Benutzerdefinierte Felder (Deals)")))
	assert.Equal(t, []map[string]string{{"Aktivität": "Rückruf", "Feld": "Gesprächsnotiz", "Wert": "wünscht Rückruf am Montag"}},
		recordMaps(dsarModule(t, person, "Benutzerdefinierte Felder (Aktivitäten)")))

	assert.Equal(t, []map[string]string{{
		"Ticket": "Lieferverzug", "Bewertung": "2 von 5",
		"Kommentar": "Zu langsam, Frau Müller war unfreundlich.", "Abgegeben": csatAt.Format(dsarTimeLayout),
	}}, recordMaps(dsarModule(t, person, "Zufriedenheitsumfragen (CSAT)")),
		"pending survey and the colleague's response must not appear")

	hooks := recordMaps(dsarModule(t, person, "Webhook-Weiterleitungen von Formulareinreichungen"))
	require.Len(t, hooks, 1, "only the delivery of the contact's own submission")
	assert.Equal(t, "Anfrageformular", hooks[0]["Formular"])
	assert.Equal(t, "hooks.example.invalid", hooks[0]["Empfänger"], "URL reduced to its host")
	assert.Equal(t, "delivered", hooks[0]["Status"])
	assert.Equal(t, "2", hooks[0]["Versuche"])
	assert.Equal(t, deliveredAt.Format(dsarTimeLayout), hooks[0]["Zugestellt"])

	for _, m := range person.Modules {
		for _, rec := range m.Records {
			for _, f := range rec.Fields {
				for _, secret := range []string{"geheimes-umfrage-token", "signing-secret", "url-token", "user:pw", "Fremdwert", "Klasse Z", "Kaltakquise", "Kollegen"} {
					assert.NotContains(t, f.Value, secret, "module %q leaked %q", m.Module, secret)
				}
			}
		}
	}

	// A colleague contact with no email-matching submission sees only its own rows.
	colleagues, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Kollege")
	require.NoError(t, err)
	require.Len(t, colleagues, 1)
	assert.Equal(t, []map[string]string{{"Firma": "Kollegen KG", "Feld": "Umsatzklasse", "Wert": "Klasse Z"}},
		recordMaps(dsarModule(t, colleagues[0], "Benutzerdefinierte Felder (Firma)")))
	assert.NotContains(t, moduleNames(colleagues[0]), "Webhook-Weiterleitungen von Formulareinreichungen",
		"his e-mail is not in any email-typed answer")

	// --- tenant isolation -----------------------------------------------------------------
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Wertekontakt")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	assert.Equal(t, []map[string]string{{"Deal": "Fremd-Deal", "Feld": "Fremd-Feld", "Wert": "Fremdwert"}},
		recordMaps(dsarModule(t, foreign[0], "Benutzerdefinierte Felder (Deals)")))
	for _, name := range []string{"Benutzerdefinierte Felder (Firma)", "Benutzerdefinierte Felder (Aktivitäten)",
		"Zufriedenheitsumfragen (CSAT)", "Webhook-Weiterleitungen von Formulareinreichungen"} {
		assert.NotContains(t, moduleNames(foreign[0]), name)
	}

	// Forged tenant ID: the other tenant's pool context cannot read tenantOwn's rows.
	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Wertekontakt")
	require.NoError(t, err)
	assert.Empty(t, forged)
}
