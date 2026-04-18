# Cosmi — Kern-Roadmap

> **Status:** Sprint 0 abgeschlossen (2026-04-18), alle 9 Tasks gemerged, auf Kurs fuer Launch 01.07.
> **Launch-Datum:** **2026-07-01** (+4 Wochen verschoben nach Runde 2 — 9 neue P0-Launch-Blocker + Option-B-Full + finance-Normalisierung)
> **UG-Gruendung:** 2026-06-01 bleibt, Launch-Tag separat
> **Konsolidiert aus:** ROADMAP (alt), BUSINESS-ROADMAP, PRODUCT-STRATEGY, DIALER-ROADMAP, I18N-ROADMAP, PERFORMANCE-PLAN, .knowledge/milestones.md
> **Eigentuemer dieser Datei:** Luke. Jede andere Roadmap-Datei ist SUPERSEDED.

---

## 1. Context & North Star

### Warum diese Roadmap

Bis 2026-04-18 lagen Produkt-, Technik- und Business-Pfade auf 11 verschiedene Dokumente verteilt. Zwei Rigorosum-Runden am 18.04. haben harte Defizite aufgedeckt: **Runde 1** (Gesamtnote 3.3, wild-wren-Plan) identifizierte 7 P0-Launch-Blocker in Backend/Frontend/Ops. **Runde 2 Vertiefung** (Gesamtnote 4.1, functional-seahorse-Plan) lieferte 9 zusaetzliche P0-Blocker in Integrationen, Realtime-Kern und DB-Schema — darunter ein komplett fehlender TURN/STUN-Server und ein faktisch wirkungsloser Recording-Consent-Check. Kombinierte Launch-Reife: **3.7**.

Konsequenz: Launch von 01.06. auf **01.07. verschoben** (+4 Wochen), um 16 P0-Blocker + Option-B-Full-Retrofit (~50 Tabellen) + finance_line_items-Normalisierung sauber umzusetzen. UG-Gruendung bleibt auf 01.06.

Diese Datei ist die einzige gueltige Roadmap bis zum Launch. Alle anderen werden deprecatet (siehe §8).

### North Star

**Cosmi 1.0 geht am 2026-07-01 live mit:**
- **14 echten Modulen** (keine Mock-Daten mehr in user-sichtbaren Pfaden)
- **Multi-Tenancy Option-B aktiv** (RLS auf ~50 Tabellen, Instanz-pro-Pilot + tenant_id-Isolation — kein Downgrade-Risiko)
- **DSGVO-Consent-Enforcement** in allen Send-Flows (Email, Dialer) + Realtime-Recording (Join-with-Consent + persistenter Banner)
- **Sicherheits-Posture auf Pilot-Niveau** (16 P0-Fixes aus Rigorosum Runde 1 + 2 erledigt)
- **TURN/STUN self-hosted** (coturn auf eigenem CPX11, kein Vendor-Lock)
- **WASM-Plugin-System Feature-Flag OFF** — Config-Plugins aktiv, WASM-Haertung in Phase D (ehrlicher Pitch)
- **`finance_invoices.line_items` normalisiert** (eigene `finance_invoice_lines`-Tabelle, GoBD/ZUGFeRD-tauglich, Finance-Test-Coverage erweitert)
- **Zweiter Review-Zyklus abgeschlossen** (Sprint 5, Peer-Review + Rigorosum Runde 3)
- **Ehrlicher Pitch:** Mobile = PWA auf Desktop-Basis, keine falschen Native-Versprechen

### Pilot-Strategie

- **Pilot-0 (ZFA, Anfang Juli):** Warm-Einstieg, Design-Partner, Dialer-Nutzung, kostenlos
- **Pilot 1–3 (Juli–August, Dienstleister-Segment):** Kostenlos, Feedback-Loop woechentlich, Referenzen aufbauen
- **Handwerk-Piloten:** ab Oktober 2026, wenn Rapporte/Schichten/Fuhrpark auf Branchen-Daten kalibriert sind

---

## 2. Aktueller Stand (Post-Rigorosum 18.04.)

