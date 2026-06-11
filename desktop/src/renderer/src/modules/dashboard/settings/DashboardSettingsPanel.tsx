import { useTranslation } from 'react-i18next'
import { SlidersHorizontal, LayoutGrid, ShieldCheck, Rows3, Rows4 } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { useDashboardPrefsStore, type DashboardDensity } from '@/stores/dashboardPrefs'
import { useDashboardSettingsStore } from '@/stores/dashboardSettings'
import { ALL_WIDGET_IDS, type WidgetId } from '@/stores/dashboard'

// Deliberately NOT importing WidgetRegistry here: this panel is reachable from
// the boot-time settings registry, and WidgetRegistry resolves its names via
// i18next.t() at module load — importing it that early yields empty names
// app-wide. Labels are translated at render time instead (same keys).
const WIDGET_NAME_KEYS = {
  'recent-contacts': 'widgets.registry.recentContacts.name',
  'deal-pipeline': 'widgets.registry.dealPipeline.name',
  'unread-messages': 'widgets.registry.unreadMessages.name',
  'activity-feed': 'widgets.registry.activityFeed.name',
  'quick-actions': 'widgets.registry.quickActions.name',
  'notification-summary': 'widgets.registry.notificationSummary.name',
  'notification-feed': 'widgets.registry.notificationFeed.name',
  'kpi-revenue': 'widgets.registry.kpiRevenue.name',
  'kpi-tasks': 'widgets.registry.kpiTasks.name',
  'kpi-deals': 'widgets.registry.kpiDeals.name',
  'calendar-upcoming': 'widgets.registry.calendarUpcoming.name',
  'team-status': 'widgets.registry.teamStatus.name',
  'mini-chart': 'widgets.registry.miniChart.name',
  'my-tasks': 'widgets.registry.myTasks.name',
  'my-calendar': 'widgets.registry.myCalendar.name',
  'time-clock': 'widgets.registry.timeClock.name',
  'team-chat': 'widgets.registry.teamChat.name',
  'absences': 'widgets.registry.absences.name',
  'birthdays': 'widgets.registry.birthdays.name',
} as const satisfies Record<WidgetId, string>

const segBtn = (active: boolean) =>
  `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
    active
      ? 'border-primary bg-primary/5 font-medium text-primary'
      : 'border-border text-foreground hover:bg-secondary'
  }`

const chipBtn = (active: boolean) =>
  `flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs transition-colors ${
    active
      ? 'border-primary bg-primary/5 font-medium text-primary'
      : 'border-border text-muted-foreground hover:bg-secondary'
  }`

// ─── Persönlich ──────────────────────────────────────────────────

function DashboardPersonalPrefs() {
  const { t } = useTranslation()
  const showGreeting = useDashboardPrefsStore((s) => s.showGreeting)
  const density = useDashboardPrefsStore((s) => s.density)
  const setShowGreeting = useDashboardPrefsStore((s) => s.setShowGreeting)
  const setDensity = useDashboardPrefsStore((s) => s.setDensity)

  // Literal labelKey inference keeps t() happy under typed i18n keys.
  const densityOptions = [
    { id: 'comfortable', labelKey: 'dashboard.settings.personal.densityComfortable', icon: Rows3 },
    { id: 'compact', labelKey: 'dashboard.settings.personal.densityCompact', icon: Rows4 },
  ] as const satisfies ReadonlyArray<{ id: DashboardDensity; labelKey: string; icon: typeof Rows3 }>

  return (
    <div className="space-y-5">
      {/* Greeting on/off */}
      <div className="space-y-1.5">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showGreeting}
            onChange={(e) => setShowGreeting(e.target.checked)}
            className="h-4 w-4 rounded border-border accent-primary"
          />
          <span className="text-sm font-medium text-foreground">
            {t('dashboard.settings.personal.showGreeting')}
          </span>
        </label>
        <p className="text-xs text-muted-foreground">{t('dashboard.settings.personal.showGreetingHint')}</p>
      </div>

      {/* Widget grid density */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dashboard.settings.personal.density')}
        </label>
        <div className="flex gap-2">
          {densityOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDensity(opt.id)} className={segBtn(density === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
        <p className="text-xs text-muted-foreground">{t('dashboard.settings.personal.densityHint')}</p>
      </div>
    </div>
  )
}

// ─── Für alle: Widget-Chips (shared list UI) ─────────────────────

function WidgetChipList({
  selected,
  onToggle,
  disabledIds,
}: {
  selected: WidgetId[]
  onToggle: (id: WidgetId) => void
  /** Widgets rendered non-interactive (e.g. not allowed tenant-wide). */
  disabledIds?: WidgetId[]
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap gap-2">
      {ALL_WIDGET_IDS.map((id) => {
        const disabled = disabledIds?.includes(id)
        return (
          <button
            key={id}
            onClick={() => onToggle(id)}
            disabled={disabled}
            className={`${chipBtn(selected.includes(id))} ${disabled ? 'opacity-40 cursor-not-allowed' : ''}`}
          >
            {t(WIDGET_NAME_KEYS[id])}
          </button>
        )
      })}
    </div>
  )
}

function TeamDefaultWidgetsSettings() {
  const { t } = useTranslation()
  const defaultWidgets = useDashboardSettingsStore((s) => s.defaultWidgets)
  const allowedWidgets = useDashboardSettingsStore((s) => s.allowedWidgets)
  const toggleDefaultWidget = useDashboardSettingsStore((s) => s.toggleDefaultWidget)

  const notAllowed = ALL_WIDGET_IDS.filter((id) => !allowedWidgets.includes(id))

  return (
    <div className="space-y-2">
      <WidgetChipList selected={defaultWidgets} onToggle={toggleDefaultWidget} disabledIds={notAllowed} />
      <p className="text-xs text-muted-foreground">{t('dashboard.settings.teamDefault.hint')}</p>
    </div>
  )
}

function AllowedWidgetsSettings() {
  const { t } = useTranslation()
  const allowedWidgets = useDashboardSettingsStore((s) => s.allowedWidgets)
  const toggleAllowedWidget = useDashboardSettingsStore((s) => s.toggleAllowedWidget)

  return (
    <div className="space-y-2">
      <WidgetChipList selected={allowedWidgets} onToggle={toggleAllowedWidget} />
      <p className="text-xs text-muted-foreground">{t('dashboard.settings.allowed.hint')}</p>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * DashboardSettingsPanel — the "Dashboard" entry of the module-settings
 * overlay. Personal prefs (greeting, grid density) apply for real. Tenant
 * sections run mock-first on stores/dashboardSettings (admin-only — the
 * dashboard is not delegable as Modul-Leiter); the allowed-widgets list
 * already filters the widget picker for real. Layout persistence itself is
 * server-synced separately (stores/dashboard).
 */
export function DashboardSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'dashboard.settings.personal.title',
      descriptionKey: 'dashboard.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <DashboardPersonalPrefs />,
    },
    {
      id: 'teamDefault',
      titleKey: 'dashboard.settings.teamDefault.title',
      descriptionKey: 'dashboard.settings.teamDefault.desc',
      scope: 'tenant',
      icon: LayoutGrid,
      children: <TeamDefaultWidgetsSettings />,
    },
    {
      id: 'allowed',
      titleKey: 'dashboard.settings.allowed.title',
      descriptionKey: 'dashboard.settings.allowed.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <AllowedWidgetsSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="dashboard"
      titleKey="dashboard.settings.title"
      descriptionKey="dashboard.settings.subtitle"
      sections={sections}
    />
  )
}
