# Batch-Feinplan: work (Projekte/Aufgaben) — 2026-06-08

> Fünfer-Batch nach Standard-Workflow. Modul ist bereits sehr reich (Kanban/Gantt/Liste/
> Tasks/Subtasks/Dependencies/Kommentare/Files/Timer/Custom-Fields/CRM-Links auf echtem
> Backend). Fokus: Demo-Mode QA-bar machen, settings-komplett, + 4 Markt-Gaps.

## Entscheidungen Darien (2026-06-08)
- **Sidebar:** zwei Einträge bleiben (Projekte + Aufgaben), KEIN Merge.
- **Zeit-Scope:** schwere Zeitwirtschaft (ArbZG/Pausen/Lohn-DATEV) → eigenes **zeiterfassung**-Modul. `work` behält nur leichten integrierten Timer + Task-Zeit + Reporting.
- **Markt-Gaps:** alle 4 rein (Labels/Tags · Portfolios · Kalender-Sicht · Auslastung/Budget echt).

## Ist-Stand (geprüft 2026-06-08)
- Echtes Backend via `useProjects`/`useTasks`/`useTimeEntries`/`useTaskComments`/`useTaskFiles`/`useTaskActivities`. Views: Kanban (dnd-kit), Liste (Gruppierung), Gantt (Critical-Path/Deps), Task-Detail-Page + Slide-over-Panel.
- **Lücken:** kein `WorkSettingsPanel` (nicht in `module-settings-registry.tsx`); MSW-Handler ~50% fehlen → Demo-Mode bricht; keine Labels/Tags; keine Portfolios; keine Kalender-Sicht; Auslastung/Budget/Stunden→Rechnung sind Mock.
- `### work`-Block im alten `module-phase-plans.md` beschreibt fälschlich **zeiterfassung** — dieser Feinplan ist der echte work-Plan.

---

## Phase 1 — Demo-Mode reparieren (Grundlage für QA)
**Ziel:** Alle work-Features im Demo-Mode (MSW) funktionsfähig — sonst nicht QA-/pitch-bar.
**MSW-Handler ergänzen** (`mocks/handlers/work.ts`, zustandsbehaftet wo sinnvoll): Timer start/stop + active · Zeiteinträge CRUD · Dependencies CRUD · Custom-Fields get/set · Entity-Links CRUD · Projekt-Members CRUD · Status PUT/DELETE/Reorder · Templates save/create-from · Files list/attach/remove · Activities.
**QA:** `scripts/qa-work-demo.mjs` — jede View + Task-Detail-Tabs ohne Fehler, Kanban-DnD, Timer, Kommentar, Subtask.

## Phase 2 — WorkSettingsPanel (settings-komplett)
**Ziel:** Modul „settings-komplett" via `ModuleSettingsShell`, in `module-settings-registry.tsx` registrieren (Gruppe module, navMatch `/work`).
- **Persönlich:** Standard-Ansicht (Kanban/Liste/Gantt), Standard-Gruppierung (MyTasks), Dichte, Standard-Projekt.
- **Für alle:** Projekt-Templates-Verwaltung, Default-Workflow/Status-Set, Custom-Field-Definitionen, **Label-Taxonomie** (legt Grundlage für P3), Zeit-Regeln (billable-Default).
**Store:** `stores/workPrefs.ts` (personal). Tenant mock-first wo kein Backend.

## Phase 3 — Labels/Tags auf Tasks
**Ziel:** Tasks mit Labels versehen + filtern. Markt-Standard.
- Label-Feld an Task (mock-first Overlay falls Backend fehlt → `backend-gaps.md`), Hook, Chip-UI in Kanban-Card/List-Row/Task-Detail, Add/Remove-Picker (Taxonomie aus P2-Settings), Filter in `TaskFilterBar`.

## Phase 4 — Projekt-Portfolios + Auslastung/Budget echt
**Ziel:** Modulübergreifende Übersicht + Mock-Reports real anbinden.
- **Portfolio-View:** Liste/Grid aller Projekte mit Status, Fortschritt (Tasks done/total), Fälligkeit, Budget-Ampel, Verantwortlichem. Einstieg in ProjectsListPage (Toggle Liste|Portfolio) oder eigene Sub-Route.
- **Auslastung/Budget echt:** `AuslastungReport` + `BudgetSection` von Mock auf `useTimeEntries`/`useTaskTimeSummary`-Aggregate umstellen; was Backend nicht liefert → `backend-gaps.md`. Stunden→Rechnung: light-wire zu finanzen oder dokumentieren.

## Phase 5 — Kalender-Sicht
**Ziel:** Tasks nach Fälligkeit im Kalender (Monat/Woche), Drag = Fälligkeit ändern (`useUpdateTask`).
- Neue View im Projekt-Board-Toggle (Kanban|Liste|Gantt|**Kalender**) + ggf. in MyTasks. Wiederverwendung Kalender-Primitives falls vorhanden (`modules/calendar`).

## backend-gaps.md — Sammelliste (am Batch-Ende)
Labels/Tags-CRUD (falls nicht im Task-Backend) · Budget-API · Auslastungs-Aggregat-Endpoint · Portfolio-Aggregat · Stunden→Rechnung-Verknüpfung · Guest-Project-API.
