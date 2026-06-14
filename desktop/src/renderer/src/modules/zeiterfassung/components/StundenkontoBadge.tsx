/**
 * Stundenkonto badge — cumulative flextime/overtime balance.
 *
 * Card variant: used in the module shell header.
 * API-backed via useTimeBalance (HR work-time balance endpoint).
 */
import { useTranslation } from 'react-i18next'
import { Scale } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTimeBalance } from '@/api/hooks/hr-hooks'
import { formatSignedMinutes } from '@/lib/worktime'

export function StundenkontoBadge() {
  const { t } = useTranslation()
  const { data: balance } = useTimeBalance()

  if (!balance) return null

  const min = balance.balanceMinutes
  const tone =
    min > 0 ? 'text-success' : min < 0 ? 'text-destructive' : 'text-muted-foreground'

  return (
    <div className="flex items-center gap-2.5 rounded-xl border border-border bg-card px-3.5 py-2">
      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary">
        <Scale className="h-4 w-4 text-muted-foreground" />
      </div>
      <div className="leading-tight">
        <p className="text-[10px] uppercase tracking-wider text-muted-foreground">
          {t('zeiterfassung.balance.label')}
        </p>
        <p className={cn('font-mono text-sm font-semibold tabular-nums', tone)}>
          {formatSignedMinutes(min)}
        </p>
      </div>
    </div>
  )
}
