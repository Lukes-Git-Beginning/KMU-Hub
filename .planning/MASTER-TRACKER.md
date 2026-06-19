# Master-Tracker — Kompletter Phasenplan bis Modul-Reviews

> **Der eine Ort zum Abarbeiten.** Jede Phase = eine abhakbare Zeile. Detail-Beschreibungen je Phase: `.planning/module-phase-plans.md` + `.planning/reviews/<modul>.md`.
> **Rollen:** Luke = Backend · Nico = Reviews fertiger Module · wir = Phasen abarbeiten.
> **Stand:** 2026-06-19 (work Quick-Actions done; berichte-Erstellen-Builder E-1…E-5 als Phasen eingetragen).

## Legende & Zählung
✅ fertig · 🔨 läuft · ⬜ offen · 🔒 FE bau ich mock-first, echtes Backend kommt von Luke · 🔁 Stand verifizieren
- **Eine Gesamtzahl** (Entscheidung Darien): 🔒-Phasen sind als FE-mock-first mitgezählt. Reine Backend-only-Arbeit steht separat unten unter „Nur-Luke".
- **Demo-Tiefe** ([[feedback_module_depth_standard]]): eigene Phase pro **bereits gebautem** Modul (Audit tote Buttons/Detail-Views + Fix). Bei **neu gebauten** Modulen ist Tiefe direkt eingebaut (keine Extra-Phase). finanzen P2.5 = Referenz.

## Gesamtstand
- **~155 offene Phasen** über ~32 Module (grobe, eher konservative Zahl — Verifizieren der Marathon-Module kann sie senken).
- **Fertig (bei Nico in Review):** kontakte (Kern), zeiterfassung; calendar/dokumente weitgehend (🔁).
- **Aktive Spur:** finanzen komplett fertigstellen → an Nico → nächstes Modul.
- **▶ ERLEDIGT (2026-06-17 Abend):** **Parallel-Batch dashboard + vertraege** (Main-Terminal=dashboard, Sub-Terminal=vertraege, beide Claude). dashboard **D-1…D-5** + vertraege **V-1…V-5** + **Darien-Live-Fix-Runde F1-F7** (E-Signatur RAUS aus vertraege-Detail [Modul=Verwaltung], ContractDialog→shared DetailModal, Detail schließt beim Bearbeiten, Dokumente prominent, Notification-Karten aufklappbar mit Öffnen/Anpinnen/Ignorieren, roter Drag-Placeholder→Cosmi-Türkis, user-select beim Drag) — alle gepusht (main=`f4a6844d`), je Build(echter Exit)+Playwright+Screenshot verifiziert. **dashboard + vertraege = review-reif → Nico** (Dariens finale Live-Abnahme stand beim Session-Ende noch aus). Review-Liste: `.planning/parallel-batch/qa-combined.md`; Detail: MEMORY.md Wiedereinstieg.
- **▶ ERLEDIGT (2026-06-19 nachts):** **work Quick-Actions W-1…W-3** (main `a046b0d9`, Squash, wip-Branch weg): MyTasks Geschwister-Button-Layout (kein `role=button`-Zeile mehr → Menü öffnet statt zu navigieren) + Quick-Complete + Aktions-Menü + Löschen; Kanban-Karten dieselben Quick-Actions (dnd-sicher via `data-card-control`); Task-Detail-Panel+-Page Löschen-Confirm. 14 i18n ×4. QA `qa-work-mytasks.mjs`+`qa-work-kanban.mjs` grün + Screenshots. **tsc-Lehre:** `tsconfig.workqa.json` crasht flaky (interner TS-Bug, nicht dateispezifisch) → Gate = Vite+Playwright-QA.
- **▶ NÄCHSTES (neuer Batch, 2 Terminals):** Kandidaten: **berichte-Erstellen-Builder E-1…E-5** (siehe berichte-Block unten — FE-mock-first, E-1-Kern = Feld-Picker+Viz+Live-Vorschau) · **formulare** (P1 DnD+DSGVO) · **wiki** (0 i18n-Keys + TipTap). Darien entscheidet die Paarung im neuen MAIN-Terminal. **Reihenfolge:** neues MAIN-Terminal plant (Ist-Abgleich → Klärungsfragen → Sub-Paket+Start-Text), DANN Sub-Terminal wechseln + Text geben. Paket-Muster: `.planning/parallel-batch/`. Zwei-Terminal-Ablauf: `.planning/two-terminal-nico-workflow.md`. **Build-Gate immer mit echtem Exit** ([[feedback_build_check_real_exit]], NIE `| tail`).

