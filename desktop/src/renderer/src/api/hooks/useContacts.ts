/**
 * TanStack Query hooks for Contact CRUD operations.
 *
 * All hooks use the type-safe apiClient to call backend endpoints.
 * Query keys follow the pattern ['contacts', ...params] for consistent
 * cache invalidation across related queries.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

export interface ContactListParams {
  page?: number
  page_size?: number
  search?: string
  company_id?: string
  tag_ids?: string[]
}

export function useContacts(params?: ContactListParams) {
  return useQuery({
    queryKey: ['contacts', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/contacts', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useContact(id: string) {
  return useQuery({
    queryKey: ['contacts', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/contacts/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

export function useCreateContact() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      first_name: string
      last_name: string
      email?: string
      phone?: string
      title?: string
      company_id?: string
      notes?: string
      tag_ids?: string[]
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/contacts', {
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
    },
  })
}

export function useUpdateContact() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      first_name?: string
      last_name?: string
      email?: string
      phone?: string
      title?: string
      company_id?: string
      notes?: string
      custom_fields?: Record<string, unknown>
    }) => {
      const { data, error } = await apiClient.PATCH('/api/v1/contacts/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
      queryClient.invalidateQueries({ queryKey: ['contacts', variables.id] })
    },
  })
}

export function useDeleteContact() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE('/api/v1/contacts/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
    },
  })
}
