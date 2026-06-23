# Mock-Exit-Readiness-Matrix

> **Zweck:** Modul-für-Modul-Landkarte für den Ausstieg aus dem MSW-Mock-Modus hin zum echten Backend.
> Beantwortet pro Modul: Steht das Backend? Welcher Wire-Shape kommt wirklich? Braucht es Auth/Idempotency? Ist tenant_id/RLS sauber? Wie groß ist der Swap?
> **Stand:** 2026-06-23, verifiziert gegen Code (`backend/internal/gateway/route_*.go` + Service-Handler) + FE-Hooks. Lokales Backend wird parallel hochgefahren.
> **Begründung:** Darien will raus aus dem Mock, sobald das Frontend steht. „FE-mock-fertig ≠ backend-fertig" — diese Matrix trennt beides sauber und verhindert „grün im Dunkeln".

---

## 0 · Die drei Stolpersteine (Luke, 2026-06-23) — gelten für JEDEN Swap

1. **Auth + Idempotency:** Mutations (POST/PUT/PATCH/DELETE) brauchen JWT. Idempotency-Key-Header ist im Gateway auf `IDEMPOTENCY_MODE: hard` gestellt (Dev) → **400 ohne Header**. **ABER (Code-Realität):** hart erzwungen ist der Key aktuell nur an wenigen Endpoints (verifiziert: `POST /hr/time/entries`). Viele Mutations prüfen ihn noch NICHT. → Pro Mutation einzeln verifizieren, nicht pauschal annehmen.
2. **Wire-Shape ≠ flacher Mock:** Das gRPC-Gateway liefert teils verschachtelt/gewrappt. Zwei Encoder im Einsatz:
   - `response.JSON` (`encoding/json`) → Go-Struct-Tags, **Timestamps als `{"seconds":N,"nanos":N}`** (NICHT RFC3339!).
   - `response.Proto` (`protojson`, `UseProtoNames:true, UseEnumNumbers:true`) → **Enums als Integer** (`status:0`), int64 als String, Timestamps als RFC3339. Nur **dialer** nutzt das.
   → Flache Mocks maskieren beides bis zur echten Schaltung. In der Matrix unten je Endpoint erfasst.
3. **RLS/tenant_id:** RLS ist produktiv erzwungen (fail-closed). Ein Endpoint, der tenant_id serverseitig nicht durchreicht, gibt **Phantom-404/leere Listen** statt sichtbarer Fehler. Pfad: JWT `tid` → Gateway-Middleware → gRPC `x-tenant-id`-Metadata → pgxpool `set_config('app.tenant_id',…)` → RLS-Policy `current_tenant_id()`. Ohne gesetztes GUC sind **alle Zeilen unsichtbar**.

---

## 1 · Die Matrix (schaltbare Module zuerst)

| Modul | Backend | FE-Quelle heute | Swap-Punkt | Wire-Shape (echt) | Auth | Idempotency | tenant_id/RLS | Swap-Aufwand |
|---|---|---|---|---|---|---|---|---|
| **dashboard** (Layout) | 🔌 fertig (Migr 023/114) | localStorage + `apiClient` (initFromServer/sync schon da) | `stores/dashboard.ts` | flach `{layout:[],active_widgets:[],is_custom,updated_at}` | ja | PUT /layout: **kein** Key-Check | ✅ `GetTenantID` aus JWT | **niedrig** (nur MSW-deregister) |
| **work-Labels** | 🔌 fertig (Migr 145/147) | localStorage (`workSettings`+`taskLabels`), **keine Hooks** | beide Stores → neue API-Hooks | gewrappt `{labels:[]}` | ja | POST/PUT: kein Key-Check | ⚠️ **tenant_id fehlt** im `ListLabelsRequest` → Risiko Cross-Tenant | mittel (Hooks bauen) |
| **zeiterfassung/HR** | 🔌 fertig (Migr 178–182) | `hr-hooks.ts` (echter apiClient-Pfad) → MSW | `api/hooks/hr-hooks.ts` | **inkonsistent**: `/balance` flach · `/entries` `{entries,total}` · `/projects` `{projects}` · `/team` `{team}` | ja | **POST /entries: Key HARD (400 ohne)** | ✅ `getTenantID` überall | mittel |
| **notifications** | ⚠️ fast (Spalten `is_pinned/is_dismissed` fehlen → Luke-Migr; `actor_name` nur Payload) | `apiClient` → MSW | `api/hooks/useNotifications.ts` | gewrappt `{notifications:[],total}`, **Timestamps `{seconds,nanos}`** | ja (+`RequirePermission`) | viele Mutations, **kein** Key-Check | ⚠️ kein `getTenantID` am Gateway (RLS evtl. im Service) | niedrig (nach Luke-Migr) |
| **dialer** | ⚠️ teils (LogCallOutcome + agent-status/campaign-dashboard echt; **`/supervisor`-Overview fehlt**) | `dialer-client.ts` (echter fetch) → MSW | `api/dialer-client.ts` | **protojson!** flach, **Enums als Int** (`status:0`), Timestamps RFC3339 | ja (+Permission) | viele Mutations, **0** Key-Checks | ⚠️ tenant_id nicht befüllt | mittel (+ Backend-Lücke Supervisor) |
| **kontakte** | 🔌 echt (crm, Migr 141) | teils schon echt | `useContacts`/360°-Hooks | crm-Routes (zu verifizieren) | ja | zu prüfen | ✅ | niedrig–mittel |
| **calendar** | 🔌 teils | teils gewired | — | **nested `{calendar:{…}}`** (Luke-Beispiel) | ja | zu prüfen | ✅ | mittel |
| **helpdesk** | 🔌 echt | schon echt | — | zu verifizieren | ja | zu prüfen | ✅ | niedrig |
| **finanzen** | 🔌 stark gewired (biz) | stark echt | — | zu verifizieren | ja | Postings: Key erwartet | ✅ | niedrig–mittel |
| **work** (Tasks/Projekte) | 🔌 fertig | MSW (Hooks fertig) | `api/hooks/useTimeEntries.ts` etc. | zu verifizieren (wrapped?) | ja | zu prüfen | ✅ | niedrig |

