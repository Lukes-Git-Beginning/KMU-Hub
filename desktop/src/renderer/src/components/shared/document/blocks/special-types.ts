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

/** Syntax-highlighted code listing with a language and a copy affordance. */
export interface CodeBlock extends DocBlockBase {
  type: 'code'
  /** Raw source. */
  code: string
  /** Highlight grammar id (lowlight `common`), e.g. 'typescript'. 'plaintext' = no tint. */
  language: string
}

/**
 * A hand-authored table (not query-backed — that is berichte's data `table`).
 * `cells` is row-major and kept rectangular; the first row is the header when
 * `hasHeader` is set.
 */
export interface SimpleTableBlock extends DocBlockBase {
  type: 'simpletable'
  cells: string[][]
  /** Render the first row as a header. Default true. */
  hasHeader?: boolean
}

/** A downloadable file: name, size, mime and the data (mock: a data URL). */
export interface AttachmentBlock extends DocBlockBase {
  type: 'attachment'
  name: string
  /** Size in bytes. */
  size?: number
  mime?: string
  /** Data URL (mock store) or a remote URL. Empty until a file is attached. */
  url?: string
}

/** The union of every special block type. Grows as blocks land. */
export type SpecialBlock = ToggleBlock | CodeBlock | SimpleTableBlock | AttachmentBlock
