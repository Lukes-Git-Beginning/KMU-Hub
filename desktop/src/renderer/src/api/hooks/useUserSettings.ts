/**
 * TanStack Query hooks for user-scope settings persistence.
 *
 * Wraps GET /api/v1/settings/{module_id}/user and PUT /api/v1/settings/{module_id}/user.
 *
 * Wire shape (response.JSON with proto-generated struct, structpb.Value marshals as
 * plain JSON values via json.Marshaler):
 *   GET → { entries: [{ key: string; value: unknown }] }
 *   PUT body → { settings: { [key: string]: unknown } }
 *   PUT response → { entries: [{ key: string; value: unknown }] }
 *
 * No Timestamp fields, no enum ints — no normalizer needed.
 *
 * Supported module IDs (free-form text, no server enum; matches settings.ts sections):
 *   "appearance" | "language" | "notifications" | "calendar"
 *
 * Note: mail, finance, teamAdmin, security are not wired here — their backend
 * persistence requires additional endpoints or admin-only writes (tenant-scope).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Wire types
// ---------------------------------------------------------------------------

interface SettingEntry {
  key: string
  value: unknown
}

interface SettingsResponse {
  entries: SettingEntry[]
}

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const userSettingsKeys = {
  all: ['user-settings'] as const,
  module: (moduleId: string) => [...userSettingsKeys.all, moduleId] as const,
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Converts the flat entries array from the backend into a key→value map.
 * Empty or missing entries return an empty object (caller falls back to store defaults).
 */
export function entriesToMap(entries: SettingEntry[] | undefined): Record<string, unknown> {
  if (!entries || entries.length === 0) return {}
  return Object.fromEntries(entries.map((e) => [e.key, e.value]))
}

/**
 * Converts a key→value map into the `{ settings: {...} }` request body for PUT.
 */
function mapToBody(settings: Record<string, unknown>): { settings: Record<string, unknown> } {
  return { settings }
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

/**
 * Fetch the authenticated user's settings for one module.
 * Returns a flat key→value map (empty object when no settings are stored yet).
 *
 * @param moduleId  e.g. "appearance", "language", "notifications", "calendar"
 * @param enabled   set false to skip the query (e.g. during initial auth setup)
 */
export function useGetUserSettings(moduleId: string, enabled = true) {
  return useQuery({
    queryKey: userSettingsKeys.module(moduleId),
    queryFn: async (): Promise<Record<string, unknown>> => {
      const data = await authenticatedRequest<SettingsResponse>({
        method: 'GET',
        path: `/api/v1/settings/${moduleId}/user`,
      })
      return entriesToMap(data?.entries)
    },
    enabled: enabled && !!moduleId,
    // Settings rarely change from external sources; generous stale time.
    staleTime: 5 * 60_000,
  })
}

/**
 * Save the authenticated user's settings for one module.
 * Patch semantics: only supplied keys are written; unset keys are untouched.
 *
 * Invalidates the module query on success so the next read reflects the update.
 */
export function usePutUserSettings(moduleId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (settings: Record<string, unknown>): Promise<Record<string, unknown>> => {
      const data = await authenticatedRequest<SettingsResponse>({
        method: 'PUT',
        path: `/api/v1/settings/${moduleId}/user`,
        body: mapToBody(settings),
      })
      return entriesToMap(data?.entries)
    },
    onSuccess: (updatedMap) => {
      // Directly write the returned map into the cache so queries reflect the
      // update immediately without a network round-trip.
      queryClient.setQueryData(userSettingsKeys.module(moduleId), updatedMap)
    },
  })
}
