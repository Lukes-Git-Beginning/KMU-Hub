# Feature-Brainstorm: KMU Hub Desktop

> **Status:** Review abgeschlossen (2026-02-08, Darien)
> **Ergebnis:** 103 Features bewertet, 101 JA, 2 NEIN
> **Naechster Schritt:** Design umsetzen + Feature-Liste fuer Luke erstellen

---

## Uebergreifende Design-Prinzipien (aus Review)

Diese Punkte gelten fuer ALLE Screens und muessen ueberall beruecksichtigt werden:

### 1. Rollenbasierte Ansichten
- **Admin/Teamleiter:** Sieht Einstellungen, Berechtigungen, Mitarbeiterverwaltung, Projekt-Config
- **Projektleiter:** Sieht Projekt-Einstellungen fuer eigene Projekte, aber nicht firmenweite Settings
- **Mitarbeiter:** Sieht nur was fuer ihn relevant ist, keine Admin-Panels
- Jeder Screen muss fuer verschiedene Rollen durchdacht werden

### 2. Berechtigungssystem (zieht sich durch alles)
- Dateien haben Freigabestufen (projektbezogen, teamweit, oeffentlich)
- Admin steuert wer was sehen/teilen darf
- Dateien koennen nicht an Personen ohne Freigabe geschickt werden
- Gilt fuer: Dateien, Projekte, Meetings, Chat, Kontakte

### 3. Widgets kontextbewusst platzieren
- **Fuer alle** (rollenunabhaengig): Taschenrechner, Notizen, TimeTracker, Pomodoro, Lesezeichen
- **Modulgebunden:** Buchhaltungs-Zahlen nur fuer Buchhaltungsnutzer, Projekt-Stats nur fuer Projektnutzer
- Wichtige Funktionen muessen fuer jeden erreichbar sein

### 4. Anpassbarkeit
- User soll Sidebar selbst einrichten koennen
- Quick-Actions anpassbar (eigene Aktionen hinzufuegen)
- Ein-/ausklappbare Bereiche statt feste Layouts

---

## Legende

| Symbol | Bedeutung |
|--------|-----------|
| JA | Feature wird designed |
| NEIN | Feature wird nicht umgesetzt |
| Luke | Backend existiert bereits |
| Nur Design | Wir bauen UI-Shell, Backend muss Luke noch machen |
| Aufwaendig | Technisch komplex — Luke frueh informieren |

---

## A. NAVIGATION & SHELL

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| A1 | Globale Suche | Ueber alle Module suchen | **JA** | Nur Design | Mittel | Eigenes Suchfenster von Leiste, Ergebnisse nach Typ (Projekt, Kontakt, Mail), mit Filtern |
| A2 | Tastaturkuerzel-System | Shortcuts fuer Navigation | **JA** | Klaeren mit Luke | Mittel | User soll eigene Shortcuts definieren koennen. Luke baut Logik? |
| A3 | Breadcrumb-Navigation | Pfad-Anzeige | **JA** | Nur Design | Easy | Erstmal nur fuer Datei-Ablage |
| A4 | Quick-Actions Palette | Schnellaktionen-Menue | **JA** | Nur Design | Mittel | Als hochklappbare Leiste vom Desk-Rand (Schublade unten im Eck). Auch per Tastenkuerzel |
| A5 | Favoriten/Pinned Items | Schnellzugriff auf wichtige Items | **JA** | Nur Design | Mittel | Kontakte, Projekte etc. Sidebar anpassbar, Quick-Actions anpassbar — User richtet sich alles selbst ein |
| A6 | Recent Items | Zuletzt besuchte Seiten | **JA** | Nur Design | Easy | Bei Dateien, Kontakten etc. — nicht ueberall (z.B. nicht bei Meetings) |
| A7 | Multi-Tab Interface | Mehrere Module in Tabs | **JA** | Klaeren mit Luke | Mittel | Wahrscheinlich Luke-Bereich, wir designen nur die Tab-Leiste |
| A8 | Sidebar Mini-Kalender | Monatskalender in Sidebar | **NEIN** | — | — | Erstmal nicht noetig |
| A9 | Naechste Termine | Kommende Termine anzeigen | **JA** | Nur Design | Easy | Aufklappbares Panel wie der Notiz-Bereich, nicht fest in Sidebar |

---

