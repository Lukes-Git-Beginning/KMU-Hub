# Mock-Exit — Resume-Prompts für nächste Session (Stand 2026-06-24)

## Gemeinsamer Stand (gilt für alle drei Prompts)

- **✅ PROMPT C ERLEDIGT (2026-06-24):** Alle 9 Commits + 1 Lint-Fixup (`07ad5164` `fix(notifications): drop redundant boolean cast`) **gepusht** → origin/main = `07ad5164`. CI + CI Desktop + **CD grün/deployt**. Prod: healthy, commit `07ad5164`, **Migrationskopf 230 (dirty=f)**. Modul-Stichprobe: dialer/supervisor **200** (nicht 403 → Permission-Seed wirkt), work/labels/settings/notifications je 200, pin-Route 404 auf Dummy (nicht 403/405). Leerer Tenant → work/labels liefert `{}` (proto3 lässt leeres `labels` weg; FE defaultet `?? []`).
- **✅ PROMPT A ERLEDIGT (2026-06-24):** `ae731ca1` `fix(openapi): resync notification/dialer spec + align FE types` **gepusht + CI/CI-Desktop/CD grün, CD-deployt** (origin/main = `ae731ca1`, Prod healthy commit `ae731ca1`, Kopf bleibt 230 — keine neue Migration). Inhalt: Notification is_pinned/is_dismissed/actor_name + pin/dismiss-Pfade + Int-Drift-Doku; Dialer supervisor/contact-calls Pfade+Schemas; view_type `gantt`; AcceptInvitation 410→409; FE priority medium→normal (Boundary), Member-Display via first/last name, TaskWithDerived. Gates: swagger-validate ✓, tsc=0, eslint=0, vitest 475/475. **Befunde:** Settings+Work-Labels waren schon im Spec; **types.ts hatte 18.6k Z. pre-existing Drift** (seit 06-17 nicht regeneriert) → mit-geklärt. workqa 31→5 (Reste out-of-scope: lucide title, TaskTimer, CustomFieldsSection, useProjects is_default/is_closed). **VERTAGT:** volle priority `medium`→`normal` Demo-Vereinheitlichung (Mocks+Pills+PriorityBadge-Config+Fallbacks, ~12 Dateien, braucht Screenshot-QA) — Demo-'Normal'-Filter zeigt sonst 12 statt 28 Tasks (vorbestehende Mock-Inkonsistenz). Keine Playwright-Screenshots gefahren (Änderungen typ-ebenen + Member-Display-Verbesserung).
- **→ Nächster Schritt: Prompt B (wiki + helpdesk Full-Stack) — NEUE SESSION empfohlen (Worktree-Base-Falle: Worktree-Agenten branchen von Session-Start-HEAD).**
- **Branch:** main. ~~**9 lokale Commits ahead of origin, NICHT gepusht** (HEAD `e92f80d4`)~~ — erledigt, siehe oben. Ursprüngliche Commits:
  - `5c197532` feat(settings): persist user settings to backend
  - `96f1d07a` fix(hr): map snake_case wire shape for time endpoints
  - `6e194fd1` feat(vertraege): wire contracts to real backend
  - `e739dd25` feat(notifications): add pin/dismiss/actor_name + RPCs (Migr.229)
  - `fb045f9f` feat(dialer): add supervisor overview + contact-calls endpoints
  - `d591a30a` feat(notifications): wire pin/dismiss/actor_name FE
  - `a269c72c` feat(work): wire label taxonomy to real backend
  - `9995505f` fix(security): CORS PATCH, sanitize consent HTML, RLS child tables (Migr.230)
  - `e92f80d4` fix(notifications): normalize integer priority + wire timestamps (live-QA)
- **Migrationskopf:** Repo + lokale DB + **Prod = 230** (nach Prompt-C-Deploy; Prod war vor Push 228, CD applizierte 229+230 forward, dirty=f). 227 meeting_crm_link, 228 meeting_ai_summary, 229 notifications, 230 wiki/hr-child-RLS. **Nächste freie Migr.-Nr = 231.**
- **Lokaler Stack:** evtl. noch up (`docker ps`). Falls down, neu starten:
  `docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml up -d --no-deps gateway auth crm notification dialer work vertraege wiki helpdesk biz`
  (Infra postgres/redis/minio separat hochfahren falls weg). Gateway :8080. Dev-Server FE: `cd desktop && npx electron-vite dev --mode localbackend` (→ :5173, `.env.localbackend` existiert, gitignored). Login Playwright-QA: Token via `getStoredTokens`-Stub injizieren (Muster in `desktop/scripts/qa-b12-finanzen-amounts.mjs`). Demo-User `demo@local.test` / `DemoPass123!` (lokal registriert + auf admin gehoben).
