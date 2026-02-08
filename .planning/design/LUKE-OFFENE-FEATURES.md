# Offene Features: Was fehlt noch in Lukes Roadmap?

> **Von:** Darien (Design)
> **Fuer:** Luke (Backend)
> **Stand:** 2026-02-08
> **Grundlage:** 103 genehmigte Features aus Design-Review, verglichen mit Lukes Roadmap (Phasen 4-13)
> **Status:** BESPROCHEN — Lukes Antworten eingetragen (siehe Abschnitt am Ende)

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

**OFFEN** — Luke hat sich zu J1-J3 (Bexio/Abacus/RmA) nicht geaeussert.
Vermutlich spaeter, evtl. nach Phase 15 oder als Plugin (Phase 13).

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| J1 | 🔴 | **Bexio-Integration** | Bexio REST API Client, Daten-Mapping, Sync-Logik, Fehlerbehandlung | **Gross** |
| J2 | 🔴 | **Abacus-Integration** | Abacus API Client (aeltere API, weniger gut dokumentiert) | **Gross** |
| J3 | 🔴 | **Run my Accounts** | RmA API Client | **Gross** |

---

## 2. FEATURES DIE IN BESTEHENDEN PHASEN FEHLEN

### Phase 6 (Projekte) — Gantt + Abhaengigkeiten kommen rein

> **Luke:** Gantt + Abhaengigkeiten kommen in v1, Phase 6, als Plans 06-09 + 06-10

| # | Status | Feature | Was fehlt | Aufwand extra |
|---|--------|---------|----------|---------------|
| C3 | 🟢⭐ | **Gantt-Chart** | Zeitstrahl-View mit Balken + Abhaengigkeits-Pfeilen. Plan 06-09 | **Gross** |
| C11 | 🟢⭐ | **Task-Abhaengigkeiten** | "Task A muss vor B fertig sein". Plan 06-10 | **Gross** |
| C7 | 🟡 | **Sub-Tasks** | Verschachtelte Aufgaben (parent_task_id). Nicht explizit erwaehnt | Klein-Mittel |
| C8 | 🟡 | **Zeiterfassung pro Task** | Timer-Logik pro Task. Phase 11 hat Time-Tracking allgemein | Mittel |
| C10 | 🟡 | **Vorlagen** | Projekt/Task als Vorlage speichern. Nicht explizit erwaehnt | Mittel |

---

### Phase 10 (Finance) — Buchhaltungs-Integrationen noch offen

| # | Status | Feature | Was fehlt | Aufwand extra |
|---|--------|---------|----------|---------------|
| J5 | 🟡 | **Ausgabenverwaltung** | Belege hochladen, kategorisieren, Kosten tracken. Phase 10 hat Rechnungen aber nicht Ausgaben | Mittel |
| J1-J3 | 🔴 | **Externe Integrationen** | Bexio/Abacus/RmA — noch nicht entschieden | Gross pro Stueck |

---

### Phase 11 (HR) — jetzt Documents, HR verschoben?

Luke sagt Phase 11 wird Documents & Files. Die alte Phase 11 (HR) wird vermutlich
eine andere Nummer bekommen. Hier die HR-Features die noch offen sind:

| # | Status | Feature | Was fehlt | Aufwand extra |
|---|--------|---------|----------|---------------|
| I4 | 🟡 | **Organigramm** | Visuelle Hierarchie-Darstellung. Braucht Hierarchie-Daten | Mittel |
| I5 | 🟡 | **Arbeitsinteressen** | Profil-Felder + Tags. Nicht explizit erwaehnt | Klein |

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

| # | Status | Feature | Was Luke bauen muesste | Aufwand | Wann noetig? |
|---|--------|---------|----------------------|---------|-------------|
| A1 | 🔴⭐ | **Globale Suche** | Such-API ueber alle Module. Unified Search Endpoint | **Gross** | Noch nicht eingeplant |
| A2 | 🟢 | **Tastaturkuerzel** | Frontend-only (unser Bereich) | Mittel | — |
| A7 | 🟢 | **Multi-Tabs** | Frontend-only (unser Bereich) | Mittel | — |
| B8 | 🔴 | **Wetter-Widget** | Externe Wetter-API anbinden | Klein | Nice-to-have |

