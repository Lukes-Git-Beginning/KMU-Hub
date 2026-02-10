/**
 * Style DNA — Master-Stilregeln fuer alle generierten Assets.
 * Wird in jeden Prompt eingebaut damit alles konsistent aussieht.
 *
 * WICHTIG: Diese Regeln definieren das visuelle Design-System.
 * Aenderungen hier beeinflussen ALLE zukuenftig generierten Assets.
 */

export type ThemeStyle = 'cozy' | 'dreamy' | 'nature' | 'industrial'

export const STYLE_DNA = {
  /** Basis-Stil fuer DEKO-OBJEKTE (freigestellt, transparent) */
  objectStyle: [
    'clean modern illustration style',
    'soft pastel color palette with warm tones',
    'warm natural lighting from upper left',
    'subtle soft shadows, no harsh contrasts',
    'slightly stylized, NOT photorealistic',
    'high detail but minimalistic composition',
    'MUST be on a completely transparent background with NO floor, NO surface, NO shadow on ground',
    'object is cleanly cut out as if photographed on green screen then removed',
    'no background clutter, no extra objects, no environment',
    'consistent gentle rounded forms',
    'the object should look like a sticker or cutout that can be placed on any surface',
  ],

  /** Basis-Stil fuer HINTERGRUND-TEXTUREN (fullscreen, keine Transparenz) */
  backgroundStyle: [
    'seamless tileable texture or full scene background',
    'high resolution, photographic quality with slight stylization',
    'warm natural lighting',
    'suitable as desktop application background',
    'no text, no UI elements, no objects in foreground',
  ],

  /** Basis-Stil fuer PREVIEW-BILDER (Theme-Picker Karten) */
  previewStyle: [
    'miniature desk scene overview, bird-eye perspective',
    'shows the overall mood and color palette of the theme',
    'warm and inviting, like a magazine photo of a styled workspace',
    'slightly blurred/soft to work as a small preview card',
    'square format, centered composition',
  ],

  /** Farben passend zur KMU Hub Figma-Palette */
  colors: {
    primary: 'warm teal (#1e7e74)',
    wood: 'warm oak brown, light honey tone',
    accents: 'sage green, dusty rose, soft cream, warm beige',
    avoid: 'neon colors, pure black, cold blue, oversaturated colors',
  },

  /** Licht-Setup */
  lighting: 'soft diffused daylight from upper left, gentle ambient occlusion shadows, warm color temperature',

  /** Dark-Mode Modifikator — wird an Light-Prompts angehaengt */
  darkModeModifier: [
    'DARK MODE VERSION: same composition and style but with dark, moody atmosphere',
    'much darker overall, as if the room lights are dimmed to evening/night mode',
    'rich deep shadows, warm accent lighting (desk lamp glow, monitor backlight)',
    'dark surfaces with subtle warm highlights',
    'keep the same materials and textures but in their nighttime appearance',
    'cozy evening atmosphere, not cold or sterile',
  ],

  // ============================================================
  // THEME-SPEZIFISCHE STILE
  // ============================================================

  /** Cozy — Warmes Home-Office */
  cozyStyle: [
    'warm homey feeling, like a real cozy home office',
    'natural materials: light oak wood, ceramic, terracotta, linen, cotton',
    'muted earth tones with subtle teal accents',
    'handcrafted artisan quality, slightly imperfect warmth',
    'warm beige and cream base colors',
    'feeling of soft afternoon sunlight through a window',
  ],

  /** Dreamy — Magisch/Kreativ */
  dreamyStyle: [
    'magical ethereal atmosphere, enchanted workspace',
    'soft purple and teal gradient tones throughout',
    'translucent glass-like and crystal materials',
    'subtle inner glow effects, bioluminescent hints',
    'fantasy-inspired but still elegant and professional',
    'dreamy pastel fog, floating particles of light',
    'aurora-like color transitions',
  ],

  /** Nature — Wald/Gruen */
  natureStyle: [
    'deep forest atmosphere, like an office in a woodland cabin',
    'natural materials: dark walnut wood, rough stone, moss, dried leaves',
    'deep greens, charcoal, rich earth browns, olive tones',
    'organic textures with visible natural grain and imperfections',
    'handcrafted rustic lodge quality',
    'dappled forest light filtering through trees',
    'grounding, calming, connected to nature',
  ],

  /** Industrial — Loft/Urban */
  industrialStyle: [
    'modern urban loft aesthetic, converted warehouse workspace',
    'materials: exposed concrete, brushed dark steel, copper accents, matte black metal',
    'charcoal and warm gray base, amber accent lighting',
    'clean geometric forms, visible raw materials and bolts',
    'professional workshop quality, utilitarian elegance',
    'warm Edison bulb lighting against cold concrete',
    'mix of rough industrial surfaces with refined details',
  ],
}

/** Baut den Style-Prompt fuer Deko-Objekte (freigestellt) */
export function buildObjectPrompt(theme: ThemeStyle): string {
  const themeStyles: Record<ThemeStyle, string[]> = {
    cozy: STYLE_DNA.cozyStyle,
    dreamy: STYLE_DNA.dreamyStyle,
    nature: STYLE_DNA.natureStyle,
    industrial: STYLE_DNA.industrialStyle,
  }
  const parts = [
    ...STYLE_DNA.objectStyle,
    `Color palette: ${STYLE_DNA.colors.primary}, ${STYLE_DNA.colors.accents}. Avoid: ${STYLE_DNA.colors.avoid}.`,
    `Lighting: ${STYLE_DNA.lighting}`,
    ...themeStyles[theme],
  ]
  return parts.join('. ') + '.'
}

/** Baut den Style-Prompt fuer Hintergrund-Texturen */
export function buildBackgroundPrompt(theme: ThemeStyle, isDark: boolean): string {
  const themeStyles: Record<ThemeStyle, string[]> = {
    cozy: STYLE_DNA.cozyStyle,
    dreamy: STYLE_DNA.dreamyStyle,
    nature: STYLE_DNA.natureStyle,
    industrial: STYLE_DNA.industrialStyle,
  }
  const parts = [
    ...STYLE_DNA.backgroundStyle,
    ...themeStyles[theme],
    ...(isDark ? STYLE_DNA.darkModeModifier : []),
  ]
  return parts.join('. ') + '.'
}

/** Baut den Style-Prompt fuer Preview-Bilder */
export function buildPreviewPrompt(theme: ThemeStyle): string {
  const themeStyles: Record<ThemeStyle, string[]> = {
    cozy: STYLE_DNA.cozyStyle,
    dreamy: STYLE_DNA.dreamyStyle,
    nature: STYLE_DNA.natureStyle,
    industrial: STYLE_DNA.industrialStyle,
  }
  const parts = [
    ...STYLE_DNA.previewStyle,
    ...themeStyles[theme],
  ]
  return parts.join('. ') + '.'
}

/** Legacy-Kompatibilitaet */
export function buildStylePrompt(theme: ThemeStyle): string {
  return buildObjectPrompt(theme)
}
