# Backend Launch-Plan — Wellen-Playbook bis Launch (01.09)

> ⚠️ **KALENDERMODELL ÜBERHOLT** am 2026-08-12 durch `.planning/launch-lagebild-2026-08-12.md`.
> Das Launch-Datum 2026-09-01 im Titel, der ZFA-Pilot und die Wellen-Terminierung sind entwertet.
> Gültige Sequenz: Reifegrad-Gates Etappe 0–4 (Lagebild §6). Die fachlichen Wellen-Inhalte und
> Dependency-Reihenfolge unten bleiben brauchbar — die Daten nicht.

> **Zweck:** Dependency-geordnetes Wellen-Playbook für die Backend-Arbeit Richtung Launch. Jede Welle ist ein
> **eigenständiges Auftragspaket** — du kannst sie einzeln in einer Session starten ODER 2–3 als Chain
> durchlaufen lassen. Dieses Dokument ist die Informationsquelle; die nächste Session liest es, wählt die
> Welle(n), exploriert die genaue Code-Basis und führt in Subagent-Wellen aus (Muster wie S4.1).
>
> **Quellen:** Dariens Backend-Handover (Anhang A, verbatim) + `.planning/backend-gaps.md` (Detail-Arbeitsnotiz) +
> `docs/sprint4-finance-normalization-plan.md` (Welle 4 schon vorgeplant).
> **Stand:** 2026-06-08 · Erstellt nach Abschluss S4.1 (Input-Validation, alle 4 Wellen, `f263f974`).

---

## 0. Kontext & Timeline

- **Arbeitsteilung:** Claude/Darien = Frontend (mock-first, viele Module gebaut), **Luke = Backend**. Bei
  🟢-FE-Punkten genügt der Endpoint → FE tauscht Mock-Store gegen TanStack-Hook ohne weitere Abstimmung.
- **Ein Launch am 01.09 (Stand 2026-06-28):** Pilot-0 und volle P0-Feature-Parität fallen auf **2026-09-01** zusammen — das frühere Zwei-Deadline-Modell ist aufgelöst. Die Wellen-Reihenfolge unten bleibt als **Prioritäts-Sequenz** gültig (zuerst Pilot-kritisch, dann Finance-Block), alle Deadlines ≤01.09:
  - **Pilot-kritisch (zuerst):** Online-Terminbuchung, Dialer-Consent-Absicherung, Passwort-Reset, korrekte Demo-Daten.
  - **Voller P0-Scope (danach, bis Launch):** E-Rechnung/GoBD/DATEV/Bexio + Rest.
  - ✅ **ROADMAP-Sync 2026-06-28:** `docs/ROADMAP.md` auf Ein-Launch-Modell (2026-09-01) aktualisiert, Gate S5 → 31.08, BACKEND-LAUNCH-PLAN.md als Playbook verlinkt.
- **Aktueller Stand main:** `29e77fb7` (S4.1 alle 4 Wellen). **R1-P1.7 Input-Validation ✅ KOMPLETT** — `decodeAndValidate[T]` in allen JSON-Body-Mutation-Handlern über ~52 Route-Files (verbleibende GET-only/Protokoll-Handler: Webhook/WOPI/CalDAV/proto-direct-Passthrough bewusst ausgenommen). Production auf CCX-Migration/TURN wartet noch (Pilot-0-IP-Bestellung gebündelt).

### Wie dieses Dokument zu benutzen ist
1. Welle(n) wählen (Reihenfolge/Chains siehe §3 + §4).
2. Neue Session öffnen → dieses Doc + die genannten Quell-Dateien lesen lassen → **Plan-Mode-Ultrathink** für die
   gewählte Welle (genaue Proto/Service/Repo/Route-Stellen verifizieren, Migrations-Nummer ermitteln).
3. Ausführung in Subagent-Wellen (disjunkte Pakete, Sonnet) — Build-Standard unten ist verbindlich.
4. Pro Welle: konsolidierte Verifikation → **Pause/Review** → commit+push (rebase auf Darien zuerst).

---

## 1. Build-Standard (verbindlich für JEDE Welle)

Aus `CLAUDE.md` + `.knowledge/` — gilt für jeden neuen Endpoint/Service/Migration:

- **Thick Services, Thin Handlers.** Business-Logik im Service, Gateway-Handler nur Parse → Validate → Call → Respond.
- **Input-Validation:** jeder neue Mutation-Handler nutzt `gateway.decodeAndValidate[T]` + `validate`-Tags
  (Foundation steht seit S4.1; DACH-Validatoren `iban`/`bic`/`plz_dach`/`phone_dach`/`ustid_dach`/`steuernr` verfügbar).
- **API-First:** OpenAPI-Spec (`backend/api/openapi.yaml`) **vor** Implementation; bei 🔴-Items Contract-Shape mit Darien/Claude abstimmen.
- **Migrations via golang-migrate** (`make migrate-create name=xxx`), NIE manuell. **Jede neue Tabelle MUSS `tenant_id`** haben
  (NOT NULL + RLS-fähig; Option-B). Pre-RLS-Audit-Regel beachten (Schema + Repo-INSERT + SELECT-Scan).
