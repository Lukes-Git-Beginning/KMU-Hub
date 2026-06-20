/**
 * Helpers for report documents (multi-page authoring, Schicht 4).
 * Block metadata (icon + i18n label key), outline summaries and page estimates
 * are reused by the library, the editor shell and (R-1+) the block editor.
 */
import {
  AlignLeft,
  BarChart3,
  BookOpen,
  Heading,
  Image as ImageIcon,
  Lightbulb,
  List,
  Minus,
  Scissors,
  Table as TableIcon,
  TrendingUp,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { ReportBlock, ReportBlockType, ReportDocument } from '@/api/berichte-types'

export const BLOCK_META: Record<ReportBlockType, { icon: LucideIcon; labelKey: string }> = {
  cover: { icon: BookOpen, labelKey: 'berichte.docs.blockType.cover' },
  heading: { icon: Heading, labelKey: 'berichte.docs.blockType.heading' },
  text: { icon: AlignLeft, labelKey: 'berichte.docs.blockType.text' },
  chart: { icon: BarChart3, labelKey: 'berichte.docs.blockType.chart' },
  table: { icon: TableIcon, labelKey: 'berichte.docs.blockType.table' },
  kpi: { icon: TrendingUp, labelKey: 'berichte.docs.blockType.kpi' },
  callout: { icon: Lightbulb, labelKey: 'berichte.docs.blockType.callout' },
  bullet: { icon: List, labelKey: 'berichte.docs.blockType.bullet' },
  divider: { icon: Minus, labelKey: 'berichte.docs.blockType.divider' },
  image: { icon: ImageIcon, labelKey: 'berichte.docs.blockType.image' },
  pagebreak: { icon: Scissors, labelKey: 'berichte.docs.blockType.pagebreak' },
}

/** Strip HTML tags to plain text for outline summaries. */
export function stripHtml(html: string): string {
  return html
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

/** Short plain-text summary of a block for the outline view. */
export function blockSummary(block: ReportBlock): string {
  switch (block.type) {
    case 'cover':
      return block.title
    case 'heading':
      return block.text
    case 'text':
      return stripHtml(block.html)
    case 'chart':
    case 'table':
      return block.caption ?? ''
    case 'kpi':
      return `${block.label}: ${block.value}${block.unit ?? ''}`
    case 'callout':
      return block.title ?? stripHtml(block.html)
    case 'bullet':
      return block.items.join(' · ')
    case 'image':
      return block.caption ?? block.alt ?? ''
    case 'divider':
    case 'pagebreak':
    default:
      return ''
  }
}

/** Rough page count: explicit page breaks + 1. Exact pagination follows in R-3. */
export function estimatePageCount(doc: ReportDocument): number {
  let breaks = 0
  for (const row of doc.rows) {
    for (const col of row.columns) {
      for (const block of col.blocks) {
        if (block.type === 'pagebreak') breaks += 1
      }
    }
  }
  return breaks + 1
}

/** Total number of content blocks in the document. */
export function blockCount(doc: ReportDocument): number {
  let n = 0
  for (const row of doc.rows) {
    for (const col of row.columns) {
      n += col.blocks.length
    }
  }
  return n
}
