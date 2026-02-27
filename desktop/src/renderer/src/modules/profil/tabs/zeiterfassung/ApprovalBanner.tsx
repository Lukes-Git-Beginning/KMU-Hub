import { CheckCircle2, Clock, XCircle, Send, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { toast } from 'sonner'

interface ApprovalBannerProps {
  weekStart: string // YYYY-MM-DD (Monday)
}

export default function ApprovalBanner({ weekStart }: ApprovalBannerProps) {
  const weekApprovals = useTimeTrackingStore((s) => s.weekApprovals)
  const submitWeekReport = useTimeTrackingStore((s) => s.submitWeekReport)

  const approval = weekApprovals.find((a) => a.weekStart === weekStart)
  const status = approval?.status ?? 'draft'

  const handleSubmit = () => {
    submitWeekReport(weekStart)
    toast.success('Wochenrapport eingereicht')
  }

  const handleResubmit = () => {
    submitWeekReport(weekStart)
    toast.success('Wochenrapport erneut eingereicht')
  }

  if (status === 'draft') {
    return (
      <div className="rounded-lg border border-border bg-card p-4 flex items-center gap-3">
        <Clock className="h-5 w-5 text-muted-foreground shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-medium text-foreground">
            Wochenrapport noch nicht eingereicht
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            Bitte ueberpruefen und einreichen
          </p>
        </div>
        <Button size="sm" onClick={handleSubmit} className="gap-1.5">
          <Send className="h-3.5 w-3.5" />
          Einreichen
        </Button>
      </div>
    )
  }

  if (status === 'pending') {
    return (
      <div className="rounded-lg border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/30 p-4 flex items-center gap-3">
        <Clock className="h-5 w-5 text-amber-600 dark:text-amber-400 shrink-0 animate-pulse" />
        <div className="flex-1">
          <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
            Wochenrapport eingereicht — warte auf Genehmigung
          </p>
          {approval?.submittedAt && (
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-0.5">
              Eingereicht am {new Date(approval.submittedAt).toLocaleDateString('de-DE', {
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
      <div className="rounded-lg border border-emerald-300 dark:border-emerald-700 bg-emerald-50 dark:bg-emerald-950/30 p-4 flex items-center gap-3">
        <CheckCircle2 className="h-5 w-5 text-emerald-600 dark:text-emerald-400 shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
            Wochenrapport genehmigt
            {approval?.reviewedBy && ` von ${approval.reviewedBy}`}
          </p>
          {approval?.reviewedAt && (
            <p className="text-xs text-emerald-600 dark:text-emerald-400 mt-0.5">
              Genehmigt am {new Date(approval.reviewedAt).toLocaleDateString('de-DE', {
                day: '2-digit',
                month: '2-digit',
                year: 'numeric',
              })}
            </p>
          )}
        </div>
      </div>
    )
  }

  // status === 'rejected'
  return (
    <div className="rounded-lg border border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-950/30 p-4 flex items-center gap-3">
      <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 shrink-0" />
      <div className="flex-1">
        <p className="text-sm font-medium text-red-800 dark:text-red-300">
          Wochenrapport abgelehnt
          {approval?.reviewedBy && ` von ${approval.reviewedBy}`}
        </p>
        {approval?.comment && (
          <p className="text-xs text-red-600 dark:text-red-400 mt-0.5">
            Grund: {approval.comment}
          </p>
        )}
      </div>
      <Button size="sm" variant="outline" onClick={handleResubmit} className="gap-1.5 border-red-300 dark:border-red-700 hover:bg-red-100 dark:hover:bg-red-900/30">
        <RotateCcw className="h-3.5 w-3.5" />
        Erneut einreichen
      </Button>
    </div>
  )
}
