/**
 * Lightweight fetch wrapper for CRM import/export endpoints.
 *
 * Uses raw fetch (not openapi-fetch) because these endpoints handle
 * multipart file uploads and binary file downloads, which don't fit
 * the typed openapi-fetch pattern well.
 *
 * Follows the same auth + retry pattern as security-client.ts.
 */
import { API_BASE_URL } from '@/lib/constants'
import type { ImportPreview, ImportResult } from './crm-types'

// ---------------------------------------------------------------------------
// Auth helper
// ---------------------------------------------------------------------------

async function getAuthHeaders(): Promise<Record<string, string>> {
  const { useAuthStore } = await import('@/stores/auth')
  const token = useAuthStore.getState().accessToken
  const headers: Record<string, string> = {}
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return headers
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()
      if (!newToken) {
        store.logout()
        throw new Error('Authentication expired')
      }
      throw new Error('Token refreshed, please retry')
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
// Import endpoints
// ---------------------------------------------------------------------------

/** Upload CSV file for preview (first 5 rows + detected field mapping). */
export async function previewImportCSV(file: File): Promise<ImportPreview> {
  const headers = await getAuthHeaders()
  const formData = new FormData()
  formData.append('file', file)

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/import/preview`,
    { method: 'POST', headers, body: formData },
  )
  return handleResponse<ImportPreview>(response)
}

/** Import contacts from CSV file with field mapping and options. */
export async function importContactsCSV(
  file: File,
  fieldMapping: Record<string, string>,
  visibility: string,
  mergeByEmail: boolean,
): Promise<ImportResult> {
  const headers = await getAuthHeaders()
  const formData = new FormData()
  formData.append('file', file)
  formData.append('field_mapping', JSON.stringify(fieldMapping))
  formData.append('visibility', visibility)
  formData.append('merge_by_email', String(mergeByEmail))

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/import/csv`,
    { method: 'POST', headers, body: formData },
  )
  return handleResponse<ImportResult>(response)
}

/** Import contacts from vCard file with options. */
export async function importContactsVCard(
  file: File,
  visibility: string,
  mergeByEmail: boolean,
): Promise<ImportResult> {
  const headers = await getAuthHeaders()
  const formData = new FormData()
  formData.append('file', file)
  formData.append('visibility', visibility)
  formData.append('merge_by_email', String(mergeByEmail))

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/import/vcard`,
    { method: 'POST', headers, body: formData },
  )
  return handleResponse<ImportResult>(response)
}

// ---------------------------------------------------------------------------
// Export endpoints
// ---------------------------------------------------------------------------

/** Export selected contacts to CSV, returns downloadable blob. */
export async function exportContactsCSV(
  contactIds: string[],
  fields: string[],
): Promise<Blob> {
  const headers = await getAuthHeaders()
  headers['Content-Type'] = 'application/json'

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/export/csv`,
    {
      method: 'POST',
      headers,
      body: JSON.stringify({ contact_ids: contactIds, fields }),
    },
  )

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Export failed: ${response.status}`,
    )
  }

  return response.blob()
}

/** Export selected contacts to vCard, returns downloadable blob. */
export async function exportContactsVCard(contactIds: string[]): Promise<Blob> {
  const headers = await getAuthHeaders()
  headers['Content-Type'] = 'application/json'

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/export/vcard`,
    {
      method: 'POST',
      headers,
      body: JSON.stringify({ contact_ids: contactIds }),
    },
  )

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Export failed: ${response.status}`,
    )
  }

  return response.blob()
}

// ---------------------------------------------------------------------------
// Visibility endpoint
// ---------------------------------------------------------------------------

/** Update contact visibility (shared/personal). */
export async function updateContactVisibility(
  contactId: string,
  visibility: string,
): Promise<void> {
  const headers = await getAuthHeaders()
  headers['Content-Type'] = 'application/json'

  const response = await fetch(
    `${API_BASE_URL}/api/v1/crm/contacts/${contactId}/visibility`,
    {
      method: 'PUT',
      headers,
      body: JSON.stringify({ visibility }),
    },
  )

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Update failed: ${response.status}`,
    )
  }
}
