# Design Roadmap: KMU Hub Desktop

## Overview

Design-Phasen fuer das KMU Hub Desktop Frontend, basierend auf dem Figma-Export
(`desktop/design-reference/`). Getrennt von Lukes GSD-Phasen (`.planning/ROADMAP.md`),
um Merge-Konflikte zu vermeiden.

**Branch:** `design/brainstorm`
**Designer:** Darien
**Uebergabe an:** Luke (main branch, GSD-Phasen)
**Figma-Referenz:** `desktop/design-reference/`

---

## Design-Phasen

- [x] **D1: Desk Foundation** — Schreibtisch-Metapher, Room-Scene Layout, Theme-System, Maximize-Modus
- [ ] **D2: Color System & Theme** — Figma-Farbpalette (warm beige/teal) uebernehmen, Dark Mode OKLCH, Typography
- [ ] **D3: Sidebar Redesign** — Figma-Sidebar mit Badges, Live-Indicator, Collapse, Branding
- [ ] **D4: Header Redesign** — SearchBar, DailyPlanner, LanguageSwitcher, ProfileSwitcher, NotificationCenter
- [ ] **D5: Dashboard** — ModulesGrid, Alerts, NotificationsFeed, Activity Feed, Quick Stats
- [ ] **D6: Module Screens** — Projekte, Aufgaben, Meetings (inkl. Call-UI, Whiteboard-Design), Kontakte, Team, Dokumente, Mails, Buchhaltung
- [ ] **D7: Widgets & Overlays** — TimeTracker, HelpWidget, ProfileSystem, OnboardingWizard
- [ ] **D8: Desk Polish** — Zwei Themes (Desk+Minimal), geblurrter Hintergrund, Theme Picker, Dekorationen
- [ ] **D9: Visual Polish** — Texturen, Animationen, Empty States, Micro-Interactions

---

## Abhaengigkeiten

```
D1 (done) ─┐
            ├─→ D2 (Farben muessen zuerst stehen)
            │     ├─→ D3 (Sidebar braucht neue Farben)
            │     ├─→ D4 (Header braucht neue Farben)
            │     └─→ D5 (Dashboard braucht neue Farben)
            │           └─→ D6 (Module-Screens bauen auf Dashboard-Patterns auf)
            │
            ├─→ D7 (Widgets unabhaengig, aber nach D2 sinnvoller)
            ├─→ D8 (Desk-spezifisch, unabhaengig von Module-Arbeit)
            └─→ D9 (Finaler Polish, ganz am Ende)
```

## Figma Feature-Inventar

Features aus dem Figma die NICHT in Lukes aktueller Implementation sind:

| Feature | Figma-Datei | Status im Projekt |
|---------|-------------|-------------------|
| Warme Farbpalette (Beige/Teal) | theme.css | Fehlt (nutzt slate) |
| OKLCH Dark Mode | theme.css | Fehlt (nutzt HSL) |
| SearchBar (global) | SearchBar.tsx | Fehlt |
| DailyPlanner Widget | DailyPlannerWidget.tsx | Fehlt |
| TimeTracker Widget | TimeTrackerWidget.tsx | Fehlt |
| HelpWidget (Support-Chat) | HelpWidget.tsx | Fehlt |
| ProfileSwitcher | ProfileSwitcher.tsx | Fehlt |
| ProfileMenu | ProfileMenu.tsx | Fehlt |
| Language Switcher | Header.tsx | Fehlt |
| ModulesGrid (Dashboard) | ModulesGrid.tsx | Fehlt (hat Widget-Grid) |
| NotificationsFeed (Tabs) | NotificationsFeed.tsx | Teilweise (NotificationBell) |
| OnboardingWizard | OnboardingWizard.tsx | Fehlt |
| Projekte Screen | Projekte.tsx | Fehlt |
| Aufgaben Screen | Aufgaben.tsx | Fehlt |
| Meetings Screen | Meetings.tsx | Fehlt |
| Meeting Detail View | MeetingDetailView.tsx | Fehlt |
| Dokumente Screen | Dokumente.tsx | Fehlt |
| Mails Screen | Mails.tsx | Fehlt |
| Buchhaltung Screen | Buchhaltung.tsx | Fehlt |
| Team Screen | Team.tsx | Fehlt |
| Profil Screen | Profil.tsx | Fehlt |
| Einstellungen Screen | Einstellungen.tsx | Teilweise |
| Arbeitsprofile Screen | Arbeitsprofile.tsx | Fehlt |
| VaultSettings | VaultSettings.tsx | Fehlt |

## Design-Prinzipien

1. **Figma-First:** Primaer an den Funktionen orientieren die im Figma dargestellt sind
2. **UI-Tiefe:** Die Figma-Screens zeigen die Oberflaeche — wir bauen die Tiefe dazu
   (Detail-Views, Modals, Call-UIs, etc.)
3. **Zwei Themes:** "Arbeitsplatz" (Desk mit Deko + geblurrtem Hintergrund) und
   "Minimal" (geblurrte Seiten, keine Spielereien)
4. **Warme Aesthetic:** Einladend, professionell aber gemuetlich (wie Inspiration-Bild)
5. **Meeting-Tiefe:** Gut ausgebauter Meeting-Bereich mit Detail-View, Call-UI,
   Telefon-Ansicht, und spaeter Whiteboard (Backend = Luke)

## Uebergabe-Workflow

1. Darien plant + implementiert auf `design/brainstorm`
2. Phase abgeschlossen → SUMMARY.md schreiben
3. Branch pushen, Luke Bescheid geben
4. Luke reviewed, erstellt offizielle GSD-Phase auf `main`
5. Luke merged/adaptiert die Aenderungen
