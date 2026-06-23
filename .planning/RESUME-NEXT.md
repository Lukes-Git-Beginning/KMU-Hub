# RESUME — nächster Einstieg (Stand 2026-06-23, Session-Ende #2)

> **★ MOCK-EXIT — kontakte ist KOMPLETT echt (Referenz-Modul).** READ + voller CRUD (Create/Update/Delete) durch die echte UI gegen das lokale Backend, live verifiziert (Screenshots `desktop/.qa-screenshots/crud-*.png`). Casing-Entscheidung getroffen: **Option C** (per-Modul `dual()`-Adapter, kein globaler Transform — FE ist gemischt-casing). Vollständiger Bericht + Backend-Handover + camelCase-Risiko-Set für die nächsten Module: **`.planning/kontakte-mock-exit-DONE.md`**.
>
> **Diese Session neu:** `api/casing.ts` (`dual()`-Helper), `mocks/demo-mode-flag.ts` (Leaf-Flag), kontakte-Adapter mode-branched + position↔jobTitle, useContacts PATCH→PUT, Mock-Handler PATCH→PUT, Demo-User→admin (Seed idempotent). 3 weitere Mock-verdeckte Bugs gefunden+gefixt (PUT-Methode, position-Feld, custom_fields-Array).
>
> **Voriger Durchstich (Session #1):** Login + Kontakte-Liste echt, 2 Bugs gefixt (CORS-Idempotency-Key `d4a9c1a4`, Contact-Adapter snake_case `3979b142`).

## Was diese Session fertig wurde (gepusht, `043cb372`)
- **Lokales Backend läuft** via Docker (`deploy/docker/docker-compose.yml`): **postgres + redis + auth + crm + gateway** (Minimal-Subset; voller 24-Service-Stack crasht die Maschine → nur bauen, was man braucht). Gateway auf `:8080`, Migrationen bis **000226**.
- **Demo-Seeds** (`backend/seeds/demo/demo-seed.sql`, idempotent, Tenant `…0001`): 8 companies, 12 contacts, 8 deals, 3 projects, 10 tasks. **Finance-Block auskommentiert** (line_items ist separate `finance_line_items`-Tabelle → noch fixen).
- **Mock-Exit verifiziert end-to-end:** Login (`demo@local.test` / `Demo1234!`) → Kontakte mit echten Namen/Firmen/Avataren. QA-Skripte: `desktop/scripts/qa-mock-exit-kontakte.mjs` (Token-Inject) + `qa-mock-exit-login.mjs` (echter Login-Flow).
- **2 echte Bugs gefixt (Mocks hatten sie verdeckt):**
  - `fix(gateway)` `d4a9c1a4` — **CORS allow-headers** um `Idempotency-Key` ergänzt. HardMode verlangt den Header bei jeder Mutation, CORS verbot ihn → jede Mutation (Login/Create/Update) aus jedem Browser-Client blockiert. **Betrifft Luke/Prod.**
  - `fix(kontakte)` `3979b142` — Contact-**Adapter liest snake_case** (Gateway liefert `first_name`, OpenAPI-Typen sind camelCase = **Spec-Drift X-3**). Sonst Namen/Firma leer. **Muster betrifft JEDES Modul beim Mock-Exit.**
- **Tooling** `043cb372` — `RENDERER_VITE_DEV_BYPASS_AUTH=false` erzwingt echten Login im Dev-Build (`App.tsx`); `.planning/mock-exit-readiness-matrix.md` (Modul × Backend × Wire-Shape × Auth × RLS); `SESSION-RUNBOOK.md` Markt-Recherche als Pflicht-Schritt.
- **NICHT angefasst:** Login-Animation/`AuthLayout` (läuft auf main+Hetzner korrekt; das „falsche" C-Icon war nur ein Dev-Artefakt durch wiederholte Reloads → statischer Fallback statt Animation).

## Lokal wieder hochfahren (neues Terminal)
```bash
# 1. Docker-Backend (läuft evtl. noch — prüfen):
docker ps   # postgres/redis/auth/crm/gateway healthy?
# falls weg: cd "C:/Users/darie/Documents/KMU Hub"
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/.env up -d --no-deps postgres redis auth crm gateway
# (.env liegt unter deploy/docker/.env — gitignored; Werte: deploy/docker/README.md + MIGRATION_DATABASE_URL)
# Seed (falls DB frisch): docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub --single-transaction < backend/seeds/demo/demo-seed.sql

# 2. FE gegen echtes Backend (Mode localbackend = DEMO_MODE=false + :8080 + echter Login):
cd desktop && npx electron-vite dev --mode localbackend
# Login: demo@local.test / Demo1234!  (Tenant …0001, sieht Seed-Daten)
# Hinweis: nur kontakte/firmen/deals live (crm); andere Module 503 (Service nicht gebaut)
```

## Was als Nächstes (Reihenfolge nach Hebel)
1. **~~OpenAPI-Casing~~ GELÖST** — Entscheidung Option C (per-Modul `dual()`). Globaler Transform verworfen (FE gemischt-casing, würde Tausende snake-Leser brechen). Casing-Risiko-Set + Pattern in `kontakte-mock-exit-DONE.md`.
2. **Nächstes Modul nach kontakte-Pattern echt schalten** — Reihenfolge nach Risiko-Set: crm/companies → crm/deals+pipeline-stages (DealInfo-Casing!) → work. Pro Modul: `dual()`-Adapter falls OpenAPI-getippt, Methode/Wire-Shape/Idempotency/RBAC gegen echtes Backend prüfen (nicht nur Mock).
3. **work + biz dazuholen** → Aufgaben/Projekte/Finanzen echt (`docker compose build work biz` + `up -d --no-deps`).
4. **Finance-Seed fixen** (line_items → `finance_line_items`-Tabelle) → finanzen-Demo nicht leer.
5. **RLS-scharf testen:** `DATABASE_URL` auf `kmuhub_app:app_dev` (statt Superuser) → wie Prod. Migration 000121, einmalig `ALTER ROLE kmuhub_app WITH PASSWORD 'app_dev'`.
6. **Luke-Handover offen** (siehe `kontakte-mock-exit-DONE.md`): contact-Schema zu dünn (9 Extra-Felder), OpenAPI-Spec-Drift contacts (PATCH→PUT, title→position, custom_fields-Array), Timeline-Endpoint hängt.

## Parallel: regulärer Bau-Track (MASTER-PLAN.md)
Der Mock-Exit ist Welle 1 (Echt-Schaltung) in Aktion. `MASTER-PLAN.md` bleibt die SSOT für die übrigen Wellen. SESSION-RUNBOOK-Zyklus gilt weiter.
