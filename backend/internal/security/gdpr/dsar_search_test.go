package gdpr

// Coverage for dsar_search.go — the Art. 15 GDPR cross-module lookup, which had
// no test at all (0,0 %).
//
// Two layers are exercised separately:
//
//  1. The four pure helpers (fieldValueRecord, joinName, initials, boolLabel)
//     as direct table tests, including the empty-string edges that produce the
//     "—" and "?" placeholders shown in the UI.
//  2. SearchByQuery end to end against the local Postgres as kmuhub_app
//     (NOSUPERUSER NOBYPASSRLS), so tenant isolation is proven by RLS rather
//     than asserted from a bypassing role.
//
// The property most easily lost in a refactor is that consentModule and
// dialerModule return a nil *DSARModule — not an empty one — when the subject
// has no rows there. A person with no consent and no calls must carry exactly
// one module, otherwise the UI renders empty tables for modules that hold no
// data about them.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

func TestJoinName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		first, last string
		want        string
	}{
		{"both parts", "Erika", "Musterfrau", "Erika Musterfrau"},
		{"first only", "Erika", "", "Erika"},
		{"last only", "", "Musterfrau", "Musterfrau"},
		{"both empty falls back to placeholder", "", "", "—"},
		{"whitespace only is trimmed to the placeholder", "  ", " ", "—"},
		{"inner spacing survives the trim", " Erika ", " Musterfrau ", "Erika   Musterfrau"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, joinName(tc.first, tc.last))
		})
	}
}

func TestInitials(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		first, last string
		want        string
	}{
		{"both parts", "Erika", "Musterfrau", "EM"},
		{"lowercase is upcased", "erika", "musterfrau", "EM"},
		{"first only", "Erika", "", "E"},
		{"last only", "", "Musterfrau", "M"},
		{"both empty falls back to placeholder", "", "", "?"},
		// The slice is over runes, not bytes: a byte slice would cut a
		// two-byte umlaut in half and yield mojibake.
		{"multi-byte first rune stays intact", "Ölaf", "Ärgerlich", "ÖÄ"},
		{"emoji is a single rune", "🙂lias", "Meier", "🙂M"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, initials(tc.first, tc.last))
		})
	}
}

func TestBoolLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Ja", boolLabel(true))
	assert.Equal(t, "Nein", boolLabel(false))
}

func TestFieldValueRecord(t *testing.T) {
	t.Parallel()

	rec := fieldValueRecord("Telefon", "+49 30 123456")
	require.Len(t, rec.Fields, 2)
	// Order matters — the UI renders the fields positionally, not by key.
	assert.Equal(t, DSARField{Key: "Feld", Value: "Telefon"}, rec.Fields[0])
	assert.Equal(t, DSARField{Key: "Wert", Value: "+49 30 123456"}, rec.Fields[1])

	empty := fieldValueRecord("Position", "")
	require.Len(t, empty.Fields, 2)
	assert.Equal(t, "", empty.Fields[1].Value, "an empty value must still produce a row, not be dropped")
}

// ---------------------------------------------------------------------------
// SearchByQuery — integration
// ---------------------------------------------------------------------------

