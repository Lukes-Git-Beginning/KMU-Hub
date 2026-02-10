# AI Asset Pipeline — Implementierungsplan

## Was bauen wir?

Ein System, bei dem **Claude Code plant** und **OpenAI Bilder generiert**.
Statt handgemalte SVGs fuer die Desk-Dekos bauen wir eine Pipeline:

```
Claude plant Szene
  → baut Prompt mit Style-Regeln
    → OpenAI generiert PNG (einzelnes Objekt, freigestellt)
      → Vision-Modell prueft Qualitaet
        → gut? → speichern in Asset-Library
        → schlecht? → Prompt anpassen → nochmal
```

Am Ende: professionelle, konsistente Desk-Assets die wir per CSS animieren.

---

## Uebersicht: 6 Schritte

| # | Was | Dauer ca. |
|---|-----|-----------|
| 1 | OpenAI Setup (API Key + npm Package) | 10 min |
| 2 | Style DNA definieren (Master-Prompt-Regeln) | 15 min |
| 3 | Asset-Generator-Script bauen | 30 min |
| 4 | Erste Assets generieren (Pflanze, Kaffee, etc.) | 20 min |
| 5 | Assets ins Desk-System einbauen | 30 min |
| 6 | CSS-Animationen drauf | 20 min |

---

## Schritt 1: OpenAI Setup

### Was du brauchst
- Einen **OpenAI API Key** von https://platform.openai.com/api-keys
- Das `openai` npm Package

### Was wir machen
```bash
cd desktop
npm install openai --save-dev
```

Dann eine `.env` Datei im `desktop/` Ordner:
```
OPENAI_API_KEY=sk-...dein-key...
```

> WICHTIG: `.env` wird NICHT committed (steht schon in .gitignore)

### Warum --save-dev?
Das Script laeuft nur bei uns Entwicklern zum Generieren.
Die fertige App braucht kein OpenAI — sie nutzt die fertigen PNGs.

---

## Schritt 2: Style DNA

Eine Datei die definiert wie ALLE unsere Assets aussehen muessen.
Damit sehen 100 Bilder spaeter noch gleich aus wie das erste.

### Datei: `desktop/scripts/asset-gen/style-dna.ts`

```typescript
export const STYLE_DNA = {
  // Basis-Stil der in JEDEM Prompt dabei ist
  baseStyle: [
    'clean modern illustration',
    'soft pastel color palette',
    'warm natural lighting from top-left',
    'subtle soft shadows',
    'slightly stylized, not photorealistic',
    'european cozy workspace aesthetic',
    'high detail but minimalistic',
    'white or transparent background',
    'isolated object, no background clutter',
  ],

  // Farbregeln (passend zu unserer Figma-Palette)
  colors: {
    primary: 'warm teal (#1e7e74)',
    wood: 'warm oak brown, light honey tone',
    accent: 'sage green, dusty rose, soft cream',
    avoid: 'neon colors, pure black, cold blue',
  },

  // Licht & Schatten
  lighting: 'soft daylight from upper left, gentle ambient shadows, no harsh contrasts',

  // Format-Regeln
  output: {
    format: 'PNG with transparency where possible',
    sizes: {
      decoration: '512x512',   // Desk-Dekos (Pflanze, Kaffee, etc.)
      background: '1920x1080', // Hintergruende
      frame: '256x256',        // Kleine UI-Elemente
    },
  },
}
```

### Warum das wichtig ist
Ohne Style DNA wird jedes Bild anders aussehen.
Mit Style DNA bleibt alles konsistent — wie ein echtes Design-System.

---

## Schritt 3: Asset-Generator-Script

Ein Node.js Script das du per Terminal aufrufst.
Es nimmt einen Asset-Namen, baut einen Prompt, und generiert das Bild.

### Datei: `desktop/scripts/asset-gen/generate.ts`

So funktioniert es:

```
npx tsx scripts/asset-gen/generate.ts --asset plant --theme cozy
```

### Was das Script macht (vereinfacht):

1. Liest die Style DNA
2. Liest die Asset-Definition (z.B. "Topfpflanze, Sukkulente, auf Holz-Unterteller")
3. Baut daraus einen vollstaendigen Prompt
4. Ruft OpenAI Image API auf (`gpt-image-1`)
5. Speichert das Bild als PNG
6. Optional: Schickt das Bild an Vision-Modell zur Pruefung
7. Gibt Feedback aus (gut/schlecht/Verbesserungsvorschlag)

### Asset-Definitionen: `desktop/scripts/asset-gen/assets.ts`

