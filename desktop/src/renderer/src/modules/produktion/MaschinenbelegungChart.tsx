import { useMemo } from 'react'
import { Cpu, Wrench, CircleDot } from 'lucide-react'
import type { Machine, MachineBooking } from '@/stores/produktion'

interface MaschinenbelegungChartProps {
  machines: Machine[]
  bookings: MachineBooking[]
}

const RANGE_START = new Date('2026-02-03')
const RANGE_END = new Date('2026-03-08') // exclusive: day after Mar 7
const TOTAL_DAYS = 33 // Feb 3 - Mar 7 inclusive = 33 days (Feb has 28 days in 2026)
const DAY_WIDTH = 30

const machineStatusConfig: Record<string, { dot: string; label: string }> = {
  available: { dot: 'bg-success', label: 'Verfuegbar' },
  in_use: { dot: 'bg-info', label: 'In Betrieb' },
  maintenance: { dot: 'bg-error', label: 'Wartung' },
}

function getDayOffset(dateStr: string): number {
  const date = new Date(dateStr)
  return Math.floor((date.getTime() - RANGE_START.getTime()) / (1000 * 60 * 60 * 24))
}

function getMondays(): { offset: number; label: string }[] {
  const mondays: { offset: number; label: string }[] = []
  const current = new Date(RANGE_START)

  // Move to the first Monday on or after RANGE_START
  while (current.getDay() !== 1) {
    current.setDate(current.getDate() + 1)
  }

  while (current < RANGE_END) {
    const offset = getDayOffset(current.toISOString().split('T')[0])
    const label = `${current.getDate()}.${current.getMonth() + 1}.`
    mondays.push({ offset, label })
    current.setDate(current.getDate() + 7)
  }

  return mondays
}

export default function MaschinenbelegungChart({ machines, bookings }: MaschinenbelegungChartProps) {
  const mondays = useMemo(() => getMondays(), [])

  const bookingsByMachine = useMemo(() => {
    const map = new Map<string, MachineBooking[]>()
    machines.forEach((m) => map.set(m.id, []))
    bookings.forEach((b) => {
      const list = map.get(b.machineId)
      if (list) list.push(b)
    })
    return map
  }, [machines, bookings])

  const chartWidth = TOTAL_DAYS * DAY_WIDTH

  if (machines.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <Cpu className="h-10 w-10 mb-3 opacity-40" />
        <p className="text-sm font-medium">Keine Maschinen vorhanden</p>
        <p className="text-xs mt-1">Maschinen werden hier als Belegungsplan angezeigt</p>
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
                <span className="text-xs font-medium text-muted-foreground">Maschine</span>
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
                  {/* Machine info */}
                  <div className="w-[200px] shrink-0 px-4 py-3 border-r border-border flex items-center gap-2">
                    <span className={`h-2.5 w-2.5 rounded-full shrink-0 ${statusCfg.dot}`} />
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

                    {/* Booking blocks */}
                    {mBookings.map((booking) => {
                      const startOffset = Math.max(0, getDayOffset(booking.startDate))
                      const endOffset = Math.min(TOTAL_DAYS, getDayOffset(booking.endDate))
                      const span = endOffset - startOffset

                      if (span <= 0) return null

                      return (
                        <div
                          key={booking.id}
                          className="absolute top-2 rounded-md flex items-center px-2 overflow-hidden cursor-default"
                          style={{
                            left: startOffset * DAY_WIDTH + 1,
                            width: span * DAY_WIDTH - 2,
                            height: 32,
                            backgroundColor: booking.color + '30',
                            borderLeft: `3px solid ${booking.color}`,
                          }}
                          title={`${booking.orderNr} — ${booking.product}\n${booking.startDate} bis ${booking.endDate}`}
                        >
                          <span
                            className="text-[10px] font-medium truncate"
                            style={{ color: booking.color }}
                          >
                            {booking.orderNr}
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
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Maschinenstatus:</span>
          {Object.entries(machineStatusConfig).map(([key, cfg]) => (
            <div key={key} className="flex items-center gap-1.5">
              <span className={`h-2.5 w-2.5 rounded-full ${cfg.dot}`} />
              <span className="text-xs text-muted-foreground">{cfg.label}</span>
            </div>
          ))}
        </div>

        {/* Booking colors — derive from actual data */}
        {bookings.length > 0 && (
          <div className="flex items-center gap-4">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Auftraege:</span>
            {Array.from(new Map(bookings.map((b) => [b.orderNr, b])).values()).map((b) => (
              <div key={b.orderNr} className="flex items-center gap-1.5">
                <span
                  className="h-2.5 w-5 rounded-sm"
                  style={{ backgroundColor: b.color + '60', borderLeft: `2px solid ${b.color}` }}
                />
                <span className="text-xs text-muted-foreground">{b.orderNr}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
