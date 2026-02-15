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
import { Calendar, Clock, CheckCircle2, XCircle } from 'lucide-react'
import type { HRRequest } from '@/stores/team'

const requestTypeLabels: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krankheit',
  overtime: 'Überstunden',
  doctor: 'Arzttermin',
  homeoffice: 'Homeoffice',
  education: 'Weiterbildung',
}

const requestTypeColors: Record<string, string> = {
  vacation: 'bg-info-light text-info',
  sick: 'bg-error-light text-error',
  overtime: 'bg-warning-light text-warning',
  doctor: 'bg-primary-light text-primary',
  homeoffice: 'bg-success-light text-success',
  education: 'bg-primary-light text-primary',
}

interface HRApprovalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: HRRequest | null
  onApprove: (id: string, comment?: string) => void
  onReject: (id: string, comment?: string) => void
}

export function HRApprovalDialog({
  open,
  onOpenChange,
  request,
  onApprove,
  onReject,
}: HRApprovalDialogProps) {
  const [comment, setComment] = useState('')

  if (!request) return null

  const handleApprove = () => {
    onApprove(request.id, comment.trim() || undefined)
    setComment('')
    onOpenChange(false)
  }

  const handleReject = () => {
    onReject(request.id, comment.trim() || undefined)
    setComment('')
    onOpenChange(false)
  }

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
                {request.memberInitials}
              </div>
              <div>
                <p className="text-sm font-medium text-foreground">{request.memberName}</p>
                <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${requestTypeColors[request.type]}`}>
                  {requestTypeLabels[request.type]}
                </span>
              </div>
            </div>

            <div className="space-y-1 text-xs text-muted-foreground">
              <div className="flex items-center gap-2">
                <Calendar className="h-3.5 w-3.5" />
                <span>
                  {new Date(request.startDate).toLocaleDateString('de-CH')}
                  {request.startDate !== request.endDate && ` – ${new Date(request.endDate).toLocaleDateString('de-CH')}`}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="h-3.5 w-3.5" />
                <span>{request.days} {request.days === 1 ? 'Tag' : 'Tage'}</span>
              </div>
            </div>

            <p className="text-sm text-foreground">{request.reason}</p>
          </div>

          {/* Comment */}
          <div className="space-y-1.5">
            <Label>Kommentar (optional)</Label>
            <Textarea
              placeholder="Anmerkung für den Mitarbeiter..."
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter className="flex gap-2 sm:gap-2">
          <Button variant="outline" onClick={() => { setComment(''); onOpenChange(false) }} className="flex-1">
            Abbrechen
          </Button>
          <Button
            onClick={handleReject}
            variant="outline"
            className="flex-1 border-red-200 text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            <XCircle className="mr-1.5 h-4 w-4" />
            Ablehnen
          </Button>
          <Button onClick={handleApprove} className="flex-1 bg-success hover:bg-success/90 text-white">
            <CheckCircle2 className="mr-1.5 h-4 w-4" />
            Genehmigen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
