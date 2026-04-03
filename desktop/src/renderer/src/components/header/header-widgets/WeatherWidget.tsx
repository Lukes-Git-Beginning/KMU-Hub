/**
 * Header mini-widget — shows current weather (mock data).
 * Clicking opens a popover with weather details.
 */
import { CloudSun, Droplets, Wind, Eye } from 'lucide-react'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

const MOCK = {
  temp: 8,
  condition: 'Teilweise bewölkt',
  location: 'Berlin',
  high: 11,
  low: 3,
  humidity: 62,
  wind: 14,
  visibility: 10,
}

export function WeatherWidget() {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors">
          <CloudSun className="h-3.5 w-3.5 text-warning-foreground" />
          <span className="font-semibold text-foreground">{MOCK.temp}°C</span>
        </button>
      </PopoverTrigger>
      <PopoverContent side="bottom" align="center" className="w-56 p-3">
        <div className="space-y-3">
          <div className="text-center">
            <p className="text-sm font-semibold text-foreground">{MOCK.location}</p>
            <p className="text-xs text-muted-foreground">{MOCK.condition}</p>
            <p className="text-2xl font-bold text-foreground mt-1">{MOCK.temp}°C</p>
            <p className="text-xs text-muted-foreground">H: {MOCK.high}° · T: {MOCK.low}°</p>
          </div>
          <div className="border-t border-border pt-2 grid grid-cols-3 gap-2 text-center">
            <div>
              <Droplets className="h-3 w-3 mx-auto text-muted-foreground mb-0.5" />
              <p className="text-[10px] text-muted-foreground">Feuchte</p>
              <p className="text-xs font-medium">{MOCK.humidity}%</p>
            </div>
            <div>
              <Wind className="h-3 w-3 mx-auto text-muted-foreground mb-0.5" />
              <p className="text-[10px] text-muted-foreground">Wind</p>
              <p className="text-xs font-medium">{MOCK.wind} km/h</p>
            </div>
            <div>
              <Eye className="h-3 w-3 mx-auto text-muted-foreground mb-0.5" />
              <p className="text-[10px] text-muted-foreground">Sicht</p>
              <p className="text-xs font-medium">{MOCK.visibility} km</p>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
