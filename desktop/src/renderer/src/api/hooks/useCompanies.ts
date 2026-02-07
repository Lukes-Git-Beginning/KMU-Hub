/**
 * TanStack Query hooks for Company CRUD operations.
 *
 * Includes useCompanyContacts for fetching contacts linked to a company.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

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
    mutationFn: async (body: {
      name: string
      website?: string
      industry?: string
      address?: string
      notes?: string
      tag_ids?: string[]
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/companies', {
        body,
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
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      name?: string
      website?: string
      industry?: string
      address?: string
      notes?: string
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.PATCH('/api/v1/companies/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
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
