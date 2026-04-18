/**
 * Lightweight fetch wrapper for Wiki API endpoints.
 *
 * Follows the same pattern as dialer-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once wiki routes are added
 * to openapi.yaml, hooks can migrate to the typed apiClient.
 */
import { API_BASE_URL } from '@/lib/constants'
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
  ListArticlesResponse,
  SearchArticlesResponse,
} from './wiki-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = { method: opts.method, headers }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), { ...init, headers })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
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
  body: { file_ref: string; mime: string; size: number },
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

// ---------------------------------------------------------------------------
// Share tokens
// ---------------------------------------------------------------------------

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
