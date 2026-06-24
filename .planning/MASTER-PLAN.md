# Cosmi 1.0 — Master-Plan

> **DER eine Ort zum Abarbeiten.** Löst `MASTER-TRACKER.md` ab (war Stand 19.06. + veraltet).
> **Ziel:** Cosmi **1.0** — alle Module + Funktionen fertig, vorzeigbar für Kunden/Interessenten. Kein ZFA-Pilot, keine Sprint-Deadlines mehr als Treiber.
> **Stand verifiziert:** 2026-06-23 gegen echte Commits + Code (nicht gegen alte Tracker-Stände).
> **Horizont NACH 1.0:** Website fertigstellen → Handy-App (PWA, holt GPS/Offline/Barcode der Branchen) → KI.

---

## 0 · Methodik, Legende & Arbeitsweise

### Wie dieser Plan zu nutzen ist
- Jede Phase = eine abhakbare Zeile. §6 gibt die **Reihenfolge** vor (Wellen). **Fortschritt lebt in den Haken hier, nicht im Session-Context** → nach jeder Welle neues Terminal.
- **Session-Zyklus:** Sagt Darien „mach an den Phasen weiter", folgt die Session `.planning/SESSION-RUNBOOK.md` (laden → planen → recherchieren → Rückfragen → bauen → Screenshot-QA → speichern). 2 Terminals (Main + Sub) parallel.
- Nach jeder Phase der **Build-+-Verify-Standard** (`.planning/nico-block/WORKFLOW.md`): bauen → i18n ×4 (`{var}`, ICU-Plural) → Demo-Handler nur wo Backend fehlt → gescopter Typecheck → Playwright-Screenshot-QA **(Bilder wirklich ansehen)** → iterieren bis grün → ein Commit + Push.
- **Review = finale Welle 6** (nicht rollierend): aufgeteilt im Team an der fast-fertigen Version — jeder klickt durch, Screenshots + Notizen, anpassen bis **abgenommen**.

### Legende
`✅` fertig · `🔨` läuft · `⬜` offen · `🔁` Stand verifizieren · `🔒` Backend fehlt real (→ Luke-Track §5, FE baut mock-first + swap-ready) · `🔌` Backend existiert → **echt anbinden** (Echt-Schaltung) · `📱` braucht Mobile/PWA → **Post-1.0** (Handy-App-Phase) · `🔬` Recherche/Konzept zuerst

### Arbeitsweise-Entscheidungen (Darien, 2026-06-23)
1. **Echt verbinden, nicht weiter mocken** (Darien-Regel): pro Phase prüfen, ob der Endpoint existiert. **Ja → direkt ans echte Backend hängen** (🔌, kein neuer Mock). **Nein → mock-first bauen + auf Lukes Backend-TODO** (`backend-gaps.md`/`BACKEND-PLAN.md`) **+ 🔌-„verdrahten"-Zeile hier im Plan** (🔒), damit das Nachverdrahten nach Lukes Bau nicht vergessen wird. → Echt-Schaltung ist Welle 1.
2. **Erst ALLES bauen + verkabeln, DANN reviewen.** Frontend + Backend + Verkabelung für alle Module fertigstellen, **Review als finaler Block am Ende** (Welle 6) an der (fast) fertigen Version. Begründung: Review ist händische, teure Zeit — die lohnt sich an einer fertigen Version, nicht rollierend zwischendurch (sonst reviewt man Module mehrfach, weil sich beim Verkabeln noch viel ändert).
3. **„1.0-bau-fertig" je Modul =** alle FE-Phasen + Demo-Tiefe + **echtes Backend verkabelt** (wo vorhanden; 🔒-Teile warten auf Lukes Track). **Review-Abnahme kommt danach gebündelt** (Welle 6).
4. **Mobile-Features** (GPS-Stempel, Offline-Rapporte, Barcode-Scan) = Post-1.0 (Handy-App-Phase). Im Plan mit 📱 markiert, nicht in 1.0 eingeplant.

### Gesamtstand (verifiziert 23.06., aktualisiert 24.06.)
- **~100–110 von ~220 FE-Phasen fertig (~47–50 %).** (24.06.: security S-1…S-5 gebaut + 3 Module echt-geschaltet.)
- **Welle 1 (Echt-Schaltung) stark vorangekommen:** 10 Module echt-verkabelt (kontakte/crm/work/finanzen/notifications/vertraege/work-Labels + dialer/dashboard/zeiterfassung). 3 mock-verdeckte Backend-Bugs gefunden+gefixt.
- **security/DSGVO** (P0-Launch-Blocker) von ⬜ → ✅ FE-review-reif.
- **13 Module FE-mock-fertig** (Review aber erst am Ende, Welle 6, nach Verkabelung): kontakte, calendar, zeiterfassung, dokumente, finanzen, work, team, dashboard, vertraege, helpdesk, automatisierung, profil, mails + kommunikation(Paket A) + berichte, wiki. Diese brauchen noch die **Echt-Verkabelung** (Welle 1), nicht sofort Review.
- **App-Shell** weitgehend fertig (Login-Redesign, Idle-Lock, Passwort-Reset-FE, Logo, Login-Animation, Modul-Einstellungen). Reste in §1.
- **Onboarding/Info-Center:** Bausteine existieren (OnboardingWizard, Tour-System, HelpWidget), aber halbfertig/ungenutzt — Ausbau in §1.2.

