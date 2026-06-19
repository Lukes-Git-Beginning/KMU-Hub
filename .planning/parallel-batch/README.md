# Parallel-Batch — Koordination (Main + Sub) — berichte + notifications

> **Stand 2026-06-19 (Batch 4).** Zwei Terminals bauen parallel, damit sie sich **nicht in die Quere kommen**. Main = **berichte** (Reporting/BI, mittel — MSW-Ausbau + Drilldown + Schedules), Sub = **notifications** (Benachrichtigungen, klein — Schema-Fix + DetailModal + Sidebar-Badges). Vorheriger Batch (automatisierung + profil) ist durch + bei Nico → siehe `.planning/archiv/parallel-batch/batch3-automatisierung-profil/` und `.planning/reviews/`.

## Rollen & Klone

| | **Main-Terminal** | **Sub-Terminal** |
|---|---|---|
| Klon | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (frisch geklont, node_modules ready) |
| Dev-Port | **5173** | **5174** |
| Modul | **berichte** (B-1…B-5) | **notifications** (N-1…N-5) |
| Plan-Datei | `main-berichte.md` | `sub-notifications.md` |
| QA-Protokoll | `qa-berichte.md` (selbst anlegen) | `qa-notifications.md` (selbst anlegen) |
| Branch | **`main`** (Hauptklon) | **`parallel/notifications`** (Iso, NICHT direct-to-main) |
| Zusatzrolle | plant beide Batches, beantwortet Klärungen, merged am Ende `parallel/notifications`, stellt kombinierte QA-Liste + Nico-Review-Dateien | baut nur ab, meldet Darien Fortschritt (`N-x fertig, n/5`) |

## Lane-Trennung — was kollidiert NICHT
- **Modul-Code disjunkt:** `modules/berichte/` vs. `modules/notifications/`. Kein Overlap.
- **Stores disjunkt:** berichte ist MSW-/Query-getrieben (kein eigener großer Store) · notifications nutzt `stores/notifications.ts` (transienter Toast) + `mocks/handlers/notifications.ts` (Center/Bell-Quelle). Kein Overlap.
- **MSW-Handler:** beide Handler (`mocks/handlers/berichte.ts`, `mocks/handlers/notifications.ts`) sind **bereits in `index.ts` registriert** → **kein** `index.ts`-Touch von beiden Seiten → **null `mocks/index.ts`-Konflikt.** Jeder fasst nur seinen eigenen Handler an.
- **QA-Protokolle disjunkt:** jeder schreibt nur seine eigene `qa-*.md`.
- **shared/ einfrieren:** beide **konsumieren nur** `shared/DetailModal` + `shared/SortMenu` (existieren bereits). **Niemand** baut neue `shared/`-Komponenten. Falls doch nötig → stopp, mit Main absprechen.

## Lane-Trennung — die echten Reibungspunkte (Regeln verbindlich)

1. **i18n — garantierte Kollision (an Objektgrenze, schnell lösbar).** Beide hängen Keys an `i18n/messages/{de,en,fr,it}.json` an.
   - berichte-Keys leben unter `berichte.*` · notifications-Keys unter `notifications.*`. **Verschiedene Prefix-Cluster.**
   - **Regel:** Keys ins jeweilige Prefix-Cluster einsortieren (nicht ans Datei-Ende klatschen). `{var}` **single-brace** ×4 Sprachen, ICU-Plural `{count, plural, …}` — **nie** `{{var}}`, nie `_one`/`_other`.
   - Dank Branch-Isolation (Regel 3) **kein** Live-Konflikt — wird beim finalen Merge durchs Main-Terminal gelöst (beide Key-Blöcke behalten, danach `npm run build`).

2. **`module-settings-registry.tsx` — zweite (kleine) Kollision.** Beide fügen je **einen** Eintrag hinzu (B-4 berichte, N-4 notifications).
   - Andere `id` + andere Stelle im Array → Git-Konflikt nur an der Array-Grenze, beim finalen Merge **beide Einträge behalten**.
   - **Sub trägt den Eintrag auf `parallel/notifications` ein**, Main auf `main`. Kein Live-Konflikt.