### Was fertig ist

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
| Launch 01.05. → 01.06. | **Launch 01.07.** (nach Runde 2) |
| "11 Industry-Module bleiben auf Mock" (alte ROADMAP §Scope-Entscheidungen) | **Alle 14 Module werden echt** |
| "React Native Scaffold existiert" (alter Audit) | **Mobile-Ordner leer, wird geloescht** |
| Multi-Tenancy Option-A permanent | **Option-B-Full jetzt, ~50 Tabellen Retrofit** |
| WASM-Plugin-System aktiv | **Feature-Flag OFF bis Phase D** (ehrlicher Pitch) |
| Recording ohne Consent-Fluss | **Join-with-Consent + persistenter Banner** |
| TURN/STUN nicht konfiguriert | **coturn self-hosted auf CPX11** |
| `finance_invoices.line_items` JSONB | **normalisierte Tabelle + Backfill** |
| Moritz als GTM-Lead | "In der Schwebe" (siehe MEMORY project_team_ug.md) |
| Demo-Theater als Feature | **Demo-Theater als Launch-Risiko** |

---

## 3. Sprint-Plan bis 2026-07-01

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

**Abgeschlossen:** Alle 7 R1-P0 + R2-P1.2 (WASM-OFF, zusammen mit S0.6) gefixt. Feature-Flag-System live mit 16 Flags (14 Module + `plugins.wasm` + `plugins.config`). Modul-Scope final in `docs/MODULES_SCOPE_MATRIX.md`.

---

### Sprint 1 — Backend-Offensive Teil 1 + R2-P0 Batch A (2026-04-28 – 2026-05-10, 2 Wochen)

**Ziel:** 7 von 14 Modulen echt + die fuenf teuersten R2-P0-Items.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S1.1 | **wiki** (Postgres-FTS, TipTap, Versionen, Share-Links) | 3d | Modul |
| S1.2 | **berichte** (BI-Aggregations-Service, Views, CSV/PDF-Export) | 3d | Modul |
| S1.3 | **formulare** (Form-Schema JSONB, Submissions, Webhook-Trigger) | 4d | Modul |
| S1.4 | **helpdesk** (Tickets, Agenten, Canned, Merge) | 4d | Modul |
| S1.5 | **vertraege** (Laufzeit-Engine, Erinnerungs-Trigger, Skribble-Placeholder) | 3d | Modul |
| S1.6 | **buchhaltung** Completion (FinanzenHook-Gaps, GoBD-Journal) | 2d | Modul |
| S1.7 | **video** Completion (`useVideo`-Hook, Recording-Tagging) | 2d | Modul |
| **S1.R2.1** | **TURN/STUN-Server — coturn self-hosted** auf eigenem CPX11, TURN-URLs im LiveKit-Client-Token + `use_external_ip: true` | 2d | R2-P0.1 |
| **S1.R2.2** | **LiveKit-Secrets Startup-Assertion** (keine Dev-Defaults in Prod) | 2h | R2-P0.2 |
| **S1.R2.3** | **Recording-Consent-Bug:** `StartRecording` uebergibt alle aktiven Call-Teilnehmer als `participantIDs` (`video_grpc.go:213`) | 1d | R2-P0.3 |
| **S1.R2.5** | **Egress-Webhook** ruft `CompleteRecording` (`route_video.go:1153-1176`) | 4h | R2-P0.5 |
| **S1.R2.6** | **Lexware-Webhook HMAC-Signatur-Validierung** (`webhook_handler.go:99-113`) | 1d | R2-P0.6 |

**Parallelitaets-Model:** 4 Git-Worktrees fuer Module + 1 Worktree fuer R2-P0-Batch. Je Worktree 1 Sonnet-Agent als Code-Schreiber, Luke als Reviewer.

**Ende Sprint 1:** 7 Module live, 5 R2-P0 erledigt, Coverage ≥30% pro Modul.

---

### Sprint 2 — Backend-Offensive Teil 2 + R2-P0 Batch B + Option-B Phase 1 (2026-05-11 – 2026-05-24, 2 Wochen)