## Neue projektweite Standards (2026-06-16) — bei JEDEM Modul anwenden
- **Detail = zentriertes Cosmi-Modal-Fenster** (`shared/DetailModal`), NICHT Slide-over. [[feedback_detail_modal_standard]]
- **GANZE Zeile klickbar** (`div role=button` + stopPropagation auf inneren Buttons), nicht nur Titel/3-Punkte.
- **Zurück/Close sticky/immer sichtbar**, nie wegscrollen. [[feedback_sticky_back_buttons]]
- Im Fenster ALLE Infos + Funktionen.

## Arbeitsweise: rollierende Modul-Fertigstellung (für Nicos Review-Pipeline)
[[feedback_rolling_module_completion]] — **Wir bringen Module einzeln komplett zu Ende, nicht phasen-schichtweise.** Sobald ein Modul **review-reif** ist, geht es an Nico, und wir starten das nächste → Nico reviewt **parallel**, die Pipeline läuft nie leer.
- **„Review-reif" = FE-Phasen ✅ + Demo-Tiefe ✅.** 🔒-Backend-Teile dürfen gemockt/swap-ready sein (Nico reviewt FE/UX, nicht Lukes Backend).
- **Reihenfolge mischt schnell + langsam:** FE-reiche Module (nur Tiefe-Pass nötig) zwischen größere Neubauten einstreuen → stetiger Nachschub. Siehe „Review-Pipeline" unten.

---

## Tiefe-Re-Check der „fertigen" Module — ⏸ VERTAGT (Darien 2026-06-16: „später")
Schneller Audit gegen die Tiefe-Vorgabe; die 4 Module sind solange bei Nico in Review (as-is). Wir holen den Re-Check später nach.
- [ ] **T-1** kontakte — Tiefe-Re-Check (Detail-Views/Downloads/Exporte)
- [ ] **T-2** calendar — Tiefe-Re-Check + realen Phasen-Stand verifizieren 🔁
- [ ] **T-3** dokumente — Tiefe-Re-Check + realen Phasen-Stand verifizieren 🔁
- [ ] **T-4** zeiterfassung — Tiefe-Re-Check + Dead-Code-Cleanup

---

## Cluster 1 — Vertrieb & Kommunikation

### kontakte — ✅ Kern (P0–P7)
- [x] P0–P7 (Tab-Gerüst, Detail-Tiefe, Pipeline, Leads, Aktivitäten, Auswertungen, Einstellungen)
- [ ] **P8** Finanzberatungs-Tiefe (Empfohlen-von, Beratungsprotokoll, Segmente)
- [ ] Tiefe-Re-Check → siehe T-1

### calendar — 🔁 weitgehend (Marathon)
- [x] P1 Views · P2 Events-CRUD/RRULE/DnD *(verifizieren)*
- [ ] **P3** Mehrere/geteilte Kalender + Einladungen/RSVP
- [ ] **P4** CalDAV-Sync + Ressourcen 🔒
- [ ] **P5** Buchungs-Link (Calendly-Style) + CRM-Integration 🔒
- [ ] Tiefe → T-2

### notifications — 🔨 (Nico, tw.)
- [x] P1 Notification-Center · P2 Quiet-Hours *(verifizieren)*
- [ ] **P3** Modul-Gruppierung + Sidebar-Badges + OS-Notifications
- [ ] **P4** Real-time (WebSocket) + Toasts + Sound 🔒
- [ ] **P5** Multi-Channel (E-Mail-Digest, SMS, PWA-Push) 🔒
- [ ] Demo-Tiefe-Phase

### kommunikation/chat — ⬜ (3-Panel da)
- [ ] **P2** DMs + Volltextsuche + Unread 🔒
- [ ] **P3** Unified Inbox + Edit/Löschen/Lesezeichen
- [ ] **P4** Audio/Video aus Chat (LiveKit-Bridge) 🔒
- [ ] **P5** Bots/Webhooks/Slash-Commands 🔒
- [ ] Demo-Tiefe-Phase

