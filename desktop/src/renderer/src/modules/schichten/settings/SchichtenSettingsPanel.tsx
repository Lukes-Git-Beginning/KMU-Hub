import { useTranslation } from 'react-i18next'
import { CalendarClock, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { Switch } from '@/components/ui/switch'
import { useSchichtenPrefsStore, type SchichtenTab } from '@/stores/schichtenPrefs'
import { useSchichtenTenantStore } from '@/stores/schichtenTenant'

// ─── Persönlich ──────────────────────────────────────────────────

function SchichtenPersonalPrefs() {
  const { t } = useTranslation()
  const defaultTab = useSchichtenPrefsStore((s) => s.defaultTab)
  const showSurcharges = useSchichtenPrefsStore((s) => s.showSurcharges)
  const setDefaultTab = useSchichtenPrefsStore((s) => s.setDefaultTab)
  const setShowSurcharges = useSchichtenPrefsStore((s) => s.setShowSurcharges)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('schichten.settings.personal.defaultTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultTab}
            onChange={(e) => setDefaultTab(e.target.value as SchichtenTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="wochenplan">{t('schichten.tabs.wochenplan')}</option>
            <option value="vorlagen">{t('schichten.settings.personal.tabVorlagen')}</option>
            <option value="anfragen">{t('schichten.settings.personal.tabAnfragen')}</option>
            <option value="verfügbarkeit">{t('schichten.tabs.verfuegbarkeit')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('schichten.settings.personal.defaultTabHint')}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('schichten.settings.personal.showSurcharges')}</p>
          <p className="text-xs text-muted-foreground">{t('schichten.settings.personal.showSurchargesHint')}</p>
        </div>
        <Switch checked={showSurcharges} onCheckedChange={setShowSurcharges} />
      </div>
    </div>
  )
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function SchichtenTenantSettings() {
  const { t } = useTranslation()
  const swapEnabled = useSchichtenTenantStore((s) => s.swapEnabled)
  const maxWeeklyHours = useSchichtenTenantStore((s) => s.maxWeeklyHours)
  const defaultBreakMinutes = useSchichtenTenantStore((s) => s.defaultBreakMinutes)
  const setSwapEnabled = useSchichtenTenantStore((s) => s.setSwapEnabled)
  const setMaxWeeklyHours = useSchichtenTenantStore((s) => s.setMaxWeeklyHours)
  const setDefaultBreakMinutes = useSchichtenTenantStore((s) => s.setDefaultBreakMinutes)

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{t('schichten.settings.tenant.swapEnabled')}</p>
          <p className="text-xs text-muted-foreground">{t('schichten.settings.tenant.swapEnabledHint')}</p>
        </div>
        <Switch checked={swapEnabled} onCheckedChange={setSwapEnabled} />
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('schichten.settings.tenant.maxWeeklyHours')}</label>
        <input
          type="number"
          min={20}
          max={48}
          value={maxWeeklyHours}
          onChange={(e) => setMaxWeeklyHours(Number(e.target.value))}
          className="w-24 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('schichten.settings.tenant.maxWeeklyHoursHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('schichten.settings.tenant.defaultBreak')}</label>
        <input
          type="number"
          min={0}
          max={120}
          step={5}
          value={defaultBreakMinutes}
          onChange={(e) => setDefaultBreakMinutes(Number(e.target.value))}
          className="w-24 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
        />
        <p className="text-xs text-muted-foreground">{t('schichten.settings.tenant.defaultBreakHint')}</p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * SchichtenSettingsPanel — the "Schichten" entry of the module-settings
 * overlay. Personal prefs (default tab, surcharge badges) apply per user;
 * the tenant block (swap policy, weekly hours ceiling, default break) is
 * editable only by a Schichten-Modul-Leiter / admin.
 */
export function SchichtenSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'schichten.settings.personal.title',
      descriptionKey: 'schichten.settings.personal.desc',
      scope: 'personal',
      icon: CalendarClock,
      children: <SchichtenPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'schichten.settings.tenant.title',
      descriptionKey: 'schichten.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <SchichtenTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="schichten"
      titleKey="schichten.settings.panel.title"
      descriptionKey="schichten.settings.panel.subtitle"
      sections={sections}
    />
  )
}
