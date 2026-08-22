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
	"strings"
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

func TestSearchByQuery_ContactDocuments_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Documents Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Documents Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "DocsAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Reinhilde",
		"last_name":  "Ackerknecht",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "DSAR Test Folder",
		"space_type": "team",
		"space_id":   uuid.New(),
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", folderID)

	// Linked, visible file -- must appear.
	linkedFileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id":   tenantOwn,
		"folder_id":   folderID,
		"filename":    "Ausweiskopie.pdf",
		"mime_type":   "application/pdf",
		"file_size":   204800,
		"storage_key": "dsar-test/ausweiskopie.pdf",
		"owner_id":    agentID,
	})
	defer testutil.CleanupRow(t, pool, "document_files", linkedFileID)
	linkID := testutil.SeedRow(t, pool, "document_entity_links", map[string]any{
		"tenant_id":   tenantOwn,
		"file_id":     linkedFileID,
		"entity_type": "contact",
		"entity_id":   contactID,
		"linked_by":   agentID,
	})
	defer testutil.CleanupRow(t, pool, "document_entity_links", linkID)

	var linkedAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM document_files WHERE id = $1`, linkedFileID).Scan(&linkedAt))

	// Linked, but soft-deleted file -- must NOT appear.
	trashedFileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id":   tenantOwn,
		"folder_id":   folderID,
		"filename":    "Alter Vertrag.pdf",
		"mime_type":   "application/pdf",
		"file_size":   1024,
		"storage_key": "dsar-test/alter-vertrag.pdf",
		"owner_id":    agentID,
		"is_deleted":  true,
	})
	defer testutil.CleanupRow(t, pool, "document_files", trashedFileID)
	trashedLinkID := testutil.SeedRow(t, pool, "document_entity_links", map[string]any{
		"tenant_id":   tenantOwn,
		"file_id":     trashedFileID,
		"entity_type": "contact",
		"entity_id":   contactID,
		"linked_by":   agentID,
	})
	defer testutil.CleanupRow(t, pool, "document_entity_links", trashedLinkID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Ackerknecht")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	p := persons[0]

	documents := dsarModule(t, p, "Dokumente")
	assert.Equal(t, []string{"Dateiname", "Typ", "Größe", "Hochgeladen am"}, documents.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Dateiname":      "Ausweiskopie.pdf",
			"Typ":            "application/pdf",
			"Größe":          "204800 Bytes",
			"Hochgeladen am": linkedAt.Format(dsarTimeLayout),
		},
	}, recordMaps(documents), "the trashed file must be excluded, only the live one disclosed")

	for _, f := range documents.Records[0].Fields {
		assert.NotContains(t, f.Value, "dsar-test/", "storage_key must never leave the export")
	}

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Ackerknecht")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact, let alone its documents")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Ackerknecht")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")
}

func TestSearchByQuery_ContactAdvisoryProtocols_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Advisory Protocol Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Advisory Protocol Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "AdvisorAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Sieglinde",
		"last_name":  "Falkenrath",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	products := []byte(`[{"id":"p1","name":"Global Equity ETF","riskClass":4,"risks":"Marktrisiko","recommended":true}]`)

	// Older, finalized protocol -- must appear first in a chronological export.
	oldID := testutil.SeedRow(t, pool, "advisory_protocols", map[string]any{
		"tenant_id":           tenantOwn,
		"contact_id":          contactID,
		"created_by":          agentID,
		"status":              "finalized",
		"date":                "2025-01-15",
		"advisor":             "Herr Bergmann",
		"marital_status":      "verheiratet",
		"known_asset_classes": []string{"stocks", "etf"},
		"investment_purpose":  []string{"retirement", "growth"},
		"risk_class":          4,
		"products":            products,
		"warnings_given":      []string{"risk", "costs"},
		"internal_notes":      "Kunde wirkt unsicher, im Folgetermin nochmal Risiko erklären.",
		"monthly_net_income":  4200.50,
		"created_at":          "2025-01-15T09:00:00Z",
	})
	defer testutil.CleanupRow(t, pool, "advisory_protocols", oldID)

	// Newer draft protocol -- must appear second.
	newID := testutil.SeedRow(t, pool, "advisory_protocols", map[string]any{
		"tenant_id":      tenantOwn,
		"contact_id":     contactID,
		"created_by":     agentID,
		"status":         "draft",
		"date":           "2026-03-10",
		"advisor":        "Frau Bergmann",
		"risk_class":     3,
		"products":       []byte(`[]`),
		"internal_notes": "Vertrauliche Einschätzung: eher risikoscheu trotz Angabe.",
		"created_at":     "2026-03-10T09:00:00Z",
	})
	defer testutil.CleanupRow(t, pool, "advisory_protocols", newID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Falkenrath")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	p := persons[0]

	var protocolModules []DSARModule
	for _, m := range p.Modules {
		if strings.HasPrefix(m.Module, "Beratungsprotokoll ") {
			protocolModules = append(protocolModules, m)
		}
	}
	require.Len(t, protocolModules, 2, "one module per protocol, have %v", moduleNames(p))

	assert.Contains(t, protocolModules[0].Module, "2025-01-15", "the older protocol must be disclosed first")
	assert.Contains(t, protocolModules[0].Module, "abgeschlossen")
	assert.Contains(t, protocolModules[1].Module, "2026-03-10", "the newer protocol must be disclosed second")
	assert.Contains(t, protocolModules[1].Module, "Entwurf")

	oldFields := fieldValueMap(t, protocolModules[0])
	assert.Equal(t, "2025-01-15", oldFields["Datum"])
	assert.Equal(t, "Herr Bergmann", oldFields["Berater"])
	assert.Equal(t, "verheiratet", oldFields["Familienstand"])
	assert.Equal(t, "stocks, etf", oldFields["Bekannte Anlageklassen"], "an array column must render as readable text, not Postgres array syntax")
	assert.Equal(t, "retirement, growth", oldFields["Anlagezweck"])
	assert.Equal(t, "risk, costs", oldFields["Erteilte Warnhinweise"])
	assert.Equal(t, "4200.50 EUR", oldFields["Monatliches Nettoeinkommen"])
	assert.Equal(t, "Global Equity ETF (SRI 4, empfohlen)", oldFields["Besprochene Produkte"],
		"a JSONB product list must render as readable text, not raw JSON")

	for _, m := range protocolModules {
		fields := fieldValueMap(t, m)
		_, hasInternalNotes := fields["Interne Notizen"]
		assert.False(t, hasInternalNotes, "internal_notes must never appear as a labeled field")
		for label, value := range fields {
			assert.NotContains(t, value, "unsicher", "internal note content must not leak via any field (checked %s)", label)
			assert.NotContains(t, value, "Vertrauliche Einschätzung", "internal note content must not leak via any field (checked %s)", label)
		}
	}

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Falkenrath")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact, let alone its advisory protocols")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Falkenrath")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")
}

func TestSearchByQuery_ContactAdvisoryProtocols_NoneIsNoModule_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Advisory Protocol None Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "NoAdvisorAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Waldemar",
		"last_name":  "Ohnehistorie",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Ohnehistorie")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	for _, m := range persons[0].Modules {
		assert.NotContains(t, m.Module, "Beratungsprotokoll", "a contact without any protocol must carry no advisory module at all")
	}
}

func TestSearchByQuery_ContactHelpdeskMessages_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Helpdesk Messages Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Helpdesk Messages Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "MessagesAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Isolde",
		"last_name":  "Winterfeld",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":    tenantOwn,
		"subject":      "Lieferverzug",
		"requester_id": agentID,
		"contact_id":   contactID,
	})
	// ticket_messages.ticket_id carries ON DELETE CASCADE from tickets, no
	// separate cleanup needed for the message rows below.
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	customerMsgID := testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": tenantOwn,
		"ticket_id": ticketID,
		"author_id": agentID,
		"body":      "Wo bleibt meine Lieferung?",
		"internal":  false,
	})

	// Internal note about the same contact -- must NOT be disclosed.
	testutil.SeedRow(t, pool, "ticket_messages", map[string]any{
		"tenant_id": tenantOwn,
		"ticket_id": ticketID,
		"author_id": agentID,
		"body":      "Kunde ist bereits zweimal eskaliert, bitte mit Vorsicht behandeln.",
		"internal":  true,
	})

	var customerAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM ticket_messages WHERE id = $1`, customerMsgID).Scan(&customerAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Winterfeld")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	p := persons[0]

	messages := dsarModule(t, p, "Helpdesk-Nachrichten")
	assert.Equal(t, []string{"Ticket", "Nachricht", "Datum"}, messages.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Ticket":    "Lieferverzug",
			"Nachricht": "Wo bleibt meine Lieferung?",
			"Datum":     customerAt.Format(dsarTimeLayout),
		},
	}, recordMaps(messages), "the internal note must be excluded, only the customer-facing message disclosed")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Winterfeld")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact, let alone its helpdesk messages")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Winterfeld")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")
}

