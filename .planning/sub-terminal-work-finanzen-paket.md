# Sub-Terminal-Paket — work + finanzen echt-schalten (Mock-Exit Lane B)

> Vorbereitet vom Main-Terminal (das parallel crm/companies+deals+pipeline-stages echt-schaltet). Dieses Paket ist **disjunkt** — keine gemeinsamen Dateien außer `backend/seeds/demo/demo-seed.sql` (nur Finance-Block, klar abgegrenzt → bei Gleichzeitigkeit serialisieren).

## Auftrag
work (Tasks/Projekte) **und** finanzen aus dem MSW-Mock auf das echte lokale Docker-Backend umstellen — nach dem kontakte-Referenz-Pattern. Plus den Finance-Demo-Seed reparieren.

## ZUERST LESEN (Pflicht)
1. `.planning/kontakte-mock-exit-DONE.md` — das Referenz-Pattern: `api/casing.ts` `dual()`, `mocks/demo-mode-flag.ts` mode-branch, und die typischen Mock-verdeckten Bugs (PUT≠PATCH, Feldnamen-Drift, custom_fields-Array, RBAC). **Nachbauen, nicht neu erfinden.**
2. `.planning/mock-exit-readiness-matrix.md` — Wire-Shape-Fallen, die drei Stolpersteine (Auth+Idempotency, Wire-Shape, RLS/tenant_id).

## Backend hochfahren (work + biz dazu)
Das crm-Backend läuft bereits (postgres/redis/auth/crm/gateway). **NUR work + biz ergänzen, nichts anderes anfassen:**
```bash
cd "C:/Users/darie/Documents/KMU Hub"
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/.env build work biz
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/.env up -d --no-deps work biz
docker ps   # work + biz healthy?
```
⚠️ **GUARDRAIL:** NIE `down`, NIE postgres/redis/auth/crm/gateway neu starten (Main-Terminal hängt daran). Nur `--no-deps work biz`.

## Recherche-Ergebnis (vorab erledigt — verifizieren, nicht neu suchen)

**Backend/Gateway:** work + biz sind vollständig gewired. work-Endpoints: `/api/v1/projects`, `/api/v1/tasks` (+ subtasks/comments/links/activities/files/time-entries/timer). finanzen: `/api/v1/finance/{quotes,invoices,credit-notes,payments,dunning,dashboard,...}`.

**Casing:** work + finanzen sind überwiegend **snake_case = „matcht schon"** (handgetippte Hooks/`finance-client.ts`). `dual()` NUR nötig in `modules/work/components/CustomFieldsSection.tsx` (CustomFieldInfo camelCase: `isRequired/fieldType/entityType/sortOrder`).

**Fetch-Layer:** work nutzt `apiClient` (openapi-fetch) + 3× `authenticatedRequest`. finanzen nutzt `finance-client.ts` → `authenticatedRequest` (kein openapi-fetch).

**KRITISCHE Blocker (live verifizieren + fixen):**
1. **finanzen `useInvoice`/`useQuote`:** Gateway `HandleGetInvoice` gibt das Invoice-Objekt **flach** zurück (`response.JSON(w, resp.Invoice)`), aber der Hook macht `.invoice` darauf → `undefined`. Prüfen + `select` anpassen.
2. **finanzen Idempotency:** `finance-client.ts` sendet **keinen** `Idempotency-Key` bei POST → `IDEMPOTENCY_MODE: hard` gibt 400. Header ergänzen (wie `authenticatedFetch.ts` es für andere tut — ggf. nutzt finance-client schon authenticatedRequest, dann kommt der Key automatisch; verifizieren).
3. **finanzen RBAC:** Migration 000045 seedet evtl. keine `finance:write`-Permission → 403 möglich. Live testen (Demo-User ist admin → sollte gehen). Falls 403: Permission/Rolle prüfen.
4. **work `useUpdateTask`:** filtert Demo-Felder `completed_at`/`is_closed` vor PUT (echtes Backend kennt sie nicht).
5. **work `ListProjects`:** Gateway-Request ohne TenantId-Feld → RLS-Risiko leere Liste. Wenn Projekte-Liste leer ist: hier suchen.
6. **Nicht swapbar (Mock lassen):** `useFinanceLedger` (`/finance/expenses`,`/transactions`), recurring/banking/chains, `useProjectTimeEntries/TeamUtilization/GuestOverview` — diese Endpoints existieren NICHT im Gateway. NICHT anfassen, als Mock-only belassen.

