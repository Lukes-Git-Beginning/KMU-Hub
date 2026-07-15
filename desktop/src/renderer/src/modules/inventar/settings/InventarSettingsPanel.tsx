import { useTranslation } from 'react-i18next'
import { Warehouse, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  useInventarPrefsStore,
  type InventarTab,
  type InventarDensity,
} from '@/stores/inventarPrefs'
import {
  useInventarTenantStore,
  INVENTAR_UNIT_OPTIONS,
  BARCODE_FORMATS,
  type BarcodeFormat,
} from '@/stores/inventarTenant'

const BARCODE_LABEL_KEYS: Record<BarcodeFormat, string> = {
  ean13: 'inventar.settings.tenant.barcode.ean13',
  code128: 'inventar.settings.tenant.barcode.code128',
  qr: 'inventar.settings.tenant.barcode.qr',
}

// ─── Persönlich ──────────────────────────────────────────────────

function InventarPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useInventarPrefsStore((s) => s.defaultTab)
  const density = useInventarPrefsStore((s) => s.density)
  const showMinStockWarnings = useInventarPrefsStore((s) => s.showMinStockWarnings)
  const setDefaultTab = useInventarPrefsStore((s) => s.setDefaultTab)
  const setDensity = useInventarPrefsStore((s) => s.setDensity)
  const setShowMinStockWarnings = useInventarPrefsStore((s) => s.setShowMinStockWarnings)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('inventar.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as InventarTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="artikel">{t('inventar.tab.artikel')}</option>
            <option value="lagerorte">{t('inventar.tab.lagerorte')}</option>
            <option value="bewegungen">{t('inventar.tab.bewegungen')}</option>
            <option value="inventur">{t('inventar.tab.inventur')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('inventar.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('inventar.settings.personal.density')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={density}
            onChange={(e) => setDensity(e.target.value as InventarDensity)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="comfortable">{t('inventar.settings.personal.densityComfortable')}</option>
            <option value="compact">{t('inventar.settings.personal.densityCompact')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('inventar.settings.personal.densityHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('inventar.settings.personal.minWarnings')}</p>
          <p className="text-xs text-muted-foreground">{t('inventar.settings.personal.minWarningsHint')}</p>
        </div>
        <Switch checked={showMinStockWarnings} onCheckedChange={setShowMinStockWarnings} />
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function InventarTenantSettings() {
  const { t } = useTranslation()
  const defaultUnit = useInventarTenantStore((s) => s.defaultUnit)
  const defaultMinStock = useInventarTenantStore((s) => s.defaultMinStock)
  const barcodeFormat = useInventarTenantStore((s) => s.barcodeFormat)
  const allowNegativeStock = useInventarTenantStore((s) => s.allowNegativeStock)
  const setDefaultUnit = useInventarTenantStore((s) => s.setDefaultUnit)
  const setDefaultMinStock = useInventarTenantStore((s) => s.setDefaultMinStock)
  const setBarcodeFormat = useInventarTenantStore((s) => s.setBarcodeFormat)
  const setAllowNegativeStock = useInventarTenantStore((s) => s.setAllowNegativeStock)

  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('inventar.settings.tenant.defaultUnit')}</label>
          <div className="relative">
            <select
              value={defaultUnit}
              onChange={(e) => setDefaultUnit(e.target.value)}
              className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
            >
              {INVENTAR_UNIT_OPTIONS.map((u) => (
                <option key={u} value={u}>{u}</option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
          <p className="text-xs text-muted-foreground">{t('inventar.settings.tenant.defaultUnitHint')}</p>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('inventar.settings.tenant.defaultMinStock')}</label>
          <input
            type="number"
            min={0}
            value={defaultMinStock}
            onChange={(e) => setDefaultMinStock(Number(e.target.value))}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
          />
          <p className="text-xs text-muted-foreground">{t('inventar.settings.tenant.defaultMinStockHint')}</p>
        </div>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('inventar.settings.tenant.barcodeFormat')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={barcodeFormat}
            onChange={(e) => setBarcodeFormat(e.target.value as BarcodeFormat)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {BARCODE_FORMATS.map((f) => (
              <option key={f} value={f}>{t(BARCODE_LABEL_KEYS[f])}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('inventar.settings.tenant.barcodeFormatHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('inventar.settings.tenant.negativeStock')}</p>
          <p className="text-xs text-muted-foreground">{t('inventar.settings.tenant.negativeStockHint')}</p>
        </div>
        <Switch checked={allowNegativeStock} onCheckedChange={setAllowNegativeStock} />
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * InventarSettingsPanel — the "Inventar" entry of the module-settings overlay.
 * Personal view prefs (default tab, density, low-stock warnings) apply per
 * user; the tenant block (default unit, min-stock default, barcode format,
 * negative stock) is editable only by an Inventar-Modul-Leiter / admin.
 */
export function InventarSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'inventar.settings.personal.title',
      descriptionKey: 'inventar.settings.personal.desc',
      scope: 'personal',
      icon: Warehouse,
      children: <InventarPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'inventar.settings.tenant.title',
      descriptionKey: 'inventar.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <InventarTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="inventar"
      titleKey="inventar.settings.panel.title"
      descriptionKey="inventar.settings.panel.subtitle"
      sections={sections}
    />
  )
}
