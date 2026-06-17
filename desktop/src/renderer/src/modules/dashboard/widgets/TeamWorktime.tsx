/**
 * Team Worktime widget — weekly hours per team member as CSS bars.
 *
 * Module: zeiterfassung
 * Data: useTeamWorktime() — per-member weekly minutes from the MSW
 *       /dashboard/team-worktime endpoint (swap-ready, no client-side seeding).
 * CSS bar pattern follows MiniChart.tsx / KpiRevenue.tsx — GPU-safe.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock4, Loader2 } from 'lucide-react'
import { useTeamWorktime } from '@/api/hooks/useTeamWorktime'
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

interface EmployeeBar {
  id: string
  name: string
  minutes: number
  overTarget: boolean
}

function TeamWorktime(_props: WidgetProps) {
  const { t } = useTranslation()
  const { data: members = [], isLoading } = useTeamWorktime()

  const bars: EmployeeBar[] = useMemo(
    () =>
      members.map((m) => ({
        id: m.id,
        name: m.name || t('dashboard.teamWorktime.unknownEmployee'),
        minutes: m.minutes,
        overTarget: m.minutes > TARGET_WEEKLY_MINUTES,
      })),
    [members, t],
  )

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
