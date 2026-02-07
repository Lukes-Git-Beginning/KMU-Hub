/**
 * TanStack Query hooks for Deal CRUD operations.
 *
 * Includes stage-move mutation and stage-filtered deal queries
 * for the pipeline view.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

export interface DealListParams {
  page?: number
  page_size?: number
  search?: string
  stage_id?: string
  contact_id?: string
  company_id?: string
  owner_id?: string
  tag_ids?: string[]
  sort_by?: 'name' | 'value' | 'created_at' | 'updated_at' | 'expected_close_date'
  sort_desc?: boolean
}

export function useDeals(params?: DealListParams) {
  return useQuery({
    queryKey: ['deals', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/deals', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useDeal(id: string) {
  return useQuery({
    queryKey: ['deals', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/deals/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

export function useCreateDeal() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name: string
      value: number
      currency: string
      stage_id: string
      contact_id?: string
      company_id?: string
      owner_id?: string
      expected_close_date?: string
      notes?: string
      tag_ids?: string[]
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/deals', {
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deals'] })
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useUpdateDeal() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      name?: string
      value?: number
      currency?: string
      contact_id?: string
      company_id?: string
      owner_id?: string
      expected_close_date?: string
      notes?: string
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.PATCH('/api/v1/deals/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['deals'] })
      queryClient.invalidateQueries({ queryKey: ['deals', variables.id] })
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useDeleteDeal() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE('/api/v1/deals/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deals'] })
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}

export function useMoveDealToStage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, stage_id }: { id: string; stage_id: string }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/deals/{id}/stage',
        {
          params: { path: { id } },
          body: { stage_id },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['deals'] })
      queryClient.invalidateQueries({ queryKey: ['deals', variables.id] })
      queryClient.invalidateQueries({ queryKey: ['pipeline-stages'] })
    },
  })
}
