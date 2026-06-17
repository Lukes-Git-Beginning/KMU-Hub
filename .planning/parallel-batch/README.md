# Parallel-Batch — Koordination (Main + Sub, 5+5 Tiefe-Pass)

> **Stand 2026-06-17.** Zwei Terminals bauen parallel, damit sie sich **nicht in die Quere kommen**. Main = dashboard, Sub = vertraege. Beide als gezielter **Tiefe-Pass** (Module sind je ~75 % fertig), nicht als Neubau.

## Rollen & Klone

| | **Main-Terminal** | **Sub-Terminal** |
|---|---|---|
| Klon | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (frisch geklont, node_modules ready) |
| Dev-Port | **5173** | **5174** |
| Modul | **dashboard** (5 Punkte D-1…D-5) | **vertraege** (5 Punkte V-1…V-5) |
| Plan-Datei | `main-dashboard.md` | `sub-vertraege.md` |
| QA-Protokoll | `qa-dashboard.md` | `qa-vertraege.md` |
| Zusatzrolle | plant beide Batches, beantwortet Klärungen, stellt am Ende die **kombinierte QA-Liste** für Darien zusammen | baut nur ab, meldet Darien Fortschritt (`V-x fertig, n/5`) |

## Lane-Trennung — was kollidiert NICHT
- **Modul-Code disjunkt:** `modules/dashboard/` + `components/dashboard/` vs. `modules/vertraege/`. Kein Overlap.
- **MSW-Handler disjunkt:** `mocks/handlers/dashboard.ts` vs. `mocks/handlers/vertraege.ts` (+ ggf. notifications/dokumente — siehe unten).
- **Stores disjunkt:** `stores/dashboard*.ts` vs. `stores/vertraege*.ts`.
- **QA-Protokolle disjunkt:** jeder schreibt nur seine eigene `qa-*.md`.

## Lane-Trennung — die echten Reibungspunkte (Regeln verbindlich)

1. **i18n — die einzige garantierte Kollision.** Beide hängen Keys an `i18n/messages/{de,en,fr,it}.json` an.
   - dashboard-Keys leben unter `dashboard.*` / `widgets.*`; vertraege-Keys unter `vertraege.*`. **Verschiedene Top-Level-Objekte** → Git-Konflikt nur an Objektgrenzen (Komma/Klammer), schnell lösbar.
   - **Regel:** Keys ins jeweilige Top-Level-Objekt einsortieren (nie ans Datei-Ende klatschen). `{var}` single-brace ×4 Sprachen, ICU-Plural `{count, plural, …}`.
   - **Regel:** `git pull --rebase` **vor jedem Push**. Bei JSON-Konflikt: der Rebasende löst auf (meist nur eine Komma/Klammer-Zeile an der Objektgrenze).

2. **`shared/` einfrieren.** In diesem Lauf baut **kein** Terminal neue `shared/`-Komponenten.
   - vertraege V-1 **konsumiert nur** `shared/DetailModal` (existiert bereits, von work/kontakte/dokumente). Nicht ändern, nur nutzen.
   - dashboard fasst `shared/` nicht an.
   - Falls doch ein neues shared-Pattern nötig wird → **stopp, mit Darien/Main absprechen**, Main baut es zuerst + pusht, Sub pullt. Nicht parallel.

3. **Routing/Sidebar nicht anfassen.** Beide Module sind bereits geroutet. Keine neuen Routen in diesem Batch.

4. **Atomare Pushes:** committen → `git pull --rebase` → push. So bleibt `main` linear (beide schreiben direct-to-main).

## Build-+-Verify-Standard — pro Phase IMMER (CLAUDE.md)
bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU) → Demo-/MSW-Handler falls nötig → **gescopter** Typecheck (nie Full-tsc als Gate) → **Playwright-Screenshot-QA + Bilder WIRKLICH ansehen** (Raw-Keys / Doppelklammern / Layout / leere Zustände) → iterieren bis grün → **ein Commit + Push**.

### Gates (bewährt, [[project_typecheck_slow]])
- **Kalter/scoped tsc über DetailModal-Panel-Graphen CRASHT** → stattdessen `npm run build` (electron-vite, ~1:20) als Compile-Gate. Scoped `tsconfig.*check.json` nur über Datenschicht-Dateien (handlers/hooks/types/stores) ist crashfrei.
- **Dev-Server killen (Windows):** PowerShell `Get-NetTCPConnection -LocalPort <5173|5174>` → `Stop-Process`. `pkill -f vite` greift NICHT.
- **Playwright-Klick auf 20px-Icons timeout't** (Topbar-Uhr-Re-Render → „instabil") → JS-Klick-Fallback `locator.evaluate(el => el.click())`.
- **Sub-Terminal: QA-Scripts gegen `http://localhost:5174`** (BASE-Konstante in `qa-*.mjs` anpassen).
- `add-*-i18n.mjs` bleiben **untracked**; `qa-*.mjs` + `tsconfig.*check.json` werden **committet**.

## QA-Protokoll — wie Darien am Ende reviewt
- Pro fertiger Phase **ein Eintrag** in der eigenen `qa-*.md`: was gebaut, Schlüsseldatei(en), **was Darien anschauen soll** (welcher Screen, welche Aktion, worauf achten), Screenshot-Pfad.
- Punkte, die **mehrere Phasen betreffen** (Pattern-Entscheidungen, z. B. „so sieht das DetailModal jetzt aus"), mit **`[PATTERN]`** markieren → Darien schaut die **zuerst** an.
- Sobald **alle 10 Phasen** durch sind: Main pullt, liest **beide** `qa-*.md`, baut die **kombinierte priorisierte Liste** (Pattern zuerst, dann pro Modul chronologisch) und gibt sie Darien. Darien geht sie Stück für Stück durch.

## Entscheidungen (Darien, 2026-06-17) — in die Pakete eingebaut
- **Schnitt:** 5+5 Tiefe-Pass (nicht 6+4 Neubau).
- **vertraege Detail:** Slide-over → **zentriertes DetailModal** (Standard-Konformität).
- **dashboard Lizenz-Gating:** Infrastruktur sauber, aber im **Demo alle Flags an** (niemand sucht ein fehlendes Widget).
- **vertraege E-Signatur „Senden":** **realistischer Demo-Flow** (simulierter Versand/Rücklauf, klar als Demo gekennzeichnet, kein echter Mailversand). Skribble bleibt „bald verfügbar".