func TestSearchByQuery_ContactHelpdeskMessages_Truncation_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Helpdesk Messages Truncation Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "TruncAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Roswitha",
		"last_name":  "Federkiel",
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	ticketID := testutil.SeedRow(t, pool, "tickets", map[string]any{
		"tenant_id":    tenantOwn,
		"subject":      "Vielschreiber",
		"requester_id": agentID,
		"contact_id":   contactID,
	})
	defer testutil.CleanupRow(t, pool, "tickets", ticketID)

	// dsarMaxRows+5 messages, oldest to newest by creation offset -- enough to
	// force truncation and prove which end gets dropped.
	ctxSystem := testutil.WithSystemCtx(context.Background())
	const total = dsarMaxRows + 5
	for i := range total {
		_, err := pool.Exec(ctxSystem,
			`INSERT INTO ticket_messages (tenant_id, ticket_id, author_id, body, internal, created_at)
			 VALUES ($1, $2, $3, $4, false, NOW() - make_interval(mins => $5))`,
			tenantOwn, ticketID, agentID, fmt.Sprintf("Nachricht Nr. %d", i), total-i)
		require.NoError(t, err)
	}

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Federkiel")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	messages := dsarModule(t, persons[0], "Helpdesk-Nachrichten")
	require.Len(t, messages.Records, dsarMaxRows+1, "dsarMaxRows disclosed messages plus one truncation marker")
	assert.Contains(t, messages.Records[0].Fields[1].Value, "gekürzt",
		"a truncation must be visible in the export, not a silent drop")
	assert.Equal(t, fmt.Sprintf("Nachricht Nr. %d", total-1), messages.Records[len(messages.Records)-1].Fields[1].Value,
		"truncation must keep the newest messages, not the oldest")
}

func TestSearchByQuery_ContactFormSubmissions_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Form Submissions Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Form Submissions Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	agentID := seedDSARUser(t, pool, tenantOwn, "Dsar", "FormAgent", true)
	defer testutil.CleanupRow(t, pool, "users", agentID)

	email := fmt.Sprintf("gustav.%s@formkontakt.invalid", uuid.New())
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Gustav",
		"last_name":  "Wolkenreiter",
		"email":      email,
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	fields := []byte(`[
		{"id":"f_name","type":"text","label":"Name"},
		{"id":"f_email","type":"email","label":"E-Mail","role":"requester_email"}
	]`)
	schemaID := testutil.SeedRow(t, pool, "form_schemas", map[string]any{
		"tenant_id":  tenantOwn,
		"title":      "Kontaktformular",
		"fields":     fields,
		"is_public":  true,
		"created_by": agentID,
	})
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaID)

	base := time.Now().UTC().Truncate(time.Minute).Add(-24 * time.Hour)

	// Matches: the email-typed field's answer is the contact's own address,
	// compared case-insensitively.
	matchAnswers := fmt.Appendf(nil, `{"f_name":"Gustav Wolkenreiter","f_email":"%s"}`, strings.ToUpper(email))
	matchID := testutil.SeedRow(t, pool, "form_submissions", map[string]any{
		"tenant_id":      tenantOwn,
		"form_schema_id": schemaID,
		"answers":        matchAnswers,
		"submitted_at":   base,
	})
	defer testutil.CleanupRow(t, pool, "form_submissions", matchID)

	// Does not match: a different address in the same email field.
	otherAnswers := []byte(`{"f_name":"Jemand Anders","f_email":"jemand@andere.invalid"}`)
	otherID := testutil.SeedRow(t, pool, "form_submissions", map[string]any{
		"tenant_id":      tenantOwn,
		"form_schema_id": schemaID,
		"answers":        otherAnswers,
		"submitted_at":   base.Add(-time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "form_submissions", otherID)

	// Does not match: schema deleted, form_schema_id set NULL by ON DELETE
	// SET NULL -- there is no schema left to say which field held an email,
	// so this submission must be excluded rather than guessed at.
	orphanSchemaID := testutil.SeedRow(t, pool, "form_schemas", map[string]any{
		"tenant_id":  tenantOwn,
		"title":      "Verschwundenes Formular",
		"fields":     fields,
		"created_by": agentID,
	})
	orphanAnswers := fmt.Appendf(nil, `{"f_name":"Gustav Wolkenreiter","f_email":"%s"}`, email)
	orphanID := testutil.SeedRow(t, pool, "form_submissions", map[string]any{
		"tenant_id":      tenantOwn,
		"form_schema_id": orphanSchemaID,
		"answers":        orphanAnswers,
		"submitted_at":   base.Add(-2 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "form_submissions", orphanID)
	testutil.CleanupRow(t, pool, "form_schemas", orphanSchemaID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	var matchSubmittedAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT submitted_at FROM form_submissions WHERE id = $1`, matchID).Scan(&matchSubmittedAt))

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Wolkenreiter")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	p := persons[0]

	mod := dsarModule(t, p, "Formulareinreichungen")
	assert.Equal(t, []string{"Formular", "Datum", "Feld", "Wert"}, mod.Columns)
	assert.Equal(t, []map[string]string{
		{"Formular": "Kontaktformular", "Datum": matchSubmittedAt.Format(dsarTimeLayout), "Feld": "Name", "Wert": "Gustav Wolkenreiter"},
		{"Formular": "Kontaktformular", "Datum": matchSubmittedAt.Format(dsarTimeLayout), "Feld": "E-Mail", "Wert": strings.ToUpper(email)},
	}, recordMaps(mod),
		"only the submission whose email field matches the contact appears; the orphaned and other-address submissions must not")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Wolkenreiter")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this contact, let alone its form submissions")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Wolkenreiter")
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
	require.Len(t, active.Modules, 2, "Benutzerkonto plus the always-present Kontosicherheit status, no other module has data for a bare user")
	assert.Equal(t, "Benutzerkonto", active.Modules[0].Module)
	assert.Equal(t, []string{"Feld", "Wert"}, active.Modules[0].Columns)
	assert.Equal(t, "Kontosicherheit", active.Modules[1].Module)

	var activeCreatedAt time.Time
	require.NoError(t, pool.QueryRow(ctxOwn, `SELECT created_at FROM users WHERE id = $1`, activeID).Scan(&activeCreatedAt))
	fields := fieldValueMap(t, active.Modules[0])
	assert.Equal(t, "Wendelin Falkenhorst", fields["Name"])
	assert.Equal(t, "Ja", fields["Aktiv"])
	assert.Equal(t, activeCreatedAt.Format(dsarTimeLayout), fields["Erstellt"])

	assert.Equal(t, "Nein", fieldValueMap(t, byID[inactiveID.String()].Modules[0])["Aktiv"],
		"is_active = false must render as Nein")
}

func TestSearchByQuery_UserChatMessages_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Chat Messages Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Chat Messages Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Odilo", "Sturmfeld", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherParticipantID := seedDSARUser(t, pool, tenantOwn, "Petronella", "Sturmfeld", true)
	defer testutil.CleanupRow(t, pool, "users", otherParticipantID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Projekt Nachtfalter",
		"created_by": subjectID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	ownMsgID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Ich kuemmere mich um den Bericht.",
		"created_by": subjectID,
	})
	// Message of the OTHER participant in the same room — must not be
	// disclosed when we search for the subject.
	testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Danke, bis morgen dann.",
		"created_by": otherParticipantID,
	})

	var ownAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM messages WHERE id = $1`, ownMsgID).Scan(&ownAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Odilo")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	messages := dsarModule(t, persons[0], "Chat-Nachrichten")
	assert.Equal(t, []string{"Kanal", "Nachricht", "Datum"}, messages.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Kanal":     "Projekt Nachtfalter",
			"Nachricht": "Ich kuemmere mich um den Bericht.",
			"Datum":     ownAt.Format(dsarTimeLayout),
		},
	}, recordMaps(messages), "only the subject's own message must appear, not the other participant's")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Odilo")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their chat history")

	forged, err := SearchByQuery(ctxOther, pool, tenantOwn, "Odilo")
	require.NoError(t, err)
	assert.Empty(t, forged, "RLS, not the tenantID argument, is the boundary")
}

func TestSearchByQuery_UserChatMemberships_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Chat Memberships Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Chat Memberships Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Reinhild", "Kranichfeld", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Vertrieb DACH",
		"created_by": subjectID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	// channel_memberships has no surrogate id (composite PK channel_id,
	// user_id) — SeedRow's "RETURNING id" would fail, insert directly.
	ctxSystem := testutil.WithSystemCtx(context.Background())
	var joinedAt time.Time
	require.NoError(t, pool.QueryRow(ctxSystem,
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id, role)
		 VALUES ($1, $2, $3, 'admin') RETURNING joined_at`,
		channelID, subjectID, tenantOwn).Scan(&joinedAt))
	defer func() {
		_, _ = pool.Exec(ctxSystem, `DELETE FROM channel_memberships WHERE channel_id = $1 AND user_id = $2`, channelID, subjectID)
	}()

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Kranichfeld")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	memberships := dsarModule(t, persons[0], "Chat-Kanalmitgliedschaften")
	assert.Equal(t, []string{"Kanal", "Rolle", "Beigetreten"}, memberships.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Kanal":       "Vertrieb DACH",
			"Rolle":       "admin",
			"Beigetreten": joinedAt.Format(dsarTimeLayout),
		},
	}, recordMaps(memberships))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Kranichfeld")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their channel memberships")
}

func TestSearchByQuery_UserTasks_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Tasks Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Tasks Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Siegwart", "Muehlbach", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Traudel", "Muehlbach", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	projectID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "Projekt Muehlbach",
		"project_key": fmt.Sprintf("MB%d", uuid.New().ID()%1000),
		"created_by":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projectID)
	statusID := testutil.SeedRow(t, pool, "project_statuses", map[string]any{
		"tenant_id":  tenantOwn,
		"project_id": projectID,
		"name":       "In Arbeit",
	})
	defer testutil.CleanupRow(t, pool, "project_statuses", statusID)

	// Task the subject created AND is assigned to — both roles on one row.
	ownTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectID,
		"task_number": 1,
		"title":       "Angebot pruefen",
		"status_id":   statusID,
		"assignee_id": subjectID,
		"created_by":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", ownTaskID)
	// Task assigned to the subject but created by someone else.
	assignedTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 2,
		"title":       "Rechnung freigeben",
		"assignee_id": subjectID,
		"created_by":  otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", assignedTaskID)
	// Task neither created nor assigned to the subject — must not appear.
	unrelatedTaskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 3,
		"title":       "Fremde Aufgabe",
		"assignee_id": otherUserID,
		"created_by":  otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", unrelatedTaskID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Siegwart")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	tasks := dsarModule(t, persons[0], "Aufgaben")
	assert.Equal(t, []string{"Titel", "Status", "Rolle", "Erstellt"}, tasks.Columns)
	rows := recordMaps(tasks)
	require.Len(t, rows, 2, "only the two tasks touching the subject, not the unrelated one")

	byTitle := map[string]map[string]string{}
	for _, r := range rows {
		byTitle[r["Titel"]] = r
	}
	assert.Equal(t, "In Arbeit", byTitle["Angebot pruefen"]["Status"])
	assert.Equal(t, "Ersteller, Zugewiesen", byTitle["Angebot pruefen"]["Rolle"])
	assert.Equal(t, "", byTitle["Rechnung freigeben"]["Status"], "no status assigned renders empty, not a placeholder")
	assert.Equal(t, "Zugewiesen", byTitle["Rechnung freigeben"]["Rolle"])
	assert.NotContains(t, byTitle, "Fremde Aufgabe")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Siegwart")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their tasks")
}

func TestSearchByQuery_UserTaskComments_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Task Comments Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Task Comments Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Ursula", "Rabenstein", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Volkmar", "Rabenstein", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 1,
		"title":       "Vertrag abschliessen",
		"created_by":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	ownCommentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantOwn,
		"task_id":   taskID,
		"author_id": subjectID,
		"content":   "Ich schicke den Entwurf morgen.",
	})
	defer testutil.CleanupRow(t, pool, "task_comments", ownCommentID)
	// Comment of another author on the same task — must not be disclosed.
	testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantOwn,
		"task_id":   taskID,
		"author_id": otherUserID,
		"content":   "Klingt gut, danke.",
	})

	var ownAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM task_comments WHERE id = $1`, ownCommentID).Scan(&ownAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Ursula")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	comments := dsarModule(t, persons[0], "Aufgaben-Kommentare")
	assert.Equal(t, []string{"Aufgabe", "Kommentar", "Datum"}, comments.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Aufgabe":   "Vertrag abschliessen",
			"Kommentar": "Ich schicke den Entwurf morgen.",
			"Datum":     ownAt.Format(dsarTimeLayout),
		},
	}, recordMaps(comments), "only the subject's own comment must appear, not the other author's")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Ursula")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their task comments")
}

