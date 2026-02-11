/**
 * Asset URL mappings for desk decorations.
 *
 * Decoration assets imported statically so Vite can hash and bundle them.
 * Background/room assets will be added in Batch E (asset generation phase).
 *
 * Theme associations updated for the 6-theme system:
 * cozy, dreamy, nature, raumstation, atelier (minimal has no decos).
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

// ── DECORATIONS: Nature ─────────────────────────────────

import decoBonsai from '@/../assets/desk/nature/bonsai.png'
import decoMossTerrarium from '@/../assets/desk/nature/moss-terrarium.png'
import decoPineCone from '@/../assets/desk/nature/pine-cone.png'
import decoCarvedFox from '@/../assets/desk/nature/carved-fox.png'
import decoForestLantern from '@/../assets/desk/nature/forest-lantern.png'
import decoDriedHerbs from '@/../assets/desk/nature/dried-herbs.png'

// ── DECORATIONS: Industrial (reused for Raumstation + Atelier) ──

import decoDeskLamp from '@/../assets/desk/industrial/desk-lamp.png'
import decoConcretePlanter from '@/../assets/desk/industrial/concrete-planter.png'
import decoVintageCompass from '@/../assets/desk/industrial/vintage-compass.png'
import decoMetalGear from '@/../assets/desk/industrial/metal-gear.png'
import decoBlueprintRoll from '@/../assets/desk/industrial/blueprint-roll.png'

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
const ALL_DECO_THEMES = ['cozy', 'dreamy', 'nature', 'raumstation', 'atelier']

export const DECO_ASSETS: DecoAsset[] = [
  // Shared (all non-minimal themes)
  { id: 'coffee-cup', name: 'Kaffeetasse', image: decoCoffeeCup, themes: ALL_DECO_THEMES, animation: 'desk-anim-steam' },
  { id: 'pen-holder', name: 'Stiftebecher', image: decoPenHolder, themes: ALL_DECO_THEMES, animation: null },
  { id: 'notebook', name: 'Notizbuch', image: decoNotebook, themes: ALL_DECO_THEMES, animation: null },
  { id: 'book-stack', name: 'Buecherstapel', image: decoBookStack, themes: ALL_DECO_THEMES, animation: null },
  { id: 'photo-frame', name: 'Bilderrahmen', image: decoPhotoFrame, themes: ALL_DECO_THEMES, animation: null },
  { id: 'art-print', name: 'Kunstdruck', image: decoArtPrint, themes: ALL_DECO_THEMES, animation: null },

  // Cozy exclusive
  { id: 'succulent', name: 'Sukkulente', image: decoSucculent, themes: ['cozy'], animation: 'desk-anim-sway' },
  { id: 'potted-fern', name: 'Topffarn', image: decoPottedFern, themes: ['cozy'], animation: 'desk-anim-sway' },
  { id: 'tea-cup', name: 'Teetasse', image: decoTeaCup, themes: ['cozy'], animation: 'desk-anim-steam' },
  { id: 'candle', name: 'Kerze', image: decoCandle, themes: ['cozy'], animation: 'desk-anim-flicker' },
  { id: 'snow-globe', name: 'Schneekugel', image: decoSnowGlobe, themes: ['cozy'], animation: 'desk-anim-shimmer' },
  { id: 'postcard', name: 'Postkarte', image: decoPostcard, themes: ['cozy'], animation: null },
  { id: 'reading-glasses', name: 'Lesebrille', image: decoReadingGlasses, themes: ['cozy'], animation: null },
  { id: 'open-book', name: 'Offenes Buch', image: decoOpenBook, themes: ['cozy'], animation: null },
  { id: 'flower-vase', name: 'Blumenvase', image: decoFlowerVase, themes: ['cozy'], animation: null },
  { id: 'knitted-cozy', name: 'Strick-Untersetzer', image: decoKnittedCozy, themes: ['cozy'], animation: null },
  { id: 'honey-jar', name: 'Honigglas', image: decoHoneyJar, themes: ['cozy'], animation: null },

  // Dreamy exclusive
  { id: 'crystal-plant', name: 'Kristall-Pflanze', image: decoCrystalPlant, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'crystal-orb', name: 'Kristallkugel', image: decoCrystalOrb, themes: ['dreamy'], animation: 'desk-anim-float' },
  { id: 'moon-lamp', name: 'Mondlampe', image: decoMoonLamp, themes: ['dreamy'], animation: 'desk-anim-glow' },
  { id: 'enchanted-hourglass', name: 'Magische Sanduhr', image: decoEnchantedHourglass, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'star-wand', name: 'Kristall-Stab', image: decoStarWand, themes: ['dreamy'], animation: 'desk-anim-shimmer' },
  { id: 'floating-crystals', name: 'Schwebende Kristalle', image: decoFloatingCrystals, themes: ['dreamy'], animation: 'desk-anim-float' },

  // Nature exclusive
  { id: 'bonsai', name: 'Bonsai', image: decoBonsai, themes: ['nature'], animation: 'desk-anim-sway' },
  { id: 'moss-terrarium', name: 'Moos-Terrarium', image: decoMossTerrarium, themes: ['nature'], animation: null },
  { id: 'pine-cone', name: 'Kiefernzapfen', image: decoPineCone, themes: ['nature'], animation: null },
  { id: 'carved-fox', name: 'Geschnitzter Fuchs', image: decoCarvedFox, themes: ['nature'], animation: null },
  { id: 'forest-lantern', name: 'Waldlaterne', image: decoForestLantern, themes: ['nature'], animation: 'desk-anim-flicker' },
  { id: 'dried-herbs', name: 'Kraeuterbund', image: decoDriedHerbs, themes: ['nature'], animation: 'desk-anim-sway' },

  // Raumstation (tech/sci-fi — reuses some industrial assets + shares compass/gear)
  { id: 'vintage-compass', name: 'Navigations-Kompass', image: decoVintageCompass, themes: ['raumstation'], animation: 'desk-anim-sparkle' },
  { id: 'metal-gear', name: 'Deko-Zahnrad', image: decoMetalGear, themes: ['raumstation'], animation: null },

  // Atelier (creative workshop — reuses lamp, planter, blueprint)
  { id: 'desk-lamp', name: 'Atelier-Lampe', image: decoDeskLamp, themes: ['atelier'], animation: 'desk-anim-glow' },
  { id: 'concrete-planter', name: 'Beton-Pflanzgefaess', image: decoConcretePlanter, themes: ['atelier'], animation: null },
  { id: 'blueprint-roll', name: 'Bauplan-Rolle', image: decoBlueprintRoll, themes: ['atelier'], animation: null },
]

/** Get decoration assets available for a specific theme */
export function getDecosForTheme(themeId: string): DecoAsset[] {
  return DECO_ASSETS.filter((d) => d.themes.includes(themeId))
}
