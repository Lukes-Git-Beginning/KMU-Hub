/**
 * BudgetSection — Project budget tracking with progress bar visualization.
 *
 * Shows planned budget, actual costs (hours x rate), and remaining budget.
 * Progress bar color codes: green <80%, yellow 80-100%, red >100%.
 * Collapsible section for the project detail header area.
 * Mock data for design — backend swap: real budget + time data from API.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronDown,
  ChevronUp,
  Wallet,
  TrendingUp,
  AlertTriangle,
  Clock,
} from 'lucide-react'
import { cn } from '@/lib'
import { formatCurrency } from '@/lib'
import { useTasks } from '@/api/hooks/useTasks'
import { useWorkSettingsStore } from '@/stores/workSettings'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BudgetSectionProps {
  /** Project whose tasks drive the budget estimate. */
  projectId: string
  /** Project name for display. */
  projectName?: string
}

interface TaskLike {
  estimated_hours?: number
  assignee_name?: string
  status_name?: string
  is_closed?: boolean
  completed_at?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * Budget is derived (no budget field on the backend yet): planned = sum of task
 * effort estimates × hourly rate (work settings); actual = effort of completed
 * tasks × rate. Honest approximation, flagged in the UI. Backend gap tracked.
 */
export default function BudgetSection({
  projectId,
  projectName: _projectName,
}: BudgetSectionProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(true)
  const rate = useWorkSettingsStore((s) => s.defaultHourlyRate)
  const { data } = useTasks({ project_id: projectId, include_completed: true, page_size: 200 })

  const { budget, plannedHours, totalCosts, totalHours, breakdownRows } = useMemo(() => {
    const tasks = (data?.tasks ?? []) as TaskLike[]
    const CLOSED_NAMES = ['erledigt', 'fertig', 'abgeschlossen', 'done', 'closed', 'geschlossen']
    const byPerson = new Map<string, number>()
    let planned = 0
    let doneHrs = 0
    for (const task of tasks) {
      const hours = task.estimated_hours ?? 0
      if (hours <= 0) continue
      planned += hours
      const statusClosed = task.status_name ? CLOSED_NAMES.includes(task.status_name.toLowerCase()) : false
      const isDone = task.is_closed === true || Boolean(task.completed_at) || statusClosed
      if (isDone) doneHrs += hours
      const person = task.assignee_name || t('work.tasks.unassigned')
      byPerson.set(person, (byPerson.get(person) ?? 0) + hours)
    }
    const rows = [...byPerson.entries()]
      .map(([person, hours]) => ({ person, hours, rate, total: hours * rate }))
      .sort((a, b) => b.hours - a.hours)
    return {
      budget: planned * rate,
      plannedHours: planned,
      totalCosts: doneHrs * rate,
      totalHours: doneHrs,
      breakdownRows: rows,
    }
  }, [data?.tasks, rate, t])

  const remaining = budget - totalCosts
  const percentage = budget > 0 ? (totalCosts / budget) * 100 : 0
  const isOverBudget = percentage > 100
  const isWarning = percentage >= 80 && percentage <= 100

  // Progress bar color
  const barColorClass = isOverBudget
    ? 'bg-destructive'
    : isWarning
      ? 'bg-warning'
      : 'bg-success'

  const statusLabel = isOverBudget
    ? t('work.budget.overBudget')
    : isWarning
      ? t('work.budget.warning')
      : t('work.budget.onTrack')

  const statusColorClass = isOverBudget
    ? 'text-destructive'
    : isWarning
      ? 'text-warning-foreground'
      : 'text-success'

  return (
    <div className="border-b border-border bg-card/50">
      {/* Collapsible header */}
      <button
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center justify-between px-6 py-2.5 hover:bg-muted/30 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Wallet className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">
            {t('work.budget.title')}
          </span>
          <span
            className="rounded-full bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground"
            title={t('work.budget.estimatedHint')}
          >
            {t('work.budget.estimated')}
          </span>
          {/* Compact status when collapsed */}
          {!expanded && (
            <span className="flex items-center gap-2 ml-3">
              <span className="text-xs text-muted-foreground">
                {formatCurrency(totalCosts)} / {formatCurrency(budget)}
              </span>
              <span className={cn('text-xs font-medium', statusColorClass)}>
                ({Math.round(percentage)}%)
              </span>
              {isOverBudget && (
                <AlertTriangle className="h-3.5 w-3.5 text-destructive" />
              )}
            </span>
          )}
        </div>
        {expanded ? (
          <ChevronUp className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        )}
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="px-6 pb-4 space-y-4">
          {/* Summary cards */}
          <div className="grid grid-cols-3 gap-4">
            {/* Planned budget */}
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                <Wallet className="h-3.5 w-3.5" />
                {t('work.budget.planned')}
              </div>
              <p className="text-lg font-semibold text-foreground">
                {formatCurrency(budget)}
              </p>
            </div>

            {/* Actual costs */}
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                <TrendingUp className="h-3.5 w-3.5" />
                {t('work.budget.actualCosts')}
              </div>
              <p className="text-lg font-semibold text-foreground">
                {formatCurrency(totalCosts)}
              </p>
              <p className="text-[10px] text-muted-foreground mt-0.5">
                {t('work.budget.hoursRecorded', { hours: totalHours.toFixed(0) })}
              </p>
            </div>

            {/* Remaining */}
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                {isOverBudget ? (
                  <AlertTriangle className="h-3.5 w-3.5 text-destructive" />
                ) : (
                  <Clock className="h-3.5 w-3.5" />
                )}
                {t('work.budget.remaining')}
              </div>
              <p
                className={cn(
                  'text-lg font-semibold',
                  isOverBudget ? 'text-destructive' : 'text-foreground'
                )}
              >
                {formatCurrency(Math.abs(remaining))}
                {isOverBudget && (
                  <span className="text-xs font-normal ml-1">{t('work.budget.overBudget')}</span>
                )}
              </p>
            </div>
          </div>

          {/* Progress bar */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">
                {t('work.budget.utilization')}
              </span>
              <span className={cn('font-medium', statusColorClass)}>
                {statusLabel} — {Math.round(percentage)}%
              </span>
            </div>
            <div className="h-2.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className={cn('h-full rounded-full transition-all', barColorClass)}
                style={{ width: `${Math.min(percentage, 100)}%` }}
              />
            </div>
            {isOverBudget && (
              <div className="flex items-center gap-1.5 text-xs text-destructive mt-1">
                <AlertTriangle className="h-3 w-3" />
                {t('work.budget.exceededBy', { amount: formatCurrency(Math.abs(remaining)), percent: Math.round(percentage - 100) })}
              </div>
            )}
          </div>

          {/* Cost breakdown by person */}
          <div className="rounded-md border border-border overflow-hidden">
            <div className="grid grid-cols-[1fr_70px_80px_90px] gap-2 items-center bg-secondary/50 px-3 py-1.5 text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
              <span>{t('work.utilization.employee')}</span>
              <span className="text-right">{t('work.budget.hours')}</span>
              <span className="text-right">{t('work.budget.hourlyRate')}</span>
              <span className="text-right">{t('work.budget.costs')}</span>
            </div>
            {breakdownRows.map((row) => (
              <div
                key={row.person}
                className="grid grid-cols-[1fr_70px_80px_90px] gap-2 items-center px-3 py-2 border-t border-border"
              >
                <span className="text-xs text-foreground">{row.person}</span>
                <span className="text-xs text-muted-foreground text-right font-mono">
                  {row.hours.toFixed(0)}h
                </span>
                <span className="text-xs text-muted-foreground text-right">
                  {formatCurrency(row.rate)}/h
                </span>
                <span className="text-xs font-medium text-foreground text-right">
                  {formatCurrency(row.total)}
                </span>
              </div>
            ))}
            {/* Total row — planned effort, consistent with the per-person rows */}
            <div className="grid grid-cols-[1fr_70px_80px_90px] gap-2 items-center px-3 py-2 border-t-2 border-border bg-secondary/30">
              <span className="text-xs font-semibold text-foreground">{t('work.budget.total')}</span>
              <span className="text-xs font-semibold text-foreground text-right font-mono">
                {plannedHours.toFixed(0)}h
              </span>
              <span />
              <span className="text-xs font-semibold text-foreground text-right">
                {formatCurrency(budget)}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
