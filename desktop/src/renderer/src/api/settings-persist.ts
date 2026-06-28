/**
 * Shared persistence helpers for module settings (settings foundation, Migr. 138).
 *
 * Backend wire shape (response.JSON, structpb values):
 *   GET  /api/v1/settings/{moduleId}/{scope} → { entries: [{ key, value }] }
 *   PUT  /api/v1/settings/{moduleId}/{scope}   body { settings: { [key]: value } }  (patch semantics)
 *
 * Scope:
 *   - "user"   → per-user preferences (any authenticated user)
 *   - "tenant" → tenant-wide settings (module lead / admin; the panel gates the UI)
 *
 * Used by the X-4 store rollout: each persisted Zustand store keeps `persist`
 * (localStorage) as the optimistic cache and additionally hydrates once per
 * session via `loadModuleSettings` and writes through each setter via
 * `saveModuleSettings`. Reference store: stores/dokumenteSettings.ts.
 *
 * `authenticatedRequest` injects the Authorization + Idempotency-Key headers, so
 * callers don't manage those. Both helpers swallow errors (offline / no backend):
 * the local store value already applies and re-syncs on the next session.
 */
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

export type SettingsScope = 'user' | 'tenant'

interface EntriesResponse {
  entries?: Array<{ key: string; value: unknown }>
}

/**
 * Hydrate one module's settings into a flat key→value map.
 * Returns an empty object on miss/offline so the caller falls back to its defaults.
 */
export async function loadModuleSettings(
  moduleId: string,
  scope: SettingsScope,
): Promise<Record<string, unknown>> {
  try {
    const resp = await authenticatedRequest<EntriesResponse>({
      method: 'GET',
      path: `/api/v1/settings/${moduleId}/${scope}`,
    })
    return Object.fromEntries((resp.entries ?? []).map((e) => [e.key, e.value]))
  } catch {
    return {}
  }
}

/**
 * Write through one module's settings (patch semantics — only the supplied keys
 * are sent). Failures are non-critical; the store already updated locally.
 */
export async function saveModuleSettings(
  moduleId: string,
  scope: SettingsScope,
  settings: Record<string, unknown>,
): Promise<void> {
  try {
    await authenticatedRequest({
      method: 'PUT',
      path: `/api/v1/settings/${moduleId}/${scope}`,
      body: { settings },
    })
  } catch {
    // Silent — store already updated; re-syncs on next session.
  }
}
