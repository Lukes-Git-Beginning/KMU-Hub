# D4: Decorations System — Plan

## Goal

Erweitertes Deko-System: Pflanzen, Bilderrahmen, Schreibtisch-Gegenstaende.
Benutzer kann Items in Slots platzieren und entfernen.

## Tasks

### 1. Neue Deko-Typen
- **Pflanzen** — 3-4 Varianten als SVG (Monstera, Sukkulente, kleiner Kaktus, Farn)
- **Bilderrahmen** — Rahmen-SVG mit Platzhalter-Bild (spaeter: eigenes Foto hochladen)
- **Schreibtisch-Items** — Kaffeetasse, Stifthalter, Notizblock als SVG
- Jeder Typ: eigene Renderer-Komponente im decorations/ Ordner

### 2. Deko-Platzierung UI
- Rechtsklick auf leeren Slot → Auswahl-Menue
- Oder: Deko-Panel in Sidebar mit Drag-Vorschau
- Item entfernen via Rechtsklick → "Entfernen"

### 3. Slot-Erweiterung
- Mehr Slots pro Theme (Regal an der Wand, Fensterbank, etc.)
- Slot-Typen bestimmen welche Items passen (Wand-Slots vs. Tisch-Slots)

### 4. Animations
- Items erscheinen mit leichter Scale+Fade Animation
- Pflanzen: subtiles Wackeln bei Hover (CSS keyframes)

## Files (voraussichtlich)

| Action | File |
|--------|------|
| NEW | components/desk/decorations/DeskPlant.tsx |
| NEW | components/desk/decorations/DeskPhoto.tsx |
| NEW | components/desk/decorations/DeskStationery.tsx |
| NEW | components/desk/DecorationPicker.tsx |
| MODIFY | components/desk/DeskDecorations.tsx (neue Typen) |
| MODIFY | config/desk-themes.ts (erweiterte Slots) |
| MODIFY | types/desk-theme.ts (evtl. neue Typen) |