- **RBAC:** neue `RequirePermission`-Guards brauchen **Seed-Migration** (sonst 403 für alle, inkl. Admin).
- **Idempotenz:** POST mit Seiteneffekt (Finance-Postings, Bookings, Webhooks) → Idempotency-Key.
- **Consent:** Outbound an Kontakte (Mail/Call) → `AssertConsent` (siehe Welle 0 Dialer-Fix).
- **Structured Logging (slog)**, kein `fmt.Println`. **Secrets** nur via Env/`config.Load(ctx, ...Requirement)`.
- **Tests:** kritische Pfade (Auth/Payments/Data) 60%+, gesamt 15%+ (CI-enforced). Tenant-Isolation-Test je neue Tabelle.
- **Git:** Conventional Commits, English imperative, **keine AI-Attribution**, direct-to-main, **`git pull --rebase` vor
  JEDEM Push** (Darien pusht parallel reines `desktop/` → konfliktfrei). Subagenten: **NIE `git stash/pull/reset/checkout`**
  im geteilten Tree (S4.1-Lesson).
- **Verifikation pro Welle:** scoped `go build`/`go vet`/`go test` der betroffenen Packages; Migration up+down lokal testen;
  nach Push CD-Smoke 24/24 beobachten.

---

## 2. Prioritäts-Logik (Launch-Impact, nicht Modul-Wichtigkeit)

- **Pilot-kritisch (≤ 01.09):** Welle 0 (Risk-Fixes), Welle 2 (Passwort-Reset), Welle 3 (Kalender-Booking).
- **Voll-P0 (≤ 01.09):** Welle 4–7 (Finance-Block), Rest von Welle 0/1.
- **Fundament (so früh wie möglich, entsperrt viele Module):** Welle 1 (Settings-Scope + Upload + Signatur).
- **P1 Feature-Parität (FE scharfschalten):** Welle 8–11.
- **P2 Post-Launch:** §5-Backlog (Branchen-Module, Komfort, Architektur).

---

## 3. Dependency-Graph (was blockiert was)

```
Welle 0  Quick Wins/Risk-Fixes ───────────────► (unabhängig, sofort)
Welle 1  Settings-Scope + Upload + Signatur ──► entsperrt: work/team/kontakte/finanzen-Settings,
                                                 profil-Avatar, chat/rapporte/vermietung/vertraege-Upload+Signatur
Welle 2  Passwort-Reset ───────────────────────► entsperrt: Login-Redesign (FE)
Welle 3  Kalender Public-Booking ──────────────► entsperrt: ZFA-Pilot-Akquise
Welle 4  finance_line_items Normalisierung ────► PREREQ für Welle 5 (E-Rechnung) + Welle 10 (Rechnungen am Kontakt)
   └─► Welle 5  E-Rechnung (XRechnung+ZUGFeRD)
   └─► Welle 6  DATEV-EXTF + GoBD-WORM
   └─► Welle 7  Bexio (CH) OAuth2+Sync
Welle 8  kommunikation P1 (Reactions/Inbox/Canned) ─► unabhängig (chat/inbox-Services existieren)
Welle 9  work P1 (Labels/Custom-Fields/...) ───────► profitiert von Welle 1 (tenant_settings)
Welle 10 kontakte P1 (Contracts/Invoices/Advisory/Leads) ─► Invoices-Sektion braucht Welle 4
Welle 11 team P1 (Payroll/DATEV-HR) ───────────────► profitiert von Welle 1 (tenant_settings)
```

**Cross-cutting-Hebel:** Welle 1 zuerst zahlt sich am stärksten aus — danach fällt in vielen P1-Modulen nur noch der Hook-Tausch an.

---

## 4. Die Wellen (Auftragspakete)

> Format je Welle: **Ziel · Deadline · Prereq · Backend-Scope · Contract-Entscheidungen · FE-Status · Verifikation · Aufwand**.
> 🟢 = FE wiring-ready (Endpoint genügt) · 🟡 = FE teilweise · 🔴 = FE wartet / Contract vorher abstimmen.

### Welle 0 — Quick Wins + Risk-Fixes ✅ ERLEDIGT 2026-06-11
- **Ziel:** Compliance-Loch schließen + mehrere Module visuell entsperren. **Deadline:** Pilot (≤01.09). **Prereq:** keine. **Aufwand:** ½–1 Tag.
- **Backend-Scope:**
  1. **Dialer-Consent-Fix (⚠ DSGVO) — ✅ ERLEDIGT 2026-06-09 (Chain PILOT):** `cmd/dialer/main.go:92` nutzt jetzt `NewServiceWithConsent` mit `crm/consent`-Asserter verdrahtet. Regressionstest grün. Keine Anrufe mehr ohne Consent möglich.
  2. **Demo-`userName`-Fix — ✅ ERLEDIGT 2026-06-11 (`7a367047`):** `mocks/handlers/team.ts` auf `EmployeeProfile`-camelCase-Shape (`userName`) angeglichen. Demo-Login → team/Mentions zeigen jetzt Namen. ⚠ Hinweis: Das echte Gateway serialisiert `hr.pb.go` via `encoding/json` mit snake_case-Tags (`user_name`) — Demo-Mode ist jetzt camelCase-konsistent; gegen das echte Backend würde das team-Modul weiterhin „Unbekannt" zeigen (FE↔BE-Shape-Mismatch, siehe `.planning/backend-gaps.md` §team).
