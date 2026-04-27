/**
 * Background pattern definitions for the Hintergrund-System.
 *
 * Each pattern provides:
 * - SVG-based repeating tile that uses `currentColor` for theme adaptation
 * - Sticker SVG for solid-mode watermarks on UI surfaces
 * - Metadata for the settings picker
 */
import type { LucideIcon } from 'lucide-react'
import {
  Leaf,
  PawPrint,
  Hexagon,
  Waves,
  Star,
  CircleOff,
} from 'lucide-react'

export interface BackgroundPattern {
  id: string
  name: string
  description: string
  icon: LucideIcon
  /** CSS class name for the background layer (defined in background-patterns.css) */
  bgClass: string
  /** CSS class for sticker watermarks on cards in solid mode (or null) */
  stickerClass: string | null
}

export const BACKGROUND_PATTERNS: Record<string, BackgroundPattern> = {
  none: {
    id: 'none',
    name: 'config.backgroundPattern.none.name',
    description: 'config.backgroundPattern.none.description',
    icon: CircleOff,
    bgClass: '',
    stickerClass: null,
  },
  pflanzen: {
    id: 'pflanzen',
    name: 'config.backgroundPattern.pflanzen.name',
    description: 'config.backgroundPattern.pflanzen.description',
    icon: Leaf,
    bgClass: 'bg-pattern-pflanzen',
    stickerClass: 'sticker-pflanzen',
  },
  hunde: {
    id: 'hunde',
    name: 'config.backgroundPattern.hunde.name',
    description: 'config.backgroundPattern.hunde.description',
    icon: PawPrint,
    bgClass: 'bg-pattern-hunde',
    stickerClass: 'sticker-hunde',
  },
  geometrisch: {
    id: 'geometrisch',
    name: 'config.backgroundPattern.geometrisch.name',
    description: 'config.backgroundPattern.geometrisch.description',
    icon: Hexagon,
    bgClass: 'bg-pattern-geometrisch',
    stickerClass: 'sticker-geometrisch',
  },
  wellen: {
    id: 'wellen',
    name: 'config.backgroundPattern.wellen.name',
    description: 'config.backgroundPattern.wellen.description',
    icon: Waves,
    bgClass: 'bg-pattern-wellen',
    stickerClass: 'sticker-wellen',
  },
  abstrakt: {
    id: 'abstrakt',
    name: 'config.backgroundPattern.abstrakt.name',
    description: 'config.backgroundPattern.abstrakt.description',
    icon: Star,
    bgClass: 'bg-pattern-abstrakt',
    stickerClass: 'sticker-abstrakt',
  },
}

export const BACKGROUND_PATTERN_LIST = Object.values(BACKGROUND_PATTERNS)

export type BackgroundPatternId = keyof typeof BACKGROUND_PATTERNS
