/**
 * TanStack Query hook for task activity log.
 *
 * Fetches chronological activity entries for a task including
 * status changes, assignment changes, comments, and file operations.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '../client'
import { normalizeWireTimestamps } from '../wire-time'
import type { components } from '../types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type TaskActivity = components['schemas']['TaskActivityResponse']

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

export function useTaskActivities(taskId: string, page?: number) {
  return useQuery({
    queryKey: ['task-activities', taskId, { page }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/activities',
        {
          params: {
            path: { id: taskId },
            query: { page, page_size: 50 },
          },
        }
      )
      if (error) throw error
      return normalizeWireTimestamps(data)
    },
    enabled: !!taskId,
  })
}
