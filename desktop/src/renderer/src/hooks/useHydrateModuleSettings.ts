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
import { useDialerPrefsStore } from '@/stores/dialerPrefs'
import { useVideoPrefsStore } from '@/stores/videoPrefs'
import { useAutomatisierungPrefsStore } from '@/stores/automatisierungPrefs'
import { useMailPrefsStore } from '@/stores/mailPrefs'
import { useFormularePrefsStore } from '@/stores/formularePrefs'
import { useBerichtePrefsStore } from '@/stores/berichtePrefs'
import { useInventarPrefsStore } from '@/stores/inventarPrefs'
import { useVermietungViewPrefsStore } from '@/stores/vermietungViewPrefs'
import { useRapportePrefsStore } from '@/stores/rapportePrefs'
import { useSchichtenPrefsStore } from '@/stores/schichtenPrefs'
import { useFuhrparkPrefsStore } from '@/stores/fuhrparkPrefs'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import { useWikiSettingsStore } from '@/stores/wikiSettings'
import { useDashboardSettingsStore } from '@/stores/dashboardSettings'
import { useZeiterfassungSettingsStore } from '@/stores/zeiterfassungSettings'
import { useDialerTenantStore } from '@/stores/dialerTenant'
import { useVideoTenantStore } from '@/stores/videoTenant'
import { useAutomatisierungTenantStore } from '@/stores/automatisierungTenant'
import { useMailTenantStore } from '@/stores/mailTenant'
import { useFormulareTenantStore } from '@/stores/formulareTenant'
import { useBerichteTenantStore } from '@/stores/berichteTenant'
import { useInventarTenantStore } from '@/stores/inventarTenant'
import { useVermietungTenantStore } from '@/stores/vermietungTenant'
import { useRapporteTenantStore } from '@/stores/rapporteTenant'
import { useSchichtenTenantStore } from '@/stores/schichtenTenant'
import { useFuhrparkTenantStore } from '@/stores/fuhrparkTenant'

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
  () => useDialerPrefsStore.getState().initFromServer(),
  () => useVideoPrefsStore.getState().initFromServer(),
  () => useAutomatisierungPrefsStore.getState().initFromServer(),
  () => useMailPrefsStore.getState().initFromServer(),
  () => useFormularePrefsStore.getState().initFromServer(),
  () => useBerichtePrefsStore.getState().initFromServer(),
  () => useInventarPrefsStore.getState().initFromServer(),
  () => useVermietungViewPrefsStore.getState().initFromServer(),
  () => useRapportePrefsStore.getState().initFromServer(),
  () => useSchichtenPrefsStore.getState().initFromServer(),
  () => useFuhrparkPrefsStore.getState().initFromServer(),
  // tenant scope (module-lead / admin settings — read for all, write role-gated)
  () => useFinanceTenantStore.getState().initFromServer(),
  () => useWikiSettingsStore.getState().initFromServer(),
  () => useDashboardSettingsStore.getState().initFromServer(),
  () => useZeiterfassungSettingsStore.getState().initFromServer(),
  () => useDialerTenantStore.getState().initFromServer(),
  () => useVideoTenantStore.getState().initFromServer(),
  () => useAutomatisierungTenantStore.getState().initFromServer(),
  () => useMailTenantStore.getState().initFromServer(),
  () => useFormulareTenantStore.getState().initFromServer(),
  () => useBerichteTenantStore.getState().initFromServer(),
  () => useInventarTenantStore.getState().initFromServer(),
  () => useVermietungTenantStore.getState().initFromServer(),
  () => useRapporteTenantStore.getState().initFromServer(),
  () => useSchichtenTenantStore.getState().initFromServer(),
  () => useFuhrparkTenantStore.getState().initFromServer(),
]

export function useHydrateModuleSettings(): void {
  useEffect(() => {
    for (const hydrate of HYDRATORS) void hydrate()
  }, [])
}
