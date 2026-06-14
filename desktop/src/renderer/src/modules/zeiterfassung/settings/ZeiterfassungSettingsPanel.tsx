import { useTranslation } from 'react-i18next'
import { SlidersHorizontal, Clock, Coffee, CalendarDays, BarChart3, Calendar } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { useSelfProfile } from '@/api/hooks/hr-hooks'
import { STANDARD_DAILY_HOURS } from '@/lib/worktime'
import {
  useZeiterfassungPrefsStore,
  type ZeiterfassungDefaultView,
} from '@/stores/zeiterfassungPrefs'
import {
  useZeiterfassungSettingsStore,
  HOLIDAY_REGIONS,
  ROUNDING_MODES,
} from '@/stores/zeiterfassungSettings'

const segBtn = (active: boolean) =>
  `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
    active
      ? 'border-primary bg-primary/5 font-medium text-primary'
      : 'border-border text-foreground hover:bg-secondary'
  }`

const numInput =
  'h-9 w-full rounded-lg border border-border bg-transparent px-2 text-sm text-foreground outline-none focus:border-primary'

// ─── Persönlich ──────────────────────────────────────────────────

function PersonalPrefs() {
  const { t } = useTranslation()
  const defaultView = useZeiterfassungPrefsStore((s) => s.defaultView)
  const clockOutReminder = useZeiterfassungPrefsStore((s) => s.clockOutReminder)
  const setDefaultView = useZeiterfassungPrefsStore((s) => s.setDefaultView)
  const setClockOutReminder = useZeiterfassungPrefsStore((s) => s.setClockOutReminder)
  const { data: profile } = useSelfProfile()
  const weeklyHours = profile?.workDaysPerWeek ? profile.workDaysPerWeek * STANDARD_DAILY_HOURS : null

  const viewOptions = [
    { id: 'today', labelKey: 'profil.zeiterfassung.viewToday', icon: Clock },
    { id: 'week', labelKey: 'profil.zeiterfassung.viewWeek', icon: Calendar },
    { id: 'analytics', labelKey: 'zeiterfassung.analytics.tab', icon: BarChart3 },
  ] as const satisfies ReadonlyArray<{ id: ZeiterfassungDefaultView; labelKey: string; icon: typeof Clock }>

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('zeiterfassung.settings.personal.defaultView')}</label>
        <div className="grid grid-cols-3 gap-2">
          {viewOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDefaultView(opt.id)} className={segBtn(defaultView === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {weeklyHours != null && (
        <div className="rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground">{t('zeiterfassung.settings.personal.weeklyTarget')}</span>
            <span className="text-sm font-medium tabular-nums text-foreground">
              {t('zeiterfassung.settings.personal.weeklyTargetValue', { hours: weeklyHours })}
            </span>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">{t('zeiterfassung.settings.personal.weeklyTargetHint')}</p>
        </div>
      )}

      <label className="flex cursor-pointer items-center gap-2">
        <input
          type="checkbox"
          checked={clockOutReminder}
          onChange={() => setClockOutReminder(!clockOutReminder)}
          className="h-4 w-4 rounded border-border accent-primary"
        />
        <span className="text-sm text-foreground">{t('zeiterfassung.settings.personal.clockOutReminder')}</span>
      </label>
    </div>
  )
}

// ─── Für alle: Arbeitszeit-Regeln ────────────────────────────────

function WorkRulesSettings() {
  const { t } = useTranslation()
  const autoBreakAfterHours = useZeiterfassungSettingsStore((s) => s.autoBreakAfterHours)
  const autoBreakMinutes = useZeiterfassungSettingsStore((s) => s.autoBreakMinutes)
  const rounding = useZeiterfassungSettingsStore((s) => s.rounding)
  const setAutoBreakAfterHours = useZeiterfassungSettingsStore((s) => s.setAutoBreakAfterHours)
  const setAutoBreakMinutes = useZeiterfassungSettingsStore((s) => s.setAutoBreakMinutes)
  const setRounding = useZeiterfassungSettingsStore((s) => s.setRounding)

  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <label className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Coffee className="h-4 w-4 text-muted-foreground" />
          {t('zeiterfassung.settings.rules.autoBreak')}
        </label>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <span className="mb-1 block text-xs text-muted-foreground">{t('zeiterfassung.settings.rules.afterHours')}</span>
            <input
              type="number" min={0} max={12} step={0.5}
              value={autoBreakAfterHours}
              onChange={(e) => setAutoBreakAfterHours(Math.max(0, Number(e.target.value)))}
              className={numInput}
            />
          </div>
          <div>
            <span className="mb-1 block text-xs text-muted-foreground">{t('zeiterfassung.settings.rules.breakMinutes')}</span>
            <input
              type="number" min={0} max={120} step={5}
              value={autoBreakMinutes}
              onChange={(e) => setAutoBreakMinutes(Math.max(0, Number(e.target.value)))}
              className={numInput}
            />
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{t('zeiterfassung.settings.rules.autoBreakHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('zeiterfassung.settings.rules.rounding')}</label>
        <div className="flex gap-2">
          {ROUNDING_MODES.map((m) => (
            <button key={m} onClick={() => setRounding(m)} className={segBtn(rounding === m)}>
              {t(`zeiterfassung.settings.rules.rounding_${m}`)}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ─── Für alle: Feiertagsregion ───────────────────────────────────

function HolidaySettings() {
  const { t } = useTranslation()
  const holidayRegion = useZeiterfassungSettingsStore((s) => s.holidayRegion)
  const setHolidayRegion = useZeiterfassungSettingsStore((s) => s.setHolidayRegion)

  return (
    <div className="space-y-2">
      <div className="flex gap-2">
        {HOLIDAY_REGIONS.map((r) => (
          <button key={r} onClick={() => setHolidayRegion(r)} className={segBtn(holidayRegion === r)}>
            {t(`zeiterfassung.settings.holiday.region_${r}`)}
          </button>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{t('zeiterfassung.settings.holiday.hint')}</p>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * ZeiterfassungSettingsPanel — the "Zeiterfassung" entry of the module-settings
 * overlay. Personal prefs (default view, daily target, reminder) apply for real
 * in the module tab. Tenant rules run mock-first on stores/zeiterfassungSettings
 * until the backend persists them (backend-gaps.md).
 */
export function ZeiterfassungSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'zeiterfassung.settings.personal.title',
      descriptionKey: 'zeiterfassung.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <PersonalPrefs />,
    },
    {
      id: 'rules',
      titleKey: 'zeiterfassung.settings.rules.title',
      descriptionKey: 'zeiterfassung.settings.rules.desc',
      scope: 'tenant',
      icon: Clock,
      children: <WorkRulesSettings />,
    },
    {
      id: 'holiday',
      titleKey: 'zeiterfassung.settings.holiday.title',
      descriptionKey: 'zeiterfassung.settings.holiday.desc',
      scope: 'tenant',
      icon: CalendarDays,
      children: <HolidaySettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="zeiterfassung"
      titleKey="zeiterfassung.settings.title"
      descriptionKey="zeiterfassung.settings.subtitle"
      sections={sections}
    />
  )
}
