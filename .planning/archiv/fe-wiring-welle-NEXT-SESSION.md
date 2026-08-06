# START-PROMPT — FE↔Backend-Wiring-Welle (Welle 2 + 3)

> Diese Datei in der nächsten Session als Einstieg lesen. Sie ist self-contained.
> Plan-File mit Gesamtkontext: `~/.claude/plans/alright-zuletzt-haben-wir-composed-reddy.md`

## Mission

Module von Zustand-Mock-Stores auf echte API-Hooks umstellen — **voll end-to-end**: wo
Backend-RPCs fehlen, werden sie gebaut (Proto/gRPC/Service/Migration), und **pro Modul ein
MSW-Demo-Handler** (sonst zeigt der Demo-Modus Stuck-Loading). 10 Module in 3 Wellen,
**max. 4 parallele Sonnet-Subagenten**, **Pause + Review-Gate nach jeder Welle** (NICHT
pushen bis Luke OK gibt).

## Status (Stand 2026-06-18)

**Welle 1 DONE + gepusht** (`5457361c..33fc0d04` auf main): vermietung, wiki, helpdesk,
schichten — alle store→API + MSW-Handler + verifiziert (Screenshots zeigen echte Daten).
Schichten hat neues SwapRequest-Feature (Migr. 160/161). **Migrations-Kopf jetzt 161, nächste frei 162.**

**OFFEN:**
- **Welle 2 (Medium):** Rapporte · Inventar · profil/zeiterfassung-Tab
- **Welle 3 (High-Gap):** Produktion · Fuhrpark · Einkauf

## Pro-Modul-Lückenkatalog (zu auditieren → Fall A nur-FE / Fall B Backend bauen)

| Modul | Page (Zeilen) | Store | Hook | Lücken (ggf. Backend bauen) |
|---|---|---|---|---|
| **Rapporte** | modules/rapporte/RapportePage.tsx (1744) | stores/rapporte.ts | api/hooks/useRapporte.ts | **Measurements/Aufmaß** + **Report-Templates** fehlen im Hook. PDF-Gen ist text/plain-Stub (backend rapporte_grpc.go:519) |
| **Inventar** | modules/inventar/InventarPage.tsx (1453) | stores/inventar.ts | api/hooks/useInventar.ts | **Lagerorte/Locations** + **Inventur-Sessions** (Jahresinventur) fehlen |
| **profil/zeiterfassung-Tab** | modules/profil/tabs/zeiterfassung/ (9 Sub-Views) | stores/timetracking.ts (661) | **hr-hooks.ts + useTimeEntries.ts** (existieren!) | **RECONCILE** mit existierenden Hooks, KEIN Parallel-Backend. MOCK_PROJECTS/MOCK_TASKS ersetzen. Categories/Templates/GPS evtl. im hr-Backend nachbauen |
| **Produktion** | modules/produktion/ProduktionPage.tsx (1466) | stores/produktion.ts | api/hooks/useProduktion.ts | **BOM** + **WorkSteps** + **Machines** + **QualityChecks** (4 Domänen) fehlen |
| **Fuhrpark** | modules/fuhrpark/FuhrparkPage.tsx (2046) | stores/fuhrpark.ts | api/hooks/useFuhrpark.ts | **Tankbuch/Fuel** + **Fahrtenbuch/Logbook** + **Dokumente** + **GPS-Tracking** (4 Domänen) fehlen |
| **Einkauf** | modules/einkauf/EinkaufPage.tsx (1724) | stores/einkauf.ts | api/hooks/useEinkauf.ts | **Katalog** + **Lieferantenbewertung/Ratings** + **Rahmenverträge** fehlen |

Backend liegt unter `backend/internal/<modul>/`, `backend/internal/server/<modul>_grpc.go`,
`backend/proto/<modul>/v1/<modul>.proto`, `backend/internal/gateway/route_<modul>.go`.

## Per-Modul-Subagent-Protokoll (Sonnet, ein Modul je Agent) — mit Welle-1-Lessons

**Phase 0 — Backend-Audit:** je Lücke grep ob RPC (proto + grpc + service) + Tabelle existieren.
Fall A (existiert → nur Hook/FE) vs Fall B (bauen).

**Phase 1 — Bauen:**
- **Backend (Fall B):** Migration MANUELL als `backend/migrations/0001NN_<name>.up.sql`+`.down.sql`
  mit zugewiesener Nummer (NICHT `make migrate-create` — Parallel-Kollision). `tenant_id` NOT NULL +
  `CALL enable_tenant_rls(...)` (Muster Migr. 000142) + idempotent. Neue Permission → Seed-Migration
  (sonst 403 für ALLE inkl. Admin). Proto-RPC ergänzen → **regen NUR dies Modul** (protoc-Zeile aus
  backend/Makefile `proto:`-Target, NICHT globales `make proto`; protoc 33.4 + Plugins in
  `/c/Users/Luke/go/bin`, also `export PATH=$PATH:/c/Users/Luke/go/bin`). **NIEMALS generierte
  `.pb.go` von Hand editieren.** gRPC-Server-Methode (thick service / thin handler, slog,
  Idempotency-Key für Create). Gateway-Route über **`sr.getClient()` (gRPC-Client), NIEMALS direkten
  DB-Service im Gateway** (Welle-1-Bug: schichten `sr.svc` war nil → Runtime-Fail).
- **FE:** Page store→Hook (Referenz: modules/dialer/CampaignListPage.tsx + useDialer.ts; Hybrid mit
  UI-Prefs-Store: modules/dokumente). `data?.x ?? []`, Loading→Skeleton, Empty→EmptyState,
  Error→Banner, ErrorBoundary-fest. Nested-Proto-JSON am Hook flach entpacken.