---

### CRM-Erweiterungen (ueber Phase 2 hinaus)

Luke hat sich zu diesen nicht explizit geaeussert:

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| E6 | 🔴 | **Kontakt-Import/Export** | CSV-Parser (mit Spalten-Mapping), vCard-Export | Mittel |
| E7 | 🔴 | **Duplikat-Erkennung** | Matching-Algorithmus (Name, E-Mail, Telefon), Merge-Funktion | Mittel |
| E9 | 🔴 | **Kontakt-Gruppen** | Gruppen-Tabelle + Zuordnung, fuer Rundmails | Klein |
| E10 | 🔴⭐ | **Zwei-Ebenen-Kontakte** | Zentrale Firmendatenbank + persoenliche Kontakte. Braucht `scope`-Feld + Berechtigungslogik | Mittel |
| D12 | 🔴 | **Emoji-Reaktionen im Chat** | Reaktionen-Tabelle pro Nachricht, WebSocket-Events | Klein |
| D14 | 🔴 | **Datei-Sharing + Berechtigungen** | Freigabestufen im Chat | Mittel |

---

### Personalisierung

| # | Status | Feature | Was Luke bauen muesste | Aufwand |
|---|--------|---------|----------------------|---------|
| K1 | 🔴 | **Arbeitsprofile** | Mehrere Kontexte pro User mit eigenen Configs | Mittel |

---

## 4. ZUSAMMENFASSUNG NACH LUKES ANTWORTEN

### Neue Roadmap-Struktur (Lukes Phasen, aktualisiert)

| Phase | Inhalt | Status |
|-------|--------|--------|
| 4 | Notifications + Gateway | Fertig |
| 5 | Desktop App Shell | Fast fertig (6/7 Plans) |
| 6 | Project Management + Gantt + Abhaengigkeiten (Plans 06-09, 06-10) | Geplant |
| 7 | Calendar & Scheduling | Geplant |
| 8 | Video, Voice & **Meetings** + **Presence** (erweitert!) | Geplant |
| 9 | **Security & Compliance + i18n** (NEU! War vorher Email) | Geplant |
| 10 | Email Integration (verschoben von 9 auf 10) | Geplant |
| 11 | **Documents & Files** (NEU! War vorher HR) | Geplant |
| 12 | Finance Module (verschoben) | Geplant |
| 13 | HR Module (verschoben) | Geplant |
| ... | Automation + Plugins (verschoben) | Geplant |
| 14 | **CalDAV/iCal Sync** (Mini-Phase, NEU) | Geplant |
| 15 | **Teams/Slack Integration** (Mini-Phase, NEU) | Geplant |

### Was jetzt eingeplant ist (von 🔴 auf 🟢 gewechselt)

| Feature | Wo eingeplant |
|---------|--------------|
| F1-F7 Dokumente | Phase 11 (Documents & Files) |
| D1-D4, D8, D16 Meetings | Phase 8 (Video, Voice & Meetings) |
| B10, I2 Presence | Phase 8 |
| C3 Gantt | Phase 6, Plan 06-09 |
| C11 Abhaengigkeiten | Phase 6, Plan 06-10 |
| L4-L8 Sicherheit | Phase 9 (Security & Compliance) |
| K5 i18n | Phase 9 |
| H8 CalDAV | Phase 14 (Mini-Phase) |
| H9 Teams/Slack | Phase 15 (Mini-Phase) |
| A2 Tastaturkuerzel | Frontend-only (wir) |
| A7 Multi-Tabs | Frontend-only (wir) |

### Was NOCH OFFEN ist (immer noch 🔴)

