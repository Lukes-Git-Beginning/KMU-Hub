/**
 * TanStack Query hooks for Task CRUD operations.
 *
 * Includes task list/detail, my tasks, subtasks, move, search,
 * and mutation hooks for create/update/delete.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { useAuthStore } from '@/stores/auth'

// ---------------------------------------------------------------------------
// Query params
// ---------------------------------------------------------------------------

export interface TaskListParams {
  project_id?: string
  assignee_id?: string
  status_id?: string
  priority?: 'urgent' | 'high' | 'normal' | 'low'
  parent_task_id?: string
  search?: string
  sort_by?: string
  sort_desc?: boolean
  include_completed?: boolean
  page?: number
  page_size?: number
}

export interface SearchTasksParams {
  q: string
  project_ids?: string[]
  assignee_ids?: string[]
  include_completed?: boolean
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Task queries
// ---------------------------------------------------------------------------

export function useTasks(params?: TaskListParams) {
  return useQuery({
    queryKey: ['tasks', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/tasks', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useTask(id: string) {
  return useQuery({
    queryKey: ['tasks', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/tasks/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

/**
 * Fetch tasks assigned to the current user across all projects.
 * Automatically sets assignee_id to the current user's ID.
 */
export function useMyTasks(params?: Omit<TaskListParams, 'assignee_id'>) {
  const userId = useAuthStore((s) => s.user?.id)
  return useQuery({
    queryKey: ['tasks', 'my', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/tasks', {
        params: {
          query: {
            ...params,
            assignee_id: userId,
          },
        },
      })
      if (error) throw error
      return data
    },
    enabled: !!userId,
  })
}

export function useSubtasks(taskId: string, recursive?: boolean) {
  return useQuery({
    queryKey: ['tasks', taskId, 'subtasks', { recursive }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/subtasks',
        {
          params: {
            path: { id: taskId },
            query: { recursive },
          },
        }
      )
      if (error) throw error
      return data
    },
    enabled: !!taskId,
  })
}

export function useSearchTasks(params: SearchTasksParams) {
  return useQuery({
    queryKey: ['tasks', 'search', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/work/search', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
    enabled: !!params.q && params.q.length >= 2,
  })
}

// ---------------------------------------------------------------------------
// Task mutations
// ---------------------------------------------------------------------------

export function useCreateTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      title: string
      priority: 'urgent' | 'high' | 'normal' | 'low'
      project_id?: string
      description?: string
      status_id?: string
      assignee_id?: string
      parent_task_id?: string
      due_date?: string
      custom_fields?: Array<{ field_id: string; value: string }>
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/tasks', {
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useUpdateTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      title?: string
      description?: string
      status_id?: string
      priority?: 'urgent' | 'high' | 'normal' | 'low'
      assignee_id?: string
      due_date?: string
      custom_fields?: Array<{ field_id: string; value: string }>
    }) => {
      const { data, error } = await apiClient.PUT('/api/v1/tasks/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['tasks', variables.id] })
    },
  })
}

export function useDeleteTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE('/api/v1/tasks/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useMoveTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      status_id,
      sort_order,
    }: {
      id: string
      status_id: string
      sort_order?: number
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/tasks/{id}/move',
        {
          params: { path: { id } },
          body: { status_id, sort_order },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['tasks', variables.id] })
    },
  })
}
