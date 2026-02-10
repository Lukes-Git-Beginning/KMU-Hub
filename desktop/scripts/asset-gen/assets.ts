/**
 * Asset-Definitionen — ALLE Assets die generiert werden muessen.
 *
 * 3 Kategorien:
 * - backgrounds: Wand-Texturen, Panel-Texturen, Schreibtisch-Oberflaechen (Light + Dark)
 * - decorations: Freigestellte Deko-Objekte (transparenter Hintergrund!)
 * - previews: Theme-Picker Vorschaubilder
 *
 * WICHTIG fuer Dekos: Jedes Objekt MUSS komplett freigestellt sein
 * (transparent background, kein Boden, kein Schatten auf Untergrund).
 * Das Objekt soll aussehen wie ein ausgeschnittener Sticker.
 */
import type { ThemeStyle } from './style-dna'

export interface AssetDefinition {
  name: string
  /** Welcher Theme-Stil wird fuer den Prompt verwendet */
  theme: ThemeStyle
  /** Kategorie bestimmt welcher Style-Prompt-Builder genutzt wird */
  category: 'background' | 'decoration' | 'preview'
  /** Ist das die Dark-Mode Version? (nur fuer backgrounds) */
  isDark?: boolean
  /** Beschreibung des Objekts (wird mit Style DNA kombiniert) */
  prompt: string
  /** Bildgroesse */
  size: '1024x1024' | '1536x1024' | '1024x1536'
  /** Ausgabe-Dateiname (ohne .png) */
  outputFile: string
  /** CSS-Animation die spaeter draufkommt (nur fuer decorations) */
  animation: AnimationHint | null
}

export interface AnimationHint {
  type: 'sway' | 'steam' | 'shimmer' | 'float' | 'flicker' | 'glow'
  cssClass: string
  description: string
}

// ================================================================
// ANIMATION PRESETS
// ================================================================
const ANIM_SWAY: AnimationHint = { type: 'sway', cssClass: 'desk-anim-sway', description: 'Gentle swaying like a breeze, pivot at bottom center' }
const ANIM_STEAM: AnimationHint = { type: 'steam', cssClass: 'desk-anim-steam', description: 'Subtle steam wisps rising' }
const ANIM_SHIMMER: AnimationHint = { type: 'shimmer', cssClass: 'desk-anim-shimmer', description: 'Brightness/saturation pulse' }
const ANIM_FLOAT: AnimationHint = { type: 'float', cssClass: 'desk-anim-float', description: 'Gentle floating up and down' }
const ANIM_FLICKER: AnimationHint = { type: 'flicker', cssClass: 'desk-anim-flicker', description: 'Candle flame flicker effect' }
const ANIM_GLOW: AnimationHint = { type: 'glow', cssClass: 'desk-anim-glow', description: 'Soft pulsing warm light glow' }

