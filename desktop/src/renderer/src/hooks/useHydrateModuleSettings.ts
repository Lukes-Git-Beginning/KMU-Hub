import { useEffect } from 'react'
import { useCrmPrefsStore } from '@/stores/crmPrefs'
import { useFinancePrefsStore } from '@/stores/financePrefs'
import { useTeamPrefsStore } from '@/stores/teamPrefs'
import { useDashboardPrefsStore } from '@/stores/dashboardPrefs'
import { useHelpdeskPrefsStore } from '@/stores/helpdeskPrefs'
import { useZeiterfassungPrefsStore } from '@/stores/zeiterfassungPrefs'
import { useWikiPrefsStore } from '@/stores/wikiPrefs'
import { useDokumentePrefsStore } from '@/stores/dokumentePrefs'
import { useWorkPrefsStore } from '@/stores/workPrefs'
import { useVertraegePrefsStore } from '@/stores/vertraegePrefs'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import { useWikiSettingsStore } from '@/stores/wikiSettings'
import { useDashboardSettingsStore } from '@/stores/dashboardSettings'
import { useZeiterfassungSettingsStore } from '@/stores/zeiterfassungSettings'

/**
 * Central settings hydrator (X-4 settings rollout).
 *
 * Each backend-persisted preference store keeps `persist` (localStorage) as its
 * optimistic cache and exposes `initFromServer()` (guarded by `serverInitialized`,
 * so repeated calls are no-ops). This hook fires them all once, fire-and-forget,
 * when the app shell mounts — i.e. once the user is authenticated. Non-blocking:
 * stores render their localStorage values immediately and patch in the server
 * value when it arrives.
 *
 * Both user-scope (per-user prefs) and tenant-scope (module-lead settings) stores
 * are hydrated here: the tenant GET is a read any authenticated user may do (they
 * need the tenant defaults); only the write-through is role-gated server-side.
 *
 * To add a new backend-persisted preference store: convert it to the reference
 * pattern (stores/crmPrefs.ts for user scope, stores/financeTenant.ts for tenant
 * scope) and register its initFromServer below.
 */
const HYDRATORS: Array<() => Promise<void>> = [
  // user scope (per-user preferences)
  () => useCrmPrefsStore.getState().initFromServer(),
  () => useFinancePrefsStore.getState().initFromServer(),
  () => useTeamPrefsStore.getState().initFromServer(),
  () => useDashboardPrefsStore.getState().initFromServer(),
  () => useHelpdeskPrefsStore.getState().initFromServer(),
  () => useZeiterfassungPrefsStore.getState().initFromServer(),
  () => useWikiPrefsStore.getState().initFromServer(),
  () => useDokumentePrefsStore.getState().initFromServer(),
  () => useWorkPrefsStore.getState().initFromServer(),
  () => useVertraegePrefsStore.getState().initFromServer(),
  // tenant scope (module-lead / admin settings — read for all, write role-gated)
  () => useFinanceTenantStore.getState().initFromServer(),
  () => useWikiSettingsStore.getState().initFromServer(),
  () => useDashboardSettingsStore.getState().initFromServer(),
  () => useZeiterfassungSettingsStore.getState().initFromServer(),
]

export function useHydrateModuleSettings(): void {
  useEffect(() => {
    for (const hydrate of HYDRATORS) void hydrate()
  }, [])
}
