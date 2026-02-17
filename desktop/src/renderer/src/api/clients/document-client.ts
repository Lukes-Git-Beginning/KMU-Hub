/**
 * Document API client -- typed fetch wrapper for all document HTTP endpoints.
 *
 * Follows the same auth/refresh/offline pattern as email-client.ts.
 * Gateway routes: /api/v1/documents/*
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
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
  WOPITokenResponse,
  SharePermission,
} from '../types/document-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

class OfflineError extends Error {
  constructor() {
    super('Aenderungen sind offline nicht moeglich.')
    this.name = 'OfflineError'
  }
}

let refreshPromise: Promise<string | null> | null = null

async function getToken(): Promise<string | undefined> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
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
// Folders
// ---------------------------------------------------------------------------

export const documentFolderApi = {
  list(params: ListFoldersParams = {}) {
    return request<ListFoldersResponse>(
      `/api/v1/documents/folders${qs(params as Record<string, unknown>)}`,
    )
  },

  get(id: string) {
    return request<{ folder: DocumentFolder }>(`/api/v1/documents/folders/${id}`)
  },

  create(data: CreateFolderRequest) {
    return request<{ folder: DocumentFolder }>('/api/v1/documents/folders', {
      method: 'POST',
      body: JSON.stringify(data),
    })
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

  getPath(id: string) {
    return request<{ segments: FolderPathSegment[] }>(
      `/api/v1/documents/folders/${id}/path`,
    )
  },

  initializeUserSpace() {
    return request<Record<string, never>>(
      '/api/v1/documents/folders/init/user',
      { method: 'POST' },
    )
  },

  initializeTeamSpace(teamId: string, teamName: string) {
    return request<Record<string, never>>(
      '/api/v1/documents/folders/init/team',
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
  list(params: ListFilesParams = {}) {
    const { tag_ids, ...rest } = params
    const queryParts = qs(rest as Record<string, unknown>)
    // Append tag_ids as repeated params
    const tagParams = tag_ids?.length
      ? (queryParts ? '&' : '?') +
        tag_ids.map((id) => `tag_ids=${encodeURIComponent(id)}`).join('&')
      : ''
    return request<ListFilesResponse>(
      `/api/v1/documents/files${queryParts}${tagParams}`,
    )
  },

  get(id: string) {
    return request<{ file: DocumentFile }>(`/api/v1/documents/files/${id}`)
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

  copy(id: string, targetFolderId: string) {
    return request<{ file: DocumentFile }>(
      `/api/v1/documents/files/${id}/copy`,
      {
        method: 'POST',
        body: JSON.stringify({ target_folder_id: targetFolderId }),
      },
    )
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
      `/api/v1/documents/files/${id}/download`,
    )
  },
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

export const documentVersionApi = {
  list(fileId: string) {
    return request<ListVersionsResponse>(
      `/api/v1/documents/files/${fileId}/versions`,
    )
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
      `/api/v1/documents/files/${fileId}/versions/${versionNumber}/revert`,
      { method: 'POST' },
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

  list(entityType: string, entityId: string) {
    return request<ListSharesResponse>(
      `/api/v1/documents/shares${qs({ entity_type: entityType, entity_id: entityId })}`,
    )
  },

  listSharedWithMe(entityType?: string) {
    return request<ListSharesResponse>(
      `/api/v1/documents/shares/shared-with-me${qs({ entity_type: entityType })}`,
    )
  },
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

export const documentTagApi = {
  list() {
    return request<ListTagsResponse>('/api/v1/documents/tags')
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
    return request<Record<string, never>>(
      `/api/v1/documents/files/${fileId}/tags/${tagId}`,
      { method: 'POST' },
    )
  },

  untagFile(fileId: string, tagId: string) {
    return request<Record<string, never>>(
      `/api/v1/documents/files/${fileId}/tags/${tagId}`,
      { method: 'DELETE' },
    )
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

  listByFile(fileId: string) {
    return request<ListEntityLinksResponse>(
      `/api/v1/documents/files/${fileId}/links`,
    )
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
    return request<WOPITokenResponse>(
      `/api/v1/documents/files/${fileId}/wopi-token`,
      { method: 'POST' },
    )
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
