# D2: Color System & Theme — Plan

> Renamed from "Visual Polish" — this phase is now about establishing the Figma
> color palette as the foundation for all subsequent design work.

## Goal

Die warme Beige/Teal-Farbpalette aus dem Figma-Design uebernehmen und das gesamte
Farbsystem der App umstellen. Dies ist die Grundlage fuer ALLE weiteren Design-Phasen.

## Figma Reference

- `desktop/design-reference/src/styles/theme.css` — Komplettes Farbsystem
- `desktop/design-reference/src/styles/fonts.css` — Typografie

## Tasks

### 1. Farbpalette Light Mode
Umstellung von kaltem Slate auf warmes Beige/Teal:
- Page Background: `#e8e3dd` (warm beige-gray)
- Card Background: `#f5efe8` (light warm beige)
- Primary Accent: `#1e7e74` (warm teal)
- Text: `#2c2420` (heading), `#3d3531` (body), `#6b6159` (muted)
- Borders: `#d5cac0` (warm beige border)
- Status: Success `#4a7c6a`, Warning `#c4873a`, Error `#a13f3f`, Info `#3d5c7d`

### 2. Dark Mode (OKLCH)
Umstellung von HSL auf OKLCH Farbsystem:
- Neutral cool gray (kein braun!) — hue 240
- Background: `oklch(0.15 0.005 240)`
- Card: `oklch(0.18 0.008 240)`
- Primary: `oklch(0.55 0.15 180)`
- Alle Status-Farben in OKLCH

### 3. Sidebar & Header Tokens
Eigene CSS-Variablen fuer Sidebar und Header:
- Sidebar: `--sidebar`, `--sidebar-foreground`, `--sidebar-active`, etc.
- Header: `--header-background`, `--header-border`

### 4. Shadow System
Figma-Schatten uebernehmen:
- micro, small, medium, large, card, card-hover

### 5. Typography
- Font: Geist Sans mit Fallbacks
- Base: 16px
- Headings: weight 500, line-height 1.5

### 6. Desk-Theme Integration
- Desk-Theme CSS-Variablen muessen MIT dem neuen Farbsystem harmonieren
- classic-office Theme-Farben anpassen (warm beige statt oak)

## Files

| Action | File |
|--------|------|
| MODIFY | styles/globals.css — Komplettes Farbsystem ersetzen |
| MODIFY | config/desk-themes.ts — Theme-Farben an neue Palette anpassen |
| MODIFY | components/layout/Sidebar.tsx — Neue Sidebar-Tokens nutzen |
| MODIFY | components/layout/Header.tsx — Neue Header-Tokens nutzen |
| POSSIBLY | components/layout/AppShell.tsx — bg-background statt bg-card |

## Verification

- Alle Texte lesbar (Kontrast-Check)
- Sidebar warm beige, nicht kalt grau
- Primary-Buttons teal (#1e7e74)
- Dark Mode: neutral grau, kein braun
- Desk-Rahmen harmoniert mit neuem Farbsystem
