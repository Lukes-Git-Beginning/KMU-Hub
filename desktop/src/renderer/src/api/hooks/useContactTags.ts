/**
 * TanStack Query hooks for Tag operations on contacts.
 *
 * - useTags()              — GET  /api/v1/tags  (filtered to entity_type=contact)
 * - useAddContactTags()   — POST   /api/v1/contacts/{id}/tags
 * - useRemoveContactTags()— DELETE /api/v1/contacts/{id}/tags
 *
 * Request / response shapes are derived from the OpenAPI types:
 *   TagIdsRequest  → { tag_ids: string[] }
 *   ListTagsResponse → { tags?: TagInfo[] }
 *   ContactResponse  → { contact?: ContactInfo }
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// Shared Tag shape (mirrors OpenAPI TagInfo)
// ---------------------------------------------------------------------------

export interface TagInfo {
  id: string
  name: string
  color: string
  entityType?: 'contact' | 'company' | 'deal' | 'activity'
}

// ---------------------------------------------------------------------------
// useTags — list available tags (contact scope)
// ---------------------------------------------------------------------------

export function useTags() {
  return useQuery({
    queryKey: ['tags', 'contact'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/tags', {
        params: { query: { entity_type: 'contact' } },
      })
      if (error) throw error
      // Normalize: filter tags with both id and name present
      const tags: TagInfo[] = (data?.tags ?? [])
        .filter((t): t is Required<Pick<typeof t, 'id' | 'name'>> & typeof t => !!t.id && !!t.name)
        .map((t) => ({
          id: t.id!,
          name: t.name!,
          color: t.color ?? '#6B7280',
          entityType: t.entityType,
        }))
      return tags
    },
    staleTime: 60_000,
  })
}

// ---------------------------------------------------------------------------
// useAddContactTags — POST /api/v1/contacts/{id}/tags
// ---------------------------------------------------------------------------

export function useAddContactTags() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ contactId, tagIds }: { contactId: string; tagIds: string[] }) => {
      const { data, error } = await apiClient.POST('/api/v1/contacts/{id}/tags', {
        params: { path: { id: contactId } },
        body: { tag_ids: tagIds },
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
      queryClient.invalidateQueries({ queryKey: ['contacts', variables.contactId] })
    },
  })
}

// ---------------------------------------------------------------------------
// Tag-definition CRUD (tenant-wide tag management).
//
// These endpoints are not yet in the OpenAPI spec (see backend-gaps), so they
// use raw fetch — MSW intercepts them in demo mode all the same.
// ---------------------------------------------------------------------------

export function useCreateTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { name: string; color: string }) => {
      const res = await fetch(`${API_BASE_URL}/api/v1/tags`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...body, entity_type: 'contact' }),
      })
      if (!res.ok) throw new Error('Failed to create tag')
      return res.json()
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tags', 'contact'] }),
  })
}

export function useUpdateTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, ...body }: { id: string; name?: string; color?: string }) => {
      const res = await fetch(`${API_BASE_URL}/api/v1/tags/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error('Failed to update tag')
      return res.json()
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tags', 'contact'] }),
  })
}

export function useDeleteTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`${API_BASE_URL}/api/v1/tags/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Failed to delete tag')
      return res.json()
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tags', 'contact'] }),
  })
}

// ---------------------------------------------------------------------------
// useRemoveContactTags — DELETE /api/v1/contacts/{id}/tags
// ---------------------------------------------------------------------------

export function useRemoveContactTags() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ contactId, tagIds }: { contactId: string; tagIds: string[] }) => {
      const { data, error } = await apiClient.DELETE('/api/v1/contacts/{id}/tags', {
        params: { path: { id: contactId } },
        body: { tag_ids: tagIds },
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
      queryClient.invalidateQueries({ queryKey: ['contacts', variables.contactId] })
    },
  })
}