**Ziel:** Restliche 7 Module + die restlichen R2-P0-Items + Start Option-B-Retrofit (Top-20 Tabellen).

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S2.1 | **rapporte** | 4d | Modul |
| S2.2 | **schichten** (ArbZG-Warnings im Backend) | 4d | Modul |
| S2.3 | **fuhrpark** (TÜV-Reminder) | 3d | Modul |
| S2.4 | **vermietung** (Zustandsprotokolle) | 4d | Modul |
| S2.5 | **inventar** (Bestands-Alarm) | 3d | Modul |
| S2.6 | **einkauf** (Wareneingang) | 3d | Modul |
| S2.7 | **produktion** (Maschinenbelegung) | 3d | Modul |
| **S2.R2.4** | **Frontend Recording-Consent-Modal + persistenter Banner** (Join-with-Consent-Modell: einmaliger Consent-Click beim Call-Beitritt, Banner + rotes Mic-Icon waehrend Aufnahme, Ablehnung → Kick) | 2d | R2-P0.4 |
| **S2.R2.7** | **Offline-Queue** im Desktop-WS-Client: IndexedDB-Buffer fuer Messages, Reconciliation bei Reconnect, Duplicate-Detection | 3d | R2-P0.7 |
| **S2.R2.8** | **`consent_records.created_by`** ON DELETE SET NULL (Migration 000075) | 2h | R2-P0.8 |
| **S2.R2.9** | **`gdpr_deletion_requests.contact_id`** zirkulaere Blockade aufloesen | 4h | R2-P0.9 |
| **S2.MT.1** | **Option-B Phase 1 (Top-20 Tabellen):** `deals`, `activities`, `messages`, `channels`, `channel_memberships`, `notifications`, `events`, `audit_log`, `automations`, `automation_executions`, `calendar_events`, `meetings`, `calls`, `tasks`, `projects`, `team_inboxes`, `inbox_messages`, `document_folders`, `document_files`, `recordings` — je ALTER ADD COLUMN tenant_id + Backfill + Index + RLS-Policy | 5d | Option-B |

**Parallelitaets-Model:** 4 Module-Worktrees + 1 Realtime/R2-P0-Worktree + 1 Multi-Tenancy-Worktree.

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
| S3.5 | Alertmanager + Slack-Webhook | 1d | R1-P1.5 |
| S3.6 | `cd.yml` Auto-Deploy auf main-merge (mit Green-Gate) | 1d | R1-P1.6 |
| S3.7 | Dialer-Test-Coverage 12% → ≥30% | 8d | R1-P1.8 (parallel) |

**Parallelitaets-Model:** 1 Worktree Option-B + 1 Infra/Ansible + 1 Test-Coverage. Multi-Tenancy-Migration muss zuerst fertig sein, bevor Ansible-Blueprint diese Schema-Version kennt.

**Ende Sprint 3:** Alle ~50 Tabellen mit tenant_id + RLS, Ansible-Playbook mit Option-B deployt eine Pilot-Instanz in <30 Min, CI hat trivy/gosec/npm audit gruen.

---

### Sprint 4 — Finance-Normalisierung + P1-Security + Runde-2-P1 (2026-06-08 – 2026-06-21, 2 Wochen)

**Ziel:** finance_invoices.line_items raus aus JSONB, Finance-Test-Coverage hoch, restliche R1-P1 + R2-P1.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S4.FI.1 | **`finance_invoices.line_items` → `finance_invoice_lines`-Tabelle** (Migration + Backfill + Read-Path-Update + Write-Path-Update + ZUGFeRD-Export-Anpassung) | 3d | R2-P1.12 |
| S4.FI.2 | **Finance-Test-Coverage ausbauen** — Service-Level Tests fuer Rechnungserstellung, Positions-Summen, Steuerberechnung, Zahlungs-Verbuchung, Dunning-Flow | 5d | Quality |
| S4.1 | Input-Validation-Framework (`go-playground/validator`) auf 20 kritische Handler | 5d | R1-P1.7 |
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

**Ende Sprint 4:** finance_invoices relational, Finance-Coverage ≥50%, alle R1-P1 + R2-P1 erledigt.

---

### Sprint 5 — Integration, Polish, Pre-Launch-Audit (2026-06-22 – 2026-06-30, 1.5 Wochen)

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

**Ende Sprint 5:** Launch-Freigabe. ZFA-Pilot-0-Onboarding ab 01.07.

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

**Runde 2 P0 (9 Items, Sprint 1+2):**

| # | Task | Status | Sprint |
|---|---|---|---|
| R2-P0.1 | TURN/STUN coturn self-hosted + `use_external_ip: true` | Pending | S1 |
| R2-P0.2 | LiveKit-Secrets Startup-Assertion | Pending | S1 |
| R2-P0.3 | Recording-Consent-Bug (`video_grpc.go:213` — alle Teilnehmer) | Pending | S1 |
| R2-P0.4 | Frontend Recording-Consent-Modal + Banner (Join-with-Consent) | Pending | S2 |
| R2-P0.5 | Egress-Webhook ruft `CompleteRecording` | Pending | S1 |
| R2-P0.6 | Lexware-Webhook HMAC-Signatur-Validierung | Pending | S1 |
| R2-P0.7 | Offline-Queue Desktop-WS (IndexedDB + Reconciliation) | Pending | S2 |
| R2-P0.8 | `consent_records.created_by` ON DELETE SET NULL | Pending | S2 |
| R2-P0.9 | `gdpr_deletion_requests.contact_id` zirkulaere FK aufloesen | Pending | S2 |

