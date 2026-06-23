/**
 * TanStack Query hooks for Pipeline Stage operations.
 *
 * Pipeline stages define the columns in the deal pipeline/Kanban view.
 * Stages are returned sorted by sort_order from the backend.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { components } from '../types'
import { apiClient } from '../client'
import { authenticatedRequest } from '../utils/authenticatedFetch'
import { dual } from '../casing'

type PipelineStageInfo = components['schemas']['PipelineStageInfo']

/** Normalize a backend pipeline stage (snake_case wire) to the camelCase shape
 *  the pipeline/analytics/editor views read (X-3). `total_value` is omitted when
 *  zero → defaults to 0. */
export function backendStageToUI(s: unknown): PipelineStageInfo {
  const raw = (s ?? {}) as Record<string, unknown>
  return {
    ...raw,
    sortOrder: dual<number>(s, 'sortOrder') ?? 0,
    dealCount: dual<number>(s, 'dealCount') ?? 0,
    totalValue: dual<number>(s, 'totalValue') ?? 0,
    isWon: dual<boolean>(s, 'isWon') ?? false,
    isLost: dual<boolean>(s, 'isLost') ?? false,
    createdAt: dual<string>(s, 'createdAt') ?? '',
  } as PipelineStageInfo
}

export function usePipelineStages() {
  return useQuery({
    queryKey: ['pipeline-stages'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/pipeline-stages')
      if (error) throw error
      return { ...data, stages: (data?.stages ?? []).map(backendStageToUI) }
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
      // Update is PUT server-side (spec mislabels it PATCH, X-3).
      return authenticatedRequest({
        method: 'PUT',
        path: `/api/v1/pipeline-stages/${id}`,
        body,
      })
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