---

## 1 · App-Shell & Onboarding / Info-Center

### 1.1 App-Shell-Reste (Fundament steht)
- [x] Login-Redesign (Space-BG, CosmiLaunch, 2FA, „Angemeldet bleiben") — `LoginPage.tsx`
- [x] Cosmi-Logo/Branding — `config/branding.ts`
- [x] Login→App-Animation (Fall A + B) — `LaunchOverlay.tsx`
- [x] Modul-Einstellungen-Overlay (22 Panels registriert) — `module-settings-registry.tsx`
- [x] Idle-Lock (15 Min, Re-Auth) — `IdleLock.tsx`
- [ ] **AS-1** Idle-Lock-Timeout **konfigurierbar** aus Settings (aktuell hardcoded `15*60*1000`) + optional PIN-Entsperrung
- [ ] **AS-2** Passwort-Reset **echt anbinden** 🔌 — FE fertig (`auth-reset-client.ts`), `/auth/forgot-password`+`/auth/reset-password` Backend verifizieren + MSW→echt
- [ ] **AS-3** „Angemeldet bleiben"-UX-Politur + Re-Login-Flow Feinschliff

### 1.2 Onboarding & Info-Center (NEUER BLOCK — größeres Feature)
**Vision (Darien):** Erst-Erklärer für den Grundaufbau (wo ist was; die Basis-Module die jeder hat — Zeiterfassung/Meetings/Team …). Für arbeitsspezifische Module → **„Führungskurse" im Info-Center** (z.B. Buchhaltung komplett erklärt). Neben/bei jedem Kurs ein Knopf **„Geführte Einrichtung"**, der das Modul richtig aufsetzt (Design + Einstellungen), Schritt für Schritt.
**Bestehende Bausteine:** `OnboardingWizard.tsx` (6 Schritte, simpel) · `stores/tour.ts` (5 Touren general/dashboard/team/finance/kommunikation + `TourOverlay`) · `HelpWidget.tsx` (FAQ/Shortcuts/Kontakt, **nicht eingebunden**).
- [ ] **O-0** 🔬 Recherche + Konzept (Markt: Notion/Linear/Intercom/Pendo/Userflow-Academy; Onboarding-Patterns) → Konzept **passend zum Cosmi-Style** (Editorial/Premium, kein generisches Tour-Overlay-Slop). Mit Darien abstimmen.
- [ ] **O-1** Erst-Erklärer: Grundaufbau + Navigation + Basis-Module — `OnboardingWizard` zum echten First-Run-Erklärer ausbauen
- [ ] **O-2** Info-Center-Shell: eigener Bereich/Einstieg (Andockpunkt: Sidebar-Bottom neben Einstellungen oder Header), Cosmi-Style
- [ ] **O-3** Führungskurse pro arbeitsspezifischem Modul (Kurs-System; **Buchhaltung als Pilot-Kurs**, dann Skalierung)
- [ ] **O-4** „Geführte Einrichtung" je Modul: Setup-Wizard (Design + Modul-Einstellungen geführt) — dockt an `ModuleSettingsShell` an
- [ ] **O-5** Tour-System integrieren (5 Touren prominent machen + erweitern) + **HelpWidget einbinden**
- [ ] **O-6** Demo-Tiefe + i18n ×4 + **Review-Gate**

---

## 2 · Module nach Cluster

> Format je Modul: aktueller Stand · offene FE-Phasen · Demo-Tiefe · **Review-Gate**. `🔌`=echt anbinden, `🔒`=Backend fehlt.

### Cluster 1 — Vertrieb & Kommunikation

**kontakte** — ✅ P0–P8 komplett · ✅ **Echt-Schaltung: READ + voller CRUD live verifiziert = Referenz-Modul** (`aec2df49`, 23.06.)
- [x] **CRUD echt-geschaltet** 🔌 — Liste/Detail/Create/Update/Delete live gegen lokales crm-Backend (Screenshots `desktop/.qa-screenshots/crud-*.png`). Pattern `api/casing.ts` `dual()` + `mocks/demo-mode-flag.ts` mode-branch. Mock-verdeckte Bugs gefixt: PUT≠PATCH, position≠title, custom_fields-Array, admin-Seed. Bericht: `.planning/kontakte-mock-exit-DONE.md`
- [ ] 360°-Wiring Rest 🔌 `useContactContracts/useContactInvoices` (BE da, Migr. 141) · [ ] Timeline-Endpoint (CHRONIK) hängt → Luke · [ ] Beratungsprotokoll Server-PDF 🔒 (revisionssicher) · [ ] Tiefe-Re-Check T-1 · [ ] **Review-Gate**

**calendar** — ✅ P1–P3,P5(Booking) · **echt-verdrahtet (Luke) + API-verifiziert 24.06.**
- [x] **Echt-Schaltung 🔌** — `calendar-client` nutzt `authenticatedRequest`+`normalizeWireTimestamps` (alle TS), work-Service lokal hochgefahren, `/calendar/calendars` liefert Default-Kalender, `/calendar/events`→`{}` von allen Consumern `?? []`-geguardet (KalenderPage/MyCalendar/CalendarUpcoming). · [ ] P4 CalDAV-Sync echt 🔒 · [ ] Booking-Portal öffentliche Seite (ohne Auth) · [ ] Scoped-Delete (Serien: „diesen/alle") · [ ] Tiefe-Re-Check T-2 · [ ] **Review-Gate**

**notifications** — ✅ N-1…N-5 · MSW + DND-Routes echt
- [ ] Echt-Schaltung 🔌 + Bugfix: `priority`-Mismatch (FE `high` vs DB `urgent|normal|low`) + Spalten `is_pinned/is_dismissed/actor_name` nachziehen · [ ] P4 Real-Time WebSocket 🔒 · [ ] P5 Multi-Channel (E-Mail-Digest/Push) 🔒 · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

**kommunikation/chat** — ✅ KO-1…KO-10 (Paket A review-reif) · Team-Chat teils echt, Inbox MSW
- [ ] ci-desktop-Bug fixen (`ChatFlow.test.tsx` 7/12 rot) · [ ] Inbox echt 🔒 (Status/Threading/Tags/Forward/Canned/interne Notizen — alles FE-Overlay) · [ ] Chat-Reactions+Volltextsuche Backend 🔒 · [ ] P4 Audio/Video-Bridge 🔒 · [ ] **Review-Gate**

**mails** — ✅ P1 komplett (review-reif), vollständig MSW
- [ ] P2 IMAP/SMTP echtes Backend 🔒 (Multi-Account, Unified Inbox, Regeln/Filter, Templates) · [ ] P3 Exchange/EWS 🔒 (Post-MVP) · [ ] P4 PGP 🔒 (Post-MVP) · [ ] **Review-Gate**

**video/meetings** — ✅ überwiegend echt (LiveKit, In-Call-Chat, Host-Controls)
- [ ] P3 Recording/Consent/Bibliothek vollständig 🔒 · [ ] P4 Lobby/Waiting-Room + Moderation · [ ] P5 Breakout + Shared-Notes + ActionItems→Work · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

**dialer** — ✅ D-1…D-5 · LogCallOutcome echt, Supervisor MSW
- [x] Echt-Schaltung Supervisor-Dashboard 🔌 (live verifiziert, 2 BE-Bugs gefixt: recent-calls-SQL + protojson-Null-Normalizer, `13b0988c`) · [ ] P2 LiveKit-Recording 🔒 · [ ] P3 CTI/Click-to-Dial vollständig · [ ] P4 AMD/Predictive 🔒 (Post-MVP) · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

### Cluster 2 — Arbeit

**dokumente** — ✅ P1 + Block-Engine DB-1…DB-10 (review-reif) · MinIO-Upload echt · **READ echt-geschaltet (live verifiziert 24.06.)**
- [x] **Echt-Schaltung READ 🔌** (folders/files live gegen document-Service; 5 mock-verdeckte Drifts FE-tolerant gefixt in `document-client.ts` — List/Entity bare-array+bare-object, init-user-Body-400, `{seconds,nanos}`-TS, space_type-Int-Enum; Screenshots `desktop/.qa-screenshots/dokumente-mock-exit/`; kanonischer Gateway-Fix → Luke, `backend-gaps.md §dokumente`) · [ ] **Upload live** (blockiert auf `MINIO_PUBLIC_ENDPOINT`+CORS, Infra-Rollout) · [ ] Prod-Rollout MinIO-Public (DNS `s3.zentria.tech`, CORS-Origin, OnlyOffice-CSP) · [ ] P3 externe Share-Links (Ablauf/Passwort) 🔒 · [ ] P4 Governance (Papierkorb/Retention/Audit/DSGVO) 🔒 · [ ] Versions-Download-Bug · [ ] Tiefe-Re-Check T-3 · [ ] **Review-Gate**

**zeiterfassung** — ✅ P1–P5 (review-reif) · HR-Backend partiell echt
- [ ] Echt-Schaltung 🔌 (balance/entries/projects/analytics/team — Endpoints prüfen) · [ ] `project_id`/`customer_id`/`billable` auf `hr_work_time_entries` 🔒 · [ ] Dead-Code-Cleanup (`stores/timetracking.ts`, 10 tote Views) → T-4 · [ ] **Review-Gate**

**work (Projekte & Aufgaben)** — ✅ Tiefe-Pass + Quick-Actions + Portfolio + Labels + Calendar-View (review-reif)
- [ ] Echt-Schaltung 🔌 (Labels-Wiring BE Migr. 145/147 da) · [ ] P1 DnD-Backend + Timer→Zeiteintrag-Bridge 🔒 · [ ] P2 Portfolio-Aggregation echt 🔒 · [ ] `start_date`-Feld 🔒 · [ ] P3 Automatisierungs-Regeln · [ ] **Review-Gate**

**wiki** — ✅ W/WT/WP + Phase B Block-Engine (🔁 Darien-Review)
- [ ] Spezialblöcke in `wikiBlockRegistry` (Toggle/Code/Tabelle) · [ ] Share-Token-Routes + öffentl. Read 🔒 · [ ] P1 Backend-Swap 🔒 · [ ] P3 Public-Modus/RBAC 🔒 · [ ] P4 KI-Suche 🔒 · [ ] **Darien-Review → Review-Gate**

**formulare** — ✅ F/FD/FT/FO-Batches · MSW
- [ ] FO-Lücken 3/5/7/9 (Webhooks/Embedding/Analytics) verifizieren · [ ] Öffentlicher Submit-Endpoint 🔒 · [ ] File-Upload-Feldtyp 🔒 · [ ] Submission-Mail/Webhook 🔒 · [ ] P4 Zahlungen/E-Signatur 🔒 · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

**berichte** — ✅ R-0…R-6 + Charts/Export + E-0…E-5 Builder (🔁 Darien-Review)
- [ ] Server-PDF-Download 🔒 · [ ] Cron-Executor+Mailer 🔒 · [ ] Sparkline-Normalisierung + Linienfarbe-folgt-Badge (Feinschliff) · [ ] P4 DATEV-EXTF/externe BI 🔒 · [ ] **Darien-Review → Review-Gate**

**team** — ✅ TM-1…TM-5 (review-reif)
- [ ] Auth-User-Invite-Flow echt 🔒 · [ ] P2 Organigramm editierbar + Leave-Typen · [ ] P3 DATEV-Lohn (LODAS) 🔒 · [ ] P4 HR-Selfservice · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

**helpdesk** — ✅ Tickets/SLA/KB/Routing ans echte BE (review-reif)
- [ ] Aufräum-Welle-5 Reste (DeleteSLAPolicy, KB/Stats/Routing falls noch store) · [ ] `contact_id`/`source_channel` CRM-Verknüpfung 🔒 · [ ] P2 Mail→Ticket 🔒 · [ ] P3 Eskalations-Automatisierung · [ ] P4 Self-Service-Portal · [ ] **Review-Gate**

### Cluster 3 — Finanzen

**finanzen (= „Buchhaltung")** — ✅ P1–P2.5 + stark gewired (review-reif)
- [ ] P3 DATEV-EXTF + Bexio-OAuth + BMD-CSV 🔒 · [ ] P4 **E-Rechnung (ZUGFeRD/XRechnung) + GoBD-Archiv** 🔒 (FE offen, BE da `45a8ed61`) · [ ] P5 Banking (CAMT/MT940 + Matching) + EÜR 🔒 · [ ] Wiederkehrende Rechnungen + OP-Liste + mehrstufiges Mahnwesen + Fremdwährung 🔒 · [ ] buchhaltung-Dead-Code aufräumen · [ ] **Review-Gate**

**vertraege** — ✅ V-1…V-5 + Fixes (review-reif)
- [ ] openapi.yaml nachziehen 🔒 · [ ] P1 Backend-Persistenz + Audit-Log 🔒 · [ ] P3 E-Signatur echt (EES vs Skribble) 🔒 · [ ] P4 KI-Fristencheck · [ ] **Review-Gate**

### Cluster 4 — System & Konto

**settings** — ✅ Fundament (scope/ModuleSettingsShell/Leads)
- [ ] **FE localStorage → Settings-Backend** 🔌 (BE Migr. 138: tenant_settings/user_settings/tenant_module_leads) — Cross-Cutting, pro Modul · [ ] Modul-Leiter-CRUD-UI (Team-MemberDetailPanel) · [ ] P3 Workspace-Defaults real 🔒 · [ ] P4 Integrationen-OAuth 🔒 · [ ] Demo-Tiefe-Phase · [ ] **Review-Gate**

**dashboard** — ✅ D-1…D-5 + Cross-Modul-Alerts + Team-Dashboard (review-reif)
- [x] Layout-Persistenz echt verifiziert (apiClient↔gateway-nativ, Roundtrip GET→PUT→GET live) · [ ] KPI-Endpoint 🔒 (echte KPI-Werte + Zeitreihe statt FE-synthetisch) · [ ] P2 DnD-Resize/Reorder vollständig · [ ] **Review-Gate**

**profil** — ✅ P-1…P-5 (review-reif)
- [ ] Avatar-Upload echt 🔒 (S3-Service) · [ ] User-Preferences-Persistenz 🔒 · [ ] Presence-Routing 🔒 · [ ] **Review-Gate**

**security** — ✅ FE review-reif (S-1…S-5 mock-first, Branch `parallel/security`, 2026-06-24) — **DSGVO = wichtig**
- [x] FE-Tiefe verifiziert + alle 11 Seiten crashfrei (BE-konforme Daten-Contracts) · [x] DSGVO-Tools FE vollständig stateful mock-first (Audit Filter/Export/Verify, Sessions Terminate, Vault, PW-Policy, IP-Access + Export Art.15/20 approve/deny/download, Erasure Art.17 Preview/Execute+Legal-Hold, DSAR Art.15 Cross-Modul-Suche, Retention DACH+Auto-Löschung) · [x] Hub konsolidiert (`/admin/security`, 10 Sub-Tabs, Legacy-Redirect) + i18n ×4 (0 Raw-Keys DE+EN) + Modul-Settings-Eintrag · [ ] 🔌 Echt-Schaltung gegen Go-BE (`backend-gaps.md` „security/DSGVO": X-3-Spec-Lücke + Wire-Shapes + Timestamp-Normalizer) (BE teils da `47d210d9`/`60acb782`) · [ ] P4 WebAuthn 🔒 · [ ] 🔭 **Verarbeitungsverzeichnis Art. 30 RoPA** (eigener Batch — hoch-sichtbares DSGVO-Verkaufs-Feature, siehe `qa-security.md`) · [ ] **Review-Gate**

**admin** — ⬜ AdminHub-Tabs-Gerüst, überwiegend Stub
- [ ] 🔁 Tiefe der Tabs verifizieren · [ ] P1 Benutzerverwaltung (Liste/Einladen/Rolle/Deaktivieren) 🔒 · [ ] P2 RBAC-Matrix + Modul-Leiter verwaltbar · [ ] P3 Abo/Lizenz real 🔒 · [ ] P4 Branding persistent 🔒 · [ ] P5 Ressourcen-Monitoring 🔒 · [ ] Demo-Tiefe · [ ] **Review-Gate**

**automatisierung** — ✅ A-1…A-5 (review-reif) · MSW
- [ ] P1 Backend-CRUD echt 🔒 · [ ] P2 Flow-Editor Branching/Loop · [ ] P3 Webhook-Inbound + http_request-Action 🔒 · [ ] P4 Modul-übergreifend + Permissions · [ ] P5 Template-Marktplatz · [ ] Demo-Tiefe · [ ] **Review-Gate**

---

## 3 · Branchen-Module (Cluster 5)

> **Alle 7 sind P1 ans echte Backend gewired ✅.** Offen: Tiefe P2–P5 (Desktop). Mobile-Teile (📱) = Post-1.0/Handy-App.

**rapporte** — [ ] PDF-Export echt 🔒 · [ ] Aufmaß/Measurement 🔒 · [ ] Signatur-Persist 🔒 · [ ] Approval-Backend 🔒 · [ ] GPS-Tag 📱 · [ ] Offline 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**
**schichten** — [ ] Shift-Swap-Backend 🔒 · [ ] Monats-/Mehrwochenansicht · [ ] Auto-Planer (regelbasiert) 🔒 · [ ] Minderjährigen-Schutz (JArbSchG) · [ ] Self-Service-Portal 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**
**fuhrpark** — [ ] Buchungs-Pool · [ ] Fahrtenbuch (§7 EStG) 🔒 · [ ] Führerschein-OCR 🔒 · [ ] Tankbuch/Fuel 🔒 · [ ] GPS/Telematik 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**
**vermietung** — [ ] Konflikt-Check echt 🔒 · [ ] Preismodell + Rechnung + Kaution 🔒 · [ ] Online-Buchungsportal 🔒 · [ ] Übergabe-Doku (Foto/Unterschrift) 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**
**inventar** — [ ] Chargen/Seriennummern 🔒 · [ ] Inventur-Workflow 🔒 · [ ] Picklisten/Kommissionierung 🔒 · [ ] Einkauf-Verknüpfung 🔒 · [ ] Barcode/QR 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**
**einkauf** — [ ] Wareneingangs-Dialog (Teilmengen) · [ ] Bestellfreigabe-Workflow 🔒 · [ ] Konditionen/Preislisten 🔒 · [ ] Bestellvorschläge aus Inventar · [ ] Lieferanten-Bewertung · [ ] Demo-Tiefe · [ ] **Review-Gate**
**produktion** — [ ] BOM-Modell 🔒 · [ ] Maschinenbelegung-DnD 🔒 · [ ] MRP (Material-Verfügbarkeit) 🔒 · [ ] QualityChecks 🔒 · [ ] Kalkulation Soll/Ist 🔒 · [ ] Arbeitsschritt-Rückmeldung 📱 · [ ] Demo-Tiefe · [ ] **Review-Gate**

---

## 4 · Cross-Cutting / Architektur / Bugs

### Fundament-Bausteine (blockieren mehrere Module)
- [ ] **X-1** Generischer **S3/MinIO-Foto-Upload-Service** 🔒 → fuhrpark/inventar/rapporte/vermietung/chat/profil (überall aktuell Mock)
- [ ] **X-2** Generischer **Signatur-Persistenz-Service** 🔒 → rapporte/vermietung/vertraege
- [ ] **X-3** **OpenAPI-Spec nachholen** für 6 Module: formulare, dialer, inventar, vermietung, vertraege, mails (riskantester Wiring-Fall)
- [ ] **X-4** **FE-Settings localStorage → Backend** (ModuleSettingsShell-Persistenz, alle Module; BE Migr. 138 da)
- [ ] **X-5** **Demo-Daten-Seeds pro Modul** (Muster `backend/seeds/demo/notifications.sql`) — für ~20 verbleibende Module
- [ ] **X-6** **Echter Build (Modell A)** für Nico-Review: `RENDERER_VITE_DEMO_MODE=false` + `API_URL=https://app.zentria.tech`, Demo-User auf Prod seeden
- [ ] **X-7** **Modul-Feature-Flags in Prod setzen (Auto-Deploy-kritisch, 24.06.):** helpdesk/wiki/berichte/formulare/vertraege/video/Branchen sind Gateway-`modules.*`-Flags, default OFF → ohne `COSMI_MODULE_*_ENABLED=true` in `.env.production` sind Routen+Nav weg, egal wie fertig die FE. `.env.production` pro launch-fertigem Modul ergänzen. (Lokal: `deploy/docker/docker-compose.flags.yml`, untracked.) Detail `backend-gaps.md`.

### Bekannte Bugs / technische Schulden
- [ ] **B-1** ci-desktop rot: `ChatFlow.test.tsx` 7/12 (Chat-Rework) — verhindert grünes CI
- [ ] **B-2** notifications `priority`-Mismatch + fehlende Spalten (vor Echt-Schaltung)
- [ ] **B-3** Mojibake `de.json` (~90 latin1→utf8-Artefakte) · **B-4** OpenAPI-Drift Spec↔Endpoints · **B-5** HRSettings 3 Felder
- [ ] **B-6** Versions-Download-Bug (dokumente) · **B-7** `DELETE /projects/{id}` fehlt (work) · **B-8** OnlyOffice-CSP-Block (Prod)
- [ ] **B-9** Tiefe-Re-Checks T-1…T-4 (kontakte/calendar/dokumente/zeiterfassung)
- [ ] **B-10** Git-Hygiene: `desktop/package.json` (M), `backend/seeds/` (untracked), `purge-profil-zeiterfassung-i18n.mjs` — committen oder verwerfen
- [ ] **B-11** contacts-Befunde aus Echt-Schaltung (Details `.planning/kontakte-mock-exit-DONE.md`): (a) **Contact-Schema zu dünn** — 9 UI-Extra-Felder (mobile/address/jobTitle/department/…) ohne Backend-Pendant → `extras jsonb` ODER Spalten; (b) **Spec-Drift contacts** (Teil von X-3/B-4): Route ist PUT nicht PATCH, Feld `position` nicht `title`, `custom_fields` Array nicht Objekt; (c) **Timeline-Endpoint** `GET /crm/contacts/{id}/timeline` hängt

---

## 5 · Nur-Luke (Backend-Track, parallel)

> Reine Backend-Blöcke. FE baut mock-first + swap-ready; sobald BE steht → Echt-Schaltung (🔌). Detail: `backend-gaps.md`.

- **Mail-IMAP/SMTP** (Multi-Account, Regeln, Templates) · **LiveKit-Vollausbau** (Recording/Egress/Breakout für video+dialer)
- **DSGVO-Tools** (security: Audit/Sessions/Export/Erasure/DSAR/Retention) · **DATEV** (finanzen-EXTF + team-LODAS-Lohn)
- **E-Rechnung/GoBD-Archiv** (finanzen P4) · **Automatisierungs-Engine** (CRUD/Execution/Webhook-Inbound/http_request) · **MRP** (produktion)
- **Settings-scope-Tabellen** (da, Migr. 138 — FE anbinden) · **S3-Upload-Service** + **Signatur-Service** (X-1/X-2)
- **Branchen-Feature-Endpoints** (Aufmaß, Fahrtenbuch, Inventur, BOM, Shift-Swap, Konflikt-Check, Bestellfreigabe …)
- **Auth-Invite-Flow** (team) · **Tenant-Provisioning** (`POST /tenants`) · **Billing/License-Service** · **KPI-Endpoint** (dashboard)
- **Plattform:** Auto-Update (electron-updater) + Code-Signing · DB-Partitionierung · Migrations-Drift prod↔repo (209↔213+)

---

## 6 · Batch-Queue & Review-Pipeline (Abarbeitungs-Reihenfolge)

> Jede „Welle" unten ist **KEINE einzelne Sitzung** — sie bündelt viele Modul-Phasen. **~110–130 offene FE-Phasen** gesamt, abgearbeitet in **5er-Phasen-Batches** (etablierter Rhythmus).

**Arbeitsmodus — parallel (wichtig für die Zeitrechnung):**
- **Zwei Terminals** (Main + Sub, beide Claude) auf **disjunkten Modul-Lanes** → verdoppelt den Durchsatz. Ablauf: `.planning/two-terminal-nico-workflow.md` + `.planning/multi-stream-workflow.md`. Paket-Muster: `.planning/parallel-batch/` (`main-X.md` / `sub-Y.md` / `qa-combined.md`). **Main plant das Sub-Paket + gibt den Start-Text.**
- **Kollisionsregeln:** disjunkte Lanes wählen — keine Hot-File-Überschneidung (i18n-JSONs, `mocks/handlers/index.ts`, geteilte `shared/`-Komponenten, `openapi.yaml`). Siehe `.planning/collision-map.md`.
- **Innerhalb einer Lane:** max 3–4 Sub-Agenten (Worktree-Isolation), Pause-Gate, Main reconcilet + verifiziert + **Screenshots ansehen** + commit/push.
- **Zeitschätzung:** solo ~22–26 Bau-Runden · **mit 2 Terminals ~11–15** für die Bau-Wellen (1–5) — die finale **Review-Welle (6)** kommt danach (Team, händisch), Luke-Backend läuft zu allem parallel. Bau-Wellen ④ (Branchen) / ② (Lücken) parallelisieren am besten.
- **Reihenfolge-Prinzip (Darien 23.06.):** erst bauen + verkabeln (Wellen 1–5), **dann reviewen** (Welle 6) — an der fast-fertigen Version, nicht zwischendurch.

### Welle 1 — Echt-Schaltung & Fundament  ·  ~15–20 Pakete  ·  gut parallel
- [x] **kontakte echt-geschaltet (Referenz-Pattern etabliert, `aec2df49`, 23.06.)** — READ + voller CRUD live. `api/casing.ts` `dual()` + `mocks/demo-mode-flag.ts` mode-branch sind die Vorlage für jede weitere Echt-Schaltung. Casing-Entscheidung **Option C** (per-Modul, kein globaler Transform). Bericht/Risiko-Set: `.planning/kontakte-mock-exit-DONE.md`.
- [x] **crm komplett echt-geschaltet (23.06.):** companies (CRUD, `domain→website`-Drift), deals + pipeline-stages (DealInfo/PipelineStageInfo-Casing, Liste + Auswertungen live), contact-tags (entity_type). Alle PATCH→PUT-Drift gefixt. → das gesamte crm-Casing-Risiko-Set ist erledigt.
- [x] **work + biz Backend lokal hochgefahren + Finance-Seed (23.06.):** minio/createbucket/work/biz healthy (Gateway-Restart nötig für biz-Resolution); Demo-Seed um 3 Rechnungen + 2 Angebote ergänzt (Lines in `finance_invoice_lines`/`finance_quote_lines`, NICHT JSONB).
- [x] **work FE echt-geschaltet (23.06.):** `api/wire-time.ts` normalisiert `{seconds,nanos}`→ISO über alle work-Hooks (Projekte/My-Tasks/Detail/Kanban + Kommentare/Aktivitäten/Dateien/Zeit) → rendern live mit korrekten Daten.
- [x] **Buchhaltung (finanzen) FE weitgehend echt (23.06.):** Dashboard-KPIs (flat-select `data.X ?? data` + `conversion_rate` Number-Coerce), Rechnungen+Angebote-Listen mit Status (`finance-status.ts` Enum-Int→String aus biz.pb.go). Live verifiziert.
- [ ] **Buchhaltung Rest (B-12):** invoice-**List**-DTO liefert kein `gross_total` (nur Einzel-Rechnung hat's) → Betrag-Spalte 0,00 €. Backend: Totals in List-Response aufnehmen, ODER FE aus `line_items` summieren. Credit-Notes/Dunning (0/0 Demo) ungetestet.
- [x] **dialer-Supervisor** echt (2 BE-Bugs gefixt, live) · [x] **dashboard-Layout** echt verifiziert · [x] notifications + work-Labels + vertraege (Luke-Welle 24.06.). [ ] Offen 🔌: **zeiterfassung** (biz+minio laufen jetzt, Endpoints reachable, `entries:null`-Handling + HR-Seed prüfen), settings-Persistenz. + **X-3** OpenAPI-Specs + **X-4** Settings-Backend + **X-5** Demo-Seeds + **X-6** echter Build. + Bugs **B-1/B-2** (CI grün).
*Lanes:* je-Modul-Echtschaltung in 2 Terminals aufteilbar; **aber** cross-cutting-Bausteine (X-4 Settings-Backend, X-5 Seeds) = **Main-Lane, nicht doppeln** (Hot-Files).

### Welle 2 — Lücken-Module bauen  ·  ~20 Phasen  ·  gut parallel (3 disjunkte Lanes)
**security** (DSGVO, BE früh mit Luke) ∥ **admin** (Benutzer/RBAC/Lizenz) ∥ **settings**-Rest-Phasen → dazu die **Demo-Tiefe-Phasen** der schon gebauten Module (notifications/formulare/dialer/video + Tiefe-Re-Checks kontakte/calendar/dokumente/zeiterfassung).

### Welle 3 — Onboarding / Info-Center (§1.2)  ·  ~7 Phasen  ·  eher seriell
O-0 Recherche zuerst → O-1…O-6 (zusammenhängender Block). Läuft gut **als eigene Lane parallel zu Welle 2**.

### Welle 4 — Branchen-Tiefe (Desktop, §3)  ·  ~30–35 Phasen  ·  ideal parallel (7 Lanes)
P2–P5 der 7 Branchen-Module — disjunkte Module, perfekt für 2 Terminals. Mobile-Teile (📱) ausgelassen → Handy-App-Phase. Pilot-getrieben priorisierbar.

### Welle 5 — Finanzen-Tiefe + Markt-Parität-Reste  ·  ~20 Phasen  ·  teils parallel
finanzen P3–P5 (DATEV/E-Rechnung/Banking — eher seriell, ein Modul) · die 🔒-Vertiefungen quer (Markt-Parität) auf 2 Lanes verteilbar, sobald Luke-Backend steht.

### Welle 6 — FINALE REVIEW  ·  alle Module bau-fertig → händisch abnehmen  ·  Team, aufgeteilt
**Erst jetzt**, wenn FE + Backend + Verkabelung stehen: jedes Modul aufgeteilt im Team durchklicken, Screenshots + Notizen, anpassen bis **abgenommen** — an der (fast) fertigen Version. **Voraussetzung:** Lukes Backend-Track ist für die 🔒-Module verkabelt (sonst reviewt man noch Mock-Stände).

**Bau-Status heute** (was am Ende in die Review-Welle einfließt):
| Bau-Status | Module |
|---|---|
| ✅ **echt-verkabelt** (Welle 1) | **kontakte** (Referenz) · crm (companies/deals/pipeline/tags) · work · finanzen/Buchhaltung · **dialer-Supervisor** · **dashboard-Layout** · **zeiterfassung/HR** · notifications · vertraege · work-Labels |
| ✅ **FE-mock-fertig** — noch echt-verkabeln (Welle 1) | calendar, dokumente, team, helpdesk, automatisierung, profil, mails, kommunikation, berichte, wiki, **security/DSGVO** (S-1…S-5) |
| ⬜ **noch bauen** (Wellen 2–5) | formulare/video (Demo-Tiefe) · admin, settings (Lücken) · Onboarding/Info-Center · Branchen ×7 (Tiefe) · finanzen P3–5 |

**Review-Abnahme (Welle 6):** aufgeteilt — jeder klickt das Modul durch, macht Screenshots + Notizen, passt an (tote Buttons, Detail-Views, leere Zustände, Raw-Keys, Umlaute, Style), bis es **abgenommen** ist. Haken in §2/§3.

---

## 7 · Horizont nach Cosmi 1.0
1. **Website** fertigstellen (zentria.tech — Content-Audit, echtes Delivery-Mapping)
2. **Handy-App** (PWA auf Desktop-Basis) — holt die 📱-Features: GPS-Stempel, Offline-Rapporte, Barcode-Scan, mobile Self-Service-Portale
3. **KI** — KI-Features (Viz-Empfehlung, Artikel-aus-Ticket, Fristencheck, Assistent …)
4. **Weiter offen:** Orbit-Appliance (Self-Hosted, ADR-008) · WASM-Plugin-System · Pricing/Billing-Service · SSO/SAML
