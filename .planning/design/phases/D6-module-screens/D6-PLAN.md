# D6: Module Screens — Plan

## Goal

Alle Modul-Seiten nach dem Figma-Design umbauen/erstellen. Primaer orientiert
an den Funktionen aus dem Figma-Export. Jeder Screen hat spezifische Layouts,
Filterung, und Interaktions-Patterns.

**Wichtig:** Wir orientieren uns primaer an den Figma-Funktionen und bauen
die UI-Tiefe die im Figma noch fehlt schrittweise dazu.

## Figma Reference

- `desktop/design-reference/src/app/screens/` — Alle Screen-Designs
- `desktop/design-reference/src/app/components/` — Zugehoerige Komponenten

## Screens (in Prioritaetsreihenfolge)

### 1. Projekte
- Grid-View mit Projekt-Karten (3→2→1 responsive)
- Filter-Bar: Status + Sort-By Dropdowns
- Pro Karte: farbiges Icon, Name, Beschreibung, Progress Bar, Team-Avatars, Status/Priority Badge, Deadline
- Hover: Shadow-Increase
- Figma: `Projekte.tsx`, `ProjektDetail.tsx`

### 2. Aufgaben
- Tab-Navigation: Alle | Mir zugewiesen | Von mir erstellt | Ueberfaellig
- Zeitbasierte Gruppierung: Heute | Diese Woche | Spaeter | Erledigt
- Task-Zeilen: Checkbox, Priority-Dot (rot/gelb/gruen), Titel, Projekt, Assignee, Deadline
- Strikethrough fuer erledigte
- Figma: `Aufgaben.tsx`

### 3. Meetings (TIEFE AUSBAU!)
Aus dem Figma + zusaetzliche Anforderungen:

**Meeting-Uebersicht:**
- Filter-Chips: Alle | Live (pulsierend rot) | Geplant | Vergangen
- Hierarchische Gruppierung: LIVE NOW → Heute → Diese Woche → Geplant → Vergangen
- Meeting-Karten: Icon, Titel, Projekt, Zeit, Teilnehmer, Action-Buttons
- Create-Meeting Modal (4-Step Wizard)
- Figma: `Meetings.tsx`, `MeetingDetailView.tsx`

**Meeting-Detail-View (wie im Inspiration-Bild):**
- Zentraler Bereich: Meeting-Info (Titel, Zeit, "In X Minuten")
- Tabs: Beitreten | Teilnehmer | Dateien | Aufgaben | Verlauf
- Teilnehmer-Avatars mit Namen
- Agenda mit Punkten (farbige Dots)
- Videokonferenz-Link (klickbar)
- Rechte Seite: Notizen-Panel mit Tabs (Notizen | To-Do | Info)
  - Meeting-Notizen (Checklist-Items)
  - Bereichsplanung (Checkboxen)
  - ToDos mit Checkboxen
  - "Notiz hinzufuegen" Input + Senden-Button

**Call/Telefon-Ansicht (zu planen):**
- Video-Call Layout: Haupt-Video + Thumbnails
- Audio-Call: Avatar-Kreise, Mute/Unmute, Lautsprecher
- Bildschirmfreigabe-Ansicht
- Steuerungsleiste: Mikro, Kamera, Teilen, Chat, Verlassen

**Whiteboard (Zukunft, Lukes Backend):**
- In-Meeting Zeichenflaeche
- Sichtbar fuer alle Teilnehmer (inkl. externe via Link)
- Funktioniert wie Bildschirmuebertragung fuer andere
- NOTE: Backend/Echtzeit ist Lukes Bereich, wir designen nur die UI

### 4. Nachrichten (Chat)
- Luke hat schon 3-Panel Chat gebaut
- Figma-Anpassungen: Styling der Channel-Liste, Message-Bubbles, Thread-Panel
- Call-Button in Chat fuer Direktanrufe
- Figma: `Nachrichten.tsx`

### 5. Kontakte
- Kontaktliste mit Suche und Filtern
- Kontakt-Detail mit Tabs
- Figma: `Kontakte.tsx`

### 6. Team
- Team-Uebersicht mit Rollen
- Figma: `Team.tsx`

### 7. Dokumente
- Datei-Browser mit Ordnerstruktur
- Farbcodierte Dateitypen (PDF=braun, Word=blau, Excel=gruen, etc.)
- Figma: `Dokumente.tsx`

### 8. Mails
- E-Mail-Ansicht mit Ordnern
- Figma: `Mails.tsx`

### 9. Buchhaltung
- Bexio/Abacus/Run my Accounts Integration
- Figma: `Buchhaltung.tsx`

### 10. Profil, Einstellungen, Arbeitsprofile, ModuleVerwalten
- Settings-Seiten mit Formularen
- Profil mit Avatar-Upload
- Module ein/ausschalten
- Figma: `Profil.tsx`, `Einstellungen.tsx`, `Arbeitsprofile.tsx`, `ModuleVerwalten.tsx`

## Note

Diese Phase ist die groesste und wird wahrscheinlich in mehrere Unter-Phasen
aufgeteilt wenn wir sie angehen. Meeting-Bereich allein koennte eine eigene
Unter-Phase sein wegen der Tiefe (Detail-View, Call-UI, Whiteboard-Design).
