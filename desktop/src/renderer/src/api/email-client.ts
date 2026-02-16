/**
 * Email API client — typed fetch wrapper for all email HTTP endpoints.
 *
 * Uses the same auth middleware pattern as the openapi-fetch apiClient:
 * - Reads access token from auth store
 * - Handles 401 with transparent token refresh
 * - Blocks mutations when offline
 *
 * Gateway routes: /api/v1/email/*
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  CreateEmailAccountRequest,
  UpdateEmailAccountRequest,
  TestEmailConnectionRequest,
  TestEmailConnectionResponse,
  EmailAccountInfo,
  EmailFolderInfo,
  EmailMessageInfo,
  ListMessagesParams,
  ListMessagesResponse,
  SendEmailRequest,
  SaveDraftRequest,
  ReplyEmailRequest,
  ForwardEmailRequest,
  EmailSignatureInfo,
  CreateSignatureRequest,
  UpdateSignatureRequest,
  EmailContactLinkInfo,
  GetContactEmailsResponse,
  SyncStatusResponse,
  TriggerSyncResponse,
  UploadAttachmentResponse,
  AttachmentDownloadURLResponse,
} from './email-types'

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

  // Handle 401 with refresh
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
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '' && v !== null)
  if (entries.length === 0) return ''
  return '?' + new URLSearchParams(
    entries.map(([k, v]) => [k, String(v)]),
  ).toString()
}

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------

export const emailAccountApi = {
  create(req: CreateEmailAccountRequest) {
    return request<{ account: EmailAccountInfo }>('/api/v1/email/accounts/', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },

  get(userId: string) {
    return request<{ account: EmailAccountInfo }>(`/api/v1/email/accounts/${qs({ user_id: userId })}`)
  },

  update(id: string, req: Omit<UpdateEmailAccountRequest, 'id'>) {
    return request<{ account: EmailAccountInfo }>(`/api/v1/email/accounts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/email/accounts/${id}`, {
      method: 'DELETE',
    })
  },

  testConnection(req: TestEmailConnectionRequest) {
    return request<TestEmailConnectionResponse>('/api/v1/email/accounts/test', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },
}

// ---------------------------------------------------------------------------
// Folders
// ---------------------------------------------------------------------------

export const emailFolderApi = {
  list(accountId: string) {
    return request<{ folders: EmailFolderInfo[] }>(`/api/v1/email/folders/${qs({ account_id: accountId })}`)
  },

  get(id: string) {
    return request<{ folder: EmailFolderInfo }>(`/api/v1/email/folders/${id}`)
  },

  sync(accountId: string) {
    return request<{ folders: EmailFolderInfo[] }>('/api/v1/email/folders/sync', {
      method: 'POST',
      body: JSON.stringify({ account_id: accountId }),
    })
  },
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export const emailMessageApi = {
  list(params: ListMessagesParams) {
    return request<ListMessagesResponse>(`/api/v1/email/messages/${qs(params as Record<string, unknown>)}`)
  },

  get(id: string) {
    return request<{ message: EmailMessageInfo }>(`/api/v1/email/messages/${id}`)
  },

  getThread(threadId: string) {
    return request<{ messages: EmailMessageInfo[] }>(`/api/v1/email/messages/thread/${threadId}`)
  },

  markRead(id: string) {
    return request<Record<string, never>>(`/api/v1/email/messages/${id}/read`, {
      method: 'POST',
    })
  },

  markUnread(id: string) {
    return request<Record<string, never>>(`/api/v1/email/messages/${id}/unread`, {
      method: 'POST',
    })
  },

  toggleStar(id: string) {
    return request<{ is_starred: boolean }>(`/api/v1/email/messages/${id}/star`, {
      method: 'POST',
    })
  },

  moveToFolder(messageId: string, targetFolderId: string) {
    return request<Record<string, never>>(`/api/v1/email/messages/${messageId}/move`, {
      method: 'POST',
      body: JSON.stringify({ target_folder_id: targetFolderId }),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/email/messages/${id}`, {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Send / Compose
// ---------------------------------------------------------------------------

export const emailSendApi = {
  send(req: SendEmailRequest) {
    return request<{ message: EmailMessageInfo }>('/api/v1/email/send/', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },

  saveDraft(req: SaveDraftRequest) {
    return request<{ message: EmailMessageInfo }>('/api/v1/email/send/draft', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },

  reply(req: ReplyEmailRequest) {
    return request<{ message: EmailMessageInfo }>('/api/v1/email/send/reply', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },

  forward(req: ForwardEmailRequest) {
    return request<{ message: EmailMessageInfo }>('/api/v1/email/send/forward', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },
}

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

export const emailSignatureApi = {
  list(userId: string) {
    return request<{ signatures: EmailSignatureInfo[] }>(`/api/v1/email/signatures/${qs({ user_id: userId })}`)
  },

  get(id: string) {
    return request<{ signature: EmailSignatureInfo }>(`/api/v1/email/signatures/${id}`)
  },

  create(req: CreateSignatureRequest) {
    return request<{ signature: EmailSignatureInfo }>('/api/v1/email/signatures/', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  },

  update(id: string, req: Omit<UpdateSignatureRequest, 'id'>) {
    return request<{ signature: EmailSignatureInfo }>(`/api/v1/email/signatures/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    })
  },

  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/email/signatures/${id}`, {
      method: 'DELETE',
    })
  },

  setDefault(id: string, userId: string) {
    return request<{ signature: EmailSignatureInfo }>(`/api/v1/email/signatures/${id}/default`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    })
  },
}

// ---------------------------------------------------------------------------
// CRM Links
// ---------------------------------------------------------------------------

export const emailLinkApi = {
  getByMessage(messageId: string) {
    return request<{ links: EmailContactLinkInfo[] }>(`/api/v1/email/links/message/${messageId}`)
  },

  link(messageId: string, contactId: string, linkType: 'auto' | 'manual' = 'manual') {
    return request<{ link: EmailContactLinkInfo }>('/api/v1/email/links/', {
      method: 'POST',
      body: JSON.stringify({ message_id: messageId, contact_id: contactId, link_type: linkType }),
    })
  },

  unlink(messageId: string, contactId: string) {
    return request<Record<string, never>>('/api/v1/email/links/', {
      method: 'DELETE',
      body: JSON.stringify({ message_id: messageId, contact_id: contactId }),
    })
  },

  getContactEmails(contactId: string, page?: number, perPage?: number) {
    return request<GetContactEmailsResponse>(
      `/api/v1/email/links/contact/${contactId}${qs({ page, per_page: perPage })}`,
    )
  },
}

// ---------------------------------------------------------------------------
// Sync
// ---------------------------------------------------------------------------

export const emailSyncApi = {
  trigger(accountId: string) {
    return request<TriggerSyncResponse>('/api/v1/email/sync/trigger', {
      method: 'POST',
      body: JSON.stringify({ account_id: accountId }),
    })
  },

  getStatus(accountId: string) {
    return request<SyncStatusResponse>(`/api/v1/email/sync/status${qs({ account_id: accountId })}`)
  },

  setReadFlag(messageId: string, isRead: boolean) {
    return request<Record<string, never>>('/api/v1/email/sync/read-flag', {
      method: 'POST',
      body: JSON.stringify({ message_id: messageId, is_read: isRead }),
    })
  },
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

export const emailAttachmentApi = {
  upload(accountId: string, file: File) {
    const formData = new FormData()
    formData.append('account_id', accountId)
    formData.append('file', file)

    return request<UploadAttachmentResponse>('/api/v1/email/attachments/upload', {
      method: 'POST',
      body: formData,
      // Do NOT set Content-Type — browser sets it with boundary for multipart
    })
  },

  getDownloadURL(id: string) {
    return request<AttachmentDownloadURLResponse>(`/api/v1/email/attachments/${id}/download`)
  },
}

// ---------------------------------------------------------------------------
// Contact Import/Export
// ---------------------------------------------------------------------------

export const emailContactApi = {
  importCSV(fileContent: string, fieldMapping: Record<string, string>, visibility: string, mergeByEmail: boolean) {
    return request<{ imported_count: number; merged_count: number; skipped_count: number; errors: Array<{ row: number; field: string; error: string }> }>(
      '/api/v1/email/contacts/import/csv',
      {
        method: 'POST',
        body: JSON.stringify({ file_content: fileContent, field_mapping: fieldMapping, visibility, merge_by_email: mergeByEmail }),
      },
    )
  },

  importVCard(fileContent: string, visibility: string, mergeByEmail: boolean) {
    return request<{ imported_count: number; merged_count: number; skipped_count: number; errors: Array<{ row: number; field: string; error: string }> }>(
      '/api/v1/email/contacts/import/vcard',
      {
        method: 'POST',
        body: JSON.stringify({ file_content: fileContent, visibility, merge_by_email: mergeByEmail }),
      },
    )
  },

  exportCSV(contactIds: string[], fields: string[]) {
    return request<{ file_content: string; filename: string }>('/api/v1/email/contacts/export/csv', {
      method: 'POST',
      body: JSON.stringify({ contact_ids: contactIds, fields }),
    })
  },

  exportVCard(contactIds: string[]) {
    return request<{ file_content: string; filename: string }>('/api/v1/email/contacts/export/vcard', {
      method: 'POST',
      body: JSON.stringify({ contact_ids: contactIds }),
    })
  },
}
