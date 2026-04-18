# Cosmi — Kern-Roadmap

> **Status:** Post-Rigorosum 2026-04-18, konsolidierte Single Source of Truth
> **Launch-Datum:** **2026-06-01** (UG-Gruendung + ZFA-Pilot-0 + Beta-Launch)
> **Konsolidiert aus:** ROADMAP (alt), BUSINESS-ROADMAP, PRODUCT-STRATEGY, DIALER-ROADMAP, I18N-ROADMAP, PERFORMANCE-PLAN, .knowledge/milestones.md
> **Eigentuemer dieser Datei:** Luke. Jede andere Roadmap-Datei ist SUPERSEDED.

---

## 1. Context & North Star

### Warum diese Roadmap

Bis 2026-04-18 lagen Produkt-, Technik- und Business-Pfade auf 11 verschiedene Dokumente verteilt. Das Rigorosum vom 18.04. hat harte Defizite aufgedeckt (Gesamtnote 3.3), darunter launch-verhindernde Sicherheits- und Compliance-Luecken. Gleichzeitig wurde der Launch von 01.05. auf **01.06. verschoben**, um einen ambitionierten Scope sauber umzusetzen statt mit nur 6 echten Modulen an den Markt zu gehen.

Diese Datei ist die einzige gueltige Roadmap bis zum Launch. Alle anderen werden deprecatet (siehe §8).

### North Star

**Cosmi 1.0 geht am 2026-06-01 live mit:**
- **14 echten Modulen** (keine Mock-Daten mehr in user-sichtbaren Pfaden)
- **Sauberem Multi-Tenancy-Blueprint** (Option A: Instanz-pro-Pilot, Ansible-automatisiert)
- **DSGVO-Consent-Enforcement** in allen Send-Flows (Email, Dialer)
- **Sicherheits-Posture auf Pilot-Niveau** (7 P0-Fixes aus Rigorosum erledigt)
- **Zweiter, unabhaengiger Review-Zyklus abgeschlossen** (Sprint 4, Peer-Review-Option nutzen)
- **Ehrlicher Pitch:** Mobile = PWA auf Desktop-Basis, keine falschen Native-Versprechen

### Pilot-Strategie

- **Pilot-0 (ZFA, Anfang Juni):** Warm-Einstieg, Design-Partner, Dialer-Nutzung, kostenlos
- **Pilot 1–3 (Juni–Juli, Dienstleister-Segment):** Kostenlos, Feedback-Loop woechentlich, Referenzen aufbauen
- **Handwerk-Piloten:** ab Monat 4 (September 2026), wenn Rapporte/Schichten/Fuhrpark auf Branchen-Daten kalibriert sind

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

### Was aus dem Rigorosum zaehlt

- **Kapitel-Noten:** Backend 3.3 · Frontend 3.3 · Mobile 5.0 (leer) · Ops/Sec 2.7 · Tests 4.0 · Launch-Readiness 3.7
- **7 P0-Launch-Blocker** (siehe §4)
- **8 P1-Items** (vor Pilot-1, siehe §4)
- **14 Frontend-only-Module** — werden jetzt alle echt (siehe §5)
- **Mobile-Ordner leer** — wird geloescht, Pitch korrigiert (siehe §3 Sprint 0)
- Vollstaendiges Rigorosum: `~/.claude/plans/wir-hatten-ja-schonmal-wild-wren.md`

### Was sich geaendert hat

| Alt | Neu |
|---|---|
| Launch 01.05. | **Launch 01.06.** |
| "11 Industry-Module bleiben auf Mock" (alte ROADMAP §Scope-Entscheidungen) | **Alle 14 Module werden echt** |
| "React Native Scaffold existiert" (alter Audit) | **Mobile-Ordner leer, wird geloescht** |
| Moritz als GTM-Lead | "In der Schwebe" (siehe MEMORY project_team_ug.md) |
| Demo-Theater als Feature | **Demo-Theater als Launch-Risiko** |

---

## 3. Sprint-Plan bis 2026-06-01

**6 Wochen = 4 Sprints a ~1.5 Wochen. Parallelitaet via Git-Worktrees + Sub-Agenten (Sonnet fuer Code-Volumen, Opus fuer Architektur-Entscheidungen).**

### Sprint 0 — Launch-Blocker-Sweep (2026-04-21 – 2026-04-27, 1 Woche)

**Ziel:** Alle P0-Blocker aus dem Rigorosum abraeumen. Modul-Scope final. Bibliotheken-Prerequisites.

