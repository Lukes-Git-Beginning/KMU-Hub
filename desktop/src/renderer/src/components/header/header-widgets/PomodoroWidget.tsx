/**
 * Header mini-widget — simple Pomodoro focus timer.
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { Timer, Play, Pause, RotateCcw } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'

const WORK_MINUTES = 25
const BREAK_MINUTES = 5

export function PomodoroWidget() {
  const [secondsLeft, setSecondsLeft] = useState(WORK_MINUTES * 60)
  const [running, setRunning] = useState(false)
  const [isBreak, setIsBreak] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const reset = useCallback(() => {
    setRunning(false)
    setIsBreak(false)
    setSecondsLeft(WORK_MINUTES * 60)
    if (intervalRef.current) clearInterval(intervalRef.current)
  }, [])

   
  useEffect(() => {
    if (!running) {
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }

    intervalRef.current = setInterval(() => {
      setSecondsLeft((prev) => {
        if (prev <= 1) {
          // Switch mode
          setIsBreak((b) => !b)
          setRunning(false)
          return isBreak ? WORK_MINUTES * 60 : BREAK_MINUTES * 60
        }
        return prev - 1
      })
    }, 1000)

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [running, isBreak])

  const min = Math.floor(secondsLeft / 60)
  const sec = secondsLeft % 60
  const display = `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`

  const pct = isBreak
    ? ((BREAK_MINUTES * 60 - secondsLeft) / (BREAK_MINUTES * 60)) * 100
    : ((WORK_MINUTES * 60 - secondsLeft) / (WORK_MINUTES * 60)) * 100

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <button className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors ${
              running ? 'text-primary' : ''
            }`}>
              <Timer className={`h-3.5 w-3.5 ${running ? 'text-primary animate-pulse' : 'text-muted-foreground'}`} />
              <span className={`font-mono font-semibold ${running ? 'text-primary' : 'text-foreground'}`}>
                {display}
              </span>
            </button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent side="bottom">
          Pomodoro Timer — {isBreak ? 'Pause' : 'Fokus'}
        </TooltipContent>
      </Tooltip>

      <PopoverContent className="w-56 p-4" side="bottom" align="center">
        <div className="text-center">
          <p className="text-xs font-medium text-muted-foreground mb-2">
            {isBreak ? 'Pause' : 'Fokuszeit'}
          </p>

          {/* Progress ring (simplified as bar) */}
          <div className="h-1.5 w-full rounded-full bg-secondary mb-3 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-1000 ${isBreak ? 'bg-success' : 'bg-primary'}`}
              style={{ width: `${pct}%` }}
            />
          </div>

          <p className="text-3xl font-mono font-bold text-foreground mb-4">{display}</p>

          <div className="flex items-center justify-center gap-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={reset}
            >
              <RotateCcw className="h-3.5 w-3.5" />
            </Button>
            <Button
              size="icon"
              className="h-8 w-8"
              onClick={() => setRunning(!running)}
            >
              {running ? (
                <Pause className="h-3.5 w-3.5" />
              ) : (
                <Play className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
