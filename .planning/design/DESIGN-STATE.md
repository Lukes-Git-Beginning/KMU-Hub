# Design State

## Current Position

Phase: D7 of D11 (Interaktions-Tiefe — next up)
Last completed: D6 — Module Screens (2026-02-08)
Branch: design/brainstorm (merged with main, Luke's Phase 6 included)

## NEXT SESSION TODO
1. D7 starten — Modul fuer Modul Tiefe einbauen (Detail-Views, Modals, Formulare, Kontextmenues, Empty States)
2. Jedes Modul gruendlich fertigstellen bevor zum naechsten
3. In 3-4 Tagen: Nico-Review (gesamte App, Flaw-Liste)

## Progress

- [x] D1: Desk Foundation (2026-02-07)
- [x] D2: Color System & Theme (2026-02-08)
- [x] D3: Sidebar Redesign (2026-02-08)
- [x] D4: Header Redesign (2026-02-08)
- [x] D5: Dashboard (2026-02-08)
- [x] D6: Module Screens (2026-02-08)
- [ ] D7: Interaktions-Tiefe (GROSS — alle Module bekommen Tiefe)
- [ ] D8: Widgets & Overlays (TimeTracker, HelpWidget, Onboarding)
- [ ] D9: Desk Polish (Theme Picker, Cozy/Minimal/Dreamy)
- [ ] D10: Visual Polish (Animationen, Empty States, Accessibility)
- [ ] D11: Nico-Review Fixes

## D7 — Interaktions-Tiefe (Sub-Phasen)

Jedes Modul bekommt: Detail-Ansicht, Erstellen-Formular, Bearbeiten, Kontext-Menue (3-Punkte), Bestaetigungs-Dialoge, Empty States.

| Sub-Phase | Modul | Was fehlt |
|-----------|-------|-----------|
| D7.1 | Meetings | Detail-View, Create-Modal, 3-Punkte-Menue (Bearbeiten/Loeschen/Teilen), Join-Flow, leerer Zustand |
| D7.2 | Kontakte | Kontakt-Detail (Profil-Ansicht), Create/Edit-Formular, Kontext-Menue, Import-Dialog, Gruppen |
| D7.3 | Dokumente | Datei-Vorschau, Upload-Modal, Ordner-Verwaltung, Sharing-Dialog, Kontext-Menue |
| D7.4 | Mails | Compose-Modal, Reply/Forward, Attachment-Preview, Ordner-Management |
| D7.5 | Kalender | (Bereits mit Tiefe!) Raum-Buchungsseite, Kategorie-Verwaltung, Kalender-Browse |
| D7.6 | Team & HR | Mitglied-Detail, Einladen-Dialog, Rollen-Verwaltung, HR-Antraege-Detail |
| D7.7 | Buchhaltung | Rechnungs-Detail, Erstellen-Formular, Transaktions-Detail, Export |
| D7.8 | Einstellungen | Alle Toggles funktional, Profil-Bearbeitung, Sicherheit (2FA Flow), Benachrichtigungs-Prefs |
| D7.9 | Profil & Header | Detailliertes Profil-Overlay, Profil-Bearbeitung, Notification-Panel mit Actions |
| D7.10 | Desk & Deko | Deko-Picker Overlay, Drag-to-Place, Theme-Vorschau |

## D6 Module Screens — Completed

| Sub-Phase | Module | Route | File |
|-----------|--------|-------|------|
| D6.1 | Projekte & Aufgaben | /work/* | Already restyled via design tokens |
| D6.2 | Meetings | /meetings | modules/meetings/MeetingsPage.tsx |
| D6.3 | Chat | /chat | Already restyled via design tokens |
| D6.4 | Kontakte | /kontakte | modules/kontakte/KontaktePage.tsx |
| D6.5 | Dokumente | /dokumente | modules/dokumente/DokumentePage.tsx |
| D6.6 | E-Mail | /mails | modules/mails/MailsPage.tsx |
| D6.7 | Kalender | /kalender | modules/kalender/KalenderPage.tsx |
| D6.8 | Team & HR | /team | modules/team/TeamPage.tsx |
| D6.9 | Buchhaltung | /buchhaltung | modules/buchhaltung/BuchhaltungPage.tsx |
| D6.10 | Einstellungen | /settings | modules/settings/SettingsPage.tsx |

All modules use mock data, our design tokens (warm beige/teal), and support dark mode.
Sidebar nav updated: all 12 items enabled, reordered logically.

## Figma Reference

Figma-Export gespeichert in `desktop/design-reference/`.
Wichtigste Dateien:
- `src/styles/theme.css` — Komplettes Farbsystem (Light + Dark OKLCH)
- `src/app/screens/` — Alle Screen-Designs
- `src/app/components/` — Alle Komponenten
- `src/app/contexts/` — ProfileContext, ThemeContext

## Completed Work

| Phase | Date | Commits | Key Changes |
|-------|------|---------|-------------|
| D1 | 2026-02-07 | 3 | DeskEnvironment, DeskFrame, DeskDecorations, DeskClock, Theme system, Maximize mode |
| D2 | 2026-02-08 | 1 | Figma color palette (warm beige/teal), OKLCH dark mode, .dark class toggle, @theme inline mapping, typography base styles |
| D3 | 2026-02-08 | 2 | Sidebar rewrite: 10 nav items, badge system (text/live/dot), branding header, user profile + online status, mobile drawer, tablet auto-collapse, sidebar-* CSS tokens |
| Merge | 2026-02-08 | 1 | Luke's Phase 6 (Work module) merged — Projekte + Aufgaben now active in sidebar |
| D4 | 2026-02-08 | 1 | Header redesign: SearchBar, DailyPlanner, LanguageSwitcher, ProfileSwitcher, ProfileMenu |
| D5 | 2026-02-08 | 1 | Dashboard: Greeting, Alerts, NotificationsFeed, ModulesGrid, Activity, QuickStats |
| D6 | 2026-02-08 | - | 8 new module screens + router wiring + sidebar update (all 12 nav items enabled) |

## Accumulated Decisions

- Desk themes are data-driven (add object to DESK_THEMES registry)
- CSS custom properties via inline style (CSP-safe)
- System dark mode detected reactively via matchMedia
- .dark class toggled on <html> element for CSS dark mode
- Dark mode uses .dark class (not @media prefers-color-scheme) for manual toggle support
- Maximize mode keeps 8px padding to peek background
- No animation library — pure CSS transitions
- DEV_BYPASS_AUTH in App.tsx for design work (must be removed before merge to main)
- Figma color palette: warm beige (#e8e3dd) + teal (#1e7e74) accent
- Dark mode uses OKLCH color space (neutral gray, hue 240 — NOT brown)
- Figma-Export als code reference in desktop/design-reference/
- Feature brainstorm reviewed: 103/105 features approved (2026-02-08)
- Sidebar extracted into sidebar/ subfolder (Sidebar, SidebarBranding, SidebarNav, SidebarBadge, SidebarUser)
- Nav items config-driven via sidebar/nav-items.ts (data-only, no JSX)
- Sidebar uses sidebar-* CSS tokens (not desk-sidebar-*) for Figma consistency
- Disabled nav items show "Bald verfuegbar" tooltip, no navigation
- useMediaQuery hook for responsive behavior (useSyncExternalStore)
- Mobile drawer infrastructure ready (trigger comes in D4 Header)
- Luke's Work module routes: /work/projects, /work/my-tasks, /work/search
- All module screens use mock data for design preview without backend
- New modules: meetings, kontakte, dokumente, mails, kalender, team, buchhaltung, settings
- Settings split: /settings (full settings page) + /settings/dashboard (admin dashboard config)
- Sidebar reordered: Dashboard > Projects > Tasks > Chat > Contacts > Team > Meetings > Calendar > Documents > Email > Accounting > Settings
- Team module on /team (separate from /crm which is Luke's CRM)

## Inspiration Reference (Cozy Workspace Image)

Key visual elements from Darien's reference image:
- Frosted glass / semi-transparent UI panels over desk background
- Desk visible through panels (plants, laptop, stationery, coffee cup)
- Three-column layout: Sidebar | Main Content | Detail Panel
- Sidebar: Branding + dropdown, avatar + online status, nav icons, mini calendar, upcoming appointments
- Header: Minimal — only icon row right side (chat, messages, folder, bell with red badge, profile avatar)
- Cards: Soft rounded corners, light shadows, warm colors
- Active nav item: Colored background fill (teal/blue)
- Tab navigation within panels (Notizen, To-Do, Info)
- Overall: Professional but cozy/inviting, NOT corporate-cold

## Theme Concepts (for D8)

Three planned themes:
1. **Cozy Desk** (DEFAULT) — warm beige/teal, real desk background with plants/stationery, frosted glass panels
2. **Minimal** — clean frosted glass, no desk decorations, muted neutral background
3. **Dreamy/Creative** — lila/lavendel gradient background, abstract 3D bubbles/spheres, pastel accent colors (mint, rosa, hellblau), stronger frosted glass transparency, playful/futuristic vibe for creative teams

## Key Color Tokens (from Figma — NOW LIVE)

### Light Mode
- Background: `#e8e3dd`
- Card: `#f5efe8`
- Primary: `#1e7e74` (teal)
- Text Heading: `#2c2420`
- Text Body: `#3d3531`
- Text Muted: `#6b6159`
- Border: `#d5cac0`

### Dark Mode (OKLCH)
- Background: `oklch(0.15 0.005 240)`
- Card: `oklch(0.18 0.008 240)`
- Primary: `oklch(0.55 0.15 180)`
