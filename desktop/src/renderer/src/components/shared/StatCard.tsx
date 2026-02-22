import { useEffect, useRef, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib'

interface StatCardProps {
  label: string
  value: number
  prefix?: string
  suffix?: string
  change?: { value: number; positive: boolean }
  icon?: LucideIcon
  className?: string
}

export function StatCard({
  label,
  value,
  prefix = '',
  suffix = '',
  change,
  icon: Icon,
  className,
}: StatCardProps) {
  const [displayValue, setDisplayValue] = useState(0)
  const ref = useRef<HTMLDivElement>(null)
  const animated = useRef(false)

  useEffect(() => {
    if (animated.current) return
    animated.current = true

    const duration = 600
    const start = performance.now()

    const tick = (now: number) => {
      const elapsed = now - start
      const progress = Math.min(elapsed / duration, 1)
      // ease-out cubic
      const eased = 1 - Math.pow(1 - progress, 3)
      setDisplayValue(Math.round(eased * value))
      if (progress < 1) requestAnimationFrame(tick)
    }

    requestAnimationFrame(tick)
  }, [value])

  return (
    <div
      ref={ref}
      className={cn(
        'rounded-xl border bg-card p-5 shadow-sm transition-all duration-200 hover:shadow-md animate-fade-up',
        className
      )}
    >
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        {Icon && (
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
            <Icon className="h-4.5 w-4.5 text-primary" />
          </div>
        )}
      </div>
      <div className="mt-3 flex items-baseline gap-1">
        <span className="text-3xl font-bold tracking-tight text-foreground">
          {prefix}{displayValue.toLocaleString('de-DE')}{suffix}
        </span>
      </div>
      {change && (
        <p className={cn(
          'mt-1.5 text-xs font-medium',
          change.positive ? 'text-[var(--success)]' : 'text-[var(--error)]'
        )}>
          {change.positive ? '+' : ''}{change.value}% zum Vormonat
        </p>
      )}
    </div>
  )
}
