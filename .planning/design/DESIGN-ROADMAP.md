# Design Roadmap: KMU Hub Desktop

## Overview

Design-Phasen fuer das KMU Hub Desktop Frontend. Jede Phase listet alle Features
aus dem Feature-Brainstorm (103 genehmigte Features) auf, damit nichts verloren geht.

**Branch:** `design/brainstorm`
**Designer:** Darien
**Uebergabe an:** Luke (main branch, GSD-Phasen)
**Figma-Referenz:** `desktop/design-reference/`
**Feature-Entscheidungen:** `.planning/design/FEATURE-BRAINSTORM.md`
**Luke-Liste:** `.planning/design/LUKE-FEATURE-LIST.md`

---

## Fortschritt

```
[##--------] 2/9 Phasen abgeschlossen
```

- [x] D1: Desk Foundation (2026-02-07)
- [x] D2: Color System & Theme (2026-02-08)
- [ ] D3: Sidebar Redesign
- [ ] D4: Header Redesign
- [ ] D5: Dashboard
- [ ] D6: Module Screens (groesste Phase — wird in Sub-Phasen aufgeteilt)
- [ ] D7: Widgets & Overlays
- [ ] D8: Desk Polish & Themes
- [ ] D9: Visual Polish & Accessibility

---

## Abhaengigkeiten

```
D1 (DONE) ──→ D2 (DONE) ──┬──→ D3 (Sidebar braucht neue Farben)
                            ├──→ D4 (Header braucht neue Farben)
                            └──→ D5 (Dashboard braucht neue Farben)
                                  └──→ D6 (Module bauen auf Dashboard-Patterns auf)
                            D7 (Widgets, nach D2 sinnvoller)
                            D8 (Desk-spezifisch, nach D6 fuer Theme-Varianten)
                            D9 (ganz am Ende, Polish ueber alles)
```

---

## D1: Desk Foundation — DONE (2026-02-07)

**Ziel:** Schreibtisch-Metapher als Grundgeruest der App

| Feature | Status |
|---------|--------|
| DeskEnvironment (Room-Scene Layout) | Fertig |
| DeskFrame (Wand + Schreibtisch-Oberflaeche) | Fertig |
| DeskDecorations (Uhr, Pflanzen, Fotos) | Fertig |
| DeskClock (Analoge Wanduhr) | Fertig |
| Theme-System (data-driven, CSS variables) | Fertig |
| Maximize-Modus (Ctrl+Shift+F) | Fertig |
| Classic Office Theme | Fertig |

---

## D2: Color System & Theme — DONE (2026-02-08)

**Ziel:** Figma-Farbpalette uebernehmen, Dark Mode, Typography

