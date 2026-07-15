import { useTranslation } from 'react-i18next'
import { ClipboardCheck, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import {
  useRapportePrefsStore,
  type RapporteDateFilter,
} from '@/stores/rapportePrefs'
import {
  useRapporteTenantStore,
  RAPPORTE_CURRENCY_OPTIONS,
} from '@/stores/rapporteTenant'

// ─── Persönlich ──────────────────────────────────────────────────

function RapportePersonalPrefs() {
  const { t } = useTranslation()
  const defaultDateFilter = useRapportePrefsStore((s) => s.defaultDateFilter)
  const defaultWorkStart = useRapportePrefsStore((s) => s.defaultWorkStart)
  const defaultWorkEnd = useRapportePrefsStore((s) => s.defaultWorkEnd)
  const defaultBreakMinutes = useRapportePrefsStore((s) => s.defaultBreakMinutes)
  const setDefaultDateFilter = useRapportePrefsStore((s) => s.setDefaultDateFilter)
  const setDefaultWorkStart = useRapportePrefsStore((s) => s.setDefaultWorkStart)
  const setDefaultWorkEnd = useRapportePrefsStore((s) => s.setDefaultWorkEnd)
  const setDefaultBreakMinutes = useRapportePrefsStore((s) => s.setDefaultBreakMinutes)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('rapporte.settings.personal.defaultDateFilter')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultDateFilter}
            onChange={(e) => setDefaultDateFilter(e.target.value as RapporteDateFilter)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="week">{t('rapporte.filter.thisWeek')}</option>
            <option value="month">{t('rapporte.filter.thisMonth')}</option>
            <option value="all">{t('rapporte.filter.all')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('rapporte.settings.personal.defaultDateFilterHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('rapporte.settings.personal.defaultTimes')}</label>
        <div className="flex flex-wrap items-end gap-3">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t('rapporte.dialog.workStart')}</p>
            <input
              type="time"
              value={defaultWorkStart}
              onChange={(e) => setDefaultWorkStart(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t('rapporte.dialog.workEnd')}</p>
            <input
              type="time"
              value={defaultWorkEnd}
              onChange={(e) => setDefaultWorkEnd(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t('rapporte.dialog.breakMin')}</p>
            <input
              type="number"
              min={0}
              value={defaultBreakMinutes}
              onChange={(e) => setDefaultBreakMinutes(Number(e.target.value))}
              className="w-24 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
            />
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{t('rapporte.settings.personal.defaultTimesHint')}</p>
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function RapporteTenantSettings() {
  const { t } = useTranslation()
  const requireSignature = useRapporteTenantStore((s) => s.requireSignature)
  const currency = useRapporteTenantStore((s) => s.currency)
  const setRequireSignature = useRapporteTenantStore((s) => s.setRequireSignature)
  const setCurrency = useRapporteTenantStore((s) => s.setCurrency)

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('rapporte.settings.tenant.requireSignature')}</p>
          <p className="text-xs text-muted-foreground">{t('rapporte.settings.tenant.requireSignatureHint')}</p>
        </div>
        <Switch checked={requireSignature} onCheckedChange={setRequireSignature} />
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('rapporte.settings.tenant.currency')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {RAPPORTE_CURRENCY_OPTIONS.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('rapporte.settings.tenant.currencyHint')}</p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * RapporteSettingsPanel — the "Rapporte" entry of the module-settings overlay.
 * Personal prefs (default date filter, default shift times) apply per user;
 * the tenant block (signature requirement, currency) is editable only by a
 * Rapporte-Modul-Leiter / admin.
 */
export function RapporteSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'rapporte.settings.personal.title',
      descriptionKey: 'rapporte.settings.personal.desc',
      scope: 'personal',
      icon: ClipboardCheck,
      children: <RapportePersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'rapporte.settings.tenant.title',
      descriptionKey: 'rapporte.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <RapporteTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="rapporte"
      titleKey="rapporte.settings.panel.title"
      descriptionKey="rapporte.settings.panel.subtitle"
      sections={sections}
    />
  )
}
