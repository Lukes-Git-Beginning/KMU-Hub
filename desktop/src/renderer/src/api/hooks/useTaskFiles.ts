/**
 * TanStack Query hooks for task file attachment operations.
 *
 * Task file workflow: upload via /api/v1/files/upload (multipart),
 * then attach metadata to task via /api/v1/tasks/{id}/files.
 * Download via presigned URL from /api/v1/files/{id}/download.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { API_BASE_URL } from '@/lib/constants'
import { useAuthStore } from '@/stores/auth'
import type { components } from '../types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type TaskFile = components['schemas']['TaskFileResponse']

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

export function useTaskFiles(taskId: string) {
  return useQuery({
    queryKey: ['task-files', taskId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/files',
        {
          params: { path: { id: taskId } },
        }
      )
      if (error) throw error
      return data
    },
    enabled: !!taskId,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/**
 * Upload a file to MinIO then attach it to a task.
 *
 * Two-step process:
 * 1. Upload file via multipart POST to /api/v1/files/upload
 * 2. Attach returned file metadata to the task via POST /api/v1/tasks/{id}/files
 */
export function useAttachFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      taskId,
      file,
    }: {
      taskId: string
      file: File
    }) => {
      // Step 1: Upload file via multipart form data
      // openapi-fetch does not support multipart natively, so use fetch directly
      const formData = new FormData()
      formData.append('file', file)
      // The upload endpoint requires a channel_id but for task files we use a
      // task-specific identifier. The backend work file upload uses a dedicated
      // endpoint pattern. We'll use the task files endpoint directly which
      // accepts the file metadata after upload.
      // For now, upload directly and attach metadata.
      const token = useAuthStore.getState().accessToken
      const uploadRes = await fetch(`${API_BASE_URL}/api/v1/files/upload`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      })

      if (!uploadRes.ok) {
        const errBody = await uploadRes.text().catch(() => '')
        throw new Error(
          `Datei-Upload fehlgeschlagen (${uploadRes.status}): ${errBody}`
        )
      }

      const uploadData = await uploadRes.json()
      const storageKey: string = uploadData.storage_key ?? uploadData.key ?? ''
      const thumbnailKey: string = uploadData.thumbnail_key ?? ''

      // Step 2: Attach file metadata to the task
      const { data, error } = await apiClient.POST(
        '/api/v1/tasks/{id}/files',
        {
          params: { path: { id: taskId } },
          body: {
            filename: file.name,
            mime_type: file.type || 'application/octet-stream',
            file_size: file.size,
            storage_key: storageKey,
            thumbnail_key: thumbnailKey || undefined,
          },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['task-files', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['task-activities', variables.taskId],
      })
    },
  })
}

export function useRemoveFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      fileId,
      taskId: _taskId,
    }: {
      fileId: string
      taskId: string
    }) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/task-files/{id}',
        {
          params: { path: { id: fileId } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['task-files', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['task-activities', variables.taskId],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Get a presigned download URL for a file.
 * Uses the chat file download endpoint which handles MinIO presigned URLs.
 */
export async function getFileDownloadUrl(fileId: string): Promise<string> {
  const { data, error } = await apiClient.GET(
    '/api/v1/files/{id}/download',
    {
      params: { path: { id: fileId } },
    }
  )
  if (error) throw error
  return data?.url ?? ''
}

/**
 * Format file size in human-readable format.
 */
export function formatFileSize(bytes?: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const size = bytes / Math.pow(1024, i)
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
