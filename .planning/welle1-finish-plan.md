# Welle 1 — Abschluss-Plan (autonomer Lauf, Start 2026-06-28)

> Ziel: **alles self-doable von Welle 1 fertig**, dann Übergang Welle 2. Luke-Gebundenes bleibt liegen (notiert, nicht warten).
> Lebende Checkliste — pro Item Status pflegen. Backend-Stand gegen Code verifiziert (nicht nur Lukes Wort).

## Entscheidungen (Darien, 2026-06-28)
- **Reihenfolge:** ich entscheide → Echt-Schaltungen zuerst (FE-fertig, schnell), dann documents-Härtung, dann X-4. security nur eingestreut verifizieren.
- **security:** NUR verifizieren (Backend-Realitäts-Check) + berichten. KEIN Echt-Schalten in diesem Lauf.
- **documents:** FE-tolerant abfangen (Normalizer) + Luke-Notiz für kanonischen Backend-Fix.
- **Push:** pro fertigem Modul. CI-grün (eslint geänderte Dateien + scoped tsc + `npm run build`) → Commit+Push. ⚠ Jeder grüne Push = Auto-Deploy auf Hetzner.

## Backend-Gegencheck (verifiziert aus Code, 2026-06-28)
- helpdesk: tenant-Fix bestätigt (`route_helpdesk.go:260` setzt `TenantId`). FE schon hook-basiert.
- inbox: gRPC-Service komplett (`route_inbox.go`, alle Handler). FE schon `useInboxMessages`. Läuft im `notification`-Binary.
- zeiterfassung: voll hook-basiert, null-guard `hr-hooks.ts:508` schon `?? []`. biz-Service.
- documents: Listen jetzt bare-array via `response.ProtoList`, Singles `{folder}`/`{file}`. FE `unwrapList`/`unwrapEntity` defensiv → READ ok; 8 Rand-Konflikte (s.u.).
- security: `route_security.go` ~25 Endpoints (audit/vault/gdpr/password/ip/retention) über echten `auth`-gRPC-Client — VIEL weiter als Plan-„2/10". Realität (Stub vs echt) noch zu prüfen.

---

## Block 0 — Infra (Voraussetzung)
- [ ] Docker Desktop hoch + Daemon ready
- [ ] Stack `--no-deps --no-build`: postgres redis minio createbucket auth crm gateway work biz + **helpdesk document notification wiki automation berichte chat**
- [ ] Gateway mit `-f docker-compose.flags.yml` (Feature-Flags)
- [ ] Seeds: `demo-seed.sql` + modul-spezifische (dialer/hr); Login `demo@local.test` / `Demo1234!`
- [ ] Health-Check Gateway :8080

## Block A — Echt-Schaltung (Live-QA + mock-verdeckte Bugs)
### A1 · zeiterfassung (kleinster Rest — Start hier)
- [ ] biz+minio live, Demo-Mode aus, FE-Screenshot (war Dev-Server-Quirk)
- [ ] `team/AbsenceCalendar.tsx:208` defensive guard `data?.entries?.find`
- [ ] Commit+Push
### A2 · helpdesk
- [ ] Live-QA Ticket-Liste/Detail gegen echtes Backend (tenant-Fix)
- [ ] `BusinessHoursDialog.tsx:31` Store→`useBusinessHours()`/`useUpdateBusinessHours()` (Hooks existieren)
- [ ] `CSATWidget.tsx` — Store-Bindung prüfen (Hook fehlt → Luke-Notiz oder belassen)
- [ ] Backend-Feldlücken description/category/assignee_name → **Luke-Notiz**
- [ ] mock-verdeckte Bugs fixen · Commit+Push
### A3 · kommunikation-Inbox
- [ ] notification-Binary live, Demo-Mode aus, ListMessages/UnreadCount live-QA
- [ ] Thread-Verlauf synthetisch (`buildThreadSeed` + `inboxThread.ts`) → **Luke-Notiz** (`ListThreadMessages`-RPC)
- [ ] Canned/Status/Tags lokale Stores → Luke-Notiz oder spätere Phase
- [ ] mock-verdeckte Bugs fixen · Commit+Push

## Block B — documents-Härtung (FE-tolerant)
Client: `api/clients/document-client.ts` (inline Normalizer). 8 Konflikte aus Audit:
- [ ] K6 Shares-URL: FE `/documents/shares` → Backend `/documents/shares/entity` (FE-Fix, `document-client.ts:405`)
- [ ] K7 `normalizeShare()` für Enum-Int `permission` (1→read/2→write), in `documentShareApi.list()`
- [ ] K5 ListSharedWithMe: Backend `{files,folders,total}` ≠ FE `{shares}` → neuer Typ + Hook
- [ ] K1/K2/K3 naked Link/Share/Tag: FE-tolerant `unwrapEntity` einbauen → **Luke-Notiz** (kanonisch wrappen)
- [ ] K4 RevertFileVersion: Backend `{file}` ≠ FE `{version}` → FE-Typ anpassen + Luke-Notiz
- [ ] K8 GetWOPIDiscovery fehlt im Client (optional, nur falls UI nutzt)
- [ ] Live-QA · Commit+Push

## Block C — security verifizieren (NUR Check + Bericht)
- [ ] Backend-Realitäts-Check: sind die ~25 Endpoints echt implementiert (gRPC-Methoden mit DB) oder Stubs? Spot-Check `internal/server/security_grpc.go` + Live-Call (read-only: audit-list, password-policy, ip-rules — NICHT erasure/execute)
- [ ] Bericht: was ist echt-schaltbar, was fehlt → in `backend-gaps.md` + Chat-Bericht an Darien
- [ ] KEINE Echt-Schaltung, KEIN destruktiver GDPR-Test

## Block D — X-4 Settings-Rollout
- [ ] `useTenantSettings(moduleId)`-Hook bauen (analog `api/hooks/useUserSettings.ts`, scope `/settings/{mod}/tenant`)
- [ ] 11 triviale Stores → `initFromServer` + Write-Through (Muster `dokumenteSettings.ts`):
  - personal: dokumentePrefs · crmPrefs · financePrefs · teamPrefs · dashboardPrefs · helpdeskPrefs · zeiterfassungPrefs · wikiPrefs
  - tenant: wikiSettings · dashboardSettings · zeiterfassungSettings · financeTenant
- [ ] 2 mittlere: workPrefs · vertraegePrefs
- [ ] gemischte (dialerPrefs/automatisierungPrefs/berichtePrefs/mailPrefs/formularePrefs) = Store-Split → **Welle 2**
- [ ] groß (payrollSettings/workSettings) = Backend-API teils fehlt → **Welle 2/Luke**
- [ ] Live-QA (localStorage wipen → hydratet aus Backend) · Commit+Push

---

## Luke-Notizen (sammeln in backend-gaps.md, NICHT warten)
- mails IMAP/SMTP (echter Block)
- documents: Link/Share/Tag naked→wrapped kanonisch · RevertFileVersion-Shape · ListSharedWithMe-Konsistenz
- helpdesk: description/category-Felder + assignee_name denormalisiert im WireTicket
- inbox: `ListThreadMessages`-RPC + Canned-Response-CRUD
- ⚠ **Feature-Flags Prod:** `.env.production` muss `COSMI_MODULE_*_ENABLED` setzen, sonst deployt Auto-Deploy ohne helpdesk/wiki/berichte/formulare/vertraege/video