- **Verifikation:** ✅ Dialer-Consent-Test rot→grün; Demo-Login → team/Mentions zeigen Namen.

### Welle 2 — Auth P0: Passwort-Reset
- **Ziel:** „Passwort vergessen"-Flow. **Deadline:** Pilot (≤01.09). **Prereq:** keine. **FE:** 🔴 (Login-Redesign wartet darauf). **Aufwand:** ~1 Tag.
- **Backend-Scope:** auth-Service: `POST /api/v1/auth/forgot-password` (Mail-Token, rate-limited, kein User-Enumeration-Leak)
  + `POST /api/v1/auth/reset-password` (Token + neues Passwort, Strength-Check via vorhandenem `go-password-validator`).
  Token-Tabelle (`password_reset_tokens`, tenant_id, expiry, single-use) + Mail-Versand über vorhandenen email-Dispatcher.
- **Verifikation:** Token-Lifecycle-Test (issue → reset → invalidate), Expiry, Reuse-Block.

### Welle 1 — Cross-cutting Fundament: Settings-Scope + Upload + Signatur
- **Ziel:** Foundation, die FE-weit localStorage → echte Persistenz umschaltet + Datei/Foto/Signatur überall ermöglicht.
  **Deadline:** so früh wie möglich (≤01.09, aber höchster Hebel). **Prereq:** keine. **Aufwand:** 2–3 Tage.
- **Backend-Scope:**
  1. **Settings-Scope-Hierarchie** (Tenant-Default → Modul-Leiter-Override → User-Override):
     - `tenant_module_leads` (tenant_id, user_id, module_id, granted_by, granted_at) + `GET /api/v1/tenant/module-leads?user_id=`,
       `PUT/DELETE .../module-leads/{user_id}/{module_id}`.
     - `tenant_settings` (tenant_id, module_id, key, value — nur Modul-Leiter/Admin schreibbar) + `user_settings` (user_id, module_id, key, value).
     - **Resolve-Reihenfolge + RBAC serverseitig erzwingen.** FE (`ModuleSettingsShell`, `useIsModuleLead`) hängt komplett dran. 🟢
  2. **Generischer S3/MinIO-Upload-Service:** `POST /api/v1/files/upload` (generisch, scoped) — gebraucht von profil-Avatar,
     kontakte, chat, später fuhrpark/inventar/rapporte/vermietung. MinIO-Infra existiert (document/chat nutzen sie). 🟢
  3. **Signatur-Persistenz:** `signature`-Feld/-Endpoint — vertraege/rapporte/vermietung. FE `SignatureCanvas` existiert. 🟢
- **Contract-Entscheidung:** Settings-`value`-Typ (JSON-Blob vs. typed) + key-Namespace-Konvention (`module.key`) festlegen.
- **Verifikation:** Resolve-Order-Test (user > lead > tenant), RBAC-Schreibschutz-Test, Upload-Roundtrip, Tenant-Isolation.

### Welle 3 — Kalender Public-Booking (ZFA-Pilot-kritisch)
- **Ziel:** Öffentliche Online-Terminbuchung. **Deadline:** Pilot (≤01.09) — Akquise hängt daran. **Prereq:** keine (calendar-Service existiert). **FE:** 🟢 (Flow als Mock fertig). **Aufwand:** 2–3 Tage.
- **Backend-Scope:**
  - `GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` (Slug, Services, Verfügbarkeitsregeln) — authenticated.
  - `GET /api/v1/public/book/:slug` — **öffentlich/unauthenticated** (eigene Route-Gruppe ohne authMiddleware, Rate-Limit + Bot-Schutz).
  - `GET .../availability?date=&service=` — freie Slots aus Kalender-Belegung berechnen.
  - `POST /api/v1/public/bookings` — öffentliche Terminanlage → Event + Bestätigungsmail (idempotent gegen Doppel-Submit).
- **Contract-Entscheidung:** Verfügbarkeitsregel-Modell (Wochenslots/Puffer/Vorlaufzeit) + öffentliches Buchungs-Payload-Shape.
- **Verifikation:** unauth-Route greift ohne JWT; Slot-Kollision; Consent/DSGVO für öffentliche Kontaktdaten; Idempotenz.

### Welle 4 — Finance-Fundament: `finance_line_items` Normalisierung (ADR-0007)
- **Ziel:** Relationale `finance_invoice_lines`-Tabelle. **Deadline:** ≤01.09, aber **PREREQ für Welle 5 + Welle 10**.
  **Prereq:** keine. **Aufwand:** 2–3 Tage. **Plan liegt vor:** `docs/sprint4-finance-normalization-plan.md` (nicht neu planen, ausführen).
- **Backend-Scope:** Migration `finance_invoice_lines` (tenant_id, invoice_id-FK, contact-fähig) + Backfill der JSONB-`line_items`
  + Read-Path + Write-Path + ZUGFeRD-/Export-Anpassung. DB-CHECKs (`quantity>0`, `tax_rate 0–100`) bereits in Migr. 132 (ADR-0007).