## B. DASHBOARD

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| B1 | Begruessung (zeitbasiert) | "Guten Morgen, Darien" | **JA** | Nur Design | Easy | — |
| B2 | Modul-Grid | 6 Hauptmodule als Karten mit Stats | **JA** | Teilweise Luke | Easy | — |
| B3 | Alerts/Warnungen | Deadlines, offene Tasks | **JA** | Nur Design | Easy | — |
| B4 | Notifications Feed (Tabs) | Mails, Kalender, Nachrichten | **JA** | Teilweise Luke | Mittel | — |
| B5 | Letzte Aktivitaeten | Activity Feed mit Avatars | **JA** | Nur Design | Easy | — |
| B6 | Quick Stats | Progress Bars | **JA** | Nur Design | Easy | — |
| B7 | Support CTA | "Brauchen Sie Hilfe?" | **JA** | Nur Design | Easy | — |
| B8 | Wetter-Widget | Aktuelles Wetter | **JA** | Nur Design | Easy | Externe API noetig |
| B9 | Tagesplan-Uebersicht | Heutige Termine + Tasks | **JA** | Nur Design | Mittel | — |
| B10 | Team-Status | Wer ist online/abwesend | **JA** | Nur Design | Mittel | Braucht WebSocket/Presence |
| B11 | Anpassbares Dashboard | Widgets verschieben | **JA** | Luke | Mittel | Luke hat Grid-System, wir stylen |

---

## C. PROJEKTE & AUFGABEN

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| C1 | Projekt-Grid-View | Karten mit Progress, Team | **JA** | Teilweise Luke | Easy | — |
| C2 | Projekt-Kanban-View | Drag & Drop Spalten | **JA** | Nur Design | Mittel | — |
| C3 | Projekt-Gantt-Chart | Zeitstrahl mit Abhaengigkeiten | **JA** | Nur Design | **Aufwaendig** | Library noetig, Luke frueh informieren |
| C4 | Projekt-Detail-Seite | Alle Infos auf einen Blick | **JA** | Teilweise Luke | Mittel | — |
| C5 | Task-Listen (gruppiert) | Sortierbar nach Projekt/Prio/Datum | **JA** | Teilweise Luke | Mittel | — |
| C6 | Task-Board (Kanban) | Drag & Drop fuer einzelne Tasks | **JA** | Nur Design | Mittel | — |
| C7 | Sub-Tasks | Verschachtelte Aufgaben/Checklisten | **JA** | Nur Design | Mittel | — |
| C8 | Zeiterfassung pro Task | Timer pro Aufgabe | **JA** | Nur Design | Mittel | Verknuepft mit TimeTracker (M1) |
| C9 | Custom Fields | Benutzerdefinierte Felder | **JA** | Nur Design | **Aufwaendig** | Braucht flexibles DB-Schema, eher Phase 2 |
| C10 | Vorlagen | Projekt-/Task-Templates | **JA** | Nur Design | Mittel | Als Basis beim Erstellen waehlbar, danach frei anpassbar (kopieren + bearbeiten) |
| C11 | Abhaengigkeiten | Task A vor Task B | **JA** | Nur Design | **Aufwaendig** | Zusammen mit C3 (Gantt), Pfeile zwischen Tasks |

---

## D. MEETINGS & KOMMUNIKATION

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| D1 | Meeting-Uebersicht | Live/Geplant/Vergangen | **JA** | Nur Design | Mittel | — |
| D2 | Meeting-Detail (Agenda) | Agenda, Teilnehmer, Link | **JA** | Nur Design | Mittel | — |
| D3 | Meeting-Notizen | Echtzeit-Notizen | **JA** | Nur Design | Mittel | — |
| D4 | Meeting-ToDos | Aufgaben aus Meeting | **JA** | Nur Design | Easy | — |
| D5 | Video-Call UI | Video + Steuerung | **JA** | Nur Design | Mittel | LiveKit als Basis |
| D6 | Audio-Call UI | Avatar-Kreise, Mute | **JA** | Nur Design | Mittel | — |
| D7 | Bildschirmfreigabe | Screen Sharing | **JA** | Nur Design | Mittel | — |
| D8 | Whiteboard | Zeichenflaeche | **JA** | Nur Design | **Aufwaendig** | Braucht eigene Canvas-Library |
| D9 | Meeting-Aufzeichnung | Recording | **JA** | Nur Design | Mittel | LiveKit unterstuetzt das |
| D10 | Chat (3-Panel) | Channels, Messages, Threads | **JA** | Luke | Easy | Luke hat Backend + UI gebaut, wir restylen |
| D11 | Direktnachrichten | 1:1 Chat | **JA** | Luke | Easy | — |
| D12 | Reaktionen (Emojis) | Auf Nachrichten reagieren | **JA** | Nur Design | Easy | — |
| D13 | @Mentions | Personen taggen | **JA** | Teilweise Luke | Easy | — |
| D14 | Datei-Sharing im Chat | Bilder/Docs im Chat teilen | **JA** | Teilweise Luke | Mittel | **MIT BERECHTIGUNGSSYSTEM:** Dateien haben Freigabestufen, Admin steuert wer teilen darf, keine Weiterleitung an Personen ohne Berechtigung |
| D15 | Anruf aus Chat | Direkt-Call Button | **JA** | Nur Design | Easy | — |
| D16 | Meeting-Raeume | Virtuelle Drop-in Raeume | **JA** | Nur Design | Mittel | — |

