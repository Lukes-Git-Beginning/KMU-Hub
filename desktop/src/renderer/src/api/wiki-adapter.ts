/**
 * Adapter functions — map API types (wiki-types.ts) to UI types (types/wiki.ts).
 *
 * Fields that have no backend equivalent (isPinned, viewCount, authorName,
 * tags, lastEditedBy, versions) are defaulted to sensible zero-values.
 * They can be enriched later when the backend exposes them.
 */
import type {
  WikiArticle as ApiArticle,
  WikiCategory as ApiCategory,
  WikiVersion as ApiVersion,
  TipTapContent,
} from './wiki-types'
import type { WikiArticle, WikiCategory, WikiVersion } from '@/types/wiki'
import type { DocBlockBase, DocRow } from '@/components/shared/document'

// ---------------------------------------------------------------------------
// Content normalisation
// ---------------------------------------------------------------------------

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

/** Minimal TipTap-JSON → HTML walker (fallback for legacy doc-shaped content). */
function tiptapNodeToHtml(node: Record<string, unknown>): string {
  if (!node || typeof node !== 'object') return ''
  const type = node.type as string | undefined

  if (type === 'text') {
    let text = escapeHtml((node.text as string) ?? '')
    const marks = (node.marks as Array<{ type: string; attrs?: Record<string, unknown> }>) ?? []
    for (const mark of marks) {
      if (mark.type === 'bold') text = `<strong>${text}</strong>`
      else if (mark.type === 'italic') text = `<em>${text}</em>`
      else if (mark.type === 'underline') text = `<u>${text}</u>`
      else if (mark.type === 'code') text = `<code>${text}</code>`
      else if (mark.type === 'link') text = `<a href="${escapeHtml(String(mark.attrs?.href ?? '#'))}">${text}</a>`
    }
    return text
  }

  const children = (node.content as Array<Record<string, unknown>>) ?? []
  const inner = children.map(tiptapNodeToHtml).join('')

  switch (type) {
    case 'doc':
      return inner
    case 'paragraph':
      return `<p>${inner}</p>`
    case 'heading': {
      const level = Number((node.attrs as Record<string, unknown>)?.level ?? 2)
      return `<h${level}>${inner}</h${level}>`
    }
    case 'bulletList':
      return `<ul>${inner}</ul>`
    case 'orderedList':
      return `<ol>${inner}</ol>`
    case 'listItem':
      return `<li>${inner}</li>`
    case 'blockquote':
      return `<blockquote>${inner}</blockquote>`
    case 'codeBlock':
      return `<pre><code>${inner}</code></pre>`
    case 'hardBreak':
      return '<br>'
    case 'horizontalRule':
      return '<hr>'
    default:
      return inner
  }
}

/**
 * Normalises the JSONB `content` field into the HTML string the editor and
 * reader consume. Handles every shape the store may hold:
 *   - `{ html }`     — produced by the TipTap editor on save
 *   - `{ plain }`    — legacy template scaffolds
 *   - `string`       — raw HTML
 *   - `{ type: doc }`— real TipTap document (rendered via the walker)
 */
export function extractHtml(content: TipTapContent | string | null | undefined): string {
  if (!content) return ''
  if (typeof content === 'string') return content
  const c = content as Record<string, unknown>
  if (typeof c.html === 'string') return c.html
  if (typeof c.plain === 'string') return c.plain
  if (c.type === 'doc') return tiptapNodeToHtml(c)
  return ''
}

// ---------------------------------------------------------------------------
// Block-document bridge (Phase B)
// ---------------------------------------------------------------------------

/** Wrap legacy/normalised HTML into a single long-form text block. */
export function htmlToRows(html: string): DocRow[] {
  if (!html.trim()) return []
  const textBlock = { id: 'b-legacy-text', type: 'text', html } as unknown as DocBlockBase
  return [{ id: 'r-legacy', columns: [{ id: 'c-legacy', width: 1, blocks: [textBlock] }] }]
}

/**
 * Resolve the article body to a block-document. Articles authored in Phase B
 * carry `{ rows }`; anything older (`{ html }`, `{ plain }`, a TipTap doc, or a
 * raw string) is bridged into a single long-form text block so nothing is lost.
 */
export function extractRows(content: TipTapContent | string | null | undefined): DocRow[] {
  if (content && typeof content === 'object') {
    const rows = (content as Record<string, unknown>).rows
    if (Array.isArray(rows)) return rows as DocRow[]
  }
  return htmlToRows(extractHtml(content))
}

/** Minimal block → HTML projection, used only for search + reading-time. */
function blockToHtml(block: DocBlockBase): string {
  const b = block as unknown as Record<string, unknown>
  switch (block.type) {
    case 'heading':
      return `<h${b.level === 1 ? 2 : 3}>${escapeHtml(String(b.text ?? ''))}</h${b.level === 1 ? 2 : 3}>`
    case 'text':
    case 'callout':
      return String(b.html ?? '')
    case 'bullet': {
      const items = (b.items as string[] | undefined)?.filter((i) => i.trim()) ?? []
      const tag = b.ordered ? 'ol' : 'ul'
      return `<${tag}>${items.map((i) => `<li>${escapeHtml(i)}</li>`).join('')}</${tag}>`
    }
    case 'image':
      return b.caption ? `<p>${escapeHtml(String(b.caption))}</p>` : ''
    default:
      return ''
  }
}

/** Flatten a block-document into HTML for the derived `content` field. */
export function rowsToHtml(rows: DocRow[]): string {
  return rows
    .flatMap((row) => row.columns.flatMap((col) => col.blocks.map(blockToHtml)))
    .filter(Boolean)
    .join('\n')
}

export function adaptArticle(api: ApiArticle): WikiArticle {
  const html = extractHtml(api.content)
  const body = extractRows(api.content)
  return {
    id: api.id,
    title: api.title,
    slug: api.slug,
    // Block document is the source of truth; content is a derived projection
    // kept for search snippets + reading-time until those move onto blocks.
    body,
    content: html || rowsToHtml(body),
    categoryId: api.category_id ?? '',
    status: api.published ? 'published' : 'draft',
    authorId: api.author_id,
    // authorName not returned by API — needs a join/lookup
    authorName: '',
    // tags are a real article field now; isPinned stays mock-first via the store
    tags: api.tags ?? [],
    // identity (WP-3) — served by MSW; defaults to no icon/cover
    icon: api.icon ?? undefined,
    coverUrl: api.cover_url ?? undefined,
    isPinned: false,
    // view_count is backend-tracked; tags/isPinned are enriched in the page
    viewCount: api.view_count ?? 0,
    lastEditedBy: '',
    lastEditedAt: api.updated_at,
    createdAt: api.created_at,
    // versions loaded separately via useArticleVersions
    versions: [],
  }
}

export function adaptVersion(api: ApiVersion): WikiVersion {
  // Project block-document versions ({rows}) to HTML so the (HTML-based) version
  // diff works for both legacy and Phase-B snapshots.
  const html = extractHtml(api.content) || rowsToHtml(extractRows(api.content))
  return {
    id: api.id,
    articleId: api.article_id,
    version: api.version_number,
    editorName: api.changed_by ?? '',
    editedAt: api.changed_at,
    changeNote: api.change_note ?? '',
    content: html,
  }
}

export function adaptCategory(api: ApiCategory): WikiCategory {
  return {
    id: api.id,
    name: api.name,
    // icon not stored in API — default decoration
    icon: 'BookOpen',
    parentId: api.parent_id ?? undefined,
    sortOrder: api.position,
    // articleCount computed client-side from articles
    articleCount: 0,
  }
}
