/**
 * Time Clock widget — clock in/out with today's work hours.
 */
import { memo, useState } from 'react'
import { Clock, LogIn, LogOut, Coffee } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const MOCK = {
  clockedIn: true,
  startTime: '08:15',
  breakMinutes: 45,
  entries: [
    { type: 'in' as const, time: '08:15' },
    { type: 'break' as const, time: '12:30' },
    { type: 'in' as const, time: '13:15' },
  ],
  weekHours: { mo: 8.5, di: 8.0, mi: 7.5, do: 0, fr: 0 },
  targetWeek: 40,
}

function TimeClockWidget(_props: WidgetProps) {
  const [isClockedIn, setIsClockedIn] = useState(MOCK.clockedIn)

  const now = new Date()
  const currentMinutes = now.getHours() * 60 + now.getMinutes()
  const startMinutes = 8 * 60 + 15 // 08:15
  const workedMinutes = currentMinutes - startMinutes - MOCK.breakMinutes
  const workedHours = Math.max(0, workedMinutes / 60)
  const workedDisplay = `${Math.floor(workedHours)}h ${Math.round((workedHours % 1) * 60)}min`

  const weekTotal = Object.values(MOCK.weekHours).reduce((s, h) => s + h, 0) + workedHours
  const weekPct = Math.min(100, (weekTotal / MOCK.targetWeek) * 100)

  return (
    <div className="flex h-full flex-col p-4">
      {/* Status */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${
            isClockedIn ? 'bg-emerald-500/10' : 'bg-gray-500/10'
          }`}>
            <Clock className={`h-4 w-4 ${isClockedIn ? 'text-emerald-600' : 'text-muted-foreground'}`} />
          </div>
          <div>
            <p className="text-xs font-medium text-muted-foreground">
              {isClockedIn ? 'Eingestempelt' : 'Ausgestempelt'}
            </p>
            {isClockedIn && (
              <p className="text-xs text-muted-foreground">seit {MOCK.startTime}</p>
            )}
          </div>
        </div>
        <Button
          size="sm"
          variant={isClockedIn ? 'outline' : 'default'}
          onClick={() => setIsClockedIn(!isClockedIn)}
        >
          {isClockedIn ? (
            <><LogOut className="mr-1.5 h-3.5 w-3.5" />Ausstempeln</>
          ) : (
            <><LogIn className="mr-1.5 h-3.5 w-3.5" />Einstempeln</>
          )}
        </Button>
      </div>

      {/* Today's hours */}
      <div className="rounded-lg bg-secondary/50 p-3 mb-3">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs text-muted-foreground">Heute</span>
          <span className="text-sm font-bold text-foreground">{workedDisplay}</span>
        </div>
        <div className="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
          <div
            className="h-full bg-primary rounded-full transition-all"
            style={{ width: `${Math.min(100, (workedHours / 8) * 100)}%` }}
          />
        </div>
        <p className="text-[10px] text-muted-foreground mt-1 text-right">Soll: 8h</p>
      </div>

      {/* Week progress */}
      <div className="mt-auto">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs text-muted-foreground">Woche</span>
          <span className="text-xs font-medium text-foreground">{weekTotal.toFixed(1)}h / {MOCK.targetWeek}h</span>
        </div>
        <div className="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
          <div
            className="h-full bg-emerald-500 rounded-full transition-all"
            style={{ width: `${weekPct}%` }}
          />
        </div>
      </div>

      {/* Today's log */}
      <div className="mt-3 flex items-center gap-2">
        {MOCK.entries.map((entry, i) => (
          <span key={i} className="flex items-center gap-1 text-[10px] text-muted-foreground">
            {entry.type === 'in' ? <LogIn className="h-2.5 w-2.5" /> : <Coffee className="h-2.5 w-2.5" />}
            {entry.time}
          </span>
        ))}
      </div>
    </div>
  )
}

export default memo(TimeClockWidget)
