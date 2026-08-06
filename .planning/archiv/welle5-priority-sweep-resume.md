# Mock-Exit Welle 5 — Resume-Prompt (frische Session)

## Gemeinsamer Stand (verifiziert 2026-06-24)

- **`origin/main = b49342a0`, Pipeline KOMPLETT GRÜN** (CI ✓ · CI Desktop ✓ · CD ✓ inkl. „Deploy via SSH" + „Post-deploy smoke check" success). Prod deployed. Migrationskopf **230**, **nächste freie Migr = 231** (kein Kollisionsrisiko verifiziert).
- **Paralleler Workstream (Darien) ist aktiv** auf security/DSGVO-FE, HR/zeiterfassung, dialer. Er baut linear auf main weiter. → **Vor Push immer `git fetch` + `git rebase origin/main`** (saubere, überlappungsfreie Integration; Wave-4-Rebase lief 100% konfliktfrei). **Darien's Dateien NICHT anfassen** (security/*, hr/*, dialer/*) außer triviale CI-Unblocker nach Absprache.
- **Worktree-Base-Falle:** In einer FRISCHEN Session branchen Worktree-Agenten vom aktuellen Head → Delegation ist wieder sicher. (In der Vorsession war Delegation unsicher, weil Session-Start-HEAD veraltet war.)

## Welle 4 ERLEDIGT (NICHT erneut anfassen) — wiki + helpdesk mock-exit

Commits `61a6b96e`..`0b740522` (+ `b49342a0` Security-Lint-Fix), alle gepusht + grün + CD-deployt:
- 🔴 **Wiki tenant_id-INSERT-Bug** (prod-latent seit Migr.230) gefixt: `wiki_versions/attachments/share_tokens` Models+Repo-INSERT+Service.
- Wiki **ListShareTokens**-RPC (war einziger echt fehlender RPC) + permissions proto `string`→`repeated string` (Array) + UploadAttachmentRequest.tenant_id.
- FE: **content base64↔Objekt-Adapter** (`wiki-content.ts`, UTF-8-sicher, unit-getestet + Umlaut-Round-Trip live), **Timestamp-Normalisierung** (`normalizeWireTimestamps` in wiki-client + helpdesk-client), updateArticle PUT→PATCH.
- **8 List-Endpoints unwrap-gefixt** (`unwrapList`, tolerant: Mock-bare-Array + real-`{key:[…]}` + empty→[]) — behob ErrorBoundary-Crashes (categories/canned/queues/sla/versions/attachments/share-tokens/messages), nur per Playwright gefunden.
- Helpdesk Backend real verifiziert (RLS war doch komplett via Migr.122 — Explore hatte sich geirrt; kein Code nötig außer FE-Timestamps). OpenAPI 16 Routen dokumentiert (`docs(openapi)`).
- **Bereits echt-geschaltet, nicht anfassen:** notifications, dialer, work-labels, settings, vertraege, hr, **wiki, helpdesk**.

## WELLE 5 — priority-Sweep `medium` → `normal` (FE + VOLLER Backend, Migr.231)

User-Entscheidung: **voller Backend-Sweep** (nicht nur FE). Grund: `tasks.priority` lässt `normal` aktuell nicht zu, und der Gateway-Validate-Tag ist aktiv falsch.

**Reihenfolge: Migr.231 zuerst (DB akzeptiert `normal`), dann Backend-Code, dann FE.**

### 1. Migr.231 `000231_normalize_task_priority_medium_to_normal`
- `000025_create_tasks.up.sql:15` hat aktuell: `priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('urgent','high','medium','low'))` — `normal` fehlt.
- Migration: `UPDATE tasks SET priority='normal' WHERE priority='medium';` → alten CHECK droppen → `ALTER COLUMN priority SET DEFAULT 'normal'` → `CHECK (priority IN ('urgent','high','normal','low'))`.
- **Industry-Template-Seed prüfen:** `000058_seed_industry_templates.up.sql:181` enthält JSONB `"priority":"medium"`. Zieltabelle ermitteln; entweder JSONB in 231 normalisieren ODER bestätigen, dass Template-Instanziierung im App-Layer mappt (sonst bricht Template→Task-Erzeugung am neuen CHECK).
- `tasks` hat bereits tenant_id+RLS (Rollout 117–127) → reines ALTER, **kein neuer RLS, kein neuer Permission-Guard**. down-Migration spiegeln (forward-only beim Deploy).

