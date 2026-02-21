/**
 * AuslastungReport — Team utilization visualization for project detail.
 *
 * Bar chart showing hours per team member per week/month.
 * Color coding: green <80% target, yellow 80-100%, red >100%.
 * Overload warnings displayed below chart.
 * Mock data for design — backend swap: real time tracking data from API.
 */
import { useState, useMemo } from 'react'
import {
  BarChart3,
  CalendarDays,
  CalendarRange,
  AlertTriangle,
  User,
  Clock,
  TrendingUp,
} from 'lucide-react'
import { cn } from '@/lib'
import { formatCurrency } from '@/lib'
import { Button } from '@/components/ui/button'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ViewMode = 'week' | 'month'

interface TeamMember {
  id: string
  name: string
  role: string
  avatarInitial: string
  /** Target hours per week (standard: 40) */
  weeklyTarget: number
  /** Hourly rate */
  rate: number
}

interface WeeklyHours {
  /** ISO week label, e.g., "KW 7" */
  label: string
  hours: number
}

interface MonthlyHours {
  label: string
  hours: number
}

interface MemberUtilization {
  member: TeamMember
  weeklyData: WeeklyHours[]
  monthlyData: MonthlyHours[]
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const MOCK_TEAM: TeamMember[] = [
  { id: 'm1', name: 'Anna Mueller', role: 'Frontend Lead', avatarInitial: 'A', weeklyTarget: 40, rate: 150 },
  { id: 'm2', name: 'Thomas Fischer', role: 'Backend Dev', avatarInitial: 'T', weeklyTarget: 40, rate: 140 },
  { id: 'm3', name: 'Max Schmidt', role: 'DevOps', avatarInitial: 'M', weeklyTarget: 32, rate: 130 },
  { id: 'm4', name: 'Sara Weber', role: 'UI/UX Design', avatarInitial: 'S', weeklyTarget: 40, rate: 145 },
  { id: 'm5', name: 'Lukas Braun', role: 'QA Engineer', avatarInitial: 'L', weeklyTarget: 40, rate: 120 },
  { id: 'm6', name: 'Julia Hoffmann', role: 'Projektleitung', avatarInitial: 'J', weeklyTarget: 24, rate: 160 },
]

// Generate mock weekly data (last 6 weeks)
function generateWeeklyData(target: number): WeeklyHours[] {
  const weeks = ['KW 4', 'KW 5', 'KW 6', 'KW 7', 'KW 8', 'KW 9']
  return weeks.map((label) => ({
    label,
    // Vary hours around the target with some randomness (seeded by label hash)
    hours: Math.round(
      target * (0.6 + Math.abs(Math.sin(label.charCodeAt(3) * 2.7)) * 0.8) * 10
    ) / 10,
  }))
}

function generateMonthlyData(target: number): MonthlyHours[] {
  const months = ['Dez 2025', 'Jan 2026', 'Feb 2026']
  const weekMultiplier = 4.33 // avg weeks per month
  return months.map((label) => ({
    label,
    hours: Math.round(
      target * weekMultiplier * (0.65 + Math.abs(Math.sin(label.charCodeAt(0) * 3.1)) * 0.7) * 10
    ) / 10,
  }))
}

const MOCK_UTILIZATION: MemberUtilization[] = MOCK_TEAM.map((member) => ({
  member,
  weeklyData: generateWeeklyData(member.weeklyTarget),
  monthlyData: generateMonthlyData(member.weeklyTarget),
}))

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
  if (pct > 100) return 'bg-red-500'
  if (pct >= 80) return 'bg-yellow-500'
  return 'bg-emerald-500'
}

function getUtilTextColor(pct: number): string {
  if (pct > 100) return 'text-red-500'
  if (pct >= 80) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function AuslastungReport({ projectId }: AuslastungReportProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('week')

  // Calculate summary stats
  const { overloaded, avgUtilization, totalCost, periods } = useMemo(() => {
    const overloadedMembers: Array<{ name: string; pct: number; period: string }> = []
    let totalPct = 0
    let totalPctCount = 0
    let cost = 0

    // Get the period labels based on view mode
    const periodLabels = viewMode === 'week'
      ? MOCK_UTILIZATION[0]?.weeklyData.map((d) => d.label) ?? []
      : MOCK_UTILIZATION[0]?.monthlyData.map((d) => d.label) ?? []

    for (const util of MOCK_UTILIZATION) {
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
  }, [viewMode])

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">
            Teamauslastung
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
            Woche
          </Button>
          <Button
            variant={viewMode === 'month' ? 'secondary' : 'ghost'}
            size="sm"
            className="rounded-l-none gap-1.5"
            onClick={() => setViewMode('month')}
          >
            <CalendarRange className="h-3.5 w-3.5" />
            Monat
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-auto p-4 space-y-6">
        {/* Summary cards */}
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-border bg-card p-3">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
              <User className="h-3.5 w-3.5" />
              Teammitglieder
            </div>
            <p className="text-lg font-semibold text-foreground">
              {MOCK_TEAM.length}
            </p>
          </div>

          <div className="rounded-lg border border-border bg-card p-3">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
              <TrendingUp className="h-3.5 w-3.5" />
              Durchschn. Auslastung
            </div>
            <p className={cn('text-lg font-semibold', getUtilTextColor(avgUtilization))}>
              {avgUtilization}%
            </p>
          </div>

          <div className="rounded-lg border border-border bg-card p-3">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
              <Clock className="h-3.5 w-3.5" />
              Personalkosten (Zeitraum)
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
              Mitarbeiter
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
          {MOCK_UTILIZATION.map((util) => {
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
                            className="absolute top-0 right-0 h-full bg-red-500/20 border-l-2 border-red-500"
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
              <div className="h-3 w-5 rounded bg-emerald-500" />
              <span>&lt;80% (Kapazitaet frei)</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="h-3 w-5 rounded bg-yellow-500" />
              <span>80-100% (Gut ausgelastet)</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="h-3 w-5 rounded bg-red-500" />
              <span>&gt;100% (Ueberlastet)</span>
            </div>
          </div>
        </div>

        {/* Overload warnings */}
        {overloaded.length > 0 && (
          <div className="rounded-lg border border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/20 p-4 space-y-2">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-red-500" />
              <span className="text-sm font-medium text-red-700 dark:text-red-400">
                Ueberlastungswarnungen
              </span>
            </div>
            <div className="space-y-1">
              {overloaded.slice(0, 8).map((warn, i) => (
                <div key={i} className="flex items-center gap-2 text-xs text-red-600 dark:text-red-400">
                  <span className="font-medium">{warn.name}</span>
                  <span className="text-red-400 dark:text-red-500">—</span>
                  <span>{warn.period}: {warn.pct}% Auslastung</span>
                </div>
              ))}
              {overloaded.length > 8 && (
                <p className="text-xs text-red-400">
                  ... und {overloaded.length - 8} weitere
                </p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
