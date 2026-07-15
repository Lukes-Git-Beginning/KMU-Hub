import { useTranslation } from 'react-i18next'
import { Building2, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  useVermietungViewPrefsStore,
  type VermietungTab,
} from '@/stores/vermietungViewPrefs'
import {
  useVermietungTenantStore,
  VERMIETUNG_CURRENCY_OPTIONS,
} from '@/stores/vermietungTenant'

// ─── Persönlich ──────────────────────────────────────────────────

function VermietungPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useVermietungViewPrefsStore((s) => s.defaultTab)
  const showKpis = useVermietungViewPrefsStore((s) => s.showKpis)
  const setDefaultTab = useVermietungViewPrefsStore((s) => s.setDefaultTab)
  const setShowKpis = useVermietungViewPrefsStore((s) => s.setShowKpis)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('vermietung.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as VermietungTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="objekte">{t('vermietung.settings.personal.tabObjekte')}</option>
            <option value="reservierungen">{t('vermietung.settings.personal.tabReservierungen')}</option>
            <option value="kalender">{t('vermietung.tab.kalender')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('vermietung.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('vermietung.settings.personal.showKpis')}</p>
          <p className="text-xs text-muted-foreground">{t('vermietung.settings.personal.showKpisHint')}</p>
        </div>
        <Switch checked={showKpis} onCheckedChange={setShowKpis} />
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function VermietungTenantSettings() {
  const { t } = useTranslation()
  const defaultCurrency = useVermietungTenantStore((s) => s.defaultCurrency)
  const bufferDays = useVermietungTenantStore((s) => s.bufferDays)
  const requireDeposit = useVermietungTenantStore((s) => s.requireDeposit)
  const setDefaultCurrency = useVermietungTenantStore((s) => s.setDefaultCurrency)
  const setBufferDays = useVermietungTenantStore((s) => s.setBufferDays)
  const setRequireDeposit = useVermietungTenantStore((s) => s.setRequireDeposit)

  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('vermietung.settings.tenant.defaultCurrency')}</label>
          <div className="relative">
            <select
              value={defaultCurrency}
              onChange={(e) => setDefaultCurrency(e.target.value)}
              className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
            >
              {VERMIETUNG_CURRENCY_OPTIONS.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
          <p className="text-xs text-muted-foreground">{t('vermietung.settings.tenant.defaultCurrencyHint')}</p>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('vermietung.settings.tenant.bufferDays')}</label>
          <div className="relative">
            <select
              value={String(bufferDays)}
              onChange={(e) => setBufferDays(Number(e.target.value))}
              className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
            >
              <option value="0">{t('vermietung.settings.tenant.bufferNone')}</option>
              <option value="1">{t('vermietung.settings.tenant.bufferDay', { count: 1 })}</option>
              <option value="2">{t('vermietung.settings.tenant.bufferDay', { count: 2 })}</option>
              <option value="3">{t('vermietung.settings.tenant.bufferDay', { count: 3 })}</option>
            </select>
            <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
          <p className="text-xs text-muted-foreground">{t('vermietung.settings.tenant.bufferDaysHint')}</p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('vermietung.settings.tenant.requireDeposit')}</p>
          <p className="text-xs text-muted-foreground">{t('vermietung.settings.tenant.requireDepositHint')}</p>
        </div>
        <Switch checked={requireDeposit} onCheckedChange={setRequireDeposit} />
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * VermietungSettingsPanel — the "Vermietung" entry of the module-settings
 * overlay. Personal view prefs (default tab, KPI row) apply per user; the
 * tenant block (currency, rental buffer, deposit policy) is editable only by
 * a Vermietung-Modul-Leiter / admin.
 */
export function VermietungSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'vermietung.settings.personal.title',
      descriptionKey: 'vermietung.settings.personal.desc',
      scope: 'personal',
      icon: Building2,
      children: <VermietungPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'vermietung.settings.tenant.title',
      descriptionKey: 'vermietung.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <VermietungTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="vermietung"
      titleKey="vermietung.settings.panel.title"
      descriptionKey="vermietung.settings.panel.subtitle"
      sections={sections}
    />
  )
}
