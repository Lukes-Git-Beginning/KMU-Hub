/**
 * Style DNA — Master-Stilregeln fuer alle generierten Assets.
 * Wird in jeden Prompt eingebaut damit alles konsistent aussieht.
 */

export const STYLE_DNA = {
  /** Basis-Stil der in JEDEM Prompt dabei ist */
  baseStyle: [
    'clean modern illustration style',
    'soft pastel color palette with warm tones',
    'warm natural lighting from upper left',
    'subtle soft shadows, no harsh contrasts',
    'slightly stylized, NOT photorealistic',
    'european cozy workspace aesthetic',
    'high detail but minimalistic composition',
    'isolated object on pure transparent background',
    'no background clutter, no extra objects',
    'consistent gentle rounded forms',
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

  /** Cozy-Theme spezifisch */
  cozyStyle: [
    'warm homey feeling',
    'natural materials: wood, ceramic, terracotta, linen',
    'muted earth tones with subtle teal accents',
    'handcrafted artisan quality',
  ],

  /** Dreamy-Theme spezifisch */
  dreamyStyle: [
    'magical ethereal atmosphere',
    'soft purple and teal gradient tones',
    'translucent glass-like materials',
    'subtle inner glow effects',
    'fantasy-inspired but still elegant',
  ],
}

/** Baut den vollstaendigen Style-Teil eines Prompts */
export function buildStylePrompt(theme: 'cozy' | 'dreamy'): string {
  const parts = [
    ...STYLE_DNA.baseStyle,
    `Color palette: ${STYLE_DNA.colors.primary}, ${STYLE_DNA.colors.accents}. Avoid: ${STYLE_DNA.colors.avoid}.`,
    `Lighting: ${STYLE_DNA.lighting}`,
    ...(theme === 'cozy' ? STYLE_DNA.cozyStyle : STYLE_DNA.dreamyStyle),
  ]
  return parts.join('. ') + '.'
}