- **Verifikation:** Backfill-Idempotenz, Summen-Konsistenz vor/nach, Export gegen neue Tabelle, kein Datenverlust.

### Welle 5 — Finance E-Rechnung (XRechnung + ZUGFeRD)
- **Ziel:** Gesetzliche E-Rechnung (DE Empfangspflicht seit 01.01.2025). **Deadline:** ≤01.09. **Prereq:** Welle 4. **FE:** 🟢 (nur Download/Status). **Aufwand:** groß (1–2 Wochen).
- **Backend-Scope:** Generator-Service: XRechnung-UBL + ZUGFeRD 2.x (EN-16931) erzeugen (Ausgang) →
  `POST /api/v1/finance/invoices/{id}/erechnung` (Format-Param). + Eingangs-Parser-Endpoint (XML-Extraktion).
- **Contract-Entscheidung:** Format-Enum (`xrechnung`/`zugferd`), Response (PDF/A-3 mit eingebettetem XML vs. reines XML), Validierungs-Level.
- **Verifikation:** gegen offizielle EN-16931-Validatoren (KoSIT-Validator-Testvektoren), Roundtrip Ausgang→Eingang.

### Welle 6 — Finance DATEV-EXTF-Export + GoBD-WORM-Archiv
- **Ziel:** Steuerberater-Übergabe (DE) + revisionssicheres Belegarchiv. **Deadline:** ≤01.09. **Prereq:** Welle 4. **FE:** 🟢 (Export-UI + `FinanceSettingsPanel` da). **Aufwand:** mittel-groß.
- **Backend-Scope:**
  1. **DATEV-EXTF-Export:** Buchungsstapel (EXTF ASCII/CSV, Windows-1252) + Belegbilder-ZIP. Export-Job + Settings
     (Berater-Nr., Mandanten-Nr., Sachkonto-Länge, Steuerkennzeichen→Konto-Mapping). Spec öffentlich, kein Marktplatz-Partner nötig.
  2. **GoBD-Belegarchiv:** WORM-Storage (unveränderbar) + `document_events`-Audit + Retention-Policy (8 Jahre).
- **Contract-Entscheidung:** Kontenrahmen-Default (SKR03/04), Steuerkennzeichen-Mapping-Shape.
- **Verifikation:** EXTF-Header/Encoding gegen DATEV-Spec, WORM-Unveränderbarkeit (Update/Delete blockiert), Audit-Vollständigkeit.

### Welle 7 — Finance Bexio (CH) OAuth2 + Sync
- **Ziel:** CH-Buchhaltungs-Anbindung. **Deadline:** ≤01.09. **Prereq:** Welle 4. **FE:** 🟢 (Integrations-Karte da). **Aufwand:** mittel.
- **Backend-Scope:** OAuth2-Connect + bidirektionaler Sync-Service (Rechnungen/Kontakte) + Mapping-Settings. (bexio-Service-Gerüst existiert in `internal/biz/bexio`.)
- **Verifikation:** OAuth-Token-Refresh, Sync-Idempotenz, Mapping-Roundtrip.

### Welle 8 — kommunikation P1 (Chat-Reactions + Inbox-Extensions + Canned) — Reactions-Teil ✅ 2026-06-11
- **Ziel:** Gebautes Kommunikations-Modul scharfschalten. **Deadline:** P1 (nach Pilot). **Prereq:** keine. **Aufwand:** mittel-groß.
- **Backend-Scope:**
  - ✅ **Chat-Reactions — ERLEDIGT 2026-06-11 (`c9c19380`):** `internal/work/reaction` in ChatGRPCServer wiederverwendet (Contract-Entscheidung),
    Routen `POST/GET /api/v1/messages/{id}/reactions` + `POST /api/v1/messages/reactions/summary`, 501-Stubs aus route_video entfernt,
    OpenAPI + FE-Wiring + MSW. Entscheidung: KEIN `reactions`-Feld in `MessageInfo` (separate Calls, Batch-Kosten). Follow-up: `MessageBubble.tsx` auf `useReactions` migrieren.
  - **Inbox-Extensions (mock-first, wiring-ready):** `status`-Feld (offen/wartend/gelöst/geschlossen) + Filter + Set-Status-RPC 🟢;
    Threading (`ListThreadMessages`/Conversation-Modell) 🟡; Tags-CRUD (`AddTag`/`RemoveTag`) 🟢; `ForwardMessage` 🟢; SLA-Tracking 🔴.
  - **Canned Responses:** CRUD-Endpoints 🟢.
- **Contract-Entscheidung:** Reaction-Storage (chat vs. work/reaction wiederverwenden), Inbox-Status-Enum, Conversation/Thread-Modell.
- **Hinweis ✅ bereits fertig:** File-Upload (`/files/upload` mit `message_id`), `GetUserMentions`, `SearchChat`.

