/**
 * Document upload hook using XMLHttpRequest for progress tracking.
 *
 * Uses XHR instead of fetch to get per-file upload progress via
 * XMLHttpRequest.upload.onprogress events. Supports multi-file upload
 * by calling mutateAsync in a loop with individual progress callbacks.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import type { DocumentFile } from '../types/document-types'

interface UploadParams {
  folderId: string
  file: File
  onProgress?: (percent: number) => void
}

async function uploadFile({
  folderId,
  file,
  onProgress,
}: UploadParams): Promise<DocumentFile> {
  const { useAuthStore } = await import('@/stores/auth')
  const token = useAuthStore.getState().accessToken

  return new Promise<DocumentFile>((resolve, reject) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder_id', folderId)

    const xhr = new XMLHttpRequest()

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        onProgress?.(Math.round((e.loaded / e.total) * 100))
      }
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const response = JSON.parse(xhr.responseText)
          resolve(response.file ?? response)
        } catch {
          reject(new Error('Ungültige Server-Antwort'))
        }
      } else if (xhr.status === 401) {
        reject(new Error('Session abgelaufen'))
      } else {
        try {
          const errorBody = JSON.parse(xhr.responseText)
          reject(
            new Error(
              errorBody.error ?? errorBody.message ?? `Upload fehlgeschlagen: ${xhr.status}`,
            ),
          )
        } catch {
          reject(new Error(`Upload fehlgeschlagen: ${xhr.status}`))
        }
      }
    }

    xhr.onerror = () => reject(new Error('Netzwerkfehler beim Upload'))
    xhr.onabort = () => reject(new Error('Upload abgebrochen'))

    xhr.open('POST', `${API_BASE_URL}/api/v1/documents/files/upload`)
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.send(formData)
  })
}

/**
 * Single-file upload mutation with progress callback.
 *
 * Usage:
 * ```tsx
 * const upload = useDocumentUpload()
 * upload.mutateAsync({
 *   folderId: 'folder-123',
 *   file: selectedFile,
 *   onProgress: (pct) => setProgress(pct),
 * })
 * ```
 */
export function useDocumentUpload() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: uploadFile,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

/**
 * Upload status for a single file in a multi-upload batch.
 */
export interface UploadFileStatus {
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'complete' | 'error'
  error?: string
}
