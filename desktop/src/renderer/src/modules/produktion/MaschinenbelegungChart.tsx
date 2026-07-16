import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Cpu } from 'lucide-react'
import type { MachineResponse as Machine, MachineBooking } from '@/api/produktion-types'
import { machineStatusLabelKeys, machineStatusDots } from './produktion-shared'

interface MaschinenbelegungChartProps {
  machines: Machine[]
  bookings: MachineBooking[]
  /** production_order_id → order_number (the adapter payload only carries ids). */
  orderNumbers: Map<string, string>
  onSelectOrder?: (orderId: string) => void
  onSelectMachine?: (machineId: string) => void
}

// Rolling window relative to today (the previous fixed Feb-2026 range meant
// the Gantt was permanently empty once the seeds moved on).
const DAYS_BEFORE = 7
const TOTAL_DAYS = 35
const DAY_WIDTH = 30

function startOfToday(): Date {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d
}

function hashColor(id: string): string {
  const colors = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#ef4444', '#14b8a6']
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = (hash * 31 + id.charCodeAt(i)) & 0xffff
  }
  return colors[hash % colors.length]
}

export default function MaschinenbelegungChart({
  machines,
  bookings,
  orderNumbers,
  onSelectOrder,
  onSelectMachine,
}: MaschinenbelegungChartProps) {
  const { t } = useTranslation()

  const rangeStart = useMemo(() => {
    const d = startOfToday()
    d.setDate(d.getDate() - DAYS_BEFORE)
    return d
  }, [])

  const getDayOffset = useMemo(() => {
    return (dateStr: string): number => {
      const date = new Date(dateStr)
      return Math.floor((date.getTime() - rangeStart.getTime()) / (1000 * 60 * 60 * 24))
    }
  }, [rangeStart])

  const mondays = useMemo(() => {
    const result: { offset: number; label: string }[] = []
    const current = new Date(rangeStart)
    while (current.getDay() !== 1) {
      current.setDate(current.getDate() + 1)
    }
    while ((current.getTime() - rangeStart.getTime()) / (1000 * 60 * 60 * 24) < TOTAL_DAYS) {
      const offset = Math.floor((current.getTime() - rangeStart.getTime()) / (1000 * 60 * 60 * 24))
      result.push({ offset, label: `${current.getDate()}.${current.getMonth() + 1}.` })
      current.setDate(current.getDate() + 7)
    }
    return result
  }, [rangeStart])

  const machineStatusConfig = useMemo(() => {
    return Object.fromEntries(
      Object.entries(machineStatusLabelKeys).map(([key, labelKey]) => [
        key,
        { dot: machineStatusDots[key] ?? 'bg-muted', label: t(labelKey) },
      ])
    ) as Record<string, { dot: string; label: string }>
  }, [t])

  const bookingsByMachine = useMemo(() => {
    const map = new Map<string, MachineBooking[]>()
    machines.forEach((m) => map.set(m.id, []))
    bookings.forEach((b) => {
      const list = map.get(b.machine_id)
      if (list) list.push(b)
    })
    return map
  }, [machines, bookings])

  const chartWidth = TOTAL_DAYS * DAY_WIDTH
  const todayOffset = DAYS_BEFORE

  if (machines.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <Cpu className="h-10 w-10 mb-3 opacity-40" />
        <p className="text-sm font-medium">{t('produktion.chart.keineMaschinen')}</p>
        <p className="text-xs mt-1">{t('produktion.chart.maschinenHint')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Chart container */}
      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <div style={{ minWidth: chartWidth + 200 }}>
            {/* Header row: machine name column + day columns */}
            <div className="flex border-b border-border">
              {/* Machine name header */}
              <div className="w-[200px] shrink-0 px-4 py-3 border-r border-border">
                <span className="text-xs font-medium text-muted-foreground">{t('produktion.chart.maschine')}</span>
              </div>
              {/* Day grid header */}
              <div className="relative" style={{ width: chartWidth, height: 36 }}>
                {/* Monday labels */}
                {mondays.map((m, i) => (
                  <div
                    key={i}
                    className="absolute top-0 text-[10px] text-muted-foreground font-medium"
                    style={{ left: m.offset * DAY_WIDTH, paddingTop: 10 }}
                  >
                    {m.label}
                  </div>
                ))}
                {/* Today marker label */}
                <div
                  className="absolute top-0 text-[10px] text-primary font-semibold"
                  style={{ left: todayOffset * DAY_WIDTH + 4, paddingTop: 10 }}
                >
                  {t('produktion.chart.heute')}
                </div>
              </div>
            </div>

            {/* Machine rows */}
            {machines.map((machine) => {
              const mBookings = bookingsByMachine.get(machine.id) || []
              const statusCfg = machineStatusConfig[machine.status]

              return (
                <div
                  key={machine.id}
                  className="flex border-b border-border-muted last:border-0 hover:bg-secondary/30 transition-colors"
                >
                  {/* Machine info (click-through to the machine detail modal) */}
                  <div
                    role="button"
                    tabIndex={0}
                    onClick={() => onSelectMachine?.(machine.id)}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelectMachine?.(machine.id) } }}
                    className="w-[200px] shrink-0 px-4 py-3 border-r border-border flex items-center gap-2 cursor-pointer hover:bg-secondary/50 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                  >
                    <span className={`h-2.5 w-2.5 rounded-full shrink-0 ${statusCfg?.dot ?? 'bg-muted'}`} />
                    <div className="min-w-0">
                      <p className="text-xs font-medium text-foreground truncate">{machine.name}</p>
                      <p className="text-[10px] text-muted-foreground">{machine.type}</p>
                    </div>
                  </div>

                  {/* Gantt row */}
                  <div className="relative" style={{ width: chartWidth, height: 52 }}>
                    {/* Vertical grid lines at Mondays */}
                    {mondays.map((m, i) => (
                      <div
                        key={i}
                        className="absolute top-0 bottom-0 border-l border-border-muted/50"
                        style={{ left: m.offset * DAY_WIDTH }}
                      />
                    ))}
                    {/* Today line */}
                    <div
                      className="absolute top-0 bottom-0 border-l-2 border-primary/60"
                      style={{ left: todayOffset * DAY_WIDTH }}
                    />

                    {/* Booking blocks */}
                    {mBookings.map((booking) => {
                      const startOffset = Math.max(0, getDayOffset(booking.starts_at))
                      const endOffset = Math.min(TOTAL_DAYS, getDayOffset(booking.ends_at) + 1)
                      const span = endOffset - startOffset
                      const color = hashColor(booking.production_order_id)
                      const orderLabel = orderNumbers.get(booking.production_order_id) ?? booking.production_order_id.slice(0, 8)

                      if (span <= 0) return null

                      return (
                        <div
                          key={booking.id}
                          role="button"
                          tabIndex={0}
                          onClick={() => onSelectOrder?.(booking.production_order_id)}
                          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelectOrder?.(booking.production_order_id) } }}
                          className="absolute top-2 rounded-md flex items-center px-2 overflow-hidden cursor-pointer transition-opacity hover:opacity-80 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                          style={{
                            left: startOffset * DAY_WIDTH + 1,
                            width: span * DAY_WIDTH - 2,
                            height: 32,
                            backgroundColor: color + '30',
                            borderLeft: `3px solid ${color}`,
                          }}
                          title={`${orderLabel}\n${booking.starts_at.slice(0, 10)} – ${booking.ends_at.slice(0, 10)}`}
                        >
                          <span
                            className="text-[10px] font-medium truncate"
                            style={{ color }}
                          >
                            {orderLabel}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-6 flex-wrap">
        {/* Machine statuses */}
        <div className="flex items-center gap-4">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('produktion.chart.maschinenstatus')}</span>
          {Object.entries(machineStatusConfig).map(([key, cfg]) => (
            <div key={key} className="flex items-center gap-1.5">
              <span className={`h-2.5 w-2.5 rounded-full ${cfg.dot}`} />
              <span className="text-xs text-muted-foreground">{cfg.label}</span>
            </div>
          ))}
        </div>

        {/* Booking colors — derive from actual data */}
        {bookings.length > 0 && (
          <div className="flex items-center gap-4 flex-wrap">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('produktion.chart.auftraege')}</span>
            {Array.from(new Map(bookings.map((b) => [b.production_order_id, b])).values()).map((b) => {
              const color = hashColor(b.production_order_id)
              const label = orderNumbers.get(b.production_order_id) ?? b.production_order_id.slice(0, 8)
              return (
                <div key={b.production_order_id} className="flex items-center gap-1.5">
                  <span
                    className="h-2.5 w-5 rounded-sm"
                    style={{ backgroundColor: color + '60', borderLeft: `2px solid ${color}` }}
                  />
                  <span className="text-xs text-muted-foreground">{label}</span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
