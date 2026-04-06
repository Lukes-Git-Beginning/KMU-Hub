/**
 * Embedded clock + tear-off calendar for the header bar.
 *
 * Shows digital time on the left and a mini Abreisskalender on the right,
 * styled to look like physical objects sitting in the header.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

const WEEKDAY_KEYS = [
  'header.clock.dayShort.sun',
  'header.clock.dayShort.mon',
  'header.clock.dayShort.tue',
  'header.clock.dayShort.wed',
  'header.clock.dayShort.thu',
  'header.clock.dayShort.fri',
  'header.clock.dayShort.sat',
]

const MONTH_KEYS = [
  'header.clock.monthShort.jan',
  'header.clock.monthShort.feb',
  'header.clock.monthShort.mar',
  'header.clock.monthShort.apr',
  'header.clock.monthShort.may',
  'header.clock.monthShort.jun',
  'header.clock.monthShort.jul',
  'header.clock.monthShort.aug',
  'header.clock.monthShort.sep',
  'header.clock.monthShort.oct',
  'header.clock.monthShort.nov',
  'header.clock.monthShort.dec',
]

function getRawTimeInfo() {
  const now = new Date()
  return {
    hours: String(now.getHours()).padStart(2, '0'),
    minutes: String(now.getMinutes()).padStart(2, '0'),
    day: now.getDate(),
    weekdayIndex: now.getDay(),
    monthIndex: now.getMonth(),
  }
}

export function HeaderClock() {
  const { t } = useTranslation()
  const [time, setTime] = useState(getRawTimeInfo)

   
  useEffect(() => {
    const interval = setInterval(() => setTime(getRawTimeInfo()), 1000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="flex items-center gap-2.5">
      {/* Digital clock */}
      <div className="text-right leading-none select-none">
        <span className="text-lg font-semibold tabular-nums tracking-tight text-foreground">
          {time.hours}
          <span className="animate-pulse-live">:</span>
          {time.minutes}
        </span>
      </div>

      {/* Divider */}
      <div className="h-8 w-px bg-border" />

      {/* Mini tear-off calendar */}
      <div
        className="relative select-none overflow-hidden rounded-[3px]"
        style={{
          width: 36,
          height: 42,
          boxShadow: '0 1px 4px rgba(0,0,0,0.12), 0 0.5px 1px rgba(0,0,0,0.08)',
        }}
      >
        {/* Binding strip */}
        <div
          className="h-[3px]"
          style={{ background: 'hsl(30 20% 55%)' }}
        />

        {/* Weekday header — classic red */}
        <div
          className="flex items-center justify-center py-[2px]"
          style={{ background: 'hsl(0 55% 48%)' }}
        >
          <span className="text-[8px] font-bold uppercase tracking-wider text-white leading-none">
            {t(WEEKDAY_KEYS[time.weekdayIndex])}
          </span>
        </div>

        {/* Date body */}
        <div
          className="flex flex-col items-center justify-center"
          style={{ background: 'hsl(40 30% 97%)', paddingTop: 1, paddingBottom: 2 }}
        >
          <span
            className="text-[16px] font-bold leading-none"
            style={{ color: 'hsl(25 20% 20%)' }}
          >
            {time.day}
          </span>
          <span
            className="text-[7px] font-medium uppercase tracking-wider leading-none mt-[2px]"
            style={{ color: 'hsl(25 15% 50%)' }}
          >
            {t(MONTH_KEYS[time.monthIndex])}
          </span>
        </div>

        {/* Torn bottom edge */}
        <div
          className="absolute bottom-0 left-0 right-0 h-[3px]"
          style={{
            background: 'linear-gradient(135deg, hsl(40 30% 97%) 33%, transparent 33%) 0 0 / 3px 3px, linear-gradient(225deg, hsl(40 30% 97%) 33%, transparent 33%) 0 0 / 3px 3px',
            backgroundPosition: '0 0, 1.5px 0',
          }}
        />

        {/* Stacked pages behind */}
        <div
          className="absolute -bottom-[1px] left-[2px] right-[2px] h-[1.5px] -z-10 rounded-b-sm"
          style={{ background: 'hsl(30 15% 85%)' }}
        />
        <div
          className="absolute -bottom-[2.5px] left-[4px] right-[4px] h-[1.5px] -z-20 rounded-b-sm"
          style={{ background: 'hsl(30 15% 80%)' }}
        />
      </div>
    </div>
  )
}
