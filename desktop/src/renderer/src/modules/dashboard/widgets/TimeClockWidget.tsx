/**
 * Time Clock widget — clock in/out with today's work hours.
 */
import { memo } from 'react'
import { Clock, LogIn, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useWorkTimeStatus, useClockIn, useClockOut } from '@/api/hooks/hr-hooks'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function TimeClockWidget(_props: WidgetProps) {
  const { data: status, isLoading } = useWorkTimeStatus()
  const clockIn = useClockIn()
  const clockOut = useClockOut()

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const isClockedIn = status?.isClockedIn ?? false
  const todayMinutes = status?.todayTotalMinutes ?? 0
  const workedHours = todayMinutes / 60
  const workedDisplay = `${Math.floor(workedHours)}h ${Math.round((workedHours % 1) * 60)}min`

  // Format start time from ISO
  let startTimeDisplay = ''
  if (status?.currentShiftStart) {
    const d = new Date(status.currentShiftStart)
    startTimeDisplay = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  }

  // Week progress — use daily total * 5 as rough estimate since we only have today's data
  // TODO: Integrate weekly summary for accurate week hours
  const targetWeek = 40
  const weekHoursEstimate = workedHours // Only today's hours available from status endpoint
  const weekPct = Math.min(100, (weekHoursEstimate / targetWeek) * 100)

  const handleToggle = () => {
    if (isClockedIn) {
      clockOut.mutate()
    } else {
      clockIn.mutate()
    }
  }

  const isMutating = clockIn.isPending || clockOut.isPending

  return (
    <div className="flex h-full flex-col p-4">
      {/* Status */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${
            isClockedIn ? 'bg-success/10' : 'bg-gray-500/10'
          }`}>
            <Clock className={`h-4 w-4 ${isClockedIn ? 'text-success' : 'text-muted-foreground'}`} />
          </div>
          <div>
            <p className="text-xs font-medium text-muted-foreground">
              {isClockedIn ? 'Eingestempelt' : 'Ausgestempelt'}
            </p>
            {isClockedIn && startTimeDisplay && (
              <p className="text-xs text-muted-foreground">seit {startTimeDisplay}</p>
            )}
          </div>
        </div>
        <Button
          size="sm"
          variant={isClockedIn ? 'outline' : 'default'}
          onClick={handleToggle}
          disabled={isMutating}
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
          <span className="text-xs font-medium text-foreground">{weekHoursEstimate.toFixed(1)}h / {targetWeek}h</span>
        </div>
        <div className="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
          <div
            className="h-full bg-success rounded-full transition-all"
            style={{ width: `${weekPct}%` }}
          />
        </div>
      </div>
    </div>
  )
}

export default memo(TimeClockWidget)
