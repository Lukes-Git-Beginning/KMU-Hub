import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib'

export interface InlineStatProps {
  label: string
  value: string | number
  hint?: string
  /** Akzent-Farbe (success für positive Trends, warning für ⚠, error für kritisch) */
  accent?: 'default' | 'success' | 'warning' | 'error'
  /** Optional: Icon links neben dem Label (subtle, h-3 w-3) */
  icon?: LucideIcon
  /** min-width für rhythmischen Block-Abstand in flex-wrap */
  className?: string
}

const accentClass: Record<NonNullable<InlineStatProps['accent']>, string> = {
  default: 'text-foreground',
  success: 'text-[var(--success)]',
  warning: 'text-[var(--warning)]',
  error: 'text-[var(--error)]',
}

export function InlineStat({
  label,
  value,
  hint,
  accent = 'default',
  icon: Icon,
  className,
}: InlineStatProps) {
  return (
    <div className={cn('min-w-[140px]', className)}>
      <dt className="flex items-center gap-1 text-[11px] uppercase tracking-wider text-muted-foreground">
        {Icon && <Icon className="h-3 w-3" aria-hidden="true" />}
        {label}
      </dt>
      <dd className={cn('mt-0.5 text-2xl font-semibold tabular-nums leading-none', accentClass[accent])}>
        {value}
      </dd>
      {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}