### mails — ⬜ Neubau 🔒
- [ ] **P1** Mail-UI-Grundgerüst (3-Panel, Compose, Signatur)
- [ ] **P2** Konten + Unified Inbox + Suche 🔒
- [ ] **P3** Vorlagen + Regeln/Filter + Labels
- [ ] **P4** CRM-Integration (Thread↔Kontakt, Deal-aus-Mail)
- [ ] **P5** Exchange/EWS + PGP 🔒
- *(Tiefe direkt eingebaut)*

### video/meetings — ⬜ (Mock-UI da) 🔒
- [ ] **P2** LiveKit-Anbindung (Tiles, Device-Check, Screenshare) 🔒
- [ ] **P3** Recording + Consent + Bibliothek 🔒
- [ ] **P4** Lobby/Waiting-Room + Moderation
- [ ] **P5** Breakout + Shared-Notes + ActionItems→Work
- [ ] Demo-Tiefe-Phase

### dialer — 🔨 FE-P1 ✅
- [ ] **P2** LiveKit + Recording (Consent) 🔒
- [ ] **P3** CTI/CRM-Verknüpfung (Click-to-Dial, Auto-Aktivität)
- [ ] **P4** AMD + Auto/Predictive-Dialer 🔒
- [ ] **P5** Supervisor-Dashboard + Reporting
- [ ] Demo-Tiefe-Phase

---

## Cluster 2 — Arbeit

### dokumente — 🔁 weitgehend (Marathon)
- [x] P1 Move/Copy/Rechte/Kommentare · P2 Volltext/Tags *(verifizieren)*
- [ ] **P3** Kollaboration (Co-Edit-Status, externe Share-Links m. Ablauf/Passwort) 🔒
- [ ] **P4** Governance (Papierkorb/Retention, Audit-Log, DSGVO-Export) 🔒
- [ ] Tiefe → T-3

### zeiterfassung — ✅ P1–P5
- [x] P1–P5 (Standalone, ArbZG-Report, Team, DATEV)
- [ ] Tiefe + Dead-Code-Cleanup → T-4

### Projekte & Aufgaben (Code: `work`) — ✅ Tiefe-Pass fertig → Nico (Plan: `.planning/work-tiefe-pass.md`)
- [x] **W-Tiefe-Pass** (Darien Option 3): Slide-over→DetailModal · Karten/Zeilen klickbar · MyTasks tote Buttons (Move-to-Project wirkt) · Stunden→echte Draft-Rechnung (erscheint in finanzen) · Auslastung/Gast→MSW — alle 5 Phasen verifiziert (Screenshots), gepusht `999260ea`. **review-reif → an Nico.** Offen für Nico-Review: Guest-Ansicht rendert noch in der App-Shell (standalone Share-Link = P4).
- [ ] **P1** Daily-Use finalisieren (DnD backend, Timer→Zeiteintrag-Bridge, Inline-Edit) 🔒
- [ ] **P2** Portfolio + Auslastungs-View 🔒
- [ ] **P3** Automatisierungs-Regeln (wenn/dann)
- [ ] **P4** Externer Zugang (GuestView, Share-Links, Zeit→Finanzen-Export)
- [ ] Demo-Tiefe-Phase

### wiki — 🔨 (Nico, tw.)
- [ ] **P1** Backend-Swap (TanStack Query) 🔒
- [ ] **P2** Editor-Vollständigkeit (Anhänge, Tabellen, @Mention, [[Verlinkung]])
- [ ] **P3** Zugang/Freigabe (Share-Links, Public-Modus, Kategorie-RBAC)
- [ ] **P4** Suche + KI (Volltext, Artikel-aus-Ticket) 🔒
- [ ] Demo-Tiefe-Phase

### formulare — 🔨 (Strom N, tw.)
- [ ] **P1** DnD-Reordering + DSGVO-Feld + E-Mail-Benachrichtigung
- [ ] **P2** Webhook-Config + Automatisierung + CRM-Integration 🔒
- [ ] **P3** Embedding (iFrame/JS) + Analytics
- [ ] **P4** Zahlungen (Stripe) + E-Signatur 🔒
- [ ] Demo-Tiefe-Phase

