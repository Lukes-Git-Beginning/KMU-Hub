# D2: Visual Polish — Plan

## Goal

Das Desk-Fundament visuell aufwerten: realistischere Texturen, bessere Tiefenwirkung,
Farbharmonie zwischen Raum und Arbeitsbereich, Dark-Mode-Feinschliff.

## Tasks

### 1. Wand-Texturen
- Subtile Wand-Textur via CSS (leichtes Rauschen / Putz-Effekt)
- Unterschiedliche Toene fuer linke/rechte Wand (leichter Licht-Gradient)
- Sockelleiste am Uebergang Wand → Schreibtisch

### 2. Schreibtisch-Oberflaeche
- Holzmaserung verfeinern (realistischerer CSS-Gradient)
- Leichter Glanz-Effekt (subtle highlight gradient)
- Kante/Rand des Schreibtischs (3D-Effekt via border/shadow)

### 3. Fenster / Arbeitsbereich
- Shadow-Verfeinerung (mehrere Layer fuer realistischeren Schatten)
- Fensterrahmen-Effekt (subtile innere Border / Leiste)
- Glaseffekt-Andeutung (optionaler leichter Overlay)

### 4. Dark Mode
- Alle Texturen fuer Dark Mode anpassen
- Warme Lichtquellen-Andeutung (subtiler Glow im Raum)
- Kontrast-Check: Arbeitsbereich muss lesbar bleiben

### 5. Uebergaenge
- Maximize-Animation verfeinern (evtl. Scale statt nur Padding)
- Deko-Items fade-in beim Restore (gestaffelt)

## Files (voraussichtlich)

| Action | File |
|--------|------|
| MODIFY | config/desk-themes.ts (erweiterte CSS-Variablen) |
| MODIFY | styles/globals.css (neue Variablen) |
| MODIFY | components/layout/DeskFrame.tsx (Texturen, Sockelleiste) |
| MODIFY | components/layout/DeskEnvironment.tsx (Animation) |
| MODIFY | components/desk/DeskDecorations.tsx (gestaffelte Fade-ins) |