func TestSearchByQuery_UserTimeEntries_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Time Entries Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Time Entries Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Waltraud", "Eisenberg", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 1,
		"title":       "Migration testen",
		"created_by":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	// Two entries in the same month (2400s + 3600s = 6000s = 1.6667h, rounds
	// to 1.7h) and one in an earlier month (7200s = 2.0h), to prove the
	// GROUP BY aggregates correctly rather than listing three raw rows.
	e1 := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":        tenantOwn,
		"task_id":          taskID,
		"user_id":          subjectID,
		"started_at":       "2026-08-05T09:00:00Z",
		"duration_seconds": 2400,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", e1)
	e2 := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":        tenantOwn,
		"task_id":          taskID,
		"user_id":          subjectID,
		"started_at":       "2026-08-12T09:00:00Z",
		"duration_seconds": 3600,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", e2)
	e3 := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":        tenantOwn,
		"task_id":          taskID,
		"user_id":          subjectID,
		"started_at":       "2026-05-05T09:00:00Z",
		"duration_seconds": 7200,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", e3)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Waltraud")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	entries := dsarModule(t, persons[0], "Zeiterfassung (aggregiert pro Monat)")
	assert.Equal(t, []string{"Monat", "Eintraege", "Dauer"}, entries.Columns)
	assert.Equal(t, []map[string]string{
		{"Monat": "2026-08", "Eintraege": "2", "Dauer": "1.7 Std."},
		{"Monat": "2026-05", "Eintraege": "1", "Dauer": "2.0 Std."},
	}, recordMaps(entries), "entries must be aggregated per month, newest month first")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Waltraud")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their tracked time")
}

