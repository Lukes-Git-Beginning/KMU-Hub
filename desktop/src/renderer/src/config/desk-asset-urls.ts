/**
 * Asset URL mappings for desk decorations.
 *
 * Decoration assets imported statically so Vite can hash and bundle them.
 * Background/room assets will be added in Batch E (asset generation phase).
 *
 * Theme associations updated for the 5-theme system:
 * cozy, dreamy, raumstation, clean (minimal has no decos).
 */

// ── DECORATIONS: Shared ─────────────────────────────────

import decoCoffeeCup from '@/../assets/desk/shared/coffee-cup.png'
import decoPenHolder from '@/../assets/desk/shared/pen-holder.png'
import decoNotebook from '@/../assets/desk/shared/notebook.png'
import decoBookStack from '@/../assets/desk/shared/book-stack.png'
import decoPhotoFrame from '@/../assets/desk/shared/photo-frame.png'
import decoArtPrint from '@/../assets/desk/shared/art-print.png'

// ── DECORATIONS: Cozy ───────────────────────────────────

import decoSucculent from '@/../assets/desk/cozy/succulent.png'
import decoPottedFern from '@/../assets/desk/cozy/potted-fern.png'
import decoTeaCup from '@/../assets/desk/cozy/tea-cup.png'
import decoCandle from '@/../assets/desk/cozy/candle.png'
import decoSnowGlobe from '@/../assets/desk/cozy/snow-globe.png'
import decoPostcard from '@/../assets/desk/cozy/postcard.png'
import decoReadingGlasses from '@/../assets/desk/cozy/reading-glasses.png'
import decoOpenBook from '@/../assets/desk/cozy/open-book.png'
import decoFlowerVase from '@/../assets/desk/cozy/flower-vase.png'
import decoKnittedCozy from '@/../assets/desk/cozy/knitted-cozy.png'
import decoHoneyJar from '@/../assets/desk/cozy/honey-jar.png'

// ── DECORATIONS: Dreamy ─────────────────────────────────

import decoCrystalPlant from '@/../assets/desk/dreamy/crystal-plant.png'
import decoCrystalOrb from '@/../assets/desk/dreamy/crystal-orb.png'
import decoMoonLamp from '@/../assets/desk/dreamy/moon-lamp.png'
import decoEnchantedHourglass from '@/../assets/desk/dreamy/enchanted-hourglass.png'
import decoStarWand from '@/../assets/desk/dreamy/star-wand.png'
import decoFloatingCrystals from '@/../assets/desk/dreamy/floating-crystals.png'

// ── DECORATIONS: Raumstation (industrial/sci-fi) ────────

import decoVintageCompass from '@/../assets/desk/industrial/vintage-compass.png'
import decoMetalGear from '@/../assets/desk/industrial/metal-gear.png'

// ── EXPORTS ─────────────────────────────────────────────

export type AnimationClass =
  | 'desk-anim-sway'
  | 'desk-anim-steam'
  | 'desk-anim-shimmer'
  | 'desk-anim-float'
  | 'desk-anim-flicker'
  | 'desk-anim-glow'
  | 'desk-anim-sparkle'
  | 'desk-anim-paint-drip'
  | null

export interface DecoAsset {
  id: string
  name: string
  image: string
  themes: string[]
  animation: AnimationClass
}

/** All themes that support decorations (excludes minimal). */
const ALL_DECO_THEMES = ['cozy', 'dreamy', 'raumstation', 'clean']