3. **Branch-Isolation (Sicherung gegen main-Konflikte):** Das **Sub-Terminal baut auf `parallel/notifications`** (nicht direct-to-main), das **Main-Terminal auf `main`**. So gibt es während des Laufs **null Live-Konflikt** auf `main`. Das Main-Terminal merged `parallel/notifications` am Ende **einmal kontrolliert** (i18n + registry: beide Blöcke behalten, danach `npm run build`).

4. **Sidebar/Routing:** Nur das **Sub** fasst die Sidebar an (N-4 = Modul-Unread-Badges in der Sidebar). berichte fasst Sidebar/Routing **nicht** an → kein Konflikt. Sub: minimal-invasiv am bestehenden Sidebar-/Nav-Render, kein Umbau.

## Entscheidungen (Darien delegiert: „such du aus" — hier festgelegt, keine Rückfragen mehr nötig)
- **berichte-Scope = „MSW-Ausbau + Demo-Tiefe + FE-Parität":** Die 3 toten Tabs (Erstellen/Geplant/DATEV) leben machen via MSW-Handler, Drilldown auf `shared/DetailModal`, Schedules stateful + Alerts-UI, SortMenu + Settings-Eintrag, i18n-Bereinigung. **KEIN** echter No-Code-Query-Builder (P2 🔒 Luke), **kein** echtes DATEV-Backend (P4 🔒). Ziel: review-reif als FE-mock-first.
- **notifications-Scope = „Schema-Fix + Demo-Tiefe + UX-Parität":** Seed-Schema reparieren (Unread/Priorität/Modul/Deep-Link leben), Modul-Filter + Sort, Zeilenklick → `DetailModal`, Sidebar-Badges + Settings-Eintrag, Sound-Toggle + QA. **KEIN** echter WebSocket/Realtime (P4 🔒), **kein** Multi-Channel/Push (P5 🔒). Ziel: review-reif als FE-mock-first.

## Build-+-Verify-Standard — pro Phase IMMER (CLAUDE.md)
bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU) → MSW/Demo-Daten falls nötig → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → **Playwright-Screenshot-QA + Bilder WIRKLICH ansehen** (Raw-Keys / Doppelklammern / Layout / leere Zustände) → iterieren bis grün → **ein Commit + Push** → Eintrag in `qa-<modul>.md`.

## Gates & Fallen (bewährt — diese Lehren beachten!)
- **Build-Gate IMMER mit echtem Exit. NIE `npm run build | tail`** — `$?` ist dann tails Exit (immer 0) und maskiert echte Build-Fehler! Stattdessen: `npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error /tmp/build.log`.
- `npm run build` (electron-vite) ist das Compile-Gate (~1:20), weil **kalter/scoped tsc über den DetailModal-Graphen crasht** ([[project_typecheck_slow]]). Gescopter `tsconfig.*check.json` nur über reine Datenschicht-Dateien (handlers/hooks/types/stores) geht crashfrei.
- **Dev-Server killen unter Windows:** `pkill -f vite` greift NICHT (Git Bash sieht den cmd→node→esbuild-Baum nicht). PowerShell `Get-NetTCPConnection -LocalPort 5173` (bzw. 5174) + `Stop-Process`. Nur **1 Dev-Server pro QA-Runde** (war: 6 Cosmi-Fenster offen).
- **Playwright-Klick auf 20px-Icons timeout't** → JS-Klick-Fallback `locator.evaluate(el => el.click())`.
- **Datei-Tracking:** `add-*-i18n.mjs`-Scripts bleiben **untracked**; `qa-*.mjs` + `tsconfig.*check.json` werden **getrackt**. Neue Dateien unter `mocks/data/` brauchen `git add -f` (`.gitignore` ignoriert `data/`).
