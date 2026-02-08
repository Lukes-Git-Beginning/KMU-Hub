/**
 * TanStack Query hooks for task comment operations.
 *
 * Includes list, create, update, and delete mutations for task comments.
 * Mutations invalidate both task-comments and task-activities query keys
 * since commenting creates an activity entry server-side.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import type { components } from '../types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type TaskComment = components['schemas']['TaskCommentResponse']

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

export function useTaskComments(taskId: string, page?: number) {
  return useQuery({
    queryKey: ['task-comments', taskId, { page }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/tasks/{id}/comments',
        {
          params: {
            path: { id: taskId },
            query: { page, page_size: 50 },
          },
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

export function useCreateComment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      taskId,
      content,
      quoted_comment_id,
    }: {
      taskId: string
      content: string
      quoted_comment_id?: string
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/tasks/{id}/comments',
        {
          params: { path: { id: taskId } },
          body: { content, quoted_comment_id },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['task-comments', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['task-activities', variables.taskId],
      })
    },
  })
}

export function useUpdateComment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      commentId,
      taskId,
      content,
    }: {
      commentId: string
      taskId: string
      content: string
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/task-comments/{id}',
        {
          params: { path: { id: commentId } },
          body: { content },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['task-comments', variables.taskId],
      })
    },
  })
}

export function useDeleteComment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      commentId,
      taskId,
    }: {
      commentId: string
      taskId: string
    }) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/task-comments/{id}',
        {
          params: { path: { id: commentId } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['task-comments', variables.taskId],
      })
      queryClient.invalidateQueries({
        queryKey: ['task-activities', variables.taskId],
      })
    },
  })
}
