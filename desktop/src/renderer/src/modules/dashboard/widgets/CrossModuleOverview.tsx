/**
 * Cross-Module Overview widget — "Heute im Überblick"
 *
 * Aggregates key metrics from multiple modules with links.
 * No module gate (always available). Individual data sources are
 * flag-gated dashboard-locally (fail-open) to avoid empty dashboard
 * on unavailable flags.
 *
 * Sources:
 *   • Tasks      — open tasks assigned to me (modules.tasks)
 *   • Calendar   — events today (modules.calendar)
 *   • Chat       — unread messages (modules.chat) via store
 *   • Finance    — overdue invoices (modules.finance)
 *   • Vertraege  — expiring contracts (modules.vertraege)
 *
 * CSS bar pattern: height percentage as inline style, GPU-safe (no width animation).
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import {
  CheckCircle2,
  CalendarDays,
  MessageSquare,
  Receipt,
  FileWarning,
  ChevronRight,
} from 'lucide-react'
import { useMyTasks } from '@/api/hooks/useTasks'
import { useCalendars } from '@/api/hooks/useCalendars'
import { useEventsInRange } from '@/api/hooks/useEvents'
import { useInvoices } from '@/api/hooks/useFinance'
import { useVertraegeStore } from '@/stores/vertraege'
import { useUnreadCounts } from '@/api/hooks/useChannels'
import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

// ---------------------------------------------------------------------------
// Dashboard-local module flag helper (fail-open, DEV QA-override)
// ---------------------------------------------------------------------------

function isDashboardModuleAllowed(
  moduleId: string,
  flags: Record<string, boolean> | undefined,
  flagsLoading: boolean,
  flagsError: unknown,
): boolean {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const qaOverride = (window as any).__cosmi_qa_flags__
    if (qaOverride && typeof qaOverride === 'object' && `modules.${moduleId}` in qaOverride) {
      return Boolean(qaOverride[`modules.${moduleId}`])
    }
  }
  if (flagsError || flagsLoading || !flags) return true
  return flags[`modules.${moduleId}`] ?? false
}

// ---------------------------------------------------------------------------
// Metric row component
// ---------------------------------------------------------------------------

interface MetricRowProps {
  icon: React.ReactNode
  labelKey: string
  labelVars?: Record<string, string | number>
  value: number | string
  path: string
  highlight?: boolean
}

function MetricRow({ icon, labelKey, labelVars, value, path, highlight }: MetricRowProps) {
  const { t } = useTranslation()
  return (
    <Link
      to={path}
      className="group flex items-center gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-accent/60"
    >
      <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-md ${highlight ? 'bg-destructive/10' : 'bg-muted'}`}>
        <span className={highlight ? 'text-destructive' : 'text-muted-foreground'}>
          {icon}
        </span>
      </div>
      <span className="flex-1 text-sm text-foreground">
        {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
        {(t as any)(labelKey, labelVars ?? {})}
      </span>
      <div className="flex items-center gap-1">
        <span className={`text-sm font-semibold tabular-nums ${highlight ? 'text-destructive' : 'text-foreground'}`}>
          {value}
        </span>
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
      </div>
    </Link>
  )
}

// ---------------------------------------------------------------------------
// Widget
// ---------------------------------------------------------------------------

function CrossModuleOverview(_props: WidgetProps) {
  const { t } = useTranslation()
  const { flags, isLoading: flagsLoading, error: flagsError } = useFeatureFlags()

  // ── Tasks ────────────────────────────────────────────────────────────────
  const tasksAllowed = isDashboardModuleAllowed('tasks', flags, flagsLoading, flagsError)
  const { data: tasksData } = useMyTasks({ page_size: 50, include_completed: false })
  const openTaskCount = useMemo(() => {
    if (!tasksAllowed) return null
    const tasks = (tasksData as { tasks?: unknown[] } | undefined)?.tasks ?? []
    return tasks.length
  }, [tasksData, tasksAllowed])

  // ── Calendar events today ───────────────────────────────────────────────
  const calendarAllowed = isDashboardModuleAllowed('calendar', flags, flagsLoading, flagsError)
  const todayRange = useMemo(() => {
    const today = new Date()
    const dateStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
    return { start: `${dateStr}T00:00:00Z`, end: `${dateStr}T23:59:59Z` }
  }, [])
  const { data: calData } = useCalendars()
  const calendarIds = useMemo(() => {
    if (!calendarAllowed) return []
    const calendars = (calData as { calendars?: Array<{ id: string }> })?.calendars ?? []
    return calendars.map((c) => c.id)
  }, [calData, calendarAllowed])
  const { data: eventData } = useEventsInRange(calendarIds, todayRange.start, todayRange.end)
  const eventsTodayCount = useMemo(() => {
    if (!calendarAllowed) return null
    const events = (eventData as { events?: unknown[] } | undefined)?.events ?? []
    return events.length
  }, [eventData, calendarAllowed])

  // ── Unread messages — proxy from store (kommunikation module = chat) ────
  const chatAllowed = isDashboardModuleAllowed('chat', flags, flagsLoading, flagsError)
  const { data: unreadData } = useUnreadCounts()
  const unreadCount = useMemo(() => {
    if (!chatAllowed) return null
    const counts = (unreadData as { unread_counts?: Record<string, number> } | undefined)?.unread_counts ?? {}
    return Object.values(counts).reduce((sum, n) => sum + (n ?? 0), 0)
  }, [unreadData, chatAllowed])

  // ── Finance: overdue invoices ──────────────────────────────────────────
  const financeAllowed = isDashboardModuleAllowed('finance', flags, flagsLoading, flagsError)
  const { data: invoicesData } = useInvoices(
    financeAllowed ? { status: 'overdue', page_size: 50 } : undefined,
  )
  const overdueInvoiceCount = useMemo(() => {
    if (!financeAllowed) return null
    return (invoicesData as { invoices?: unknown[] } | undefined)?.invoices?.length ?? 0
  }, [invoicesData, financeAllowed])

  // ── Vertraege: expiring ────────────────────────────────────────────────
  const vertraegeAllowed = isDashboardModuleAllowed('vertraege', flags, flagsLoading, flagsError)
  const contracts = useVertraegeStore((s) => s.contracts)
  const expiringCount = useMemo(() => {
    if (!vertraegeAllowed) return null
    return contracts.filter((c) => c.status === 'expiring').length
  }, [contracts, vertraegeAllowed])

  // Collect visible rows
  const rows = useMemo(() => {
    const r: MetricRowProps[] = []

    if (openTaskCount !== null) {
      r.push({
        icon: <CheckCircle2 className="h-4 w-4" />,
        labelKey: 'dashboard.crossModuleOverview.openTasks',
        labelVars: { count: openTaskCount },
        value: openTaskCount,
        path: '/work/tasks',
      })
    }
    if (eventsTodayCount !== null) {
      r.push({
        icon: <CalendarDays className="h-4 w-4" />,
        labelKey: 'dashboard.crossModuleOverview.eventsToday',
        labelVars: { count: eventsTodayCount },
        value: eventsTodayCount,
        path: '/kalender',
      })
    }
    if (unreadCount !== null) {
      r.push({
        icon: <MessageSquare className="h-4 w-4" />,
        labelKey: 'dashboard.crossModuleOverview.unreadMessages',
        labelVars: { count: unreadCount },
        value: unreadCount,
        path: '/kommunikation',
      })
    }
    if (overdueInvoiceCount !== null) {
      r.push({
        icon: <Receipt className="h-4 w-4" />,
        labelKey: 'dashboard.crossModuleOverview.overdueInvoices',
        labelVars: { count: overdueInvoiceCount },
        value: overdueInvoiceCount,
        path: '/finanzen',
        highlight: overdueInvoiceCount > 0,
      })
    }
    if (expiringCount !== null) {
      r.push({
        icon: <FileWarning className="h-4 w-4" />,
        labelKey: 'dashboard.crossModuleOverview.expiringContracts',
        labelVars: { count: expiringCount },
        value: expiringCount,
        path: '/vertraege',
        highlight: expiringCount > 0,
      })
    }
    return r
  }, [openTaskCount, eventsTodayCount, unreadCount, overdueInvoiceCount, expiringCount])

  if (rows.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <p className="text-sm text-muted-foreground">{t('dashboard.crossModuleOverview.noData')}</p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          {t('dashboard.crossModuleOverview.title')}
        </p>
      </div>

      {/* Metric rows */}
      <div className="flex-1 overflow-auto px-1 pb-2">
        {rows.map((row) => (
          <MetricRow key={row.labelKey} {...row} />
        ))}
      </div>
    </div>
  )
}

export default memo(CrossModuleOverview)
