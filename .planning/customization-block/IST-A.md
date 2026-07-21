# IST-Analyse: Config-Infrastruktur & Modul-Ebene
> Scope: Config-Infrastruktur-Achsen für das geplante Self-Service-Customization-Tool.
> Erstellt: 2026-07-21 | Recherche-Basis: echter Code, kein Raten.

---

## 1. Modul-Einstellungen-Shell (Achse A — zentraler Anker)

### Was existiert

Die `ModuleSettingsShell` ist **vollständig gebaut** und bereits in **25 Module** eingebunden.

**Kernkomponenten:**

| Datei | Funktion |
|-------|---------|
| `desktop/src/renderer/src/components/shared/ModuleSettingsShell.tsx` | Wiederverwendbarer Scope-aware Container |
| `desktop/src/renderer/src/lib/module-settings.ts` | `SettingsScope`, `SettingsModuleId`, `LEADABLE_MODULES` |
| `desktop/src/renderer/src/components/shared/module-settings-scope.ts` | Context-Provider für `{ scope, editable }` |
| `desktop/src/renderer/src/hooks/useModuleSettings.ts` | `useIsModuleLead(moduleId)` — RBAC-Gate |
| `desktop/src/renderer/src/hooks/useHydrateModuleSettings.ts` | Session-einmalige Server-Hydration |
| `desktop/src/renderer/src/modules/settings/module-settings-registry.tsx` | Zentrales `SETTINGS_ENTRIES`-Array mit 25 Modulen + 7 Cosmi-Tabs |
| `desktop/src/renderer/src/api/settings-persist.ts` | `loadModuleSettings` / `saveModuleSettings` (GET/PUT `/api/v1/settings/{moduleId}/{scope}`) |

**Scope-Modell (zwei Ebenen):**
- `personal` — jeder User, immer editierbar, schreibt auf `/settings/{moduleId}/user`
- `tenant` — nur Modul-Leiter oder `settings:tenant:manage`-Träger, schreibt auf `/settings/{moduleId}/tenant`

**RBAC-Gate:**
```
useIsModuleLead(moduleId):
  → settings:tenant:manage → immer true (admin/it_admin)
  → ODER: isLead(user.id, moduleId) aus useModuleLeadsStore (BE: tenant_module_leads-Tabelle)
```

**Konkrete Beispiele je Modul — was heute per UI konfigurierbar ist:**

| Modul | Personal | Tenant |
|-------|---------|--------|
| **CRM** | Default-Ansicht, Sortierung | Pipeline-Phasen (Name/Farbe/Reihenfolge/Wahrscheinlichkeit), Custom Fields (6 Typen), Tags, Segmente, Lead-Scoring |
| **Work** | Density, Default-Ansicht | Label-Taxonomie, Projekt-Templates, Default-Status-Set (Name/Farbe/Reihenfolge), Custom Fields, Zeitregeln |
| **Finance** | Default-Tab, Währung | Stammdaten (Firma/Adresse/Steuer/Bank), Rechnungseinstellungen, Kontierungskonten, Integrationen |
| **Dashboard** | Greeting on/off, Grid-Dichte | Default-Widget-Set für neue User, Allowed-Widgets-Liste (Gating) |
| **Dialer** | Nachbearbeitungszeit, Auto-Advance | Max gleichzeitige Calls, Recording-Consent-Default, Default-Outcome |
| **Helpdesk** | Start-Tab, Status-Default-Filter | Business Hours, Ticket-Routing-Config |
| **Dokumente** | Density, Default-Ansicht | Template-Sichtbarkeit, Share-Einstellungen |
| **Branding** | — | Workspace-Name, Logo, Square-Icon, Accent-Color (mock-first, localStorage) |
| **Company** | — | Firmendaten, Steuer & Recht, Bankdaten, Zahlungskonditionen, Logo-URL |

**Persistenz-Stack:**
```
Zustand-Store (Setter) → saveModuleSettings → PUT /api/v1/settings/{id}/{scope}
                      ↑
  initFromServer() → loadModuleSettings → GET /api/v1/settings/{id}/{scope}
  (via useHydrateModuleSettings, 1×/Session)
  
localStorage als Optimistic-Cache (Zustand `persist`)
```

