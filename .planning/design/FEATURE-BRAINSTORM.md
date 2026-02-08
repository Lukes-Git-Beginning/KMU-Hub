# Feature-Brainstorm: KMU Hub Desktop

> Darien: Bitte geh die Liste durch und markiere mit JA/NEIN ob wir das
> im Design-Plan beruecksichtigen sollen. Kommentare willkommen.
> Bearbeite diese Datei direkt oder sag mir Bescheid.

---

## A. NAVIGATION & SHELL

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| A1 | Globale Suche | Ueber alle Module suchen (Projekte, Kontakte, Docs, etc.) | Ja |
| A2 | Tastaturkuerzel-System | Ctrl+K fuer Suche, Shortcuts fuer Navigation | Nein |
| A3 | Breadcrumb-Navigation | Pfad-Anzeige (Dashboard > Projekte > Projekt X) | Nein |
| A4 | Quick-Actions Palette | Cmd+K Palette: "Neues Projekt", "Kontakt anlegen", etc. | Nein |
| A5 | Favoriten/Pinned Items | Haeufig genutzte Projekte/Kontakte in Sidebar pinnen | Nein |
| A6 | Recent Items | Zuletzt besuchte Seiten/Dokumente schnell wieder oeffnen | Nein |
| A7 | Multi-Tab Interface | Mehrere Module gleichzeitig in Tabs offen haben | Nein |
| A8 | Sidebar Mini-Kalender | Kleiner Monatskalender in der Sidebar (wie im Bild) | Ja (Bild) |
| A9 | Sidebar Naechste Termine | Kommende 3-5 Termine direkt in Sidebar | Ja (Bild) |

---

## B. DASHBOARD

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| B1 | Begruessung (zeitbasiert) | "Guten Morgen, Darien" | Ja |
| B2 | Modul-Grid | 6 Hauptmodule als Karten mit Stats | Ja |
| B3 | Alerts/Warnungen | Deadlines, offene Tasks, System-Hinweise | Ja |
| B4 | Notifications Feed (Tabs) | Mails, Kalender, Nachrichten, Projekte, Aufgaben | Ja |
| B5 | Letzte Aktivitaeten | Activity Feed mit Avatars und Zeitstempeln | Ja |
| B6 | Quick Stats | Progress Bars (Aufgaben, Projekte) | Ja |
| B7 | Support CTA | "Brauchen Sie Hilfe?" Box | Ja |
| B8 | Wetter-Widget | Aktuelles Wetter am Standort | Nein |
| B9 | Tagesplan-Uebersicht | Heutige Termine + Tasks auf einen Blick | Teilweise |
| B10 | Team-Status | Wer ist online/abwesend/in Meeting | Nein |
| B11 | Anpassbares Dashboard | Widgets verschieben/hinzufuegen/entfernen (Lukes Grid) | Ja (Luke) |

---

## C. PROJEKTE & AUFGABEN

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| C1 | Projekt-Grid-View | Karten mit Progress, Team, Deadlines | Ja |
| C2 | Projekt-Kanban-View | Spalten: To Do, In Progress, Review, Done | Ja (erwähnt) |
| C3 | Projekt-Gantt-Chart | Zeitstrahl-Ansicht mit Abhaengigkeiten | Ja (erwähnt) |
| C4 | Projekt-Detail-Seite | Alle Infos, Team, Dateien, Aktivitaeten | Ja |
| C5 | Task-Listen (gruppiert) | Nach Zeit, Prioritaet, Projekt gruppierbar | Ja |
| C6 | Task-Board (Kanban) | Drag & Drop zwischen Status-Spalten | Nein |
| C7 | Sub-Tasks | Verschachtelte Aufgaben (Checklisten) | Nein |
| C8 | Zeiterfassung pro Task | Timer starten/stoppen auf einzelne Aufgaben | Ja (TimeTracker) |
| C9 | Custom Fields | Benutzerdefinierte Felder pro Projekt/Task | Nein |
| C10 | Vorlagen | Projekt-/Task-Vorlagen fuer wiederkehrende Arbeit | Nein |
| C11 | Abhaengigkeiten | Task A muss vor Task B fertig sein | Nein |

---

