/**
 * My Calendar widget — personal schedule for today.
 */
import { memo } from 'react'
import { Video, MapPin, Users } from 'lucide-react'
import { TODAY_EVENTS } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const TYPE_STYLE: Record<string, { color: string; label: string }> = {
  meeting: { color: 'bg-blue-500', label: 'Meeting' },
  focus: { color: 'bg-emerald-500', label: 'Fokus' },
  call: { color: 'bg-violet-500', label: 'Anruf' },
  break: { color: 'bg-amber-500', label: 'Pause' },
  workshop: { color: 'bg-teal-500', label: 'Workshop' },
}

/** Map TODAY_EVENTS to timeline format with endTime calculation. */
const SCHEDULE = TODAY_EVENTS.map((ev) => {
  const durMatch = ev.duration.match(/(\d+)\s*(Std|Min)/)
  let endTime = ev.time
  if (durMatch) {
    const [h, m] = ev.time.split(':').map(Number)
    const addMin = durMatch[2] === 'Std' ? Number(durMatch[1]) * 60 : Number(durMatch[1])
    const total = h * 60 + m + addMin
    endTime = `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
  }
  return { ...ev, endTime, attendeeCount: ev.attendeeIds.length }
})

function MyCalendar(_props: WidgetProps) {
  const now = new Date()
  const currentHour = now.getHours()
  const currentMin = now.getMinutes()
  const currentTimeStr = `${String(currentHour).padStart(2, '0')}:${String(currentMin).padStart(2, '0')}`

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs font-medium text-muted-foreground">
          {now.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' })}
        </span>
        <span className="text-xs font-mono text-primary font-semibold">{currentTimeStr}</span>
      </div>

      {/* Timeline */}
      <div className="flex-1 overflow-auto">
        {SCHEDULE.map((slot) => {
          const isPast = slot.endTime < currentTimeStr
          const isCurrent = slot.time <= currentTimeStr && slot.endTime > currentTimeStr
          const style = TYPE_STYLE[slot.type] ?? TYPE_STYLE.meeting

          return (
            <div
              key={slot.id}
              className={`flex items-start gap-3 px-4 py-2 border-l-2 ml-4 transition-colors cursor-pointer hover:bg-accent/50 ${
                isCurrent ? 'border-l-primary bg-primary/5' : isPast ? 'border-l-border opacity-50' : 'border-l-border'
              }`}
            >
              {/* Time */}
              <div className="w-12 shrink-0 text-right">
                <p className={`text-xs font-mono ${isCurrent ? 'text-primary font-bold' : 'text-muted-foreground'}`}>
                  {slot.time}
                </p>
                <p className="text-[10px] text-muted-foreground">{slot.endTime}</p>
              </div>

              {/* Event */}
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className={`h-2 w-2 rounded-full shrink-0 ${style.color}`} />
                  <p className={`text-sm truncate ${isCurrent ? 'font-semibold text-foreground' : isPast ? 'text-muted-foreground' : 'font-medium text-foreground'}`}>
                    {slot.title}
                  </p>
                </div>
                <div className="flex items-center gap-2 mt-0.5">
                  {slot.location && (
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      {slot.location.includes('Video') ? <Video className="h-2.5 w-2.5" /> : <MapPin className="h-2.5 w-2.5" />}
                      {slot.location}
                    </span>
                  )}
                  {slot.attendeeCount > 0 && (
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      <Users className="h-2.5 w-2.5" />
                      {slot.attendeeCount}
                    </span>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default memo(MyCalendar)
