# Parallel-Batch — Koordination (Main + Sub) — team + helpdesk

> **Stand 2026-06-17 (Batch 2).** Zwei Terminals bauen parallel, damit sie sich **nicht in die Quere kommen**. Main = **team** (HR, groß, Tiefe-Pass + P1 Schulungen), Sub = **helpdesk** (Ticket-System, Demo-tief). Vorheriger Batch (dashboard + vertraege) ist durch → siehe `qa-combined.md`.

## Rollen & Klone

| | **Main-Terminal** | **Sub-Terminal** |
|---|---|---|
| Klon | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (frisch geklont, node_modules ready) |
| Dev-Port | **5173** | **5174** |
| Modul | **team** (TM-1…TM-5) | **helpdesk** (H-1…H-8) |
| Plan-Datei | `main-team.md` | `sub-helpdesk.md` |
| QA-Protokoll | `qa-team.md` (selbst anlegen) | `qa-helpdesk.md` (selbst anlegen) |
| Branch | **`main`** (Hauptklon) | **`parallel/helpdesk`** (Iso, NICHT direct-to-main) |
| Zusatzrolle | plant beide Batches, beantwortet Klärungen, merged am Ende `parallel/helpdesk`, stellt kombinierte QA-Liste | baut nur ab, meldet Darien Fortschritt (`H-x fertig, n/8`) |

## Lane-Trennung — was kollidiert NICHT
- **Modul-Code disjunkt:** `modules/team/` vs. `modules/helpdesk/`. Kein Overlap.
- **Stores disjunkt:** team nutzt `lib/hr-hooks.ts` + `stores/useTeamStore`/`stores/payroll*` · helpdesk nutzt `stores/helpdesk.ts`. Kein Overlap.
- **MSW-Handler:** Main fasst NUR `mocks/handlers/team.ts` an (bereits in `index.ts` registriert → **kein** `index.ts`-Touch). Sub fasst `mocks/` **gar nicht** an (Demo-tief = Store-Actions, kein neuer MSW-Handler). → **null `mocks/index.ts`-Konflikt.**
- **Settings-Panels disjunkt:** team hat schon ein Panel · Sub legt `HelpdeskSettingsPanel` neu an + trägt EINEN Eintrag in `module-settings-registry.tsx` ein (Main fasst diese Datei nicht an → kein Konflikt).
- **QA-Protokolle disjunkt:** jeder schreibt nur seine eigene `qa-*.md`.

## Lane-Trennung — die echten Reibungspunkte (Regeln verbindlich)

1. **i18n — die einzige garantierte Kollision.** Beide hängen Keys an `i18n/messages/{de,en,fr,it}.json` an.
   - team-Keys leben unter `team.*` · helpdesk-Keys unter `helpdesk.*`. **Verschiedene Prefix-Cluster** → Git-Konflikt nur an Objektgrenzen, schnell lösbar.
   - **Regel:** Keys ins jeweilige Prefix-Cluster einsortieren (nicht ans Datei-Ende klatschen). `{var}` **single-brace** ×4 Sprachen, ICU-Plural `{count, plural, …}` — **nie** `{{var}}`, nie `_one`/`_other`.
   - Dank Branch-Isolation (Regel 4) entsteht **kein** Live-i18n-Konflikt — der eine mögliche Konflikt wird beim **finalen Merge** durch das Main-Terminal gelöst (beide Key-Blöcke behalten, danach `npm run build`).

2. **`shared/` einfrieren.** In diesem Lauf baut **kein** Terminal neue `shared/`-Komponenten.
   - helpdesk **konsumiert nur** `shared/DetailModal` + `shared/SortMenu` (existieren bereits). Nicht ändern, nur nutzen.
   - team fasst `shared/` nicht an.
   - Falls doch ein neues shared-Pattern nötig wird → **stopp, mit Darien/Main absprechen**, Main baut es zuerst + pusht, Sub pullt. Nicht parallel.

3. **Routing/Sidebar nicht anfassen.** Beide Module sind bereits geroutet. (Main hat vor dem Batch einmalig `nav-items.ts` + `layout.navItems.automatisierung` für das automatisierung-Modul angefasst — das ist abgeschlossen & gepusht, bevor der Sub startet. Sub fasst Routing/Sidebar NICHT an.)

4. **Branch-Isolation (Sicherung gegen main-Konflikte):** Das **Sub-Terminal baut auf `parallel/helpdesk`** (nicht direct-to-main), das **Main-Terminal auf `main`**. So gibt es während des Laufs **null Live-Konflikt** auf `main`. Das Main-Terminal merged `parallel/helpdesk` am Ende **einmal kontrolliert** (i18n-Konflikt: beide Key-Blöcke behalten, danach `npm run build`).

## Entscheidungen (Darien, beantwortet — keine Rückfragen mehr nötig)
- **helpdesk-Scope = „Demo-tief":** Store-Actions (Mutationen wirken!) + DetailModal + Settings-Panel + Demo-Tiefe + i18n. **KEIN** TanStack-Migration, **KEIN** MSW-Handler, **KEIN** CRM-Kontakt-Lookup (das ist Lukes Backend / späterer Batch). Ziel: review-reif als FE-mock-first.
- **team-Scope = „Tiefe-Pass + P1 Schulungen":** Bugs/Stubs/i18n review-reif machen + Schulungen-Tab auf echten Hook. P2 (Personalakte↔Dok, Organigramm editierbar) ist NICHT in diesem Batch.

## Build-+-Verify-Standard — pro Phase IMMER (CLAUDE.md)
bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU) → Demo-/Store-Daten falls nötig → **gescopter** Typecheck (nie Full-tsc als Gate) → **Playwright-Screenshot-QA + Bilder WIRKLICH ansehen** (Raw-Keys / Doppelklammern / Layout / leere Zustände) → iterieren bis grün → **ein Commit + Push** → Eintrag in `qa-<modul>.md`.

## Gates & Fallen (bewährt — diese Lehren beachten!)
- **Build-Gate IMMER mit echtem Exit. NIE `npm run build | tail`** — `$?` ist dann tails Exit (immer 0) und maskiert echte Build-Fehler! Stattdessen: `npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + dann `grep -i error /tmp/build.log`.
- `npm run build` (electron-vite) ist das Compile-Gate (~1:20), weil **kalter/scoped tsc über den DetailModal-Graphen crasht** ([[project_typecheck_slow]]). Gescopter `tsconfig.*check.json` nur über reine Datenschicht-Dateien (handlers/hooks/types/stores) geht crashfrei.
- **`\n`-Prefix-Edit-Trick beim Entfernen von Code kann Import-Zeilen VERSCHMELZEN** (`'@/lib/format'import{…}` = Missing-Semicolon). Nach solchen Edits IMMER Build prüfen.
- **Dev-Server killen unter Windows:** `pkill -f vite` greift NICHT (Git Bash sieht den cmd→node→esbuild-Baum nicht). PowerShell `Get-NetTCPConnection -LocalPort 5173` (bzw. 5174) + `Stop-Process`. Nur 1 Dev-Server pro QA-Runde.
- **Playwright-Klick auf 20px-Icons timeout't** (Topbar-Uhr-Re-Render → „instabil") → JS-Klick-Fallback `locator.evaluate(el => el.click())`.
- **PDF/Preview headed testen** (headless hat keinen PDF-Viewer).
- **Datei-Tracking:** `add-*-i18n.mjs`-Scripts bleiben **untracked**; `qa-*.mjs` + `tsconfig.*check.json` werden **getrackt**. Neue Dateien unter `mocks/data/` brauchen `git add -f` (`.gitignore` ignoriert `data/`).
