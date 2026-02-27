/**
 * Header mini-widget — shows the next upcoming meeting.
 */
import { Calendar, Clock } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

const MOCK_NEXT = {
  title: 'Sprint Planning',
  time: '09:00',
  in: '23 Min',
}

export function NextMeetingWidget() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors">
          <Calendar className="h-3.5 w-3.5 text-primary" />
          <span className="hidden md:inline text-muted-foreground font-medium max-w-[100px] truncate">
            {MOCK_NEXT.title}
          </span>
          <span className="text-primary font-semibold">{MOCK_NEXT.in}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="flex items-center gap-2">
          <Clock className="h-3.5 w-3.5" />
          <span>{MOCK_NEXT.title} um {MOCK_NEXT.time}</span>
        </div>
      </TooltipContent>
    </Tooltip>
  )
}
