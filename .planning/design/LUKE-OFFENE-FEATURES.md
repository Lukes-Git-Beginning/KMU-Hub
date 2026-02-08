# Offene Features: Was fehlt noch in Lukes Roadmap?

> **Von:** Darien (Design)
> **Fuer:** Luke (Backend)
> **Stand:** 2026-02-08
> **Grundlage:** 103 genehmigte Features aus Design-Review, verglichen mit Lukes Roadmap (Phasen 4-13)
> **Status:** ABGESCHLOSSEN — Alle Features zugeordnet (2x mit Luke besprochen)

## Worum geht es?

Wir haben im Design-Review 103 Features fuer KMU Hub freigegeben. Viele davon brauchen
Backend-Arbeit. Ich habe Lukes bestehende Roadmap (Phasen 4-13) Punkt fuer Punkt verglichen
und geschaut: **Was ist schon eingeplant, was fehlt komplett, und was ist nur teilweise abgedeckt?**

Diese Liste ist zum Besprechen gedacht — nicht alles muss sofort rein, aber wir sollten
wissen wo die Luecken sind und wann wir was angehen.

---

## Legende

| Symbol | Bedeutung |
|--------|-----------|
| 🔴 | **Komplett fehlend** — keine Phase deckt das ab |
| 🟡 | **Teilweise abgedeckt** — Phase existiert, aber Feature nicht explizit drin |
| 🟢 | **Abgedeckt** — ist in Lukes Roadmap enthalten |
| ⭐ | **Besonders wichtig** — strategisch oder USP-relevant |

---

## 1. KOMPLETT FEHLENDE BEREICHE (keine Phase geplant)

### Dokumente & Dateien — KEIN Plan vorhanden!

Das ist die groesste Luecke. Lukes Roadmap hat keine Phase fuer Dokumenten-Management.
MinIO wird zwar fuer Anhaenge benutzt (Chat, E-Mail), aber ein richtiges Dateisystem fehlt.

> **Luke:** Eigene Phase 11 (Documents & Files)

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| F1 | 🟢 | **Datei-Browser** | Ordnerstruktur-API, Verzeichnis-Operationen (erstellen, verschieben, loeschen) | Mittel |
| F2 | 🟢 | **Drag & Drop Upload** | File-Upload Endpoint mit Fortschrittsanzeige (evtl. Multipart/Chunked) | Klein |
| F3 | 🟢 | **Datei-Vorschau** | Thumbnail-Generation, PDF-Rendering serverseitig | Mittel |
| F4 | 🟢 | **Versionierung** | Versionshistorie pro Datei, alte Versionen abrufbar, Speicherplatz-Management | Mittel |
| F5 | 🟢 | **Freigabe/Sharing** | Berechtigungsstufen pro Datei (privat, Team, Projekt, oeffentlich) + Freigabe-Links | Mittel |
| F6 | 🟢⭐ | **Volltext-Suche in Docs** | Indexing-Service (z.B. Meilisearch), PDF/Word-Text-Extraktion, Suchindex-Updates | **Gross** |
| F7 | 🟢 | **Dokument-Tags** | Tag-System fuer Dateien (aehnlich wie Kontakt-Tags) | Klein |

---

### Meeting-Management — Video ja, Meetings nein

Lukes Phase 8 deckt Video/Audio/Screenshare ab (LiveKit). Aber das ganze **Meeting-Drumherum**
(Planung, Agenda, Notizen, Aufgaben) fehlt komplett.

