/**
 * TanStack Query hook for the global search API.
 *
 * Calls GET /api/v1/search/global?q=...&limit=... which fans out to
 * CRM, Documents, and Email backends in parallel on the gateway.
 */
import { useQuery } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'

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
// Auth token helper
// ---------------------------------------------------------------------------

async function getToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
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
      const params = new URLSearchParams({ q: query, limit: String(limit) })
      if (modules?.length) params.set('modules', modules.join(','))

      const token = await getToken()
      const headers: Record<string, string> = {}
      if (token) headers['Authorization'] = `Bearer ${token}`

      const response = await fetch(
        `${API_BASE_URL}/api/v1/search/global?${params}`,
        { headers },
      )
      if (!response.ok) throw new Error('Search failed')
      return response.json()
    },
    enabled: enabled && query.length >= 2,
    staleTime: 30_000,
    gcTime: 60_000,
  })
}
