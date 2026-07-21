# IST-Analyse B — Datenmodell & No-Code-Builder

**Scope:** Custom Fields / Custom Objects / Formulare / Automatisierung / Feld-Layouts / Listen-Konfiguration  
**Stand:** 2026-07-21  
**Erstellt von:** Sub-Agent (Darien-Session #24+)

---

## Übersichts-Tabelle

| Dimension | Existiert? | Wo (Pfad) | In-Cosmi-UI oder Code/Deploy | Wer darf (RBAC-Key) | Lücke |
|-----------|-----------|-----------|-------------------------------|---------------------|-------|
| A — Custom Fields (Work/Tasks) | ✅ VOLLSTÄNDIG (BE+FE) | `backend/internal/work/customfield/`, Migration `000146`, FE `WorkCustomFieldsManager`, `CustomFieldsSection` | In Cosmi — Manager/Admin konfiguriert über Work-Settings; alle drei Rollen sehen/setzen Werte | `work_custom_fields:read/write` (admin+manager+member) · `work_custom_fields:delete` (admin+manager only) | FE nutzt noch Zustand-Store statt Live-API; `CustomFieldsSection` fällt auf alte `custom_field_definitions`-Tabelle zurück statt `work_custom_field_definitions` |
| A — Custom Fields (CRM: Kontakte, Firmen, Deals, Aktivitäten) | ⚠️ TEILWEISE (BE vollständig, FE Demo-only) | Migration `000005`, `000007`, `000009`, `000010`; Gateway `/api/v1/custom-fields`; FE `kontakte/CustomFieldsManager` | In Cosmi — UI existiert im Kontakte-Modul (Manager, Settings-Bereich) + CRM-Settings-Panel | `custom_fields:read/write/delete`; admin=alle, manager=read+write, member=read | FE-CustomFieldsManager nutzt Zustand-Store (`useContactsStore`) statt Backend-API — Änderungen sind session-persistent, nicht tenant-persistent; kein Mapping der FE-Typen (dropdown) ↔ BE-Typen (select) |
| E — Formulare (No-Code-Form-Builder) | ✅ SEHR MÄCHTIG (BE+FE vollständig) | FE: `modules/formulare/FormularePage.tsx` (5393 Zeilen), `api/formulare-types.ts`, `api/formulare-client.ts`; BE: `backend/internal/formulare/`, `backend/cmd/formulare/main.go`, Gateway `route_formulare.go` | In Cosmi — vollständiger WYSIWYG-Builder | `formulare:schemas:read/write`, `formulare:submissions:read/write`, `formulare:webhooks:read/write` | Share-Link-Backend (FD-4), SMTP-Dispatch (Luke's Lane), embed-Route öffentlich — noch gemockt (MSW) |
| F — Automatisierung (No-Code-Workflow-Builder) | ✅ VOLLSTÄNDIG (BE vollständig, FE vollständig) | FE: `modules/automatisierung/` (Wizard + Editor + ConditionBuilder + ActionConfigurator + TemplateGallery); BE: `backend/internal/automation/trigger/registry.go`, `action/`, `engine/`, `workflow/`; Migration `000052` | In Cosmi — Wizard + Advanced-Editor-Modus | `automations:read/write` | Kein Approval-Schritt (kein `wait_for_approval`-Action). Trigger-Katalog hardcoded (17 Trigger) — kein Nutzer-erweiterbar |
| C — Feld-/Ansichts-Layouts (Detail-Ansicht) | ❌ NICHT EXISTENT | — | — | — | Detail-Ansichten sind fest im Komponenten-Code. Keine DB-Tabelle, kein Layout-Konfigurator, keine `tenant_settings`-Keys für Feld-Reihenfolge/-Sichtbarkeit. Grüne Wiese |
| D — Listen-/Spalten-Konfiguration | ⚠️ PARTIELL (Sortierung ja, Spalten-Sichtbarkeit nein) | FE: `components/shared/SortMenu.tsx` (Feld + Richtung); BE: `saved_filters`-Tabelle (Migration `000012`, mit `tenant_id` seit `000106`, RLS seit `000122`) | In Cosmi — SortMenu in allen Modulen; SavedFilters über API `useSavedFilters.ts` | `saved_filters` — implizit via settings:read/write | Spalten-Sichtbarkeit hardcodiert. Keine Spalten-Toggle-UI. Kein Density-Setting im Backend |
| G — Custom Objects / neue Entitätstypen | ❌ NICHT EXISTENT | — | — | — | Vollständige grüne Wiese. Kein Schema-Builder für eigene Entitäten. Industry-Templates (`000057/058`) sind statische SQL-Seeds, kein Laufzeit-Builder |

---

## Dimension A — Custom Fields: Detailanalyse

### 1. `work_custom_field_definitions` — Task-Custom-Fields (VOLLSTÄNDIG)

**Migration:** `000146_work_custom_field_definitions.up.sql`  
**9 Feldtypen (verifiziert aus Migrations-Check-Constraint UND `backend/internal/work/customfield/models.go`):**

```
text | number | date | boolean | select | multi_select | url | email | phone
```

**Scope:** Nur Tasks/Aufgaben (Work-Modul). Tenant-isoliert via `tenant_id UUID NOT NULL` + RLS-Policy (`tenant_isolation ON work_custom_field_definitions`). Multi-Tenant-sicher.

**Backend:** Vollständiger CRUD-Service unter `backend/internal/work/customfield/` (6 Dateien: `models.go`, `repository.go`, `postgres_repository.go`, `service.go`, `errors.go`, `customfield_test.go`).  
API-Endpunkte unter `GET/POST /api/v1/work/custom-fields` + `GET/PUT/DELETE /api/v1/work/custom-fields/{id}` + Werte-Endpunkte `GET /api/v1/tasks/{id}/custom-fields` + `PUT /api/v1/tasks/{id}/custom-fields`.

**RBAC** (Migration `000147`):
- admin, manager, member: `work_custom_fields:read`, `work_custom_fields:write`
- admin, manager only: `work_custom_fields:delete`

**Frontend:** `WorkCustomFieldsManager` (`desktop/src/renderer/src/modules/work/settings/WorkCustomFieldsManager.tsx`) + `CustomFieldsSection` (`modules/work/components/CustomFieldsSection.tsx`). 
- Manager/Admin definieren Felder über Work-Settings-Panel in Cosmi
- `CustomFieldsSection` rendert Felder auf Task-Ebene (wirksame API-Anbindung via `useTaskCustomFields` + `useSetTaskCustomFields`)
- **Lücke:** `CustomFieldsSection` nutzt noch `/api/v1/custom-fields` (alte CRM-Tabelle, `entity_type=task` Filter), nicht `/api/v1/work/custom-fields` — ein technischer Schuldner, der vor dem Go-Live behoben werden muss

**FE-Typen vs. BE-Typen:** FE-`WorkCustomFieldsManager` kennt 5 Typen (text, number, date, dropdown, checkbox). BE unterstützt 9. Delta: multi_select, url, email, phone fehlen in der FE-UI.

---

### 2. `custom_field_definitions` — Alte CRM-Custom-Fields (TEILWEISE)

**Migration:** `000005_create_custom_field_definitions.up.sql`  
**6 Feldtypen (aus Check-Constraint):**

```
text | number | date | boolean | select | multiselect
```

**Scope:** CRM-Entitäten: `contact`, `company`, `deal`, `activity` (Check-Constraint auf `entity_type`). Kein `task` in dieser Tabelle.

**Tenant-Isolierung:** `tenant_id` via Retrofit-Migration `000106` + RLS aktiviert `000122`. Weniger sauber als `work_custom_field_definitions` (retrofit, kein nativer NOT NULL).

**Backend:** Gateway `route_crm.go` Zeile 39–45: `/api/v1/custom-fields` CRUD (alle 5 Verben). Werte-Tabellen: `contact_custom_field_values`, `company_custom_field_values`, `deal_custom_field_values`, `activity_custom_field_values` — alle per Migration `000007/009/010/026`.

**Frontend (CRM-Modul):**
- `modules/kontakte/CustomFieldsManager.tsx` — CRUD-UI, eingebettet in `CustomFieldsConfig`-Dialog und CRM-Settings-Panel
- 6 FE-Typen: text, number, date, dropdown, checkbox, url
- **KRITISCH:** Store ist `useContactsStore` (Zustand + persist) — keine Live-API-Calls. Änderungen überleben den Browser-Reload, aber nicht einen Device-Wechsel oder anderen User. Nicht production-tauglich.
- `ContactDetailPage.tsx`, `DealDetailPage.tsx`, `CompaniesListPage.tsx` zeigen Custom-Field-Werte an (read path vorhanden).

---

### 3. `task_custom_field_values` (Migration `000026`)

Ältere Werte-Tabelle unter `custom_field_definitions` (entity_type='task' gibt's eigentlich nicht — ist ein Relikt). FK auf `custom_field_definitions.id`. Faktisch totes Relikt neben dem neuen `work_custom_field_definitions`-Stack.

---

### Zusammenfassung Custom Fields

| | Work/Tasks | CRM (Kontakt/Firma/Deal) |
|-|-----------|--------------------------|
| BE vollständig | ✅ (2026-06-11) | ✅ (früh, seit 000005) |
| Tenant-isoliert + RLS | ✅ nativ | ⚠️ Retrofit |
| FE-Manager-UI | ✅ (Work-Settings) | ⚠️ Demo-Store, kein Live-API |
| Anzahl BE-Typen | 9 | 6 |
| Anzahl FE-Typen | 5 | 6 |
| In-Cosmi-konfigurierbar | ✅ | ⚠️ Nur demo-persist |
| Generisch (alle Module) | ❌ nur Tasks | ❌ nur CRM-Entitäten |

**Custom Fields sind NICHT generisch** — es gibt keinen gemeinsamen "Felder zu beliebigen Entitäten hinzufügen"-Layer. Jede Entität hat ihr eigenes Werte-Schema. Für das No-Code-Tool müsste ein generischer Custom-Field-Layer oder eine Erweiterung des Schemas auf weitere Entitäten geplant werden.

---

## Dimension E — Formulare (No-Code-Form-Builder)

**Reifegrad: SEHR MÄCHTIG — am nächsten an "sofort nutzbar"**

### Feldtypen (aus `formulare-types.ts`)
```
text | textarea | email | number | select | radio | checkbox | date | file | consent | rating
```
- `consent` = DSGVO-Einwilligungs-Checkbox mit Datenschutz-Link (vollständig implementiert)
- `rating` = NPS-Skala (5 Sterne oder 10er-Skala) — FE-only (Backend-Proto-Whitelist enthält diese nicht)
- `pageTitle` = Pseudo-Feld für Seiten-Umbruch (mehrstufige Formulare)

### Was heute möglich ist (FE — `FormularePage.tsx`, 5393 Zeilen)
- WYSIWYG Drag-&-Drop-Builder (dnd-kit)
- Mehrsprachige Validierung (`FieldValidation`: minLength, maxLength, min, max, `patternType`: free/plz/phone/iban)
- **Mehrstufige Formulare** (`pageCount`, Seiten-Umbruch-Pseudo-Felder)
- **Bedingte Logik** (`conditionalLogic` pro Feld — fieldId + operator + value)
- Formular-Lifecycle: draft → active → closed → archived
- Submissions-Ansicht mit Status (new/read/archived) + Export (CSV/xlsx)
- Share-Links (link/email/embed/QR) mit Ablaufdatum + Submission-Limit
- Webhook-Konfiguration (Delivery-Log)
- Notification-Config (Empfänger + Bestätigungs-Mail an Einreicher)
- Formular-Vorlagen (`isTemplate: boolean`)
- Thank-You-Message + Redirect-URL nach Submit
- Closed-Message für abgelaufene Formulare

### Backend-Status
Vollständiger gRPC-Microservice (`backend/cmd/formulare/`, `backend/internal/formulare/`). Gateway-Route `route_formulare.go` mit Feature-Flag `modules.formulare`. Echter BE-Service vorhanden.

### Lücken
- Share-Link-Frontend (token + öffentliche Füll-Seite, FD-4) noch MSW-gemockt
- SMTP-Dispatch für Benachrichtigungen = Luke's Lane
- Bedingte Logik ist FE-only (kein Proto-Support; `conditionalLogic?: unknown` im Type = TBD)

---

## Dimension F — Automatisierung (No-Code-Workflow-Builder)

**Reifegrad: VOLLSTÄNDIG (BE + FE) — aber kein Approval-Flow**

### Trigger-Katalog (17 Trigger, hardcoded in `backend/internal/automation/trigger/registry.go`)

| Modul | Trigger-Typen |
|-------|--------------|
| CRM (3) | `crm.deal.stage_changed`, `crm.contact.created`, `crm.deal.assigned` |
| Work (3) | `work.task.created`, `work.task.status_changed`, `work.task.completed` |
| Email (2) | `email.message.received`, `email.message.sent` |
| Finance (3) | `biz.invoice.sent`, `biz.invoice.overdue` (zeitbasiert), `biz.quote.created` |
| HR (2) | `hr.leave.approved`, `hr.shift.ended` |
| Kalender (1) | `calendar.event.upcoming` (zeitbasiert, 15 Min. vorher) |
| Dialer (3) | `dialer.call.outcome_logged`, `dialer.campaign.completed`, `dialer.contact.callback_scheduled` |

### Aktions-Katalog (8 Actions aus den Action-Files)

```
crm.update_deal_field | crm.create_contact | work.create_task |
email.send | notification.send | calendar.create_event |
biz.create_invoice_draft | biz.create_dunning
```

### Bedingungen
- `simple`: field-operator-value (AND/OR verschachtelbar)
- `expression`: freier Ausdruck (Advanced-Modus)
- Test-Condition-Endpunkt + Dry-Run-Endpunkt vorhanden

### Was fehlt
- **Genehmigungsketten** (`wait_for_approval`): nicht implementiert. Kein `approval`-Action-Typ in keiner Action-Datei, kein Proto-Feld
- Trigger-Katalog ist hardcoded — Nutzer können keine eigenen Trigger definieren
- Kein visueller Flow-Graph (kein Reactflow o.Ä.) — Wizard-/Editor-UI ist listenbasiert

---

## Dimension C — Feld-/Ansichts-Layouts

**Reifegrad: GRÜNE WIESE**

Keine einzige Abstraktion für Layout-Konfiguration existiert:
- `desktop/src/renderer/src/` hat kein `LayoutBuilder`, kein `FieldLayout`, kein `detailLayout`-Key in `tenant_settings` oder `user_settings`
- Detail-Ansichten (z.B. `ContactDetailPage.tsx`, `DealDetailPage.tsx`) haben die Felder hardcoded im JSX
- `tenant_settings` + `user_settings` (Migration `000138`) wären das korrekte Fundament — sind aber bisher nur für allgemeine Einstellungen genutzt, nicht für Layout-Config
- Kein Admin-UI zum Konfigurieren von Pflichtfeldern, Feld-Reihenfolge, Sichtbarkeit

---

## Dimension D — Listen-/Spalten-Konfiguration

**Reifegrad: PARTIELL — Sortierung ja, Spalten-Sichtbarkeit nein**

### Was existiert
- `shared/SortMenu.tsx` — universelle Sortier-Komponente (Feld + Richtung). Wird in vielen Modulen eingesetzt (dokumente, formulare, fuhrpark, einkauf, berichte, dialer …)
- `saved_filters`-Tabelle (Migration `000012`) — Suchfilter pro User/Tenant, tenant_id retrofitted `000106`, RLS `000122`
- `useSavedFilters.ts` — API-Hook vorhanden

### Was fehlt
- Spalten-Sichtbarkeit: kein `columnVisibility`, kein Column-Toggle in Listen. Alle Spalten hardcoded im JSX
- Kein "Kompakt/Komfortabel/Tabellarisch"-Density-Toggle im Backend (nur FE-Prefs lokal)
- Keine persisten Spalten-Einstellungen pro User/Tenant

---

## Dimension G — Custom Objects / neue Entitätstypen

**Reifegrad: GRÜNE WIESE**

Keine Infrastruktur für nutzer-definierte Entitäts-Typen:
- `industry_templates` (Migration `000057/058`) speichert Custom-Field-Templates für CRM-Entitäten als JSON-Seeds — aber das sind statische SQL-Rows, kein Laufzeit-Schema-Builder
- Validation Rules (`validation_rules`-Tabelle, `plugin_manifests`) + Workflow Rules (`workflow_rules`) sind Plugin-System-Artefakte (Feature-Flag OFF)
- Kein UI zum Anlegen neuer Entitäts-Klassen. Kein "Objekt-Typ"-Metaschema in der DB

---

## Ergänzende Befunde: Settings-Fundament

Migration `000138` (2026-06-xx) legt das 3-Level-Scope-Modell an:

```
tenant_settings (module_id + key + JSONB value) — Modul-Lead/Admin schreibt
user_settings   (user_id + module_id + key + JSONB value) — User schreibt eigene
tenant_module_leads — wer ist Modul-Lead für welches Modul
```

Dieses Fundament ist **der einzige generische Konfigurations-Layer** im System. Er ist heute für Modul-Einstellungen (Arbeitszeit-Defaults, CRM-Defaults etc.) genutzt — aber NICHT für Feld-Layouts, Spalten-Konfiguration oder Custom-Objekte. Es wäre der natürliche Ort, dort anzuknüpfen.

---

## Plugin-System (Feature-Flag OFF)

Migration `000057` legt vor:
- `plugin_manifests` (slug, settings_schema JSONB, hook_registrations JSONB, wasm_binary)
- `plugin_installations` (tenant_id, manifest_id, settings JSONB, status)
- `validation_rules` (tenant_id, entity_type, field_name, rule_type + rule_config JSONB)
- `workflow_rules` (tenant_id, trigger_event, conditions JSONB, actions JSONB)
- `industry_templates` (slug, custom_fields JSONB, validation_rules JSONB, workflow_rules JSONB)
- `plugin_kv_store`

FE-UI dazu unter `modules/admin/plugins/`: `PluginListPage`, `IndustryTemplateGallery`, `ValidationRulesEditor`, `WorkflowRulesEditor` — vollständige UI vorhanden.

Dieses Plugin-System ist **strukturell das Nearest-Neighbor zum No-Code-Customization-Tool** — es hat bereits Industry-Templates, Validation-Rules per Entität, Workflow-Rules, eine settings_schema-basierte Konfiguration. Aber alles liegt hinter Feature-Flag OFF und ist auf "Plugin installieren" ausgerichtet (extern), nicht auf "Kunde konfiguriert sein Cosmi" (intern).
