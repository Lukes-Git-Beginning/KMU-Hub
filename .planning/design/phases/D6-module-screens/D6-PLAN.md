# D6: Module Screens — Plan

## Goal

Alle Modul-Seiten nach dem Figma-Design umbauen/erstellen. Jeder Screen hat
spezifische Layouts, Filterung, und Interaktions-Patterns.

## Figma Reference

- `desktop/design-reference/src/app/screens/Projekte.tsx`
- `desktop/design-reference/src/app/screens/ProjektDetail.tsx`
- `desktop/design-reference/src/app/screens/Aufgaben.tsx`
- `desktop/design-reference/src/app/screens/Meetings.tsx`
- `desktop/design-reference/src/app/components/MeetingDetailView.tsx`
- `desktop/design-reference/src/app/screens/Nachrichten.tsx`
- `desktop/design-reference/src/app/screens/Kontakte.tsx`
- `desktop/design-reference/src/app/screens/Team.tsx`
- `desktop/design-reference/src/app/screens/Dokumente.tsx`
- `desktop/design-reference/src/app/screens/Mails.tsx`
- `desktop/design-reference/src/app/screens/Buchhaltung.tsx`
- `desktop/design-reference/src/app/screens/Profil.tsx`
- `desktop/design-reference/src/app/screens/Einstellungen.tsx`
- `desktop/design-reference/src/app/screens/Arbeitsprofile.tsx`
- `desktop/design-reference/src/app/screens/ModuleVerwalten.tsx`

## Screens (in Prioritaetsreihenfolge)

### 1. Projekte
- Grid-View mit Projekt-Karten (3→2→1 responsive)
- Filter-Bar: Status + Sort-By Dropdowns
- Pro Karte: farbiges Icon, Name, Beschreibung, Progress Bar, Team-Avatars, Status/Priority Badge, Deadline
- Hover: Shadow-Increase

### 2. Aufgaben
- Tab-Navigation: Alle | Mir zugewiesen | Von mir erstellt | Ueberfaellig
- Zeitbasierte Gruppierung: Heute | Diese Woche | Spaeter | Erledigt
- Task-Zeilen: Checkbox, Priority-Dot (rot/gelb/gruen), Titel, Projekt, Assignee, Deadline
- Strikethrough fuer erledigte

### 3. Meetings
- Filter-Chips: Alle | Live (pulsierend rot) | Geplant | Vergangen
- Hierarchische Gruppierung: LIVE NOW → Heute → Diese Woche → Geplant → Vergangen
- Meeting-Karten: Icon, Titel, Projekt, Zeit, Teilnehmer, Action-Buttons
- Create-Meeting Modal (4-Step Wizard)

### 4. Nachrichten (Chat)
- Luke hat schon ein 3-Panel Chat gebaut
- Figma-Anpassungen: Styling der Channel-Liste, Message-Bubbles, Thread-Panel

### 5. Kontakte, Team, Dokumente, Mails, Buchhaltung
- Jeder Screen hat spezifisches Layout im Figma
- Tabellen, Filter, Detail-Views

### 6. Profil, Einstellungen, Arbeitsprofile, ModuleVerwalten
- Settings-Seiten mit Formularen
- Profil mit Avatar-Upload
- Module ein/ausschalten

## Note

Diese Phase ist die groesste und wird wahrscheinlich in mehrere Unter-Phasen
aufgeteilt wenn wir sie angehen.
