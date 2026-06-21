/**
 * Core block types shared across every document surface. Modules extend their own
 * block union with these (plus module-specific blocks like chart/kpi for berichte
 * or toggle/code for wiki). Shapes are intentionally minimal and stable.
 */
import type { DocBlockBase } from '../types'

export interface HeadingBlock extends DocBlockBase {
  type: 'heading'
  level: 1 | 2
  text: string
}

/** Rich text (TipTap HTML). The long-form writing block. */
export interface TextBlock extends DocBlockBase {
  type: 'text'
  html: string
}

export interface BulletBlock extends DocBlockBase {
  type: 'bullet'
  items: string[]
  ordered?: boolean
}

export type CalloutVariant = 'info' | 'success' | 'warning' | 'recommendation'

export interface CalloutBlock extends DocBlockBase {
  type: 'callout'
  variant: CalloutVariant
  title?: string
  html: string
}

export interface DividerBlock extends DocBlockBase {
  type: 'divider'
}

export interface ImageBlock extends DocBlockBase {
  type: 'image'
  url: string
  caption?: string
  alt?: string
}

/** The union of all core block types. */
export type CoreBlock =
  | HeadingBlock
  | TextBlock
  | BulletBlock
  | CalloutBlock
  | DividerBlock
  | ImageBlock
