/**
 * React Query hook for the feature-flag registry.
 *
 * Fetches GET /api/v1/feature-flags once and caches for 5 minutes.
 * Provides an isEnabled(key) helper so callers never have to inspect the raw map.
 *
 * Fail-closed policy (app-wide default): unknown flags, backend errors, and
 * loading states all resolve to false. This prevents unlicensed modules from
 * being exposed if the backend is temporarily unreachable.
 *
 * Exception — dashboard widget gating: WidgetContainer uses a local fail-open
 * fallback so the dashboard is usable during QA / dev / offline scenarios.
 * That policy lives in WidgetContainer, NOT here.
 *
 * ⚠ Darien review question (User decision 2026-06-10 — Luke):
 * dashboard-lokales fail-open ist beschlossen. Offene Frage: reicht das für den
 * Pilot-Betrieb, oder soll auch dort fail-closed mit Cache (z.B. 24h localStorage)
 * gelten? Feedback in .planning/reviews/dashboard.md Phase 5 eintragen.
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
  /**
   * Returns true if the flag is enabled.
   * - Unknown flags: false
   * - Backend unreachable / loading / no data: false (fail-closed, app-wide default)
   * - Demo mode: true (no backend available)
   * - Dashboard widget gating applies its own local fail-open — see WidgetContainer.
   */
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
    // Do not retry immediately on failure -- the first miss triggers fail-open
    retry: 1,
  })

  const isEnabled = (key: string): boolean => {
    // QA override (DEV only, tree-shaken in production builds):
    // window.__cosmi_qa_flags__ = { 'modules.crm': false, ... } lets Playwright
    // tests inject specific flag states without a real backend.
    if (import.meta.env.DEV) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const qaOverride = (window as any).__cosmi_qa_flags__
      if (qaOverride && typeof qaOverride === 'object' && key in qaOverride) {
        return Boolean(qaOverride[key])
      }
    }
    // Demo mode → all flags enabled (no backend available)
    if (IS_DEMO) return true
    // Fail-closed: error / loading / empty response → false.
    // Dashboard widget gating uses its own local fail-open fallback (WidgetContainer).
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
