# Master-Tracker — Kompletter Phasenplan bis Modul-Reviews

> **Der eine Ort zum Abarbeiten.** Jede Phase = eine abhakbare Zeile. Detail-Beschreibungen je Phase: `.planning/module-phase-plans.md` + `.planning/reviews/<modul>.md`.
> **Rollen:** Luke = Backend · Nico = Reviews fertiger Module · wir = Phasen abarbeiten.
> **Stand:** 2026-06-16.

## Legende & Zählung
✅ fertig · 🔨 läuft · ⬜ offen · 🔒 FE bau ich mock-first, echtes Backend kommt von Luke · 🔁 Stand verifizieren
- **Eine Gesamtzahl** (Entscheidung Darien): 🔒-Phasen sind als FE-mock-first mitgezählt. Reine Backend-only-Arbeit steht separat unten unter „Nur-Luke".
- **Demo-Tiefe** ([[feedback_module_depth_standard]]): eigene Phase pro **bereits gebautem** Modul (Audit tote Buttons/Detail-Views + Fix). Bei **neu gebauten** Modulen ist Tiefe direkt eingebaut (keine Extra-Phase). finanzen P2.5 = Referenz.

## Gesamtstand
- **~155 offene Phasen** über ~32 Module (grobe, eher konservative Zahl — Verifizieren der Marathon-Module kann sie senken).
- **Fertig:** kontakte (Kern), zeiterfassung; calendar/dokumente weitgehend (🔁).
- **Aktive Spur:** Tiefe-Re-Check der 4 fertigen Module → finanzen P2.5 → finanzen P3–5 → Module nach Reihenfolge.

---

## ▶ SOFORT — Tiefe-Re-Check der „fertigen" Module (vor Nicos Review)
Schneller Audit gegen die Tiefe-Vorgabe + offensichtliche Fixes, damit Nico tiefe-vollständige Module reviewt. 1 Phase pro Modul.
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

### work — ⬜ (sehr vollständig FE)
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

### berichte — 🔨 (Nico-Sparklines, tw.)
- [ ] **P1** echte Charts (recharts) + Zeitraum-Filter + verschiebbare Kacheln
- [ ] **P2** No-Code Query-Builder (modul-übergreifend) 🔒
- [ ] **P3** Export (PDF/CSV/XLSX) + geplante Berichte + Alerts
- [ ] **P4** DATEV + externe BI 🔒
- [ ] Demo-Tiefe-Phase

### team — ⬜ (12 Tabs, tw. Query)
- [ ] **P1** Trainings Backend-Swap + Pflichtschulung-Tracking 🔒
- [ ] **P2** Personalakte↔Dokumente + Organigramm editierbar + Leave-Typen
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
- [x] P1 Faktura-Kette · P2 Ausgaben/Kontierung · **P2.5a** Angebots-/Gutschrift-Detail
- [ ] **P2.5b** Ausgaben-/Transaktions-/Wiederkehrend-Detail + OP-Liste→Rechnung + Dashboard-Listen
- [ ] **P2.5c** PDF-Vorschau aus echten Daten + Downloads (PDF/CSV) sichtbar wirksam
- [ ] **P2.5d** Mahnwesen verkabeln + Mahn-Detail + Zahlung/Settings speichern 🔒
- [ ] **P2.5e** Hardcoded-Mocks→MSW (Banking, Belegkette, Stunden→Rechnung, Audit-Log)
- [ ] **P3** DATEV-EXTF + Bexio-OAuth + BMD-CSV 🔒
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

## Offene Reihenfolge-Klärung
Nach finanzen: Vorschlag **security/DSGVO früh** (P0-Blocker) → mails → kommunikation → team → work → dashboard → profil → restliche System → Branchen (pilot-nah). Reihenfolge bestätigen, dann arbeite ich sie hier von oben ab und aktualisiere Haken + Gesamtzahl.
