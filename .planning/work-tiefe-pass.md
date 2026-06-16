# work — Tiefe-Pass + Standard-Konformität (Handoff für neue Session)

> **Status:** vorbereitet 2026-06-16, Bau startet in frischem Terminal.
> **Auftrag (Darien):** work auf „review-reif" bringen — voller Tiefe-Pass inkl. Daten-Swap (gewählte Option 3). Danach → Nico-Review.
> **Modul-Wahl-Begründung:** Pipeline #2, FE bereits sehr vollständig → schnellster Weg zu review-reif. finanzen ist FE-fertig (P1·P2·P2.5a–e·Modal-Tiefe·Banking-Fixes) und kann parallel an Nico.

## Ziel
work erfüllt die 6 projektweiten Pflicht-Standards und hat keine toten Buttons / Toast-Stubs / hardcoded Daten mehr. Backend darf gemockt (MSW) bleiben — Nico reviewt FE/UX, nicht Lukes Backend.

Die 6 Standards (Verweise: [[feedback_detail_modal_standard]], [[feedback_sticky_back_buttons]], [[feedback_module_depth_standard]], [[feedback_recurring_ux_patterns]]):
1. Detail-Öffnen = zentriertes `components/shared/DetailModal.tsx` (NICHT Slide-over).
2. GANZE Karte/Zeile klickbar (`role="button"` + `onKeyDown` Enter/Space; innere Buttons `e.stopPropagation()`).
3. Keine toten Buttons / Toast-Stubs (echte Wirkung).
4. Echte Downloads/Exporte (Blob), kein Toast-Stub.
5. Daten via MSW + TanStack-Query, kein hardcoded Array / reiner Zustand.
6. Sticky/immer sichtbarer Zurück-/Close-Button.

## Ist-Stand (Audit 2026-06-16 — Zeilennummern vor Bau verifizieren, Code kann sich geändert haben)

**Gut (konform, nicht anfassen):** Tasks/Projekte/Budget/Timer/Kommentare/Labels/Files laufen über `api/hooks/useTasks.ts` + `mocks/handlers/work.ts` (TanStack-Query). `TaskFileAttachments` lädt echt per presigned URL. ProjectsListPage-Karten navigieren korrekt.

**Modul-Struktur:**
- `modules/work/WorkLayout.tsx` — Router. Routen: `/work/projects` · `/projects/:id` · `/projects/:id/tasks/:taskId` · `/my-tasks` · `/search`.
- `projects/ProjectsListPage.tsx` · `projects/ProjectDetailPage.tsx` (Board: List/Kanban/Gantt/Kalender/Auslastung) · `projects/ProjectPortfolioView.tsx` · `projects/GuestProjectView.tsx`
- `tasks/TaskDetailPage.tsx` (Vollseite) · `tasks/TaskDetailPanel.tsx` (**Slide-over**) · `tasks/MyTasksPage.tsx`
- `kanban/KanbanBoard.tsx` + `KanbanCard.tsx` · `list/TaskListView.tsx` + `TaskRow.tsx` · `gantt/GanttChart.tsx` · `calendar/WorkCalendarView.tsx`
- `components/AuslastungReport.tsx` · `components/HoursToInvoiceDialog.tsx` · `components/BudgetSection.tsx` (konform) · `settings/*`
- Panel-State: `stores/work.ts` (`activeTaskId`, `taskPanelOpen`, `openTaskPanel`, `closeTaskPanel`).

## Phasen (alle in diesem Tiefe-Pass; in dieser Reihenfolge bauen)

### W-1 — Slide-over → DetailModal (groß, Kern-Verstoß)
`tasks/TaskDetailPanel.tsx` (~Z.186–223): `fixed right-0 top-0 ... w-[400px]` + Backdrop → durch `<DetailModal open={taskPanelOpen} onClose={closeTaskPanel} title={task.title} subtitle={ticketNr/projectKey} maxWidth="max-w-2xl">` ersetzen. Body (Meta-Grid, Description, Subtasks, Kommentare) als `children`, Reply/Kommentar-Eingabe als `footer`. DetailModal hat ScrollArea + sticky Header/Footer eingebaut → Standard 6 automatisch erfüllt.
- Aufrufer bleiben unverändert (rufen nur `openTaskPanel(id)`): `KanbanBoard.tsx`, `WorkCalendarView.tsx`, `TaskRow.tsx`, `GanttChart.tsx`.
- **Referenz:** `modules/finanzen/InvoiceDetailPanel.tsx` / `DunningDetailPanel.tsx` zeigen das `<DetailModal …>`-Muster mit `footer`.