- **i18n FRAGMENT:** neue Keys in `src/renderer/src/i18n/_wave-fragments/<modul>.json`
  Form `{"de":{"<ns>":{...}},"en":{...},"fr":{...},"it":{...}}`. `{var}` nicht `{{var}}`, **ICU-Plural**
  `{count, plural, one {# x} other {# x}}` NICHT `_one`/`_other`. Umlaute korrekt (kein „fuer"/„ae").
  NICHT in die großen JSONs schreiben (Main merged).
- **MSW-DEMO-HANDLER (PFLICHT! Welle-1-Lesson):** `src/renderer/src/mocks/handlers/<modul>.ts`,
  `export const <modul>Handlers = [...]`. Muster: mocks/handlers/dialer.ts (msw `http.get/post`,
  `API_BASE_URL` aus `@/lib/constants`, shared-ids, date-helpers). **Response in exakter Wire-Shape**
  die der Client/Adapter parst (snake_case, Pagination-Wrapper wie `{objects:[...],total}` — Client
  genau lesen, falsche Shape = leere Seite). Demo-Daten aus den alten `MOCK_*` der Store-Datei
  portieren. GET-Endpoints mit Daten + Mutations als Echo. index.ts NICHT anfassen (Main registriert).

**Phase 2 — gescopter tsc:** eigene Dateien (tsconfig extends tsconfig.web.json + 3 .d.ts), NICHT durch
Pipe. **Kombinierter tsc crasht (Tooling-Bug „Debug Failure")** → per-Modul-Scope. Repo hat ~98
Baseline-typed-i18n-Fehler. Known-i18n-misses für neue Fragment-Keys ignorieren (Merge löst).

**Phase 3 — go test:** nur eigenes Package (`./internal/<modul>/...`), nicht `./...`. Tenant-Isolation
+ Happy-Path je neuem RPC.

**Phase 4 — QA-Script schreiben (NICHT ausführen):** `scripts/qa-<modul>.mjs` (Boilerplate
.planning/nico-block/WORKFLOW.md), Route **`/#/<modul>` (Hash!)**, Onboarding-Suppress-InitScript drin.

**Subagent committet/pusht NICHT.** Report: Audit A/B, Dateiliste, Migr.-Nummern, Fragment-Pfad,
MSW-Handler-Pfad+Export-Name, OpenAPI-Block, neue Permissions, QA-Route, Risiken.

## Main-Session-Integration (Opus) pro Welle

1. i18n-Fragmente in de/en/fr/it.json mergen — **flache gepunktete Keys, APPEND-only, NICHT
   re-sortieren** (Sort-Falle! Datei ist nicht simpel sortierbar). Muster: scripts/add-*-i18n.mjs.
   Mojibake prüfen (latin1→utf8-Roundtrip bei Werten mit Ã/Â).
2. MSW-Handler in `mocks/handlers/index.ts` registrieren (import + spread).
3. OpenAPI-Blöcke in backend/api/openapi.yaml mergen.
4. **Voll-Build/Vet/Test** `go build/vet/test ./...` + golangci-lint betroffene Pakete.
5. **EINEN Dev-Server** `cd desktop && npm run dev` (:5173, electron-vite demo) starten, die 4
   qa-*.mjs laufen, **Screenshots mit Read-Tool ANSEHEN** (Agent-Asserts sind unzuverlässig!),
   echte Daten + Umlaute prüfen. Dev-Server danach killen (`taskkill //F //IM electron.exe //T`).
6. Per-Modul-Commits (Conventional, englisch, **keine AI-Attribution**), i18n/index.ts/openapi als
   eigener `chore`-Commit. **PAUSE — Review-Gate für Luke. NICHT pushen bis OK.** Dann push → CI.

## Pre-Flight (jede Session/Welle)

`git fetch` — **Darien pusht aktiv auf main + erstellt `parallel/*`-Branches** (Welle 1: parallel/helpdesk
kollidierte). Prüfen ob Module der kommenden Welle schon anderweitig in Arbeit sind. CI-Status prüfen
(`gh run list --workflow CI`). Migrations-Kopf neu lesen (`ls backend/migrations | tail`).

## Deferred Follow-ups aus Welle 1

- **Helpdesk** KB-Artikel/Stats/Routing-Rules: bleiben store-basiert, kein Backend-RPC → Backend bauen.
- **Wiki** `DeleteCategory`/`CreateShareToken`/`RevokeShareToken`/`UpdateCategory`: Hooks existieren,
  Backend-Routes nicht (waren schon vor Welle 404) → sauberer Proto-Zyklus.
- **Schichten** FE-Restpunkte: from/to-Datumsfilter an gRPC durchreichen, Drag&Drop-Persist,
  echte Mitarbeiterliste statt EMPLOYEES-Fallback, CreateShift/PublishShifts aus Page verdrahten.
- **Repo-weiter Mojibake-Sweep:** noch **105 Vorkommen** in anderen Namespaces (de/en/fr/it).
- **Baseline:** crm/finance/hr MSW-Handler haben vorbestehende tsc-Fehler (schon auf main).

## Referenz-Pfade

- FE-Muster: `modules/dialer/CampaignListPage.tsx` + `api/hooks/useDialer.ts`; Hybrid `modules/dokumente`
- Backend-Schichtung: `internal/work/` + `proto/work/v1/work.proto` + `internal/server/work_grpc.go` + `route_work.go`
- MSW: `mocks/handlers/dialer.ts` (Template), `mocks/handlers/index.ts` (Registrierung), `mocks/demo-mode.ts`
- i18n: `scripts/add-payroll-masterdata-i18n.mjs` (Append-Muster), `messages/{de,en,fr,it}.json` (flat)
- Verify-Loop: `.planning/nico-block/WORKFLOW.md`