> **Luke:** In Phase 8 integriert (Video, Voice & Meetings). Presence-System ebenfalls Phase 8.

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| D1 | 🟢 | **Meeting-Uebersicht** | Meeting-Service mit CRUD, Filterung nach Status (live/geplant/vergangen) | Mittel |
| D2 | 🟢 | **Meeting-Detail + Agenda** | Agenda-Datenmodell, Teilnehmerverwaltung, Meeting-Links | Mittel |
| D3 | 🟢 | **Meeting-Notizen** | Echtzeit-Notizen (SharedDoc oder CRDT), pro-Meeting gespeichert | Mittel |
| D4 | 🟢 | **Meeting-ToDos** | Aufgaben aus Meeting erstellen → verknuepft mit Task-System (Phase 6) | Klein |
| D8 | 🟢⭐ | **Whiteboard** | Canvas-Library, Echtzeit-Sync ueber WebSocket/CRDT, Zeichendaten speichern | **Gross** |
| D16 | 🟢 | **Meeting-Raeume** | Persistente virtuelle Raeume, Drop-in Logik, Raum-Status (wer ist drin) | Mittel |
| B10 | 🟢 | **Team-Status/Presence** | WebSocket Presence-System: Wer ist online, abwesend, in Meeting | Mittel |
| I2 | 🟢 | **Online-Status** | Presence-Dots ueberall (Chat, Team-Seite, Sidebar) | (Teil von B10) |

---

### Externe Integrationen — das groesste Backend-Thema

Drei Integrations-Bloecke fehlen komplett in der Roadmap:

#### a) Kalender-Sync (CalDAV/iCal)

> **Luke:** Eigene Mini-Phase 14 (nicht in Calendar-Phase)

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| H8 | 🟢⭐ | **CalDAV/iCal Sync** | OAuth-Flows fuer Google/Outlook, bidirektionaler Sync, Konfliktloesung | **Gross** |

#### b) Kommunikations-Plattformen

> **Luke:** Eigene Mini-Phase 15

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| H9 | 🟢⭐⭐ | **Teams/Slack Integration** | API-Anbindung an MS Teams + Slack, Webhook-Empfang, Nachrichten-Routing, Antworten senden, evtl. Microsoft-Zertifizierung fuer MS Teams | **Sehr gross** |

#### c) Buchhaltungs-Software

> **Luke (Runde 2):** Jede Integration bekommt eine eigene Phase.

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| J1 | 🟢 | **Bexio-Integration** | Phase 16. Bexio REST API Client, Daten-Mapping, Sync-Logik, Fehlerbehandlung | **Gross** |
| J2 | 🟢 | **Abacus-Integration** | Phase 17. OAuth2/API-Key Auth, bidirektionaler Kontakt-Sync, Rechnungs-Sync | **Gross** |
| J3 | 🟢 | **Run my Accounts** | Phase 18. Auth, Kontakt-Sync, Finanzdokument-Sync | **Gross** |

---

## 2. FEATURES DIE IN BESTEHENDEN PHASEN FEHLEN

### Phase 6 (Projekte) — alles eingeplant oder fertig

> **Luke (Runde 1):** Gantt + Abhaengigkeiten kommen in v1, Phase 6, als Plans 06-09 + 06-10
> **Luke (Runde 2):** Sub-Tasks + Vorlagen bereits fertig gebaut. Zeiterfassung als naechstes (Plan 06-10).

| # | Status | Feature | Stand | Aufwand |
|---|--------|---------|-------|---------|
| C3 | 🟢⭐ | **Gantt-Chart** | Phase 6, Plan 06-09 | **Gross** |
| C11 | 🟢⭐ | **Task-Abhaengigkeiten** | Phase 6, Plan 06-10 | **Gross** |
| C7 | 🟢 | **Sub-Tasks** | **Fertig gebaut!** Multi-level mit parent_task_id | Klein-Mittel |
| C8 | 🟢 | **Zeiterfassung pro Task** | Phase 6, Plan 06-10 (naechster Plan). Timer Start/Stop/Pause + manuell | Mittel |
| C10 | 🟢 | **Vorlagen** | **Fertig gebaut!** Projekte + Tasks als Vorlage speichern + daraus erstellen | Mittel |

---

### Phase 12 (Finance) — Ausgabenverwaltung auf v2 verschoben

> **Luke (Runde 2):** Ausgabenverwaltung kommt in v2. Integrationen haben eigene Phasen (16-18).