```typescript
export const ASSET_DEFINITIONS = {
  // Desk-Dekos (Cozy Theme)
  'cozy/plant': {
    name: 'Topfpflanze',
    prompt: 'A small potted succulent plant in a terracotta pot, sitting on a tiny wooden coaster',
    size: '512x512',
    animation: { type: 'sway', pivot: 'bottom-center', amplitude: '2deg', duration: '6s' },
  },
  'cozy/coffee': {
    name: 'Kaffeetasse',
    prompt: 'A warm cup of coffee in a ceramic mug, slight steam rising, on a small saucer',
    size: '512x512',
    animation: { type: 'steam', effect: 'rising-wisps', duration: '4s' },
  },
  'cozy/pen-holder': {
    name: 'Stiftebecher',
    prompt: 'A wooden pen holder cup with 3-4 colored pencils and a fountain pen',
    size: '512x512',
    animation: null, // statisch
  },
  'cozy/photo-frame': {
    name: 'Bilderrahmen',
    prompt: 'A small wooden photo frame showing a blurred landscape photo, leaning slightly',
    size: '512x512',
    animation: null,
  },
  'cozy/books': {
    name: 'Buecherstapel',
    prompt: 'A small stack of 3 hardcover books in muted pastel colors, slightly offset',
    size: '512x512',
    animation: null,
  },

  // Dreamy Theme
  'dreamy/plant': {
    name: 'Kristall-Pflanze',
    prompt: 'A magical crystal plant with translucent purple and teal leaves, soft glow, fantasy style',
    size: '512x512',
    animation: { type: 'shimmer', effect: 'glow-pulse', duration: '3s' },
  },
  'dreamy/orb': {
    name: 'Leucht-Kugel',
    prompt: 'A floating glass orb with swirling pastel mist inside, magical, dreamy',
    size: '512x512',
    animation: { type: 'float', amplitude: '8px', duration: '5s' },
  },

  // Hintergruende
  'bg/cozy-wall': {
    name: 'Cozy Wandhintergrund',
    prompt: 'A warm room wall with soft natural light coming from a window on the left, beige and cream tones, subtle texture',
    size: '1920x1080',
    animation: null,
  },
  'bg/dreamy-gradient': {
    name: 'Dreamy Hintergrund',
    prompt: 'A dreamy gradient background, soft purple to teal, with subtle floating light particles',
    size: '1920x1080',
    animation: null,
  },
}
```

---

## Schritt 4: Assets generieren

### Ablauf pro Asset:

```
1. Script generiert Bild via OpenAI API
2. Bild wird gespeichert: desktop/src/renderer/assets/desk/cozy/plant.png
3. Vision-Modell checkt:
   - Passt der Stil zur DNA?
   - Stimmen die Farben?
   - Ist der Hintergrund transparent/sauber?
   - Passt die Perspektive zu den anderen Assets?
4. Wenn nein → Prompt wird angepasst → nochmal generieren
5. Wenn ja → fertig, naechstes Asset
```

### Ordner-Struktur der fertigen Assets:

```
desktop/src/renderer/assets/desk/
  cozy/
    plant.png
    coffee.png
    pen-holder.png
    photo-frame.png
    books.png
    wall-bg.png
  dreamy/
    plant.png
    orb.png
    gradient-bg.png
  minimal/
    (keine Bild-Assets — nur CSS)
```

### Quality-Check via Vision (automatisch im Script):

Das Script schickt das generierte Bild an GPT-4.1 (Vision-faehig) mit der Frage:
- "Passt dieses Bild zu folgender Style DNA: [...]?"
- "Ist der Hintergrund sauber freigestellt?"
- "Welche Verbesserungen wuerdest du vorschlagen?"

Wenn die Antwort negativ ist → neuer Versuch mit angepasstem Prompt.
Max 3 Versuche pro Asset, dann manuelle Entscheidung.

---

## Schritt 5: Assets ins Desk-System einbauen

### Was sich aendert:

**`desk-theme.ts` (Typ-Erweiterung)**
```typescript
// Neues Feld pro Theme
assetPath?: string  // z.B. 'cozy' → laedt aus assets/desk/cozy/
```

**`DeskDecorations.tsx` (Renderer-Update)**
```typescript
// Statt nur SVG-Komponenten:
case 'plant':
  return <img
    src={`/assets/desk/${theme.assetPath}/plant.png`}
    alt="Pflanze"
    className="desk-deco desk-anim-sway"
  />
case 'coffee':
  return <img
    src={`/assets/desk/${theme.assetPath}/coffee.png`}
    alt="Kaffee"
    className="desk-deco desk-anim-steam"
  />
```

**`DeskEnvironment.tsx` (Hintergrund-Update)**
```typescript
// backgroundColor → background (Gradient oder Bild)
style={{
  background: theme.assetPath
    ? `url(/assets/desk/${theme.assetPath}/wall-bg.png) center/cover`
    : 'var(--desk-wall-bg)',
}}
```

### Bestehende SVG-Dekos (DeskClock, DeskCalendar)
Bleiben wie sie sind! Die sind interaktiv (klickbar, Echtzeit-Daten).
AI-Assets sind rein dekorativ — kein Klick, keine Daten.

---

