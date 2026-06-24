/**
 * Document API client -- typed fetch wrapper for all document HTTP endpoints.
 *
 * Follows the same auth/refresh/offline pattern as email-client.ts.
 * Gateway routes: /api/v1/documents/*
 */
import { API_BASE_URL } from '@/lib/constants'
import { normalizeWireTimestamps } from '@/api/wire-time'
import type {
  FolderSpaceType,
  ListFilesParams,
  ListFoldersParams,
  CreateFolderRequest,
  UpdateFolderRequest,
  UpdateFileRequest,
  ShareEntityRequest,
  CreateTagRequest,
  LinkFileRequest,
  SearchFilesParams,
  DocumentFolder,
  DocumentFile,
  DocumentFileVersion,
  DocumentShare,
  DocumentTag,
  DocumentEntityLink,
  VirtualFile,
  FolderPathSegment,
  FileSearchResult,
  ListFilesResponse,
  ListFoldersResponse,
  ListVersionsResponse,
  ListSharesResponse,
  ListTagsResponse,
  ListEntityLinksResponse,
  SearchFilesResponse,
  ListVirtualFilesResponse,
  DownloadURLResponse,
  ListActivityResponse,
  WOPITokenResponse,
  SharePermission,
  DocumentActivity,
} from '../types/document-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich.')
    this.name = 'OfflineError'
  }
}

let refreshPromise: Promise<string | null> | null = null

async function getToken(): Promise<string | undefined> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken ?? undefined
}

async function refreshToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  const store = useAuthStore.getState()
  if (!refreshPromise) {
    refreshPromise = store.refreshToken().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const method = options.method ?? 'GET'

  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new OfflineError()
  }

  const token = await getToken()
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  let res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  // Handle 401 with transparent refresh
  if (res.status === 401 && !path.includes('/auth/')) {
    const newToken = await refreshToken()
    if (!newToken) {
      const { useAuthStore } = await import('@/stores/auth')
      useAuthStore.getState().logout()
      throw new Error('Session abgelaufen')
    }
    headers.set('Authorization', `Bearer ${newToken}`)
    res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? body.message ?? `HTTP ${res.status}`)
  }

  // 204 No Content
  if (res.status === 204) return {} as T

  return res.json() as Promise<T>
}

function qs(params: Record<string, unknown>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null,
  )
  if (entries.length === 0) return ''
  return (
    '?' +
    new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString()
  )
}

// ---------------------------------------------------------------------------
// List-shape normalization
//
// The gateway is inconsistent for document list endpoints: some return a bare
// JSON array (which serializes to `null` when empty — e.g. folders, tags,
// shares, versions, links, activity), others a wrapped `{ items, total }`
// object (e.g. files, which returns `{ files: null, total: 0 }` when empty).
// MSW always returned the FE-typed wrapped shape, so this drift was mock-hidden
// and only surfaces against the real backend. Coalesce both into the typed
// `{ [key]: T[], total }` every consumer expects. Canonical fix = wrap all list
// responses consistently in the gateway (handed to Luke, see backend-gaps.md).
// ---------------------------------------------------------------------------

function unwrapList<T>(raw: unknown, key: string): T[] {
  if (Array.isArray(raw)) return raw as T[]
  if (raw && typeof raw === 'object') {
    const inner = (raw as Record<string, unknown>)[key]
    if (Array.isArray(inner)) return inner as T[]
  }
  return []
}

function listTotal(raw: unknown, fallback: number): number {
  if (raw && typeof raw === 'object') {
    const t = (raw as Record<string, unknown>).total
    if (typeof t === 'number') return t
  }
  return fallback
}

// Single-entity GET/POST responses are equally inconsistent: the gateway
// returns the bare object (`{ id, name, ... }`), while the FE + MSW expect it
// wrapped (`{ folder: {...} }` / `{ file: {...} }`). Accept both.
function unwrapEntity(raw: unknown, key: string): unknown {
  if (raw && typeof raw === 'object' && key in (raw as Record<string, unknown>)) {
    return (raw as Record<string, unknown>)[key]
  }
  return raw
}

// ---------------------------------------------------------------------------
// Wire-shape normalization for folder/file objects
//
// protojson emits Timestamps as `{ seconds, nanos }` and enums as ints (the
// gateway returns `space_type: 1`), while the FE types expect ISO strings and
// string unions. int64 `file_size` may also arrive as a string. MSW returned
// the already-typed shapes, so this was mock-hidden. Normalize on the way in.
// ---------------------------------------------------------------------------

const SPACE_TYPE_BY_INT: Record<number, FolderSpaceType> = {
  1: 'personal',
  2: 'team',
  3: 'project',
}

function toSpaceType(v: unknown): FolderSpaceType {
  if (typeof v === 'number') return SPACE_TYPE_BY_INT[v] ?? 'personal'
  const s = String(v ?? '').toLowerCase()
  if (s.includes('team')) return 'team'
  if (s.includes('project')) return 'project'
  return 'personal'
}