| # | Task | Aufwand | Owner | Verifikation |
|---|---|---|---|---|
| S0.1 | Migration 000071: `consent_records.contact_id` CASCADE → SET NULL | 4h | Luke | `\d consent_records` |
| S0.2 | `assertConsent()`-Wrapper vor `SendEmail` + `InitiateDialerCall` | 3d | Luke+Sonnet | Integration-Test schlaegt ohne Consent fehl |
| S0.3 | `WOPI_JWT_SECRET`, `MINIO_SECRET_KEY`, `VAULT_MASTER_SECRET` mit `required: true` + Startup-Assertion | 2h | Luke | Service-Start ohne Env-Var → harter Abbruch |
| S0.4 | DOMPurify in 7 `dangerouslySetInnerHTML`-Stellen (insb. `MailsPage.tsx:577`, `WikiArticle.tsx:72`) | 1d | Luke+Sonnet | grep zeigt nur noch sanitize-wrapped Aufrufe |
| S0.5 | OnlyOffice JWT in Prod-Override explizit `JWT_ENABLED: "true"` + Secret-Sync | 2h | Luke | OnlyOffice-Request ohne JWT → 401 |
| S0.6 | Feature-Flag-Registry (Config-basiert) — Safety Net fuer Modul-Slippage | 2d | Luke+Sonnet | Nav zeigt nur aktivierte Module, Route-Direct-Access → 404 |
| S0.7 | ICU-Plural-Klammern-Fix in de.json/en.json/fr.json/it.json (18 Strings × 4 Sprachen) | 1h | Luke+Sonnet | i18n-Test mit `count: 5` rendert korrekt |
| S0.8 | `mobile/`-Ordner loeschen, MEMORY.md + Website-Pitch korrigieren (PWA statt Native) | 2h | Luke | Ordner weg, Pitch ehrlich |
| S0.9 | 14-Modul-Backend-Scope-Matrix erstellen (siehe §5) | 1d | Luke | Matrix dokumentiert pro Modul: Tabellen, gRPC-RPCs, Abhaengigkeiten |

**Ende Sprint 0:** Alle P0 gefixt, Feature-Flag-System live, Modul-Scope final.

---

### Sprint 1 — Backend-Offensive Teil 1 (2026-04-28 – 2026-05-10, 2 Wochen)

**Ziel:** 7 von 14 Modulen voll ans Backend angebunden. Parallelisierung per Worktree (3–4 Module gleichzeitig mit Sonnet-Sub-Agenten).

Reihenfolge nach Aufwand und Abhaengigkeit:

| # | Modul | Kern-Scope | Aufwand | Abhaengigkeit |
|---|---|---|---|---|
| S1.1 | **wiki** | Postgres-FTS, TipTap-Content speichern, Versionen, Share-Links | 3d | — |
| S1.2 | **berichte** | BI-Aggregations-Service, View-Definitionen in DB, CSV/PDF-Export | 3d | biz, crm (read) |
| S1.3 | **formulare** | Form-Schema JSONB, Submissions-Table, Webhook-Trigger fuer Automation | 4d | automation |
| S1.4 | **helpdesk** | Tickets, Agenten-Zuweisung, Canned Responses, Merge-Funktion | 4d | auth, email |
| S1.5 | **vertraege** | Vertrags-Entity, Laufzeit-Engine, Erinnerungs-Trigger, Skribble-Placeholder | 3d | crm |
| S1.6 | **buchhaltung** (Completion) | Fehlende FinanzenHook-Gaps schliessen, GoBD-konforme Journal-Tabelle | 2d | biz (exists) |
| S1.7 | **video** (Completion) | `useVideo`-Hook vervollstaendigen, Recording-Tagging in Meetings | 2d | work/livekit (exists) |

**Parallelitaets-Model:** 4 Git-Worktrees (`wt-wiki`, `wt-formulare`, `wt-helpdesk`, `wt-vertraege`) + 1 Main-Branch fuer Completions. Je Worktree 1 Sonnet-Agent als Code-Schreiber, Luke als Reviewer.

**Ende Sprint 1:** 7 Module live auf main, Coverage ≥30% pro Modul (Regel: kein Merge ohne Tests).

---

### Sprint 2 — Backend-Offensive Teil 2 + Multi-Tenancy (2026-05-11 – 2026-05-24, 2 Wochen)

