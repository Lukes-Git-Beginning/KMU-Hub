# Feature-Liste fuer Luke: Backend-Anforderungen aus Design-Review

> **Von:** Darien (Design)
> **Datum:** 2026-02-08
> **Kontext:** Wir haben alle ~105 Features reviewed und 103 davon fuer das Design freigegeben.
> Hier sind die Features die Backend-Arbeit brauchen, sortiert nach Prioritaet.
> Features die "Nur Design" sind (UI-Shells) sind NICHT in dieser Liste — die bauen wir selber.

---

## DRINGEND: Technisch aufwaendige Features (frueh einplanen!)

Diese Features brauchen signifikante Backend-Arbeit und sollten frueh in die Planung:

| # | Feature | Was wird gebraucht | Warum aufwaendig |
|---|---------|-------------------|-----------------|
| H9 | **Teams/Slack Integration** | API-Anbindung an MS Teams, Slack etc. Nachrichten von/zu externen Plattformen in KMU Hub anzeigen. Badge zeigt Herkunft (Teams/Slack). User antwortet direkt aus KMU Hub. | API-Keys, Webhooks, evtl. Microsoft-Zertifizierung. **KILLER-USP fuer unser Produkt!** |
| C3 | **Gantt-Chart** | Zeitstrahl-Ansicht fuer Projekte mit Balken, Abhaengigkeiten als Pfeile | Braucht spezialisierte Frontend-Library + Backend-Datenmodell fuer Abhaengigkeiten |
| C9 | **Custom Fields** | User kann eigene Felder zu Projekten/Tasks hinzufuegen (z.B. "Budget", "Kundennummer") | Braucht flexibles Datenbankschema (EAV oder JSONB) |
| C11 | **Task-Abhaengigkeiten** | "Task A muss vor Task B fertig sein" — Graph-Logik, Pfeile im Gantt | Zusammen mit C3, braucht Zyklen-Erkennung |
| D8 | **Whiteboard** | Echtzeit-Zeichenflaeche fuer Meetings, sichtbar fuer alle Teilnehmer | Eigene Canvas-Library + Echtzeit-Sync (WebSocket/CRDT) |
| F6 | **Volltext-Suche in Dokumenten** | PDF/Doc-Inhalte durchsuchbar machen | Braucht Indexing-Service (z.B. Elasticsearch/Meilisearch) |
| H8 | **CalDAV/iCal Sync** | Google Calendar, Outlook anbinden | Externe API-Integrationen, OAuth-Flows |
| J1 | **Bexio-Integration** | Automatischer Daten-Sync mit Bexio | Bexio API, Mapping, Fehlerbehandlung |
| J2 | **Abacus-Integration** | Automatischer Daten-Sync mit Abacus | Abacus API |
| J3 | **Run my Accounts Integration** | Automatischer Daten-Sync | RmA API |

---

## Uebergreifendes System: Berechtigungen + Rollen

**Das ist das Wichtigste ueberhaupt** — zieht sich durch die gesamte App:

### Rollenbasierte Ansichten
- **Admin/Teamleiter:** Sieht Einstellungen, Berechtigungen, Mitarbeiterverwaltung, Projekt-Config
- **Projektleiter:** Sieht Projekt-Einstellungen fuer eigene Projekte, nicht firmenweite Settings
- **Mitarbeiter:** Sieht nur was relevant ist, keine Admin-Panels

### Datei-Berechtigungssystem
- Dateien haben Freigabestufen: projektbezogen, teamweit, oeffentlich
- Admin steuert wer was sehen/teilen darf
- Dateien koennen NICHT an Personen ohne Freigabe weitergeleitet werden
- Gilt fuer: Dateien, Projekte, Meetings, Chat, Kontakte

---

## Neue Features (nicht im urspruenglichen Plan)

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| E10 | **Zwei-Ebenen-Kontaktdatenbank** | Zentrale Firmendatenbank (Admin pflegt, alle sehen) + persoenliche Kontakte (nur fuer einzelnen User). Neue Mitarbeiter haben sofort Zugriff auf Firmenkontakte. Verknuepfung moeglich. | Mittel (scope-Feld pro Kontakt + Berechtigungslogik) |
| H9 | **Externe Plattform-Integration** | Unified Inbox: Teams/Slack/etc. Nachrichten in KMU Hub, mit Herkunfts-Badge. Direkt antworten. | Aufwaendig (siehe oben) |
| I5 | **Arbeitsinteressen** (statt Skill-Matrix) | Mitarbeiter traegt eigene Interessen/Wuensche ein ("Mache gerne Design"). Admin sieht es bei Aufgabenzuweisung. | Easy (Profil-Felder + Tags) |

---

## Features die du schon hast (wir restylen nur)

Wir aendern hier nur das Design, Backend bleibt:

| # | Feature | Was existiert | Was wir machen |
|---|---------|--------------|----------------|
| B11 | Anpassbares Dashboard | Grid-System | Neue Farben + Styling |
| D10 | Chat (3-Panel) | Backend + UI | Komplett restylen |
| D11 | Direktnachrichten | Backend | UI bauen |
| E3 | Firmen-Verwaltung | Backend + UI | Restylen |
| E4 | Deal-Pipeline | Backend + UI | Restylen |
| L1 | Offline-Modus | Funktioniert | — |
| L2 | Offline-Banner | Funktioniert | Evtl. restylen |
| M4 | Notification Center | NotificationBell | Erweitern (Dropdown, Pinning) |

---

## Features wo wir dich brauchen (nach Kategorie)

