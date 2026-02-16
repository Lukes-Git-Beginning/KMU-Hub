# D8: Desk Polish — Plan

## Goal

Das Desk-Metapher-System erweitern: Zwei Hauptthemes, geblurrter Natur-Hintergrund,
Theme Picker UI, Dekorationen (Pflanzen/Fotos/Items).

## Inspiration

Warmer, einladender Arbeitsplatz wie im Referenzbild:
- Geblurrter Natur-/Aquarell-Hintergrund hinter dem Arbeitsbereich (wie eine Aussicht)
- Realistische Pflanzen und Deko-Items am unteren Rand (Schreibtisch)
- Weiche, warme Aesthetic — professionell aber gemuetlich
- Im rausgezoomten Modus: Schreibtisch, Deko, und "Aussicht" sichtbar

## Tasks

### 1. Zwei Hauptthemes (Prioritaet!)

**Theme A: "Arbeitsplatz" (Desk)**
- Schreibtisch-Oberflaeche unten (Holz, verstellbare Deko)
- Seitenpanels: Wandflaeche mit Regalen, Bildern, Pflanzen
- Hintergrund: Geblurrter Natur-/Himmel-Hintergrund (wie Fensteraussicht)
  - CSS: `background-image` mit weichem Blur-Filter
  - Mehrere Hintergrund-Optionen waehlbar (Berge, Wald, See, Stadtpark)
- Deko verstellbar: Pflanzen, Kaffeetasse, Stifthalter, Bilderrahmen
- Maximiert: 8px Peek zeigt Hintergrund-Blur am Rand

**Theme B: "Minimal"**
- Kein Schreibtisch, keine Deko
- Rausgezoomter Modus: Seiten sind geblurrt (Frosted Glass Effekt)
  - CSS: `backdrop-filter: blur()` oder geblurrter Hintergrund
  - Dezenter, nicht ablenkend
- Fuer User die keine Spielereien wollen, nur saubere Arbeitsflaeche
- Maximiert: Fast kein visueller Unterschied (minimal Blur-Rand)

### 2. Theme Picker UI
- Erreichbar ueber Sidebar oder Settings
- Miniatur-Vorschau jedes Themes (Mini-Room-Scene)
- Live-Preview bei Klick (sofort sichtbar)
- Grid-Layout mit Theme-Karten
- Hintergrund-Auswahl im Desk-Theme (dropdown mit Vorschau)

### 3. Deko-System (nur fuer Desk-Theme)
- Pflanzen-SVGs: Monstera, Sukkulente, Kaktus, Farn (vom Inspiration-Bild)
- Schreibtisch-Items: Kaffeetasse, Stifthalter, Notizblock, Buecher
- Bilderrahmen: Rahmen-SVG mit Platzhalter-Bild
- Rechtsklick-Menue fuer Platzierung/Entfernung
- Items am unteren Rand (Schreibtisch-Kante) platzierbar

### 4. Hintergrund-System
- Geblurrte Hintergruende als CSS-Gradients oder leichte Bilder
  - Option 1: Reine CSS-Gradients (kein Asset noetig)
  - Option 2: Kleine komprimierte Bilder (< 50KB) mit CSS blur
- Aquarell-/Wasserfarben-Stil wie im Inspiration-Bild
- Wechselbar im Theme Picker

### 5. Desk-Texturen
- Holzmaserung verfeinern (realistischer)
- Sockelleiste Wand → Schreibtisch
- Schatten fuer 3D-Tiefe

## Files

| Action | File |
|--------|------|
| NEW | components/desk/ThemePicker.tsx |
| NEW | components/desk/ThemePreviewCard.tsx |
| NEW | components/desk/DecorationPicker.tsx |
| NEW | components/desk/decorations/DeskPlant.tsx |
| NEW | components/desk/decorations/DeskPhoto.tsx |
| NEW | components/desk/decorations/DeskStationery.tsx |
| MODIFY | config/desk-themes.ts — Zwei Themes + Hintergruende |
| MODIFY | types/desk-theme.ts — Background-Type + Minimal-Modus |
| MODIFY | components/desk/DeskDecorations.tsx — Neue Typen |
| MODIFY | components/layout/DeskFrame.tsx — Blur-Hintergrund, Minimal |
| MODIFY | components/layout/DeskEnvironment.tsx — Theme-Switching |
| NEW | assets/desk-backgrounds/ — Hintergrundbilder (falls noetig) |
