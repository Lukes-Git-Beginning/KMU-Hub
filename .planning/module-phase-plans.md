# Modul-Phasen-Planung (alle Cosmi-Module)

> Zweck: Für jedes Modul ohne Detail-Phasenplan einen erstellen (Markt-Recherche + cosmi-modul-marktvergleich), alle offenen Fragen sammeln → Darien klärt in einem Zug → dann autonomer Durchlauf am Stück.
> Reihenfolge (Darien 2026-06-07): **Kontakte ganz fertig → dann nächstes Modul**. Settings-Fundament VOR Modul-Einstellungen.
> Status dieses Dokuments: ⏳ in Arbeit (wird modulweise gefüllt).

## ✅ ENTSCHEIDUNGS-LOG (Darien, 2026-06-07 — Batch-Klärung)
- **Tiefe:** Volle Markt-Parität pro Modul (nicht nur MUSS-Kern).
- **Reihenfolge:** Settings-Fundament → Kontakte P7+P8 → finanzen → team → (mails zurückgestellt) → kommunikation → work → … → Branchen (pilot-getrieben). Security-Modul (DSGVO-P0) **ans Ende** der Modulliste (Darien macht das mit Luke zusammen).
- **Settings-Modell:** 3-Ebenen (Tenant-Default → Modul-Leiter setzt tenant-weit → User-Override). Admin (Chef/IT) aktiviert pro Mitarbeiter im **Team-Modul** (ModuleAssignmentTab/Mitarbeiter-DB) die „erweiterten Moduleinstellungen" für ein/mehrere Module. Wer das Flag hat, sieht in den Moduleinstellungen dieser Module zusätzlich die tenant-weiten („für alle") Settings; sonst nur persönliche. FE mock-first → `backend-gaps.md` für Luke (`tenant_module_leads`-Tabelle).
- **finanzen:** Hybrid — EÜR-Standard, doppelte BuHa als optionale Stufe. SKR03/04-Kontierung + DATEV-Export.
- **team-DATEV:** Markt-getrieben: Stammdaten-Export + praktikable Lohn-Schnittstelle, **DE im MVP**, AT/CH später.
- **mails:** Backend-Architektur (IMAP via Go-Proxy vs Electron-Node) ZURÜCKGESTELLT — Darien klärt mit Luke. mails-Phasen erst danach.
- **Kontakte P8:** „Empfohlen von" = Kontakt-Picker + Empfehler-Report; Segmente A/B/C regelbasiert (Umsatzpotenzial); Beratungsprotokoll nach Finanzberatungs-Standard (Claude defaultet Felder).

## Modul-Inventar (~25 Cosmi-Module, Quelle: cosmi-modul-marktvergleich, 32 Vergleichszeilen inkl. Konkurrenz)

| Gruppe | Module | Phasenplan? |
|---|---|---|
| Vertrieb/Kunde | **kontakte** (inkl. crm/pipeline/leads/aktivitäten/auswertungen), dialer, kommunikation, mails, video, calendar, notifications | kontakte ✅ (Phase 0-8, Block A+B+6 fertig); Rest ✗ |
| Arbeit | work, zeiterfassung, dokumente, wiki, team, helpdesk, formulare, berichte | ✗ |
| Finanzen | finanzen, buchhaltung, vertraege | ✗ (buchhaltung-Migration in current_goals notiert) |
| System/Konto | dashboard, admin, profil, security, settings, automatisierung | ✗ |
| Branchen | rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion | ✗ |

## Kontakte — verbleibende Phasen (greenlit, aber Input-Punkte offen)

