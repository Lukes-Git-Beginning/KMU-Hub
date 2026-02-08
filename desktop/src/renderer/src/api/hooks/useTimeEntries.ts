/**
 * TanStack Query hooks for time tracking operations.
 *
 * Includes timer start/stop, active timer polling, manual entry CRUD,
 * time entry list, and task time summary.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { useAuthStore } from '@/stores/auth'

// ---------------------------------------------------------------------------
// Timer queries
// ---------------------------------------------------------------------------

/**
 * Poll the active timer for the current user.
 * Polls every 60 seconds to keep elapsed display roughly synced.
 */
export function useActiveTimer() {
  return useQuery({
    queryKey: ['timer', 'active'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/timer/active')
      if (error) throw error
      return data
    },
    refetchInterval: 60_000,
  })
}

// ---------------------------------------------------------------------------
// Time entry queries
// ---------------------------------------------------------------------------

export function useTimeEntries(taskId: string, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['time-entries', taskId, { page, pageSize }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/time-entries',
        {
          params: {
            path: { id: taskId },
            query: { page, page_size: pageSize },
          },
        }
      )
      if (error) throw error
      return data
    },
    enabled: !!taskId,
  })
}

export function useTaskTimeSummary(taskId: string) {
  return useQuery({
    queryKey: ['time-summary', taskId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/time-summary',
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
// Timer mutations
// ---------------------------------------------------------------------------

export function useStartTimer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (taskId: string) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/tasks/{id}/timer/start',
        {
          params: { path: { id: taskId } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, taskId) => {
      queryClient.invalidateQueries({ queryKey: ['timer', 'active'] })
      queryClient.invalidateQueries({ queryKey: ['time-entries', taskId] })
      queryClient.invalidateQueries({ queryKey: ['time-summary', taskId] })
    },
  })
}

export function useStopTimer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await apiClient.POST('/api/v1/timer/stop')
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['timer'] })
      queryClient.invalidateQueries({ queryKey: ['time-entries'] })
      queryClient.invalidateQueries({ queryKey: ['time-summary'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Manual time entry mutations
// ---------------------------------------------------------------------------

export function useAddManualTimeEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      taskId,
      started_at,
      duration_seconds,
      description,
    }: {
      taskId: string
      started_at: string
      duration_seconds: number
      description?: string
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/tasks/{id}/time-entries',
        {
          params: { path: { id: taskId } },
          body: { started_at, duration_seconds, description },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['time-entries', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['time-summary', variables.taskId],
      })
    },
  })
}

export function useUpdateTimeEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      entryId,
      ...body
    }: {
      entryId: string
      taskId: string // for cache invalidation only
      started_at?: string
      duration_seconds?: number
      description?: string
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/time-entries/{id}',
        {
          params: { path: { id: entryId } },
          body,
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['time-entries', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['time-summary', variables.taskId],
      })
    },
  })
}

export function useDeleteTimeEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      entryId,
    }: {
      entryId: string
      taskId: string // for cache invalidation only
    }) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/time-entries/{id}',
        {
          params: { path: { id: entryId } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['time-entries', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['time-summary', variables.taskId],
      })
    },
  })
}