### Welle 9 — work P1 (Labels + Custom-Field-Defs + Settings-Anwendung) — Kern ✅ 2026-06-11
- **Ziel:** work-Modul-Settings scharfschalten. **Deadline:** P1. **Prereq:** Welle 1 (tenant_settings) empfohlen. **Aufwand:** mittel.
- **Backend-Scope:**
  - ✅ **Label-Taxonomie — ERLEDIGT 2026-06-11 (`2b8447b6`, Migr. 000145+000147):** `/api/v1/work/labels` CRUD + `PUT /tasks/{id}/labels` +
    `label_ids` im TaskProto + Permission-Seeds. Follow-ups: Batch-Load `label_ids` in Get/ListTasks; `filter_label_ids` SQL-JOIN im task-Repo.
  - ✅ **Custom-Field-Definitionen — ERLEDIGT 2026-06-11 (`2b8447b6`, Migr. 000146+000147):** `/api/v1/work/custom-fields` CRUD, tenant-scoped + RLS.
    FE-Adapter-Follow-up: `field_type`→`type`, `position`→`sortOrder`.
  - Offen: Default-Status-Set (tenant-Setting `default_project_statuses`) · Projekt-Vorlagen löschen · Zeit-Regeln (billable-Default/Stundensatz) ·
    Portfolio + Auslastungs-Aggregat (🟡) + optional `start_date` am Task (Gantt) + `due_from`/`due_to`-Filter.
- **FE:** überwiegend 🟢 (`stores/workSettings.*`, `taskLabels.ts`) — Wiring auf echte API = FE-Lane.

### Welle 10 — kontakte P1 (360°-Verknüpfungen + Finanzberatung)
- **Ziel:** CRM-360° + Finanzberatungs-Features. **Deadline:** P1 (Advisory pilot-nah). **Prereq:** Welle 4 (für Invoices-Sektion). **Aufwand:** mittel-groß.
- **Backend-Scope:**
  - Verträge am Kontakt (🔴 `contact_id`-Filter + `GET /api/v1/contracts?contact_id=`) · Rechnungen am Kontakt (🔴 `GET /api/v1/finance/invoices?contact_id=` — braucht Welle 4 Kontakt-FK).
  - **Beratungsprotokoll (🟢, Finanzberatung):** `advisory_protocols` (contact_id, ~40 Felder/8 Abschnitte, **immutable nach Aushändigung**,
    10-Jahre-Retention, DSGVO Art. 6(1)(c)) + CRUD + PDF-Export.
  - „Empfohlen von" + Mandanten-Segmente (Self-Ref + Empfehler-Report; Segment A/B/C regelbasiert) 🟢 · Leads als
    **Kontakt-Lifecycle-Status** (`contacts.lifecycle_stage`: lead→qualified→customer, `lead_source/score/temperature/status`,
    `GET /api/v1/leads`, Convert; Scoring serverseitig spiegeln; Dialer-Rückrufwunsch→Lead) 🟢 · XLSX-Import 🟡.
- **Contract-Entscheidung:** advisory_protocols-Felder/Abschnitte (40 Felder) + Immutability-Trigger; Lead-Scoring-Regel.

### Welle 11 — team P1 (Lohnvorbereitung / DATEV-HR)
- **Ziel:** Payroll-Vorbereitung + DATEV-HR-Export. **Deadline:** P1. **Prereq:** Welle 1 (tenant_settings für Berater/Mandanten-Nr.). **FE:** 🟢 (`PayrollPrepPanel`, mock-first). **Aufwand:** mittel-groß. **Spec:** `team-datev-lohn-spec.md`.
- **Backend-Scope:** `payroll_runs` (period, group, status locked/exported, exported_at, employee_count) + **DATEV-HR-Datei-Generierung**
  (LODAS / Lohn&Gehalt mit Lohnarten + Abwesenheitsschlüssel) + Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe.
  `tenant_settings` (module_id='team', `payroll.*`) für Berater-/Mandanten-Nr + Mappings. + Onboarding-Workflow-API (Template+Checklist, 🟡).

---

## 5. P2-Backlog (Post-Launch — NICHT pilot/P0)

Detail in Anhang A (Dariens Handover, P2) + `.planning/backend-gaps.md`. Kurzliste:
- **Produktivität:** vertraege (UploadDocument/Audit/Signatur-Workflow), dokumente (Kommentare/Share-Links), mails (Multi-Account/Vorlagen/Regeln),
  helpdesk (contact_id/source_channel/Inbox→Ticket/KB/time_spent), berichte (Query-Builder/Cross/Pivot), wiki (Share-Token/Templates),
  notifications (E-Mail+SMS-Kanal exponieren), automatisierung (Branch/Merge/http_request/webhook.received/Cron), video (Breakout/Recording-DL/Recurrence),
  zeiterfassung (Saldo/Export/project_id).
- **admin/settings:** Tenant-Provisioning + Onboarding · Super-Admin · Billing/License-Service · Workspace-Branding · Modul-Toggle exponieren.
- **finanzen-Komfort:** wiederkehrende Rechnungen · OP-Liste · mehrstufiges Mahnwesen · CAMT.053/MT940-Import · `currency`+Wechselkurs · BMD(AT)/Lexware.
- **security (post-launch):** SSO (SAML/OIDC), LDAP/AD, WebAuthn/Passkeys.
- **Branchen-Module (Solar-Pilot ab Nov):** fuhrpark/inventar/vermietung/einkauf/produktion/schichten/rapporte-Erweiterungen +
  Mobile/PWA-Zugangsweg + Einkauf↔Inventar-Sync (`einkauf.ReceiveGoods`→`inventar.RecordMovement`).