## Finance-Seed-Fix (Resume-Punkt #3)
**Der Kommentar im Seed ist FALSCH:** `finance_invoices.line_items` ist eine **JSONB-Spalte**, KEINE separate `finance_line_items`-Tabelle (Migration `000045_create_finance_tables.up.sql`). → Den auskommentierten Finance-Block (Abschnitte 10+11) direkt aktivieren, `line_items` als JSONB-Array. FK-Reihenfolge: `company_settings` → `finance_number_sequences` → `finance_invoices` → `finance_quotes` → `finance_credit_notes` → `finance_payments`. Idempotent (`ON CONFLICT (id) DO NOTHING`). Demo-Tenant `…0001`, created_by Demo-User-UUID. Danach Seed einspielen (stdin-Pipe, nicht docker cp):
```bash
docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub --single-transaction < backend/seeds/demo/demo-seed.sql
```

## Login / FE
`cd desktop && npx electron-vite dev --mode localbackend` (läuft evtl. schon auf :5173 — nur 1 Dev-Server, prüfen). Login: **`demo@local.test` / `Demo1234!`** (ist admin).

## Workflow (Gate VOR dem Bauen)
1. **Verifizieren:** Backend hochfahren, je Endpoint live proben (curl mit Token, wie kontakte-DONE-Doc zeigt) — Methode, Wire-Shape, Wrapping, RBAC. Bugs notieren.
2. **GEBÜNDELT Fragen an Darien** (offene Produktentscheidungen, z.B. Mock-only-Felder, wie mit Wire-Mismatches umgehen) — BEVOR du baust.
3. **Bauen:** je Modul Hooks/Adapter fixen, `dual()` nur wo camelCase, Mock-Handler an echte Methoden angleichen (wie kontakte PATCH→PUT). Mode-branch wo nötig.
4. **QA:** Playwright-Screenshot-Skript (`desktop/scripts/qa-mock-exit-*.mjs` als Vorlage) → Login → Modul → Liste/Detail/CRUD → **Screenshots wirklich ansehen** (keine Raw-Keys, echte Daten, keine leeren Listen).
5. **Lint:** `node_modules/.bin/eslint --quiet <geänderte src-Dateien>` (CI fährt `eslint src/`).
6. **Scoped tsc:** eigene `tsconfig.<name>check.json` über nur die geänderten Dateien (Full-tsc ist ~30 Min, kein Gate).
7. **Commit + Push** (Conventional, English, imperativ, KEINE AI-Attribution). Danach MASTER-PLAN abhaken (work/finanzen Echt-Schaltung) + RESUME-NEXT aktualisieren.

## Hot-File-Guardrails
- **NICHT anfassen** (Main-Terminal): `mocks/handlers/crm.ts`, `api/hooks/useContacts.ts`, `api/casing.ts`, `modules/crm/*`, `modules/kontakte/*`.
- `api/casing.ts` darfst du **importieren/lesen** (read-only), nicht editieren.
- `backend/seeds/demo/demo-seed.sql`: nur den Finance-Block (10+11). Bei Konflikt: kurz abstimmen.
- Deine Dateien: `api/hooks/useTasks|useProjects|useTimeEntries|useTaskComments|useTaskFiles|useTaskActivities.ts`, `api/hooks/useFinance.ts`, `api/finance-client.ts`, `mocks/handlers/work.ts`, `mocks/handlers/finance.ts`, `modules/work/*`, `modules/finanzen/*`.
