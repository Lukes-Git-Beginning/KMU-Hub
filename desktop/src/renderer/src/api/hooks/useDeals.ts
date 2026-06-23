/**
 * TanStack Query hooks for Deal CRUD operations.
 *
 * Includes stage-move mutation and stage-filtered deal queries
 * for the pipeline view.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { components } from '../types'
import { apiClient } from '../client'
import { authenticatedRequest } from '../utils/authenticatedFetch'
import { dual } from '../casing'
import { DEMO_MODE } from '@/mocks/demo-mode-flag'

type DealInfo = components['schemas']['DealInfo']

/** Normalize a backend deal (snake_case wire) to the camelCase shape the
 *  pipeline/analytics/360° views read (X-3 casing drift). */
export function backendDealToUI(d: unknown): DealInfo {
  const raw = (d ?? {}) as Record<string, unknown>
  return {
    ...raw,
    stageId: dual<string>(d, 'stageId') ?? '',
    stageName: dual<string>(d, 'stageName') ?? '',
    contactName: dual<string>(d, 'contactName') ?? '',
    companyName: dual<string>(d, 'companyName') ?? '',
    ownerName: dual<string>(d, 'ownerName') ?? '',
    expectedCloseDate: dual<string>(d, 'expectedCloseDate') ?? '',
    createdAt: dual<string>(d, 'createdAt') ?? '',
  } as DealInfo
}

/** Real backend wants `custom_fields` as an array; the deal forms send a
 *  `_`-prefixed extras object. Drop it in real mode (backend gap), keep it in
 *  DEMO_MODE so the mock retains the extras. Core fields are already snake_case. */
function toDealPayload(body: Record<string, unknown>): Record<string, unknown> {
  if (DEMO_MODE) return body
  const next = { ...body }
  next.custom_fields = []
  return next
}

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
    queryKey: ['deals', params?.page, params?.page_size, params?.search, params?.stage_id, params?.contact_id, params?.company_id, params?.owner_id, params?.tag_ids, params?.sort_by, params?.sort_desc],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/deals', {
        params: { query: params },
      })
      if (error) throw error
      return { ...data, deals: (data?.deals ?? []).map(backendDealToUI) }
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
      return data?.deal ? { ...data, deal: backendDealToUI(data.deal) } : data
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
        body: toDealPayload(body) as never,
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
      // Update is PUT server-side (spec mislabels it PATCH, X-3).
      return authenticatedRequest({
        method: 'PUT',
        path: `/api/v1/deals/${id}`,
        body: toDealPayload(body),
      })
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
