/**
 * React Query hook for the feature-flag registry.
 *
 * Fetches GET /api/v1/feature-flags once and caches for 5 minutes.
 * Provides an isEnabled(key) helper so callers never have to inspect the raw map.
 */
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'

/** Shape of the backend response envelope. */
interface FeatureFlagsResponse {
  flags: Record<string, boolean>
  version: string
}

/** Return value of useFeatureFlags. */
export interface UseFeatureFlagsResult {
  flags: Record<string, boolean> | undefined
  isLoading: boolean
  error: Error | null
  /** Returns true if the flag is enabled. Unknown flags return false. */
  isEnabled: (key: string) => boolean
}

export const featureFlagKeys = {
  all: ['feature-flags'] as const,
}

/**
 * Returns all resolved feature flags from the backend.
 * Cached for 5 minutes; refetched on window focus.
 */
export function useFeatureFlags(): UseFeatureFlagsResult {
  const { data, isLoading, error } = useQuery<FeatureFlagsResponse, Error>({
    queryKey: featureFlagKeys.all,
    queryFn: async () => {
      const result = await apiClient.GET('/api/v1/feature-flags' as never)
      if ((result as { error?: unknown }).error) {
        throw new Error('Failed to load feature flags')
      }
      return (result as { data: FeatureFlagsResponse }).data
    },
    staleTime: 5 * 60 * 1_000, // 5 minutes
    refetchOnWindowFocus: true,
  })

  const isEnabled = (key: string): boolean => {
    if (!data?.flags) return false
    return data.flags[key] ?? false
  }

  return {
    flags: data?.flags,
    isLoading,
    error,
    isEnabled,
  }
}
