# Backend-Gaps für Luke

> Was im Backend fehlt oder nicht „ready zum Verknüpfen" ist, damit das Frontend zu Feature-Parität andocken kann.
> Claude sammelt, Darien reicht an Luke weiter. Stand: Welle 1 (2026-06-01). **Status-Update 2026-06-10 (additiv, ✅-Markierungen):** erledigte Punkte aus Chain PILOT (2026-06-09) + Marathon-Tag-2-Wellen 1+2 (2026-06-10) sind inline markiert.
> Priorität: 🔴 ZFA-Pilot-kritisch · 🟠 wichtig · ⚪ später/Post-Launch.

## ✅ Verifikationslauf 2026-08-02 (Vorbereitung Backend-Nachtlauf 4)

> Diese Datei wurde Punkt für Punkt gegen den **laufenden Code** geprüft: lokale DB auf
> Migrationskopf **268** (als App-Rolle `kmuhub_app`), 780 Pfade in `backend/api/openapi.yaml`,
> Gateway-Registrierungen, FE-Clients + MSW-Handler. Ergebnis: rund zwei Drittel der Einträge
> waren durch die Nachtläufe 1–3 bereits geschlossen. Sie sind unten inline als erledigt markiert;
> hier die Sammelsicht, damit niemand sie erneut als Aufgabe liest.

**Gebaut (Beleg jeweils in Klammern):**

| Bereich | Beleg |
|---|---|
| einkauf `POST /pos/{id}/cancel` | Route + openapi |
| produktion Order↔BOM | Spalte `production_orders.bom_id` |
| inventar Inventur-Modell | `inventur_sessions`/`inventur_counts` + 5 Routen |
| rapporte Aufmaß-Modell | `measurements`/`measurement_positions` + 4 Routen |
| schichten Swap-Anträge | `shift_swap_requests` + `/swap-requests` inkl. approve/reject |
| helpdesk `contact_id`/`org_id`/`source_channel`/`ticket_number`/KB | Spalten + `helpdesk_kb_articles` |
| zeiterfassung komplett | 25 `/hr/time/*`-Routen + `hr_week_approvals` |
| RBAC-Seed-Lücken (helpdesk fein, zeiterfassung, infrastructure, wiki) | 456 Zeilen in `permissions` |
| inbox Status/Threading/Tags/Forward/Canned | 29 `/inbox/*`-Routen + 2 Tabellen |
| berichte Server-PDF, KPI-Endpoint, `report_runs`, Cron, öffentlicher Share | `/berichte/*` + `/public/berichte/reports/{token}` |
| dokumente Kommentare, Share-Links, Activity-Log | 3 Tabellen + Routen |
| finanzen wiederkehrende Rechnungen, OP-Liste, mehrstufiges Mahnwesen, CAMT/MT940 | `/finance/recurring/*`, `/open-items`, `/dunning/{id}/escalate`, `/bank-statements/import` |
| video Breakout-Räume, Recording-Download | 7 `/meetings/{id}/breakout-rooms/*` + `/video/recordings/{id}/download` |
| wiki Share-Token-Routen | `route_wiki.go:85-109` + openapi |
| crm Leads als `lifecycle_stage` | Spalten + `route_crm_leads.go` |
| security/DSGVO (~25 Endpoints) | `route_security.go`, alle Tabellen |

**Als Nicht-Gap widerlegt** — die Forderung wird nicht gebraucht oder wurde bewusst anders entschieden:

- **crm Umsatz-Forecast-Endpoint:** `DealForecastView.tsx` rechnet client-seitig aus
  `usePipelineStages` + `useDeals`. Kein Endpoint nötig.
- **profil Avatar-Upload:** läuft über das generische Presign-Muster (`/files/presign-upload`,
  Spalte `users.avatar_url`).
- **dunning „fatal bei fehlenden Company-Settings":** Fail-closed ist in
  `internal/biz/dunning/service.go:295-320` bewusst kommentiert (GoBD — „sent" muss echte
  Zustellung bedeuten). Entscheidung, kein Bug. Nach dem Merge beobachten, da SMTP jetzt gesetzt ist.

**Neu gefunden, jetzt als Units im Nachtlauf-4-Backlog (Block G):** sechs FE-Aufrufe laufen ins
Leere, weil FE-Pfad und Gateway-Pfad auseinanderlaufen (4× GDPR-Export, `/hr/personnel-documents`,
`/hr/employees/{id}/documents/categories`, `finance/invoices/{id}/mark-paid`,
`notifications/mutes/{id}`, `documents/files/upload`, `email/messages/bulk`); der KPI-Endpoint
`/berichte/kpis` umgeht die Modul-Sichtbarkeit; der Kontakt-Import verwirft die Firma;
`/admin/users*`, HR-Change-Requests, HR-Offboard und die R-6-Overrides fehlen komplett.

> **Methodenhinweis für die nächste Runde:** Der Routen-Diff vom Vormittag deckte nur
> `desktop/src/renderer/src/api/**` ab. In `modules/**` stehen direkte
> ``fetch(`${API_BASE_URL}/api/v1/...`)``-Aufrufe, die kein Client kapselt — dort kamen weitere
> tote Pfade hoch. Immer beide Bäume diffen.

## 🔭 Vorausschau Welle 2 + 3 (Heads-up für Luke, 2026-06-25 — NICHT blockierend)
> Wir bauen Welle 2/3 jetzt **FE mock-first**; daraus entsteht nachgelagerter BE-Bedarf für die spätere Echt-Schaltung. Damit Luke sequenzieren kann (Detail je Modul in den Abschnitten unten `### admin/settings/profil/security`):
- **admin** (größter neuer Block, Sub baut FE grade): Auth-Invite-Flow · User-Account-Mgmt (Liste/Rolle/Deaktivieren) · RBAC-Persistenz (Rollen + Permission-Matrix schreibbar, Modul-Leiter) · Tenant-Provisioning (`POST /tenants`) · **Billing/License-Service** (Plan/Seats/Modul-Aktivierung tenant-weit) · Branding-Persistenz (Logo→S3) · Ressourcen-Monitoring.
- **settings**: P3 Workspace-Defaults (Firmendaten/Währung/Zeitzone/GJ-Start, `tenant_settings` erweitern) · P4 Integrationen-OAuth (Bexio/Lexware/DATEV).
- **profil**: Avatar-Upload (→S3 X-1) · User-Preferences-Persistenz · Presence-Routing.
- **security/DSGVO** (P0): audit-log/sessions/ip-access/retention/dsar/erasure (s.u.) + **Art.30 RoPA** (FE mock-first, BE später).
- **Welle 3 Onboarding/Info-Center**: praktisch KEIN neues Backend — reines FE; Onboarding-/Kurs-Fortschritt läuft über das vorhandene **user-settings** (Migr. 138). Falls serverseitige Kurs-Inhalte gewünscht → später separat.
- **FE-Priorität:** (1) admin Invite + License/Modul-Aktivierung · (2) security-DSGVO (P0) · (3) **S3-Service (X-1)** entsperrt admin-Branding/profil-Avatar/Branchen/vertraege · (4) settings-OAuth.

## 🔴 RBAC-Baukasten (neuer FE-Block ab 2026-07-16 — Luke parallel ab R-1)

> Darien-Entscheid: Custom Roles tenant-scoped + Multi-Rollen pro Account + 3-Ebenen-Capability-Modell über alle 32 Module. Konzept/Recherche: `.planning/rbac-block/KONZEPT.md` + `CAPABILITY-KATALOG.md`. Das bestehende BE-Fundament (roles/permissions/role_permissions/user_roles + `RequirePermission` an allen Modul-Routen + 30 Seed-Migrationen) ist die Basis — folgende Erweiterungen:

- 🔴 **`roles.tenant_id` (NULL = System-Preset) + `based_on`-Referenz** — Custom-Rollen pro Firma, Presets unveränderlich.
- 🔴 **Rollen-Verwaltungs-API:** `GET/POST/PATCH/DELETE /api/v1/admin/roles` + `GET/PUT /api/v1/admin/roles/{id}/permissions` (Matrix lesen/schreiben). Ersetzt den A-2-Mock-Contract.
  - **Contract FINAL (FE R-2 gebaut 2026-07-18 — MSW `mocks/handlers/rbac.ts` = Referenz-Implementierung inkl. Guardrails):**
    - `GET /admin/roles/{id}/permissions` → `{ "roleId": "...", "grants": { "work:task:edit": { "scope": "own|team|all" } } }` (kompletter Grant-Satz EINER Rolle; `role_permissions` + neue scope-Spalte 1:1).
    - `POST /admin/roles` Body `{ name, description, color, basedOn }` → **Create = immer Klon**: Grants von `basedOn` kopieren (Presets + Customs klonbar). 201 `{ role }`.
    - `PATCH /admin/roles/{id}` Body `{ name?, description?, color? }` → `{ role }` · `PUT /admin/roles/{id}/permissions` Body `{ grants }` (Vollersatz) → `{ roleId, grants }` · `DELETE` → 204.
    - `POST /api/v1/users/{id}/roles` Body `{ roleId }` / `DELETE /api/v1/users/{id}/roles/{roleId}` → beide `{ "roles": ["admin", ...] }` (Routen existieren in route_auth.go — Response-Shape angleichen).
    - **`GET /api/v1/admin/users/{id}/permissions`** (NEU) → gleiche Shape wie `me/permissions`, für die Admin-/HR-Ansicht „Effektive Rechte pro User" (Team-Modul + A-1).
    - **Fehler-Codes** (FE mappt auf i18n): `{ "error": "preset_immutable" | "role_limit_reached" | "role_name_exists" | "role_has_members" | "last_admin" | "not_found" }` — Preset-Schutz 403, Konflikte 409. Custom-Limit 20/Tenant; DELETE nur bei 0 Trägern; letzter `admin`-Träger nie entziehbar.
- 🔴 **`GET /api/v1/auth/me/permissions`** — aufgelöste effektive Rechte des eingeloggten Users (Union aller Rollen, inkl. Scope je resource). Die eine Quelle fürs FE-Gating.
  - **Contract FINAL (FE R-1 gebaut 2026-07-18, `desktop/src/renderer/src/api/rbac-types.ts` + MSW `mocks/handlers/rbac.ts` als Referenz-Implementierung):**
    ```json
    { "permissions": {
        "roles": [{ "id": "manager", "name": "Team Lead", "isSystem": true, "color": "hsl(217 91% 60%)" }],
        "capabilities": { "work:task:edit": { "scope": "own|team|all", "sources": ["manager", "hr_admin"] } }
    } }
    ```
    Fehlender Key = verboten (Default-Deny, kein Wildcard — auch admin trägt explizite Grants). `sources` = Rollen-Herkunft für die „Effektive Rechte"-Ansicht (Union: weitester Scope gewinnt, sources kumulieren). Ebene-1-Sichtbarkeit = `<modul>:module:view`.
  - **`GET /api/v1/admin/roles`** → `{ "roles": [Role] }` mit `tenantId|null`, `basedOn`, `isSystem`, `memberCount`, `capabilityCount`.
  - **Preset-Rollen-IDs (7, ersetzen die alten 5):** `admin` · `it_admin` · `hr_admin` · `manager` · `member` · `readonly` · `extern`. **Migrations-Mapping:** `hr`→`hr_admin`, `it_support`→`it_admin`; `admin/manager/member` bleiben (BE-Seeds kompatibel). Grants pro Preset: `mocks/data/rbac.ts` `ROLE_DEFS` (= gewünschter Seed-Inhalt für die BE-Migration).
  - FE hat Client-Fallback (löst Presets lokal aus `user.roles` auf), solange der Endpoint fehlt — App bricht gegen echtes BE nicht.
- 🔴 **Validator-Entkopplung:** `assignRoleRequest` erlaubt nur `oneof=admin manager member` (route_auth.go) → dynamisch gegen roles-Tabelle validieren. FE↔BE-Rollen-Drift beheben (FE kennt hr/it_support).
- 🔴 **Daten-Scope-Dimension** im Grant-Modell: `own / team(reporting_line) / all` pro resource (heute nur resource×action).
- 🔴 **Guardrails serverseitig:** Mindestens-1-Admin · Selbst-Aussperr-Schutz · Privilege-Escalation-Guard (niemand vergibt Rechte über die eigenen hinaus) · Default-Deny für neue Module · `admin:role`×create/edit (IT) getrennt von ×assign (HR).
- 🔴 **Audit-Events für Rechteänderungen** (wer→wem→was→wann, immutable; kann auf `audit_log` aufsetzen).
- 🟠 Permission-Cache/Propagation: Rollen-Änderung muss ohne Re-Login wirken (oder definierter Refresh).
- 🟠 Zentria-Setup-Zugang (GDAP-light): Setup-Rolle mit Ablaufdatum + Audit-Sichtbarkeit für den Kunden.
- ⚪ R-6 später: zeitlich befristete Rollen-Zuweisungen (`user_roles.expires_at`), Vertretungs-Delegation, Gehalts-Feldgruppen.
- 🟠 **R-6 Per-User-Overrides — Design-Vormerkung (Darien-Entscheid 2026-07-19, VOLLE Variante inkl. Entzug):** Beim Grant-Datenmodell die Erweiterungsstelle von Anfang an mitdenken: `user_permission_overrides` (tenant_id, user_id, permission_key, **mode allow|deny**, scope, created_by, timestamps) als Schicht ÜBER der Rollen-Union — Resolution: Rollen-Union wie gehabt, danach Overrides anwenden (deny entfernt Key, allow setzt/erweitert Scope; Override gewinnt pro Key über alle Rollen). `GET /admin/users/{id}/permissions` liefert Override-Herkunft in `sources` mit; CRUD `PUT/DELETE /admin/users/{id}/overrides`; Guardrails serverseitig (Eskalations-Guard, Last-Admin, Selbst-Aussperrung) + Audit-Events pro Override-Änderung. FE-Paket + UI-Bild: `.planning/rbac-block/R6-USER-OVERRIDES-BRIEFING.md` (Umsetzung nach dem RBAC-Sammel-Review).

**R-3 Batch 1 (FE-Enforcement work/documents/crm/finance/wiki, gebaut 2026-07-19) — Seed-Abgleich + neue Gaps:**

- **2 neue Katalog-Keys (in `ROLE_DEFS`-Seed-Vorlage bereits drin):** `crm:deal:delete` (base, scopeable; admin + manager `team`) · `wiki:category:manage` (fine; admin + manager).
- **BE-Seed-Lücke (Abgleich gegen `backend/migrations/*permission*`):** BE kennt für die Batch-1-Ressourcen nur grobe Paare (`contacts|deals|documents|files|finance` × read/write/delete(/admin) + `work_labels`/`work_custom_fields`); **`wiki` hat GAR KEINE Seeds**. Es fehlen alle feinen FE-Capabilities (FE-Katalog `config/capability-catalog.ts` = SSOT): work `task|project`×(create/edit/delete) + `be_assigned/comment/manage_members/time:log/board:export` · documents `download/upload/share/share_link/version/template` · crm `export/import/pipeline/advisory/segment` + `deal:delete` · finance `send/dunning/amounts/export/incoming/settings` · wiki komplett (`article`×CRUD + `publish/share_token/template/category`).
- **Owner-Felder fürs Scope-Enforcement (`own`) fehlen im Wire:** CRM contact/deal/aktivität tragen KEIN `owner_id`/`created_by` (FE fällt bei `own`-Grants auf deny zurück — im Seed hat crm niemand `own`, aber Custom-Rollen könnten) · finance invoice/quote ohne `created_by` (Katalog dort bewusst nicht scopeable) · work: Projekt-API-Typ ohne `owner_id` (Mock-Seed hat es), Task-**Listen**-Responses ohne `created_by`/`reporter_id` → `own` greift in Listen nur über `assignee_id`.
- **Scope `team` wird FE-seitig wie `all` behandelt** (kein Team-Modell am Objekt) — sobald BE `team(reporting_line)` auflöst, FE-Vergleich nachziehen.

**R-3 Batch 2 (FE-Enforcement team-Aktionen / dashboard Ebene-2 / admin+security-Tabs, gebaut 2026-07-19):**

