/**
 * Lightweight fetch wrapper for CRM import/export endpoints.
 *
 * Uses raw fetch (not openapi-fetch) because these endpoints handle
 * multipart file uploads and binary file downloads, which don't fit
 * the typed openapi-fetch pattern well.
 *
 * Follows the same auth + idempotency pattern as authenticatedFetch.ts.
 */
import type { ImportPreview, ImportResult } from './crm-types'
import { authenticatedRequest, authenticatedBlobRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Import endpoints
// ---------------------------------------------------------------------------

/** Upload CSV file for preview (first 5 rows + detected field mapping). */
export async function previewImportCSV(file: File): Promise<ImportPreview> {
  const formData = new FormData()
  formData.append('file', file)
  return authenticatedRequest<ImportPreview>({
    method: 'POST',
    path: '/api/v1/crm/contacts/import/preview',
    body: formData,
  })
}

/** Import contacts from CSV file with field mapping and options. */
export async function importContactsCSV(
  file: File,
  fieldMapping: Record<string, string>,
  visibility: string,
  mergeByEmail: boolean,
): Promise<ImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('field_mapping', JSON.stringify(fieldMapping))
  formData.append('visibility', visibility)
  formData.append('merge_by_email', String(mergeByEmail))
  return authenticatedRequest<ImportResult>({
    method: 'POST',
    path: '/api/v1/crm/contacts/import/csv',
    body: formData,
  })
}

/** Import contacts from vCard file with options. */
export async function importContactsVCard(
  file: File,
  visibility: string,
  mergeByEmail: boolean,
): Promise<ImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('visibility', visibility)
  formData.append('merge_by_email', String(mergeByEmail))
  return authenticatedRequest<ImportResult>({
    method: 'POST',
    path: '/api/v1/crm/contacts/import/vcard',
    body: formData,
  })
}

// ---------------------------------------------------------------------------
// Export endpoints
// ---------------------------------------------------------------------------

/** Export selected contacts to CSV, returns downloadable blob. */
export async function exportContactsCSV(
  contactIds: string[],
  fields: string[],
): Promise<Blob> {
  const response = await authenticatedBlobRequest({
    method: 'POST',
    path: '/api/v1/crm/contacts/export/csv',
    body: { contact_ids: contactIds, fields },
  })

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
  const response = await authenticatedBlobRequest({
    method: 'POST',
    path: '/api/v1/crm/contacts/export/vcard',
    body: { contact_ids: contactIds },
  })

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
  await authenticatedRequest<void>({
    method: 'PUT',
    path: `/api/v1/crm/contacts/${contactId}/visibility`,
    body: { visibility },
  })
}