| # | Status | Feature | Stand | Aufwand |
|---|--------|---------|-------|---------|
| J5 | v2 | **Ausgabenverwaltung** | Bewusst auf v2 verschoben. Belege + Kategorisierung | Mittel |
| J1 | 🟢 | **Bexio** | Phase 16 | Gross |
| J2 | 🟢 | **Abacus** | Phase 17 | Gross |
| J3 | 🟢 | **Run my Accounts** | Phase 18 | Gross |

---

### Phase 13 (HR) — 2 Features nicht explizit erwaehnt

HR ist jetzt Phase 13. Diese zwei Features wurden nicht angesprochen, passen aber thematisch rein:

| # | Status | Feature | Vermutung | Aufwand |
|---|--------|---------|-----------|---------|
| I4 | 🟡 | **Organigramm** | Passt in Phase 13 (HR). Braucht Hierarchie-Daten (Vorgesetzter-Beziehung) | Mittel |
| I5 | 🟡 | **Arbeitsinteressen** | Passt in Phase 13 (HR). Einfache Profil-Felder + Tags | Klein |

---

## 3. SYSTEM-FEATURES — GELOEST: Phase 9 (Security & Compliance)

> **Luke:** 2FA + DSGVO + Audit-Log + i18n kommen alle in Phase 9 (Security & Compliance),
> direkt nach Phase 8. Das heisst Phase 9 ist NICHT mehr Email sondern Security.
> Email rueckt auf Phase 10 (oder spaeter).

### Sicherheit & Compliance — Phase 9

| # | Status | Feature | Was Luke bauen muesste | Aufwand | Wann noetig? |
|---|--------|---------|----------------------|---------|-------------|
| L4 | 🟢 | **Vault/Sicherheit** | Verschluesselungs-Einstellungen, Key-Management | Mittel | Phase 9 |
| L5 | 🟢⭐ | **2FA** | TOTP (Google Authenticator) und/oder WebAuthn. Setup-Flow, Backup-Codes | Mittel | Phase 9 |
| L6 | 🟢 | **Session-Verwaltung** | Aktive Sessions auflisten + einzeln beenden koennen | Klein | Phase 9 |
| L7 | 🟢⭐ | **Audit-Log** | Event-Logging: Wer hat was wann geaendert. Admin-Feature | Mittel | Phase 9 |
| L8 | 🟢⭐ | **DSGVO-Export** | User kann alle eigenen Daten als ZIP herunterladen. **EU-Pflicht!** | Mittel | Phase 9 |
| L3 | 🟡 | **Auto-Save** | Draft-Storage fuer Formulare | Klein | Nicht in Phase 9 erwaehnt |
| K5 | 🟢 | **Sprache wechselbar (i18n)** | i18n-Framework (DE/FR/IT/EN). Infrastructure-Level | Mittel | Phase 9 |

---

### Navigation & Uebergreifendes

> **Luke (Runde 2):** Globale Suche kommt in Phase 11 (Documents & Files) als Unified Search Endpoint.

| # | Status | Feature | Was Luke bauen muesste | Aufwand | Wann noetig? |
|---|--------|---------|----------------------|---------|-------------|
| A1 | 🟢⭐ | **Globale Suche** | Phase 11. Unified Search ueber CRM, PM, Chat, Email, Dateien | **Gross** | Phase 11 |
| A2 | 🟢 | **Tastaturkuerzel** | Frontend-only (unser Bereich) | Mittel | — |
| A7 | 🟢 | **Multi-Tabs** | Frontend-only (unser Bereich) | Mittel | — |
| B8 | 🔴 | **Wetter-Widget** | Externe Wetter-API anbinden | Klein | Nice-to-have |

---

### CRM- & Chat-Erweiterungen

> **Luke (Runde 2):** Zwei-Ebenen-Kontakte + Import/Export kommen in Phase 10 (Email).
> Emoji-Reaktionen in Phase 8. Chat-Datei-Berechtigungen in Phase 11.
> Duplikat-Erkennung + Kontakt-Gruppen bewusst auf v2 verschoben.

