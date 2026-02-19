import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Calendar, Clock, CheckCircle2, XCircle, AlertTriangle, Loader2 } from 'lucide-react'
import { useLeaveRequests, useApproveLeaveRequest, useRejectLeaveRequest } from '@/api/hooks/hr-hooks'
import type { LeaveRequest } from '@/api/hr-types'

const leaveTypeColors: Record<string, string> = {
  urlaub: 'bg-info-light text-info',
  krankheit: 'bg-error-light text-error',
  sonderurlaub: 'bg-primary-light text-primary',
  elternzeit: 'bg-success-light text-success',
  unbezahlt: 'bg-warning-light text-warning',
}

const statusColors: Record<string, string> = {
  pending: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
  cancelled: 'bg-secondary text-muted-foreground',
}

const statusLabels: Record<string, string> = {
  pending: 'Ausstehend',
  approved: 'Genehmigt',
  rejected: 'Abgelehnt',
  cancelled: 'Storniert',
}

interface HRApprovalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: LeaveRequest | null
}

export function HRApprovalDialog({
  open,
  onOpenChange,
  request,
}: HRApprovalDialogProps) {
  const [comment, setComment] = useState('')

  const approveMutation = useApproveLeaveRequest()
  const rejectMutation = useRejectLeaveRequest()

  // Check for overlapping requests when approving
  const { data: allPendingData } = useLeaveRequests({ status: 'approved' })
  const approvedRequests = allPendingData?.requests ?? []

  if (!request) return null

  // Find overlapping approved requests for the same employee
  const overlapping = approvedRequests.filter(
    (r) =>
      r.employeeId === request.employeeId &&
      r.id !== request.id &&
      r.startDate <= request.endDate &&
      r.endDate >= request.startDate,
  )

  const handleApprove = () => {
    approveMutation.mutate(
      { id: request.id, data: { comment: comment.trim() || undefined } },
      {
        onSuccess: () => {
          setComment('')
          onOpenChange(false)
        },
      },
    )
  }

  const handleReject = () => {
    rejectMutation.mutate(
      { id: request.id, data: { comment: comment.trim() || undefined } },
      {
        onSuccess: () => {
          setComment('')
          onOpenChange(false)
        },
      },
    )
  }

  const isPending = approveMutation.isPending || rejectMutation.isPending

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) setComment(''); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Antrag bearbeiten</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Request info */}
          <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
                {request.employeeName
                  ?.split(' ')
                  .map((n) => n[0])
                  .join('')
                  .slice(0, 2)
                  .toUpperCase() ?? '??'}
              </div>
              <div>
                <p className="text-sm font-medium text-foreground">
                  {request.employeeName ?? 'Unbekannt'}
                </p>
                <div className="flex items-center gap-2 mt-0.5">
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                    leaveTypeColors[request.leaveType?.key ?? ''] ?? 'bg-secondary text-muted-foreground'
                  }`}>
                    {request.leaveType?.name ?? 'Abwesenheit'}
                  </span>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusColors[request.status]}`}>
                    {statusLabels[request.status]}
                  </span>
                </div>
              </div>
            </div>

            <div className="space-y-1 text-xs text-muted-foreground">
              <div className="flex items-center gap-2">
                <Calendar className="h-3.5 w-3.5" />
                <span>
                  {new Date(request.startDate).toLocaleDateString('de-DE')}
                  {request.startDate !== request.endDate && ` – ${new Date(request.endDate).toLocaleDateString('de-DE')}`}
                  {request.isHalfDayStart && ` (${request.halfDayPeriodStart === 'morning' ? 'Vormittag' : 'Nachmittag'})`}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="h-3.5 w-3.5" />
                <span>{request.totalDays} {request.totalDays === 1 ? 'Tag' : 'Tage'}</span>
              </div>
            </div>

            {request.reason && (
              <p className="text-sm text-foreground">{request.reason}</p>
            )}
          </div>

          {/* Overlap warning */}
          {overlapping.length > 0 && (
            <div className="rounded-lg border border-warning bg-warning-light/30 p-3 flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
              <div>
                <p className="text-xs font-medium text-warning">Ueberlappende Abwesenheit</p>
                {overlapping.map((o) => (
                  <p key={o.id} className="text-xs text-muted-foreground mt-0.5">
                    {o.leaveType?.name ?? 'Abwesenheit'}: {new Date(o.startDate).toLocaleDateString('de-DE')} – {new Date(o.endDate).toLocaleDateString('de-DE')}
                  </p>
                ))}
              </div>
            </div>
          )}

          {/* Comment */}
          <div className="space-y-1.5">
            <Label>Kommentar (optional)</Label>
            <Textarea
              placeholder="Anmerkung fuer den Mitarbeiter..."
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter className="flex gap-2 sm:gap-2">
          <Button variant="outline" onClick={() => { setComment(''); onOpenChange(false) }} className="flex-1" disabled={isPending}>
            Abbrechen
          </Button>
          <Button
            onClick={handleReject}
            variant="outline"
            className="flex-1 border-red-200 text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
            disabled={isPending}
          >
            {rejectMutation.isPending ? (
              <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
            ) : (
              <XCircle className="mr-1.5 h-4 w-4" />
            )}
            Ablehnen
          </Button>
          <Button onClick={handleApprove} className="flex-1 bg-success hover:bg-success/90 text-white" disabled={isPending}>
            {approveMutation.isPending ? (
              <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
            ) : (
              <CheckCircle2 className="mr-1.5 h-4 w-4" />
            )}
            Genehmigen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
