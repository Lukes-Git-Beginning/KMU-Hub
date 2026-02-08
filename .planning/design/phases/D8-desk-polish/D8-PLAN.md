# D8: Desk Polish — Plan

## Goal

Das Desk-Metapher-System erweitern: Theme Picker UI, Dekorationen
(Pflanzen, Fotos, Items), zusaetzliche Desk-Themes.

## Tasks

### 1. Theme Picker UI
- Erreichbar ueber Sidebar oder Settings
- Miniatur-Vorschau jedes Themes
- Live-Preview bei Klick
- Grid-Layout mit Theme-Karten

### 2. Neue Themes
- "Modern Minimal" — Weiss/Grau, clean
- "Cozy Home Office" — Warme Erdtoene, Pflanzen
- "Night Owl" — Dunkel, blaues Akzentlicht
- Jedes Theme mit eigenen cssVariables + decorationSlots

### 3. Deko-System erweitern
- Pflanzen-SVGs: Monstera, Sukkulente, Kaktus, Farn
- Bilderrahmen: Rahmen-SVG mit Platzhalter
- Schreibtisch-Items: Kaffeetasse, Stifthalter, Notizblock
- Rechtsklick-Menue fuer Platzierung/Entfernung

### 4. Desk-Texturen
- Wand-Texturen (subtiler Putz-Effekt)
- Holzmaserung verfeinern
- Sockelleiste Wand → Schreibtisch
- Schatten-Verfeinerung

## Files

| Action | File |
|--------|------|
| NEW | components/desk/ThemePicker.tsx |
| NEW | components/desk/ThemePreviewCard.tsx |
| NEW | components/desk/DecorationPicker.tsx |
| NEW | components/desk/decorations/DeskPlant.tsx |
| NEW | components/desk/decorations/DeskPhoto.tsx |
| NEW | components/desk/decorations/DeskStationery.tsx |
| MODIFY | config/desk-themes.ts — Neue Themes |
| MODIFY | components/desk/DeskDecorations.tsx — Neue Typen |
| MODIFY | components/layout/DeskFrame.tsx — Texturen |