export const ASSETS: Record<string, AssetDefinition> = {

  // ================================================================
  //  COZY THEME — BACKGROUNDS (Light + Dark)
  // ================================================================

  'cozy-wall-light': {
    name: 'Cozy Wand (Light)',
    theme: 'cozy',
    category: 'background',
    prompt: 'A warm room interior wall seen straight-on. Soft cream-beige painted wall with subtle plaster texture. On the left side, warm golden sunlight streams through an unseen window, casting soft light gradients across the wall. The wall transitions from warm bright (left, where the window light hits) to slightly shadowed warm beige (right). Faint shadow of window frame mullions visible on the wall. This is the background of a cozy home office — the app window will sit in front of this wall like a monitor on a desk. The feel should be: you are sitting at a desk looking at a warm sunlit wall.',
    size: '1536x1024',
    outputFile: 'cozy/wall-light',
    animation: null,
  },
  'cozy-wall-dark': {
    name: 'Cozy Wand (Dark)',
    theme: 'cozy',
    category: 'background',
    isDark: true,
    prompt: 'A warm room interior wall at night. Same cream-beige painted wall but in evening darkness. Warm amber glow from a desk lamp creates a pool of warm light on the left side. The rest fades into rich dark shadows. Subtle warm highlights on the wall texture. Cozy evening atmosphere, not cold. This is the background of a cozy home office at night.',
    size: '1536x1024',
    outputFile: 'cozy/wall-dark',
    animation: null,
  },

  'cozy-panel-light': {
    name: 'Cozy Seitenpanel (Light)',
    theme: 'cozy',
    category: 'background',
    prompt: 'A vertical section of a warm room wall with subtle cream-beige plaster texture. Soft natural light from above. This is a narrow vertical panel (side wall of a desk area). Clean, warm, with very subtle texture variation. No objects, no decorations — just the warm wall surface.',
    size: '1024x1536',
    outputFile: 'cozy/panel-light',
    animation: null,
  },
  'cozy-panel-dark': {
    name: 'Cozy Seitenpanel (Dark)',
    theme: 'cozy',
    category: 'background',
    isDark: true,
    prompt: 'A vertical section of a warm room wall at night. Same plaster texture in darkness with warm ambient light from above. Rich shadows, warm undertones. Cozy evening side wall panel.',
    size: '1024x1536',
    outputFile: 'cozy/panel-dark',
    animation: null,
  },

  'cozy-surface-light': {
    name: 'Cozy Schreibtisch (Light)',
    theme: 'cozy',
    category: 'background',
    prompt: 'Top-down view of a light oak wood desk surface. Warm honey-toned wood grain running horizontally. Natural wood imperfections, subtle knots. Smooth polished finish with soft light reflections. The desk looks real and touchable. Clean surface, no objects on it.',
    size: '1536x1024',
    outputFile: 'cozy/surface-light',
    animation: null,
  },
  'cozy-surface-dark': {
    name: 'Cozy Schreibtisch (Dark)',
    theme: 'cozy',
    category: 'background',
    isDark: true,
    prompt: 'Top-down view of a light oak wood desk surface at night. Same wood grain but in dim warm lamp light. Darker overall with warm amber reflections on the polished wood. Rich shadows in the grain. Evening atmosphere.',
    size: '1536x1024',
    outputFile: 'cozy/surface-dark',
    animation: null,
  },

  // ================================================================
  //  DREAMY THEME — BACKGROUNDS (Light + Dark)
  // ================================================================

  'dreamy-wall-light': {
    name: 'Dreamy Hintergrund (Light)',
    theme: 'dreamy',
    category: 'background',
    prompt: 'A magical gradient background for a creative workspace. Soft transition from pale lavender at the top through pastel purple in the middle to soft teal at the bottom. Tiny floating particles of golden light scattered throughout, like dust motes in sunlight. Subtle aurora-like color wisps. Ethereal, dreamy, professional. The app window will float in front of this like a magic portal.',
    size: '1536x1024',
    outputFile: 'dreamy/wall-light',
    animation: null,
  },
  'dreamy-wall-dark': {
    name: 'Dreamy Hintergrund (Dark)',
    theme: 'dreamy',
    category: 'background',
    isDark: true,
    prompt: 'A deep magical gradient background at night. Rich dark purple at the top transitioning to deep teal-black at the bottom. Scattered tiny glowing particles like stars. Subtle bioluminescent wisps of light in purple and teal. Mysterious but inviting. Dark magic evening atmosphere.',
    size: '1536x1024',
    outputFile: 'dreamy/wall-dark',
    animation: null,
  },

  'dreamy-panel-light': {
    name: 'Dreamy Seitenpanel (Light)',
    theme: 'dreamy',
    category: 'background',
    prompt: 'A vertical gradient panel in soft pastel purple and teal. Translucent frosted glass appearance with subtle light particles. Ethereal side panel for a magical workspace.',
    size: '1024x1536',
    outputFile: 'dreamy/panel-light',
    animation: null,
  },
  'dreamy-panel-dark': {
    name: 'Dreamy Seitenpanel (Dark)',
    theme: 'dreamy',
    category: 'background',
    isDark: true,
    prompt: 'A vertical gradient panel in deep dark purple and teal. Translucent dark glass appearance with faint glowing particles. Dark magical side panel.',
    size: '1024x1536',
    outputFile: 'dreamy/panel-dark',
    animation: null,
  },

  'dreamy-surface-light': {
    name: 'Dreamy Schreibtisch (Light)',
    theme: 'dreamy',
    category: 'background',
    prompt: 'Top-down view of a translucent crystalline desk surface. Semi-transparent frosted glass with subtle purple and teal color shifts beneath. Occasional tiny sparkle highlights. Magical and elegant, like a desk made of enchanted ice.',
    size: '1536x1024',
    outputFile: 'dreamy/surface-light',
    animation: null,
  },
  'dreamy-surface-dark': {
    name: 'Dreamy Schreibtisch (Dark)',
    theme: 'dreamy',
    category: 'background',
    isDark: true,
    prompt: 'Top-down view of a dark translucent crystalline desk surface. Deep purple-black glass with faint inner glow in teal. Subtle light refractions. Dark magical desk surface.',
    size: '1536x1024',
    outputFile: 'dreamy/surface-dark',
    animation: null,
  },

  // ================================================================
  //  NATURE THEME — BACKGROUNDS (Light + Dark)
  // ================================================================

  'nature-wall-light': {
    name: 'Nature Wand (Light)',
    theme: 'nature',
    category: 'background',
    prompt: 'A forest cabin interior wall. Dark stacked stone wall with subtle moss growing between some stones. Dappled green-golden light filtering in from the left, as if through trees outside a window. Natural, grounding, earthy. Deep greens and warm browns. The app window will sit in front of this like a monitor in a woodland office.',
    size: '1536x1024',
    outputFile: 'nature/wall-light',
    animation: null,
  },
  'nature-wall-dark': {
    name: 'Nature Wand (Dark)',
    theme: 'nature',
    category: 'background',
    isDark: true,
    prompt: 'A forest cabin interior wall at night. Same stone wall in deep darkness. A warm lantern or fireplace casts amber-orange light from the left. Deep shadows between stones. Moss barely visible. Rich, dark, cozy cabin at night.',
    size: '1536x1024',
    outputFile: 'nature/wall-dark',
    animation: null,
  },

  'nature-panel-light': {
    name: 'Nature Seitenpanel (Light)',
    theme: 'nature',
    category: 'background',
    prompt: 'A vertical section of dark stone wall with subtle moss between stones. Forest cabin side panel. Dappled natural light from above. Earthy, natural texture.',
    size: '1024x1536',
    outputFile: 'nature/panel-light',
    animation: null,
  },
  'nature-panel-dark': {
    name: 'Nature Seitenpanel (Dark)',
    theme: 'nature',
    category: 'background',
    isDark: true,
    prompt: 'A vertical section of dark stone wall at night. Deep shadows, warm ambient firelight. Forest cabin side panel in darkness.',
    size: '1024x1536',
    outputFile: 'nature/panel-dark',
    animation: null,
  },

  'nature-surface-light': {
    name: 'Nature Schreibtisch (Light)',
    theme: 'nature',
    category: 'background',
    prompt: 'Top-down view of a dark walnut wood desk surface. Rich deep brown grain, slightly rough natural finish. Visible wood rings and character marks. Heavy, solid, crafted. Dappled forest light on the surface.',
    size: '1536x1024',
    outputFile: 'nature/surface-light',
    animation: null,
  },
  'nature-surface-dark': {
    name: 'Nature Schreibtisch (Dark)',
    theme: 'nature',
    category: 'background',
    isDark: true,
    prompt: 'Top-down view of a dark walnut wood desk surface at night. Same rich grain but in warm lantern light. Deep shadows in the wood grain. Dark and moody.',
    size: '1536x1024',
    outputFile: 'nature/surface-dark',
    animation: null,
  },

  // ================================================================
  //  INDUSTRIAL THEME — BACKGROUNDS (Light + Dark)
  // ================================================================

  'industrial-wall-light': {
    name: 'Industrial Wand (Light)',
    theme: 'industrial',
    category: 'background',
    prompt: 'A modern loft interior wall. Exposed concrete with form marks and subtle imperfections. A section of aged red-brown brick visible on the left third. Warm daylight from large industrial windows on the left. Clean, urban, professional. The app window will sit in front of this like a monitor in a converted warehouse office.',
    size: '1536x1024',
    outputFile: 'industrial/wall-light',
    animation: null,
  },
  'industrial-wall-dark': {
    name: 'Industrial Wand (Dark)',
    theme: 'industrial',
    category: 'background',
    isDark: true,
    prompt: 'A modern loft interior wall at night. Same concrete and brick but in darkness. Warm Edison bulb amber glow from above casting dramatic shadows. Copper pipe fixtures catching the light. Dark industrial atmosphere with warm accent lighting.',
    size: '1536x1024',
    outputFile: 'industrial/wall-dark',
    animation: null,
  },

  'industrial-panel-light': {
    name: 'Industrial Seitenpanel (Light)',
    theme: 'industrial',
    category: 'background',
    prompt: 'A vertical section of exposed concrete wall with subtle form marks. Clean industrial loft side panel. Daylight from above. Urban, minimal texture.',
    size: '1024x1536',
    outputFile: 'industrial/panel-light',
    animation: null,
  },
  'industrial-panel-dark': {
    name: 'Industrial Seitenpanel (Dark)',
    theme: 'industrial',
    category: 'background',
    isDark: true,
    prompt: 'A vertical section of exposed concrete wall at night. Warm amber accent light from above. Dark industrial side panel with dramatic shadows.',
    size: '1024x1536',
    outputFile: 'industrial/panel-dark',
    animation: null,
  },

  'industrial-surface-light': {
    name: 'Industrial Schreibtisch (Light)',
    theme: 'industrial',
    category: 'background',
    prompt: 'Top-down view of a dark brushed metal desk surface with subtle steel grain. Matte black-gray finish. Two thin copper accent lines running along the edges. Clean, heavy, utilitarian. Warm light reflections.',
    size: '1536x1024',
    outputFile: 'industrial/surface-light',
    animation: null,
  },
  'industrial-surface-dark': {
    name: 'Industrial Schreibtisch (Dark)',
    theme: 'industrial',
    category: 'background',
    isDark: true,
    prompt: 'Top-down view of a dark brushed metal desk surface at night. Same steel grain in warm Edison bulb light. Copper accents glowing warmly. Dark and moody industrial surface.',
    size: '1536x1024',
    outputFile: 'industrial/surface-dark',
    animation: null,
  },

  // ================================================================
  //  PREVIEW-BILDER (Theme-Picker Karten)
  // ================================================================

  'preview-cozy': {
    name: 'Preview: Cozy',
    theme: 'cozy',
    category: 'preview',
    prompt: 'A beautiful miniature bird-eye view of a cozy home office desk setup. Light oak desk, terracotta plant pot with succulent, coffee mug, stack of pastel books, warm afternoon sunlight. Warm beige room. Inviting, hygge feeling. Overview that captures the whole mood at a glance.',
    size: '1024x1024',
    outputFile: 'cozy/preview',
    animation: null,
  },
  'preview-dreamy': {
    name: 'Preview: Dreamy',
    theme: 'dreamy',
    category: 'preview',
    prompt: 'A beautiful miniature bird-eye view of a magical creative desk setup. Translucent crystal desk surface, floating glass orbs, purple-teal gradient atmosphere, tiny light particles, crystal plant. Ethereal and enchanted. Overview that captures the magical mood.',
    size: '1024x1024',
    outputFile: 'dreamy/preview',
    animation: null,
  },
  'preview-minimal': {
    name: 'Preview: Minimal',
    theme: 'cozy',
    category: 'preview',
    prompt: 'A beautiful miniature bird-eye view of an ultra-minimalist desk setup. Clean white-gray desk, nothing on it except a single thin laptop. Lots of white space. Pure, focused, Scandinavian minimalism. Calm and distraction-free. Almost empty but elegant.',
    size: '1024x1024',
    outputFile: 'minimal/preview',
    animation: null,
  },
  'preview-nature': {
    name: 'Preview: Nature',
    theme: 'nature',
    category: 'preview',
    prompt: 'A beautiful miniature bird-eye view of a forest cabin office desk. Dark walnut desk, stone wall background, lots of plants (fern, moss terrarium, bonsai), candle, dark green and brown tones. Dappled forest light. Grounding and natural.',
    size: '1024x1024',
    outputFile: 'nature/preview',
    animation: null,
  },
  'preview-industrial': {
    name: 'Preview: Industrial',
    theme: 'industrial',
    category: 'preview',
    prompt: 'A beautiful miniature bird-eye view of a modern loft office desk. Dark brushed metal desk, concrete wall, copper desk lamp, books, coffee. Edison bulb warm lighting. Urban, professional, warehouse aesthetic.',
    size: '1024x1024',
    outputFile: 'industrial/preview',
    animation: null,
  },

  // ================================================================
  //  SHARED DEKO-OBJEKTE (verfuegbar in mehreren Themes)
  // ================================================================

  'deco-coffee-cup': {
    name: 'Kaffeetasse',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A warm cup of coffee in a handmade cream-white ceramic mug with a subtle teal glaze ring near the rim. The coffee is dark brown with a thin crema. A very faint wisp of steam rises from the cup. Small round saucer beneath. COMPLETELY isolated on transparent background, NO floor, NO table surface, NO ground shadow. Cutout sticker style.',
    size: '1024x1024',
    outputFile: 'shared/coffee-cup',
    animation: ANIM_STEAM,
  },
  'deco-pen-holder': {
    name: 'Stiftebecher',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A wooden pen holder cup made of light oak wood. Inside: 3 colored pencils (sage green, dusty rose, warm yellow) and one black fountain pen. Simple, clean, handcrafted quality. COMPLETELY isolated on transparent background, NO floor, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'shared/pen-holder',
    animation: null,
  },
  'deco-notebook': {
    name: 'Notizbuch',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A closed leather-bound notebook in warm cognac brown. Slightly worn edges showing character. A thin ribbon bookmark peeking out. Elegant and personal. COMPLETELY isolated on transparent background, NO floor, NO shadow. Sticker cutout.',
    size: '1024x1024',
    outputFile: 'shared/notebook',
    animation: null,
  },
  'deco-book-stack': {
    name: 'Buecherstapel',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small stack of 3 hardcover books. Bottom book: cream/ivory. Middle: sage green. Top: dusty rose. Books are slightly offset from each other. Cloth-bound with subtle gold lettering on spines. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow beneath. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'shared/book-stack',
    animation: null,
  },
  'deco-photo-frame': {
    name: 'Bilderrahmen',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small wooden photo frame in light oak. Inside: a soft blurred landscape photo showing green hills and blue sky. Frame is slightly tilted to the right. Warm and personal feeling. COMPLETELY isolated on transparent background, NO wall behind it, NO shadow. Cutout sticker style.',
    size: '1024x1024',
    outputFile: 'shared/photo-frame',
    animation: null,
  },
  'deco-art-print': {
    name: 'Kunstdruck',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small framed abstract art print in a thin black frame. The art inside features warm tones: dusty rose, sage green, and gold abstract shapes on cream background. Modern and tasteful. COMPLETELY isolated on transparent background, NO wall, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'shared/art-print',
    animation: null,
  },

  // ================================================================
  //  COZY-EXKLUSIV (Warmes Home-Office)
  // ================================================================

  'deco-succulent': {
    name: 'Sukkulente',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small potted succulent plant in a warm terracotta pot on a tiny round wooden coaster. Thick green-grey rosette leaves with subtle teal tints at the tips. The pot has a handmade quality. COMPLETELY isolated on transparent background, cleanly cut out with NO floor and NO shadow beneath. The object floats as if photographed on a green screen.',
    size: '1024x1024',
    outputFile: 'cozy/succulent',
    animation: ANIM_SWAY,
  },
  'deco-potted-fern': {
    name: 'Topffarn',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A delicate potted fern in a white ceramic pot with a matte finish. Graceful arching fronds with fine detailed leaves. Fresh green color. COMPLETELY isolated on transparent background, cleanly cut out, NO floor, NO shadow beneath, NO surface. Looks like a sticker.',
    size: '1024x1024',
    outputFile: 'cozy/potted-fern',
    animation: ANIM_SWAY,
  },
  'deco-tea-cup': {
    name: 'Teetasse',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A delicate tea cup in thin white porcelain with a small saucer. Light green tea visible inside. Subtle floral pattern on the cup. A tiny wisp of steam. Elegant and refined. COMPLETELY isolated on transparent background, NO floor, NO shadow beneath. Cutout sticker style.',
    size: '1024x1024',
    outputFile: 'cozy/tea-cup',
    animation: ANIM_STEAM,
  },
  'deco-candle': {
    name: 'Kerze',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small pillar candle in a round ceramic holder. Warm cream-white candle with a lit flame, soft warm glow around the flame. The ceramic holder is terracotta with a matte glaze. Cozy and inviting. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/candle',
    animation: ANIM_FLICKER,
  },
  'deco-snow-globe': {
    name: 'Schneekugel',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small decorative snow globe on a dark wood base. Inside: a miniature winter scene with tiny trees and a cabin. Soft scattered snowflakes suspended in the water. Magical and nostalgic. COMPLETELY isolated on transparent background, NO floor, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/snow-globe',
    animation: ANIM_SHIMMER,
  },
  'deco-postcard': {
    name: 'Postkarte',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A motivational postcard pinned with a small strip of washi tape. The postcard reads "Focus on what matters" in elegant handwritten typography on a cream background with subtle watercolor floral accents. COMPLETELY isolated on transparent background, NO wall, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/postcard',
    animation: null,
  },
  'deco-reading-glasses': {
    name: 'Lesebrille',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A pair of elegant wire-frame reading glasses, folded closed, resting on a small soft cloth. Thin gold metal frames. Intellectual and warm. COMPLETELY isolated on transparent background, NO floor, NO surface shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/reading-glasses',
    animation: null,
  },
  'deco-open-book': {
    name: 'Offenes Buch',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'An open hardcover book lying flat with slightly curled pages. Warm cream-colored pages with visible but unreadable text. The cover is a warm brown leather. Inviting, literary feeling. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/open-book',
    animation: null,
  },
  'deco-flower-vase': {
    name: 'Blumenvase',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small ceramic vase in matte white with dried flowers: a few sprigs of dried lavender (sage-purple), dried pampas grass (cream), and a dried eucalyptus branch (dusty green). Elegant and natural. COMPLETELY isolated on transparent background, NO floor, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/flower-vase',
    animation: null,
  },
  'deco-knitted-cozy': {
    name: 'Strick-Untersetzer',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small hand-knitted mug cozy/coaster in cream-white chunky yarn. Soft cable-knit pattern, slightly irregular handmade charm. The cozy is round, flat, sitting like a thick knitted doily. Warm, tactile, hygge. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/knitted-cozy',
    animation: null,
  },
  'deco-honey-jar': {
    name: 'Honigglas',
    theme: 'cozy',
    category: 'decoration',
    prompt: 'A small glass jar of golden honey with a tiny wooden honey dipper resting across the top. The honey has a warm amber glow. A small handwritten label says "Honig". Rustic and charming. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'cozy/honey-jar',
    animation: null,
  },

  // ================================================================
  //  DREAMY-EXKLUSIV (Magisch/Kreativ)
  // ================================================================

  'deco-crystal-plant': {
    name: 'Kristall-Pflanze',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'A magical crystal plant growing from a translucent purple glass pot. Crystalline leaves that glow softly in teal and lavender. Subtle inner light. Ethereal and fantasy-inspired but elegant. COMPLETELY isolated on transparent background, NO floor, NO shadow. Cleanly cut out sticker-style.',
    size: '1024x1024',
    outputFile: 'dreamy/crystal-plant',
    animation: ANIM_SHIMMER,
  },
  'deco-crystal-orb': {
    name: 'Kristallkugel',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'A floating translucent glass sphere with swirling soft pastel mist inside (purple, teal, gold wisps). Gentle inner glow. Magical and dreamy. The orb hovers slightly above where a surface would be. COMPLETELY isolated on transparent background, NO floor, NO ground, NO shadow beneath. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'dreamy/crystal-orb',
    animation: ANIM_FLOAT,
  },
  'deco-moon-lamp': {
    name: 'Mondlampe',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'A small decorative crescent moon lamp, about the size of a hand. Soft pearlescent white surface with subtle crater textures. Emits a gentle warm golden glow from the inner curve. Sitting on a tiny translucent crystal stand. Magical and serene. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'dreamy/moon-lamp',
    animation: ANIM_GLOW,
  },
  'deco-enchanted-hourglass': {
    name: 'Magische Sanduhr',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'An elegant hourglass with a translucent purple glass frame. Inside, glowing golden sand particles slowly fall, leaving shimmering trails. The sand in the bottom half glows warmly. Delicate, magical, ornate. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'dreamy/enchanted-hourglass',
    animation: ANIM_SHIMMER,
  },
  'deco-star-wand': {
    name: 'Kristall-Stab',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'A decorative crystal wand or staff, about 20cm long, lying at a slight angle. Faceted amethyst-purple crystal at the top, tapering to a thin silver handle. Subtle light refractions along the crystal facets. Elegant, not childish. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'dreamy/star-wand',
    animation: ANIM_SHIMMER,
  },
  'deco-floating-crystals': {
    name: 'Schwebende Kristalle',
    theme: 'dreamy',
    category: 'decoration',
    prompt: 'A small cluster of 4-5 floating gemstones/crystals at different heights. Colors: soft amethyst purple, rose quartz pink, aquamarine teal, and clear crystal. They hover as if suspended by magic, with tiny sparkle points between them. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'dreamy/floating-crystals',
    animation: ANIM_FLOAT,
  },

  // ================================================================
  //  NATURE-EXKLUSIV (Wald/Gruen)
  // ================================================================

  'deco-bonsai': {
    name: 'Bonsai',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A miniature bonsai tree in a dark ceramic shallow bowl. Gnarled trunk with tiny detailed green leaves. Classic windswept bonsai form. Moss covering the soil. COMPLETELY isolated on transparent background, cleanly cut out, NO floor, NO shadow on ground. Sticker-style cutout.',
    size: '1024x1024',
    outputFile: 'nature/bonsai',
    animation: ANIM_SWAY,
  },
  'deco-moss-terrarium': {
    name: 'Moos-Terrarium',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A small glass dome terrarium with lush green moss, a tiny red-capped mushroom, and a small stone inside. The glass dome sits on a dark wood base. Enchanting miniature world. COMPLETELY isolated on transparent background, NO floor, NO surface shadow. Cutout sticker style.',
    size: '1024x1024',
    outputFile: 'nature/moss-terrarium',
    animation: null,
  },
  'deco-pine-cone': {
    name: 'Kiefernzapfen',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A large decorative pine cone sitting on a small round slice of birch wood. The pine cone is dark brown with open scales showing detailed texture. Natural, forest-found quality. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'nature/pine-cone',
    animation: null,
  },
  'deco-carved-fox': {
    name: 'Geschnitzter Fuchs',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A small hand-carved wooden fox figurine, about 8cm tall. Sitting position with bushy tail wrapped around feet. Dark walnut wood with visible grain. Folk-art rustic style, slightly stylized not realistic. Charming woodland character. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'nature/carved-fox',
    animation: null,
  },
  'deco-forest-lantern': {
    name: 'Waldlaterne',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A small rustic iron lantern with a warm flickering candle inside. Dark aged metal frame with glass panels. The candle casts a warm amber glow through the glass. Rustic cabin quality, slightly weathered. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'nature/forest-lantern',
    animation: ANIM_FLICKER,
  },
  'deco-dried-herbs': {
    name: 'Kraeuterbund',
    theme: 'nature',
    category: 'decoration',
    prompt: 'A small bundle of dried herbs tied together with natural twine. Includes dried rosemary, thyme, and sage. Muted greens and browns, slightly curled leaves. Rustic and aromatic-looking. Hanging from a tiny hook as if drying. COMPLETELY isolated on transparent background, NO wall, NO floor, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'nature/dried-herbs',
    animation: ANIM_SWAY,
  },

  // ================================================================
  //  INDUSTRIAL-EXKLUSIV (Loft/Urban)
  // ================================================================

  'deco-desk-lamp': {
    name: 'Industrie-Lampe',
    theme: 'industrial',
    category: 'decoration',
    prompt: 'A small adjustable desk lamp with a brass/copper finish and a warm glowing Edison bulb. Industrial design with visible joints and matte black accents. The bulb emits a warm amber glow. Utilitarian elegance. COMPLETELY isolated on transparent background, NO floor, NO desk surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'industrial/desk-lamp',
    animation: ANIM_GLOW,
  },
  'deco-concrete-planter': {
    name: 'Beton-Pflanzgefaess',
    theme: 'industrial',
    category: 'decoration',
    prompt: 'A small geometric concrete planter (hexagonal shape) with a dark gray matte finish. Inside: a small round cactus with a single tiny pink flower on top. The concrete has visible raw texture and slight imperfections. Modern, urban. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'industrial/concrete-planter',
    animation: null,
  },
  'deco-vintage-compass': {
    name: 'Vintage-Kompass',
    theme: 'industrial',
    category: 'decoration',
    prompt: 'An open antique brass compass with a detailed face showing cardinal directions. The lid is flipped open, revealing the compass rose inside. Warm aged patina on the brass. The needle points north. Adventurous and refined. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'industrial/vintage-compass',
    animation: null,
  },
  'deco-metal-gear': {
    name: 'Deko-Zahnrad',
    theme: 'industrial',
    category: 'decoration',
    prompt: 'A decorative copper/bronze gear wheel standing upright on a small matte black metal stand. The gear has detailed teeth, visible bolt holes, and a warm copper patina with subtle verdigris. Steampunk-inspired desk art. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'industrial/metal-gear',
    animation: null,
  },
  'deco-blueprint-roll': {
    name: 'Bauplan-Rolle',
    theme: 'industrial',
    category: 'decoration',
    prompt: 'A rolled-up blueprint/architectural drawing with a copper clip holding it together. The paper is slightly aged with a blue-white tint. One end slightly unrolled showing faint technical drawing lines. Professional and creative. COMPLETELY isolated on transparent background, NO floor, NO surface, NO shadow. Cutout sticker.',
    size: '1024x1024',
    outputFile: 'industrial/blueprint-roll',
    animation: null,
  },
}

// ================================================================
//  HELPER EXPORTS
// ================================================================

/** Alle Asset-Keys */
export const ALL_ASSET_KEYS = Object.keys(ASSETS)

/** Filtern nach Kategorie */
export const BACKGROUND_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].category === 'background')
export const DECORATION_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].category === 'decoration')
export const PREVIEW_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].category === 'preview')

/** Filtern nach Theme */
export const COZY_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'cozy')
export const DREAMY_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'dreamy')
export const NATURE_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'nature')
export const INDUSTRIAL_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'industrial')

/** Nur Light-Backgrounds */
export const LIGHT_BG_KEYS = BACKGROUND_KEYS.filter((k) => !ASSETS[k].isDark)
/** Nur Dark-Backgrounds */
export const DARK_BG_KEYS = BACKGROUND_KEYS.filter((k) => ASSETS[k].isDark)