## D. MEETINGS & KOMMUNIKATION

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| D1 | Meeting-Uebersicht | Live/Geplant/Vergangen mit Filtern | Ja |
| D2 | Meeting-Detail (Agenda) | Agenda, Teilnehmer, Videokonferenz-Link | Ja |
| D3 | Meeting-Notizen | Echtzeit-Notizen waehrend Meeting (Tabs) | Ja (Bild) |
| D4 | Meeting-ToDos | Aufgaben direkt aus Meeting erstellen | Ja (Bild) |
| D5 | Video-Call UI | Haupt-Video + Thumbnails, Steuerungsleiste | Nein (zu planen) |
| D6 | Audio-Call UI | Avatar-Kreise, Mute, Lautsprecher | Nein (zu planen) |
| D7 | Bildschirmfreigabe | Screen Sharing Ansicht | Nein (zu planen) |
| D8 | Whiteboard | Zeichenflaeche, sichtbar fuer alle (Luke Backend) | Nein (zu planen) |
| D9 | Meeting-Aufzeichnung | Recording starten/stoppen, spaeter abspielen | Ja (erwähnt) |
| D10 | Chat (3-Panel) | Channels, Messages, Threads | Ja (Luke gebaut) |
| D11 | Direktnachrichten | 1:1 Chat mit Kontakten | Ja |
| D12 | Reaktionen (Emojis) | Auf Nachrichten reagieren | Nein |
| D13 | @Mentions | Personen in Nachrichten taggen | Ja (erwähnt) |
| D14 | Datei-Sharing im Chat | Bilder, Docs direkt im Chat teilen | Ja (erwähnt) |
| D15 | Anruf aus Chat | Direkt-Call Button in Einzelchats | Nein (zu planen) |
| D16 | Meeting-Raeume | Virtuelle Raeume die immer offen sind (Drop-in) | Teilweise |

---

## E. KONTAKTE & CRM

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| E1 | Kontaktliste mit Filtern | Suche, Sortierung, Tags | Ja |
| E2 | Kontakt-Detail | Alle Infos, Aktivitaeten, verknuepfte Deals | Ja |
| E3 | Firmen-Verwaltung | Firmen mit zugehoerigen Kontakten | Ja (Luke) |
| E4 | Deal-Pipeline | Kanban fuer Sales-Prozess | Ja (Luke) |
| E5 | Aktivitaeten-Log | Anrufe, E-Mails, Meetings pro Kontakt | Ja |
| E6 | Kontakt-Import/Export | CSV Import, vCard Export | Nein |
| E7 | Duplikat-Erkennung | Automatisch doppelte Kontakte finden | Nein |
| E8 | Tags/Labels | Kontakte kategorisieren (Kunde, Partner, Lead) | Nein |
| E9 | Kontakt-Gruppen | Gruppen fuer Rundmails, Einladungen | Nein |

---

## F. DOKUMENTE & DATEIEN

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| F1 | Datei-Browser | Ordnerstruktur mit Farbcodierung | Ja |
| F2 | Drag & Drop Upload | Dateien per Drag & Drop hochladen | Nein |
| F3 | Vorschau | PDF, Bilder, Videos inline anzeigen | Nein |
| F4 | Versionierung | Aeltere Versionen eines Dokuments sehen | Ja (erwähnt) |
| F5 | Freigabe/Sharing | Dokument mit Team oder extern teilen | Nein |
| F6 | Suche in Dokumenten | Volltext-Suche in PDFs/Docs | Nein |
| F7 | Tags | Dokumente taggen fuer schnelleres Finden | Nein |

---

## G. E-MAIL

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| G1 | E-Mail-Postfach | Inbox, Gesendet, Entwuerfe, Ordner | Ja |
| G2 | E-Mail-Verfassen | Rich-Text Editor, Anhaenge | Ja (implizit) |
| G3 | CRM-Verknuepfung | E-Mails automatisch zu Kontakten zuordnen | Ja (erwähnt) |
| G4 | E-Mail-Vorlagen | Templates fuer wiederkehrende Mails | Nein |
| G5 | Signaturen | Mehrere E-Mail-Signaturen verwalten | Nein |

---

## H. KALENDER

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| H1 | Monats/Wochen/Tag-Ansicht | Verschiedene Kalender-Views | Nein (zu planen) |
| H2 | Termin erstellen | Datum, Zeit, Teilnehmer, Erinnerung | Nein (zu planen) |
| H3 | Wiederkehrende Termine | Taeglich, woechentlich, monatlich | Nein |
| H4 | Raumbuchung | Meeting-Raeume reservieren | Nein |
| H5 | DACH-Feiertage | CH/DE/AT Feiertage automatisch | Nein |
| H6 | Geteilte Kalender | Team-Kalender, Projekt-Kalender | Nein |
| H7 | Verfuegbarkeitsanzeige | Wann ist jemand frei/besetzt | Nein |
| H8 | CalDAV/iCal Sync | Externes Kalender-Sync (Google, Outlook) | Nein |