---

## E. KONTAKTE & CRM

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| E1 | Kontaktliste mit Filtern | Suche, Sortierung, Tags | **JA** | Teilweise Luke | Mittel | — |
| E2 | Kontakt-Detail | Alle Infos, Deals, Aktivitaeten | **JA** | Teilweise Luke | Mittel | — |
| E3 | Firmen-Verwaltung | Firmen + zugehoerige Kontakte | **JA** | Luke | Easy | — |
| E4 | Deal-Pipeline | Kanban fuer Sales | **JA** | Luke | Easy | — |
| E5 | Aktivitaeten-Log | Anrufe, E-Mails, Meetings pro Kontakt | **JA** | Nur Design | Mittel | — |
| E6 | Kontakt-Import/Export | CSV Import, vCard Export | **JA** | Nur Design | Mittel | — |
| E7 | Duplikat-Erkennung | Doppelte Kontakte finden | **JA** | Nur Design | Mittel | — |
| E8 | Tags/Labels | Kategorisierung (Kunde, Partner, Lead) | **JA** | Nur Design | Easy | — |
| E9 | Kontakt-Gruppen | Gruppen fuer Rundmails | **JA** | Nur Design | Mittel | — |
| E10 | Zwei-Ebenen-Kontaktdatenbank | Firma + Persoenlich | **JA** | Nur Design | Mittel | **NEU:** Zentrale Firmendatenbank (alle sehen sie, Admin pflegt) + persoenliche Kontakte (nur fuer dich). Neue Mitarbeiter haben sofort Zugriff auf Firmenkontakte. Kontakte aus Firmendatenbank koennen in persoenliche verknuepft werden. |

---

## F. DOKUMENTE & DATEIEN

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| F1 | Datei-Browser | Ordnerstruktur mit Farbcodes | **JA** | Nur Design | Mittel | — |
| F2 | Drag & Drop Upload | Dateien hochladen | **JA** | Nur Design | Easy | — |
| F3 | Vorschau | PDF, Bilder, Videos inline | **JA** | Nur Design | Mittel | — |
| F4 | Versionierung | Aeltere Versionen sehen | **JA** | Nur Design | Mittel | — |
| F5 | Freigabe/Sharing | Mit Team oder extern teilen | **JA** | Nur Design | Mittel | Verknuepft mit uebergreifendem Berechtigungssystem |
| F6 | Suche in Dokumenten | Volltext-Suche | **JA** | Nur Design | **Aufwaendig** | Braucht Indexing-Service |
| F7 | Tags | Dokumente taggen | **JA** | Nur Design | Easy | — |

---

## G. E-MAIL

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| G1 | E-Mail-Postfach | Inbox, Gesendet, Entwuerfe | **JA** | Nur Design | Mittel | — |
| G2 | E-Mail-Verfassen | Rich-Text Editor, Anhaenge | **JA** | Nur Design | Mittel | — |
| G3 | CRM-Verknuepfung | Mails auto zu Kontakten | **JA** | Nur Design | Mittel | — |
| G4 | E-Mail-Vorlagen | Templates fuer Mails | **JA** | Nur Design | Easy | — |
| G5 | Signaturen | Mehrere verwalten | **JA** | Nur Design | Easy | — |

---

## H. KALENDER

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| H1 | Monats/Wochen/Tag-Ansicht | Verschiedene Views | **JA** | Nur Design | Mittel | — |
| H2 | Termin erstellen | Datum, Zeit, Teilnehmer | **JA** | Nur Design | Mittel | — |
| H3 | Wiederkehrende Termine | Taeglich/woechentlich/monatlich | **JA** | Nur Design | Mittel | — |
| H4 | Raumbuchung | Meeting-Raeume reservieren | **JA** | Nur Design | Mittel | — |
| H5 | DACH-Feiertage | CH/DE/AT automatisch | **JA** | Nur Design | Easy | — |
| H6 | Geteilte Kalender | Team-/Projekt-Kalender | **JA** | Nur Design | Mittel | — |
| H7 | Verfuegbarkeitsanzeige | Frei/besetzt | **JA** | Nur Design | Mittel | — |
| H8 | CalDAV/iCal Sync | Google, Outlook anbinden | **JA** | Nur Design | **Aufwaendig** | Externe API-Integrationen |
| H9 | Externe Plattform-Integration | Teams, Slack etc. unified | **JA** | Nur Design | **Aufwaendig** | **NEU:** Nachrichten von/zu Teams/Slack/etc. in KMU Hub. Badge zeigt Herkunft. Antworten direkt aus App. Braucht API-Keys, Webhooks, evtl. Microsoft-Zertifizierung. KILLER-USP! |

