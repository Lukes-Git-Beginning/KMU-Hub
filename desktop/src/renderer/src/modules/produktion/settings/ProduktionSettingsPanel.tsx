import { useTranslation } from 'react-i18next'
import { Factory, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  useProduktionPrefsStore,
  type ProduktionTab,
  type ProduktionStatusFilter,
} from '@/stores/produktionPrefs'
import { useProduktionTenantStore } from '@/stores/produktionTenant'
import { orderStatusLabelKeys, priorityLabelKeys, PRIORITY_VALUES } from '../produktion-shared'

const selectCls =
  'w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer'

// ─── Persönlich ──────────────────────────────────────────────────

const STATUS_FILTER_VALUES: ProduktionStatusFilter[] = [
  'all', 'planned', 'in_progress', 'completed', 'cancelled',
]

function ProduktionPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useProduktionPrefsStore((s) => s.defaultTab)
  const defaultStatusFilter = useProduktionPrefsStore((s) => s.defaultStatusFilter)
  const setDefaultTab = useProduktionPrefsStore((s) => s.setDefaultTab)
  const setDefaultStatusFilter = useProduktionPrefsStore((s) => s.setDefaultStatusFilter)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as ProduktionTab)}
            className={selectCls}
          >
            <option value="auftraege">{t('produktion.settings.personal.tabAuftraege')}</option>
            <option value="stuecklisten">{t('produktion.settings.personal.tabStuecklisten')}</option>
            <option value="qualitaet">{t('produktion.settings.personal.tabQualitaet')}</option>
            <option value="maschinen">{t('produktion.settings.personal.tabMaschinen')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('produktion.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.personal.defaultStatusFilter')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultStatusFilter}
            onChange={(e) => setDefaultStatusFilter(e.target.value as ProduktionStatusFilter)}
            className={selectCls}
          >
            {STATUS_FILTER_VALUES.map((v) => (
              <option key={v} value={v}>
                {v === 'all' ? t('produktion.filter.all') : t(orderStatusLabelKeys[v] ?? v)}
              </option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('produktion.settings.personal.defaultStatusFilterHint')}</p>
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function ProduktionTenantSettings() {
  const { t } = useTranslation()
  const orderNumberPrefix = useProduktionTenantStore((s) => s.orderNumberPrefix)
  const defaultPriority = useProduktionTenantStore((s) => s.defaultPriority)
  const defaultLeadDays = useProduktionTenantStore((s) => s.defaultLeadDays)
  const requireQcBeforeComplete = useProduktionTenantStore((s) => s.requireQcBeforeComplete)
  const scrapWarnThreshold = useProduktionTenantStore((s) => s.scrapWarnThreshold)
  const setOrderNumberPrefix = useProduktionTenantStore((s) => s.setOrderNumberPrefix)
  const setDefaultPriority = useProduktionTenantStore((s) => s.setDefaultPriority)
  const setDefaultLeadDays = useProduktionTenantStore((s) => s.setDefaultLeadDays)
  const setRequireQcBeforeComplete = useProduktionTenantStore((s) => s.setRequireQcBeforeComplete)
  const setScrapWarnThreshold = useProduktionTenantStore((s) => s.setScrapWarnThreshold)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.tenant.orderNumberPrefix')}</label>
        <input
          type="text"
          maxLength={8}
          value={orderNumberPrefix}
          onChange={(e) => setOrderNumberPrefix(e.target.value)}
          className="w-32 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground font-mono uppercase focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
        <p className="text-xs text-muted-foreground">
          {t('produktion.settings.tenant.orderNumberPrefixHint', {
            example: `${orderNumberPrefix}-${new Date().getFullYear()}-0716-4821`,
          })}
        </p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.tenant.defaultPriority')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={String(defaultPriority)}
            onChange={(e) => setDefaultPriority(Number(e.target.value))}
            className={selectCls}
          >
            {PRIORITY_VALUES.map((p) => (
              <option key={p} value={String(p)}>{t(priorityLabelKeys[p])}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('produktion.settings.tenant.defaultPriorityHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.tenant.defaultLeadDays')}</label>
        <input
          type="number"
          min={1}
          max={365}
          value={defaultLeadDays}
          onChange={(e) => setDefaultLeadDays(Number(e.target.value))}
          className="w-32 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('produktion.settings.tenant.defaultLeadDaysHint')}</p>
      </div>

      <div className="flex items-start justify-between gap-4">
        <div className="space-y-0.5">
          <label className="text-sm font-medium text-foreground">{t('produktion.settings.tenant.requireQc')}</label>
          <p className="text-xs text-muted-foreground">{t('produktion.settings.tenant.requireQcHint')}</p>
        </div>
        <Switch checked={requireQcBeforeComplete} onCheckedChange={setRequireQcBeforeComplete} />
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('produktion.settings.tenant.scrapWarnThreshold')}</label>
        <input
          type="number"
          min={0}
          max={100}
          value={scrapWarnThreshold}
          onChange={(e) => setScrapWarnThreshold(Number(e.target.value))}
          className="w-32 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('produktion.settings.tenant.scrapWarnThresholdHint')}</p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * ProduktionSettingsPanel — the "Produktion" entry of the module-settings
 * overlay. Personal prefs (default tab, saved status filter) apply per user;
 * the tenant block (PA number prefix, default priority/lead time, QC gate
 * before completion, scrap warning threshold) is editable only by a
 * Produktions-Modul-Leiter / admin.
 */
export function ProduktionSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'produktion.settings.personal.title',
      descriptionKey: 'produktion.settings.personal.desc',
      scope: 'personal',
      icon: Factory,
      children: <ProduktionPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'produktion.settings.tenant.title',
      descriptionKey: 'produktion.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <ProduktionTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="produktion"
      titleKey="produktion.settings.panel.title"
      descriptionKey="produktion.settings.panel.subtitle"
      sections={sections}
    />
  )
}