- **Disziplin (verbindlich):** Plan-Datei `~/.claude/plans/purrfect-frolicking-neumann.md` + MEMORY.md. Kurz: max 3 Subagenten, `isolation:"worktree"` Pflicht, Worktree-Agenten haben kein `node_modules` (Main gatet gemergten Tree), Hot-Files nur Main (`mocks/handlers/index.ts`, `i18n/messages/*.json`, `backend/api/openapi.yaml`), „Grün≠korrekt" → unabhängig gegenprüfen + Screenshots ansehen.
- **⚠ Worktree-Base-Falle (heute aufgetreten):** Worktree-Agenten branchen vom Commit, der bei Session-Start HEAD war — NICHT von Commits, die *innerhalb derselben Session* gemacht wurden. In neuer Session = HEAD inkl. heutiger Commits, also ok SOLANGE vorher committet/gepusht. Trotzdem: bereits-gemergten Stand explizit im Agent-Prompt nennen; Main reconciled am Gate.
- **Offene Follow-ups:** actor_name Emit-Seite befüllt nicht (FE zeigt null); vertraege lokal Feature-Flag-gated (404 erwartet); ContractType-Mapping verlustbehaftet (6→5); task-label-Assignments noch Store-basiert (echte Task-UUIDs nötig).

---

## PROMPT C — Sammel-Push (EMPFOHLEN ZUERST)

```
Wir setzen die Mock-Exit-Arbeit fort. Lies zuerst .planning/mock-exit-resume-2026-06-24.md (gemeinsamer Stand). Ziel dieser Session: die 9 lokalen Commits (HEAD e92f80d4) kontrolliert nach origin/main pushen und den CD-Deploy verifizieren.

Vorgehen:
1. Final-Gate auf finalem Tree: scoped `tsc --noEmit` (Default-tsconfig, nur geänderte Dateien) + eslint (unused-imports=error) + `npx vitest run` im desktop/. Backend: `go build`/`go vet` der betroffenen Pakete (notification, dialer, gateway, middleware) seriell (kein ./... → OOM).
2. Migrations-Sequenz prüfen: backend/migrations lückenlos bis 230, forward-only. `git worktree list` muss leer sein.
3. Prod-Migrationskopf VOR Push feststellen (ssh + psql -U kmuhub -d kmuhub -tAc "SELECT version FROM schema_migrations" am Prod-Host 178.104.38.195) → bestätigen, dass 227–230 sauber forward angewendet werden (keine Drift).
4. `git push origin main` (ein Push). KEIN Force, KEIN Revert-nach-Deploy (= Migrations-Drift).
5. CD beobachten: `gh run watch` bzw. `gh run view --json jobs` (Step-Counts, NICHT Logs — 0-Step-Fails = Billing-Wall). Auf auth/notification/dialer-Crash-Loops achten.
6. Nach Deploy: Prod-Smoke (24/24) + Stichprobe der echt-geschalteten Module gegen Prod (notifications pin/dismiss, work-labels, settings-Persistenz, zeiterfassung-Saldo).

Disziplin + Hazards: siehe Resume-Datei + MEMORY.md (deploy-only-Commit triggert kein CI → ggf. `gh workflow run CD`; compose-Secrets nur als ${VAR:-default}).
```

---

## PROMPT A — Welle 5: OpenAPI-Drift-Resync

