import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Inventar-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Inventar-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche
 * Ansichts-Prefs liegen in stores/inventarPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (Zoho Inventory Preferences, weclapp/myfactory
 * Lagereinstellungen): Standard-Mengeneinheit, Mindestbestand-Default,
 * Barcode-Format und Negativbestand-Sperre sind dort durchgängig Org-Settings.
 */
const MODULE_ID = 'inventar'

export const INVENTAR_UNIT_OPTIONS = ['Stück', 'kg', 'Meter', 'Liter', 'Packung', 'Rolle']

export type BarcodeFormat = 'ean13' | 'code128' | 'qr'
export const BARCODE_FORMATS: BarcodeFormat[] = ['ean13', 'code128', 'qr']

interface InventarTenantState {
  /** Default unit pre-selected when creating a new item. */
  defaultUnit: string
  /** Default minimum stock for new items. */
  defaultMinStock: number
  /** Expected barcode symbology (scanner dialog hint + label printing later). */
  barcodeFormat: BarcodeFormat
  /** Allow outgoing movements to push stock below zero. */
  allowNegativeStock: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultUnit: (v: string) => void
  setDefaultMinStock: (v: number) => void
  setBarcodeFormat: (v: BarcodeFormat) => void
  setAllowNegativeStock: (v: boolean) => void
  /** Hydrate from GET /settings/inventar/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: InventarTenantState): Record<string, unknown> {
  return {
    defaultUnit: s.defaultUnit,
    defaultMinStock: s.defaultMinStock,
    barcodeFormat: s.barcodeFormat,
    allowNegativeStock: s.allowNegativeStock,
  }
}

export const useInventarTenantStore = create<InventarTenantState>()(
  persist(
    (set, get) => ({
      defaultUnit: 'Stück',
      defaultMinStock: 5,
      barcodeFormat: 'ean13',
      allowNegativeStock: false,
      serverInitialized: false,
      setDefaultUnit: (defaultUnit) => {
        set({ defaultUnit })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultMinStock: (defaultMinStock) => {
        set({ defaultMinStock: Math.max(0, defaultMinStock) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setBarcodeFormat: (barcodeFormat) => {
        set({ barcodeFormat })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setAllowNegativeStock: (allowNegativeStock) => {
        set({ allowNegativeStock })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          defaultUnit:
            typeof map.defaultUnit === 'string' && INVENTAR_UNIT_OPTIONS.includes(map.defaultUnit)
              ? map.defaultUnit
              : s.defaultUnit,
          defaultMinStock:
            typeof map.defaultMinStock === 'number' && map.defaultMinStock >= 0
              ? map.defaultMinStock
              : s.defaultMinStock,
          barcodeFormat: BARCODE_FORMATS.includes(map.barcodeFormat as BarcodeFormat)
            ? (map.barcodeFormat as BarcodeFormat)
            : s.barcodeFormat,
          allowNegativeStock:
            typeof map.allowNegativeStock === 'boolean'
              ? map.allowNegativeStock
              : s.allowNegativeStock,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-inventar-tenant' },
  ),
)