func TestSearchByQuery_UserCalendarEvents_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Calendar Events Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Calendar Events Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Adalbert", "Wolkenstein", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Brunhilde", "Wolkenstein", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	calendarID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Team Kalender",
		"owner_id":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	// Event created by the subject.
	ownEventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": calendarID,
		"title":       "Strategiemeeting",
		"start_time":  "2026-09-01T09:00:00Z",
		"end_time":    "2026-09-01T10:00:00Z",
		"location":    "Raum A",
		"created_by":  subjectID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", ownEventID)

	// Event created by someone else; both the subject and a third person
	// attend — only the subject's own RSVP must appear, not the attendee list.
	sharedEventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": calendarID,
		"title":       "Kundentermin",
		"start_time":  "2026-09-02T14:00:00Z",
		"end_time":    "2026-09-02T15:00:00Z",
		"created_by":  otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", sharedEventID)

	// event_attendees has no surrogate id (composite PK event_id, user_id) —
	// SeedRow's "RETURNING id" would fail, insert directly.
	ctxSystem := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctxSystem,
		`INSERT INTO event_attendees (event_id, user_id, tenant_id, rsvp_status) VALUES ($1, $2, $3, 'accepted')`,
		sharedEventID, subjectID, tenantOwn)
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(ctxSystem, `DELETE FROM event_attendees WHERE event_id = $1 AND user_id = $2`, sharedEventID, subjectID)
	}()
	_, err = pool.Exec(ctxSystem,
		`INSERT INTO event_attendees (event_id, user_id, tenant_id, rsvp_status) VALUES ($1, $2, $3, 'declined')`,
		sharedEventID, otherUserID, tenantOwn)
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(ctxSystem, `DELETE FROM event_attendees WHERE event_id = $1 AND user_id = $2`, sharedEventID, otherUserID)
	}()

	// Event neither created nor attended by the subject — must not appear.
	unrelatedEventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": calendarID,
		"title":       "Fremder Termin",
		"start_time":  "2026-09-03T09:00:00Z",
		"end_time":    "2026-09-03T10:00:00Z",
		"created_by":  otherUserID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", unrelatedEventID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Adalbert")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	events := dsarModule(t, persons[0], "Kalendertermine")
	assert.Equal(t, []string{"Titel", "Beginn", "Ort", "Rolle"}, events.Columns)
	rows := recordMaps(events)
	require.Len(t, rows, 2, "only the two events touching the subject, not the unrelated one")

	byTitle := map[string]map[string]string{}
	for _, r := range rows {
		byTitle[r["Titel"]] = r
	}
	assert.Equal(t, "Ersteller", byTitle["Strategiemeeting"]["Rolle"])
	assert.Equal(t, "Raum A", byTitle["Strategiemeeting"]["Ort"])
	assert.Equal(t, "Teilnehmer (accepted)", byTitle["Kundentermin"]["Rolle"],
		"only the subject's own RSVP appears, not the other attendee's 'declined'")
	assert.NotContains(t, byTitle, "Fremder Termin")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Adalbert")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their calendar")
}

func TestSearchByQuery_UserCalendarPreferences_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Calendar Prefs Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Calendar Prefs Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Cordula", "Habichtsburg", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	// user_calendar_preferences has no surrogate id (PK user_id) — insert directly.
	ctxSystem := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctxSystem,
		`INSERT INTO user_calendar_preferences (tenant_id, user_id, default_view, week_days, subdivision_code, show_task_deadlines)
		 VALUES ($1, $2, 'month', 7, 'DE-BY', false)`, tenantOwn, subjectID)
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(ctxSystem, `DELETE FROM user_calendar_preferences WHERE user_id = $1`, subjectID)
	}()

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Cordula")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	prefs := fieldValueMap(t, dsarModule(t, persons[0], "Kalendereinstellungen"))
	assert.Equal(t, "month", prefs["Standardansicht"])
	assert.Equal(t, "7", prefs["Wochentage"])
	assert.Equal(t, "DE-BY", prefs["Feiertagsregion"])
	assert.Equal(t, "Nein", prefs["Aufgaben-Termine anzeigen"])

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Cordula")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their calendar preferences")
}

func TestSearchByQuery_UserNotifications_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Notifications Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Notifications Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Dietlind", "Amselgrund", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Eberhart", "Amselgrund", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	ownNotifID := testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      tenantOwn,
		"user_id":        subjectID,
		"event_type_key": "task.assigned",
		"module_id":      "work",
		"title":          "Neue Aufgabe zugewiesen",
		"body":           "Angebot pruefen wurde dir zugewiesen.",
		"is_read":        true,
	})
	defer testutil.CleanupRow(t, pool, "notifications", ownNotifID)
	// Notification of another user — must not be disclosed.
	testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      tenantOwn,
		"user_id":        otherUserID,
		"event_type_key": "task.assigned",
		"module_id":      "work",
		"title":          "Fremde Benachrichtigung",
	})

	var ownAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM notifications WHERE id = $1`, ownNotifID).Scan(&ownAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Dietlind")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	notifs := dsarModule(t, persons[0], "Benachrichtigungen")
	assert.Equal(t, []string{"Titel", "Text", "Datum", "Gelesen"}, notifs.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Titel":   "Neue Aufgabe zugewiesen",
			"Text":    "Angebot pruefen wurde dir zugewiesen.",
			"Datum":   ownAt.Format(dsarTimeLayout),
			"Gelesen": "Ja",
		},
	}, recordMaps(notifs), "only the subject's own notifications, not another user's")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Dietlind")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their notifications")
}

func TestSearchByQuery_UserNotificationPreferences_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Notification Prefs Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Notification Prefs Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Fridolin", "Buchenhain", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	prefID := testutil.SeedRow(t, pool, "notification_preferences", map[string]any{
		"tenant_id":    tenantOwn,
		"user_id":      subjectID,
		"module_id":    "helpdesk",
		"in_app":       true,
		"desktop_push": false,
		"sound":        "chime",
	})
	defer testutil.CleanupRow(t, pool, "notification_preferences", prefID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Fridolin")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	prefs := dsarModule(t, persons[0], "Benachrichtigungseinstellungen")
	assert.Equal(t, []string{"Ereignistyp", "Modul", "In-App", "Desktop", "Ton"}, prefs.Columns)
	assert.Equal(t, []map[string]string{
		{"Ereignistyp": "", "Modul": "helpdesk", "In-App": "Ja", "Desktop": "Nein", "Ton": "chime"},
	}, recordMaps(prefs))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Fridolin")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their notification preferences")
}

func TestSearchByQuery_UserNotificationQuietHours_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Quiet Hours Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Quiet Hours Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Gerlinde", "Steinfurth", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	qhID := testutil.SeedRow(t, pool, "notification_quiet_hours", map[string]any{
		"tenant_id":  tenantOwn,
		"user_id":    subjectID,
		"start_time": "20:00",
		"end_time":   "07:00",
		"enabled":    true,
		"manual_dnd": false,
	})
	defer testutil.CleanupRow(t, pool, "notification_quiet_hours", qhID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Gerlinde")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	qh := fieldValueMap(t, dsarModule(t, persons[0], "Ruhezeiten"))
	assert.Equal(t, "Ja", qh["Aktiviert"])
	assert.Equal(t, "20:00:00", qh["Von"])
	assert.Equal(t, "07:00:00", qh["Bis"])
	assert.Equal(t, "Nein", qh["Manuell aktiv"])

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Gerlinde")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their quiet hours")
}

func TestSearchByQuery_UserNotificationMutes_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Notification Mutes Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Notification Mutes Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Hartwig", "Distelmeier", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	muteID := testutil.SeedRow(t, pool, "notification_mutes", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     subjectID,
		"module_id":   "chat",
		"resource_id": "channel-123",
	})
	defer testutil.CleanupRow(t, pool, "notification_mutes", muteID)

	var mutedAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM notification_mutes WHERE id = $1`, muteID).Scan(&mutedAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Hartwig")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	mutes := dsarModule(t, persons[0], "Stummgeschaltete Ressourcen")
	assert.Equal(t, []string{"Modul", "Ressource", "Seit"}, mutes.Columns)
	assert.Equal(t, []map[string]string{
		{"Modul": "chat", "Ressource": "channel-123", "Seit": mutedAt.Format(dsarTimeLayout)},
	}, recordMaps(mutes))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Hartwig")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their muted resources")
}

