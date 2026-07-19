# R-3 Enforcement-Sweep — Terminal-Briefing (erstellt Session #15, 2026-07-18)

> **⚡⚡⚡⚡ UPDATE Session #19 (2026-07-19): BATCH 3 GEBAUT (inventar · einkauf · produktion · vertraege · helpdesk) — Details/QA/Funde im `RESUME-NEXT.md`-Top-Block #19. Nächstes Bau-Terminal: BATCH 4 (Arbeitspaket direkt hierunter). Konvention aus #16 gilt weiter, kein neues Gate.**

## Batch-4-Arbeitspaket (für das nächste Bau-Terminal, Stand nach #19)

**Scope: schichten · fuhrpark · vermietung · rapporte · dialer.** Danach Batch 5 = berichte · formulare · automatisierung + Standard-Rest (kommunikation/kalender/zeiterfassung/video/infrastructure/notifications — teils reicht Ebene 1 bzw. Mini-Katalog).

**Gleiche Reihenfolge wie Batch 3 (Katalog dort LEER):** ① UI-Inventar via Explore-Agents → ② Katalog kuratieren (`config/capability-catalog.ts`) + ③ ROLE_DEFS-Grants nachziehen (admin ALLE! manager voll operativ, member Basics, readonly reads, extern nichts) + ④ i18n subject/action ×4 (ordnungserhaltendes Insert-Skript, Muster `scripts/i18n-rbac-r3b3.mjs`) → ⑤ gaten (Referenz-Implementierung: **`modules/inventar/InventarPage.tsx`** — Hooks-Block vor early returns, `TAB_CAPABILITY`+`visibleTabs`+Fallback-Effect, moduleEmpty-Early-Return mit PageHeader, RestrictedModeBadge im actions-Slot, konditionale ItemActions-Arrays; disabled+Tooltip-Ausnahme-Muster: `EinkaufDetailModals.tsx` OrderDetailModal „Senden").

**⚠ BE-Seed-Abgleich Batch 4 (Subjekt-Namen übernehmen, nicht neu erfinden):** schichten 000095 + 000161 (`schichten:swap_request`!) · fuhrpark 000097 + 000196 (extended) · vermietung 000099 · rapporte 000093 + **000100 `rapporte:approve` existiert BE-seitig schon als feines Paar!** + 000164 · dialer 000068. In `backend/migrations/` nachsehen, gleiche resource-Namen verwenden.

**Bekannte modul-spezifische Startpunkte:** rapporte: approve (BE-Key da!) vs. erstellen/eigene · schichten: Tausch-Anfragen (swap_request) vs. Plan verwalten · dialer: Kampagnen verwalten vs. selbst wählen, Outcomes erfassen · fuhrpark: Fahrzeuge verwalten vs. Fahrten buchen · vermietung: Buchungen vs. Katalog ansehen. Owner-Felder je Modul prüfen (scope-own-Kandidaten: rapporte-Ersteller, dialer-Agent, schichten-Mitarbeiter) — wo nur Freitext/null: NICHT stillschweigend all, als Gap notieren (Muster Batch 3).

**QA:** Muster `scripts/qa-rbac-enforcement-b3.mjs` (admin-Regression je Modul + readonly-Chip + member-Rolle + extern-NoAccess + Bilder ansehen; Switcher NUR von `/#/settings`; Labels exakt aus de.json ziehen — „Statistik" ≠ „Statistiken").

## Batch-3-Arbeitspaket — ✅ GEBAUT in Session #19 (Historie)

**Scope: inventar · einkauf · produktion · vertraege · helpdesk** (Material-/Betriebs-Kette + die zwei Genehmigungs-/Vertrags-Module). Danach Batch 4 = schichten · fuhrpark · vermietung · rapporte · dialer, Batch 5 = berichte · formulare · automatisierung + Standard-Rest (kommunikation/kalender/zeiterfassung/video/infrastructure/notifications — teils reicht Ebene 1 bzw. Mini-Katalog).

**⚠ Der eine Unterschied zu Batch 1/2: `CAPABILITY_CATALOG[modul]` ist für ALLE Batch-3-Module LEER.** Pro Modul gilt daher zwingend die Reihenfolge:
1. **Katalog kuratieren:** Modul-UI sichten (Explore-Agent-Inventar wie gehabt), Basis-Aktionen (`base`, scopeable wo Owner-Feld existiert) + Fein-Schalter (`fine`) in `config/capability-catalog.ts` eintragen. Orientierung am Muster der Kern-Module (CRUD je Subjekt + export/import/settings/approve-artige fines).
2. **ROLE_DEFS-Grants nachziehen** (`mocks/data/rbac.ts`): admin bekommt ALLE neuen Keys (sonst verschwinden die Aktionen für den Chef — Default-Deny!), manager/member/readonly sinnvoll kuratieren (Muster: manager voll operativ, member own/team-basics, readonly nur reads, extern nichts).
3. **i18n `rbac.subject.<subject>` / `rbac.action.<action>` ×4** für jedes neue Segment (Drei-Stellen-Regel komplett: Katalog + Grants + i18n).
4. **BE-Seed-Abgleich VOR dem Erfinden:** `backend/migrations/*permission*` — produktion (`produktion:bom`×write) und rapporte (`rapporte:approve`) u.a. haben BE-seitig SCHON feine Paare → gleiche Key-Namen verwenden, nicht doppelt erfinden; Lücken in backend-gaps §RBAC nachtragen.
5. Dann gaten nach Konvention (§1-Arbeitsanweisung + R3-RECHERCHE §3) — Rezept, shared-Bausteine und QA-Muster wie Batch 1/2.

**Bekannte modul-spezifische Startpunkte:** einkauf: Genehmigen/Senden (approvalThreshold!), Warenkorb, Lieferanten-Deaktivieren, Exporte · produktion: Statuswechsel (start/complete/cancel), QS-Prüfungen, Maschinen, Laufkarte-PDF · inventar: Buchungen/Korrekturen vs. Ansehen · vertraege: anlegen/kündigen/Erinnerungen vs. Laufzeiten sehen · helpdesk: Tickets zuweisen/schließen vs. eigene Tickets (scope-own-Kandidat: assignee/requester).

**QA:** Muster `scripts/qa-rbac-enforcement-b2.mjs` + dessen Learnings (Switcher NUR von `/#/settings` aus öffnen — auf /admin/users sind User-Namen klickbare Zeilen; Panel-Toggle-Zustand prüfen statt blind klicken; Text-Checks präzise halten — Wörter wie „Buchhaltung" existieren auch als Abteilungsname → `getByRole('link'/'button')` statt Volltext). Mindestens: admin-Regression + readonly/extern-Preview je Modul + Bilder ansehen.

> **⚡⚡ UPDATE Session #17 (2026-07-19): BATCH 1 GEBAUT (work · documents · crm · finance · wiki) — Details/QA/Funde im `RESUME-NEXT.md`-Top-Block #17. Shared-Bausteine stehen (RestrictedModeBadge, NoAccessView+ModuleGate an ALLEN Routen, useScopedCapability, ItemActions-title). Darien-Entscheid Session-Ende #17: KEIN Review-Gate zwischen Batches — alle Reviews gesammelt am Ende (`hetzner-review-checklist.md`).**

## Batch-2-Arbeitspaket — ✅ GEBAUT in Session #18 (Historie)

**Scope: team-Aktions-Rest · dashboard Ebene-2 · settings/security-Rest.** Gleiches Rezept wie Batch 1 (Inventar via Explore-Agents → bauen nach §1-Arbeitsanweisung + R3-RECHERCHE §3 → Gates → QA). Bekannte Startpunkte:

- **dashboard Ebene-2 (der R-1-Merker):** Modul-Karten + Alert-Banner in `modules/dashboard/DashboardPage.tsx` müssen der Modul-Sichtbarkeit folgen (`useCapabilitySet` + `moduleViewKey` — Aushilfe sieht heute noch fremde Karten/Banner). Kein eigener Katalog nötig (Ebene 1 reicht); Widget→Modul-Zuordnung erheben.
- **team-Aktions-Rest:** Tabs + Rollen-Sektion sind seit R-1/R-2 gated — es fehlen die AKTIONEN: Mitarbeiter anlegen/einladen (`team:employee:create` — CreateEmployeeWizard/InviteMemberDialog), bearbeiten/deaktivieren (`team:employee:edit`(scope!)/`:deactivate`), Abwesenheit genehmigen/ablehnen (`team:absence:approve`), Korrekturen/Payroll/Trainings/Onboarding-Aktionen (je `team:*`-Katalog-Keys), PersonnelDocuments (`team:documents:view/edit`), Datenkategorie-Felder in MemberDetail (`team:data_personal/data_job/salary` — nur Basis-Gating; die TIEFE ist R-4!). Scope-own: `team:employee:read/edit` sind scopeable — Owner = eigener Datensatz (User-Id-Vergleich); manager hat edit `team` (≈all, dokumentieren).
- **settings/security-Rest:** ModuleSettingsShell-tenant ist seit #17 ZENTRAL gegated (`settings:tenant:manage`) — prüfen, was übrig bleibt: AdminHub-Sub-Tabs (`admin:license/branding/it/integrations/company/modules/ai:manage` je Tab), security-Hub `/admin/security` 10 Sub-Tabs (`security:audit:read` · `security:policy:manage` · `security:gdpr:execute` zuordnen), SettingsPage/Overlay-Restposten gegen `SETTINGS_TAB/ENTRY_CAPABILITY` abgleichen (R-1-Mappings in `config/capabilities.ts`).
- **Gates wie immer** + `tsconfig.rbaccheck.json` um Batch-2-Dateien erweitern; QA-Muster `scripts/qa-rbac-enforcement.mjs` (Editor-Preview + Switcher; Kaltstart: auf Buttons warten, nicht auf Timeouts).

> **⚡ UPDATE Session #16 (2026-07-19): Recherche-Gate Batch 1 DURCHLAUFEN + alle 4 Darien-Entscheide gefallen — Ergebnis + verbindliche Gating-Konvention in `R3-RECHERCHE.md`. Das Bau-Terminal kann Batch 1 DIREKT bauen (Schritte 1–3 unten sind erledigt; §2 nur noch Historie). Darien-Review R-1/R-2: ohne Feedback freigegeben („Kein Feedback, direkt R-3").**
> **Kern-Konvention (ersetzt §1.3-Konvention):** Default VERSTECKEN (auch Erstellen/Bearbeiten) · disabled+Tooltip NUR für Ausnahmen-Liste (crm:import:run, finance:invoice:send/quote:send, documents:file:download) · Read-only-Felder mit sichtbarem Wert als drittes Pattern · shared „Nur Ansicht"-Header-Chip · scope-own = Edit-Controls still weg · shared „Kein Zugriff"-Seite an Modul-Routen.
>
> **Für das frische Bau-Terminal.** Erst `git pull` (Stand ≥ `08ebbbd4`). Ablauf zwingend:
> **0) Falls Darien-Review-Feedback zu R-1/R-2 vorliegt: ZUERST als Phasen abarbeiten.**
> **1) Recherche-Gate → 2) gebündelte Fragen an Darien → 3) Darien-OK → 4) bauen → 5) Gates.**
> Kontext: `KONZEPT.md` (§3/§4) + `CAPABILITY-KATALOG.md` (§Arbeitsanweisung je R-3-Batch). R-1 (Fundament) + R-2 (Baukasten) sind komplett — NICHTS davon neu erfinden.