**Ziel:** Restliche 7 Module + Instanz-pro-Pilot-Blueprint + P1-Blocker.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S2.1 | **rapporte** (Arbeitsrapporte Handwerk) | 4d | Modul |
| S2.2 | **schichten** (Schichtplanung mit ArbZG-Warnings auch im Backend) | 4d | Modul |
| S2.3 | **fuhrpark** (Fahrzeuge, Schadensmeldungen, Termine) | 3d | Modul |
| S2.4 | **vermietung** (Mietgeraete, Zustandsprotokolle, Rueckgabe-Flow) | 4d | Modul |
| S2.5 | **inventar** (Lager, Bestandsbewegungen, Bestands-Warnungen) | 3d | Modul |
| S2.6 | **einkauf** (Bestellungen, Lieferanten, Wareneingangs-Flow) | 3d | Modul |
| S2.7 | **produktion** (Produktionsplanung, Maschinenbelegung) | 3d | Modul |
| S2.8 | Ansible-Playbook Instanz-pro-Pilot (Hetzner-VPS bootstrappen, Secrets generieren, Docker-Stack deployen, Backup-Cron, Caddy-Domain) | 5d | P1.1 |
| S2.9 | Dependency-Security-Scans (`trivy`, `gosec`, `npm audit`) in CI | 2d | P1.2 |
| S2.10 | Dialer `LogCallOutcome` in Transaktion + Integration-Test | 2d | P1.3 |
| S2.11 | Prod-Image-Tags pinnen (7× `latest` → Version-Hash) | 30min | P1.4 |
| S2.12 | Alertmanager + Slack-Webhook | 1d | P1.5 |
| S2.13 | `cd.yml` Auto-Deploy auf main-merge (mit Green-Gate) | 1d | P1.6 |
| S2.14 | Dialer-Test-Coverage 12% → ≥30% | 8d | P1.8 (parallel zu Module) |

**Parallelitaets-Model:** 4 Worktrees fuer Module + 1 Worktree fuer Infrastruktur (Ansible/CI) + 1 fuer Test-Coverage-Sprint.

**Ende Sprint 2:** Alle 14 Module auf main, Ansible-Blueprint tested, Dialer ≥30% Coverage, Prod-Security-Posture auf P1-Niveau.

---

### Sprint 3 — Integration, Polish, Pre-Launch-Audit (2026-05-25 – 2026-05-31, 1 Woche)

**Ziel:** End-to-End-Tests, Content-Cleanup, zweite Review-Runde, Launch-Freigabe.

| # | Task | Aufwand | Kategorie |
|---|---|---|---|
| S3.1 | Website-Content-Audit (zentria.tech) — "Viele Versprechungen, wenig dahinter" fixen: Features-Liste mit realem Delivery-Status abgleichen | 1d | GTM |
| S3.2 | Input-Validation-Framework (`go-playground/validator`) ausrollen auf 20 kritische Handler | 5d | P1.7 |
| S3.3 | End-to-End-Smoke-Test: alle 14 neuen Module durchklicken, Daten persistieren, Reload, Rollen-Check | 1d | QA |
| S3.4 | Legal: AGB, Impressum, Datenschutzerklaerung final (mit UG-Daten nach 01.06.) | 1d | Legal |
| S3.5 | Peer-Review durch Luke's Ex-Mitgruender (Sonderaktivierung) | 2d extern | Review |
| S3.6 | **Review-Zyklus 2** (Rigorosum Runde 2): Claude re-auditiert alle neuen Module + Ansible-Blueprint | 1d | Review |
| S3.7 | Finaler Smoke auf Prod (`deploy/scripts/smoke.sh`) | 30min | Launch |
| S3.8 | UG-Notartermin wahrnehmen, Konto eroeffnen | 1d extern | Legal |

**Ende Sprint 3:** Launch-Freigabe. ZFA-Pilot-0-Onboarding ab 01.06.

---

## 4. Launch-Readiness-Checkliste (P0–P3)

Quelle: Rigorosum 2026-04-18, Teil III. Priorisierung bleibt bestehen, Deadlines passen sich an neuen Launch an.

### P0 — Launch-Blocker (Ende Sprint 0)

| # | Task | Status | Sprint |
|---|---|---|---|
| P0.1 | Migration 000071 CASCADE → SET NULL | Pending | S0 |
| P0.2 | `assertConsent()` vor Email/Dialer-Send | Pending | S0 |
| P0.3 | Prod-Secrets required | Pending | S0 |
| P0.4 | DOMPurify in dangerouslySetInnerHTML | Pending | S0 |
| P0.5 | OnlyOffice JWT in Prod | Pending | S0 |
| P0.6 | Feature-Flag-Registry | Pending | S0 |
| P0.7 | ICU-Plural-Klammern-Fix | Pending | S0 |

### P1 — Vor Pilot-1 (Ende Sprint 2)

| # | Task | Status | Sprint |
|---|---|---|---|
| P1.1 | Ansible Instanz-pro-Pilot | Pending | S2 |
| P1.2 | Dependency-Security-Scans in CI | Pending | S2 |
| P1.3 | Dialer LogCallOutcome Transaktion | Pending | S2 |
| P1.4 | Prod-Image-Tags pinnen | Pending | S2 |
| P1.5 | Alertmanager + Slack | Pending | S2 |
| P1.6 | cd.yml Auto-Deploy | Pending | S2 |
| P1.7 | Input-Validation-Framework | Pending | S3 |
| P1.8 | Dialer-Coverage 12 → 30% | Pending | S2 |

