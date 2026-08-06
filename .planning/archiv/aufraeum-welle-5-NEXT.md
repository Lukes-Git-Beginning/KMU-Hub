# START-PROMPT — Aufräum-Welle (Welle 5)

> Konsolidiert alle Deferred-Follow-ups aus Welle 1–4 + die Mini-Lücken aus der Welle-3/4-Session.
> Ist-Stand am 2026-06-18 per Explore-Audit verifiziert (nicht angenommen). Self-contained.

## Kontext

Die 10-Modul-FE↔Backend-Wiring-Initiative ist abgeschlossen (Welle 1–4 live auf Prod, Migr.-Kopf 209).
Diese Welle räumt die bewusst zurückgestellten Reste + neu entdeckte Mini-Lücken auf. Gleiches
Build-+-Verify-Protokoll wie Welle 3/4, **Worktree-Isolation Pflicht, Pause-Gate, Main committet ALLES.**

## Streams (max 4 parallele Sonnet-Subagenten, je 1 Modul; Quick-Fixes macht die Main-Session)

### Stream A — Helpdesk (Backend + FE, GROSS)
KB-Artikel, Stats und Routing-Rules sind noch store-basiert; `DeleteSLAPolicy` fehlt (Client wirft
`backend route not yet implemented`).
- **Bauen:** RPCs `GetHelpdeskStats`, `List/Create/Update/DeleteKBArticle` (4), `List/Create/Update/DeleteRoutingRule` (4), `DeleteSLAPolicy` (1) → Proto + gRPC + Service + Gateway + Migrationen (`helpdesk_kb_articles`, `helpdesk_routing_rules` + Permission-Seeds). Stats als Aggregat-Query (kein eigenes Table).
- **FE:** `HelpdeskPage.tsx` (kbArticles/stats aus `useHelpdeskStore` → Hooks), `TicketRoutingConfig.tsx` (routingRules → Hooks), `useDeleteSLAPolicy`/`helpdesk-client.ts` reparieren. MSW-Handler erweitern.
- Dateien: `backend/proto/helpdesk/v1/helpdesk.proto`, `internal/server/helpdesk_grpc.go`, `internal/.../helpdesk/`, `route_helpdesk.go`; `modules/helpdesk/HelpdeskPage.tsx`+`TicketRoutingConfig.tsx`, `api/hooks/useHelpdesk.ts`, `api/helpdesk-client.ts`, `mocks/handlers/helpdesk.ts`.
- **Migr.-Block: 000210–000214.**

### Stream B — Wiki fehlende Ops (Backend, MITTEL)
FE-Client ruft 5 Ops auf, die backend-seitig 404en: `DeleteCategory`, `UpdateCategory`,
`CreateShareToken` (`POST /articles/{id}/share`), `RevokeShareToken` (`DELETE /share/{tokenId}`),
`GetVersion` (standalone `GET /versions/{id}`).
- **Bauen:** 5 Proto-RPCs + gRPC-Handler + Gateway-Routes + Service-Logik. Neue Tabelle `wiki_share_tokens` (Migration + Permission-Seed). DeleteCategory-Logik (Artikel-Reassign/Guard).
- **FE:** bestehende Hooks (existieren bereits in `useWiki*`) sind schon da — nur Backend bauen + MSW-Handler ergänzen.
- Dateien: `backend/proto/wiki/v1/wiki.proto`, `wiki_grpc.go`, `internal/wiki/`, `route_wiki.go`, `mocks/handlers/wiki.ts`.
- **Migr.-Block: 000215–000217.**

### Stream C — Schichten FE-Restpunkte (FE-only, GROSS — Backend existiert)
`SchichtenPage.tsx` lädt den Wochenplan aus `buildMockAssignments()`; 4 Restpunkte:
- (a) `useShiftsList({from,to})` mit `weekDates` verdrahten statt Mock-Grid (`buildMockAssignments` raus).
- (b) Echte Mitarbeiterliste aus HR-API statt hardcodiertem `EMPLOYEES`-Array (Z.164–173).
- (c) Drag&Drop persistieren: `handleDrop` → `assignEmployeeMutation`/`unassignMutation` (statt nur Toast).
- (d) `useCreateShift`/`usePublishShifts` aus der Page verdrahten (sind in `useSchichten.ts` definiert, ungenutzt); `handleAssignSubmit`-TODO auflösen.
- Dateien: `modules/schichten/SchichtenPage.tsx`, `api/hooks/useSchichten.ts` (nur nutzen), ggf. MSW-Handler erweitern (echte Mitarbeiter/Shifts).

