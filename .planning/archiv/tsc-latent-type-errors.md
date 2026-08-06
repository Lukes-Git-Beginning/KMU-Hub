# Latente Typfehler im work-Modul (vom tsc-Crash bisher verdeckt)

> **Kontext (2026-06-19):** Der scoped `tsc` crashte bisher mit einem internen TS-Compiler-Bug
> (`Debug Failure. No error for last overload signature`, microsoft/TypeScript#63195) — ausgelöst
> durch `i18next.d.ts` `resources: typeof de.json` (~8.5k Keys). Fix: `resources`-Typ weggelassen
> (Commit `87868ade`). Seitdem **läuft tsc wieder durch** — und deckt ~31 **vorbestehende** latente
> Typfehler im work-Modul auf, die der Crash verbarg. **Keiner stammt aus dem neuen Quick-Actions-Code.**
> Alle laufen im Demo (MSW liefert die Felder), aber es ist echter Typ-/Mock-Drift → aufräumen.
> Reproduzieren: `cd desktop && node node_modules/typescript/bin/tsc -p tsconfig.workqa.json --noEmit`

## Gruppiert (Aufräum-Aufgaben)

### 1. Member-Typ: `display_name` / `email` fehlen (häufigster)
Code nutzt `member.display_name` / `member.email`, der OpenAPI-Member-Typ hat aber nur
`first_name`/`last_name`/`user_id`/`role`. → Member-Typ erweitern **oder** Code auf vorhandene Felder umstellen.
- `components/CommentThread.tsx` 92, 93, 152, 466, 472, 476
- `components/TaskCreateDialog.tsx` 223
- `tasks/TaskDetailPage.tsx` 595
- `tasks/TaskDetailPanel.tsx` 425

### 2. Task-Typ: `is_closed` / `created_by_name` fehlen
`useTask`-Ergebnis-Typ (OpenAPI) kennt `is_closed`/`created_by_name` nicht; MSW liefert sie.
→ Task-Response-Schema ergänzen (Luke) oder Felder im FE ableiten.
- `tasks/TaskDetailPage.tsx` 379, 392, 486, 738 (`created_by_name`→`created_by`?), 741
- `tasks/TaskDetailPanel.tsx` 312

### 3. priority `medium` vs OpenAPI `normal`
App nutzt durchgängig `'medium'`, OpenAPI-Enum ist `'low'|'normal'|'high'|'urgent'`. Langjähriger Mismatch.
→ Eine Schreibweise festlegen (Mapping im Client oder Enum angleichen).
- `api/hooks/useTasks.ts` 50, 83, 166 · `tasks/MyTasksPage.tsx` 170

### 4. lucide-react akzeptiert kein `title`-Prop
`<Lock title={…} />` etc. → `aria-label` nutzen oder in `<span title>` wrappen.
- `kanban/KanbanCard.tsx` 217 · `list/TaskRow.tsx` 258

### 5. useProjects-Mismatches
- 316: `is_default`/`is_closed` optional vs. required im Ziel-Typ.
- 421: `view_type: 'gantt'` nicht im OpenAPI-Enum `'list'|'kanban'`.

### 6. Einzelne
- `components/CustomFieldsSection.tsx` 53 — Funktion mit 1 Argument ohne Argument aufgerufen.
- `components/TaskTimer.tsx` 191 — `summary.entry_count` möglicherweise `undefined` (Guard fehlt).

## Hinweis i18n-Typsicherheit
Mit dem Fix verliert `t('…')` die compile-zeitliche Key-Validierung (Autocomplete/Tippfehler).
Gate dafür bleibt die **Playwright-Screenshot-QA** (Raw-Key-Erkennung). Falls Key-Typsicherheit
zurück soll, ohne den Crash: i18next-Namespaces splitten (mehrere kleine JSON statt einer 8.5k-Datei)
und je NS typisieren — dann bleibt jede Resource-Union klein genug.
