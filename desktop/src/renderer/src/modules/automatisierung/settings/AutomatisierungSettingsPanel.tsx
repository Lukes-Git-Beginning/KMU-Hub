import { useTranslation } from 'react-i18next'
import * as Switch from '@radix-ui/react-switch'
import { SlidersHorizontal, Archive, BellRing, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useAutomatisierungPrefsStore,
  type AutomatisierungStartTab,
} from '@/stores/automatisierungPrefs'
import { useAutomatisierungTenantStore } from '@/stores/automatisierungTenant'

// ─── Persönlich ──────────────────────────────────────────────────

function PersonalPrefs() {
  const { t } = useTranslation()
  const startTab = useAutomatisierungPrefsStore((s) => s.startTab)
  const setStartTab = useAutomatisierungPrefsStore((s) => s.setStartTab)

  const tabOptions: { value: AutomatisierungStartTab; labelKey: string }[] = [
    { value: 'automations', labelKey: 'automatisierung.tabs.myAutomations' },
    { value: 'templates', labelKey: 'automatisierung.tabs.templates' },
    { value: 'log', labelKey: 'automatisierung.tabs.log' },
  ]

  return (
    <div className="space-y-1.5">
      <label className="text-sm font-medium text-foreground">{t('automatisierung.settings.personal.startTab')}</label>
      <div className="relative w-64 max-w-full">
        <select
          value={startTab}
          onChange={(e) => setStartTab(e.target.value as AutomatisierungStartTab)}
          className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
        >
          {tabOptions.map((o) => <option key={o.value} value={o.value}>{t(o.labelKey)}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      </div>
      <p className="text-xs text-muted-foreground">{t('automatisierung.settings.personal.startTabHint')}</p>
    </div>
  )
}

// ─── Für alle: Protokoll-Aufbewahrung ────────────────────────────

function RetentionPrefs() {
  const { t } = useTranslation()
  const days = useAutomatisierungTenantStore((s) => s.logRetentionDays)
  const setDays = useAutomatisierungTenantStore((s) => s.setLogRetentionDays)

  return (
    <div className="space-y-1.5">
      <label className="text-sm font-medium text-foreground">{t('automatisierung.settings.retention.label')}</label>
      <div className="relative w-64 max-w-full">
        <select
          value={days}
          onChange={(e) => setDays(Number(e.target.value))}
          className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
        >
          {[30, 90, 365].map((d) => (
            <option key={d} value={d}>{t('automatisierung.settings.retention.days', { days: d })}</option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      </div>
      <p className="text-xs text-muted-foreground">{t('automatisierung.settings.retention.hint')}</p>
    </div>
  )
}

// ─── Für alle: Fehler-Benachrichtigung ───────────────────────────

function FailureNotifyPrefs() {
  const { t } = useTranslation()
  const on = useAutomatisierungTenantStore((s) => s.notifyOnFailure)
  const setOn = useAutomatisierungTenantStore((s) => s.setNotifyOnFailure)

  return (
    <div className="flex items-center justify-between gap-4">
      <div>
        <label className="text-sm font-medium text-foreground">{t('automatisierung.settings.failure.label')}</label>
        <p className="mt-0.5 text-xs text-muted-foreground">{t('automatisierung.settings.failure.hint')}</p>
      </div>
      <Switch.Root
        checked={on}
        onCheckedChange={setOn}
        className="relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
      >
        <Switch.Thumb className="block h-4 w-4 rounded-full bg-white shadow-sm transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5" />
      </Switch.Root>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * AutomatisierungSettingsPanel — the "Automatisierung" entry of the module-settings
 * overlay. Personal start-tab applies for real (AutomatisierungPage reads it).
 * Tenant sections (log retention, failure notification) are demo-stateful and
 * editable only by a Modul-Leiter / admin (ModuleSettingsShell gate).
 */
export function AutomatisierungSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'automatisierung.settings.personal.title',
      descriptionKey: 'automatisierung.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalPrefs />,
    },
    {
      id: 'retention',
      titleKey: 'automatisierung.settings.retention.title',
      descriptionKey: 'automatisierung.settings.retention.desc',
      scope: 'tenant',
      icon: Archive,
      children: <RetentionPrefs />,
    },
    {
      id: 'failure',
      titleKey: 'automatisierung.settings.failure.title',
      descriptionKey: 'automatisierung.settings.failure.desc',
      scope: 'tenant',
      icon: BellRing,
      children: <FailureNotifyPrefs />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="automatisierung"
      titleKey="automatisierung.settings.title"
      descriptionKey="automatisierung.settings.subtitle"
      sections={sections}
    />
  )
}
