/**
 * BudgetSection — Project budget tracking with progress bar visualization.
 *
 * Shows planned budget, actual costs (hours x rate), and remaining budget.
 * Progress bar color codes: green <80%, yellow 80-100%, red >100%.
 * Collapsible section for the project detail header area.
 * Mock data for design — backend swap: real budget + time data from API.
 */
import { useState, useMemo } from 'react'
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

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

interface MockTimeData {
  person: string
  hours: number
  rate: number
}

// TODO: Replace MOCK_TIME_DATA with API call — Backend needed: GET /api/v1/projects/{id}/time-entries?grouped_by=person
// This mock provides per-person time entry data for the budget/hours breakdown.
const MOCK_TIME_DATA: MockTimeData[] = [
  { person: 'Anna Mueller', hours: 64, rate: 150 },
  { person: 'Thomas Fischer', hours: 48, rate: 140 },
  { person: 'Max Schmidt', hours: 32, rate: 130 },
  { person: 'Sara Weber', hours: 24, rate: 145 },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BudgetSectionProps {
  /** Project's planned budget. Falls back to 50000 if not set. */
  budget?: number
  /** Project name for display. */
  projectName?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function BudgetSection({
  budget = 50000,
  projectName: _projectName,
}: BudgetSectionProps) {
  const [expanded, setExpanded] = useState(true)

  // Calculate actual costs from mock time entries
  const { totalCosts, totalHours, breakdownRows } = useMemo(() => {
    const rows = MOCK_TIME_DATA.map((entry) => ({
      ...entry,
      total: entry.hours * entry.rate,
    }))
    const costs = rows.reduce((sum, r) => sum + r.total, 0)
    const hours = rows.reduce((sum, r) => sum + r.hours, 0)
    return { totalCosts: costs, totalHours: hours, breakdownRows: rows }
  }, [])

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
    ? 'Ueber Budget'
    : isWarning
      ? 'Warnung'
      : 'Im Rahmen'

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
            Projektbudget
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
                Geplantes Budget
              </div>
              <p className="text-lg font-semibold text-foreground">
                {formatCurrency(budget)}
              </p>
            </div>

            {/* Actual costs */}
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
                <TrendingUp className="h-3.5 w-3.5" />
                Tatsaechliche Kosten
              </div>
              <p className="text-lg font-semibold text-foreground">
                {formatCurrency(totalCosts)}
              </p>
              <p className="text-[10px] text-muted-foreground mt-0.5">
                {totalHours.toFixed(0)}h erfasst
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
                Verbleibend
              </div>
              <p
                className={cn(
                  'text-lg font-semibold',
                  isOverBudget ? 'text-destructive' : 'text-foreground'
                )}
              >
                {formatCurrency(Math.abs(remaining))}
                {isOverBudget && (
                  <span className="text-xs font-normal ml-1">ueber Budget</span>
                )}
              </p>
            </div>
          </div>

          {/* Progress bar */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">
                Budgetauslastung
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
                Budget um {formatCurrency(Math.abs(remaining))} ueberschritten (
                {Math.round(percentage - 100)}% ueber Plan)
              </div>
            )}
          </div>

          {/* Cost breakdown by person */}
          <div className="rounded-md border border-border overflow-hidden">
            <div className="grid grid-cols-[1fr_70px_80px_90px] gap-2 items-center bg-secondary/50 px-3 py-1.5 text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
              <span>Mitarbeiter</span>
              <span className="text-right">Stunden</span>
              <span className="text-right">Stundensatz</span>
              <span className="text-right">Kosten</span>
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
            {/* Total row */}
            <div className="grid grid-cols-[1fr_70px_80px_90px] gap-2 items-center px-3 py-2 border-t-2 border-border bg-secondary/30">
              <span className="text-xs font-semibold text-foreground">Gesamt</span>
              <span className="text-xs font-semibold text-foreground text-right font-mono">
                {totalHours.toFixed(0)}h
              </span>
              <span />
              <span className="text-xs font-semibold text-foreground text-right">
                {formatCurrency(totalCosts)}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
