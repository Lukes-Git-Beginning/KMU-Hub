/**
 * KB block registry — wires the shared document engine into the Helpdesk
 * knowledge base (G3). A KB article is now authored/read as a block document,
 * the same substrate as wiki + berichte, so "Einträge erstellen" feels
 * consistent everywhere and gains structure (callouts, code, tables, toggles).
 *
 * Unlike the wiki registry we do NOT reuse the wiki text/heading overrides — those
 * couple the text block to the wiki store ([[link]]/@mention resolution), which
 * doesn't belong in the helpdesk. KB stays self-contained on the plain shared
 * core + a knowledge-appropriate subset of the special blocks.
 */
import {
  buildRegistry,
  createCoreBlockDefs,
  createSpecialBlockDefs,
  docUid,
  type BlockRegistry,
  type BlockTypeDef,
  type DocRow,
  type TextBlock,
} from '@/components/shared/document'

const core = Object.fromEntries(createCoreBlockDefs().map((d) => [d.type, d])) as Record<
  string,
  BlockTypeDef
>

// Structural elements a knowledge article wants — collapsible sections, code,
// tables, quotes — minus wiki-only bookmarks/attachments.
const special = createSpecialBlockDefs({ only: ['toggle', 'code', 'simpletable', 'quote'] })

/** Insert-menu order: prose first, then structure and the special elements. */
export const kbBlockRegistry: BlockRegistry = buildRegistry([
  core.text,
  core.heading,
  core.bullet,
  core.callout,
  core.image,
  core.divider,
  ...special,
])

// ── Content ↔ rows adapter ────────────────────────────────────────────────────

const escapeHtml = (s: string): string =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/**
 * Load a KB article's stored `content` into block rows. New articles store a
 * JSON block document; legacy seeds store HTML or plain text — those are wrapped
 * in a single text block so nothing is lost and they open cleanly in the editor.
 */
export function kbContentToRows(content: string): DocRow[] {
  const raw = (content ?? '').trim()
  if (!raw) return []
  // Stored block document? (JSON array of rows)
  if (raw.startsWith('[')) {
    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) return parsed as DocRow[]
    } catch {
      // fall through — treat as HTML/plain text
    }
  }
  const looksHtml = /<[a-z][\s\S]*>/i.test(raw)
  const html = looksHtml
    ? raw
    : raw
        .split('\n\n')
        .map((p) => `<p>${escapeHtml(p).replace(/\n/g, '<br>')}</p>`)
        .join('')
  const textBlock: TextBlock = { id: docUid('text'), type: 'text', html }
  return [
    {
      id: docUid('row'),
      columns: [{ id: docUid('col'), width: 1, blocks: [textBlock] }],
    },
  ]
}

/** Serialise block rows back to the article's `content` string (JSON). */
export function kbRowsToContent(rows: DocRow[]): string {
  return JSON.stringify(rows)
}

const stripTags = (html: string): string => html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()

/**
 * Plain-text excerpt of an article's content for list cards — works for both the
 * new block-document JSON (pulls text out of the blocks) and legacy HTML/plain
 * seeds, so a card never shows raw JSON.
 */
export function kbContentPreview(content: string): string {
  const raw = (content ?? '').trim()
  if (!raw) return ''
  if (raw.startsWith('[')) {
    try {
      const rows = JSON.parse(raw) as DocRow[]
      const parts: string[] = []
      for (const row of rows) {
        for (const col of row.columns ?? []) {
          for (const block of col.blocks ?? []) {
            const b = block as unknown as Record<string, unknown>
            if (typeof b.html === 'string') parts.push(stripTags(b.html))
            else if (typeof b.text === 'string') parts.push(b.text)
            else if (Array.isArray(b.items)) parts.push((b.items as string[]).join(' '))
          }
        }
      }
      return parts.join(' ').trim()
    } catch {
      // fall through
    }
  }
  return stripTags(raw)
}
