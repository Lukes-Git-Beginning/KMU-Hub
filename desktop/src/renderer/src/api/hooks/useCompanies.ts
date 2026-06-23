/**
 * TanStack Query hooks for Company CRUD operations.
 *
 * Includes useCompanyContacts for fetching contacts linked to a company.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { authenticatedRequest } from '../utils/authenticatedFetch'
import { DEMO_MODE } from '@/mocks/demo-mode-flag'

/** Body shape the company forms produce (camelCase `website`, `custom_fields`
 *  as a `_`-prefixed extras object). */
interface CompanyWriteBody {
  name?: string
  website?: string
  industry?: string
  address?: string
  notes?: string
  tag_ids?: string[]
  custom_fields?: Record<string, unknown>
}

/** Map a form body to the real-backend contract: `domain` (not `website`) and
 *  `custom_fields` as an array. UI-only extras (_phone/_email/_size/_tags) are
 *  dropped (backend gap, see kontakte-mock-exit-DONE.md). In DEMO_MODE the
 *  original body is kept so the mock retains the full field set. */
function toCompanyPayload(body: CompanyWriteBody): Record<string, unknown> {
  if (DEMO_MODE) return body as Record<string, unknown>
  return {
    name: body.name,
    domain: body.website || undefined,
    industry: body.industry || undefined,
    address: body.address || undefined,
    notes: body.notes || undefined,
    tag_ids: body.tag_ids,
    custom_fields: [],
  }
}

export interface CompanyListParams {
  page?: number
  page_size?: number
  search?: string
  tag_ids?: string[]
}

export function useCompanies(params?: CompanyListParams) {
  return useQuery({
    queryKey: ['companies', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/companies', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useCompany(id: string) {
  return useQuery({
    queryKey: ['companies', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/companies/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

export function useCompanyContacts(companyId: string) {
  return useQuery({
    queryKey: ['companies', companyId, 'contacts'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/companies/{id}/contacts',
        {
          params: { path: { id: companyId } },
        }
      )
      if (error) throw error
      return data
    },
    enabled: !!companyId,
  })
}

export function useCreateCompany() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: CompanyWriteBody & { name: string }) => {
      const { data, error } = await apiClient.POST('/api/v1/companies', {
        body: toCompanyPayload(body) as never,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
    },
  })
}

export function useUpdateCompany() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, ...body }: CompanyWriteBody & { id: string }) => {
      // Update is PUT server-side (the OpenAPI spec mislabels it PATCH, X-3) →
      // method-explicit authenticatedRequest. Payload mapped to the real contract.
      return authenticatedRequest({
        method: 'PUT',
        path: `/api/v1/companies/${id}`,
        body: toCompanyPayload(body),
      })
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      queryClient.invalidateQueries({ queryKey: ['companies', variables.id] })
    },
  })
}

export function useDeleteCompany() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/companies/{id}',
        {
          params: { path: { id } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
    },
  })
}