func TestSearchByQuery_UserHRProfile_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Profile Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Profile Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	managerID := seedDSARUser(t, pool, tenantOwn, "Roswitha", "Kessler", true)
	defer testutil.CleanupRow(t, pool, "users", managerID)
	subjectID := seedDSARUser(t, pool, tenantOwn, "Falk", "Brennecke", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	profileID := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
		"tenant_id":               tenantOwn,
		"user_id":                 subjectID,
		"department":              "Fuhrpark",
		"position_title":          "Disponent",
		"contract_type":           "part_time",
		"work_days_per_week":      4,
		"annual_leave_days":       24,
		"manager_user_id":         managerID,
		"start_date":              "2024-03-01",
		"emergency_contact_name":  "Ines Brennecke",
		"emergency_contact_phone": "+49 151 2233445",
		"address_street":          "Kastanienallee 9",
		"address_city":            "Leipzig",
		"address_postal_code":     "04109",
		"address_country":         "DE",
		"hourly_rate":             "21.50",
		"status":                  "active",
		"is_minor":                false,
	})
	defer testutil.CleanupRow(t, pool, "hr_employee_profiles", profileID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Falk")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	profile := fieldValueMap(t, dsarModule(t, persons[0], "Personalprofil"))
	assert.Equal(t, "Fuhrpark", profile["Abteilung"])
	assert.Equal(t, "Disponent", profile["Position"])
	assert.Equal(t, "part_time", profile["Vertragsart"])
	assert.Equal(t, "4", profile["Arbeitstage pro Woche"])
	assert.Equal(t, "24", profile["Urlaubsanspruch (Tage/Jahr)"])
	assert.Equal(t, "Roswitha Kessler", profile["Vorgesetzte(r)"])
	assert.Equal(t, "2024-03-01", profile["Eintrittsdatum"])
	assert.Equal(t, "Ines Brennecke", profile["Notfallkontakt"])
	assert.Equal(t, "+49 151 2233445", profile["Notfallkontakt Telefon"])
	assert.Equal(t, "Kastanienallee 9, 04109 Leipzig, DE", profile["Adresse"])
	assert.Equal(t, "21.50 EUR", profile["Stundenlohn"])
	assert.Equal(t, "active", profile["Status"])
	assert.Equal(t, "Nein", profile["Minderjährig"])
	assert.Equal(t, "", profile["Austrittsdatum"], "an active employee has no exit date")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Falk")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their HR profile")
}

func TestSearchByQuery_UserHRProfile_NoneIsNoModule_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Profile None Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Gudrun", "Vossberg", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Gudrun")
	require.NoError(t, err)
	require.Len(t, persons, 1)
	assert.NotContains(t, moduleNames(persons[0]), "Personalprofil", "a user with no HR record must carry no HR profile module")
}

// TestSearchByQuery_UserHRDocuments_Integration also pins the RLS fix:
// hr_employee_documents carries a role-gated policy (hr_document_access) on
// top of tenant_isolation, so this seeds and queries the row with a caller
// context that holds neither an HR role nor a user identity -- exactly what
// DSARSearch's caller looks like -- and would return zero rows without the
// sysctx.With wrap in hrEmployeeDocumentsModule.
func TestSearchByQuery_UserHRDocuments_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Documents Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Documents Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Elmar", "Habicht", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	categoryID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Arbeitsvertrag",
		"key":        fmt.Sprintf("contract-%s", uuid.New()),
		"visibility": "hr_only",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", categoryID)

	docID := testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
		"tenant_id":   tenantOwn,
		"employee_id": subjectID,
		"category_id": categoryID,
		"uploaded_by": subjectID,
		"title":       "Arbeitsvertrag 2024",
		"file_name":   "vertrag_2024.pdf",
		"file_size":   "184 KB",
	})
	defer testutil.CleanupRow(t, pool, "hr_employee_documents", docID)

	var uploadedAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM hr_employee_documents WHERE id = $1`, docID).Scan(&uploadedAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Elmar")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	docs := dsarModule(t, persons[0], "Personaldokumente (Metadaten)")
	assert.Equal(t, []string{"Titel", "Dateiname", "Kategorie", "Größe", "Hochgeladen am", "Läuft ab am"}, docs.Columns)
	assert.Equal(t, []map[string]string{{
		"Titel":          "Arbeitsvertrag 2024",
		"Dateiname":      "vertrag_2024.pdf",
		"Kategorie":      "Arbeitsvertrag",
		"Größe":          "184 KB",
		"Hochgeladen am": uploadedAt.Format(dsarTimeLayout),
		"Läuft ab am":    "",
	}}, recordMaps(docs), "an hr_only-visibility document must still surface to a DSAR caller with no HR role")

	for _, r := range docs.Records {
		for _, f := range r.Fields {
			assert.NotContains(t, f.Value, "/", "no path or URL segment may appear in document metadata")
		}
	}

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Elmar")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their HR documents")
}

func TestSearchByQuery_UserHRProfileChangeRequests_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Change Requests Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Change Requests Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Petra", "Sonnenschein", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	reqID := testutil.SeedRow(t, pool, "hr_profile_change_requests", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     subjectID,
		"drawer":      "personal",
		"field":       "address_city",
		"field_label": "Stadt",
		"old_value":   "Erfurt",
		"new_value":   "Jena",
		"status":      "pending",
		"reason":      "Umzug",
	})
	defer testutil.CleanupRow(t, pool, "hr_profile_change_requests", reqID)

	var createdAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at FROM hr_profile_change_requests WHERE id = $1`, reqID).Scan(&createdAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Petra")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	reqs := dsarModule(t, persons[0], "Änderungsanträge (Profil)")
	assert.Equal(t, []string{
		"Bereich", "Feld", "Alter Wert", "Neuer Wert", "Status", "Grund", "Erstellt am", "Entschieden am",
	}, reqs.Columns)
	assert.Equal(t, []map[string]string{{
		"Bereich":        "personal",
		"Feld":           "Stadt",
		"Alter Wert":     "Erfurt",
		"Neuer Wert":     "Jena",
		"Status":         "pending",
		"Grund":          "Umzug",
		"Erstellt am":    createdAt.Format(dsarTimeLayout),
		"Entschieden am": "",
	}}, recordMaps(reqs))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Petra")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their change requests")
}