---

## I. TEAM & HR

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| I1 | Team-Uebersicht | Alle Mitarbeiter mit Rollen | Ja |
| I2 | Online-Status | Wer ist verfuegbar/abwesend/in Meeting | Nein |
| I3 | Abwesenheitsverwaltung | Urlaub, Krankheit beantragen/genehmigen | Nein |
| I4 | Organigramm | Visuelle Team-Struktur | Nein |
| I5 | Skill-Matrix | Wer kann was (fuer Projekt-Zuweisungen) | Nein |

---

## J. BUCHHALTUNG & FINANZEN

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| J1 | Bexio-Integration | Automatischer Daten-Sync | Ja |
| J2 | Abacus-Integration | Automatischer Daten-Sync | Ja |
| J3 | Run my Accounts | Automatischer Daten-Sync | Ja |
| J4 | Rechnungserstellung | Angebote → Rechnungen direkt in der App | Nein |
| J5 | Ausgabenverwaltung | Belege erfassen und kategorisieren | Nein |
| J6 | Dashboard-Zahlen | Umsatz, offene Rechnungen, Cashflow | Nein |

---

## K. PERSONALISIERUNG & UX

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| K1 | Arbeitsprofile | Mehrere Kontexte mit eigenen Configs | Ja |
| K2 | Theme: Arbeitsplatz | Schreibtisch, Deko, geblurrter Hintergrund | Ja (Konzept) |
| K3 | Theme: Minimal | Saubere Flaeche, Frosted Glass | Konzept |
| K4 | Deko verstellbar | Pflanzen, Fotos, Items auf Schreibtisch | Konzept |
| K5 | Sprache wechselbar | DE, FR, IT, EN | Ja |
| K6 | Onboarding-Wizard | Erste Schritte bei erstem Start | Ja |
| K7 | Tastaturkuerzel-Hilfe | Overlay das alle Shortcuts zeigt | Nein |
| K8 | Benachrichtigungs-Praeferenzen | Pro Modul ein/ausschaltbar | Ja |
| K9 | Schriftgroesse anpassbar | Accessibility: groessere Schrift | Nein |
| K10 | Kompakte/Komfortable Ansicht | Weniger/mehr Spacing zwischen Elementen | Nein |

---

## L. SYSTEM & SICHERHEIT

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| L1 | Offline-Modus | App funktioniert ohne Internet (Basis) | Ja (Luke) |
| L2 | Offline-Banner | "Keine Verbindung" Anzeige | Ja (Luke) |
| L3 | Auto-Save | Formulare automatisch zwischenspeichern | Nein |
| L4 | Vault/Sicherheit | Verschluesselungseinstellungen | Ja |
| L5 | 2FA | Zwei-Faktor-Authentifizierung | Nein |
| L6 | Session-Verwaltung | Aktive Sessions sehen und beenden | Nein |
| L7 | Audit-Log | Wer hat was wann geaendert | Nein |
| L8 | Daten-Export | Eigene Daten exportieren (DSGVO) | Nein |

---

## M. WIDGETS & UTILITIES

| # | Feature | Beschreibung | Im Figma? |
|---|---------|-------------|-----------|
| M1 | TimeTracker | Zeiterfassung mit Projekt-Zuordnung | Ja |
| M2 | Daily Planner | Tagesplan im Header, Prioritaeten | Ja |
| M3 | Help Widget | Support-Chat + Hilfeartikel | Ja |
| M4 | Notification Center | Bell + Dropdown + Pinning | Ja |
| M5 | Pomodoro-Timer | 25min Arbeit / 5min Pause Zyklus | Nein |
| M6 | Notiz-Widget | Schnelle Notizen (Sticky-Notes Stil) | Nein |
| M7 | Lesezeichen/Links | Externe Links sammeln und organisieren | Nein |
| M8 | Rechner | Schneller Taschenrechner | Nein |

---

**Gesamt: ~100 Features in 13 Kategorien**

Bitte markiere mit JA/NEIN welche ins Design sollen.
"Im Figma?" zeigt ob das Feature schon im Figma-Export vorhanden ist.