### Stream D — (optional 4. Agent) project_id end-to-end (Backend, MITTEL)
`hr_work_time_entries` hat keine `project_id`-Spalte → by-project-Analytics im echten Backend leer.
- Migration `ALTER TABLE hr_work_time_entries ADD project_id UUID NULL REFERENCES hr_time_projects(id)`,
  `project_id` in `WorkTimeEntry`-Proto-Response + `toProtoWorkTimeEntry` + `CreateManualEntry`-Handler (liest
  `req.ProjectId` aktuell nicht) + `GetTimeAnalytics`-by-project-Query.
- **Migr.-Block: 000218.** (Falls nur 3 Agenten: Main-Session macht das.)

### Main-Session Quick-Fixes (kein Agent — klein, querschnitt)
1. **Mojibake de.json**: 90 Zeilen latin1→utf8-Artefakte (`Ã¼`/`Ã¶`/`Ã¤`/`ÃŸ`/`Ã„` etc.) NUR in
   `messages/de.json` (fr/en/it sauber). Script: pro betroffenem Value `Buffer.from(v,'latin1').toString('utf8')`
   — aber selektiv (nur Values mit `Ã`/`Â`), JSON neu schreiben (UTF-8 ohne BOM, kein Re-Sort). Danach validieren + Stichprobe ansehen.
2. **HRSettings**: 3 fehlende Felder (`work_hours_per_day`, `max_daily_hours`, `break_after_hours`) in
   `HRSettings`-Typ (`hr-types.ts`) + Backend-Endpoint ergänzen; `as any`-Casts in `OverviewView.tsx`/`CategoriesView.tsx` entfernen; `dailyTarget=480` durch `work_hours_per_day` ersetzen.
3. **OpenAPI**: Welle-3/4-Endpoints (produktion/fuhrpark/einkauf/hr-time) + schichten/helpdesk/wiki in
   `backend/api/openapi.yaml` dokumentieren (CI validiert nur Wohlgeformtheit → nicht-blockierend, aber API-First-Schuld).

## Verbindliche Lektionen (aus Welle 3/4 — NICHT wiederholen)
- **Permission-Seeds:** `permissions` hat `(name, resource, action)` — KEINE `description`-Spalte. Seed `INSERT ... (name,resource,action) ... ON CONFLICT (name) DO NOTHING` + admin-Grant via `role_permissions`. (Welle-3-CI-Fail.)
- **CHECK-Constraint-Namen:** nie eine explizite `CONSTRAINT <table>_<col>_check` neben einer Spalten-CHECK auf derselben Spalte (Postgres auto-named → Kollision). (Welle-3-CI-Fail.)
- **Migrationen lokal testen:** `migrate -path backend/migrations -database <pgvector/pgvector:pg16> up` muss bis zum Kopf clean — `postgres:alpine` fehlt `vector`-Extension.
- **MSW-Handler-PFLICHT inkl. Fall-A:** auch bestehende Endpoints, die eine rewired Page jetzt zieht, brauchen MSW-Routen — sonst Stuck-Loading (Welle-3-Produktion-Bug: orders/bookings vergessen).
- **Gateway via gRPC-Client** (kein Direct-Svc), **Proto-Regen NUR betroffenes Modul**, keine `.pb.go`-Handedits, keine `codes.Unimplemented`-Stubs, slog, Idempotency bei Create.
- **i18n:** flache gepunktete Keys, APPEND-only, ICU-Plural, `{var}` nicht `{{var}}`, korrekte Umlaute (nie ASCII „fuer"/„Auftraege"), auch in **Demo-Seed-Daten** der MSW-Handler.
- **Gates auf finalem Tree:** `eslint src/` 0 Errors (unused-imports/prefer-const = ERROR), `npx tsc --noEmit` (Default-tsconfig, NICHT scoped-extends — das crasht), `go build ./cmd/gateway/... ./cmd/biz/...` (Voll-`./...` OOM-t beim 24-Binary-Link), `go vet ./...`, Modul-`go test`. Nie durch Pipe (Exit-Masking).
- **Subagenten:** `isolation:"worktree"`, committen im Worktree, NICHT pushen/mergen; Main reconcilet (`git worktree list`, Diff-vs-Report, Duplikat-/Streuner-Check), merged, verifiziert unabhängig, committet, **Screenshots ansehen**, Pause-Gate, dann push.

## Verifikation + Abschluss
Pro Stream: go build(gezielt)/vet/test + eslint+tsc grün → i18n-Fragmente mergen + MSW in index.ts (falls neu) → **EIN Dev-Server :5173 demo + qa-<modul>.mjs + Screenshots ansehen** (echte Daten, keine Raw-Keys/Crashes) → per-Modul-Commits → Pause-Gate → push → CI+CI-Desktop+CD beobachten → Prod-Migrationskopf + Smoke.