func TestSearchByQuery_UserHRLeaveRequests_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Leave Requests Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Leave Requests Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	approverID := seedDSARUser(t, pool, tenantOwn, "Marlene", "Ostwald", true)
	defer testutil.CleanupRow(t, pool, "users", approverID)
	subjectID := seedDSARUser(t, pool, tenantOwn, "Karsten", "Wiegand", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	leaveTypeID := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Erholungsurlaub",
		"key":       fmt.Sprintf("vacation-%s", uuid.New()),
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_types", leaveTypeID)

	approvedAt := time.Now().UTC().Truncate(time.Minute)
	reqID := testutil.SeedRow(t, pool, "hr_leave_requests", map[string]any{
		"tenant_id":        tenantOwn,
		"employee_id":      subjectID,
		"leave_type_id":    leaveTypeID,
		"start_date":       "2025-06-02",
		"end_date":         "2025-06-06",
		"total_days":       "5.0",
		"reason":           "Sommerurlaub",
		"status":           "approved",
		"approved_by":      approverID,
		"approval_comment": "Genehmigt",
		"approved_at":      approvedAt,
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_requests", reqID)

	// Read back rather than format the pre-insertion Go time: pgx scans a
	// timestamptz into the session's timezone, not necessarily UTC, so the
	// only value that is guaranteed to match what SearchByQuery discloses is
	// the one read back through the same driver.
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT approved_at FROM hr_leave_requests WHERE id = $1`, reqID).Scan(&approvedAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Karsten")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	mod := dsarModule(t, persons[0], "Urlaubsanträge")
	assert.Equal(t, []string{
		"Art", "Zeitraum", "Tage", "Status", "Grund", "Entscheidungskommentar", "Genehmigt von", "Genehmigt am",
	}, mod.Columns)
	assert.Equal(t, []map[string]string{{
		"Art":                    "Erholungsurlaub",
		"Zeitraum":               "2025-06-02 – 2025-06-06",
		"Tage":                   "5.0",
		"Status":                 "approved",
		"Grund":                  "Sommerurlaub",
		"Entscheidungskommentar": "Genehmigt",
		"Genehmigt von":          "Marlene Ostwald",
		"Genehmigt am":           approvedAt.Format(dsarTimeLayout),
	}}, recordMaps(mod))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Karsten")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their leave requests")
}

func TestSearchByQuery_UserHRLeaveBalances_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Leave Balances Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Leave Balances Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Ilonka", "Freimuth", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	bal2024 := testutil.SeedRow(t, pool, "hr_leave_balances", map[string]any{
		"tenant_id":    tenantOwn,
		"employee_id":  subjectID,
		"year":         2024,
		"entitlement":  "28.0",
		"carried_over": "2.0",
		"used":         "30.0",
		"remaining":    "0.0",
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_balances", bal2024)

	bal2025 := testutil.SeedRow(t, pool, "hr_leave_balances", map[string]any{
		"tenant_id":            tenantOwn,
		"employee_id":          subjectID,
		"year":                 2025,
		"entitlement":          "28.0",
		"carried_over":         "0.0",
		"used":                 "5.0",
		"remaining":            "23.0",
		"carryover_expires_at": "2025-03-31",
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_balances", bal2025)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Ilonka")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	var idx2024, idx2025 = -1, -1
	for i, m := range persons[0].Modules {
		switch m.Module {
		case "Urlaubskonto 2024":
			idx2024 = i
		case "Urlaubskonto 2025":
			idx2025 = i
		}
	}
	require.NotEqual(t, -1, idx2024, "2024 balance module missing")
	require.NotEqual(t, -1, idx2025, "2025 balance module missing")
	assert.Less(t, idx2024, idx2025, "years must disclose oldest first, one module per year")

	bal := fieldValueMap(t, dsarModule(t, persons[0], "Urlaubskonto 2025"))
	assert.Equal(t, "28.0", bal["Anspruch (Tage)"])
	assert.Equal(t, "0.0", bal["Übertrag (Tage)"])
	assert.Equal(t, "5.0", bal["Verwendet (Tage)"])
	assert.Equal(t, "23.0", bal["Verbleibend (Tage)"])
	assert.Equal(t, "2025-03-31", bal["Übertrag läuft ab am"])

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Ilonka")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their leave balances")
}

// TestSearchByQuery_UserHRWorkTime_Integration also pins the migration 000248
// fix: a superseded original must not add its minutes on top of the
// correction that replaced it, or a corrected day would count twice.
func TestSearchByQuery_UserHRWorkTime_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Work Time Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Work Time Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Torben", "Landmann", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	may := time.Date(2025, 5, 12, 8, 0, 0, 0, time.UTC)
	entryMay := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id":               uuid.New(),
		"tenant_id":        tenantOwn,
		"employee_id":      subjectID,
		"clock_in":         may,
		"clock_out":        may.Add(8 * time.Hour),
		"net_work_minutes": 480,
		"status":           "completed",
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", entryMay)

	june := time.Date(2025, 6, 3, 8, 0, 0, 0, time.UTC)
	supersededID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id":               uuid.New(),
		"tenant_id":        tenantOwn,
		"employee_id":      subjectID,
		"clock_in":         june,
		"clock_out":        june.Add(6 * time.Hour),
		"net_work_minutes": 360,
		"status":           "superseded",
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", supersededID)

	correctionID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id":                 uuid.New(),
		"tenant_id":          tenantOwn,
		"employee_id":        subjectID,
		"clock_in":           june,
		"clock_out":          june.Add(8 * time.Hour),
		"net_work_minutes":   480,
		"status":             "correction_approved",
		"is_correction":      true,
		"original_entry_id":  supersededID,
		"correction_reason":  "Vergessen auszustempeln",
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", correctionID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Torben")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	mod := dsarModule(t, persons[0], "Arbeitszeit (aggregiert pro Monat)")
	byMonth := make(map[string]map[string]string)
	for _, r := range recordMaps(mod) {
		byMonth[r["Monat"]] = r
	}
	assert.Equal(t, "1", byMonth["2025-05"]["Einträge"])
	assert.Equal(t, "8.0 Std.", byMonth["2025-05"]["Dauer"])
	assert.Equal(t, "1", byMonth["2025-06"]["Einträge"],
		"only the correction counts for June, not the superseded original")
	assert.Equal(t, "8.0 Std.", byMonth["2025-06"]["Dauer"],
		"360+480 would be the double-counted total the migration 000248 comment warns about")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Torben")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their work time")
}

// TestSearchByQuery_UserHRWorkTimeCorrections_Integration pins that a
// correction is disclosed individually with its original and corrected
// times, not folded into the monthly aggregate above.
func TestSearchByQuery_UserHRWorkTimeCorrections_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR HR Work Time Corrections Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR HR Work Time Corrections Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	approverID := seedDSARUser(t, pool, tenantOwn, "Rosalinde", "Herbst", true)
	defer testutil.CleanupRow(t, pool, "users", approverID)
	subjectID := seedDSARUser(t, pool, tenantOwn, "Detlef", "Wachsmuth", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	origIn := time.Date(2025, 4, 8, 8, 0, 0, 0, time.UTC)
	origID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id":               uuid.New(),
		"tenant_id":        tenantOwn,
		"employee_id":      subjectID,
		"clock_in":         origIn,
		"clock_out":        origIn.Add(6 * time.Hour),
		"net_work_minutes": 360,
		"status":           "superseded",
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", origID)

	approvedAt := time.Now().UTC().Truncate(time.Minute)
	corrIn := origIn
	corrOut := origIn.Add(8 * time.Hour)
	corrID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id":                     uuid.New(),
		"tenant_id":              tenantOwn,
		"employee_id":            subjectID,
		"clock_in":               corrIn,
		"clock_out":              corrOut,
		"net_work_minutes":       480,
		"status":                 "correction_approved",
		"is_correction":          true,
		"original_entry_id":      origID,
		"correction_reason":      "Auszeit vergessen",
		"correction_approved_by": approverID,
		"correction_approved_at": approvedAt,
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", corrID)

	// Read every timestamp back rather than format the pre-insertion Go
	// values: pgx scans a timestamptz into the session's timezone, not
	// necessarily UTC, so only a value read back through the same driver is
	// guaranteed to match what SearchByQuery discloses.
	var submittedAt, origOut, corrInReadBack, corrOutReadBack time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at, clock_in, clock_out FROM hr_work_time_entries WHERE id = $1`, corrID).
		Scan(&submittedAt, &corrInReadBack, &corrOutReadBack))
	var origInReadBack time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT clock_in, clock_out FROM hr_work_time_entries WHERE id = $1`, origID).
		Scan(&origInReadBack, &origOut))
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT correction_approved_at FROM hr_work_time_entries WHERE id = $1`, corrID).Scan(&approvedAt))
	corrIn, corrOut = corrInReadBack, corrOutReadBack
	origIn = origInReadBack

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Detlef")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	mod := dsarModule(t, persons[0], "Arbeitszeitkorrekturen")
	assert.Equal(t, []string{
		"Eingereicht am", "Ursprünglicher Beginn", "Ursprüngliches Ende",
		"Korrigierter Beginn", "Korrigiertes Ende", "Grund", "Status", "Genehmigt von", "Genehmigt am",
	}, mod.Columns)
	assert.Equal(t, []map[string]string{{
		"Eingereicht am":        submittedAt.Format(dsarTimeLayout),
		"Ursprünglicher Beginn": origIn.Format(dsarTimeLayout),
		"Ursprüngliches Ende":   origOut.Format(dsarTimeLayout),
		"Korrigierter Beginn":   corrIn.Format(dsarTimeLayout),
		"Korrigiertes Ende":     corrOut.Format(dsarTimeLayout),
		"Grund":                 "Auszeit vergessen",
		"Status":                "correction_approved",
		"Genehmigt von":         "Rosalinde Herbst",
		"Genehmigt am":          approvedAt.Format(dsarTimeLayout),
	}}, recordMaps(mod))

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Detlef")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their work time corrections")
}

