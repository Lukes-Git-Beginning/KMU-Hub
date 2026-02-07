/**
 * TanStack Query hooks for Pipeline Stage operations.
 *
 * Pipeline stages define the columns in the deal pipeline/Kanban view.
 * Stages are returned sorted by sort_order from the backend.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

export function usePipelineStages() {
  return useQuery({
    queryKey: ['pipeline-stages'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/pipeline-stages')
      if (error) throw error
      return data
    },
  })
}

export function useCreatePipelineStage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name: string
      color: string
      is_won: boolean
      is_lost: boolean
      probability: number
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/pipeline-stages', {
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useUpdatePipelineStage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      name?: string
      color?: string
      is_won?: boolean
      is_lost?: boolean
      probability?: number
    }) => {
      const { data, error } = await apiClient.PATCH(
        '/api/v1/pipeline-stages/{id}',
        {
          params: { path: { id } },
          body,
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useDeletePipelineStage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/pipeline-stages/{id}',
        {
          params: { path: { id } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useReorderPipelineStages() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (stage_ids: string[]) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/pipeline-stages/reorder',
        {
          body: { stage_ids },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}
