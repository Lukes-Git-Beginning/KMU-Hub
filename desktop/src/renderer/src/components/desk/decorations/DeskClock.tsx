/**
 * Analog clock decoration for the desk frame.
 *
 * Pure CSS/SVG implementation — no image assets needed.
 * Uses setInterval for 1-second updates and CSS transform
 * for smooth hand rotation (GPU-accelerated).
 */
import { useState, useEffect } from 'react'

interface DeskClockProps {
  size?: number
}

function getHandAngles() {
  const now = new Date()
  const seconds = now.getSeconds()
  const minutes = now.getMinutes()
  const hours = now.getHours() % 12

  return {
    second: seconds * 6,              // 360 / 60
    minute: minutes * 6 + seconds * 0.1, // smooth minute progression
    hour: hours * 30 + minutes * 0.5,    // smooth hour progression
  }
}

export function DeskClock({ size = 18 }: DeskClockProps) {
  const [angles, setAngles] = useState(getHandAngles)

  useEffect(() => {
    const interval = setInterval(() => {
      setAngles(getHandAngles())
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  const cx = 50
  const cy = 50

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      className="drop-shadow-sm"
    >
      {/* Clock face */}
      <circle
        cx={cx}
        cy={cy}
        r="46"
        fill="hsl(40 30% 95%)"
        stroke="hsl(30 20% 60%)"
        strokeWidth="3"
      />

      {/* Hour markers */}
      {Array.from({ length: 12 }, (_, i) => {
        const angle = (i * 30 - 90) * (Math.PI / 180)
        const inner = 36
        const outer = 42
        return (
          <line
            key={i}
            x1={cx + inner * Math.cos(angle)}
            y1={cy + inner * Math.sin(angle)}
            x2={cx + outer * Math.cos(angle)}
            y2={cy + outer * Math.sin(angle)}
            stroke="hsl(30 15% 40%)"
            strokeWidth={i % 3 === 0 ? 2.5 : 1.5}
            strokeLinecap="round"
          />
        )
      })}

      {/* Hour hand */}
      <line
        x1={cx}
        y1={cy}
        x2={cx}
        y2={cy - 24}
        stroke="hsl(25 20% 25%)"
        strokeWidth="3"
        strokeLinecap="round"
        style={{
          transform: `rotate(${angles.hour}deg)`,
          transformOrigin: `${cx}px ${cy}px`,
          transition: 'transform 0.3s ease',
        }}
      />

      {/* Minute hand */}
      <line
        x1={cx}
        y1={cy}
        x2={cx}
        y2={cy - 34}
        stroke="hsl(25 20% 30%)"
        strokeWidth="2"
        strokeLinecap="round"
        style={{
          transform: `rotate(${angles.minute}deg)`,
          transformOrigin: `${cx}px ${cy}px`,
          transition: 'transform 0.3s ease',
        }}
      />

      {/* Second hand */}
      <line
        x1={cx}
        y1={cy + 8}
        x2={cx}
        y2={cy - 36}
        stroke="hsl(0 60% 45%)"
        strokeWidth="1"
        strokeLinecap="round"
        style={{
          transform: `rotate(${angles.second}deg)`,
          transformOrigin: `${cx}px ${cy}px`,
        }}
      />

      {/* Center dot */}
      <circle cx={cx} cy={cy} r="3" fill="hsl(25 20% 30%)" />
    </svg>
  )
}
