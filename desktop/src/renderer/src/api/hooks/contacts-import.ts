/**
 * TanStack Query hooks for contact import, export, and visibility.
 *
 * Uses the crm-import-client for raw fetch calls (multipart uploads,
 * blob downloads) rather than the typed apiClient.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  previewImportCSV,
  importContactsCSV,
  importContactsVCard,
  exportContactsCSV,
  exportContactsVCard,
  updateContactVisibility,
} from '../crm-import-client'

// ---------------------------------------------------------------------------
// Import hooks
// ---------------------------------------------------------------------------

/** Preview CSV file: returns columns, sample rows, and detected field mapping. */
export function usePreviewImportCSV() {
  return useMutation({
    mutationFn: (file: File) => previewImportCSV(file),
  })
}

/** Import contacts from CSV with field mapping and options. */
export function useImportContactsCSV() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      file,
      fieldMapping,
      visibility,
      mergeByEmail,
    }: {
      file: File
      fieldMapping: Record<string, string>
      visibility: string
      mergeByEmail: boolean
    }) => importContactsCSV(file, fieldMapping, visibility, mergeByEmail),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
    },
  })
}

/** Import contacts from vCard file with options. */
export function useImportContactsVCard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      file,
      visibility,
      mergeByEmail,
    }: {
      file: File
      visibility: string
      mergeByEmail: boolean
    }) => importContactsVCard(file, visibility, mergeByEmail),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Export hooks
// ---------------------------------------------------------------------------

/** Export selected contacts to CSV. Triggers browser file download. */
export function useExportContactsCSV() {
  return useMutation({
    mutationFn: async ({
      contactIds,
      fields,
    }: {
      contactIds: string[]
      fields: string[]
    }) => {
      const blob = await exportContactsCSV(contactIds, fields)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `kontakte-${new Date().toISOString().slice(0, 10)}.csv`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },
  })
}

/** Export selected contacts to vCard. Triggers browser file download. */
export function useExportContactsVCard() {
  return useMutation({
    mutationFn: async ({ contactIds }: { contactIds: string[] }) => {
      const blob = await exportContactsVCard(contactIds)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `kontakte-${new Date().toISOString().slice(0, 10)}.vcf`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },
  })
}

// ---------------------------------------------------------------------------
// Visibility hook
// ---------------------------------------------------------------------------

/** Update visibility of a single contact. Invalidates contacts cache. */
export function useUpdateContactVisibility() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      contactId,
      visibility,
    }: {
      contactId: string
      visibility: string
    }) => updateContactVisibility(contactId, visibility),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
    },
  })
}