**🔒 Backend fehlt real (kein Swap möglich, warten auf Luke):** mails (IMAP/SMTP) · security/DSGVO-Tools · automatisierung-Engine · formulare öffentl. Submit · Branchen-Feature-Endpoints (Aufmaß/Fahrtenbuch/Inventur/BOM…) · E-Rechnung/DATEV · S3-Upload + Signatur-Service · echte KPI-Werte (dashboard-Widgets) · Auth-Invite (team).

---

## 2 · Wire-Shape-Fallen konkret (was Mocks brechen wird)

- **notifications (höchstes Risiko):** `{notifications:[],total}` UND Timestamps als `{seconds,nanos}` statt String → jeder Mock mit nacktem Array oder ISO-Datum bricht. FE-Hook muss unwrappen + Timestamp konvertieren.
- **work-Labels:** `{labels:[]}` gewrappt + **fehlende tenant_id** → ohne Fix evtl. Cross-Tenant-Labels. Backend-Ticket an Luke.
- **HR:** uneinheitliches Wrapping (4 verschiedene Formen) → kein generischer Pagination-Adapter, pro Endpoint einzeln.
- **dialer:** protojson → **Enums als Integer**. Mock mit `"AGENT_READY"` bricht gegen `1`. Mapping-Tabelle im Client nötig.

---

## 3 · Seed-Lage (X-5) — der eigentliche Engpass

**Status: DB ist nach Migration leer.** Es gibt KEINE Demo-Seeds (`backend/seeds/` existiert nicht), keinen Demo-User, keinen Demo-Content. Vorhanden sind nur Schema-Defaults (Rollen/Permissions/Pipeline-Stages/Event-Types) + Bootstrap-Tenant `00000000-0000-0000-0000-000000000001`.

**Login lokal:** kein vorkonfigurierter User → entweder `POST /api/v1/auth/register` (bekommt `member`-Rolle, Henne-Ei bei Admin-Rechten) oder manueller Bootstrap-Admin per SQL (bcrypt-Hash, dann admin-Rolle zuweisen).

**Minimal-Seed (ein SQL-Block, fixe `tenant_id …0001`, FK-Reihenfolge):** Admin-User+Rolle → contacts/companies → `UPDATE pipeline_stages SET tenant_id=…` → deals → projects/tasks → hr_employee_profiles+leave_types → invoices/quotes → inventar_items → chat_channel #general. Damit zeigen die ~10 schaltbaren Module sichtbare Daten.

---

## 4 · Lokales Backend — Setup-Stand

- **Compose:** `deploy/docker/docker-compose.yml` — voller Stack (24 Services + Gateway :8080 + Postgres + Redis + MinIO + LiveKit + OnlyOffice). Gateway-CORS erlaubt `localhost:5173`, `IDEMPOTENCY_MODE: hard`.
- **`.env`:** angelegt unter `deploy/docker/.env`. **Aktuell beide DB-URLs auf `kmuhub`-Superuser** (RLS NICHT scharf — fürs erste Hochfahren). Für RLS-scharfen Test wie Prod: `DATABASE_URL` auf `kmuhub_app:app_dev` umstellen (Migration 000121, einmalig `ALTER ROLE kmuhub_app WITH PASSWORD 'app_dev'`). `MIGRATION_DATABASE_URL` bleibt Superuser (DDL/RLS-Setup).
- **Es existiert `.env.example`** (mit `kmuhub_app:app_dev`) — als Referenz für die RLS-scharfe Variante.

---

## 5 · Empfohlene Schalt-Reihenfolge

1. **dashboard-Layout** — kleinster Swap (Code schon da), beweist die Pipeline End-to-End.
2. **work / kontakte / helpdesk / finanzen** — Backend echt, Hooks da; Wire-Shape je verifizieren.
3. **zeiterfassung** — backend-fertig, aber Idempotency-Key an POST /entries + inkonsistente Shapes einplanen.
4. **work-Labels** — erst nach tenant_id-Fix (Luke) oder mit bewusstem Single-Tenant-Workaround.
5. **notifications** — erst nach Luke-Migration (`is_pinned/is_dismissed`).
6. **dialer** — Enum-Int-Mapping im Client + Supervisor-Overview-Route (Luke) abwarten.

**Voraussetzung für sichtbare Verifikation:** Minimal-Seed (Abschnitt 3) muss laufen, sonst echte-aber-leere Module.
