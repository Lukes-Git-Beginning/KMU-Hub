import { useState, useMemo, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Play, Square, Clock, Calendar,
  Coffee, AlertTriangle, Loader2, Edit3, Plus, Receipt, BarChart3, Users,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/cn'
import { formatDate } from '@/lib/format'
import { toast } from 'sonner'
import { EmptyState } from '@/components/shared'
import { EmptyCalendar } from '@/components/shared/illustrations'
import {
  useWorkTimeStatus,
  useWorkTimeEntries,
  useDailySummary,
  useWeeklySummary,
  useClockIn,
  useClockOut,
  useStartBreak,
  useEndBreak,
  useSubmitCorrection,
  useApproveCorrection,
} from '@/api/hooks/hr-hooks'
import { useAuthStore } from '@/stores/auth'
import { useZeiterfassungPrefsStore } from '@/stores/zeiterfassungPrefs'
import { STANDARD_DAILY_HOURS } from '@/lib/worktime'
import { useIsModuleLead } from '@/hooks/useModuleSettings'
import { ManualEntryDialog } from '@/modules/zeiterfassung/components/ManualEntryDialog'
import { AuswertungenView } from '@/modules/zeiterfassung/components/AuswertungenView'
import { TeamView } from '@/modules/zeiterfassung/components/TeamView'
import { WeekSubmitBanner } from '@/modules/zeiterfassung/components/WeekSubmitBanner'
import type { WorkTimeEntry, DailySummary as DailySummaryType } from '@/api/hr-types'

type ViewKey = 'today' | 'week' | 'analytics' | 'team' | 'corrections'

function formatMinutesDisplay(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${h}h ${m.toString().padStart(2, '0')}m`
}

function formatTimeFromISO(iso: string): string {
  const d = new Date(iso)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

function getArbZGColor(severity: string): string {
  switch (severity) {
    case 'info': return 'text-info'
    case 'warning': return 'text-warning'
    case 'error': return 'text-error'
    default: return 'text-muted-foreground'
  }
}

export default function ZeiterfassungTab() {
  const { t } = useTranslation()
  const isLead = useIsModuleLead('zeiterfassung')

  const VIEWS: { key: ViewKey; label: string; icon: typeof Clock }[] = [
    { key: 'today', label: t('profil.zeiterfassung.viewToday'), icon: Clock },
    { key: 'week', label: t('profil.zeiterfassung.viewWeek'), icon: Calendar },
    { key: 'analytics', label: t('zeiterfassung.analytics.tab'), icon: BarChart3 },
    ...(isLead ? [{ key: 'team' as const, label: t('zeiterfassung.team.tab'), icon: Users }] : []),
    { key: 'corrections', label: t('profil.zeiterfassung.viewCorrections'), icon: Edit3 },
  ]

  function getArbZGLabel(severity: string): string {
    switch (severity) {
      case 'info': return t('profil.zeiterfassung.arbzg8h')
      case 'warning': return t('profil.zeiterfassung.arbzg9h')
      case 'error': return t('profil.zeiterfassung.arbzg10h')
      default: return ''
    }
  }
  const defaultView = useZeiterfassungPrefsStore((s) => s.defaultView)
  const [activeView, setActiveView] = useState<ViewKey>(defaultView)
  const [showManualEntry, setShowManualEntry] = useState(false)
  const [showCorrectionDialog, setShowCorrectionDialog] = useState(false)
  const [correctionEntryId, setCorrectionEntryId] = useState('')
  const [correctionClockIn, setCorrectionClockIn] = useState('')
  const [correctionClockOut, setCorrectionClockOut] = useState('')
  const [correctionBreak, setCorrectionBreak] = useState(0)
  const [correctionReason, setCorrectionReason] = useState('')

  const user = useAuthStore((s) => s.user)

  // HR API hooks
  const { data: status } = useWorkTimeStatus()
  const clockInMutation = useClockIn()
  const clockOutMutation = useClockOut()
  const startBreakMutation = useStartBreak()
  const endBreakMutation = useEndBreak()
  const correctionMutation = useSubmitCorrection()
  const approveCorrectionMutation = useApproveCorrection()

  // Today's date for daily summary
  const todayStr = useMemo(() => {
    const d = new Date()
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }, [])

  // Week start (Monday)
  const weekStartStr = useMemo(() => {
    const d = new Date()
    const day = d.getDay()
    const diff = d.getDate() - day + (day === 0 ? -6 : 1)
    const monday = new Date(d.setDate(diff))
    return `${monday.getFullYear()}-${String(monday.getMonth() + 1).padStart(2, '0')}-${String(monday.getDate()).padStart(2, '0')}`
  }, [])

  const { data: dailySummary } = useDailySummary(todayStr)
  const { data: weeklySummary } = useWeeklySummary(weekStartStr)
  const { data: entriesData } = useWorkTimeEntries(
    user ? { employee_id: user.id } : undefined,
  )
  const { data: pendingCorrections } = useWorkTimeEntries({ status: 'correction_pending' })

  // Live timer display using requestAnimationFrame
  const [elapsedDisplay, setElapsedDisplay] = useState('00:00:00')
  const rafRef = useRef<number>(0)

  const updateTimer = useCallback(() => {
    if (status?.isClockedIn && status.currentShiftStart && !status.isOnBreak) {
      const start = new Date(status.currentShiftStart).getTime()
      const now = Date.now()
      const totalSeconds = Math.floor((now - start) / 1000)
      const hours = Math.floor(totalSeconds / 3600)
      const minutes = Math.floor((totalSeconds % 3600) / 60)
      const seconds = totalSeconds % 60
      setElapsedDisplay(
        `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`,
      )
    }
    // eslint-disable-next-line react-hooks/immutability -- ref assigned in useCallback for rAF loop
    rafRef.current = requestAnimationFrame(updateTimer)
  }, [status?.isClockedIn, status?.currentShiftStart, status?.isOnBreak])

   
  useEffect(() => {
    if (status?.isClockedIn) {
      rafRef.current = requestAnimationFrame(updateTimer)
    } else {
      setElapsedDisplay('00:00:00')
    }
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current)
    }
  }, [status?.isClockedIn, updateTimer])

  const todayMinutes = dailySummary?.netWorkMinutes ?? status?.todayTotalMinutes ?? 0
  const targetMinutes = STANDARD_DAILY_HOURS * 60 // contractual daily norm
  const overtimeMinutes = Math.max(0, todayMinutes - targetMinutes)
  const progressPercent = Math.min(100, Math.round((todayMinutes / targetMinutes) * 100))

  const entries = entriesData?.entries ?? []

  const handleClockIn = () => clockInMutation.mutate()
  const handleClockOut = () => clockOutMutation.mutate()
  const handleStartBreak = () => startBreakMutation.mutate()
  const handleEndBreak = () => endBreakMutation.mutate()

  const handleSubmitCorrection = () => {
    if (!correctionEntryId || !correctionClockIn || !correctionClockOut || !correctionReason.trim()) {
      toast.error(t('profil.zeiterfassung.errorFillAllFields'))
      return
    }
    correctionMutation.mutate(
      {
        originalEntryId: correctionEntryId,
        correctedClockIn: correctionClockIn,
        correctedClockOut: correctionClockOut,
        correctedBreakMinutes: correctionBreak,
        reason: correctionReason.trim(),
      },
      {
        onSuccess: () => {
          setShowCorrectionDialog(false)
          setCorrectionEntryId('')
          setCorrectionClockIn('')
          setCorrectionClockOut('')
          setCorrectionBreak(0)
          setCorrectionReason('')
        },
      },
    )
  }

  const isMutating = clockInMutation.isPending || clockOutMutation.isPending || startBreakMutation.isPending || endBreakMutation.isPending

  return (
    <div className="h-full flex flex-col">
      {/* Timer Toolbar */}
      <div className="border-b border-border bg-card/50 px-6 py-4">
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4 max-w-4xl mx-auto">
          {/* Clock In/Out Controls */}
          <div className="flex items-center gap-2">
            {!status?.isClockedIn ? (
              <Button size="sm" onClick={handleClockIn} disabled={isMutating} className="gap-2">
                <Play className="h-4 w-4" />
                {t('profil.zeiterfassung.clockIn')}
              </Button>
            ) : status.isOnBreak ? (
              <Button size="sm" onClick={handleEndBreak} disabled={isMutating} className="gap-2">
                <Coffee className="h-4 w-4" />
                {t('profil.zeiterfassung.endBreak')}
              </Button>
            ) : (
              <>
                <Button size="sm" variant="outline" onClick={handleStartBreak} disabled={isMutating} className="gap-2">
                  <Coffee className="h-4 w-4" />
                  {t('profil.zeiterfassung.break')}
                </Button>
                <Button size="sm" variant="destructive" onClick={handleClockOut} disabled={isMutating} className="gap-2">
                  <Square className="h-4 w-4" />
                  {t('profil.zeiterfassung.clockOut')}
                </Button>
              </>
            )}

            {status?.isClockedIn && (
              <span className="text-lg font-mono font-bold text-primary tabular-nums">
                {status.isOnBreak ? t('profil.zeiterfassung.break') : elapsedDisplay}
              </span>
            )}
          </div>

          {/* ArbZG indicator */}
          {status?.arbzgSeverity && status.arbzgSeverity !== 'none' && (
            <div className={cn('flex items-center gap-1.5 text-xs font-medium', getArbZGColor(status.arbzgSeverity))}>
              <AlertTriangle className="h-3.5 w-3.5" />
              <span>{getArbZGLabel(status.arbzgSeverity)}</span>
            </div>
          )}

          {/* Daily Summary */}
          <div className="ml-auto flex items-center gap-3 text-sm">
            <span className="text-muted-foreground">{t('profil.zeiterfassung.viewToday')}:</span>
            <span className="font-semibold text-foreground">
              {formatMinutesDisplay(todayMinutes)}
            </span>
            {overtimeMinutes > 0 && (
              <span className="text-xs text-warning font-medium">
                (+{formatMinutesDisplay(overtimeMinutes)} {t('profil.zeiterfassung.overtime')})
              </span>
            )}
            <div className="w-24 h-2 rounded-full bg-secondary overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  progressPercent >= 100 ? 'bg-warning' : progressPercent >= 80 ? 'bg-success' : 'bg-primary',
                )}
                style={{ width: `${progressPercent}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground">{progressPercent}%</span>
          </div>
        </div>
      </div>

      {/* View Switcher */}
      <div className="flex items-center justify-between border-b border-border bg-card/30 px-6">
        <nav className="flex gap-1" role="tablist">
          {VIEWS.map((view) => {
            const Icon = view.icon
            return (
              <button
                key={view.key}
                role="tab"
                aria-selected={activeView === view.key}
                onClick={() => setActiveView(view.key)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 transition-colors',
                  activeView === view.key
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground',
                )}
              >
                <Icon className="h-3.5 w-3.5" />
                {view.label}
                {view.key === 'corrections' && (pendingCorrections?.entries?.length ?? 0) > 0 && (
                  <span className="ml-1 rounded-full bg-warning px-1.5 py-0.5 text-[10px] font-bold text-white">
                    {pendingCorrections?.entries?.length}
                  </span>
                )}
              </button>
            )
          })}
        </nav>
        <Button size="sm" variant="outline" onClick={() => setShowManualEntry(true)} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />
          {t('zeiterfassung.manual.newEntry')}
        </Button>
      </div>

      {/* Active View */}
      <div className="flex-1 overflow-auto p-6">
        {activeView === 'today' && (
          <TodayView
            summary={dailySummary}
            entries={entries.filter((e) => e.clockIn?.startsWith(todayStr))}
          />
        )}
        {activeView === 'week' && (
          <div className="max-w-3xl">
            <WeekSubmitBanner />
            <WeeklyView summary={weeklySummary} />
          </div>
        )}
        {activeView === 'analytics' && <AuswertungenView />}
        {activeView === 'team' && <TeamView />}
        {activeView === 'corrections' && (
          <CorrectionsView
            entries={entries}
            pendingEntries={pendingCorrections?.entries ?? []}
            onRequestCorrection={(entryId) => {
              setCorrectionEntryId(entryId)
              setShowCorrectionDialog(true)
            }}
            onApproveCorrection={(id) => approveCorrectionMutation.mutate(id)}
            isApproving={approveCorrectionMutation.isPending}
          />
        )}
      </div>

      {/* Correction Dialog */}
      <Dialog open={showCorrectionDialog} onOpenChange={setShowCorrectionDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t('profil.zeiterfassung.correctionTitle')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label>{t('profil.zeiterfassung.correctedStart')} *</Label>
                <Input type="datetime-local" value={correctionClockIn} onChange={(e) => setCorrectionClockIn(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <Label>{t('profil.zeiterfassung.correctedEnd')} *</Label>
                <Input type="datetime-local" value={correctionClockOut} onChange={(e) => setCorrectionClockOut(e.target.value)} />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label>{t('profil.zeiterfassung.breakMinutes')}</Label>
              <Input type="number" min={0} value={correctionBreak} onChange={(e) => setCorrectionBreak(Number(e.target.value))} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('profil.zeiterfassung.reason')} *</Label>
              <textarea
                value={correctionReason}
                onChange={(e) => setCorrectionReason(e.target.value)}
                rows={2}
                placeholder={t('profil.zeiterfassung.correctionReasonPlaceholder')}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCorrectionDialog(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSubmitCorrection} disabled={correctionMutation.isPending}>
              {correctionMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {t('profil.zeiterfassung.submitCorrection')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ManualEntryDialog open={showManualEntry} onOpenChange={setShowManualEntry} />
    </div>
  )
}

// ============================================================
// Sub-views
// ============================================================

function TodayView({ summary, entries }: { summary?: DailySummaryType; entries: WorkTimeEntry[] }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-6 max-w-3xl">
      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.workTime')}</p>
            <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.totalWorkedMinutes)}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.break')}</p>
            <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.totalBreakMinutes)}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.net')}</p>
            <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.netWorkMinutes)}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.overtime')}</p>
            <p className={cn('text-lg font-semibold', summary.overtimeMinutes > 0 ? 'text-warning' : 'text-foreground')}>
              {summary.overtimeMinutes > 0 ? '+' : ''}{formatMinutesDisplay(summary.overtimeMinutes)}
            </p>
          </div>
        </div>
      )}

      {/* Entries */}
      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">{t('profil.zeiterfassung.entries')}</h3>
        {entries.length === 0 ? (
          <EmptyState
            illustration={<EmptyCalendar />}
            title={t('profil.zeiterfassung.noEntriesToday')}
            description={t('profil.zeiterfassung.clockInToTrack')}
          />
        ) : (
          <div className="space-y-2">
            {entries.map((entry) => (
              <div key={entry.id} className="flex items-start gap-3 p-3 rounded-lg border border-border bg-card">
                <Clock className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2 text-sm">
                    <span className="font-mono tabular-nums text-foreground">
                      {formatTimeFromISO(entry.clockIn)}
                      {entry.clockOut ? ` – ${formatTimeFromISO(entry.clockOut)}` : ' – ...'}
                    </span>
                    {entry.status === 'active' && (
                      <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success">{t('profil.zeiterfassung.active')}</span>
                    )}
                    {entry.isCorrection && (
                      <span className="rounded-full bg-warning-light px-2 py-0.5 text-[10px] font-medium text-warning">{t('profil.zeiterfassung.correction')}</span>
                    )}
                    {entry.isManual && (
                      <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">{t('zeiterfassung.manual.manualBadge')}</span>
                    )}
                    {entry.billable && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-success/10 px-2 py-0.5 text-[10px] font-medium text-success">
                        <Receipt className="h-3 w-3" />{t('zeiterfassung.manual.billableShort')}
                      </span>
                    )}
                  </div>
                  {(entry.projectName || entry.activity) && (
                    <p className="mt-1 truncate text-xs text-foreground">
                      {entry.projectName}
                      {entry.customerName && <span className="text-muted-foreground"> · {entry.customerName}</span>}
                      {entry.activity && <span className="text-muted-foreground"> — {entry.activity}</span>}
                    </p>
                  )}
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('profil.zeiterfassung.break')}: {entry.breakMinutes}min
                    {entry.netWorkMinutes != null && ` · ${t('profil.zeiterfassung.net')}: ${formatMinutesDisplay(entry.netWorkMinutes)}`}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function WeeklyView({ summary }: { summary?: { weekStart: string; days: DailySummaryType[]; totalWorkedMinutes: number; totalBreakMinutes: number; netWorkMinutes: number; totalOvertimeMinutes: number } }) {
  const { t } = useTranslation()
  if (!summary) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    )
  }

  const dayNames = [
    t('profil.zeiterfassung.dayShort.mon'),
    t('profil.zeiterfassung.dayShort.tue'),
    t('profil.zeiterfassung.dayShort.wed'),
    t('profil.zeiterfassung.dayShort.thu'),
    t('profil.zeiterfassung.dayShort.fri'),
    t('profil.zeiterfassung.dayShort.sat'),
    t('profil.zeiterfassung.dayShort.sun'),
  ]

  return (
    <div className="space-y-6 max-w-3xl">
      {/* Week totals */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.weekTotal')}</p>
          <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.totalWorkedMinutes)}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.breaks')}</p>
          <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.totalBreakMinutes)}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.net')}</p>
          <p className="text-lg font-semibold text-foreground">{formatMinutesDisplay(summary.netWorkMinutes)}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">{t('profil.zeiterfassung.overtime')}</p>
          <p className={cn('text-lg font-semibold', summary.totalOvertimeMinutes > 0 ? 'text-warning' : 'text-foreground')}>
            {summary.totalOvertimeMinutes > 0 ? '+' : ''}{formatMinutesDisplay(summary.totalOvertimeMinutes)}
          </p>
        </div>
      </div>

      {/* Day-by-day breakdown */}
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-secondary/50">
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('profil.zeiterfassung.day')}</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('profil.zeiterfassung.workTime')}</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('profil.zeiterfassung.break')}</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('profil.zeiterfassung.net')}</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('profil.zeiterfassung.overtime')}</th>
            </tr>
          </thead>
          <tbody>
            {summary.days.map((day, i) => (
              <tr key={day.date} className="border-b border-border-muted">
                <td className="px-4 py-3 text-foreground">
                  <span className="font-medium">{dayNames[i] ?? ''}</span>
                  <span className="text-muted-foreground ml-2 text-xs">
                    {new Date(day.date).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })}
                  </span>
                </td>
                <td className="px-4 py-3 text-right tabular-nums text-foreground">
                  {day.totalWorkedMinutes > 0 ? formatMinutesDisplay(day.totalWorkedMinutes) : '-'}
                </td>
                <td className="px-4 py-3 text-right tabular-nums text-muted-foreground">
                  {day.totalBreakMinutes > 0 ? formatMinutesDisplay(day.totalBreakMinutes) : '-'}
                </td>
                <td className="px-4 py-3 text-right tabular-nums text-foreground font-medium">
                  {day.netWorkMinutes > 0 ? formatMinutesDisplay(day.netWorkMinutes) : '-'}
                </td>
                <td className={cn('px-4 py-3 text-right tabular-nums', day.overtimeMinutes > 0 ? 'text-warning font-medium' : 'text-muted-foreground')}>
                  {day.overtimeMinutes > 0 ? `+${formatMinutesDisplay(day.overtimeMinutes)}` : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function CorrectionsView({
  entries,
  pendingEntries,
  onRequestCorrection,
  onApproveCorrection,
  isApproving,
}: {
  entries: WorkTimeEntry[]
  pendingEntries: WorkTimeEntry[]
  onRequestCorrection: (entryId: string) => void
  onApproveCorrection: (id: string) => void
  isApproving: boolean
}) {
  const { t } = useTranslation()
  // Recent completed entries that can be corrected
  const correctableEntries = entries
    .filter((e) => e.status === 'completed')
    .slice(0, 10)

  return (
    <div className="space-y-6 max-w-3xl">
      {/* Pending corrections for managers */}
      {pendingEntries.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-foreground mb-3 flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-warning" />
            {t('profil.zeiterfassung.pendingCorrections', { count: pendingEntries.length })}
          </h3>
          <div className="space-y-2">
            {pendingEntries.map((entry) => (
              <div key={entry.id} className="flex items-center gap-3 p-3 rounded-lg border border-warning/30 bg-warning-light/10">
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-foreground">
                    {formatTimeFromISO(entry.clockIn)} – {entry.clockOut ? formatTimeFromISO(entry.clockOut) : '...'}
                  </div>
                  {entry.correctionReason && (
                    <p className="text-xs text-muted-foreground mt-0.5">{t('profil.zeiterfassung.reason')}: {entry.correctionReason}</p>
                  )}
                </div>
                <Button
                  size="sm"
                  onClick={() => onApproveCorrection(entry.id)}
                  disabled={isApproving}
                >
                  {t('profil.zeiterfassung.approve')}
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Request correction */}
      <div>
        <h3 className="text-sm font-medium text-foreground mb-3">{t('profil.zeiterfassung.requestCorrection')}</h3>
        {correctableEntries.length === 0 ? (
          <EmptyState
            icon={Edit3}
            title={t('profil.zeiterfassung.noCorrectableEntries')}
            description={t('profil.zeiterfassung.correctableDescription')}
          />
        ) : (
          <div className="space-y-2">
            {correctableEntries.map((entry) => (
              <div key={entry.id} className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card">
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-foreground font-mono tabular-nums">
                    {formatDate(entry.clockIn)}:{' '}
                    {formatTimeFromISO(entry.clockIn)} – {entry.clockOut ? formatTimeFromISO(entry.clockOut) : '...'}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('profil.zeiterfassung.break')}: {entry.breakMinutes}min
                    {entry.netWorkMinutes != null && ` · ${t('profil.zeiterfassung.net')}: ${formatMinutesDisplay(entry.netWorkMinutes)}`}
                  </p>
                </div>
                <Button size="sm" variant="outline" onClick={() => onRequestCorrection(entry.id)}>
                  <Edit3 className="h-3.5 w-3.5 mr-1.5" />
                  {t('profil.zeiterfassung.correct')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