## Schritt 6: CSS-Animationen

Die Bilder sind statische PNGs. Bewegung kommt rein durch CSS.

### In `globals.css`:

```css
/* Basis fuer alle Desk-Dekos */
.desk-deco {
  pointer-events: none;
  user-select: none;
  image-rendering: auto;
}

/* Pflanze wackelt sanft */
@keyframes desk-sway {
  0%, 100% { transform: rotate(-1.5deg); }
  50% { transform: rotate(1.5deg); }
}
.desk-anim-sway {
  animation: desk-sway 6s ease-in-out infinite;
  transform-origin: bottom center;
}

/* Kaffee dampft */
@keyframes desk-steam {
  0% { opacity: 0.3; transform: translateY(0) scale(1); }
  50% { opacity: 0.6; transform: translateY(-4px) scale(1.05); }
  100% { opacity: 0.3; transform: translateY(0) scale(1); }
}
.desk-anim-steam {
  position: relative;
}
.desk-anim-steam::after {
  content: '';
  position: absolute;
  top: -8px;
  left: 30%;
  width: 40%;
  height: 12px;
  background: radial-gradient(ellipse, rgba(255,255,255,0.4), transparent);
  animation: desk-steam 4s ease-in-out infinite;
  pointer-events: none;
}

/* Dreamy Schimmern */
@keyframes desk-shimmer {
  0%, 100% { filter: brightness(1) saturate(1); }
  50% { filter: brightness(1.15) saturate(1.2); }
}
.desk-anim-shimmer {
  animation: desk-shimmer 3s ease-in-out infinite;
}

/* Dreamy Schweben */
@keyframes desk-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}
.desk-anim-float {
  animation: desk-float 5s ease-in-out infinite;
}

/* Respektiere reduced-motion */
@media (prefers-reduced-motion: reduce) {
  .desk-anim-sway,
  .desk-anim-steam::after,
  .desk-anim-shimmer,
  .desk-anim-float {
    animation: none;
  }
}
```

---

## Zusammenfassung: Was entsteht

### Neue Dateien
| Datei | Zweck |
|-------|-------|
| `desktop/scripts/asset-gen/generate.ts` | Haupt-Script zum Generieren |
| `desktop/scripts/asset-gen/style-dna.ts` | Master-Stilregeln |
| `desktop/scripts/asset-gen/assets.ts` | Asset-Definitionen + Prompts |
| `desktop/scripts/asset-gen/quality-check.ts` | Vision-Pruefung |
| `desktop/src/renderer/assets/desk/cozy/*.png` | Generierte Cozy-Assets |
| `desktop/src/renderer/assets/desk/dreamy/*.png` | Generierte Dreamy-Assets |

### Geaenderte Dateien
| Datei | Aenderung |
|-------|-----------|
| `package.json` | + `openai` devDependency |
| `desk-theme.ts` | + `assetPath` Feld |
| `desk-themes.ts` | Cozy + Dreamy + Minimal Themes (ersetzen classic-office) |
| `DeskDecorations.tsx` | Rendert `<img>` fuer AI-Assets |
| `DeskEnvironment.tsx` | `background` statt `backgroundColor` |
| `DeskFrame.tsx` | Hintergrund-Bild Support |
| `globals.css` | Animation-Klassen |

### Was NICHT geaendert wird
- DeskClock, DeskCalendar (bleiben SVG, sind interaktiv)
- Sidebar (kommt spaeter mit Frosted Glass)
- Settings (Theme-Picker kommt als separater Schritt)

---

## Kosten-Schaetzung

Pro Asset-Generierung (512x512): ca. $0.02-0.04
Pro Quality-Check (Vision): ca. $0.01-0.02
Bei 10 Assets mit je 2 Versuchen: ca. **$0.60-1.20 total**

Sehr guenstig. Budget fuer die ganze Nacht: ~$5 reichen locker.

---

## Reihenfolge fuer heute Nacht

1. **API Key holen** (du brauchst einen OpenAI Account)
2. **npm install + .env** einrichten
3. **Style DNA + Asset-Definitionen** schreiben
4. **Generator-Script** bauen
5. **Erstes Asset testen** (Pflanze = einfachstes Objekt)
6. **Alle Cozy-Assets** generieren
7. **Ins Desk-System einbauen**
8. **Animationen drauf**
9. **Dreamy-Assets** generieren
10. **Testen, iterieren, fertig**

---

## Offene Fragen fuer Darien

1. Hast du schon einen OpenAI API Key? Falls nicht → https://platform.openai.com
2. Wieviel Budget willst du max. ausgeben? ($5 reichen fuer alles)
3. Sollen wir mit Cozy anfangen (naheliegend) oder Dreamy zuerst?
4. Willst du die generierten Bilder committen oder per .gitignore ignorieren?
   - Empfehlung: **Committen** — damit Luke sie auch hat und kein API-Key braucht
