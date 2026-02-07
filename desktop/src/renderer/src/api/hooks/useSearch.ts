/**
 * TanStack Query hook for global CRM search.
 *
 * Searches across contacts, companies, deals, and activities.
 * Only fires when query is at least 2 characters to avoid
 * excessive API calls.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '../client'

export interface CRMSearchParams {
  types?: string
  limit?: number
}

export function useCRMSearch(query: string, params?: CRMSearchParams) {
  return useQuery({
    queryKey: ['crm-search', query, params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/search', {
        params: {
          query: {
            q: query,
            types: params?.types,
            limit: params?.limit,
          },
        },
      })
      if (error) throw error
      return data
    },
    enabled: query.length >= 2,
  })
}
