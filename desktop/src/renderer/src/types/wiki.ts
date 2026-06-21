/**
 * Type definitions for the Wiki (Knowledge Base) module.
 *
 * Covers articles, categories, version history, and reusable templates.
 */
import type { DocRow } from '@/components/shared/document'

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type WikiArticleStatus = 'draft' | 'published' | 'archived'

// ---------------------------------------------------------------------------
// Category
// ---------------------------------------------------------------------------

export interface WikiCategory {
  id: string
  name: string
  icon: string
  parentId?: string
  sortOrder: number
  articleCount: number
}

// ---------------------------------------------------------------------------
// Article
// ---------------------------------------------------------------------------

export interface WikiArticle {
  id: string
  title: string
  slug: string
  /**
   * Block document (Phase B) — the source of truth the editor and reader use.
   * A row → column → block tree, rendered by the shared document engine.
   */
  body: DocRow[]
  /**
   * Derived plain-HTML projection of {@link body}, kept only for search snippets
   * and reading-time until those move onto the block model (PB-5).
   */
  content: string
  categoryId: string
  status: WikiArticleStatus
  authorId: string
  authorName: string
  tags: string[]
  /** Identity (WP-3) — lucide icon name (or '') and cover (image URL or 'grad:id'). */
  icon?: string
  coverUrl?: string
  isPinned: boolean
  viewCount: number
  lastEditedBy: string
  lastEditedAt: string
  createdAt: string
  versions: WikiVersion[]
}

// ---------------------------------------------------------------------------
// Version history
// ---------------------------------------------------------------------------

export interface WikiVersion {
  id: string
  articleId: string
  version: number
  editorName: string
  editedAt: string
  changeNote: string
  /** Normalised HTML content of this revision — used for the diff/preview. */
  content: string
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

export interface WikiTemplate {
  id: string
  name: string
  description: string
  content: string
  icon: string
  category: string
}