### P1 — Vor Pilot-1 (Ende Sprint 4)

**Runde 1 P1 (8 Items, Sprint 3+4):**

| # | Task | Status | Sprint |
|---|---|---|---|
| R1-P1.1 | Ansible Instanz-pro-Pilot (mit Option-B-Schema) | Pending | S3 |
| R1-P1.2 | Dependency-Security-Scans in CI | Pending | S3 |
| R1-P1.3 | Dialer LogCallOutcome Transaktion | Pending | S3 |
| R1-P1.4 | Prod-Image-Tags pinnen | Pending | S3 |
| R1-P1.5 | Alertmanager + Slack | Pending | S3 |
| R1-P1.6 | cd.yml Auto-Deploy | Pending | S3 |
| R1-P1.7 | Input-Validation-Framework | Pending | S4 |
| R1-P1.8 | Dialer-Coverage 12 → 30% | Pending | S3 |

**Runde 2 P1 (12 Items, Sprint 4):**

| # | Task | Status | Sprint |
|---|---|---|---|
| R2-P1.1 | LiveKit-Webhook-Signatur-Validierung | Pending | S4 |
| R2-P1.2 | WASM-Plugin-System Feature-Flag OFF | ✅ Done | S0 (zusammen mit R1-P0.6, PR #11) |
| R2-P1.3 | Automation-Semaphor tenant-isolieren | Pending | S4 |
| R2-P1.4 | Bexio+DATEV Circuit-Breaker/Retry | Pending | S4 |
| R2-P1.5 | StartMeeting Rollen-Check + Organizer-only | Pending | S4 |
| R2-P1.6 | WS-Token in-session revalidieren | Pending | S4 |
| R2-P1.7 | Redis-backed WS-Subscription-State | Pending | S4 |
| R2-P1.8 | Dialer `outcome_id`-Indizes (Migration 000076) | Pending | S4 |
| R2-P1.9 | ~10 FKs ohne ON DELETE nachziehen (Migration 000077) | Pending | S4 |
| R2-P1.10 | Partitionierung + pg_cron-Retention | Pending | S4 |
| R2-P1.11 | `CleanupExpiredRecordings`-Cronjob | Pending | S4 |
| R2-P1.12 | `finance_invoices.line_items` normalisieren | Pending | S4 |

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

## 6. Post-Launch Phase C (Juni – September 2026)

### Phase C — Pilot-Betrieb (Juni – Juli 2026)

- **Track: Legal** — AVV/DPA mit Anwalt finalisieren
- **Track: Onboarding** — ZFA-Pilot-0 + 2 Dienstleister-Piloten (kostenlos)
- **Track: Feedback-Loop** — woechentliche Pilot-Calls, Issue-Tracking, Hotfix-Cycle ≤72h
- **Track: P2-Aufholsprint** — DSGVO-Erasure, Rollen-Route-Guards, Duplikat-Konsolidierung
- **Track: Demo-Video** — Hero (3–5 Min) + Outreach (60–90 Sek)

### Phase D — Production Hardening + Handwerks-Pilot (August – September 2026)

- Redis Caching Layer (3 Tiers) — aus PERFORMANCE-PLAN §5.2
- PgBouncer als DB Connection Pool
- Desktop-Test-Coverage aufbauen (derzeit <5%, Ziel ≥30% fuer Kern-Module)
- Handwerks-Pilot-1 (Rapporte/Schichten/Fuhrpark-Stress-Test)
- OnlyOffice → Collabora Migration (Lizenz-Risiko mitigieren)
- Dialer Phase 2 (PSTN-SIP) — wenn ZFA-Pilot Phase 1 stabil (siehe alter DIALER-ROADMAP)

### Phase E — Kommerzieller Launch (Q4 2026)

- Stripe/SEPA-Integration
- Desktop-Installer-Pipeline (`electron-builder`)
- Self-Hosted ORBIT-Paket (Docker auf Synology NAS)
- Pricing-Seite auf zentria.tech
- LinkedIn + Cold Outreach (UWG §7 konform)
- FinAPI Banking
- Pitch-Deck finalisieren

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

### Gate S1 (2026-05-10)

- [ ] 7 Module live: wiki, berichte, formulare, helpdesk, vertraege, buchhaltung-Completion, video-Completion
- [ ] Coverage pro neuem Modul ≥30%
- [ ] **R2-P0 Batch A erledigt:** TURN/STUN-Server laeuft, LiveKit-Prod-Assertion greift, Recording-Consent uebergibt alle Teilnehmer, Egress-Webhook ruft CompleteRecording, Lexware-HMAC validiert
- [ ] `make test` gruen auf main
- [ ] Smoke-Script durchlaeuft ohne neue Fehler

### Gate S2 (2026-05-24)

- [ ] Alle 14 Module live
- [ ] **Alle 9 R2-P0 erledigt:** Frontend-Consent-Modal + Banner, Offline-Queue, consent_records/gdpr-FKs gefixt
- [ ] **Option-B Phase 1 live:** Top-20 Tabellen mit tenant_id + Backfill + RLS
- [ ] Integration-Test: Tenant-A-User kann Tenant-B-Daten nicht lesen (RLS-Gate)
- [ ] Realtime-Smoke: Call mit 3 Teilnehmern, Recording-Consent von allen, Playback funktioniert
- [ ] Offline-Queue-Test: Electron-Client disconnected, Messages geschickt, reconnected, Server-Reconciliation ohne Duplikate

### Gate S3 (2026-06-07)

- [ ] **Option-B Phase 2 live:** restliche ~30 Tabellen, insgesamt ~50 Tabellen mit tenant_id + RLS
- [ ] Ansible-Playbook: `ansible-playbook bootstrap-pilot.yml -i pilot-test` deployt vollstaendige Cosmi-Instanz auf frischen Hetzner-Host in <30 Min, Option-B-Schema korrekt
- [ ] CI hat trivy+gosec+npm audit, keine High/Critical CVEs
- [ ] Dialer-Coverage ≥30%
- [ ] Alertmanager sendet Test-Alert in Slack
- [ ] cd.yml deployt automatisch bei main-merge

### Gate S4 (2026-06-21)

- [ ] **`finance_invoices.line_items` normalisiert:** `finance_invoice_lines`-Tabelle existiert, Backfill migriert alle bestehenden Rechnungen, ZUGFeRD-Export laeuft gegen neue Tabelle
- [ ] **Finance-Service-Coverage ≥50%** (vorher <15%): Rechnungserstellung, Positions-Summen, Steuerberechnung, Zahlungen, Dunning getestet
- [ ] Alle R1-P1 + R2-P1 erledigt (Input-Validation, LiveKit-HMAC, Automation-Tenant-Isolation, Circuit-Breaker, Meeting-Rollen, WS-Token, Redis-Subscription-State, DB-FKs-Nachzug, Partitionierung, Recording-Cronjob)

### Gate S5 (2026-06-30) — LAUNCH-FREIGABE

- [ ] Peer-Review durch Ex-Mitgruender abgeschlossen, kein P0/P1 offen
- [ ] **Rigorosum Runde 3 (Claude)**: Gesamtnote ≥2.3, alle R1-P0 + R2-P0 geschlossen
- [ ] Website zentria.tech: Features-Liste stimmt mit Delivery ueberein, WASM-Plugin-Claims entfernt
- [ ] AGB, Impressum, Datenschutzerklaerung mit UG-Daten live
- [ ] End-to-End-Smoke aller 14 Module gruen, inkl. Realtime-Flows (Chat+Meetings+Recording) und Tenant-Isolation
- [ ] UG eingetragen (seit 01.06.), Konto aktiv

**Go-Live: 2026-07-01**

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
5. **TURN/STUN:** **Build, not Buy** — coturn self-hosted auf eigenem CPX11 (~€5/Monat), kein Vendor-Lock auf LiveKit Cloud.
6. **`finance_invoices.line_items`:** **Vor Launch normalisieren** in `finance_invoice_lines`-Tabelle + parallel Finance-Test-Coverage auf ≥50% ausbauen (vorher <15%). GoBD/ZUGFeRD-tauglich machen.
7. **Launch-Datum:** **2026-07-01** (+4 Wochen gegenueber alter Planung). UG-Gruendung 01.06 bleibt stehen.

### Weiterhin offen (nicht launch-blockend)

- **Annabel-Track:** abhaengig von CV-Gespraech. Wenn positiv: Aufgabenpaket und Start-Datum in separater Entscheidung.
- **Preismodell-Rollout:** Default-Annahme — alle Piloten kostenlos bis Q4 2026, ab Phase E (kommerzieller Launch) Stripe/SEPA + Pricing-Seite.

---

*Letztes Update: 2026-04-18 (Sprint 0 abgeschlossen — 9/9 Tasks gemerged, Gate S0 bestanden, Sprint 1 startet 2026-04-28)*
*Naechste Ueberarbeitung: Gate S1, 2026-05-10*