function normalizeFolder(raw: unknown): DocumentFolder {
  const f = normalizeWireTimestamps(raw) as Record<string, unknown>
  return {
    ...(f as unknown as DocumentFolder),
    space_type: toSpaceType(f.space_type),
    file_count: typeof f.file_count === 'number' ? f.file_count : 0,
  }
}

function normalizeFile(raw: unknown): DocumentFile {
  const f = normalizeWireTimestamps(raw) as Record<string, unknown>
  return {
    ...(f as unknown as DocumentFile),
    file_size:
      typeof f.file_size === 'string' ? Number(f.file_size) : (f.file_size as number),
    tags: Array.isArray(f.tags) ? (f.tags as DocumentTag[]) : [],
  }
}

// ---------------------------------------------------------------------------
// Folders
// ---------------------------------------------------------------------------

export const documentFolderApi = {
  async list(params: ListFoldersParams = {}): Promise<ListFoldersResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/folders${qs(params as Record<string, unknown>)}`,
    )
    const folders = unwrapList<DocumentFolder>(raw, 'folders').map(normalizeFolder)
    return { folders, total: listTotal(raw, folders.length) }
  },

  async get(id: string): Promise<{ folder: DocumentFolder }> {
    const raw = await request<unknown>(
      `/api/v1/documents/folders/${id}`,
    )
    return { folder: normalizeFolder(unwrapEntity(raw, 'folder')) }
  },

  async create(data: CreateFolderRequest): Promise<{ folder: DocumentFolder }> {
    const raw = await request<unknown>('/api/v1/documents/folders', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    return { folder: normalizeFolder(unwrapEntity(raw, 'folder')) }
  },

  update(id: string, data: UpdateFolderRequest) {
    return request<Record<string, never>>(`/api/v1/documents/folders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/documents/folders/${id}`, {
      method: 'DELETE',
    })
  },

  async getPath(id: string): Promise<{ segments: FolderPathSegment[] }> {
    const raw = await request<unknown>(
      `/api/v1/documents/folders/${id}/path`,
    )
    return { segments: unwrapList<FolderPathSegment>(raw, 'segments') }
  },

  initializeUserSpace() {
    // Gateway decodes+validates the body even though user_id is optional, so an
    // empty/absent body is rejected with 400 "invalid request body". Send `{}`.
    return request<Record<string, never>>(
      '/api/v1/documents/folders/initialize-user',
      { method: 'POST', body: JSON.stringify({}) },
    )
  },

  initializeTeamSpace(teamId: string, teamName: string) {
    return request<Record<string, never>>(
      '/api/v1/documents/folders/initialize-team',
      {
        method: 'POST',
        body: JSON.stringify({ team_id: teamId, team_name: teamName }),
      },
    )
  },
}

// ---------------------------------------------------------------------------
// Files
// ---------------------------------------------------------------------------

export const documentFileApi = {
  async list(params: ListFilesParams = {}): Promise<ListFilesResponse> {
    const { tag_ids, ...rest } = params
    const queryParts = qs(rest as Record<string, unknown>)
    // Append tag_ids as repeated params
    const tagParams = tag_ids?.length
      ? (queryParts ? '&' : '?') +
        tag_ids.map((id) => `tag_ids=${encodeURIComponent(id)}`).join('&')
      : ''
    const raw = await request<unknown>(
      `/api/v1/documents/files${queryParts}${tagParams}`,
    )
    const files = unwrapList<DocumentFile>(raw, 'files').map(normalizeFile)
    return { files, total: listTotal(raw, files.length) }
  },

  async get(id: string): Promise<{ file: DocumentFile }> {
    const raw = await request<unknown>(
      `/api/v1/documents/files/${id}`,
    )
    return { file: normalizeFile(unwrapEntity(raw, 'file')) }
  },

  update(id: string, data: UpdateFileRequest) {
    return request<Record<string, never>>(`/api/v1/documents/files/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/documents/files/${id}`, {
      method: 'DELETE',
    })
  },

  async copy(id: string, targetFolderId: string): Promise<{ file: DocumentFile }> {
    const raw = await request<unknown>(
      `/api/v1/documents/files/${id}/copy`,
      {
        method: 'POST',
        body: JSON.stringify({ target_folder_id: targetFolderId }),
      },
    )
    return { file: normalizeFile(unwrapEntity(raw, 'file')) }
  },

  move(id: string, targetFolderId: string) {
    return request<Record<string, never>>(
      `/api/v1/documents/files/${id}/move`,
      {
        method: 'POST',
        body: JSON.stringify({ target_folder_id: targetFolderId }),
      },
    )
  },

  getDownloadURL(id: string) {
    return request<DownloadURLResponse>(
      `/api/v1/documents/files/${id}/download-url`,
    )
  },

  async listActivity(id: string): Promise<ListActivityResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/files/${id}/activity`,
    )
    return { activities: unwrapList<DocumentActivity>(raw, 'activities') }
  },
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