### W-2 — Karten/Zeilen ganz klickbar (klein, Standard 2)
`role="button"` + `tabIndex={0}` + `onKeyDown` (Enter/Space → öffnen) auf die äußeren Container; innere Aktions-Buttons (Status/Priorität/Assignee-Popover) `e.stopPropagation()`.
- `kanban/KanbanCard.tsx` (~Z.71): **Achtung** `useSortable` von @dnd-kit setzt schon ARIA/role — nicht `role="button"` doppeln, sondern `onKeyDown` + Klick-Open ergänzen und testen, dass DnD nicht bricht.
- `list/TaskRow.tsx` (~Z.169): ganze Zeile statt nur Titel-Button klickbar.
- `tasks/MyTasksPage.tsx` (~Z.422): `div` → `role="button"` + `onKeyDown`.
- **Referenz-Muster:** `modules/finanzen/tabs/ExpensesTab.tsx` (`div role="button" tabIndex={0} onClick onKeyDown` + innere Buttons `stopPropagation`).

### W-3 — MyTasksPage tote Buttons (klein, Standard 3)
`tasks/MyTasksPage.tsx`:
- „Move to Project" (~Z.500–505): `updateTask.mutate({ id })` → **`project_id: p.id` ergänzen**. Ungenutzte `_handleMoveToProject` (~Z.253) entweder fertigstellen+nutzen oder entfernen.
- Standalone-Task-Klick (~Z.429–435): bei Tasks ohne `project_id` beim Klick `openTaskPanel(task.id)` aufrufen (öffnet nach W-1 das DetailModal) statt ins Leere.

### W-4 — „Stunden abrechnen" → echte Draft-Rechnung (mittel) ⭐ Darien-Workflow
`components/HoursToInvoiceDialog.tsx`:
- `handleCreateInvoice` (~Z.135): Toast-Stub → **echte Draft-Rechnung** via `useCreateInvoice()` aus `api/hooks/useFinance.ts` (genau wie der finanzen-HoursToInvoiceDialog, Commit `6d5bac8` — als 1:1-Vorlage nehmen: `customer`, `tax_mode:'standard'`, `currency:'EUR'`, `invoice_date`, `payment_terms_days:30`, `line_items` aus den gewählten Stunden). Ergebnis: Draft erscheint im finanzen-Rechnungen-Tab → Buchhaltung prüft. **Das ist Dariens Workflow „Leistung entsteht in work → abrechnen → Entwurf geht an Buchhaltung".**
- Hardcoded `MOCK_TIME_ENTRIES` (~Z.47–56) → MSW: neuer Handler `GET /api/v1/projects/:id/time-entries?billed=false` in `mocks/handlers/work.ts` + Hook `useProjectTimeEntries`. Seed in `mocks/data/`.
- Kunde: work-Dialog hat ggf. Projekt-Kontext → Kunde aus Projekt ableiten falls vorhanden, sonst Kunde-Feld wie beim finanzen-Dialog.
- **Offene Entscheidung mit Darien:** finanzen-Header-Button „Stunden abrechnen" danach entfernen/reduzieren, da work der primäre Ort ist? (Sein Punkt 2026-06-16.)

### W-5 — Daten-Swap Auslastung + Gast-Ansicht (mittel-groß, Standard 5)
- `components/AuslastungReport.tsx` (~Z.64–100): hardcoded `MOCK_TEAM`/`MOCK_UTILIZATION` → MSW-Handler `GET /api/v1/projects/:id/team-utilization` + Hook; `projectId`-Prop nutzen. Seed in `mocks/data/`.
- `projects/GuestProjectView.tsx` (~Z.45–95): hardcoded `MOCK_PROJECT`/`MOCK_MILESTONES`/`MOCK_STATUS_UPDATES`, ignoriert `projectId` (`_projectId`) → `useProject(projectId)` + echte Felder; Milestones/Status-Updates über MSW-Handler oder vorhandene Task-Daten ableiten.

