/**
 * TypeScript types for the Wiki module.
 *
 * Mirrors backend/internal/wiki/models.go and service.go input types.
 * UUIDs are represented as strings; dates as ISO 8601 strings.
 * TipTap document content is typed as Record<string, unknown>.
 */

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

/** TipTap document node (JSONB content stored on articles/versions). */
export type TipTapContent = Record<string, unknown>

export interface WikiArticle {
  id: string
  tenant_id: string
  title: string
  slug: string
  /** Raw TipTap JSON document. */
  content: TipTapContent
  author_id: string
  category_id: string | null
  published: boolean
  /** Free-form labels for filtering/grouping. */
  tags?: string[]
  /** Identity (WP-3): icon = a lucide name or "" ; cover = image URL or "grad:id". */
  icon?: string
  cover_url?: string
  /** Read counter — incremented by the backend on article detail GET. */
  view_count?: number
  created_at: string
  updated_at: string
}

export interface WikiVersion {
  id: string
  article_id: string
  version_number: number
  content: TipTapContent
  changed_by: string | null
  changed_at: string
  /** Optional author-supplied summary of what changed in this revision. */
  change_note?: string
}

export interface WikiAttachment {
  id: string
  article_id: string
  file_ref: string
  mime: string
  /** int64 on the wire — protojson serializes it as a string. */
  size: number | string
  uploaded_by: string | null
  created_at: string
  /** Demo blob (data URL) — enables download + image preview without a backend. */
  data_url?: string
}

export interface WikiCategory {
  id: string
  tenant_id: string
  name: string
  parent_id: string | null
  position: number
  created_at: string
  updated_at: string
}

export interface WikiShareToken {
  id: string
  article_id: string
  token: string
  expires_at: string | null
  permissions: string[]
  created_at: string
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateArticleInput {
  title: string
  slug?: string
  content?: TipTapContent
  category_id?: string
  published?: boolean
  tags?: string[]
  icon?: string
  cover_url?: string
}

export interface UpdateArticleInput {
  title?: string
  slug?: string
  content?: TipTapContent
  category_id?: string | null
  published?: boolean
  tags?: string[]
  icon?: string | null
  cover_url?: string | null
  /** Optional note stored on the version snapshot a content edit produces. */
  change_note?: string
}

export interface ListArticlesParams {
  category_id?: string
  author_id?: string
  published?: boolean
  search?: string
  page?: number
  page_size?: number
  sort_by?: string
  sort_desc?: boolean
}

export interface CreateCategoryInput {
  name: string
  parent_id?: string
  position?: number
}

export interface UploadAttachmentInput {
  file_ref: string
  mime: string
  size: number
  data_url?: string
}

export interface CreateShareTokenInput {
  expires_at?: string
  permissions?: string[]
}

// ---------------------------------------------------------------------------
// Templates (WP-5) — reusable content scaffolds, served by MSW (FE-mock).
// The shape matches the UI type WikiTemplate (content is an HTML string).
// ---------------------------------------------------------------------------

export interface CreateTemplateInput {
  name: string
  description?: string
  content?: string
  icon?: string
  category?: string
}

export interface UpdateTemplateInput {
  name?: string
  description?: string
  content?: string
  icon?: string
  category?: string
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListArticlesResponse {
  articles: WikiArticle[]
  total: number
}

export interface SearchArticlesResponse {
  articles: WikiArticle[]
}
