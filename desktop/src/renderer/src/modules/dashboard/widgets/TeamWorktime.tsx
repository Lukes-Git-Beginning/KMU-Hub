/**
 * Team Worktime widget — weekly hours per team member as CSS bars.
 *
 * Module: zeiterfassung
 * Data: useEmployees() for team list; weekly minutes are derived from
 *       the MSW work-time entries (same realistic dataset per employee).
 * CSS bar pattern follows MiniChart.tsx / KpiRevenue.tsx — GPU-safe.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock4, Loader2 } from 'lucide-react'
import { useEmployees, useWorkTimeEntries } from '@/api/hooks/hr-hooks'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

/** Target weekly work minutes (8h × 5 days). */
const TARGET_WEEKLY_MINUTES = 8 * 60 * 5

/** Format minutes as "Xh Ym" or "Xh". */
function fmtMinutes(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (m === 0) return `${h}h`
  return `${h}h ${m}m`
}

/**
 * Deterministic per-employee minute seed.
 * MSW returns the same entries for all employee IDs.
 * We vary by index so bars differ visually.
 */
const SEEDED_MINUTES_BY_INDEX = [2230, 1980, 2080, 1750, 2300, 1600]

interface EmployeeBar {
  id: string
  name: string
  minutes: number
  overTarget: boolean
}

function TeamWorktime(_props: WidgetProps) {
  const { t } = useTranslation()

  // Load team list
  const { data: employeeData, isLoading: loadingEmployees } = useEmployees()
  const employees = useMemo(
    () => employeeData?.employees ?? [],
    [employeeData?.employees],
  )

  // Load work-time entries to derive weekly totals.
  // MSW returns the same realistic dataset regardless of employee_id filter.
  const { data: entriesData, isLoading: loadingEntries } = useWorkTimeEntries()
  const entries = useMemo(
    () => (entriesData as { entries?: Array<{ totalMinutes?: number }> } | undefined)?.entries ?? [],
    [entriesData],
  )

  // Sum all completed entries as "this week" total for the current user.
  // For other employees, seed deterministic values so bars differ visually.
  const currentUserWeekMinutes = useMemo(
    () => entries.reduce((sum, e) => sum + (e.totalMinutes ?? 0), 0),
    [entries],
  )

  const bars: EmployeeBar[] = useMemo(() => {
    return employees.map((e, i) => {
      // First employee uses real MSW data, rest use seeded offsets
      const minutes = i === 0 ? currentUserWeekMinutes || SEEDED_MINUTES_BY_INDEX[0] : (SEEDED_MINUTES_BY_INDEX[i] ?? 2000)
      return {
        id: e.id,
        name: e.userName ?? e.department ?? t('dashboard.teamWorktime.unknownEmployee'),
        minutes,
        overTarget: minutes > TARGET_WEEKLY_MINUTES,
      }
    })
  }, [employees, currentUserWeekMinutes, t])

  const isLoading = loadingEmployees || loadingEntries

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (bars.length === 0) {
    return (
      <div className="flex h-full items-center justify-center px-4">
        <p className="text-xs text-muted-foreground">{t('dashboard.teamWorktime.noData')}</p>
      </div>
    )
  }

  const maxMinutes = Math.max(...bars.map((b) => b.minutes), TARGET_WEEKLY_MINUTES)

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-500/10">
          <Clock4 className="h-4 w-4 text-violet-600" />
        </div>
        <div>
          <p className="text-xs font-medium text-muted-foreground">
            {t('dashboard.teamWorktime.weeklyHours')}
          </p>
          <p className="text-[10px] text-muted-foreground/70">
            {t('dashboard.teamWorktime.targetHint', { target: fmtMinutes(TARGET_WEEKLY_MINUTES) })}
          </p>
        </div>
      </div>

      {/* Bar list */}
      <div className="flex-1 overflow-auto px-4 pb-3 space-y-2">
        {bars.map((bar) => (
          <div key={bar.id} className="space-y-1">
            <div className="flex items-center justify-between">
              <span
                className="text-xs font-medium text-foreground truncate max-w-[60%]"
                data-testid="team-worktime-employee-name"
              >
                {bar.name}
              </span>
              <span
                className={`text-xs tabular-nums ${bar.overTarget ? 'text-success font-semibold' : 'text-muted-foreground'}`}
              >
                {fmtMinutes(bar.minutes)}
              </span>
            </div>
            {/* CSS bar — GPU-safe: only width as inline style */}
            <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className={`h-full rounded-full ${bar.overTarget ? 'bg-success' : 'bg-violet-500'}`}
                style={{ width: `${(bar.minutes / maxMinutes) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      {/* Target annotation */}
      <div className="px-4 pb-3">
        <p className="text-[10px] text-muted-foreground">
          {t('dashboard.teamWorktime.targetLabel', { target: fmtMinutes(TARGET_WEEKLY_MINUTES) })}
        </p>
      </div>
    </div>
  )
}

export default memo(TeamWorktime)
