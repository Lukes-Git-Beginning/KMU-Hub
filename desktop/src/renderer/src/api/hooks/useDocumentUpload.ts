/**
 * Document upload hook using the presigned-URL flow.
 *
 * The gateway does not stream file bytes; uploads go browser-direct to object
 * storage (MinIO) via a short-lived presigned PUT URL. Three steps:
 *   1. POST /api/v1/files/presign-upload  → { upload_url, object_key }
 *   2. PUT the bytes to upload_url        (XHR for per-file progress)
 *   3. POST /api/v1/documents/files       → register metadata, returns the file
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import { generateIdempotencyKey } from '../idempotency'
import type { DocumentFile } from '../types/document-types'

interface UploadParams {
  folderId: string
  file: File
  onProgress?: (percent: number) => void
}

interface PresignResponse {
  upload_url: string
  object_key: string
}

async function uploadFile({
  folderId,
  file,
  onProgress,
}: UploadParams): Promise<DocumentFile> {
  const { useAuthStore } = await import('@/stores/auth')
  const token = useAuthStore.getState().accessToken

  const authHeaders = (): Record<string, string> => {
    const h: Record<string, string> = {
      'Content-Type': 'application/json',
      'Idempotency-Key': generateIdempotencyKey(),
    }
    if (token) h['Authorization'] = `Bearer ${token}`
    return h
  }

  const contentType = file.type || 'application/octet-stream'

  // 1) Request a short-lived presigned PUT URL from the gateway.
  const presignRes = await fetch(`${API_BASE_URL}/api/v1/files/presign-upload`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({
      scope: 'documents',
      file_name: file.name,
      content_type: contentType,
      size_bytes: file.size,
    }),
  })
  if (!presignRes.ok) {
    if (presignRes.status === 401) throw new Error('Session abgelaufen')
    const err = await presignRes.json().catch(() => ({}))
    throw new Error(err.error ?? `Upload-Vorbereitung fehlgeschlagen: ${presignRes.status}`)
  }
  const { upload_url, object_key } = (await presignRes.json()) as PresignResponse

  // 2) PUT the bytes straight to object storage. The presigned URL carries its
  //    own signature — do NOT attach the bearer token. XHR gives upload progress.
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress?.(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () =>
      xhr.status >= 200 && xhr.status < 300
        ? resolve()
        : reject(new Error(`Upload fehlgeschlagen: ${xhr.status}`))
    xhr.onerror = () => reject(new Error('Netzwerkfehler beim Upload'))
    xhr.onabort = () => reject(new Error('Upload abgebrochen'))
    xhr.open('PUT', upload_url)
    xhr.setRequestHeader('Content-Type', contentType)
    xhr.send(file)
  })

  // 3) Register the uploaded object so it appears in the folder.
  const registerRes = await fetch(`${API_BASE_URL}/api/v1/documents/files`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({
      folder_id: folderId,
      filename: file.name,
      mime_type: contentType,
      file_size: file.size,
      storage_key: object_key,
    }),
  })
  if (!registerRes.ok) {
    if (registerRes.status === 401) throw new Error('Session abgelaufen')
    const err = await registerRes.json().catch(() => ({}))
    throw new Error(err.error ?? `Registrierung fehlgeschlagen: ${registerRes.status}`)
  }
  const registered = await registerRes.json()
  return (registered.file ?? registered) as DocumentFile
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
