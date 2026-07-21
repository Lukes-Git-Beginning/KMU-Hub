# Backend-Wellen — Master-Briefing

> **Zweck:** SSOT für die Backend-Aufhol-Strecke via parallele Subagent-Wellen. Analog zu
> `nico-block/WORKFLOW.md` (FE-Delegation) — nur für **Go-Backend**.
> **Detail-Backlog:** `../backend-gaps.md` (453 Zeilen, kuratiert). Dieses Dokument ist der
> **Orchestrierungs-Layer** darüber: Wellenplan + Serialisierungs-Regeln + Gates + Per-Agent-Template.
> **Stand:** 2026-07-21. Migrationskopf Repo **000243** (Prod 242) → nächste freie Nummer **000244**.
> **Fokus (entschieden):** phasiert — RBAC-Fundament zuerst, dann breiter Backlog. **Launch-kritisch (01.09) zuerst**; Branchen/Post-Launch bewusst ausgeklammert (Phase 4, hier nicht ausgearbeitet).

---

## 0. Wie man dieses Dokument benutzt

Zwei Ebenen, ein Ablauf:

1. **Master-Briefing (dieses Doc)** — liest die **Hauptsession** (Opus, Plan-Phase). Sie legt Wellen-Zuschnitt, Migrations-Nummern und Reihenfolge fest.
2. **Per-Agent-Prompt (§6)** — kopierbarer Block. Pro Service/Modul ausgefüllt an je **einen Subagenten** (Sonnet, `isolation: "worktree"`). Funktioniert identisch für separate Terminals **und** Task-Tool-Spawns.

**Grundmodell (aus den Memory-Lektionen, nicht verhandelbar):**
- Agenten arbeiten **read-write nur in ihrem Worktree**, committen NICHT, pushen NICHT.
- **Die Hauptsession merged, gatet, committet und pusht ALLES** — sie ist die einzige Instanz, die `git` auf `main` anfasst.
- Nach **jeder Welle Pause** für dein Review-Gate (§7).

---

## 1. Backend-Ist-Zustand (verifiziert 2026-07-21)

**Gesamtbild:** substanziell echt. 24 Services, **alle** 30 `internal/server/*_grpc.go`-Implementierungen vorhanden, RLS produktiv, das meiste CRUD real. **Kein komplett gestubter Service.** Die `codes.Unimplemented`-Treffer stecken ausschließlich in generierten `*_grpc.pb.go` (gRPC-Basisklasse) = Rauschen.