func TestSearchByQuery_ContactWithAllModules_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Search Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Search Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "Agent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	companyID := testutil.SeedRow(t, pool, "companies", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Nordwind Logistik GmbH",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "companies", companyID)

	email := fmt.Sprintf("erika.%s@nordwind.invalid", uuid.New())
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Sturmvogel",
		"email":      email,
		"phone":      "+49 30 9988776",
		"position":   "Disponentin",
		"company_id": companyID,
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	base := time.Now().UTC().Truncate(time.Minute).Add(-48 * time.Hour)
	grantedAt := base.Add(30 * time.Minute)
	revokedAt := base.Add(-90 * time.Minute)

	// Newest first: granted -> "Erteilt", explicit granted_at wins over created_at.
	consentGranted := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"tenant_id":    tenantOwn,
		"contact_id":   contactID,
		"consent_type": "marketing_email",
		"granted":      true,
		"legal_basis":  "consent",
		"granted_at":   grantedAt,
		"created_at":   base,
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentGranted)

	// granted = false -> "Verweigert", no granted_at so the COALESCE falls back.
	consentDenied := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"tenant_id":    tenantOwn,
		"contact_id":   contactID,
		"consent_type": "newsletter",
		"granted":      false,
		"legal_basis":  "legitimate_interest",
		"created_at":   base.Add(-time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentDenied)

	// revoked_at set -> "Widerrufen" wins even though granted is true.
	consentRevoked := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"tenant_id":    tenantOwn,
		"contact_id":   contactID,
		"consent_type": "profiling",
		"granted":      true,
		"legal_basis":  "contract",
		"revoked_at":   revokedAt,
		"created_at":   base.Add(-2 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentRevoked)

	campaignID := testutil.SeedRow(t, pool, "dialer_campaigns", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Nordwind Rueckgewinnung",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaigns", campaignID)

	campaignContactID := testutil.SeedRow(t, pool, "dialer_campaign_contacts", map[string]any{
		"tenant_id":   tenantOwn,
		"campaign_id": campaignID,
		"contact_id":  contactID,
		"position":    1,
	})
	defer testutil.CleanupRow(t, pool, "dialer_campaign_contacts", campaignContactID)

	callNewID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"tenant_id":           tenantOwn,
		"campaign_contact_id": campaignContactID,
		"agent_id":            agentID,
		"duration_seconds":    125,
		"notes":               "Rueckruf fuer Mittwoch vereinbart",
		"created_at":          base,
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", callNewID)

	// duration_seconds and notes stay NULL — both are COALESCEd in the query.
	callOldID := testutil.SeedRow(t, pool, "dialer_call_sessions", map[string]any{
		"tenant_id":           tenantOwn,
		"campaign_contact_id": campaignContactID,
		"agent_id":            agentID,
		"created_at":          base.Add(-3 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "dialer_call_sessions", callOldID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// The formatted timestamps are compared against the values Postgres actually
	// stored, read back through the same connection — that pins the layout
	// without making the assertion depend on the session time zone.
	var contactCreatedAt, callNewCreatedAt, callOldCreatedAt, deniedCreatedAt, revokedCreatedAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM contacts WHERE id = $1`, contactID).Scan(&contactCreatedAt))
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM dialer_call_sessions WHERE id = $1`, callNewID).Scan(&callNewCreatedAt))
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM dialer_call_sessions WHERE id = $1`, callOldID).Scan(&callOldCreatedAt))
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM consent_records WHERE id = $1`, consentDenied).Scan(&deniedCreatedAt))
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM consent_records WHERE id = $1`, consentRevoked).Scan(&revokedCreatedAt))
	var grantedAtStored time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT granted_at FROM consent_records WHERE id = $1`, consentGranted).Scan(&grantedAtStored))

	// --- match by last-name substring -------------------------------------
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "turmvog")
	require.NoError(t, err)
	require.Len(t, persons, 1, "the last-name substring must match exactly the seeded contact")

	p := persons[0]
	assert.Equal(t, contactID.String(), p.ID)
	assert.Equal(t, "Erika Sturmvogel", p.Name)
	assert.Equal(t, email, p.Email)
	assert.Equal(t, "Nordwind Logistik GmbH", p.Company, "the company name comes from the LEFT JOIN, not from the contact row")
	assert.Equal(t, "ES", p.Avatar)

	require.Len(t, p.Modules, 3, "CRM + consent + dialer")
	assert.Equal(t, []string{"CRM Kontakte", "Einwilligungen", "Anrufe"}, moduleNames(p),
		"module order is part of the contract — the UI renders them positionally")

	crm := dsarModule(t, p, "CRM Kontakte")
	assert.Equal(t, []string{"Feld", "Wert"}, crm.Columns)
	assert.Equal(t, map[string]string{
		"Name":        "Erika Sturmvogel",
		"E-Mail":      email,
		"Telefon":     "+49 30 9988776",
		"Position":    "Disponentin",
		"Unternehmen": "Nordwind Logistik GmbH",
		"Erstellt":    contactCreatedAt.Format(dsarTimeLayout),
	}, fieldValueMap(t, crm))

	consent := dsarModule(t, p, "Einwilligungen")
	assert.Equal(t, []string{"Typ", "Status", "Rechtsgrundlage", "Datum"}, consent.Columns)
	require.Len(t, consent.Records, 3)
	assert.Equal(t, []map[string]string{
		{"Typ": "marketing_email", "Status": "Erteilt", "Rechtsgrundlage": "consent", "Datum": grantedAtStored.Format(dsarTimeLayout)},
		{"Typ": "newsletter", "Status": "Verweigert", "Rechtsgrundlage": "legitimate_interest", "Datum": deniedCreatedAt.Format(dsarTimeLayout)},
		{"Typ": "profiling", "Status": "Widerrufen", "Rechtsgrundlage": "contract", "Datum": revokedCreatedAt.Format(dsarTimeLayout)},
	}, recordMaps(consent), "newest consent first; a revocation outranks granted = true")

	calls := dsarModule(t, p, "Anrufe")
	assert.Equal(t, []string{"Datum", "Dauer (s)", "Notiz"}, calls.Columns)
	assert.Equal(t, []map[string]string{
		{"Datum": callNewCreatedAt.Format(dsarTimeLayout), "Dauer (s)": "125", "Notiz": "Rueckruf fuer Mittwoch vereinbart"},
		{"Datum": callOldCreatedAt.Format(dsarTimeLayout), "Dauer (s)": "0", "Notiz": ""},
	}, recordMaps(calls), "NULL duration and notes must surface as 0 and the empty string, not as an error")

	// --- match by email substring -----------------------------------------
	byEmail, err := SearchByQuery(ctxOwn, pool, tenantOwn, "@nordwind.invalid")
	require.NoError(t, err)
	require.Len(t, byEmail, 1, "the email substring must find the same contact")
	assert.Equal(t, contactID.String(), byEmail[0].ID)

	// --- match by the concatenated full name --------------------------------
	byFullName, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Erika Sturmvogel")
	require.NoError(t, err)
	require.Len(t, byFullName, 1, "first_name || ' ' || last_name must match as one string")
	assert.Equal(t, contactID.String(), byFullName[0].ID)

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "turmvog")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact")

	// A forged tenant argument does not help: the connection is stamped from
	// the context, so RLS still filters the rows out.
	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "turmvog")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")

	// --- no hit --------------------------------------------------------------
	none, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Suedwind")
	require.NoError(t, err)
	require.NotNil(t, none, "no hit must yield an empty slice, never nil — it is marshalled as [] downstream")
	assert.Empty(t, none)
}