func TestSearchByQuery_UserSessions_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Sessions Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Sessions Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Waltraud", "Kirchhoff", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Bernd", "Achterberg", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	now := time.Now().UTC()

	activeToken := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"tenant_id":  tenantOwn,
		"user_id":    subjectID,
		"token_hash": fmt.Sprintf("active-%s", uuid.New()),
		"expires_at": now.Add(24 * time.Hour),
		"revoked":    false,
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", activeToken)
	revokedToken := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"tenant_id":  tenantOwn,
		"user_id":    subjectID,
		"token_hash": fmt.Sprintf("revoked-%s", uuid.New()),
		"expires_at": now.Add(24 * time.Hour),
		"revoked":    true,
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", revokedToken)
	expiredToken := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"tenant_id":  tenantOwn,
		"user_id":    subjectID,
		"token_hash": fmt.Sprintf("expired-%s", uuid.New()),
		"expires_at": now.Add(-24 * time.Hour),
		"revoked":    false,
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", expiredToken)

	sessionActive := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":        tenantOwn,
		"user_id":          subjectID,
		"refresh_token_id": activeToken,
		"device_name":      "MacBook Pro",
		"device_type":      "desktop",
		"ip_address":       "203.0.113.10",
		"location":         "Berlin, DE",
		"last_active_at":   now.Add(-1 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionActive)
	sessionRevoked := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":        tenantOwn,
		"user_id":          subjectID,
		"refresh_token_id": revokedToken,
		"device_name":      "iPhone 15",
		"device_type":      "mobile",
		"ip_address":       "203.0.113.20",
		"location":         "Hamburg, DE",
		"last_active_at":   now.Add(-2 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionRevoked)
	sessionExpired := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":        tenantOwn,
		"user_id":          subjectID,
		"refresh_token_id": expiredToken,
		"device_name":      "Windows PC",
		"device_type":      "desktop",
		"ip_address":       "203.0.113.30",
		"location":         "Munich, DE",
		"last_active_at":   now.Add(-3 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionExpired)
	sessionEnded := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":      tenantOwn,
		"user_id":        subjectID,
		"device_name":    "Old Tablet",
		"device_type":    "mobile",
		"location":       "Cologne, DE",
		"last_active_at": now,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionEnded)

	otherSession := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":      tenantOwn,
		"user_id":        otherUserID,
		"device_name":    "Someone Else's Laptop",
		"device_type":    "desktop",
		"location":       "Bremen, DE",
		"last_active_at": now,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", otherSession)

	// last_active_at round-trips through the DB session's own time zone
	// setting, so the expected strings are read back from Postgres rather
	// than formatted from the Go-side `now` value used to seed them.
	activeAt := querySessionLastActive(t, pool, sessionActive)
	revokedAt := querySessionLastActive(t, pool, sessionRevoked)
	expiredAt := querySessionLastActive(t, pool, sessionExpired)
	endedAt := querySessionLastActive(t, pool, sessionEnded)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Waltraud")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	sessions := dsarModule(t, persons[0], "Sitzungsverlauf")
	assert.Equal(t, []string{"Gerät", "Typ", "IP-Adresse", "Standort", "Letzte Aktivität", "Status"}, sessions.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Gerät": "Old Tablet", "Typ": "mobile", "IP-Adresse": "", "Standort": "Cologne, DE",
			"Letzte Aktivität": endedAt.Format(dsarTimeLayout), "Status": "Beendet",
		},
		{
			"Gerät": "MacBook Pro", "Typ": "desktop", "IP-Adresse": "203.0.113.10", "Standort": "Berlin, DE",
			"Letzte Aktivität": activeAt.Format(dsarTimeLayout), "Status": "Aktiv",
		},
		{
			"Gerät": "iPhone 15", "Typ": "mobile", "IP-Adresse": "203.0.113.20", "Standort": "Hamburg, DE",
			"Letzte Aktivität": revokedAt.Format(dsarTimeLayout), "Status": "Widerrufen",
		},
		{
			"Gerät": "Windows PC", "Typ": "desktop", "IP-Adresse": "203.0.113.30", "Standort": "Munich, DE",
			"Letzte Aktivität": expiredAt.Format(dsarTimeLayout), "Status": "Abgelaufen",
		},
	}, recordMaps(sessions), "only the subject's own sessions, ordered by last activity, each with the right derived status")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Waltraud")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their sessions")
}

func TestSearchByQuery_UserAccountSecurity_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Account Security Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Account Security Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	twoFactorEnabledAt := time.Now().UTC().Add(-30 * 24 * time.Hour)
	subjectID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":             tenantOwn,
		"email":                 fmt.Sprintf("dsar-%s@search.invalid", uuid.New()),
		"password_hash":         "$argon2id$v=19$secret-must-not-leak",
		"first_name":            "Sieglinde",
		"last_name":             "Torwart",
		"is_active":             true,
		"two_factor_enabled":    true,
		"two_factor_enabled_at": twoFactorEnabledAt,
	})
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	pw1 := testutil.SeedRow(t, pool, "password_history", map[string]any{
		"tenant_id":     tenantOwn,
		"user_id":       subjectID,
		"password_hash": "$argon2id$v=19$old-secret-must-not-leak",
	})
	defer testutil.CleanupRow(t, pool, "password_history", pw1)
	pw2 := testutil.SeedRow(t, pool, "password_history", map[string]any{
		"tenant_id":     tenantOwn,
		"user_id":       subjectID,
		"password_hash": "$argon2id$v=19$older-secret-must-not-leak",
	})
	defer testutil.CleanupRow(t, pool, "password_history", pw2)

	rc1 := testutil.SeedRow(t, pool, "recovery_codes", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   subjectID,
		"code_hash": "recovery-hash-1-must-not-leak",
		"used_at":   time.Now().UTC().Add(-10 * 24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "recovery_codes", rc1)
	rc2 := testutil.SeedRow(t, pool, "recovery_codes", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   subjectID,
		"code_hash": "recovery-hash-2-must-not-leak",
	})
	defer testutil.CleanupRow(t, pool, "recovery_codes", rc2)
	rc3 := testutil.SeedRow(t, pool, "recovery_codes", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   subjectID,
		"code_hash": "recovery-hash-3-must-not-leak",
	})
	defer testutil.CleanupRow(t, pool, "recovery_codes", rc3)

	var lastPasswordChange time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT max(created_at) FROM password_history WHERE user_id = $1`, subjectID).Scan(&lastPasswordChange))
	// two_factor_enabled_at round-trips through the DB session's own time
	// zone setting, so it is read back rather than formatted from the
	// Go-side value used to seed it (see querySessionLastActive above).
	var enabledAt time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT two_factor_enabled_at FROM users WHERE id = $1`, subjectID).Scan(&enabledAt))

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Sieglinde")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	security := fieldValueMap(t, dsarModule(t, persons[0], "Kontosicherheit"))
	assert.Equal(t, "Ja", security["Zwei-Faktor-Authentifizierung"])
	assert.Equal(t, enabledAt.Format(dsarTimeLayout), security["Aktiviert am"])
	assert.Equal(t, "2", security["Passwortänderungen (Anzahl)"])
	assert.Equal(t, lastPasswordChange.Format(dsarTimeLayout), security["Letzte Passwortänderung"])
	assert.Equal(t, "3", security["Wiederherstellungscodes (gesamt)"])
	assert.Equal(t, "1", security["Wiederherstellungscodes (genutzt)"])

	for _, v := range security {
		assert.NotContains(t, v, "secret", "no password or recovery-code hash may ever surface in the export")
		assert.NotContains(t, v, "recovery-hash", "no recovery code hash may ever surface in the export")
	}

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Sieglinde")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their account security status")
}

// TestSearchByQuery_UserAccountSecurity_DefaultsWhenNoHistory_Integration pins
// that accountSecurityModule, unlike most modules in this file, never
// returns nil: a fresh account with no password change and no 2FA still
// carries a security status, it just reads "Nein" and zero everywhere.
func TestSearchByQuery_UserAccountSecurity_DefaultsWhenNoHistory_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Account Security Defaults Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Volker", "Nussbaumer", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Volker")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	security := fieldValueMap(t, dsarModule(t, persons[0], "Kontosicherheit"))
	assert.Equal(t, "Nein", security["Zwei-Faktor-Authentifizierung"])
	assert.Equal(t, "", security["Aktiviert am"])
	assert.Equal(t, "0", security["Passwortänderungen (Anzahl)"])
	assert.Equal(t, "", security["Letzte Passwortänderung"])
	assert.Equal(t, "0", security["Wiederherstellungscodes (gesamt)"])
	assert.Equal(t, "0", security["Wiederherstellungscodes (genutzt)"])
}

