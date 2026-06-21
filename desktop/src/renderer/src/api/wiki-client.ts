/**
 * Lightweight fetch wrapper for Wiki API endpoints.
 *
 * Follows the same pattern as dialer-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once wiki routes are added
 * to openapi.yaml, hooks can migrate to the typed apiClient.
 */
import type {
  WikiArticle,
  WikiVersion,
  WikiAttachment,
  WikiCategory,
  WikiShareToken,
  CreateArticleInput,
  UpdateArticleInput,
  ListArticlesParams,
  CreateCategoryInput,
  CreateShareTokenInput,
  CreateTemplateInput,
  UpdateTemplateInput,
  ListArticlesResponse,
  SearchArticlesResponse,
} from './wiki-types'
import type { WikiTemplate } from '@/types/wiki'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/wiki'

// ---------------------------------------------------------------------------
// Articles
// ---------------------------------------------------------------------------

export function listArticles(params?: ListArticlesParams) {
  return request<ListArticlesResponse>({
    method: 'GET',
    path: `${BASE}/articles`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getArticle(id: string) {
  return request<WikiArticle>({ method: 'GET', path: `${BASE}/articles/${id}` })
}

export function createArticle(body: CreateArticleInput) {
  return request<WikiArticle>({ method: 'POST', path: `${BASE}/articles`, body })
}

export function updateArticle(id: string, body: UpdateArticleInput) {
  return request<WikiArticle>({ method: 'PUT', path: `${BASE}/articles/${id}`, body })
}

export function deleteArticle(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/articles/${id}` })
}

export function searchArticles(query: string, limit?: number) {
  return request<SearchArticlesResponse>({
    method: 'GET',
    path: `${BASE}/search`,
    params: { q: query, ...(limit !== undefined ? { limit } : {}) },
  })
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

export function listVersions(articleId: string) {
  return request<WikiVersion[]>({
    method: 'GET',
    path: `${BASE}/articles/${articleId}/versions`,
  })
}

export function getVersion(versionId: string) {
  return request<WikiVersion>({ method: 'GET', path: `${BASE}/versions/${versionId}` })
}

export function restoreVersion(articleId: string, versionId: string) {
  return request<WikiArticle>({
    method: 'POST',
    path: `${BASE}/articles/${articleId}/versions/${versionId}/restore`,
  })
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

export function listAttachments(articleId: string) {
  return request<WikiAttachment[]>({
    method: 'GET',
    path: `${BASE}/articles/${articleId}/attachments`,
  })
}

export function uploadAttachment(
  articleId: string,
  body: { file_ref: string; mime: string; size: number; data_url?: string },
) {
  return request<WikiAttachment>({
    method: 'POST',
    path: `${BASE}/articles/${articleId}/attachments`,
    body,
  })
}

export function deleteAttachment(attachmentId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/attachments/${attachmentId}` })
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

export function listCategories() {
  return request<WikiCategory[]>({ method: 'GET', path: `${BASE}/categories` })
}

export function createCategory(body: CreateCategoryInput) {
  return request<WikiCategory>({ method: 'POST', path: `${BASE}/categories`, body })
}

export function deleteCategory(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/categories/${id}` })
}

export function updateCategory(
  id: string,
  body: { name?: string; parent_id?: string | null; position?: number },
) {
  return request<WikiCategory>({ method: 'PATCH', path: `${BASE}/categories/${id}`, body })
}

// ---------------------------------------------------------------------------
// Share tokens
// ---------------------------------------------------------------------------

export function listShareTokens(articleId: string) {
  return request<WikiShareToken[]>({
    method: 'GET',
    path: `${BASE}/articles/${articleId}/share`,
  })
}

export function createShareToken(articleId: string, body?: CreateShareTokenInput) {
  return request<WikiShareToken>({
    method: 'POST',
    path: `${BASE}/articles/${articleId}/share`,
    body,
  })
}

export function revokeShareToken(tokenId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/share/${tokenId}` })
}

// ---------------------------------------------------------------------------
// Templates (WP-5)
// ---------------------------------------------------------------------------

export function listTemplates() {
  return request<WikiTemplate[]>({ method: 'GET', path: `${BASE}/templates` })
}

export function createTemplate(body: CreateTemplateInput) {
  return request<WikiTemplate>({ method: 'POST', path: `${BASE}/templates`, body })
}

export function updateTemplate(id: string, body: UpdateTemplateInput) {
  return request<WikiTemplate>({ method: 'PUT', path: `${BASE}/templates/${id}`, body })
}

export function deleteTemplate(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/templates/${id}` })
}
