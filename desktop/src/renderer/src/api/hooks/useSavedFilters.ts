/**
 * TanStack Query hooks for Saved Filters.
 *
 * Saved filters allow users to persist contact/company/deal filter presets
 * and reuse them — e.g. for campaign contact import.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '../client'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const savedFilterKeys = {
  all: ['saved-filters'] as const,
  list: (entityType?: string) => ['saved-filters', 'list', entityType] as const,
  detail: (id: string) => ['saved-filters', id] as const,
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

export function useSavedFilters(
  entityType?: 'contact' | 'company' | 'deal' | 'activity',
) {
  return useQuery({
    queryKey: savedFilterKeys.list(entityType),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/saved-filters', {
        params: { query: { entity_type: entityType } },
      })
      if (error) throw error
      return data?.filters ?? []
    },
    staleTime: 60_000,
  })
}