### Navigation
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| A1 | Globale Suche | Such-API ueber alle Module (Projekte, Kontakte, Mails, Docs) mit Typ-Filter |
| A2 | Tastaturkuerzel | Frage: Baust du die Logik oder machen wir das rein im Frontend? User soll eigene Shortcuts definieren koennen. |
| A7 | Multi-Tabs | Frage: Faellt das in deinen Bereich? Wir designen die Tab-Leiste. |

### Dashboard
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| B8 | Wetter-Widget | Externe Wetter-API anbinden |
| B10 | Team-Status | WebSocket/Presence-System (wer ist online/abwesend/in Meeting) |

### Projekte & Aufgaben
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| C2 | Kanban-View | Status-Updates via API (Drag & Drop) |
| C6 | Task-Board | Wie C2, fuer einzelne Tasks |
| C7 | Sub-Tasks | Verschachtelte Task-Struktur im Datenmodell |
| C8 | Zeiterfassung/Task | Timer-Logik + Projekt-Zuordnung |
| C10 | Vorlagen | Projekt/Task kopieren als Vorlage, beim Erstellen als Basis waehlbar |

### Meetings & Kommunikation
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| D5-D7 | Video/Audio/Screenshare | LiveKit Integration (hast du schon geplant?) |
| D9 | Meeting-Aufzeichnung | LiveKit Recording |
| D12 | Emoji-Reaktionen | Reaktionen-Tabelle pro Nachricht |
| D14 | Datei-Sharing + Berechtigungen | Freigabestufen-System (siehe oben) |
| D16 | Meeting-Raeume | Persistente Raeume die immer offen sind |

### Kontakte & CRM
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| E5 | Aktivitaeten-Log | Alle Interaktionen pro Kontakt aggregieren |
| E6 | Import/Export | CSV-Parser, vCard-Generator |
| E7 | Duplikat-Erkennung | Matching-Algorithmus (Name, Email, Telefon) |
| E8 | Tags/Labels | Tag-System (evtl. schon vorhanden?) |
| E9 | Kontakt-Gruppen | Gruppen-Tabelle, Zuordnung |

### Dokumente
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| F1 | Datei-Browser | Ordnerstruktur-API |
| F2 | Upload | File-Upload Endpoint (evtl. schon da?) |
| F4 | Versionierung | Versionshistorie pro Datei |
| F5 | Freigabe | Berechtigungssystem (siehe oben) |

### E-Mail
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| G1-G3 | Mail-System | IMAP/SMTP Integration, CRM-Verknuepfung |
| G4 | Vorlagen | Template-Storage |
| G5 | Signaturen | Signatur-Storage pro User |

### Kalender
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| H1-H4 | Kalender komplett | Kalender-Service (Termine, Recurring, Raumbuchung) |
| H5 | DACH-Feiertage | Feiertags-Datenbank CH/DE/AT |
| H6-H7 | Geteilte Kalender + Verfuegbarkeit | Team-Kalender, Frei/Besetzt-Logik |

### Team & HR
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| I2 | Online-Status | Presence-System (WebSocket) |
| I3 | Abwesenheitsverwaltung | Antrags-Workflow (beantragen → genehmigen) |
| I4 | Organigramm | Hierarchie-Daten |

### Buchhaltung
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| J4 | Rechnungserstellung | Rechnungs-Service, PDF-Generation |
| J5 | Ausgabenverwaltung | Beleg-Upload + Kategorisierung |
| J6 | Dashboard-Zahlen | Aggregierte Finanzdaten-API |

**Ansatz Buchhaltung:** Zweigleisig — wer Bexio/Abacus/RmA hat bekommt Integration. Wer nichts hat bekommt einfache eingebaute Buchhaltung (Rechnungen, Ausgaben, Uebersicht). Kein SAP, sondern "schnell Rechnung schreiben".

### System & Sicherheit
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| L3 | Auto-Save | Draft-Storage fuer Formulare |
| L4 | Vault | Verschluesselungs-Einstellungen |
| L5 | 2FA | TOTP/WebAuthn Implementation |
| L6 | Session-Verwaltung | Sessions auflisten + invalidieren |
| L7 | Audit-Log | Event-Logging (wer, was, wann) |
| L8 | DSGVO-Export | Alle User-Daten als Download (EU-Pflicht) |

### Widgets
| # | Feature | Was braucht Backend |
|---|---------|-------------------|
| M1 | TimeTracker | Timer-Service + Projekt-Zuordnung |
| M3 | Help Widget | Support-Ticket-System oder Chat |

---

## Zusammenfassung

| Kategorie | Features gesamt | Davon Backend noetig | Schon gebaut |
|-----------|----------------|---------------------|-------------|
| Navigation | 8 | 3 | 0 |
| Dashboard | 11 | 2 | 1 |
| Projekte | 11 | 8 | 0 |
| Meetings | 16 | 6 | 3 |
| Kontakte | 10 | 6 | 2 |
| Dokumente | 7 | 5 | 0 |
| E-Mail | 5 | 5 | 0 |
| Kalender | 9 | 9 | 0 |
| Team & HR | 5 | 3 | 0 |
| Buchhaltung | 6 | 6 | 0 |
| Personalisierung | 9 | 0 | 0 |
| Sicherheit | 8 | 6 | 2 |
| Widgets | 8 | 2 | 1 |
| **GESAMT** | **103** | **~61** | **9** |

**~61 Features brauchen Backend-Arbeit, 9 sind schon gebaut, ~42 sind rein Frontend/Design.**