**Backend-Realität:** Settings-Service live und scharf. Go-Seite: `backend/internal/settings/postgres_repository.go` mit `tenant_settings` + `user_settings` Tabellen (JSONB, UPSERT-Semantik, RLS-konform über `tenant_id`). Routen: `route_settings.go` mit 5 Endpunkten.

**Lücken (Achse A):**
- Business-Profile-ID (`businessProfileId`) liegt im Frontend-Only-Store (`stores/profile.ts`), wird NICHT auf dem Settings-Endpunkt persistiert → kein Server-Sync des Branchen-Profils
- Branding (Workspace-Name, Logo) ist noch mock-first in localStorage (Backend-Upload = Luke's Track, noch offen)
- Kein `updated_by`-Timestamp im FE sichtbar; BE speichert `updated_by` in `tenant_settings`, aber kein Audit-Event für Settings-Änderungen (nur für RBAC)
- `settings:tenant:manage`-RBAC-Key ist hart auf admin/it_admin gemappt; kein Mechanismus um Settings-Editing ans Self-Service zu delegieren

---

## 2. RBAC als Blaupause (Achse B)

### Was existiert

**Zwei vollständige Two-Pane-Editoren:**
- `RoleEditorPage.tsx` (`/admin/roles/:roleId`) — Rollen-Editor
- `UserOverrideEditorPage.tsx` (`/admin/users/:userId/overrides`) — Per-User-Override-Editor

**Wiederverwendbare Muster, die ein Config-Editor direkt nachnutzen kann:**

| Muster | Datei / Beschreibung |
|--------|---------------------|
| **Two-Pane-Layout** | Linke Spalte (w-64, scrollbare Liste) + rechte Pane (scrollbarer Detail-Bereich) — exakt derselbe DOM-Aufbau in beiden Editoren |
| **Staged-Commit-Footer** | Dirty-State-Tracking + Staged-Count + Discard/Apply-Buttons + Popover-Übersicht der Änderungen |
| **AlertDialog Guardrails** | Escalation-Block + Self-Lockout + Last-Admin — Pattern übertragbar auf Config-Guardrails (z.B. "letztes Pflichtfeld entfernen") |
| **`startPreview` / `endPreview`** | `stores/permissions.ts` — Overlay-Preview ohne Logout (View-as-Modus); übertragbar als "Kunden-Preview" im Customization-Tool |
| **Audit-Interceptor** | `mocks/data/audit-events.ts` + `writeAuditEvent()` — bereits für `setting.changed`-Event-Typ vorgesehen (Zeile 18 in audit-events.ts) |
| **Template-Galerie** | `mocks/data/industry-role-templates.ts` — `INDUSTRY_ROLE_SETS` (3 Branchenpakete × 4 Rollen = 12 Templates) inkl. `orderedSetsForProfile()` (Sortierung nach aktuellem Business-Profile) |
| **TRI-STATE-Row** | `OverrideRow` in UserOverrideEditorPage — inherited/allow/deny mit Cycle-Tap, Scope-Select, Reset-Button; Template für jede Config-Zeile mit Default+Override-Logik |
| **`PermissionPreviewBanner`** | `components/layout/PermissionPreviewBanner.tsx` — sticky Banner während Preview-Modus |

**Was noch NICHT existiert:**
- Kein generischer "Config-Item-Editor" abstrahiert aus den RBAC-Editoren — beide sind feature-spezifisch implementiert
- Keine Template-Galerie für Module-Configs (nur für Rollen)
- `setting.changed`-Audit-Event existiert nur als reserved Token (Zeile 18 audit-events.ts), wird noch nicht gefeuert

---

## 3. Feature-Flags & Modul-Aktivierung (Achse C)

### Deploy/Env-Zeit vs. In-Cosmi — genaue Grenze

**Env-Zeit (NICHT selbst-bedienbar, Stand heute):**

18 Flags in `backend/internal/featureflag/registry.go`:
- 14 optionale Industrie-Module: `COSMI_MODULE_{NAME}_ENABLED` (wiki, berichte, formulare, helpdesk, vertraege, buchhaltung, video, rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion)
- 1 Integration: `COSMI_INTEGRATION_BEXIO_ENABLED`
- 3 Plugin-Flags: `COSMI_WASM_PLUGINS_ENABLED` (breaking/security), `COSMI_CONFIG_PLUGINS_ENABLED` (on by default), `COSMI_PLUGIN_API_ENABLED` (security)

Alle Modul-Flags sind `Risk: SafeRisk, LLMToggleSafe: true` — technisch wäre ein In-Cosmi-Toggle ohne Restart möglich (kein Service-Restart, keine Migration nötig). Die Grenze ist aktuell **konventionell, nicht technisch** erzwungen.

**In-Cosmi (bereits selbst-bedienbar):**
- `ModuleAssignmentSettingsPanel` → `user_module_grants`-Tabelle: welcher User welches Modul verwenden darf (Sitzung-zu-Modul-Zuteilung, nicht Modul-An/Aus)
- Business-Profile-Modul-Sichtbarkeit (FE-only, kein BE-Sync)
- Modul-Pinning in der Sidebar (FE-only, localStorage)

**Was müsste verschoben werden für Self-Service:**
- Modul-An/Aus (`modules.*`-Flags) müsste von Env-Var auf DB-gestützte Tenant-Config wechseln — ein neues `tenant_feature_flags`-Modell oder Erweiterung von `tenant_settings` mit moduleId='modules'
- Backend müsste Flags per Tenant und nicht global lesen (derzeit: ein globales Registry-Objekt pro Prozess)

---

## 4. Business-Profiles / Branchen-Presets (Achse D)

### Was existiert

`desktop/src/renderer/src/config/business-profiles.ts` — vollständig definiert.

**10 Profile:** allgemein, handwerk, gastronomie, einzelhandel, dienstleistung, it_tech, produktion, logistik, gesundheit, bau

**Was sie steuern:**
- `defaultModules: string[]` — Module immer sichtbar (nicht RBAC-gated, profile-gated)
- `optionalModules: string[]` — Module sichtbar nur wenn in `enabledOptionalModules` enthalten
- `ALWAYS_VISIBLE_MODULES` — dashboard, settings, security-admin, profil, notifications, contacts, admin (immer sichtbar, nie profile-gated)

**Mechanik:**
```
useFilteredNavItems() filtert navItems mit 3 Gates (in Reihenfolge):
  1. RBAC: moduleViewKey muss granted sein
  2. Feature-Flags: MODULE_FLAG_IDS muss BE-Flag = true haben
  3. Business-Profile: isModuleAllowedForProfile(id, businessProfileId, enabledOptionals)
```

**Kritische Lücke:** `businessProfileId` und `enabledOptionalModules` liegen in `stores/profile.ts` mit `persist({ name: 'cosmi-profiles' })` — **localStorage-only**, kein Server-Sync. Das Branchen-Profil ist heute nur auf einem Gerät gesetzt und geht beim Browser-Cache-Löschen verloren.

**INDUSTRY_ROLE_SETS** in `mocks/data/industry-role-templates.ts` verlinkten bereits auf `businessProfileIds` via `orderedSetsForProfile()` — das Template-Galerie-Pattern existiert also bereits als Muster.

---

## 5. Dashboards / Widgets (Achse E / Dimension I)

### Was existiert

**22 Widget-Typen** in `stores/dashboard.ts` (`ALL_WIDGET_IDS`).

**Personal-Scope (jeder User):**
- Widget-Layout (react-grid-layout, `stores/dashboard.ts`) — Drag & Drop, persönliche Anordnung
- `DashboardScope`: 'personal' | 'team' — separate Layout-Slots
- Dichte: comfortable / compact (`stores/dashboardPrefs.ts`)
- Greeting on/off

**Tenant-Scope (Admin-Only, nicht delegierbar):**
- `allowedWidgets: WidgetId[]` — Whitelist, User kann nur diese Widgets hinzufügen
- `defaultWidgets: WidgetId[]` — Starter-Layout für neue User (eingeschränkt auf allowedWidgets)

**Persistenz:** `stores/dashboardSettings.ts` — server-synced via `GET/PUT /settings/dashboard/tenant`. Dashboard-Layout selbst via separaten Dashboard-Sync-Endpunkt (BE: Luke's Track).

**Lücke:** Kein Tenant-Default-Layout als DnD-Vorschau editierbar; allowedWidgets ist nur Chip-Toggle-Liste, kein Positioning. Kein Branding/Umbenennung von Widgets möglich.

---

## 6. Modul & Navigation (Achse F / Dimension H)

### Was existiert

**Modul-Pinning:** `stores/ui.ts` → `pinnedModules: string[]` — User kann Module in die Sidebar pinnen/entpinnen (FE-only, localStorage).

**Modul-Reihenfolge:** Implizit durch `pinnedModules`-Array-Reihenfolge. `setPinnedModules(modules)` überschreibt die ganze Liste. Kein Drag-Drop-Reorder-UI vorhanden.

**Modul-Sichtbarkeit:** Heute durch drei unabhängige Schichten kontrolliert:
1. RBAC (`moduleViewKey`) — wer darf Modul sehen
2. Feature-Flags (BE-Env) — ob Modul deployed ist
3. Business-Profile (FE-localStorage) — welche Module zum Branchen-Preset gehören

**Modul-Umbenennung/eigene Icons:** Nicht vorhanden. Icons und Labels in `nav-items.ts` fest codiert (Lucide-Icons + i18n-Schlüssel).

**Lücken:** Kein Tenant-Config für Modul-Namen. Kein Admin-UI für Reihenfolge. Kein "Modul deaktivieren"-Toggle in Cosmi (nur via Env-Var oder user_module_grants).

---

## 7. Status-Sets / Wertelisten / Pipelines (Achse G / Dimension M) + Benachrichtigungs-Regeln (Dimension N)

### Was existiert — Status-Sets & Wertelisten

**CRM — konfigurierbar:**
- Pipeline-Phasen: Name, Farbe, Reihenfolge, Wahrscheinlichkeit, is_won/is_lost → `PipelineStagesEditor.tsx`, API-backed (`usePipelineStages`)
- Custom Fields: Text/Number/Date/Dropdown/Checkbox/URL → `CustomFieldsManager.tsx`, `stores/contacts.ts`
- Tags: `TagManager.tsx`
- Segmente: `SegmentSettings.tsx`
- Lead-Scoring-Regeln: `LeadScoringSettings.tsx`

**Work — konfigurierbar:**
- Default-Status-Set für neue Projekte: Name, Farbe, Reihenfolge → `DefaultStatusSetEditor.tsx`, `stores/workSettings.ts` (mock-first)
- Label-Taxonomie: `LabelTaxonomyManager.tsx`
- Projekt-Templates: `ProjectTemplatesManager.tsx`
- Custom Fields: `WorkCustomFieldsManager.tsx`
- Zeitregeln: `TimeRulesSettings.tsx`

**Helpdesk:** Ticket-Status-Labels hardcodiert in `HelpdeskSettingsPanel.tsx` (enum: open/in_progress/waiting/resolved/closed) — **NICHT konfigurierbar**. Business Hours und Routing-Config editierbar.

**Finance:** Kontierungskonten (`KontierungSettings.tsx`), Zahlungskonditionen, Rechnungs-Defaults — editierbar.

**Benachrichtigungs-Regeln (Dimension N):**
- `NotificationsSettingsPanel` existiert in der Registry
- Inhalt aus `stores/notifications`-Prefs — personal scope only (Kanal-Präferenzen: in-app/email/push)
- Kein tenant-weites Notification-Policy-Setting gefunden (z.B. "Alle Mitarbeiter kriegen bestimmten Alert-Typ")

### Was hardcodiert ist (nicht konfigurierbar)

- Helpdesk-Status-Enum (open/in_progress/waiting/resolved/closed) — Code
- Inventar-Kategorien — nicht gesucht, vermutlich hardcodiert
- Projekt-Status per Projekt — editierbar per Projekt, aber kein Tenant-Default-Override
- Nav-Item-Icons und -Label-Keys — `nav-items.ts` fest

---

## Übersichts-Tabelle

| Dimension | Existiert? | Wo (Pfad) | In-Cosmi-UI oder Code/Deploy/Env | Wer darf (RBAC-Key) | Lücke |
|-----------|-----------|-----------|----------------------------------|---------------------|-------|
| ModuleSettingsShell | **Ja, vollständig** | `components/shared/ModuleSettingsShell.tsx`, `lib/module-settings.ts` | In-Cosmi (Settings-Overlay) | `settings:tenant:manage` ODER Modul-Leiter (`tenant_module_leads`) | Kein Audit-Log für Setting-Änderungen; kein Self-Service-Delegation-Key |
| RBAC-Two-Pane-Editor | **Ja** | `modules/admin/roles/RoleEditorPage.tsx`, `modules/admin/users/UserOverrideEditorPage.tsx` | In-Cosmi (Admin-Hub) | `admin:role:edit`, `admin:user_override:manage` | Nicht generisch abstrahiert für Config-Editoren |
| Industry-Role-Templates | **Ja** | `mocks/data/industry-role-templates.ts` | In-Cosmi (CloneRoleDialog) | `admin:role:edit` | Nur für Rollen; kein analoges Template-System für Module-Config |
| startPreview / View-as | **Ja** | `stores/permissions.ts`, `PermissionPreviewBanner.tsx` | In-Cosmi | `admin:role:view` (zum Triggern) | Nur permissions-basiert; kein "Config-Preview-Modus" |
| Audit-Events | **Teilweise** | `mocks/data/audit-events.ts` | In-Cosmi (Security-Modul) | `admin:audit:read` | `setting.changed` nur als reserved Token, wird nicht gefeuert |
| Feature-Flags (Modul-An/Aus) | **Ja, Env-only** | `backend/internal/featureflag/registry.go` | Deploy/Env | — (kein User) | Kein In-Cosmi-Toggle; globaler Registry-Scope, nicht per Tenant |
| Modul-Zuteilung per User | **Ja** | `modules/team/ModuleAssignmentTab` → `user_module_grants` | In-Cosmi (Team-Settings) | `module-grants:write` | Nur User-zu-Modul, kein Tenant-weites Modul-An/Aus |
| Business-Profiles | **Ja, FE-only** | `config/business-profiles.ts`, `stores/profile.ts` | In-Cosmi (Onboarding-Wizard, DevTools) | niemand (localStorage) | Kein BE-Sync; kein Admin-UI; kein "Profil nachträglich ändern"-Flow |
| Dashboard-Widgets | **Ja** | `modules/dashboard/settings/DashboardSettingsPanel.tsx`, `stores/dashboardSettings.ts` | In-Cosmi | `settings:tenant:manage` (nicht delegierbar) | Kein Default-Layout-Editor; kein Widget-Umbenennen |
| Modul-Pinning/Reihenfolge | **Teilweise** | `stores/ui.ts` `pinnedModules` | In-Cosmi (Sidebar) | jeder User | Nur personal; kein Tenant-Default; kein Drag-Drop-Reorder |
| Modul-Icons/-Namen | **Nein** | `components/layout/sidebar/nav-items.ts` (hardcodiert) | Code | — | Kein API für Tenant-Custom-Labels |
| CRM Pipeline-Phasen | **Ja** | `modules/kontakte/PipelineStagesEditor.tsx` | In-Cosmi (CRM-Settings) | `settings:tenant:manage` ODER crm-Modul-Leiter | — |
| CRM Custom Fields | **Ja** | `modules/kontakte/CustomFieldsManager.tsx` | In-Cosmi | wie oben | Kein FE-Typing → BE-Schema-Sync (mock-first stores/contacts.ts) |
| Work Status-Sets | **Ja (mock-first)** | `modules/work/settings/DefaultStatusSetEditor.tsx` | In-Cosmi | `settings:tenant:manage` ODER work-Modul-Leiter | Store nicht BE-synced (workSettings.ts mock-only) |
| Helpdesk-Status | **Nein** | `modules/helpdesk/settings/HelpdeskSettingsPanel.tsx` | Code-Enum | — | Hardcodiert (open/in_progress/waiting/resolved/closed) |
| Benachrichtigungs-Regeln (Tenant) | **Nein** | — | — | — | Nur personal-scope Kanal-Prefs; kein Tenant-Policy-System |
| Tenant-Branding | **Ja (mock-first)** | `modules/admin/branding/BrandingAdminHubTab.tsx` | In-Cosmi (Branding-Tab) | `settings:tenant:manage` | Nur localStorage; Logo-Upload-Backend ausstehend (Luke) |
| Settings-Backend | **Ja, produktiv** | `backend/internal/settings/` + `route_settings.go` | — | — | JSONB-Keys sind schema-frei; kein Validierungs-Layer für Keys |

---

## Zusammenfassung / Handlungsempfehlung

### 3 größte vorhandene Fundamente

1. **ModuleSettingsShell + Settings-Persist-Backend** — ein vollständiger zweistufiger Config-Stack (personal/tenant), 25 Module eingebunden, BE-Live mit RLS, FE mit Hydration-Hook. Das ist der klarste Anker: Ein Self-Service-Customization-Tool ist eine Erweiterung der bestehenden Settings-Fläche, nicht eine neue.

2. **RBAC Two-Pane-Editor als UX-Blaupause** — insbesondere `UserOverrideEditorPage` mit Staged-Commit-Footer, TRI-STATE-Rows, Guardrail-AlertDialog und `startPreview`. Diese Muster können 1:1 in einen Config-Editor überführt werden (Default-Config + Customer-Override = dieselbe Logik wie Role-Union + Per-User-Override).

3. **Business-Profiles + Industry-Role-Templates als Template-System** — die Mechanik von "Branchen-Preset wählen → Module vorauswählen + Rollen vorausfüllen" ist vollständig implementiert. `orderedSetsForProfile()` sortiert Templates bereits nach aktivem Profil. Das ist der Backbone für "Config-Vorlagen".

### 3 größte Lücken

1. **Feature-Flag-An/Aus ist Env-Var-only (globaler Scope)** — kein In-Cosmi-Toggle, kein per-Tenant-Scope. Für Self-Service müsste Modul-Aktivierung auf `tenant_settings` oder eine neue `tenant_feature_flags`-Tabelle migriert werden. Das ist Luke's Backend-Track und der größte strukturelle Gap.

2. **Business-Profile ohne BE-Sync** — `businessProfileId` liegt in `stores/profile.ts` (localStorage). Wenn Zentria beim Onboarding das Branchen-Profil setzt, greift es nur auf dem lokalen Gerät. Für Self-Service (Kunde wählt Profil selbst, gilt tenant-weit) braucht es einen `PUT /settings/profile/tenant`-Endpunkt und Migration des `profileStore` auf Settings-Foundation.

3. **Kein Audit-Trail für Settings-Änderungen** — `writeAuditEvent` ist für `setting.changed` vorbereitet, wird aber noch nie aufgerufen. Für ein Self-Service-Tool mit mehreren Config-Admins ist ein Audit-Log essenziell (wer hat wann was geändert). Heute keine Nachvollziehbarkeit.

### Taugt die Modul-Einstellungen-Shell als zentraler Anker?

**Ja, ohne Einschränkung.** Die Shell löst bereits das schwierigste Problem: Scope-Separation (personal vs. tenant), RBAC-Gate (Modul-Leiter-Delegation), UI-Rendering (locked/unlocked), Backend-Persistenz (Settings-Foundation) und Hydration. Ein Self-Service-Customization-Tool muss die Shell nicht ersetzen — es muss sie ausweiten: neue Settings-Sektionen hinzufügen (z.B. Modul-Custom-Label, Wertelisten-Editor, Status-Set-Editor für alle Module) und den Zugang von "nur admin/Modul-Leiter" auf "Self-Service-Config-Admin" erweitern (ein neuer RBAC-Key z.B. `customization:manage`). Zentria-Onboarding nutzt exakt dieselbe Oberfläche mit erhöhten Rechten.
