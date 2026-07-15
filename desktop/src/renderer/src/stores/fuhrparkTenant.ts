import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Fuhrpark-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Fuhrpark-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche Prefs
 * liegen in stores/fuhrparkPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (Vimcar/Fleetster/Avrios): Erinnerungs-Vorlauf
 * für HU/Versicherung, Firmen-Währung, Standard-Kraftstoff und die
 * Privatfahrten-Policy sind dort firmenweite Einstellungen.
 */
const MODULE_ID = 'fuhrpark'

export const FUHRPARK_CURRENCY_OPTIONS = ['EUR', 'CHF']
export const FUHRPARK_FUEL_OPTIONS = ['diesel', 'petrol', 'electric', 'hybrid'] as const
export type FuhrparkFuelType = (typeof FUHRPARK_FUEL_OPTIONS)[number]

interface FuhrparkTenantState {
  /** Days before TÜV/insurance expiry that vehicles show as "due soon". */
  reminderLeadDays: number
  /** Currency used for all cost displays (TCO, tables). */
  currency: string
  /** Default fuel type for new vehicles and fuel logs. */
  defaultFuelType: FuhrparkFuelType
  /** Whether private trips may be logged in the Fahrtenbuch. */
  privateTripsEnabled: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setReminderLeadDays: (v: number) => void
  setCurrency: (v: string) => void
  setDefaultFuelType: (v: FuhrparkFuelType) => void
  setPrivateTripsEnabled: (v: boolean) => void
  /** Hydrate from GET /settings/fuhrpark/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: FuhrparkTenantState): Record<string, unknown> {
  return {
    reminderLeadDays: s.reminderLeadDays,
    currency: s.currency,
    defaultFuelType: s.defaultFuelType,
    privateTripsEnabled: s.privateTripsEnabled,
  }
}

function clampLeadDays(v: number): number {
  return Math.min(180, Math.max(7, Math.round(v)))
}

export const useFuhrparkTenantStore = create<FuhrparkTenantState>()(
  persist(
    (set, get) => ({
      reminderLeadDays: 30,
      currency: 'EUR',
      defaultFuelType: 'diesel',
      privateTripsEnabled: true,
      serverInitialized: false,
      setReminderLeadDays: (reminderLeadDays) => {
        set({ reminderLeadDays: clampLeadDays(reminderLeadDays) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setCurrency: (currency) => {
        set({ currency: FUHRPARK_CURRENCY_OPTIONS.includes(currency) ? currency : 'EUR' })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultFuelType: (defaultFuelType) => {
        set({
          defaultFuelType: FUHRPARK_FUEL_OPTIONS.includes(defaultFuelType)
            ? defaultFuelType
            : 'diesel',
        })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setPrivateTripsEnabled: (privateTripsEnabled) => {
        set({ privateTripsEnabled })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          reminderLeadDays:
            typeof map.reminderLeadDays === 'number' && map.reminderLeadDays > 0
              ? clampLeadDays(map.reminderLeadDays)
              : s.reminderLeadDays,
          currency:
            typeof map.currency === 'string' && FUHRPARK_CURRENCY_OPTIONS.includes(map.currency)
              ? map.currency
              : s.currency,
          defaultFuelType: FUHRPARK_FUEL_OPTIONS.includes(map.defaultFuelType as FuhrparkFuelType)
            ? (map.defaultFuelType as FuhrparkFuelType)
            : s.defaultFuelType,
          privateTripsEnabled:
            typeof map.privateTripsEnabled === 'boolean'
              ? map.privateTripsEnabled
              : s.privateTripsEnabled,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-fuhrpark-tenant' },
  ),
)
