/**
 * Week-submission banner shown above the weekly view. Lets the user submit the
 * current week for manager approval and reflects the approval status.
 */
import { useTranslation } from 'react-i18next'
import { Send, Clock, CheckCircle2, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useMyWeekStatus, useSubmitWeek } from '@/api/hooks/hr-hooks'
import { currentWeekStart } from './weekUtils'

export function WeekSubmitBanner() {
  const { t } = useTranslation()
  const weekStart = currentWeekStart()
  const { data: status } = useMyWeekStatus(weekStart)
  const submit = useSubmitWeek()

  const s = status?.status ?? 'open'

  const config = {
    open: { icon: Send, tone: 'border-border bg-card', text: 'zeiterfassung.week.openHint' },
    submitted: { icon: Clock, tone: 'border-warning/30 bg-warning-light/10', text: 'zeiterfassung.week.submittedHint' },
    approved: { icon: CheckCircle2, tone: 'border-success/30 bg-success-light/10', text: 'zeiterfassung.week.approvedHint' },
    rejected: { icon: AlertTriangle, tone: 'border-destructive/30 bg-destructive/5', text: 'zeiterfassung.week.rejectedHint' },
  }[s] ?? { icon: Send, tone: 'border-border bg-card', text: 'zeiterfassung.week.openHint' }

  const Icon = config.icon
  const iconTone =
    s === 'approved' ? 'text-success' : s === 'submitted' ? 'text-warning' : s === 'rejected' ? 'text-destructive' : 'text-muted-foreground'

  return (
    <div className={cn('mb-5 flex items-center justify-between gap-4 rounded-xl border px-4 py-3', config.tone)}>
      <div className="flex items-center gap-2.5">
        <Icon className={cn('h-4 w-4 shrink-0', iconTone)} />
        <p className="text-sm text-foreground">{t(config.text)}</p>
      </div>
      {(s === 'open' || s === 'rejected') && (
        <button
          onClick={() => submit.mutate(weekStart)}
          disabled={submit.isPending}
          className="flex shrink-0 items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          <Send className="h-3.5 w-3.5" />
          {t(s === 'rejected' ? 'zeiterfassung.week.resubmit' : 'zeiterfassung.week.submit')}
        </button>
      )}
    </div>
  )
}
