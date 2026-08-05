import { useTranslation } from 'react-i18next'
import { SlidersHorizontal, Clock, Route, ChevronDown, Star } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import {
  useHelpdeskPrefsStore,
  type HelpdeskStartTab,
  type HelpdeskStatusDefault,
} from '@/stores/helpdeskPrefs'
import { useHelpdeskStore } from '@/stores/helpdesk'
import { BusinessHoursDialog } from '../BusinessHoursDialog'
import { TicketRoutingConfig } from '../TicketRoutingConfig'

// ─── Persönlich ──────────────────────────────────────────────────

function HelpdeskPersonalPrefs() {
  const { t } = useTranslation()
  const startTab = useHelpdeskPrefsStore((s) => s.startTab)
  const defaultStatusFilter = useHelpdeskPrefsStore((s) => s.defaultStatusFilter)
  const setStartTab = useHelpdeskPrefsStore((s) => s.setStartTab)
  const setDefaultStatusFilter = useHelpdeskPrefsStore((s) => s.setDefaultStatusFilter)

  const tabOptions: { value: HelpdeskStartTab; labelKey: string }[] = [
    { value: 'tickets', labelKey: 'helpdesk.settings.personal.tabTickets' },
    { value: 'wissensdatenbank', labelKey: 'helpdesk.settings.personal.tabKb' },
    { value: 'statistik', labelKey: 'helpdesk.settings.personal.tabStats' },
  ]
  const statusOptions: { value: HelpdeskStatusDefault; labelKey: string }[] = [
    { value: 'all', labelKey: 'helpdesk.filter.allStatuses' },
    { value: 'open', labelKey: 'helpdesk.status.open' },
    { value: 'in_progress', labelKey: 'helpdesk.status.inProgress' },
    { value: 'waiting', labelKey: 'helpdesk.status.waiting' },
    { value: 'resolved', labelKey: 'helpdesk.status.resolved' },
    { value: 'closed', labelKey: 'helpdesk.status.closed' },
  ]

  return (
    <div className="space-y-5">
      {/* Start tab */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('helpdesk.settings.personal.startTab')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={startTab}
            onChange={(e) => setStartTab(e.target.value as HelpdeskStartTab)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {tabOptions.map((o) => <option key={o.value} value={o.value}>{t(o.labelKey)}</option>)}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('helpdesk.settings.personal.startTabHint')}</p>
      </div>

      {/* Default status filter */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('helpdesk.settings.personal.defaultStatus')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={defaultStatusFilter}
            onChange={(e) => setDefaultStatusFilter(e.target.value as HelpdeskStatusDefault)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            {statusOptions.map((o) => <option key={o.value} value={o.value}>{t(o.labelKey)}</option>)}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('helpdesk.settings.personal.defaultStatusHint')}</p>
      </div>
    </div>
  )
}

// ─── CSAT (tenant) ───────────────────────────────────────────────

const CSAT_DELAY_OPTIONS: { hours: number; labelKey: string }[] = [
  { hours: 0, labelKey: 'helpdesk.settings.csat.delayImmediate' },
  { hours: 1, labelKey: 'helpdesk.settings.csat.delay1h' },
  { hours: 4, labelKey: 'helpdesk.settings.csat.delay4h' },
  { hours: 24, labelKey: 'helpdesk.settings.csat.delay1d' },
  { hours: 72, labelKey: 'helpdesk.settings.csat.delay3d' },
]

function HelpdeskCsatConfig() {
  const { t } = useTranslation()
  const csatEnabled = useHelpdeskStore((s) => s.csatEnabled)
  const csatDelayHours = useHelpdeskStore((s) => s.csatDelayHours)
  const csatQuestion = useHelpdeskStore((s) => s.csatQuestion)
  const setCsatEnabled = useHelpdeskStore((s) => s.setCsatEnabled)
  const setCsatDelayHours = useHelpdeskStore((s) => s.setCsatDelayHours)
  const setCsatQuestion = useHelpdeskStore((s) => s.setCsatQuestion)

  return (
    <div className="space-y-5">
      {/* On/off */}
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <label className="text-sm font-medium text-foreground">{t('helpdesk.settings.csat.enabledLabel')}</label>
          <p className="text-xs text-muted-foreground">{t('helpdesk.settings.csat.enabledHint')}</p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={csatEnabled}
          onClick={() => setCsatEnabled(!csatEnabled)}
          className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${csatEnabled ? 'bg-primary' : 'bg-secondary'}`}
        >
          <span className={`inline-block h-4 w-4 rounded-full bg-white shadow transition-transform ${csatEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
        </button>
      </div>

      {/* Delay */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('helpdesk.settings.csat.delayLabel')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={csatDelayHours}
            disabled={!csatEnabled}
            onChange={(e) => setCsatDelayHours(Number(e.target.value))}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {CSAT_DELAY_OPTIONS.map((o) => <option key={o.hours} value={o.hours}>{t(o.labelKey)}</option>)}
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('helpdesk.settings.csat.delayHint')}</p>
      </div>

      {/* Survey question — what the customer sees in the survey */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('helpdesk.settings.csat.questionLabel')}</label>
        <textarea
          value={csatQuestion}
          disabled={!csatEnabled}
          onChange={(e) => setCsatQuestion(e.target.value)}
          rows={2}
          placeholder={t('helpdesk.settings.csat.questionPlaceholder')}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none disabled:opacity-50 disabled:cursor-not-allowed"
        />
        <p className="text-xs text-muted-foreground">{t('helpdesk.settings.csat.questionHint')}</p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * HelpdeskSettingsPanel — the "Helpdesk" entry of the module-settings overlay.
 * Personal prefs (start tab, default filter) apply for real. Tenant sections
 * (business hours, routing rules) save to the helpdesk store via H-1 actions —
 * editable only by a Helpdesk-Modul-Leiter / admin (ModuleSettingsShell gate).
 */
export function HelpdeskSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'helpdesk.settings.personal.title',
      descriptionKey: 'helpdesk.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <HelpdeskPersonalPrefs />,
    },
    {
      id: 'businessHours',
      titleKey: 'helpdesk.settings.businessHours.title',
      descriptionKey: 'helpdesk.settings.businessHours.desc',
      scope: 'tenant',
      icon: Clock,
      children: <BusinessHoursDialog embedded />,
    },
    {
      id: 'routing',
      titleKey: 'helpdesk.settings.routing.title',
      descriptionKey: 'helpdesk.settings.routing.desc',
      scope: 'tenant',
      icon: Route,
      children: <TicketRoutingConfig embedded />,
    },
    {
      id: 'csat',
      titleKey: 'helpdesk.settings.csat.title',
      descriptionKey: 'helpdesk.settings.csat.desc',
      scope: 'tenant',
      icon: Star,
      children: <HelpdeskCsatConfig />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="helpdesk"
      titleKey="helpdesk.settings.title"
      descriptionKey="helpdesk.settings.subtitle"
      sections={sections}
    />
  )
}