func TestSearchByQuery_ContactWithoutConsentOrCalls_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "DSAR Bare Contact Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	creatorID := seedDSARUser(t, pool, tenantID, "Dsar", "Creator", true)
	defer testutil.CleanupRow(t, pool, "users", creatorID)

	// No email, no phone, no position, no company — every COALESCE in
	// matchContacts falls back to the empty string.
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Kasimir",
		"last_name":  "Habichtsberg",
		"created_by": creatorID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	persons, err := SearchByQuery(ctx, pool, tenantID, "Habichtsberg")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	p := persons[0]
	assert.Equal(t, "", p.Email)
	assert.Equal(t, "", p.Company)
	require.Len(t, p.Modules, 1,
		"consentModule and dialerModule must return nil, not an empty module — an empty module renders as a table with no rows")
	assert.Equal(t, "CRM Kontakte", p.Modules[0].Module)

	crm := fieldValueMap(t, p.Modules[0])
	assert.Equal(t, "", crm["E-Mail"])
	assert.Equal(t, "", crm["Telefon"])
	assert.Equal(t, "", crm["Position"])
	assert.Equal(t, "", crm["Unternehmen"])
	assert.Equal(t, "Kasimir Habichtsberg", crm["Name"])
}

func TestSearchByQuery_ContactCustomFieldsAndTags_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Custom Fields Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Custom Fields Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "FieldsAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Ottilie",
		"last_name":  "Wolkenbruch",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	// A text field (renders its JSON string value verbatim) and a number field
	// (renders the JSON float without quotes or trailing zeros) — proving
	// formatCustomFieldValue does not just special-case strings.
	textFieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   tenantOwn,
		"entity_type": "contact",
		"field_name":  "customer_number",
		"field_label": "Kundennummer",
		"field_type":  "text",
		"sort_order":  0,
		"created_by":  agentID,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", textFieldID)

	numberFieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   tenantOwn,
		"entity_type": "contact",
		"field_name":  "credit_limit",
		"field_label": "Kreditlimit",
		"field_type":  "number",
		"sort_order":  1,
		"created_by":  agentID,
	})
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", numberFieldID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	contactRepo := contact.NewPostgresRepository(pool)
	// No explicit cleanup for the value rows: field_id carries ON DELETE
	// CASCADE from custom_field_definitions, so deleting the two definitions
	// below removes them too.
	require.NoError(t, contactRepo.SetCustomFieldValues(ctxOwn, contactID, map[uuid.UUID]any{
		textFieldID:   "KD-4711",
		numberFieldID: 4200.5,
	}))

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "VIP",
		"color":       "#ff0000",
		"entity_type": "contact",
	})
	// Same here: contact_tags.tag_id cascades from tags, no separate cleanup.
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	require.NoError(t, contactRepo.AddTags(ctxOwn, contactID, []uuid.UUID{tagID}))

	var tagAssignedAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM contact_tags WHERE contact_id = $1 AND tag_id = $2`, contactID, tagID).Scan(&tagAssignedAt))

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Wolkenbruch")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	p := persons[0]

	customFields := dsarModule(t, p, "Benutzerdefinierte Felder")
	assert.Equal(t, []string{"Feld", "Wert"}, customFields.Columns)
	assert.Equal(t, map[string]string{
		"Kundennummer": "KD-4711",
		"Kreditlimit":  "4200.5",
	}, fieldValueMap(t, customFields), "custom fields must disclose the definition's field_label, not the field_id or field_name")

	tags := dsarModule(t, p, "Tags")
	assert.Equal(t, []string{"Tag", "Zugewiesen"}, tags.Columns)
	assert.Equal(t, []map[string]string{
		{"Tag": "VIP", "Zugewiesen": tagAssignedAt.Format(dsarTimeLayout)},
	}, recordMaps(tags))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Wolkenbruch")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact, let alone its custom fields or tags")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Wolkenbruch")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")
}

func TestSearchByQuery_MatchesUsers_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR User Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR User Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	activeID := seedDSARUser(t, pool, tenantOwn, "Wendelin", "Falkenhorst", true)
	defer testutil.CleanupRow(t, pool, "users", activeID)
	inactiveID := seedDSARUser(t, pool, tenantOwn, "Wilma", "Falkenhorst", false)
	defer testutil.CleanupRow(t, pool, "users", inactiveID)
	// Same name, different tenant — must stay invisible.
	foreignID := seedDSARUser(t, pool, tenantOther, "Fremder", "Falkenhorst", true)
	defer testutil.CleanupRow(t, pool, "users", foreignID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Falkenhorst")
	require.NoError(t, err)
	require.Len(t, persons, 2, "both own-tenant users, never the foreign one")

	byID := map[string]DSARPerson{}
	for _, p := range persons {
		byID[p.ID] = p
	}
	require.Contains(t, byID, activeID.String())
	require.Contains(t, byID, inactiveID.String())
	require.NotContains(t, byID, foreignID.String(), "a same-named user of another tenant must not be disclosed")

	active := byID[activeID.String()]
	assert.Equal(t, "Wendelin Falkenhorst", active.Name)
	assert.Equal(t, "WF", active.Avatar)
	assert.Equal(t, "", active.Company, "users carry no company — the field stays empty")
	require.Len(t, active.Modules, 1)
	assert.Equal(t, "Benutzerkonto", active.Modules[0].Module)
	assert.Equal(t, []string{"Feld", "Wert"}, active.Modules[0].Columns)

	var activeCreatedAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM users WHERE id = $1`, activeID).Scan(&activeCreatedAt))
	fields := fieldValueMap(t, active.Modules[0])
	assert.Equal(t, "Wendelin Falkenhorst", fields["Name"])
	assert.Equal(t, "Ja", fields["Aktiv"])
	assert.Equal(t, activeCreatedAt.Format(dsarTimeLayout), fields["Erstellt"])

	assert.Equal(t, "Nein", fieldValueMap(t, byID[inactiveID.String()].Modules[0])["Aktiv"],
		"is_active = false must render as Nein")
}