---

## 6. Empfohlene Chains (für „2–3 Wellen am Stück durchlaufen lassen")

- **Chain PILOT (höchste Dringlichkeit, ≤01.09):** Welle 0 → Welle 2 → Welle 3. *(klein+klein+mittel, alle unabhängig)*
- **Chain FUNDAMENT (max. Hebel):** Welle 1 allein laufen lassen (breit, entsperrt P1-Module).
- **Chain FINANCE-P0 (≤01.09, sequentiell wegen Prereq):** Welle 4 → Welle 5 → (Welle 6 ∥ Welle 7).
- **Chain FE-SCHARFSCHALTUNG (P1):** Welle 8 → Welle 9 → Welle 10. *(jede entsperrt ein gebautes FE-Modul)*

**Single-Wave-Start:** Jede Welle ist eigenständig startbar — Session öffnen, dieses Doc + die Welle-Quellen lesen, Plan-Mode-Ultrathink, ausführen.

---

## 7. Offene Contract-Entscheidungen (🔴 — VOR Bau mit Darien/Claude abstimmen)

Bei 🟢-Punkten genügt der Endpoint (FE zieht ohne Abstimmung nach). Bei diesen 🔴/Shape-kritischen Punkten vorher Feld-/Shape-Form klären:
- Settings-`value`-Typ + key-Namespace (Welle 1).
- Verfügbarkeitsregel- + Public-Booking-Payload-Shape (Welle 3).
- E-Rechnung Format-Enum + Response-Form (Welle 5).
- DATEV Kontenrahmen-Default + Steuerkennzeichen-Mapping (Welle 6).
- Chat-Reaction-Storage-Strategie + Inbox-Status-Enum + Thread/Conversation-Modell (Welle 8).
- work Label-/Custom-Field-Contract (Welle 9).
- advisory_protocols-Feldliste + Immutability + Lead-Scoring-Regel (Welle 10).

---

## Anhang A — Dariens Backend-Handover (Quelle, verbatim, Stand 2026-06-08)

> Konsolidierter, nach Launch-Impact priorisierter Plan dessen, was im Backend fehlt, damit das Frontend zu
> Feature-Parität andocken kann. Ersetzt als Lesefassung die organisch gewachsene `.planning/backend-gaps.md`
> (die bleibt als Detail-/Arbeitsnotiz bestehen). Arbeitsteilung: Claude = Frontend, Luke = Backend.

### Lese-Legende
- **Priorität = Launch-Impact:** P0 Launch-Blocker · P1 Feature-Parität · P2 Später/Post-Launch.
- **FE-Status:** 🟢 wiring-ready (Mock-Store → Hook) · 🟡 teilweise (Verkabelung/Feinschliff offen) · 🔴 FE wartet/Neubau.

### P0 — Launch-Blocker
**finanzen (Buchhaltung) — DE/CH gesetzlich + Steuerberater-Übergabe.** Cosmi macht die Vorkette (Angebot → Zahlungseingang)
und übergibt an DATEV/Bexio. Modul ~90 % gebaut. Echte Backend-Lücken:
- **E-Rechnung (Pflicht):** XRechnung-UBL + ZUGFeRD 2.x (EN-16931) Ausgang + Empfang/XML-Extraktion. Empfangspflicht DE seit 01.01.2025.
  Vorschlag: Generator-Service + `POST /api/v1/finance/invoices/{id}/erechnung` (Format-Param) · Eingangs-Parser. FE 🟢.
- **GoBD-Belegarchiv:** unveränderbar, Änderungshistorie, 8 Jahre Retention. Vorschlag: WORM-Storage + `document_events`-Audit + Retention. FE 🟢.
- **DATEV EXTF-Export (DE):** Buchungsstapel (EXTF ASCII/CSV, Windows-1252) + Belegbilder-ZIP. Vorschlag: Export-Job + Settings
  (Berater-Nr., Mandanten-Nr., Sachkonto-Länge, Steuerkennzeichen→Konto-Mapping). FE 🟢 (FinanceSettingsPanel).
- **Bexio-API (CH):** OAuth2, Rechnungen/Kontakte bidirektional. Vorschlag: OAuth2-Connect + Sync + Mapping-Settings. FE 🟢.

**kalender — Online-Terminbuchung (🔴 ZFA-Pilot-kritisch).** FE-Flow als Mock komplett, Backend fehlt ganz. Vorschlag:
`GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` · `GET /api/v1/public/book/:slug` (öffentlich) ·
`GET .../availability?date=&service=` · `POST /api/v1/public/bookings`. FE 🟢.

**dialer — DSGVO-Consent (⚠ Risiko).** `consentAsserter` im Standard-`NewService` `nil`; nur `NewServiceWithConsent` verdrahtet
den Check. To-do: prüfen ob Standard-Konstruktor aktiv → sonst Anrufe **ohne Consent**. Für Finanzberatung heikel.

