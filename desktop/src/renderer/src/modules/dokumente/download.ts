/**
 * Real file download: resolve the (expiring) download URL via the API and
 * trigger a browser download through a temporary anchor. Works in Electron
 * and plain Chromium; the demo handler returns a data: URL so downloads
 * behave realistically in demo mode too.
 */
import { documentFileApi } from '@/api/clients/document-client'

export async function downloadDocumentFile(id: string, filename: string): Promise<void> {
  const { url } = await documentFileApi.getDownloadURL(id)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}