**Echte Stubs (verifiziert, im Handler-/Service-Layer):** CRM-Advisory-PDF (`route_crm_advisory.go:338`, 501), einkauf `ExportPO` (`einkauf/service.go:716` — FE macht's clientseitig → löschen-Kandidat), rapporte `ExportPDF` (Fake-Text), vertraege-PDF (TODO), **Lexware komplett Placeholder** (`biz/lexware/service.go` — Post-Launch). CRM-Tag-Mutation via HTTP bewusst 501 (gRPC-Pfad lebt).

**Gap-Klassen (nach Hebel):**

| Klasse | Inhalt | Parallelisierung |
|---|---|---|
| **A — RBAC-BE-Nachzug** 🔴 | Datenmodell (`roles.tenant_id`+`based_on`, `role_permissions.scope`) **komplett Greenfield** · `/auth/me/permissions` fehlt · Rollen-Admin-API fehlt · Seeds für **wiki/zeiterfassung/infrastructure = NULL** · helpdesk/chat/kommunikation nur grob · Owner-FKs fürs `own`-Scope fehlen · Guardrails serverseitig | Fundament **seriell**, Enforcement **pro Modul parallel** |
| **B — Contract/Wire-Shape** 🟠 | document-List-Handler (bare→`null`) · 31 security-Endpoints nicht in `openapi.yaml` · helpdesk-WireTicket ohne Namen/Nummer | pro Service parallel |
| **C — Feature-Endpoints (FE mock-first)** 🟠 | zeiterfassung (balance/entries/analytics/Wochen-Freigabe) · admin (tenant-provisioning/billing/invite) · inbox (status/thread/tags/forward — Migr. 237/238 legen Tabellen schon an!) · berichte (KPI/Server-PDF/Cron) · finance (recurring/OP/Mahnwesen/CAMT) | **exzellent** service-partitioniert |
| **D — Cross-cutting (einmal → viele frei)** 🔴 | **S3/MinIO-Upload-Service** (fuhrpark/inventar/rapporte/vermietung/chat/avatar/branding) · **PDF-Service** (rapporte/vertraege/advisory/berichte) · Signatur-Persistenz · einkauf↔inventar-Sync | **seriell zuerst** |
| **E — Stubs finishen/löschen** | einkauf `total_amount` nie berechnet (0,00 €) · finance-Liste 0,00 € (#17) · `POST /pos/{id}/cancel` fehlt | einzeln |
| **F — Branchen-BE** ⚪ | fuhrpark/inventar/vermietung/schichten/rapporte-Modelle | **Post-Launch — hier ausgeklammert** |

---

## 2. Die 3 Serialisierungs-Punkte — NUR Hauptsession, NIE Agenten

Diese drei kollidieren garantiert bei parallelen Agenten. Die Hauptsession besitzt sie exklusiv.

1. **Migrations-Nummern.** Repo-Kopf **000243**. Die Hauptsession **vergibt vor jeder Welle einen Block** und schreibt jedem Agenten seine **exakte Nummer** in den Prompt (`{{MIGRATION_NR}}`). Agenten legen die `.sql`-Datei **mit der vorgegebenen Nummer** an, führen sie **nicht aus** (kein DB-Zugriff im Worktree). Migrations via `make migrate-create name=xxx`-Muster (nie manuell die Sequenz raten). **Forward-only** (Memory: Revert-nach-Deploy = Drift).
2. **Proto-Regen.** Touch an `backend/proto/**/*.proto` → **zentral einmal** regenerieren, nie pro Agent (Memory: „nicht-regeneriertes Proto"). protoc-Pfad: `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/Google.Protobuf_Microsoft.Winget.Source_8wekyb3d8bbwe/bin/protoc.exe`. **Regel:** Proto-ändernde Arbeit wird zur **eigenen seriellen Vor-Welle** (Hauptsession regen + commit), Feature-Wellen bauen auf dem regenerierten Code.
3. **Geteilte Registrierungs-Dateien.** `cmd/gateway/main.go` (Service-Registry + Route-Mount), `go.mod`/`go.sum`, `internal/featureflag/registry.go`. Agenten legen **nur ihre eigene neue Route-Datei** an (`route_<modul>_xyz.go`); die **Hauptsession verdrahtet** sie in `main.go`. Kein Agent editiert `main.go`.

---

## 3. Verbindliche Subagent-Regeln (aus den Memory-Lektionen)

- **`isolation: "worktree"` immer.** „do not push"-Text ist KEINE Grenze (Welle-2-Beleg: 2/3 pushten trotzdem). Worktree erzwingt Isolation physisch.
- **Kein Commit/Push durch Agenten.** Hauptsession committet ALLES. Nach der Welle: `git worktree list` prüfen + `TaskStop` + Commit-Dateiliste **gegen den Agent-Report abgleichen** (Memory: Background-Agent-Duplikate im Haupt-Tree).
- **„grün ≠ korrekt" — explizit gegenprüfen** (Memory-Kernlektion). Der Agent-Claim „build/test grün" verbirgt regelmäßig:
  - **gRPC-Layer-Umgehung**: Handler ruft die Service-Impl **direkt** im Gateway statt über den gRPC-Client → **RLS-Bypass + Phantom-404**. → Verify: neuer Handler geht über `<svc>Client.<RPC>(ctx, …)`, nicht über eine direkt injizierte Service-Instanz.
  - **Stub statt Impl** (`Unimplemented`, leerer Return, Fake-Bytes).
  - **Nicht-regeneriertes Proto** (Agent editiert `.proto`, `.pb.go` bleibt alt).
  - **Fehlende Seed-Migration** zu einem neuen `RequirePermission`-Guard → **403 für ALLE inkl. Admin** (Memory: Permission-Seed-Pflicht).
- **nullable-tenant_id-Audit** bei jeder neuen Tabelle: Schema `tenant_id UUID NOT NULL` + RLS-Policy **und** Repo-`INSERT`-Wiring **und** `SELECT`-Scan (Read-Seite → sonst Phantom-404). Oder explizit in die System-Global-Liste (ADR-006).
- **Neue `config.RequireX`-Assertion = Prod-Deploy-Hazard** (COSMI_ENV=production live + CD auto-deployt). Reihenfolge: erst Compose-Passthrough (`${VAR:-}`) + `.env.production`-Wert, DANN Assertion-Commit. Agenten **fügen keine `config.RequireX` hinzu** ohne Vermerk im Report → Hauptsession rollt es kontrolliert aus.
- **Neuer Modul-Guard/Feature-Flag**: `modules.*` default OFF (`COSMI_MODULE_<NAME>_ENABLED`) — sonst 404 + FE-Nav blendet aus. Im Report vermerken.
- **Conclusions, not transcripts.** Report = was gebaut, welche Dateien, welche Serialisierungs-Punkte offen (Proto/Migration/Registry), welche Verify-Schritte die Hauptsession noch fahren muss. Keine Datei-Dumps.
- **Max 3 Agenten parallel.** Self-contained (Agenten können nicht nachfragen).
- **Model:** Sonnet-Baseline. Opus nur mit Begründung im Zuschnitt.

**Gate-Split — was Agenten im Worktree KÖNNEN vs. was die Hauptsession macht:**

| Agent (im Worktree, Go-Module-Cache geteilt) | Hauptsession (auf gemergtem Tree, mit DB/protoc) |
|---|---|
| `go build ./...` · `go vet ./...` | Proto-Regen (protoc) |
| `golangci-lint run` (config `backend/.golangci.yml`, v2.8) | Migration ausführen (`migrate` gegen lokale DB) |
| Unit-Tests (nicht-DB) `go test ./internal/<svc>/...` | pgtc-Integrationstests (brauchen Postgres) |
| Wire-Shape gegen FE-Typ prüfen (lesen) | **RLS-Smoke** (kmuhub_app-Rolle, cross-tenant-Read = 0 Zeilen) |
| | **gRPC-Layer-Check** + Seed-vs-Guard-Diff + Route-Registrierung |

> **Backend-Vorteil ggü. FE-Wellen:** Der Go-Module-Cache (`$HOME/go/pkg/mod`) ist geteilt → Worktree-Agenten können `go build`/`vet`/`lint`/Unit-Tests **wirklich laufen** (anders als FE-Worktrees ohne `node_modules`). Nur DB/Proto bleibt Hauptsession.

---

## 4. Der Wellenplan (phasiert, launch-kritisch)

Reihenfolge ist bindend — spätere Phasen hängen an früheren.

### Phase 0 — Cross-cutting Foundations · **seriell** (Hauptsession oder 1 Agent)
Zwei Dienste entsperren die meisten anderen Module. Bewusst NICHT parallel (beide sind geteilte Infrastruktur).
- **0a — Generischer S3/MinIO-Upload-Service.** Ein Upload-Endpoint (multipart → presign → PUT → PATCH `*_url`) für Foto-/Datei-Anhänge. Entsperrt fuhrpark(Schaden)/inventar(Bewegung)/rapporte(Doku)/vermietung(Protokoll)/chat/profil-Avatar/admin-Branding. Presign-Infra existiert bereits (`1aef2f45`, `MINIO_PUBLIC_ENDPOINT`) — hier nur der generische, wiederverwendbare Wrapper + einheitliche Route.
- **0b — Server-PDF-Service.** Ein Render-Dienst (chromedp/gotenberg headless → `application/pdf`) ersetzt die 4 Stubs/window.print (rapporte/vertraege/advisory/berichte). Token-geschützte Render-URL, Schriften eingebettet.

> Grund für seriell: beides sind geteilte Bausteine, an denen Phase-3-Agenten andocken. Erst bauen, dann fächern.

### Phase 1 — RBAC-Fundament · **seriell** (ein kohärentes System, NICHT partitionierbar)
Das Datenmodell + der Resolver sind EINE Einheit. Ein Agent (oder Hauptsession) baut sie am Stück. **Wahrscheinlich Proto-Touch an `auth.proto`** (`GetEffectivePermissions`-RPC) → als Proto-Vor-Welle behandeln.
- **1a — Datenmodell (Migration 000244+):** `roles.tenant_id UUID NULL` (NULL = System-Preset) + `roles.based_on UUID NULL` + `role_permissions.scope VARCHAR` (`own|team|all`, default `all`). Presets unveränderlich. 7 Preset-Rollen seeden (`admin·it_admin·hr_admin·manager·member·readonly·extern`), Mapping `hr→hr_admin`/`it_support→it_admin` (Detail: `backend-gaps.md` §RBAC + FE `mocks/data/rbac.ts` `ROLE_DEFS` = Seed-Vorlage).
- **1b — `GET /api/v1/auth/me/permissions`** — aufgelöste effektive Rechte (Union aller Rollen, weitester Scope gewinnt, `sources` kumulieren). Default-Deny, kein Wildcard. **Die eine Quelle fürs FE-Gating.** Contract FINAL in `backend-gaps.md` Z.30-37.
- **1c — Rollen-Admin-API:** `GET/POST/PATCH/DELETE /api/v1/admin/roles` + `GET/PUT /admin/roles/{id}/permissions` (Matrix). Create = immer Klon von `based_on`. Fehler-Codes (`preset_immutable`/`role_limit_reached`/`role_name_exists`/`role_has_members`/`last_admin`/`not_found`). Custom-Limit 20/Tenant. Contract FINAL in `backend-gaps.md` Z.22-28.
- **1d — Server-Guardrails:** Mindestens-1-Admin · Selbst-Aussperr-Schutz · Privilege-Escalation-Guard · Validator-Entkopplung (`assignRoleRequest oneof=admin manager member` → dynamisch gegen `roles`-Tabelle, `route_auth.go`).
- **1e — Audit-Events** für Rechteänderungen (append-only, kann auf `audit_log`/Migr. 000222 aufsetzen).

> **Prerequisite für Phase 2**: die `scope`-Spalte + der Resolver müssen stehen, bevor Module scope-tragende Seeds bekommen.

### Phase 2 — RBAC-Enforcement pro Modul · **PARALLEL** (hier fächern die Wellen)
Pro Modul dieselbe self-contained Aufgabe: **Seeds (Katalog-Keys aus dem FE-SSOT spiegeln) + `RequirePermission`-Granularität an den Routen + Owner-FK + Listen-Filter fürs `own`-Scope.** FE-SSOT = `desktop/src/renderer/src/config/capability-catalog.ts` + `ROLE_DEFS` in `mocks/data/rbac.ts` (= gewünschter Seed-Inhalt).

**Priorisierung nach Seed-Lücke (launch-kritisch zuerst):**
- **Welle 2a (härteste Lücken, 3 Agenten):** `wiki` (NULL Seeds) · `zeiterfassung` (NULL Seeds + Genehmigungen/DATEV-Export müssen serverseitig gegated) · `infrastructure` (NULL Seeds).
- **Welle 2b (nur grob vorhanden, 3 Agenten):** `helpdesk` (Requester-Modell + Listen-Filter serverseitig) · `kommunikation`/chat (channel:manage + webhook:manage, keine chat-Seeds) · `kalender` (`booking-pages` Bindestrich vs. FE-Underscore-Mapping + category:manage).
- **Welle 2c (fein nachziehen, 3 Agenten):** je ein Batch aus {work·documents·crm·finance} / {inventar·einkauf·produktion·vertraege} / {schichten·fuhrpark·vermietung·rapporte·dialer·berichte·formulare·automatisierung} — Katalog kuratiert, aber BE kennt nur grobe read/write-Paare (Detail je Modul in `backend-gaps.md` R-3-Batch-1…5-Blöcke).
- **Welle 2d (HR-Tiefe, 1-2 Agenten):** Reporting-Line-Resolver serverseitig + Change-Request-Endpoints + Offboard-Kaskade (`backend-gaps.md` R-4-Block).

> **Owner-FK-Muster** (wiederkehrend): viele Module tragen kein `created_by`/`owner_id` → `own`-Scope unmöglich. Wo FE `own` will (rapporte/berichte/formulare/schichten-swap/helpdesk): `created_by`/`author_id` beim Create **aus dem Auth-Context** befüllen + **Liste bei scope=own serverseitig filtern** (nicht nur 403 am Fremd-Detail).

### Phase 3 — Feature-Endpoints + Contract-Fixes · **PARALLEL, service-partitioniert**
Reine Service-Grenzen, minimale Kollision. Je Agent ein Service:
- `zeiterfassung`: `/hr/time/balance·entries·projects·analytics·team` + `time_week_submissions`-Tabelle + Wochen-Freigabe.
- `admin`: tenant-provisioning · billing/license-Service · user-invite-Flow (Token, Seat-Konsum) · branding→S3 (nutzt Phase 0a).
- `inbox`: status/thread/tags/forward-RPCs + canned-CRUD (**Tabellen liegen schon**: Migr. 000237/000238).
- `berichte`: echter KPI-Service + Server-PDF (nutzt Phase 0b) + Cron-Scheduler+Mailer + Share-Token-Public-Page.
- `finance`: recurring invoices · OP-Liste · mehrstufiges Mahnwesen · CAMT.053/MT940-Import · **Liste-0,00-€-Fix (#17)**.
- `document`: List-Handler konsistent `{key,total}` + leere Slices `[]` statt `null` + Single-Entity wrappen (`route_document.go`).
- `security`: 31 Endpoints in `openapi.yaml` nachziehen (X-3-Spec-Lücke) + Pfad-/Shape-Abgleich.
- `helpdesk`: WireTicket um `assignee_name`/`requester_name`/`description`/`category`/`ticket_number`-Sequenz.
- `einkauf`: `total_amount`-Recompute + `POST /pos/{id}/cancel` + `ExportPO`-Stub löschen.

### Phase 4 — Branchen-BE · **ausgeklammert** (Post-Launch, Solar-Pilot Nov)
Nicht Teil dieser Strecke. Bei Bedarf eigenes Briefing.

---

## 5. Abhängigkeits-Fahrplan (die Reihenfolge)

```
Proto-Vor-Welle (auth.proto GetEffectivePermissions)  ──┐  [Hauptsession, seriell]
Phase 0a S3-Upload ─┐                                     │
Phase 0b PDF-Service ┘  [seriell]                         │
                        └──► Phase 1 RBAC-Fundament ──────┘  [seriell, 1 Agent]
                                    └──► Phase 2 Enforcement  [PARALLEL 2a→2b→2c→2d, je 3 Agenten]
Phase 3 Feature/Contract  [PARALLEL, unabhängig von RBAC — kann ab Phase 0 laufen]
```

**Nebenläufigkeit:** Phase 3 (Feature/Contract) hängt NICHT an RBAC → kann parallel zu Phase 1/2 laufen, sobald Phase 0 steht. Praktisch: nach Phase 0 zwei Stränge gleichzeitig (RBAC-Strang seriell + Feature-Strang parallel), begrenzt durch den 3-Agenten-Cap.

---

## 6. Per-Agent-Prompt-Template (kopierbar)

> Pro Agent ein Block. `{{PLATZHALTER}}` füllt die Hauptsession. Alles Nötige muss drinstehen — der Agent kann nicht nachfragen.

```
ROLLE: Du bist ein Backend-Agent für das KMU-Hub-CRM (Go, gRPC-Microservices).
Arbeite AUSSCHLIESSLICH in deinem Worktree. Du committest NICHT, pushst NICHT, mergst NICHT.

AUFGABE ({{SERVICE}} / {{PHASE}}):
{{KONKRETE_AUFGABE — was gebaut wird, welche Endpoints/Tabellen/Guards}}

QUELLEN (lesen vor dem Bauen):
- Detail-Backlog: .planning/backend-gaps.md §{{ABSCHNITT}}
- FE-Contract/SSOT: {{z.B. desktop/src/renderer/src/config/capability-catalog.ts + mocks/data/rbac.ts}}
- Bestehendes Muster im Repo: {{z.B. backend/internal/gateway/route_booking.go für RequirePermission}}

ARCHITEKTUR-REGELN (verbindlich):
- Thick services, thin handlers. Business-Logik in internal/{{svc}}/service.go, Handler nur Parse/Call/Respond.
- Handler geht IMMER über den gRPC-Client (<svc>Client.<RPC>(ctx, …)) — NIE direkt eine Service-Instanz
  im Gateway aufrufen (das umgeht RLS → RLS-Bypass/Phantom-404). Das ist ein Fehlschlag-Kriterium.
- structured logging (slog), kein fmt.Println. Prepared statements. Input-Validierung an der Grenze.
- Neue Tabelle: tenant_id UUID NOT NULL + RLS-Policy + INSERT-Wiring (tenant aus middleware.GetTenantID(ctx))
  + SELECT-Scan tenant-gefiltert. Kein nacktes SELECT ohne tenant-Scope.

SERIALISIERUNGS-GRENZEN (NICHT anfassen — der Hauptsession melden):
- Migrations-Datei: Lege sie mit der Nummer {{MIGRATION_NR}} an ({{name}}.up.sql + .down.sql), aber
  FÜHRE SIE NICHT AUS (keine DB). Forward-only, kein Ändern bestehender Migrationen.
- Proto: {{"Kein Proto-Touch nötig." ODER "Falls du .proto ändern musst: NUR die .proto editieren,
  NICHT regenerieren — im Report melden, die Hauptsession regeneriert zentral."}}
- cmd/gateway/main.go, go.mod, featureflag/registry.go NICHT editieren. Lege deine Route-Datei als
  route_{{modul}}_{{xyz}}.go an; die Registrierung in main.go macht die Hauptsession.
- Neue RequirePermission-Guards brauchen einen Seed in DEINER Migration (sonst 403 für alle inkl. Admin).
- Keine neue config.RequireX-Assertion ohne expliziten Report-Vermerk.

GATE (musst du selbst grün bekommen, im Worktree):
- go build ./...  UND  go vet ./...  UND  golangci-lint run (config backend/.golangci.yml)
- go test ./internal/{{svc}}/...  (Unit; DB-Integrationstests darfst du nicht laufen — im Report vermerken)
- Wire-Shape deiner Responses gegen den FE-Typ prüfen (gewrappt {key,total}, leere Slice = [], nicht null;
  snake_case; Timestamps als {seconds,nanos} nur wenn response.Proto — sonst response.JSON).

RÜCKGABE (Conclusions, not transcripts):
1. Was gebaut (Endpoints/RPCs/Tabellen/Guards), Dateiliste (exakt — für den Commit-Abgleich).
2. Offene Serialisierungs-Punkte: Migration {{NR}} anzulegen? Proto-Regen nötig? Route-Registrierung wo?
3. Was die Hauptsession noch verifizieren/laufen muss (Migration apply, pgtc, RLS-Smoke, Proto-Regen).
4. Gate-Status (build/vet/lint/unit — je grün/rot mit Grund).
5. Abweichungen vom Plan + Annahmen, die du treffen musstest.
```

---

## 7. Verifikations-Gate — Hauptsession, nach JEDER Welle

„grün ≠ korrekt" — der Agent-Report ist ein Vorschlag, kein Beweis. Vor dem Commit:

1. **Git-Hygiene:** `git worktree list` — jeder Agent-Worktree da? `TaskStop` für fertige Agenten. **Commit-Dateiliste gegen jeden Report abgleichen** (Duplikat-Agenten/Streu-Dateien im Haupt-Tree?). Explizit stagen, nichts blind `add -A`.
2. **Serialisierung einsammeln:** Migrations-Nummern kollisionsfrei + fortlaufend ab 000244? Proto-Touch → **jetzt zentral regenerieren** (protoc-Pfad §2) + `.pb.go`-Diff prüfen. Route-Dateien in `cmd/gateway/main.go` verdrahten.
3. **gRPC-Layer-Check:** neue Handler rufen `<svc>Client.<RPC>`, **nicht** eine direkt injizierte Service-Instanz. (Grep die neuen Handler.)
4. **Seed-vs-Guard-Diff:** jeder neue `RequirePermission("x","y")` hat einen passenden Seed in einer Migration? (Sonst 403 für alle.)
5. **nullable-tenant_id-Audit:** neue Tabellen `NOT NULL` + RLS-Policy + INSERT setzt tenant + SELECT scoped.
6. **Gate auf gemergtem Tree:** `go build ./...` + `go vet` + `golangci-lint` + relevante `go test` (inkl. pgtc gegen lokale DB). Nicht auf einem gestashten Zwischenstand.
7. **RLS-Smoke:** als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) — cross-tenant-Read liefert 0 Zeilen.
8. **Ein Commit pro Welle** (Conventional Commit), dann Push. **Pause fürs Review-Gate** (dein Go/No-Go), erst dann nächste Welle.

**Deploy-Awareness:** COSMI_ENV=production ist live + CD auto-deployt bei Push auf `main`. Neue `config.RequireX` oder neuer `modules.*`-Flag → erst Compose-Passthrough + `.env.production`-Werte (Hetzner), DANN pushen. Sonst Service-Crash-Loop.

---

## 8. Startpunkt für die Hauptsession

1. `git pull` (frischen Stand ziehen).
2. Entscheiden: startest du mit **Proto-Vor-Welle + Phase 0** (Foundations) oder ziehst du den **Feature-Strang (Phase 3)** parallel vor? Empfehlung: Phase 0 seriell zuerst (entsperrt am meisten), dann RBAC-Strang + Feature-Strang nebenläufig unter dem 3-Agenten-Cap.
3. Migrations-Nummern-Block für die erste Welle festlegen (ab 000244).
4. Per-Agent-Prompts (§6) ausfüllen, Agenten mit `isolation: "worktree"` spawnen (max 3).
5. Nach der Welle: §7 durchgehen → Commit → Push → Pause.
