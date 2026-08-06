# Mock-Exit Master-Resume — Subagent-Wellen bis Cosmi 1.0 (neues Fenster)

> **Zweck:** Ein neues Fenster orchestriert in **Subagent-Wellen** die restliche Mock-Exit-/Echt-Schaltungs-Arbeit
> (MASTER-PLAN §6 „Welle 1" + doable Cross-Cutting), bis alle backend-fähigen Module echt verkabelt + die bekannten
> Bugs zu sind. **Autoritativer Plan:** `.planning/MASTER-PLAN.md` (§6 = Wellen-Queue, §2/§3 = Modul-Status,
> §4 = Cross-Cutting/Bugs, §5 = Luke-Backend-Track). Dieses Doc bündelt **Stand + Disziplin + Operatives + Pattern**,
> damit die neue Session sofort produktiv ist.
> **Stand:** 2026-06-24 (nach Welle A+B). `origin/main` grün (CI ✓ / CD „Deploy to Production: success"), Prod-Migrationskopf **232**, nächste freie Migr = **233**. **NÄCHSTE WELLE = C (X-3 OpenAPI voller Sweep)** — Details unten §3 + Plan `~/.claude/plans/mock-exit-master-resume-federated-moonbeam.md`.

---

## 1 · DONE — NICHT erneut anfassen (echt-geschaltet/gefixt + deployt)

**Echt-verkabelt (Welle 1, live verifiziert):** kontakte · crm (companies/deals/pipeline-stages/tags) · work
(Tasks/Projekte/Kommentare/Aktivitäten/Dateien/Zeit) · finanzen/Buchhaltung (Dashboard-KPIs + Listen) ·
dialer-Supervisor · dashboard-Layout · zeiterfassung/HR (weitgehend) · notifications · vertraege · work-Labels ·
settings (Persistenz teils) · **wiki + helpdesk** (Welle 4).

**Diese Session zusätzlich (gepusht + CD-deployt):**
- **Welle 4** `61a6b96e`..`0b740522` — wiki+helpdesk mock-exit (tenant_id-INSERT-Bug, ListShareTokens, content-base64-Adapter, Timestamp-Normalisierung, 8 unwrap-Fixes, OpenAPI 16 Routen).
- **Welle 5** `c84a7c72` — **Task-Priority `medium→normal` voller Stack** (Migr.231: CHECK/DEFAULT/Backfill + `industry_templates`-JSONB; Backend models/gateway-validate/automation; ~20 FE-Dateien; `toApiPriority`-Shim entfernt). i18n unberührt (`work.priority.normal` existierte bereits).
- **Welle 6** `c5dfd7c7` — **`actor_name` in Task-Notifications** (EventPayload.ActorName, `Repository.GetUserDisplayName`, 4 Emit-Sites, ProcessEvent persistiert, extractSenderName-Fallback). Keine Migration (Spalte seit Migr.229).

**PROMPT A/C (Vor-Sessions):** OpenAPI-Drift-Resync notification/dialer (`ae731ca1`) · Sammel-Push 9 Commits (`07ad5164`).

**Session 2026-06-24 #2 — Welle A + Welle B (alle gepusht + CD-deployt auf Prod):**
- **Welle A** `3f8c444b` — calendar wire-time (`normalizeWireTimestamps` im calendar-client; Array-Params + Booking-Public waren schon ok) · de.json Mojibake B-3 (nur **2** Em-Dash-Stellen, nicht ~90) · **B-12 = Dead-Code:** toter `modules/buchhaltung/`-Ordner entfernt (0,00€-Bug war nur in der unerreichbaren Seite; FinanzenPage korrekt). **B-1 war bereits grün** (ChatFlow 12/12 — „7/12 rot" war stale). **HR hands-off** (FE schon echt, `entries:null` via `?? []`; Darien-Lane, nicht angefasst).
- **Welle B1** `ea89d70e` — **work `DELETE /projects/{id}`** volle RPC-Kette (proto+server+repo+gateway, `projects:delete` geseedet, FK-Cascade) + FE `useDeleteProject`+Lösch-Button. Live: create→delete(200)→get(404).
- **Welle B3** `ea89d70e` — **helpdesk BusinessHours-BE** (Migr.232 `helpdesk_business_hours`, tenant_id+RLS, Permission reused) + Get/Update-RPCs + Gateway-Routen + 5 FE-Store-Reste→API (BusinessHoursDialog, OpenTickets-Widget, useAlerts, KB-Body-Edit). Live: GET→PUT→GET persistiert. **CSAT bleibt Store** (kein BE-Endpoint).
- **Welle B2** `2b0845fc` — **dokumente Upload** via Presign-Flow (`RegisterUploadedFile`-RPC + service.Register + Gateway `POST /documents/files`; FE `useDocumentUpload` = presign→PUT→register) + 6 Client-Pfad-Drifts gefixt (init-user/team, download-url, tags→JSON, wopi→JSON, version-revert→JSON). **Byte-PUT braucht öffentlichen MinIO-Endpoint+CORS** (Infra-Punkt, separat).
- **CI-Fix** `945a4bd8` — `mockProjectRepo.Delete` im task-Test (Interface-Change brach paketübergreifenden typecheck; erster Push war CI-rot → forward-fix, CD war korrekt skipped).
- **Gaps offen (kein BE):** CSAT-Persistenz · doc Activity-Feed + Create-Version · share/link-Delete-by-id (Gateway will entity+user) · MinIO-Public-Endpoint für Live-Byte-Upload.
- **⚠ Operativ:** `isolation:"worktree"` isolierte NICHT (beide Subagenten im Haupt-Tree) — gerettet durch disjunkte Datei-Lanes + Main-Reconcile. Konkurrierende protoc-Regen clobbert sich (Stray-Dir). Vor Push IMMER golangci-lint übers ganze geänderte Modul (nicht nur scoped go test). Siehe Memory `feedback_subagent_for_large_waves.md`.

---

## 2 · REST-SCOPE — was das neue Fenster abarbeitet

> Quelle: MASTER-PLAN §2–§4. **Drei Kategorien strikt trennen** („FE-mock-fertig ≠ backend-fertig"):

### A) JETZT machbar (Backend existiert → echt-schalten / Bug fixen) — **Subagent-Wellen-Ziel**
- **Echt-Schaltung-Reste 🔌:** calendar (verify + Scoped-Delete Serien + Booking-Public-Seite) · zeiterfassung-Rest (`entries:null`-Handling, HR-Seed) · dokumente (MinIO-Upload verifizieren) · settings-Persistenz **X-4** (localStorage→Backend, BE Migr.138 da) · helpdesk-Reste (DeleteSLAPolicy, KB/Stats/Routing falls noch Store).
- **Cross-Cutting machbar:** **X-3** OpenAPI-Specs nachholen für formulare/dialer/inventar/vermietung/vertraege/mails · **X-5** Demo-Seeds pro Modul (Muster `backend/seeds/demo/*.sql`) · **X-6** echter Build (Modell A) für Review.
- **Bugs machbar:** **B-1** ci-desktop rot (`ChatFlow.test.tsx` 7/12) · **B-3** Mojibake `de.json` (~90 latin1→utf8) · **B-4** OpenAPI-Drift (überlappt X-3) · **B-7** `DELETE /projects/{id}` fehlt (work) · **B-12** finanzen invoice-**List** liefert kein `gross_total` (Betrag 0,00 €) · **B-11** contacts Timeline-Endpoint + Spec-Drift (Teil X-3).
- **Demo-Tiefe-Phasen** der gebauten Module (notifications/formulare/dialer/video) + Tiefe-Re-Checks T-1..T-4 (kontakte/calendar/dokumente/zeiterfassung).

### B) 🔒 BLOCKIERT (Backend fehlt real = Luke-Track §5) — **NICHT bauen, nur mock-first/swap-ready lassen**
mails (IMAP/SMTP) · security/DSGVO-Echt-Schaltung (FE ist mock-fertig; BE teils da, Wire-Shapes prüfen — **P0-Launch-Blocker**) · automatisierung-Engine (CRUD/Execution/Webhook) · kommunikation-Inbox · profil (Avatar/Preferences/Presence) · berichte Server-PDF/Cron · E-Rechnung/DATEV · **X-1** S3/MinIO-Upload-Service · **X-2** Signatur-Service · Auth-Invite (team) · KPI-Endpoint (dashboard) · Branchen-Feature-Endpoints.

### C) 📱 POST-1.0 (Handy-App-Phase) — **gar nicht einplanen**
GPS-Stempel · Offline-Rapporte/Queue · Barcode/QR-Scan · mobile Self-Service-Portale.

> **Goldene Regel (Darien):** Pro Phase prüfen ob der Endpoint existiert. **Ja → direkt echt anhängen (🔌).
> Nein → mock-first + Eintrag in `backend-gaps.md` + 🔒-„verdrahten"-Zeile im Plan.** Nie blind weiter mocken.

---

## 3 · Wellen-Plan (Subagent-orchestriert)

> Folgt MASTER-PLAN §6. **Pro Welle pausieren** (Review-Gate für Darien). **Disjunkte Modul-Lanes** wählen (keine
> Hot-File-Kollision — siehe §5). Subagenten bauen in **Worktree-Isolation**, **Main reconciled + gatet + committet + pusht ALLES.**

- **Welle A — Echt-Schaltung-Reste + Bugs** (gut parallel, 2–3 disjunkte Lanes): calendar ∥ zeiterfassung-Rest ∥ dokumente; dazu Main-Lane Cross-Cutting **X-4 Settings-Backend** (Hot-File → NICHT doppeln). Bugs B-7/B-12 (work/finanzen) als eigene kleine Lanes. **B-1/B-3** zuerst (grünes CI-Fundament).
- **Welle B — Cross-Cutting Fundament** (Main-lastig): **X-3** OpenAPI-Specs (6 Module, `openapi.yaml` = Hot-File → Main-only, kein paralleler Agent) → `npm run api:generate` → scoped tsc. **X-5** Demo-Seeds (Main-Lane, FK-Reihenfolge, fixe `tenant_id …0001`).
- **Welle C — Demo-Tiefe + Lücken-wo-BE-da** (parallel): Demo-Tiefe-Phasen + Tiefe-Re-Checks T-1..T-4 + admin-Tiefe-Verify + settings-Rest-Phasen (nur die ohne 🔒).
- **Welle D (optional, wenn Luke-BE landet):** Echt-Schaltung der dann-entsperrten 🔒-Module (security/DSGVO zuerst — P0). Wire-Shape gegen echtes BE prüfen, nicht gegen Mock.
- **Welle 6 (Review)** = manueller Team-Block am Ende (NICHT Subagent) — an der fast-fertigen Version.

**Subagent-Einheit:** ein **disjunktes Modul** (oder ein isolierter Bug) pro Worktree-Agent. Max **3 gleichzeitig**, Sonnet-Baseline.
Per-Modul-Echt-Schaltung ist eine gute Subagent-Einheit; Cross-Cutting (X-3/X-4/X-5) bleibt **Main-Lane** (Hot-Files).

---

## 4 · DISZIPLIN (verbindlich — aus MEMORY.md + dieser Session hart erkauft)

**Subagenten:**
- **`isolation:"worktree"` PFLICHT** für code-schreibende Agenten. „do not push"-Text ist **KEINE Grenze** — Subagenten ignorieren ihn (dokumentiert: 2/3 pushten in Welle 2 trotz Verbot → CI rot + Stub-CD nach Prod). **Main committet + pusht ALLES.**
- **„Grün ≠ korrekt":** Agent-Claims unabhängig gegenprüfen — keine `Unimplemented`-Stubs, gRPC-Schicht statt Direct-Svc (RLS-Bypass/Phantom-404), Proto regeneriert, keine ErrorBoundary-Crashes. **Screenshots WIRKLICH ansehen.**
- **Worktree-Agenten haben kein `node_modules`** → **FE-Gates (tsc/eslint/vitest) NUR in Main** auf dem gemergten Tree. Go+protoc laufen im Worktree (protoc-Pfad in §6).
- Subagenten können **nicht nachfragen** → Prompt self-contained (Pfade, Constraints, Akzeptanzkriterien).
- **Background-Agent-Duplikate:** Harness kann pro Launch Duplikate spawnen, teils direkt im Haupt-Tree → vor Commit `git worktree list` + Commit-Dateiliste vs. Report abgleichen, explizit stagen.

**Gates (nie durch Pipes — `| head`/`| tail` maskiert Exit):**
- **Backend seriell** (kein `./...` → OOM): `go build`/`vet`/`test` der betroffenen Pakete + relevante `cmd/*`.
- **FE in Main:** scoped `tsc --noEmit` (Default-tsconfig, grep auf eigene Dateien wg. ~98 vorbestehender i18n-Fehler) + **voller `eslint src/`** (`unused-imports/no-unused-imports`=error, `prefer-const`=error) + `npx vitest run`. **Gate nur auf finalem Tree** (nach `stash`/`pop`/`checkout` neu laufen).
- **OpenAPI:** `npx @apidevtools/swagger-cli validate backend/api/openapi.yaml`.
- **Playwright-Screenshot-QA** ansehen (Raw-Keys/Emojis/Layout/leere Zustände/`{seconds,nanos}`/`Invalid Date`).

**Migrations (forward-only):**
- Kopf **231**, nächste **232**. NIE alte Migration editieren. Neue Tabelle ⇒ `tenant_id NOT NULL` + RLS-Policy (oder System-Global-Liste ADR-006) + **Permission-Seed** falls neuer `RequirePermission`-Guard (sonst 403 für ALLE inkl. Admin).
- **CHECK-Swap-Reihenfolge** (Welle-5-Lehre): alten CHECK **zuerst** droppen → backfillen → neuen CHECK setzen (sonst scheitert der Backfill am alten CHECK).
- **Neue `config.RequireX`-Assertion = Prod-Deploy-Hazard:** erst Compose-Passthrough (`${VAR:-}`) + `.env.production`-Wert, DANN Assertion-Commit. **Revert-nach-Deploy = Migrations-Drift** → forward-only fixen, NIE `git revert` eines deployten Migr-Commits.
- **PS 5.1 `Set-Content` = BOM/Mojibake** → Repo-Text nur via Edit-Tool oder `[System.IO.File]` mit `UTF8Encoding($false)`.

**Git:** Conventional Commits (`feat|fix|docs|refactor|test|chore`), English imperative, **KEINE AI-Attribution**.
**Vor Push: `git fetch` + `git rebase origin/main`** (Darien arbeitet parallel auf main — security/HR/dialer; seine
Dateien NICHT anfassen). **Nach Push CD ~18 min beobachten** via `gh run view <id> --json jobs` (Step-Counts, nicht
Logs — 0-Step-Fails = Billing-Wall; `gh run watch` bricht oft vorzeitig ab → Poll-Loop nutzen), bis grün — **damit
Darien nicht auf rotem main sitzt.** Backend-only-Push triggert kein CI-Desktop (path-gefiltert) — normal.

---

## 5 · Echt-Schalt-Pattern + Wire-Shape-Fallen (kontakte = Referenz)

**Pattern (`.planning/kontakte-mock-exit-DONE.md`):**
1. **Casing nur für OpenAPI-getippte Module:** `api/casing.ts` `dual(obj,'camelName')` liest beide Casings (Gateway snake_case ↔ OpenAPI camelCase). Handgetippte snake_case-Clients matchen schon — aber trotzdem Methode/Shape/Idempotency/RBAC prüfen.
2. **Write mode-branched:** `DEMO_MODE` aus `mocks/demo-mode-flag.ts` (Leaf) — Mock voller Feldumfang, echtes Backend nur backend-konforme Felder.
3. **Wire-Shape gegen ECHTES Backend prüfen**, nicht Mock.

**Drei Stolpersteine (gelten für JEDEN Swap):**
- **Auth/Idempotency:** Mutations brauchen JWT. `IDEMPOTENCY_MODE: hard` (Dev) → 400 ohne `Idempotency-Key`-Header — **aber nur an wenigen Endpoints hart erzwungen** (verifiziert: `POST /hr/time/entries`) → pro Mutation einzeln prüfen.
- **Wire-Shape ≠ flacher Mock:** `response.JSON` → **Timestamps `{seconds,nanos}`** (nicht RFC3339); `response.Proto` (nur **dialer**) → **Enums als Integer**, int64 als String. Flache Mocks maskieren beides. → `api/wire-time.ts normalizeWireTimestamps` am Hook/Client; nested Sub-Message unwrappen (`{calendar:{…}}`, `{labels:[…]}`).
- **RLS/tenant_id (fail-closed):** Endpoint ohne durchgereichtes tenant_id → **Phantom-404/leere Liste** statt Fehler. Pfad: JWT `tid` → Gateway-Middleware → gRPC `x-tenant-id` → `set_config('app.tenant_id',…)` → RLS-Policy. **NULLABLE-tenant_id-Audit vor jeder RLS-Welle:** Schema-NOT-NULL **+** Repo-INSERT-Wiring **+** SELECT-Scan (Read-Seite → Phantom-404!).

**Häufige mock-verdeckte Bugs:** Update-Methode **PUT statt PATCH** (405) · Feldname-Drift (`position`≠`title`, `domain`→`website`) · `custom_fields` Array `[{field_id,value}]` statt Objekt (400) · RBAC `member` ohne `*:write` (403 → Demo-User=admin) · proto3 lässt leere Felder weg (FE `?? []`).

---

## 6 · OPERATIVES (Login, Server, Harness, Pfade)

**Lokaler Stack** (läuft ggf. noch — `docker ps`; sonst neu hoch):
```
docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml up -d --no-deps gateway auth crm notification dialer work vertraege wiki helpdesk biz
```
Gateway :8080. Postgres/Redis/MinIO separat falls weg. **Migration lokal anwenden:** `docker compose … up --build migrate`.
**psql lokal:** `docker exec <postgres> psql -U kmuhub -d kmuhub -tAc "…"`.

**Dev-Server (localbackend, MSW aus, API→:8080):** `cd desktop && npx electron-vite dev --mode localbackend` → :5173
(`.env.localbackend`, `RENDERER_VITE_DEMO_MODE=false`, gitignored). HMR greift Disk-Änderungen; `page.goto` lädt frisch.

**Login (KORREKT):** `demo@local.test` / **`DemoPass123!`** — ⚠ NICHT `Demo1234!` (alte qa-*.mjs-Skripte + kontakte-DONE.md
haben das falsche Passwort). Demo-User ist **admin** (sonst 403 bei Mutationen). Users lokal: Demo User
`1396243a-…`, Admin User `cc181dac-…`, tenant `00000000-0000-0000-0000-000000000001`.

**QA-Harness (Token-Injektion gegen echtes Backend):** Muster `desktop/scripts/qa-mock-exit-w4.mjs` /
`qa-priority-sweep.mjs` — per `fetch` einloggen, Token via `electronAPI.auth.getStoredTokens`-**Stub** injizieren
(App zeigt sonst Dashboard ohne Auth → 401 überall), `ELECTRON_STUB` + `SUPPRESS_ONBOARDING` als `addInitScript`.
Mutations brauchen `Idempotency-Key`-Header. Screenshots in `desktop/.qa-screenshots/` (gitignored) — **ansehen via Read-Tool.**
**Modul-Flags lokal an** (falls flag-gated): temporäre Compose-Override (nur Gateway, nicht committen)
`environment: COSMI_MODULE_X_ENABLED: "true"` + `up -d --no-deps --force-recreate gateway`.

**Event→Notification e2e** (Welle-6-Muster, ohne FE): `SELECT pg_notify('events', json_build_object('type','…','tenant_id','…','target_user_ids',json_build_array('…'),…)::text)` → notification-Consumer verarbeitet.

**Env-Pfade:** Go 1.25.6 `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"` · protoc
`C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/Google.Protobuf_*/bin/protoc.exe` (Proto-Gen via `protoc … --go_out … --go-grpc_out …`, `make` fehlt in Git Bash) · gh `"C:/Program Files/GitHub CLI/gh.exe"` · golangci-lint v2.8, `backend/.golangci.yml`.

**Prod (Hetzner, nur mit expliziter User-Freigabe pro Session):** `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`,
`.env.production` in `/opt/kmuhub/`, psql `-U kmuhub -d kmuhub`. **Direkte Prod-Reads werden vom Auto-Mode-Classifier
geblockt, wenn der User sie diese Session nicht benannt hat** → CD-`gh`-Monitoring + Post-Deploy-Smoke reichen als Beleg.
Deploy macht Luke / CD-Auto. CD-Smoke-Fail → forward-fix, NIE revert.

---

## 7 · START-PROMPT (in neues Fenster kopieren)

```
Wir setzen die Mock-Exit-/Echt-Schaltungs-Arbeit fort (frisches Fenster). Lies ZUERST
.planning/mock-exit-master-resume.md (Stand + Rest-Scope + Disziplin + Operatives + Pattern) und
.planning/MASTER-PLAN.md §6 (autoritative Wellen-Queue) + §2-§4 (Modul-Status/Cross-Cutting/Bugs).

Stand: origin/main grün (CI/CD ✓), Prod-Migrationskopf 231, nächste freie Migr = 232. Welle 4/5/6
(wiki+helpdesk, Task-Priority medium→normal, actor_name) sind erledigt+deployt — nicht anfassen.
Darien arbeitet parallel auf main (security/HR/dialer) → vor jedem Push git fetch + rebase origin/main,
seine Dateien nicht anfassen.

ZIEL: die restliche backend-fähige Mock-Exit-Arbeit in Subagent-Wellen fertigstellen — NUR Kategorie A
(Backend existiert → echt-schalten / Bug fixen), Kategorie B (🔒 Luke-Backend) + C (📱 Post-1.0) auslassen.
Reihenfolge laut Resume §3: Welle A (Echt-Schaltung-Reste calendar/zeiterfassung/dokumente + X-4 Settings-
Backend + Bugs B-1/B-3/B-7/B-12) → Welle B (X-3 OpenAPI-Specs + X-5 Seeds, Main-only) → Welle C (Demo-Tiefe +
Tiefe-Re-Checks). Pro Welle disjunkte Modul-Lanes, max 3 Subagenten in Worktree-Isolation, Main reconciled +
gatet (FE-Gates NUR in Main, kein node_modules im Worktree) + committet + pusht ALLES. Pro Welle pausieren.

Vorgehen je Welle: Ist-Abgleich (Endpoint existiert? Wire-Shape gegen ECHTES Backend, nicht Mock) →
Klärungsfragen an mich → Subagent-Pakete (self-contained, Worktree) → Main-Gate (go build/vet/test seriell +
scoped tsc + voller eslint + vitest + Playwright-Screenshots ANSEHEN) → ein Commit + Push nach Review-Gate →
CD beobachten bis grün. Disziplin („Grün≠korrekt", Subagent-Push-Bypass, RLS-Phantom-404, PUT≠PATCH,
{seconds,nanos}, Idempotency, Permission-Seed, forward-only-Migrations) steht im Resume-Doc.

Nutze ultrathink für die Wellen-Schnittplanung (disjunkte Lanes ohne Hot-File-Kollision) und für jede
Migration-/RLS-Frage.
```

---

## 8 · Schnell-Referenz Dokumente
| Doc | Inhalt |
|---|---|
| `MASTER-PLAN.md` | **Autoritativ.** §6 Wellen-Queue · §2/§3 Modul-Status · §4 Cross-Cutting/Bugs · §5 Luke-Track |
| `mock-exit-readiness-matrix.md` | Pro-Modul Swap-Map (Backend/Wire-Shape/Auth/Idempotency/RLS/Aufwand) — Stand 06-23, mit DONE-Liste oben abgleichen |
| `kontakte-mock-exit-DONE.md` | **Echt-Schalt-Referenz-Pattern** + camelCase-Risiko-Set + Backend-Handover |
| `mock-exit-resume-2026-06-24.md` | Vorsession-Stand (PROMPT A/B/C Details) |
| `welle5-priority-sweep-resume.md` | Welle-5-Detail (Migr.231-Lehren, QA-Harness-Muster) |
| `backend-gaps.md` / `BACKEND-PLAN.md` | Luke-Backend-TODOs (🔒-Track) |
| `collision-map.md` · `parallel-batch/` · `two-terminal-nico-workflow.md` | Lane-/Hot-File-/Paket-Muster für Parallelarbeit |
| `nico-block/WORKFLOW.md` | Build-+-Verify-Standard (kopierbare Vorlagen) |
