# D3: Theme Picker — Plan

## Goal

UI-Komponente zum Wechseln des Desk-Themes. Benutzer soll visuell sehen wie
jedes Theme aussieht, bevor er wechselt. Spaeter: Custom-Theme-Editor.

## Tasks

### 1. Theme-Vorschau-Karten
- Miniatur-Vorschau jedes Themes (Mini-Room-Scene als Thumbnail)
- Aktives Theme hervorgehoben
- Hover-Effekt mit vergroesserter Vorschau

### 2. Theme-Picker Overlay/Panel
- Erreichbar ueber Sidebar oder Settings
- Grid-Layout mit Theme-Karten
- Live-Preview: Theme wechselt sofort beim Klick (kein "Speichern" noetig, Zustand ist persistiert)

### 3. Zusaetzliche Themes erstellen
- "Modern Minimal" — Weiss/Grau, kein Holz, Clean
- "Cozy Home Office" — Warme Erdtoene, Pflanzen-Vibes
- "Night Owl" — Dunkle Toene, blaues Akzentlicht
- Jedes Theme: eigene cssVariables, cssVariablesDark, decorationSlots

### 4. Deko-Sichtbarkeit Toggle
- Schalter im Theme-Picker: Dekorationen ein/aus
- Bereits im Store vorhanden (deskDecorationsVisible)

## Files (voraussichtlich)

| Action | File |
|--------|------|
| NEW | components/desk/ThemePicker.tsx |
| NEW | components/desk/ThemePreviewCard.tsx |
| MODIFY | config/desk-themes.ts (neue Themes) |
| MODIFY | components/layout/Sidebar.tsx (Theme-Picker Zugang) |
