# Cosmi — Kern-Roadmap

> **Status (2026-08-11):** **Sprint 0–4 abgeschlossen, Sprint 5 laeuft** (Pre-Launch-Audit + Rigorosum R3, bis 08-31). Noch **21 Tage bis Launch**. Parallel zu S5 laufen die **Backend-Nachtlaeufe** (seit 07-26, Migrationen **243–310**): Feature-Nachzug quer durch die Module, RLS-Welle (`knownRLSGaps` seither leer), RBAC Phase 1, ~110 neue REST-Pfade, zuletzt zwei reine Coverage-Laeufe. Laeufe 1–8 sind gemergt und deployt — **Prod-Kopf = Repo-Kopf 310 clean**, 30 von 36 Containern healthy (0 unhealthy). **Der Coverage-Engpass ist geschlossen:** Lauf 8 hob 47,7 → **60,0 %**, `biz` 48 → **70,6 %** und `crm` 51 → **71,7 %** liegen damit ueber dem 60-%-Ziel. Der Engpass ist jetzt **Korrektheit statt Abdeckung**: derselbe Lauf foerderte zehn verifizierte Produktionsbugs zutage, ausgerechnet in den Paketen mit der hoechsten Coverage. Lauf 9 (seit 11-08 16:00) arbeitet sie ab. Einziger echter Launch-Blocker bleibt **Legal (AVV/DPA)**. Gemessener Ist-Stand mit allen Zahlen: [`.planning/status-overview.md`](../.planning/status-overview.md).
>
> <details><summary>Frueherer Status (2026-04-26)</summary>
>
> Sprint 1 abgeschlossen + Sprint 2 Welle 0+1 done. 4 commits auf main: 2245ecb (Welle-0 FK-Fixes), 9438ba0 (4 Module backend), e4b98b9 (3 Sprint-1-Carry-Items), ad04191 (20-Blocker-Bugfix-Sweep aus Welle-0+1-Review). Auf Kurs fuer Launch 01.09.
> </details>
> **Ein-Launch-Modell** (Stand 2026-06-28, Playbook: `docs/BACKEND-LAUNCH-PLAN.md`): Pilot-0 und volle P0-Feature-Parität fallen auf **2026-09-01** zusammen — das frühere Zwei-Deadline-Modell (Pilot-0 01.07 / volle P0 01.09) ist aufgelöst.
> - **Launch: 2026-09-01** — ZFA-Pilot-tauglicher Kern (Kalender-Booking, Dialer-Consent, Passwort-Reset, korrekte Demo-Daten) **plus** voller P0-Scope: E-Rechnung/GoBD/DATEV/Bexio + Finance-Block-Wellen 4–7.
> **UG-Gruendung:** 2026-06-01 bleibt, Pilot-Tag separat
> **Konsolidiert aus:** ROADMAP (alt), BUSINESS-ROADMAP, PRODUCT-STRATEGY, DIALER-ROADMAP, I18N-ROADMAP, PERFORMANCE-PLAN, .knowledge/milestones.md
> **Eigentuemer dieser Datei:** Luke. Jede andere Roadmap-Datei ist SUPERSEDED.

---

## 1. Context & North Star

### Warum diese Roadmap

Bis 2026-04-18 lagen Produkt-, Technik- und Business-Pfade auf 11 verschiedene Dokumente verteilt. Zwei Rigorosum-Runden am 18.04. haben harte Defizite aufgedeckt: **Runde 1** (Gesamtnote 3.3, wild-wren-Plan) identifizierte 7 P0-Launch-Blocker in Backend/Frontend/Ops. **Runde 2 Vertiefung** (Gesamtnote 4.1, functional-seahorse-Plan) lieferte 9 zusaetzliche P0-Blocker in Integrationen, Realtime-Kern und DB-Schema — darunter ein komplett fehlender TURN/STUN-Server und ein faktisch wirkungsloser Recording-Consent-Check. Kombinierte Launch-Reife: **3.7**.

Konsequenz: Launch von 01.06. auf 01.07. (Runde 2, +4 Wochen), dann auf **01.09. verschoben** — um 16 P0-Blocker + Option-B-Full-Retrofit (~50 Tabellen) + finance_line_items-Normalisierung + den vollen Finance-Block (Wellen 4–7) sauber umzusetzen. UG-Gruendung bleibt auf 01.06.

Diese Datei ist die einzige gueltige Roadmap bis zum Launch. Alle anderen werden deprecatet (siehe §8).

### North Star

**Cosmi geht am 2026-09-01 live** (ein Launch, Playbook: `docs/BACKEND-LAUNCH-PLAN.md`):
- **Launch (2026-09-01):** ZFA-tauglicher Kern — alle 14 Module, keine Mock-Daten, Option-B aktiv, Pilot-kritische Flows (Terminbuchung, Dialer-Consent, Passwort-Reset) abgesichert — **plus** volle P0-Feature-Parität: E-Rechnung/GoBD/DATEV/Bexio, vollständige Finance-Block-Wellen 4–7.

**Der Launch am 2026-09-01 erfordert:**
- **14 echten Modulen** (keine Mock-Daten mehr in user-sichtbaren Pfaden)
- **Multi-Tenancy Option-B aktiv** (RLS auf ~50 Tabellen, Instanz-pro-Pilot + tenant_id-Isolation — kein Downgrade-Risiko)
- **DSGVO-Consent-Enforcement** in allen Send-Flows (Email, Dialer) + Realtime-Recording (Join-with-Consent + persistenter Banner)
- **Sicherheits-Posture auf Pilot-Niveau** (16 P0-Fixes aus Rigorosum Runde 1 + 2 erledigt)
- **TURN/STUN self-hosted** (coturn auf eigenem CAX11, kein Vendor-Lock)
- **WASM-Plugin-System Feature-Flag OFF** — Config-Plugins aktiv, WASM-Haertung in Phase D (ehrlicher Pitch)
- **`finance_invoices.line_items` normalisiert** (eigene `finance_invoice_lines`-Tabelle, GoBD/ZUGFeRD-tauglich, Finance-Test-Coverage erweitert)
- **Zweiter Review-Zyklus abgeschlossen** (Sprint 5, Peer-Review + Rigorosum Runde 3)
- **Ehrlicher Pitch:** Mobile = PWA auf Desktop-Basis, keine falschen Native-Versprechen

**Erst zum 2026-09-01 (volle P0) kommen hinzu:**
- E-Rechnung (XRechnung/ZUGFeRD), GoBD-Belegarchiv WORM, DATEV-EXTF-Export, Bexio-OAuth-Sync

### Pilot-Strategie

- **Pilot-0 (ZFA, Anfang Juli):** Warm-Einstieg, Design-Partner, Dialer-Nutzung, kostenlos
- **Pilot 1–3 (Juli–August, Dienstleister-Segment):** Kostenlos, Feedback-Loop woechentlich, Referenzen aufbauen
- **Handwerk-Piloten:** ab Oktober 2026, wenn Rapporte/Schichten/Fuhrpark auf Branchen-Daten kalibriert sind

---

## 2. Aktueller Stand

### Gemessen am 2026-08-11

Alle Zahlen selbst erhoben, nicht aus Doku uebernommen. Vollstaendige Fassung mit Messkommandos:
[`.planning/status-overview.md`](../.planning/status-overview.md).

| Bereich | Stand |
|---|---|
| Backend | 24 Services (23 µSvc + Gateway), **1.154 gRPC-RPCs**, **836 OpenAPI-Pfade** / 1.192 Operationen |
| Datenbank | Migrationskopf **310**, Prod identisch und clean; RLS produktiv, **`knownRLSGaps` leer** |
| Tests | 711 Test-Dateien, **Coverage 60,0 %** (CI-Gate 15 %). security 79,5 % ✅ · crm 71,7 % ✅ · biz 70,6 % ✅ · auth 67,9 % ✅ · ⚠ **gateway 46,0 %** (schwaechstes Kernpaket) |
| Frontend | **34 Module**, 81 API-Hook-Dateien (993 Hooks), 1.234 TS/TSX-Dateien; **17 von 17 operativen Modulen voll gewired**, kein Mock-Seed mehr im Produktionspfad |
| i18n | **12.072 Keys × 4 Sprachen, Paritaet vollstaendig**, BOM entfernt, per Test gepinnt |
| Feature-Flags | 17, 16 davon default OFF |
| Produktion | Hetzner CPX42, `app.zentria.tech`, 30 von 36 Containern healthy (0 unhealthy), `COSMI_ENV=production` scharf |

**Was seit dem Juni-Stand dazukam:** acht Backend-Nachtlaeufe (Migr. 243–310) mit RLS-Welle,
RBAC Phase 1, Partitionierung (242), rund 110 neuen REST-Pfaden und zuletzt zwei reinen
Coverage-Laeufen (30,2 → 47,7 → 60,0 %); die FE↔Backend-Wiring-Wellen sind durch.

