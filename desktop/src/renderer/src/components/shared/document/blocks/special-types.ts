/**
 * Special block types — the rich, structural elements that sit between the core
 * prose blocks (text/heading/bullet/callout/image/divider). Built once in
 * SpecialBlocks.tsx, registered à la carte per surface: a wiki article wants the
 * full set; a printed report wants the ones that make sense on paper.
 *
 * Shapes stay minimal and serialisable (plain JSON) so a document round-trips
 * through the mock store unchanged.
 */
import type { DocBlockBase } from '../types'

/** Collapsible section: a summary line plus a body that folds away. */
export interface ToggleBlock extends DocBlockBase {
  type: 'toggle'
  /** Summary line, always visible. */
  title: string
  /** Rich-text body (TipTap HTML), revealed when expanded. */
  html: string
  /** Default expanded state when the document is read. Default false. */
  open?: boolean
}

/** The union of every special block type. Grows as blocks land. */
export type SpecialBlock = ToggleBlock