### berichte — 🔨 Charts/Schedules/Export ✅ (B-1…B-5 + F1/F2) · Erstellen-Builder (E-1…E-5) offen
- [x] **P1 + P3 (B-1…B-5 + Live-Fixes, main `a046b0d9`):** echte recharts-Charts + Zeitraum-Filter + verschiebbare Kacheln · Export (PDF/CSV/XLSX als MSW-Blob) + geplante Berichte (stateful, Nächster-Lauf/Cron) + Alert-Schwellwerte · Schedule-Zeilen→DetailModal+Lauf-Historie · KPI-Sparklines mit Hover-Tooltip · SortMenu + Modul-Settings + 38 i18n-Keys ×4.
- [ ] **P4** DATEV-EXTF-Detail + externe BI 🔒

**▶ Erstellen-Builder** (= alte „P2 No-Code Query-Builder"; **FE-mock-first**, echter Query-Executor = Luke 🔒). Ist heute flach (`ReportBuilder.tsx`: System-Bericht-Dropdown + Zeitraum + Format/Export). Marktanalyse (Metabase/HubSpot/Looker Studio) → 5 Phasen. Detail-Roadmap + Bug-Kontext: `.planning/HANDOFF-berichte-work.md`.
- [ ] **E-1** Modus-Switch „System-Bericht ↔ Eigener Bericht" · Datenquelle (Modul) wählen → **Feld-Picker** (Checkbox-Liste) → **Viz-Typ-Picker** (Tabelle/Balken/Linie/Fläche/Donut/KPI) → **sofortige recharts-Live-Vorschau** + Zeitraum-Selektor. MSW: Feld-Metadaten + Demo-Series je Modul. **Datenquellen-Start (Vorschlag, beim E-1-Start mit Darien bestätigen): finanzen + work tief**, kontakte/deals danach.
- [ ] **E-2** Filter-Builder: typ-aware (Feld→Operator is/contains/>/</between→Wert), bis 5 Filter mit AND/OR, Filter-Chips, Vorschau reagiert.
- [ ] **E-3** Aggregation + Grouping: Dimension vs. Measure, Aggregation (Count/Sum/Avg/Min/Max), Group-by bis 2 Dimensionen, Pivot-light. `query_config` ausdehnen, MSW liefert passende series/totals.
- [ ] **E-4** Speichern + Bibliothek + Dashboard-Pin: „Meine Berichte"-Liste (benennen, Modul-Badge, zuletzt bearbeitet) · öffnen stellt Builder-State wieder her · „Zu Dashboard hinzufügen". (🔒 echter Persist-Endpoint = Luke; FE-State reicht für Demo.)
- [ ] **E-5** Advanced (Post-MVP): berechnete Felder (FE-Formel) · Scheduled Export 🔒 · Sharing 🔒 · KI-Viz-Empfehlung (heuristisch FE).
- Viz-Kernset (recharts-baubar): Tabelle · Balken · Linie · Fläche · Donut · KPI-Zahl · Combo · Gauge.
- [ ] **Demo-Tiefe-Phase** (nach E-1…E-4: Zeilen-Klick→Detail, echte Downloads, leere Zustände)

### team — ✅ Tiefe-Pass review-reif → Nico (Main-Terminal TM-1…TM-5, 7 Commits → `8a49415c`)
- [x] **Tiefe-Pass:** Abwesenheiten-Bug (entries+camelCase+Date-Filter) · SelfService verkabelt (useSelfProfile/Balance/Requests + Create-Flow + Blob-Download) · PersonnelDocuments→MSW (echte MA, Download/Preview-Dialog/Upload) · OrgChart E-Mail/Anruf wirken + 8× `{{}}`→`{}` + page.title · Deactivate komplett wirksam (Action-Item+PUT+status+Inaktiv-Render) · Umlaut-Cleanup. Alle Build-grün + Screenshot-verifiziert. QA: `.planning/parallel-batch/qa-team.md`.
- [ ] **P1** Trainings Backend-Swap (🔒 Luke — Zustand-Store funktional+swap-ready, im FE-Review nicht nötig) + Pflichtschulung-Tracking
- [ ] **P2** Personalakte↔Dokumente (Verknüpfung tiefer) + Organigramm editierbar + Leave-Typen
- [ ] **P3** DATEV-Lohnschnittstelle 🔒
- [ ] **P4** HR-Selfservice + Minderjährigen-Regeln + Vertrags-Vorlagen
- [ ] Demo-Tiefe-Phase

### helpdesk — ⬜ (auf Zustand)
- [ ] **P1** Backend-Swap + CRM-Kontakt-Lookup 🔒
- [ ] **P2** Multi-Channel (Mail→Ticket) + Merge + Ticket-Zeiterfassung 🔒
- [ ] **P3** Automatisierung (Eskalation, Auto-Close)
- [ ] **P4** Self-Service-Portal + öffentliche KB
- [ ] Demo-Tiefe-Phase

---

## Cluster 3 — Finanzen

### finanzen (= „Buchhaltung") — 🔨 aktiv
- [x] P1 Faktura-Kette · P2 Ausgaben/Kontierung · **P2.5a** Angebots-/Gutschrift-Detail · **P2.5b** Ledger-Detail + OP/Dashboard klickbar · **P2.5c** echte PDF-Vorschau + PDF/CSV-Downloads
- [x] **P2.5d** Mahnwesen verkabeln + Mahn-Detail + Zahlung/Settings speichern 🔒
- [x] **P2.5e** Hardcoded-Mocks→MSW (Banking, Belegkette, Stunden→Rechnung, Audit-Log) + Recurring-Liste-Fix
- [ ] **P3** DATEV-EXTF + Bexio-OAuth + BMD-CSV 🔒 ◀ als Nächstes
- [ ] **P4** E-Rechnung (ZUGFeRD/XRechnung) + GoBD-Belegarchiv 🔒 *(Launch-Blocker)*
- [ ] **P5** Banking (CAMT/MT940 + Matching) + EÜR-Auswertung + Settings-Feinschliff 🔒

### buchhaltung (dead) — ⬜
- [ ] **P1** Aufräumen (Test→finanzen, Ordner entfernen, i18n bereinigen) — *Rename finanzen→buchhaltung geparkt, mit Luke klären*

### vertraege — ⬜ (FE-Only auf Zustand)
- [ ] **P1** Backend-Anbindung + Audit-Log + Erinnerungs-Notifications 🔒
- [ ] **P2** Dokumente echt (Upload, Versionen, PDF-Viewer)
- [ ] **P3** E-Signatur echt (EES vs. Skribble) 🔒
- [ ] **P4** CRM/Finanzen-Verknüpfung + KI-Fristencheck
- [ ] Demo-Tiefe-Phase

---

## Cluster 4 — System & Konto

### settings — 🔨 (Fundament ✅)
- [x] P1 Settings-Fundament (scope, ModuleSettingsShell, Modul-Leiter)
- [ ] **P2** Modul-Settings-Tabs für fehlende Module
- [ ] **P3** Workspace-Defaults real (Firmendaten, Währung, Zeitzone, GJ-Start) 🔒
- [ ] **P4** Integrationen echt (Bexio/Lexware/DATEV OAuth) 🔒
- [ ] **P5** Notification-Settings + KI-Governance
- [ ] Demo-Tiefe-Phase

### dashboard — ⬜ (vollständig, Persistenz Mock)
- [ ] **P1** Backend-Persistenz + Widget-Konfiguration 🔒
- [ ] **P2** DnD-Resize/Reorder
- [ ] **P3** KPI-Widgets modul-/lizenzabhängig
- [ ] **P4** Modul-übergreifende Übersicht + echte Alerts 🔒
- [ ] **P5** Team-Dashboard + Rollen-Templates + Presence
- [ ] Demo-Tiefe-Phase

### profil — ⬜
- [ ] **P1** Avatar-Upload echt + Presence-Status 🔒
- [ ] **P2** Benachrichtigungs-Präferenzen im Profil
- [ ] **P3** Shortcuts zu Appearance/Security + Account-Info
- [ ] **P4** Profil-Karte (Overlay anywhere, Ping→Chat)
- [ ] Demo-Tiefe-Phase

### security — ⬜ 🔒 **DSGVO = P0-Launch-Blocker**
- [ ] **P1** Audit-Log + Sessions echt (Export unveränderlich) 🔒
- [ ] **P2** Passwort-Policy + IP-Zugriff enforced 🔒
- [ ] **P3** DSGVO-Tools real (Export Art.15/20, Erasure, DSAR, Retention) 🔒 *(P0)*
- [ ] **P4** MFA-Erweiterung (WebAuthn) + Vault 🔒
- [ ] **P5** SSO/SAML/OIDC (Post-MVP) 🔒
- [ ] Demo-Tiefe-Phase

### admin — ⬜ 🔒
- [ ] **P1** Benutzerverwaltung (Liste, Einladen, Rolle, deaktivieren) 🔒
- [ ] **P2** Rollenmodell/RBAC-Matrix + Modul-Leiter verwaltbar
- [ ] **P3** Abo/Lizenz real + Modul-Aktivierung tenant-weit 🔒
- [ ] **P4** Branding persistent
- [ ] **P5** Ressourcen-Monitoring + Tenant-Verwaltung (Option-B)
- [ ] Demo-Tiefe-Phase

### automatisierung — ⬜ 🔒 (Engine = großer Block)
- [ ] **P1** Wizard→echtes Backend (CRUD, Enable/Disable, Execution-Log) 🔒
- [ ] **P2** Flow-Editor mit Branching/Loop
- [ ] **P3** Webhook-Inbound + HTTP-Action 🔒
- [ ] **P4** Modul-übergreifende Workflows + Permissions
- [ ] **P5** Template-Marktplatz + KI-Assistent
- [ ] Demo-Tiefe-Phase

---

## Cluster 5 — Branchen (pilot-getrieben; ⚠ = PWA-Bedarf)

### rapporte ⚠ — ⬜
- [ ] **P1** Backend + Foto-Upload 🔒 · **P2** PDF-Export echt 🔒 · **P3** GPS/Standort · **P4** Offline (SW + IndexedDB-Queue) · **P5** Approval-Backend + Notifications 🔒 · **P6** Solar-Vertiefung
- [ ] Demo-Tiefe-Phase

### schichten ⚠ — ⬜
- [ ] **P1** Backend + Mitarbeiter aus Team 🔒 · **P2** Self-Service-Portal (mobil) · **P3** Monats-/Mehrwochenansicht · **P4** Auto-Planer 🔒 · **P5** Minderjährigen-Schutz + AT/CH-Feiertage · **P6** Reporting + DATEV 🔒
- [ ] Demo-Tiefe-Phase

### fuhrpark ⚠ — ⬜
- [ ] **P1** Backend + TÜV/Versicherung-Reminder 🔒 · **P2** Fahrzeugbuchungs-Pool · **P3** Fahrtenbuch live (§7 EStG) · **P4** Führerscheinkontrolle (OCR) 🔒 · **P5** GPS-Echtverfolgung 🔒 · **P6** TCO + Tankkartenimport
- [ ] Demo-Tiefe-Phase

### vermietung ⚠ — ⬜
- [ ] **P1** Backend + Konflikt-Check 🔒 · **P2** Preismodell + Rechnungs-Verknüpfung + Kaution 🔒 · **P3** Übergabe-Doku (Foto/Unterschrift mobil) · **P4** Online-Buchungsportal 🔒 · **P5** Auslastungs-/Umsatz-Reporting
- [ ] Demo-Tiefe-Phase

### inventar ⚠ — ⬜
- [ ] **P1** Backend + Mindestmengen-Alarm 🔒 · **P2** Chargen/Seriennummern · **P3** Barcode/QR-Scan echt · **P4** Picklisten/Kommissionierung 🔒 · **P5** Einkauf-Verknüpfung 🔒 · **P6** Inventur-Workflow
- [ ] Demo-Tiefe-Phase

### einkauf — ⬜
- [ ] **P1** Backend + Wareneingang→Inventar + Mail an Lieferant 🔒 · **P2** Wareneingangs-Dialog (Teilmengen) · **P3** Bestellfreigabe-Workflow 🔒 · **P4** Konditionen/Preislisten 🔒 · **P5** Bestellvorschläge aus Inventar · **P6** Lieferanten-Bewertung + Reporting
- [ ] Demo-Tiefe-Phase

### produktion ⚠ — ⬜ 🔒 (MRP = größter BE-Block; ggf. Feature-Flag OFF bis Handwerk-Pilot)
- [ ] **P1** Backend 🔒 · **P2** Arbeitsschritt-Rückmeldung (Tablet) · **P3** Maschinenbelegung (Konflikt-Check, DnD) 🔒 · **P4** MRP 🔒 · **P5** Kalkulation (Soll/Ist) 🔒 · **P6** QS-Vertiefung
- [ ] Demo-Tiefe-Phase

---

## Nur-Luke (reine Backend-Blöcke, nicht in unserer FE-Zählung)
Mail-IMAP/SMTP · LiveKit (video+dialer) · DSGVO-Tools (security, P0) · DATEV (finanzen/team) · Automatisierungs-Engine · MRP (produktion) · Settings-scope-Tabellen · diverse `/finance/*`-Endpoints (siehe `backend-gaps.md`).

## Review-Pipeline (rollierend für Nico) — Reihenfolge der Modul-Fertigstellung
Jedes Modul wird zu **review-reif** gebracht, dann „→ Nico". Vorschlag mischt schnelle Tiefe-Pässe (FE-reife Module) mit Neubauten, damit Nico nie wartet:

| # | Modul | Aufwand bis review-reif | Status |
|---|---|---|---|
| — | kontakte, calendar, dokumente, zeiterfassung | (Tiefe-Re-Check später) | ✅ bei Nico |
| 1 | **finanzen** | P2.5b–e + P3–5 (FE/mock) | ✅ review-reif → Nico |
| 2 | **Projekte & Aufgaben** (Code: work) | Tiefe-Pass W-1…W-5 (verifiziert, `999260ea`) | ✅ review-reif → Nico |
| 3 | **team** | TM-1…TM-5 (Abwesenheiten-Bug, SelfService, PersonnelDocuments-MSW, OrgChart+i18n, Deactivate) + Umlaut | ✅ review-reif → Nico |
| 4 | **dashboard** | D-1…D-5 + Darien-Fixes F6/F7 (`f4a6844d`) | ✅ review-reif → Nico |
| 5 | **vertraege** | V-1…V-5 + Darien-Fixes F1-F5 (`f4a6844d`) | ✅ review-reif → Nico |
| 6 | **helpdesk** | Demo-tief H-1…H-8 (Store-Actions, DetailModal, Assign/Escalate/Merge, Canned-CRUD, Settings, SLA+Sort, i18n) | ✅ review-reif → Nico (gemergt `a221278d`, QA pending) |
| 7 | **automatisierung** | A-1…A-5 (MSW-Vertrag, DetailModal, Löschen/Duplizieren, Log/Editor, Settings) | ✅ review-reif → Nico (`8274f821`→`29b7d5cd`) |
| 8 | **profil** | P-1…P-5 (Stefan-Vogel-seed, Dokumente-MSW, Avatar/DND, Cleanup, Schlusscheck) | ✅ review-reif → Nico (Sub, gemergt `6f44d65a`) |
| 9 | **security** | FE+Tiefe (DSGVO P0, Backend = Luke früh!) | ⬜ |
| 10 | **mails** | Neubau (5 Phasen) | ⬜ groß |
| 11 | **kommunikation** | 4 Phasen | ⬜ |
| 12 | **video / dialer** | LiveKit-FE | ⬜ |
| 13 | **settings / admin** | Rest-Phasen | ⬜ |
| 14 | **berichte / wiki / formulare / notifications** | Marathon-Reste fertig + Tiefe | ⬜ (Stand verifizieren) |
| 15 | **Branchen ×7** | pilot-nah, zuletzt | ⬜ |

> Reihenfolge ist ein Vorschlag — bei Bedarf umsortieren. **security/DSGVO** ist P0-Launch-Blocker und hängt an Luke → seine Backend-Arbeit dafür früh einplanen, auch wenn das FE-Review später kommt. Nach jedem fertigen Modul: Haken setzen, „→ Nico" markieren, Gesamtzahl aktualisieren.
