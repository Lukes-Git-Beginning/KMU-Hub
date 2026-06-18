import { useTranslation } from 'react-i18next'
import { CheckCircle2, Clock, XCircle, Send, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMyWeekStatus, useSubmitWeek } from '@/api/hooks/hr-hooks'

interface ApprovalBannerProps {
  weekStart: string // YYYY-MM-DD (Monday)
}

export default function ApprovalBanner({ weekStart }: ApprovalBannerProps) {
  const { t } = useTranslation()
  const { data: weekStatus } = useMyWeekStatus(weekStart)
  const submitWeek = useSubmitWeek()

  // Map backend status (open/submitted/approved/rejected) to display state
  const rawStatus = weekStatus?.status ?? 'open'
  const status =
    rawStatus === 'open' ? 'draft'
    : rawStatus === 'submitted' ? 'pending'
    : (rawStatus as 'approved' | 'rejected')

  const handleSubmit = () => submitWeek.mutate(weekStart)
  const handleResubmit = () => submitWeek.mutate(weekStart)

  if (status === 'draft') {
    return (
      <div className="rounded-lg border border-border bg-card p-4 flex items-center gap-3">
        <Clock className="h-5 w-5 text-muted-foreground shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-medium text-foreground">
            {t('profil.zeiterfassung.approval.draftTitle')}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t('profil.zeiterfassung.approval.draftDescription')}
          </p>
        </div>
        <Button size="sm" onClick={handleSubmit} disabled={submitWeek.isPending} className="gap-1.5">
          <Send className="h-3.5 w-3.5" />
          {t('profil.zeiterfassung.approval.submit')}
        </Button>
      </div>
    )
  }

  if (status === 'pending') {
    return (
      <div className="rounded-lg border border-warning/30 bg-warning-light p-4 flex items-center gap-3">
        <Clock className="h-5 w-5 text-warning-foreground shrink-0 animate-pulse" />
        <div className="flex-1">
          <p className="text-sm font-medium text-warning-foreground">
            {t('profil.zeiterfassung.approval.pendingTitle')}
          </p>
          {weekStatus?.submittedAt && (
            <p className="text-xs text-warning-foreground mt-0.5">
              {t('profil.zeiterfassung.approval.submittedAt')}{' '}
              {new Date(weekStatus.submittedAt).toLocaleDateString('de-DE', {
                day: '2-digit',
                month: '2-digit',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </p>
          )}
        </div>
      </div>
    )
  }

  if (status === 'approved') {
    return (
      <div className="rounded-lg border border-success/30 bg-success-light p-4 flex items-center gap-3">
        <CheckCircle2 className="h-5 w-5 text-success shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-medium text-success">
            {t('profil.zeiterfassung.approval.approvedTitle')}
            {weekStatus?.approvedBy && ` ${t('profil.zeiterfassung.approval.by')} ${weekStatus.approvedBy}`}
          </p>
        </div>
      </div>
    )
  }

  // status === 'rejected'
  return (
    <div className="rounded-lg border border-destructive/30 bg-error-light p-4 flex items-center gap-3">
      <XCircle className="h-5 w-5 text-destructive shrink-0" />
      <div className="flex-1">
        <p className="text-sm font-medium text-destructive">
          {t('profil.zeiterfassung.approval.rejectedTitle')}
          {weekStatus?.approvedBy && ` ${t('profil.zeiterfassung.approval.by')} ${weekStatus.approvedBy}`}
        </p>
        {weekStatus?.rejectionReason && (
          <p className="text-xs text-destructive mt-0.5">
            {t('profil.zeiterfassung.reason')}: {weekStatus.rejectionReason}
          </p>
        )}
      </div>
      <Button
        size="sm"
        variant="outline"
        onClick={handleResubmit}
        disabled={submitWeek.isPending}
        className="gap-1.5 border-destructive/30 hover:bg-error-light"
      >
        <RotateCcw className="h-3.5 w-3.5" />
        {t('profil.zeiterfassung.approval.resubmit')}
      </Button>
    </div>
  )
}
