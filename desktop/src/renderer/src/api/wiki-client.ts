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
import { normalizeWireTimestamps } from './wire-time'
import { decodeWikiContent, encodeWikiContent } from './wiki-content'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  // The wiki gRPC service serialises via response.JSON over pb.go structs, so
  // google.protobuf.Timestamp fields arrive as {seconds, nanos} rather than ISO
  // strings. Normalise them so the module's formatDate(...) calls (article/version/
  // attachment/share-token timestamps) render correctly against the real backend.
  // Mock-mode responses are already ISO strings and pass through untouched.
  const data = await authenticatedRequest<T>(opts)
  return normalizeWireTimestamps(data)
}

// ---------------------------------------------------------------------------
// Content codec — article/version `content` rides the wire as base64 (proto
// bytes); decode on read, encode on write so the UI keeps a TipTap object.
// ---------------------------------------------------------------------------

function decodeArticle(a: WikiArticle): WikiArticle {
  return { ...a, content: decodeWikiContent(a.content) }
}

function decodeVersion(v: WikiVersion): WikiVersion {
  return { ...v, content: decodeWikiContent(v.content) }
}

function encodeArticleBody<T extends { content?: unknown }>(body: T): T {
  if (body.content === undefined) return body
  // The wire field is a base64 string; the FE type is TipTapContent. The body is
  // sent as an opaque payload, so the runtime string is correct on the wire.
  return { ...body, content: encodeWikiContent(body.content) as unknown as T['content'] }
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
  }).then((r) => ({ ...r, articles: r.articles.map(decodeArticle) }))
}

export function getArticle(id: string) {
  return request<WikiArticle>({ method: 'GET', path: `${BASE}/articles/${id}` }).then(decodeArticle)
}

export function createArticle(body: CreateArticleInput) {
  return request<WikiArticle>({
    method: 'POST',
    path: `${BASE}/articles`,
    body: encodeArticleBody(body),
  }).then(decodeArticle)
}

export function updateArticle(id: string, body: UpdateArticleInput) {
  return request<WikiArticle>({
    method: 'PATCH',
    path: `${BASE}/articles/${id}`,
    body: encodeArticleBody(body),
  }).then(decodeArticle)
}

export function deleteArticle(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/articles/${id}` })
}

export function searchArticles(query: string, limit?: number) {
  return request<SearchArticlesResponse>({
    method: 'GET',
    path: `${BASE}/search`,
    params: { q: query, ...(limit !== undefined ? { limit } : {}) },
  }).then((r) => ({ ...r, articles: r.articles.map(decodeArticle) }))
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

export function listVersions(articleId: string) {
  return request<WikiVersion[]>({
    method: 'GET',
    path: `${BASE}/articles/${articleId}/versions`,
  }).then((vs) => vs.map(decodeVersion))
}

export function getVersion(versionId: string) {
  return request<WikiVersion>({
    method: 'GET',
    path: `${BASE}/versions/${versionId}`,
  }).then(decodeVersion)
}

export function restoreVersion(articleId: string, versionId: string) {
  return request<WikiArticle>({
    method: 'POST',
    path: `${BASE}/articles/${articleId}/versions/${versionId}/restore`,
  }).then(decodeArticle)
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