**Was offen ist** (Details in `.planning/status-overview.md` §4): zehn verifizierte
Produktionsbugs aus Lauf 8, in Arbeit in Lauf 9 — schwerster ist ein Typ-Scan-Fehler in
`security/audit`, der Audit-Viewer, Export und Chain-Verifikation fuer jeden aktiven Tenant
lahmlegt (Compliance) · `gateway`-Coverage bei 46 % · 118 TypeScript-Fehler im Desktop
(Vorbestand, auch Produktionscode) · **Electron 33.4.11 mit 34 High-Advisories**, von
`npm audit --omit=dev` ausgeblendet obwohl es ausgeliefert wird · CSAT bleibt stillgelegt
(„Public Web Surface") · **Legal (AVV/DPA)** als einziger echter Launch-Blocker.

**Am 2026-08-11 geschlossen:** Mock-Seed in vier Zustand-Stores (`timetracking` und `team` waren
ungegatet fuer jeden Nutzer erreichbar, `team` mit erfundenen Gehaltsdaten) · `scans.yml` wieder
gruen (react-router 7.18.2 + dompurify 3.4.13) · MinIO-Backup lief nie, weil das MinIO-Image kein
`tar` enthaelt — jetzt ueber einen Sidecar · i18n-Paritaet fr/it · Passwort-Reset-Link zeigte ins
Leere (kein GET-Handler unter `/reset-password`), jetzt eine eingebettete Seite im Gateway.

### Was fertig ist (Stand Rigorosum 18.04., historisch)

| Bereich | Status |
|---|---|
| 20 Feature-Phasen (Auth, CRM, Chat, Video, Kalender, Work, Email, Finanzen, HR, Automation, Plugin-System) | ✅ Backend + Frontend komplett |
| Beta Phase A + B1–B10 (UI-Hardening, Crash-Fixes, Design Audit, Rebrand Cosmi, Locale de-DE) | ✅ |
| i18n-Migration (7.221 Keys × 4 Sprachen) | ✅ MIT CAVEAT: alle 18 ICU-Plurals sind kaputt (siehe P0.7) |
| Performance-Sprint (N+1 weg, Chunk-Splitting, React Compiler, V8 Cache) | ✅ (Redis-Caching offen, Post-Launch) |
| Dialer Phase 1 (27 RPCs, 5 Migrations, 25 REST-Endpoints, WebRTC-Workspace) | ✅ Feature-komplett, 12.4% Test-Coverage = Risiko |
| Infrastruktur Hetzner CPX42 (11 Services healthy, HTTPS, Backup-Cron, Prometheus) | ✅ |
| Website zentria.tech (Astro 5, Vercel) | ⚠ "Viele Versprechungen, wenig dahinter" — Inhalts-Cleanup noetig |

### Was aus den Rigorosum-Runden zaehlt

**Runde 1 (wild-wren, Gesamtnote 3.3):**
- Kapitel-Noten: Backend 3.3 · Frontend 3.3 · Mobile 5.0 (leer) · Ops/Sec 2.7 · Tests 4.0 · Launch-Readiness 3.7
- 7 P0-Launch-Blocker + 8 P1-Items (siehe §4)
- Vollstaendig: `~/.claude/plans/wir-hatten-ja-schonmal-wild-wren.md`

**Runde 2 Vertiefung (functional-seahorse, Gesamtnote 4.1):**
- Kapitel-Noten: Integrationen 4.1 · Realtime-Kern 4.5 (schlechteste Zone) · DB-Schema 3.8
- 9 neue P0-Launch-Blocker (R2-P0.1–9, siehe §4): TURN-Config, LiveKit-Secrets, Recording-Consent-Bug, Frontend-Consent-Modal, Egress-Webhook, Lexware-HMAC, Offline-Queue, `consent_records.created_by`-FK, `gdpr_deletion_requests`-zirkulaere-FK
- 12 P1 + 15 P2 + 6 P3 zusaetzlich
- Vollstaendig: `~/.claude/plans/du-bist-ein-schlecht-functional-seahorse.md`

**Kombinierte Launch-Reife: 3.7** (nach oben korrigiert von Runde-1-3.3, weil Runde 2 neue Blocker ans Licht holte).

**Andere Kern-Entscheidungen:**
- 14 Frontend-only-Module — werden jetzt alle echt (siehe §5)
- Mobile-Ordner leer — wird geloescht, Pitch korrigiert (siehe §3 Sprint 0)
- Multi-Tenancy **Option-B voll durchgezogen** — ~50 Tabellen bekommen `tenant_id` + RLS-Policies in Sprint 2+3
- WASM-Plugin-System **Feature-Flag OFF** bis Phase D — Config-Plugins aktiv, keine WASM-Runtime in Production
- `finance_invoices.line_items` **vor Launch normalisiert** in eigene Tabelle

### Was sich geaendert hat

| Alt | Neu |
|---|---|
| Launch 01.05. → 01.06. → 01.07. | **Ein Launch: 01.09.** (Pilot-0 + volle P0-Parität zusammengelegt; vormals Zwei-Deadline-Modell nach Runde 2 + Dariens Handover; Playbook: `docs/BACKEND-LAUNCH-PLAN.md`) |
| "11 Industry-Module bleiben auf Mock" (alte ROADMAP §Scope-Entscheidungen) | **Alle 14 Module werden echt** |
| "React Native Scaffold existiert" (alter Audit) | **Mobile-Ordner leer, wird geloescht** |
| Multi-Tenancy Option-A permanent | **Option-B-Full jetzt, ~50 Tabellen Retrofit** |
| WASM-Plugin-System aktiv | **Feature-Flag OFF bis Phase D** (ehrlicher Pitch) |
| Recording ohne Consent-Fluss | **Join-with-Consent + persistenter Banner** |
| TURN/STUN nicht konfiguriert | **coturn self-hosted auf CAX11** |
| `finance_invoices.line_items` JSONB | **normalisierte Tabelle + Backfill** |
| Moritz als GTM-Lead | "In der Schwebe" (siehe MEMORY project_team_ug.md) |
| Demo-Theater als Feature | **Demo-Theater als Launch-Risiko** |

---

## 3. Sprint-Plan bis 2026-09-01

**10 Wochen = 6 Sprints. Parallelitaet via Git-Worktrees + Sub-Agenten (Sonnet fuer Code-Volumen, Opus fuer Architektur-Entscheidungen).**

### Sprint 0 — Runde-1-Launch-Blocker (2026-04-21 – 2026-04-27, 1 Woche) ✅ ABGESCHLOSSEN 2026-04-18

**Ergebnis:** Alle 7 R1-P0-Blocker + Cleanup + Modul-Scope-Matrix in 3 Wellen parallel erledigt, 9 PRs gemerged.

| # | Task | Aufwand | Status | PR |
|---|---|---|---|---|
| S0.1 | Migration 000075: `consent_records.contact_id` CASCADE → SET NULL | 4h | ✅ Done | #5 |
| S0.2 | `assertConsent()`-Wrapper vor `SendEmail` + `InitiateDialerCall` | 3d | ✅ Done | #10 |
| S0.3 | `WOPI_JWT_SECRET`, `MINIO_SECRET_KEY`, `VAULT_MASTER_SECRET` mit `required: true` + Startup-Assertion | 2h | ✅ Done | #6 |
| S0.4 | DOMPurify in 5 `dangerouslySetInnerHTML`-Stellen (Mails, Wiki, Template, Signature, IT-Admin) | 1d | ✅ Done | #9 |
| S0.5 | OnlyOffice JWT in Prod-Override explizit `JWT_ENABLED: "true"` + Secret-Sync | 2h | ✅ Done | #7 |
| S0.6 | Feature-Flag-Registry + `GET /api/v1/feature-flags` + `useFeatureFlags`/`FeatureGate` + **WASM-Plugin-System Feature-Flag OFF** (Build-Tag `!no_wasm`) | 2d | ✅ Done | #11 |
| S0.7 | ICU-Plural-Klammern-Fix in de.json/en.json/fr.json/it.json (18 Strings × 4 Sprachen) | 1h | ✅ Done | #3 |
| S0.8 | `mobile/`-Ordner loeschen, Pitch korrigiert (PWA statt Native) | 2h | ✅ Done | #4 |
| S0.9 | 14-Modul-Backend-Scope-Matrix (`docs/MODULES_SCOPE_MATRIX.md`) | 1d | ✅ Done | #8 |

**Abgeschlossen:** Alle 7 R1-P0 + R2-P1.2 (WASM-OFF, zusammen mit S0.6) gefixt. Feature-Flag-System live mit 17 Flags (14 Module + `plugins.wasm`/`plugins.config`/`plugins.api`; `plugins.api` kam in Audit-Response 2026-05-10 dazu). Modul-Scope final in `docs/MODULES_SCOPE_MATRIX.md`.

---

### Sprint 1 — Backend-Offensive Teil 1 + R2-P0 Batch A (2026-04-28 – 2026-05-10, 2 Wochen)

**Ziel:** 7 von 14 Modulen echt + die fuenf teuersten R2-P0-Items.

| # | Task | Aufwand | Kategorie | Status |
|---|---|---|---|---|
| S1.1 | **wiki** (Postgres-FTS, TipTap, Versionen, Share-Links) | 3d | Modul | ✅ Done 2026-04-18 — 15 RPCs, FTS tsvector+GIN, Coverage 38.2% |
| S1.2 | **berichte** (BI-Aggregations-Service, Views, CSV/PDF/XLSX-Export, Scheduled Reports, Recharts) | 3d | Modul | ✅ Done 2026-04-19 — Alle 11 WPs abgeschlossen. Wave 1–3: WP-3 Export-Layer (80.2% Cov), WP-5 gRPC+cmd (77.6% Cov), WP-6 Gateway-Routes + ACL-Seed 000080 (57% Cov), WP-7 Docker-Compose, WP-11 Flag+Smoke. Ports 50063/9103. Plan `docs/SPRINT1_BERICHTE.md`. |
| S1.3 | **formulare** (Form-Schema JSONB, Submissions, Webhook-Trigger) | 4d | Modul | ✅ Done 2026-04-19 — 18 RPCs (Schema+Submission+Webhook+Delivery+Stats), Migration 000081 (4 Tabellen), Webhook-Worker (HMAC-SHA256, Exp-Backoff 30s→2h, Dead-Letter), CSV+XLSX Export, Feature-Flag modules.formulare. Ports 50064/9104. |
| S1.4 | **helpdesk** (Tickets, Agenten, Canned, Merge) | 4d | Modul | ✅ Done 2026-04-18 — 22 RPCs, SLA + Merge, Coverage 39.3% |
| S1.5 | **vertraege** (Laufzeit-Engine, Erinnerungs-Trigger, Skribble-Placeholder) | 3d | Modul | ✅ Done 2026-04-26 (`9438ba0`) — Port 50073, Migration 000089+000090, Reminder-Worker (5+60min Ticker, advisory-lock-claim), 36.7% Coverage |
| S1.6 | **buchhaltung** Completion (FinanzenHook-Gaps, GoBD-Journal) | 2d | Modul | ✅ Done 2026-04-26 (`e4b98b9`) — 7 GoBD-RPCs (JournalSummary, ValidateInvoiceNumber, LockInvoice, PaymentStats, UpdateDunningStatus, SendDunningNotice [Sprint-3-Email-Stub], GenerateGoBDExport). LockInvoice-Bypass in 5 Service-Methoden geguardet (Bugfix-Sweep `ad04191`). CSV-Format-Version `"GoBD-Sprint3-Preview-NotYetCompliant"` mit TODO fuer Pflichtfelder |
| S1.7 | **video** Completion (`useVideo`-Hook, Recording-Tagging) | 2d | Modul | ✅ Done 2026-04-26 (`e4b98b9`) — Migration 000091 (consent_snapshot NOT NULL via batched-backfill+VALIDATE-Pattern), 6 Recording-Tag-RPCs, 4 neue Hooks. DeleteRecording-Bug + Tag-Endpoints-Auth-Luecke + Frontend-URL-Mismatch in `ad04191` gefixt |
| **S1.R2.1** | **TURN/STUN-Server — coturn self-hosted** auf eigenem CAX11 (ARM, ~€3.80/M, Falkenstein), TURN-URLs im LiveKit-Client-Token + `use_external_ip: true` | 2d | R2-P0.1 | ✅ Done 2026-04-26 (`e4b98b9`) — coturn live auf `turn.zentria.tech:3478` (CAX11 FSN1), LiveKit `use_external_ip: true` aktiv, **LiveKit-Wiring fertig**: video-service schreibt per-Session-TURN-Credentials (HMAC-SHA1) als Metadata-JSON in den AccessToken. README Step 5 in `ad04191` korrigiert (alter `livekit-turn.yaml`-Ansatz war falsch), config TURN-Symmetrie-Check ergaenzt |
| **S1.R2.2** | **LiveKit-Secrets Startup-Assertion** (keine Dev-Defaults in Prod) | 2h | R2-P0.2 | ✅ Done 2026-04-18 (`310c803`) |
| **S1.R2.3** | **Recording-Consent-Bug:** `StartRecording` uebergibt alle aktiven Call-Teilnehmer als `participantIDs` (`video_grpc.go:213`) | 1d | R2-P0.3 | ✅ Done 2026-04-18 (`efd752a`) |
| **S1.R2.5** | **Egress-Webhook** ruft `CompleteRecording` (`route_video.go:1153-1176`) | 4h | R2-P0.5 | ✅ Done 2026-04-18 (`d8f89d4`) |
| **S1.R2.6** | **Lexware-Webhook HMAC-Signatur-Validierung** (`webhook_handler.go:99-113`) | 1d | R2-P0.6 | ✅ Done 2026-04-18 (`787c327`) |

**Parallelitaets-Model:** 4 Git-Worktrees fuer Module + 1 Worktree fuer R2-P0-Batch. Je Worktree 1 Sonnet-Agent als Code-Schreiber, Luke als Reviewer.

**Session 2026-04-18 (vor Sprint-Start):** Direct-to-main mit Subagent-Wellen statt Worktrees — Welle-Protokoll in `memory/project_sprint1_progress.md`. Erledigt: S1.1 wiki, S1.4 helpdesk, S1.R2.1 Code-Teil, S1.R2.2/3/5/6 komplett, S1.7 Teilstuecke.

**Session 2026-04-19:** S1.2 berichte komplett geschlossen (6 Commits `5039f79`..`a4b2cc9`) via 3-Wellen-Subagent-Pipeline. Ports 50063/9103, ACL-Seed-Migration 000080, alle Coverage-Ziele erfuellt (Export 80.2%, gRPC 77.6%, Routes 57%, Scheduler 89.4%, Service 52.2%). Knowledge-Stand aktualisiert.

**Progress Sprint 1 (Stand 2026-04-26):** ✅ **7/7 Module done** (S1.1 wiki, S1.2 berichte, S1.3 formulare, S1.4 helpdesk + S1.5 vertraege, S1.6 buchhaltung-Completion, S1.7 video-Completion). ✅ **5/5 R2-P0 Batch A done** (S1.R2.1 TURN-Wiring komplett in `e4b98b9` + Bugfixes in `ad04191`, S1.R2.2/3/5/6 alle done). Sprint 1 inhaltlich abgeschlossen, Gate S1 bestanden — siehe §7.

**S1.PREP Production-Redeploy (2026-04-19/20):** ✅ Full-Redeploy CPX42 von `fa17fc3` (2026-03-08) → `980eba3`. 171 Commits, 20 Migrations (62→81), 4 neue Services (wiki/helpdesk/berichte/formulare) live mit Feature-Flags default OFF. Deploy-Hygiene-Commit `980eba3` fixt 3 `deploy.sh`-Bugs (COMPOSE_FILES_DIR, --env-file, Rolling-Restart-Liste) + 000079-Idempotenz + PRODUCTION_TEMPLATE. 6 weitere Infra-Bugs server-seitig ad-hoc per `skip-worktree` gepatched (docker-compose.yml 18× hardcoded `kmuhub_dev`, Healthcheck `--spider`→GET, formulare `/health`→`/healthz`) — müssen in Sprint 2 als saubere Commits auf `main`. Post-Deploy: alle 15 Business-Services healthy, `/health` liefert `commit: 980eba3`. Details Memory `project_server_redeploy_20260419.md`.

**Ende Sprint 1:** 7 Module live, 5 R2-P0 erledigt, Coverage ≥30% pro Modul.

---

### Sprint 2 — Backend-Offensive Teil 2 + R2-P0 Batch B + Option-B Phase 1 (2026-04-26 – 2026-05-24, frueher gestartet)

**Ziel:** Restliche 7 Module + die restlichen R2-P0-Items + Start Option-B-Retrofit (Top-20 Tabellen).

**Stand 2026-04-28 (Welle 2C done):** Sprint 2 Welle 0 + Welle 1 + Welle 2A + Welle 2B + Welle 2C done. **7/7 Welle-2-Module Backend + Frontend-Hooks live** mit Bugfix-Sweep nach `ad04191`-Pattern. Cross-Module JWT-Claim-Extraction-Task (Welle-1-Altlast in 7 Routes) und Welle 3 (R2-P0.4/7 + Option-B Phase 1) stehen aus.

| # | Task | Aufwand | Kategorie | Status |
|---|---|---|---|---|
| S2.1 | **rapporte** | 4d | Modul | ✅ Done 2026-04-28 (`c52839f`) — Port 50074/9114, Migration 000092+000093, 18 RPCs, Approval-State-Machine, GPS-Tag, Coverage 35.6% (35 Tests) |
| S2.2 | **schichten** (ArbZG-Warnings im Backend) | 4d | Modul | ✅ Done 2026-04-28 (`c52839f`) — Port 50075/9115, Migration 000094+000095, 16 RPCs, ArbZG §5 Pre-Check (11h Ruhezeit, DST-aware), Coverage 35.6% (38 Tests) |
| S2.3 | **fuhrpark** (TÜV-Reminder) | 3d | Modul | ✅ Done 2026-04-28 (`e4b1a62`) — Port 50076/9116, Migration 000096+000097, 18 RPCs, TÜV-Reminder-Cron-Worker (advisory-lock 7d/1d), Coverage 39.7% (21 Tests). TÜV-Notification-Delivery noch Stub (Sprint-3-Wiring an `notification`-gRPC) |
| S2.4 | **vermietung** (Zustandsprotokolle) | 4d | Modul | ✅ Done 2026-04-28 (`c52839f`) — Port 50077/9117, Migration 000098+000099, 20 RPCs, GIST tstzrange-Overlap-Index gegen Doppelbuchung, Coverage 41.3% (34 Tests) |
| S2.5 | **inventar** (Bestands-Alarm) | 3d | Modul | ✅ Done 2026-04-26 (`9438ba0`) — Port 50070, Migration 000083+000084, Coverage 45.4%. Oversell-Bug + tenant_id-Filter + RowsAffected-Checks in `ad04191` gefixt |
| S2.6 | **einkauf** (Wareneingang) | 3d | Modul | ✅ Done 2026-04-26 (`9438ba0`) — Port 50071, Migration 000085+000086, Coverage 32.2%, ReceiveGoods-Stub fuer Sprint-3-Inventar-Wiring. allFullyReceived-String-Drift + PartialReceive-Validation + DeletePO-Status-Guard + RowsAffected-Checks in `ad04191` gefixt |
| S2.7 | **produktion** (Maschinenbelegung) | 3d | Modul | ✅ Done 2026-04-26 (`9438ba0`) — Port 50072, Migration 000087+000088, Coverage 31.2%. CreateBookingWithLock mit pg_advisory_xact_lock + Tx in `ad04191` ergaenzt (Race-Test mit 50 Goroutines) |
| **S2.QA** | **Code-Review + Bugfix-Sweep aller Welle-0+1-Outputs** — 6 parallele Explore-Subagents identifizierten 21 Blocker (1 false positive). 20 echte Blocker in 3 Wellen gefixt: Quick Wins (RowsAffected, tenant-Filter, JSON-Parse, BOM, TURN-Symmetrie, Frontend-URLs, Tag-Endpoints-Auth), Logik-Bugs (Oversell, String-Drift, Validation, LockInvoice-Bypass), Architektur (advisory-lock, Migration-91-Restruktur, GoBD-Format-Version + Tests). 27 Files, +1112/-154, 21 neue Tests | 1d | QA | ✅ Done 2026-04-26 (`ad04191`) |
| **S2.R2.1b** | **LiveKit TURN-Wiring** — `video`-Service um per-Session-TURN-Credentials im `AccessToken` erweitern (HMAC-SHA1 vom `TURN_SECRET`, Expiry 4h, Username-Format `<expiry>:<identity>`). Danach End-to-End-Smoke-Test mit `RTCPeerConnection.getStats()` → `candidateType: relay`. Siehe `deploy/turn/livekit-integration.md` Option B. | 1d | R2-P0.1 (Teil 2) | ✅ Done 2026-04-26 — Vorgezogen in S1.R2.1 (siehe Sprint 1) |
| **S2.PREP** | **Full-Redeploy CPX42** — Server auf main-HEAD heben (6 Wochen Rückstand seit 2026-03-08), LiveKit-Secrets echt setzen, alle Sprint-0-R1-P0 + Sprint-1-R2-P0-Fixes produktiv. Muss VOR S2.R2.1b laufen sonst fehlt Code-Basis. | 0.5d | Ops | ✅ Done 2026-04-19/20 (`980eba3`, S1.PREP) |
| **S2.R2.4** | **Frontend Recording-Consent-Modal + persistenter Banner** (Join-with-Consent-Modell: einmaliger Consent-Click beim Call-Beitritt, Banner + rotes Mic-Icon waehrend Aufnahme, Ablehnung → Kick) | 2d | R2-P0.4 | Pending — Welle 2/3 |
| **S2.R2.7** | **Offline-Queue** im Desktop-WS-Client: IndexedDB-Buffer fuer Messages, Reconciliation bei Reconnect, Duplicate-Detection | 3d | R2-P0.7 | Pending — Welle 2/3 |
| **S2.R2.8** | **`consent_records.created_by`** ON DELETE SET NULL (Migration 000082) | 2h | R2-P0.8 | ✅ Done 2026-04-26 (`2245ecb`) — Welle 0 |
| **S2.R2.9** | **`gdpr_deletion_requests.contact_id`** zirkulaere Blockade aufloesen | 4h | R2-P0.9 | ✅ Done 2026-04-26 (`2245ecb`) — Welle 0 |
| **S2.MT.1** | **Option-B Phase 1 (Top-20 Tabellen):** `deals`, `activities`, `messages`, `channels`, `channel_memberships`, `notifications`, `events`, `audit_log`, `automations`, `automation_executions`, `calendar_events`, `meetings`, `calls`, `tasks`, `projects`, `team_inboxes`, `inbox_messages`, `document_folders`, `document_files`, `recordings` — je ALTER ADD COLUMN tenant_id + Backfill + Index + RLS-Policy | 5d | Option-B | Pending — Welle 3 |

**Parallelitaets-Model:** 4 Module-Worktrees + 1 Realtime/R2-P0-Worktree + 1 Multi-Tenancy-Worktree.

**Progress Sprint 2 (Stand 2026-04-28 Abend):** **7/7 Welle-2-Module Backend + Frontend-Hooks + Bugfix-Sweep done.** Welle 1 (`9438ba0`) + Welle 2A (`e4b1a62`+`c52839f`) + Welle 2B Frontend-Hooks (`1a94503`, 12 Files, 2.904 LOC, ~70 Hooks) + Welle 2C 23-Bugfix-Sweep (`a4d189e`, 36 Files, +866/-124, 4 neue Migrations 100-103). Coverage rapporte 33.9%, schichten 35.2%, fuhrpark 39.8%, vermietung 40.9%. **Welle-1-Altlast aufgedeckt:** hardcoded Placeholder-TenantID in 7 Routes (rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion) → eigene Sprint-2-Cross-Module-Task fuer JWT-Claim-Extraction-Refactor vor Pilot-1. **Naechste Schritte:** Cross-Module JWT-Refactor → Welle 3 (S2.R2.4 Consent-Modal + S2.R2.7 Offline-Queue + S2.MT.1 Option-B Phase 1).

**Ende Sprint 2:** Alle 14 Module live, alle 9 R2-P0 erledigt, Option-B Top-20 aktiv.

---

### Sprint 3 — Option-B Phase 2 + Infrastruktur-P1 (2026-05-25 – 2026-06-07, 2 Wochen)

**Ziel:** Restliche ~30 Tabellen Option-B-Retrofit + Ansible + Security-Scans.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S3.MT.2 | **Option-B Phase 2:** restliche ~30 Tabellen mit tenant_id + RLS (alle Hilfstabellen, Preferences, Settings, Mappings, Cache-Layer) | 5d | Option-B |
| S3.1 | Ansible-Playbook Instanz-pro-Pilot (jetzt mit Option-B-tauglichem Schema) | 5d | R1-P1.1 |
| S3.2 | Dependency-Security-Scans (`trivy`, `gosec`, `npm audit`) in CI | 2d | R1-P1.2 |
| S3.3 | Dialer `LogCallOutcome` in Transaktion + Integration-Test | 2d | R1-P1.3 |
| S3.4 | Prod-Image-Tags pinnen (7× `latest` → Version-Hash) | 30min | R1-P1.4 |
| S3.5 | Alertmanager + Discord-Webhook (Slack-Compat-Mode) | 1d | R1-P1.5 |
| S3.6 | `cd.yml` Auto-Deploy auf main-merge (mit Green-Gate) | 1d | R1-P1.6 |
| S3.7 | Dialer-Test-Coverage 12% → ≥30% | 8d | R1-P1.8 (parallel) |

**Parallelitaets-Model:** 1 Worktree Option-B + 1 Infra/Ansible + 1 Test-Coverage. Multi-Tenancy-Migration muss zuerst fertig sein, bevor Ansible-Blueprint diese Schema-Version kennt.

**Progress Sprint 3 (Stand 2026-05-08, abgeschlossen):** ✅ **8/8 Tasks done.** S3.MT.2 Option-B Phase 2 ✅ (Migrations 000114+115, ~38 neue Tabellen mit `tenant_id`, gestern committed). S3.2 trivy/gosec/npm audit ✅. S3.3 Dialer-Tx ✅. S3.4 Image-Pins ✅ (heute 2 Korrekturen fuer withdrawn redis 7.2.7 + minio/mc tag-rotation). S3.5 Alertmanager + Discord ✅ (`ALERT_WEBHOOK_URL` Server-Env-Set + GitHub-Secret-Update noch ausstehend nach 2026-05-09-Refactor, kein Blocker). S3.6 `cd.yml` ✅. **Production-Server-Deploy 2026-05-08:** Migration 81 → 115, 32 Container healthy auf `3abec5f`, 9 Hotfix-Commits zur Deploy-Infrastruktur (serial-build OOM-Fix, tenants-Tabellen-Bootstrap, redis 7.4-bump, healthcheck.sh dreifach gefixt). **Welle 2 + 3 (heute):** S3.7 Dialer-Coverage 12% → **31.8%** (Commit `1f6c4c0`, 4 neue Test-Files, Bonus-Fix `consent.ErrNoConsent` → `codes.PermissionDenied`). S3.1 Ansible-Playbook komplett (Phase 1 `a8d77fc` foundation+secrets+inventory, Phase 2A `71f7c90` app-deploy+Caddyfile.j2 mit `pilot_domain`-Templating, Phase 2B `562e9c5` turn+Let's-Encrypt+DNS-Helper). 4 Roles, 50 Ansible-Tasks insgesamt, ansible-lint production-profile **0 failures**. Tooling-Notiz: Ansible-Verifikation laeuft ueber Docker-Wrapper (`willhallonline/ansible:latest`), weil Native-Windows-Ansible nicht funktioniert. **S3.MT.4 Audit** verschoben auf Sprint 5 (laut User-Entscheidung). Gate S3 bestanden.

**Ende Sprint 3:** Alle ~50 Tabellen mit tenant_id + RLS, Ansible-Playbook mit Option-B deployt eine Pilot-Instanz in <30 Min, CI hat trivy/gosec/npm audit gruen.

---

### Sprint 4 — Finance-Normalisierung + P1-Security + Runde-2-P1 (2026-06-08 – 2026-06-21, 2 Wochen)

**Ziel:** finance_invoices.line_items raus aus JSONB, Finance-Test-Coverage hoch, restliche R1-P1 + R2-P1.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S4.FI.1 | **`finance_invoices.line_items` → `finance_invoice_lines`-Tabelle** (Migration + Backfill + Read-Path-Update + Write-Path-Update + ZUGFeRD-Export-Anpassung) | 3d | R2-P1.12 |
| S4.FI.2 | **Finance-Test-Coverage ausbauen** — Service-Level Tests fuer Rechnungserstellung, Positions-Summen, Steuerberechnung, Zahlungs-Verbuchung, Dunning-Flow | 5d | Quality |
| S4.1 | ✅ Input-Validation-Framework (`go-playground/validator`) — Scope erweitert auf **alle** Mutation-Handler (statt nur 20), 4 Wellen, 2026-06-08 | 5d | R1-P1.7 |
| S4.2 | LiveKit-Webhook-Signatur-Validierung (Stub aufloesen) | 1d | R2-P1.1 |
| S4.3 | Automation-Semaphor tenant-isolieren (kein Global-20) | 1d | R2-P1.3 |
| S4.4 | Bexio + DATEV Circuit-Breaker + DATEV-Retry | 1d | R2-P1.4 |
| S4.5 | StartMeeting + `POST /meetings` Rollen-Check + Organizer-only-Start | 4h | R2-P1.5 |
| S4.6 | WS-Token periodisch in-session revalidieren | 4h | R2-P1.6 |
| S4.7 | Redis-backed WS-Subscription-State (Gateway-Restart-resistent) | 2d | R2-P1.7 |
| S4.8 | `dialer_*.outcome_id`-Indizes (Migration 000076) | 30min | R2-P1.8 |
| S4.9 | ~10 FKs ohne ON DELETE nachziehen (Migration 000077) | 1d | R2-P1.9 |
| S4.10 | Partitionierung audit_log/events/dialer_call_events/automation_executions + pg_cron-Retention (30d Rolling) | 3d | R2-P1.10 |
| S4.11 | `CleanupExpiredRecordings`-Cronjob aktivieren | 2h | R2-P1.11 |

**Parallelitaets-Model:** 1 Finance-Worktree (normalize + test) + 1 Security/Validation + 1 DB-Ops. Finance-Test-Coverage laeuft mindestens parallel mit Normalisierung.

**Progress Sprint-4-Vorzug (2026-06-05, zwei Sessions):** Session 1 (Vormittag): S4.2–S4.9 + S4.11 ✅ (R2-P1-Batch 9/10, Commits `f5788d8d`/`98337921`/`5dd862eb`), R2-P0.4 ✅ (`19d5adb7`), Smoke 24/24 + CI Desktop erstmals gruen. Session 2 (Abend): **LiveKit/COSMI_ENV-Cluster komplett geschlossen** — Production-Secrets-Sweep (alle 24 Services liefen mit Dev-JWT_SECRET/MinIO/WOPI/Vault aus der Basis-Compose!), `${VAR:-dev-default}`-Interpolation + Assertion-Haertung mit Requirements-API (`config.Load(ctx, ...Requirement)`), LiveKit-URL-Split intern/public + Caddy-`/rtc*`-Proxy + echte Join-Tokens (StartMeeting/JoinCall lieferten IMMER leere token/ws_url!), RLS-Read-Gap `call_sessions.tenant_id`, `COSMI_ENV=production` scharf. **Video-Calls in Production erstmals end-to-end funktional** (`/rtc/validate` = 200). 5 Commits `68158907`/`7d492bb6`/`5f16f0d9`/`78043a63`/`ce2a5e5d`, Befund-Historie `docs/livekit-env-production-followups.md`. Restscope Sprint 4: S4.FI.1/2 (Finance, Kernstueck) + S4.1 (Input-Validation) + S4.10 (Partitionierung).

**Progress S4.FI (2026-06-08):** ✅ **S4.FI.1 + S4.FI.2 erledigt** (ADR-0007 relationaler Cutover). Migrationen **000132** (3 Line-Tabellen `finance_invoice_lines`/`finance_quote_lines`/`finance_credit_note_lines` mit FK CASCADE + RLS + `tax_rate`-CHECK DACH-sicher 0–100, statt DE-only; + `locked_at`/`locked_by` auf finance_invoices) + **000133** (idempotenter Backfill + Lock-Migration aus snapshot_data). Repos invoice/quote/creditnote auf relationale Lines umgestellt (atomare Tx, Bulk-Read ohne N+1); `service_gobd.go`-Lock auf echte Spalten (snapshot_data-Hack raus). **Sauberer Cutover ohne Dual-Write/Feature-Flag** (keine Prod-Finance-Daten). `line_items` JSONB-Spalte bleibt synchron befuellt → gRPC/pdf/datev/dashboard unveraendert (kein API-Bruch — Proto war schon `repeated LineItem`); **JSONB-Drop deferred auf Sprint 5**. Test-Coverage via testcontainers-go-Integrationstests (echtes PG16, RLS/Constraints/Idempotenz/Tenant-Isolation): **invoice 69.6% · quote 63.7% · creditnote 51.3%**. ⚠ Integrationstests sind `//go:build integration`-gated → CI muss `-tags=integration` (+Docker) laufen lassen, damit die Coverage CI-seitig zaehlt (Follow-up CI-Workflow).

**Progress S4.1 (2026-06-08):** 🔄 **Welle 1+2 von 4 erledigt** (R1-P1.7, Input-Validation). User-Scope-Entscheidung: **ALLE ~380 Mutation-Handler** (nicht nur 20) + **maximale DACH-Validatoren** + strukturierte Error-Shape. Welle 1 (`3937ff2d`): `go-playground/validator/v10`-Foundation — neue Packages `internal/dachfmt` (IBAN mod-97/BIC/PLZ/USt-IdNr-EU-VIES/Steuernummer + `NormalizePhoneE164` aus dialer extrahiert) + `internal/validation` (Singleton, 7 Custom-Validatoren, `decodeAndValidate[T]`-Helper, Error-Shape `{error, code:"validation_failed", details}`), `route_auth.go` als Referenz. Welle 2 (`cb784f79`): Finance/Integrations/Dialer/CRM/Helpdesk — **240 `validate`-Tags / 128 Call-Sites / 22 Route-Files**; Webhook-Handler (LiveKit/Lexware) bewusst raw (Signatur zuerst). **Offen: Welle 3** (Collaboration: work/calendar/caldav/chat/email/inbox/document/wopi/automation/formulare/notification) **+ Welle 4** (Modul-Services: rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion/vertraege/hr/plugin/berichte/dashboard/search/registrar/feature_flags). Keine Migration. Details siehe [[security]] "Validation-Framework".

**Ende Sprint 4:** finance_invoices relational, Finance-Coverage ≥50%, alle R1-P1 + R2-P1 erledigt.

---

### Sprint 5 — Integration, Polish, Pre-Launch-Audit (bis Gate S5 2026-08-31)

**Ziel:** End-to-End-Tests, Content-Cleanup, Peer-Review, Rigorosum Runde 3, Launch-Freigabe.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S5.1 | Website-Content-Audit (zentria.tech) — Features-Liste mit realem Delivery-Status abgleichen, WASM-Plugin-Claims entfernen | 1d | GTM |
| S5.2 | End-to-End-Smoke-Test: alle 14 Module + Realtime-Flows (Chat+Meetings+Recording) + Option-B-Tenant-Isolation | 2d | QA |
| S5.3 | Legal: AGB, Impressum, Datenschutzerklaerung final (mit UG-Daten nach 01.06.) | 1d | Legal |
| S5.4 | Peer-Review durch Luke's Ex-Mitgruender | 2d extern | Review |
| S5.5 | **Rigorosum Runde 3** (Claude re-auditiert alle neuen Module + Ansible + Option-B + Realtime-Haertung + R2-P0-Closure) | 1d | Review |
| S5.6 | Finaler Smoke auf Prod (`deploy/scripts/smoke.sh`) | 30min | Launch |
| S5.7 | UG-Notartermin (01.06. parallel zu Sprint 3), Konto aktiv | extern | Legal |

**Ende Sprint 5:** Launch-Freigabe. ZFA-Pilot-0-Onboarding ab 01.09. (Hinweis: Der durch die Verschiebung auf 01.09 gewonnene Puffer trägt den jetzt launch-kritischen Finance-Block — Wellen 4–7, vormals „volle P0 01.09".)

---

## 4. Launch-Readiness-Checkliste (P0–P3)

Quelle: Rigorosum Runde 1 (wild-wren, 2026-04-18) + Rigorosum Runde 2 Vertiefung (functional-seahorse, 2026-04-18). R1 = Runde 1, R2 = Runde 2 Vertiefung.

### P0 — Launch-Blocker (16 Items, bis Ende Sprint 2 dicht)

**Runde 1 P0 (7 Items, Sprint 0) — ✅ ALLE DONE 2026-04-18:**

| # | Task | Status | PR |
|---|---|---|---|
| R1-P0.1 | Migration 000075 CASCADE → SET NULL | ✅ Done | #5 |
| R1-P0.2 | `assertConsent()` vor Email/Dialer-Send | ✅ Done | #10 |
| R1-P0.3 | Prod-Secrets required | ✅ Done | #6 |
| R1-P0.4 | DOMPurify in dangerouslySetInnerHTML | ✅ Done | #9 |
| R1-P0.5 | OnlyOffice JWT in Prod | ✅ Done | #7 |
| R1-P0.6 | Feature-Flag-Registry (inkl. WASM-OFF) | ✅ Done | #11 |
| R1-P0.7 | ICU-Plural-Klammern-Fix | ✅ Done | #3 |

**Runde 2 P0 (9 Items, Sprint 1+2) — ✅ ALLE 9 DONE (Stand 2026-06-05):**

| # | Task | Status | Sprint |
|---|---|---|---|
| R2-P0.1 | TURN/STUN coturn self-hosted + `use_external_ip: true` + LiveKit-Wiring | ✅ Done 2026-04-26 (`e4b98b9`+`ad04191`) — coturn live (CAX11 FSN1, `turn.zentria.tech:3478`), LiveKit `use_external_ip: true`, video-service schreibt HMAC-SHA1-TURN-Credentials als Metadata-JSON in AccessToken, config-Symmetrie-Check, README korrigiert | S1 |
| R2-P0.2 | LiveKit-Secrets Startup-Assertion | ✅ Done (`310c803`) | S1 |
| R2-P0.3 | Recording-Consent-Bug (`video_grpc.go:213` — alle Teilnehmer) | ✅ Done (`efd752a`) | S1 |
| R2-P0.4 | Frontend Recording-Consent-Modal + Banner (Join-with-Consent) | ✅ Done 2026-06-05 (`19d5adb7`, MeetingLobby + RecordingActiveBanner) | S4-Vorzug |
| R2-P0.5 | Egress-Webhook ruft `CompleteRecording` | ✅ Done (`d8f89d4`) | S1 |
| R2-P0.6 | Lexware-Webhook HMAC-Signatur-Validierung | ✅ Done (`787c327`) | S1 |
| R2-P0.7 | Offline-Queue Desktop-WS (IndexedDB + Reconciliation) | ✅ Done 2026-04-28 (`174a7e4`, idb-keyval + Idempotency-Middleware) | S2 Welle 3 |
| R2-P0.8 | `consent_records.created_by` ON DELETE SET NULL | ✅ Done 2026-04-26 (`2245ecb`, Migration 000082) | S2 Welle 0 |
| R2-P0.9 | `gdpr_deletion_requests.contact_id` zirkulaere FK aufloesen | ✅ Done 2026-04-26 (`2245ecb`, Migration 000082) | S2 Welle 0 |

### P1 — Vor Pilot-1 (Ende Sprint 4)

**Runde 1 P1 (8 Items, Sprint 3+4) — 7/8 done:**

| # | Task | Status | Sprint |
|---|---|---|---|
| R1-P1.1 | Ansible Instanz-pro-Pilot (mit Option-B-Schema) | ✅ Done 2026-05-08 (`a8d77fc`+`71f7c90`+`562e9c5`, 4 Roles, 50 Tasks) | S3 |
| R1-P1.2 | Dependency-Security-Scans in CI | ✅ Done 2026-05-08 (`241686e`, trivy/gosec/npm audit) | S3 |
| R1-P1.3 | Dialer LogCallOutcome Transaktion | ✅ Done 2026-05-08 (`eab0181`) | S3 |
| R1-P1.4 | Prod-Image-Tags pinnen | ✅ Done 2026-05-08 (`7a22d83`) | S3 |
| R1-P1.5 | Alertmanager + Discord (Slack-Compat) | ✅ Done 2026-05-08/09 (`7a22d83`+`2330add`, live in #cosmi-prod-alerts) | S3 |
| R1-P1.6 | cd.yml Auto-Deploy | ✅ Done 2026-05-08 (`7a22d83`) | S3 |
| R1-P1.7 | Input-Validation-Framework | ✅ Done 2026-06-08 — alle 4 Wellen (`3937ff2d`+`cb784f79`+`45898f4b`+`29e77fb7`), alle JSON-Body-Mutation-Handler ueber ~41 Route-Files (Webhook/WOPI/CalDAV-Protokoll/proto-direct-Passthrough bewusst ausgenommen) | S4 |
| R1-P1.8 | Dialer-Coverage 12 → 30% | ✅ Done 2026-05-08 (`1f6c4c0`, 31.8%) | S3 |

**Runde 2 P1 (12 Items, Sprint 4) — 10/12 done:**

| # | Task | Status | Sprint |
|---|---|---|---|
| R2-P1.1 | LiveKit-Webhook-Signatur-Validierung | ✅ Done 2026-06-05 (`f5788d8d`) | S4 |
| R2-P1.2 | WASM-Plugin-System Feature-Flag OFF | ✅ Done | S0 (zusammen mit R1-P0.6, PR #11) |
| R2-P1.3 | Automation-Semaphor tenant-isolieren | ✅ Done 2026-06-05 (`f5788d8d`, 5/Tenant in global 20) | S4 |
| R2-P1.4 | Bexio+DATEV Circuit-Breaker/Retry | ✅ Done 2026-06-05 (`5dd862eb`, internal/circuitbreaker) | S4 |
| R2-P1.5 | StartMeeting Rollen-Check + Organizer-only | ✅ Done 2026-06-05 (`98337921`, Migration 000131) | S4 |
| R2-P1.6 | WS-Token in-session revalidieren | ✅ Done 2026-06-05 (`98337921`, 5-min-Ticker) | S4 |
| R2-P1.7 | Redis-backed WS-Subscription-State | ✅ Done 2026-06-05 (`5dd862eb`) | S4 |
| R2-P1.8 | Dialer `outcome_id`-Indizes | ✅ Done 2026-06-05 (`98337921`, Migration 000130) | S4 |
| R2-P1.9 | ~10 FKs ohne ON DELETE nachziehen | ✅ Done 2026-06-05 (`98337921`, Migration 000130) | S4 |
| R2-P1.10 | Partitionierung + pg_cron-Retention | Pending (S4.10 abgespalten — kein pg_cron im pgvector-Image, braucht Maintenance-Window) | S4 |
| R2-P1.11 | `CleanupExpiredRecordings`-Cronjob | ✅ Done 2026-06-05 (`f5788d8d`, 24h-Cron) | S4 |
| R2-P1.12 | `finance_invoices.line_items` normalisieren | ✅ Done 2026-06-08 (Migr. 000132/000133 relationaler Cutover, ADR-0007; JSONB synchron befuellt → kein API-Bruch, JSONB-Drop deferred → S5) | S4 |

### P2 — Vor Pilot-Skalierung (Post-Launch Phase C, August–November)

**Runde 1 P2:** DSGVO-Erasure Stores · Rollen-Route-Guards + DEV_BYPASS_AUTH-Hardening · Duplikat-Komponenten konsolidieren · Virtualisierung Top-10-Listen · CSP in Caddy · crm/contacts ↔ kontakte-Konsolidierung · Error-Wrapping-Hygiene · .pb.go aus Repo + buf generate

**Runde 2 P2 (15 Items):** Redundante Indizes bereinigen (consent_records, email_accounts, audit_log) · Compound-Indizes (activities, hr_leave, deals, message_mentions) · WOPI-Token-Masking in Logs · Slack-Acknowledge in DB markieren · Automation-DLQ · Guest-Chat Content-Scanning · Typing-Timeout + WS-Ping/Pong · Reaction-Toggle mit SELECT FOR UPDATE · GDPR-Erasure-Pfad deckt recordings · CHECK-Constraints auf consent_records.source/gdpr_status · inbox_messages.crm_contact_id FK · Moderator-Audit-Log · Recording-Tagging (started_by, consent_snapshot) · Guest-Chat Hard-Delete · LiveKit-Token-Grants differenzieren (Moderator/Gast, 24h→1h)

### P3 — Nice-to-Have (Phase D, ab November 2026)

**Runde 1 P3:** Go-Stable-Version · Panic-Fix automation/templates.go · Naming (email/mails, biz/finanzen/buchhaltung, kalender/calendar) · Tenant-scoped Rate-Limit · React.memo Review · Hard-coded Hex → CSS-Vars · Desktop Auto-Update · Loki/Log-Aggregation · Mobile-Scaffold (wenn Markt es verlangt)

**Runde 2 P3 (6 Items):** UUID-Generator-Konsistenz (Migration 000033) · Dead Tables entfernen (event_types, storage_quotas, task_dependencies, notification_mutes) · BEGIN/COMMIT-Wrapping in Migrations · VARCHAR(20) status durchgehend CHECK-constrained · GIN-Index auf dialer_campaigns.assigned_agent_ids wenn >100 Agents · **WASM-Plugin-System Haertung** (Ed25519-Signing + WASI-Deny-Set) statt Deaktivierung — nur wenn Phase-D-Markt-Signal fuer Plugin-Dev-Interesse

---

## 5. Die 14 Module — Realisierungs-Plan

**Prinzip:** Pro Modul **minimum viable Backend** bis 01.06. Kein Over-Engineering. Aufbau erfolgt nach folgendem Template:

### Standard-Modul-Template

1. **Tabellen + Migration** (mit `tenant_id` von Anfang an, auch bei Single-Tenant)
2. **gRPC-Proto** (12–20 RPCs typisch: Create/Update/Delete/List/Get + Domain-spezifische Flows)
3. **Go-Service** (`service.go` + `postgres_repository.go` + `service_test.go` mit ≥30% Coverage)
4. **Gateway-Routes** (`route_<modul>.go`)
5. **Frontend-API-Hooks** (`api/hooks/use<Modul>.ts`) ersetzt Zustand-Store-Persister
6. **Feature-Flag-Aktivierung** final, Mock-Daten aus Store entfernt

### Modul-Matrix

| Modul | Sprint | Kern-Tabellen | ~RPCs | Prio-Pilot | Notizen |
|---|---|---|---|---|---|
| **wiki** | S1 | wiki_articles, wiki_versions, wiki_attachments | 14 | Dienstleister | FTS, TipTap-Content, Share-Links |
| **berichte** | S1 | report_definitions, report_cache | 10 | Dienstleister | Aggregations-Views |
| **formulare** | S1 | form_schemas, form_submissions | 16 | Cross | JSONB-Schema, Webhook-Trigger |
| **helpdesk** | S1 | tickets, ticket_messages, ticket_queues, canned_responses | 22 | Dienstleister | Merge, SLA, Canned |
| **vertraege** | S1 | contracts, contract_parties, contract_reminders | 14 | Dienstleister | Laufzeit-Engine |
| **buchhaltung** | S1 | Completion | — | Cross | GoBD-Journal |
| **video** | S1 | Completion | — | Cross | Recording-Tagging |
| **rapporte** | S2 | work_reports, report_lines, report_attachments | 18 | Handwerk | Foto-Uploads, GPS-Tag |
| **schichten** | S2 | shifts, shift_assignments, shift_templates | 16 | Handwerk | ArbZG §5 Backend-Check |
| **fuhrpark** | S2 | vehicles, vehicle_services, vehicle_damages | 18 | Handwerk | TÜV-Reminder |
| **vermietung** | S2 | rental_objects, rentals, rental_inspections | 20 | Handwerk | Zustandsprotokolle |
| **inventar** | S2 | inventory_items, inventory_movements, stock_warnings | 16 | Cross | Bestands-Alarm |
| **einkauf** | S2 | purchase_orders, suppliers, po_lines | 18 | Cross | Wareneingangs-Flow |
| **produktion** | S2 | production_orders, machine_bookings, production_plans | 16 | Handwerk | Maschinenbelegung |

**Summe:** ~240 neue RPCs, ~70 neue Tabellen. Parallelisiert ueber 4 Worktrees × 2 Wochen pro Sprint. Realistisch mit 20×-Plan, eng mit Puffer.

### Safety Net

Falls ein Modul bis zum Launch-Freeze (2026-05-28) nicht fertig ist: **Feature-Flag OFF + "Coming Q3 2026"-Overlay**. Besser ein halbes echtes Produkt mit ehrlichem Coming-Soon als ein Mock-Theater.

---

## 6. Post-Launch Phase C (September – Dezember 2026)

### Phase C — Pilot-Betrieb (September – Oktober 2026)

- **Track: Legal** — AVV/DPA mit Anwalt finalisieren
- **Track: Onboarding** — ZFA-Pilot-0 + 2 Dienstleister-Piloten (kostenlos)
- **Track: Feedback-Loop** — woechentliche Pilot-Calls, Issue-Tracking, Hotfix-Cycle ≤72h
- **Track: P2-Aufholsprint** — DSGVO-Erasure, Rollen-Route-Guards, Duplikat-Konsolidierung
- **Track: Demo-Video** — Hero (3–5 Min) + Outreach (60–90 Sek)

### Phase D — Production Hardening + Handwerks-Pilot (November – Dezember 2026)

- Redis Caching Layer (3 Tiers) — aus PERFORMANCE-PLAN §5.2
- PgBouncer als DB Connection Pool
- Desktop-Test-Coverage aufbauen (derzeit <5%, Ziel ≥30% fuer Kern-Module)
- Handwerks-Pilot-1 (Rapporte/Schichten/Fuhrpark-Stress-Test)
- OnlyOffice → Collabora Migration (Lizenz-Risiko mitigieren)
- Dialer Phase 2 (PSTN-SIP) — wenn ZFA-Pilot Phase 1 stabil (siehe alter DIALER-ROADMAP)

### Phase E — Kommerzieller Launch (Q1 2027)

- Stripe/SEPA-Integration
- Desktop-Installer-Pipeline (`electron-builder`)
- **Orbit Appliance-Paket (Self-Hosted)** — Mini-PC + Standard-Linux statt Synology/DSM, siehe **ADR-008**. Detail-Plan unten.
- Pricing-Seite auf zentria.tech
- LinkedIn + Cold Outreach (UWG §7 konform)
- FinAPI Banking
- Pitch-Deck finalisieren

#### Orbit Appliance — Roadmap (ADR-008)

**Grundsatz:** Zentria = Integrator (kein Hardware-Hersteller). Cosmi auf neutralem Mini-PC + nacktem Standard-Linux + Docker, identisch zur Hetzner-Cloud. Kein eigenes OS, kein Synology/DSM. ⚠ Integrator-/CRA-Rechtslage vor Bau anwaltlich bestaetigen. Kein Launch-Blocker fuer 01.09.

**Spikes / Gates (Vorarbeiten):**

| # | Spike | Owner |
|---|---|---|
| SP-1 | Ressourcen-Spike (**HARTES GATE**) — ausgewachsener schlanker Stack auf Kandidaten-Mini-PC (2× NVMe) messen → Pod/Station-Modell + RAM-Profil fix. Erst wenn Stack feature-fertig. Inkl. Mini-PC-vs-Synology-Vergleich | Luke |
| SP-2 | Legal — Produktrecht-Anwalt: Integrator-Linie + CRA-Software-Pflicht + AVV/§203/§43e-Vorlagen | Business |
| SP-3 | Registry/CI-Design (Gitea vs Harbor; Multi-Service-Image-Bau; Rollback-/Migrate-Sicherheit) | Luke |
| SP-4 | Remote-Layer-Konsolidierung (Headscale ↔ Portainer-Edge) | Luke |
| SP-5 | Storage (RAID-1/LUKS) + DNS-01-Automation Proof | Luke |

**Epics (Bau):**

| # | Epic | Owner |
|---|---|---|
| E1 | Orbit-Image & Provisioning (Ubuntu + cloud-init + Docker + Compose-Tier-Profile, Zero-Touch-Image, LUKS, RAID-1) | Infra |
| E2 | Deploy-Pipeline & Registry (versionierte Multi-Service-Images → private Registry; `compose pull`; Migrate forward-only; Rollback Tag-Pin) — **teilt sich mit Cloud, vorziehen** | Infra |
| E3 | Orbit-Modul in Cosmi (Health/Backup/Update/Lizenz/User; Feature-Gate `orbit`; gebrandetes Onboarding ×2) | **Frontend (Darien)** |
| E4 | Host-Agent (System-/SMART-/Backup-/Update-Status → Orbit-Modul + Aktionen) | Infra |
| E5 | Fleet & Remote-Mgmt (Headscale + Tailscale; Portainer Business + Edge; self-hosted Hetzner) | Infra |
| E6 | Monitoring & Alerting (lokaler Exporter + Remote-Aggregation Opt-in + lokale Minimal-Alarme) | Infra |
| E7 | Backup & DR (lokal + verschluesseltes Cloud-Offsite, Restore-Test, Retention, No-Cloud-Opt-out) | Infra |
| E8 | Lizenz & Aktivierung (Ed25519-Signing + Offline-Verifikation; Re-Issue/RMA; Security-Patch-ohne-Abo-Logik) | Infra |
| E9 | TLS & Netzwerk (DNS-01 pro Kunde, Split-DNS-Doku, transparenter VPN) | Infra |
| E10 | Security/Compliance/Legal (LUKS/Firewall/offizielle Images; AVV/§203/§43e; TOMs; Pentest-Budget; CRA) | Infra + Business |
| E11 | Video-Tiering (Pod=SFU-light/Cloud, Recording ab Station/Mini-PC) | Infra |
| E12 | GTM/Pricing-Ops (Erloes-Modell Kauf + optionales Service-Abo + Rueckkauf; Export-Tool Cloud↔Orbit; Beschaffung/RMA) | Business |

**Reihenfolge:** SP-1 (Gate) + SP-2 + **E2 vorziehen** → E1 + (E3 ‖ E4 gekoppelt) → E5–E11 parallel → E12/E10 begleitend.

---

## 7. Verifikations-Gates

Jeder Sprint endet mit einem harten Gate. Ohne gruenes Gate kein Fortschritt.

### Gate S0 (2026-04-27) ✅ BESTANDEN 2026-04-18

- [x] Alle 7 R1-P0-Items erledigt, Migration 000075 deployed (PR #5)
- [x] `grep -rn "dangerouslySetInnerHTML" desktop/src/renderer/src/` zeigt nur sanitize-wrapped oder i18n-trusted (PR #9)
- [x] Service-Start ohne `WOPI_JWT_SECRET`/`MINIO_SECRET_KEY`/`VAULT_MASTER_SECRET` bricht ab (PR #6)
- [x] Feature-Flag-Registry deckt 14 Module + `plugins.wasm` + `plugins.config`, `useFilteredNavItems` versteckt abgeschaltete Routen, WASM-Runtime per Build-Tag `no_wasm` in Prod-Build eliminiert (PR #11)
- [x] i18n-Test fuer ICU-Plurals gruen (PR #3)
- [x] `mobile/`-Ordner weg, Pitch korrigiert (PR #4)
- [x] 14-Modul-Scope-Matrix in `docs/MODULES_SCOPE_MATRIX.md` (PR #8)

### Gate S1 (2026-05-10) ✅ BESTANDEN 2026-04-26

- [x] 7 Module live: wiki, berichte, formulare, helpdesk, vertraege, buchhaltung-Completion, video-Completion
- [x] Coverage pro neuem Modul ≥30% (wiki 38.2%, berichte 57%+, formulare ≥30%, helpdesk 39.3%, vertraege 36.7%, buchhaltung-RPCs 12 neue Tests, video-Tagging-Tests)
- [x] **R2-P0 Batch A erledigt:** TURN/STUN-Server laeuft + LiveKit-Wiring komplett, LiveKit-Prod-Assertion greift, Recording-Consent uebergibt alle Teilnehmer, Egress-Webhook ruft CompleteRecording, Lexware-HMAC validiert
- [x] `make test` gruen auf main (alle Pakete in `go test ./...` PASS, vet sauber)
- [x] Smoke-Script durchlaeuft (siehe S1.PREP-Redeploy-Verifikation)

### Gate S2 (2026-05-24) — 5/13 Items erfuellt (Stand 2026-04-26)

- [ ] Alle 14 Module live (7/14 done: 7 Sprint-1-Module + 3 Sprint-2-Module inventar/einkauf/produktion; 4 Welle-2-Module noch ausstehend)
- [ ] **Alle 9 R2-P0 erledigt** (7/9 done: R2-P0.1/2/3/5/6/8/9; offen: R2-P0.4 Frontend-Consent-Modal + Banner, R2-P0.7 Offline-Queue)
- [ ] **Option-B Phase 1 live:** Top-20 Tabellen mit tenant_id + Backfill + RLS (Welle 3)
- [ ] Integration-Test: Tenant-A-User kann Tenant-B-Daten nicht lesen (RLS-Gate)
- [ ] Realtime-Smoke: Call mit 3 Teilnehmern, Recording-Consent von allen, Playback funktioniert
- [ ] Offline-Queue-Test: Electron-Client disconnected, Messages geschickt, reconnected, Server-Reconciliation ohne Duplikate
- [x] **Sprint-2-Welle-0+1-Bugfix-Sweep abgeschlossen** (`ad04191`): 20 Blocker aus 6-Subagent-Code-Review behoben, Race-Condition produktion via advisory-lock, Migration 91 ohne Full-Table-Lock, GoBD-CSV-Format ehrlich gekennzeichnet

### Gate S3 (2026-06-07)

- [ ] **Option-B Phase 2 live:** restliche ~30 Tabellen, insgesamt ~50 Tabellen mit tenant_id + RLS
- [ ] Ansible-Playbook: `ansible-playbook bootstrap-pilot.yml -i pilot-test` deployt vollstaendige Cosmi-Instanz auf frischen Hetzner-Host in <30 Min, Option-B-Schema korrekt
- [ ] CI hat trivy+gosec+npm audit, keine High/Critical CVEs
- [ ] Dialer-Coverage ≥30%
- [ ] Alertmanager sendet Test-Alert in Discord (`#cosmi-prod-alerts` via `ALERT_WEBHOOK_URL` mit `/slack`-Suffix)
- [ ] cd.yml deployt automatisch bei main-merge

### Gate S4 (2026-06-21)

- [x] **`finance_invoices.line_items` normalisiert:** ✅ 2026-06-08 — `finance_invoice_lines`/`finance_quote_lines`/`finance_credit_note_lines` (Migr. 000132/000133), Backfill aus `snapshot_data`, ZUGFeRD/PDF/DATEV unveraendert (JSONB synchron befuellt, Drop → S5)
- [x] **Finance-Service-Coverage ≥50%** (vorher <15%): ✅ invoice 69.6% · quote 63.7% · creditnote 51.3% (testcontainers-go, `-tags=integration`)
- [ ] Alle R1-P1 + R2-P1 erledigt (Input-Validation, LiveKit-HMAC, Automation-Tenant-Isolation, Circuit-Breaker, Meeting-Rollen, WS-Token, Redis-Subscription-State, DB-FKs-Nachzug, Partitionierung, Recording-Cronjob)

### Gate S5 (2026-08-31) — LAUNCH-FREIGABE (ZFA, 01.09.)

- [ ] Peer-Review durch Ex-Mitgruender abgeschlossen, kein P0/P1 offen
- [ ] **Rigorosum Runde 3 (Claude)**: Gesamtnote ≥2.3, alle R1-P0 + R2-P0 geschlossen
- [ ] Website zentria.tech: Features-Liste stimmt mit Delivery ueberein, WASM-Plugin-Claims entfernt
- [ ] AGB, Impressum, Datenschutzerklaerung mit UG-Daten live
- [ ] End-to-End-Smoke aller 14 Module gruen, inkl. Realtime-Flows (Chat+Meetings+Recording) und Tenant-Isolation
- [ ] UG eingetragen (seit 01.06.), Konto aktiv
- [ ] Pilot-kritische Flows gruen: Kalender-Terminbuchung, Dialer-Consent, Passwort-Reset, Demo-Daten korrekt

**Launch / Pilot-0-Go-Live: 2026-09-01** (ZFA-Einstieg, voller P0-Scope inkl. E-Rechnung/GoBD/DATEV/Bexio — Finance-Block-Wellen 4–7 aus `docs/BACKEND-LAUNCH-PLAN.md`)

---

## 8. Superseded Documents

Folgende Dateien werden mit dem Check-in dieser Roadmap zu historischen Referenzen. Kopf-Warnblock ("SUPERSEDED by docs/ROADMAP.md 2026-04-18") wird einfuegt, Inhalt bleibt fuer git-blame und Audit-Trail erhalten.

| Datei | Grund |
|---|---|
| `docs/BUSINESS-ROADMAP.md` | Launch-Datum, Team, Moritz-Status veraltet. Finanzprojektionen wandern in `docs/PRICING.md` |
| `docs/PRODUCT-STRATEGY.md` | Stand Februar 2026, bereits teilweise veraltet durch Audit-Erkenntnisse. Positioning-Kern bleibt in `docs/STRATEGY.md` |
| `docs/I18N-ROADMAP.md` | Migration abgeschlossen, ICU-Fix wird Teil von Sprint 0 |
| `docs/PERFORMANCE-PLAN.md` | Phasen 1–5 abgeschlossen, Redis-Caching in Phase D |
| `docs/DIALER-ROADMAP.md` | Phase 1 abgeschlossen, Phase 2+3 bleiben als strategische Referenz erhalten (nicht SUPERSEDED, nur um-kontextualisiert) |
| Alte `docs/ROADMAP.md` | diese Datei ersetzt sie vollstaendig |

**Nicht betroffen** (bleiben aktiv):
- `docs/STRATEGY.md` — Wettbewerber-Positioning, OnlyOffice-Strategie
- `docs/ARCHITECTURE.md` — ADRs, Technik-Entscheidungen
- `docs/LEARNINGS.md` — Vorgaenger-Projekt-Learnings (slot_booking_webapp)
- `docs/PRICING.md` — Pricing bleibt als eigene Datei
- `docs/PLUGIN_DEVELOPMENT.md` — WASM-Plugin-Guide

---

## 9. Team & Verantwortlichkeiten

| Person | Rolle (2026-04-18) | Commitment |
|---|---|---|
| Luke | Tech-Lead, Full-Stack, Solo bis zum Launch | Vollzeit + 30% GTM |
| Darien | UI/UX, CFO (UG) | On-demand, keine Launch-Pfad-Tasks |
| Nico | QA/Testing, Kunden-Kontakt | 30h/Woche ab Juni |
| Moritz | Marketing (Status: "in der Schwebe") | Nicht einplanen bis geklaert |
| Annabel | Test-Track-Kandidatin (nach CV-Gespraech final) | Q2 2026 wenn committed |
| Ex-Mitgruender | Einmaliger Peer-Review Sprint 3 | 2 Tage Kontingent |

**Luke-Solo-Risiko:** Siehe MEMORY `project_launch_decisions.md`. Bei Krankheit > 3 Tage verschiebt sich der Launch. Puffertag in jedem Sprint eingerechnet.

---

## 10. Entschiedene Strategien (nach Rigorosum Runde 2)

**Am 2026-04-18 nach Runde 2 entschieden:**

1. **Handwerk-Module zum Launch:** Alle 14 Module werden echt (nicht "Beta-Preview"). Safety-Net bleibt der Feature-Flag: falls ein Modul bis Launch-Freeze nicht fertig ist, Feature-Flag OFF + "Coming Q3 2026"-Overlay.
2. **Multi-Tenancy:** **Option-B volle Breitseite jetzt** — ~50 Tabellen bekommen `tenant_id` + Backfill + Index + RLS-Policies in Sprint 2 (Top-20) + Sprint 3 (Rest). Kein spaeteres Retrofit, kein Option-A-Permanent.
3. **Recording-Consent-Modell:** **Join-with-Consent + Banner** — einmaliger Consent-Click beim ersten Call-Beitritt (persistiert als Consent-Record), Recording startet mit persistenter Banner-Anzeige + rotem Mic-Icon, Ablehnung kickt Teilnehmer aus Call. Mitte zwischen nutzerfreundlich und wasserdicht.
4. **Plugin-System:** **Feature-Flag OFF bis Phase D** — Config-Plugins bleiben aktiv, WASM-Runtime nicht instanziert. Ehrlicher Pitch. WASM-Haertung (Ed25519-Signing + WASI-Deny-Set) nur wenn Phase-D-Markt-Signal positiv.
5. **TURN/STUN:** **Build, not Buy** — coturn self-hosted auf eigenem CAX11 (ARM Ampere, ~€3.80/Monat, 20TB Traffic), kein Vendor-Lock auf LiveKit Cloud.
6. **`finance_invoices.line_items`:** **Vor Launch normalisieren** in `finance_invoice_lines`-Tabelle + parallel Finance-Test-Coverage auf ≥50% ausbauen (vorher <15%). GoBD/ZUGFeRD-tauglich machen.
7. **Ein-Launch-Modell** (aktualisiert 2026-06-28; vormals Zwei-Deadline nach Dariens Handover 2026-06-08): **Launch 2026-09-01** — ZFA-Kern + volle P0-Feature-Parität (Finance-Block Wellen 4–7) zusammengelegt. UG-Gruendung 01.06 bleibt stehen. Playbook: `docs/BACKEND-LAUNCH-PLAN.md`.

### Weiterhin offen (nicht launch-blockend)

- **Annabel-Track:** abhaengig von CV-Gespraech. Wenn positiv: Aufgabenpaket und Start-Datum in separater Entscheidung.
- **Preismodell-Rollout:** Default-Annahme — alle Piloten kostenlos bis Q4 2026, ab Phase E (kommerzieller Launch) Stripe/SEPA + Pricing-Seite.

---

*Letztes Update: 2026-06-28 — Launch auf **2026-09-01** korrigiert, Zwei-Deadline-Modell aufgelöst (Pilot-0 + volle P0-Parität = ein Launch), Gate S5 → 31.08, Pilot-/Phase-Fenster mitverschoben. Vorheriger Stand: 2026-06-11 (Zwei-Deadline-Modell synchronisiert).*
*Naechste Ueberarbeitung: Gate S5 (2026-08-31)*
