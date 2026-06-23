/**
 * TanStack Query hooks for Contact CRUD operations.
 *
 * All hooks use the type-safe apiClient to call backend endpoints.
 * Query keys follow the pattern ['contacts', ...params] for consistent
 * cache invalidation across related queries.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { authenticatedRequest } from '../utils/authenticatedFetch'

export interface ContactListParams {
  page?: number
  page_size?: number
  search?: string
  company_id?: string
  tag_ids?: string[]
}

export function useContacts(params?: ContactListParams) {
  return useQuery({
    queryKey: ['contacts', params?.page, params?.page_size, params?.search, params?.company_id, params?.tag_ids],
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

export interface ContactFile {
  id: string
  contact_id: string
  filename: string
  mime_type: string
  storage_key: string
  created_at: string
}

/** Files/links attached to a contact (e.g. report references from berichte). */
export function useContactFiles(id: string) {
  return useQuery({
    queryKey: ['contacts', id, 'files'],
    queryFn: () =>
      authenticatedRequest<{ files: ContactFile[] }>({
        method: 'GET',
        path: `/api/v1/contacts/${id}/files`,
      }),
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
      // Update ist serverseitig PUT (die OpenAPI-Spec nennt fälschlich PATCH,
      // X-3) → method-expliziter authenticatedRequest statt openapi-fetch, das
      // nur die (falsche) PATCH-Methode typt.
      return authenticatedRequest({
        method: 'PUT',
        path: `/api/v1/contacts/${id}`,
        body,
      })
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