### Settings-Fundament (VOR Phase 7, von Darien „zuerst bauen" gewählt)
Generische, modulübergreifende Settings-Architektur:
- **Scope pro Setting:** `personal` (User-Komfort: Default-Ansicht, Sortierung…) vs. `tenant` (modulweit, nur Modul-Leiter/Admin).
- **Modul-Leitung-Rollen:** `<modul>_lead` (z.B. `finance_lead`, `team_lead`) — dürfen tenant-Settings ändern. (current_goals „Pending Phase 2": `tenant_module_leads`-Tabelle = Luke-Backend.)
- **Lock-Icon-Pattern:** tenant-Settings für Nicht-Leiter sichtbar aber gesperrt (Schloss).
- **Wiederverwendbare Settings-Shell:** ein generisches Settings-Fenster/Modal, das je aktivem Modul ein deklariertes Schema rendert (passt zum „Modul-Einstellungen"-Konzept aus `pre-launch-todos.md`).
- **Anbindung an „Modul-Einstellungen"-Konzept (pre-launch-todos):** eigener Einstieg (Knopf links unten?), als Overlay/Modal (kein Routing), kontext-sensitiv (aktives Modul), anpinnbar, RBAC.

→ **OFFENE FRAGEN ↓ (Q1-Q5)**

### Phase 7 — Kontakte-Einstellungen (auf dem Fundament)
Pipeline-Stage-Editor (Hooks da) · Custom-Field-Builder (`CustomFieldsConfig` anbinden) · Lead-Scoring-Regeln · Tag-Verwaltung.
→ Großteils klar; baut auf Fundament + Scope-Marker.

### Phase 8 — Finanzberatungs-Tiefe (Domänen-Input nötig)
„Empfohlen von"-Lookup + Empfehlungs-Report · Beratungsprotokoll-Tab · Mandanten-Segmente (A/B/C).
→ **OFFENE FRAGEN ↓ (Q6-Q8)** — braucht dein Finanzberatungs-Domänenwissen.

---

## OFFENE FRAGEN (für Batch-Klärung)

### Settings-Fundament
- **Q1 — Einstieg:** Ersetzt der „Einstellungen"-Knopf (links unten) die globalen App-Settings durch Modul-Einstellungen, oder zweiter/separater Knopf? (pre-launch-todos: vermutlich separat.)
- **Q2 — Modul-Leitung:** Welche Module brauchen einen „Leiter"-Rang? Nur finance/team/it, oder jedes Modul? Wie wird der gesetzt (Admin vergibt)?
- **Q3 — Scope-Default:** Wenn ich pro Setting personal/tenant entscheide — soll ich das selbst sinnvoll defaulten oder willst du pro Modul drüberschauen?
- **Q4 — Backend:** Settings-Persistenz (User-Scope + Tenant-Scope) + `tenant_module_leads` ist Luke-Backend. Baue ich FE mock-first (wie Leads) und dokumentiere für Luke?
- **Q5 — Anpinnen:** Modul-Einstellungen-Button an die bestehende Modul-Anpin-Logik (oben links) andocken — bestätigt?

### Phase 8 Finanzberatung
- **Q6 — „Empfohlen von":** Freitext, Kontakt-Picker (Empfehler ist selbst Kontakt), oder beides? Empfehlungs-Report = wer hat wie viele gebracht?
- **Q7 — Beratungsprotokoll:** Welche Felder braucht ein Beratungsprotokoll fachlich (DSGVO/Doku-Pflicht Finanzberatung)? Template?
- **Q8 — Segmente A/B/C:** Nach welchen Kriterien (Umsatzpotenzial, Vermögen, Aktivität)? Manuell gesetzt oder regelbasiert?

### Cross-cutting (für den großen Durchlauf)
- **Q9 — Modul-Reihenfolge nach Kontakte:** build-progress sagt work→dashboard→mails→kommunikation. current_goals (älter) sagt Buchhaltung-Migration + Team-Refactor zuerst. Was gilt?
- **Q10 — Tiefe pro Modul:** Volle Markt-Parität je Modul (alle Marktvergleich-Features) oder erst die Kern-Features (MUSS), Rest später?

---

## Executive Summary
- **~25 Cosmi-Module**, davon **kontakte** fertig (Block A+B+Phase 6); Rest hier durchgeplant (je 3-6 Phasen) via Markt-Recherche (cosmi-modul-marktvergleich) + Code-Ist-Stand.
- **Erkenntnis:** Sehr viele Module sind FE bereits weit (oft „Zustand-Mock, kein Backend") — d.h. Hauptarbeit ist oft (a) Backend-Swap (Zustand → TanStack Query + Luke-Endpoints), (b) Feature-Lücken zur Markt-Parität, (c) Cross-Modul-Verknüpfung.
- **Empfohlene Reihenfolge** (nach Kontakte; zu bestätigen Q9): Settings-Fundament → finanzen (inkl. buchhaltung-Löschung) → team → mails → kommunikation → work/zeiterfassung → dokumente/wiki → dashboard → notifications → calendar → helpdesk/formulare/berichte → admin/profil/security → automatisierung → Branchen-Module (pilot-getrieben).
- **Größte Backend-Blöcke (Luke):** Mail-IMAP/SMTP, LiveKit (video+dialer), DSGVO-Tools (security, P0), DATEV (team/finanzen), Automatisierungs-Engine, MRP (produktion).

---

## Cluster 1 — Vertrieb & Kommunikation

### dialer  (FE Phase-1-komplett; LiveKit P0)
- P2 LiveKit-Anbindung + Recording (Consent) · P3 CTI/CRM-Verknüpfung (Click-to-Dial, Auto-Aktivität) · P4 AMD + Auto/Predictive-Dialer · P5 Supervisor-Dashboard + Reporting.
- **Q:** WebRTC-only oder SIP-Trunk (Anbietervertrag)? Predictive-Dialer launch-relevant (UWG §7)? · **BE:** P2-5 fast alles Luke.

### kommunikation/chat  (3-Panel-Chat da)
- P2 DMs + Volltextsuche + Unread · P3 Unified Inbox + Edit/Löschen/Lesezeichen · P4 Audio/Video aus Chat (LiveKit-Bridge) · P5 Bots/Webhooks/Slash-Commands.
- **Q:** Unified Inbox eigenes Modul oder Tab? Huddle via LiveKit? · **BE:** Unread/Aggregation/Bridge = Luke.

### mails  (Mock-Handler da, **kein FE-Modul** — Neubau)
- P1 Mail-UI-Grundgerüst (3-Panel, Compose, Signatur) · P2 Konten + Unified Inbox + Suche · P3 Vorlagen + Regeln/Filter + Labels · P4 CRM-Integration (Thread↔Kontakt, Deal-aus-Mail) · P5 Exchange/EWS + PGP.
- **Q:** IMAP via Electron-Node direkt oder Go-Proxy? Exchange/EWS Launch-relevant? Editor (TipTap)? · **BE:** Konten/Regeln/Sync = Luke.

### video/meetings  (volles Mock-UI, LiveKit P0)
- P2 LiveKit-Anbindung (echte Tiles, Device-Check, Screenshare) · P3 Recording + Consent (gebaut) + Bibliothek · P4 Lobby/Waiting-Room + Moderation · P5 Breakout + Shared-Notes + ActionItems→Work.
- **Q:** Token via Go-Gateway (Security!)? Recording-Storage (Hetzner)? Breakout launch-relevant? Guest-Access. · **BE:** P2-5 fast alles Luke.

### calendar  (nur Layout-Shell, Views = Platzhalter)
- P1 Views (Tag/Woche/Monat + Mini-Cal + Kalender-Liste) · P2 Events-CRUD + RRULE + DnD + Erinnerungen · P3 Mehrere/geteilte Kalender + Einladungen/RSVP · P4 CalDAV-Sync (Admin-Seite existiert!) + Ressourcen · P5 Buchungs-Link (Calendly-Style) + CRM-Integration.
- **Q:** CalDAV-Backend schon echt? Buchungs-Link = PWA/Web-Route? RRULE-Edit-Pattern. · **BE:** Sharing/CalDAV/Buchungs-Link = Luke.

### notifications  (Mock-Handler komplett, nur Widget-FE)
- P1 Notification-Center (Slide-Over, Tabs, Deep-Links, Gruppierung) · P2 Präferenzen + DND/Quiet-Hours · P3 Modul-Gruppierung + Sidebar-Badges + OS-Notifications · P4 Real-time (WebSocket) + Toasts + Sound · P5 Multi-Channel (E-Mail-Digest, SMS, PWA-Push).
- **Q:** Eigene Route + Slide-Over? E-Mail-Absender? Sidebar-Badges mit App-Shell koordinieren? · **BE:** Real-time-Event-Bus + Digest = Luke.

---

## Cluster 2 — Arbeit

### work  (sehr vollständig: Projekte/Kanban/Gantt/Tasks/Timer)
- P1 Daily-Use finalisieren (DnD backend-backed, Timer→Zeiteintrag-Bridge, Inline-Edit) · P2 Portfolio + Auslastungs-View · P3 Automatisierungs-Regeln (wenn/dann) · P4 Externer Zugang (GuestView, Share-Links, Zeit→Finanzen-Export).
- **Q:** Ressourcen-Buchung nötig? · **BE:** Timer-Bridge + Portfolio-Aggregation = Luke.

### zeiterfassung  (dünner Wrapper um Profil-Tab)
- P1 Standalone-Modul (Timer + Projekt/Kunde-Zuordnung, Stundenkonten-Saldo) · P2 ArbZG-Compliance-Report + CSV/XLSX-Export + Pausen-Regeln · P3 Team-Zeiterfassung + Genehmigungs-Workflow · P4 Lohn/DATEV-Export + Zuschläge.
- **Q:** Tray-Widget? Feiertagskalender DE/AT/CH? · **BE:** Stundenkonto + DATEV = Luke.

### dokumente  (sehr vollständig: Spaces/Versionen/OnlyOffice/KI)
- P1 „Coming-soon" beseitigen (Move/Copy, granulare Rechte, Datei-Kommentare) · P2 Volltextsuche (Inhalt) + Metadaten/Tags · P3 Kollaboration (Co-Edit-Status, externe Share-Links m. Ablauf/Passwort) · P4 Governance (Papierkorb/Retention, Audit-Log, DSGVO-Export).
- **Q:** OnlyOffice bleibt? Storage-Quota pro Tier? · **BE:** Move/Copy + Volltext-Index = Luke.

### wiki  (volle UI auf Zustand-Store)
- P1 Backend-Swap (TanStack Query, Luke-Endpoints) · P2 Editor-Vollständigkeit (Anhänge, Tabellen, @Mention, `[[Verlinkung]]`) · P3 Zugang/Freigabe (echte Share-Links, Public-Modus, Kategorie-RBAC) · P4 Suche + KI (Volltext, Artikel-aus-Ticket-Vorschlag).
- **Q:** Wiki auch als externe Kunden-KB (Helpdesk-Selfservice)? · **BE:** P1 komplett Luke.

### team  (12 Tabs, Mitglieder/Leave via Query, Rest Mock)
- P1 Trainings Backend-Swap + Pflichtschulung-Tracking · P2 Personalakte ↔ Dokumente-Modul + Organigramm editierbar + Leave-Typen · P3 DATEV-Lohnschnittstelle · P4 HR-Selfservice + Minderjährigen-Regeln + Vertrags-Vorlagen.
- **Q:** DATEV-Tiefe (nur Stammdaten oder Lohn)? Nur DE im MVP? · **BE:** hr-Endpoints + DATEV = Luke. (Hinweis: current_goals nennt Team-Refactor + Permission-Editor + Wizard-Polish als priorisiert.)

### helpdesk  (gut, auf Zustand-Store)
- P1 Backend-Swap + CRM-Kontakt-Lookup · P2 Multi-Channel (Mail→Ticket) + Merge + Ticket-Zeiterfassung · P3 Automatisierung (Eskalation, Auto-Close) · P4 Self-Service-Portal + öffentliche KB.
- **Q:** Mail-Eingang via IMAP-Poll? Portal in Cosmi oder separate URL? · **BE:** P1/2 Luke + Mail-Listener.

### formulare  (sehr vollständig: Builder/Logik/Submissions)
- P1 DnD-Reordering + DSGVO-Feld + E-Mail-Benachrichtigung · P2 Webhook-Config + Automatisierung + CRM-Integration (Eingang→Kontakt) · P3 Embedding (iFrame/JS) + Analytics · P4 Zahlungen (Stripe) + E-Signatur.
- **Q:** Free/Paid-Tier? Embed-CORS wer konfiguriert? · **BE:** Webhook/CRM-Link = Luke.

### berichte  (4 Tabs, Hooks da, kaum echte Charts)
- P1 echte Charts (recharts — wie Phase-6-Auswertungen!) + Zeitraum-Filter + verschiebbare Kacheln · P2 No-Code Query-Builder (modul-übergreifend) · P3 Export (PDF/CSV/XLSX) + geplante Berichte + Alerts · P4 DATEV + externe BI (Power BI/Metabase).
- **Q:** Welche 3-5 Kern-KPIs täglich? Konfigurierbares Startdashboard? · **BE:** flexibler Query-Endpoint = Luke. (Synergie: `useChartTheme` + Auswertungen-Muster wiederverwenden.)

---

## Cluster 3 — Finanzen

### finanzen  (13 Tabs, Kern backend-connected; Ausgaben/Transaktionen noch Zustand)
- P1 **Migration abschließen** (buchhaltung löschen, ExpensesTab/TransactionsTab → TanStack Query, Berichte-Tab-Entscheidung) · P2 Kern-Workflows (Angebot→Rechnung-Konvertierung, wiederkehrende Rechnungen, Fremdwährung CHF/USD, mehrstufiges Mahnwesen) · P3 GoBD + Steuerberater-Zugang + EÜR-Auswertung · P4 Banking + Zahlungsabgleich (CAMT/MT940) + Bexio-API.
- **Q:** Nur EÜR oder auch doppelte BuHa? SKR03/04-Kontierung oder DATEV-Export? Fremdwährung: manuell oder ECB-Feed? · **BE:** expenses/transactions/DATEV/Recurring/Banking = Luke.

### buchhaltung  (dead code, Migrations-Aufgabe)
- P1 Aufräumen: Test nach finanzen verschieben, Ordner `git rm`, i18n bereinigen. **Keine Feature-Phasen** — lebt in finanzen.
- **Q:** Sidebar-Name „Buchhaltung" statt „Finanzen"? Berichte-Tab wohin?

### vertraege  (vollständige FE-Only auf Zustand-Store)
- P1 Backend-Anbindung + Audit-Log + Erinnerungs-Notifications · P2 Dokumente echt anbinden (Upload statt String, Versionen, PDF-Viewer) · P3 E-Signatur echt (Eigen-Canvas EES vs. Skribble eIDAS) · P4 CRM/Finanzen-Verknüpfung + KI-Fristencheck.
- **Q:** Signatur-Rechtsstufe (EES vs. Skribble-Kosten)? Arbeitsverträge in Cosmi oder Team-Modul? · **BE:** contracts-CRUD + Signer = Luke.

---

## Cluster 4 — System & Konto

### settings  (14 Tabs, ausgebaut) — **enthält das Settings-Fundament (VOR Kontakte-Phase 7)**
- P1 **Settings-Fundament:** scope-Awareness (personal/tenant), `<ModuleSettingsShell>` (→ `components/shared/`), Modul-Leiter-Rollen, 3-Ebenen-Hierarchie (Tenant-Default → Modul-Lead → User) · P2 Modul-Settings-Tabs für fehlende Module (CRM/Dialer/Work/Helpdesk…) · P3 Workspace-Defaults real (Firmendaten, Währung, Zeitzone, GJ-Start) · P4 Integrationen echt (Bexio/Lexware/DATEV OAuth) · P5 Notification-Settings + KI-Governance.
- **Q (kritisch):** Settings-DB-Modell — `tenant_defaults` + `user_overrides` (sauber für Multi-Tenant, empfohlen) vs. einfache `user_settings.scope`? Wer darf Modul-Leiter setzen? · **BE:** Settings-scope-Tabellen = Luke.

### admin  (4 Tabs, RBAC-Guard)
- P1 Benutzerverwaltung (Liste, Einladen, Rolle ändern, deaktivieren) · P2 Rollenmodell/RBAC-Matrix + Modul-Leiter verwaltbar · P3 Abo/Lizenz real + Modul-Aktivierung tenant-weit · P4 Branding persistent (tenant_branding) · P5 Ressourcen-Monitoring + Tenant-Verwaltung (Option-B Sprint 2/3).
- **Q:** User-Mgmt in Admin vs. Team-Modul? Custom Roles oder 5 Fixed? Modul-Aktivierung zur Laufzeit? · **BE:** Invite-Flow/RBAC/Billing = Luke.

### profil  (4 Tabs, gut)
- P1 Avatar-Upload echt + Presence-Status · P2 Benachrichtigungs-Präferenzen im Profil · P3 Shortcuts zu Appearance/Security + Account-Info · P4 Profil-Karte (Overlay anywhere, Ping→Chat).
- **Q:** Avatar-Storage jetzt oder Phase E? Presence real-time oder Polling? · **BE:** Avatar-Upload + Presence = Luke.

### security  (9 Sub-Seiten, viel Mock) — **DSGVO-Tools = P0-Launch-Blocker (R2)**
- P1 Audit-Log + Sessions echt (Export unveränderlich) · P2 Passwort-Policy + IP-Zugriff enforced · P3 **DSGVO-Tools real** (Export Art.15/20, Erasure, DSAR, Retention) · P4 MFA-Erweiterung (WebAuthn) + Vault · P5 SSO/SAML/OIDC (Post-MVP).
- **Q:** Vault eigen (pgcrypto) vs. extern? DSGVO-Export-Tiefe (alle Module)? · **BE:** alles Luke; P1+P3 sind P0.

### automatisierung  (sehr vollständig: Wizard/ReactFlow/Mock)
- P1 Wizard→echtes Backend (CRUD, Enable/Disable, Execution-Log) · P2 Flow-Editor mit Branching/Loop · P3 Webhook-Inbound + HTTP-Action · P4 Modul-übergreifende Workflows + Permissions · P5 Template-Marktplatz + KI-Assistent.
- **Q:** Execution-Engine eigener Go-Service + Job-Queue? ReactFlow-Editor-Reife? Webhook-URL rotierbar? · **BE:** Engine = Luke (größerer Block).

### dashboard  (vollständig, Layout-Persistenz Mock)
- P1 Backend-Persistenz + Widget-Konfiguration · P2 DnD-Resize/Reorder · P3 KPI-Widgets modul-/lizenzabhängig · P4 Modul-übergreifende Übersicht + echte Alerts · P5 Team-Dashboard + Rollen-Templates + Presence.
- **Q:** 12-Spalten-Grid (empf.) vs. frei? Team-Default-Layout durch Manager? · **BE:** Layout-Persistenz + Alert-Queries = Luke.

---

## Cluster 5 — Branchen (pilot-getrieben; ⚠ = Mobile/PWA-Bedarf)

### rapporte  (reif; Solar-Pilot-Kern) ⚠
- P1 Backend-Persistenz + Foto-Upload (Hetzner Storage) · P2 PDF-Export echt (Gotenberg/wkhtmltopdf) · P3 ⚠ GPS/Standort · P4 ⚠ **Offline-Fähigkeit** (Service Worker + IndexedDB-Queue) · P5 Approval-Backend + Notifications · P6 Solar-Vertiefung (PV-Vorlagen).
- **Q:** Offline (P4) VOR GPS (P3) für Solar-Pilot? Approval auch mobil? · **BE:** P1/2/5 = Luke; P3/4 = FE/PWA.

### schichten  (reif; ArbZG-Check da; Mitarbeiter hardcoded) ⚠
- P1 Backend + Mitarbeiter aus Team-Modul · P2 ⚠ Self-Service-Portal (eigene Schichten/Tausch, mobil) · P3 Monats-/Mehrwochenansicht · P4 Auto-Planer (regelbasiert) · P5 Minderjährigen-Schutz + AT/CH-Feiertage · P6 Reporting + DATEV.
- **Q:** Schichten+Rapporte gleiche User-Basis? Self-Service als PWA? · **BE:** P1/4/6 = Luke.

### fuhrpark  (reif; GPS/Fahrtenbuch Mock) ⚠
- P1 Backend + TÜV/Versicherung-Reminder · P2 Fahrzeugbuchungs-Pool · P3 ⚠ Fahrtenbuch live (§7 EStG, 1%-Regel) · P4 Führerscheinkontrolle (OCR) · P5 GPS-Echtverfolgung (Telematik-Hardware) · P6 TCO + Tankkartenimport.
- **Q:** Telematik-Hardware beim Pilot? Pool-Logik mit Vermietung teilen? · **BE:** P1/4/5 = Luke.

### vermietung  (vollständig) ⚠
- P1 Backend + Konflikt-Check · P2 Preismodell + Rechnungs-Verknüpfung + Kaution · P3 ⚠ Übergabe-Doku (Foto/Unterschrift mobil) · P4 Online-Buchungsportal · P5 Auslastungs-/Umsatz-Reporting.
- **Q:** `fahrzeug`-Überlappung mit Fuhrpark? Buchungsportal in App oder Website? · **BE:** P1/2/4 = Luke.

### inventar  (vollständig; Scan-Icon nur Mock) ⚠
- P1 Backend + Mindestmengen-Alarm · P2 Chargen/Seriennummern · P3 ⚠ Barcode/QR-Scan echt (Kamera/Handscanner) · P4 Picklisten/Kommissionierung · P5 Einkauf-Verknüpfung (Bestellvorschläge) · P6 Inventur-Workflow (Differenz, GoBD-Buchung).
- **Q:** Handscanner (HID) vs. Smartphone-Kamera? Chargen für Pharma/Lebensmittel? · **BE:** P1/4/5 = Luke.

### einkauf  (vollständig)
- P1 Backend + Wareneingang→Inventar + Mail an Lieferant · P2 Wareneingangs-Dialog (Teilmengen) · P3 Bestellfreigabe-Workflow (mehrstufig) · P4 Konditionen/Preislisten · P5 Bestellvorschläge aus Inventar · P6 Lieferanten-Bewertung + Reporting.
- **Q:** Freigabe für Solar-Pilot relevant? Mail via Cosmi-Mails oder SMTP? · **BE:** P1/3/4 = Luke.

### produktion  (vollständig; MRP fehlt) ⚠
- P1 Backend · P2 ⚠ Arbeitsschritt-Rückmeldung (Tablet/Maschine) · P3 Maschinenbelegung echt (Konflikt-Check, DnD) · P4 **MRP** (Material-Verfügbarkeit ggü. Inventar, Fehlteile→Einkauf) · P5 Kalkulation (Soll/Ist) · P6 QS-Vertiefung (Prüfpläne, Messwerte, Sperr-Workflow).
- **Q:** Produktion überhaupt Launch-relevant oder Feature-Flag OFF bis Handwerk-Pilot? MRP = größter BE-Block. · **BE:** P1/3/4/5 = Luke (MRP komplex).

---

## KONSOLIDIERTE OFFENE FRAGEN (Batch-Klärung)
> Die Q1-Q10 oben (Settings-Fundament, Phase 8, Reihenfolge/Tiefe) PLUS die wichtigsten pro Modul. Beim nächsten Mal in einem Zug klären, dann autonomer Durchlauf.

**Strategie/Querschnitt**
- Q9 Modul-Reihenfolge bestätigen (Vorschlag oben) · Q10 Tiefe: volle Parität vs. MUSS-zuerst.
- A. **Backend-Realität:** Sehr viel ist „FE fertig, Backend fehlt". Baue ich konsequent FE-mock-first + `backend-gaps.md` für Luke, und Luke zieht nach? (Sonst blockieren wir uns.)
- B. **Pilot-Fokus:** Solar-Pilot (Juli) → rapporte/schichten/fuhrpark + Mobile/PWA. Wann kommt die PWA-Basis (Offline)? Das gated mehrere Branchen-Phasen.
- C. **DSGVO-Security P0:** security P1+P3 sind Launch-Blocker — vorziehen?

**Modul-spezifisch (Top-Entscheidungen)**
- mails: IMAP via Electron-Node oder Go-Proxy? Exchange/EWS Launch?
- video/dialer: LiveKit-Token via Gateway (Security), Recording-Storage, SIP-Trunk ja/nein?
- settings: DB-Modell (tenant_defaults+user_overrides empfohlen) + wer setzt Modul-Leiter?
- finanzen: nur EÜR oder doppelte BuHa? SKR-Kontierung oder DATEV-Export?
- vertraege: E-Signatur EES-eigen vs. Skribble (Kosten)?
- team: DATEV-Tiefe + nur DE im MVP?
- produktion: Launch-relevant oder Feature-Flag OFF?
- calendar: CalDAV-Backend schon echt? Buchungs-Link Scope?

---

## Status / Nächster Schritt
- ✅ Phasenpläne für alle Module erstellt (dieser Doc).
- ⏭ Wenn Darien zurück: konsolidierte Fragen in einem Zug klären → dann autonomer Durchlauf in bestätigter Reihenfolge, je Phase: bauen + Screenshot-QA + commit.
- Greenlit & sofort startbar ohne weitere Klärung: **finanzen P1 (buchhaltung-Löschung + Expenses/Transactions-Migration)** und **Settings-Fundament P1** — beide von Darien bereits in current_goals/Antworten priorisiert.
