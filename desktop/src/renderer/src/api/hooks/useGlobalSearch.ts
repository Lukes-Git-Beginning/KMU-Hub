/**
 * TanStack Query hook for the global search API.
 *
 * Calls GET /api/v1/search/global?q=...&limit=... which fans out to
 * CRM, Documents, and Email backends in parallel on the gateway.
 */
import { useQuery } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Types matching gateway globalSearchResult struct
// ---------------------------------------------------------------------------

export interface GlobalSearchResultItem {
  id: string
  title: string
  description?: string
  url?: string
  // CRM-specific fields
  email?: string
  company?: string
  // File-specific fields
  mime_type?: string
  content_text?: string
  file_name?: string
  // Email-specific fields
  subject?: string
  // Task-specific fields
  project_name?: string
  status?: string
  // Message-specific fields
  channel?: string
}

export interface GlobalSearchModule {
  module: string
  results: GlobalSearchResultItem[]
  total: number
  error?: string
}

export interface GlobalSearchResponse {
  query: string
  modules: GlobalSearchModule[]
}

// ---------------------------------------------------------------------------
// Hook options
// ---------------------------------------------------------------------------

interface UseGlobalSearchOptions {
  query: string
  modules?: string[]
  limit?: number
  enabled?: boolean
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useGlobalSearch({
  query,
  modules,
  limit = 5,
  enabled = true,
}: UseGlobalSearchOptions) {
  return useQuery<GlobalSearchResponse>({
    queryKey: ['global-search', query, modules, limit],
    queryFn: async () => {
      const params: Record<string, string> = { q: query, limit: String(limit) }
      if (modules?.length) params['modules'] = modules.join(',')

      return authenticatedRequest<GlobalSearchResponse>({
        method: 'GET',
        path: '/api/v1/search/global',
        params,
      })
    },
    enabled: enabled && query.length >= 2,
    staleTime: 30_000,
    gcTime: 60_000,
  })
}