## 1. Scope R-3 (aus KONZEPT §4 — batch-weise à ~5 Module, bis 1.0 alle 32)

Aktions-Gating IN den Modulen: jeden Button/Dialog/Export an `useCapability` hängen. Ebene 1 (Nav/Sichtbarkeit) ist seit R-1 scharf — R-3 macht Ebene 2+3 scharf.

**Batch 1 (Kern, Katalog vor-kuratiert):** work · documents · crm · finance · wiki.
**Batch 2:** team (Aktions-Rest; Tabs+Rollen-Sektion sind schon gated) · **dashboard Ebene-2** (Modul-Karten + Alert-Banner folgen Sichtbarkeit — der bekannte R-1-Merker!) · settings/security-Rest.
**Batch 3+:** Branchen-Module (je Batch erst Katalog kuratieren — `CAPABILITY_CATALOG[modul]` ist dort noch leer; Ebene-2/3-Grants dann auch in `ROLE_DEFS`-Presets nachziehen, sonst hat niemand die neuen Keys!).

**Pro Modul (Arbeitsanweisung CAPABILITY-KATALOG §Ende):**
1. Modul-UI sichten: jeden sichtbaren Button/Dialog/Export/Menüpunkt listen → Capability-Key zuordnen (Katalog = SSOT `config/capability-catalog.ts`).
2. Fehlende Fein-Keys: erst Katalog + `ROLE_DEFS`-Grants ergänzen (+ i18n `rbac.subject/action`), DANN gaten.
3. Gating: **Konvention sicherheitsrelevant = VERSTECKT** (Export, Löschen, Admin-Aktionen), **workflow-relevant = disabled + Tooltip mit Begründung** (Erstellen/Bearbeiten). Leere Zustände sauber: Nur-Lesen-Rolle darf keine leeren Action-Bars/kaputten Empty-States sehen.
4. **Scope-Enforcement (own/team):** `useCapability(key).scope` liefert die Breite. `own` ⇒ Aktion nur auf eigenen Objekten (Vergleich gegen `CURRENT_USER.id` / assignee/owner-Feld), `team` pragmatisch wie `all` behandeln wo kein Team-Modell am Objekt hängt (dokumentieren!). Wo Mock-Objekte kein Owner-Feld tragen → Owner nachseeden oder als Gap notieren, NICHT stillschweigend `all`.
5. Screenshot-QA mit mind. 2 Rollen (Vollzugriff + Aushilfe/Nur-Lesen) — **Preview „Als Rolle anzeigen" aus dem Editor ist das schnellste QA-Werkzeug** (Banner oben, ohne User-Wechsel; ProfileSwitcher unten rechts für echte Session). Bilder ansehen.
6. BE-Seed-Abgleich: fehlende resource×action-Paare in backend-gaps §RBAC nachtragen (`backend/migrations/*seed*permissions*` als Ist).

