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

export function adaptArticle(api: ApiArticle): WikiArticle {
  return {
    id: api.id,
    title: api.title,
    slug: api.slug,
    // content is TipTap JSONB — normalise to HTML for the editor + reader
    content: extractHtml(api.content),
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
  return {
    id: api.id,
    articleId: api.article_id,
    version: api.version_number,
    editorName: api.changed_by ?? '',
    editedAt: api.changed_at,
    changeNote: api.change_note ?? '',
    content: extractHtml(api.content),
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