## Referenz-Patterns (Vorlagen im Code)
- **DetailModal-Nutzung + footer:** `modules/finanzen/InvoiceDetailPanel.tsx`, `RecurringDetailPanel.tsx`. Props: `open, onClose, title?, subtitle?, badge?, children, footer?, maxWidth?, onBack?` (`components/shared/DetailModal.tsx`).
- **Klickbare Zeile:** `modules/finanzen/tabs/ExpensesTab.tsx` + `TransactionsTab.tsx`.
- **MSW-Handler stateful:** `mocks/handlers/finance.ts` (z.B. bank-transactions match/connect), Array `workHandlers` analog in `mocks/handlers/work.ts`.
- **Echte Rechnung aus Stunden:** `modules/finanzen/HoursToInvoiceDialog.tsx` (Commit `6d5bac8`) — direkte Vorlage für W-4.
- **Client+Hook-Muster:** `api/finance-client.ts` + `api/hooks/useFinance.ts`.

## Gates & Workflow (verbindlich)
- **Build statt kaltem tsc** ([[project_typecheck_slow]]): kalter/scoped `tsc` über schwere Panel-Graphen CRASHT (`Debug Failure. No error for last overload signature`) — NICHT als Gate brauchbar. Stattdessen `npm run build` (electron-vite, ~1–2 min) als Bundle-Gate. Optional scoped `tsc -p tsconfig.<name>check.json` NUR über Datenschicht-Dateien (handlers/client/hooks/data, OHNE DetailModal-Importeure) für Typsicherheit — die crasht nicht.
- **Playwright-QA + Screenshots WIRKLICH ansehen** ([[feedback_qa_thoroughness]]): Dev-Server `npm run dev` (Port 5173), QA-Script-Muster siehe `desktop/scripts/qa-finanzen-banking-fix.mjs` / `qa-finanzen-p25e.mjs` (ELECTRON_STUB + SUPPRESS_ONBOARDING, `goto /#/work/...`, screenshots in `.qa-screenshots/`, scanRawKeys + pageErrors). Pro Fix: Zeilen-/Karten-Klick → DetailModal offen, tote Buttons wirken, 0 Raw-Keys, 0 pageErrors.
- **Dev-Server killen (Windows)** ([[feedback_kill_dev_server_windows]]): PowerShell `Get-NetTCPConnection -LocalPort 5173` → Prozessbaum `Stop-Process -Force` + `Get-Process electron | Stop-Process`. Nur 1 Dev-Server pro QA-Runde.
- **i18n** ([[reference_i18next_icu_braces]]): neue Keys ×4 (de/en/fr/it) via `desktop/scripts/add-*-i18n.mjs` (Muster vorhanden), `{var}` single-brace, Plural als ICU. `add-*-i18n.mjs` bleiben **untracked** (Konvention); `qa-*.mjs` + `tsconfig.*check.json` werden **getrackt**.
- **mocks/data/** ([[reference_gitignore_data_dir]]): neue Dateien dort brauchen `git add -f` (von `.gitignore data/` ignoriert).
- **Git:** direct-to-main, Conventional Commits, keine AI-Attribution, am Session-Ende pushen. Vor Push `git fetch` + `git pull --rebase` (Luke pusht in Wellen — diese Session kam 0f49fd7 dazwischen).

## Commit-Plan (Vorschlag)
1. `refactor(work): replace task slide-over with centered DetailModal` (W-1)
2. `feat(work): make task cards/rows fully clickable + keyboard` (W-2)
3. `fix(work): repair move-to-project + standalone-task open in MyTasks` (W-3)
4. `feat(work): issue real draft invoice from tracked hours` (W-4)
5. `feat(work): serve utilization + guest view from MSW` (W-5)
(Bei Bedarf zusammenfassen — geteilte Dateien wie `mocks/handlers/work.ts` ggf. bündeln.)

## Verifikation (end-to-end)
`npm run build` grün → Dev-Server + Playwright: Aufgabe in Kanban/Liste/Kalender/Gantt anklicken → zentriertes DetailModal; MyTasks „Move to Project" wirkt + Standalone-Task öffnet Detail; „Stunden abrechnen" → Draft erscheint im finanzen-Rechnungen-Tab; Auslastung/Gast zeigen MSW-Daten. Screenshots ansehen. Danach Master-Tracker work abhaken + „→ Nico".

---
## Starttext für die neue Session (kopieren)
> „Wir machen den work-Tiefe-Pass (review-reif). Voller Plan inkl. Audit, Phasen W-1…W-5, Referenz-Patterns und Gates steht in `.planning/work-tiefe-pass.md` — zuerst lesen. Erst `git fetch` + `git pull --rebase` (Luke pusht in Wellen), dann Zeilennummern im Audit gegen den aktuellen Code verifizieren, dann W-1 starten. Build statt kaltem tsc, Playwright-Screenshots wirklich ansehen, i18n ×4, `git add -f` für mocks/data."