**security — „Passwort vergessen".** Login hat keinen Reset-Link. Vorschlag: `POST /api/v1/auth/forgot-password` (Mail-Token) +
`POST /reset-password`. FE 🔴 (Login-Redesign wartet u. a. darauf).

### P1 — Feature-Parität
**kommunikation (Team-Chat + Posteingang).** Meiste gRPC-Endpoints existieren; im Demo-Mode (MSW) nachgebaut.
- **🔴 Chat-Reactions:** proto deklariert `ToggleReaction/ListReactions/GetReactionSummary`, Service implementiert nicht,
  keine Route, `MessageInfo` ohne `reactions`. Vorschlag: implementieren oder `internal/work/reaction` mitnutzen +
  `POST/GET /api/v1/messages/{id}/reactions` + `reactions` in `MessageInfo`. FE 🟢 (`stores/chatReactions.ts`).
- **Posteingang:** Inbox-Status (offen/wartend/gelöst/geschlossen) — `status`-Feld + Filter + Set-RPC (FE 🟢 `inboxStatus.ts`) ·
  Threading — `ListThreadMessages`/Conversation (FE 🟡 `inboxThread.ts`) · Tags-CRUD `AddTag`/`RemoveTag` (FE 🟢 `inboxTags.ts`) ·
  Forward — `ForwardMessage` (FE 🟢 `ForwardDialog`) · SLA-Tracking (FE 🔴).
- **Canned Responses:** CRUD (FE 🟢 `CannedResponseManager`). · **Channels-Connect** (E-Mail/WhatsApp/Widget): OAuth/Connect (FE 🟡;
  Routing-Rules-Infra im inbox-Service ist Basis). · **Interne Notizen + @Mention im Kunden-Thread** (FE 🟡). ·
  **Collision-Hinweis** Live-Presence-pro-Conversation (FE 🟡 `inbox-collision.ts`). · **Call-Bridge aus Posteingang**:
  `createCall`-Verkabelung + Kunde→user_id (FE 🟡). · **Per-Channel-Mute / eigener Status** (FE 🟢). ·
  **Slash-Commands + Webhooks + Bots:** Runtime + Webhook-CRUD/Delivery — Neubau (FE 🟡 Mock-Shell). ·
  **Gruppen-DMs, Pin/Lesezeichen, Channel-Notif-Settings:** proto fehlt — Neubau (FE 🔴).
- **✅ backend-fertig:** File-Upload (`POST /files/upload` mit `message_id`), `GetUserMentions`, `SearchChat`.

**work (Projekte/Aufgaben).**
- **🔴 Label-Taxonomie:** Tasks haben `tags` (string[]), keine strukturierten Labels (Name+Farbe, tenant). Vorschlag:
  `/api/v1/work/labels` CRUD + `label_ids` + Filter im `listTasks`. FE 🟢 (`workSettings.labels`, `taskLabels.ts`).
- **🔴 Custom-Field-Definitionen (Task):** Werte gibt es (`GET/PUT /tasks/{id}/custom-fields`), keine Definitions-Verwaltung.
  Vorschlag: `/api/v1/work/custom-fields` analog CRM. FE 🟢.
- Default-Status-Set (`default_project_statuses`, FE 🟢) · Projekt-Vorlagen löschen (FE 🟢) · Zeit-Regeln (billable/Stundensatz, FE 🟢) ·
  Portfolio + Auslastung/Budget (`start_date` am Task, Portfolio-/Auslastungs-Aggregat; FE 🟡) ·
  ✅ Kalender-Sicht nutzt `PUT /tasks/{id}` (due_date); latent `due_from`/`due_to`-Filter.

**kontakte (CRM) — 360° + Finanzberatung.**
- Verträge am Kontakt: `GET /api/v1/contracts?contact_id=` (FE 🔴) · Rechnungen am Kontakt: `GET /api/v1/finance/invoices?contact_id=`
  (finance_line_items Kontakt-FK = Sprint-4-Normalisierung Voraussetzung; FE 🔴) ·
  Beratungsprotokoll (P8): `advisory_protocols` (contact_id, ~40 Felder/8 Abschnitte, immutable nach Aushändigung, 10-J-Retention,
  DSGVO Art. 6(1)(c)) + CRUD + PDF (FE 🟢) · „Empfohlen von" + Segmente A/B/C (FE 🟢) ·
  Leads als Kontakt-Lifecycle-Status (`lifecycle_stage`: lead→qualified→customer; `lead_source/score/temperature/status`;
  `GET /api/v1/leads`; Convert; Scoring serverseitig; Dialer-Rückrufwunsch→Lead; FE 🟢) · XLSX-Import (FE 🟡).

