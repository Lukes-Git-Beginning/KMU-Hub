/**
 * Type definitions for the Wiki (Knowledge Base) module.
 *
 * Covers articles, categories, version history, and reusable templates.
 */

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
  content: string
  categoryId: string
  status: WikiArticleStatus
  authorId: string
  authorName: string
  tags: string[]
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