- **Keine neuen Katalog-Keys** — team/admin/security/settings waren seit R-2 vollständig kuratiert.
- **BE-Seed-Abgleich Verwaltung:** FE erzwingt jetzt `admin:user:read/invite/deactivate` · `admin:role:*` · `admin:license/branding/it/integrations/ai:manage` (AdminHub-Tabs) · `security:audit:read` / `security:policy:manage` (retention+password-policy+ip+vault+privacy) / `security:gdpr:execute` (gdpr+dsar) · den kompletten team-Katalog (`employee:create/edit(scope own→userId)/deactivate`, `absence:approve`, `payroll:run`≠`view`, `documents:view/edit`, `salary:view/edit`, `data_personal/data_job:view`). **BE-Routen (route_security.go ~25 Endpoints, admin-Routen) sollten dieselben resource×action-Paare via `RequirePermission` erzwingen** — FE-Gating ist nur UI, die API muss dieselbe Sprache sprechen. Seeds für diese feinen Paare fehlen (BE kennt nur grobe read/write/delete).
- **Alt-Muster entfernt:** 6 security-Pages prüften `user.roles.includes('admin')` hart — jetzt Capability-basiert. BE-seitig darf es kein Pendant geben, das nur die Rolle `admin` akzeptiert (Custom-Rollen mit `security:*`-Grants müssen durchkommen).
- **🟠 Notification-Seeds sind empfänger-agnostisch:** extern/readonly sehen im Benachrichtigungs-Widget CRM-Events („Neuer Deal erstellt") aus den globalen Mock-Seeds. BE-Anforderung (wenn Notifications echt): Benachrichtigungen empfängerbezogen erzeugen UND beim Ausliefern gegen Modul-Sichtbarkeit des Empfängers filtern (kein CRM-Event an Rollen ohne `crm:module:view`).
- team scope-own nutzt `employee.userId` (Auth-Account-ID) — im HR-Wire vorhanden, kein Gap.

**R-3 Batch 3 (FE-Enforcement inventar/einkauf/produktion/vertraege/helpdesk, gebaut 2026-07-19):**

- **Katalog kuratiert (`config/capability-catalog.ts` + `ROLE_DEFS`-Grants = Seed-Vorlage). Subjekt-Namen spiegeln die BE-Seeds** (000084/000185/000241 inventar · 000086/000209 einkauf · 000088/000191 produktion · 000090 vertraege), die **Actions sind feiner als die BE-read/write-Paare**: inventar `item×CRUD / location:read / movement:read+create+adjust / inventur:read+create+count+book / attachment:manage / export:run` · einkauf `po×CRUD + send/approve/cancel/receive / supplier×CRU + deactivate / rating:create / catalog:read / contract:read+call / export:run` · produktion `order×CRUD + start/complete/cancel / workstep:edit / bom:read+create+edit / quality:read+create / machine:read + machine:manage / export:run` · vertraege `contract×CRUD + terminate` · helpdesk `ticket:read(scopeable)+create+reply+edit(scopeable) / kb:manage / canned:manage / stats:view`.
- **🔴 helpdesk hat BE-seitig NUR das grobe Modul-Paar `helpdesk:read/write` (000129)** — der komplette Fein-Katalog fehlt als Seeds UND als RequirePermission-Granularität.
- **🔴 helpdesk Requester-Modell serverseitig:** `ticket:read` scope=own heißt „sieht nur Tickets, bei denen er Requester ODER Assignee ist" — die **Liste muss serverseitig gefiltert** werden (nicht nur 403 am Fremd-Detail); `ticket:edit` scope=own = nur der zugewiesene Agent. FE vergleicht `requester_id`/`assignee_id` (neue DisplayTicket-Felder `requesterId`/`assigneeId`).
- **Owner-Felder-Gap (kein scope-own möglich):** inventar/einkauf/produktion/vertraege tragen KEINE nutzbaren Owner-FKs (`created_by` überall null; produktion `inspector`/`assignee` sind Freitext; vertraege `history[].user` Freitext) → Katalog hat dort bewusst KEINE scopeable-Keys. Falls own-Scopes gewünscht: `created_by` beim Create aus dem Auth-Context befüllen.
- **helpdesk Assign-Flow schreibt Agent-NAMEN als `assignee_id`** (FE-`HELPDESK_AGENTS`-Freitext, Vorbestand) — deckt sich mit dem 06-28-Befund: Ticket-Response braucht echte User-Referenzen + denormalisierte `assignee_name`/`requester_name`. Übergangs-Hack im FE: `USER_DISPLAY_NAMES`-Map in `mocks/data/shared-ids.ts` löst bekannte User-Ids zu Namen auf (mock-only, fliegt mit echtem BE raus).
- **einkauf `po:approve` = Freigabegrenze:** FE zeigt Genehmigen nur mit `einkauf:po:approve` (approvalThreshold aus tenant-Settings) — BE sollte Submit über Threshold ohne approve-Recht ablehnen. `po:send` (draft→submitted) ist vom Erstellen getrennt (Zoho-Books-Muster).
- **produktion:** `requireQcBeforeComplete` (tenant-Setting) bleibt Prozess-Gate ZUSÄTZLICH zu `order:complete` — BE sollte beides prüfen. Neue-Tickets-Handler (MSW) setzt `requester_id` jetzt aus der Demo-Session — BE nimmt den Auth-Context.

**R-3 Batch 4 (FE-Enforcement schichten/fuhrpark/vermietung/rapporte/dialer, gebaut 2026-07-19):**

- **Katalog kuratiert, Subjekte spiegeln die BE-Seeds** (000095/000161 schichten inkl. `swap:create/read/approve` · 000097/000196 fuhrpark · 000099 vermietung · 000093/000100/000164 rapporte inkl. `rapporte:report:approve` · 000068 dialer mit **PLURAL-Subjekten** `campaigns/calls/outcomes/agent`). Actions feiner als die BE-read/write-Paare: schichten `shift:read+publish / assignment:manage / template:read+manage / swap:read(scopeable)+create(scopeable)+approve / export:run` · fuhrpark `vehicle:read+manage / service:read+create / fuel:read+create / trip:read+create / damage:create / gps:read / export:run` · vermietung `object×CRUD / rental:read+create+edit+cancel+handover / inspection:create / export:run` · rapporte `report×CRUD(scopeable)+approve / measurement:read+manage / template:read / export:run` · dialer `campaigns:read+manage / calls:read+write / outcomes:manage / agent:manage`. Mapping: FE faltet rapporte `line`/`attachment` in `report:*` und fuhrpark `document` in `vehicle:read` (Detail zeigt Dokumente read-only).
- **✅ `rapporte:report:approve` (000100) hat jetzt FE-UI** — Genehmigen/Ablehnen im Report-Detail (reject nur mit `review_note`). MSW-Submit erlaubt jetzt `draft→submitted` UND `rejected→submitted` (Resubmit) — BE-Transition-Guard entsprechend auslegen. `reviewer_id` schickt das FE aus der Session — BE sollte ihn aus dem Auth-Context nehmen, nicht dem Body vertrauen.
- **🔴 rapporte Owner-Modell:** Mock-Seeds trugen Freitext-Namen als `author_id` → auf echte User-Ids umgestellt (FE zeigt via `displayUserName()`). BE: `author_id` beim Create aus Auth-Context + **Liste bei `report:read` scope=own serverseitig auf author filtern** (FE filtert nur die Demo-Daten client-seitig).
- **🔴 schichten Swap-own-Modell serverseitig:** `swap:read` scope=own = Liste auf `requested_by_employee_id`/`swap_with_employee_id` == User filtern; `swap:create` nur für die eigene Assignment. Die Seeds existieren (000161) — es fehlen RequirePermission-Verdrahtung + Listen-Filter. `shift:publish`/`assignment:manage`/`template:manage`/`export:run` haben BE-seitig nur die groben `shift/assignment/template`×write-Paare.
- **🟠 fuhrpark `gps:read` = Bewegungsprofile (personenbezogen):** im Preset nur admin+manager (bewusst NICHT it_admin/readonly) — BE sollte die GPS-/Routen-Endpoints separat unter `fuhrpark:gps:read` gaten, nicht unter `vehicle:read`.
- **🔴 Owner-FK-Gap fuhrpark/vermietung (kein scope-own möglich):** trip `driver_name` Freitext (kein `driver_id`-User-FK), `vehicle.assigned_driver_id` überall null, `damage.reported_by` null; vermietung rental ohne `created_by` (`contact_id` = CRM-Kontakt, kein User) → Katalog dort bewusst ohne scopeable-Keys. Falls „Fahrer sieht nur eigene Fahrten" gewünscht: `driver_id` als echte User-Referenz einführen.
- **🟠 dialer:** Campaign-POST setzt `created_by` hardcoded (Mock) — BE nimmt Auth-Context. Zuordnung: `calls:write` deckt den kompletten Agent-Flow (next/dial/outcome/notes/agent-status), `campaigns:manage` die Verwaltung inkl. Queue-Skip/Requeue, Supervisor-Dashboards unter `agent:manage`, Outcome-CRUD unter `outcomes:manage` — BE-Routen entsprechend mappen.
- **Nebenbefund (Vorbestand, nicht RBAC):** CampaignListPage-Edit-Dialog feuert `createMutation` statt `updateCampaign` — der Edit-Pfad war nie verdrahtet (Karten-Dropdown „Bearbeiten" legt real eine neue Kampagne an). Separat fixen.

**R-3 Batch 5 (FE-Enforcement berichte/formulare/automatisierung + Standard-Mini-Kataloge kommunikation/kalender/zeiterfassung/infrastructure, gebaut 2026-07-19 — R-3-ABSCHLUSS):**

- **Katalog kuratiert, Subjekte spiegeln die BE-Seeds wo vorhanden:** berichte **`reports` PLURAL** (000080) · formulare **`schemas`/`submissions`** (000129) · automatisierung **BE-resource heißt `automations` OHNE Modul-Präfix** (000129) — FE-Keys tragen den Präfix (`automatisierung:automations:*`), Gateway-Mapping nötig. Actions feiner als die BE-read/write-Paare: berichte `reports×CRUD(edit/delete scopeable)+publish / schedule:manage / share:manage / datev:read / export:run` · formulare `schemas×CRUD+publish / submissions:read+write / share:manage / export:run` · automatisierung `automations:read+create+edit/delete/toggle(scopeable) / executions:read` · kommunikation `channel/team_inbox/routing/canned/webhook je :manage` · kalender `booking_page:manage / category:manage` · zeiterfassung `team:view / week:approve / corrections:approve / export:run` · infrastructure `service/backup/security/updates je :manage + logs:export`.
- ✅ **Seed-Lücken zeiterfassung + infrastructure GESCHLOSSEN** (verifiziert 2026-08-02: `permissions` führt 456 Zeilen, darunter `zeiterfassung` (5) und `infrastructure` (6); ebenso die zuvor fehlenden `wiki` (13) und die feinen `helpdesk`-Keys (10)). Ursprünglicher Befund: **🔴 zeiterfassung + infrastructure haben GAR KEINE BE-Permission-Seeds** — beide Fein-Kataloge fehlen komplett (Seeds + RequirePermission). zeiterfassung-Genehmigungen (week/corrections) und der DATEV-Zeitdaten-Export MÜSSEN serverseitig gegated werden.
- **🟠 berichte `datev:read` = Finanzdaten-Privacy-Ausnahme** (Muster fuhrpark gps:read): BWA/SuSa nur admin + readonly (Steuerberater-Fall) — bewusst NICHT it_admin/manager (beide ohne finance-Zugang). BE sollte die DATEV-Definitions/-Runs separat unter einem eigenen Recht gaten, nicht unter `reports:read`.
- **🔴 berichte Owner-Modell:** Doc-Seeds trugen Platzhalter `created_by:'u-demo'` → auf echte Roster-User-Ids umgestellt; Create-Handler (documents/definitions/schedules) stempeln jetzt die Demo-Session. BE: `created_by` aus dem Auth-Context + bei `reports:edit/delete` scope=own serverseitig gegen den Autor prüfen (FE versteckt nur Controls).
- **🟠 berichte Dashboard-KPIs folgen der Modul-Sichtbarkeit** (`report-module-visibility.ts`, unbekannte Quellen fail-closed): KPI-Endpoint (`/berichte/kpis`) + Hero-Runs liefern heute ALLE Module — BE muss die KPI-Liste serverseitig nach den Modul-Rechten des Users filtern (sonst holt sich ein member die Umsätze per API).
- **🔴 formulare `createdBy` ist Freitext-NAME** (`'Lena Hoffmann'`/`'Du'`), Submissions `submittedBy` ebenso → kein scope-own möglich, Katalog bewusst ohne scopeable-Keys. Echte User-FKs beim Create aus dem Auth-Context befüllen. formulare-**Webhooks**: BE-Seeds existieren (000129), MSW+Hooks auch — aber die Webhook-UI ist ein **unmounted Stub** → bewusst KEIN Katalog-Key bis die UI kommt.
- **🟠 automatisierung member-Entscheid:** Modul für member KOMPLETT unsichtbar (Flows sind Verwaltung; können Module berühren, die der member nicht sieht). `owner_id` ist schon eine echte User-UUID — „persönliche Automationen für member" ist damit später rein grant-seitig machbar (edit/delete/toggle sind scopeable). it_admin hat automatisierung + infrastructure VOLL (IT-Domäne, Muster helpdesk↔it_admin); hr_admin hat zeiterfassung-Verwaltung VOLL (HR-Domäne, Muster schichten).
- **🟠 kommunikation:** Kanal-ANLEGEN jetzt hinter `channel:manage` (admin/manager/member — Slack-Default; readonly/extern nicht). Chat-interne Owner/Admin-Rechte am Kanal (my_role) bleiben unberührt — BE-Pendant: channels-Create-Route gaten, Member-Verwaltung bleibt channel-rollenbasiert. `webhook:manage` nur admin/it_admin (manager bewusst NICHT — technische Integration). BE hat für chat/channels KEINE Permission-Seeds (nur inbox read/write 000129) → Gap.
- **🟠 kalender:** BE-resource heißt **`booking-pages` MIT Bindestrich** (000136, admin-only) — FE-Key `kalender:booking_page:manage` (Underscore), Mapping dokumentieren; manager braucht die BE-Grants zusätzlich (000136 seedet nur admin). `category:manage`/fremde-Termine-Rechte haben keine BE-Seeds; Owner-Check für fremde Termine fehlt FE-seitig weiter (nur meetings hat BE-Organizer-Check) → bewusst kein `event:manage_others`-Key in diesem Batch.
- **🟠 zeiterfassung Genehmigungs-Loch geschlossen:** CorrectionsView zeigte Genehmigen/Ablehnen fremder Korrekturen für ALLE — jetzt hinter `corrections:approve`; TeamView-Wochen-Freigabe hinter `week:approve`; Team-Tab hinter `team:view` (ersetzt `useIsModuleLead`). Export (CSV/XLSX/**DATEV-Lohn**) hinter `export:run` — vorher völlig ungegated.
- **Nebenbefund (Vorbestand):** berichte `ReportBuilderShell`/`MyReportsLibrary` (No-Code-Builder) sind nirgends gemountet (toter Code, nur PinnedReports nutzt Builder-Definitions) · berichte-Seitenbeschreibung nennt die Zeitplan-Anzahl auch für Rollen ohne `schedule:manage` (Kosmetik).

**R-4 (HR-Datenkategorien-Tiefe, gebaut 2026-07-20 — Personio-Recherche-Gate + 6 Darien-Entscheide, FE = Referenz-Implementierung):**
- **🔴 Scope `team` = echte Reporting-Line (NUR team/HR-Modul):** ganze Vorgesetzten-Linie NACH UNTEN (alle Ebenen, zyklus-sicher via `managerUserId`-Kette), eigene Daten eingeschlossen, NIE nach oben. FE-Resolver: `modules/team/reporting-line.ts` + `useHrScopedCapability` (useTeamPermissions.ts). BE muss denselben Resolver serverseitig bauen (Listen + Detail + Dokumente filtern); alle anderen Module behalten team≈all (dokumentiert).
- **🔴 3 neue Capability-Seeds:** `team:self:propose` (Self-Service-Änderungsantrag; member/manager ja, extern/readonly NEIN — Darien §0.1) · `team:directory:full` (ohne = Mitarbeiterliste/Organigramm nur 2-Ebenen-Umgebung: 2 rauf + 2 runter + Geschwister; admin/hr_admin/it_admin haben ihn) · `team:employee:offboard` (admin/hr_admin). NEU `team:absence_data:view` (scopeable, AKTEN-Schublade Salden/Historie — getrennt vom Kalender-Board `team:absence:read`, das für member ALL bleibt!). Schubladen-Keys data_personal/data_job/salary/documents view+edit sind jetzt **scopeable** (manager: personal/job/documents/absence_data view TEAM, **kein salary** · member: alle view OWN inkl. salary).
- **🔴 Change-Request-Flow (BambooHR-Muster):** `GET/POST /api/v1/hr/change-requests` + `POST .../:id/approve|reject|cancel` — Contract + stateful Referenz in `api/hr-change-requests.ts` + `mocks/handlers/hr-change-requests.ts`. approve mutiert den Employee, reject mit Pflicht-Grund; 409 bei doppeltem pending-Antrag pro Feld. Serverseitig: Genehmigen erfordert data_personal:edit mit Scope-Treffer auf den Antragsteller.
- **🔴 Offboard-Endpoint mit Kaskade (transaktional):** `POST /api/v1/hr/employees/:id/offboard` {lastWorkDay, exitDate, exitType(resignation|termination|mutual_termination|retirement), reason?, backfill, successorUserId?} → HR-Status inactive + Auth-Account deactivated (Login sperren) + Seat frei + Rollen-Zuweisungen entfernen + **managerUserId aller Betroffenen auf successor umhängen** (Pflicht, wenn Reports existieren — schließt Personios Approver-Verwaisungs-Lücke). Kein One-Click: Einstieg nur im Profil, zweistufiger Dialog.
- **🟠 `GET /hr/employees/me` muss session-aware sein** (FE-Mock jetzt via Demo-Session-User) · `GET /hr/employees/me/salary-statements` neu (eigene Abrechnungen, nur mit salary:view own) · Dokument-Kategorie-`visibility` (hr_only/manager/employee) serverseitig erzwingen — FE wertet sie jetzt aus (PersonnelDocuments + Profil-Sektion: Gehalts-Kategorie `hrcat-payroll` zusätzlich hinter salary:view, Manager sieht NIE Lohn-PDFs).
- **🟠 Wizard-Eskalations-Guard serverseitig:** Mitarbeiter-Anlegen darf Rollen nur mit role:assign vergeben; ohne → Default member. Rollen oberhalb der eigenen Rechte nie anbieten/akzeptieren.
- **Nebenbefunde (Vorbestand):** Personalakte-Dokument-Seeds hängen alle an CURRENT_USER und werden für JEDEN Mitarbeiter identisch geliefert (Tim zeigt „Arbeitsvertrag_Stefan_Vogel.pdf") → pro-Employee-Seeds nötig · Anfragen-Tab zeigt Abwesenheits-Requester teils als „Unbekannt" (userId-Drift) · mock-db↔IDS-Kollisionen bestehen weiter (usr-e6 Felix/Tim, usr-e9 Lena/Nina, usr-e14 Sophie/Andrea, usr-e16 Martin=extern-max) — der große Seed-Sweep bleibt offen; R-4 hat chirurgisch gefixt: managerUserId wird jetzt KANONISCH aufgelöst (toHrEmployee), 'Sarah Beck'→usr-e7 gemappt, PersonnelDocuments-Upload sendet weiterhin keine employeeId (Vorbestand) · Trainings-Tab: `t(\`…type.\${x}\`)`-Template-Keys treffen die camelCase-Keys nicht (defaultValue-Fallback deckt nur DE) — Kosmetik.

**R-5 (RBAC: Audit-Live-Events · View-as · Zentria-Setup-Zugang GDAP-light v3 · Branchen-Templates, gebaut 2026-07-21 — FE = Referenz, Modell-Entscheide in `.planning/rbac-block/R5-BRIEFING.md` §0):**
- **🔴 Audit-Write serverseitig an ALLE Rollen-/User-Mutations-Routen** (route_auth.go hat heute KEINEN Audit-Call). Taxonomie = `mocks/data/audit-events.ts`-Kopfkommentar (`role.assigned/revoked`, `role.definition_created/updated/deleted`, `user.invited/deactivated/reactivated/offboarded/view_as`, `vendor_access.*`, `permission.override_*` reserviert für R-6). Old/New-Snapshot als JSONB in `audit_log.details` (`old_value`/`new_value` — FE-Delta-Panel rendert genau diese Felder; Grant-Diffs nur GEÄNDERTE Keys, Referenz `grantsDelta()` in `mocks/handlers/rbac.ts`). Append-only/Hash-Chain existiert (000039/000222) — Interceptor in Gateway-Middleware, nicht pro Handler streuen.
- **🔴 vendor_access-Persistenz + Status-Maschine v3:** Tabellen für `VendorAccessRequest` (Contract: `api/vendor-access-types.ts` — reason/description/ticket_ref/agents/scope[]/requested_start/duration_days≤30/expires_at/status/counter_proposed_start/sensitive_ack/extension_of). Status: `pending → approved|declined|counter_proposed → (Zentria bestätigt) → active → expired|revoked|completed_by_vendor`. **Hard-Cap 30 Tage in der API validieren** (auch für Zentria-seitige Anfragen). Expiry-Job (Auto-Ablauf → Event `vendor_access.expired`). approve mit sensitivem Scope OHNE `sensitive_ack` → 422 (FE-Referenz: `mocks/handlers/vendor-access.ts`).
- **🔴 Scope-Enforcement Zentria-Session:** Capabilities der Zentria-Session werden aus dem BESTÄTIGTEN `scope` abgeleitet (Bereichs-Katalog `VENDOR_ACCESS_AREAS` mit `sensitive`-Flag = Seed-Vorlage; Default-Preset „Setup-Standard" = alle non-sensitive). Zentria-Rolle NIE als Tenant-Rolle zuweisbar, nur über vendor_access wirksam. Scope-Erweiterung = neue Teil-Anfrage (`extension_of`), Zugriff erst nach Kunden-Bestätigung.
- **🔴 Zentria-seitiges Anfrage-Portal = späteres Deliverable:** API-Endpoint für Partner-seitige Anfrage-Erstellung mit Partner-Credential-Auth vorsehen; Ziel-Bild Atlassian-Kopplung (Support-Ticket Kunde→Zentria, Ticket-Close → Auto-Entzug). Im FE-Mock kommen Anfragen als Seeds; `ticket_ref` ist als Freitext-Referenz schon im Modell.
- **🟠 View-as serverseitig = Impersonation-Token:** Key `admin:impersonate:run` (existierte ungenutzt im Katalog, jetzt verdrahtet; admin-only). Guardrails serverseitig: nie auf Accounts mit admin-Rolle, nie auf sich selbst; jeder Start schreibt `user.view_as`-Audit-Event. FE-Mechanik (`stores/viewAs.ts` via Demo-Session-Swap) ist Mock-only!
- **🟠 Audit-Query frisch halten:** FE erzwingt jetzt `staleTime: 0` + `refetchOnMount: 'always'` auf der Audit-Query (useSecurity.ts) — serverseitig kein Caching-Layer vor `GET /security/audit` legen. Kettenprüfung (`/audit/verify`) läuft im Mock nur über die 50 Seeds; serverseitig müssen Live-Einträge normal mitgehasht werden (sequence_num fortlaufend, Mock startet Live bei 51).
- **🟠 Branchen-Templates = reine FE-Instanziierung:** `INDUSTRY_ROLE_TEMPLATES` (`mocks/data/industry-role-templates.ts`, 3 Sets × 4 Rollen, grants voll ausformuliert gegen Katalog-Keys) erzeugt normale Custom-Rollen via bestehendem POST /admin/roles + PUT permissions — **kein BE-Neubau nötig**, aber Template-Katalog als Seed-Vorlage übernehmen. Neuer Katalog-Key `security:vendor_access:manage` (admin-only) braucht BE-Seed.
- **Feld-Ebene-Vermerke (Entscheid „keine Feld-Ebene 1.0" bestätigt):** Handel-Template „Verkauf/Kasse" hat inventar item:read → sieht am Artikel das EK-Preis-FELD · „Lager/Logistik" analog bei einkauf po:read (Preise). Wenn Feld-Maskierung kommt (P2), zuerst diese zwei Fälle.
- **Nebenbefund (Vorbestand):** Audit-Seed-Action `user_role_changed` (000039-Ära) weicht von der neuen Taxonomie (`role.assigned`) ab — BE-seitig auf die neue Taxonomie migrieren oder Alt-Namen im Export dokumentieren.
- **⚠ TAXONOMIE-ABGLEICH NÖTIG (2026-07-21, beide Seiten am selben Tag parallel entstanden):** Lukes `PHASE-1-RBAC-PLAN.md` §4 plant `permission.role_created/updated/deleted` + `permission.assigned/revoked` — das heute GEBAUTE FE (R-5) schreibt/rendert `role.definition_created/updated/deleted` + `role.assigned/revoked` (Label-Map `AuditEventDetailModal.tsx`, Interceptoren `mocks/handlers/rbac.ts`, Store `mocks/data/audit-events.ts`). VOR Welle 1b EIN Schema festlegen — FE-Seite ist committed + QA-verifiziert; `permission.override_*` für R-6 ist auf beiden Seiten identisch reserviert. Umbenennen ist überall billig, nur nicht doppelt fahren.

**R-6 (RBAC: Per-User-Overrides, gebaut 2026-07-21 — VOLLE Variante allow+deny, FE = Referenz, Darien-Entscheide in `.planning/rbac-block/R6-USER-OVERRIDES-BRIEFING.md` §0):**
- **🔴 `user_permission_overrides`-Tabelle** (`tenant_id`, `user_id`, `permission_key`, `mode allow|deny`, `scope own|team|all`, `created_by`, `created_at`, `updated_at`; RLS + tenant-scoped). Contract: `api/rbac-types.ts` (`CapabilityOverride`, `UserOverrides`). CRUD `GET/PUT/DELETE /admin/users/{id}/overrides` (PUT ersetzt die ganze Map, leere Map = alle löschen). FE-Referenz stateful: `USER_OVERRIDES` in `mocks/data/rbac.ts` + Handler in `mocks/handlers/rbac.ts`.
- **🔴 Resolution serverseitig = Rollen-Union DANN Overrides** (eine Naht, `applyUserOverrides()` in `mocks/data/rbac.ts` ist die Referenz): `deny` entfernt den Key komplett, `allow` setzt/hebt den Scope — Override gewinnt pro Key über ALLE Rollen. `GET /auth/me/permissions` + `/admin/users/{id}/permissions` liefern die aufgelösten Capabilities MIT Overrides + zwei Zusatzfelder: `hasOverrides: boolean` (Benutzerlisten-Badge/Filter) und `deniedByOverride: {key, roleScope, sources}[]` (durchgestrichene „persönlich entzogen"-Zeilen in Effektive-Rechte). **`?base=1`-Param liefert die REINE Rollen-Union ohne Overrides** (Editor-Baseline) — serverseitig spiegeln.
- **🔴 Provenance:** allow-Override taggt den Grant mit dem Sentinel `sources: ['override', ...]` (Konstante `OVERRIDE_SOURCE`) → Effektive-Rechte-Chip „Persönlich". deny-Keys wandern in `deniedByOverride`.
- **🟠 Guardrails serverseitig spiegeln (FE erzwingt sie nur als UI):** Eskalation (allow auf Key, den der Setzende selbst nicht hält → blocken) · Selbst-Aussperrung (eigener Account nicht editierbar) · Last-Admin/`admin:*`-deny am letzten Vollzugriff blocken · neuer Katalog-Key `admin:user_override:manage` (admin-only Default, hr_admin hat `role:assign` OHNE dieses Recht) braucht BE-Seed.
- **🟠 Audit:** jede Override-Änderung → `permission.override_set` / `permission.override_removed` (pro geänderten Key, old/new = alter/neuer Override-Stand) via denselben Audit-Interceptor wie R-5. Rollen-Wechsel BEHÄLT Overrides (Entscheid ①: kein Kaskaden-Cleanup serverseitig; der FE-Confirm-Dialog ist reine UX).
- **Datenmodell-Hinweis:** Overrides können Modul-Sichtbarkeit (Ebene 1, `<module>:module:view`) ebenso schalten wie Fein-Rechte (Entscheid ②) — kein Sonderfall im Resolver nötig, `module:view` ist ein normaler Key.

## 🟠 Echt-Schaltung-Befunde 2026-06-24 (Darien, lokale Live-Verifikation)
- **🟠 Modul-Feature-Flags = Deploy-/Auto-Deploy-kritisch:** helpdesk/wiki/berichte/formulare/vertraege/buchhaltung/video + alle Branchen-Module sind im Gateway hinter `modules.*`-Flags (`featureflag/registry.go`, default **OFF**, Env `COSMI_MODULE_<NAME>_ENABLED`). Solange das Env-Var fehlt, sind die Routen **nicht registriert** (404) **und** die FE-Nav blendet das Modul aus — egal wie fertig die FE ist. **→ Für Hetzner/Prod muss `.env.production` die gewünschten `COSMI_MODULE_*_ENABLED=true` setzen, sonst deployt der Auto-Deploy ein Build ohne diese Module.** (Lokal via `deploy/docker/docker-compose.flags.yml`-Override aktivierbar, untracked.)
- **🟠 kommunikation Inbox — notification-Service + `/inbox/messages/unread-count`-Pfad:** Der chat/Team-Teil ist echt (chat-Service, `/channels` 200, rendert sauber). Die **Inbox** liegt im **notification**-Service (`InboxRoutes.ServiceName()="notification"`). `GET /inbox/messages/unread-count` → **400 „invalid id"** — der Pfad matcht offenbar `/inbox/messages/{id}` (kein dediziertes `unread-count`-Route-Segment davor) → „unread-count" wird als UUID-Param abgelehnt. Entweder dedizierte Route vor `/{id}` registrieren oder FE-Pfad korrigieren; + Inbox-Backend (Status/Threading/Tags/Forward/Canned) für die echte Inbox. FE degradiert sauber (kein Crash).
- **🟠 helpdesk `ListTickets` — `missing or invalid tenant_id` (400) im helpdesk-Service:** Mit gültigem Demo-Token (tenant …0001) liefern `helpdesk/queues` etc. **200**, aber `GET /helpdesk/tickets` **400 „missing or invalid tenant_id"**. crm/document mit demselben Token = ok → **ticket-Pfad-spezifisch** (helpdesk-Service `ListTickets` zieht/validiert tenant anders als die übrigen helpdesk-RPCs; tenant erreicht den Service nicht bzw. wird strenger geprüft). Helpdesk-Service hat `TenantInboundUnaryInterceptor` — also vermutlich ListTickets-eigene Validierung oder fehlende Metadata-Weitergabe nur auf diesem RPC. Blockiert die helpdesk-Ticket-Liste echt.

## 🟠 Echt-Schaltung-Befunde 2026-06-28 (Darien, lokale Live-Verifikation, nach Lukes 06-26-Welle)

**✅ Durch Lukes 06-26-Welle gefixt (live bestätigt):** helpdesk `ListTickets` tenant_id (Gateway setzt jetzt `TenantId` aus Context → `GET /helpdesk/tickets` 200) · inbox `/messages/unread-count`-Route (dediziertes Segment vor `/{id}`). Beide aus dem 06-24-Block oben damit erledigt.

**🟢 security/DSGVO — Backend ist viel weiter als „2/10 Endpoints" (Verify-Befund 06-28):** `route_security.go` hat ~25 HTTP-Endpoints (audit/vault/gdpr-export+erasure/password-policy/ip-rules/retention) über echten `auth`-gRPC-Client; alle DB-Tabellen existieren (`audit_log`, `ip_access_rules`, `password_policies`, `retention_policies`, `vault_secrets`, gdpr `data_exports`). Read-only live geprüft: alle 200, `password/policy` liefert echte Row (min_length 12 etc.) — **kein Stub**. audit/retention nur leer (keine Demo-Daten). **→ security ist echt-schaltbar (FE S-1…S-5 gegen echtes Backend), NICHT mehr Luke-blockiert.** Echt-Schaltung = separate Entscheidung (GDPR-Erasure/Execute NICHT getestet, destruktiv). Master-Plan-Eintrag „nur 2/10" ist überholt.

**Neue Befunde (FE-tolerant gefixt wo möglich, Rest = Luke):**

- **🟠 helpdesk — Ticket-Liste zeigt rohe IDs statt Namen (kanonisch Backend):** `WireTicket` liefert nur `assignee_id`/`requester_id` (UUID), keine Namen → die Liste rendert die rohe UUID in „Zugewiesen an". Kein FE-Lookup möglich (kein tenant-weites User-Verzeichnis). **→ Ticket-Response sollte `assignee_name` + `requester_name` denormalisiert mitliefern** (Standard für Helpdesk-Listen). Ebenso fehlen `description` (Detail-Body immer leer) und `category` (Spalte immer leer) komplett im WireTicket. Außerdem: kein `ticket_number`-Feld → das FE bastelt eine Pseudo-Nummer aus den ersten 4 Hex-Chars der UUID (`HD-YYYY-NNNN`, kollisionsanfällig). Sauber wäre eine fortlaufende `ticket_number`-Sequenz pro Tenant.
- **🟠 documents — restliche Wire-Shape-Drift (FE-tolerant abgefangen, kanonisch Gateway):** Ergänzend zum 06-24-dokumente-Block: (a) `POST /documents/shares`, `POST /documents/tags`, `POST /documents/files/{id}/links` antworten **naked** (`resp.Share`/`resp.Tag`/`resp.Link`), FE-Typ erwartet `{share}`/`{tag}`/`{link}` → kanonisch wrappen wie folder/file. (b) `POST .../versions/revert` gibt `{file}`, FE-Typ sagt `{version}` — Shape klären. (c) `GET /documents/shares/shared-with-me` gibt `{files,folders,total}`, FE erwartet `{shares}` → liefert still `[]` (kein Crash); vereinheitlichen. **FE-seitig gefixt (06-28):** Share-List-URL war `/shares` → 405, korrigiert auf `/shares/entity`; `permission` kam als Int (ProtoList `UseEnumNumbers`) → `normalizeShare` mappt 1/2→read/write.
- ✅ **inbox Thread-Verlauf + Canned-Responses — ERLEDIGT** (verifiziert 2026-08-02: `inbox_thread_messages` + `inbox_canned_responses`, Routen `/inbox/messages/{id}/thread`, `/inbox/canned-responses`; zusammen mit `/status`, `/tags`, `/forward` sind damit alle vier Punkte aus dem Phase-4-Block weiter unten geschlossen). Ursprünglicher Befund:
- **🟠 inbox — Thread-Verlauf + Canned-Responses fehlen im Backend:** Nachrichtenliste echt (`/inbox/messages`), aber der **Gesprächs-Thread** ist FE-synthetisch (`buildThreadSeed` + lokaler `inboxThread.ts`) — kein `ListThreadMessages`-RPC. Reply feuert echt, wird aber nur lokal angehängt. Ebenso lokal-only: **Canned-Responses** (kein CRUD), `inboxStatus`/`inboxTags`. Für echte Inbox: Thread-Endpoint + Canned-Response-CRUD.

## 🟠 Echt-Schaltung-Befunde 2026-07-05 (Darien, Verifikation von Lukes 07-05-Quick-Win-Welle)

> Verifiziert nach `git pull` (Lukes CRM-Import/Export-Pfad-Fix `a24d186d`, Video-Incoming-Call `44b23e77`, Dunning-Mail `273f1b6b`). Lokal gegen echtes Backend (crm/gateway neu gebaut, DB Migr. 242).

**CRM Kontakt-CSV-Import — 2 mock-verdeckte Bugs gefunden, beide gefixt (07-05):**
- **✅ FE-seitig gefixt: Wire-Contract-Mismatch Field-Mapping.** Das FE (`crm-import-client.ts`) sendete das Spalten-Mapping als **ein** JSON-Feld `field_mapping`; der Gateway (`route_crm_contacts.go:HandleImportContactsCSV`) liest es aber aus **einzelnen `map_<csvSpalte>=<crmFeld>`-Formularfeldern**. → Mapping kam leer an, **jede** Zeile wurde mit „both first_name and last_name are empty" geskippt (`imported_count:0`). FE sendet jetzt `map_*`-Felder. Live verifiziert: 2/2 importiert, in DB. **Kanonisch (optional, Luke):** Gateway könnte zusätzlich einen `field_mapping`-JSON-Blob akzeptieren (robuster für Spaltennamen mit Leerzeichen/Sonderzeichen im `map_`-Suffix) — nicht nötig, nur Härtung.
- **✅ Backend-seitig gefixt: Auto-Detection-Lücke `snake_case`.** `knownMappings` (`internal/email/contact/import_service.go`) kannte `vorname`/`firstname`/`first name`, aber **nicht `first_name`/`last_name`** (Unterstrich) — und genau so heißen die **Export**-Spalten des CRM selbst. → Export→Import-Round-Trip erkannte die Namen nicht auto. `first_name`/`last_name` ergänzt.
- **🟠 GAP für Luke: Company-Feld beim Import ignoriert + Export leer (Round-Trip kaputt).** `importSingleContact` persistiert nur email/first_name/last_name/phone/position/notes — `fields["company"]` wird komplett ignoriert. Der Export liest `Firma` aus der company-Relation → immer leer, obwohl die CSV/das Mapping eine Firma trägt. **→ Für echten Round-Trip:** company beim Import auf die company-Relation (oder ein Kontakt-Firmenfeld) mappen; Export entsprechend füllen.

**Dunning-Mahnung E-Mail + PDF (`273f1b6b`) — live verifiziert end-to-end, funktioniert + degradiert sauber.** Create (draft) → Send → PDF gegen echtes biz/minio: Send 200 (status→sent), PDF 200 (valides %PDF, 6722 B). biz-Log: „dunning record created" → „system SMTP not configured — dunning notice email suppressed" (Empfänger + Dateiname korrekt) → „dunning notice sent". Nil-Mailer/leerer-Host-Fallback greift wie designed.
- **🟠 Heads-up für Luke (Prod-Risiko, kein lokaler Bug):** `sendAndNotify` bricht bei konfiguriertem Mailer **fatal** ab, wenn `emailNotice` scheitert — und `emailNotice` erfordert **Company-Settings** (`settings == nil → error`) + PDF-Gen vor dem `sent_at`-Update. In Prod mit gesetztem `SYSTEM_SMTP_*` würde ein Tenant **ohne Company-Settings** beim Mahnungs-Send einen 500 bekommen (vorher wurde nur „sent" markiert). Erwägen: Mail-/PDF-Fehler non-fatal machen (sent markieren + Fehler loggen, wie der nil-Mailer-Zweig) **oder** Company-Settings beim Onboarding erzwingen. `created_by`/`sent_by` der Dunning-Records sind zudem nil-UUID (User-ID wird im Handler nicht propagiert — kleiner Audit-Trail-Gap, vorbestehend).

**Video Incoming-Call/Decline (`44b23e77`) — Code-Review: sauber, keine Bugs.** Backend-Round-Trip vollständig: `HandleCreateCall` broadcastet `call.incoming` an Teilnehmer; FE `useIncomingCallListener` → `IncomingCallOverlay`; Decline sendet `call.declined` → `videoWSAdapter.NotifyCallDeclined` → GetCall+EndCall+`BroadcastCallEnded` (reason=declined) → Overlay des Initiators verschwindet. Nur nicht abschließend live-getestet (braucht 2 Clients/echten Anruf); `caller_name`/`avatar` sind serverseitig noch nicht aufgelöst (Broadcast trägt nur `caller_id`, FE fällt auf ID zurück) — **optional (Luke):** User-Lookup in den `call.incoming`-Broadcast aufnehmen.

## ✅ Erledigt seit 2026-06-09 (nicht ursprünglich in dieser Liste)
- **GDPR-Export/-Erasure-Handler echt** (waren Stubs): alle 14 Handler auf echte SQL, tenant+user-gefiltert, Art.-17(3)(e)-Retention (`47d210d9`, 2026-06-10).
- **Beratungsprotokoll ZFA**: `advisory_protocols` Migration 000137 + 7 CRM-RPCs (`6b211222`, 2026-06-10) — Detail-Eintrag unten ebenfalls markiert.

## 🔴 ZFA-Pilot-kritisch

### ✅ kalender — Terminbuchungs-Link (Online-Terminbuchung) — ERLEDIGT 2026-06-09 (Chain PILOT, Migrationen 000135/000136)
ZFA-Akquise hängt an Online-Terminbuchung. FE-Flow existiert komplett als Mock, BE fehlt ganz.
- `GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` — Buchungsseiten (Slug, Services, Verfügbarkeitsregeln)
- `GET /api/v1/public/book/:slug` — **öffentlich/unauthenticated** (Kunde bucht ohne Login)
- `GET .../availability?date=&service=` — freie Slots aus Kalender-Belegung berechnen
- `POST /api/v1/public/bookings` — öffentliche Terminanlage → erzeugt Event + Bestätigung

### ✅ dialer — DSGVO-Consent-Absicherung — ERLEDIGT 2026-06-09 (Chain PILOT, Asserter in `cmd/dialer/main.go` verdrahtet + Regressionstest)
`consentAsserter` ist im Standard-`NewService`-Konstruktor `nil` — nur `NewServiceWithConsent` verdrahtet den Einwilligungs-Check. Prüfen, ob der Standard-Konstruktor irgendwo aktiv ist → sonst Anrufe ohne Consent möglich. Für Finanzberatung heikel.

## 🟠 Wichtig (Kern-Module, ZFA-relevant)

### kontakte
- XLSX/Excel-Import-Endpoint (CSV/vCard existieren, XLSX fehlt)

#### ✅ Kontakte/360° — fehlende Hooks/Endpoints für Kunden-360°-Ansicht — BACKEND ERLEDIGT 2026-06-10 (Welle 2, `52a74373`, Migration 000141): `GET /contracts?contact_id=` (contract_parties-EXISTS) + `GET /finance/invoices?contact_id=` (contact_id-Spalte + Backfill quote→deal→contact). FE-Hooks nachziehen = Claude/FE-Lane.
Folgende Verknüpfungen konnten im ContactDetailPanel NICHT gebaut werden, weil Hooks + Endpoints fehlen:
- **Verträge am Kontakt**: kein `GET /api/v1/contracts?contact_id={id}` und kein Frontend-Hook `useContactContracts(contactId)`. Vertragsservice hat nur generisches CRUD; die Filterung nach `contact_id` fehlt im Modell und der Route.
- **Rechnungen am Kontakt**: kein `GET /api/v1/finance/invoices?contact_id={id}` und kein Frontend-Hook `useContactInvoices(contactId)`. finance_line_items hat keinen direkten Kontakt-FK; Normalisierung (Sprint 4) Voraussetzung.
- Sobald Luke diese Endpoints + contact_id-Felder ergänzt, kann das FE die beiden Sektionen in ContactDetailPanel nachziehen (Muster: analog useDeals mit contact_id-Filter).

### crm
- Lead-Scoring: ✅ **Feld erledigt** (`contacts.lead_score` + `lead_source`/`lead_status`/`lead_temperature`/`lifecycle_stage`, `route_crm_leads.go`); **offen** bleibt nur der serverseitige Berechnungsservice (das FE rechnet in `computeLeadScore`)
- ~~Umsatz-Forecasting: dedizierter `/api/v1/reports/forecast`-Endpoint~~ — **kein Gap** (verifiziert 2026-08-02): `modules/crm/deals/DealForecastView.tsx` rechnet client-seitig aus `usePipelineStages` + `useDeals({page_size:200})`. Erst wenn die Deal-Zahl das clientseitige Limit sprengt, wird daraus ein Endpoint-Bedarf.
- E-Mail-Marketing/Kampagnen: kompletter Service fehlt (`/api/v1/campaigns`) — **weiterhin offen**, bewusst nicht als Nachtlauf-Unit (Neubau, kein FE-Vertrag)

### vertraege
- ✅ **`UploadDocument` — ERLEDIGT 2026-06-11 (`a362b98d`):** Stub-Endpoint entfernt; FE nutzt den generischen Presign-Flow (presign-upload → PUT → PATCH `document_url`). ⚠ Browser-PUT braucht `MINIO_PUBLIC_ENDPOINT` in Prod (siehe §dokumente). FE-Aufrufer von `useUploadDocument` auf `{contractId, file}` umgestellt.
- Audit-Log: `contract_events`-Tabelle (action/user/timestamp/payload) + `GET /contracts/{id}/events`
- Digitale Signatur-Workflow (Phase D, Skribble/DocuSign): `POST /contracts/{id}/send-for-signing`, `/sign`, Status-Endpoint + Webhook-Receiver

### kommunikation (chat + inbox, werden zusammengeführt)
- ✅ **Reaction-Endpoints — ERLEDIGT 2026-06-11 (`c9c19380`):** `POST /api/v1/messages/{id}/reactions` (Toggle) + `GET .../reactions` + `POST /api/v1/messages/reactions/summary` (Batch). Bestehender `work/reaction`-Service in ChatGRPCServer verdrahtet, 501-Stubs aus route_video entfernt, FE + MSW umgestellt. ✅ Follow-up erledigt 2026-06-11 (`507487b9`): `MessageBubble.tsx` nutzt `useToggleReaction`, `MessageList` batch-fetcht via `useReactionSummary`, Demo-Store `stores/chatReactions.ts` gelöscht.
- Chat-Datei-Upload-Route: `POST /api/v1/channels/{id}/files` (Multipart) — Service `Upload()` existiert, Route fehlt
- **Externe Kanal-Verknüpfungen verwaltbar machen** (für Modul-Merge): Settings/CRUD um nicht-interne Kanäle (Mail/WhatsApp/Widget) anzubinden — Routing-Rules-Infra im inbox-Service ist Basis

### dokumente
- 🟠 **Echt-Schaltung-Befund 2026-06-24 (Gateway-Wire-Shape-Inkonsistenz, FE-tolerant gefixt, kanonischer Fix = Backend):** Beim Anbinden des FE an das echte document-Backend kamen mehrere Drift-Punkte hoch, die MSW verdeckt hatte. FE-seitig in `document-client.ts` abgefangen (Normalizer), damit die UI live rendert — **kanonisch gehört das ins Gateway** (`backend/internal/gateway/route_document.go`), dann kann der FE-Normalizer wieder weg:
  1. **List-Responses inkonsistent:** `HandleListFolders`/`ListTags`/`ListShares`/`ListVersions`/`ListEntityLinks`/`ListActivity`/folder-path geben das **bare Array** zurück (`response.JSON(w, …, resp.Folders)`) → bei leer serialisiert protojson das zu **`null`** (nicht `[]`). `HandleListFiles` dagegen wrappt (`{files, total}`). FE-Typen + MSW erwarten überall die gewrappte Form `{folders, total}` etc. → empfohlen: alle List-Handler konsistent in `{<key>, total}` wrappen und leere Slices als `[]` (nicht `null`) emittieren.
  2. **Single-Entity-Responses bare:** `get`/`create`/`copy` (folder+file) geben das **bare Objekt** zurück (`resp.Folder`), FE+MSW erwarten `{folder}`/`{file}`. → konsistent wrappen.
  3. **`POST /documents/folders/initialize-user` verlangt einen Body:** `decodeAndValidate[initializeUserSpaceRequest]` schlägt bei leerem Body mit **400 „invalid request body"** fehl, obwohl `user_id` optional ist. FE sendete keinen Body → init schlug immer fehl. FE schickt jetzt `{}`. → entweder leeren Body tolerieren oder im OpenAPI als pflicht-`{}` dokumentieren.
  4. **protojson-Wire-Shapes (erwartbar, FE normalisiert):** Timestamps `{seconds,nanos}` statt ISO; `space_type` als Enum-**Int** (`1`=personal/`2`=team/`3`=project) statt String; `file_count` fehlt am Folder-Objekt (FE default 0); `file_size` (int64) ggf. als String.
  - **Verifiziert:** READ live gegen lokales document-Backend (Ordner Bilder/Dokumente/Meine Dateien/Vorlagen rendern, keine Crashes/Invalid-Dates, Screenshots `desktop/.qa-screenshots/dokumente-mock-exit/`), Create-Pfad per API (Folder 201 + erscheint in der Liste). **Upload live nicht testbar** lokal (braucht `MINIO_PUBLIC_ENDPOINT`+CORS, s.u.).
- ✅ **Presign-Upload öffentlicher MinIO-Endpoint — CODE ERLEDIGT 2026-06-11 (`1aef2f45`):** `MINIO_PUBLIC_ENDPOINT`/`MINIO_PUBLIC_USE_SSL` + zweiter presign-only minio-go-Client, Caddy-Block `s3.zentria.tech → minio:9000` (docker + ansible `minio_public_domain`), CORS via `mc cors set` (`MINIO_CORS_ALLOW_ORIGIN`). ⚠ **Prod-Rollout offen:** DNS-Eintrag `s3.zentria.tech` (Cloudflare, DNS-only!) + Env in `/opt/kmuhub/.env.production` + Electron-Origin für CORS verifizieren.
- ✅ **Datei-Kommentare — ERLEDIGT** (Lauf 3, `65f5918f`: `document_file_comments` + `/documents/files/{id}/comments` + `/documents/comments/{id}`)
- ✅ **Externe Share-Links — ERLEDIGT** (Lauf 3, `96238d9c`: `document_share_links` mit Passwort + Ablauf, `/documents/files/{id}/share-links`, `/documents/share-links/{id}`, öffentliches Resolve `/api/v1/public/documents/share/{token}`)
- Tenant-Settings dokumente (2026-06-10, Strom D): `stores/dokumenteSettings.ts` ist mock-first (Dateityp-Gruppen, Standard-Freigabe, OnlyOffice-Schalter, Papierkorb-Tage). Settings-Foundation (Migration 138, `route_settings.go`) liegt inzwischen auf main → nach Merge nur noch FE-Wiring auf `tenant_settings`, kein neues Backend nötig. Enforcement der erlaubten Dateitypen beim Upload wäre Backend-seitig sinnvoll (aktuell nur Verwaltung).
- Versionsspezifischer Download (2026-06-10, Strom D): `GET /api/v1/documents/files/{id}/versions/{n}/download` fehlt — der „Herunterladen"-Button im Versionsverlauf kann nur die aktuelle Datei laden.
- Template-Storage (2026-06-10, Strom D): echte Dokument-Vorlagen (.docx/.xlsx/.pptx) + `POST /api/v1/documents/files/from-template/{templateId}` — FE lädt bis dahin generierte Platzhalter-Dateien hoch (TemplateGalleryDialog).
- ✅ **Activity-Log — CODE ERLEDIGT 2026-08-02 (Migration 000264):** `document_file_activity` (append-only, DB-Trigger wie `audit_log`) + `GET /api/v1/documents/files/{id}/activity` + Schreiben bei Upload/Rename/Move/Copy/Download/Share/Version/Revert. Urspr. Gap (2026-06-10, Strom D, aus Darien-Feedback): FE (Viewer-Info-Panel „Aktivität") lief bis dahin mock-first.
- Thumbnail-Rendering (2026-06-10, Strom D, aus Darien-Feedback): Erstseiten-Vorschau für Kacheln (`thumbnail_key` existiert am Modell, Rendering-Service + Abruf-Endpoint fehlen) — FE zeigt bis dahin eine Seiten-Optik.
- ⚠ CSP-Hinweis (kein Gap, Review): `frame-src 'self' blob:` neu in `desktop/src/renderer/index.html` (Dokument-Viewer). Der OnlyOffice-iframe (externe `VITE_ONLYOFFICE_URL`) ist von `frame-src` vermutlich weiterhin blockiert — bei OnlyOffice-Scharfschaltung CSP um die Office-Domain erweitern.

### mails
- Multi-Account: Tabelle + `ListEmailAccounts` (aktuell 1 Account/User)
- Vorlagen/Quicktext: Template-CRUD (`email/template/`)
- Regeln & Filter: `email/rule/` + Endpoints

### helpdesk
- ✅ **`contact_id`/`org_id` ins Ticket-Modell — ERLEDIGT** (Lauf 3, `93cead56`; Spalten in `tickets` verifiziert 2026-08-02)
- ✅ **`source_channel` + Inbox→Ticket-Adapter — ERLEDIGT** (Lauf 3, `a4542dc7`)
- ✅ **Knowledge-Base-Endpoint — ERLEDIGT** (`helpdesk_kb_articles` + Routen)
- ✅ **`ticket_number`-Sequenz — ERLEDIGT** (Spalte + `helpdesk_ticket_counters`; ersetzt den Pseudo-Nummern-Hack im FE)
- `time_spent`-Feld (Ticket-Zeiterfassung) — **weiterhin offen**, Spalte fehlt. Kein FE-Vertrag dafür, deshalb 2026-08-02 bewusst nicht als Unit eingeplant.

### berichte
- Query-Builder: BE-Executor liest `query_config` schon — Editor-Contract festzurren
- `ExecuteKindCross`-Methode im Executor (datenquellen-übergreifend)
- Breakout/Pivot-Schema in `RunReportRequest.Params`
- ✅ **Dashboard-KPIs — BACKEND ERLEDIGT** (`GET /api/v1/berichte/kpis` + `HandleGetDashboardKPIs` → `GetDashboardKPIs`). ⚠ **Aber neuer Sicherheitsbefund 2026-08-02:** der Handler übernimmt die Modulliste ungeprüft aus `?modules=` und kennt nur den Guard `berichte:reports:read` — ein `member` ohne `finanzen:module:view` zieht darüber die Umsatzzahlen. Das FE filtert nur clientseitig (`report-module-visibility.ts`). **Unit `fix-berichte-kpi-module-scope`** im Nachtlauf-4-Backlog.
- **KPI-Zeitreihe für Sparkline (P-nico-02):** Die KPI-Karten zeigen eine Mini-Trendlinie, deren Verlauf aktuell FE-seitig deterministisch aus `kpi.id` + `change_percent` synthetisiert wird (`buildSparklineSeries` in `DashboardGrid.tsx`). Echte Zeitreihe pro KPI (z.B. letzte 8 Perioden) sollte das Backend liefern → dann `sparklineData` aus echten Punkten speisen.
- ✅ **R-3b Server-PDF, R-4 Cron-Executor + Mailer, R-5c externer Token-Zugriff — ALLE DREI ERLEDIGT** (verifiziert 2026-08-02: `/berichte/documents/{id}/export/pdf`, `report_schedules` + `report_runs` + Scheduler — ab dem Merge von PR #16 versenden geplante Berichte real, weil `SYSTEM_SMTP_*` jetzt durchgereicht wird —, `report_share_tokens` + `/berichte/documents/{id}/shares` + öffentliche Leseseite `/api/v1/public/berichte/reports/{token}`). Die PDF-Erzeugung läuft über `internal/berichte/export/pdf.go` (maroto/v2), **nicht** über Playwright/Chromium wie unten skizziert. Ursprüngliche Anforderungen:
- **🔴 R-3b Server-PDF (Bericht-Authoring, 2026-06-20, FE B5-1 fertig):** Lese-Modus hat jetzt „Drucken / Als PDF" über `window.print()` + Print-CSS (`report-print.css`, blendet App-Shell aus, paginiert die A4-Bögen — Chromium-Druckvorschau verifiziert, 2-Seiten-PDF). Das reicht für lokalen Druck/„als PDF speichern". **Backend nötig für echten Server-Download:** `berichte-pdf`-Service (Token-geschützte Render-URL `/berichte/documents/:id/print` ohne Chrome → Playwright `page.pdf({format:'A4'})` → `application/pdf`-Blob), FE-seitig `GET …/documents/:id/export/pdf` → `<a download>`. Schriften eingebettet, Charts als Vektoren.
- **🔴 R-4 Cron-Executor + Mailer (Bericht-Scheduling, 2026-06-20, FE B5-2…B5-5 fertig):** FE-Demo vollständig — Scheduling-Modal am Bericht (Rhythmus-Picker→cron, interne/externe Empfänger, Format aus Tenant-Allowlist, aktiv-Toggle), Lauf-Historie + „Jetzt senden" laufen mock-first (`POST /schedules/:id/run` setzt `last_run_at`/`last_run_status` stateful; `ReportSchedule.definition_id = doc.id` koppelt an das Dokument). **Backend nötig:** Cron-Scheduler der fällige `ReportSchedule`s ausführt (Bericht rendern → PDF/XLSX/CSV → an `recipients` mailen), echte `report_runs`-Persistenz (statt FE-`buildRunHistory`-Seed) + Domain-Allowlist-Enforcement (`tenant_settings` `berichte` `schedule.allowed_domains`) + Release-Gate (nur `status='released'` planbar).
- **🟠 R-5 Integration/Verteilung (Bericht-Authoring, 2026-06-21, FE B6-1…B6-5 fertig):** FE-Demo vollständig — „Teilen"-Menü im Lese-Modus: an Aufgabe anhängen (`POST /tasks/:id/files` mit Verweis-Metadaten, vorhandener Endpunkt), an Kontakt anhängen (neuer stateful `POST /contacts/:id/files`-Mock), als PDF in Dokumente (`POST /documents/files/upload` mit Platzhalter-Blob + „Bericht"-Tag), externer Share-Link (neues `ReportShareToken`-Modell + stateful create/list/revoke). **Backend nötig:** (a) Bericht-Verweis als echter Typ in task-/contact-files (statt `mime_type:'application/cosmi-report'`-Konvention) + Anzeige als „Bericht-Link" in work/CRM; (b) echter PDF-Blob für R-5b (= R-3b Server-PDF statt Platzhalter); (c) **R-5c externer unauth. Zugriff**: öffentliche Token-Lese-Seite (`/share/report/:token` ohne Auth, Passwort-Check, Ablauf-Enforcement, `view_count`-Tracking) — `share_token`-Persistenz serverseitig.

### team
- ✅ **FE↔BE-Shape-Mismatch HR-Employees — ERLEDIGT 2026-06-11 (`67fd78b9`):** Doppelt-toleranter Adapter `adaptEmployee()` in `hr-client.ts` (snake_case vom Gateway, camelCase vom MSW-Demo; Enum akzeptiert Integer/Proto-String/Slug). `ContractType`-Union auf Proto-Wahrheit erweitert (`full_time|part_time|mini_job|intern|temporary`), i18n ×4 + Demo-Daten migriert (`praktikum`→`intern`, `freelance`→`temporary`). Team-Modul-API-Swap ist damit entsperrt.
- ✅ **CreateEmployee-Endpoint — TEILSCHNITT ERLEDIGT 2026-06-11 (`a3ad7158` + Fixes `97f30324`/`c2cc98ad`):** `POST /api/v1/hr/employees` (hr:admin) legt Profil für EXISTIERENDEN User an (`user_id` im Body). Prod-verifiziert 201 mit Schema-Defaults. ⚠ Follow-up: Auth-User-Anlage (Invite-Flow: email + temporary_password + roles, transaktional) — FE-Wizard muss bis dahin auf User-Picker oder zweistufigen Flow (register → create profile) umgestellt werden.
- ✅ **Latenter HR-Read-Layer-Bug GEFIXT 2026-06-11 (`c2cc98ad`):** alle users-JOINs in biz/hr referenzierten die nie existente Spalte `users.display_name` → Namens-Resolution + `GET /hr/employees/me` waren in Prod dead-on-arrival (unsichtbar wegen Demo-Mode). Jetzt `CONCAT_WS(first_name, last_name)`-Fallback auf email. ✅ Test-Gap geschlossen 2026-06-11 (`6ff7989a`): 9 pgtc-Integrationstests für employee/absence/leave (CONCAT_WS-Regression + Cross-Tenant) gegen echtes Migrations-Schema; zusätzlich 16x `req.GetTenantId()`→`middleware.GetTenantID(ctx)` in hr_grpc.go gesweept.
- Onboarding-Workflow-API (Template + Checklist)
- DATEV-**HR-Lohn**-Endpoint (bestehender `route_datev_upload.go` ist nur Buchungsdaten)
- **Lohnvorbereitung / Lohnlauf (P-team, 2026-06-07)** — siehe `team-datev-lohn-spec.md`. FE mock-first gebaut (`PayrollPrepPanel` + `payrollRuns`/`payrollSettings`-Stores). Backend: `payroll_runs` (period, group, status locked/exported, exported_at, employee_count) + **DATEV-Datei-Generierung** (LODAS / Lohn&Gehalt-Format mit Lohnarten + Abwesenheitsschlüssel) bzw. **Lohnimportdatenservice** (DATEVconnect, Akkreditierung). Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe. tenant_settings (module_id='team', key='payroll.*') für Berater-/Mandanten-Nr + Mappings.
- **Lohnauswertungsdatenservice** (Phase 2): Abrechnungen/Auswertungen zurück nach Cosmi importieren.
- ✅ **Demo-Daten-Lücke (modulweit) — ERLEDIGT 2026-06-11 (`7a367047`):** `mocks/handlers/team.ts` auf `EmployeeProfile`-camelCase-Shape angeglichen (`userName` befüllt). Demo zeigt jetzt Namen. ⚠ Echter API-Swap braucht noch einen Shape-Adapter — siehe neuen Gap-Eintrag oben (FE↔BE-Shape-Mismatch).

## ⚪ Später / Post-Launch / Architektur
- dialer: Gesprächsaufzeichnung (recording_url, an Video-Infra gekoppelt), AMD, Predictive (Phase 3 — bewusst)
- crm: Mobile App (PWA-Architekturentscheidung)
- mails: Exchange/EWS, PGP/S-MIME
- formulare: öffentlicher Submit-Endpoint (IsPublic-Flag da), File-Upload-Feldtyp, Submission-Mail

## 🟠 Modul-Editor / Customization (FE Mock-First, „Anpassungen"-Block, ab 2026-07-22)

FE baut den No-Code-Massanfertigungs-Editor mock-first (Overlay-Prinzip wie R-6). Luke-Seite, wenn der FE-Editor review-reif ist:
- **`tenant_customization_drafts`**-Tabelle (sparse `payload` = nur Abweichungen: labels + value-sets + **customFields**; Status-Maschine `draft→scheduled→live→superseded`, `scheduled_at`) + Promotion-**Cron** (steht heute als `runDueScheduledDeploys()`-Mock; terminiertes Deployment = USP). Deploy-Modi now/scheduled/draft.
- **Draft-Overlay serverseitig** nur innerhalb der Editor-Session (4. Schicht über tenant, gewinnt) — Resolver `resolveLabelOverrides(locale, base?, draftOverlay?)` / `resolveValueSet(id, base?, draftOverlay?)` ist die FE-Referenz.
- **Rollback** modul-granular: Snapshot VOR Promotion (tenant-Overlay **+ Custom-Field-Store**), Restore setzt beides zurück. Audit `customization.deploy_live/deploy_scheduled/rolled_back/draft_saved/draft_deleted`.
- **E-3c Custom-Fields im Draft (Snapshot-Diff):** Felder haben eigene BE-Persistenz (`work_custom_field_definitions` + CRM `custom_field_definitions`), liegen NICHT im Overlay. Der Draft trägt pro Entity die **Soll-Feldliste**; Deploy diff't gegen den Live-Store → create/update/delete. Serverseitig gleich: Draft-Feldintent sammeln, bei Promotion transaktional anwenden. **NEU additiv: FE-Mock hat jetzt eine `helpdesk_ticket`-Entity** (3 Seed-Felder) — reine FE-Erweiterung, Lukes bestehende Tabellen unberührt; BE = neue additive Entity-Familie (nichts migrieren, bis Helpdesk-Felder scharf).
- **★ G2 Custom-Field-WERTE am Ticket (2026-07-25):** Das FE rendert die definierten `helpdesk_ticket`-Felder jetzt im Ticket-Detail (Sektion „Zusatzfelder", alle Felder inkl. leer) und als Eingaben im Neu-Dialog — aber der **Wire-`Ticket`-Typ trägt noch KEINE `custom_fields`** (backend-aligned, bewusst nicht erweitert). FE nutzt einen Display-Layer-Overlay (`DEMO_TICKET_CUSTOM_FIELDS` + Session-`createdCustomFields`). **BE-Bedarf, wenn Helpdesk-Felder scharf:** Ticket-Tabelle/DTO bekommt `custom_fields JSONB` (keyed auf field.key), `POST/PUT /tickets` persistieren die Werte, GET liefert sie zurück → Adapter mappt sie in `DisplayTicket.customFields`. Muster generisch für alle Entitäten mit Custom-Fields (nicht nur helpdesk). Konsum-Hook FE = `useModuleCustomFields(entity)` (Draft ⊕ live).
- **★ G3 KB-Content = Block-Dokument (2026-07-25):** Die Helpdesk-Wissensdatenbank läuft jetzt auf der shared Block-Engine (`shared/document`, wie wiki/berichte). `KBArticle.content` speichert ab sofort ein **Block-Doc-JSON** (`DocRow[]`, `JSON.stringify`) statt HTML. BE muss `content` **opak** behandeln (Text/JSONB, kein HTML-Sanitizing nötig — Rendering client-seitig über die Registry). **Legacy-HTML/Plain-Content bleibt lesbar** (FE-Adapter `kbContentToRows` wrappt ihn beim Öffnen in einen Text-Block; erst beim nächsten Speichern wird JSON persistiert). `create/update KB` unverändert (`content: string`). Kein Migrations-Zwang; alte Artikel konvertieren sich beim ersten Editieren.
- **★ Regel Modul-Namen sind unveränderlich (Darien 2026-07-22):** nur INHALT ist anpassbar (Objekt-/Datensatz-Begriffe, Felder). Sidebar-Nav (`layout.navItems.*`) + Modul-Name (`rbac.module.*`) sind KEINE anpassbaren Keys — aus der LABEL_WHITELIST entfernt. Serverseitig: diese Key-Präfixe nie in den Override-Layer aufnehmen.
- **Editierbares Manifest pro Modul** (Vendor-Ebene, welche Module editierbar sind) — später, vor Galerie-Ausbau (E-4).

---

# Welle 2 — System, Produktivität, Finanzen, Automatisierung, Video

## 🔴 Pre-Launch wichtig
- ✅ **security — „Passwort vergessen"-Flow** — ERLEDIGT 2026-06-09 (Chain PILOT, Migration 000134: `password_reset_tokens` + forgot/reset-Endpoints, rate-limited, kein Enumeration-Leak).
- **profil/settings — User-Preferences-Persistenz**: Sprache/Theme/Region nur client-seitig (Store/localStorage). Für Multi-Device BE-Endpoint `GET/PUT /users/preferences`. (Für Electron-Single-Device tolerierbar.)

## 🟠 Wichtig
### admin
- Tenant-Provisioning-Endpoint (`POST /api/v1/tenants`) + Onboarding-Flow
- Super-Admin/System-Level-Rolle (über Tenant hinaus)
- Billing/License-Service (`/api/v1/billing`) — aktuell nur statische Mock-Daten
- Tenant-Ressourcen-Monitoring-API (Metrics intern da, kein HTTP-Endpoint)
- **A-1 Benutzerverwaltung (FE-Mock-First-Batch, Branch `parallel/admin`):** Account/Access-Layer (Login-Accounts ≠ HR-Personalakte in `team`). FE läuft komplett auf MSW (`mocks/handlers/admin.ts`), Contract in `api/admin-types.ts` (`AdminUser`), Hooks in `api/hooks/useAdminUsers.ts`. Beim Echt-Anschluss benötigt:
  - `GET /api/v1/admin/users` → `{ users: AdminUser[] }` (id, firstName, lastName, email, jobTitle, role `RoleId`, status `active|invited|deactivated`, lastLoginAt, invitedAt). Quelle = dieselben Identitäten wie `/api/v1/users` bzw. HR-Roster (eine Person, zwei Layer) — **nicht** doppelte User-Tabelle.
  - `POST /api/v1/admin/users/invite` (echter Invite-Flow: E-Mail-Versand + Single-Use-Token, 7-Tage-Ablauf) → `invited`-Account, Seat-Konsum.
  - `PATCH /api/v1/admin/users/:id` (Rolle/Status) → Gateway-RBAC-Rollenzuweisung + Account-Deaktivierung (Login sperren, Seat freigeben).
  - `POST /api/v1/admin/users/:id/resend-invite`.
  - Seat-Modell: aktive + pending-Invites konsumieren einen Platz (`useTenant().totalSeats`) — Server muss Seat-Limit beim Invite erzwingen (FE warnt nur inline).
- **A-2 RBAC-Matrix (FE-Mock-First, Branch `parallel/admin`):** kanonische Rollen-/Rechte-Matrix (5 feste Rollen aus `@/config/roles`, 7 Capability-Gruppen × 17 Capabilities). FE mock-persisted via `/api/v1/admin/permissions` (`mocks/handlers/admin.ts`), Contract `PermissionGroup`/`PermissionMatrix` in `api/admin-types.ts`. Beim Echt-Anschluss: `GET/PATCH /api/v1/admin/permissions` gegen echte Gateway-RBAC-Persistenz; **echte Enforcement** (Capability-Grants wirken auf Modul-/Aktions-Zugriff, heute nur via statisches `@/config/roles` Nav-Gating). Custom Roles/ABAC = post-1.0.
- **Konsolidierung settings/Main-Lane:** Der Legacy-„Berechtigungen"-Subtab in `settings/tabs/ITAdminTab.tsx` (`PermissionsSection`, Fake-Matrix: falsche Rollen, kein Persist) ist durch die A-2-Matrix abgelöst → sollte entfernt werden (verhindert doppelte Permission-UI im Admin-Hub).
- **A-3 Lizenz/Modul-Aktivierung (FE-Mock-First, Branch `parallel/admin`):** tenant-weite Modul-Aktivierung (Provisioning-Layer, neben dem read-only Billing-Tab). FE mock-persisted via `/api/v1/admin/license` (`mocks/handlers/admin.ts`), Contract `TenantModule` in `api/admin-types.ts`. Beim Echt-Anschluss: `GET/PATCH /api/v1/admin/license` gegen echten Licensing/Provisioning-Service (welche Module tenant-weit gebucht/aktiv, Seat-Caps). Aktivierung muss tatsächlich Modul-Sichtbarkeit/Feature-Flags tenant-weit schalten (heute nur Demo-State). Plan/Seats kommen aus `useTenant` (heute Mock in `useBilling.ts`) — siehe Billing/License-Service-Punkt oben.
- **A-4 Branding (FE-Mock-First, Branch `parallel/admin`):** tenant-weites Branding (Name, Logo, Icon, Akzentfarbe aus Cosmi-Palette). FE persistiert heute in localStorage (`cosmi:brand:*`) + setzt `--brand-accent`. Beim Echt-Anschluss: echter Logo-/Icon-Upload → S3/MinIO + tenant-Branding-Persistenz (Endpoint z.B. `PUT /api/v1/admin/branding`), serverseitige Anwendung (Logo in Topbar, Akzent als Theme-Override). Akzent ist bewusst auf die Swatch-Palette beschränkt (kein freier Hex → kein Theme-Bruch).

### security / DSGVO  (FE-Mock-First-Batch S-1…S-5, Branch `parallel/security`)
> BE existiert weitgehend echt: GDPR-Export/-Erasure-Handler (`47d210d9`, alle 14 auf echte SQL), Audit/Sessions/PW-Policy/IP-Rules/2FA in `route_security.go`/`route_auth.go`. Das FE läuft mock-first (MSW); Verdrahtung gegen das echte BE = später (Claude/FE-Lane).
- **🔴 X-3-Spec-Lücke (alle 31 security/auth-Endpoints):** KEINER ist in `backend/api/openapi.yaml` dokumentiert (Spec endet bei `auth/reset-password`). Betroffen: `/security/audit|vault|gdpr/*|password/*|ip-rules|dsar/search`, `/auth/sessions*|2fa/*`. → openapi-Spec nachziehen, sonst bricht jede Typ-Regen-Runde + jeder Echt-Anschluss erneut.
- **Wire-Shape-Befund (encoding/json über protobuf, snake_case):** Alle Handler nutzen `response.JSON` (nicht `response.Proto`). Konsequenz für den FE-Client beim Echt-Anschluss: (a) **Listen sind gewrappt** — `{secrets:[…]}`, `{rules:[…]}`, `{export_requests:[…]}`, `{sessions:[…]}`, `{policies:[…]}`, `{entries,total}` (nie nacktes Array). FE-Client (`security-client.ts`) ist in S-1 bereits darauf ausgerichtet (entpackt gewrappte Shape). (b) **Timestamps als `{seconds,nanos}`** statt RFC3339 → beim Echt-Anschluss `normalizeWireTimestamps()` (`api/wire-time.ts`) im Client anwenden (MSW liefert ISO, geht durch).
- **🔴 Pfad-Abweichungen FE-Client ↔ echtes BE — BESTÄTIGT 2026-08-02, vier tote Aufrufe:** GDPR-Export-Request `POST /gdpr/export` (Client: `/gdpr/export/request`), Download `GET /gdpr/download/{token}` (Client: `/gdpr/export/{token}/download`), Approve/Deny über `/gdpr/exports/{id}/approve|deny` (Client: `/gdpr/export/{id}/…`). 2FA-Policy-Pfad `policy` (singular) vs Client `policies`. **Fünfter Fund:** `modules/settings/PrivacySettingsTab.tsx:139` verlinkt `/api/v1/gdpr/exports/{id}/download` — komplett ohne `/security`-Präfix. Damit ist der DSGVO-Auskunftspfad über die UI heute nicht bedienbar (fristgebunden!). **Unit `fix-security-gdpr-paths`** im Nachtlauf-4-Backlog; Richtungsentscheid: das Gateway zieht auf die FE-Pfade um.
- **🔌 Verdrahten (nach Echt-Schaltung):** `security-client.ts` + Hooks gegen das echte Backend testen (Demo-Mode aus), Pfade + Wire-Shapes obiger Liste abgleichen, Timestamp-Normalizer einhängen.

### profil
- ✅ **Avatar-Upload — ERLEDIGT/kein Gap** (verifiziert 2026-08-02): läuft über das generische Presign-Muster `POST /api/v1/files/presign-upload` → PUT → `GET /api/v1/files/presign-download`; die Spalte `users.avatar_url` hält den Objekt-Key. Kein eigener Endpoint nötig.

### settings
- Workspace-Branding-Persistenz (`/api/v1/tenant/branding`) — aktuell nur localStorage
- Modul-Aktivierungs-Toggle exponieren (Flag-Registry existiert)

#### ✅ Settings-Fundament (Scope-Hierarchie) — ERLEDIGT 2026-06-10 (Welle 1, `360f92e6`, Migration 000138: `tenant_settings`/`user_settings`/`tenant_module_leads`, 3-Ebenen-Resolve serverseitig, tenant-Writes nur Lead/Admin, co-located im auth-Binary). FE-Umstellung localStorage→Endpoints = nächster Schritt.
3-Ebenen-Modell: **Tenant-Default → Modul-Leiter-Override (tenant-weit) → User-Override (persönlich)**. FE komplett gebaut (`ModuleSettingsShell`, `useIsModuleLead`, `useModuleLeadsStore`), persistiert aktuell nur in localStorage.
- **`tenant_module_leads`-Tabelle** (`tenant_id`, `user_id`, `module_id`, `granted_by`, `granted_at`) — wer ist Modul-Leiter für welches Modul. Admin setzt das im Team-Modul (MemberDetailPanel → „Erweiterte Moduleinstellungen"). Endpoints: `GET /api/v1/tenant/module-leads?user_id=`, `PUT/DELETE .../module-leads/{user_id}/{module_id}`.
- **Settings-Scope-Persistenz**: zwei Ebenen statt einem Blob. `tenant_settings` (`tenant_id`, `module_id`, `key`, `value`) für tenant-weite Defaults (nur Modul-Leiter/Admin schreibbar) + `user_settings` (`user_id`, `module_id`, `key`, `value`) für persönliche Overrides. Resolve-Reihenfolge serverseitig erzwingen (RBAC: tenant-Writes nur mit module-lead/admin).
- Beispiel CalendarSettings: `defaultView/weekStartsOn/defaultReminder` = user-scope; `workStartHour/workEndHour/holidayRegion` = tenant-scope.

### notifications
- E-Mail- + SMS-Kanal im Gateway exponieren (Dispatcher existiert intern)

### work
- `start_date`-Feld im Task-Modell + Proto (für vollwertigen Gantt; aktuell nur due_date)
- Projekt-Portfolio-Entität + Aggregations-Endpoint

### zeiterfassung

> ✅ **KOMPLETT ERLEDIGT — verifiziert 2026-08-02.** Alle unten genannten Endpoints existieren
> (25 Routen unter `/api/v1/hr/time/*`, inkl. `balance`, `entries`, `projects`, `analytics`,
> `team`, `weeks/submit|approve|reject|reopen|status`, `corrections/{id}/approve`,
> `summary/daily|weekly`, `categories`, `templates`, `create-invoice`), dazu die Tabellen
> `hr_work_time_entries`, `hr_break_entries`, `hr_time_projects`, `hr_time_categories`,
> `hr_time_templates`, `hr_week_approvals`. Auch die Permission-Seeds für `zeiterfassung`
> existieren (5 Keys). Der folgende Abschnitt ist Historie.

- **`GET /api/v1/hr/time/balance`** (Stundenkonto-Saldo, kumuliert + Perioden-Übertrag) — **P1 FE-mock-first verdrahtet** (`useTimeBalance`, Shape `{balanceMinutes, asOf, periodStart, targetWeeklyMinutes}`); braucht echten Endpoint.
- Export-API: CSV ist **P4 client-seitig** real; **DATEV-Lohn (LODAS)** + XLSX + PDF brauchen Serverside-Generierung.
- `tenant_settings` für zeiterfassung-Regeln (Wochensoll, Auto-Pause-Schwellen, Rundung, Feiertagsregion) — **P4 FE-mock-first** (`stores/zeiterfassungSettings`).
- **`POST /api/v1/hr/time/entries`** (manueller Eintrag) + **`GET /api/v1/hr/time/projects`** (Projekt-Taxonomie) — **P2 FE-mock-first verdrahtet** (`useCreateTimeEntry`/`useTimeProjects`); brauchen echte Endpoints.
- **`GET /api/v1/hr/time/analytics?range=week|month`** (KPI-/Tagestrend-/Projekt-/Billable-Aggregation) — **P3 FE-mock-first verdrahtet** (`useTimeAnalytics`); echter Aggregations-Endpoint oder client-seitig aus Entries.
- HR-Worktime-Entry um `project_id`/`customer_id`/`service_code` (+ `billable`) erweitern — Projekt-Liste ggf. an work-Projekte/CRM-Kunden koppeln statt eigener Taxonomie.
- **Wochen-Freigabe-Workflow** (submit/approve/reject auf Wochenebene) + **Team-Wochenübersicht** (`GET /hr/time/team`) — **P5 FE-mock-first verdrahtet** (`useSubmitWeek`/`useApproveWeek`/`useRejectWeek`/`useTeamTime`); brauchen echte Endpoints + `time_week_submissions`-Tabelle.
- Weekly-Summary braucht `totalBreakMinutes` (pro Tag + Top-Level) — im echten Endpoint mitliefern (Mock war lückenhaft, P5 gefixt).
- ✅ **Architektur-Entscheidung getroffen (Darien, 2026-06-14):** HR-API = Single Source. P1 hat das Header-Widget auf API konsolidiert (`WorkClockWidget`; `TimeTrackerWidget`+`ClockInButton` gelöscht) und die Demo-Doppelquelle behoben (idle `/hr/time/*`-Handler aus `team.ts` raus → `hr.ts` serviert). Die 10 toten Store-Views werden ab P2/P3 an die HR-API portiert, danach `stores/timetracking.ts` gelöscht. Details `reviews/zeiterfassung.md`.

### wiki
- Share-Token-Routes in `route_wiki.go` registrieren (Repo-Methoden existieren) + öffentl. Read-Endpoint
- Artikel-Templates-Endpoint (FE-Dialog existiert)

### finanzen (Symbiose-Ziel — NICHT Vollersatz, siehe finanzen-buchhaltung-strategy.md)
Strategie: Cosmi macht Vorkette (Angebot→Zahlungseingang) eigenständig, übergibt an DATEV/Bexio. Steuerberater macht Kontierung/Bilanz/USt/Lohn.
- ✅ **DATEV EXTF-Export (DE, Launch-kritisch)** — EXISTIERT KOMPLETT (Befund 2026-06-10: `internal/biz/datev/exporter.go`, EXTF ASCII/CSV Windows-1252, `POST /finance/export/datev`; Liste war veraltet).
- **Bexio-API (CH, Launch-kritisch):** OAuth2, Rechnungen/Kontakte bidirektional sync. *(Status-Check 2026-06-10: Service-Gerüst `internal/biz/bexio/` existiert substanziell — Scope-Abgleich gegen Welle 7 läuft, siehe `.planning/bexio-scope-check.md`.)*
- ✅ **E-Rechnung (Launch-Blocker)** — ERLEDIGT 2026-06-10 (Welle 2, `887d5b36`, Migration 000140): Ausgang existierte (XRechnung-UBL + ZUGFeRD); Eingang neu — `POST /finance/invoices/import` (multipart, CII/UBL-Parser + PDF-Extraktion via pdfcpu) + `finance_incoming_invoices` (received→reviewed→booked/rejected).
- ✅ **GoBD-Belegarchiv (Launch-Blocker)** — ERLEDIGT 2026-06-10 (Welle 2, `45a8ed61`, Migration 000139): `gobd_documents` immutable + SHA-256 + `gobd_document_events` append-only, Retention 31.12.(Jahr+8), Routen `/finance/gobd-archive`, MinIO-Storage.
- ✅ **Wiederkehrende Rechnungen · OP-Liste · mehrstufiges Mahnwesen — ERLEDIGT** (`finance_recurring_invoices`/`_runs` + `/finance/recurring/*`, `/finance/open-items`, `/finance/dunning/{id}/escalate`)
- ✅ **Zahlungsabgleich CAMT.053/MT940 + Matching — ERLEDIGT** (`finance_bank_statements`/`_transactions`, `/bank-statements/import`, `/bank-transactions/{id}/match|reject-match|ignore`). finAPI/HBCI-Banking bleibt Post-Launch.
- `currency`-Feld + Wechselkurslogik (aktuell EUR hardcoded; CHF/USD) — **weiterhin offen** (Spalte `currency` existiert an `purchase_orders`, eine Umrechnungslogik gibt es nirgends)
- BMD-Export (AT) + Lexware/lexoffice-Anbindung (Selbstbucher) — Post-Launch

### ✅ kontakte — Beratungsprotokoll (Finanzberatung, P8) — BACKEND ERLEDIGT 2026-06-10 (Welle 1, `6b211222`, Migration 000137: 57 Spalten/8 Abschnitte, Immutability nach HandOver, `referred_by_contact_id` + `client_segment` A/B/C, 7 CRM-RPCs; PDF-Endpoint = 501-Stub, FE nutzt window.print)
- `advisory_protocols`-Tabelle (contact_id, ~40 Felder über 8 Abschnitte, **immutable nach Aushändigung**, 10-Jahre-Retention, DSGVO Art.6(1)(c)). Endpoints CRUD + PDF-Export (Aushändigung).
- „Empfohlen von"-Feld am Contact (Self-Referenz) + Empfehler-Report-Aggregation.
- Mandanten-Segment A/B/C (regelbasiert nach Umsatzpotenzial) — Feld + Berechnungsregel.

#### kontakte P7/P8 — FE-Finish (2026-06-09): mock-first → Backend-Persistenz
FE ist jetzt komplett (UI + Verdrahtung). Folgende Stores sind localStorage und müssen server-seitig/tenant-weit persistiert werden:
- **Beratungsprotokoll-PDF:** FE generiert die Geeignetheitserklärung jetzt per `window.print()` (dauerhafter Datenträger, MiFID II/§64 WpHG/FinVermV). **Backend bleibt nötig:** server-seitige PDF-Generierung + **revisionssichere, unveränderliche Ablage** finalisierter Protokolle (10 J.) — `window.print` erfüllt die Aufbewahrungspflicht NICHT (`useAdvisoryProtocolsStore`).
- **Lead-Scoring-Konfiguration:** Punkte-Regeln + Schwellen jetzt konfigurierbar (`useLeadScoringStore`) → `tenant_settings (module_id='crm', key='leadScoring.*')`. (Score-Feld/Engine am Lead siehe oben.)
- **Manuelle Segment-Überschreibung:** `useSegmentOverrideStore` pro Kontakt → Feld `contacts.segment_override` (ergänzt die regelbasierte Berechnung).
- **CustomFields-Definitionen (CRM):** `useContactsStore` localStorage → Definitions-CRUD-Endpoint (analog work, s.u.).
- **Tags:** `/api/v1/tags` in OpenAPI-Spec aufnehmen (aktuell raw-fetch, untypisiert).
- Entfernt: `NewsletterPanel` (toter Mock-Stub) — ein echtes Newsletter-/Kampagnen-Feature bräuchte ein E-Mail-Kampagnen-Backend.

### automatisierung
- Branch-/Merge-Step im Workflow-Modell + Engine (aktuell sequenziell) — offen
- `http_request`-Action + inbound `webhook.received`-Trigger — offen, **als Units im Nachtlauf-4-Backlog** (`g-automation-http-action` ist SSRF-kritisch, `g-automation-webhook-trigger`)
- Zeitbasierte/Cron-Trigger: Poller-Integration verifizieren/aktivieren — offen, **Unit `g-automation-cron-poller`**

### video / meetings
- ✅ **Breakout-Räume — ERLEDIGT** (7 Routen `/meetings/{id}/breakout-rooms/*` + `meeting_breakout_rooms`/`_assignments`)
- ✅ **Recording-Download/List — ERLEDIGT** (`route_video.go:122` `GET /recordings/{id}/download` + `/video/recordings`; offen bleibt nur die Review-Frage, ob die Berechtigung gegen die Teilnehmerliste statt nur gegen den Tenant prüft)
- Meeting-Recurrence-Logik — **weiterhin offen** (`/calendar/events/{id}/recurring` deckt nur den Kalender ab)

## ⚪ Später / Post-Launch
- buchhaltung (Brücke): automatische Kontierung (Kontenplan SKR03/04), EÜR-Endpoint, Steuerberater-Rolle (read-only)
- security: SSO (SAML/OAuth2/OIDC), Federation (LDAP/AD), WebAuthn/Passkeys
- admin/dashboard: zentraler modul-übergreifender Activity-Feed-Aggregator

---

# Welle 3 — Branchen-Module (Post-Launch / Solar-Pilot)

## ⭐ Cross-cutting (einmal bauen → viele Module profitieren)
- **S3/MinIO-Foto-Upload-Service**: generischer Upload-Endpoint für Foto-Anhänge — gebraucht von fuhrpark (Schaden), inventar (Bewegung), rapporte (Doku), vermietung (Protokoll), chat, profil (Avatar). Aktuell überall Mock.
- **Signatur-Persistenz**: `signature`-Feld/-Endpoint — gebraucht von rapporte, vermietung, vertraege. SignatureCanvas existiert im FE, BE nimmt es nirgends an.
- **Einkauf↔Inventar-Sync**: Wareneingang (`einkauf.ReceiveGoods`) muss `inventar.RecordMovement` triggern (Code-Kommentar „Sprint-3-Item").

## 🟠 Branchen-BE-Lücken (für Solar-Pilot ab Nov relevant)
### fuhrpark
- Führerscheinkontrolle-Modell (`LicenseCheck`: Fahrer, Ablauf, letzter Check) + Route — **offen, verifiziert** (keine Tabelle mit `license` im Namen); **Unit `g-fuhrpark-license-check`** im Nachtlauf-4-Backlog
- Fahrtenbuch: ✅ `trip_logs` + `/fuhrpark/trip-logs` existieren; **offen** bleibt nur der finanzamtkonforme PDF-Export
- Fahrzeugbuchung/Pool (`VehicleBooking` + Conflict-Check) — **weiterhin offen** (`machine_bookings`/`resource_bookings` gibt es, für Fahrzeuge nichts)
- ✅ **`FuelRecord` (Tankprotokoll) — ERLEDIGT** (`fuel_logs` + `/fuhrpark/fuel-logs`); Tankkartenverwaltung offen
- ✅ **GPS/Telematik — ERLEDIGT** (`gps_positions` + `/fuhrpark/gps/ingest|positions|routes`)

### inventar
- `batch_number`/`serial_numbers` im Item-Modell (Chargen/Seriennummern) — **weiterhin offen**
- ✅ **Inventur-Modell — ERLEDIGT** (`inventur_sessions`/`inventur_counts` + `/inventar/inventur` inkl. `/counts`, `/status`, `/book`)
- Kommissionierung/Picklisten-Modell — **weiterhin offen**

### vermietung
- Strukturiertes Checklist-Format für Zustandsprotokolle (BE nur notes+photo_urls)
- `signature_url` im Inspection-Modell
- Online-Buchungsportal (öffentl. Endpoint/Embed)
- Tarif-Erweiterung (Wochensatz, Staffeln, Saison)

### einkauf
- ~~`SupplierRating`-Modell~~ ✅ · ~~`FrameworkContract`-Modell + Katalogartikel~~ ✅ (BE + Client + MSW vorhanden, FE seit Demo-Tiefe 2026-07-16 verdrahtet: Rating-Formular, Abrufe via `CreateContractCall`)
- ✅ **`POST /pos/{id}/cancel` — ERLEDIGT** (Lauf 1, `route_einkauf.go:91` + openapi; verifiziert 2026-08-02)
- 🔒 **`PurchaseOrder.total_amount` wird nie berechnet**: `CreatePO` setzt `"0"` (`service.go:317`), Add/Update/DeletePOLine rechnen den Kopfbetrag nicht nach → gegen echtes BE zeigen alle Bestellungen 0,00 €. MSW macht das Recompute (Netto-Zeilensumme) vor — gleiche Semantik ins BE. **Verifiziert 2026-08-02, Unit `fix-einkauf-po-total`** im Nachtlauf-4-Backlog.
- 🔒 **`ExportPO` ist Stub** (service.go:716): FE erzeugt Bestell-PDF/CSVs seit 2026-07-16 client-seitig (`modules/einkauf/einkauf-export.ts`), `exportPO` aus dem FE-Client entfernt. BE-Endpoint kann gestrichen oder später echt (Briefpapier/Versand an Lieferant) gebaut werden.
- 2-stufiger Bestellfreigabe-Workflow (`approved_by`, `/approve`-Endpoint) — FE nutzt aktuell `einkaufTenant.approvalThreshold` (Settings) + Freigeben=Submit
- Automatische Bestellvorschläge (Inventar-MinQty → PO) — FE hat jetzt Katalog-Warenkorb mit Bündelung pro Lieferant als Vorstufe

### produktion (Brücke — MRP-Tiefe bewusst begrenzen)

> Stand 2026-07-16 (Demo-Tiefe-Session): BOM/WorkSteps/Machines/Quality-CRUD + Start/Complete/Cancel existieren im BE komplett (`route_produktion_ext.go`, Service mit Transition-Guards + Tests) — die früheren Zeilen dazu waren veraltet. Verbleibende echte Gaps:

- ✅ **Order↔BOM-Verknüpfung — ERLEDIGT** (Spalte `production_orders.bom_id` verifiziert 2026-08-02). Ursprünglicher Befund (Demo-Tiefe 2026-07-16): `production_orders` hatte kein `bom_id`, Create-/UpdateOrderInput nahmen keines an. FE-Auftragsdetail (Stückliste-Sektion, Materialverfügbarkeit, Laufkarten-PDF-Materialbedarf) hängt daran — läuft mock-first (`CreateOrderInput.bom_id` FE-seitig ergänzt, MSW persistiert). → Spalte + Create/Update-Feld + Response.
- Material-Verfügbarkeit: Inventar-Abgleich (FE nutzt deterministischen Fake-Hash, `produktion-shared.ts getMaterialAvailability`)
- Kalkulation (Soll/Ist-Kosten auf Order/Steps — Katana/MRPeasy-Parität)
- `progress`/`scrap` braucht KEIN BE-Feld mehr: FE berechnet Fortschritt aus WorkSteps (completed/total) und Ausschuss aus QualityChecks (Σ defects_found) — bewusste Ableitung statt Denormalisierung

### schichten (Self-Service Pilot-kritisch)
- ✅ **`shift_swap_requests`-Modell + approve/reject — ERLEDIGT** (Tabelle + `/schichten/swap-requests` inkl. `/approve`, `/reject`)
- Availability-Tabelle (employee × weekday) + Qualifikations-Tabelle — **weiterhin offen**
- Echter regelbasierter Auto-Planer (ApplyTemplate ist nur Datums-Kopie) — **weiterhin offen**
- `is_minor`-Flag + JArbSchG-Compliance — **teilweise**: `/schichten/compliance` existiert, das Flag am Mitarbeiter nicht

### rapporte (Solar-Außendienst)
- ✅ **Signatur-Persistenz — ERLEDIGT** (`/rapporte/reports/{id}/signature`, ebenso `/vermietung/rentals/{id}/signature` und `/vertraege/contracts/{id}/signature` — der cross-cutting-Punkt ist damit für alle drei Module abgehakt)
- ✅ **Aufmaß-Modell — ERLEDIGT** (`measurements`/`measurement_positions` + 4 Routen)
- `weather`-Feld auf WorkReport (Bautagesbericht) — **weiterhin offen**
- ReportLine: Material-vs-Leistung-Unterscheidung schärfen

## 🟠 Mobile/Offline (Solar-Pilot)
- PWA/Mobile-Zugangsweg (App ist Electron-Desktop) — rapporte + schichten brauchen Außendienst-Zugang
- `rapporte-client.ts` + `schichten-client` auf offline-queue-fähigen Basis-`client.ts` umstellen (offline-queue.ts existiert)

## 🟡 Leads (CRM, Phase 4 — 2026-06-06)

> ✅ **Datenmodell + Endpoints ERLEDIGT — verifiziert 2026-08-02.** `contacts` trägt
> `lifecycle_stage`, `lead_source`, `lead_score`, `lead_status`, `lead_temperature`, dazu
> `route_crm_leads.go` (+ Tests). **Offen bleiben zwei Punkte:** das serverseitige Spiegeln der
> Scoring-Regel (das FE rechnet in `computeLeadScore`) und die Dialer→Lead-Verknüpfung.

- **Architektur-Entscheidung:** Lead als **Kontakt-Lifecycle-Status** modellieren, nicht als separate Tabelle. Empfehlung: `contacts.lifecycle_stage` (`lead` → `qualified` → `customer`). Frontend baut die Inbox als gefilterte Sicht.
- Lead-Metadaten am Kontakt: `lead_source` (manual/csv/dialer), `lead_score` (0–100, auto), `lead_temperature_override` (hot/warm/cold, sticky), `lead_status` (new/contacted/qualified/disqualified).
- Endpoints: `GET /api/v1/leads` (= Kontakte mit lifecycle=lead), Status-/Temperatur-Patch, Convert (lead → contact + opt. company + opt. deal). FE nutzt aktuell In-Memory-Mock (`api/hooks/useLeads.ts`) — swapbar.
- Scoring-Regel serverseitig spiegeln (FE-Logik in `computeLeadScore`): Quelle-Basis + Vollständigkeits-Boni.
- Quelle „Dialer": Dialer-Outcomes mit Rückruf-Wunsch sollten automatisch Leads erzeugen (Verknüpfung Dialer↔Leads).

---

## 🟠 kommunikation (Team-Chat + Posteingang, vereintes Modul, Stand 2026-06-08)

> Beim Scharfschalten: echte gRPC-Endpoints existieren überwiegend, aber im **Demo-Mode (MSW)** fehlen Handler — FE-seitig nachgebaut wo nötig. Echtes Backend / Verkabelung durch Luke:

- **🔴 Chat-Reactions NICHT verdrahtet (Luke):** chat-proto deklariert `ToggleReaction/ListReactions/GetReactionSummary`, aber der **chat-Service implementiert sie nicht** (kein `reaction`-Ordner in `internal/chat/`) und das **Gateway exponiert keine chat-Reactions-Route** (route_chat.go hat keine `/reactions`). Einzige echte Impl = `internal/work/reaction` (video/work). `MessageInfo` hat zudem kein `reactions`-Feld (weder proto noch OpenAPI). → To-do Luke: (a) Reactions am chat-Service implementieren ODER work-reaction-Service mitnutzen, (b) Gateway-Route z.B. `POST/GET /api/v1/messages/{id}/reactions`, (c) `reactions` in `MessageInfo` + OpenAPI aufnehmen. **FE-Status (Update 2026-06-11, `507487b9`):** ✅ Migration abgeschlossen — MessageBubble/MessageList nutzen die echten Hooks (`useToggleReaction` + `useReactionSummary`-Batch), der Demo-Store `stores/chatReactions.ts` ist gelöscht; MSW-Handler bedienen die Reaction-Endpoints in-memory.
- **Chat-Volltextsuche:** `GET /api/v1/chat/search?q=&channel_id=` (SearchChat-RPC) existiert im Backend; FE-Hook `useChatSearch` + Demo-Handler (durchsucht Mock-Messages) jetzt gebaut. Realer Index/Ranking + File-Treffer = Luke.
- **✅ File-Upload im Chat (FE verdrahtet):** `POST /api/v1/files/upload` (multipart, nimmt `channel_id` + optional `message_id`) + `GET /{id}/files` + download/thumbnail existieren, `MessageInfo.files` (FileInfo) ist in proto+OpenAPI. FE lädt jetzt beim Senden via `useChatFileUpload` mit `message_id` hoch und rendert `message.files`. **Luke (optional/nice):** echter Storage/Virus-Scan/Thumbnail-Gen statt Stub; `GetFileThumbnailURL` für Bild-Previews verdrahten.
- **Gruppen-DMs, Pin/Lesezeichen, Channel-Notification-Settings:** im chat-proto nicht vorhanden — Neubau.
- **✅ Mentions-Inbox (FE verdrahtet):** `GET /api/v1/messages/mentions` (`GetUserMentions`) + `UserMentionsResponse` in OpenAPI. FE-Hook `useUserMentions` + `MentionsPanel` + Demo-Handler gebaut. Real out-of-the-box.
- **✅ Posteingang (Inbox) scharfgeschaltet (Phase 4, 2026-06-08, FE):** Snooze/Claim/Assign verdrahtet (SnoozePopover/Assignee-Picker via `useEmployees`), Bulk-Toolbar (BulkMarkRead/Archive), Team-Postfächer + Routing-Regeln in `KommunikationSettingsPanel` (FÜR-ALLE, tenant-scoped). **Demo-Handler (MSW) ergänzt** für `snooze`/`unsnooze`/`claim` + Team-Inbox-CRUD+Members + Routing-Rules-CRUD+test (zustandsbehaftet). Echte gRPC-Endpoints existieren — Luke verdrahtet Gateway/Service falls noch offen.
  - **✅ Backend-Neubau ERLEDIGT (verifiziert 2026-08-02)** — Status, Threading, Tags und Forward haben eigene Routen (`/inbox/messages/{id}/status|tags|thread|forward`), SLA-Tracking bleibt als einziger Punkt offen (`sla_policies` existiert, Verdrahtung nicht geprüft). Ursprüngliche Liste:
    - **Inbox-Status** (offen/wartend/gelöst/geschlossen): kein Feld in `InboxMessage`/proto. FE-Overlay `stores/inboxStatus.ts` (persistiert). → `status`-Feld + Filter im `ListMessagesRequest` + Set-Status-RPC.
    - **Threading** (mehrere Msg/Conversation): kein Thread-RPC. FE synthetisiert Seed + persistiert Replies/Notizen in `stores/inboxThread.ts`. → `ListThreadMessages`/Conversation-Modell.
    - **Tags-CRUD:** `repeated string tags` existiert, aber kein Add/Remove-RPC. FE-Overlay `stores/inboxTags.ts`. → `AddTag`/`RemoveTag`-RPC.
    - **Forward:** kein RPC. FE = `ForwardDialog` (Empfänger+Notiz) mit Erfolgs-Toast. → `ForwardMessage`-RPC.
    - **SLA-Tracking:** noch nicht modelliert.
- **Audio/Video aus Chat + Bots/Webhooks/Slash-Commands:** keine RPCs — kompletter Neubau.
- **✅ Synergie + Moduleinstellungen (Phase 5, 2026-06-08, FE):**
  - **Interne Notizen + @Mention im Kunden-Thread:** verdrahtet — `InternalNoteComposer`/`ReplyComposer` nutzen jetzt `MentionTextarea` (wraps chat-`MentionAutocomplete`); interne Notizen landen als `direction:'internal'` im Thread-Overlay (`stores/inboxThread.ts`). **Backend nötig:** interne-Notiz-Persistenz + echtes @Mention-Notify. ✅ Demo zeigt jetzt Namen (2026-06-11, `7a367047`) — Mock-Handler auf camelCase angeglichen. ⚠ Echter API-Swap braucht noch Shape-Adapter (siehe §team FE↔BE-Shape-Mismatch).
  - **Collision-Hinweis „X bearbeitet gerade":** Mock-first deterministisch (`lib/inbox-collision.ts`). **Backend nötig:** Live-Presence-pro-Conversation (Viewers-Event).
  - **Call-Bridge:** Audio/Video-Buttons im Thread-Header → `useMeetingsStore().startCall` (gleicher Pfad wie Team/Kontakte). Echte LiveKit-Calls aus dem Posteingang heraus brauchen `createCall`-Verkabelung + ggf. Kunde→user_id-Mapping.
  - **Slash-Commands + Webhooks:** Mock-Shell (`SlashCommandPalette` /giphy /umfrage /erinnerung; `WebhookConfig`). **Backend nötig:** Bot/Command-Runtime + Webhook-CRUD/-Delivery.
  - **Canned Responses:** CRUD-UI (`CannedResponseManager`) auf Store-Backing (`updateCannedResponse` ergänzt). **Backend nötig:** Canned-Response-CRUD-Endpoints.
  - **Channels-Connect (`ChannelSettingsDialog`):** Mock-Connect (Toast). **Backend nötig:** echte OAuth/Connect-Flows E-Mail/WhatsApp/Widget.
  - **Per-Channel-Mute / eigener Status:** FE-Prefs (`kommunikationPrefs.mutedChannels`, `presence.myStatus`). **Backend nötig:** serverseitige Notification-Routing-Beachtung.

## 🟠 work (Projekte/Aufgaben, Stand 2026-06-08)

> **P1 (Demo-Mode):** MSW-Handler in `mocks/handlers/work.ts` komplett (zustandsbehaftet) — Demo läuft. Kein Backend-Bedarf, das ist nur Mock.
> **P2 (WorkSettingsPanel, settings-komplett):** Persönliche Prefs = lokal (`stores/workPrefs`, kein Backend nötig). Tenant-Settings laufen **mock-first** (`stores/workSettings`, lokal persistiert) — brauchen echtes Backend:

- ✅ **Label-Taxonomie + Task-Labels — BACKEND ERLEDIGT 2026-06-11 (`2b8447b6`, Migrationen 000145+000147):** `work_labels`+`task_labels` (RLS), Label-CRUD `/api/v1/work/labels`, `PUT /tasks/{id}/labels`, `label_ids` im TaskProto, Permission-Seeds `work_labels:*`. ✅ Follow-ups erledigt 2026-06-11 (`d028b8ea`): `label_ids` batch-geladen in Get/ListTasks (GetLabelsByTaskIDs, 1 Query), `filter_label_ids` als tenant-gescopte EXISTS-Clause im task-Repo; zusätzlich CreateTask/ListTasks auf `middleware.GetTenantID(ctx)` (`772483fd`). Offen nur noch: FE-Wiring (Chip-UI/Filter).
- ✅ **Custom-Field-Definitionen (Task) — BACKEND ERLEDIGT 2026-06-11 (`2b8447b6`, Migrationen 000146+000147):** `work_custom_field_definitions` (tenant-scoped, RLS, 9 Feldtypen) + CRUD `/api/v1/work/custom-fields`, Seeds `work_custom_fields:*`. Follow-up FE-Adapter: `field_type`→`type`, `position`→`sortOrder`, Store kennt `dropdown` statt `select`.
- **🟡 Default-Status-Set:** `stores/workSettings.defaultStatuses` mock-first — das Status-Set, mit dem neue Projekte starten. Aktuell seedet der MSW-Create-Handler ein festes Set. → tenant-Setting `default_project_statuses` + Anwendung in `createProject`.
- **🟡 Projekt-Vorlagen löschen:** Liste (`templates_only`) + Umbenennen (PUT name) + „aus Vorlage erstellen" (`from-template`) laufen echt. **Löschen fehlt** (kein `DELETE /projects/{id}`-Endpoint im Client/Spec). → Delete-Project-Endpoint oder Template-Archivierung.
- **🟡 Zeit-Regeln (billable-Default, Stundensatz):** `stores/workSettings.billableByDefault`/`defaultHourlyRate` mock-first. → tenant-Setting; Anwendung beim Anlegen von Time-Entries + Stunden→Rechnung (P4).
- **🟢 P5 (Kalender-Sicht):** KEIN neuer Backend-Bedarf. `WorkCalendarView` bucketet `useTasks({project_id})` nach `due_date`; Drag = Fälligkeit ändern via bestehendes `PUT /tasks/{id}` (`due_date`). Nur ein latentes Komfort-Feld offen: ein `due_date`-Bereichsfilter im `listTasks`-Query (`due_from`/`due_to`) würde bei sehr vielen Tasks das clientseitige Bucketing entlasten — heute irrelevant (page_size 500).

## 🔴 helpdesk — CSAT / Kundenzufriedenheit (neu 2026-07-26, TERMINIERT vor Team-Review)

> **Darien-Vorgabe 2026-07-26:** CSAT muss FERTIG sein, bevor das Team (Darien+Luke+Nico) die grosse Review-Runde fährt — nur der Onboarding-Wizard darf danach kommen. Aktuell zeigt der Statistik-Tab „Kundenzufriedenheit" nur einen Mock-Wert (`mocks/handlers/helpdesk.ts` stats-Handler, hartcodiert `4.6/5`); es gibt keine echte Datenquelle. Kachel + Verteilungs-Chart sind im FE **gesperrt ausgeblendet** (`CSAT_FEATURE_ENABLED=false` in `HelpdeskPage.tsx`; im Editor-Statistik-Katalog `locked` in `editorModules.ts` helpdesk.statWidgets). Flag flippen, sobald das BE liefert.
> Markt-Recherche (Zendesk/Freshdesk/Zoho, `.planning/customization-block/STATISTIK-RECHERCHE.md`): überall derselbe Mechanismus — Ticket schliesst → Umfrage-Mail mit 1-Klick-Bewertungslink → aggregiert zu %. ~3–5 Tage BE.

- 🔴 **CSAT-Erhebung:** Ticket-Close-Trigger → Survey-Job (konfigurierbarer Delay) → E-Mail mit tokenisiertem 1-Klick-Bewertungslink (Skala 1–5, intern auf gut/schlecht mapbar) → Response-Endpoint (kein Login) → Speicherung am Ticket (`ticket_csat_responses`: ticket_id, rating, comment?, submitted_at, token; tenant-scoped + RLS).
- 🔴 **Aggregation:** `GET /helpdesk/stats` um echte CSAT-Werte erweitern (Durchschnitt + Sterne-Verteilung + Antwortzahl), tenant-gescopt. Ersetzt den Mock `customer_satisfaction`.
- 🟠 **CSAT-Konfiguration** (tenant-Setting): an/aus, Delay, Mail-Text/Skala. Gehört in die Helpdesk-Moduleinstellungen.
- ⚪ FE nach BE: `CSAT_FEATURE_ENABLED` auf `true`, `locked` an den 2 Statistik-Widgets entfernen, Kachel/Chart konsumieren echte Daten.
- ⚪ **FE-Fundament schon da (Session #31, P0):** CSAT liegt jetzt am Wire-Ticket (`csat_rating`/`csat_comment`) statt im Legacy-Zustand-Store; Mock-Endpoint `POST /helpdesk/tickets/:id/csat` + Hook `useSaveCsat`; 3 Seed-Tickets tragen echte Ratings. BE muss diesen Endpoint + die Persistenz real machen (siehe CSAT-Erhebung oben).

## 🔴 helpdesk — Ticket-Intake Daten-Fundament (neu Session #31, 2026-07-27, FE-mock-first gebaut)

> Kontext: Ticket-Intake-Block (Briefing `.planning/helpdesk-intake-block/KONZEPT-BRIEFING.md`). FE ist mock-first gebaut (P0 Wire-Modell + P1a Agent-Kanal); BE muss die erweiterten Felder real persistieren. Kein Blocker für die weiteren FE-Phasen ausser der öffentlichen Route (Kanal 1/extern).

- 🔴 **`CreateTicketInput` erweitern** (FE sendet bereits): `description`, `category`, `channel` (`agent|selfservice|external`), `requester_name`, `requester_email`, `requester_is_external`, `custom_fields` (Map). MSW nimmt sie heute an — BE-Handler + Proto/pb + Service müssen nachziehen.
- 🔴 **`Ticket.custom_fields`** aufs Wire + persistieren (`helpdesk_ticket`-Custom-Field-Werte pro Ticket; bislang FE-Session-Overlay, P1b zieht sie auf den Wire). Tenant-scoped + RLS.
- 🔴 **Requester-Modell:** Externe (Nicht-User) als Requester speicherbar (Name/E-Mail statt User-UUID), `requester_is_external`-Flag, `scope=own`-Filter-Verhalten für externe Requester definieren (heute matcht `scope=own` Externe nie).
- 🔴 **Herkunfts-Feld `channel`/`source`** am Ticket persistieren — treibt die geplanten Herkunfts-Reiter im Modul (agent / selfservice / external, mit „zusammenführen").
- 🟠 **Formular→Ticket-Aktion** (P2, FE gebaut Session #31, 2026-07-27): shared Engine `components/shared/intake` (modul-agnostisch, Registry + `mapSubmissionToRecord` + `useIntakeSubmit`), Helpdesk = erste Instanz (`modules/helpdesk/intake/helpdesk-ticket-target.ts` → `CreateTicketInput`). **Wire-Modell-Entscheid (Abweichung vom Briefing):** statt ephemerer `FormAction 'helpdesk_ticket'` = **persistiertes `FormSchema.intakeTargetId`** (via MSW create/update/duplicate; besser für die späteren Kanäle P4/P5, die das Ziel server-seitig kennen müssen). Feld-Rollen liegen als `FormField.role` (subject/description/priority/category/requester_name/requester_email); unmarkiert → `custom_fields`. **BE-Bedarf:** (1) `FormSchema.intake_target_id` + `FormField.role` persistieren; (2) Einreichung mit gebundenem Ziel erzeugt server-seitig das Ticket (Webhook-Basis vorhanden); (3) **Refinement:** freie Felder landen aktuell in `custom_fields` **keyed auf Slug(label)** — matcht NICHT die definierten `helpdesk_ticket`-Feld-Keys (sla_tier…), darum im Ticket-Detail-Zusatzfelder-Block (nur definierte Felder) unsichtbar; entweder Extras-Key auf ein definiertes Feld mappbar machen (Editor-UI) ODER Detail zeigt auch undefinierte `custom_fields` read-only.
- 🔴 **Öffentliche Formular-Route** (`/r/:token`, kein Login) — Haupt-Blocker für Kanal 1 (extern); + Anti-Spam/Rate-Limit + echtes Datei-Storage.

# Hinweis zur Arbeitsweise (Claude = FE, Luke = BE)
Die meisten 🟡-Punkte (FE-Page von Mock-Store auf fertige TanStack-Hooks umstellen) brauchen KEIN Backend von Luke — die Hooks + Endpoints existieren bereits. Luke-Bedarf = nur die 🔴- und cross-cutting-Punkte oben.