---

## I. TEAM & HR

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| I1 | Team-Uebersicht | Alle Mitarbeiter + Rollen | **JA** | Teilweise Luke | Easy | — |
| I2 | Online-Status | Verfuegbar/abwesend/in Meeting | **JA** | Nur Design | Mittel | Braucht Presence-System |
| I3 | Abwesenheitsverwaltung | Urlaub/Krankheit beantragen | **JA** | Nur Design | Mittel | — |
| I4 | Organigramm | Visuelle Teamstruktur | **JA** | Nur Design | Mittel | — |
| I5 | Arbeitsinteressen/Wuensche | Mitarbeiter traegt Interessen ein | **JA** | Nur Design | Easy | **GEAENDERT:** Statt Skill-Matrix: Mitarbeiter traegt eigene Arbeitsinteressen ein ("Mache gerne Design", "Interesse an Kundenkontakt"). Admin sieht es bei Aufgabenzuweisung. Einfache Profil-Felder + Tags. |

---

## J. BUCHHALTUNG & FINANZEN

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| J1 | Bexio-Integration | Automatischer Daten-Sync | **JA** | Nur Design | **Aufwaendig** | Externe API |
| J2 | Abacus-Integration | Automatischer Daten-Sync | **JA** | Nur Design | **Aufwaendig** | Externe API |
| J3 | Run my Accounts | Automatischer Daten-Sync | **JA** | Nur Design | **Aufwaendig** | Externe API |
| J4 | Rechnungserstellung | Angebote -> Rechnungen | **JA** | Nur Design | Mittel | — |
| J5 | Ausgabenverwaltung | Belege erfassen | **JA** | Nur Design | Mittel | — |
| J6 | Dashboard-Zahlen | Umsatz, Cashflow | **JA** | Nur Design | Mittel | — |

**Ansatz:** Zweigleisig — wer schon Bexio/Abacus/RmA nutzt bekommt Integration. Wer nichts hat bekommt einfache eingebaute Buchhaltung (Rechnungen, Ausgaben, Uebersicht) die fuer kleine KMUs reicht. Nicht SAP, sondern "schnell Rechnung schreiben und sehen was reinkam".

---

## K. PERSONALISIERUNG & UX

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| K1 | Arbeitsprofile | Mehrere Kontexte | **JA** | Nur Design | Mittel | — |
| K2 | Theme: Arbeitsplatz | Schreibtisch, Deko, Blur | **JA** | Vorhanden (D1) | Easy | Basis in D1 gebaut |
| K3 | Theme: Minimal | Frosted Glass | **JA** | Nur Design | Mittel | — |
| K4 | Deko verstellbar | Pflanzen, Fotos, Items | **JA** | Nur Design | Mittel | — |
| K5 | Sprache wechselbar | DE/FR/IT/EN | **JA** | Nur Design | Mittel | i18n System noetig |
| K6 | Onboarding-Wizard | Erste Schritte | **JA** | Nur Design | Mittel | — |
| K7 | Tastaturkuerzel-Hilfe | Shortcut-Overlay | **JA** | Nur Design | Easy | Gehoert zu A2 |
| K8 | Benachrichtigungs-Praeferenzen | Pro Modul ein/aus | **JA** | Nur Design | Easy | — |
| K9 | Schriftgroesse anpassbar | Accessibility | **JA** | Nur Design | Easy | Verschiedene Groessen in Einstellungen waehlbar |
| K10 | Kompakte/Komfortable Ansicht | Spacing-Varianten | **NEIN** | — | — | Loesen wir ueber ein-/ausklappbare Bereiche statt globaler Setting |

---

