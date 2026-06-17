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
} from './wiki-types'
import type { WikiArticle, WikiCategory, WikiVersion } from '@/types/wiki'

export function adaptArticle(api: ApiArticle): WikiArticle {
  return {
    id: api.id,
    title: api.title,
    slug: api.slug,
    // content is TipTap JSONB — extract plain text best-effort for the UI
    content:
      typeof api.content === 'string'
        ? (api.content as string)
        : ((api.content as Record<string, unknown>)?.plain as string | undefined) ?? '',
    categoryId: api.category_id ?? '',
    status: api.published ? 'published' : 'draft',
    authorId: api.author_id,
    // authorName not returned by API — needs a join/lookup
    authorName: '',
    // tags, isPinned, viewCount not in API schema yet
    tags: [],
    isPinned: false,
    viewCount: 0,
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
    changeNote: '',
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
