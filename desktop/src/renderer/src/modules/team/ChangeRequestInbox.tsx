/**
 * ChangeRequestInbox — HR view of pending profile change requests (R-4 P3).
 *
 * Gate: only accessible with `team:data_personal:edit` (all-scope = HR/admin).
 * Per-card: additional scoped check via useHrScopedCapability ensures the
 * viewer can actually edit the specific antragsteller's drawer.
 *
 * Layout: Diff-card Alt links | Neu rechts (BambooHR pattern).
 * Decided requests (approved/rejected) are collapsible below pending.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronUp,
  Loader2,
  ArrowRight,
} from 'lucide-react'
import { EmptyState } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { useHasCapability } from '@/hooks/useCapability'
import { useHrScopedCapability } from './useTeamPermissions'
import {
  useChangeRequests,
  useApproveChangeRequest,
  useRejectChangeRequest,
  type ProfileChangeRequest,
} from '@/api/hr-change-requests'
import { formatDate } from '@/lib/format'

// ============================================================
// Sub-component: single request card
// ============================================================

interface ChangeRequestCardProps {
  req: ProfileChangeRequest
  onApprove: (id: string) => void
  onReject: (req: ProfileChangeRequest) => void
  isApproving: boolean
}

function ChangeRequestCard({ req, onApprove, onReject, isApproving }: ChangeRequestCardProps) {
  const { t } = useTranslation()

  // Per-card scope check — can this viewer edit the antragsteller's data_personal?
  const canAct = useHrScopedCapability('team:data_personal:edit', req.userId)
  const isDone = req.status !== 'pending'

  return (
    <div className={`rounded-lg border bg-card p-4 transition-opacity ${isDone ? 'opacity-70' : ''} ${
      isDone ? 'border-border-muted' : 'border-border'
    }`}>
      {/* Header */}
      <div className="flex items-start justify-between gap-4 mb-3">
        <div>
          <p className="text-sm font-medium text-foreground">{req.userName}</p>
          <p className="text-xs text-muted-foreground">
            {req.fieldLabel} · {formatDate(req.createdAt)}
          </p>
        </div>
        <span className={`flex-shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
          req.status === 'pending' ? 'bg-warning-light text-warning' :
          req.status === 'approved' ? 'bg-success-light text-success' :
          req.status === 'cancelled' ? 'bg-secondary text-muted-foreground' :
          'bg-error-light text-error'
        }`}>
          {req.status === 'pending' ? t('team.changeRequest.statusPending') :
           req.status === 'approved' ? t('team.changeRequest.statusApproved') :
           req.status === 'cancelled' ? t('team.selfService.statusCancelled') :
           t('team.changeRequest.statusRejected')}
        </span>
      </div>

      {/* Diff: Alt links | Neu rechts */}
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 rounded-md bg-secondary/40 px-4 py-3 mb-3">
        <div>
          <p className="text-[10px] uppercase tracking-wide text-muted-foreground mb-0.5">{t('team.changeRequest.oldValue')}</p>
          <p className="text-sm text-foreground font-medium truncate">{req.oldValue || '—'}</p>
        </div>
        <ArrowRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
        <div>
          <p className="text-[10px] uppercase tracking-wide text-muted-foreground mb-0.5">{t('team.changeRequest.newValue')}</p>
          <p className="text-sm text-primary font-semibold truncate">{req.newValue}</p>
        </div>
      </div>

      {/* Rejection reason */}
      {req.status === 'rejected' && req.reason && (
        <p className="mb-3 rounded-md bg-error-light px-3 py-2 text-xs text-error">
          {t('team.changeRequest.rejectionReason')}: {req.reason}
        </p>
      )}

      {/* Decided-by */}
      {req.decidedByName && req.decidedAt && (
        <p className="mb-3 text-xs text-muted-foreground">
          {req.status === 'approved' ? t('team.changeRequest.approvedBy') : t('team.changeRequest.rejectedBy')}: {req.decidedByName} · {formatDate(req.decidedAt)}
        </p>
      )}

      {/* Actions */}
      {req.status === 'pending' && canAct && (
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            onClick={() => onApprove(req.id)}
            disabled={isApproving}
            className="bg-success text-white hover:bg-success/90 h-8 px-3 text-xs"
          >
            {isApproving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <CheckCircle2 className="h-3.5 w-3.5 mr-1" />}
            {t('team.changeRequest.approve')}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => onReject(req)}
            disabled={isApproving}
            className="h-8 px-3 text-xs text-error border-error/30 hover:bg-error-light"
          >
            <XCircle className="h-3.5 w-3.5 mr-1" />
            {t('team.changeRequest.reject')}
          </Button>
        </div>
      )}
    </div>
  )
}

