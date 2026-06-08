/**
 * Chat file upload hook (multipart) via XMLHttpRequest.
 *
 * Wraps POST /api/v1/files/upload — attaches a file to a channel and,
 * optionally, to a specific message. Real backend exists; a demo MSW
 * handler echoes a FileInfo so it works without a backend.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import { messageKeys } from './useMessages'
import type { components } from '@/api/types'

type FileInfo = components['schemas']['FileInfo']

interface UploadParams {
  channelId: string
  file: File
  messageId?: string
  onProgress?: (percent: number) => void
}

async function uploadChatFile({ channelId, file, messageId, onProgress }: UploadParams): Promise<FileInfo> {
  const { useAuthStore } = await import('@/stores/auth')
  const token = useAuthStore.getState().accessToken

  return new Promise<FileInfo>((resolve, reject) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('channel_id', channelId)
    if (messageId) formData.append('message_id', messageId)

    const xhr = new XMLHttpRequest()
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress?.(Math.round((e.loaded / e.total) * 100))
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
        reject(new Error(`Upload fehlgeschlagen: ${xhr.status}`))
      }
    }
    xhr.onerror = () => reject(new Error('Netzwerkfehler beim Upload'))
    xhr.open('POST', `${API_BASE_URL}/api/v1/files/upload`)
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.send(formData)
  })
}

export function useChatFileUpload() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: uploadChatFile,
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: messageKeys.list(variables.channelId) })
    },
  })
}

export type { FileInfo }
