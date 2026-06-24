# RESUME — nächster Einstieg (Stand 2026-06-23, Session-Ende #2)

> **★ UPDATE 2026-06-24 #2 — dialer SUPERVISOR echt-geschaltet (Welle 1), live verifiziert + gepusht (`48b5daf9`).**
> Lukes neue Endpoints (`GET /dialer/supervisor` + `/dialer/contacts/{id}/calls`, `fb045f9f`) ans FE gehängt. **Zwei mock-verdeckte Bugs gefunden:**
> (1) **`recent_calls` immer leer** — Lukes Query las `cc.contact_name` (Spalte existiert nicht in `dialer_campaign_contacts`); SQL-Fehler wurde still als WARN geschluckt → Feed leer. Gefixt: crm-`contacts`+`companies`-Join in `GetRecentCallsForTenant` (`c10f8d2f`, im dialer-Service).
> (2) **protojson lässt Null-Werte weg** (`EmitUnpopulated:false`) → `totals.active_agents`/`on_call` fehlten, `recent_calls` fehlte ganz → FE wäre bei leerem Dialer abgestürzt (`recent_calls.length` auf undefined). FE-Normalizer `api/dialer-normalize.ts` füllt die Defaults (`48b5daf9`), eingehängt in `dialer-client.ts`.
> **Verifikation:** Docker-Stack hoch (postgres/redis/auth/crm/gateway/dialer healthy), Dialer-Demo-Seed `backend/seeds/demo/dialer-demo.sql` (Kampagne + 2 Agents + 3 Outcomes + 5 Call-Sessions HEUTE → 5 Calls/2 Termine). Live gegen :8080: Supervisor zeigt KPIs (Aktive 0/Im Gespräch 0/Anrufe 5/Termine 2 — die 0 rendern statt undefined = Normalizer wirkt), Team mit calls_today, Letzte Anrufe voll. Screenshots `desktop/.qa-screenshots/dialer-supervisor/`, Skript `qa-dialer-supervisor-localbackend.mjs`.
> **LEHRE (Windows):** `curl | python -m json.tool` zeigt UTF-8-Umlaute fälschlich als Mojibake (`Ã¼`) — Python 3.14 liest stdin als cp1252, NICHT UTF-8. Echte Bytes mit `xxd` prüfen (`C3 BC` = sauberes ü). Es gab KEINEN Encoding-Bug; die Namen rendern sauber.
> **Gebaut-aber-nicht-meine-Lane:** `npm run build` war rot wegen `@livekit/track-processors` — nur stale node_modules, `npm install` fixt es (Dep ist in package.json). Danach Build grün.
> **PARALLEL:** Sub-Terminal baut `security`/DSGVO auf Branch `parallel/security` (Paket `.planning/parallel-batch/sub-security.md`, S-0 done, S-1…S-5 freigegeben). Main merged den Branch am Ende.
> **Docker läuft noch** (nur Main fasst Docker an). Offen für dialer: Contact-Calls-Detail im UI screenshotten (ContactDetailModal), Supervisor-Leer-Zustand sauber live testen (Pass B nutzte Cache).


> **★ UPDATE 2026-06-24 — B-12 DONE, Buchhaltung KOMPLETT echt (gepusht `4712857a`).**
> (1) Betrag-Fix `protoTaxBreakdown()`-Fallback in `biz_grpc.go` — `toProto{Invoice,Quote,CreditNote}` lesen jetzt das `tax_breakdown`-JSONB **oder** die Einzelspalten `subtotal/total_tax/gross_total` (der Seed füllt nur die Spalten → vorher 0,00 €). (2) Zweiter mock-verdeckter Bug gefixt: Dunning-Pfade in `finance-client.ts` **und** `mocks/handlers/finance.ts` Plural→Singular (`/finance/dunning` + `/dunning/config`) — das Gateway routet Singular, Plural gab 404 und der Mahnungen-Tab degradierte still zum Empty-State. Alle finanzen-Tabs live verifiziert: `desktop/.qa-screenshots/b12-finanzen/`. QA-Skripte: `qa-b12-finanzen-amounts.mjs`, `qa-b12-dunning-settle.mjs`.
> **Recovery-Lehre:** `docker compose up` OHNE `--no-deps` zieht den ganzen gateway-Dependency-Graph (alle 23 µSvc) rein und baut sie → WSL2-vmmem-Explosion (16 GB Maschine, RAM auf 1,4 GB) → Daemon-Hänger. Immer `up -d --no-deps <nur-was-man-braucht>`. Recovery: Docker Desktop killen + `wsl --shutdown` (gibt vmmem frei) + neu starten.

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