| # | Status | Feature | Stand | Aufwand |
|---|--------|---------|-------|---------|
| E10 | 🟢⭐ | **Zwei-Ebenen-Kontakte** | Phase 10. Zentrale vs. persoenliche Kontakte mit Scope-Feld | Mittel |
| E6 | 🟢 | **Kontakt-Import/Export** | Phase 10. CSV/vCard Import/Export zusammen mit Email-Phase | Mittel |
| D12 | 🟢 | **Emoji-Reaktionen im Chat** | Phase 8. Zusammen mit Presence-System | Klein |
| D14 | 🟢 | **Datei-Sharing + Berechtigungen** | Phase 11. Chat-Dateien ueber zentralen File Manager mit Berechtigungen pro User/Rolle | Mittel |
| E7 | v2 | **Duplikat-Erkennung** | Bewusst auf v2 verschoben. Fuzzy Matching + Merge-UI komplex | Mittel |
| E9 | v2 | **Kontakt-Gruppen** | Bewusst auf v2 verschoben. Statische + dynamische Gruppen | Klein |

---

### Personalisierung

Nicht angesprochen, vermutlich Kleinkram fuer spaeter:

| # | Status | Feature | Vermutung | Aufwand |
|---|--------|---------|-----------|---------|
| K1 | 🔴 | **Arbeitsprofile** | Evtl. Phase 9 oder Frontend-only | Mittel |

---

## 4. FINALE ZUSAMMENFASSUNG

### Lukes Roadmap (20 Phasen, final)

| Phase | Inhalt | Status |
|-------|--------|--------|
| 1-3 | Foundation (Auth, CRM, Chat) | Fertig |
| 4 | Notifications + Gateway | Fertig |
| 5 | Desktop App Shell | Fast fertig (6/7 Plans) |
| 6 | Project Management + Gantt + Abhaengigkeiten + Sub-Tasks + Vorlagen + Zeiterfassung | In Arbeit |
| 7 | Calendar & Scheduling | Geplant |
| 8 | Video, Voice & Meetings + Presence + Emoji-Reaktionen | Geplant |
| 9 | Security & Compliance + i18n | Geplant |
| 10 | Email Integration + Zwei-Ebenen-Kontakte + Import/Export | Geplant |
| 11 | Documents & Files + Globale Suche + Chat-Datei-Berechtigungen | Geplant |
| 12 | Finance Module | Geplant |
| 13 | HR Module | Geplant |
| 14 | CalDAV/iCal Sync (Mini-Phase) | Geplant |
| 15 | Teams/Slack Integration (Mini-Phase) | Geplant |
| 16 | Bexio Integration | Geplant |
| 17 | Abacus Integration | Geplant |
| 18 | Run my Accounts Integration | Geplant |
| 19 | Automation Engine | Geplant |
| 20 | Plugin System | Geplant |

### Alle eingeplanten Features (von 🔴 auf 🟢)

| Feature | Wo eingeplant |
|---------|--------------|
| C7 Sub-Tasks, C10 Vorlagen | Phase 6 — **bereits fertig gebaut** |
| C3 Gantt, C11 Abhaengigkeiten, C8 Zeiterfassung | Phase 6 (Plans 06-09, 06-10) |
| F1-F7 Dokumente | Phase 11 (Documents & Files) |
| A1 Globale Suche | Phase 11 (Unified Search) |
| D1-D4, D8, D16 Meetings | Phase 8 (Video, Voice & Meetings) |
| B10, I2 Presence | Phase 8 |
| D12 Emoji-Reaktionen | Phase 8 |
| D14 Chat-Datei-Berechtigungen | Phase 11 |
| L4-L8 Sicherheit | Phase 9 (Security & Compliance) |
| K5 i18n | Phase 9 |
| E10 Zwei-Ebenen-Kontakte | Phase 10 (Email) |
| E6 Import/Export | Phase 10 (Email) |
| H8 CalDAV | Phase 14 |
| H9 Teams/Slack | Phase 15 |
| J1 Bexio | Phase 16 |
| J2 Abacus | Phase 17 |
| J3 Run my Accounts | Phase 18 |
| A2 Tastaturkuerzel | Frontend-only (wir) |
| A7 Multi-Tabs | Frontend-only (wir) |

