/**
 * Tear-off daily calendar decoration for the desk frame.
 *
 * Styled like a physical Abreisskalender — a small block with
 * the weekday on a colored header strip and the date/month below.
 * Updates automatically at midnight.
 *
 * Click navigates to /kalender.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

interface DeskCalendarProps {
  size?: number
}

const WEEKDAYS_SHORT = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']
const MONTHS_SHORT = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']

function getDateInfo() {
  const now = new Date()
  return {
    day: now.getDate(),
    weekdayShort: WEEKDAYS_SHORT[now.getDay()],
    month: MONTHS_SHORT[now.getMonth()],
    isWeekend: now.getDay() === 0 || now.getDay() === 6,
  }
}

export function DeskCalendar({ size = 64 }: DeskCalendarProps) {
  const [dateInfo, setDateInfo] = useState(getDateInfo)
  const navigate = useNavigate()

  useEffect(() => {
    const now = new Date()
    const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
    const msUntilMidnight = tomorrow.getTime() - now.getTime()

    const timeout = setTimeout(() => {
      setDateInfo(getDateInfo())
    }, msUntilMidnight)

    return () => clearTimeout(timeout)
  }, [dateInfo])

  const scale = size / 64

  return (
    <button
      onClick={() => navigate('/kalender')}
      className="select-none cursor-pointer hover:scale-105 transition-transform"
      title="Klick: Kalender öffnen"
      style={{ width: size, height: size * 1.15 }}
    >
      <div
        className="relative overflow-hidden"
        style={{
          width: size,
          height: size * 1.15,
          borderRadius: `${3 * scale}px`,
          boxShadow: `0 ${2 * scale}px ${6 * scale}px rgba(0,0,0,0.15), 0 ${1 * scale}px ${2 * scale}px rgba(0,0,0,0.1)`,
        }}
      >
        {/* Binding strip */}
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            height: `${4 * scale}px`,
            background: 'hsl(30 20% 55%)',
            borderRadius: `${3 * scale}px ${3 * scale}px 0 0`,
          }}
        />

        {/* Weekday header */}
        <div
          style={{
            position: 'relative',
            background: 'hsl(0 55% 48%)',
            paddingTop: `${6 * scale}px`,
            paddingBottom: `${3 * scale}px`,
            textAlign: 'center',
          }}
        >
          <span
            style={{
              display: 'block',
              color: 'white',
              fontSize: `${8 * scale}px`,
              fontWeight: 600,
              letterSpacing: `${0.5 * scale}px`,
              textTransform: 'uppercase',
              lineHeight: 1,
            }}
          >
            {dateInfo.weekdayShort}
          </span>
        </div>

        {/* Date body */}
        <div
          style={{
            background: 'hsl(40 30% 97%)',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            flex: 1,
            paddingTop: `${2 * scale}px`,
            paddingBottom: `${1 * scale}px`,
          }}
        >
          <span
            style={{
              display: 'block',
              color: 'hsl(25 20% 20%)',
              fontSize: `${22 * scale}px`,
              fontWeight: 700,
              lineHeight: 1,
            }}
          >
            {dateInfo.day}
          </span>
          <span
            style={{
              display: 'block',
              color: 'hsl(25 15% 45%)',
              fontSize: `${7 * scale}px`,
              fontWeight: 500,
              letterSpacing: `${0.5 * scale}px`,
              textTransform: 'uppercase',
              marginTop: `${2 * scale}px`,
              lineHeight: 1,
            }}
          >
            {dateInfo.month}
          </span>
        </div>

        {/* Torn bottom edge */}
        <div
          style={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            height: `${5 * scale}px`,
            background: `linear-gradient(135deg, hsl(40 30% 97%) 33.33%, transparent 33.33%) 0 0 / ${4 * scale}px ${5 * scale}px, linear-gradient(225deg, hsl(40 30% 97%) 33.33%, transparent 33.33%) 0 0 / ${4 * scale}px ${5 * scale}px`,
            backgroundPosition: `0 0, ${2 * scale}px 0`,
          }}
        />

        {/* Page stack shadows */}
        <div
          style={{
            position: 'absolute',
            bottom: `${1 * scale}px`,
            left: `${2 * scale}px`,
            right: `${2 * scale}px`,
            height: `${2 * scale}px`,
            background: 'hsl(30 15% 88%)',
            borderRadius: `0 0 ${2 * scale}px ${2 * scale}px`,
            zIndex: -1,
          }}
        />
        <div
          style={{
            position: 'absolute',
            bottom: `${-1 * scale}px`,
            left: `${4 * scale}px`,
            right: `${4 * scale}px`,
            height: `${2 * scale}px`,
            background: 'hsl(30 15% 82%)',
            borderRadius: `0 0 ${2 * scale}px ${2 * scale}px`,
            zIndex: -2,
          }}
        />
      </div>
    </button>
  )
}
