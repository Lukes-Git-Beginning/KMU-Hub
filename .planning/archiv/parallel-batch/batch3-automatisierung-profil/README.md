# Parallel-Batch — Koordination (Main + Sub) — automatisierung + profil

> **Stand 2026-06-19 (Batch 3).** Zwei Terminals bauen parallel, je **5 Phasen**, damit sie sich **nicht in die Quere kommen**. Main = **automatisierung** (Tiefe-Pass, MSW-Vertrag reparieren), Sub = **profil** (Dokumente echt + Cleanup). Vorheriger Batch (team + helpdesk) ist durch.

## Rollen & Klone

| | **Main-Terminal** | **Sub-Terminal** |
|---|---|---|
| Klon | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (auf `main @ 0596e5bc`, node_modules ready) |
| Dev-Port | **5173** | **5174** |
| Modul | **automatisierung** (A-1…A-5) | **profil** (P-1…P-5) |
| Plan-Datei | `main-automatisierung.md` | `sub-profil.md` |
| QA-Protokoll | `qa-automatisierung.md` (selbst anlegen) | `qa-profil.md` (selbst anlegen) |
| Branch | **`main`** (Hauptklon, direct-to-main) | **`parallel/profil`** (Iso, NICHT direct-to-main) |
| Zusatzrolle | plant beide Batches, beantwortet Klärungen, merged am Ende `parallel/profil`, stellt kombinierte QA-Liste | baut nur ab, meldet Darien Fortschritt (`P-x fertig, n/5`) |

## Lane-Trennung — was kollidiert NICHT
- **Modul-Code disjunkt:** `modules/automatisierung/` vs. `modules/profil/`. Kein Overlap.
- **Stores disjunkt:** automatisierung nutzt `stores/automatisierung.ts` · profil nutzt `stores/settings`/`presence`/`notifications`/`auth` (read). Kein Overlap.
- **MSW-Handler:** Main fasst NUR `mocks/handlers/automation.ts` an (+ ggf. EINMALIG `mocks/handlers/index.ts`, falls `automationHandlers` dort noch nicht registriert ist — A-1). Sub fasst NUR `mocks/handlers/hr.ts` an (Dokumente/Avatar-Handler, **bereits** in `index.ts` registriert → **kein** `index.ts`-Touch). → höchstens Main berührt `index.ts`, Sub nie → **null index.ts-Konflikt**.
- **Settings-Registry:** Main (A-5) ergänzt EINEN `automatisierung`-Eintrag in `module-settings-registry.tsx`. **Sub fasst diese Datei NICHT an** (profil-Einstellungen leben per Designentscheidung unter `/settings`, nicht in der Modul-Registry) → kein Konflikt.
- **QA-Protokolle disjunkt:** jeder schreibt nur seine eigene `qa-*.md`.

## Lane-Trennung — die echten Reibungspunkte (Regeln verbindlich)

1. **i18n — die einzige garantierte Kollision.** Beide hängen Keys an `i18n/messages/{de,en,fr,it}.json` an.
   - automatisierung-Keys leben unter `automatisierung.*` (+ `api.automation.*`) · profil-Keys unter `profil.*`. **Verschiedene Prefix-Cluster** → Git-Konflikt nur an Objektgrenzen.
   - **Regel:** Keys ins jeweilige Prefix-Cluster einsortieren (nicht ans Datei-Ende klatschen). `{var}` **single-brace** ×4 Sprachen, ICU-Plural `{count, plural, …}` — **nie** `{{var}}`, nie `_one`/`_other`.
   - Dank Branch-Isolation (Regel 4) entsteht **kein** Live-i18n-Konflikt — der eine mögliche Konflikt wird beim **finalen Merge** durch das Main-Terminal gelöst (beide Key-Blöcke behalten, danach `npm run build`).

2. **`shared/` einfrieren.** In diesem Lauf baut **kein** Terminal neue `shared/`-Komponenten.
   - automatisierung **konsumiert nur** `shared/DetailModal` + `shared/ConfirmDialog` (existieren). profil **konsumiert nur** `shared/DetailModal` + `shared/RichTextEditor` (existieren). Nicht ändern, nur nutzen.
   - Falls doch ein neues shared-Pattern nötig wird → **stopp, mit Darien/Main absprechen**, Main baut es zuerst + pusht, Sub pullt. Nicht parallel.