| # | Feature | Aufwand | Kommentar |
|---|---------|---------|-----------|
| J1-J3 | Bexio/Abacus/RmA Integration | Gross x3 | Luke hat sich nicht geaeussert. Evtl. spaetere Phase oder Plugin |
| A1 | Globale Suche | Gross | Noch kein Platz in Roadmap |
| E10 | Zwei-Ebenen-Kontakte | Mittel | Nicht angesprochen |
| E6 | Kontakt-Import/Export | Mittel | Nicht angesprochen |
| E7 | Duplikat-Erkennung | Mittel | Nicht angesprochen |
| E9 | Kontakt-Gruppen | Klein | Nicht angesprochen |
| D12 | Emoji-Reaktionen | Klein | Nicht angesprochen |
| D14 | Datei-Sharing + Berechtigungen | Mittel | Nicht angesprochen |
| K1 | Arbeitsprofile | Mittel | Nicht angesprochen |
| B8 | Wetter-Widget | Klein | Nice-to-have |
| L3 | Auto-Save | Klein | Nice-to-have |
| J5 | Ausgabenverwaltung | Mittel | Phase 10 erwaehnt nur Rechnungen |
| C7 | Sub-Tasks | Klein-Mittel | Phase 6 nicht explizit |
| C8 | Zeiterfassung pro Task | Mittel | Phase 6 nicht explizit |
| C10 | Vorlagen | Mittel | Phase 6 nicht explizit |
| I4 | Organigramm | Mittel | HR-Phase nicht explizit |
| I5 | Arbeitsinteressen | Klein | HR-Phase nicht explizit |

**→ ~17 Features noch ohne klaren Platz** (von urspruenglich ~41)
**→ Davon 3 grosse Brocken** (Bexio/Abacus/RmA) und **1 strategisch wichtiges** (Globale Suche)
**→ Rest sind kleinere Sachen** die vermutlich in bestehende Phasen passen

---

## 5. OFFENE FRAGEN (nach Lukes Antworten aktualisiert)

### Beantwortet

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

### Noch offen (naechstes Gespraech)

1. **Buchhaltungs-Integrationen (J1-J3):** Bexio/Abacus/RmA — wann? Eigene Phase? Plugin?
2. **Globale Suche (A1):** Wo einplanen? Cross-Module Search ist Kernfunktion
3. **Zwei-Ebenen-Kontakte (E10):** In Phase 2 (CRM) nachtragen? Oder eigene Erweiterung?
4. **CRM-Erweiterungen (E6, E7, E9):** Import/Export, Duplikate, Gruppen — in bestehende CRM-Phase?
5. **Sub-Tasks + Vorlagen (C7, C10):** Kommen die in Phase 6 mit rein?
6. **Chat-Erweiterungen (D12, D14):** Emoji-Reaktionen + Datei-Berechtigungen — Phase 3 erweitern?

---

## 6. ZAHLEN (aktualisiert)

| Kategorie | 🟢 Geplant | 🔴 Fehlt | 🟡 Teilweise |
|-----------|-----------|---------|-------------|
| Navigation (8) | 2 | 1 | 0 |
| Dashboard (11) | 2 | 1 | 0 |
| Projekte (11) | 8 | 0 | 3 |
| Meetings (16) | 13 | 2 | 0 |
| Kontakte (10) | 4 | 5 | 0 |
| Dokumente (7) | **7** | 0 | 0 |
| E-Mail (5) | 4 | 0 | 1 |
| Kalender (9) | **9** | 0 | 0 |
| Team & HR (5) | 3 | 0 | 2 |
| Buchhaltung (6) | 2 | 3 | 1 |
| Personalisierung (9) | 1 | 1 | 0 |
| Sicherheit (8) | 7 | 0 | 1 |
| Widgets (8) | 1 | 0 | 0 |
| **GESAMT (103)** | **~63** | **~13** | **~8** |

**Vorher:** ~41 fehlend → **Jetzt:** ~13 fehlend + ~8 teilweise
**Luke hat mit einer Antwort ~28 Features eingeplant.**