| Feature | Status |
|---------|--------|
| Warme Farbpalette Light Mode (#e8e3dd, #1e7e74) | Fertig |
| OKLCH Dark Mode (neutral grau, hue 240) | Fertig |
| 100+ CSS Custom Properties | Fertig |
| @theme inline Mapping fuer Tailwind v4 | Fertig |
| .dark Class Toggle auf html Element | Fertig |
| Typography Base Styles (h1-h4, label, button, input) | Fertig |
| Status-Farben (success, warning, error, info) | Fertig |
| File-Type-Farben (PDF, Word, Excel...) | Fertig |
| Shadow-System (micro, small, medium, large, card) | Fertig |

---

## D3: Sidebar Redesign — NAECHSTE PHASE

**Ziel:** Sidebar komplett nach Figma + Inspiration-Bild umbauen

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| — | Branding-Header | "KMU Hub" Logo + Workspace-Name + Dropdown | Klein |
| — | User-Profil-Bereich | Avatar (rund) + Name + Online-Status (gruen/rot/gelb) | Klein |
| A5 | Favoriten/Pinned Items | Anpinnbare Projekte/Kontakte in Sidebar, User richtet sich alles selbst ein | Mittel |
| — | Navigation Items | Icons + Labels fuer alle Module (Chat, Projekte, Aufgaben, Kalender, Dateien, Kontakte, Apps) | Mittel |
| — | Active State | Farbiger Hintergrund (teal) + Active Border fuer ausgewaehltes Modul | Klein |
| — | Badges | Ungelesene-Zahlen an Nav-Items (z.B. Chat: 3) | Klein |
| — | Live Indicator | Gruener Punkt bei laufenden Meetings | Klein |
| A9 | Naechste Termine | Aufklappbares Panel (wie Notiz-Bereich) mit kommenden 3-5 Terminen | Mittel |
| — | Responsive Collapse | Sidebar ein-/ausklappbar, nur Icons im collapsed State | Mittel |
| — | Sidebar Anpassbar | User kann Reihenfolge/Sichtbarkeit der Nav-Items aendern | Mittel |

---

## D4: Header Redesign

**Ziel:** Header mit Suche, Icons, Profil — minimal wie im Inspiration-Bild

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| A1 | Globale Suche | Eigenes Suchfenster von Leiste, Ergebnisse nach Typ mit Filtern | Mittel |
| A2 | Tastaturkuerzel-System | Shortcuts + User kann eigene definieren (UI-Teil) | Mittel |
| A3 | Breadcrumb-Navigation | Pfad-Anzeige, erstmal nur fuer Datei-Ablage | Klein |
| M2 | Daily Planner | Tagesplan-Widget im Header, Prioritaeten | Mittel |
| M4 | Notification Center | Glocke + Dropdown + Pinning + roter Badge | Mittel |
| K5 | Sprache wechselbar | Language Switcher (DE/FR/IT/EN) im Header/Settings | Klein |
| — | Profil-Avatar | Runder Avatar rechts mit Dropdown-Menue | Klein |
| — | Header Icons | Chat, Nachrichten, Ordner, Glocke — wie im Inspiration-Bild | Klein |
| K7 | Tastaturkuerzel-Hilfe | Overlay das alle Shortcuts zeigt (verbunden mit A2) | Klein |

---

## D5: Dashboard

**Ziel:** Hauptscreen nach Login — alle wichtigen Infos auf einen Blick

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| B1 | Begruessung | "Guten Morgen, Darien" zeitbasiert | Klein |
| B2 | Modul-Grid | 6 Hauptmodule als Karten mit Stats/Zahlen | Mittel |
| B3 | Alerts/Warnungen | Deadlines, offene Tasks, System-Hinweise | Mittel |
| B4 | Notifications Feed | Tabs: Mails, Kalender, Nachrichten, Projekte, Aufgaben | Mittel |
| B5 | Letzte Aktivitaeten | Activity Feed mit Avatars und Zeitstempeln | Mittel |
| B6 | Quick Stats | Progress Bars fuer Aufgaben und Projekte | Klein |
| B7 | Support CTA | "Brauchen Sie Hilfe?" Box | Klein |
| B8 | Wetter-Widget | Aktuelles Wetter am Standort | Klein |
| B9 | Tagesplan-Uebersicht | Heutige Termine + Tasks auf einen Blick | Mittel |
| B10 | Team-Status | Wer ist online/abwesend/in Meeting | Mittel |
| B11 | Anpassbares Dashboard | Widgets verschieben/hinzufuegen/entfernen (Luke Grid restylen) | Mittel |
| A6 | Recent Items | "Zuletzt besucht" Liste (Dateien, Kontakte — nicht Meetings) | Klein |

---

## D6: Module Screens (GROESSTE PHASE)

Wird in Sub-Phasen aufgeteilt. Jeder Screen braucht Uebersicht + Detail-Ansicht + Modals.

### D6.1: Projekte & Aufgaben

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| C1 | Projekt-Grid-View | Karten mit Progress, Team, Deadlines | Mittel |
| C2 | Projekt-Kanban-View | Drag & Drop Spalten (To Do, In Progress, Review, Done) | Mittel |
| C3 | Projekt-Gantt-Chart | Zeitstrahl mit Abhaengigkeiten (braucht Library) | Gross |
| C4 | Projekt-Detail-Seite | Alle Infos, Team, Dateien, Aktivitaeten | Gross |
| C5 | Task-Listen (gruppiert) | Sortierbar nach Projekt/Prioritaet/Datum | Mittel |
| C6 | Task-Board (Kanban) | Drag & Drop fuer einzelne Tasks | Mittel |
| C7 | Sub-Tasks | Verschachtelte Aufgaben/Checklisten | Mittel |
| C8 | Zeiterfassung pro Task | Timer pro Aufgabe (verknuepft mit TimeTracker M1) | Klein |
| C9 | Custom Fields | Benutzerdefinierte Felder pro Projekt/Task | Mittel |
| C10 | Vorlagen | Templates als Basis beim Erstellen, dann frei anpassbar | Mittel |
| C11 | Abhaengigkeiten | Task A vor Task B, Pfeile im Gantt | Mittel |

### D6.2: Meetings & Kommunikation

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| D1 | Meeting-Uebersicht | Live/Geplant/Vergangen mit Filtern | Mittel |
| D2 | Meeting-Detail | Agenda, Teilnehmer, Link (wie im Inspiration-Bild!) | Mittel |
| D3 | Meeting-Notizen | Echtzeit-Notizen mit Tabs (Notizen/To-Do/Info) | Mittel |
| D4 | Meeting-ToDos | Aufgaben direkt aus Meeting erstellen | Klein |
| D5 | Video-Call UI | Haupt-Video + Thumbnails + Steuerungsleiste | Gross |
| D6 | Audio-Call UI | Avatar-Kreise, Mute, Lautsprecher | Mittel |
| D7 | Bildschirmfreigabe | Screen Sharing Ansicht | Mittel |
| D8 | Whiteboard | Zeichenflaeche (Design-Entwurf, Backend = Luke) | Gross |
| D9 | Meeting-Aufzeichnung | Recording starten/stoppen UI | Klein |
| D15 | Anruf aus Chat | Direkt-Call Button in Einzelchats | Klein |
| D16 | Meeting-Raeume | Virtuelle Drop-in Raeume UI | Mittel |

### D6.3: Chat (Restyling)

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| D10 | Chat (3-Panel) | Lukes Chat restylen mit neuen Farben | Mittel |
| D11 | Direktnachrichten | 1:1 Chat UI | Klein |
| D12 | Reaktionen (Emojis) | Emoji-Picker + Reaktions-Anzeige | Klein |
| D13 | @Mentions | Autocomplete beim Tippen von @ | Mittel |
| D14 | Datei-Sharing + Berechtigungen | Upload-UI + Freigabestufen-Anzeige | Mittel |

### D6.4: Kontakte & CRM (Restyling + Neue Screens)

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| E1 | Kontaktliste | Filter, Suche, Sortierung, Tags | Mittel |
| E2 | Kontakt-Detail | Alle Infos, verknuepfte Deals, Aktivitaeten | Mittel |
| E3 | Firmen-Verwaltung | Lukes Screen restylen | Klein |
| E4 | Deal-Pipeline | Lukes Kanban restylen | Klein |
| E5 | Aktivitaeten-Log | Timeline pro Kontakt (Anrufe, Mails, Meetings) | Mittel |
| E6 | Kontakt-Import/Export | Upload-Dialog + Mapping-UI | Mittel |
| E7 | Duplikat-Erkennung | Merge-Dialog wenn Duplikat gefunden | Mittel |
| E8 | Tags/Labels | Tag-Chips + Vergabe-UI | Klein |
| E9 | Kontakt-Gruppen | Gruppen-Verwaltung fuer Rundmails | Mittel |
| E10 | Zwei-Ebenen-Kontakte | Umschalter Firma/Persoenlich, getrennte Listen | Mittel |

### D6.5: Dokumente & Dateien

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| F1 | Datei-Browser | Ordnerstruktur mit Farbcodes + Breadcrumbs (A3) | Mittel |
| F2 | Drag & Drop Upload | Upload-Zone + Progress-Anzeige | Klein |
| F3 | Vorschau | Inline-Viewer fuer PDF, Bilder, Videos | Mittel |
| F4 | Versionierung | Versions-Liste + Vergleich | Mittel |
| F5 | Freigabe/Sharing | Berechtigungsstufen-Dialog | Mittel |
| F6 | Volltext-Suche | Such-UI mit Ergebnis-Highlighting | Mittel |
| F7 | Tags | Tag-Chips an Dokumenten | Klein |

### D6.6: E-Mail

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| G1 | E-Mail-Postfach | Inbox mit Ordnerstruktur links, Liste mitte, Preview rechts | Gross |
| G2 | E-Mail-Verfassen | Rich-Text Editor + Anhaenge | Gross |
| G3 | CRM-Verknuepfung | Kontakt-Badge in Mails + Auto-Zuordnung | Mittel |
| G4 | E-Mail-Vorlagen | Template-Auswahl beim Verfassen | Klein |
| G5 | Signaturen | Signatur-Verwaltung in Settings | Klein |

### D6.7: Kalender

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| H1 | Monats/Wochen/Tag-Ansicht | Drei Views mit Umschalter | Gross |
| H2 | Termin erstellen | Modal mit Datum, Zeit, Teilnehmer, Erinnerung | Mittel |
| H3 | Wiederkehrende Termine | Wiederholungs-Optionen im Termin-Modal | Klein |
| H4 | Raumbuchung | Raum-Auswahl im Termin-Modal | Klein |
| H5 | DACH-Feiertage | Feiertage als Markierungen im Kalender | Klein |
| H6 | Geteilte Kalender | Team-/Projekt-Kalender Overlay | Mittel |
| H7 | Verfuegbarkeit | Frei/Besetzt Anzeige bei Teilnehmer-Auswahl | Mittel |
| H8 | CalDAV/iCal Sync | Settings-Screen fuer externe Kalender | Mittel |
| H9 | Teams/Slack Integration | Plattform-Badge + Unified Inbox UI | Gross |

### D6.8: Team & HR

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| I1 | Team-Uebersicht | Mitarbeiter-Grid mit Rollen + Online-Status | Mittel |
| I2 | Online-Status | Presence-Dots (gruen/gelb/rot/grau) | Klein |
| I3 | Abwesenheitsverwaltung | Antrags-Formular + Genehmigungs-UI | Mittel |
| I4 | Organigramm | Visuelle Baumstruktur | Mittel |
| I5 | Arbeitsinteressen | Profil-Felder + Tags im Mitarbeiter-Detail | Klein |

### D6.9: Buchhaltung & Finanzen

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| J1 | Bexio-Integration | Settings: API-Key eingeben, Sync-Status | Mittel |
| J2 | Abacus-Integration | Settings: API-Key eingeben, Sync-Status | Mittel |
| J3 | Run my Accounts | Settings: API-Key eingeben, Sync-Status | Mittel |
| J4 | Rechnungserstellung | Rechnungs-Editor (Positionen, Summen, PDF-Preview) | Gross |
| J5 | Ausgabenverwaltung | Beleg-Upload + Kategorisierung | Mittel |
| J6 | Dashboard-Zahlen | Umsatz, Cashflow Charts | Mittel |

### D6.10: Einstellungen & System

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| L3 | Auto-Save | Indicator "Gespeichert" / "Speichern..." in Formularen | Klein |
| L4 | Vault/Sicherheit | Verschluesselungs-Settings Screen | Mittel |
| L5 | 2FA | 2FA-Setup Screen (QR-Code, Backup-Codes) | Mittel |
| L6 | Session-Verwaltung | Liste aktiver Sessions + "Abmelden" Button | Klein |
| L7 | Audit-Log | Tabelle mit Aenderungs-Historie (Admin-Only) | Mittel |
| L8 | DSGVO-Export | "Meine Daten exportieren" Button + Bestaetigungs-Dialog | Klein |
| K1 | Arbeitsprofile | Profil-Wechsler mit eigenen Configs | Mittel |
| K8 | Benachrichtigungs-Praeferenzen | Pro-Modul Toggle On/Off | Klein |
| K9 | Schriftgroesse | Groessen-Auswahl in Einstellungen | Klein |

---

## D7: Widgets & Overlays

**Ziel:** Alle Pop-up/Overlay-Elemente die ueber dem Content schweben

| # | Feature | Beschreibung | Aufwand |
|---|---------|-------------|---------|
| A4 | Quick-Actions Palette | Hochklappbare Leiste vom Desk-Rand (Schublade unten im Eck) + Tastenkuerzel | Mittel |
| A7 | Multi-Tab Interface | Tab-Leiste fuer mehrere offene Module (klaeren mit Luke) | Mittel |
| M1 | TimeTracker | Zeiterfassung-Widget mit Projekt-Zuordnung | Mittel |
| M3 | Help Widget | Support-Chat + Hilfeartikel Overlay | Mittel |
| M5 | Pomodoro-Timer | 25min/5min Zyklus Widget (fuer alle verfuegbar) | Klein |
| M6 | Notiz-Widget | Sticky-Notes Overlay (fuer alle verfuegbar) | Klein |
| M7 | Lesezeichen/Links | Links sammeln + organisieren (fuer alle verfuegbar) | Klein |
| M8 | Rechner | Taschenrechner Widget (fuer alle verfuegbar) | Klein |
| K6 | Onboarding-Wizard | Erste-Schritte-Flow bei erstem Start | Mittel |
| L1 | Offline-Modus | Offline-Indicator Restyling (Luke hat Backend) | Klein |
| L2 | Offline-Banner | "Keine Verbindung" Banner Restyling | Klein |

---

## D8: Desk Polish & Themes

**Ziel:** Drei Themes fertigstellen, Theme-Picker, Dekorationen erweitern

| Feature | Beschreibung | Aufwand |
|---------|-------------|---------|
| Theme Picker | UI zum Wechseln zwischen Themes | Mittel |
| K2 Theme: Cozy Desk | Standard-Theme verfeinern (Schreibtisch, Deko, Blur) | Mittel |
| K3 Theme: Minimal | Saubere Flaeche, subtiler Frosted Glass, keine Deko | Mittel |
| Theme: Dreamy/Creative | Lila Gradient, 3D Bubbles, Pastell (Inspiration-Bild 2) | Gross |
| K4 Deko verstellbar | Pflanzen, Fotos, Items auf Schreibtisch anpassen | Mittel |
| Desk-Hintergrund Auswahl | Verschiedene "Raeume" / Hintergruende waehlbar | Mittel |

---

## D9: Visual Polish & Accessibility

**Ziel:** Alles verfeinern, Animationen, leere Zustaende, Barrierefreiheit

| Feature | Beschreibung | Aufwand |
|---------|-------------|---------|
| Hover-Animationen | Subtile Hover-Effekte auf Karten, Buttons, Nav-Items | Mittel |
| Page Transitions | Sanfte Uebergaenge zwischen Screens | Mittel |
| Loading States | Skeleton-Screens waehrend Daten laden | Mittel |
| Empty States | Illustrationen + Hilfetext wenn Listen leer sind | Mittel |
| Micro-Interactions | Button-Press Feedback, Toggle-Animationen | Klein |
| Focus-Management | Tastatur-Navigation durch alle Elemente | Mittel |
| Screen Reader | ARIA Labels, Role Attributes | Mittel |
| Reduced Motion | Animationen ausschalten wenn System-Preference | Klein |
| Contrast Check | Alle Farbkombinationen auf WCAG AA pruefen | Klein |

---

## Feature-Zuordnung Checkliste (103/103)

Alle 103 genehmigten Features sind einer Phase zugeordnet:

| Phase | Features | Anzahl |
|-------|----------|--------|
| D1 | Desk-System | 6 (eigene) |
| D2 | Farbsystem | 9 (eigene) |
| D3 | A5, A9 + Sidebar-eigene | 10 |
| D4 | A1, A2, A3, K5, K7, M2, M4 | 9 |
| D5 | B1-B11, A6 | 12 |
| D6.1 | C1-C11 | 11 |
| D6.2 | D1-D9, D15, D16 | 11 |
| D6.3 | D10-D14 | 5 |
| D6.4 | E1-E10 | 10 |
| D6.5 | F1-F7 | 7 |
| D6.6 | G1-G5 | 5 |
| D6.7 | H1-H9 | 9 |
| D6.8 | I1-I5 | 5 |
| D6.9 | J1-J6 | 6 |
| D6.10 | L3-L8, K1, K8, K9 | 9 |
| D7 | A4, A7, M1, M3, M5-M8, K6, L1, L2 | 11 |
| D8 | K2, K3, K4 + Theme-eigene | 6 |
| D9 | Polish-eigene | — |
| **NICHT zugeordnet** | **A8 (NEIN), K10 (NEIN)** | **2 abgelehnt** |

---

## Uebergreifende Prinzipien (gelten fuer ALLE Phasen)

### 1. Rollenbasierte Ansichten
Jeder Screen muss fuer Admin, Projektleiter UND Mitarbeiter durchdacht werden.

### 2. Berechtigungssystem
Dateien, Projekte, Meetings, Chat — ueberall Freigabestufen beruecksichtigen.

### 3. Widgets kontextbewusst
Manche Widgets fuer alle (Rechner, Notizen), manche modulgebunden (Buchhaltungszahlen).

### 4. Anpassbarkeit
Sidebar, Quick-Actions, Dashboard — User richtet sich alles selbst ein.

---

## Uebergabe-Workflow

1. Darien plant + implementiert auf `design/brainstorm`
2. Phase abgeschlossen → SUMMARY.md schreiben
3. Branch pushen, Luke Bescheid geben
4. Luke reviewed, erstellt offizielle GSD-Phase auf `main`
5. Luke merged/adaptiert die Aenderungen
