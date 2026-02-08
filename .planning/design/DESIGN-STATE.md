# Design State

## Current Position

Phase: D2 of D9 (Color System & Theme — next up)
Last completed: D1 — Desk Foundation (2026-02-07)
Branch: design/brainstorm (up to date with main, Phase 5 merged)

## Progress

- [x] D1: Desk Foundation (2026-02-07)
- [ ] D2: Color System & Theme
- [ ] D3: Sidebar Redesign
- [ ] D4: Header Redesign
- [ ] D5: Dashboard
- [ ] D6: Module Screens
- [ ] D7: Widgets & Overlays
- [ ] D8: Desk Polish
- [ ] D9: Visual Polish

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

## Accumulated Decisions

- Desk themes are data-driven (add object to DESK_THEMES registry)
- CSS custom properties via inline style (CSP-safe)
- System dark mode detected reactively via matchMedia
- Maximize mode keeps 8px padding to peek background
- No animation library — pure CSS transitions
- DEV_BYPASS_AUTH in App.tsx for design work (must be removed before merge to main)
- Figma color palette: warm beige (#e8e3dd) + teal (#1e7e74) accent
- Dark mode will use OKLCH color space (neutral gray, hue 240)
- Figma-Export als code reference in desktop/design-reference/

## Key Color Tokens (from Figma)

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
