/**
 * Team time view (manager) — weekly hours per member + week-approval workflow.
 *
 * Visible only to a zeiterfassung Modul-Leiter / admin (gated in the tab).
 * Mock-first HR API; approve/reject mutate the demo state in place.
 */
import { useTranslation } from 'react-i18next'
import { Loader2, Check, X, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTeamTime, useApproveWeek, useRejectWeek } from '@/api/hooks/hr-hooks'
import { EmptyState } from '@/components/shared'
import { formatWorkMinutes } from '@/lib/worktime'
import { currentWeekStart } from './weekUtils'
import type { WeekStatus } from '@/api/hr-types'

const STATUS_TONE: Record<WeekStatus, string> = {
  open: 'bg-secondary text-muted-foreground',
  submitted: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-destructive/10 text-destructive',
}

export function TeamView({ canApproveWeek = false }: { canApproveWeek?: boolean }) {
  const { t } = useTranslation()
  const weekStart = currentWeekStart()
  const { data: rows, isLoading } = useTeamTime(weekStart)
  const approve = useApproveWeek()
  const reject = useRejectWeek()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    )
  }

  const members = rows ?? []
  if (members.length === 0) {
    return (
      <EmptyState
        icon={AlertTriangle}
        title={t('zeiterfassung.team.empty')}
        description={t('zeiterfassung.team.emptyDesc')}
      />
    )
  }

  const pending = members.filter((m) => m.weekStatus === 'submitted').length
  const busy = approve.isPending || reject.isPending

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">{t('zeiterfassung.team.title')}</h3>
        {pending > 0 && (
          <span className="rounded-full bg-warning-light px-2.5 py-0.5 text-xs font-medium text-warning">
            {t('zeiterfassung.team.pending', { count: pending })}
          </span>
        )}
      </div>

      <div className="overflow-hidden rounded-xl border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-secondary/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('zeiterfassung.team.member')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('zeiterfassung.team.week')}</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('zeiterfassung.team.overtime')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('zeiterfassung.team.status')}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {members.map((m) => {
              const pct = Math.min(100, Math.round((m.weekMinutes / m.targetMinutes) * 100))
              return (
                <tr key={m.employeeId} className="border-b border-border-muted last:border-0">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {m.clockedIn && <span className="h-2 w-2 shrink-0 rounded-full bg-success" title={t('zeiterfassung.team.clockedIn')} />}
                      <span className="font-medium text-foreground">{m.name}</span>
                    </div>
                    {m.department && <span className="text-xs text-muted-foreground">{m.department}</span>}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-secondary">
                        <div className={cn('h-full rounded-full', pct >= 100 ? 'bg-success' : 'bg-primary')} style={{ width: `${pct}%` }} />
                      </div>
                      <span className="tabular-nums text-foreground">{formatWorkMinutes(m.weekMinutes)}</span>
                    </div>
                  </td>
                  <td className={cn('px-4 py-3 text-right tabular-nums', m.overtimeMinutes > 0 ? 'text-warning' : 'text-muted-foreground')}>
                    {m.overtimeMinutes > 0 ? `+${formatWorkMinutes(m.overtimeMinutes)}` : '–'}
                  </td>
                  <td className="px-4 py-3">
                    <span className={cn('rounded-full px-2 py-0.5 text-[11px] font-medium', STATUS_TONE[m.weekStatus])}>
                      {t(`zeiterfassung.team.weekStatus.${m.weekStatus}`)}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {canApproveWeek && m.weekStatus === 'submitted' && (
                      <div className="flex justify-end gap-1.5">
                        <button
                          onClick={() => approve.mutate({ employeeId: m.employeeId, weekStart })}
                          disabled={busy}
                          className="flex items-center gap-1 rounded-md bg-success/10 px-2 py-1 text-xs font-medium text-success transition-colors hover:bg-success/20"
                        >
                          <Check className="h-3.5 w-3.5" />
                          {t('zeiterfassung.team.approve')}
                        </button>
                        <button
                          onClick={() => reject.mutate({ employeeId: m.employeeId, weekStart, reason: '' })}
                          disabled={busy}
                          className="flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-1 text-xs font-medium text-destructive transition-colors hover:bg-destructive/20"
                        >
                          <X className="h-3.5 w-3.5" />
                          {t('zeiterfassung.team.reject')}
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
