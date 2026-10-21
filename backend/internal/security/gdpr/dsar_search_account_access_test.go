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

func TestSearchByQuery_UserAccountAccessAndAuditExtract_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "DSAR Access Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "DSAR Access Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	// Unique surnames: audit_log is append-only and FK-blocks user deletion,
	// so these users outlive the test (same tradeoff as the audit_log tests).
	subjectName := "Zugang" + uuid.NewString()[:8]
	colleagueName := "Kollege" + uuid.NewString()[:8]
	subjectID := seedDSARUser(t, pool, tenantOwn, "Alma", subjectName, true)
	colleagueID := seedDSARUser(t, pool, tenantOwn, "Kurt", colleagueName, true)
	foreignUserID := seedDSARUser(t, pool, tenantOther, "Alma", subjectName, true)
	defer testutil.CleanupRow(t, pool, "users", subjectID)
	defer testutil.CleanupRow(t, pool, "users", colleagueID)
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	sys := testutil.WithSystemCtx(context.Background())
	exec := func(sql string, args ...any) {
		t.Helper()
		_, err := pool.Exec(sys, sql, args...)
		require.NoError(t, err, sql)
	}

	var permName string
	require.NoError(t, pool.QueryRow(sys, `SELECT name FROM permissions ORDER BY name LIMIT 1`).Scan(&permName))

	// --- roles ----------------------------------------------------------------
	roleOwn := testutil.SeedRow(t, pool, "roles", map[string]any{"tenant_id": tenantOwn, "name": "Vertrieb " + subjectName})
	roleColleague := testutil.SeedRow(t, pool, "roles", map[string]any{"tenant_id": tenantOwn, "name": "Buchhaltung " + colleagueName})
	defer testutil.CleanupRow(t, pool, "roles", roleOwn)
	defer testutil.CleanupRow(t, pool, "roles", roleColleague)
	exec(`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3), ($4, $5, $3)`,
		subjectID, roleOwn, tenantOwn, colleagueID, roleColleague)

	// --- overrides, grants, settings, layouts, preferences -------------------
	exec(`INSERT INTO user_permission_overrides (tenant_id, user_id, permission_key, mode) VALUES ($1, $2, $3, 'allow'), ($1, $4, $3, 'deny')`,
		tenantOwn, subjectID, permName, colleagueID)
	exec(`INSERT INTO user_module_grants (tenant_id, user_id, module_id) VALUES ($1, $2, 'kalender'), ($1, $3, 'colleague-module')`,
		tenantOwn, subjectID, colleagueID)
	exec(`INSERT INTO user_settings (tenant_id, user_id, module_id, key, value) VALUES ($1, $2, 'kalender', 'weekStart', '"monday"'::jsonb), ($1, $3, 'kalender', 'colleague-key', '"x"'::jsonb)`,
		tenantOwn, subjectID, colleagueID)
	exec(`INSERT INTO user_dashboard_layouts (tenant_id, user_id, active_widgets) VALUES ($1, $2, '["revenue-widget"]'::jsonb), ($1, $3, '["colleague-widget"]'::jsonb)`,
		tenantOwn, subjectID, colleagueID)
	project := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id": tenantOwn, "name": "Access Projekt", "project_key": "ACC", "created_by": subjectID,
	})
	defer testutil.CleanupRow(t, pool, "projects", project)
	exec(`INSERT INTO user_project_preferences (tenant_id, user_id, project_id, view_type) VALUES ($1, $2, $3, 'kanban'), ($1, $4, $3, 'gantt')`,
		tenantOwn, subjectID, project, colleagueID)

	// --- linked accounts, filters, app passwords -----------------------------
	exec(`INSERT INTO integration_account_links (platform, external_user_id, kmuhub_user_id, external_display_name, tenant_id) VALUES ('slack', 'ext-' || $4::text, $1, 'Alma auf Slack', $3), ('slack', 'ext-c-' || $4::text, $2, 'Kurt auf Slack', $3)`,
		subjectID, colleagueID, tenantOwn, subjectName)
	exec(`INSERT INTO saved_filters (name, entity_type, filter_json, created_by, tenant_id) VALUES ('Meine offenen Deals', 'deal', '{"stage":"open"}'::jsonb, $1, $3), ('Kollegen-Filter', 'deal', '{}'::jsonb, $2, $3)`,
		subjectID, colleagueID, tenantOwn)
	exec(`INSERT INTO app_specific_passwords (user_id, label, password_hash, password_prefix, tenant_id) VALUES ($1, 'iPhone Kalender', 'SECRET-HASH-VALUE', 'ab12', $3), ($2, 'Kollegen-Laptop', 'OTHER-HASH', 'cd34', $3)`,
		subjectID, colleagueID, tenantOwn)

	// --- audit log: 55 entries for the subject, 1 for the colleague ----------
	for i := 0; i < dsarAuditExtractRows+5; i++ {
		exec(`INSERT INTO audit_log (user_id, action, target, target_type, details, ip_address, user_agent, entry_hash, tenant_id, "timestamp")
		      VALUES ($1, $2, $3, 'contact', '{"secret":"details-must-not-leak"}'::jsonb, '203.0.113.7', 'TestBrowser/1.0', 'hash', $4, now() - ($5 || ' minutes')::interval)`,
			subjectID, fmt.Sprintf("contact.view.%02d", i), "target-"+subjectName, tenantOwn, fmt.Sprint(i))
	}
	exec(`INSERT INTO audit_log (user_id, action, entry_hash, tenant_id) VALUES ($1, 'colleague.action', 'hash', $2)`, colleagueID, tenantOwn)
	// A stray row of the subject under another tenant must not show up.
	exec(`INSERT INTO audit_log (user_id, action, entry_hash, tenant_id) VALUES ($1, 'stray.foreign.tenant', 'hash', $2)`, subjectID, tenantOther)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	persons, err := SearchByQuery(ctxOwn, pool, tenantOwn, subjectName)
	require.NoError(t, err)
	require.Len(t, persons, 1)
	person := persons[0]

	only := func(module, col, want string) {
		t.Helper()
		recs := recordMaps(dsarModule(t, person, module))
		require.Len(t, recs, 1, module)
		assert.Equal(t, want, recs[0][col], module)
	}
	only("Rollen", "Rolle", "Vertrieb "+subjectName)
	only("Berechtigungs-Overrides", "Modus", "allow")
	only("Modul-Freigaben", "Modul", "kalender")
	only("Verknüpfte Konten", "Anzeigename", "Alma auf Slack")
	only("Einstellungen", "Schlüssel", "weekStart")
	only("Dashboard-Layout", "Aktive Widgets", `["revenue-widget"]`)
	only("Projekt-Ansichten", "Ansicht", "kanban")
	only("Gespeicherte Filter", "Name", "Meine offenen Deals")
	only("App-Passwörter", "Bezeichnung", "iPhone Kalender")

	// Password material and audit details never leave the database.
	for _, mod := range person.Modules {
		for _, rec := range mod.Records {
			for _, f := range rec.Fields {
				assert.NotContains(t, f.Value, "SECRET-HASH", mod.Module)
				assert.NotContains(t, f.Value, "ab12", mod.Module)
				assert.NotContains(t, f.Value, "details-must-not-leak", mod.Module)
			}
		}
	}

	// Audit extract: capped, newest first, total count in the title, IP/UA shown.
	wantTitle := fmt.Sprintf("Zugriffsprotokoll (neueste %d von %d Einträgen)", dsarAuditExtractRows, dsarAuditExtractRows+5)
	audit := recordMaps(dsarModule(t, person, wantTitle))
	require.Len(t, audit, dsarAuditExtractRows)
	assert.Equal(t, "contact.view.00", audit[0]["Aktion"], "newest entry first")
	assert.Equal(t, "203.0.113.7", audit[0]["IP-Adresse"])
	assert.Equal(t, "TestBrowser/1.0", audit[0]["Browser"])
	for _, a := range audit {
		assert.NotEqual(t, "colleague.action", a["Aktion"])
		assert.NotEqual(t, "stray.foreign.tenant", a["Aktion"])
	}

	// The colleague gets only their own rows; a short log carries no "neueste" wording.
	colleaguePersons, err := SearchByQuery(ctxOwn, pool, tenantOwn, colleagueName)
	require.NoError(t, err)
	require.Len(t, colleaguePersons, 1)
	cp := colleaguePersons[0]
	assert.Equal(t, "Buchhaltung "+colleagueName, recordMaps(dsarModule(t, cp, "Rollen"))[0]["Rolle"])
	assert.Equal(t, "deny", recordMaps(dsarModule(t, cp, "Berechtigungs-Overrides"))[0]["Modus"])
	assert.Equal(t, "Kollegen-Filter", recordMaps(dsarModule(t, cp, "Gespeicherte Filter"))[0]["Name"])
	assert.Equal(t, "colleague.action", recordMaps(dsarModule(t, cp, "Zugriffsprotokoll (1 Einträge)"))[0]["Aktion"])

	// --- tenant isolation: same-named user in another tenant sees nothing -----
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := SearchByQuery(ctxOther, pool, tenantOther, subjectName)
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	for _, m := range foreign[0].Modules {
		switch m.Module {
		case "Rollen", "Berechtigungs-Overrides", "Modul-Freigaben", "Verknüpfte Konten",
			"Einstellungen", "Dashboard-Layout", "Projekt-Ansichten", "Gespeicherte Filter", "App-Passwörter":
			t.Errorf("foreign tenant must not see own-tenant module %q", m.Module)
		}
		assert.NotContains(t, m.Module, "Zugriffsprotokoll")
	}
}
