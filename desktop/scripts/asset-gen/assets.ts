/**
 * Asset-Definitionen — was generiert werden soll.
 * Jedes Asset hat einen Namen, Prompt, Groesse und optionale Animations-Info.
 */

export interface AssetDefinition {
  name: string
  theme: 'cozy' | 'dreamy'
  /** Beschreibung des Objekts (wird mit Style DNA kombiniert) */
  prompt: string
  /** Bildgroesse */
  size: '1024x1024' | '1536x1024' | '1024x1536'
  /** Ausgabe-Dateiname (ohne .png) */
  outputFile: string
  /** CSS-Animation die spaeter draufkommt */
  animation: AnimationHint | null
}

export interface AnimationHint {
  type: 'sway' | 'steam' | 'shimmer' | 'float'
  cssClass: string
  description: string
}

export const ASSETS: Record<string, AssetDefinition> = {
  // ========== COZY THEME ==========
  'cozy-plant': {
    name: 'Topfpflanze',
    theme: 'cozy',
    prompt:
      'A small potted succulent plant in a warm terracotta pot, sitting on a tiny round wooden coaster. The plant has thick green-grey leaves with subtle teal tints. Simple, charming, single object only.',
    size: '1024x1024',
    outputFile: 'cozy/plant',
    animation: {
      type: 'sway',
      cssClass: 'desk-anim-sway',
      description: 'Gentle swaying like a breeze, pivot at bottom center',
    },
  },

  'cozy-coffee': {
    name: 'Kaffeetasse',
    theme: 'cozy',
    prompt:
      'A warm cup of coffee in a handmade ceramic mug, cream-white with a subtle teal glaze ring. Slight wisp of steam rising. Sitting on a small round saucer. Cozy morning feeling. Single object only.',
    size: '1024x1024',
    outputFile: 'cozy/coffee',
    animation: {
      type: 'steam',
      cssClass: 'desk-anim-steam',
      description: 'Subtle steam wisps rising from the cup',
    },
  },

  'cozy-pen-holder': {
    name: 'Stiftebecher',
    theme: 'cozy',
    prompt:
      'A wooden pen holder cup made of light oak wood, containing 3 colored pencils (sage green, dusty rose, warm yellow) and one black fountain pen. Simple, clean arrangement. Single object only.',
    size: '1024x1024',
    outputFile: 'cozy/pen-holder',
    animation: null,
  },

  'cozy-photo-frame': {
    name: 'Bilderrahmen',
    theme: 'cozy',
    prompt:
      'A small wooden photo frame in light oak, showing a soft blurred landscape (green hills, blue sky). Frame is slightly tilted to the right. Warm and personal feeling. Single object only.',
    size: '1024x1024',
    outputFile: 'cozy/photo-frame',
    animation: null,
  },

  'cozy-books': {
    name: 'Buecherstapel',
    theme: 'cozy',
    prompt:
      'A small stack of 3 hardcover books in muted pastel colors: one cream, one sage green, one dusty rose. Books are slightly offset from each other. Clean and minimal. Single object only.',
    size: '1024x1024',
    outputFile: 'cozy/books',
    animation: null,
  },

  // ========== DREAMY THEME ==========
  'dreamy-plant': {
    name: 'Kristall-Pflanze',
    theme: 'dreamy',
    prompt:
      'A magical crystal plant growing from a translucent purple glass pot. The leaves are crystalline, glowing softly in teal and lavender. Ethereal and fantasy-inspired but elegant. Single object only.',
    size: '1024x1024',
    outputFile: 'dreamy/plant',
    animation: {
      type: 'shimmer',
      cssClass: 'desk-anim-shimmer',
      description: 'Gentle brightness/saturation pulse like inner glow',
    },
  },

  'dreamy-orb': {
    name: 'Leucht-Kugel',
    theme: 'dreamy',
    prompt:
      'A floating translucent glass sphere with swirling soft pastel mist inside (purple, teal, gold). Gentle inner glow. Magical and dreamy. Hovering slightly above surface. Single object only.',
    size: '1024x1024',
    outputFile: 'dreamy/orb',
    animation: {
      type: 'float',
      cssClass: 'desk-anim-float',
      description: 'Gentle floating up and down motion',
    },
  },
}

/** Alle Asset-Keys als Array */
export const ALL_ASSET_KEYS = Object.keys(ASSETS)

/** Nur Cozy-Assets */
export const COZY_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'cozy')

/** Nur Dreamy-Assets */
export const DREAMY_ASSET_KEYS = ALL_ASSET_KEYS.filter((k) => ASSETS[k].theme === 'dreamy')