export const DECO_ASSETS: DecoAsset[] = [
  // Shared (all non-minimal themes)
  { id: 'coffee-cup', name: 'config.decoAsset.coffee-cup.name', image: decoCoffeeCup, themes: ALL_DECO_THEMES, animation: 'desk-anim-steam' },
  { id: 'pen-holder', name: 'config.decoAsset.pen-holder.name', image: decoPenHolder, themes: ALL_DECO_THEMES, animation: null },
  { id: 'notebook', name: 'config.decoAsset.notebook.name', image: decoNotebook, themes: ALL_DECO_THEMES, animation: null },
  { id: 'book-stack', name: 'config.decoAsset.book-stack.name', image: decoBookStack, themes: ALL_DECO_THEMES, animation: null },
  { id: 'photo-frame', name: 'config.decoAsset.photo-frame.name', image: decoPhotoFrame, themes: ALL_DECO_THEMES, animation: null },
  { id: 'art-print', name: 'config.decoAsset.art-print.name', image: decoArtPrint, themes: ALL_DECO_THEMES, animation: null },

  // Cozy exclusive
  { id: 'succulent', name: 'config.decoAsset.succulent.name', image: decoSucculent, themes: ['cozy'], animation: 'desk-anim-sway' },
  { id: 'potted-fern', name: 'config.decoAsset.potted-fern.name', image: decoPottedFern, themes: ['cozy'], animation: 'desk-anim-sway' },
  { id: 'tea-cup', name: 'config.decoAsset.tea-cup.name', image: decoTeaCup, themes: ['cozy'], animation: 'desk-anim-steam' },
  { id: 'candle', name: 'config.decoAsset.candle.name', image: decoCandle, themes: ['cozy'], animation: 'desk-anim-flicker' },
  { id: 'snow-globe', name: 'config.decoAsset.snow-globe.name', image: decoSnowGlobe, themes: ['cozy'], animation: 'desk-anim-shimmer' },
  { id: 'postcard', name: 'config.decoAsset.postcard.name', image: decoPostcard, themes: ['cozy'], animation: null },
  { id: 'reading-glasses', name: 'config.decoAsset.reading-glasses.name', image: decoReadingGlasses, themes: ['cozy'], animation: null },
  { id: 'open-book', name: 'config.decoAsset.open-book.name', image: decoOpenBook, themes: ['cozy'], animation: null },
  { id: 'flower-vase', name: 'config.decoAsset.flower-vase.name', image: decoFlowerVase, themes: ['cozy'], animation: null },
  { id: 'knitted-cozy', name: 'config.decoAsset.knitted-cozy.name', image: decoKnittedCozy, themes: ['cozy'], animation: null },
  { id: 'honey-jar', name: 'config.decoAsset.honey-jar.name', image: decoHoneyJar, themes: ['cozy'], animation: null },

  // Dreamy exclusive
  { id: 'crystal-plant', name: 'config.decoAsset.crystal-plant.name', image: decoCrystalPlant, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'crystal-orb', name: 'config.decoAsset.crystal-orb.name', image: decoCrystalOrb, themes: ['dreamy'], animation: 'desk-anim-float' },
  { id: 'moon-lamp', name: 'config.decoAsset.moon-lamp.name', image: decoMoonLamp, themes: ['dreamy'], animation: 'desk-anim-glow' },
  { id: 'enchanted-hourglass', name: 'config.decoAsset.enchanted-hourglass.name', image: decoEnchantedHourglass, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'star-wand', name: 'config.decoAsset.star-wand.name', image: decoStarWand, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'floating-crystals', name: 'config.decoAsset.floating-crystals.name', image: decoFloatingCrystals, themes: ['dreamy'], animation: 'desk-anim-float' },

  // Raumstation (tech/sci-fi)
  { id: 'vintage-compass', name: 'config.decoAsset.vintage-compass.name', image: decoVintageCompass, themes: ['raumstation'], animation: 'desk-anim-sparkle' },
  { id: 'metal-gear', name: 'config.decoAsset.metal-gear.name', image: decoMetalGear, themes: ['raumstation'], animation: null },
]

/** Get decoration assets available for a specific theme */
export function getDecosForTheme(themeId: string): DecoAsset[] {
  return DECO_ASSETS.filter((d) => d.themes.includes(themeId))
}
