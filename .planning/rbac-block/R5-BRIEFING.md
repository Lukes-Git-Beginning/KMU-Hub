# R-5 BRIEFING — Audit-Log-UI · View-as · Zentria-Setup-Zugang (GDAP-light) · 3 Branchen-Template-Sets

> Stand 2026-07-20 (Session #23). Recherche-Gate durchlaufen (Explore-Ist-Analyse + 2 Web-Agents: Entra/Google/Personio/Zoho/weclapp-Audit + GDAP/Salesforce/Zendesk/Atlassian + Handwerk/Dienstleister/Handel-Branchensoftware).
> Scope-Quelle: `KONZEPT.md` §Roadmap R-5 („Rechteänderungs-Log (UI) · View as ausgebaut · Zentria-Setup-Zugang (Ablauf + Sichtbarkeit, GDAP-light) · 3 Branchen-Template-Sets finalisieren, je Set mit Darien").
> Konventionen aus R-3/R-4 gelten unverändert (Default VERSTECKEN · Tab=Key · scope-own still · Drei-Stellen-Regel · i18n ×4 · gescopter tsc · Screenshot-QA mit Bilder-Ansehen).

## §0 Darien-Entscheide (Gate VOR dem Bauen)

**Runde 1 ENTSCHIEDEN (2026-07-20, alle 4 Empfehlungen bestätigt):**

1. **Platzierung Rechteänderungs-Log: Beide Sichten, eine Quelle.** Security-Audit-Log bekommt Detail-Panel mit Alt/Neu-Delta (Entra-Muster); ZUSÄTZLICH im Rollen-Modul ein Tab „Protokoll", der dieselben Daten vorgefiltert auf Rechteänderungen zeigt. Eine Datenquelle, kein Drift.
2. **View-as-Umfang: Echtes „Als Benutzer anzeigen".** Button im User-Detail (admin-only, neuer Katalog-Key `admin:user:view_as`), echte Session wechselt auf Ziel-User (scope-korrekt), persistenter Banner mit „Verlassen", Audit-Event `user.view_as`, Guardrail: nicht auf andere Admins anwendbar. Rollen-Overlay-Preview bleibt unverändert für Editor-Drafts.
3. **Zentria-Rollen-Umfang: Admin OHNE HR/Gehalt.** Voller Setup-Zugriff, aber keine Personalakten/Gehaltsdaten — need-to-know, konsistent mit it_admin-USP, Verkaufsargument.
4. **Sichtbarkeits-Indikator: Badge in der App-Shell.** Dezente Pill („Zentria-Zugriff aktiv · noch X Tage"), sichtbar für Admins, klickbar → Verwaltungsseite mit Entziehen-Button.

**Runde 2 OFFEN (Session #23 hier unterbrochen — Darien hatte Klärungsbedarf, VOR erneutem Stellen der Fragen erst klären was):**

5. GDAP-Laufzeiten: ___ (Vorschlag: feste Auswahl 3/7/14/30 Tage, Default 14, KEIN unbegrenzt, Verlängerung = neue Anfrage)
6. Branchen-Sets (je Set durchgehen, Vorschlags-Tabellen §5): Handwerk ___ (Vorschlag 4 Rollen inkl. Azubi; Subunternehmer bewusst kein Template) · Dienstleister ___ (Vorschlag 4 Rollen inkl. Freelancer) · Handel ___ (Vorschlag 4 Rollen inkl. Filialleiter mit dokumentiertem 1.0-Limit „kein Standort-Scope")

**Ohne Frage bereits gesetzt (Konsequenz aus R-4-Entscheiden / Markt, im Chat vermerkt):**
- **Alt/Neu-Delta im Audit-Detail-Panel = Pflicht in R-5** (Entra-Muster; Zoho/Google/Personio haben es NICHT → Differenzierung; im Mock billig: Snapshot bei Write).
- **Keine Feld-Ebene 1.0 bleibt bestehen** (R-4-Entscheid ④): EK-Preis-/Honorar-Trennung in den Branchen-Sets wird über Modul-/Aktions-Sichtbarkeit gelöst, NICHT über Feld-Maskierung. Wo das nicht reicht (z.B. Verkäufer + Artikel-Stammdaten mit EK-Feld) → Vermerk in backend-gaps + Set-Beschreibung.
- **Audit-Retention-Kommunikation:** Hinweiszeile auf der Log-Seite „Einträge werden 24 Monate aufbewahrt und können nicht verändert werden." (ein Satz, kein Tech-Detail; Hash-Chain existiert schon via 000222/000039).
- **Templates ≠ System-Rollen:** Branchen-Rollen werden als VORLAGEN instanziiert (Custom-Rolle beim Übernehmen), ROLE_DEFS/7 System-Presets bleiben unangetastet.

## §1 Scope

1. **Rechteänderungs-Audit (UI + Live-Events):** RBAC-Mutationen erzeugen Audit-Einträge (Mock-seitig stateful), sichtbar mit Alt/Neu-Delta. Event-Taxonomie §4.1.
2. **View-as-Ausbau:** über die Rollen-Overlay-Preview hinaus (Entscheid #2).
3. **Zentria-Setup-Zugang GDAP-light:** Beziehung mit Laufzeit/Status/Entzug + Kunden-Sichtbarkeit + Audit-Anbindung. Datenmodell §4.3.
4. **3 Branchen-Template-Sets** (Handwerk/Dienstleister/Handel) als Vorlagen-Galerie im Rollen-Anlege-Pfad. Sets §5.

NICHT in R-5: vollständiges Activity-Log aller Admin-Aktionen (Entra-Niveau, Roadmap) · echtes Session-Tracking des Zentria-Agents (nur Events, kein Live-Monitor) · Standort-/Filial-Scope (Architektur-Thema, backend-gaps).

## §2 Ist-Analyse (Kern, mit Pfaden)

### Audit
- `modules/security/AuditLogPage.tsx` — fertige Tabelle (Timestamp/User/Action/Target/IP/Result, PAGE_SIZE 25), Filter (Datum/Action-Suche/Result), CSV/JSON-Export (`useExportAuditLog`), Kettenprüfung (`useVerifyAuditChain`). Lazy im `SecurityAdminHubTab` (audit-Sub-Tab, `security:audit:read`).
- Typen `AuditEntry/AuditFilter/...` in `api/security-types.ts` (Z. 49–86) · Client `api/security-client.ts` · Hooks `api/hooks/useSecurity.ts`.
- MSW `mocks/handlers/security.ts`: GET /security/audit (Z. 206), Export (342), Verify (359). **50 statische Seeds** (`rawAuditLogs` Z. 13–64, ein hartkodiertes `user_role_changed`).
- Backend: `000039` audit_log mit Hash-Chain · `000222` append-only-Trigger (GoBD). **Kein Write-Interceptor** bei Rollen-Mutationen — weder MSW noch BE.
- **Lücke:** `writeAuditEvent()`-Mechanik + Detail-Panel mit old/new fehlen.

### View-as
- Rollen-Preview: `RoleEditorPage` Z. 296–305 → `usePermissionsStore.startPreview({label, capabilities})`, Banner `components/layout/PermissionPreviewBanner.tsx`, Checks in `hooks/useCapability.ts` Z. 29–35.
- **Schwächen:** Scopes flach als `all` injiziert (own/team nicht simuliert) · API-Calls laufen weiter als echter Admin (Session unverändert) · kein „als konkreter User".
- User-Ebene existiert nur dev-only: `ProfileSwitcher` (8 DEMO_PROFILES) via `setDemoSessionUserId()` (`mocks/data/rbac.ts` Z. 609–616) + `useAuthStore.setState` → permissions-refresh. **Diese Mechanik ist der saubere Unterbau für echtes View-as-User** (Session wechselt wirklich → Scopes/Daten automatisch korrekt).

### Rollen-Anlage / Presets
- Kein Wizard — bewusst: **Clone-only** (`modules/admin/roles/CloneRoleDialog.tsx`: basedOn/name/description/color). `ROLE_DEFS` (7 System-Presets) + `createCustomRole()` in `mocks/data/rbac.ts`.
- `BusinessProfileId` existiert (`business-profiles.ts`: handwerk, gastronomie, einzelhandel, dienstleistung, it_tech, produktion, logistik, gesundheit, bau) — deckt nur Modul-Sichtbarkeit, kein RBAC.
- **Einhängepunkt Templates:** neue Konstante `INDUSTRY_ROLE_TEMPLATES` + „Aus Vorlage"-Tab im CloneRoleDialog (Vorlage = Pre-Fill von basedOn + grants → wird Custom-Rolle).

### Zentria-Zugang
- **Null Bestand.** Kein valid_until an user_roles (000002 hat nur assigned_at), keine Impersonation, kein partner_access. `expires_at` nur an Auth-Standard-Flows (refresh_tokens/invitations/guest_sessions/password_reset).

### Infra
- MSW-stateful-Muster: `mocks/handlers/hr-change-requests.ts` (Map + Seeds + session-aware via `getDemoSessionUserId()`), Registrierung `mocks/handlers/index.ts`.
- i18n: `scripts/i18n-rbac-r4.mjs`-Muster (ADD-Objekt [de,en,fr,it], ordnungserhaltendes Insert). QA: `scripts/qa-rbac-enforcement-r4.mjs`-Muster (switchTo, rawKeys, pageerror-Listener, Screenshots). Typecheck: `desktop/tsconfig.rbaccheck.json` erweitern.

## §3 Markt-Kern

### Audit-UIs
- **Entra = Benchmark:** 4 Filter (Datum/Service/Kategorie/Aktivität), Side-Panel mit **Old Value / New Value** — das entscheidende Muster. Explizite Immutability-Kommunikation („system generated, can't be changed").
- Zoho: 3 Jahre Retention, Role/Profile/Group-Events, CSV — aber kein Delta. Google: konfigurierbare Spalten, kein Delta. Personio: Negativ-Referenz (lückenhaft, „Performed by" teils leer, kein Export). **weclapp: gar kein dokumentiertes Rechte-Audit → DACH-Marktlücke.**
- Standard-Spalten: Zeit · Akteur (Name+Rolle) · Event · Ziel · Status · Detail. Standard-Filter: Zeitraum · Akteur · Event-Typ · Ziel.

### GDAP / Anbieter-Zugang
- **GDAP-Mechanik:** Partner beantragt Beziehung (Name + Laufzeit + Rollen) → Kunde genehmigt per Link → beide können jederzeit kündigen → Auto-Expiry, optional Auto-Extend. Kunde sieht aktive/ausstehende Beziehungen in einer Liste. Seit 04/2026 eigene Kunden-Rolle für GDAP-Verwaltung.
- Salesforce „Grant Login Access": Laufzeit-Dropdown (1 Tag…1 Jahr). Zendesk „Assume": bis „Indefinitely" (Schwäche). **Atlassian: Zugang pro Ticket, Auto-Entzug bei Ticket-Close, jede Support-Aktion im Audit-Log filterbar → bestes Transparenz-Muster.**
- DSGVO-Erwartung: AVV (Art. 28) vorausgesetzt · Protokollierung ALLER Anbieter-Zugriffe = Compliance-Anforderung, nicht Feature · Need-to-know (kein Full-Admin wenn vermeidbar) · 7–14 Tage marktüblich fürs Onboarding.

### Branchen-Sets (Markt-Konsens der Rechte-Trennungen)
- **Handwerk** (ToolTime 3 Rollen, plancraft Büro/Mobil+PL, HERO 5 Rollen, Craftnote projektbasiert): Monteur = nur eigene Aufträge, **null Preise/Kalkulation/Angebote**; Büro = alles operative ohne Löhne; Subunternehmer = projektbasierte Einladung (kein eigener Preset im Markt).
- **Dienstleister** (MOCO Extern-Flag, awork Admin/User/Guest, Teamleader „beschränkter CRM-Zugang", Scoro Finanz-Permission-Sets): Freelancer = kein Kundenstamm/keine Stundensätze; Operative = keine Umsatz/Margen; Backoffice = Zeiten aller (Abrechnung) aber keine Pipeline/Margen.
- **Handel** (JTL Benutzergruppen Einkauf/Verkauf/Lager/Buchhaltung + EK-Preis-Einzelrecht, plentymarkets Rollen, weclapp Rollenbausteine): Verkäufer = null EK/Marge/Lieferanten; Lager = kein POS/keine Finanzen; Filialleiter = eigener Standort (Scope-Thema!).

## §4 Bau-Plan

### 4.1 Audit-Events + Rechteänderungs-UI (Arbeitspaket A)
- Neu `mocks/data/audit-events.ts`: stateful Event-Store (Map, Muster hr-change-requests) + `writeAuditEvent({action, actorId, target, targetType, oldValue?, newValue?})`; `getDemoSessionUserId()` als Actor. security.ts-GET merged Seeds + Live-Events (Sortierung nach ts, sequence_num fortlaufend).
- Taxonomie (action-Keys): `role.assigned` · `role.revoked` · `role.definition_created` · `role.definition_updated` · `role.definition_deleted` · `permission.override_set/removed` (R-6-Vorgriff, Keys reservieren) · `user.invited` · `user.deactivated` · `user.reactivated` · `user.offboarded` (R-4-Kaskade!) · `user.view_as` · `vendor_access.requested/granted/revoked/expired` · `setting.changed` (nur wo trivial andockbar).
- Interceptor-Stellen (MSW): Rollen-Zuweisung/-Entzug am User · Rollen-Editor „Übernehmen" (definition_updated mit grants-Diff als old/new) · Clone/Create/Delete Custom-Rolle · User-Lifecycle inkl. OffboardEmployeeDialog-Flow · GDAP-Endpoints (4.3).
- UI: Detail-Panel (Klick auf Zeile → Side-Panel/Modal, Muster Entra: Event/Akteur/Ziel/Alt/Neu/Zeit/IP) in AuditLogPage + Platzierung laut Entscheid #1. Retention-Hinweiszeile.

### 4.2 View-as (Arbeitspaket B — Umfang laut Entscheid #2)
- Falls „Als Benutzer anzeigen": Button in UserDetailModal (Katalog-Key neu `admin:user:view_as`, fine, admin-only per Default), Mechanik = `setDemoSessionUserId(target)` + authStore-Swap + Merken des Rückwegs; persistenter Banner (Muster PermissionPreviewBanner) „Du siehst Cosmi als {Name} — Verlassen"; Audit-Event `user.view_as`; **Guardrail: nicht auf User mit admin-Rolle anwendbar** (Privilege-Escalation-Optik vermeiden).
- Rollen-Overlay-Preview bleibt für Editor-Drafts (dokumentierte Grenze: flache Scopes) — Hinweis-Tooltip im Preview-Banner ergänzen („Scope-genaue Sicht: Als Benutzer anzeigen").

### 4.3 GDAP-light (Arbeitspaket C)
- Datenmodell (`api/vendor-access-types.ts` + `mocks/handlers/vendor-access.ts` stateful):
  `VendorAccessRelationship { id, name, requested_at, approved_at?, approved_by?, expires_at, status: 'pending'|'active'|'expired'|'revoked', revoked_at?, revoked_by?, roles: string[] }`.
- Endpoints (MSW): list · approve · revoke · (Anfrage kommt „von Zentria" = Seed pending + 1 active Demo). Jede Transition → writeAuditEvent.
- UI: Verwaltungs-Karte laut Platzierung (Vorschlag: Security-Hub neuer Sub-Tab „Anbieter-Zugriff", Key `security:vendor_access:manage` neu im Katalog, admin-only) — Liste der Beziehungen mit Status-Pill, Restlaufzeit, „Zugang entziehen" (Confirm-Dialog), Historie (revoked/expired). + Sichtbarkeits-Indikator laut Entscheid #4.
- Zentria-Rolle: als System-Konstante (`ZENTRIA_SETUP_ROLE`) mit Grant-Set laut Entscheid #3 — NICHT in ROLE_DEFS/Rollen-Liste des Tenants (kein zuweisbarer Preset), nur über vendor_access wirksam.

### 4.4 Branchen-Template-Sets (Arbeitspaket D)
- Neu in `mocks/data/rbac.ts` (oder eigene Datei `industry-role-templates.ts`): `INDUSTRY_ROLE_TEMPLATES: { setId: 'handwerk'|'dienstleister'|'handel', label, businessProfiles: BusinessProfileId[], roles: { id, label, description, basedOn: RoleId, grants: GrantSpec }[] }[]` — grants VOLLSTÄNDIG ausformuliert (kein Delta-Merge zur Laufzeit, Drift-Gefahr).
- UI: CloneRoleDialog → zweiter Einstieg „Aus Vorlage" (Tab oder vorgeschalteter Schritt): Set-Auswahl (Business-Profil des Tenants vorsortiert) → Rollen-Karte mit Beschreibung + Modul-Zusammenfassung → Übernehmen = createCustomRole mit Template-grants (Name editierbar). Templates erzeugen CUSTOM-Rollen (voll editierbar danach).
- Mapping businessProfiles: handwerk→[handwerk,bau] · dienstleister→[dienstleistung,it_tech] · handel→[einzelhandel,logistik]; alle Sets bleiben für alle sichtbar, nur Sortierung.

### Agent-Aufteilung (Muster R-4: Fundament selbst, dann 3 Sonnet-Agents)
- Selbst: audit-events-Store + writeAuditEvent + Interceptor in rbac-Mutations-Handlern + Katalog-Keys + ROLE_DEFS-Nachzug + i18n-Script.
- Agent A: AuditLogPage-Detail-Panel + Platzierungs-UI + Retention-Hinweis.
- Agent B: GDAP-light komplett (Typen, MSW, Security-Sub-Tab, Indikator).
- Agent C: Template-Galerie im CloneRoleDialog + INDUSTRY_ROLE_TEMPLATES-Feinausformulierung + View-as-User (falls entschieden; sonst zu B).

## §5 Branchen-Set-Vorschläge (je Set mit Darien durchgehen — Entscheid #6)

Konvention: „Voll" = operative Grants wie member/manager-Muster des Moduls aus R-3 · „—" = kein Grant (Modul unsichtbar). Feinausformulierung beim Bauen gegen `capability-catalog.ts`; kritische Trennungen sind verbindlich.

### Set 1: Handwerk (4 Rollen)
| Rolle | basedOn | Voll | Lesend/eng | Verborgen | Kritische Trennung |
|---|---|---|---|---|---|
| Büro & Auftragswesen | manager | crm, finance (operativ), kalender, dokumente, formulare, vertraege, einkauf, zeiterfassung(team:view) | berichte (ohne Finanz-KPIs schwer → reports:read) | HR-Gehalt, settings:tenant, infrastructure | Löhne/Personalakten zu |
| Bauleiter/Projektleiter | manager | work, rapporte (inkl. approve), zeiterfassung (team+approve), schichten (assignment), kalender, dokumente, formulare | crm (read), inventar (read) | **finance komplett**, einkauf, berichte, HR | Führung ohne Zahlen: keine Kalkulation/Preise |
| Monteur/Geselle | member | rapporte (create, own), zeiterfassung (own), schichten (own+swap), formulare, inventar (movement:create) | kalender (own), dokumente (read) | **crm, finance, einkauf, berichte, HR** | Null Preis-/Finanz-Sicht, nur eigene Aufträge |
| Azubi | member (eng) | zeiterfassung (own), formulare | rapporte (read), kalender (own), wiki (read = Lernmaterial), dokumente (read) | Rest | Wie Monteur minus inventar/schichten-swap |

### Set 2: Dienstleister (4 Rollen)
| Rolle | basedOn | Voll | Lesend/eng | Verborgen | Kritische Trennung |
|---|---|---|---|---|---|
| Projektleiter/Senior | manager | crm, work, kalender, zeiterfassung (team), dokumente, wiki, vertraege, formulare | finance (view ohne export), berichte (reports:read) | HR-Gehalt, settings:tenant | Kundenstamm voll, GuV/Margen-Exporte zu |
| Consultant/Bearbeiter | member | work (own/team), zeiterfassung (own), wiki (edit), dokumente, kalender (own) | crm (read) | **finance, berichte, HR, dialer** | Keine Stundensätze/Deal-Werte/Umsätze |
| Backoffice/Office | member+ | finance (expenses/invoices operativ), zeiterfassung (team:view+export), kalender, dokumente, vertraege, einkauf | crm (Kontakte read), berichte (reports:read) | **crm-Deals/Pipeline**, HR-Gehalt | Zeiten aller (Abrechnung) ja, Pipeline/Margen nein |
| Freelancer/Extern | extern | zeiterfassung (own), formulare | work (own, wie extern-Basis), dokumente (read), kalender (own) | **crm gesamt, finance, berichte, wiki, HR** | Kein Kundenstamm, keine Honorare |

### Set 3: Handel (4 Rollen)
| Rolle | basedOn | Voll | Lesend/eng | Verborgen | Kritische Trennung |
|---|---|---|---|---|---|
| Filialleiter | manager | crm, kalender, zeiterfassung (team), schichten, inventar, team (read/directory) | finance (view), berichte (reports:read) | **einkauf (EK!)**, HR-Gehalt, settings:tenant | 1.0 OHNE Standort-Scope (Vermerk backend-gaps) |
| Verkauf/Kasse | member (eng) | zeiterfassung (own), schichten (own+swap) | crm (Kontakte read/create), kalender (own), inventar (item:read = Bestandsauskunft; EK-Feld = Feld-Ebene-Vermerk) | **einkauf, finance, berichte, HR** | Null EK/Marge/Lieferanten |
| Lager/Logistik | member | inventar (voll: movement, inventur, wareneingang), zeiterfassung (own), schichten (own) | einkauf (po:receive + read für Wareneingang; Preise = Feld-Ebene-Vermerk) | **crm, finance, berichte, HR** | Kein Kassen-/Kundenkontext, keine Finanzen |
| Einkauf | member+ | einkauf (voll), inventar (voll inkl. item:edit), vertraege, dokumente | crm (read: Lieferanten), berichte (reports:read) | **finance (Kundenseite), HR**, dialer | EK ja, Kunden-Margen/VK-Kalkulation nein |

## §6 Gates + QA
- i18n `scripts/i18n-rbac-r5.mjs` ×4 (`rbac.audit.*`, `rbac.viewAs.*`, `rbac.vendorAccess.*`, `rbac.template.*` + Set-/Rollen-Labels; ICU einfache Klammern!).
- `tsconfig.rbaccheck.json` + neue Dateien · `eslint src/ --quiet` · Lint vor Push.
- QA `scripts/qa-rbac-enforcement-r5.mjs`: (1) admin: Rolle zuweisen → Audit-Eintrag mit Alt/Neu erscheint (2) Rollen-Editor speichern → definition_updated-Delta (3) Vendor-Access: active-Karte + Entzug-Flow + Audit (4) Indikator sichtbar/weg nach Entzug (5) Template-Galerie: Set öffnen, Rolle instanziieren, in Rollen-Liste als Custom (6) instanziierte Monteur-Rolle im Editor-Preview: finance/crm unsichtbar (7) View-as (falls gebaut): Banner + Verlassen + Audit-Event (8) readonly/extern: kein Zugriff auf neue Flächen. ALLE Bilder ansehen.

## §7 Luke-Paket (backend-gaps §RBAC R-5-Block, beim Abschluss eintragen)
- Audit-Write serverseitig an ALLE Rollen-/User-Mutations-Routen (route_auth.go hat heute keinen Audit-Call!) — Taxonomie §4.1 als Vorgabe, old/new-Snapshot als JSONB `details`.
- `user_roles` + valid_until/granted_by (GDAP + befristete Rollen P2) · vendor_access-Tabellen + Expiry-Job · Zentria-Rolle serverseitig NICHT als Tenant-Rolle zuweisbar.
- View-as serverseitig = Impersonation-Token mit Audit + Guardrail (nie auf admin) — FE-Mechanik ist Mock-only!
- Template-Instanziierung = normaler Custom-Role-Create (kein BE-Neubau), aber Template-Katalog als Seed-Vorlage übernehmen.
- Feld-Ebene-Vermerke: inventar item EK-Feld · einkauf po-Preise (Verkauf/Lager-Fälle §5).

## §8 Stolpersteine
- security.ts-Handler: Seeds + Live-Events mergen ohne sequence_num-Kollision (Live ab 51); Verify-Endpoint muss Live-Einträge mit-hashen oder Live-Einträge von der Kettenprüfung ausnehmen (Hinweis im UI: „Sitzungs-Einträge werden bei Neustart zusammengeführt" NICHT nötig — einfach: Verify läuft nur über Seeds, dokumentieren).
- ProfileSwitcher (dev) vs. View-as (Produkt): gleiche Mechanik, getrennte UI — View-as darf NICHT im DEV_BYPASS_AUTH-Gate hängen.
- Template-grants gegen ECHTE Katalog-Keys ausformulieren (Drei-Stellen-Regel: Katalog-Key existiert · ROLE_DEFS-frei · i18n-Label) — keine erfundenen Keys (R-4-Lektion NAME_TO_USER_ID).
- CloneRoleDialog: „Aus Vorlage" darf Clone-Flow nicht brechen (Clone bleibt Default-Pfad).
- Offboard-Flow (R-4) beim Audit-Interceptor mitnehmen — der Kaskaden-Dialog mutiert Rollen/Login/Seat.
