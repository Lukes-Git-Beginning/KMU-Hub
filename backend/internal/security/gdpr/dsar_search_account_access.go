package gdpr

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Account access data and access-log extract of a matched user — Art. 15 GDPR.
//
// Read-only. Roles, permission overrides, module grants, linked external
// accounts, settings, layouts and saved filters are listed in full (capped by
// dsarMaxRows). audit_log is the largest table and is kept on purpose after
// erasure (Art. 17(3), § 147 AO, hash chain): only the newest
// dsarAuditExtractRows entries are shown, titled with the total count. Entry
// details (free JSON), hashes and password material are never selected.
// ---------------------------------------------------------------------------

// dsarAuditExtractRows caps the access-log extract per person.
const dsarAuditExtractRows = 50

// jsonPreviewChars truncates jsonb values so a large layout cannot bloat the export.
const jsonPreviewChars = 200

// accountAccessModules collects the account-access modules for one user;
// empty ones are skipped.
func accountAccessModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) ([]DSARModule, error) {
	specs := []struct {
		what, title string
		cols        []string
		sql         string
	}{
		{"user roles", "Rollen", []string{"Rolle", "Zugewiesen am"},
			`SELECT r.name, ur.assigned_at
			 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
			 WHERE ur.tenant_id = $1 AND ur.user_id = $2
			 ORDER BY ur.assigned_at DESC LIMIT $3`},
		{"permission overrides", "Berechtigungs-Overrides", []string{"Berechtigung", "Modus", "Geltungsbereich", "Angelegt am"},
			`SELECT permission_key, mode, scope, created_at
			 FROM user_permission_overrides
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY created_at DESC LIMIT $3`},
		{"module grants", "Modul-Freigaben", []string{"Modul", "Freigegeben am", "Zuletzt aktiv"},
			`SELECT module_id, granted_at, last_active_at
			 FROM user_module_grants
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY granted_at DESC LIMIT $3`},
		{"linked accounts", "Verknüpfte Konten", []string{"Plattform", "Externe Nutzer-ID", "Anzeigename", "Verknüpft am"},
			`SELECT platform, external_user_id, COALESCE(external_display_name, ''), linked_at
			 FROM integration_account_links
			 WHERE tenant_id = $1 AND kmuhub_user_id = $2
			 ORDER BY linked_at DESC LIMIT $3`},
		{"user settings", "Einstellungen", []string{"Modul", "Schlüssel", "Wert", "Geändert am"},
			fmt.Sprintf(`SELECT module_id, key, LEFT(value::text, %d), updated_at
			 FROM user_settings
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY module_id, key LIMIT $3`, jsonPreviewChars)},
		{"dashboard layouts", "Dashboard-Layout", []string{"Aktive Widgets", "Geändert am"},
			fmt.Sprintf(`SELECT LEFT(active_widgets::text, %d), updated_at
			 FROM user_dashboard_layouts
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY updated_at DESC LIMIT $3`, jsonPreviewChars)},
		{"project preferences", "Projekt-Ansichten", []string{"Projekt-ID", "Ansicht", "Gruppierung", "Sortierung", "Absteigend", "Geändert am"},
			`SELECT project_id, view_type, COALESCE(list_group_by, ''), COALESCE(list_sort_by, ''),
			        COALESCE(list_sort_desc, false), updated_at
			 FROM user_project_preferences
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY updated_at DESC LIMIT $3`},
		{"saved filters", "Gespeicherte Filter", []string{"Name", "Objektart", "Standard", "Filter", "Angelegt am"},
			fmt.Sprintf(`SELECT name, entity_type, is_default, LEFT(filter_json::text, %d), created_at
			 FROM saved_filters
			 WHERE tenant_id = $1 AND created_by = $2
			 ORDER BY created_at DESC LIMIT $3`, jsonPreviewChars)},
		// password_hash / password_prefix are deliberately not selected.
		{"app passwords", "App-Passwörter", []string{"Bezeichnung", "Verwendung", "Angelegt am", "Zuletzt genutzt", "Widerrufen am"},
			`SELECT label, scope, created_at, last_used_at, revoked_at
			 FROM app_specific_passwords
			 WHERE tenant_id = $1 AND user_id = $2
			 ORDER BY created_at DESC LIMIT $3`},
	}

	var out []DSARModule
	for _, sp := range specs {
		mod, err := userRowsModule(ctx, pool, sp.what, sp.title, sp.cols, sp.sql,
			[]any{tenantID, userID, dsarMaxRows}, valuesRecord(sp.cols))
		if err != nil {
			return nil, fmt.Errorf("dsar: account access: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
	}

	audit, err := auditExtractModule(ctx, pool, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if audit != nil {
		out = append(out, *audit)
	}
	return out, nil
}

// auditExtractModule shows the newest dsarAuditExtractRows access-log entries
// of the user with the total count in the title. Both queries filter on
// tenant_id and user_id (idx_audit_log_user_id / idx_audit_log_tenant_time).
func auditExtractModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	var total int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("dsar: count audit log: %w", err)
	}
	if total == 0 {
		return nil, nil
	}
	title := fmt.Sprintf("Zugriffsprotokoll (%d Einträge)", total)
	if total > dsarAuditExtractRows {
		title = fmt.Sprintf("Zugriffsprotokoll (neueste %d von %d Einträgen)", dsarAuditExtractRows, total)
	}
	cols := []string{"Zeitpunkt", "Aktion", "Objekt", "Objektart", "Ergebnis", "IP-Adresse", "Browser"}
	return userRowsModule(ctx, pool, "audit log", title, cols,
		`SELECT "timestamp", action, COALESCE(target, ''), COALESCE(target_type, ''), result,
		        COALESCE(host(ip_address), ''), COALESCE(user_agent, '')
		 FROM audit_log
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY "timestamp" DESC, sequence_num DESC
		 LIMIT $3`, []any{tenantID, userID, dsarAuditExtractRows}, valuesRecord(cols))
}
