# Design State

## Current Position

Phase: BACKEND UPDATE RECEIVED — Luke's massive cleanup + Phase 17.5 Gast-Chat komplett
Last completed: Feature Brainstorm + Strategy Docs + Pricing v2 (2026-02-25)
Branch: design/brainstorm (pushed FEATURE-BRAINSTORM.md + STRATEGY.md + PRICING.md v2)

## Luke Status Update (2026-02-25)

**Siehe:** `.planning/design/LUKE-ANTWORT-2026-02-25.md`

**Highlights:**
- CRM CRUD komplett fertig (voller CRUD, nicht nur GET)
- Phase 17.5 Gast-Chat komplett (Backend + Frontend SPA in `guest-chat/`)
- Email-Modul komplett (39 RPCs)
- DATEV-Export voll implementiert
- Gap-Analyse Wave 3: ~60% bereits gebaut
- Frontend kann JETZT gegen echte APIs bauen (kein Mock-Modus nötig)

## Action Items (2026-02-25)

**Option A: Gast-Chat Design-Upgrade** (Quick Win)
- Standalone SPA in `guest-chat/` stylen (aktuell plain CSS)
- 8 Komponenten redesignen (PreChatForm, ChatWindow, MessageBubble, etc.)

**Option B: Admin-UI für Gast-Channels** (Neues Feature)
- Toggle "Gast-Chat aktivieren" pro Channel
- Config: Logo, Primärfarbe, Willkommensnachricht, File-Limit

**Option C: Von Mocks zu echten APIs** (Migration)
- TanStack Query Hooks schreiben
- Mail/CRM-Module gegen echte Endpoints bauen

**Option D: Notification-Präferenzen** (Offene Frage von Luke)
- Wie wird Mitarbeiter benachrichtigt wenn Gast schreibt?
- Design für Settings

## Completed Strategy Work (2026-02-17 + 2026-02-25)

**2026-02-17:**
- 14 Research-Dokumente (~19.000 Zeilen) in `.planning/design/research/`
- Handoff-Dokument: `docs/PRODUCT-STRATEGY.md`
- Luke-Prompt: `research/PROMPT-FUER-LUKE.md`
- Build/Integrate Matrix, Gap-Analyse, Compliance-Framework, DB-Modelle
- Backend-Plan (Part 1+2), Frontend-Plan, Kostenanalyse, Infrastruktur-Matrix

**2026-02-25:**
- FEATURE-BRAINSTORM.md (103 Features, alle bewertet)
- STRATEGY.md (Digitale Souveränität + MS Office Koexistenz)
- PRICING.md v2 (Role-Based Pricing + Einmalkauf + Branchenpakete)

## NEXT SESSION TODO
1. **Entscheidung:** Welches Action Item (A/B/C/D)?
2. D10 Visual Polish — Animationen, Accessibility (später)
3. D11 Nico-Review — Gesamte App durchgehen lassen (später)

## Progress

- [x] D1: Desk Foundation (2026-02-07)
- [x] D2: Color System & Theme (2026-02-08)
- [x] D3: Sidebar Redesign (2026-02-08)
- [x] D4: Header Redesign (2026-02-08)
- [x] D5: Dashboard (2026-02-08)
- [x] D6: Module Screens (2026-02-08)
- [x] D7: Interaktions-Tiefe (2026-02-13) — ALL sub-phases complete
- [x] D8: Widgets & Overlays (2026-02-13) — Already existed: TimeTracker, HelpWidget, Onboarding, DailyPlanner
- [ ] D9: Desk Polish (PAUSED — manual design by Darien)
- [ ] D10: Visual Polish (Animationen, Accessibility)
- [ ] D11: Nico-Review Fixes

## D7 — Interaktions-Tiefe (Sub-Phasen) — COMPLETE

| Sub-Phase | Modul | Status | Key Files |
|-----------|-------|--------|-----------|
| D7.1 | Meetings | DONE (2026-02-13) | MeetingsPage (color stripes, timeline, countdown), MeetingDetailPanel (gradient header, 3-tab, agenda), MeetingFormDialog (color picker, agenda editor) |
| D7.2 | Kontakte | DONE (2026-02-13) | ContactDetailPanel, ContactFormDialog, ImportContactsDialog (CSV parser), GroupManagerDialog, store (groups, bulkAdd) |
| D7.3 | Dokumente | DONE (pre-existing) | FileDetailPanel, FilePreviewModal, FolderCreateDialog, RenameDialog, ShareDialog (6 files) |
| D7.4 | Mails | DONE (pre-existing) | ComposeModal (new/reply/forward), ItemActions, folder management |
| D7.5 | Kalender | DONE (pre-existing) | Day/Week/Month views, EventForm, EventDetail, RoomBooking, CategoryManager (4 files) |
| D7.6 | Team & HR | DONE (pre-existing) | MemberDetailPanel, InviteMemberDialog, EditMemberDialog, HRApprovalDialog, AbsenceCalendar (6 files) |
| D7.7 | Buchhaltung | DONE (pre-existing) | InvoiceDetailPanel, InvoiceFormDialog, ExpenseFormDialog, PaymentRecordDialog, ExportDialog (6 files) |
| D7.8 | Einstellungen | DONE (pre-existing) | 7 files incl. 5 tab components, desk theme picker, 2FA setup, notification matrix |
| D7.9 | Profil & Header | DONE (pre-existing) | ProfilPage (3 tabs: Profil/Zeiterfassung/Abwesenheiten), ProfileMenu, NotificationBell+Center |
| D7.10 | Desk & Deko | SKIPPED | Deko removed from rendering, desk design handled separately by Darien |

## D8 — Widgets & Overlays — COMPLETE (pre-existing)

| Widget | File | Status |
|--------|------|--------|
| TimeTrackerWidget | header/TimeTrackerWidget.tsx | DONE — Timer controls, categories, today's entries, progress bar |
| DailyPlannerWidget | header/DailyPlannerWidget.tsx | DONE — Task planner with priority, tabs (Heute/Morgen/Spaeter) |
| HelpWidget | components/widgets/HelpWidget.tsx | DONE — Floating button, FAQ, shortcuts, contact, docs |
| OnboardingWizard | components/onboarding/OnboardingWizard.tsx | DONE — 6-step wizard, confetti, profile setup |
| SearchBar | header/SearchBar.tsx | DONE — Ctrl+K global search with category filters |
| NotificationBell | notifications/NotificationBell.tsx | DONE — Header popover with unread count |
| Dashboard Widgets | components/widgets/ | DONE — WidgetRegistry, WidgetContainer (react-grid-layout), WidgetWrapper |

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
| D7.1 | 2026-02-13 | - | Meetings polish: color stripes, countdown badges, Grid/Timeline toggle, gradient detail header, 3-tab (Details/Agenda/Notizen), form color picker + agenda editor |
| D7.2 | 2026-02-13 | - | Kontakte: ImportContactsDialog (CSV parser), GroupManagerDialog, groups in sidebar, store extensions (bulkAdd, groups CRUD) |
| D9 | 2026-02-12/13 | - | Room scene pipeline (paused), 6 desk themes with room scenes, DeskDecorations removed |

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
