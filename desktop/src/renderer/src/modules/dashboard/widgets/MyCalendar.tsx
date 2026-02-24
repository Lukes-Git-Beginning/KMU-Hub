/**
 * My Calendar widget — personal schedule for today.
 */
import { memo } from 'react'
import { Clock, Video, MapPin, Users } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

interface TimeSlot {
  id: string
  time: string
  endTime: string
  title: string
  type: 'meeting' | 'focus' | 'call' | 'break'
  location?: string
  attendees?: number
}

const MOCK_SCHEDULE: TimeSlot[] = [
  { id: '1', time: '08:30', endTime: '09:00', title: 'Tagesplanung', type: 'focus' },
  { id: '2', time: '09:00', endTime: '10:00', title: 'Sprint Planning', type: 'meeting', location: 'Raum A', attendees: 8 },
  { id: '3', time: '10:30', endTime: '11:00', title: 'Abstimmung mit Marketing', type: 'call' },
  { id: '4', time: '11:00', endTime: '11:45', title: 'Kundentermin Meier GmbH', type: 'meeting', location: 'Video-Call', attendees: 3 },
  { id: '5', time: '12:30', endTime: '13:30', title: 'Mittagspause', type: 'break' },
  { id: '6', time: '14:00', endTime: '14:30', title: 'Design Review', type: 'meeting', location: 'Raum B', attendees: 4 },
  { id: '7', time: '15:00', endTime: '16:30', title: 'Fokuszeit: Angebot schreiben', type: 'focus' },
]

const TYPE_STYLE = {
  meeting: { color: 'bg-blue-500', label: 'Meeting' },
  focus: { color: 'bg-emerald-500', label: 'Fokus' },
  call: { color: 'bg-violet-500', label: 'Anruf' },
  break: { color: 'bg-amber-500', label: 'Pause' },
}

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
        {MOCK_SCHEDULE.map((slot) => {
          const isPast = slot.endTime < currentTimeStr
          const isCurrent = slot.time <= currentTimeStr && slot.endTime > currentTimeStr
          const style = TYPE_STYLE[slot.type]

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
                  {slot.attendees && (
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      <Users className="h-2.5 w-2.5" />
                      {slot.attendees}
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