### 2. Backend-Code
- `backend/internal/models/task.go`: `TaskPriorityMedium = "medium"` → `TaskPriorityNormal = "normal"`; `ValidTaskPriorities`-Map anpassen. **Grep alle Backend-Refs** auf `TaskPriorityMedium`/`"medium"` im Task-Kontext.
- `backend/internal/gateway/route_work_tasks.go:~22`: Validate-Tag `oneof=low medium high critical` (lehnt normal+urgent ab!) → `oneof=low normal high urgent`. (Es gibt Create- UND Update-Body-Structs — beide.)

### 3. FE-Sweep (~48 Stellen, ~15 Dateien — Explore-Trefferliste a–i, Stand valide, Parallel-Workstream hat work/priority NICHT angefasst)
- Mocks: `mocks/data/mock-db.ts` (~10 Task-Werte + `TaskPriority`-Typ), `mocks/handlers/work.ts` (3) — `priority:'medium'`→`'normal'`.
- Filter-Pills: `TaskFilterBar` (~:54), `TaskListHeader` (~:60), `MyTasksPage` (~:71) — value `medium`→`normal`.
- `modules/work/components/PriorityBadge.tsx`: `type Priority` (:11) + config-Key (:31) + Fallback (:55). **Exportierter Typ** → zentral.
- ~14× `?? 'medium'` → `?? 'normal'` (Grep `?? 'medium'`).
- 4× Sort-Arrays `['urgent','high','medium','low']` → `['urgent','high','normal','low']`.
- `modules/work/.../WorkCalendarView.tsx`: priorityDot-Key (~46–51) + Fallback (~354).
- `modules/work/gantt/gantt-utils.ts`: Fallback (~469) + `case 'medium'` (~506).
- `api/hooks/useTasks.ts`: Union-Typen (:23 TaskListParams, :173 useCreateTask, :207 useUpdateTask) `medium`→`normal`; `toApiPriority` (:47–53) vereinfachen (medium-Branch wird tot).
- **i18n (HOT-FILE, nur Main):** falls Priority-Label-Keys `medium`/`normal` betroffen → `desktop/.../i18n/messages/*.json` ×4 (`{var}` nie `{{var}}`).

### 4. Gate (verbindlich)
- Backend: `go build`/`vet`/`test` für work + gateway + models **seriell** (kein `./...` → OOM). Migr.231 lokal anwenden: `docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml up --build migrate`.
- FE: scoped `tsc --noEmit` (Default-tsconfig, grep auf eigene Dateien wg. ~98 vorbestehender i18n-Fehler) + **voller `eslint src/`** (unused-imports=error, prefer-const=error) + `npx vitest run`.
- **Playwright-Screenshots work-Modul WIRKLICH ansehen** — der 'Normal'-Filter muss **28 statt 12 Tasks** zeigen (das ist der eigentliche Akzeptanztest). 1 Commit. **Push erst nach Review-Gate** (User pausiert pro Welle).

