/**
 * Unified work-clock widget for the header bar.
 *
 * API-backed (HR work-time endpoints) — single source of truth. Replaces the
 * former mock-store TimeTrackerWidget and the unused ClockInButton.
 *
 * Shows clock-in/out + break controls, today's progress vs. 8h target, the
 * cumulative flextime balance (Stundenkonto) and today's entries, with a
 * shortcut into the full /zeiterfassung module. Project/task picking lands in
 * P2 once the HR API carries project_id.
 *
 * Live timer uses requestAnimationFrame (same pattern as the module tab).
 */
import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Play, Square, Coffee, Timer, ArrowRight } from 'lucide-react'
import { cn } from '@/lib/cn'
import {
  useWorkTimeStatus,
  useDailySummary,
  useWorkTimeEntries,
  useTimeBalance,
  useClockIn,
  useClockOut,
  useStartBreak,
  useEndBreak,
} from '@/api/hooks/hr-hooks'
import { formatSignedMinutes, formatWorkMinutes } from '@/lib/worktime'

const TARGET_MINUTES = 8 * 60

function todayDateStr(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function timeFromISO(iso: string): string {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export function WorkClockWidget() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data: status } = useWorkTimeStatus()
  const todayStr = useMemo(() => todayDateStr(), [])
  const { data: daily } = useDailySummary(todayStr)
  const { data: entriesData } = useWorkTimeEntries()
  const { data: balance } = useTimeBalance()

  const clockInMutation = useClockIn()
  const clockOutMutation = useClockOut()
  const startBreakMutation = useStartBreak()
  const endBreakMutation = useEndBreak()

  const [elapsed, setElapsed] = useState('00:00:00')
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const rafRef = useRef<number>(0)

  const isMutating =
    clockInMutation.isPending ||
    clockOutMutation.isPending ||
    startBreakMutation.isPending ||
    endBreakMutation.isPending

  // Live work timer (rAF) — frozen while on break.
  const updateTimer = useCallback(() => {
    if (status?.isClockedIn && status.currentShiftStart && !status.isOnBreak) {
      const start = new Date(status.currentShiftStart).getTime()
      const totalSeconds = Math.floor((Date.now() - start) / 1000)
      const h = Math.floor(totalSeconds / 3600)
      const m = Math.floor((totalSeconds % 3600) / 60)
      const s = totalSeconds % 60
      setElapsed(`${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`)
    }
    // eslint-disable-next-line react-hooks/immutability -- ref assigned in useCallback for rAF loop
    rafRef.current = requestAnimationFrame(updateTimer)
  }, [status?.isClockedIn, status?.currentShiftStart, status?.isOnBreak])

  useEffect(() => {
    if (status?.isClockedIn) {
      rafRef.current = requestAnimationFrame(updateTimer)
    } else {
      setElapsed('00:00:00')
    }
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current)
    }
  }, [status?.isClockedIn, updateTimer])

  // Close on outside click.
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const todayMinutes = daily?.netWorkMinutes ?? status?.todayTotalMinutes ?? 0
  const progressPercent = Math.min(100, Math.round((todayMinutes / TARGET_MINUTES) * 100))

  const todayEntries = useMemo(() => {
    const all = entriesData?.entries ?? []
    return all.filter((e) => e.clockIn?.startsWith(todayStr)).slice(-4)
  }, [entriesData, todayStr])

  const saldoMinutes = balance?.balanceMinutes ?? 0

  const handleGoToModule = () => {
    navigate('/zeiterfassung')
    setIsOpen(false)
  }
  const handleClockIn = () => clockInMutation.mutate()
  const handleClockOut = () => {
    clockOutMutation.mutate()
    setIsOpen(false)
  }

  return (
    <div className="relative" ref={containerRef} data-tour="time-tracker">
      {/* Trigger */}
      <button
        onClick={() => setIsOpen((v) => !v)}
        disabled={isMutating}
        className={cn(
          'flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-sm transition-colors',
          status?.isClockedIn
            ? status.isOnBreak
              ? 'bg-warning/10 hover:bg-warning/15'
              : 'bg-success/10 hover:bg-success/15'
            : 'hover:bg-accent',
        )}
        title={
          status?.isClockedIn
            ? status.isOnBreak
              ? t('header.timeTracker.break')
              : t('header.timeTracker.clockedIn')
            : t('header.timeTracker.clockIn')
        }
      >
        {!status?.isClockedIn ? (
          <>
            <Timer className="h-4 w-4 text-muted-foreground" />
            <span className="hidden text-xs text-muted-foreground sm:inline">
              {t('header.timeTracker.clockIn')}
            </span>
          </>
        ) : status.isOnBreak ? (
          <>
            <Coffee className="h-4 w-4 text-warning" />
            <span className="text-xs font-medium text-warning-foreground">
              {t('header.timeTracker.break')}
            </span>
          </>
        ) : (
          <>
            <span className="relative flex h-2.5 w-2.5">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success/75 opacity-75" />
              <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-success" />
            </span>
            <span className="font-mono text-sm font-semibold tabular-nums text-primary">
              {elapsed}
            </span>
          </>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 top-full z-50 mt-2 w-80 overflow-hidden rounded-xl border border-border bg-card shadow-xl">
          {/* ── Not clocked in ── */}
          {!status?.isClockedIn ? (
            <div className="space-y-3 p-4">
              <div className="space-y-1 text-center">
                <Timer className="mx-auto h-6 w-6 text-muted-foreground" />
                <p className="text-sm font-medium text-foreground">{t('header.timeTracker.startDayTitle')}</p>
                <p className="text-xs text-muted-foreground">{t('header.timeTracker.startDaySubtitle')}</p>
              </div>
              {saldoMinutes !== 0 && (
                <div className="text-center">
                  <span
                    className={cn(
                      'text-xs font-medium tabular-nums',
                      saldoMinutes >= 0 ? 'text-success' : 'text-destructive',
                    )}
                  >
                    {t('header.timeTracker.saldo')}: {formatSignedMinutes(saldoMinutes)}
                  </span>
                </div>
              )}
              <button
                onClick={handleClockIn}
                disabled={isMutating}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
              >
                <Play className="h-4 w-4" />
                {t('header.timeTracker.clockIn')}
              </button>
              <button
                onClick={handleGoToModule}
                className="flex w-full items-center justify-center gap-1.5 text-xs text-primary transition-colors hover:text-primary/80"
              >
                {t('header.timeTracker.goToTimeTracking')}
                <ArrowRight className="h-3 w-3" />
              </button>
            </div>
          ) : (
            <>
              {/* ── Work-time header ── */}
              <div className="border-b border-border px-4 py-3">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">{t('header.timeTracker.workTime')}</span>
                  <span className="font-mono text-sm font-semibold tabular-nums text-foreground">
                    {status.isOnBreak ? t('header.timeTracker.break') : elapsed}
                  </span>
                </div>
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                  <div
                    className={cn(
                      'h-full rounded-full transition-all',
                      progressPercent >= 100 ? 'bg-warning' : progressPercent >= 80 ? 'bg-success' : 'bg-primary',
                    )}
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
                <div className="mt-1 flex items-center justify-between">
                  <span className="text-[10px] tabular-nums text-muted-foreground">
                    {formatWorkMinutes(todayMinutes)} / {formatWorkMinutes(TARGET_MINUTES)}
                  </span>
                  <span
                    className={cn(
                      'text-[10px] font-medium tabular-nums',
                      saldoMinutes >= 0 ? 'text-success' : 'text-destructive',
                    )}
                  >
                    {t('header.timeTracker.saldo')}: {formatSignedMinutes(saldoMinutes)}
                  </span>
                </div>
              </div>

              {/* ── Today's entries ── */}
              {todayEntries.length > 0 && (
                <div className="border-b border-border px-4 py-2">
                  <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
                    {t('header.timeTracker.today')}
                  </span>
                  {todayEntries.map((entry) => (
                    <div key={entry.id} className="flex items-center gap-2 py-1 text-xs">
                      <span className="w-[78px] shrink-0 font-mono tabular-nums text-muted-foreground">
                        {timeFromISO(entry.clockIn)}–{entry.clockOut ? timeFromISO(entry.clockOut) : '…'}
                      </span>
                      <span className="flex-1 truncate text-foreground">
                        {entry.netWorkMinutes != null ? formatWorkMinutes(entry.netWorkMinutes) : '—'}
                      </span>
                      {entry.status === 'active' && (
                        <span className="rounded-full bg-success-light px-1.5 py-0.5 text-[9px] font-medium text-success">
                          {t('header.timeTracker.clockedIn')}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* ── Footer actions ── */}
              <div className="space-y-2 px-4 py-3">
                <div className="flex gap-2">
                  {status.isOnBreak ? (
                    <button
                      onClick={() => endBreakMutation.mutate()}
                      disabled={isMutating}
                      className="flex flex-1 items-center justify-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
                    >
                      <Play className="h-3.5 w-3.5" />
                      {t('header.timeTracker.endBreak')}
                    </button>
                  ) : (
                    <button
                      onClick={() => startBreakMutation.mutate()}
                      disabled={isMutating}
                      className="flex flex-1 items-center justify-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
                    >
                      <Coffee className="h-3.5 w-3.5" />
                      {t('header.timeTracker.break')}
                    </button>
                  )}
                  <button
                    onClick={handleClockOut}
                    disabled={isMutating}
                    className="flex flex-1 items-center justify-center gap-1.5 rounded-md bg-destructive px-3 py-1.5 text-xs font-medium text-destructive-foreground transition-colors hover:bg-destructive/90"
                  >
                    <Square className="h-3.5 w-3.5" />
                    {t('header.timeTracker.clockOut')}
                  </button>
                </div>
                <button
                  onClick={handleGoToModule}
                  className="flex w-full items-center justify-center gap-1.5 text-xs text-primary transition-colors hover:text-primary/80"
                >
                  {t('header.timeTracker.goToTimeTracking')}
                  <ArrowRight className="h-3 w-3" />
                </button>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}