## L. SYSTEM & SICHERHEIT

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| L1 | Offline-Modus | App funktioniert ohne Internet | **JA** | Luke | Easy | Bereits gebaut |
| L2 | Offline-Banner | "Keine Verbindung" | **JA** | Luke | Easy | Bereits gebaut |
| L3 | Auto-Save | Formulare zwischenspeichern | **JA** | Nur Design | Mittel | — |
| L4 | Vault/Sicherheit | Verschluesselung | **JA** | Nur Design | Mittel | — |
| L5 | 2FA | Zwei-Faktor-Auth | **JA** | Nur Design | Mittel | Reines Backend, wir designen Settings-Screen |
| L6 | Session-Verwaltung | Aktive Sessions sehen | **JA** | Nur Design | Easy | Settings-Screen |
| L7 | Audit-Log | Wer hat was wann geaendert | **JA** | Nur Design | Mittel | Admin-Feature |
| L8 | Daten-Export (DSGVO) | Eigene Daten exportieren | **JA** | Nur Design | Easy | EU-Pflicht: Button in Einstellungen, User kann alle eigenen Daten herunterladen |

---

## M. WIDGETS & UTILITIES

| # | Feature | Beschreibung | Status | Backend | Schwierigkeit | Notizen |
|---|---------|-------------|--------|---------|---------------|---------|
| M1 | TimeTracker | Zeiterfassung + Projekt | **JA** | Nur Design | Mittel | Verknuepft mit C8 |
| M2 | Daily Planner | Tagesplan im Header | **JA** | Nur Design | Mittel | — |
| M3 | Help Widget | Support-Chat + Hilfe | **JA** | Nur Design | Mittel | — |
| M4 | Notification Center | Bell + Dropdown + Pinning | **JA** | Teilweise Luke | Easy | Luke hat NotificationBell |
| M5 | Pomodoro-Timer | 25min/5min Zyklus | **JA** | Nur Design | Easy | Fuer alle verfuegbar (rollenunabhaengig) |
| M6 | Notiz-Widget | Sticky-Notes | **JA** | Nur Design | Easy | Fuer alle verfuegbar |
| M7 | Lesezeichen/Links | Links sammeln | **JA** | Nur Design | Easy | Fuer alle verfuegbar |
| M8 | Rechner | Taschenrechner | **JA** | Nur Design | Easy | Fuer alle verfuegbar |

---

## Zusammenfassung

| Kategorie | Total | JA | NEIN | Neu hinzugefuegt |
|-----------|-------|-----|------|-------------------|
| A. Navigation & Shell | 9 | 8 | 1 | — |
| B. Dashboard | 11 | 11 | 0 | — |
| C. Projekte & Aufgaben | 11 | 11 | 0 | — |
| D. Meetings & Kommunikation | 16 | 16 | 0 | — |
| E. Kontakte & CRM | 10 | 10 | 0 | E10 (Zwei-Ebenen-DB) |
| F. Dokumente & Dateien | 7 | 7 | 0 | — |
| G. E-Mail | 5 | 5 | 0 | — |
| H. Kalender | 9 | 9 | 0 | H9 (Plattform-Integration) |
| I. Team & HR | 5 | 5 | 0 | I5 geaendert |
| J. Buchhaltung & Finanzen | 6 | 6 | 0 | — |
| K. Personalisierung & UX | 10 | 9 | 1 | — |
| L. System & Sicherheit | 8 | 8 | 0 | — |
| M. Widgets & Utilities | 8 | 8 | 0 | — |
| **GESAMT** | **105** | **103** | **2** | **+3 neue Features** |

---

## Technisch aufwaendige Features (Luke frueh informieren!)

| # | Feature | Warum aufwaendig |
|---|---------|-----------------|
| C3 | Gantt-Chart | Braucht spezialisierte Library, komplexe Interaktion |
| C9 | Custom Fields | Flexibles Datenbankschema, dynamische UI |
| C11 | Task-Abhaengigkeiten | Zusammen mit Gantt, Graph-Logik |
| D8 | Whiteboard | Eigene Canvas-Library, Echtzeit-Sync |
| F6 | Volltext-Suche in Docs | Indexing-Service noetig |
| H8 | CalDAV/iCal Sync | Externe Kalender-APIs (Google, Outlook) |
| H9 | Teams/Slack Integration | API-Keys, Webhooks, evtl. Microsoft-Zertifizierung |
| J1-J3 | Buchhaltungs-Integrationen | Externe APIs (Bexio, Abacus, Run my Accounts) |

---

## Features die Luke bereits gebaut hat

| # | Feature | Was existiert |
|---|---------|--------------|
| B11 | Anpassbares Dashboard | Grid-System |
| D10 | Chat (3-Panel) | Backend + UI |
| D11 | Direktnachrichten | Backend |
| E3 | Firmen-Verwaltung | Backend + UI |
| E4 | Deal-Pipeline | Backend + UI |
| L1 | Offline-Modus | Funktioniert |
| L2 | Offline-Banner | Funktioniert |
| M4 | Notification Center | NotificationBell |