## Disziplin (verbindlich — MEMORY.md)
- Max 3 Subagenten, Sonnet, self-contained Prompts. `isolation:"worktree"` Pflicht für code-schreibende Agenten („do not push"-Text ist KEINE Grenze; Main committet ALLES, reconciled am Gate; `git worktree list` clean vor Commit).
- Worktree-Agenten haben kein `node_modules` → **FE-Gates NUR in Main** auf gemergtem Tree. Go+protoc laufen im Worktree (protoc-Pfad: `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/Google.Protobuf_*/bin/protoc.exe`; Proto-Gen via `make`-äquivalentem `protoc … --go_out … --go-grpc_out …` da `make` in Git Bash fehlt).
- **„Grün≠korrekt":** Agent-Claims unabhängig prüfen (keine Unimplemented-Stubs, gRPC-Schicht statt Direct-Svc, Proto regeneriert, keine ErrorBoundary). **Screenshots ansehen.**
- Gate-Kommandos nie durch Pipes (`| head` maskiert Exit). openapi-Gate = `npx @apidevtools/swagger-cli validate backend/api/openapi.yaml`.
- Git: Conventional Commits, English imperative, **KEINE AI-Attribution**. **Vor Push: `git fetch` + `git rebase origin/main`** (Darien aktiv). CD-Smoke-Fail → forward-fix, NIE revert (Migr-Drift).
- Nach Push: `gh run view --json jobs` (Step-Counts, nicht Logs). CD braucht ~18 min; warten bis grün, **damit Darien nicht auf rotem main sitzt.**

## Lokales QA-Harness (nützlich, wiederverwendbar)
- **localbackend Screenshot-QA gegen echtes Backend:** `scripts/qa-mock-exit-w4.mjs` ist das Muster — loggt per `fetch` ein und **injiziert den Token via `electronAPI.auth.getStoredTokens`-Stub** (App zeigt sonst Dashboard ohne Auth → 401 überall). Für Welle 5 ein analoges `qa-priority-sweep.mjs` schreiben (work-Modul, 'Normal'-Filter-Count prüfen).
- **Login:** `demo@local.test` / **`DemoPass123!`** (NICHT `Demo1234!` — die alten qa-*.mjs-Skripte haben das falsche Passwort).
- Dev-Server: `cd desktop && npx electron-vite dev --mode localbackend` (→ :5173, `.env.localbackend` mit `RENDERER_VITE_DEMO_MODE=false` = MSW aus, API→:8080).
- Modul-Flags lokal an (für Module hinter Feature-Flag): temporäre Compose-Override-Datei (nur Gateway, nicht committen): `services: gateway: environment: COSMI_MODULE_X_ENABLED: "true"` + `docker compose … -f <override> up -d --no-deps --force-recreate gateway`. (Für Welle 5 NICHT nötig — work ist nicht flag-gated.)
- Mutations brauchen `Idempotency-Key`-Header (lokal `IDEMPOTENCY_MODE=hard`).

## Welle 6 (nach Welle 5, falls fortgesetzt)
Nur **actor_name** (User hat ContractType + task-labels abgewählt): `models.EventPayload` um `ActorName string` erweitern, an `task/service.go` EmitTaskEvent-Sites befüllen, in `notification/service.go ProcessEvent` setzen (`cmd/notification/main.go extractSenderName` liest es bereits). Reines Backend, keine Migration (notifications.actor_name existiert seit Migr.229).

---

## START-PROMPT (in frische Session kopieren)

```
Wir setzen die Mock-Exit-Arbeit fort (frische Session). Lies ZUERST .planning/welle5-priority-sweep-resume.md (vollständiger Stand + Scope + Disziplin).

Stand: origin/main grün (CI/CI Desktop/CD ✓), Migrationskopf 230, nächste freie Migr = 231. Welle 4 (wiki+helpdesk mock-exit) ist erledigt+deployt — nicht anfassen. Darien arbeitet parallel auf main (security/HR/dialer) → vor jedem Push git fetch + rebase origin/main, seine Dateien nicht anfassen.

ZIEL Welle 5: priority-Sweep 'medium' → 'normal', VOLLER Backend-Sweep + FE. Reihenfolge: Migr.231 (tasks CHECK/DEFAULT/backfill + Migr.058-Template-Seed) → Backend (models/task.go + gateway-validate-tag + alle Refs) → FE (~48 Stellen, Liste im Resume-Doc). Gate: go build/vet/test seriell + Migr.231 lokal + scoped tsc + voller eslint + vitest + Playwright-Screenshot work-Modul (Akzeptanz: 'Normal'-Filter zeigt 28 statt 12 Tasks). Ein Commit, Push erst nach Review-Gate.

Ausführung: frische Session → Worktree-Delegation ist wieder sicher (1 Sonnet-Worktree-Agent für die mechanische Backend+FE-Sweep möglich; Main gatet FE + Playwright). Oder Main direkt — deine Wahl je nach Context. Disziplin-Gates + QA-Harness (qa-mock-exit-w4.mjs-Muster, Token-Injektion, DemoPass123!) stehen im Resume-Doc.

Pro Welle pausieren. Ultrathink für die Migr.231-Template-Seed-Frage.
```
