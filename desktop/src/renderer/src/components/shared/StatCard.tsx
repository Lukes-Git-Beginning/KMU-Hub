import { useEffect, useRef, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib'
import { useTiltEffect } from '@/hooks/useTiltEffect'

interface StatCardProps {
  /**
   * Numbers count up from 0 on mount. Strings (e.g. a "3:24" duration or a
   * formatted ratio) render as-is without the count-up — passing a string to a
   * numeric count-up would otherwise display NaN.
   */
  value: number | string
  label: string
  prefix?: string
  suffix?: string
  change?: { value: number; positive: boolean }
  icon?: LucideIcon
  className?: string
  /** Enable 3D tilt effect for large hero cards */
  hero?: boolean
}

export function StatCard({
  label,
  value,
  prefix = '',
  suffix = '',
  change,
  icon: Icon,
  className,
  hero = false,
}: StatCardProps) {
  const isNumeric = typeof value === 'number' && Number.isFinite(value)
  const [displayValue, setDisplayValue] = useState<number>(0)
  const tiltRef = useTiltEffect<HTMLDivElement>({ maxAngle: 4, scale: 1.02, enabled: hero })
  const plainRef = useRef<HTMLDivElement>(null)
  const animated = useRef(false)


  useEffect(() => {
    if (!isNumeric || animated.current) return
    animated.current = true

    const duration = 600
    const start = performance.now()

    const tick = (now: number) => {
      const elapsed = now - start
      const progress = Math.min(elapsed / duration, 1)
      // ease-out cubic
      const eased = 1 - Math.pow(1 - progress, 3)
      setDisplayValue(Math.round(eased * (value as number)))
      if (progress < 1) requestAnimationFrame(tick)
    }

    requestAnimationFrame(tick)
  }, [value, isNumeric])

  return (
    <div
      ref={hero ? tiltRef : plainRef}
      className={cn(
        'rounded-xl border bg-card p-5 shadow-sm animate-fade-up',
        hero ? 'tilt-card' : 'hover-lift',
        className
      )}
    >
      {hero && <div className="tilt-shine" />}
      <div className="flex items-center justify-between relative z-[2]">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        {Icon && (
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
            <Icon className="h-4.5 w-4.5 text-primary" />
          </div>
        )}
      </div>
      <div className="mt-3 flex items-baseline gap-1 relative z-[2]">
        <span className="text-3xl font-bold tracking-tight text-foreground">
          {prefix}{isNumeric ? displayValue.toLocaleString('de-DE') : value}{suffix}
        </span>
      </div>
      {change && (
        <p className={cn(
          'mt-1.5 text-xs font-medium relative z-[2]',
          change.positive ? 'text-[var(--success)]' : 'text-[var(--error)]'
        )}>
          {change.positive ? '+' : ''}{change.value}% zum Vormonat
        </p>
      )}
    </div>
  )
}
