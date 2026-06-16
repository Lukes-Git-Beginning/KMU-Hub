/**
 * AuslastungReport — Team utilization visualization for project detail.
 *
 * Bar chart showing hours per team member per week/month.
 * Color coding: green <80% target, yellow 80-100%, red >100%.
 * Overload warnings displayed below chart.
 * Data: MSW (GET /projects/:id/team-utilization via useProjectTeamUtilization),
 * swap-ready for the real Zeiterfassung capacity endpoint.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart3,
  CalendarDays,
  CalendarRange,
  AlertTriangle,
  User,
  Clock,
  TrendingUp,
  Loader2,
} from 'lucide-react'
import { cn } from '@/lib'
import { formatCurrency } from '@/lib'
import { Button } from '@/components/ui/button'
import { useProjectTeamUtilization } from '@/api/hooks/useProjects'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ViewMode = 'week' | 'month'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AuslastungReportProps {
  projectId: string
}

// ---------------------------------------------------------------------------
// Helper: percentage color
// ---------------------------------------------------------------------------

function getUtilColor(pct: number): string {
  if (pct > 100) return 'bg-destructive'
  if (pct >= 80) return 'bg-warning'
  return 'bg-success'
}

function getUtilTextColor(pct: number): string {
  if (pct > 100) return 'text-destructive'
  if (pct >= 80) return 'text-warning-foreground'
  return 'text-success'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function AuslastungReport({ projectId }: AuslastungReportProps) {
  const { t } = useTranslation()
  const [viewMode, setViewMode] = useState<ViewMode>('week')
  const { data: utilization = [], isLoading } = useProjectTeamUtilization(projectId)

  // Calculate summary stats
  const { overloaded, avgUtilization, totalCost, periods } = useMemo(() => {
    const overloadedMembers: Array<{ name: string; pct: number; period: string }> = []
    let totalPct = 0
    let totalPctCount = 0
    let cost = 0

    // Get the period labels based on view mode
    const periodLabels = viewMode === 'week'
      ? utilization[0]?.weeklyData.map((d) => d.label) ?? []
      : utilization[0]?.monthlyData.map((d) => d.label) ?? []

    for (const util of utilization) {
      const data = viewMode === 'week' ? util.weeklyData : util.monthlyData
      const target = viewMode === 'week'
        ? util.member.weeklyTarget
        : util.member.weeklyTarget * 4.33

      for (const period of data) {
        const pct = target > 0 ? (period.hours / target) * 100 : 0
        totalPct += pct
        totalPctCount++
        cost += period.hours * util.member.rate

        if (pct > 100) {
          overloadedMembers.push({
            name: util.member.name,
            pct: Math.round(pct),
            period: period.label,
          })
        }
      }
    }

    return {
      overloaded: overloadedMembers,
      avgUtilization: totalPctCount > 0 ? Math.round(totalPct / totalPctCount) : 0,
      totalCost: cost,
      periods: periodLabels,
    }
  }, [viewMode, utilization])

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">
            {t('work.utilization.title')}
          </span>
        </div>

        {/* View toggle */}
        <div className="flex items-center gap-1 rounded-md border border-border">
          <Button
            variant={viewMode === 'week' ? 'secondary' : 'ghost'}
            size="sm"
            className="rounded-r-none gap-1.5"
            onClick={() => setViewMode('week')}
          >
            <CalendarDays className="h-3.5 w-3.5" />
            {t('work.utilization.week')}
          </Button>
          <Button
            variant={viewMode === 'month' ? 'secondary' : 'ghost'}
            size="sm"
            className="rounded-l-none gap-1.5"
            onClick={() => setViewMode('month')}
          >
            <CalendarRange className="h-3.5 w-3.5" />
            {t('work.utilization.month')}
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-auto p-4 space-y-6">
        {/* Preview notice — capacity/utilization aggregates are not yet provided
            by the backend; this is a design preview with sample data. */}
        <div className="flex items-start gap-2.5 rounded-lg border border-border bg-secondary/40 px-4 py-2.5">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
          <p className="text-xs leading-relaxed text-muted-foreground">{t('work.utilization.previewBanner')}</p>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin" />
          </div>
        ) : utilization.length === 0 ? (
          <div className="py-16 text-center text-sm text-muted-foreground">
            {t('work.utilization.noData')}
          </div>
        ) : (
          <>
            {/* Summary cards */}
            <div className="grid grid-cols-3 gap-4">
              <div className="rounded-lg border border-border bg-card p-3">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                  <User className="h-3.5 w-3.5" />
                  {t('work.utilization.teamMembers')}
                </div>
                <p className="text-lg font-semibold text-foreground">
                  {utilization.length}
                </p>
              </div>

              <div className="rounded-lg border border-border bg-card p-3">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                  <TrendingUp className="h-3.5 w-3.5" />
                  {t('work.utilization.avgUtilization')}
                </div>
                <p className={cn('text-lg font-semibold', getUtilTextColor(avgUtilization))}>
                  {avgUtilization}%
                </p>
              </div>

              <div className="rounded-lg border border-border bg-card p-3">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                  <Clock className="h-3.5 w-3.5" />
                  {t('work.utilization.personnelCosts')}
                </div>
                <p className="text-lg font-semibold text-foreground">
                  {formatCurrency(totalCost)}
                </p>
              </div>
            </div>

            {/* Utilization chart (horizontal bar chart per member) */}
            <div className="space-y-1">
              {/* Period headers */}
              <div className="grid items-center gap-2" style={{ gridTemplateColumns: `180px repeat(${periods.length}, 1fr)` }}>
                <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
                  {t('work.utilization.employee')}
                </span>
                {periods.map((period) => (
                  <span
                    key={period}
                    className="text-[10px] font-medium text-muted-foreground text-center"
                  >
                    {period}
                  </span>
                ))}
              </div>

              {/* Member rows */}
              {utilization.map((util) => {
                const data = viewMode === 'week' ? util.weeklyData : util.monthlyData
                const target = viewMode === 'week'
                  ? util.member.weeklyTarget
                  : util.member.weeklyTarget * 4.33

                return (
                  <div
                    key={util.member.id}
                    className="grid items-center gap-2 py-1.5 border-t border-border"
                    style={{ gridTemplateColumns: `180px repeat(${periods.length}, 1fr)` }}
                  >
                    {/* Member info */}
                    <div className="flex items-center gap-2 min-w-0">
                      <div className="flex-shrink-0 flex items-center justify-center rounded-full bg-muted text-[10px] font-medium w-6 h-6">
                        {util.member.avatarInitial}
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-foreground truncate">
                          {util.member.name}
                        </p>
                        <p className="text-[10px] text-muted-foreground truncate">
                          {util.member.role}
                        </p>
                      </div>
                    </div>

                    {/* Period bars */}
                    {data.map((period) => {
                      const pct = target > 0 ? (period.hours / target) * 100 : 0
                      const barWidth = Math.min(pct, 120) // cap visual at 120%
                      const barColor = getUtilColor(pct)

                      return (
                        <div key={period.label} className="flex flex-col items-center gap-0.5">
                          {/* Bar container */}
                          <div className="w-full h-6 bg-muted/40 rounded relative overflow-hidden">
                            <div
                              className={cn('h-full rounded transition-all', barColor)}
                              style={{ width: `${Math.min(barWidth, 100)}%` }}
                            />
                            {/* Overflow stripe for >100% */}
                            {pct > 100 && (
                              <div
                                className="absolute top-0 right-0 h-full bg-destructive/20 border-l-2 border-destructive"
                                style={{ width: `${Math.min(pct - 100, 20)}%` }}
                              />
                            )}
                            {/* Hours label inside bar */}
                            <span className="absolute inset-0 flex items-center justify-center text-[9px] font-medium text-white mix-blend-difference">
                              {period.hours.toFixed(0)}h
                            </span>
                          </div>
                          {/* Percentage */}
                          <span className={cn('text-[9px] font-mono', getUtilTextColor(pct))}>
                            {Math.round(pct)}%
                          </span>
                        </div>
                      )
                    })}
                  </div>
                )
              })}

              {/* Target line legend */}
              <div className="flex items-center gap-4 pt-3 text-[10px] text-muted-foreground border-t border-border mt-2">
                <div className="flex items-center gap-1.5">
                  <div className="h-3 w-5 rounded bg-success" />
                  <span>{t('work.utilization.legendFree')}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="h-3 w-5 rounded bg-warning" />
                  <span>{t('work.utilization.legendGood')}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="h-3 w-5 rounded bg-destructive" />
                  <span>{t('work.utilization.legendOverloaded')}</span>
                </div>
              </div>
            </div>

            {/* Overload warnings */}
            {overloaded.length > 0 && (
              <div className="rounded-lg border border-destructive/30 bg-error-light p-4 space-y-2">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-destructive" />
                  <span className="text-sm font-medium text-destructive">
                    {t('work.utilization.overloadWarnings')}
                  </span>
                </div>
                <div className="space-y-1">
                  {overloaded.slice(0, 8).map((warn, i) => (
                    <div key={i} className="flex items-center gap-2 text-xs text-destructive">
                      <span className="font-medium">{warn.name}</span>
                      <span className="text-destructive">—</span>
                      <span>{warn.period}: {warn.pct}% {t('work.utilization.utilization')}</span>
                    </div>
                  ))}
                  {overloaded.length > 8 && (
                    <p className="text-xs text-destructive">
                      {t('work.utilization.andMore', { count: overloaded.length - 8 })}
                    </p>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