```
Wir setzen die Mock-Exit-Arbeit fort. Lies zuerst .planning/mock-exit-resume-2026-06-24.md. Ziel: backend/api/openapi.yaml mit der Implementierung resynchronisieren (Hot-File → MAIN-ONLY, kein paralleler Agent darauf). Die heutigen Wellen haben die Spec-Schuld vergrößert.

Hinzufügen/fixen:
1. Notification-Schema: Felder is_pinned, is_dismissed, actor_name ergänzen. Endpoints POST/DELETE /api/v1/notifications/{id}/pin, POST /api/v1/notifications/{id}/dismiss. priority: Backend serialisiert via response.JSON als Integer-Enum (0-4), FE normalisiert zu low|normal|urgent|high — Spec auf die FE-Contract-Strings setzen + im Description-Feld den Int-Drift dokumentieren.
2. Dialer: GET /api/v1/dialer/supervisor (SupervisorOverview: agents[], recent_calls[], totals{}) + GET /api/v1/dialer/contacts/{id}/calls. Shapes aus backend/internal/server/dialer_grpc.go / proto/dialer.
3. Work-Labels: GET/POST/PUT/DELETE /api/v1/work/labels (Shape {labels:[{id,name,color}]}), PUT /api/v1/tasks/{id}/labels.
4. Settings: GET/PUT /api/v1/settings/{module_id}/user (entries[] / {settings:{...}}).
5. CORS: PATCH ist jetzt erlaubt (cors.go) — Patch-Operationen im Spec konsistent halten.
6. Bekannte Pre-Drifts (aus .planning/tsc-latent-type-errors.md): Member-Schema fehlt display_name/email; Task fehlt is_closed/created_by_name; Priority-Enum (medium vs low|normal|high|urgent); useProjects view_type 'gantt' fehlt im Enum (list|kanban); AcceptInvitation 410→409.

Vorgehen: openapi.yaml editieren → `cd desktop && npm run api:generate` (regeneriert src/renderer/src/api/types.ts) → scoped tsc auf betroffene Hooks. Achtung: Nach Regen kennt der Notification-Typ is_pinned/is_dismissed evtl. nativ → der Cast in api/hooks/useNotifications.ts (normalizeNotification) kann vereinfacht werden, aber NICHT brechen. CI openapi-validate muss grün bleiben. Optional 1 Explore-Agent für „implementierte Routen nicht im Spec"-Inventar, aber alle Edits macht Main.
```

---

## PROMPT B — Welle 4: wiki + helpdesk Full-Stack

```
Wir setzen die Mock-Exit-Arbeit fort. Lies zuerst .planning/mock-exit-resume-2026-06-24.md. Ziel: wiki + helpdesk echt-schalten (Backend-RPCs + FE-Wiring). 2 Worktree-Subagenten parallel (max 3), Main regeneriert Proto + gatet den gemergten Tree.

Migrationsnummer falls nötig: nächste freie = 231 (Kopf ist 230). wiki-Child-Tabellen haben seit Migr.230 tenant_id+RLS.

Agent D1 — wiki (Worktree, Go+FE): 5 fehlende RPCs implementieren — DeleteCategory, UpdateCategory, CreateShareToken, RevokeShareToken, GetVersion. Dateien: backend/proto/wiki/v1/wiki.proto, backend/internal/server/wiki_grpc.go, backend/internal/gateway/route_wiki.go, dann FE-Hooks desktop/src/renderer/src/api/hooks/useWiki.ts + wiki-Modul. Thick-Services/Thin-Handlers, tenant-scoped, über gRPC-Schicht (kein Direct-Svc). Demo-Handler in mocks/handlers/wiki.ts behalten+angleichen (NICHT entfernen — Standalone-Demo).

Agent D2 — helpdesk (Worktree, Go+FE): KB-Artikel (Tabelle helpdesk_kb_articles Migr.210), Stats, Routing-Rules (helpdesk_routing_rules Migr.211), DeleteSLAPolicy. Backend RPCs + Routes + FE useHelpdesk.ts/Helpdesk-Modul. Falls neue Tabelle/Spalte: Migr.231, tenant_id NOT NULL + RLS-Policy + Permission-Seed falls neuer RequirePermission-Guard.

Main-Gate je Welle: Proto regenerieren (protoc-Pfad in MEMORY.md), `go build`/`go vet` betroffene Pakete + cmd/wiki, cmd/helpdesk, cmd/gateway seriell, Migration lokal anwenden (docker compose up --build migrate), curl-Smoke der neuen Endpoints mit Idempotency-Key-Header (Mutations brauchen ihn!) + Admin-Token, dann FE-Gate (scoped tsc/eslint/vitest) + Playwright-Screenshots ansehen. Lokal committen pro Modul. Pause nach jeder Welle.

Wichtig: Agenten bekommen den Hinweis, dass notifications/dialer/work-labels bereits echt-geschaltet sind (heutige Commits) — nicht erneut anfassen.
```
