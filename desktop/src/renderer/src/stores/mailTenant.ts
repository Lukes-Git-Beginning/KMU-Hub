/**
 * Mail — tenant-weite Compliance-Vorgaben ("Für alle"-Scope, siehe
 * ModuleSettingsShell): Aufbewahrungs-Badges + Policy für externe Bilder.
 * Editierbar nur durch Modul-Leiter/Admin (der PUT wird serverseitig für andere
 * Rollen verworfen; das Panel gatet die UI). Persönliche Prefs liegen in
 * stores/mailPrefs.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/mail/tenant):
 * initFromServer() hydratet einmal pro Session (via useHydrateModuleSettings);
 * Setter schreiben durch. localStorage bleibt Optimistic-Cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

const MODULE_ID = 'mail'

interface MailTenantState {
  /** Whether retention badges are shown on messages. */
  showRetentionBadges: boolean
  /** Whether external images load automatically (privacy). */
  loadExternalImages: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setShowRetentionBadges: (v: boolean) => void
  setLoadExternalImages: (v: boolean) => void
  initFromServer: () => Promise<void>
}

function tenantPayload(s: MailTenantState): Record<string, unknown> {
  return {
    showRetentionBadges: s.showRetentionBadges,
    loadExternalImages: s.loadExternalImages,
  }
}

export const useMailTenantStore = create<MailTenantState>()(
  persist(
    (set, get) => ({
      showRetentionBadges: true,
      loadExternalImages: false,
      serverInitialized: false,
      setShowRetentionBadges: (showRetentionBadges) => {
        set({ showRetentionBadges })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setLoadExternalImages: (loadExternalImages) => {
        set({ loadExternalImages })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          showRetentionBadges:
            typeof map.showRetentionBadges === 'boolean'
              ? map.showRetentionBadges
              : s.showRetentionBadges,
          loadExternalImages:
            typeof map.loadExternalImages === 'boolean'
              ? map.loadExternalImages
              : s.loadExternalImages,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-mail-tenant' },
  ),
)