3. **Routing/Sidebar/nav-items nicht anfassen.** Beide Module sind bereits geroutet. Sub fasst Routing/Sidebar NICHT an. (Main braucht für A-1…A-5 auch keine Routing-Änderung.)

4. **Branch-Isolation (Sicherung gegen main-Konflikte):** Das **Sub-Terminal baut auf `parallel/profil`** (nicht direct-to-main), das **Main-Terminal auf `main`**. So gibt es während des Laufs **null Live-Konflikt** auf `main`. Das Main-Terminal merged `parallel/profil` am Ende **einmal kontrolliert** (i18n-Konflikt: beide Key-Blöcke behalten, danach `npm run build`).

5. **`git pull` vor erstem Branch (Sub) und vor jedem Main-Push.** Beide starten auf `0596e5bc`; Main pusht laufend nach main, Sub bleibt isoliert bis zum Merge.

## Scope-Entscheidungen (Darien-Default — keine Rückfragen nötig)
- **automatisierung = „Tiefe-Pass, FE-mock-first":** MSW-Vertrag reparieren (Demo wird lebendig), DetailModal, Löschen/Duplizieren, Log/Editor verkabeln, Settings-Panel, i18n-Schlusscheck. **KEINE** echte Engine, **KEIN** echtes Backend (das ist Lukes Block). Ziel: review-reif als FE-mock-first.
- **profil = „Dokumente echt (MSW) + current-user + Cleanup":** Dokumente-Tab über MSW lebendig, current-user-Single-Source, Avatar/DND demo-tief, toter `tabs/zeiterfassung/`-Ordner weg, Demo-Tiefe-Schlusscheck. **KEIN** echtes Avatar-/Doc-Storage-Backend (Luke). Ziel: review-reif als FE-mock-first.

## Build-+-Verify-Standard — pro Phase IMMER (CLAUDE.md)
bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU) → Demo-/MSW-Daten falls nötig → **Compile-Gate `npm run build`** (nie Full-tsc als Gate) → **Playwright-Screenshot-QA gegen den eigenen Port + Bilder WIRKLICH ansehen** (Raw-Keys / Doppelklammern / Layout / leere Zustände) → iterieren bis grün → **ein Commit + Push** → Eintrag in `qa-<modul>.md`.

## Gates & Fallen (bewährt — beachten!)
- **Build-Gate IMMER mit echtem Exit. NIE `npm run build | tail`** — `$?` ist dann tails Exit (immer 0) und maskiert Build-Fehler! Stattdessen: `npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error /tmp/build.log`. [[feedback_build_check_real_exit]]
- `npm run build` (electron-vite, ~1:20) ist das Compile-Gate, weil kalter/scoped tsc über den DetailModal-Graphen crasht. [[project_typecheck_slow]]
- **`\n`-Prefix-Edit-Trick beim Entfernen von Code kann Import-Zeilen VERSCHMELZEN** → nach solchen Edits IMMER Build prüfen.
- **Dev-Server killen unter Windows:** `pkill -f vite` greift NICHT. PowerShell `Get-NetTCPConnection -LocalPort 5173` (Main) bzw. `5174` (Sub) + `Stop-Process`. Nur 1 Dev-Server pro QA-Runde pro Klon.
- **Playwright-Skeleton/Suspense + 20px-Icons:** JS-Klick-Fallback `locator.evaluate(el => el.click())`; Onboarding via `localStorage cosmi-ui onboardingCompleted=true` + electronAPI-Stub (siehe `scripts/qa-skeletons.mjs`).
- **Datei-Tracking:** `add-*-i18n.mjs`-Scripts bleiben **untracked**; `qa-*.mjs` werden **getrackt**. Neue Dateien unter `mocks/data/` brauchen `git add -f` (`.gitignore` ignoriert `data/`).
- **Sub: QA-Script `BASE` auf `http://localhost:5174` setzen** (nicht 5173).
