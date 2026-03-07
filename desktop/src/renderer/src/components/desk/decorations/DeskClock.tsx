/**
 * Clock decoration for the desk frame.
 *
 * Supports two variants:
 * - "analog": Pure CSS/SVG clock with smooth hand rotation
 * - "digital": LED-style digital display
 *
 * Click toggles between variants (persisted via callback).
 */
import { useState, useEffect } from 'react'
import { useUIStore } from '@/stores/ui'

interface DeskClockProps {
  size?: number
}

function getHandAngles() {
  const now = new Date()
  const seconds = now.getSeconds()
  const minutes = now.getMinutes()
  const hours = now.getHours() % 12

  return {
    second: seconds * 6,
    minute: minutes * 6 + seconds * 0.1,
    hour: hours * 30 + minutes * 0.5,
  }
}

function getTimeStr() {
  const now = new Date()
  return {
    hours: String(now.getHours()).padStart(2, '0'),
    minutes: String(now.getMinutes()).padStart(2, '0'),
    seconds: String(now.getSeconds()).padStart(2, '0'),
  }
}

function AnalogClock({ size }: { size: number }) {
  const [angles, setAngles] = useState(getHandAngles)

   
  useEffect(() => {
    const interval = setInterval(() => setAngles(getHandAngles()), 1000)
    return () => clearInterval(interval)
  }, [])

  const cx = 50
  const cy = 50

  return (
    <svg width={size} height={size} viewBox="0 0 100 100" className="drop-shadow-sm">
      <circle cx={cx} cy={cy} r="46" fill="hsl(40 30% 95%)" stroke="hsl(30 20% 60%)" strokeWidth="3" />

      {Array.from({ length: 12 }, (_, i) => {
        const angle = (i * 30 - 90) * (Math.PI / 180)
        return (
          <line
            key={i}
            x1={cx + 36 * Math.cos(angle)}
            y1={cy + 36 * Math.sin(angle)}
            x2={cx + 42 * Math.cos(angle)}
            y2={cy + 42 * Math.sin(angle)}
            stroke="hsl(30 15% 40%)"
            strokeWidth={i % 3 === 0 ? 2.5 : 1.5}
            strokeLinecap="round"
          />
        )
      })}

      <line x1={cx} y1={cy} x2={cx} y2={cy - 24} stroke="hsl(25 20% 25%)" strokeWidth="3" strokeLinecap="round"
        style={{ transform: `rotate(${angles.hour}deg)`, transformOrigin: `${cx}px ${cy}px`, transition: 'transform 0.3s ease' }} />
      <line x1={cx} y1={cy} x2={cx} y2={cy - 34} stroke="hsl(25 20% 30%)" strokeWidth="2" strokeLinecap="round"
        style={{ transform: `rotate(${angles.minute}deg)`, transformOrigin: `${cx}px ${cy}px`, transition: 'transform 0.3s ease' }} />
      <line x1={cx} y1={cy + 8} x2={cx} y2={cy - 36} stroke="hsl(0 60% 45%)" strokeWidth="1" strokeLinecap="round"
        style={{ transform: `rotate(${angles.second}deg)`, transformOrigin: `${cx}px ${cy}px` }} />
      <circle cx={cx} cy={cy} r="3" fill="hsl(25 20% 30%)" />
    </svg>
  )
}

function DigitalClock({ size }: { size: number }) {
  const [time, setTime] = useState(getTimeStr)

   
  useEffect(() => {
    const interval = setInterval(() => setTime(getTimeStr()), 1000)
    return () => clearInterval(interval)
  }, [])

  const scale = size / 64

  return (
    <div
      className="flex items-center justify-center select-none"
      style={{
        width: size,
        height: size,
        borderRadius: `${4 * scale}px`,
        background: 'hsl(25 15% 18%)',
        boxShadow: `0 ${2 * scale}px ${6 * scale}px rgba(0,0,0,0.2)`,
        border: `${1.5 * scale}px solid hsl(30 20% 35%)`,
      }}
    >
      <div className="text-center">
        <span
          className="font-mono tabular-nums"
          style={{
            fontSize: `${14 * scale}px`,
            color: 'hsl(120 80% 65%)',
            textShadow: '0 0 6px hsl(120 80% 50% / 0.4)',
            letterSpacing: `${1 * scale}px`,
          }}
        >
          {time.hours}:{time.minutes}
        </span>
        <span
          className="font-mono tabular-nums block"
          style={{
            fontSize: `${8 * scale}px`,
            color: 'hsl(120 50% 45%)',
            marginTop: `${-1 * scale}px`,
          }}
        >
          :{time.seconds}
        </span>
      </div>
    </div>
  )
}

export function DeskClock({ size = 64 }: DeskClockProps) {
  const decoration = useUIStore((s) => s.deskDecorations['left-wall-clock'])
  const setDecoration = useUIStore((s) => s.setDeskDecoration)
  const variant = decoration?.variant ?? 'analog'

  const toggleVariant = () => {
    const newVariant = variant === 'analog' ? 'digital' : 'analog'
    setDecoration('left-wall-clock', {
      slotId: 'left-wall-clock',
      type: 'clock',
      variant: newVariant,
    })
  }

  return (
    <button
      onClick={toggleVariant}
      className="cursor-pointer hover:scale-105 transition-transform"
      title="Klick: Analog/Digital wechseln"
    >
      {variant === 'digital' ? (
        <DigitalClock size={size} />
      ) : (
        <AnalogClock size={size} />
      )}
    </button>
  )
}