// ============================================================
// Main component
// ============================================================

export function ChangeRequestInbox() {
  const { t } = useTranslation()
  const canAccess = useHasCapability('team:data_personal:edit')

  // Separate queries: pending inbox + historical
  const { data: pendingRequests = [], isLoading: loadingPending } =
    useChangeRequests('pending')
  const { data: decidedRequests = [], isLoading: loadingDecided } =
    useChangeRequests(undefined) // all — we filter below

  const decidedOnly = decidedRequests.filter((r) => r.status !== 'pending')

  const approveMutation = useApproveChangeRequest()
  const rejectMutation = useRejectChangeRequest()

  const [showDecided, setShowDecided] = useState(false)

  // Reject dialog state
  const [rejectTarget, setRejectTarget] = useState<ProfileChangeRequest | null>(null)
  const [rejectReason, setRejectReason] = useState('')

  const handleApprove = (id: string) => {
    approveMutation.mutate(id)
  }

  const openRejectDialog = (req: ProfileChangeRequest) => {
    setRejectTarget(req)
    setRejectReason('')
  }

  const submitReject = () => {
    if (!rejectTarget || !rejectReason.trim()) return
    rejectMutation.mutate(
      { id: rejectTarget.id, data: { reason: rejectReason.trim() } },
      { onSuccess: () => setRejectTarget(null) },
    )
  }

  if (!canAccess) {
    return null
  }

  const isLoading = loadingPending || loadingDecided

  return (
    <div className="space-y-4">
      {/* Section header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-foreground">{t('team.changeRequest.inboxTitle')}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t('team.changeRequest.inboxSubtitle', { count: pendingRequests.length })}
          </p>
        </div>
      </div>

      {/* Pending requests */}
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
        </div>
      ) : pendingRequests.length === 0 ? (
        <EmptyState
          icon={CheckCircle2}
          title={t('team.changeRequest.inboxEmpty')}
          description={t('team.changeRequest.inboxEmptyHint')}
        />
      ) : (
        <div className="space-y-3">
          {pendingRequests.map((req) => (
            <ChangeRequestCard
              key={req.id}
              req={req}
              onApprove={handleApprove}
              onReject={openRejectDialog}
              isApproving={approveMutation.isPending && approveMutation.variables === req.id}
            />
          ))}
        </div>
      )}

      {/* Decided requests — collapsible */}
      {decidedOnly.length > 0 && (
        <div>
          <button
            onClick={() => setShowDecided((v) => !v)}
            className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors py-2"
          >
            {showDecided ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
            {t('team.changeRequest.showDecided', { count: decidedOnly.length })}
          </button>
          {showDecided && (
            <div className="space-y-3 mt-2">
              {decidedOnly.map((req) => (
                <ChangeRequestCard
                  key={req.id}
                  req={req}
                  onApprove={handleApprove}
                  onReject={openRejectDialog}
                  isApproving={false}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Reject reason dialog */}
      <Dialog open={!!rejectTarget} onOpenChange={(o) => { if (!o) setRejectTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('team.changeRequest.rejectDialogTitle')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {rejectTarget && (
              <div className="rounded-md bg-secondary/50 px-3 py-2 text-sm">
                <span className="text-muted-foreground">{rejectTarget.userName} — {rejectTarget.fieldLabel}: </span>
                <span className="font-medium">{rejectTarget.newValue}</span>
              </div>
            )}
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('team.changeRequest.rejectionReasonLabel')} *
              </label>
              <textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                rows={3}
                placeholder={t('team.changeRequest.rejectionReasonPlaceholder')}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectTarget(null)}>{t('common.cancel')}</Button>
            <Button
              onClick={submitReject}
              disabled={!rejectReason.trim() || rejectMutation.isPending}
              className="bg-error text-white hover:bg-error/90"
            >
              {rejectMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {t('team.changeRequest.rejectConfirm')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