// TestSearchByQuery_NoMinimumLengthGuard_Integration pins where the guard for
// short queries lives. SearchByQuery itself has none: the pattern is built as
// "%" + query + "%", so an empty query lists every subject of the tenant up to
// dsarMaxSubjects. The only guard is in SecurityGRPCServer.DSARSearch
// (internal/server/security_grpc.go), which rejects queries under two runes —
// a second caller of SearchByQuery would have to repeat it.
func TestSearchByQuery_NoMinimumLengthGuard_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "DSAR Empty Query Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := seedDSARUser(t, pool, tenantID, "Ingeborg", "Rabenstein", true)
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Ingeborg",
		"last_name":  "Rabenstein",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	persons, err := SearchByQuery(ctx, pool, tenantID, "")
	require.NoError(t, err)
	assert.Len(t, persons, 2, "an empty query matches every subject of the tenant — the length guard sits in the RPC handler")
}

func TestSearchByQuery_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	persons, err := SearchByQuery(ctx, pool, testutil.TenantA, "Sturmvogel")
	require.Error(t, err, "a DB failure must surface — a silent empty result would look like 'no data held about you'")
	assert.Nil(t, persons)
	assert.Contains(t, err.Error(), "dsar: query contacts:", "the error must name the failing stage")
}

// ---------------------------------------------------------------------------
// Fixtures and assertion helpers
// ---------------------------------------------------------------------------

func seedDSARUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, first, last string, active bool) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("dsar-%s@search.invalid", uuid.New()),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    first,
		"last_name":     last,
		"is_active":     active,
	})
}

func moduleNames(p DSARPerson) []string {
	names := make([]string, 0, len(p.Modules))
	for _, m := range p.Modules {
		names = append(names, m.Module)
	}
	return names
}

func dsarModule(t *testing.T, p DSARPerson, name string) DSARModule {
	t.Helper()
	for _, m := range p.Modules {
		if m.Module == name {
			return m
		}
	}
	t.Fatalf("module %q not found, have %v", name, moduleNames(p))
	return DSARModule{}
}

// fieldValueMap flattens a Feld/Wert module (the shape fieldValueRecord
// produces) into a map for order-independent assertions.
func fieldValueMap(t *testing.T, m DSARModule) map[string]string {
	t.Helper()
	out := make(map[string]string, len(m.Records))
	for _, r := range m.Records {
		require.Len(t, r.Fields, 2, "a Feld/Wert record must carry exactly two fields")
		require.Equal(t, "Feld", r.Fields[0].Key)
		require.Equal(t, "Wert", r.Fields[1].Key)
		out[r.Fields[0].Value] = r.Fields[1].Value
	}
	return out
}

// recordMaps turns the key/value fields of every record into a map, keeping
// record order — used for the multi-column modules where row order is part of
// the contract but column order is already asserted via Columns.
func recordMaps(m DSARModule) []map[string]string {
	out := make([]map[string]string, 0, len(m.Records))
	for _, r := range m.Records {
		row := make(map[string]string, len(r.Fields))
		for _, f := range r.Fields {
			row[f.Key] = f.Value
		}
		out = append(out, row)
	}
	return out
}