**team — Lohnvorbereitung / DATEV-HR.**
- Lohnvorbereitung: `payroll_runs` (period, group, status locked/exported, exported_at, employee_count) + DATEV-HR-Generierung
  (LODAS / Lohn&Gehalt mit Lohnarten + Abwesenheitsschlüssel) bzw. Lohnimportdatenservice (DATEVconnect, Akkreditierung).
  Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe. `tenant_settings` (team, `payroll.*`). FE 🟢
  (`PayrollPrepPanel`, mock-first). Spec `team-datev-lohn-spec.md`. · Lohnauswertungsdatenservice (Phase 2) ·
  Onboarding-Workflow-API (FE 🟡) · **⚠ Demo-`userName`-Lücke (modulweit, Quick Win):** `/api/v1/hr/employees` Demo liefert leeren
  `userName` → „Unbekannt". Fixtures befüllen.

**Settings-Fundament (Scope-Hierarchie) — querliegend.** 3-Ebenen Tenant-Default → Modul-Leiter-Override → User-Override.
FE komplett (`ModuleSettingsShell`, `useIsModuleLead`), persistiert nur localStorage. `tenant_module_leads`-Tabelle +
`GET /api/v1/tenant/module-leads?user_id=`, `PUT/DELETE .../module-leads/{user_id}/{module_id}`. `tenant_settings` +
`user_settings` + Resolve + RBAC serverseitig. FE 🟢.

**Cross-cutting Quick Wins.** S3/MinIO-Upload-Service (profil/kontakte/chat/fuhrpark/inventar/rapporte/vermietung) ·
Signatur-Persistenz (vertraege/rapporte/vermietung; FE 🟢 SignatureCanvas) · User-Preferences-Persistenz
(`GET/PUT /users/preferences`; für Electron-Single-Device tolerierbar; FE 🟢).

### P2 — Später / Post-Launch
**Produktivität/System:** vertraege (UploadDocument/MinIO · `contract_events`-Audit · Signatur-Workflow Skribble/DocuSign) ·
dokumente (Datei-Kommentare · externe Share-Links) · mails (Multi-Account · Vorlagen/Quicktext · Regeln/Filter · post-launch Exchange/EWS, PGP/S-MIME) ·
helpdesk (`contact_id`/`org_id`/`source_channel` · Inbox→Ticket · KB-Endpoint · `time_spent`) ·
berichte (Query-Builder-Contract · `ExecuteKindCross` · Breakout/Pivot) · wiki (Share-Token-Routes · Templates) ·
notifications (E-Mail+SMS-Kanal exponieren) · automatisierung (Branch/Merge · `http_request` + `webhook.received` · Cron) ·
video (Breakout · Recording-DL/List · Recurrence) · zeiterfassung (Saldo · Export CSV/DATEV-Lohn · project_id/customer_id/service_code).

**admin/settings/finanzen-Brücke:** admin (Tenant-Provisioning `POST /api/v1/tenants` · Super-Admin · Billing/License `/api/v1/billing` ·
Ressourcen-Monitoring) · settings (Branding `/api/v1/tenant/branding` · Modul-Toggle exponieren) ·
finanzen-Komfort (wiederkehrende Rechnungen+Scheduler · OP-Liste · mehrstufiges Mahnwesen · CAMT.053/MT940-Import+Matching ·
`currency`+Wechselkurs · BMD(AT)+Lexware/lexoffice) · buchhaltung-Brücke (autom. Kontierung SKR03/04 · EÜR · Steuerberater-Rolle read-only) ·
security (SSO SAML/OIDC · LDAP/AD · WebAuthn/Passkeys).

**Branchen-Module (Welle 3, Solar-Pilot ab Nov):** fuhrpark (Führerscheinkontrolle · Fahrtenbuch+PDF · Pool-Buchung · Tankprotokoll · GPS-Webhook) ·
inventar (Chargen/Seriennummern · Inventur-Session · Kommissionierung) · vermietung (Zustandsprotokoll · `signature_url` · Buchungsportal · Tarif-Staffeln) ·
einkauf (SupplierRating · Rahmenverträge+Katalog · 2-stufige Freigabe · autom. Bestellvorschläge) ·
produktion (BOM · progress/work_steps/scrap · Maschinen-Register · Material-Verfügbarkeit · QualityCheck · Kalkulation) ·
schichten (`shift_swap_requests`+approve/reject · Availability+Qualifikationen · Auto-Planer · `is_minor`+JArbSchG) ·
rapporte (Signatur · Aufmaß · `weather` · Material/Leistung) · Mobile/Offline (PWA-Zugangsweg · `rapporte`/`schichten`-client auf offline-queue) ·
Einkauf↔Inventar-Sync (`einkauf.ReceiveGoods`→`inventar.RecordMovement`).

### Dariens empfohlene Reihenfolge
1. **P0 zuerst** — finanzen (E-Rechnung/GoBD/DATEV/Bexio), kalender-Terminbuchung (ZFA-Pilot), dialer-Consent, Passwort-vergessen.
2. **Quick Wins parallel** — Demo-`userName`-Fix, S3/MinIO-Upload, Settings-Scope-Tabellen (entsperren je mehrere Module).
3. **P1 modulweise** — pro Modul die 🟢-Punkte (nur Hook-Tausch im FE): kommunikation → work → kontakte → team.
4. **P2 nach Launch.**
> Bei 🟢 genügt der Endpoint (FE zieht nach). Bei 🔴 Contract-Form (Felder/Shape) vorher mit Darien/Claude abstimmen.
