import { useState } from 'react'
import {
  Clock,
  Check,
  X,
  AlertCircle,
  Loader2,
  Send,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { EmptyState } from '@/components/shared'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  useWorkTimeEntries,
  useSubmitCorrection,
  useApproveCorrection,
} from '@/api/hooks/hr-hooks'
import type { WorkTimeEntry } from '@/api/hr-types'

export function TimeCorrectionPanel() {
  const { data: entriesData, isLoading } = useWorkTimeEntries({ status: 'correction_pending' })
  const approveMutation = useApproveCorrection()
  const submitMutation = useSubmitCorrection()

  const [showSubmitDialog, setShowSubmitDialog] = useState(false)
  const [originalEntryId, setOriginalEntryId] = useState('')
  const [correctedClockIn, setCorrectedClockIn] = useState('')
  const [correctedClockOut, setCorrectedClockOut] = useState('')
  const [correctedBreakMinutes, setCorrectedBreakMinutes] = useState(0)
  const [reason, setReason] = useState('')

  const corrections = entriesData?.entries ?? []

  const handleSubmit = () => {
    if (!originalEntryId || !correctedClockIn || !correctedClockOut || !reason.trim()) {
      toast.error('Bitte alle Felder ausfuellen')
      return
    }
    submitMutation.mutate(
      {
        originalEntryId,
        correctedClockIn,
        correctedClockOut,
        correctedBreakMinutes,
        reason: reason.trim(),
      },
      {
        onSuccess: () => {
          setShowSubmitDialog(false)
          resetForm()
        },
      },
    )
  }

  const handleApprove = (id: string) => {
    approveMutation.mutate(id)
  }

  const resetForm = () => {
    setOriginalEntryId('')
    setCorrectedClockIn('')
    setCorrectedClockOut('')
    setCorrectedBreakMinutes(0)
    setReason('')
  }

  const formatTime = (iso?: string) => {
    if (!iso) return '--:--'
    return new Date(iso).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
  }

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <p className="text-sm text-muted-foreground">
          {corrections.length} offene Korrekturantraege
        </p>
        <Button size="sm" onClick={() => setShowSubmitDialog(true)}>
          <Send className="mr-1.5 h-4 w-4" />
          Korrektur einreichen
        </Button>
      </div>

      {corrections.length === 0 ? (
        <EmptyState
          icon={Clock}
          title="Keine offenen Korrekturen"
          description="Alle Zeitkorrekturen wurden bearbeitet"
        />
      ) : (
        <div className="space-y-2">
          {corrections.map((entry: WorkTimeEntry) => (
            <div
              key={entry.id}
              className="flex items-center gap-4 rounded-lg border border-border bg-card p-4"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-warning-light shrink-0">
                <AlertCircle className="h-4 w-4 text-warning" />
              </div>

              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground">
                  {formatDate(entry.clockIn)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatTime(entry.clockIn)} - {formatTime(entry.clockOut)}
                  {' · '}{entry.breakMinutes} Min. Pause
                </p>
                {entry.correctionReason && (
                  <p className="text-xs text-muted-foreground mt-0.5 italic">
                    &quot;{entry.correctionReason}&quot;
                  </p>
                )}
              </div>

              <div className="flex items-center gap-1.5 shrink-0">
                <Button
                  size="sm"
                  variant="outline"
                  className="h-8 w-8 p-0 text-success hover:bg-success/10"
                  onClick={() => handleApprove(entry.id)}
                  disabled={approveMutation.isPending}
                >
                  <Check className="h-4 w-4" />
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  className="h-8 w-8 p-0 text-error hover:bg-error/10"
                  onClick={() => toast.info('Ablehnung wird noch nicht unterstuetzt')}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Submit correction dialog */}
      <Dialog open={showSubmitDialog} onOpenChange={setShowSubmitDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Zeitkorrektur einreichen</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label>Original-Eintrag ID</Label>
              <Input
                value={originalEntryId}
                onChange={(e) => setOriginalEntryId(e.target.value)}
                placeholder="ID des zu korrigierenden Eintrags"
                className="font-mono text-xs"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label>Korrigierte Einstempelzeit</Label>
                <Input
                  type="datetime-local"
                  value={correctedClockIn}
                  onChange={(e) => setCorrectedClockIn(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label>Korrigierte Ausstempelzeit</Label>
                <Input
                  type="datetime-local"
                  value={correctedClockOut}
                  onChange={(e) => setCorrectedClockOut(e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label>Pausenminuten</Label>
              <Input
                type="number"
                min={0}
                value={correctedBreakMinutes}
                onChange={(e) => setCorrectedBreakMinutes(Number(e.target.value))}
                className="w-32"
              />
            </div>

            <div className="space-y-1.5">
              <Label>Begruendung</Label>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={2}
                placeholder="Warum muss der Eintrag korrigiert werden?"
                className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowSubmitDialog(false); resetForm() }}>
              Abbrechen
            </Button>
            <Button onClick={handleSubmit} disabled={submitMutation.isPending}>
              {submitMutation.isPending && <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />}
              Einreichen
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
