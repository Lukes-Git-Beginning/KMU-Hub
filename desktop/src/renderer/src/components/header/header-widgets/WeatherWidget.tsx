/**
 * Header mini-widget — shows current weather (mock data).
 */
import { CloudSun } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

const MOCK = {
  temp: 8,
  condition: 'Teilweise bewölkt',
  location: 'Zürich',
  high: 11,
  low: 3,
}

export function WeatherWidget() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors">
          <CloudSun className="h-3.5 w-3.5 text-amber-500" />
          <span className="font-semibold text-foreground">{MOCK.temp}°C</span>
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="text-center">
          <p className="font-medium">{MOCK.location}</p>
          <p className="text-xs text-muted-foreground">{MOCK.condition}</p>
          <p className="text-xs mt-0.5">H: {MOCK.high}° · T: {MOCK.low}°</p>
        </div>
      </TooltipContent>
    </Tooltip>
  )
}
