/**
 * Browser-direct file upload via the gateway's presigned-PUT flow.
 *
 * This is steps 1+2 of the canonical upload flow (see useDocumentUpload):
 *   1. POST /api/v1/files/presign-upload  → { upload_url, object_key }
 *   2. PUT the bytes straight to MinIO     (XHR for per-file progress)
 * and returns the object_key. The caller registers that key against its own
 * entity (avatar → PATCH /auth/profile, inventar → POST .../attachments, …).
 *
 * The gateway never streams file bytes; tenant isolation is enforced server-side
 * via the {tenant_id}/{scope}/ object-key prefix.
 */
import { authenticatedRequest } from './authenticatedFetch'

/** Scopes accepted by the gateway presign allowlist (backend presign.go). */
export type PresignScope =
  | 'avatar'
  | 'chat'
  | 'rapporte'
  | 'vermietung'
  | 'vertraege'
  | 'fuhrpark'
  | 'inventar'
  | 'kontakte'
  | 'documents'

interface PresignResponse {
  upload_url: string
  object_key: string
}

export async function presignUpload(
  scope: PresignScope,
  file: File,
  onProgress?: (percent: number) => void,
): Promise<string> {
  const contentType = file.type || 'application/octet-stream'

  // 1) Short-lived presigned PUT URL from the gateway.
  const { upload_url, object_key } = await authenticatedRequest<PresignResponse>({
    method: 'POST',
    path: '/api/v1/files/presign-upload',
    body: {
      scope,
      file_name: file.name,
      content_type: contentType,
      size_bytes: file.size,
    },
  })

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

  return object_key
}
