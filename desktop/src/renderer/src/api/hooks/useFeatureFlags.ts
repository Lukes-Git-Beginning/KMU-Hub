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
 * Demo mode (electron-vite --mode demo) has no backend.
 * Without an override, every flag would resolve to false and feature-gated
 * modules would be hidden from designers. Treat all flags as enabled.
 */
const IS_DEMO = import.meta.env.MODE === 'demo'

/**
 * Returns all resolved feature flags from the backend.
 * Cached for 5 minutes; refetched on window focus.
 * In demo mode, returns all flags as enabled without a network call.
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
    enabled: !IS_DEMO,
  })

  const isEnabled = (key: string): boolean => {
    if (IS_DEMO) return true
    if (!data?.flags) return false
    return data.flags[key] ?? false
  }

  return {
    flags: data?.flags,
    isLoading: IS_DEMO ? false : isLoading,
    error: IS_DEMO ? null : error,
    isEnabled,
  }
}
