# D1: Desk Foundation — Plan

## Goal

Schreibtisch-Metapher als aeusserstes Layout der Desktop-App implementieren.
Der Arbeitsbereich ("Fenster") wird von einem Raum umgeben: Waende links/rechts
fuer Dekoration, Schreibtisch-Oberflaeche unten, Wandfarbe oben.

## Scope

1. **Type System** — DeskTheme, DecorationSlot, DecorationPlacement Interfaces
2. **Theme Registry** — Data-driven Theme-System mit "Klassisches Buero" als Default
3. **DeskEnvironment** — Aeusserster Wrapper, Theme-Variablen, Dark-Mode-Erkennung
4. **DeskFrame** — Room-Scene: Wand-Panels (180px), Schreibtisch (120px), Top-Gap
5. **DeskDecorations** — Slot-basiertes Deko-System mit Clock als Proof-of-Concept
6. **DeskClock** — Analog-Uhr als reines SVG/CSS (kein Asset)
7. **Maximize-Modus** — Ctrl+Shift+F / Button, 8px Peek, Smooth Transition
8. **Bestehende Komponenten anpassen** — AppShell, Sidebar, Header integrieren

## Files

| Action | File |
|--------|------|
| NEW | types/desk-theme.ts |
| NEW | config/desk-themes.ts |
| NEW | components/layout/DeskEnvironment.tsx |
| NEW | components/layout/DeskFrame.tsx |
| NEW | components/desk/DeskDecorations.tsx |
| NEW | components/desk/decorations/DeskClock.tsx |
| MODIFY | stores/ui.ts |
| MODIFY | styles/globals.css |
| MODIFY | App.tsx |
| MODIFY | components/layout/AppShell.tsx |
| MODIFY | components/layout/Sidebar.tsx |
| MODIFY | components/layout/Header.tsx |
