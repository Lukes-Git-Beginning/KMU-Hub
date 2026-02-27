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
  Sparkles,
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
    name: 'Keine',
    description: 'Kein Hintergrundmuster',
    icon: CircleOff,
    bgClass: '',
    stickerClass: null,
  },
  pflanzen: {
    id: 'pflanzen',
    name: 'Pflanzen',
    description: 'Botanische Blaetter und Ranken',
    icon: Leaf,
    bgClass: 'bg-pattern-pflanzen',
    stickerClass: 'sticker-pflanzen',
  },
  hunde: {
    id: 'hunde',
    name: 'Hunde',
    description: 'Pfotenabdruecke und Knochen',
    icon: PawPrint,
    bgClass: 'bg-pattern-hunde',
    stickerClass: 'sticker-hunde',
  },
  geometrisch: {
    id: 'geometrisch',
    name: 'Geometrisch',
    description: 'Dreiecke und Sechsecke',
    icon: Hexagon,
    bgClass: 'bg-pattern-geometrisch',
    stickerClass: 'sticker-geometrisch',
  },
  wellen: {
    id: 'wellen',
    name: 'Wellen',
    description: 'Fliessende Wellenlinien',
    icon: Waves,
    bgClass: 'bg-pattern-wellen',
    stickerClass: 'sticker-wellen',
  },
  abstrakt: {
    id: 'abstrakt',
    name: 'Abstrakt',
    description: 'Weiche Formen und Kreise',
    icon: Sparkles,
    bgClass: 'bg-pattern-abstrakt',
    stickerClass: 'sticker-abstrakt',
  },
}

export const BACKGROUND_PATTERN_LIST = Object.values(BACKGROUND_PATTERNS)

export type BackgroundPatternId = keyof typeof BACKGROUND_PATTERNS
