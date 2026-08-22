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
		"tenant_id":            tenantOwn,
		"contact_id":           contactID,
		"created_by":           agentID,
		"status":               "finalized",
		"date":                 "2025-01-15",
		"advisor":              "Herr Bergmann",
		"marital_status":       "verheiratet",
		"known_asset_classes":  []string{"stocks", "etf"},
		"investment_purpose":   []string{"retirement", "growth"},
		"risk_class":           4,
		"products":             products,
		"warnings_given":       []string{"risk", "costs"},
		"internal_notes":       "Kunde wirkt unsicher, im Folgetermin nochmal Risiko erklären.",
		"monthly_net_income":   4200.50,
		"created_at":           "2025-01-15T09:00:00Z",
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
		"tenant_id":   tenantOwn,
		"user_id":     subjectID,
		"start_time":  "20:00",
		"end_time":    "07:00",
		"enabled":     true,
		"manual_dnd":  false,
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
