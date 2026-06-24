# Sub-Terminal — admin: Benutzer / Rollen / Lizenz / Branding (A-0 … A-5)

> **Du bist das SUB-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174, Demo-Mode (MSW, KEIN Backend nötig).**
> Du baust **nur das `admin`-Modul** — und davon **nur die neuen Bereiche Benutzerverwaltung, Rollen/RBAC, Lizenz/Modul-Aktivierung, Branding**. Die echte Backend-Anbindung ist Lukes Track (🔒); du baust mock-first + swap-ready.
> **WICHTIG — Docker:** Du fährst **niemals** Docker hoch. Das Main-Terminal hält den Docker-Stack (Echt-Schaltung documents) — zwei Stacks = WSL2-OOM (16-GB-Maschine). admin läuft komplett auf MSW.
> **Ablauf:** erst **A-0** (Ist-Research + Marktrecherche + Klärungsfragen an Darien) — **STOPP am Gate, nicht weiterbauen, bis Darien antwortet.** Danach A-1…A-5 autonom, Meldung nach jeder Phase (`A-x fertig, n/5`).

---

## Warum admin jetzt (Kontext)
admin ist **Welle 2 — Lücken-Module** (`MASTER-PLAN.md` §6, §2 „admin"). Stand: `AdminHubPage` hat 4 Tabs (IT | Sicherheit | Abrechnung | Integrationen), aber **keine Benutzerverwaltung, kein Rollenmodell/RBAC-Matrix, keine tenant-weite Lizenz-/Modul-Aktivierung, kein persistentes Branding**. Das sind die P1–P4 aus dem Plan. Ziel dieses Batches: **diese vier Bereiche FE-mock-first review-reif** bauen, im Cosmi-Stil (Premium/Editorial, keine generische Admin-Tabelle), eingehängt als neue Tabs in den AdminHub.

## Ist-Stand (Main hat grob vor-recherchiert — DU verifizierst tief in A-0)
- **AdminHub:** `desktop/src/renderer/src/modules/admin/AdminHubPage.tsx` — 4 Tabs via `ROUTE_TO_TAB`/`TAB_TO_ROUTE` (`/admin/it|security|billing|integrations`), lazy-geladen, Gating `userHasRole(['admin','it_support'])`. **Hier dockst du neue Tabs an** (Benutzer, Rollen; Lizenz ggf. in den Billing-Tab integrieren).
- **Vorhandene Tabs (NICHT umbauen, nur als Muster lesen):** `tabs/ITAdminHubTab.tsx` (138 Z.), `tabs/SecurityAdminHubTab.tsx` (= gerade frisch gebautes security-Modul, **TABU**), `tabs/BillingAdminHubTab.tsx` (13 Z., wrappt `settings/tabs/BillingSettingsTab` — **settings/ ist TABU, nur als read-only-Import behalten**), `tabs/IntegrationsAdminHubTab.tsx`.
- **Infrastruktur:** `InfrastrukturPage.tsx` (929 Z.) existiert = Ressourcen-Monitoring (P5) ist schon da → **nicht nochmal bauen**, höchstens in A-5 kurz auf Konsistenz prüfen.
- **Rollen-Quelle:** `@/config/roles` (`userHasRole`) — lies das, daran orientiert sich die RBAC-Matrix (welche Rollen existieren: admin/it_support/manager/member/…).
- **MSW:** prüfen, ob `mocks/handlers/admin.ts` existiert (sonst neu anlegen) — stateful Endpoints für Users/Rollen/Lizenz.
- **CURRENT_USER / Seed:** Demo-User-Quelle ist `shared-ids.ts` (`CURRENT_USER`) — Mocks **nie** Namen hardcoden ([[reference_current_user_source]]).

## ⚠ Lane-Trennung & Kollisions-Regeln (verbindlich)
- **NUR anfassen:** `modules/admin/*` (neue Dateien + `AdminHubPage.tsx`-Tabs), `mocks/handlers/admin.ts`, admin-i18n-Namespaces, **ein** Eintrag in `module-settings-registry.tsx`.
- **TABU:** `modules/admin/tabs/SecurityAdminHubTab.tsx` + alles unter `modules/security/*` (frisch gemergt) · `modules/team/*` (HR-Personalverwaltung ≠ tenant-User-Accounts — siehe A-0-Frage 2) · `modules/settings/*` (Main-Lane cross-cutting) · `shared/*` umbauen (nur **konsumieren**: `DetailModal`, `SortMenu`, `ModuleSettingsShell`).
- **Hot-Files mit Main:** `i18n/messages/{de,en,fr,it}.json` + `mocks/handlers/index.ts` + `module-settings-registry.tsx`. Main fasst diese in seiner documents-Lane **kaum** an, aber halte deine Edits **additiv** (nur neue Keys/Zeilen anhängen, nichts Bestehendes umsortieren) → konfliktfreier Merge.

## Out of scope (🔒 Luke — NICHT bauen, nur mock-first + swap-ready)
- Echter Auth-Invite-Flow (E-Mail-Versand, Token), echte Rollen-/Permission-Persistenz im Gateway-RBAC, echte tenant-weite Modul-Lizenzierung/Billing-Service, echtes persistentes Branding (Logo-Upload→S3), Tenant-Provisioning (`POST /tenants`). → alles FE-mock-first, Backend-Bedarf in `.planning/backend-gaps.md` (Abschnitt „admin") notieren.

---

## A-0 — Gate: Ist-Research + Marktrecherche + Klärungsfragen  ⛔ STOPP nach diesem Schritt
**Technische Ist-Research (Pflicht, gründlich):**
1. AdminHub im echten UI öffnen (Demo-Mode, :5174, `/admin/it`) + alle 4 Tabs durchklicken + **Screenshots ansehen** → festhalten, was funktional vs. Stub ist (v.a. Billing-Tab).
2. `@/config/roles` lesen → welche Rollen existieren, wie wird gegated. `userHasRole`-Signatur.
3. `mocks/handlers/admin.ts` + `mocks/handlers/index.ts` lesen → was ist registriert, stateful?
4. Prüfen, ob `team`-Modul schon User-/Mitarbeiter-Listen hat (Überschneidungs-Risiko klären, Frage 2).
5. i18n: existiert ein `admin.*`-Namespace? Welche Keys fehlen für die neuen Bereiche?

**Marktrecherche (Pflicht — WebSearch, dann mit Cosmi-Ist abgleichen). ZWEI gleichwertige Achsen: Funktion UND Gestaltung.**
Referenzen (Admin-Konsolen B2B-SaaS): **Linear** (Members/Roles), **Notion** (Members & Groups), **Slack** (Workspace-Admin), **GitLab** (Admin Area), **Microsoft 365 Admin Center**, **Auth0/Okta** (User-Management + RBAC), **Vanta** (Access). Pro Bereich **beide** Achsen recherchieren:

- **Achse A — Funktion (was kann/zeigt es):** Was gehört in eine **Benutzerliste** (Spalten, Status, Bulk-Aktionen, Einladen-Flow)? Wie modelliert man eine **RBAC-/Permission-Matrix** (Rollen × Module/Aktionen, vererbte vs. eigene Rechte, custom Roles)? Wie zeigt man **Lizenz/Seats/Modul-Aktivierung** tenant-weit (gebucht, Sitzplätze belegt/frei)? Wie macht man **Branding** (Logo/Farben/Name) editierbar?
- **Achse B — Gestaltung/Design (wie sieht es aus, wie fühlt es sich an):** **Echte Screenshots/Layouts ansehen** (WebSearch Images / Doku-Seiten) und festhalten: Layout-Struktur (Liste vs. Karten, Tabellen-Dichte, Spalten-Priorisierung), visuelle Hierarchie (was ist primär, wie werden Rollen/Status visuell codiert — Badges/Farben/Icons), wie eine **Permission-Matrix visuell lesbar** bleibt (Sticky-Header/-Spalte, Zebra, Toggle-Darstellung), Empty-States, Einladen-/Detail-Interaktionsmuster (Modal vs. Inline vs. Slide-over). → **Diese Muster in Cosmi-Sprache übersetzen, NICHT kopieren** ([[feedback_design_philosophy]]): Premium/Editorial, Apple-Linse (Reduktion/Hierarchie, weil Admin Daily-Use-streng ist), Cosmi-Token (keine fremden Farben/Fonts), Detail = `shared/DetailModal`, ganze Zeile klickbar, sticky Close. Keine generische Bootstrap-Admin-Tabelle, keine Emojis.
- **Design-Skills nutzen** ([[feedback_skill_orchestration]]): `frontend-design` ist auto-geladen; für Layout/Hierarchie-Entscheide `arrange`/`critique`, beim Bau-Feinschliff `polish`. Proaktiv einsetzen, nicht auf Aufforderung warten.
- **Ergebnis je Bereich:** 1 knapper Abschnitt „Markt vs. Cosmi-Ist" mit **getrennten Notizen zu Funktion und Gestaltung** + 1–2 Satz Cosmi-Übersetzung (welches Muster, wie umgesetzt) — in `qa-admin.md`. `/intel-recall admin` laufen lassen, falls Keepers existieren.

**Dann: gebündelte Klärungsfragen an Darien** (eine Nachricht, nummeriert). Erwartbare Fragen:
1. Scope-Tiefe: alle 4 Bereiche (Benutzer/Rollen/Lizenz/Branding) gleich tief, oder Benutzer+Rollen priorisieren und Lizenz/Branding leichter?
2. **Abgrenzung team ↔ admin:** team verwaltet HR-Mitarbeiter (Personalakte). admin verwaltet tenant-**Login-Accounts** + Rollen. Eine gemeinsame Personen-Quelle (CURRENT_USER/Seed) oder bewusst getrennt? (Verhindert Doppelpflege.)
3. RBAC-Matrix: editierbar (Rollen/Rechte umschaltbar, mock-persist) oder erst read-only Übersicht? Custom-Roles erlauben?
4. Lizenz/Modul-Aktivierung: schaltet das tenant-weit Module sichtbar/unsichtbar (Demo), oder reine Anzeige „gebucht/nicht gebucht"?
5. Einladen-Flow: bis wohin mock-first (Formular + Pending-State + „Einladung erneut senden") — reicht das?

**STOPP. Erst nach Dariens Antworten A-1 starten.**

---

## A-1 — Benutzerverwaltung (mock-first, stateful)  `[CORE]`
**Soll:** Neuer Tab **„Benutzer"** im AdminHub (Route `/admin/users`). Benutzerliste: Avatar/Name/E-Mail/Rolle/Status (aktiv/inaktiv/eingeladen)/letzter Login. `shared/SortMenu` (Feld + Richtung), Suche, Status-Filter. **Ganze Zeile klickbar** → `shared/DetailModal` mit Profil + Rolle ändern + Aktivieren/Deaktivieren + „Einladung erneut senden". **„Benutzer einladen"**-Flow (E-Mail + Rolle wählen → Pending-Eintrag). Alles stateful MSW (überlebt Navigation).
**Verify:** Liste sortier-/filterbar; Einladen erzeugt Pending-User sichtbar; Rolle ändern + Deaktivieren wirken + spiegeln in der Liste; Detail-Modal vollständig + sticky Close; leere/Empty-States sauber.

## A-2 — Rollen & RBAC-Matrix  `[CORE]`
**Soll:** Neuer Tab **„Rollen"** (Route `/admin/roles`). Rollen-Liste (admin/it_support/manager/member + ggf. custom) mit Beschreibung + Nutzer-Anzahl. **Permission-Matrix:** Rollen × Module (oder Modul-Gruppen) × Aktionen (Sehen/Bearbeiten/Verwalten) als übersichtliches Grid. Modul-Leiter-Konzept abbilden falls aus A-0 bestätigt. Editier-Modus mock-persist (oder read-only je A-0-Antwort 3).
**Verify:** Matrix lesbar bei vielen Modulen (Scroll/Sticky-Header); Toggle wirkt + persistiert (mock); Rollen-Detail zeigt zugeordnete Nutzer; keine toten Schalter.

## A-3 — Lizenz / Modul-Aktivierung (tenant-weit)  `[CORE]`
**Soll:** Lizenz-Übersicht (aktueller Plan, Seats belegt/frei, Verlängerungsdatum) + **Modul-Aktivierungs-Liste:** welche Cosmi-Module sind tenant-weit gebucht/aktiv. Toggle (Demo) je A-0-Antwort 4. Integration: entweder eigener Tab „Lizenz" **oder** in den bestehenden Billing-Tab einbetten (nicht duplizieren — `BillingSettingsTab` ist read-only-Import, NICHT editieren; eigene Komponente daneben). Mock-first.
**Verify:** Seats-Zählung stimmig mit A-1-Userliste; Modul-Toggle wirkt sichtbar (Demo-Banner „nicht enforced" wo sinnvoll); kein Bruch mit dem bestehenden Billing-Inhalt.

## A-4 — Branding (tenant-weit, mock-persist)
**Soll:** Branding-Bereich (eigener Tab oder Unterabschnitt): Firmenname, Logo (Upload mock → Vorschau), Akzentfarbe (innerhalb der Cosmi-Token-Palette, **kein** Theme-Bruch), optional Subdomain-Anzeige. Live-Vorschau. Mock-persist (überlebt Reload via MSW-Seed). Echter Logo-Upload = 🔒 Luke (S3).
**Verify:** Eingaben persistieren (mock); Vorschau aktualisiert; Farbwahl bleibt im erlaubten Token-Rahmen; Demo-Hinweis bei Upload.

## A-5 — Demo-Tiefe-Schlusscheck + Modul-Settings + Screenshot-QA  `[GATE]`
**Soll:**
- Modul-Settings-Eintrag `id: 'admin'` in `module-settings-registry.tsx` (`ModuleSettingsShell`, personal + tenant sinnvoll). **Additiv** anhängen (Merge-sicher).
- Sweep über alle neuen Bereiche: keine toten Buttons/Toast-only-Stubs; Sortierung via `shared/SortMenu` wo Listen; ganze Zeile klickbar wo Detail; sticky Back/Close; Skeleton-Loading statt Spinner; Empty-States.
- Screenshot-QA **alle neuen Tabs** @1440 + @1024, **DE + EN** → 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors. FR+IT key-vollständig.
**Verify:** Alles grün, **Bilder wirklich angesehen**, in `qa-admin.md` dokumentiert.

---

## Branch-Setup (einmalig, ZUERST)
Bau **NICHT** direct-to-main. Im Review-Klon: `git checkout main && git pull`, dann `git checkout -b parallel/admin`. Alle A-Punkte committest + pushst du auf **diesen** Branch (`git push -u origin parallel/admin`). Das Main-Terminal merged `parallel/admin` am Ende kontrolliert (i18n + registry + handlers/index: additive Blöcke behalten, danach `npm run build`).

## Workflow pro Phase (Build-+-Verify-Standard, CLAUDE.md)
bauen → i18n ×4 (`{var}` single-brace, ICU-Plural `{count, plural, …}`, **nie** `{{var}}`/`_one`) → MSW/Demo-Daten stateful → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error /tmp/build.log`, **NIE `| tail` als Gate**) → eslint auf geänderte Dateien (`node_modules/.bin/eslint <files> --quiet`) → Playwright-Screenshot-QA gegen **:5174** + **Bilder WIRKLICH ansehen** → iterieren bis grün → ein Commit + Push auf `parallel/admin` → Eintrag in `qa-admin.md`.

## Dev-Server (Sub, Demo-Mode, kein Backend)
```bash
cd "C:/Users/darie/Documents/KMU-Hub-review" && git checkout main && git pull && git checkout -b parallel/admin
cd desktop && npm install   # falls node_modules nicht aktuell
npm run dev -- --port 5174  # Demo-Mode (MSW), AdminHub unter /admin/it
# Login Demo-Mode: Auto-Login (Stefan Vogel) — admin-Rolle nötig fürs Gating, prüfen
```
Dev-Server killen (Windows): PowerShell `Get-NetTCPConnection -LocalPort 5174 | Select -ExpandProperty OwningProcess | ForEach-Object { Stop-Process -Id $_ -Force }`. Nur 1 Dev-Server pro QA-Runde ([[feedback_kill_dev_server_windows]]).

## Definition of Done (admin-Lücken review-reif)
Alle A-1…A-5 verifiziert (Screenshots angesehen), 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors über alle neuen Tabs DE+EN, Benutzer-/Rollen-/Lizenz-/Branding-Flows mock-first durchklickbar + stateful, jede Phase ein Commit+Push auf `parallel/admin`, `qa-admin.md` gepflegt, Backend-Bedarf in `backend-gaps.md` (Abschnitt „admin"). Dann Darien: „admin 5/5 fertig".