export const documentVersionApi = {
  async list(fileId: string): Promise<ListVersionsResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/files/${fileId}/versions`,
    )
    return { versions: unwrapList<DocumentFileVersion>(raw, 'versions') }
  },

  create(fileId: string, data: { version_label?: string }) {
    return request<{ version: DocumentFileVersion }>(
      `/api/v1/documents/files/${fileId}/versions`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    )
  },

  revert(fileId: string, versionNumber: number) {
    return request<{ version: DocumentFileVersion }>(
      `/api/v1/documents/files/${fileId}/versions/revert`,
      {
        method: 'POST',
        body: JSON.stringify({ version_number: versionNumber }),
      },
    )
  },
}

// ---------------------------------------------------------------------------
// Shares
// ---------------------------------------------------------------------------

export const documentShareApi = {
  share(data: ShareEntityRequest) {
    return request<{ share: DocumentShare }>('/api/v1/documents/shares', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  unshare(shareId: string) {
    return request<Record<string, never>>(
      `/api/v1/documents/shares/${shareId}`,
      { method: 'DELETE' },
    )
  },

  async list(entityType: string, entityId: string): Promise<ListSharesResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/shares${qs({ entity_type: entityType, entity_id: entityId })}`,
    )
    return { shares: unwrapList<DocumentShare>(raw, 'shares') }
  },

  async listSharedWithMe(entityType?: string): Promise<ListSharesResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/shares/shared-with-me${qs({ entity_type: entityType })}`,
    )
    return { shares: unwrapList<DocumentShare>(raw, 'shares') }
  },
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

export const documentTagApi = {
  async list(): Promise<ListTagsResponse> {
    const raw = await request<unknown>('/api/v1/documents/tags')
    return { tags: unwrapList<DocumentTag>(raw, 'tags') }
  },

  create(data: CreateTagRequest) {
    return request<{ tag: DocumentTag }>('/api/v1/documents/tags', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/documents/tags/${id}`, {
      method: 'DELETE',
    })
  },

  tagFile(fileId: string, tagId: string) {
    return request<Record<string, never>>('/api/v1/documents/tags/file', {
      method: 'POST',
      body: JSON.stringify({ file_id: fileId, tag_id: tagId }),
    })
  },

  untagFile(fileId: string, tagId: string) {
    return request<Record<string, never>>('/api/v1/documents/tags/file', {
      method: 'DELETE',
      body: JSON.stringify({ file_id: fileId, tag_id: tagId }),
    })
  },
}

// ---------------------------------------------------------------------------
// Entity Links
// ---------------------------------------------------------------------------

export const documentLinkApi = {
  link(fileId: string, data: LinkFileRequest) {
    return request<{ link: DocumentEntityLink }>(
      `/api/v1/documents/files/${fileId}/links`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    )
  },

  unlink(linkId: string) {
    return request<Record<string, never>>(
      `/api/v1/documents/links/${linkId}`,
      { method: 'DELETE' },
    )
  },

  async listByFile(fileId: string): Promise<ListEntityLinksResponse> {
    const raw = await request<unknown>(
      `/api/v1/documents/files/${fileId}/links`,
    )
    return { links: unwrapList<DocumentEntityLink>(raw, 'links') }
  },
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

export const documentSearchApi = {
  searchFiles(
    query: string,
    params: SearchFilesParams = {},
  ) {
    const { tag_ids, ...rest } = params
    const allParams = { ...rest, q: query }
    const queryParts = qs(allParams as Record<string, unknown>)
    const tagParams = tag_ids?.length
      ? (queryParts ? '&' : '?') +
        tag_ids.map((id) => `tag_ids=${encodeURIComponent(id)}`).join('&')
      : ''
    return request<SearchFilesResponse>(
      `/api/v1/documents/search${queryParts}${tagParams}`,
    )
  },

  listVirtualFiles(sourceType?: string, limit?: number, offset?: number) {
    return request<ListVirtualFilesResponse>(
      `/api/v1/documents/virtual${qs({ source_type: sourceType, limit, offset })}`,
    )
  },
}

// ---------------------------------------------------------------------------
// WOPI
// ---------------------------------------------------------------------------

export const documentWopiApi = {
  generateToken(fileId: string) {
    return request<WOPITokenResponse>('/api/v1/documents/wopi/token', {
      method: 'POST',
      body: JSON.stringify({ file_id: fileId }),
    })
  },
}

// Re-export types for convenience
export type {
  DocumentFolder,
  DocumentFile,
  DocumentFileVersion,
  DocumentShare,
  DocumentTag,
  DocumentEntityLink,
  VirtualFile,
  FolderPathSegment,
  FileSearchResult,
  SharePermission,
}