func TestSearchByQuery_UserDriverLicenses_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Driver License Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Driver License Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Falk", "Sonnenschein", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	otherUserID := seedDSARUser(t, pool, tenantOwn, "Rieke", "Dornbusch", true)
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	nextDue := today.AddDate(1, 0, 0)
	expiry := today.AddDate(5, 0, 0)

	licenseID := testutil.SeedRow(t, pool, "driver_licenses", map[string]any{
		"tenant_id":           tenantOwn,
		"driver_id":           subjectID,
		"license_classes":     []string{"B", "BE"},
		"expiry_date":         expiry,
		"checked_at":          today,
		"next_check_due_date": nextDue,
		"notes":               "Kopie im Personalordner",
	})
	defer testutil.CleanupRow(t, pool, "driver_licenses", licenseID)

	otherLicenseID := testutil.SeedRow(t, pool, "driver_licenses", map[string]any{
		"tenant_id":           tenantOwn,
		"driver_id":           otherUserID,
		"license_classes":     []string{"C"},
		"checked_at":          today,
		"next_check_due_date": nextDue,
	})
	defer testutil.CleanupRow(t, pool, "driver_licenses", otherLicenseID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Falk")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	licenses := dsarModule(t, persons[0], "Führerscheinkontrolle")
	assert.Equal(t, []string{"Führerscheinklassen", "Ablaufdatum", "Geprüft am", "Nächste Prüfung fällig", "Notizen"}, licenses.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Führerscheinklassen": "B, BE", "Ablaufdatum": expiry.Format(dsarDateLayout),
			"Geprüft am": today.Format(dsarDateLayout), "Nächste Prüfung fällig": nextDue.Format(dsarDateLayout),
			"Notizen": "Kopie im Personalordner",
		},
	}, recordMaps(licenses), "only the subject's own driver license check, not the other driver's")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Falk")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their driver license history")
}

func TestSearchByQuery_UserVehicleBookings_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Vehicle Booking Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Vehicle Booking Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	subjectID := seedDSARUser(t, pool, tenantOwn, "Traugott", "Wiesengrund", true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Hilde", "Ackermann", true)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	plate := "M-DS " + uuid.New().String()[:6]
	vehicleID := testutil.SeedRow(t, pool, "vehicles", map[string]any{
		"tenant_id":     tenantOwn,
		"license_plate": plate,
		"make":          "VW",
		"model":         "Caddy",
		"year":          2023,
	})
	defer testutil.CleanupRow(t, pool, "vehicles", vehicleID)
	vehicleLabel := fmt.Sprintf("VW Caddy (%s)", plate)

	now := time.Now().UTC().Truncate(time.Second)

	// Subject is both driver and booker.
	ownBooking := testutil.SeedRow(t, pool, "vehicle_bookings", map[string]any{
		"tenant_id":  tenantOwn,
		"vehicle_id": vehicleID,
		"user_id":    subjectID,
		"created_by": subjectID,
		"starts_at":  now.Add(24 * time.Hour),
		"ends_at":    now.Add(26 * time.Hour),
		"purpose":    "Kundentermin",
		"status":     "booked",
	})
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", ownBooking)

	// Subject booked the vehicle for a colleague -- "Buchender", not "Fahrer".
	bookedForColleague := testutil.SeedRow(t, pool, "vehicle_bookings", map[string]any{
		"tenant_id":  tenantOwn,
		"vehicle_id": vehicleID,
		"user_id":    colleagueID,
		"created_by": subjectID,
		"starts_at":  now.Add(48 * time.Hour),
		"ends_at":    now.Add(50 * time.Hour),
		"purpose":    "Materialtransport",
		"status":     "booked",
	})
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", bookedForColleague)

	// Neither driver nor booker -- must not appear.
	unrelatedBooking := testutil.SeedRow(t, pool, "vehicle_bookings", map[string]any{
		"tenant_id":  tenantOwn,
		"vehicle_id": vehicleID,
		"user_id":    colleagueID,
		"created_by": colleagueID,
		"starts_at":  now.Add(72 * time.Hour),
		"ends_at":    now.Add(74 * time.Hour),
		"purpose":    "Werkstatt",
		"status":     "booked",
	})
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", unrelatedBooking)

	// starts_at/ends_at round-trip through the DB session's own time zone
	// setting, so the expected strings are read back from Postgres rather
	// than formatted from the Go-side `now` value used to seed them (see
	// querySessionLastActive above).
	ownStarts, ownEnds := queryVehicleBookingWindow(t, pool, ownBooking)
	colleagueStarts, colleagueEnds := queryVehicleBookingWindow(t, pool, bookedForColleague)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Traugott")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	bookings := dsarModule(t, persons[0], "Fahrzeugbuchungen")
	assert.Equal(t, []string{"Fahrzeug", "Beginn", "Ende", "Zweck", "Status", "Rolle"}, bookings.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Fahrzeug": vehicleLabel, "Beginn": colleagueStarts.Format(dsarTimeLayout),
			"Ende": colleagueEnds.Format(dsarTimeLayout), "Zweck": "Materialtransport",
			"Status": "booked", "Rolle": "Buchender",
		},
		{
			"Fahrzeug": vehicleLabel, "Beginn": ownStarts.Format(dsarTimeLayout),
			"Ende": ownEnds.Format(dsarTimeLayout), "Zweck": "Kundentermin",
			"Status": "booked", "Rolle": "Fahrer und Buchender",
		},
	}, recordMaps(bookings), "own bookings ordered by start time, each carrying the right role -- the unrelated booking must not appear")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Traugott")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their vehicle bookings")
}

func TestSearchByQuery_UserInvitationHistory_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Invitation Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Invitation Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	inviterID := seedDSARUser(t, pool, tenantOwn, "Petra", "Leitwolf", true)
	defer testutil.CleanupRow(t, pool, "users", inviterID)

	subjectEmail := fmt.Sprintf("dsar-invite-%s@search.invalid", uuid.New())
	subjectID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         subjectEmail,
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Timo",
		"last_name":     "Wachtberg",
		"is_active":     true,
	})
	defer testutil.CleanupRow(t, pool, "users", subjectID)

	// A second, still-pending invitation to the SAME address (e.g. a stray
	// re-invite of an already-onboarded person) must not surface -- it has
	// no accepted_at, so it never became this account.
	pendingID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"tenant_id":  tenantOwn,
		"email":      subjectEmail,
		"first_name": "Nie",
		"last_name":  "Angenommen",
		"role":       "member",
		"token_hash": fmt.Sprintf("pending-%s", uuid.New()),
		"created_by": inviterID,
		"expires_at": time.Now().Add(72 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", pendingID)

	acceptedAt := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	invitationID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"tenant_id":   tenantOwn,
		"email":       subjectEmail,
		"first_name":  "Timo",
		"last_name":   "Wachtberg",
		"role":        "admin",
		"token_hash":  fmt.Sprintf("accepted-%s", uuid.New()),
		"created_by":  inviterID,
		"expires_at":  time.Now().Add(72 * time.Hour),
		"accepted_at": acceptedAt,
	})
	defer testutil.CleanupRow(t, pool, "invitations", invitationID)

	invCreatedAt, invAcceptedAt := queryInvitationTimestamps(t, pool, invitationID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, "Timo")
	require.NoError(t, err)
	require.Len(t, persons, 1)

	history := dsarModule(t, persons[0], "Einladungshistorie")
	assert.Equal(t, []string{"Name bei Einladung", "Rolle", "Eingeladen von", "Eingeladen am", "Angenommen am"}, history.Columns)
	assert.Equal(t, []map[string]string{
		{
			"Name bei Einladung": "Timo Wachtberg", "Rolle": "admin",
			"Eingeladen von": "Petra Leitwolf",
			"Eingeladen am":  invCreatedAt.Format(dsarTimeLayout),
			"Angenommen am":  invAcceptedAt.Format(dsarTimeLayout),
		},
	}, recordMaps(history), "only the accepted invitation must appear, the pending one to another address must not")

	// --- tenant isolation ---------------------------------------------------
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, "Timo")
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant must not see this user, let alone their invitation history")
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

func querySessionLastActive(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID) time.Time {
	t.Helper()
	var lastActive time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT last_active_at FROM user_sessions WHERE id = $1`, sessionID).Scan(&lastActive))
	return lastActive
}

func queryVehicleBookingWindow(t *testing.T, pool *pgxpool.Pool, bookingID uuid.UUID) (starts, ends time.Time) {
	t.Helper()
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT starts_at, ends_at FROM vehicle_bookings WHERE id = $1`, bookingID).Scan(&starts, &ends))
	return starts, ends
}

func queryInvitationTimestamps(t *testing.T, pool *pgxpool.Pool, invitationID uuid.UUID) (createdAt, acceptedAt time.Time) {
	t.Helper()
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_at, accepted_at FROM invitations WHERE id = $1`, invitationID).Scan(&createdAt, &acceptedAt))
	return createdAt, acceptedAt
}

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
