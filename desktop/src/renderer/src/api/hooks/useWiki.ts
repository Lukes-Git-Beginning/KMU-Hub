/**
 * TanStack Query hooks for the Wiki module.
 *
 * Queries for articles, versions, attachments, categories, and search.
 * Mutations for article lifecycle, version restore, attachment upload,
 * category management, and share token creation.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listArticles,
  getArticle,
  createArticle,
  updateArticle,
  deleteArticle,
  searchArticles,
  listVersions,
  getVersion,
  restoreVersion,
  listAttachments,
  uploadAttachment,
  deleteAttachment,
  listCategories,
  createCategory,
  deleteCategory,
  createShareToken,
  revokeShareToken,
} from '../wiki-client'
import type {
  CreateArticleInput,
  UpdateArticleInput,
  ListArticlesParams,
  CreateCategoryInput,
  CreateShareTokenInput,
} from '../wiki-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const wikiKeys = {
  all: ['wiki'] as const,
  articles: (params?: ListArticlesParams) => ['wiki', 'articles', params] as const,
  article: (id: string) => ['wiki', 'articles', id] as const,
  versions: (articleId: string) => ['wiki', 'articles', articleId, 'versions'] as const,
  version: (versionId: string) => ['wiki', 'versions', versionId] as const,
  attachments: (articleId: string) => ['wiki', 'articles', articleId, 'attachments'] as const,
  categories: () => ['wiki', 'categories'] as const,
  search: (query: string) => ['wiki', 'search', query] as const,
}

// ---------------------------------------------------------------------------
// Queries — Articles
// ---------------------------------------------------------------------------

export function useArticles(params?: ListArticlesParams) {
  return useQuery({
    queryKey: wikiKeys.articles(params),
    queryFn: () => listArticles(params),
  })
}

export function useArticle(id: string) {
  return useQuery({
    queryKey: wikiKeys.article(id),
    queryFn: () => getArticle(id),
    enabled: !!id,
  })
}

export function useSearchArticles(query: string) {
  return useQuery({
    queryKey: wikiKeys.search(query),
    queryFn: () => searchArticles(query),
    enabled: query.trim().length > 1,
  })
}

// ---------------------------------------------------------------------------
// Queries — Versions
// ---------------------------------------------------------------------------

export function useArticleVersions(articleId: string) {
  return useQuery({
    queryKey: wikiKeys.versions(articleId),
    queryFn: () => listVersions(articleId),
    enabled: !!articleId,
  })
}

export function useArticleVersion(versionId: string) {
  return useQuery({
    queryKey: wikiKeys.version(versionId),
    queryFn: () => getVersion(versionId),
    enabled: !!versionId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Attachments
// ---------------------------------------------------------------------------

export function useAttachments(articleId: string) {
  return useQuery({
    queryKey: wikiKeys.attachments(articleId),
    queryFn: () => listAttachments(articleId),
    enabled: !!articleId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Categories
// ---------------------------------------------------------------------------

export function useCategories() {
  return useQuery({
    queryKey: wikiKeys.categories(),
    queryFn: listCategories,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Articles
// ---------------------------------------------------------------------------

export function useCreateArticle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateArticleInput) => createArticle(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['wiki', 'articles'] }),
  })
}

export function useUpdateArticle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateArticleInput & { id: string }) =>
      updateArticle(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: wikiKeys.article(variables.id) })
      qc.invalidateQueries({ queryKey: ['wiki', 'articles'] })
    },
  })
}

export function useDeleteArticle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteArticle(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['wiki', 'articles'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Versions
// ---------------------------------------------------------------------------

export function useRestoreVersion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ articleId, versionId }: { articleId: string; versionId: string }) =>
      restoreVersion(articleId, versionId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: wikiKeys.article(variables.articleId) })
      qc.invalidateQueries({ queryKey: wikiKeys.versions(variables.articleId) })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Attachments
// ---------------------------------------------------------------------------

export function useUploadAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      articleId,
      ...body
    }: {
      articleId: string
      file_ref: string
      mime: string
      size: number
    }) => uploadAttachment(articleId, body),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: wikiKeys.attachments(variables.articleId) }),
  })
}

export function useDeleteAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      attachmentId,
      articleId,
    }: {
      attachmentId: string
      articleId: string
    }) => deleteAttachment(attachmentId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: wikiKeys.attachments(variables.articleId) }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Categories
// ---------------------------------------------------------------------------

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCategoryInput) => createCategory(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: wikiKeys.categories() }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteCategory(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: wikiKeys.categories() }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Share tokens
// ---------------------------------------------------------------------------

export function useCreateShareToken() {
  return useMutation({
    mutationFn: ({
      articleId,
      ...input
    }: CreateShareTokenInput & { articleId: string }) =>
      createShareToken(articleId, input),
  })
}

export function useRevokeShareToken() {
  return useMutation({
    mutationFn: (tokenId: string) => revokeShareToken(tokenId),
  })
}