### P2 — Vor Pilot-Skalierung (Post-Launch Phase C, Juli–September)

P2.1 DSGVO-Erasure Stores · P2.2 Rollen-Route-Guards + DEV_BYPASS_AUTH-Hardening · P2.3 Duplikat-Komponenten konsolidieren · P2.4 Virtualisierung Top-10-Listen · P2.5 CSP in Caddy · P2.6 crm/contacts ↔ kontakte-Konsolidierung · P2.7 Error-Wrapping-Hygiene · P2.8 .pb.go aus Repo + buf generate

### P3 — Nice-to-Have (Phase D, ab Oktober 2026)

P3.1 Go-Stable-Version · P3.2 Panic-Fix automation/templates.go · P3.3 Naming (email/mails, biz/finanzen/buchhaltung, kalender/calendar) · P3.4 Tenant-scoped Rate-Limit · P3.5 React.memo Review · P3.6 Hard-coded Hex → CSS-Vars · P3.7 Desktop Auto-Update · P3.8 Loki/Log-Aggregation · P3.9 Mobile-Scaffold (wenn Markt es verlangt)

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

### Gate S0 (2026-04-27)

- [ ] Alle 7 P0-Items erledigt, Migration 000071 deployed
- [ ] `grep -rn "dangerouslySetInnerHTML" desktop/src/renderer/src/` zeigt nur sanitize-wrapped
- [ ] Service-Start ohne `WOPI_JWT_SECRET` bricht ab
- [ ] Feature-Flag-Registry versteckt alle 14 alten Mock-Routen
- [ ] i18n-Test fuer ICU-Plurals grün
- [ ] `mobile/`-Ordner weg, Pitch korrigiert
- [ ] 14-Modul-Scope-Matrix in diesem Dokument gepflegt

### Gate S1 (2026-05-10)

- [ ] 7 Module live: wiki, berichte, formulare, helpdesk, vertraege, buchhaltung-Completion, video-Completion
- [ ] Coverage pro neuem Modul ≥30%
- [ ] `make test` grün auf main
- [ ] Smoke-Script durchlaeuft ohne neue Fehler

### Gate S2 (2026-05-24)

- [ ] Alle 14 Module live
- [ ] Ansible-Playbook: `ansible-playbook bootstrap-pilot.yml -i pilot-test` deployt vollstaendige Cosmi-Instanz auf frischen Hetzner-Host in <30 Min
- [ ] CI hat trivy+gosec+npm audit, keine High/Critical CVEs
- [ ] Dialer-Coverage ≥30%
- [ ] Alertmanager sendet Test-Alert in Slack
- [ ] cd.yml deployt automatisch bei main-merge

### Gate S3 (2026-05-31) — LAUNCH-FREIGABE

- [ ] Peer-Review durch Ex-Mitgruender abgeschlossen, kein P0/P1 offen
- [ ] Rigorosum Runde 2 (Claude): Gesamtnote ≥2.3
- [ ] Website zentria.tech: Features-Liste stimmt mit Delivery ueberein
- [ ] AGB, Impressum, Datenschutzerklaerung mit UG-Daten live
- [ ] End-to-End-Smoke aller 14 Module grün
- [ ] UG eingetragen, Konto live

**Go-Live: 2026-06-01**

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

## 10. Offene strategische Entscheidungen

Wird im naechsten Austausch zwischen Luke und Claude (Rigorosum Runde 2, Ende Sprint 3) bestaetigt oder verworfen:

1. Handwerk-Module (rapporte, schichten, fuhrpark, vermietung, produktion) wirklich zum Launch echt, auch wenn kein Handwerks-Pilot dabei ist? Alternative: Dienstleister-Piloten zum Launch, Handwerks-Module als "Beta-Preview" kennzeichnen.
2. Multi-Tenancy Option A bis Skalierung durchhalten oder ab Pilot-3 auf RLS migrieren?
3. Annabel-Track — wenn CV-Gespraech positiv: welches Aufgabenpaket ab wann?
4. Preismodell-Rollout: ab Pilot-0 kostenlos, ab Pilot-1 rabattiert (50%), ab Pilot-4 Vollpreis? Oder alle Piloten kostenlos bis Q4?

---

*Letztes Update: 2026-04-18 (Post-Rigorosum-Konsolidierung)*
*Naechste Ueberarbeitung: Gate S3, 2026-05-31*