### Bewusst auf v2 verschoben

| # | Feature | Grund |
|---|---------|-------|
| E7 | Duplikat-Erkennung | Fuzzy Matching + Merge-UI komplex |
| E9 | Kontakt-Gruppen | Statische + dynamische Gruppen |
| J5 | Ausgabenverwaltung | Belege + Kategorisierung |

### Nicht angesprochen (Kleinkram, vermutlich in bestehende Phasen integrierbar)

| # | Feature | Aufwand | Vermutung |
|---|---------|---------|-----------|
| I4 | Organigramm | Mittel | Phase 13 (HR) |
| I5 | Arbeitsinteressen | Klein | Phase 13 (HR) |
| K1 | Arbeitsprofile | Mittel | Phase 9 oder Frontend-only |
| B8 | Wetter-Widget | Klein | Nice-to-have, evtl. Frontend-only mit externer API |
| L3 | Auto-Save | Klein | Nice-to-have |

---

## 5. FRAGEN-LOG

### Runde 1 (10 Fragen) — alle beantwortet

1. ~~Dokumente~~ → Phase 11
2. ~~Meeting-Management~~ → Phase 8
3. ~~Gantt + Abhaengigkeiten~~ → Phase 6, Plans 06-09 + 06-10
4. ~~Teams/Slack~~ → Phase 15
5. ~~CalDAV~~ → Phase 14
6. ~~2FA + DSGVO + Audit-Log~~ → Phase 9 (Security & Compliance)
7. ~~Presence~~ → Phase 8
8. ~~i18n~~ → Phase 9
9. ~~Tastaturkuerzel~~ → Frontend-only (wir)
10. ~~Multi-Tabs~~ → Frontend-only (wir)

### Runde 2 (6 Fragen) — alle beantwortet

1. ~~Globale Suche~~ → Phase 11
2. ~~Zwei-Ebenen-Kontakte + Import/Export~~ → Phase 10
3. ~~Emoji-Reaktionen~~ → Phase 8
4. ~~Chat-Datei-Berechtigungen~~ → Phase 11
5. ~~Bexio/Abacus/RmA~~ → Phasen 16, 17, 18
6. ~~Sub-Tasks, Vorlagen, Zeiterfassung~~ → Phase 6 (fertig bzw. naechster Plan)
7. ~~Duplikate, Gruppen, Ausgaben~~ → v2

---

## 6. ZAHLEN (final)

| Kategorie | 🟢 v1 | v2 | 🔴 Offen | Frontend |
|-----------|-------|-----|---------|----------|
| Navigation (8) | 3 | 0 | 1 | 2 |
| Dashboard (11) | 2 | 0 | 1 | 0 |
| Projekte (11) | **11** | 0 | 0 | 0 |
| Meetings (16) | **15** | 0 | 0 | 0 |
| Kontakte (10) | 6 | 2 | 0 | 0 |
| Dokumente (7) | **7** | 0 | 0 | 0 |
| E-Mail (5) | **5** | 0 | 0 | 0 |
| Kalender (9) | **9** | 0 | 0 | 0 |
| Team & HR (5) | 3 | 0 | 2 | 0 |
| Buchhaltung (6) | 5 | 1 | 0 | 0 |
| Personalisierung (9) | 1 | 0 | 1 | 0 |
| Sicherheit (8) | 7 | 0 | 1 | 0 |
| Widgets (8) | 1 | 0 | 0 | 0 |
| **GESAMT (103)** | **~75** | **3** | **~5** | **2** |

**Von 103 Features: 75 in v1 eingeplant, 3 auf v2, 2 Frontend-only, 5 Kleinkram offen.**
**Ergebnis: 97% der Features haben einen klaren Platz.**