**NICHT R-3:** HR-Datenkategorien-Tiefe (R-4) · Audit-Log-UI/Zentria-Zugang/Branchen-Sets (R-5) · Objekt-Grants (Phase 2).

## 2. Recherche-Gate R-3 Batch 1 (VOR den Fragen)

**Auftrag:** Wie gaten Marktführer GENAU DIESE Aktionen — beide Achsen (Verhalten + Optik):
- **work:** Asana/monday/ClickUp — Aufgabe erstellen/löschen ohne Recht: Button weg oder disabled? Kommentar-only-Rollen? Wie sieht ein Board für Viewer aus?
- **documents:** Google Drive/SharePoint/Dropbox — Download-Sperre (Viewer ohne Download!), Upload-Gating, Freigabe-Rechte-Stufen, wie kommuniziert die UI „nur ansehen"?
- **crm:** Zoho/Pipedrive/HubSpot — Export-Sperre (DSGVO), eigene-vs-fremde Kontakte bearbeiten (Scope-UI!), Import nur Admin?
- **finance:** lexoffice/sevdesk/Zoho Books — Beträge-verbergen-Muster (Assistenz-Rolle), Versenden ≠ Erstellen, Mahnwesen-Gating.
- **wiki:** Confluence/Notion — publish ≠ edit, fremde Artikel bearbeiten, Read-only-Space-Optik.
Fokusfragen: disabled+Tooltip vs. hidden je Aktionstyp · Empty/Reduced-States für rechtelose Rollen · wie zeigen sie „gehört nicht dir" (Scope) · Toast/Fehler wenn's doch einer versucht.
Ergebnis → Entscheidungsvorlage (z. B. „Standard-Konvention bestätigen oder je Aktionstyp differenzieren?") für die gebündelten Fragen.

**Erwartbare Darien-Fragen:** hidden-vs-disabled-Grenzfälle (z. B. „Neue Aufgabe" für Nur-Lesen: weg oder ausgegraut?) · Scope-own-Härte (fremde Aufgabe: Edit-Button weg oder Hinweis?) · Batch-1-Reihenfolge/Umfang OK?

## 3. Datei-Karte nach R-2 (Basis — nicht neu bauen)

| Bereich | Datei(en) |
|---|---|
| Katalog (SSOT) | `config/capability-catalog.ts` (CapabilityDef fine/scopeable, MODULE_CATEGORY) — Ebene-2/3-Keys je Modul; Rest-Module leer |
| Check-Hook | `hooks/useCapability.ts` (`useCapability` liefert allowed+scope+sources; `useCapabilitySet` für Listen; Preview-aware!) |
| Store | `stores/permissions.ts` (persist + Auth-Subscription + Client-Fallback + `preview`-Mode startPreview/endPreview) |
| Seeds/Presets | `mocks/data/rbac.ts` (ROLE_DEFS-Grants = wer darf was; CUSTOM_ROLES; USER_ROLE_ASSIGNMENTS mutierbar) |
| Handler | `mocks/handlers/rbac.ts` (me/permissions, admin/users/:id/permissions, Rollen-CRUD, Zuweisung, Fehler-Codes) |
| Baukasten-UI | `modules/admin/roles/` (RolesBuilderTab, RoleEditorPage, CloneRoleDialog, RoleCompareModal) — Referenz für Muster |
| Shared | `components/shared/rbac/` (UserRolesSection, EffectivePermissionsView) · `lib/rbac-format.ts` (capabilityLabel/moduleLabel/Fehler-Mapping) · `lib/rbac-diff.ts` |
| Preview-Banner | `components/layout/PermissionPreviewBanner.tsx` (global in App.tsx) |
| Ownership-Quelle | `mocks/data/shared-ids.ts` `CURRENT_USER` (nie Namen hardcoden); ⚠ mock-db↔IDS-Versatz nur in team.ts per NAME_TO_USER_ID gefixt — andere Module prüfen! |
| i18n | `rbac.*` ×4 komplett (subject/action/scope/module + builder/editor/compare/preview/assignment); Script-Muster `scripts/i18n-rbac-r2.mjs` |
| Gates | `tsconfig.rbaccheck.json` (erweitern um Batch-Dateien) · QA-Muster `scripts/qa-rbac-baukasten.mjs` (STUB/ONB/NOLAUNCH, waitForText, Switcher unten rechts, innerText ist UPPERCASE bei SectionTitles → /i matchen) |

## 4. Bekannte Stolpersteine (aus R-1/R-2 gelernt)

- **Neue Capability-Keys brauchen DREI Stellen:** Katalog + ROLE_DEFS-Grants (sonst hat sie niemand → Feature für alle weg!) + i18n subject/action ×4.
- **Default-Deny heißt:** vergisst du das Seeden eines Keys im admin-Preset, verschwindet die Aktion auch für den Chef. QA immer zuerst als Vollzugriff.
- **Loading = deny** (useCapability pessimistisch): Aktions-Buttons dürfen beim Kaltstart nicht flackern — bei Listen `useCapabilitySet().ready` beachten.
- **Preview gated auch das Verwaltungs-Modul selbst:** in der Aushilfen-Vorschau fliegt man aus /admin raus (gewollt) — Beenden über das Banner.
- **Dialer/produktion etc. haben BE-seitig schon feine Seeds** (`produktion:bom`×write, `rapporte:approve`) — beim Batch gegen `backend/migrations` abgleichen statt Keys doppelt zu erfinden.
- Bekannte offene Punkte: Seat-Meter „15 von 14" (Billing-Mock-Vorbestand) · Preview-Banner überlappt Topbar-Mitte leicht (kosmetisch) · CreateEmployeeWizard bietet nur Presets an.
- Standard-Gates: i18n ×4 (`{var}`, ICU-Plural) · gescopter tsc (nie Full-tsc) · `eslint src/ --quiet` vor Push · Screenshot-QA + **Bilder ansehen** · 1 Dev-Server (`npm run dev`, Port 5173; Kill nur per PowerShell Stop-Process) · 1 Commit + Push pro Batch (= Auto-Deploy, cd.yml scharf).

## 5. Nach R-3

R-4 HR-Seite (Datenkategorien × Zugriffsebene × Scope, Personio-Detail-Recherche) → R-5 Audit-Log-UI + Zentria-Setup-Zugang (GDAP-light) + 3 Branchen-Template-Sets (je Set mit Darien durchgehen). Onboarding O-0 erst nach RBAC-Review (Darien-Entscheid #13).
